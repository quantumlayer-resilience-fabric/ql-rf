// PR #27 / CONN-007 — Azure Run Command dry-run tool.
//
// Mirrors PR #20's ssm_send_patch_command exactly. State-change tool
// (risk = state_change_prod, RequiresApproval = true) but the dry-run
// path constructs the command plan as a plain Go struct via the
// AzureRunCommandClient interface and never reaches the SDK's
// state-change RunCommand methods. The structural safety test in
// no_azure_runcommand_sdk_import_test.go enforces this by name.
//
// Invocable ONLY via PR #20's /api/v1/ai/tools/{name}/dry-run endpoint —
// PR #19's /invoke strictly rejects state-change tools by design. Live
// mode (real Azure VirtualMachineRunCommandsClient.BeginCreateOrUpdate)
// lands as PR #28 with stronger gates (env opt-in, per-VM whitelist,
// two-approver workflow).
package tools

import (
	"context"
	"fmt"
)

// AzureRunCommandDryRunTool constructs (does not send) Azure VM Run
// Command requests. The Execute method returns an envelope explicitly
// marked dry_run / real_changes:false so audit-log consumers can
// distinguish this from a hypothetical future live invocation.
type AzureRunCommandDryRunTool struct {
	client AzureRunCommandClient
}

// NewAzureRunCommandDryRunTool wires the tool with its backing client.
func NewAzureRunCommandDryRunTool(client AzureRunCommandClient) *AzureRunCommandDryRunTool {
	return &AzureRunCommandDryRunTool{client: client}
}

func (t *AzureRunCommandDryRunTool) Name() string {
	return "azure_run_command"
}

func (t *AzureRunCommandDryRunTool) Description() string {
	return "Construct an Azure VM Run Command (dry-run only in PR #27)."
}

func (t *AzureRunCommandDryRunTool) Risk() RiskLevel {
	return RiskStateChangeProd
}

func (t *AzureRunCommandDryRunTool) Scope() Scope {
	return ScopeOrganization
}

func (t *AzureRunCommandDryRunTool) Idempotent() bool {
	// Dry-run is idempotent — same input always produces the same plan.
	return true
}

func (t *AzureRunCommandDryRunTool) RequiresApproval() bool {
	// Production state change. OPA policy enforces this; the dry-run
	// endpoint adds an additional structural check (state-change-only).
	return true
}

func (t *AzureRunCommandDryRunTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"resource_group": map[string]any{
				"type":        "string",
				"description": "Azure resource group containing the target VM.",
			},
			"vm_name": map[string]any{
				"type":        "string",
				"description": "Azure VM name (validated against Azure naming rules).",
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

// Execute coerces params, delegates to AzureRunCommandClient.BuildRunCommandPlan,
// and wraps the resulting plan in an audit-friendly envelope that makes
// the dry-run nature unmissable.
//
// Always returns dry_run:true and real_changes:false. PR #28's live mode
// introduces a separate tool with a different envelope; this Execute
// method has no flag that could flip to live behavior.
func (t *AzureRunCommandDryRunTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	if t.client == nil {
		return nil, fmt.Errorf("azure_run_command: client not configured")
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

	plan, err := t.client.BuildRunCommandPlan(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"dry_run":      true,
		"real_changes": false,
		"would_invoke": "armcompute.VirtualMachineRunCommandsClient.BeginCreateOrUpdate",
		"command_plan": plan,
		"explanation":  "PR #27: command constructed without invocation. Live BeginCreateOrUpdate requires PR #28's env+whitelist+two-approver gates.",
	}, nil
}

// RegisterAzureRunCommandDryRunTools registers Azure Run Command dry-run
// tools on the registry. Called from main.go ONLY after an
// AzureRunCommandClient has been constructed (real or mock). Safe to call
// zero or one time per registry lifetime.
//
// The state-change tools registered here are invocable ONLY via the
// /api/v1/ai/tools/{name}/dry-run endpoint (PR #20). PR #19's /invoke
// endpoint structurally rejects them via its read-only-only gate.
func (r *Registry) RegisterAzureRunCommandDryRunTools(client AzureRunCommandClient) {
	if client == nil {
		r.log.Warn("RegisterAzureRunCommandDryRunTools called with nil client; no Azure run-command tools registered")
		return
	}
	r.register(NewAzureRunCommandDryRunTool(client))
	r.log.Info("azure run-command tools: registered", "tools", []string{"azure_run_command"})
}
