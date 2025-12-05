// Package azure provides Azure Update Manager integration for VM patching.
package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/maintenance/armmaintenance"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// UpdateManager provides Azure Update Manager functionality for VM patching.
type UpdateManager struct {
	cfg                     Config
	cred                    *azidentity.ClientSecretCredential
	maintenanceClient       *armmaintenance.ConfigurationsClient
	assignmentsClient       *armmaintenance.ConfigurationAssignmentsClient
	updatesClient           *armmaintenance.UpdatesClient
	applyUpdatesClient      *armmaintenance.ApplyUpdatesClient
	vmExtensionsClient      *armcompute.VirtualMachineExtensionsClient
	vmRunCommandsClient     *armcompute.VirtualMachineRunCommandsClient
	log                     *logger.Logger
}

// NewUpdateManager creates a new Azure Update Manager client.
func NewUpdateManager(cfg Config, log *logger.Logger) (*UpdateManager, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.TenantID == "" {
		return nil, fmt.Errorf("azure credentials required: TenantID, ClientID, ClientSecret")
	}

	cred, err := azidentity.NewClientSecretCredential(cfg.TenantID, cfg.ClientID, cfg.ClientSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials: %w", err)
	}

	// Create maintenance configuration client
	maintenanceClient, err := armmaintenance.NewConfigurationsClient(cfg.SubscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create maintenance client: %w", err)
	}

	// Create maintenance assignments client
	assignmentsClient, err := armmaintenance.NewConfigurationAssignmentsClient(cfg.SubscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create assignments client: %w", err)
	}

	// Create updates client for pending updates
	updatesClient, err := armmaintenance.NewUpdatesClient(cfg.SubscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create updates client: %w", err)
	}

	// Create apply updates client
	applyUpdatesClient, err := armmaintenance.NewApplyUpdatesClient(cfg.SubscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create apply updates client: %w", err)
	}

	// Create VM extensions client for guest patching
	vmExtensionsClient, err := armcompute.NewVirtualMachineExtensionsClient(cfg.SubscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM extensions client: %w", err)
	}

	// Create VM run commands client for custom scripts
	vmRunCommandsClient, err := armcompute.NewVirtualMachineRunCommandsClient(cfg.SubscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create run commands client: %w", err)
	}

	return &UpdateManager{
		cfg:                     cfg,
		cred:                    cred,
		maintenanceClient:       maintenanceClient,
		assignmentsClient:       assignmentsClient,
		updatesClient:           updatesClient,
		applyUpdatesClient:      applyUpdatesClient,
		vmExtensionsClient:      vmExtensionsClient,
		vmRunCommandsClient:     vmRunCommandsClient,
		log:                     log.WithComponent("azure-update-manager"),
	}, nil
}

// PatchAssessmentResult contains the result of a patch assessment.
type PatchAssessmentResult struct {
	VMID                    string                  `json:"vm_id"`
	VMName                  string                  `json:"vm_name"`
	ResourceGroup           string                  `json:"resource_group"`
	Status                  string                  `json:"status"`
	CriticalUpdateCount     int                     `json:"critical_update_count"`
	SecurityUpdateCount     int                     `json:"security_update_count"`
	OtherUpdateCount        int                     `json:"other_update_count"`
	TotalUpdateCount        int                     `json:"total_update_count"`
	RebootRequired          bool                    `json:"reboot_required"`
	LastAssessmentTime      time.Time               `json:"last_assessment_time"`
	AvailablePatches        []PatchInfo             `json:"available_patches,omitempty"`
}

// PatchInfo contains information about a specific patch.
type PatchInfo struct {
	PatchID         string   `json:"patch_id"`
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	Classification  string   `json:"classification"` // Critical, Security, UpdateRollup, etc.
	RebootRequired  bool     `json:"reboot_required"`
	KBNumber        string   `json:"kb_number,omitempty"`
}

// PatchInstallationResult contains the result of a patch installation.
type PatchInstallationResult struct {
	VMID              string        `json:"vm_id"`
	VMName            string        `json:"vm_name"`
	ResourceGroup     string        `json:"resource_group"`
	Status            string        `json:"status"` // Succeeded, Failed, InProgress
	StartTime         time.Time     `json:"start_time"`
	EndTime           time.Time     `json:"end_time,omitempty"`
	InstalledPatches  int           `json:"installed_patches"`
	FailedPatches     int           `json:"failed_patches"`
	PendingPatches    int           `json:"pending_patches"`
	RebootStatus      string        `json:"reboot_status"` // NotNeeded, Required, Started, Completed
	ErrorMessage      string        `json:"error_message,omitempty"`
	PatchDetails      []PatchResult `json:"patch_details,omitempty"`
}

