# QL-RF Architecture

> AI-Powered Infrastructure Resilience & Compliance Platform

**Last Updated:** 2025-12-05

---

## System Overview

QL-RF (QuantumLayer Resilience Fabric) is an LLM-first infrastructure operations platform that transforms traditional dashboard-based workflows into AI-driven, human-approved automation.

### Codebase Metrics (as of 2025-12-05)

| Metric | Count |
|--------|-------|
| Go Services | 4 |
| Go Files | 220+ |
| Go LOC | ~87,000 |
| Test Files | 60+ |
| Test LOC | ~21,000 |
| UI Components | 60 |
| Dashboard Pages | 15 |
| AI Agents | 10 |
| Tools | 29+ |
| OPA Policies | 6 |
| Migrations | 15 |
| E2E Tests | 230+ |

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
│  │           LAYER 2: SPECIALIST AGENTS (10 total)         │    │
│  │  ┌─────────┐ ┌─────────┐ ┌──────────┐ ┌──────────┐     │    │
│  │  │ Drift   │ │ Patch   │ │Compliance│ │ Incident │     │    │
│  │  │ Agent   │ │ Agent   │ │  Agent   │ │  Agent   │     │    │
│  │  └─────────┘ └─────────┘ └──────────┘ └──────────┘     │    │
│  │  ┌─────────┐ ┌─────────┐ ┌──────────┐ ┌──────────┐     │    │
│  │  │   DR    │ │  Cost   │ │ Security │ │  Image   │     │    │
│  │  │  Agent  │ │  Agent  │ │  Agent   │ │  Agent   │     │    │
│  │  └─────────┘ └─────────┘ └──────────┘ └──────────┘     │    │
│  │  ┌─────────┐ ┌─────────┐                               │    │
│  │  │   SOP   │ │ Adapter │  (+ shared BaseAgent)         │    │
│  │  │  Agent  │ │  Agent  │                               │    │
│  │  └─────────┘ └─────────┘                               │    │
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

## Enterprise Packages

QL-RF includes comprehensive enterprise-grade packages for production deployments:

### 1. RBAC Package (`pkg/rbac/`)

Hierarchical role-based access control with fine-grained permissions.

**System Roles (8 levels):**
- `org_owner` - Full organizational control
- `org_admin` - Organization administration
- `infra_admin` - Infrastructure management
- `security_admin` - Security and compliance management
- `dr_admin` - Disaster recovery operations
- `operator` - Day-to-day operations
- `analyst` - Read-only analysis
- `viewer` - Read-only dashboard access

**Permission Model:**
- **Actions**: read, write, delete, execute, approve, admin
- **Resources**: assets, images, sites, drift, compliance, dr, tasks, organization, audit
- **Permission Sources**: role-based, direct grants, team-based

**Key Features:**
- Hierarchical role inheritance with parent roles
- Resource-level permissions (per-asset, per-site, per-image)
- Team-based permissions for group collaboration
- Time-based permission expiration
- Full audit trail of permission grants/revocations

**Database Functions:**
```sql
check_permission(user_id, org_id, resource_type, resource_id, action) -> boolean
get_user_permissions(user_id, org_id) -> table
```

### 2. Multi-Tenancy Package (`pkg/multitenancy/`)

Organization isolation with quota management and usage tracking.

**Organization Quotas:**
- Max assets, images, sites, users, teams
- Max AI tasks per day
- Max AI tokens per month
- Max concurrent tasks
- Storage limits (bytes)
- API rate limits (per minute/per day)

**Feature Flags:**
- DR operations enabled/disabled
- Compliance frameworks enabled/disabled
- Advanced analytics access
- Custom integrations allowed

**Subscription Plans:**
Pre-configured tiers (Starter, Professional, Enterprise) with:
- Default quota values
- Feature inclusions
- Monthly/annual pricing
- Trial period support
- External billing system integration (Stripe, etc.)

**Usage Tracking:**
Real-time usage counters for:
- Asset count, image count, site count
- User count, team count
- Storage used (bytes)
- AI tasks today, AI tokens this month
- API requests (per minute/per day)

