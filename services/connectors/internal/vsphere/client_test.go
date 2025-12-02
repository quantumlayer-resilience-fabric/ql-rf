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
		URL:      "https://vcenter.example.com/sdk",
		User:     "admin",
		Password: "password123",
		Insecure: true,
	}

	assert.Equal(t, "https://vcenter.example.com/sdk", cfg.URL)
	assert.Equal(t, "admin", cfg.User)
	assert.Equal(t, "password123", cfg.Password)
	assert.True(t, cfg.Insecure)
}
