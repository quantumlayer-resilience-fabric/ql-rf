# QL-RF Architecture

> AI-Powered Infrastructure Resilience & Compliance Platform

**Last Updated:** 2025-12-04

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
│  │           LAYER 2: SPECIALIST AGENTS (11 total)         │    │
│  │  ┌─────────┐ ┌─────────┐ ┌──────────┐ ┌──────────┐     │    │
│  │  │ Drift   │ │ Patch   │ │Compliance│ │ Incident │     │    │
│  │  │ Agent   │ │ Agent   │ │  Agent   │ │  Agent   │     │    │
│  │  └─────────┘ └─────────┘ └──────────┘ └──────────┘     │    │
│  │  ┌─────────┐ ┌─────────┐ ┌──────────┐ ┌──────────┐     │    │
│  │  │   DR    │ │  Cost   │ │ Security │ │  Image   │     │    │
│  │  │  Agent  │ │  Agent  │ │  Agent   │ │  Agent   │     │    │
│  │  └─────────┘ └─────────┘ └──────────┘ └──────────┘     │    │
│  │  ┌─────────┐ ┌─────────┐ ┌──────────┐                  │    │
│  │  │   SOP   │ │ Adapter │ │  Base    │                  │    │
│  │  │  Agent  │ │  Agent  │ │  Agent   │                  │    │
│  │  └─────────┘ └─────────┘ └──────────┘                  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              ↓                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              LAYER 3: TOOL REGISTRY (29 tools)          │    │
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
| AWS | AWS SDK v2 | EC2 instances, AMIs, EBS, VPCs, **SSM Patch Management** |
| Azure | Azure SDK for Go | VMs, managed disks, images, VNets |
| GCP | Google Cloud Go SDK | Compute instances, images, disks |
| vSphere | govmomi | VMs, templates, datastores |

**AWS SSM Patch Management:**

The AWS connector includes native SSM integration for patch operations:

```go
// SSM Patcher capabilities
type SSMPatcher interface {
    ApplyPatchBaseline(ctx, instanceID, operation, rebootOption string) (*PatchOperation, error)
    ScanForPatches(ctx, instanceID string) (*PatchOperation, error)
    InstallPatches(ctx, instanceID string) (*PatchOperation, error)
    GetPatchComplianceStatus(ctx, instanceID string) (*PatchComplianceStatus, error)
    GetManagedInstances(ctx context.Context) ([]ManagedInstance, error)
    WaitForCommand(ctx, commandID, instanceID string, timeout time.Duration) (*PatchOperation, error)
}
```

**Patch Operations:**
- `Scan` - Check for missing patches without installing
- `Install` - Install missing patches (uses `AWS-RunPatchBaseline` document)
- Configurable reboot options: `RebootIfNeeded`, `NoReboot`
- Compliance status tracking: `COMPLIANT`, `NON_COMPLIANT`, `PENDING`

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

**Transaction Support:**

The sync service uses database transactions for atomic asset synchronization:

```go
// BeginTx starts a transaction and returns a transactional repository
tx, txRepo, err := s.repo.BeginTx(ctx)
defer func() {
    if err != nil {
        tx.Rollback(ctx)
    }
}()

// All operations use txRepo for consistency
existing, err := txRepo.ListAssetsByPlatform(ctx, orgID, platform)
dbAsset, isNew, err := txRepo.UpsertAsset(ctx, params)
txRepo.MarkAssetTerminated(ctx, existingAsset.ID)

// Commit only on success
tx.Commit(ctx)
```

### 4. Drift Service (`services/drift/`)

Real-time drift detection engine using Kafka event streaming.

**Components:**
- Kafka consumer for asset change events
- Drift calculation engine (real database queries)
- Trend analysis with scope-based grouping
- Alert generation

**Drift Engine Queries:**
- `getGoldenImageBaselines()` - Queries `images` table for production status baselines
- `getFleetAssets()` - Queries `assets` table with platform/site/environment filters
- `calculateByScope()` - Aggregates drift metrics grouped by environment/platform/site

### 5. Compliance Service (`services/api/internal/service/`)

Compliance tracking with database-backed framework and control queries.

