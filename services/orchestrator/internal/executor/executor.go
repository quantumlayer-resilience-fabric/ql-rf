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

// Engine is the execution engine that runs approved plans.
type Engine struct {
	db         *database.DB
	tools      *tools.Registry
	log        *logger.Logger
	executions map[string]*Execution
	mu         sync.RWMutex

	// Callbacks for notifications
	onPhaseStart    func(exec *Execution, phase *PhaseExecution)
	onPhaseComplete func(exec *Execution, phase *PhaseExecution)
	onExecutionDone func(exec *Execution)
}

// NewEngine creates a new execution engine.
func NewEngine(db *database.DB, toolReg *tools.Registry, log *logger.Logger) *Engine {
	return &Engine{
		db:         db,
		tools:      toolReg,
		log:        log.WithComponent("executor"),
		executions: make(map[string]*Execution),
	}
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

	// Store execution
	e.mu.Lock()
	e.executions[exec.ID] = exec
	e.mu.Unlock()

	// Save to database
	if err := e.saveExecution(ctx, exec); err != nil {
		e.log.Error("failed to save execution", "error", err)
	}

	// Execute in background
	go e.runExecution(context.Background(), exec, plan)

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
		e.saveExecution(ctx, exec)
		if e.onExecutionDone != nil {
			e.onExecutionDone(exec)
		}
	}()

	for i, phase := range plan.Phases {
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

// processAsset processes a single asset (simulated for now).
func (e *Engine) processAsset(ctx context.Context, exec *Execution, asset *AssetExecution, phase *ExecutionPhase) error {
	// In a real implementation, this would:
	// 1. Call the drift service to apply changes
	// 2. Call the patch service to apply patches
	// 3. Call Terraform to apply infrastructure changes
	// 4. etc.

	// For now, simulate processing
	e.log.Info("processing asset",
		"asset_id", asset.AssetID,
		"asset_name", asset.AssetName,
		"phase", phase.Name,
		"execution_id", exec.ID,
	)

	// Simulate some work
	select {
	case <-time.After(100 * time.Millisecond):
		asset.Output = "Simulated execution completed successfully"
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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

// performHealthCheck performs the actual health check.
func (e *Engine) performHealthCheck(ctx context.Context, hc *HealthCheck) error {
	// In real implementation, would perform HTTP/TCP/command checks
	// For now, simulate success
	return nil
}

// rollback executes the rollback plan.
func (e *Engine) rollback(ctx context.Context, exec *Execution, rollback *RollbackPlan) error {
	e.log.Info("executing rollback", "execution_id", exec.ID)

	// Execute rollback phases in reverse order
	for i := len(rollback.Phases) - 1; i >= 0; i-- {
		phase := &rollback.Phases[i]
		e.log.Info("rollback phase", "phase", phase.Name, "execution_id", exec.ID)

		// In real implementation, would execute rollback actions
		// For now, simulate rollback
		time.Sleep(100 * time.Millisecond)
	}

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
	// In real implementation, would query database
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []*Execution
	for _, exec := range e.executions {
		if exec.TaskID == taskID {
			result = append(result, exec)
		}
	}
	return result, nil
}

// CancelExecution cancels a running execution.
func (e *Engine) CancelExecution(ctx context.Context, execID string) error {
	e.mu.Lock()
	exec, ok := e.executions[execID]
	e.mu.Unlock()

	if !ok {
		return fmt.Errorf("execution not found: %s", execID)
	}

	if exec.Status != StatusRunning && exec.Status != StatusPaused {
		return fmt.Errorf("execution is not running: %s", exec.Status)
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

	data, err := json.Marshal(exec)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO executions (id, task_id, org_id, status, data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			data = EXCLUDED.data,
			updated_at = EXCLUDED.updated_at
	`

	err = e.db.Exec(ctx, query,
		exec.ID,
		exec.TaskID,
		exec.OrgID,
		string(exec.Status),
		data,
		exec.StartedAt,
		time.Now(),
	)

	return err
}

// loadExecution loads execution from database.
func (e *Engine) loadExecution(ctx context.Context, execID string) (*Execution, error) {
	if e.db == nil {
		return nil, fmt.Errorf("execution not found: %s", execID)
	}

	query := `SELECT data FROM executions WHERE id = $1`

	var data []byte
	err := e.db.QueryRow(ctx, query, execID).Scan(&data)
	if err != nil {
		return nil, err
	}

	var exec Execution
	if err := json.Unmarshal(data, &exec); err != nil {
		return nil, err
	}

	return &exec, nil
}
