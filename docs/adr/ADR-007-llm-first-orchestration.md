# ADR-007: LLM-First Orchestration Architecture

## Status
Accepted

## Context
QL-RF currently provides a dashboard-centric experience where humans:
1. View drift/compliance/DR status on dashboards
2. Analyze data manually to identify issues
3. Decide on remediation actions
4. Execute actions through manual processes or scripts

This model scales poorly:
- Ops teams are overwhelmed by alert volume
- Context-switching between dashboards fragments understanding
- Remediation knowledge lives in tribal documentation
- Compliance evidence preparation is labor-intensive

### Relationship to Existing ADRs
This ADR extends (not replaces) existing architecture decisions:
- **ADR-001 (Contracts-First)**: LLM agents read/generate contracts; contracts remain the source of truth
- **ADR-002 (Agentless)**: Infrastructure remains agentless; "agents" here are LLM reasoning modules, not deployed software
- **ADR-004 (Temporal)**: LLM orchestration integrates with Temporal for durable execution of AI-generated plans
- **ADR-005 (OPA)**: OPA validates LLM outputs before execution; LLM augments but doesn't replace policy engine

## Decision
We adopt an **LLM-First Orchestration** architecture with four layers:

```
┌─────────────────────────────────────────────────────────────────┐
│                     AI ORCHESTRATOR SERVICE                      │
├─────────────────────────────────────────────────────────────────┤
│  LAYER 1: META-PROMPT ENGINE                                     │
│  User Intent → TaskSpec (structured task definition)             │
├─────────────────────────────────────────────────────────────────┤
│  LAYER 2: SPECIALIST AGENTS                                      │
│  DriftAgent │ PatchAgent │ ComplianceAgent │ DRAgent │ ...       │
│  (LLM reasoning modules with domain expertise)                   │
├─────────────────────────────────────────────────────────────────┤
│  LAYER 3: TOOL REGISTRY                                          │
│  query_assets │ compare_versions │ generate_plan │ execute_*     │
│  (Deterministic functions the LLM can invoke)                    │
├─────────────────────────────────────────────────────────────────┤
│  LAYER 4: VALIDATION PIPELINE                                    │
│  JSONSchema → OPA Policies → Safety Checks → HITL Gates          │
│  (Guardrails before any execution)                               │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│              EXISTING QL-RF SERVICES (unchanged)                 │
│   API │ Connectors │ Drift Engine │ Compliance │ Temporal        │
└─────────────────────────────────────────────────────────────────┘
```

### Core Principles

1. **LLM as Planner, Tools as Executors**
   - LLM generates plans and artifacts (never executes directly)
   - Tools perform actual operations (read-only or state-changing)
   - All state changes require explicit tool invocation

2. **Human-in-the-Loop (HITL) by Default**
   - Any state-changing operation requires human approval
   - Risk levels determine approval requirements
   - Tenants can configure autonomy levels

3. **Validation Before Execution**
   - Every LLM output passes through validation pipeline
   - OPA policies enforce safety invariants
   - Schema validation ensures structural correctness

4. **Separation of Concerns**
   - Meta-prompt engine: Intent understanding
   - Agents: Domain reasoning
   - Tools: Operations
   - Validation: Safety

### Eight Specialist Agents

| Agent | Domain | Primary Tools | Output |
|-------|--------|---------------|--------|
| DriftAgent | Configuration drift | query_assets, compare_versions, generate_patch_plan | DriftRemediationPlan |
| PatchAgent | Patch orchestration | get_cve_feed, calculate_risk, generate_rollout | PatchRolloutPlan |
| ComplianceAgent | Audit & evidence | check_controls, generate_evidence | ComplianceReport |
| IncidentAgent | RCA & resolution | query_logs, correlate_events | IncidentAnalysis |
| DRAgent | DR planning & drills | check_rto, simulate_failover | DRRunbook |
| CostAgent | Cost optimization | get_billing_data, recommend | CostOptimizationPlan |
| SecurityAgent | Vulnerability mgmt | scan_vulns, check_exposure | SecurityReport |
| ImageAgent | Image lifecycle | build_image, validate, promote | ImageSpec |

### Tool Categories

```go
// Risk taxonomy for tools
type ToolRisk string
const (
    ToolRiskReadOnly        ToolRisk = "read_only"        // No approval needed
    ToolRiskPlanOnly        ToolRisk = "plan_only"        // Generates artifacts, no execution
    ToolRiskStateChangeProd ToolRisk = "state_change_prod" // Always requires HITL
)
```

### LLM Provider Strategy
- Primary: Claude (Anthropic) via API
- Fallback: Azure OpenAI for enterprise deployments
- Model selection per task type (fast models for parsing, capable models for reasoning)

## Consequences

### Positive
- **10x faster operations**: Drift resolution from hours to minutes
- **Consistent quality**: Encoded best practices in agent prompts
- **Reduced cognitive load**: AI handles analysis, human approves actions
- **Audit trail**: All AI decisions logged and traceable
- **Scalable expertise**: Domain knowledge encoded in agents, not tribal docs
- **Compliance automation**: Evidence generation in hours, not weeks

### Negative
- LLM cost per operation (token usage)
- Latency for LLM calls (seconds, not milliseconds)
- Risk of LLM hallucination affecting plans
- Dependency on external LLM providers
- Complexity in prompt engineering and maintenance

### Mitigations
- **Cost**: Cache common queries, use smaller models for simple tasks
- **Latency**: Async processing, streaming responses for UX
- **Hallucination**: Multi-layer validation pipeline, OPA guardrails
- **Provider dependency**: Abstract LLM client, support multiple providers
- **Prompt complexity**: Version prompts, offline evaluation harness

## Implementation Notes

### Integration with Temporal (ADR-004)
LLM orchestration runs as Temporal workflows:
```go
func AITaskWorkflow(ctx workflow.Context, task TaskSpec) error {
    // Activity 1: Agent reasoning (LLM call)
    var plan Plan
    err := workflow.ExecuteActivity(ctx, AgentReasoningActivity, task).Get(ctx, &plan)

    // Activity 2: Validation (OPA + safety)
    err = workflow.ExecuteActivity(ctx, ValidatePlanActivity, plan).Get(ctx, nil)

    // Activity 3: Wait for HITL approval (signal)
    workflow.GetSignalChannel(ctx, "approval").Receive(ctx, &approval)

    // Activity 4: Execute approved plan
    return workflow.ExecuteActivity(ctx, ExecutePlanActivity, plan, approval).Get(ctx, nil)
}
```

### Integration with OPA (ADR-005)
OPA validates all AI-generated plans:
```rego
package ql.ai.safety

# Deny production changes without canary
deny[msg] {
    input.plan.environment == "production"
    not has_canary_phase(input.plan)
    msg := "Production changes require canary phase"
}

# Deny batch size > 20% for production
deny[msg] {
    input.plan.environment == "production"
    some phase in input.plan.phases
    phase.batch_percent > 20
    msg := sprintf("Batch %s exceeds 20%% limit", [phase.name])
}
```

### Telemetry Requirements
Every AI task must log:
- task_id, user, org_id
- TaskSpec (sanitized)
- Agent invocations, tool calls
- LLM latency, token usage
- Validation results
- HITL decisions
- Execution outcomes

## References
- [Anthropic Claude Documentation](https://docs.anthropic.com/)
- [OpenAI Function Calling](https://platform.openai.com/docs/guides/function-calling)
- [Temporal Go SDK](https://docs.temporal.io/dev-guide/go)
- ADR-001, ADR-002, ADR-004, ADR-005
