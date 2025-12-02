package gcp

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
	c := New(Config{ProjectID: "test-project"}, newTestLogger())
	assert.Equal(t, "gcp", c.Name())
}

func TestConnector_Platform(t *testing.T) {
	c := New(Config{ProjectID: "test-project"}, newTestLogger())
	assert.Equal(t, models.PlatformGCP, c.Platform())
}

func TestExtractZoneFromKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "standard zone key",
			key:      "zones/us-central1-a",
			expected: "us-central1-a",
		},
		{
			name:     "zone key with prefix",
			key:      "projects/my-project/zones/us-east1-b",
			expected: "us-east1-b",
		},
		{
			name:     "simple zone name",
			key:      "us-west1-c",
			expected: "us-west1-c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractZoneFromKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractRegionFromZone(t *testing.T) {
	tests := []struct {
		name     string
		zone     string
		expected string
	}{
		{
			name:     "us-central1-a",
			zone:     "us-central1-a",
			expected: "us-central1",
		},
		{
			name:     "us-east1-b",
			zone:     "us-east1-b",
			expected: "us-east1",
		},
		{
			name:     "europe-west1-c",
			zone:     "europe-west1-c",
			expected: "europe-west1",
		},
		{
			name:     "asia-northeast1-a",
			zone:     "asia-northeast1-a",
			expected: "asia-northeast1",
		},
		{
			name:     "no zone suffix - returns truncated",
			zone:     "us-central1",
			expected: "us", // function truncates at last dash
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRegionFromZone(tt.zone)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractResourceName(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "disk URL",
			url:      "https://www.googleapis.com/compute/v1/projects/my-project/zones/us-central1-a/disks/boot-disk",
			expected: "boot-disk",
		},
		{
			name:     "instance URL",
			url:      "https://compute.googleapis.com/compute/v1/projects/my-project/zones/us-east1-b/instances/my-instance",
			expected: "my-instance",
		},
		{
			name:     "simple name",
			url:      "my-resource",
			expected: "my-resource",
		},
		{
			name:     "empty string",
			url:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractResourceName(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPtrUint32(t *testing.T) {
	val := ptrUint32(42)
	assert.NotNil(t, val)
	assert.Equal(t, uint32(42), *val)
}

func TestConnector_NotConnected(t *testing.T) {
	c := New(Config{ProjectID: "test-project"}, newTestLogger())

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
	c := New(Config{ProjectID: "test-project"}, newTestLogger())
	c.connected = true

	err := c.Close()
	assert.NoError(t, err)
	assert.False(t, c.connected)
}
