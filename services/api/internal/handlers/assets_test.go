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

func TestAssetHandler_List(t *testing.T) {
	log := logger.New("debug", "text")
	repo := mocks.NewMockAssetRepository()
	svc := service.NewAssetService(repo)
	handler := handlers.NewAssetHandler(svc, log)

	orgID := testOrg().ID

	// Setup: Add assets
	for i := 0; i < 5; i++ {
		repo.AddAsset(&service.Asset{
			ID:           uuid.New(),
			OrgID:        orgID,
			Platform:     "aws",
			InstanceID:   "i-" + uuid.New().String()[:8],
			State:        "running",
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
		})
	}

	t.Run("returns paginated assets", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/assets?page=1&page_size=10", nil)
		req = withOrgContext(req)

		rr := executeRequest(handler.List, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response models.AssetListResponse
		require.NoError(t, decodeJSON(rr, &response))
		assert.Len(t, response.Assets, 5)
		assert.Equal(t, 1, response.Page)
	})

	t.Run("filters by platform", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/assets?platform=aws", nil)
		req = withOrgContext(req)

		rr := executeRequest(handler.List, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response models.AssetListResponse
		require.NoError(t, decodeJSON(rr, &response))
		assert.Len(t, response.Assets, 5)
	})

	t.Run("unauthorized without org context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/assets", nil)
		// No org context

		rr := executeRequest(handler.List, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestAssetHandler_Get(t *testing.T) {
	log := logger.New("debug", "text")
	repo := mocks.NewMockAssetRepository()
	svc := service.NewAssetService(repo)
	handler := handlers.NewAssetHandler(svc, log)

	orgID := testOrg().ID
	assetID := uuid.New()
	region := "us-east-1"

	// Setup: Add an asset
	repo.AddAsset(&service.Asset{
		ID:           assetID,
		OrgID:        orgID,
		Platform:     "aws",
		Region:       &region,
		InstanceID:   "i-12345678",
		State:        "running",
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	})

	t.Run("returns asset by ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/"+assetID.String(), nil)
		req = withOrgContext(req)

		// Add chi URL params
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", assetID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := executeRequest(handler.Get, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response models.Asset
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, assetID, response.ID)
		assert.Equal(t, models.PlatformAWS, response.Platform)
		assert.Equal(t, "i-12345678", response.InstanceID)
	})

	t.Run("returns 404 for non-existent asset", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/"+nonExistentID.String(), nil)
		req = withOrgContext(req)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", nonExistentID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := executeRequest(handler.Get, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/not-a-uuid", nil)
		req = withOrgContext(req)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "not-a-uuid")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := executeRequest(handler.Get, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestAssetHandler_Summary(t *testing.T) {
	log := logger.New("debug", "text")
	repo := mocks.NewMockAssetRepository()
	svc := service.NewAssetService(repo)
	handler := handlers.NewAssetHandler(svc, log)

	orgID := testOrg().ID
	imageRef := "ami-12345678"
	imageVersion := "1.0.0"

	// Setup: Add assets with different states
	repo.AddAsset(&service.Asset{
		ID:           uuid.New(),
		OrgID:        orgID,
		Platform:     "aws",
		InstanceID:   "i-compliant1",
		State:        "running",
		ImageRef:     &imageRef,
		ImageVersion: &imageVersion,
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	})
	repo.AddAsset(&service.Asset{
		ID:           uuid.New(),
		OrgID:        orgID,
		Platform:     "aws",
		InstanceID:   "i-noncompliant1",
		State:        "running",
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	})
	repo.AddAsset(&service.Asset{
		ID:           uuid.New(),
		OrgID:        orgID,
		Platform:     "aws",
		InstanceID:   "i-stopped1",
		State:        "stopped",
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	})

	t.Run("returns asset summary", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/summary", nil)
		req = withOrgContext(req)

		rr := executeRequest(handler.Summary, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response models.AssetSummary
		require.NoError(t, decodeJSON(rr, &response))
		// Summary returns aggregated counts from the service
	})

	t.Run("unauthorized without org context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/summary", nil)
		// No org context

		rr := executeRequest(handler.Summary, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}
