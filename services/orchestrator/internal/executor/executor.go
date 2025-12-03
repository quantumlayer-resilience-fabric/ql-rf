// Package executor implements the plan execution engine.
// It takes approved plans and executes them step by step.
package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/quantumlayerhq/ql-rf/pkg/database"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
)

// ExecutionStatus represents the status of an execution.
type ExecutionStatus string

const (
	StatusPending    ExecutionStatus = "pending"
	StatusRunning    ExecutionStatus = "running"
	StatusPaused     ExecutionStatus = "paused"
	StatusCompleted  ExecutionStatus = "completed"
	StatusFailed     ExecutionStatus = "failed"
	StatusRolledBack ExecutionStatus = "rolled_back"
	StatusCancelled  ExecutionStatus = "cancelled"
)

// PhaseStatus represents the status of a phase execution.
type PhaseStatus string

const (
	PhaseStatusPending   PhaseStatus = "pending"
	PhaseStatusRunning   PhaseStatus = "running"
	PhaseStatusWaiting   PhaseStatus = "waiting"
	PhaseStatusCompleted PhaseStatus = "completed"
	PhaseStatusFailed    PhaseStatus = "failed"
	PhaseStatusSkipped   PhaseStatus = "skipped"
)

// Execution represents a plan execution instance.
type Execution struct {
	ID            string                 `json:"id"`
	TaskID        string                 `json:"task_id"`
	PlanID        string                 `json:"plan_id"`
	OrgID         string                 `json:"org_id"`
	Status        ExecutionStatus        `json:"status"`
	StartedAt     time.Time              `json:"started_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	StartedBy     string                 `json:"started_by"`
	Phases        []PhaseExecution       `json:"phases"`
	CurrentPhase  int                    `json:"current_phase"`
	TotalPhases   int                    `json:"total_phases"`
	Error         string                 `json:"error,omitempty"`
	RollbackError string                 `json:"rollback_error,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// PhaseExecution tracks execution of a single phase.
type PhaseExecution struct {
	Name        string                 `json:"name"`
	Status      PhaseStatus            `json:"status"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Assets      []AssetExecution       `json:"assets"`
	WaitUntil   *time.Time             `json:"wait_until,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metrics     map[string]interface{} `json:"metrics,omitempty"`
}

// AssetExecution tracks execution for a single asset.
type AssetExecution struct {
	AssetID     string     `json:"asset_id"`
	AssetName   string     `json:"asset_name"`
	Status      string     `json:"status"` // pending, running, completed, failed, skipped
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
	Output      string     `json:"output,omitempty"`
}

// ExecutionPlan is the input plan to execute.
type ExecutionPlan struct {
	TaskID      string                 `json:"task_id"`
	PlanID      string                 `json:"plan_id"`
	OrgID       string                 `json:"org_id"`
	UserID      string                 `json:"user_id"`
	TaskType    string                 `json:"task_type"`
	Environment string                 `json:"environment"`
	Phases      []ExecutionPhase       `json:"phases"`
	Rollback    *RollbackPlan          `json:"rollback,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ExecutionPhase defines a phase in the execution plan.
type ExecutionPhase struct {
	Name          string                   `json:"name"`
	Assets        []map[string]interface{} `json:"assets"`
	WaitTime      string                   `json:"wait_time,omitempty"` // e.g., "30m"
	RollbackIf    string                   `json:"rollback_if,omitempty"`
	HealthChecks  []HealthCheck            `json:"health_checks,omitempty"`
	Actions       []PhaseAction            `json:"actions,omitempty"`
	ContinueOnFail bool                    `json:"continue_on_fail,omitempty"`
}

// PhaseAction defines an action to perform in a phase.
type PhaseAction struct {
	Type       string                 `json:"type"`
	Tool       string                 `json:"tool"`
	Parameters map[string]interface{} `json:"parameters"`
}

// HealthCheck defines a health check to run.
type HealthCheck struct {
	Name      string `json:"name"`
	Type      string `json:"type"` // http, tcp, command
	Target    string `json:"target"`
	Expected  string `json:"expected,omitempty"`
	Timeout   string `json:"timeout,omitempty"`
	Retries   int    `json:"retries,omitempty"`
}

// RollbackPlan defines how to rollback the execution.
type RollbackPlan struct {
	Strategy string           `json:"strategy"` // auto, manual, none
	Phases   []ExecutionPhase `json:"phases,omitempty"`
	Timeout  string           `json:"timeout,omitempty"`
}

// MaxExecutionTimeout is the maximum time an execution can run.
const MaxExecutionTimeout = 4 * time.Hour

// Engine is the execution engine that runs approved plans.
type Engine struct {
	db             *database.DB
	tools          *tools.Registry
	log            *logger.Logger
	executions     map[string]*Execution
	cancelFuncs    map[string]context.CancelFunc // Track cancel functions for running executions
	mu             sync.RWMutex
	healthChecker  *HealthChecker
	assetProcessor *AssetProcessor

	// Callbacks for notifications
	onPhaseStart    func(exec *Execution, phase *PhaseExecution)
	onPhaseComplete func(exec *Execution, phase *PhaseExecution)
	onExecutionDone func(exec *Execution)
}

// NewEngine creates a new execution engine.
func NewEngine(db *database.DB, toolReg *tools.Registry, log *logger.Logger) *Engine {
	var pool *database.DB
	if db != nil {
		pool = db
	}

	e := &Engine{
		db:          db,
		tools:       toolReg,
		log:         log.WithComponent("executor"),
		executions:  make(map[string]*Execution),
		cancelFuncs: make(map[string]context.CancelFunc),
	}

	// Initialize health checker
	e.healthChecker = NewHealthChecker(log)

	// Initialize asset processor with database pool
	if pool != nil {
		e.assetProcessor = NewAssetProcessor(pool.Pool, log)
	} else {
		e.assetProcessor = NewAssetProcessor(nil, log)
	}

	return e
}

// RegisterPlatformClient registers a platform client for asset operations.
func (e *Engine) RegisterPlatformClient(platform string, client PlatformClient) {
	// Convert string to Platform type
	var p models.Platform
	switch platform {
	case "aws":
		p = models.PlatformAWS
	case "azure":
		p = models.PlatformAzure
	case "gcp":
		p = models.PlatformGCP
	case "vsphere":
		p = models.PlatformVSphere
	default:
		e.log.Warn("unknown platform", "platform", platform)
		return
	}
	e.assetProcessor.RegisterPlatformClient(p, client)
}

// SetCallbacks sets the notification callbacks.
func (e *Engine) SetCallbacks(
	onPhaseStart func(exec *Execution, phase *PhaseExecution),
	onPhaseComplete func(exec *Execution, phase *PhaseExecution),
	onExecutionDone func(exec *Execution),
) {
	e.onPhaseStart = onPhaseStart
	e.onPhaseComplete = onPhaseComplete
	e.onExecutionDone = onExecutionDone
}

// Execute starts execution of a plan.
func (e *Engine) Execute(ctx context.Context, plan *ExecutionPlan) (*Execution, error) {
	exec := &Execution{
		ID:           uuid.New().String(),
		TaskID:       plan.TaskID,
		PlanID:       plan.PlanID,
		OrgID:        plan.OrgID,
		Status:       StatusRunning,
		StartedAt:    time.Now(),
		StartedBy:    plan.UserID,
		CurrentPhase: 0,
		TotalPhases:  len(plan.Phases),
		Phases:       make([]PhaseExecution, len(plan.Phases)),
		Metadata:     plan.Metadata,
	}

	// Initialize phases
	for i, phase := range plan.Phases {
		exec.Phases[i] = PhaseExecution{
			Name:   phase.Name,
			Status: PhaseStatusPending,
			Assets: make([]AssetExecution, len(phase.Assets)),
		}
		for j, asset := range phase.Assets {
			assetID, _ := asset["id"].(string)
			assetName, _ := asset["name"].(string)
			exec.Phases[i].Assets[j] = AssetExecution{
				AssetID:   assetID,
				AssetName: assetName,
				Status:    "pending",
			}
		}
	}

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(context.Background(), MaxExecutionTimeout)

	// Store execution and cancel function
	e.mu.Lock()
	e.executions[exec.ID] = exec
	e.cancelFuncs[exec.ID] = cancel
	e.mu.Unlock()

	// Save to database
	if err := e.saveExecution(ctx, exec); err != nil {
		e.log.Error("failed to save execution", "error", err)
	}

	// Execute in background with timeout context
	go e.runExecution(execCtx, exec, plan)

	e.log.Info("started execution",
		"execution_id", exec.ID,
		"task_id", plan.TaskID,
		"phases", len(plan.Phases),
	)

	return exec, nil
}

// runExecution runs the execution phases.
func (e *Engine) runExecution(ctx context.Context, exec *Execution, plan *ExecutionPlan) {
	defer func() {
		if r := recover(); r != nil {
			e.log.Error("execution panic", "error", r, "execution_id", exec.ID)
			exec.Status = StatusFailed
			exec.Error = fmt.Sprintf("panic: %v", r)
		}
		now := time.Now()
		exec.CompletedAt = &now

		// Clean up cancel function
		e.mu.Lock()
		if cancel, ok := e.cancelFuncs[exec.ID]; ok {
			cancel() // Ensure context is cancelled
			delete(e.cancelFuncs, exec.ID)
		}
		e.mu.Unlock()

		// Use a fresh context for final save since exec context may be cancelled
		saveCtx, saveCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer saveCancel()
		e.saveExecution(saveCtx, exec)

		if e.onExecutionDone != nil {
			e.onExecutionDone(exec)
		}
	}()

	for i, phase := range plan.Phases {
		// Check for cancellation before starting each phase
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				exec.Status = StatusFailed
				exec.Error = "execution timed out"
			} else {
				exec.Status = StatusCancelled
				exec.Error = "execution cancelled"
			}
			e.log.Info("execution stopped", "execution_id", exec.ID, "reason", ctx.Err())
			return
		default:
		}

		exec.CurrentPhase = i
		phaseExec := &exec.Phases[i]

		// Notify phase start
		if e.onPhaseStart != nil {
			e.onPhaseStart(exec, phaseExec)
		}

		// Execute phase
		err := e.executePhase(ctx, exec, phaseExec, &phase)
		if err != nil {
			e.log.Error("phase failed",
				"phase", phase.Name,
				"error", err,
				"execution_id", exec.ID,
			)

			phaseExec.Status = PhaseStatusFailed
			phaseExec.Error = err.Error()

			// Check if we should rollback
			if phase.RollbackIf != "" && plan.Rollback != nil && plan.Rollback.Strategy == "auto" {
				e.log.Info("initiating rollback", "execution_id", exec.ID)
				if rollbackErr := e.rollback(ctx, exec, plan.Rollback); rollbackErr != nil {
					exec.RollbackError = rollbackErr.Error()
				}
				exec.Status = StatusRolledBack
			} else if !phase.ContinueOnFail {
				exec.Status = StatusFailed
				exec.Error = fmt.Sprintf("Phase '%s' failed: %s", phase.Name, err.Error())
				return
			}
		}

		// Notify phase complete
		if e.onPhaseComplete != nil {
			e.onPhaseComplete(exec, phaseExec)
		}

		// Handle wait time between phases
		if phase.WaitTime != "" && i < len(plan.Phases)-1 {
			waitDuration, err := time.ParseDuration(phase.WaitTime)
			if err == nil && waitDuration > 0 {
				e.log.Info("waiting between phases",
					"duration", phase.WaitTime,
					"execution_id", exec.ID,
				)
				waitUntil := time.Now().Add(waitDuration)
				phaseExec.WaitUntil = &waitUntil
				e.saveExecution(ctx, exec)

				select {
				case <-time.After(waitDuration):
				case <-ctx.Done():
					exec.Status = StatusCancelled
					return
				}
			}
		}
	}

	exec.Status = StatusCompleted
	e.log.Info("execution completed", "execution_id", exec.ID)
}

// executePhase runs a single phase.
func (e *Engine) executePhase(ctx context.Context, exec *Execution, phaseExec *PhaseExecution, phase *ExecutionPhase) error {
	now := time.Now()
	phaseExec.StartedAt = &now
	phaseExec.Status = PhaseStatusRunning
	e.saveExecution(ctx, exec)

	// Execute actions if defined
	for _, action := range phase.Actions {
		if err := e.executeAction(ctx, exec, &action); err != nil {
			return fmt.Errorf("action %s failed: %w", action.Type, err)
		}
	}

	// Process assets in the phase
	for i := range phaseExec.Assets {
		asset := &phaseExec.Assets[i]
		assetStart := time.Now()
		asset.StartedAt = &assetStart
		asset.Status = "running"
		e.saveExecution(ctx, exec)

		// Simulate asset processing (in real implementation, call actual services)
		// This would call drift service, patch service, etc.
		err := e.processAsset(ctx, exec, asset, phase)

		assetEnd := time.Now()
		asset.CompletedAt = &assetEnd

		if err != nil {
			asset.Status = "failed"
			asset.Error = err.Error()
			return fmt.Errorf("asset %s failed: %w", asset.AssetName, err)
		}
		asset.Status = "completed"
		e.saveExecution(ctx, exec)
	}

	// Run health checks
	for _, hc := range phase.HealthChecks {
		if err := e.runHealthCheck(ctx, &hc); err != nil {
			return fmt.Errorf("health check '%s' failed: %w", hc.Name, err)
		}
	}

	phaseEnd := time.Now()
	phaseExec.CompletedAt = &phaseEnd
	phaseExec.Status = PhaseStatusCompleted
	e.saveExecution(ctx, exec)

	return nil
}

// executeAction executes a phase action using tools.
func (e *Engine) executeAction(ctx context.Context, exec *Execution, action *PhaseAction) error {
	if action.Tool == "" {
		return nil // No tool to execute
	}

	tool, ok := e.tools.Get(action.Tool)
	if !ok {
		return fmt.Errorf("tool not found: %s", action.Tool)
	}

	result, err := tool.Execute(ctx, action.Parameters)
	if err != nil {
		return err
	}

	e.log.Debug("action executed",
		"tool", action.Tool,
		"result", result,
		"execution_id", exec.ID,
	)

	return nil
}

// processAsset processes a single asset using the asset processor.
func (e *Engine) processAsset(ctx context.Context, exec *Execution, asset *AssetExecution, phase *ExecutionPhase) error {
	e.log.Info("processing asset",
		"asset_id", asset.AssetID,
		"asset_name", asset.AssetName,
		"phase", phase.Name,
		"execution_id", exec.ID,
	)

	// Get asset details from database
	assetInfo, err := e.getAssetInfo(ctx, asset.AssetID)
	if err != nil {
		return fmt.Errorf("failed to get asset info: %w", err)
	}

	// Determine action based on phase actions or default to validate
	action := ActionValidate
	var actionParams map[string]interface{}

	for _, phaseAction := range phase.Actions {
		if phaseAction.Type == "reimage" || phaseAction.Tool == "reimage_asset" {
			action = ActionReimage
			actionParams = phaseAction.Parameters
			break
		} else if phaseAction.Type == "reboot" || phaseAction.Tool == "reboot_asset" {
			action = ActionReboot
			actionParams = phaseAction.Parameters
			break
		} else if phaseAction.Type == "patch" || phaseAction.Tool == "patch_asset" {
			action = ActionPatch
			actionParams = phaseAction.Parameters
			break
		} else if phaseAction.Type == "terminate" || phaseAction.Tool == "terminate_asset" {
			action = ActionTerminate
			actionParams = phaseAction.Parameters
			break
		}
	}

	// Process the asset
	result, err := e.assetProcessor.ProcessAsset(ctx, assetInfo, action, actionParams)
	if err != nil {
		return err
	}

	asset.Output = result.Output
	return nil
}

// getAssetInfo retrieves asset information from the database.
func (e *Engine) getAssetInfo(ctx context.Context, assetID string) (*AssetInfo, error) {
	if e.db == nil {
		// Return minimal info if no database
		return &AssetInfo{
			ID:       assetID,
			Platform: models.PlatformAWS, // Default
		}, nil
	}

	query := `
		SELECT id, name, platform, region, instance_id, image_ref, image_version
		FROM assets
		WHERE id = $1
	`

	var info AssetInfo
	var name, imageRef, imageVersion *string
	err := e.db.QueryRow(ctx, query, assetID).Scan(
		&info.ID, &name, &info.Platform, &info.Region, &info.InstanceID, &imageRef, &imageVersion,
	)
	if err != nil {
		return nil, fmt.Errorf("asset not found: %w", err)
	}

	if name != nil {
		info.Name = *name
	}
	if imageRef != nil {
		info.CurrentImage = *imageRef
	}
	if imageVersion != nil {
		info.TargetImage = *imageVersion
	}

	return &info, nil
}

// runHealthCheck executes a health check.
func (e *Engine) runHealthCheck(ctx context.Context, hc *HealthCheck) error {
	e.log.Debug("running health check",
		"name", hc.Name,
		"type", hc.Type,
		"target", hc.Target,
	)

	// Parse timeout
	timeout := 30 * time.Second
	if hc.Timeout != "" {
		if d, err := time.ParseDuration(hc.Timeout); err == nil {
			timeout = d
		}
	}

	retries := hc.Retries
	if retries == 0 {
		retries = 3
	}

	var lastErr error
	for i := 0; i < retries; i++ {
		checkCtx, cancel := context.WithTimeout(ctx, timeout)
		err := e.performHealthCheck(checkCtx, hc)
		cancel()

		if err == nil {
			return nil
		}
		lastErr = err

		// Wait before retry
		if i < retries-1 {
			time.Sleep(5 * time.Second)
		}
	}

	return fmt.Errorf("health check failed after %d retries: %w", retries, lastErr)
}

// performHealthCheck performs the actual health check using the health checker.
func (e *Engine) performHealthCheck(ctx context.Context, hc *HealthCheck) error {
	result, err := e.healthChecker.Check(ctx, hc)
	if err != nil {
		e.log.Error("health check failed",
			"name", hc.Name,
			"type", hc.Type,
			"target", hc.Target,
			"error", err,
			"duration", result.Duration,
		)
		return err
	}

	e.log.Info("health check passed",
		"name", hc.Name,
		"type", hc.Type,
		"target", hc.Target,
		"duration", result.Duration,
	)
	return nil
}

// rollback executes the rollback plan.
func (e *Engine) rollback(ctx context.Context, exec *Execution, rollback *RollbackPlan) error {
	e.log.Info("executing rollback", "execution_id", exec.ID, "strategy", rollback.Strategy)

	// Parse rollback timeout
	timeout := 30 * time.Minute
	if rollback.Timeout != "" {
		if d, err := time.ParseDuration(rollback.Timeout); err == nil {
			timeout = d
		}
	}

	rollbackCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute rollback phases in reverse order
	for i := len(rollback.Phases) - 1; i >= 0; i-- {
		phase := &rollback.Phases[i]
		e.log.Info("executing rollback phase",
			"phase", phase.Name,
			"phase_index", i,
			"execution_id", exec.ID,
		)

		// Execute rollback actions for each asset in the phase
		for _, assetData := range phase.Assets {
			assetID, _ := assetData["id"].(string)
			assetName, _ := assetData["name"].(string)

			if assetID == "" {
				continue
			}

			e.log.Info("rolling back asset",
				"asset_id", assetID,
				"asset_name", assetName,
				"phase", phase.Name,
			)

			// Get asset info
			assetInfo, err := e.getAssetInfo(rollbackCtx, assetID)
			if err != nil {
				e.log.Error("failed to get asset info for rollback",
					"asset_id", assetID,
					"error", err,
				)
				continue // Try to rollback other assets
			}

			// Determine rollback action
			// Default is to reimage with previous image version
			action := ActionReimage
			actionParams := map[string]interface{}{}

			// Check if previous image is specified in asset data
			if prevImage, ok := assetData["previous_image"].(string); ok {
				actionParams["target_image"] = prevImage
			}

			// Execute rollback action for this asset
			result, err := e.assetProcessor.ProcessAsset(rollbackCtx, assetInfo, action, actionParams)
			if err != nil {
				e.log.Error("rollback failed for asset",
					"asset_id", assetID,
					"error", err,
				)
				// Continue with other assets even if one fails
				continue
			}

			e.log.Info("asset rollback completed",
				"asset_id", assetID,
				"result", result.Output,
			)
		}

		// Execute phase-level rollback actions
		for _, action := range phase.Actions {
			if action.Tool == "" {
				continue
			}

			e.log.Info("executing rollback action",
				"tool", action.Tool,
				"phase", phase.Name,
			)

			if err := e.executeAction(rollbackCtx, exec, &action); err != nil {
				e.log.Error("rollback action failed",
					"tool", action.Tool,
					"error", err,
				)
				// Continue with other actions
			}
		}

		// Run health checks after rollback phase
		for _, hc := range phase.HealthChecks {
			if err := e.runHealthCheck(rollbackCtx, &hc); err != nil {
				e.log.Warn("rollback health check failed",
					"name", hc.Name,
					"error", err,
				)
				// Don't fail rollback on health check failure
			}
		}
	}

	e.log.Info("rollback completed", "execution_id", exec.ID)
	return nil
}

// GetExecution retrieves an execution by ID.
func (e *Engine) GetExecution(ctx context.Context, execID string) (*Execution, error) {
	e.mu.RLock()
	exec, ok := e.executions[execID]
	e.mu.RUnlock()

	if ok {
		return exec, nil
	}

	// Try loading from database
	return e.loadExecution(ctx, execID)
}

// ListExecutions lists executions for a task.
func (e *Engine) ListExecutions(ctx context.Context, taskID string) ([]*Execution, error) {
	// First check in-memory executions
	e.mu.RLock()
	var result []*Execution
	for _, exec := range e.executions {
		if exec.TaskID == taskID {
			result = append(result, exec)
		}
	}
	e.mu.RUnlock()

	// If we have in-memory results, return them
	if len(result) > 0 {
		return result, nil
	}

	// Query database
	if e.db == nil {
		return result, nil
	}

	query := `
		SELECT id, plan_id, task_id, environment, initiated_by,
		       current_phase, phases_completed, phases_remaining, percent_complete,
		       state, error, started_at, completed_at
		FROM ai_runs
		WHERE task_id = $1
		ORDER BY created_at DESC
	`

	rows, err := e.db.Pool.Query(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var exec Execution
		var phasesCompletedJSON, phasesRemainingJSON []byte
		var dbState, environment, currentPhase string
		var errorStr *string

		err := rows.Scan(
			&exec.ID, &exec.PlanID, &exec.TaskID, &environment, &exec.StartedBy,
			&currentPhase, &phasesCompletedJSON, &phasesRemainingJSON, &exec.CurrentPhase,
			&dbState, &errorStr, &exec.StartedAt, &exec.CompletedAt,
		)
		if err != nil {
			continue
		}

		exec.Status = mapDBStateToStatus(dbState)
		if errorStr != nil {
			exec.Error = *errorStr
		}

		// Parse phases
		var phasesCompleted, phasesRemaining []string
		_ = json.Unmarshal(phasesCompletedJSON, &phasesCompleted)
		_ = json.Unmarshal(phasesRemainingJSON, &phasesRemaining)

		exec.Phases = []PhaseExecution{}
		for _, name := range phasesCompleted {
			exec.Phases = append(exec.Phases, PhaseExecution{Name: name, Status: PhaseStatusCompleted})
		}
		for _, name := range phasesRemaining {
			exec.Phases = append(exec.Phases, PhaseExecution{Name: name, Status: PhaseStatusPending})
		}
		exec.TotalPhases = len(exec.Phases)

		result = append(result, &exec)
	}

	return result, nil
}

// CancelExecution cancels a running execution.
func (e *Engine) CancelExecution(ctx context.Context, execID string) error {
	e.mu.Lock()
	exec, ok := e.executions[execID]
	cancel, hasCancel := e.cancelFuncs[execID]
	e.mu.Unlock()

	if !ok {
		return fmt.Errorf("execution not found: %s", execID)
	}

	if exec.Status != StatusRunning && exec.Status != StatusPaused {
		return fmt.Errorf("execution is not running: %s", exec.Status)
	}

	// Cancel the execution context to stop the running goroutine
	if hasCancel {
		cancel()
		e.log.Info("execution cancellation requested", "execution_id", execID)
	}

	exec.Status = StatusCancelled
	now := time.Now()
	exec.CompletedAt = &now

	return e.saveExecution(ctx, exec)
}

// PauseExecution pauses a running execution.
func (e *Engine) PauseExecution(ctx context.Context, execID string) error {
	e.mu.Lock()
	exec, ok := e.executions[execID]
	e.mu.Unlock()

	if !ok {
		return fmt.Errorf("execution not found: %s", execID)
	}

	if exec.Status != StatusRunning {
		return fmt.Errorf("execution is not running: %s", exec.Status)
	}

	exec.Status = StatusPaused
	return e.saveExecution(ctx, exec)
}

// ResumeExecution resumes a paused execution.
func (e *Engine) ResumeExecution(ctx context.Context, execID string) error {
	e.mu.Lock()
	exec, ok := e.executions[execID]
	e.mu.Unlock()

	if !ok {
		return fmt.Errorf("execution not found: %s", execID)
	}

	if exec.Status != StatusPaused {
		return fmt.Errorf("execution is not paused: %s", exec.Status)
	}

	exec.Status = StatusRunning
	return e.saveExecution(ctx, exec)
}

// saveExecution persists execution state to database.
func (e *Engine) saveExecution(ctx context.Context, exec *Execution) error {
	if e.db == nil {
		return nil // No database configured
	}

	// Build phases completed and remaining lists
	phasesCompleted := []string{}
	phasesRemaining := []string{}
	for i, phase := range exec.Phases {
		if phase.Status == PhaseStatusCompleted {
			phasesCompleted = append(phasesCompleted, phase.Name)
		} else if i >= exec.CurrentPhase {
			phasesRemaining = append(phasesRemaining, phase.Name)
		}
	}

	phasesCompletedJSON, _ := json.Marshal(phasesCompleted)
	phasesRemainingJSON, _ := json.Marshal(phasesRemaining)

	// Build metrics
	var assetsTotal, assetsChanged, assetsFailed, assetsSkipped int
	for _, phase := range exec.Phases {
		for _, asset := range phase.Assets {
			assetsTotal++
			switch asset.Status {
			case "completed":
				assetsChanged++
			case "failed":
				assetsFailed++
			case "skipped":
				assetsSkipped++
			}
		}
	}

	metrics := map[string]interface{}{
		"duration_seconds":      0,
		"assets_total":          assetsTotal,
		"assets_changed":        assetsChanged,
		"assets_failed":         assetsFailed,
		"assets_skipped":        assetsSkipped,
		"rollback_triggered":    exec.Status == StatusRolledBack,
		"rollback_assets":       0,
		"observed_error_rate":   0,
		"health_check_failures": 0,
	}
	if exec.CompletedAt != nil && exec.StartedAt.Before(*exec.CompletedAt) {
		metrics["duration_seconds"] = int(exec.CompletedAt.Sub(exec.StartedAt).Seconds())
	}
	metricsJSON, _ := json.Marshal(metrics)

	// Build audit log entry
	auditEntry := map[string]interface{}{
		"timestamp": time.Now().UTC(),
		"event":     string(exec.Status),
		"phase":     exec.CurrentPhase,
	}
	if exec.Error != "" {
		auditEntry["error"] = exec.Error
	}
	auditEntryJSON, _ := json.Marshal([]interface{}{auditEntry})

	// Calculate percent complete
	percentComplete := 0
	if exec.TotalPhases > 0 {
		completedPhases := len(phasesCompleted)
		percentComplete = (completedPhases * 100) / exec.TotalPhases
	}

	// Map our status to database state
	dbState := mapStatusToDBState(exec.Status)

	// Get current phase name
	currentPhaseName := ""
	if exec.CurrentPhase < len(exec.Phases) {
		currentPhaseName = exec.Phases[exec.CurrentPhase].Name
	}

	query := `
		INSERT INTO ai_runs (
			id, plan_id, task_id, environment, initiated_by,
			current_phase, phases_completed, phases_remaining, percent_complete,
			state, error, metrics, audit_log,
			started_at, completed_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			current_phase = EXCLUDED.current_phase,
			phases_completed = EXCLUDED.phases_completed,
			phases_remaining = EXCLUDED.phases_remaining,
			percent_complete = EXCLUDED.percent_complete,
			state = EXCLUDED.state,
			error = EXCLUDED.error,
			metrics = EXCLUDED.metrics,
			audit_log = ai_runs.audit_log || EXCLUDED.audit_log,
			completed_at = EXCLUDED.completed_at,
			updated_at = NOW()
	`

	return e.db.Exec(ctx, query,
		exec.ID,
		exec.PlanID,
		exec.TaskID,
		"production", // default environment
		exec.StartedBy,
		currentPhaseName,
		phasesCompletedJSON,
		phasesRemainingJSON,
		percentComplete,
		dbState,
		exec.Error,
		metricsJSON,
		auditEntryJSON,
		exec.StartedAt,
		exec.CompletedAt,
	)
}

// mapStatusToDBState maps ExecutionStatus to database state.
func mapStatusToDBState(status ExecutionStatus) string {
	switch status {
	case StatusPending:
		return "queued"
	case StatusRunning:
		return "executing"
	case StatusPaused:
		return "paused"
	case StatusCompleted:
		return "completed"
	case StatusRolledBack:
		return "rolled_back"
	case StatusFailed:
		return "failed"
	case StatusCancelled:
		return "failed" // Map cancelled to failed in DB
	default:
		return "queued"
	}
}

// loadExecution loads execution from database.
func (e *Engine) loadExecution(ctx context.Context, execID string) (*Execution, error) {
	if e.db == nil {
		return nil, fmt.Errorf("execution not found: %s", execID)
	}

	query := `
		SELECT id, plan_id, task_id, environment, initiated_by,
		       current_phase, phases_completed, phases_remaining, percent_complete,
		       state, error, metrics, started_at, completed_at
		FROM ai_runs WHERE id = $1
	`

	var exec Execution
	var phasesCompletedJSON, phasesRemainingJSON, metricsJSON []byte
	var dbState, environment, currentPhase string
	var errorStr *string

	err := e.db.QueryRow(ctx, query, execID).Scan(
		&exec.ID, &exec.PlanID, &exec.TaskID, &environment, &exec.StartedBy,
		&currentPhase, &phasesCompletedJSON, &phasesRemainingJSON, &exec.CurrentPhase,
		&dbState, &errorStr, &metricsJSON, &exec.StartedAt, &exec.CompletedAt,
	)
	if err != nil {
		return nil, err
	}

	// Map database state to ExecutionStatus
	exec.Status = mapDBStateToStatus(dbState)
	if errorStr != nil {
		exec.Error = *errorStr
	}

	// Parse phases from JSON
	var phasesCompleted, phasesRemaining []string
	_ = json.Unmarshal(phasesCompletedJSON, &phasesCompleted)
	_ = json.Unmarshal(phasesRemainingJSON, &phasesRemaining)

	// Reconstruct phases
	exec.Phases = []PhaseExecution{}
	for _, name := range phasesCompleted {
		exec.Phases = append(exec.Phases, PhaseExecution{
			Name:   name,
			Status: PhaseStatusCompleted,
		})
	}
	for _, name := range phasesRemaining {
		exec.Phases = append(exec.Phases, PhaseExecution{
			Name:   name,
			Status: PhaseStatusPending,
		})
	}
	exec.TotalPhases = len(exec.Phases)

	return &exec, nil
}

// mapDBStateToStatus maps database state to ExecutionStatus.
func mapDBStateToStatus(dbState string) ExecutionStatus {
	switch dbState {
	case "queued":
		return StatusPending
	case "executing":
		return StatusRunning
	case "paused":
		return StatusPaused
	case "completed":
		return StatusCompleted
	case "rolled_back":
		return StatusRolledBack
	case "failed":
		return StatusFailed
	default:
		return StatusPending
	}
}
