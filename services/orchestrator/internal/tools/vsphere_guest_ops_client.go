// PR #34 / CONN-013 — vSphere guest-ops client (DRY-RUN ONLY).
//
// SAFETY (READ THIS BEFORE EDITING):
// This file is the vSphere equivalent of the SSM / Azure / GCP dry-run
// clients. It builds guest-program-run plans as plain Go structs and
// never calls the state-change SDK path. PR #35 will introduce
// `live_vsphere_guest_ops_client.go` as the SOLE caller of the
// state-change method — the structural test in
// no_vsphere_guest_ops_sdk_import_test.go enforces this by name.
//
// The structural test for vSphere follows the Azure / GCP pattern:
// function-name matching rather than import-path forbidding, because
// govmomi is already legitimately imported by PR #33's read-only path.
package tools

import (
	"context"
	"fmt"
	"regexp"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// VSphereGuestOpsClient builds (does not start) vSphere guest-OS
// programs.
type VSphereGuestOpsClient interface {
	BuildGuestProgramPlan(ctx context.Context, req VSphereGuestProgramRequest) (*VSphereGuestProgramPlan, error)
}

// VSphereGuestProgramRequest is the typed input. Validation lives here.
type VSphereGuestProgramRequest struct {
	VMName string
	// GuestUser / GuestPassword — credentials for the guest OS. Stored
	// in the plan so the live client can pass them; never logged.
	GuestUser     string
	GuestPassword string
	// ProgramPath — absolute path of the binary to run inside the guest
	// (e.g. "/bin/bash", "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe").
	ProgramPath string
	// Arguments — passed verbatim to the program.
	Arguments string
	// WorkingDirectory — optional cwd for the program.
	WorkingDirectory string
}

// VSphereGuestProgramPlan mirrors the relevant fields of
// `types.GuestProgramSpec` but is OUR own struct — never the SDK type.
type VSphereGuestProgramPlan struct {
	VMName           string `json:"vm_name"`
	GuestUser        string `json:"guest_user"`
	ProgramPath      string `json:"program_path"`
	Arguments        string `json:"arguments,omitempty"`
	WorkingDirectory string `json:"working_directory,omitempty"`
	// GuestPassword is intentionally NOT in JSON output — auditors
	// shouldn't see the credential in `result` JSONB. The live client
	// reads it from a side-channel (the plan struct in memory).
	GuestPassword string `json:"-"`
	Comment       string `json:"comment,omitempty"`
	DryRun        bool   `json:"dry_run"`
	RealChanges   bool   `json:"real_changes"`
}

// realVSphereGuestOpsClient validates the request and constructs the
// plan. No SDK imports. No network calls.
type realVSphereGuestOpsClient struct {
	log *logger.Logger
}

// NewRealVSphereGuestOpsClient constructs the validation-only client.
func NewRealVSphereGuestOpsClient(log *logger.Logger) VSphereGuestOpsClient {
	return &realVSphereGuestOpsClient{log: log.WithComponent("vsphere-guest-ops")}
}

// BuildGuestProgramPlan validates the request and constructs the plan.
func (c *realVSphereGuestOpsClient) BuildGuestProgramPlan(_ context.Context, req VSphereGuestProgramRequest) (*VSphereGuestProgramPlan, error) {
	if err := validateVSphereVMName(req.VMName); err != nil {
		return nil, err
	}
	if req.GuestUser == "" {
		return nil, fmt.Errorf("guest_user is required")
	}
	if req.GuestPassword == "" {
		return nil, fmt.Errorf("guest_password is required (will be passed through to the live SDK call but never logged or audited)")
	}
	if err := validateVSphereProgramPath(req.ProgramPath); err != nil {
		return nil, err
	}

	return &VSphereGuestProgramPlan{
		VMName:           req.VMName,
		GuestUser:        req.GuestUser,
		ProgramPath:      req.ProgramPath,
		Arguments:        req.Arguments,
		WorkingDirectory: req.WorkingDirectory,
		GuestPassword:    req.GuestPassword,
		Comment:          "QL-RF dry-run (PR #34): constructed without invocation.",
		DryRun:           true,
		RealChanges:      false,
	}, nil
}

// mockVSphereGuestOpsClient returns a deterministic plan.
type mockVSphereGuestOpsClient struct{}

// NewMockVSphereGuestOpsClient constructs the deterministic fixture client.
func NewMockVSphereGuestOpsClient() VSphereGuestOpsClient {
	return &mockVSphereGuestOpsClient{}
}

// BuildGuestProgramPlan returns a fixed plan regardless of input.
func (m *mockVSphereGuestOpsClient) BuildGuestProgramPlan(_ context.Context, req VSphereGuestProgramRequest) (*VSphereGuestProgramPlan, error) {
	prog := req.ProgramPath
	if prog == "" {
		prog = "/bin/bash"
	}
	return &VSphereGuestProgramPlan{
		VMName:        "mock-esx-vm-prod-01",
		GuestUser:     "mock-guest-user",
		ProgramPath:   prog,
		Arguments:     req.Arguments,
		GuestPassword: "mock-guest-password",
		Comment:       "QL-RF dry-run mock (PR #34): no real VMs.",
		DryRun:        true,
		RealChanges:   false,
	}, nil
}

// validateVSphereVMName — VM names in vSphere can contain spaces and
// most printable chars; the only hard constraint is non-empty + at most
// 80 chars. The full name validation lives in govmomi at call time.
var vsphereVMNamePattern = regexp.MustCompile(`^[\w\-. ]{1,80}$`)

func validateVSphereVMName(name string) error {
	if name == "" {
		return fmt.Errorf("vm_name is required")
	}
	if !vsphereVMNamePattern.MatchString(name) {
		return fmt.Errorf("invalid vm_name %q: must match vSphere naming rules (letters/digits/hyphen/period/space, 1-80 chars)", name)
	}
	return nil
}

// validateVSphereProgramPath — program path must be an absolute path.
// We accept both POSIX (`/bin/bash`) and Windows (`C:\\Windows\\...`)
// forms.
func validateVSphereProgramPath(p string) error {
	if p == "" {
		return fmt.Errorf("program_path is required")
	}
	// POSIX absolute or Windows drive letter.
	if p[0] != '/' && !(len(p) >= 2 && p[1] == ':') {
		return fmt.Errorf("invalid program_path %q: must be an absolute path", p)
	}
	return nil
}
