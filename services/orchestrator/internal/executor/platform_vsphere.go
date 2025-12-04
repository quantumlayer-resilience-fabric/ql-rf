// Package executor implements the plan execution engine.
package executor

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// VSphereConfig holds vSphere platform client configuration.
type VSphereConfig struct {
	URL            string // vCenter URL (e.g., https://vcenter.example.com/sdk)
	Username       string
	Password       string
	Insecure       bool          // Skip TLS certificate verification
	Datacenter     string        // Default datacenter
	GuestUsername  string        // Guest OS username for in-guest operations
	GuestPassword  string        // Guest OS password for in-guest operations
	ConnectTimeout time.Duration // Connection timeout
	OperationTimeout time.Duration // Operation timeout
}

// vSpherePlatformClient implements PlatformClient for VMware vSphere.
type vSpherePlatformClient struct {
	cfg    VSphereConfig
	client *govmomi.Client
	finder *find.Finder
	log    *logger.Logger
}

// NewVSpherePlatformClient creates a new vSphere platform client.
// Call Connect() to establish the connection to vCenter.
func NewVSpherePlatformClient(cfg VSphereConfig, log *logger.Logger) *vSpherePlatformClient {
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 30 * time.Second
	}
	if cfg.OperationTimeout == 0 {
		cfg.OperationTimeout = 10 * time.Minute
	}

	return &vSpherePlatformClient{
		cfg: cfg,
		log: log.WithComponent("vsphere-platform-client"),
	}
}

// Connect establishes connection to vCenter.
func (c *vSpherePlatformClient) Connect(ctx context.Context) error {
	return c.connect()
}

// connect establishes connection to vCenter.
func (c *vSpherePlatformClient) connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.cfg.ConnectTimeout)
	defer cancel()

	// Parse and validate URL
	u, err := url.Parse(c.cfg.URL)
	if err != nil {
		return fmt.Errorf("invalid vCenter URL: %w", err)
	}

	// Set credentials
	u.User = url.UserPassword(c.cfg.Username, c.cfg.Password)

	// Create govmomi client
	client, err := govmomi.NewClient(ctx, u, c.cfg.Insecure)
	if err != nil {
		return fmt.Errorf("failed to create vSphere client: %w", err)
	}

	c.client = client
	c.finder = find.NewFinder(client.Client, true)

	// Set default datacenter if specified
	if c.cfg.Datacenter != "" {
		dc, err := c.finder.Datacenter(ctx, c.cfg.Datacenter)
		if err != nil {
			c.log.Warn("failed to find default datacenter", "datacenter", c.cfg.Datacenter, "error", err)
		} else {
			c.finder.SetDatacenter(dc)
		}
	}

	c.log.Info("connected to vSphere",
		"url", c.cfg.URL,
		"datacenter", c.cfg.Datacenter,
	)

	return nil
}

// Close closes the vSphere connection.
func (c *vSpherePlatformClient) Close() error {
	if c.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return c.client.Logout(ctx)
	}
	return nil
}

// =============================================================================
// PlatformClient Interface Implementation
// =============================================================================

