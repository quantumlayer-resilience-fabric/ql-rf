// PR #33 / CONN-012 — unit tests for QueryVSphereVMsTool + helpers.

package tools

import (
	"context"
	"strings"
	"testing"
)

func TestQueryVSphereVMsTool_Shape(t *testing.T) {
	tool := NewQueryVSphereVMsTool(NewMockVSphereClient())
	if tool.Name() != "query_vsphere_vms" {
		t.Errorf("Name = %q, want query_vsphere_vms", tool.Name())
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

func TestQueryVSphereVMsTool_ExecuteWithMockClient_ReturnsTwoFixtureVMs(t *testing.T) {
	tool := NewQueryVSphereVMsTool(NewMockVSphereClient())
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
	vms, ok := m["vms"].([]VSphereVM)
	if !ok || len(vms) != 2 {
		t.Fatalf("vms = %T (len %d), want []VSphereVM of length 2", m["vms"], len(vms))
	}
	for _, vm := range vms {
		if !strings.HasPrefix(vm.Name, "mock-esx-vm-") {
			t.Errorf("mock returned non-mock VM name %q", vm.Name)
		}
		if vm.Tags["mock_origin"] != "ql-rf-test" {
			t.Errorf("mock VM %q missing mock_origin tag", vm.Name)
		}
	}
}

func TestQueryVSphereVMsTool_NilClient(t *testing.T) {
	tool := NewQueryVSphereVMsTool(nil)
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("Execute with nil client: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "client not configured") {
		t.Errorf("error = %q, want substring 'client not configured'", err.Error())
	}
}

func TestRegisterVSphereCloudTools_NoOpOnNilClient(t *testing.T) {
	r := NewRegistry(nil, testLoggerForSSM())
	r.RegisterVSphereCloudTools(nil)
	if _, ok := r.Get("query_vsphere_vms"); ok {
		t.Error("expected no tool registered with nil client")
	}
	r.RegisterVSphereCloudTools(NewMockVSphereClient())
	if _, ok := r.Get("query_vsphere_vms"); !ok {
		t.Error("expected query_vsphere_vms registered after RegisterVSphereCloudTools with mock")
	}
}
