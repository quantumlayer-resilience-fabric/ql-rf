// PR #35 / CONN-014 — vSphere guest-ops live tool.
//
// The vSphere twin of PR #21's ssm_send_patch_command_live, PR #28's
// azure_run_command_live, and PR #31's gcp_os_config_patch_live.
// Registered ONLY when main.go's registerVSphereGuestOpsLiveTools
// succeeds (env opt-in + non-empty whitelist + non-conflicting
// fallback_to_mock).
package tools

import (
	"context"
	"fmt"
)

// VSphereGuestProgramLiveTool fires the real vSphere guest-program path.
//
//nolint:dupl // structural duplication with other clouds' live tools is deliberate.
type VSphereGuestProgramLiveTool struct {
	dryClient  VSphereGuestOpsClient
	liveClient LiveVSphereGuestOpsClient
}

// NewVSphereGuestProgramLiveTool wires both clients.
func NewVSphereGuestProgramLiveTool(dry VSphereGuestOpsClient, live LiveVSphereGuestOpsClient) *VSphereGuestProgramLiveTool {
	return &VSphereGuestProgramLiveTool{dryClient: dry, liveClient: live}
}

func (t *VSphereGuestProgramLiveTool) Name() string {
	return "vsphere_run_guest_program_live"
}

func (t *VSphereGuestProgramLiveTool) Description() string {
	return "Run a real vSphere guest-OS program. Requires two-approver workflow. Whitelisted VMs only."
}

func (t *VSphereGuestProgramLiveTool) Risk() RiskLevel {
	return RiskStateChangeProd
}

func (t *VSphereGuestProgramLiveTool) Scope() Scope {
	return ScopeOrganization
}

func (t *VSphereGuestProgramLiveTool) Idempotent() bool {
	return false
}

func (t *VSphereGuestProgramLiveTool) RequiresApproval() bool {
	return true
}

func (t *VSphereGuestProgramLiveTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"vm_name": map[string]any{
				"type":        "string",
				"description": "vSphere VM name. Must be on the boot-loaded whitelist.",
			},
			"guest_user": map[string]any{
				"type":        "string",
				"description": "Guest OS username for authentication.",
			},
			"guest_password": map[string]any{
				"type":        "string",
				"description": "Guest OS password. Stored in the plan in-memory; never serialized.",
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
				"description": "Optional working directory.",
			},
		},
		"required": []string{"vm_name", "guest_user", "guest_password", "program_path"},
	}
}

// Execute builds the program-run plan then fires the live SDK call.
func (t *VSphereGuestProgramLiveTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	if t.dryClient == nil || t.liveClient == nil {
		return nil, fmt.Errorf("vsphere_run_guest_program_live: clients not configured")
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

	plan, err := t.dryClient.BuildGuestProgramPlan(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("build plan: %w", err)
	}

	pid, err := t.liveClient.RunGuestProgram(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("live send: %w", err)
	}

	return map[string]any{
		"dry_run":      false,
		"real_changes": true,
		"would_invoke": "vsphere.ProcessManager.StartProgram",
		"did_invoke":   true,
		"guest_pid":    pid,
		"program_plan": plan,
		"explanation":  "PR #35: live vSphere guest program fired. Requires two-approver workflow + whitelist (both enforced).",
	}, nil
}

// RegisterVSphereLiveGuestOpsTools registers the live tool.
func (r *Registry) RegisterVSphereLiveGuestOpsTools(dry VSphereGuestOpsClient, live LiveVSphereGuestOpsClient) {
	if dry == nil || live == nil {
		r.log.Warn("RegisterVSphereLiveGuestOpsTools called with nil client; no live vSphere guest-ops tools registered",
			"dry_nil", dry == nil, "live_nil", live == nil)
		return
	}
	r.register(NewVSphereGuestProgramLiveTool(dry, live))
	r.log.Info("vsphere live guest-ops tools: registered", "tools", []string{"vsphere_run_guest_program_live"})
}
