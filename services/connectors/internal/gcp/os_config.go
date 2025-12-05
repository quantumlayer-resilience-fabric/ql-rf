// Package gcp provides GCP OS Config Agent integration for VM patching.
package gcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	osconfig "cloud.google.com/go/osconfig/apiv1"
	"cloud.google.com/go/osconfig/apiv1/osconfigpb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/type/datetime"
	"google.golang.org/genproto/googleapis/type/dayofweek"
	"google.golang.org/genproto/googleapis/type/timeofday"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// OSConfigManager provides GCP OS Config functionality for VM patching.
type OSConfigManager struct {
	cfg                 Config
	patchJobsClient     *osconfig.OsConfigZonalClient
	patchDeployClient   *osconfig.OsConfigZonalClient
	log                 *logger.Logger
}

// NewOSConfigManager creates a new GCP OS Config Manager client.
func NewOSConfigManager(cfg Config, log *logger.Logger) (*OSConfigManager, error) {
	ctx := context.Background()

	var opts []option.ClientOption
	if cfg.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
	}

	// Create OS Config zonal client
	zonalClient, err := osconfig.NewOsConfigZonalClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OS Config client: %w", err)
	}

	return &OSConfigManager{
		cfg:             cfg,
		patchJobsClient: zonalClient,
		log:             log.WithComponent("gcp-os-config"),
	}, nil
}

// Close closes the OS Config clients.
func (m *OSConfigManager) Close() error {
	if m.patchJobsClient != nil {
		return m.patchJobsClient.Close()
	}
	return nil
}

// PatchJobResult contains the result of a patch job.
type PatchJobResult struct {
	JobID               string            `json:"job_id"`
	DisplayName         string            `json:"display_name"`
	State               string            `json:"state"`
	CreateTime          time.Time         `json:"create_time"`
	UpdateTime          time.Time         `json:"update_time"`
	Duration            time.Duration     `json:"duration,omitempty"`
	InstanceDetailCount InstanceCounts    `json:"instance_counts"`
	PatchConfig         *PatchConfigInfo  `json:"patch_config,omitempty"`
	ErrorMessage        string            `json:"error_message,omitempty"`
}

// InstanceCounts tracks instance states in a patch job.
type InstanceCounts struct {
	PendingInstances    int64 `json:"pending_instances"`
	InactiveInstances   int64 `json:"inactive_instances"`
	NotifiedInstances   int64 `json:"notified_instances"`
	StartedInstances    int64 `json:"started_instances"`
	DownloadingInstances int64 `json:"downloading_instances"`
	ApplyingInstances   int64 `json:"applying_instances"`
	RebootingInstances  int64 `json:"rebooting_instances"`
	SucceededInstances  int64 `json:"succeeded_instances"`
	FailedInstances     int64 `json:"failed_instances"`
	AckedInstances      int64 `json:"acked_instances"`
	TimedOutInstances   int64 `json:"timed_out_instances"`
	PrePatchStepInstances int64 `json:"pre_patch_step_instances"`
	PostPatchStepInstances int64 `json:"post_patch_step_instances"`
	NoAgentInstances    int64 `json:"no_agent_instances"`
}

// PatchConfigInfo contains patch configuration details.
type PatchConfigInfo struct {
	RebootConfig       string   `json:"reboot_config"`
	AptConfig          *AptConfig `json:"apt_config,omitempty"`
	YumConfig          *YumConfig `json:"yum_config,omitempty"`
	WindowsUpdateConfig *WindowsUpdateConfig `json:"windows_update_config,omitempty"`
}

// AptConfig contains APT-specific patching configuration.
type AptConfig struct {
	Type     string   `json:"type"` // DIST, UPGRADE
	Excludes []string `json:"excludes,omitempty"`
}

// YumConfig contains YUM-specific patching configuration.
type YumConfig struct {
	Security bool     `json:"security"`
	Minimal  bool     `json:"minimal"`
	Excludes []string `json:"excludes,omitempty"`
}

// WindowsUpdateConfig contains Windows Update configuration.
type WindowsUpdateConfig struct {
	Classifications []string `json:"classifications"` // CRITICAL, SECURITY, etc.
	Excludes        []string `json:"excludes,omitempty"`
}

