# Agent Behaviors Documentation

## Overview

This document provides detailed documentation for each of the 11 specialist AI agents in QL-RF. Each agent has a defined identity, capabilities, decision logic, tools, guardrails, and performance metrics.

---

## Table of Contents

1. [Agent Architecture](#agent-architecture)
2. [Drift Agent](#drift-agent)
3. [Patch Agent](#patch-agent)
4. [Compliance Agent](#compliance-agent)
5. [Incident Agent](#incident-agent)
6. [DR Agent](#dr-agent)
7. [Cost Agent](#cost-agent)
8. [Security Agent](#security-agent)
9. [Image Agent](#image-agent)
10. [SOP Agent](#sop-agent)
11. [Adapter Agent](#adapter-agent)
12. [Base Agent](#base-agent)
13. [Agent Selection Logic](#agent-selection-logic)
14. [Cross-Agent Collaboration](#cross-agent-collaboration)

---

## Agent Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         AGENT ARCHITECTURE                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                      META-PROMPT ENGINE                               │   │
│  │                                                                       │   │
│  │   User Intent ──▶ Parse ──▶ Select Agent ──▶ Route Task              │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                      │                                       │
│                                      ▼                                       │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                      SPECIALIST AGENTS (11)                           │   │
│  │                                                                       │   │
│  │   ┌─────────┐ ┌─────────┐ ┌──────────┐ ┌──────────┐ ┌─────────┐     │   │
│  │   │  Drift  │ │  Patch  │ │Compliance│ │ Incident │ │   DR    │     │   │
│  │   └────┬────┘ └────┬────┘ └────┬─────┘ └────┬─────┘ └────┬────┘     │   │
│  │        │           │           │            │            │          │   │
│  │   ┌─────────┐ ┌─────────┐ ┌──────────┐ ┌──────────┐ ┌─────────┐     │   │
│  │   │  Cost   │ │Security │ │  Image   │ │   SOP    │ │ Adapter │     │   │
│  │   └────┬────┘ └────┬────┘ └────┬─────┘ └────┬─────┘ └────┬────┘     │   │
│  │        │           │           │            │            │          │   │
│  │        └───────────┴───────────┴────────────┴────────────┘          │   │
│  │                                │                                     │   │
│  │                       ┌────────┴────────┐                           │   │
│  │                       │   Base Agent    │                           │   │
│  │                       │ (shared logic)  │                           │   │
│  │                       └─────────────────┘                           │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                      │                                       │
│                                      ▼                                       │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                       TOOL REGISTRY (29 tools)                        │   │
│  │                                                                       │   │
│  │   Query Tools ─── Analysis Tools ─── Planning Tools ─── Exec Tools   │   │
│  │        │               │                  │                │         │   │
│  │   [read_only]     [read_only]        [plan_only]    [state_change]   │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Drift Agent

### Identity

| Field | Value |
|-------|-------|
| **Agent ID** | `drift-agent` |
| **Name** | Drift Remediation Agent |
| **Version** | 1.0.0 |
| **Department** | Platform Operations |
| **Risk Profile** | `state_change_prod` |
| **HITL Required** | Yes (for state changes) |

### Capabilities

**Primary:**
- Detect configuration drift across fleet
- Analyze drift severity and root cause
- Compare actual state vs golden image specifications
- Generate remediation plans (patch, reimage, rollback)
- Execute approved phased remediations

**Secondary:**
- Coordinate with Patch Agent for complex package updates
- Generate drift reports for compliance
- Predict drift likelihood based on patterns

### Decision Logic

```
INPUT: User intent about drift
│
▼
STEP 1: DISCOVER
├── query_assets(filters from intent)
├── Filter to drifted assets
└── Retrieve drift details
│
▼
STEP 2: ANALYZE
├── get_drift_status(each asset)
├── get_golden_image(image family)
├── Compare actual vs expected
├── Classify severity: minor | major | critical
└── Identify drift types: packages | configs | services
│
▼
STEP 3: PLAN
├── analyze_drift(root cause)
├── Determine remediation type:
│   ├── Package update → generate_patch_plan
│   ├── Config reset → generate_config_plan
│   └── Full reimage → generate_reimage_plan
├── Calculate risk score
└── Create phased rollout
│
▼
STEP 4: VALIDATE
├── simulate_rollout(dry_run=true)
├── Check OPA policies
├── Validate dependencies
└── Generate quality score
│
▼
OUTPUT: Plan for human approval
```

### Tools Used

| Tool | Risk Level | Purpose |
|------|------------|---------|
| `query_assets` | read_only | Fetch assets matching criteria |
| `get_drift_status` | read_only | Get drift details per asset |
| `get_golden_image` | read_only | Retrieve target specification |
| `analyze_drift` | read_only | Root cause analysis |
| `generate_patch_plan` | plan_only | Create package update plan |
| `simulate_rollout` | plan_only | Validate plan before execution |
| `propose_rollout` | plan_only | Submit plan for approval |

### Guardrails

```yaml
constraints:
  max_assets_per_batch: 20%     # Never remediate >20% of fleet at once
  require_canary_phase: true     # Always start with canary
  canary_size: 2-5               # Canary phase size
  canary_wait_time: 5m           # Minimum wait after canary
  auto_rollback_threshold: 5%    # Rollback if >5% fail
  production_approval: required  # HITL for prod changes
  health_check_required: true    # Must pass health checks

prohibited_actions:
  - Remediate without rollback plan
  - Skip canary phase in production
  - Modify critical infrastructure without dual approval
  - Execute during declared incidents
```

### Example Interaction

```
USER: "What's the drift situation in production?"

DRIFT AGENT:
├── query_assets(env=prod, drift_status=drifted)
│   └── Returns: 15 assets with drift
├── get_drift_status(asset_ids)
│   └── Returns: Drift details per asset
└── Formats response:

"I found 15 production assets with drift:

Critical (3 assets):
- web-prod-001: nginx 1.22.1 → 1.24.0 (security patch missing)
- web-prod-002: nginx 1.22.1 → 1.24.0
- api-prod-005: openssl 3.0.2 → 3.0.13 (CVE remediation needed)

Major (7 assets):
- [Package version drift in app tier]

Minor (5 assets):
- [Configuration drift - non-security]

Would you like me to create a remediation plan?"
```

---

## Patch Agent

### Identity

| Field | Value |
|-------|-------|
| **Agent ID** | `patch-agent` |
| **Name** | Patch Orchestration Agent |
| **Version** | 1.0.0 |
| **Department** | Platform Security |
| **Risk Profile** | `state_change_prod` |
| **HITL Required** | Yes |

### Capabilities

**Primary:**
- Identify assets requiring patches (CVE-based or version-based)
- Generate patch rollout plans with risk-aware sequencing
- Coordinate multi-phase patch deployments
- Handle patch dependencies and conflicts
- Execute approved patch operations

**Secondary:**
- Generate patch compliance reports
- Track patch SLA compliance
- Coordinate with Drift Agent for post-patch validation

### Decision Logic

```
INPUT: Patch request (CVE, package, or general update)
│
▼
STEP 1: SCOPE
├── Identify CVEs or packages to patch
├── query_assets(vulnerable to CVE or outdated)
├── Group by criticality, environment, dependencies
└── Calculate blast radius
│
▼
STEP 2: SEQUENCE
├── Dependency analysis (what must patch first?)
├── Risk ranking (least critical first)
├── Maintenance window detection
└── Generate phase sequence
│
▼
STEP 3: PLAN
├── generate_patch_plan(
│     assets,
│     phases,
│     validation_checks
│   )
├── Include pre-patch snapshots
├── Include rollback procedures
└── Calculate estimated duration
│
▼
STEP 4: VALIDATE
├── simulate_rollout(plan)
├── Check for conflicts
├── Verify maintenance windows
└── OPA policy check
│
▼
OUTPUT: Patch plan for approval
```

### Tools Used

| Tool | Risk Level | Purpose |
|------|------------|---------|
| `query_assets` | read_only | Find vulnerable/outdated assets |
| `get_drift_status` | read_only | Current patch levels |
| `compare_versions` | read_only | Version comparison |
| `generate_patch_plan` | plan_only | Create patch rollout |
| `generate_rollout_plan` | plan_only | Multi-phase strategy |
| `simulate_rollout` | plan_only | Validate before execution |
| `calculate_risk_score` | plan_only | Risk assessment |

### Guardrails

```yaml
constraints:
  max_concurrent_patches: 10%    # Parallel patching limit
  require_reboot_window: true    # Schedule reboots in windows
  pre_patch_snapshot: required   # Snapshot before patching
  patch_validation_timeout: 30m  # Max time for patch + validate

patch_ordering:
  - Non-production first
  - Least critical services first
  - Database/stateful services last
  - Always canary before bulk

prohibited_actions:
  - Patch production before non-prod validation
  - Skip dependency checks
  - Patch during peak traffic
  - Exceed maintenance window
```

---

## Compliance Agent

### Identity

| Field | Value |
|-------|-------|
| **Agent ID** | `compliance-agent` |
| **Name** | Compliance Assurance Agent |
| **Version** | 1.0.0 |
| **Department** | Governance & Risk |
| **Risk Profile** | `read_only` |
| **HITL Required** | No (read-only operations) |

### Capabilities

**Primary:**
- Query compliance status across frameworks (SOC 2, PCI-DSS, HIPAA, CIS)
- Check specific control effectiveness
- Generate compliance evidence packages
- Export audit-ready reports

**Secondary:**
- Identify compliance gaps
- Recommend remediation priorities
- Track compliance trends over time

### Decision Logic

```
INPUT: Compliance query or evidence request
│
▼
STEP 1: SCOPE
├── Identify framework(s) requested
├── Identify control(s) to check
├── Determine time period
└── Identify relevant assets
│
▼
STEP 2: ASSESS
├── get_compliance_status(framework, scope)
├── check_control(control_id, assets)
├── For each control:
│   ├── Evaluate control implementation
│   ├── Check evidence availability
│   └── Note exceptions/gaps
└── Aggregate compliance score
│
▼
STEP 3: REPORT
├── generate_compliance_evidence(
│     framework,
│     period,
│     include_narratives=true
│   )
├── Format for audience (auditor vs internal)
└── Include supporting data
│
▼
OUTPUT: Compliance report or evidence package
```

### Tools Used

| Tool | Risk Level | Purpose |
|------|------------|---------|
| `query_assets` | read_only | Asset inventory for compliance |
| `get_compliance_status` | read_only | Framework compliance status |
| `check_control` | read_only | Individual control check |
| `get_drift_status` | read_only | Drift as compliance indicator |
| `generate_compliance_evidence` | read_only | Evidence package generation |

### Guardrails

```yaml
constraints:
  evidence_freshness: 24h        # Evidence must be recent
  include_exceptions: always     # Never hide gaps
  audit_trail: required          # Log all evidence generation

compliance_frameworks_supported:
  - SOC 2 Type II
  - PCI-DSS v4.0
  - HIPAA
  - CIS Benchmarks (Level 1 & 2)
  - NIST 800-53

prohibited_actions:
  - Generate evidence for non-existent controls
  - Omit failing controls from reports
  - Backdate evidence
```

---

## Incident Agent

### Identity

| Field | Value |
|-------|-------|
| **Agent ID** | `incident-agent` |
| **Name** | Incident Response Agent |
| **Version** | 1.0.0 |
| **Department** | Site Reliability |
| **Risk Profile** | `state_change_nonprod` |
| **HITL Required** | Yes (for remediation) |

### Capabilities

**Primary:**
- Query and correlate alerts
- Identify affected assets and blast radius
- Analyze recent changes as potential causes
- Generate remediation recommendations
- Acknowledge and update alerts

**Secondary:**
- Create incident timelines
- Coordinate with Drift Agent for drift-related incidents
- Generate post-incident reports

### Decision Logic

```
INPUT: Incident investigation request
│
▼
STEP 1: TRIAGE
├── query_alerts(filters from intent)
├── Identify affected services/assets
├── Determine severity based on:
│   ├── Alert priority
│   ├── Number of assets affected
│   └── Customer impact
└── Create initial assessment
│
▼
STEP 2: CORRELATE
├── query_assets(affected services)
├── get_drift_status(assets)
├── Get recent changes (last 24-48h)
├── Look for patterns:
│   ├── Drift correlation with incident start
│   ├── Recent deployments
│   └── Configuration changes
└── Identify probable cause
│
▼
STEP 3: RECOMMEND
├── Generate remediation options:
│   ├── Rollback recent change
│   ├── Fix drifted configuration
│   ├── Scale up resources
│   └── Failover to DR
├── Rank by confidence and risk
└── Include validation steps
│
▼
OUTPUT: Analysis + remediation options
```

### Tools Used

| Tool | Risk Level | Purpose |
|------|------------|---------|
| `query_alerts` | read_only | Get firing alerts |
| `query_assets` | read_only | Affected asset inventory |
| `get_drift_status` | read_only | Check drift correlation |
| `analyze_drift` | read_only | Drift root cause |
| `acknowledge_alert` | state_change_nonprod | Ack alerts during response |

### Guardrails

```yaml
constraints:
  correlation_window: 48h        # Look back for causes
  max_remediation_scope: 25%     # Limit blast radius of fixes
  require_validation: true       # Verify fix worked

incident_priorities:
  SEV1: Immediate response required
  SEV2: Response within 15 minutes
  SEV3: Response within 1 hour
  SEV4: Response within 24 hours

prohibited_actions:
  - Make production changes without SEV1/SEV2 incident
  - Ignore correlated alerts
  - Remediate without incident ticket
```

---

## DR Agent

### Identity

| Field | Value |
|-------|-------|
| **Agent ID** | `dr-agent` |
| **Name** | DR Readiness Agent |
| **Version** | 1.0.0 |
| **Department** | Business Continuity |
| **Risk Profile** | `state_change_prod` |
| **HITL Required** | Yes |

### Capabilities

**Primary:**
- Assess DR readiness and RTO/RPO compliance
- Generate DR runbooks and procedures
- Execute DR drills with measurement
- Simulate failover scenarios
- Execute actual failovers (with approval)

**Secondary:**
- Track DR metrics over time
- Identify DR gaps
- Generate DR compliance evidence

### Decision Logic

```
INPUT: DR readiness query or drill request
│
▼
STEP 1: ASSESS
├── get_dr_status(system/service)
├── Check replication status
├── Verify backup freshness
├── Calculate current RTO/RPO
└── Compare to targets
│
▼
STEP 2: PLAN (if drill requested)
├── generate_dr_runbook(
│     system,
│     failover_type,  # planned vs unplanned
│     include_validation=true
│   )
├── Identify pre-requisites
├── Define success criteria
└── Include rollback steps
│
▼
STEP 3: SIMULATE
├── simulate_failover(runbook, dry_run=true)
├── Validate dependencies
├── Check for blockers
└── Estimate duration
│
▼
STEP 4: EXECUTE (if approved)
├── Execute failover steps
├── Measure actual RTO/RPO
├── Validate services in DR
└── Generate evidence
│
▼
OUTPUT: DR status, runbook, or drill results
```

### Tools Used

| Tool | Risk Level | Purpose |
|------|------------|---------|
| `get_dr_status` | read_only | Current DR readiness |
| `query_assets` | read_only | DR site inventory |
| `generate_dr_runbook` | plan_only | Create drill procedures |
| `simulate_failover` | plan_only | Validate runbook |
| (future) `execute_failover` | state_change_prod | Actual failover |

### Guardrails

```yaml
constraints:
  drill_notification_required: 24h  # Advance notice for drills
  max_drill_duration: 4h           # Time limit for drills
  require_rollback_plan: true      # Must be able to fail-back

rto_rpo_targets:
  tier1_critical:
    rto: 1h
    rpo: 15m
  tier2_important:
    rto: 4h
    rpo: 1h
  tier3_standard:
    rto: 24h
    rpo: 24h

prohibited_actions:
  - Failover without communication
  - Skip fail-back validation
  - Execute unplanned DR without incident
```

---

## Cost Agent

### Identity

| Field | Value |
|-------|-------|
| **Agent ID** | `cost-agent` |
| **Name** | Cost Optimization Agent |
| **Version** | 1.0.0 |
| **Department** | FinOps |
| **Risk Profile** | `plan_only` |
| **HITL Required** | No (analysis only) |

### Capabilities

**Primary:**
- Analyze infrastructure costs
- Identify optimization opportunities
- Recommend right-sizing
- Identify unused resources
- Project cost trends

**Secondary:**
- Compare costs across environments
- Track savings from optimizations
- Generate cost reports

### Tools Used

| Tool | Risk Level | Purpose |
|------|------------|---------|
| `query_assets` | read_only | Asset inventory with sizing |
| `get_compliance_status` | read_only | Resource utilization data |
| `calculate_risk_score` | read_only | Cost-benefit analysis |

### Guardrails

```yaml
constraints:
  recommendation_only: true      # Never auto-resize
  min_observation_period: 7d     # Data before recommending

optimization_types:
  - Right-sizing (CPU, memory)
  - Reserved instance recommendations
  - Idle resource identification
  - Storage tier optimization
```

---

## Security Agent

### Identity

| Field | Value |
|-------|-------|
| **Agent ID** | `security-agent` |
| **Name** | Security Posture Agent |
| **Version** | 1.0.0 |
| **Department** | Information Security |
| **Risk Profile** | `read_only` |
| **HITL Required** | No (analysis only) |

### Capabilities

**Primary:**
- Assess security posture
- Identify vulnerabilities by CVE
- Check security configurations
- Generate security reports

**Secondary:**
- Prioritize remediation by risk
- Track vulnerability trends
- Coordinate with Patch Agent

### Tools Used

| Tool | Risk Level | Purpose |
|------|------------|---------|
| `query_assets` | read_only | Asset inventory |
| `get_drift_status` | read_only | Security-related drift |
| `check_control` | read_only | Security control validation |
| `calculate_risk_score` | read_only | Vulnerability scoring |

---

## Image Agent

### Identity

| Field | Value |
|-------|-------|
| **Agent ID** | `image-agent` |
| **Name** | Golden Image Agent |
| **Version** | 1.0.0 |
| **Department** | Platform Engineering |
| **Risk Profile** | `state_change_prod` |
| **HITL Required** | Yes |

### Capabilities

**Primary:**
- Generate image contracts (specifications)
- Create Packer templates for multi-cloud
- Generate Ansible playbooks for configuration
- Build and validate images
- Promote images through environments
- Track image versions and lineage

### Decision Logic

```
INPUT: Image build or promotion request
│
▼
STEP 1: SPECIFY
├── generate_image_contract(
│     family,
│     base_image,
│     packages,
│     configurations
│   )
├── Define compliance requirements
└── Set validation criteria
│
▼
STEP 2: GENERATE
├── generate_packer_template(
│     contract,
│     platforms=[aws, azure, gcp, vsphere]
│   )
├── generate_ansible_playbook(
│     contract,
│     hardening=true
│   )
└── Validate templates
│
▼
STEP 3: BUILD
├── build_image(template, validate=true)
├── Run security scan
├── Run compliance checks
├── Generate SBOM
└── Sign image (cosign)
│
▼
STEP 4: PROMOTE
├── promote_image(
│     image_id,
│     from_env,
│     to_env
│   )
├── Update image registry
├── Notify dependent teams
└── Update CMDB
│
▼
OUTPUT: Built image or promotion status
```

### Tools Used

| Tool | Risk Level | Purpose |
|------|------------|---------|
| `query_assets` | read_only | Find assets using image |
| `get_golden_image` | read_only | Current image specs |
| `list_image_versions` | read_only | Image version history |
| `generate_image_contract` | plan_only | Image specification |
| `generate_packer_template` | plan_only | Build template |
| `generate_ansible_playbook` | plan_only | Configuration playbook |
| `build_image` | state_change_nonprod | Build in non-prod |
| `promote_image` | state_change_prod | Promote to production |

### Guardrails

```yaml
constraints:
  require_security_scan: true    # Must pass vuln scan
  require_compliance_check: true # Must pass CIS benchmark
  require_sbom: true             # SBOM must be generated
  require_signing: true          # Must be signed
  promotion_path: dev → staging → prod

prohibited_actions:
  - Promote without validation
  - Skip security scanning
  - Build from untrusted base
```

---

## SOP Agent

### Identity

| Field | Value |
|-------|-------|
| **Agent ID** | `sop-agent` |
| **Name** | SOP Management Agent |
| **Version** | 1.0.0 |
| **Department** | Operations |
| **Risk Profile** | `state_change_prod` |
| **HITL Required** | Yes |

### Capabilities

**Primary:**
- Generate SOPs and runbooks
- Validate SOP correctness
- Simulate SOP execution
- Execute approved SOPs
- List and search existing SOPs

### Tools Used

| Tool | Risk Level | Purpose |
|------|------------|---------|
| `list_sops` | read_only | List available SOPs |
| `generate_sop` | plan_only | Create new SOP |
| `validate_sop` | plan_only | Check SOP validity |
| `simulate_sop` | plan_only | Dry-run SOP |
| `execute_sop` | state_change_prod | Run SOP |

### Guardrails

```yaml
constraints:
  require_validation: true       # Must validate before execute
  require_simulation: true       # Must simulate first
  max_sop_duration: 2h          # Time limit for SOPs

prohibited_actions:
  - Execute unvalidated SOP
  - Skip simulation step
  - Execute without approval
```

---

## Adapter Agent

### Identity

| Field | Value |
|-------|-------|
| **Agent ID** | `adapter-agent` |
| **Name** | Integration Adapter Agent |
| **Version** | 1.0.0 |
| **Department** | Integration |
| **Risk Profile** | `read_only` |
| **HITL Required** | No |

### Capabilities

**Primary:**
- Translate between QL-RF and external systems
- Normalize data from different sources
- Handle API format differences
- Coordinate cross-system queries

### Note

This agent is primarily used internally to adapt between different data sources and systems. It's not typically invoked directly by users.

---

## Base Agent

### Identity

| Field | Value |
|-------|-------|
| **Agent ID** | `base-agent` |
| **Name** | Base Agent |
| **Version** | 1.0.0 |
| **Department** | Core |
| **Risk Profile** | N/A |

### Purpose

Base Agent provides shared functionality inherited by all specialist agents:

- Common tool invocation patterns
- Standard response formatting
- Error handling
- Audit logging
- Quality scoring calculation

### Not User-Facing

This agent is not directly invoked by users. It provides the foundation that other agents build upon.

---

## Agent Selection Logic

The Meta-Prompt Engine selects agents based on intent analysis:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       AGENT SELECTION MATRIX                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  KEYWORDS/PATTERNS                              → AGENT                      │
│  ─────────────────────────────────────────────────────────────               │
│  drift, configuration, diverge, mismatch       → Drift Agent                │
│  patch, CVE, vulnerability, update, upgrade    → Patch Agent                │
│  compliance, SOC, PCI, HIPAA, audit, evidence  → Compliance Agent           │
│  incident, alert, outage, investigate          → Incident Agent             │
│  DR, disaster, failover, RTO, RPO, drill       → DR Agent                   │
│  cost, optimize, right-size, savings           → Cost Agent                 │
│  security, vulnerability, posture, scan        → Security Agent             │
│  image, golden, build, AMI, template           → Image Agent                │
│  SOP, runbook, procedure                       → SOP Agent                  │
│                                                                              │
│  AMBIGUOUS INTENTS                                                           │
│  ─────────────────                                                           │
│  If intent matches multiple agents:                                          │
│  1. Ask clarifying question                                                 │
│  2. Or select based on primary verb:                                        │
│     - "check/show/list" → read-only agent                                   │
│     - "fix/remediate/update" → state-change agent                          │
│     - "plan/generate" → planning agent                                      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Cross-Agent Collaboration

Some tasks require multiple agents working together:

### Example: Drift → Patch Collaboration

```
USER: "Fix the nginx drift on production servers"

1. META-PROMPT routes to DRIFT AGENT
   └── Analyzes drift, finds package version mismatch

2. DRIFT AGENT recognizes package update needed
   └── Coordinates with PATCH AGENT

3. PATCH AGENT generates patch plan
   └── Returns plan to DRIFT AGENT

4. DRIFT AGENT presents unified plan
   └── Single approval for combined operation
```

### Example: Incident → Drift → Patch Collaboration

```
USER: "Investigate the checkout service errors"

1. META-PROMPT routes to INCIDENT AGENT
   └── Correlates alerts, finds drift as probable cause

2. INCIDENT AGENT queries DRIFT AGENT
   └── Confirms drift on affected servers

3. INCIDENT AGENT queries PATCH AGENT
   └── Gets remediation options

4. INCIDENT AGENT presents:
   └── "Found drift correlation. Recommend patch rollforward."
```

---

## Summary

| Agent | Primary Use | Risk Level | Approval |
|-------|-------------|------------|----------|
| Drift | Fix configuration drift | state_change_prod | Required |
| Patch | Apply security patches | state_change_prod | Required |
| Compliance | Generate audit evidence | read_only | No |
| Incident | Investigate alerts | state_change_nonprod | Required |
| DR | DR readiness & drills | state_change_prod | Required |
| Cost | Cost optimization | plan_only | No |
| Security | Security assessment | read_only | No |
| Image | Golden image lifecycle | state_change_prod | Required |
| SOP | Runbook management | state_change_prod | Required |
| Adapter | System integration | read_only | No |
| Base | Shared foundation | N/A | N/A |
