// PR #39 / CONN-016 — structural safety test for K8s server-side-apply.
//
// Mechanically verifies that no file in the tools package references
// the K8s state-change SDK call signature. PR #40 will introduce
// `live_k8s_apply_client.go` as the SOLE caller of the state-change
// method — at that point this test will be updated with a positive
// complement that asserts the live file IS using the SDK.
//
// Why function-call-token matching: client-go is already legitimately
// imported by k8s_client.go (PR #38's read-only path). An import-path
// forbidding test (the PR #20 SSM pattern) wouldn't work — too many
// client-go subpackages are shared. Instead this matches the specific
// signature unique to server-side apply.
package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// k8sApplyStateChangeToken is the unique SDK signature that PR #39 must
// never reference. client-go server-side apply uses
// `metav1.ApplyOptions{...}` as the options struct and the operation
// goes through either `clientset.<Group>().Apply(ctx, applyConfig, opts)`
// or `Patch(ctx, name, types.ApplyPatchType, ...)`. The ApplyOptions
// struct literal is uniquely used by apply paths and is the cleanest
// audit-by-grep token.
//
// PR #40 introduces `live_k8s_apply_client.go` as the single allowlisted caller.
const k8sApplyStateChangeToken = "metav1.ApplyOptions{"

// k8sLiveApplyFile is the file PR #40 will introduce.
const k8sLiveApplyFile = "live_k8s_apply_client.go"

// TestNoK8sApplyStateChangeMethodInToolsPackage scans every non-test
// .go file in the package and asserts none references the state-change
// signature.
func TestNoK8sApplyStateChangeMethodInToolsPackage(t *testing.T) {
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
		if name == k8sLiveApplyFile {
			// PR #40's exception. Doesn't exist yet in PR #39.
			continue
		}

		body, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(body), k8sApplyStateChangeToken) {
			t.Errorf("%s references %q — PR #39's structural safety guarantee is that no tools-package file invokes the K8s state-change apply path. PR #40 will allow %s as the single exception.",
				name, k8sApplyStateChangeToken, k8sLiveApplyFile)
		}
		checked++
	}

	if checked == 0 {
		t.Fatal("scanned 0 files; the package layout may have changed in a way that defeats this test")
	}

	found := false
	for _, e := range entries {
		if e.Name() == "k8s_apply_client.go" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("k8s_apply_client.go not found; PR #39's structural safety test is no longer guarding the right files")
	}
}

// TestK8sApplyClient_DocumentsTheGuarantee — the source file should
// carry the SAFETY comment near the top.
func TestK8sApplyClient_DocumentsTheGuarantee(t *testing.T) {
	body, err := os.ReadFile(filepath.Clean("k8s_apply_client.go"))
	if err != nil {
		t.Fatalf("read k8s_apply_client.go: %v", err)
	}
	if !strings.Contains(string(body), "never calls the state-change") &&
		!strings.Contains(string(body), "never reaches the state-change") {
		t.Errorf("k8s_apply_client.go is missing its SAFETY comment about NOT calling the state-change SDK path; restore it before merging")
	}
}
