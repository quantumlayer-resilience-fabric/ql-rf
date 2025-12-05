// Package handlers provides HTTP handlers for the AI orchestrator service.
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/temporal/workflows"
)

// =============================================================================
// Patch Campaign Types
// =============================================================================

// PatchCampaignStatus represents the status of a patch campaign.
type PatchCampaignStatus string

const (
	PatchCampaignStatusDraft           PatchCampaignStatus = "draft"
	PatchCampaignStatusPendingApproval PatchCampaignStatus = "pending_approval"
	PatchCampaignStatusApproved        PatchCampaignStatus = "approved"
	PatchCampaignStatusScheduled       PatchCampaignStatus = "scheduled"
	PatchCampaignStatusInProgress      PatchCampaignStatus = "in_progress"
	PatchCampaignStatusPaused          PatchCampaignStatus = "paused"
	PatchCampaignStatusCompleted       PatchCampaignStatus = "completed"
	PatchCampaignStatusFailed          PatchCampaignStatus = "failed"
	PatchCampaignStatusRolledBack      PatchCampaignStatus = "rolled_back"
	PatchCampaignStatusCancelled       PatchCampaignStatus = "cancelled"
)

// PatchCampaign represents a patch campaign.
type PatchCampaign struct {
	ID                   string              `json:"id"`
	OrgID                string              `json:"org_id"`
	Name                 string              `json:"name"`
	Description          *string             `json:"description,omitempty"`
	CampaignType         string              `json:"campaign_type"`
	CVEAlertIDs          []string            `json:"cve_alert_ids,omitempty"`
	Status               PatchCampaignStatus `json:"status"`
	RequiresApproval     bool                `json:"requires_approval"`
	ApprovedBy           *string             `json:"approved_by,omitempty"`
	ApprovedAt           *time.Time          `json:"approved_at,omitempty"`
	RolloutStrategy      string              `json:"rollout_strategy"`
	CanaryPercentage     *int                `json:"canary_percentage,omitempty"`
	WavePercentage       *int                `json:"wave_percentage,omitempty"`
	FailureThreshold     *int                `json:"failure_threshold_percentage,omitempty"`
	HealthCheckEnabled   bool                `json:"health_check_enabled"`
	AutoRollbackEnabled  bool                `json:"auto_rollback_enabled"`
	TotalAssets          int                 `json:"total_assets"`
	PendingAssets        int                 `json:"pending_assets"`
	InProgressAssets     int                 `json:"in_progress_assets"`
	CompletedAssets      int                 `json:"completed_assets"`
	FailedAssets         int                 `json:"failed_assets"`
	SkippedAssets        int                 `json:"skipped_assets"`
	ScheduledStartAt     *time.Time          `json:"scheduled_start_at,omitempty"`
	StartedAt            *time.Time          `json:"started_at,omitempty"`
	CompletedAt          *time.Time          `json:"completed_at,omitempty"`
	CreatedBy            string              `json:"created_by"`
	CreatedAt            time.Time           `json:"created_at"`
	UpdatedAt            time.Time           `json:"updated_at"`
	Phases               []PatchCampaignPhase `json:"phases,omitempty"`
}

