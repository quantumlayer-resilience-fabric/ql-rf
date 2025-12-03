// Package workflows defines Temporal workflows for task execution.
package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// DRDrillWorkflowInput contains the input for the DR drill workflow.
type DRDrillWorkflowInput struct {
	DrillID       string                 `json:"drill_id"`
	OrgID         string                 `json:"org_id"`
	UserID        string                 `json:"user_id"`
	DrillType     string                 `json:"drill_type"` // full, partial, tabletop
	DrPairIDs     []string               `json:"dr_pair_ids"`
	Environment   string                 `json:"environment"`
	Runbook       map[string]interface{} `json:"runbook,omitempty"`
	TargetRTO     time.Duration          `json:"target_rto"`
	TargetRPO     time.Duration          `json:"target_rpo"`
	NotifyOnStart bool                   `json:"notify_on_start"`
	NotifyOnEnd   bool                   `json:"notify_on_end"`
}

// DRDrillWorkflowResult contains the result of the DR drill workflow.
type DRDrillWorkflowResult struct {
	DrillID       string                   `json:"drill_id"`
	Status        string                   `json:"status"`
	StartedAt     time.Time                `json:"started_at"`
	CompletedAt   time.Time                `json:"completed_at"`
	Duration      time.Duration            `json:"duration"`
	PairsTestedOK int                      `json:"pairs_tested_ok"`
	PairsFailed   int                      `json:"pairs_failed"`
	ActualRTO     time.Duration            `json:"actual_rto"`
	ActualRPO     time.Duration            `json:"actual_rpo"`
	RTOAchieved   bool                     `json:"rto_achieved"`
	RPOAchieved   bool                     `json:"rpo_achieved"`
	Phases        []DRPhaseResult          `json:"phases"`
	FailureLog    []DRFailureEntry         `json:"failure_log,omitempty"`
	Metrics       map[string]interface{}   `json:"metrics"`
	Error         string                   `json:"error,omitempty"`
}

