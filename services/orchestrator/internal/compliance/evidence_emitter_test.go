// PR #24 / CONN-004 — unit tests for the evidence emitter.
//
// Six tests cover:
//
//   1. lookupControlForTool — org-specific exact wins over global exact.
//   2. lookupControlForTool — exact wins over wildcard at same scope.
//   3. lookupControlForTool — unmapped tool returns "" without error.
//   4. EmitToolEvidence — synthetic params (_simulated:true) skipped.
//   5. EmitToolEvidence — content_hash matches SHA-256 of result JSON.
//   6. EmitToolEvidence — ai_tool_invocation_id populated on the row.
//
// Tests use a real DB (skip if unavailable) for the SQL-bound assertions.
// The DB-skip pattern matches handlers/cve_alerts_query_test.go.

//nolint:errcheck // tests intentionally skip checking some cleanup/exec errors

package compliance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/database"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// testDB returns a DB pool or skips the test. Mirrors handlerTestDB in
// services/orchestrator/internal/handlers but isolated here to keep this
// package self-contained.
func testDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Skipf("config load: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db, err := database.New(ctx, cfg.Database)
	if err != nil {
		t.Skip("DB not available")
	}
	if err := db.Health(ctx); err != nil {
		db.Close()
		t.Skip("DB not available")
	}
	t.Cleanup(db.Close)
	return db.Pool
}

// emitterFixture seeds an org + a compliance framework + control + a
// pre-existing ai_tool_invocations row. Returns all the IDs the tests
// need plus a cleanup-on-Cleanup t.Cleanup.
type emitterFixture struct {
	OrgID, ControlID, FrameworkID, UserID, TaskID, InvocationID string
}

