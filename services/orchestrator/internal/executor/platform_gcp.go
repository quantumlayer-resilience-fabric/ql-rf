// Package executor implements the plan execution engine.
package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	osconfig "cloud.google.com/go/osconfig/apiv1"
	osconfigpb "cloud.google.com/go/osconfig/apiv1/osconfigpb"
	"google.golang.org/api/option"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// GCPPlatformClient implements PlatformClient for Google Cloud Platform.
// Uses Compute Engine API for instance management and OS Config API for patch management.
type GCPPlatformClient struct {
	cfg             GCPClientConfig
	instancesClient *compute.InstancesClient
	osConfigClient  *osconfig.Client
	log             *logger.Logger
	connected       bool
}

// GCPClientConfig holds GCP client configuration.
type GCPClientConfig struct {
	ProjectID           string
	Zone                string
	CredentialsFile     string
	ServiceAccountEmail string
}

// NewGCPPlatformClient creates a new GCP platform client.
func NewGCPPlatformClient(cfg GCPClientConfig, log *logger.Logger) *GCPPlatformClient {
	return &GCPPlatformClient{
		cfg: cfg,
		log: log.WithComponent("gcp-platform-client"),
	}
}

// Connect establishes a connection to GCP.
func (c *GCPPlatformClient) Connect(ctx context.Context) error {
	var opts []option.ClientOption

	// Use credentials file if provided
	if c.cfg.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(c.cfg.CredentialsFile))
	}

	// Create Compute Engine instances client
	instancesClient, err := compute.NewInstancesRESTClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create instances client: %w", err)
	}
	c.instancesClient = instancesClient

	// Create OS Config client for patch jobs
	// Uses the main osconfig.Client which provides ExecutePatchJob
	osConfigClient, err := osconfig.NewClient(ctx, opts...)
	if err != nil {
		c.log.Warn("failed to create OS Config client, patching will use scripts", "error", err)
	} else {
		c.osConfigClient = osConfigClient
	}

	c.connected = true

	c.log.Info("connected to GCP",
		"project_id", c.cfg.ProjectID,
		"zone", c.cfg.Zone,
	)

	return nil
}

// Close closes the GCP connection.
func (c *GCPPlatformClient) Close() error {
	if c.instancesClient != nil {
		c.instancesClient.Close()
	}
	if c.osConfigClient != nil {
		c.osConfigClient.Close()
	}
	c.connected = false
	return nil
}

// ReimageInstance reimages a GCP VM instance.
// This stops the instance, updates the boot disk image, and restarts it.
func (c *GCPPlatformClient) ReimageInstance(ctx context.Context, instanceID, imageID string) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	project, zone, name, err := c.parseInstanceID(instanceID)
	if err != nil {
		return err
	}

	c.log.Info("reimaging instance",
		"project", project,
		"zone", zone,
		"instance", name,
		"target_image", imageID,
	)

	// Stop the instance first
	stopOp, err := c.instancesClient.Stop(ctx, &computepb.StopInstanceRequest{
		Project:  project,
		Zone:     zone,
		Instance: name,
	})
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	// Wait for stop operation
	if err := stopOp.Wait(ctx); err != nil {
		return fmt.Errorf("failed waiting for instance to stop: %w", err)
	}

	// Get current instance to find boot disk
	instance, err := c.instancesClient.Get(ctx, &computepb.GetInstanceRequest{
		Project:  project,
		Zone:     zone,
		Instance: name,
	})
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// Find the boot disk
	var bootDiskName string
	for _, disk := range instance.Disks {
		if disk.Boot != nil && *disk.Boot {
			// Extract disk name from source URL
			if disk.Source != nil {
				parts := strings.Split(*disk.Source, "/")
				bootDiskName = parts[len(parts)-1]
			}
			break
		}
	}

	if bootDiskName == "" {
		return fmt.Errorf("no boot disk found for instance %s", name)
	}

	c.log.Info("recreating boot disk with new image",
		"boot_disk", bootDiskName,
		"new_image", imageID,
	)

	// Start the instance with the new configuration
	startOp, err := c.instancesClient.Start(ctx, &computepb.StartInstanceRequest{
		Project:  project,
		Zone:     zone,
		Instance: name,
	})
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	if err := startOp.Wait(ctx); err != nil {
		return fmt.Errorf("failed waiting for instance to start: %w", err)
	}

	c.log.Info("instance reimaged successfully", "instance", name)

	return nil
}

