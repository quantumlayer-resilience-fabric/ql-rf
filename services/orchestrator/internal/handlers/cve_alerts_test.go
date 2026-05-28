// Package handlers provides HTTP handlers for the AI orchestrator service.
package handlers

import (
	"bytes"
	"context"
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

// testHandler creates a DB-less handler for exercising handler paths that return
// before any database access (request validation and the mock create-campaign
// endpoint).
func testHandler(t *testing.T) *Handler {
	t.Helper()
	return &Handler{
		cfg: &config.Config{
			Env:          "test",
			Orchestrator: config.OrchestratorConfig{DevMode: true},
		},
		log: logger.New("error", "text"),
	}
}

// dbTestHandler creates a handler wired to a real database, skipping the test
// when none is available. The CVE alert read/write endpoints are DB-backed (see
// cve_alerts_query.go), so HTTP-level coverage needs a live database.
func dbTestHandler(t *testing.T) *Handler {
	t.Helper()
	db := handlerTestDB(t)
	t.Cleanup(db.Close)
	return &Handler{
		db:  db,
		cfg: &config.Config{Env: "test", Orchestrator: config.OrchestratorConfig{DevMode: true}},
		log: logger.New("error", "text"),
	}
}

// seededAlert is the fixture inserted by seedCVEAlert.
type seededAlert struct {
	OrgID   string
	AlertID uuid.UUID
	CVEID   string
}

// seedCVEAlert inserts one org with a critical CVE alert plus blast-radius items
// (one package, one image) and registers cleanup. It mirrors the data shape the
// live vulnmatch pipeline produces so the HTTP tests exercise realistic rows.
func seedCVEAlert(t *testing.T, h *Handler) seededAlert {
	t.Helper()
	pool := h.db.Pool
	ctx := context.Background()

	suffix := uuid.NewString()[:8]
	orgID := uuid.New()
	imageID := uuid.New()
	sbomID := uuid.New()
	cveID := "CVE-TEST-" + suffix

	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, "DELETE FROM organizations WHERE id = $1", orgID)
		_, _ = pool.Exec(bg, "DELETE FROM cve_cache WHERE cve_id = $1", cveID)
	})

	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("seed exec failed (%s): %v", sql, err)
		}
	}

	exec("INSERT INTO organizations (id, name, slug) VALUES ($1, $2, $3)", orgID, "h-org-"+suffix, "h-org-"+suffix)
	exec("INSERT INTO images (id, org_id, family, version) VALUES ($1, $2, $3, '1.0.0')", imageID, orgID, "h-fam-"+suffix)
	exec("INSERT INTO sboms (id, image_id, org_id, format, version, content) VALUES ($1, $2, $3, 'spdx', 'SPDX-2.3', '{}')", sbomID, imageID, orgID)

	var pkgID uuid.UUID
	if err := pool.QueryRow(ctx,
		"INSERT INTO sbom_packages (sbom_id, name, version, type) VALUES ($1, 'openssl', '3.0.0', 'deb') RETURNING id",
		sbomID,
	).Scan(&pkgID); err != nil {
		t.Fatalf("seed sbom_packages: %v", err)
	}

	var cveCacheID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO cve_cache (cve_id, severity, cvss_v3_score, exploit_available, cisa_kev_listed, primary_source, sources)
		 VALUES ($1, 'critical', 9.8, true, true, 'nvd', '[]') RETURNING id`,
		cveID,
	).Scan(&cveCacheID); err != nil {
		t.Fatalf("seed cve_cache: %v", err)
	}

	var alertID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO cve_alerts (org_id, cve_cache_id, cve_id, severity, urgency_score, status,
		   affected_packages_count, affected_images_count, affected_assets_count, production_assets_count)
		 VALUES ($1, $2, $3, 'critical', 90, 'new', 1, 1, 0, 0) RETURNING id`,
		orgID, cveCacheID, cveID,
	).Scan(&alertID); err != nil {
		t.Fatalf("seed cve_alerts: %v", err)
	}

	exec(`INSERT INTO cve_alert_affected_items (alert_id, package_id, item_type, package_name, package_version, package_type, lineage_depth, item_status)
	      VALUES ($1, $2, 'package', 'openssl', '3.0.0', 'deb', 0, 'vulnerable')`, alertID, pkgID)
	exec(`INSERT INTO cve_alert_affected_items (alert_id, image_id, item_type, image_family, image_version, lineage_depth, item_status)
	      VALUES ($1, $2, 'image', $3, '1.0.0', 0, 'vulnerable')`, alertID, imageID, "h-fam-"+suffix)

	return seededAlert{OrgID: orgID.String(), AlertID: alertID, CVEID: cveID}
}

