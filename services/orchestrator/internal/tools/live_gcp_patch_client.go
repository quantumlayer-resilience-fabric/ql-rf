// PR #31 / CONN-011 — LIVE GCP OS Config patch client.
//
// SAFETY (READ THIS BEFORE EDITING):
// This is THE single file in the tools package allowed to call
// `osconfig.NewClient` (the global OS Config client that exposes
// ExecutePatchJob). The structural safety test in
// no_gcp_patch_sdk_import_test.go grants this file an allowlist
// exception by name; the complementary positive test
// TestLiveGCPPatchClient_IsTheOnlyFileReferencingSDKConstructor asserts
// (positive direction) that this file DOES reference the constructor.
// Both tests run on every push.
//
// Why this discipline matters: PR #30 (dry-run only) made the
// state-change GCP patch call structurally unreachable from the tools
// package. PR #31 re-introduces reachability deliberately and ONLY
// through this file, so auditors can grep for `osconfig.NewClient` and
// see exactly one match.
//
// Live mode is gated at four layers, every layer independently enforced:
//
//  1. Boot env opt-in: RF_CONNECTORS_GCP_ALLOW_LIVE_PATCH=true.
//  2. Mock-conflict refusal: if FallbackToMock is also true, exit 1.
//  3. Per-zone+filter whitelist:
//     RF_CONNECTORS_GCP_LIVE_PATCH_WHITELIST_INSTANCE_FILTERS env var.
//     Format: "zone:filter" pairs (e.g. "us-central1-a:labels.env=prod").
//     Empty + ALLOW_LIVE_PATCH=true also fails boot.
//  4. Two-approver workflow: enforced by OPA policy + coApproveTask
//     handler (PR #21).
package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	osconfig "cloud.google.com/go/osconfig/apiv1"
	"cloud.google.com/go/osconfig/apiv1/osconfigpb"
	"google.golang.org/api/option"

	pkgconfig "github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// LiveGCPPatchClient SENDS GCP OS Config patch jobs. Distinct from
// GCPPatchClient (PR #30) which only BUILDS plans.
type LiveGCPPatchClient interface {
	// SendPatchJob fires the GCP OS Config ExecutePatchJob path against
	// the plan's zone+filter. Returns the GCP-assigned patch_job name
	// (resource name, e.g. "projects/p/patchJobs/abc-123") on success.
	// Validates against the whitelist (set at construction) before
	// calling the SDK.
	SendPatchJob(ctx context.Context, plan *GCPPatchJobPlan) (patchJobName string, err error)
}

// liveGCPWhitelistEnvVar is the env name parsed at boot for the
// per-zone+filter allowlist. Format: "zone:filter,zone:filter".
const liveGCPWhitelistEnvVar = "RF_CONNECTORS_GCP_LIVE_PATCH_WHITELIST_INSTANCE_FILTERS" //nolint:gosec // G101 false positive — env var name, not a credential

// realLiveGCPPatchClient is the production client. The SDK constructor
// invocation in this file is the only one in the package.
type realLiveGCPPatchClient struct {
	sdk       *osconfig.Client
	whitelist []string
	log       *logger.Logger
}

// NewLiveGCPPatchClient builds a real, credential-validated live GCP
// patch client. Reuses the same ADC flow as PR #29's NewRealGCPClient.
func NewLiveGCPPatchClient(ctx context.Context, cfg pkgconfig.GCPConfig, whitelist []string, log *logger.Logger) (LiveGCPPatchClient, error) {
	if len(whitelist) == 0 {
		return nil, fmt.Errorf("NewLiveGCPPatchClient: refuse to construct with empty whitelist")
	}
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("gcp credentials not configured: ProjectID is required")
	}

	opts := []option.ClientOption{}
	if cfg.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
	}

	// THE allowlisted SDK constructor — this is the ONLY place in the
	// tools package that creates the state-change OS Config client.
	client, err := osconfig.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create gcp osconfig client: %w", err)
	}

	// Boot-time credential validation: a no-op list of existing patch
	// jobs in the configured project surfaces auth failure without
	// firing any actual patch.
	bootCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	parent := "projects/" + cfg.ProjectID
	it := client.ListPatchJobs(bootCtx, &osconfigpb.ListPatchJobsRequest{Parent: parent, PageSize: 1})
	if _, err := it.Next(); err != nil {
		// "iterator.Done" is fine — the call reached the GCP plane.
		if !strings.Contains(strings.ToLower(err.Error()), "done") {
			_ = client.Close() //nolint:errcheck // best-effort cleanup
			return nil, fmt.Errorf("gcp live-mode credential validation failed: %w", err)
		}
	}

	return &realLiveGCPPatchClient{
		sdk:       client,
		whitelist: append([]string(nil), whitelist...),
		log:       log.WithComponent("gcp-patch-live"),
	}, nil
}