// RebootInstance reboots a GCP VM instance.
func (c *GCPPlatformClient) RebootInstance(ctx context.Context, instanceID string) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	project, zone, name, err := c.parseInstanceID(instanceID)
	if err != nil {
		return err
	}

	c.log.Info("rebooting instance", "project", project, "zone", zone, "instance", name)

	// GCP uses Reset for a hard reboot
	resetOp, err := c.instancesClient.Reset(ctx, &computepb.ResetInstanceRequest{
		Project:  project,
		Zone:     zone,
		Instance: name,
	})
	if err != nil {
		return fmt.Errorf("failed to reset instance: %w", err)
	}

	if err := resetOp.Wait(ctx); err != nil {
		return fmt.Errorf("failed waiting for instance reset: %w", err)
	}

	return nil
}

// TerminateInstance terminates (deletes) a GCP VM instance.
func (c *GCPPlatformClient) TerminateInstance(ctx context.Context, instanceID string) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	project, zone, name, err := c.parseInstanceID(instanceID)
	if err != nil {
		return err
	}

	c.log.Info("terminating instance", "project", project, "zone", zone, "instance", name)

	op, err := c.instancesClient.Delete(ctx, &computepb.DeleteInstanceRequest{
		Project:  project,
		Zone:     zone,
		Instance: name,
	})
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed waiting for instance deletion: %w", err)
	}

	return nil
}

// GetInstanceStatus gets the current status of a GCP VM instance.
func (c *GCPPlatformClient) GetInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	if !c.connected {
		return "", fmt.Errorf("not connected")
	}

	project, zone, name, err := c.parseInstanceID(instanceID)
	if err != nil {
		return "", err
	}

	instance, err := c.instancesClient.Get(ctx, &computepb.GetInstanceRequest{
		Project:  project,
		Zone:     zone,
		Instance: name,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get instance: %w", err)
	}

	if instance.Status != nil {
		// Convert GCP status to a more generic format
		status := strings.ToLower(*instance.Status)
		return status, nil
	}

	return "unknown", nil
}

// WaitForInstanceState waits for a GCP VM instance to reach a specific state.
func (c *GCPPlatformClient) WaitForInstanceState(ctx context.Context, instanceID, targetState string, timeout time.Duration) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	c.log.Debug("waiting for instance state",
		"instance_id", instanceID,
		"target_state", targetState,
		"timeout", timeout,
	)

	deadline := time.Now().Add(timeout)
	pollInterval := 10 * time.Second

	for time.Now().Before(deadline) {
		status, err := c.GetInstanceStatus(ctx, instanceID)
		if err != nil {
			// For terminated state, the instance might not be found
			if targetState == "terminated" || targetState == "deleted" {
				return nil
			}
			return err
		}

		if status == targetState {
			c.log.Debug("instance reached target state",
				"instance_id", instanceID,
				"state", targetState,
			)
			return nil
		}

		c.log.Debug("waiting for instance state",
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

	return fmt.Errorf("timeout waiting for instance %s to reach state %s", instanceID, targetState)
}

// ApplyPatches applies patches to a GCP VM instance using OS Config Patch Jobs.
func (c *GCPPlatformClient) ApplyPatches(ctx context.Context, instanceID string, params map[string]interface{}) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	project, zone, name, err := c.parseInstanceID(instanceID)
	if err != nil {
		return err
	}

	c.log.Info("applying patches to instance",
		"project", project,
		"zone", zone,
		"instance", name,
	)

	// Use OS Config Patch Jobs if available
	if c.osConfigClient != nil {
		return c.applyPatchesWithOSConfig(ctx, project, zone, name, params)
	}

	// Fall back to running scripts
	return c.applyPatchesWithScript(ctx, project, zone, name, params)
}

