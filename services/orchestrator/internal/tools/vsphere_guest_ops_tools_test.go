// PR #34 / CONN-013 — unit tests for VSphereGuestProgramDryRunTool.

package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func newVSphereGuestOpsRealForTest(t *testing.T) VSphereGuestOpsClient {
	t.Helper()
	return NewRealVSphereGuestOpsClient(testLoggerForSSM())
}

func TestVSphereGuestOps_BuildsPlan(t *testing.T) {
	tool := NewVSphereGuestProgramDryRunTool(newVSphereGuestOpsRealForTest(t))

	got, err := tool.Execute(context.Background(), map[string]any{
		"vm_name":        "prod-app-01",
		"guest_user":     "ql-rf",
		"guest_password": "test-password",
		"program_path":   "/bin/bash",
		"arguments":      "-c 'apt-get update && apt-get upgrade -y'",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	envelope, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("got = %T, want map[string]any", got)
	}
	if envelope["dry_run"] != true {
		t.Errorf("dry_run = %v, want true", envelope["dry_run"])
	}
	plan, ok := envelope["program_plan"].(*VSphereGuestProgramPlan)
	if !ok {
		t.Fatalf("program_plan = %T, want *VSphereGuestProgramPlan", envelope["program_plan"])
	}
	if plan.ProgramPath != "/bin/bash" {
		t.Errorf("ProgramPath = %q, want /bin/bash", plan.ProgramPath)
	}
}

func TestVSphereGuestOps_RejectsEmptyPassword(t *testing.T) {
	tool := NewVSphereGuestProgramDryRunTool(newVSphereGuestOpsRealForTest(t))

	_, err := tool.Execute(context.Background(), map[string]any{
		"vm_name":      "prod-app-01",
		"guest_user":   "ql-rf",
		"program_path": "/bin/bash",
	})
	if err == nil {
		t.Fatal("expected error for missing guest_password, got nil")
	}
	if !strings.Contains(err.Error(), "guest_password is required") {
		t.Errorf("error = %q, want substring 'guest_password is required'", err.Error())
	}
}

func TestVSphereGuestOps_RejectsRelativeProgramPath(t *testing.T) {
	tool := NewVSphereGuestProgramDryRunTool(newVSphereGuestOpsRealForTest(t))

	_, err := tool.Execute(context.Background(), map[string]any{
		"vm_name":        "prod-app-01",
		"guest_user":     "ql-rf",
		"guest_password": "test",
		"program_path":   "bash", // relative — invalid
	})
	if err == nil {
		t.Fatal("expected error for relative program_path, got nil")
	}
	if !strings.Contains(err.Error(), "absolute path") {
		t.Errorf("error = %q, want substring 'absolute path'", err.Error())
	}
}

func TestVSphereGuestOps_AcceptsWindowsProgramPath(t *testing.T) {
	tool := NewVSphereGuestProgramDryRunTool(newVSphereGuestOpsRealForTest(t))

	_, err := tool.Execute(context.Background(), map[string]any{
		"vm_name":        "prod-app-01",
		"guest_user":     "ql-rf",
		"guest_password": "test",
		"program_path":   "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe",
	})
	if err != nil {
		t.Fatalf("Windows program_path should be accepted: %v", err)
	}
}

func TestVSphereGuestProgramPlan_OmitsPasswordFromJSON(t *testing.T) {
	// Defensive: the GuestPassword field has json:"-" — auditors must
	// never see the credential in the audit log's `result` JSONB.
	plan := &VSphereGuestProgramPlan{
		VMName:        "test",
		GuestUser:     "u",
		GuestPassword: "should-not-appear",
		ProgramPath:   "/bin/bash",
	}
	// json encode
	encoded := mustJSONEncode(plan)
	if strings.Contains(encoded, "should-not-appear") {
		t.Errorf("password leaked into JSON: %s", encoded)
	}
}

func TestMockVSphereGuestOpsClient(t *testing.T) {
	c := NewMockVSphereGuestOpsClient()
	plan, err := c.BuildGuestProgramPlan(context.Background(), VSphereGuestProgramRequest{
		VMName: "garbage",
	})
	if err != nil {
		t.Fatalf("BuildGuestProgramPlan: %v", err)
	}
	if plan.VMName != "mock-esx-vm-prod-01" {
		t.Errorf("mock returned non-mock VMName %q", plan.VMName)
	}
	if !plan.DryRun {
		t.Errorf("DryRun = false; mock client must always produce dry-run plans")
	}
}

func TestRegisterVSphereGuestOpsDryRunTools_NoOpOnNil(t *testing.T) {
	r := NewRegistry(nil, testLoggerForSSM())
	r.RegisterVSphereGuestOpsDryRunTools(nil)
	if _, ok := r.Get("vsphere_run_guest_program"); ok {
		t.Error("should not register with nil client")
	}
}

// mustJSONEncode is a tiny test helper.
func mustJSONEncode(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
