// Package handlers — conversation memory for Mission Control (Phase B.2 / AI-003).
//
// A conversation is a persistent thread of messages between a user and Mission
// Control. Submissions made within a 60-minute window for the same user append
// to the same conversation; outside that window a new one starts. Lifecycle is
// server-decided — no UI affordance is required to start or end a conversation
// in B.2 (that lands in B.3).
//
// This file owns:
//   - ensureActiveConversation: lifecycle decision (append vs. new) under a
//     SELECT FOR UPDATE so concurrent submits from the same user serialize on
//     the row lock and at most one wins the append.
//   - insertMessage: typed insert for ai_conversation_messages. Uses
//     clock_timestamp() so successive inserts inside the same transaction get
//     strictly increasing created_at values.
//   - synthesizeAssistantMessage: deterministic, server-side projection of a
//     TaskSpec + AgentResult into a one- or two-sentence assistant summary.
//     Decoupled from the LLM provider — same code path for stub and live LLMs.
//     Auditable: text is a function of validated agent output, not arbitrary
//     LLM-controlled string. Never contains the literal word "stub".
//   - listConversations / getConversationMessages: read-only HTTP handlers.
//
// The B.1 safety boundary is preserved: this file never invokes an agent,
// an LLM, or a cloud SDK. It only writes to ai_conversations and
// ai_conversation_messages (always inside the caller's transaction) and reads
// from those tables in the GET handlers.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/agents"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/middleware"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/validation"
)

// conversationAppendWindow is the duration during which successive submissions
// from the same user fold into the same conversation. Outside this window a
// new conversation is created. Hard-coded for B.2; envify in a later phase if
// product feedback requires.
const conversationAppendWindow = 60 * time.Minute

// missionControlDevUserID is the seeded dev-user UUID used when the
// middleware injects the literal "dev-user" placeholder. Matches the seed
// in scripts/seed-e2e-data/main.go (seedMissionControl).
const missionControlDevUserID = "e0000000-0000-0000-0000-000000000001"

// dbExec is the small subset of pgx methods both *pgxpool.Pool and pgx.Tx
// satisfy. Functions that may run inside a transaction take dbExec rather
// than the concrete type, so executeTask can pass a pgx.Tx while the
// read-only GET handlers can pass the bare pool.
type dbExec interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// ConversationSummary is the JSON shape returned by
// GET /api/v1/ai/conversations.
type ConversationSummary struct {
	ID           string `json:"id"`
	Title        string `json:"title,omitempty"`
	State        string `json:"state"`
	MessageCount int    `json:"message_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// ConversationMessage is the JSON shape for a single message in
// GET /api/v1/ai/conversations/{id}/messages. The raw LLM content lives under
// metadata.raw_llm_content for audit only; the UI renders `content`.
type ConversationMessage struct {
	ID        string         `json:"id"`
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	TaskID    *string        `json:"task_id,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt string         `json:"created_at"`
}

// ensureActiveConversation finds the user's currently-active conversation or
// creates a new one if none is within the append window. The SELECT takes a
// row lock (FOR UPDATE) so concurrent submits from the same user serialize on
// the row and at most one wins the append.
//
// All work happens inside the caller's transaction (db). The function NEVER
// touches an agent, LLM, or cloud SDK — it is pure DB I/O.
func (h *Handler) ensureActiveConversation(ctx context.Context, db dbExec, orgID, userID, prompt string) (conversationID string, isNew bool, err error) {
	cutoff := time.Now().Add(-conversationAppendWindow)

	const selectQ = `
		SELECT id
		FROM ai_conversations
		WHERE org_id = $1
		  AND created_by = $2
		  AND state = 'active'
		  AND updated_at >= $3
		ORDER BY updated_at DESC
		LIMIT 1
		FOR UPDATE`

	var existing string
	err = db.QueryRow(ctx, selectQ, orgID, userID, cutoff).Scan(&existing)
	if err == nil {
		// Active conversation in-window — bump updated_at and reuse.
		if _, uErr := db.Exec(ctx, `UPDATE ai_conversations SET updated_at = NOW() WHERE id = $1`, existing); uErr != nil {
			return "", false, fmt.Errorf("bump conversation updated_at: %w", uErr)
		}
		return existing, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", false, fmt.Errorf("lookup active conversation: %w", err)
	}

	title := truncateTitle(prompt, 80)
	const insertQ = `
		INSERT INTO ai_conversations (org_id, created_by, title, state)
		VALUES ($1, $2, $3, 'active')
		RETURNING id`
	var newID string
	if iErr := db.QueryRow(ctx, insertQ, orgID, userID, title).Scan(&newID); iErr != nil {
		return "", false, fmt.Errorf("insert conversation: %w", iErr)
	}
	return newID, true, nil
}

