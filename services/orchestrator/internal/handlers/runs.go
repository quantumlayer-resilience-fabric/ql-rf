// Package handlers — Run detail read endpoints (PR #16 / UX-001).
//
// Surfaces the ai_runs.audit_log evidence ledger in Mission Control. Two
// read-only endpoints:
//
//	GET /api/v1/ai/runs[?limit=N]      — recent runs for the caller's org
//	GET /api/v1/ai/runs/{runID}        — single run + audit_log + invocations
//
// Both are scoped to the caller's org via middleware.GetOrgID; cross-org
// access returns an empty list (list) or 404 (detail). The B.3 simulator
// is the only writer to ai_runs / ai_tool_invocations — this file only
// reads.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/middleware"
)

const (
	defaultRunsLimit = 5
	maxRunsLimit     = 50
)

// RunSummary is the per-row shape returned by GET /api/v1/ai/runs.
// Joined with ai_tasks and ai_plans so the rail can render the
// user_intent + plan_type without a second round-trip.
type RunSummary struct {
	ID              string  `json:"id"`
	TaskID          string  `json:"task_id"`
	PlanID          string  `json:"plan_id"`
	UserIntent      string  `json:"user_intent"`
	PlanType        string  `json:"plan_type"`
	State           string  `json:"state"`
	CurrentPhase    string  `json:"current_phase"`
	PercentComplete int     `json:"percent_complete"`
	Environment     string  `json:"environment"`
	PhasesTotal     int     `json:"phases_total"`
	StartedAt       *string `json:"started_at,omitempty"`
	CompletedAt     *string `json:"completed_at,omitempty"`
	UpdatedAt       string  `json:"updated_at"`
	Simulated       bool    `json:"simulated"`
}

// RunDetail extends RunSummary with the full audit_log, phase trackers,
// metrics, and any error text. audit_log + metrics are passed through as
// json.RawMessage so the orchestrator never re-marshals them.
type RunDetail struct {
	RunSummary
	AuditLog        []json.RawMessage `json:"audit_log"`
	PhasesCompleted []string          `json:"phases_completed"`
	PhasesRemaining []string          `json:"phases_remaining"`
	Metrics         json.RawMessage   `json:"metrics"`
	Error           string            `json:"error,omitempty"`
}

