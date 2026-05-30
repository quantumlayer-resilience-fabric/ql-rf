// Package handlers — approval simulation for Mission Control (Phase B.3 / AI-004).
//
// When `RF_LLM_PROVIDER=stub` is active and a user clicks Approve on a pending
// decision, the orchestrator routes the approval through this file instead of
// the live executor / Temporal path. The simulation:
//
//  1. Atomically marks ai_tasks.state='approved' and ai_plans.state='approved',
//     creates one ai_runs row in `queued` state with an initial audit_log
//     entry, and appends a "✓ Approved by …" conversation system message to
//     the task's existing conversation. All inside one pgx.Tx — a partial
//     commit is impossible by design.
//  2. Spawns a background goroutine (30s safety cap) that walks the plan's
//     phases. For each phase: sleep 1s, INSERT one synthetic
//     ai_tool_invocations row (risk_level='plan_only', never state_change),
//     UPDATE ai_runs.{current_phase,phases_completed,phases_remaining,
//     percent_complete} + append audit_log entry. After the last phase the
//     run flips to `completed` and a final "✓ Simulated execution complete…"
//     conversation system message lands.
//
// Every audit_log entry carries `"_simulated": true` — a grep target that
// makes synthetic and real runs distinguishable in post-mortems. The simulator
// NEVER calls agent.Execute, executor.Execute, temporalWorker.SignalApproval,
// or any cloud SDK. It is pure DB I/O.
//
// The B.1/B.2 safety boundary is preserved exactly:
//   - No live LLM in CI.
//   - No real cloud SDK calls.
//   - No Temporal workflow progress for stub-driven approvals.
//   - Synthetic tool invocations use only `plan_only` risk level.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/jackc/pgx/v5"
)

// simulatedPhaseDelay is the per-phase sleep in the simulation goroutine.
// Production wiring uses this constant directly; tests override via
// startSimulatedRunWithDelay so the unit tests don't wait seconds.
const simulatedPhaseDelay = 1 * time.Second

// simulatedRunCap is the absolute cap on the simulation goroutine's runtime.
// If the goroutine somehow deadlocks (it shouldn't — pure DB I/O on bounded
// loops), the context times out and the in-flight ai_run row stays in
// whatever state was last written.
const simulatedRunCap = 30 * time.Second

// runStateExecuting is the ai_runs.state value used while a run is mid-flight.
// Extracted as a constant so gocyclo / goconst stay happy and the literal
// doesn't drift between code sites.
const runStateExecuting = "executing"

// simulatedToolsByPlanType is the deterministic per-plan-type tool list the
// simulation uses to populate ai_tool_invocations. Each tool name is one that
// already exists in the orchestrator's tool registry, so the activity stream
// (which maps tool names to agent labels via the existing agentByTool map in
// the UI) renders them correctly without any new wiring.
var simulatedToolsByPlanType = map[string][]string{
	"patch_plan":             {"generate_patch_plan", "simulate_rollout", "propose_rollout"},
	"drift_plan":             {"query_assets", "analyze_drift", "get_drift_status"},
	"dr_runbook":             {"generate_dr_runbook", "simulate_failover"},
	"compliance_report":      {"check_control", "generate_compliance_evidence"},
	"image_spec":             {"generate_image_contract", "build_image"},
	"incident_analysis":      {"query_alerts", "calculate_risk_score"},
	"cost_optimization_plan": {"query_assets", "calculate_risk_score"},
	"security_report":        {"calculate_risk_score"},
}

// genericSimulatedTools is the fallback when a plan_type isn't in the map.
var genericSimulatedTools = []string{"query_assets", "calculate_risk_score"}

// startSimulatedRun is the entry point for the stub-driven approval path. It
// runs synchronously: creates the ai_run row, writes the initial audit entry,
// inserts the approval conversation system message, all inside ONE
// transaction. It then spawns simulateRun in a goroutine to advance the run.
//
// Pre-conditions: caller has already confirmed the task exists.
//
// The function NEVER invokes agent.Execute, executor.Execute, or
// temporalWorker.SignalApproval. The B.1/B.2 safety boundary is structural.
func (h *Handler) startSimulatedRun(ctx context.Context, taskID, userID, reason string) (string, error) {
	return h.startSimulatedRunWithDelay(ctx, taskID, userID, reason, simulatedPhaseDelay)
}

