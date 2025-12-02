package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/service/mocks"
)

func TestAssetService_GetAsset(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockAssetRepository()
	svc := service.NewAssetService(repo)

	orgID := uuid.New()
	assetID := uuid.New()

	// Setup: Add an asset
	repo.AddAsset(&service.Asset{
		ID:         assetID,
		OrgID:      orgID,
		Platform:   "aws",
		InstanceID: "i-1234567890abcdef0",
		State:      "running",
		UpdatedAt:  time.Now(),
	})

	t.Run("success", func(t *testing.T) {
		result, err := svc.GetAsset(ctx, service.GetAssetInput{
			ID:    assetID,
			OrgID: orgID,
		})

		require.NoError(t, err)
		assert.Equal(t, assetID, result.ID)
		assert.Equal(t, "aws", result.Platform)
		assert.Equal(t, "running", result.State)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetAsset(ctx, service.GetAssetInput{
			ID:    uuid.New(), // Non-existent ID
			OrgID: orgID,
		})

		assert.ErrorIs(t, err, service.ErrNotFound)
	})

	t.Run("wrong org returns not found", func(t *testing.T) {
		_, err := svc.GetAsset(ctx, service.GetAssetInput{
			ID:    assetID,
			OrgID: uuid.New(), // Different org
		})

		assert.ErrorIs(t, err, service.ErrNotFound)
	})
}

func TestAssetService_ListAssets(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockAssetRepository()
	svc := service.NewAssetService(repo)

	orgID := uuid.New()
	otherOrgID := uuid.New()

	// Setup: Add assets for multiple orgs
	for i := 0; i < 25; i++ {
		platform := "aws"
		if i%2 == 0 {
			platform = "azure"
		}
		state := "running"
		if i%3 == 0 {
			state = "stopped"
		}
		repo.AddAsset(&service.Asset{
			ID:         uuid.New(),
			OrgID:      orgID,
			Platform:   platform,
			InstanceID: "i-" + uuid.New().String()[:8],
			State:      state,
			UpdatedAt:  time.Now(),
		})
	}

	// Add asset for different org
	repo.AddAsset(&service.Asset{
		ID:         uuid.New(),
		OrgID:      otherOrgID,
		Platform:   "gcp",
		InstanceID: "other-instance",
		State:      "running",
		UpdatedAt:  time.Now(),
	})

	t.Run("returns paginated results", func(t *testing.T) {
		result, err := svc.ListAssets(ctx, service.ListAssetsInput{
			OrgID:    orgID,
			Page:     1,
			PageSize: 10,
		})

		require.NoError(t, err)
		assert.Len(t, result.Assets, 10)
		assert.Equal(t, int64(25), result.Total)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 10, result.PageSize)
		assert.Equal(t, 3, result.TotalPages)
	})

	t.Run("second page", func(t *testing.T) {
		result, err := svc.ListAssets(ctx, service.ListAssetsInput{
			OrgID:    orgID,
			Page:     2,
			PageSize: 10,
		})

		require.NoError(t, err)
		assert.Len(t, result.Assets, 10)
		assert.Equal(t, 2, result.Page)
	})

	t.Run("applies defaults", func(t *testing.T) {
		result, err := svc.ListAssets(ctx, service.ListAssetsInput{
			OrgID: orgID,
		})

		require.NoError(t, err)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PageSize)
	})

	t.Run("filter by platform", func(t *testing.T) {
		platform := "aws"
		result, err := svc.ListAssets(ctx, service.ListAssetsInput{
			OrgID:    orgID,
			Platform: &platform,
			PageSize: 100,
		})

		require.NoError(t, err)
		for _, asset := range result.Assets {
			assert.Equal(t, "aws", asset.Platform)
		}
	})

	t.Run("filter by state", func(t *testing.T) {
		state := "running"
		result, err := svc.ListAssets(ctx, service.ListAssetsInput{
			OrgID:    orgID,
			State:    &state,
			PageSize: 100,
		})

		require.NoError(t, err)
		for _, asset := range result.Assets {
			assert.Equal(t, "running", asset.State)
		}
	})
}

func TestAssetService_GetAssetSummary(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockAssetRepository()
	svc := service.NewAssetService(repo)

	orgID := uuid.New()

	// Setup: Add assets with different states
	for i := 0; i < 10; i++ {
		repo.AddAsset(&service.Asset{
			ID:         uuid.New(),
			OrgID:      orgID,
			Platform:   "aws",
			InstanceID: "running-" + uuid.New().String()[:8],
			State:      "running",
			UpdatedAt:  time.Now(),
		})
	}
	for i := 0; i < 5; i++ {
		repo.AddAsset(&service.Asset{
			ID:         uuid.New(),
			OrgID:      orgID,
			Platform:   "azure",
			InstanceID: "stopped-" + uuid.New().String()[:8],
			State:      "stopped",
			UpdatedAt:  time.Now(),
		})
	}
	for i := 0; i < 3; i++ {
		repo.AddAsset(&service.Asset{
			ID:         uuid.New(),
			OrgID:      orgID,
			Platform:   "gcp",
			InstanceID: "pending-" + uuid.New().String()[:8],
			State:      "pending",
			UpdatedAt:  time.Now(),
		})
	}

	t.Run("returns correct counts", func(t *testing.T) {
		result, err := svc.GetAssetSummary(ctx, service.GetAssetSummaryInput{
			OrgID: orgID,
		})

		require.NoError(t, err)
		assert.Equal(t, int64(18), result.TotalAssets)
		assert.Equal(t, int64(10), result.RunningAssets)
		assert.Equal(t, int64(5), result.StoppedAssets)
		assert.Equal(t, int64(10), result.ByState["running"])
		assert.Equal(t, int64(5), result.ByState["stopped"])
		assert.Equal(t, int64(3), result.ByState["other"])
	})
}

