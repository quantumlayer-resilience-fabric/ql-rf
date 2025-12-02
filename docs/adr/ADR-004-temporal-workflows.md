# ADR-004: Temporal for Workflows

## Status
**Implemented** ✅

## Context
QL-RF requires orchestration of long-running, multi-step workflows:
- AI task execution with human-in-the-loop (HITL) approval
- Drift remediation campaigns (canary → staged → fleet)
- Patch rollout campaigns
- DR drills (provision pilot-light → validate → measure RTO)
- Compliance evidence generation

These workflows must be:
- Durable (survive crashes, restarts)
- Observable (track progress, debug failures)
- Recoverable (retry failed steps, compensate)
- Scalable (handle concurrent campaigns)
- Support signals for runtime interaction (e.g., approval signals)

Options considered:
1. **Argo Workflows**: Kubernetes-native, YAML-based
2. **Temporal**: Language-native, code-as-workflow
3. **AWS Step Functions**: Managed, JSON/YAML state machines
4. **Custom queue + state machine**: Full control, high effort

## Decision
We adopt **Temporal** for workflow orchestration:

1. **Code-as-workflow**: Write workflows in Go (our primary language)
2. **Durable execution**: Automatic state persistence and recovery
3. **Native retries**: Configurable retry policies per activity
4. **Observability**: Built-in UI for workflow inspection (port 8088)
5. **Signals/queries**: Runtime interaction with workflows

## Implementation

### Location
- `services/orchestrator/internal/temporal/workflows/` - Workflow definitions
- `services/orchestrator/internal/temporal/activities/` - Activity implementations
- `services/orchestrator/internal/temporal/worker/` - Worker setup

### Task Execution Workflow
The primary workflow handles AI task execution with HITL approval:

```go
// services/orchestrator/internal/temporal/workflows/task_workflow.go
func TaskExecutionWorkflow(ctx workflow.Context, input TaskWorkflowInput) (*TaskWorkflowResult, error) {
    // Step 1: Update task status to pending
    workflow.ExecuteActivity(ctx, "UpdateTaskStatus", input.TaskID, StatusPending)

    // Step 2: Wait for approval signal (24h timeout)
    approvalCh := workflow.GetSignalChannel(ctx, SignalApproval)
    var approval ApprovalSignal

    selector := workflow.NewSelector(ctx)
    selector.AddReceive(approvalCh, func(c workflow.ReceiveChannel, more bool) {
        c.Receive(ctx, &approval)
    })
    selector.AddFuture(workflow.NewTimer(ctx, 24*time.Hour), func(f workflow.Future) {
        // Timeout - cancel task
    })
    selector.Select(ctx)

    // Step 3: Handle approval action (approve/reject/modify)
    switch approval.Action {
    case "approve":
        // Execute the task
        workflow.ExecuteActivity(ctx, "ExecuteTask", execInput)
    case "reject":
        // Record rejection
    case "modify":
        // Update plan and wait for re-approval
    }

    // Step 4: Send notifications and record audit log
    workflow.ExecuteActivity(ctx, "SendNotification", ...)
    workflow.ExecuteActivity(ctx, "RecordAuditLog", ...)

    return result, nil
}
```

### Activities
- `UpdateTaskStatus` - Updates task state in PostgreSQL
- `RecordAuditLog` - Records audit trail via ai_tool_invocations table
- `SendNotification` - Logs notifications (extensible for email/slack)
- `UpdateTaskPlan` - Handles task modifications
- `ExecuteTask` - Executes tasks by type (drift_remediation, patch_rollout, etc.)

### Handler Integration
The HTTP handlers signal workflows for approval/rejection:

```go
// services/orchestrator/internal/handlers/handlers.go
func (h *Handler) approveTask(w http.ResponseWriter, r *http.Request) {
    approval := workflows.ApprovalSignal{
        Action:     "approve",
        ApprovedBy: userID,
    }
    h.temporalWorker.SignalApproval(ctx, taskID, approval)
}
```

### Infrastructure
- Temporal Server runs via Docker Compose (temporalio/auto-setup:1.24.2)
- Temporal UI available at http://localhost:8088
- Uses PostgreSQL for persistence (shared with QL-RF)
- Task queue: `ql-rf-orchestrator`

## Consequences

### Positive
- Workflows written in Go (same as services)
- Automatic retry, timeout, heartbeat handling
- Workflow versioning for safe updates
- Excellent debugging via Temporal UI
- Scales to thousands of concurrent workflows
- HITL approval with 24-hour timeout built-in

### Negative
- Additional infrastructure (Temporal Server + persistence)
- Learning curve for Temporal concepts
- Vendor dependency (though open-source)

### Mitigations
- ✅ Docker Compose setup for local development
- ✅ Worker gracefully starts even if Temporal unavailable
- ✅ Fallback to direct database updates if workflow fails
- Create workflow templates for common patterns
- Use Temporal Go SDK's testing framework
