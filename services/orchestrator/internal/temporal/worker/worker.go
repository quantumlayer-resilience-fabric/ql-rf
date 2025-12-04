// Package worker provides the Temporal worker for the AI orchestrator.
package worker

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/agents"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/executor"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/temporal/activities"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/temporal/workflows"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

const (
	// TaskQueue is the default task queue name for the orchestrator.
	TaskQueue = "ql-rf-orchestrator"

	// WorkflowTypeTaskExecution is the workflow type for task execution.
	WorkflowTypeTaskExecution = "TaskExecutionWorkflow"

	// WorkflowTypeDRDrill is the workflow type for DR drill execution.
	WorkflowTypeDRDrill = "DRDrillWorkflow"
)

// Worker wraps the Temporal worker with our dependencies.
type Worker struct {
	client        client.Client
	worker        worker.Worker
	log           *logger.Logger
	activities    *activities.Activities
}

// Config holds configuration for creating a worker.
type Config struct {
	Temporal      config.TemporalConfig
	DB            *pgxpool.Pool
	Logger        *logger.Logger
	AgentRegistry *agents.Registry
	ToolRegistry  *tools.Registry
}

// New creates a new Temporal worker.
func New(cfg Config) (*Worker, error) {
	log := cfg.Logger.WithComponent("temporal-worker")

	// Create Temporal client options
	clientOpts := client.Options{
		HostPort:  cfg.Temporal.Address(),
		Namespace: cfg.Temporal.Namespace,
		Logger:    newTemporalLogger(log),
	}

	// Configure TLS if enabled (for Temporal Cloud)
	if cfg.Temporal.TLSEnabled {
		cert, err := tls.LoadX509KeyPair(cfg.Temporal.TLSCertPath, cfg.Temporal.TLSKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificates: %w", err)
		}
		clientOpts.ConnectionOptions = client.ConnectionOptions{
			TLS: &tls.Config{
				Certificates: []tls.Certificate{cert},
			},
		}
	}

	// Create Temporal client
	c, err := client.Dial(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client: %w", err)
	}

	// Determine task queue
	taskQueue := cfg.Temporal.TaskQueue
	if taskQueue == "" {
		taskQueue = TaskQueue
	}

	// Create activities instance
	acts := activities.NewActivities(cfg.DB, cfg.Logger, cfg.AgentRegistry, cfg.ToolRegistry)

	// Create worker
	workerOpts := worker.Options{
		MaxConcurrentWorkflowTaskExecutionSize: cfg.Temporal.MaxConcurrentWorkflows,
		MaxConcurrentActivityExecutionSize:     cfg.Temporal.MaxConcurrentActivities,
	}
	if workerOpts.MaxConcurrentWorkflowTaskExecutionSize == 0 {
		workerOpts.MaxConcurrentWorkflowTaskExecutionSize = 100
	}
	if workerOpts.MaxConcurrentActivityExecutionSize == 0 {
		workerOpts.MaxConcurrentActivityExecutionSize = 50
	}

	w := worker.New(c, taskQueue, workerOpts)

	// Register workflows
	w.RegisterWorkflow(workflows.TaskExecutionWorkflow)
	w.RegisterWorkflow(workflows.DRDrillWorkflow)

	// Register task execution activities
	w.RegisterActivity(acts.UpdateTaskStatus)
	w.RegisterActivity(acts.RecordAuditLog)
	w.RegisterActivity(acts.SendNotification)
	w.RegisterActivity(acts.UpdateTaskPlan)
	w.RegisterActivity(acts.ExecuteTask)

	// Register DR drill activities
	w.RegisterActivity(acts.ExecuteDRPhase)
	w.RegisterActivity(acts.NotifyDRDrillStarted)
	w.RegisterActivity(acts.NotifyDRDrillCompleted)
	w.RegisterActivity(acts.StoreDRDrillResult)

	log.Info("Temporal worker created",
		"task_queue", taskQueue,
		"namespace", cfg.Temporal.Namespace,
		"host", cfg.Temporal.Address(),
	)

	return &Worker{
		client:     c,
		worker:     w,
		log:        log,
		activities: acts,
	}, nil
}

// Start starts the Temporal worker.
func (w *Worker) Start() error {
	w.log.Info("Starting Temporal worker")
	return w.worker.Start()
}

// Run starts the worker and blocks until stopped.
func (w *Worker) Run(interrupt <-chan interface{}) error {
	w.log.Info("Running Temporal worker")
	return w.worker.Run(interrupt)
}

// Stop stops the Temporal worker gracefully.
func (w *Worker) Stop() {
	w.log.Info("Stopping Temporal worker")
	w.worker.Stop()
	w.client.Close()
}

// Client returns the underlying Temporal client for starting workflows.
func (w *Worker) Client() client.Client {
	return w.client
}

// StartTaskWorkflow starts a new task execution workflow.
func (w *Worker) StartTaskWorkflow(ctx context.Context, input workflows.TaskWorkflowInput) (string, error) {
	workflowOpts := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("task-%s", input.TaskID),
		TaskQueue: TaskQueue,
	}

	we, err := w.client.ExecuteWorkflow(ctx, workflowOpts, workflows.TaskExecutionWorkflow, input)
	if err != nil {
		return "", fmt.Errorf("failed to start workflow: %w", err)
	}

	w.log.Info("Started task workflow",
		"workflow_id", we.GetID(),
		"run_id", we.GetRunID(),
		"task_id", input.TaskID,
	)

	return we.GetRunID(), nil
}