func seedEmitterFixture(t *testing.T, pool *pgxpool.Pool) emitterFixture {
	t.Helper()
	ctx := context.Background()
	f := emitterFixture{
		OrgID:        uuid.NewString(),
		ControlID:    uuid.NewString(),
		FrameworkID:  uuid.NewString(),
		UserID:       uuid.NewString(),
		TaskID:       uuid.NewString(),
		InvocationID: uuid.NewString(),
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM compliance_evidence WHERE org_id = $1", f.OrgID)                                                                //nolint:errcheck
		_, _ = pool.Exec(context.Background(), "DELETE FROM tool_compliance_mappings WHERE org_id = $1 OR org_id IS NULL AND tool_name_pattern LIKE 'test_pr24%'", f.OrgID) //nolint:errcheck
		_, _ = pool.Exec(context.Background(), "DELETE FROM ai_tool_invocations WHERE id = $1", f.InvocationID)                                                             //nolint:errcheck
		_, _ = pool.Exec(context.Background(), "DELETE FROM ai_tasks WHERE id = $1", f.TaskID)                                                                              //nolint:errcheck
		_, _ = pool.Exec(context.Background(), "DELETE FROM compliance_controls WHERE id = $1", f.ControlID)                                                                //nolint:errcheck
		_, _ = pool.Exec(context.Background(), "DELETE FROM compliance_frameworks WHERE id = $1", f.FrameworkID)                                                            //nolint:errcheck
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", f.UserID)                                                                                 //nolint:errcheck
		_, _ = pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1", f.OrgID)                                                                          //nolint:errcheck
	})

	// Org + user + framework + control + task + invocation, in FK order.
	if _, err := pool.Exec(ctx,
		`INSERT INTO organizations (id, name, slug) VALUES ($1, $2, $3)`,
		f.OrgID, "PR24 test org", "pr24-"+uuid.NewString()[:8]); err != nil {
		t.Fatalf("seed org: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, org_id, external_id, email, name)
		 VALUES ($1, $2, $3, $4, $5)`,
		f.UserID, f.OrgID, "ext-"+f.UserID[:8], f.UserID[:8]+"@example.test", "PR24 Test"); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO compliance_frameworks (id, org_id, name, description, level, enabled)
		 VALUES ($1, $2, $3, $4, $5, true)`,
		f.FrameworkID, f.OrgID, "PR24-CIS", "PR24 test framework", 1); err != nil {
		t.Fatalf("seed framework: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO compliance_controls (id, framework_id, control_id, title, severity)
		 VALUES ($1, $2, $3, $4, $5)`,
		f.ControlID, f.FrameworkID, "PR24-PATCH", "Test patch control", "high"); err != nil {
		t.Fatalf("seed control: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO ai_tasks (id, org_id, created_by, user_intent, state)
		 VALUES ($1, $2, $3, $4, 'planned')`,
		f.TaskID, f.OrgID, f.UserID, "PR24 test task"); err != nil {
		t.Fatalf("seed task: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO ai_tool_invocations (id, task_id, tool_name, risk_level, parameters, result, created_at)
		 VALUES ($1, $2, $3, $4, '{}', '{"command_id":"cmd-test-001"}', NOW())`,
		f.InvocationID, f.TaskID, "test_pr24_tool", "state_change_prod"); err != nil {
		t.Fatalf("seed invocation: %v", err)
	}
	return f
}

func quietLog() *logger.Logger {
	return logger.New("error", "text")
}

// TestLookupControlForTool_OrgSpecificBeatsGlobal — an org-specific mapping
// with the same pattern as a global mapping wins.
func TestLookupControlForTool_OrgSpecificBeatsGlobal(t *testing.T) {
	pool := testDB(t)
	f := seedEmitterFixture(t, pool)

	// Org-specific control id distinct from f.ControlID so we can verify
	// which one came back.
	orgSpecificControlID := uuid.NewString()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM compliance_controls WHERE id = $1", orgSpecificControlID) //nolint:errcheck
	})
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO compliance_controls (id, framework_id, control_id, title, severity)
		 VALUES ($1, $2, $3, $4, $5)`,
		orgSpecificControlID, f.FrameworkID, "PR24-PATCH-ORG", "Org-specific patch control", "high"); err != nil {
		t.Fatalf("seed org control: %v", err)
	}

	// Insert both mappings for the same tool name.
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO tool_compliance_mappings (org_id, tool_name_pattern, control_id) VALUES (NULL, $1, $2), ($3, $1, $4)`,
		"test_pr24_tool", f.ControlID, f.OrgID, orgSpecificControlID); err != nil {
		t.Fatalf("seed mappings: %v", err)
	}

	got, err := lookupControlForTool(context.Background(), pool, f.OrgID, "test_pr24_tool")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got != orgSpecificControlID {
		t.Errorf("got control_id %q, want org-specific %q (got global %q?)", got, orgSpecificControlID, f.ControlID)
	}
}

// TestLookupControlForTool_ExactBeatsWildcard — an exact-name mapping
// wins over a wildcard mapping at the same scope.
func TestLookupControlForTool_ExactBeatsWildcard(t *testing.T) {
	pool := testDB(t)
	f := seedEmitterFixture(t, pool)

	exactControlID := uuid.NewString()
	wildcardControlID := uuid.NewString()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM compliance_controls WHERE id IN ($1, $2)", exactControlID, wildcardControlID) //nolint:errcheck
	})
	for _, id := range []string{exactControlID, wildcardControlID} {
		if _, err := pool.Exec(context.Background(),
			`INSERT INTO compliance_controls (id, framework_id, control_id, title, severity) VALUES ($1, $2, $3, 'test', 'high')`,
			id, f.FrameworkID, "test_"+id[:8]); err != nil {
			t.Fatalf("seed control %s: %v", id, err)
		}
	}

	// Wildcard global + exact global. Exact must win.
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO tool_compliance_mappings (org_id, tool_name_pattern, control_id)
		 VALUES (NULL, 'test_pr24_*', $1), (NULL, 'test_pr24_tool', $2)`,
		wildcardControlID, exactControlID); err != nil {
		t.Fatalf("seed mappings: %v", err)
	}

	got, err := lookupControlForTool(context.Background(), pool, f.OrgID, "test_pr24_tool")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got != exactControlID {
		t.Errorf("got %q, want exact %q (got wildcard %q?)", got, exactControlID, wildcardControlID)
	}
}

// TestLookupControlForTool_NoMatchReturnsEmpty — an unmapped tool name
// returns "" and a nil error. The emitter relies on this to silently skip.
func TestLookupControlForTool_NoMatchReturnsEmpty(t *testing.T) {
	pool := testDB(t)
	f := seedEmitterFixture(t, pool)

	got, err := lookupControlForTool(context.Background(), pool, f.OrgID, "definitely_not_mapped_anywhere")
	if err != nil {
		t.Fatalf("expected nil error for unmapped tool, got %v", err)
	}
	if got != "" {
		t.Errorf("expected empty control id, got %q", got)
	}
}

// TestEmitToolEvidence_SkipsSimulated — params containing _simulated:true
// skip the entire emission path; no row gets inserted.
func TestEmitToolEvidence_SkipsSimulated(t *testing.T) {
	pool := testDB(t)
	f := seedEmitterFixture(t, pool)

	// Map the tool so a non-simulated row would normally produce evidence.
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO tool_compliance_mappings (org_id, tool_name_pattern, control_id) VALUES (NULL, $1, $2)`,
		"test_pr24_tool", f.ControlID); err != nil {
		t.Fatalf("seed mapping: %v", err)
	}

	id, err := EmitToolEvidence(context.Background(), pool, quietLog(), EmitOpts{
		InvocationID: f.InvocationID,
		ToolName:     "test_pr24_tool",
		RiskLevel:    "state_change_prod",
		OrgID:        f.OrgID,
		ParamsJSON:   []byte(`{"_simulated":true,"foo":"bar"}`),
		ResultJSON:   []byte(`{"result":"ok"}`),
	})
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	if id != "" {
		t.Errorf("evidence id = %q, want empty (simulated rows must skip)", id)
	}

	// Confirm no evidence row landed.
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM compliance_evidence WHERE org_id = $1 AND ai_tool_invocation_id = $2`,
		f.OrgID, f.InvocationID,
	).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("evidence rows for simulated invocation = %d, want 0", n)
	}
}