**Database Functions:**
```sql
check_quota(org_id, resource_type, increment) -> boolean
increment_usage(org_id, resource_type, increment)
decrement_usage(org_id, resource_type, decrement)
check_api_rate_limit(org_id) -> boolean
set_tenant_context(org_id, user_id)
```

### 3. Compliance Package (`pkg/compliance/`)

Compliance framework management with pre-populated controls and evidence tracking.

**Supported Frameworks:**
- CIS Benchmarks (Linux Level 1/2, Windows, Kubernetes)
- SOC 2 Type I/II controls
- NIST 800-53 (Rev 5)
- ISO 27001:2013
- PCI-DSS v4.0
- HIPAA Security Rule

**Control Management:**
- Control definitions with severity (critical, high, medium, low)
- Control families and categories
- Implementation guidance
- Assessment procedures
- Automation support levels (automated, hybrid, manual)

**Cross-Framework Mappings:**
- Control mappings between frameworks (equivalent, partial, related)
- Confidence scores for mappings
- Evidence reuse across frameworks

**Assessment Lifecycle:**
1. Create assessment (scope: sites, assets, frameworks)
2. Start assessment (status: pending → in_progress)
3. Evaluate controls (passed, failed, not_applicable, manual_review)
4. Record evidence (screenshots, logs, configs, reports, attestations)
5. Complete assessment with score

**Evidence Management:**
- Evidence types: screenshot, log, config, report, attestation
- Storage integration (S3, Azure Blob, GCS, local)
- Content hashing for integrity
- Validity periods with expiration
- Review workflow (reviewed_by, review_status)
- Auto-collection from scanners

**Exemptions:**
- Control exemptions with risk acceptance
- Compensating controls documentation
- Time-boxed expiration
- Periodic review requirements
- Approval workflow

### 4. Audit Trail Package (`pkg/audit/`)

Comprehensive audit logging with configurable retention and export.

**Audit Event Types:**
- Authentication events (login, logout, failed attempts)
- Authorization events (permission grants, denials)
- Data access events (read, write, delete)
- Configuration changes (settings, policies, integrations)
- AI task lifecycle (created, approved, rejected, executed)
- Infrastructure changes (assets, images, sites, DR drills)

**Event Schema:**
```go
type AuditEvent struct {
    ID           uuid.UUID
    OrgID        uuid.UUID
    EventType    string
    ActorID      string
    ActorEmail   string
    ActorIP      string
    ResourceType string
    ResourceID   uuid.UUID
    Action       string
    OldValue     json.RawMessage
    NewValue     json.RawMessage
    Metadata     json.RawMessage
    Timestamp    time.Time
}
```

**Features:**
- Immutable append-only log
- IP address and user agent tracking
- Before/after value tracking
- Retention policies (30d, 90d, 1y, 7y for compliance)
- Compressed cold storage
- Audit log export to S3/Azure/GCS
- CSV/JSON/NDJSON export formats
- Search and filtering capabilities

**Export Configuration:**
```bash
RF_AUDIT_RETENTION_DAYS=90
RF_AUDIT_EXPORT_ENABLED=true
RF_AUDIT_EXPORT_BUCKET=s3://audit-logs-bucket
RF_AUDIT_EXPORT_SCHEDULE=daily
```

### 5. Billing Package (`pkg/billing/`)

LLM cost tracking with per-model pricing.

**Cost Tracking:**
- Per-organization LLM usage tracking
- Per-task cost attribution
- Per-model pricing configuration
- Real-time cost accumulation

**Supported Models:**
- Claude (Anthropic): Sonnet 3.5, Opus 3.5, Haiku 3.5
- GPT (OpenAI): GPT-4, GPT-4-Turbo, GPT-3.5-Turbo
- Azure OpenAI: Same models via Azure endpoint

**Pricing Model:**
```go
type LLMPricing struct {
    Provider      string
    Model         string
    InputCostPer1M  float64  // Cost per 1M input tokens
    OutputCostPer1M float64  // Cost per 1M output tokens
}
```

**Usage Metrics:**
```go
type LLMUsage struct {
    OrgID          uuid.UUID
    TaskID         uuid.UUID
    Provider       string
    Model          string
    InputTokens    int64
    OutputTokens   int64
    TotalTokens    int64
    EstimatedCostUSD float64
    Timestamp      time.Time
}
```

