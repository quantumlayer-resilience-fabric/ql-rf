# AI Orchestrator Schemas

This directory contains JSON Schema definitions for the LLM-First AI Orchestration system.

## Schema Overview

| Schema | Description | Used By |
|--------|-------------|---------|
| [task.schema.json](task.schema.json) | AI Task entity (user request + status) | API, Database |
| [task-spec.schema.json](task-spec.schema.json) | Parsed TaskSpec from meta-prompt engine | Agents, Validation |
| [plan.schema.json](plan.schema.json) | AI-generated execution plan | Agents, HITL, Execution |
| [run.schema.json](run.schema.json) | Execution record with audit trail | Execution, Monitoring |

## Entity Relationships

```
┌──────────────────────────────────────────────────────────────────┐
│                         TASK                                      │
│  (User intent → Parsed TaskSpec)                                  │
│                                                                    │
│  id: uuid                                                          │
│  user_intent: "Fix drift on prod web servers"                     │
│  task_spec: { parsed TaskSpec }                                   │
│  status: { state: created|parsing|planned|failed }                │
└─────────────────────────┬────────────────────────────────────────┘
                          │ 1:N
                          ▼
┌──────────────────────────────────────────────────────────────────┐
│                         PLAN                                      │
│  (AI-generated execution plan)                                    │
│                                                                    │
│  id: uuid                                                          │
│  task_id: uuid (FK)                                               │
│  type: drift_plan|patch_plan|dr_runbook|...                       │
│  payload: { phases, health_checks, rollback_conditions }          │
│  validation: { schema_valid, opa_valid, safety_valid }            │
│  status: { state: draft|validated|approved|rejected|... }         │
└─────────────────────────┬────────────────────────────────────────┘
                          │ 1:N
                          ▼
┌──────────────────────────────────────────────────────────────────┐
│                          RUN                                      │
│  (Execution attempt with metrics)                                 │
│                                                                    │
│  id: uuid                                                          │
│  plan_id: uuid (FK)                                               │
│  environment: production|staging|...                              │
│  progress: { current_phase, phases_completed, ... }               │
│  metrics: { duration, assets_changed, error_rate, ... }           │
│  audit_log: [ { timestamp, event, details } ]                     │
│  status: { state: queued|executing|completed|rolled_back|failed } │
└──────────────────────────────────────────────────────────────────┘
```

## State Machines

### Task States
```
created ──→ parsing ──→ planned
              │
              └──→ failed
```

### Plan States
```
draft ──→ validated ──→ awaiting_approval ──→ approved
  │          │                │
  │          └──→ draft       └──→ rejected
  │          (validation failed)
  └──→ superseded
```

### Run States
```
queued ──→ executing ──→ completed
              │
              ├──→ paused ──→ executing
              ├──→ rolled_back
              └──→ failed
```

## Validation

Validate schemas using `ajv`:

```bash
# Install ajv-cli
npm install -g ajv-cli

# Validate a task
ajv validate -s schemas/ai/task.schema.json -d task.json

# Validate a plan
ajv validate -s schemas/ai/plan.schema.json -d plan.json
```

## Example Documents

### Task Example
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "org_id": "org-123",
  "created_by": "user-456",
  "user_intent": "Fix drift on prod web servers",
  "task_spec": {
    "task_type": "drift_remediation",
    "goal": "Remediate drift for production web servers",
    "context": {
      "environment": "production",
      "scope": {
        "platforms": ["aws", "azure"],
        "asset_filter": "role:web-server"
      }
    },
    "agents": [{"name": "drift_agent", "priority": 1}],
    "tools_required": ["query_assets", "compare_versions", "generate_patch_plan"],
    "risk_level": "high",
    "hitl_required": true,
    "constraints": {
      "canary_required": true,
      "max_batch_percent": 10,
      "rollback_trigger": "error_rate > 1%"
    }
  },
  "status": {
    "state": "planned",
    "updated_at": "2024-12-01T10:30:00Z"
  }
}
```

### Plan Example (Drift Remediation)
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "drift_plan",
  "payload": {
    "summary": "Remediate 47 drifted servers in us-east-1 using ubuntu-base-v2.4.1",
    "affected_assets": ["asset-1", "asset-2", "..."],
    "phases": [
      {
        "name": "Canary",
        "assets": ["asset-1", "asset-2", "asset-3", "asset-4", "asset-5"],
        "wait_time": "30m",
        "health_checks": [
          {"type": "http_200", "target": "/health", "timeout": "30s"},
          {"type": "error_rate", "threshold": "< 1%", "timeout": "5m"}
        ],
        "rollback_if": "error_rate > 1%"
      },
      {
        "name": "Wave1",
        "assets": ["asset-6", "...", "asset-20"],
        "wait_time": "15m",
        "health_checks": [
          {"type": "error_rate", "threshold": "< 0.5%", "timeout": "5m"}
        ],
        "rollback_if": "error_rate > 0.5%"
      },
      {
        "name": "Wave2",
        "assets": ["asset-21", "...", "asset-47"],
        "health_checks": [
          {"type": "error_rate", "threshold": "< 0.5%", "timeout": "5m"}
        ]
      }
    ],
    "estimated_duration": "2h",
    "risk_assessment": "Medium - well-tested image, canary phase included"
  },
  "validation": {
    "schema_valid": true,
    "opa_valid": true,
    "safety_valid": true,
    "overall_valid": true
  },
  "quality_score": 85,
  "status": {
    "state": "awaiting_approval",
    "updated_at": "2024-12-01T10:35:00Z"
  }
}
```

## Related Documentation

- [ADR-007: LLM-First Orchestration](../../docs/adr/ADR-007-llm-first-orchestration.md)
- [ADR-008: Task/Plan/Run Lifecycle](../../docs/adr/ADR-008-task-plan-run-lifecycle.md)
- [ADR-009: Tool Risk Taxonomy & HITL](../../docs/adr/ADR-009-tool-risk-taxonomy-hitl.md)
