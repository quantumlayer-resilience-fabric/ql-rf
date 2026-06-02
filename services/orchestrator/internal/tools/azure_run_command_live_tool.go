// PR #28 / CONN-008 — Azure Run Command live tool.
//
// The Azure twin of PR #21's ssm_send_patch_command_live. Registered ONLY
// when main.go's registerAzureLiveRunCommandTools succeeds, which itself
// requires:
//
//   - RF_CONNECTORS_AZURE_ALLOW_LIVE_RUN_COMMAND=true
//   - RF_CONNECTORS_AZURE_FALLBACK_TO_MOCK=false
//   - RF_CONNECTORS_AZURE_LIVE_RUN_COMMAND_WHITELIST_VMS non-empty
//
// Even with all three set, the tool's Execute method re-validates the
// whitelist (defense in depth) and the OPA tool_authorization policy
// blocks invocation unless `state_change_prod` tasks have two distinct
// approvers set on the ai_plan row (the two-approver workflow from
// PR #21).
//
// The tool reuses PR #27's AzureRunCommandClient + AzureRunCommandPlan
// to build the command plan structure, then hands the plan to the
// LiveAzureRunCommandClient for the SDK call. Two clients in one tool:
// the dry-run plan-builder and the live sender. This keeps the
// construction path identical to the dry-run tool so the audit log
// records the same shape, distinguished only by `dry_run`.
package tools

import (
	"context"
	"fmt"
)

// AzureRunCommandLiveTool fires the real Azure VirtualMachine Run
// Command path for a target VM. The Execute method:
//
//  1. Coerces params (resource_group, vm_name, command_id, script) —
//     same shape as the dry-run tool.
//  2. Calls dryClient.BuildRunCommandPlan to validate + construct the plan.
//  3. Calls liveClient.SendRunCommand to fire it.
//  4. Returns an audit envelope marked `dry_run:false, real_changes:true`.
//
// The envelope shape mirrors the dry-run tool's so SQL queries over
// `ai_tool_invocations` can distinguish the two variants by one field.
type AzureRunCommandLiveTool struct {
	dryClient  AzureRunCommandClient
	liveClient LiveAzureRunCommandClient
}

// NewAzureRunCommandLiveTool wires both clients. Both must be non-nil —
// RegisterAzureLiveRunCommandTools enforces this.
func NewAzureRunCommandLiveTool(dry AzureRunCommandClient, live LiveAzureRunCommandClient) *AzureRunCommandLiveTool {
	return &AzureRunCommandLiveTool{dryClient: dry, liveClient: live}
}

func (t *AzureRunCommandLiveTool) Name() string {
	return "azure_run_command_live"
}

func (t *AzureRunCommandLiveTool) Description() string {
	return "Fire a real Azure VM Run Command. Requires two-approver workflow. Whitelisted VMs only."
}

func (t *AzureRunCommandLiveTool) Risk() RiskLevel {
	return RiskStateChangeProd
}

func (t *AzureRunCommandLiveTool) Scope() Scope {
	return ScopeOrganization
}

// Idempotent is false. A real Run Command creates a new operation on
// every call; replaying the same params produces a second Azure
// execution, not a no-op. The executor is responsible for not
// double-firing approved plans.
func (t *AzureRunCommandLiveTool) Idempotent() bool {
	return false
}

// RequiresApproval is true AND the OPA policy enforces TWO distinct
// approvers for state_change_prod tools. The handler layer
// (coApproveTask from PR #21) is the runtime gate; this method only
// reports the requirement to consumers of the tools listing API.
func (t *AzureRunCommandLiveTool) RequiresApproval() bool {
	return true
}

// Parameters mirror the dry-run tool exactly. The audit log can then
// compare dry and live params field-by-field for the same logical
// command.
func (t *AzureRunCommandLiveTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"resource_group": map[string]any{
				"type":        "string",
				"description": "Azure resource group containing the target VM.",
			},
			"vm_name": map[string]any{
				"type":        "string",
				"description": "Azure VM name. Must be on the boot-loaded whitelist (rg/vm format).",
			},
			"command_id": map[string]any{
				"type":        "string",
				"description": "Run Command document: RunShellScript (Linux) or RunPowerShellScript (Windows).",
				"enum":        []string{azureRunCommandRunShellScript, azureRunCommandRunPowerShell},
				"default":     azureRunCommandRunShellScript,
			},
			"script": map[string]any{
				"type":        "string",
				"description": "Inline script body for the run command document.",
			},
		},
		"required": []string{"resource_group", "vm_name"},
	}
}

// Execute builds the run-command plan (via the dry-run client) then
// sends it (via the live client). Returns an envelope explicitly marked
// `dry_run:false` and `real_changes:true`, plus the operation token for
// traceability. The audit row captures both fields verbatim.
func (t *AzureRunCommandLiveTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	if t.dryClient == nil || t.liveClient == nil {
		return nil, fmt.Errorf("azure_run_command_live: clients not configured")
	}

	req := AzureRunCommandRequest{}
	if v, ok := params["resource_group"].(string); ok {
		req.ResourceGroup = v
	}
	if v, ok := params["vm_name"].(string); ok {
		req.VMName = v
	}
	if v, ok := params["command_id"].(string); ok {
		req.CommandID = v
	}
	if v, ok := params["script"].(string); ok {
		req.Script = v
	}

	plan, err := t.dryClient.BuildRunCommandPlan(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("build plan: %w", err)
	}

	opToken, err := t.liveClient.SendRunCommand(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("live send: %w", err)
	}

	// The live envelope mirrors PR #27's dry-run envelope structure so
	// SQL queries over ai_tool_invocations.result can extract
	// `command_plan` the same way for both. The key difference is
	// `dry_run:false`, `real_changes:true`, and the presence of
	// `operation_token`.
	return map[string]any{
		"dry_run":         false,
		"real_changes":    true,
		"would_invoke":    "armcompute.VirtualMachineRunCommandsClient.BeginCreateOrUpdate",
		"did_invoke":      true,
		"operation_token": opToken,
		"command_plan":    plan,
		"explanation":     "PR #28: live Azure Run Command fired. Requires two-approver workflow + whitelist (both enforced).",
	}, nil
}

// RegisterAzureLiveRunCommandTools registers the live Azure Run Command
// tool on the registry. Both clients must be non-nil — the dry client
// builds the plan, the live client sends it. Called by main.go's
// registerAzureLiveRunCommandTools ONLY after the env opt-in + whitelist
// + mock-conflict gates have all passed.
func (r *Registry) RegisterAzureLiveRunCommandTools(dry AzureRunCommandClient, live LiveAzureRunCommandClient) {
	if dry == nil || live == nil {
		r.log.Warn("RegisterAzureLiveRunCommandTools called with nil client; no live Azure run-command tools registered",
			"dry_nil", dry == nil, "live_nil", live == nil)
		return
	}
	r.register(NewAzureRunCommandLiveTool(dry, live))
	r.log.Info("azure live run-command tools: registered", "tools", []string{"azure_run_command_live"})
}
