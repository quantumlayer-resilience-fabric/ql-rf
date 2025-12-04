package executor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

func TestNewVSpherePlatformClient(t *testing.T) {
	log := logger.New("error", "text")

	tests := []struct {
		name string
		cfg  VSphereConfig
	}{
		{
			name: "minimal config",
			cfg: VSphereConfig{
				URL:      "https://vcenter.example.com/sdk",
				Username: "admin@vsphere.local",
				Password: "password",
			},
		},
		{
			name: "full config",
			cfg: VSphereConfig{
				URL:              "https://vcenter.example.com/sdk",
				Username:         "admin@vsphere.local",
				Password:         "password",
				Insecure:         true,
				Datacenter:       "DC1",
				GuestUsername:    "root",
				GuestPassword:    "guestpass",
				ConnectTimeout:   60 * time.Second,
				OperationTimeout: 30 * time.Minute,
			},
		},
		{
			name: "with empty optional fields",
			cfg: VSphereConfig{
				URL:      "https://vcenter.example.com/sdk",
				Username: "admin@vsphere.local",
				Password: "password",
				Insecure: false,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := NewVSpherePlatformClient(tc.cfg, log)
			require.NotNil(t, client)

			// Verify default timeouts are set
			assert.GreaterOrEqual(t, client.cfg.ConnectTimeout, 30*time.Second)
			assert.GreaterOrEqual(t, client.cfg.OperationTimeout, 10*time.Minute)
		})
	}
}

func TestVSpherePlatformClient_DefaultTimeouts(t *testing.T) {
	log := logger.New("error", "text")

	// Test that default timeouts are applied
	client := NewVSpherePlatformClient(VSphereConfig{
		URL:      "https://vcenter.example.com/sdk",
		Username: "admin",
		Password: "pass",
	}, log)

	assert.Equal(t, 30*time.Second, client.cfg.ConnectTimeout)
	assert.Equal(t, 10*time.Minute, client.cfg.OperationTimeout)
}

func TestVSpherePlatformClient_CustomTimeouts(t *testing.T) {
	log := logger.New("error", "text")

	// Test that custom timeouts are preserved
	client := NewVSpherePlatformClient(VSphereConfig{
		URL:              "https://vcenter.example.com/sdk",
		Username:         "admin",
		Password:         "pass",
		ConnectTimeout:   60 * time.Second,
		OperationTimeout: 30 * time.Minute,
	}, log)

	assert.Equal(t, 60*time.Second, client.cfg.ConnectTimeout)
	assert.Equal(t, 30*time.Minute, client.cfg.OperationTimeout)
}

