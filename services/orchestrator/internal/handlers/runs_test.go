// Run-detail endpoint tests — PR #16 / UX-001.
//
// Three DB-backed tests covering the org-isolation and audit-log passthrough
// invariants. The handlerTestDB helper skips cleanly if no database is
// available — same pattern as conversations_test.go and approval_simulation_test.go.

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/middleware"
)

// runsTestHandler returns a Handler with a real DB pool and a quiet logger.
// The new run endpoints don't need executor/notifier/temporalWorker, so we
// leave those nil.
func runsTestHandler(t *testing.T) *Handler {
	t.Helper()
	db := handlerTestDB(t)
	t.Cleanup(db.Close)
	return &Handler{
		db:  db,
		cfg: &config.Config{Env: "test", Orchestrator: config.OrchestratorConfig{DevMode: true}},
		log: logger.New("error", "text"),
	}
}

// seedRunFixture inserts one org + user + task + plan + run + N invocations.
// Returns the IDs needed for assertions. Registers cleanup that cascades.
func seedRunFixture(t *testing.T, h *Handler, auditLog string, toolCount int) (orgID, runID string) {
	t.Helper()
	pool := h.db.Pool
	ctx := context.Background()

	orgID = uuid.NewString()
	userID := uuid.NewString()
	taskID := uuid.NewString()
	planID := uuid.NewString()
	runID = uuid.NewString()

	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, "DELETE FROM organizations WHERE id = $1", orgID)
	})

	if _, err := pool.Exec(ctx,
		`INSERT INTO organizations (id, name, slug) VALUES ($1, $2, $3)`,
		orgID, "PR16 test org", "pr16-"+uuid.NewString()[:8]); err != nil {
		t.Fatalf("seed org: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, external_id, email, name, role, org_id)
		 VALUES ($1, $2, $3, 'PR16', 'admin', $4)`,
		userID, "pr16-"+userID[:8], userID[:8]+"@pr16.test", orgID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO ai_tasks (id, org_id, created_by, user_intent, state, source)
		 VALUES ($1, $2, $3, 'pr16 test intent', 'planned', 'chat')`,
		taskID, orgID, userID); err != nil {
		t.Fatalf("seed task: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO ai_plans (id, task_id, type, payload, state)
		 VALUES ($1, $2, 'patch_plan', '{"phases":["canary","monitor","full_rollout"]}'::jsonb, 'approved')`,
		planID, taskID); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO ai_runs (id, plan_id, task_id, environment, initiated_by,
			current_phase, phases_completed, phases_remaining, percent_complete,
			state, audit_log)
		 VALUES ($1, $2, $3, 'staging', $4, '', '[]'::jsonb, '[]'::jsonb, 100, 'completed', $5::jsonb)`,
		runID, planID, taskID, userID, auditLog); err != nil {
		t.Fatalf("seed run: %v", err)
	}

	for i := range toolCount {
		if _, err := pool.Exec(ctx, `
			INSERT INTO ai_tool_invocations (task_id, plan_id, run_id, tool_name, risk_level, duration_ms, parameters, result)
			VALUES ($1, $2, $3, $4, 'plan_only', $5, '{}'::jsonb, '{}'::jsonb)`,
			taskID, planID, runID, "test_tool_"+uuid.NewString()[:6], 250+i*10,
		); err != nil {
			t.Fatalf("seed invocation: %v", err)
		}
	}
	return orgID, runID
}

// requestWithOrg builds an httptest request whose context carries the given
// org ID — bypasses the auth middleware so we can exercise the handler
// directly with the org-scoped query.
func requestWithOrg(t *testing.T, method, path, orgID string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, http.NoBody)
	ctx := context.WithValue(req.Context(), middleware.OrgIDKey, orgID)
	return req.WithContext(ctx)
}