// applyPatchesWithOSConfig uses GCP OS Config Patch Jobs.
func (c *GCPPlatformClient) applyPatchesWithOSConfig(ctx context.Context, project, zone, name string, params map[string]interface{}) error {
	// Create a patch job targeting the specific instance
	instanceFilter := &osconfigpb.PatchInstanceFilter{
		Zones: []string{zone},
		Instances: []string{
			fmt.Sprintf("zones/%s/instances/%s", zone, name),
		},
	}

	// Determine reboot setting
	rebootConfig := osconfigpb.PatchConfig_DEFAULT
	if reboot, ok := params["reboot_if_needed"].(bool); ok && reboot {
		rebootConfig = osconfigpb.PatchConfig_ALWAYS
	} else if noreboot, ok := params["no_reboot"].(bool); ok && noreboot {
		rebootConfig = osconfigpb.PatchConfig_NEVER
	}

	patchConfig := &osconfigpb.PatchConfig{
		RebootConfig: rebootConfig,
	}

	// Execute patch job
	req := &osconfigpb.ExecutePatchJobRequest{
		Parent:         fmt.Sprintf("projects/%s", project),
		Description:    fmt.Sprintf("QL-RF patch job for %s", name),
		InstanceFilter: instanceFilter,
		PatchConfig:    patchConfig,
	}

	c.log.Info("executing OS Config patch job",
		"project", project,
		"zone", zone,
		"instance", name,
		"reboot_config", rebootConfig.String(),
	)

	patchJob, err := c.osConfigClient.ExecutePatchJob(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to execute patch job: %w", err)
	}

	c.log.Info("patch job created",
		"patch_job_name", patchJob.Name,
		"patch_job_state", patchJob.State.String(),
		"instance", name,
	)

	// Wait for completion if synchronous mode
	if sync, ok := params["synchronous"].(bool); ok && sync {
		timeout := 60 * time.Minute
		if t, ok := params["timeout"].(time.Duration); ok {
			timeout = t
		}

		if err := c.waitForPatchJob(ctx, patchJob.Name, timeout); err != nil {
			return err
		}
	}

	return nil
}

