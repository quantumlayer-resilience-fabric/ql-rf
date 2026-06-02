// PR #38 / CONN-015 — unit tests for the K8s read-only client.
package tools

import (
	"context"
	"testing"
)

func TestMockKubernetesClient_ReturnsDeterministicFixture(t *testing.T) {
	c := NewMockKubernetesClient()
	pods, err := c.ListPods(context.Background())
	if err != nil {
		t.Fatalf("ListPods: %v", err)
	}
	if len(pods) != 2 {
		t.Fatalf("pod count = %d, want 2", len(pods))
	}

	first := pods[0]
	if first.Name != "mock-app-prod-7d4f8c-abcde" {
		t.Errorf("first.Name = %q, want mock-app-prod-7d4f8c-abcde", first.Name)
	}
	if first.Namespace != "prod" {
		t.Errorf("first.Namespace = %q, want prod", first.Namespace)
	}
	if first.Phase != "Running" {
		t.Errorf("first.Phase = %q, want Running", first.Phase)
	}
	if first.Labels["mock_origin"] != "ql-rf-test" {
		t.Errorf("mock origin marker missing on mock pod; got labels=%v", first.Labels)
	}
	if len(first.Images) == 0 {
		t.Error("first.Images should be non-empty")
	}
}

func TestMockKubernetesClient_Deterministic(t *testing.T) {
	c := NewMockKubernetesClient()
	a, err := c.ListPods(context.Background())
	if err != nil {
		t.Fatalf("ListPods (a): %v", err)
	}
	b, err := c.ListPods(context.Background())
	if err != nil {
		t.Fatalf("ListPods (b): %v", err)
	}
	if len(a) != len(b) {
		t.Fatalf("non-deterministic count: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].UID != b[i].UID {
			t.Errorf("pod %d non-deterministic UID: %q vs %q", i, a[i].UID, b[i].UID)
		}
	}
}
