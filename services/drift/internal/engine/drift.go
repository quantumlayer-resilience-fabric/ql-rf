// Package engine provides the drift calculation engine.
package engine

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/database"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

// ImageBaseline represents a golden image version with its release timestamp.
type ImageBaseline struct {
	Version   string
	CreatedAt time.Time
}

// Engine calculates drift by comparing fleet assets against golden image baselines.
type Engine struct {
	db     *database.DB
	log    *logger.Logger
	config models.DriftConfig

	// imageBaselines is cached during calculation for drift age computation
	imageBaselines map[string]ImageBaseline
}

// New creates a new drift engine.
func New(db *database.DB, log *logger.Logger, config models.DriftConfig) *Engine {
	return &Engine{
		db:     db,
		log:    log.WithComponent("drift-engine"),
		config: config,
	}
}

// Calculate calculates drift for the given scope.
func (e *Engine) Calculate(ctx context.Context, req models.DriftCalculationRequest) (*models.DriftReport, []models.OutdatedAsset, error) {
	e.log.Info("calculating drift",
		"org_id", req.OrgID,
		"env_id", req.EnvID,
		"platform", req.Platform,
		"site", req.Site,
	)

	startTime := time.Now()

	// Get golden image baselines
	baselines, err := e.getGoldenImageBaselines(ctx, req.OrgID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get baselines: %w", err)
	}

	// Get fleet assets
	assets, err := e.getFleetAssets(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get assets: %w", err)
	}

	// Calculate compliance
	compliantCount := 0
	var outdatedAssets []models.OutdatedAsset

	for _, asset := range assets {
		// Skip non-running assets
		if !asset.State.IsActive() {
			continue
		}

		// Check if asset is using the latest image
		expectedVersion, ok := baselines[asset.ImageRef]
		if !ok {
			// No baseline for this image ref, try to match by family
			expectedVersion = e.findBaselineByFamily(baselines, asset.ImageRef)
		}

		if expectedVersion == "" {
			// No baseline found, consider compliant (might be a different image family)
			compliantCount++
			continue
		}

		if asset.ImageVersion == expectedVersion {
			compliantCount++
		} else {
			// Calculate drift age
			driftAge := e.calculateDriftAge(asset, expectedVersion)
			severity := e.calculateSeverity(driftAge)

			outdatedAssets = append(outdatedAssets, models.OutdatedAsset{
				Asset:           asset,
				CurrentVersion:  asset.ImageVersion,
				ExpectedVersion: expectedVersion,
				DriftAge:        driftAge,
				Severity:        severity,
			})
		}
	}

	// Sort outdated assets by drift age (most outdated first)
	sort.Slice(outdatedAssets, func(i, j int) bool {
		return outdatedAssets[i].DriftAge > outdatedAssets[j].DriftAge
	})

	// Limit top offenders
	topOffenders := outdatedAssets
	if len(topOffenders) > e.config.MaxOffenders {
		topOffenders = topOffenders[:e.config.MaxOffenders]
	}

	// Calculate coverage percentage
	totalAssets := len(assets)
	coveragePct := 0.0
	if totalAssets > 0 {
		coveragePct = float64(compliantCount) / float64(totalAssets) * 100
	}

	// Determine status
	status := models.CalculateStatus(coveragePct, e.config.WarningThreshold, e.config.CriticalThreshold)

	report := &models.DriftReport{
		ID:              uuid.New(),
		OrgID:           req.OrgID,
		EnvID:           req.EnvID,
		Platform:        req.Platform,
		Site:            req.Site,
		TotalAssets:     totalAssets,
		CompliantAssets: compliantCount,
		CoveragePct:     coveragePct,
		Status:          status,
		CalculatedAt:    time.Now(),
	}

	duration := time.Since(startTime)
	e.log.Info("drift calculation completed",
		"total_assets", totalAssets,
		"compliant_assets", compliantCount,
		"coverage_pct", fmt.Sprintf("%.2f", coveragePct),
		"status", status,
		"outdated_count", len(outdatedAssets),
		"duration", duration.String(),
	)

	return report, topOffenders, nil
}

