package azure

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
