// PR #40 / CONN-017 — LIVE Kubernetes server-side-apply client.
//
// SAFETY (READ THIS BEFORE EDITING):
// This is THE single file in the tools package allowed to construct
// `metav1.ApplyOptions{...}`. The structural safety test in
// no_k8s_apply_sdk_state_change_test.go grants this file an allowlist
// exception by name; the complementary positive test
// TestLiveK8sApplyClient_IsTheOnlyFileReferencingSDKToken asserts
// (positive direction) that this file DOES reference the token. Both
// tests run on every push.
//
// Live mode is gated at four layers:
//
//  1. Boot env opt-in: RF_CONNECTORS_K8S_ALLOW_LIVE_APPLY=true.
//  2. Mock-conflict refusal: if FallbackToMock is also true, exit 1.
//  3. Per-(namespace,kind) whitelist:
//     RF_CONNECTORS_K8S_LIVE_APPLY_WHITELIST_TARGETS env var.
//     Format: "ns1/Kind1,ns2/Kind2".
//  4. Two-approver workflow: OPA policy + coApproveTask handler (PR #21).
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	pkgconfig "github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// LiveK8sApplyClient FIRES Kubernetes server-side-apply against the
// configured cluster. Distinct from K8sApplyClient (PR #39) which only
// BUILDS plans.
type LiveK8sApplyClient interface {
	// Apply fires the server-side-apply path against the manifest.
	// Returns the applied object's UID + resourceVersion on success.
	// Validates against the (namespace, kind) whitelist (set at
	// construction) before calling the SDK method.
	Apply(ctx context.Context, plan *K8sApplyPlan, manifest string) (*LiveK8sApplyResult, error)
}

// LiveK8sApplyResult is the audit-friendly result of a live apply.
type LiveK8sApplyResult struct {
	UID             string `json:"uid"`
	ResourceVersion string `json:"resource_version"`
	AppliedAt       string `json:"applied_at"`
}

// liveK8sWhitelistEnvVar is the env name parsed at boot.
const liveK8sWhitelistEnvVar = "RF_CONNECTORS_K8S_LIVE_APPLY_WHITELIST_TARGETS"

// liveK8sSupportedGVRs is the hardcoded apiVersion+Kind → GroupVersionResource
// table. Extending live mode to additional kinds = one row here.
//
// Production deployments that need more should fork this map; the
// structural-safety test ensures the SDK call site stays in this file.
var liveK8sSupportedGVRs = map[string]schema.GroupVersionResource{
	"apps/v1+Deployment":  {Group: "apps", Version: "v1", Resource: "deployments"},
	"apps/v1+StatefulSet": {Group: "apps", Version: "v1", Resource: "statefulsets"},
	"apps/v1+DaemonSet":   {Group: "apps", Version: "v1", Resource: "daemonsets"},
	"v1+ConfigMap":        {Group: "", Version: "v1", Resource: "configmaps"},
	"v1+Service":          {Group: "", Version: "v1", Resource: "services"},
	"v1+Secret":           {Group: "", Version: "v1", Resource: "secrets"},
	"batch/v1+Job":        {Group: "batch", Version: "v1", Resource: "jobs"},
	"batch/v1+CronJob":    {Group: "batch", Version: "v1", Resource: "cronjobs"},
}

// realLiveK8sApplyClient is the production client. The metav1.ApplyOptions{...}
// call in this file is the only one in the package.
type realLiveK8sApplyClient struct {
	dyn       dynamic.Interface
	whitelist []string
	log       *logger.Logger
}

// NewLiveK8sApplyClient builds a production live K8s apply client.
// Construction validates whitelist + kubeconfig and returns an error
// if either is missing. The four-gate boot logic in main.go calls
// this only after env opt-in + non-mock conflict have already passed.
func NewLiveK8sApplyClient(
	ctx context.Context,
	cfg pkgconfig.K8sConfig,
	whitelist []string,
	log *logger.Logger,
) (LiveK8sApplyClient, error) {
	if len(whitelist) == 0 {
		return nil, fmt.Errorf("k8s live apply: whitelist is empty — refusing to construct a live client without targets")
	}
	restCfg, err := buildRESTConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("k8s live apply: kubeconfig not configured: %w", err)
	}
	dyn, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("k8s live apply: dynamic client: %w", err)
	}
	// Probe is the read-only path's responsibility (PR #38). Trust the
	// kubeconfig is good if we got this far.
	_ = ctx
	return &realLiveK8sApplyClient{
		dyn:       dyn,
		whitelist: whitelist,
		log:       log.WithComponent("k8s-live-apply"),
	}, nil
}

