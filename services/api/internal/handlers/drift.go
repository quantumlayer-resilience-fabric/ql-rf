package handlers

import (
	"fmt"
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
// Response format matches frontend DriftSummary type.
func (h *DriftHandler) Summary(w http.ResponseWriter, r *http.Request) {
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

	// Calculate drifted assets and critical drift
	driftedAssets := summary.Overall.TotalAssets - summary.Overall.CompliantAssets
	criticalDrift := int64(0)
	for _, env := range summary.ByEnvironment {
		if env.Status == "critical" {
			criticalDrift += env.TotalAssets - env.CompliantAssets
		}
	}

	// Calculate drift percentage (inverse of coverage)
	driftPercentage := 0.0
	if summary.Overall.TotalAssets > 0 {
		driftPercentage = float64(driftedAssets) / float64(summary.Overall.TotalAssets) * 100
	}

	// Build byEnvironment array matching frontend type
	byEnvironment := make([]map[string]interface{}, 0, len(summary.ByEnvironment))
	for _, env := range summary.ByEnvironment {
		byEnvironment = append(byEnvironment, map[string]interface{}{
			"environment": env.Scope,
			"compliant":   env.CompliantAssets,
			"total":       env.TotalAssets,
			"percentage":  env.CoveragePct,
		})
	}

	// Build bySite array matching frontend type
	bySite := make([]map[string]interface{}, 0, len(summary.BySite))
	for _, site := range summary.BySite {
		bySite = append(bySite, map[string]interface{}{
			"siteId":   site.Scope,
			"siteName": site.Scope,
			"coverage": site.CoveragePct,
			"status":   site.Status,
		})
	}

	// Get drift age distribution from database
	ageDistribution, err := h.svc.GetDriftAgeDistribution(ctx, service.GetDriftAgeDistributionInput{
		OrgID: org.ID,
	})

	// Build byAge array from actual drift age distribution
	byAge := make([]map[string]interface{}, 0, 4)
	averageDriftAge := "0 days"
	if err == nil && ageDistribution != nil {
		for _, r := range ageDistribution.ByRange {
			byAge = append(byAge, map[string]interface{}{
				"range":      r.Range,
				"count":      r.Count,
				"percentage": r.Percentage,
			})
		}
		// Format average drift age
		avgDays := int(ageDistribution.AverageDays)
		if avgDays == 1 {
			averageDriftAge = "1 day"
		} else {
			averageDriftAge = fmt.Sprintf("%d days", avgDays)
		}
	} else {
		// Fallback if query fails
		byAge = []map[string]interface{}{
			{"range": "0-7 days", "count": 0, "percentage": 0.0},
			{"range": "8-14 days", "count": 0, "percentage": 0.0},
			{"range": "15-30 days", "count": 0, "percentage": 0.0},
			{"range": "30+ days", "count": 0, "percentage": 0.0},
		}
	}

	// Return response matching frontend DriftSummary type
	response := struct {
		TotalAssets     int64                    `json:"totalAssets"`
		CompliantAssets int64                    `json:"compliantAssets"`
		DriftedAssets   int64                    `json:"driftedAssets"`
		DriftPercentage float64                  `json:"driftPercentage"`
		CriticalDrift   int64                    `json:"criticalDrift"`
		AverageDriftAge string                   `json:"averageDriftAge"`
		ByEnvironment   []map[string]interface{} `json:"byEnvironment"`
		BySite          []map[string]interface{} `json:"bySite"`
		ByAge           []map[string]interface{} `json:"byAge"`
	}{
		TotalAssets:     summary.Overall.TotalAssets,
		CompliantAssets: summary.Overall.CompliantAssets,
		DriftedAssets:   driftedAssets,
		DriftPercentage: driftPercentage,
		CriticalDrift:   criticalDrift,
		AverageDriftAge: averageDriftAge,
		ByEnvironment:   byEnvironment,
		BySite:          bySite,
		ByAge:           byAge,
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
	id, err := uuid.Parse(reportID)
	if err != nil {
		http.Error(w, "invalid report ID", http.StatusBadRequest)
		return
	}

	// Get report from service
	report, err := h.svc.GetDriftReport(ctx, service.GetDriftReportInput{
		ID:    id,
		OrgID: org.ID,
	})
	if err != nil {
		if err == service.ErrNotFound {
			http.Error(w, "report not found", http.StatusNotFound)
			return
		}
		h.log.Error("failed to get drift report", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, serviceDriftReportToModel(*report))
}

// TopOffenders returns the assets most out of compliance (drifted from golden image).
func (h *DriftHandler) TopOffenders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	// Parse limit parameter
	limit := int32(10)
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = int32(parsed)
		}
	}

	// Get top offenders from service
	assets, err := h.svc.GetTopOffenders(ctx, service.GetTopOffendersInput{
		OrgID: org.ID,
		Limit: limit,
	})
	if err != nil {
		h.log.Error("failed to get top offenders", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to response format matching frontend Asset type
	response := make([]map[string]interface{}, 0, len(assets))
	for _, a := range assets {
		asset := map[string]interface{}{
			"id":                  a.ID.String(),
			"hostname":            getStrOrDefault(a.Name, a.InstanceID),
			"siteId":              getStrOrDefault(a.Site, ""),
			"siteName":            getStrOrDefault(a.Site, "Unknown"),
			"platform":            a.Platform,
			"environment":         "production", // Default, could be derived from EnvID
			"currentImageId":      getStrOrDefault(a.ImageRef, ""),
			"currentImageVersion": getStrOrDefault(a.ImageVersion, ""),
			"goldenImageId":       "", // Would need to look up latest production image
			"goldenImageVersion":  "", // Would need to look up latest production image
			"isDrifted":           true,
			"lastScannedAt":       a.UpdatedAt.Format(time.RFC3339),
			"metadata":            map[string]string{},
			"createdAt":           a.DiscoveredAt.Format(time.RFC3339),
			"updatedAt":           a.UpdatedAt.Format(time.RFC3339),
		}
		response = append(response, asset)
	}

	writeJSON(w, http.StatusOK, response)
}

// getStrOrDefault returns the string value or a default if nil.
func getStrOrDefault(s *string, def string) string {
	if s != nil {
		return *s
	}
	return def
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
