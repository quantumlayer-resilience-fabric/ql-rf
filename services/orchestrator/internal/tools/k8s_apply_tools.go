// PR #39 / CONN-016 — K8s server-side-apply dry-run tool.
//
// Mirrors PR #20 (SSM), PR #27 (Azure), PR #30 (GCP), and PR #34 (vSphere)
// exactly. State-change tool (risk = state_change_prod, RequiresApproval
// = true) but the dry-run path constructs the apply plan as a plain Go
// struct via K8sApplyClient and never reaches the state-change SDK
// method.
package tools

import (
	"context"
	"fmt"
)

// K8sApplyDryRunTool constructs (does not apply) Kubernetes server-side
// apply plans.
//
//nolint:dupl // structural duplication with other clouds' state-change tools is deliberate.
type K8sApplyDryRunTool struct {
	client K8sApplyClient
}

// NewK8sApplyDryRunTool wires the tool with its backing client.
func NewK8sApplyDryRunTool(client K8sApplyClient) *K8sApplyDryRunTool {
	return &K8sApplyDryRunTool{client: client}
}

func (t *K8sApplyDryRunTool) Name() string {
	return "k8s_apply"
}

func (t *K8sApplyDryRunTool) Description() string {
	return "Construct a Kubernetes server-side-apply plan (dry-run only in PR #39)."
}

func (t *K8sApplyDryRunTool) Risk() RiskLevel {
	return RiskStateChangeProd
}

func (t *K8sApplyDryRunTool) Scope() Scope {
	return ScopeOrganization
}

func (t *K8sApplyDryRunTool) Idempotent() bool {
	return true
}

func (t *K8sApplyDryRunTool) RequiresApproval() bool {
	return true
}

func (t *K8sApplyDryRunTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"namespace": map[string]any{
				"type":        "string",
				"description": "Target namespace (DNS-1123 label rules).",
			},
			"manifest": map[string]any{
				"type":        "string",
				"description": "JSON-encoded Kubernetes object to apply.",
			},
			"field_manager": map[string]any{
				"type":        "string",
				"description": "Server-side-apply field manager identifier. Typically \"ql-rf\".",
			},
			"force": map[string]any{
				"type":        "boolean",
				"description": "Force conflict resolution against other field managers. Default false.",
			},
		},
		"required": []string{"namespace", "manifest", "field_manager"},
	}
}

// Execute coerces params, delegates to K8sApplyClient.BuildApplyPlan,
// and wraps the resulting plan in an audit-friendly envelope.
func (t *K8sApplyDryRunTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	if t.client == nil {
		return nil, fmt.Errorf("k8s_apply: client not configured")
	}

	req := K8sApplyRequest{}
	if v, ok := params["namespace"].(string); ok {
		req.Namespace = v
	}
	if v, ok := params["manifest"].(string); ok {
		req.Manifest = v
	}
	if v, ok := params["field_manager"].(string); ok {
		req.FieldManager = v
	}
	if v, ok := params["force"].(bool); ok {
		req.Force = v
	}

	plan, err := t.client.BuildApplyPlan(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"dry_run":      true,
		"real_changes": false,
		"would_invoke": "k8s.io/client-go server-side-apply (clientset.<group>().Apply with metav1.ApplyOptions)",
		"apply_plan":   plan,
		"explanation":  "PR #39: server-side-apply plan constructed without invocation. Live execution requires PR #40's env+whitelist+two-approver gates.",
	}, nil
}

// RegisterK8sApplyDryRunTools registers K8s apply dry-run tools.
func (r *Registry) RegisterK8sApplyDryRunTools(client K8sApplyClient) {
	if client == nil {
		r.log.Warn("RegisterK8sApplyDryRunTools called with nil client; no k8s apply tools registered")
		return
	}
	r.register(NewK8sApplyDryRunTool(client))
	r.log.Info("k8s apply tools: registered", "tools", []string{"k8s_apply"})
}
