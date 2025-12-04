# QL-RF User Journeys

## Overview

This document describes the end-to-end user journeys for interacting with the QL-RF AI Copilot, comparing manual processes with AI-enhanced workflows.

---

## Table of Contents

1. [Personas](#personas)
2. [Journey 1: Drift Detection & Remediation](#journey-1-drift-detection--remediation)
3. [Journey 2: Patch Rollout Planning](#journey-2-patch-rollout-planning)
4. [Journey 3: Compliance Evidence Generation](#journey-3-compliance-evidence-generation)
5. [Journey 4: DR Drill Execution](#journey-4-dr-drill-execution)
6. [Journey 5: Incident Response](#journey-5-incident-response)
7. [Journey 6: Golden Image Management](#journey-6-golden-image-management)
8. [Cross-Journey: Task Approval Workflow](#cross-journey-task-approval-workflow)

---

## Personas

| Persona | Role | Primary Goals | AI Interaction Level |
|---------|------|---------------|---------------------|
| **Platform Engineer** | Day-to-day operations | Reduce toil, automate repetitive tasks | Heavy - submits intents, monitors execution |
| **SRE/DevOps Lead** | Reliability & uptime | Faster MTTR, fewer incidents | Moderate - reviews plans, approves changes |
| **Compliance Officer** | Audit & governance | Evidence collection, control validation | Light - consumes reports, validates controls |
| **IT Manager** | Team oversight | Resource optimization, cost control | Dashboard consumer, approves high-risk tasks |
| **CISO/Security Lead** | Security posture | Vulnerability remediation, compliance | Reviews security-related plans |
| **CTO/VP Engineering** | Strategic oversight | ROI, platform adoption, risk reduction | Executive dashboards, policy approval |

---

## Journey 1: Drift Detection & Remediation

### The Problem
Infrastructure drift occurs when deployed assets diverge from their golden image specifications. This creates security vulnerabilities, compliance gaps, and unpredictable behavior.

### Manual Process (Before QL-RF)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        MANUAL DRIFT REMEDIATION                              │
│                         Total Time: 4-8 hours                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. DETECTION (60-120 min)                                                   │
│     ├── Run configuration management scan (Chef/Puppet/Ansible)             │
│     ├── Export scan results to spreadsheet                                  │
│     ├── Cross-reference with CMDB                                           │
│     └── Manually identify drifted assets                                    │
│                                                                              │
│  2. ANALYSIS (60-90 min)                                                     │
│     ├── SSH into each drifted server                                        │
│     ├── Compare packages/configs manually                                   │
│     ├── Document differences in ticket                                      │
│     └── Assess risk level per asset                                         │
│                                                                              │
│  3. PLANNING (45-60 min)                                                     │
│     ├── Create remediation runbook                                          │
│     ├── Schedule maintenance window                                         │
│     ├── Notify stakeholders via email                                       │
│     └── Create change request in ServiceNow                                 │
│                                                                              │
│  4. APPROVAL (2-24 hours)                                                    │
│     ├── Wait for CAB meeting (weekly)                                       │
│     ├── Present change to board                                             │
│     └── Get sign-off from multiple stakeholders                             │
│                                                                              │
│  5. EXECUTION (60-120 min)                                                   │
│     ├── SSH into servers sequentially                                       │
│     ├── Run remediation commands                                            │
│     ├── Verify each server manually                                         │
│     └── Document completion in ticket                                       │
│                                                                              │
│  6. VALIDATION (30-45 min)                                                   │
│     ├── Re-run configuration scan                                           │
│     ├── Verify drift resolved                                               │
│     └── Close ticket and change request                                     │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

Pain Points:
- Multiple tools with no integration
- Manual data gathering and correlation
- Slow approval process (CAB meetings)
- Sequential execution (one server at a time)
- High risk of human error
- No rollback automation
- Tribal knowledge required
```

### AI-Enhanced Process (With QL-RF)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      AI-ENHANCED DRIFT REMEDIATION                           │
│                         Total Time: 15-45 minutes                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  USER INPUT (30 seconds)                                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  "Show me all production servers with critical drift and create     │    │
│  │   a remediation plan for the web tier"                              │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                           │                                                  │
│                           ▼                                                  │
│  AI PROCESSING (2-3 minutes)                                                │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  Meta-Prompt Engine                                                  │    │
│  │  ├── Parse intent → TaskSpec                                        │    │
│  │  ├── Select agent: Drift Agent                                      │    │
│  │  ├── Identify tools: query_assets, get_drift_status, get_golden_image│   │
│  │  └── Assess risk: state_change_prod → requires approval             │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                           │                                                  │
│                           ▼                                                  │
│  AGENT EXECUTION (1-2 minutes)                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  Drift Agent                                                         │    │
│  │  ├── query_assets(env=prod, drift_status=drifted)                   │    │
│  │  ├── get_drift_status(asset_ids=[...])                              │    │
│  │  ├── get_golden_image(family=web-tier)                              │    │
│  │  ├── analyze_drift(compare golden vs actual)                        │    │
│  │  ├── generate_patch_plan(assets, remediation_type=patch)            │    │
│  │  └── simulate_rollout(plan, dry_run=true)                           │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                           │                                                  │
│                           ▼                                                  │
│  PLAN PRESENTATION (Instant)                                                │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  Generated Plan:                                                     │    │
│  │  • 12 servers identified with critical drift                        │    │
│  │  • Drift types: 8 package versions, 4 config changes                │    │
│  │  • Risk score: 72/100 (requires approval)                           │    │
│  │  • Rollout strategy: 3 phases (canary → 25% → remaining)            │    │
│  │  • Estimated duration: 25 minutes                                   │    │
│  │  • Rollback plan: Automatic on >5% failure                          │    │
│  │                                                                      │    │
│  │  [Approve] [Reject] [Modify Plan]                                   │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                           │                                                  │
│                           ▼                                                  │
│  HUMAN APPROVAL (1-5 minutes)                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  Engineer reviews plan:                                              │    │
│  │  ✓ Affected assets look correct                                     │    │
│  │  ✓ Phased rollout is appropriate                                    │    │
│  │  ✓ Rollback threshold acceptable                                    │    │
│  │  → Clicks [Approve]                                                 │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                           │                                                  │
│                           ▼                                                  │
│  AUTOMATED EXECUTION (10-25 minutes)                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  Execution Engine (via Temporal)                                     │    │
│  │                                                                      │    │
│  │  Phase 1: Canary (2 servers)                                        │    │
│  │  ├── Execute remediation                                            │    │
│  │  ├── Run health checks                                              │    │
│  │  ├── Wait 5 minutes                                                 │    │
│  │  └── ✓ Pass → Continue                                              │    │
│  │                                                                      │    │
│  │  Phase 2: 25% (3 servers)                                           │    │
│  │  ├── Execute in parallel                                            │    │
│  │  ├── Health check all                                               │    │
│  │  └── ✓ Pass → Continue                                              │    │
│  │                                                                      │    │
│  │  Phase 3: Remaining (7 servers)                                     │    │
│  │  ├── Execute in parallel                                            │    │
│  │  └── ✓ All complete                                                 │    │
│  │                                                                      │    │
│  │  Real-time progress visible in UI                                   │    │
│  │  [Pause] [Cancel] controls available                                │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                           │                                                  │
│                           ▼                                                  │
│  COMPLETION & AUDIT (Automatic)                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  • ServiceNow change request auto-created and closed                │    │
│  │  • Full audit trail in ai_tasks, ai_plans, ai_runs tables           │    │
│  │  • Slack notification sent to team                                  │    │
│  │  • Compliance evidence generated                                    │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Comparison Summary

| Metric | Manual Process | AI-Enhanced | Improvement |
|--------|---------------|-------------|-------------|
| **Total Time** | 4-8 hours | 15-45 minutes | **90% faster** |
| **Human Effort** | 3-4 hours active work | 5-10 minutes | **95% reduction** |
| **Error Rate** | 5-10% (manual mistakes) | <1% (automated) | **90% reduction** |
| **Rollback Time** | 30-60 minutes | Automatic, <2 min | **95% faster** |
| **Documentation** | Manual ticket updates | Auto-generated | **100% automated** |
| **Approval Cycle** | Days (CAB) | Minutes (async) | **99% faster** |

### API Endpoints Used

```bash
# Submit drift remediation intent
POST /api/v1/ai/execute
{
  "intent": "Remediate critical drift on production web servers",
  "context": {
    "environment": "prod",
    "tier": "web"
  }
}

# View generated plan
GET /api/v1/ai/tasks/{task_id}

# Approve the plan
POST /api/v1/ai/tasks/{task_id}/approve
{
  "approver_notes": "Verified affected assets, proceeding with remediation"
}

# Monitor execution
GET /api/v1/ai/tasks/{task_id}/executions
GET /api/v1/ai/executions/{execution_id}

# Control execution
POST /api/v1/ai/executions/{execution_id}/pause
POST /api/v1/ai/executions/{execution_id}/resume
POST /api/v1/ai/executions/{execution_id}/cancel
```

### UI Screens

1. **AI Copilot Chat** (`/ai`) - Submit natural language intent
2. **Task Detail** (`/ai/tasks/{id}`) - View plan, approve/reject
3. **Execution Progress** - Real-time phase progress with asset status
4. **Audit Trail** - Complete history of AI decisions and tool invocations

---

## Journey 2: Patch Rollout Planning

### The Problem
Rolling out patches across a large fleet requires careful planning to avoid outages while meeting compliance deadlines.

### Manual Process (Before QL-RF)

```
Time: 2-4 days

1. Identify vulnerable assets (security scan exports)
2. Manually group by criticality, environment, dependencies
3. Create rollout plan in spreadsheet
4. Calculate maintenance windows
5. Draft communication to stakeholders
6. Submit change request
7. Wait for approval
8. Execute manually per group
9. Validate and document
```

### AI-Enhanced Process (With QL-RF)

```
Time: 30-60 minutes

User: "Create a patch rollout plan for CVE-2024-1234 across all affected
       production systems, prioritizing by criticality"

AI Agent (Patch Agent):
├── query_assets(cve=CVE-2024-1234, env=prod)
├── calculate_risk_score(assets, cve_severity=critical)
├── generate_rollout_plan(
│     strategy=phased,
│     phases=[canary, 10%, 25%, remaining],
│     maintenance_window=auto_detect
│   )
├── simulate_rollout(plan, check_dependencies=true)
└── generate_compliance_evidence(cve, plan)

Output:
• 47 affected assets identified
• Risk-ranked rollout in 4 phases
• Auto-detected low-traffic maintenance windows
• Dependency-aware sequencing
• Pre-generated compliance evidence

[Approve] [Modify] [Reject]
```

### Comparison

| Metric | Manual | AI-Enhanced | Improvement |
|--------|--------|-------------|-------------|
| Planning Time | 4-8 hours | 5 minutes | **98% faster** |
| Total Cycle | 2-4 days | 30-60 min | **95% faster** |
| Risk Assessment | Subjective | Algorithmic | **Consistent** |
| Compliance Evidence | Manual | Auto-generated | **100% automated** |

---

## Journey 3: Compliance Evidence Generation

### The Problem
Auditors require evidence of control effectiveness. Gathering this evidence is time-consuming and error-prone.

### Manual Process (Before QL-RF)

```
Time: 1-2 days per audit

1. Receive audit request list
2. Export data from multiple systems (CMDB, monitoring, ticketing)
3. Screenshot dashboards
4. Compile evidence in folders
5. Cross-reference with control framework
6. Write narrative explanations
7. Submit to auditor
8. Answer follow-up questions
9. Provide additional evidence
```

### AI-Enhanced Process (With QL-RF)

```
Time: 5-15 minutes

User: "Generate SOC 2 evidence package for CC6.1 (logical access controls)
       for Q4 2024"

AI Agent (Compliance Agent):
├── get_compliance_status(framework=soc2, control=CC6.1)
├── query_assets(timeframe=Q4-2024)
├── check_control(control=CC6.1, assets=all)
├── generate_compliance_evidence(
│     framework=soc2,
│     control=CC6.1,
│     period=Q4-2024,
│     include_narratives=true
│   )
└── export_report(format=pdf)

Output:
• 156-page evidence package
• Control effectiveness summary
• Exception list with remediation status
• Automated narratives
• Supporting screenshots/data exports

[Download PDF] [Export ZIP] [Share with Auditor]
```

### PDF Export Feature
The compliance page (`/compliance`) includes a PDF export feature that generates professional audit-ready reports with:
- Framework compliance summary
- Control-by-control status
- Failing controls with details
- Image compliance matrix
- Timestamp and generation metadata

---

## Journey 4: DR Drill Execution

### The Problem
DR drills are required for compliance but are disruptive, risky, and require significant coordination.

### Manual Process (Before QL-RF)

```
Time: 1-2 weeks preparation, 4-8 hours execution

1. Schedule drill with all stakeholders
2. Document runbook (often outdated)
3. Coordinate with multiple teams
4. Execute failover steps manually
5. Measure RTO/RPO manually
6. Document results
7. Conduct post-mortem
8. Update runbooks
```

### AI-Enhanced Process (With QL-RF)

```
Time: 30 minutes preparation, 1-2 hours execution

User: "Execute DR drill for the payment processing system, measuring
       actual RTO/RPO against our 4-hour/15-minute targets"

AI Agent (DR Agent):
├── get_dr_status(system=payment-processing)
├── generate_dr_runbook(system, include_validation=true)
├── simulate_failover(runbook, dry_run=true)
├── [After approval] execute_failover(runbook)
├── measure_rto_rpo(start_time, recovery_time)
└── generate_dr_evidence(drill_results)

Real-time during drill:
• Live RTO countdown
• Step-by-step execution with validation
• Automatic rollback if issues detected
• Real-time Slack updates to stakeholders

Post-drill:
• Automated DR drill report
• RTO/RPO measurements vs targets
• Identified gaps and recommendations
• Compliance evidence package
```

---

## Journey 5: Incident Response

### The Problem
During incidents, engineers waste time gathering context and coordinating response instead of fixing issues.

### Manual Process (Before QL-RF)

```
During outage (high stress):

1. Get paged
2. Login to multiple systems
3. Manually correlate alerts
4. Identify affected assets
5. Check recent changes
6. Hypothesize root cause
7. Coordinate via Slack/calls
8. Execute remediation manually
9. Verify resolution
10. Write post-mortem
```

### AI-Enhanced Process (With QL-RF)

```
During outage:

User: "We're seeing errors on the checkout service. Help me investigate
       and suggest remediation"

AI Agent (Incident Agent):
├── query_alerts(service=checkout, status=firing)
├── query_assets(service=checkout)
├── get_drift_status(assets)  // Check for recent drift
├── analyze_drift(correlation with incident start)
├── get_recent_changes(service, timeframe=24h)
└── generate_remediation_options(ranked by confidence)

Output:
• 3 servers showing critical drift (package downgrade detected)
• Drift started 2 hours ago (correlates with incident)
• Recent change: Image rollback at 14:32 UTC
• Recommended action: Roll forward to latest golden image

[Approve Remediation] [Investigate Further]

Post-incident:
• Auto-generated timeline
• Root cause analysis
• Remediation steps taken
• Prevention recommendations
```

---

## Journey 6: Golden Image Management

### The Problem
Managing golden images across multiple platforms (AWS, Azure, GCP, vSphere) requires maintaining consistency and tracking versions.

### Manual Process (Before QL-RF)

```
Time: 2-4 hours per image update

1. Build image locally or in CI/CD
2. Test on each platform separately
3. Manually update CMDB with new versions
4. Notify teams via email/Slack
5. Track which assets use which versions
6. Manually identify outdated assets
```

### AI-Enhanced Process (With QL-RF)

```
Time: 15-30 minutes

User: "Build and publish a new web-tier golden image with the latest
       security patches, then identify all assets that need updating"

AI Agent (Image Agent):
├── generate_image_contract(family=web-tier, updates=security-patches)
├── generate_packer_template(contract, platforms=[aws, azure, gcp])
├── build_image(template, validate=true)
├── promote_image(image_id, environments=[dev, staging])
├── query_assets(image_family=web-tier, version!=latest)
└── generate_rollout_plan(outdated_assets, new_image)

Output:
• New image web-tier-2024.12.04 built and validated
• Promoted to dev and staging registries
• 34 production assets identified on older versions
• Rollout plan generated for upgrade

[Approve Promotion to Prod] [View Rollout Plan]
```

---

## Cross-Journey: Task Approval Workflow

All state-changing operations follow a consistent approval workflow:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         TASK APPROVAL WORKFLOW                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  TASK SUBMITTED                                                              │
│       │                                                                      │
│       ▼                                                                      │
│  ┌─────────────┐     ┌──────────────────────────────────────────────────┐   │
│  │ Risk Level  │────▶│ read_only / plan_only: Auto-approved             │   │
│  │ Assessment  │     │ state_change_nonprod: Single approval            │   │
│  └─────────────┘     │ state_change_prod: Requires engineer+ approval   │   │
│       │              └──────────────────────────────────────────────────┘   │
│       ▼                                                                      │
│  ┌─────────────┐                                                            │
│  │   PENDING   │◀─────────────────────────────────────────┐                │
│  │  APPROVAL   │                                          │                │
│  └─────────────┘                                          │                │
│       │                                                   │                │
│       ├──────────────┬──────────────┬────────────────────┤                │
│       ▼              ▼              ▼                    │                │
│  [Approve]      [Reject]       [Modify]             [Timeout]             │
│       │              │              │                    │                │
│       ▼              ▼              ▼                    ▼                │
│  ┌─────────┐   ┌──────────┐   ┌─────────┐        ┌──────────┐            │
│  │APPROVED │   │ REJECTED │   │RE-SUBMIT│        │ EXPIRED  │            │
│  └─────────┘   └──────────┘   └─────────┘        └──────────┘            │
│       │              │                                   │                │
│       ▼              ▼                                   ▼                │
│  ┌─────────┐   ┌──────────┐                       ┌──────────┐            │
│  │EXECUTING│   │  CLOSED  │                       │ ESCALATE │            │
│  └─────────┘   └──────────┘                       └──────────┘            │
│       │                                                                    │
│       ├──────────────┬───────────────┐                                    │
│       ▼              ▼               ▼                                    │
│  [Success]      [Failure]       [Paused]                                  │
│       │              │               │                                    │
│       ▼              ▼               ▼                                    │
│  ┌─────────┐   ┌──────────┐   ┌──────────┐                               │
│  │COMPLETED│   │ ROLLBACK │   │ WAITING  │                               │
│  └─────────┘   │ EXECUTED │   │  RESUME  │                               │
│                └──────────┘   └──────────┘                               │
│                                                                           │
├─────────────────────────────────────────────────────────────────────────────┤
│  NOTIFICATIONS                                                              │
│  • Slack: Task pending approval, execution started/completed/failed        │
│  • Email: High-risk tasks, approval timeouts, failures                     │
│  • Webhook: All events for external integrations                           │
│  • ServiceNow: Change requests auto-created for state changes              │
├─────────────────────────────────────────────────────────────────────────────┤
│  AUDIT TRAIL                                                                │
│  • ai_tasks: Task metadata, intent, status                                 │
│  • ai_plans: Generated plans with quality scores                           │
│  • ai_runs: Execution details, phases, timing                              │
│  • ai_tool_invocations: Every tool call with parameters and results        │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Approval Timeout

Tasks awaiting approval have a **24-hour timeout** (configurable). If not actioned:
1. Task marked as `expired`
2. Notification sent to escalation contacts
3. Original submitter notified
4. Task can be re-submitted if still needed

### Required Permissions

| Action | Required Permission | Minimum Role |
|--------|-------------------|--------------|
| Submit task | `execute:ai-tasks` | operator |
| Approve non-prod | `approve:ai-tasks` | engineer |
| Approve production | `approve:ai-tasks` | engineer |
| Pause/Resume | `execute:ai-tasks` | operator |
| Cancel | `execute:ai-tasks` | operator |
| Emergency override | `approve:exceptions` | admin |

---

## Quick Reference

### Common Intents by Journey

| Journey | Example Intent |
|---------|---------------|
| Drift | "Show drift status for production" |
| Drift | "Remediate critical drift on web servers" |
| Patch | "Create patch plan for CVE-2024-XXXX" |
| Patch | "Roll out security patches to staging" |
| Compliance | "Generate SOC 2 evidence for Q4" |
| Compliance | "Check CIS benchmark compliance" |
| DR | "What's our DR readiness score?" |
| DR | "Execute DR drill for database tier" |
| Incident | "Investigate alerts on checkout service" |
| Incident | "What changed in the last 24 hours?" |
| Image | "Build new golden image with latest patches" |
| Image | "Which assets are running outdated images?" |

### Response Time Expectations

| Operation Type | Expected Response |
|---------------|-------------------|
| Read-only query | 2-5 seconds |
| Plan generation | 30-60 seconds |
| Plan approval | Instant |
| Execution start | 5-10 seconds |
| Full execution | Varies by scope |
