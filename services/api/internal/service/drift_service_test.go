package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/service/mocks"
)

func TestDriftService_GetCurrentDrift(t *testing.T) {
	ctx := context.Background()

	t.Run("healthy status >= 90%", func(t *testing.T) {
		assetRepo := mocks.NewMockAssetRepository()
		driftRepo := mocks.NewMockDriftRepository()
		imageRepo := mocks.NewMockImageRepository()
		svc := service.NewDriftService(driftRepo, assetRepo, imageRepo)

		orgID := uuid.New()

		// Setup: 100 running, 95 compliant = 95%
		assetRepo.CountAssetsByStateFunc = func(ctx context.Context, orgID uuid.UUID, state string) (int64, error) {
			if state == "running" {
				return 100, nil
			}
			return 0, nil
		}
		assetRepo.CountCompliantAssetsFunc = func(ctx context.Context, orgID uuid.UUID) (int64, error) {
			return 95, nil
		}

		result, err := svc.GetCurrentDrift(ctx, service.GetCurrentDriftInput{
			OrgID: orgID,
		})

		require.NoError(t, err)
		assert.Equal(t, int64(100), result.TotalAssets)
		assert.Equal(t, int64(95), result.CompliantAssets)
		assert.Equal(t, 95.0, result.CoveragePct)
		assert.Equal(t, "healthy", result.Status)
	})

	t.Run("warning status 70-89%", func(t *testing.T) {
		assetRepo := mocks.NewMockAssetRepository()
		driftRepo := mocks.NewMockDriftRepository()
		imageRepo := mocks.NewMockImageRepository()
		svc := service.NewDriftService(driftRepo, assetRepo, imageRepo)

		orgID := uuid.New()

		// Setup: 100 running, 80 compliant = 80%
		assetRepo.CountAssetsByStateFunc = func(ctx context.Context, orgID uuid.UUID, state string) (int64, error) {
			if state == "running" {
				return 100, nil
			}
			return 0, nil
		}
		assetRepo.CountCompliantAssetsFunc = func(ctx context.Context, orgID uuid.UUID) (int64, error) {
			return 80, nil
		}

		result, err := svc.GetCurrentDrift(ctx, service.GetCurrentDriftInput{
			OrgID: orgID,
		})

		require.NoError(t, err)
		assert.Equal(t, 80.0, result.CoveragePct)
		assert.Equal(t, "warning", result.Status)
	})

	t.Run("critical status < 70%", func(t *testing.T) {
		assetRepo := mocks.NewMockAssetRepository()
		driftRepo := mocks.NewMockDriftRepository()
		imageRepo := mocks.NewMockImageRepository()
		svc := service.NewDriftService(driftRepo, assetRepo, imageRepo)

		orgID := uuid.New()

		// Setup: 100 running, 50 compliant = 50%
		assetRepo.CountAssetsByStateFunc = func(ctx context.Context, orgID uuid.UUID, state string) (int64, error) {
			if state == "running" {
				return 100, nil
			}
			return 0, nil
		}
		assetRepo.CountCompliantAssetsFunc = func(ctx context.Context, orgID uuid.UUID) (int64, error) {
			return 50, nil
		}

		result, err := svc.GetCurrentDrift(ctx, service.GetCurrentDriftInput{
			OrgID: orgID,
		})

		require.NoError(t, err)
		assert.Equal(t, 50.0, result.CoveragePct)
		assert.Equal(t, "critical", result.Status)
	})

	t.Run("zero assets returns 0% coverage", func(t *testing.T) {
		assetRepo := mocks.NewMockAssetRepository()
		driftRepo := mocks.NewMockDriftRepository()
		imageRepo := mocks.NewMockImageRepository()
		svc := service.NewDriftService(driftRepo, assetRepo, imageRepo)

		orgID := uuid.New()

		// Setup: 0 running assets
		assetRepo.CountAssetsByStateFunc = func(ctx context.Context, orgID uuid.UUID, state string) (int64, error) {
			return 0, nil
		}
		assetRepo.CountCompliantAssetsFunc = func(ctx context.Context, orgID uuid.UUID) (int64, error) {
			return 0, nil
		}

		result, err := svc.GetCurrentDrift(ctx, service.GetCurrentDriftInput{
			OrgID: orgID,
		})

		require.NoError(t, err)
		assert.Equal(t, int64(0), result.TotalAssets)
		assert.Equal(t, 0.0, result.CoveragePct)
	})
}

