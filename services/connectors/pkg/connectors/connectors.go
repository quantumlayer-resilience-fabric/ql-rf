// Package connectors provides public connector types and factory functions
// for use by other services in the monorepo.
package connectors

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/aws"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/azure"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/connector"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/gcp"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/k8s"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/vsphere"
)

// Connector represents a cloud platform connector interface.
type Connector = connector.Connector

// ImageInfo represents discovered image information.
type ImageInfo = connector.ImageInfo

// AWSConfig holds AWS connector configuration.
type AWSConfig = aws.Config

// AzureConfig holds Azure connector configuration.
type AzureConfig = azure.Config

// GCPConfig holds GCP connector configuration.
type GCPConfig = gcp.Config

// VSphereConfig holds vSphere connector configuration.
type VSphereConfig = vsphere.Config

// K8sConfig holds Kubernetes connector configuration.
type K8sConfig = k8s.Config

// SyncResult contains results of an asset sync operation.
type SyncResult struct {
	AssetsCreated int
	AssetsUpdated int
	SitesCreated  int
	Errors        []error
}

// SyncService handles asset synchronization to the database.
type SyncService struct {
	pool *pgxpool.Pool
	log  *logger.Logger
}

// NewAWSConnector creates a new AWS connector.
func NewAWSConnector(cfg AWSConfig, log *logger.Logger) Connector {
	return aws.New(cfg, log)
}

// NewAzureConnector creates a new Azure connector.
func NewAzureConnector(cfg AzureConfig, log *logger.Logger) Connector {
	return azure.New(cfg, log)
}

// NewGCPConnector creates a new GCP connector.
func NewGCPConnector(cfg GCPConfig, log *logger.Logger) Connector {
	return gcp.New(cfg, log)
}

// NewVSphereConnector creates a new vSphere connector.
func NewVSphereConnector(cfg VSphereConfig, log *logger.Logger) Connector {
	return vsphere.New(cfg, log)
}

// NewK8sConnector creates a new Kubernetes connector.
func NewK8sConnector(cfg K8sConfig, log *logger.Logger) Connector {
	return k8s.New(cfg, log)
}

// NewSyncService creates a new sync service.
func NewSyncService(pool *pgxpool.Pool, log *logger.Logger) *SyncService {
	return &SyncService{
		pool: pool,
		log:  log.WithComponent("asset-sync"),
	}
}

// SyncAssets syncs discovered assets to the database.
func (s *SyncService) SyncAssets(ctx context.Context, orgID uuid.UUID, platform string, assets []models.NormalizedAsset) (*SyncResult, error) {
	result := &SyncResult{}

	for _, asset := range assets {
		tagsJSON, err := json.Marshal(asset.Tags)
		if err != nil {
			s.log.Error("failed to marshal tags", "error", err, "instance_id", asset.InstanceID)
			result.Errors = append(result.Errors, err)
			continue
		}

		// Upsert asset
		var assetID uuid.UUID
		var created bool
		err = s.pool.QueryRow(ctx, `
			INSERT INTO assets (org_id, platform, account, region, instance_id, name, image_ref, image_version, state, tags)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (org_id, platform, instance_id) DO UPDATE SET
				account = EXCLUDED.account,
				region = EXCLUDED.region,
				name = EXCLUDED.name,
				image_ref = EXCLUDED.image_ref,
				image_version = EXCLUDED.image_version,
				state = EXCLUDED.state,
				tags = EXCLUDED.tags,
				updated_at = NOW()
			RETURNING id, (xmax = 0) as created
		`, orgID, string(asset.Platform), asset.Account, asset.Region, asset.InstanceID,
			asset.Name, asset.ImageRef, asset.ImageVersion, string(asset.State), tagsJSON).Scan(&assetID, &created)

		if err != nil {
			s.log.Error("failed to upsert asset", "error", err, "instance_id", asset.InstanceID)
			result.Errors = append(result.Errors, fmt.Errorf("failed to upsert asset %s: %w", asset.InstanceID, err))
			continue
		}

		if created {
			result.AssetsCreated++
		} else {
			result.AssetsUpdated++
		}
	}

	s.log.Info("asset sync completed",
		"org_id", orgID,
		"platform", platform,
		"assets_created", result.AssetsCreated,
		"assets_updated", result.AssetsUpdated,
		"errors", len(result.Errors),
	)

	return result, nil
}

// SyncAssets is a helper function for backward compatibility.
func SyncAssets(ctx context.Context, svc *SyncService, orgID uuid.UUID, platform string, assets []models.NormalizedAsset) (*SyncResult, error) {
	return svc.SyncAssets(ctx, orgID, platform, assets)
}