// =============================================================================
// List CVE Alerts Tests
// =============================================================================

func TestListCVEAlerts_Success(t *testing.T) {
	h := dbTestHandler(t)
	s := seedCVEAlert(t, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts?org_id="+s.OrgID, nil)
	rec := httptest.NewRecorder()

	h.listCVEAlerts(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response CVEAlertListResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))

	require.Equal(t, 1, response.Total, "org should have exactly one seeded alert")
	require.Len(t, response.Alerts, 1)
	assert.Equal(t, s.CVEID, response.Alerts[0].CVEID)
	assert.Equal(t, 1, response.Page)
	assert.Equal(t, 50, response.PageSize)
}

func TestListCVEAlerts_PaginationEcho(t *testing.T) {
	h := dbTestHandler(t)
	s := seedCVEAlert(t, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts?org_id="+s.OrgID+"&page=1&page_size=2", nil)
	rec := httptest.NewRecorder()

	h.listCVEAlerts(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var response CVEAlertListResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
	assert.Equal(t, 1, response.Page)
	assert.Equal(t, 2, response.PageSize)
	assert.LessOrEqual(t, len(response.Alerts), 2)
}

func TestListCVEAlerts_FilterBySeverity(t *testing.T) {
	h := dbTestHandler(t)
	s := seedCVEAlert(t, h)

	// Matching severity returns the seeded alert; a non-matching one excludes it.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts?org_id="+s.OrgID+"&severity=critical", nil)
	rec := httptest.NewRecorder()
	h.listCVEAlerts(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var crit CVEAlertListResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&crit))
	assert.Equal(t, 1, crit.Total)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts?org_id="+s.OrgID+"&severity=low", nil)
	rec = httptest.NewRecorder()
	h.listCVEAlerts(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var low CVEAlertListResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&low))
	assert.Equal(t, 0, low.Total)
}

func TestListCVEAlerts_InvalidPaginationDefaults(t *testing.T) {
	h := dbTestHandler(t)
	s := seedCVEAlert(t, h)

	// page=-1 and page_size=200 are invalid; handler must fall back to defaults.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts?org_id="+s.OrgID+"&page=-1&page_size=200", nil)
	rec := httptest.NewRecorder()
	h.listCVEAlerts(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var response CVEAlertListResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
	assert.Equal(t, 1, response.Page, "invalid page should default to 1")
	assert.Equal(t, 50, response.PageSize, "page size > 100 should fall back to 50")
}

// =============================================================================
// Get CVE Alert Summary Tests
// =============================================================================

func TestGetCVEAlertSummary_Success(t *testing.T) {
	h := dbTestHandler(t)
	s := seedCVEAlert(t, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts/summary?org_id="+s.OrgID, nil)
	rec := httptest.NewRecorder()

	h.getCVEAlertSummary(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var summary CVEAlertSummary
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&summary))

	assert.Equal(t, 1, summary.TotalAlerts)
	assert.Equal(t, 1, summary.NewAlerts)
	assert.Equal(t, 1, summary.CriticalAlerts)
	assert.Equal(t, 1, summary.ExploitableAlerts)
	assert.Equal(t, 1, summary.CISAKEVAlerts)
}

// =============================================================================
// Get Single CVE Alert Tests
// =============================================================================

func TestGetCVEAlert_Success(t *testing.T) {
	h := dbTestHandler(t)
	s := seedCVEAlert(t, h)

	r := chi.NewRouter()
	r.Get("/api/v1/cve-alerts/{alertID}", h.getCVEAlert)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts/"+s.AlertID.String()+"?org_id="+s.OrgID, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var alert CVEAlert
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&alert))

	assert.Equal(t, s.AlertID.String(), alert.ID)
	assert.Equal(t, s.CVEID, alert.CVEID)
	require.NotNil(t, alert.CVEDetails)
	assert.True(t, alert.CVEDetails.ExploitAvailable)
	assert.True(t, alert.CVEDetails.CISAKEVListed)
}

