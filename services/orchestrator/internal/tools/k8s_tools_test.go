// PR #38 / CONN-015 — unit tests for the K8s read-only tool.
package tools

import (
	"context"
	"testing"
)

func TestQueryPodsTool_Metadata(t *testing.T) {
	tool := NewQueryPodsTool(NewMockKubernetesClient())
	if tool.Name() != "query_pods" {
		t.Errorf("Name = %q, want query_pods", tool.Name())
	}
	if tool.Risk() != RiskReadOnly {
		t.Errorf("Risk = %v, want RiskReadOnly", tool.Risk())
	}
	if tool.RequiresApproval() {
		t.Error("RequiresApproval should be false for read_only tool")
	}
	if !tool.Idempotent() {
		t.Error("Idempotent should be true for read_only tool")
	}
}

func TestQueryPodsTool_ExecuteReturnsPodList(t *testing.T) {
	tool := NewQueryPodsTool(NewMockKubernetesClient())
	got, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	envelope, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("envelope type = %T, want map[string]any", got)
	}
	if envelope["pod_count"] != 2 {
		t.Errorf("pod_count = %v, want 2", envelope["pod_count"])
	}
	pods, ok := envelope["pods"].([]PodInfo)
	if !ok {
		t.Fatalf("pods type = %T, want []PodInfo", envelope["pods"])
	}
	if len(pods) != 2 {
		t.Errorf("pods length = %d, want 2", len(pods))
	}
}

func TestQueryPodsTool_ExecuteRefusesNilClient(t *testing.T) {
	tool := &QueryPodsTool{client: nil}
	if _, err := tool.Execute(context.Background(), nil); err == nil {
		t.Error("expected error for nil client, got nil")
	}
}

func TestRegisterK8sCloudTools_RegistersWithClient(t *testing.T) {
	r := NewRegistry(nil, testLoggerForSSM())
	r.RegisterK8sCloudTools(NewMockKubernetesClient())
	if _, ok := r.Get("query_pods"); !ok {
		t.Error("query_pods should be registered")
	}
}

func TestRegisterK8sCloudTools_NoOpOnNilClient(t *testing.T) {
	r := NewRegistry(nil, testLoggerForSSM())
	r.RegisterK8sCloudTools(nil)
	if _, ok := r.Get("query_pods"); ok {
		t.Error("query_pods should NOT be registered when client is nil")
	}
}
