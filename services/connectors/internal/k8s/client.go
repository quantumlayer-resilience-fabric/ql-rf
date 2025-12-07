// Package k8s provides Kubernetes connector functionality.
package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/connector"
)

// KubernetesClient is an interface for kubernetes clientset operations.
type KubernetesClient interface {
	kubernetes.Interface
}

// Connector implements the Kubernetes platform connector.
type Connector struct {
	cfg         Config
	client      KubernetesClient
	restConfig  *rest.Config
	log         *logger.Logger
	connected   bool
	clusterName string
}

// Config holds Kubernetes-specific configuration.
type Config struct {
	// Kubeconfig file path. If empty, uses in-cluster config or default kubeconfig.
	Kubeconfig string

	// Context to use from kubeconfig. If empty, uses current-context.
	Context string

	// Namespaces to scan. If empty, scans all namespaces.
	Namespaces []string

	// ExcludeNamespaces to skip during discovery.
	ExcludeNamespaces []string

	// DiscoverNodes enables node discovery.
	DiscoverNodes bool

	// DiscoverDeployments includes deployment metadata.
	DiscoverDeployments bool

	// LabelSelector to filter pods.
	LabelSelector string

	// ClusterName is an optional friendly name for the cluster.
	ClusterName string
}

// New creates a new Kubernetes connector.
func New(cfg Config, log *logger.Logger) *Connector {
	return &Connector{
		cfg: cfg,
		log: log.WithComponent("k8s-connector"),
	}
}

// Name returns the connector name.
func (c *Connector) Name() string {
	return "k8s"
}

// Platform returns the platform type.
func (c *Connector) Platform() models.Platform {
	return models.PlatformK8s
}

// Connect establishes a connection to the Kubernetes cluster.
func (c *Connector) Connect(ctx context.Context) error {
	var restConfig *rest.Config
	var err error

	// Try to load kubeconfig
	if c.cfg.Kubeconfig != "" {
		// Check if kubeconfig is raw YAML content (starts with common YAML markers)
		if isRawKubeconfig(c.cfg.Kubeconfig) {
			// Parse raw YAML content directly
			restConfig, err = c.loadKubeconfigFromContent(c.cfg.Kubeconfig, c.cfg.Context)
		} else {
			// Treat as file path
			restConfig, err = c.loadKubeconfig(c.cfg.Kubeconfig, c.cfg.Context)
		}
	} else {
		// Try in-cluster config first
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to default kubeconfig location
			homeDir, _ := os.UserHomeDir()
			defaultKubeconfig := filepath.Join(homeDir, ".kube", "config")
			restConfig, err = c.loadKubeconfig(defaultKubeconfig, c.cfg.Context)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to create kubernetes config: %w", err)
	}

	// Create clientset
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	c.restConfig = restConfig
	c.client = client
	c.connected = true

	// Try to determine cluster name
	c.clusterName = c.determineClusterName(ctx)

	c.log.Info("connected to Kubernetes cluster",
		"cluster", c.clusterName,
		"host", restConfig.Host,
	)

	return nil
}

// loadKubeconfig loads a kubeconfig file from a path.
func (c *Connector) loadKubeconfig(path string, context string) (*rest.Config, error) {
	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: path}
	configOverrides := &clientcmd.ConfigOverrides{}

	if context != "" {
		configOverrides.CurrentContext = context
	}

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	return kubeConfig.ClientConfig()
}

// loadKubeconfigFromContent loads kubeconfig from raw YAML content.
func (c *Connector) loadKubeconfigFromContent(content string, context string) (*rest.Config, error) {
	// Parse the raw kubeconfig YAML content
	clientConfig, err := clientcmd.NewClientConfigFromBytes([]byte(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig content: %w", err)
	}

	// If a specific context is requested, we need to build a config with that context
	if context != "" {
		// Load the raw config to check available contexts
		rawConfig, err := clientConfig.RawConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get raw config: %w", err)
		}

		// Check if the context exists
		if _, ok := rawConfig.Contexts[context]; !ok {
			return nil, fmt.Errorf("context %q not found in kubeconfig", context)
		}

		// Create a new client config with the specified context
		configOverrides := &clientcmd.ConfigOverrides{
			CurrentContext: context,
		}
		clientConfig = clientcmd.NewNonInteractiveClientConfig(rawConfig, context, configOverrides, nil)
	}

	return clientConfig.ClientConfig()
}

