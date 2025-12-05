package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// =============================================================================
// Patch Campaigns
// =============================================================================

// PatchCampaign represents a multi-phase patch rollout campaign.
type PatchCampaign struct {
	ID                          uuid.UUID       `json:"id" db:"id"`
	OrgID                       uuid.UUID       `json:"orgId" db:"org_id"`
	Name                        string          `json:"name" db:"name"`
	Description                 *string         `json:"description,omitempty" db:"description"`
	CampaignType                string          `json:"campaignType" db:"campaign_type"`                               // cve_response, scheduled, emergency, compliance
	CVEAlertIDs                 pq.StringArray  `json:"cveAlertIds" db:"cve_alert_ids"`
	Status                      string          `json:"status" db:"status"`
	RequiresApproval            bool            `json:"requiresApproval" db:"requires_approval"`
	ApprovalRequestID           *uuid.UUID      `json:"approvalRequestId,omitempty" db:"approval_request_id"`
	ApprovedBy                  *string         `json:"approvedBy,omitempty" db:"approved_by"`
	ApprovedAt                  *time.Time      `json:"approvedAt,omitempty" db:"approved_at"`
	RejectionReason             *string         `json:"rejectionReason,omitempty" db:"rejection_reason"`
	ScheduledStartAt            *time.Time      `json:"scheduledStartAt,omitempty" db:"scheduled_start_at"`
	ScheduledEndAt              *time.Time      `json:"scheduledEndAt,omitempty" db:"scheduled_end_at"`
	MaintenanceWindowID         *uuid.UUID      `json:"maintenanceWindowId,omitempty" db:"maintenance_window_id"`
	TotalAssets                 int             `json:"totalAssets" db:"total_assets"`
	PendingAssets               int             `json:"pendingAssets" db:"pending_assets"`
	InProgressAssets            int             `json:"inProgressAssets" db:"in_progress_assets"`
	CompletedAssets             int             `json:"completedAssets" db:"completed_assets"`
	FailedAssets                int             `json:"failedAssets" db:"failed_assets"`
	SkippedAssets               int             `json:"skippedAssets" db:"skipped_assets"`
	RolloutStrategy             string          `json:"rolloutStrategy" db:"rollout_strategy"`
	CanaryPercentage            *int            `json:"canaryPercentage,omitempty" db:"canary_percentage"`
	WavePercentage              *int            `json:"wavePercentage,omitempty" db:"wave_percentage"`
	FailureThresholdPercentage  *int            `json:"failureThresholdPercentage,omitempty" db:"failure_threshold_percentage"`
	HealthCheckEnabled          bool            `json:"healthCheckEnabled" db:"health_check_enabled"`
	HealthCheckTimeoutSeconds   *int            `json:"healthCheckTimeoutSeconds,omitempty" db:"health_check_timeout_seconds"`
	HealthCheckIntervalSeconds  *int            `json:"healthCheckIntervalSeconds,omitempty" db:"health_check_interval_seconds"`
	AutoRollbackEnabled         bool            `json:"autoRollbackEnabled" db:"auto_rollback_enabled"`
	RollbackOnFailurePercentage *int            `json:"rollbackOnFailurePercentage,omitempty" db:"rollback_on_failure_percentage"`
	StartedAt                   *time.Time      `json:"startedAt,omitempty" db:"started_at"`
	CompletedAt                 *time.Time      `json:"completedAt,omitempty" db:"completed_at"`
	AITaskID                    *uuid.UUID      `json:"aiTaskId,omitempty" db:"ai_task_id"`
	CreatedBy                   string          `json:"createdBy" db:"created_by"`
	CreatedAt                   time.Time       `json:"createdAt" db:"created_at"`
	UpdatedAt                   time.Time       `json:"updatedAt" db:"updated_at"`

	// Enriched fields (not in DB)
	Phases []PatchCampaignPhase `json:"phases,omitempty" db:"-"`
}

// PatchCampaignType constants.
type PatchCampaignType string

const (
	PatchCampaignTypeCVEResponse PatchCampaignType = "cve_response"
	PatchCampaignTypeScheduled   PatchCampaignType = "scheduled"
	PatchCampaignTypeEmergency   PatchCampaignType = "emergency"
	PatchCampaignTypeCompliance  PatchCampaignType = "compliance"
)

