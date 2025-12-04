# AI Accountability Framework

## Overview

This document establishes QL-RF's framework for treating AI agents as accountable "team members" with defined identities, responsibilities, and answerability chains. Just as human employees have job descriptions, performance reviews, and accountability structures, our AI agents operate under similar governance.

---

## Table of Contents

1. [Philosophy: AI as Team Member](#philosophy-ai-as-team-member)
2. [Agent Identity Model](#agent-identity-model)
3. [Decision Transparency](#decision-transparency)
4. [Answerability Chain](#answerability-chain)
5. [Performance Management](#performance-management)
6. [Incident Ownership](#incident-ownership)
7. [Agent "HR" Lifecycle](#agent-hr-lifecycle)

---

## Philosophy: AI as Team Member

### The Problem with Black-Box AI

Traditional AI implementations suffer from:
- **Opacity**: Decisions are made without explanation
- **Unaccountability**: No clear owner when things go wrong
- **Inconsistency**: Behavior varies unpredictably
- **Untraceable**: No audit trail for compliance

### QL-RF's Approach

We treat each AI agent as a **team member** with:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        AI AGENT = TEAM MEMBER                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  HUMAN EMPLOYEE                      AI AGENT                                │
│  ─────────────────                   ─────────────────                       │
│  Name & Title                   →    Agent Name & Type                       │
│  Job Description                →    Capabilities & Tools                    │
│  Manager/Reports To             →    Answerability Chain                     │
│  Performance Reviews            →    Quality Scores & Metrics                │
│  Incident Ownership             →    Failure Attribution                     │
│  Training & Development         →    Prompt Tuning & Updates                 │
│  Access Controls (Badge)        →    Tool Permissions (RBAC)                 │
│  Audit Trail (Timesheets)       →    Tool Invocation Logs                    │
│  Escalation Path                →    Human-in-the-Loop Triggers              │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Agent Identity Model

### Agent Profile Structure

Each of the 11 agents in QL-RF has a defined identity:

```yaml
# Example: Drift Agent Profile
agent:
  id: drift-agent
  name: Drift Remediation Agent
  version: 1.0.0

  identity:
    role: Infrastructure Drift Specialist
    department: Platform Operations
    reports_to: Human Platform Engineer (approver)

  capabilities:
    primary:
      - Detect configuration drift across fleet
      - Analyze drift severity and root cause
      - Generate remediation plans
      - Execute approved remediations
    secondary:
      - Coordinate with Patch Agent for complex fixes
      - Generate compliance evidence for drift status

  tools_authorized:
    read_only:
      - query_assets
      - get_drift_status
      - get_golden_image
    plan_only:
      - analyze_drift
      - generate_patch_plan
      - simulate_rollout
    state_change:
      - execute_rollout  # Requires HITL approval

  constraints:
    max_assets_per_batch: 20%
    require_canary_phase: true
    auto_rollback_threshold: 5%
    production_approval_required: true

  performance_targets:
    plan_quality_score: ≥80/100
    execution_success_rate: ≥95%
    rollback_frequency: <5%
    mean_time_to_remediate: <60 minutes
```

### Agent Registry

| Agent ID | Name | Primary Domain | Risk Profile |
|----------|------|---------------|--------------|
| `drift-agent` | Drift Remediation Agent | Configuration drift | state_change_prod |
| `patch-agent` | Patch Orchestration Agent | Vulnerability patching | state_change_prod |
| `compliance-agent` | Compliance Assurance Agent | Audit & controls | read_only |
| `incident-agent` | Incident Response Agent | Alert triage | state_change_nonprod |
| `dr-agent` | DR Readiness Agent | Disaster recovery | state_change_prod |
| `cost-agent` | Cost Optimization Agent | Resource efficiency | plan_only |
| `security-agent` | Security Posture Agent | Vulnerability assessment | read_only |
| `image-agent` | Golden Image Agent | Image lifecycle | state_change_prod |
| `sop-agent` | SOP Management Agent | Runbook automation | state_change_prod |
| `adapter-agent` | Integration Adapter Agent | External systems | read_only |
| `base-agent` | Base Agent | Shared functionality | N/A |

---

## Decision Transparency

### Every Decision is Logged

When an agent makes a decision, the following is captured:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        DECISION AUDIT RECORD                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  WHAT was decided?                                                           │
│  ├── Task ID: task_abc123                                                   │
│  ├── Decision: "Remediate 12 servers via package update"                    │
│  └── Plan ID: plan_xyz789                                                   │
│                                                                              │
│  WHO made the decision?                                                      │
│  ├── Agent: drift-agent v1.0.0                                              │
│  ├── Approved by: jane.engineer@company.com                                 │
│  └── Organization: acme-corp                                                │
│                                                                              │
│  WHY was this decision made?                                                 │
│  ├── Intent: "Remediate critical drift on production web servers"           │
│  ├── Context: env=prod, tier=web, drift_severity=critical                   │
│  ├── Agent Reasoning: [Captured from LLM response]                          │
│  │   "Found 12 servers with package version drift exceeding threshold.      │
│  │    Golden image specifies nginx 1.24.0 but servers have 1.22.1.          │
│  │    Recommended action: Patch upgrade with canary deployment."            │
│  └── Quality Score: 87/100                                                  │
│                                                                              │
│  HOW was it executed?                                                        │
│  ├── Tools Invoked:                                                         │
│  │   ├── query_assets(env=prod, drift_status=drifted) → 12 assets          │
│  │   ├── get_drift_status(asset_ids) → drift details                       │
│  │   ├── get_golden_image(family=web-tier) → target spec                   │
│  │   ├── generate_patch_plan(assets, type=upgrade) → plan                  │
│  │   └── simulate_rollout(plan, dry_run=true) → validation passed          │
│  ├── Execution Phases: 3 (canary → 25% → remaining)                        │
│  └── Duration: 23 minutes                                                   │
│                                                                              │
│  WHAT was the outcome?                                                       │
│  ├── Status: completed                                                      │
│  ├── Assets Remediated: 12/12 (100%)                                        │
│  ├── Rollbacks: 0                                                           │
│  └── ServiceNow Change: CHG0012345                                          │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Database Schema for Audit

```sql
-- ai_tasks: High-level task tracking
CREATE TABLE ai_tasks (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL,
    intent TEXT NOT NULL,           -- Original user request
    agent_type VARCHAR(50),         -- Which agent handled it
    status VARCHAR(20),             -- pending, approved, executing, completed, failed
    risk_level VARCHAR(30),         -- read_only, plan_only, state_change_*
    created_by UUID,                -- User who submitted
    approved_by UUID,               -- User who approved (HITL)
    created_at TIMESTAMP,
    approved_at TIMESTAMP,
    completed_at TIMESTAMP
);

-- ai_plans: Generated plans with quality metrics
CREATE TABLE ai_plans (
    id UUID PRIMARY KEY,
    task_id UUID REFERENCES ai_tasks(id),
    plan_json JSONB NOT NULL,       -- Full plan structure
    quality_score INTEGER,          -- 0-100 quality score
    risk_score INTEGER,             -- Calculated risk
    llm_reasoning TEXT,             -- Agent's explanation
    created_at TIMESTAMP
);

-- ai_runs: Execution tracking
CREATE TABLE ai_runs (
    id UUID PRIMARY KEY,
    task_id UUID REFERENCES ai_tasks(id),
    plan_id UUID REFERENCES ai_plans(id),
    status VARCHAR(20),
    phases_completed INTEGER,
    phases_total INTEGER,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error TEXT
);

-- ai_tool_invocations: Every tool call
CREATE TABLE ai_tool_invocations (
    id UUID PRIMARY KEY,
    task_id UUID REFERENCES ai_tasks(id),
    tool_name VARCHAR(100),         -- e.g., query_assets
    tool_category VARCHAR(50),      -- query, analysis, execution
    risk_level VARCHAR(30),
    input_params JSONB,             -- Parameters passed
    output_result JSONB,            -- Result returned
    duration_ms INTEGER,
    invoked_at TIMESTAMP
);
```

### Viewing the Audit Trail

**UI Path**: `/ai/tasks/{taskId}` → "Audit Trail" tab

**API Endpoint**:
```bash
GET /api/v1/ai/tasks/{task_id}/audit
```

Returns complete decision history including:
- Original intent
- Agent selection rationale
- All tool invocations with inputs/outputs
- Plan generation with quality score
- Approval details
- Execution phases and outcomes

---

## Answerability Chain

### Hierarchy of Responsibility

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       ANSWERABILITY CHAIN                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│                         ┌─────────────────┐                                 │
│                         │  ORGANIZATION   │                                 │
│                         │  (Accountable)  │                                 │
│                         └────────┬────────┘                                 │
│                                  │                                          │
│                    ┌─────────────┴─────────────┐                            │
│                    │                           │                            │
│           ┌────────▼────────┐        ┌────────▼────────┐                   │
│           │  POLICY OWNER   │        │   RISK OWNER    │                   │
│           │  (Defines rules)│        │ (Accepts risk)  │                   │
│           └────────┬────────┘        └────────┬────────┘                   │
│                    │                          │                            │
│                    └──────────┬───────────────┘                            │
│                               │                                            │
│                    ┌──────────▼──────────┐                                 │
│                    │   HUMAN APPROVER    │                                 │
│                    │   (Responsible)     │                                 │
│                    │                     │                                 │
│                    │ • Reviews AI plans  │                                 │
│                    │ • Approves/rejects  │                                 │
│                    │ • Monitors execution│                                 │
│                    │ • Owns outcomes     │                                 │
│                    └──────────┬──────────┘                                 │
│                               │                                            │
│                    ┌──────────▼──────────┐                                 │
│                    │     AI AGENT        │                                 │
│                    │   (Executes)        │                                 │
│                    │                     │                                 │
│                    │ • Generates plans   │                                 │
│                    │ • Executes approved │                                 │
│                    │ • Reports status    │                                 │
│                    │ • Logs all actions  │                                 │
│                    └─────────────────────┘                                 │
│                                                                            │
├─────────────────────────────────────────────────────────────────────────────┤
│  KEY PRINCIPLE:                                                             │
│  AI agents PROPOSE, humans APPROVE, organizations are ACCOUNTABLE          │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Responsibility Matrix (RACI)

| Activity | AI Agent | Human Approver | Risk Owner | Organization |
|----------|----------|----------------|------------|--------------|
| Task Submission | - | R | - | - |
| Plan Generation | R | - | - | - |
| Plan Review | I | R | C | - |
| Approval Decision | - | R/A | C | I |
| Execution | R | I | I | - |
| Monitoring | R | R | I | - |
| Incident Response | R | R/A | A | I |
| Post-Incident Review | I | R | R | A |
| Policy Updates | - | C | R | A |

**R** = Responsible, **A** = Accountable, **C** = Consulted, **I** = Informed

### Escalation Paths

```
SCENARIO: AI plan rejected 3 times
  └── Escalate to: Risk Owner
      └── Decision: Review policy constraints or provide guidance

SCENARIO: Execution fails with rollback
  └── Escalate to: Human Approver + Risk Owner
      └── Decision: Investigate root cause, decide retry/abandon

SCENARIO: AI generates invalid/dangerous plan
  └── Escalate to: Security Team + Risk Owner
      └── Decision: Review agent constraints, update policies

SCENARIO: Approval timeout (24h)
  └── Escalate to: Original Submitter + Manager
      └── Decision: Re-submit or cancel

SCENARIO: AI-caused production incident
  └── Escalate to: Incident Commander + CISO + Risk Owner
      └── Decision: Immediate remediation, post-incident review
```

---

## Performance Management

### Quality Scoring (0-100)

Every AI-generated plan receives a quality score based on 5 dimensions:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        QUALITY SCORE BREAKDOWN                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  DIMENSION              WEIGHT    CRITERIA                                   │
│  ─────────────────────  ─────     ──────────────────────────────────────    │
│  Completeness           25%       All required fields present                │
│                                   All affected assets identified             │
│                                   Rollback plan included                     │
│                                                                              │
│  Safety                 25%       OPA policies pass                          │
│                                   Risk level appropriate                     │
│                                   No dangerous operations                    │
│                                                                              │
│  Feasibility            20%       Resources available                        │
│                                   Dependencies satisfied                     │
│                                   Maintenance window viable                  │
│                                                                              │
│  Efficiency             15%       Minimal blast radius                       │
│                                   Optimal batch sizing                       │
│                                   Reasonable duration                        │
│                                                                              │
│  Clarity                15%       Clear description                          │
│                                   Understandable steps                       │
│                                   Explicit success criteria                  │
│                                                                              │
│  TOTAL                  100%                                                 │
│                                                                              │
├─────────────────────────────────────────────────────────────────────────────┤
│  THRESHOLDS                                                                  │
│  ─────────────────────                                                       │
│  Production (state_change_prod):      ≥80 required                          │
│  Non-Production (state_change_nonprod): ≥70 required                        │
│  Plan-Only:                           ≥60 required                          │
│  Read-Only:                           No minimum                            │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Agent Performance Metrics

Track these metrics per agent over time:

| Metric | Description | Target |
|--------|-------------|--------|
| **Plan Quality Score** | Average quality score | ≥85/100 |
| **Approval Rate** | Plans approved vs rejected | ≥90% |
| **Execution Success Rate** | Completed vs failed | ≥95% |
| **Rollback Frequency** | Executions requiring rollback | <5% |
| **Mean Time to Plan** | Time from intent to plan | <60 seconds |
| **Mean Time to Execute** | Time from approval to completion | Varies |
| **Token Efficiency** | Tokens per successful task | Trending down |
| **Human Override Rate** | Plans modified by humans | <20% |

### Performance Dashboard

**UI Path**: `/ai/agents` → Select agent → "Performance" tab

Displays:
- Historical quality scores
- Success/failure trends
- Comparison to targets
- Token consumption
- Most common failure reasons

---

## Incident Ownership

### When AI Causes an Incident

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    AI-CAUSED INCIDENT RESPONSE                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  STEP 1: IMMEDIATE RESPONSE                                                  │
│  ├── Execution automatically paused/rolled back                             │
│  ├── Incident ticket created (ServiceNow INC)                               │
│  ├── Notifications sent to:                                                 │
│  │   ├── Human approver who approved the task                              │
│  │   ├── On-call engineer                                                  │
│  │   └── Risk owner                                                        │
│  └── All related AI tasks paused                                           │
│                                                                              │
│  STEP 2: TRIAGE                                                              │
│  ├── Identify: Which AI task caused the incident?                          │
│  ├── Retrieve: Full audit trail (plan, tools, execution)                   │
│  ├── Assess: Blast radius and severity                                     │
│  └── Decide: Continue rollback or manual intervention                       │
│                                                                              │
│  STEP 3: REMEDIATION                                                         │
│  ├── If rollback sufficient: Let automation complete                        │
│  ├── If manual needed: Human takes over                                     │
│  └── Document all actions in incident ticket                                │
│                                                                              │
│  STEP 4: POST-INCIDENT                                                       │
│  ├── Root cause analysis (RCA)                                              │
│  │   ├── Was the plan flawed?                                              │
│  │   ├── Was execution buggy?                                              │
│  │   ├── Was approval decision wrong?                                      │
│  │   └── Were constraints insufficient?                                    │
│  ├── Attribute responsibility:                                              │
│  │   ├── AI Agent: Update constraints/prompts                              │
│  │   ├── Human Approver: Additional training                               │
│  │   ├── Policy: Update OPA rules                                          │
│  │   └── System: Fix execution engine                                      │
│  └── Update documentation and runbooks                                      │
│                                                                              │
│  STEP 5: PREVENTION                                                          │
│  ├── Add new OPA policy rule if needed                                      │
│  ├── Update agent constraints                                               │
│  ├── Add to integration test suite                                          │
│  └── Share learnings with team                                              │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Incident Classification

| Severity | Description | Example | Response Time |
|----------|-------------|---------|---------------|
| **SEV-1** | Production outage | AI rolled out bad config to entire fleet | Immediate |
| **SEV-2** | Significant impact | AI caused partial service degradation | 15 minutes |
| **SEV-3** | Minor impact | AI task failed, no customer impact | 1 hour |
| **SEV-4** | No impact | AI generated invalid plan (caught before approval) | 24 hours |

---

## Agent "HR" Lifecycle

### Onboarding a New Agent

```
1. DEFINE IDENTITY
   ├── Agent name and version
   ├── Capabilities and tools
   ├── Constraints and guardrails
   └── Performance targets

2. CONFIGURE ACCESS
   ├── Assign tool permissions
   ├── Set risk level
   ├── Configure HITL requirements
   └── Define escalation paths

3. VALIDATE BEHAVIOR
   ├── Run in sandbox mode
   ├── Execute test scenarios
   ├── Verify OPA policies apply
   └── Confirm audit logging

4. GRADUAL ROLLOUT
   ├── Enable for non-production first
   ├── Monitor quality scores
   ├── Gather human feedback
   └── Promote to production

5. CONTINUOUS MONITORING
   ├── Track performance metrics
   ├── Review rejection reasons
   ├── Update prompts as needed
   └── Regular performance reviews
```

### Agent "Performance Review" (Quarterly)

```yaml
agent_review:
  agent_id: drift-agent
  review_period: Q4-2025

  metrics:
    tasks_completed: 342
    approval_rate: 94%
    execution_success_rate: 97%
    rollback_rate: 2.1%
    avg_quality_score: 86

  incidents:
    sev1: 0
    sev2: 1  # Partial rollout failure on 2025-11-15
    sev3: 3
    sev4: 8

  improvements_made:
    - Added constraint: max 20% fleet per batch
    - Updated prompt for better canary sizing
    - Added new health check validation

  recommendations:
    - Consider adding rollback dry-run capability
    - Improve handling of mixed OS environments
    - Add support for container workloads
```

### Retiring/Deprecating an Agent

```
1. ANNOUNCE DEPRECATION
   ├── Notify users of timeline
   ├── Document replacement (if any)
   └── Provide migration path

2. DISABLE NEW TASKS
   ├── Agent stops accepting new intents
   ├── Redirect to replacement agent
   └── Log all redirect attempts

3. COMPLETE IN-FLIGHT
   ├── Allow executing tasks to complete
   ├── Monitor for issues
   └── Extend timeout if needed

4. ARCHIVE DATA
   ├── Export agent performance history
   ├── Archive audit logs
   └── Retain per policy (7 years)

5. REMOVE FROM REGISTRY
   ├── Delete agent configuration
   ├── Update documentation
   └── Remove from UI
```

---

## Summary

The AI Accountability Framework ensures:

1. **Transparency**: Every AI decision is logged and explainable
2. **Answerability**: Clear chain of responsibility from AI to humans to organization
3. **Performance**: Measurable quality scores and success metrics
4. **Governance**: Consistent policies enforced via OPA
5. **Improvement**: Continuous monitoring and iterative enhancement

This framework enables organizations to adopt AI automation confidently, knowing that:
- AI actions are traceable
- Humans remain in control
- Incidents can be investigated
- Performance can be measured
- Accountability is clear
