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

// SiteRepositoryAdapter adapts Repository to implement service.SiteRepository.
type SiteRepositoryAdapter struct {
	repo *Repository
}

// NewSiteRepositoryAdapter creates a new SiteRepositoryAdapter.
func NewSiteRepositoryAdapter(pool *pgxpool.Pool) *SiteRepositoryAdapter {
	return &SiteRepositoryAdapter{repo: New(pool)}
}

// GetSite returns a site by ID.
func (a *SiteRepositoryAdapter) GetSite(ctx context.Context, id uuid.UUID) (*service.Site, error) {
	site, err := a.repo.GetSite(ctx, id)
	if err != nil {
		return nil, err
	}
	return repoSiteToService(site), nil
}

// ListSites returns a list of sites.
func (a *SiteRepositoryAdapter) ListSites(ctx context.Context, params service.ListSitesParams) ([]service.Site, error) {
	repoParams := ListSitesParams{
		OrgID:    params.OrgID,
		Platform: params.Platform,
		Region:   params.Region,
		Limit:    params.Limit,
		Offset:   params.Offset,
	}

	sites, err := a.repo.ListSites(ctx, repoParams)
	if err != nil {
		return nil, err
	}

	result := make([]service.Site, 0, len(sites))
	for _, site := range sites {
		result = append(result, *repoSiteToService(&site))
	}
	return result, nil
}

// CountSitesByOrg counts sites for an organization.
func (a *SiteRepositoryAdapter) CountSitesByOrg(ctx context.Context, orgID uuid.UUID) (int64, error) {
	return a.repo.CountSitesByOrg(ctx, orgID)
}

// GetSiteWithAssetStats retrieves a site with computed asset statistics.
func (a *SiteRepositoryAdapter) GetSiteWithAssetStats(ctx context.Context, id uuid.UUID, orgID uuid.UUID) (*service.SiteWithStats, error) {
	site, err := a.repo.GetSiteWithAssetStats(ctx, id, orgID)
	if err != nil {
		return nil, err
	}
	return repoSiteWithStatsToService(site), nil
}

