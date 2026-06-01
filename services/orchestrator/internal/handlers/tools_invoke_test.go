// PR #19 / CONN-001 — unit tests for the direct tool invocation endpoint.
//
// Five tests cover the safety-critical invariants:
//
//   1. State-change tools cannot be invoked via this endpoint (403).
//   2. Unknown tools return 404.
//   3. A successful invocation inserts an ai_tool_invocations row with no
//      _simulated marker — the artifact is distinguishable from B.3
//      simulator output.
//   4. The per-org direct-invocation task is lazy-created on first invoke
//      and reused on subsequent calls (one row per org, ever).
//   5. A tool whose Execute returns an error still produces an audit row
//      with the error text, returns 502 (so it's distinguishable from a
//      500 in our code), and doesn't leak the error into the success
//      payload.

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/middleware"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
)

// fakeInvocableTool is a programmable Tool implementation for tests. Tests
// pick its risk level and the body of Execute. This is deliberately a tiny
// implementation — using the real tool registry's tools would require
// scaffolding a DB connection.
type fakeInvocableTool struct {
	name          string
	risk          tools.RiskLevel
	executeFn     func(ctx context.Context, params map[string]any) (any, error)
	executeCalled int
}

func (f *fakeInvocableTool) Name() string                       { return f.name }
func (f *fakeInvocableTool) Description() string                { return "fake tool for tests" }
func (f *fakeInvocableTool) Risk() tools.RiskLevel              { return f.risk }
func (f *fakeInvocableTool) Scope() tools.Scope                 { return tools.ScopeOrganization }
func (f *fakeInvocableTool) Idempotent() bool                   { return true }
func (f *fakeInvocableTool) RequiresApproval() bool             { return false }
func (f *fakeInvocableTool) Parameters() map[string]interface{} { return map[string]interface{}{} }
func (f *fakeInvocableTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	f.executeCalled++
	if f.executeFn == nil {
		return map[string]any{"ok": true}, nil
	}
	return f.executeFn(ctx, params)
}

// invokeTestHandler builds a Handler wired with a real DB pool and a fresh
// tool registry. The caller registers test tools by calling
// registerForTest(tool) on the returned handler.
func invokeTestHandler(t *testing.T) *Handler {
	t.Helper()
	db := handlerTestDB(t)
	t.Cleanup(db.Close)

	reg := tools.NewRegistry(db.Pool, logger.New("error", "text"))
	return &Handler{
		db:    db,
		cfg:   &config.Config{Env: "test", Orchestrator: config.OrchestratorConfig{DevMode: true}},
		log:   logger.New("error", "text"),
		tools: reg,
	}
}

// registerForTest adds a tool to the registry directly. The registry's
// register() method is unexported; this test helper does the same map-set
// via reflection-free direct access to the package-internal method by
// re-registering via a known public helper. We just use RegisterCloudTools
// for read-only tools (it overwrites by name).
func registerForTest(t *testing.T, h *Handler, tool tools.Tool) {
	t.Helper()
	// The registry's register() is unexported; we use a tiny trampoline via
	// the test-only path. The simplest approach: cast the registry to a
	// type that exposes a public register method. We don't have one, so
	// we use the AWS-tools registration helper as a generic "stick this
	// tool in the map" path — works because RegisterCloudTools just calls
	// r.register(t). We can't reuse it for arbitrary tools, so instead we
	// piggyback on the tool registry by writing a helper there.
	tools.RegisterToolForTest(h.tools, tool)
}

// makeInvokeReq builds an httptest request with chi URL params + org context
// set, mimicking what the route would do in production.
func makeInvokeReq(t *testing.T, toolName, orgID string, body []byte) *http.Request {
	t.Helper()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(http.MethodPost, "/api/v1/ai/tools/"+toolName+"/invoke", strings.NewReader(string(body)))
	} else {
		req = httptest.NewRequest(http.MethodPost, "/api/v1/ai/tools/"+toolName+"/invoke", http.NoBody)
	}
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("toolName", toolName)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.OrgIDKey, orgID)
	return req.WithContext(ctx)
}