// ExecutePatchJobParams contains parameters for executing a patch job.
type ExecutePatchJobParams struct {
	DisplayName         string
	Description         string
	InstanceFilter      InstanceFilter
	PatchConfig         PatchConfig
	DurationSeconds     int64
	DryRun              bool
}

// InstanceFilter defines which instances to patch.
type InstanceFilter struct {
	All             bool              // Patch all instances
	GroupLabels     []map[string]string // Patch instances matching labels
	Zones           []string          // Patch instances in specific zones
	Instances       []string          // Specific instance URIs
	InstanceNamePrefixes []string      // Instance name prefixes
}

// PatchConfig defines how to patch instances.
type PatchConfig struct {
	RebootConfig       string // DEFAULT, ALWAYS, NEVER
	Apt                *AptPatchConfig
	Yum                *YumPatchConfig
	WindowsUpdate      *WindowsUpdatePatchConfig
	PreStep            *ExecStep
	PostStep           *ExecStep
	MigInstancesAllowed bool
}

// AptPatchConfig for Debian/Ubuntu systems.
type AptPatchConfig struct {
	Type              string   // DIST, UPGRADE
	Excludes          []string
	ExclusivePackages []string // Only install these packages
}

// YumPatchConfig for RHEL/CentOS systems.
type YumPatchConfig struct {
	Security          bool
	Minimal           bool
	Excludes          []string
	ExclusivePackages []string
}

// WindowsUpdatePatchConfig for Windows systems.
type WindowsUpdatePatchConfig struct {
	Classifications []string // CRITICAL, SECURITY, DEFINITION, DRIVER, etc.
	Excludes        []string // KB IDs to exclude
	ExclusivePatches []string // Only install these patches
}

// ExecStep defines a script to run before or after patching.
type ExecStep struct {
	LinuxExecStepConfig   *ExecStepConfig
	WindowsExecStepConfig *ExecStepConfig
}

// ExecStepConfig defines the execution configuration for a step.
type ExecStepConfig struct {
	GcsObject    *GcsObject // Script from GCS
	LocalPath    string     // Local script path
	AllowedSuccessCodes []int32
	Interpreter  string    // SHELL, POWERSHELL
}

// GcsObject references a script in Google Cloud Storage.
type GcsObject struct {
	Bucket         string
	Object         string
	GenerationNumber int64
}

