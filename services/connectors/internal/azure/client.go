// Package azure provides Azure connector functionality.
package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/connector"
)

// Connector implements the Azure platform connector.
type Connector struct {
	cfg            Config
	vmClient       *armcompute.VirtualMachinesClient
	vmssClient     *armcompute.VirtualMachineScaleSetsClient
	vmssVMsClient  *armcompute.VirtualMachineScaleSetVMsClient
	imagesClient   *armcompute.ImagesClient
	galleriesClient *armcompute.GalleriesClient
	galleryImagesClient *armcompute.GalleryImagesClient
	galleryImageVersionsClient *armcompute.GalleryImageVersionsClient
	log            *logger.Logger
	connected      bool
	subscriptionID string
}

// Config holds Azure-specific configuration.
type Config struct {
	TenantID       string
	ClientID       string
	ClientSecret   string
	SubscriptionID string
	ResourceGroups []string // Optional: filter to specific resource groups
}

// New creates a new Azure connector.
func New(cfg Config, log *logger.Logger) *Connector {
	return &Connector{
		cfg: cfg,
		log: log.WithComponent("azure-connector"),
	}
}

// Name returns the connector name.
func (c *Connector) Name() string {
	return "azure"
}

// Platform returns the platform type.
func (c *Connector) Platform() models.Platform {
	return models.PlatformAzure
}

// Connect establishes a connection to Azure.
func (c *Connector) Connect(ctx context.Context) error {
	var cred *azidentity.ClientSecretCredential
	var err error

	// Use service principal credentials if provided
	if c.cfg.ClientID != "" && c.cfg.ClientSecret != "" && c.cfg.TenantID != "" {
		cred, err = azidentity.NewClientSecretCredential(
			c.cfg.TenantID,
			c.cfg.ClientID,
			c.cfg.ClientSecret,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to create credentials: %w", err)
		}
	} else {
		return fmt.Errorf("azure credentials not configured: TenantID, ClientID, and ClientSecret are required")
	}

	// Create VM client
	vmClient, err := armcompute.NewVirtualMachinesClient(c.cfg.SubscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create VM client: %w", err)
	}

	// Create VMSS client
	vmssClient, err := armcompute.NewVirtualMachineScaleSetsClient(c.cfg.SubscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create VMSS client: %w", err)
	}

	// Create VMSS VMs client
	vmssVMsClient, err := armcompute.NewVirtualMachineScaleSetVMsClient(c.cfg.SubscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create VMSS VMs client: %w", err)
	}

	// Create images client
	imagesClient, err := armcompute.NewImagesClient(c.cfg.SubscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create images client: %w", err)
	}

	// Create Shared Image Gallery clients
	galleriesClient, err := armcompute.NewGalleriesClient(c.cfg.SubscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create galleries client: %w", err)
	}

	galleryImagesClient, err := armcompute.NewGalleryImagesClient(c.cfg.SubscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create gallery images client: %w", err)
	}

	galleryImageVersionsClient, err := armcompute.NewGalleryImageVersionsClient(c.cfg.SubscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create gallery image versions client: %w", err)
	}

	c.vmClient = vmClient
	c.vmssClient = vmssClient
	c.vmssVMsClient = vmssVMsClient
	c.imagesClient = imagesClient
	c.galleriesClient = galleriesClient
	c.galleryImagesClient = galleryImagesClient
	c.galleryImageVersionsClient = galleryImageVersionsClient
	c.subscriptionID = c.cfg.SubscriptionID
	c.connected = true

	c.log.Info("connected to Azure",
		"subscription_id", c.cfg.SubscriptionID,
		"tenant_id", c.cfg.TenantID,
	)

	return nil
}

// Close closes the Azure connection.
func (c *Connector) Close() error {
	c.connected = false
	return nil
}

// Health checks the health of the Azure connection.
func (c *Connector) Health(ctx context.Context) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	// Try to list VMs as a health check (limited to 1)
	pager := c.vmClient.NewListAllPager(&armcompute.VirtualMachinesClientListAllOptions{})
	if pager.More() {
		_, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("health check failed: %w", err)
		}
	}

	return nil
}

// DiscoverAssets discovers all Virtual Machines and VMSS instances from Azure.
func (c *Connector) DiscoverAssets(ctx context.Context, orgID uuid.UUID) ([]models.NormalizedAsset, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	var allAssets []models.NormalizedAsset

	// Discover standalone VMs
	vmAssets, err := c.discoverVMs(ctx)
	if err != nil {
		c.log.Error("failed to discover VMs", "error", err)
	} else {
		allAssets = append(allAssets, vmAssets...)
	}

	// Discover VMSS instances
	vmssAssets, err := c.discoverVMSSInstances(ctx)
	if err != nil {
		c.log.Error("failed to discover VMSS instances", "error", err)
	} else {
		allAssets = append(allAssets, vmssAssets...)
	}

	c.log.Info("asset discovery completed",
		"total_assets", len(allAssets),
		"vms", len(vmAssets),
		"vmss_instances", len(vmssAssets),
		"subscription", c.cfg.SubscriptionID,
	)

	return allAssets, nil
}

