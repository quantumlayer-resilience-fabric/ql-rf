// PR #30 / CONN-010 — GCP OS Config patch dry-run tool.
//
// Mirrors PR #20's ssm_send_patch_command and PR #27's
// azure_run_command exactly. State-change tool (risk = state_change_prod,
// RequiresApproval = true) but the dry-run path constructs the patch
// job plan as a plain Go struct via GCPPatchClient and never reaches
// the state-change SDK's ExecutePatchJob method.
//
// Invocable ONLY via PR #20's /api/v1/ai/tools/{name}/dry-run endpoint.
// Live mode (real ExecutePatchJob) lands as PR #31 with stronger gates.
package tools

import (
	"context"
	"fmt"
)

// GCPPatchJobDryRunTool constructs (does not send) GCP OS Config patch
// jobs.
//
//nolint:dupl // structural duplication with ssm_tools.go / azure_run_command_tools.go is deliberate — each cloud's state-change tool reads as its own audit-by-grep surface.
type GCPPatchJobDryRunTool struct {
	client GCPPatchClient
}

// NewGCPPatchJobDryRunTool wires the tool with its backing client.
func NewGCPPatchJobDryRunTool(client GCPPatchClient) *GCPPatchJobDryRunTool {
	return &GCPPatchJobDryRunTool{client: client}
}

func (t *GCPPatchJobDryRunTool) Name() string {
	return "gcp_os_config_patch"
}

func (t *GCPPatchJobDryRunTool) Description() string {
	return "Construct a GCP OS Config patch job plan (dry-run only in PR #30)."
}

func (t *GCPPatchJobDryRunTool) Risk() RiskLevel {
	return RiskStateChangeProd
}

func (t *GCPPatchJobDryRunTool) Scope() Scope {
	return ScopeOrganization
}

func (t *GCPPatchJobDryRunTool) Idempotent() bool {
	return true
}

func (t *GCPPatchJobDryRunTool) RequiresApproval() bool {
	return true
}

func (t *GCPPatchJobDryRunTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{
				"type":        "string",
				"description": "GCP project ID containing the target instances.",
			},
			"zone": map[string]any{
				"type":        "string",
				"description": "GCP zone (e.g. us-central1-a).",
			},
			"instance_filter": map[string]any{
				"type":        "string",
				"description": "GCE label filter selecting instances (e.g. labels.env=prod).",
			},
			"reboot_config": map[string]any{
				"type":        "string",
				"description": "Reboot policy: DEFAULT, ALWAYS, or NEVER.",
				"enum":        []string{gcpRebootDefault, gcpRebootAlways, gcpRebootNever},
				"default":     gcpRebootDefault,
			},
			"display_name": map[string]any{
				"type":        "string",
				"description": "Operator-friendly job name.",
			},
		},
		"required": []string{"project_id", "zone", "instance_filter"},
	}
}

// Execute coerces params, delegates to GCPPatchClient.BuildPatchJobPlan,
// and wraps the resulting plan in an audit-friendly envelope.
func (t *GCPPatchJobDryRunTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	if t.client == nil {
		return nil, fmt.Errorf("gcp_os_config_patch: client not configured")
	}

	req := GCPPatchJobRequest{}
	if v, ok := params["project_id"].(string); ok {
		req.ProjectID = v
	}
	if v, ok := params["zone"].(string); ok {
		req.Zone = v
	}
	if v, ok := params["instance_filter"].(string); ok {
		req.InstanceFilter = v
	}
	if v, ok := params["reboot_config"].(string); ok {
		req.RebootConfig = v
	}
	if v, ok := params["display_name"].(string); ok {
		req.DisplayName = v
	}

	plan, err := t.client.BuildPatchJobPlan(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"dry_run":      true,
		"real_changes": false,
		"would_invoke": "osconfig.Client.ExecutePatchJob",
		"patch_plan":   plan,
		"explanation":  "PR #30: patch job constructed without invocation. Live ExecutePatchJob requires PR #31's env+whitelist+two-approver gates.",
	}, nil
}

// RegisterGCPPatchDryRunTools registers GCP OS Config dry-run tools on
// the registry. Called from main.go ONLY after a GCPPatchClient has been
// constructed.
func (r *Registry) RegisterGCPPatchDryRunTools(client GCPPatchClient) {
	if client == nil {
		r.log.Warn("RegisterGCPPatchDryRunTools called with nil client; no GCP patch tools registered")
		return
	}
	r.register(NewGCPPatchJobDryRunTool(client))
	r.log.Info("gcp patch tools: registered", "tools", []string{"gcp_os_config_patch"})
}