// PatchCampaignPhase represents a phase in a patch campaign.
type PatchCampaignPhase struct {
	ID               string     `json:"id"`
	CampaignID       string     `json:"campaign_id"`
	PhaseNumber      int        `json:"phase_number"`
	Name             string     `json:"name"`
	PhaseType        string     `json:"phase_type"`
	TargetPercentage int        `json:"target_percentage"`
	Status           string     `json:"status"`
	TotalAssets      int        `json:"total_assets"`
	CompletedAssets  int        `json:"completed_assets"`
	FailedAssets     int        `json:"failed_assets"`
	HealthCheckPassed *bool     `json:"health_check_passed,omitempty"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

// PatchCampaignListResponse is the response for listing patch campaigns.
type PatchCampaignListResponse struct {
	Campaigns  []PatchCampaign `json:"campaigns"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// PatchCampaignSummary provides aggregate statistics for patch campaigns.
type PatchCampaignSummary struct {
	TotalCampaigns     int     `json:"total_campaigns"`
	ActiveCampaigns    int     `json:"active_campaigns"`
	CompletedCampaigns int     `json:"completed_campaigns"`
	FailedCampaigns    int     `json:"failed_campaigns"`
	TotalAssetsPatched int     `json:"total_assets_patched"`
	TotalRollbacks     int     `json:"total_rollbacks"`
	SuccessRate        float64 `json:"success_rate"`
}

// CreatePatchCampaignRequest is the request body for creating a patch campaign.
type CreatePatchCampaignRequest struct {
	Name                 string    `json:"name"`
	Description          *string   `json:"description,omitempty"`
	CampaignType         string    `json:"campaign_type"`
	CVEAlertIDs          []string  `json:"cve_alert_ids,omitempty"`
	RolloutStrategy      string    `json:"rollout_strategy"`
	CanaryPercentage     *int      `json:"canary_percentage,omitempty"`
	WavePercentage       *int      `json:"wave_percentage,omitempty"`
	FailureThreshold     *int      `json:"failure_threshold_percentage,omitempty"`
	HealthCheckEnabled   bool      `json:"health_check_enabled"`
	AutoRollbackEnabled  bool      `json:"auto_rollback_enabled"`
	RequiresApproval     bool      `json:"requires_approval"`
	ScheduledStartAt     *string   `json:"scheduled_start_at,omitempty"`
	TargetAssetIDs       []string  `json:"target_asset_ids,omitempty"`
}

// ApprovePatchCampaignRequest is the request body for approving a patch campaign.
type ApprovePatchCampaignRequest struct {
	ApprovedBy string  `json:"approved_by"`
	Comment    *string `json:"comment,omitempty"`
}

// RejectPatchCampaignRequest is the request body for rejecting a patch campaign.
type RejectPatchCampaignRequest struct {
	RejectedBy string `json:"rejected_by"`
	Reason     string `json:"reason"`
}

// TriggerRollbackRequest is the request body for triggering a rollback.
type TriggerRollbackRequest struct {
	Scope    string   `json:"scope"`
	Reason   string   `json:"reason"`
	AssetIDs []string `json:"asset_ids,omitempty"`
	PhaseID  *string  `json:"phase_id,omitempty"`
}

// =============================================================================
// Patch Campaign Handlers
// =============================================================================

// RegisterPatchCampaignRoutes registers the patch campaign routes.
func (h *Handler) RegisterPatchCampaignRoutes(r chi.Router) {
	r.Route("/patch-campaigns", func(r chi.Router) {
		r.Get("/", h.listPatchCampaigns)
		r.Get("/summary", h.getPatchCampaignSummary)
		r.Post("/", h.createPatchCampaign)
		r.Get("/{campaignID}", h.getPatchCampaign)
		r.Post("/{campaignID}/approve", h.approvePatchCampaign)
		r.Post("/{campaignID}/reject", h.rejectPatchCampaign)
		r.Post("/{campaignID}/start", h.startPatchCampaign)
		r.Post("/{campaignID}/pause", h.pausePatchCampaign)
		r.Post("/{campaignID}/resume", h.resumePatchCampaign)
		r.Post("/{campaignID}/cancel", h.cancelPatchCampaign)
		r.Post("/{campaignID}/rollback", h.rollbackPatchCampaign)
		r.Get("/{campaignID}/phases", h.getPatchCampaignPhases)
		r.Get("/{campaignID}/assets", h.getPatchCampaignAssets)
		r.Get("/{campaignID}/progress", h.getPatchCampaignProgress)
	})
}

func (h *Handler) listPatchCampaigns(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	status := r.URL.Query().Get("status")
	campaignType := r.URL.Query().Get("campaign_type")
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")

	page := 1
	pageSize := 50
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Generate mock campaigns
	campaigns := h.generateMockPatchCampaigns()

	// Apply filters
	filtered := campaigns
	if status != "" {
		result := []PatchCampaign{}
		for _, c := range filtered {
			if string(c.Status) == status {
				result = append(result, c)
			}
		}
		filtered = result
	}
	if campaignType != "" {
		result := []PatchCampaign{}
		for _, c := range filtered {
			if c.CampaignType == campaignType {
				result = append(result, c)
			}
		}
		filtered = result
	}

	total := len(filtered)
	totalPages := (total + pageSize - 1) / pageSize

	// Paginate
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	paged := filtered[start:end]

	h.respond(w, http.StatusOK, PatchCampaignListResponse{
		Campaigns:  paged,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

func (h *Handler) getPatchCampaignSummary(w http.ResponseWriter, r *http.Request) {
	summary := PatchCampaignSummary{
		TotalCampaigns:     8,
		ActiveCampaigns:    2,
		CompletedCampaigns: 5,
		FailedCampaigns:    1,
		TotalAssetsPatched: 342,
		TotalRollbacks:     3,
		SuccessRate:        94.5,
	}

	h.respond(w, http.StatusOK, summary)
}

func (h *Handler) createPatchCampaign(w http.ResponseWriter, r *http.Request) {
	var req CreatePatchCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Validate required fields
	if req.Name == "" {
		h.respondError(w, http.StatusBadRequest, "name is required", nil)
		return
	}
	if req.CampaignType == "" {
		h.respondError(w, http.StatusBadRequest, "campaign_type is required", nil)
		return
	}
	if req.RolloutStrategy == "" {
		h.respondError(w, http.StatusBadRequest, "rollout_strategy is required", nil)
		return
	}

	now := time.Now().UTC()
	campaign := PatchCampaign{
		ID:                  uuid.New().String(),
		OrgID:               "default-org",
		Name:                req.Name,
		Description:         req.Description,
		CampaignType:        req.CampaignType,
		CVEAlertIDs:         req.CVEAlertIDs,
		Status:              PatchCampaignStatusDraft,
		RequiresApproval:    req.RequiresApproval,
		RolloutStrategy:     req.RolloutStrategy,
		CanaryPercentage:    req.CanaryPercentage,
		WavePercentage:      req.WavePercentage,
		FailureThreshold:    req.FailureThreshold,
		HealthCheckEnabled:  req.HealthCheckEnabled,
		AutoRollbackEnabled: req.AutoRollbackEnabled,
		TotalAssets:         len(req.TargetAssetIDs),
		PendingAssets:       len(req.TargetAssetIDs),
		CreatedBy:           "current-user",
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	// If requires approval, set status to pending
	if req.RequiresApproval {
		campaign.Status = PatchCampaignStatusPendingApproval
	}

	h.respond(w, http.StatusCreated, campaign)
}

func (h *Handler) getPatchCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "campaignID")
	campaign := h.generateMockPatchCampaign(campaignID)
	h.respond(w, http.StatusOK, campaign)
}

func (h *Handler) approvePatchCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "campaignID")

	var req ApprovePatchCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// If Temporal worker is available, signal approval
	if h.temporalWorker != nil {
		err := h.temporalWorker.SignalPatchCampaignApproval(r.Context(), campaignID, workflows.PatchCampaignApprovalSignal{
			Action:     "approve",
			ApprovedBy: req.ApprovedBy,
		})
		if err != nil {
			h.log.Warn("Failed to signal Temporal workflow", "campaign_id", campaignID, "error", err)
		}
	}

	now := time.Now().UTC()
	campaign := h.generateMockPatchCampaign(campaignID)
	campaign.Status = PatchCampaignStatusApproved
	campaign.ApprovedBy = &req.ApprovedBy
	campaign.ApprovedAt = &now
	campaign.UpdatedAt = now

	h.respond(w, http.StatusOK, campaign)
}

func (h *Handler) rejectPatchCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "campaignID")

	var req RejectPatchCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// If Temporal worker is available, signal rejection
	if h.temporalWorker != nil {
		err := h.temporalWorker.SignalPatchCampaignApproval(r.Context(), campaignID, workflows.PatchCampaignApprovalSignal{
			Action:     "reject",
			ApprovedBy: req.RejectedBy,
			Reason:     req.Reason,
		})
		if err != nil {
			h.log.Warn("Failed to signal Temporal workflow", "campaign_id", campaignID, "error", err)
		}
	}

	campaign := h.generateMockPatchCampaign(campaignID)
	campaign.Status = PatchCampaignStatusCancelled
	campaign.UpdatedAt = time.Now().UTC()

	h.respond(w, http.StatusOK, campaign)
}