func TestDriftService_GetDriftSummary(t *testing.T) {
	ctx := context.Background()

	t.Run("returns summary with all dimensions", func(t *testing.T) {
		assetRepo := mocks.NewMockAssetRepository()
		driftRepo := mocks.NewMockDriftRepository()
		imageRepo := mocks.NewMockImageRepository()
		svc := service.NewDriftService(driftRepo, assetRepo, imageRepo)

		orgID := uuid.New()

		// Setup asset counts
		assetRepo.CountAssetsByStateFunc = func(ctx context.Context, orgID uuid.UUID, state string) (int64, error) {
			if state == "running" {
				return 100, nil
			}
			return 0, nil
		}
		assetRepo.CountCompliantAssetsFunc = func(ctx context.Context, orgID uuid.UUID) (int64, error) {
			return 85, nil
		}

		// Setup drift by dimensions
		driftRepo.GetDriftByEnvironmentFunc = func(ctx context.Context, orgID uuid.UUID) ([]service.DriftByScope, error) {
			return []service.DriftByScope{
				{Scope: "production", TotalAssets: 60, CompliantAssets: 55, CoveragePct: 91.67},
				{Scope: "staging", TotalAssets: 40, CompliantAssets: 30, CoveragePct: 75.0},
			}, nil
		}
		driftRepo.GetDriftByPlatformFunc = func(ctx context.Context, orgID uuid.UUID) ([]service.DriftByScope, error) {
			return []service.DriftByScope{
				{Scope: "aws", TotalAssets: 70, CompliantAssets: 65, CoveragePct: 92.86},
				{Scope: "azure", TotalAssets: 30, CompliantAssets: 20, CoveragePct: 66.67},
			}, nil
		}
		driftRepo.GetDriftBySiteFunc = func(ctx context.Context, orgID uuid.UUID) ([]service.DriftByScope, error) {
			return []service.DriftByScope{
				{Scope: "us-east-1", TotalAssets: 50, CompliantAssets: 45, CoveragePct: 90.0},
				{Scope: "eu-west-1", TotalAssets: 50, CompliantAssets: 40, CoveragePct: 80.0},
			}, nil
		}

		result, err := svc.GetDriftSummary(ctx, service.GetDriftSummaryInput{
			OrgID: orgID,
		})

		require.NoError(t, err)

		// Check overall
		assert.Equal(t, int64(100), result.Overall.TotalAssets)
		assert.Equal(t, int64(85), result.Overall.CompliantAssets)
		assert.Equal(t, 85.0, result.Overall.CoveragePct)

		// Check by environment
		assert.Len(t, result.ByEnvironment, 2)
		assert.Equal(t, "production", result.ByEnvironment[0].Scope)
		assert.Equal(t, "staging", result.ByEnvironment[1].Scope)

		// Check by platform
		assert.Len(t, result.ByPlatform, 2)
		assert.Equal(t, "aws", result.ByPlatform[0].Scope)

		// Check by site
		assert.Len(t, result.BySite, 2)
	})
}

func TestDriftService_GetDriftTrends(t *testing.T) {
	ctx := context.Background()

	t.Run("returns trends for specified days", func(t *testing.T) {
		assetRepo := mocks.NewMockAssetRepository()
		driftRepo := mocks.NewMockDriftRepository()
		imageRepo := mocks.NewMockImageRepository()
		svc := service.NewDriftService(driftRepo, assetRepo, imageRepo)

		orgID := uuid.New()

		trends, err := svc.GetDriftTrends(ctx, service.GetDriftTrendsInput{
			OrgID: orgID,
			Days:  14,
		})

		require.NoError(t, err)
		assert.Len(t, trends, 14)
	})

	t.Run("defaults to 30 days", func(t *testing.T) {
		assetRepo := mocks.NewMockAssetRepository()
		driftRepo := mocks.NewMockDriftRepository()
		imageRepo := mocks.NewMockImageRepository()
		svc := service.NewDriftService(driftRepo, assetRepo, imageRepo)

		orgID := uuid.New()

		trends, err := svc.GetDriftTrends(ctx, service.GetDriftTrendsInput{
			OrgID: orgID,
			Days:  0, // Should default to 30
		})

		require.NoError(t, err)
		assert.Len(t, trends, 30)
	})

	t.Run("caps at 365 days", func(t *testing.T) {
		assetRepo := mocks.NewMockAssetRepository()
		driftRepo := mocks.NewMockDriftRepository()
		imageRepo := mocks.NewMockImageRepository()
		svc := service.NewDriftService(driftRepo, assetRepo, imageRepo)

		orgID := uuid.New()

		trends, err := svc.GetDriftTrends(ctx, service.GetDriftTrendsInput{
			OrgID: orgID,
			Days:  500, // Should cap at 365
		})

		require.NoError(t, err)
		assert.Len(t, trends, 365)
	})
}