// PatchCampaignStatus constants.
type PatchCampaignStatus string

const (
	PatchCampaignStatusDraft           PatchCampaignStatus = "draft"
	PatchCampaignStatusPendingApproval PatchCampaignStatus = "pending_approval"
	PatchCampaignStatusApproved        PatchCampaignStatus = "approved"
	PatchCampaignStatusScheduled       PatchCampaignStatus = "scheduled"
	PatchCampaignStatusInProgress      PatchCampaignStatus = "in_progress"
	PatchCampaignStatusPaused          PatchCampaignStatus = "paused"
	PatchCampaignStatusCompleted       PatchCampaignStatus = "completed"
	PatchCampaignStatusFailed          PatchCampaignStatus = "failed"
	PatchCampaignStatusRolledBack      PatchCampaignStatus = "rolled_back"
	PatchCampaignStatusCancelled       PatchCampaignStatus = "cancelled"
)

// RolloutStrategy constants.
type RolloutStrategy string

const (
	RolloutStrategyImmediate RolloutStrategy = "immediate"
	RolloutStrategyCanary    RolloutStrategy = "canary"
	RolloutStrategyBlueGreen RolloutStrategy = "blue_green"
	RolloutStrategyRolling   RolloutStrategy = "rolling"
)

// =============================================================================
// Patch Campaign Phases
// =============================================================================

// PatchCampaignPhase represents a phase within a campaign.
type PatchCampaignPhase struct {
	ID                   uuid.UUID        `json:"id" db:"id"`
	CampaignID           uuid.UUID        `json:"campaignId" db:"campaign_id"`
	PhaseNumber          int              `json:"phaseNumber" db:"phase_number"`
	Name                 string           `json:"name" db:"name"`
	PhaseType            string           `json:"phaseType" db:"phase_type"`                              // canary, wave, final
	TargetPercentage     int              `json:"targetPercentage" db:"target_percentage"`
	TargetAssetIDs       pq.StringArray   `json:"targetAssetIds" db:"target_asset_ids"`
	TargetCriteria       json.RawMessage  `json:"targetCriteria,omitempty" db:"target_criteria"`
	Status               string           `json:"status" db:"status"`
	TotalAssets          int              `json:"totalAssets" db:"total_assets"`
	CompletedAssets      int              `json:"completedAssets" db:"completed_assets"`
	FailedAssets         int              `json:"failedAssets" db:"failed_assets"`
	HealthCheckPassed    *bool            `json:"healthCheckPassed,omitempty" db:"health_check_passed"`
	HealthCheckResults   json.RawMessage  `json:"healthCheckResults,omitempty" db:"health_check_results"`
	EstimatedDurationMin *int             `json:"estimatedDurationMinutes,omitempty" db:"estimated_duration_minutes"`
	ActualDurationMin    *int             `json:"actualDurationMinutes,omitempty" db:"actual_duration_minutes"`
	StartedAt            *time.Time       `json:"startedAt,omitempty" db:"started_at"`
	CompletedAt          *time.Time       `json:"completedAt,omitempty" db:"completed_at"`
	CreatedAt            time.Time        `json:"createdAt" db:"created_at"`
	UpdatedAt            time.Time        `json:"updatedAt" db:"updated_at"`

	// Enriched fields
	Assets []PatchCampaignAsset `json:"assets,omitempty" db:"-"`
}

// PatchPhaseType constants.
type PatchPhaseType string

const (
	PatchPhaseTypeCanary PatchPhaseType = "canary"
	PatchPhaseTypeWave   PatchPhaseType = "wave"
	PatchPhaseTypeFinal  PatchPhaseType = "final"
)

// PatchPhaseStatus constants.
type PatchPhaseStatus string

const (
	PatchPhaseStatusPending         PatchPhaseStatus = "pending"
	PatchPhaseStatusInProgress      PatchPhaseStatus = "in_progress"
	PatchPhaseStatusHealthCheck     PatchPhaseStatus = "health_check"
	PatchPhaseStatusWaitingApproval PatchPhaseStatus = "waiting_approval"
	PatchPhaseStatusCompleted       PatchPhaseStatus = "completed"
	PatchPhaseStatusFailed          PatchPhaseStatus = "failed"
	PatchPhaseStatusRolledBack      PatchPhaseStatus = "rolled_back"
	PatchPhaseStatusSkipped         PatchPhaseStatus = "skipped"
)

