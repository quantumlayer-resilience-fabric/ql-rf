package temporal

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// TaskExecutionInput is the input for the task execution workflow.
type TaskExecutionInput struct {
	TaskID      string                 `json:"task_id"`
	PlanID      string                 `json:"plan_id"`
	OrgID       string                 `json:"org_id"`
	UserID      string                 `json:"user_id"`
	TaskType    string                 `json:"task_type"`
	Environment string                 `json:"environment"`
	Phases      []PhaseInput           `json:"phases"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PhaseInput defines a phase in the execution.
type PhaseInput struct {
	Name          string   `json:"name"`
	Assets        []string `json:"assets"`
	Action        string   `json:"action"`
	Parameters    string   `json:"parameters"` // JSON encoded
	WaitDuration  string   `json:"wait_duration,omitempty"`
	HealthCheck   bool     `json:"health_check"`
	RollbackOnFail bool    `json:"rollback_on_fail"`
}

// TaskExecutionResult is the result of the task execution workflow.
type TaskExecutionResult struct {
	TaskID        string         `json:"task_id"`
	Status        string         `json:"status"`
	PhaseResults  []PhaseResult  `json:"phase_results"`
	StartedAt     time.Time      `json:"started_at"`
	CompletedAt   time.Time      `json:"completed_at"`
	Error         string         `json:"error,omitempty"`
}

// PhaseResult tracks the result of a single phase.
type PhaseResult struct {
	Name         string        `json:"name"`
	Status       string        `json:"status"`
	AssetResults []AssetResult `json:"asset_results"`
	Duration     time.Duration `json:"duration"`
	Error        string        `json:"error,omitempty"`
}

// AssetResult tracks the result for a single asset.
type AssetResult struct {
	AssetID   string `json:"asset_id"`
	Status    string `json:"status"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
}

// TaskExecutionWorkflow orchestrates multi-phase task execution with rollback support.
func TaskExecutionWorkflow(ctx workflow.Context, input TaskExecutionInput) (*TaskExecutionResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting task execution workflow", "task_id", input.TaskID, "phases", len(input.Phases))

	result := &TaskExecutionResult{
		TaskID:       input.TaskID,
		Status:       "running",
		PhaseResults: make([]PhaseResult, 0, len(input.Phases)),
		StartedAt:    workflow.Now(ctx),
	}

	// Activity options with retries
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		HeartbeatTimeout:    1 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    1 * time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	// Execute phases sequentially
	for i, phase := range input.Phases {
		logger.Info("Executing phase", "phase", phase.Name, "index", i)

		phaseResult := PhaseResult{
			Name:   phase.Name,
			Status: "running",
		}
		phaseStart := workflow.Now(ctx)

		// Execute phase
		var phaseOutput PhaseExecutionOutput
		err := workflow.ExecuteActivity(ctx, "ExecutePhase", PhaseExecutionInput{
			TaskID:     input.TaskID,
			OrgID:      input.OrgID,
			Phase:      phase,
			PhaseIndex: i,
		}).Get(ctx, &phaseOutput)

		phaseResult.Duration = workflow.Now(ctx).Sub(phaseStart)
		phaseResult.AssetResults = phaseOutput.AssetResults

		if err != nil {
			phaseResult.Status = "failed"
			phaseResult.Error = err.Error()
			result.PhaseResults = append(result.PhaseResults, phaseResult)

			// Attempt rollback if configured
			if phase.RollbackOnFail {
				logger.Info("Phase failed, initiating rollback", "phase", phase.Name)
				
				rollbackErr := workflow.ExecuteActivity(ctx, "RollbackPhase", RollbackInput{
					TaskID:     input.TaskID,
					OrgID:      input.OrgID,
					PhaseIndex: i,
					Phases:     result.PhaseResults,
				}).Get(ctx, nil)

				if rollbackErr != nil {
					result.Status = "rollback_failed"
					result.Error = fmt.Sprintf("phase %s failed: %v, rollback failed: %v", phase.Name, err, rollbackErr)
				} else {
					result.Status = "rolled_back"
					result.Error = fmt.Sprintf("phase %s failed: %v, successfully rolled back", phase.Name, err)
				}
			} else {
				result.Status = "failed"
				result.Error = err.Error()
			}

			result.CompletedAt = workflow.Now(ctx)
			return result, nil
		}

		phaseResult.Status = "completed"
		result.PhaseResults = append(result.PhaseResults, phaseResult)

		// Health check if configured
		if phase.HealthCheck {
			logger.Info("Running health check", "phase", phase.Name)
			
			var healthResult HealthCheckOutput
			err := workflow.ExecuteActivity(ctx, "RunHealthCheck", HealthCheckInput{
				TaskID:     input.TaskID,
				OrgID:      input.OrgID,
				PhaseIndex: i,
				Assets:     phase.Assets,
			}).Get(ctx, &healthResult)

			if err != nil || !healthResult.Healthy {
				logger.Warn("Health check failed", "phase", phase.Name, "error", err)
				// Continue but log the issue
			}
		}

		// Wait if configured
		if phase.WaitDuration != "" {
			duration, err := time.ParseDuration(phase.WaitDuration)
			if err == nil && duration > 0 {
				logger.Info("Waiting between phases", "duration", duration)
				workflow.Sleep(ctx, duration)
			}
		}
	}

	result.Status = "completed"
	result.CompletedAt = workflow.Now(ctx)
	
	// Update task status
	_ = workflow.ExecuteActivity(ctx, "UpdateTaskStatus", TaskStatusUpdate{
		TaskID: input.TaskID,
		Status: "completed",
	}).Get(ctx, nil)

	logger.Info("Task execution workflow completed", "task_id", input.TaskID, "status", result.Status)
	return result, nil
}