**Monthly Cost Reporting:**
- Per-organization monthly summaries
- Per-model cost breakdown
- Token usage trends
- Cost forecasting
- Budget alerts and quota enforcement

### 6. OpenTelemetry Package (`pkg/telemetry/`)

Distributed tracing infrastructure with OpenTelemetry.

**Tracing Features:**
- Automatic span creation for HTTP handlers
- Database query tracing
- LLM API call tracing with token counts
- External service calls (Kafka, Redis, external APIs)
- Error and exception tracking

**Exporters:**
- OTLP (OpenTelemetry Protocol) - default
- Jaeger (for local development)
- Zipkin
- Datadog APM
- AWS X-Ray

**Configuration:**
```bash
RF_OTEL_ENABLED=true
RF_OTEL_EXPORTER=otlp
RF_OTEL_ENDPOINT=http://localhost:4318
RF_OTEL_SERVICE_NAME=ql-rf-api
RF_OTEL_ENVIRONMENT=production
RF_OTEL_SAMPLE_RATE=0.1  # 10% sampling
```

**Instrumentation:**
```go
// Automatic HTTP middleware tracing
router.Use(telemetry.Middleware())

// Manual span creation
ctx, span := telemetry.StartSpan(ctx, "operation-name")
defer span.End()

// Add attributes
span.SetAttributes(
    attribute.String("org.id", orgID),
    attribute.Int64("asset.count", count),
)
```

### 7. Secrets Management Package (`pkg/secrets/`)

HashiCorp Vault integration for secure credential storage.

**Supported Backends:**
- HashiCorp Vault (KV v2 engine)
- AWS Secrets Manager (planned)
- Azure Key Vault (planned)
- GCP Secret Manager (planned)

**Secret Types:**
- Database credentials with auto-rotation
- Cloud provider credentials (AWS, Azure, GCP)
- vCenter credentials
- API keys (Anthropic, OpenAI, Slack, etc.)
- Webhook signing secrets

**Key Features:**
- Secret versioning
- Automatic lease renewal
- Dynamic secret generation (database, cloud IAM)
- Secret rotation policies
- Access audit logging

**Configuration:**
```bash
RF_VAULT_ENABLED=true
RF_VAULT_ADDRESS=https://vault.example.com
RF_VAULT_TOKEN=vault-token
RF_VAULT_NAMESPACE=ql-rf
RF_VAULT_MOUNT_PATH=secret
```

**Usage:**
```go
// Fetch secret
secret, err := secretsClient.GetSecret(ctx, "database/postgres")

// Store secret
err = secretsClient.PutSecret(ctx, "api/anthropic", map[string]string{
    "api_key": "sk-ant-...",
})
```

### 8. SBOM Package (`pkg/sbom/`)

Full Software Bill of Materials generation and management.

**Supported Formats:**
- SPDX 2.3 (JSON)
- CycloneDX 1.5 (JSON)

**Key Features:**
- Container image scanning via Syft integration
- Vulnerability matching with OSV/NVD databases
- License analysis and SPDX identifier mapping
- Component dependency graphing
- Automated SBOM generation on image promotion

**Components:**
```go
pkg/sbom/
├── sbom.go           // Service interface and implementation
├── generator.go      // SBOM generation from container images
├── parser.go         // Parse existing SBOMs (SPDX/CycloneDX)
├── formats.go        // Format conversion utilities
├── vulnerabilities.go // CVE matching and scoring
└── types.go          // Data structures
```

**API Endpoints:**
```
GET  /api/v1/sbom/images/{id}           # Get SBOM for image
POST /api/v1/sbom/images/{id}/generate  # Generate SBOM
GET  /api/v1/sbom/components            # List all components
GET  /api/v1/sbom/vulnerabilities       # Query vulnerabilities
POST /api/v1/sbom/import                # Import external SBOM
GET  /api/v1/sbom/export/{format}       # Export (spdx/cyclonedx)
GET  /api/v1/sbom/licenses              # License summary
```

### 9. FinOps Package (`pkg/finops/`)

Multi-cloud cost optimization and budget management.

**Supported Clouds:**
- AWS (Cost Explorer API)
- Azure (Cost Management API)
- GCP (Cloud Billing API)