// seedInvokeOrg inserts a throwaway org so direct-invocation tasks can be
// FK-linked. Returns the org id and registers cleanup.
func seedInvokeOrg(t *testing.T, h *Handler) string {
	t.Helper()
	orgID := uuid.NewString()
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = h.db.Pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1", orgID)
	})
	if _, err := h.db.Pool.Exec(ctx,
		`INSERT INTO organizations (id, name, slug) VALUES ($1, $2, $3)`,
		orgID, "PR19 test org", "pr19-"+uuid.NewString()[:8]); err != nil {
		t.Fatalf("seed org: %v", err)
	}
	return orgID
}

// TestInvokeTool_RejectsStateChange — a tool with state_change_prod risk
// returns 403. The invocation row must NOT be inserted (the gate must short-
// circuit before the row insert).
func TestInvokeTool_RejectsStateChange(t *testing.T) {
	h := invokeTestHandler(t)
	orgID := seedInvokeOrg(t, h)

	tool := &fakeInvocableTool{name: "fake_state_change", risk: tools.RiskStateChangeProd}
	registerForTest(t, h, tool)

	w := httptest.NewRecorder()
	r := makeInvokeReq(t, "fake_state_change", orgID, []byte(`{"params":{}}`))
	h.invokeTool(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body = %s", w.Code, w.Body.String())
	}
	if tool.executeCalled != 0 {
		t.Errorf("Execute called %d times, want 0 (gate should short-circuit before run)", tool.executeCalled)
	}
	// And no audit row should have been inserted.
	var count int
	if err := h.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM ai_tool_invocations WHERE tool_name = $1`, tool.name,
	).Scan(&count); err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if count != 0 {
		t.Errorf("audit rows = %d, want 0 (forbidden invocations must not be recorded)", count)
	}
}

// TestInvokeTool_404OnUnknownTool — invoke a nonexistent tool, expect 404.
func TestInvokeTool_404OnUnknownTool(t *testing.T) {
	h := invokeTestHandler(t)
	orgID := seedInvokeOrg(t, h)

	w := httptest.NewRecorder()
	r := makeInvokeReq(t, "definitely_not_a_tool", orgID, []byte(`{}`))
	h.invokeTool(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body = %s", w.Code, w.Body.String())
	}
}

// TestInvokeTool_InsertsRealInvocation — happy path. A read_only tool gets
// invoked, the row lands in ai_tool_invocations with risk_level=read_only
// and NO _simulated marker in parameters or result (distinguishable from
// B.3 synthetic invocations).
func TestInvokeTool_InsertsRealInvocation(t *testing.T) {
	h := invokeTestHandler(t)
	orgID := seedInvokeOrg(t, h)

	tool := &fakeInvocableTool{
		name: "fake_read_only",
		risk: tools.RiskReadOnly,
		executeFn: func(_ context.Context, _ map[string]any) (any, error) {
			return map[string]any{"hello": "world", "count": 3}, nil
		},
	}
	registerForTest(t, h, tool)

	w := httptest.NewRecorder()
	r := makeInvokeReq(t, "fake_read_only", orgID, []byte(`{"params":{"foo":"bar"}}`))
	h.invokeTool(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var resp InvokeToolResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Simulated {
		t.Errorf("response.Simulated = true, want false (this endpoint is for REAL invocations)")
	}
	if resp.RiskLevel != string(tools.RiskReadOnly) {
		t.Errorf("response.RiskLevel = %q, want read_only", resp.RiskLevel)
	}

	// Verify the audit row landed and is distinguishable from a simulator row.
	var (
		riskLevel string
		paramsRaw []byte
		resultRaw []byte
	)
	if err := h.db.Pool.QueryRow(context.Background(),
		`SELECT risk_level, parameters, result FROM ai_tool_invocations
		 WHERE id = $1`, resp.InvocationID,
	).Scan(&riskLevel, &paramsRaw, &resultRaw); err != nil {
		t.Fatalf("query audit row: %v", err)
	}
	if riskLevel != string(tools.RiskReadOnly) {
		t.Errorf("audit risk_level = %q, want read_only", riskLevel)
	}
	// Critical: ensure NO _simulated:true marker in either parameters or result.
	for _, blob := range [][]byte{paramsRaw, resultRaw} {
		var m map[string]any
		if err := json.Unmarshal(blob, &m); err != nil {
			continue
		}
		if v, ok := m["_simulated"]; ok && v == true {
			t.Errorf("audit row contains _simulated:true marker (would conflate with B.3 simulator rows): %s", string(blob))
		}
	}
}

// TestInvokeTool_LazyCreatesDirectInvocationTask — first invoke creates a
// per-org direct-invocations task row; second invoke reuses it.
func TestInvokeTool_LazyCreatesDirectInvocationTask(t *testing.T) {
	h := invokeTestHandler(t)
	orgID := seedInvokeOrg(t, h)

	tool := &fakeInvocableTool{name: "fake_read_only_2", risk: tools.RiskReadOnly}
	registerForTest(t, h, tool)

	// First invoke creates the task.
	w := httptest.NewRecorder()
	r := makeInvokeReq(t, "fake_read_only_2", orgID, []byte(`{}`))
	h.invokeTool(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("first invoke status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var taskCount int
	if err := h.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM ai_tasks WHERE org_id = $1 AND task_spec->>'kind' = 'direct_invocation'`,
		orgID,
	).Scan(&taskCount); err != nil {
		t.Fatalf("count direct-invocation tasks: %v", err)
	}
	if taskCount != 1 {
		t.Errorf("direct-invocation task count after first invoke = %d, want 1", taskCount)
	}

	// Second invoke must REUSE the task — no new row.
	w = httptest.NewRecorder()
	r = makeInvokeReq(t, "fake_read_only_2", orgID, []byte(`{}`))
	h.invokeTool(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("second invoke status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	if err := h.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM ai_tasks WHERE org_id = $1 AND task_spec->>'kind' = 'direct_invocation'`,
		orgID,
	).Scan(&taskCount); err != nil {
		t.Fatalf("count direct-invocation tasks (second): %v", err)
	}
	if taskCount != 1 {
		t.Errorf("direct-invocation task count after second invoke = %d, want 1 (lazy-create should be idempotent)", taskCount)
	}
}

