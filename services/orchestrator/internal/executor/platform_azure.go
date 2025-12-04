// Package executor implements the plan execution engine.
package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// AzurePlatformClient implements PlatformClient for Azure.
type AzurePlatformClient struct {
	cfg             AzureClientConfig
	credential      azcore.TokenCredential
	vmClient        *armcompute.VirtualMachinesClient
	vmExtClient     *armcompute.VirtualMachineExtensionsClient
	resourcesClient *armresources.Client
	log             *logger.Logger
	connected       bool
}

// AzureClientConfig holds Azure client configuration.
type AzureClientConfig struct {
	SubscriptionID string
	ResourceGroup  string
	TenantID       string
	ClientID       string
	ClientSecret   string
}

// NewAzurePlatformClient creates a new Azure platform client.
func NewAzurePlatformClient(cfg AzureClientConfig, log *logger.Logger) *AzurePlatformClient {
	return &AzurePlatformClient{
		cfg: cfg,
		log: log.WithComponent("azure-platform-client"),
	}
}

// Connect establishes a connection to Azure.
func (c *AzurePlatformClient) Connect(ctx context.Context) error {
	var cred azcore.TokenCredential
	var err error

	// Try client secret credential if provided
	if c.cfg.ClientID != "" && c.cfg.ClientSecret != "" {
		cred, err = azidentity.NewClientSecretCredential(c.cfg.TenantID, c.cfg.ClientID, c.cfg.ClientSecret, nil)
		if err != nil {
			return fmt.Errorf("failed to create client secret credential: %w", err)
		}
	} else {
		// Fall back to default Azure credential (MSI, CLI, etc.)
		cred, err = azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return fmt.Errorf("failed to create default credential: %w", err)
		}
	}

	c.credential = cred

	// Create ARM clients with retry policy
	clientOptions := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Retry: policy.RetryOptions{
				MaxRetries: 3,
				RetryDelay: 1 * time.Second,
			},
		},
	}

	vmClient, err := armcompute.NewVirtualMachinesClient(c.cfg.SubscriptionID, cred, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to create VM client: %w", err)
	}
	c.vmClient = vmClient

	vmExtClient, err := armcompute.NewVirtualMachineExtensionsClient(c.cfg.SubscriptionID, cred, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to create VM extensions client: %w", err)
	}
	c.vmExtClient = vmExtClient

	resourcesClient, err := armresources.NewClient(c.cfg.SubscriptionID, cred, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to create resources client: %w", err)
	}
	c.resourcesClient = resourcesClient

	c.connected = true

	c.log.Info("connected to Azure",
		"subscription_id", c.cfg.SubscriptionID,
		"resource_group", c.cfg.ResourceGroup,
	)

	return nil
}

// Close closes the Azure connection.
func (c *AzurePlatformClient) Close() error {
	c.connected = false
	return nil
}

// ReimageInstance reimages an Azure VM.
// For Azure, this uses the Reimage operation which restores the VM to its initial state.
func (c *AzurePlatformClient) ReimageInstance(ctx context.Context, instanceID, imageID string) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	// Parse instance ID to get resource group and VM name
	resourceGroup, vmName, err := c.parseInstanceID(instanceID)
	if err != nil {
		return err
	}

	c.log.Info("reimaging VM",
		"resource_group", resourceGroup,
		"vm_name", vmName,
		"target_image", imageID,
	)

	// If a new image is specified, we need to update the VM image reference and reimage
	if imageID != "" {
		// Get the current VM configuration
		vm, err := c.vmClient.Get(ctx, resourceGroup, vmName, nil)
		if err != nil {
			return fmt.Errorf("failed to get VM: %w", err)
		}

		// Update the image reference
		if vm.Properties.StorageProfile.ImageReference != nil {
			vm.Properties.StorageProfile.ImageReference.ID = &imageID
		}

		// Start the update (async operation)
		poller, err := c.vmClient.BeginCreateOrUpdate(ctx, resourceGroup, vmName, vm.VirtualMachine, nil)
		if err != nil {
			return fmt.Errorf("failed to start VM update: %w", err)
		}

		// Wait for completion
		_, err = poller.PollUntilDone(ctx, nil)
		if err != nil {
			return fmt.Errorf("VM update failed: %w", err)
		}
	}

	// Reimage the VM
	poller, err := c.vmClient.BeginReimage(ctx, resourceGroup, vmName, nil)
	if err != nil {
		return fmt.Errorf("failed to start reimage: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("reimage failed: %w", err)
	}

	c.log.Info("VM reimaged successfully",
		"resource_group", resourceGroup,
		"vm_name", vmName,
	)

	return nil
}

