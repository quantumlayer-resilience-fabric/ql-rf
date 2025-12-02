# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

QuantumLayer Resilience Fabric (QL-RF) is an AI-powered infrastructure resilience and compliance platform providing:
- Golden image management across multi-cloud and data center environments
- Patch drift detection and orchestration
- BCP/DR readiness scoring and failover automation
- Compliance evidence generation (SBOM, SLSA, CIS benchmarks)

## Technology Stack

| Layer | Technology |
|-------|------------|
| Backend | Go (all services) |
| HTTP Router | chi (stdlib-compatible) |
| Database | PostgreSQL + sqlc |
| Cache | Redis |
| Message Broker | Kafka (Sarama) |
| Auth | Clerk (OIDC) |
| Frontend | Next.js 14, Tailwind, shadcn/ui, Socket.IO |
| IaC | Terraform + Helm + Kubernetes |
| Contracts | YAML + JSONSchema + OPA (Rego) |
| AI | OpenAI / Anthropic Claude APIs |
| Observability | Prometheus + Grafana + OpenTelemetry |
| Workflows | Temporal (Go SDK) |

## Project Structure

```
ql-rf/
├── go.mod              # Root module
├── go.work             # Go workspace
├── Makefile            # Build/test/lint commands
├── docker-compose.yml  # Local dev environment
├── pkg/                # Shared libraries
│   ├── config/        # Viper-based configuration
│   ├── database/      # PostgreSQL connection
│   ├── logger/        # Structured logging (slog)
│   ├── kafka/         # Kafka producer/consumer
│   └── models/        # Domain models
├── services/
│   ├── api/           # Main API service (chi router)
│   │   ├── cmd/api/
│   │   └── internal/
│   ├── connectors/    # Platform connectors
│   │   ├── cmd/connectors/
│   │   └── internal/aws|azure|gcp|vsphere/
│   └── drift/         # Drift detection engine
│       ├── cmd/drift/
│       └── internal/engine/
├── migrations/         # PostgreSQL migrations
├── ui/control-tower/   # Next.js dashboard
├── contracts/          # YAML contracts
├── policy/            # OPA/Rego policies
├── deploy/helm/       # Helm charts
└── docs/              # Documentation
```

## Development Commands

```bash
# Development
make dev              # Start local dev environment (docker-compose up)
make dev-down         # Stop local dev environment

# Build
make build            # Build all services
make build-api        # Build API service only
make build-connectors # Build connectors service only
make build-drift      # Build drift service only

# Testing
make test             # Run all tests
make test-unit        # Run unit tests only
make test-integration # Run integration tests
make test-coverage    # Run tests with coverage report

# Code Quality
make lint             # Run golangci-lint
make fmt              # Format code with gofmt

# Database
make migrate-up       # Run database migrations
make migrate-down     # Rollback last migration
make migrate-create   # Create new migration (NAME=xxx)
make sqlc-generate    # Generate type-safe SQL code

# Docker
make docker-build     # Build Docker images
make docker-push      # Push to registry

# Run individual services
make run-api          # Run API service locally
make run-connectors   # Run connectors service locally
make run-drift        # Run drift service locally
```

## Architecture

### Three-Layer Architecture

1. **Experience Layer**: Control Tower UI (Next.js) + AI Copilot
2. **Control Plane** (K8s): API Gateway, Services, Event Bus
3. **Data Plane**: Multi-cloud connectors (AWS, Azure, GCP, vSphere)

### Core Services

| Service | Port | Description |
|---------|------|-------------|
| api | 8080 | Main REST API, handles all client requests |
| connectors | 8081 | Discovers assets from cloud platforms |
| drift | 8082 | Calculates patch drift, publishes events |

### Key API Endpoints

```
GET  /healthz                      # Liveness probe
GET  /readyz                       # Readiness probe
GET  /version                      # Build info

GET  /api/v1/images                # List golden images
POST /api/v1/images                # Register image
GET  /api/v1/images/{family}/latest # Get latest version

GET  /api/v1/assets                # List assets (with filters)
GET  /api/v1/drift                 # Get drift report
GET  /api/v1/drift?env=prod        # Filtered by environment
```

### Event Bus (Kafka Topics)

| Topic | Publisher | Consumer | Description |
|-------|-----------|----------|-------------|
| asset.discovered | connectors | drift | New/updated asset found |
| drift.detected | drift | api | Drift threshold exceeded |
| image.published | api | connectors | New golden image registered |

## Code Conventions

### Go Project Layout

- `cmd/` - Main applications (minimal code, just wiring)
- `internal/` - Private application code (not importable)
- `pkg/` - Shared libraries (importable by other modules)

### Error Handling

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to fetch assets: %w", err)
}
```

### Logging

```go
// Use structured logging (slog)
slog.Info("asset discovered",
    "platform", asset.Platform,
    "instance_id", asset.InstanceID,
)
```

### Configuration

Environment variables with `RF_` prefix:
- `RF_DATABASE_URL` - PostgreSQL connection string
- `RF_KAFKA_BROKERS` - Comma-separated Kafka brokers
- `RF_REDIS_URL` - Redis connection string
- `RF_CLERK_SECRET_KEY` - Clerk authentication key

## Database

PostgreSQL with these core tables:
- `organizations`, `projects`, `environments` - Multi-tenancy
- `images`, `image_coordinates` - Golden image registry
- `assets` - Fleet inventory
- `drift_reports` - Drift snapshots

Run migrations: `make migrate-up`

## Testing

```bash
# Unit tests (fast, no external deps)
go test ./... -short

# Integration tests (requires docker-compose)
make dev
go test ./... -tags=integration

# Specific package
go test ./services/api/internal/handlers/...
```

## Design System

Frontend uses "Command Center" aesthetic (dark-first, data-dense).
See `docs/FRONTEND_DESIGN_SYSTEM.md` for design tokens and components.
