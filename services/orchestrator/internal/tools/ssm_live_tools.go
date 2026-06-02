// PR #21 / CONN-003 — SSM live patch tool.
//
// The first tool in the orchestrator that fires a real, cloud-mutating
// API call. Registered ONLY when main.go's registerSSMLiveTools succeeds,
// which itself requires:
//
//   - RF_CONNECTORS_AWS_ALLOW_LIVE_PATCH=true
//   - RF_CONNECTORS_AWS_FALLBACK_TO_MOCK=false
//   - RF_CONNECTORS_AWS_LIVE_PATCH_WHITELIST_INSTANCE_IDS non-empty
//
// Even with all three set, the tool's Execute method re-validates the
// whitelist (defense in depth) and the OPA tool_authorization policy
// blocks invocation unless `state_change_prod` tasks have two distinct
// approvers set on the ai_plan row.
//
// The tool reuses PR #20's SSMClient + SSMCommandPlan to build the
// command structure, then hands the plan to the LiveSSMClient for the
// SDK call. Two clients in one tool: the dry builder and the live sender.
// This keeps the construction path identical to the dry-run tool so the
// audit log records the same shape, distinguished only by `dry_run`.
package tools

import (
	"context"
	"fmt"
)

// SSMSendPatchCommandLiveTool fires the real AWS SSM SendCommand for an
// AWS-RunPatchBaseline document. The Execute method:
//
//  1. Coerces params (region, instance_ids, operation) — same shape as
//     the dry-run tool.
//  2. Calls dryClient.BuildPatchCommand to validate + construct the plan.
//  3. Calls liveClient.SendCommand to fire it.
//  4. Returns an audit envelope marked `dry_run:false, real_changes:true`.
//
// The envelope shape mirrors the dry-run tool's so SQL queries over
// `ai_tool_invocations` can distinguish the two variants by one field.
type SSMSendPatchCommandLiveTool struct {
	dryClient  SSMClient
	liveClient LiveSSMClient
}

// NewSSMSendPatchCommandLiveTool wires both clients. Both must be non-nil
// — RegisterLiveStateChangeTools enforces this.
func NewSSMSendPatchCommandLiveTool(dry SSMClient, live LiveSSMClient) *SSMSendPatchCommandLiveTool {
	return &SSMSendPatchCommandLiveTool{dryClient: dry, liveClient: live}
}

func (t *SSMSendPatchCommandLiveTool) Name() string {
	return "ssm_send_patch_command_live"
}

func (t *SSMSendPatchCommandLiveTool) Description() string {
	return "Fire a real AWS SSM AWS-RunPatchBaseline command. Requires two-approver workflow. Whitelisted instances only."
}

func (t *SSMSendPatchCommandLiveTool) Risk() RiskLevel {
	return RiskStateChangeProd
}

func (t *SSMSendPatchCommandLiveTool) Scope() Scope {
	return ScopeOrganization
}

// Idempotent is false. A real SendCommand creates a new command_id on every
// call — replaying the same params produces a second AWS execution, not a
// no-op. The executor is responsible for not double-firing approved plans.
func (t *SSMSendPatchCommandLiveTool) Idempotent() bool {
	return false
}

// RequiresApproval is true AND the OPA policy enforces TWO distinct
// approvers for state_change_prod tools. The handler layer
// (coApproveTask) is the runtime gate; this method only reports the
// requirement to consumers of the tools listing API.
func (t *SSMSendPatchCommandLiveTool) RequiresApproval() bool {
	return true
}

// Parameters mirror the dry-run tool exactly. The audit log can then
// compare dry and live params field-by-field for the same logical patch.
func (t *SSMSendPatchCommandLiveTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"region": map[string]any{
				"type":        "string",
				"description": "AWS region (defaults to us-east-1).",
				"default":     "us-east-1",
			},
			"instance_ids": map[string]any{
				"type":        "array",
				"description": "EC2 instance IDs to target. Every ID must be on the boot-loaded whitelist.",
				"items":       map[string]any{"type": "string"},
			},
			"operation": map[string]any{
				"type":        "string",
				"description": "Patch operation: Scan (safer) or Install (actually patches).",
				"enum":        []string{patchOpScan, patchOpInstall},
				"default":     patchOpScan,
			},
		},
		"required": []string{"instance_ids"},
	}
}

// Execute builds the SSM command plan (via the dry-run client) then sends
// it (via the live client). Returns an envelope explicitly marked
// `dry_run:false` and `real_changes:true`, plus the AWS-returned
// command_id for traceability. The audit row in ai_tool_invocations
// captures both fields verbatim.
func (t *SSMSendPatchCommandLiveTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	if t.dryClient == nil || t.liveClient == nil {
		return nil, fmt.Errorf("ssm_send_patch_command_live: clients not configured")
	}

	req := PatchCommandRequest{}
	if v, ok := params["region"].(string); ok {
		req.Region = v
	}
	if v, ok := params["operation"].(string); ok {
		req.Operation = v
	}
	if raw, ok := params["instance_ids"].([]any); ok {
		for _, item := range raw {
			if s, ok := item.(string); ok {
				req.InstanceIDs = append(req.InstanceIDs, s)
			}
		}
	}

	plan, err := t.dryClient.BuildPatchCommand(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("build plan: %w", err)
	}

	commandID, err := t.liveClient.SendCommand(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("live send: %w", err)
	}

	// The live envelope mirrors PR #20's dry-run envelope structure so SQL
	// queries over ai_tool_invocations.result can extract `command_plan`
	// the same way for both. The key difference is `dry_run:false`,
	// `real_changes:true`, and the presence of `command_id`.
	return map[string]any{
		"dry_run":      false,
		"real_changes": true,
		"would_invoke": "ssm:SendCommand",
		"did_invoke":   true,
		"command_id":   commandID,
		"command_plan": plan,
		"explanation":  "PR #21: live ssm:SendCommand fired. Requires two-approver workflow + whitelist (both enforced).",
	}, nil
}

// RegisterLiveStateChangeTools registers the live SSM tool on the registry.
// Both clients must be non-nil — the dry client builds the plan, the live
// client sends it. Called by main.go's registerSSMLiveTools ONLY after the
// env opt-in + whitelist + mock-conflict gates have all passed.
func (r *Registry) RegisterLiveStateChangeTools(dry SSMClient, live LiveSSMClient) {
	if dry == nil || live == nil {
		r.log.Warn("RegisterLiveStateChangeTools called with nil client; no live SSM tools registered",
			"dry_nil", dry == nil, "live_nil", live == nil)
		return
	}
	r.register(NewSSMSendPatchCommandLiveTool(dry, live))
	r.log.Info("ssm live tools: registered", "tools", []string{"ssm_send_patch_command_live"})
}
