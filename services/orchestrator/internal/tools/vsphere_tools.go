// PR #33 / CONN-012 — vSphere tools (read-only).
//
// First real vSphere cloud-touching tool in the orchestrator. Registered
// only when a VSphereClient was successfully constructed at boot.
// Read-only by API contract — listing VMs cannot modify state.
//
// PR #34 / #35 will add the dry-run + live state-change vSphere tools
// (vsphere_run_guest_command + *_live) following the same arc the AWS,
// Azure, and GCP connectors used.
package tools

import (
	"context"
	"fmt"
)

// QueryVSphereVMsTool lists VMs in the configured vCenter via a
// VSphereClient (real or mock). Risk = read_only.
//
//nolint:dupl // structural duplication with azure_tools.go / gcp_tools.go is deliberate — each cloud's tool metadata reads as its own audit-by-grep surface.
type QueryVSphereVMsTool struct {
	client VSphereClient
}

func NewQueryVSphereVMsTool(client VSphereClient) *QueryVSphereVMsTool {
	return &QueryVSphereVMsTool{client: client}
}

func (t *QueryVSphereVMsTool) Name() string { return "query_vsphere_vms" }

func (t *QueryVSphereVMsTool) Description() string {
	return "List vSphere virtual machines visible to the configured vCenter (read-only)."
}

func (t *QueryVSphereVMsTool) Risk() RiskLevel { return RiskReadOnly }

func (t *QueryVSphereVMsTool) Scope() Scope { return ScopeOrganization }

func (t *QueryVSphereVMsTool) Idempotent() bool { return true }

func (t *QueryVSphereVMsTool) RequiresApproval() bool { return false }

// Parameters: no inputs required. The vCenter is fixed at boot via config.
func (t *QueryVSphereVMsTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

// Execute calls VSphereClient.ListVMs and wraps the result.
func (t *QueryVSphereVMsTool) Execute(ctx context.Context, _ map[string]any) (any, error) {
	if t.client == nil {
		return nil, fmt.Errorf("query_vsphere_vms: client not configured")
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

// RegisterVSphereCloudTools registers vSphere-backed cloud tools.
func (r *Registry) RegisterVSphereCloudTools(client VSphereClient) {
	if client == nil {
		r.log.Warn("RegisterVSphereCloudTools called with nil client; no vSphere tools registered")
		return
	}
	r.register(NewQueryVSphereVMsTool(client))
	r.log.Info("vsphere tools: registered", "tools", []string{"query_vsphere_vms"})
}
