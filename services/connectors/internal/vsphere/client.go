// Package vsphere provides vSphere connector functionality.
package vsphere

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/connector"
)

// Connector implements the vSphere platform connector.
type Connector struct {
	cfg       Config
	client    *govmomi.Client
	log       *logger.Logger
	connected bool
}

// Config holds vSphere-specific configuration.
type Config struct {
	URL      string // e.g., https://vcenter.example.com/sdk
	User     string
	Password string
	Insecure bool // Skip TLS verification
}

// New creates a new vSphere connector.
func New(cfg Config, log *logger.Logger) *Connector {
	return &Connector{
		cfg: cfg,
		log: log.WithComponent("vsphere-connector"),
	}
}

// Name returns the connector name.
func (c *Connector) Name() string {
	return "vsphere"
}

// Platform returns the platform type.
func (c *Connector) Platform() models.Platform {
	return models.PlatformVSphere
}

// Connect establishes a connection to vSphere.
func (c *Connector) Connect(ctx context.Context) error {
	// Parse URL
	u, err := url.Parse(c.cfg.URL)
	if err != nil {
		return fmt.Errorf("failed to parse vSphere URL: %w", err)
	}

	// Set credentials
	u.User = url.UserPassword(c.cfg.User, c.cfg.Password)

	// Create client
	client, err := govmomi.NewClient(ctx, u, c.cfg.Insecure)
	if err != nil {
		return fmt.Errorf("failed to connect to vSphere: %w", err)
	}

	c.client = client
	c.connected = true

	c.log.Info("connected to vSphere",
		"url", c.cfg.URL,
		"insecure", c.cfg.Insecure,
	)

	return nil
}

// Close closes the vSphere connection.
func (c *Connector) Close() error {
	if c.client != nil {
		ctx := context.Background()
		if err := c.client.Logout(ctx); err != nil {
			c.log.Warn("failed to logout from vSphere", "error", err)
		}
	}
	c.connected = false
	return nil
}

// Health checks the health of the vSphere connection.
func (c *Connector) Health(ctx context.Context) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	// Check if session is valid
	if !c.client.IsVC() {
		return fmt.Errorf("not connected to vCenter")
	}

	// Try to get about info - if we can access ServiceContent, connection is healthy
	_ = c.client.Client.ServiceContent.About.FullName

	return nil
}

// DiscoverAssets discovers all Virtual Machines from vSphere.
func (c *Connector) DiscoverAssets(ctx context.Context, orgID uuid.UUID) ([]models.NormalizedAsset, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	// Create a view manager
	viewMgr := view.NewManager(c.client.Client)

	// Create a container view of all VMs
	containerView, err := viewMgr.CreateContainerView(
		ctx,
		c.client.Client.ServiceContent.RootFolder,
		[]string{"VirtualMachine"},
		true, // recursive
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create container view: %w", err)
	}
	defer containerView.Destroy(ctx)

	// Retrieve VM properties
	var vms []mo.VirtualMachine
	err = containerView.Retrieve(
		ctx,
		[]string{"VirtualMachine"},
		[]string{
			"name",
			"config",
			"runtime",
			"guest",
			"summary",
			"parent",
		},
		&vms,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve VMs: %w", err)
	}

	// Get datacenter mapping for region info
	dcMap, err := c.getDatacenterMap(ctx)
	if err != nil {
		c.log.Warn("failed to get datacenter map", "error", err)
		dcMap = make(map[string]string)
	}

	var allAssets []models.NormalizedAsset
	for _, vm := range vms {
		// Skip templates
		if vm.Config != nil && vm.Config.Template {
			continue
		}

		asset := c.normalizeVM(vm, dcMap)
		allAssets = append(allAssets, asset)
	}

	c.log.Info("asset discovery completed",
		"total_assets", len(allAssets),
		"vcenter", c.cfg.URL,
	)

	return allAssets, nil
}

