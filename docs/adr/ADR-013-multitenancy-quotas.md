# ADR-013: Multi-Tenancy with Quota-Based Resource Management

## Status
Accepted

## Context

QL-RF is transitioning from a single-tenant platform to a SaaS offering serving multiple organizations with varying sizes and requirements. We need to:

1. **Ensure Strict Isolation**: One organization must not access another's data or resources
2. **Prevent Resource Exhaustion**: A single organization cannot consume unlimited platform resources
3. **Enable Tiered Pricing**: Different subscription plans with different resource limits
4. **Track Usage**: Monitor resource consumption for billing and capacity planning
5. **Enforce Fair Use**: Rate limiting and quotas to prevent abuse

Challenges to address:
- **Scale**: Platform must support thousands of organizations
- **Performance**: Quota checks cannot significantly impact API latency
- **Flexibility**: Different organizations have different resource needs
- **Growth**: Organizations need to upgrade/downgrade plans seamlessly
- **Billing Integration**: Usage data must integrate with external billing systems (Stripe, etc.)

## Decision

We implement a **quota-based multi-tenancy system** with the following architecture:

### 1. Organization Quotas

Each organization has configurable quotas across 10 resource types:

| Quota Type | Description | Default (Starter) | Default (Enterprise) |
|------------|-------------|-------------------|----------------------|
| `assets` | Maximum infrastructure assets | 100 | 10,000 |
| `images` | Maximum golden images | 10 | 500 |
| `sites` | Maximum geographic sites | 5 | 100 |
| `users` | Maximum user accounts | 10 | Unlimited |
| `teams` | Maximum teams | 3 | 50 |
| `ai_tasks` | AI tasks per day | 20 | 500 |
| `ai_tokens` | AI tokens per month | 1M | 100M |
| `storage` | Storage in bytes | 10GB | 10TB |
| `api_requests` | API rate limit (per minute) | 100/min | 10,000/min |
| `concurrent_tasks` | Max concurrent AI tasks | 3 | 50 |

**Quota Configuration**:
```go
type OrganizationQuota struct {
    OrgID                 uuid.UUID
    MaxAssets             int
    MaxImages             int
    MaxSites              int
    MaxUsers              int
    MaxTeams              int
    MaxAITasksPerDay      int
    MaxAITokensPerMonth   int64
    MaxConcurrentTasks    int
    MaxStorageBytes       int64
    MaxArtifactSizeBytes  int64
    APIRateLimitPerMinute int
    APIRateLimitPerDay    int
}
```

### 2. Real-Time Usage Tracking

Usage counters updated in real-time via database functions:

```sql
-- Increment usage (with quota check)
SELECT increment_usage(org_id, 'assets', 1);

-- Decrement usage (when resource deleted)
SELECT decrement_usage(org_id, 'assets', 1);

-- Check quota before operation
SELECT check_quota(org_id, 'assets', 1); -- Returns boolean
```

**Usage Metrics**:
```go
type OrganizationUsage struct {
    OrgID                uuid.UUID
    AssetCount           int
    ImageCount           int
    SiteCount            int
    UserCount            int
    TeamCount            int
    StorageUsedBytes     int64
    AITasksToday         int        // Resets daily
    AITokensThisMonth    int64      // Resets monthly
    APIRequestsToday     int        // Resets daily
    APIRequestsMinute    int        // Rolling 1-minute window
    UpdatedAt            time.Time
}
```

### 3. Subscription Plans

Pre-configured plans with associated quotas and features:

**Plan Tiers**:
- **Starter** (Free/Trial): Basic quotas, core features only
- **Professional**: 10x Starter quotas, DR and compliance enabled
- **Enterprise**: Unlimited users, advanced analytics, custom integrations

**Plan Schema**:
```go
type SubscriptionPlan struct {
    Name                   string
    DisplayName            string
    PlanType               string   // free, paid, enterprise
    DefaultMaxAssets       int
    DefaultMaxImages       int
    DefaultMaxAITasks      int
    DefaultMaxAITokens     int64
    DRIncluded             bool
    ComplianceIncluded     bool
    AdvancedAnalytics      bool
    CustomIntegrations     bool
    MonthlyPriceUSD        *float64
    AnnualPriceUSD         *float64
}
```

### 4. Feature Flags

Quota-based feature enablement:

```go
// Check if organization has feature enabled
enabled, err := multitenancy.IsFeatureEnabled(ctx, orgID, "dr")
enabled, err := multitenancy.IsFeatureEnabled(ctx, orgID, "compliance")
enabled, err := multitenancy.IsFeatureEnabled(ctx, orgID, "advanced_analytics")
enabled, err := multitenancy.IsFeatureEnabled(ctx, orgID, "custom_integrations")
```

**Feature Gating**:
- **DR Operations**: Requires `dr_enabled = true` in quota
- **Compliance Frameworks**: Requires `compliance_enabled = true`
- **Advanced Analytics**: Predictive risk scoring, anomaly detection
- **Custom Integrations**: ServiceNow, webhooks, custom OIDC providers

### 5. Database Implementation

**Core Tables**:
- `organization_quotas` - Quota limits per organization
- `organization_usage` - Real-time usage counters
- `subscription_plans` - Plan templates
- `organization_subscriptions` - Active subscriptions with billing info

**Database Functions**:
```sql
-- Check and enforce quota
check_quota(org_id, resource_type, increment) → boolean

-- Increment usage (fails if quota exceeded)
increment_usage(org_id, resource_type, increment) → void

-- Decrement usage
decrement_usage(org_id, resource_type, decrement) → void

-- Check API rate limit (per-minute and per-day)
check_api_rate_limit(org_id) → boolean

-- Set tenant context for Row-Level Security
set_tenant_context(org_id, user_id) → void
clear_tenant_context() → void
```

