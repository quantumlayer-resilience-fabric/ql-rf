package handlers

import (
	"net/http"
	"strconv"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
)

// RiskHandler handles risk scoring requests.
type RiskHandler struct {
	svc *service.RiskService
	log *logger.Logger
}

// NewRiskHandler creates a new RiskHandler.
func NewRiskHandler(svc *service.RiskService, log *logger.Logger) *RiskHandler {
	return &RiskHandler{
		svc: svc,
		log: log.WithComponent("risk-handler"),
	}
}

// Summary returns organization-wide risk summary.
func (h *RiskHandler) Summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	summary, err := h.svc.GetRiskSummary(ctx, service.GetRiskSummaryInput{
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to get risk summary", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

// TopRisks returns the highest risk assets.
func (h *RiskHandler) TopRisks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	// Parse limit parameter
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	risks, err := h.svc.GetTopRisks(ctx, service.GetTopRisksInput{
		OrgID: org.ID,
		Limit: limit,
	})
	if err != nil {
		h.log.Error("failed to get top risks", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, risks)
}
