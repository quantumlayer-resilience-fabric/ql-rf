# FinOps - Cost Optimization Module

This package provides comprehensive FinOps (Financial Operations) capabilities for QL-RF, enabling cost tracking, optimization, and budget management across multi-cloud infrastructure.

## Features

### Core Capabilities

1. **Cost Tracking**
   - Record and aggregate costs across AWS, Azure, GCP, vSphere, and Kubernetes
   - Track costs by resource, service, region, site, and cloud provider
   - Historical cost data with trend analysis

2. **Cost Optimization**
   - Automated recommendation generation:
     - Rightsizing (downsize overprovisioned resources)
     - Reserved Instances / Savings Plans
     - Spot Instances for fault-tolerant workloads
     - Idle resource detection
     - Storage optimization
     - Unused volume cleanup
   - Potential savings calculations
   - Priority-based recommendation ranking

3. **Budget Management**
   - Create budgets with configurable scopes (organization, cloud, service, site)
   - Automatic threshold monitoring (e.g., alert at 80% spend)
   - Cost alerts when budgets are exceeded
   - Multiple time periods (daily, weekly, monthly, quarterly, yearly)

4. **Cost Analytics**
   - Cost summaries with trend analysis
   - Breakdown by multiple dimensions (cloud, service, region, site, resource type)
   - Time-series cost trends
   - Top resource cost analysis

## Package Structure

```
pkg/finops/
├── finops.go           # Core CostService implementation
├── types.go            # Data types and models
├── finops_test.go      # Unit tests
├── README.md           # This file
└── collectors/         # Cloud-specific cost collectors
    ├── aws.go          # AWS Cost Explorer integration
    ├── azure.go        # Azure Cost Management integration
    └── gcp.go          # GCP Cloud Billing integration
```

## Usage

### Initialize the Service

```go
import (
    "github.com/quantumlayerhq/ql-rf/pkg/finops"
    "github.com/jackc/pgx/v5/pgxpool"
)

// Create service with database connection
db := pgxpool.Pool{...}
costSvc := finops.NewCostService(db)
```

### Get Cost Summary

```go
// Get cost summary for last 30 days
timeRange := finops.NewTimeRangeLast(30)
summary, err := costSvc.GetCostSummary(ctx, orgID, timeRange)
if err != nil {
    // Handle error
}

fmt.Printf("Total Cost: %.2f %s\n", summary.TotalCost, summary.Currency)
fmt.Printf("Trend: %.1f%%\n", summary.TrendChange)
```

### Get Cost Recommendations

```go
recommendations, err := costSvc.GetCostOptimizationRecommendations(ctx, orgID)
if err != nil {
    // Handle error
}

for _, rec := range recommendations {
    fmt.Printf("%s: %s - Save %.2f %s\n",
        rec.Type,
        rec.Action,
        rec.PotentialSavings,
        rec.Currency,
    )
}
```

### Create a Budget

```go
budget := finops.CostBudget{
    OrgID:          orgID,
    Name:           "Monthly AWS Budget",
    Amount:         5000.00,
    Currency:       "USD",
    Period:         string(finops.PeriodMonthly),
    Scope:          string(finops.ScopeCloud),
    ScopeValue:     "aws",
    AlertThreshold: 80.0,
    StartDate:      time.Now(),
    CreatedBy:      userID,
}

created, err := costSvc.CreateBudget(ctx, budget)
```

### Get Cost Trends

```go
// Get daily cost trend for last 90 days
trends, err := costSvc.GetCostTrend(ctx, orgID, 90)
if err != nil {
    // Handle error
}

for _, trend := range trends {
    fmt.Printf("%s: %.2f %s\n",
        trend.Date.Format("2006-01-02"),
        trend.Cost,
        trend.Currency,
    )
}
```

## API Endpoints

The FinOps module exposes the following REST API endpoints via `services/api/internal/handlers/finops.go`:

### Cost Summary
```
GET /api/v1/finops/summary?period=30d
```
Returns aggregated cost data with breakdowns by cloud, service, site.

**Query Parameters:**
- `period`: `7d`, `30d`, `90d`, `365d`, `this_month`, `last_month`

### Cost Breakdown
```
GET /api/v1/finops/breakdown?dimension=cloud&period=30d
```
Returns cost breakdown by a specific dimension.

**Query Parameters:**
- `dimension`: `cloud`, `service`, `region`, `site`, `resource_type`
- `period`: Same as summary

### Cost Trend
```
GET /api/v1/finops/trend?days=30
```
Returns time-series cost data.

