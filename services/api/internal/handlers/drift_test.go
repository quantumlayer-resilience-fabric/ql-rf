package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/handlers"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/service/mocks"
)

func TestDriftHandler_GetCurrent(t *testing.T) {
	log := logger.New("debug", "text")
	driftRepo := mocks.NewMockDriftRepository()
	assetRepo := mocks.NewMockAssetRepository()
	imageRepo := mocks.NewMockImageRepository()
	svc := service.NewDriftService(driftRepo, assetRepo, imageRepo)
	handler := handlers.NewDriftHandler(svc, log)

	orgID := testOrg().ID

	// Setup: Add a drift report
	driftRepo.AddReport(&service.DriftReport{
		ID:              uuid.New(),
		OrgID:           orgID,
		TotalAssets:     100,
		CompliantAssets: 95,
		CoveragePct:     95.0,
		Status:          "healthy",
		CalculatedAt:    time.Now(),
	})

	// Setup: Add scopes for summary
	driftRepo.AddEnvironmentScope(&service.DriftByScope{
		Scope:           "production",
		TotalAssets:     50,
		CompliantAssets: 48,
		CoveragePct:     96.0,
		Status:          "healthy",
	})
	driftRepo.AddPlatformScope(&service.DriftByScope{
		Scope:           "aws",
		TotalAssets:     60,
		CompliantAssets: 57,
		CoveragePct:     95.0,
		Status:          "healthy",
	})

	t.Run("returns current drift status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/drift", nil)
		req = withOrgContext(req)

		rr := executeRequest(handler.GetCurrent, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response models.DriftSummary
		require.NoError(t, decodeJSON(rr, &response))
		// Response contains drift summary from service
		assert.NotEmpty(t, response.Status)
	})

	t.Run("unauthorized without org context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/drift", nil)
		// No org context

		rr := executeRequest(handler.GetCurrent, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestDriftHandler_Summary(t *testing.T) {
	log := logger.New("debug", "text")
	driftRepo := mocks.NewMockDriftRepository()
	assetRepo := mocks.NewMockAssetRepository()
	imageRepo := mocks.NewMockImageRepository()
	svc := service.NewDriftService(driftRepo, assetRepo, imageRepo)
	handler := handlers.NewDriftHandler(svc, log)

	// Setup: Add a drift report for the service to find
	orgID := testOrg().ID
	driftRepo.AddReport(&service.DriftReport{
		ID:              uuid.New(),
		OrgID:           orgID,
		TotalAssets:     100,
		CompliantAssets: 90,
		CoveragePct:     90.0,
		Status:          "healthy",
		CalculatedAt:    time.Now(),
	})

	t.Run("returns drift summary", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/drift/summary", nil)
		req = withOrgContext(req)

		rr := executeRequest(handler.Summary, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		// Summary returns a simplified response
		var response struct {
			FleetSize       int64     `json:"fleet_size"`
			CoveragePct     float64   `json:"coverage_pct"`
			Status          string    `json:"status"`
			LastCalculation time.Time `json:"last_calculation"`
		}
		require.NoError(t, decodeJSON(rr, &response))
		// Response contains drift summary fields
		assert.NotEmpty(t, response.Status)
	})

	t.Run("unauthorized without org context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/drift/summary", nil)
		// No org context

		rr := executeRequest(handler.Summary, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestDriftHandler_Trends(t *testing.T) {
	log := logger.New("debug", "text")
	driftRepo := mocks.NewMockDriftRepository()
	assetRepo := mocks.NewMockAssetRepository()
	imageRepo := mocks.NewMockImageRepository()
	svc := service.NewDriftService(driftRepo, assetRepo, imageRepo)
	handler := handlers.NewDriftHandler(svc, log)

	orgID := testOrg().ID

	// Setup: Add trend data
	now := time.Now()
	for i := 0; i < 7; i++ {
		driftRepo.AddTrendPoint(&service.DriftTrendPoint{
			Date:            now.AddDate(0, 0, -i),
			AvgCoverage:     90.0 + float64(i),
			TotalAssets:     int64(100 + i),
			CompliantAssets: int64(90 + i),
		}, orgID)
	}

	t.Run("returns drift trends", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/drift/trends?days=7", nil)
		req = withOrgContext(req)

		rr := executeRequest(handler.Trends, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response []models.DriftTrend
		require.NoError(t, decodeJSON(rr, &response))
		assert.Len(t, response, 7)
	})

	t.Run("defaults to 30 days", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/drift/trends", nil)
		req = withOrgContext(req)

		rr := executeRequest(handler.Trends, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("unauthorized without org context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/drift/trends", nil)
		// No org context

		rr := executeRequest(handler.Trends, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestDriftHandler_ListReports(t *testing.T) {
	log := logger.New("debug", "text")
	driftRepo := mocks.NewMockDriftRepository()
	assetRepo := mocks.NewMockAssetRepository()
	imageRepo := mocks.NewMockImageRepository()
	svc := service.NewDriftService(driftRepo, assetRepo, imageRepo)
	handler := handlers.NewDriftHandler(svc, log)

	orgID := testOrg().ID

	// Setup: Add reports
	for i := 0; i < 5; i++ {
		driftRepo.AddReport(&service.DriftReport{
			ID:              uuid.New(),
			OrgID:           orgID,
			TotalAssets:     100 + i,
			CompliantAssets: 90 + i,
			CoveragePct:     90.0 + float64(i),
			Status:          "healthy",
			CalculatedAt:    time.Now().Add(-time.Duration(i) * time.Hour),
		})
	}

	t.Run("returns paginated reports", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/drift/reports?page=1&page_size=10", nil)
		req = withOrgContext(req)

		rr := executeRequest(handler.ListReports, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response struct {
			Reports    []models.DriftReport `json:"reports"`
			Page       int                  `json:"page"`
			PageSize   int                  `json:"page_size"`
			TotalPages int                  `json:"total_pages"`
		}
		require.NoError(t, decodeJSON(rr, &response))
		assert.Len(t, response.Reports, 5)
	})

	t.Run("unauthorized without org context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/drift/reports", nil)
		// No org context

		rr := executeRequest(handler.ListReports, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestDriftHandler_GetReport(t *testing.T) {
	log := logger.New("debug", "text")
	driftRepo := mocks.NewMockDriftRepository()
	assetRepo := mocks.NewMockAssetRepository()
	imageRepo := mocks.NewMockImageRepository()
	svc := service.NewDriftService(driftRepo, assetRepo, imageRepo)
	handler := handlers.NewDriftHandler(svc, log)

	orgID := testOrg().ID
	reportID := uuid.New()

	// Setup: Add a report
	driftRepo.AddReport(&service.DriftReport{
		ID:              reportID,
		OrgID:           orgID,
		TotalAssets:     100,
		CompliantAssets: 95,
		CoveragePct:     95.0,
		Status:          "healthy",
		CalculatedAt:    time.Now(),
	})

	t.Run("returns 404 for get report (not implemented)", func(t *testing.T) {
		// NOTE: GetReport is marked as TODO in the handler and always returns 404
		req := httptest.NewRequest(http.MethodGet, "/api/v1/drift/reports/"+reportID.String(), nil)
		req = withOrgContext(req)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", reportID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := executeRequest(handler.GetReport, req)

		// Currently returns 404 as GetReport is not implemented
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns 404 for non-existent report", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/drift/reports/"+nonExistentID.String(), nil)
		req = withOrgContext(req)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", nonExistentID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := executeRequest(handler.GetReport, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/drift/reports/not-a-uuid", nil)
		req = withOrgContext(req)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "not-a-uuid")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := executeRequest(handler.GetReport, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
