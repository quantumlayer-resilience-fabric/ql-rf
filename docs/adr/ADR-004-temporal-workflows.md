# ADR-004: Temporal for Workflows

## Status
Accepted

## Context
QL-RF requires orchestration of long-running, multi-step workflows:
- Image build pipelines (packer build → test → sign → publish)
- Patch rollout campaigns (canary → staged → fleet)
- DR drills (provision pilot-light → validate → measure RTO)
- Compliance evidence generation

These workflows must be:
- Durable (survive crashes, restarts)
- Observable (track progress, debug failures)
- Recoverable (retry failed steps, compensate)
- Scalable (handle concurrent campaigns)

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
4. **Observability**: Built-in UI for workflow inspection
5. **Signals/queries**: Runtime interaction with workflows

Example workflow:
```go
func ImageBuildWorkflow(ctx workflow.Context, req ImageBuildRequest) error {
    // Step 1: Build image
    var imageID string
    err := workflow.ExecuteActivity(ctx, BuildImage, req).Get(ctx, &imageID)
    if err != nil {
        return err
    }

    // Step 2: Run tests
    err = workflow.ExecuteActivity(ctx, TestImage, imageID).Get(ctx, nil)
    if err != nil {
        // Compensate: delete failed image
        workflow.ExecuteActivity(ctx, DeleteImage, imageID)
        return err
    }

    // Step 3: Sign and publish
    return workflow.ExecuteActivity(ctx, SignAndPublish, imageID).Get(ctx, nil)
}
```

## Consequences

### Positive
- Workflows written in Go (same as services)
- Automatic retry, timeout, heartbeat handling
- Workflow versioning for safe updates
- Excellent debugging via Temporal UI
- Scales to thousands of concurrent workflows

### Negative
- Additional infrastructure (Temporal Server + persistence)
- Learning curve for Temporal concepts
- Vendor dependency (though open-source)

### Mitigations
- Use Temporal Cloud initially, self-host later if needed
- Create workflow templates for common patterns
- Document workflow development guidelines
- Use Temporal Go SDK's testing framework