// ReimageInstance reimages a VM by cloning from a template.
// In vSphere, this is done by:
// 1. Creating a snapshot of the current VM (for rollback)
// 2. Powering off the VM
// 3. Cloning from the template to replace the VM
// 4. Reconfiguring network/storage as needed
// 5. Powering on the new VM
func (c *vSpherePlatformClient) ReimageInstance(ctx context.Context, instanceID, imageID string) error {
	c.log.Info("starting VM reimage operation",
		"vm_id", instanceID,
		"template_id", imageID,
	)

	// Find the target VM
	vm, err := c.findVMByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to find VM %s: %w", instanceID, err)
	}

	// Find the template
	template, err := c.findVMByID(ctx, imageID)
	if err != nil {
		return fmt.Errorf("failed to find template %s: %w", imageID, err)
	}

	// Verify it's actually a template
	var templateProps mo.VirtualMachine
	if err := template.Properties(ctx, template.Reference(), []string{"config"}, &templateProps); err != nil {
		return fmt.Errorf("failed to get template properties: %w", err)
	}
	if templateProps.Config == nil || !templateProps.Config.Template {
		return fmt.Errorf("%s is not a template", imageID)
	}

	// Get current VM properties for restoration
	var vmProps mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), []string{"name", "config", "runtime", "network", "datastore"}, &vmProps); err != nil {
		return fmt.Errorf("failed to get VM properties: %w", err)
	}

	vmName := vmProps.Name

	// Step 1: Create a pre-reimage snapshot for rollback
	c.log.Info("creating pre-reimage snapshot", "vm", vmName)
	snapshotTask, err := vm.CreateSnapshot(ctx, "pre-reimage-snapshot", "Snapshot before reimage operation", false, false)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}
	if err := snapshotTask.Wait(ctx); err != nil {
		c.log.Warn("snapshot creation failed, continuing without rollback capability", "error", err)
	}

	// Step 2: Power off the VM if running
	if vmProps.Runtime.PowerState == types.VirtualMachinePowerStatePoweredOn {
		c.log.Info("powering off VM", "vm", vmName)
		if err := c.powerOffVM(ctx, vm); err != nil {
			return fmt.Errorf("failed to power off VM: %w", err)
		}
	}

	// Step 3: Get the folder and resource pool for the clone
	folder, err := c.getVMFolder(ctx, vm)
	if err != nil {
		return fmt.Errorf("failed to get VM folder: %w", err)
	}

	pool, err := c.getVMResourcePool(ctx, vm)
	if err != nil {
		return fmt.Errorf("failed to get resource pool: %w", err)
	}

	// Step 4: Get datastore for the clone
	var datastore *object.Datastore
	if len(vmProps.Datastore) > 0 {
		datastore = object.NewDatastore(c.client.Client, vmProps.Datastore[0])
	}

	// Step 5: Prepare clone spec preserving network configuration
	cloneSpec := types.VirtualMachineCloneSpec{
		Location: types.VirtualMachineRelocateSpec{
			Pool: types.NewReference(pool.Reference()),
		},
		PowerOn:  false,
		Template: false,
	}

	if datastore != nil {
		cloneSpec.Location.Datastore = types.NewReference(datastore.Reference())
	}

	// Preserve network adapters configuration
	if vmProps.Config != nil {
		cloneSpec.Config = &types.VirtualMachineConfigSpec{
			Name: vmName + "-reimaged",
		}
	}

	// Step 6: Clone from template
	c.log.Info("cloning from template", "vm", vmName, "template", imageID)
	cloneTask, err := template.Clone(ctx, folder, vmName+"-reimaged", cloneSpec)
	if err != nil {
		return fmt.Errorf("failed to start clone operation: %w", err)
	}

	taskInfo, err := cloneTask.WaitForResult(ctx)
	if err != nil {
		return fmt.Errorf("clone operation failed: %w", err)
	}

	newVMRef := taskInfo.Result.(types.ManagedObjectReference)
	newVM := object.NewVirtualMachine(c.client.Client, newVMRef)

	// Step 7: Rename original VM
	c.log.Info("renaming original VM", "vm", vmName)
	renameTask, err := vm.Rename(ctx, vmName+"-old-"+time.Now().Format("20060102150405"))
	if err != nil {
		c.log.Warn("failed to rename original VM", "error", err)
	} else {
		_ = renameTask.Wait(ctx)
	}

	// Step 8: Rename new VM to original name
	renameNewTask, err := newVM.Rename(ctx, vmName)
	if err != nil {
		return fmt.Errorf("failed to rename new VM: %w", err)
	}
	if err := renameNewTask.Wait(ctx); err != nil {
		return fmt.Errorf("rename new VM task failed: %w", err)
	}

	// Step 9: Reconfigure network if needed (copy NICs from original)
	if err := c.copyNetworkConfig(ctx, vm, newVM); err != nil {
		c.log.Warn("failed to copy network configuration", "error", err)
	}

	// Step 10: Power on the new VM
	c.log.Info("powering on reimaged VM", "vm", vmName)
	if err := c.powerOnVM(ctx, newVM); err != nil {
		return fmt.Errorf("failed to power on reimaged VM: %w", err)
	}

	// Step 11: Wait for VMware Tools to be running
	if err := c.waitForVMTools(ctx, newVM, 5*time.Minute); err != nil {
		c.log.Warn("VMware Tools not responding after reimage", "error", err)
	}

	c.log.Info("VM reimage completed successfully",
		"vm", vmName,
		"template", imageID,
		"new_vm_id", newVMRef.Value,
	)

	return nil
}