func (c *Connector) normalizeVM(vm mo.VirtualMachine, dcMap map[string]string) models.NormalizedAsset {
	// Extract tags/custom attributes (using summary annotations if available)
	tags := make(map[string]string)
	if vm.Summary.Config.Annotation != "" {
		tags["annotation"] = vm.Summary.Config.Annotation
	}
	if vm.Config != nil && vm.Config.ExtraConfig != nil {
		for _, opt := range vm.Config.ExtraConfig {
			if bov, ok := opt.(*types.OptionValue); ok {
				// Only include custom guestinfo/tags
				if strings.HasPrefix(bov.Key, "guestinfo.") {
					key := strings.TrimPrefix(bov.Key, "guestinfo.")
					if strVal, ok := bov.Value.(string); ok {
						tags[key] = strVal
					}
				}
			}
		}
	}

	// Map vSphere power state to our state
	state := models.AssetStateUnknown
	if vm.Runtime.PowerState != "" {
		switch vm.Runtime.PowerState {
		case types.VirtualMachinePowerStatePoweredOn:
			state = models.AssetStateRunning
		case types.VirtualMachinePowerStatePoweredOff:
			state = models.AssetStateStopped
		case types.VirtualMachinePowerStateSuspended:
			state = models.AssetStateStopped
		}
	}

	// Get image reference (guest OS ID or template name)
	imageRef := ""
	if vm.Config != nil {
		if vm.Config.GuestId != "" {
			imageRef = vm.Config.GuestId
		}
	}

	// Get VM name
	name := vm.Name

	// Get instance ID (MoRef ID)
	instanceID := vm.Self.Value

	// Get region from datacenter
	region := "unknown"
	if vm.Parent != nil {
		// Try to find datacenter from parent chain
		parentRef := vm.Parent.Value
		if dc, ok := dcMap[parentRef]; ok {
			region = dc
		}
	}

	// Try to get host/cluster info for better region mapping
	if vm.Runtime.Host != nil {
		hostRef := vm.Runtime.Host.Value
		if dc, ok := dcMap[hostRef]; ok {
			region = dc
		}
	}

	return models.NormalizedAsset{
		Platform:     models.PlatformVSphere,
		Account:      extractHostFromURL(c.cfg.URL),
		Region:       region,
		InstanceID:   instanceID,
		Name:         name,
		ImageRef:     imageRef,
		ImageVersion: "",
		State:        state,
		Tags:         tags,
	}
}

// getDatacenterMap creates a mapping of managed object references to datacenter names.
func (c *Connector) getDatacenterMap(ctx context.Context) (map[string]string, error) {
	finder := find.NewFinder(c.client.Client, true)

	dcs, err := finder.DatacenterList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("failed to find datacenters: %w", err)
	}

	dcMap := make(map[string]string)
	pc := property.DefaultCollector(c.client.Client)

	for _, dc := range dcs {
		var moDC mo.Datacenter
		err := pc.RetrieveOne(ctx, dc.Reference(), []string{"name", "hostFolder", "vmFolder"}, &moDC)
		if err != nil {
			continue
		}

		dcName := moDC.Name
		dcMap[dc.Reference().Value] = dcName

		// Map host folder and vm folder to datacenter
		if moDC.HostFolder.Value != "" {
			dcMap[moDC.HostFolder.Value] = dcName
		}
		if moDC.VmFolder.Value != "" {
			dcMap[moDC.VmFolder.Value] = dcName
		}
	}

	return dcMap, nil
}

// DiscoverImages discovers all VM templates in vSphere.
func (c *Connector) DiscoverImages(ctx context.Context) ([]connector.ImageInfo, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	// Create a view manager
	viewMgr := view.NewManager(c.client.Client)

	// Create a container view of all VMs (templates are also VMs)
	containerView, err := viewMgr.CreateContainerView(
		ctx,
		c.client.Client.ServiceContent.RootFolder,
		[]string{"VirtualMachine"},
		true,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create container view: %w", err)
	}
	defer containerView.Destroy(ctx)

	// Retrieve VM properties
	var vms []mo.VirtualMachine
	err = containerView.Retrieve(
		ctx,
		[]string{"VirtualMachine"},
		[]string{"name", "config", "summary"},
		&vms,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve VMs: %w", err)
	}

	var images []connector.ImageInfo
	for _, vm := range vms {
		// Only include templates
		if vm.Config == nil || !vm.Config.Template {
			continue
		}

		tags := make(map[string]string)
		if vm.Summary.Config.Annotation != "" {
			tags["annotation"] = vm.Summary.Config.Annotation
		}
		if vm.Config.GuestId != "" {
			tags["guestId"] = vm.Config.GuestId
		}

		images = append(images, connector.ImageInfo{
			Platform:    models.PlatformVSphere,
			Identifier:  vm.Self.Value,
			Name:        vm.Name,
			Region:      "datacenter", // Would require additional lookup
			CreatedAt:   "",           // vSphere doesn't expose template creation time easily
			Description: vm.Summary.Config.Annotation,
			Tags:        tags,
		})
	}

	c.log.Info("image discovery completed", "count", len(images))

	return images, nil
}

// Helper functions

func extractHostFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Host
}