// PatchDeploymentInput is the input for patch deployment workflow.
type PatchDeploymentInput struct {
	OrgID            string   `json:"org_id"`
	AssetIDs         []string `json:"asset_ids"`
	PatchType        string   `json:"patch_type"` // security, critical, all
	RebootOption     string   `json:"reboot_option"` // always, never, if_required
	MaintenanceWindow string  `json:"maintenance_window,omitempty"`
	MaxParallel      int      `json:"max_parallel"`
	DryRun           bool     `json:"dry_run"`
}

// PatchDeploymentResult is the result of patch deployment.
type PatchDeploymentResult struct {
	TotalAssets     int              `json:"total_assets"`
	PatchedAssets   int              `json:"patched_assets"`
	FailedAssets    int              `json:"failed_assets"`
	SkippedAssets   int              `json:"skipped_assets"`
	PatchResults    []PatchResult    `json:"patch_results"`
	StartedAt       time.Time        `json:"started_at"`
	CompletedAt     time.Time        `json:"completed_at"`
}

// PatchResult tracks patch status for a single asset.
type PatchResult struct {
	AssetID        string   `json:"asset_id"`
	Status         string   `json:"status"`
	PatchesApplied int      `json:"patches_applied"`
	RebootRequired bool     `json:"reboot_required"`
	RebootInitiated bool    `json:"reboot_initiated"`
	Error          string   `json:"error,omitempty"`
}