// insertMessage writes one row to ai_conversation_messages. Uses
// clock_timestamp() rather than the column DEFAULT (NOW()) — within a single
// transaction NOW() is constant, so successive inserts would share a
// created_at value and dock-thread ordering would be nondeterministic.
//
// Always called inside the caller's transaction. taskID is optional.
func (h *Handler) insertMessage(ctx context.Context, db dbExec, convID, role, content string, taskID *string, metadata map[string]any) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal message metadata: %w", err)
	}
	const insertQ = `
		INSERT INTO ai_conversation_messages
			(conversation_id, role, content, task_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, clock_timestamp())`
	if _, err := db.Exec(ctx, insertQ, convID, role, content, taskID, metaJSON); err != nil {
		return fmt.Errorf("insert conversation message: %w", err)
	}
	return nil
}

// synthesizeAssistantMessage projects a TaskSpec + AgentResult + quality score
// into a one- or two-sentence assistant summary. The output is rendered as
// the assistant chat bubble in the dock.
//
// Properties:
//   - Deterministic. Same inputs → same string.
//   - Never contains JSON.
//   - Never contains the literal word "stub" — the audit marker stays in
//     ai_conversation_messages.metadata and in logs, not in user-visible text.
//   - Decoupled from the LLM provider. The future live-LLM phase swaps the
//     provider without touching this function.
func synthesizeAssistantMessage(spec *agents.TaskSpec, result *agents.AgentResult, qs *validation.QualityScore) string {
	if spec == nil || result == nil {
		return "Couldn't draft a plan: orchestrator returned no result."
	}
	if len(result.Errors) > 0 {
		return fmt.Sprintf("Couldn't draft a plan: %s.", result.Errors[0])
	}

	// Prefer UserIntent (what the user actually typed) over Goal (the meta
	// engine's normalised re-statement). The assistant should echo the user's
	// own words back to them — that's what makes a chat exchange feel like
	// memory, not paraphrase. Falls back to Goal if UserIntent is empty.
	goal := strings.TrimSpace(spec.UserIntent)
	if goal == "" {
		goal = strings.TrimSpace(spec.Goal)
	}
	if len(goal) > 80 {
		goal = goal[:79] + "…"
	}

	taskType := strings.TrimSpace(string(spec.TaskType))
	if taskType == "" {
		taskType = "task"
	}

	risk := strings.TrimSpace(spec.RiskLevel)
	if risk == "" {
		risk = "medium"
	}

	hitlSuffix := ""
	if spec.HITLRequired {
		hitlSuffix = " (HITL required)"
	}

	parts := []string{
		fmt.Sprintf("Drafted plan-only %s for: %q.", taskType, goal),
		fmt.Sprintf("Risk: %s%s.", risk, hitlSuffix),
	}
	if qs != nil {
		parts = append(parts, fmt.Sprintf("Quality: %d/100.", qs.Total))
	}
	parts = append(parts, "Awaiting your approval.")
	return strings.Join(parts, " ")
}

// resolveUserID returns the caller's user UUID, applying the dev-user →
// seeded UUID mapping used elsewhere in this package. Extracted so both
// executeTask and the conversation handlers share one source of truth for
// the dev-mode identity dance.
func resolveUserID(ctx context.Context) string {
	userID := middleware.GetUserID(ctx)
	if userID == "" || userID == "dev-user" {
		userID = missionControlDevUserID
	}
	return userID
}

