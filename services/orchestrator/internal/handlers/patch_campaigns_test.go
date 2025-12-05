// Package handlers provides HTTP handlers for the AI orchestrator service.
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// List Patch Campaigns Tests
// =============================================================================

func TestListPatchCampaigns_Success(t *testing.T) {
	h := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/patch-campaigns", nil)
	rec := httptest.NewRecorder()

	h.listPatchCampaigns(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response PatchCampaignListResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(response.Campaigns), 1, "should return at least one campaign")
	assert.Greater(t, response.Total, 0, "total should be greater than 0")
	assert.Equal(t, 1, response.Page, "default page should be 1")
	assert.Equal(t, 50, response.PageSize, "default page size should be 50")
}

func TestListPatchCampaigns_WithPagination(t *testing.T) {
	h := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/patch-campaigns?page=1&page_size=2", nil)
	rec := httptest.NewRecorder()

	h.listPatchCampaigns(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response PatchCampaignListResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, 1, response.Page)
	assert.Equal(t, 2, response.PageSize)
	assert.LessOrEqual(t, len(response.Campaigns), 2, "should not exceed page size")
}

func TestListPatchCampaigns_FilterByStatus(t *testing.T) {
	h := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/patch-campaigns?status=in_progress", nil)
	rec := httptest.NewRecorder()

	h.listPatchCampaigns(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response PatchCampaignListResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	for _, campaign := range response.Campaigns {
		assert.Equal(t, PatchCampaignStatusInProgress, campaign.Status, "all campaigns should be in_progress")
	}
}

func TestListPatchCampaigns_FilterByCampaignType(t *testing.T) {
	h := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/patch-campaigns?campaign_type=cve_response", nil)
	rec := httptest.NewRecorder()

	h.listPatchCampaigns(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response PatchCampaignListResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	for _, campaign := range response.Campaigns {
		assert.Equal(t, "cve_response", campaign.CampaignType, "all campaigns should be cve_response")
	}
}

func TestListPatchCampaigns_InvalidPagination(t *testing.T) {
	h := testHandler(t)

	// Invalid page should default to 1
	req := httptest.NewRequest(http.MethodGet, "/api/v1/patch-campaigns?page=-1", nil)
	rec := httptest.NewRecorder()

	h.listPatchCampaigns(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response PatchCampaignListResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, 1, response.Page, "invalid page should default to 1")
}

func TestListPatchCampaigns_PageSizeLimit(t *testing.T) {
	h := testHandler(t)

	// Page size > 100 should be limited to 100
	req := httptest.NewRequest(http.MethodGet, "/api/v1/patch-campaigns?page_size=200", nil)
	rec := httptest.NewRecorder()

	h.listPatchCampaigns(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response PatchCampaignListResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	// Should use default of 50 since 200 > 100
	assert.Equal(t, 50, response.PageSize, "page size > 100 should use default")
}

// =============================================================================
// Get Patch Campaign Summary Tests
// =============================================================================

func TestGetPatchCampaignSummary_Success(t *testing.T) {
	h := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/patch-campaigns/summary", nil)
	rec := httptest.NewRecorder()

	h.getPatchCampaignSummary(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var summary PatchCampaignSummary
	err := json.NewDecoder(rec.Body).Decode(&summary)
	require.NoError(t, err)

	// Verify mock data values
	assert.Equal(t, 8, summary.TotalCampaigns)
	assert.Equal(t, 2, summary.ActiveCampaigns)
	assert.Equal(t, 5, summary.CompletedCampaigns)
	assert.Equal(t, 1, summary.FailedCampaigns)
	assert.Equal(t, 342, summary.TotalAssetsPatched)
	assert.Equal(t, 3, summary.TotalRollbacks)
	assert.Equal(t, 94.5, summary.SuccessRate)
}

func TestGetPatchCampaignSummary_ValidJSON(t *testing.T) {
	h := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/patch-campaigns/summary", nil)
	rec := httptest.NewRecorder()

	h.getPatchCampaignSummary(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	// Verify response is valid JSON
	var result map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)

	// Check required fields exist
	requiredFields := []string{
		"total_campaigns",
		"active_campaigns",
		"completed_campaigns",
		"failed_campaigns",
		"total_assets_patched",
		"total_rollbacks",
		"success_rate",
	}

	for _, field := range requiredFields {
		_, exists := result[field]
		assert.True(t, exists, "missing field: %s", field)
	}
}

// =============================================================================
// Get Single Patch Campaign Tests
// =============================================================================

func TestGetPatchCampaign_Success(t *testing.T) {
	h := testHandler(t)

	campaignID := uuid.New().String()

	// Create a router to handle URL params
	r := chi.NewRouter()
	r.Get("/api/v1/patch-campaigns/{campaignID}", h.getPatchCampaign)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/patch-campaigns/"+campaignID, nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var campaign PatchCampaign
	err := json.NewDecoder(rec.Body).Decode(&campaign)
	require.NoError(t, err)

	assert.Equal(t, campaignID, campaign.ID)
	assert.NotEmpty(t, campaign.Name)
	assert.NotEmpty(t, campaign.Status)
	assert.NotEmpty(t, campaign.RolloutStrategy)
}

func TestGetPatchCampaign_HasPhases(t *testing.T) {
	h := testHandler(t)

	campaignID := uuid.New().String()

	r := chi.NewRouter()
	r.Get("/api/v1/patch-campaigns/{campaignID}", h.getPatchCampaign)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/patch-campaigns/"+campaignID, nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var campaign PatchCampaign
	err := json.NewDecoder(rec.Body).Decode(&campaign)
	require.NoError(t, err)

	// Verify phases are populated
	assert.NotNil(t, campaign.Phases)
	assert.Greater(t, len(campaign.Phases), 0, "should have at least one phase")

	// Check first phase has required fields
	phase := campaign.Phases[0]
	assert.NotEmpty(t, phase.ID)
	assert.Equal(t, campaignID, phase.CampaignID)
	assert.NotEmpty(t, phase.Name)
	assert.NotEmpty(t, phase.PhaseType)
}

// =============================================================================
// Create Patch Campaign Tests
// =============================================================================

func TestCreatePatchCampaign_Success(t *testing.T) {
	h := testHandler(t)

	r := chi.NewRouter()
	r.Post("/api/v1/patch-campaigns", h.createPatchCampaign)

	requestBody := CreatePatchCampaignRequest{
		Name:                "Test Campaign",
		CampaignType:        "cve_response",
		RolloutStrategy:     "canary",
		CanaryPercentage:    intPtr(5),
		HealthCheckEnabled:  true,
		AutoRollbackEnabled: true,
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patch-campaigns", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var campaign PatchCampaign
	err := json.NewDecoder(rec.Body).Decode(&campaign)
	require.NoError(t, err)

	assert.NotEmpty(t, campaign.ID)
	assert.Equal(t, requestBody.Name, campaign.Name)
	assert.Equal(t, requestBody.CampaignType, campaign.CampaignType)
	assert.Equal(t, requestBody.RolloutStrategy, campaign.RolloutStrategy)
	assert.Equal(t, PatchCampaignStatusDraft, campaign.Status)
}

func TestCreatePatchCampaign_WithApproval(t *testing.T) {
	h := testHandler(t)

	r := chi.NewRouter()
	r.Post("/api/v1/patch-campaigns", h.createPatchCampaign)

	requestBody := CreatePatchCampaignRequest{
		Name:             "Test Campaign",
		CampaignType:     "cve_response",
		RolloutStrategy:  "rolling",
		RequiresApproval: true,
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patch-campaigns", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var campaign PatchCampaign
	err := json.NewDecoder(rec.Body).Decode(&campaign)
	require.NoError(t, err)

	assert.Equal(t, PatchCampaignStatusPendingApproval, campaign.Status)
	assert.True(t, campaign.RequiresApproval)
}

func TestCreatePatchCampaign_InvalidBody(t *testing.T) {
	h := testHandler(t)

	r := chi.NewRouter()
	r.Post("/api/v1/patch-campaigns", h.createPatchCampaign)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patch-campaigns", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreatePatchCampaign_MissingName(t *testing.T) {
	h := testHandler(t)

	r := chi.NewRouter()
	r.Post("/api/v1/patch-campaigns", h.createPatchCampaign)

	requestBody := CreatePatchCampaignRequest{
		CampaignType:    "cve_response",
		RolloutStrategy: "canary",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patch-campaigns", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreatePatchCampaign_MissingCampaignType(t *testing.T) {
	h := testHandler(t)

	r := chi.NewRouter()
	r.Post("/api/v1/patch-campaigns", h.createPatchCampaign)

	requestBody := CreatePatchCampaignRequest{
		Name:            "Test Campaign",
		RolloutStrategy: "canary",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patch-campaigns", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreatePatchCampaign_MissingRolloutStrategy(t *testing.T) {
	h := testHandler(t)

	r := chi.NewRouter()
	r.Post("/api/v1/patch-campaigns", h.createPatchCampaign)

	requestBody := CreatePatchCampaignRequest{
		Name:         "Test Campaign",
		CampaignType: "cve_response",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patch-campaigns", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreatePatchCampaign_AllStrategies(t *testing.T) {
	h := testHandler(t)

	strategies := []string{"immediate", "canary", "blue_green", "rolling"}

	for _, strategy := range strategies {
		t.Run(strategy, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/api/v1/patch-campaigns", h.createPatchCampaign)

			requestBody := CreatePatchCampaignRequest{
				Name:            "Test Campaign",
				CampaignType:    "cve_response",
				RolloutStrategy: strategy,
			}
			body, _ := json.Marshal(requestBody)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/patch-campaigns", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusCreated, rec.Code)

			var campaign PatchCampaign
			err := json.NewDecoder(rec.Body).Decode(&campaign)
			require.NoError(t, err)

			assert.Equal(t, strategy, campaign.RolloutStrategy)
		})
	}
}

// =============================================================================
// Approve/Reject Patch Campaign Tests
// =============================================================================

func TestApprovePatchCampaign_Success(t *testing.T) {
	h := testHandler(t)

	campaignID := uuid.New().String()

	r := chi.NewRouter()
	r.Post("/api/v1/patch-campaigns/{campaignID}/approve", h.approvePatchCampaign)

	requestBody := ApprovePatchCampaignRequest{
		ApprovedBy: "admin@example.com",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patch-campaigns/"+campaignID+"/approve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var campaign PatchCampaign
	err := json.NewDecoder(rec.Body).Decode(&campaign)
	require.NoError(t, err)

	assert.Equal(t, PatchCampaignStatusApproved, campaign.Status)
	require.NotNil(t, campaign.ApprovedBy)
	assert.Equal(t, "admin@example.com", *campaign.ApprovedBy)
	assert.NotNil(t, campaign.ApprovedAt)
}

func TestApprovePatchCampaign_InvalidBody(t *testing.T) {
	h := testHandler(t)

	campaignID := uuid.New().String()

	r := chi.NewRouter()
	r.Post("/api/v1/patch-campaigns/{campaignID}/approve", h.approvePatchCampaign)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patch-campaigns/"+campaignID+"/approve", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRejectPatchCampaign_Success(t *testing.T) {
	h := testHandler(t)

	campaignID := uuid.New().String()

	r := chi.NewRouter()
	r.Post("/api/v1/patch-campaigns/{campaignID}/reject", h.rejectPatchCampaign)

	requestBody := RejectPatchCampaignRequest{
		RejectedBy: "security-team@example.com",
		Reason:     "Requires additional review",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patch-campaigns/"+campaignID+"/reject", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var campaign PatchCampaign
	err := json.NewDecoder(rec.Body).Decode(&campaign)
	require.NoError(t, err)

	assert.Equal(t, PatchCampaignStatusCancelled, campaign.Status)
}

func TestRejectPatchCampaign_InvalidBody(t *testing.T) {
	h := testHandler(t)

	campaignID := uuid.New().String()

	r := chi.NewRouter()
	r.Post("/api/v1/patch-campaigns/{campaignID}/reject", h.rejectPatchCampaign)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patch-campaigns/"+campaignID+"/reject", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// =============================================================================
// Campaign Lifecycle Tests (Start/Pause/Resume/Cancel)
// =============================================================================

func TestStartPatchCampaign_Success(t *testing.T) {
	h := testHandler(t)

	campaignID := uuid.New().String()

	r := chi.NewRouter()
	r.Post("/api/v1/patch-campaigns/{campaignID}/start", h.startPatchCampaign)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patch-campaigns/"+campaignID+"/start", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var campaign PatchCampaign
	err := json.NewDecoder(rec.Body).Decode(&campaign)
	require.NoError(t, err)

	assert.Equal(t, PatchCampaignStatusInProgress, campaign.Status)
	assert.NotNil(t, campaign.StartedAt)
}

func TestPausePatchCampaign_Success(t *testing.T) {
	h := testHandler(t)

	campaignID := uuid.New().String()

	r := chi.NewRouter()
	r.Post("/api/v1/patch-campaigns/{campaignID}/pause", h.pausePatchCampaign)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patch-campaigns/"+campaignID+"/pause", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var campaign PatchCampaign
	err := json.NewDecoder(rec.Body).Decode(&campaign)
	require.NoError(t, err)

	assert.Equal(t, PatchCampaignStatusPaused, campaign.Status)
}

func TestResumePatchCampaign_Success(t *testing.T) {
	h := testHandler(t)

	campaignID := uuid.New().String()

	r := chi.NewRouter()
	r.Post("/api/v1/patch-campaigns/{campaignID}/resume", h.resumePatchCampaign)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patch-campaigns/"+campaignID+"/resume", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var campaign PatchCampaign
	err := json.NewDecoder(rec.Body).Decode(&campaign)
	require.NoError(t, err)

	assert.Equal(t, PatchCampaignStatusInProgress, campaign.Status)
}

func TestCancelPatchCampaign_Success(t *testing.T) {
	h := testHandler(t)

	campaignID := uuid.New().String()

	r := chi.NewRouter()
	r.Post("/api/v1/patch-campaigns/{campaignID}/cancel", h.cancelPatchCampaign)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patch-campaigns/"+campaignID+"/cancel", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var campaign PatchCampaign
	err := json.NewDecoder(rec.Body).Decode(&campaign)
	require.NoError(t, err)

	assert.Equal(t, PatchCampaignStatusCancelled, campaign.Status)
}

// =============================================================================
// Rollback Tests
// =============================================================================

func TestRollbackPatchCampaign_Success(t *testing.T) {
	h := testHandler(t)

	campaignID := uuid.New().String()

	r := chi.NewRouter()
	r.Post("/api/v1/patch-campaigns/{campaignID}/rollback", h.rollbackPatchCampaign)

	requestBody := TriggerRollbackRequest{
		Scope:  "all",
		Reason: "Health check failed",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patch-campaigns/"+campaignID+"/rollback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "all", result["rollback_scope"])
	assert.Equal(t, "Health check failed", result["rollback_reason"])
	assert.Equal(t, "Rollback initiated successfully", result["message"])

	campaign, ok := result["campaign"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, string(PatchCampaignStatusRolledBack), campaign["status"])
}

func TestRollbackPatchCampaign_PartialScope(t *testing.T) {
	h := testHandler(t)

	campaignID := uuid.New().String()

	r := chi.NewRouter()
	r.Post("/api/v1/patch-campaigns/{campaignID}/rollback", h.rollbackPatchCampaign)

	requestBody := TriggerRollbackRequest{
		Scope:    "partial",
		Reason:   "Failed assets only",
		AssetIDs: []string{uuid.New().String(), uuid.New().String()},
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patch-campaigns/"+campaignID+"/rollback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRollbackPatchCampaign_InvalidBody(t *testing.T) {
	h := testHandler(t)

	campaignID := uuid.New().String()

	r := chi.NewRouter()
	r.Post("/api/v1/patch-campaigns/{campaignID}/rollback", h.rollbackPatchCampaign)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patch-campaigns/"+campaignID+"/rollback", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// =============================================================================
// Get Phases/Assets/Progress Tests
// =============================================================================

func TestGetPatchCampaignPhases_Success(t *testing.T) {
	h := testHandler(t)

	campaignID := uuid.New().String()

	r := chi.NewRouter()
	r.Get("/api/v1/patch-campaigns/{campaignID}/phases", h.getPatchCampaignPhases)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/patch-campaigns/"+campaignID+"/phases", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, campaignID, result["campaign_id"])

	phases, ok := result["phases"].([]interface{})
	require.True(t, ok, "phases should be an array")
	require.Greater(t, len(phases), 0, "should have at least one phase")

	// Check first phase has required fields
	phase, ok := phases[0].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, phase["id"])
	assert.NotEmpty(t, phase["name"])
	assert.NotEmpty(t, phase["phase_type"])
	assert.NotNil(t, phase["target_percentage"])
}

func TestGetPatchCampaignAssets_Success(t *testing.T) {
	h := testHandler(t)

	campaignID := uuid.New().String()

	r := chi.NewRouter()
	r.Get("/api/v1/patch-campaigns/{campaignID}/assets", h.getPatchCampaignAssets)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/patch-campaigns/"+campaignID+"/assets", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, campaignID, result["campaign_id"])

	assets, ok := result["assets"].([]interface{})
	require.True(t, ok, "assets should be an array")
	require.Greater(t, len(assets), 0, "should have at least one asset")

	// Check first asset has required fields
	asset, ok := assets[0].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, asset["id"])
	assert.NotEmpty(t, asset["asset_name"])
	assert.NotEmpty(t, asset["platform"])
	assert.NotEmpty(t, asset["status"])
}

func TestGetPatchCampaignProgress_Success(t *testing.T) {
	h := testHandler(t)

	campaignID := uuid.New().String()

	r := chi.NewRouter()
	r.Get("/api/v1/patch-campaigns/{campaignID}/progress", h.getPatchCampaignProgress)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/patch-campaigns/"+campaignID+"/progress", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, campaignID, result["campaign_id"])
	assert.NotNil(t, result["total_assets"])
	assert.NotNil(t, result["completed_assets"])
	assert.NotNil(t, result["failed_assets"])
	assert.NotNil(t, result["completion_percentage"])
	assert.NotNil(t, result["total_phases"])
	assert.NotNil(t, result["completed_phases"])
	assert.NotNil(t, result["current_phase"])
}

// =============================================================================
// Route Registration Tests
// =============================================================================

func TestRegisterPatchCampaignRoutes(t *testing.T) {
	h := testHandler(t)

	r := chi.NewRouter()
	h.RegisterPatchCampaignRoutes(r)

	// Test each endpoint exists
	testCases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/patch-campaigns"},
		{http.MethodGet, "/patch-campaigns/summary"},
		{http.MethodPost, "/patch-campaigns"},
		{http.MethodGet, "/patch-campaigns/" + uuid.New().String()},
		{http.MethodPost, "/patch-campaigns/" + uuid.New().String() + "/approve"},
		{http.MethodPost, "/patch-campaigns/" + uuid.New().String() + "/reject"},
		{http.MethodPost, "/patch-campaigns/" + uuid.New().String() + "/start"},
		{http.MethodPost, "/patch-campaigns/" + uuid.New().String() + "/pause"},
		{http.MethodPost, "/patch-campaigns/" + uuid.New().String() + "/resume"},
		{http.MethodPost, "/patch-campaigns/" + uuid.New().String() + "/cancel"},
		{http.MethodPost, "/patch-campaigns/" + uuid.New().String() + "/rollback"},
		{http.MethodGet, "/patch-campaigns/" + uuid.New().String() + "/phases"},
		{http.MethodGet, "/patch-campaigns/" + uuid.New().String() + "/assets"},
		{http.MethodGet, "/patch-campaigns/" + uuid.New().String() + "/progress"},
	}

	for _, tc := range testCases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			var body *bytes.Reader
			if tc.method == http.MethodPost {
				body = bytes.NewReader([]byte(`{
					"name": "test",
					"campaign_type": "cve_response",
					"rollout_strategy": "canary",
					"approved_by": "admin",
					"rejected_by": "admin",
					"reason": "test",
					"scope": "all"
				}`))
			} else {
				body = bytes.NewReader(nil)
			}

			req := httptest.NewRequest(tc.method, tc.path, body)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			// Should not be 404 (route not found) or 405 (method not allowed)
			assert.NotEqual(t, http.StatusNotFound, rec.Code, "route should exist")
			assert.NotEqual(t, http.StatusMethodNotAllowed, rec.Code, "method should be allowed")
		})
	}
}

// =============================================================================
// Mock Campaign Data Tests
// =============================================================================

func TestMockPatchCampaigns_HasVariety(t *testing.T) {
	h := testHandler(t)

	campaigns := h.generateMockPatchCampaigns()

	// Verify we have multiple campaigns
	assert.GreaterOrEqual(t, len(campaigns), 4, "should have at least 4 mock campaigns")

	// Verify variety in statuses
	statuses := make(map[PatchCampaignStatus]bool)
	for _, c := range campaigns {
		statuses[c.Status] = true
	}
	assert.GreaterOrEqual(t, len(statuses), 3, "should have at least 3 different statuses")

	// Verify variety in rollout strategies
	strategies := make(map[string]bool)
	for _, c := range campaigns {
		strategies[c.RolloutStrategy] = true
	}
	assert.GreaterOrEqual(t, len(strategies), 2, "should have at least 2 different rollout strategies")
}

func TestMockPatchCampaign_ValidAssetCounts(t *testing.T) {
	h := testHandler(t)

	campaignID := uuid.New().String()
	campaign := h.generateMockPatchCampaign(campaignID)

	// Total should equal sum of pending + in_progress + completed + failed + skipped
	calculatedTotal := campaign.PendingAssets + campaign.InProgressAssets + campaign.CompletedAssets + campaign.FailedAssets + campaign.SkippedAssets
	assert.Equal(t, campaign.TotalAssets, calculatedTotal, "total assets should equal sum of all statuses")
}
