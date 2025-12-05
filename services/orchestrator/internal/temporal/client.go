// Package temporal provides Temporal workflow integration for durable task execution.
package temporal

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Config holds Temporal client configuration.
type Config struct {
	Host      string
	Port      int
	Namespace string
	TaskQueue string
}

// DefaultConfig returns default Temporal configuration.
func DefaultConfig() *Config {
	return &Config{
		Host:      "localhost",
		Port:      7233,
		Namespace: "default",
		TaskQueue: "qlrf-task-queue",
	}
}

// Client wraps the Temporal client with QL-RF specific functionality.
type Client struct {
	cfg      *Config
	client   client.Client
	worker   worker.Worker
	isWorker bool
}

// NewClient creates a new Temporal client.
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	
	opts := client.Options{
		HostPort:  addr,
		Namespace: cfg.Namespace,
	}

	c, err := client.Dial(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Temporal: %w", err)
	}

	return &Client{
		cfg:    cfg,
		client: c,
	}, nil
}

// Close closes the Temporal client.
func (c *Client) Close() {
	if c.worker != nil {
		c.worker.Stop()
	}
	if c.client != nil {
		c.client.Close()
	}
}

// GetClient returns the underlying Temporal client.
func (c *Client) GetClient() client.Client {
	return c.client
}

// StartWorker starts the Temporal worker with registered workflows and activities.
func (c *Client) StartWorker(activities *Activities) error {
	workerOpts := worker.Options{
		MaxConcurrentActivityExecutionSize:     10,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
		EnableSessionWorker:                    true,
	}

	c.worker = worker.New(c.client, c.cfg.TaskQueue, workerOpts)

	// Register workflows
	c.worker.RegisterWorkflow(TaskExecutionWorkflow)
	c.worker.RegisterWorkflow(PatchDeploymentWorkflow)
	c.worker.RegisterWorkflow(DRDrillWorkflow)
	c.worker.RegisterWorkflow(ComplianceScanWorkflow)

	// Register activities
	c.worker.RegisterActivity(activities)

	c.isWorker = true
	return c.worker.Start()
}

// ExecuteTaskWorkflow starts a task execution workflow.
func (c *Client) ExecuteTaskWorkflow(ctx context.Context, input TaskExecutionInput) (string, error) {
	workflowID := fmt.Sprintf("task-execution-%s", input.TaskID)
	
	options := client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                c.cfg.TaskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
		WorkflowTaskTimeout:      5 * time.Minute,
	}

	run, err := c.client.ExecuteWorkflow(ctx, options, TaskExecutionWorkflow, input)
	if err != nil {
		return "", fmt.Errorf("failed to start task execution workflow: %w", err)
	}

	return run.GetID(), nil
}

// ExecutePatchWorkflow starts a patch deployment workflow.
func (c *Client) ExecutePatchWorkflow(ctx context.Context, input PatchDeploymentInput) (string, error) {
	workflowID := fmt.Sprintf("patch-deployment-%s-%d", input.OrgID, time.Now().Unix())
	
	options := client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                c.cfg.TaskQueue,
		WorkflowExecutionTimeout: 8 * time.Hour,
		WorkflowTaskTimeout:      5 * time.Minute,
	}

	run, err := c.client.ExecuteWorkflow(ctx, options, PatchDeploymentWorkflow, input)
	if err != nil {
		return "", fmt.Errorf("failed to start patch deployment workflow: %w", err)
	}

	return run.GetID(), nil
}

// ExecuteDRDrillWorkflow starts a DR drill workflow.
func (c *Client) ExecuteDRDrillWorkflow(ctx context.Context, input DRDrillInput) (string, error) {
	workflowID := fmt.Sprintf("dr-drill-%s-%d", input.DrillID, time.Now().Unix())
	
	options := client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                c.cfg.TaskQueue,
		WorkflowExecutionTimeout: 72 * time.Hour, // DR drills can run for days
		WorkflowTaskTimeout:      5 * time.Minute,
	}

	run, err := c.client.ExecuteWorkflow(ctx, options, DRDrillWorkflow, input)
	if err != nil {
		return "", fmt.Errorf("failed to start DR drill workflow: %w", err)
	}

	return run.GetID(), nil
}

// ExecuteComplianceScanWorkflow starts a compliance scan workflow.
func (c *Client) ExecuteComplianceScanWorkflow(ctx context.Context, input ComplianceScanInput) (string, error) {
	workflowID := fmt.Sprintf("compliance-scan-%s-%d", input.OrgID, time.Now().Unix())
	
	options := client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                c.cfg.TaskQueue,
		WorkflowExecutionTimeout: 4 * time.Hour,
		WorkflowTaskTimeout:      5 * time.Minute,
	}

	run, err := c.client.ExecuteWorkflow(ctx, options, ComplianceScanWorkflow, input)
	if err != nil {
		return "", fmt.Errorf("failed to start compliance scan workflow: %w", err)
	}

	return run.GetID(), nil
}

// GetWorkflowStatus retrieves the status of a running workflow.
func (c *Client) GetWorkflowStatus(ctx context.Context, workflowID string) (*WorkflowStatus, error) {
	desc, err := c.client.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to describe workflow: %w", err)
	}

	status := &WorkflowStatus{
		WorkflowID: workflowID,
		Status:     desc.WorkflowExecutionInfo.Status.String(),
		StartTime:  desc.WorkflowExecutionInfo.StartTime.AsTime(),
	}

	if desc.WorkflowExecutionInfo.CloseTime != nil {
		closeTime := desc.WorkflowExecutionInfo.CloseTime.AsTime()
		status.EndTime = &closeTime
	}

	return status, nil
}

// CancelWorkflow cancels a running workflow.
func (c *Client) CancelWorkflow(ctx context.Context, workflowID string) error {
	return c.client.CancelWorkflow(ctx, workflowID, "")
}

// SignalWorkflow sends a signal to a running workflow.
func (c *Client) SignalWorkflow(ctx context.Context, workflowID, signalName string, signalArg interface{}) error {
	return c.client.SignalWorkflow(ctx, workflowID, "", signalName, signalArg)
}

// WorkflowStatus represents the status of a workflow execution.
type WorkflowStatus struct {
	WorkflowID string     `json:"workflow_id"`
	Status     string     `json:"status"`
	StartTime  time.Time  `json:"start_time"`
	EndTime    *time.Time `json:"end_time,omitempty"`
}
