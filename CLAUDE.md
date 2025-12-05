# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**Last Updated:** 2025-12-05

## Project Overview

QuantumLayer Resilience Fabric (QL-RF) is an **LLM-first infrastructure resilience platform** that transforms traditional dashboard-based infrastructure operations into AI-driven, human-approved automation.

### Core Capabilities

- **Golden Image Management** - Lifecycle management across AWS, Azure, GCP, vSphere, and Kubernetes with full lineage tracking
- **Patch Drift Detection** - Real-time drift analysis with AI-powered remediation recommendations
- **BCP/DR Automation** - Disaster recovery readiness scoring, failover testing, and automated DR drills
- **Compliance Evidence** - SBOM generation, SLSA attestations, and CIS benchmark validation
- **AI Copilot** - Natural language interface for infrastructure operations with human-in-the-loop approval
- **Enterprise RBAC** - Hierarchical roles with resource-level permissions and team collaboration
- **Multi-Tenancy** - Organization quotas, usage tracking, subscription plans, and API rate limiting
- **Compliance Frameworks** - Pre-populated CIS, SOC2, NIST, ISO 27001, PCI-DSS, HIPAA controls
- **Audit Trail** - Comprehensive audit logging with configurable retention and export
- **LLM Cost Tracking** - Per-organization usage tracking with per-model pricing

### Codebase Metrics

| Metric | Count |
|--------|-------|
| Go Services | 4 |
| Go Files | 185 |
| Go LOC | ~73,000 |
| Test Files | 50 |
| Migrations | 12 |
| UI Components | 60 |
| Dashboard Pages | 15 |
| AI Agents | 10 |
| Tools | 29+ |
| OPA Policies | 6 |

## Quick Start

```bash
# 1. Start infrastructure (PostgreSQL, Redis, Kafka, Temporal, OPA)
make dev

# 2. Run database migrations
make migrate-up

# 3. Start all services (in separate terminals or use docker-compose)
go run ./services/api/cmd/api              # Port 8080
go run ./services/orchestrator/cmd/orchestrator  # Port 8083

# 4. Start frontend
cd ui/control-tower && npm install && npm run dev  # Port 3000

# Or run everything with Docker Compose:
docker-compose up -d
```

## Development Commands

### Build & Run

```bash
# Build all services
make build

# Run individual services
go run ./services/api/cmd/api              # REST API (Port 8080)
go run ./services/orchestrator/cmd/orchestrator  # AI Orchestrator (Port 8083)
go run ./services/connectors/cmd/connectors      # Cloud Connectors (Port 8081)
go run ./services/drift/cmd/drift                # Drift Engine (Port 8082)

# Run with environment
source .env && go run ./services/api/cmd/api
```

### Testing

```bash
# All tests
go test ./...

# Unit tests only (fast, no external deps)
go test ./... -short
make test-unit

# Integration tests (requires make dev first)
make test-integration

# Single service
go test ./services/api/...

# Single test by name
go test -run TestName ./...

# With coverage
go test -cover ./...
make test-coverage

# Race detection
go test -race ./...
make test-race
```

### Code Quality

```bash
# Lint
golangci-lint run ./...
make lint

# Format
go fmt ./...
make fmt

# Vet
go vet ./...
make vet

# Tidy modules
go mod tidy
make tidy
```

### Database

```bash
# Run migrations
make migrate-up

# Rollback last migration
make migrate-down

# Create new migration
make migrate-create NAME=add_feature

# Check migration version
make migrate-version
```

### Contracts & Policy

```bash
# Validate OpenAPI contracts
make contracts-validate

# Generate API types
make contracts-generate

# OPA policy tests
make opa-test

# OPA format check
make opa-fmt
```

### Docker