func (h *Handler) startPatchCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "campaignID")

	// Generate phases for the campaign
	assetIDs := []string{
		uuid.New().String(),
		uuid.New().String(),
		uuid.New().String(),
		uuid.New().String(),
		uuid.New().String(),
	}

	// If Temporal worker is available, start the workflow
	if h.temporalWorker != nil {
		phases := workflows.GeneratePatchCampaignPhases(campaignID, assetIDs, "canary", 10, 25)
		input := workflows.PatchCampaignWorkflowInput{
			CampaignID:            campaignID,
			OrgID:                 "default-org",
			UserID:                "current-user",
			CampaignName:          "Test Campaign",
			CampaignType:          "cve_response",
			RolloutStrategy:       "canary",
			CanaryPercentage:      10,
			WavePercentage:        25,
			FailureThreshold:      10,
			HealthCheckEnabled:    true,
			AutoRollbackEnabled:   true,
			NotifyOnStart:         true,
			NotifyOnPhaseComplete: true,
			NotifyOnComplete:      true,
			NotifyOnFailure:       true,
			Phases:                phases,
		}

		runID, err := h.temporalWorker.StartPatchCampaignWorkflow(r.Context(), input)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "failed to start campaign workflow", err)
			return
		}
		h.log.Info("Started patch campaign workflow", "campaign_id", campaignID, "run_id", runID)
	}

	now := time.Now().UTC()
	campaign := h.generateMockPatchCampaign(campaignID)
	campaign.Status = PatchCampaignStatusInProgress
	campaign.StartedAt = &now
	campaign.UpdatedAt = now

	h.respond(w, http.StatusOK, campaign)
}