// isRawKubeconfig checks if the string is raw YAML kubeconfig content rather than a file path.
func isRawKubeconfig(s string) bool {
	trimmed := strings.TrimSpace(s)
	// Check for common kubeconfig YAML markers
	// A kubeconfig file always starts with "apiVersion:" or has multiline YAML structure
	if strings.HasPrefix(trimmed, "apiVersion:") {
		return true
	}
	// Also check if it contains multiple lines with YAML-like content
	// File paths typically don't contain newlines or colons followed by spaces
	if strings.Contains(trimmed, "\n") && strings.Contains(trimmed, ": ") {
		return true
	}
	// Check for base64-encoded certificate data (common in kubeconfig)
	if strings.Contains(trimmed, "certificate-authority-data:") ||
		strings.Contains(trimmed, "client-certificate-data:") ||
		strings.Contains(trimmed, "client-key-data:") {
		return true
	}
	return false
}

// determineClusterName attempts to determine the cluster name.
func (c *Connector) determineClusterName(ctx context.Context) string {
	// Use configured cluster name if provided
	if c.cfg.ClusterName != "" {
		return c.cfg.ClusterName
	}

	// Try to extract from kubeconfig context
	if c.cfg.Context != "" {
		return c.cfg.Context
	}

	// Try to get from cluster info
	if c.restConfig != nil && c.restConfig.Host != "" {
		// Extract hostname from API server URL
		host := c.restConfig.Host
		host = strings.TrimPrefix(host, "https://")
		host = strings.TrimPrefix(host, "http://")
		if idx := strings.Index(host, ":"); idx > 0 {
			host = host[:idx]
		}
		return host
	}

	return "unknown-cluster"
}

// Close closes the Kubernetes connection.
func (c *Connector) Close() error {
	c.connected = false
	return nil
}

// Health checks the health of the Kubernetes connection.
func (c *Connector) Health(ctx context.Context) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	// Try to get server version as health check
	_, err := c.client.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// DiscoverAssets discovers all pods and optionally nodes from Kubernetes.
func (c *Connector) DiscoverAssets(ctx context.Context, orgID uuid.UUID) ([]models.NormalizedAsset, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	var allAssets []models.NormalizedAsset

	// Discover pods
	pods, err := c.discoverPods(ctx)
	if err != nil {
		c.log.Error("failed to discover pods", "error", err)
	} else {
		allAssets = append(allAssets, pods...)
	}

	// Optionally discover nodes
	if c.cfg.DiscoverNodes {
		nodes, err := c.discoverNodes(ctx)
		if err != nil {
			c.log.Error("failed to discover nodes", "error", err)
		} else {
			allAssets = append(allAssets, nodes...)
		}
	}

	c.log.Info("asset discovery completed",
		"total_assets", len(allAssets),
		"cluster", c.clusterName,
	)

	return allAssets, nil
}

// discoverPods discovers all pods matching the configuration.
func (c *Connector) discoverPods(ctx context.Context) ([]models.NormalizedAsset, error) {
	var assets []models.NormalizedAsset

	// Get namespaces to scan
	namespaces, err := c.getNamespacesToScan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespaces: %w", err)
	}

	// Build list options
	listOpts := metav1.ListOptions{}
	if c.cfg.LabelSelector != "" {
		listOpts.LabelSelector = c.cfg.LabelSelector
	}

	// Get deployment info map if enabled
	var deploymentMap map[string]deploymentInfo
	if c.cfg.DiscoverDeployments {
		deploymentMap, err = c.getDeploymentMap(ctx, namespaces)
		if err != nil {
			c.log.Warn("failed to get deployment info", "error", err)
			deploymentMap = make(map[string]deploymentInfo)
		}
	}

	for _, namespace := range namespaces {
		pods, err := c.client.CoreV1().Pods(namespace).List(ctx, listOpts)
		if err != nil {
			c.log.Warn("failed to list pods in namespace",
				"namespace", namespace,
				"error", err,
			)
			continue
		}

		for _, pod := range pods.Items {
			// Skip pods that are not running or pending
			if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodPending {
				continue
			}

			asset := c.normalizePod(pod, deploymentMap)
			assets = append(assets, asset)
		}
	}

	c.log.Debug("discovered pods",
		"count", len(assets),
		"namespaces", len(namespaces),
	)

	return assets, nil
}

