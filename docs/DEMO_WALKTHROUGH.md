# QL-RF Demo Walkthrough

## Overview

This guide provides step-by-step instructions for demonstrating the QuantumLayer Resilience Fabric platform. The demo showcases AI-driven infrastructure operations across multi-cloud environments.

## Prerequisites

### Start Local Environment

```bash
# Start all services
make dev

# Verify services are running
curl http://localhost:8080/healthz  # API
curl http://localhost:8083/health   # Orchestrator

# Seed demo data
psql -d qlrf -f migrations/seed_demo_data.sql
```

### Demo Credentials

| User | Email | Role | Can Do |
|------|-------|------|--------|
| Alice Admin | admin@acme.example | Admin | Full access, approve all |
| Bob Operations | ops@acme.example | Operator | Submit tasks, view |
| Carol Viewer | viewer@acme.example | Viewer | Read-only |

**Demo Organization ID**: `11111111-1111-1111-1111-111111111111`

---

## Demo Scenarios

### Scenario 1: Drift Detection & Remediation (10 min)

**Story**: "Show me our current drift situation and fix the critical issues."

#### Step 1: View Current Drift Status

Open Control Tower: http://localhost:3000/overview

**Key Points to Highlight:**
- Multi-cloud dashboard showing AWS, Azure, GCP, vSphere, K8s
- Real-time drift percentages by platform
- Critical vs warning status indicators
- 6 drifted assets across 20 total (70% compliance)

#### Step 2: Ask AI About Drift

Navigate to AI page: http://localhost:3000/ai

**Natural Language Query:**
```
What is our current drift situation? Which assets are most critical?
```

**Expected Response:**
- AI analyzes drift across all platforms
- Identifies 6 drifted assets with severity ranking
- Highlights prod-db-001 as critical (25+ days drifted)
- Shows image version mismatches

#### Step 3: Request Remediation Plan

**Natural Language Query:**
```
Create a plan to remediate drift on critical production assets
```

**Walk Through the Plan:**
- AI generates phased remediation plan
- Shows canary → rolling → full deployment
- Calculates risk scores per asset
- Estimates RTO impact
- Requests approval before execution

#### Step 4: Approve and Execute

Click "Approve Plan" to demonstrate:
- Human-in-the-loop approval workflow
- Audit trail creation
- Execution progress tracking
- Rollback capability

---

### Scenario 2: Compliance Evidence Generation (8 min)

**Story**: "Our auditors need SOC 2 evidence by end of week."

#### Step 1: View Compliance Status

Navigate to compliance section or use AI:

**Natural Language Query:**
```
What is our current SOC 2 compliance status?
```

**Expected Response:**
- Overall compliance score
- Passing/failing controls breakdown
- Assets requiring attention

#### Step 2: Generate Evidence Package

**Natural Language Query:**
```
Generate SOC 2 compliance evidence package for CC6.1 (logical access controls) for Q4 2025
```

**Walk Through:**
- AI collects evidence across all platforms
- Generates formatted report
- Includes screenshots and logs
- Shows timestamp and attestation

#### Step 3: Show Audit Trail

Navigate to task details to show:
- Complete tool invocation history
- Every action logged with timestamps
- User attributions
- Immutable audit record

---

### Scenario 3: DR Drill Execution (12 min)

**Story**: "We need to validate our disaster recovery capability."

#### Step 1: View DR Pairs

**Natural Language Query:**
```
Show me the status of all DR pairs
```

**Expected Response:**
- 3 DR pairs configured
- US East-West: Healthy, tested 30 days ago
- EU Primary: Warning, 90 days since test
- Azure Multi-Region: Healthy, tested 14 days ago

#### Step 2: Initiate DR Drill

**Natural Language Query:**
```
Run a DR drill for the US East-West pair with 4-hour RTO target
```

**Walk Through the Plan:**
- 7-phase DR drill workflow
- Pre-check → Sync → Failover → Validation → Failback → Post-check → Report
- Target RTO: 4 hours
- Target RPO: 15 minutes

#### Step 3: Execute and Monitor

After approval, show:
- Real-time phase progression
- Heartbeat updates
- RTO/RPO measurement
- Success/failure tracking

#### Step 4: Review Results

**Natural Language Query:**
```
What were the results of the DR drill?
```

**Show:**
- Actual RTO vs Target RTO
- Actual RPO vs Target RPO
- Pairs successfully tested
- Any failures or warnings
- Executive summary for auditors

---

### Scenario 4: Patch Rollout Orchestration (10 min)

**Story**: "Security found a critical CVE affecting our Ubuntu servers."

#### Step 1: Assess Impact

**Natural Language Query:**
```
How many assets would be affected by a critical Ubuntu security patch?
```

**Expected Response:**
- Count of Ubuntu-based assets
- Breakdown by environment (prod/staging/dev)
- Current version distribution
- Risk assessment