**Key Features:**
- Real-time cost collection and aggregation
- Budget management with alerts
- Cost allocation by tags, services, resources
- Optimization recommendations (right-sizing, reserved instances)
- Usage trend analysis and forecasting

**Components:**
```go
pkg/finops/
├── finops.go              // Service interface
├── types.go               // Cost data structures
├── collectors/
│   ├── aws.go            // AWS Cost Explorer collector
│   ├── azure.go          // Azure Cost Management collector
│   └── gcp.go            // GCP Billing API collector
```

**API Endpoints:**
```
GET  /api/v1/finops/costs                 # Get cost data
GET  /api/v1/finops/costs/by-service      # Breakdown by service
GET  /api/v1/finops/costs/by-tag          # Breakdown by tag
GET  /api/v1/finops/budgets               # List budgets
POST /api/v1/finops/budgets               # Create budget
GET  /api/v1/finops/recommendations       # Optimization recommendations
GET  /api/v1/finops/forecast              # Cost forecast
```

### 10. InSpec Package (`pkg/inspec/`)

Automated compliance scanning with Chef InSpec integration.

**Supported Profiles:**
- CIS AWS Foundations Benchmark
- CIS Linux Benchmark
- SOC 2 Type II Controls
- Custom organizational profiles

**Key Features:**
- Temporal workflow-based execution for durability
- Profile-to-control mapping for compliance frameworks
- Automated evidence collection and storage
- Scheduled scan orchestration
- Result aggregation and trending

**Components:**
```go
pkg/inspec/
├── inspec.go              // Service interface
├── types.go               // Scan data structures
├── profiles/
│   ├── cis_aws.go        // CIS AWS profile mapping
│   ├── cis_linux.go      // CIS Linux profile mapping
│   └── soc2.go           // SOC2 control mapping
```

**Temporal Workflows:**
```go
services/orchestrator/internal/temporal/
├── workflows/inspec_workflow.go     // Scan orchestration
└── activities/inspec_activities.go  // Scan execution
```

