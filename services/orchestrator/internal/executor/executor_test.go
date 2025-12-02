package executor

import (
	"context"
	"testing"
	"time"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEngine(t *testing.T) {
	log := logger.New("debug", "json")
	toolReg := tools.NewRegistry(nil, log)

	engine := NewEngine(nil, toolReg, log)

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.executions)
	assert.NotNil(t, engine.tools)
}

func TestExecute(t *testing.T) {
	log := logger.New("debug", "json")
	toolReg := tools.NewRegistry(nil, log)
	engine := NewEngine(nil, toolReg, log)

	ctx := context.Background()

	plan := &ExecutionPlan{
		TaskID:      "task-123",
		PlanID:      "plan-123",
		OrgID:       "org-123",
		UserID:      "user-123",
		TaskType:    "drift_remediation",
		Environment: "staging",
		Phases: []ExecutionPhase{
			{
				Name: "Canary",
				Assets: []map[string]interface{}{
					{"id": "asset-1", "name": "web-server-1"},
					{"id": "asset-2", "name": "web-server-2"},
				},
			},
			{
				Name: "Wave 1",
				Assets: []map[string]interface{}{
					{"id": "asset-3", "name": "web-server-3"},
					{"id": "asset-4", "name": "web-server-4"},
				},
				WaitTime: "100ms",
			},
		},
	}

	exec, err := engine.Execute(ctx, plan)

	require.NoError(t, err)
	assert.NotEmpty(t, exec.ID)
	assert.Equal(t, "task-123", exec.TaskID)
	assert.Equal(t, "plan-123", exec.PlanID)
	assert.Equal(t, StatusRunning, exec.Status)
	assert.Len(t, exec.Phases, 2)
	assert.Equal(t, 2, exec.TotalPhases)

	// Wait for execution to complete
	time.Sleep(500 * time.Millisecond)

	// Check final status
	exec, err = engine.GetExecution(ctx, exec.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, exec.Status)
}

func TestExecuteWithCallbacks(t *testing.T) {
	log := logger.New("debug", "json")
	toolReg := tools.NewRegistry(nil, log)
	engine := NewEngine(nil, toolReg, log)

	var phaseStartCount, phaseCompleteCount int
	var executionDone bool

	engine.SetCallbacks(
		func(exec *Execution, phase *PhaseExecution) {
			phaseStartCount++
		},
		func(exec *Execution, phase *PhaseExecution) {
			phaseCompleteCount++
		},
		func(exec *Execution) {
			executionDone = true
		},
	)

	ctx := context.Background()

	plan := &ExecutionPlan{
		TaskID:      "task-456",
		PlanID:      "plan-456",
		OrgID:       "org-123",
		UserID:      "user-123",
		TaskType:    "patch_rollout",
		Environment: "staging",
		Phases: []ExecutionPhase{
			{Name: "Phase 1", Assets: []map[string]interface{}{{"id": "a1", "name": "Asset 1"}}},
			{Name: "Phase 2", Assets: []map[string]interface{}{{"id": "a2", "name": "Asset 2"}}},
		},
	}

	_, err := engine.Execute(ctx, plan)
	require.NoError(t, err)

	// Wait for execution to complete
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, 2, phaseStartCount)
	assert.Equal(t, 2, phaseCompleteCount)
	assert.True(t, executionDone)
}

func TestGetExecution(t *testing.T) {
	log := logger.New("debug", "json")
	toolReg := tools.NewRegistry(nil, log)
	engine := NewEngine(nil, toolReg, log)

	ctx := context.Background()

	plan := &ExecutionPlan{
		TaskID:      "task-789",
		PlanID:      "plan-789",
		OrgID:       "org-123",
		UserID:      "user-123",
		TaskType:    "compliance_audit",
		Environment: "production",
		Phases: []ExecutionPhase{
			{Name: "Audit", Assets: []map[string]interface{}{}},
		},
	}

	exec, err := engine.Execute(ctx, plan)
	require.NoError(t, err)

	// Get the execution
	retrieved, err := engine.GetExecution(ctx, exec.ID)
	require.NoError(t, err)
	assert.Equal(t, exec.ID, retrieved.ID)
	assert.Equal(t, exec.TaskID, retrieved.TaskID)
}

