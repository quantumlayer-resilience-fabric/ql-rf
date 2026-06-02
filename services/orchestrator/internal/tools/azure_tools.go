// PR #26 / CONN-006 — Azure tools (read-only).
//
// First real Azure cloud-touching tool in the orchestrator. Registered
// only when an AzureClient was successfully constructed at boot (see
// main.go's registerAzureCloudTools); otherwise the tool is absent from
// the registry and GET /api/v1/ai/tools doesn't list it. Read-only by
// API contract — VirtualMachines List cannot modify cloud state.
//
// PR #27 / #28 add the dry-run + live state-change Azure tools
// (azure_run_command + azure_run_command_live) following the same arc
// PR #19 → #20 → #21 used for AWS SSM.
package tools

import (
	"context"
	"fmt"
)

// QueryAzureVMsTool lists virtual machines in the configured Azure
// subscription via a configured AzureClient (real or mock). Risk =
// read_only. Idempotent. Never modifies cloud state.
type QueryAzureVMsTool struct {
	client AzureClient
}

// NewQueryAzureVMsTool constructs the tool with a backing AzureClient.
// The caller is responsible for choosing real vs mock (see main.go).
func NewQueryAzureVMsTool(client AzureClient) *QueryAzureVMsTool {
	return &QueryAzureVMsTool{client: client}
}

func (t *QueryAzureVMsTool) Name() string {
	return "query_azure_vms"
}

func (t *QueryAzureVMsTool) Description() string {
	return "List Azure virtual machines in the configured subscription (read-only)."
}

func (t *QueryAzureVMsTool) Risk() RiskLevel {
	return RiskReadOnly
}

func (t *QueryAzureVMsTool) Scope() Scope {
	return ScopeOrganization
}

func (t *QueryAzureVMsTool) Idempotent() bool {
	return true
}

func (t *QueryAzureVMsTool) RequiresApproval() bool {
	return false
}

// Parameters: no inputs required. The subscription is fixed at boot via
// config; tools that need per-call subscription targeting can be added
// later if customers ask. Keeping the input shape empty matches the
// minimum-surface-area discipline of the AWS read-only tool.
func (t *QueryAzureVMsTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

// Execute calls AzureClient.ListVMs and wraps the result in an audit-
// log-friendly envelope. The tool surfaces only the redacted AzureVM
// projection — no extension data, no NIC IDs, no disk URIs.
func (t *QueryAzureVMsTool) Execute(ctx context.Context, _ map[string]any) (any, error) {
	if t.client == nil {
		return nil, fmt.Errorf("query_azure_vms: client not configured")
	}

	vms, err := t.client.ListVMs(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"vm_count": len(vms),
		"vms":      vms,
	}, nil
}

// RegisterAzureCloudTools registers Azure-backed cloud tools on the
// registry. Called from main.go ONLY after an AzureClient was
// successfully constructed (real or mock-with-fallback). Safe to call
// zero or one time per registry lifetime.
//
// PR #27/#28 will add `azure_run_command*` registrations through their
// own files (mirroring how PR #20/#21's SSM tools added
// RegisterStateChangeTools / RegisterLiveStateChangeTools).
func (r *Registry) RegisterAzureCloudTools(client AzureClient) {
	if client == nil {
		r.log.Warn("RegisterAzureCloudTools called with nil client; no Azure tools registered")
		return
	}
	r.register(NewQueryAzureVMsTool(client))
	r.log.Info("azure tools: registered", "tools", []string{"query_azure_vms"})
}
