package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/compliance"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
)

// ComplianceHandler handles compliance-related requests.
type ComplianceHandler struct {
	svc        *service.ComplianceService
	compSvc    *compliance.Service
	log        *logger.Logger
}

// NewComplianceHandler creates a new ComplianceHandler.
func NewComplianceHandler(svc *service.ComplianceService, log *logger.Logger) *ComplianceHandler {
	return &ComplianceHandler{
		svc: svc,
		log: log.WithComponent("compliance-handler"),
	}
}

// NewComplianceHandlerWithSvc creates a new ComplianceHandler with both service layers.
func NewComplianceHandlerWithSvc(svc *service.ComplianceService, compSvc *compliance.Service, log *logger.Logger) *ComplianceHandler {
	return &ComplianceHandler{
		svc:     svc,
		compSvc: compSvc,
		log:     log.WithComponent("compliance-handler"),
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

// ListControls returns all controls for a specific framework.
func (h *ComplianceHandler) ListControls(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	if h.compSvc == nil {
		http.Error(w, "compliance service not available", http.StatusServiceUnavailable)
		return
	}

	frameworkIDStr := chi.URLParam(r, "frameworkId")
	frameworkID, err := uuid.Parse(frameworkIDStr)
	if err != nil {
		http.Error(w, "invalid framework ID", http.StatusBadRequest)
		return
	}

	controls, err := h.compSvc.ListControls(ctx, frameworkID)
	if err != nil {
		h.log.Error("failed to list controls", "error", err, "framework_id", frameworkID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"controls": controls,
	})
}

// GetControlMappings returns cross-framework mappings for a control.
func (h *ComplianceHandler) GetControlMappings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	if h.compSvc == nil {
		http.Error(w, "compliance service not available", http.StatusServiceUnavailable)
		return
	}

	controlIDStr := chi.URLParam(r, "controlId")
	controlID, err := uuid.Parse(controlIDStr)
	if err != nil {
		http.Error(w, "invalid control ID", http.StatusBadRequest)
		return
	}

	mappedControls, err := h.compSvc.GetMappedControls(ctx, controlID)
	if err != nil {
		h.log.Error("failed to get control mappings", "error", err, "control_id", controlID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"mappings": mappedControls,
	})
}

// ListAssessments returns compliance assessments.
func (h *ComplianceHandler) ListAssessments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	if h.compSvc == nil {
		http.Error(w, "compliance service not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	assessments, err := h.compSvc.ListAssessments(ctx, org.ID, limit)
	if err != nil {
		h.log.Error("failed to list assessments", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"assessments": assessments,
	})
}

// CreateAssessment starts a new compliance assessment.
func (h *ComplianceHandler) CreateAssessment(w http.ResponseWriter, r *http.Request) {
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

	if h.compSvc == nil {
		http.Error(w, "compliance service not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		FrameworkID    string      `json:"framework_id"`
		Name           string      `json:"name"`
		Description    string      `json:"description,omitempty"`
		AssessmentType string      `json:"assessment_type"`
		ScopeSites     []uuid.UUID `json:"scope_sites,omitempty"`
		ScopeAssets    []uuid.UUID `json:"scope_assets,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.FrameworkID == "" || req.Name == "" {
		http.Error(w, "framework_id and name are required", http.StatusBadRequest)
		return
	}

	frameworkID, err := uuid.Parse(req.FrameworkID)
	if err != nil {
		http.Error(w, "invalid framework ID", http.StatusBadRequest)
		return
	}

	// Default to automated
	if req.AssessmentType == "" {
		req.AssessmentType = "automated"
	}

	assessment := compliance.Assessment{
		OrgID:          org.ID,
		FrameworkID:    frameworkID,
		Name:           req.Name,
		Description:    req.Description,
		AssessmentType: req.AssessmentType,
		ScopeSites:     req.ScopeSites,
		ScopeAssets:    req.ScopeAssets,
		InitiatedBy:    user.ExternalID,
	}

	created, err := h.compSvc.CreateAssessment(ctx, assessment)
	if err != nil {
		h.log.Error("failed to create assessment", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.log.Info("assessment created",
		"assessment_id", created.ID,
		"framework_id", frameworkID,
		"initiated_by", user.ExternalID,
	)

	writeJSON(w, http.StatusCreated, created)
}

// GetAssessment returns assessment details and results.
func (h *ComplianceHandler) GetAssessment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	if h.compSvc == nil {
		http.Error(w, "compliance service not available", http.StatusServiceUnavailable)
		return
	}

	assessmentIDStr := chi.URLParam(r, "assessmentId")
	assessmentID, err := uuid.Parse(assessmentIDStr)
	if err != nil {
		http.Error(w, "invalid assessment ID", http.StatusBadRequest)
		return
	}

	assessment, err := h.compSvc.GetAssessment(ctx, assessmentID)
	if err != nil {
		h.log.Error("failed to get assessment", "error", err, "assessment_id", assessmentID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if assessment == nil {
		http.Error(w, "assessment not found", http.StatusNotFound)
		return
	}

	// Verify org matches
	if assessment.OrgID != org.ID {
		http.Error(w, "assessment not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, assessment)
}

// ListEvidence returns compliance evidence.
func (h *ComplianceHandler) ListEvidence(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	if h.compSvc == nil {
		http.Error(w, "compliance service not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	controlIDStr := r.URL.Query().Get("control_id")
	if controlIDStr == "" {
		http.Error(w, "control_id query parameter is required", http.StatusBadRequest)
		return
	}

	controlID, err := uuid.Parse(controlIDStr)
	if err != nil {
		http.Error(w, "invalid control ID", http.StatusBadRequest)
		return
	}

	evidence, err := h.compSvc.ListEvidence(ctx, org.ID, controlID)
	if err != nil {
		h.log.Error("failed to list evidence", "error", err, "control_id", controlID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"evidence": evidence,
	})
}

// UploadEvidence uploads compliance evidence for a control.
func (h *ComplianceHandler) UploadEvidence(w http.ResponseWriter, r *http.Request) {
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

	if h.compSvc == nil {
		http.Error(w, "compliance service not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ControlID    string `json:"control_id"`
		EvidenceType string `json:"evidence_type"`
		Title        string `json:"title"`
		Description  string `json:"description,omitempty"`
		StorageType  string `json:"storage_type"`
		StoragePath  string `json:"storage_path,omitempty"`
		ValidUntil   string `json:"valid_until,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ControlID == "" || req.EvidenceType == "" || req.Title == "" || req.StorageType == "" {
		http.Error(w, "control_id, evidence_type, title, and storage_type are required", http.StatusBadRequest)
		return
	}

	controlID, err := uuid.Parse(req.ControlID)
	if err != nil {
		http.Error(w, "invalid control ID", http.StatusBadRequest)
		return
	}

	evidence := compliance.Evidence{
		OrgID:        org.ID,
		ControlID:    controlID,
		EvidenceType: compliance.EvidenceType(req.EvidenceType),
		Title:        req.Title,
		Description:  req.Description,
		StorageType:  req.StorageType,
		StoragePath:  req.StoragePath,
		CollectedBy:  user.ExternalID,
	}

	created, err := h.compSvc.CreateEvidence(ctx, evidence)
	if err != nil {
		h.log.Error("failed to upload evidence", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.log.Info("evidence uploaded",
		"evidence_id", created.ID,
		"control_id", controlID,
		"collected_by", user.ExternalID,
	)

	writeJSON(w, http.StatusCreated, created)
}

// ListExemptions returns active compliance exemptions.
func (h *ComplianceHandler) ListExemptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	if h.compSvc == nil {
		http.Error(w, "compliance service not available", http.StatusServiceUnavailable)
		return
	}

	exemptions, err := h.compSvc.GetActiveExemptions(ctx, org.ID)
	if err != nil {
		h.log.Error("failed to list exemptions", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"exemptions": exemptions,
	})
}

// CreateExemption requests a compliance exemption.
func (h *ComplianceHandler) CreateExemption(w http.ResponseWriter, r *http.Request) {
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

	if h.compSvc == nil {
		http.Error(w, "compliance service not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ControlID            string     `json:"control_id"`
		AssetID              *uuid.UUID `json:"asset_id,omitempty"`
		SiteID               *uuid.UUID `json:"site_id,omitempty"`
		Reason               string     `json:"reason"`
		RiskAcceptance       string     `json:"risk_acceptance,omitempty"`
		CompensatingControls string     `json:"compensating_controls,omitempty"`
		ExpiresAt            string     `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ControlID == "" || req.Reason == "" || req.ExpiresAt == "" {
		http.Error(w, "control_id, reason, and expires_at are required", http.StatusBadRequest)
		return
	}

	controlID, err := uuid.Parse(req.ControlID)
	if err != nil {
		http.Error(w, "invalid control ID", http.StatusBadRequest)
		return
	}

	// Parse expires_at
	expiresAt, err := parseTime(req.ExpiresAt)
	if err != nil {
		http.Error(w, "invalid expires_at format", http.StatusBadRequest)
		return
	}

	exemption := compliance.Exemption{
		OrgID:                org.ID,
		ControlID:            controlID,
		AssetID:              req.AssetID,
		SiteID:               req.SiteID,
		Reason:               req.Reason,
		RiskAcceptance:       req.RiskAcceptance,
		CompensatingControls: req.CompensatingControls,
		ApprovedBy:           user.ExternalID,
		ExpiresAt:            expiresAt,
		ReviewFrequencyDays:  90, // Default to 90 days
	}

	created, err := h.compSvc.CreateExemption(ctx, exemption)
	if err != nil {
		h.log.Error("failed to create exemption", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.log.Info("exemption created",
		"exemption_id", created.ID,
		"control_id", controlID,
		"approved_by", user.ExternalID,
	)

	writeJSON(w, http.StatusCreated, created)
}

// GetScore returns the overall compliance score.
func (h *ComplianceHandler) GetScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	if h.compSvc == nil {
		http.Error(w, "compliance service not available", http.StatusServiceUnavailable)
		return
	}

	// Parse optional framework_id query parameter
	var frameworkID *uuid.UUID
	if frameworkIDStr := r.URL.Query().Get("framework_id"); frameworkIDStr != "" {
		id, err := uuid.Parse(frameworkIDStr)
		if err != nil {
			http.Error(w, "invalid framework ID", http.StatusBadRequest)
			return
		}
		frameworkID = &id
	}

	score, err := h.compSvc.GetComplianceScore(ctx, org.ID, frameworkID)
	if err != nil {
		h.log.Error("failed to get compliance score", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, score)
}

// Helper function to parse time strings
func parseTime(s string) (t time.Time, err error) {
	// Try RFC3339 first
	t, err = time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}
	// Try other common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, format := range formats {
		t, err = time.Parse(format, s)
		if err == nil {
			return t, nil
		}
	}
	return t, err
}