func TestDriftService_ListDriftReports(t *testing.T) {
	ctx := context.Background()
	assetRepo := mocks.NewMockAssetRepository()
	driftRepo := mocks.NewMockDriftRepository()
	imageRepo := mocks.NewMockImageRepository()
	svc := service.NewDriftService(driftRepo, assetRepo, imageRepo)

	orgID := uuid.New()

	// Setup: Add reports
	for i := 0; i < 25; i++ {
		driftRepo.AddReport(&service.DriftReport{
			ID:              uuid.New(),
			OrgID:           orgID,
			TotalAssets:     100,
			CompliantAssets: 85 + i%10,
			CoveragePct:     85.0 + float64(i%10),
			CalculatedAt:    time.Now().Add(-time.Duration(i) * time.Hour),
		})
	}

	t.Run("returns paginated results", func(t *testing.T) {
		result, err := svc.ListDriftReports(ctx, service.ListDriftReportsInput{
			OrgID:    orgID,
			Page:     1,
			PageSize: 10,
		})

		require.NoError(t, err)
		assert.Len(t, result.Reports, 10)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 10, result.PageSize)
	})

	t.Run("applies defaults", func(t *testing.T) {
		result, err := svc.ListDriftReports(ctx, service.ListDriftReportsInput{
			OrgID: orgID,
		})

		require.NoError(t, err)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PageSize)
	})

	t.Run("caps page size at 100", func(t *testing.T) {
		result, err := svc.ListDriftReports(ctx, service.ListDriftReportsInput{
			OrgID:    orgID,
			PageSize: 500, // Should be capped
		})

		require.NoError(t, err)
		assert.Equal(t, 20, result.PageSize) // Falls back to default since > 100
	})
}

func TestDriftService_CalculateDrift(t *testing.T) {
	ctx := context.Background()

	t.Run("calculates and stores drift report", func(t *testing.T) {
		assetRepo := mocks.NewMockAssetRepository()
		driftRepo := mocks.NewMockDriftRepository()
		imageRepo := mocks.NewMockImageRepository()
		svc := service.NewDriftService(driftRepo, assetRepo, imageRepo)

		orgID := uuid.New()

		// Setup
		assetRepo.CountAssetsByStateFunc = func(ctx context.Context, orgID uuid.UUID, state string) (int64, error) {
			if state == "running" {
				return 100, nil
			}
			return 0, nil
		}
		assetRepo.CountCompliantAssetsFunc = func(ctx context.Context, orgID uuid.UUID) (int64, error) {
			return 90, nil
		}

		result, err := svc.CalculateDrift(ctx, service.CalculateDriftInput{
			OrgID: orgID,
		})

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, orgID, result.OrgID)
		assert.Equal(t, 100, result.TotalAssets)
		assert.Equal(t, 90, result.CompliantAssets)
		assert.Equal(t, 90.0, result.CoveragePct)
	})

	t.Run("calculates drift for specific scope", func(t *testing.T) {
		assetRepo := mocks.NewMockAssetRepository()
		driftRepo := mocks.NewMockDriftRepository()
		imageRepo := mocks.NewMockImageRepository()
		svc := service.NewDriftService(driftRepo, assetRepo, imageRepo)

		orgID := uuid.New()
		envID := uuid.New()
		platform := "aws"

		// Setup
		assetRepo.CountAssetsByStateFunc = func(ctx context.Context, orgID uuid.UUID, state string) (int64, error) {
			return 50, nil
		}
		assetRepo.CountCompliantAssetsFunc = func(ctx context.Context, orgID uuid.UUID) (int64, error) {
			return 45, nil
		}

		var capturedParams service.CreateDriftReportParams
		driftRepo.CreateDriftReportFunc = func(ctx context.Context, params service.CreateDriftReportParams) (*service.DriftReport, error) {
			capturedParams = params
			return &service.DriftReport{
				ID:              uuid.New(),
				OrgID:           params.OrgID,
				EnvID:           params.EnvID,
				Platform:        params.Platform,
				TotalAssets:     params.TotalAssets,
				CompliantAssets: params.CompliantAssets,
				CoveragePct:     params.CoveragePct,
				CalculatedAt:    time.Now(),
			}, nil
		}

		result, err := svc.CalculateDrift(ctx, service.CalculateDriftInput{
			OrgID:    orgID,
			EnvID:    &envID,
			Platform: &platform,
		})

		require.NoError(t, err)
		assert.Equal(t, &envID, capturedParams.EnvID)
		assert.Equal(t, &platform, capturedParams.Platform)
		assert.NotNil(t, result)
	})
}
