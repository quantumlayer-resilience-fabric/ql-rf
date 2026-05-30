package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/middleware"
)

// FleetStatus is the response shape for GET /api/v1/ai/fleet/status. It is a
// small read-only aggregation that powers the Mission Control page header,
// pending-decisions rail and activity stream in one round-trip — the doc-blessed
// alternative to duplicating fleet-count and license-summary logic client-side
// (see docs/E2E-011-ai-mission-control.md §8 "Backend").
type FleetStatus struct {
	Agents            AgentCounts       `json:"agents"`
	PendingApprovals  []PendingDecision `json:"pending_approvals"`
	RecentInvocations []ToolInvocation  `json:"recent_invocations"`
	// RecentActivity is the unified activity feed introduced in Phase B.2.
	// Each element is discriminated by `kind` ("tool_invocation" or
	// "conversation_message"); the frontend dispatches on kind to render.
	// RecentInvocations is preserved untouched for backward compatibility
	// with Phase A tests and external consumers.
	RecentActivity       []ActivityEvent `json:"recent_activity"`
	ToolInvocationsToday int             `json:"tool_invocations_today"`
	LLMSpendTodayCents   int             `json:"llm_spend_today_cents"`
	LLMSpendBudgetCents  int             `json:"llm_spend_budget_cents"`
}

// ActivityEvent is one row of the unified activity feed. Tool invocations and
// conversation messages share this struct discriminated by Kind; fields
// specific to one kind stay zero (and absent from JSON via `,omitempty`) for
// the other. The frontend reads Kind first and renders accordingly.
type ActivityEvent struct {
	Kind      string `json:"kind"` // "tool_invocation" | "conversation_message"
	TaskID    string `json:"task_id,omitempty"`
	CreatedAt string `json:"created_at"`

	// tool_invocation fields
	ToolName   string `json:"tool_name,omitempty"`
	RiskLevel  string `json:"risk_level,omitempty"`
	DurationMs *int   `json:"duration_ms,omitempty"`

	// conversation_message fields
	ConversationID string `json:"conversation_id,omitempty"`
	MessageID      string `json:"message_id,omitempty"`
	Role           string `json:"role,omitempty"`
	ContentPreview string `json:"content_preview,omitempty"`
}

// AgentCounts summarizes the fleet for the status bar.
type AgentCounts struct {
	Total   int `json:"total"`
	Working int `json:"working"`
	Idle    int `json:"idle"`
	Blocked int `json:"blocked"`
}

// PendingDecision is a plan awaiting human approval, enriched with the data the
// Mission Control pending-decisions rail needs to render: quality score, OPA
// result, blast radius, environment.
type PendingDecision struct {
	TaskID            string `json:"task_id"`
	PlanID            string `json:"plan_id"`
	UserIntent        string `json:"user_intent"`
	PlanType          string `json:"plan_type"`
	QualityScore      *int   `json:"quality_score,omitempty"`
	OPAPass           bool   `json:"opa_pass"`
	BlastRadiusAssets int    `json:"blast_radius_assets"`
	Environment       string `json:"environment"`
	CreatedAt         string `json:"created_at"`
}

// ToolInvocation is one row of the activity stream — a recent tool call.
type ToolInvocation struct {
	TaskID     string `json:"task_id"`
	ToolName   string `json:"tool_name"`
	RiskLevel  string `json:"risk_level"`
	DurationMs *int   `json:"duration_ms,omitempty"`
	CreatedAt  string `json:"created_at"`
}

// Phase A budget cap (matches the "$50 today" placeholder in the design
// sketch). Future Phase C will read this from a per-org config table.
const defaultLLMBudgetCents = 5000