func TestGetExecution_NotFound(t *testing.T) {
	log := logger.New("debug", "json")
	toolReg := tools.NewRegistry(nil, log)
	engine := NewEngine(nil, toolReg, log)

	ctx := context.Background()

	_, err := engine.GetExecution(ctx, "nonexistent-id")
	assert.Error(t, err)
}

func TestListExecutions(t *testing.T) {
	log := logger.New("debug", "json")
	toolReg := tools.NewRegistry(nil, log)
	engine := NewEngine(nil, toolReg, log)

	ctx := context.Background()

	// Create multiple executions for the same task
	plan1 := &ExecutionPlan{
		TaskID:      "task-list",
		PlanID:      "plan-1",
		OrgID:       "org-123",
		UserID:      "user-123",
		TaskType:    "test",
		Environment: "staging",
		Phases:      []ExecutionPhase{{Name: "P1", Assets: []map[string]interface{}{}}},
	}

	plan2 := &ExecutionPlan{
		TaskID:      "task-list",
		PlanID:      "plan-2",
		OrgID:       "org-123",
		UserID:      "user-123",
		TaskType:    "test",
		Environment: "staging",
		Phases:      []ExecutionPhase{{Name: "P1", Assets: []map[string]interface{}{}}},
	}

	_, err := engine.Execute(ctx, plan1)
	require.NoError(t, err)

	_, err = engine.Execute(ctx, plan2)
	require.NoError(t, err)

	// Wait for completions
	time.Sleep(300 * time.Millisecond)

	executions, err := engine.ListExecutions(ctx, "task-list")
	require.NoError(t, err)
	assert.Len(t, executions, 2)
}

func TestCancelExecution(t *testing.T) {
	log := logger.New("debug", "json")
	toolReg := tools.NewRegistry(nil, log)
	engine := NewEngine(nil, toolReg, log)

	ctx := context.Background()

	// Create a long-running execution with multiple phases and wait time between them
	plan := &ExecutionPlan{
		TaskID:      "task-cancel",
		PlanID:      "plan-cancel",
		OrgID:       "org-123",
		UserID:      "user-123",
		TaskType:    "test",
		Environment: "staging",
		Phases: []ExecutionPhase{
			{
				Name:     "Phase 1",
				WaitTime: "5s", // Wait time after this phase
				Assets:   []map[string]interface{}{{"id": "a1", "name": "Asset 1"}},
			},
			{
				Name:   "Phase 2",
				Assets: []map[string]interface{}{{"id": "a2", "name": "Asset 2"}},
			},
		},
	}

	exec, err := engine.Execute(ctx, plan)
	require.NoError(t, err)

	// Wait for first phase to complete and wait time to start
	time.Sleep(200 * time.Millisecond)

	err = engine.CancelExecution(ctx, exec.ID)
	// Cancel may fail if execution already completed, which is acceptable in tests
	if err == nil {
		// Check status
		exec, err = engine.GetExecution(ctx, exec.ID)
		require.NoError(t, err)
		assert.Equal(t, StatusCancelled, exec.Status)
	} else {
		// If cancel failed, execution should have completed
		exec, _ = engine.GetExecution(ctx, exec.ID)
		assert.Equal(t, StatusCompleted, exec.Status)
	}
}

func TestPauseAndResumeExecution(t *testing.T) {
	log := logger.New("debug", "json")
	toolReg := tools.NewRegistry(nil, log)
	engine := NewEngine(nil, toolReg, log)

	ctx := context.Background()

	plan := &ExecutionPlan{
		TaskID:      "task-pause",
		PlanID:      "plan-pause",
		OrgID:       "org-123",
		UserID:      "user-123",
		TaskType:    "test",
		Environment: "staging",
		Phases: []ExecutionPhase{
			{Name: "P1", Assets: []map[string]interface{}{{"id": "a1", "name": "Asset"}}},
		},
	}

	exec, err := engine.Execute(ctx, plan)
	require.NoError(t, err)

	// Wait for execution to be in running state
	time.Sleep(50 * time.Millisecond)

	// Pause
	err = engine.PauseExecution(ctx, exec.ID)
	// May fail if execution already completed, which is fine
	if err == nil {
		exec, _ = engine.GetExecution(ctx, exec.ID)
		assert.Equal(t, StatusPaused, exec.Status)

		// Resume
		err = engine.ResumeExecution(ctx, exec.ID)
		require.NoError(t, err)

		exec, _ = engine.GetExecution(ctx, exec.ID)
		assert.Equal(t, StatusRunning, exec.Status)
	}
}

