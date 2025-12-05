// Package workflows defines Temporal workflows for InSpec execution.
package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// InSpecWorkflowInput contains the input for the InSpec execution workflow.
type InSpecWorkflowInput struct {
	RunID     string `json:"run_id"`
	ProfileID string `json:"profile_id"`
	AssetID   string `json:"asset_id"`
	OrgID     string `json:"org_id"`
	ProfileURL string `json:"profile_url"`
	AssetType string `json:"asset_type"` // vm, container, cloud_account
	Platform  string `json:"platform"`  // linux, windows, aws, azure, gcp
}

// InSpecWorkflowResult contains the result of the InSpec execution workflow.
type InSpecWorkflowResult struct {
	RunID       string        `json:"run_id"`
	Status      string        `json:"status"`
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt time.Time     `json:"completed_at"`
	Duration    time.Duration `json:"duration"`
	TotalTests  int           `json:"total_tests"`
	PassedTests int           `json:"passed_tests"`
	FailedTests int           `json:"failed_tests"`
	SkippedTests int          `json:"skipped_tests"`
	Error       string        `json:"error,omitempty"`
}

const (
	// InSpec workflow statuses
	InSpecStatusPending   = "pending"
	InSpecStatusRunning   = "running"
	InSpecStatusCompleted = "completed"
	InSpecStatusFailed    = "failed"
	InSpecStatusCancelled = "cancelled"
)

