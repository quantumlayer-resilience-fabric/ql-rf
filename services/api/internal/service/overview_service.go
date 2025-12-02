package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// OverviewService handles overview/dashboard business logic.
type OverviewService struct {
	assetRepo    AssetRepository
	driftRepo    DriftRepository
	siteRepo     SiteRepository
	alertRepo    AlertRepository
	activityRepo ActivityRepository
}

// NewOverviewService creates a new OverviewService.
func NewOverviewService(
	assetRepo AssetRepository,
	driftRepo DriftRepository,
	siteRepo SiteRepository,
	alertRepo AlertRepository,
	activityRepo ActivityRepository,
) *OverviewService {
	return &OverviewService{
		assetRepo:    assetRepo,
		driftRepo:    driftRepo,
		siteRepo:     siteRepo,
		alertRepo:    alertRepo,
		activityRepo: activityRepo,
	}
}

// GetOverviewMetricsInput contains input for getting overview metrics.
type GetOverviewMetricsInput struct {
	OrgID uuid.UUID
}

// MetricTrend represents the trend direction for a metric.
type MetricTrend struct {
	Direction string `json:"direction"` // up, down, neutral
	Value     string `json:"value"`     // e.g., "+5%", "-2%"
	Period    string `json:"period"`    // e.g., "vs last 7 days"
}

// MetricWithTrend represents a metric value with its trend.
type MetricWithTrend struct {
	Value int64       `json:"value"`
	Trend MetricTrend `json:"trend"`
}

// FloatMetricWithTrend represents a float metric value with its trend.
type FloatMetricWithTrend struct {
	Value float64     `json:"value"`
	Trend MetricTrend `json:"trend"`
}

// PlatformCount represents asset count by platform.
type PlatformCount struct {
	Platform   string  `json:"platform"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// OverviewMetrics represents the dashboard overview metrics.
type OverviewMetrics struct {
	FleetSize            MetricWithTrend      `json:"fleetSize"`
	DriftScore           FloatMetricWithTrend `json:"driftScore"`
	Compliance           FloatMetricWithTrend `json:"compliance"`
	DRReadiness          FloatMetricWithTrend `json:"drReadiness"`
	PlatformDistribution []PlatformCount      `json:"platformDistribution"`
	Alerts               []AlertCount         `json:"alerts"`
	RecentActivity       []Activity           `json:"recentActivity"`
}

// GetOverviewMetrics retrieves dashboard overview metrics.
func (s *OverviewService) GetOverviewMetrics(ctx context.Context, input GetOverviewMetricsInput) (*OverviewMetrics, error) {
	// Get fleet size (total assets)
	fleetSize, err := s.assetRepo.CountAssetsByOrg(ctx, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("count assets: %w", err)
	}

	// Get latest drift report for drift score
	var driftScore float64 = 100.0 // Default to 100% if no drift data
	driftReport, err := s.driftRepo.GetLatestDriftReport(ctx, input.OrgID)
	if err == nil && driftReport != nil {
		driftScore = driftReport.CoveragePct
	}

	// Get compliance (compliant assets percentage)
	compliantCount, err := s.assetRepo.CountCompliantAssets(ctx, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("count compliant assets: %w", err)
	}
	var compliance float64 = 100.0
	if fleetSize > 0 {
		compliance = float64(compliantCount) / float64(fleetSize) * 100
	}

	// Get DR readiness (percentage of sites with DR pairs)
	sites, err := s.siteRepo.ListSitesWithStats(ctx, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}
	var drReadiness float64 = 0.0
	drPairedCount := 0
	for _, site := range sites {
		if site.DRPaired {
			drPairedCount++
		}
	}
	if len(sites) > 0 {
		drReadiness = float64(drPairedCount) / float64(len(sites)) * 100
	}

	// Get platform distribution
	driftByPlatform, err := s.driftRepo.GetDriftByPlatform(ctx, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("get drift by platform: %w", err)
	}
	platformDist := make([]PlatformCount, 0, len(driftByPlatform))
	for _, d := range driftByPlatform {
		pct := float64(0)
		if fleetSize > 0 {
			pct = float64(d.TotalAssets) / float64(fleetSize) * 100
		}
		platformDist = append(platformDist, PlatformCount{
			Platform:   d.Scope,
			Count:      int(d.TotalAssets),
			Percentage: pct,
		})
	}

	// Get alert counts by severity
	alertCounts, err := s.alertRepo.CountAlertsBySeverity(ctx, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("count alerts by severity: %w", err)
	}

	// Get recent activities
	activities, err := s.activityRepo.ListRecentActivities(ctx, input.OrgID, 10)
	if err != nil {
		return nil, fmt.Errorf("list recent activities: %w", err)
	}

	return &OverviewMetrics{
		FleetSize: MetricWithTrend{
			Value: fleetSize,
			Trend: MetricTrend{
				Direction: "neutral",
				Value:     "0%",
				Period:    "vs last 7 days",
			},
		},
		DriftScore: FloatMetricWithTrend{
			Value: driftScore,
			Trend: MetricTrend{
				Direction: "neutral",
				Value:     "0%",
				Period:    "vs last 7 days",
			},
		},
		Compliance: FloatMetricWithTrend{
			Value: compliance,
			Trend: MetricTrend{
				Direction: "neutral",
				Value:     "0%",
				Period:    "vs last 7 days",
			},
		},
		DRReadiness: FloatMetricWithTrend{
			Value: drReadiness,
			Trend: MetricTrend{
				Direction: "neutral",
				Value:     "0%",
				Period:    "vs last 7 days",
			},
		},
		PlatformDistribution: platformDist,
		Alerts:               alertCounts,
		RecentActivity:       activities,
	}, nil
}