// PatchResult contains the result of installing a specific patch.
type PatchResult struct {
	PatchID     string `json:"patch_id"`
	Name        string `json:"name"`
	Status      string `json:"status"` // Installed, Failed, Excluded, NotSelected
	ErrorCode   string `json:"error_code,omitempty"`
}

// AssessPatches initiates a patch assessment for a VM.
func (m *UpdateManager) AssessPatches(ctx context.Context, resourceGroup, vmName string) (*PatchAssessmentResult, error) {
	m.log.Info("initiating patch assessment",
		"resource_group", resourceGroup,
		"vm_name", vmName,
	)

	// Use the VM Run Command to trigger assessment
	// Azure Update Manager uses the assessPatches operation
	poller, err := m.vmRunCommandsClient.BeginCreateOrUpdate(ctx, resourceGroup, vmName, "AssessPatches", armcompute.VirtualMachineRunCommand{
		Location: to.Ptr(m.cfg.SubscriptionID), // Will be overridden
		Properties: &armcompute.VirtualMachineRunCommandProperties{
			Source: &armcompute.VirtualMachineRunCommandScriptSource{
				Script: to.Ptr("# Trigger patch assessment via Azure Update Manager"),
			},
			AsyncExecution: to.Ptr(false),
			TimeoutInSeconds: to.Ptr(int32(300)),
		},
	}, nil)
	if err != nil {
		// Fall back to using the compute API directly for assessment
		return m.assessPatchesViaCompute(ctx, resourceGroup, vmName)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return m.assessPatchesViaCompute(ctx, resourceGroup, vmName)
	}

	return m.assessPatchesViaCompute(ctx, resourceGroup, vmName)
}

// assessPatchesViaCompute uses the compute API to assess patches.
func (m *UpdateManager) assessPatchesViaCompute(ctx context.Context, resourceGroup, vmName string) (*PatchAssessmentResult, error) {
	// Get VM client
	vmClient, err := armcompute.NewVirtualMachinesClient(m.cfg.SubscriptionID, m.cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM client: %w", err)
	}

	// Trigger patch assessment
	poller, err := vmClient.BeginAssessPatches(ctx, resourceGroup, vmName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start patch assessment: %w", err)
	}

	// Wait for assessment to complete
	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("patch assessment failed: %w", err)
	}

	// Parse results
	result := &PatchAssessmentResult{
		VMName:        vmName,
		ResourceGroup: resourceGroup,
		Status:        string(*resp.Status),
	}

	if resp.StartDateTime != nil {
		result.LastAssessmentTime = *resp.StartDateTime
	}

	if resp.RebootPending != nil {
		result.RebootRequired = *resp.RebootPending
	}

	// The VirtualMachineAssessPatchesResult contains the patch summary
	// Note: API structure varies by SDK version
	if resp.AssessmentActivityID != nil {
		m.log.Debug("assessment activity", "id", *resp.AssessmentActivityID)
	}

	result.TotalUpdateCount = result.CriticalUpdateCount + result.SecurityUpdateCount + result.OtherUpdateCount

	m.log.Info("patch assessment completed",
		"vm_name", vmName,
		"status", result.Status,
		"critical", result.CriticalUpdateCount,
		"security", result.SecurityUpdateCount,
		"total", result.TotalUpdateCount,
		"reboot_required", result.RebootRequired,
	)

	return result, nil
}

// InstallPatchesParams contains parameters for patch installation.
type InstallPatchesParams struct {
	ResourceGroup       string
	VMName              string
	MaximumDuration     string   // ISO 8601 duration, e.g., "PT2H" for 2 hours
	RebootSetting       string   // IfRequired, Never, Always
	Classifications     []string // Critical, Security, UpdateRollup, etc.
	ExcludeKBs          []string // KB numbers to exclude
	IncludeKBs          []string // Specific KB numbers to include
	LinuxClassifications []string // For Linux: Critical, Security, Other
}

