# AI Orchestrator Service

The AI Orchestrator is the intelligent automation engine for QL-RF. It converts natural language requests into validated, auditable infrastructure operations through specialist AI agents.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     QL-AI-ORCHESTRATOR                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              LAYER 1: META-PROMPT ENGINE                │    │
│  │   User Intent → TaskSpec (agent, tools, validation)     │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              ↓                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              LAYER 2: SPECIALIST AGENTS                 │    │
│  │  DriftAgent │ PatchAgent │ ComplianceAgent │ DRAgent    │    │
│  │  CostAgent │ SecurityAgent │ ImageAgent │ SOPAgent      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              ↓                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              LAYER 3: TOOL REGISTRY                     │    │
│  │   query_assets │ get_golden_image │ generate_patch_plan │    │
│  │   check_control │ simulate_failover │ terraform_plan    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              ↓                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              LAYER 4: VALIDATION PIPELINE               │    │
│  │   Schema │ OPA Policies │ Drift Safety │ HITL Gates     │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              ↓                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              LAYER 5: EXECUTION ENGINE                  │    │
│  │   Phased Rollout │ Health Checks │ Rollback │ Notify    │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

## Features

- **10 Specialist Agents**: Drift, Patch, Compliance, Incident, DR, Cost, Security, Image, SOP, Adapter
- **Quality Score Model**: Multi-dimensional scoring for trust and validation
- **OPA Policy Engine**: Production safety rules (canary, batch limits, environment rules)
- **HITL Workflow**: Human-in-the-loop approval for high-risk operations
- **Execution Engine**: Phased rollout with health checks and automatic rollback
- **Notifications**: Slack, Email, and Webhook notifications for all events

## API Endpoints

### Task Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/ai/execute` | Submit a natural language task |
| `GET` | `/api/v1/ai/tasks` | List all tasks (with optional filters) |
| `GET` | `/api/v1/ai/tasks/{id}` | Get task details |
| `POST` | `/api/v1/ai/tasks/{id}/approve` | Approve a task for execution |
| `POST` | `/api/v1/ai/tasks/{id}/reject` | Reject a task |

### Execution Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/ai/tasks/{id}/executions` | List executions for a task |
| `GET` | `/api/v1/ai/executions/{id}` | Get execution details |
| `POST` | `/api/v1/ai/executions/{id}/pause` | Pause a running execution |
| `POST` | `/api/v1/ai/executions/{id}/resume` | Resume a paused execution |
| `POST` | `/api/v1/ai/executions/{id}/cancel` | Cancel an execution |

### Agents & Tools

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/ai/agents` | List available agents |
| `GET` | `/api/v1/ai/tools` | List available tools |

## API Examples

### Submit a Task

```bash
curl -X POST http://localhost:8083/api/v1/ai/execute \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Fix drift on production web servers",
    "org_id": "org-123",
    "environment": "production",
    "context": {
      "fleet_size": 100,
      "drift_score": 85
    }
  }'
```

Response:
```json
{
  "task_id": "task-abc123",
  "status": "pending_approval",
  "task_spec": {
    "task_type": "drift_remediation",
    "goal": "Remediate drift for production web servers",
    "risk_level": "high",
    "environment": "production"
  },
  "agent_result": {
    "agent_name": "drift_agent",
    "plan": "## Drift Remediation Plan\n...",
    "summary": "47 servers require drift remediation",
    "affected_assets": 47,
    "risk_level": "high"
  },
  "quality_score": {
    "total": 75,
    "structural": 20,
    "policy_compliance": 15,
    "test_coverage": 10,
    "operational_history": 15,
    "human_review": 15,
    "requires_approval": true,
    "allowed_environments": ["staging", "production"]
  },
  "requires_hitl": true
}
```

### Approve a Task

```bash
curl -X POST http://localhost:8083/api/v1/ai/tasks/task-abc123/approve \
  -H "Content-Type: application/json" \
  -d '{"reason": "Approved after review"}'
```

### Get Execution Status

```bash
curl http://localhost:8083/api/v1/ai/executions/exec-xyz789
```

Response:
```json
{
  "id": "exec-xyz789",
  "task_id": "task-abc123",
  "status": "running",
  "started_at": "2024-01-15T10:30:00Z",
  "current_phase": 1,
  "total_phases": 3,
  "phases": [
    {
      "name": "Canary",
      "status": "completed",
      "assets": [
        {"asset_id": "server-1", "status": "completed"},
        {"asset_id": "server-2", "status": "completed"}
      ]
    },
    {
      "name": "Wave 1",
      "status": "running",
      "assets": [
        {"asset_id": "server-3", "status": "running"},
        {"asset_id": "server-4", "status": "pending"}
      ]
    }
  ]
}
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8083` | HTTP server port |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `LOG_FORMAT` | `json` | Log format (json, text) |
| `DATABASE_URL` | - | PostgreSQL connection string |
| `ANTHROPIC_API_KEY` | - | Claude API key |
| `LLM_MODEL` | `claude-sonnet-4-20250514` | LLM model to use |
| `OPA_URL` | - | OPA server URL (optional) |

### Notification Configuration

```yaml
notification:
  slack_enabled: true
  slack_webhook_url: "https://hooks.slack.com/services/..."
  slack_channel: "#infrastructure-alerts"

  email_enabled: true
  smtp_host: "smtp.example.com"
  smtp_port: 587
  smtp_user: "alerts@example.com"
  smtp_password: "..."
  email_from: "QL-RF AI <alerts@example.com>"
  email_to:
    - "ops-team@example.com"

  webhook_enabled: true
  webhook_url: "https://api.example.com/webhooks/ql-rf"
