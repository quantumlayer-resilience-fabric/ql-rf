# Patch-as-Code

> Declarative patch policies in YAML

**Location:** `contracts/patch.contract.yaml`

---

## Overview

Patch-as-Code enables declarative, version-controlled patch policies. Define your patching strategy in YAML, commit to git, and let the platform execute it with full audit trail.

---

## Contract Structure

```yaml
apiVersion: qlrf.io/v1
kind: PatchPolicy
metadata:
  name: policy-name
  namespace: environment
  labels:
    team: platform
    compliance: pci-dss
spec:
  selector:     # Target selection
  patches:      # What to patch
  strategy:     # How to roll out
  schedule:     # When to execute
  notifications: # Who to notify
```

---

## Target Selection

### By Platform
```yaml
selector:
  platform:
    - aws
    - azure
    - gcp
```

### By Environment
```yaml
selector:
  environment: production
```

### By Tags
```yaml
selector:
  tags:
    compliance:
      - pci-dss
      - hipaa
    tier:
      - web
      - api
```

### By Asset Name Pattern
```yaml
selector:
  namePattern: "prod-web-*"
```

### Combined Selectors
```yaml
selector:
  platform: [aws]
  environment: production
  tags:
    compliance: [pci-dss]
  namePattern: "prod-*"
```

---

## Patch Configuration

```yaml
patches:
  severity:
    - critical
    - high
  categories:
    - security
    - bugfix
  excludeKBs:
    - KB5001234  # Known problematic patch
  includeKBs:
    - KB5005678  # Force include specific patch
  maxAge: 30d    # Only patches released in last 30 days
```

---

## Rollout Strategies

### Immediate
Apply to all targets at once. Use for dev/test environments.

```yaml
strategy:
  type: immediate
```

### Rolling
Sequential batches with health checks between each.

```yaml
strategy:
  type: rolling
  rolling:
    batchSize: 10%
    interval: 5m
    maxUnavailable: 1
```

### Canary
Progressive rollout with metrics-driven promotion.

```yaml
strategy:
  type: canary
  canary:
    initialPercent: 5
    increment: 15
    interval: 10m
    analysisTemplate: standard
    thresholds:
      errorRate: 0.01
      latencyP99: 500ms
```

### Blue-Green
Full parallel deployment swap.

```yaml
strategy:
  type: blue-green
  blueGreen:
    prePromotionAnalysis: true
    autoPromotion: false
    scaleDownDelaySeconds: 30
```

### Maintenance Window
Execute only during defined windows.

```yaml
strategy:
  type: maintenance-window
```

---

## Rollback Configuration

```yaml
strategy:
  rollback:
    automatic: true           # Auto-rollback on failure
    threshold: 0.05           # 5% failure rate triggers rollback
    timeoutMinutes: 30        # Max time before rollback check
    preserveRollbackHistory: 3  # Keep last 3 rollback states
```

---

## Schedule Configuration

### One-Time
```yaml
schedule:
  type: once
  at: "2025-12-15T02:00:00Z"
```

### Recurring
```yaml
schedule:
  type: recurring
  cron: "0 2 * * 6"  # Every Saturday at 2 AM
  timezone: UTC
```

### Maintenance Window
```yaml
schedule:
  type: maintenance-window
  windows:
    - day: saturday
      start: "02:00"
      end: "06:00"
      timezone: America/New_York
    - day: sunday
      start: "02:00"
      end: "06:00"
      timezone: America/New_York
```

### Event-Driven
```yaml
schedule:
  type: event
  trigger: cve.critical  # Execute on critical CVE detection
  delay: 24h             # Wait 24 hours for vendor patches
```

---

## Notifications

```yaml
notifications:
  slack:
    channel: "#ops-alerts"
    events:
      - started
      - completed
      - failed
      - rollback
  email:
    recipients:
      - platform-team@example.com
    events:
      - failed
      - rollback
  webhook:
    url: https://api.example.com/patch-events
    events:
      - all
```

---

## Complete Example

```yaml
apiVersion: qlrf.io/v1
kind: PatchPolicy
metadata:
  name: critical-security-patches
  namespace: production
  labels:
    team: platform
    compliance: pci-dss
spec:
  selector:
    platform: [aws, azure, gcp]
    environment: production
    tags:
      compliance: [pci-dss]

  patches:
    severity: [critical, high]
    categories: [security]
    maxAge: 7d

  strategy:
    type: canary
    canary:
      initialPercent: 5
      increment: 15
      interval: 10m
      analysisTemplate: standard
    rollback:
      automatic: true
      threshold: 0.05

  schedule:
    type: maintenance-window
    windows:
      - day: saturday
        start: "02:00"
        end: "06:00"
        timezone: UTC

  notifications:
    slack:
      channel: "#security-patches"
      events: [started, completed, failed, rollback]
```

---

## API Usage

### Apply Policy

```bash
curl -X POST http://localhost:8083/api/v1/policies \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/yaml" \
  --data-binary @patch-policy.yaml
```

### List Policies

```bash
curl http://localhost:8083/api/v1/policies \
  -H "Authorization: Bearer $TOKEN"
```

### Get Policy Status

```bash
curl http://localhost:8083/api/v1/policies/critical-security-patches/status \
  -H "Authorization: Bearer $TOKEN"
```

### Trigger Manual Execution

```bash
curl -X POST http://localhost:8083/api/v1/policies/critical-security-patches/execute \
  -H "Authorization: Bearer $TOKEN"
```

---

## Validation

Policies are validated against JSONSchema before application:

```bash
# Validate locally
qlrf validate policy patch-policy.yaml

# Dry-run
curl -X POST "http://localhost:8083/api/v1/policies?dryRun=true" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/yaml" \
  --data-binary @patch-policy.yaml
```

---

## Example Policies

Located in `contracts/examples/`:

| File | Purpose |
|------|---------|
| `patch-critical-security.yaml` | Critical security patches with canary |
| `patch-monthly-maintenance.yaml` | Monthly maintenance window patching |
| `patch-kubernetes-rolling.yaml` | Kubernetes rolling update policy |
| `patch-emergency-hotfix.yaml` | Emergency hotfix with immediate rollout |

---

## See Also

- [Canary Analysis](CANARY_ANALYSIS.md) - Canary deployment validation
- [Risk Scoring](RISK_SCORING.md) - Risk assessment for operations
- [ARCHITECTURE.md](../ARCHITECTURE.md) - Full system architecture