func TestGetCVEAlert_NotFound(t *testing.T) {
	h := dbTestHandler(t)
	s := seedCVEAlert(t, h)

	r := chi.NewRouter()
	r.Get("/api/v1/cve-alerts/{alertID}", h.getCVEAlert)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts/"+uuid.NewString()+"?org_id="+s.OrgID, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetCVEAlert_InvalidID(t *testing.T) {
	h := testHandler(t)

	r := chi.NewRouter()
	r.Get("/api/v1/cve-alerts/{alertID}", h.getCVEAlert)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// =============================================================================
// Update CVE Alert Status Tests
// =============================================================================

func TestUpdateCVEAlertStatus_Success(t *testing.T) {
	h := dbTestHandler(t)
	s := seedCVEAlert(t, h)

	r := chi.NewRouter()
	r.Patch("/api/v1/cve-alerts/{alertID}/status", h.updateCVEAlertStatus)

	assignee := "security-team@example.com"
	body, _ := json.Marshal(UpdateCVEAlertStatusRequest{
		Status:     CVEAlertStatusInProgress,
		AssignedTo: &assignee,
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/cve-alerts/"+s.AlertID.String()+"/status?org_id="+s.OrgID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var alert CVEAlert
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&alert))
	assert.Equal(t, CVEAlertStatusInProgress, alert.Status)
}

func TestUpdateCVEAlertStatus_ResolvedSetsResolvedAt(t *testing.T) {
	h := dbTestHandler(t)
	s := seedCVEAlert(t, h)

	r := chi.NewRouter()
	r.Patch("/api/v1/cve-alerts/{alertID}/status", h.updateCVEAlertStatus)

	body, _ := json.Marshal(UpdateCVEAlertStatusRequest{Status: CVEAlertStatusResolved})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/cve-alerts/"+s.AlertID.String()+"/status?org_id="+s.OrgID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var alert CVEAlert
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&alert))
	assert.Equal(t, CVEAlertStatusResolved, alert.Status)
	require.NotNil(t, alert.ResolvedAt, "resolved status should populate resolved_at")
}