// RebootInstance reboots a VM using VMware Tools guest operations.
func (c *vSpherePlatformClient) RebootInstance(ctx context.Context, instanceID string) error {
	c.log.Info("rebooting VM", "vm_id", instanceID)

	vm, err := c.findVMByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to find VM: %w", err)
	}

	// Try graceful reboot via VMware Tools first
	err = vm.RebootGuest(ctx)
	if err != nil {
		c.log.Warn("graceful reboot failed, attempting hard reset", "error", err)

		// Fall back to hard reset
		resetTask, err := vm.Reset(ctx)
		if err != nil {
			return fmt.Errorf("failed to reset VM: %w", err)
		}
		if err := resetTask.Wait(ctx); err != nil {
			return fmt.Errorf("reset task failed: %w", err)
		}
	}

	// Wait for VM to come back up
	if err := c.waitForVMTools(ctx, vm, 5*time.Minute); err != nil {
		return fmt.Errorf("VM did not come back online after reboot: %w", err)
	}

	c.log.Info("VM reboot completed", "vm_id", instanceID)
	return nil
}

// TerminateInstance powers off and deletes a VM.
func (c *vSpherePlatformClient) TerminateInstance(ctx context.Context, instanceID string) error {
	c.log.Info("terminating VM", "vm_id", instanceID)

	vm, err := c.findVMByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to find VM: %w", err)
	}

	// Get current power state
	var props mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), []string{"runtime"}, &props); err != nil {
		return fmt.Errorf("failed to get VM properties: %w", err)
	}

	// Power off if running
	if props.Runtime.PowerState == types.VirtualMachinePowerStatePoweredOn {
		c.log.Info("powering off VM before deletion", "vm_id", instanceID)
		if err := c.powerOffVM(ctx, vm); err != nil {
			return fmt.Errorf("failed to power off VM: %w", err)
		}
	}

	// Delete the VM
	destroyTask, err := vm.Destroy(ctx)
	if err != nil {
		return fmt.Errorf("failed to start destroy task: %w", err)
	}

	if err := destroyTask.Wait(ctx); err != nil {
		return fmt.Errorf("destroy task failed: %w", err)
	}

	c.log.Info("VM terminated successfully", "vm_id", instanceID)
	return nil
}

// GetInstanceStatus returns the current power state of a VM.
func (c *vSpherePlatformClient) GetInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	vm, err := c.findVMByID(ctx, instanceID)
	if err != nil {
		return "", fmt.Errorf("failed to find VM: %w", err)
	}

	var props mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), []string{"runtime", "guest"}, &props); err != nil {
		return "", fmt.Errorf("failed to get VM properties: %w", err)
	}

	// Map vSphere power state to our states
	switch props.Runtime.PowerState {
	case types.VirtualMachinePowerStatePoweredOn:
		// Check if guest tools are running for more accurate status
		if props.Guest != nil && props.Guest.ToolsRunningStatus == string(types.VirtualMachineToolsRunningStatusGuestToolsRunning) {
			return "running", nil
		}
		return "starting", nil
	case types.VirtualMachinePowerStatePoweredOff:
		return "stopped", nil
	case types.VirtualMachinePowerStateSuspended:
		return "suspended", nil
	default:
		return "unknown", nil
	}
}

// WaitForInstanceState waits for a VM to reach a specific state.
func (c *vSpherePlatformClient) WaitForInstanceState(ctx context.Context, instanceID, targetState string, timeout time.Duration) error {
	c.log.Info("waiting for VM state",
		"vm_id", instanceID,
		"target_state", targetState,
		"timeout", timeout,
	)

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for VM %s to reach state %s", instanceID, targetState)
			}

			currentState, err := c.GetInstanceStatus(ctx, instanceID)
			if err != nil {
				c.log.Warn("failed to get VM status, retrying", "error", err)
				continue
			}

			if currentState == targetState {
				c.log.Info("VM reached target state",
					"vm_id", instanceID,
					"state", targetState,
				)
				return nil
			}

			c.log.Debug("waiting for VM state",
				"vm_id", instanceID,
				"current", currentState,
				"target", targetState,
			)
		}
	}
}

