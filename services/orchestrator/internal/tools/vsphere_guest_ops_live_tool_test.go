// PR #35 / CONN-014 — unit tests for VSphereGuestProgramLiveTool +
// live client + whitelist helpers.

package tools

import (
	"context"
	"strings"
	"testing"

	pkgconfig "github.com/quantumlayerhq/ql-rf/pkg/config"
)

func TestParseVSphereLiveWhitelistCSV(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{",,,", nil},
		{"prod-app-01", []string{"prod-app-01"}},
		{" prod-app-01 , stage-app-02 ", []string{"prod-app-01", "stage-app-02"}},
	}
	for _, c := range cases {
		got := parseVSphereLiveWhitelistCSV(c.in)
		if !stringSliceEq(got, c.want) {
			t.Errorf("parseVSphereLiveWhitelistCSV(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestMockLiveVSphereGuestOpsClient_ValidatesAndReturnsPID(t *testing.T) {
	c := NewMockLiveVSphereGuestOpsClient([]string{"prod-app-01"})
	plan := &VSphereGuestProgramPlan{
		VMName:      "prod-app-01",
		ProgramPath: "/bin/bash",
	}
	pid, err := c.RunGuestProgram(context.Background(), plan)
	if err != nil {
		t.Fatalf("RunGuestProgram: %v", err)
	}
	if pid < 900000 {
		t.Errorf("mock pid = %d, want >= 900000 (mock-pid range)", pid)
	}
}

func TestMockLiveVSphereGuestOpsClient_RejectsNonWhitelistedVM(t *testing.T) {
	c := NewMockLiveVSphereGuestOpsClient([]string{"prod-app-01"})
	plan := &VSphereGuestProgramPlan{VMName: "other-vm"}
	if _, err := c.RunGuestProgram(context.Background(), plan); err == nil {
		t.Fatal("expected whitelist rejection, got nil")
	}
}

func TestMockLiveVSphereGuestOpsClient_RejectsNilPlan(t *testing.T) {
	c := NewMockLiveVSphereGuestOpsClient([]string{"prod-app-01"})
	if _, err := c.RunGuestProgram(context.Background(), nil); err == nil {
		t.Fatal("expected nil-plan rejection, got nil")
	}
}

func TestNewLiveVSphereGuestOpsClient_RefusesEmptyWhitelist(t *testing.T) {
	_, err := NewLiveVSphereGuestOpsClient(context.Background(), pkgconfig.VSphereConfig{}, nil, testLoggerForSSM())
	if err == nil {
		t.Fatal("expected refusal for empty whitelist, got nil")
	}
	if !strings.Contains(err.Error(), "whitelist") {
		t.Errorf("error should mention whitelist; got %q", err.Error())
	}
}

func TestNewLiveVSphereGuestOpsClient_RefusesMissingCreds(t *testing.T) {
	_, err := NewLiveVSphereGuestOpsClient(context.Background(), pkgconfig.VSphereConfig{}, []string{"prod-app-01"}, testLoggerForSSM())
	if err == nil {
		t.Fatal("expected refusal for missing credentials, got nil")
	}
	if !strings.Contains(err.Error(), "credentials not configured") {
		t.Errorf("error should mention credentials; got %q", err.Error())
	}
}

func TestVSphereGuestProgramLiveTool_FiresMockAndReturnsLiveEnvelope(t *testing.T) {
	whitelist := []string{"prod-app-01"}
	tool := NewVSphereGuestProgramLiveTool(
		NewRealVSphereGuestOpsClient(testLoggerForSSM()),
		NewMockLiveVSphereGuestOpsClient(whitelist),
	)

	got, err := tool.Execute(context.Background(), map[string]any{
		"vm_name":        "prod-app-01",
		"guest_user":     "ql-rf",
		"guest_password": "test",
		"program_path":   "/bin/bash",
		"arguments":      "-c 'echo hello'",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	envelope, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("got = %T, want map[string]any", got)
	}
	if envelope["dry_run"] != false {
		t.Errorf("dry_run = %v, want false", envelope["dry_run"])
	}
	if envelope["real_changes"] != true {
		t.Errorf("real_changes = %v, want true", envelope["real_changes"])
	}
	pid, ok := envelope["guest_pid"].(int64)
	if !ok || pid == 0 {
		t.Errorf("guest_pid = %v, want non-zero int64", envelope["guest_pid"])
	}
}

func TestVSphereGuestProgramLiveTool_RejectsNonWhitelistedVM(t *testing.T) {
	tool := NewVSphereGuestProgramLiveTool(
		NewRealVSphereGuestOpsClient(testLoggerForSSM()),
		NewMockLiveVSphereGuestOpsClient([]string{"prod-app-01"}),
	)

	_, err := tool.Execute(context.Background(), map[string]any{
		"vm_name":        "other-vm",
		"guest_user":     "ql-rf",
		"guest_password": "test",
		"program_path":   "/bin/bash",
	})
	if err == nil {
		t.Fatal("expected whitelist rejection, got nil")
	}
}

func TestVSphereGuestProgramLiveTool_RisksAndApprovalShape(t *testing.T) {
	tool := NewVSphereGuestProgramLiveTool(
		NewRealVSphereGuestOpsClient(testLoggerForSSM()),
		NewMockLiveVSphereGuestOpsClient([]string{"prod-app-01"}),
	)
	if tool.Risk() != RiskStateChangeProd {
		t.Errorf("Risk = %v, want %v", tool.Risk(), RiskStateChangeProd)
	}
	if !tool.RequiresApproval() {
		t.Error("RequiresApproval should be true")
	}
	if tool.Idempotent() {
		t.Error("Idempotent should be false")
	}
	if tool.Name() != "vsphere_run_guest_program_live" {
		t.Errorf("Name = %q, want vsphere_run_guest_program_live", tool.Name())
	}
}

func TestRegisterVSphereLiveGuestOpsTools_NoOpOnNilClient(t *testing.T) {
	r := NewRegistry(nil, testLoggerForSSM())
	r.RegisterVSphereLiveGuestOpsTools(nil, NewMockLiveVSphereGuestOpsClient([]string{"prod-app-01"}))
	if _, ok := r.Get("vsphere_run_guest_program_live"); ok {
		t.Error("should not register with nil dry client")
	}
	r.RegisterVSphereLiveGuestOpsTools(NewRealVSphereGuestOpsClient(testLoggerForSSM()), nil)
	if _, ok := r.Get("vsphere_run_guest_program_live"); ok {
		t.Error("should not register with nil live client")
	}
}
