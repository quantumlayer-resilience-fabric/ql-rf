// Conversation memory tests — Mission Control Phase B.2 / AI-003.
//
// Five tests exercise the lifecycle logic in conversations.go without any HTTP
// surface — pure DB I/O plus one pure-function table for the synthesized
// assistant message. The DB-backed tests skip cleanly when no database is
// available (same pattern as cve_alerts_query_test.go).

package handlers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/agents"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/validation"
)

// convTestHandler returns a Handler whose only wired dependency is a real DB
// pool plus a quiet logger. The conversation helpers don't need anything else.
func convTestHandler(t *testing.T) *Handler {
	t.Helper()
	db := handlerTestDB(t)
	t.Cleanup(db.Close)
	return &Handler{
		db:  db,
		cfg: &config.Config{Env: "test", Orchestrator: config.OrchestratorConfig{DevMode: true}},
		log: logger.New("error", "text"),
	}
}

// seedConvOrgUser inserts a throwaway org + user the conversation tests can
// reference. Registers cleanup that cascades through the conversation tables.
func seedConvOrgUser(t *testing.T, h *Handler) (orgID, userID string) {
	t.Helper()
	pool := h.db.Pool
	ctx := context.Background()
	orgID = uuid.NewString()
	userID = uuid.NewString()

	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, "DELETE FROM organizations WHERE id = $1", orgID)
		// users are FK-cascaded via organizations; conversations cascade via
		// users; messages cascade via conversations.
	})

	if _, err := pool.Exec(ctx,
		`INSERT INTO organizations (id, name, slug) VALUES ($1, $2, $3)`,
		orgID, "B.2 test org", "b2-"+uuid.NewString()[:8]); err != nil {
		t.Fatalf("seed org: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, external_id, email, name, role, org_id)
		 VALUES ($1, $2, $3, $4, 'admin', $5)`,
		userID, "b2-test-"+userID[:8], userID[:8]+"@b2.test", "B2 Test User", orgID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return orgID, userID
}

// TestEnsureActiveConversation_CreatesNewWhenNoneExist — empty DB + user;
// assert one new conversation row, state=active, title=truncated prompt,
// isNew=true.
func TestEnsureActiveConversation_CreatesNewWhenNoneExist(t *testing.T) {
	h := convTestHandler(t)
	orgID, userID := seedConvOrgUser(t, h)
	ctx := context.Background()

	prompt := "Patch CVE-2024-3094 on production assets"
	convID, isNew, err := h.ensureActiveConversation(ctx, h.db.Pool, orgID, userID, prompt)
	if err != nil {
		t.Fatalf("ensureActiveConversation: %v", err)
	}
	if !isNew {
		t.Errorf("isNew = false, want true (no prior conversation)")
	}
	if _, parseErr := uuid.Parse(convID); parseErr != nil {
		t.Errorf("returned id is not a UUID: %q", convID)
	}

	// Verify the row landed with expected state + title.
	var state, title string
	if err := h.db.Pool.QueryRow(ctx,
		`SELECT state, title FROM ai_conversations WHERE id = $1`, convID,
	).Scan(&state, &title); err != nil {
		t.Fatalf("SELECT inserted conversation: %v", err)
	}
	if state != "active" {
		t.Errorf("state = %q, want active", state)
	}
	if title != prompt {
		t.Errorf("title = %q, want exact prompt (no truncation under 80c)", title)
	}
}

// TestEnsureActiveConversation_AppendsToActiveWithinWindow — pre-insert a
// conversation with updated_at 5 min ago; call ensure; assert the same id
// is returned, isNew=false, and updated_at has been bumped.
func TestEnsureActiveConversation_AppendsToActiveWithinWindow(t *testing.T) {
	h := convTestHandler(t)
	orgID, userID := seedConvOrgUser(t, h)
	ctx := context.Background()

	// Pre-insert a recent conversation.
	var existingID string
	if err := h.db.Pool.QueryRow(ctx, `
		INSERT INTO ai_conversations (org_id, created_by, title, state, updated_at)
		VALUES ($1, $2, 'pre-existing', 'active', NOW() - INTERVAL '5 minutes')
		RETURNING id`,
		orgID, userID,
	).Scan(&existingID); err != nil {
		t.Fatalf("seed existing conversation: %v", err)
	}

	var beforeUpdated time.Time
	if err := h.db.Pool.QueryRow(ctx,
		`SELECT updated_at FROM ai_conversations WHERE id = $1`, existingID,
	).Scan(&beforeUpdated); err != nil {
		t.Fatalf("read updated_at: %v", err)
	}

	got, isNew, err := h.ensureActiveConversation(ctx, h.db.Pool, orgID, userID, "follow-up prompt")
	if err != nil {
		t.Fatalf("ensureActiveConversation: %v", err)
	}
	if got != existingID {
		t.Errorf("returned id = %s, want existing %s", got, existingID)
	}
	if isNew {
		t.Errorf("isNew = true, want false (should have appended)")
	}

	var afterUpdated time.Time
	if err := h.db.Pool.QueryRow(ctx,
		`SELECT updated_at FROM ai_conversations WHERE id = $1`, existingID,
	).Scan(&afterUpdated); err != nil {
		t.Fatalf("read updated_at: %v", err)
	}
	if !afterUpdated.After(beforeUpdated) {
		t.Errorf("updated_at not bumped: before=%s after=%s", beforeUpdated, afterUpdated)
	}
}

// TestEnsureActiveConversation_CreatesNewWhenStale — pre-insert a conversation
// with updated_at 90 min ago (outside the 60-min window); call ensure; assert
// a new conversation id is returned (not the stale one), isNew=true.
func TestEnsureActiveConversation_CreatesNewWhenStale(t *testing.T) {
	h := convTestHandler(t)
	orgID, userID := seedConvOrgUser(t, h)
	ctx := context.Background()

	var staleID string
	if err := h.db.Pool.QueryRow(ctx, `
		INSERT INTO ai_conversations (org_id, created_by, title, state, updated_at)
		VALUES ($1, $2, 'stale', 'active', NOW() - INTERVAL '90 minutes')
		RETURNING id`,
		orgID, userID,
	).Scan(&staleID); err != nil {
		t.Fatalf("seed stale conversation: %v", err)
	}

	got, isNew, err := h.ensureActiveConversation(ctx, h.db.Pool, orgID, userID, "new session")
	if err != nil {
		t.Fatalf("ensureActiveConversation: %v", err)
	}
	if !isNew {
		t.Errorf("isNew = false, want true (stale conv should be ignored)")
	}
	if got == staleID {
		t.Errorf("returned stale conv id %s, expected fresh one", staleID)
	}
}

// TestEnsureActiveConversation_IsolatesByUser — pre-insert a recent
// conversation for user A; call ensure for user B; assert user B gets a new
// conversation and user A's row is untouched.
func TestEnsureActiveConversation_IsolatesByUser(t *testing.T) {
	h := convTestHandler(t)
	orgID, userA := seedConvOrgUser(t, h)
	// Add a second user in the same org.
	userB := uuid.NewString()
	if _, err := h.db.Pool.Exec(context.Background(),
		`INSERT INTO users (id, external_id, email, name, role, org_id)
		 VALUES ($1, $2, $3, 'B', 'admin', $4)`,
		userB, "b-"+userB[:8], userB[:8]+"@b.test", orgID,
	); err != nil {
		t.Fatalf("seed userB: %v", err)
	}

	ctx := context.Background()
	var userAConv string
	if err := h.db.Pool.QueryRow(ctx, `
		INSERT INTO ai_conversations (org_id, created_by, title, state, updated_at)
		VALUES ($1, $2, 'A-conv', 'active', NOW() - INTERVAL '2 minutes')
		RETURNING id`,
		orgID, userA,
	).Scan(&userAConv); err != nil {
		t.Fatalf("seed userA conv: %v", err)
	}

	got, isNew, err := h.ensureActiveConversation(ctx, h.db.Pool, orgID, userB, "B's prompt")
	if err != nil {
		t.Fatalf("ensureActiveConversation: %v", err)
	}
	if !isNew {
		t.Errorf("isNew = false, want true (different user)")
	}
	if got == userAConv {
		t.Errorf("user B got user A's conversation: %s", got)
	}

	// userA's conversation should not have moved.
	var aState string
	if err := h.db.Pool.QueryRow(ctx,
		`SELECT state FROM ai_conversations WHERE id = $1`, userAConv,
	).Scan(&aState); err != nil {
		t.Fatalf("read userA conv: %v", err)
	}
	if aState != "active" {
		t.Errorf("userA conv state changed to %q", aState)
	}
}

// TestSynthesizeAssistantMessage — pure-function table-driven test. Asserts
// that the synthesized text references task_type and the "Awaiting your
// approval" tail, and NEVER contains the literal word "stub" regardless of
// inputs. No DB needed.
func TestSynthesizeAssistantMessage(t *testing.T) {
	qs := &validation.QualityScore{Total: 87}

	cases := []struct {
		name        string
		spec        *agents.TaskSpec
		result      *agents.AgentResult
		wantSubstrs []string
	}{
		{
			name: "cve intent → patch_rollout",
			spec: &agents.TaskSpec{
				ID:           "t1",
				TaskType:     agents.TaskTypePatchRollout,
				UserIntent:   "Patch CVE-2024-3094",
				Goal:         "Stub-generated plan for: ...",
				RiskLevel:    "high",
				HITLRequired: true,
			},
			result:      &agents.AgentResult{Status: agents.AgentStatusPendingApproval},
			wantSubstrs: []string{"patch_rollout", "Patch CVE-2024-3094", "Risk: high", "HITL required", "Quality: 87/100", "Awaiting your approval"},
		},
		{
			name: "drift intent",
			spec: &agents.TaskSpec{
				ID:           "t2",
				TaskType:     agents.TaskTypeDriftRemediation,
				UserIntent:   "Analyze drift across azure production sites",
				Goal:         "drift remediation",
				RiskLevel:    "medium",
				HITLRequired: false,
			},
			result:      &agents.AgentResult{Status: agents.AgentStatusPendingApproval},
			wantSubstrs: []string{"drift_remediation", "Analyze drift", "Risk: medium", "Quality: 87/100", "Awaiting your approval"},
		},
		{
			name: "agent error path",
			spec: &agents.TaskSpec{
				ID:         "t3",
				TaskType:   agents.TaskTypePatchRollout,
				UserIntent: "ignored",
			},
			result: &agents.AgentResult{
				Status: agents.AgentStatusFailed,
				Errors: []string{"agent timed out"},
			},
			wantSubstrs: []string{"Couldn't draft a plan", "agent timed out"},
		},
		{
			name: "long intent gets truncated",
			spec: &agents.TaskSpec{
				ID:         "t4",
				TaskType:   agents.TaskTypeDriftRemediation,
				UserIntent: strings.Repeat("very-long-user-prompt-", 10),
			},
			result:      &agents.AgentResult{Status: agents.AgentStatusPendingApproval},
			wantSubstrs: []string{"drift_remediation", "…"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := synthesizeAssistantMessage(tc.spec, tc.result, qs)

			// Hard rule: the user-visible string must never contain the
			// literal word "stub" regardless of inputs. Audit marker stays in
			// metadata, not in the rendered bubble.
			if strings.Contains(strings.ToLower(out), "stub") {
				t.Errorf("output contains 'stub' (forbidden in user-visible text): %q", out)
			}
			// Output must be plain text — never raw JSON.
			if strings.HasPrefix(strings.TrimSpace(out), "{") {
				t.Errorf("output looks like JSON: %q", out)
			}
			for _, want := range tc.wantSubstrs {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q\noutput: %s", want, out)
				}
			}
		})
	}
}
