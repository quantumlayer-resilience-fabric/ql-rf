// Package workflows defines Temporal workflows for task execution.
package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// TaskWorkflowInput contains the input for the task execution workflow.
type TaskWorkflowInput struct {
	TaskID      string                 `json:"task_id"`
	TaskType    string                 `json:"task_type"`
	OrgID       string                 `json:"org_id"`
	UserID      string                 `json:"user_id"`
	Environment string                 `json:"environment"`
	Goal        string                 `json:"goal"`
	RiskLevel   string                 `json:"risk_level"`
	Plan        map[string]interface{} `json:"plan"`
	Context     map[string]interface{} `json:"context"`
}

// TaskWorkflowResult contains the result of the task execution workflow.
type TaskWorkflowResult struct {
	TaskID        string                 `json:"task_id"`
	Status        string                 `json:"status"`
	ExecutedAt    time.Time              `json:"executed_at"`
	CompletedAt   time.Time              `json:"completed_at"`
	Duration      time.Duration          `json:"duration"`
	Result        map[string]interface{} `json:"result"`
	Error         string                 `json:"error,omitempty"`
	AffectedItems int                    `json:"affected_items"`
}

// ApprovalSignal represents an approval action from a user.
type ApprovalSignal struct {
	Action     string `json:"action"` // approve, reject, modify
	ApprovedBy string `json:"approved_by"`
	Reason     string `json:"reason,omitempty"`
}