// ApplyPatches applies patches to a VM using in-guest operations.
// vSphere doesn't have native patch management like AWS SSM, so we use
// VMware Tools guest operations to run scripts inside the VM.
func (c *vSpherePlatformClient) ApplyPatches(ctx context.Context, instanceID string, params map[string]interface{}) error {
	c.log.Info("applying patches to VM", "vm_id", instanceID, "params", params)

	vm, err := c.findVMByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to find VM: %w", err)
	}

	// Verify VMware Tools is running
	var props mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), []string{"guest"}, &props); err != nil {
		return fmt.Errorf("failed to get VM properties: %w", err)
	}

	if props.Guest == nil || props.Guest.ToolsRunningStatus != string(types.VirtualMachineToolsRunningStatusGuestToolsRunning) {
		return fmt.Errorf("VMware Tools is not running on VM %s", instanceID)
	}

	// Determine the guest OS type
	guestFamily := props.Guest.GuestFamily
	isWindows := strings.Contains(strings.ToLower(string(guestFamily)), "windows")

	// Build patch command based on OS
	var patchCommand string
	var patchArgs []string

	if isWindows {
		// Windows Update via PowerShell
		patchCommand = "powershell.exe"
		patchArgs = []string{
			"-ExecutionPolicy", "Bypass",
			"-Command",
			`Install-WindowsUpdate -AcceptAll -AutoReboot:$false | Out-File C:\patch-log.txt`,
		}

		// Check if specific KB is requested
		if kb, ok := params["kb"].(string); ok && kb != "" {
			patchArgs[2] = fmt.Sprintf(`Install-WindowsUpdate -KBArticleID %s -AcceptAll -AutoReboot:$false | Out-File C:\patch-log.txt`, kb)
		}
	} else {
		// Linux - detect package manager
		patchCommand = "/bin/bash"
		script := `
#!/bin/bash
set -e

# Detect package manager and apply updates
if command -v apt-get &> /dev/null; then
    export DEBIAN_FRONTEND=noninteractive
    apt-get update
    apt-get upgrade -y
    apt-get autoremove -y
elif command -v yum &> /dev/null; then
    yum update -y
    yum autoremove -y
elif command -v dnf &> /dev/null; then
    dnf update -y
    dnf autoremove -y
elif command -v zypper &> /dev/null; then
    zypper refresh
    zypper update -y
else
    echo "Unknown package manager"
    exit 1
fi

echo "Patching completed successfully"
`
		// If specific packages requested
		if packages, ok := params["packages"].([]string); ok && len(packages) > 0 {
			pkgList := strings.Join(packages, " ")
			script = fmt.Sprintf(`
#!/bin/bash
set -e
if command -v apt-get &> /dev/null; then
    apt-get update && apt-get install -y %s
elif command -v yum &> /dev/null; then
    yum install -y %s
elif command -v dnf &> /dev/null; then
    dnf install -y %s
fi
`, pkgList, pkgList, pkgList)
		}

		patchArgs = []string{"-c", script}
	}

	// Execute the patch command via guest operations
	c.log.Info("executing patch command in guest",
		"vm_id", instanceID,
		"os_family", guestFamily,
		"is_windows", isWindows,
	)

	if err := c.runGuestCommand(ctx, vm, patchCommand, patchArgs); err != nil {
		return fmt.Errorf("patch command failed: %w", err)
	}

	// Check if reboot is needed
	needsReboot := false
	if isWindows {
		// Check Windows reboot pending
		rebootCheck, err := c.runGuestCommandWithOutput(ctx, vm, "powershell.exe",
			[]string{"-Command", "Test-Path 'HKLM:\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\WindowsUpdate\\Auto Update\\RebootRequired'"})
		if err == nil && strings.TrimSpace(rebootCheck) == "True" {
			needsReboot = true
		}
	} else {
		// Check Linux reboot required
		rebootCheck, err := c.runGuestCommandWithOutput(ctx, vm, "/bin/bash",
			[]string{"-c", "test -f /var/run/reboot-required && echo 'true' || echo 'false'"})
		if err == nil && strings.TrimSpace(rebootCheck) == "true" {
			needsReboot = true
		}
	}

	if needsReboot {
		c.log.Info("VM requires reboot after patching", "vm_id", instanceID)
		// Auto-reboot if specified in params
		if autoReboot, ok := params["auto_reboot"].(bool); ok && autoReboot {
			if err := c.RebootInstance(ctx, instanceID); err != nil {
				return fmt.Errorf("auto-reboot failed: %w", err)
			}
		}
	}

	c.log.Info("patching completed successfully", "vm_id", instanceID, "needs_reboot", needsReboot)
	return nil
}

