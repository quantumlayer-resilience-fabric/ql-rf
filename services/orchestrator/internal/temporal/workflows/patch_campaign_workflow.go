// Package workflows defines Temporal workflows for task execution.
package workflows

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// =============================================================================
// Workflow Input/Output Types
// =============================================================================

// PatchCampaignWorkflowInput contains the input for the patch campaign workflow.
type PatchCampaignWorkflowInput struct {
	CampaignID             string                 `json:"campaign_id"`
	OrgID                  string                 `json:"org_id"`
	UserID                 string                 `json:"user_id"`
	CampaignName           string                 `json:"campaign_name"`
	CampaignType           string                 `json:"campaign_type"` // cve_response, scheduled, emergency, compliance
	CVEAlertIDs            []string               `json:"cve_alert_ids,omitempty"`
	RolloutStrategy        string                 `json:"rollout_strategy"` // immediate, canary, rolling, blue_green
	CanaryPercentage       int                    `json:"canary_percentage"`
	WavePercentage         int                    `json:"wave_percentage"`
	FailureThreshold       int                    `json:"failure_threshold"` // % of failures before stop
	HealthCheckEnabled     bool                   `json:"health_check_enabled"`
	HealthCheckTimeout     time.Duration          `json:"health_check_timeout"`
	HealthCheckInterval    time.Duration          `json:"health_check_interval"`
	AutoRollbackEnabled    bool                   `json:"auto_rollback_enabled"`
	RollbackThreshold      int                    `json:"rollback_threshold"` // % of failures before rollback
	RequiresApproval       bool                   `json:"requires_approval"`
	NotifyOnStart          bool                   `json:"notify_on_start"`
	NotifyOnPhaseComplete  bool                   `json:"notify_on_phase_complete"`
	NotifyOnComplete       bool                   `json:"notify_on_complete"`
	NotifyOnFailure        bool                   `json:"notify_on_failure"`
	Phases                 []PatchPhaseInput      `json:"phases"`
	Metadata               map[string]interface{} `json:"metadata,omitempty"`
}

// PatchPhaseInput defines a phase within the campaign.
type PatchPhaseInput struct {
	PhaseID          string   `json:"phase_id"`
	Name             string   `json:"name"`
	PhaseType        string   `json:"phase_type"` // canary, wave, final
	TargetPercentage int      `json:"target_percentage"`
	AssetIDs         []string `json:"asset_ids"`
	Order            int      `json:"order"`
}

// PatchCampaignWorkflowResult contains the result of the patch campaign workflow.
type PatchCampaignWorkflowResult struct {
	CampaignID            string                 `json:"campaign_id"`
	Status                string                 `json:"status"`
	StartedAt             time.Time              `json:"started_at"`
	CompletedAt           time.Time              `json:"completed_at"`
	Duration              time.Duration          `json:"duration"`
	TotalAssets           int                    `json:"total_assets"`
	SuccessfulPatches     int                    `json:"successful_patches"`
	FailedPatches         int                    `json:"failed_patches"`
	SkippedPatches        int                    `json:"skipped_patches"`
	RolledBackAssets      int                    `json:"rolled_back_assets"`
	SuccessRate           float64                `json:"success_rate"`
	Phases                []PatchPhaseResult     `json:"phases"`
	Rollbacks             []PatchRollbackRecord  `json:"rollbacks,omitempty"`
	Evidence              map[string]interface{} `json:"evidence,omitempty"`
	Error                 string                 `json:"error,omitempty"`
}

// PatchPhaseResult represents the result of a single phase.
type PatchPhaseResult struct {
	PhaseID           string                 `json:"phase_id"`
	Name              string                 `json:"name"`
	PhaseType         string                 `json:"phase_type"`
	Status            string                 `json:"status"` // pending, in_progress, health_check, completed, failed, rolled_back
	StartedAt         time.Time              `json:"started_at,omitempty"`
	CompletedAt       time.Time              `json:"completed_at,omitempty"`
	Duration          time.Duration          `json:"duration,omitempty"`
	TotalAssets       int                    `json:"total_assets"`
	SuccessfulPatches int                    `json:"successful_patches"`
	FailedPatches     int                    `json:"failed_patches"`
	HealthCheckPassed bool                   `json:"health_check_passed"`
	HealthCheckResults []HealthCheckResultWF `json:"health_check_results,omitempty"`
	Error             string                 `json:"error,omitempty"`
}