func TestExecutionPlan_WithRollback(t *testing.T) {
	log := logger.New("debug", "json")
	toolReg := tools.NewRegistry(nil, log)
	engine := NewEngine(nil, toolReg, log)

	ctx := context.Background()

	plan := &ExecutionPlan{
		TaskID:      "task-rollback",
		PlanID:      "plan-rollback",
		OrgID:       "org-123",
		UserID:      "user-123",
		TaskType:    "drift_remediation",
		Environment: "production",
		Phases: []ExecutionPhase{
			{
				Name:       "Canary",
				Assets:     []map[string]interface{}{{"id": "a1", "name": "Asset 1"}},
				RollbackIf: "error_rate > 0.01",
			},
		},
		Rollback: &RollbackPlan{
			Strategy: "auto",
			Phases: []ExecutionPhase{
				{Name: "Rollback Canary", Assets: []map[string]interface{}{}},
			},
		},
	}

	exec, err := engine.Execute(ctx, plan)
	require.NoError(t, err)

	// Wait for completion
	time.Sleep(300 * time.Millisecond)

	exec, err = engine.GetExecution(ctx, exec.ID)
	require.NoError(t, err)
	// Should complete successfully since no actual failure occurs
	assert.Equal(t, StatusCompleted, exec.Status)
}

func TestPhaseExecution_AssetTracking(t *testing.T) {
	log := logger.New("debug", "json")
	toolReg := tools.NewRegistry(nil, log)
	engine := NewEngine(nil, toolReg, log)

	ctx := context.Background()

	plan := &ExecutionPlan{
		TaskID:      "task-assets",
		PlanID:      "plan-assets",
		OrgID:       "org-123",
		UserID:      "user-123",
		TaskType:    "patch_rollout",
		Environment: "staging",
		Phases: []ExecutionPhase{
			{
				Name: "Patch Wave",
				Assets: []map[string]interface{}{
					{"id": "server-1", "name": "web-server-1"},
					{"id": "server-2", "name": "web-server-2"},
					{"id": "server-3", "name": "web-server-3"},
				},
			},
		},
	}

	exec, err := engine.Execute(ctx, plan)
	require.NoError(t, err)

	// Wait for completion
	time.Sleep(500 * time.Millisecond)

	exec, err = engine.GetExecution(ctx, exec.ID)
	require.NoError(t, err)

	assert.Len(t, exec.Phases[0].Assets, 3)
	for _, asset := range exec.Phases[0].Assets {
		assert.Equal(t, "completed", asset.Status)
		assert.NotNil(t, asset.StartedAt)
		assert.NotNil(t, asset.CompletedAt)
	}
}

func TestExecutionStatus_Constants(t *testing.T) {
	assert.Equal(t, ExecutionStatus("pending"), StatusPending)
	assert.Equal(t, ExecutionStatus("running"), StatusRunning)
	assert.Equal(t, ExecutionStatus("paused"), StatusPaused)
	assert.Equal(t, ExecutionStatus("completed"), StatusCompleted)
	assert.Equal(t, ExecutionStatus("failed"), StatusFailed)
	assert.Equal(t, ExecutionStatus("rolled_back"), StatusRolledBack)
	assert.Equal(t, ExecutionStatus("cancelled"), StatusCancelled)
}

func TestPhaseStatus_Constants(t *testing.T) {
	assert.Equal(t, PhaseStatus("pending"), PhaseStatusPending)
	assert.Equal(t, PhaseStatus("running"), PhaseStatusRunning)
	assert.Equal(t, PhaseStatus("waiting"), PhaseStatusWaiting)
	assert.Equal(t, PhaseStatus("completed"), PhaseStatusCompleted)
	assert.Equal(t, PhaseStatus("failed"), PhaseStatusFailed)
	assert.Equal(t, PhaseStatus("skipped"), PhaseStatusSkipped)
}