// discoverVMs discovers standalone Virtual Machines.
func (c *Connector) discoverVMs(ctx context.Context) ([]models.NormalizedAsset, error) {
	var assets []models.NormalizedAsset

	// List all VMs in the subscription
	pager := c.vmClient.NewListAllPager(&armcompute.VirtualMachinesClientListAllOptions{
		StatusOnly: ptrString("false"),
	})

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list VMs: %w", err)
		}

		for _, vm := range page.Value {
			// Check resource group filter
			resourceGroup := extractResourceGroupFromID(ptrToString(vm.ID))
			if !c.isResourceGroupAllowed(resourceGroup) {
				continue
			}

			vmName := ptrToString(vm.Name)

			// Get instance view for power state
			instanceView, err := c.vmClient.InstanceView(ctx, resourceGroup, vmName, nil)
			if err != nil {
				c.log.Warn("failed to get instance view",
					"vm", vmName,
					"resource_group", resourceGroup,
					"error", err,
				)
			}

			asset := c.normalizeVM(vm, &instanceView.VirtualMachineInstanceView, resourceGroup)
			assets = append(assets, asset)
		}
	}

	return assets, nil
}

// discoverVMSSInstances discovers instances in Virtual Machine Scale Sets.
func (c *Connector) discoverVMSSInstances(ctx context.Context) ([]models.NormalizedAsset, error) {
	var assets []models.NormalizedAsset

	// List all VMSS in the subscription
	vmssPager := c.vmssClient.NewListAllPager(nil)

	for vmssPager.More() {
		vmssPage, err := vmssPager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list VMSS: %w", err)
		}

		for _, vmss := range vmssPage.Value {
			resourceGroup := extractResourceGroupFromID(ptrToString(vmss.ID))
			if !c.isResourceGroupAllowed(resourceGroup) {
				continue
			}

			vmssName := ptrToString(vmss.Name)

			// Get image reference from VMSS
			var imageRef string
			var imageVersion string
			if vmss.Properties != nil && vmss.Properties.VirtualMachineProfile != nil {
				profile := vmss.Properties.VirtualMachineProfile
				if profile.StorageProfile != nil && profile.StorageProfile.ImageReference != nil {
					imageRef, imageVersion = c.extractImageReference(profile.StorageProfile.ImageReference)
				}
			}

			// List all VMs in this VMSS
			vmPager := c.vmssVMsClient.NewListPager(resourceGroup, vmssName, nil)

			for vmPager.More() {
				vmPage, err := vmPager.NextPage(ctx)
				if err != nil {
					c.log.Warn("failed to list VMSS VMs",
						"vmss", vmssName,
						"resource_group", resourceGroup,
						"error", err,
					)
					break
				}

				for _, vm := range vmPage.Value {
					asset := c.normalizeVMSSVM(vm, vmssName, resourceGroup, imageRef, imageVersion)
					assets = append(assets, asset)
				}
			}
		}
	}

	return assets, nil
}

// isResourceGroupAllowed checks if the resource group is in the allowed list.
func (c *Connector) isResourceGroupAllowed(resourceGroup string) bool {
	if len(c.cfg.ResourceGroups) == 0 {
		return true // No filter, allow all
	}
	for _, rg := range c.cfg.ResourceGroups {
		if strings.EqualFold(rg, resourceGroup) {
			return true
		}
	}
	return false
}