func (h *Handler) pausePatchCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "campaignID")

	// If Temporal worker is available, pause the workflow
	if h.temporalWorker != nil {
		err := h.temporalWorker.PausePatchCampaign(r.Context(), campaignID)
		if err != nil {
			h.log.Warn("Failed to pause Temporal workflow", "campaign_id", campaignID, "error", err)
		}
	}

	campaign := h.generateMockPatchCampaign(campaignID)
	campaign.Status = PatchCampaignStatusPaused
	campaign.UpdatedAt = time.Now().UTC()

	h.respond(w, http.StatusOK, campaign)
}

func (h *Handler) resumePatchCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "campaignID")

	// If Temporal worker is available, resume the workflow
	if h.temporalWorker != nil {
		err := h.temporalWorker.ResumePatchCampaign(r.Context(), campaignID)
		if err != nil {
			h.log.Warn("Failed to resume Temporal workflow", "campaign_id", campaignID, "error", err)
		}
	}

	now := time.Now().UTC()
	campaign := h.generateMockPatchCampaign(campaignID)
	campaign.Status = PatchCampaignStatusInProgress
	campaign.StartedAt = &now
	campaign.UpdatedAt = now

	h.respond(w, http.StatusOK, campaign)
}

func (h *Handler) cancelPatchCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "campaignID")

	// If Temporal worker is available, cancel the workflow
	if h.temporalWorker != nil {
		err := h.temporalWorker.CancelPatchCampaign(r.Context(), campaignID)
		if err != nil {
			h.log.Warn("Failed to cancel Temporal workflow", "campaign_id", campaignID, "error", err)
		}
	}

	campaign := h.generateMockPatchCampaign(campaignID)
	campaign.Status = PatchCampaignStatusCancelled
	campaign.UpdatedAt = time.Now().UTC()

	h.respond(w, http.StatusOK, campaign)
}

func (h *Handler) rollbackPatchCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "campaignID")

	var req TriggerRollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// In a real implementation, this would trigger a rollback via Temporal

	campaign := h.generateMockPatchCampaign(campaignID)
	campaign.Status = PatchCampaignStatusRolledBack
	campaign.UpdatedAt = time.Now().UTC()

	h.respond(w, http.StatusOK, map[string]interface{}{
		"campaign":        campaign,
		"rollback_scope":  req.Scope,
		"rollback_reason": req.Reason,
		"message":         "Rollback initiated successfully",
	})
}