const (
	// Signal names
	SignalApproval = "approval"

	// Workflow statuses
	StatusPending   = "pending_approval"
	StatusApproved  = "approved"
	StatusRejected  = "rejected"
	StatusExecuting = "executing"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// TaskExecutionWorkflow orchestrates the full lifecycle of a task:
// 1. Wait for HITL approval (if required)
// 2. Execute the task plan
// 3. Report results
func TaskExecutionWorkflow(ctx workflow.Context, input TaskWorkflowInput) (*TaskWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting task execution workflow",
		"task_id", input.TaskID,
		"task_type", input.TaskType,
		"risk_level", input.RiskLevel,
	)

	result := &TaskWorkflowResult{
		TaskID:     input.TaskID,
		Status:     StatusPending,
		ExecutedAt: workflow.Now(ctx),
	}

	// Configure activity options
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	// Step 1: Update task status to pending
	if err := workflow.ExecuteActivity(ctx, "UpdateTaskStatus", input.TaskID, StatusPending).Get(ctx, nil); err != nil {
		logger.Warn("Failed to update task status", "error", err)
		// Continue anyway
	}

	// Step 2: Wait for approval signal (with timeout)
	approvalCh := workflow.GetSignalChannel(ctx, SignalApproval)

	var approval ApprovalSignal
	approvalTimeout := 24 * time.Hour // Tasks expire after 24 hours without action

	selector := workflow.NewSelector(ctx)
	var timedOut bool

	selector.AddReceive(approvalCh, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &approval)
	})

	selector.AddFuture(workflow.NewTimer(ctx, approvalTimeout), func(f workflow.Future) {
		timedOut = true
	})

	selector.Select(ctx)

	if timedOut {
		logger.Info("Task approval timed out", "task_id", input.TaskID)
		result.Status = StatusCancelled
		result.Error = "approval timeout: no action taken within 24 hours"
		result.CompletedAt = workflow.Now(ctx)
		result.Duration = result.CompletedAt.Sub(result.ExecutedAt)

		// Update status in database
		_ = workflow.ExecuteActivity(ctx, "UpdateTaskStatus", input.TaskID, StatusCancelled).Get(ctx, nil)

		return result, nil
	}

	// Step 3: Handle the approval action
	logger.Info("Received approval signal",
		"task_id", input.TaskID,
		"action", approval.Action,
		"approved_by", approval.ApprovedBy,
	)

	switch approval.Action {
	case "reject":
		result.Status = StatusRejected
		result.Error = approval.Reason
		result.CompletedAt = workflow.Now(ctx)
		result.Duration = result.CompletedAt.Sub(result.ExecutedAt)

		_ = workflow.ExecuteActivity(ctx, "UpdateTaskStatus", input.TaskID, StatusRejected).Get(ctx, nil)
		_ = workflow.ExecuteActivity(ctx, "RecordAuditLog", AuditLogInput{
			TaskID:    input.TaskID,
			Action:    "reject",
			UserID:    approval.ApprovedBy,
			Reason:    approval.Reason,
			Timestamp: workflow.Now(ctx),
		}).Get(ctx, nil)

		return result, nil

	case "modify":
		// For modify, we update the task and keep waiting for final approval
		_ = workflow.ExecuteActivity(ctx, "UpdateTaskPlan", input.TaskID, approval.Reason).Get(ctx, nil)

		// Wait for another approval signal
		approvalCh.Receive(ctx, &approval)
		if approval.Action != "approve" {
			result.Status = StatusRejected
			result.CompletedAt = workflow.Now(ctx)
			result.Duration = result.CompletedAt.Sub(result.ExecutedAt)
			return result, nil
		}
		fallthrough

	case "approve":
		result.Status = StatusApproved
		_ = workflow.ExecuteActivity(ctx, "UpdateTaskStatus", input.TaskID, StatusApproved).Get(ctx, nil)
		_ = workflow.ExecuteActivity(ctx, "RecordAuditLog", AuditLogInput{
			TaskID:    input.TaskID,
			Action:    "approve",
			UserID:    approval.ApprovedBy,
			Reason:    approval.Reason,
			Timestamp: workflow.Now(ctx),
		}).Get(ctx, nil)
	}

	// Step 4: Execute the task
	logger.Info("Executing task", "task_id", input.TaskID, "task_type", input.TaskType)

	_ = workflow.ExecuteActivity(ctx, "UpdateTaskStatus", input.TaskID, StatusExecuting).Get(ctx, nil)

	// Execute based on task type
	var execResult ExecuteTaskActivityResult
	execInput := ExecuteTaskActivityInput{
		TaskID:      input.TaskID,
		TaskType:    input.TaskType,
		OrgID:       input.OrgID,
		Environment: input.Environment,
		Plan:        input.Plan,
		Context:     input.Context,
	}

	// Use longer timeout for execution
	execOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		HeartbeatTimeout:    time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 5,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute * 5,
			MaximumAttempts:    2, // Limited retries for execution
		},
	}
	execCtx := workflow.WithActivityOptions(ctx, execOpts)

	if err := workflow.ExecuteActivity(execCtx, "ExecuteTask", execInput).Get(execCtx, &execResult); err != nil {
		logger.Error("Task execution failed", "task_id", input.TaskID, "error", err)
		result.Status = StatusFailed
		result.Error = err.Error()
		result.CompletedAt = workflow.Now(ctx)
		result.Duration = result.CompletedAt.Sub(result.ExecutedAt)

		_ = workflow.ExecuteActivity(ctx, "UpdateTaskStatus", input.TaskID, StatusFailed).Get(ctx, nil)
		_ = workflow.ExecuteActivity(ctx, "SendNotification", NotificationInput{
			TaskID:  input.TaskID,
			Type:    "task_failed",
			Message: "Task execution failed: " + err.Error(),
			UserID:  input.UserID,
		}).Get(ctx, nil)

		return result, nil
	}

	// Step 5: Complete successfully
	logger.Info("Task completed successfully",
		"task_id", input.TaskID,
		"affected_items", execResult.AffectedItems,
	)

	result.Status = StatusCompleted
	result.CompletedAt = workflow.Now(ctx)
	result.Duration = result.CompletedAt.Sub(result.ExecutedAt)
	result.Result = execResult.Result
	result.AffectedItems = execResult.AffectedItems

	_ = workflow.ExecuteActivity(ctx, "UpdateTaskStatus", input.TaskID, StatusCompleted).Get(ctx, nil)
	_ = workflow.ExecuteActivity(ctx, "RecordAuditLog", AuditLogInput{
		TaskID:    input.TaskID,
		Action:    "complete",
		UserID:    "system",
		Timestamp: workflow.Now(ctx),
	}).Get(ctx, nil)
	_ = workflow.ExecuteActivity(ctx, "SendNotification", NotificationInput{
		TaskID:  input.TaskID,
		Type:    "task_completed",
		Message: "Task completed successfully",
		UserID:  input.UserID,
	}).Get(ctx, nil)

	return result, nil
}

// AuditLogInput for recording audit logs.
type AuditLogInput struct {
	TaskID    string    `json:"task_id"`
	Action    string    `json:"action"`
	UserID    string    `json:"user_id"`
	Reason    string    `json:"reason,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// NotificationInput for sending notifications.
type NotificationInput struct {
	TaskID  string `json:"task_id"`
	Type    string `json:"type"`
	Message string `json:"message"`
	UserID  string `json:"user_id"`
}

// ExecuteTaskActivityInput for task execution activity.
type ExecuteTaskActivityInput struct {
	TaskID      string                 `json:"task_id"`
	TaskType    string                 `json:"task_type"`
	OrgID       string                 `json:"org_id"`
	Environment string                 `json:"environment"`
	Plan        map[string]interface{} `json:"plan"`
	Context     map[string]interface{} `json:"context"`
}

// ExecuteTaskActivityResult from task execution.
type ExecuteTaskActivityResult struct {
	Success       bool                   `json:"success"`
	Result        map[string]interface{} `json:"result"`
	AffectedItems int                    `json:"affected_items"`
	Error         string                 `json:"error,omitempty"`
}
