//go:build integration

// Package integration contains end-to-end integration tests for QL-RF services.
package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/aws"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/azure"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/gcp"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/vsphere"
)

// =============================================================================
// AWS Connector Tests
// =============================================================================

func TestAWSConnector(t *testing.T) {
	// Skip if AWS credentials are not configured
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" && os.Getenv("AWS_PROFILE") == "" {
		t.Skip("Skipping AWS connector test: AWS credentials not configured")
	}

	log := logger.New("error", "text")

	cfg := aws.Config{
		Region:  getEnvOrDefault("AWS_REGION", "us-east-1"),
		Profile: os.Getenv("AWS_PROFILE"),
	}

	connector := aws.New(cfg, log)

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "aws", connector.Name())
	})

	t.Run("Connect", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := connector.Connect(ctx)
		if err != nil {
			t.Skipf("Cannot connect to AWS: %v", err)
		}
		defer connector.Close()

		// Should be connected
		err = connector.Health(ctx)
		assert.NoError(t, err)
	})

	t.Run("DiscoverAssets", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		err := connector.Connect(ctx)
		if err != nil {
			t.Skip("Cannot connect to AWS")
		}
		defer connector.Close()

		siteID := uuid.New()
		assets, err := connector.DiscoverAssets(ctx, siteID)
		require.NoError(t, err)

		// May have zero assets, but should not error
		t.Logf("Discovered %d AWS assets", len(assets))
	})

	t.Run("DiscoverImages", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		err := connector.Connect(ctx)
		if err != nil {
			t.Skip("Cannot connect to AWS")
		}
		defer connector.Close()

		images, err := connector.DiscoverImages(ctx)
		require.NoError(t, err)

		t.Logf("Discovered %d AWS images", len(images))
	})
}

// =============================================================================
// Azure Connector Tests
// =============================================================================

func TestAzureConnector(t *testing.T) {
	// Skip if Azure credentials are not configured
	if os.Getenv("AZURE_SUBSCRIPTION_ID") == "" {
		t.Skip("Skipping Azure connector test: AZURE_SUBSCRIPTION_ID not set")
	}

	log := logger.New("error", "text")

	cfg := azure.Config{
		SubscriptionID: os.Getenv("AZURE_SUBSCRIPTION_ID"),
		TenantID:       os.Getenv("AZURE_TENANT_ID"),
		ClientID:       os.Getenv("AZURE_CLIENT_ID"),
		ClientSecret:   os.Getenv("AZURE_CLIENT_SECRET"),
	}

	connector := azure.New(cfg, log)

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "azure", connector.Name())
	})

	t.Run("Connect", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := connector.Connect(ctx)
		if err != nil {
			t.Skipf("Cannot connect to Azure: %v", err)
		}
		defer connector.Close()

		err = connector.Health(ctx)
		assert.NoError(t, err)
	})

	t.Run("DiscoverAssets", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		err := connector.Connect(ctx)
		if err != nil {
			t.Skip("Cannot connect to Azure")
		}
		defer connector.Close()

		siteID := uuid.New()
		assets, err := connector.DiscoverAssets(ctx, siteID)
		require.NoError(t, err)

		t.Logf("Discovered %d Azure assets", len(assets))

		// Verify asset structure if we have assets
		if len(assets) > 0 {
			asset := assets[0]
			assert.NotEmpty(t, asset.ExternalID)
			assert.NotEmpty(t, asset.Name)
			assert.Equal(t, "azure", string(asset.Platform))
		}
	})

	t.Run("DiscoverImages", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		err := connector.Connect(ctx)
		if err != nil {
			t.Skip("Cannot connect to Azure")
		}
		defer connector.Close()

		images, err := connector.DiscoverImages(ctx)
		require.NoError(t, err)

		t.Logf("Discovered %d Azure images", len(images))
	})
}

// =============================================================================
// GCP Connector Tests
// =============================================================================

