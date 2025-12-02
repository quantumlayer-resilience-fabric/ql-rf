// Package repository provides adapters to implement service interfaces.
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
)

// ImageRepositoryAdapter adapts Repository to implement service.ImageRepository.
type ImageRepositoryAdapter struct {
	repo *Repository
}

// NewImageRepositoryAdapter creates a new ImageRepositoryAdapter.
func NewImageRepositoryAdapter(pool *pgxpool.Pool) *ImageRepositoryAdapter {
	return &ImageRepositoryAdapter{repo: New(pool)}
}

// GetImage returns an image by ID.
func (a *ImageRepositoryAdapter) GetImage(ctx context.Context, id uuid.UUID) (*service.Image, error) {
	img, err := a.repo.GetImage(ctx, id)
	if err != nil {
		return nil, err
	}
	return repoImageToService(img), nil
}

// GetLatestImageByFamily returns the latest production image for a family.
func (a *ImageRepositoryAdapter) GetLatestImageByFamily(ctx context.Context, orgID uuid.UUID, family string) (*service.Image, error) {
	img, err := a.repo.GetLatestImageByFamily(ctx, orgID, family)
	if err != nil {
		return nil, err
	}
	return repoImageToService(img), nil
}

// ListImages returns a list of images.
func (a *ImageRepositoryAdapter) ListImages(ctx context.Context, params service.ListImagesParams) ([]service.Image, error) {
	repoParams := ListImagesParams{
		OrgID:  params.OrgID,
		Family: params.Family,
		Status: params.Status,
		Limit:  params.Limit,
		Offset: params.Offset,
	}

	images, err := a.repo.ListImages(ctx, repoParams)
	if err != nil {
		return nil, err
	}

	result := make([]service.Image, 0, len(images))
	for _, img := range images {
		result = append(result, *repoImageToService(&img))
	}
	return result, nil
}

// CreateImage creates a new image.
func (a *ImageRepositoryAdapter) CreateImage(ctx context.Context, params service.CreateImageParams) (*service.Image, error) {
	repoParams := CreateImageParams{
		OrgID:     params.OrgID,
		Family:    params.Family,
		Version:   params.Version,
		OSName:    params.OSName,
		OSVersion: params.OSVersion,
		CISLevel:  params.CISLevel,
		SBOMUrl:   params.SBOMUrl,
		Signed:    params.Signed,
		Status:    params.Status,
	}

	img, err := a.repo.CreateImage(ctx, repoParams)
	if err != nil {
		return nil, err
	}
	return repoImageToService(img), nil
}

// UpdateImageStatus updates an image's status.
func (a *ImageRepositoryAdapter) UpdateImageStatus(ctx context.Context, id uuid.UUID, status string) (*service.Image, error) {
	img, err := a.repo.UpdateImageStatus(ctx, id, status)
	if err != nil {
		return nil, err
	}
	return repoImageToService(img), nil
}

// CountImagesByOrg counts images for an organization.
func (a *ImageRepositoryAdapter) CountImagesByOrg(ctx context.Context, orgID uuid.UUID) (int64, error) {
	return a.repo.CountImagesByOrg(ctx, orgID)
}

// GetImageCoordinates returns coordinates for an image.
func (a *ImageRepositoryAdapter) GetImageCoordinates(ctx context.Context, imageID uuid.UUID) ([]service.ImageCoordinate, error) {
	coords, err := a.repo.GetImageCoordinates(ctx, imageID)
	if err != nil {
		return nil, err
	}

	result := make([]service.ImageCoordinate, 0, len(coords))
	for _, c := range coords {
		result = append(result, *repoCoordinateToService(&c))
	}
	return result, nil
}

// CreateImageCoordinate creates a new image coordinate.
func (a *ImageRepositoryAdapter) CreateImageCoordinate(ctx context.Context, params service.CreateImageCoordinateParams) (*service.ImageCoordinate, error) {
	repoParams := CreateImageCoordinateParams{
		ImageID:    params.ImageID,
		Platform:   params.Platform,
		Region:     params.Region,
		Identifier: params.Identifier,
	}

	coord, err := a.repo.CreateImageCoordinate(ctx, repoParams)
	if err != nil {
		return nil, err
	}
	return repoCoordinateToService(coord), nil
}

// AssetRepositoryAdapter adapts Repository to implement service.AssetRepository.
type AssetRepositoryAdapter struct {
	repo *Repository
}

// NewAssetRepositoryAdapter creates a new AssetRepositoryAdapter.
func NewAssetRepositoryAdapter(pool *pgxpool.Pool) *AssetRepositoryAdapter {
	return &AssetRepositoryAdapter{repo: New(pool)}
}

// GetAsset returns an asset by ID.
func (a *AssetRepositoryAdapter) GetAsset(ctx context.Context, id uuid.UUID) (*service.Asset, error) {
	asset, err := a.repo.GetAsset(ctx, id)
	if err != nil {
		return nil, err
	}
	return repoAssetToService(asset), nil
}

