# QuantumLayer Resilience Fabric
## Execution Checklist & GitHub Issues

This document contains ready-to-use GitHub issues for tracking the implementation of QL-RF.

---

## Phase 1: Foundation (Month 1)

### Week 1-2 Issues

#### Issue #1: Repository Bootstrap
**Title:** `[SETUP] Initialize resilience-fabric repository`

**Labels:** `setup`, `priority:high`, `week:1`

**Description:**
Create the foundational repository structure for QuantumLayer Resilience Fabric.

**Acceptance Criteria:**
- [ ] Repository created at `quantumlayerhq/resilience-fabric`
- [ ] Branch protection enabled on `main`
- [ ] PR reviews required
- [ ] Conventional commits configured
- [ ] Basic README with project overview
- [ ] `.gitignore` for Python/Node/Go
- [ ] `LICENSE` file added

**Assignee:** @engineering-lead

---

#### Issue #2: ADR Documentation
**Title:** `[DOCS] Create and lock ADRs 001-006`

**Labels:** `documentation`, `priority:high`, `week:1`

**Description:**
Document architectural decisions that will guide implementation.

**Acceptance Criteria:**
- [ ] ADR-001: Contracts-First Design
- [ ] ADR-002: Agentless by Default
- [ ] ADR-003: Cosign for Artifact Signing
- [ ] ADR-004: Temporal for Workflows
- [ ] ADR-005: OPA as Policy Engine
- [ ] ADR-006: SBOM Format (SPDX)
- [ ] All ADRs reviewed and approved

**Template:**
```markdown
# ADR-00X: [Title]

## Status
Accepted

## Context
[Why we need to make this decision]

## Decision
[What we decided]

## Consequences
[What are the implications]
```

---

#### Issue #3: Contract Schemas
**Title:** `[CONTRACTS] Define v1 image and provisioning contracts`

**Labels:** `contracts`, `priority:high`, `week:1`

**Description:**
Create the foundational YAML contracts that define golden image specifications and provisioning interfaces.

**Acceptance Criteria:**
- [ ] `contracts/image.contract.yaml` with full schema
- [ ] `contracts/image.contract.windows.yaml` variant
- [ ] `contracts/provisioning.contract.tf` interface
- [ ] `contracts/events.schema.json` event definitions
- [ ] JSON Schema validation for contracts
- [ ] Example filled contracts in `contracts/examples/`

---

#### Issue #4: API Service Bootstrap
**Title:** `[API] Bootstrap FastAPI service skeleton`

**Labels:** `backend`, `priority:high`, `week:1`

**Description:**
Create the core API service with health checks, versioning, and basic structure.

**Acceptance Criteria:**
- [ ] FastAPI project structure in `services/api/`
- [ ] `/healthz` endpoint
- [ ] `/version` endpoint
- [ ] `/api/v1/images` stub
- [ ] `/api/v1/drift` stub
- [ ] Postgres connection setup
- [ ] Redis connection setup
- [ ] Docker compose for local dev
- [ ] Unit test structure with pytest

---

#### Issue #5: RBAC Skeleton
**Title:** `[AUTH] Implement RBAC skeleton`

**Labels:** `security`, `priority:high`, `week:1`

**Description:**
Implement the basic RBAC model with org/project/env hierarchy.

**Acceptance Criteria:**
- [ ] `Organization` model
- [ ] `Project` model
- [ ] `Environment` model
- [ ] `User` model with roles
- [ ] Four roles: Viewer, Operator, Engineer, Admin
- [ ] Permission constants defined
- [ ] FastAPI dependencies for auth
- [ ] Row-level security patterns documented

---

#### Issue #6: AWS Connector
**Title:** `[CONNECTOR] AWS inventory connector (read-only)`

**Labels:** `connector`, `priority:high`, `week:2`

**Description:**
Implement read-only AWS connector for AMI and EC2 instance discovery.

