package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
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

// withChiURLParams adds chi URL params to the request context.
func withChiURLParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestImageHandler_List(t *testing.T) {
	log := logger.New("debug", "text")
	repo := mocks.NewMockImageRepository()
	svc := service.NewImageService(repo)
	handler := handlers.NewImageHandler(svc, log)

	orgID := testOrg().ID

	// Setup: Add images
	for i := 0; i < 5; i++ {
		repo.AddImage(&service.Image{
			ID:        uuid.New(),
			OrgID:     orgID,
			Family:    "test-family",
			Version:   "1.0." + string(rune('0'+i)),
			Status:    "production",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}

	t.Run("returns paginated images", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/images?page=1&page_size=10", nil)
		req = withOrgContext(req)

		rr := executeRequest(handler.List, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response models.ImageListResponse
		require.NoError(t, decodeJSON(rr, &response))
		assert.Len(t, response.Images, 5)
		assert.Equal(t, 1, response.Page)
	})

	t.Run("unauthorized without org context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/images", nil)
		// No org context

		rr := executeRequest(handler.List, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestImageHandler_Get(t *testing.T) {
	log := logger.New("debug", "text")
	repo := mocks.NewMockImageRepository()
	svc := service.NewImageService(repo)
	handler := handlers.NewImageHandler(svc, log)

	orgID := testOrg().ID
	imageID := uuid.New()

	// Setup: Add an image
	repo.AddImage(&service.Image{
		ID:        imageID,
		OrgID:     orgID,
		Family:    "test-family",
		Version:   "1.0.0",
		Status:    "production",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	t.Run("returns image by ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/images/"+imageID.String(), nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": imageID.String()})

		rr := executeRequest(handler.Get, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response models.Image
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, imageID, response.ID)
		assert.Equal(t, "test-family", response.Family)
	})

	t.Run("returns 404 for non-existent image", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/images/"+nonExistentID.String(), nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": nonExistentID.String()})

		rr := executeRequest(handler.Get, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/images/not-a-uuid", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": "not-a-uuid"})

		rr := executeRequest(handler.Get, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestImageHandler_GetLatest(t *testing.T) {
	log := logger.New("debug", "text")
	repo := mocks.NewMockImageRepository()
	svc := service.NewImageService(repo)
	handler := handlers.NewImageHandler(svc, log)

	orgID := testOrg().ID

	// Setup: Add images with different versions
	repo.AddImage(&service.Image{
		ID:        uuid.New(),
		OrgID:     orgID,
		Family:    "test-family",
		Version:   "1.0.0",
		Status:    "deprecated",
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now(),
	})

	latestID := uuid.New()
	repo.AddImage(&service.Image{
		ID:        latestID,
		OrgID:     orgID,
		Family:    "test-family",
		Version:   "2.0.0",
		Status:    "production",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	t.Run("returns latest production image", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/images/test-family/latest", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"family": "test-family"})

		rr := executeRequest(handler.GetLatest, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response models.Image
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, latestID, response.ID)
		assert.Equal(t, "2.0.0", response.Version)
	})

	t.Run("returns 404 for non-existent family", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/images/non-existent/latest", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"family": "non-existent"})

		rr := executeRequest(handler.GetLatest, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func TestImageHandler_Create(t *testing.T) {
	log := logger.New("debug", "text")
	repo := mocks.NewMockImageRepository()
	svc := service.NewImageService(repo)
	handler := handlers.NewImageHandler(svc, log)

	t.Run("creates new image", func(t *testing.T) {
		body := models.CreateImageRequest{
			Family:    "new-family",
			Version:   "1.0.0",
			OSName:    "ubuntu",
			OSVersion: "22.04",
			CISLevel:  2,
			Signed:    true,
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/images", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)

		rr := executeRequest(handler.Create, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var response models.Image
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, "new-family", response.Family)
		assert.Equal(t, "1.0.0", response.Version)
		assert.Equal(t, models.ImageStatusDraft, response.Status)
		assert.True(t, response.Signed)
	})

	t.Run("returns 400 for missing family", func(t *testing.T) {
		body := models.CreateImageRequest{
			Version: "1.0.0",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/images", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)

		rr := executeRequest(handler.Create, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns 400 for missing version", func(t *testing.T) {
		body := models.CreateImageRequest{
			Family: "test-family",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/images", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)

		rr := executeRequest(handler.Create, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/images", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)

		rr := executeRequest(handler.Create, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestImageHandler_Promote(t *testing.T) {
	log := logger.New("debug", "text")
	repo := mocks.NewMockImageRepository()
	svc := service.NewImageService(repo)
	handler := handlers.NewImageHandler(svc, log)

	orgID := testOrg().ID
	imageID := uuid.New()

	// Setup: Add an image in draft status
	repo.AddImage(&service.Image{
		ID:        imageID,
		OrgID:     orgID,
		Family:    "test-family",
		Version:   "1.0.0",
		Status:    "draft",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	t.Run("promotes image to testing", func(t *testing.T) {
		body := map[string]string{"status": "testing"}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+imageID.String()+"/promote", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": imageID.String()})

		rr := executeRequest(handler.Promote, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response models.Image
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, models.ImageStatus("testing"), response.Status)
	})

	t.Run("returns 400 for invalid status", func(t *testing.T) {
		body := map[string]string{"status": "invalid"}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+imageID.String()+"/promote", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": imageID.String()})

		rr := executeRequest(handler.Promote, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns 404 for non-existent image", func(t *testing.T) {
		body := map[string]string{"status": "testing"}
		bodyBytes, _ := json.Marshal(body)

		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+nonExistentID.String()+"/promote", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": nonExistentID.String()})

		rr := executeRequest(handler.Promote, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func TestImageHandler_AddCoordinate(t *testing.T) {
	log := logger.New("debug", "text")
	repo := mocks.NewMockImageRepository()
	svc := service.NewImageService(repo)
	handler := handlers.NewImageHandler(svc, log)

	orgID := testOrg().ID
	imageID := uuid.New()

	// Setup: Add an image
	repo.AddImage(&service.Image{
		ID:        imageID,
		OrgID:     orgID,
		Family:    "test-family",
		Version:   "1.0.0",
		Status:    "production",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	t.Run("adds coordinate to image", func(t *testing.T) {
		body := models.AddCoordinateRequest{
			Platform:   models.PlatformAWS,
			Region:     "us-east-1",
			Identifier: "ami-12345678",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+imageID.String()+"/coordinates", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": imageID.String()})

		rr := executeRequest(handler.AddCoordinate, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var response models.ImageCoordinate
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, models.PlatformAWS, response.Platform)
		assert.Equal(t, "ami-12345678", response.Identifier)
	})

	t.Run("returns 400 for invalid platform", func(t *testing.T) {
		body := models.AddCoordinateRequest{
			Platform:   "invalid",
			Identifier: "test",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+imageID.String()+"/coordinates", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": imageID.String()})

		rr := executeRequest(handler.AddCoordinate, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns 400 for missing identifier", func(t *testing.T) {
		body := models.AddCoordinateRequest{
			Platform:   models.PlatformAWS,
			Identifier: "",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+imageID.String()+"/coordinates", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": imageID.String()})

		rr := executeRequest(handler.AddCoordinate, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
