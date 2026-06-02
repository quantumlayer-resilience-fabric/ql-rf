//nolint:errcheck // tests intentionally skip checking decode/scan/cleanup errors

// PR #21 / CONN-003 — unit tests for the two-approver workflow.
//
// Five tests cover the safety-critical invariants:
//
//   1. approveTask on a state_change_prod plan transitions to
//      awaiting-second-approval shape (state stays 'awaiting_approval'
//      in DB, but approved_by is set and second_approver is NULL).
//   2. coApproveTask rejects callers that match approved_by (self-second).
//   3. coApproveTask rejects plans that don't require two approvers (wrong
//      endpoint).
//   4. coApproveTask happy path transitions to 'approved' with both
//      approver columns set.
//   5. planRequiresTwoApprovers correctly identifies plans with
//      state_change_prod tools embedded in the payload.
//
// All tests use a real DB (skip if unavailable) for the SQL-bound
// assertions. The registry uses test-only fake tools to avoid pulling in
// the full registry's DB dependencies.

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

	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/middleware"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
)

// seedTwoApproverTask builds an org + two users + ai_task + ai_plan with
// a payload that references a state_change_prod tool. Returns the task
// id, the two user ids, and the registered tool (so the test can also
// remove it). Cleans up via t.Cleanup.
//
// The errcheck violations on the cleanup Exec calls are intentional:
// cleanup runs in t.Cleanup, where Fatalf is unsafe and there's no
// meaningful recovery for "delete from organizations failed" anyway.
//
//nolint:errcheck // test cleanup intentionally ignores DB errors
func seedTwoApproverTask(t *testing.T, h *Handler) (taskID, firstUserID, secondUserID string) {
	t.Helper()
	ctx := context.Background()
	orgID := uuid.NewString()
	firstUserID = uuid.NewString()
	secondUserID = uuid.NewString()
	taskID = uuid.NewString()

	t.Cleanup(func() {
		_, _ = h.db.Pool.Exec(context.Background(), "DELETE FROM ai_plans WHERE task_id = $1", taskID)
		_, _ = h.db.Pool.Exec(context.Background(), "DELETE FROM ai_tasks WHERE id = $1", taskID)
		_, _ = h.db.Pool.Exec(context.Background(), "DELETE FROM users WHERE id IN ($1, $2)", firstUserID, secondUserID)
		_, _ = h.db.Pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1", orgID)
	})

	if _, err := h.db.Pool.Exec(ctx,
		`INSERT INTO organizations (id, name, slug) VALUES ($1, $2, $3)`,
		orgID, "PR21 two-approver test org", "pr21-"+uuid.NewString()[:8]); err != nil {
		t.Fatalf("seed org: %v", err)
	}
	for _, uid := range []string{firstUserID, secondUserID} {
		if _, err := h.db.Pool.Exec(ctx,
			`INSERT INTO users (id, org_id, external_id, email, name) VALUES ($1, $2, $3, $4, $5)`,
			uid, orgID, "ext-"+uid[:8], uid[:8]+"@example.test", "Test User "+uid[:8]); err != nil {
			t.Fatalf("seed user %s: %v", uid, err)
		}
	}
	if _, err := h.db.Pool.Exec(ctx,
		`INSERT INTO ai_tasks (id, org_id, created_by, user_intent, state, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW(), NOW())`,
		taskID, orgID, firstUserID, "PR21 two-approver test task", "planned"); err != nil {
		t.Fatalf("seed task: %v", err)
	}

	// Plan payload references a fake state_change_prod tool name so
	// planRequiresTwoApprovers can detect it by walking the payload.
	planPayload := map[string]any{
		"phases": []map[string]any{
			{"tool": "fake_state_change_for_co_approve"},
		},
	}
	planJSON, _ := json.Marshal(planPayload)
	if _, err := h.db.Pool.Exec(ctx,
		`INSERT INTO ai_plans (id, task_id, type, payload, state, created_at, updated_at)
		 VALUES ($1, $2, $3, $4::jsonb, $5, NOW(), NOW())`,
		uuid.NewString(), taskID, "patch_plan", string(planJSON), "awaiting_approval"); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	return taskID, firstUserID, secondUserID
}

// makeCoApproveReq mirrors makeInvokeReq for the /co-approve route.
func makeCoApproveReq(t *testing.T, taskID, userID string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/tasks/"+taskID+"/co-approve", http.NoBody)
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskID", taskID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	return req.WithContext(ctx)
}