// =============================================================================
// Patch Campaign Assets
// =============================================================================

// PatchCampaignAsset represents the patch status for a single asset.
type PatchCampaignAsset struct {
	ID                  uuid.UUID        `json:"id" db:"id"`
	CampaignID          uuid.UUID        `json:"campaignId" db:"campaign_id"`
	PhaseID             *uuid.UUID       `json:"phaseId,omitempty" db:"phase_id"`
	AssetID             uuid.UUID        `json:"assetId" db:"asset_id"`
	Status              string           `json:"status" db:"status"`
	Executor            *string          `json:"executor,omitempty" db:"executor"`                      // ssm, azure_update_mgr, gcp_os_config, k8s_rollout
	ExecutionID         *string          `json:"executionId,omitempty" db:"execution_id"`
	BeforeVersion       *string          `json:"beforeVersion,omitempty" db:"before_version"`
	AfterVersion        *string          `json:"afterVersion,omitempty" db:"after_version"`
	BeforePackages      json.RawMessage  `json:"beforePackages,omitempty" db:"before_packages"`
	AfterPackages       json.RawMessage  `json:"afterPackages,omitempty" db:"after_packages"`
	HealthCheckPassed   *bool            `json:"healthCheckPassed,omitempty" db:"health_check_passed"`
	HealthCheckResults  json.RawMessage  `json:"healthCheckResults,omitempty" db:"health_check_results"`
	HealthCheckAttempts int              `json:"healthCheckAttempts" db:"health_check_attempts"`
	ErrorMessage        *string          `json:"errorMessage,omitempty" db:"error_message"`
	ErrorCode           *string          `json:"errorCode,omitempty" db:"error_code"`
	RetryCount          int              `json:"retryCount" db:"retry_count"`
	MaxRetries          int              `json:"maxRetries" db:"max_retries"`
	RollbackAvailable   bool             `json:"rollbackAvailable" db:"rollback_available"`
	RollbackSnapshotID  *string          `json:"rollbackSnapshotId,omitempty" db:"rollback_snapshot_id"`
	RolledBackAt        *time.Time       `json:"rolledBackAt,omitempty" db:"rolled_back_at"`
	RollbackReason      *string          `json:"rollbackReason,omitempty" db:"rollback_reason"`
	QueuedAt            *time.Time       `json:"queuedAt,omitempty" db:"queued_at"`
	StartedAt           *time.Time       `json:"startedAt,omitempty" db:"started_at"`
	CompletedAt         *time.Time       `json:"completedAt,omitempty" db:"completed_at"`
	CreatedAt           time.Time        `json:"createdAt" db:"created_at"`
	UpdatedAt           time.Time        `json:"updatedAt" db:"updated_at"`

	// Enriched fields
	Asset *Asset `json:"asset,omitempty" db:"-"`
}

// PatchAssetStatus constants.
type PatchAssetStatus string

const (
	PatchAssetStatusPending     PatchAssetStatus = "pending"
	PatchAssetStatusQueued      PatchAssetStatus = "queued"
	PatchAssetStatusPreflight   PatchAssetStatus = "preflight"
	PatchAssetStatusInProgress  PatchAssetStatus = "in_progress"
	PatchAssetStatusHealthCheck PatchAssetStatus = "health_check"
	PatchAssetStatusCompleted   PatchAssetStatus = "completed"
	PatchAssetStatusFailed      PatchAssetStatus = "failed"
	PatchAssetStatusRolledBack  PatchAssetStatus = "rolled_back"
	PatchAssetStatusSkipped     PatchAssetStatus = "skipped"
)

// PatchExecutor constants.
type PatchExecutor string

const (
	PatchExecutorSSM           PatchExecutor = "ssm"
	PatchExecutorAzureUpdate   PatchExecutor = "azure_update_mgr"
	PatchExecutorGCPOSConfig   PatchExecutor = "gcp_os_config"
	PatchExecutorK8sRollout    PatchExecutor = "k8s_rollout"
	PatchExecutorVSphereUpdate PatchExecutor = "vsphere_update"
	PatchExecutorManual        PatchExecutor = "manual"
)