// DRPhaseResult represents the result of a single DR drill phase.
type DRPhaseResult struct {
	Name        string                 `json:"name"`
	Status      string                 `json:"status"` // pending, running, completed, failed, skipped
	StartedAt   time.Time              `json:"started_at,omitempty"`
	CompletedAt time.Time              `json:"completed_at,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// DRFailureEntry represents a failure during the DR drill.
type DRFailureEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Phase     string    `json:"phase"`
	PairID    string    `json:"pair_id"`
	Error     string    `json:"error"`
	Severity  string    `json:"severity"` // critical, major, minor
}

// DR Drill phases
const (
	DRPhasePreCheck      = "pre_check"
	DRPhaseReplication   = "replication_sync"
	DRPhaseFailover      = "failover"
	DRPhaseValidation    = "validation"
	DRPhaseFailback      = "failback"
	DRPhasePostCheck     = "post_check"
	DRPhaseReport        = "report"
)

// DRDrillWorkflow orchestrates a full disaster recovery drill.
// Phases:
// 1. Pre-check: Verify DR pair health and replication status
// 2. Replication Sync: Ensure all data is synced
// 3. Failover: Execute failover to DR site
// 4. Validation: Validate services are running on DR site
// 5. Failback: Restore to primary site
// 6. Post-check: Verify everything is back to normal
// 7. Report: Generate DR drill report
func DRDrillWorkflow(ctx workflow.Context, input DRDrillWorkflowInput) (*DRDrillWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting DR drill workflow",
		"drill_id", input.DrillID,
		"drill_type", input.DrillType,
		"dr_pairs", len(input.DrPairIDs),
	)

	result := &DRDrillWorkflowResult{
		DrillID:   input.DrillID,
		Status:    "running",
		StartedAt: workflow.Now(ctx),
		Phases:    make([]DRPhaseResult, 0),
		Metrics:   make(map[string]interface{}),
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
		HeartbeatTimeout:    time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 5,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute * 5,
			MaximumAttempts:    2,
		},
	}

	shortCtx := workflow.WithActivityOptions(ctx, shortOpts)
	longCtx := workflow.WithActivityOptions(ctx, longOpts)

	// Send start notification
	if input.NotifyOnStart {
		_ = workflow.ExecuteActivity(shortCtx, "NotifyDRDrillStarted", DRDrillNotification{
			DrillID:   input.DrillID,
			OrgID:     input.OrgID,
			DrillType: input.DrillType,
			PairCount: len(input.DrPairIDs),
		}).Get(shortCtx, nil)
	}

	// Phase 1: Pre-check
	logger.Info("Starting pre-check phase")
	preCheckResult, err := executeDRPhase(shortCtx, DRPhaseInput{
		DrillID:   input.DrillID,
		PhaseName: DRPhasePreCheck,
		DrPairIDs: input.DrPairIDs,
		Action:    "verify_health",
	})
	result.Phases = append(result.Phases, *preCheckResult)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("Pre-check failed: %s", err.Error())
		result.FailureLog = append(result.FailureLog, DRFailureEntry{
			Timestamp: workflow.Now(ctx),
			Phase:     DRPhasePreCheck,
			Error:     err.Error(),
			Severity:  "critical",
		})
		return finalizeResult(ctx, result, input)
	}

	// Phase 2: Replication Sync
	logger.Info("Starting replication sync phase")
	syncResult, err := executeDRPhase(longCtx, DRPhaseInput{
		DrillID:   input.DrillID,
		PhaseName: DRPhaseReplication,
		DrPairIDs: input.DrPairIDs,
		Action:    "sync_replication",
	})
	result.Phases = append(result.Phases, *syncResult)
	if err != nil {
		logger.Warn("Replication sync had issues, continuing with drill", "error", err)
		result.FailureLog = append(result.FailureLog, DRFailureEntry{
			Timestamp: workflow.Now(ctx),
			Phase:     DRPhaseReplication,
			Error:     err.Error(),
			Severity:  "major",
		})
	}

	// Phase 3: Failover
	logger.Info("Starting failover phase")
	failoverStart := workflow.Now(ctx)
	failoverResult, err := executeDRPhase(longCtx, DRPhaseInput{
		DrillID:   input.DrillID,
		PhaseName: DRPhaseFailover,
		DrPairIDs: input.DrPairIDs,
		Action:    "execute_failover",
	})
	result.Phases = append(result.Phases, *failoverResult)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("Failover failed: %s", err.Error())
		result.FailureLog = append(result.FailureLog, DRFailureEntry{
			Timestamp: workflow.Now(ctx),
			Phase:     DRPhaseFailover,
			Error:     err.Error(),
			Severity:  "critical",
		})
		// Still try to failback even if failover had issues
	}

	// Phase 4: Validation
	logger.Info("Starting validation phase")
	validationResult, err := executeDRPhase(shortCtx, DRPhaseInput{
		DrillID:   input.DrillID,
		PhaseName: DRPhaseValidation,
		DrPairIDs: input.DrPairIDs,
		Action:    "validate_services",
	})
	result.Phases = append(result.Phases, *validationResult)
	failoverEnd := workflow.Now(ctx)

	// Calculate actual RTO (time from failover start to validation complete)
	result.ActualRTO = failoverEnd.Sub(failoverStart)
	result.RTOAchieved = result.ActualRTO <= input.TargetRTO

	if err != nil {
		result.FailureLog = append(result.FailureLog, DRFailureEntry{
			Timestamp: workflow.Now(ctx),
			Phase:     DRPhaseValidation,
			Error:     err.Error(),
			Severity:  "major",
		})
	}

	// Count successful pairs from validation
	if validationResult.Details != nil {
		if pairsOK, ok := validationResult.Details["pairs_ok"].(int); ok {
			result.PairsTestedOK = pairsOK
		}
		if pairsFailed, ok := validationResult.Details["pairs_failed"].(int); ok {
			result.PairsFailed = pairsFailed
		}
	}

	// Phase 5: Failback (unless drill type is "tabletop")
	if input.DrillType != "tabletop" {
		logger.Info("Starting failback phase")
		failbackResult, err := executeDRPhase(longCtx, DRPhaseInput{
			DrillID:   input.DrillID,
			PhaseName: DRPhaseFailback,
			DrPairIDs: input.DrPairIDs,
			Action:    "execute_failback",
		})
		result.Phases = append(result.Phases, *failbackResult)
		if err != nil {
			result.FailureLog = append(result.FailureLog, DRFailureEntry{
				Timestamp: workflow.Now(ctx),
				Phase:     DRPhaseFailback,
				Error:     err.Error(),
				Severity:  "major",
			})
		}
	}

	// Phase 6: Post-check
	logger.Info("Starting post-check phase")
	postCheckResult, err := executeDRPhase(shortCtx, DRPhaseInput{
		DrillID:   input.DrillID,
		PhaseName: DRPhasePostCheck,
		DrPairIDs: input.DrPairIDs,
		Action:    "verify_restored",
	})
	result.Phases = append(result.Phases, *postCheckResult)
	if err != nil {
		result.FailureLog = append(result.FailureLog, DRFailureEntry{
			Timestamp: workflow.Now(ctx),
			Phase:     DRPhasePostCheck,
			Error:     err.Error(),
			Severity:  "minor",
		})
	}

	// Calculate RPO from replication metrics
	if syncResult.Details != nil {
		if lagDuration, ok := syncResult.Details["max_lag"].(string); ok {
			if lag, err := time.ParseDuration(lagDuration); err == nil {
				result.ActualRPO = lag
				result.RPOAchieved = result.ActualRPO <= input.TargetRPO
			}
		}
	}

	// Determine final status
	if result.Error == "" {
		if len(result.FailureLog) > 0 {
			result.Status = "completed_with_warnings"
		} else {
			result.Status = "completed"
		}
	}

	return finalizeResult(ctx, result, input)
}

// DRPhaseInput is input for a DR drill phase.
type DRPhaseInput struct {
	DrillID   string   `json:"drill_id"`
	PhaseName string   `json:"phase_name"`
	DrPairIDs []string `json:"dr_pair_ids"`
	Action    string   `json:"action"`
}

// executeDRPhase executes a single phase of the DR drill.
func executeDRPhase(ctx workflow.Context, input DRPhaseInput) (*DRPhaseResult, error) {
	result := &DRPhaseResult{
		Name:      input.PhaseName,
		Status:    "running",
		StartedAt: workflow.Now(ctx),
	}

	var phaseOutput DRPhaseOutput
	err := workflow.ExecuteActivity(ctx, "ExecuteDRPhase", input).Get(ctx, &phaseOutput)

	result.CompletedAt = workflow.Now(ctx)
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
	result.Details = phaseOutput.Details

	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, err
	}

	result.Status = "completed"
	return result, nil
}

// DRPhaseOutput is the output from a DR phase activity.
type DRPhaseOutput struct {
	Success bool                   `json:"success"`
	Details map[string]interface{} `json:"details"`
	Error   string                 `json:"error,omitempty"`
}

// DRDrillNotification for start/end notifications.
type DRDrillNotification struct {
	DrillID   string `json:"drill_id"`
	OrgID     string `json:"org_id"`
	DrillType string `json:"drill_type"`
	PairCount int    `json:"pair_count"`
	Status    string `json:"status,omitempty"`
	Duration  string `json:"duration,omitempty"`
}

// finalizeResult completes the drill result and sends notifications.
func finalizeResult(ctx workflow.Context, result *DRDrillWorkflowResult, input DRDrillWorkflowInput) (*DRDrillWorkflowResult, error) {
	result.CompletedAt = workflow.Now(ctx)
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	// Collect metrics
	result.Metrics["total_pairs"] = len(input.DrPairIDs)
	result.Metrics["pairs_tested_ok"] = result.PairsTestedOK
	result.Metrics["pairs_failed"] = result.PairsFailed
	result.Metrics["success_rate"] = float64(result.PairsTestedOK) / float64(len(input.DrPairIDs)) * 100
	result.Metrics["rto_target"] = input.TargetRTO.String()
	result.Metrics["rto_actual"] = result.ActualRTO.String()
	result.Metrics["rto_achieved"] = result.RTOAchieved
	result.Metrics["rpo_target"] = input.TargetRPO.String()
	result.Metrics["rpo_actual"] = result.ActualRPO.String()
	result.Metrics["rpo_achieved"] = result.RPOAchieved

	shortOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	shortCtx := workflow.WithActivityOptions(ctx, shortOpts)

	// Store drill result
	_ = workflow.ExecuteActivity(shortCtx, "StoreDRDrillResult", result).Get(shortCtx, nil)

	// Send end notification
	if input.NotifyOnEnd {
		_ = workflow.ExecuteActivity(shortCtx, "NotifyDRDrillCompleted", DRDrillNotification{
			DrillID:   input.DrillID,
			OrgID:     input.OrgID,
			DrillType: input.DrillType,
			PairCount: len(input.DrPairIDs),
			Status:    result.Status,
			Duration:  result.Duration.String(),
		}).Get(shortCtx, nil)
	}

	logger := workflow.GetLogger(ctx)
	logger.Info("DR drill workflow completed",
		"drill_id", input.DrillID,
		"status", result.Status,
		"duration", result.Duration,
		"pairs_ok", result.PairsTestedOK,
		"pairs_failed", result.PairsFailed,
		"rto_achieved", result.RTOAchieved,
		"rpo_achieved", result.RPOAchieved,
	)

	return result, nil
}