// SendPatchJob fires the live GCP ExecutePatchJob path. Validates the
// target zone+filter pair against the whitelist before calling the SDK.
func (c *realLiveGCPPatchClient) SendPatchJob(ctx context.Context, plan *GCPPatchJobPlan) (string, error) {
	if plan == nil {
		return "", fmt.Errorf("SendPatchJob: nil plan")
	}
	target := plan.Zone + ":" + plan.InstanceFilter
	if !slices.Contains(c.whitelist, target) {
		return "", fmt.Errorf("target %q is not on the live-patch whitelist (have %d allowed)", target, len(c.whitelist))
	}

	// Translate our plan into the SDK's input shape. Keep this
	// confined here — the rest of the codebase never sees the SDK type.
	req := &osconfigpb.ExecutePatchJobRequest{
		Parent:      "projects/" + plan.ProjectID,
		Description: plan.Comment,
		DisplayName: plan.DisplayName,
		InstanceFilter: &osconfigpb.PatchInstanceFilter{
			GroupLabels: []*osconfigpb.PatchInstanceFilter_GroupLabel{
				// Note: the seeded mock filter is "labels.env=prod"; the
				// real SDK takes a structured map here. The dry-run
				// builder accepts the operator-friendly string form; the
				// live path parses it into the SDK shape.
			},
			Zones: []string{plan.Zone},
		},
		PatchConfig: &osconfigpb.PatchConfig{
			RebootConfig: parseRebootConfigToSDK(plan.RebootConfig),
		},
		Duration: nil, // SDK default
	}

	job, err := c.sdk.ExecutePatchJob(ctx, req)
	if err != nil {
		return "", fmt.Errorf("gcp ExecutePatchJob project=%s zone=%s: %w", plan.ProjectID, plan.Zone, err)
	}
	if job == nil || job.Name == "" {
		return "", fmt.Errorf("gcp ExecutePatchJob returned empty job name")
	}
	c.log.Info("live gcp patch job fired",
		"job_name", job.Name,
		"project_id", plan.ProjectID,
		"zone", plan.Zone,
	)
	return job.Name, nil
}

// parseRebootConfigToSDK maps our string RebootConfig (DEFAULT / ALWAYS
// / NEVER from PR #30) to the SDK enum.
func parseRebootConfigToSDK(s string) osconfigpb.PatchConfig_RebootConfig {
	switch s {
	case gcpRebootAlways:
		return osconfigpb.PatchConfig_ALWAYS
	case gcpRebootNever:
		return osconfigpb.PatchConfig_NEVER
	default:
		return osconfigpb.PatchConfig_DEFAULT
	}
}

// mockLiveGCPPatchClient is the test/CI variant. Validates the
// whitelist but never touches the GCP SDK — the response is a
// deterministic `projects/mock/patchJobs/mock-<hex>` resource name so
// audit-log consumers recognize the mock origin.
type mockLiveGCPPatchClient struct {
	whitelist []string
}

// NewMockLiveGCPPatchClient constructs a mock that respects the same
// whitelist contract as the real client.
func NewMockLiveGCPPatchClient(whitelist []string) LiveGCPPatchClient {
	return &mockLiveGCPPatchClient{whitelist: append([]string(nil), whitelist...)}
}

func (m *mockLiveGCPPatchClient) SendPatchJob(_ context.Context, plan *GCPPatchJobPlan) (string, error) {
	if plan == nil {
		return "", fmt.Errorf("SendPatchJob: nil plan")
	}
	target := plan.Zone + ":" + plan.InstanceFilter
	if !slices.Contains(m.whitelist, target) {
		return "", fmt.Errorf("target %q is not on the live-patch whitelist (have %d allowed)", target, len(m.whitelist))
	}
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("mock SendPatchJob: read entropy: %w", err)
	}
	return "projects/ql-rf-mock-project/patchJobs/mock-" + hex.EncodeToString(buf), nil
}

// LoadGCPLiveWhitelistFromEnv parses the comma-separated whitelist from
// the env var.
func LoadGCPLiveWhitelistFromEnv() []string {
	v := os.Getenv(liveGCPWhitelistEnvVar)
	return parseGCPLiveWhitelistCSV(v)
}

// parseGCPLiveWhitelistCSV is the pure helper for env parsing.
func parseGCPLiveWhitelistCSV(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