// RunToolInvocation is the per-invocation shape attached to a RunDetail
// response so the timeline can render phase → tool inline.
type RunToolInvocation struct {
	ID         string          `json:"id"`
	ToolName   string          `json:"tool_name"`
	RiskLevel  string          `json:"risk_level"`
	DurationMs *int            `json:"duration_ms,omitempty"`
	Parameters json.RawMessage `json:"parameters"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      string          `json:"error,omitempty"`
	CreatedAt  string          `json:"created_at"`
}

// listRuns responds to GET /api/v1/ai/runs?limit=N. Returns the caller's
// org's most recent runs (across all lifecycle states) by updated_at DESC.
//
// limit defaults to defaultRunsLimit (5) and is capped at maxRunsLimit (50).
// Read-only: never invokes an agent, LLM, or cloud SDK.
func (h *Handler) listRuns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrgID(ctx)

	limit := defaultRunsLimit
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			if parsed > maxRunsLimit {
				parsed = maxRunsLimit
			}
			limit = parsed
		}
	}

	const q = `
		SELECT r.id, r.task_id, r.plan_id,
		       COALESCE(t.user_intent, '') AS user_intent,
		       COALESCE(p.type, '')        AS plan_type,
		       r.state, COALESCE(r.current_phase, '') AS current_phase,
		       r.percent_complete, r.environment,
		       jsonb_array_length(r.phases_completed)
		         + jsonb_array_length(r.phases_remaining)
		         + CASE WHEN COALESCE(r.current_phase, '') = '' THEN 0 ELSE 1 END AS phases_total,
		       r.started_at, r.completed_at, r.updated_at,
		       (r.audit_log @> '[{"_simulated": true}]'::jsonb) AS simulated
		FROM ai_runs r
		JOIN ai_tasks t ON t.id = r.task_id
		LEFT JOIN ai_plans p ON p.id = r.plan_id
		WHERE t.org_id = $1
		ORDER BY r.updated_at DESC
		LIMIT $2`
	rows, err := h.db.Pool.Query(ctx, q, orgID, limit)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to list runs", err)
		return
	}
	defer rows.Close()

	runs := []RunSummary{}
	for rows.Next() {
		s, err := scanRunSummary(rows)
		if err != nil {
			h.log.Warn("listRuns: scan failed", "error", err)
			continue
		}
		runs = append(runs, s)
	}

	h.respond(w, http.StatusOK, map[string]any{"runs": runs})
}

// getRun responds to GET /api/v1/ai/runs/{runID}. Returns the full run row
// plus its tool invocations. Authorized to the caller's org via a join on
// ai_tasks; cross-org access returns 404. Read-only.
func (h *Handler) getRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	runID := chi.URLParam(r, "runID")
	if runID == "" {
		h.respondError(w, http.StatusBadRequest, "run id is required", nil)
		return
	}
	orgID := middleware.GetOrgID(ctx)

	const q = `
		SELECT r.id, r.task_id, r.plan_id,
		       COALESCE(t.user_intent, '') AS user_intent,
		       COALESCE(p.type, '')        AS plan_type,
		       r.state, COALESCE(r.current_phase, '') AS current_phase,
		       r.percent_complete, r.environment,
		       jsonb_array_length(r.phases_completed)
		         + jsonb_array_length(r.phases_remaining)
		         + CASE WHEN COALESCE(r.current_phase, '') = '' THEN 0 ELSE 1 END AS phases_total,
		       r.started_at, r.completed_at, r.updated_at,
		       (r.audit_log @> '[{"_simulated": true}]'::jsonb) AS simulated,
		       r.audit_log, r.phases_completed, r.phases_remaining,
		       r.metrics, COALESCE(r.error, '') AS error
		FROM ai_runs r
		JOIN ai_tasks t ON t.id = r.task_id
		LEFT JOIN ai_plans p ON p.id = r.plan_id
		WHERE r.id = $1 AND t.org_id = $2`

	var (
		detail              RunDetail
		auditLogJSON        []byte
		phasesCompletedJSON []byte
		phasesRemainingJSON []byte
		metricsJSON         []byte
		startedAt           *time.Time
		completedAt         *time.Time
		updatedAt           time.Time
	)
	err := h.db.Pool.QueryRow(ctx, q, runID, orgID).Scan(
		&detail.ID, &detail.TaskID, &detail.PlanID,
		&detail.UserIntent, &detail.PlanType,
		&detail.State, &detail.CurrentPhase,
		&detail.PercentComplete, &detail.Environment,
		&detail.PhasesTotal,
		&startedAt, &completedAt, &updatedAt,
		&detail.Simulated,
		&auditLogJSON, &phasesCompletedJSON, &phasesRemainingJSON,
		&metricsJSON, &detail.Error,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.respondError(w, http.StatusNotFound, "run not found", nil)
			return
		}
		h.respondError(w, http.StatusInternalServerError, "failed to fetch run", err)
		return
	}

	if startedAt != nil {
		s := startedAt.UTC().Format(time.RFC3339Nano)
		detail.StartedAt = &s
	}
	if completedAt != nil {
		c := completedAt.UTC().Format(time.RFC3339Nano)
		detail.CompletedAt = &c
	}
	detail.UpdatedAt = updatedAt.UTC().Format(time.RFC3339Nano)

	// audit_log is a JSON array — split into per-entry RawMessages so the
	// frontend can iterate without losing field order or boolean precision.
	if len(auditLogJSON) > 0 {
		var entries []json.RawMessage
		if err := json.Unmarshal(auditLogJSON, &entries); err != nil {
			h.log.Warn("getRun: audit_log unmarshal failed", "error", err)
			entries = []json.RawMessage{}
		}
		detail.AuditLog = entries
	} else {
		detail.AuditLog = []json.RawMessage{}
	}
	if err := json.Unmarshal(phasesCompletedJSON, &detail.PhasesCompleted); err != nil {
		detail.PhasesCompleted = []string{}
	}
	if err := json.Unmarshal(phasesRemainingJSON, &detail.PhasesRemaining); err != nil {
		detail.PhasesRemaining = []string{}
	}
	detail.Metrics = metricsJSON

	// Tool invocations for this run, ordered by created_at so the timeline's
	// phase ↔ tool 1:1 alignment is deterministic.
	const invQ = `
		SELECT id, tool_name, risk_level, duration_ms,
		       parameters, result, COALESCE(error, '') AS error, created_at
		FROM ai_tool_invocations
		WHERE run_id = $1
		ORDER BY created_at ASC`
	invRows, iErr := h.db.Pool.Query(ctx, invQ, runID)
	invocations := []RunToolInvocation{}
	if iErr != nil {
		h.log.Warn("getRun: tool invocations query failed", "error", iErr)
	} else {
		defer invRows.Close()
		for invRows.Next() {
			var (
				inv       RunToolInvocation
				durMs     *int
				result    []byte
				createdAt time.Time
			)
			if err := invRows.Scan(
				&inv.ID, &inv.ToolName, &inv.RiskLevel, &durMs,
				&inv.Parameters, &result, &inv.Error, &createdAt,
			); err != nil {
				h.log.Warn("getRun: scan invocation failed", "error", err)
				continue
			}
			inv.DurationMs = durMs
			if len(result) > 0 {
				inv.Result = result
			}
			inv.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
			invocations = append(invocations, inv)
		}
	}

	h.respond(w, http.StatusOK, map[string]any{
		"run":              detail,
		"tool_invocations": invocations,
	})
}

// scanRunSummary scans one row from the listRuns / similar query into a
// RunSummary. Extracted so the row-iteration loop stays compact.
func scanRunSummary(rows pgx.Rows) (RunSummary, error) {
	var (
		s           RunSummary
		startedAt   *time.Time
		completedAt *time.Time
		updatedAt   time.Time
	)
	if err := rows.Scan(
		&s.ID, &s.TaskID, &s.PlanID,
		&s.UserIntent, &s.PlanType,
		&s.State, &s.CurrentPhase,
		&s.PercentComplete, &s.Environment,
		&s.PhasesTotal,
		&startedAt, &completedAt, &updatedAt,
		&s.Simulated,
	); err != nil {
		return s, err
	}
	if startedAt != nil {
		v := startedAt.UTC().Format(time.RFC3339Nano)
		s.StartedAt = &v
	}
	if completedAt != nil {
		v := completedAt.UTC().Format(time.RFC3339Nano)
		s.CompletedAt = &v
	}
	s.UpdatedAt = updatedAt.UTC().Format(time.RFC3339Nano)
	return s, nil
}
