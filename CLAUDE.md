# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

QuantumLayer Resilience Fabric (QL-RF) is an LLM-first infrastructure resilience platform providing:
- Golden image management across multi-cloud (AWS, Azure, GCP, vSphere, Kubernetes)
- Patch drift detection with AI-powered remediation
- BCP/DR readiness scoring and failover automation
- Compliance evidence generation (SBOM, SLSA, CIS benchmarks)

## Development Commands

```bash
# Start infrastructure (PostgreSQL, Redis, Kafka, Temporal)
make dev

# Run database migrations
make migrate-up

# Build all services
make build

# Run individual services
go run ./services/api/cmd/api              # Port 8080
go run ./services/orchestrator/cmd/orchestrator  # Port 8083
go run ./services/connectors/cmd/connectors      # Port 8081
go run ./services/drift/cmd/drift                # Port 8082

# Testing
go test ./...                    # All tests
go test ./... -short             # Unit tests only
go test ./services/api/...       # Single service
go test -run TestName ./...      # Single test by name
go test -cover ./...             # With coverage

# Code quality
golangci-lint run ./...
go fmt ./...

# Database
make migrate-create NAME=add_feature  # New migration
make migrate-down                     # Rollback last
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     CONTROL TOWER UI (Next.js)              │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────┼──────────────────────────────────┐
│  API Service :8080       │  AI Orchestrator :8083           │
│  • Assets, Images        │  • Meta-prompt engine            │
│  • Drift, Compliance     │  • 10 specialist agents          │
│  • Risk scoring          │  • Tool registry (29 tools)      │
└──────────────────────────┴──────────────────────────────────┘
                           │
┌──────────────────────────┼──────────────────────────────────┐
│                    TEMPORAL CLUSTER                         │
│         TaskExecutionWorkflow │ DRDrillWorkflow             │
└─────────────────────────────────────────────────────────────┘
                           │
┌──────────┬───────────┬───────────┬───────────┬──────────────┐
│   AWS    │   Azure   │    GCP    │  vSphere  │  Kubernetes  │
└──────────┴───────────┴───────────┴───────────┴──────────────┘
```

### Go Workspace Structure

The project uses Go workspaces (`go.work`) with separate modules per service:

```
ql-rf/
├── go.work              # Workspace: root + services/*
├── pkg/                 # Shared libraries (auth, config, database, kafka, models)
├── services/
│   ├── api/            # REST API - chi router, handlers, repository pattern
│   ├── connectors/     # Platform SDKs (AWS, Azure, GCP, vSphere, K8s)
│   ├── drift/          # Drift engine with Kafka consumer
│   └── orchestrator/   # AI orchestrator (see below)
├── migrations/         # PostgreSQL migrations (golang-migrate)
└── ui/control-tower/   # Next.js 16 + shadcn/ui + Clerk auth
```

### AI Orchestrator Layers

The orchestrator (`services/orchestrator/`) processes natural language → infrastructure changes:

1. **Meta-prompt engine** (`internal/meta/`) - Parses NL intent → TaskSpec
2. **Agent registry** (`internal/agents/`) - 10 specialist agents (drift, patch, compliance, DR, security, cost, image, SOP, adapter, incident)
3. **Tool registry** (`internal/tools/`) - 29 tools for querying/mutating infrastructure
4. **Validation pipeline** (`internal/validation/`) - JSON Schema + OPA policy validation
5. **Executor** (`internal/executor/`) - Phased execution with health checks and rollback
6. **Temporal workflows** (`internal/temporal/`) - Durable execution for long-running operations

## Key Patterns

### Service Structure

Each service follows the pattern:
```
services/<name>/
├── cmd/<name>/main.go  # Entry point (minimal wiring)
├── internal/
│   ├── handlers/       # HTTP handlers
│   ├── service/        # Business logic
│   └── repository/     # Database access
└── go.mod              # Per-service module
```

### Configuration

All config via environment variables with `RF_` prefix:
```bash
RF_DATABASE_URL=postgres://...
RF_KAFKA_BROKERS=localhost:9092
RF_CLERK_SECRET_KEY=...
RF_LLM_PROVIDER=anthropic         # or azure_openai, openai
RF_LLM_API_KEY=...
RF_LLM_MODEL=claude-3-5-sonnet-20241022
RF_ORCHESTRATOR_DEV_MODE=true     # Skip JWT validation in dev

# Notifications (optional)
RF_NOTIFICATIONS_APP_BASE_URL=https://app.example.com
RF_NOTIFICATIONS_SLACK_ENABLED=true
RF_NOTIFICATIONS_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
RF_NOTIFICATIONS_SLACK_CHANNEL=#alerts
RF_NOTIFICATIONS_TEAMS_ENABLED=true
RF_NOTIFICATIONS_TEAMS_WEBHOOK_URL=https://...webhook.office.com/...
RF_NOTIFICATIONS_WEBHOOK_ENABLED=true
RF_NOTIFICATIONS_WEBHOOK_URL=https://your-webhook-endpoint.com
RF_NOTIFICATIONS_WEBHOOK_SECRET=your-secret-for-hmac
```

### Database

PostgreSQL with golang-migrate. Key tables:
- `organizations`, `users`, `sites`, `assets`
- `images`, `image_lineage`, `image_vulnerabilities`, `image_components`
- `drift_reports`, `compliance_frameworks`, `compliance_controls`
- `ai_tasks`, `ai_plans`, `ai_runs`, `ai_tool_invocations`

### Logging

Use structured logging (slog):
```go
slog.Info("operation completed",
    "asset_id", asset.ID,
    "platform", asset.Platform,
)
```

### Error Handling

Wrap errors with context:
```go
if err != nil {
    return fmt.Errorf("failed to process asset %s: %w", assetID, err)
}
```

## Frontend (Control Tower)

Next.js 16 with App Router, located in `ui/control-tower/`:

```bash
cd ui/control-tower
npm install
npm run dev    # localhost:3000
npm run build
npm run lint
```

Stack: React 19, Tailwind CSS 4, shadcn/ui, TanStack Query, Clerk auth.

Design system: Dark-first "Command Center" aesthetic. See `docs/FRONTEND_DESIGN_SYSTEM.md`.

## Testing

```bash
# Unit tests (fast, no external deps)
make test-unit

# Integration tests (requires make dev first)
make test-integration

# Test files in tests/integration/:
# - orchestrator_test.go - AI orchestrator API
# - api_test.go - Main API endpoints
# - connectors_test.go - Cloud connectors
```

## API Quick Reference

### Core API (:8080)
```
GET  /api/v1/assets              # List assets
GET  /api/v1/images              # List golden images
GET  /api/v1/images/{id}/lineage # Image lineage tree
GET  /api/v1/drift               # Drift report
GET  /api/v1/compliance          # Compliance status
GET  /api/v1/risk/summary        # Risk scoring
```

### AI Orchestrator (:8083)
```
POST /api/v1/ai/execute          # Execute NL task
GET  /api/v1/ai/tasks            # List tasks
POST /api/v1/ai/tasks/{id}/approve  # Approve task
POST /api/v1/ai/tasks/{id}/reject   # Reject task
GET  /api/v1/ai/agents           # List agents
GET  /api/v1/ai/tools            # List tools
```
