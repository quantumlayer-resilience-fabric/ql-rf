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
	finder    *find.Finder
	log       *logger.Logger
	connected bool
}

// Config holds vSphere-specific configuration.
type Config struct {
	URL         string   // e.g., https://vcenter.example.com/sdk
	User        string
	Password    string
	Insecure    bool     // Skip TLS verification
	Datacenters []string // Optional: filter to specific datacenters
	Clusters    []string // Optional: filter to specific clusters
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
	c.finder = find.NewFinder(client.Client, true)
	c.connected = true

	c.log.Info("connected to vSphere",
		"url", c.cfg.URL,
		"insecure", c.cfg.Insecure,
		"datacenter_filter", len(c.cfg.Datacenters) > 0,
		"cluster_filter", len(c.cfg.Clusters) > 0,
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

	// Get datacenter mapping and cluster info for filtering
	dcMap, clusterMap, err := c.buildInventoryMaps(ctx)
	if err != nil {
		c.log.Warn("failed to build inventory maps", "error", err)
		dcMap = make(map[string]string)
		clusterMap = make(map[string]string)
	}

	// Get resource pool info for additional metadata
	rpMap, err := c.getResourcePoolMap(ctx)
	if err != nil {
		c.log.Warn("failed to get resource pool map", "error", err)
		rpMap = make(map[string]resourcePoolInfo)
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
			"resourcePool",
		},
		&vms,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve VMs: %w", err)
	}

	var allAssets []models.NormalizedAsset
	for _, vm := range vms {
		// Skip templates
		if vm.Config != nil && vm.Config.Template {
			continue
		}

		// Get datacenter for this VM
		dcName := c.getVMDatacenter(vm, dcMap)

		// Apply datacenter filter if configured
		if !c.isDatacenterAllowed(dcName) {
			continue
		}

		// Get cluster for this VM
		clusterName := c.getVMCluster(vm, clusterMap)

		// Apply cluster filter if configured
		if !c.isClusterAllowed(clusterName) {
			continue
		}

		asset := c.normalizeVM(vm, dcName, clusterName, rpMap)
		allAssets = append(allAssets, asset)
	}

	c.log.Info("asset discovery completed",
		"total_assets", len(allAssets),
		"vcenter", c.cfg.URL,
	)

	return allAssets, nil
}

// buildInventoryMaps creates mappings for datacenters and clusters.
func (c *Connector) buildInventoryMaps(ctx context.Context) (map[string]string, map[string]string, error) {
	dcMap := make(map[string]string)
	clusterMap := make(map[string]string)

	dcs, err := c.finder.DatacenterList(ctx, "*")
	if err != nil {
		return dcMap, clusterMap, fmt.Errorf("failed to find datacenters: %w", err)
	}

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

		// Get clusters in this datacenter
		c.finder.SetDatacenter(dc)
		clusters, err := c.finder.ClusterComputeResourceList(ctx, "*")
		if err != nil {
			c.log.Debug("no clusters found in datacenter", "datacenter", dcName)
			continue
		}

		for _, cluster := range clusters {
			var moCluster mo.ClusterComputeResource
			err := pc.RetrieveOne(ctx, cluster.Reference(), []string{"name", "host"}, &moCluster)
			if err != nil {
				continue
			}

			clusterMap[cluster.Reference().Value] = moCluster.Name

			// Map hosts to their cluster
			for _, hostRef := range moCluster.Host {
				clusterMap[hostRef.Value] = moCluster.Name
				dcMap[hostRef.Value] = dcName
			}
		}
	}

	return dcMap, clusterMap, nil
}

// resourcePoolInfo holds resource pool information.
type resourcePoolInfo struct {
	Name    string
	Cluster string
}

// getResourcePoolMap creates a mapping of resource pool references to their info.
func (c *Connector) getResourcePoolMap(ctx context.Context) (map[string]resourcePoolInfo, error) {
	rpMap := make(map[string]resourcePoolInfo)

	viewMgr := view.NewManager(c.client.Client)
	containerView, err := viewMgr.CreateContainerView(
		ctx,
		c.client.Client.ServiceContent.RootFolder,
		[]string{"ResourcePool"},
		true,
	)
	if err != nil {
		return rpMap, err
	}
	defer containerView.Destroy(ctx)

	var rps []mo.ResourcePool
	err = containerView.Retrieve(ctx, []string{"ResourcePool"}, []string{"name", "parent"}, &rps)
	if err != nil {
		return rpMap, err
	}

	for _, rp := range rps {
		rpMap[rp.Self.Value] = resourcePoolInfo{
			Name: rp.Name,
		}
	}

	return rpMap, nil
}

// getVMDatacenter determines the datacenter for a VM.
func (c *Connector) getVMDatacenter(vm mo.VirtualMachine, dcMap map[string]string) string {
	// Try host first
	if vm.Runtime.Host != nil {
		if dc, ok := dcMap[vm.Runtime.Host.Value]; ok {
			return dc
		}
	}
	// Try parent folder
	if vm.Parent != nil {
		if dc, ok := dcMap[vm.Parent.Value]; ok {
			return dc
		}
	}
	return "unknown"
}

