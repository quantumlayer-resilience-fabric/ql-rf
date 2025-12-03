package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
)

// PredictionHandler handles risk prediction requests.
type PredictionHandler struct {
	svc *service.PredictionService
	log *logger.Logger
}

// NewPredictionHandler creates a new PredictionHandler.
func NewPredictionHandler(svc *service.PredictionService, log *logger.Logger) *PredictionHandler {
	return &PredictionHandler{
		svc: svc,
		log: log.WithComponent("prediction-handler"),
	}
}

// GetForecast returns organization-wide risk forecast.
func (h *PredictionHandler) GetForecast(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	forecast, err := h.svc.GetRiskForecast(ctx, service.GetRiskForecastInput{
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to get risk forecast", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, forecast)
}

// GetAssetPrediction returns risk prediction for a specific asset.
func (h *PredictionHandler) GetAssetPrediction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	assetIDStr := chi.URLParam(r, "id")
	assetID, err := uuid.Parse(assetIDStr)
	if err != nil {
		http.Error(w, "invalid asset id", http.StatusBadRequest)
		return
	}

	prediction, err := h.svc.GetAssetPrediction(ctx, assetID)
	if err != nil {
		h.log.Error("failed to get asset prediction", "error", err, "asset_id", assetID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, prediction)
}

// GetRecommendations returns top risk mitigation recommendations.
func (h *PredictionHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	forecast, err := h.svc.GetRiskForecast(ctx, service.GetRiskForecastInput{
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to get recommendations", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, forecast.TopRecommendations)
}

// GetAnomalies returns detected risk anomalies.
func (h *PredictionHandler) GetAnomalies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	forecast, err := h.svc.GetRiskForecast(ctx, service.GetRiskForecastInput{
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to get anomalies", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, forecast.Anomalies)
}