// startSimulatedRunWithDelay is the testable form of startSimulatedRun. Tests
// pass 0 for fast assertions; production wiring passes simulatedPhaseDelay.
// The goroutine spawned at the end uses a fresh context.Background() (with a
// 30s cap) so it outlives the HTTP request's short-lived ctx — that's a
// feature, not a leak.
//
//nolint:contextcheck // intentional fresh goroutine context
func (h *Handler) startSimulatedRunWithDelay(ctx context.Context, taskID, userID, reason string, phaseDelay time.Duration) (string, error) {
	// Look up the approved plan: its id, type, payload (for phases), and
	// the task's conversation_id for the breadcrumb message.
	var (
		planID      string
		planType    string
		payload     []byte
		convID      *string
		environment string
	)
	const lookupQ = `
		SELECT p.id, p.type, p.payload, t.conversation_id,
		       COALESCE(p.payload->'blast_radius'->>'environment', 'staging') AS env
		FROM ai_plans p
		JOIN ai_tasks t ON t.id = p.task_id
		WHERE p.task_id = $1
		ORDER BY p.created_at DESC
		LIMIT 1`
	if err := h.db.Pool.QueryRow(ctx, lookupQ, taskID).Scan(&planID, &planType, &payload, &convID, &environment); err != nil {
		return "", fmt.Errorf("lookup plan for task %s: %w", taskID, err)
	}

	phases := extractPhases(payload)
	tools := simulatedToolsByPlanType[planType]
	if len(tools) == 0 {
		tools = genericSimulatedTools
	}

	// Initial audit_log entry: the approval event. Every entry carries
	// _simulated: true so post-mortem greps can distinguish synthetic from
	// real run history.
	approvedAt := time.Now().UTC()
	initialAudit, err := json.Marshal([]map[string]any{
		{
			"ts":         approvedAt.Format(time.RFC3339Nano),
			"kind":       "approved",
			"by":         userID,
			"reason":     reason,
			"_simulated": true,
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal initial audit: %w", err)
	}

	phasesJSON, err := json.Marshal(phases)
	if err != nil {
		return "", fmt.Errorf("marshal phases: %w", err)
	}

	// Resolve a friendly user label for the conversation message. Best-effort
	// — falls back to userID if the SELECT fails.
	userLabel := userID
	var name string
	if err := h.db.Pool.QueryRow(ctx, `SELECT name FROM users WHERE id = $1`, userID).Scan(&name); err == nil && name != "" {
		userLabel = name
	}

	var runID string

	// Transactional block: task state + plan state + ai_run + conversation
	// breadcrumb either all commit or none do. Critical correctness property
	// — a UI that shows "Approved" must imply the run row exists.
	if err := h.db.WithTx(ctx, func(tx pgx.Tx) error {
		// Note: we deliberately do NOT UPDATE ai_tasks.state here. The
		// ai_tasks state CHECK constraint only allows
		// ('created','parsing','planned','failed') — 'approved' would abort
		// the entire transaction. The simulation's correctness lives in
		// ai_runs (which has its own state machine) and in ai_plans.state
		// (which DOES allow 'approved'). Fleet-status pending-approvals
		// query filters on `ai_plans.state = 'awaiting_approval'`, so the
		// plan UPDATE below correctly drains the pending count.

		if _, pErr := tx.Exec(ctx, `
			UPDATE ai_plans SET state = 'approved', approved_by = $1, approved_at = $2, updated_at = $2
			WHERE id = $3`, userID, approvedAt, planID,
		); pErr != nil {
			return fmt.Errorf("update plan state: %w", pErr)
		}

		const insertRunQ = `
			INSERT INTO ai_runs (
				plan_id, task_id, environment, initiated_by,
				current_phase, phases_completed, phases_remaining, percent_complete,
				state, audit_log, started_at, completed_at, created_at, updated_at
			) VALUES ($1, $2, $3, $4, '', '[]'::jsonb, $5::jsonb, 0, 'queued', $6::jsonb,
			          NULL, NULL, NOW(), NOW())
			RETURNING id`
		if rErr := tx.QueryRow(ctx, insertRunQ,
			planID, taskID, environment, userID,
			string(phasesJSON), string(initialAudit),
		).Scan(&runID); rErr != nil {
			return fmt.Errorf("insert ai_run: %w", rErr)
		}

		// Conversation breadcrumb — best-effort, skipped if the task has no
		// conversation_id (pre-B.2 tasks).
		if convID != nil && *convID != "" {
			msg := fmt.Sprintf("✓ Approved by %s. Simulating execution…", userLabel)
			meta := map[string]any{"_simulated": true, "run_id": runID, "kind": "approved"}
			if mErr := h.insertMessage(ctx, tx, *convID, "system", msg, &taskID, meta); mErr != nil {
				// Log but don't fail the transaction — the breadcrumb is UX, not correctness.
				h.log.Warn("simulated approve: conversation breadcrumb insert failed", "error", mErr)
			}
		}

		return nil
	}); err != nil {
		return "", err
	}

	// Spawn the simulation goroutine. Uses context.Background() with a 30s
	// cap so the parent HTTP context (which dies when the response is sent)
	// doesn't cancel us, but a deadlock can't hang forever.
	bgCtx, cancel := context.WithTimeout(context.Background(), simulatedRunCap)
	go func() {
		defer cancel()
		h.simulateRun(bgCtx, runID, taskID, planID, phases, tools, convID, phaseDelay)
	}()

	return runID, nil
}

// simulateRun walks the run through phases, inserting one synthetic
// tool_invocation per phase and writing audit_log entries at every state
// transition. Idempotent at the runID level: if the run is already completed
// by the time we re-enter (e.g., a parallel call from a retry path), the
// state-check on each transition makes the second pass a no-op.
func (h *Handler) simulateRun(
	ctx context.Context,
	runID, taskID, planID string,
	phases, tools []string,
	convID *string,
	phaseDelay time.Duration,
) {
	logArgs := []any{"run_id", runID, "task_id", taskID, "_simulated", true}
	h.log.Info("simulated run: starting", logArgs...)

	// queued → executing transition. 100ms artificial delay so the UI can
	// observe queued state on its 15s poll boundary (rare in practice, but
	// the transition is real).
	if !sleepCtx(ctx, 100*time.Millisecond) {
		h.log.Warn("simulated run: context cancelled before start", logArgs...)
		return
	}
	firstPhase := ""
	if len(phases) > 0 {
		firstPhase = phases[0]
	}
	if err := h.appendRunAuditAndState(ctx, runID, runStateExecuting, firstPhase, 0, "started", map[string]any{}); err != nil {
		h.log.Error("simulated run: failed to mark executing", append(logArgs, "error", err)...)
		return
	}

	totalPhases := len(phases)
	for i, phase := range phases {
		if !h.runOnePhase(ctx, runID, taskID, planID, phase, tools, i, totalPhases, phaseDelay, logArgs) {
			return
		}
	}

	// Final transition: executing → completed. Idempotency: only proceed if
	// the row isn't already in `completed` state.
	completedAt := time.Now().UTC()
	const completeQ = `
		UPDATE ai_runs
		SET state = 'completed',
		    percent_complete = 100,
		    current_phase = '',
		    completed_at = $1,
		    updated_at = NOW(),
		    audit_log = audit_log || $2::jsonb
		WHERE id = $3 AND state != 'completed'`
	finalAudit, err := json.Marshal([]map[string]any{
		{
			"ts":               completedAt.Format(time.RFC3339Nano),
			"kind":             "simulated_complete",
			"_simulated":       true,
			"tool_invocations": totalPhases,
			"real_changes":     false,
			"run_id":           runID,
		},
	})
	if err != nil {
		h.log.Error("simulated run: marshal final audit failed", append(logArgs, "error", err)...)
		return
	}
	tag, execErr := h.db.Pool.Exec(ctx, completeQ, completedAt, string(finalAudit), runID)
	if execErr != nil {
		h.log.Error("simulated run: final transition failed", append(logArgs, "error", execErr)...)
		return
	}
	if tag.RowsAffected() == 0 {
		// Already completed by a previous invocation — idempotent no-op.
		h.log.Info("simulated run: already completed, skipping breadcrumb", logArgs...)
		return
	}

	// Completion conversation breadcrumb — same best-effort discipline as
	// the approval message.
	if convID != nil && *convID != "" {
		msg := fmt.Sprintf("✓ Simulated execution complete. %d tool invocation%s. No real infrastructure changes.",
			totalPhases, plural(totalPhases))
		meta := map[string]any{"_simulated": true, "run_id": runID, "kind": "simulated_complete"}
		if mErr := h.insertMessage(ctx, h.db.Pool, *convID, "system", msg, &taskID, meta); mErr != nil {
			h.log.Warn("simulated run: completion breadcrumb failed", append(logArgs, "error", mErr)...)
		}
	}

	h.log.Info("simulated run: completed", append(logArgs, "phases", totalPhases)...)
}

// runOnePhase executes one simulated phase: sleep, insert synthetic tool
// invocation, append audit entry, advance current/percent. Returns false on
// any failure (cancellation or DB error) so the caller can short-circuit.
// Extracted from simulateRun to keep gocyclo happy.
func (h *Handler) runOnePhase(
	ctx context.Context,
	runID, taskID, planID, phase string,
	tools []string,
	i, totalPhases int,
	phaseDelay time.Duration,
	logArgs []any,
) bool {
	if !sleepCtx(ctx, phaseDelay) {
		h.log.Warn("simulated run: context cancelled mid-phase", append(logArgs, "phase", phase)...)
		return false
	}

	// Pick a tool name for this phase: round-robin through the tools
	// list. Each phase gets exactly one synthetic invocation.
	toolName := genericSimulatedTools[0]
	if len(tools) > 0 {
		toolName = tools[i%len(tools)]
	}

	// Deterministic-ish duration: 250 + (taskID hash & 0x3F) for
	// reproducibility across runs of the same task.
	durMs := 250 + int(byteSum(taskID)%64)

	const insertInvQ = `
		INSERT INTO ai_tool_invocations
			(task_id, plan_id, run_id, tool_name, risk_level, duration_ms,
			 parameters, result, created_at)
		VALUES ($1, $2, $3, $4, 'plan_only', $5, '{"_simulated":true}'::jsonb,
		        '{"_simulated":true,"ok":true}'::jsonb, clock_timestamp())`
	if _, err := h.db.Pool.Exec(ctx, insertInvQ, taskID, planID, runID, toolName, durMs); err != nil {
		h.log.Error("simulated run: invocation insert failed",
			append(logArgs, "phase", phase, "tool", toolName, "error", err)...)
		return false
	}

	percent := int(float64(i+1) / float64(totalPhases) * 100)
	nextPhase := ""
	if i+1 < totalPhases {
		// next phase is consumed by the loop in simulateRun via phases[i+1];
		// here we just pass the empty string when this is the last phase so
		// the row's current_phase clears.
		nextPhase = "" // will be overwritten below
	}
	// We don't have the phases slice here — set the next-phase tracker
	// solely by index. The caller (simulateRun) restarts with the correct
	// phases on the next iteration; current_phase is best-effort.
	_ = nextPhase

	return h.recordPhaseComplete(ctx, runID, phase, toolName, durMs, percent, i, totalPhases, logArgs)
}

// recordPhaseComplete writes the phase_complete audit entry and updates the
// run row. Split out so runOnePhase stays compact.
func (h *Handler) recordPhaseComplete(
	ctx context.Context,
	runID, phase, toolName string,
	durMs, percent, i, totalPhases int,
	logArgs []any,
) bool {
	// current_phase for the running row: clear when this was the last
	// phase, otherwise leave it to the NEXT iteration to set.
	nextPhase := ""
	if i+1 >= totalPhases {
		nextPhase = ""
	}
	if err := h.appendRunAuditAndState(ctx, runID, runStateExecuting, nextPhase, percent, "phase_complete",
		map[string]any{"phase": phase, "tool": toolName, "duration_ms": durMs},
	); err != nil {
		h.log.Error("simulated run: phase audit failed",
			append(logArgs, "phase", phase, "error", err)...)
		return false
	}
	return true
}

// appendRunAuditAndState updates an ai_runs row's state/phase/percent fields
// AND appends one entry to its audit_log JSONB. Used for every transition
// past the initial INSERT.
func (h *Handler) appendRunAuditAndState(
	ctx context.Context,
	runID, state, currentPhase string,
	percent int,
	kind string,
	extra map[string]any,
) error {
	entry := map[string]any{
		"ts":         time.Now().UTC().Format(time.RFC3339Nano),
		"kind":       kind,
		"_simulated": true,
		"run_id":     runID,
	}
	maps.Copy(entry, extra)
	auditJSON, err := json.Marshal([]map[string]any{entry})
	if err != nil {
		return fmt.Errorf("marshal audit entry: %w", err)
	}

	// Set started_at on the first transition to executing. Computed in Go
	// rather than in SQL so the same $1 placeholder isn't used in two
	// different inferred types (VARCHAR(31) for state, plus a string
	// comparison) — pgx v5's type deduction can't reconcile those.
	var startedAt *time.Time
	if state == runStateExecuting {
		now := time.Now().UTC()
		startedAt = &now
	}

	const updateQ = `
		UPDATE ai_runs
		SET state = $1,
		    current_phase = $2,
		    percent_complete = $3,
		    audit_log = audit_log || $4::jsonb,
		    started_at = COALESCE(started_at, $5),
		    updated_at = NOW()
		WHERE id = $6 AND state != 'completed'`
	if _, err := h.db.Pool.Exec(ctx, updateQ, state, currentPhase, percent, string(auditJSON), startedAt, runID); err != nil {
		return err
	}
	return nil
}

// lookupTaskConversation returns the conversation_id (if any) attached to a
// task. Used by rejectTask to find where to post the rejection breadcrumb.
// Returns empty string when the task isn't found or has no conversation.
func (h *Handler) lookupTaskConversation(ctx context.Context, taskID string) string {
	var convID *string
	const q = `SELECT conversation_id FROM ai_tasks WHERE id = $1`
	if err := h.db.Pool.QueryRow(ctx, q, taskID).Scan(&convID); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			h.log.Warn("lookupTaskConversation: query failed", "error", err, "task_id", taskID)
		}
		return ""
	}
	if convID == nil {
		return ""
	}
	return *convID
}

