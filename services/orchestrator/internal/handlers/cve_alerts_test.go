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

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// testHandler creates a handler for testing CVE alerts endpoints.
func testHandler(t *testing.T) *Handler {
	t.Helper()

	cfg := &config.Config{
		Env: "test",
		Orchestrator: config.OrchestratorConfig{
			DevMode: true,
		},
	}

	log := logger.New("error", "text")

	return &Handler{
		cfg: cfg,
		log: log,
	}
}

// =============================================================================
// List CVE Alerts Tests
// =============================================================================

func TestListCVEAlerts_Success(t *testing.T) {
	h := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts", nil)
	rec := httptest.NewRecorder()

	h.listCVEAlerts(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response CVEAlertListResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(response.Alerts), 1, "should return at least one alert")
	assert.Greater(t, response.Total, 0, "total should be greater than 0")
	assert.Equal(t, 1, response.Page, "default page should be 1")
	assert.Equal(t, 50, response.PageSize, "default page size should be 50")
}

func TestListCVEAlerts_WithPagination(t *testing.T) {
	h := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts?page=1&page_size=2", nil)
	rec := httptest.NewRecorder()

	h.listCVEAlerts(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response CVEAlertListResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, 1, response.Page)
	assert.Equal(t, 2, response.PageSize)
	assert.LessOrEqual(t, len(response.Alerts), 2, "should not exceed page size")
}

func TestListCVEAlerts_FilterBySeverity(t *testing.T) {
	h := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts?severity=critical", nil)
	rec := httptest.NewRecorder()

	h.listCVEAlerts(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response CVEAlertListResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	for _, alert := range response.Alerts {
		assert.Equal(t, CVESeverityCritical, alert.Severity, "all alerts should be critical")
	}
}

func TestListCVEAlerts_FilterByStatus(t *testing.T) {
	h := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts?status=new", nil)
	rec := httptest.NewRecorder()

	h.listCVEAlerts(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response CVEAlertListResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	for _, alert := range response.Alerts {
		assert.Equal(t, CVEAlertStatusNew, alert.Status, "all alerts should have 'new' status")
	}
}

func TestListCVEAlerts_MultipleSeverities(t *testing.T) {
	h := testHandler(t)

	testCases := []struct {
		name     string
		severity string
		expected CVESeverity
	}{
		{"critical", "critical", CVESeverityCritical},
		{"high", "high", CVESeverityHigh},
		{"medium", "medium", CVESeverityMedium},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts?severity="+tc.severity, nil)
			rec := httptest.NewRecorder()

			h.listCVEAlerts(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)

			var response CVEAlertListResponse
			err := json.NewDecoder(rec.Body).Decode(&response)
			require.NoError(t, err)

			for _, alert := range response.Alerts {
				assert.Equal(t, tc.expected, alert.Severity)
			}
		})
	}
}

func TestListCVEAlerts_InvalidPagination(t *testing.T) {
	h := testHandler(t)

	// Invalid page should default to 1
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts?page=-1", nil)
	rec := httptest.NewRecorder()

	h.listCVEAlerts(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response CVEAlertListResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, 1, response.Page, "invalid page should default to 1")
}