// ListSitesWithStats retrieves all sites with asset statistics.
func (a *SiteRepositoryAdapter) ListSitesWithStats(ctx context.Context, orgID uuid.UUID) ([]service.SiteWithStats, error) {
	sites, err := a.repo.ListSitesWithStats(ctx, orgID)
	if err != nil {
		return nil, err
	}

	result := make([]service.SiteWithStats, 0, len(sites))
	for _, site := range sites {
		result = append(result, *repoSiteWithStatsToService(&site))
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

func repoSiteToService(site *Site) *service.Site {
	return &service.Site{
		ID:             site.ID,
		OrgID:          site.OrgID,
		Name:           site.Name,
		Region:         site.Region,
		Platform:       site.Platform,
		Environment:    site.Environment,
		DRPairedSiteID: site.DRPairedSiteID,
		LastSyncAt:     site.LastSyncAt,
		Metadata:       site.Metadata,
		CreatedAt:      site.CreatedAt,
		UpdatedAt:      site.UpdatedAt,
	}
}

func repoSiteWithStatsToService(site *SiteWithStats) *service.SiteWithStats {
	return &service.SiteWithStats{
		Site: service.Site{
			ID:             site.ID,
			OrgID:          site.OrgID,
			Name:           site.Name,
			Region:         site.Region,
			Platform:       site.Platform,
			Environment:    site.Environment,
			DRPairedSiteID: site.DRPairedSiteID,
			LastSyncAt:     site.LastSyncAt,
			Metadata:       site.Metadata,
			CreatedAt:      site.CreatedAt,
			UpdatedAt:      site.UpdatedAt,
		},
		AssetCount:         site.AssetCount,
		CompliantCount:     site.CompliantCount,
		DriftedCount:       site.DriftedCount,
		CoveragePercentage: site.CoveragePercentage,
		Status:             site.Status,
		DRPaired:           site.DRPaired,
	}
}

// AlertRepositoryAdapter adapts Repository to implement service.AlertRepository.
type AlertRepositoryAdapter struct {
	repo *Repository
}

// NewAlertRepositoryAdapter creates a new AlertRepositoryAdapter.
func NewAlertRepositoryAdapter(pool *pgxpool.Pool) *AlertRepositoryAdapter {
	return &AlertRepositoryAdapter{repo: New(pool)}
}

// GetAlert returns an alert by ID.
func (a *AlertRepositoryAdapter) GetAlert(ctx context.Context, id uuid.UUID) (*service.Alert, error) {
	alert, err := a.repo.GetAlert(ctx, id)
	if err != nil {
		return nil, err
	}
	return repoAlertToService(alert), nil
}

// ListAlerts returns a list of alerts.
func (a *AlertRepositoryAdapter) ListAlerts(ctx context.Context, params service.ListAlertsParams) ([]service.Alert, error) {
	repoParams := ListAlertsParams{
		OrgID:    params.OrgID,
		Severity: params.Severity,
		Status:   params.Status,
		Source:   params.Source,
		SiteID:   params.SiteID,
		Limit:    params.Limit,
		Offset:   params.Offset,
	}

	alerts, err := a.repo.ListAlerts(ctx, repoParams)
	if err != nil {
		return nil, err
	}

	result := make([]service.Alert, 0, len(alerts))
	for _, alert := range alerts {
		result = append(result, *repoAlertToService(&alert))
	}
	return result, nil
}

// CountAlertsByOrg counts alerts for an organization.
func (a *AlertRepositoryAdapter) CountAlertsByOrg(ctx context.Context, orgID uuid.UUID) (int64, error) {
	return a.repo.CountAlertsByOrg(ctx, orgID)
}

// CountAlertsBySeverity counts alerts grouped by severity.
func (a *AlertRepositoryAdapter) CountAlertsBySeverity(ctx context.Context, orgID uuid.UUID) ([]service.AlertCount, error) {
	counts, err := a.repo.CountAlertsBySeverity(ctx, orgID)
	if err != nil {
		return nil, err
	}

	result := make([]service.AlertCount, 0, len(counts))
	for _, c := range counts {
		result = append(result, service.AlertCount{
			Severity: c.Severity,
			Count:    c.Count,
		})
	}
	return result, nil
}

// UpdateAlertStatus updates an alert's status.
func (a *AlertRepositoryAdapter) UpdateAlertStatus(ctx context.Context, id uuid.UUID, status string, userID *uuid.UUID) error {
	return a.repo.UpdateAlertStatus(ctx, id, status, userID)
}

// CreateAlert creates a new alert.
func (a *AlertRepositoryAdapter) CreateAlert(ctx context.Context, params service.CreateAlertParams) (*service.Alert, error) {
	repoParams := CreateAlertParams{
		OrgID:       params.OrgID,
		Severity:    params.Severity,
		Title:       params.Title,
		Description: params.Description,
		Source:      params.Source,
		SiteID:      params.SiteID,
		AssetID:     params.AssetID,
		ImageID:     params.ImageID,
	}

	alert, err := a.repo.CreateAlert(ctx, repoParams)
	if err != nil {
		return nil, err
	}
	return repoAlertToService(alert), nil
}

func repoAlertToService(alert *Alert) *service.Alert {
	return &service.Alert{
		ID:             alert.ID,
		OrgID:          alert.OrgID,
		Severity:       alert.Severity,
		Title:          alert.Title,
		Description:    alert.Description,
		Source:         alert.Source,
		SiteID:         alert.SiteID,
		AssetID:        alert.AssetID,
		ImageID:        alert.ImageID,
		Status:         alert.Status,
		CreatedAt:      alert.CreatedAt,
		AcknowledgedAt: alert.AcknowledgedAt,
		AcknowledgedBy: alert.AcknowledgedBy,
		ResolvedAt:     alert.ResolvedAt,
		ResolvedBy:     alert.ResolvedBy,
	}
}

// ActivityRepositoryAdapter adapts Repository to implement service.ActivityRepository.
type ActivityRepositoryAdapter struct {
	repo *Repository
}

// NewActivityRepositoryAdapter creates a new ActivityRepositoryAdapter.
func NewActivityRepositoryAdapter(pool *pgxpool.Pool) *ActivityRepositoryAdapter {
	return &ActivityRepositoryAdapter{repo: New(pool)}
}

// ListRecentActivities retrieves recent activities.
func (a *ActivityRepositoryAdapter) ListRecentActivities(ctx context.Context, orgID uuid.UUID, limit int) ([]service.Activity, error) {
	activities, err := a.repo.ListRecentActivities(ctx, orgID, limit)
	if err != nil {
		return nil, err
	}

	result := make([]service.Activity, 0, len(activities))
	for _, act := range activities {
		result = append(result, *repoActivityToService(&act))
	}
	return result, nil
}

// CreateActivity creates a new activity.
func (a *ActivityRepositoryAdapter) CreateActivity(ctx context.Context, params service.CreateActivityParams) (*service.Activity, error) {
	repoParams := CreateActivityParams{
		OrgID:   params.OrgID,
		Type:    params.Type,
		Action:  params.Action,
		Detail:  params.Detail,
		UserID:  params.UserID,
		SiteID:  params.SiteID,
		AssetID: params.AssetID,
		ImageID: params.ImageID,
	}

	activity, err := a.repo.CreateActivity(ctx, repoParams)
	if err != nil {
		return nil, err
	}
	return repoActivityToService(activity), nil
}

func repoActivityToService(activity *Activity) *service.Activity {
	return &service.Activity{
		ID:        activity.ID,
		OrgID:     activity.OrgID,
		Type:      activity.Type,
		Action:    activity.Action,
		Detail:    activity.Detail,
		UserID:    activity.UserID,
		SiteID:    activity.SiteID,
		AssetID:   activity.AssetID,
		ImageID:   activity.ImageID,
		Timestamp: activity.Timestamp,
	}
}
