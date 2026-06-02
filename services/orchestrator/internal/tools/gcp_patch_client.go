// PR #30 / CONN-010 — GCP OS Config patch client (DRY-RUN ONLY).
//
// SAFETY (READ THIS BEFORE EDITING):
// This file is the GCP equivalent of PR #20's ssm_client.go and PR #27's
// azure_run_command_client.go. It builds patch-job plans as plain Go
// structs and never calls the state-change SDK path. PR #31 will
// introduce `live_gcp_patch_client.go` as the SOLE caller of
// the state-change SDK constructor (see no_gcp_patch_sdk_import_test.go
// for the exact name) — the structural test in
// no_gcp_patch_sdk_import_test.go enforces this by name. The same SDK
// isolation discipline that made the SSM and Azure live paths
// mechanically safe to review applies here.
//
// The structural test for GCP follows the Azure pattern: function-name
// matching (the package-level `NewClient` constructor from
// `cloud.google.com/go/osconfig/apiv1`) rather than import-path
// forbidding, because the read-only PR #29 path uses a different SDK
// package (`compute/apiv1`) so there's no shared import to police.
//
// Live mode (real ExecutePatchJob) lands as PR #31 with stronger gates
// (env opt-in, per-instance whitelist, two-approver workflow).
package tools

import (
	"context"
	"fmt"
	"regexp"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// GCPPatchClient builds (does not send) GCP OS Config patch jobs. The
// interface is narrow — one method, zero network calls.
type GCPPatchClient interface {
	BuildPatchJobPlan(ctx context.Context, req GCPPatchJobRequest) (*GCPPatchJobPlan, error)
}

// GCPPatchJobRequest is the typed input to BuildPatchJobPlan. Keeping
// it as a struct (not loose params) means validation lives at the
// boundary, not inside the tool's params-coercion code.
type GCPPatchJobRequest struct {
	ProjectID string
	// Zone is the GCP zone (e.g. "us-central1-a") containing the
	// targeted instances. Patch jobs target a zone+filter pair.
	Zone string
	// InstanceFilter — GCE label filter (e.g. "env=prod"). The patch
	// job runs against every instance in the zone matching the filter.
	InstanceFilter string
	// RebootConfig — one of "DEFAULT", "ALWAYS", "NEVER". Matches the
	// SDK's PatchConfig.RebootConfig enum.
	RebootConfig string
	// DisplayName — operator-friendly name for the job. Lands in the
	// SDK call and audit row description.
	DisplayName string
}

// GCPPatchJobPlan mirrors the relevant fields of
// `osconfigpb.ExecutePatchJobRequest` but is OUR own struct — never the
// SDK type. The audit log records this verbatim. DryRun and RealChanges
// are always true / false respectively in PR #30.
type GCPPatchJobPlan struct {
	ProjectID      string `json:"project_id"`
	Zone           string `json:"zone"`
	InstanceFilter string `json:"instance_filter"`
	RebootConfig   string `json:"reboot_config"`
	DisplayName    string `json:"display_name,omitempty"`
	// DurationSeconds — how long the patch job can run before timing
	// out. SDK default is 1 hour; we pin 2 hours to match the SSM /
	// Azure plans.
	DurationSeconds int32 `json:"duration_seconds"`
	// Comment — operator-friendly note that lands in the SDK call's
	// description and our audit row.
	Comment     string `json:"comment,omitempty"`
	DryRun      bool   `json:"dry_run"`
	RealChanges bool   `json:"real_changes"`
}

// Supported GCP RebootConfig values. Hoisted to constants so the
// validator + the tool's parameter enum + future docs all reference the
// same list. Matches osconfigpb.PatchConfig_RebootConfig values.
const (
	gcpRebootDefault = "DEFAULT"
	gcpRebootAlways  = "ALWAYS"
	gcpRebootNever   = "NEVER"
)

// realGCPPatchClient validates the request and constructs the plan. No
// SDK imports. No network calls.
type realGCPPatchClient struct {
	log *logger.Logger
}

// NewRealGCPPatchClient constructs the validation-only client. Always
// succeeds — there's nothing to fail at boot, since no network calls
// are made.
func NewRealGCPPatchClient(log *logger.Logger) GCPPatchClient {
	return &realGCPPatchClient{log: log.WithComponent("gcp-patch")}
}

// BuildPatchJobPlan validates the request and constructs the plan.
// Returns errors for malformed project/zone names or unsupported reboot
// configs. The plan always carries DryRun:true and RealChanges:false —
// PR #31 is what flips those.
func (c *realGCPPatchClient) BuildPatchJobPlan(_ context.Context, req GCPPatchJobRequest) (*GCPPatchJobPlan, error) {
	if err := validateGCPProjectID(req.ProjectID); err != nil {
		return nil, err
	}
	if err := validateGCPZone(req.Zone); err != nil {
		return nil, err
	}
	if req.InstanceFilter == "" {
		return nil, fmt.Errorf("instance_filter is required (e.g. \"labels.env=prod\")")
	}

	reboot := req.RebootConfig
	if reboot == "" {
		reboot = gcpRebootDefault
	}
	if reboot != gcpRebootDefault && reboot != gcpRebootAlways && reboot != gcpRebootNever {
		return nil, fmt.Errorf("reboot_config must be %q, %q, or %q; got %q",
			gcpRebootDefault, gcpRebootAlways, gcpRebootNever, reboot)
	}

	return &GCPPatchJobPlan{
		ProjectID:       req.ProjectID,
		Zone:            req.Zone,
		InstanceFilter:  req.InstanceFilter,
		RebootConfig:    reboot,
		DisplayName:     req.DisplayName,
		DurationSeconds: 7200,
		Comment:         "QL-RF dry-run (PR #30): constructed without invocation.",
		DryRun:          true,
		RealChanges:     false,
	}, nil
}

// mockGCPPatchClient returns a deterministic plan tagged with a mock
// project. Used by unit tests and CI.
type mockGCPPatchClient struct{}

// NewMockGCPPatchClient constructs the deterministic fixture client.
func NewMockGCPPatchClient() GCPPatchClient {
	return &mockGCPPatchClient{}
}

// BuildPatchJobPlan returns a fixed plan with the mock fixture
// regardless of input.
func (m *mockGCPPatchClient) BuildPatchJobPlan(_ context.Context, req GCPPatchJobRequest) (*GCPPatchJobPlan, error) {
	reboot := req.RebootConfig
	if reboot == "" {
		reboot = gcpRebootDefault
	}
	return &GCPPatchJobPlan{
		ProjectID:       "ql-rf-mock-project",
		Zone:            "us-central1-a",
		InstanceFilter:  "labels.env=mock-prod",
		RebootConfig:    reboot,
		DisplayName:     req.DisplayName,
		DurationSeconds: 7200,
		Comment:         "QL-RF dry-run mock (PR #30): no real GCE instances.",
		DryRun:          true,
		RealChanges:     false,
	}, nil
}

// GCP project naming rules: 6-30 chars, lowercase letters/digits/hyphens,
// starts with a letter. The SDK rejects anything else before the live
// call too; we mirror the check at the boundary.
var (
	gcpProjectIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)
	// Zone format: "<region>-<letter>", region matches "[a-z]+-[a-z0-9]+".
	gcpZonePattern = regexp.MustCompile(`^[a-z]+-[a-z0-9]+-[a-z]$`)
)

// validateGCPProjectID returns a descriptive error if the project ID
// doesn't match GCP's naming rules.
func validateGCPProjectID(id string) error {
	if id == "" {
		return fmt.Errorf("project_id is required")
	}
	if !gcpProjectIDPattern.MatchString(id) {
		return fmt.Errorf("invalid project_id %q: must match GCP project naming rules", id)
	}
	return nil
}

// validateGCPZone returns a descriptive error if the zone doesn't match
// GCP's zone naming rules.
func validateGCPZone(zone string) error {
	if zone == "" {
		return fmt.Errorf("zone is required (e.g. \"us-central1-a\")")
	}
	if !gcpZonePattern.MatchString(zone) {
		return fmt.Errorf("invalid zone %q: must match <region>-<letter>", zone)
	}
	return nil
}