**API Endpoints:**
```
GET  /api/v1/inspec/profiles              # List profiles
GET  /api/v1/inspec/profiles/{id}         # Get profile details
POST /api/v1/inspec/scans                 # Trigger scan
GET  /api/v1/inspec/scans                 # List scans
GET  /api/v1/inspec/scans/{id}            # Get scan details
GET  /api/v1/inspec/scans/{id}/results    # Get scan results
GET  /api/v1/inspec/scans/{id}/evidence   # Get collected evidence
POST /api/v1/inspec/schedules             # Create scan schedule
GET  /api/v1/inspec/schedules             # List schedules
DELETE /api/v1/inspec/schedules/{id}      # Delete schedule
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

-- Compliance (Migrations 000008-000012)
compliance_frameworks, compliance_controls, control_evidence
compliance_assessments, compliance_assessment_results
compliance_exemptions, control_mappings

-- Audit Trail (Migration 000008)
audit_events, permission_grants_log

-- LLM Cost Tracking (Migration 000009)
llm_usage, llm_pricing

-- RBAC (Migration 000010)
roles, permissions, role_permissions
user_roles, resource_permissions
teams, team_members

-- Multi-Tenancy (Migration 000011)
organization_quotas, organization_usage
subscription_plans, organization_subscriptions

-- DR
dr_pairs, dr_drills, dr_drill_results

-- AI tasks
ai_tasks, ai_task_plans, ai_runs, ai_tool_invocations
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

---

## Phase 4: Automation Features

### Autonomy Modes (`services/orchestrator/internal/autonomy/`)

Five levels of AI autonomy with progressive automation:

| Mode | Description | Human Approval |
|------|-------------|----------------|
| `plan_only` | AI generates plans, humans execute manually | Always |
| `approve_all` | Human approval required for all operations | Always |
| `canary_only` | Auto-execute canary phases, approve full rollout | Canary: Auto, Full: Manual |
| `risk_based` | Auto-execute low/medium risk, approve high/critical | Low/Medium: Auto, High+: Manual |
| `full_auto` | Full automation with guardrails and alerting | Never (alerts only) |

**Configuration:**
```go
type AutonomyConfig struct {
    Mode              AutonomyMode
    MaxRiskLevel      string            // risk_based: max auto-approve level
    RequireCanary     bool              // full_auto: require canary first
    NotifyOnAuto      bool              // Send alerts on auto-execution
    AllowedHours      []int             // Time windows for auto-execution
    ExcludedEnvs      []string          // Environments requiring approval
}
```

**Usage:**
```go
engine := autonomy.NewEngine(config)
decision := engine.ShouldAutoApprove(task, riskLevel)
// Returns: AutoApprove, RequireApproval, or Block
```

### Risk Scoring Service (`services/orchestrator/internal/risk/`)

Calculates operation risk to inform autonomy decisions and batch sizing.

**Risk Factors (8 total):**

| Factor | Weight | Description |
|--------|--------|-------------|
| Asset Criticality | 20% | Production: 100, DR: 70, Staging: 40, Dev: 20 |
| Change Type | 20% | Reimage: 100, Patch: 60, Config: 40, Status: 20 |
| Blast Radius | 15% | % of environment affected |
| Time of Day | 10% | Business hours: 100, Off-hours: 30 |
| Historical Failure | 15% | Past failure rate for similar operations |
| Rollback Complexity | 10% | Easy: 20, Medium: 50, Hard: 100 |
| Dependencies | 5% | Count of dependent services |
| Compliance Impact | 5% | Affects compliance controls: 100, No: 0 |

**Risk Levels:**
- **Low** (0-24): Safe for automation
- **Medium** (25-49): Automation with monitoring
- **High** (50-74): Requires approval
- **Critical** (75-100): Escalation required

**Batch Recommendations:**
```go
scorer.GetBatchSizeRecommendation(riskLevel)
// Low: 25%, Medium: 10%, High: 5%, Critical: 1 asset
```

### Canary Analysis (`services/orchestrator/internal/canary/`)

Progressive rollout validation with metrics-driven promotion.

**Metrics Providers:**
- Prometheus (default)
- CloudWatch (AWS)
- Datadog
- Custom webhook

**Analysis Templates:**

| Template | Duration | Metrics |
|----------|----------|---------|
| Basic | 5 min | Error rate only |
| Standard | 10 min | Error rate, latency |
| Comprehensive | 30 min | Error rate, latency, CPU, memory, custom |

**Thresholds:**
```yaml
thresholds:
  error_rate: 0.01        # 1% max error rate
  latency_p99: 500ms      # p99 latency
  cpu_utilization: 80%    # Max CPU
  memory_utilization: 85% # Max memory
```

**Canary Phases:**
1. Deploy canary (5% traffic)
2. Monitor baseline period
3. Promote to 25% (if passing)
4. Promote to 50% (if passing)
5. Full rollout (100%)
6. Cleanup canary deployment

### Patch-as-Code (`contracts/patch.contract.yaml`)

Declarative patch policies in YAML with JSONSchema validation.

**Contract Structure:**
```yaml
apiVersion: qlrf.io/v1
kind: PatchPolicy
metadata:
  name: critical-security-patches
  namespace: production
spec:
  # Target Selection
  selector:
    platform: [aws, azure, gcp]
    environment: production
    tags:
      compliance: [pci-dss, hipaa]

  # Patch Configuration
  patches:
    severity: [critical, high]
    categories: [security]
    excludeKBs: []

  # Rollout Strategy
  strategy:
    type: canary           # immediate, rolling, canary, blue-green, maintenance-window
    canary:
      initialPercent: 5
      increment: 15
      interval: 10m
      analysisTemplate: standard

    rollback:
      automatic: true
      threshold: 0.05      # 5% failure triggers rollback

  # Schedule
  schedule:
    type: maintenance-window
    windows:
      - day: saturday
        start: "02:00"
        end: "06:00"
        timezone: UTC

  # Notifications
  notifications:
    slack:
      channel: "#ops-alerts"
      events: [started, completed, failed, rollback]
