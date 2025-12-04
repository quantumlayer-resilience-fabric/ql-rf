package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
)

// MockAssetRepository is a mock implementation of AssetRepository.
type MockAssetRepository struct {
	mu     sync.RWMutex
	assets map[uuid.UUID]*service.Asset

	// Control behavior for testing
	GetAssetFunc             func(ctx context.Context, id uuid.UUID) (*service.Asset, error)
	GetAssetByInstanceIDFunc func(ctx context.Context, orgID uuid.UUID, platform, instanceID string) (*service.Asset, error)
	ListAssetsFunc           func(ctx context.Context, params service.ListAssetsParams) ([]service.Asset, error)
	ListDriftedAssetsFunc    func(ctx context.Context, orgID uuid.UUID, limit int32) ([]service.Asset, error)
	UpsertAssetFunc          func(ctx context.Context, params service.UpsertAssetParams) (*service.Asset, error)
	DeleteAssetFunc          func(ctx context.Context, id uuid.UUID) error
	CountAssetsByOrgFunc     func(ctx context.Context, orgID uuid.UUID) (int64, error)
	CountAssetsByStateFunc   func(ctx context.Context, orgID uuid.UUID, state string) (int64, error)
	CountCompliantAssetsFunc func(ctx context.Context, orgID uuid.UUID) (int64, error)
}

// NewMockAssetRepository creates a new MockAssetRepository.
func NewMockAssetRepository() *MockAssetRepository {
	return &MockAssetRepository{
		assets: make(map[uuid.UUID]*service.Asset),
	}
}

// GetAsset returns an asset by ID.
func (m *MockAssetRepository) GetAsset(ctx context.Context, id uuid.UUID) (*service.Asset, error) {
	if m.GetAssetFunc != nil {
		return m.GetAssetFunc(ctx, id)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if asset, ok := m.assets[id]; ok {
		return asset, nil
	}
	return nil, service.ErrNotFound
}

// GetAssetByInstanceID returns an asset by instance ID.
func (m *MockAssetRepository) GetAssetByInstanceID(ctx context.Context, orgID uuid.UUID, platform, instanceID string) (*service.Asset, error) {
	if m.GetAssetByInstanceIDFunc != nil {
		return m.GetAssetByInstanceIDFunc(ctx, orgID, platform, instanceID)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, asset := range m.assets {
		if asset.OrgID == orgID && asset.Platform == platform && asset.InstanceID == instanceID {
			return asset, nil
		}
	}
	return nil, service.ErrNotFound
}

// ListAssets returns a list of assets.
func (m *MockAssetRepository) ListAssets(ctx context.Context, params service.ListAssetsParams) ([]service.Asset, error) {
	if m.ListAssetsFunc != nil {
		return m.ListAssetsFunc(ctx, params)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []service.Asset
	for _, asset := range m.assets {
		if asset.OrgID != params.OrgID {
			continue
		}
		if params.EnvID != nil && (asset.EnvID == nil || *asset.EnvID != *params.EnvID) {
			continue
		}
		if params.Platform != nil && asset.Platform != *params.Platform {
			continue
		}
		if params.State != nil && asset.State != *params.State {
			continue
		}
		result = append(result, *asset)
	}

	// Apply pagination
	start := int(params.Offset)
	if start >= len(result) {
		return []service.Asset{}, nil
	}
	end := start + int(params.Limit)
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
}

// UpsertAsset creates or updates an asset.
func (m *MockAssetRepository) UpsertAsset(ctx context.Context, params service.UpsertAssetParams) (*service.Asset, error) {
	if m.UpsertAssetFunc != nil {
		return m.UpsertAssetFunc(ctx, params)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if asset exists
	var existing *service.Asset
	for _, asset := range m.assets {
		if asset.OrgID == params.OrgID && asset.Platform == params.Platform && asset.InstanceID == params.InstanceID {
			existing = asset
			break
		}
	}

	if existing != nil {
		// Update existing
		existing.EnvID = params.EnvID
		existing.Account = params.Account
		existing.Region = params.Region
		existing.ImageRef = params.ImageRef
		existing.ImageVersion = params.ImageVersion
		existing.State = params.State
		existing.Tags = params.Tags
		existing.UpdatedAt = time.Now()
		return existing, nil
	}

	// Create new
	asset := &service.Asset{
		ID:           uuid.New(),
		OrgID:        params.OrgID,
		EnvID:        params.EnvID,
		Platform:     params.Platform,
		Account:      params.Account,
		Region:       params.Region,
		InstanceID:   params.InstanceID,
		ImageRef:     params.ImageRef,
		ImageVersion: params.ImageVersion,
		State:        params.State,
		Tags:         params.Tags,
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}

	m.assets[asset.ID] = asset
	return asset, nil
}

// DeleteAsset deletes an asset.
func (m *MockAssetRepository) DeleteAsset(ctx context.Context, id uuid.UUID) error {
	if m.DeleteAssetFunc != nil {
		return m.DeleteAssetFunc(ctx, id)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.assets[id]; !ok {
		return service.ErrNotFound
	}

	delete(m.assets, id)
	return nil
}

// CountAssetsByOrg counts assets for an organization.
func (m *MockAssetRepository) CountAssetsByOrg(ctx context.Context, orgID uuid.UUID) (int64, error) {
	if m.CountAssetsByOrgFunc != nil {
		return m.CountAssetsByOrgFunc(ctx, orgID)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var count int64
	for _, asset := range m.assets {
		if asset.OrgID == orgID {
			count++
		}
	}
	return count, nil
}

// CountAssetsByState counts assets by state.
func (m *MockAssetRepository) CountAssetsByState(ctx context.Context, orgID uuid.UUID, state string) (int64, error) {
	if m.CountAssetsByStateFunc != nil {
		return m.CountAssetsByStateFunc(ctx, orgID, state)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var count int64
	for _, asset := range m.assets {
		if asset.OrgID == orgID && asset.State == state {
			count++
		}
	}
	return count, nil
}

// CountCompliantAssets counts compliant assets.
func (m *MockAssetRepository) CountCompliantAssets(ctx context.Context, orgID uuid.UUID) (int64, error) {
	if m.CountCompliantAssetsFunc != nil {
		return m.CountCompliantAssetsFunc(ctx, orgID)
	}

	// For testing, return 0 by default
	return 0, nil
}

// ListDriftedAssets returns drifted assets (assets where image doesn't match golden image).
func (m *MockAssetRepository) ListDriftedAssets(ctx context.Context, orgID uuid.UUID, limit int32) ([]service.Asset, error) {
	if m.ListDriftedAssetsFunc != nil {
		return m.ListDriftedAssetsFunc(ctx, orgID, limit)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []service.Asset
	for _, asset := range m.assets {
		if asset.OrgID != orgID {
			continue
		}
		// For testing, consider assets without image version as drifted
		if asset.ImageVersion == nil || *asset.ImageVersion == "" {
			result = append(result, *asset)
		}
		if int32(len(result)) >= limit {
			break
		}
	}

	return result, nil
}

// AddAsset adds an asset directly to the mock (for test setup).
func (m *MockAssetRepository) AddAsset(asset *service.Asset) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.assets[asset.ID] = asset
}

// Reset clears all data from the mock.
func (m *MockAssetRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.assets = make(map[uuid.UUID]*service.Asset)
}
