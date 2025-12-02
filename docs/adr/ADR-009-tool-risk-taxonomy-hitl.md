# ADR-009: Tool Risk Taxonomy & HITL Policy

## Status
Accepted

## Context
ADR-007 establishes that LLM agents invoke tools to perform operations. These tools range from harmless queries to production-impacting state changes. Without clear risk categorization:
- Dangerous operations could execute without approval
- Safe operations could be blocked by excessive approval requirements
- Tenants cannot configure autonomy levels appropriately
- Audit requirements are unclear

We need:
1. Standardized risk taxonomy for all tools
2. Clear HITL (Human-in-the-Loop) policies per risk level
3. Tenant-configurable autonomy modes
4. Simulation-first patterns for dangerous operations

## Decision

### Tool Risk Taxonomy

Every tool declares three properties:

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() JSONSchema

    // Risk classification
    Risk() ToolRisk           // read_only | plan_only | state_change_nonprod | state_change_prod
    Idempotent() bool         // Can be safely retried
    Scope() ToolScope         // asset | environment | organization

    Execute(ctx context.Context, params map[string]any) (any, error)
}

type ToolRisk string
const (
    RiskReadOnly            ToolRisk = "read_only"             // No side effects
    RiskPlanOnly            ToolRisk = "plan_only"             // Generates artifacts, no execution
    RiskStateChangeNonProd  ToolRisk = "state_change_nonprod"  // Modifies non-production
    RiskStateChangeProd     ToolRisk = "state_change_prod"     // Modifies production
)

type ToolScope string
const (
    ScopeAsset       ToolScope = "asset"        // Single resource
    ScopeEnvironment ToolScope = "environment"  // Environment-wide
    ScopeOrganization ToolScope = "organization" // Org-wide
)
```

### Tool Registry by Risk Level

| Risk Level | Example Tools | HITL Required | Logging |
|------------|---------------|---------------|---------|
| **read_only** | query_assets, get_drift_status, get_compliance | Never | Basic |
| **plan_only** | generate_patch_plan, generate_dr_runbook, compare_versions | Never | Detailed |
| **state_change_nonprod** | execute_rollout (staging), trigger_dr_drill (sandbox) | Configurable | Full audit |
| **state_change_prod** | execute_rollout (prod), acknowledge_alert, modify_firewall | Always | Full audit + evidence |

### Full Tool Catalog

```yaml
# Read-Only Tools (no approval, minimal logging)
tools:
  query_assets:
    risk: read_only
    idempotent: true
    scope: organization
    description: "Query assets with filters"

  get_drift_status:
    risk: read_only
    idempotent: true
    scope: environment
    description: "Get drift analysis for assets"

  get_compliance_status:
    risk: read_only
    idempotent: true
    scope: environment
    description: "Get compliance posture"

  get_golden_image:
    risk: read_only
    idempotent: true
    scope: organization
    description: "Get current golden image for a family"

  query_alerts:
    risk: read_only
    idempotent: true
    scope: environment
    description: "Query active alerts"

  get_dr_status:
    risk: read_only
    idempotent: true
    scope: environment
    description: "Get DR readiness status"

# Plan-Only Tools (generates artifacts, no execution)
  compare_versions:
    risk: plan_only
    idempotent: true
    scope: asset
    description: "Compare current vs target versions"

  generate_patch_plan:
    risk: plan_only
    idempotent: true
    scope: environment
    description: "Generate phased patch rollout plan"

  generate_rollout_plan:
    risk: plan_only
    idempotent: true
    scope: environment
    description: "Generate rollout strategy with canary"

  generate_dr_runbook:
    risk: plan_only
    idempotent: true
    scope: environment
    description: "Generate DR runbook from infrastructure"

  generate_compliance_evidence:
    risk: plan_only
    idempotent: true
    scope: organization
    description: "Generate compliance evidence pack"

  simulate_rollout:
    risk: plan_only
    idempotent: true
    scope: environment
    description: "Dry-run rollout, predict impact"

  simulate_failover:
    risk: plan_only
    idempotent: true
    scope: environment
    description: "Simulate DR failover, predict RTO"

  calculate_risk_score:
    risk: plan_only
    idempotent: true
    scope: asset
    description: "Calculate risk score for change"

