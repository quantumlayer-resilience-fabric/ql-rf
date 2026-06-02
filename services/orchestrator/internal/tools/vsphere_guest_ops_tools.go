// PR #34 / CONN-013 — vSphere guest-ops dry-run tool.
//
// Mirrors PR #20 (SSM), PR #27 (Azure), and PR #30 (GCP) exactly.
// State-change tool (risk = state_change_prod, RequiresApproval = true)
// but the dry-run path constructs the program-run plan as a plain Go
// struct via VSphereGuestOpsClient and never reaches the state-change
// SDK method.
package tools

import (
	"context"
	"fmt"
)

// VSphereGuestProgramDryRunTool constructs (does not start) vSphere
// guest-OS programs.
//
//nolint:dupl // structural duplication with other clouds' state-change tools is deliberate.
type VSphereGuestProgramDryRunTool struct {
	client VSphereGuestOpsClient
}

// NewVSphereGuestProgramDryRunTool wires the tool with its backing client.
func NewVSphereGuestProgramDryRunTool(client VSphereGuestOpsClient) *VSphereGuestProgramDryRunTool {
	return &VSphereGuestProgramDryRunTool{client: client}
}

func (t *VSphereGuestProgramDryRunTool) Name() string {
	return "vsphere_run_guest_program"
}

func (t *VSphereGuestProgramDryRunTool) Description() string {
	return "Construct a vSphere guest-OS program-run plan (dry-run only in PR #34)."
}

func (t *VSphereGuestProgramDryRunTool) Risk() RiskLevel {
	return RiskStateChangeProd
}

func (t *VSphereGuestProgramDryRunTool) Scope() Scope {
	return ScopeOrganization
}

func (t *VSphereGuestProgramDryRunTool) Idempotent() bool {
	return true
}

func (t *VSphereGuestProgramDryRunTool) RequiresApproval() bool {
	return true
}

func (t *VSphereGuestProgramDryRunTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"vm_name": map[string]any{
				"type":        "string",
				"description": "vSphere VM name (validated against naming rules).",
			},
			"guest_user": map[string]any{
				"type":        "string",
				"description": "Guest OS username for authentication.",
			},
			"guest_password": map[string]any{
				"type":        "string",
				"description": "Guest OS password. Stored in the plan in-memory; never serialized to audit JSON.",
			},
			"program_path": map[string]any{
				"type":        "string",
				"description": "Absolute path of the program to run inside the guest.",
			},
			"arguments": map[string]any{
				"type":        "string",
				"description": "Arguments passed to the program verbatim.",
			},
			"working_directory": map[string]any{
				"type":        "string",
				"description": "Optional working directory for the program.",
			},
		},
		"required": []string{"vm_name", "guest_user", "guest_password", "program_path"},
	}
}

// Execute coerces params, delegates to VSphereGuestOpsClient.BuildGuestProgramPlan,
// and wraps the resulting plan in an audit-friendly envelope.
func (t *VSphereGuestProgramDryRunTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	if t.client == nil {
		return nil, fmt.Errorf("vsphere_run_guest_program: client not configured")
	}

	req := VSphereGuestProgramRequest{}
	if v, ok := params["vm_name"].(string); ok {
		req.VMName = v
	}
	if v, ok := params["guest_user"].(string); ok {
		req.GuestUser = v
	}
	if v, ok := params["guest_password"].(string); ok {
		req.GuestPassword = v
	}
	if v, ok := params["program_path"].(string); ok {
		req.ProgramPath = v
	}
	if v, ok := params["arguments"].(string); ok {
		req.Arguments = v
	}
	if v, ok := params["working_directory"].(string); ok {
		req.WorkingDirectory = v
	}

	plan, err := t.client.BuildGuestProgramPlan(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"dry_run":      true,
		"real_changes": false,
		"would_invoke": "vsphere.GuestOperationsManager process-manager start-program-in-guest",
		"program_plan": plan,
		"explanation":  "PR #34: program-run plan constructed without invocation. Live execution requires PR #35's env+whitelist+two-approver gates.",
	}, nil
}

// RegisterVSphereGuestOpsDryRunTools registers vSphere guest-ops dry-run
// tools on the registry.
func (r *Registry) RegisterVSphereGuestOpsDryRunTools(client VSphereGuestOpsClient) {
	if client == nil {
		r.log.Warn("RegisterVSphereGuestOpsDryRunTools called with nil client; no vSphere guest-ops tools registered")
		return
	}
	r.register(NewVSphereGuestProgramDryRunTool(client))
	r.log.Info("vsphere guest-ops tools: registered", "tools", []string{"vsphere_run_guest_program"})
}