// PatchDeploymentWorkflow orchestrates patching across multiple assets.
func PatchDeploymentWorkflow(ctx workflow.Context, input PatchDeploymentInput) (*PatchDeploymentResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting patch deployment workflow", "org_id", input.OrgID, "assets", len(input.AssetIDs))

	result := &PatchDeploymentResult{
		TotalAssets:  len(input.AssetIDs),
		PatchResults: make([]PatchResult, 0, len(input.AssetIDs)),
		StartedAt:    workflow.Now(ctx),
	}

	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Hour,
		HeartbeatTimeout:    5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    5 * time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	// Pre-patch assessment
	var assessmentResults []PatchAssessmentResult
	err := workflow.ExecuteActivity(ctx, "AssessPatches", PatchAssessmentInput{
		OrgID:    input.OrgID,
		AssetIDs: input.AssetIDs,
	}).Get(ctx, &assessmentResults)

	if err != nil {
		return nil, fmt.Errorf("patch assessment failed: %w", err)
	}

	if input.DryRun {
		logger.Info("Dry run mode - skipping actual patching")
		result.CompletedAt = workflow.Now(ctx)
		return result, nil
	}

	// Process assets in batches
	maxParallel := input.MaxParallel
	if maxParallel <= 0 {
		maxParallel = 5
	}

	// Use a selector for parallel execution with limit
	selector := workflow.NewSelector(ctx)
	inFlight := 0
	assetIndex := 0

	for assetIndex < len(input.AssetIDs) || inFlight > 0 {
		// Start new activities up to maxParallel
		for inFlight < maxParallel && assetIndex < len(input.AssetIDs) {
			assetID := input.AssetIDs[assetIndex]
			future := workflow.ExecuteActivity(ctx, "PatchAsset", PatchAssetInput{
				OrgID:        input.OrgID,
				AssetID:      assetID,
				PatchType:    input.PatchType,
				RebootOption: input.RebootOption,
			})

			selector.AddFuture(future, func(f workflow.Future) {
				var patchResult PatchResult
				if err := f.Get(ctx, &patchResult); err != nil {
					patchResult = PatchResult{
						AssetID: assetID,
						Status:  "failed",
						Error:   err.Error(),
					}
					result.FailedAssets++
				} else {
					result.PatchedAssets++
				}
				result.PatchResults = append(result.PatchResults, patchResult)
				inFlight--
			})

			assetIndex++
			inFlight++
		}

		// Wait for at least one to complete
		if inFlight > 0 {
			selector.Select(ctx)
		}
	}

	result.CompletedAt = workflow.Now(ctx)
	logger.Info("Patch deployment completed", "total", result.TotalAssets, "patched", result.PatchedAssets, "failed", result.FailedAssets)

	return result, nil
}

// DRDrillInput is the input for DR drill workflow.
type DRDrillInput struct {
	DrillID          string            `json:"drill_id"`
	OrgID            string            `json:"org_id"`
	DrillType        string            `json:"drill_type"` // tabletop, functional, full
	TargetSites      []string          `json:"target_sites"`
	FailoverPairs    map[string]string `json:"failover_pairs"` // primary -> secondary
	RPOTargetMinutes int               `json:"rpo_target_minutes"`
	RTOTargetMinutes int               `json:"rto_target_minutes"`
	NotifyOnComplete bool              `json:"notify_on_complete"`
}

// DRDrillResult is the result of a DR drill.
type DRDrillResult struct {
	DrillID          string        `json:"drill_id"`
	Status           string        `json:"status"`
	RPOAchieved      int           `json:"rpo_achieved_minutes"`
	RTOAchieved      int           `json:"rto_achieved_minutes"`
	RPOMet           bool          `json:"rpo_met"`
	RTOMet           bool          `json:"rto_met"`
	FailoverResults  []FailoverResult `json:"failover_results"`
	Observations     []string      `json:"observations"`
	Recommendations  []string      `json:"recommendations"`
	StartedAt        time.Time     `json:"started_at"`
	CompletedAt      time.Time     `json:"completed_at"`
}

// FailoverResult tracks failover for a single pair.
type FailoverResult struct {
	PrimarySite    string    `json:"primary_site"`
	SecondarySite  string    `json:"secondary_site"`
	Status         string    `json:"status"`
	FailoverTime   time.Duration `json:"failover_time"`
	DataLossMinutes int       `json:"data_loss_minutes"`
	Error          string    `json:"error,omitempty"`
}