# State-Changing (Non-Prod)
  execute_rollout_nonprod:
    risk: state_change_nonprod
    idempotent: false
    scope: environment
    description: "Execute rollout in non-prod environment"
    requires_plan: true  # Must reference approved Plan

  trigger_dr_drill_sandbox:
    risk: state_change_nonprod
    idempotent: false
    scope: environment
    description: "Trigger DR drill in sandbox"
    requires_plan: true

# State-Changing (Production) - Always requires HITL
  execute_rollout_prod:
    risk: state_change_prod
    idempotent: false
    scope: environment
    description: "Execute rollout in production"
    requires_plan: true
    requires_hitl: always

  acknowledge_alert:
    risk: state_change_prod
    idempotent: true
    scope: asset
    description: "Acknowledge and close alert"
    requires_hitl: always

  trigger_dr_failover:
    risk: state_change_prod
    idempotent: false
    scope: environment
    description: "Trigger actual DR failover"
    requires_plan: true
    requires_hitl: always
```

### HITL Policy Matrix

| Risk Level | Autonomy: plan_only | Autonomy: canary_only | Autonomy: full_auto |
|------------|---------------------|----------------------|---------------------|
| read_only | Execute | Execute | Execute |
| plan_only | Execute | Execute | Execute |
| state_change_nonprod | Block | Execute | Execute |
| state_change_prod | Block | Canary only | Execute with approval |

### Tenant Autonomy Modes

Tenants configure their autonomy level:

```yaml
# Organization settings
autonomy:
  mode: plan_only | canary_only | full_auto

  # Per-environment overrides
  environments:
    production:
      mode: plan_only  # Never auto-execute in prod
      allowed_approvers: [PlatformAdmin, SRELead]
      require_two_approvers: true  # For critical risk

    staging:
      mode: canary_only
      allowed_approvers: [PlatformAdmin, SRELead, DevOpsEngineer]

    development:
      mode: full_auto  # Auto-execute approved plans
      allowed_approvers: [any]

  # Tool-specific overrides
  tool_overrides:
    trigger_dr_failover:
      mode: plan_only  # Always block, even in full_auto
      require_two_approvers: true
```

### Simulation-First Pattern

For every dangerous tool, provide a simulation variant:

```go
// Simulation tool (safe)
type SimulateRolloutTool struct{}
func (t *SimulateRolloutTool) Risk() ToolRisk { return RiskPlanOnly }
func (t *SimulateRolloutTool) Execute(ctx context.Context, params map[string]any) (any, error) {
    // Returns predicted impact without executing
    return &RolloutSimulation{
        AffectedAssets: 47,
        EstimatedDuration: "2h",
        PredictedRisk: "medium",
        HealthCheckPoints: [...],
    }, nil
}