// TestEmitToolEvidence_ComputesContentHash — the inserted content_hash is
// SHA-256 of the result JSON, hex-encoded.
func TestEmitToolEvidence_ComputesContentHash(t *testing.T) {
	pool := testDB(t)
	f := seedEmitterFixture(t, pool)

	if _, err := pool.Exec(context.Background(),
		`INSERT INTO tool_compliance_mappings (org_id, tool_name_pattern, control_id) VALUES (NULL, $1, $2)`,
		"test_pr24_tool", f.ControlID); err != nil {
		t.Fatalf("seed mapping: %v", err)
	}

	resultJSON := []byte(`{"command_id":"cmd-test-001","dry_run":false}`)
	id, err := EmitToolEvidence(context.Background(), pool, quietLog(), EmitOpts{
		InvocationID: f.InvocationID,
		ToolName:     "test_pr24_tool",
		RiskLevel:    "state_change_prod",
		OrgID:        f.OrgID,
		ParamsJSON:   []byte(`{}`),
		ResultJSON:   resultJSON,
	})
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	if id == "" {
		t.Fatal("expected evidence id, got empty")
	}

	expectedHashBytes := sha256.Sum256(resultJSON)
	expected := hex.EncodeToString(expectedHashBytes[:])
	var got string
	if err := pool.QueryRow(context.Background(),
		`SELECT content_hash FROM compliance_evidence WHERE id = $1`, id,
	).Scan(&got); err != nil {
		t.Fatalf("query content_hash: %v", err)
	}
	if got != expected {
		t.Errorf("content_hash = %q, want %q (SHA-256 of result JSON)", got, expected)
	}
}

// TestEmitToolEvidence_LinksToInvocation — the new ai_tool_invocation_id
// column points at the source audit row.
func TestEmitToolEvidence_LinksToInvocation(t *testing.T) {
	pool := testDB(t)
	f := seedEmitterFixture(t, pool)

	if _, err := pool.Exec(context.Background(),
		`INSERT INTO tool_compliance_mappings (org_id, tool_name_pattern, control_id) VALUES (NULL, $1, $2)`,
		"test_pr24_tool", f.ControlID); err != nil {
		t.Fatalf("seed mapping: %v", err)
	}

	id, err := EmitToolEvidence(context.Background(), pool, quietLog(), EmitOpts{
		InvocationID: f.InvocationID,
		ToolName:     "test_pr24_tool",
		RiskLevel:    "state_change_prod",
		OrgID:        f.OrgID,
		ParamsJSON:   []byte(`{}`),
		ResultJSON:   []byte(`{"ok":true}`),
	})
	if err != nil {
		t.Fatalf("emit: %v", err)
	}

	var (
		linkedInvID *string
		storagePath string
	)
	if err := pool.QueryRow(context.Background(),
		`SELECT ai_tool_invocation_id::text, storage_path FROM compliance_evidence WHERE id = $1`, id,
	).Scan(&linkedInvID, &storagePath); err != nil {
		t.Fatalf("query: %v", err)
	}
	if linkedInvID == nil || *linkedInvID != f.InvocationID {
		t.Errorf("ai_tool_invocation_id = %v, want %s", linkedInvID, f.InvocationID)
	}
	if storagePath != "ai_tool_invocations:"+f.InvocationID {
		t.Errorf("storage_path = %q, want ai_tool_invocations:%s", storagePath, f.InvocationID)
	}
}