func (c *Connector) normalizeVM(vm *armcompute.VirtualMachine, instanceView *armcompute.VirtualMachineInstanceView, resourceGroup string) models.NormalizedAsset {
	// Extract tags
	tags := make(map[string]string)
	if vm.Tags != nil {
		for k, v := range vm.Tags {
			if v != nil {
				tags[k] = *v
			}
		}
	}
	// Add resource group as tag for reference
	tags["azure:resource_group"] = resourceGroup

	// Determine power state from instance view
	state := models.AssetStateUnknown
	if instanceView != nil && instanceView.Statuses != nil {
		for _, status := range instanceView.Statuses {
			if status.Code != nil && strings.HasPrefix(*status.Code, "PowerState/") {
				powerState := strings.TrimPrefix(*status.Code, "PowerState/")
				switch powerState {
				case "running":
					state = models.AssetStateRunning
				case "stopped", "deallocated":
					state = models.AssetStateStopped
				case "starting", "stopping":
					state = models.AssetStatePending
				}
				break
			}
		}
	}

	// Extract image reference
	var imageRef, imageVersion string
	if vm.Properties != nil && vm.Properties.StorageProfile != nil && vm.Properties.StorageProfile.ImageReference != nil {
		imageRef, imageVersion = c.extractImageReference(vm.Properties.StorageProfile.ImageReference)
	}

	// Get VM name
	name := ptrToString(vm.Name)

	// Get instance ID from resource ID
	instanceID := ""
	if vm.Properties != nil && vm.Properties.VMID != nil {
		instanceID = *vm.Properties.VMID
	} else if vm.ID != nil {
		instanceID = extractResourceName(*vm.ID)
	}

	// Get region/location
	region := ptrToString(vm.Location)

	return models.NormalizedAsset{
		Platform:     models.PlatformAzure,
		Account:      c.subscriptionID,
		Region:       region,
		InstanceID:   instanceID,
		Name:         name,
		ImageRef:     imageRef,
		ImageVersion: imageVersion,
		State:        state,
		Tags:         tags,
	}
}

// normalizeVMSSVM normalizes a VMSS VM instance to our asset model.
func (c *Connector) normalizeVMSSVM(vm *armcompute.VirtualMachineScaleSetVM, vmssName, resourceGroup, imageRef, imageVersion string) models.NormalizedAsset {
	// Extract tags
	tags := make(map[string]string)
	if vm.Tags != nil {
		for k, v := range vm.Tags {
			if v != nil {
				tags[k] = *v
			}
		}
	}
	tags["azure:resource_group"] = resourceGroup
	tags["azure:vmss"] = vmssName

	// Determine power state
	state := models.AssetStateUnknown
	if vm.Properties != nil && vm.Properties.InstanceView != nil {
		for _, status := range vm.Properties.InstanceView.Statuses {
			if status.Code != nil && strings.HasPrefix(*status.Code, "PowerState/") {
				powerState := strings.TrimPrefix(*status.Code, "PowerState/")
				switch powerState {
				case "running":
					state = models.AssetStateRunning
				case "stopped", "deallocated":
					state = models.AssetStateStopped
				case "starting", "stopping":
					state = models.AssetStatePending
				}
				break
			}
		}
	}

	// Get VM name (includes VMSS name and instance ID)
	name := ptrToString(vm.Name)

	// Get instance ID
	instanceID := ""
	if vm.Properties != nil && vm.Properties.VMID != nil {
		instanceID = *vm.Properties.VMID
	} else if vm.InstanceID != nil {
		instanceID = fmt.Sprintf("%s_%s", vmssName, *vm.InstanceID)
	}

	// Get region/location
	region := ptrToString(vm.Location)

	return models.NormalizedAsset{
		Platform:     models.PlatformAzure,
		Account:      c.subscriptionID,
		Region:       region,
		InstanceID:   instanceID,
		Name:         name,
		ImageRef:     imageRef,
		ImageVersion: imageVersion,
		State:        state,
		Tags:         tags,
	}
}

// extractImageReference extracts image reference and version from Azure ImageReference.
func (c *Connector) extractImageReference(imgRef *armcompute.ImageReference) (string, string) {
	if imgRef == nil {
		return "", ""
	}

	// Shared Image Gallery reference
	if imgRef.ID != nil {
		// ID format: /subscriptions/.../resourceGroups/.../providers/Microsoft.Compute/galleries/.../images/.../versions/...
		id := *imgRef.ID
		parts := strings.Split(id, "/")

		// Extract version if present
		var version string
		for i, part := range parts {
			if strings.EqualFold(part, "versions") && i+1 < len(parts) {
				version = parts[i+1]
				break
			}
		}

		// Extract image name
		imageName := extractResourceName(id)
		if version != "" {
			// Remove version from image name if it's there
			imageName = strings.TrimSuffix(imageName, "/versions/"+version)
		}

		return imageName, version
	}

	// Marketplace image
	if imgRef.Publisher != nil {
		imageRef := fmt.Sprintf("%s:%s:%s",
			ptrToString(imgRef.Publisher),
			ptrToString(imgRef.Offer),
			ptrToString(imgRef.SKU),
		)
		version := ptrToString(imgRef.Version)
		return imageRef, version
	}

	return "", ""
}

