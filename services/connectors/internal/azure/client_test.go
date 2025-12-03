package azure

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/stretchr/testify/assert"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

func newTestLogger() *logger.Logger {
	return logger.New("error", "text")
}

func TestConnector_Name(t *testing.T) {
	c := New(Config{SubscriptionID: "test-sub"}, newTestLogger())
	assert.Equal(t, "azure", c.Name())
}

func TestConnector_Platform(t *testing.T) {
	c := New(Config{SubscriptionID: "test-sub"}, newTestLogger())
	assert.Equal(t, models.PlatformAzure, c.Platform())
}

func TestExtractResourceGroupFromID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected string
	}{
		{
			name:     "standard VM ID",
			id:       "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm",
			expected: "my-rg",
		},
		{
			name:     "lowercase resourcegroups",
			id:       "/subscriptions/12345678-1234-1234-1234-123456789abc/resourcegroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm",
			expected: "my-rg",
		},
		{
			name:     "no resource group",
			id:       "/subscriptions/12345678-1234-1234-1234-123456789abc/providers/Microsoft.Compute/virtualMachines/my-vm",
			expected: "",
		},
		{
			name:     "empty string",
			id:       "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractResourceGroupFromID(tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractResourceName(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected string
	}{
		{
			name:     "VM resource ID",
			id:       "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm",
			expected: "my-vm",
		},
		{
			name:     "image resource ID",
			id:       "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.Compute/images/my-image",
			expected: "my-image",
		},
		{
			name:     "simple name",
			id:       "my-resource",
			expected: "my-resource",
		},
		{
			name:     "empty string",
			id:       "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractResourceName(tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPtrToString(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{
			name:     "non-nil string",
			input:    ptrString("hello"),
			expected: "hello",
		},
		{
			name:     "nil string",
			input:    nil,
			expected: "",
		},
		{
			name:     "empty string",
			input:    ptrString(""),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ptrToString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPtrString(t *testing.T) {
	result := ptrString("test")
	assert.NotNil(t, result)
	assert.Equal(t, "test", *result)
}

func TestConnector_NotConnected(t *testing.T) {
	c := New(Config{SubscriptionID: "test-sub"}, newTestLogger())

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
	c := New(Config{SubscriptionID: "test-sub"}, newTestLogger())
	c.connected = true

	err := c.Close()
	assert.NoError(t, err)
	assert.False(t, c.connected)
}

func TestConnector_ConnectMissingCredentials(t *testing.T) {
	c := New(Config{SubscriptionID: "test-sub"}, newTestLogger())

	err := c.Connect(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "credentials not configured")
}

func TestConnector_IsResourceGroupAllowed(t *testing.T) {
	tests := []struct {
		name          string
		configRGs     []string
		checkRG       string
		expected      bool
	}{
		{
			name:      "empty filter allows all",
			configRGs: nil,
			checkRG:   "any-rg",
			expected:  true,
		},
		{
			name:      "exact match",
			configRGs: []string{"my-rg", "other-rg"},
			checkRG:   "my-rg",
			expected:  true,
		},
		{
			name:      "case insensitive match",
			configRGs: []string{"My-RG"},
			checkRG:   "my-rg",
			expected:  true,
		},
		{
			name:      "not in list",
			configRGs: []string{"allowed-rg"},
			checkRG:   "blocked-rg",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(Config{
				SubscriptionID: "test-sub",
				ResourceGroups: tt.configRGs,
			}, newTestLogger())
			result := c.isResourceGroupAllowed(tt.checkRG)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConnector_ExtractImageReference(t *testing.T) {
	c := New(Config{SubscriptionID: "test-sub"}, newTestLogger())

	tests := []struct {
		name            string
		publisher       *string
		offer           *string
		sku             *string
		version         *string
		id              *string
		expectedRef     string
		expectedVersion string
	}{
		{
			name:            "marketplace image",
			publisher:       ptrString("Canonical"),
			offer:           ptrString("UbuntuServer"),
			sku:             ptrString("20_04-lts"),
			version:         ptrString("latest"),
			expectedRef:     "Canonical:UbuntuServer:20_04-lts",
			expectedVersion: "latest",
		},
		{
			name:            "gallery image with version",
			id:              ptrString("/subscriptions/xxx/resourceGroups/rg/providers/Microsoft.Compute/galleries/gallery/images/myimage/versions/1.0.0"),
			expectedRef:     "1.0.0",
			expectedVersion: "1.0.0",
		},
		{
			name:            "nil reference",
			expectedRef:     "",
			expectedVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var imgRef *armcompute.ImageReference
			if tt.publisher != nil || tt.id != nil {
				imgRef = &armcompute.ImageReference{
					Publisher: tt.publisher,
					Offer:     tt.offer,
					SKU:       tt.sku,
					Version:   tt.version,
					ID:        tt.id,
				}
			}
			ref, ver := c.extractImageReference(imgRef)
			assert.Equal(t, tt.expectedRef, ref)
			assert.Equal(t, tt.expectedVersion, ver)
		})
	}
}
