// PR #29 / CONN-009 — GCP client interface for real cloud tool invocations.
//
// Mirrors aws_client.go (PR #19) and azure_client.go (PR #26): a narrow
// interface (just ListInstances) with a real implementation backed by
// the GCP compute SDK and a deterministic mock for unit tests + CI.
//
// Credential model: GCP application-default credentials (ADC) with an
// optional credentials-file override (`GOOGLE_APPLICATION_CREDENTIALS`).
// Same shape the existing connectors service uses
// (services/connectors/internal/gcp/client.go). Boot validates the
// credential by listing one page of instances aggregated across all
// zones — a cheap call that surfaces auth misconfiguration loudly
// rather than waiting for first real invocation.
//
// PR #30 will introduce a GCP dry-run tool (`gcp_os_config_patch`) and
// PR #31 the live variant (`gcp_os_config_patch_live`) following the
// SSM and Azure arcs (PR #19→#20→#21 and PR #26→#27→#28).
package tools

import (
	"context"
	"errors"
	"fmt"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// GCPClient is the narrow interface the GCP tools call. Keeping it
// small means the mock is trivial and the surface to audit stays tight.
// Each method is read-only by API contract for PR #29.
type GCPClient interface {
	// ListInstances lists Compute Engine instances aggregated across
	// every zone in the configured project. Returns the redacted
	// projection.
	ListInstances(ctx context.Context) ([]GCPInstance, error)
}

// GCPInstance is the redacted projection of a compute.Instance we
// surface to the audit log and the UI. Fields are deliberately boring —
// no service-account scopes, no metadata items, no NIC IDs — so an
// audit-log leak is low-risk.
type GCPInstance struct {
	Name          string            `json:"name"`
	Zone          string            `json:"zone"`
	MachineType   string            `json:"machine_type,omitempty"`
	Status        string            `json:"status,omitempty"`
	InternalIP    string            `json:"internal_ip,omitempty"`
	ProjectID     string            `json:"project_id"`
	ProvisionedAt string            `json:"provisioned_at,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
}

// realGCPClient wraps compute.InstancesClient. Constructed once at boot;
// the ProjectID is fixed (GCP resource hierarchy is per-project) so a
// single client services all listing calls.
type realGCPClient struct {
	instances *compute.InstancesClient
	projectID string
	log       *logger.Logger
}

// NewRealGCPClient builds a real GCP instance-listing client. Validates
// credentials at construction via a cheap aggregated-list first page,
// returning an error if credentials are missing or unusable.
func NewRealGCPClient(ctx context.Context, cfg config.GCPConfig, log *logger.Logger) (GCPClient, error) {
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("gcp credentials not configured: ProjectID is required")
	}

	opts := []option.ClientOption{}
	if cfg.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
	}
	client, err := compute.NewInstancesRESTClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create gcp instances client: %w", err)
	}

	// Boot-time credential validation. Advance the aggregated-list
	// iterator once to confirm the credential works. Doing this with a
	// short timeout keeps a slow / blackholed network from blocking
	// orchestrator boot.
	bootCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	it := client.AggregatedList(bootCtx, &computepb.AggregatedListInstancesRequest{
		Project: cfg.ProjectID,
	})
	if _, err := it.Next(); err != nil && !errors.Is(err, iterator.Done) {
		_ = client.Close() //nolint:errcheck // best-effort cleanup
		return nil, fmt.Errorf("gcp credential validation (aggregated-list) failed: %w", err)
	}

	return &realGCPClient{
		instances: client,
		projectID: cfg.ProjectID,
		log:       log.WithComponent("gcp-tools"),
	}, nil
}

// ListInstances pages through every Compute Engine instance in the
// configured project. The orchestrator's audit log holds the redacted
// projection.
func (c *realGCPClient) ListInstances(ctx context.Context) ([]GCPInstance, error) {
	var results []GCPInstance
	it := c.instances.AggregatedList(ctx, &computepb.AggregatedListInstancesRequest{
		Project: c.projectID,
	})
	for {
		pair, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("gcp aggregated list: %w", err)
		}
		// pair.Key is the zone scope (e.g. "zones/us-central1-a").
		// pair.Value contains InstancesScopedList with Instances slice.
		zone := pair.Key
		if pair.Value == nil {
			continue
		}
		for _, inst := range pair.Value.Instances {
			if inst == nil {
				continue
			}
			results = append(results, normalizeGCPInstance(inst, zone, c.projectID))
		}
	}
	return results, nil
}

// normalizeGCPInstance projects the (large) compute.Instance proto into
// the small GCPInstance shape we surface. Drops fields with security
// relevance — service accounts, metadata, network interfaces — so the
// audit log doesn't carry them.
func normalizeGCPInstance(inst *computepb.Instance, zoneKey, projectID string) GCPInstance {
	out := GCPInstance{
		Name:      inst.GetName(),
		ProjectID: projectID,
	}
	// pair.Key looks like "zones/us-central1-a"; trim the prefix.
	if len(zoneKey) > len("zones/") && zoneKey[:len("zones/")] == "zones/" {
		out.Zone = zoneKey[len("zones/"):]
	} else {
		out.Zone = zoneKey
	}
	if mt := inst.GetMachineType(); mt != "" {
		// machine_type is a URL — extract the last path segment.
		out.MachineType = lastPathSegment(mt)
	}
	out.Status = inst.GetStatus()
	// First NIC's first private IP. Single value to keep the
	// projection narrow; an audit-log shape doesn't need NIC ids.
	if nics := inst.GetNetworkInterfaces(); len(nics) > 0 {
		out.InternalIP = nics[0].GetNetworkIP()
	}
	if ts := inst.GetCreationTimestamp(); ts != "" {
		// GCP returns RFC 3339 directly.
		out.ProvisionedAt = ts
	}
	if labels := inst.GetLabels(); len(labels) > 0 {
		out.Labels = make(map[string]string, len(labels))
		for k, v := range labels {
			out.Labels[k] = v
		}
	}
	return out
}

// lastPathSegment returns the substring after the final '/'. Used to
// shorten GCP resource URLs (e.g. machine_type) into bare names.
func lastPathSegment(s string) string {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return s[i+1:]
		}
	}
	return s
}

// mockGCPClient returns a deterministic two-instance fixture. Used by
// unit tests and by CI (RF_CONNECTORS_GCP_FALLBACK_TO_MOCK=true). The
// mock emits a `mock_origin` label so audit-log consumers can filter.
type mockGCPClient struct{}

// NewMockGCPClient constructs the deterministic fixture client.
func NewMockGCPClient() GCPClient {
	return &mockGCPClient{}
}

// ListInstances returns a fixed pair of mock instances regardless of
// context. The names contain `mock-` so the audit log's origin is
// obvious.
func (m *mockGCPClient) ListInstances(_ context.Context) ([]GCPInstance, error) {
	return []GCPInstance{
		{
			Name:          "mock-gce-prod-01",
			Zone:          "us-central1-a",
			MachineType:   "e2-medium",
			Status:        "RUNNING",
			InternalIP:    "10.0.0.5",
			ProjectID:     "ql-rf-mock-project",
			ProvisionedAt: "2026-04-12T14:11:00Z",
			Labels: map[string]string{
				"env":         "prod",
				"team":        "platform",
				"mock_origin": "ql-rf-test",
			},
		},
		{
			Name:          "mock-gce-stage-02",
			Zone:          "europe-west2-b",
			MachineType:   "n2-standard-2",
			Status:        "RUNNING",
			InternalIP:    "10.0.1.7",
			ProjectID:     "ql-rf-mock-project",
			ProvisionedAt: "2026-05-08T09:45:00Z",
			Labels: map[string]string{
				"env":         "stage",
				"team":        "platform",
				"mock_origin": "ql-rf-test",
			},
		},
	}, nil
}