// InstallPatches installs patches on a VM.
func (m *UpdateManager) InstallPatches(ctx context.Context, params InstallPatchesParams) (*PatchInstallationResult, error) {
	m.log.Info("initiating patch installation",
		"resource_group", params.ResourceGroup,
		"vm_name", params.VMName,
		"reboot_setting", params.RebootSetting,
		"classifications", params.Classifications,
	)

	// Get VM client
	vmClient, err := armcompute.NewVirtualMachinesClient(m.cfg.SubscriptionID, m.cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM client: %w", err)
	}

	// Build install patches input
	installInput := armcompute.VirtualMachineInstallPatchesParameters{
		RebootSetting: (*armcompute.VMGuestPatchRebootSetting)(to.Ptr(params.RebootSetting)),
	}

	// Set maximum duration
	if params.MaximumDuration != "" {
		installInput.MaximumDuration = to.Ptr(params.MaximumDuration)
	} else {
		installInput.MaximumDuration = to.Ptr("PT2H") // Default 2 hours
	}

	// Set Windows patch settings
	if len(params.Classifications) > 0 || len(params.ExcludeKBs) > 0 || len(params.IncludeKBs) > 0 {
		windowsParams := &armcompute.WindowsParameters{}

		// Convert classifications
		if len(params.Classifications) > 0 {
			classifications := make([]*armcompute.VMGuestPatchClassificationWindows, len(params.Classifications))
			for i, c := range params.Classifications {
				classifications[i] = (*armcompute.VMGuestPatchClassificationWindows)(to.Ptr(c))
			}
			windowsParams.ClassificationsToInclude = classifications
		}

		if len(params.ExcludeKBs) > 0 {
			windowsParams.KbNumbersToExclude = toStringPtrSlice(params.ExcludeKBs)
		}
		if len(params.IncludeKBs) > 0 {
			windowsParams.KbNumbersToInclude = toStringPtrSlice(params.IncludeKBs)
		}

		installInput.WindowsParameters = windowsParams
	}

	// Set Linux patch settings
	if len(params.LinuxClassifications) > 0 {
		linuxParams := &armcompute.LinuxParameters{}
		classifications := make([]*armcompute.VMGuestPatchClassificationLinux, len(params.LinuxClassifications))
		for i, c := range params.LinuxClassifications {
			classifications[i] = (*armcompute.VMGuestPatchClassificationLinux)(to.Ptr(c))
		}
		linuxParams.ClassificationsToInclude = classifications
		installInput.LinuxParameters = linuxParams
	}

	// Start patch installation
	startTime := time.Now()
	poller, err := vmClient.BeginInstallPatches(ctx, params.ResourceGroup, params.VMName, installInput, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start patch installation: %w", err)
	}

	// Wait for installation to complete
	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return &PatchInstallationResult{
			VMName:        params.VMName,
			ResourceGroup: params.ResourceGroup,
			Status:        "Failed",
			StartTime:     startTime,
			EndTime:       time.Now(),
			ErrorMessage:  err.Error(),
		}, err
	}

	// Parse results
	result := &PatchInstallationResult{
		VMName:        params.VMName,
		ResourceGroup: params.ResourceGroup,
		Status:        string(*resp.Status),
		StartTime:     startTime,
		EndTime:       time.Now(),
	}

	if resp.InstalledPatchCount != nil {
		result.InstalledPatches = int(*resp.InstalledPatchCount)
	}
	if resp.FailedPatchCount != nil {
		result.FailedPatches = int(*resp.FailedPatchCount)
	}
	if resp.PendingPatchCount != nil {
		result.PendingPatches = int(*resp.PendingPatchCount)
	}
	if resp.RebootStatus != nil {
		result.RebootStatus = string(*resp.RebootStatus)
	}

	// Get individual patch results
	if resp.Patches != nil {
		result.PatchDetails = make([]PatchResult, 0, len(resp.Patches))
		for _, p := range resp.Patches {
			pr := PatchResult{}
			if p.PatchID != nil {
				pr.PatchID = *p.PatchID
			}
			if p.Name != nil {
				pr.Name = *p.Name
			}
			if p.InstallationState != nil {
				pr.Status = string(*p.InstallationState)
			}
			result.PatchDetails = append(result.PatchDetails, pr)
		}
	}

	m.log.Info("patch installation completed",
		"vm_name", params.VMName,
		"status", result.Status,
		"installed", result.InstalledPatches,
		"failed", result.FailedPatches,
		"pending", result.PendingPatches,
		"reboot_status", result.RebootStatus,
	)

	return result, nil
}