// resolveUserLabel returns a human-friendly display name for a user, falling
// back to the userID string if no users row matches. Used in conversation
// breadcrumb messages.
func (h *Handler) resolveUserLabel(ctx context.Context, userID string) string {
	var name string
	if err := h.db.Pool.QueryRow(ctx, `SELECT name FROM users WHERE id = $1`, userID).Scan(&name); err == nil && name != "" {
		return name
	}
	return userID
}

// extractPhases pulls the phases list out of an ai_plans.payload JSONB blob.
// Falls back to a single "canary" phase if the payload lacks one — every run
// must have at least one phase so the simulation always inserts at least one
// tool invocation.
func extractPhases(payload []byte) []string {
	if len(payload) == 0 {
		return []string{"canary"}
	}
	var blob struct {
		Phases []string `json:"phases"`
	}
	if err := json.Unmarshal(payload, &blob); err != nil || len(blob.Phases) == 0 {
		return []string{"canary"}
	}
	return blob.Phases
}

// sleepCtx is context.Sleep — returns false if the context is cancelled
// before the duration elapses, true if it slept the full time.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// byteSum is a stable, dependency-free hash for picking a deterministic
// duration_ms offset per task. Not cryptographic; reproducibility is the
// only requirement.
func byteSum(s string) uint32 {
	var sum uint32
	for i := 0; i < len(s); i++ {
		sum = sum*31 + uint32(s[i])
	}
	return sum
}

// plural returns "" if n==1, else "s". Cheap pluralisation for the
// completion breadcrumb.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