func TestListCVEAlerts_PageSizeLimit(t *testing.T) {
	h := testHandler(t)

	// Page size > 100 should be limited to 100
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts?page_size=200", nil)
	rec := httptest.NewRecorder()

	h.listCVEAlerts(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response CVEAlertListResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	// Should use default of 50 since 200 > 100
	assert.Equal(t, 50, response.PageSize, "page size > 100 should use default")
}

// =============================================================================
// Get CVE Alert Summary Tests
// =============================================================================

func TestGetCVEAlertSummary_Success(t *testing.T) {
	h := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts/summary", nil)
	rec := httptest.NewRecorder()

	h.getCVEAlertSummary(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var summary CVEAlertSummary
	err := json.NewDecoder(rec.Body).Decode(&summary)
	require.NoError(t, err)

	// Verify mock data values
	assert.Equal(t, 12, summary.TotalAlerts)
	assert.Equal(t, 4, summary.NewAlerts)
	assert.Equal(t, 3, summary.InProgressAlerts)
	assert.Equal(t, 5, summary.ResolvedAlerts)
	assert.Equal(t, 2, summary.CriticalAlerts)
	assert.Equal(t, 4, summary.HighAlerts)
	assert.Equal(t, 4, summary.MediumAlerts)
	assert.Equal(t, 2, summary.LowAlerts)
	assert.Equal(t, 1, summary.SLABreachedAlerts)
	assert.Equal(t, 3, summary.ExploitableAlerts)
	assert.Equal(t, 2, summary.CISAKEVAlerts)
	assert.Equal(t, 68.5, summary.AverageUrgencyScore)
	assert.Equal(t, 156, summary.TotalAffectedAssets)
	assert.Equal(t, 42, summary.ProductionAffectedAssets)
}

func TestGetCVEAlertSummary_ValidJSON(t *testing.T) {
	h := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts/summary", nil)
	rec := httptest.NewRecorder()

	h.getCVEAlertSummary(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	// Verify response is valid JSON
	var result map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)

	// Check required fields exist
	requiredFields := []string{
		"total_alerts",
		"new_alerts",
		"in_progress_alerts",
		"resolved_alerts",
		"critical_alerts",
		"high_alerts",
		"medium_alerts",
		"low_alerts",
		"sla_breached_alerts",
		"exploitable_alerts",
		"cisa_kev_alerts",
		"average_urgency_score",
		"total_affected_assets",
		"production_affected_assets",
	}

	for _, field := range requiredFields {
		_, exists := result[field]
		assert.True(t, exists, "missing field: %s", field)
	}
}

// =============================================================================
// Get Single CVE Alert Tests
// =============================================================================

func TestGetCVEAlert_Success(t *testing.T) {
	h := testHandler(t)

	alertID := uuid.New().String()

	// Create a router to handle URL params
	r := chi.NewRouter()
	r.Get("/api/v1/cve-alerts/{alertID}", h.getCVEAlert)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts/"+alertID, nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var alert CVEAlert
	err := json.NewDecoder(rec.Body).Decode(&alert)
	require.NoError(t, err)

	assert.Equal(t, alertID, alert.ID)
	assert.NotEmpty(t, alert.CVEID)
	assert.NotEmpty(t, alert.Severity)
	assert.NotEmpty(t, alert.Status)
	assert.NotNil(t, alert.CVEDetails)
}

func TestGetCVEAlert_HasCVEDetails(t *testing.T) {
	h := testHandler(t)

	alertID := uuid.New().String()

	r := chi.NewRouter()
	r.Get("/api/v1/cve-alerts/{alertID}", h.getCVEAlert)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts/"+alertID, nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var alert CVEAlert
	err := json.NewDecoder(rec.Body).Decode(&alert)
	require.NoError(t, err)

	// Verify CVE details are populated
	require.NotNil(t, alert.CVEDetails)
	assert.NotEmpty(t, alert.CVEDetails.CVEID)
	assert.NotEmpty(t, alert.CVEDetails.Severity)
	assert.NotEmpty(t, alert.CVEDetails.PrimarySource)
}

// =============================================================================
// Update CVE Alert Status Tests
// =============================================================================

func TestUpdateCVEAlertStatus_Success(t *testing.T) {
	h := testHandler(t)

	alertID := uuid.New().String()

	r := chi.NewRouter()
	r.Patch("/api/v1/cve-alerts/{alertID}/status", h.updateCVEAlertStatus)

	requestBody := UpdateCVEAlertStatusRequest{
		Status: CVEAlertStatusInvestigating,
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/cve-alerts/"+alertID+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var alert CVEAlert
	err := json.NewDecoder(rec.Body).Decode(&alert)
	require.NoError(t, err)

	assert.Equal(t, CVEAlertStatusInvestigating, alert.Status)
}

func TestUpdateCVEAlertStatus_WithAssignment(t *testing.T) {
	h := testHandler(t)

	alertID := uuid.New().String()

	r := chi.NewRouter()
	r.Patch("/api/v1/cve-alerts/{alertID}/status", h.updateCVEAlertStatus)

	assignee := "security-team@example.com"
	requestBody := UpdateCVEAlertStatusRequest{
		Status:     CVEAlertStatusInProgress,
		AssignedTo: &assignee,
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/cve-alerts/"+alertID+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var alert CVEAlert
	err := json.NewDecoder(rec.Body).Decode(&alert)
	require.NoError(t, err)

	assert.Equal(t, CVEAlertStatusInProgress, alert.Status)
	require.NotNil(t, alert.AssignedTo)
	assert.Equal(t, assignee, *alert.AssignedTo)
}

func TestUpdateCVEAlertStatus_WithResolution(t *testing.T) {
	h := testHandler(t)

	alertID := uuid.New().String()

	r := chi.NewRouter()
	r.Patch("/api/v1/cve-alerts/{alertID}/status", h.updateCVEAlertStatus)

	notes := "Patched via automated rollout"
	requestBody := UpdateCVEAlertStatusRequest{
		Status:          CVEAlertStatusResolved,
		ResolutionNotes: &notes,
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/cve-alerts/"+alertID+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var alert CVEAlert
	err := json.NewDecoder(rec.Body).Decode(&alert)
	require.NoError(t, err)

	assert.Equal(t, CVEAlertStatusResolved, alert.Status)
	require.NotNil(t, alert.ResolutionNotes)
	assert.Equal(t, notes, *alert.ResolutionNotes)
}

func TestUpdateCVEAlertStatus_InvalidBody(t *testing.T) {
	h := testHandler(t)

	alertID := uuid.New().String()

	r := chi.NewRouter()
	r.Patch("/api/v1/cve-alerts/{alertID}/status", h.updateCVEAlertStatus)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/cve-alerts/"+alertID+"/status", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateCVEAlertStatus_AllStatuses(t *testing.T) {
	h := testHandler(t)

	statuses := []CVEAlertStatus{
		CVEAlertStatusNew,
		CVEAlertStatusInvestigating,
		CVEAlertStatusConfirmed,
		CVEAlertStatusInProgress,
		CVEAlertStatusResolved,
		CVEAlertStatusDismissed,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			alertID := uuid.New().String()

			r := chi.NewRouter()
			r.Patch("/api/v1/cve-alerts/{alertID}/status", h.updateCVEAlertStatus)

			requestBody := UpdateCVEAlertStatusRequest{
				Status: status,
			}
			body, _ := json.Marshal(requestBody)

			req := httptest.NewRequest(http.MethodPatch, "/api/v1/cve-alerts/"+alertID+"/status", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)

			var alert CVEAlert
			err := json.NewDecoder(rec.Body).Decode(&alert)
			require.NoError(t, err)

			assert.Equal(t, status, alert.Status)
		})
	}
}

// =============================================================================
// Get Blast Radius Tests
// =============================================================================

func TestGetCVEAlertBlastRadius_Success(t *testing.T) {
	h := testHandler(t)

	alertID := uuid.New().String()

	r := chi.NewRouter()
	r.Get("/api/v1/cve-alerts/{alertID}/blast-radius", h.getCVEAlertBlastRadius)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts/"+alertID+"/blast-radius", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)

	// Verify required fields
	assert.NotEmpty(t, result["cve_id"])
	assert.NotNil(t, result["total_packages"])
	assert.NotNil(t, result["total_images"])
	assert.NotNil(t, result["total_assets"])
	assert.NotNil(t, result["production_assets"])
	assert.NotNil(t, result["affected_platforms"])
	assert.NotNil(t, result["affected_regions"])
	assert.NotNil(t, result["affected_packages"])
	assert.NotNil(t, result["affected_images"])
	assert.NotNil(t, result["affected_assets"])
	assert.NotNil(t, result["urgency_score"])
	assert.NotNil(t, result["calculated_at"])
	assert.Equal(t, alertID, result["alert_id"])
}

func TestGetCVEAlertBlastRadius_ContainsPackages(t *testing.T) {
	h := testHandler(t)

	alertID := uuid.New().String()

	r := chi.NewRouter()
	r.Get("/api/v1/cve-alerts/{alertID}/blast-radius", h.getCVEAlertBlastRadius)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts/"+alertID+"/blast-radius", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)

	packages, ok := result["affected_packages"].([]interface{})
	require.True(t, ok, "affected_packages should be an array")
	require.Greater(t, len(packages), 0, "should have at least one affected package")

	// Check first package has required fields
	pkg, ok := packages[0].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, pkg["package_name"])
	assert.NotEmpty(t, pkg["package_version"])
}

func TestGetCVEAlertBlastRadius_ContainsImages(t *testing.T) {
	h := testHandler(t)

	alertID := uuid.New().String()

	r := chi.NewRouter()
	r.Get("/api/v1/cve-alerts/{alertID}/blast-radius", h.getCVEAlertBlastRadius)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts/"+alertID+"/blast-radius", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)

	images, ok := result["affected_images"].([]interface{})
	require.True(t, ok, "affected_images should be an array")
	require.Greater(t, len(images), 0, "should have at least one affected image")

	// Check first image has required fields
	img, ok := images[0].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, img["image_family"])
	assert.NotEmpty(t, img["image_version"])
	assert.NotNil(t, img["is_direct"])
}

func TestGetCVEAlertBlastRadius_ContainsAssets(t *testing.T) {
	h := testHandler(t)

	alertID := uuid.New().String()

	r := chi.NewRouter()
	r.Get("/api/v1/cve-alerts/{alertID}/blast-radius", h.getCVEAlertBlastRadius)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts/"+alertID+"/blast-radius", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)

	assets, ok := result["affected_assets"].([]interface{})
	require.True(t, ok, "affected_assets should be an array")
	require.Greater(t, len(assets), 0, "should have at least one affected asset")

	// Check first asset has required fields
	asset, ok := assets[0].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, asset["asset_name"])
	assert.NotEmpty(t, asset["platform"])
	assert.NotEmpty(t, asset["region"])
	assert.NotNil(t, asset["is_production"])
}

