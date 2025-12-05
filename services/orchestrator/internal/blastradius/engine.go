// Package blastradius provides CVE impact analysis across images, lineage, and assets.
package blastradius

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

// Engine calculates CVE blast radius by traversing package matches, image lineage, and assets.
type Engine struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

// NewEngine creates a new blast radius engine.
func NewEngine(db *pgxpool.Pool, logger *slog.Logger) *Engine {
	if logger == nil {
		logger = slog.Default()
	}
	return &Engine{
		db:     db,
		logger: logger.With("component", "blastradius"),
	}
}

// CalculateInput contains the parameters for blast radius calculation.
type CalculateInput struct {
	OrgID      uuid.UUID
	CVEID      string
	CVECacheID *uuid.UUID
}

// Calculate computes the blast radius for a CVE within an organization.
// Flow: CVE → Package Matches → SBOM Packages → Images → Lineage Propagation → Assets
func (e *Engine) Calculate(ctx context.Context, input CalculateInput) (*models.BlastRadiusResult, error) {
	e.logger.Info("calculating blast radius",
		"org_id", input.OrgID,
		"cve_id", input.CVEID,
	)

	result := &models.BlastRadiusResult{
		CVEID:        input.CVEID,
		CalculatedAt: time.Now().UTC(),
	}

	// Step 1: Find affected packages by matching against CVE package patterns
	packages, err := e.findAffectedPackages(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("find affected packages: %w", err)
	}
	result.AffectedPackages = packages
	result.TotalPackages = len(packages)

	if len(packages) == 0 {
		e.logger.Info("no affected packages found", "cve_id", input.CVEID)
		return result, nil
	}

	// Step 2: Find directly affected images from packages
	imageIDs := e.extractImageIDs(packages)
	directImages, err := e.findDirectlyAffectedImages(ctx, input.OrgID, imageIDs)
	if err != nil {
		return nil, fmt.Errorf("find directly affected images: %w", err)
	}

	// Step 3: Propagate through image lineage (children inherit parent CVEs)
	allImages, err := e.propagateThroughLineage(ctx, input.OrgID, directImages)
	if err != nil {
		return nil, fmt.Errorf("propagate through lineage: %w", err)
	}
	result.AffectedImages = allImages
	result.TotalImages = len(allImages)

	// Step 4: Find affected assets through image coordinates
	assets, err := e.findAffectedAssets(ctx, input.OrgID, allImages)
	if err != nil {
		return nil, fmt.Errorf("find affected assets: %w", err)
	}
	result.AffectedAssets = assets
	result.TotalAssets = len(assets)

	// Step 5: Calculate production assets and platform/region breakdown
	result.ProductionAssets = e.countProductionAssets(assets)
	result.AffectedPlatforms = e.extractUniquePlatforms(assets)
	result.AffectedRegions = e.extractUniqueRegions(assets)

	// Step 6: Calculate urgency score
	cveDetails, err := e.getCVEDetails(ctx, input.CVEID)
	if err != nil {
		e.logger.Warn("failed to get CVE details for scoring", "error", err)
	}

	totalAssetCount, err := e.getTotalAssetCount(ctx, input.OrgID)
	if err != nil {
		e.logger.Warn("failed to get total asset count", "error", err)
	}

	result.UrgencyScore = e.calculateUrgencyScore(cveDetails, result, totalAssetCount)

	e.logger.Info("blast radius calculated",
		"cve_id", input.CVEID,
		"packages", result.TotalPackages,
		"images", result.TotalImages,
		"assets", result.TotalAssets,
		"production_assets", result.ProductionAssets,
		"urgency_score", result.UrgencyScore,
	)

	return result, nil
}

