// PR #28 / CONN-008 — unit tests for AzureRunCommandLiveTool + the live
// client + whitelist helpers.
//
// Pure-Go tests against the mock client. The real client is exercised
// only at boot in production; its SDK calls are out of scope for unit
// tests. CI runs with RF_CONNECTORS_AZURE_ALLOW_LIVE_RUN_COMMAND unset
// (default), so the live tool is never even registered there.

package tools

import (
	"context"
	"strings"
	"testing"

	pkgconfig "github.com/quantumlayerhq/ql-rf/pkg/config"
)

// TestParseAzureLiveWhitelistCSV — empty, single, multiple, whitespace.
func TestParseAzureLiveWhitelistCSV(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{",,,", nil},
		{"rg-a/vm-1", []string{"rg-a/vm-1"}},
		{" rg-a/vm-1 , rg-b/vm-2 ", []string{"rg-a/vm-1", "rg-b/vm-2"}},
		{"rg-a/vm-1,,rg-b/vm-2", []string{"rg-a/vm-1", "rg-b/vm-2"}},
	}
	for _, c := range cases {
		got := parseAzureLiveWhitelistCSV(c.in)
		if !stringSliceEq(got, c.want) {
			t.Errorf("parseAzureLiveWhitelistCSV(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestMockLiveAzureRunCommandClient_ValidatesAndReturnsOpToken — the
// mock client's contract: validate whitelist, return deterministic-shape
// op-azmock-<hex> without touching the SDK.
func TestMockLiveAzureRunCommandClient_ValidatesAndReturnsOpToken(t *testing.T) {
	wl := []string{"rg-a/vm-1"}
	c := NewMockLiveAzureRunCommandClient(wl)
	plan := &AzureRunCommandPlan{
		ResourceGroup: "rg-a",
		VMName:        "vm-1",
		CommandID:     "RunShellScript",
		Script:        "echo hello",
	}
	tok, err := c.SendRunCommand(context.Background(), plan)
	if err != nil {
		t.Fatalf("SendRunCommand: %v", err)
	}
	if !strings.HasPrefix(tok, "op-azmock-") {
		t.Errorf("op token = %q, want op-azmock-* prefix", tok)
	}
}

// TestMockLiveAzureRunCommandClient_RejectsNonWhitelistedVM — the mock
// enforces the whitelist just like the real client. Tests that exercise
// the live tool's Execute against the mock therefore catch whitelist
// regressions without needing a real client.
func TestMockLiveAzureRunCommandClient_RejectsNonWhitelistedVM(t *testing.T) {
	c := NewMockLiveAzureRunCommandClient([]string{"rg-a/vm-1"})
	plan := &AzureRunCommandPlan{
		ResourceGroup: "rg-b",
		VMName:        "vm-2",
	}
	if _, err := c.SendRunCommand(context.Background(), plan); err == nil {
		t.Fatal("expected whitelist rejection, got nil")
	}
}

// TestMockLiveAzureRunCommandClient_RejectsNilPlan — defensive against
// programmer error in tool wiring.
func TestMockLiveAzureRunCommandClient_RejectsNilPlan(t *testing.T) {
	c := NewMockLiveAzureRunCommandClient([]string{"rg-a/vm-1"})
	if _, err := c.SendRunCommand(context.Background(), nil); err == nil {
		t.Fatal("expected nil-plan rejection, got nil")
	}
}

// TestNewLiveAzureRunCommandClient_RefusesEmptyWhitelist — the boot path
// passes the whitelist after validating it's non-empty, but the
// constructor adds a belt-and-braces refusal so a programmer error
// upstream still fails loudly rather than constructing an unusable
// client.
func TestNewLiveAzureRunCommandClient_RefusesEmptyWhitelist(t *testing.T) {
	_, err := NewLiveAzureRunCommandClient(context.Background(), pkgconfig.AzureConfig{}, nil, testLoggerForSSM())
	if err == nil {
		t.Fatal("expected refusal for empty whitelist, got nil")
	}
	if !strings.Contains(err.Error(), "whitelist") {
		t.Errorf("error should mention whitelist; got %q", err.Error())
	}
}

// TestNewLiveAzureRunCommandClient_RefusesMissingCreds — credentials
// missing → refuse to construct. The empty-whitelist check runs first
// (cheapest), but if a caller passes a non-empty whitelist with no
// credentials, the constructor still errors out.
func TestNewLiveAzureRunCommandClient_RefusesMissingCreds(t *testing.T) {
	_, err := NewLiveAzureRunCommandClient(context.Background(), pkgconfig.AzureConfig{}, []string{"rg-a/vm-1"}, testLoggerForSSM())
	if err == nil {
		t.Fatal("expected refusal for missing credentials, got nil")
	}
	if !strings.Contains(err.Error(), "credentials not configured") {
		t.Errorf("error should mention credentials; got %q", err.Error())
	}
}

// TestAzureRunCommandLiveTool_FiresMockAndReturnsLiveEnvelope — happy
// path. The envelope MUST have dry_run:false and real_changes:true;
// the operation_token MUST be present.
func TestAzureRunCommandLiveTool_FiresMockAndReturnsLiveEnvelope(t *testing.T) {
	whitelist := []string{"rg-prod-eus/vm-app-01"}
	tool := NewAzureRunCommandLiveTool(
		NewRealAzureRunCommandClient(testLoggerForSSM()),
		NewMockLiveAzureRunCommandClient(whitelist),
	)

	got, err := tool.Execute(context.Background(), map[string]any{
		"resource_group": "rg-prod-eus",
		"vm_name":        "vm-app-01",
		"command_id":     "RunPowerShellScript",
		"script":         "Update-Module Az -Force",
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
	tok, ok := envelope["operation_token"].(string)
	if !ok || !strings.HasPrefix(tok, "op-azmock-") {
		t.Errorf("operation_token = %v, want op-azmock-* string", envelope["operation_token"])
	}
}

// TestAzureRunCommandLiveTool_RejectsNonWhitelistedVM — defense in
// depth: even if the OPA policy + handler somehow let a non-whitelisted
// VM through, the tool's call to liveClient.SendRunCommand fails.
func TestAzureRunCommandLiveTool_RejectsNonWhitelistedVM(t *testing.T) {
	tool := NewAzureRunCommandLiveTool(
		NewRealAzureRunCommandClient(testLoggerForSSM()),
		NewMockLiveAzureRunCommandClient([]string{"rg-prod-eus/vm-app-01"}),
	)

	_, err := tool.Execute(context.Background(), map[string]any{
		"resource_group": "rg-prod-eus",
		"vm_name":        "vm-app-02", // valid format but not on whitelist
	})
	if err == nil {
		t.Fatal("expected whitelist rejection, got nil")
	}
}

// TestAzureRunCommandLiveTool_RisksAndApprovalShape — locked to the
// state_change_prod two-approver contract. A future refactor that
// loosens these would break the OPA policy's enforcement assumption.
func TestAzureRunCommandLiveTool_RisksAndApprovalShape(t *testing.T) {
	tool := NewAzureRunCommandLiveTool(
		NewRealAzureRunCommandClient(testLoggerForSSM()),
		NewMockLiveAzureRunCommandClient([]string{"rg/vm"}),
	)
	if tool.Risk() != RiskStateChangeProd {
		t.Errorf("Risk = %v, want %v", tool.Risk(), RiskStateChangeProd)
	}
	if !tool.RequiresApproval() {
		t.Error("RequiresApproval should be true")
	}
	if tool.Idempotent() {
		t.Error("Idempotent should be false — a live Run Command creates a new operation every call")
	}
	if tool.Name() != "azure_run_command_live" {
		t.Errorf("Name = %q, want azure_run_command_live", tool.Name())
	}
}

// TestRegisterAzureLiveRunCommandTools_NoOpOnNilClient — registry
// helper refuses to register the tool when either client is nil.
func TestRegisterAzureLiveRunCommandTools_NoOpOnNilClient(t *testing.T) {
	r := NewRegistry(nil, testLoggerForSSM())
	r.RegisterAzureLiveRunCommandTools(nil, NewMockLiveAzureRunCommandClient([]string{"rg/vm"}))
	if _, ok := r.Get("azure_run_command_live"); ok {
		t.Error("should not register with nil dry client")
	}
	r.RegisterAzureLiveRunCommandTools(NewRealAzureRunCommandClient(testLoggerForSSM()), nil)
	if _, ok := r.Get("azure_run_command_live"); ok {
		t.Error("should not register with nil live client")
	}
}