// ExecutePatchJob executes a patch job on GCP instances.
func (m *OSConfigManager) ExecutePatchJob(ctx context.Context, params ExecutePatchJobParams) (*PatchJobResult, error) {
	m.log.Info("executing patch job",
		"display_name", params.DisplayName,
		"dry_run", params.DryRun,
	)

	// Build instance filter
	instanceFilter := &osconfigpb.PatchInstanceFilter{}

	if params.InstanceFilter.All {
		instanceFilter.All = true
	} else {
		if len(params.InstanceFilter.GroupLabels) > 0 {
			for _, labels := range params.InstanceFilter.GroupLabels {
				instanceFilter.GroupLabels = append(instanceFilter.GroupLabels, &osconfigpb.PatchInstanceFilter_GroupLabel{
					Labels: labels,
				})
			}
		}
		if len(params.InstanceFilter.Zones) > 0 {
			instanceFilter.Zones = params.InstanceFilter.Zones
		}
		if len(params.InstanceFilter.Instances) > 0 {
			instanceFilter.Instances = params.InstanceFilter.Instances
		}
		if len(params.InstanceFilter.InstanceNamePrefixes) > 0 {
			instanceFilter.InstanceNamePrefixes = params.InstanceFilter.InstanceNamePrefixes
		}
	}

	// Build patch config
	patchConfig := &osconfigpb.PatchConfig{
		MigInstancesAllowed: params.PatchConfig.MigInstancesAllowed,
	}

	// Set reboot config
	switch params.PatchConfig.RebootConfig {
	case "ALWAYS":
		patchConfig.RebootConfig = osconfigpb.PatchConfig_ALWAYS
	case "NEVER":
		patchConfig.RebootConfig = osconfigpb.PatchConfig_NEVER
	default:
		patchConfig.RebootConfig = osconfigpb.PatchConfig_DEFAULT
	}

	// Set APT config
	if params.PatchConfig.Apt != nil {
		patchConfig.Apt = &osconfigpb.AptSettings{
			Excludes:          params.PatchConfig.Apt.Excludes,
			ExclusivePackages: params.PatchConfig.Apt.ExclusivePackages,
		}
		switch params.PatchConfig.Apt.Type {
		case "DIST":
			patchConfig.Apt.Type = osconfigpb.AptSettings_DIST
		default:
			patchConfig.Apt.Type = osconfigpb.AptSettings_UPGRADE
		}
	}

	// Set YUM config
	if params.PatchConfig.Yum != nil {
		patchConfig.Yum = &osconfigpb.YumSettings{
			Security:          params.PatchConfig.Yum.Security,
			Minimal:           params.PatchConfig.Yum.Minimal,
			Excludes:          params.PatchConfig.Yum.Excludes,
			ExclusivePackages: params.PatchConfig.Yum.ExclusivePackages,
		}
	}

	// Set Windows Update config
	if params.PatchConfig.WindowsUpdate != nil {
		patchConfig.WindowsUpdate = &osconfigpb.WindowsUpdateSettings{
			ExclusivePatches: params.PatchConfig.WindowsUpdate.ExclusivePatches,
		}
		for _, c := range params.PatchConfig.WindowsUpdate.Classifications {
			switch strings.ToUpper(c) {
			case "CRITICAL":
				patchConfig.WindowsUpdate.Classifications = append(
					patchConfig.WindowsUpdate.Classifications,
					osconfigpb.WindowsUpdateSettings_CRITICAL,
				)
			case "SECURITY":
				patchConfig.WindowsUpdate.Classifications = append(
					patchConfig.WindowsUpdate.Classifications,
					osconfigpb.WindowsUpdateSettings_SECURITY,
				)
			case "DEFINITION":
				patchConfig.WindowsUpdate.Classifications = append(
					patchConfig.WindowsUpdate.Classifications,
					osconfigpb.WindowsUpdateSettings_DEFINITION,
				)
			case "DRIVER":
				patchConfig.WindowsUpdate.Classifications = append(
					patchConfig.WindowsUpdate.Classifications,
					osconfigpb.WindowsUpdateSettings_DRIVER,
				)
			case "FEATURE_PACK":
				patchConfig.WindowsUpdate.Classifications = append(
					patchConfig.WindowsUpdate.Classifications,
					osconfigpb.WindowsUpdateSettings_FEATURE_PACK,
				)
			case "SERVICE_PACK":
				patchConfig.WindowsUpdate.Classifications = append(
					patchConfig.WindowsUpdate.Classifications,
					osconfigpb.WindowsUpdateSettings_SERVICE_PACK,
				)
			case "TOOL":
				patchConfig.WindowsUpdate.Classifications = append(
					patchConfig.WindowsUpdate.Classifications,
					osconfigpb.WindowsUpdateSettings_TOOL,
				)
			case "UPDATE_ROLLUP":
				patchConfig.WindowsUpdate.Classifications = append(
					patchConfig.WindowsUpdate.Classifications,
					osconfigpb.WindowsUpdateSettings_UPDATE_ROLLUP,
				)
			case "UPDATE":
				patchConfig.WindowsUpdate.Classifications = append(
					patchConfig.WindowsUpdate.Classifications,
					osconfigpb.WindowsUpdateSettings_UPDATE,
				)
			}
		}
	}

	// Build request
	req := &osconfigpb.ExecutePatchJobRequest{
		Parent:         fmt.Sprintf("projects/%s", m.cfg.ProjectID),
		DisplayName:    params.DisplayName,
		Description:    params.Description,
		InstanceFilter: instanceFilter,
		PatchConfig:    patchConfig,
		DryRun:         params.DryRun,
	}

	if params.DurationSeconds > 0 {
		req.Duration = durationpb.New(time.Duration(params.DurationSeconds) * time.Second)
	}

	// Execute the patch job using the non-zonal client
	// Note: We need to use the regular OS Config client for patch jobs
	client, err := osconfig.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OS Config client: %w", err)
	}
	defer client.Close()

	patchJob, err := client.ExecutePatchJob(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute patch job: %w", err)
	}

	result := m.convertPatchJob(patchJob)

	m.log.Info("patch job initiated",
		"job_id", result.JobID,
		"state", result.State,
		"display_name", result.DisplayName,
	)

	return result, nil
}