// findAffectedPackages finds SBOM packages that match the CVE's package patterns.
func (e *Engine) findAffectedPackages(ctx context.Context, input CalculateInput) ([]models.AffectedPackage, error) {
	query := `
		WITH cve_patterns AS (
			SELECT
				cpm.package_name,
				cpm.package_type,
				cpm.version_start,
				cpm.version_end,
				cpm.version_constraint,
				cpm.fixed_version,
				cpm.purl_pattern,
				cpm.cpe_pattern
			FROM cve_package_matches cpm
			JOIN cve_cache cc ON cc.id = cpm.cve_cache_id
			WHERE cc.cve_id = $1
		)
		SELECT DISTINCT
			sp.id as package_id,
			sp.sbom_id,
			s.image_id,
			sp.name as package_name,
			sp.version as package_version,
			sp.type as package_type,
			cp.fixed_version
		FROM sbom_packages sp
		JOIN sboms s ON s.id = sp.sbom_id
		JOIN images i ON i.id = s.image_id
		JOIN cve_patterns cp ON (
			LOWER(sp.name) = LOWER(cp.package_name)
			AND (cp.package_type IS NULL OR LOWER(sp.type) = LOWER(cp.package_type))
			AND (
				cp.version_constraint = 'all'
				OR (cp.version_constraint = 'exact' AND sp.version = cp.version_start)
				OR (cp.version_constraint = 'less_than' AND sp.version < cp.version_end)
				OR (cp.version_constraint = 'less_than_eq' AND sp.version <= cp.version_end)
				OR (cp.version_constraint = 'range' AND sp.version >= cp.version_start AND sp.version < cp.version_end)
			)
		)
		WHERE i.org_id = $2
	`

	rows, err := e.db.Query(ctx, query, input.CVEID, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("query affected packages: %w", err)
	}
	defer rows.Close()

	var packages []models.AffectedPackage
	for rows.Next() {
		var pkg models.AffectedPackage
		err := rows.Scan(
			&pkg.PackageID,
			&pkg.SBOMID,
			&pkg.ImageID,
			&pkg.PackageName,
			&pkg.PackageVersion,
			&pkg.PackageType,
			&pkg.FixedVersion,
		)
		if err != nil {
			return nil, fmt.Errorf("scan package row: %w", err)
		}
		packages = append(packages, pkg)
	}

	return packages, rows.Err()
}

// extractImageIDs extracts unique image IDs from affected packages.
func (e *Engine) extractImageIDs(packages []models.AffectedPackage) []uuid.UUID {
	seen := make(map[uuid.UUID]bool)
	var ids []uuid.UUID
	for _, pkg := range packages {
		if !seen[pkg.ImageID] {
			seen[pkg.ImageID] = true
			ids = append(ids, pkg.ImageID)
		}
	}
	return ids
}

// findDirectlyAffectedImages retrieves image details for directly affected images.
func (e *Engine) findDirectlyAffectedImages(ctx context.Context, orgID uuid.UUID, imageIDs []uuid.UUID) ([]models.AffectedImage, error) {
	if len(imageIDs) == 0 {
		return nil, nil
	}

	query := `
		SELECT
			id as image_id,
			family as image_family,
			version as image_version
		FROM images
		WHERE id = ANY($1) AND org_id = $2
	`

	rows, err := e.db.Query(ctx, query, imageIDs, orgID)
	if err != nil {
		return nil, fmt.Errorf("query images: %w", err)
	}
	defer rows.Close()

	var images []models.AffectedImage
	for rows.Next() {
		var img models.AffectedImage
		err := rows.Scan(&img.ImageID, &img.ImageFamily, &img.ImageVersion)
		if err != nil {
			return nil, fmt.Errorf("scan image row: %w", err)
		}
		img.IsDirect = true
		img.LineageDepth = 0
		images = append(images, img)
	}

	return images, rows.Err()
}

// propagateThroughLineage expands affected images to include child images via lineage.
func (e *Engine) propagateThroughLineage(ctx context.Context, orgID uuid.UUID, directImages []models.AffectedImage) ([]models.AffectedImage, error) {
	if len(directImages) == 0 {
		return nil, nil
	}

	// Start with directly affected images
	allImages := make(map[uuid.UUID]models.AffectedImage)
	for _, img := range directImages {
		allImages[img.ImageID] = img
	}

	// BFS to find all child images
	queue := make([]models.AffectedImage, len(directImages))
	copy(queue, directImages)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Find children of this image
		children, err := e.findChildImages(ctx, orgID, current.ImageID)
		if err != nil {
			e.logger.Warn("failed to find child images",
				"parent_id", current.ImageID,
				"error", err,
			)
			continue
		}

		// Update parent's child list
		if existing, ok := allImages[current.ImageID]; ok {
			childIDs := make([]uuid.UUID, len(children))
			for i, child := range children {
				childIDs[i] = child.ImageID
			}
			existing.ChildImageIDs = childIDs
			allImages[current.ImageID] = existing
		}

		for _, child := range children {
			if _, exists := allImages[child.ImageID]; !exists {
				// Mark as inherited from parent
				child.IsDirect = false
				parentID := current.ImageID
				child.InheritedFrom = &parentID
				child.LineageDepth = current.LineageDepth + 1

				allImages[child.ImageID] = child
				queue = append(queue, child)
			}
		}
	}

	// Convert map to slice
	result := make([]models.AffectedImage, 0, len(allImages))
	for _, img := range allImages {
		result = append(result, img)
	}

	return result, nil
}

