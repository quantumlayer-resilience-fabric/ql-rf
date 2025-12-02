package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
)

// ResilienceHandler handles resilience/DR-related requests.
type ResilienceHandler struct {
	svc *service.ResilienceService
	log *logger.Logger
}

// NewResilienceHandler creates a new ResilienceHandler.
func NewResilienceHandler(svc *service.ResilienceService, log *logger.Logger) *ResilienceHandler {
	return &ResilienceHandler{
		svc: svc,
		log: log.WithComponent("resilience-handler"),
	}
}

// Summary returns resilience summary.
func (h *ResilienceHandler) Summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	summary, err := h.svc.GetResilienceSummary(ctx, service.GetResilienceSummaryInput{
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to get resilience summary", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	response := serviceResilienceSummaryToModel(summary)
	writeJSON(w, http.StatusOK, response)
}

// ListDRPairs returns DR pairs.
func (h *ResilienceHandler) ListDRPairs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	pairs, err := h.svc.ListDRPairs(ctx, service.ListDRPairsInput{
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to list DR pairs", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	result := make([]models.DRPair, 0, len(pairs))
	for _, p := range pairs {
		result = append(result, serviceDRPairToModel(p))
	}
	writeJSON(w, http.StatusOK, result)
}

// GetDRPair returns a specific DR pair.
func (h *ResilienceHandler) GetDRPair(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	pairID := chi.URLParam(r, "id")
	id, err := uuid.Parse(pairID)
	if err != nil {
		http.Error(w, "invalid pair ID", http.StatusBadRequest)
		return
	}

	pair, err := h.svc.GetDRPair(ctx, service.GetDRPairInput{
		ID:    id,
		OrgID: org.ID,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.Error(w, "DR pair not found", http.StatusNotFound)
			return
		}
		h.log.Error("failed to get DR pair", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, serviceDRPairToModel(*pair))
}

// TriggerFailoverTest triggers a DR failover test.
func (h *ResilienceHandler) TriggerFailoverTest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	pairID := chi.URLParam(r, "id")
	id, err := uuid.Parse(pairID)
	if err != nil {
		http.Error(w, "invalid pair ID", http.StatusBadRequest)
		return
	}

	response, err := h.svc.TriggerFailoverTest(ctx, service.TriggerFailoverTestInput{
		ID:    id,
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to trigger failover test", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusAccepted, models.TriggerFailoverTestResponse{
		JobID:     response.JobID,
		Status:    response.Status,
		StartedAt: response.StartedAt,
	})
}

// TriggerSync triggers a DR sync.
func (h *ResilienceHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	pairID := chi.URLParam(r, "id")
	id, err := uuid.Parse(pairID)
	if err != nil {
		http.Error(w, "invalid pair ID", http.StatusBadRequest)
		return
	}

	response, err := h.svc.TriggerSync(ctx, service.TriggerSyncInput{
		ID:    id,
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to trigger sync", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusAccepted, models.TriggerSyncResponse{
		JobID:     response.JobID,
		Status:    response.Status,
		StartedAt: response.StartedAt,
	})
}

// Helper functions
func serviceResilienceSummaryToModel(s *service.ResilienceSummary) models.ResilienceSummary {
	pairs := make([]models.DRPair, 0, len(s.DRPairs))
	for _, p := range s.DRPairs {
		pairs = append(pairs, serviceDRPairToModel(p))
	}

	unpairedSites := make([]models.ResilienceSite, 0, len(s.UnpairedSites))
	for _, site := range s.UnpairedSites {
		unpairedSites = append(unpairedSites, models.ResilienceSite{
			ID:         site.ID,
			Name:       site.Name,
			Region:     site.Region,
			Platform:   models.Platform(site.Platform),
			AssetCount: site.AssetCount,
			Status:     site.Status,
		})
	}

	return models.ResilienceSummary{
		DRReadiness:      s.DRReadiness,
		RPOCompliance:    s.RPOCompliance,
		RTOCompliance:    s.RTOCompliance,
		LastFailoverTest: s.LastFailoverTest,
		TotalPairs:       s.TotalPairs,
		HealthyPairs:     s.HealthyPairs,
		DRPairs:          pairs,
		UnpairedSites:    unpairedSites,
	}
}

func serviceDRPairToModel(p service.DRPair) models.DRPair {
	return models.DRPair{
		ID:                p.ID,
		OrgID:             p.OrgID,
		Name:              p.Name,
		PrimarySiteID:     p.PrimarySiteID,
		DRSiteID:          p.DRSiteID,
		Status:            p.Status,
		ReplicationStatus: p.ReplicationStatus,
		RPO:               p.RPO,
		RTO:               p.RTO,
		LastFailoverTest:  p.LastFailoverTest,
		LastSyncAt:        p.LastSyncAt,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
		PrimarySite: models.ResilienceSite{
			ID:             p.PrimarySite.ID,
			Name:           p.PrimarySite.Name,
			Region:         p.PrimarySite.Region,
			Platform:       models.Platform(p.PrimarySite.Platform),
			AssetCount:     p.PrimarySite.AssetCount,
			Status:         p.PrimarySite.Status,
			LastSyncAt:     p.PrimarySite.LastSyncAt,
			RPO:            p.PrimarySite.RPO,
			RTO:            p.PrimarySite.RTO,
			ReplicationLag: p.PrimarySite.ReplicationLag,
		},
		DRSite: models.ResilienceSite{
			ID:             p.DRSite.ID,
			Name:           p.DRSite.Name,
			Region:         p.DRSite.Region,
			Platform:       models.Platform(p.DRSite.Platform),
			AssetCount:     p.DRSite.AssetCount,
			Status:         p.DRSite.Status,
			LastSyncAt:     p.DRSite.LastSyncAt,
			RPO:            p.DRSite.RPO,
			RTO:            p.DRSite.RTO,
			ReplicationLag: p.DRSite.ReplicationLag,
		},
	}
}