// getGoldenImageBaselines returns a map of image references to their latest versions.
func (e *Engine) getGoldenImageBaselines(ctx context.Context, orgID uuid.UUID) (map[string]string, error) {
	baselines := make(map[string]string)
	e.imageBaselines = make(map[string]ImageBaseline)

	// Query for the latest production images for each family, including creation timestamp
	query := `
		SELECT DISTINCT ON (i.family)
			i.family,
			i.version,
			ic.identifier,
			i.created_at
		FROM images i
		LEFT JOIN image_coordinates ic ON ic.image_id = i.id
		WHERE i.org_id = $1
		AND i.status = 'production'
		ORDER BY i.family, i.created_at DESC
	`

	rows, err := e.db.Pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query golden images: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var family, version string
		var identifier *string
		var createdAt time.Time
		if err := rows.Scan(&family, &version, &identifier, &createdAt); err != nil {
			e.log.Warn("failed to scan golden image row", "error", err)
			continue
		}

		baseline := ImageBaseline{
			Version:   version,
			CreatedAt: createdAt,
		}

		// Map family name to version and baseline
		baselines[family] = version
		e.imageBaselines[family] = baseline

		// Also map platform-specific identifiers to version and baseline
		if identifier != nil && *identifier != "" {
			baselines[*identifier] = version
			e.imageBaselines[*identifier] = baseline
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating golden images: %w", err)
	}

	e.log.Debug("loaded golden image baselines",
		"org_id", orgID,
		"count", len(baselines),
	)

	return baselines, nil
}

// getFleetAssets returns all assets matching the given criteria.
func (e *Engine) getFleetAssets(ctx context.Context, req models.DriftCalculationRequest) ([]models.Asset, error) {
	// Build dynamic query based on filters
	query := `
		SELECT
			a.id,
			a.org_id,
			a.env_id,
			a.platform,
			a.account,
			a.region,
			a.site,
			a.instance_id,
			a.name,
			a.image_ref,
			a.image_version,
			a.state,
			a.tags,
			a.discovered_at,
			a.updated_at
		FROM assets a
		WHERE a.org_id = $1
	`
	args := []interface{}{req.OrgID}
	argNum := 2

	// Apply optional filters
	if req.EnvID != uuid.Nil {
		query += fmt.Sprintf(" AND a.env_id = $%d", argNum)
		args = append(args, req.EnvID)
		argNum++
	}

	if req.Platform != "" {
		query += fmt.Sprintf(" AND a.platform = $%d", argNum)
		args = append(args, req.Platform)
		argNum++
	}

	if req.Site != "" {
		query += fmt.Sprintf(" AND a.site = $%d", argNum)
		args = append(args, req.Site)
		argNum++
	}

	// Only get active assets by default
	query += " ORDER BY a.discovered_at DESC"

	rows, err := e.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query assets: %w", err)
	}
	defer rows.Close()

	var assets []models.Asset
	for rows.Next() {
		var asset models.Asset
		var envID *uuid.UUID
		var account, region, site, name, imageRef, imageVersion *string
		var tags []byte

		if err := rows.Scan(
			&asset.ID,
			&asset.OrgID,
			&envID,
			&asset.Platform,
			&account,
			&region,
			&site,
			&asset.InstanceID,
			&name,
			&imageRef,
			&imageVersion,
			&asset.State,
			&tags,
			&asset.DiscoveredAt,
			&asset.UpdatedAt,
		); err != nil {
			e.log.Warn("failed to scan asset row", "error", err)
			continue
		}

		// Set optional fields
		if envID != nil {
			asset.EnvID = *envID
		}
		if account != nil {
			asset.Account = *account
		}
		if region != nil {
			asset.Region = *region
		}
		if site != nil {
			asset.Site = *site
		}
		if name != nil {
			asset.Name = *name
		}
		if imageRef != nil {
			asset.ImageRef = *imageRef
		}
		if imageVersion != nil {
			asset.ImageVersion = *imageVersion
		}

		// Set tags JSON directly (it's already json.RawMessage type)
		if len(tags) > 0 {
			asset.Tags = tags
		}

		assets = append(assets, asset)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating assets: %w", err)
	}

	e.log.Debug("loaded fleet assets",
		"org_id", req.OrgID,
		"count", len(assets),
	)

	return assets, nil
}

// findBaselineByFamily attempts to match an image ref to a family.
func (e *Engine) findBaselineByFamily(baselines map[string]string, imageRef string) string {
	// This is a simplified implementation
	// In production, this would parse the image ref and match against image families
	for family, version := range baselines {
		if family == "ql-base-linux" {
			return version
		}
	}
	return ""
}

