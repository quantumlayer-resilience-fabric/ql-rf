// Package connector defines the interface for cloud platform connectors.
package connector

import (
	"context"

	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

// Connector is the interface that all platform connectors must implement.
type Connector interface {
	// Name returns the name of the connector.
	Name() string

	// Platform returns the platform type (aws, azure, gcp, vsphere).
	Platform() models.Platform

	// Connect establishes a connection to the platform.
	Connect(ctx context.Context) error

	// Close closes the connection to the platform.
	Close() error

	// DiscoverAssets discovers all assets from the platform.
	DiscoverAssets(ctx context.Context, orgID uuid.UUID) ([]models.NormalizedAsset, error)

	// DiscoverImages discovers all available images from the platform.
	DiscoverImages(ctx context.Context) ([]ImageInfo, error)

	// Health checks the health of the connector.
	Health(ctx context.Context) error
}

// ImageInfo represents basic image information from a platform.
type ImageInfo struct {
	Platform    models.Platform
	Identifier  string
	Name        string
	Region      string
	CreatedAt   string
	Description string
	Tags        map[string]string
}

// Config holds common configuration for connectors.
type Config struct {
	OrgID        uuid.UUID
	Enabled      bool
	SyncInterval string
}

// DiscoveryResult holds the result of an asset discovery operation.
type DiscoveryResult struct {
	Platform     models.Platform
	Assets       []models.NormalizedAsset
	Images       []ImageInfo
	DiscoveredAt string
	Duration     string
	Errors       []string
}

// SyncStatus represents the status of a connector sync operation.
type SyncStatus struct {
	Platform     models.Platform
	Status       string // running, completed, failed
	StartedAt    string
	CompletedAt  string
	AssetsFound  int
	AssetsNew    int
	AssetsUpdated int
	AssetsRemoved int
	Errors       []string
}