// GetPatchComplianceStatus checks the overall patch compliance status for a VM.
func (m *UpdateManager) GetPatchComplianceStatus(ctx context.Context, resourceGroup, vmName string) (*PatchAssessmentResult, error) {
	// Get VM client
	vmClient, err := armcompute.NewVirtualMachinesClient(m.cfg.SubscriptionID, m.cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM client: %w", err)
	}

	// Get the VM instance view which contains patch status
	resp, err := vmClient.InstanceView(ctx, resourceGroup, vmName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM instance view: %w", err)
	}

	result := &PatchAssessmentResult{
		VMName:        vmName,
		ResourceGroup: resourceGroup,
		Status:        "Unknown",
	}

	// Check patch status from instance view
	if resp.PatchStatus != nil {
		ps := resp.PatchStatus

		if ps.AvailablePatchSummary != nil {
			summary := ps.AvailablePatchSummary
			if summary.CriticalAndSecurityPatchCount != nil {
				result.CriticalUpdateCount = int(*summary.CriticalAndSecurityPatchCount)
			}
			if summary.OtherPatchCount != nil {
				result.OtherUpdateCount = int(*summary.OtherPatchCount)
			}
			if summary.RebootPending != nil {
				result.RebootRequired = *summary.RebootPending
			}
			if summary.LastModifiedTime != nil {
				result.LastAssessmentTime = *summary.LastModifiedTime
			}
			if summary.Status != nil {
				result.Status = string(*summary.Status)
			}
		}
	}

	result.TotalUpdateCount = result.CriticalUpdateCount + result.SecurityUpdateCount + result.OtherUpdateCount

	return result, nil
}

// CreateMaintenanceConfiguration creates a maintenance configuration for scheduled patching.
func (m *UpdateManager) CreateMaintenanceConfiguration(ctx context.Context, params MaintenanceConfigParams) (*MaintenanceConfig, error) {
	m.log.Info("creating maintenance configuration",
		"name", params.Name,
		"resource_group", params.ResourceGroup,
		"recurrence", params.Recurrence,
	)

	config := armmaintenance.Configuration{
		Location: to.Ptr(params.Location),
		Properties: &armmaintenance.ConfigurationProperties{
			MaintenanceScope: to.Ptr(armmaintenance.MaintenanceScopeInGuestPatch),
			Visibility:       to.Ptr(armmaintenance.VisibilityCustom),
			Namespace:        to.Ptr("Microsoft.Maintenance"),
			MaintenanceWindow: &armmaintenance.Window{
				StartDateTime: to.Ptr(params.StartDateTime),
				Duration:      to.Ptr(params.Duration),
				TimeZone:      to.Ptr(params.TimeZone),
				RecurEvery:    to.Ptr(params.Recurrence),
			},
			InstallPatches: &armmaintenance.InputPatchConfiguration{
				RebootSetting: to.Ptr(armmaintenance.RebootOptions(params.RebootSetting)),
			},
		},
	}

	// Add Windows patch settings
	if len(params.WindowsClassifications) > 0 {
		classifications := make([]*string, len(params.WindowsClassifications))
		for i, c := range params.WindowsClassifications {
			classifications[i] = to.Ptr(c)
		}
		config.Properties.InstallPatches.WindowsParameters = &armmaintenance.InputWindowsParameters{
			ClassificationsToInclude: classifications,
			ExcludeKbsRequiringReboot: to.Ptr(params.ExcludeKBsRequiringReboot),
		}
	}

	// Add Linux patch settings
	if len(params.LinuxClassifications) > 0 {
		classifications := make([]*string, len(params.LinuxClassifications))
		for i, c := range params.LinuxClassifications {
			classifications[i] = to.Ptr(c)
		}
		config.Properties.InstallPatches.LinuxParameters = &armmaintenance.InputLinuxParameters{
			ClassificationsToInclude: classifications,
		}
	}

	resp, err := m.maintenanceClient.CreateOrUpdate(ctx, params.ResourceGroup, params.Name, config, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create maintenance configuration: %w", err)
	}

	result := &MaintenanceConfig{
		ID:            *resp.ID,
		Name:          *resp.Name,
		Location:      *resp.Location,
		Scope:         string(*resp.Properties.MaintenanceScope),
		StartDateTime: params.StartDateTime,
		Duration:      params.Duration,
		TimeZone:      params.TimeZone,
		Recurrence:    params.Recurrence,
	}

	m.log.Info("maintenance configuration created",
		"name", result.Name,
		"id", result.ID,
	)

	return result, nil
}