```bash
# Build all images
make docker-build

# Build specific service
make docker-build-api
make docker-build-orchestrator

# Push images
make docker-push
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          EXPERIENCE LAYER                                │
│  ┌────────────────────────────┐  ┌────────────────────────────────────┐ │
│  │     Control Tower UI        │  │          AI Copilot                │ │
│  │   Next.js 16 + shadcn/ui    │  │   NL tasks, approval, execution   │ │
│  │   React 19 + TanStack Query │  │   Chat interface + task history   │ │
│  │   Clerk Auth                │  │                                    │ │
│  └────────────────────────────┘  └────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          CONTROL PLANE                                   │
│  ┌──────────────┐ ┌────────────────┐ ┌─────────────┐ ┌───────────────┐  │
│  │  API Service │ │ AI Orchestrator│ │   Drift     │ │  Connectors   │  │
│  │  (Port 8080) │ │  (Port 8083)   │ │   Engine    │ │   Service     │  │
│  │              │ │                │ │ (Port 8082) │ │  (Port 8081)  │  │
│  │  23 endpoints│ │ 10 agents      │ │             │ │  5 platforms  │  │
│  │  Chi router  │ │ 29+ tools      │ │ Kafka-driven│ │               │  │
│  └──────────────┘ └────────────────┘ └─────────────┘ └───────────────┘  │
│                           │                                              │
│                           ▼                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │                      TEMPORAL CLUSTER                                ││
│  │   TaskExecutionWorkflow │ DRDrillWorkflow │ Durable Activities      ││
│  └─────────────────────────────────────────────────────────────────────┘│
│                           │                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │                      POLICY ENGINE (OPA)                             ││
│  │   plan_safety │ tool_authorization │ image_safety │ sop_safety      ││
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
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           DATA LAYER                                     │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                     │
│  │  PostgreSQL  │ │    Redis     │ │    Kafka     │                     │
│  │  (Primary DB)│ │   (Cache)    │ │   (Events)   │                     │
│  └──────────────┘ └──────────────┘ └──────────────┘                     │
└─────────────────────────────────────────────────────────────────────────┘
```

## Project Structure

```
ql-rf/
├── go.work                    # Go workspace (root + services/*)
├── pkg/                       # Shared libraries (23 files, ~4,800 LOC)
│   ├── auth/                  # Clerk authentication utilities
│   ├── config/                # Environment config loading
│   ├── database/              # PostgreSQL connection pool
│   ├── kafka/                 # Kafka producer/consumer
│   ├── logger/                # Structured logging (slog)
│   ├── models/                # Domain models (17 files)
│   └── resilience/            # Circuit breaker pattern
│
├── services/
│   ├── api/                   # REST API Service (Port 8080)
│   │   ├── cmd/api/           # Entry point
│   │   └── internal/
│   │       ├── handlers/      # HTTP handlers (13 files)
│   │       ├── service/       # Business logic (9 files)
│   │       ├── repository/    # Data access (2 files)
│   │       ├── middleware/    # Auth, logging, rate limiting (4 files)
│   │       └── routes/        # Route definitions
│   │
│   ├── orchestrator/          # AI Orchestrator (Port 8083)
│   │   ├── cmd/orchestrator/  # Entry point
│   │   └── internal/
│   │       ├── agents/        # 10 specialist agents (18 files)
│   │       ├── llm/           # Multi-provider LLM (5 files)
│   │       ├── tools/         # 29+ tools (4 files)
│   │       ├── executor/      # Platform executors (9 files)
│   │       ├── temporal/      # Workflow engine (6 files)
│   │       ├── validation/    # JSON Schema + OPA
│   │       ├── meta/          # Meta-prompt engine
│   │       ├── notifier/      # Slack, Teams, Webhooks
│   │       ├── autonomy/      # Manual/semi-auto/auto modes
│   │       └── middleware/    # Auth, tracing
│   │
│   ├── connectors/            # Cloud Connectors (Port 8081)
│   │   └── internal/
│   │       ├── aws/           # AWS SDK v2
│   │       ├── azure/         # Azure Go SDK
│   │       ├── gcp/           # GCP Go SDK
│   │       ├── vsphere/       # govmomi
│   │       ├── k8s/           # client-go
│   │       └── sync/          # Kafka-driven sync
│   │
│   └── drift/                 # Drift Engine (Port 8082)
│       └── internal/engine/   # Drift detection logic
│
├── ui/control-tower/          # Frontend (Port 3000)
│   ├── src/app/               # Next.js App Router (15 pages)
│   │   ├── (auth)/            # Login, Signup
│   │   ├── (dashboard)/       # Overview, Drift, Images, etc.
│   │   └── (marketing)/       # Landing, Features, Pricing
│   ├── src/components/        # 60 React components
│   ├── src/lib/               # API client, types, utilities
│   └── src/providers/         # Auth, Theme, Query providers
│
├── migrations/                # PostgreSQL migrations (7 files)
├── contracts/                 # OpenAPI & JSON Schema specs
├── policy/                    # OPA/Rego policies (6 files)
├── api/openapi/               # Generated OpenAPI spec
└── docs/                      # Documentation (16 files + ADRs)
```