// calculateDriftAge calculates how many days the asset has been outdated.
// It uses the image baseline timestamp to determine when the new golden image was released,
// then calculates how long the asset has been behind since that release.
func (e *Engine) calculateDriftAge(asset models.Asset, expectedVersion string) int {
	// If asset version matches expected, no drift
	if asset.ImageVersion == expectedVersion {
		return 0
	}

	now := time.Now()

	// Strategy 1: Use the golden image's release timestamp
	// The drift age is how long since the new golden image was released
	// and the asset is still on an older version
	if e.imageBaselines != nil {
		// Try to find the baseline by image ref first
		if baseline, ok := e.imageBaselines[asset.ImageRef]; ok {
			driftDays := int(now.Sub(baseline.CreatedAt).Hours() / 24)
			if driftDays > 0 {
				return driftDays
			}
		}

		// Try to find baseline by family name pattern
		for key, baseline := range e.imageBaselines {
			if strings.Contains(asset.ImageRef, key) || strings.Contains(key, "ql-base") {
				driftDays := int(now.Sub(baseline.CreatedAt).Hours() / 24)
				if driftDays > 0 {
					return driftDays
				}
			}
		}
	}

	// Strategy 2: Use the asset's last update time
	// If the asset hasn't been updated recently, calculate drift from its last update
	if !asset.UpdatedAt.IsZero() {
		driftDays := int(now.Sub(asset.UpdatedAt).Hours() / 24)
		if driftDays > 0 {
			return driftDays
		}
	}

	// Strategy 3: Use the asset's discovery time as fallback
	// This represents how long the asset has potentially been outdated
	if !asset.DiscoveredAt.IsZero() {
		driftDays := int(now.Sub(asset.DiscoveredAt).Hours() / 24)
		if driftDays > 0 {
			return driftDays
		}
	}

	// Strategy 4: Version comparison fallback
	// If no timestamps available, use semantic version comparison as heuristic
	return e.estimateDriftFromVersions(asset.ImageVersion, expectedVersion)
}

// estimateDriftFromVersions estimates drift age based on version number differences.
// This is a fallback when timestamps are not available.
func (e *Engine) estimateDriftFromVersions(currentVersion, expectedVersion string) int {
	// Parse version numbers (simplified - assumes semver-like format X.Y.Z)
	currentParts := strings.Split(currentVersion, ".")
	expectedParts := strings.Split(expectedVersion, ".")

	// Calculate version distance
	majorDiff := 0
	minorDiff := 0
	patchDiff := 0

	if len(currentParts) >= 1 && len(expectedParts) >= 1 {
		var curr, exp int
		fmt.Sscanf(currentParts[0], "%d", &curr)
		fmt.Sscanf(expectedParts[0], "%d", &exp)
		majorDiff = exp - curr
	}

	if len(currentParts) >= 2 && len(expectedParts) >= 2 {
		var curr, exp int
		fmt.Sscanf(currentParts[1], "%d", &curr)
		fmt.Sscanf(expectedParts[1], "%d", &exp)
		minorDiff = exp - curr
	}

	if len(currentParts) >= 3 && len(expectedParts) >= 3 {
		var curr, exp int
		fmt.Sscanf(currentParts[2], "%d", &curr)
		fmt.Sscanf(expectedParts[2], "%d", &exp)
		patchDiff = exp - curr
	}

	// Estimate days based on typical release cadence:
	// Major version = ~90 days (quarterly releases)
	// Minor version = ~30 days (monthly releases)
	// Patch version = ~7 days (weekly patches)
	estimatedDays := (majorDiff * 90) + (minorDiff * 30) + (patchDiff * 7)

	if estimatedDays < 0 {
		return 0
	}

	return estimatedDays
}

// calculateSeverity determines the severity based on drift age.
func (e *Engine) calculateSeverity(driftAgeDays int) models.DriftStatus {
	if driftAgeDays > 30 {
		return models.DriftStatusCritical
	}
	if driftAgeDays > 14 {
		return models.DriftStatusWarning
	}
	return models.DriftStatusHealthy
}