**Database Queries:**
- `getFrameworks()` - Queries `compliance_frameworks` with control pass/fail counts
- `getFailingControls()` - Queries `compliance_controls` joined with results, sorted by severity
- `getImageCompliance()` - Queries `images` with `image_compliance` for CIS/SLSA/Cosign status

**Key Features:**
- Weighted score calculation across enabled frameworks
- Severity-sorted failing controls (critical → low)
- Image compliance aggregation (CIS, SLSA level, Cosign signing)
- Sigstore verification percentage

### 6. Health Checks (`services/api/internal/handlers/health.go`)

Production-ready health check endpoints with dependency status.

**Endpoints:**
- `/healthz` - Liveness probe (returns 200 if service is running)
- `/readyz` - Readiness probe (checks all dependencies)
- `/version` - Build info and git commit
- `/metrics` - Prometheus metrics (if enabled)

**Readiness Checks:**
- **Database** - PostgreSQL connection health (critical)
- **Kafka** - Kafka broker connectivity (non-critical)
- **Redis** - Redis ping/pong (non-critical)

**Response Statuses:**
- `ok` - All critical checks passing
- `degraded` - Non-critical checks failing (returns 503)

### 7. Multi-Tenant Middleware (`services/api/internal/middleware/auth.go`)

Database-backed organization resolution for multi-tenancy.

**Resolution Order:**
1. User's organization (from authenticated claims)
2. Claims-based organization ID
3. Database lookup by external ID
4. Development mode fallback (first org in database)

**Context Values:**
- `orgIDKey` - Organization UUID
- `userIDKey` - User UUID
- `claimsKey` - JWT claims

### 8. Risk Scoring Service (`services/api/internal/service/risk_service.go`)

AI-powered risk scoring with weighted factors for prioritizing remediation efforts.

**Risk Calculation Model:**
```
Risk Score = Σ(Factor Weight × Factor Score) × Environment Multiplier
```

**Risk Factors (Weights):**

| Factor | Weight | Scoring Logic |
|--------|--------|---------------|
| Drift Age | 25% | 2 points per day of drift (max 100) |
| Vulnerability Count | 20% | 5 points per open vulnerability (max 100) |
| Critical Vulnerabilities | 25% | 25 points per critical CVE (max 100) |
| Compliance Status | 15% | 100 if non-compliant, 0 if compliant |
| Environment Impact | 15% | Multiplied by environment factor |

**Environment Multipliers:**

| Environment | Multiplier |
|-------------|------------|
| Production | 1.5x |
| DR | 1.2x |
| Staging | 1.0x |
| Development | 0.5x |

**Risk Levels:**
- **Critical** (≥80): Immediate action required
- **High** (≥60): Priority remediation
- **Medium** (≥40): Scheduled remediation
- **Low** (<40): Monitor

**Key Functions:**
```go
func (s *RiskService) GetRiskSummary(ctx context.Context, orgID uuid.UUID) (*models.RiskSummary, error)
func (s *RiskService) GetTopRisks(ctx context.Context, orgID uuid.UUID, limit int) ([]models.AssetRiskScore, error)
```

### 9. Temporal Workflows

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

### 10. Execution Engine (`services/orchestrator/internal/executor/`)

Phased execution engine with proper context management and cancellation support.

**Key Features:**
- **Context Timeouts**: All executions have configurable maximum timeout (default: 4 hours)
- **Cancellation Support**: Proper cancellation propagation with cleanup
- **Phased Rollout**: Execute plans in phases with health checks between phases
- **Rollback Logic**: Automatic rollback on phase failures via platform connectors

**Execution Flow:**
```
Task Approved → Create Context with Timeout → Execute Phases → Health Checks → Complete/Rollback
                      ↓
              Store Cancel Function → Cancellable at any time
```

### 11. Notification Service (`services/orchestrator/internal/notifier/`)

Multi-channel notification system for task lifecycle events.

**Channels:**
- **Slack**: Webhook-based notifications with rich formatting
- **Email**: SMTP-based email notifications
- **Webhook**: Generic HTTP webhooks with HMAC-SHA256 signatures
- **MS Teams**: Adaptive Cards (v1.4) with rich formatting and action buttons

