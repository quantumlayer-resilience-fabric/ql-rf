# Canary Analysis

> Progressive rollout validation with metrics-driven promotion

**Location:** `services/orchestrator/internal/canary/analyzer.go`

---

## Overview

Canary analysis validates deployments by comparing canary metrics against baseline before promoting to full rollout. This reduces risk by catching issues early with minimal blast radius.

---

## Canary Phases

```
5% Traffic → Monitor → 25% → Monitor → 50% → Monitor → 100%
     │          │         │        │        │        │
     └──────────┴─────────┴────────┴────────┴────────┘
                    Pass/Fail at each stage
```

| Phase | Traffic | Duration | Action on Failure |
|-------|---------|----------|-------------------|
| Canary | 5% | 5-30 min | Immediate rollback |
| Early | 25% | 3-5 min | Rollback |
| Mid | 50% | 3-5 min | Rollback |
| Full | 100% | - | Monitor |

---

## Metrics Providers

### Prometheus (Default)
```go
type PrometheusProvider struct {
    URL      string
    Queries  map[string]string
}
```

### CloudWatch (AWS)
```go
type CloudWatchProvider struct {
    Region    string
    Namespace string
    Metrics   []MetricConfig
}
```

### Datadog
```go
type DatadogProvider struct {
    APIKey  string
    AppKey  string
    Site    string
}
```

### Custom Webhook
```go
type WebhookProvider struct {
    URL     string
    Headers map[string]string
}
```

---

## Analysis Templates

### Basic (5 minutes)
```yaml
name: basic
duration: 5m
metrics:
  - name: error_rate
    query: "rate(http_requests_total{status=~\"5..\"}[5m])"
    threshold: 0.01
    comparison: less_than
```

### Standard (10 minutes)
```yaml
name: standard
duration: 10m
metrics:
  - name: error_rate
    query: "rate(http_requests_total{status=~\"5..\"}[5m])"
    threshold: 0.01
    comparison: less_than
  - name: latency_p99
    query: "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))"
    threshold: 0.5
    comparison: less_than
```

### Comprehensive (30 minutes)
```yaml
name: comprehensive
duration: 30m
metrics:
  - name: error_rate
    threshold: 0.01
  - name: latency_p99
    threshold: 0.5
  - name: cpu_utilization
    threshold: 0.8
  - name: memory_utilization
    threshold: 0.85
  - name: custom_business_metric
    query: "custom_metric{app=\"myapp\"}"
    threshold: 100
```

---

## Configuration

```go
type CanaryConfig struct {
    Provider        string            // prometheus, cloudwatch, datadog, webhook
    ProviderConfig  map[string]string // Provider-specific settings
    Template        string            // basic, standard, comprehensive
    Thresholds      map[string]float64
    PromotionSteps  []int             // [5, 25, 50, 100]
    IntervalMinutes int               // Time between promotions
    AutoRollback    bool              // Rollback on failure
}
```

### Environment Variables

```bash
RF_CANARY_PROVIDER=prometheus
RF_CANARY_PROMETHEUS_URL=http://prometheus:9090
RF_CANARY_TEMPLATE=standard
RF_CANARY_AUTO_ROLLBACK=true
RF_CANARY_INTERVAL_MINUTES=10
```

---

## API Usage

### Start Canary Analysis

```bash
curl -X POST http://localhost:8083/api/v1/ai/tasks/{id}/canary \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "template": "standard",
    "initial_percent": 5,
    "target_percent": 100,
    "interval_minutes": 10
  }'
```

### Check Canary Status

```bash
curl http://localhost:8083/api/v1/ai/tasks/{id}/canary/status \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "phase": "early",
  "current_percent": 25,
  "status": "passing",
  "metrics": {
    "error_rate": {
      "value": 0.005,
      "threshold": 0.01,
      "passing": true
    },
    "latency_p99": {
      "value": 0.234,
      "threshold": 0.5,
      "passing": true
    }
  },
  "next_promotion_at": "2025-12-04T10:30:00Z"
}
```

### Manual Promotion

```bash
curl -X POST http://localhost:8083/api/v1/ai/tasks/{id}/canary/promote \
  -H "Authorization: Bearer $TOKEN"
```

### Rollback Canary

```bash
curl -X POST http://localhost:8083/api/v1/ai/tasks/{id}/canary/rollback \
  -H "Authorization: Bearer $TOKEN"
```

---

## Analysis Flow

```go
func (a *Analyzer) Run(ctx context.Context, task *Task) error {
    for _, percent := range a.config.PromotionSteps {
        // Deploy to percentage
        if err := a.deploy(ctx, task, percent); err != nil {
            return a.rollback(ctx, task)
        }

        // Wait for baseline period
        time.Sleep(a.config.BaselineDuration)

        // Collect and compare metrics
        result := a.analyze(ctx, task)
        if !result.Passing {
            return a.rollback(ctx, task)
        }

        // Notify progress
        a.notifier.Send(CanaryProgress{
            TaskID:  task.ID,
            Percent: percent,
            Status:  "passing",
        })
    }

    return nil
}
```

---

## Rollback Triggers

Automatic rollback occurs when:

1. **Error rate exceeds threshold** - Default: >1%
2. **Latency exceeds threshold** - Default: p99 >500ms
3. **Resource exhaustion** - CPU >80% or Memory >85%
4. **Health check failures** - >3 consecutive failures
5. **Manual trigger** - User-initiated rollback

---

## Integration with CI/CD

The canary analysis integrates with the CD pipeline:

```yaml
# .github/workflows/cd.yml
- name: Deploy canary (5%)
  run: |
    helm upgrade --install ql-rf-canary ./deploy/helm/ql-rf \
      --set global.canary.enabled=true \
      --set global.canary.weight=5

- name: Monitor canary (5 minutes)
  run: |
    sleep 300
    kubectl -n ql-rf-production get pods -l app.kubernetes.io/name=ql-rf-canary

- name: Promote canary to 25%
  run: |
    helm upgrade --install ql-rf-canary ./deploy/helm/ql-rf \
      --reuse-values \
      --set global.canary.weight=25
```

---

## See Also

- [Risk Scoring](RISK_SCORING.md) - Risk assessment for operations
- [Autonomy Modes](AUTONOMY_MODES.md) - Automation configuration
- [ARCHITECTURE.md](../ARCHITECTURE.md) - Full system architecture