// GetAssetByInstanceID returns an asset by instance ID.
func (a *AssetRepositoryAdapter) GetAssetByInstanceID(ctx context.Context, orgID uuid.UUID, platform, instanceID string) (*service.Asset, error) {
	asset, err := a.repo.GetAssetByInstanceID(ctx, orgID, platform, instanceID)
	if err != nil {
		return nil, err
	}
	return repoAssetToService(asset), nil
}

// ListAssets returns a list of assets.
func (a *AssetRepositoryAdapter) ListAssets(ctx context.Context, params service.ListAssetsParams) ([]service.Asset, error) {
	repoParams := ListAssetsParams{
		OrgID:    params.OrgID,
		EnvID:    params.EnvID,
		Platform: params.Platform,
		State:    params.State,
		Limit:    params.Limit,
		Offset:   params.Offset,
	}

	assets, err := a.repo.ListAssets(ctx, repoParams)
	if err != nil {
		return nil, err
	}

	result := make([]service.Asset, 0, len(assets))
	for _, asset := range assets {
		result = append(result, *repoAssetToService(&asset))
	}
	return result, nil
}

// UpsertAsset creates or updates an asset.
func (a *AssetRepositoryAdapter) UpsertAsset(ctx context.Context, params service.UpsertAssetParams) (*service.Asset, error) {
	repoParams := UpsertAssetParams{
		OrgID:        params.OrgID,
		EnvID:        params.EnvID,
		Platform:     params.Platform,
		Account:      params.Account,
		Region:       params.Region,
		InstanceID:   params.InstanceID,
		ImageRef:     params.ImageRef,
		ImageVersion: params.ImageVersion,
		State:        params.State,
		Tags:         params.Tags,
	}

	asset, err := a.repo.UpsertAsset(ctx, repoParams)
	if err != nil {
		return nil, err
	}
	return repoAssetToService(asset), nil
}

// DeleteAsset deletes an asset.
func (a *AssetRepositoryAdapter) DeleteAsset(ctx context.Context, id uuid.UUID) error {
	return a.repo.DeleteAsset(ctx, id)
}

// CountAssetsByOrg counts assets for an organization.
func (a *AssetRepositoryAdapter) CountAssetsByOrg(ctx context.Context, orgID uuid.UUID) (int64, error) {
	return a.repo.CountAssetsByOrg(ctx, orgID)
}

// CountAssetsByState counts assets by state.
func (a *AssetRepositoryAdapter) CountAssetsByState(ctx context.Context, orgID uuid.UUID, state string) (int64, error) {
	return a.repo.CountAssetsByState(ctx, orgID, state)
}

// CountCompliantAssets counts compliant assets.
func (a *AssetRepositoryAdapter) CountCompliantAssets(ctx context.Context, orgID uuid.UUID) (int64, error) {
	return a.repo.CountCompliantAssets(ctx, orgID)
}

// DriftRepositoryAdapter adapts Repository to implement service.DriftRepository.
type DriftRepositoryAdapter struct {
	repo *Repository
}

// NewDriftRepositoryAdapter creates a new DriftRepositoryAdapter.
func NewDriftRepositoryAdapter(pool *pgxpool.Pool) *DriftRepositoryAdapter {
	return &DriftRepositoryAdapter{repo: New(pool)}
}

// GetLatestDriftReport returns the latest drift report for an org.
func (a *DriftRepositoryAdapter) GetLatestDriftReport(ctx context.Context, orgID uuid.UUID) (*service.DriftReport, error) {
	report, err := a.repo.GetLatestDriftReport(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return repoDriftReportToService(report), nil
}

// ListDriftReports returns a paginated list of drift reports.
func (a *DriftRepositoryAdapter) ListDriftReports(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]service.DriftReport, error) {
	reports, err := a.repo.ListDriftReports(ctx, orgID, limit, offset)
	if err != nil {
		return nil, err
	}

	result := make([]service.DriftReport, 0, len(reports))
	for _, r := range reports {
		result = append(result, *repoDriftReportToService(&r))
	}
	return result, nil
}

// CreateDriftReport creates a new drift report.
func (a *DriftRepositoryAdapter) CreateDriftReport(ctx context.Context, params service.CreateDriftReportParams) (*service.DriftReport, error) {
	repoParams := CreateDriftReportParams{
		OrgID:           params.OrgID,
		EnvID:           params.EnvID,
		Platform:        params.Platform,
		Site:            params.Site,
		TotalAssets:     params.TotalAssets,
		CompliantAssets: params.CompliantAssets,
		CoveragePct:     params.CoveragePct,
	}

	report, err := a.repo.CreateDriftReport(ctx, repoParams)
	if err != nil {
		return nil, err
	}
	return repoDriftReportToService(report), nil
}