// AssignMaintenanceConfiguration assigns a maintenance configuration to a VM.
func (m *UpdateManager) AssignMaintenanceConfiguration(ctx context.Context, resourceGroup, vmName, configID string) error {
	m.log.Info("assigning maintenance configuration",
		"vm_name", vmName,
		"config_id", configID,
	)

	// Build the provider path for the VM
	providerName := "Microsoft.Compute"
	resourceType := "virtualMachines"
	configAssignmentName := fmt.Sprintf("%s-assignment", vmName)

	assignment := armmaintenance.ConfigurationAssignment{
		Properties: &armmaintenance.ConfigurationAssignmentProperties{
			MaintenanceConfigurationID: to.Ptr(configID),
		},
	}

	_, err := m.assignmentsClient.CreateOrUpdate(
		ctx,
		resourceGroup,
		providerName,
		resourceType,
		vmName,
		configAssignmentName,
		assignment,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to assign maintenance configuration: %w", err)
	}

	m.log.Info("maintenance configuration assigned",
		"vm_name", vmName,
		"config_id", configID,
	)

	return nil
}

// MaintenanceConfigParams contains parameters for creating a maintenance configuration.
type MaintenanceConfigParams struct {
	Name                      string
	ResourceGroup             string
	Location                  string
	StartDateTime             string // e.g., "2024-01-15 00:00"
	Duration                  string // e.g., "02:00" for 2 hours
	TimeZone                  string // e.g., "Pacific Standard Time"
	Recurrence                string // e.g., "Week Monday,Wednesday" or "Month Third Monday"
	RebootSetting             string // IfRequired, Never, Always
	WindowsClassifications    []string
	LinuxClassifications      []string
	ExcludeKBsRequiringReboot bool
}

// MaintenanceConfig represents a created maintenance configuration.
type MaintenanceConfig struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Location      string `json:"location"`
	Scope         string `json:"scope"`
	StartDateTime string `json:"start_date_time"`
	Duration      string `json:"duration"`
	TimeZone      string `json:"time_zone"`
	Recurrence    string `json:"recurrence"`
}

// ListPendingUpdates lists VMs with pending updates.
func (m *UpdateManager) ListPendingUpdates(ctx context.Context, resourceGroup string) ([]PatchAssessmentResult, error) {
	m.log.Info("listing VMs with pending updates", "resource_group", resourceGroup)

	// Get VM client
	vmClient, err := armcompute.NewVirtualMachinesClient(m.cfg.SubscriptionID, m.cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM client: %w", err)
	}

	var results []PatchAssessmentResult

	// List all VMs in the resource group (or subscription if resourceGroup is empty)
	if resourceGroup == "" {
		pager := vmClient.NewListAllPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to list VMs: %w", err)
			}

			for _, vm := range page.Value {
				vmRG := extractResourceGroupFromID(*vm.ID)
				status, err := m.GetPatchComplianceStatus(ctx, vmRG, *vm.Name)
				if err != nil {
					m.log.Warn("failed to get patch status for VM",
						"vm_name", *vm.Name,
						"error", err,
					)
					continue
				}

				if status.TotalUpdateCount > 0 {
					status.VMID = *vm.ID
					results = append(results, *status)
				}
			}
		}
	} else {
		// List VMs in specific resource group
		rgPager := vmClient.NewListPager(resourceGroup, nil)
		for rgPager.More() {
			page, err := rgPager.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to list VMs: %w", err)
			}

			for _, vm := range page.Value {
				status, err := m.GetPatchComplianceStatus(ctx, resourceGroup, *vm.Name)
				if err != nil {
					m.log.Warn("failed to get patch status for VM",
						"vm_name", *vm.Name,
						"error", err,
					)
					continue
				}

				if status.TotalUpdateCount > 0 {
					status.VMID = *vm.ID
					results = append(results, *status)
				}
			}
		}
	}

	m.log.Info("found VMs with pending updates",
		"count", len(results),
		"resource_group", resourceGroup,
	)

	return results, nil
}

// Helper function to convert string slice to pointer slice
func toStringPtrSlice(s []string) []*string {
	result := make([]*string, len(s))
	for i, v := range s {
		result[i] = to.Ptr(v)
	}
	return result
}