// GetPatchJob retrieves the status of a patch job.
func (m *OSConfigManager) GetPatchJob(ctx context.Context, jobID string) (*PatchJobResult, error) {
	client, err := osconfig.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OS Config client: %w", err)
	}
	defer client.Close()

	req := &osconfigpb.GetPatchJobRequest{
		Name: jobID,
	}

	patchJob, err := client.GetPatchJob(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get patch job: %w", err)
	}

	return m.convertPatchJob(patchJob), nil
}

// ListPatchJobs lists all patch jobs for the project.
func (m *OSConfigManager) ListPatchJobs(ctx context.Context, filter string, pageSize int32) ([]*PatchJobResult, error) {
	client, err := osconfig.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OS Config client: %w", err)
	}
	defer client.Close()

	req := &osconfigpb.ListPatchJobsRequest{
		Parent:   fmt.Sprintf("projects/%s", m.cfg.ProjectID),
		Filter:   filter,
		PageSize: pageSize,
	}

	var results []*PatchJobResult
	it := client.ListPatchJobs(ctx, req)
	for {
		job, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list patch jobs: %w", err)
		}
		results = append(results, m.convertPatchJob(job))
	}

	return results, nil
}

// CancelPatchJob cancels a running patch job.
func (m *OSConfigManager) CancelPatchJob(ctx context.Context, jobID string) (*PatchJobResult, error) {
	client, err := osconfig.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OS Config client: %w", err)
	}
	defer client.Close()

	req := &osconfigpb.CancelPatchJobRequest{
		Name: jobID,
	}

	patchJob, err := client.CancelPatchJob(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel patch job: %w", err)
	}

	m.log.Info("patch job cancelled", "job_id", jobID)

	return m.convertPatchJob(patchJob), nil
}

// GetPatchJobInstanceDetails retrieves details about instances in a patch job.
func (m *OSConfigManager) GetPatchJobInstanceDetails(ctx context.Context, jobID string) ([]InstancePatchDetail, error) {
	client, err := osconfig.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OS Config client: %w", err)
	}
	defer client.Close()

	req := &osconfigpb.ListPatchJobInstanceDetailsRequest{
		Parent: jobID,
	}

	var details []InstancePatchDetail
	it := client.ListPatchJobInstanceDetails(ctx, req)
	for {
		detail, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list instance details: %w", err)
		}
		details = append(details, InstancePatchDetail{
			InstanceName:      detail.Name,
			InstanceSystemID:  detail.InstanceSystemId,
			State:             detail.State.String(),
			AttemptCount:      detail.AttemptCount,
			FailureReason:     detail.FailureReason,
		})
	}

	return details, nil
}

// InstancePatchDetail contains patching details for a specific instance.
type InstancePatchDetail struct {
	InstanceName     string `json:"instance_name"`
	InstanceSystemID string `json:"instance_system_id"`
	State            string `json:"state"`
	AttemptCount     int64  `json:"attempt_count"`
	FailureReason    string `json:"failure_reason,omitempty"`
}