**Acceptance Criteria:**
- [ ] STS AssumeRole authentication
- [ ] List AMIs (owned by self)
- [ ] Describe EC2 instances
- [ ] Extract image_ref from instances
- [ ] Normalize to `Asset` model
- [ ] Pagination support
- [ ] Exponential backoff for rate limits
- [ ] Context timeouts
- [ ] Unit tests with moto mocking

**Asset Model:**
```python
class Asset:
    platform: str      # "aws"
    account: str       # AWS account ID
    region: str        # "eu-west-1"
    instance_id: str   # "i-abc123"
    image_ref: str     # "ami-xyz789"
    version: str       # extracted from AMI name/tags
    tags: Dict[str, str]
    discovered_at: datetime
```

---

#### Issue #7: Control Tower UI Skeleton
**Title:** `[UI] Bootstrap Next.js Control Tower`

**Labels:** `frontend`, `priority:high`, `week:2`

**Description:**
Create the Control Tower dashboard skeleton with basic routing and layout.

**Acceptance Criteria:**
- [ ] Next.js 14 project in `ui/control-tower/`
- [ ] Tailwind CSS configured
- [ ] shadcn/ui components installed
- [ ] Basic layout (sidebar, header)
- [ ] Routing: `/`, `/images`, `/drift`, `/sites`
- [ ] Auth integration (Clerk)
- [ ] API client setup (React Query)
- [ ] Socket.IO for real-time updates
- [ ] Placeholder cards for dashboard

---

### Week 3-4 Issues

#### Issue #8: Azure Connector
**Title:** `[CONNECTOR] Azure inventory connector (read-only)`

**Labels:** `connector`, `priority:high`, `week:3`

**Description:**
Implement read-only Azure connector for Shared Image Gallery and VMSS discovery.

**Acceptance Criteria:**
- [ ] MSI/Service Principal authentication
- [ ] List Shared Image Gallery versions
- [ ] List VMSS instances
- [ ] Extract image_ref from VMSS model
- [ ] Normalize to `Asset` model
- [ ] Pagination support
- [ ] Unit tests

---

#### Issue #9: GCP Connector
**Title:** `[CONNECTOR] GCP inventory connector (read-only)`

**Labels:** `connector`, `priority:high`, `week:3`

**Description:**
Implement read-only GCP connector for Compute Engine images and MIGs.

**Acceptance Criteria:**
- [ ] Workload Identity authentication
- [ ] List Compute Engine image families
- [ ] List MIG instances
- [ ] Extract image_ref from instance templates
- [ ] Normalize to `Asset` model
- [ ] Pagination support
- [ ] Unit tests

---

#### Issue #10: vSphere Connector
**Title:** `[CONNECTOR] vSphere inventory connector (read-only)`

**Labels:** `connector`, `priority:high`, `week:3`

**Description:**
Implement read-only vSphere connector for templates and VMs.

**Acceptance Criteria:**
- [ ] vCenter service account authentication (via Vault)
- [ ] List VM templates from Content Library
- [ ] List VMs with template references
- [ ] Extract template version from VM config
- [ ] Normalize to `Asset` model
- [ ] Handle multiple vCenter instances
- [ ] Unit tests with vcsim

---

#### Issue #11: Asset Normalization Layer
**Title:** `[CORE] Unified asset normalization layer`

**Labels:** `core`, `priority:high`, `week:3`

**Description:**
Create a unified layer that normalizes assets from all connectors.

**Acceptance Criteria:**
- [ ] `Asset` database model
- [ ] `AssetGraph` for relationship tracking
- [ ] Connector-to-Asset mapper interface
- [ ] Deduplication logic
- [ ] Change detection (new/updated/removed)
- [ ] Bulk upsert performance optimization
- [ ] Database migrations

---

#### Issue #12: Cosign Verification Gate
**Title:** `[SECURITY] Cosign signature verification in plan gate`

**Labels:** `security`, `priority:high`, `week:4`

**Description:**
Add cosign verification step before any plan/apply can proceed.

