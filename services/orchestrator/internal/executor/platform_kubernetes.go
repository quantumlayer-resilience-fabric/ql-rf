// Package executor implements the plan execution engine.
package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// KubernetesPlatformClient implements PlatformClient for Kubernetes.
// Handles rolling updates, canary deployments, and rollbacks for Kubernetes workloads.
type KubernetesPlatformClient struct {
	cfg       KubernetesClientConfig
	clientset *kubernetes.Clientset
	log       *logger.Logger
	connected bool
}

// KubernetesClientConfig holds Kubernetes client configuration.
type KubernetesClientConfig struct {
	KubeConfig     string // Path to kubeconfig file (for out-of-cluster)
	Context        string // Kubernetes context to use
	InCluster      bool   // Use in-cluster configuration
	DefaultNS      string // Default namespace if not specified
	RolloutTimeout time.Duration
}

// NewKubernetesPlatformClient creates a new Kubernetes platform client.
func NewKubernetesPlatformClient(cfg KubernetesClientConfig, log *logger.Logger) *KubernetesPlatformClient {
	if cfg.DefaultNS == "" {
		cfg.DefaultNS = "default"
	}
	if cfg.RolloutTimeout == 0 {
		cfg.RolloutTimeout = 10 * time.Minute
	}
	return &KubernetesPlatformClient{
		cfg: cfg,
		log: log.WithComponent("kubernetes-platform-client"),
	}
}

// Connect establishes a connection to Kubernetes.
func (c *KubernetesPlatformClient) Connect(ctx context.Context) error {
	var config *rest.Config
	var err error

	if c.cfg.InCluster {
		// Use in-cluster configuration
		config, err = rest.InClusterConfig()
		if err != nil {
			return fmt.Errorf("failed to get in-cluster config: %w", err)
		}
		c.log.Info("using in-cluster Kubernetes configuration")
	} else {
		// Build configuration from kubeconfig
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		if c.cfg.KubeConfig != "" {
			loadingRules.ExplicitPath = c.cfg.KubeConfig
		}

		configOverrides := &clientcmd.ConfigOverrides{}
		if c.cfg.Context != "" {
			configOverrides.CurrentContext = c.cfg.Context
		}

		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err = kubeConfig.ClientConfig()
		if err != nil {
			return fmt.Errorf("failed to build kubeconfig: %w", err)
		}
		c.log.Info("using kubeconfig", "path", c.cfg.KubeConfig, "context", c.cfg.Context)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}
	c.clientset = clientset

	// Verify connection by getting server version
	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to connect to kubernetes: %w", err)
	}

	c.connected = true
	c.log.Info("connected to Kubernetes",
		"server_version", version.String(),
	)

	return nil
}

// Close closes the Kubernetes connection.
func (c *KubernetesPlatformClient) Close() error {
	c.connected = false
	return nil
}

// ReimageInstance updates a Kubernetes workload with a new container image.
// For Kubernetes, this triggers a rolling update of the deployment/daemonset.
// instanceID format: "namespace/kind/name" e.g., "production/deployment/nginx"
func (c *KubernetesPlatformClient) ReimageInstance(ctx context.Context, instanceID, imageID string) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	namespace, kind, name, err := c.parseInstanceID(instanceID)
	if err != nil {
		return err
	}

	c.log.Info("updating workload image",
		"namespace", namespace,
		"kind", kind,
		"name", name,
		"new_image", imageID,
	)

	switch strings.ToLower(kind) {
	case "deployment":
		return c.updateDeploymentImage(ctx, namespace, name, imageID)
	case "daemonset":
		return c.updateDaemonSetImage(ctx, namespace, name, imageID)
	case "statefulset":
		return c.updateStatefulSetImage(ctx, namespace, name, imageID)
	default:
		return fmt.Errorf("unsupported workload kind: %s", kind)
	}
}

// updateDeploymentImage updates a Deployment's container image and waits for rollout.
func (c *KubernetesPlatformClient) updateDeploymentImage(ctx context.Context, namespace, name, imageID string) error {
	deploymentsClient := c.clientset.AppsV1().Deployments(namespace)

	// Get current deployment
	deployment, err := deploymentsClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Update all containers if imageID doesn't specify a container
	// Format: "container-name=image:tag" or just "image:tag" for all containers
	containerName, image := parseImageSpec(imageID)

	updated := false
	for i := range deployment.Spec.Template.Spec.Containers {
		container := &deployment.Spec.Template.Spec.Containers[i]
		if containerName == "" || container.Name == containerName {
			c.log.Info("updating container image",
				"container", container.Name,
				"old_image", container.Image,
				"new_image", image,
			)
			container.Image = image
			updated = true
		}
	}

	if !updated {
		return fmt.Errorf("no matching containers found to update")
	}

	// Add annotation to force rollout even if image tag is the same
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["qlrf.quantumlayer.io/rollout-time"] = time.Now().Format(time.RFC3339)

	// Apply the update
	_, err = deploymentsClient.Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	// Wait for rollout to complete
	return c.waitForDeploymentRollout(ctx, namespace, name)
}