// HealthCheckResultWF represents a health check result.
type HealthCheckResultWF struct {
	CheckName string `json:"check_name"`
	Passed    bool   `json:"passed"`
	Message   string `json:"message"`
	Duration  int    `json:"duration_ms"`
}

// PatchRollbackRecord records a rollback event.
type PatchRollbackRecord struct {
	Timestamp       time.Time `json:"timestamp"`
	TriggerType     string    `json:"trigger_type"` // automatic, manual, health_check, timeout
	Scope           string    `json:"scope"`        // asset, phase, campaign
	PhaseID         string    `json:"phase_id,omitempty"`
	AssetIDs        []string  `json:"asset_ids,omitempty"`
	Reason          string    `json:"reason"`
	Success         bool      `json:"success"`
	RolledBackCount int       `json:"rolled_back_count"`
}

// =============================================================================
// Activity Input/Output Types
// =============================================================================

// PatchPhaseExecuteInput is input for executing a patch phase.
type PatchPhaseExecuteInput struct {
	CampaignID string   `json:"campaign_id"`
	PhaseID    string   `json:"phase_id"`
	PhaseName  string   `json:"phase_name"`
	PhaseType  string   `json:"phase_type"`
	AssetIDs   []string `json:"asset_ids"`
}

// PatchPhaseExecuteOutput is output from executing a patch phase.
type PatchPhaseExecuteOutput struct {
	Success           bool                            `json:"success"`
	TotalAssets       int                             `json:"total_assets"`
	SuccessfulPatches int                             `json:"successful_patches"`
	FailedPatches     int                             `json:"failed_patches"`
	AssetResults      []PatchAssetResult              `json:"asset_results"`
	Error             string                          `json:"error,omitempty"`
}

