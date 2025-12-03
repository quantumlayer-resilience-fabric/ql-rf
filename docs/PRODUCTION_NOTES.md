# Production Deployment Notes

This document outlines known limitations, configuration requirements, and considerations for deploying QL-RF to production.

## Configuration Requirements

### Required Environment Variables

Before deploying to production, ensure these variables are set:

| Variable | Description | Example |
|----------|-------------|---------|
| `RF_ENV` | Must be `production` | `production` |
| `RF_DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@host:5432/qlrf?sslmode=require` |
| `RF_REDIS_URL` | Redis connection string | `redis://:password@host:6379/0` |
| `RF_CLERK_SECRET_KEY` | Clerk authentication secret | `sk_live_xxx` |
| `RF_CLERK_PUBLISHABLE_KEY` | Clerk publishable key | `pk_live_xxx` |
| `RF_LLM_API_KEY` | LLM provider API key | See LLM Configuration |
| `RF_NOTIFICATION_APP_BASE_URL` | Base URL for email links | `https://control-tower.quantumlayer.dev` |

### LLM Configuration

Configure one of these LLM providers:

```bash
# Azure Anthropic (Recommended)
RF_LLM_PROVIDER=azure_anthropic
RF_LLM_AZURE_ANTHROPIC_ENDPOINT=https://your-resource.services.ai.azure.com
RF_LLM_API_KEY=your-key
RF_LLM_MODEL=claude-sonnet-4-5

# Direct Anthropic
RF_LLM_PROVIDER=anthropic
RF_LLM_API_KEY=sk-ant-xxx
RF_LLM_MODEL=claude-sonnet-4-5-20250514
```

## Known Limitations

### 1. Simulated Operations

The following operations return simulated/mock responses and require implementation for full production use:

| Component | File | Description |
|-----------|------|-------------|
| DR Drill Operations | `services/orchestrator/internal/temporal/activities/dr_drill_activities.go` | Failover, failback, and sync operations are simulated |

**Implemented Operations:**
- ✅ **Asset Patching** - AWS SSM integration now implemented (`services/connectors/internal/aws/ssm_patcher.go`)
- ✅ **Drift Age Calculation** - Now uses actual drift report timestamps

**Impact**: DR drills will show successful completion but won't actually perform infrastructure changes. This is acceptable for initial deployment with proper documentation.

### 2. Pending Features (TODOs)

These features are stubbed but not fully implemented:

| Feature | Location | Status |
|---------|----------|--------|
| Platform breakdown in asset stats | `asset_service.go:139` | Returns empty map |
| DR notification delivery | `dr_drill_activities.go:407,420` | Logged but not sent |

**Implemented Features:**
- ✅ **MS Teams Notifications** - Adaptive Cards (v1.4) format now supported (`notifier/notifier.go`)

### 3. Rate Limiting

Rate limiting is automatically enabled in production (disabled in development):
- **Default**: 100 requests/second per IP with burst of 200
- **Behavior**: Returns HTTP 429 with `Retry-After: 1` header

To customize, update `middleware/ratelimit.go:DefaultRateLimitConfig()`.

## Security Considerations

### Health Check Commands

The health checker (`executor/health_checker.go`) can execute arbitrary commands specified in health check configurations. Ensure:
1. Health check configurations are admin-controlled only
2. Consider whitelisting allowed commands in production
3. Commands run with service account permissions

### Authentication

- JWT validation is enforced when `RF_CLERK_PUBLISHABLE_KEY` is set
- DevMode (`RF_ORCHESTRATOR_DEV_MODE=true`) bypasses authentication - **never enable in production**

## Performance Recommendations

### Database Connection Pool

```bash
RF_DATABASE_MAX_OPEN_CONNS=25  # Adjust based on load
RF_DATABASE_MAX_IDLE_CONNS=5
RF_DATABASE_CONN_MAX_LIFETIME=5m
```

### HTTP Timeouts

Configured in services:
- API: Read 30s, Write 30s
- Orchestrator: Read 30s, Write 60s (longer for LLM responses)

### Horizontal Pod Autoscaling

Kubernetes HPA is configured for:
- CPU target: 70%
- Memory target: 80%
- Scale-down stabilization: 5 minutes

## Monitoring Checklist

Before going live, verify:

- [ ] Prometheus metrics endpoint (`/metrics`) is being scraped
- [ ] Grafana dashboards are configured
- [ ] Alert rules are set up for:
  - High error rates (5xx responses)
  - LLM API failures
  - Database connection pool exhaustion
  - Rate limit triggers
- [ ] Log aggregation is configured
- [ ] Health check endpoints (`/healthz`, `/readyz`) are monitored

## Rollback Procedure

1. Scale down new deployment: `kubectl scale deployment/ql-rf-api --replicas=0`
2. Apply previous version: `kubectl apply -k deployments/kubernetes/previous/`
3. Verify health: `kubectl get pods -n ql-rf`
4. Check readiness: `curl https://api.quantumlayer.dev/readyz`

## Support

- Issues: https://github.com/quantumlayerhq/ql-rf/issues
- Contact: platform-ops@quantumlayer.dev