```

**Strategy Types:**

| Strategy | Description | Use Case |
|----------|-------------|----------|
| `immediate` | Apply to all targets at once | Dev/test environments |
| `rolling` | Sequential batches with health checks | Standard deployments |
| `canary` | Progressive rollout with metrics | Production critical |
| `blue-green` | Full parallel deployment swap | Zero-downtime required |
| `maintenance-window` | Scheduled during defined windows | Change-controlled envs |

### CI/CD Pipeline (`.github/workflows/`)

Automated build, test, and deployment pipeline.

**CI Pipeline (`ci.yml`):**
- Go lint (golangci-lint)
- Go test (with PostgreSQL service)
- Go build (matrix: api, connectors, drift, orchestrator)
- Frontend lint & build
- Docker image build & push (main branch)
- Security scan (Trivy)

**CD Pipeline (`cd.yml`):**
- Triggered by CI success on main
- Staging deployment with Helm
- Integration tests on staging
- Canary analysis
- Production approval gate
- Progressive production rollout (5% → 25% → 50% → 100%)
- Automatic rollback on failure

**Deployment Environments:**
- `staging`: Auto-deploy on CI pass
- `production-approval`: Manual approval gate
- `production`: Progressive canary deployment

---

## Phase 5: Quality & Testing

### E2E Testing (`ui/control-tower/e2e/`)

Comprehensive end-to-end testing using Playwright.

**Test Suites:**

| Suite | Description | Tests |
|-------|-------------|-------|
| `navigation.spec.ts` | Sidebar navigation, page routing | 8 |
| `overview.spec.ts` | Dashboard metrics, widgets | 8 |
| `ai-copilot.spec.ts` | AI chat, task submission | 7 |
| `images.spec.ts` | Golden images table, filtering | 4 |
| `drift.spec.ts` | Drift detection, site breakdown | 4 |
| `accessibility.spec.ts` | WCAG compliance, keyboard nav | 8 |

**Commands:**
```bash
npm run test:e2e              # Run all tests
npm run test:e2e:chromium     # Chromium only
npm run test:e2e:headed       # Headed mode (visible browser)
npm run test:e2e:ui           # Playwright UI mode
npm run test:e2e:report       # View HTML report
```

**Configuration (`playwright.config.ts`):**
- 60s test timeout
- 10s assertion timeout
- Screenshot on failure
- Video on failure
- Parallel execution
- Multi-browser: Chromium, Firefox, WebKit
- Mobile viewports: Pixel 5, iPhone 12

### Accessibility (WCAG 2.1 AA)

**Lighthouse Scores:**
- Performance: 70% (dev mode)
- Accessibility: 94%
- Best Practices: 93%
- SEO: 58% (Clerk dev mode)

**Implemented Features:**
- `lang="en"` on HTML element
- Proper heading hierarchy (h1-h6)
- ARIA labels on icon buttons
- `sr-only` text for screen readers
- `aria-hidden="true"` on decorative icons
- Keyboard navigation support
- Focus management in modals
- `main` landmark for content
- `nav` landmark for navigation
- `banner` landmark for header

### LLM Integration

**Provider:** Azure Anthropic (Microsoft Foundry)

**Model:** Claude Sonnet 4.5 (`claude-sonnet-4-5-20241022`)

**Endpoint:** `https://quantumlayer-rf-resource.services.ai.azure.com`

**Configuration:**
```bash
RF_LLM_PROVIDER=azure_anthropic
RF_AZURE_ANTHROPIC_ENDPOINT=https://your-resource.services.ai.azure.com
RF_AZURE_ANTHROPIC_API_KEY=your-api-key
RF_LLM_MODEL=claude-sonnet-4-5-20241022
```

**Features:**
- Streaming responses
- Tool calling with structured outputs
- Human-in-the-loop approval workflow
- Task status tracking (pending, planning, pending_approval, approved, executing, completed, failed)

---

## Current Status (December 2025)

### Completed Features
- Multi-cloud connectors (AWS, Azure, GCP, vSphere, K8s)
- AI Orchestrator with 10 specialist agents
- 29+ infrastructure tools
- Human-in-the-loop approval workflow
- Notification system (Slack, Teams, Email, Webhooks)
- Risk scoring and prediction
- Image lineage tracking with SBOM
- Compliance dashboard with PDF export
- E2E test suite with Playwright
- WCAG 2.1 AA accessibility compliance

### Production Ready
- Health checks (liveness/readiness)
- Kubernetes deployment manifests
- HorizontalPodAutoscaler
- TLS termination
- RBAC with permission gates
- OPA policy validation
- Structured logging (JSON)
- OpenTelemetry tracing