### 6. API Rate Limiting

Two-tier rate limiting:

**Per-Minute Limit**:
- Rolling 1-minute window tracked in `organization_usage.api_requests_this_minute`
- Updated on every API request
- Enforced via middleware before request processing

**Per-Day Limit**:
- Daily counter in `organization_usage.api_requests_today`
- Resets at midnight UTC
- Soft limit with warnings

**Rate Limit Response**:
```http
HTTP/1.1 429 Too Many Requests
Retry-After: 42
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1701792000

{
  "error": "rate_limit_exceeded",
  "message": "API rate limit exceeded. Retry after 42 seconds.",
  "quota_type": "api_requests",
  "limit": 100,
  "current": 100
}
```

## Consequences

### Positive

1. **Scalability**: Prevents any single organization from overwhelming the platform
2. **Predictable Costs**: Quotas enable accurate capacity planning and cost forecasting
3. **Fair Resource Allocation**: All organizations get fair access to shared resources
4. **Flexible Pricing**: Easy to create new subscription tiers with different quotas
5. **Billing Integration**: Usage data directly maps to billing events
6. **Self-Service Upgrades**: Organizations can upgrade when hitting quotas

### Negative

1. **User Friction**: Users may encounter quota errors requiring plan upgrades
2. **Database Load**: Real-time usage tracking adds database overhead
3. **Migration Complexity**: Existing single-tenant deployments need data migration
4. **Edge Cases**: Quota enforcement during peak loads can fail operations
5. **Customer Support**: More support tickets related to quota limits

### Mitigations

1. **Graceful Degradation**: Soft quotas with warnings before hard limits
2. **Database Optimization**:
   - Materialized views for usage aggregation
   - Write-optimized schema with minimal contention
   - Periodic cleanup of old usage data
3. **Migration Tools**:
   - Automated migration scripts
   - Data validation and reconciliation
4. **Quota Buffer**: 10% buffer before hard limits with proactive notifications
5. **Support Automation**:
   - Self-service plan upgrades
   - Usage dashboards showing quota consumption
   - Automated alerts at 80% and 95% thresholds

## Implementation Notes

### Quota Check Flow

```go
1. Middleware extracts org_id from auth context
2. Before resource creation:
   - Call check_quota(org_id, resource_type, 1)
   - If false, return 429 with quota details
   - If true, proceed
3. After successful creation:
   - Call increment_usage(org_id, resource_type, 1)
4. On resource deletion:
   - Call decrement_usage(org_id, resource_type, 1)
```

### Usage Reset Jobs

**Daily Reset** (00:00 UTC):
```sql
UPDATE organization_usage
SET ai_tasks_today = 0, api_requests_today = 0
WHERE updated_at < CURRENT_DATE;
```

**Monthly Reset** (1st of month, 00:00 UTC):
```sql
UPDATE organization_usage
SET ai_tokens_this_month = 0
WHERE EXTRACT(DAY FROM updated_at) != 1;
```

### Subscription Lifecycle

**Trial Start**:
```sql
INSERT INTO organization_subscriptions (org_id, plan_id, status, trial_ends_at)
SELECT org_id, (SELECT id FROM subscription_plans WHERE name = 'starter'), 'trialing', NOW() + INTERVAL '14 days'
```

**Trial Expiration** (automated job):
```sql
UPDATE organization_subscriptions
SET status = 'expired'
WHERE status = 'trialing' AND trial_ends_at < NOW()
```

**Plan Upgrade**:
```sql
UPDATE organization_subscriptions SET plan_id = $new_plan_id, updated_at = NOW()
WHERE org_id = $org_id;

-- Update quotas from new plan
UPDATE organization_quotas
SET (max_assets, max_images, ...) = (SELECT ... FROM subscription_plans WHERE id = $new_plan_id)
WHERE org_id = $org_id;
```

### Integration with External Billing

**Stripe Webhooks**:
- `customer.subscription.created` → Create subscription
- `customer.subscription.updated` → Update plan
- `customer.subscription.deleted` → Cancel subscription
- `invoice.payment_succeeded` → Extend current period

**Usage-Based Billing**:
```sql
-- Export monthly usage for billing
SELECT org_id,
       ai_tokens_this_month,
       storage_used_bytes,
       api_requests_total
FROM organization_usage
WHERE org_id = $org_id
AND updated_at >= DATE_TRUNC('month', CURRENT_DATE)
```

## Migration Path

**Phase 1** (Completed):
- Create multi-tenancy tables (quotas, usage, plans, subscriptions)
- Implement database functions for quota checks
- Implement Go service layer (`pkg/multitenancy`)

**Phase 2** (In Progress):
- Add quota enforcement middleware to API services
- Implement usage tracking in resource handlers
- Create usage reset background jobs

**Phase 3** (Planned):
- Build subscription management UI
- Integrate with Stripe for payments
- Add usage dashboards
- Implement quota upgrade flow

## Performance Considerations

**Read Optimization**:
- Quota checks cached in-memory (1-minute TTL)
- Usage counters read from replica databases
- API rate limit cached in Redis

**Write Optimization**:
- Usage increments use `UPDATE ... SET counter = counter + 1` (no locks)
- Batch usage updates for bulk operations
- Async decrement on resource deletion

**Scalability**:
- Horizontal scaling: Each org's data is independent
- Sharding strategy: Organization ID as shard key
- Read replicas for quota/usage queries

## References

- Migration 000011: Multi-tenancy database schema
- `pkg/multitenancy/tenant.go`: Core multi-tenancy service
- PRD Section 21: Monetization Strategy
- Stripe API Documentation: https://stripe.com/docs/api/subscriptions
