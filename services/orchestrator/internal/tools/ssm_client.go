// PR #20 / CONN-002 — SSM client for the state-change patch tool (dry-run).
//
// SAFETY: this file deliberately does NOT import
// `github.com/aws/aws-sdk-go-v2/service/ssm`. The state-change patch tool
// in this PR is dry-run only — it constructs the command plan as a plain
// Go struct and never reaches the SSM SDK. A depguard rule in .golangci.yml
// forbids the import package-wide in this PR's code.
//
// PR #21 will introduce a separate file (e.g. `live_ssm_client.go`) that
// imports the SDK and exposes a SendCommand method gated by env opt-in +
// per-instance whitelist + two-approver workflow. That import lives in its
// own file so the no-SDK invariant remains greppable and the depguard rule
// adds an explicit exception for that single file.
package tools

import (
	"context"
	"fmt"
	"regexp"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// patchOpScan / patchOpInstall — AWS-RunPatchBaseline document operations.
// Hoisted to constants because the literal "Scan" appears in defaults,
// validation, and tests; goconst flags it otherwise.
const (
	patchOpScan    = "Scan"
	patchOpInstall = "Install"
)

// SSMClient builds (does not send) AWS SSM patch commands. The interface is
// deliberately narrow — one method, one read of the SDK's shape, zero
// network calls. PR #21's live client implementation will satisfy the same
// interface plus an extra Send method behind explicit safety gates.
type SSMClient interface {
	BuildPatchCommand(ctx context.Context, req PatchCommandRequest) (*SSMCommandPlan, error)
}

// PatchCommandRequest is the typed input to BuildPatchCommand. Keeping it as
// a struct (not loose params) means the validation lives at the boundary,
// not inside the tool's params-coercion code.
type PatchCommandRequest struct {
	Region      string
	InstanceIDs []string
	Operation   string // "Scan" or "Install"
}

// SSMCommandPlan mirrors the relevant fields of `ssm.SendCommandInput` but
// is our own struct — never the SDK type. The audit log records this
// verbatim. DryRun and RealChanges are always true / false respectively
// in PR #20; PR #21's live mode will flip them per-invocation.
type SSMCommandPlan struct {
	DocumentName    string              `json:"document_name"`
	DocumentVersion string              `json:"document_version"`
	Parameters      map[string][]string `json:"parameters"`
	InstanceIDs     []string            `json:"instance_ids"`
	Region          string              `json:"region"`
	Comment         string              `json:"comment"`
	TimeoutSeconds  int32               `json:"timeout_seconds"`
	DryRun          bool                `json:"dry_run"`
	RealChanges     bool                `json:"real_changes"`
}

// realSSMClient validates and constructs patch command plans. No SDK
// imports. No network calls. Just struct construction + basic validation.
//
// The "real" in the name is relative: this client builds what real SSM
// SendCommand would receive, but never sends it. The mock client below is
// a more aggressive variant for CI; both are dry-run.
type realSSMClient struct {
	log *logger.Logger
}

// NewRealSSMClient constructs the validation-only SSM client. Always
// succeeds — there's nothing to fail at boot, since no network calls are
// made.
func NewRealSSMClient(log *logger.Logger) SSMClient {
	return &realSSMClient{log: log.WithComponent("ssm-tools")}
}

// BuildPatchCommand validates the request and constructs the plan. Returns
// errors for malformed instance IDs or unsupported operations. The plan
// always carries `DryRun: true` and `RealChanges: false` — PR #21 is what
// flips those.
func (c *realSSMClient) BuildPatchCommand(_ context.Context, req PatchCommandRequest) (*SSMCommandPlan, error) {
	if err := validateInstanceIDs(req.InstanceIDs); err != nil {
		return nil, err
	}

	op := req.Operation
	if op == "" {
		op = patchOpScan
	}
	if op != patchOpScan && op != patchOpInstall {
		return nil, fmt.Errorf("operation must be %q or %q, got %q", patchOpScan, patchOpInstall, op)
	}

	region := req.Region
	if region == "" {
		region = defaultAWSRegion
	}

	return &SSMCommandPlan{
		DocumentName:    "AWS-RunPatchBaseline",
		DocumentVersion: "$LATEST",
		Parameters: map[string][]string{
			"Operation":    {op},
			"RebootOption": {"RebootIfNeeded"},
		},
		InstanceIDs:    append([]string(nil), req.InstanceIDs...),
		Region:         region,
		Comment:        "QL-RF dry-run (PR #20): constructed without invocation.",
		TimeoutSeconds: 3600,
		DryRun:         true,
		RealChanges:    false,
	}, nil
}

// mockSSMClient returns a deterministic plan tagged with i-mock-* instance
// IDs so the mock origin is obvious in the audit log. Used by unit tests
// and by CI (where `RF_CONNECTORS_AWS_FALLBACK_TO_MOCK=true` from PR #19
// gates AWS credentials anyway).
//
// The mock client is intentionally MORE strict than the real client about
// what it accepts — it ignores caller-provided instance IDs and always
// returns the same two fakes. This guarantees CI tests are deterministic
// regardless of input.
type mockSSMClient struct{}

// NewMockSSMClient constructs the deterministic fixture client.
func NewMockSSMClient() SSMClient {
	return &mockSSMClient{}
}

// BuildPatchCommand returns a fixed plan with two i-mock- instance IDs,
// regardless of input. Used by unit tests and CI.
func (m *mockSSMClient) BuildPatchCommand(_ context.Context, req PatchCommandRequest) (*SSMCommandPlan, error) {
	op := req.Operation
	if op == "" {
		op = patchOpScan
	}
	region := req.Region
	if region == "" {
		region = defaultAWSRegion
	}
	return &SSMCommandPlan{
		DocumentName:    "AWS-RunPatchBaseline",
		DocumentVersion: "$LATEST",
		Parameters: map[string][]string{
			"Operation":    {op},
			"RebootOption": {"RebootIfNeeded"},
		},
		InstanceIDs:    []string{"i-mock-0001", "i-mock-0002"},
		Region:         region,
		Comment:        "QL-RF dry-run mock (PR #20): no real instances.",
		TimeoutSeconds: 3600,
		DryRun:         true,
		RealChanges:    false,
	}, nil
}

// instanceIDPattern matches AWS EC2 instance IDs in both legacy (8 hex
// chars) and modern (17 hex chars) formats. Strict-by-default — junk
// values stop here before they pollute the audit log with shapes a real
// SendCommand call would reject anyway.
var instanceIDPattern = regexp.MustCompile(`^i-[a-f0-9]{8}([a-f0-9]{9})?$`)

// validateInstanceIDs is a pure helper that returns a descriptive error if
// any ID in the list doesn't look like an EC2 instance ID. Empty list also
// errors — at least one target instance is required.
func validateInstanceIDs(ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("at least one instance_id is required")
	}
	for _, id := range ids {
		if !instanceIDPattern.MatchString(id) {
			return fmt.Errorf("invalid instance_id %q: must match i-[hex]{8,17}", id)
		}
	}
	return nil
}
