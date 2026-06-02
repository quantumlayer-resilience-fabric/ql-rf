// PR #27 / CONN-007 — unit tests for AzureRunCommandDryRunTool +
// validation helpers.
//
// Five behavioral tests + the two structural safety tests in
// no_azure_runcommand_sdk_import_test.go = the package-level coverage
// for PR #27.

package tools

import (
	"context"
	"strings"
	"testing"
)

func newAzureRunCommandRealForTest(t *testing.T) AzureRunCommandClient {
	t.Helper()
	return NewRealAzureRunCommandClient(testLoggerForSSM())
}

// TestAzureRunCommand_BuildsPlan — happy path with a valid VM + script.
// The returned envelope must mark dry_run:true and real_changes:false;
// the embedded command_plan must carry the requested fields.
func TestAzureRunCommand_BuildsPlan(t *testing.T) {
	tool := NewAzureRunCommandDryRunTool(newAzureRunCommandRealForTest(t))

	got, err := tool.Execute(context.Background(), map[string]any{
		"resource_group": "rg-prod-eus",
		"vm_name":        "vm-app-01",
		"command_id":     "RunPowerShellScript",
		"script":         "Update-Module Az -Force",
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
	plan, ok := m["command_plan"].(*AzureRunCommandPlan)
	if !ok {
		t.Fatalf("command_plan = %T, want *AzureRunCommandPlan", m["command_plan"])
	}
	if plan.CommandID != "RunPowerShellScript" {
		t.Errorf("CommandID = %q, want RunPowerShellScript", plan.CommandID)
	}
	if plan.VMName != "vm-app-01" {
		t.Errorf("VMName = %q, want vm-app-01", plan.VMName)
	}
}

// TestAzureRunCommand_DefaultsToRunShellScript — empty command_id
// defaults to RunShellScript (the safer Linux baseline).
func TestAzureRunCommand_DefaultsToRunShellScript(t *testing.T) {
	tool := NewAzureRunCommandDryRunTool(newAzureRunCommandRealForTest(t))

	got, err := tool.Execute(context.Background(), map[string]any{
		"resource_group": "rg-prod-eus",
		"vm_name":        "vm-app-01",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	envelope, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("got = %T, want map[string]any", got)
	}
	plan, ok := envelope["command_plan"].(*AzureRunCommandPlan)
	if !ok {
		t.Fatalf("command_plan is %T, want *AzureRunCommandPlan", envelope["command_plan"])
	}
	if plan.CommandID != "RunShellScript" {
		t.Errorf("default CommandID = %q, want RunShellScript", plan.CommandID)
	}
}

// TestAzureRunCommand_RejectsInvalidVMName — Azure VM naming rules are
// strict; a name with an invalid char gets rejected before the plan is
// constructed.
func TestAzureRunCommand_RejectsInvalidVMName(t *testing.T) {
	tool := NewAzureRunCommandDryRunTool(newAzureRunCommandRealForTest(t))

	_, err := tool.Execute(context.Background(), map[string]any{
		"resource_group": "rg-prod-eus",
		"vm_name":        "bad!name", // ! is not allowed
	})
	if err == nil {
		t.Fatal("Execute with bad vm_name: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid vm_name") {
		t.Errorf("error = %q, want substring 'invalid vm_name'", err.Error())
	}
}

// TestAzureRunCommand_RejectsBadCommandID — command_id must be one of
// the two supported documents; anything else (e.g., "RunCustomScript")
// is rejected at the boundary.
func TestAzureRunCommand_RejectsBadCommandID(t *testing.T) {
	tool := NewAzureRunCommandDryRunTool(newAzureRunCommandRealForTest(t))

	_, err := tool.Execute(context.Background(), map[string]any{
		"resource_group": "rg-prod-eus",
		"vm_name":        "vm-app-01",
		"command_id":     "RunCustomScript",
	})
	if err == nil {
		t.Fatal("Execute with bad command_id: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "command_id must be") {
		t.Errorf("error = %q, want substring 'command_id must be'", err.Error())
	}
}

// TestAzureRunCommand_RejectsEmptyResourceGroup — empty resource_group
// returns an explicit error rather than building a plan with no target
// context.
func TestAzureRunCommand_RejectsEmptyResourceGroup(t *testing.T) {
	tool := NewAzureRunCommandDryRunTool(newAzureRunCommandRealForTest(t))

	_, err := tool.Execute(context.Background(), map[string]any{
		"resource_group": "",
		"vm_name":        "vm-app-01",
	})
	if err == nil {
		t.Fatal("Execute with empty resource_group: expected error, got nil")
	}
}

// TestMockAzureRunCommandClient — confirms the mock client's documented
// behavior: ignores caller-provided fields and always returns the same
// `mock-vm-prod-01` fixture. Makes CI tests deterministic regardless of
// input shape.
func TestMockAzureRunCommandClient(t *testing.T) {
	c := NewMockAzureRunCommandClient()
	plan, err := c.BuildRunCommandPlan(context.Background(), AzureRunCommandRequest{
		ResourceGroup: "rg-garbage",
		VMName:        "garbage-vm",
		CommandID:     "RunPowerShellScript",
	})
	if err != nil {
		t.Fatalf("BuildRunCommandPlan: %v", err)
	}
	if !strings.HasPrefix(plan.VMName, "mock-vm-") {
		t.Errorf("mock returned non-mock VM name %q; viewers won't recognize the mock origin", plan.VMName)
	}
	if !plan.DryRun {
		t.Errorf("DryRun = false; mock client must always produce dry-run plans")
	}
	if plan.RealChanges {
		t.Errorf("RealChanges = true; mock client must never claim real changes")
	}
}

// TestRegisterAzureRunCommandDryRunTools_NoOpOnNil — the registry helper
// refuses to register the tool when the client is nil.
func TestRegisterAzureRunCommandDryRunTools_NoOpOnNil(t *testing.T) {
	r := NewRegistry(nil, testLoggerForSSM())
	r.RegisterAzureRunCommandDryRunTools(nil)
	if _, ok := r.Get("azure_run_command"); ok {
		t.Error("should not register with nil client")
	}
}