// updateDaemonSetImage updates a DaemonSet's container image.
func (c *KubernetesPlatformClient) updateDaemonSetImage(ctx context.Context, namespace, name, imageID string) error {
	daemonSetsClient := c.clientset.AppsV1().DaemonSets(namespace)

	daemonset, err := daemonSetsClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get daemonset: %w", err)
	}

	containerName, image := parseImageSpec(imageID)

	for i := range daemonset.Spec.Template.Spec.Containers {
		container := &daemonset.Spec.Template.Spec.Containers[i]
		if containerName == "" || container.Name == containerName {
			container.Image = image
		}
	}

	if daemonset.Spec.Template.Annotations == nil {
		daemonset.Spec.Template.Annotations = make(map[string]string)
	}
	daemonset.Spec.Template.Annotations["qlrf.quantumlayer.io/rollout-time"] = time.Now().Format(time.RFC3339)

	_, err = daemonSetsClient.Update(ctx, daemonset, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update daemonset: %w", err)
	}

	return c.waitForDaemonSetRollout(ctx, namespace, name)
}

// updateStatefulSetImage updates a StatefulSet's container image.
func (c *KubernetesPlatformClient) updateStatefulSetImage(ctx context.Context, namespace, name, imageID string) error {
	statefulSetsClient := c.clientset.AppsV1().StatefulSets(namespace)

	statefulset, err := statefulSetsClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get statefulset: %w", err)
	}

	containerName, image := parseImageSpec(imageID)

	for i := range statefulset.Spec.Template.Spec.Containers {
		container := &statefulset.Spec.Template.Spec.Containers[i]
		if containerName == "" || container.Name == containerName {
			container.Image = image
		}
	}

	if statefulset.Spec.Template.Annotations == nil {
		statefulset.Spec.Template.Annotations = make(map[string]string)
	}
	statefulset.Spec.Template.Annotations["qlrf.quantumlayer.io/rollout-time"] = time.Now().Format(time.RFC3339)

	_, err = statefulSetsClient.Update(ctx, statefulset, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update statefulset: %w", err)
	}

	return c.waitForStatefulSetRollout(ctx, namespace, name)
}