**Configuration:**
```bash
# Slack
RF_SLACK_ENABLED=true
RF_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/xxx

# MS Teams
RF_TEAMS_ENABLED=true
RF_TEAMS_WEBHOOK_URL=https://outlook.office.com/webhook/xxx

# Email
RF_EMAIL_ENABLED=true
RF_SMTP_HOST=smtp.example.com
RF_SMTP_PORT=587

# Webhooks
RF_WEBHOOK_ENABLED=true
RF_WEBHOOK_URL=https://your-endpoint.com/webhook
RF_WEBHOOK_SECRET=your-hmac-secret
```

**Security:**
- Webhooks signed with `X-QL-Signature: sha256=<hmac>` header
- Configurable secret per organization

### 12. ServiceNow Integration (`services/orchestrator/internal/integrations/servicenow/`)

Enterprise ITSM integration for change management and incident tracking.

**Capabilities:**
- **Change Requests**: Auto-create CHG records for AI-planned operations
- **Incidents**: Auto-create INC records for execution failures
- **CMDB Sync**: Sync assets to ServiceNow Configuration Items
- **Audit Trail**: Link QL-RF tasks to ServiceNow tickets

**Configuration:**
```bash
RF_SERVICENOW_ENABLED=true
RF_SERVICENOW_INSTANCE_URL=https://mycompany.service-now.com
RF_SERVICENOW_USERNAME=api_user
RF_SERVICENOW_PASSWORD=api_token
```

**Risk Level Mapping:**
| QL-RF Risk | ServiceNow Risk | Priority |
|------------|-----------------|----------|
| state_change_prod | High | 1 (Critical) |
| state_change_nonprod | Moderate | 2 (High) |
| plan_only | Low | 3 (Moderate) |
| read_only | Low | 4 (Low) |

### 13. Prediction Service (`services/api/internal/service/prediction_service.go`)

AI-powered risk prediction using real database data.

**Data Sources:**
- `drift_reports` table for historical risk trends
- `assets` table for current asset state analysis
- Real-time drift age calculation

**Risk Calculation:**
```go
score := 30.0 // Base score
switch state {
case "running": score -= 10
case "stopped", "terminated": score += 20
case "unknown": score += 30
}
if driftAge > 30 { score += 30 }
else if driftAge > 14 { score += 20 }
else if driftAge > 7 { score += 10 }
if !hasImage { score += 15 }
```

---

## Frontend Architecture

### Control Tower UI (`ui/control-tower/`)

Next.js 16 application with App Router.

**Tech Stack:**
- Next.js 16 (App Router)
- React 19
- Tailwind CSS
- shadcn/ui components
- React Query (TanStack Query)
- Clerk (authentication)

