# QuantumLayer Resilience Fabric (QL-RF)

> AI-Powered Infrastructure Resilience & Compliance Platform

[![License](https://img.shields.io/badge/license-Proprietary-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-0.1.0-green.svg)](CHANGELOG.md)

## Overview

QuantumLayer Resilience Fabric (QL-RF) is an enterprise-grade platform for **golden image management**, **patch orchestration**, and **BCP/DR readiness** across hybrid environments (cloud + on-prem DCs). It provides a single control tower for compliance drift detection, automated patching, DR orchestration, and audit-ready reporting.

## Core Value Proposition

- **Single control tower** for fleet drift, patch parity, and DR readiness
- **Platform-agnostic:** AWS, Azure, GCP, VMware vSphere, bare metal, Kubernetes
- **Contracts-first:** Versioned YAML contracts for images, provisioning, and validation
- **AI-assisted:** CVE triage, canary analysis, RCA generation, and predictive risk alerts
- **Audit-ready:** SBOM, SLSA provenance, and automated compliance evidence

## Quick Start

```bash
# Clone the repository
git clone https://github.com/quantumlayerhq/ql-rf.git
cd ql-rf

# Copy environment variables
cp .env.example .env

# Start infrastructure (Postgres, Redis, Kafka)
make dev-infra

# Run database migrations
make migrate-up

# Run all services in development mode
make dev

# Run tests
make test
```

## Development Commands

```bash
# Build
make build                    # Build all services
make build-api               # Build API service only
make build-connectors        # Build connectors service only
make build-drift             # Build drift service only

# Test
make test                    # Run all tests
make test-coverage           # Run tests with coverage report
make test-race               # Run tests with race detection

# Lint & Format
make lint                    # Run golangci-lint
make fmt                     # Format Go code

# Database
make migrate-up              # Apply migrations
make migrate-down            # Rollback migrations
make migrate-create NAME=x   # Create new migration

# Docker
make docker-build            # Build all Docker images
make docker-up               # Start all services with Docker Compose
make docker-down             # Stop all services

# Development
make dev-infra               # Start only infrastructure (Postgres, Redis, Kafka)
make run-api                 # Run API service locally
make run-connectors          # Run connectors service locally
make run-drift               # Run drift service locally
```

## Project Structure

```
ql-rf/
├── pkg/                     # Shared Go libraries
│   ├── config/             # Viper-based configuration
│   ├── database/           # PostgreSQL connection pool
│   ├── kafka/              # Kafka producer/consumer
│   ├── logger/             # Structured logging (slog)
│   └── models/             # Domain models
├── services/               # Backend microservices
│   ├── api/               # REST API (chi router)
│   │   ├── cmd/api/       # Entry point
│   │   └── internal/      # Handlers, middleware, routes
│   ├── connectors/        # Platform connectors
│   │   ├── cmd/connectors/
│   │   └── internal/      # AWS, Azure, GCP, vSphere
│   ├── drift/             # Drift detection engine
│   │   ├── cmd/drift/
│   │   └── internal/      # Engine, Kafka consumer
│   └── orchestrator/      # AI Orchestrator (LLM-first operations)
│       ├── cmd/orchestrator/
│       └── internal/
│           ├── agents/    # Specialist AI agents (drift, patch, etc.)
│           ├── handlers/  # HTTP handlers
│           ├── llm/       # LLM clients (Anthropic, Azure OpenAI)
│           ├── meta/      # Meta-prompt engine
│           ├── temporal/  # Temporal workflows & activities
│           ├── tools/     # AI tool registry
│           └── validation/ # OPA policy validation
├── migrations/             # Database migrations
├── docs/                   # Documentation
│   ├── PRD.md             # Product Requirements Document
│   ├── FRONTEND_DESIGN.md # Frontend Design System
│   └── adr/               # Architectural Decision Records
├── contracts/              # YAML contracts
├── ui/control-tower/       # Next.js dashboard
│   └── src/
│       ├── app/           # Next.js App Router pages
│       ├── components/    # React components (shadcn/ui)
│       └── hooks/         # React Query hooks
├── policy/                 # OPA/Rego policies
├── infrastructure/         # Infrastructure configs
│   └── temporal/          # Temporal dynamic config
├── go.mod                  # Go module definition
├── go.work                 # Go workspace (multi-module)
├── Makefile                # Build/test/run commands
├── docker-compose.yml      # Local dev infrastructure
└── .env.example            # Environment template
```

## Documentation

| Document | Description |
|----------|-------------|
| [PRD](docs/PRD.md) | Product Requirements Document |
| [Frontend Design](docs/FRONTEND_DESIGN_SYSTEM.md) | Design system and components |
| [Execution Checklist](docs/EXECUTION_CHECKLIST.md) | Implementation tasks |
| [ADRs](docs/adr/) | Architectural Decision Records |

## Architecture

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
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      DATA PLANE                                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐ │
│  │   AWS    │ │  Azure   │ │   GCP    │ │ vSphere  │ │  K8s   │ │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Tech Stack

| Layer | Technologies |
|-------|-------------|
| Backend | Go 1.24, chi router, pgx |
| Database | PostgreSQL 16, Redis 7 |
| Messaging | Apache Kafka (Sarama client) |
| Workflows | Temporal (durable execution) |
| Frontend | Next.js 16, Tailwind, shadcn/ui |
| Auth | Clerk (OIDC/JWT) |
| Cloud SDKs | AWS SDK v2, Azure SDK, GCP, govmomi (vSphere) |
| IaC | Terraform + Helm + Kubernetes |
| Contracts | YAML + JSONSchema + OPA (Rego policies) |
| AI | Azure Anthropic (Claude) / OpenAI API |
| Policy Engine | Open Policy Agent (OPA) |
| Observability | Prometheus + Grafana + OpenTelemetry |

## API Endpoints

### Core API (Port 8080)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/healthz` | Liveness probe (always returns 200 if running) |
| GET | `/readyz` | Readiness probe (checks DB, Kafka, Redis) |
| GET | `/version` | Build info and git commit |
| GET | `/metrics` | Prometheus metrics (if enabled) |
| GET | `/api/v1/images` | List golden images |
| POST | `/api/v1/images` | Register new image |
| GET | `/api/v1/images/{family}/latest` | Get latest version |
| GET | `/api/v1/images/{id}/lineage` | Image lineage (parents, children, vulns) |
| GET | `/api/v1/images/families/{family}/lineage-tree` | Family tree view |
| GET | `/api/v1/images/{id}/vulnerabilities` | CVE list for image |
| POST | `/api/v1/images/{id}/vulnerabilities/import` | Import scanner results (Trivy, Grype, Snyk, etc.) |
| GET | `/api/v1/images/{id}/builds` | Build provenance history |
| GET | `/api/v1/images/{id}/deployments` | Where image is deployed |
| GET | `/api/v1/images/{id}/components` | SBOM components |
| POST | `/api/v1/images/{id}/sbom` | Import SBOM (SPDX, CycloneDX, Syft) |
| GET | `/api/v1/assets` | List assets with filters |
| GET | `/api/v1/assets/{id}` | Get asset details |
| GET | `/api/v1/drift` | Current drift report |
| GET | `/api/v1/drift/summary` | Drift summary by scope |
| GET | `/api/v1/drift/trends` | Drift trends over time |
| GET | `/api/v1/risk/summary` | Organization-wide risk summary |
| GET | `/api/v1/risk/top` | Top risk assets |

### AI Orchestrator API (Port 8083)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Orchestrator health check |
| POST | `/api/v1/ai/execute` | Execute AI task from natural language |
| GET | `/api/v1/ai/tasks` | List pending/completed tasks |
| GET | `/api/v1/ai/tasks/{id}` | Get task details |
| POST | `/api/v1/ai/tasks/{id}/approve` | Approve task for execution |
| POST | `/api/v1/ai/tasks/{id}/reject` | Reject task |
| POST | `/api/v1/ai/tasks/{id}/modify` | Modify task plan |
| GET | `/api/v1/ai/agents` | List available AI agents |
| GET | `/api/v1/ai/tools` | List available tools |

## Roadmap

### Phase 1: Foundation ✅
- [x] Repository setup
- [x] Go backend services (API, Connectors, Drift)
- [x] Database schema and migrations
- [x] Kafka event streaming
- [x] Control Tower dashboard MVP
- [x] OPA policy engine integration

### Phase 2: AI-First Operations ✅
- [x] AI Orchestrator service
- [x] Meta-prompt engine (natural language → TaskSpec)
- [x] Specialist AI agents (drift, patch, compliance, DR, etc.)
- [x] Tool registry with database queries
- [x] OPA validation pipeline
- [x] Human-in-the-loop (HITL) approval workflows
- [x] Temporal workflow integration
- [x] AI Copilot UI with task approval cards

### Phase 3: Expansion ✅
- [x] Azure connector with real Azure SDK calls
- [x] GCP connector with real Google Cloud SDK calls
- [x] vSphere connector with govmomi SDK
- [x] RBAC-based UI visibility (PermissionGate)
- [x] DR drill Temporal workflows
- [x] E2E integration test suite

### Phase 4: Golden Image Lineage ✅
- [x] Image lineage data model (parent-child relationships)
- [x] Build provenance tracking (SLSA-compatible)
- [x] Vulnerability (CVE) tracking per image
- [x] Deployment tracking (where images run)
- [x] SBOM components storage
- [x] Lineage API endpoints
- [x] Lineage visualization UI (tree view, vulnerability cards)
- [x] Interactive lineage graph visualization (canvas-based)
- [x] Vulnerability trend charts with time-range filtering
- [x] Scanner integration API (Trivy, Grype, Snyk, Clair, Anchore, Aqua, Twistlock, Qualys)
- [x] SBOM import API (SPDX, CycloneDX, Syft formats)

### Phase 5: Production Readiness ✅
- [x] Real database queries for drift engine (baselines, fleet assets, scope aggregation)
- [x] Real database queries for compliance service (frameworks, controls, image compliance)
- [x] Production configuration validation (fail-fast for missing required config)
- [x] Multi-tenant middleware with database-backed org lookup
- [x] Health check endpoints with Kafka/Redis status
- [x] Task modification/cancellation with proper state validation

### Phase 6: Risk Scoring & Kubernetes ✅
- [x] AI-powered risk scoring with weighted factors (drift age, vulns, compliance, environment)
- [x] Risk summary and top risks API endpoints
- [x] Risk dashboard UI with gauge, tables, and trend charts
- [x] Kubernetes deployment manifests (Kustomize)
- [x] HorizontalPodAutoscaler configuration
- [x] Ingress with TLS termination
- [x] Security contexts (non-root, read-only FS)

### Phase 7: Full Automation (Next)
- [ ] Predictive risk scoring
- [ ] Full DR failover orchestration
- [ ] Auto-remediation mode
- [ ] Event bridge to QuantumLayer

## Kubernetes Deployment

Production deployment uses Kustomize with manifests in `deployments/kubernetes/`.

### Quick Deploy

```bash
# Deploy to Kubernetes cluster
kubectl apply -k deployments/kubernetes/

# Verify deployment
kubectl get pods -n ql-rf

# Check services
kubectl get svc -n ql-rf
```

### Components

| Component | Replicas | Description |
|-----------|----------|-------------|
| ql-rf-api | 3 | Core REST API |
| ql-rf-orchestrator | 2 | AI Orchestrator |
| ql-rf-ui | 2 | Control Tower UI |

### Scaling

HorizontalPodAutoscaler is configured for all services:
- CPU target: 70%
- Memory target: 80%
- Scale-down stabilization: 5 minutes

```bash
# Manual scaling
kubectl scale deployment/ql-rf-api --replicas=5 -n ql-rf
```

### Configuration

Update secrets before production deployment:

```bash
# Edit secrets (use sealed-secrets or external-secrets in production)
kubectl edit secret ql-rf-secrets -n ql-rf
```

Required secrets:
- `database-url` - PostgreSQL connection string
- `redis-url` - Redis connection string
- `clerk-secret-key` - Clerk authentication
- `anthropic-api-key` - AI features

### Ingress

TLS-enabled ingress with cert-manager:
- `control-tower.quantumlayer.dev` → UI
- `api.quantumlayer.dev` → API + Orchestrator

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is proprietary software. See [LICENSE](LICENSE) for details.

## Contact

- **Author:** Subrahmanya Satish Gonella
- **Email:** satish@quantumlayer.dev
- **Website:** https://quantumlayer.dev