// discoverNodes discovers all nodes in the cluster.
func (c *Connector) discoverNodes(ctx context.Context) ([]models.NormalizedAsset, error) {
	var assets []models.NormalizedAsset

	nodes, err := c.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	for _, node := range nodes.Items {
		asset := c.normalizeNode(node)
		assets = append(assets, asset)
	}

	c.log.Debug("discovered nodes", "count", len(assets))

	return assets, nil
}

// getNamespacesToScan returns the list of namespaces to scan.
func (c *Connector) getNamespacesToScan(ctx context.Context) ([]string, error) {
	// If specific namespaces are configured, use those
	if len(c.cfg.Namespaces) > 0 {
		return c.cfg.Namespaces, nil
	}

	// Otherwise, list all namespaces and filter excluded ones
	nsList, err := c.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	excludeSet := make(map[string]bool)
	for _, ns := range c.cfg.ExcludeNamespaces {
		excludeSet[ns] = true
	}

	var namespaces []string
	for _, ns := range nsList.Items {
		if !excludeSet[ns.Name] {
			namespaces = append(namespaces, ns.Name)
		}
	}

	return namespaces, nil
}

// deploymentInfo holds information about a deployment.
type deploymentInfo struct {
	Name     string
	Replicas int32
	Labels   map[string]string
}

// getDeploymentMap returns a map of pod owner -> deployment info.
func (c *Connector) getDeploymentMap(ctx context.Context, namespaces []string) (map[string]deploymentInfo, error) {
	deploymentMap := make(map[string]deploymentInfo)

	for _, namespace := range namespaces {
		deployments, err := c.client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}

		for _, deploy := range deployments.Items {
			key := fmt.Sprintf("%s/%s", namespace, deploy.Name)
			var replicas int32
			if deploy.Spec.Replicas != nil {
				replicas = *deploy.Spec.Replicas
			}
			deploymentMap[key] = deploymentInfo{
				Name:     deploy.Name,
				Replicas: replicas,
				Labels:   deploy.Labels,
			}
		}
	}

	return deploymentMap, nil
}

// normalizePod converts a Kubernetes pod to a NormalizedAsset.
func (c *Connector) normalizePod(pod corev1.Pod, deploymentMap map[string]deploymentInfo) models.NormalizedAsset {
	// Extract container images
	var imageRef, imageVersion string
	if len(pod.Spec.Containers) > 0 {
		// Use the first container's image as the primary image
		imageRef, imageVersion = parseContainerImage(pod.Spec.Containers[0].Image)
	}

	// Build tags from labels and annotations
	tags := make(map[string]string)
	for k, v := range pod.Labels {
		tags["label:"+k] = v
	}

	// Add namespace as tag
	tags["namespace"] = pod.Namespace

	// Add node name
	if pod.Spec.NodeName != "" {
		tags["node"] = pod.Spec.NodeName
	}

	// Add owner reference info
	for _, owner := range pod.OwnerReferences {
		tags["owner:kind"] = owner.Kind
		tags["owner:name"] = owner.Name
		break // Just take the first owner
	}

	// Add deployment info if available
	if deploymentMap != nil {
		for _, owner := range pod.OwnerReferences {
			if owner.Kind == "ReplicaSet" {
				// ReplicaSet name format: deployment-name-<hash>
				// Try to find matching deployment
				parts := strings.Split(owner.Name, "-")
				if len(parts) >= 2 {
					// Remove last part (hash) to get deployment name
					deployName := strings.Join(parts[:len(parts)-1], "-")
					key := fmt.Sprintf("%s/%s", pod.Namespace, deployName)
					if depInfo, ok := deploymentMap[key]; ok {
						tags["deployment"] = depInfo.Name
						tags["deployment:replicas"] = fmt.Sprintf("%d", depInfo.Replicas)
					}
				}
			}
		}
	}

	// Add all container images as tags
	for i, container := range pod.Spec.Containers {
		tags[fmt.Sprintf("container:%d:name", i)] = container.Name
		tags[fmt.Sprintf("container:%d:image", i)] = container.Image
	}

	// Map pod phase to asset state
	state := models.AssetStateUnknown
	switch pod.Status.Phase {
	case corev1.PodRunning:
		state = models.AssetStateRunning
	case corev1.PodPending:
		state = models.AssetStatePending
	case corev1.PodSucceeded, corev1.PodFailed:
		state = models.AssetStateTerminated
	}

	return models.NormalizedAsset{
		Platform:     models.PlatformK8s,
		Account:      c.clusterName,
		Region:       pod.Namespace, // Use namespace as region equivalent
		InstanceID:   string(pod.UID),
		Name:         pod.Name,
		ImageRef:     imageRef,
		ImageVersion: imageVersion,
		State:        state,
		Tags:         tags,
	}
}