**Acceptance Criteria:**
- [ ] Cosign verify integration
- [ ] Verification in dev: warn only
- [ ] Verification in prod: enforce (fail if missing)
- [ ] Verification logs stored
- [ ] Test with signed and unsigned images
- [ ] Documentation for signing workflow

---

#### Issue #13: Event Schema & Outbox
**Title:** `[EVENTS] Implement event schema and outbox pattern`

**Labels:** `events`, `priority:high`, `week:4`

**Description:**
Implement event publishing with outbox pattern for reliability.

**Acceptance Criteria:**
- [ ] Event base schema (id, type, timestamp, payload)
- [ ] Event types: `image.*`, `drift.*`, `asset.*`
- [ ] Outbox table for transactional events
- [ ] Outbox relay worker (Kafka/Redis Streams)
- [ ] Idempotency via event IDs
- [ ] Event replay capability
- [ ] Unit tests

---

### Week 5-6 Issues

#### Issue #14: Drift Engine
**Title:** `[DRIFT] Implement drift detection engine`

**Labels:** `core`, `priority:high`, `week:5`

**Description:**
Calculate patch drift by comparing fleet state against golden image baseline.

**Acceptance Criteria:**
- [ ] Load current golden image versions from registry
- [ ] Query assets grouped by platform/env/site
- [ ] Calculate coverage percentage
- [ ] Identify outdated assets
- [ ] Calculate drift age (days behind)
- [ ] Store drift snapshots for trending
- [ ] API endpoint: `GET /api/v1/drift?env=...`
- [ ] <1% false positive rate

**Drift Model:**
```python
class DriftReport:
    environment: str
    platform: str
    site: Optional[str]
    total_assets: int
    compliant_assets: int
    coverage_pct: float
    outdated_assets: List[OutdatedAsset]
    calculated_at: datetime
```

---

#### Issue #15: Dashboard Heatmaps
**Title:** `[UI] Implement drift heatmap visualization`

**Labels:** `frontend`, `priority:high`, `week:5`

**Description:**
Build the visual heatmap showing drift status across sites and environments.

**Acceptance Criteria:**
- [ ] Heatmap component (RAG coloring)
- [ ] Site-level aggregation view
- [ ] Environment-level aggregation view
- [ ] Drill-down to asset list
- [ ] Color thresholds: red <70%, amber <90%, green ≥90%
- [ ] Real-time updates via Socket.IO
- [ ] Responsive design
- [ ] Accessibility (color-blind friendly)

---

#### Issue #16: SBOM Integration
**Title:** `[COMPLIANCE] SBOM storage and display`

**Labels:** `compliance`, `priority:medium`, `week:5`

**Description:**
Store and display SBOM information for golden images.

**Acceptance Criteria:**
- [ ] SBOM storage (S3/blob reference)
- [ ] SBOM metadata in image registry
- [ ] SBOM viewer in Control Tower
- [ ] SBOM diff between versions
- [ ] API endpoint: `GET /api/v1/images/{id}/sbom`

---

#### Issue #17: Compliance Badge System
**Title:** `[UI] Compliance badges for images and assets`

**Labels:** `frontend`, `compliance`, `week:6`

**Description:**
Show compliance status badges throughout the UI.

**Acceptance Criteria:**
- [ ] Badge component (CIS, SLSA, Signed, etc.)
- [ ] Image compliance summary
- [ ] Asset compliance inheritance
- [ ] Tooltip with details
- [ ] Filter by compliance status

---

### Week 7-8 Issues

#### Issue #18: API Rate Limiting & Caching
**Title:** `[API] Implement rate limiting and caching`

**Labels:** `backend`, `priority:high`, `week:7`

**Description:**
Add rate limiting and caching for API stability.

**Acceptance Criteria:**
- [ ] Redis-based rate limiting
- [ ] Per-tenant rate limits
- [ ] Cache layer for read endpoints
- [ ] Cache invalidation on writes
- [ ] Error budget tracking
- [ ] Circuit breaker for external APIs

