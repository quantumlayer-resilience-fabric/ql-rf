package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/inspec"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
)

// InSpecServiceInterface defines the methods required from the InSpec service.
type InSpecServiceInterface interface {
	GetAvailableProfiles(ctx context.Context) ([]inspec.AvailableProfile, error)
	GetProfile(ctx context.Context, profileID uuid.UUID) (*inspec.Profile, error)
	CreateProfile(ctx context.Context, profile inspec.Profile) (*inspec.Profile, error)
	CreateRun(ctx context.Context, orgID, assetID, profileID uuid.UUID) (*inspec.Run, error)
	ListRuns(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]inspec.RunSummary, error)
	GetRun(ctx context.Context, runID uuid.UUID) (*inspec.Run, error)
	GetRunResults(ctx context.Context, runID uuid.UUID) ([]inspec.Result, error)
	UpdateRunStatus(ctx context.Context, runID uuid.UUID, status inspec.RunStatus, errorMsg string) error
	GetControlMappings(ctx context.Context, profileID uuid.UUID) ([]inspec.ControlMapping, error)
	CreateControlMapping(ctx context.Context, mapping inspec.ControlMapping) (*inspec.ControlMapping, error)
}

// InSpecHandler handles InSpec-related requests.
type InSpecHandler struct {
	svc InSpecServiceInterface
	log *logger.Logger
}

// NewInSpecHandler creates a new InSpecHandler.
func NewInSpecHandler(svc *inspec.Service, log *logger.Logger) *InSpecHandler {
	return &InSpecHandler{
		svc: svc,
		log: log.WithComponent("inspec-handler"),
	}
}

// NewInSpecHandlerWithInterface creates a new InSpecHandler with interface dependency (for testing).
func NewInSpecHandlerWithInterface(svc InSpecServiceInterface, log *logger.Logger) *InSpecHandler {
	return &InSpecHandler{
		svc: svc,
		log: log.WithComponent("inspec-handler"),
	}
}

// ListProfiles returns all available InSpec profiles.
// GET /api/v1/inspec/profiles
func (h *InSpecHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	profiles, err := h.svc.GetAvailableProfiles(ctx)
	if err != nil {
		h.log.Error("failed to list profiles", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"profiles": profiles,
	})
}