// =============================================================================
// Create Patch Campaign Tests
// =============================================================================

func TestCreatePatchCampaignFromAlert_Success(t *testing.T) {
	h := testHandler(t)

	alertID := uuid.New().String()

	r := chi.NewRouter()
	r.Post("/api/v1/cve-alerts/{alertID}/create-campaign", h.createPatchCampaignFromAlert)

	requestBody := map[string]interface{}{
		"name":             "CVE-2024-1234 Remediation",
		"description":      "Emergency patch for critical vulnerability",
		"campaign_type":    "cve_response",
		"rollout_strategy": "canary",
		"canary_percentage": 5,
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cve-alerts/"+alertID+"/create-campaign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var result map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)

	assert.NotEmpty(t, result["campaign_id"])
	assert.Equal(t, alertID, result["alert_id"])
	assert.NotEmpty(t, result["message"])
}

func TestCreatePatchCampaignFromAlert_InvalidBody(t *testing.T) {
	h := testHandler(t)

	alertID := uuid.New().String()

	r := chi.NewRouter()
	r.Post("/api/v1/cve-alerts/{alertID}/create-campaign", h.createPatchCampaignFromAlert)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cve-alerts/"+alertID+"/create-campaign", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreatePatchCampaignFromAlert_AllStrategies(t *testing.T) {
	h := testHandler(t)

	strategies := []string{"immediate", "canary", "blue_green", "rolling"}

	for _, strategy := range strategies {
		t.Run(strategy, func(t *testing.T) {
			alertID := uuid.New().String()

			r := chi.NewRouter()
			r.Post("/api/v1/cve-alerts/{alertID}/create-campaign", h.createPatchCampaignFromAlert)

			requestBody := map[string]interface{}{
				"name":             "Test Campaign",
				"campaign_type":    "cve_response",
				"rollout_strategy": strategy,
			}
			body, _ := json.Marshal(requestBody)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/cve-alerts/"+alertID+"/create-campaign", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusCreated, rec.Code)
		})
	}
}

