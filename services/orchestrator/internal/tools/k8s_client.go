// PR #38 / CONN-015 — Kubernetes client interface for real cloud tool invocations.
//
// Mirrors aws_client.go (PR #19), azure_client.go (PR #26), gcp_client.go
// (PR #29), and vsphere_client.go (PR #33): a narrow interface (just
// ListPods) with a real implementation backed by client-go and a
// deterministic mock for unit tests + CI.
//
// Credential model: kubeconfig file path (same shape the existing
// connectors service uses, services/connectors/internal/k8s/client.go).
// If empty, client-go falls back to in-cluster config then to the default
// kubeconfig location ($HOME/.kube/config). Boot validates by listing one
// page of pods across the namespace selector.
//
// PR #39 will introduce a K8s dry-run server-side-apply tool, PR #40 the
// live variant — following the SSM / Azure / GCP / vSphere arc patterns.
package tools

import (
	"context"
	"fmt"
	"maps"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// KubernetesClient is the narrow interface the K8s tools call. Keeping
// it small means the mock is trivial and the surface to audit stays
// tight. Each method is read-only by API contract for PR #38.
type KubernetesClient interface {
	// ListPods returns every pod the configured cluster sees, projected
	// into the redacted PodInfo shape (no env vars, no secrets, no
	// container args — fields with security relevance are dropped).
	ListPods(ctx context.Context) ([]PodInfo, error)
}

// PodInfo is the redacted projection of a corev1.Pod surfaced to the
// audit log and the UI.
type PodInfo struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace,omitempty"`
	UID       string            `json:"uid,omitempty"`
	NodeName  string            `json:"node_name,omitempty"`
	Phase     string            `json:"phase,omitempty"`
	PodIP     string            `json:"pod_ip,omitempty"`
	HostIP    string            `json:"host_ip,omitempty"`
	StartTime string            `json:"start_time,omitempty"`
	Images    []string          `json:"images,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// realKubernetesClient wraps a client-go clientset. Constructed once at
// boot; client-go handles connection pooling and token renewal.
type realKubernetesClient struct {
	clientset kubernetes.Interface
	cfg       config.K8sConfig
	log       *logger.Logger
}

// NewRealKubernetesClient builds a real K8s pod-listing client. Validates
// credentials at construction by establishing a clientset and pinging the
// API server (via a minimal List call). Returns an error if the
// kubeconfig is malformed, the context is missing, or the API server
// rejects the request.
func NewRealKubernetesClient(ctx context.Context, cfg config.K8sConfig, log *logger.Logger) (KubernetesClient, error) {
	restCfg, err := buildRESTConfig(cfg)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("k8s clientset: %w", err)
	}

	// Ping the API server with a minimal list to fail fast if credentials
	// are wrong or RBAC denies even read access.
	bootCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, err = clientset.CoreV1().Pods(metav1.NamespaceAll).List(bootCtx, metav1.ListOptions{Limit: 1})
	if err != nil {
		return nil, fmt.Errorf("k8s boot ping: %w", err)
	}

	return &realKubernetesClient{
		clientset: clientset,
		cfg:       cfg,
		log:       log.WithComponent("k8s-tools"),
	}, nil
}

// buildRESTConfig resolves the rest.Config from the orchestrator's
// K8sConfig. Tries in-cluster first (for production deployments running
// inside a pod), then the explicit kubeconfig path, then the default
// loading rules ($HOME/.kube/config).
func buildRESTConfig(cfg config.K8sConfig) (*rest.Config, error) {
	if cfg.Kubeconfig == "" {
		if inCluster, err := rest.InClusterConfig(); err == nil {
			return inCluster, nil
		}
	}
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if cfg.Kubeconfig != "" {
		loadingRules.ExplicitPath = cfg.Kubeconfig
	}
	overrides := &clientcmd.ConfigOverrides{}
	if cfg.Context != "" {
		overrides.CurrentContext = cfg.Context
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
}

// ListPods enumerates pods across the namespaces configured at boot.
// Honors the LabelSelector and ExcludeNamespaces filters from K8sConfig.
func (c *realKubernetesClient) ListPods(ctx context.Context) ([]PodInfo, error) {
	listOpts := metav1.ListOptions{}
	if c.cfg.LabelSelector != "" {
		listOpts.LabelSelector = c.cfg.LabelSelector
	}

	namespaces := c.cfg.Namespaces
	if len(namespaces) == 0 {
		namespaces = []string{metav1.NamespaceAll}
	}

	exclude := make(map[string]struct{}, len(c.cfg.ExcludeNamespaces))
	for _, ns := range c.cfg.ExcludeNamespaces {
		exclude[ns] = struct{}{}
	}

	var out []PodInfo
	for _, ns := range namespaces {
		pods, err := c.clientset.CoreV1().Pods(ns).List(ctx, listOpts)
		if err != nil {
			return nil, fmt.Errorf("k8s list pods (ns=%q): %w", ns, err)
		}
		for i := range pods.Items {
			pod := &pods.Items[i]
			if _, skip := exclude[pod.Namespace]; skip {
				continue
			}
			out = append(out, normalizePod(pod))
		}
	}
	return out, nil
}

// normalizePod projects the (large) corev1.Pod type into the small
// PodInfo shape. Drops env vars, container args, and secrets — fields
// with security relevance.
func normalizePod(pod *corev1.Pod) PodInfo {
	info := PodInfo{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		UID:       string(pod.UID),
		NodeName:  pod.Spec.NodeName,
		Phase:     string(pod.Status.Phase),
		PodIP:     pod.Status.PodIP,
		HostIP:    pod.Status.HostIP,
	}
	if pod.Status.StartTime != nil {
		info.StartTime = pod.Status.StartTime.UTC().Format(time.RFC3339)
	}
	if len(pod.Spec.Containers) > 0 {
		info.Images = make([]string, 0, len(pod.Spec.Containers))
		for i := range pod.Spec.Containers {
			info.Images = append(info.Images, pod.Spec.Containers[i].Image)
		}
	}
	if len(pod.Labels) > 0 {
		info.Labels = make(map[string]string, len(pod.Labels))
		maps.Copy(info.Labels, pod.Labels)
	}
	return info
}

// mockKubernetesClient returns a deterministic two-pod fixture. Used by
// unit tests and CI.
type mockKubernetesClient struct{}

// NewMockKubernetesClient constructs the deterministic fixture client.
func NewMockKubernetesClient() KubernetesClient {
	return &mockKubernetesClient{}
}

// ListPods returns a fixed pair of mock pods.
func (m *mockKubernetesClient) ListPods(_ context.Context) ([]PodInfo, error) {
	return []PodInfo{
		{
			Name:      "mock-app-prod-7d4f8c-abcde",
			Namespace: "prod",
			UID:       "mock-uid-prod-01",
			NodeName:  "mock-node-01",
			Phase:     "Running",
			PodIP:     "10.244.0.5",
			HostIP:    "10.0.1.5",
			StartTime: "2026-06-02T10:00:00Z",
			Images:    []string{"nginx:1.27.0", "envoyproxy/envoy:v1.30.0"},
			Labels: map[string]string{
				"app":         "mock-app",
				"env":         "prod",
				"mock_origin": "ql-rf-test",
			},
		},
		{
			Name:      "mock-app-stage-9f2c3d-fghij",
			Namespace: "stage",
			UID:       "mock-uid-stage-02",
			NodeName:  "mock-node-02",
			Phase:     "Running",
			PodIP:     "10.244.0.7",
			HostIP:    "10.0.1.7",
			StartTime: "2026-06-02T10:30:00Z",
			Images:    []string{"nginx:1.27.0"},
			Labels: map[string]string{
				"app":         "mock-app",
				"env":         "stage",
				"mock_origin": "ql-rf-test",
			},
		},
	}, nil
}
