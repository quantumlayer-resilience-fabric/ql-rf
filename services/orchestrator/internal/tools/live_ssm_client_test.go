// PR #21 / CONN-003 — unit tests for the live SSM client + whitelist helpers.
//
// Pure-Go tests against the mock client and the whitelist parsing helpers.
// The real client is exercised only indirectly (constructor surface) — its
// SDK calls are out of scope for unit tests; CI's live-mode boot path
// uses the mock client (RF_CONNECTORS_AWS_LIVE_PATCH_CLIENT_MODE=mock) so
// no real AWS credentials or network calls happen in CI either.

package tools

import (
	"context"
	"strings"
	"testing"

	pkgconfig "github.com/quantumlayerhq/ql-rf/pkg/config"
)

// awsConfigForLiveTest returns a minimal AWSConfig — the constructor we
// test only reaches the whitelist check before erroring out, so the
// region and credentials fields are never used.
func awsConfigForLiveTest() pkgconfig.AWSConfig {
	return pkgconfig.AWSConfig{Region: "us-east-1"}
}

// TestParseWhitelistCSV — empty, single, multiple, whitespace handling.
func TestParseWhitelistCSV(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{",,,", nil},
		{"i-001", []string{"i-001"}},
		{" i-001 , i-002 ", []string{"i-001", "i-002"}},
		{"i-001,,i-002", []string{"i-001", "i-002"}},
	}
	for _, c := range cases {
		got := parseWhitelistCSV(c.in)
		if !stringSliceEq(got, c.want) {
			t.Errorf("parseWhitelistCSV(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestIsInstanceWhitelisted — basic membership + edge cases.
func TestIsInstanceWhitelisted(t *testing.T) {
	wl := []string{"i-001", "i-002"}
	if !isInstanceWhitelisted("i-001", wl) {
		t.Error("expected i-001 to be whitelisted")
	}
	if isInstanceWhitelisted("i-999", wl) {
		t.Error("expected i-999 to NOT be whitelisted")
	}
	if isInstanceWhitelisted("i-001", nil) {
		t.Error("nil whitelist should reject everything")
	}
}

// TestRequireAllWhitelisted_AcceptsAllOnList — happy path.
func TestRequireAllWhitelisted_AcceptsAllOnList(t *testing.T) {
	wl := []string{"i-001", "i-002", "i-003"}
	if err := requireAllWhitelisted([]string{"i-001", "i-003"}, wl); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// TestRequireAllWhitelisted_RejectsAnyOffList — single off-list ID rejects
// the entire batch. Partial sends would create audit-log entries that are
// misleading about scope, so all-or-nothing is the safer default.
func TestRequireAllWhitelisted_RejectsAnyOffList(t *testing.T) {
	wl := []string{"i-001"}
	err := requireAllWhitelisted([]string{"i-001", "i-999"}, wl)
	if err == nil {
		t.Fatal("expected rejection, got nil")
	}
	if !strings.Contains(err.Error(), "i-999") {
		t.Errorf("error should name the offending instance; got %q", err.Error())
	}
}

// TestRequireAllWhitelisted_RejectsEmpty — empty target list is also an
// error. Zero-target SendCommand calls are meaningless and probably
// indicate a programmer error upstream.
func TestRequireAllWhitelisted_RejectsEmpty(t *testing.T) {
	wl := []string{"i-001"}
	err := requireAllWhitelisted(nil, wl)
	if err == nil {
		t.Fatal("expected rejection for empty targets, got nil")
	}
}

// TestMockLiveSSMClient_ValidatesAndReturnsCommandID — the mock client's
// contract: validate whitelist, return deterministic-shape cmd-mock-<hex>
// without touching the SDK.
func TestMockLiveSSMClient_ValidatesAndReturnsCommandID(t *testing.T) {
	wl := []string{"i-001"}
	c := NewMockLiveSSMClient(wl)
	plan := &SSMCommandPlan{
		DocumentName: "AWS-RunPatchBaseline",
		InstanceIDs:  []string{"i-001"},
		Region:       "us-east-1",
	}
	id, err := c.SendCommand(context.Background(), plan)
	if err != nil {
		t.Fatalf("SendCommand: %v", err)
	}
	if !strings.HasPrefix(id, "cmd-mock-") {
		t.Errorf("command id = %q, want cmd-mock-* prefix", id)
	}
}

// TestMockLiveSSMClient_RejectsNonWhitelistedInstance — the mock enforces
// the whitelist just like the real client. Tests that exercise the live
// tool's Execute against the mock therefore catch whitelist regressions
// without needing a real client.
func TestMockLiveSSMClient_RejectsNonWhitelistedInstance(t *testing.T) {
	c := NewMockLiveSSMClient([]string{"i-001"})
	plan := &SSMCommandPlan{
		InstanceIDs: []string{"i-999"},
	}
	if _, err := c.SendCommand(context.Background(), plan); err == nil {
		t.Fatal("expected whitelist rejection, got nil")
	}
}

// TestMockLiveSSMClient_RejectsNilPlan — defensive against a programmer
// error in tool wiring (live tool's Execute always builds a plan first,
// but the client shouldn't trust callers).
func TestMockLiveSSMClient_RejectsNilPlan(t *testing.T) {
	c := NewMockLiveSSMClient([]string{"i-001"})
	if _, err := c.SendCommand(context.Background(), nil); err == nil {
		t.Fatal("expected nil-plan rejection, got nil")
	}
}

// TestNewLiveSSMClient_RefusesEmptyWhitelist — the boot path passes the
// whitelist after validating it's non-empty, but the constructor adds a
// belt-and-braces refusal so a programmer error upstream still fails
// loudly rather than constructing an unusable client.
func TestNewLiveSSMClient_RefusesEmptyWhitelist(t *testing.T) {
	// Use a minimal config; the function never reaches LoadDefaultConfig
	// when the whitelist is empty.
	_, err := NewLiveSSMClient(context.Background(), awsConfigForLiveTest(), nil, testLoggerForSSM())
	if err == nil {
		t.Fatal("expected refusal for empty whitelist, got nil")
	}
	if !strings.Contains(err.Error(), "whitelist") {
		t.Errorf("error should mention whitelist; got %q", err.Error())
	}
}

// stringSliceEq compares two string slices for ordered equality (nil and
// empty slices are treated as equivalent).
func stringSliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