// Apply fires the K8s server-side-apply path. This is the ONLY place in
// the tools package that constructs `metav1.ApplyOptions{...}`. Any
// other file referencing that token will fail the structural test.
func (c *realLiveK8sApplyClient) Apply(ctx context.Context, plan *K8sApplyPlan, manifest string) (*LiveK8sApplyResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("k8s live apply: plan is nil")
	}
	target := plan.Namespace + "/" + plan.Kind
	if !slices.Contains(c.whitelist, target) {
		return nil, fmt.Errorf("k8s live apply: %q is not on the whitelist; allowed: %s", target, strings.Join(c.whitelist, ", "))
	}

	gvr, ok := liveK8sSupportedGVRs[plan.APIVersion+"+"+plan.Kind]
	if !ok {
		return nil, fmt.Errorf("k8s live apply: kind %q apiVersion %q not in liveK8sSupportedGVRs (extend the map to add)", plan.Kind, plan.APIVersion)
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(manifest), &raw); err != nil {
		return nil, fmt.Errorf("k8s live apply: manifest JSON parse: %w", err)
	}
	obj := &unstructured.Unstructured{Object: raw}
	obj.SetNamespace(plan.Namespace)

	force := plan.Force
	result, err := c.dyn.Resource(gvr).Namespace(plan.Namespace).Apply(
		ctx,
		plan.Name,
		obj,
		metav1.ApplyOptions{FieldManager: plan.FieldManager, Force: force},
	)
	if err != nil {
		return nil, fmt.Errorf("k8s live apply: SDK call: %w", err)
	}

	return &LiveK8sApplyResult{
		UID:             string(result.GetUID()),
		ResourceVersion: result.GetResourceVersion(),
		AppliedAt:       result.GetCreationTimestamp().UTC().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// mockLiveK8sApplyClient validates whitelist + GVR table but returns a
// synthetic UID instead of touching any cluster. Used for local smoke +
// integration tests that want to exercise the full live-mode boot path.
type mockLiveK8sApplyClient struct {
	whitelist []string
}

// NewMockLiveK8sApplyClient constructs the mock live client.
func NewMockLiveK8sApplyClient(whitelist []string) LiveK8sApplyClient {
	return &mockLiveK8sApplyClient{whitelist: whitelist}
}

// Apply validates against the whitelist + GVR table but does not touch
// any cluster. Returns a mock-* UID to signal mock origin.
func (m *mockLiveK8sApplyClient) Apply(_ context.Context, plan *K8sApplyPlan, _ string) (*LiveK8sApplyResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("k8s live apply: plan is nil")
	}
	target := plan.Namespace + "/" + plan.Kind
	if !slices.Contains(m.whitelist, target) {
		return nil, fmt.Errorf("k8s live apply: %q is not on the whitelist (mock)", target)
	}
	if _, ok := liveK8sSupportedGVRs[plan.APIVersion+"+"+plan.Kind]; !ok {
		return nil, fmt.Errorf("k8s live apply: kind %q apiVersion %q not supported (mock)", plan.Kind, plan.APIVersion)
	}
	return &LiveK8sApplyResult{
		UID:             "mock-uid-9000000",
		ResourceVersion: "mock-rv-1",
		AppliedAt:       "2026-06-02T15:00:00Z",
	}, nil
}

// LoadK8sLiveWhitelistFromEnv reads the comma-separated env var and
// returns the parsed targets. Empty slice if the env is unset.
func LoadK8sLiveWhitelistFromEnv(getenv func(string) string) []string {
	return parseK8sLiveWhitelistCSV(getenv(liveK8sWhitelistEnvVar))
}

// parseK8sLiveWhitelistCSV is the lazy parse helper: trim spaces, drop
// empties, accept "ns/Kind" pairs.
func parseK8sLiveWhitelistCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