func TestCreatePatchCampaignFromAlert_WithTargetAssets(t *testing.T) {
	h := testHandler(t)

	alertID := uuid.New().String()

	r := chi.NewRouter()
	r.Post("/api/v1/cve-alerts/{alertID}/create-campaign", h.createPatchCampaignFromAlert)

	requestBody := map[string]interface{}{
		"name":             "Targeted Patch Campaign",
		"campaign_type":    "cve_response",
		"rollout_strategy": "rolling",
		"target_asset_ids": []string{
			uuid.New().String(),
			uuid.New().String(),
		},
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cve-alerts/"+alertID+"/create-campaign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var result map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)

	assert.NotEmpty(t, result["campaign_id"])
}

// =============================================================================
// Route Registration Tests
// =============================================================================

func TestRegisterCVEAlertRoutes(t *testing.T) {
	h := testHandler(t)

	r := chi.NewRouter()
	h.RegisterCVEAlertRoutes(r)

	// Test each endpoint exists
	testCases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/cve-alerts"},
		{http.MethodGet, "/cve-alerts/summary"},
		{http.MethodGet, "/cve-alerts/" + uuid.New().String()},
		{http.MethodPatch, "/cve-alerts/" + uuid.New().String() + "/status"},
		{http.MethodGet, "/cve-alerts/" + uuid.New().String() + "/blast-radius"},
		{http.MethodPost, "/cve-alerts/" + uuid.New().String() + "/create-campaign"},
	}

	for _, tc := range testCases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			var body *bytes.Reader
			if tc.method == http.MethodPost || tc.method == http.MethodPatch {
				body = bytes.NewReader([]byte(`{"status": "investigating", "name": "test", "campaign_type": "cve_response", "rollout_strategy": "canary"}`))
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
// Mock Alert Data Tests
// =============================================================================

func TestMockCVEAlerts_HasVariety(t *testing.T) {
	h := testHandler(t)

	alerts := h.generateMockCVEAlerts()

	// Verify we have multiple alerts
	assert.GreaterOrEqual(t, len(alerts), 5, "should have at least 5 mock alerts")

	// Verify variety in severities
	severities := make(map[CVESeverity]bool)
	for _, a := range alerts {
		severities[a.Severity] = true
	}
	assert.GreaterOrEqual(t, len(severities), 3, "should have at least 3 different severity levels")

	// Verify variety in statuses
	statuses := make(map[CVEAlertStatus]bool)
	for _, a := range alerts {
		statuses[a.Status] = true
	}
	assert.GreaterOrEqual(t, len(statuses), 3, "should have at least 3 different statuses")
}

func TestMockCVEAlerts_HasCISAKEV(t *testing.T) {
	h := testHandler(t)

	alerts := h.generateMockCVEAlerts()

	hasKEV := false
	for _, a := range alerts {
		if a.CVEDetails != nil && a.CVEDetails.CISAKEVListed {
			hasKEV = true
			break
		}
	}
	assert.True(t, hasKEV, "should have at least one CISA KEV listed alert")
}

func TestMockCVEAlerts_HasExploit(t *testing.T) {
	h := testHandler(t)

	alerts := h.generateMockCVEAlerts()

	hasExploit := false
	for _, a := range alerts {
		if a.CVEDetails != nil && a.CVEDetails.ExploitAvailable {
			hasExploit = true
			break
		}
	}
	assert.True(t, hasExploit, "should have at least one alert with exploit available")
}

func TestMockCVEAlert_ValidUrgencyScore(t *testing.T) {
	h := testHandler(t)

	alertID := uuid.New().String()
	alert := h.generateMockCVEAlert(alertID)

	assert.GreaterOrEqual(t, alert.UrgencyScore, 0.0, "urgency score should be >= 0")
	assert.LessOrEqual(t, alert.UrgencyScore, 100.0, "urgency score should be <= 100")
}