// SignalApproval sends an approval signal to a task workflow.
func (w *Worker) SignalApproval(ctx context.Context, taskID string, approval workflows.ApprovalSignal) error {
	workflowID := fmt.Sprintf("task-%s", taskID)

	err := w.client.SignalWorkflow(ctx, workflowID, "", workflows.SignalApproval, approval)
	if err != nil {
		return fmt.Errorf("failed to signal workflow: %w", err)
	}

	w.log.Info("Sent approval signal",
		"workflow_id", workflowID,
		"action", approval.Action,
		"approved_by", approval.ApprovedBy,
	)

	return nil
}

// GetWorkflowStatus gets the status of a task workflow.
func (w *Worker) GetWorkflowStatus(ctx context.Context, taskID string) (string, error) {
	workflowID := fmt.Sprintf("task-%s", taskID)

	desc, err := w.client.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return "", fmt.Errorf("failed to describe workflow: %w", err)
	}

	return desc.WorkflowExecutionInfo.Status.String(), nil
}

// CancelWorkflow cancels a running task workflow.
func (w *Worker) CancelWorkflow(ctx context.Context, taskID string) error {
	workflowID := fmt.Sprintf("task-%s", taskID)

	err := w.client.CancelWorkflow(ctx, workflowID, "")
	if err != nil {
		return fmt.Errorf("failed to cancel workflow: %w", err)
	}

	w.log.Info("Cancelled workflow", "workflow_id", workflowID)
	return nil
}

// StartDRDrillWorkflow starts a new DR drill workflow.
func (w *Worker) StartDRDrillWorkflow(ctx context.Context, input workflows.DRDrillWorkflowInput) (string, error) {
	workflowOpts := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("dr-drill-%s", input.DrillID),
		TaskQueue: TaskQueue,
	}

	we, err := w.client.ExecuteWorkflow(ctx, workflowOpts, workflows.DRDrillWorkflow, input)
	if err != nil {
		return "", fmt.Errorf("failed to start DR drill workflow: %w", err)
	}

	w.log.Info("Started DR drill workflow",
		"workflow_id", we.GetID(),
		"run_id", we.GetRunID(),
		"drill_id", input.DrillID,
		"drill_type", input.DrillType,
		"pairs", len(input.DrPairIDs),
	)

	return we.GetRunID(), nil
}

// GetDRDrillStatus gets the status of a DR drill workflow.
func (w *Worker) GetDRDrillStatus(ctx context.Context, drillID string) (string, error) {
	workflowID := fmt.Sprintf("dr-drill-%s", drillID)

	desc, err := w.client.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return "", fmt.Errorf("failed to describe DR drill workflow: %w", err)
	}

	return desc.WorkflowExecutionInfo.Status.String(), nil
}

// GetDRDrillResult gets the result of a completed DR drill workflow.
func (w *Worker) GetDRDrillResult(ctx context.Context, drillID string, runID string) (*workflows.DRDrillWorkflowResult, error) {
	workflowID := fmt.Sprintf("dr-drill-%s", drillID)

	workflowRun := w.client.GetWorkflow(ctx, workflowID, runID)

	var result workflows.DRDrillWorkflowResult
	if err := workflowRun.Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("failed to get DR drill result: %w", err)
	}

	return &result, nil
}

// CancelDRDrill cancels a running DR drill workflow.
func (w *Worker) CancelDRDrill(ctx context.Context, drillID string) error {
	workflowID := fmt.Sprintf("dr-drill-%s", drillID)

	err := w.client.CancelWorkflow(ctx, workflowID, "")
	if err != nil {
		return fmt.Errorf("failed to cancel DR drill workflow: %w", err)
	}

	w.log.Info("Cancelled DR drill workflow", "workflow_id", workflowID)
	return nil
}

// RegisterPlatformClient registers a platform client for real asset operations.
// This enables activities to perform actual patching, reimaging, etc. on cloud platforms.
func (w *Worker) RegisterPlatformClient(platform models.Platform, client executor.PlatformClient) {
	if w.activities != nil {
		w.activities.RegisterPlatformClient(platform, client)
		w.log.Info("Registered platform client", "platform", platform)
	}
}

// temporalLogger adapts our logger to Temporal's logger interface.
type temporalLogger struct {
	log *logger.Logger
}

func newTemporalLogger(log *logger.Logger) *temporalLogger {
	return &temporalLogger{log: log}
}

func (l *temporalLogger) Debug(msg string, keyvals ...interface{}) {
	l.log.Debug(msg, keyvals...)
}

func (l *temporalLogger) Info(msg string, keyvals ...interface{}) {
	l.log.Info(msg, keyvals...)
}

func (l *temporalLogger) Warn(msg string, keyvals ...interface{}) {
	l.log.Warn(msg, keyvals...)
}

func (l *temporalLogger) Error(msg string, keyvals ...interface{}) {
	l.log.Error(msg, keyvals...)
}
