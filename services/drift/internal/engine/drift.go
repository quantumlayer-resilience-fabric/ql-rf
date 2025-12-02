// Package engine provides the drift calculation engine.
package engine

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/database"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

// Engine calculates drift by comparing fleet assets against golden image baselines.
type Engine struct {
	db     *database.DB
	log    *logger.Logger
	config models.DriftConfig
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
	// TODO: Query database for golden images with production status
	// For now, return mock data
	return map[string]string{
		"ami-0123456789abcdef0": "1.6.4",
		"ami-fedcba9876543210f": "1.6.3",
		"ql-base-linux":        "1.6.4",
	}, nil
}

// getFleetAssets returns all assets matching the given criteria.
func (e *Engine) getFleetAssets(ctx context.Context, req models.DriftCalculationRequest) ([]models.Asset, error) {
	// TODO: Query database for assets
	// For now, return mock data
	return []models.Asset{
		{
			ID:           uuid.New(),
			OrgID:        req.OrgID,
			Platform:     models.PlatformAWS,
			Region:       "us-east-1",
			InstanceID:   "i-001",
			ImageRef:     "ami-0123456789abcdef0",
			ImageVersion: "1.6.4",
			State:        models.AssetStateRunning,
		},
		{
			ID:           uuid.New(),
			OrgID:        req.OrgID,
			Platform:     models.PlatformAWS,
			Region:       "us-east-1",
			InstanceID:   "i-002",
			ImageRef:     "ami-fedcba9876543210f",
			ImageVersion: "1.6.2",
			State:        models.AssetStateRunning,
		},
		{
			ID:           uuid.New(),
			OrgID:        req.OrgID,
			Platform:     models.PlatformAWS,
			Region:       "us-east-1",
			InstanceID:   "i-003",
			ImageRef:     "ami-0123456789abcdef0",
			ImageVersion: "1.6.4",
			State:        models.AssetStateRunning,
		},
	}, nil
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
func (e *Engine) calculateDriftAge(asset models.Asset, expectedVersion string) int {
	// In production, this would compare version timestamps
	// For now, return a mock value based on version difference
	if asset.ImageVersion < expectedVersion {
		// Simple heuristic: minor version = 7 days, patch version = 14 days
		return 14
	}
	return 0
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
	// TODO: Query database and calculate per-scope metrics
	// For now, return mock data
	switch scopeType {
	case "environment":
		return []models.DriftByScope{
			{Scope: "prod", TotalAssets: 8234, CompliantAssets: 7180, CoveragePct: 87.2, Status: models.DriftStatusWarning},
			{Scope: "staging", TotalAssets: 2456, CompliantAssets: 2362, CoveragePct: 96.2, Status: models.DriftStatusHealthy},
			{Scope: "dev", TotalAssets: 1523, CompliantAssets: 1412, CoveragePct: 92.7, Status: models.DriftStatusHealthy},
		}
	case "platform":
		return []models.DriftByScope{
			{Scope: "aws", TotalAssets: 4231, CompliantAssets: 4020, CoveragePct: 95.0, Status: models.DriftStatusHealthy},
			{Scope: "azure", TotalAssets: 3892, CompliantAssets: 3700, CoveragePct: 95.1, Status: models.DriftStatusHealthy},
			{Scope: "gcp", TotalAssets: 2156, CompliantAssets: 2050, CoveragePct: 95.1, Status: models.DriftStatusHealthy},
		}
	case "site":
		return []models.DriftByScope{
			{Scope: "us-east-1", TotalAssets: 3456, CompliantAssets: 3380, CoveragePct: 97.8, Status: models.DriftStatusHealthy},
			{Scope: "eu-west-1", TotalAssets: 2891, CompliantAssets: 2750, CoveragePct: 95.1, Status: models.DriftStatusHealthy},
		}
	default:
		return nil
	}
}
