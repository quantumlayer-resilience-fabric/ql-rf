// PR #21 / CONN-003 — second-approver handler for state_change_prod tasks.
//
// The first approver hits POST /api/v1/ai/tasks/{id}/approve (existing
// endpoint) and the handler records `approved_by` + `approved_at` but
// does NOT transition the plan to `approved` and does NOT trigger the
// executor. Instead it returns `status: awaiting_second_approval`.
//
// A second, DISTINCT approver hits POST /api/v1/ai/tasks/{id}/co-approve
// (this file's handler) and that's what records `second_approver` +
// `second_approved_at`, transitions to `state=approved`, and fires the
// executor.
//
// Three gates on /co-approve:
//
//  1. Plan must currently be in `awaiting_approval` with `approved_by`
//     set and `second_approver` NULL — otherwise the request is in the
//     wrong state and we 409 / 404.
//  2. The caller's user_id must differ from `approved_by` — no self-
//     second-approve. The OPA policy also enforces this at tool
//     invocation time as defense in depth.
//  3. The plan must currently require two approvers (i.e. plan involves
//     a state_change_prod tool); otherwise the caller is using the wrong
//     endpoint.
//
// On success the plan flips to `approved` and the same executor path
// used by approveTask fires. Notifier and Temporal signals also fire,
// keeping the downstream shape identical to a single-approver approval.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
)

// coApproveTask handles POST /api/v1/ai/tasks/{taskID}/co-approve.
//
// Status codes:
//   - 200 + body on success
//   - 400 if user_id can't be resolved
//   - 403 if the caller is the same as the first approver
//   - 404 if the task isn't found
//   - 409 if the plan isn't in `awaiting_approval` with first approval
//     already recorded
//   - 422 if the plan doesn't require two approvers (wrong endpoint)
//
// coApproveValidationResult is the typed output of the precondition
// checks at the top of coApproveTask. Carrying the validated firstApprover
// out of the helper avoids a second DB roundtrip.
type coApproveValidationResult struct {
	userID        string
	firstApprover string
}

// validateCoApprove runs the gates that must pass before we touch the
// plan row. Extracted so coApproveTask itself stays under the cyclomatic
// complexity limit. Returns the validated user/first-approver pair on
// success; on failure it has already written the HTTP error response and
// the caller must just return.
func (h *Handler) validateCoApprove(w http.ResponseWriter, r *http.Request) (coApproveValidationResult, bool) {
	taskID := chi.URLParam(r, "taskID")
	ctx := r.Context()

	var req ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != decoderEOFMessage {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return coApproveValidationResult{}, false
	}

	userID := resolveUserID(ctx)
	if userID == "" {
		h.respondError(w, http.StatusBadRequest, "user_id required", nil)
		return coApproveValidationResult{}, false
	}

	requiresSecond, err := h.planRequiresTwoApprovers(ctx, taskID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "task or plan not found", err)
		return coApproveValidationResult{}, false
	}
	if !requiresSecond {
		h.respondError(w, http.StatusUnprocessableEntity,
			"this plan does not require two approvers; use POST /tasks/{id}/approve", nil)
		return coApproveValidationResult{}, false
	}

	exists, firstApprover, fErr := h.firstApprovalState(ctx, taskID)
	if fErr != nil || !exists {
		h.respondError(w, http.StatusNotFound, "plan not found for task", fErr)
		return coApproveValidationResult{}, false
	}
	if firstApprover == "" {
		h.respondError(w, http.StatusConflict,
			"first approval has not been recorded yet; call /approve first", nil)
		return coApproveValidationResult{}, false
	}
	if firstApprover == userID {
		h.respondError(w, http.StatusForbidden,
			"second approver must differ from first approver", nil)
		return coApproveValidationResult{}, false
	}
	return coApproveValidationResult{userID: userID, firstApprover: firstApprover}, true
}

// decoderEOFMessage is the runtime string returned by json.Decoder.Decode
// when the body is empty. Extracted to a constant so goconst doesn't flag
// repeated literal usage across the handlers package.
const decoderEOFMessage = "EOF"

