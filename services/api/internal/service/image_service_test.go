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

func TestImageService_GetImage(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockImageRepository()
	svc := service.NewImageService(repo)

	orgID := uuid.New()
	imageID := uuid.New()

	// Setup: Add an image
	repo.AddImage(&service.Image{
		ID:        imageID,
		OrgID:     orgID,
		Family:    "test-family",
		Version:   "1.0.0",
		Status:    "production",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	t.Run("success", func(t *testing.T) {
		result, err := svc.GetImage(ctx, service.GetImageInput{
			ID:    imageID,
			OrgID: orgID,
		})

		require.NoError(t, err)
		assert.Equal(t, imageID, result.ID)
		assert.Equal(t, "test-family", result.Family)
		assert.Equal(t, "1.0.0", result.Version)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetImage(ctx, service.GetImageInput{
			ID:    uuid.New(), // Non-existent ID
			OrgID: orgID,
		})

		assert.Error(t, err)
	})

	t.Run("wrong org returns not found", func(t *testing.T) {
		_, err := svc.GetImage(ctx, service.GetImageInput{
			ID:    imageID,
			OrgID: uuid.New(), // Different org
		})

		assert.ErrorIs(t, err, service.ErrNotFound)
	})
}

func TestImageService_GetLatestImage(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockImageRepository()
	svc := service.NewImageService(repo)

	orgID := uuid.New()

	// Setup: Add multiple versions
	repo.AddImage(&service.Image{
		ID:        uuid.New(),
		OrgID:     orgID,
		Family:    "test-family",
		Version:   "1.0.0",
		Status:    "deprecated",
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now(),
	})

	latestID := uuid.New()
	repo.AddImage(&service.Image{
		ID:        latestID,
		OrgID:     orgID,
		Family:    "test-family",
		Version:   "2.0.0",
		Status:    "production",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	t.Run("returns latest production version", func(t *testing.T) {
		result, err := svc.GetLatestImage(ctx, service.GetLatestImageInput{
			OrgID:  orgID,
			Family: "test-family",
		})

		require.NoError(t, err)
		assert.Equal(t, latestID, result.ID)
		assert.Equal(t, "2.0.0", result.Version)
	})

	t.Run("family not found", func(t *testing.T) {
		_, err := svc.GetLatestImage(ctx, service.GetLatestImageInput{
			OrgID:  orgID,
			Family: "non-existent",
		})

		assert.Error(t, err)
	})

	t.Run("family required", func(t *testing.T) {
		_, err := svc.GetLatestImage(ctx, service.GetLatestImageInput{
			OrgID:  orgID,
			Family: "",
		})

		assert.ErrorIs(t, err, service.ErrInvalidInput)
	})
}

func TestImageService_ListImages(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockImageRepository()
	svc := service.NewImageService(repo)

	orgID := uuid.New()
	otherOrgID := uuid.New()

	// Setup: Add images for multiple orgs
	for i := 0; i < 25; i++ {
		repo.AddImage(&service.Image{
			ID:        uuid.New(),
			OrgID:     orgID,
			Family:    "test-family",
			Version:   "1.0." + string(rune('0'+i%10)),
			Status:    "production",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}

	// Add image for different org (should not appear in results)
	repo.AddImage(&service.Image{
		ID:        uuid.New(),
		OrgID:     otherOrgID,
		Family:    "other-family",
		Version:   "1.0.0",
		Status:    "production",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	t.Run("returns paginated results", func(t *testing.T) {
		result, err := svc.ListImages(ctx, service.ListImagesInput{
			OrgID:    orgID,
			Page:     1,
			PageSize: 10,
		})

		require.NoError(t, err)
		assert.Len(t, result.Images, 10)
		assert.Equal(t, int64(25), result.Total)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 10, result.PageSize)
		assert.Equal(t, 3, result.TotalPages)
	})

	t.Run("second page", func(t *testing.T) {
		result, err := svc.ListImages(ctx, service.ListImagesInput{
			OrgID:    orgID,
			Page:     2,
			PageSize: 10,
		})

		require.NoError(t, err)
		assert.Len(t, result.Images, 10)
		assert.Equal(t, 2, result.Page)
	})

	t.Run("applies defaults", func(t *testing.T) {
		result, err := svc.ListImages(ctx, service.ListImagesInput{
			OrgID: orgID,
			// Page and PageSize not set
		})

		require.NoError(t, err)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PageSize)
	})
}

func TestImageService_CreateImage(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockImageRepository()
	svc := service.NewImageService(repo)

	orgID := uuid.New()

	t.Run("success", func(t *testing.T) {
		osName := "ubuntu"
		osVersion := "22.04"
		cisLevel := 2

		result, err := svc.CreateImage(ctx, service.CreateImageInput{
			OrgID:     orgID,
			Family:    "new-family",
			Version:   "1.0.0",
			OSName:    osName,
			OSVersion: osVersion,
			CISLevel:  cisLevel,
			Signed:    true,
		})

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, orgID, result.OrgID)
		assert.Equal(t, "new-family", result.Family)
		assert.Equal(t, "1.0.0", result.Version)
		assert.Equal(t, "draft", result.Status) // Default status
		assert.True(t, result.Signed)
	})

	t.Run("family required", func(t *testing.T) {
		_, err := svc.CreateImage(ctx, service.CreateImageInput{
			OrgID:   orgID,
			Family:  "",
			Version: "1.0.0",
		})

		assert.ErrorIs(t, err, service.ErrInvalidInput)
	})

	t.Run("version required", func(t *testing.T) {
		_, err := svc.CreateImage(ctx, service.CreateImageInput{
			OrgID:   orgID,
			Family:  "test",
			Version: "",
		})

		assert.ErrorIs(t, err, service.ErrInvalidInput)
	})
}

func TestImageService_PromoteImage(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockImageRepository()
	svc := service.NewImageService(repo)

	orgID := uuid.New()
	imageID := uuid.New()

	// Setup
	repo.AddImage(&service.Image{
		ID:        imageID,
		OrgID:     orgID,
		Family:    "test-family",
		Version:   "1.0.0",
		Status:    "draft",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	t.Run("promote to testing", func(t *testing.T) {
		result, err := svc.PromoteImage(ctx, service.PromoteImageInput{
			ID:       imageID,
			OrgID:    orgID,
			ToStatus: "testing",
		})

		require.NoError(t, err)
		assert.Equal(t, "testing", result.Status)
	})

	t.Run("promote to production", func(t *testing.T) {
		result, err := svc.PromoteImage(ctx, service.PromoteImageInput{
			ID:       imageID,
			OrgID:    orgID,
			ToStatus: "production",
		})

		require.NoError(t, err)
		assert.Equal(t, "production", result.Status)
	})

	t.Run("invalid status", func(t *testing.T) {
		_, err := svc.PromoteImage(ctx, service.PromoteImageInput{
			ID:       imageID,
			OrgID:    orgID,
			ToStatus: "invalid",
		})

		assert.ErrorIs(t, err, service.ErrInvalidInput)
	})

	t.Run("wrong org", func(t *testing.T) {
		_, err := svc.PromoteImage(ctx, service.PromoteImageInput{
			ID:       imageID,
			OrgID:    uuid.New(),
			ToStatus: "production",
		})

		assert.ErrorIs(t, err, service.ErrNotFound)
	})
}

func TestImageService_AddCoordinate(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockImageRepository()
	svc := service.NewImageService(repo)

	orgID := uuid.New()
	imageID := uuid.New()

	// Setup
	repo.AddImage(&service.Image{
		ID:        imageID,
		OrgID:     orgID,
		Family:    "test-family",
		Version:   "1.0.0",
		Status:    "production",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	t.Run("success with region", func(t *testing.T) {
		result, err := svc.AddCoordinate(ctx, service.AddCoordinateInput{
			ImageID:    imageID,
			OrgID:      orgID,
			Platform:   "aws",
			Region:     "us-east-1",
			Identifier: "ami-12345678",
		})

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, imageID, result.ImageID)
		assert.Equal(t, "aws", result.Platform)
		assert.Equal(t, "us-east-1", *result.Region)
		assert.Equal(t, "ami-12345678", result.Identifier)
	})

	t.Run("invalid platform", func(t *testing.T) {
		_, err := svc.AddCoordinate(ctx, service.AddCoordinateInput{
			ImageID:    imageID,
			OrgID:      orgID,
			Platform:   "invalid",
			Identifier: "test",
		})

		assert.ErrorIs(t, err, service.ErrInvalidInput)
	})

	t.Run("identifier required", func(t *testing.T) {
		_, err := svc.AddCoordinate(ctx, service.AddCoordinateInput{
			ImageID:    imageID,
			OrgID:      orgID,
			Platform:   "aws",
			Identifier: "",
		})

		assert.ErrorIs(t, err, service.ErrInvalidInput)
	})

	t.Run("wrong org", func(t *testing.T) {
		_, err := svc.AddCoordinate(ctx, service.AddCoordinateInput{
			ImageID:    imageID,
			OrgID:      uuid.New(),
			Platform:   "aws",
			Identifier: "ami-12345678",
		})

		assert.ErrorIs(t, err, service.ErrNotFound)
	})
}
