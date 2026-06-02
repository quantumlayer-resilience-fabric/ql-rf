// PR #38 / CONN-015 — Kubernetes tools (read-only).
//
// First real K8s cloud-touching tool in the orchestrator. Registered
// only when a KubernetesClient was successfully constructed at boot.
// Read-only by API contract — listing pods cannot modify state.
//
// PR #39 / #40 will add the dry-run + live state-change K8s tools
// (k8s_apply_dry_run + k8s_apply_live) following the same arc the
// AWS, Azure, GCP, and vSphere connectors used.
package tools

import (
	"context"
	"fmt"
)

// QueryPodsTool lists pods in the configured Kubernetes cluster via a
// KubernetesClient (real or mock). Risk = read_only.
//
//nolint:dupl // structural duplication with vsphere_tools.go / gcp_tools.go is deliberate — each cloud's tool metadata reads as its own audit-by-grep surface.
type QueryPodsTool struct {
	client KubernetesClient
}

func NewQueryPodsTool(client KubernetesClient) *QueryPodsTool {
	return &QueryPodsTool{client: client}
}

func (t *QueryPodsTool) Name() string { return "query_pods" }

func (t *QueryPodsTool) Description() string {
	return "List Kubernetes pods visible to the configured cluster (read-only)."
}

func (t *QueryPodsTool) Risk() RiskLevel { return RiskReadOnly }

func (t *QueryPodsTool) Scope() Scope { return ScopeOrganization }

func (t *QueryPodsTool) Idempotent() bool { return true }

func (t *QueryPodsTool) RequiresApproval() bool { return false }

// Parameters: no inputs required. The cluster is fixed at boot via config.
func (t *QueryPodsTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

// Execute calls KubernetesClient.ListPods and wraps the result.
func (t *QueryPodsTool) Execute(ctx context.Context, _ map[string]any) (any, error) {
	if t.client == nil {
		return nil, fmt.Errorf("query_pods: client not configured")
	}

	pods, err := t.client.ListPods(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"pod_count": len(pods),
		"pods":      pods,
	}, nil
}

// RegisterK8sCloudTools registers Kubernetes-backed cloud tools.
func (r *Registry) RegisterK8sCloudTools(client KubernetesClient) {
	if client == nil {
		r.log.Warn("RegisterK8sCloudTools called with nil client; no k8s tools registered")
		return
	}
	r.register(NewQueryPodsTool(client))
	r.log.Info("k8s tools: registered", "tools", []string{"query_pods"})
}
