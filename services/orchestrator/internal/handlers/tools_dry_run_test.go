// PR #20 / CONN-002 — unit tests for the /dry-run endpoint.
//
// Five tests cover the safety-critical invariants symmetric to PR #19's
// /invoke endpoint:
//
//   1. Read-only tools cannot be dry-run-ed (403).
//   2. Plan-only tools cannot be dry-run-ed (403) — defensive; the
//      distinction matters for the activity-log semantic.
//   3. State-change tools dry-run successfully and the audit row carries
//      the dry_run:true marker so SQL queries can distinguish three kinds
//      (synthetic, real-readonly, dry-run-statechange).
//   4. Unknown tools return 404.
//   5. Tool-side errors still produce audit rows; response is 502.

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/middleware"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
)

// makeDryRunReq builds an httptest request whose URL params + org context
// match what the live route would inject. Mirrors makeInvokeReq from
// tools_invoke_test.go but targets the /dry-run path.
func makeDryRunReq(t *testing.T, toolName, orgID string, body []byte) *http.Request {
	t.Helper()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(http.MethodPost, "/api/v1/ai/tools/"+toolName+"/dry-run", strings.NewReader(string(body)))
	} else {
		req = httptest.NewRequest(http.MethodPost, "/api/v1/ai/tools/"+toolName+"/dry-run", http.NoBody)
	}
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("toolName", toolName)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.OrgIDKey, orgID)
	return req.WithContext(ctx)
}

// TestDryRunTool_RejectsReadOnly — read-only tool gets 403.
func TestDryRunTool_RejectsReadOnly(t *testing.T) {
	h := invokeTestHandler(t)
	orgID := seedInvokeOrg(t, h)
	tool := &fakeInvocableTool{name: "fake_read_only_for_dryrun", risk: tools.RiskReadOnly}
	registerForTest(t, h, tool)

	w := httptest.NewRecorder()
	r := makeDryRunReq(t, "fake_read_only_for_dryrun", orgID, []byte(`{}`))
	h.dryRunTool(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	if tool.executeCalled != 0 {
		t.Errorf("Execute called %d times, want 0", tool.executeCalled)
	}
}

// TestDryRunTool_RejectsPlanOnly — plan-only tool gets 403.
func TestDryRunTool_RejectsPlanOnly(t *testing.T) {
	h := invokeTestHandler(t)
	orgID := seedInvokeOrg(t, h)
	tool := &fakeInvocableTool{name: "fake_plan_only_for_dryrun", risk: tools.RiskPlanOnly}
	registerForTest(t, h, tool)

	w := httptest.NewRecorder()
	r := makeDryRunReq(t, "fake_plan_only_for_dryrun", orgID, []byte(`{}`))
	h.dryRunTool(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (plan-only is not dry-run-able)", w.Code)
	}
}

// TestDryRunTool_AcceptsStateChange_InsertsAuditRow — state-change tool
// runs, audit row exists with risk_level=state_change_prod AND parameters
// contains dry_run:true.
func TestDryRunTool_AcceptsStateChange_InsertsAuditRow(t *testing.T) {
	h := invokeTestHandler(t)
	orgID := seedInvokeOrg(t, h)
	tool := &fakeInvocableTool{
		name: "fake_state_change_for_dryrun",
		risk: tools.RiskStateChangeProd,
		executeFn: func(_ context.Context, p map[string]any) (any, error) {
			// Confirm the endpoint injected dry_run:true into params.
			if p["dry_run"] != true {
				return nil, &fakeError{msg: "endpoint did not inject dry_run:true into params"}
			}
			return map[string]any{"command_plan": map[string]any{"document_name": "TestDoc"}, "dry_run": true}, nil
		},
	}
	registerForTest(t, h, tool)

	w := httptest.NewRecorder()
	r := makeDryRunReq(t, "fake_state_change_for_dryrun", orgID, []byte(`{"params":{}}`))
	h.dryRunTool(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var resp InvokeToolResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.RiskLevel != string(tools.RiskStateChangeProd) {
		t.Errorf("RiskLevel = %q, want state_change_prod", resp.RiskLevel)
	}
	if resp.Simulated {
		t.Errorf("Simulated = true; dry-run is a separate kind from simulator")
	}

	var (
		riskLevel string
		paramsRaw []byte
	)
	if err := h.db.Pool.QueryRow(context.Background(),
		`SELECT risk_level, parameters FROM ai_tool_invocations WHERE id = $1`,
		resp.InvocationID,
	).Scan(&riskLevel, &paramsRaw); err != nil {
		t.Fatalf("query audit row: %v", err)
	}
	if riskLevel != string(tools.RiskStateChangeProd) {
		t.Errorf("audit risk_level = %q, want state_change_prod", riskLevel)
	}

	var params map[string]any
	if err := json.Unmarshal(paramsRaw, &params); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if params["dry_run"] != true {
		t.Errorf("audit params missing dry_run:true marker: %s", string(paramsRaw))
	}
}

// TestDryRunTool_404OnUnknownTool — unknown tool name returns 404.
func TestDryRunTool_404OnUnknownTool(t *testing.T) {
	h := invokeTestHandler(t)
	orgID := seedInvokeOrg(t, h)

	w := httptest.NewRecorder()
	r := makeDryRunReq(t, "definitely_not_a_tool", orgID, []byte(`{}`))
	h.dryRunTool(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// TestDryRunTool_RecordsToolError — a tool whose Execute fails still
// produces an audit row; the response is 502 to distinguish from 500s in
// our own code.
func TestDryRunTool_RecordsToolError(t *testing.T) {
	h := invokeTestHandler(t)
	orgID := seedInvokeOrg(t, h)
	tool := &fakeInvocableTool{
		name: "fake_state_change_err_for_dryrun",
		risk: tools.RiskStateChangeProd,
		executeFn: func(_ context.Context, _ map[string]any) (any, error) {
			return nil, &fakeError{msg: "ssm validation failed: malformed input"}
		},
	}
	registerForTest(t, h, tool)

	w := httptest.NewRecorder()
	r := makeDryRunReq(t, "fake_state_change_err_for_dryrun", orgID, []byte(`{}`))
	h.dryRunTool(w, r)

	if w.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502 for tool-side errors", w.Code)
	}

	var errText string
	if err := h.db.Pool.QueryRow(context.Background(),
		`SELECT COALESCE(error, '') FROM ai_tool_invocations WHERE tool_name = $1 ORDER BY created_at DESC LIMIT 1`,
		tool.name,
	).Scan(&errText); err != nil {
		t.Fatalf("query audit row: %v", err)
	}
	if !strings.Contains(errText, "malformed input") {
		t.Errorf("audit error text = %q, want substring 'malformed input'", errText)
	}
}
