// Package repository provides database access for the API service.
package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository provides database operations.
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a new repository.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// =============================================================================
// Organization Models and Methods
// =============================================================================

// Organization represents an organization (tenant).
type Organization struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetOrganization retrieves an organization by ID.
func (r *Repository) GetOrganization(ctx context.Context, id uuid.UUID) (*Organization, error) {
	var org Organization
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, created_at, updated_at
		FROM organizations WHERE id = $1
	`, id).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// GetOrganizationBySlug retrieves an organization by slug.
func (r *Repository) GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	var org Organization
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, created_at, updated_at
		FROM organizations WHERE slug = $1
	`, slug).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// =============================================================================
// Image Models and Methods
// =============================================================================

// Image represents a golden image.
type Image struct {
	ID        uuid.UUID  `json:"id"`
	OrgID     uuid.UUID  `json:"org_id"`
	Family    string     `json:"family"`
	Version   string     `json:"version"`
	OSName    *string    `json:"os_name,omitempty"`
	OSVersion *string    `json:"os_version,omitempty"`
	CISLevel  *int       `json:"cis_level,omitempty"`
	SBOMUrl   *string    `json:"sbom_url,omitempty"`
	Signed    bool       `json:"signed"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ImageCoordinate represents platform-specific image coordinates.
type ImageCoordinate struct {
	ID         uuid.UUID `json:"id"`
	ImageID    uuid.UUID `json:"image_id"`
	Platform   string    `json:"platform"`
	Region     *string   `json:"region,omitempty"`
	Identifier string    `json:"identifier"`
	CreatedAt  time.Time `json:"created_at"`
}

// ListImagesParams contains parameters for listing images.
type ListImagesParams struct {
	OrgID  uuid.UUID
	Family *string
	Status *string
	Limit  int32
	Offset int32
}

// GetImage retrieves an image by ID.
func (r *Repository) GetImage(ctx context.Context, id uuid.UUID) (*Image, error) {
	var img Image
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, family, version, os_name, os_version,
		       cis_level, sbom_url, signed, status, created_at, updated_at
		FROM images WHERE id = $1
	`, id).Scan(
		&img.ID, &img.OrgID, &img.Family, &img.Version,
		&img.OSName, &img.OSVersion, &img.CISLevel, &img.SBOMUrl,
		&img.Signed, &img.Status, &img.CreatedAt, &img.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &img, nil
}

