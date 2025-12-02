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

// AlertHandler handles alert-related requests.
type AlertHandler struct {
	svc *service.AlertService
	log *logger.Logger
}

// NewAlertHandler creates a new AlertHandler.
func NewAlertHandler(svc *service.AlertService, log *logger.Logger) *AlertHandler {
	return &AlertHandler{
		svc: svc,
		log: log.WithComponent("alert-handler"),
	}
}

// List returns a paginated list of alerts.
func (h *AlertHandler) List(w http.ResponseWriter, r *http.Request) {
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
	input := service.ListAlertsInput{
		OrgID:    org.ID,
		Page:     page,
		PageSize: pageSize,
	}

	// Parse optional filters
	if severity := r.URL.Query().Get("severity"); severity != "" {
		input.Severity = &severity
	}
	if status := r.URL.Query().Get("status"); status != "" {
		input.Status = &status
	}
	if source := r.URL.Query().Get("source"); source != "" {
		input.Source = &source
	}
	if siteID := r.URL.Query().Get("site_id"); siteID != "" {
		if id, err := uuid.Parse(siteID); err == nil {
			input.SiteID = &id
		}
	}

	// Call service
	result, err := h.svc.ListAlerts(ctx, input)
	if err != nil {
		h.log.Error("failed to list alerts", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	response := models.AlertListResponse{
		Alerts:     serviceAlertsToModel(result.Alerts),
		Total:      int(result.Total),
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	}

	writeJSON(w, http.StatusOK, response)
}

// Get returns a single alert by ID.
func (h *AlertHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	alertID := chi.URLParam(r, "id")
	id, err := uuid.Parse(alertID)
	if err != nil {
		http.Error(w, "invalid alert ID", http.StatusBadRequest)
		return
	}

	// Call service
	alert, err := h.svc.GetAlert(ctx, service.GetAlertInput{
		ID:    id,
		OrgID: org.ID,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.Error(w, "alert not found", http.StatusNotFound)
			return
		}
		h.log.Error("failed to get alert", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, serviceAlertToModel(*alert))
}

// Acknowledge marks an alert as acknowledged.
func (h *AlertHandler) Acknowledge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	user := middleware.GetUser(ctx)
	if user == nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	alertID := chi.URLParam(r, "id")
	id, err := uuid.Parse(alertID)
	if err != nil {
		http.Error(w, "invalid alert ID", http.StatusBadRequest)
		return
	}

	// Call service
	err = h.svc.AcknowledgeAlert(ctx, service.AcknowledgeAlertInput{
		ID:     id,
		OrgID:  org.ID,
		UserID: user.ID,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.Error(w, "alert not found", http.StatusNotFound)
			return
		}
		h.log.Error("failed to acknowledge alert", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
}

// Resolve marks an alert as resolved.
func (h *AlertHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	user := middleware.GetUser(ctx)
	if user == nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	alertID := chi.URLParam(r, "id")
	id, err := uuid.Parse(alertID)
	if err != nil {
		http.Error(w, "invalid alert ID", http.StatusBadRequest)
		return
	}

	// Call service
	err = h.svc.ResolveAlert(ctx, service.ResolveAlertInput{
		ID:     id,
		OrgID:  org.ID,
		UserID: user.ID,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.Error(w, "alert not found", http.StatusNotFound)
			return
		}
		h.log.Error("failed to resolve alert", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
}

// Summary returns aggregated alert statistics.
func (h *AlertHandler) Summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	// Call service
	summary, err := h.svc.GetAlertSummary(ctx, service.GetAlertSummaryInput{
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to get alert summary", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

// Helper functions to convert between service and model types
func serviceAlertToModel(alert service.Alert) models.Alert {
	return models.Alert{
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

func serviceAlertsToModel(alerts []service.Alert) []models.Alert {
	result := make([]models.Alert, 0, len(alerts))
	for _, a := range alerts {
		result = append(result, serviceAlertToModel(a))
	}
	return result
}
