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

// CertificateInfo represents certificate information from a platform.
type CertificateInfo struct {
	Platform           models.Platform   `json:"platform"`
	Fingerprint        string            `json:"fingerprint"`
	SerialNumber       string            `json:"serial_number"`
	CommonName         string            `json:"common_name"`
	SubjectAltNames    []string          `json:"subject_alt_names"`
	Organization       string            `json:"organization"`
	IssuerCommonName   string            `json:"issuer_common_name"`
	IssuerOrganization string            `json:"issuer_organization"`
	IsSelfSigned       bool              `json:"is_self_signed"`
	IsCA               bool              `json:"is_ca"`
	NotBefore          string            `json:"not_before"`
	NotAfter           string            `json:"not_after"`
	KeyAlgorithm       string            `json:"key_algorithm"`
	KeySize            int               `json:"key_size"`
	SignatureAlgorithm string            `json:"signature_algorithm"`
	Source             string            `json:"source"` // acm, key_vault, k8s_secret, etc.
	SourceRef          string            `json:"source_ref"`
	Region             string            `json:"region"`
	AutoRenew          bool              `json:"auto_renew"`
	Status             string            `json:"status"` // active, pending_validation, expired, revoked
	Tags               map[string]string `json:"tags"`
	Usages             []CertificateUsageInfo `json:"usages,omitempty"`
}

// CertificateUsageInfo represents where a certificate is used.
type CertificateUsageInfo struct {
	UsageType   string `json:"usage_type"` // load_balancer, cloudfront, api_gateway, ingress, etc.
	UsageRef    string `json:"usage_ref"`
	ServiceName string `json:"service_name,omitempty"`
	Endpoint    string `json:"endpoint,omitempty"`
	Port        int    `json:"port,omitempty"`
}

// CertificateDiscoverer is an optional interface for connectors that can discover certificates.
type CertificateDiscoverer interface {
	// DiscoverCertificates discovers all certificates from the platform.
	DiscoverCertificates(ctx context.Context) ([]CertificateInfo, error)
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