**Query Parameters:**
- `days`: Number of days (1-365)

### Optimization Recommendations
```
GET /api/v1/finops/recommendations?type=all
```
Returns cost optimization recommendations.

**Query Parameters:**
- `type`: `all`, `rightsizing`, `reserved_instances`, `spot_instances`, `idle_resources`, `storage_optimization`

### Create Budget
```
POST /api/v1/finops/budgets
Content-Type: application/json

{
  "name": "Monthly AWS Budget",
  "amount": 5000.00,
  "currency": "USD",
  "period": "monthly",
  "scope": "cloud",
  "scope_value": "aws",
  "alert_threshold": 80.0,
  "start_date": "2024-01-01T00:00:00Z"
}
```

### List Budgets
```
GET /api/v1/finops/budgets?active_only=true
```

### Resource Costs
```
GET /api/v1/finops/resources?resource_type=ec2_instance&period=30d
```

## Database Schema

### Tables

1. **cost_records** - Historical cost data
   - Stores resource-level cost data over time
   - Indexed by org, cloud, service, recorded_at
   - Supports JSON tags for metadata

2. **cost_recommendations** - Optimization recommendations
   - Generated recommendations with savings potential
   - Tracked status (pending, applied, dismissed)
   - Priority-based ranking

3. **cost_budgets** - Budget definitions
   - Configurable scopes and alert thresholds
   - Support for multiple time periods
   - Current spend tracking

4. **cost_alerts** - Budget threshold alerts
   - Triggered when budgets exceed thresholds
   - Acknowledgement tracking
   - Severity levels (warning, critical)

5. **mv_daily_cost_summary** - Materialized view
   - Pre-aggregated daily summaries for fast reporting
   - Refreshed periodically

## Cloud Collectors

The `collectors/` package provides cloud-specific cost data collection:

### AWS (`aws.go`)
- AWS Cost Explorer integration
- CloudWatch metrics analysis
- Trusted Advisor recommendations
- Compute Optimizer data

### Azure (`azure.go`)
- Azure Cost Management API
- Azure Advisor integration
- Reserved capacity recommendations
- Storage tier optimization

### GCP (`gcp.go`)
- Cloud Billing API
- BigQuery export analysis
- Recommender API integration
- Committed use discount suggestions

**Note:** Current implementation uses mock data for demonstration. Production deployment requires:
1. Cloud SDK configuration
2. API credentials
3. IAM permissions for cost/billing APIs

## Testing

Run tests:
```bash
go test ./pkg/finops/...
```

Run with coverage:
```bash
go test -cover ./pkg/finops/...
```

## Migration

Apply database schema:
```bash
make migrate-up
```

This runs migration `000014_add_finops_tables.up.sql`.

Rollback:
```bash
make migrate-down
```

## Security

- Row-Level Security (RLS) enabled on all tables
- Organization isolation enforced at database level
- Budget creation/modification requires authentication
- All queries scoped to current organization

## Performance Considerations

1. **Indexing**: Optimized indexes on org_id, recorded_at, cloud, service
2. **Materialized Views**: Pre-aggregated daily summaries for faster queries
3. **Pagination**: Limit result sets to prevent memory issues
4. **Caching**: Consider Redis caching for frequently accessed summaries

## Future Enhancements

1. **Real-time Cost Tracking**: Stream cost data from cloud providers
2. **ML-based Forecasting**: Predict future costs using historical patterns
3. **Anomaly Detection**: Alert on unusual cost spikes
4. **Cost Allocation**: Chargeback/showback by team/project
5. **Right-sizing Automation**: Auto-apply approved recommendations
6. **Multi-currency Support**: Handle costs in different currencies
7. **Custom Dashboards**: User-defined cost visualization
8. **Webhook Notifications**: Real-time alerts to Slack/Teams/email

## Contributing

When adding new features:
1. Update types in `types.go`
2. Implement service methods in `finops.go`
3. Add API handlers in `services/api/internal/handlers/finops.go`
4. Write tests in `finops_test.go`
5. Update migration if schema changes
6. Document in this README

## References

- [AWS Cost Explorer API](https://docs.aws.amazon.com/aws-cost-management/latest/APIReference/API_Operations_AWS_Cost_Explorer_Service.html)
- [Azure Cost Management API](https://docs.microsoft.com/en-us/rest/api/cost-management/)
- [GCP Cloud Billing API](https://cloud.google.com/billing/docs/apis)
- [FinOps Foundation](https://www.finops.org/)
