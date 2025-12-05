# QL-RF Feature Roadmap: Enterprise Use Cases

## Executive Summary

QL-RF's architecture follows a **Detect → Correlate → Assess → Plan → Execute → Validate** pattern that naturally extends to 15+ high-value enterprise use cases. This document maps priority use cases to existing components and identifies new agents/tools needed.

---

## Current Architecture Inventory

### Existing Agents (10)

| Agent | Task Types | Description |
|-------|------------|-------------|
| DriftAgent | drift_remediation | Detects and remediates configuration drift |
| PatchAgent | patch_rollout | Orchestrates patch deployment across platforms |
| ComplianceAgent | compliance_audit | Runs compliance checks and generates evidence |
| IncidentAgent | incident_investigation | Investigates and responds to incidents |
| DRAgent | dr_drill | Manages DR drills and failover operations |
| CostAgent | cost_optimization | Identifies cost optimization opportunities |
| SecurityAgent | security_scan | Performs security scanning and assessment |
| ImageAgent | image_management | Manages golden image lifecycle |
| SOPAgent | sop_authoring | Generates and validates SOPs |
| AdapterAgent | terraform_generation | Generates infrastructure-as-code |

### Existing Tools (29)

**Query Tools (Read-Only)**
- `query_assets` - Query assets with filters
- `get_drift_status` - Get drift analysis
- `get_compliance_status` - Get compliance posture
- `get_golden_image` - Get golden image details
- `query_alerts` - Query active alerts
- `get_dr_status` - Get DR readiness status

**Analysis Tools**
- `analyze_drift` - Deep drift analysis
- `check_control` - Check specific compliance control

**Planning Tools**
- `compare_versions` - Compare image versions
- `generate_patch_plan` - Generate patch rollout plan
- `generate_rollout_plan` - Generate phased rollout
- `generate_dr_runbook` - Generate DR runbook
- `simulate_rollout` - Simulate deployment
- `calculate_risk_score` - Calculate risk score
- `simulate_failover` - Simulate DR failover
- `generate_compliance_evidence` - Generate audit evidence

**Execution Tools**
- `propose_rollout` - Propose a rollout for approval
- `acknowledge_alert` - Acknowledge alerts

**Image Tools**
- `generate_image_contract` - Generate image contract
- `generate_packer_template` - Generate Packer template
- `generate_ansible_playbook` - Generate Ansible playbook
- `build_image` - Trigger image build
- `list_image_versions` - List image versions
- `promote_image` - Promote image status

**SOP Tools**
- `generate_sop` - Generate SOP document
- `validate_sop` - Validate SOP structure
- `simulate_sop` - Dry-run SOP
- `execute_sop` - Execute SOP steps
- `list_sops` - List available SOPs

---

## Priority Use Cases (Top 3)

### 1. Certificate Expiry → Impact → Rotation Automation ✅ COMPLETE

**Status**: Fully implemented as of December 2025

**Business Value**: Prevent outages from expired certificates (very common enterprise issue)

**Flow**:
```
Detect expiring cert → Trace usage (lineage) → Compute blast radius →
Auto-generate renewal plan → Phase rollout → Validate TLS handshake
```

**Implementation Summary**:

| Component | Status | Location |
|-----------|--------|----------|
| Database Schema | ✅ | `migrations/000015_certificates.up.sql` |
| API Handlers | ✅ | `services/api/internal/handlers/certificate.go` |
| Repository Layer | ✅ | `services/api/internal/repository/certificate.go` |
| OpenAPI Contract | ✅ | `contracts/api/certificates.yaml` |
| UI Dashboard | ✅ | `ui/control-tower/src/app/(dashboard)/certificates/` |
| UI Components | ✅ | `ui/control-tower/src/components/certificates/` |

**AI Orchestrator Tools** (6 tools in `services/orchestrator/internal/tools/certificate_tools.go`):