func (h *Handler) getPatchCampaignPhases(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "campaignID")

	phases := []PatchCampaignPhase{
		{
			ID:               uuid.New().String(),
			CampaignID:       campaignID,
			PhaseNumber:      1,
			Name:             "Canary",
			PhaseType:        "canary",
			TargetPercentage: 5,
			Status:           "completed",
			TotalAssets:      2,
			CompletedAssets:  2,
			FailedAssets:     0,
		},
		{
			ID:               uuid.New().String(),
			CampaignID:       campaignID,
			PhaseNumber:      2,
			Name:             "Wave 1",
			PhaseType:        "wave",
			TargetPercentage: 25,
			Status:           "in_progress",
			TotalAssets:      10,
			CompletedAssets:  6,
			FailedAssets:     1,
		},
		{
			ID:               uuid.New().String(),
			CampaignID:       campaignID,
			PhaseNumber:      3,
			Name:             "Wave 2",
			PhaseType:        "wave",
			TargetPercentage: 35,
			Status:           "pending",
			TotalAssets:      14,
			CompletedAssets:  0,
			FailedAssets:     0,
		},
		{
			ID:               uuid.New().String(),
			CampaignID:       campaignID,
			PhaseNumber:      4,
			Name:             "Final",
			PhaseType:        "final",
			TargetPercentage: 35,
			Status:           "pending",
			TotalAssets:      14,
			CompletedAssets:  0,
			FailedAssets:     0,
		},
	}

	h.respond(w, http.StatusOK, map[string]interface{}{
		"campaign_id": campaignID,
		"phases":      phases,
	})
}

func (h *Handler) getPatchCampaignAssets(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "campaignID")
	status := r.URL.Query().Get("status")
	phaseID := r.URL.Query().Get("phase_id")

	_ = status
	_ = phaseID

	assets := []map[string]interface{}{
		{
			"id":           uuid.New().String(),
			"campaign_id":  campaignID,
			"asset_id":     uuid.New().String(),
			"asset_name":   "web-server-prod-1",
			"platform":     "aws",
			"status":       "completed",
			"before_version": "1.0.0",
			"after_version":  "1.0.1",
		},
		{
			"id":           uuid.New().String(),
			"campaign_id":  campaignID,
			"asset_id":     uuid.New().String(),
			"asset_name":   "api-server-prod-2",
			"platform":     "aws",
			"status":       "in_progress",
		},
		{
			"id":           uuid.New().String(),
			"campaign_id":  campaignID,
			"asset_id":     uuid.New().String(),
			"asset_name":   "db-server-staging-1",
			"platform":     "azure",
			"status":       "failed",
			"error_message": "Connection timeout",
		},
	}

	h.respond(w, http.StatusOK, map[string]interface{}{
		"campaign_id": campaignID,
		"assets":      assets,
		"total":       len(assets),
	})
}

func (h *Handler) getPatchCampaignProgress(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "campaignID")

	progress := map[string]interface{}{
		"campaign_id":           campaignID,
		"status":                "in_progress",
		"total_assets":          40,
		"completed_assets":      22,
		"failed_assets":         2,
		"skipped_assets":        1,
		"completion_percentage": 55.0,
		"failure_percentage":    5.0,
		"total_phases":          4,
		"completed_phases":      1,
		"current_phase":         "Wave 1",
		"current_phase_progress": 60.0,
		"estimated_completion":  time.Now().Add(2 * time.Hour).UTC(),
		"started_at":            time.Now().Add(-1 * time.Hour).UTC(),
		"elapsed_time_minutes":  60,
	}

	h.respond(w, http.StatusOK, progress)
}

// =============================================================================
// Mock Data Generators
// =============================================================================

