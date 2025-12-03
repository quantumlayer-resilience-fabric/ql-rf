package vsphere

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

func newTestLogger() *logger.Logger {
	return logger.New("error", "text")
}

func TestConnector_Name(t *testing.T) {
	c := New(Config{URL: "https://vcenter.example.com/sdk"}, newTestLogger())
	assert.Equal(t, "vsphere", c.Name())
}

func TestConnector_Platform(t *testing.T) {
	c := New(Config{URL: "https://vcenter.example.com/sdk"}, newTestLogger())
	assert.Equal(t, models.PlatformVSphere, c.Platform())
}

func TestExtractHostFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "standard vCenter URL",
			url:      "https://vcenter.example.com/sdk",
			expected: "vcenter.example.com",
		},
		{
			name:     "vCenter with port",
			url:      "https://vcenter.example.com:443/sdk",
			expected: "vcenter.example.com:443",
		},
		{
			name:     "IP address",
			url:      "https://192.168.1.100/sdk",
			expected: "192.168.1.100",
		},
		{
			name:     "IP with port",
			url:      "https://10.0.0.50:443/sdk",
			expected: "10.0.0.50:443",
		},
		{
			name:     "invalid URL returns as-is",
			url:      "not-a-valid-url",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHostFromURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConnector_NotConnected(t *testing.T) {
	c := New(Config{URL: "https://vcenter.example.com/sdk"}, newTestLogger())

	t.Run("DiscoverAssets returns error when not connected", func(t *testing.T) {
		_, err := c.DiscoverAssets(nil, [16]byte{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})

	t.Run("DiscoverImages returns error when not connected", func(t *testing.T) {
		_, err := c.DiscoverImages(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})

	t.Run("Health returns error when not connected", func(t *testing.T) {
		err := c.Health(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})
}

func TestConnector_Close(t *testing.T) {
	c := New(Config{URL: "https://vcenter.example.com/sdk"}, newTestLogger())
	// Simulate connected state without actual client
	c.connected = true

	err := c.Close()
	assert.NoError(t, err)
	assert.False(t, c.connected)
}

func TestConfig(t *testing.T) {
	cfg := Config{
		URL:         "https://vcenter.example.com/sdk",
		User:        "admin",
		Password:    "password123",
		Insecure:    true,
		Datacenters: []string{"DC1", "DC2"},
		Clusters:    []string{"Cluster1"},
	}

	assert.Equal(t, "https://vcenter.example.com/sdk", cfg.URL)
	assert.Equal(t, "admin", cfg.User)
	assert.Equal(t, "password123", cfg.Password)
	assert.True(t, cfg.Insecure)
	assert.Equal(t, []string{"DC1", "DC2"}, cfg.Datacenters)
	assert.Equal(t, []string{"Cluster1"}, cfg.Clusters)
}

func TestConnector_IsDatacenterAllowed(t *testing.T) {
	tests := []struct {
		name        string
		configDCs   []string
		checkDC     string
		expected    bool
	}{
		{
			name:      "empty filter allows all",
			configDCs: nil,
			checkDC:   "any-datacenter",
			expected:  true,
		},
		{
			name:      "exact match",
			configDCs: []string{"DC1", "DC2"},
			checkDC:   "DC1",
			expected:  true,
		},
		{
			name:      "case insensitive match",
			configDCs: []string{"Production-DC"},
			checkDC:   "production-dc",
			expected:  true,
		},
		{
			name:      "not in list",
			configDCs: []string{"DC1"},
			checkDC:   "DC2",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(Config{
				URL:         "https://vcenter.example.com/sdk",
				Datacenters: tt.configDCs,
			}, newTestLogger())
			result := c.isDatacenterAllowed(tt.checkDC)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConnector_IsClusterAllowed(t *testing.T) {
	tests := []struct {
		name           string
		configClusters []string
		checkCluster   string
		expected       bool
	}{
		{
			name:           "empty filter allows all",
			configClusters: nil,
			checkCluster:   "any-cluster",
			expected:       true,
		},
		{
			name:           "empty cluster with no filter",
			configClusters: nil,
			checkCluster:   "",
			expected:       true,
		},
		{
			name:           "exact match",
			configClusters: []string{"Production", "Development"},
			checkCluster:   "Production",
			expected:       true,
		},
		{
			name:           "case insensitive match",
			configClusters: []string{"Prod-Cluster"},
			checkCluster:   "prod-cluster",
			expected:       true,
		},
		{
			name:           "not in list",
			configClusters: []string{"Production"},
			checkCluster:   "Development",
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(Config{
				URL:      "https://vcenter.example.com/sdk",
				Clusters: tt.configClusters,
			}, newTestLogger())
			result := c.isClusterAllowed(tt.checkCluster)
			assert.Equal(t, tt.expected, result)
		})
	}
}
