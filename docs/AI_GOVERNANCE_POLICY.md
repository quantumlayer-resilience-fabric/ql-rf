# AI Governance Policy

## Document Information

| Field | Value |
|-------|-------|
| **Document ID** | POL-AI-001 |
| **Version** | 1.0 |
| **Effective Date** | 2024-12-01 |
| **Review Frequency** | Annual |
| **Owner** | CTO / VP Engineering |
| **Approver** | Executive Leadership |

---

## Table of Contents

1. [Purpose](#purpose)
2. [Scope](#scope)
3. [AI Usage Policy](#ai-usage-policy)
4. [Approval Authority Matrix](#approval-authority-matrix)
5. [Audit & Logging Requirements](#audit--logging-requirements)
6. [Data Retention Policy](#data-retention-policy)
7. [Incident Classification](#incident-classification)
8. [Cost Governance](#cost-governance)
9. [Model Selection & Validation](#model-selection--validation)
10. [Prompt Change Control](#prompt-change-control)
11. [Compliance & Regulatory](#compliance--regulatory)
12. [Policy Exceptions](#policy-exceptions)
13. [Policy Violations](#policy-violations)

---

## Purpose

This policy establishes governance requirements for the use of AI capabilities within QL-RF. It ensures:

- **Safety**: AI operations do not cause unintended harm
- **Accountability**: Clear ownership for AI-driven decisions
- **Transparency**: All AI actions are auditable
- **Compliance**: AI usage meets regulatory requirements
- **Cost Control**: AI usage stays within budgets

---

## Scope

This policy applies to:

- All QL-RF AI agents and their operations
- All users interacting with AI capabilities
- All environments (development, staging, production)
- All integrations using AI-generated outputs

This policy does NOT apply to:

- Non-AI features of QL-RF
- Third-party AI services not integrated with QL-RF

---

## AI Usage Policy

### 3.1 Permitted Uses

AI capabilities MAY be used for:

| Use Case | Approval | Notes |
|----------|----------|-------|
| Querying asset status | Auto-approved | Read-only |
| Generating reports | Auto-approved | Read-only |
| Analyzing drift | Auto-approved | Read-only |
| Generating plans | Auto-approved | Plan review required |
| Non-production changes | Single approval | Engineer+ |
| Production changes | Single/Dual approval | Based on risk |
| DR drills | Dual approval | With advance notice |
| Emergency remediation | Override process | Admin + MFA |

### 3.2 Prohibited Uses

AI capabilities MUST NOT be used for:

| Prohibited Use | Reason |
|----------------|--------|
| Direct database modifications | Bypass data governance |
| Credential/secret handling | Security risk |
| Financial transactions | Requires human authorization |
| Customer data access without audit | Privacy compliance |
| Circumventing approval workflows | Policy violation |
| Training on customer data | Privacy compliance |

### 3.3 Human-in-the-Loop Requirements

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    HITL REQUIREMENT MATRIX                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  RISK LEVEL              HITL REQUIREMENT           APPROVAL TIMEOUT         │
│  ─────────────────       ─────────────────          ─────────────────        │
│  read_only               None                       N/A                      │
│  plan_only               None (plan review)         N/A                      │
│  state_change_nonprod    Single approval            24 hours                 │
│  state_change_prod       Single approval            24 hours                 │
│  state_change_prod       Dual approval              24 hours                 │
│  (critical systems)      (both required)                                     │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Approval Authority Matrix

### 4.1 Role-Based Approval Authority

| Role | Can Approve | Cannot Approve |
|------|-------------|----------------|
| **Viewer** | Nothing | All changes |
| **Operator** | Nothing (can submit) | All changes |
| **Engineer** | Non-prod changes, Standard prod changes | Critical systems, Emergency overrides |
| **Admin** | All changes including critical | N/A (highest authority) |

### 4.2 Task-Specific Authority

| Task Type | Minimum Approver | Additional Requirements |
|-----------|-----------------|------------------------|
| Drift remediation (non-prod) | Engineer | None |
| Drift remediation (prod) | Engineer | Dual approval if >20 assets |
| Patch rollout (non-prod) | Engineer | None |
| Patch rollout (prod) | Engineer | Dual approval if critical CVE |
| DR drill | Engineer | 24h advance notice |
| DR failover (actual) | Admin | Incident ticket required |
| Image promotion to prod | Engineer | Security scan passed |
| Emergency override | Admin | MFA + justification |

### 4.3 Separation of Duties

| Constraint | Requirement |
|------------|-------------|
| Self-approval | Prohibited for production changes |
| Dual approval | Two different individuals required |
| Approver availability | At least 2 approvers should be available |

---

## Audit & Logging Requirements

### 5.1 What Must Be Logged

| Event | Required Fields | Retention |
|-------|-----------------|-----------|
| Task submission | Intent, user, timestamp, context | 7 years |
| Plan generation | Plan details, quality score, agent | 7 years |
| Approval/Rejection | Decision, approver, notes, timestamp | 7 years |
| Tool invocation | Tool, parameters, result, duration | 7 years |
| Execution status | Phase, assets, status, errors | 7 years |
| Rollback | Trigger, affected assets, outcome | 7 years |
| Override | Justification, approver, MFA status | Permanent |

### 5.2 Audit Log Integrity

| Requirement | Implementation |
|-------------|----------------|
| Immutability | Audit logs cannot be modified or deleted |
| Completeness | All AI actions must be logged |
| Timestamp accuracy | NTP-synchronized timestamps |
| Access control | Read-only access, Admin only |

### 5.3 Audit Access

| Role | Access Level |
|------|-------------|
| Viewer | Own tasks only |
| Operator | Own tasks only |
| Engineer | Team tasks |
| Admin | All tasks |
| Auditor | All tasks (external) |

---

## Data Retention Policy

### 6.1 Retention Periods

| Data Type | Retention Period | Justification |
|-----------|-----------------|---------------|
| AI task records | 7 years | Compliance (SOX, PCI) |
| Plan details | 7 years | Audit trail |
| Tool invocation logs | 7 years | Forensics capability |
| Execution logs | 7 years | Incident investigation |
| LLM prompts/responses | 3 years | Model behavior analysis |
| Performance metrics | 2 years | Trend analysis |
| Emergency overrides | Permanent | Security audit |

### 6.2 Data Deletion

| Requirement | Policy |
|-------------|--------|
| Customer request | Within 30 days (subject to legal holds) |
| Retention expiry | Automated purge at end of period |
| Legal hold | Preserved until hold released |

---

## Incident Classification

### 7.1 AI-Caused Incident Severity

| Severity | Definition | Response Time | Escalation |
|----------|------------|---------------|------------|
| **SEV-1** | Production outage, customer impact, data loss | Immediate | CEO, CISO |
| **SEV-2** | Partial outage, significant degradation | 15 minutes | VP Eng, On-call |
| **SEV-3** | Service degradation, no customer impact | 1 hour | Team lead |
| **SEV-4** | Minor issue, caught before impact | 24 hours | Regular triage |

### 7.2 Incident Response Requirements

| Severity | Post-Mortem Required | Timeline |
|----------|---------------------|----------|
| SEV-1 | Yes (full RCA) | 5 business days |
| SEV-2 | Yes (full RCA) | 10 business days |
| SEV-3 | Yes (abbreviated) | 15 business days |
| SEV-4 | Optional | As needed |

### 7.3 Mandatory Actions After AI Incidents

| Action | SEV-1 | SEV-2 | SEV-3 | SEV-4 |
|--------|-------|-------|-------|-------|
| Pause related AI tasks | ✓ | ✓ | Optional | No |
| Security review | ✓ | ✓ | Optional | No |
| Policy review | ✓ | ✓ | Optional | No |
| Agent constraint update | If needed | If needed | If needed | No |
| Customer notification | If impacted | If impacted | No | No |

---

## Cost Governance

### 8.1 Token Budget Structure

| Level | Monthly Budget | Alert Thresholds |
|-------|---------------|------------------|
| Organization | Set by Admin | 75%, 90%, 100% |
| Project | Allocated from Org | 75%, 90%, 100% |
| User | Optional limit | 90%, 100% |

### 8.2 Budget Enforcement

| Threshold | Action |
|-----------|--------|
| 75% | Warning notification to Admin |
| 90% | Alert to Admin + Finance |
| 100% | New state-change tasks blocked |
| 120% | All AI functions disabled |

### 8.3 Budget Override Process

| Requirement | Process |
|-------------|---------|
| Increase request | Submit to Finance with justification |
| Approval authority | Finance Manager + VP Eng |
| SLA | 24-hour response |
| Documentation | Budget change logged |

### 8.4 Cost Optimization Requirements

| Requirement | Frequency |
|-------------|-----------|
| Cost review | Monthly |
| Optimization recommendations | Quarterly |
| Model efficiency analysis | Semi-annually |

---

## Model Selection & Validation

### 9.1 Approved LLM Providers

| Provider | Status | Use Cases |
|----------|--------|-----------|
| Anthropic Claude | Approved | Primary reasoning |
| Azure OpenAI | Approved | Backup/specific tasks |
| OpenAI | Approved | Backup |

### 9.2 Model Selection Criteria

| Criterion | Weight | Evaluation |
|-----------|--------|------------|
| Accuracy | 30% | Benchmark testing |
| Safety | 25% | Red team testing |
| Cost efficiency | 20% | Token economics |
| Latency | 15% | Response time |
| Compliance | 10% | Data residency, certifications |

### 9.3 Model Validation Requirements

| Requirement | Frequency |
|-------------|-----------|
| Accuracy benchmarks | Monthly |
| Safety testing | Quarterly |
| Bias evaluation | Semi-annually |
| Cost analysis | Monthly |

### 9.4 Model Change Process

| Step | Requirement |
|------|-------------|
| Proposal | Document rationale |
| Testing | Full regression in non-prod |
| Security review | Security team sign-off |
| Approval | VP Eng approval |
| Rollout | Gradual (canary → full) |

---

## Prompt Change Control

### 10.1 Prompt Versioning

| Requirement | Implementation |
|-------------|----------------|
| Version control | All prompts in Git |
| Change tracking | Full history preserved |
| Rollback capability | Any version recoverable |

### 10.2 Prompt Change Process

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    PROMPT CHANGE WORKFLOW                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. PROPOSE                                                                  │
│     ├── Create PR with prompt change                                        │
│     ├── Document rationale                                                  │
│     └── Include test cases                                                  │
│                                                                              │
│  2. REVIEW                                                                   │
│     ├── Technical review (Engineer)                                         │
│     ├── Safety review (Security)                                            │
│     └── Business review (if behavior change)                                │
│                                                                              │
│  3. TEST                                                                     │
│     ├── Unit tests pass                                                     │
│     ├── Regression tests pass                                               │
│     └── Staging validation                                                  │
│                                                                              │
│  4. APPROVE                                                                  │
│     ├── 2 reviewers required                                                │
│     └── Security sign-off for safety-related                                │
│                                                                              │
│  5. DEPLOY                                                                   │
│     ├── Canary deployment first                                             │
│     ├── Monitor quality scores                                              │
│     └── Full rollout if stable                                              │
│                                                                              │
│  6. MONITOR                                                                  │
│     ├── Watch for quality degradation                                       │
│     ├── Track rejection rate changes                                        │
│     └── Ready to rollback if issues                                         │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 10.3 Emergency Prompt Changes

| Requirement | Policy |
|-------------|--------|
| Justification | Security or critical bug only |
| Approval | Security Lead + VP Eng |
| Testing | Abbreviated (safety only) |
| Documentation | Within 24 hours post-change |

---

## Compliance & Regulatory

### 11.1 Applicable Regulations

| Regulation | Applicability | Key Requirements |
|------------|--------------|------------------|
| SOC 2 | All deployments | Audit trails, access control |
| PCI-DSS | Payment systems | Data handling, logging |
| HIPAA | Healthcare | PHI protection |
| GDPR | EU data | Data subject rights |
| SOX | Public companies | Financial controls |

### 11.2 AI-Specific Compliance

| Requirement | Implementation |
|-------------|----------------|
| Explainability | All AI decisions logged with reasoning |
| Human oversight | HITL for state-changing operations |
| Audit trail | Complete, immutable logs |
| Right to explanation | Decision audit available to users |

### 11.3 Data Residency

| Data Type | Residency Requirement |
|-----------|----------------------|
| Customer data | Customer-specified region |
| AI prompts/responses | Same region as customer data |
| Audit logs | Same region as customer data |
| Aggregated metrics | May be global |

---

## Policy Exceptions

### 12.1 Exception Request Process

| Step | Requirement |
|------|-------------|
| Document | Written exception request |
| Justify | Business need explained |
| Risk assess | Security review completed |
| Approve | Risk Owner + CISO |
| Time limit | Maximum 90 days |
| Review | Mandatory renewal review |

### 12.2 Exception Documentation

| Field | Required |
|-------|----------|
| Exception ID | Yes |
| Requestor | Yes |
| Justification | Yes |
| Risk assessment | Yes |
| Compensating controls | Yes |
| Expiration date | Yes |
| Review date | Yes |
| Approvers | Yes |

---

## Policy Violations

### 13.1 Violation Categories

| Category | Examples | Severity |
|----------|----------|----------|
| **Critical** | Bypassing approval, data exfiltration | Immediate action |
| **Major** | Repeated unauthorized access attempts | Investigation |
| **Minor** | Procedural non-compliance | Coaching |

### 13.2 Violation Response

| Severity | Action |
|----------|--------|
| Critical | Immediate access suspension, security investigation |
| Major | Access review, manager notification, remediation plan |
| Minor | Documented warning, training requirement |

### 13.3 Reporting Violations

| Requirement | Policy |
|-------------|--------|
| Mandatory reporting | All suspected violations |
| Reporting channel | security@company.com or anonymous hotline |
| Non-retaliation | Whistleblower protection applies |

---

## Policy Review & Updates

| Activity | Frequency | Owner |
|----------|-----------|-------|
| Full policy review | Annual | CTO |
| Incident-triggered review | As needed | Security |
| Regulatory update review | Quarterly | Compliance |
| Stakeholder feedback | Ongoing | Policy Owner |

---

## Approval

| Role | Name | Date | Signature |
|------|------|------|-----------|
| CTO | _________________ | ____/____/____ | _____________ |
| CISO | _________________ | ____/____/____ | _____________ |
| VP Engineering | _________________ | ____/____/____ | _____________ |
| General Counsel | _________________ | ____/____/____ | _____________ |

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2024-12-01 | AI Governance Team | Initial policy |