---

#### Issue #19: Exportable Reports
**Title:** `[UI] Exportable CSV/PDF reports`

**Labels:** `frontend`, `priority:medium`, `week:7`

**Description:**
Allow users to export drift and compliance reports.

**Acceptance Criteria:**
- [ ] CSV export for drift data
- [ ] PDF export for executive summary
- [ ] Scheduled report generation
- [ ] Email delivery option (future)
- [ ] Report templates

---

#### Issue #20: Trend Charts
**Title:** `[UI] "What changed this week?" trend charts`

**Labels:** `frontend`, `priority:medium`, `week:7`

**Description:**
Show historical trends for drift and compliance.

**Acceptance Criteria:**
- [ ] Time-series chart component
- [ ] Drift % over time
- [ ] Coverage % over time
- [ ] Week-over-week comparison
- [ ] Configurable time ranges

---

#### Issue #21: Windows Image Contract
**Title:** `[CONTRACTS] Windows golden image contract`

**Labels:** `contracts`, `priority:medium`, `week:7`

**Description:**
Define Windows-specific golden image contract variant.

**Acceptance Criteria:**
- [ ] Windows CIS requirements
- [ ] WSUS/Update Manager integration points
- [ ] Defender configuration
- [ ] LAPS configuration
- [ ] Credential Guard settings
- [ ] Example Packer stub

---

#### Issue #22: Evidence Pack Generator
**Title:** `[COMPLIANCE] Evidence pack generator CLI`

**Labels:** `compliance`, `priority:medium`, `week:8`

**Description:**
CLI tool to generate audit evidence packs.

**Acceptance Criteria:**
- [ ] CLI command: `rf evidence-pack generate`
- [ ] Bundle SBOM + SARIF + InSpec reports
- [ ] JSON index with metadata
- [ ] ZIP archive output
- [ ] Filter by date range
- [ ] Filter by environment

---

#### Issue #23: Helm Charts
**Title:** `[DEPLOY] Production-ready Helm charts`

**Labels:** `deployment`, `priority:high`, `week:8`

**Description:**
Create Helm charts for deploying QL-RF.

**Acceptance Criteria:**
- [ ] API service chart
- [ ] UI service chart
- [ ] PostgreSQL dependency (optional)
- [ ] Redis dependency (optional)
- [ ] Kafka dependency (optional)
- [ ] Ingress configuration
- [ ] TLS configuration
- [ ] Resource limits
- [ ] HPA configuration
- [ ] values.yaml documentation
- [ ] One-command install docs

---

#### Issue #24: CI/CD Pipeline
**Title:** `[CI] GitHub Actions CI/CD pipeline`

**Labels:** `ci-cd`, `priority:high`, `week:8`

**Description:**
Set up complete CI/CD pipeline.

**Acceptance Criteria:**
- [ ] `ci.yml`: lint, test, build
- [ ] `release.yml`: semantic-release, tagging
- [ ] `security.yml`: Trivy scan, SARIF upload
- [ ] Docker image builds
- [ ] Push to GHCR
- [ ] Preview deployments for PRs
- [ ] Production deploy on tag

---

## Phase 2 Epic Issues (Month 2-3)

#### Issue #25: AI Insight Engine
**Title:** `[AI] LLM-based RCA and summarization engine`

**Labels:** `ai`, `priority:high`, `phase:2`

**Acceptance Criteria:**
- [ ] OpenAI/Claude API integration
- [ ] Drift explanation generation
- [ ] Risk summarization by site/env
- [ ] Confidence scoring
- [ ] Prompt templates
- [ ] Rate limiting for AI calls

---

#### Issue #26: Event Bridge to QuantumLayer
**Title:** `[INTEGRATION] Event bridge to QuantumLayer core`

**Labels:** `integration`, `priority:high`, `phase:2`