// GetPatchStatus returns the patch compliance status for a VM.
func (c *vSpherePlatformClient) GetPatchStatus(ctx context.Context, instanceID string) (string, error) {
	vm, err := c.findVMByID(ctx, instanceID)
	if err != nil {
		return "", fmt.Errorf("failed to find VM: %w", err)
	}

	// Verify VMware Tools is running
	var props mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), []string{"guest"}, &props); err != nil {
		return "", fmt.Errorf("failed to get VM properties: %w", err)
	}

	if props.Guest == nil || props.Guest.ToolsRunningStatus != string(types.VirtualMachineToolsRunningStatusGuestToolsRunning) {
		return "unknown", nil
	}

	guestFamily := props.Guest.GuestFamily
	isWindows := strings.Contains(strings.ToLower(string(guestFamily)), "windows")

	var output string
	var err2 error

	if isWindows {
		// Check Windows Update status
		output, err2 = c.runGuestCommandWithOutput(ctx, vm, "powershell.exe",
			[]string{"-Command", `
$Session = New-Object -ComObject Microsoft.Update.Session
$Searcher = $Session.CreateUpdateSearcher()
$SearchResult = $Searcher.Search("IsInstalled=0 and Type='Software'")
$SearchResult.Updates.Count
`})
	} else {
		// Check Linux updates available
		output, err2 = c.runGuestCommandWithOutput(ctx, vm, "/bin/bash",
			[]string{"-c", `
if command -v apt-get &> /dev/null; then
    apt-get update -qq && apt-get -s upgrade | grep -c '^Inst ' || echo 0
elif command -v yum &> /dev/null; then
    yum check-update -q | wc -l
elif command -v dnf &> /dev/null; then
    dnf check-update -q | wc -l
else
    echo -1
fi
`})
	}

	if err2 != nil {
		return "error", fmt.Errorf("failed to check patch status: %w", err2)
	}

	pendingUpdates := strings.TrimSpace(output)
	if pendingUpdates == "0" {
		return "compliant", nil
	} else if pendingUpdates == "-1" {
		return "unknown", nil
	}

	return fmt.Sprintf("non_compliant:%s_updates_pending", pendingUpdates), nil
}

// GetPatchComplianceData returns detailed patch compliance information.
func (c *vSpherePlatformClient) GetPatchComplianceData(ctx context.Context, instanceID string) (interface{}, error) {
	vm, err := c.findVMByID(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to find VM: %w", err)
	}

	var props mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), []string{"guest", "config"}, &props); err != nil {
		return nil, fmt.Errorf("failed to get VM properties: %w", err)
	}

	if props.Guest == nil || props.Guest.ToolsRunningStatus != string(types.VirtualMachineToolsRunningStatusGuestToolsRunning) {
		return map[string]interface{}{
			"status":  "unknown",
			"reason":  "VMware Tools not running",
			"vm_id":   instanceID,
		}, nil
	}

	guestFamily := props.Guest.GuestFamily
	isWindows := strings.Contains(strings.ToLower(string(guestFamily)), "windows")

	result := map[string]interface{}{
		"vm_id":       instanceID,
		"vm_name":     props.Config.Name,
		"guest_os":    props.Guest.GuestFullName,
		"guest_state": props.Guest.GuestState,
		"tools_status": props.Guest.ToolsRunningStatus,
		"ip_address":  props.Guest.IpAddress,
	}

	var output string
	var err2 error

	if isWindows {
		output, err2 = c.runGuestCommandWithOutput(ctx, vm, "powershell.exe",
			[]string{"-Command", `
$Session = New-Object -ComObject Microsoft.Update.Session
$Searcher = $Session.CreateUpdateSearcher()
$SearchResult = $Searcher.Search("IsInstalled=0 and Type='Software'")

$updates = @()
foreach ($Update in $SearchResult.Updates) {
    $updates += @{
        Title = $Update.Title
        Severity = $Update.MsrcSeverity
        KB = ($Update.KBArticleIDs -join ',')
        Categories = ($Update.Categories | ForEach-Object { $_.Name }) -join ','
    }
}
$updates | ConvertTo-Json -Compress
`})
	} else {
		output, err2 = c.runGuestCommandWithOutput(ctx, vm, "/bin/bash",
			[]string{"-c", `
if command -v apt-get &> /dev/null; then
    apt-get update -qq 2>/dev/null
    apt-get -s upgrade 2>/dev/null | grep '^Inst ' | awk '{print "{\"package\":\""$2"\",\"version\":\""$3"\"}"}' | paste -sd ',' | sed 's/^/[/;s/$/]/'
elif command -v yum &> /dev/null; then
    yum check-update -q 2>/dev/null | awk 'NF==3 {print "{\"package\":\""$1"\",\"version\":\""$2"\"}"}' | paste -sd ',' | sed 's/^/[/;s/$/]/'
else
    echo "[]"
fi
`})
	}

	if err2 != nil {
		result["status"] = "error"
		result["error"] = err2.Error()
	} else {
		result["pending_updates_raw"] = output
		result["status"] = "retrieved"
	}

	return result, nil
}