## Services Deep Dive

### API Service (`:8080`)

Core REST API for the Control Tower UI.

**Key Endpoints:**
```
# Assets & Sites
GET  /api/v1/assets                    # List assets with filters
GET  /api/v1/sites                     # List sites
GET  /api/v1/overview                  # Dashboard summary

# Golden Images
GET  /api/v1/images                    # List image families
GET  /api/v1/images/{id}               # Get image details
GET  /api/v1/images/{id}/lineage       # Full lineage with vulns
GET  /api/v1/images/families/{family}/lineage-tree  # Family tree
POST /api/v1/images/{id}/sbom          # Import SBOM data

# Drift & Compliance
GET  /api/v1/drift                     # Current drift report
GET  /api/v1/compliance                # Compliance status

# Risk & Resilience
GET  /api/v1/risk/summary              # Risk scoring
GET  /api/v1/resilience                # DR pairs and status
POST /api/v1/resilience/drill          # Trigger DR drill

# Alerts
GET  /api/v1/alerts                    # List alerts
POST /api/v1/alerts/{id}/acknowledge   # Acknowledge alert
```

**Authentication:** Clerk JWT with dev mode fallback
**Authorization:** Role-based (admin, engineer, operator, viewer) + permission-based

### AI Orchestrator (`:8083`)

LLM-first operations engine that converts natural language to infrastructure tasks.

**Agents (10 total):**
| Agent | Purpose |
|-------|---------|
| `drift` | Drift detection and analysis |
| `patch` | Patch management and rollout |
| `compliance` | Compliance checking and reporting |
| `dr` | Disaster recovery operations |
| `security` | Security scanning and hardening |
| `cost` | Cost optimization |
| `image` | Golden image lifecycle |
| `sop` | Standard operating procedures |
| `adapter` | Platform-specific adaptations |
| `incident` | Incident response |

**Tools (29+ total):** Query assets, create images, trigger patches, run compliance checks, execute DR drills, etc.

**Endpoints:**
```
POST /api/v1/ai/execute               # Execute NL task
GET  /api/v1/ai/tasks                 # List tasks
GET  /api/v1/ai/tasks/{id}            # Task details
POST /api/v1/ai/tasks/{id}/approve    # Approve pending task
POST /api/v1/ai/tasks/{id}/reject     # Reject pending task
GET  /api/v1/ai/agents                # List available agents
GET  /api/v1/ai/tools                 # List available tools
GET  /api/v1/ai/usage                 # LLM usage metrics
```

**LLM Providers:** Azure Anthropic (default), Anthropic, OpenAI, Azure OpenAI

## Configuration

All configuration via environment variables with `RF_` prefix:

```bash
# Core
RF_ENV=development                     # development | staging | production
RF_LOG_LEVEL=debug                     # debug | info | warn | error
RF_DEV_MODE=true                       # Skip JWT validation in dev

# Database
RF_DATABASE_URL=postgres://postgres:postgres@localhost:5432/qlrf?sslmode=disable

# Messaging
RF_KAFKA_BROKERS=localhost:9092
RF_REDIS_URL=redis://localhost:6379/0

# Authentication (Clerk)
RF_CLERK_PUBLISHABLE_KEY=pk_test_...
RF_CLERK_SECRET_KEY=sk_test_...

# LLM Configuration
RF_LLM_PROVIDER=azure_anthropic        # azure_anthropic | anthropic | openai | azure_openai
RF_LLM_API_KEY=...
RF_LLM_MODEL=claude-sonnet-4-5         # Model name
RF_LLM_MAX_TOKENS=4096
RF_LLM_TEMPERATURE=0.3

# Azure Anthropic Specific
RF_LLM_AZURE_ANTHROPIC_ENDPOINT=https://your-resource.services.ai.azure.com

# Temporal Workflow Engine
RF_TEMPORAL_HOST=localhost
RF_TEMPORAL_PORT=7233
RF_TEMPORAL_NAMESPACE=default

# OPA Policy Engine
RF_OPA_ENABLED=true
RF_OPA_MODE=embedded                   # embedded | remote
RF_OPA_POLICIES_DIR=./policy

# Notifications (optional)
RF_NOTIFICATIONS_SLACK_ENABLED=true
RF_NOTIFICATIONS_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
RF_NOTIFICATIONS_TEAMS_ENABLED=true
RF_NOTIFICATIONS_TEAMS_WEBHOOK_URL=https://...webhook.office.com/...

# Enterprise Features
# Audit Trail
RF_AUDIT_RETENTION_DAYS=90                   # Audit log retention period
RF_AUDIT_EXPORT_ENABLED=true                 # Enable audit export
RF_AUDIT_EXPORT_BUCKET=s3://audit-logs       # Export destination
RF_AUDIT_EXPORT_SCHEDULE=daily               # Export schedule

# OpenTelemetry
RF_OTEL_ENABLED=true                         # Enable distributed tracing
RF_OTEL_EXPORTER=otlp                        # OTLP, Jaeger, Zipkin
RF_OTEL_ENDPOINT=http://localhost:4318       # Exporter endpoint
RF_OTEL_SERVICE_NAME=ql-rf-api               # Service name
RF_OTEL_SAMPLE_RATE=0.1                      # Sample 10% of traces

# Secrets Management (HashiCorp Vault)
RF_VAULT_ENABLED=true                        # Enable Vault integration
RF_VAULT_ADDRESS=https://vault.example.com   # Vault server URL
RF_VAULT_TOKEN=...                           # Vault token
RF_VAULT_NAMESPACE=ql-rf                     # Vault namespace
RF_VAULT_MOUNT_PATH=secret                   # KV mount path
```

## Database Schema

PostgreSQL with golang-migrate. Key tables:

```
# Multi-tenancy
organizations, users, organization_quotas, organization_usage
subscription_plans, organization_subscriptions

# Infrastructure
sites, assets

# Golden Images
images, image_lineage, image_vulnerabilities, image_components

# RBAC & Teams
roles, permissions, role_permissions, user_roles
resource_permissions, teams, team_members

# Compliance
compliance_frameworks, compliance_controls, control_mappings
compliance_assessments, compliance_assessment_results
compliance_evidence, compliance_exemptions

# Operations
drift_reports, alerts, activities

# Audit Trail
audit_events, permission_grants_log

# LLM Cost Tracking
llm_usage, llm_pricing

# AI Orchestration
ai_tasks, ai_plans, ai_runs, ai_tool_invocations
```

**Migrations:**
- `000001_init_schema` - Core tables
- `000002_add_connector_status` - Connector tracking
- `000003_add_sites_alerts_compliance` - Alerts and compliance
- `000004_add_ai_orchestration` - AI task lifecycle
- `000005_add_row_level_security` - RLS policies
- `000006_add_image_lineage` - Image family tree
- `000007_add_drift_optimization_indexes` - Query optimization
- `000008_add_audit_trail` - Audit logging infrastructure
- `000009_add_llm_cost_tracking` - LLM usage and cost tracking
- `000010_add_rbac` - RBAC roles, permissions, and teams
- `000011_add_multitenancy` - Organization quotas and subscriptions
- `000012_add_compliance_frameworks` - Compliance frameworks and controls