// TestListRuns_ScopesToOrgAndReturnsRecent — seed two orgs each with one
// run; call listRuns scoped to org A; assert exactly one run returned.
func TestListRuns_ScopesToOrgAndReturnsRecent(t *testing.T) {
	h := runsTestHandler(t)
	auditA := `[{"kind":"approved","_simulated":true,"ts":"2026-06-01T10:00:00Z"}]`
	auditB := `[{"kind":"approved","_simulated":true,"ts":"2026-06-01T10:01:00Z"}]`
	orgA, runA := seedRunFixture(t, h, auditA, 0)
	_, _ = seedRunFixture(t, h, auditB, 0) // different org

	w := httptest.NewRecorder()
	r := requestWithOrg(t, http.MethodGet, "/api/v1/ai/runs", orgA)
	h.listRuns(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var resp struct {
		Runs []RunSummary `json:"runs"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Runs) != 1 {
		t.Fatalf("runs count = %d, want 1 (org isolation); got: %+v", len(resp.Runs), resp.Runs)
	}
	if resp.Runs[0].ID != runA {
		t.Errorf("returned run = %s, want %s", resp.Runs[0].ID, runA)
	}
	if !resp.Runs[0].Simulated {
		t.Errorf("simulated flag = false, want true (audit_log has _simulated)")
	}
}

// TestGetRun_ReturnsAuditLogAndToolInvocations — verify the detail endpoint
// returns the full audit_log as an array of JSON entries and the joined
// tool invocations.
func TestGetRun_ReturnsAuditLogAndToolInvocations(t *testing.T) {
	h := runsTestHandler(t)
	audit := `[
		{"kind":"approved","_simulated":true,"ts":"2026-06-01T10:00:00Z"},
		{"kind":"started","_simulated":true,"ts":"2026-06-01T10:00:01Z"},
		{"kind":"phase_complete","phase":"canary","tool":"generate_patch_plan","_simulated":true,"ts":"2026-06-01T10:00:02Z"},
		{"kind":"simulated_complete","_simulated":true,"ts":"2026-06-01T10:00:03Z"}
	]`
	orgID, runID := seedRunFixture(t, h, audit, 2)

	w := httptest.NewRecorder()
	r := requestWithOrg(t, http.MethodGet, "/api/v1/ai/runs/"+runID, orgID)
	// chi.URLParam reads from a route context; install one.
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("runID", runID)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	h.getRun(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var resp struct {
		Run             RunDetail           `json:"run"`
		ToolInvocations []RunToolInvocation `json:"tool_invocations"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Run.AuditLog) != 4 {
		t.Errorf("audit_log entries = %d, want 4", len(resp.Run.AuditLog))
	}
	// Spot-check that the first entry is approved with _simulated:true.
	var first map[string]any
	if err := json.Unmarshal(resp.Run.AuditLog[0], &first); err != nil {
		t.Fatalf("first audit entry unmarshal: %v", err)
	}
	if first["kind"] != "approved" {
		t.Errorf("first entry kind = %v, want approved", first["kind"])
	}
	if first["_simulated"] != true {
		t.Errorf("first entry missing _simulated:true marker")
	}
	if len(resp.ToolInvocations) != 2 {
		t.Errorf("tool_invocations = %d, want 2", len(resp.ToolInvocations))
	}
	for _, inv := range resp.ToolInvocations {
		if inv.RiskLevel != "plan_only" && inv.RiskLevel != "read_only" {
			t.Errorf("invocation %s has risk %s, want plan_only/read_only", inv.ID, inv.RiskLevel)
		}
	}
}

// TestGetRun_404OnOrgMismatch — a run seeded under org A returns 404 when
// queried with org B's context.
func TestGetRun_404OnOrgMismatch(t *testing.T) {
	h := runsTestHandler(t)
	_, runID := seedRunFixture(t, h, `[{"kind":"approved","_simulated":true}]`, 0)
	otherOrg := uuid.NewString()

	w := httptest.NewRecorder()
	r := requestWithOrg(t, http.MethodGet, "/api/v1/ai/runs/"+runID, otherOrg)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("runID", runID)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	h.getRun(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (cross-org access); body = %s", w.Code, w.Body.String())
	}
}
