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
	cfg                       Config
	instancesClient           *compute.InstancesClient
	imagesClient              *compute.ImagesClient
	instanceGroupManagerClient *compute.InstanceGroupManagersClient
	instanceTemplatesClient   *compute.InstanceTemplatesClient
	log                       *logger.Logger
	connected                 bool
}

// Config holds GCP-specific configuration.
type Config struct {
	ProjectID       string
	CredentialsFile string   // Optional: uses ADC if not set
	Zones           []string // Optional: filter to specific zones
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

	// Create instance group managers client (for MIG discovery)
	igmClient, err := compute.NewInstanceGroupManagersRESTClient(ctx, opts...)
	if err != nil {
		instancesClient.Close()
		imagesClient.Close()
		return fmt.Errorf("failed to create instance group managers client: %w", err)
	}

	// Create instance templates client
	templatesClient, err := compute.NewInstanceTemplatesRESTClient(ctx, opts...)
	if err != nil {
		instancesClient.Close()
		imagesClient.Close()
		igmClient.Close()
		return fmt.Errorf("failed to create instance templates client: %w", err)
	}

	c.instancesClient = instancesClient
	c.imagesClient = imagesClient
	c.instanceGroupManagerClient = igmClient
	c.instanceTemplatesClient = templatesClient
	c.connected = true

	c.log.Info("connected to GCP",
		"project_id", c.cfg.ProjectID,
		"credentials_file", c.cfg.CredentialsFile != "",
		"zone_filter", len(c.cfg.Zones) > 0,
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
	if c.instanceGroupManagerClient != nil {
		c.instanceGroupManagerClient.Close()
	}
	if c.instanceTemplatesClient != nil {
		c.instanceTemplatesClient.Close()
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
// This includes standalone VMs and instances that are part of Managed Instance Groups (MIGs).
func (c *Connector) DiscoverAssets(ctx context.Context, orgID uuid.UUID) ([]models.NormalizedAsset, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	var allAssets []models.NormalizedAsset

	// Discover standalone instances
	standaloneAssets, err := c.discoverInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover instances: %w", err)
	}
	allAssets = append(allAssets, standaloneAssets...)

	// Discover MIG instances with additional metadata
	migAssets, err := c.discoverMIGInstances(ctx)
	if err != nil {
		c.log.Warn("failed to discover MIG instances, continuing with standalone instances",
			"error", err)
	} else {
		// Merge MIG metadata into existing assets
		allAssets = c.mergeMIGMetadata(allAssets, migAssets)
	}

	c.log.Info("asset discovery completed",
		"total_assets", len(allAssets),
		"project", c.cfg.ProjectID,
	)

	return allAssets, nil
}

// discoverInstances discovers all Compute Engine VM instances.
func (c *Connector) discoverInstances(ctx context.Context) ([]models.NormalizedAsset, error) {
	req := &computepb.AggregatedListInstancesRequest{
		Project: c.cfg.ProjectID,
	}

	var assets []models.NormalizedAsset

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

		// Apply zone filter if configured
		if !c.isZoneAllowed(zone) {
			continue
		}

		for _, instance := range pair.Value.Instances {
			asset := c.normalizeInstance(instance, zone, "")
			assets = append(assets, asset)
		}
	}

	return assets, nil
}

// discoverMIGInstances discovers Managed Instance Group information to enrich instance data.
func (c *Connector) discoverMIGInstances(ctx context.Context) (map[string]migInstanceInfo, error) {
	req := &computepb.AggregatedListInstanceGroupManagersRequest{
		Project: c.cfg.ProjectID,
	}

	migInstances := make(map[string]migInstanceInfo)

	it := c.instanceGroupManagerClient.AggregatedList(ctx, req)
	for {
		pair, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate instance group managers: %w", err)
		}

		if pair.Value.InstanceGroupManagers == nil {
			continue
		}

		zone := extractZoneFromKey(pair.Key)

		// Apply zone filter if configured
		if !c.isZoneAllowed(zone) {
			continue
		}

		for _, igm := range pair.Value.InstanceGroupManagers {
			if igm.Name == nil {
				continue
			}

			migName := *igm.Name
			templateName := ""
			if igm.InstanceTemplate != nil {
				templateName = extractResourceName(*igm.InstanceTemplate)
			}

			// Get target size
			targetSize := int32(0)
			if igm.TargetSize != nil {
				targetSize = *igm.TargetSize
			}

			// Store MIG info for instances in this group
			if igm.BaseInstanceName != nil {
				migInstances[*igm.BaseInstanceName] = migInstanceInfo{
					MIGName:      migName,
					TemplateName: templateName,
					TargetSize:   targetSize,
					Zone:         zone,
				}
			}
		}
	}

	return migInstances, nil
}

