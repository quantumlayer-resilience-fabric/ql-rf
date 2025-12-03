package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssetProcessor_ProcessAsset(t *testing.T) {
	log := logger.New("debug", "text")
	processor := NewAssetProcessor(nil, log)

	// Register mock client
	mockClient := NewMockPlatformClient()
	processor.RegisterPlatformClient(models.PlatformAWS, mockClient)

	t.Run("validate action - success", func(t *testing.T) {
		asset := &AssetInfo{
			ID:         "asset-1",
			Name:       "test-server",
			Platform:   models.PlatformAWS,
			InstanceID: "i-1234567890",
		}

		result, err := processor.ProcessAsset(context.Background(), asset, ActionValidate, nil)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "asset-1", result.AssetID)
		assert.Equal(t, ActionValidate, result.Action)
		assert.NotZero(t, result.Duration)
	})

	t.Run("reboot action - success", func(t *testing.T) {
		asset := &AssetInfo{
			ID:         "asset-2",
			Name:       "test-server-2",
			Platform:   models.PlatformAWS,
			InstanceID: "i-0987654321",
		}

		result, err := processor.ProcessAsset(context.Background(), asset, ActionReboot, nil)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "Successfully rebooted")
	})

	t.Run("terminate action - success", func(t *testing.T) {
		asset := &AssetInfo{
			ID:         "asset-3",
			Name:       "test-server-3",
			Platform:   models.PlatformAWS,
			InstanceID: "i-terminate",
		}

		result, err := processor.ProcessAsset(context.Background(), asset, ActionTerminate, nil)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "Successfully terminated")
	})

	t.Run("reimage action - success", func(t *testing.T) {
		asset := &AssetInfo{
			ID:           "asset-4",
			Name:         "test-server-4",
			Platform:     models.PlatformAWS,
			InstanceID:   "i-reimage",
			CurrentImage: "ami-old",
			TargetImage:  "ami-new",
		}

		result, err := processor.ProcessAsset(context.Background(), asset, ActionReimage, nil)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "Successfully reimaged")
		assert.Equal(t, "ami-new", result.Metadata["target_image"])
		assert.Equal(t, "ami-old", result.Metadata["current_image"])
	})

	t.Run("reimage action - with params", func(t *testing.T) {
		asset := &AssetInfo{
			ID:           "asset-5",
			Name:         "test-server-5",
			Platform:     models.PlatformAWS,
			InstanceID:   "i-reimage-2",
			CurrentImage: "ami-old",
		}

		params := map[string]interface{}{
			"target_image": "ami-custom",
		}

		result, err := processor.ProcessAsset(context.Background(), asset, ActionReimage, params)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "ami-custom", result.Metadata["target_image"])
	})

	t.Run("reimage action - missing target image", func(t *testing.T) {
		asset := &AssetInfo{
			ID:         "asset-6",
			Name:       "test-server-6",
			Platform:   models.PlatformAWS,
			InstanceID: "i-no-image",
		}

		result, err := processor.ProcessAsset(context.Background(), asset, ActionReimage, nil)
		require.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, err.Error(), "target image is required")
	})

	t.Run("update action - success", func(t *testing.T) {
		asset := &AssetInfo{
			ID:       "asset-7",
			Name:     "test-server-7",
			Platform: models.PlatformAWS,
		}

		params := map[string]interface{}{
			"tags": map[string]interface{}{
				"environment": "production",
			},
			"state": "active",
		}

		result, err := processor.ProcessAsset(context.Background(), asset, ActionUpdate, params)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "Updated asset")
	})

	t.Run("unsupported action", func(t *testing.T) {
		asset := &AssetInfo{
			ID:       "asset-8",
			Name:     "test-server-8",
			Platform: models.PlatformAWS,
		}

		result, err := processor.ProcessAsset(context.Background(), asset, "unsupported", nil)
		require.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, err.Error(), "unsupported action")
	})

	t.Run("unregistered platform", func(t *testing.T) {
		asset := &AssetInfo{
			ID:       "asset-9",
			Name:     "test-server-9",
			Platform: models.PlatformAzure,
		}

		result, err := processor.ProcessAsset(context.Background(), asset, ActionValidate, nil)
		require.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, err.Error(), "no platform client registered")
	})
}