// DiscoverImages discovers all custom images and Shared Image Gallery images in the subscription.
func (c *Connector) DiscoverImages(ctx context.Context) ([]connector.ImageInfo, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	var images []connector.ImageInfo

	// Discover managed images
	managedImages, err := c.discoverManagedImages(ctx)
	if err != nil {
		c.log.Error("failed to discover managed images", "error", err)
	} else {
		images = append(images, managedImages...)
	}

	// Discover Shared Image Gallery images
	sigImages, err := c.discoverGalleryImages(ctx)
	if err != nil {
		c.log.Error("failed to discover gallery images", "error", err)
	} else {
		images = append(images, sigImages...)
	}

	c.log.Info("image discovery completed",
		"total", len(images),
		"managed", len(managedImages),
		"gallery", len(sigImages),
	)

	return images, nil
}

// discoverManagedImages discovers Azure managed images.
func (c *Connector) discoverManagedImages(ctx context.Context) ([]connector.ImageInfo, error) {
	var images []connector.ImageInfo

	pager := c.imagesClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list images: %w", err)
		}

		for _, img := range page.Value {
			tags := make(map[string]string)
			if img.Tags != nil {
				for k, v := range img.Tags {
					if v != nil {
						tags[k] = *v
					}
				}
			}
			tags["azure:image_type"] = "managed"

			images = append(images, connector.ImageInfo{
				Platform:    models.PlatformAzure,
				Identifier:  ptrToString(img.ID),
				Name:        ptrToString(img.Name),
				Region:      ptrToString(img.Location),
				CreatedAt:   "",
				Description: "",
				Tags:        tags,
			})
		}
	}

	return images, nil
}

// discoverGalleryImages discovers images from Azure Shared Image Galleries.
func (c *Connector) discoverGalleryImages(ctx context.Context) ([]connector.ImageInfo, error) {
	var images []connector.ImageInfo

	// List all galleries in the subscription
	galleryPager := c.galleriesClient.NewListPager(nil)

	for galleryPager.More() {
		galleryPage, err := galleryPager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list galleries: %w", err)
		}

		for _, gallery := range galleryPage.Value {
			galleryName := ptrToString(gallery.Name)
			resourceGroup := extractResourceGroupFromID(ptrToString(gallery.ID))

			// List images in this gallery
			imagePager := c.galleryImagesClient.NewListByGalleryPager(resourceGroup, galleryName, nil)

			for imagePager.More() {
				imagePage, err := imagePager.NextPage(ctx)
				if err != nil {
					c.log.Warn("failed to list gallery images",
						"gallery", galleryName,
						"error", err,
					)
					break
				}

				for _, img := range imagePage.Value {
					imageName := ptrToString(img.Name)

					// List versions of this image
					versionPager := c.galleryImageVersionsClient.NewListByGalleryImagePager(
						resourceGroup, galleryName, imageName, nil,
					)

					for versionPager.More() {
						versionPage, err := versionPager.NextPage(ctx)
						if err != nil {
							c.log.Warn("failed to list image versions",
								"gallery", galleryName,
								"image", imageName,
								"error", err,
							)
							break
						}

						for _, version := range versionPage.Value {
							tags := make(map[string]string)
							if version.Tags != nil {
								for k, v := range version.Tags {
									if v != nil {
										tags[k] = *v
									}
								}
							}
							tags["azure:image_type"] = "gallery"
							tags["azure:gallery"] = galleryName

							// Extract version name
							versionName := ptrToString(version.Name)

							// Get creation date if available
							createdAt := ""
							if version.Properties != nil && version.Properties.PublishingProfile != nil {
								if version.Properties.PublishingProfile.PublishedDate != nil {
									createdAt = version.Properties.PublishingProfile.PublishedDate.String()
								}
							}

							// Get OS type if available
							osType := ""
							if img.Properties != nil && img.Properties.OSType != nil {
								osType = string(*img.Properties.OSType)
							}
							if osType != "" {
								tags["azure:os_type"] = osType
							}

							images = append(images, connector.ImageInfo{
								Platform:    models.PlatformAzure,
								Identifier:  ptrToString(version.ID),
								Name:        fmt.Sprintf("%s/%s", imageName, versionName),
								Region:      ptrToString(gallery.Location),
								CreatedAt:   createdAt,
								Description: ptrToString(img.Properties.Description),
								Tags:        tags,
							})
						}
					}
				}
			}
		}
	}

	return images, nil
}

// Helper functions

func extractResourceGroupFromID(id string) string {
	// ID is like: /subscriptions/.../resourceGroups/myRG/providers/Microsoft.Compute/virtualMachines/myVM
	parts := strings.Split(id, "/")
	for i, part := range parts {
		if strings.EqualFold(part, "resourceGroups") && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func extractResourceName(id string) string {
	// ID is like: /subscriptions/.../resourceGroups/myRG/providers/Microsoft.Compute/virtualMachines/myVM
	parts := strings.Split(id, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return id
}

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrString(s string) *string {
	return &s
}