**Structure:**
```
ui/control-tower/src/
├── app/
│   ├── (dashboard)/           # Dashboard routes
│   │   ├── ai/               # AI Copilot
│   │   │   ├── tasks/        # Task list & detail
│   │   │   ├── agents/       # Agent status dashboard
│   │   │   └── usage/        # AI usage metrics
│   │   ├── overview/         # Fleet overview + ROI widget
│   │   ├── assets/           # Asset management
│   │   ├── images/           # Golden images
│   │   │   └── [id]/lineage/ # Image lineage detail
│   │   ├── drift/            # Drift analysis
│   │   ├── compliance/       # Compliance dashboard + PDF export
│   │   ├── risk/             # Risk scoring dashboard
│   │   └── resilience/       # DR management
│   └── (marketing)/          # Public pages
├── components/
│   ├── ai/                   # AI-specific components
│   │   ├── task-approval-card.tsx
│   │   ├── agent-status-dashboard.tsx
│   │   ├── execution-progress.tsx
│   │   ├── plan-modification-dialog.tsx
│   │   └── tool-invocation-audit.tsx
│   ├── data/                 # Data visualization components
│   │   ├── metric-card.tsx   # Generic metric display
│   │   └── value-delivered-card.tsx  # ROI/savings widget
│   ├── images/               # Image lineage components
│   │   ├── lineage-tree.tsx  # Hierarchical tree visualization
│   │   ├── lineage-graph.tsx # Interactive canvas graph
│   │   ├── vulnerability-summary.tsx
│   │   └── vulnerability-trend-chart.tsx
│   └── ui/                   # shadcn/ui components
├── lib/
│   └── pdf-export.ts         # PDF report generation (jsPDF)
└── hooks/
    ├── use-ai.ts             # AI task hooks
    ├── use-lineage.ts        # Image lineage hooks
    ├── use-risk.ts           # Risk scoring hooks
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

### Production Configuration Validation

The platform validates critical configuration at startup in production environments:

**Required Settings (Production):**
- `RF_DATABASE_URL` - Must not be localhost
- `RF_CLERK_SECRET_KEY` - Must be configured
- `RF_CLERK_PUBLISHABLE_KEY` - Must be configured
- `RF_ANTHROPIC_API_KEY` - Required for AI features

**Optional but Recommended:**
- `RF_KAFKA_BROKERS` - For event streaming
- `RF_REDIS_URL` - For caching

The validation fails fast at startup if production environment is missing required configuration, preventing deployment with insecure defaults.

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

### Kubernetes Deployment (`deployments/kubernetes/`)

Production-ready Kubernetes manifests using Kustomize.

**Directory Structure:**
```
deployments/kubernetes/
├── kustomization.yaml      # Kustomize config
├── namespace.yaml          # ql-rf namespace
├── config.yaml             # ConfigMap + Secrets
├── api-deployment.yaml     # API service (3 replicas)
├── orchestrator-deployment.yaml  # Orchestrator (2 replicas)
├── ui-deployment.yaml      # UI service (2 replicas)
├── ingress.yaml            # NGINX ingress with TLS
└── hpa.yaml                # HorizontalPodAutoscaler
```

**Deployment Specs:**

| Service | Replicas | CPU Request | Memory Request | HPA Max |
|---------|----------|-------------|----------------|---------|
| API | 3 | 100m | 128Mi | 10 |
| Orchestrator | 2 | 200m | 256Mi | 5 |
| UI | 2 | 50m | 128Mi | 6 |

**Ingress Routing:**
```
control-tower.quantumlayer.dev → ql-rf-ui:80
api.quantumlayer.dev/api/v1    → ql-rf-api:80
api.quantumlayer.dev/api/v1/ai → ql-rf-orchestrator:80
```

**HPA Configuration:**
- Scale on CPU (70% threshold) and Memory (80% threshold)
- Scale-down stabilization: 300 seconds
- Scale-up: aggressive (Max of 100% increase or +4 pods per 15s)

**Deployment Commands:**
```bash
# Deploy with Kustomize
kubectl apply -k deployments/kubernetes/

# Check deployment status
kubectl get pods -n ql-rf

# View logs
kubectl logs -f deployment/ql-rf-api -n ql-rf

# Scale manually (if needed)
kubectl scale deployment/ql-rf-api --replicas=5 -n ql-rf
```

**Security Features:**
- Non-root containers (runAsNonRoot: true)
- Read-only root filesystem
- No privilege escalation
- Service accounts per deployment
- TLS termination at ingress (cert-manager/letsencrypt)

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
| GET | `/api/v1/drift/summary` | Drift summary |
| GET | `/api/v1/compliance` | Compliance status |
| GET | `/api/v1/risk/summary` | Organization risk summary |
| GET | `/api/v1/risk/top` | Top risk assets |
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

### Test Coverage

**Critical Package Tests:**

| Package | Test File | Coverage |
|---------|-----------|----------|
| `pkg/auth` | `clerk_test.go` | JWT verification, JWKS fetching, key caching |
| `pkg/database` | `postgres_test.go` | Config validation, connection handling |
| `pkg/models` | `risk_test.go` | Risk calculations, model validation |
| `services/drift/internal/engine` | `drift_test.go` | Severity calculation, status thresholds |
| `services/orchestrator/internal/executor` | `executor_test.go` | Phase execution, cancellation |
| `services/orchestrator/internal/validation` | `schema_test.go` | JSON Schema validation |

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