// InSpecExecutionWorkflow orchestrates the execution of an InSpec profile:
// 1. Prepare execution environment
// 2. Execute InSpec profile
// 3. Parse and store results
// 4. Map results to compliance controls
// 5. Update assessment results
func InSpecExecutionWorkflow(ctx workflow.Context, input InSpecWorkflowInput) (*InSpecWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting InSpec execution workflow",
		"run_id", input.RunID,
		"profile_id", input.ProfileID,
		"asset_id", input.AssetID,
		"platform", input.Platform,
	)

	result := &InSpecWorkflowResult{
		RunID:     input.RunID,
		Status:    InSpecStatusRunning,
		StartedAt: workflow.Now(ctx),
	}

	// Configure activity options
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute, // InSpec runs can take time
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 2,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute * 2,
			MaximumAttempts:    3,
			NonRetryableErrorTypes: []string{
				"InvalidProfileError",
				"AssetNotReachableError",
			},
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	// Step 1: Update run status to running
	if err := workflow.ExecuteActivity(ctx, "UpdateInSpecRunStatus", input.RunID, InSpecStatusRunning, "").Get(ctx, nil); err != nil {
		logger.Warn("Failed to update run status", "error", err)
		// Continue anyway
	}

	// Step 2: Prepare execution environment
	var prepareResult struct {
		TempDir     string `json:"temp_dir"`
		ProfilePath string `json:"profile_path"`
		Ready       bool   `json:"ready"`
	}
	if err := workflow.ExecuteActivity(ctx, "PrepareInSpecExecution", input).Get(ctx, &prepareResult); err != nil {
		logger.Error("Failed to prepare execution environment", "error", err)
		result.Status = InSpecStatusFailed
		result.Error = err.Error()
		result.CompletedAt = workflow.Now(ctx)
		result.Duration = result.CompletedAt.Sub(result.StartedAt)

		// Update run status to failed
		_ = workflow.ExecuteActivity(ctx, "UpdateInSpecRunStatus", input.RunID, InSpecStatusFailed, err.Error()).Get(ctx, nil)
		return result, err
	}

	if !prepareResult.Ready {
		err := temporal.NewApplicationError("execution environment not ready", "PreparationError")
		result.Status = InSpecStatusFailed
		result.Error = "execution environment not ready"
		result.CompletedAt = workflow.Now(ctx)
		result.Duration = result.CompletedAt.Sub(result.StartedAt)

		_ = workflow.ExecuteActivity(ctx, "UpdateInSpecRunStatus", input.RunID, InSpecStatusFailed, result.Error).Get(ctx, nil)
		return result, err
	}

	// Step 3: Execute InSpec profile
	var execResult struct {
		Success     bool   `json:"success"`
		OutputJSON  string `json:"output_json"`
		TotalTests  int    `json:"total_tests"`
		PassedTests int    `json:"passed_tests"`
		FailedTests int    `json:"failed_tests"`
		SkippedTests int   `json:"skipped_tests"`
		Duration    int    `json:"duration"` // seconds
		Error       string `json:"error,omitempty"`
	}

	execInput := struct {
		RunID       string `json:"run_id"`
		AssetID     string `json:"asset_id"`
		ProfilePath string `json:"profile_path"`
		Platform    string `json:"platform"`
		AssetType   string `json:"asset_type"`
	}{
		RunID:       input.RunID,
		AssetID:     input.AssetID,
		ProfilePath: prepareResult.ProfilePath,
		Platform:    input.Platform,
		AssetType:   input.AssetType,
	}

	if err := workflow.ExecuteActivity(ctx, "ExecuteInSpecProfile", execInput).Get(ctx, &execResult); err != nil {
		logger.Error("Failed to execute InSpec profile", "error", err)
		result.Status = InSpecStatusFailed
		result.Error = err.Error()
		result.CompletedAt = workflow.Now(ctx)
		result.Duration = result.CompletedAt.Sub(result.StartedAt)

		_ = workflow.ExecuteActivity(ctx, "UpdateInSpecRunStatus", input.RunID, InSpecStatusFailed, err.Error()).Get(ctx, nil)
		_ = workflow.ExecuteActivity(ctx, "CleanupInSpecEnvironment", prepareResult.TempDir).Get(ctx, nil)
		return result, err
	}

	if !execResult.Success {
		result.Status = InSpecStatusFailed
		result.Error = execResult.Error
		result.CompletedAt = workflow.Now(ctx)
		result.Duration = result.CompletedAt.Sub(result.StartedAt)

		_ = workflow.ExecuteActivity(ctx, "UpdateInSpecRunStatus", input.RunID, InSpecStatusFailed, execResult.Error).Get(ctx, nil)
		_ = workflow.ExecuteActivity(ctx, "CleanupInSpecEnvironment", prepareResult.TempDir).Get(ctx, nil)
		return result, temporal.NewApplicationError("InSpec execution failed", "ExecutionError")
	}

	// Update result with execution stats
	result.TotalTests = execResult.TotalTests
	result.PassedTests = execResult.PassedTests
	result.FailedTests = execResult.FailedTests
	result.SkippedTests = execResult.SkippedTests

	// Step 4: Parse and store results
	parseInput := struct {
		RunID      string `json:"run_id"`
		OutputJSON string `json:"output_json"`
	}{
		RunID:      input.RunID,
		OutputJSON: execResult.OutputJSON,
	}

	if err := workflow.ExecuteActivity(ctx, "ParseInSpecResults", parseInput).Get(ctx, nil); err != nil {
		logger.Error("Failed to parse InSpec results", "error", err)
		result.Status = InSpecStatusFailed
		result.Error = err.Error()
		result.CompletedAt = workflow.Now(ctx)
		result.Duration = result.CompletedAt.Sub(result.StartedAt)

		_ = workflow.ExecuteActivity(ctx, "UpdateInSpecRunStatus", input.RunID, InSpecStatusFailed, err.Error()).Get(ctx, nil)
		_ = workflow.ExecuteActivity(ctx, "CleanupInSpecEnvironment", prepareResult.TempDir).Get(ctx, nil)
		return result, err
	}

	// Step 5: Map results to compliance controls
	mapInput := struct {
		RunID     string `json:"run_id"`
		ProfileID string `json:"profile_id"`
	}{
		RunID:     input.RunID,
		ProfileID: input.ProfileID,
	}

	if err := workflow.ExecuteActivity(ctx, "MapInSpecToComplianceControls", mapInput).Get(ctx, nil); err != nil {
		logger.Warn("Failed to map InSpec results to compliance controls", "error", err)
		// This is non-fatal, continue
	}

	// Step 6: Update compliance assessment results
	assessInput := struct {
		RunID string `json:"run_id"`
		OrgID string `json:"org_id"`
	}{
		RunID: input.RunID,
		OrgID: input.OrgID,
	}

	if err := workflow.ExecuteActivity(ctx, "UpdateComplianceAssessment", assessInput).Get(ctx, nil); err != nil {
		logger.Warn("Failed to update compliance assessment", "error", err)
		// This is non-fatal, continue
	}

	// Step 7: Cleanup execution environment
	if err := workflow.ExecuteActivity(ctx, "CleanupInSpecEnvironment", prepareResult.TempDir).Get(ctx, nil); err != nil {
		logger.Warn("Failed to cleanup execution environment", "error", err)
		// Non-fatal
	}

	// Step 8: Update run status to completed
	completeInput := struct {
		RunID       string `json:"run_id"`
		Duration    int    `json:"duration"`
		TotalTests  int    `json:"total_tests"`
		PassedTests int    `json:"passed_tests"`
		FailedTests int    `json:"failed_tests"`
		SkippedTests int   `json:"skipped_tests"`
	}{
		RunID:       input.RunID,
		Duration:    execResult.Duration,
		TotalTests:  execResult.TotalTests,
		PassedTests: execResult.PassedTests,
		FailedTests: execResult.FailedTests,
		SkippedTests: execResult.SkippedTests,
	}

	if err := workflow.ExecuteActivity(ctx, "CompleteInSpecRun", completeInput).Get(ctx, nil); err != nil {
		logger.Error("Failed to complete run", "error", err)
		// Continue anyway
	}

	// Finalize result
	result.Status = InSpecStatusCompleted
	result.CompletedAt = workflow.Now(ctx)
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	logger.Info("InSpec execution workflow completed",
		"run_id", input.RunID,
		"status", result.Status,
		"total_tests", result.TotalTests,
		"passed_tests", result.PassedTests,
		"failed_tests", result.FailedTests,
		"duration", result.Duration,
	)

	return result, nil
}

