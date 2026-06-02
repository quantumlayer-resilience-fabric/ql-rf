// PR #20 / CONN-002 — SSM patch tool (state-change, dry-run only).
//
// First state-change cloud tool in the orchestrator. Registered when an
// SSMClient is available (always, since the client makes no network calls
// and needs no credentials). Risk = state_change_prod; OPA policy gates
// approval-required production tools. Invocable only via the new
// /api/v1/ai/tools/{name}/dry-run endpoint — PR #19's /invoke endpoint
// strictly rejects state-change tools by design.
//
// Live mode (real ssm:SendCommand) lands as a follow-up PR with stronger
// gates (env opt-in, per-instance whitelist, two-approver workflow).
package tools

import (
	"context"
	"fmt"
)

// SSMSendPatchCommandTool constructs (does not send) AWS SSM patch
// commands. The Execute method returns an envelope explicitly marked
// dry_run / real_changes:false so audit-log consumers can distinguish
// this from a hypothetical future live invocation.
type SSMSendPatchCommandTool struct {
	client SSMClient
}

// NewSSMSendPatchCommandTool wires the tool with its backing client.
func NewSSMSendPatchCommandTool(client SSMClient) *SSMSendPatchCommandTool {
	return &SSMSendPatchCommandTool{client: client}
}

func (t *SSMSendPatchCommandTool) Name() string {
	return "ssm_send_patch_command"
}

func (t *SSMSendPatchCommandTool) Description() string {
	return "Construct an AWS SSM AWS-RunPatchBaseline command (dry-run only in PR #20)."
}

func (t *SSMSendPatchCommandTool) Risk() RiskLevel {
	return RiskStateChangeProd
}

func (t *SSMSendPatchCommandTool) Scope() Scope {
	return ScopeOrganization
}

func (t *SSMSendPatchCommandTool) Idempotent() bool {
	// Dry-run is idempotent — same input always produces the same plan.
	return true
}

func (t *SSMSendPatchCommandTool) RequiresApproval() bool {
	// Production state change. OPA policy enforces this; the dry-run
	// endpoint adds an additional structural check (state-change-only).
	return true
}

func (t *SSMSendPatchCommandTool) Parameters() map[string]any {
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
				"description": "EC2 instance IDs to target (validated against i-[hex]{8,17}).",
				"items":       map[string]any{"type": "string"},
			},
			"operation": map[string]any{
				"type":        "string",
				"description": "Patch operation: Scan (default) or Install.",
				"enum":        []string{"Scan", "Install"},
				"default":     "Scan",
			},
		},
		"required": []string{"instance_ids"},
	}
}

// Execute coerces params, delegates to SSMClient.BuildPatchCommand, and
// wraps the resulting plan in an audit-friendly envelope that makes the
// dry-run nature unmissable.
//
// Always returns dry_run:true and real_changes:false. PR #21's live mode
// will introduce a separate tool (or extend this one) with a different
// envelope; for now, this Execute method has no flag that could flip to
// live behavior.
func (t *SSMSendPatchCommandTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	if t.client == nil {
		return nil, fmt.Errorf("ssm_send_patch_command: client not configured")
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

	plan, err := t.client.BuildPatchCommand(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"dry_run":      true,
		"real_changes": false,
		"would_invoke": "ssm:SendCommand",
		"command_plan": plan,
		"explanation":  "PR #20: command constructed without invocation. Live SendCommand requires PR #21's env+whitelist+two-approver gates.",
	}, nil
}

// RegisterStateChangeTools registers SSM-backed state-change tools on the
// registry. Called from main.go ONLY after an SSMClient has been
// constructed (real or mock). Safe to call zero or one time per registry
// lifetime.
//
// The state-change tools registered here are invocable ONLY via the
// /api/v1/ai/tools/{name}/dry-run endpoint in PR #20. PR #19's /invoke
// endpoint structurally rejects them via its read-only-only gate.
func (r *Registry) RegisterStateChangeTools(client SSMClient) {
	if client == nil {
		r.log.Warn("RegisterStateChangeTools called with nil client; no SSM tools registered")
		return
	}
	r.register(NewSSMSendPatchCommandTool(client))
	r.log.Info("ssm tools: registered", "tools", []string{"ssm_send_patch_command"})
}