// migInstanceInfo holds MIG-specific information for an instance.
type migInstanceInfo struct {
	MIGName      string
	TemplateName string
	TargetSize   int32
	Zone         string
}

// mergeMIGMetadata enriches instance assets with MIG metadata.
func (c *Connector) mergeMIGMetadata(assets []models.NormalizedAsset, migInfo map[string]migInstanceInfo) []models.NormalizedAsset {
	for i := range assets {
		// Check if instance belongs to a MIG by matching base name pattern
		for baseName, info := range migInfo {
			if strings.HasPrefix(assets[i].Name, baseName) {
				assets[i].Tags["mig_name"] = info.MIGName
				assets[i].Tags["instance_template"] = info.TemplateName
				assets[i].Tags["mig_target_size"] = fmt.Sprintf("%d", info.TargetSize)
				break
			}
		}
	}
	return assets
}

// isZoneAllowed checks if a zone is in the allowed list.
func (c *Connector) isZoneAllowed(zone string) bool {
	if len(c.cfg.Zones) == 0 {
		return true // No filter means all zones allowed
	}
	for _, allowed := range c.cfg.Zones {
		if strings.EqualFold(allowed, zone) {
			return true
		}
	}
	return false
}

func (c *Connector) normalizeInstance(instance *computepb.Instance, zone string, migName string) models.NormalizedAsset {
	// Extract labels as tags
	tags := make(map[string]string)
	if instance.Labels != nil {
		for k, v := range instance.Labels {
			tags[k] = v
		}
	}

	// Add zone as tag
	tags["zone"] = zone

	// Add MIG name if present
	if migName != "" {
		tags["mig_name"] = migName
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

	// Extract image reference and version from boot disk
	imageRef, imageVersion := c.extractImageFromDisks(instance.Disks)

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

	// Add machine type to tags
	if instance.MachineType != nil {
		tags["machine_type"] = extractResourceName(*instance.MachineType)
	}

	return models.NormalizedAsset{
		Platform:     models.PlatformGCP,
		Account:      c.cfg.ProjectID,
		Region:       region,
		InstanceID:   instanceID,
		Name:         name,
		ImageRef:     imageRef,
		ImageVersion: imageVersion,
		State:        state,
		Tags:         tags,
	}
}

// extractImageFromDisks extracts the image reference and version from instance disks.
func (c *Connector) extractImageFromDisks(disks []*computepb.AttachedDisk) (string, string) {
	if disks == nil {
		return "", ""
	}

	for _, disk := range disks {
		if disk.Boot == nil || !*disk.Boot {
			continue
		}

		// Check initialize params for source image (new instances)
		if disk.InitializeParams != nil && disk.InitializeParams.SourceImage != nil {
			sourceImage := *disk.InitializeParams.SourceImage
			return c.parseImageURL(sourceImage)
		}

		// For existing disks, the source is the disk URL, not the image
		// We store the disk name as reference
		if disk.Source != nil {
			return extractResourceName(*disk.Source), ""
		}
	}

	return "", ""
}

// parseImageURL parses a GCP image URL and returns reference and version.
// URL formats:
// - projects/PROJECT/global/images/IMAGE_NAME
// - projects/PROJECT/global/images/family/FAMILY_NAME
// - https://www.googleapis.com/compute/v1/projects/PROJECT/global/images/IMAGE_NAME
func (c *Connector) parseImageURL(url string) (string, string) {
	if url == "" {
		return "", ""
	}

	// Check for image family reference
	if strings.Contains(url, "/images/family/") {
		parts := strings.Split(url, "/images/family/")
		if len(parts) >= 2 {
			family := parts[len(parts)-1]
			// Extract project from URL
			project := extractProjectFromURL(url)
			return fmt.Sprintf("%s/family/%s", project, family), "latest"
		}
	}

	// Standard image reference
	imageName := extractResourceName(url)
	project := extractProjectFromURL(url)

	// Try to extract version from image name (e.g., "ubuntu-2004-focal-v20231101")
	version := extractVersionFromImageName(imageName)

	if project != "" && project != c.cfg.ProjectID {
		return fmt.Sprintf("%s/%s", project, imageName), version
	}

	return imageName, version
}

// extractProjectFromURL extracts the project ID from a GCP URL.
func extractProjectFromURL(url string) string {
	parts := strings.Split(url, "/")
	for i, part := range parts {
		if part == "projects" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// extractVersionFromImageName attempts to extract a version from an image name.
func extractVersionFromImageName(name string) string {
	// Common patterns: ubuntu-2004-focal-v20231101, debian-11-bullseye-v20231010
	parts := strings.Split(name, "-")
	for _, part := range parts {
		if strings.HasPrefix(part, "v20") && len(part) == 9 {
			return part
		}
	}
	return ""
}

// DiscoverImages discovers all custom images and image families owned by the project.
func (c *Connector) DiscoverImages(ctx context.Context) ([]connector.ImageInfo, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	var allImages []connector.ImageInfo

	// Discover project images
	projectImages, err := c.discoverProjectImages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover project images: %w", err)
	}
	allImages = append(allImages, projectImages...)

	// Group images by family to identify latest in each family
	familyImages := c.groupImagesByFamily(projectImages)
	for family, latestImage := range familyImages {
		// Add a synthetic entry for the family reference
		familyEntry := connector.ImageInfo{
			Platform:    models.PlatformGCP,
			Identifier:  fmt.Sprintf("family/%s", family),
			Name:        fmt.Sprintf("%s (family)", family),
			Region:      "global",
			CreatedAt:   latestImage.CreatedAt,
			Description: fmt.Sprintf("Image family: %s (latest: %s)", family, latestImage.Name),
			Tags: map[string]string{
				"type":         "family",
				"family":       family,
				"latest_image": latestImage.Name,
			},
		}
		allImages = append(allImages, familyEntry)
	}

	c.log.Info("image discovery completed",
		"project_images", len(projectImages),
		"families", len(familyImages),
		"total", len(allImages),
	)

	return allImages, nil
}

// discoverProjectImages discovers all images in the project.
func (c *Connector) discoverProjectImages(ctx context.Context) ([]connector.ImageInfo, error) {
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

		// Extract family if present
		family := ""
		if image.Family != nil {
			family = *image.Family
			tags["family"] = family
		}

		// Extract status
		status := ""
		if image.Status != nil {
			status = *image.Status
			tags["status"] = status
		}

		// Extract disk size
		if image.DiskSizeGb != nil {
			tags["disk_size_gb"] = fmt.Sprintf("%d", *image.DiskSizeGb)
		}

		// Extract architecture
		if image.Architecture != nil {
			tags["architecture"] = *image.Architecture
		}

		// Extract source disk if present
		if image.SourceDisk != nil {
			tags["source_disk"] = extractResourceName(*image.SourceDisk)
		}

		// Determine image version from name
		version := extractVersionFromImageName(name)
		if version != "" {
			tags["version"] = version
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

	return images, nil
}

// groupImagesByFamily groups images by their family and returns the latest image in each family.
func (c *Connector) groupImagesByFamily(images []connector.ImageInfo) map[string]connector.ImageInfo {
	families := make(map[string]connector.ImageInfo)

	for _, img := range images {
		family, ok := img.Tags["family"]
		if !ok || family == "" {
			continue
		}

		// Keep the latest image (based on creation timestamp)
		existing, exists := families[family]
		if !exists || img.CreatedAt > existing.CreatedAt {
			families[family] = img
		}
	}

	return families
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