// waitForDeploymentRollout waits for a deployment rollout to complete.
func (c *KubernetesPlatformClient) waitForDeploymentRollout(ctx context.Context, namespace, name string) error {
	deploymentsClient := c.clientset.AppsV1().Deployments(namespace)

	c.log.Info("waiting for deployment rollout", "namespace", namespace, "name", name)

	deadline := time.Now().Add(c.cfg.RolloutTimeout)
	pollInterval := 5 * time.Second

	for time.Now().Before(deadline) {
		deployment, err := deploymentsClient.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get deployment: %w", err)
		}

		// Check rollout status
		if deployment.Generation == deployment.Status.ObservedGeneration {
			if deployment.Status.UpdatedReplicas == *deployment.Spec.Replicas &&
				deployment.Status.ReadyReplicas == *deployment.Spec.Replicas &&
				deployment.Status.AvailableReplicas == *deployment.Spec.Replicas {
				c.log.Info("deployment rollout complete",
					"namespace", namespace,
					"name", name,
					"replicas", *deployment.Spec.Replicas,
				)
				return nil
			}
		}

		c.log.Debug("deployment rollout in progress",
			"updated", deployment.Status.UpdatedReplicas,
			"ready", deployment.Status.ReadyReplicas,
			"available", deployment.Status.AvailableReplicas,
			"desired", *deployment.Spec.Replicas,
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return fmt.Errorf("timeout waiting for deployment rollout")
}

// waitForDaemonSetRollout waits for a daemonset rollout to complete.
func (c *KubernetesPlatformClient) waitForDaemonSetRollout(ctx context.Context, namespace, name string) error {
	daemonSetsClient := c.clientset.AppsV1().DaemonSets(namespace)

	c.log.Info("waiting for daemonset rollout", "namespace", namespace, "name", name)

	deadline := time.Now().Add(c.cfg.RolloutTimeout)
	pollInterval := 5 * time.Second

	for time.Now().Before(deadline) {
		daemonset, err := daemonSetsClient.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get daemonset: %w", err)
		}

		if daemonset.Generation == daemonset.Status.ObservedGeneration {
			if daemonset.Status.UpdatedNumberScheduled == daemonset.Status.DesiredNumberScheduled &&
				daemonset.Status.NumberReady == daemonset.Status.DesiredNumberScheduled {
				c.log.Info("daemonset rollout complete",
					"namespace", namespace,
					"name", name,
					"nodes", daemonset.Status.DesiredNumberScheduled,
				)
				return nil
			}
		}

		c.log.Debug("daemonset rollout in progress",
			"updated", daemonset.Status.UpdatedNumberScheduled,
			"ready", daemonset.Status.NumberReady,
			"desired", daemonset.Status.DesiredNumberScheduled,
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return fmt.Errorf("timeout waiting for daemonset rollout")
}

// waitForStatefulSetRollout waits for a statefulset rollout to complete.
func (c *KubernetesPlatformClient) waitForStatefulSetRollout(ctx context.Context, namespace, name string) error {
	statefulSetsClient := c.clientset.AppsV1().StatefulSets(namespace)

	c.log.Info("waiting for statefulset rollout", "namespace", namespace, "name", name)

	deadline := time.Now().Add(c.cfg.RolloutTimeout)
	pollInterval := 5 * time.Second

	for time.Now().Before(deadline) {
		statefulset, err := statefulSetsClient.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get statefulset: %w", err)
		}

		if statefulset.Generation == statefulset.Status.ObservedGeneration {
			if statefulset.Status.UpdatedReplicas == *statefulset.Spec.Replicas &&
				statefulset.Status.ReadyReplicas == *statefulset.Spec.Replicas {
				c.log.Info("statefulset rollout complete",
					"namespace", namespace,
					"name", name,
					"replicas", *statefulset.Spec.Replicas,
				)
				return nil
			}
		}

		c.log.Debug("statefulset rollout in progress",
			"updated", statefulset.Status.UpdatedReplicas,
			"ready", statefulset.Status.ReadyReplicas,
			"desired", *statefulset.Spec.Replicas,
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return fmt.Errorf("timeout waiting for statefulset rollout")
}

// RebootInstance restarts all pods in a workload by patching annotations.
func (c *KubernetesPlatformClient) RebootInstance(ctx context.Context, instanceID string) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	namespace, kind, name, err := c.parseInstanceID(instanceID)
	if err != nil {
		return err
	}

	c.log.Info("restarting workload", "namespace", namespace, "kind", kind, "name", name)

	// Patch to trigger a rollout restart
	patchData := []byte(fmt.Sprintf(
		`{"spec":{"template":{"metadata":{"annotations":{"qlrf.quantumlayer.io/restart-time":"%s"}}}}}`,
		time.Now().Format(time.RFC3339),
	))

	switch strings.ToLower(kind) {
	case "deployment":
		_, err = c.clientset.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, patchData, metav1.PatchOptions{})
		if err == nil {
			err = c.waitForDeploymentRollout(ctx, namespace, name)
		}
	case "daemonset":
		_, err = c.clientset.AppsV1().DaemonSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, patchData, metav1.PatchOptions{})
		if err == nil {
			err = c.waitForDaemonSetRollout(ctx, namespace, name)
		}
	case "statefulset":
		_, err = c.clientset.AppsV1().StatefulSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, patchData, metav1.PatchOptions{})
		if err == nil {
			err = c.waitForStatefulSetRollout(ctx, namespace, name)
		}
	default:
		return fmt.Errorf("unsupported workload kind: %s", kind)
	}

	return err
}

// TerminateInstance scales a workload to zero replicas.
func (c *KubernetesPlatformClient) TerminateInstance(ctx context.Context, instanceID string) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	namespace, kind, name, err := c.parseInstanceID(instanceID)
	if err != nil {
		return err
	}

	c.log.Info("scaling workload to zero", "namespace", namespace, "kind", kind, "name", name)

	zero := int32(0)

	switch strings.ToLower(kind) {
	case "deployment":
		deployment, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get deployment: %w", err)
		}
		deployment.Spec.Replicas = &zero
		_, err = c.clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
		return err
	case "statefulset":
		statefulset, err := c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get statefulset: %w", err)
		}
		statefulset.Spec.Replicas = &zero
		_, err = c.clientset.AppsV1().StatefulSets(namespace).Update(ctx, statefulset, metav1.UpdateOptions{})
		return err
	default:
		return fmt.Errorf("cannot terminate workload of kind: %s", kind)
	}
}