```

## Development

### Prerequisites

- Go 1.21+
- PostgreSQL 14+
- OPA (optional, for policy validation)

### Build

```bash
cd services/orchestrator
go build -o orchestrator ./cmd/orchestrator
```

### Run Tests

```bash
go test ./...
```

### Run with Coverage

```bash
go test -cover ./...
```

### Run Locally

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/qlrf?sslmode=disable"
export ANTHROPIC_API_KEY="sk-ant-..."
./orchestrator
```

## Deployment

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o orchestrator ./cmd/orchestrator

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/orchestrator /orchestrator
EXPOSE 8083
ENTRYPOINT ["/orchestrator"]
```

### Docker Compose

```yaml
services:
  orchestrator:
    build:
      context: .
      dockerfile: services/orchestrator/Dockerfile
    ports:
      - "8083:8083"
    environment:
      - PORT=8083
      - DATABASE_URL=postgres://qlrf:qlrf@postgres:5432/qlrf?sslmode=disable
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
    depends_on:
      - postgres

  postgres:
    image: postgres:14
    environment:
      POSTGRES_USER: qlrf
      POSTGRES_PASSWORD: qlrf
      POSTGRES_DB: qlrf
    volumes:
      - postgres_data:/var/lib/postgresql/data
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ai-orchestrator
  namespace: ql-rf
spec:
  replicas: 2
  selector:
    matchLabels:
      app: ai-orchestrator
  template:
    metadata:
      labels:
        app: ai-orchestrator
    spec:
      containers:
      - name: orchestrator
        image: qlrf/orchestrator:latest
        ports:
        - containerPort: 8083
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: url
        - name: ANTHROPIC_API_KEY
          valueFrom:
            secretKeyRef:
              name: llm-credentials
              key: anthropic-key
        resources:
          requests:
            cpu: 100m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /health
            port: 8083
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8083
          initialDelaySeconds: 5
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: ai-orchestrator
  namespace: ql-rf
spec:
  selector:
    app: ai-orchestrator
  ports:
  - port: 8083
    targetPort: 8083
```

## OPA Policies

The orchestrator uses OPA for production safety validation. Policies are located in `policies/`:

- `production_safety.rego` - Canary requirements, batch limits, rollback criteria
- `sop_safety.rego` - SOP execution guardrails
- `image_safety.rego` - Image build and promotion rules
- `terraform_safety.rego` - Infrastructure change validation

### Example Policy

```rego
package ql.safety

# Require canary phase for production deployments
deny[msg] {
    input.environment == "production"
    not has_canary_phase
    msg := "Production changes require a canary phase"
}

# Limit batch size in production
deny[msg] {
    input.environment == "production"
    some phase
    phase := input.phases[_]
    count(phase.assets) / input.total_assets > 0.2
    msg := sprintf("Batch size %d%% exceeds 20%% limit", [count(phase.assets) / input.total_assets * 100])
}

has_canary_phase {
    input.phases[0].name == "Canary"
}
```

## Agent Types

| Agent | Purpose | Key Tools |
|-------|---------|-----------|
| **DriftAgent** | Detect and remediate configuration drift | query_assets, compare_versions, generate_patch_plan |
| **PatchAgent** | Orchestrate patching operations | cve_feed, risk_score, generate_rollout |
| **ComplianceAgent** | Audit compliance and generate evidence | check_controls, generate_evidence |
| **IncidentAgent** | Investigate incidents and suggest fixes | query_logs, correlate_events |
| **DRAgent** | DR planning and drill execution | infra_graph, simulate_failover |
| **CostAgent** | Cost optimization recommendations | billing_data, forecast, recommend |
| **SecurityAgent** | Vulnerability and misconfiguration scanning | scan_vulns, check_exposure |
| **ImageAgent** | Golden image lifecycle management | build_image, validate, promote |
| **SOPAgent** | Standard Operating Procedure management | generate_sop, validate_sop, execute_sop |
| **AdapterAgent** | Dynamic API integration | discover_api, generate_adapter |

## Quality Score Model

The Quality Score determines trust level and allowed environments:

| Dimension | Weight | Description |
|-----------|--------|-------------|
| Structural | 20% | Valid plan structure, all required fields |
| Policy Compliance | 20% | Passes all OPA policies |
| Test Coverage | 20% | Plan tested in lower environments |
| Operational History | 20% | Agent's track record |
| Human Review | 20% | Prior human approvals |

Score thresholds:
- **80+**: Allowed in production, no approval needed
- **60-79**: Allowed in production with approval
- **40-59**: Staging only, approval required
- **<40**: Development only

## Monitoring

### Health Endpoints

- `GET /health` - Liveness check
- `GET /ready` - Readiness check (includes DB connectivity)

### Metrics (Prometheus)

- `orchestrator_tasks_total` - Total tasks by status
- `orchestrator_executions_total` - Total executions by status
- `orchestrator_agent_latency_seconds` - Agent execution latency
- `orchestrator_tool_calls_total` - Tool calls by name and result

## License

Copyright © 2024 QuantumLayer. All rights reserved.