| Tool | Description | Risk Level |
|------|-------------|------------|
| `list_certificates` | List certificates with filters | read_only |
| `get_certificate_details` | Get certificate metadata (expiry, issuer, SANs) | read_only |
| `map_certificate_usage` | Map where a cert is used (blast radius) | read_only |
| `generate_cert_renewal_plan` | Generate renewal plan | plan_only |
| `propose_cert_rotation` | Execute certificate rotation | state_change_prod |
| `validate_tls_handshake` | Verify TLS works post-rotation | read_only |

**Database Tables** (Migration 000015):
- `certificates` - SSL/TLS certificates from all platforms
- `certificate_usages` - Where certificates are deployed (blast radius tracking)
- `certificate_rotations` - Rotation history and status
- `certificate_alerts` - Expiry and security alerts

**API Endpoints**:
- `GET /api/v1/certificates` - List with filters
- `GET /api/v1/certificates/{id}` - Certificate details
- `GET /api/v1/certificates/summary` - Aggregate stats
- `GET /api/v1/certificates/{id}/usage` - Usage/blast radius
- `GET /api/v1/certificates/rotations` - Rotation history
- `GET /api/v1/certificates/alerts` - Expiry alerts
- `POST /api/v1/certificates/alerts/{id}/acknowledge` - Acknowledge alert

---

### 2. Secret Leakage → Impact → Containment → Rotation

**Business Value**: CISOs will pay premium for automated secret rotation

**Flow**:
```
Detect secret leak → Identify all workloads using secret →
Auto-generate rotation plan → Restart services safely → Validate connectivity
```

**Existing Components Used**:
| Component | Usage |
|-----------|-------|
| IncidentAgent | Investigate leak source |
| SecurityAgent | Assess security impact |
| query_assets | Find affected workloads |
| generate_rollout_plan | Phase service restarts |
| simulate_rollout | Validate no breaking changes |

**New Components Required**:

| Type | Name | Description | Risk Level |
|------|------|-------------|------------|
| Tool | `scan_secrets` | Scan for secrets in env vars, K8s secrets, param stores | read_only |
| Tool | `map_secret_usage` | Map where a secret is used | read_only |
| Tool | `generate_secret_rotation_plan` | Generate rotation plan | plan_only |
| Tool | `rotate_secret` | Rotate secret in vault/param store | state_change_prod |
| Tool | `restart_workloads` | Rolling restart workloads | state_change_prod |
| Tool | `validate_connectivity` | Verify services can connect post-rotation | read_only |
| Agent | **SecretRotationAgent** | Orchestrates secret lifecycle | - |

**New Task Type**: `secret_rotation`

**Database Tables Needed**:
```sql
CREATE TABLE secrets_inventory (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL,
    secret_name VARCHAR(255),
    secret_type VARCHAR(50), -- api_key, db_password, token, certificate
    storage_location VARCHAR(50), -- vault, aws_ssm, azure_keyvault, k8s_secret, env_var
    storage_ref VARCHAR(255),
    platform VARCHAR(20),
    last_rotated_at TIMESTAMPTZ,
    rotation_policy_days INT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE secret_usage (
    id UUID PRIMARY KEY,
    secret_id UUID REFERENCES secrets_inventory(id),
    asset_id UUID REFERENCES assets(id),
    usage_type VARCHAR(50), -- env_var, mounted_secret, config_map
    usage_ref VARCHAR(255),
    discovered_at TIMESTAMPTZ DEFAULT NOW()
);
```

**Estimated Effort**: ~2 weeks (65% reuse)

---

### 3. Cost Anomaly → Impact → Rightsizing Plan

**Business Value**: Sellable to both tech AND finance teams

**Flow**:
```
Cost spike detected → AI explains root cause → Map affected resources →
Generate rightsizing/cleanup plan → Approval → Apply → Validate
```

**Existing Components Used**:
| Component | Usage |
|-----------|-------|
| CostAgent | Already handles cost optimization |
| query_assets | Query resources by cost attributes |
| calculate_risk_score | Assess impact of changes |
| propose_rollout | HITL approval for changes |

**New Components Required**:

