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
	cfg          Config
	vmClient     *armcompute.VirtualMachinesClient
	imagesClient *armcompute.ImagesClient
	log          *logger.Logger
	connected    bool
}

// Config holds Azure-specific configuration.
type Config struct {
	TenantID       string
	ClientID       string
	ClientSecret   string
	SubscriptionID string
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

	// Create images client
	imagesClient, err := armcompute.NewImagesClient(c.cfg.SubscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create images client: %w", err)
	}

	c.vmClient = vmClient
	c.imagesClient = imagesClient
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

// DiscoverAssets discovers all Virtual Machines from Azure.
func (c *Connector) DiscoverAssets(ctx context.Context, orgID uuid.UUID) ([]models.NormalizedAsset, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	var allAssets []models.NormalizedAsset

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
			// Get instance view for power state
			resourceGroup := extractResourceGroupFromID(ptrToString(vm.ID))
			vmName := ptrToString(vm.Name)

			instanceView, err := c.vmClient.InstanceView(ctx, resourceGroup, vmName, nil)
			if err != nil {
				c.log.Warn("failed to get instance view",
					"vm", vmName,
					"resource_group", resourceGroup,
					"error", err,
				)
			}

			asset := c.normalizeVM(vm, &instanceView.VirtualMachineInstanceView)
			allAssets = append(allAssets, asset)
		}
	}

	c.log.Info("asset discovery completed",
		"total_assets", len(allAssets),
		"subscription", c.cfg.SubscriptionID,
	)

	return allAssets, nil
}

func (c *Connector) normalizeVM(vm *armcompute.VirtualMachine, instanceView *armcompute.VirtualMachineInstanceView) models.NormalizedAsset {
	// Extract tags
	tags := make(map[string]string)
	if vm.Tags != nil {
		for k, v := range vm.Tags {
			if v != nil {
				tags[k] = *v
			}
		}
	}

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
	imageRef := ""
	if vm.Properties != nil && vm.Properties.StorageProfile != nil && vm.Properties.StorageProfile.ImageReference != nil {
		imgRef := vm.Properties.StorageProfile.ImageReference
		if imgRef.ID != nil {
			// Custom image
			imageRef = extractResourceName(ptrToString(imgRef.ID))
		} else if imgRef.Publisher != nil {
			// Marketplace image
			imageRef = fmt.Sprintf("%s:%s:%s:%s",
				ptrToString(imgRef.Publisher),
				ptrToString(imgRef.Offer),
				ptrToString(imgRef.SKU),
				ptrToString(imgRef.Version),
			)
		}
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
		Account:      c.cfg.SubscriptionID,
		Region:       region,
		InstanceID:   instanceID,
		Name:         name,
		ImageRef:     imageRef,
		ImageVersion: "", // Version is part of imageRef for marketplace images
		State:        state,
		Tags:         tags,
	}
}

// DiscoverImages discovers all custom images in the subscription.
func (c *Connector) DiscoverImages(ctx context.Context) ([]connector.ImageInfo, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	var images []connector.ImageInfo

	pager := c.imagesClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list images: %w", err)
		}

		for _, img := range page.Value {
			// Extract tags
			tags := make(map[string]string)
			if img.Tags != nil {
				for k, v := range img.Tags {
					if v != nil {
						tags[k] = *v
					}
				}
			}

			name := ptrToString(img.Name)
			location := ptrToString(img.Location)

			images = append(images, connector.ImageInfo{
				Platform:    models.PlatformAzure,
				Identifier:  ptrToString(img.ID),
				Name:        name,
				Region:      location,
				CreatedAt:   "", // Azure images don't have a simple creation timestamp in this API
				Description: "", // No description field in armcompute.Image
				Tags:        tags,
			})
		}
	}

	c.log.Info("image discovery completed", "count", len(images))

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
