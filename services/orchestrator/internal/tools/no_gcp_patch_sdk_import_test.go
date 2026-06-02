// PR #30 / CONN-010 — structural safety test for GCP OS Config patches.
//
// Mechanically verifies that no file in the tools package references
// the GCP OS Config state-change client constructor name. Calling it
// would expose `ExecutePatchJob`, which actually runs patches on
// production VMs. PR #31 introduces a dedicated file
// (`live_gcp_patch_client.go`) that DOES reference the constructor;
// when that PR lands, this test will be updated to whitelist that
// single file by name (same shape as PR #20/#21 and PR #27/#28).
//
// Why a name-based check: the GCP OS Config SDK is in a different
// package from the read-only Compute SDK (PR #29 uses compute/apiv1;
// patches use osconfig/apiv1). The narrow state-change constructor
// name is the most specific signal that a file invokes patches.
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

// gcpPatchStateChangeConstructor is the SDK function name PR #30 must
// never reference. PR #31 will introduce an explicit allowlist for its
// live-mode file.
//
// The osconfig SDK has multiple clients (NewOsConfigZonalClient for
// reads, plain NewClient for the global ExecutePatchJob). The latter is
// the state-change entry point we forbid.
const gcpPatchStateChangeConstructor = "osconfig.NewClient"

// gcpLivePatchFile is the file PR #31 will introduce that legitimately
// references the constructor. PR #30 doesn't have this file yet.
const gcpLivePatchFile = "live_gcp_patch_client.go"

// TestNoGCPPatchStateChangeConstructorInToolsPackage scans every
// non-test .go file in the package and asserts none references the
// state-change constructor name.
func TestNoGCPPatchStateChangeConstructorInToolsPackage(t *testing.T) {
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
		if name == gcpLivePatchFile {
			// PR #31's exception. Doesn't exist yet in PR #30.
			continue
		}

		body, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(body), gcpPatchStateChangeConstructor) {
			t.Errorf("%s references %q — PR #30's structural safety guarantee is that no tools-package file invokes the GCP OS Config state-change client. PR #31 will allow %s as the single exception.",
				name, gcpPatchStateChangeConstructor, gcpLivePatchFile)
		}
		checked++
	}

	if checked == 0 {
		t.Fatal("scanned 0 files; the package layout may have changed in a way that defeats this test")
	}

	// Belt and suspenders: confirm we find the dry-run client file.
	found := false
	for _, e := range entries {
		if e.Name() == "gcp_patch_client.go" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("gcp_patch_client.go not found; PR #30's structural safety test is no longer guarding the right files")
	}
}

// TestGCPPatchClient_DocumentsTheGuarantee — defensive: the source file
// should carry the SAFETY comment near the top.
func TestGCPPatchClient_DocumentsTheGuarantee(t *testing.T) {
	body, err := os.ReadFile(filepath.Clean("gcp_patch_client.go"))
	if err != nil {
		t.Fatalf("read gcp_patch_client.go: %v", err)
	}
	if !strings.Contains(string(body), "never calls the state-change") &&
		!strings.Contains(string(body), "never reaches the state-change") {
		t.Errorf("gcp_patch_client.go is missing its SAFETY comment about NOT calling the state-change SDK path; restore it before merging")
	}
}

// TestLiveGCPPatchClient_IsTheOnlyFileReferencingSDKConstructor — PR #31
// strengthens the structural safety guarantee. The "no constructor"
// half is enforced by TestNoGCPPatchStateChangeConstructorInToolsPackage;
// THIS test is the positive half. Both must pass for the invariant
// ("constructor reachable from exactly ONE named file") to hold.
func TestLiveGCPPatchClient_IsTheOnlyFileReferencingSDKConstructor(t *testing.T) {
	body, err := os.ReadFile(filepath.Clean(gcpLivePatchFile))
	if err != nil {
		t.Fatalf("read %s: %v — PR #31's live mode requires this file. If you intentionally removed live mode, also remove this test.", gcpLivePatchFile, err)
	}
	if !strings.Contains(string(body), gcpPatchStateChangeConstructor) {
		t.Errorf("%s must reference %q — PR #31's live mode is unreachable without it.",
			gcpLivePatchFile, gcpPatchStateChangeConstructor)
	}
}