| Type | Name | Description | Risk Level |
|------|------|-------------|------------|
| Tool | `get_cost_anomalies` | Detect cost anomalies | read_only |
| Tool | `explain_cost_spike` | AI-powered root cause analysis | read_only |
| Tool | `get_resource_utilization` | Get CPU/memory/disk metrics | read_only |
| Tool | `generate_rightsizing_plan` | Generate rightsizing recommendations | plan_only |
| Tool | `resize_instance` | Execute instance resize | state_change_prod |
| Tool | `cleanup_unused_resources` | Delete unused EBS, snapshots, etc. | state_change_prod |

**New Task Type**: `cost_anomaly_resolution`

**Database Tables Needed**:
```sql
CREATE TABLE cost_anomalies (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL,
    detected_at TIMESTAMPTZ NOT NULL,
    anomaly_type VARCHAR(50), -- spike, unusual_pattern, budget_breach
    resource_id VARCHAR(255),
    resource_type VARCHAR(100),
    platform VARCHAR(20),
    baseline_cost DECIMAL(12,2),
    actual_cost DECIMAL(12,2),
    deviation_percent DECIMAL(5,2),
    root_cause TEXT,
    status VARCHAR(20) DEFAULT 'open',
    resolved_at TIMESTAMPTZ
);
```

**Estimated Effort**: ~1.5 weeks (80% reuse - CostAgent exists)

---

## Medium Priority Use Cases (Next 4)

### 4. Configuration Drift → Risk → Real-Time Impact → Auto-Remediation

**Extends existing**: DriftAgent + get_drift_status + analyze_drift

**New Tools Needed**:
- `get_realtime_signals` - Query Prometheus/CloudWatch for impact metrics
- `generate_drift_remediation_plan` - Auto-fix plan
- `apply_drift_fix` - Execute remediation

**Estimated Effort**: ~1 week (85% reuse)

---

### 5. Resource Overload Prediction → Impact → Auto-Scaling Plan

**New Agent**: **CapacityPlanningAgent**

**New Tools Needed**:
- `get_capacity_metrics` - Current CPU/memory/disk trends
- `predict_resource_exhaustion` - ML-based prediction
- `generate_scaling_plan` - HPA, node pool, instance resize
- `execute_scaling_action` - Apply scaling changes

**Estimated Effort**: ~2 weeks (60% reuse)

---

### 6. Compliance Misconfig → Blast Radius Mapping

**Extends existing**: ComplianceAgent + check_control + generate_compliance_evidence

**New Tools Needed**:
- `map_misconfig_blast_radius` - Lineage-based impact mapping
- `generate_remediation_priority` - Risk-ranked fix order
- `apply_compliance_fix` - Auto-remediate misconfigs

**Estimated Effort**: ~1 week (85% reuse)

---

### 7. Dependency Outage → Real-Time Service Impact Mapping

**New Agent**: **DependencyImpactAgent**

**New Tools Needed**:
- `build_dependency_graph` - Service-to-service dependencies
- `detect_outage` - Monitor health of dependencies
- `map_outage_impact` - Which services affected
- `generate_failover_plan` - Failover/traffic shift plan

**Estimated Effort**: ~2 weeks (50% reuse - needs dependency discovery)

---

## Lower Priority Use Cases (Remaining 3)

### 8. K8s Deployment Failure → Auto Triage → Remediation
- Reuses: IncidentAgent, query_assets
- New: `diagnose_k8s_failure`, `suggest_k8s_fix`, `rollback_deployment`
- Effort: ~1 week

### 9. End-of-Life OS/Library → Impact → Migration Plan
- Reuses: ImageAgent, ComplianceAgent, SBOM
- New: `check_eol_status`, `map_eol_impact`, `generate_migration_plan`
- Effort: ~1.5 weeks

### 10. DR Posture Degradation → Automated DR Drills
- Reuses: DRAgent, get_dr_status, simulate_failover
- New: `detect_dr_degradation`, `schedule_dr_drill`, `generate_dr_evidence`
- Effort: ~1 week

---

## Implementation Roadmap

### Phase 1: Quick Wins (Weeks 1-3)
1. **Cost Anomaly Resolution** - 80% reuse, immediate CFO appeal
2. **Configuration Drift Enhancement** - 85% reuse, extends existing