// CalculateSummary calculates a complete drift summary for the organization.
func (e *Engine) CalculateSummary(ctx context.Context, orgID uuid.UUID) (*models.DriftSummary, error) {
	// Calculate overall drift
	report, topOffenders, err := e.Calculate(ctx, models.DriftCalculationRequest{
		OrgID: orgID,
	})
	if err != nil {
		return nil, err
	}

	// Calculate by environment
	byEnv := e.calculateByScope(ctx, orgID, "environment")

	// Calculate by platform
	byPlatform := e.calculateByScope(ctx, orgID, "platform")

	// Calculate by site
	bySite := e.calculateByScope(ctx, orgID, "site")

	return &models.DriftSummary{
		OrgID:           orgID,
		TotalAssets:     report.TotalAssets,
		CompliantAssets: report.CompliantAssets,
		CoveragePct:     report.CoveragePct,
		Status:          report.Status,
		ByEnvironment:   byEnv,
		ByPlatform:      byPlatform,
		BySite:          bySite,
		TopOffenders:    topOffenders,
		CalculatedAt:    time.Now(),
	}, nil
}

// calculateByScope calculates drift metrics grouped by a scope.
func (e *Engine) calculateByScope(ctx context.Context, orgID uuid.UUID, scopeType string) []models.DriftByScope {
	// Get baselines first
	baselines, err := e.getGoldenImageBaselines(ctx, orgID)
	if err != nil {
		e.log.Warn("failed to get baselines for scope calculation", "error", err)
		return nil
	}

	// Determine the grouping column
	var groupColumn string
	switch scopeType {
	case "environment":
		groupColumn = "e.name"
	case "platform":
		groupColumn = "a.platform"
	case "site":
		groupColumn = "a.site"
	default:
		return nil
	}

	// Build query with grouping
	var query string
	if scopeType == "environment" {
		query = fmt.Sprintf(`
			SELECT
				COALESCE(%s, 'unknown') as scope,
				COUNT(*) as total_assets,
				a.image_ref,
				a.image_version
			FROM assets a
			LEFT JOIN environments e ON e.id = a.env_id
			WHERE a.org_id = $1
			AND a.state IN ('running', 'stopped')
			GROUP BY %s, a.image_ref, a.image_version
			ORDER BY scope
		`, groupColumn, groupColumn)
	} else {
		query = fmt.Sprintf(`
			SELECT
				COALESCE(%s, 'unknown') as scope,
				COUNT(*) as total_assets,
				a.image_ref,
				a.image_version
			FROM assets a
			WHERE a.org_id = $1
			AND a.state IN ('running', 'stopped')
			GROUP BY %s, a.image_ref, a.image_version
			ORDER BY scope
		`, groupColumn, groupColumn)
	}

	rows, err := e.db.Pool.Query(ctx, query, orgID)
	if err != nil {
		e.log.Warn("failed to query scope metrics", "error", err, "scope_type", scopeType)
		return nil
	}
	defer rows.Close()

	// Aggregate results by scope
	scopeMetrics := make(map[string]*models.DriftByScope)

	for rows.Next() {
		var scope string
		var count int
		var imageRef, imageVersion *string

		if err := rows.Scan(&scope, &count, &imageRef, &imageVersion); err != nil {
			e.log.Warn("failed to scan scope row", "error", err)
			continue
		}

		// Initialize scope if needed
		if _, exists := scopeMetrics[scope]; !exists {
			scopeMetrics[scope] = &models.DriftByScope{
				Scope:           scope,
				TotalAssets:     0,
				CompliantAssets: 0,
			}
		}

		metric := scopeMetrics[scope]
		metric.TotalAssets += count

		// Check compliance
		isCompliant := false
		if imageRef != nil && imageVersion != nil {
			expectedVersion, ok := baselines[*imageRef]
			if !ok {
				// Try family-based match
				for family, version := range baselines {
					if family == *imageRef || strings.Contains(*imageRef, family) {
						expectedVersion = version
						ok = true
						break
					}
				}
			}
			if !ok {
				// No baseline found - consider compliant
				isCompliant = true
			} else {
				isCompliant = *imageVersion == expectedVersion
			}
		}

		if isCompliant {
			metric.CompliantAssets += count
		}
	}

	// Convert to slice and calculate percentages
	var results []models.DriftByScope
	for _, metric := range scopeMetrics {
		if metric.TotalAssets > 0 {
			metric.CoveragePct = float64(metric.CompliantAssets) / float64(metric.TotalAssets) * 100
			metric.Status = models.CalculateStatus(metric.CoveragePct, e.config.WarningThreshold, e.config.CriticalThreshold)
		}
		results = append(results, *metric)
	}

	// Sort by scope name
	sort.Slice(results, func(i, j int) bool {
		return results[i].Scope < results[j].Scope
	})

	return results
}
