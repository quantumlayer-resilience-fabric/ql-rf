package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// AssetService handles asset business logic.
type AssetService struct {
	repo AssetRepository
}

// NewAssetService creates a new AssetService.
func NewAssetService(repo AssetRepository) *AssetService {
	return &AssetService{repo: repo}
}

// GetAssetInput contains input for getting an asset.
type GetAssetInput struct {
	ID    uuid.UUID
	OrgID uuid.UUID
}

// GetAsset retrieves an asset by ID with authorization check.
func (s *AssetService) GetAsset(ctx context.Context, input GetAssetInput) (*Asset, error) {
	asset, err := s.repo.GetAsset(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("get asset: %w", err)
	}

	// Authorization: verify org ownership
	if asset.OrgID != input.OrgID {
		return nil, ErrNotFound
	}

	return asset, nil
}

// ListAssetsInput contains input for listing assets.
type ListAssetsInput struct {
	OrgID    uuid.UUID
	EnvID    *uuid.UUID
	Platform *string
	State    *string
	Page     int
	PageSize int
}

// ListAssetsOutput contains output for listing assets.
type ListAssetsOutput struct {
	Assets     []Asset `json:"assets"`
	Total      int64   `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	TotalPages int     `json:"total_pages"`
}

// ListAssets retrieves a paginated list of assets.
func (s *AssetService) ListAssets(ctx context.Context, input ListAssetsInput) (*ListAssetsOutput, error) {
	// Apply defaults
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	offset := int32((input.Page - 1) * input.PageSize)

	assets, err := s.repo.ListAssets(ctx, ListAssetsParams{
		OrgID:    input.OrgID,
		EnvID:    input.EnvID,
		Platform: input.Platform,
		State:    input.State,
		Limit:    int32(input.PageSize),
		Offset:   offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}

	total, err := s.repo.CountAssetsByOrg(ctx, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("count assets: %w", err)
	}

	totalPages := int(total) / input.PageSize
	if int(total)%input.PageSize > 0 {
		totalPages++
	}

	return &ListAssetsOutput{
		Assets:     assets,
		Total:      total,
		Page:       input.Page,
		PageSize:   input.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetAssetSummaryInput contains input for getting asset summary.
type GetAssetSummaryInput struct {
	OrgID uuid.UUID
}

// AssetSummary contains summary statistics for assets.
type AssetSummary struct {
	TotalAssets   int64            `json:"total_assets"`
	RunningAssets int64            `json:"running_assets"`
	StoppedAssets int64            `json:"stopped_assets"`
	ByPlatform    map[string]int64 `json:"by_platform"`
	ByState       map[string]int64 `json:"by_state"`
}

// GetAssetSummary retrieves asset summary statistics.
func (s *AssetService) GetAssetSummary(ctx context.Context, input GetAssetSummaryInput) (*AssetSummary, error) {
	total, err := s.repo.CountAssetsByOrg(ctx, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("count assets: %w", err)
	}

	running, err := s.repo.CountAssetsByState(ctx, input.OrgID, "running")
	if err != nil {
		return nil, fmt.Errorf("count running assets: %w", err)
	}

	stopped, err := s.repo.CountAssetsByState(ctx, input.OrgID, "stopped")
	if err != nil {
		return nil, fmt.Errorf("count stopped assets: %w", err)
	}

	return &AssetSummary{
		TotalAssets:   total,
		RunningAssets: running,
		StoppedAssets: stopped,
		ByPlatform:    make(map[string]int64), // TODO: Add platform breakdown
		ByState: map[string]int64{
			"running": running,
			"stopped": stopped,
			"other":   total - running - stopped,
		},
	}, nil
}

// UpsertAssetInput contains input for upserting an asset.
type UpsertAssetInput struct {
	OrgID        uuid.UUID
	EnvID        *uuid.UUID
	Platform     string
	Account      string
	Region       string
	InstanceID   string
	ImageRef     string
	ImageVersion string
	State        string
	Tags         map[string]string
}

// Validate validates the upsert asset input.
func (i UpsertAssetInput) Validate() error {
	validPlatforms := map[string]bool{
		"aws":        true,
		"azure":      true,
		"gcp":        true,
		"vsphere":    true,
		"kubernetes": true,
	}
	if !validPlatforms[i.Platform] {
		return fmt.Errorf("%w: invalid platform %s", ErrInvalidInput, i.Platform)
	}
	if i.InstanceID == "" {
		return fmt.Errorf("%w: instance_id is required", ErrInvalidInput)
	}
	return nil
}

// UpsertAsset creates or updates an asset.
func (s *AssetService) UpsertAsset(ctx context.Context, input UpsertAssetInput) (*Asset, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	params := UpsertAssetParams{
		OrgID:      input.OrgID,
		EnvID:      input.EnvID,
		Platform:   input.Platform,
		InstanceID: input.InstanceID,
		State:      input.State,
	}

	// Set optional fields
	if input.Account != "" {
		params.Account = &input.Account
	}
	if input.Region != "" {
		params.Region = &input.Region
	}
	if input.ImageRef != "" {
		params.ImageRef = &input.ImageRef
	}
	if input.ImageVersion != "" {
		params.ImageVersion = &input.ImageVersion
	}

	// Convert tags to JSON
	if len(input.Tags) > 0 {
		tagsJSON, err := json.Marshal(input.Tags)
		if err != nil {
			return nil, fmt.Errorf("marshal tags: %w", err)
		}
		params.Tags = tagsJSON
	}

	asset, err := s.repo.UpsertAsset(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("upsert asset: %w", err)
	}

	return asset, nil
}

// DeleteAssetInput contains input for deleting an asset.
type DeleteAssetInput struct {
	ID    uuid.UUID
	OrgID uuid.UUID
}

// DeleteAsset deletes an asset.
func (s *AssetService) DeleteAsset(ctx context.Context, input DeleteAssetInput) error {
	// Verify ownership
	asset, err := s.repo.GetAsset(ctx, input.ID)
	if err != nil {
		return fmt.Errorf("get asset: %w", err)
	}
	if asset.OrgID != input.OrgID {
		return ErrNotFound
	}

	if err := s.repo.DeleteAsset(ctx, input.ID); err != nil {
		return fmt.Errorf("delete asset: %w", err)
	}

	return nil
}