// findChildImages finds images that derive from the given parent image.
func (e *Engine) findChildImages(ctx context.Context, orgID uuid.UUID, parentID uuid.UUID) ([]models.AffectedImage, error) {
	query := `
		SELECT
			i.id as image_id,
			i.family as image_family,
			i.version as image_version
		FROM images i
		JOIN image_lineage il ON il.image_id = i.id
		WHERE il.parent_image_id = $1 AND i.org_id = $2
	`

	rows, err := e.db.Query(ctx, query, parentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("query child images: %w", err)
	}
	defer rows.Close()

	var images []models.AffectedImage
	for rows.Next() {
		var img models.AffectedImage
		err := rows.Scan(&img.ImageID, &img.ImageFamily, &img.ImageVersion)
		if err != nil {
			return nil, fmt.Errorf("scan child image row: %w", err)
		}
		images = append(images, img)
	}

	return images, rows.Err()
}

// findAffectedAssets finds assets that use any of the affected images.
func (e *Engine) findAffectedAssets(ctx context.Context, orgID uuid.UUID, images []models.AffectedImage) ([]models.AffectedAsset, error) {
	if len(images) == 0 {
		return nil, nil
	}

	imageIDs := make([]uuid.UUID, len(images))
	for i, img := range images {
		imageIDs[i] = img.ImageID
	}

	// Find assets by matching image_ref to image coordinates
	query := `
		SELECT DISTINCT
			a.id as asset_id,
			a.name as asset_name,
			a.platform,
			a.region,
			a.environment,
			CASE WHEN a.environment = 'production' THEN true ELSE false END as is_production,
			a.image_ref,
			ic.image_id
		FROM assets a
		LEFT JOIN image_coordinates ic ON (
			ic.identifier = a.image_ref
			OR ic.identifier LIKE '%' || a.image_ref || '%'
		)
		WHERE a.org_id = $1
		AND (
			ic.image_id = ANY($2)
			OR EXISTS (
				SELECT 1 FROM images i
				WHERE i.id = ANY($2)
				AND (
					a.image_ref LIKE '%' || i.family || '%'
					OR a.image_ref LIKE '%' || i.version || '%'
				)
			)
		)
	`

	rows, err := e.db.Query(ctx, query, orgID, imageIDs)
	if err != nil {
		return nil, fmt.Errorf("query affected assets: %w", err)
	}
	defer rows.Close()

	var assets []models.AffectedAsset
	for rows.Next() {
		var a models.AffectedAsset
		err := rows.Scan(
			&a.AssetID,
			&a.AssetName,
			&a.Platform,
			&a.Region,
			&a.Environment,
			&a.IsProduction,
			&a.ImageRef,
			&a.ImageID,
		)
		if err != nil {
			return nil, fmt.Errorf("scan asset row: %w", err)
		}
		assets = append(assets, a)
	}

	return assets, rows.Err()
}

// countProductionAssets counts assets marked as production.
func (e *Engine) countProductionAssets(assets []models.AffectedAsset) int {
	count := 0
	for _, a := range assets {
		if a.IsProduction {
			count++
		}
	}
	return count
}

// extractUniquePlatforms extracts unique platforms from assets.
func (e *Engine) extractUniquePlatforms(assets []models.AffectedAsset) []string {
	seen := make(map[string]bool)
	var platforms []string
	for _, a := range assets {
		if a.Platform != "" && !seen[a.Platform] {
			seen[a.Platform] = true
			platforms = append(platforms, a.Platform)
		}
	}
	return platforms
}

// extractUniqueRegions extracts unique regions from assets.
func (e *Engine) extractUniqueRegions(assets []models.AffectedAsset) []string {
	seen := make(map[string]bool)
	var regions []string
	for _, a := range assets {
		if a.Region != "" && !seen[a.Region] {
			seen[a.Region] = true
			regions = append(regions, a.Region)
		}
	}
	return regions
}

// getCVEDetails retrieves CVE information for urgency scoring.
func (e *Engine) getCVEDetails(ctx context.Context, cveID string) (*models.CVECache, error) {
	query := `
		SELECT
			id, cve_id, cvss_v3_score, cvss_v3_vector, severity,
			epss_score, epss_percentile, exploit_available, exploit_maturity,
			cisa_kev_listed, cisa_kev_due_date, cisa_kev_ransomware,
			description, published_date, primary_source
		FROM cve_cache
		WHERE cve_id = $1
	`

	var cve models.CVECache
	err := e.db.QueryRow(ctx, query, cveID).Scan(
		&cve.ID, &cve.CVEID, &cve.CVSSV3Score, &cve.CVSSV3Vector, &cve.Severity,
		&cve.EPSSScore, &cve.EPSSPercentile, &cve.ExploitAvailable, &cve.ExploitMaturity,
		&cve.CISAKEVListed, &cve.CISAKEVDueDate, &cve.CISAKEVRansomware,
		&cve.Description, &cve.PublishedDate, &cve.PrimarySource,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query cve_cache: %w", err)
	}

	return &cve, nil
}