### Phase 2: High Value (Weeks 4-7)
3. **Certificate Rotation** - Zero competition, high pain point
4. **Secret Rotation** - CISO must-have

### Phase 3: Platform Expansion (Weeks 8-11)
5. **Capacity Planning** - CapacityPlanningAgent
6. **Compliance Remediation** - Audit-driven demand
7. **Dependency Impact** - SRE essential

### Phase 4: Operational Excellence (Weeks 12-14)
8. **K8s Failure Triage** - DevOps favorite
9. **EOL Migration Planning** - Enterprise modernization
10. **DR Automation** - Premium feature

---

## New Agents Summary

| Agent | Task Type | Priority | Effort |
|-------|-----------|----------|--------|
| CertificateAgent | certificate_rotation | P1 | 2 weeks |
| SecretRotationAgent | secret_rotation | P1 | 2 weeks |
| CapacityPlanningAgent | capacity_planning | P2 | 2 weeks |
| DependencyImpactAgent | dependency_analysis | P2 | 2 weeks |

---

## New Tools Summary (30 tools → ~45 tools)

| Category | New Tools Count | Names |
|----------|-----------------|-------|
| Certificate | 6 | scan_certificates, get_cert_details, map_cert_usage, generate_cert_renewal_plan, rotate_cert, validate_tls_handshake |
| Secrets | 6 | scan_secrets, map_secret_usage, generate_secret_rotation_plan, rotate_secret, restart_workloads, validate_connectivity |
| Cost | 4 | get_cost_anomalies, explain_cost_spike, get_resource_utilization, resize_instance |
| Capacity | 4 | get_capacity_metrics, predict_resource_exhaustion, generate_scaling_plan, execute_scaling_action |
| Dependencies | 4 | build_dependency_graph, detect_outage, map_outage_impact, generate_failover_plan |

---

## Database Schema Additions

```sql
-- Certificate Management
CREATE TABLE certificates (...);
CREATE TABLE certificate_usage (...);

-- Secret Management
CREATE TABLE secrets_inventory (...);
CREATE TABLE secret_usage (...);

-- Cost Management (extends existing)
CREATE TABLE cost_anomalies (...);

-- Dependency Graph
CREATE TABLE service_dependencies (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL,
    source_asset_id UUID REFERENCES assets(id),
    target_asset_id UUID REFERENCES assets(id),
    dependency_type VARCHAR(50), -- database, api, queue, cache
    discovered_at TIMESTAMPTZ DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## Connector Enhancements

### AWS Connector
- Certificate Manager (ACM) scanning
- Secrets Manager integration
- Cost Explorer anomaly detection
- CloudWatch metrics for capacity

### Azure Connector
- Key Vault certificate scanning
- Key Vault secrets integration
- Cost Management anomaly API
- Azure Monitor metrics

### GCP Connector
- Certificate Manager scanning
- Secret Manager integration
- Billing anomaly detection
- Cloud Monitoring metrics

### Kubernetes Connector
- TLS secrets scanning
- Secret references in pods
- Resource metrics (metrics-server)
- Service dependency discovery

### vSphere Connector
- Certificate stores scanning
- Guest customization secrets
- vRealize Operations metrics

---

## Success Metrics

| Use Case | Metric | Target |
|----------|--------|--------|
| Certificate Rotation | MTTR for cert expiry | < 1 hour |
| Secret Rotation | Time to containment | < 30 minutes |
| Cost Anomaly | Anomaly detection time | < 15 minutes |
| Drift Remediation | Auto-fix rate | > 80% |
| Capacity Planning | Prediction accuracy | > 90% |

---

## Competitive Moat

No vendor currently provides:
1. **Cross-platform coverage** (AWS + Azure + GCP + vSphere + K8s)
2. **Full lifecycle automation** (Detect → Plan → Execute → Validate)
3. **Unified lineage/dependency graph** across infrastructure
4. **AI-driven explanation and planning**
5. **Human-in-the-loop safety** for production changes

QL-RF is positioned to own the **AI-Driven Infrastructure Resilience** category.