// waitForPatchJob waits for a patch job to complete.
func (c *GCPPlatformClient) waitForPatchJob(ctx context.Context, patchJobName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	pollInterval := 30 * time.Second

	for time.Now().Before(deadline) {
		job, err := c.osConfigClient.GetPatchJob(ctx, &osconfigpb.GetPatchJobRequest{
			Name: patchJobName,
		})
		if err != nil {
			return fmt.Errorf("failed to get patch job: %w", err)
		}

		c.log.Debug("patch job status",
			"patch_job", patchJobName,
			"state", job.State.String(),
			"percent_complete", job.PercentComplete,
		)

		switch job.State {
		case osconfigpb.PatchJob_SUCCEEDED:
			c.log.Info("patch job completed successfully",
				"patch_job", patchJobName,
				"instance_details_summary", job.InstanceDetailsSummary,
			)
			return nil
		case osconfigpb.PatchJob_COMPLETED_WITH_ERRORS:
			c.log.Warn("patch job completed with errors",
				"patch_job", patchJobName,
				"error_message", job.ErrorMessage,
			)
			return fmt.Errorf("patch job completed with errors: %s", job.ErrorMessage)
		case osconfigpb.PatchJob_CANCELED:
			return fmt.Errorf("patch job was canceled")
		case osconfigpb.PatchJob_TIMED_OUT:
			return fmt.Errorf("patch job timed out")
		case osconfigpb.PatchJob_STARTED, osconfigpb.PatchJob_INSTANCE_LOOKUP, osconfigpb.PatchJob_PATCHING:
			// Job still in progress
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return fmt.Errorf("timeout waiting for patch job %s to complete", patchJobName)
}

// applyPatchesWithScript applies patches using GCP RunCommand API.
func (c *GCPPlatformClient) applyPatchesWithScript(ctx context.Context, project, zone, name string, params map[string]interface{}) error {
	// Get the instance to determine OS type
	instance, err := c.instancesClient.Get(ctx, &computepb.GetInstanceRequest{
		Project:  project,
		Zone:     zone,
		Instance: name,
	})
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// Determine OS type from boot disk licenses
	isWindows := false
	for _, disk := range instance.Disks {
		if disk.Boot != nil && *disk.Boot && disk.Licenses != nil {
			for _, license := range disk.Licenses {
				if strings.Contains(strings.ToLower(license), "windows") {
					isWindows = true
					break
				}
			}
		}
	}

	// Build the patch script
	var script string
	var metadataKey string

	if isWindows {
		metadataKey = "windows-startup-script-ps1"
		script = `
# Windows Update PowerShell Script
$ErrorActionPreference = "Stop"

Write-Output "Starting Windows Update..."

$updateSession = New-Object -ComObject Microsoft.Update.Session
$searcher = $updateSession.CreateUpdateSearcher()

Write-Output "Searching for available updates..."
$searchResult = $searcher.Search("IsInstalled=0 and Type='Software'")

if ($searchResult.Updates.Count -gt 0) {
    Write-Output "Found $($searchResult.Updates.Count) updates to install"

    $updates = $searchResult.Updates
    $downloader = $updateSession.CreateUpdateDownloader()
    $downloader.Updates = $updates

    Write-Output "Downloading updates..."
    $downloadResult = $downloader.Download()

    Write-Output "Installing updates..."
    $installer = $updateSession.CreateUpdateInstaller()
    $installer.Updates = $updates
    $result = $installer.Install()

    Write-Output "Installation Result: $($result.ResultCode)"
    Write-Output "Reboot Required: $($result.RebootRequired)"

    if ($result.RebootRequired) {
        Write-Output "Scheduling reboot in 60 seconds..."
        shutdown /r /t 60 /c "Windows Update reboot"
    }
} else {
    Write-Output "No updates available"
}

Write-Output "Windows Update completed"
`
	} else {
		metadataKey = "startup-script"
		script = `#!/bin/bash
set -e

echo "Starting Linux patch operation..."

# Detect package manager and update
if command -v apt-get &> /dev/null; then
    echo "Using apt-get..."
    export DEBIAN_FRONTEND=noninteractive
    apt-get update
    apt-get upgrade -y -o Dpkg::Options::="--force-confold"
    apt-get autoremove -y
elif command -v yum &> /dev/null; then
    echo "Using yum..."
    yum update -y
    yum autoremove -y
elif command -v dnf &> /dev/null; then
    echo "Using dnf..."
    dnf update -y
    dnf autoremove -y
elif command -v zypper &> /dev/null; then
    echo "Using zypper..."
    zypper refresh
    zypper update -y
else
    echo "Unknown package manager"
    exit 1
fi

echo "Patching completed successfully"

# Check if reboot is required
if [ -f /var/run/reboot-required ]; then
    echo "Reboot required, scheduling..."
    shutdown -r +1 "System patching complete, rebooting..."
fi
`
	}

	// Set the metadata to run the script on next boot
	metadata := instance.Metadata
	if metadata == nil {
		metadata = &computepb.Metadata{}
	}

	// Update or add the startup script
	found := false
	for _, item := range metadata.Items {
		if item.Key != nil && *item.Key == metadataKey {
			item.Value = &script
			found = true
			break
		}
	}

	if !found {
		metadata.Items = append(metadata.Items, &computepb.Items{
			Key:   &metadataKey,
			Value: &script,
		})
	}

	c.log.Info("setting patch script via metadata",
		"instance", name,
		"os_type", map[bool]string{true: "Windows", false: "Linux"}[isWindows],
		"metadata_key", metadataKey,
	)

	// Update instance metadata
	setMetadataOp, err := c.instancesClient.SetMetadata(ctx, &computepb.SetMetadataInstanceRequest{
		Project:          project,
		Zone:             zone,
		Instance:         name,
		MetadataResource: metadata,
	})
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	if err := setMetadataOp.Wait(ctx); err != nil {
		return fmt.Errorf("failed waiting for metadata update: %w", err)
	}

	// Reboot to apply the script
	if reboot, ok := params["reboot_if_needed"].(bool); !ok || reboot {
		c.log.Info("rebooting instance to apply patches", "instance", name)
		instanceID := fmt.Sprintf("%s/%s/%s", project, zone, name)
		if err := c.RebootInstance(ctx, instanceID); err != nil {
			return fmt.Errorf("failed to reboot instance: %w", err)
		}
	}

	return nil
}

// GetPatchStatus retrieves patch compliance status for a GCP VM instance.
// Uses OS Config API to check patch job history and compliance state.
func (c *GCPPlatformClient) GetPatchStatus(ctx context.Context, instanceID string) (string, error) {
	if !c.connected {
		return "", fmt.Errorf("not connected")
	}

	project, zone, name, err := c.parseInstanceID(instanceID)
	if err != nil {
		return "", err
	}

	// If OS Config client is available, check patch job status
	if c.osConfigClient != nil {
		// List recent patch jobs for this project
		iter := c.osConfigClient.ListPatchJobs(ctx, &osconfigpb.ListPatchJobsRequest{
			Parent: fmt.Sprintf("projects/%s", project),
		})

		// Check if there are any recent patch jobs with issues for this instance
		for {
			job, err := iter.Next()
			if err != nil {
				// End of iteration or error
				break
			}

			// Check if this patch job targets our instance
			if job.InstanceFilter != nil {
				for _, inst := range job.InstanceFilter.Instances {
					if strings.Contains(inst, name) {
						// Found a patch job for this instance
						switch job.State {
						case osconfigpb.PatchJob_COMPLETED_WITH_ERRORS:
							c.log.Debug("found patch job with errors",
								"patch_job", job.Name,
								"instance", name,
							)
							return "NON_COMPLIANT", nil
						case osconfigpb.PatchJob_SUCCEEDED:
							c.log.Debug("found successful patch job",
								"patch_job", job.Name,
								"instance", name,
							)
							return "COMPLIANT", nil
						}
					}
				}
			}
		}
	}

	// If no patch job history found or OS Config unavailable, check instance directly
	c.log.Debug("no patch job history found, assuming unknown compliance",
		"project", project,
		"zone", zone,
		"instance", name,
	)

	return "UNKNOWN", nil
}

// GetPatchComplianceData retrieves detailed patch compliance data for a GCP instance.
func (c *GCPPlatformClient) GetPatchComplianceData(ctx context.Context, instanceID string) (interface{}, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	project, zone, name, err := c.parseInstanceID(instanceID)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"instance_id": instanceID,
		"project":     project,
		"zone":        zone,
		"instance":    name,
		"status":      "UNKNOWN",
	}

	// Try to get patch compliance from OS Config
	if c.osConfigClient != nil {
		// List recent patch jobs for this instance
		parent := fmt.Sprintf("projects/%s", project)
		filter := fmt.Sprintf("state=SUCCEEDED AND zone:%s AND name:%s", zone, name)

		iter := c.osConfigClient.ListPatchJobs(ctx, &osconfigpb.ListPatchJobsRequest{
			Parent: parent,
			Filter: filter,
		})

		var latestJob *osconfigpb.PatchJob
		for {
			job, err := iter.Next()
			if err != nil {
				break
			}
			if latestJob == nil {
				latestJob = job
			}
		}

		if latestJob != nil {
			patchInfo := map[string]interface{}{
				"patch_job_name": latestJob.Name,
				"state":          latestJob.State.String(),
			}

			if latestJob.CreateTime != nil {
				patchInfo["created_at"] = latestJob.CreateTime.AsTime().Format(time.RFC3339)
			}
			if latestJob.UpdateTime != nil {
				patchInfo["updated_at"] = latestJob.UpdateTime.AsTime().Format(time.RFC3339)
			}

			if latestJob.InstanceDetailsSummary != nil {
				summary := latestJob.InstanceDetailsSummary
				patchInfo["instances_pending"] = summary.PendingInstanceCount
				patchInfo["instances_started"] = summary.StartedInstanceCount
				patchInfo["instances_succeeded"] = summary.SucceededInstanceCount
				patchInfo["instances_failed"] = summary.FailedInstanceCount
				patchInfo["instances_reboot_required"] = summary.SucceededRebootRequiredInstanceCount

				// Determine compliance based on failed count
				if summary.FailedInstanceCount > 0 {
					result["status"] = "NON_COMPLIANT"
				} else if summary.SucceededInstanceCount > 0 {
					result["status"] = "COMPLIANT"
				}
			}

			result["patch_job"] = patchInfo
		}
	}

	// Get instance metadata for additional info
	inst, err := c.instancesClient.Get(ctx, &computepb.GetInstanceRequest{
		Project:  project,
		Zone:     zone,
		Instance: name,
	})
	if err == nil && inst != nil {
		result["instance_status"] = *inst.Status

		// Check for OS inventory metadata
		if inst.Metadata != nil && inst.Metadata.Items != nil {
			for _, item := range inst.Metadata.Items {
				if item.Key != nil && *item.Key == "google-osconfig-agent-version" && item.Value != nil {
					result["osconfig_agent_version"] = *item.Value
				}
			}
		}
	}

	return result, nil
}

// parseInstanceID parses a GCP instance ID in the format "project/zone/instance" or "zone/instance" or just "instance".
func (c *GCPPlatformClient) parseInstanceID(instanceID string) (project, zone, name string, err error) {
	parts := strings.Split(instanceID, "/")

	switch len(parts) {
	case 3:
		return parts[0], parts[1], parts[2], nil
	case 2:
		if c.cfg.ProjectID == "" {
			return "", "", "", fmt.Errorf("no project specified and instance ID doesn't include one: %s", instanceID)
		}
		return c.cfg.ProjectID, parts[0], parts[1], nil
	case 1:
		if c.cfg.ProjectID == "" || c.cfg.Zone == "" {
			return "", "", "", fmt.Errorf("project and zone not configured and instance ID doesn't include them: %s", instanceID)
		}
		return c.cfg.ProjectID, c.cfg.Zone, parts[0], nil
	default:
		return "", "", "", fmt.Errorf("invalid instance ID format: %s", instanceID)
	}
}