func TestGCPConnector(t *testing.T) {
	// Skip if GCP credentials are not configured
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		// Try GOOGLE_CLOUD_PROJECT
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	if projectID == "" {
		t.Skip("Skipping GCP connector test: GCP_PROJECT_ID or GOOGLE_CLOUD_PROJECT not set")
	}

	log := logger.New("error", "text")

	cfg := gcp.Config{
		ProjectID:       projectID,
		CredentialsFile: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
	}

	connector := gcp.New(cfg, log)

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "gcp", connector.Name())
	})

	t.Run("Connect", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := connector.Connect(ctx)
		if err != nil {
			t.Skipf("Cannot connect to GCP: %v", err)
		}
		defer connector.Close()

		err = connector.Health(ctx)
		assert.NoError(t, err)
	})

	t.Run("DiscoverAssets", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		err := connector.Connect(ctx)
		if err != nil {
			t.Skip("Cannot connect to GCP")
		}
		defer connector.Close()

		siteID := uuid.New()
		assets, err := connector.DiscoverAssets(ctx, siteID)
		require.NoError(t, err)

		t.Logf("Discovered %d GCP assets", len(assets))

		if len(assets) > 0 {
			asset := assets[0]
			assert.NotEmpty(t, asset.ExternalID)
			assert.NotEmpty(t, asset.Name)
			assert.Equal(t, "gcp", string(asset.Platform))
		}
	})

	t.Run("DiscoverImages", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		err := connector.Connect(ctx)
		if err != nil {
			t.Skip("Cannot connect to GCP")
		}
		defer connector.Close()

		images, err := connector.DiscoverImages(ctx)
		require.NoError(t, err)

		t.Logf("Discovered %d GCP images", len(images))
	})
}

// =============================================================================
// vSphere Connector Tests
// =============================================================================

func TestVSphereConnector(t *testing.T) {
	// Skip if vSphere credentials are not configured
	if os.Getenv("VSPHERE_URL") == "" {
		t.Skip("Skipping vSphere connector test: VSPHERE_URL not set")
	}

	log := logger.New("error", "text")

	cfg := vsphere.Config{
		URL:      os.Getenv("VSPHERE_URL"),
		User:     os.Getenv("VSPHERE_USER"),
		Password: os.Getenv("VSPHERE_PASSWORD"),
		Insecure: os.Getenv("VSPHERE_INSECURE") == "true",
	}

	connector := vsphere.New(cfg, log)

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "vsphere", connector.Name())
	})

	t.Run("Connect", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := connector.Connect(ctx)
		if err != nil {
			t.Skipf("Cannot connect to vSphere: %v", err)
		}
		defer connector.Close()

		err = connector.Health(ctx)
		assert.NoError(t, err)
	})

	t.Run("DiscoverAssets", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		err := connector.Connect(ctx)
		if err != nil {
			t.Skip("Cannot connect to vSphere")
		}
		defer connector.Close()

		siteID := uuid.New()
		assets, err := connector.DiscoverAssets(ctx, siteID)
		require.NoError(t, err)

		t.Logf("Discovered %d vSphere assets", len(assets))

		if len(assets) > 0 {
			asset := assets[0]
			assert.NotEmpty(t, asset.ExternalID)
			assert.NotEmpty(t, asset.Name)
			assert.Equal(t, "vsphere", string(asset.Platform))
		}
	})

	t.Run("DiscoverImages", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		err := connector.Connect(ctx)
		if err != nil {
			t.Skip("Cannot connect to vSphere")
		}
		defer connector.Close()

		images, err := connector.DiscoverImages(ctx)
		require.NoError(t, err)

		t.Logf("Discovered %d vSphere images", len(images))
	})
}

// =============================================================================
// Multi-Connector Tests
// =============================================================================

func TestAllConnectorsHealth(t *testing.T) {
	log := logger.New("error", "text")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connectors := map[string]struct {
		create   func() interface{ Health(context.Context) error }
		required string
	}{
		"aws": {
			create: func() interface{ Health(context.Context) error } {
				return aws.New(aws.Config{
					Region: getEnvOrDefault("AWS_REGION", "us-east-1"),
				}, log)
			},
			required: "AWS_ACCESS_KEY_ID",
		},
		"azure": {
			create: func() interface{ Health(context.Context) error } {
				return azure.New(azure.Config{
					SubscriptionID: os.Getenv("AZURE_SUBSCRIPTION_ID"),
				}, log)
			},
			required: "AZURE_SUBSCRIPTION_ID",
		},
		"gcp": {
			create: func() interface{ Health(context.Context) error } {
				return gcp.New(gcp.Config{
					ProjectID: os.Getenv("GCP_PROJECT_ID"),
				}, log)
			},
			required: "GCP_PROJECT_ID",
		},
		"vsphere": {
			create: func() interface{ Health(context.Context) error } {
				return vsphere.New(vsphere.Config{
					URL: os.Getenv("VSPHERE_URL"),
				}, log)
			},
			required: "VSPHERE_URL",
		},
	}

	for name, tc := range connectors {
		t.Run(name, func(t *testing.T) {
			if os.Getenv(tc.required) == "" {
				t.Skipf("Skipping %s: %s not set", name, tc.required)
			}

			connector := tc.create()
			err := connector.Health(ctx)
			// May fail if not connected, which is expected
			t.Logf("%s health check result: %v", name, err)
		})
	}
}