// GetInstanceStatus gets the current status of a Kubernetes workload.
func (c *KubernetesPlatformClient) GetInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	if !c.connected {
		return "", fmt.Errorf("not connected")
	}

	namespace, kind, name, err := c.parseInstanceID(instanceID)
	if err != nil {
		return "", err
	}

	switch strings.ToLower(kind) {
	case "deployment":
		deployment, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to get deployment: %w", err)
		}
		return c.getDeploymentStatus(deployment), nil
	case "daemonset":
		daemonset, err := c.clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to get daemonset: %w", err)
		}
		return c.getDaemonSetStatus(daemonset), nil
	case "statefulset":
		statefulset, err := c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to get statefulset: %w", err)
		}
		return c.getStatefulSetStatus(statefulset), nil
	default:
		return "", fmt.Errorf("unsupported workload kind: %s", kind)
	}
}

func (c *KubernetesPlatformClient) getDeploymentStatus(d *appsv1.Deployment) string {
	if d.Spec.Replicas != nil && *d.Spec.Replicas == 0 {
		return "scaled_to_zero"
	}
	if d.Status.ReadyReplicas == *d.Spec.Replicas {
		return "running"
	}
	if d.Status.UpdatedReplicas < *d.Spec.Replicas {
		return "updating"
	}
	return "degraded"
}

func (c *KubernetesPlatformClient) getDaemonSetStatus(ds *appsv1.DaemonSet) string {
	if ds.Status.NumberReady == ds.Status.DesiredNumberScheduled {
		return "running"
	}
	if ds.Status.UpdatedNumberScheduled < ds.Status.DesiredNumberScheduled {
		return "updating"
	}
	return "degraded"
}

func (c *KubernetesPlatformClient) getStatefulSetStatus(ss *appsv1.StatefulSet) string {
	if ss.Spec.Replicas != nil && *ss.Spec.Replicas == 0 {
		return "scaled_to_zero"
	}
	if ss.Status.ReadyReplicas == *ss.Spec.Replicas {
		return "running"
	}
	if ss.Status.UpdatedReplicas < *ss.Spec.Replicas {
		return "updating"
	}
	return "degraded"
}

// WaitForInstanceState waits for a workload to reach a specific state.
func (c *KubernetesPlatformClient) WaitForInstanceState(ctx context.Context, instanceID, targetState string, timeout time.Duration) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	c.log.Debug("waiting for workload state",
		"instance_id", instanceID,
		"target_state", targetState,
		"timeout", timeout,
	)

	deadline := time.Now().Add(timeout)
	pollInterval := 5 * time.Second

	for time.Now().Before(deadline) {
		status, err := c.GetInstanceStatus(ctx, instanceID)
		if err != nil {
			return err
		}

		if status == targetState {
			c.log.Debug("workload reached target state",
				"instance_id", instanceID,
				"state", targetState,
			)
			return nil
		}

		select {
		case <-time.After(pollInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("timeout waiting for workload to reach state %s", targetState)
}

// ApplyPatches for Kubernetes means updating container images.
// This is the primary "patching" mechanism for containerized workloads.
func (c *KubernetesPlatformClient) ApplyPatches(ctx context.Context, instanceID string, params map[string]interface{}) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	// Get the new image from params
	imageID, ok := params["image"].(string)
	if !ok || imageID == "" {
		return fmt.Errorf("image parameter required for Kubernetes patching")
	}

	// Use ReimageInstance which handles the rolling update
	return c.ReimageInstance(ctx, instanceID, imageID)
}

// GetPatchStatus for Kubernetes checks if the workload is running the expected image.
func (c *KubernetesPlatformClient) GetPatchStatus(ctx context.Context, instanceID string) (string, error) {
	if !c.connected {
		return "", fmt.Errorf("not connected")
	}

	namespace, kind, name, err := c.parseInstanceID(instanceID)
	if err != nil {
		return "", err
	}

	// Get the workload and check all pods have the same image
	switch strings.ToLower(kind) {
	case "deployment":
		deployment, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to get deployment: %w", err)
		}
		if deployment.Status.UpdatedReplicas == *deployment.Spec.Replicas &&
			deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
			return "COMPLIANT", nil
		}
		return "NON_COMPLIANT", nil
	case "daemonset":
		daemonset, err := c.clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to get daemonset: %w", err)
		}
		if daemonset.Status.UpdatedNumberScheduled == daemonset.Status.DesiredNumberScheduled {
			return "COMPLIANT", nil
		}
		return "NON_COMPLIANT", nil
	case "statefulset":
		statefulset, err := c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to get statefulset: %w", err)
		}
		if statefulset.Status.UpdatedReplicas == *statefulset.Spec.Replicas {
			return "COMPLIANT", nil
		}
		return "NON_COMPLIANT", nil
	default:
		return "", fmt.Errorf("unsupported workload kind: %s", kind)
	}
}