// BatchInSpecExecutionWorkflow executes InSpec profiles across multiple assets.
type BatchInSpecExecutionInput struct {
	ProfileID string   `json:"profile_id"`
	AssetIDs  []string `json:"asset_ids"`
	OrgID     string   `json:"org_id"`
}

type BatchInSpecExecutionResult struct {
	TotalAssets      int      `json:"total_assets"`
	SuccessfulRuns   int      `json:"successful_runs"`
	FailedRuns       int      `json:"failed_runs"`
	RunIDs           []string `json:"run_ids"`
	CompletedAt      time.Time `json:"completed_at"`
}

// BatchInSpecExecutionWorkflow executes an InSpec profile across multiple assets in parallel.
func BatchInSpecExecutionWorkflow(ctx workflow.Context, input BatchInSpecExecutionInput) (*BatchInSpecExecutionResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting batch InSpec execution workflow",
		"profile_id", input.ProfileID,
		"asset_count", len(input.AssetIDs),
	)

	result := &BatchInSpecExecutionResult{
		TotalAssets: len(input.AssetIDs),
		RunIDs:      make([]string, 0, len(input.AssetIDs)),
	}

	// Create child workflows for each asset
	childWorkflows := make([]workflow.ChildWorkflowFuture, 0, len(input.AssetIDs))

	for _, assetID := range input.AssetIDs {
		childInput := InSpecWorkflowInput{
			ProfileID: input.ProfileID,
			AssetID:   assetID,
			OrgID:     input.OrgID,
		}

		childOptions := workflow.ChildWorkflowOptions{
			WorkflowID: "inspec-" + input.ProfileID + "-" + assetID,
		}
		childCtx := workflow.WithChildOptions(ctx, childOptions)

		future := workflow.ExecuteChildWorkflow(childCtx, InSpecExecutionWorkflow, childInput)
		childWorkflows = append(childWorkflows, future)
	}

	// Wait for all child workflows to complete
	for _, future := range childWorkflows {
		var childResult InSpecWorkflowResult
		if err := future.Get(ctx, &childResult); err != nil {
			logger.Warn("Child workflow failed", "error", err)
			result.FailedRuns++
		} else {
			result.SuccessfulRuns++
			result.RunIDs = append(result.RunIDs, childResult.RunID)
		}
	}

	result.CompletedAt = workflow.Now(ctx)

	logger.Info("Batch InSpec execution workflow completed",
		"total_assets", result.TotalAssets,
		"successful_runs", result.SuccessfulRuns,
		"failed_runs", result.FailedRuns,
	)

	return result, nil
}
