// PR #40 / CONN-017 — K8s server-side-apply live tool.
//
// The K8s twin of PR #21's ssm_send_patch_command_live, PR #28's
// azure_run_command_live, PR #31's gcp_os_config_patch_live, and
// PR #35's vsphere_run_guest_program_live. Registered ONLY when
// main.go's registerK8sLiveApplyTools succeeds (env opt-in + non-empty
// whitelist + non-conflicting fallback_to_mock).
package tools

import (
	"context"
	"fmt"
)

// K8sApplyLiveTool fires the real K8s server-side-apply path.
//
//nolint:dupl // structural duplication with other clouds' live tools is deliberate.
type K8sApplyLiveTool struct {
	dryClient  K8sApplyClient
	liveClient LiveK8sApplyClient
}

// NewK8sApplyLiveTool wires both clients.
func NewK8sApplyLiveTool(dry K8sApplyClient, live LiveK8sApplyClient) *K8sApplyLiveTool {
	return &K8sApplyLiveTool{dryClient: dry, liveClient: live}
}

func (t *K8sApplyLiveTool) Name() string {
	return "k8s_apply_live"
}

func (t *K8sApplyLiveTool) Description() string {
	return "Apply a real Kubernetes manifest via server-side apply. Requires two-approver workflow. Whitelisted (namespace, kind) pairs only."
}

func (t *K8sApplyLiveTool) Risk() RiskLevel {
	return RiskStateChangeProd
}

func (t *K8sApplyLiveTool) Scope() Scope {
	return ScopeOrganization
}

func (t *K8sApplyLiveTool) Idempotent() bool {
	return false
}

func (t *K8sApplyLiveTool) RequiresApproval() bool {
	return true
}

func (t *K8sApplyLiveTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"namespace": map[string]any{
				"type":        "string",
				"description": "Target namespace. Must be on the boot-loaded whitelist as \"namespace/Kind\".",
			},
			"manifest": map[string]any{
				"type":        "string",
				"description": "JSON-encoded Kubernetes object to apply.",
			},
			"field_manager": map[string]any{
				"type":        "string",
				"description": "Server-side-apply field manager identifier.",
			},
			"force": map[string]any{
				"type":        "boolean",
				"description": "Force conflict resolution. Default false.",
			},
		},
		"required": []string{"namespace", "manifest", "field_manager"},
	}
}

// Execute builds the apply plan via the dry-run client, then fires the
// live SDK call against the same plan.
func (t *K8sApplyLiveTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	if t.dryClient == nil || t.liveClient == nil {
		return nil, fmt.Errorf("k8s_apply_live: clients not configured")
	}

	req := K8sApplyRequest{}
	manifest := ""
	if v, ok := params["namespace"].(string); ok {
		req.Namespace = v
	}
	if v, ok := params["manifest"].(string); ok {
		req.Manifest = v
		manifest = v
	}
	if v, ok := params["field_manager"].(string); ok {
		req.FieldManager = v
	}
	if v, ok := params["force"].(bool); ok {
		req.Force = v
	}

	plan, err := t.dryClient.BuildApplyPlan(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("build plan: %w", err)
	}

	result, err := t.liveClient.Apply(ctx, plan, manifest)
	if err != nil {
		return nil, fmt.Errorf("live apply: %w", err)
	}

	return map[string]any{
		"dry_run":      false,
		"real_changes": true,
		"would_invoke": "k8s.io/client-go dynamic Apply (server-side)",
		"did_invoke":   true,
		"applied_uid":  result.UID,
		"resource_ver": result.ResourceVersion,
		"applied_at":   result.AppliedAt,
		"apply_plan":   plan,
		"explanation":  "PR #40: live K8s server-side-apply fired. Requires two-approver workflow + whitelist (both enforced).",
	}, nil
}

// RegisterK8sLiveApplyTools registers the live tool.
func (r *Registry) RegisterK8sLiveApplyTools(dry K8sApplyClient, live LiveK8sApplyClient) {
	if dry == nil || live == nil {
		r.log.Warn("RegisterK8sLiveApplyTools called with nil client; no live k8s apply tools registered",
			"dry_nil", dry == nil, "live_nil", live == nil)
		return
	}
	r.register(NewK8sApplyLiveTool(dry, live))
	r.log.Info("k8s live apply tools: registered", "tools", []string{"k8s_apply_live"})
}