// RebootInstance reboots an Azure VM.
func (c *AzurePlatformClient) RebootInstance(ctx context.Context, instanceID string) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	resourceGroup, vmName, err := c.parseInstanceID(instanceID)
	if err != nil {
		return err
	}

	c.log.Info("rebooting VM", "resource_group", resourceGroup, "vm_name", vmName)

	poller, err := c.vmClient.BeginRestart(ctx, resourceGroup, vmName, nil)
	if err != nil {
		return fmt.Errorf("failed to start reboot: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("reboot failed: %w", err)
	}

	return nil
}

// TerminateInstance terminates (deletes) an Azure VM.
func (c *AzurePlatformClient) TerminateInstance(ctx context.Context, instanceID string) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	resourceGroup, vmName, err := c.parseInstanceID(instanceID)
	if err != nil {
		return err
	}

	c.log.Info("terminating VM", "resource_group", resourceGroup, "vm_name", vmName)

	poller, err := c.vmClient.BeginDelete(ctx, resourceGroup, vmName, nil)
	if err != nil {
		return fmt.Errorf("failed to start VM deletion: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("VM deletion failed: %w", err)
	}

	return nil
}

// GetInstanceStatus gets the current status of an Azure VM.
func (c *AzurePlatformClient) GetInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	if !c.connected {
		return "", fmt.Errorf("not connected")
	}

	resourceGroup, vmName, err := c.parseInstanceID(instanceID)
	if err != nil {
		return "", err
	}

	// Get instance view to get the power state
	expand := armcompute.InstanceViewTypesInstanceView
	vm, err := c.vmClient.Get(ctx, resourceGroup, vmName, &armcompute.VirtualMachinesClientGetOptions{
		Expand: &expand,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get VM: %w", err)
	}

	if vm.Properties.InstanceView != nil && vm.Properties.InstanceView.Statuses != nil {
		for _, status := range vm.Properties.InstanceView.Statuses {
			if status.Code != nil && len(*status.Code) > 11 && (*status.Code)[:11] == "PowerState/" {
				return (*status.Code)[11:], nil
			}
		}
	}

	return "unknown", nil
}

