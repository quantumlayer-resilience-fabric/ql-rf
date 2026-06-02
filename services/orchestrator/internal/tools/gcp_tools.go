//nolint:dupl // structural duplication with azure_tools.go is deliberate — each cloud's tool metadata reads as its own self-contained file. Sharing a helper would force callers to thread two cloud-specific types through one renderer, which loses the audit-by-grep property the per-tool files have.

// PR #29 / CONN-009 — GCP tools (read-only).
//
// First real GCP cloud-touching tool in the orchestrator. Registered
// only when a GCPClient was successfully constructed at boot (see
// main.go's registerGCPCloudTools); otherwise the tool is absent from
// the registry and GET /api/v1/ai/tools doesn't list it. Read-only by
// API contract — Compute Engine aggregated-list cannot modify cloud
// state.
//
// PR #30 / #31 will add the dry-run + live state-change GCP tools
// (gcp_os_config_patch + gcp_os_config_patch_live) following the same
// arc PR #19 → #20 → #21 used for AWS SSM and PR #26 → #27 → #28 used
// for Azure Run Command.
package tools

import (
	"context"
	"fmt"
)

// QueryGCPInstancesTool lists Compute Engine instances in the
// configured GCP project via a configured GCPClient (real or mock).
// Risk = read_only. Idempotent. Never modifies cloud state.
//
//nolint:dupl // structural duplication with azure_tools.go is deliberate — each cloud's tool metadata reads as its own self-contained file.
type QueryGCPInstancesTool struct {
	client GCPClient
}

// NewQueryGCPInstancesTool constructs the tool with a backing GCPClient.
func NewQueryGCPInstancesTool(client GCPClient) *QueryGCPInstancesTool {
	return &QueryGCPInstancesTool{client: client}
}

func (t *QueryGCPInstancesTool) Name() string {
	return "query_gcp_instances"
}

func (t *QueryGCPInstancesTool) Description() string {
	return "List GCP Compute Engine instances in the configured project (read-only)."
}

func (t *QueryGCPInstancesTool) Risk() RiskLevel {
	return RiskReadOnly
}

func (t *QueryGCPInstancesTool) Scope() Scope {
	return ScopeOrganization
}

func (t *QueryGCPInstancesTool) Idempotent() bool {
	return true
}

func (t *QueryGCPInstancesTool) RequiresApproval() bool {
	return false
}

// Parameters: no inputs required. The project is fixed at boot via
// config.
func (t *QueryGCPInstancesTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

// Execute calls GCPClient.ListInstances and wraps the result in an
// audit-log-friendly envelope.
func (t *QueryGCPInstancesTool) Execute(ctx context.Context, _ map[string]any) (any, error) {
	if t.client == nil {
		return nil, fmt.Errorf("query_gcp_instances: client not configured")
	}

	instances, err := t.client.ListInstances(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"instance_count": len(instances),
		"instances":      instances,
	}, nil
}

// RegisterGCPCloudTools registers GCP-backed cloud tools on the
// registry. Called from main.go ONLY after a GCPClient was successfully
// constructed (real or mock-with-fallback). PR #30/#31 will add
// `gcp_os_config_patch*` registrations through their own files.
func (r *Registry) RegisterGCPCloudTools(client GCPClient) {
	if client == nil {
		r.log.Warn("RegisterGCPCloudTools called with nil client; no GCP tools registered")
		return
	}
	r.register(NewQueryGCPInstancesTool(client))
	r.log.Info("gcp tools: registered", "tools", []string{"query_gcp_instances"})
}