func TestAssetService_UpsertAsset(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockAssetRepository()
	svc := service.NewAssetService(repo)

	orgID := uuid.New()

	t.Run("create new asset", func(t *testing.T) {
		account := "123456789012"
		region := "us-east-1"

		result, err := svc.UpsertAsset(ctx, service.UpsertAssetInput{
			OrgID:      orgID,
			Platform:   "aws",
			Account:    account,
			Region:     region,
			InstanceID: "i-newinstance",
			State:      "running",
			Tags: map[string]string{
				"Environment": "production",
				"Team":        "platform",
			},
		})

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, orgID, result.OrgID)
		assert.Equal(t, "aws", result.Platform)
		assert.Equal(t, "i-newinstance", result.InstanceID)
		assert.Equal(t, "running", result.State)
	})

	t.Run("update existing asset", func(t *testing.T) {
		// Create initial
		initial, err := svc.UpsertAsset(ctx, service.UpsertAssetInput{
			OrgID:      orgID,
			Platform:   "aws",
			InstanceID: "i-updatetest",
			State:      "running",
		})
		require.NoError(t, err)

		// Update
		updated, err := svc.UpsertAsset(ctx, service.UpsertAssetInput{
			OrgID:      orgID,
			Platform:   "aws",
			InstanceID: "i-updatetest",
			State:      "stopped",
		})

		require.NoError(t, err)
		assert.Equal(t, initial.ID, updated.ID)
		assert.Equal(t, "stopped", updated.State)
	})

	t.Run("invalid platform", func(t *testing.T) {
		_, err := svc.UpsertAsset(ctx, service.UpsertAssetInput{
			OrgID:      orgID,
			Platform:   "invalid",
			InstanceID: "i-test",
			State:      "running",
		})

		assert.ErrorIs(t, err, service.ErrInvalidInput)
	})

	t.Run("valid platforms", func(t *testing.T) {
		validPlatforms := []string{"aws", "azure", "gcp", "vsphere", "kubernetes"}

		for _, platform := range validPlatforms {
			result, err := svc.UpsertAsset(ctx, service.UpsertAssetInput{
				OrgID:      orgID,
				Platform:   platform,
				InstanceID: "instance-" + platform,
				State:      "running",
			})

			require.NoError(t, err, "platform %s should be valid", platform)
			assert.Equal(t, platform, result.Platform)
		}
	})

	t.Run("instance_id required", func(t *testing.T) {
		_, err := svc.UpsertAsset(ctx, service.UpsertAssetInput{
			OrgID:      orgID,
			Platform:   "aws",
			InstanceID: "",
			State:      "running",
		})

		assert.ErrorIs(t, err, service.ErrInvalidInput)
	})
}

func TestAssetService_DeleteAsset(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockAssetRepository()
	svc := service.NewAssetService(repo)

	orgID := uuid.New()
	assetID := uuid.New()

	// Setup
	repo.AddAsset(&service.Asset{
		ID:         assetID,
		OrgID:      orgID,
		Platform:   "aws",
		InstanceID: "i-todelete",
		State:      "running",
		UpdatedAt:  time.Now(),
	})

	t.Run("success", func(t *testing.T) {
		err := svc.DeleteAsset(ctx, service.DeleteAssetInput{
			ID:    assetID,
			OrgID: orgID,
		})

		require.NoError(t, err)

		// Verify deleted
		_, err = svc.GetAsset(ctx, service.GetAssetInput{
			ID:    assetID,
			OrgID: orgID,
		})
		assert.ErrorIs(t, err, service.ErrNotFound)
	})

	t.Run("not found", func(t *testing.T) {
		err := svc.DeleteAsset(ctx, service.DeleteAssetInput{
			ID:    uuid.New(),
			OrgID: orgID,
		})

		assert.Error(t, err)
	})

	t.Run("wrong org", func(t *testing.T) {
		// Add another asset
		anotherID := uuid.New()
		repo.AddAsset(&service.Asset{
			ID:         anotherID,
			OrgID:      orgID,
			Platform:   "aws",
			InstanceID: "i-another",
			State:      "running",
			UpdatedAt:  time.Now(),
		})

		err := svc.DeleteAsset(ctx, service.DeleteAssetInput{
			ID:    anotherID,
			OrgID: uuid.New(), // Different org
		})

		assert.ErrorIs(t, err, service.ErrNotFound)
	})
}