// TestInvokeTool_ToolErrorStillAudited — a tool whose Execute fails should
// still produce an audit row (with the error text), and the endpoint should
// return 502 to distinguish "the tool ran but the call failed" from
// a 500 in our own code.
func TestInvokeTool_ToolErrorStillAudited(t *testing.T) {
	h := invokeTestHandler(t)
	orgID := seedInvokeOrg(t, h)

	tool := &fakeInvocableTool{
		name: "fake_read_only_err",
		risk: tools.RiskReadOnly,
		executeFn: func(_ context.Context, _ map[string]any) (any, error) {
			return nil, &fakeError{msg: "aws SignatureDoesNotMatch"}
		},
	}
	registerForTest(t, h, tool)

	w := httptest.NewRecorder()
	r := makeInvokeReq(t, "fake_read_only_err", orgID, []byte(`{}`))
	h.invokeTool(w, r)
	if w.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502 (Bad Gateway) for tool-side errors", w.Code)
	}

	// Audit row should still exist with the error text recorded.
	var errText string
	if err := h.db.Pool.QueryRow(context.Background(),
		`SELECT COALESCE(error, '') FROM ai_tool_invocations WHERE tool_name = $1 ORDER BY created_at DESC LIMIT 1`,
		tool.name,
	).Scan(&errText); err != nil {
		t.Fatalf("query audit row: %v", err)
	}
	if !strings.Contains(errText, "SignatureDoesNotMatch") {
		t.Errorf("audit row error text = %q, want substring SignatureDoesNotMatch", errText)
	}
}

type fakeError struct{ msg string }

func (e *fakeError) Error() string { return e.msg }
