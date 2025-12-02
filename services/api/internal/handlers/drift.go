package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
)

// DriftHandler handles drift-related requests.
type DriftHandler struct {
	svc *service.DriftService
	log *logger.Logger
}

// NewDriftHandler creates a new DriftHandler.
func NewDriftHandler(svc *service.DriftService, log *logger.Logger) *DriftHandler {
	return &DriftHandler{
		svc: svc,
		log: log.WithComponent("drift-handler"),
	}
}

// GetCurrent returns the current drift analysis.
func (h *DriftHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	// Get drift summary from service
	summary, err := h.svc.GetDriftSummary(ctx, service.GetDriftSummaryInput{
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to get drift summary", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	response := models.DriftSummary{
		OrgID:           org.ID,
		TotalAssets:     int(summary.Overall.TotalAssets),
		CompliantAssets: int(summary.Overall.CompliantAssets),
		CoveragePct:     summary.Overall.CoveragePct,
		Status:          statusToModel(summary.Overall.Status),
		ByEnvironment:   serviceDriftByScopesToModel(summary.ByEnvironment),
		ByPlatform:      serviceDriftByScopesToModel(summary.ByPlatform),
		BySite:          serviceDriftByScopesToModel(summary.BySite),
		CalculatedAt:    time.Now(),
	}

	writeJSON(w, http.StatusOK, response)
}

// Summary returns a high-level drift summary for dashboards.
func (h *DriftHandler) Summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	// Get current drift from service
	drift, err := h.svc.GetCurrentDrift(ctx, service.GetCurrentDriftInput{
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to get current drift", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Return a simplified summary for dashboard widgets
	response := struct {
		FleetSize       int64     `json:"fleet_size"`
		CoveragePct     float64   `json:"coverage_pct"`
		Status          string    `json:"status"`
		LastCalculation time.Time `json:"last_calculation"`
	}{
		FleetSize:       drift.TotalAssets,
		CoveragePct:     drift.CoveragePct,
		Status:          drift.Status,
		LastCalculation: time.Now(),
	}

	writeJSON(w, http.StatusOK, response)
}

// Trends returns historical drift data for charting.
func (h *DriftHandler) Trends(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	// Parse date range
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days < 1 || days > 365 {
		days = 30
	}

	// Get trends from service
	trends, err := h.svc.GetDriftTrends(ctx, service.GetDriftTrendsInput{
		OrgID: org.ID,
		Days:  days,
	})
	if err != nil {
		h.log.Error("failed to get drift trends", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	response := make([]models.DriftTrend, 0, len(trends))
	for _, t := range trends {
		status := models.DriftStatusHealthy
		if t.AvgCoverage < 70 {
			status = models.DriftStatusCritical
		} else if t.AvgCoverage < 90 {
			status = models.DriftStatusWarning
		}

		response = append(response, models.DriftTrend{
			Date:        t.Date,
			CoveragePct: t.AvgCoverage,
			Status:      status,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

// ListReports returns a paginated list of drift reports.
func (h *DriftHandler) ListReports(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	// Parse pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Get reports from service
	result, err := h.svc.ListDriftReports(ctx, service.ListDriftReportsInput{
		OrgID:    org.ID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.log.Error("failed to list drift reports", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	reports := make([]models.DriftReport, 0, len(result.Reports))
	for _, r := range result.Reports {
		reports = append(reports, serviceDriftReportToModel(r))
	}

	response := struct {
		Reports    []models.DriftReport `json:"reports"`
		Page       int                  `json:"page"`
		PageSize   int                  `json:"page_size"`
		TotalPages int                  `json:"total_pages"`
	}{
		Reports:    reports,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	}

	writeJSON(w, http.StatusOK, response)
}

// GetReport returns a single drift report by ID.
func (h *DriftHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	reportID := chi.URLParam(r, "id")
	_, err := uuid.Parse(reportID)
	if err != nil {
		http.Error(w, "invalid report ID", http.StatusBadRequest)
		return
	}

	// TODO: Implement GetReport in service layer
	http.Error(w, "report not found", http.StatusNotFound)
}

// Helper functions to convert between service and model types
func statusToModel(status string) models.DriftStatus {
	switch status {
	case "healthy":
		return models.DriftStatusHealthy
	case "warning":
		return models.DriftStatusWarning
	case "critical":
		return models.DriftStatusCritical
	default:
		return models.DriftStatusHealthy
	}
}

func serviceDriftByScopesToModel(scopes []service.DriftByScope) []models.DriftByScope {
	result := make([]models.DriftByScope, 0, len(scopes))
	for _, s := range scopes {
		result = append(result, models.DriftByScope{
			Scope:           s.Scope,
			TotalAssets:     int(s.TotalAssets),
			CompliantAssets: int(s.CompliantAssets),
			CoveragePct:     s.CoveragePct,
			Status:          statusToModel(s.Status),
		})
	}
	return result
}

func serviceDriftReportToModel(report service.DriftReport) models.DriftReport {
	result := models.DriftReport{
		ID:              report.ID,
		OrgID:           report.OrgID,
		TotalAssets:     report.TotalAssets,
		CompliantAssets: report.CompliantAssets,
		CoveragePct:     report.CoveragePct,
		Status:          statusToModel(report.Status),
		CalculatedAt:    report.CalculatedAt,
	}
	if report.EnvID != nil {
		result.EnvID = *report.EnvID
	}
	if report.Platform != nil {
		result.Platform = models.Platform(*report.Platform)
	}
	if report.Site != nil {
		result.Site = *report.Site
	}
	return result
}
