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

### Notification events

| Event type | When | Who should care |
|---|---|---|
| `task_pending_approval` | Plan generated, awaiting first approval | First approver |
| `task_awaiting_second_approval` (PR #25) | First approval recorded; state_change_prod plan needs a second, distinct approver | Anyone with execute-ai-tasks permission (must differ from first approver) |
| `task_approved` | Plan fully approved; executor about to fire | Operators monitoring execution start |
| `task_rejected` | Plan rejected | Requester |
| `execution_started` / `execution_completed` / `execution_failed` | Run lifecycle | Operators |
| `phase_started` / `phase_completed` / `phase_failed` | Phase lifecycle | Detail-watchers |
| `cve_alert_*` / `campaign_*` | Vulnerability response events | Security + ops |

The `task_awaiting_second_approval` event carries:

- `task_id` — links to the pending decision card in Mission Control
- `user_id` — the first approver (recipients confirm they're not the same person)
- `environment` — pulled from the plan's `blast_radius.environment` field
- `summary` — the task's `user_intent` so a channel viewer sees what's being requested without clicking through

Fire-and-forget: a notifier failure logs a Warn and the approval response is unaffected.

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

## Live LLM development

The docker-compose default for `RF_LLM_PROVIDER` is `stub` — a deterministic,
canned-response provider that makes no external calls and requires no API key.
This is the right default for local dev and for CI E2E. The blocking E2E suite
depends on this determinism; CI also pins `RF_LLM_PROVIDER=stub` explicitly in
the `frontend-e2e` job (belt-and-suspenders).

When the stub is active, the orchestrator's `executeTask` handler
**short-circuits to plan-only**: a submitted prompt creates `ai_tasks` and
`ai_plans` rows in `awaiting_approval` state and **never** invokes
`agent.Execute`, a Temporal workflow, or any cloud SDK. The constructor emits
a loud WARN log at startup:

```
llm: STUB PROVIDER ENABLED — provider=stub model=stub-canned deterministic=true external_calls=false. …
```

Use it as a grep target for any post-mortem.

To enable the real Azure Anthropic provider for local development:

```bash
# in .env at the repo root
RF_LLM_PROVIDER=azure_anthropic
RF_LLM_API_KEY=…
```

Then `docker compose up -d --build orchestrator`. The compose file reads
`${RF_LLM_PROVIDER:-stub}`, so the env override takes effect.

**Never set `RF_LLM_PROVIDER=stub` in production.** The stub will refuse to
contact any external service, the validator path is short-circuited, and
every plan it produces carries an audit marker (`payload._stub: true`) that
will fail any production-grade compliance review. The loud WARN log is a
safeguard, not a license.

See `docs/E2E-011-ai-mission-control.md` for the full Phase B design.

## Conversations (Phase B.2)

Every prompt submitted through Mission Control's dock is captured as a
persistent `ai_conversations` row plus two `ai_conversation_messages` rows
(user + server-synthesized assistant summary). The lifecycle is decided
server-side: successive submissions from the same user within **60 minutes**
fold into the same conversation; outside that window a new conversation
starts. There is no UI affordance to switch threads in B.2 — that lands in
B.3.

The assistant message stored in the thread is a deterministic projection of
the validated `TaskSpec` + `AgentResult` (see
`synthesizeAssistantMessage` in `internal/handlers/conversations.go`). It
never contains JSON, never contains the literal word "stub", and is
identical for stub and live LLM paths. The raw LLM `Content` is preserved
in `ai_conversation_messages.metadata.raw_llm_content` for audit only — the
UI never reads it.

The activity stream gains a `recent_activity` discriminator field on
`GET /api/v1/ai/fleet/status` that unifies tool invocations and user-role
conversation messages. The existing `recent_invocations` field is unchanged
for backward compatibility.

`ai_conversations`, `ai_conversation_messages`, and `ai_tasks.conversation_id`
all land in migration `000019_add_ai_conversations`. The persisted writes —
task, plan, conversation, both messages — happen inside a single
`pgx.Tx` (see `executeTask`); a partial commit is impossible by design.

## Approval simulation (Phase B.3)

When `RF_LLM_PROVIDER=stub` is active and a user clicks Approve on a pending
decision, the orchestrator routes the approval through a deterministic
in-memory simulator instead of touching Temporal, the executor, or any cloud
SDK. The simulation creates a real `ai_runs` row, advances it through
`queued → executing → completed` over ~3 seconds, inserts one synthetic
`ai_tool_invocations` row per plan phase (always at `risk_level='plan_only'`,
never `state_change_*`), and writes structured entries to
`ai_runs.audit_log` at every transition.

Every audit entry carries `"_simulated": true` — a grep target that makes
synthetic and real runs distinguishable in post-mortems:

```sql
SELECT id, jsonb_array_length(audit_log) AS entries,
       audit_log->-1->>'kind' AS last_event
FROM ai_runs
WHERE audit_log @> '[{"_simulated": true}]'::jsonb;
```

The approve response payload also carries `_simulated: true` and a `run_id`.
Production deployments with a real LLM provider (`RF_LLM_PROVIDER=azure_anthropic`)
never reach this branch — the existing Temporal-signal + executor path runs
unchanged. See `services/orchestrator/internal/handlers/approval_simulation.go`
for the simulator's structural guards.

Rejection follows the same conversation-breadcrumb pattern: a `system`-role
message ("✗ Rejected by …") is appended to the task's conversation, but no
`ai_runs` or `ai_tool_invocations` row is ever created for a rejection.

## Run detail (PR #16)

Two new read-only endpoints surface the `ai_runs.audit_log` evidence ledger
in Mission Control:

- `GET /api/v1/ai/runs?limit=N` — recent runs for the caller's org (default 5,
  max 50), ordered by `updated_at DESC` across all lifecycle states. Powers
  the "Recent runs" rail.
- `GET /api/v1/ai/runs/{runID}` — single run with full `audit_log`, phase
  trackers, metrics, and joined `ai_tool_invocations`. 404 on cross-org
  access (same isolation pattern as `getConversationMessages`).

Both endpoints are read-only and additive; they do not modify any state.
The frontend rail polls at 2s while any run is in-flight and 15s otherwise —
so the dashboard animates during a B.3 simulation but stays cheap when idle.

## Real tools (PR #19 / CONN-001)

The orchestrator's tool registry includes the first real cloud-touching tool —
`query_aws_instances`, which calls AWS EC2 `DescribeInstances` (read-only) via
the SDK v2. Registered at boot only when an AWS client can be constructed.

**Three operational modes** (decided at orchestrator startup):

1. **Real AWS client** — `RF_CONNECTORS_AWS_REGION` is set and standard AWS
   credentials are reachable (env vars, profile, or assume-role ARN). The
   client is validated via `sts:GetCallerIdentity` at boot; success → INFO
   log `aws tools: real client initialized`.
2. **Mock fallback** — `RF_CONNECTORS_AWS_FALLBACK_TO_MOCK=true` and the real
   client failed to initialize. A deterministic two-instance fixture
   (`i-mock-0001`, `i-mock-0002`) is returned. A loud WARN log at boot
   announces the fallback. Used by local dev and CI.
3. **Skipped** — no real creds and fallback disabled (production default).
   Tool is simply not registered; `GET /api/v1/ai/tools` doesn't list it.

**Invocation endpoint:** `POST /api/v1/ai/tools/{toolName}/invoke`. Strict
whitelist: only tools with `risk == read_only` are invocable here.
State-change tools return 403; they must flow through the approval pipeline.
The endpoint inserts an `ai_tool_invocations` row WITHOUT the `_simulated`
marker — distinguishable from the B.3 simulator's synthetic rows by the
absence of `parameters._simulated = true`.

**Audit queries:** real vs simulated runs are distinguishable in SQL:

```sql
-- Real invocations (no _simulated marker)
SELECT tool_name, risk_level, duration_ms
FROM ai_tool_invocations
WHERE NOT (parameters @> '{"_simulated": true}'::jsonb)
  AND NOT (result @> '{"_simulated": true}'::jsonb);

-- Simulated invocations (B.3 simulator)
SELECT tool_name, risk_level
FROM ai_tool_invocations
WHERE parameters @> '{"_simulated": true}'::jsonb
   OR result @> '{"_simulated": true}'::jsonb;
```

**Never use `RF_CONNECTORS_AWS_FALLBACK_TO_MOCK=true` in production.** The
loud WARN at boot is the safety net; a misconfigured production deployment
would silently serve fake instance lists. CI sets it explicitly to true to
keep the demo working without real AWS credentials.

## Real Azure tools (PR #26 / CONN-006)

Same registration discipline, different cloud. `query_azure_vms` calls
`armcompute.VirtualMachinesClient.NewListAllPager` (read-only) and lands
in `ai_tool_invocations` with `risk_level='read_only'`. Returns a redacted
projection — no extensions, no NIC IDs, no disk URIs — so an audit-log
leak is low-risk.

Credentials come from the existing `connectors.azure.*` config (TenantID
+ ClientID + ClientSecret + SubscriptionID). The same service-principal
shape the connectors service uses for asset discovery. Set via env:

```
RF_CONNECTORS_AZURE_TENANT_ID=...
RF_CONNECTORS_AZURE_CLIENT_ID=...
RF_CONNECTORS_AZURE_CLIENT_SECRET=...
RF_CONNECTORS_AZURE_SUBSCRIPTION_ID=...
RF_CONNECTORS_AZURE_FALLBACK_TO_MOCK=true  # dev/CI only
```

Boot validates the credential by advancing the VMs pager one page — a
cheap Resource Manager call that surfaces auth misconfiguration loudly
rather than waiting for first real invocation. If validation fails:

- `fallback_to_mock=true` → mock client active, loud WARN, fixed pair of
  `mock-vm-prod-01` + `mock-vm-stage-02` returned with a `mock_origin`
  tag so audit-log SQL can filter.
- `fallback_to_mock=false` → tool not registered; the orchestrator boots
  normally and `GET /api/v1/ai/tools` simply doesn't list `query_azure_vms`.

**Never use `RF_CONNECTORS_AZURE_FALLBACK_TO_MOCK=true` in production.**
Same safety story as AWS. CI sets it true so the read-only demo arc
works without a real Azure subscription.

## Azure Run Command dry-run (PR #27 / CONN-007)

The first Azure state-change tool — dry-run only. `azure_run_command`
constructs an Azure VM Run Command plan (RunShellScript or
RunPowerShellScript document, target VM, inline script) and records it
as audit; no call to the state-change SDK constructor is made.

The structural guarantee:

- `azure_run_command_client.go` builds the plan as a plain Go struct.
- A Go test (`no_azure_runcommand_sdk_import_test.go`) scans every
  non-test file in the tools package and fails CI if any references
  the state-change Run Command client constructor by name. PR #28 will
  add `live_azure_runcommand_client.go` as the single allowlist
  exception.

The pattern mirrors PR #20's SSM dry-run exactly. The Azure structural
test uses **function-name matching** instead of import-path forbidding,
because `armcompute/v5` is legitimately used by PR #26's read-only path —
only the state-change-specific constructor must be off-limits.

**Invocable only** via PR #20's `/api/v1/ai/tools/{name}/dry-run`
endpoint. PR #19's `/invoke` strictly rejects state-change tools.

The seeded compliance mapping links both `azure_run_command` (dry-run)
and `azure_run_command_live` (PR #28 placeholder) to CIS-1.4, so
dry-run invocations of the Azure tool produce `compliance_evidence`
attestations the same way SSM dry-run does.

**PR #28** will introduce `live_azure_runcommand_client.go` — the sole
allowlisted caller of the state-change SDK constructor — with the same
four-gate safety pattern PR #21 used for SSM live (env opt-in,
mock-conflict refusal, per-VM whitelist, two-approver workflow).

## State-change dry-run (PR #20 / CONN-002)

The orchestrator's tool registry now includes the first state-change cloud
tool — `ssm_send_patch_command`. Risk level `state_change_prod`. Invocable
only via a new endpoint `POST /api/v1/ai/tools/{toolName}/dry-run` that
strictly accepts state-change tools (read-only and plan-only return 403,
symmetric to PR #19's `/invoke` gating).

**The tool is dry-run only in PR #20.** It builds an `SSMCommandPlan`
struct describing the AWS-RunPatchBaseline command that WOULD be sent, but
never calls `ssm:SendCommand`. The structural guarantee:

- `services/orchestrator/internal/tools/ssm_client.go` deliberately does
  NOT import `github.com/aws/aws-sdk-go-v2/service/ssm`.
- A Go test (`no_ssm_sdk_import_test.go`) parses every non-test file in
  the tools package at test time and fails if the forbidden import shows
  up. CI runs this test as part of `go test ./services/orchestrator/...`.

**Three-way SQL audit distinction** — every `ai_tool_invocations` row in
the product falls into exactly one of:

```sql
-- Synthetic (B.3 simulator):
WHERE parameters @> '{"_simulated": true}'::jsonb

-- Real read-only (PR #19):
WHERE risk_level = 'read_only'
  AND NOT (parameters @> '{"_simulated": true}'::jsonb)

-- State-change dry-run (PR #20):
WHERE risk_level IN ('state_change_nonprod', 'state_change_prod')
  AND parameters @> '{"dry_run": true}'::jsonb
```

A future fourth kind — state-change live (PR #21) — will be queryable as
`risk_level LIKE 'state_change_%' AND parameters @> '{"dry_run": false}'`.
The audit shape is forward-compatible.

## Live state-change (PR #21 / CONN-003)

The first cloud-mutating call in the orchestrator. `ssm_send_patch_command_live`
actually fires `ssm:SendCommand`. Four independent gates control reachability:

| Gate | Where | Trigger |
|------|-------|---------|
| Env opt-in | `cmd/orchestrator/main.go:registerSSMLiveTools` | `RF_CONNECTORS_AWS_ALLOW_LIVE_PATCH=true`. Default off everywhere. |
| Mock-conflict refusal | same | Boot fails if `FALLBACK_TO_MOCK=true` is also set. |
| Per-instance whitelist | same + `live_ssm_client.go` | `RF_CONNECTORS_AWS_LIVE_PATCH_WHITELIST_INSTANCE_IDS=i-001,i-002,...` — non-empty required at boot. Tool re-validates before SDK call. |
| Two-approver workflow | `handlers/co_approve.go` + `policy/tool_authorization.rego` | First approver hits `/approve`; second, distinct approver hits `/co-approve`. OPA also enforces. |

**Structural isolation:** `live_ssm_client.go` is the ONLY file in the
tools package allowed to import `aws-sdk-go-v2/service/ssm`. The negative
half (no other file may import) is enforced by
`TestNoSSMSDKImportInToolsPackage`; the positive half (this file MUST
import) is enforced by `TestLiveSSMClient_IsTheOnlyFileImportingSDK`.
Both run on every push.

**Four-way SQL audit distinction** — every `ai_tool_invocations` row now
falls into exactly one of:

```sql
-- Synthetic (B.3 simulator):
WHERE parameters @> '{"_simulated": true}'::jsonb

-- Real read-only (PR #19):
WHERE risk_level = 'read_only'
  AND NOT (parameters @> '{"_simulated": true}'::jsonb)

-- State-change dry-run (PR #20):
WHERE risk_level = 'state_change_prod'
  AND parameters @> '{"dry_run": true}'::jsonb

-- Live state-change (PR #21):
WHERE risk_level = 'state_change_prod'
  AND parameters @> '{"dry_run": false}'::jsonb
```

**Two-approver flow:**

1. Plan generated. State `awaiting_approval`. `approved_by` and
   `second_approver` both NULL.
2. First user: `POST /api/v1/ai/tasks/{id}/approve`. The handler detects
   the plan references a `state_change_prod` tool and DIVERTS: records
   `approved_by` but state stays `awaiting_approval`. Response includes
   `status: awaiting_second_approval`.
3. Second user (must differ): `POST /api/v1/ai/tasks/{id}/co-approve`.
   Atomic UPDATE sets `second_approver`, `second_approved_at`, flips
   state to `approved`. Executor fires.
4. Executor invokes `ssm_send_patch_command_live`. Live client validates
   whitelist, calls `ssm:SendCommand`, returns command_id. Audit row has
   `dry_run:false`, `real_changes:true`, `command_id` set.

**Local smoke (mock live client):**

```sh
export RF_CONNECTORS_AWS_FALLBACK_TO_MOCK=false
export RF_CONNECTORS_AWS_ALLOW_LIVE_PATCH=true
export RF_CONNECTORS_AWS_LIVE_PATCH_WHITELIST_INSTANCE_IDS=i-0a1b2c3d4e5f6a7b8
export RF_CONNECTORS_AWS_LIVE_PATCH_CLIENT_MODE=mock  # avoids real AWS in dev
go run ./services/orchestrator/cmd/orchestrator
# Look for: "LIVE SSM MODE ENABLED — real cloud mutations possible after two-approver workflow"
```

In production, set `LIVE_PATCH_CLIENT_MODE=real` (or omit — `real` is the
default). CI keeps `ALLOW_LIVE_PATCH=false`, so the live tool isn't
registered and no live calls are possible regardless of credentials.

**Mission Control UI for co-approval** is the next PR — until it lands,
the `/co-approve` endpoint is API-only.

## Compliance evidence emission (PR #24 / CONN-004)

Every real (non-synthetic) `ai_tool_invocations` row now produces a
`compliance_evidence` attestation when a `tool_compliance_mappings` row
maps the tool name to a `compliance_controls.id` for the row's org.
Mappings are **opt-in** — a tool without a mapping silently produces no
evidence.

The four-way audit kind distinction from PR #21 still holds; PR #24 adds
the auditor layer on top:

| Audit kind | Evidence emitted? | Notes |
|---|---|---|
| Synthetic (B.3) | No | `_simulated:true` in params skips the emitter |
| Real read-only (PR #19) | Yes if mapped | `query_aws_instances` ships unmapped by default; ops can add an SOC2-CC7.1 mapping per-org |
| State-change dry-run (PR #20) | Yes if mapped | Seed maps `ssm_send_patch_command*` → CIS-1.4 |
| Live state-change (PR #21) | Yes if mapped | Same `ssm_send_patch_command*` mapping covers the live tool |

### Lookup precedence

For each invocation the emitter walks `tool_compliance_mappings`:

1. Org-specific exact-name match (`org_id=X, tool_name_pattern=tool_name`)
2. Org-specific wildcard match (`org_id=X, tool_name_pattern=prefix*`)
3. Global exact-name match (`org_id IS NULL, tool_name_pattern=tool_name`)
4. Global wildcard match (`org_id IS NULL, tool_name_pattern=prefix*`)

A customer can override the global default for a specific tool by
inserting an org-specific row — it wins by precedence even if it maps
to a different control.

### Schema

`tool_compliance_mappings` (migration `000020`):

```sql
id, org_id, tool_name_pattern, control_id, notes, created_at, updated_at
```

`compliance_evidence` gains one column:

```sql
ai_tool_invocation_id UUID REFERENCES ai_tool_invocations(id) ON DELETE SET NULL
```

### Adding a new mapping

```sql
-- Global default — applies to all orgs unless overridden.
INSERT INTO tool_compliance_mappings (org_id, tool_name_pattern, control_id, notes)
VALUES (NULL, 'azure_run_command*', '<control_uuid>', 'Azure VM patch operations');

-- Org-specific override.
INSERT INTO tool_compliance_mappings (org_id, tool_name_pattern, control_id, notes)
VALUES ('<org_uuid>', 'ssm_send_patch_command_live', '<soc2_control_uuid>',
        'Customer X also tracks SSM patches against SOC2 CC7.1');
```

### Audit trail (SQL view)

```sql
SELECT
  CASE
    WHEN p.parameters @> '{"_simulated":true}'::jsonb       THEN 'synthetic'
    WHEN p.risk_level = 'read_only'                          THEN 'real_readonly'
    WHEN p.parameters @> '{"dry_run":true}'::jsonb           THEN 'dry_run_statechange'
    WHEN p.parameters @> '{"dry_run":false}'::jsonb
         AND p.risk_level = 'state_change_prod'              THEN 'live_statechange'
    ELSE 'unknown'
  END AS audit_kind,
  COUNT(p.id)              AS invocations,
  COUNT(e.id)              AS attested,
  COUNT(p.id) - COUNT(e.id) AS unmapped
FROM ai_tool_invocations p
LEFT JOIN compliance_evidence e ON e.ai_tool_invocation_id = p.id
GROUP BY 1
ORDER BY 1;
```

### Executor audit gap (closed by PR #24)

Before PR #24, tools fired via the executor's `executeAction` path
(post-approval, e.g. via the co-approve workflow) produced **no audit
row at all** — the executor called `tool.Execute()` without writing to
`ai_tool_invocations`. PR #24 fixes this incidentally because the
compliance emitter needs the audit row to link to. Both the audit and
the evidence emission are now best-effort: a DB failure logs a Warn and
the tool's original return value is preserved unchanged.

## License

Copyright © 2024 QuantumLayer. All rights reserved.