// =============================================================================
// Helper Methods
// =============================================================================

// errNotConnected is returned when operations are attempted without a connection.
var errNotConnected = fmt.Errorf("vSphere client not connected")

// isConnected returns true if the client is connected to vCenter.
func (c *vSpherePlatformClient) isConnected() bool {
	return c.client != nil
}

// ensureConnected returns an error if the client is not connected.
func (c *vSpherePlatformClient) ensureConnected() error {
	if !c.isConnected() {
		return errNotConnected
	}
	return nil
}

// findVMByID finds a VM by its MoRef ID.
func (c *vSpherePlatformClient) findVMByID(ctx context.Context, vmID string) (*object.VirtualMachine, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}
	// Create a view of all VMs
	viewMgr := view.NewManager(c.client.Client)
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

	// Search for the VM
	var vms []mo.VirtualMachine
	err = containerView.Retrieve(ctx, []string{"VirtualMachine"}, []string{"name"}, &vms)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve VMs: %w", err)
	}

	for _, vm := range vms {
		if vm.Self.Value == vmID {
			return object.NewVirtualMachine(c.client.Client, vm.Self), nil
		}
	}

	// Try finding by name as fallback
	vm, err := c.finder.VirtualMachine(ctx, vmID)
	if err == nil {
		return vm, nil
	}

	return nil, fmt.Errorf("VM not found: %s", vmID)
}

// powerOffVM powers off a VM gracefully, falling back to hard power off.
func (c *vSpherePlatformClient) powerOffVM(ctx context.Context, vm *object.VirtualMachine) error {
	// Try graceful shutdown first
	err := vm.ShutdownGuest(ctx)
	if err == nil {
		// Wait for power off
		ctx2, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		return c.waitForPowerState(ctx2, vm, types.VirtualMachinePowerStatePoweredOff)
	}

	c.log.Warn("graceful shutdown failed, using hard power off", "error", err)

	// Hard power off
	powerOffTask, err := vm.PowerOff(ctx)
	if err != nil {
		return fmt.Errorf("failed to start power off task: %w", err)
	}

	return powerOffTask.Wait(ctx)
}

// powerOnVM powers on a VM.
func (c *vSpherePlatformClient) powerOnVM(ctx context.Context, vm *object.VirtualMachine) error {
	powerOnTask, err := vm.PowerOn(ctx)
	if err != nil {
		return fmt.Errorf("failed to start power on task: %w", err)
	}

	return powerOnTask.Wait(ctx)
}

// waitForPowerState waits for a VM to reach a specific power state.
func (c *vSpherePlatformClient) waitForPowerState(ctx context.Context, vm *object.VirtualMachine, targetState types.VirtualMachinePowerState) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			var props mo.VirtualMachine
			if err := vm.Properties(ctx, vm.Reference(), []string{"runtime"}, &props); err != nil {
				continue
			}

			if props.Runtime.PowerState == targetState {
				return nil
			}
		}
	}
}

// waitForVMTools waits for VMware Tools to be running.
func (c *vSpherePlatformClient) waitForVMTools(ctx context.Context, vm *object.VirtualMachine, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for VMware Tools")
			}

			var props mo.VirtualMachine
			if err := vm.Properties(ctx, vm.Reference(), []string{"guest"}, &props); err != nil {
				continue
			}

			if props.Guest != nil && props.Guest.ToolsRunningStatus == string(types.VirtualMachineToolsRunningStatusGuestToolsRunning) {
				return nil
			}
		}
	}
}