func TestVSpherePlatformClient_ConnectRequiresURL(t *testing.T) {
	log := logger.New("error", "text")

	// Test that Connect fails gracefully without a URL
	client := NewVSpherePlatformClient(VSphereConfig{
		URL:      "", // Empty URL
		Username: "admin",
		Password: "pass",
	}, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	assert.Error(t, err, "Connect should fail with empty URL")
}

func TestVSpherePlatformClient_ConnectRequiresCredentials(t *testing.T) {
	log := logger.New("error", "text")

	// Test that Connect fails gracefully without credentials
	client := NewVSpherePlatformClient(VSphereConfig{
		URL:      "https://vcenter.example.com/sdk",
		Username: "", // Empty username
		Password: "",
	}, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	assert.Error(t, err, "Connect should fail with empty credentials")
}

func TestVSpherePlatformClient_MethodsRequireConnection(t *testing.T) {
	log := logger.New("error", "text")

	// Create client without connecting
	client := NewVSpherePlatformClient(VSphereConfig{
		URL:      "https://vcenter.example.com/sdk",
		Username: "admin",
		Password: "pass",
	}, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// All methods should fail gracefully when not connected
	t.Run("ReimageInstance", func(t *testing.T) {
		err := client.ReimageInstance(ctx, "vm-123", "template-456")
		assert.Error(t, err)
	})

	t.Run("RebootInstance", func(t *testing.T) {
		err := client.RebootInstance(ctx, "vm-123")
		assert.Error(t, err)
	})

	t.Run("TerminateInstance", func(t *testing.T) {
		err := client.TerminateInstance(ctx, "vm-123")
		assert.Error(t, err)
	})

	t.Run("GetInstanceStatus", func(t *testing.T) {
		_, err := client.GetInstanceStatus(ctx, "vm-123")
		assert.Error(t, err)
	})

	t.Run("ApplyPatches", func(t *testing.T) {
		err := client.ApplyPatches(ctx, "vm-123", nil)
		assert.Error(t, err)
	})

	t.Run("GetPatchStatus", func(t *testing.T) {
		_, err := client.GetPatchStatus(ctx, "vm-123")
		assert.Error(t, err)
	})

	t.Run("GetPatchComplianceData", func(t *testing.T) {
		_, err := client.GetPatchComplianceData(ctx, "vm-123")
		assert.Error(t, err)
	})
}

func TestVSpherePlatformClient_WaitForInstanceStateTimeout(t *testing.T) {
	log := logger.New("error", "text")

	client := NewVSpherePlatformClient(VSphereConfig{
		URL:      "https://vcenter.example.com/sdk",
		Username: "admin",
		Password: "pass",
	}, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Should fail because not connected
	err := client.WaitForInstanceState(ctx, "vm-123", "poweredOn", 1*time.Second)
	assert.Error(t, err)
}

// VSphereConfigValidation tests configuration validation
func TestVSphereConfig_Validation(t *testing.T) {
	tests := []struct {
		name      string
		cfg       VSphereConfig
		expectErr bool
	}{
		{
			name: "valid config",
			cfg: VSphereConfig{
				URL:      "https://vcenter.example.com/sdk",
				Username: "admin@vsphere.local",
				Password: "password123",
			},
			expectErr: false,
		},
		{
			name: "missing URL",
			cfg: VSphereConfig{
				Username: "admin@vsphere.local",
				Password: "password123",
			},
			expectErr: true,
		},
		{
			name: "missing username",
			cfg: VSphereConfig{
				URL:      "https://vcenter.example.com/sdk",
				Password: "password123",
			},
			expectErr: true,
		},
		{
			name: "missing password",
			cfg: VSphereConfig{
				URL:      "https://vcenter.example.com/sdk",
				Username: "admin@vsphere.local",
			},
			expectErr: true,
		},
	}

	log := logger.New("error", "text")

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := NewVSpherePlatformClient(tc.cfg, log)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := client.Connect(ctx)

			if tc.expectErr {
				assert.Error(t, err)
			}
			// Note: For valid configs, we can't test Connect() success without a real vCenter
		})
	}
}

// TestVSpherePatchParams tests patch parameter parsing
func TestVSpherePatchParams(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]interface{}
	}{
		{
			name:   "nil params",
			params: nil,
		},
		{
			name:   "empty params",
			params: map[string]interface{}{},
		},
		{
			name: "windows params",
			params: map[string]interface{}{
				"os_type":       "windows",
				"reboot_policy": "if_needed",
				"categories":    []string{"Security", "Critical"},
			},
		},
		{
			name: "linux params",
			params: map[string]interface{}{
				"os_type":       "linux",
				"reboot_policy": "always",
				"packages":      []string{"kernel", "openssl"},
			},
		},
	}

	log := logger.New("error", "text")
	client := NewVSpherePlatformClient(VSphereConfig{
		URL:      "https://vcenter.example.com/sdk",
		Username: "admin",
		Password: "pass",
	}, log)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Just verify the params are acceptable (client not connected, so will fail)
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			err := client.ApplyPatches(ctx, "vm-123", tc.params)
			// Will fail due to not connected, but shouldn't panic
			assert.Error(t, err)
		})
	}
}

// BenchmarkVSpherePlatformClientCreation benchmarks client creation
func BenchmarkVSpherePlatformClientCreation(b *testing.B) {
	log := logger.New("error", "text")
	cfg := VSphereConfig{
		URL:      "https://vcenter.example.com/sdk",
		Username: "admin@vsphere.local",
		Password: "password",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewVSpherePlatformClient(cfg, log)
	}
}