// convertPatchJob converts a proto PatchJob to our result type.
func (m *OSConfigManager) convertPatchJob(job *osconfigpb.PatchJob) *PatchJobResult {
	result := &PatchJobResult{
		JobID:       job.Name,
		DisplayName: job.DisplayName,
		State:       job.State.String(),
	}

	if job.CreateTime != nil {
		result.CreateTime = job.CreateTime.AsTime()
	}
	if job.UpdateTime != nil {
		result.UpdateTime = job.UpdateTime.AsTime()
	}
	if job.Duration != nil {
		result.Duration = job.Duration.AsDuration()
	}
	if job.ErrorMessage != "" {
		result.ErrorMessage = job.ErrorMessage
	}

	// Copy instance counts
	if job.InstanceDetailsSummary != nil {
		result.InstanceDetailCount = InstanceCounts{
			PendingInstances:       job.InstanceDetailsSummary.PendingInstanceCount,
			InactiveInstances:      job.InstanceDetailsSummary.InactiveInstanceCount,
			NotifiedInstances:      job.InstanceDetailsSummary.NotifiedInstanceCount,
			StartedInstances:       job.InstanceDetailsSummary.StartedInstanceCount,
			DownloadingInstances:   job.InstanceDetailsSummary.DownloadingPatchesInstanceCount,
			ApplyingInstances:      job.InstanceDetailsSummary.ApplyingPatchesInstanceCount,
			RebootingInstances:     job.InstanceDetailsSummary.RebootingInstanceCount,
			SucceededInstances:     job.InstanceDetailsSummary.SucceededInstanceCount,
			FailedInstances:        job.InstanceDetailsSummary.FailedInstanceCount,
			AckedInstances:         job.InstanceDetailsSummary.AckedInstanceCount,
			TimedOutInstances:      job.InstanceDetailsSummary.TimedOutInstanceCount,
			PrePatchStepInstances:  job.InstanceDetailsSummary.PrePatchStepInstanceCount,
			PostPatchStepInstances: job.InstanceDetailsSummary.PostPatchStepInstanceCount,
			NoAgentInstances:       job.InstanceDetailsSummary.NoAgentDetectedInstanceCount,
		}
	}

	// Copy patch config
	if job.PatchConfig != nil {
		result.PatchConfig = &PatchConfigInfo{
			RebootConfig: job.PatchConfig.RebootConfig.String(),
		}
		if job.PatchConfig.Apt != nil {
			result.PatchConfig.AptConfig = &AptConfig{
				Type:     job.PatchConfig.Apt.Type.String(),
				Excludes: job.PatchConfig.Apt.Excludes,
			}
		}
		if job.PatchConfig.Yum != nil {
			result.PatchConfig.YumConfig = &YumConfig{
				Security: job.PatchConfig.Yum.Security,
				Minimal:  job.PatchConfig.Yum.Minimal,
				Excludes: job.PatchConfig.Yum.Excludes,
			}
		}
		if job.PatchConfig.WindowsUpdate != nil {
			classifications := make([]string, len(job.PatchConfig.WindowsUpdate.Classifications))
			for i, c := range job.PatchConfig.WindowsUpdate.Classifications {
				classifications[i] = c.String()
			}
			result.PatchConfig.WindowsUpdateConfig = &WindowsUpdateConfig{
				Classifications: classifications,
			}
		}
	}

	return result
}

