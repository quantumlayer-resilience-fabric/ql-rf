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

// SiteHandler handles site-related requests.
type SiteHandler struct {
	svc *service.SiteService
	log *logger.Logger
}

// NewSiteHandler creates a new SiteHandler.
func NewSiteHandler(svc *service.SiteService, log *logger.Logger) *SiteHandler {
	return &SiteHandler{
		svc: svc,
		log: log.WithComponent("site-handler"),
	}
}

// List returns a paginated list of sites.
func (h *SiteHandler) List(w http.ResponseWriter, r *http.Request) {
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
	input := service.ListSitesInput{
		OrgID:    org.ID,
		Page:     page,
		PageSize: pageSize,
	}

	// Parse optional filters
	if platform := r.URL.Query().Get("platform"); platform != "" {
		input.Platform = &platform
	}
	if region := r.URL.Query().Get("region"); region != "" {
		input.Region = &region
	}

	// Call service
	result, err := h.svc.ListSites(ctx, input)
	if err != nil {
		h.log.Error("failed to list sites", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	response := models.SiteListResponse{
		Sites:      serviceSitesToModel(result.Sites),
		Total:      int(result.Total),
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	}

	writeJSON(w, http.StatusOK, response)
}

// Get returns a single site by ID.
func (h *SiteHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	siteID := chi.URLParam(r, "id")
	id, err := uuid.Parse(siteID)
	if err != nil {
		http.Error(w, "invalid site ID", http.StatusBadRequest)
		return
	}

	// Call service
	site, err := h.svc.GetSite(ctx, service.GetSiteInput{
		ID:    id,
		OrgID: org.ID,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.Error(w, "site not found", http.StatusNotFound)
			return
		}
		h.log.Error("failed to get site", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, serviceSiteToModel(*site))
}

// Summary returns aggregated site statistics.
func (h *SiteHandler) Summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	// Call service
	summary, err := h.svc.GetSiteSummary(ctx, service.GetSiteSummaryInput{
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to get site summary", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

// Helper functions to convert between service and model types
func serviceSiteToModel(site service.SiteWithStats) models.Site {
	result := models.Site{
		ID:                 site.ID,
		OrgID:              site.OrgID,
		Name:               site.Name,
		Region:             site.Region,
		Platform:           models.Platform(site.Platform),
		Environment:        site.Environment,
		AssetCount:         site.AssetCount,
		CompliantCount:     site.CompliantCount,
		DriftedCount:       site.DriftedCount,
		CoveragePercentage: site.CoveragePercentage,
		Status:             site.Status,
		DRPaired:           site.DRPaired,
		CreatedAt:          site.CreatedAt,
		UpdatedAt:          site.UpdatedAt,
	}
	if site.LastSyncAt != nil {
		result.LastSyncAt = site.LastSyncAt
	}
	if site.DRPairedSiteID != nil {
		result.DRPairedSiteID = site.DRPairedSiteID
	}
	return result
}

func serviceSitesToModel(sites []service.SiteWithStats) []models.Site {
	result := make([]models.Site, 0, len(sites))
	for _, s := range sites {
		result = append(result, serviceSiteToModel(s))
	}
	return result
}
