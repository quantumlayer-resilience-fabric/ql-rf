// PR #29 / CONN-009 — unit tests for QueryGCPInstancesTool + helpers.

package tools

import (
	"context"
	"strings"
	"testing"
)

func TestQueryGCPInstancesTool_Shape(t *testing.T) {
	tool := NewQueryGCPInstancesTool(NewMockGCPClient())
	if tool.Name() != "query_gcp_instances" {
		t.Errorf("Name = %q, want query_gcp_instances", tool.Name())
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

func TestQueryGCPInstancesTool_ExecuteWithMockClient_ReturnsTwoFixtureInstances(t *testing.T) {
	tool := NewQueryGCPInstancesTool(NewMockGCPClient())
	got, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("Execute returned %T, want map[string]any", got)
	}
	if m["instance_count"] != 2 {
		t.Errorf("instance_count = %v, want 2 (mock fixture)", m["instance_count"])
	}
	instances, ok := m["instances"].([]GCPInstance)
	if !ok || len(instances) != 2 {
		t.Fatalf("instances = %T (len %d), want []GCPInstance of length 2", m["instances"], len(instances))
	}
	for _, inst := range instances {
		if !strings.HasPrefix(inst.Name, "mock-gce-") {
			t.Errorf("mock returned non-mock instance name %q", inst.Name)
		}
		if inst.Labels["mock_origin"] != "ql-rf-test" {
			t.Errorf("mock instance %q missing mock_origin label", inst.Name)
		}
	}
}

func TestQueryGCPInstancesTool_NilClient(t *testing.T) {
	tool := NewQueryGCPInstancesTool(nil)
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("Execute with nil client: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "client not configured") {
		t.Errorf("error = %q, want substring 'client not configured'", err.Error())
	}
}

func TestLastPathSegment(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"projects/p/zones/us-central1-a/machineTypes/e2-medium", "e2-medium"},
		{"e2-medium", "e2-medium"},
		{"", ""},
		{"/", ""},
	}
	for _, c := range cases {
		got := lastPathSegment(c.in)
		if got != c.want {
			t.Errorf("lastPathSegment(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRegisterGCPCloudTools_NoOpOnNilClient(t *testing.T) {
	r := NewRegistry(nil, testLoggerForSSM())
	r.RegisterGCPCloudTools(nil)
	if _, ok := r.Get("query_gcp_instances"); ok {
		t.Error("expected no tool registered with nil client")
	}
	r.RegisterGCPCloudTools(NewMockGCPClient())
	if _, ok := r.Get("query_gcp_instances"); !ok {
		t.Error("expected query_gcp_instances registered after RegisterGCPCloudTools with mock")
	}
}