**Acceptance Criteria:**
- [ ] Kafka topic configuration
- [ ] Event schema alignment
- [ ] Producer implementation
- [ ] Consumer for QL events
- [ ] Dead letter queue
- [ ] Monitoring dashboards

---

#### Issue #27: DR Drill Framework
**Title:** `[DR] Basic DR drill simulation framework`

**Labels:** `dr`, `priority:high`, `phase:2`

**Acceptance Criteria:**
- [ ] Drill definition schema
- [ ] Pilot-light infra provisioning
- [ ] RTO/RPO measurement
- [ ] Health check framework
- [ ] Drill result storage
- [ ] Drill scheduling

---

## Tracking Dashboard

### Week 1 Progress
| Issue | Title | Status | Assignee |
|-------|-------|--------|----------|
| #1 | Repository Bootstrap | ⬜ | |
| #2 | ADR Documentation | ⬜ | |
| #3 | Contract Schemas | ⬜ | |
| #4 | API Service Bootstrap | ⬜ | |
| #5 | RBAC Skeleton | ⬜ | |

### Week 2 Progress
| Issue | Title | Status | Assignee |
|-------|-------|--------|----------|
| #6 | AWS Connector | ⬜ | |
| #7 | Control Tower UI Skeleton | ⬜ | |

### Week 3 Progress
| Issue | Title | Status | Assignee |
|-------|-------|--------|----------|
| #8 | Azure Connector | ⬜ | |
| #9 | GCP Connector | ⬜ | |
| #10 | vSphere Connector | ⬜ | |
| #11 | Asset Normalization Layer | ⬜ | |

### Week 4 Progress
| Issue | Title | Status | Assignee |
|-------|-------|--------|----------|
| #12 | Cosign Verification Gate | ⬜ | |
| #13 | Event Schema & Outbox | ⬜ | |

### Week 5-6 Progress
| Issue | Title | Status | Assignee |
|-------|-------|--------|----------|
| #14 | Drift Engine | ⬜ | |
| #15 | Dashboard Heatmaps | ⬜ | |
| #16 | SBOM Integration | ⬜ | |
| #17 | Compliance Badge System | ⬜ | |

### Week 7-8 Progress
| Issue | Title | Status | Assignee |
|-------|-------|--------|----------|
| #18 | API Rate Limiting & Caching | ⬜ | |
| #19 | Exportable Reports | ⬜ | |
| #20 | Trend Charts | ⬜ | |
| #21 | Windows Image Contract | ⬜ | |
| #22 | Evidence Pack Generator | ⬜ | |
| #23 | Helm Charts | ⬜ | |
| #24 | CI/CD Pipeline | ⬜ | |

---

## GitHub Project Board Setup

Create a GitHub Project with these columns:
1. **Backlog** - All issues not yet started
2. **Sprint** - Current sprint items
3. **In Progress** - Actively being worked on
4. **In Review** - PR open, awaiting review
5. **Done** - Completed and merged

### Labels to Create
```
priority:high
priority:medium
priority:low
phase:1
phase:2
phase:3
week:1
week:2
week:3
week:4
week:5
week:6
week:7
week:8
backend
frontend
connector
contracts
security
compliance
ai
dr
ci-cd
deployment
documentation
setup
core
events
integration
```

---

## Success Metrics Tracking

### 30-Day Exit Criteria Checklist
- [ ] ≥80% fleet coverage for AWS
- [ ] 3 clouds + 1 DC connected (AWS, Azure, GCP, vSphere)
- [ ] Drift visible in Control Tower UI
- [ ] RAG status displayed correctly
- [ ] Contract schema v1 published
- [ ] <1% false positive rate on drift detection
- [ ] CI/CD pipeline operational
- [ ] Helm install working

### Quality Gates
- [ ] Unit test coverage ≥80%
- [ ] No critical security vulnerabilities (Trivy)
- [ ] API response time p95 <2s
- [ ] Zero unhandled exceptions in production logs
