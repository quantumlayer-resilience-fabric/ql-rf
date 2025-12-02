# ADR-008: Task/Plan/Run Lifecycle & State Machine

## Status
Accepted

## Context
ADR-007 introduces LLM-first orchestration where AI agents generate plans for infrastructure operations. These AI-driven workflows need:
- Clear state tracking for observability
- Idempotent operations for retries
- Audit trail for compliance
- Metrics collection for AI quality assessment
- Correlation between intent, plan, and execution outcome

Without explicit lifecycle management, we risk:
- Lost state on service restarts
- Duplicate executions
- Incomplete audit records
- Inability to measure AI effectiveness

## Decision
We define three first-class entities with explicit lifecycles:

### 1. Task (User Intent)
Represents what the user asked for.

```yaml
Task:
  id: uuid
  org_id: uuid
  created_by: user_id
  created_at: timestamp

  # User input
  user_intent: "Fix drift on prod web servers"

  # Parsed by meta-prompt engine
  task_spec:
    task_type: drift_remediation
    goal: "Remediate drift for production web servers"
    context:
      environment: production
      scope:
        platforms: [aws, azure]
        asset_filter: "role:web-server"
    agents: [drift_agent]
    tools_required: [query_assets, compare_versions, generate_patch_plan]
    risk_level: high
    hitl_required: true
    constraints:
      canary_required: true
      max_batch_percent: 10
      rollback_trigger: "error_rate > 1%"

  # Execution policy (tenant-configurable)
  execution_policy:
    mode: plan_only | canary_only | full_auto
    allowed_approver_roles: [PlatformAdmin, SRELead]

  # Versioning for regression tracking
  llm_profile:
    model: "claude-3.5-sonnet"
    agent_version: "drift_agent@1.2.0"
    prompts_version: "2024-12-01"

  # Lifecycle
  status:
    state: created | parsing | planned | failed
    updated_at: timestamp
    error: string?
```

### 2. Plan (AI-Generated Artifact)
Represents what the AI proposes to do.

```yaml
Plan:
  id: uuid
  task_id: uuid  # References Task
  created_at: timestamp

  # Plan type and content
  type: drift_plan | patch_plan | dr_runbook | compliance_report | ...
  payload:
    summary: "Remediate 47 drifted servers in us-east-1"
    affected_assets: [asset_ids...]
    phases:
      - name: Canary
        assets: [5 asset_ids]
        wait_time: "30m"
        health_checks: [http_200, error_rate_below_1pct]
        rollback_if: "error_rate > 1%"
      - name: Wave1
        assets: [15 asset_ids]
        wait_time: "15m"
        health_checks: [...]
        rollback_if: "error_rate > 0.5%"
      - name: Wave2
        assets: [27 asset_ids]
        health_checks: [...]
    estimated_duration: "2h"
    risk_assessment: "Medium - well-tested image, canary phase included"

  # Validation results
  validation:
    schema_valid: true
    schema_errors: []
    opa_valid: true
    opa_violations: []
    safety_valid: true
    safety_violations: []
    overall_valid: true

  # Quality scoring (for AI feedback loop)
  quality_score: 85  # 0-100, based on completeness/safety

  # Lifecycle
  status:
    state: draft | validated | awaiting_approval | approved | rejected | superseded
    updated_at: timestamp
    approved_by: user_id?
    approved_at: timestamp?
    rejection_reason: string?
```

### 3. Run (Execution Attempt)
Represents an actual execution of a Plan.

```yaml
Run:
  id: uuid
  plan_id: uuid  # References Plan
  task_id: uuid  # References Task (denormalized for queries)
  started_at: timestamp
  completed_at: timestamp?

  # Execution context
  environment: production | staging | development
  initiated_by: user_id  # Who approved

  # Progress tracking
  current_phase: "Wave1"
  phases_completed: ["Canary"]
  phases_remaining: ["Wave2"]

  # Lifecycle
  status:
    state: queued | executing | paused | completed | rolled_back | failed
    updated_at: timestamp
    error: string?

  # Outcome metrics (for AI feedback)
  metrics:
    duration_seconds: 3600
    assets_changed: 42
    assets_failed: 0
    rollback_triggered: false
    error_rate: 0.2%  # Observed during execution
    health_check_failures: 0

  # Audit
  audit_log:
    - timestamp: ...
      event: phase_started
      phase: Canary
    - timestamp: ...
      event: health_check_passed
      phase: Canary
    - timestamp: ...
      event: phase_completed
      phase: Canary
```

### State Machine

