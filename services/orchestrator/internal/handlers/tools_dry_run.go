// PR #20 / CONN-002 — direct dry-run endpoint for state-change tools.
//
// Symmetric to PR #19's POST /api/v1/ai/tools/{name}/invoke, but for the
// opposite end of the risk spectrum. /invoke accepts read_only tools only;
// /dry-run accepts state_change_* tools only. Plan-only tools have no
// dry-run semantic — they're already non-mutating; they're rejected too.
//
// Audit shape: identical to /invoke EXCEPT the inserted parameters and
// result JSONB blobs carry a `dry_run: true` marker. SQL queries can
// distinguish the three audit kinds today:
//
//   - synthetic (B.3 simulator):    `parameters @> '{"_simulated": true}'`
//   - real read-only (PR #19):      no marker, risk_level='read_only'
//   - state-change dry-run (PR #20): `parameters @> '{"dry_run": true}'`
//
// Live state-change (PR #21) will land as a fourth kind:
// `parameters @> '{"dry_run": false}'` AND risk_level='state_change_prod'.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/middleware"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
)

// dryRunTool handles POST /api/v1/ai/tools/{toolName}/dry-run.
//
// Behavior:
//   - 404 if the tool isn't registered.
//   - 403 if the tool's risk isn't state_change_nonprod or state_change_prod.
//   - 400 if the caller has no org_id in the request context.
//   - 502 if the tool ran but returned an error (audit row still inserted).
//   - 200 + InvokeToolResponse on success.
//
// The InvokeToolResponse type is reused from PR #19; the only response-
// shape difference is that the `parameters` and `result` recorded in the
// audit row carry the `dry_run: true` marker that downstream consumers
// (this file, SQL queries, the activity stream) use to distinguish dry-run
// from hypothetical live invocations.
func (h *Handler) dryRunTool(w http.ResponseWriter, r *http.Request) {
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

	// PR #20 strict gate: ONLY state-change tools are dry-run-able. Read-
	// only tools have no dry-run meaning (they're already non-mutating);
	// plan-only tools are descriptions of what would happen, not requests
	// to make it happen. Both 403 here.
	risk := tool.Risk()
	if risk != tools.RiskStateChangeNonProd && risk != tools.RiskStateChangeProd {
		h.respondError(w, http.StatusForbidden,
			"/dry-run only accepts state-change tools; use /invoke for read-only and plan-only", nil)
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
	// Force the dry_run marker into params so it's surfaced in the audit
	// row even if the tool's own result envelope somehow drops it. Defense
	// in depth: the dry-run distinction lives in TWO places (params AND
	// result), so a SQL `parameters @> '{"dry_run":true}'` query catches
	// every dry-run row regardless of tool implementation.
	req.Params["dry_run"] = true

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
	durationMs := int(time.Since(started) / time.Millisecond)

	invID, insErr := h.recordDryRunInvocation(ctx, taskID, toolName, string(risk),
		durationMs, req.Params, result, execErr)
	if insErr != nil {
		h.log.Error("dryRunTool: insert ai_tool_invocations failed",
			"tool", toolName, "error", insErr)
		h.respondError(w, http.StatusInternalServerError, "failed to record invocation", insErr)
		return
	}
	errText := ""
	if execErr != nil {
		errText = execErr.Error()
	}

	paramsJSON, _ := json.Marshal(req.Params)
	resultJSON, _ := json.Marshal(result)
	h.tryEmitComplianceEvidence(ctx, invID, toolName, string(risk), orgID, paramsJSON, resultJSON)

	resp := InvokeToolResponse{
		InvocationID: invID,
		ToolName:     toolName,
		RiskLevel:    string(risk),
		DurationMs:   durationMs,
		Result:       result,
		Error:        errText,
		// Simulated is false because this isn't B.3 simulator output. The
		// dry-run distinction lives in the result envelope and the audit
		// row's `parameters @> '{"dry_run":true}'` marker.
		Simulated: false,
	}

	status := http.StatusOK
	if execErr != nil {
		status = http.StatusBadGateway
	}
	h.respond(w, status, resp)
}

// recordDryRunInvocation marshals params + result, then inserts the
// ai_tool_invocations row. Extracted from dryRunTool to keep its cyclomatic
// complexity below the gocyclo threshold; the marshal-fallback branches and
// nullable-error branch all live here.
func (h *Handler) recordDryRunInvocation(
	ctx context.Context,
	taskID, toolName, riskLevel string,
	durationMs int,
	params map[string]any,
	result any,
	execErr error,
) (string, error) {
	paramsJSON, mErr := json.Marshal(params)
	if mErr != nil {
		paramsJSON = []byte(`{"dry_run":true}`)
	}
	resultJSON := []byte(`null`)
	if result != nil {
		if b, jErr := json.Marshal(result); jErr == nil {
			resultJSON = b
		}
	}
	errText := ""
	if execErr != nil {
		errText = execErr.Error()
	}

	const insertQ = `
		INSERT INTO ai_tool_invocations
			(task_id, run_id, tool_name, risk_level, duration_ms,
			 parameters, result, error, created_at)
		VALUES ($1, NULL, $2, $3, $4, $5::jsonb, $6::jsonb, NULLIF($7, ''), clock_timestamp())
		RETURNING id`
	var invID string
	if err := h.db.Pool.QueryRow(ctx, insertQ,
		taskID, toolName, riskLevel, durationMs,
		string(paramsJSON), string(resultJSON), errText,
	).Scan(&invID); err != nil {
		return "", err
	}
	return invID, nil
}