## Frontend (Control Tower)

Next.js 16 with App Router, located in `ui/control-tower/`:

**Stack:** React 19, Tailwind CSS 4, shadcn/ui, TanStack Query, Clerk Auth

**Commands:**
```bash
cd ui/control-tower
npm install
npm run dev      # localhost:3000
npm run build    # Production build
npm run lint     # ESLint
```

**Pages:**
- `/overview` - Dashboard with metrics
- `/images` - Golden image management with lineage viewer
- `/drift` - Drift analysis and remediation
- `/compliance` - Compliance frameworks and controls
- `/resilience` - DR pairs and failover testing
- `/risk` - Risk scoring and predictions
- `/sites` - Site management
- `/ai` - AI Copilot chat interface
- `/ai/tasks` - Task history
- `/ai/agents` - Agent registry

**Design System:** Dark-first "Command Center" aesthetic. See `docs/FRONTEND_DESIGN_SYSTEM.md`.

## Key Patterns

### Service Structure
Each service follows hexagonal architecture:
```
services/<name>/
├── cmd/<name>/main.go      # Entry point (minimal wiring)
├── internal/
│   ├── handlers/           # HTTP layer (ports)
│   ├── service/            # Business logic (core)
│   └── repository/         # Data access (adapters)
└── go.mod                  # Per-service module
```

### Dependency Injection
Constructor injection in route setup:
```go
func SetupRoutes(db *pgxpool.Pool, log *logger.Logger) *chi.Mux {
    repo := repository.NewRepository(db)
    svc := service.NewService(repo, log)
    h := handlers.NewHandler(svc, log)
    // ...
}
```

### Error Handling
Wrap errors with context:
```go
if err != nil {
    return fmt.Errorf("failed to process asset %s: %w", assetID, err)
}
```

### Structured Logging
Use slog with context:
```go
slog.Info("operation completed",
    "asset_id", asset.ID,
    "platform", asset.Platform,
)
```

## Testing

**Test Files:** 49 total (~18,800 LOC)

**Test Locations:**
```
services/api/internal/handlers/*_test.go      # API handler tests
services/api/internal/service/*_test.go       # Service layer tests
services/orchestrator/internal/agents/*_test.go  # Agent tests
services/orchestrator/internal/llm/*_test.go     # LLM tests
services/connectors/internal/*_test.go        # Connector tests
tests/integration/                            # Integration tests
```

**Running Tests:**
```bash
make test              # All tests
make test-unit         # Unit only
make test-integration  # Integration (needs docker-compose)
make test-coverage     # With coverage report
make test-race         # Race detector
```

## Documentation

See `docs/` for comprehensive documentation:

| Document | Purpose |
|----------|---------|
| `ARCHITECTURE.md` | System design and components |
| `PRD.md` | Product requirements |
| `AGENT_BEHAVIORS.md` | AI agent specifications |
| `AI_SOPS.md` | AI governance procedures |
| `FRONTEND_DESIGN_SYSTEM.md` | UI/UX guidelines |
| `adr/` | Architecture Decision Records (11 ADRs) |
| `features/` | Feature-specific documentation |

## Architecture Decision Records (ADRs)

| ADR | Decision |
|-----|----------|
| ADR-001 | Contracts-first API design |
| ADR-002 | Agentless by default |
| ADR-003 | Cosign image signing |
| ADR-004 | Temporal for durable workflows |
| ADR-005 | OPA for policy enforcement |
| ADR-006 | SPDX for SBOM |
| ADR-007 | LLM-first orchestration |
| ADR-008 | Task-plan-run lifecycle |
| ADR-009 | Tool risk taxonomy & HITL |
| ADR-010 | RBAC authorization model |
| ADR-011 | Row-level security |
| ADR-012 | Enterprise RBAC with hierarchical roles |
| ADR-013 | Multi-tenancy with quota management |
| ADR-014 | Compliance framework integration |