// makeApproveReq mirrors the route for the existing /approve endpoint.
func makeApproveReq(t *testing.T, taskID, userID string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/tasks/"+taskID+"/approve", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskID", taskID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	return req.WithContext(ctx)
}

// registerStateChangeForTest adds a fake state_change_prod tool to the
// handler's registry so planRequiresTwoApprovers returns true.
func registerStateChangeForTest(t *testing.T, h *Handler) {
	t.Helper()
	tools.RegisterToolForTest(h.tools, &fakeInvocableTool{
		name: "fake_state_change_for_co_approve",
		risk: tools.RiskStateChangeProd,
	})
}

// TestApprove_DivertsStateChangeProd — first approval on a plan that
// references a state_change_prod tool records approved_by but state
// stays 'awaiting_approval' and the executor is NOT triggered. The
// response shape signals the next step explicitly.
func TestApprove_DivertsStateChangeProd(t *testing.T) {
	h := invokeTestHandler(t)
	registerStateChangeForTest(t, h)
	taskID, firstUser, _ := seedTwoApproverTask(t, h)

	w := httptest.NewRecorder()
	r := makeApproveReq(t, taskID, firstUser)
	h.approveTask(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	if resp["status"] != "awaiting_second_approval" {
		t.Errorf("status = %v, want awaiting_second_approval; body = %v", resp["status"], resp)
	}

	// Plan state should still be awaiting_approval, but approved_by set.
	var state string
	var approvedBy *string
	if err := h.db.Pool.QueryRow(context.Background(),
		`SELECT state, approved_by::text FROM ai_plans WHERE task_id = $1`,
		taskID,
	).Scan(&state, &approvedBy); err != nil {
		t.Fatalf("query plan: %v", err)
	}
	if state != "awaiting_approval" {
		t.Errorf("plan state = %q, want awaiting_approval (no transition until second approver)", state)
	}
	if approvedBy == nil || *approvedBy != firstUser {
		t.Errorf("approved_by = %v, want %s", approvedBy, firstUser)
	}
}

// TestCoApprove_HappyPath — first approve + co-approve by a different
// user transitions the plan to 'approved' with both approver columns
// set.
func TestCoApprove_HappyPath(t *testing.T) {
	h := invokeTestHandler(t)
	registerStateChangeForTest(t, h)
	taskID, firstUser, secondUser := seedTwoApproverTask(t, h)

	// First approval via /approve.
	w := httptest.NewRecorder()
	h.approveTask(w, makeApproveReq(t, taskID, firstUser))
	if w.Code != http.StatusOK {
		t.Fatalf("first approve failed: status %d, body = %s", w.Code, w.Body.String())
	}

	// Second approval via /co-approve from a distinct user.
	w2 := httptest.NewRecorder()
	h.coApproveTask(w2, makeCoApproveReq(t, taskID, secondUser))
	if w2.Code != http.StatusOK {
		t.Fatalf("co-approve failed: status %d, body = %s", w2.Code, w2.Body.String())
	}

	// Both columns set, state flipped to approved.
	var state string
	var approvedBy, secondApprover *string
	if err := h.db.Pool.QueryRow(context.Background(),
		`SELECT state, approved_by::text, second_approver::text FROM ai_plans WHERE task_id = $1`,
		taskID,
	).Scan(&state, &approvedBy, &secondApprover); err != nil {
		t.Fatalf("query plan: %v", err)
	}
	if state != "approved" {
		t.Errorf("plan state = %q, want approved", state)
	}
	if approvedBy == nil || *approvedBy != firstUser {
		t.Errorf("approved_by = %v, want %s", approvedBy, firstUser)
	}
	if secondApprover == nil || *secondApprover != secondUser {
		t.Errorf("second_approver = %v, want %s", secondApprover, secondUser)
	}
}

// TestCoApprove_RejectsSameApprover — second approver cannot equal first.
// The handler returns 403 (forbidden, not 409) so the caller knows this
// is a policy refusal, not a race condition.
func TestCoApprove_RejectsSameApprover(t *testing.T) {
	h := invokeTestHandler(t)
	registerStateChangeForTest(t, h)
	taskID, firstUser, _ := seedTwoApproverTask(t, h)

	// First approval.
	h.approveTask(httptest.NewRecorder(), makeApproveReq(t, taskID, firstUser))

	// Same user attempts co-approve.
	w := httptest.NewRecorder()
	h.coApproveTask(w, makeCoApproveReq(t, taskID, firstUser))

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403 (self-second-approve must be rejected); body = %s", w.Code, w.Body.String())
	}

	// State must not have advanced.
	var state string
	if sErr := h.db.Pool.QueryRow(context.Background(),
		`SELECT state FROM ai_plans WHERE task_id = $1`, taskID,
	).Scan(&state); sErr != nil {
		t.Fatalf("query plan state: %v", sErr)
	}
	if state != "awaiting_approval" {
		t.Errorf("plan state advanced to %q despite self-second-approve rejection", state)
	}
}