// PatchAssetResult represents the result of patching a single asset.
type PatchAssetResult struct {
	AssetID       string `json:"asset_id"`
	Status        string `json:"status"` // completed, failed, skipped
	BeforeVersion string `json:"before_version,omitempty"`
	AfterVersion  string `json:"after_version,omitempty"`
	Executor      string `json:"executor"`
	ExecutionID   string `json:"execution_id,omitempty"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

// HealthCheckInput is input for running health checks.
type HealthCheckInput struct {
	CampaignID string   `json:"campaign_id"`
	PhaseID    string   `json:"phase_id"`
	AssetIDs   []string `json:"asset_ids"`
	Timeout    time.Duration `json:"timeout"`
}

// HealthCheckOutput is output from health checks.
type HealthCheckOutput struct {
	Passed  bool                  `json:"passed"`
	Results []HealthCheckResultWF `json:"results"`
	FailureRate float64           `json:"failure_rate"`
}

// RollbackInput is input for rollback operations.
type RollbackInput struct {
	CampaignID  string   `json:"campaign_id"`
	PhaseID     string   `json:"phase_id,omitempty"`
	TriggerType string   `json:"trigger_type"`
	Scope       string   `json:"scope"` // asset, phase, campaign
	AssetIDs    []string `json:"asset_ids"`
	Reason      string   `json:"reason"`
}

// RollbackOutput is output from rollback operations.
type RollbackOutput struct {
	Success         bool     `json:"success"`
	RolledBackCount int      `json:"rolled_back_count"`
	FailedCount     int      `json:"failed_count"`
	FailedAssetIDs  []string `json:"failed_asset_ids,omitempty"`
	Error           string   `json:"error,omitempty"`
}

// PatchCampaignNotification for notifications.
type PatchCampaignNotification struct {
	CampaignID   string `json:"campaign_id"`
	CampaignName string `json:"campaign_name"`
	OrgID        string `json:"org_id"`
	EventType    string `json:"event_type"` // started, phase_complete, completed, failed, rollback
	PhaseID      string `json:"phase_id,omitempty"`
	PhaseName    string `json:"phase_name,omitempty"`
	Status       string `json:"status,omitempty"`
	Message      string `json:"message,omitempty"`
	TotalAssets  int    `json:"total_assets,omitempty"`
	Successful   int    `json:"successful,omitempty"`
	Failed       int    `json:"failed,omitempty"`
}

// Signal names
const (
	SignalPatchCampaignApproval = "patch_campaign_approval"
	SignalPatchCampaignPause    = "patch_campaign_pause"
	SignalPatchCampaignResume   = "patch_campaign_resume"
	SignalPatchCampaignCancel   = "patch_campaign_cancel"
)

// PatchCampaignApprovalSignal is sent to approve/reject the campaign.
type PatchCampaignApprovalSignal struct {
	Action     string `json:"action"` // approve, reject
	ApprovedBy string `json:"approved_by"`
	Reason     string `json:"reason,omitempty"`
}

// =============================================================================
// Patch Campaign Workflow
// =============================================================================

// PatchCampaignWorkflow orchestrates a multi-phase patch rollout campaign.
// Phases:
// 1. Preflight: Validate campaign configuration and asset readiness
// 2. Canary (optional): Patch a small subset (e.g., 5%) and validate
// 3. Waves: Progressive rollout in configurable waves (e.g., 25% each)
// 4. Final: Complete remaining assets
// Each phase includes health checks and can trigger auto-rollback on failure.
func PatchCampaignWorkflow(ctx workflow.Context, input PatchCampaignWorkflowInput) (*PatchCampaignWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting patch campaign workflow",
		"campaign_id", input.CampaignID,
		"campaign_type", input.CampaignType,
		"rollout_strategy", input.RolloutStrategy,
		"phases", len(input.Phases),
	)

	result := &PatchCampaignWorkflowResult{
		CampaignID: input.CampaignID,
		Status:     "running",
		StartedAt:  workflow.Now(ctx),
		Phases:     make([]PatchPhaseResult, 0),
		Rollbacks:  make([]PatchRollbackRecord, 0),
		Evidence:   make(map[string]interface{}),
	}

	// Count total assets
	for _, phase := range input.Phases {
		result.TotalAssets += len(phase.AssetIDs)
	}

	// Configure activity options
	shortOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}

	longOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		HeartbeatTimeout:    2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 5,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute * 5,
			MaximumAttempts:    2,
		},
	}

	shortCtx := workflow.WithActivityOptions(ctx, shortOpts)
	longCtx := workflow.WithActivityOptions(ctx, longOpts)

	// Set up pause/cancel signal handlers
	paused := false
	cancelled := false

	pauseCh := workflow.GetSignalChannel(ctx, SignalPatchCampaignPause)
	resumeCh := workflow.GetSignalChannel(ctx, SignalPatchCampaignResume)
	cancelCh := workflow.GetSignalChannel(ctx, SignalPatchCampaignCancel)

	// Update campaign status to in_progress
	_ = workflow.ExecuteActivity(shortCtx, "UpdatePatchCampaignStatus", input.CampaignID, "in_progress").Get(shortCtx, nil)

	// Send start notification
	if input.NotifyOnStart {
		_ = workflow.ExecuteActivity(shortCtx, "NotifyPatchCampaignEvent", PatchCampaignNotification{
			CampaignID:   input.CampaignID,
			CampaignName: input.CampaignName,
			OrgID:        input.OrgID,
			EventType:    "started",
			TotalAssets:  result.TotalAssets,
		}).Get(shortCtx, nil)
	}

	// Wait for approval if required
	if input.RequiresApproval {
		logger.Info("Waiting for campaign approval")
		_ = workflow.ExecuteActivity(shortCtx, "UpdatePatchCampaignStatus", input.CampaignID, "pending_approval").Get(shortCtx, nil)

		approvalCh := workflow.GetSignalChannel(ctx, SignalPatchCampaignApproval)
		var approval PatchCampaignApprovalSignal

		approvalCh.Receive(ctx, &approval)

		if approval.Action != "approve" {
			result.Status = "rejected"
			result.Error = fmt.Sprintf("Campaign rejected by %s: %s", approval.ApprovedBy, approval.Reason)
			_ = workflow.ExecuteActivity(shortCtx, "UpdatePatchCampaignStatus", input.CampaignID, "cancelled").Get(shortCtx, nil)
			return finalizePatcHCampaignResult(ctx, result, input)
		}

		logger.Info("Campaign approved", "approved_by", approval.ApprovedBy)
		_ = workflow.ExecuteActivity(shortCtx, "UpdatePatchCampaignStatus", input.CampaignID, "in_progress").Get(shortCtx, nil)
	}

	// Execute phases
	for i, phase := range input.Phases {
		// Check for pause/cancel signals before each phase
		for {
			workflow.NewSelector(ctx).
				AddReceive(pauseCh, func(c workflow.ReceiveChannel, more bool) {
					var signal struct{}
					c.Receive(ctx, &signal)
					paused = true
					logger.Info("Campaign paused")
					_ = workflow.ExecuteActivity(shortCtx, "UpdatePatchCampaignStatus", input.CampaignID, "paused").Get(shortCtx, nil)
				}).
				AddReceive(resumeCh, func(c workflow.ReceiveChannel, more bool) {
					var signal struct{}
					c.Receive(ctx, &signal)
					paused = false
					logger.Info("Campaign resumed")
					_ = workflow.ExecuteActivity(shortCtx, "UpdatePatchCampaignStatus", input.CampaignID, "in_progress").Get(shortCtx, nil)
				}).
				AddReceive(cancelCh, func(c workflow.ReceiveChannel, more bool) {
					var signal struct{}
					c.Receive(ctx, &signal)
					cancelled = true
					logger.Info("Campaign cancelled")
				}).
				AddDefault(func() {})

			if !paused {
				break
			}
			// Wait while paused
			_ = workflow.Sleep(ctx, 10*time.Second)
		}

		if cancelled {
			result.Status = "cancelled"
			_ = workflow.ExecuteActivity(shortCtx, "UpdatePatchCampaignStatus", input.CampaignID, "cancelled").Get(shortCtx, nil)
			return finalizePatcHCampaignResult(ctx, result, input)
		}

		// Execute phase
		logger.Info("Executing phase",
			"phase_number", i+1,
			"phase_name", phase.Name,
			"phase_type", phase.PhaseType,
			"assets", len(phase.AssetIDs),
		)

		phaseResult := PatchPhaseResult{
			PhaseID:     phase.PhaseID,
			Name:        phase.Name,
			PhaseType:   phase.PhaseType,
			Status:      "in_progress",
			StartedAt:   workflow.Now(ctx),
			TotalAssets: len(phase.AssetIDs),
		}

		// Update phase status
		_ = workflow.ExecuteActivity(shortCtx, "UpdatePatchPhaseStatus", phase.PhaseID, "in_progress").Get(shortCtx, nil)

		// Execute phase patching
		var phaseOutput PatchPhaseExecuteOutput
		err := workflow.ExecuteActivity(longCtx, "ExecutePatchPhase", PatchPhaseExecuteInput{
			CampaignID: input.CampaignID,
			PhaseID:    phase.PhaseID,
			PhaseName:  phase.Name,
			PhaseType:  phase.PhaseType,
			AssetIDs:   phase.AssetIDs,
		}).Get(longCtx, &phaseOutput)

		phaseResult.SuccessfulPatches = phaseOutput.SuccessfulPatches
		phaseResult.FailedPatches = phaseOutput.FailedPatches
		result.SuccessfulPatches += phaseOutput.SuccessfulPatches
		result.FailedPatches += phaseOutput.FailedPatches

		if err != nil {
			phaseResult.Status = "failed"
			phaseResult.Error = err.Error()
			logger.Error("Phase execution failed", "phase", phase.Name, "error", err)

			// Check if we should rollback
			if input.AutoRollbackEnabled && shouldRollback(result, input.RollbackThreshold) {
				rollbackResult := executeRollback(ctx, shortCtx, input, result, phase.PhaseID, "health_check", "Phase execution failed")
				result.Rollbacks = append(result.Rollbacks, rollbackResult)
				result.RolledBackAssets += rollbackResult.RolledBackCount
			}

			result.Phases = append(result.Phases, phaseResult)
			result.Status = "failed"
			result.Error = fmt.Sprintf("Phase %s failed: %s", phase.Name, err.Error())

			// Notify failure
			if input.NotifyOnFailure {
				_ = workflow.ExecuteActivity(shortCtx, "NotifyPatchCampaignEvent", PatchCampaignNotification{
					CampaignID:   input.CampaignID,
					CampaignName: input.CampaignName,
					OrgID:        input.OrgID,
					EventType:    "failed",
					PhaseID:      phase.PhaseID,
					PhaseName:    phase.Name,
					Message:      err.Error(),
				}).Get(shortCtx, nil)
			}

			_ = workflow.ExecuteActivity(shortCtx, "UpdatePatchCampaignStatus", input.CampaignID, "failed").Get(shortCtx, nil)
			return finalizePatcHCampaignResult(ctx, result, input)
		}

		// Run health checks if enabled
		if input.HealthCheckEnabled {
			logger.Info("Running health checks", "phase", phase.Name)
			_ = workflow.ExecuteActivity(shortCtx, "UpdatePatchPhaseStatus", phase.PhaseID, "health_check").Get(shortCtx, nil)

			var healthOutput HealthCheckOutput
			err := workflow.ExecuteActivity(longCtx, "RunHealthChecks", HealthCheckInput{
				CampaignID: input.CampaignID,
				PhaseID:    phase.PhaseID,
				AssetIDs:   phase.AssetIDs,
				Timeout:    input.HealthCheckTimeout,
			}).Get(longCtx, &healthOutput)

			phaseResult.HealthCheckResults = healthOutput.Results
			phaseResult.HealthCheckPassed = healthOutput.Passed

			if err != nil || !healthOutput.Passed {
				logger.Warn("Health checks failed", "phase", phase.Name, "failure_rate", healthOutput.FailureRate)

				// Check if we should rollback
				if input.AutoRollbackEnabled {
					rollbackResult := executeRollback(ctx, shortCtx, input, result, phase.PhaseID, "health_check", "Health check failed")
					result.Rollbacks = append(result.Rollbacks, rollbackResult)
					result.RolledBackAssets += rollbackResult.RolledBackCount

					phaseResult.Status = "rolled_back"
					phaseResult.Error = "Health check failed, rolled back"
				} else {
					phaseResult.Status = "failed"
					phaseResult.Error = "Health check failed"
				}

				result.Phases = append(result.Phases, phaseResult)

				if healthOutput.FailureRate > float64(input.FailureThreshold)/100 {
					result.Status = "failed"
					result.Error = fmt.Sprintf("Phase %s health check failed with %.1f%% failure rate", phase.Name, healthOutput.FailureRate*100)
					_ = workflow.ExecuteActivity(shortCtx, "UpdatePatchCampaignStatus", input.CampaignID, "failed").Get(shortCtx, nil)
					return finalizePatcHCampaignResult(ctx, result, input)
				}

				continue
			}
		}

		// Phase completed successfully
		phaseResult.Status = "completed"
		phaseResult.CompletedAt = workflow.Now(ctx)
		phaseResult.Duration = phaseResult.CompletedAt.Sub(phaseResult.StartedAt)
		result.Phases = append(result.Phases, phaseResult)

		_ = workflow.ExecuteActivity(shortCtx, "UpdatePatchPhaseStatus", phase.PhaseID, "completed").Get(shortCtx, nil)

		// Notify phase complete
		if input.NotifyOnPhaseComplete {
			_ = workflow.ExecuteActivity(shortCtx, "NotifyPatchCampaignEvent", PatchCampaignNotification{
				CampaignID:   input.CampaignID,
				CampaignName: input.CampaignName,
				OrgID:        input.OrgID,
				EventType:    "phase_complete",
				PhaseID:      phase.PhaseID,
				PhaseName:    phase.Name,
				Status:       "completed",
				Successful:   phaseResult.SuccessfulPatches,
				Failed:       phaseResult.FailedPatches,
			}).Get(shortCtx, nil)
		}

		logger.Info("Phase completed",
			"phase", phase.Name,
			"successful", phaseResult.SuccessfulPatches,
			"failed", phaseResult.FailedPatches,
			"duration", phaseResult.Duration,
		)
	}

	// Campaign completed
	result.Status = "completed"
	if result.TotalAssets > 0 {
		result.SuccessRate = float64(result.SuccessfulPatches) / float64(result.TotalAssets) * 100
	}

	_ = workflow.ExecuteActivity(shortCtx, "UpdatePatchCampaignStatus", input.CampaignID, "completed").Get(shortCtx, nil)

	return finalizePatcHCampaignResult(ctx, result, input)
}

// shouldRollback determines if a rollback should be triggered based on failure threshold.
func shouldRollback(result *PatchCampaignWorkflowResult, threshold int) bool {
	if result.TotalAssets == 0 {
		return false
	}
	currentFailureRate := float64(result.FailedPatches) / float64(result.TotalAssets) * 100
	return currentFailureRate >= float64(threshold)
}

// executeRollback performs a rollback operation.
func executeRollback(ctx workflow.Context, shortCtx workflow.Context, input PatchCampaignWorkflowInput, result *PatchCampaignWorkflowResult, phaseID, triggerType, reason string) PatchRollbackRecord {
	logger := workflow.GetLogger(ctx)
	logger.Info("Triggering rollback", "trigger", triggerType, "reason", reason)

	record := PatchRollbackRecord{
		Timestamp:   workflow.Now(ctx),
		TriggerType: triggerType,
		Scope:       "phase",
		PhaseID:     phaseID,
		Reason:      reason,
	}

	// Collect asset IDs that need rollback
	var assetIDs []string
	for _, phase := range input.Phases {
		if phase.PhaseID == phaseID {
			assetIDs = phase.AssetIDs
			break
		}
	}
	record.AssetIDs = assetIDs

	var rollbackOutput RollbackOutput
	err := workflow.ExecuteActivity(shortCtx, "ExecuteRollback", RollbackInput{
		CampaignID:  input.CampaignID,
		PhaseID:     phaseID,
		TriggerType: triggerType,
		Scope:       "phase",
		AssetIDs:    assetIDs,
		Reason:      reason,
	}).Get(shortCtx, &rollbackOutput)

	if err != nil {
		record.Success = false
		logger.Error("Rollback failed", "error", err)
	} else {
		record.Success = rollbackOutput.Success
		record.RolledBackCount = rollbackOutput.RolledBackCount
	}

	// Notify about rollback
	_ = workflow.ExecuteActivity(shortCtx, "NotifyPatchCampaignEvent", PatchCampaignNotification{
		CampaignID:   input.CampaignID,
		CampaignName: input.CampaignName,
		OrgID:        input.OrgID,
		EventType:    "rollback",
		PhaseID:      phaseID,
		Message:      reason,
	}).Get(shortCtx, nil)

	return record
}

// finalizePatcHCampaignResult completes the campaign result and sends notifications.
func finalizePatcHCampaignResult(ctx workflow.Context, result *PatchCampaignWorkflowResult, input PatchCampaignWorkflowInput) (*PatchCampaignWorkflowResult, error) {
	result.CompletedAt = workflow.Now(ctx)
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	// Calculate final success rate
	if result.TotalAssets > 0 {
		result.SuccessRate = float64(result.SuccessfulPatches) / float64(result.TotalAssets) * 100
	}

	// Collect evidence
	result.Evidence["campaign_id"] = input.CampaignID
	result.Evidence["campaign_type"] = input.CampaignType
	result.Evidence["rollout_strategy"] = input.RolloutStrategy
	result.Evidence["total_assets"] = result.TotalAssets
	result.Evidence["successful_patches"] = result.SuccessfulPatches
	result.Evidence["failed_patches"] = result.FailedPatches
	result.Evidence["rolled_back_assets"] = result.RolledBackAssets
	result.Evidence["success_rate"] = result.SuccessRate
	result.Evidence["duration"] = result.Duration.String()
	result.Evidence["phases_count"] = len(result.Phases)
	result.Evidence["rollbacks_count"] = len(result.Rollbacks)

	shortOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	shortCtx := workflow.WithActivityOptions(ctx, shortOpts)

	// Store campaign result
	_ = workflow.ExecuteActivity(shortCtx, "StorePatchCampaignResult", result).Get(shortCtx, nil)

	// Record evidence for compliance
	_ = workflow.ExecuteActivity(shortCtx, "RecordVulnerabilityEvidence", map[string]interface{}{
		"campaign_id":  input.CampaignID,
		"org_id":       input.OrgID,
		"event_type":   "patch_campaign_completed",
		"evidence":     result.Evidence,
		"cve_alert_ids": input.CVEAlertIDs,
	}).Get(shortCtx, nil)

	// Send completion notification
	if input.NotifyOnComplete {
		_ = workflow.ExecuteActivity(shortCtx, "NotifyPatchCampaignEvent", PatchCampaignNotification{
			CampaignID:   input.CampaignID,
			CampaignName: input.CampaignName,
			OrgID:        input.OrgID,
			EventType:    "completed",
			Status:       result.Status,
			TotalAssets:  result.TotalAssets,
			Successful:   result.SuccessfulPatches,
			Failed:       result.FailedPatches,
			Message:      fmt.Sprintf("Campaign completed with %.1f%% success rate", result.SuccessRate),
		}).Get(shortCtx, nil)
	}

	logger := workflow.GetLogger(ctx)
	logger.Info("Patch campaign workflow completed",
		"campaign_id", input.CampaignID,
		"status", result.Status,
		"duration", result.Duration,
		"total_assets", result.TotalAssets,
		"successful", result.SuccessfulPatches,
		"failed", result.FailedPatches,
		"success_rate", result.SuccessRate,
	)

	return result, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// GeneratePatchCampaignPhases generates phases based on rollout strategy.
func GeneratePatchCampaignPhases(
	campaignID string,
	assetIDs []string,
	strategy string,
	canaryPct int,
	wavePct int,
) []PatchPhaseInput {
	phases := make([]PatchPhaseInput, 0)
	totalAssets := len(assetIDs)

	if totalAssets == 0 {
		return phases
	}

	switch strategy {
	case "immediate":
		// Single phase with all assets
		phases = append(phases, PatchPhaseInput{
			PhaseID:          uuid.New().String(),
			Name:             "All Assets",
			PhaseType:        "final",
			TargetPercentage: 100,
			AssetIDs:         assetIDs,
			Order:            1,
		})

	case "canary":
		// Canary phase + final phase
		canaryCount := max(1, totalAssets*canaryPct/100)

		phases = append(phases, PatchPhaseInput{
			PhaseID:          uuid.New().String(),
			Name:             "Canary",
			PhaseType:        "canary",
			TargetPercentage: canaryPct,
			AssetIDs:         assetIDs[:canaryCount],
			Order:            1,
		})

		if canaryCount < totalAssets {
			phases = append(phases, PatchPhaseInput{
				PhaseID:          uuid.New().String(),
				Name:             "Final",
				PhaseType:        "final",
				TargetPercentage: 100 - canaryPct,
				AssetIDs:         assetIDs[canaryCount:],
				Order:            2,
			})
		}

	case "rolling":
		// Multiple waves
		remaining := assetIDs
		waveNum := 1

		for len(remaining) > 0 {
			waveCount := max(1, len(remaining)*wavePct/100)
			if waveCount > len(remaining) {
				waveCount = len(remaining)
			}

			phaseType := "wave"
			phaseName := fmt.Sprintf("Wave %d", waveNum)
			if waveCount == len(remaining) {
				phaseType = "final"
				phaseName = "Final Wave"
			}

			phases = append(phases, PatchPhaseInput{
				PhaseID:          uuid.New().String(),
				Name:             phaseName,
				PhaseType:        phaseType,
				TargetPercentage: wavePct,
				AssetIDs:         remaining[:waveCount],
				Order:            waveNum,
			})

			remaining = remaining[waveCount:]
			waveNum++
		}

	case "blue_green":
		// Two equal phases
		halfCount := totalAssets / 2
		if halfCount == 0 {
			halfCount = 1
		}

		phases = append(phases, PatchPhaseInput{
			PhaseID:          uuid.New().String(),
			Name:             "Blue",
			PhaseType:        "wave",
			TargetPercentage: 50,
			AssetIDs:         assetIDs[:halfCount],
			Order:            1,
		})

		if halfCount < totalAssets {
			phases = append(phases, PatchPhaseInput{
				PhaseID:          uuid.New().String(),
				Name:             "Green",
				PhaseType:        "final",
				TargetPercentage: 50,
				AssetIDs:         assetIDs[halfCount:],
				Order:            2,
			})
		}

	default:
		// Default to single phase
		phases = append(phases, PatchPhaseInput{
			PhaseID:          uuid.New().String(),
			Name:             "All Assets",
			PhaseType:        "final",
			TargetPercentage: 100,
			AssetIDs:         assetIDs,
			Order:            1,
		})
	}

	return phases
}
