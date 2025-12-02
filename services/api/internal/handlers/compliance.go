package handlers

import (
	"net/http"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
)

// ComplianceHandler handles compliance-related requests.
type ComplianceHandler struct {
	svc *service.ComplianceService
	log *logger.Logger
}

// NewComplianceHandler creates a new ComplianceHandler.
func NewComplianceHandler(svc *service.ComplianceService, log *logger.Logger) *ComplianceHandler {
	return &ComplianceHandler{
		svc: svc,
		log: log.WithComponent("compliance-handler"),
	}
}

// Summary returns compliance summary.
func (h *ComplianceHandler) Summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	summary, err := h.svc.GetComplianceSummary(ctx, service.GetComplianceSummaryInput{
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to get compliance summary", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	response := serviceComplianceSummaryToModel(summary)
	writeJSON(w, http.StatusOK, response)
}

// ListFrameworks returns compliance frameworks.
func (h *ComplianceHandler) ListFrameworks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	frameworks, err := h.svc.ListFrameworks(ctx, service.ListFrameworksInput{
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to list frameworks", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	result := make([]models.ComplianceFramework, 0, len(frameworks))
	for _, f := range frameworks {
		result = append(result, serviceFrameworkToModel(f))
	}
	writeJSON(w, http.StatusOK, result)
}

// FailingControls returns failing compliance controls.
func (h *ComplianceHandler) FailingControls(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	controls, err := h.svc.ListFailingControls(ctx, service.ListFailingControlsInput{
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to list failing controls", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	result := make([]models.FailingControl, 0, len(controls))
	for _, c := range controls {
		result = append(result, models.FailingControl{
			ID:             c.ID,
			Framework:      c.Framework,
			Title:          c.Title,
			Severity:       c.Severity,
			AffectedAssets: c.AffectedAssets,
			Recommendation: c.Recommendation,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

// ImageCompliance returns image compliance status.
func (h *ComplianceHandler) ImageCompliance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	images, err := h.svc.ListImageCompliance(ctx, service.ListImageComplianceInput{
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to list image compliance", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	result := make([]models.ImageComplianceStatus, 0, len(images))
	for _, i := range images {
		result = append(result, models.ImageComplianceStatus{
			FamilyID:     i.FamilyID,
			FamilyName:   i.FamilyName,
			Version:      i.Version,
			CIS:          i.CIS,
			SLSALevel:    i.SLSALevel,
			CosignSigned: i.CosignSigned,
			LastScanAt:   i.LastScanAt,
			IssueCount:   i.IssueCount,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

// TriggerAudit triggers a compliance audit.
func (h *ComplianceHandler) TriggerAudit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	response, err := h.svc.TriggerAudit(ctx, service.TriggerAuditInput{
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to trigger audit", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusAccepted, models.TriggerAuditResponse{
		JobID:     response.JobID,
		Status:    response.Status,
		StartedAt: response.StartedAt,
	})
}

// Helper functions
func serviceComplianceSummaryToModel(s *service.ComplianceSummary) models.ComplianceSummary {
	frameworks := make([]models.ComplianceFramework, 0, len(s.Frameworks))
	for _, f := range s.Frameworks {
		frameworks = append(frameworks, serviceFrameworkToModel(f))
	}

	failingControls := make([]models.FailingControl, 0, len(s.FailingControls))
	for _, c := range s.FailingControls {
		failingControls = append(failingControls, models.FailingControl{
			ID:             c.ID,
			Framework:      c.Framework,
			Title:          c.Title,
			Severity:       c.Severity,
			AffectedAssets: c.AffectedAssets,
			Recommendation: c.Recommendation,
		})
	}

	imageCompliance := make([]models.ImageComplianceStatus, 0, len(s.ImageCompliance))
	for _, i := range s.ImageCompliance {
		imageCompliance = append(imageCompliance, models.ImageComplianceStatus{
			FamilyID:     i.FamilyID,
			FamilyName:   i.FamilyName,
			Version:      i.Version,
			CIS:          i.CIS,
			SLSALevel:    i.SLSALevel,
			CosignSigned: i.CosignSigned,
			LastScanAt:   i.LastScanAt,
			IssueCount:   i.IssueCount,
		})
	}

	return models.ComplianceSummary{
		OverallScore:     s.OverallScore,
		CISCompliance:    s.CISCompliance,
		SLSALevel:        s.SLSALevel,
		SigstoreVerified: s.SigstoreVerified,
		LastAuditAt:      s.LastAuditAt,
		Frameworks:       frameworks,
		FailingControls:  failingControls,
		ImageCompliance:  imageCompliance,
	}
}

func serviceFrameworkToModel(f service.ComplianceFramework) models.ComplianceFramework {
	return models.ComplianceFramework{
		ID:              f.ID,
		OrgID:           f.OrgID,
		Name:            f.Name,
		Description:     f.Description,
		Level:           f.Level,
		Enabled:         f.Enabled,
		Score:           f.Score,
		PassingControls: f.PassingControls,
		TotalControls:   f.TotalControls,
		Status:          f.Status,
		CreatedAt:       f.CreatedAt,
		UpdatedAt:       f.UpdatedAt,
	}
}