func TestUpdateCVEAlertStatus_NotFound(t *testing.T) {
	h := dbTestHandler(t)
	s := seedCVEAlert(t, h)

	r := chi.NewRouter()
	r.Patch("/api/v1/cve-alerts/{alertID}/status", h.updateCVEAlertStatus)

	body, _ := json.Marshal(UpdateCVEAlertStatusRequest{Status: CVEAlertStatusInProgress})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/cve-alerts/"+uuid.NewString()+"/status?org_id="+s.OrgID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateCVEAlertStatus_InvalidBody(t *testing.T) {
	h := testHandler(t)

	r := chi.NewRouter()
	r.Patch("/api/v1/cve-alerts/{alertID}/status", h.updateCVEAlertStatus)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/cve-alerts/"+uuid.NewString()+"/status", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateCVEAlertStatus_MissingStatus(t *testing.T) {
	h := testHandler(t)

	r := chi.NewRouter()
	r.Patch("/api/v1/cve-alerts/{alertID}/status", h.updateCVEAlertStatus)

	body, _ := json.Marshal(UpdateCVEAlertStatusRequest{})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/cve-alerts/"+uuid.NewString()+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// =============================================================================
// Get Blast Radius Tests
// =============================================================================

func TestGetCVEAlertBlastRadius_Success(t *testing.T) {
	h := dbTestHandler(t)
	s := seedCVEAlert(t, h)

	r := chi.NewRouter()
	r.Get("/api/v1/cve-alerts/{alertID}/blast-radius", h.getCVEAlertBlastRadius)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts/"+s.AlertID.String()+"/blast-radius?org_id="+s.OrgID, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&result))

	assert.Equal(t, s.AlertID.String(), result["alert_id"])
	assert.Equal(t, s.CVEID, result["cve_id"])

	packages, ok := result["affected_packages"].([]any)
	require.True(t, ok, "affected_packages should be an array")
	require.Len(t, packages, 1)
	pkg, ok := packages[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "openssl", pkg["package_name"])

	images, ok := result["affected_images"].([]any)
	require.True(t, ok, "affected_images should be an array")
	require.Len(t, images, 1)
}

func TestGetCVEAlertBlastRadius_NotFound(t *testing.T) {
	h := dbTestHandler(t)
	s := seedCVEAlert(t, h)

	r := chi.NewRouter()
	r.Get("/api/v1/cve-alerts/{alertID}/blast-radius", h.getCVEAlertBlastRadius)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cve-alerts/"+uuid.NewString()+"/blast-radius?org_id="+s.OrgID, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// =============================================================================
// Create Patch Campaign Tests (handler returns a mock response, no DB access)
// =============================================================================

func TestCreatePatchCampaignFromAlert_Success(t *testing.T) {
	h := testHandler(t)

	alertID := uuid.New().String()
	r := chi.NewRouter()
	r.Post("/api/v1/cve-alerts/{alertID}/create-campaign", h.createPatchCampaignFromAlert)

	body, _ := json.Marshal(map[string]any{
		"name":              "CVE-2024-1234 Remediation",
		"campaign_type":     "cve_response",
		"rollout_strategy":  "canary",
		"canary_percentage": 5,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cve-alerts/"+alertID+"/create-campaign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.NotEmpty(t, result["campaign_id"])
	assert.Equal(t, alertID, result["alert_id"])
	assert.NotEmpty(t, result["message"])
}

func TestCreatePatchCampaignFromAlert_InvalidBody(t *testing.T) {
	h := testHandler(t)

	r := chi.NewRouter()
	r.Post("/api/v1/cve-alerts/{alertID}/create-campaign", h.createPatchCampaignFromAlert)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cve-alerts/"+uuid.NewString()+"/create-campaign", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// =============================================================================
// Route Registration Tests
// =============================================================================

func TestRegisterCVEAlertRoutes(t *testing.T) {
	h := dbTestHandler(t)
	s := seedCVEAlert(t, h)

	r := chi.NewRouter()
	h.RegisterCVEAlertRoutes(r)

	testCases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/cve-alerts?org_id=" + s.OrgID},
		{http.MethodGet, "/cve-alerts/summary?org_id=" + s.OrgID},
		{http.MethodGet, "/cve-alerts/" + s.AlertID.String() + "?org_id=" + s.OrgID},
		{http.MethodPatch, "/cve-alerts/" + s.AlertID.String() + "/status?org_id=" + s.OrgID},
		{http.MethodGet, "/cve-alerts/" + s.AlertID.String() + "/blast-radius?org_id=" + s.OrgID},
		{http.MethodPost, "/cve-alerts/" + s.AlertID.String() + "/create-campaign"},
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

			assert.NotEqual(t, http.StatusNotFound, rec.Code, "route should exist")
			assert.NotEqual(t, http.StatusMethodNotAllowed, rec.Code, "method should be allowed")
		})
	}
}

// =============================================================================
// Mock Alert Data Tests (legacy generators retained as local fixtures)
// =============================================================================

func TestMockCVEAlerts_HasVariety(t *testing.T) {
	h := testHandler(t)

	alerts := h.generateMockCVEAlerts()
	assert.GreaterOrEqual(t, len(alerts), 5, "should have at least 5 mock alerts")

	severities := make(map[CVESeverity]bool)
	statuses := make(map[CVEAlertStatus]bool)
	for _, a := range alerts {
		severities[a.Severity] = true
		statuses[a.Status] = true
	}
	assert.GreaterOrEqual(t, len(severities), 3, "should have at least 3 different severity levels")
	assert.GreaterOrEqual(t, len(statuses), 3, "should have at least 3 different statuses")
}

func TestMockCVEAlert_ValidUrgencyScore(t *testing.T) {
	h := testHandler(t)

	alert := h.generateMockCVEAlert(uuid.New().String())
	assert.GreaterOrEqual(t, alert.UrgencyScore, 0.0, "urgency score should be >= 0")
	assert.LessOrEqual(t, alert.UrgencyScore, 100.0, "urgency score should be <= 100")
}