func (h *Handler) generateMockPatchCampaigns() []PatchCampaign {
	now := time.Now().UTC()
	campaigns := []PatchCampaign{}

	// In progress campaign
	startedAt := now.Add(-1 * time.Hour)
	campaigns = append(campaigns, PatchCampaign{
		ID:                  uuid.New().String(),
		OrgID:               "default-org",
		Name:                "CVE-2024-3094 Emergency Patch",
		CampaignType:        "cve_response",
		CVEAlertIDs:         []string{uuid.New().String()},
		Status:              PatchCampaignStatusInProgress,
		RolloutStrategy:     "canary",
		CanaryPercentage:    intPtr(5),
		WavePercentage:      intPtr(25),
		HealthCheckEnabled:  true,
		AutoRollbackEnabled: true,
		TotalAssets:         45,
		PendingAssets:       20,
		InProgressAssets:    5,
		CompletedAssets:     18,
		FailedAssets:        2,
		StartedAt:           &startedAt,
		CreatedBy:           "admin",
		CreatedAt:           now.Add(-2 * time.Hour),
		UpdatedAt:           now,
	})

	// Pending approval campaign
	campaigns = append(campaigns, PatchCampaign{
		ID:               uuid.New().String(),
		OrgID:            "default-org",
		Name:             "Monthly Security Updates - December",
		CampaignType:     "scheduled",
		Status:           PatchCampaignStatusPendingApproval,
		RequiresApproval: true,
		RolloutStrategy:  "rolling",
		WavePercentage:   intPtr(25),
		TotalAssets:      120,
		PendingAssets:    120,
		CreatedBy:        "scheduler",
		CreatedAt:        now.Add(-12 * time.Hour),
		UpdatedAt:        now.Add(-12 * time.Hour),
	})

	// Completed campaign
	startedAt2 := now.Add(-48 * time.Hour)
	completedAt := now.Add(-24 * time.Hour)
	campaigns = append(campaigns, PatchCampaign{
		ID:               uuid.New().String(),
		OrgID:            "default-org",
		Name:             "OpenSSL Security Patch",
		CampaignType:     "cve_response",
		CVEAlertIDs:      []string{uuid.New().String()},
		Status:           PatchCampaignStatusCompleted,
		RolloutStrategy:  "canary",
		CanaryPercentage: intPtr(10),
		WavePercentage:   intPtr(30),
		TotalAssets:      67,
		CompletedAssets:  65,
		FailedAssets:     2,
		StartedAt:        &startedAt2,
		CompletedAt:      &completedAt,
		CreatedBy:        "security-team",
		CreatedAt:        now.Add(-72 * time.Hour),
		UpdatedAt:        completedAt,
	})

	// Failed campaign
	startedAt3 := now.Add(-6 * time.Hour)
	completedAt2 := now.Add(-4 * time.Hour)
	campaigns = append(campaigns, PatchCampaign{
		ID:                  uuid.New().String(),
		OrgID:               "default-org",
		Name:                "Database Security Patch",
		CampaignType:        "cve_response",
		Status:              PatchCampaignStatusFailed,
		RolloutStrategy:     "immediate",
		AutoRollbackEnabled: true,
		TotalAssets:         15,
		CompletedAssets:     3,
		FailedAssets:        12,
		StartedAt:           &startedAt3,
		CompletedAt:         &completedAt2,
		CreatedBy:           "admin",
		CreatedAt:           now.Add(-8 * time.Hour),
		UpdatedAt:           completedAt2,
	})

	return campaigns
}

func (h *Handler) generateMockPatchCampaign(campaignID string) PatchCampaign {
	now := time.Now().UTC()
	startedAt := now.Add(-1 * time.Hour)

	passed := true
	phaseStarted := now.Add(-45 * time.Minute)
	phaseCompleted := now.Add(-30 * time.Minute)

	return PatchCampaign{
		ID:                  campaignID,
		OrgID:               "default-org",
		Name:                "CVE-2024-3094 Emergency Patch",
		CampaignType:        "cve_response",
		CVEAlertIDs:         []string{uuid.New().String()},
		Status:              PatchCampaignStatusInProgress,
		RolloutStrategy:     "canary",
		CanaryPercentage:    intPtr(5),
		WavePercentage:      intPtr(25),
		FailureThreshold:    intPtr(10),
		HealthCheckEnabled:  true,
		AutoRollbackEnabled: true,
		TotalAssets:         40,
		PendingAssets:       18,
		InProgressAssets:    5,
		CompletedAssets:     15,
		FailedAssets:        2,
		StartedAt:           &startedAt,
		CreatedBy:           "admin",
		CreatedAt:           now.Add(-2 * time.Hour),
		UpdatedAt:           now,
		Phases: []PatchCampaignPhase{
			{
				ID:                uuid.New().String(),
				CampaignID:        campaignID,
				PhaseNumber:       1,
				Name:              "Canary",
				PhaseType:         "canary",
				TargetPercentage:  5,
				Status:            "completed",
				TotalAssets:       2,
				CompletedAssets:   2,
				HealthCheckPassed: &passed,
				StartedAt:         &phaseStarted,
				CompletedAt:       &phaseCompleted,
			},
			{
				ID:               uuid.New().String(),
				CampaignID:       campaignID,
				PhaseNumber:      2,
				Name:             "Wave 1",
				PhaseType:        "wave",
				TargetPercentage: 25,
				Status:           "in_progress",
				TotalAssets:      10,
				CompletedAssets:  6,
				FailedAssets:     1,
				StartedAt:        &now,
			},
		},
	}
}

func intPtr(i int) *int {
	return &i
}