// getVMCluster determines the cluster for a VM.
func (c *Connector) getVMCluster(vm mo.VirtualMachine, clusterMap map[string]string) string {
	if vm.Runtime.Host != nil {
		if cluster, ok := clusterMap[vm.Runtime.Host.Value]; ok {
			return cluster
		}
	}
	return ""
}

// isDatacenterAllowed checks if a datacenter is in the allowed list.
func (c *Connector) isDatacenterAllowed(dc string) bool {
	if len(c.cfg.Datacenters) == 0 {
		return true
	}
	for _, allowed := range c.cfg.Datacenters {
		if strings.EqualFold(allowed, dc) {
			return true
		}
	}
	return false
}

// isClusterAllowed checks if a cluster is in the allowed list.
func (c *Connector) isClusterAllowed(cluster string) bool {
	if len(c.cfg.Clusters) == 0 {
		return true
	}
	for _, allowed := range c.cfg.Clusters {
		if strings.EqualFold(allowed, cluster) {
			return true
		}
	}
	return false
}

func (c *Connector) normalizeVM(vm mo.VirtualMachine, datacenter, cluster string, rpMap map[string]resourcePoolInfo) models.NormalizedAsset {
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

	// Add datacenter and cluster info
	tags["datacenter"] = datacenter
	if cluster != "" {
		tags["cluster"] = cluster
	}

	// Add resource pool info
	if vm.ResourcePool != nil {
		if rpInfo, ok := rpMap[vm.ResourcePool.Value]; ok {
			tags["resource_pool"] = rpInfo.Name
		}
	}

	// Add host info
	if vm.Runtime.Host != nil {
		tags["host"] = vm.Runtime.Host.Value
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

	// Get image reference (guest OS ID)
	imageRef := ""
	imageVersion := ""
	if vm.Config != nil {
		if vm.Config.GuestId != "" {
			imageRef = vm.Config.GuestId
		}
		// Try to get guest OS full name for version
		if vm.Config.GuestFullName != "" {
			imageVersion = vm.Config.GuestFullName
		}
	}

	// Add hardware info
	if vm.Config != nil {
		tags["num_cpu"] = fmt.Sprintf("%d", vm.Config.Hardware.NumCPU)
		tags["memory_mb"] = fmt.Sprintf("%d", vm.Config.Hardware.MemoryMB)
		if vm.Config.Hardware.NumCoresPerSocket > 0 {
			tags["cores_per_socket"] = fmt.Sprintf("%d", vm.Config.Hardware.NumCoresPerSocket)
		}
	}

	// Add guest info if available
	if vm.Guest != nil {
		if vm.Guest.IpAddress != "" {
			tags["ip_address"] = vm.Guest.IpAddress
		}
		if vm.Guest.HostName != "" {
			tags["hostname"] = vm.Guest.HostName
		}
		if vm.Guest.ToolsStatus != "" {
			tags["vmtools_status"] = string(vm.Guest.ToolsStatus)
		}
	}

	// Get VM name
	name := vm.Name

	// Get instance ID (MoRef ID)
	instanceID := vm.Self.Value

	return models.NormalizedAsset{
		Platform:     models.PlatformVSphere,
		Account:      extractHostFromURL(c.cfg.URL),
		Region:       datacenter,
		InstanceID:   instanceID,
		Name:         name,
		ImageRef:     imageRef,
		ImageVersion: imageVersion,
		State:        state,
		Tags:         tags,
	}
}

// DiscoverImages discovers all VM templates in vSphere.
func (c *Connector) DiscoverImages(ctx context.Context) ([]connector.ImageInfo, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	// Build inventory maps for datacenter lookup
	dcMap, _, err := c.buildInventoryMaps(ctx)
	if err != nil {
		c.log.Warn("failed to build inventory maps for images", "error", err)
		dcMap = make(map[string]string)
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
		[]string{"name", "config", "summary", "parent", "runtime"},
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
			tags["guest_id"] = vm.Config.GuestId
		}
		if vm.Config.GuestFullName != "" {
			tags["guest_os"] = vm.Config.GuestFullName
		}

		// Add hardware info
		tags["num_cpu"] = fmt.Sprintf("%d", vm.Config.Hardware.NumCPU)
		tags["memory_mb"] = fmt.Sprintf("%d", vm.Config.Hardware.MemoryMB)

		// Determine datacenter
		datacenter := c.getVMDatacenter(vm, dcMap)

		// Apply datacenter filter if configured
		if !c.isDatacenterAllowed(datacenter) {
			continue
		}

		tags["datacenter"] = datacenter

		images = append(images, connector.ImageInfo{
			Platform:    models.PlatformVSphere,
			Identifier:  vm.Self.Value,
			Name:        vm.Name,
			Region:      datacenter,
			CreatedAt:   "", // vSphere doesn't expose template creation time easily
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
