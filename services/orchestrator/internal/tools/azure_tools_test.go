// PR #26 / CONN-006 — unit tests for QueryAzureVMsTool + helpers.
//
// Pure-Go tests against the mock client. The real client is exercised
// only at boot in production; its SDK calls are out of scope for unit
// tests and CI runs with RF_CONNECTORS_AZURE_FALLBACK_TO_MOCK=true so no
// real Azure subscription is ever needed.

package tools

import (
	"context"
	"strings"
	"testing"
)

// TestQueryAzureVMsTool_Shape — the tool's static metadata (name, risk,
// scope, idempotency, approval) is locked to the read-only contract.
// A future state-change Azure tool MUST use a different name to keep
// the risk-level invariant on the existing tool stable.
func TestQueryAzureVMsTool_Shape(t *testing.T) {
	tool := NewQueryAzureVMsTool(NewMockAzureClient())
	if tool.Name() != "query_azure_vms" {
		t.Errorf("Name = %q, want query_azure_vms", tool.Name())
	}
	if tool.Risk() != RiskReadOnly {
		t.Errorf("Risk = %v, want %v", tool.Risk(), RiskReadOnly)
	}
	if !tool.Idempotent() {
		t.Error("Idempotent should be true for a list call")
	}
	if tool.RequiresApproval() {
		t.Error("RequiresApproval should be false for read-only")
	}
}

// TestQueryAzureVMsTool_ExecuteWithMockClient_ReturnsTwoFixtureVMs —
// confirms the mock client's documented contract: a fixed pair of
// `mock-vm-*` entries flow through the tool's envelope unchanged.
func TestQueryAzureVMsTool_ExecuteWithMockClient_ReturnsTwoFixtureVMs(t *testing.T) {
	tool := NewQueryAzureVMsTool(NewMockAzureClient())

	got, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("Execute returned %T, want map[string]any", got)
	}
	if m["vm_count"] != 2 {
		t.Errorf("vm_count = %v, want 2 (mock fixture)", m["vm_count"])
	}
	vms, ok := m["vms"].([]AzureVM)
	if !ok || len(vms) != 2 {
		t.Fatalf("vms = %T (len %d), want []AzureVM of length 2", m["vms"], len(vms))
	}
	for _, vm := range vms {
		if !strings.HasPrefix(vm.Name, "mock-vm-") {
			t.Errorf("mock returned non-mock VM name %q; viewers won't recognize the mock origin", vm.Name)
		}
		if vm.Tags["mock_origin"] != "ql-rf-test" {
			t.Errorf("mock VM %q missing mock_origin tag", vm.Name)
		}
	}
}

// TestQueryAzureVMsTool_NilClient — defensive: a misconfigured registry
// that somehow registered the tool without a client returns a clean
// error rather than panicking.
func TestQueryAzureVMsTool_NilClient(t *testing.T) {
	tool := NewQueryAzureVMsTool(nil)
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("Execute with nil client: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "client not configured") {
		t.Errorf("error = %q, want substring 'client not configured'", err.Error())
	}
}

// TestParseResourceGroupFromID — extracts the RG name from a canonical
// Azure resource ID. Verifies the case-fold fallback path and the
// no-marker (malformed ID) case.
func TestParseResourceGroupFromID(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/subscriptions/abc/resourceGroups/rg-prod/providers/Microsoft.Compute/virtualMachines/vm1", "rg-prod"},
		{"/SUBSCRIPTIONS/abc/RESOURCEGROUPS/rg-fold-case/providers/...", "rg-fold-case"},
		{"not-an-azure-id", ""},
		{"", ""},
	}
	for _, c := range cases {
		got := parseResourceGroupFromID(c.in)
		if got != c.want {
			t.Errorf("parseResourceGroupFromID(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestRegisterAzureCloudTools_NoOpOnNilClient — the registry helper
// skips registration when called with nil rather than registering a
// tool that panics on first Execute.
func TestRegisterAzureCloudTools_NoOpOnNilClient(t *testing.T) {
	r := NewRegistry(nil, testLoggerForSSM())
	r.RegisterAzureCloudTools(nil)
	if _, ok := r.Get("query_azure_vms"); ok {
		t.Error("expected no tool registered with nil client")
	}

	// And on the happy path the tool IS registered.
	r.RegisterAzureCloudTools(NewMockAzureClient())
	if _, ok := r.Get("query_azure_vms"); !ok {
		t.Error("expected query_azure_vms registered after RegisterAzureCloudTools with mock")
	}
}
