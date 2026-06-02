// PR #31 / CONN-011 — GCP OS Config patch live tool.
//
// The GCP twin of PR #21's ssm_send_patch_command_live and PR #28's
// azure_run_command_live. Registered ONLY when main.go's
// registerGCPLivePatchTools succeeds, which itself requires:
//
//   - RF_CONNECTORS_GCP_ALLOW_LIVE_PATCH=true
//   - RF_CONNECTORS_GCP_FALLBACK_TO_MOCK=false
//   - RF_CONNECTORS_GCP_LIVE_PATCH_WHITELIST_INSTANCE_FILTERS non-empty
//
// Even with all three set, the tool's Execute method re-validates the
// whitelist (defense in depth) and the OPA policy blocks invocation
// unless `state_change_prod` tasks have two distinct approvers set on
// the ai_plan row.
package tools

import (
	"context"
	"fmt"
)

// GCPPatchJobLiveTool fires the real GCP OS Config patch job path.
//
//nolint:dupl // structural duplication with ssm_live + azure_live tools is deliberate — each cloud's live tool reads as its own audit-by-grep surface.
type GCPPatchJobLiveTool struct {
	dryClient  GCPPatchClient
	liveClient LiveGCPPatchClient
}

// NewGCPPatchJobLiveTool wires both clients. Both must be non-nil.
func NewGCPPatchJobLiveTool(dry GCPPatchClient, live LiveGCPPatchClient) *GCPPatchJobLiveTool {
	return &GCPPatchJobLiveTool{dryClient: dry, liveClient: live}
}

func (t *GCPPatchJobLiveTool) Name() string {
	return "gcp_os_config_patch_live"
}

func (t *GCPPatchJobLiveTool) Description() string {
	return "Fire a real GCP OS Config patch job. Requires two-approver workflow. Whitelisted zone+filter pairs only."
}

func (t *GCPPatchJobLiveTool) Risk() RiskLevel {
	return RiskStateChangeProd
}

func (t *GCPPatchJobLiveTool) Scope() Scope {
	return ScopeOrganization
}

// Idempotent is false. A real ExecutePatchJob creates a new patch_job
// on every call.
func (t *GCPPatchJobLiveTool) Idempotent() bool {
	return false
}

func (t *GCPPatchJobLiveTool) RequiresApproval() bool {
	return true
}

func (t *GCPPatchJobLiveTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{
				"type":        "string",
				"description": "GCP project ID containing the target instances.",
			},
			"zone": map[string]any{
				"type":        "string",
				"description": "GCP zone (e.g. us-central1-a). Must form an allowed zone:filter pair with instance_filter.",
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

// Execute builds the patch job plan (via the dry-run client) then sends
// it (via the live client). Returns an envelope marked `dry_run:false`
// and `real_changes:true`, plus the patch job's resource name for
// traceability.
func (t *GCPPatchJobLiveTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	if t.dryClient == nil || t.liveClient == nil {
		return nil, fmt.Errorf("gcp_os_config_patch_live: clients not configured")
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

	plan, err := t.dryClient.BuildPatchJobPlan(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("build plan: %w", err)
	}

	jobName, err := t.liveClient.SendPatchJob(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("live send: %w", err)
	}

	return map[string]any{
		"dry_run":        false,
		"real_changes":   true,
		"would_invoke":   "osconfig.Client.ExecutePatchJob",
		"did_invoke":     true,
		"patch_job_name": jobName,
		"patch_plan":     plan,
		"explanation":    "PR #31: live GCP patch job fired. Requires two-approver workflow + whitelist (both enforced).",
	}, nil
}

// RegisterGCPLivePatchTools registers the live GCP patch tool. Both
// clients must be non-nil.
func (r *Registry) RegisterGCPLivePatchTools(dry GCPPatchClient, live LiveGCPPatchClient) {
	if dry == nil || live == nil {
		r.log.Warn("RegisterGCPLivePatchTools called with nil client; no live GCP patch tools registered",
			"dry_nil", dry == nil, "live_nil", live == nil)
		return
	}
	r.register(NewGCPPatchJobLiveTool(dry, live))
	r.log.Info("gcp live patch tools: registered", "tools", []string{"gcp_os_config_patch_live"})
}