// =============================================================================
// Patch Rollbacks
// =============================================================================

// PatchRollback represents a rollback operation.
type PatchRollback struct {
	ID                  uuid.UUID        `json:"id" db:"id"`
	CampaignID          uuid.UUID        `json:"campaignId" db:"campaign_id"`
	PhaseID             *uuid.UUID       `json:"phaseId,omitempty" db:"phase_id"`
	TriggerType         string           `json:"triggerType" db:"trigger_type"`                       // automatic, manual, health_check, timeout
	TriggeredBy         *string          `json:"triggeredBy,omitempty" db:"triggered_by"`
	TriggerReason       string           `json:"triggerReason" db:"trigger_reason"`
	RollbackScope       string           `json:"rollbackScope" db:"rollback_scope"`                   // asset, phase, campaign
	AssetIDs            pq.StringArray   `json:"assetIds" db:"asset_ids"`
	Status              string           `json:"status" db:"status"`
	TotalAssets         int              `json:"totalAssets" db:"total_assets"`
	SuccessfulRollbacks int              `json:"successfulRollbacks" db:"successful_rollbacks"`
	FailedRollbacks     int              `json:"failedRollbacks" db:"failed_rollbacks"`
	RollbackResults     json.RawMessage  `json:"rollbackResults,omitempty" db:"rollback_results"`
	StartedAt           time.Time        `json:"startedAt" db:"started_at"`
	CompletedAt         *time.Time       `json:"completedAt,omitempty" db:"completed_at"`
	CreatedAt           time.Time        `json:"createdAt" db:"created_at"`
}

// RollbackTriggerType constants.
type RollbackTriggerType string

const (
	RollbackTriggerAutomatic   RollbackTriggerType = "automatic"
	RollbackTriggerManual      RollbackTriggerType = "manual"
	RollbackTriggerHealthCheck RollbackTriggerType = "health_check"
	RollbackTriggerTimeout     RollbackTriggerType = "timeout"
)

// RollbackScope constants.
type RollbackScope string

const (
	RollbackScopeAsset    RollbackScope = "asset"
	RollbackScopePhase    RollbackScope = "phase"
	RollbackScopeCampaign RollbackScope = "campaign"
)

// RollbackStatus constants.
type RollbackStatus string

const (
	RollbackStatusInProgress RollbackStatus = "in_progress"
	RollbackStatusCompleted  RollbackStatus = "completed"
	RollbackStatusFailed     RollbackStatus = "failed"
	RollbackStatusPartial    RollbackStatus = "partial"
)

// =============================================================================
// API Request/Response Types
// =============================================================================

// PatchCampaignFilter represents filters for listing campaigns.
type PatchCampaignFilter struct {
	Status       string     `json:"status,omitempty"`
	CampaignType string     `json:"campaignType,omitempty"`
	CreatedBy    string     `json:"createdBy,omitempty"`
	StartedAfter *time.Time `json:"startedAfter,omitempty"`
	StartedBefore *time.Time `json:"startedBefore,omitempty"`
}

// PatchCampaignListResponse represents a paginated list of campaigns.
type PatchCampaignListResponse struct {
	Campaigns  []PatchCampaign `json:"campaigns"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	TotalPages int             `json:"totalPages"`
}

// CreatePatchCampaignRequest represents a request to create a campaign.
type CreatePatchCampaignRequest struct {
	Name                       string       `json:"name" validate:"required,min=1,max=255"`
	Description                *string      `json:"description,omitempty"`
	CampaignType               string       `json:"campaignType" validate:"required,oneof=cve_response scheduled emergency compliance"`
	CVEAlertIDs                []uuid.UUID  `json:"cveAlertIds,omitempty"`
	RolloutStrategy            string       `json:"rolloutStrategy" validate:"required,oneof=immediate canary blue_green rolling"`
	CanaryPercentage           *int         `json:"canaryPercentage,omitempty"`
	WavePercentage             *int         `json:"wavePercentage,omitempty"`
	FailureThresholdPercentage *int         `json:"failureThresholdPercentage,omitempty"`
	HealthCheckEnabled         bool         `json:"healthCheckEnabled"`
	AutoRollbackEnabled        bool         `json:"autoRollbackEnabled"`
	RequiresApproval           bool         `json:"requiresApproval"`
	ScheduledStartAt           *time.Time   `json:"scheduledStartAt,omitempty"`
	TargetAssetIDs             []uuid.UUID  `json:"targetAssetIds,omitempty"`
	TargetCriteria             *TargetCriteria `json:"targetCriteria,omitempty"`
}

// TargetCriteria defines criteria for selecting assets.
type TargetCriteria struct {
	Platforms    []string `json:"platforms,omitempty"`
	Regions      []string `json:"regions,omitempty"`
	Environments []string `json:"environments,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
	ExcludeAssetIDs []uuid.UUID `json:"excludeAssetIds,omitempty"`
}