// GetLatestImageByFamily retrieves the latest production image for a family.
func (r *Repository) GetLatestImageByFamily(ctx context.Context, orgID uuid.UUID, family string) (*Image, error) {
	var img Image
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, family, version, os_name, os_version,
		       cis_level, sbom_url, signed, status, created_at, updated_at
		FROM images
		WHERE org_id = $1 AND family = $2 AND status = 'production'
		ORDER BY created_at DESC
		LIMIT 1
	`, orgID, family).Scan(
		&img.ID, &img.OrgID, &img.Family, &img.Version,
		&img.OSName, &img.OSVersion, &img.CISLevel, &img.SBOMUrl,
		&img.Signed, &img.Status, &img.CreatedAt, &img.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &img, nil
}

// ListImages retrieves images for an organization.
func (r *Repository) ListImages(ctx context.Context, params ListImagesParams) ([]Image, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, family, version, os_name, os_version,
		       cis_level, sbom_url, signed, status, created_at, updated_at
		FROM images
		WHERE org_id = $1
		ORDER BY family, created_at DESC
		LIMIT $2 OFFSET $3
	`, params.OrgID, params.Limit, params.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []Image
	for rows.Next() {
		var img Image
		if err := rows.Scan(
			&img.ID, &img.OrgID, &img.Family, &img.Version,
			&img.OSName, &img.OSVersion, &img.CISLevel, &img.SBOMUrl,
			&img.Signed, &img.Status, &img.CreatedAt, &img.UpdatedAt,
		); err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, rows.Err()
}

// CreateImageParams contains parameters for creating an image.
type CreateImageParams struct {
	OrgID     uuid.UUID
	Family    string
	Version   string
	OSName    *string
	OSVersion *string
	CISLevel  *int
	SBOMUrl   *string
	Signed    bool
	Status    string
}

// CreateImage creates a new image.
func (r *Repository) CreateImage(ctx context.Context, params CreateImageParams) (*Image, error) {
	var img Image
	err := r.pool.QueryRow(ctx, `
		INSERT INTO images (org_id, family, version, os_name, os_version, cis_level, sbom_url, signed, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, org_id, family, version, os_name, os_version, cis_level, sbom_url, signed, status, created_at, updated_at
	`, params.OrgID, params.Family, params.Version, params.OSName, params.OSVersion,
		params.CISLevel, params.SBOMUrl, params.Signed, params.Status,
	).Scan(
		&img.ID, &img.OrgID, &img.Family, &img.Version,
		&img.OSName, &img.OSVersion, &img.CISLevel, &img.SBOMUrl,
		&img.Signed, &img.Status, &img.CreatedAt, &img.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &img, nil
}

// GetImageCoordinates retrieves coordinates for an image.
func (r *Repository) GetImageCoordinates(ctx context.Context, imageID uuid.UUID) ([]ImageCoordinate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, image_id, platform, region, identifier, created_at
		FROM image_coordinates WHERE image_id = $1
	`, imageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var coords []ImageCoordinate
	for rows.Next() {
		var c ImageCoordinate
		if err := rows.Scan(&c.ID, &c.ImageID, &c.Platform, &c.Region, &c.Identifier, &c.CreatedAt); err != nil {
			return nil, err
		}
		coords = append(coords, c)
	}
	return coords, rows.Err()
}

// CountImagesByOrg counts images for an organization.
func (r *Repository) CountImagesByOrg(ctx context.Context, orgID uuid.UUID) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM images WHERE org_id = $1`, orgID).Scan(&count)
	return count, err
}

// UpdateImageStatus updates an image's status.
func (r *Repository) UpdateImageStatus(ctx context.Context, id uuid.UUID, status string) (*Image, error) {
	var img Image
	err := r.pool.QueryRow(ctx, `
		UPDATE images SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, org_id, family, version, os_name, os_version,
		          cis_level, sbom_url, signed, status, created_at, updated_at
	`, id, status).Scan(
		&img.ID, &img.OrgID, &img.Family, &img.Version,
		&img.OSName, &img.OSVersion, &img.CISLevel, &img.SBOMUrl,
		&img.Signed, &img.Status, &img.CreatedAt, &img.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &img, nil
}

// CreateImageCoordinateParams contains parameters for creating an image coordinate.
type CreateImageCoordinateParams struct {
	ImageID    uuid.UUID
	Platform   string
	Region     *string
	Identifier string
}

// CreateImageCoordinate creates a new image coordinate.
func (r *Repository) CreateImageCoordinate(ctx context.Context, params CreateImageCoordinateParams) (*ImageCoordinate, error) {
	var coord ImageCoordinate
	err := r.pool.QueryRow(ctx, `
		INSERT INTO image_coordinates (image_id, platform, region, identifier)
		VALUES ($1, $2, $3, $4)
		RETURNING id, image_id, platform, region, identifier, created_at
	`, params.ImageID, params.Platform, params.Region, params.Identifier).Scan(
		&coord.ID, &coord.ImageID, &coord.Platform, &coord.Region, &coord.Identifier, &coord.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &coord, nil
}

// =============================================================================
// Asset Models and Methods
// =============================================================================

// Asset represents a discovered fleet asset.
type Asset struct {
	ID           uuid.UUID       `json:"id"`
	OrgID        uuid.UUID       `json:"org_id"`
	EnvID        *uuid.UUID      `json:"env_id,omitempty"`
	Platform     string          `json:"platform"`
	Account      *string         `json:"account,omitempty"`
	Region       *string         `json:"region,omitempty"`
	Site         *string         `json:"site,omitempty"`
	InstanceID   string          `json:"instance_id"`
	Name         *string         `json:"name,omitempty"`
	ImageRef     *string         `json:"image_ref,omitempty"`
	ImageVersion *string         `json:"image_version,omitempty"`
	State        string          `json:"state"`
	Tags         json.RawMessage `json:"tags,omitempty"`
	DiscoveredAt time.Time       `json:"discovered_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// ListAssetsParams contains parameters for listing assets.
type ListAssetsParams struct {
	OrgID    uuid.UUID
	EnvID    *uuid.UUID
	Platform *string
	State    *string
	Limit    int32
	Offset   int32
}

// GetAsset retrieves an asset by ID.
func (r *Repository) GetAsset(ctx context.Context, id uuid.UUID) (*Asset, error) {
	var a Asset
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, env_id, platform, account, region, site,
		       instance_id, name, image_ref, image_version, state, tags,
		       discovered_at, updated_at
		FROM assets WHERE id = $1
	`, id).Scan(
		&a.ID, &a.OrgID, &a.EnvID, &a.Platform, &a.Account, &a.Region, &a.Site,
		&a.InstanceID, &a.Name, &a.ImageRef, &a.ImageVersion, &a.State, &a.Tags,
		&a.DiscoveredAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListAssets retrieves assets for an organization with optional filters.
func (r *Repository) ListAssets(ctx context.Context, params ListAssetsParams) ([]Asset, error) {
	query := `
		SELECT id, org_id, env_id, platform, account, region, site,
		       instance_id, name, image_ref, image_version, state, tags,
		       discovered_at, updated_at
		FROM assets
		WHERE org_id = $1
	`
	args := []interface{}{params.OrgID}
	argIdx := 2

	if params.EnvID != nil {
		query += ` AND env_id = $` + string(rune('0'+argIdx))
		args = append(args, *params.EnvID)
		argIdx++
	}
	if params.Platform != nil {
		query += ` AND platform = $` + string(rune('0'+argIdx))
		args = append(args, *params.Platform)
		argIdx++
	}
	if params.State != nil {
		query += ` AND state = $` + string(rune('0'+argIdx))
		args = append(args, *params.State)
		argIdx++
	}

	query += ` ORDER BY discovered_at DESC LIMIT $` + string(rune('0'+argIdx)) + ` OFFSET $` + string(rune('0'+argIdx+1))
	args = append(args, params.Limit, params.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var a Asset
		if err := rows.Scan(
			&a.ID, &a.OrgID, &a.EnvID, &a.Platform, &a.Account, &a.Region, &a.Site,
			&a.InstanceID, &a.Name, &a.ImageRef, &a.ImageVersion, &a.State, &a.Tags,
			&a.DiscoveredAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}
	return assets, rows.Err()
}

// UpsertAssetParams contains parameters for upserting an asset.
type UpsertAssetParams struct {
	OrgID        uuid.UUID
	EnvID        *uuid.UUID
	Platform     string
	Account      *string
	Region       *string
	InstanceID   string
	ImageRef     *string
	ImageVersion *string
	State        string
	Tags         json.RawMessage
}

// UpsertAsset creates or updates an asset.
func (r *Repository) UpsertAsset(ctx context.Context, params UpsertAssetParams) (*Asset, error) {
	var a Asset
	err := r.pool.QueryRow(ctx, `
		INSERT INTO assets (org_id, env_id, platform, account, region, instance_id, image_ref, image_version, state, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (org_id, platform, instance_id)
		DO UPDATE SET
			env_id = EXCLUDED.env_id,
			account = EXCLUDED.account,
			region = EXCLUDED.region,
			image_ref = EXCLUDED.image_ref,
			image_version = EXCLUDED.image_version,
			state = EXCLUDED.state,
			tags = EXCLUDED.tags,
			updated_at = NOW()
		RETURNING id, org_id, env_id, platform, account, region, site,
		          instance_id, name, image_ref, image_version, state, tags,
		          discovered_at, updated_at
	`, params.OrgID, params.EnvID, params.Platform, params.Account, params.Region,
		params.InstanceID, params.ImageRef, params.ImageVersion, params.State, params.Tags,
	).Scan(
		&a.ID, &a.OrgID, &a.EnvID, &a.Platform, &a.Account, &a.Region, &a.Site,
		&a.InstanceID, &a.Name, &a.ImageRef, &a.ImageVersion, &a.State, &a.Tags,
		&a.DiscoveredAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// CountAssetsByOrg counts assets for an organization.
func (r *Repository) CountAssetsByOrg(ctx context.Context, orgID uuid.UUID) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM assets WHERE org_id = $1`, orgID).Scan(&count)
	return count, err
}

// GetAssetByInstanceID retrieves an asset by instance ID.
func (r *Repository) GetAssetByInstanceID(ctx context.Context, orgID uuid.UUID, platform, instanceID string) (*Asset, error) {
	var a Asset
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, env_id, platform, account, region, site,
		       instance_id, name, image_ref, image_version, state, tags,
		       discovered_at, updated_at
		FROM assets WHERE org_id = $1 AND platform = $2 AND instance_id = $3
	`, orgID, platform, instanceID).Scan(
		&a.ID, &a.OrgID, &a.EnvID, &a.Platform, &a.Account, &a.Region, &a.Site,
		&a.InstanceID, &a.Name, &a.ImageRef, &a.ImageVersion, &a.State, &a.Tags,
		&a.DiscoveredAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// DeleteAsset deletes an asset by ID.
func (r *Repository) DeleteAsset(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM assets WHERE id = $1`, id)
	return err
}

// CountAssetsByState counts assets by state.
func (r *Repository) CountAssetsByState(ctx context.Context, orgID uuid.UUID, state string) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM assets WHERE org_id = $1 AND state = $2
	`, orgID, state).Scan(&count)
	return count, err
}

// CountCompliantAssets counts assets running the latest golden image.
func (r *Repository) CountCompliantAssets(ctx context.Context, orgID uuid.UUID) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM assets a
		JOIN images i ON a.org_id = i.org_id AND a.image_ref = i.family AND a.image_version = i.version
		WHERE a.org_id = $1 AND a.state = 'running' AND i.status = 'production'
	`, orgID).Scan(&count)
	return count, err
}

// =============================================================================
// Drift Models and Methods
// =============================================================================

// DriftReport represents a point-in-time drift snapshot.
type DriftReport struct {
	ID              uuid.UUID  `json:"id"`
	OrgID           uuid.UUID  `json:"org_id"`
	EnvID           *uuid.UUID `json:"env_id,omitempty"`
	Platform        *string    `json:"platform,omitempty"`
	Site            *string    `json:"site,omitempty"`
	TotalAssets     int        `json:"total_assets"`
	CompliantAssets int        `json:"compliant_assets"`
	CoveragePct     float64    `json:"coverage_pct"`
	Status          string     `json:"status"`
	CalculatedAt    time.Time  `json:"calculated_at"`
}

// DriftByScope represents drift aggregated by a scope (env, platform, site).
type DriftByScope struct {
	Scope           string  `json:"scope"`
	TotalAssets     int64   `json:"total_assets"`
	CompliantAssets int64   `json:"compliant_assets"`
	CoveragePct     float64 `json:"coverage_pct"`
	Status          string  `json:"status"`
}

// GetLatestDriftReport retrieves the latest drift report for an organization.
func (r *Repository) GetLatestDriftReport(ctx context.Context, orgID uuid.UUID) (*DriftReport, error) {
	var dr DriftReport
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, env_id, platform, site, total_assets, compliant_assets,
		       coverage_pct, status, calculated_at
		FROM drift_reports
		WHERE org_id = $1
		ORDER BY calculated_at DESC
		LIMIT 1
	`, orgID).Scan(
		&dr.ID, &dr.OrgID, &dr.EnvID, &dr.Platform, &dr.Site,
		&dr.TotalAssets, &dr.CompliantAssets, &dr.CoveragePct, &dr.Status, &dr.CalculatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &dr, nil
}

// ListDriftReports retrieves drift reports for an organization.
func (r *Repository) ListDriftReports(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]DriftReport, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, env_id, platform, site, total_assets, compliant_assets,
		       coverage_pct, status, calculated_at
		FROM drift_reports
		WHERE org_id = $1
		ORDER BY calculated_at DESC
		LIMIT $2 OFFSET $3
	`, orgID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []DriftReport
	for rows.Next() {
		var dr DriftReport
		if err := rows.Scan(
			&dr.ID, &dr.OrgID, &dr.EnvID, &dr.Platform, &dr.Site,
			&dr.TotalAssets, &dr.CompliantAssets, &dr.CoveragePct, &dr.Status, &dr.CalculatedAt,
		); err != nil {
			return nil, err
		}
		reports = append(reports, dr)
	}
	return reports, rows.Err()
}

// CreateDriftReportParams contains parameters for creating a drift report.
type CreateDriftReportParams struct {
	OrgID           uuid.UUID
	EnvID           *uuid.UUID
	Platform        *string
	Site            *string
	TotalAssets     int
	CompliantAssets int
	CoveragePct     float64
}

// CreateDriftReport creates a new drift report.
func (r *Repository) CreateDriftReport(ctx context.Context, params CreateDriftReportParams) (*DriftReport, error) {
	status := calculateDriftStatus(params.CoveragePct)

	var dr DriftReport
	err := r.pool.QueryRow(ctx, `
		INSERT INTO drift_reports (org_id, env_id, platform, site, total_assets, compliant_assets, coverage_pct, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, org_id, env_id, platform, site, total_assets, compliant_assets, coverage_pct, status, calculated_at
	`, params.OrgID, params.EnvID, params.Platform, params.Site,
		params.TotalAssets, params.CompliantAssets, params.CoveragePct, status,
	).Scan(
		&dr.ID, &dr.OrgID, &dr.EnvID, &dr.Platform, &dr.Site,
		&dr.TotalAssets, &dr.CompliantAssets, &dr.CoveragePct, &dr.Status, &dr.CalculatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &dr, nil
}

// GetDriftByEnvironment retrieves drift aggregated by environment.
func (r *Repository) GetDriftByEnvironment(ctx context.Context, orgID uuid.UUID) ([]DriftByScope, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			COALESCE(e.name, 'unassigned') as scope,
			COUNT(DISTINCT a.id) as total_assets,
			COUNT(DISTINCT CASE
				WHEN i.id IS NOT NULL AND a.image_version = i.version THEN a.id
			END) as compliant_assets
		FROM assets a
		LEFT JOIN environments e ON a.env_id = e.id
		LEFT JOIN images i ON a.org_id = i.org_id
			AND a.image_ref = i.family
			AND i.status = 'production'
		WHERE a.org_id = $1 AND a.state = 'running'
		GROUP BY e.id, e.name
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DriftByScope
	for rows.Next() {
		var d DriftByScope
		if err := rows.Scan(&d.Scope, &d.TotalAssets, &d.CompliantAssets); err != nil {
			return nil, err
		}
		if d.TotalAssets > 0 {
			d.CoveragePct = float64(d.CompliantAssets) / float64(d.TotalAssets) * 100
		}
		d.Status = calculateDriftStatus(d.CoveragePct)
		results = append(results, d)
	}
	return results, rows.Err()
}

// GetDriftByPlatform retrieves drift aggregated by platform.
func (r *Repository) GetDriftByPlatform(ctx context.Context, orgID uuid.UUID) ([]DriftByScope, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			a.platform as scope,
			COUNT(*) as total_assets,
			COUNT(CASE
				WHEN i.id IS NOT NULL AND a.image_version = i.version THEN 1
			END) as compliant_assets
		FROM assets a
		LEFT JOIN images i ON a.org_id = i.org_id
			AND a.image_ref = i.family
			AND i.status = 'production'
		WHERE a.org_id = $1 AND a.state = 'running'
		GROUP BY a.platform
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DriftByScope
	for rows.Next() {
		var d DriftByScope
		if err := rows.Scan(&d.Scope, &d.TotalAssets, &d.CompliantAssets); err != nil {
			return nil, err
		}
		if d.TotalAssets > 0 {
			d.CoveragePct = float64(d.CompliantAssets) / float64(d.TotalAssets) * 100
		}
		d.Status = calculateDriftStatus(d.CoveragePct)
		results = append(results, d)
	}
	return results, rows.Err()
}

// GetDriftBySite retrieves drift aggregated by site/region.
func (r *Repository) GetDriftBySite(ctx context.Context, orgID uuid.UUID) ([]DriftByScope, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			COALESCE(a.region, 'unknown') as scope,
			COUNT(*) as total_assets,
			COUNT(CASE
				WHEN i.id IS NOT NULL AND a.image_version = i.version THEN 1
			END) as compliant_assets
		FROM assets a
		LEFT JOIN images i ON a.org_id = i.org_id
			AND a.image_ref = i.family
			AND i.status = 'production'
		WHERE a.org_id = $1 AND a.state = 'running'
		GROUP BY a.region
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DriftByScope
	for rows.Next() {
		var d DriftByScope
		if err := rows.Scan(&d.Scope, &d.TotalAssets, &d.CompliantAssets); err != nil {
			return nil, err
		}
		if d.TotalAssets > 0 {
			d.CoveragePct = float64(d.CompliantAssets) / float64(d.TotalAssets) * 100
		}
		d.Status = calculateDriftStatus(d.CoveragePct)
		results = append(results, d)
	}
	return results, rows.Err()
}

// DriftTrendPoint represents a single point in drift trend data.
type DriftTrendPoint struct {
	Date            time.Time `json:"date"`
	AvgCoverage     float64   `json:"avg_coverage"`
	TotalAssets     int64     `json:"total_assets"`
	CompliantAssets int64     `json:"compliant_assets"`
}

// GetDriftTrend retrieves drift trend data for the specified number of days.
func (r *Repository) GetDriftTrend(ctx context.Context, orgID uuid.UUID, days int) ([]DriftTrendPoint, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			DATE_TRUNC('day', calculated_at) as date,
			AVG(coverage_pct) as avg_coverage,
			SUM(total_assets) as total_assets,
			SUM(compliant_assets) as compliant_assets
		FROM drift_reports
		WHERE org_id = $1
		  AND calculated_at >= NOW() - $2::interval
		GROUP BY DATE_TRUNC('day', calculated_at)
		ORDER BY date DESC
	`, orgID, time.Duration(days)*24*time.Hour)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []DriftTrendPoint
	for rows.Next() {
		var p DriftTrendPoint
		if err := rows.Scan(&p.Date, &p.AvgCoverage, &p.TotalAssets, &p.CompliantAssets); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// =============================================================================
// Helper Functions
// =============================================================================

// calculateDriftStatus determines drift status based on coverage percentage.
func calculateDriftStatus(coveragePct float64) string {
	if coveragePct >= 90 {
		return "healthy"
	} else if coveragePct >= 70 {
		return "warning"
	}
	return "critical"
}

// ErrNoRows is returned when a query returns no rows.
var ErrNoRows = pgx.ErrNoRows
