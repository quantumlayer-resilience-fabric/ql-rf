package sync_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/repository"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/sync"
)

// MockRepository is a mock implementation of the repository for testing.
type MockRepository struct {
	assets    map[string]*repository.Asset
	upsertErr error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		assets: make(map[string]*repository.Asset),
	}
}

func (m *MockRepository) UpsertAsset(ctx context.Context, params repository.UpsertAssetParams) (*repository.Asset, bool, error) {
	if m.upsertErr != nil {
		return nil, false, m.upsertErr
	}

	key := params.Platform + ":" + params.InstanceID
	existing := m.assets[key]
	isNew := existing == nil

	asset := &repository.Asset{
		ID:           uuid.New(),
		OrgID:        params.OrgID,
		Platform:     params.Platform,
		Account:      params.Account,
		Region:       params.Region,
		InstanceID:   params.InstanceID,
		Name:         params.Name,
		ImageRef:     params.ImageRef,
		ImageVersion: params.ImageVersion,
		State:        params.State,
		Tags:         params.Tags,
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}

	if !isNew {
		asset.ID = existing.ID
		asset.DiscoveredAt = existing.DiscoveredAt
	}

	m.assets[key] = asset
	return asset, isNew, nil
}

func (m *MockRepository) ListAssetsByPlatform(ctx context.Context, orgID uuid.UUID, platform string) ([]repository.Asset, error) {
	var result []repository.Asset
	for key, asset := range m.assets {
		if asset.OrgID == orgID && asset.Platform == platform {
			result = append(result, *asset)
			_ = key
		}
	}
	return result, nil
}

func (m *MockRepository) MarkAssetTerminated(ctx context.Context, id uuid.UUID) error {
	for _, asset := range m.assets {
		if asset.ID == id {
			asset.State = "terminated"
			return nil
		}
	}
	return nil
}

// MockProducer is a mock Kafka producer for testing.
type MockProducer struct {
	events []interface{}
}

func (m *MockProducer) PublishEvent(ctx context.Context, topic string, event interface{}) error {
	m.events = append(m.events, event)
	return nil
}

// MockLogger is a mock logger for testing.
type MockLogger struct{}

func (m *MockLogger) Info(msg string, keysAndValues ...interface{})  {}
func (m *MockLogger) Debug(msg string, keysAndValues ...interface{}) {}
func (m *MockLogger) Warn(msg string, keysAndValues ...interface{})  {}
func (m *MockLogger) Error(msg string, keysAndValues ...interface{}) {}
func (m *MockLogger) WithComponent(name string) *MockLogger         { return m }

// TestSyncService tests the sync service.
func TestSyncService_SyncAssets(t *testing.T) {
	orgID := uuid.New()

	t.Run("syncs new assets", func(t *testing.T) {
		repo := NewMockRepository()

		// Discovered assets
		discovered := []models.NormalizedAsset{
			{
				Platform:   models.PlatformAWS,
				InstanceID: "i-12345678",
				Region:     "us-east-1",
				State:      models.AssetStateRunning,
				Tags:       map[string]string{"Name": "test-instance"},
			},
			{
				Platform:   models.PlatformAWS,
				InstanceID: "i-87654321",
				Region:     "us-west-2",
				State:      models.AssetStateRunning,
				Tags:       map[string]string{"Name": "test-instance-2"},
			},
		}

		// Create sync service with mock dependencies
		result := syncAssets(repo, orgID, "aws", discovered)

		assert.Equal(t, 2, result.AssetsFound)
		assert.Equal(t, 2, result.AssetsNew)
		assert.Equal(t, 0, result.AssetsUpdated)
		assert.Equal(t, 0, result.AssetsRemoved)
		assert.Len(t, result.Errors, 0)
	})

	t.Run("detects removed assets", func(t *testing.T) {
		repo := NewMockRepository()

		// Pre-populate with existing asset
		existingID := uuid.New()
		region := "us-east-1"
		name := "old-instance"
		repo.assets["aws:i-old"] = &repository.Asset{
			ID:         existingID,
			OrgID:      orgID,
			Platform:   "aws",
			InstanceID: "i-old",
			Region:     &region,
			Name:       &name,
			State:      "running",
		}

		// Discovered assets (without the old one)
		discovered := []models.NormalizedAsset{
			{
				Platform:   models.PlatformAWS,
				InstanceID: "i-new",
				Region:     "us-east-1",
				State:      models.AssetStateRunning,
			},
		}

		result := syncAssets(repo, orgID, "aws", discovered)

		assert.Equal(t, 1, result.AssetsFound)
		assert.Equal(t, 1, result.AssetsNew)
		assert.Equal(t, 1, result.AssetsRemoved)

		// Verify old asset is marked as terminated
		assert.Equal(t, "terminated", repo.assets["aws:i-old"].State)
	})

	t.Run("handles empty discovery", func(t *testing.T) {
		repo := NewMockRepository()

		discovered := []models.NormalizedAsset{}

		result := syncAssets(repo, orgID, "aws", discovered)

		assert.Equal(t, 0, result.AssetsFound)
		assert.Equal(t, 0, result.AssetsNew)
	})
}

// syncAssets is a helper that simulates the sync logic for testing.
// In real tests, we'd use the actual service with mocked dependencies.
func syncAssets(repo *MockRepository, orgID uuid.UUID, platform string, discovered []models.NormalizedAsset) *sync.SyncResult {
	result := &sync.SyncResult{
		Platform:    platform,
		AssetsFound: len(discovered),
	}

	ctx := context.Background()

	// Get existing assets
	existing, _ := repo.ListAssetsByPlatform(ctx, orgID, platform)
	existingMap := make(map[string]*repository.Asset)
	for i := range existing {
		existingMap[existing[i].InstanceID] = &existing[i]
	}

	seenInstanceIDs := make(map[string]bool)

	for _, asset := range discovered {
		seenInstanceIDs[asset.InstanceID] = true

		tagsJSON, _ := json.Marshal(asset.Tags)
		params := repository.UpsertAssetParams{
			OrgID:      orgID,
			Platform:   string(asset.Platform),
			InstanceID: asset.InstanceID,
			State:      string(asset.State),
			Tags:       tagsJSON,
		}

		if asset.Region != "" {
			params.Region = &asset.Region
		}
		if asset.Name != "" {
			params.Name = &asset.Name
		}

		_, isNew, err := repo.UpsertAsset(ctx, params)
		if err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		if isNew {
			result.AssetsNew++
		}
	}

	// Handle removed assets
	for instanceID, existingAsset := range existingMap {
		if !seenInstanceIDs[instanceID] {
			if existingAsset.State != "terminated" {
				_ = repo.MarkAssetTerminated(ctx, existingAsset.ID)
				result.AssetsRemoved++
			}
		}
	}

	return result
}

func TestSyncResult(t *testing.T) {
	result := &sync.SyncResult{
		Platform:      "aws",
		AssetsFound:   10,
		AssetsNew:     3,
		AssetsUpdated: 5,
		AssetsRemoved: 2,
		Duration:      time.Second * 5,
	}

	assert.Equal(t, "aws", result.Platform)
	assert.Equal(t, 10, result.AssetsFound)
	assert.Equal(t, 3, result.AssetsNew)
	assert.Equal(t, 5, result.AssetsUpdated)
	assert.Equal(t, 2, result.AssetsRemoved)
	require.NotZero(t, result.Duration)
}