#### Step 2: Plan Patch Rollout

**Natural Language Query:**
```
Create a patch rollout plan for Ubuntu assets, starting with staging
```

**Walk Through the Plan:**
- Phased rollout: staging → canary (10%) → full production
- Automatic health checks between phases
- Rollback triggers defined
- Maintenance window consideration

#### Step 3: Execute with Canary

Show the execution:
- First 10% of production assets updated
- Health monitoring for 30 minutes
- Automatic progression if healthy
- Manual override option

---

### Scenario 5: Cost Optimization Analysis (6 min)

**Story**: "Finance wants to reduce our cloud spend by 15%."

#### Step 1: Analyze Costs

**Natural Language Query:**
```
Analyze our cloud infrastructure for cost optimization opportunities
```

**Expected Response:**
- Right-sizing recommendations
- Unused resource identification
- Reserved instance opportunities
- Multi-cloud optimization suggestions

#### Step 2: Generate Recommendations

**Natural Language Query:**
```
What specific changes would achieve 15% cost reduction?
```

**Walk Through:**
- Prioritized list of recommendations
- Estimated savings per recommendation
- Risk assessment for each change
- Implementation complexity

---

## Technical Deep Dives

### AI Agent Architecture

Show the AI page to explain:
- 10 specialist agents (Drift, Patch, Compliance, DR, etc.)
- 29 tools with different risk levels (query, analyze, plan, execute)
- Human-in-the-loop for state-changing operations
- Tool invocation audit trail

### Multi-Cloud Support

Demonstrate connector status:
- AWS: 3 regions, SSM patching
- Azure: Update Management integration
- GCP: OS Config API
- vSphere: govmomi integration
- Kubernetes: Rolling updates, canary deployments

### Temporal Workflows

Explain workflow benefits:
- Durable execution
- Automatic retries
- Long-running operations
- Audit trail via activities

---

## Common Objections & Responses

### "What if the AI makes a mistake?"

**Response:**
1. Human-in-the-loop for all state-changing operations
2. Approval required before any execution
3. Full audit trail of every decision
4. Automatic rollback on failures
5. Quality scores gate environment deployment

### "How do we know it's doing the right thing?"

**Response:**
1. Plans are generated and shown before execution
2. Every tool invocation is logged
3. Plans include risk assessment
4. Dry-run mode available
5. Complete explainability of AI reasoning

### "What about security?"

**Response:**
1. RBAC controls who can approve what
2. Dual approval for critical operations
3. JWT-based authentication
4. Complete audit trail (7-year retention)
5. No credentials stored in AI context

---

## Demo Reset

To reset the demo environment:

```bash
# Reset database
make migrate-down
make migrate-up

# Re-seed demo data
psql -d qlrf -f migrations/seed_demo_data.sql

# Verify
curl http://localhost:8080/api/v1/assets | jq '.total'
# Should return: 20
```

---

## Demo Environment URLs

| Service | URL | Purpose |
|---------|-----|---------|
| Control Tower | http://localhost:3000 | Main UI |
| API | http://localhost:8080 | REST API |
| Orchestrator | http://localhost:8083 | AI API |
| Temporal UI | http://localhost:8088 | Workflow monitoring |
| PostgreSQL | localhost:5432 | Database |

---

## Quick Reference: AI Queries

### Drift
- "What is our current drift situation?"
- "Show me assets drifted more than 14 days"
- "Remediate drift on staging web servers"

### Compliance
- "What is our SOC 2 compliance status?"
- "Run CIS benchmark audit on production"
- "Generate compliance evidence for HIPAA"

### DR/BCP
- "Show DR pair status"
- "Run DR drill for US East-West"
- "What is our current RTO/RPO?"

### Patching
- "Which assets need security patches?"
- "Plan patch rollout for critical CVE"
- "Show patch compliance across all platforms"

### Images
- "Show golden image versions"
- "Promote Ubuntu 2.5.0 to production"
- "Which assets are using deprecated images?"

### Cost
- "Analyze infrastructure for cost savings"
- "Find unused resources"
- "Right-sizing recommendations for production"

---

## Appendix: Demo Data Summary

| Category | Count | Notes |
|----------|-------|-------|
| Organizations | 1 | Acme Corporation |
| Users | 3 | Admin, Operator, Viewer |
| Sites | 8 | Multi-cloud + on-prem |
| DR Pairs | 3 | Various health states |
| Golden Images | 10 | Ubuntu, Amazon Linux, Windows, Containers |
| Assets | 20 | Mix of compliant/drifted |
| Drifted Assets | 6 | ~30% non-compliant |
| Alerts | 6 | Critical/Warning/Info |
| Compliance Frameworks | 5 | CIS, SLSA, SOC2, HIPAA, PCI-DSS |
