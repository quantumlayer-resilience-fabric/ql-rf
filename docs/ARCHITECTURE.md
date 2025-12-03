# QL-RF Architecture

> AI-Powered Infrastructure Resilience & Compliance Platform

**Last Updated:** 2025-12-03

---

## System Overview

QL-RF (QuantumLayer Resilience Fabric) is an LLM-first infrastructure operations platform that transforms traditional dashboard-based workflows into AI-driven, human-approved automation.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          EXPERIENCE LAYER                                │
│  ┌────────────────────────────┐  ┌────────────────────────────────────┐ │
│  │     Control Tower UI        │  │          AI Copilot                │ │
│  │   (Next.js + shadcn/ui)     │  │   NL tasks, approval, execution   │ │
│  └────────────────────────────┘  └────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          CONTROL PLANE                                   │
│  ┌──────────────┐ ┌────────────────┐ ┌─────────────┐ ┌───────────────┐  │
│  │  API Service │ │ AI Orchestrator│ │   Drift     │ │  Connectors   │  │
│  │  (Port 8080) │ │  (Port 8083)   │ │   Engine    │ │   Service     │  │
│  └──────────────┘ └────────────────┘ └─────────────┘ └───────────────┘  │
│                           │                                              │
│                           ▼                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │                      TEMPORAL CLUSTER                                ││
│  │   TaskExecutionWorkflow │ DRDrillWorkflow │ Durable Activities      ││
│  └─────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           DATA PLANE                                     │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐  │
│  │   AWS    │ │  Azure   │ │   GCP    │ │ vSphere  │ │  Kubernetes  │  │
│  │ (SDK v2) │ │ (Go SDK) │ │ (Go SDK) │ │(govmomi) │ │  (client-go) │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Service Architecture

### 1. API Service (`services/api/`)

Core REST API for the Control Tower UI.

**Port:** 8080

**Responsibilities:**
- Asset management (CRUD operations)
- Golden image registry
- Drift status queries
- Compliance reporting
- Site and DR pair management

**Key Endpoints:**
```
GET  /api/v1/assets           # List assets with filters
GET  /api/v1/images           # List golden images
GET  /api/v1/drift            # Current drift report
GET  /api/v1/compliance       # Compliance status
GET  /api/v1/resilience       # DR pairs and status

# Image Lineage Endpoints
GET  /api/v1/images/{id}/lineage              # Full lineage with parents, children, vulns
GET  /api/v1/images/families/{family}/lineage-tree  # Tree view for family
POST /api/v1/images/{id}/lineage/parents      # Add parent relationship
GET  /api/v1/images/{id}/vulnerabilities      # CVE list for image
POST /api/v1/images/{id}/vulnerabilities      # Record new vulnerability
POST /api/v1/images/{id}/vulnerabilities/import  # Bulk import from scanners
GET  /api/v1/images/{id}/builds               # Build provenance history
GET  /api/v1/images/{id}/deployments          # Where image is deployed
GET  /api/v1/images/{id}/components           # SBOM components
POST /api/v1/images/{id}/sbom                 # Import SBOM data
```

### 2. AI Orchestrator (`services/orchestrator/`)

LLM-first operations engine that converts natural language to executed infrastructure tasks.

**Port:** 8083

