// PR #20 / CONN-002 — structural safety test.
//
// Mechanically verifies that no file in the tools package imports
// `aws-sdk-go-v2/service/ssm`. Importing that package would expose the
// `SendCommand` method and undo PR #20's "dry-run only" guarantee. PR #21
// will introduce a dedicated file that DOES import the SDK; when that
// happens, this test must be updated to whitelist that single file by name.
//
// Why a test instead of a golangci-lint depguard rule:
//   - depguard is currently disabled project-wide in .golangci.yml.
//   - Enabling it would force every other rule's scoping decisions on us.
//   - A plain Go test is greppable, version-controlled, and runs in the
//     existing CI Go Test job — no new tooling.
//
// Running `go test ./services/orchestrator/internal/tools/...` will fail
// if the invariant is broken. CI already executes that command.
package tools

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// forbiddenSDKImport is the import path PR #20 must never use. PR #21 will
// introduce an explicit allowlist for its live-mode file.
const forbiddenSDKImport = "github.com/aws/aws-sdk-go-v2/service/ssm"

// pr21LiveClientFile is the file PR #21 will introduce that legitimately
// imports the forbidden package. PR #20 doesn't have this file yet; this
// constant is here so PR #21's diff is small and obvious.
const pr21LiveClientFile = "live_ssm_client.go"

// TestNoSSMSDKImportInToolsPackage parses every non-test .go file in this
// package and asserts none imports the SSM SDK. Test files are skipped on
// purpose — fixtures may need different rules.
func TestNoSSMSDKImportInToolsPackage(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}

	fset := token.NewFileSet()
	checked := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if name == pr21LiveClientFile {
			// PR #21's exception. Doesn't exist yet in PR #20.
			continue
		}

		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if path == forbiddenSDKImport {
				t.Errorf("%s imports %q — PR #20's structural safety guarantee is that no tools-package file imports the SSM SDK. PR #21 will allow %s as the single exception.",
					name, forbiddenSDKImport, pr21LiveClientFile)
			}
		}
		checked++
	}

	if checked == 0 {
		t.Fatal("scanned 0 files; the package layout may have changed in a way that defeats this test")
	}

	// Belt and suspenders: also confirm we actually find the dry-run client.
	// If ssm_client.go ever vanishes, the package layout changed and this
	// test needs review.
	foundDryRunClient := false
	for _, e := range entries {
		if e.Name() == "ssm_client.go" {
			foundDryRunClient = true
			break
		}
	}
	if !foundDryRunClient {
		t.Fatal("ssm_client.go not found; PR #20's structural safety test is no longer guarding the right files")
	}
}

// TestSSMClientFile_DocumentsTheGuarantee — defensive: the source file
// itself should carry a SAFETY comment near the top so a reader skimming
// the package understands the invariant. If someone removes the comment,
// they should at least notice this test failing.
func TestSSMClientFile_DocumentsTheGuarantee(t *testing.T) {
	body, err := os.ReadFile(filepath.Clean("ssm_client.go"))
	if err != nil {
		t.Fatalf("read ssm_client.go: %v", err)
	}
	if !strings.Contains(string(body), "does NOT import") {
		t.Errorf("ssm_client.go is missing its SAFETY comment about NOT importing the SDK; restore it before merging")
	}
}

// TestLiveSSMClient_IsTheOnlyFileImportingSDK — PR #21 / CONN-003 strengthens
// the structural safety guarantee. The "no SDK" half is enforced by
// TestNoSSMSDKImportInToolsPackage above; THIS test is the positive half:
// live_ssm_client.go MUST import the SDK. If it doesn't, PR #21's live
// mode is structurally unreachable — either the file was deleted without
// removing the live tool, or a refactor moved the SDK into a different
// file and the negative test would now mis-allowlist the wrong file.
//
// Both halves must pass for the safety invariant ("SDK is reachable from
// exactly ONE file in the package, with that file's name fixed") to hold.
func TestLiveSSMClient_IsTheOnlyFileImportingSDK(t *testing.T) {
	// Use the same AST-import parsing TestNoSSMSDKImportInToolsPackage
	// uses so comments mentioning the SDK path don't false-positive.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, pr21LiveClientFile, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v — PR #21's live mode requires this file. If you intentionally removed live mode, also remove this test and the allowlist constant.", pr21LiveClientFile, err)
	}
	hasImport := false
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) == forbiddenSDKImport {
			hasImport = true
			break
		}
	}
	if !hasImport {
		t.Errorf("%s must import %q — PR #21's live mode is unreachable without it. If you intentionally removed live mode, also remove this test.",
			pr21LiveClientFile, forbiddenSDKImport)
	}
}