// DRDrillWorkflow orchestrates DR drill execution.
func DRDrillWorkflow(ctx workflow.Context, input DRDrillInput) (*DRDrillResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting DR drill workflow", "drill_id", input.DrillID, "type", input.DrillType)

	result := &DRDrillResult{
		DrillID:         input.DrillID,
		Status:          "running",
		FailoverResults: make([]FailoverResult, 0),
		Observations:    make([]string, 0),
		Recommendations: make([]string, 0),
		StartedAt:       workflow.Now(ctx),
	}

	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 4 * time.Hour,
		HeartbeatTimeout:    5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    10 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    10 * time.Minute,
			MaximumAttempts:    2, // DR operations are sensitive - limit retries
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	// Pre-drill validation
	var validation DrillValidationResult
	err := workflow.ExecuteActivity(ctx, "ValidateDRDrill", DrillValidationInput{
		DrillID:       input.DrillID,
		OrgID:         input.OrgID,
		FailoverPairs: input.FailoverPairs,
	}).Get(ctx, &validation)

	if err != nil || !validation.Valid {
		result.Status = "validation_failed"
		result.Observations = append(result.Observations, "Pre-drill validation failed")
		result.CompletedAt = workflow.Now(ctx)
		return result, fmt.Errorf("drill validation failed: %v", validation.Issues)
	}

	// Execute failovers based on drill type
	if input.DrillType == "tabletop" {
		// Tabletop drill - simulation only
		result.Observations = append(result.Observations, "Tabletop drill - no actual failover performed")
	} else {
		// Functional or full drill
		for primary, secondary := range input.FailoverPairs {
			logger.Info("Executing failover", "primary", primary, "secondary", secondary)
			
			failoverStart := workflow.Now(ctx)
			
			var failoverOutput FailoverOutput
			err := workflow.ExecuteActivity(ctx, "ExecuteFailover", FailoverInput{
				DrillID:       input.DrillID,
				PrimarySite:   primary,
				SecondarySite: secondary,
				DrillType:     input.DrillType,
			}).Get(ctx, &failoverOutput)

			failoverResult := FailoverResult{
				PrimarySite:   primary,
				SecondarySite: secondary,
				FailoverTime:  workflow.Now(ctx).Sub(failoverStart),
			}

			if err != nil {
				failoverResult.Status = "failed"
				failoverResult.Error = err.Error()
			} else {
				failoverResult.Status = "success"
				failoverResult.DataLossMinutes = failoverOutput.DataLossMinutes
			}

			result.FailoverResults = append(result.FailoverResults, failoverResult)
		}
	}

	// Calculate RTO/RPO
	totalFailoverMinutes := 0
	maxDataLoss := 0
	for _, fr := range result.FailoverResults {
		totalFailoverMinutes += int(fr.FailoverTime.Minutes())
		if fr.DataLossMinutes > maxDataLoss {
			maxDataLoss = fr.DataLossMinutes
		}
	}
	
	if len(result.FailoverResults) > 0 {
		result.RTOAchieved = totalFailoverMinutes / len(result.FailoverResults)
	}
	result.RPOAchieved = maxDataLoss
	result.RTOMet = result.RTOAchieved <= input.RTOTargetMinutes
	result.RPOMet = result.RPOAchieved <= input.RPOTargetMinutes

	// Generate recommendations
	if !result.RTOMet {
		result.Recommendations = append(result.Recommendations, 
			fmt.Sprintf("RTO target not met (%d min vs %d min target) - review failover automation", result.RTOAchieved, input.RTOTargetMinutes))
	}
	if !result.RPOMet {
		result.Recommendations = append(result.Recommendations, 
			fmt.Sprintf("RPO target not met (%d min vs %d min target) - increase replication frequency", result.RPOAchieved, input.RPOTargetMinutes))
	}

	// Post-drill cleanup for non-tabletop drills
	if input.DrillType != "tabletop" {
		_ = workflow.ExecuteActivity(ctx, "CleanupDRDrill", DrillCleanupInput{
			DrillID:       input.DrillID,
			FailoverPairs: input.FailoverPairs,
		}).Get(ctx, nil)
	}

	result.Status = "completed"
	result.CompletedAt = workflow.Now(ctx)

	// Notify if configured
	if input.NotifyOnComplete {
		_ = workflow.ExecuteActivity(ctx, "NotifyDrillComplete", DrillNotificationInput{
			DrillID: input.DrillID,
			OrgID:   input.OrgID,
			Result:  result,
		}).Get(ctx, nil)
	}

	logger.Info("DR drill completed", "drill_id", input.DrillID, "rpo_met", result.RPOMet, "rto_met", result.RTOMet)
	return result, nil
}