// GetDriftByEnvironment returns drift grouped by environment.
func (a *DriftRepositoryAdapter) GetDriftByEnvironment(ctx context.Context, orgID uuid.UUID) ([]service.DriftByScope, error) {
	scopes, err := a.repo.GetDriftByEnvironment(ctx, orgID)
	if err != nil {
		return nil, err
	}

	result := make([]service.DriftByScope, 0, len(scopes))
	for _, s := range scopes {
		result = append(result, service.DriftByScope{
			Scope:           s.Scope,
			TotalAssets:     s.TotalAssets,
			CompliantAssets: s.CompliantAssets,
			CoveragePct:     s.CoveragePct,
			Status:          s.Status,
		})
	}
	return result, nil
}

// GetDriftByPlatform returns drift grouped by platform.
func (a *DriftRepositoryAdapter) GetDriftByPlatform(ctx context.Context, orgID uuid.UUID) ([]service.DriftByScope, error) {
	scopes, err := a.repo.GetDriftByPlatform(ctx, orgID)
	if err != nil {
		return nil, err
	}

	result := make([]service.DriftByScope, 0, len(scopes))
	for _, s := range scopes {
		result = append(result, service.DriftByScope{
			Scope:           s.Scope,
			TotalAssets:     s.TotalAssets,
			CompliantAssets: s.CompliantAssets,
			CoveragePct:     s.CoveragePct,
			Status:          s.Status,
		})
	}
	return result, nil
}

// GetDriftBySite returns drift grouped by site.
func (a *DriftRepositoryAdapter) GetDriftBySite(ctx context.Context, orgID uuid.UUID) ([]service.DriftByScope, error) {
	scopes, err := a.repo.GetDriftBySite(ctx, orgID)
	if err != nil {
		return nil, err
	}

	result := make([]service.DriftByScope, 0, len(scopes))
	for _, s := range scopes {
		result = append(result, service.DriftByScope{
			Scope:           s.Scope,
			TotalAssets:     s.TotalAssets,
			CompliantAssets: s.CompliantAssets,
			CoveragePct:     s.CoveragePct,
			Status:          s.Status,
		})
	}
	return result, nil
}

// GetDriftTrend returns drift trend over time.
func (a *DriftRepositoryAdapter) GetDriftTrend(ctx context.Context, orgID uuid.UUID, days int) ([]service.DriftTrendPoint, error) {
	trends, err := a.repo.GetDriftTrend(ctx, orgID, days)
	if err != nil {
		return nil, err
	}

	result := make([]service.DriftTrendPoint, 0, len(trends))
	for _, t := range trends {
		result = append(result, service.DriftTrendPoint{
			Date:            t.Date,
			AvgCoverage:     t.AvgCoverage,
			TotalAssets:     t.TotalAssets,
			CompliantAssets: t.CompliantAssets,
		})
	}
	return result, nil
}

// Helper functions to convert between repository and service types
func repoImageToService(img *Image) *service.Image {
	return &service.Image{
		ID:        img.ID,
		OrgID:     img.OrgID,
		Family:    img.Family,
		Version:   img.Version,
		OSName:    img.OSName,
		OSVersion: img.OSVersion,
		CISLevel:  img.CISLevel,
		SBOMUrl:   img.SBOMUrl,
		Signed:    img.Signed,
		Status:    img.Status,
		CreatedAt: img.CreatedAt,
		UpdatedAt: img.UpdatedAt,
	}
}

func repoCoordinateToService(coord *ImageCoordinate) *service.ImageCoordinate {
	return &service.ImageCoordinate{
		ID:         coord.ID,
		ImageID:    coord.ImageID,
		Platform:   coord.Platform,
		Region:     coord.Region,
		Identifier: coord.Identifier,
		CreatedAt:  coord.CreatedAt,
	}
}

func repoAssetToService(asset *Asset) *service.Asset {
	return &service.Asset{
		ID:           asset.ID,
		OrgID:        asset.OrgID,
		EnvID:        asset.EnvID,
		Platform:     asset.Platform,
		Account:      asset.Account,
		Region:       asset.Region,
		Site:         asset.Site,
		InstanceID:   asset.InstanceID,
		Name:         asset.Name,
		ImageRef:     asset.ImageRef,
		ImageVersion: asset.ImageVersion,
		State:        asset.State,
		Tags:         asset.Tags,
		DiscoveredAt: asset.DiscoveredAt,
		UpdatedAt:    asset.UpdatedAt,
	}
}

func repoDriftReportToService(report *DriftReport) *service.DriftReport {
	return &service.DriftReport{
		ID:              report.ID,
		OrgID:           report.OrgID,
		EnvID:           report.EnvID,
		Platform:        report.Platform,
		Site:            report.Site,
		TotalAssets:     report.TotalAssets,
		CompliantAssets: report.CompliantAssets,
		CoveragePct:     report.CoveragePct,
		Status:          report.Status,
		CalculatedAt:    report.CalculatedAt,
	}
}
