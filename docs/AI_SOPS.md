# AI Standard Operating Procedures (SOPs)

## Overview

This document defines standard operating procedures for all AI-related workflows in QL-RF. These SOPs ensure consistent, auditable, and safe operation of AI capabilities.

---

## Table of Contents

1. [SOP Index](#sop-index)
2. [AI-SOP-001: Task Submission](#ai-sop-001-task-submission)
3. [AI-SOP-002: Plan Review & Approval](#ai-sop-002-plan-review--approval)
4. [AI-SOP-003: Dual Approval (Production Critical)](#ai-sop-003-dual-approval-production-critical)
5. [AI-SOP-004: Task Rejection](#ai-sop-004-task-rejection)
6. [AI-SOP-005: Execution Monitoring](#ai-sop-005-execution-monitoring)
7. [AI-SOP-006: Pause/Resume Execution](#ai-sop-006-pauseresume-execution)
8. [AI-SOP-007: Rollback Procedure](#ai-sop-007-rollback-procedure)
9. [AI-SOP-008: Emergency Override](#ai-sop-008-emergency-override)
10. [AI-SOP-009: AI Incident Response](#ai-sop-009-ai-incident-response)
11. [AI-SOP-010: AI Hallucination Report](#ai-sop-010-ai-hallucination-report)
12. [AI-SOP-011: Approval Timeout Handling](#ai-sop-011-approval-timeout-handling)
13. [AI-SOP-012: Cost Overrun Response](#ai-sop-012-cost-overrun-response)

---

## SOP Index

| SOP ID | Title | Trigger | Primary Role | Risk Level |
|--------|-------|---------|--------------|------------|
| AI-SOP-001 | Task Submission | User wants AI assistance | Operator+ | Low |
| AI-SOP-002 | Plan Review & Approval | Task awaiting approval | Engineer+ | Medium |
| AI-SOP-003 | Dual Approval | Critical production task | 2x Engineer | High |
| AI-SOP-004 | Task Rejection | Plan doesn't meet requirements | Engineer+ | Low |
| AI-SOP-005 | Execution Monitoring | Task executing | Operator+ | Low |
| AI-SOP-006 | Pause/Resume | Health check fails or concern | Engineer+ | Medium |
| AI-SOP-007 | Rollback | Execution fails | Engineer+ | High |
| AI-SOP-008 | Emergency Override | Urgent action needed | Admin | Critical |
| AI-SOP-009 | AI Incident Response | AI causes outage | Admin | Critical |
| AI-SOP-010 | Hallucination Report | AI generates invalid plan | Any | Medium |
| AI-SOP-011 | Approval Timeout | 24h without action | Operator+ | Low |
| AI-SOP-012 | Cost Overrun | Token budget exceeded | Admin | Medium |

---

## AI-SOP-001: Task Submission

### Purpose
Define the standard process for submitting AI tasks via natural language intent.

### Scope
All users with `execute:ai-tasks` permission submitting requests to the AI Copilot.

### Prerequisites
- User authenticated with valid session
- User has `execute:ai-tasks` permission (Operator, Engineer, or Admin role)
- Target environment accessible

### Procedure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  STEP 1: FORMULATE INTENT                                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  DO:                                                                         │
│  ✓ Be specific about what you want                                          │
│  ✓ Include environment (prod, staging, dev)                                 │
│  ✓ Specify scope (which systems, assets, regions)                           │
│  ✓ Include any constraints (maintenance window, batch size)                 │
│                                                                              │
│  DON'T:                                                                      │
│  ✗ Use vague requests ("fix everything")                                    │
│  ✗ Combine multiple unrelated tasks in one intent                           │
│  ✗ Include credentials or secrets in the intent                             │
│                                                                              │
│  GOOD EXAMPLES:                                                              │
│  • "Show drift status for production web servers in us-east-1"              │
│  • "Create patch plan for CVE-2024-1234 on staging database tier"           │
│  • "Generate SOC 2 compliance evidence for Q4 2025"                         │
│                                                                              │
│  BAD EXAMPLES:                                                               │
│  • "Fix the servers" (too vague)                                            │
│  • "Patch everything and also check compliance" (multiple tasks)            │
│  • "Update server with password admin123" (contains credentials)            │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  STEP 2: SUBMIT VIA UI OR API                                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  UI PATH:                                                                    │
│  1. Navigate to /ai (AI Copilot page)                                       │
│  2. Enter intent in chat input                                              │
│  3. Click Send or press Enter                                               │
│  4. Wait for plan generation (30-60 seconds)                                │
│                                                                              │
│  API:                                                                        │
│  POST /api/v1/ai/execute                                                    │
│  {                                                                          │
│    "intent": "Your natural language request",                               │
│    "context": {                                                             │
│      "environment": "prod",                                                 │
│      "region": "us-east-1"                                                  │
│    }                                                                        │
│  }                                                                          │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  STEP 3: REVIEW AI RESPONSE                                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  For READ-ONLY tasks:                                                        │
│  • Results displayed immediately                                            │
│  • No approval required                                                     │
│  • Data available for export                                                │
│                                                                              │
│  For STATE-CHANGING tasks:                                                   │
│  • Plan generated and displayed                                             │
│  • Risk level assessed                                                      │
│  • Approval required (see AI-SOP-002)                                       │
│                                                                              │
│  If ERROR:                                                                   │
│  • Check error message for guidance                                         │
│  • Reformulate intent if unclear                                            │
│  • Contact support if persistent                                            │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Decision Criteria
- If intent is ambiguous → AI will ask clarifying questions
- If no matching agent → Error returned with suggestions
- If permission denied → Error with required permission listed

### Audit Requirements
- Task submission logged to `ai_tasks` table
- Intent, submitter, timestamp captured
- Context parameters stored

---

## AI-SOP-002: Plan Review & Approval

### Purpose
Define the standard process for reviewing and approving AI-generated plans.

### Scope
All state-changing tasks that require human approval before execution.

### Prerequisites
- User has `approve:ai-tasks` permission (Engineer or Admin role)
- Task in `pending_approval` status
- User is NOT the original submitter (for production tasks)

### Procedure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  STEP 1: RECEIVE NOTIFICATION                                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Notifications sent via:                                                     │
│  • Slack: #ai-approvals channel                                             │
│  • Email: To engineers with approve permission                              │
│  • UI: Badge on AI Copilot navigation item                                  │
│                                                                              │
│  Notification includes:                                                      │
│  • Task ID and summary                                                      │
│  • Submitter name                                                           │
│  • Risk level                                                               │
│  • Quick link to review                                                     │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  STEP 2: REVIEW PLAN DETAILS                                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Navigate to: /ai/tasks/{task_id}                                           │
│                                                                              │
│  REVIEW CHECKLIST:                                                           │
│                                                                              │
│  □ SCOPE                                                                     │
│    • Are the affected assets correct?                                       │
│    • Is the scope appropriate (not too broad)?                              │
│    • Are critical systems properly identified?                              │
│                                                                              │
│  □ RISK                                                                      │
│    • Is the risk level appropriate?                                         │
│    • Is the blast radius acceptable?                                        │
│    • Are rollback triggers defined?                                         │
│                                                                              │
│  □ PLAN QUALITY                                                              │
│    • Is the quality score acceptable (≥80 for prod)?                        │
│    • Are all phases defined?                                                │
│    • Are health checks included?                                            │
│                                                                              │
│  □ TIMING                                                                    │
│    • Is the maintenance window appropriate?                                 │
│    • Is the duration estimate reasonable?                                   │
│    • Are dependencies considered?                                           │
│                                                                              │
│  □ ROLLBACK                                                                  │
│    • Is rollback plan included?                                             │
│    • Are rollback triggers clear?                                           │
│    • Is rollback tested/validated?                                          │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  STEP 3: MAKE DECISION                                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  OPTION A: APPROVE                                                           │
│  • Click [Approve] button                                                   │
│  • Add approver notes (optional but recommended)                            │
│  • Execution begins automatically                                           │
│                                                                              │
│  OPTION B: REJECT                                                            │
│  • Click [Reject] button                                                    │
│  • Provide rejection reason (REQUIRED)                                      │
│  • Task marked as rejected                                                  │
│  • Submitter notified                                                       │
│  • See AI-SOP-004 for rejection handling                                    │
│                                                                              │
│  OPTION C: MODIFY                                                            │
│  • Click [Modify Plan] button                                               │
│  • Adjust parameters (batch size, phases, timing)                           │
│  • Re-validate modified plan                                                │
│  • Then Approve or Reject                                                   │
│                                                                              │
│  OPTION D: REQUEST MORE INFO                                                 │
│  • Add comment requesting clarification                                     │
│  • Submitter notified                                                       │
│  • Task remains in pending status                                           │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Decision Criteria

| Condition | Action |
|-----------|--------|
| Quality score ≥80, scope correct | Approve |
| Quality score <80 | Request modification or reject |
| Scope too broad | Reject with narrower scope suggestion |
| Missing rollback plan | Reject |
| Unclear success criteria | Request clarification |
| Production + critical | Follow AI-SOP-003 (dual approval) |

### Audit Requirements
- Approval/rejection logged with timestamp
- Approver identity recorded
- Notes captured
- Reason required for rejections

---

## AI-SOP-003: Dual Approval (Production Critical)

### Purpose
Define enhanced approval process for high-risk production changes.

### Scope
Tasks meeting ANY of these criteria:
- Risk level: `state_change_prod`
- Affects >50% of production fleet
- Touches critical infrastructure (databases, auth, payments)
- Requested during business hours

### Prerequisites
- Two users with `approve:ai-tasks` permission
- Neither approver is the original submitter
- Both have reviewed the plan independently

### Procedure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  DUAL APPROVAL WORKFLOW                                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Task Submitted                                                              │
│       │                                                                      │
│       ▼                                                                      │
│  ┌─────────────────┐                                                        │
│  │ Risk Assessment │                                                        │
│  │ Detects: DUAL   │                                                        │
│  │ APPROVAL NEEDED │                                                        │
│  └────────┬────────┘                                                        │
│           │                                                                  │
│           ▼                                                                  │
│  ┌─────────────────────────────────────────────────────────────────┐        │
│  │                    PARALLEL REVIEW                               │        │
│  │                                                                  │        │
│  │  ┌─────────────────┐              ┌─────────────────┐           │        │
│  │  │   APPROVER 1    │              │   APPROVER 2    │           │        │
│  │  │                 │              │                 │           │        │
│  │  │ Reviews plan    │              │ Reviews plan    │           │        │
│  │  │ independently   │              │ independently   │           │        │
│  │  │                 │              │                 │           │        │
│  │  │ [Approve]       │              │ [Approve]       │           │        │
│  │  └────────┬────────┘              └────────┬────────┘           │        │
│  │           │                                │                    │        │
│  │           └────────────┬───────────────────┘                    │        │
│  │                        ▼                                        │        │
│  │               ┌────────────────┐                                │        │
│  │               │ BOTH APPROVED? │                                │        │
│  │               └────────┬───────┘                                │        │
│  │                        │                                        │        │
│  │           ┌────────────┴────────────┐                          │        │
│  │           ▼                         ▼                          │        │
│  │        [YES]                      [NO]                         │        │
│  │           │                         │                          │        │
│  │           ▼                         ▼                          │        │
│  │    ┌─────────────┐          ┌─────────────┐                    │        │
│  │    │  EXECUTION  │          │   BLOCKED   │                    │        │
│  │    │   BEGINS    │          │  (see note) │                    │        │
│  │    └─────────────┘          └─────────────┘                    │        │
│  │                                                                 │        │
│  └─────────────────────────────────────────────────────────────────┘        │
│                                                                              │
│  NOTE: If either approver rejects, task is rejected.                        │
│        If one approves and one hasn't acted, task waits.                    │
│        24-hour timeout applies to both approvers.                           │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Audit Requirements
- Both approvers recorded
- Timestamps for each approval
- Order of approvals logged

---

## AI-SOP-004: Task Rejection

### Purpose
Define the process for rejecting AI-generated plans and enabling re-submission.

### Scope
Any task that fails review criteria.

### Procedure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  REJECTION WORKFLOW                                                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  STEP 1: DOCUMENT REJECTION                                                  │
│  ─────────────────────────────                                               │
│  When clicking [Reject], MUST provide:                                       │
│  • Clear reason for rejection                                               │
│  • Specific issues identified                                               │
│  • Suggested improvements (if any)                                          │
│                                                                              │
│  REJECTION REASONS (select one or more):                                     │
│  □ Scope too broad                                                          │
│  □ Quality score too low                                                    │
│  □ Missing rollback plan                                                    │
│  □ Incorrect assets targeted                                                │
│  □ Timing/maintenance window inappropriate                                  │
│  □ Risk too high for requested change                                       │
│  □ Policy violation                                                         │
│  □ Other (specify)                                                          │
│                                                                              │
│  STEP 2: NOTIFY SUBMITTER                                                    │
│  ─────────────────────────                                                   │
│  Automatic notification sent with:                                           │
│  • Rejection reason                                                         │
│  • Approver feedback                                                        │
│  • Suggested next steps                                                     │
│                                                                              │
│  STEP 3: SUBMITTER OPTIONS                                                   │
│  ─────────────────────────                                                   │
│  A. Re-submit with refined intent                                           │
│     • Address feedback in new request                                       │
│     • Creates new task (linked to original)                                 │
│                                                                              │
│  B. Request escalation                                                       │
│     • If disagreement with rejection                                        │
│     • Escalates to Risk Owner                                               │
│                                                                              │
│  C. Abandon                                                                  │
│     • If task no longer needed                                              │
│     • Close task as cancelled                                               │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## AI-SOP-005: Execution Monitoring

### Purpose
Define the process for monitoring AI task execution in real-time.

### Scope
All tasks in `executing` status.

### Procedure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  MONITORING WORKFLOW                                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  UI PATH: /ai/tasks/{task_id} → Execution tab                               │
│                                                                              │
│  MONITORING DASHBOARD SHOWS:                                                 │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  Phase Progress: [████████░░░░░░░░░░░░] 40%                         │    │
│  │                                                                      │    │
│  │  Phase 1: Canary (2 servers)          ✓ Complete                    │    │
│  │  Phase 2: 25% rollout (5 servers)     ◉ In Progress (3/5)           │    │
│  │  Phase 3: Remaining (13 servers)      ○ Pending                     │    │
│  │                                                                      │    │
│  │  Current Phase Details:                                              │    │
│  │  ├── web-prod-003: ✓ Complete                                       │    │
│  │  ├── web-prod-007: ✓ Complete                                       │    │
│  │  ├── web-prod-012: ◉ Executing (45s)                                │    │
│  │  ├── web-prod-015: ○ Queued                                         │    │
│  │  └── web-prod-019: ○ Queued                                         │    │
│  │                                                                      │    │
│  │  Health Status: ✓ All checks passing                                │    │
│  │  Estimated Completion: 12 minutes                                   │    │
│  │                                                                      │    │
│  │  [Pause] [Cancel]                                                   │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  WHAT TO MONITOR:                                                            │
│  • Phase progression (on schedule?)                                         │
│  • Health check status (any failures?)                                      │
│  • Individual asset status                                                  │
│  • Error messages (if any)                                                  │
│  • Rollback trigger conditions                                              │
│                                                                              │
│  WHEN TO INTERVENE:                                                          │
│  • Health checks failing → Consider [Pause]                                 │
│  • Progress stalled → Investigate                                           │
│  • Errors accumulating → Consider [Cancel]                                  │
│  • External incident → [Pause] immediately                                  │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## AI-SOP-006: Pause/Resume Execution

### Purpose
Define the process for pausing and resuming AI task execution.

### Scope
Tasks in `executing` status where intervention is needed.

### Prerequisites
- User has `execute:ai-tasks` permission
- Task currently executing

### Procedure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  PAUSE PROCEDURE                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  WHEN TO PAUSE:                                                              │
│  • Health checks showing warnings (not yet failures)                        │
│  • External incident declared                                               │
│  • Stakeholder concern raised                                               │
│  • Need to validate intermediate state                                      │
│  • Approaching business hours (if not approved for)                         │
│                                                                              │
│  HOW TO PAUSE:                                                               │
│  1. Click [Pause] button on execution dashboard                             │
│  2. Provide pause reason (required)                                         │
│  3. Current phase completes, next phase waits                               │
│  4. Notifications sent to stakeholders                                      │
│                                                                              │
│  API:                                                                        │
│  POST /api/v1/ai/executions/{execution_id}/pause                            │
│  { "reason": "Investigating health check warning on web-prod-003" }         │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  RESUME PROCEDURE                                                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  BEFORE RESUMING:                                                            │
│  □ Validate pause condition resolved                                        │
│  □ Check current asset health                                               │
│  □ Confirm stakeholder concerns addressed                                   │
│  □ Verify maintenance window still valid                                    │
│                                                                              │
│  HOW TO RESUME:                                                              │
│  1. Click [Resume] button                                                   │
│  2. Confirm resume action                                                   │
│  3. Execution continues from next phase                                     │
│                                                                              │
│  API:                                                                        │
│  POST /api/v1/ai/executions/{execution_id}/resume                           │
│                                                                              │
│  NOTE: If paused >4 hours, re-validation of plan may be required            │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## AI-SOP-007: Rollback Procedure

### Purpose
Define the process for rolling back failed AI task executions.

### Scope
Tasks where execution has failed or produced unintended results.

### Procedure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  ROLLBACK TRIGGERS                                                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  AUTOMATIC ROLLBACK (no human action needed):                                │
│  • >5% of assets in current phase fail health check                         │
│  • Critical error detected during execution                                 │
│  • Dependency service becomes unavailable                                   │
│                                                                              │
│  MANUAL ROLLBACK (human initiates):                                          │
│  • Post-execution issues discovered                                         │
│  • Stakeholder requests reversal                                            │
│  • Unexpected side effects observed                                         │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  AUTOMATIC ROLLBACK WORKFLOW                                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. TRIGGER DETECTED                                                         │
│     └── Execution engine detects rollback condition                         │
│                                                                              │
│  2. PAUSE NEW ACTIONS                                                        │
│     └── Stop processing queued assets                                       │
│                                                                              │
│  3. EXECUTE ROLLBACK PLAN                                                    │
│     ├── Revert completed assets in reverse order                           │
│     ├── Run health checks after each revert                                │
│     └── Log all rollback actions                                           │
│                                                                              │
│  4. NOTIFY STAKEHOLDERS                                                      │
│     ├── Slack alert with rollback reason                                   │
│     ├── Email to approver and submitter                                    │
│     └── ServiceNow incident created                                        │
│                                                                              │
│  5. MARK TASK FAILED                                                         │
│     └── Status: `failed_with_rollback`                                      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  MANUAL ROLLBACK PROCEDURE                                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  STEP 1: ASSESS IMPACT                                                       │
│  • Identify which assets were modified                                      │
│  • Determine current state vs desired state                                 │
│  • Evaluate rollback feasibility                                            │
│                                                                              │
│  STEP 2: INITIATE ROLLBACK                                                   │
│  UI: Click [Rollback] on task detail page                                   │
│  API: POST /api/v1/ai/executions/{id}/rollback                              │
│       { "reason": "...", "scope": "all" | "partial" }                       │
│                                                                              │
│  STEP 3: MONITOR ROLLBACK                                                    │
│  • Same monitoring as forward execution                                     │
│  • Watch for rollback failures                                              │
│                                                                              │
│  STEP 4: VALIDATE                                                            │
│  • Confirm assets returned to previous state                                │
│  • Run health checks                                                        │
│  • Update stakeholders                                                      │
│                                                                              │
│  STEP 5: POST-MORTEM                                                         │
│  • Document what went wrong                                                 │
│  • Update AI constraints if needed                                          │
│  • Follow AI-SOP-009 if incident                                            │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## AI-SOP-008: Emergency Override

### Purpose
Define the process for bypassing normal approval workflows in emergencies.

### Scope
Critical situations requiring immediate AI-assisted action.

### Prerequisites
- User has `approve:exceptions` permission (Admin only)
- Genuine emergency (not convenience)
- MFA verification required

### Procedure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  ⚠️  EMERGENCY OVERRIDE - USE WITH EXTREME CAUTION                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  VALID EMERGENCY SCENARIOS:                                                  │
│  • Active security breach requiring immediate patching                      │
│  • Production outage requiring AI-assisted remediation                      │
│  • Compliance deadline with no time for normal approval                     │
│  • DR event requiring immediate failover                                    │
│                                                                              │
│  NOT VALID:                                                                  │
│  • Approver unavailable (use timeout escalation instead)                    │
│  • Convenience or time pressure                                             │
│  • Disagreement with rejection (use escalation instead)                     │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  OVERRIDE PROCEDURE                                                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  STEP 1: AUTHENTICATE                                                        │
│  • Login with Admin credentials                                             │
│  • Complete MFA challenge (required)                                        │
│  • Session elevated for override                                            │
│                                                                              │
│  STEP 2: DOCUMENT JUSTIFICATION                                              │
│  • Emergency type (dropdown)                                                │
│  • Detailed justification (free text, minimum 100 characters)               │
│  • Impact if not overridden                                                 │
│  • Incident ticket number (if exists)                                       │
│                                                                              │
│  STEP 3: EXECUTE OVERRIDE                                                    │
│  • Click [Emergency Override]                                               │
│  • Confirm understanding of audit trail                                     │
│  • Task bypasses normal approval                                            │
│  • Execution begins immediately                                             │
│                                                                              │
│  STEP 4: IMMEDIATE NOTIFICATIONS                                             │
│  • CISO notified via page                                                   │
│  • Risk Owner notified                                                      │
│  • All Admins notified                                                      │
│  • Audit log flagged for review                                             │
│                                                                              │
│  STEP 5: POST-EMERGENCY                                                      │
│  • Mandatory incident review within 24 hours                                │
│  • Document lessons learned                                                 │
│  • Update emergency procedures if needed                                    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Audit Requirements
- Override logged with maximum detail
- Justification preserved indefinitely
- Reviewed in monthly security audit
- Pattern analysis for abuse detection

---

## AI-SOP-009: AI Incident Response

### Purpose
Define the response process when AI causes a production incident.

### Scope
Any incident where AI task execution is the root cause or contributing factor.

### Procedure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  AI INCIDENT RESPONSE WORKFLOW                                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  PHASE 1: IMMEDIATE RESPONSE (0-15 minutes)                                  │
│  ──────────────────────────────────────────                                  │
│                                                                              │
│  □ STOP THE BLEEDING                                                         │
│    • Pause all related AI executions                                        │
│    • Cancel queued tasks from same agent                                    │
│    • Enable "AI Hold" mode if severe (pauses all AI)                        │
│                                                                              │
│  □ DECLARE INCIDENT                                                          │
│    • Create incident ticket (or link to existing)                           │
│    • Set severity based on impact                                           │
│    • Page on-call + AI system owner                                         │
│                                                                              │
│  □ INITIAL ASSESSMENT                                                        │
│    • Which AI task caused the incident?                                     │
│    • What was the intended vs actual outcome?                               │
│    • How many assets/customers affected?                                    │
│                                                                              │
│  PHASE 2: INVESTIGATION (15-60 minutes)                                      │
│  ───────────────────────────────────────                                     │
│                                                                              │
│  □ GATHER EVIDENCE                                                           │
│    • Export full AI audit trail (task, plan, run, tool invocations)         │
│    • Capture current state of affected assets                               │
│    • Collect relevant logs and metrics                                      │
│                                                                              │
│  □ IDENTIFY ROOT CAUSE                                                       │
│    Questions to answer:                                                      │
│    • Was the plan flawed? (AI generated bad plan)                           │
│    • Was execution buggy? (Plan was good, execution failed)                 │
│    • Was approval wrong? (Human approved bad plan)                          │
│    • Was there a system issue? (Infrastructure failure)                     │
│    • Were constraints insufficient? (Guardrails missing)                    │
│                                                                              │
│  □ DETERMINE REMEDIATION                                                     │
│    • Rollback possible? Execute AI-SOP-007                                  │
│    • Manual fix needed? Document steps                                      │
│    • Customer communication needed? Draft messaging                         │
│                                                                              │
│  PHASE 3: REMEDIATION (Variable)                                             │
│  ───────────────────────────────                                             │
│                                                                              │
│  □ EXECUTE FIX                                                               │
│    • Rollback if possible                                                   │
│    • Manual remediation if needed                                           │
│    • Verify fix successful                                                  │
│                                                                              │
│  □ RESTORE SERVICE                                                           │
│    • Confirm all affected assets recovered                                  │
│    • Monitor for recurrence                                                 │
│    • Update incident status                                                 │
│                                                                              │
│  PHASE 4: POST-INCIDENT (24-48 hours)                                        │
│  ─────────────────────────────────────                                       │
│                                                                              │
│  □ POST-MORTEM                                                               │
│    • Timeline of events                                                     │
│    • Root cause analysis (use 5 Whys)                                       │
│    • Contributing factors                                                   │
│    • What worked well                                                       │
│                                                                              │
│  □ ACTION ITEMS                                                              │
│    • Update OPA policies if constraint gap                                  │
│    • Update agent prompts if reasoning gap                                  │
│    • Add integration tests for scenario                                     │
│    • Update this SOP if process gap                                         │
│                                                                              │
│  □ COMMUNICATION                                                             │
│    • Internal post-mortem shared                                            │
│    • Customer communication if needed                                       │
│    • Update status page                                                     │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## AI-SOP-010: AI Hallucination Report

### Purpose
Define the process for reporting and handling AI hallucinations (invalid or nonsensical outputs).

### Scope
Any AI output that is factually incorrect, logically inconsistent, or potentially harmful.

### Procedure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  HALLUCINATION INDICATORS                                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  WHAT COUNTS AS HALLUCINATION:                                               │
│  • References to assets that don't exist                                    │
│  • Incorrect technical facts (wrong package versions, etc.)                 │
│  • Logically impossible plans (circular dependencies)                       │
│  • Invented procedures or commands                                          │
│  • Contradictory statements in same plan                                    │
│  • Confident assertions about unknown data                                  │
│                                                                              │
│  NOT HALLUCINATION (different issues):                                       │
│  • Plan is valid but not optimal → Rejection feedback                       │
│  • Scope is wrong → Rejection with guidance                                 │
│  • Risk assessment differs from human → Normal review process               │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  REPORTING PROCEDURE                                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  STEP 1: REJECT THE TASK                                                     │
│  • Click [Reject]                                                           │
│  • Select reason: "AI Hallucination"                                        │
│  • Document specific hallucinated content                                   │
│                                                                              │
│  STEP 2: SUBMIT HALLUCINATION REPORT                                         │
│  • Click [Report Hallucination] on task page                                │
│  • Fill out report:                                                         │
│    - Task ID (auto-filled)                                                 │
│    - Hallucinated content (quote exactly)                                  │
│    - Why it's incorrect (evidence)                                         │
│    - Severity (minor/moderate/severe)                                      │
│                                                                              │
│  STEP 3: ENGINEERING REVIEW                                                  │
│  • AI team reviews report                                                   │
│  • Investigates prompt/context issues                                       │
│  • Updates prompts or constraints                                           │
│  • Adds to regression test suite                                            │
│                                                                              │
│  STEP 4: FEEDBACK LOOP                                                       │
│  • Reporter notified of resolution                                          │
│  • Similar future hallucinations should be caught                           │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## AI-SOP-011: Approval Timeout Handling

### Purpose
Define the process when tasks exceed the 24-hour approval timeout.

### Scope
Tasks in `pending_approval` status for >24 hours.

### Procedure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  TIMEOUT WORKFLOW                                                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  AUTOMATIC ACTIONS AT 24 HOURS:                                              │
│  1. Task status → `expired`                                                 │
│  2. Notifications sent:                                                      │
│     • Original submitter: "Your task expired"                               │
│     • Approvers: "Task expired without action"                              │
│     • Manager (if configured): "Approval SLA missed"                        │
│                                                                              │
│  SUBMITTER OPTIONS:                                                          │
│                                                                              │
│  OPTION A: RE-SUBMIT                                                         │
│  • If task still needed                                                     │
│  • Creates new task (linked to expired one)                                 │
│  • New 24-hour timer starts                                                 │
│  • Consider: Is intent still valid?                                         │
│                                                                              │
│  OPTION B: ESCALATE                                                          │
│  • If approval delays are systemic                                          │
│  • Notify Risk Owner                                                        │
│  • Review approver availability                                             │
│                                                                              │
│  OPTION C: CLOSE                                                             │
│  • If task no longer needed                                                 │
│  • Mark as `cancelled`                                                      │
│                                                                              │
│  PREVENTION:                                                                 │
│  • Set up approval rotation                                                 │
│  • Use Slack/email notifications                                            │
│  • Configure backup approvers                                               │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## AI-SOP-012: Cost Overrun Response

### Purpose
Define the process when AI token consumption exceeds budgets.

### Scope
Organizations exceeding configured token limits.

### Procedure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  COST GOVERNANCE THRESHOLDS                                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  THRESHOLD LEVELS:                                                           │
│  • 75% of monthly budget: Warning notification                              │
│  • 90% of monthly budget: Alert to Admin                                    │
│  • 100% of monthly budget: New tasks blocked (read-only allowed)            │
│  • 120% of monthly budget: All AI functions disabled                        │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  RESPONSE PROCEDURE                                                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  AT 75% (Warning):                                                           │
│  • Review current usage patterns                                            │
│  • Identify high-cost tasks                                                 │
│  • Consider optimization opportunities                                      │
│                                                                              │
│  AT 90% (Alert):                                                             │
│  • Admin reviews usage                                                      │
│  • Prioritize remaining budget                                              │
│  • Defer non-critical AI tasks                                              │
│  • Consider budget increase request                                         │
│                                                                              │
│  AT 100% (Blocked):                                                          │
│  • New state-changing tasks blocked                                         │
│  • Read-only queries still allowed                                          │
│  • Admin can:                                                               │
│    a) Increase budget (requires approval)                                  │
│    b) Wait for monthly reset                                               │
│    c) Use emergency override (AI-SOP-008) for critical tasks               │
│                                                                              │
│  AT 120% (Disabled):                                                         │
│  • All AI functions disabled                                                │
│  • Finance team notified                                                    │
│  • Emergency budget approval required                                       │
│                                                                              │
│  BUDGET INCREASE REQUEST:                                                    │
│  1. Document justification                                                  │
│  2. Submit to Finance/Risk Owner                                            │
│  3. Approval within 24 hours (SLA)                                          │
│  4. Budget updated in org settings                                          │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Appendix: Quick Reference Card

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    AI SOP QUICK REFERENCE                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  SUBMITTING TASKS                                                            │
│  • Be specific with intent                                                  │
│  • Include environment and scope                                            │
│  • One task per intent                                                      │
│                                                                              │
│  APPROVING TASKS                                                             │
│  • Check scope, risk, quality score                                         │
│  • Verify rollback plan exists                                              │
│  • Provide notes with approval                                              │
│                                                                              │
│  MONITORING EXECUTION                                                        │
│  • Watch phase progress                                                     │
│  • Check health status                                                      │
│  • Be ready to pause if issues                                              │
│                                                                              │
│  HANDLING FAILURES                                                           │
│  • Let automatic rollback complete                                          │
│  • Document what went wrong                                                 │
│  • Follow AI-SOP-009 for incidents                                          │
│                                                                              │
│  EMERGENCY CONTACTS                                                          │
│  • AI System Owner: ai-team@company.com                                     │
│  • On-Call: PagerDuty escalation                                            │
│  • Security: security@company.com                                           │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```