**Architecture:**
```
┌─────────────────────────────────────────────────────────────────┐
│                     AI ORCHESTRATOR                              │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              LAYER 1: META-PROMPT ENGINE                │    │
│  │   Natural Language → TaskSpec (agent, tools, risk)      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              ↓                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              LAYER 2: SPECIALIST AGENTS                 │    │
│  │  ┌─────────┐ ┌─────────┐ ┌──────────┐ ┌──────────┐     │    │
│  │  │ Drift   │ │ Patch   │ │Compliance│ │ Incident │     │    │
│  │  │ Agent   │ │ Agent   │ │  Agent   │ │  Agent   │     │    │
│  │  └─────────┘ └─────────┘ └──────────┘ └──────────┘     │    │
│  │  ┌─────────┐ ┌─────────┐ ┌──────────┐ ┌──────────┐     │    │
│  │  │   DR    │ │  Cost   │ │ Security │ │  Image   │     │    │
│  │  │  Agent  │ │  Agent  │ │  Agent   │ │  Agent   │     │    │
│  │  └─────────┘ └─────────┘ └──────────┘ └──────────┘     │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              ↓                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              LAYER 3: TOOL REGISTRY (26+ tools)         │    │
│  │   query_assets │ get_golden_image │ generate_patch_plan │    │
│  │   check_control │ simulate_failover │ generate_sop      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              ↓                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              LAYER 4: VALIDATION PIPELINE               │    │
│  │   Schema Validation │ OPA Policies │ Safety Checks      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              ↓                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              LAYER 5: EXECUTION ENGINE                  │    │
│  │   Phased Rollout │ Health Checks │ Rollback Logic       │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

**Key Endpoints:**
```
POST /api/v1/ai/execute           # Execute NL task
GET  /api/v1/ai/tasks             # List tasks
POST /api/v1/ai/tasks/{id}/approve # Approve task
POST /api/v1/ai/tasks/{id}/reject  # Reject task
GET  /api/v1/ai/agents            # List agents
GET  /api/v1/ai/tools             # List tools
```

### 3. Connectors Service (`services/connectors/`)

Platform-specific adapters for infrastructure discovery and operations.

**Implemented Connectors:**

| Platform | SDK | Capabilities |
|----------|-----|--------------|
| AWS | AWS SDK v2 | EC2 instances, AMIs, EBS, VPCs |
| Azure | Azure SDK for Go | VMs, managed disks, images, VNets |
| GCP | Google Cloud Go SDK | Compute instances, images, disks |
| vSphere | govmomi | VMs, templates, datastores |

**Interface:**
```go
type Connector interface {
    Name() string
    Connect(ctx context.Context) error
    Health(ctx context.Context) error
    DiscoverAssets(ctx context.Context, siteID uuid.UUID) ([]models.Asset, error)
    DiscoverImages(ctx context.Context) ([]models.GoldenImage, error)
    Close() error
}
```

### 4. Drift Service (`services/drift/`)

Real-time drift detection engine using Kafka event streaming.

**Components:**
- Kafka consumer for asset change events
- Drift calculation engine
- Trend analysis
- Alert generation

### 5. Temporal Workflows

Durable workflow execution for long-running operations.

**Workflows:**

| Workflow | Purpose |
|----------|---------|
| TaskExecutionWorkflow | AI task execution with phases |
| DRDrillWorkflow | DR drill with failover/failback |

**DR Drill Phases:**
1. Pre-check - Verify DR pair health
2. Replication Sync - Ensure data sync
3. Failover - Execute failover to DR
4. Validation - Validate DR services
5. Failback - Restore to primary
6. Post-check - Verify restoration

---

## Frontend Architecture

### Control Tower UI (`ui/control-tower/`)

Next.js 14+ application with App Router.

**Tech Stack:**
- Next.js 14 (App Router)
- React 18
- Tailwind CSS
- shadcn/ui components
- React Query (TanStack Query)
- Clerk (authentication)

**Structure:**
```
ui/control-tower/src/
├── app/
│   ├── (dashboard)/           # Dashboard routes
│   │   ├── ai/               # AI task management
│   │   │   ├── tasks/        # Task list
│   │   │   └── chat/         # AI chat interface
│   │   ├── assets/           # Asset management
│   │   ├── images/           # Golden images
│   │   │   └── [id]/lineage/ # Image lineage detail
│   │   ├── drift/            # Drift analysis
│   │   ├── compliance/       # Compliance dashboard
│   │   └── resilience/       # DR management
│   └── (marketing)/          # Public pages
├── components/
│   ├── ai/                   # AI-specific components
│   │   ├── task-approval-card.tsx
│   │   ├── pending-task-card.tsx
│   │   ├── execution-status.tsx
│   │   └── ai-chat-interface.tsx
│   ├── images/               # Image lineage components
│   │   ├── lineage-tree.tsx  # Hierarchical tree visualization
│   │   ├── lineage-graph.tsx # Interactive canvas graph
│   │   ├── vulnerability-summary.tsx
│   │   ├── vulnerability-trend-chart.tsx  # Time-series chart
│   │   └── build-history.tsx
│   └── ui/                   # shadcn/ui components
└── hooks/
    ├── use-ai.ts             # AI task hooks
    ├── use-lineage.ts        # Image lineage hooks
    └── use-permissions.ts    # RBAC hooks
```

### RBAC Implementation

Permission-based UI rendering with PermissionGate:

```tsx
<PermissionGate permission={Permissions.APPROVE_AI_TASKS}>
  <Button onClick={approve}>Approve</Button>
</PermissionGate>
```

**Permissions:**
- `APPROVE_AI_TASKS` - Approve/reject AI tasks
- `EXECUTE_AI_TASKS` - Control task execution
- `TRIGGER_DRILL` - Start DR drills
- `VIEW_AUDIT_LOGS` - View audit trail

---

## Data Architecture

### PostgreSQL Schema

**Core Tables:**
```sql
-- Organizations and users
organizations, users, org_memberships

-- Sites and assets
sites, assets, asset_tags

-- Golden images
images (golden_image_families, golden_images)

-- Image Lineage (Migration 000006)
image_lineage          -- Parent-child relationships (derived_from, patched_from, rebuilt_from)
image_builds           -- SLSA-compatible build provenance
image_vulnerabilities  -- CVE tracking per image
image_deployments      -- Where images are deployed
image_promotions       -- Status transition audit trail
image_components       -- SBOM data
image_tags             -- Custom key-value metadata

