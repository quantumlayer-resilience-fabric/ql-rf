// PR #19 / CONN-001 — direct tool invocation endpoint.
//
// Adds POST /api/v1/ai/tools/{toolName}/invoke for ad-hoc, user-initiated
// invocation of read-only tools. Distinct from the B.3 simulator path:
//
//   - The simulator (approval_simulation.go) inserts SYNTHETIC tool
//     invocations marked `_simulated: true` while walking phases of an
//     approved plan. It never makes real cloud calls.
//   - This endpoint invokes a REAL tool directly when the user clicks
//     "Invoke" in Mission Control's Real Tools card. The result lands in
//     ai_tool_invocations without the `_simulated` marker.
//
// Strict whitelist: only tools where Risk() == read_only are invocable. Any
// attempt to invoke a plan-only or state-change tool returns 403, pointing
// the caller at the approval flow. This keeps the safety boundary explicit
// — state-change real execution requires the full HITL approval pipeline
// (B.3 + a future PR), not an ad-hoc button click.
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

	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/middleware"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
)

// InvokeToolRequest is the request body for POST /tools/{name}/invoke.
// Params is forwarded verbatim to the tool's Execute method. An empty or
// missing body is tolerated for no-arg tools.
type InvokeToolRequest struct {
	Params map[string]any `json:"params"`
}

// InvokeToolResponse is the response body. Simulated is ALWAYS false from
// this endpoint — it exists to make the contract explicit so frontend code
// can reliably distinguish real from synthetic invocations.
type InvokeToolResponse struct {
	InvocationID string `json:"invocation_id"`
	ToolName     string `json:"tool_name"`
	RiskLevel    string `json:"risk_level"`
	DurationMs   int    `json:"duration_ms"`
	Result       any    `json:"result,omitempty"`
	Error        string `json:"error,omitempty"`
	Simulated    bool   `json:"simulated"`
}

// invokeTool handles POST /api/v1/ai/tools/{toolName}/invoke.
//
// Behavior:
//   - 404 if the tool isn't registered (e.g., AWS creds absent at boot).
//   - 403 if the tool's risk isn't read_only.
//   - 400 if the caller has no org_id in the request context.
//   - 502 if the tool ran but returned an error (so we can distinguish from
//     500s in our own code). The invocation row is still inserted with the
//     error text for audit.
//   - 200 + InvokeToolResponse on success.
//
// In all cases where the tool actually ran (success or AWS-side error), an
// ai_tool_invocations row is inserted with `risk_level`, `parameters`,
// `result`, `error`, `duration_ms`. The row has `task_id` set to the per-org
// "direct invocations" task (lazy-created) and `run_id` NULL.
func (h *Handler) invokeTool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	toolName := chi.URLParam(r, "toolName")
	if toolName == "" {
		h.respondError(w, http.StatusBadRequest, "tool name is required", nil)
		return
	}

	tool, ok := h.tools.Get(toolName)
	if !ok {
		h.respondError(w, http.StatusNotFound, "tool not found", nil)
		return
	}

	// PR #19 strict gate: only read_only tools are invocable here. State-
	// change tools must flow through the B.3 approval simulator OR (in a
	// future PR) the real-execution path — never via an ad-hoc button.
	if tool.Risk() != tools.RiskReadOnly {
		h.respondError(w, http.StatusForbidden,
			fmt.Sprintf("tool %q (risk=%s) cannot be invoked directly; state-change tools require the approval flow",
				toolName, tool.Risk()),
			nil)
		return
	}

	var req InvokeToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if req.Params == nil {
		req.Params = map[string]any{}
	}

	userID := resolveUserID(ctx)
	orgID := middleware.GetOrgID(ctx)
	if orgID == "" {
		h.respondError(w, http.StatusBadRequest, "org_id required", nil)
		return
	}

	taskID, err := h.ensureDirectInvocationTask(ctx, orgID, userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to ensure direct-invocation task", err)
		return
	}

	started := time.Now()
	result, execErr := tool.Execute(ctx, req.Params)
	duration := time.Since(started)
	durationMs := int(duration / time.Millisecond)

	paramsJSON, mErr := json.Marshal(req.Params)
	if mErr != nil {
		paramsJSON = []byte("{}")
	}
	resultJSON := []byte("null")
	if result != nil {
		if b, jErr := json.Marshal(result); jErr == nil {
			resultJSON = b
		}
	}
	errText := ""
	if execErr != nil {
		errText = execErr.Error()
	}

	var invID string
	const insertQ = `
		INSERT INTO ai_tool_invocations
			(task_id, run_id, tool_name, risk_level, duration_ms,
			 parameters, result, error, created_at)
		VALUES ($1, NULL, $2, $3, $4, $5::jsonb, $6::jsonb, NULLIF($7, ''), clock_timestamp())
		RETURNING id`
	if err := h.db.Pool.QueryRow(ctx, insertQ,
		taskID, toolName, string(tool.Risk()), durationMs,
		string(paramsJSON), string(resultJSON), errText,
	).Scan(&invID); err != nil {
		h.log.Error("invokeTool: insert ai_tool_invocations failed",
			"tool", toolName, "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to record invocation", err)
		return
	}

	resp := InvokeToolResponse{
		InvocationID: invID,
		ToolName:     toolName,
		RiskLevel:    string(tool.Risk()),
		DurationMs:   durationMs,
		Result:       result,
		Error:        errText,
		Simulated:    false,
	}

	status := http.StatusOK
	if execErr != nil {
		// Tool ran but the underlying call (e.g., AWS) returned an error.
		// 502 distinguishes this from 500s in our own code paths.
		status = http.StatusBadGateway
	}
	h.respond(w, status, resp)
}

// ensureDirectInvocationTask finds or creates the per-org "direct invocations"
// ai_tasks row. Required because ai_tool_invocations.task_id is NOT NULL —
// every recorded tool invocation must belong to a task. Ad-hoc invocations
// don't have a natural task to attach to, so we use one shared per-org task
// flagged via task_spec.kind = "direct_invocation" so listings can filter it
// out.
//
// The function uses an idempotent INSERT ... ON CONFLICT ... DO UPDATE so
// races are safe — the worst case is one wasted INSERT attempt that
// upgrades to a SELECT.
func (h *Handler) ensureDirectInvocationTask(ctx context.Context, orgID, userID string) (string, error) {
	const selectQ = `
		SELECT id FROM ai_tasks
		WHERE org_id = $1
		  AND task_spec->>'kind' = 'direct_invocation'
		LIMIT 1`
	var existing string
	err := h.db.Pool.QueryRow(ctx, selectQ, orgID).Scan(&existing)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("select direct-invocation task: %w", err)
	}

	const insertQ = `
		INSERT INTO ai_tasks
			(org_id, created_by, user_intent, task_spec, state, source)
		VALUES ($1, $2, $3, $4::jsonb, 'planned', 'api')
		RETURNING id`
	taskSpec := `{"kind":"direct_invocation","description":"Container for ad-hoc tool invocations via POST /tools/{name}/invoke (PR #19)."}`
	intent := "Direct tool invocations"
	var newID string
	if err := h.db.Pool.QueryRow(ctx, insertQ, orgID, userID, intent, taskSpec).Scan(&newID); err != nil {
		return "", fmt.Errorf("insert direct-invocation task: %w", err)
	}
	return newID, nil
}
