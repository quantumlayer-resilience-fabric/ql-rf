package gcp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/connector"
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

func TestConnector_IsZoneAllowed(t *testing.T) {
	tests := []struct {
		name        string
		configZones []string
		checkZone   string
		expected    bool
	}{
		{
			name:        "empty filter allows all",
			configZones: nil,
			checkZone:   "us-central1-a",
			expected:    true,
		},
		{
			name:        "exact match",
			configZones: []string{"us-central1-a", "us-east1-b"},
			checkZone:   "us-central1-a",
			expected:    true,
		},
		{
			name:        "case insensitive match",
			configZones: []string{"US-CENTRAL1-A"},
			checkZone:   "us-central1-a",
			expected:    true,
		},
		{
			name:        "not in list",
			configZones: []string{"us-central1-a"},
			checkZone:   "us-east1-b",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(Config{
				ProjectID: "test-project",
				Zones:     tt.configZones,
			}, newTestLogger())
			result := c.isZoneAllowed(tt.checkZone)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractProjectFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "standard image URL",
			url:      "https://www.googleapis.com/compute/v1/projects/my-project/global/images/my-image",
			expected: "my-project",
		},
		{
			name:     "short path",
			url:      "projects/ubuntu-os-cloud/global/images/ubuntu-2004-focal-v20231101",
			expected: "ubuntu-os-cloud",
		},
		{
			name:     "family URL",
			url:      "projects/debian-cloud/global/images/family/debian-11",
			expected: "debian-cloud",
		},
		{
			name:     "no project",
			url:      "global/images/my-image",
			expected: "",
		},
		{
			name:     "empty string",
			url:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractProjectFromURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractVersionFromImageName(t *testing.T) {
	tests := []struct {
		name     string
		imgName  string
		expected string
	}{
		{
			name:     "ubuntu image with version",
			imgName:  "ubuntu-2004-focal-v20231101",
			expected: "v20231101",
		},
		{
			name:     "debian image with version",
			imgName:  "debian-11-bullseye-v20231010",
			expected: "v20231010",
		},
		{
			name:     "no version suffix",
			imgName:  "custom-image-base",
			expected: "",
		},
		{
			name:     "empty string",
			imgName:  "",
			expected: "",
		},
		{
			name:     "version at wrong position",
			imgName:  "v20231101-custom-image",
			expected: "v20231101",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVersionFromImageName(tt.imgName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConnector_ParseImageURL(t *testing.T) {
	c := New(Config{ProjectID: "my-project"}, newTestLogger())

	tests := []struct {
		name            string
		url             string
		expectedRef     string
		expectedVersion string
	}{
		{
			name:            "image family reference",
			url:             "projects/debian-cloud/global/images/family/debian-11",
			expectedRef:     "debian-cloud/family/debian-11",
			expectedVersion: "latest",
		},
		{
			name:            "specific image from another project",
			url:             "projects/ubuntu-os-cloud/global/images/ubuntu-2004-focal-v20231101",
			expectedRef:     "ubuntu-os-cloud/ubuntu-2004-focal-v20231101",
			expectedVersion: "v20231101",
		},
		{
			name:            "image from same project",
			url:             "projects/my-project/global/images/custom-image-v20231015",
			expectedRef:     "custom-image-v20231015",
			expectedVersion: "v20231015",
		},
		{
			name:            "empty URL",
			url:             "",
			expectedRef:     "",
			expectedVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, version := c.parseImageURL(tt.url)
			assert.Equal(t, tt.expectedRef, ref)
			assert.Equal(t, tt.expectedVersion, version)
		})
	}
}

func TestConnector_GroupImagesByFamily(t *testing.T) {
	c := New(Config{ProjectID: "test-project"}, newTestLogger())

	images := []connector.ImageInfo{
		{
			Name:      "my-app-v1",
			CreatedAt: "2023-10-01T00:00:00Z",
			Tags:      map[string]string{"family": "my-app"},
		},
		{
			Name:      "my-app-v2",
			CreatedAt: "2023-11-01T00:00:00Z",
			Tags:      map[string]string{"family": "my-app"},
		},
		{
			Name:      "other-app-v1",
			CreatedAt: "2023-10-15T00:00:00Z",
			Tags:      map[string]string{"family": "other-app"},
		},
		{
			Name:      "no-family-image",
			CreatedAt: "2023-12-01T00:00:00Z",
			Tags:      map[string]string{},
		},
	}

	result := c.groupImagesByFamily(images)

	assert.Len(t, result, 2)
	assert.Equal(t, "my-app-v2", result["my-app"].Name)     // Latest in my-app family
	assert.Equal(t, "other-app-v1", result["other-app"].Name) // Only one in other-app
	_, hasNoFamily := result[""]
	assert.False(t, hasNoFamily) // Images without family should not be included
}
