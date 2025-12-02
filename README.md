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
│   └── drift/             # Drift detection engine
│       ├── cmd/drift/
│       └── internal/      # Engine, Kafka consumer
├── migrations/             # Database migrations
├── docs/                   # Documentation
│   ├── PRD.md             # Product Requirements Document
│   └── FRONTEND_DESIGN.md # Frontend Design System
├── contracts/              # YAML contracts (coming)
├── ui/control-tower/       # Next.js dashboard (coming)
├── policy/                 # OPA/Rego policies (coming)
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
| Backend | Go 1.23, chi router, sqlc |
| Database | PostgreSQL 16, Redis 7 |
| Messaging | Apache Kafka (Sarama client) |
| Frontend | Next.js 14, Tailwind, shadcn/ui |
| Auth | Clerk (OIDC/JWT) |
| Cloud SDKs | AWS SDK v2, Azure SDK, GCP, govmomi (vSphere) |
| IaC | Terraform + Helm + Kubernetes |
| Contracts | YAML + JSONSchema + OPA (Rego policies) |
| AI | OpenAI API / Anthropic Claude |
| Observability | Prometheus + Grafana + OpenTelemetry |

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/healthz` | Liveness probe |
| GET | `/readyz` | Readiness probe |
| GET | `/version` | Build info |
| GET | `/api/v1/images` | List golden images |
| POST | `/api/v1/images` | Register new image |
| GET | `/api/v1/images/{family}/latest` | Get latest version |
| GET | `/api/v1/assets` | List assets with filters |
| GET | `/api/v1/assets/{id}` | Get asset details |
| GET | `/api/v1/drift` | Current drift report |
| GET | `/api/v1/drift/summary` | Drift summary by scope |
| GET | `/api/v1/drift/trends` | Drift trends over time |

## Roadmap

### Phase 1: Foundation
- [x] Repository setup
- [x] Go backend services (API, Connectors, Drift)
- [x] Database schema and migrations
- [x] Kafka event streaming
- [ ] Control Tower dashboard MVP
- [ ] Contract format v1

### Phase 2: Expansion
- [ ] AI Insight Engine
- [ ] Event bridge to QuantumLayer
- [ ] RBAC and multi-tenancy
- [ ] DR simulation hooks

### Phase 3: Automation
- [ ] Patch-as-Code workflows
- [ ] Predictive risk scoring
- [ ] Full DR failover orchestration

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is proprietary software. See [LICENSE](LICENSE) for details.

## Contact

- **Author:** Subrahmanya Satish Gonella
- **Email:** satish@quantumlayer.dev
- **Website:** https://quantumlayer.dev