// truncateTitle returns prompt trimmed to at most limit characters, suffixed
// with an ellipsis if it was cut.
func truncateTitle(prompt string, limit int) string {
	t := strings.TrimSpace(prompt)
	if len(t) <= limit {
		return t
	}
	return t[:limit-1] + "…"
}

// =============================================================================
// HTTP handlers
// =============================================================================

// listConversations responds to GET /api/v1/ai/conversations.
// Filtered to the caller's user; limited by the `limit` query parameter
// (default 1, max 50). Read-only — never invokes an agent, LLM, or cloud SDK.
func (h *Handler) listConversations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := resolveUserID(ctx)

	limit := 1
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			if parsed > 50 {
				parsed = 50
			}
			limit = parsed
		}
	}

	const q = `
		SELECT c.id, c.title, c.state,
		       COUNT(m.id)::int AS message_count,
		       c.created_at, c.updated_at
		FROM ai_conversations c
		LEFT JOIN ai_conversation_messages m ON m.conversation_id = c.id
		WHERE c.created_by = $1
		GROUP BY c.id
		ORDER BY c.updated_at DESC
		LIMIT $2`
	rows, err := h.db.Pool.Query(ctx, q, userID, limit)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to list conversations", err)
		return
	}
	defer rows.Close()

	conversations := []ConversationSummary{}
	for rows.Next() {
		var (
			s         ConversationSummary
			title     *string
			createdAt time.Time
			updatedAt time.Time
		)
		if err := rows.Scan(&s.ID, &title, &s.State, &s.MessageCount, &createdAt, &updatedAt); err != nil {
			h.log.Warn("listConversations: scan failed", "error", err)
			continue
		}
		if title != nil {
			s.Title = *title
		}
		s.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		s.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		conversations = append(conversations, s)
	}

	h.respond(w, http.StatusOK, map[string]any{
		"conversations": conversations,
	})
}

// getConversationMessages responds to
// GET /api/v1/ai/conversations/{conversationID}/messages.
// Authorizes on org: 404 if the conversation isn't in the caller's org.
// Read-only.
func (h *Handler) getConversationMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	convID := chi.URLParam(r, "conversationID")
	if convID == "" {
		h.respondError(w, http.StatusBadRequest, "conversation id is required", nil)
		return
	}

	orgID := middleware.GetOrgID(ctx)

	var (
		conv      ConversationSummary
		title     *string
		createdAt time.Time
		updatedAt time.Time
	)
	const convQ = `
		SELECT c.id, c.title, c.state, c.created_at, c.updated_at
		FROM ai_conversations c
		WHERE c.id = $1 AND c.org_id = $2`
	err := h.db.Pool.QueryRow(ctx, convQ, convID, orgID).Scan(&conv.ID, &title, &conv.State, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.respondError(w, http.StatusNotFound, "conversation not found", nil)
			return
		}
		h.respondError(w, http.StatusInternalServerError, "failed to fetch conversation", err)
		return
	}
	if title != nil {
		conv.Title = *title
	}
	conv.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	conv.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)

	const msgsQ = `
		SELECT id, role, content, task_id, metadata, created_at
		FROM ai_conversation_messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC`
	rows, err := h.db.Pool.Query(ctx, msgsQ, convID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to list messages", err)
		return
	}
	defer rows.Close()

	messages := []ConversationMessage{}
	for rows.Next() {
		var (
			m        ConversationMessage
			taskID   *string
			metadata []byte
			created  time.Time
		)
		if err := rows.Scan(&m.ID, &m.Role, &m.Content, &taskID, &metadata, &created); err != nil {
			h.log.Warn("getConversationMessages: scan failed", "error", err)
			continue
		}
		if taskID != nil {
			m.TaskID = taskID
		}
		if len(metadata) > 0 {
			meta := map[string]any{}
			if jErr := json.Unmarshal(metadata, &meta); jErr == nil {
				m.Metadata = meta
			}
		}
		m.CreatedAt = created.UTC().Format(time.RFC3339Nano)
		messages = append(messages, m)
	}
	conv.MessageCount = len(messages)

	h.respond(w, http.StatusOK, map[string]any{
		"conversation": conv,
		"messages":     messages,
	})
}