func TestAssetProcessor_ProcessAsset_WithErrors(t *testing.T) {
	log := logger.New("debug", "text")
	processor := NewAssetProcessor(nil, log)

	t.Run("reboot failure", func(t *testing.T) {
		mockClient := &MockPlatformClient{
			RebootInstanceFunc: func(ctx context.Context, instanceID string) error {
				return errors.New("reboot failed: instance not found")
			},
		}
		processor.RegisterPlatformClient(models.PlatformAWS, mockClient)

		asset := &AssetInfo{
			ID:         "asset-err-1",
			Name:       "test-server",
			Platform:   models.PlatformAWS,
			InstanceID: "i-notfound",
		}

		result, err := processor.ProcessAsset(context.Background(), asset, ActionReboot, nil)
		require.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "reboot failed")
	})

	t.Run("instance not running for validate", func(t *testing.T) {
		mockClient := &MockPlatformClient{
			GetInstanceStatusFunc: func(ctx context.Context, instanceID string) (string, error) {
				return "stopped", nil
			},
		}
		processor.RegisterPlatformClient(models.PlatformAWS, mockClient)

		asset := &AssetInfo{
			ID:         "asset-err-2",
			Name:       "test-server",
			Platform:   models.PlatformAWS,
			InstanceID: "i-stopped",
		}

		result, err := processor.ProcessAsset(context.Background(), asset, ActionValidate, nil)
		require.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, err.Error(), "instance is not running")
	})

	t.Run("wait for state timeout", func(t *testing.T) {
		mockClient := &MockPlatformClient{
			WaitForInstanceStateFunc: func(ctx context.Context, instanceID, targetState string, timeout time.Duration) error {
				return errors.New("timeout waiting for instance")
			},
		}
		processor.RegisterPlatformClient(models.PlatformAWS, mockClient)

		asset := &AssetInfo{
			ID:         "asset-err-3",
			Name:       "test-server",
			Platform:   models.PlatformAWS,
			InstanceID: "i-timeout",
		}

		result, err := processor.ProcessAsset(context.Background(), asset, ActionReboot, nil)
		require.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, err.Error(), "did not return to running state")
	})
}

func TestAssetProcessor_Patch(t *testing.T) {
	log := logger.New("debug", "text")
	processor := NewAssetProcessor(nil, log)

	mockClient := NewMockPlatformClient()
	processor.RegisterPlatformClient(models.PlatformAWS, mockClient)

	t.Run("patch action - success", func(t *testing.T) {
		asset := &AssetInfo{
			ID:         "asset-patch-1",
			Name:       "test-server",
			Platform:   models.PlatformAWS,
			InstanceID: "i-patch",
		}

		params := map[string]interface{}{
			"patch_group": "security",
			"baseline_id": "pb-123",
		}

		result, err := processor.ProcessAsset(context.Background(), asset, ActionPatch, params)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.True(t, result.NeedsReboot)
		assert.Equal(t, "security", result.Metadata["patch_group"])
		assert.Equal(t, "pb-123", result.Metadata["baseline_id"])
	})

	t.Run("patch action - instance not running", func(t *testing.T) {
		mockClient := &MockPlatformClient{
			GetInstanceStatusFunc: func(ctx context.Context, instanceID string) (string, error) {
				return "stopped", nil
			},
		}
		processor.RegisterPlatformClient(models.PlatformAWS, mockClient)

		asset := &AssetInfo{
			ID:         "asset-patch-2",
			Name:       "test-server",
			Platform:   models.PlatformAWS,
			InstanceID: "i-stopped",
		}

		result, err := processor.ProcessAsset(context.Background(), asset, ActionPatch, nil)
		require.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, err.Error(), "instance is not running")
	})
}

func TestAssetProcessor_Validate_WithHealthChecks(t *testing.T) {
	log := logger.New("debug", "text")
	processor := NewAssetProcessor(nil, log)

	mockClient := NewMockPlatformClient()
	processor.RegisterPlatformClient(models.PlatformAWS, mockClient)

	t.Run("validate with health checks - success", func(t *testing.T) {
		asset := &AssetInfo{
			ID:         "asset-hc-1",
			Name:       "test-server",
			Platform:   models.PlatformAWS,
			InstanceID: "i-healthcheck",
		}

		params := map[string]interface{}{
			"health_checks": []interface{}{
				map[string]interface{}{
					"name":     "dns-check",
					"type":     "dns",
					"target":   "localhost",
					"expected": "",
				},
			},
		}

		result, err := processor.ProcessAsset(context.Background(), asset, ActionValidate, params)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, true, result.Metadata["healthcheck_dns-check"])
	})
}

func TestNewAssetProcessor(t *testing.T) {
	log := logger.New("debug", "text")
	processor := NewAssetProcessor(nil, log)

	assert.NotNil(t, processor)
	assert.NotNil(t, processor.platformClients)
	assert.NotNil(t, processor.healthChecker)
}

func TestAssetAction_Constants(t *testing.T) {
	assert.Equal(t, AssetAction("reimage"), ActionReimage)
	assert.Equal(t, AssetAction("reboot"), ActionReboot)
	assert.Equal(t, AssetAction("terminate"), ActionTerminate)
	assert.Equal(t, AssetAction("patch"), ActionPatch)
	assert.Equal(t, AssetAction("update"), ActionUpdate)
	assert.Equal(t, AssetAction("validate"), ActionValidate)
}
