# QuantumLayer Resilience Fabric (QL-RF)
## Product Requirements Document v1.0

**AI-Powered Infrastructure Resilience & Compliance Platform**

| Field | Value |
|-------|-------|
| Version | 1.0 |
| Date | December 2025 |
| Author | Subrahmanya Satish Gonella |
| Status | Draft — Ready for Architectural Review |

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Product Vision](#2-product-vision)
3. [Problem Statement](#3-problem-statement)
4. [Goals and Non-Goals](#4-goals-and-non-goals)
5. [Target Users](#5-target-users-and-personas)
6. [Competitive Landscape](#6-competitive-landscape)
7. [System Architecture](#7-system-architecture)
8. [Core Features](#8-core-features-and-components)
9. [Data Contracts](#9-data-contracts-and-schemas)
10. [MVP Scope](#10-mvp-scope-30-days)
11. [Phase Plan](#11-phase-plan-and-roadmap)
12. [Technical Architecture](#12-technical-architecture)
13. [Identity & RBAC](#13-identity-and-rbac)
14. [Compliance & Evidence](#14-compliance-and-evidence-packs)
15. [Windows Baseline](#15-windows-baseline-support)
16. [Air-Gapped Mode](#16-air-gapped-and-regulated-site-support)
17. [Networking & Failover](#17-networking-and-failover-adapters)
18. [FinOps](#18-finops-and-capacity-considerations)
19. [KPIs](#19-kpis-and-success-metrics)
20. [Risks](#20-risks-and-mitigations)
21. [Monetization](#21-monetization-strategy)
22. [ADRs](#22-architectural-decision-records)
23. [Execution Plan](#23-execution-plan)

---

## 1. Executive Summary

QuantumLayer Resilience Fabric (QL-RF) is an **AI-powered Infrastructure Resilience and Compliance Platform** that provides real-time visibility, automation, and control over golden images, patch drift, compliance, and disaster-recovery posture across multi-cloud and data-centre environments.

### Core Value Proposition

- **Single control tower** for fleet drift, patch parity, and DR readiness
- **Platform-agnostic:** AWS, Azure, GCP, VMware vSphere, bare metal, Kubernetes
- **Contracts-first:** versioned YAML contracts for images, provisioning, and validation
- **AI-assisted:** CVE triage, canary analysis, RCA generation, and predictive risk alerts
- **Audit-ready:** SBOM, SLSA provenance, and automated compliance evidence

### Strategic Objective

Transform QuantumLayer from a multi-cloud orchestration fabric into a **resilience & compliance fabric**—a capability that neither Terraform, VMware Aria, nor ServiceNow ITOM provide holistically today.

---

## 2. Product Vision

> *"From drift to resilience—one control tower to rule all clouds."*

The vision of QL-RF is to make resilience **measurable**, **observable**, and **automatable**—to transform ad-hoc infrastructure patching and DR testing into a continuously validated process.

### Core Principles

| Principle | Description |
|-----------|-------------|
| **Platform-Agnostic** | Works across public cloud and private data centres without vendor lock-in |
| **Contracts-First** | Uses versioned YAML contracts for images, provisioning, and validation rules |
| **Event-Driven** | Integrates with QuantumLayer through clean event streams (Kafka or Redis Streams) |
| **AI-Assisted** | Uses LLMs for anomaly summarisation, RCA generation, and predictive risk alerts |
| **Non-Disruptive** | Starts read-only; evolves to partial automation safely with policy gates |

---

## 3. Problem Statement

Today, most enterprise environments suffer from:

### Fragmented Image Lifecycle Management
Each cloud or DC handles golden images differently. No unified registry, versioning, or attestation across platforms.

### Patch Drift and Configuration Sprawl
Systems gradually diverge from baselines. Patch levels vary across regions, environments, and DR sites.

### No Unified Visibility
No central dashboard to view fleet patch health and resilience status across hybrid infrastructure.

### Manual DR Tests
Failover simulations are infrequent, poorly validated, and often fail when needed most.

### High Compliance Burden
Manual audits, inconsistent evidence collection, and poor traceability. Auditors require reproducible artefacts (SBOMs, provenance).

> **QL-RF solves these** by creating a single control tower that observes, validates, and eventually enforces image and resilience hygiene across the enterprise estate.

---

## 4. Goals and Non-Goals

### Goals (Phase 1–3)

- [ ] Build a read-only inventory and drift dashboard across clouds and DCs
- [ ] Provide contracts for golden images, provisioning, and compliance validation
- [ ] Enable automated drift detection and summarised anomaly insights
- [ ] Deliver AI-driven recommendations for remediation and risk scoring
- [ ] Integrate with QuantumLayer via event contracts and provisioning hooks
- [ ] Support BCP/DR orchestration with drill automation and RTO/RPO tracking

### Non-Goals (v1.0)

- Backup/restore management (partner with Veeam, Zerto instead)
- Application-level DR (focus on infra layer first)
- Custom hardware integration beyond VMware/Bare Metal
- Direct patch application in v1 (deferred to Phase 3+)

---

## 5. Target Users and Personas

| Persona | Role | Pain Points | Value from QL-RF |
|---------|------|-------------|------------------|
| Cloud Ops Engineer | Manages fleet patching and image updates | Manual inventory, lack of visibility | Real-time drift view, single dashboard |
| Security/Compliance Officer | Ensures OS and app compliance | Manual audit trail, inconsistent evidence | Automated SBOM/patch traceability |
| SRE/Platform Lead | Maintains uptime and recovery | Manual DR, no validation of backups | Automated DR simulation hooks |
| CIO/CTO | Strategic risk oversight | No unified risk posture | Central resilience score + trends |
| QuantumLayer Integrator | Developer integrating AI orchestration | Lacks baseline compliance hooks | Contracts and event APIs |

---

## 6. Competitive Landscape

| Competitor | Focus | Limitation | QL-RF Opportunity |
|------------|-------|------------|-------------------|
| AWS Systems Manager | AWS-only patching | No multi-cloud/DC support | Multi-cloud governance |
| Azure Automanage | Azure-only automation | No AI, no hybrid support | Cross-platform abstraction |
| HashiCorp Terraform + Packer | IaC provisioning | No drift/patch validation | Lifecycle visibility |
| Qualys/Tenable | Security scanners | Reactive, not preventive | Contract-based automation |
| ServiceNow CMDB | Asset inventory | No real-time compliance | Real-time drift awareness |
| VMware SRM | DC failover | VMware-centric only | Multi-platform DR |

### Positioning

- **HashiCorp vs QL-RF:** Terraform is execution engine; QL-RF is the fabric that wraps it with compliance + DR
- **VMware vs QL-RF:** VMware is substrate-centric; QL-RF is substrate-agnostic
- **ServiceNow vs QL-RF:** ServiceNow is workflow-first; QL-RF is infra-first
- **Zerto/Veeam vs QL-RF:** They are backup-centric; QL-RF is live orchestration-centric

---

## 7. System Architecture

### 7.1 Architectural Layers

```
┌─────────────────────────────────────────────────────────────────┐
│                    EXPERIENCE LAYER                              │
│  ┌──────────────────────┐  ┌──────────────────────────────────┐ │
│  │   Control Tower UI    │  │        AI Copilot               │ │
│  │   (Next.js)           │  │   NL queries, RCA, DR guidance  │ │
│  └──────────────────────┘  └──────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    CONTROL PLANE (K8s)                           │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌──────────────┐  │
│  │ API Gateway│ │Orchestrator│ │  Resilience│ │Image Registry│  │
│  │  (Envoy)   │ │ (TF runner)│ │    Plane   │ │   Service    │  │
│  └────────────┘ └────────────┘ └────────────┘ └──────────────┘  │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌──────────────┐  │
│  │  Update    │ │  Policy    │ │  Inventory │ │  Connectors  │  │
│  │  Service   │ │  Service   │ │  Discovery │ │  (Multi-plat)│  │
│  └────────────┘ └────────────┘ └────────────┘ └──────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      DATA PLANE                                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐ │
│  │   AWS    │ │  Azure   │ │   GCP    │ │ vSphere  │ │  K8s   │ │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └────────┘ │
│  ┌──────────┐ ┌──────────┐                                      │
│  │Bare Metal│ │   Edge   │                                      │
│  └──────────┘ └──────────┘                                      │
└─────────────────────────────────────────────────────────────────┘
```

### 7.2 Core Services

#### Image Registry Service
Tracks image families and versions across substrates.

```
POST /images/{family}:{version}  # register/publish
GET  /images?family=...          # list approved
GET  /images/{family}/latest     # per-env lookup
Events: image.published, image.promoted
```

#### Update Service (Patching)
Ingests CVE feeds, maps against SBOMs → impact score.

```
POST /cve/ingest                 # internal CVE ingestion
GET  /patch/drift?env=prod       # drift analysis
POST /rebuild?family=...         # trigger rebuild
Events: cve.triaged, image.rebuilt, patch.drift_detected
```

#### Orchestrator
Declarative ExecutionSpec → Terraform runs + Ansible/Helm steps.

```
POST /exec           # plan/apply
POST /rollback       # rollback
GET  /exec/{id}      # status
Events: exec.started/succeeded/failed, rollout.batch_progress
```

#### Resilience Plane (BCP/DR)
Failover strategies: pilot-light, hot-standby, active/active.

```
POST /dr/drill       # initiate drill
POST /dr/failover    # trigger failover
POST /dr/failback    # orchestrate failback
GET  /dr/readiness   # readiness status
Events: dr.drill_*, dr.failover_*
```

---

## 8. Core Features and Components

### 8.1 Golden Image Management
- Versioned image registry with metadata (OS, packages, SBOM, signatures)
- Multi-cloud/DC format support (AMI, Azure SIG, GCE, vSphere, OCI)
- Policy-based promotion (dev → staging → prod)
- Cosign signature verification at plan/apply time
- SLSA Level 3 provenance attestation

### 8.2 Patch Orchestration
- CVE ingestion & triage with EPSS/KEV scoring
- Automated rebuild of golden images via Packer pipelines
- Canary rollout + observability health gates
- Fleet patch parity enforcement across sites
- Emergency hotfix with TTL enforcement (72h max)

### 8.3 Resilience Plane (BCP/DR)
- Failover strategies: pilot-light, hot-standby, active/active
- Automated DR drills with RTO/RPO measurement
- Traffic switch adapters: DNS, BGP/Anycast
- Failback orchestration with validation gates
- DR readiness scoring per workload/region

### 8.4 AI Copilot
- Predictive patching (forecast lag, prioritize CVEs)
- Guided failover decision-making with what-if simulation
- Natural language queries ('show me patch drift in EU DC')
- Canary analysis with auto-continue/halt recommendations
- RCA generation for incidents and drift events

### 8.5 Control Tower Dashboard
- Golden image compliance (% workloads on current version)
- Patch lag metrics (avg days behind baseline)
- DR readiness score (per region/site)
- Traffic-light (RAG) indicators for executives
- Drill-down from executive view to individual assets

---

## 9. Data Contracts and Schemas

### 9.1 Image Contract (`image.contract.yaml`)

```yaml
schema: v1
id: ql-base-linux
version: 1.6.3
os:
  name: ubuntu
  version: "22.04"
hardening:
  cis_level: 1
  ssh: disabled_password_auth
  firewall: enabled
runtimes:
  - name: docker
    version: "26.x"
  - name: nodejs
    version: "20.x"
baseline_packages:
  - curl
  - jq
  - ca-certificates
services:
  logging: "otel-collector"
  metrics: "node-exporter"
image_checks:
  - inspec_profile: profiles/linux-baseline
  - script: tests/smoke/boot.sh
attestations:
  sbom: spdx
  provenance: slsa-level-3
patch_policy:
  default: immutable_rebuild
  emergency_hotfix_ttl_hours: 72
sla:
  critical_hours: 24
  high_days: 7
  medium_days: 30
  low_days: 90
platform_coords:
  aws:
    name: "ql-base-linux-1.6.3"
  azure:
    gallery: "/subscriptions/.../versions/1.6.3"
  gcp:
    family: "ql-base-linux"
  vsphere:
    template: "ql-base-linux-1.6.3"
  oci:
    image: "harbor.local/ql/base:1.6.3"
```

### 9.2 Event Schema

| Event | Description |
|-------|-------------|
| `image.published` | New image version registered |
| `image.promoted` | Image promoted to higher environment |
| `drift.detected` | Patch drift identified in fleet |
| `cve.triaged` | CVE mapped to image impact |
| `rollout.batch_progress` | Canary batch completion |
| `dr.drill_started` | DR simulation started |
| `dr.drill_completed` | DR simulation completed |
| `dr.failover_started` | Actual failover triggered |
| `dr.failover_completed` | Failover completed |

---

## 10. MVP Scope (30 Days)

**Goal:** Live dashboard showing golden image coverage & patch drift across AWS, Azure, GCP, and vSphere—read-only.

| Component | Deliverable | Success Metric |
|-----------|-------------|----------------|
| AWS Connector | Inventory + Drift % | Fleet coverage ≥80% |
| Azure/GCP/vSphere Connectors | Normalised inventory | 3 clouds + 1 DC |
| Control Tower | RAG dashboard with drill-down | Drift visible in UI |
| Contract Registry | image.contract.yaml v1 | Versioned schema |
| Drift Engine | Daily scheduled drift calculation | <1% false positives |
| AI Summary | Simple LLM-based RCA prototype | Working prototype |
| CI/CD | Helm deploy + GH Actions | Pipeline ready |

---

## 11. Phase Plan and Roadmap

### Phase 1: Foundation (Month 1)
- [ ] Repo creation (`quantumlayerhq/resilience-fabric`)
- [ ] Read-only inventory + drift detection (AWS/Azure/GCP/vSphere)
- [ ] Contract format v1 (`image.contract.yaml`)
- [ ] Control Tower dashboard MVP
- [ ] API + Postgres + Redis infrastructure
- [ ] GH Actions CI/CD + Helm deploy

### Phase 2: Expansion (Month 2–3)
- [x] AI Insight Engine (LLM-based summarisation) — AI Orchestrator with 8 specialist agents
- [ ] Event bridge to QuantumLayer
- [x] RBAC and org-level multi-tenancy — Permission-based auth + Row-Level Security
- [ ] SBOM + Compliance attestations
- [ ] Basic DR simulation hooks
- [ ] Cosign signature verification

### Phase 3: Automation (Month 4–6)
- [ ] Controlled 'Patch-as-Code' workflows
- [ ] Predictive risk scoring
- [ ] Automated golden image rebuild suggestions
- [ ] Integration with QuantumLayer validation mesh
- [ ] Full DR failover orchestration

### Phase 4: DR Simulation & Resilience Scoring (Q2 2026)
- [ ] Digital twin for simulation and risk forecasting
- [ ] Composite Resilience Score (0–100) dashboard
- [ ] AI-driven DR 'game days' automation
- [ ] What-if scenario simulation

### Phase 5: Full Automation (Q3 2026)
- [ ] Patch-as-Code pipelines with auto-rebuild
- [ ] Closed-loop feedback for rollout decisions
- [ ] Autonomous canary analysis and promotion

### Phase 6: Ecosystem & Marketplace (Q4 2026)
- [ ] Public contract registry + capsule sharing
- [ ] B2B API for partner integrations
- [ ] Compliance pack marketplace

---

## 12. Technical Architecture

### 12.1 Tech Stack

| Layer | Technologies |
|-------|-------------|
| Backend | Python (FastAPI) / Go, Postgres, Redis, Kafka |
| Frontend | Next.js 14, Tailwind, shadcn/ui, Socket.IO |
| IaC | Terraform + Helm + Kubernetes |
| Contracts | YAML + JSONSchema + OPA (Rego policies) |
| AI | OpenAI API / Anthropic Claude |
| Security | OIDC (Clerk/Auth0), TLS, Cosign |
| Observability | Prometheus + Grafana + Loki + OpenTelemetry |
| Workflows | Temporal (Go/TypeScript) |

### 12.2 Deployment Topology

- **Control Plane:** Kubernetes (managed or self-hosted) with namespaces per env/tenant
- **Perimeter:** API Gateway (Envoy), WAF (optional), OIDC
- **State:** Postgres (Cloud SQL/Aurora) with read replicas; Kafka for events
- **Secrets:** Vault (KMS-sealed); short-lived creds via STS/MSI/WI

### 12.3 Multi-Tenancy & Security

- **Tenant Model:** org → projects → environments
- **Per-tenant RBAC** with row-level scoping
- **Fine-grained permissions:** plan/apply/failover/drill/approve
- **Policy isolation:** per-tenant OPA bundles with shared baseline
- **Supply chain:** cosign verify-before-use; SLSA provenance required

---

## 13. Identity and RBAC

### 13.1 Authentication
- OIDC providers: Clerk, Auth0, Okta, Azure AD
- Service accounts for CI/CD and automation
- Short-lived tokens via STS/MSI/Workload Identity

### 13.2 Authorization Model

| Role | Permissions |
|------|------------|
| Viewer | Read dashboards, view drift, export reports |
| Operator | Viewer + trigger DR drills, acknowledge alerts |
| Engineer | Operator + execute rollouts, manage images, apply patches |
| Admin | Engineer + manage RBAC, configure integrations, approve exceptions |

### 13.3 Secrets Management
- HashiCorp Vault for credential storage
- Cloud principals: AWS STS, Azure MSI, GCP Workload Identity
- vCenter credentials via Vault with rotation policy
- No long-lived credentials in code

---

## 14. Compliance and Evidence Packs

### 14.1 Compliance Packs
- CIS Benchmarks (Linux Level 1/2, Windows)
- NIST 800-53 controls mapping
- ISO 27001 control evidence
- PCI-DSS requirements mapping
- HIPAA safeguards alignment

### 14.2 Evidence Pack Format
Auto-generated evidence bundles:
- JSON index with metadata and timestamps
- SBOM (SPDX format) for each image
- SARIF vulnerability scan results
- InSpec/OSCAP compliance reports
- SLSA provenance attestations
- Cosign signatures and verification logs

### 14.3 Exception Workflow
- Risk documentation with compensating controls
- Time-boxed exception with review date
- Automatic reminders at T-7d, T-1d, T-0
- Leadership ACK required to extend
- Full audit trail

---

## 15. Windows Baseline Support

### 15.1 Golden Image Contract (Windows)
- Monthly Cumulative Updates via WSUS/Azure Update Manager
- Security-Only updates separated from feature updates
- Defender for Endpoint configuration
- LAPS enabled
- Credential Guard and BitLocker defaults

### 15.2 Patching Strategy
- Bake CU into golden VHD/Azure SIG
- Emergency Security-Only with 72h TTL
- Feature updates out-of-band

### 15.3 Compliance Checks
- Windows CIS Benchmark Level 1/2
- PowerShell DSC validation
- Azure Policy / Intune compliance

---

## 16. Air-Gapped and Regulated Site Support

### 16.1 Artifact Mirroring
- Harbor/ECR/ACR offline registry sync
- vCenter Content Library replication
- PXE image distribution to isolated networks
- SBOM sync via secure file transfer

### 16.2 Offline Operations
- Local Control Plane deployment
- Deferred signature verification
- Offline CI runners with artifact staging
- Manual evidence pack export

### 16.3 Connectivity Patterns
- Unidirectional data diode support
- Bastion-based sync
- Scheduled sync windows for CVE feeds

---

## 17. Networking and Failover Adapters

### 17.1 DNS-Based Traffic Switch
- AWS Route53 health checks
- Azure Traffic Manager profiles
- GCP Cloud DNS with geographic routing
- Configurable TTL for rapid failover

### 17.2 BGP/Anycast Patterns
- For latency-sensitive workloads
- Integration with network automation
- Health-based route advertisement

### 17.3 Guardrails
- OPA policy: no internet-facing LB in prod without exception
- Pre-failover health validation
- Traffic percentage rollout (10% → 50% → 100%)
- Auto-rollback on health degradation

---

## 18. FinOps and Capacity Considerations

### 18.1 Cost Visibility
- Pre-patch capacity checks
- Cost impact estimation for canary batches
- DR pilot-light cost budgets and alarms
- Resource tagging for cost attribution

### 18.2 Capacity Planning
- DC capacity reservations for DR
- Auto-scaling limits for surge during rollout
- Reserved instance alignment

### 18.3 Optimization
- Right-sizing recommendations
- Spot/preemptible for non-critical canaries
- Cost anomaly detection

---

## 19. KPIs and Success Metrics

| KPI | Target | Measurement |
|-----|--------|-------------|
| Fleet Coverage | ≥ 95% | Assets discovered / Total |
| Drift Detection Accuracy | ≥ 98% | Validated vs baseline |
| Dashboard Latency | < 2s | API response p95 |
| Risk Summary Accuracy | ≥ 90% | Manual validation |
| Adoption | 3 pilots | Signed MoUs |
| Integration Latency | < 5s | Event E2E time |
| Patch SLA Compliance | Critical ≤24h | Time from CVE to rollout |
| DR Drill Success Rate | ≥ 95% | RTO/RPO met |

### Operational SLOs
- Scan → Rebuild SLA: < 4 hours for Critical CVEs
- Plan/Apply throughput: 100 concurrent executions
- Dashboard p95 latency: < 2 seconds
- Event processing: < 500ms p99
- Auto-halt threshold: 5% error rate increase

---

## 20. Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Scope creep | Strict phase gates, MVP-first, weekly milestones |
| Cloud API throttling | Caching, exponential backoff, per-cloud quotas |
| Connector instability | Sandbox testing, circuit breakers |
| Data sensitivity | Read-only creds, least privilege, audit logging |
| AI hallucination | Validation via rules + confidence scoring |
| Integration with QL | Contract versioning + adapters |
| Market confusion | Position as 'Resilience Fabric' |
| DR complexity | Start with pilot-light, expand later |
| Vendor lock-in fears | Open APIs, Terraform-first |

---

## 21. Monetization Strategy

### 21.1 Core Pricing
- **SaaS Subscription:** $50–150/VM/year (patch + compliance)
- **DR Modules:** 2–3x premium
- **Enterprise License:** $250k–$1M (on-prem)

### 21.2 Add-Ons
- AI Copilot (separate SKU)
- Advanced Compliance Packs (PCI, HIPAA, ISO)
- Premium Integrations (ServiceNow, SAP)
- Executive Dashboards (ESG/energy impact)

### 21.3 Go-to-Market
- **Freemium:** 50 workloads, basic drift dashboard
- **Land & Expand:** Compliance → DR → AI
- **Marketplace:** AWS/Azure/GCP listings
- **Partners:** MSP revenue share, OEM bundles

---

## 22. Architectural Decision Records

### ADR-001: Contracts-First Design
**Decision:** Use versioned YAML contracts with data-driven platform coordinates.  
**Rationale:** Enables platform-agnostic operations, simplifies testing, supports audit.

### ADR-002: Agentless by Default
**Decision:** Prefer agentless connectors; limited agent for emergency patch jobs.  
**Rationale:** Reduces attack surface, simplifies deployment.

### ADR-003: Cosign for Artifact Signing
**Decision:** Sign everything with cosign; verify-before-use in prod.  
**Rationale:** Industry standard, integrates with SLSA.

### ADR-004: Temporal for Workflows
**Decision:** Use Temporal for long-running DR workflows.  
**Rationale:** Durable execution, retries, visibility.

### ADR-005: OPA as Policy Engine
**Decision:** OPA for plan/apply gates; Gatekeeper for K8s.  
**Rationale:** Unified Rego language, strong ecosystem.

### ADR-006: SBOM Format
**Decision:** SPDX (JSON) as standard.  
**Rationale:** Tooling support, government compliance.

---

## 23. Execution Plan

### Week 1–2: Skeleton & AWS
- [x] Contracts + ADRs committed
- [x] API + Inventory model + AWS connector
- [x] UI stub: env/region cards
- [x] Lock ADRs 001–006
- [x] RBAC skeleton (org/project/env)

### Week 3–4: Azure + GCP + vSphere
- [ ] Azure SIG + VMSS inventory
- [ ] GCP Images + MIG discovery
- [ ] vSphere templates + VM inventory
- [ ] Normalize to `Asset{platform, env, site, workload, image_ref}`
- [ ] Cosign verify in plan gate

### Week 5–6: Drift Engine + Heatmaps
- [ ] Compute coverage % and drift list
- [ ] `/drift?env=prod` API
- [ ] UI: heatmap per site (RAG), drill-down
- [ ] SBOM pointers and compliance badge
- [ ] Event schemas + outbox pattern

### Week 7–8: Hardening & Launch
- [ ] Caching, rate-limits, error budgets
- [x] RBAC enforcement (permission-based middleware + RLS)
- [ ] RAG status + exportable reports
- [ ] Trend charts
- [ ] Helm install docs
- [ ] Windows contract + Packer stub
- [ ] Evidence Pack generator CLI

### Exit Criteria (30 Days)
✅ Executive can see % on latest per platform/site, top offenders, and weekly trend.

---

## Appendix

### A. Repository Structure

```
resilience-fabric/
├── services/
│   ├── api/            # FastAPI/Go backend
│   ├── inventory/      # Cloud+DC discovery
│   ├── drift/          # Drift computation
│   └── connectors/     # Platform adapters
├── ui/control-tower/   # Next.js dashboard
├── contracts/          # YAML contracts
├── policy/             # OPA/Rego policies
├── docs/               # Architecture, ADRs
├── deploy/             # Helm charts
└── ops/                # GH Actions, TF
```

### B. Key References

- `contracts/image.contract.yaml` — Golden image specification
- `contracts/provisioning.contract.tf` — Terraform interface
- `contracts/events.schema.json` — Event payload schemas
- `policy/enforce.rego` — OPA policy rules
- `docs/architecture.md` — Technical architecture
- `docs/adr/` — Architectural decision records
- `sops/patching.md` — Patch lifecycle SOPs
- `sops/dr.md` — DR drill and failover SOPs

---

**Document Version:** 1.0  
**Last Updated:** December 2025  
**Author:** Subrahmanya Satish Gonella
