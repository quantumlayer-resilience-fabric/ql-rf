// Package sync provides asset synchronization logic.
package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/kafka"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/repository"
)

// Service handles asset synchronization.
type Service struct {
	repo     *repository.Repository
	producer *kafka.Producer
	topic    string
	log      *logger.Logger
}

// New creates a new sync service.
func New(repo *repository.Repository, producer *kafka.Producer, topic string, log *logger.Logger) *Service {
	return &Service{
		repo:     repo,
		producer: producer,
		topic:    topic,
		log:      log.WithComponent("sync-service"),
	}
}

// SyncResult contains the results of a sync operation.
type SyncResult struct {
	Platform      string
	AssetsFound   int
	AssetsNew     int
	AssetsUpdated int
	AssetsRemoved int
	Errors        []error
	Duration      time.Duration
}

// SyncAssets synchronizes discovered assets with the database.
// All database operations are performed within a single transaction for consistency.
func (s *Service) SyncAssets(ctx context.Context, orgID uuid.UUID, platform string, discovered []models.NormalizedAsset) (*SyncResult, error) {
	startTime := time.Now()
	result := &SyncResult{
		Platform:    platform,
		AssetsFound: len(discovered),
	}

	// Start transaction for atomic sync operation
	tx, txRepo, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				s.log.Error("failed to rollback transaction", "error", rbErr)
			}
		}
	}()

	// Get existing assets for this platform
	existing, err := txRepo.ListAssetsByPlatform(ctx, orgID, platform)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing assets: %w", err)
	}

	// Build a map of existing assets by instance ID
	existingMap := make(map[string]*repository.Asset)
	for i := range existing {
		existingMap[existing[i].InstanceID] = &existing[i]
	}

	// Track which assets we've seen in this discovery
	seenInstanceIDs := make(map[string]bool)

	// Process discovered assets
	for _, asset := range discovered {
		seenInstanceIDs[asset.InstanceID] = true

		// Convert tags to JSON
		tagsJSON, err := json.Marshal(asset.Tags)
		if err != nil {
			s.log.Error("failed to marshal tags", "error", err, "instance_id", asset.InstanceID)
			result.Errors = append(result.Errors, err)
			continue
		}

		// Prepare upsert params
		params := repository.UpsertAssetParams{
			OrgID:      orgID,
			Platform:   string(asset.Platform),
			InstanceID: asset.InstanceID,
			State:      string(asset.State),
			Tags:       tagsJSON,
		}

		// Set optional fields
		if asset.Account != "" {
			params.Account = &asset.Account
		}
		if asset.Region != "" {
			params.Region = &asset.Region
		}
		if asset.Name != "" {
			params.Name = &asset.Name
		}
		if asset.ImageRef != "" {
			params.ImageRef = &asset.ImageRef
		}
		if asset.ImageVersion != "" {
			params.ImageVersion = &asset.ImageVersion
		}

		// Upsert asset
		dbAsset, isNew, err := txRepo.UpsertAsset(ctx, params)
		if err != nil {
			s.log.Error("failed to upsert asset", "error", err, "instance_id", asset.InstanceID)
			result.Errors = append(result.Errors, err)
			continue
		}

		// Determine action and publish event
		var action string
		if isNew {
			action = "created"
			result.AssetsNew++
		} else {
			// Check if anything changed
			existingAsset := existingMap[asset.InstanceID]
			if existingAsset != nil && hasChanged(existingAsset, dbAsset) {
				action = "updated"
				result.AssetsUpdated++
			}
		}

		// Publish event if there was an action
		if action != "" {
			if err := s.publishAssetEvent(ctx, dbAsset, action); err != nil {
				s.log.Error("failed to publish event", "error", err, "instance_id", asset.InstanceID)
				result.Errors = append(result.Errors, err)
			}
		}
	}

	// Handle removed assets (assets that exist in DB but not in discovery)
	for instanceID, existingAsset := range existingMap {
		if !seenInstanceIDs[instanceID] {
			// Asset no longer exists in the platform
			if existingAsset.State != "terminated" {
				// Mark as terminated
				if err := txRepo.MarkAssetTerminated(ctx, existingAsset.ID); err != nil {
					s.log.Error("failed to mark asset terminated", "error", err, "instance_id", instanceID)
					result.Errors = append(result.Errors, err)
					continue
				}

				// Publish removed event
				existingAsset.State = "terminated"
				if err := s.publishAssetEvent(ctx, existingAsset, "removed"); err != nil {
					s.log.Error("failed to publish removal event", "error", err, "instance_id", instanceID)
					result.Errors = append(result.Errors, err)
				}

				result.AssetsRemoved++
			}
		}
	}

	// Commit transaction if no critical errors
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	result.Duration = time.Since(startTime)

	s.log.Info("sync completed",
		"platform", platform,
		"found", result.AssetsFound,
		"new", result.AssetsNew,
		"updated", result.AssetsUpdated,
		"removed", result.AssetsRemoved,
		"errors", len(result.Errors),
		"duration", result.Duration.String(),
	)

	return result, nil
}

// hasChanged checks if an asset has changed.
func hasChanged(old, new *repository.Asset) bool {
	if old.State != new.State {
		return true
	}
	if ptrStringValue(old.ImageRef) != ptrStringValue(new.ImageRef) {
		return true
	}
	if ptrStringValue(old.ImageVersion) != ptrStringValue(new.ImageVersion) {
		return true
	}
	if ptrStringValue(old.Region) != ptrStringValue(new.Region) {
		return true
	}
	if ptrStringValue(old.Name) != ptrStringValue(new.Name) {
		return true
	}
	return false
}

func ptrStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// publishAssetEvent publishes an asset discovered/updated/removed event.
func (s *Service) publishAssetEvent(ctx context.Context, asset *repository.Asset, action string) error {
	event := kafka.Event{
		ID:        fmt.Sprintf("%s-%s-%s", asset.Platform, asset.InstanceID, action),
		Type:      "asset." + action,
		Source:    "connectors",
		Timestamp: time.Now(),
		Data: models.AssetDiscoveredEvent{
			Asset:     repoAssetToModel(asset),
			Action:    action,
			Timestamp: time.Now(),
		},
	}

	return s.producer.PublishEvent(ctx, s.topic, event)
}

// repoAssetToModel converts a repository asset to a model asset.
func repoAssetToModel(a *repository.Asset) models.Asset {
	asset := models.Asset{
		ID:           a.ID,
		OrgID:        a.OrgID,
		Platform:     models.Platform(a.Platform),
		InstanceID:   a.InstanceID,
		State:        models.AssetState(a.State),
		Tags:         a.Tags,
		DiscoveredAt: a.DiscoveredAt,
		UpdatedAt:    a.UpdatedAt,
	}

	if a.EnvID != nil {
		asset.EnvID = *a.EnvID
	}
	if a.Account != nil {
		asset.Account = *a.Account
	}
	if a.Region != nil {
		asset.Region = *a.Region
	}
	if a.Site != nil {
		asset.Site = *a.Site
	}
	if a.Name != nil {
		asset.Name = *a.Name
	}
	if a.ImageRef != nil {
		asset.ImageRef = *a.ImageRef
	}
	if a.ImageVersion != nil {
		asset.ImageVersion = *a.ImageVersion
	}

	return asset
}