// WaitForInstanceState waits for an Azure VM to reach a specific state.
func (c *AzurePlatformClient) WaitForInstanceState(ctx context.Context, instanceID, targetState string, timeout time.Duration) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	c.log.Debug("waiting for VM state",
		"instance_id", instanceID,
		"target_state", targetState,
		"timeout", timeout,
	)

	deadline := time.Now().Add(timeout)
	pollInterval := 10 * time.Second

	for time.Now().Before(deadline) {
		status, err := c.GetInstanceStatus(ctx, instanceID)
		if err != nil {
			// For deallocated/deleted, the VM might not be found
			if targetState == "deallocated" || targetState == "deleted" {
				return nil
			}
			return err
		}

		if status == targetState {
			c.log.Debug("VM reached target state",
				"instance_id", instanceID,
				"state", targetState,
			)
			return nil
		}

		c.log.Debug("waiting for VM state",
			"instance_id", instanceID,
			"current_state", status,
			"target_state", targetState,
		)

		select {
		case <-time.After(pollInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("timeout waiting for VM %s to reach state %s", instanceID, targetState)
}

// ApplyPatches applies patches to an Azure VM using Azure Update Management.
// This uses the Azure Update Management extension or Run Command.
func (c *AzurePlatformClient) ApplyPatches(ctx context.Context, instanceID string, params map[string]interface{}) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	resourceGroup, vmName, err := c.parseInstanceID(instanceID)
	if err != nil {
		return err
	}

	c.log.Info("applying patches to VM",
		"resource_group", resourceGroup,
		"vm_name", vmName,
	)

	// Determine OS type for appropriate patching method
	vm, err := c.vmClient.Get(ctx, resourceGroup, vmName, nil)
	if err != nil {
		return fmt.Errorf("failed to get VM: %w", err)
	}

	isWindows := vm.Properties.StorageProfile.OSDisk.OSType != nil &&
		*vm.Properties.StorageProfile.OSDisk.OSType == armcompute.OperatingSystemTypesWindows

	// Build the patch script based on OS
	var script string
	if isWindows {
		// Windows Update via PowerShell
		script = `
$updateSession = New-Object -ComObject Microsoft.Update.Session
$searcher = $updateSession.CreateUpdateSearcher()
$searchResult = $searcher.Search("IsInstalled=0 and Type='Software'")

if ($searchResult.Updates.Count -gt 0) {
    $updates = $searchResult.Updates
    $downloader = $updateSession.CreateUpdateDownloader()
    $downloader.Updates = $updates
    $downloader.Download()

    $installer = $updateSession.CreateUpdateInstaller()
    $installer.Updates = $updates
    $result = $installer.Install()

    Write-Output "Installation Result: $($result.ResultCode)"
    Write-Output "Reboot Required: $($result.RebootRequired)"
}
else {
    Write-Output "No updates available"
}
`
	} else {
		// Linux patching via apt/yum/dnf
		script = `#!/bin/bash
set -e

# Detect package manager and update
if command -v apt-get &> /dev/null; then
    apt-get update
    apt-get upgrade -y
    apt-get autoremove -y
elif command -v yum &> /dev/null; then
    yum update -y
    yum autoremove -y
elif command -v dnf &> /dev/null; then
    dnf update -y
    dnf autoremove -y
else
    echo "Unknown package manager"
    exit 1
fi

echo "Patching completed successfully"
`
	}

	// Use Run Command to execute the patching script
	runCommandInput := armcompute.RunCommandInput{
		CommandID: stringPtr("RunShellScript"),
	}

	if isWindows {
		runCommandInput.CommandID = stringPtr("RunPowerShellScript")
	}

	runCommandInput.Script = []*string{&script}

	c.log.Info("executing patch command",
		"resource_group", resourceGroup,
		"vm_name", vmName,
		"os_type", map[bool]string{true: "Windows", false: "Linux"}[isWindows],
	)

	poller, err := c.vmClient.BeginRunCommand(ctx, resourceGroup, vmName, runCommandInput, nil)
	if err != nil {
		return fmt.Errorf("failed to run patch command: %w", err)
	}

	// Wait for completion if synchronous mode is requested
	if sync, ok := params["synchronous"].(bool); ok && sync {
		timeout := 60 * time.Minute // Patching can take a while
		if t, ok := params["timeout"].(time.Duration); ok {
			timeout = t
		}

		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		result, err := poller.PollUntilDone(ctx, nil)
		if err != nil {
			return fmt.Errorf("patch command failed: %w", err)
		}

		if result.Value != nil && len(result.Value) > 0 && result.Value[0].Message != nil {
			c.log.Info("patch command output",
				"resource_group", resourceGroup,
				"vm_name", vmName,
				"output", *result.Value[0].Message,
			)
		}
	}

	// Handle reboot if requested
	if reboot, ok := params["reboot_if_needed"].(bool); ok && reboot {
		c.log.Info("rebooting VM after patching", "vm_name", vmName)
		if err := c.RebootInstance(ctx, instanceID); err != nil {
			c.log.Warn("failed to reboot after patching", "error", err)
		}
	}

	return nil
}

// GetPatchStatus retrieves patch compliance status for an Azure VM.
func (c *AzurePlatformClient) GetPatchStatus(ctx context.Context, instanceID string) (string, error) {
	if !c.connected {
		return "", fmt.Errorf("not connected")
	}

	resourceGroup, vmName, err := c.parseInstanceID(instanceID)
	if err != nil {
		return "", err
	}

	// Get instance view which includes patch assessment status
	expand := armcompute.InstanceViewTypesInstanceView
	vm, err := c.vmClient.Get(ctx, resourceGroup, vmName, &armcompute.VirtualMachinesClientGetOptions{
		Expand: &expand,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get VM: %w", err)
	}

	// Check if patch assessment is available
	if vm.Properties.InstanceView != nil && vm.Properties.InstanceView.PatchStatus != nil {
		status := vm.Properties.InstanceView.PatchStatus
		if status.AvailablePatchSummary != nil {
			summary := status.AvailablePatchSummary

			// Check if there are critical or security patches
			criticalCount := int32(0)

			if summary.CriticalAndSecurityPatchCount != nil {
				criticalCount = *summary.CriticalAndSecurityPatchCount
			}

			if criticalCount > 0 {
				return "NON_COMPLIANT", nil
			}
		}
	}

	// If we can't determine status, assume compliant
	return "COMPLIANT", nil
}

// GetPatchComplianceData retrieves detailed patch compliance data for an Azure VM.
func (c *AzurePlatformClient) GetPatchComplianceData(ctx context.Context, instanceID string) (interface{}, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	resourceGroup, vmName, err := c.parseInstanceID(instanceID)
	if err != nil {
		return nil, err
	}

	// Get instance view which includes patch assessment status
	expand := armcompute.InstanceViewTypesInstanceView
	vm, err := c.vmClient.Get(ctx, resourceGroup, vmName, &armcompute.VirtualMachinesClientGetOptions{
		Expand: &expand,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get VM: %w", err)
	}

	result := map[string]interface{}{
		"instance_id":    instanceID,
		"resource_group": resourceGroup,
		"vm_name":        vmName,
		"status":         "COMPLIANT",
	}

	// Extract patch status information if available
	if vm.Properties.InstanceView != nil && vm.Properties.InstanceView.PatchStatus != nil {
		status := vm.Properties.InstanceView.PatchStatus
		if status.AvailablePatchSummary != nil {
			summary := status.AvailablePatchSummary
			patchInfo := map[string]interface{}{}

			if summary.CriticalAndSecurityPatchCount != nil {
				patchInfo["critical_and_security_count"] = *summary.CriticalAndSecurityPatchCount
				if *summary.CriticalAndSecurityPatchCount > 0 {
					result["status"] = "NON_COMPLIANT"
				}
			}
			if summary.OtherPatchCount != nil {
				patchInfo["other_count"] = *summary.OtherPatchCount
			}
			if summary.Status != nil {
				patchInfo["assessment_status"] = string(*summary.Status)
			}
			if summary.StartTime != nil {
				patchInfo["last_assessment"] = summary.StartTime.Format(time.RFC3339)
			}
			if summary.RebootPending != nil {
				patchInfo["reboot_pending"] = *summary.RebootPending
			}

			result["patches"] = patchInfo
		}

		if status.LastPatchInstallationSummary != nil {
			lastInstall := status.LastPatchInstallationSummary
			installInfo := map[string]interface{}{}

			if lastInstall.Status != nil {
				installInfo["status"] = string(*lastInstall.Status)
			}
			if lastInstall.InstalledPatchCount != nil {
				installInfo["installed_count"] = *lastInstall.InstalledPatchCount
			}
			if lastInstall.FailedPatchCount != nil {
				installInfo["failed_count"] = *lastInstall.FailedPatchCount
			}
			if lastInstall.StartTime != nil {
				installInfo["start_time"] = lastInstall.StartTime.Format(time.RFC3339)
			}

			result["last_installation"] = installInfo
		}
	}

	return result, nil
}

// parseInstanceID parses an Azure instance ID in the format "resourceGroup/vmName" or just "vmName".
func (c *AzurePlatformClient) parseInstanceID(instanceID string) (resourceGroup, vmName string, err error) {
	// Try splitting by "/"
	parts := splitString(instanceID, "/")
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}
	if len(parts) == 1 {
		// Use default resource group from config
		if c.cfg.ResourceGroup == "" {
			return "", "", fmt.Errorf("no resource group specified and instance ID doesn't include one: %s", instanceID)
		}
		return c.cfg.ResourceGroup, parts[0], nil
	}
	return "", "", fmt.Errorf("invalid instance ID format: %s", instanceID)
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// Helper function to split string
func splitString(s string, sep string) []string {
	var result []string
	current := ""
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			if current != "" {
				result = append(result, current)
			}
			current = ""
			i += len(sep) - 1
		} else {
			current += string(s[i])
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
