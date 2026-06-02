// PR #21 / CONN-003 — unit tests for SSMSendPatchCommandLiveTool.
//
// Exercises the tool's Execute method against the mock live client so the
// full live-mode envelope shape (dry_run:false, real_changes:true,
// command_id present) is verified without firing real AWS calls.

package tools

import (
	"context"
	"strings"
	"testing"
)

// newLiveToolForTest builds the live tool with the dry-run client (for
// plan construction) and the mock live client (for the send). The
// whitelist matches the instance IDs the tests use.
func newLiveToolForTest(t *testing.T, whitelist []string) *SSMSendPatchCommandLiveTool {
	t.Helper()
	dry := NewRealSSMClient(testLoggerForSSM())
	live := NewMockLiveSSMClient(whitelist)
	return NewSSMSendPatchCommandLiveTool(dry, live)
}

// TestSSMLiveTool_FiresMockAndReturnsLiveEnvelope — happy path. The
// envelope MUST have dry_run:false and real_changes:true; the command_id
// MUST be present.
func TestSSMLiveTool_FiresMockAndReturnsLiveEnvelope(t *testing.T) {
	whitelist := []string{"i-0a1b2c3d4e5f6a7b8"}
	tool := newLiveToolForTest(t, whitelist)

	got, err := tool.Execute(context.Background(), map[string]any{
		"region":       "us-east-1",
		"instance_ids": []any{"i-0a1b2c3d4e5f6a7b8"},
		"operation":    "Install",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("Execute returned %T, want map[string]any", got)
	}
	if m["dry_run"] != false {
		t.Errorf("dry_run = %v, want false", m["dry_run"])
	}
	if m["real_changes"] != true {
		t.Errorf("real_changes = %v, want true", m["real_changes"])
	}
	if m["did_invoke"] != true {
		t.Errorf("did_invoke = %v, want true", m["did_invoke"])
	}
	cmdID, ok := m["command_id"].(string)
	if !ok || !strings.HasPrefix(cmdID, "cmd-mock-") {
		t.Errorf("command_id = %v, want cmd-mock-* string", m["command_id"])
	}
	if _, ok := m["command_plan"].(*SSMCommandPlan); !ok {
		t.Errorf("command_plan should be *SSMCommandPlan; got %T", m["command_plan"])
	}
}

// TestSSMLiveTool_RejectsNonWhitelistedInstance — defense in depth: even
// if the OPA policy + handler somehow let a non-whitelisted instance
// through, the tool's call to liveClient.SendCommand will fail.
func TestSSMLiveTool_RejectsNonWhitelistedInstance(t *testing.T) {
	tool := newLiveToolForTest(t, []string{"i-0a1b2c3d4e5f6a7b8"})

	_, err := tool.Execute(context.Background(), map[string]any{
		"instance_ids": []any{"i-1234567890abcdef0"}, // valid SSM format, not on whitelist
	})
	if err == nil {
		t.Fatal("expected whitelist rejection, got nil")
	}
}

// TestSSMLiveTool_RejectsInvalidInstanceID — the dry-run builder validates
// the instance ID pattern before the live send is attempted. PR #20's
// regex catches malformed IDs early.
func TestSSMLiveTool_RejectsInvalidInstanceID(t *testing.T) {
	tool := newLiveToolForTest(t, []string{"i-0a1b2c3d4e5f6a7b8"})

	_, err := tool.Execute(context.Background(), map[string]any{
		"instance_ids": []any{"not-an-instance-id"},
	})
	if err == nil {
		t.Fatal("expected invalid-id rejection, got nil")
	}
}

// TestSSMLiveTool_RisksAndApprovalShape — the tool must report
// state_change_prod risk + requires-approval so OPA and the handler
// layer can route it through the two-approver workflow.
func TestSSMLiveTool_RisksAndApprovalShape(t *testing.T) {
	tool := newLiveToolForTest(t, []string{"i-001"})
	if tool.Risk() != RiskStateChangeProd {
		t.Errorf("Risk = %v, want %v", tool.Risk(), RiskStateChangeProd)
	}
	if !tool.RequiresApproval() {
		t.Error("RequiresApproval should be true")
	}
	if tool.Idempotent() {
		t.Error("Idempotent should be false — a live SendCommand creates a new command id every call")
	}
	if tool.Name() != "ssm_send_patch_command_live" {
		t.Errorf("Name = %q, want ssm_send_patch_command_live", tool.Name())
	}
}

// TestRegisterLiveStateChangeTools_NoOpOnNilClient — registry helper
// refuses to register the tool when either client is nil. Tested so a
// future refactor that accidentally drops one client doesn't silently
// produce a tool that panics on first Execute.
func TestRegisterLiveStateChangeTools_NoOpOnNilClient(t *testing.T) {
	r := NewRegistry(nil, testLoggerForSSM())
	r.RegisterLiveStateChangeTools(nil, NewMockLiveSSMClient([]string{"i-001"}))
	if _, ok := r.Get("ssm_send_patch_command_live"); ok {
		t.Error("should not register with nil dry client")
	}
	r.RegisterLiveStateChangeTools(NewRealSSMClient(testLoggerForSSM()), nil)
	if _, ok := r.Get("ssm_send_patch_command_live"); ok {
		t.Error("should not register with nil live client")
	}
}