```
TASK STATES
───────────
created ──→ parsing ──→ planned ──→ (terminal)
              │
              └──→ failed (parse error)


PLAN STATES
───────────
draft ──→ validated ──→ awaiting_approval ──→ approved ──→ (terminal)
  │          │                │
  │          │                └──→ rejected (terminal)
  │          │
  │          └──→ draft (validation failed, retry)
  │
  └──→ superseded (new plan generated)


RUN STATES
──────────
queued ──→ executing ──→ completed (terminal)
              │
              ├──→ paused ──→ executing (resume)
              │       │
              │       └──→ failed (timeout)
              │
              ├──→ rolled_back (terminal)
              │
              └──→ failed (terminal)
```

### State Transitions

| Entity | From | To | Trigger |
|--------|------|----|---------|
| Task | created | parsing | Meta-prompt starts |
| Task | parsing | planned | TaskSpec generated successfully |
| Task | parsing | failed | Parse/validation error |
| Plan | draft | validated | All validation passes |
| Plan | validated | awaiting_approval | HITL required |
| Plan | awaiting_approval | approved | User approves |
| Plan | awaiting_approval | rejected | User rejects |
| Plan | * | superseded | New plan for same task |
| Run | queued | executing | Execution starts |
| Run | executing | paused | Health check failure (auto-pause) |
| Run | executing | completed | All phases done |
| Run | executing | rolled_back | Rollback triggered |
| Run | executing | failed | Unrecoverable error |
| Run | paused | executing | Manual resume |

## Consequences

### Positive
- **Observability**: Clear states visible in UI timeline
- **Idempotency**: Safe to retry operations
- **Auditability**: Complete trail from intent to outcome
- **Metrics**: Can measure AI quality per version
- **Debugging**: Easy to trace failures
- **Compliance**: Evidence of approval workflow

### Negative
- More database tables/queries
- State management complexity
- Need to handle edge cases (orphaned tasks, stuck runs)

### Mitigations
- Use Temporal for durable state (integrates with ADR-004)
- Background jobs for cleanup (orphan detection)
- Clear API for state transitions
- Comprehensive error handling

## API Design

```
# Task operations
POST   /api/v1/ai/tasks              # Create task (submits user intent)
GET    /api/v1/ai/tasks              # List tasks (with filters)
GET    /api/v1/ai/tasks/{id}         # Get task details
DELETE /api/v1/ai/tasks/{id}         # Cancel task (if not executing)

# Plan operations
GET    /api/v1/ai/tasks/{id}/plans   # List plans for task
GET    /api/v1/ai/plans/{id}         # Get plan details
POST   /api/v1/ai/plans/{id}/approve # Approve plan
POST   /api/v1/ai/plans/{id}/reject  # Reject plan
POST   /api/v1/ai/plans/{id}/modify  # Request modifications

# Run operations
GET    /api/v1/ai/plans/{id}/runs    # List runs for plan
GET    /api/v1/ai/runs/{id}          # Get run details
POST   /api/v1/ai/runs/{id}/pause    # Pause execution
POST   /api/v1/ai/runs/{id}/resume   # Resume execution
POST   /api/v1/ai/runs/{id}/rollback # Trigger rollback
```

## Database Schema

```sql
-- Tasks table
CREATE TABLE ai_tasks (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id),
    created_by UUID NOT NULL REFERENCES users(id),
    user_intent TEXT NOT NULL,
    task_spec JSONB,
    execution_policy JSONB,
    llm_profile JSONB,
    state VARCHAR(20) NOT NULL DEFAULT 'created',
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Plans table
CREATE TABLE ai_plans (
    id UUID PRIMARY KEY,
    task_id UUID NOT NULL REFERENCES ai_tasks(id),
    type VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    validation JSONB,
    quality_score INTEGER,
    state VARCHAR(20) NOT NULL DEFAULT 'draft',
    approved_by UUID REFERENCES users(id),
    approved_at TIMESTAMPTZ,
    rejection_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Runs table
CREATE TABLE ai_runs (
    id UUID PRIMARY KEY,
    plan_id UUID NOT NULL REFERENCES ai_plans(id),
    task_id UUID NOT NULL REFERENCES ai_tasks(id),
    environment VARCHAR(20) NOT NULL,
    initiated_by UUID NOT NULL REFERENCES users(id),
    current_phase VARCHAR(100),
    state VARCHAR(20) NOT NULL DEFAULT 'queued',
    metrics JSONB,
    audit_log JSONB,
    error TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_ai_tasks_org ON ai_tasks(org_id);
CREATE INDEX idx_ai_tasks_state ON ai_tasks(state);
CREATE INDEX idx_ai_plans_task ON ai_plans(task_id);
CREATE INDEX idx_ai_plans_state ON ai_plans(state);
CREATE INDEX idx_ai_runs_plan ON ai_runs(plan_id);
CREATE INDEX idx_ai_runs_state ON ai_runs(state);
```

## References
- ADR-007: LLM-First Orchestration Architecture
- ADR-004: Temporal for Workflows
- [Temporal Workflow States](https://docs.temporal.io/workflows#status)