// RollbackDeployment rolls back a deployment to a previous revision.
func (c *KubernetesPlatformClient) RollbackDeployment(ctx context.Context, namespace, name string, revision int64) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	c.log.Info("rolling back deployment",
		"namespace", namespace,
		"name", name,
		"revision", revision,
	)

	deploymentsClient := c.clientset.AppsV1().Deployments(namespace)

	// Get current deployment
	deployment, err := deploymentsClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Get the replicasets to find the target revision
	replicaSetList, err := c.clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(deployment.Spec.Selector),
	})
	if err != nil {
		return fmt.Errorf("failed to list replicasets: %w", err)
	}

	// Find the target revision
	var targetRS *appsv1.ReplicaSet
	for i := range replicaSetList.Items {
		rs := &replicaSetList.Items[i]
		if rs.Annotations != nil {
			if rev, ok := rs.Annotations["deployment.kubernetes.io/revision"]; ok {
				if rev == fmt.Sprintf("%d", revision) {
					targetRS = rs
					break
				}
			}
		}
	}

	if targetRS == nil {
		return fmt.Errorf("revision %d not found", revision)
	}

	// Copy the pod template from the target replicaset
	deployment.Spec.Template = targetRS.Spec.Template

	// Update
	_, err = deploymentsClient.Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to rollback deployment: %w", err)
	}

	return c.waitForDeploymentRollout(ctx, namespace, name)
}

// ScaleWorkload scales a workload to the specified number of replicas.
func (c *KubernetesPlatformClient) ScaleWorkload(ctx context.Context, instanceID string, replicas int32) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	namespace, kind, name, err := c.parseInstanceID(instanceID)
	if err != nil {
		return err
	}

	c.log.Info("scaling workload",
		"namespace", namespace,
		"kind", kind,
		"name", name,
		"replicas", replicas,
	)

	switch strings.ToLower(kind) {
	case "deployment":
		deployment, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		deployment.Spec.Replicas = &replicas
		_, err = c.clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
		return err
	case "statefulset":
		statefulset, err := c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		statefulset.Spec.Replicas = &replicas
		_, err = c.clientset.AppsV1().StatefulSets(namespace).Update(ctx, statefulset, metav1.UpdateOptions{})
		return err
	default:
		return fmt.Errorf("cannot scale workload of kind: %s", kind)
	}
}

