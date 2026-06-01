// PR #20 / CONN-002 — unit tests for SSMSendPatchCommandTool.
//
// Four behavioral tests + the two structural safety tests in
// no_ssm_sdk_import_test.go = the package-level test coverage for PR #20.

package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// realClient returns a real SSMClient (which validates input strictly).
// Mock testing is covered in TestMockSSMClient_IgnoresInputAndReturnsFixture.
func newRealForTest(t *testing.T) SSMClient {
	t.Helper()
	// We don't need a logger since the client's only call site
	// is `log.WithComponent` which slog handles cleanly with a default
	// logger. Construct via the public constructor to ensure boot-time
	// initialization is exercised.
	return NewRealSSMClient(testLoggerForSSM())
}

// TestSSMTool_BuildsPatchCommandPlan — happy path with two real-looking
// instance IDs. The returned envelope must mark dry_run:true and
// real_changes:false, and the embedded command_plan must use the
// AWS-RunPatchBaseline document.
func TestSSMTool_BuildsPatchCommandPlan(t *testing.T) {
	tool := NewSSMSendPatchCommandTool(newRealForTest(t))

	got, err := tool.Execute(context.Background(), map[string]any{
		"region":       "eu-west-1",
		"instance_ids": []any{"i-0a1b2c3d4e5f6a7b8", "i-1234567890abcdef0"},
		"operation":    "Install",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("Execute returned %T, want map[string]any", got)
	}
	if m["dry_run"] != true {
		t.Errorf("dry_run = %v, want true", m["dry_run"])
	}
	if m["real_changes"] != false {
		t.Errorf("real_changes = %v, want false", m["real_changes"])
	}
	plan, ok := m["command_plan"].(*SSMCommandPlan)
	if !ok {
		t.Fatalf("command_plan = %T, want *SSMCommandPlan", m["command_plan"])
	}
	if plan.DocumentName != "AWS-RunPatchBaseline" {
		t.Errorf("DocumentName = %q, want AWS-RunPatchBaseline", plan.DocumentName)
	}
	if plan.Region != "eu-west-1" {
		t.Errorf("Region = %q, want eu-west-1", plan.Region)
	}
	if len(plan.InstanceIDs) != 2 {
		t.Errorf("InstanceIDs len = %d, want 2", len(plan.InstanceIDs))
	}
	if plan.Parameters["Operation"][0] != "Install" {
		t.Errorf("Operation = %v, want Install", plan.Parameters["Operation"])
	}
}

// TestSSMTool_DefaultOperationIsScan — empty operation param defaults to
// "Scan" (the safer choice; Install actually patches when live mode lands
// in PR #21).
func TestSSMTool_DefaultOperationIsScan(t *testing.T) {
	tool := NewSSMSendPatchCommandTool(newRealForTest(t))

	got, err := tool.Execute(context.Background(), map[string]any{
		"instance_ids": []any{"i-0a1b2c3d4e5f6a7b8"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	plan := got.(map[string]any)["command_plan"].(*SSMCommandPlan)
	if plan.Parameters["Operation"][0] != "Scan" {
		t.Errorf("default Operation = %v, want Scan", plan.Parameters["Operation"])
	}
}

// TestSSMTool_RejectsInvalidInstanceID — a non-i- instance ID is rejected
// before the plan is constructed. Defensive: garbage in the audit log is
// worse than no entry at all.
func TestSSMTool_RejectsInvalidInstanceID(t *testing.T) {
	tool := NewSSMSendPatchCommandTool(newRealForTest(t))

	_, err := tool.Execute(context.Background(), map[string]any{
		"instance_ids": []any{"not-an-instance-id"},
	})
	if err == nil {
		t.Fatal("Execute with bad instance id: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid instance_id") {
		t.Errorf("error = %q, want substring 'invalid instance_id'", err.Error())
	}
}

// TestSSMTool_RejectsBadOperation — operation must be "Scan" or "Install";
// "Reboot" (a valid SSM document parameter, but not for AWS-RunPatchBaseline)
// is rejected.
func TestSSMTool_RejectsBadOperation(t *testing.T) {
	tool := NewSSMSendPatchCommandTool(newRealForTest(t))

	_, err := tool.Execute(context.Background(), map[string]any{
		"instance_ids": []any{"i-0a1b2c3d4e5f6a7b8"},
		"operation":    "Reboot",
	})
	if err == nil {
		t.Fatal("Execute with bad operation: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "operation must be") {
		t.Errorf("error = %q, want substring 'operation must be'", err.Error())
	}
}

// TestSSMTool_RejectsEmptyInstanceList — empty list returns an explicit
// error (rather than silently building a plan with no targets).
func TestSSMTool_RejectsEmptyInstanceList(t *testing.T) {
	tool := NewSSMSendPatchCommandTool(newRealForTest(t))

	_, err := tool.Execute(context.Background(), map[string]any{
		"instance_ids": []any{},
	})
	if err == nil {
		t.Fatal("Execute with empty instance list: expected error, got nil")
	}
}

// TestMockSSMClient_IgnoresInputAndReturnsFixture — confirms the mock
// client's documented behavior: it ignores caller-provided instance IDs
// and always returns the same two i-mock-* IDs. This makes CI tests
// deterministic regardless of input shape.
func TestMockSSMClient_IgnoresInputAndReturnsFixture(t *testing.T) {
	c := NewMockSSMClient()
	plan, err := c.BuildPatchCommand(context.Background(), PatchCommandRequest{
		Region:      "us-west-2",
		InstanceIDs: []string{"this-is-garbage-and-ignored"},
		Operation:   "Install",
	})
	if err != nil {
		t.Fatalf("BuildPatchCommand: %v", err)
	}
	if len(plan.InstanceIDs) != 2 {
		t.Fatalf("InstanceIDs len = %d, want 2 (mock fixture)", len(plan.InstanceIDs))
	}
	for _, id := range plan.InstanceIDs {
		if !strings.HasPrefix(id, "i-mock-") {
			t.Errorf("mock returned non-mock id %q; viewers won't recognize the mock origin", id)
		}
	}
	if !plan.DryRun {
		t.Errorf("DryRun = false; mock client must always produce dry-run plans")
	}
}

// testLoggerForSSM returns a quiet logger for unit tests.
func testLoggerForSSM() *logger.Logger {
	return logger.New("error", "text")
}