// GetProfile returns a specific profile by ID.
// GET /api/v1/inspec/profiles/{profileId}
func (h *InSpecHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	profileIDStr := chi.URLParam(r, "profileId")
	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		http.Error(w, "invalid profile ID", http.StatusBadRequest)
		return
	}

	profile, err := h.svc.GetProfile(ctx, profileID)
	if err != nil {
		h.log.Error("failed to get profile", "error", err, "profile_id", profileID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if profile == nil {
		http.Error(w, "profile not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

// CreateProfile creates a new InSpec profile.
// POST /api/v1/inspec/profiles
func (h *InSpecHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
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

	var req inspec.CreateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" || req.Version == "" || req.Title == "" || req.FrameworkID == uuid.Nil {
		http.Error(w, "name, version, title, and framework_id are required", http.StatusBadRequest)
		return
	}

	profile := inspec.Profile{
		Name:        req.Name,
		Version:     req.Version,
		Title:       req.Title,
		Maintainer:  req.Maintainer,
		Summary:     req.Summary,
		FrameworkID: req.FrameworkID,
		ProfileURL:  req.ProfileURL,
		Platforms:   req.Platforms,
	}

	created, err := h.svc.CreateProfile(ctx, profile)
	if err != nil {
		h.log.Error("failed to create profile", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.log.Info("profile created",
		"profile_id", created.ID,
		"name", created.Name,
		"created_by", user.ExternalID,
	)

	writeJSON(w, http.StatusCreated, created)
}

// RunProfile initiates an InSpec profile run against an asset.
// POST /api/v1/inspec/run
func (h *InSpecHandler) RunProfile(w http.ResponseWriter, r *http.Request) {
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

	var req inspec.RunProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ProfileID == uuid.Nil || req.AssetID == uuid.Nil {
		http.Error(w, "profile_id and asset_id are required", http.StatusBadRequest)
		return
	}

	// Create the run
	run, err := h.svc.CreateRun(ctx, org.ID, req.AssetID, req.ProfileID)
	if err != nil {
		h.log.Error("failed to create run", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.log.Info("InSpec run created",
		"run_id", run.ID,
		"profile_id", req.ProfileID,
		"asset_id", req.AssetID,
		"initiated_by", user.ExternalID,
	)

	// TODO: Trigger Temporal workflow to execute the InSpec run asynchronously
	// For now, we just return the created run with pending status

	writeJSON(w, http.StatusAccepted, run)
}

// ListRuns returns InSpec runs for the organization.
// GET /api/v1/inspec/runs
func (h *InSpecHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	runs, err := h.svc.ListRuns(ctx, org.ID, limit, offset)
	if err != nil {
		h.log.Error("failed to list runs", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"runs":   runs,
		"limit":  limit,
		"offset": offset,
	})
}

// GetRun returns details of a specific run.
// GET /api/v1/inspec/runs/{runId}
func (h *InSpecHandler) GetRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	runIDStr := chi.URLParam(r, "runId")
	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		http.Error(w, "invalid run ID", http.StatusBadRequest)
		return
	}

	run, err := h.svc.GetRun(ctx, runID)
	if err != nil {
		h.log.Error("failed to get run", "error", err, "run_id", runID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if run == nil {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	// Verify org matches
	if run.OrgID != org.ID {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, run)
}

// GetRunResults returns the control results for a specific run.
// GET /api/v1/inspec/runs/{runId}/results
func (h *InSpecHandler) GetRunResults(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	runIDStr := chi.URLParam(r, "runId")
	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		http.Error(w, "invalid run ID", http.StatusBadRequest)
		return
	}

	// First verify the run exists and belongs to this org
	run, err := h.svc.GetRun(ctx, runID)
	if err != nil {
		h.log.Error("failed to get run", "error", err, "run_id", runID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if run == nil {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	if run.OrgID != org.ID {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	// Get the results
	results, err := h.svc.GetRunResults(ctx, runID)
	if err != nil {
		h.log.Error("failed to get run results", "error", err, "run_id", runID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	response := inspec.RunResultsResponse{
		Run:     *run,
		Results: results,
	}

	writeJSON(w, http.StatusOK, response)
}

// CancelRun cancels a pending or running InSpec run.
// POST /api/v1/inspec/runs/{runId}/cancel
func (h *InSpecHandler) CancelRun(w http.ResponseWriter, r *http.Request) {
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

	runIDStr := chi.URLParam(r, "runId")
	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		http.Error(w, "invalid run ID", http.StatusBadRequest)
		return
	}

	// Verify the run exists and belongs to this org
	run, err := h.svc.GetRun(ctx, runID)
	if err != nil {
		h.log.Error("failed to get run", "error", err, "run_id", runID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if run == nil {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	if run.OrgID != org.ID {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	// Check if run can be cancelled
	if run.Status != inspec.RunStatusPending && run.Status != inspec.RunStatusRunning {
		http.Error(w, "run cannot be cancelled in current state", http.StatusBadRequest)
		return
	}

	// Update status to cancelled
	if err := h.svc.UpdateRunStatus(ctx, runID, inspec.RunStatusCancelled, "Cancelled by user"); err != nil {
		h.log.Error("failed to cancel run", "error", err, "run_id", runID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.log.Info("InSpec run cancelled",
		"run_id", runID,
		"cancelled_by", user.ExternalID,
	)

	// TODO: Signal Temporal workflow to cancel execution

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "run cancelled successfully",
		"run_id":  runID,
	})
}

// GetControlMappings returns control mappings for a profile.
// GET /api/v1/inspec/profiles/{profileId}/mappings
func (h *InSpecHandler) GetControlMappings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	profileIDStr := chi.URLParam(r, "profileId")
	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		http.Error(w, "invalid profile ID", http.StatusBadRequest)
		return
	}

	mappings, err := h.svc.GetControlMappings(ctx, profileID)
	if err != nil {
		h.log.Error("failed to get control mappings", "error", err, "profile_id", profileID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"mappings": mappings,
	})
}

// CreateControlMapping creates a new control mapping.
// POST /api/v1/inspec/profiles/{profileId}/mappings
func (h *InSpecHandler) CreateControlMapping(w http.ResponseWriter, r *http.Request) {
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

	profileIDStr := chi.URLParam(r, "profileId")
	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		http.Error(w, "invalid profile ID", http.StatusBadRequest)
		return
	}

	var req struct {
		InSpecControlID     string    `json:"inspec_control_id"`
		ComplianceControlID uuid.UUID `json:"compliance_control_id"`
		MappingConfidence   float64   `json:"mapping_confidence"`
		Notes               string    `json:"notes,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.InSpecControlID == "" || req.ComplianceControlID == uuid.Nil {
		http.Error(w, "inspec_control_id and compliance_control_id are required", http.StatusBadRequest)
		return
	}

	// Default confidence to 1.0 if not specified
	if req.MappingConfidence <= 0 || req.MappingConfidence > 1.0 {
		req.MappingConfidence = 1.0
	}

	mapping := inspec.ControlMapping{
		InSpecControlID:     req.InSpecControlID,
		ComplianceControlID: req.ComplianceControlID,
		ProfileID:           profileID,
		MappingConfidence:   req.MappingConfidence,
		Notes:               req.Notes,
	}

	created, err := h.svc.CreateControlMapping(ctx, mapping)
	if err != nil {
		h.log.Error("failed to create control mapping", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.log.Info("control mapping created",
		"mapping_id", created.ID,
		"profile_id", profileID,
		"inspec_control", req.InSpecControlID,
		"created_by", user.ExternalID,
	)

	writeJSON(w, http.StatusCreated, created)
}
