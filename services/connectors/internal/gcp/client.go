// Package gcp provides GCP connector functionality.
package gcp

import (
	"context"
	"fmt"
	"strings"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/connector"
)

// Connector implements the GCP platform connector.
type Connector struct {
	cfg             Config
	instancesClient *compute.InstancesClient
	imagesClient    *compute.ImagesClient
	log             *logger.Logger
	connected       bool
}

// Config holds GCP-specific configuration.
type Config struct {
	ProjectID       string
	CredentialsFile string // Optional: uses ADC if not set
}

// New creates a new GCP connector.
func New(cfg Config, log *logger.Logger) *Connector {
	return &Connector{
		cfg: cfg,
		log: log.WithComponent("gcp-connector"),
	}
}

// Name returns the connector name.
func (c *Connector) Name() string {
	return "gcp"
}

// Platform returns the platform type.
func (c *Connector) Platform() models.Platform {
	return models.PlatformGCP
}

// Connect establishes a connection to GCP.
func (c *Connector) Connect(ctx context.Context) error {
	var opts []option.ClientOption
	if c.cfg.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(c.cfg.CredentialsFile))
	}

	// Create instances client
	instancesClient, err := compute.NewInstancesRESTClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create instances client: %w", err)
	}

	// Create images client
	imagesClient, err := compute.NewImagesRESTClient(ctx, opts...)
	if err != nil {
		instancesClient.Close()
		return fmt.Errorf("failed to create images client: %w", err)
	}

	c.instancesClient = instancesClient
	c.imagesClient = imagesClient
	c.connected = true

	c.log.Info("connected to GCP",
		"project_id", c.cfg.ProjectID,
		"credentials_file", c.cfg.CredentialsFile != "",
	)

	return nil
}

// Close closes the GCP connection.
func (c *Connector) Close() error {
	if c.instancesClient != nil {
		c.instancesClient.Close()
	}
	if c.imagesClient != nil {
		c.imagesClient.Close()
	}
	c.connected = false
	return nil
}

// Health checks the health of the GCP connection.
func (c *Connector) Health(ctx context.Context) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	// Try to list zones as a health check (limited to 1)
	req := &computepb.AggregatedListInstancesRequest{
		Project:    c.cfg.ProjectID,
		MaxResults: ptrUint32(1),
	}

	it := c.instancesClient.AggregatedList(ctx, req)
	_, err := it.Next()
	if err != nil && err != iterator.Done {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// DiscoverAssets discovers all Compute Engine instances from GCP.
func (c *Connector) DiscoverAssets(ctx context.Context, orgID uuid.UUID) ([]models.NormalizedAsset, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	req := &computepb.AggregatedListInstancesRequest{
		Project: c.cfg.ProjectID,
	}

	var allAssets []models.NormalizedAsset

	it := c.instancesClient.AggregatedList(ctx, req)
	for {
		pair, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate instances: %w", err)
		}

		// pair.Key is like "zones/us-central1-a"
		// pair.Value is InstancesScopedList
		if pair.Value.Instances == nil {
			continue
		}

		zone := extractZoneFromKey(pair.Key)
		for _, instance := range pair.Value.Instances {
			asset := c.normalizeInstance(instance, zone)
			allAssets = append(allAssets, asset)
		}
	}

	c.log.Info("asset discovery completed",
		"total_assets", len(allAssets),
		"project", c.cfg.ProjectID,
	)

	return allAssets, nil
}

func (c *Connector) normalizeInstance(instance *computepb.Instance, zone string) models.NormalizedAsset {
	// Extract labels as tags
	tags := make(map[string]string)
	if instance.Labels != nil {
		for k, v := range instance.Labels {
			tags[k] = v
		}
	}

	// Map GCP status to our state
	state := models.AssetStateUnknown
	if instance.Status != nil {
		switch *instance.Status {
		case "RUNNING":
			state = models.AssetStateRunning
		case "STOPPED":
			state = models.AssetStateStopped
		case "TERMINATED":
			state = models.AssetStateTerminated
		case "STAGING", "PROVISIONING":
			state = models.AssetStatePending
		case "SUSPENDING", "SUSPENDED":
			state = models.AssetStateStopped
		}
	}

	// Extract image reference from boot disk
	imageRef := ""
	if instance.Disks != nil {
		for _, disk := range instance.Disks {
			if disk.Boot != nil && *disk.Boot {
				if disk.Source != nil {
					// Source is a URL like: projects/.../zones/.../disks/disk-name
					imageRef = extractResourceName(*disk.Source)
				}
				break
			}
		}
	}

	// Get instance name
	name := ""
	if instance.Name != nil {
		name = *instance.Name
	}

	// Get instance ID
	instanceID := ""
	if instance.Id != nil {
		instanceID = fmt.Sprintf("%d", *instance.Id)
	}

	// Extract region from zone (e.g., "us-central1-a" -> "us-central1")
	region := extractRegionFromZone(zone)

	return models.NormalizedAsset{
		Platform:     models.PlatformGCP,
		Account:      c.cfg.ProjectID,
		Region:       region,
		InstanceID:   instanceID,
		Name:         name,
		ImageRef:     imageRef,
		ImageVersion: "", // Would require additional API call to get image details
		State:        state,
		Tags:         tags,
	}
}

// DiscoverImages discovers all custom images owned by the project.
func (c *Connector) DiscoverImages(ctx context.Context) ([]connector.ImageInfo, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	req := &computepb.ListImagesRequest{
		Project: c.cfg.ProjectID,
	}

	var images []connector.ImageInfo

	it := c.imagesClient.List(ctx, req)
	for {
		image, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate images: %w", err)
		}

		// Extract labels as tags
		tags := make(map[string]string)
		if image.Labels != nil {
			for k, v := range image.Labels {
				tags[k] = v
			}
		}

		name := ""
		if image.Name != nil {
			name = *image.Name
		}

		description := ""
		if image.Description != nil {
			description = *image.Description
		}

		createdAt := ""
		if image.CreationTimestamp != nil {
			createdAt = *image.CreationTimestamp
		}

		images = append(images, connector.ImageInfo{
			Platform:    models.PlatformGCP,
			Identifier:  name,
			Name:        name,
			Region:      "global", // GCP images are global
			CreatedAt:   createdAt,
			Description: description,
			Tags:        tags,
		})
	}

	c.log.Info("image discovery completed", "count", len(images))

	return images, nil
}

// Helper functions

func extractZoneFromKey(key string) string {
	// key is like "zones/us-central1-a"
	parts := strings.Split(key, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return key
}

func extractRegionFromZone(zone string) string {
	// zone is like "us-central1-a" -> "us-central1"
	lastDash := strings.LastIndex(zone, "-")
	if lastDash > 0 {
		return zone[:lastDash]
	}
	return zone
}

func extractResourceName(url string) string {
	// URL is like "https://www.googleapis.com/compute/v1/projects/.../zones/.../disks/disk-name"
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return url
}

func ptrUint32(v uint32) *uint32 {
	return &v
}
