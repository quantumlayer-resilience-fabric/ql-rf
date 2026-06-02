// PR #27 / CONN-007 — Azure Run Command client (DRY-RUN ONLY).
//
// SAFETY (READ THIS BEFORE EDITING):
// This file is the Azure equivalent of PR #20's ssm_client.go. It builds
// run-command plans as plain Go structs and never calls the Azure SDK's
// state-change RunCommand path. PR #28 will introduce
// `live_azure_runcommand_client.go` as the SOLE caller of the
// state-change run-command client constructor — the structural test in
// no_azure_runcommand_sdk_import_test.go enforces this by name. The same
// SDK isolation discipline that made the SSM live path mechanically safe
// to review applies here.
//
// Why the SDK isolation is different shape for Azure:
//
//   - SSM (PR #20): the entire `service/ssm` package was new to the tools
//     package; the SSM structural test forbade the whole import path.
//   - Azure Run Command (PR #27): `armcompute/v5` is ALREADY in the tools
//     package because PR #26's `azure_client.go` uses `VirtualMachinesClient`
//     for read-only listing. So the structural test forbids the SPECIFIC
//     state-change-client constructor function name (the entry point
//     for the state-change client), not the whole package import.
//
// Live mode (real Run Command) lands as PR #28 with stronger gates (env
// opt-in, per-VM whitelist, two-approver workflow).
package tools

