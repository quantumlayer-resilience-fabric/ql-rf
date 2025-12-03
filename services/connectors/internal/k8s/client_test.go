package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	fakeclientset "k8s.io/client-go/kubernetes/fake"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

func TestParseContainerImage(t *testing.T) {
	tests := []struct {
		name            string
		image           string
		expectedRef     string
		expectedVersion string
	}{
		{
			name:            "simple image with tag",
			image:           "nginx:1.19",
			expectedRef:     "nginx",
			expectedVersion: "1.19",
		},
		{
			name:            "image without tag",
			image:           "nginx",
			expectedRef:     "nginx",
			expectedVersion: "latest",
		},
		{
			name:            "full registry path with tag",
			image:           "registry.example.com/nginx:1.19",
			expectedRef:     "registry.example.com/nginx",
			expectedVersion: "1.19",
		},
		{
			name:            "registry with port and tag",
			image:           "registry.example.com:5000/nginx:1.19",
			expectedRef:     "registry.example.com:5000/nginx",
			expectedVersion: "1.19",
		},
		{
			name:            "image with digest",
			image:           "nginx@sha256:abc123def456",
			expectedRef:     "nginx",
			expectedVersion: "sha256:abc123def456",
		},
		{
			name:            "gcr image",
			image:           "gcr.io/my-project/my-app:v1.0.0",
			expectedRef:     "gcr.io/my-project/my-app",
			expectedVersion: "v1.0.0",
		},
		{
			name:            "ecr image",
			image:           "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app:latest",
			expectedRef:     "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app",
			expectedVersion: "latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, version := parseContainerImage(tt.image)
			if ref != tt.expectedRef {
				t.Errorf("parseContainerImage(%q) ref = %q, want %q", tt.image, ref, tt.expectedRef)
			}
			if version != tt.expectedVersion {
				t.Errorf("parseContainerImage(%q) version = %q, want %q", tt.image, version, tt.expectedVersion)
			}
		})
	}
}

func TestConnector_Name(t *testing.T) {
	log := logger.New("debug", "json")
	c := New(Config{}, log)

	if c.Name() != "k8s" {
		t.Errorf("Name() = %q, want %q", c.Name(), "k8s")
	}
}

func TestConnector_Platform(t *testing.T) {
	log := logger.New("debug", "json")
	c := New(Config{}, log)

	if c.Platform() != models.PlatformK8s {
		t.Errorf("Platform() = %q, want %q", c.Platform(), models.PlatformK8s)
	}
}