-- Drift tracking
drift_reports, drift_items

-- Compliance
compliance_frameworks, compliance_controls, control_evidence

-- DR
dr_pairs, dr_drills, dr_drill_results

-- AI tasks
ai_tasks, ai_task_plans, ai_tool_invocations
```

**Lineage Views:**
```sql
v_image_lineage_tree      -- Recursive CTE for tree traversal
v_image_vuln_summary      -- Vulnerability counts by severity
v_image_deployment_summary -- Deployment statistics
```

### Event Streaming (Kafka)

**Topics:**
- `asset.changes` - Asset state changes
- `drift.detected` - Drift events
- `task.status` - AI task status updates
- `audit.events` - Audit log events

---

## Security Architecture

### Authentication
- Clerk OIDC/JWT for user authentication
- Service-to-service mTLS

### Authorization
- Role-based access control (RBAC)
- OPA policies for fine-grained authorization
- Permission gates in UI

### Policy Validation
```
AI Plan → Schema Validation → OPA Policy → Safety Check → HITL → Execute
```

**OPA Policies:**
- Production safety rules
- Batch size limits
- Canary requirements
- Rollback criteria validation

---

## Deployment Architecture

### Local Development
```bash
docker compose up -d           # Start infrastructure
make migrate-up                # Apply migrations
make run-api                   # Run API service
make run-orchestrator          # Run AI orchestrator
```

### Infrastructure Components
- PostgreSQL 16
- Redis 7
- Apache Kafka
- Temporal Server

### Production (Target)
- Kubernetes (AKS/EKS/GKE)
- Istio service mesh
- Prometheus + Grafana
- OpenTelemetry

---

## API Reference

### Core API (Port 8080)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/healthz` | Liveness probe |
| GET | `/readyz` | Readiness probe |
| GET | `/api/v1/assets` | List assets |
| GET | `/api/v1/images` | List golden images |
| GET | `/api/v1/images/{id}/lineage` | Image lineage |
| GET | `/api/v1/images/families/{family}/lineage-tree` | Family tree |
| GET | `/api/v1/images/{id}/vulnerabilities` | CVE list |
| POST | `/api/v1/images/{id}/vulnerabilities/import` | Import scanner results |
| GET | `/api/v1/images/{id}/builds` | Build history |
| GET | `/api/v1/images/{id}/deployments` | Deployments |
| GET | `/api/v1/images/{id}/components` | SBOM |
| POST | `/api/v1/images/{id}/sbom` | Import SBOM |
| GET | `/api/v1/drift` | Drift report |
| GET | `/api/v1/compliance` | Compliance status |
| GET | `/api/v1/resilience/dr-pairs` | DR pairs |

### AI Orchestrator (Port 8083)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| POST | `/api/v1/ai/execute` | Execute AI task |
| GET | `/api/v1/ai/tasks` | List tasks |
| GET | `/api/v1/ai/tasks/{id}` | Get task |
| POST | `/api/v1/ai/tasks/{id}/approve` | Approve |
| POST | `/api/v1/ai/tasks/{id}/reject` | Reject |
| POST | `/api/v1/ai/tasks/{id}/modify` | Modify |
| GET | `/api/v1/ai/agents` | List agents |
| GET | `/api/v1/ai/tools` | List tools |

---

## Testing Strategy

### Unit Tests
```bash
make test              # Run all unit tests
make test-coverage     # With coverage report
make test-race         # With race detector
```

### Integration Tests
```bash
make test-integration  # Integration tests (requires Docker)
make test-e2e          # Full E2E tests
```

### Test Files
- `tests/integration/orchestrator_test.go` - Orchestrator API
- `tests/integration/api_test.go` - Main API
- `tests/integration/connectors_test.go` - Cloud connectors

---

## Monitoring & Observability

### Metrics (Prometheus)
- Request latency
- Error rates
- Task execution duration
- Connector health

### Logging (Structured JSON)
- Component-based logging
- Request tracing
- Audit events

### Tracing (OpenTelemetry)
- Distributed tracing
- Span correlation
- Service dependencies

---

## ADRs (Architectural Decision Records)

| ADR | Decision |
|-----|----------|
| [ADR-001](adr/ADR-001-contracts-first.md) | Contracts-first approach |
| [ADR-002](adr/ADR-002-agentless-by-default.md) | Agentless infrastructure |
| [ADR-003](adr/ADR-003-cosign-signing.md) | Image signing with Cosign |
| [ADR-005](adr/ADR-005-opa-policy-engine.md) | OPA for policy validation |
| [ADR-006](adr/ADR-006-sbom-spdx.md) | SBOM with SPDX format |