// TestCoApprove_RejectsPlanThatDoesntNeedTwoApprovers — calling
// /co-approve on a plan whose tools don't require two approvers is a
// 422. The caller used the wrong endpoint.
func TestCoApprove_RejectsPlanThatDoesntNeedTwoApprovers(t *testing.T) {
	h := invokeTestHandler(t)
	// Register a different fake tool, only read_only, then seed a task
	// whose plan payload references THAT tool (not the state_change one).
	tools.RegisterToolForTest(h.tools, &fakeInvocableTool{
		name: "fake_read_only_no_two_approvers",
		risk: tools.RiskReadOnly,
	})

	ctx := context.Background()
	orgID := uuid.NewString()
	userID := uuid.NewString()
	taskID := uuid.NewString()
	t.Cleanup(func() {
		_, _ = h.db.Pool.Exec(context.Background(), "DELETE FROM ai_plans WHERE task_id = $1", taskID) //nolint:errcheck
		_, _ = h.db.Pool.Exec(context.Background(), "DELETE FROM ai_tasks WHERE id = $1", taskID)      //nolint:errcheck
		_, _ = h.db.Pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", userID)         //nolint:errcheck
		_, _ = h.db.Pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1", orgID)  //nolint:errcheck
	})
	_, _ = h.db.Pool.Exec(ctx, `INSERT INTO organizations (id, name, slug) VALUES ($1, $2, $3)`, //nolint:errcheck
		orgID, "PR21 no-two-approver", "pr21-"+uuid.NewString()[:8])
	_, _ = h.db.Pool.Exec(ctx, `INSERT INTO users (id, org_id, external_id, email, name) VALUES ($1, $2, $3, $4, $5)`, //nolint:errcheck
		userID, orgID, "ext-"+userID[:8], userID[:8]+"@example.test", "Test User")
	_, _ = h.db.Pool.Exec(ctx, //nolint:errcheck
		`INSERT INTO ai_tasks (id, org_id, created_by, user_intent, state, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW(), NOW())`,
		taskID, orgID, userID, "PR21 no-two-approver task", "planned")
	planPayload, _ := json.Marshal(map[string]any{"phases": []map[string]any{{"tool": "fake_read_only_no_two_approvers"}}}) //nolint:errcheck
	_, _ = h.db.Pool.Exec(ctx,                                                                                              //nolint:errcheck
		`INSERT INTO ai_plans (id, task_id, type, payload, state, created_at, updated_at)
		 VALUES ($1, $2, $3, $4::jsonb, $5, NOW(), NOW())`,
		uuid.NewString(), taskID, "patch_plan", string(planPayload), "awaiting_approval")

	w := httptest.NewRecorder()
	h.coApproveTask(w, makeCoApproveReq(t, taskID, userID))

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422; body = %s", w.Code, w.Body.String())
	}
}

// TestPlanRequiresTwoApprovers_DetectsToolByName — walks the JSONB
// payload and matches tool names against the registry. State_change_prod
// tools return true; read_only/plan_only return false; missing plan
// returns an error.
func TestPlanRequiresTwoApprovers_DetectsToolByName(t *testing.T) {
	h := invokeTestHandler(t)
	registerStateChangeForTest(t, h)

	taskID, _, _ := seedTwoApproverTask(t, h)
	got, err := h.planRequiresTwoApprovers(context.Background(), taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Error("expected planRequiresTwoApprovers true for state_change_prod tool, got false")
	}

	// Missing task — error.
	if _, err := h.planRequiresTwoApprovers(context.Background(), uuid.NewString()); err == nil {
		t.Error("expected error for missing task, got nil")
	}
}