func TestConnector_NormalizePod(t *testing.T) {
	log := logger.New("debug", "json")
	c := &Connector{
		cfg:         Config{},
		log:         log.WithComponent("k8s-connector"),
		clusterName: "test-cluster",
	}

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-123"),
			Labels: map[string]string{
				"app":  "my-app",
				"tier": "backend",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "ReplicaSet",
					Name: "my-deployment-abc123",
				},
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
			Containers: []corev1.Container{
				{
					Name:  "main",
					Image: "my-app:v1.2.3",
				},
				{
					Name:  "sidecar",
					Image: "envoy:1.20",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	deploymentMap := map[string]deploymentInfo{
		"default/my-deployment": {
			Name:     "my-deployment",
			Replicas: 3,
		},
	}

	asset := c.normalizePod(pod, deploymentMap)

	// Check basic fields
	if asset.Platform != models.PlatformK8s {
		t.Errorf("Platform = %q, want %q", asset.Platform, models.PlatformK8s)
	}
	if asset.Account != "test-cluster" {
		t.Errorf("Account = %q, want %q", asset.Account, "test-cluster")
	}
	if asset.Region != "default" {
		t.Errorf("Region = %q, want %q", asset.Region, "default")
	}
	if asset.InstanceID != "test-uid-123" {
		t.Errorf("InstanceID = %q, want %q", asset.InstanceID, "test-uid-123")
	}
	if asset.Name != "test-pod" {
		t.Errorf("Name = %q, want %q", asset.Name, "test-pod")
	}
	if asset.ImageRef != "my-app" {
		t.Errorf("ImageRef = %q, want %q", asset.ImageRef, "my-app")
	}
	if asset.ImageVersion != "v1.2.3" {
		t.Errorf("ImageVersion = %q, want %q", asset.ImageVersion, "v1.2.3")
	}
	if asset.State != models.AssetStateRunning {
		t.Errorf("State = %q, want %q", asset.State, models.AssetStateRunning)
	}

	// Check tags
	if asset.Tags["namespace"] != "default" {
		t.Errorf("Tags[namespace] = %q, want %q", asset.Tags["namespace"], "default")
	}
	if asset.Tags["node"] != "node-1" {
		t.Errorf("Tags[node] = %q, want %q", asset.Tags["node"], "node-1")
	}
	if asset.Tags["label:app"] != "my-app" {
		t.Errorf("Tags[label:app] = %q, want %q", asset.Tags["label:app"], "my-app")
	}
	if asset.Tags["deployment"] != "my-deployment" {
		t.Errorf("Tags[deployment] = %q, want %q", asset.Tags["deployment"], "my-deployment")
	}
}

func TestConnector_NormalizeNode(t *testing.T) {
	log := logger.New("debug", "json")
	c := &Connector{
		cfg:         Config{},
		log:         log.WithComponent("k8s-connector"),
		clusterName: "test-cluster",
	}

	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-1",
			UID:  types.UID("node-uid-123"),
			Labels: map[string]string{
				"topology.kubernetes.io/zone":   "us-east-1a",
				"node.kubernetes.io/instance-type": "m5.large",
			},
		},
		Status: corev1.NodeStatus{
			NodeInfo: corev1.NodeSystemInfo{
				OSImage:                 "Ubuntu 20.04.3 LTS",
				KubeletVersion:          "v1.25.0",
				Architecture:            "amd64",
				OperatingSystem:         "linux",
				KernelVersion:           "5.4.0-91-generic",
				ContainerRuntimeVersion: "containerd://1.5.9",
			},
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	asset := c.normalizeNode(node)

	// Check basic fields
	if asset.Platform != models.PlatformK8s {
		t.Errorf("Platform = %q, want %q", asset.Platform, models.PlatformK8s)
	}
	if asset.Account != "test-cluster" {
		t.Errorf("Account = %q, want %q", asset.Account, "test-cluster")
	}
	if asset.Region != "us-east-1a" {
		t.Errorf("Region = %q, want %q", asset.Region, "us-east-1a")
	}
	if asset.Name != "node-1" {
		t.Errorf("Name = %q, want %q", asset.Name, "node-1")
	}
	if asset.ImageRef != "Ubuntu 20.04.3 LTS" {
		t.Errorf("ImageRef = %q, want %q", asset.ImageRef, "Ubuntu 20.04.3 LTS")
	}
	if asset.ImageVersion != "v1.25.0" {
		t.Errorf("ImageVersion = %q, want %q", asset.ImageVersion, "v1.25.0")
	}
	if asset.State != models.AssetStateRunning {
		t.Errorf("State = %q, want %q", asset.State, models.AssetStateRunning)
	}

	// Check tags
	if asset.Tags["type"] != "node" {
		t.Errorf("Tags[type] = %q, want %q", asset.Tags["type"], "node")
	}
	if asset.Tags["arch"] != "amd64" {
		t.Errorf("Tags[arch] = %q, want %q", asset.Tags["arch"], "amd64")
	}
}

func TestConnector_DiscoverAssets_WithFakeClient(t *testing.T) {
	log := logger.New("debug", "json")

	// Create fake k8s objects
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-server-1",
			Namespace: "production",
			UID:       types.UID("pod-1-uid"),
			Labels: map[string]string{
				"app": "web-server",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:1.21",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-server-1",
			Namespace: "production",
			UID:       types.UID("pod-2-uid"),
			Labels: map[string]string{
				"app": "api-server",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "api",
					Image: "my-api:v2.0.0",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	// Pod in excluded namespace - should be filtered out
	systemPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kube-proxy-abc",
			Namespace: "kube-system",
			UID:       types.UID("system-pod-uid"),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "kube-proxy",
					Image: "k8s.gcr.io/kube-proxy:v1.25.0",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	node1 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "worker-1",
			UID:  types.UID("node-1-uid"),
		},
		Status: corev1.NodeStatus{
			NodeInfo: corev1.NodeSystemInfo{
				OSImage:        "Ubuntu 20.04",
				KubeletVersion: "v1.25.0",
				Architecture:   "amd64",
				OperatingSystem: "linux",
			},
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	ns1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "production",
		},
	}

	ns2 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kube-system",
		},
	}

	// Create fake client
	objects := []runtime.Object{pod1, pod2, systemPod, node1, ns1, ns2}
	fakeClient := fakeclientset.NewSimpleClientset(objects...)

	c := &Connector{
		cfg: Config{
			ExcludeNamespaces:   []string{"kube-system", "kube-public"},
			DiscoverNodes:       true,
			DiscoverDeployments: false,
		},
		client:      fakeClient,
		log:         log.WithComponent("k8s-connector"),
		connected:   true,
		clusterName: "test-cluster",
	}

	ctx := context.Background()
	assets, err := c.DiscoverAssets(ctx, [16]byte{})
	if err != nil {
		t.Fatalf("DiscoverAssets() error = %v", err)
	}

	// Should have 2 pods (production namespace) + 1 node = 3 assets
	// The kube-system pod should be filtered out
	if len(assets) != 3 {
		t.Errorf("DiscoverAssets() returned %d assets, want 3", len(assets))
	}

	// Verify we have the expected assets
	foundPod1, foundPod2, foundNode := false, false, false
	for _, asset := range assets {
		switch asset.Name {
		case "web-server-1":
			foundPod1 = true
			if asset.ImageVersion != "1.21" {
				t.Errorf("web-server-1 ImageVersion = %q, want %q", asset.ImageVersion, "1.21")
			}
		case "api-server-1":
			foundPod2 = true
			if asset.ImageVersion != "v2.0.0" {
				t.Errorf("api-server-1 ImageVersion = %q, want %q", asset.ImageVersion, "v2.0.0")
			}
		case "worker-1":
			foundNode = true
			if asset.Tags["type"] != "node" {
				t.Errorf("worker-1 should have type=node tag")
			}
		}
	}

	if !foundPod1 {
		t.Error("web-server-1 pod not found in assets")
	}
	if !foundPod2 {
		t.Error("api-server-1 pod not found in assets")
	}
	if !foundNode {
		t.Error("worker-1 node not found in assets")
	}
}

