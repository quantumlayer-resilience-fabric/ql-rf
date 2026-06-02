// PR #39 / CONN-016 — K8s server-side-apply client (DRY-RUN ONLY).
//
// SAFETY (READ THIS BEFORE EDITING):
// This file is the K8s equivalent of the SSM / Azure / GCP / vSphere
// dry-run clients. It builds apply plans as plain Go structs and
// never calls the state-change SDK path; PR #40 will introduce
// `live_k8s_apply_client.go` as the SOLE caller of the K8s apply
// method — the structural test in no_k8s_apply_sdk_state_change_test.go
// enforces this by name.
//
// The structural test uses function-call matching (forbidding the
// metav1.ApplyOptions struct literal) rather than import-path
// forbidding, because client-go is already legitimately imported by
// PR #38's read-only path (query_pods).
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// K8sApplyClient builds (does not apply) Kubernetes server-side-apply plans.
type K8sApplyClient interface {
	BuildApplyPlan(ctx context.Context, req K8sApplyRequest) (*K8sApplyPlan, error)
}

// K8sApplyRequest is the typed input. Validation lives here.
type K8sApplyRequest struct {
	// Namespace is the target namespace. Required.
	Namespace string
	// Manifest is the raw JSON-encoded Kubernetes object to apply.
	// (YAML inputs should be converted to JSON before reaching this
	// boundary — keeps the dry-run path stdlib-only with no
	// SDK imports.)
	Manifest string
	// FieldManager identifies the actor for server-side-apply
	// conflict resolution. Required (typically "ql-rf").
	FieldManager string
	// Force overrides conflicts with other field managers.
	// Surfaced in the plan for transparency; consumed by PR #40.
	Force bool
}

// K8sApplyPlan describes the apply that WOULD happen. Never the SDK type.
type K8sApplyPlan struct {
	APIVersion   string `json:"api_version"`
	Kind         string `json:"kind"`
	Namespace    string `json:"namespace"`
	Name         string `json:"name"`
	FieldManager string `json:"field_manager"`
	Force        bool   `json:"force"`
	// ManifestDigest is a deterministic identifier of the manifest body
	// — currently a SHA-style fingerprint of the parsed object's
	// canonical key set. Used to detect drift between plan + live call.
	ManifestDigest string `json:"manifest_digest"`
	// ManifestPreview surfaces a redacted summary (kind/namespace/name +
	// known top-level spec keys) for audit-by-eyeball. Full manifest
	// stays in the request payload, never in the plan JSON.
	ManifestPreview map[string]any `json:"manifest_preview"`
	Comment         string         `json:"comment,omitempty"`
	DryRun          bool           `json:"dry_run"`
	RealChanges     bool           `json:"real_changes"`
}

// realK8sApplyClient validates the request and constructs the plan.
// No client-go imports. No network calls.
type realK8sApplyClient struct {
	log *logger.Logger
}

// NewRealK8sApplyClient constructs the validation-only client.
func NewRealK8sApplyClient(log *logger.Logger) K8sApplyClient {
	return &realK8sApplyClient{log: log.WithComponent("k8s-apply")}
}

// BuildApplyPlan validates the request and constructs the plan.
func (c *realK8sApplyClient) BuildApplyPlan(_ context.Context, req K8sApplyRequest) (*K8sApplyPlan, error) {
	if err := validateK8sNamespace(req.Namespace); err != nil {
		return nil, err
	}
	if req.Manifest == "" {
		return nil, fmt.Errorf("manifest is required (JSON-encoded Kubernetes object)")
	}
	if req.FieldManager == "" {
		return nil, fmt.Errorf("field_manager is required (typically \"ql-rf\")")
	}

	var obj map[string]any
	if err := json.Unmarshal([]byte(req.Manifest), &obj); err != nil {
		return nil, fmt.Errorf("manifest must be valid JSON: %w", err)
	}

	apiVersion, _ := obj["apiVersion"].(string)
	kind, _ := obj["kind"].(string)
	if apiVersion == "" || kind == "" {
		return nil, fmt.Errorf("manifest must include apiVersion and kind")
	}

	metadata, _ := obj["metadata"].(map[string]any)
	name, _ := metadata["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("manifest.metadata.name is required")
	}

	// If manifest has its own namespace, it must match the request's.
	if mns, _ := metadata["namespace"].(string); mns != "" && mns != req.Namespace {
		return nil, fmt.Errorf("manifest.metadata.namespace %q != request namespace %q (refusing to apply across namespaces silently)", mns, req.Namespace)
	}

	preview := map[string]any{
		"apiVersion": apiVersion,
		"kind":       kind,
		"namespace":  req.Namespace,
		"name":       name,
	}
	if spec, ok := obj["spec"].(map[string]any); ok {
		keys := make([]string, 0, len(spec))
		for k := range spec {
			keys = append(keys, k)
		}
		preview["spec_top_level_keys"] = keys
	}

	return &K8sApplyPlan{
		APIVersion:      apiVersion,
		Kind:            kind,
		Namespace:       req.Namespace,
		Name:            name,
		FieldManager:    req.FieldManager,
		Force:           req.Force,
		ManifestDigest:  manifestDigest(obj),
		ManifestPreview: preview,
		Comment:         "QL-RF dry-run (PR #39): plan constructed without invocation.",
		DryRun:          true,
		RealChanges:     false,
	}, nil
}

// mockK8sApplyClient returns a deterministic plan.
type mockK8sApplyClient struct{}

// NewMockK8sApplyClient constructs the deterministic fixture client.
func NewMockK8sApplyClient() K8sApplyClient {
	return &mockK8sApplyClient{}
}

// BuildApplyPlan returns a fixed plan regardless of input.
func (m *mockK8sApplyClient) BuildApplyPlan(_ context.Context, req K8sApplyRequest) (*K8sApplyPlan, error) {
	ns := req.Namespace
	if ns == "" {
		ns = "mock-prod"
	}
	return &K8sApplyPlan{
		APIVersion:     "apps/v1",
		Kind:           "Deployment",
		Namespace:      ns,
		Name:           "mock-app",
		FieldManager:   "ql-rf",
		Force:          req.Force,
		ManifestDigest: "mock-digest-deterministic",
		ManifestPreview: map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"namespace":  ns,
			"name":       "mock-app",
		},
		Comment:     "QL-RF dry-run mock (PR #39): no real cluster.",
		DryRun:      true,
		RealChanges: false,
	}, nil
}

// k8sNamespacePattern follows DNS-1123 label rules used by K8s for
// namespaces: lowercase, hyphens, 1-63 chars.
var k8sNamespacePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?$`)

func validateK8sNamespace(ns string) error {
	if ns == "" {
		return fmt.Errorf("namespace is required")
	}
	if !k8sNamespacePattern.MatchString(ns) {
		return fmt.Errorf("invalid namespace %q: must match DNS-1123 label rules (lowercase + hyphens, 1-63 chars)", ns)
	}
	return nil
}

// manifestDigest produces a stable identifier from the top-level keys
// of the parsed manifest. NOT a cryptographic hash — just enough to
// detect "did the LLM hand the live caller the same thing it dry-ran?"
// at audit time. Live invocation in PR #40 will recompute and compare.
func manifestDigest(obj map[string]any) string {
	apiVersion, _ := obj["apiVersion"].(string)
	kind, _ := obj["kind"].(string)
	metadata, _ := obj["metadata"].(map[string]any)
	name, _ := metadata["name"].(string)
	return fmt.Sprintf("%s/%s/%s", apiVersion, kind, name)
}