// getTotalAssetCount gets the total number of assets for the organization.
func (e *Engine) getTotalAssetCount(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := e.db.QueryRow(ctx, "SELECT COUNT(*) FROM assets WHERE org_id = $1", orgID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// calculateUrgencyScore computes the urgency score (0-100).
func (e *Engine) calculateUrgencyScore(cve *models.CVECache, result *models.BlastRadiusResult, totalAssets int) int {
	input := models.UrgencyScoreInput{
		ProductionAssetCount: result.ProductionAssets,
		TotalAssetCount:      result.TotalAssets,
	}

	if cve != nil {
		if cve.CVSSV3Score != nil {
			input.CVSSScore = *cve.CVSSV3Score
		}
		if cve.EPSSScore != nil {
			input.EPSSScore = *cve.EPSSScore
		}
		input.ExploitAvailable = cve.ExploitAvailable
		input.CISAKEVListed = cve.CISAKEVListed
	}

	// Calculate fleet percentage
	if totalAssets > 0 {
		input.FleetPercentage = float64(result.TotalAssets) / float64(totalAssets) * 100
	}

	return models.CalculateUrgencyScore(input)
}

// StoreBlastRadius persists the blast radius result to the cve_alert_affected_items table.
func (e *Engine) StoreBlastRadius(ctx context.Context, alertID uuid.UUID, result *models.BlastRadiusResult) error {
	tx, err := e.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Clear existing affected items for this alert
	_, err = tx.Exec(ctx, "DELETE FROM cve_alert_affected_items WHERE alert_id = $1", alertID)
	if err != nil {
		return fmt.Errorf("delete existing items: %w", err)
	}

	// Insert packages
	for _, pkg := range result.AffectedPackages {
		_, err = tx.Exec(ctx, `
			INSERT INTO cve_alert_affected_items (
				alert_id, package_id, item_type, package_name, package_version,
				package_type, fixed_version, lineage_depth, item_status
			) VALUES ($1, $2, 'package', $3, $4, $5, $6, 0, 'vulnerable')
		`, alertID, pkg.PackageID, pkg.PackageName, pkg.PackageVersion,
			pkg.PackageType, pkg.FixedVersion)
		if err != nil {
			return fmt.Errorf("insert package item: %w", err)
		}
	}

	// Insert images
	for _, img := range result.AffectedImages {
		var inheritedFrom *uuid.UUID
		if !img.IsDirect {
			inheritedFrom = img.InheritedFrom
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO cve_alert_affected_items (
				alert_id, image_id, item_type, image_family, image_version,
				inherited_from_image_id, lineage_depth, item_status
			) VALUES ($1, $2, 'image', $3, $4, $5, $6, 'vulnerable')
		`, alertID, img.ImageID, img.ImageFamily, img.ImageVersion,
			inheritedFrom, img.LineageDepth)
		if err != nil {
			return fmt.Errorf("insert image item: %w", err)
		}
	}

	// Insert assets
	for _, asset := range result.AffectedAssets {
		_, err = tx.Exec(ctx, `
			INSERT INTO cve_alert_affected_items (
				alert_id, asset_id, item_type, asset_name, asset_platform,
				asset_environment, asset_region, is_production, lineage_depth, item_status
			) VALUES ($1, $2, 'asset', $3, $4, $5, $6, $7, 0, 'vulnerable')
		`, alertID, asset.AssetID, asset.AssetName, asset.Platform,
			asset.Environment, asset.Region, asset.IsProduction)
		if err != nil {
			return fmt.Errorf("insert asset item: %w", err)
		}
	}

	// Update alert counts
	_, err = tx.Exec(ctx, `
		UPDATE cve_alerts SET
			affected_packages_count = $2,
			affected_images_count = $3,
			affected_assets_count = $4,
			production_assets_count = $5,
			urgency_score = $6,
			updated_at = NOW()
		WHERE id = $1
	`, alertID, result.TotalPackages, result.TotalImages,
		result.TotalAssets, result.ProductionAssets, result.UrgencyScore)
	if err != nil {
		return fmt.Errorf("update alert counts: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	e.logger.Info("blast radius stored",
		"alert_id", alertID,
		"packages", result.TotalPackages,
		"images", result.TotalImages,
		"assets", result.TotalAssets,
	)

	return nil
}
