// PR #34 / CONN-013 — structural safety test for vSphere guest-ops.
//
// Mechanically verifies that no file in the tools package references
// the govmomi state-change method name. PR #35 will introduce a
// dedicated file (`live_vsphere_guest_ops_client.go`) that DOES
// reference it; when that PR lands, this test will be updated to
// whitelist that file by name.
//
// Why a name-based check: govmomi is in a different surface than the
// read-only path (PR #33 uses container view + mo types; the
// state-change call is `ProcessManager.StartProgramInGuest`). Match
// the method name verbatim.
package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// vsphereGuestOpsStateChangeMethod is the govmomi method-call signature
// that PR #34 must never reference. govmomi exposes
// `ProcessManager.StartProgram(ctx, auth, spec)` as the wrapper around
// the VMODL `StartProgramInGuest` operation; the leading dot disambiguates
// it from accidentally matching variable / function names that happen to
// contain "StartProgram".
//
// PR #35 introduces `live_vsphere_guest_ops_client.go` as the single
// allowlisted caller.
const vsphereGuestOpsStateChangeMethod = ".StartProgram("

// vsphereLiveGuestOpsFile is the file PR #35 will introduce.
const vsphereLiveGuestOpsFile = "live_vsphere_guest_ops_client.go"

// TestNoVSphereGuestOpsStateChangeMethodInToolsPackage scans every
// non-test .go file in the package and asserts none references the
// state-change method name.
func TestNoVSphereGuestOpsStateChangeMethodInToolsPackage(t *testing.T) {
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
		if name == vsphereLiveGuestOpsFile {
			// PR #35's exception. Doesn't exist yet in PR #34.
			continue
		}

		body, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(body), vsphereGuestOpsStateChangeMethod) {
			t.Errorf("%s references %q — PR #34's structural safety guarantee is that no tools-package file invokes the vSphere state-change guest-ops method. PR #35 will allow %s as the single exception.",
				name, vsphereGuestOpsStateChangeMethod, vsphereLiveGuestOpsFile)
		}
		checked++
	}

	if checked == 0 {
		t.Fatal("scanned 0 files; the package layout may have changed in a way that defeats this test")
	}

	found := false
	for _, e := range entries {
		if e.Name() == "vsphere_guest_ops_client.go" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("vsphere_guest_ops_client.go not found; PR #34's structural safety test is no longer guarding the right files")
	}
}

// TestVSphereGuestOpsClient_DocumentsTheGuarantee — the source file
// should carry the SAFETY comment near the top.
func TestVSphereGuestOpsClient_DocumentsTheGuarantee(t *testing.T) {
	body, err := os.ReadFile(filepath.Clean("vsphere_guest_ops_client.go"))
	if err != nil {
		t.Fatalf("read vsphere_guest_ops_client.go: %v", err)
	}
	if !strings.Contains(string(body), "never calls the state-change") &&
		!strings.Contains(string(body), "never reaches the state-change") {
		t.Errorf("vsphere_guest_ops_client.go is missing its SAFETY comment about NOT calling the state-change SDK path; restore it before merging")
	}
}

// TestLiveVSphereGuestOpsClient_IsTheOnlyFileReferencingSDKMethod —
// PR #35's positive complement: the live file MUST reference
// `.StartProgram(`. Otherwise live mode is unreachable and the
// allowlist exception is mis-pointed.
func TestLiveVSphereGuestOpsClient_IsTheOnlyFileReferencingSDKMethod(t *testing.T) {
	body, err := os.ReadFile(filepath.Clean(vsphereLiveGuestOpsFile))
	if err != nil {
		t.Fatalf("read %s: %v — PR #35's live mode requires this file. If you intentionally removed live mode, also remove this test.", vsphereLiveGuestOpsFile, err)
	}
	if !strings.Contains(string(body), vsphereGuestOpsStateChangeMethod) {
		t.Errorf("%s must reference %q — PR #35's live mode is unreachable without it.",
			vsphereLiveGuestOpsFile, vsphereGuestOpsStateChangeMethod)
	}
}