// ComplianceScanInput is the input for compliance scan workflow.
type ComplianceScanInput struct {
	OrgID       string   `json:"org_id"`
	Frameworks  []string `json:"frameworks"` // CIS, SOC2, NIST, etc.
	AssetIDs    []string `json:"asset_ids,omitempty"` // Empty = all assets
	GenerateReport bool  `json:"generate_report"`
}

// ComplianceScanResult is the result of a compliance scan.
type ComplianceScanResult struct {
	ScanID         string           `json:"scan_id"`
	OrgID          string           `json:"org_id"`
	OverallScore   float64          `json:"overall_score"`
	FrameworkResults []FrameworkResult `json:"framework_results"`
	CriticalFindings int             `json:"critical_findings"`
	HighFindings    int             `json:"high_findings"`
	MediumFindings  int             `json:"medium_findings"`
	LowFindings     int             `json:"low_findings"`
	ReportURL      string           `json:"report_url,omitempty"`
	StartedAt      time.Time        `json:"started_at"`
	CompletedAt    time.Time        `json:"completed_at"`
}

// FrameworkResult tracks compliance for a single framework.
type FrameworkResult struct {
	Framework      string    `json:"framework"`
	Score          float64   `json:"score"`
	PassedControls int       `json:"passed_controls"`
	FailedControls int       `json:"failed_controls"`
	TotalControls  int       `json:"total_controls"`
}

// ComplianceScanWorkflow orchestrates compliance scanning.
func ComplianceScanWorkflow(ctx workflow.Context, input ComplianceScanInput) (*ComplianceScanResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting compliance scan workflow", "org_id", input.OrgID, "frameworks", input.Frameworks)

	scanID := workflow.GetInfo(ctx).WorkflowExecution.ID

	result := &ComplianceScanResult{
		ScanID:          scanID,
		OrgID:           input.OrgID,
		FrameworkResults: make([]FrameworkResult, 0),
		StartedAt:       workflow.Now(ctx),
	}

	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		HeartbeatTimeout:    2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    2 * time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	// Scan each framework
	totalScore := 0.0
	for _, framework := range input.Frameworks {
		var scanOutput FrameworkScanOutput
		err := workflow.ExecuteActivity(ctx, "ScanFramework", FrameworkScanInput{
			OrgID:     input.OrgID,
			Framework: framework,
			AssetIDs:  input.AssetIDs,
		}).Get(ctx, &scanOutput)

		if err != nil {
			logger.Warn("Framework scan failed", "framework", framework, "error", err)
			continue
		}

		frameworkResult := FrameworkResult{
			Framework:      framework,
			Score:          scanOutput.Score,
			PassedControls: scanOutput.PassedControls,
			FailedControls: scanOutput.FailedControls,
			TotalControls:  scanOutput.PassedControls + scanOutput.FailedControls,
		}
		result.FrameworkResults = append(result.FrameworkResults, frameworkResult)
		totalScore += scanOutput.Score

		result.CriticalFindings += scanOutput.CriticalFindings
		result.HighFindings += scanOutput.HighFindings
		result.MediumFindings += scanOutput.MediumFindings
		result.LowFindings += scanOutput.LowFindings
	}

	if len(result.FrameworkResults) > 0 {
		result.OverallScore = totalScore / float64(len(result.FrameworkResults))
	}

	// Generate report if requested
	if input.GenerateReport {
		var reportOutput ReportGenerationOutput
		err := workflow.ExecuteActivity(ctx, "GenerateComplianceReport", ReportGenerationInput{
			ScanID: scanID,
			OrgID:  input.OrgID,
			Result: result,
		}).Get(ctx, &reportOutput)

		if err == nil {
			result.ReportURL = reportOutput.ReportURL
		}
	}

	result.CompletedAt = workflow.Now(ctx)
	logger.Info("Compliance scan completed", "org_id", input.OrgID, "score", result.OverallScore)

	return result, nil
}