// normalizeNode converts a Kubernetes node to a NormalizedAsset.
func (c *Connector) normalizeNode(node corev1.Node) models.NormalizedAsset {
	// Extract node info
	var imageRef, imageVersion string

	// Use OS image as the "image" for nodes
	if node.Status.NodeInfo.OSImage != "" {
		imageRef = node.Status.NodeInfo.OSImage
	}
	if node.Status.NodeInfo.KubeletVersion != "" {
		imageVersion = node.Status.NodeInfo.KubeletVersion
	}

	// Build tags from labels
	tags := make(map[string]string)
	for k, v := range node.Labels {
		tags["label:"+k] = v
	}

	// Add node info
	tags["arch"] = node.Status.NodeInfo.Architecture
	tags["os"] = node.Status.NodeInfo.OperatingSystem
	tags["kernel"] = node.Status.NodeInfo.KernelVersion
	tags["container_runtime"] = node.Status.NodeInfo.ContainerRuntimeVersion
	tags["type"] = "node"

	// Add capacity info
	if cpu := node.Status.Capacity.Cpu(); cpu != nil {
		tags["capacity:cpu"] = cpu.String()
	}
	if mem := node.Status.Capacity.Memory(); mem != nil {
		tags["capacity:memory"] = mem.String()
	}

	// Determine node state from conditions
	state := models.AssetStateUnknown
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				state = models.AssetStateRunning
			} else {
				state = models.AssetStateStopped
			}
			break
		}
	}

	// Determine region from labels (common patterns)
	region := ""
	if zone, ok := node.Labels["topology.kubernetes.io/zone"]; ok {
		region = zone
	} else if zone, ok := node.Labels["failure-domain.beta.kubernetes.io/zone"]; ok {
		region = zone
	}

	return models.NormalizedAsset{
		Platform:     models.PlatformK8s,
		Account:      c.clusterName,
		Region:       region,
		InstanceID:   string(node.UID),
		Name:         node.Name,
		ImageRef:     imageRef,
		ImageVersion: imageVersion,
		State:        state,
		Tags:         tags,
	}
}

// parseContainerImage parses a container image string into image ref and version.
func parseContainerImage(image string) (imageRef, version string) {
	// Container images can be in formats like:
	// - nginx
	// - nginx:1.19
	// - registry.example.com/nginx:1.19
	// - registry.example.com/nginx@sha256:abc123
	// - registry.example.com:5000/nginx:1.19

	// Check for digest
	if idx := strings.Index(image, "@"); idx > 0 {
		imageRef = image[:idx]
		version = image[idx+1:]
		return
	}

	// Check for tag
	// Need to be careful about registry ports (e.g., registry:5000/image:tag)
	lastColon := strings.LastIndex(image, ":")
	if lastColon > 0 {
		// Check if this colon is part of a port (registry:port/image)
		afterColon := image[lastColon+1:]
		if !strings.Contains(afterColon, "/") {
			// This is a tag, not a port
			imageRef = image[:lastColon]
			version = afterColon
			return
		}
	}

	// No version found
	imageRef = image
	version = "latest"
	return
}

// DiscoverImages discovers container images used in the cluster.
func (c *Connector) DiscoverImages(ctx context.Context) ([]connector.ImageInfo, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	// Collect unique images from all pods
	imageSet := make(map[string]bool)
	var images []connector.ImageInfo

	// Get namespaces to scan
	namespaces, err := c.getNamespacesToScan(ctx)
	if err != nil {
		return nil, err
	}

	for _, namespace := range namespaces {
		pods, err := c.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}

		for _, pod := range pods.Items {
			for _, container := range pod.Spec.Containers {
				if imageSet[container.Image] {
					continue
				}
				imageSet[container.Image] = true

				imageRef, version := parseContainerImage(container.Image)
				images = append(images, connector.ImageInfo{
					Platform:   models.PlatformK8s,
					Identifier: container.Image,
					Name:       imageRef,
					Region:     namespace,
					CreatedAt:  "", // Not available from pod spec
					Tags: map[string]string{
						"version":   version,
						"namespace": namespace,
					},
				})
			}
		}
	}

	c.log.Info("image discovery completed", "count", len(images))

	return images, nil
}