import (
	"context"
	"fmt"
	"regexp"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// AzureRunCommandClient builds (does not send) Azure VM Run Command
// requests. The interface is narrow — one method, one read of the SDK's
// shape, zero network calls. PR #28's live client implementation will
// satisfy the same interface plus an extra Send method behind explicit
// safety gates.
type AzureRunCommandClient interface {
	BuildRunCommandPlan(ctx context.Context, req AzureRunCommandRequest) (*AzureRunCommandPlan, error)
}

// AzureRunCommandRequest is the typed input to BuildRunCommandPlan.
// Keeping it as a struct (not loose params) means validation lives at
// the boundary, not inside the tool's params-coercion code.
type AzureRunCommandRequest struct {
	ResourceGroup string
	VMName        string
	// CommandID — the canonical Azure Run Command document id. Must be
	// one of the supported values (azureRunCommandIDs constants below).
	CommandID string
	// Script — for documents that take inline shell scripts. Kept off
	// the audit log description body to avoid leaking command content;
	// stored verbatim on the plan for forensic purposes.
	Script string
}

// AzureRunCommandPlan mirrors the relevant fields of
// `armcompute.RunCommandInput` and `VirtualMachineRunCommand` but is OUR
// own struct — never the SDK type. The audit log records this verbatim.
// DryRun and RealChanges are always true / false respectively in PR #27;
// PR #28's live mode will flip them per-invocation.
type AzureRunCommandPlan struct {
	ResourceGroup string `json:"resource_group"`
	VMName        string `json:"vm_name"`
	CommandID     string `json:"command_id"`
	// Script content. Captured for the audit log. PR #28's live path
	// will also use this verbatim when constructing the real SDK call.
	Script string `json:"script,omitempty"`
	// TimeoutSeconds — the Azure SDK's default if unset is 90s. We pin
	// a longer value for patching scripts; auditors can read the plan
	// and see exactly how long the operation could have run.
	TimeoutSeconds int32 `json:"timeout_seconds"`
	// Comment — operator-friendly note that lands in the SDK call's
	// `--debug` output and our audit row's description.
	Comment string `json:"comment,omitempty"`
	DryRun  bool   `json:"dry_run"`
	// RealChanges — the symmetry counterpart of DryRun. Both fields exist
	// so a typo-driven "dry_run=false" doesn't pass for live; live mode
	// must set BOTH (dry_run=false AND real_changes=true).
	RealChanges bool `json:"real_changes"`
}

// Supported Azure Run Command document IDs. Hoisted to constants so the
// validator + the tool's parameter enum + future docs all reference the
// same list.
const (
	azureRunCommandRunShellScript = "RunShellScript"      // Linux
	azureRunCommandRunPowerShell  = "RunPowerShellScript" // Windows
)

// realAzureRunCommandClient validates the request and constructs the
// plan. No SDK imports. No network calls. Just struct construction +
// basic validation. The "real" in the name is relative: this client
// builds what the live Run Command would receive, but never sends it.
// The mock client below is a more aggressive variant for CI; both are
// dry-run.
type realAzureRunCommandClient struct {
	log *logger.Logger
}

// NewRealAzureRunCommandClient constructs the validation-only client.
// Always succeeds — there's nothing to fail at boot, since no network
// calls are made.
func NewRealAzureRunCommandClient(log *logger.Logger) AzureRunCommandClient {
	return &realAzureRunCommandClient{log: log.WithComponent("azure-runcommand")}
}

// BuildRunCommandPlan validates the request and constructs the plan.
// Returns errors for malformed VM names or unsupported command IDs. The
// plan always carries DryRun:true and RealChanges:false — PR #28 is what
// flips those.
func (c *realAzureRunCommandClient) BuildRunCommandPlan(_ context.Context, req AzureRunCommandRequest) (*AzureRunCommandPlan, error) {
	if err := validateAzureResourceGroupName(req.ResourceGroup); err != nil {
		return nil, err
	}
	if err := validateAzureVMName(req.VMName); err != nil {
		return nil, err
	}

	cmdID := req.CommandID
	if cmdID == "" {
		cmdID = azureRunCommandRunShellScript
	}
	if cmdID != azureRunCommandRunShellScript && cmdID != azureRunCommandRunPowerShell {
		return nil, fmt.Errorf("command_id must be %q or %q, got %q",
			azureRunCommandRunShellScript, azureRunCommandRunPowerShell, cmdID)
	}

	return &AzureRunCommandPlan{
		ResourceGroup:  req.ResourceGroup,
		VMName:         req.VMName,
		CommandID:      cmdID,
		Script:         req.Script,
		TimeoutSeconds: 3600,
		Comment:        "QL-RF dry-run (PR #27): constructed without invocation.",
		DryRun:         true,
		RealChanges:    false,
	}, nil
}

// mockAzureRunCommandClient returns a deterministic plan tagged with a
// `mock-vm-*` name so the mock origin is obvious in the audit log. Used
// by unit tests and by CI (where `RF_CONNECTORS_AZURE_FALLBACK_TO_MOCK=true`
// gates Azure credentials anyway).
//
// The mock client is intentionally MORE strict than the real client about
// what it accepts — it ignores caller-provided VM names and always
// returns the same fixture. This guarantees CI tests are deterministic
// regardless of input.
type mockAzureRunCommandClient struct{}

// NewMockAzureRunCommandClient constructs the deterministic fixture client.
func NewMockAzureRunCommandClient() AzureRunCommandClient {
	return &mockAzureRunCommandClient{}
}

// BuildRunCommandPlan returns a fixed plan with the mock VM fixture,
// regardless of input. Used by unit tests and CI.
func (m *mockAzureRunCommandClient) BuildRunCommandPlan(_ context.Context, req AzureRunCommandRequest) (*AzureRunCommandPlan, error) {
	cmdID := req.CommandID
	if cmdID == "" {
		cmdID = azureRunCommandRunShellScript
	}
	return &AzureRunCommandPlan{
		ResourceGroup:  "rg-mock-prod",
		VMName:         "mock-vm-prod-01",
		CommandID:      cmdID,
		Script:         req.Script,
		TimeoutSeconds: 3600,
		Comment:        "QL-RF dry-run mock (PR #27): no real VMs.",
		DryRun:         true,
		RealChanges:    false,
	}, nil
}

// Azure resource naming rules (subset): 1-90 chars, alphanumeric +
// hyphen + underscore + period, must not end with period. The full
// regex matches Azure ARM's validation closely enough for our boundary
// check; the SDK will reject anything else before the live call too.
var (
	azureRGNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.\-()]{1,89}[a-zA-Z0-9_\-()]$`)
	azureVMNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-]{0,62}[a-zA-Z0-9]$`)
)

// validateAzureResourceGroupName returns a descriptive error if the name
// doesn't match Azure's resource group naming rules. Defensive — garbage
// in the audit log is worse than no entry at all.
func validateAzureResourceGroupName(name string) error {
	if name == "" {
		return fmt.Errorf("resource_group is required")
	}
	if !azureRGNamePattern.MatchString(name) {
		return fmt.Errorf("invalid resource_group %q: must match Azure naming rules", name)
	}
	return nil
}

// validateAzureVMName returns a descriptive error if the name doesn't
// match Azure VM naming rules (1-64 chars, alphanumeric + hyphen, must
// not start/end with hyphen).
func validateAzureVMName(name string) error {
	if name == "" {
		return fmt.Errorf("vm_name is required")
	}
	if !azureVMNamePattern.MatchString(name) {
		return fmt.Errorf("invalid vm_name %q: must match Azure VM naming rules", name)
	}
	return nil
}