// UpdatePatchCampaignStatusRequest represents a request to update campaign status.
type UpdatePatchCampaignStatusRequest struct {
	Status          string  `json:"status" validate:"required,oneof=approved in_progress paused completed failed cancelled"`
	RejectionReason *string `json:"rejectionReason,omitempty"`
}

// TriggerRollbackRequest represents a request to trigger a rollback.
type TriggerRollbackRequest struct {
	Scope       string      `json:"scope" validate:"required,oneof=asset phase campaign"`
	Reason      string      `json:"reason" validate:"required,min=1,max=500"`
	AssetIDs    []uuid.UUID `json:"assetIds,omitempty"`
	PhaseID     *uuid.UUID  `json:"phaseId,omitempty"`
}

// PatchCampaignProgress represents campaign progress details.
type PatchCampaignProgress struct {
	CampaignID          uuid.UUID `json:"campaignId"`
	Status              string    `json:"status"`
	TotalAssets         int       `json:"totalAssets"`
	CompletedAssets     int       `json:"completedAssets"`
	FailedAssets        int       `json:"failedAssets"`
	SkippedAssets       int       `json:"skippedAssets"`
	CompletionPercentage float64  `json:"completionPercentage"`
	FailurePercentage    float64  `json:"failurePercentage"`
	TotalPhases         int       `json:"totalPhases"`
	CompletedPhases     int       `json:"completedPhases"`
	CurrentPhase        *string   `json:"currentPhase,omitempty"`
	EstimatedCompletion *time.Time `json:"estimatedCompletion,omitempty"`
}

// PatchCampaignSummary represents aggregated campaign statistics.
type PatchCampaignSummary struct {
	TotalCampaigns    int `json:"totalCampaigns"`
	ActiveCampaigns   int `json:"activeCampaigns"`
	CompletedCampaigns int `json:"completedCampaigns"`
	FailedCampaigns   int `json:"failedCampaigns"`
	TotalAssetsPatched int `json:"totalAssetsPatched"`
	TotalRollbacks    int `json:"totalRollbacks"`
	SuccessRate       float64 `json:"successRate"`
}

// =============================================================================
// Health Check Types
// =============================================================================

// HealthCheckConfig represents health check configuration.
type HealthCheckConfig struct {
	Enabled         bool     `json:"enabled"`
	TimeoutSeconds  int      `json:"timeoutSeconds"`
	IntervalSeconds int      `json:"intervalSeconds"`
	RetryCount      int      `json:"retryCount"`
	Checks          []HealthCheck `json:"checks"`
}

// HealthCheck represents a single health check.
type HealthCheck struct {
	Type     string            `json:"type"`     // http, tcp, command, ssm_status, azure_status, k8s_ready
	Name     string            `json:"name"`
	Target   string            `json:"target"`   // URL, port, command, etc.
	Expected string            `json:"expected"` // Expected result
	Metadata map[string]string `json:"metadata,omitempty"`
}

// HealthCheckResult represents the result of a health check.
type HealthCheckResult struct {
	CheckName   string    `json:"checkName"`
	CheckType   string    `json:"checkType"`
	Passed      bool      `json:"passed"`
	Message     string    `json:"message"`
	Duration    int       `json:"durationMs"`
	Timestamp   time.Time `json:"timestamp"`
	Details     map[string]interface{} `json:"details,omitempty"`
}