// Execution tool (dangerous)
type ExecuteRolloutTool struct{}
func (t *ExecuteRolloutTool) Risk() ToolRisk { return RiskStateChangeProd }
func (t *ExecuteRolloutTool) Execute(ctx context.Context, params map[string]any) (any, error) {
    // Actually executes the rollout
    // Must have approved Plan reference
    // Must pass HITL gate
}
```

Agents MUST call simulate_* before proposing execute_*:

```
Agent Flow:
1. query_assets → Get affected assets
2. generate_rollout_plan → Create plan
3. simulate_rollout → Predict impact  ← REQUIRED before execution
4. [HITL approval]
5. execute_rollout → Actually execute
```

### HITL Approval Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                     PLAN APPROVAL FLOW                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Plan Generated                                                  │
│       │                                                          │
│       ▼                                                          │
│  ┌─────────────┐                                                │
│  │  Validate   │ ← Schema + OPA + Safety                        │
│  └──────┬──────┘                                                │
│         │                                                        │
│         ▼                                                        │
│  ┌─────────────┐     No HITL needed                             │
│  │ Check HITL  │────────────────────→ Execute                   │
│  │  Required   │                                                │
│  └──────┬──────┘                                                │
│         │ HITL required                                          │
│         ▼                                                        │
│  ┌─────────────┐                                                │
│  │   Notify    │ ← Slack/Email/UI notification                  │
│  │  Approvers  │                                                │
│  └──────┬──────┘                                                │
│         │                                                        │
│         ▼                                                        │
│  ┌─────────────┐                                                │
│  │   Await     │ ← timeout_minutes from TaskSpec                │
│  │  Decision   │                                                │
│  └──────┬──────┘                                                │
│         │                                                        │
│    ┌────┴────┬────────────┐                                     │
│    ▼         ▼            ▼                                     │
│ Approved  Modified     Rejected                                 │
│    │         │            │                                     │
│    ▼         ▼            ▼                                     │
│ Execute   Re-plan      Archive                                  │
│    │         │                                                  │
│    ▼         │                                                  │
│  Log to   ◄──┘                                                  │
│  Audit                                                          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### OPA Policies for Tool Execution

```rego
package ql.ai.tools

import future.keywords.in

# Deny state_change_prod without approved plan
deny[msg] {
    input.tool.risk == "state_change_prod"
    not input.plan.state == "approved"
    msg := sprintf("Tool %s requires approved plan", [input.tool.name])
}

# Deny state_change_prod in plan_only mode
deny[msg] {
    input.tool.risk == "state_change_prod"
    input.autonomy.mode == "plan_only"
    msg := "Production state changes blocked in plan_only mode"
}

# Deny execution without simulation
deny[msg] {
    input.tool.name == "execute_rollout_prod"
    not simulation_completed(input.task_id)
    msg := "Must run simulate_rollout before execute_rollout_prod"
}

# Require two approvers for critical tools
deny[msg] {
    input.tool.name in ["trigger_dr_failover", "execute_rollout_prod"]
    input.autonomy.require_two_approvers
    count(input.approvals) < 2
    msg := sprintf("Tool %s requires two approvers", [input.tool.name])
}

# Check approver role
deny[msg] {
    input.tool.risk in ["state_change_prod", "state_change_nonprod"]
    not approver_allowed(input.approver, input.autonomy.allowed_approvers)
    msg := sprintf("User %s not authorized to approve", [input.approver])
}

approver_allowed(user, allowed) {
    "any" in allowed
}

approver_allowed(user, allowed) {
    some role in user.roles
    role in allowed
}
```

## Consequences

### Positive
- **Safety by default**: Production changes always require approval
- **Flexibility**: Tenants can configure autonomy per environment
- **Auditability**: Full trail of who approved what
- **Predictability**: Simulation-first ensures no surprises
- **Defense in depth**: OPA + HITL + Safety checks

### Negative
- Approval latency for urgent operations
- Configuration complexity for multi-environment setups
- Need to maintain simulation accuracy

### Mitigations
- **Latency**: Slack/mobile push for approvals, short timeouts
- **Complexity**: Sensible defaults, UI for configuration
- **Simulation accuracy**: Continuous validation against actual outcomes

## Emergency Override

For true emergencies, provide break-glass mechanism:

```yaml
emergency_override:
  enabled: true
  requires_mfa: true
  requires_justification: true
  audit_level: critical
  auto_creates_incident: true
  notifies: [security-team, platform-leads]
```

When invoked:
1. User provides MFA + written justification
2. Operation executes immediately
3. Incident auto-created for post-mortem
4. Security team notified
5. Full audit trail preserved

## References
- ADR-007: LLM-First Orchestration Architecture
- ADR-008: Task/Plan/Run Lifecycle
- ADR-005: OPA as Policy Engine
- [NIST SP 800-53: AC-6 Least Privilege](https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final)