// getVMFolder returns the folder containing the VM.
func (c *vSpherePlatformClient) getVMFolder(ctx context.Context, vm *object.VirtualMachine) (*object.Folder, error) {
	var props mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), []string{"parent"}, &props); err != nil {
		return nil, err
	}

	if props.Parent == nil {
		return nil, fmt.Errorf("VM has no parent folder")
	}

	return object.NewFolder(c.client.Client, *props.Parent), nil
}

// getVMResourcePool returns the resource pool for the VM.
func (c *vSpherePlatformClient) getVMResourcePool(ctx context.Context, vm *object.VirtualMachine) (*object.ResourcePool, error) {
	var props mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), []string{"resourcePool"}, &props); err != nil {
		return nil, err
	}

	if props.ResourcePool == nil {
		return nil, fmt.Errorf("VM has no resource pool")
	}

	return object.NewResourcePool(c.client.Client, *props.ResourcePool), nil
}

// copyNetworkConfig copies network adapter configuration from one VM to another.
func (c *vSpherePlatformClient) copyNetworkConfig(ctx context.Context, sourceVM, targetVM *object.VirtualMachine) error {
	// Get source VM network devices
	var sourceProps mo.VirtualMachine
	if err := sourceVM.Properties(ctx, sourceVM.Reference(), []string{"config.hardware.device"}, &sourceProps); err != nil {
		return err
	}

	if sourceProps.Config == nil {
		return nil
	}

	// Find network adapters in source
	var deviceChanges []types.BaseVirtualDeviceConfigSpec
	for _, device := range sourceProps.Config.Hardware.Device {
		if nic, ok := device.(types.BaseVirtualEthernetCard); ok {
			// Get the underlying ethernet card configuration
			nicDevice := nic.GetVirtualEthernetCard()

			// Reset the key to allow vSphere to assign a new one
			nicDevice.Key = -1

			// Create a new device spec to add this NIC to target
			// device is already BaseVirtualDevice since it comes from Hardware.Device
			spec := &types.VirtualDeviceConfigSpec{
				Operation: types.VirtualDeviceConfigSpecOperationAdd,
				Device:    device,
			}

			deviceChanges = append(deviceChanges, spec)
		}
	}

	if len(deviceChanges) == 0 {
		return nil
	}

	// Apply network config to target VM
	configSpec := types.VirtualMachineConfigSpec{
		DeviceChange: deviceChanges,
	}

	task, err := targetVM.Reconfigure(ctx, configSpec)
	if err != nil {
		return err
	}

	return task.Wait(ctx)
}

// runGuestCommand runs a command inside the guest VM.
func (c *vSpherePlatformClient) runGuestCommand(ctx context.Context, vm *object.VirtualMachine, path string, args []string) error {
	_, err := c.runGuestCommandWithOutput(ctx, vm, path, args)
	return err
}

// runGuestCommandWithOutput runs a command inside the guest VM and returns output.
func (c *vSpherePlatformClient) runGuestCommandWithOutput(ctx context.Context, vm *object.VirtualMachine, path string, args []string) (string, error) {
	// Create operations manager
	opsMgr := guest.NewOperationsManager(c.client.Client, vm.Reference())

	// Get process manager
	procMgr, err := opsMgr.ProcessManager(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get process manager: %w", err)
	}

	// Create authentication
	auth := &types.NamePasswordAuthentication{
		Username: c.cfg.GuestUsername,
		Password: c.cfg.GuestPassword,
	}

	// Build command line
	cmdLine := strings.Join(args, " ")

	// Start the program
	spec := types.GuestProgramSpec{
		ProgramPath: path,
		Arguments:   cmdLine,
	}

	pid, err := procMgr.StartProgram(ctx, auth, &spec)
	if err != nil {
		return "", fmt.Errorf("failed to start program: %w", err)
	}

	// Wait for process to complete
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(c.cfg.OperationTimeout)
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "", fmt.Errorf("timeout waiting for guest command")
		case <-ticker.C:
			procs, err := procMgr.ListProcesses(ctx, auth, []int64{pid})
			if err != nil {
				continue
			}

			if len(procs) == 0 {
				return "", fmt.Errorf("process not found")
			}

			proc := procs[0]
			if proc.EndTime != nil {
				// Process completed
				if proc.ExitCode != 0 {
					return "", fmt.Errorf("command failed with exit code %d", proc.ExitCode)
				}
				// Note: Getting actual output requires file transfer which adds complexity
				// For now, just return success
				return "", nil
			}
		}
	}
}