// CreatePatchDeployment creates a scheduled patch deployment.
func (m *OSConfigManager) CreatePatchDeployment(ctx context.Context, params PatchDeploymentParams) (*PatchDeployment, error) {
	m.log.Info("creating patch deployment",
		"deployment_id", params.DeploymentID,
		"description", params.Description,
	)

	client, err := osconfig.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OS Config client: %w", err)
	}
	defer client.Close()

	// Build instance filter
	instanceFilter := &osconfigpb.PatchInstanceFilter{}
	if params.InstanceFilter.All {
		instanceFilter.All = true
	} else {
		instanceFilter.Zones = params.InstanceFilter.Zones
		instanceFilter.InstanceNamePrefixes = params.InstanceFilter.InstanceNamePrefixes
		for _, labels := range params.InstanceFilter.GroupLabels {
			instanceFilter.GroupLabels = append(instanceFilter.GroupLabels, &osconfigpb.PatchInstanceFilter_GroupLabel{
				Labels: labels,
			})
		}
	}

	// Build patch config (same as ExecutePatchJob)
	patchConfig := &osconfigpb.PatchConfig{
		MigInstancesAllowed: params.PatchConfig.MigInstancesAllowed,
	}
	switch params.PatchConfig.RebootConfig {
	case "ALWAYS":
		patchConfig.RebootConfig = osconfigpb.PatchConfig_ALWAYS
	case "NEVER":
		patchConfig.RebootConfig = osconfigpb.PatchConfig_NEVER
	default:
		patchConfig.RebootConfig = osconfigpb.PatchConfig_DEFAULT
	}

	// Build schedule
	var schedule *osconfigpb.PatchDeployment_RecurringSchedule
	if params.Schedule.Frequency != "" {
		recurringSchedule := &osconfigpb.RecurringSchedule{
			TimeZone: &datetime.TimeZone{
				Id: params.Schedule.TimeZone,
			},
			TimeOfDay: params.Schedule.TimeOfDay,
		}

		switch strings.ToUpper(params.Schedule.Frequency) {
		case "WEEKLY":
			recurringSchedule.Frequency = osconfigpb.RecurringSchedule_WEEKLY
			if params.Schedule.WeeklyDay != "" {
				day := dayofweek.DayOfWeek_MONDAY // default
				switch strings.ToUpper(params.Schedule.WeeklyDay) {
				case "TUESDAY":
					day = dayofweek.DayOfWeek_TUESDAY
				case "WEDNESDAY":
					day = dayofweek.DayOfWeek_WEDNESDAY
				case "THURSDAY":
					day = dayofweek.DayOfWeek_THURSDAY
				case "FRIDAY":
					day = dayofweek.DayOfWeek_FRIDAY
				case "SATURDAY":
					day = dayofweek.DayOfWeek_SATURDAY
				case "SUNDAY":
					day = dayofweek.DayOfWeek_SUNDAY
				}
				recurringSchedule.ScheduleConfig = &osconfigpb.RecurringSchedule_Weekly{
					Weekly: &osconfigpb.WeeklySchedule{
						DayOfWeek: day,
					},
				}
			}
		case "MONTHLY":
			recurringSchedule.Frequency = osconfigpb.RecurringSchedule_MONTHLY
			if params.Schedule.MonthlyDay > 0 {
				recurringSchedule.ScheduleConfig = &osconfigpb.RecurringSchedule_Monthly{
					Monthly: &osconfigpb.MonthlySchedule{
						DayOfMonth: &osconfigpb.MonthlySchedule_MonthDay{
							MonthDay: params.Schedule.MonthlyDay,
						},
					},
				}
			}
		}

		schedule = &osconfigpb.PatchDeployment_RecurringSchedule{
			RecurringSchedule: recurringSchedule,
		}
	}

	req := &osconfigpb.CreatePatchDeploymentRequest{
		Parent:            fmt.Sprintf("projects/%s", m.cfg.ProjectID),
		PatchDeploymentId: params.DeploymentID,
		PatchDeployment: &osconfigpb.PatchDeployment{
			Description:    params.Description,
			InstanceFilter: instanceFilter,
			PatchConfig:    patchConfig,
			Schedule:       schedule,
		},
	}

	if params.DurationSeconds > 0 {
		req.PatchDeployment.Duration = durationpb.New(time.Duration(params.DurationSeconds) * time.Second)
	}

	deployment, err := client.CreatePatchDeployment(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create patch deployment: %w", err)
	}

	result := &PatchDeployment{
		Name:        deployment.Name,
		Description: deployment.Description,
		State:       deployment.State.String(),
	}
	if deployment.CreateTime != nil {
		result.CreateTime = deployment.CreateTime.AsTime()
	}

	m.log.Info("patch deployment created",
		"name", result.Name,
		"state", result.State,
	)

	return result, nil
}

// PatchDeploymentParams contains parameters for creating a patch deployment.
type PatchDeploymentParams struct {
	DeploymentID    string
	Description     string
	InstanceFilter  InstanceFilter
	PatchConfig     PatchConfig
	Schedule        PatchSchedule
	DurationSeconds int64
}

// PatchSchedule defines the schedule for a patch deployment.
type PatchSchedule struct {
	Frequency  string // WEEKLY, MONTHLY
	TimeZone   string // e.g., "America/Los_Angeles"
	TimeOfDay  *timeofday.TimeOfDay
	WeeklyDay  string // MONDAY, TUESDAY, etc. (for WEEKLY)
	MonthlyDay int32  // 1-31 (for MONTHLY)
}

// PatchDeployment represents a created patch deployment.
type PatchDeployment struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	State       string    `json:"state"`
	CreateTime  time.Time `json:"create_time"`
}

// DeletePatchDeployment deletes a patch deployment.
func (m *OSConfigManager) DeletePatchDeployment(ctx context.Context, name string) error {
	client, err := osconfig.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create OS Config client: %w", err)
	}
	defer client.Close()

	req := &osconfigpb.DeletePatchDeploymentRequest{
		Name: name,
	}

	if err := client.DeletePatchDeployment(ctx, req); err != nil {
		return fmt.Errorf("failed to delete patch deployment: %w", err)
	}

	m.log.Info("patch deployment deleted", "name", name)
	return nil
}