func (h *Handler) coApproveTask(w http.ResponseWriter, r *http.Request) {
	v, ok := h.validateCoApprove(w, r)
	if !ok {
		return
	}
	taskID := chi.URLParam(r, "taskID")
	ctx := r.Context()
	userID := v.userID
	firstApprover := v.firstApprover

	now := time.Now().UTC()

	// Atomic update: set second_approver fields AND transition state, in
	// one statement, gated by the current state + first-approver shape.
	// This blocks two co-approvers racing: only one UPDATE matches the
	// WHERE clause.
	tag, err := h.db.Pool.Exec(ctx, `
		UPDATE ai_plans
		SET state = 'approved',
		    second_approver = $1,
		    second_approved_at = $2,
		    updated_at = $2
		WHERE task_id = $3
		  AND state = 'awaiting_approval'
		  AND approved_by IS NOT NULL
		  AND second_approver IS NULL
		  AND approved_by != $1
	`, userID, now, taskID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to record second approval", err)
		return
	}
	if tag.RowsAffected() == 0 {
		// Most likely race-loss to another concurrent co-approver, or the
		// plan state moved out from under us.
		h.respondError(w, http.StatusConflict,
			"plan state changed during co-approval; refresh and retry", nil)
		return
	}

	// Update the task row to match the plan's new approved state so the
	// activity stream sees the consistent shape PR #20 already expects.
	if _, err := h.db.Pool.Exec(ctx, `
		UPDATE ai_tasks
		SET state = 'approved', updated_at = $1
		WHERE id = $2
	`, now, taskID); err != nil {
		h.log.Warn("co-approve: failed to update ai_tasks state",
			"task_id", taskID, "error", err)
	}

	h.log.Info("second approval recorded; triggering execution",
		"task_id", taskID,
		"approved_by", firstApprover,
		"second_approver", userID,
	)

	var executionID string
	if h.executor != nil {
		execution, eErr := h.startExecution(ctx, taskID, userID)
		if eErr != nil {
			h.log.Error("co-approve: failed to start execution", "error", eErr)
		} else {
			executionID = execution.ID
			if _, eUpd := h.db.Pool.Exec(ctx, `
				UPDATE ai_tasks SET state = 'executing', updated_at = $1 WHERE id = $2
			`, time.Now().UTC(), taskID); eUpd != nil {
				h.log.Warn("co-approve: failed to mark task executing",
					"task_id", taskID, "error", eUpd)
			}
		}
	}

	if h.notifier != nil {
		// Notification is fire-and-forget; use a fresh background context
		// because the request ctx may be cancelled by the caller before
		// the webhook returns. Matches the pattern in approveTask.
		//nolint:contextcheck // intentional detached context for fire-and-forget notifier
		go func() {
			notifyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if nErr := h.notifier.NotifyTaskApproved(notifyCtx, taskID, userID); nErr != nil {
				h.log.Error("failed to send co-approval notification", "error", nErr)
			}
		}()
	}

	resp := map[string]any{
		"task_id":         taskID,
		"status":          "approved",
		"approved_by":     firstApprover,
		"second_approver": userID,
		"co_approved_at":  now,
	}
	if executionID != "" {
		resp["execution_id"] = executionID
	}
	h.respond(w, http.StatusOK, resp)
}

// planRequiresTwoApprovers returns true if any tool referenced by the
// task's plan has risk `state_change_prod`. Falls back to checking the
// registry by tool name when the plan payload doesn't embed risk_level
// directly.
//
// Errors propagate so the caller can distinguish "plan doesn't exist"
// from "plan exists but doesn't require two approvers".
func (h *Handler) planRequiresTwoApprovers(ctx context.Context, taskID string) (bool, error) {
	var payload []byte
	err := h.db.Pool.QueryRow(ctx, `
		SELECT payload
		FROM ai_plans
		WHERE task_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, taskID).Scan(&payload)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf("no plan for task %s", taskID)
		}
		return false, fmt.Errorf("read plan payload: %w", err)
	}

	var doc map[string]any
	if jErr := json.Unmarshal(payload, &doc); jErr != nil {
		return false, fmt.Errorf("parse plan payload: %w", jErr)
	}

	for _, name := range collectToolNames(doc) {
		t, ok := h.tools.Get(name)
		if !ok {
			continue
		}
		if t.Risk() == tools.RiskStateChangeProd {
			return true, nil
		}
	}
	return false, nil
}

// firstApprovalState returns whether the plan exists and the user ID of
// the first approver (empty string if not yet recorded). Used by both
// approveTask (to detect a misrouted second approval) and coApproveTask
// (to enforce distinct approvers).
func (h *Handler) firstApprovalState(ctx context.Context, taskID string) (exists bool, approvedBy string, err error) {
	var approver *string
	err = h.db.Pool.QueryRow(ctx, `
		SELECT approved_by::text
		FROM ai_plans
		WHERE task_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, taskID).Scan(&approver)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", nil
		}
		return false, "", err
	}
	if approver == nil {
		return true, "", nil
	}
	return true, *approver, nil
}

// collectToolNames extracts tool names from the plan payload's phases.
// The payload shape varies by plan type; this helper walks the JSON
// generically and pulls strings keyed by "tool" or "tool_name".
func collectToolNames(doc map[string]any) []string {
	var out []string
	var walk func(v any)
	walk = func(v any) {
		switch t := v.(type) {
		case map[string]any:
			for k, val := range t {
				if k == "tool" || k == "tool_name" {
					if s, ok := val.(string); ok && s != "" {
						out = append(out, s)
					}
				}
				walk(val)
			}
		case []any:
			for _, item := range t {
				walk(item)
			}
		}
	}
	walk(doc)
	return out
}
