// PR #27 / CONN-007 — structural safety test for Azure Run Command.
//
// Mechanically verifies that no file in the tools package references
// `NewVirtualMachineRunCommandsClient` — the SDK constructor for Azure's
// state-change Run Command client. Calling it would expose
// `BeginCreateOrUpdate`, which actually executes scripts on production
// VMs. PR #28 introduces a dedicated file
// (`live_azure_runcommand_client.go`) that DOES reference the
// constructor; when that PR lands, this test will be updated to
// whitelist that single file by name (same shape as PR #20/#21's SSM
// structural test).
//
// Why a name-based check instead of an import-path forbid:
//
//   - PR #26's azure_client.go ALREADY imports armcompute/v5 for the
//     read-only VirtualMachinesClient. The package import itself is
//     legitimate; what we need to forbid is the specific
//     state-change-client constructor.
//   - The constructor `NewVirtualMachineRunCommandsClient` only appears
//     when code is actively building the state-change client. A grep
//     for the function name is a faithful proxy for "this file invokes
//     run commands".
//
// Running `go test ./services/orchestrator/internal/tools/...` will fail
// if the invariant is broken. CI already executes that command.
package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// azureRunCommandStateChangeConstructor is the SDK function name PR #27
// must never reference. PR #28 will introduce an explicit allowlist for
// its live-mode file.
const azureRunCommandStateChangeConstructor = "NewVirtualMachineRunCommandsClient"

// azureLiveRunCommandFile is the file PR #28 will introduce that
// legitimately references the state-change constructor. PR #27 doesn't
// have this file yet; this constant is here so PR #28's diff is small
// and obvious.
const azureLiveRunCommandFile = "live_azure_runcommand_client.go"

// TestNoAzureRunCommandStateChangeConstructorInToolsPackage scans every
// non-test .go file in the package and asserts none references
// `NewVirtualMachineRunCommandsClient`. Test files are skipped on
// purpose — fixtures may need different rules.
func TestNoAzureRunCommandStateChangeConstructorInToolsPackage(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}

	checked := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if name == azureLiveRunCommandFile {
			// PR #28's exception. Doesn't exist yet in PR #27.
			continue
		}

		body, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(body), azureRunCommandStateChangeConstructor) {
			t.Errorf("%s references %q — PR #27's structural safety guarantee is that no tools-package file invokes the Azure state-change run-command client. PR #28 will allow %s as the single exception.",
				name, azureRunCommandStateChangeConstructor, azureLiveRunCommandFile)
		}
		checked++
	}

	if checked == 0 {
		t.Fatal("scanned 0 files; the package layout may have changed in a way that defeats this test")
	}

	// Belt and suspenders: confirm we find the dry-run client file. If
	// it ever vanishes, the package layout changed and this test needs
	// review.
	found := false
	for _, e := range entries {
		if e.Name() == "azure_run_command_client.go" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("azure_run_command_client.go not found; PR #27's structural safety test is no longer guarding the right files")
	}
}

// TestAzureRunCommandClient_DocumentsTheGuarantee — defensive: the
// source file itself should carry the SAFETY comment near the top so a
// reader skimming the package understands the invariant. If someone
// removes the comment, they should at least notice this test failing.
func TestAzureRunCommandClient_DocumentsTheGuarantee(t *testing.T) {
	body, err := os.ReadFile(filepath.Clean("azure_run_command_client.go"))
	if err != nil {
		t.Fatalf("read azure_run_command_client.go: %v", err)
	}
	if !strings.Contains(string(body), "never calls the Azure SDK's") &&
		!strings.Contains(string(body), "never reaches the SDK") {
		t.Errorf("azure_run_command_client.go is missing its SAFETY comment about NOT calling the state-change SDK path; restore it before merging")
	}
}

// TestLiveAzureRunCommandClient_IsTheOnlyFileReferencingSDKConstructor —
// PR #28 / CONN-008 strengthens the structural safety guarantee. The
// "no constructor" half is enforced by
// TestNoAzureRunCommandStateChangeConstructorInToolsPackage above; THIS
// test is the positive half: live_azure_runcommand_client.go MUST
// reference the constructor. If it doesn't, PR #28's live mode is
// structurally unreachable — either the file was deleted without
// removing the live tool, or a refactor moved the constructor into a
// different file and the negative test would now mis-allowlist the
// wrong file.
//
// Both halves must pass for the safety invariant ("the state-change
// constructor is reachable from exactly ONE file in the package, with
// that file's name fixed") to hold.
func TestLiveAzureRunCommandClient_IsTheOnlyFileReferencingSDKConstructor(t *testing.T) {
	body, err := os.ReadFile(filepath.Clean(azureLiveRunCommandFile))
	if err != nil {
		t.Fatalf("read %s: %v — PR #28's live mode requires this file. If you intentionally removed live mode, also remove this test and the allowlist constant.", azureLiveRunCommandFile, err)
	}
	if !strings.Contains(string(body), azureRunCommandStateChangeConstructor) {
		t.Errorf("%s must reference %q — PR #28's live mode is unreachable without it. If you intentionally removed live mode, also remove this test.",
			azureLiveRunCommandFile, azureRunCommandStateChangeConstructor)
	}
}
