// Package service provides business logic for the API.
package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Repository Interfaces - For dependency injection and testing
// =============================================================================

// ImageRepository defines the interface for image data access.
type ImageRepository interface {
	GetImage(ctx context.Context, id uuid.UUID) (*Image, error)
	GetLatestImageByFamily(ctx context.Context, orgID uuid.UUID, family string) (*Image, error)
	ListImages(ctx context.Context, params ListImagesParams) ([]Image, error)
	CreateImage(ctx context.Context, params CreateImageParams) (*Image, error)
	UpdateImageStatus(ctx context.Context, id uuid.UUID, status string) (*Image, error)
	CountImagesByOrg(ctx context.Context, orgID uuid.UUID) (int64, error)
	GetImageCoordinates(ctx context.Context, imageID uuid.UUID) ([]ImageCoordinate, error)
	CreateImageCoordinate(ctx context.Context, params CreateImageCoordinateParams) (*ImageCoordinate, error)
}

// AssetRepository defines the interface for asset data access.
type AssetRepository interface {
	GetAsset(ctx context.Context, id uuid.UUID) (*Asset, error)
	GetAssetByInstanceID(ctx context.Context, orgID uuid.UUID, platform, instanceID string) (*Asset, error)
	ListAssets(ctx context.Context, params ListAssetsParams) ([]Asset, error)
	UpsertAsset(ctx context.Context, params UpsertAssetParams) (*Asset, error)
	DeleteAsset(ctx context.Context, id uuid.UUID) error
	CountAssetsByOrg(ctx context.Context, orgID uuid.UUID) (int64, error)
	CountAssetsByState(ctx context.Context, orgID uuid.UUID, state string) (int64, error)
	CountCompliantAssets(ctx context.Context, orgID uuid.UUID) (int64, error)
}

// DriftRepository defines the interface for drift data access.
type DriftRepository interface {
	GetLatestDriftReport(ctx context.Context, orgID uuid.UUID) (*DriftReport, error)
	ListDriftReports(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]DriftReport, error)
	CreateDriftReport(ctx context.Context, params CreateDriftReportParams) (*DriftReport, error)
	GetDriftByEnvironment(ctx context.Context, orgID uuid.UUID) ([]DriftByScope, error)
	GetDriftByPlatform(ctx context.Context, orgID uuid.UUID) ([]DriftByScope, error)
	GetDriftBySite(ctx context.Context, orgID uuid.UUID) ([]DriftByScope, error)
	GetDriftTrend(ctx context.Context, orgID uuid.UUID, days int) ([]DriftTrendPoint, error)
}

// OrganizationRepository defines the interface for organization data access.
type OrganizationRepository interface {
	GetOrganization(ctx context.Context, id uuid.UUID) (*Organization, error)
	GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error)
}

// =============================================================================
// Domain Models - Used by services and repositories
// =============================================================================

// Organization represents a tenant.
type Organization struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Image represents a golden image.
type Image struct {
	ID          uuid.UUID         `json:"id"`
	OrgID       uuid.UUID         `json:"org_id"`
	Family      string            `json:"family"`
	Version     string            `json:"version"`
	OSName      *string           `json:"os_name,omitempty"`
	OSVersion   *string           `json:"os_version,omitempty"`
	CISLevel    *int              `json:"cis_level,omitempty"`
	SBOMUrl     *string           `json:"sbom_url,omitempty"`
	Signed      bool              `json:"signed"`
	Status      string            `json:"status"`
	Coordinates []ImageCoordinate `json:"coordinates,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// ImageCoordinate represents platform-specific image location.
type ImageCoordinate struct {
	ID         uuid.UUID `json:"id"`
	ImageID    uuid.UUID `json:"image_id"`
	Platform   string    `json:"platform"`
	Region     *string   `json:"region,omitempty"`
	Identifier string    `json:"identifier"`
	CreatedAt  time.Time `json:"created_at"`
}

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

// DriftByScope represents drift aggregated by a scope.
type DriftByScope struct {
	Scope           string  `json:"scope"`
	TotalAssets     int64   `json:"total_assets"`
	CompliantAssets int64   `json:"compliant_assets"`
	CoveragePct     float64 `json:"coverage_pct"`
	Status          string  `json:"status"`
}

// DriftTrendPoint represents a single point in drift trend.
type DriftTrendPoint struct {
	Date            time.Time `json:"date"`
	AvgCoverage     float64   `json:"avg_coverage"`
	TotalAssets     int64     `json:"total_assets"`
	CompliantAssets int64     `json:"compliant_assets"`
}

// =============================================================================
// Parameter Types
// =============================================================================

// ListImagesParams contains parameters for listing images.
type ListImagesParams struct {
	OrgID  uuid.UUID
	Family *string
	Status *string
	Limit  int32
	Offset int32
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

// CreateImageCoordinateParams contains parameters for creating an image coordinate.
type CreateImageCoordinateParams struct {
	ImageID    uuid.UUID
	Platform   string
	Region     *string
	Identifier string
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
