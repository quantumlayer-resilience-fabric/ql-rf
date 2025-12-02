package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
)

// AssetHandler handles asset-related requests.
type AssetHandler struct {
	svc *service.AssetService
	log *logger.Logger
}

// NewAssetHandler creates a new AssetHandler.
func NewAssetHandler(svc *service.AssetService, log *logger.Logger) *AssetHandler {
	return &AssetHandler{
		svc: svc,
		log: log.WithComponent("asset-handler"),
	}
}

// List returns a paginated list of assets.
func (h *AssetHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Build input
	input := service.ListAssetsInput{
		OrgID:    org.ID,
		Page:     page,
		PageSize: pageSize,
	}

	// Parse optional filters
	if platform := r.URL.Query().Get("platform"); platform != "" {
		input.Platform = &platform
	}
	if state := r.URL.Query().Get("state"); state != "" {
		input.State = &state
	}
	if envID := r.URL.Query().Get("env_id"); envID != "" {
		if id, err := uuid.Parse(envID); err == nil {
			input.EnvID = &id
		}
	}

	// Call service
	result, err := h.svc.ListAssets(ctx, input)
	if err != nil {
		h.log.Error("failed to list assets", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	response := models.AssetListResponse{
		Assets:     serviceAssetsToModel(result.Assets),
		Total:      int(result.Total),
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	}

	writeJSON(w, http.StatusOK, response)
}

// Get returns a single asset by ID.
func (h *AssetHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	assetID := chi.URLParam(r, "id")
	id, err := uuid.Parse(assetID)
	if err != nil {
		http.Error(w, "invalid asset ID", http.StatusBadRequest)
		return
	}

	// Call service
	asset, err := h.svc.GetAsset(ctx, service.GetAssetInput{
		ID:    id,
		OrgID: org.ID,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.Error(w, "asset not found", http.StatusNotFound)
			return
		}
		h.log.Error("failed to get asset", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, serviceAssetToModel(*asset))
}

// Summary returns aggregated asset statistics.
func (h *AssetHandler) Summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	// Call service
	summary, err := h.svc.GetAssetSummary(ctx, service.GetAssetSummaryInput{
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to get asset summary", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	response := struct {
		TotalAssets   int64            `json:"total_assets"`
		RunningAssets int64            `json:"running_assets"`
		StoppedAssets int64            `json:"stopped_assets"`
		ByPlatform    map[string]int64 `json:"by_platform"`
		ByState       map[string]int64 `json:"by_state"`
	}{
		TotalAssets:   summary.TotalAssets,
		RunningAssets: summary.RunningAssets,
		StoppedAssets: summary.StoppedAssets,
		ByPlatform:    summary.ByPlatform,
		ByState:       summary.ByState,
	}

	writeJSON(w, http.StatusOK, response)
}

// Helper functions to convert between service and model types
func serviceAssetToModel(asset service.Asset) models.Asset {
	result := models.Asset{
		ID:         asset.ID,
		OrgID:      asset.OrgID,
		Platform:   models.Platform(asset.Platform),
		InstanceID: asset.InstanceID,
		State:      models.AssetState(asset.State),
	}
	if asset.EnvID != nil {
		result.EnvID = *asset.EnvID
	}
	if asset.Account != nil {
		result.Account = *asset.Account
	}
	if asset.Region != nil {
		result.Region = *asset.Region
	}
	if asset.Site != nil {
		result.Site = *asset.Site
	}
	if asset.Name != nil {
		result.Name = *asset.Name
	}
	if asset.ImageRef != nil {
		result.ImageRef = *asset.ImageRef
	}
	if asset.ImageVersion != nil {
		result.ImageVersion = *asset.ImageVersion
	}
	return result
}

func serviceAssetsToModel(assets []service.Asset) []models.Asset {
	result := make([]models.Asset, 0, len(assets))
	for _, a := range assets {
		result = append(result, serviceAssetToModel(a))
	}
	return result
}