func TestConnector_GetNamespacesToScan(t *testing.T) {
	log := logger.New("debug", "json")

	ns1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "default"},
	}
	ns2 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "production"},
	}
	ns3 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "kube-system"},
	}

	fakeClient := fakeclientset.NewSimpleClientset(ns1, ns2, ns3)

	tests := []struct {
		name              string
		configNamespaces  []string
		excludeNamespaces []string
		want              []string
	}{
		{
			name:              "specific namespaces configured",
			configNamespaces:  []string{"production"},
			excludeNamespaces: []string{},
			want:              []string{"production"},
		},
		{
			name:              "all namespaces with exclusions",
			configNamespaces:  []string{},
			excludeNamespaces: []string{"kube-system"},
			want:              []string{"default", "production"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Connector{
				cfg: Config{
					Namespaces:        tt.configNamespaces,
					ExcludeNamespaces: tt.excludeNamespaces,
				},
				client:    fakeClient,
				log:       log.WithComponent("k8s-connector"),
				connected: true,
			}

			got, err := c.getNamespacesToScan(context.Background())
			if err != nil {
				t.Fatalf("getNamespacesToScan() error = %v", err)
			}

			if len(got) != len(tt.want) {
				t.Errorf("getNamespacesToScan() returned %d namespaces, want %d", len(got), len(tt.want))
			}
		})
	}
}