// GetPods returns the pods for a workload.
func (c *KubernetesPlatformClient) GetPods(ctx context.Context, instanceID string) ([]corev1.Pod, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	namespace, kind, name, err := c.parseInstanceID(instanceID)
	if err != nil {
		return nil, err
	}

	var labelSelector string

	switch strings.ToLower(kind) {
	case "deployment":
		deployment, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		labelSelector = metav1.FormatLabelSelector(deployment.Spec.Selector)
	case "daemonset":
		daemonset, err := c.clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		labelSelector = metav1.FormatLabelSelector(daemonset.Spec.Selector)
	case "statefulset":
		statefulset, err := c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		labelSelector = metav1.FormatLabelSelector(statefulset.Spec.Selector)
	default:
		return nil, fmt.Errorf("unsupported workload kind: %s", kind)
	}

	podList, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

// GetPatchComplianceData retrieves detailed patch/rollout compliance data for a Kubernetes workload.
func (c *KubernetesPlatformClient) GetPatchComplianceData(ctx context.Context, instanceID string) (interface{}, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	namespace, kind, name, err := c.parseInstanceID(instanceID)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"instance_id": instanceID,
		"namespace":   namespace,
		"kind":        kind,
		"name":        name,
		"status":      "UNKNOWN",
	}

	switch strings.ToLower(kind) {
	case "deployment":
		deployment, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get deployment: %w", err)
		}

		result["generation"] = deployment.Generation
		result["observed_generation"] = deployment.Status.ObservedGeneration
		result["replicas_desired"] = *deployment.Spec.Replicas
		result["replicas_ready"] = deployment.Status.ReadyReplicas
		result["replicas_available"] = deployment.Status.AvailableReplicas
		result["replicas_updated"] = deployment.Status.UpdatedReplicas

		// Check conditions
		var conditions []map[string]interface{}
		for _, cond := range deployment.Status.Conditions {
			conditions = append(conditions, map[string]interface{}{
				"type":    string(cond.Type),
				"status":  string(cond.Status),
				"reason":  cond.Reason,
				"message": cond.Message,
			})

			// Determine compliance
			if cond.Type == appsv1.DeploymentAvailable {
				if cond.Status == corev1.ConditionTrue {
					result["status"] = "COMPLIANT"
				} else {
					result["status"] = "NON_COMPLIANT"
				}
			}
		}
		result["conditions"] = conditions

		// Extract current images
		var images []string
		for _, container := range deployment.Spec.Template.Spec.Containers {
			images = append(images, container.Image)
		}
		result["images"] = images

	case "daemonset":
		ds, err := c.clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get daemonset: %w", err)
		}

		result["generation"] = ds.Generation
		result["observed_generation"] = ds.Status.ObservedGeneration
		result["desired_number_scheduled"] = ds.Status.DesiredNumberScheduled
		result["current_number_scheduled"] = ds.Status.CurrentNumberScheduled
		result["number_ready"] = ds.Status.NumberReady
		result["number_available"] = ds.Status.NumberAvailable
		result["updated_number_scheduled"] = ds.Status.UpdatedNumberScheduled

		// Determine compliance
		if ds.Status.NumberReady == ds.Status.DesiredNumberScheduled &&
			ds.Status.UpdatedNumberScheduled == ds.Status.DesiredNumberScheduled {
			result["status"] = "COMPLIANT"
		} else if ds.Status.NumberReady < ds.Status.DesiredNumberScheduled {
			result["status"] = "NON_COMPLIANT"
		}

		// Extract current images
		var images []string
		for _, container := range ds.Spec.Template.Spec.Containers {
			images = append(images, container.Image)
		}
		result["images"] = images

	case "statefulset":
		ss, err := c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get statefulset: %w", err)
		}

		result["generation"] = ss.Generation
		result["observed_generation"] = ss.Status.ObservedGeneration
		result["replicas_desired"] = *ss.Spec.Replicas
		result["replicas_ready"] = ss.Status.ReadyReplicas
		result["replicas_current"] = ss.Status.CurrentReplicas
		result["replicas_updated"] = ss.Status.UpdatedReplicas

		// Check conditions
		var conditions []map[string]interface{}
		for _, cond := range ss.Status.Conditions {
			conditions = append(conditions, map[string]interface{}{
				"type":    string(cond.Type),
				"status":  string(cond.Status),
				"reason":  cond.Reason,
				"message": cond.Message,
			})
		}
		result["conditions"] = conditions

		// Determine compliance
		if ss.Status.ReadyReplicas == *ss.Spec.Replicas &&
			ss.Status.UpdatedReplicas == *ss.Spec.Replicas {
			result["status"] = "COMPLIANT"
		} else if ss.Status.ReadyReplicas < *ss.Spec.Replicas {
			result["status"] = "NON_COMPLIANT"
		}

		// Extract current images
		var images []string
		for _, container := range ss.Spec.Template.Spec.Containers {
			images = append(images, container.Image)
		}
		result["images"] = images

	default:
		return nil, fmt.Errorf("unsupported kind: %s", kind)
	}

	return result, nil
}

// parseInstanceID parses a Kubernetes instance ID.
// Format: "namespace/kind/name" or "kind/name" (uses default namespace)
func (c *KubernetesPlatformClient) parseInstanceID(instanceID string) (namespace, kind, name string, err error) {
	parts := strings.Split(instanceID, "/")

	switch len(parts) {
	case 3:
		return parts[0], parts[1], parts[2], nil
	case 2:
		return c.cfg.DefaultNS, parts[0], parts[1], nil
	default:
		return "", "", "", fmt.Errorf("invalid instance ID format (expected namespace/kind/name or kind/name): %s", instanceID)
	}
}

// parseImageSpec parses an image specification.
// Format: "container-name=image:tag" or just "image:tag"
func parseImageSpec(spec string) (containerName, image string) {
	if idx := strings.Index(spec, "="); idx > 0 {
		return spec[:idx], spec[idx+1:]
	}
	return "", spec
}