// getFleetStatus aggregates everything the Mission Control header + rails need
// for a single render: agent fleet counts, pending decisions enriched with
// quality + OPA, recent tool invocations for the activity stream, today's LLM
// spend. It is read-only and never invokes an agent or an LLM.
func (h *Handler) getFleetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.URL.Query().Get("org_id")
	if orgID == "" {
		orgID = middleware.GetOrgID(ctx)
	}

	resp := FleetStatus{
		Agents:              AgentCounts{},
		PendingApprovals:    []PendingDecision{},
		RecentInvocations:   []ToolInvocation{},
		RecentActivity:      []ActivityEvent{},
		LLMSpendBudgetCents: defaultLLMBudgetCents,
	}

	// Agent counts: total comes from the in-memory registry; working/blocked
	// derive from active ai_runs scoped to this org.
	if h.agents != nil {
		resp.Agents.Total = len(h.agents.ListAgents())
	}
	if err := h.db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE r.state = 'executing'),
			COUNT(*) FILTER (WHERE r.state IN ('paused', 'failed'))
		FROM ai_runs r
		JOIN ai_tasks t ON t.id = r.task_id
		WHERE t.org_id = $1`, orgID).Scan(&resp.Agents.Working, &resp.Agents.Blocked); err != nil {
		h.log.Warn("fleet status: agent counts query failed", "error", err)
	}
	resp.Agents.Idle = resp.Agents.Total - resp.Agents.Working - resp.Agents.Blocked
	if resp.Agents.Idle < 0 {
		resp.Agents.Idle = 0
	}

	// Pending approvals: plans in awaiting_approval state for this org, joined
	// with their task and validation/quality fields.
	pendingRows, err := h.db.Pool.Query(ctx, `
		SELECT
			t.id, p.id, t.user_intent, p.type, p.quality_score,
			COALESCE((p.validation->>'opa_valid')::bool, false) AS opa_pass,
			COALESCE((p.payload->'blast_radius'->>'assets')::int, 0) AS blast_assets,
			COALESCE(p.payload->'blast_radius'->>'environment', '') AS environment,
			p.created_at
		FROM ai_tasks t
		JOIN ai_plans p ON p.task_id = t.id
		WHERE t.org_id = $1 AND p.state = 'awaiting_approval'
		ORDER BY p.created_at DESC
		LIMIT 20`, orgID)
	if err != nil {
		h.log.Warn("fleet status: pending approvals query failed", "error", err)
	} else {
		defer pendingRows.Close()
		for pendingRows.Next() {
			var (
				d         PendingDecision
				quality   *int
				createdAt time.Time
			)
			if err := pendingRows.Scan(&d.TaskID, &d.PlanID, &d.UserIntent, &d.PlanType,
				&quality, &d.OPAPass, &d.BlastRadiusAssets, &d.Environment, &createdAt); err != nil {
				h.log.Warn("fleet status: scan pending failed", "error", err)
				continue
			}
			d.QualityScore = quality
			d.CreatedAt = createdAt.UTC().Format(time.RFC3339)
			resp.PendingApprovals = append(resp.PendingApprovals, d)
		}
	}

	// Recent tool invocations (activity stream).
	invRows, err := h.db.Pool.Query(ctx, `
		SELECT i.task_id, i.tool_name, i.risk_level, i.duration_ms, i.created_at
		FROM ai_tool_invocations i
		JOIN ai_tasks t ON t.id = i.task_id
		WHERE t.org_id = $1
		ORDER BY i.created_at DESC
		LIMIT 20`, orgID)
	if err != nil {
		h.log.Warn("fleet status: invocations query failed", "error", err)
	} else {
		defer invRows.Close()
		for invRows.Next() {
			var (
				inv       ToolInvocation
				dur       *int
				createdAt time.Time
			)
			if err := invRows.Scan(&inv.TaskID, &inv.ToolName, &inv.RiskLevel, &dur, &createdAt); err != nil {
				h.log.Warn("fleet status: scan invocation failed", "error", err)
				continue
			}
			inv.DurationMs = dur
			inv.CreatedAt = createdAt.UTC().Format(time.RFC3339)
			resp.RecentInvocations = append(resp.RecentInvocations, inv)
		}
	}

	// Unified activity feed (Phase B.2): tool invocations + user-role
	// conversation messages, merged on the DB side and ordered by created_at.
	// recent_invocations above is left intact for Phase A consumers; this is
	// purely additive. Assistant messages are intentionally filtered out —
	// they're conversation UX, not activity, and surfacing them would
	// double-count every submission.
	actRows, err := h.db.Pool.Query(ctx, `
		SELECT * FROM (
			SELECT 'tool_invocation'::text AS kind,
			       i.task_id::text         AS task_id,
			       i.tool_name             AS tool_name,
			       i.risk_level            AS risk_level,
			       i.duration_ms           AS duration_ms,
			       NULL::uuid              AS conversation_id,
			       NULL::uuid              AS message_id,
			       NULL::text              AS role,
			       NULL::text              AS content_preview,
			       i.created_at            AS created_at
			FROM ai_tool_invocations i
			JOIN ai_tasks t ON t.id = i.task_id
			WHERE t.org_id = $1
			UNION ALL
			SELECT 'conversation_message'::text AS kind,
			       m.task_id::text              AS task_id,
			       NULL::varchar                AS tool_name,
			       NULL::varchar                AS risk_level,
			       NULL::int                    AS duration_ms,
			       c.id                         AS conversation_id,
			       m.id                         AS message_id,
			       m.role                       AS role,
			       LEFT(m.content, 120)         AS content_preview,
			       m.created_at                 AS created_at
			FROM ai_conversation_messages m
			JOIN ai_conversations c ON c.id = m.conversation_id
			WHERE c.org_id = $1 AND m.role = 'user'
		) AS act
		ORDER BY created_at DESC
		LIMIT 20`, orgID)
	if err != nil {
		h.log.Warn("fleet status: recent_activity query failed", "error", err)
	} else {
		defer actRows.Close()
		for actRows.Next() {
			var (
				ev        ActivityEvent
				taskID    *string
				toolName  *string
				riskLevel *string
				durMs     *int
				convID    *string
				msgID     *string
				role      *string
				preview   *string
				createdAt time.Time
			)
			if err := actRows.Scan(&ev.Kind, &taskID, &toolName, &riskLevel, &durMs, &convID, &msgID, &role, &preview, &createdAt); err != nil {
				h.log.Warn("fleet status: scan activity failed", "error", err)
				continue
			}
			if taskID != nil {
				ev.TaskID = *taskID
			}
			if toolName != nil {
				ev.ToolName = *toolName
			}
			if riskLevel != nil {
				ev.RiskLevel = *riskLevel
			}
			ev.DurationMs = durMs
			if convID != nil {
				ev.ConversationID = *convID
			}
			if msgID != nil {
				ev.MessageID = *msgID
			}
			if role != nil {
				ev.Role = *role
			}
			if preview != nil {
				ev.ContentPreview = *preview
			}
			ev.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
			resp.RecentActivity = append(resp.RecentActivity, ev)
		}
	}

	// Tool-invocations-today and LLM-spend-today are both scoped to "today"
	// (UTC); the Mission Control header surfaces them.
	if err := h.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM ai_tool_invocations i
		JOIN ai_tasks t ON t.id = i.task_id
		WHERE t.org_id = $1 AND i.created_at::date = CURRENT_DATE`, orgID).Scan(&resp.ToolInvocationsToday); err != nil {
		h.log.Warn("fleet status: tool-invocations-today query failed", "error", err)
	}
	if err := h.db.Pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(total_cost_cents), 0)
		FROM llm_usage
		WHERE org_id = $1 AND timestamp::date = CURRENT_DATE`, orgID).Scan(&resp.LLMSpendTodayCents); err != nil {
		h.log.Warn("fleet status: llm-spend-today query failed", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.log.Warn("fleet status: encode failed", "error", err)
	}
}
