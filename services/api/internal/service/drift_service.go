package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// DriftService handles drift business logic.
type DriftService struct {
	driftRepo DriftRepository
	assetRepo AssetRepository
	imageRepo ImageRepository
}

// NewDriftService creates a new DriftService.
func NewDriftService(driftRepo DriftRepository, assetRepo AssetRepository, imageRepo ImageRepository) *DriftService {
	return &DriftService{
		driftRepo: driftRepo,
		assetRepo: assetRepo,
		imageRepo: imageRepo,
	}
}

// GetCurrentDriftInput contains input for getting current drift.
type GetCurrentDriftInput struct {
	OrgID uuid.UUID
}

// CurrentDrift contains current drift status.
type CurrentDrift struct {
	TotalAssets     int64   `json:"total_assets"`
	CompliantAssets int64   `json:"compliant_assets"`
	CoveragePct     float64 `json:"coverage_pct"`
	Status          string  `json:"status"`
}

// GetCurrentDrift retrieves the current drift status.
func (s *DriftService) GetCurrentDrift(ctx context.Context, input GetCurrentDriftInput) (*CurrentDrift, error) {
	// Get running assets count
	running, err := s.assetRepo.CountAssetsByState(ctx, input.OrgID, "running")
	if err != nil {
		return nil, fmt.Errorf("count running assets: %w", err)
	}

	// Get compliant assets count
	compliant, err := s.assetRepo.CountCompliantAssets(ctx, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("count compliant assets: %w", err)
	}

	// Calculate coverage
	coveragePct := 0.0
	if running > 0 {
		coveragePct = float64(compliant) / float64(running) * 100
	}

	status := calculateStatus(coveragePct)

	return &CurrentDrift{
		TotalAssets:     running,
		CompliantAssets: compliant,
		CoveragePct:     coveragePct,
		Status:          status,
	}, nil
}

// GetDriftSummaryInput contains input for getting drift summary.
type GetDriftSummaryInput struct {
	OrgID uuid.UUID
}

// DriftSummary contains drift summary by different dimensions.
type DriftSummary struct {
	Overall       CurrentDrift   `json:"overall"`
	ByEnvironment []DriftByScope `json:"by_environment"`
	ByPlatform    []DriftByScope `json:"by_platform"`
	BySite        []DriftByScope `json:"by_site"`
}

// GetDriftSummary retrieves drift summary across all dimensions.
func (s *DriftService) GetDriftSummary(ctx context.Context, input GetDriftSummaryInput) (*DriftSummary, error) {
	// Get overall drift
	overall, err := s.GetCurrentDrift(ctx, GetCurrentDriftInput{OrgID: input.OrgID})
	if err != nil {
		return nil, fmt.Errorf("get current drift: %w", err)
	}

	// Get drift by environment
	byEnv, err := s.driftRepo.GetDriftByEnvironment(ctx, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("get drift by environment: %w", err)
	}

	// Get drift by platform
	byPlatform, err := s.driftRepo.GetDriftByPlatform(ctx, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("get drift by platform: %w", err)
	}

	// Get drift by site
	bySite, err := s.driftRepo.GetDriftBySite(ctx, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("get drift by site: %w", err)
	}

	return &DriftSummary{
		Overall:       *overall,
		ByEnvironment: byEnv,
		ByPlatform:    byPlatform,
		BySite:        bySite,
	}, nil
}

// GetDriftTrendsInput contains input for getting drift trends.
type GetDriftTrendsInput struct {
	OrgID uuid.UUID
	Days  int
}

// GetDriftTrends retrieves drift trends over time.
func (s *DriftService) GetDriftTrends(ctx context.Context, input GetDriftTrendsInput) ([]DriftTrendPoint, error) {
	// Default to 30 days
	if input.Days <= 0 {
		input.Days = 30
	}
	if input.Days > 365 {
		input.Days = 365
	}

	trends, err := s.driftRepo.GetDriftTrend(ctx, input.OrgID, input.Days)
	if err != nil {
		return nil, fmt.Errorf("get drift trend: %w", err)
	}

	return trends, nil
}

// GetDriftReportInput contains input for getting a single drift report.
type GetDriftReportInput struct {
	ID    uuid.UUID
	OrgID uuid.UUID
}

// GetDriftReport retrieves a single drift report by ID.
func (s *DriftService) GetDriftReport(ctx context.Context, input GetDriftReportInput) (*DriftReport, error) {
	report, err := s.driftRepo.GetDriftReport(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	// Verify organization ownership
	if report.OrgID != input.OrgID {
		return nil, ErrNotFound
	}

	return report, nil
}

// ListDriftReportsInput contains input for listing drift reports.
type ListDriftReportsInput struct {
	OrgID    uuid.UUID
	Page     int
	PageSize int
}

// ListDriftReportsOutput contains output for listing drift reports.
type ListDriftReportsOutput struct {
	Reports    []DriftReport `json:"reports"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

// ListDriftReports retrieves historical drift reports.
func (s *DriftService) ListDriftReports(ctx context.Context, input ListDriftReportsInput) (*ListDriftReportsOutput, error) {
	// Apply defaults
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	offset := int32((input.Page - 1) * input.PageSize)

	reports, err := s.driftRepo.ListDriftReports(ctx, input.OrgID, int32(input.PageSize), offset)
	if err != nil {
		return nil, fmt.Errorf("list drift reports: %w", err)
	}

	return &ListDriftReportsOutput{
		Reports:  reports,
		Page:     input.Page,
		PageSize: input.PageSize,
	}, nil
}

// CalculateDriftInput contains input for calculating drift.
type CalculateDriftInput struct {
	OrgID    uuid.UUID
	EnvID    *uuid.UUID
	Platform *string
	Site     *string
}

// CalculateDrift calculates and stores a drift report.
func (s *DriftService) CalculateDrift(ctx context.Context, input CalculateDriftInput) (*DriftReport, error) {
	// Get running assets count
	running, err := s.assetRepo.CountAssetsByState(ctx, input.OrgID, "running")
	if err != nil {
		return nil, fmt.Errorf("count running assets: %w", err)
	}

	// Get compliant assets count
	compliant, err := s.assetRepo.CountCompliantAssets(ctx, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("count compliant assets: %w", err)
	}

	// Calculate coverage
	coveragePct := 0.0
	if running > 0 {
		coveragePct = float64(compliant) / float64(running) * 100
	}

	// Store drift report
	report, err := s.driftRepo.CreateDriftReport(ctx, CreateDriftReportParams{
		OrgID:           input.OrgID,
		EnvID:           input.EnvID,
		Platform:        input.Platform,
		Site:            input.Site,
		TotalAssets:     int(running),
		CompliantAssets: int(compliant),
		CoveragePct:     coveragePct,
	})
	if err != nil {
		return nil, fmt.Errorf("create drift report: %w", err)
	}

	return report, nil
}

// calculateStatus determines drift status based on coverage percentage.
func calculateStatus(coveragePct float64) string {
	if coveragePct >= 90 {
		return "healthy"
	} else if coveragePct >= 70 {
		return "warning"
	}
	return "critical"
}
