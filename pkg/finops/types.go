// Package finops provides FinOps cost optimization and tracking functionality.
package finops

import (
	"time"

	"github.com/google/uuid"
)

// CostSummary represents aggregated cost data for an organization.
type CostSummary struct {
	OrgID       uuid.UUID              `json:"org_id"`
	TotalCost   float64                `json:"total_cost"`
	Currency    string                 `json:"currency"`
	Period      string                 `json:"period"` // daily, weekly, monthly
	StartDate   time.Time              `json:"start_date"`
	EndDate     time.Time              `json:"end_date"`
	ByCloud     map[string]float64     `json:"by_cloud"`     // aws: 1000, azure: 500
	ByService   map[string]float64     `json:"by_service"`   // ec2: 500, rds: 300
	BySite      map[string]float64     `json:"by_site"`      // us-east-1: 800, eu-west-1: 700
	ByResource  map[string]ResourceCost `json:"by_resource"`  // Top resources by cost
	TrendChange float64                `json:"trend_change"` // % change from previous period
}

// ResourceCost represents cost for a specific resource.
type ResourceCost struct {
	ResourceID   string    `json:"resource_id"`
	ResourceType string    `json:"resource_type"`
	ResourceName string    `json:"resource_name"`
	Platform     string    `json:"platform"`
	Cost         float64   `json:"cost"`
	Currency     string    `json:"currency"`
	UsageHours   float64   `json:"usage_hours,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
}

// CostRecord represents a single cost data point.
type CostRecord struct {
	ID           uuid.UUID         `json:"id" db:"id"`
	OrgID        uuid.UUID         `json:"org_id" db:"org_id"`
	ResourceID   string            `json:"resource_id" db:"resource_id"`
	ResourceType string            `json:"resource_type" db:"resource_type"`
	ResourceName string            `json:"resource_name,omitempty" db:"resource_name"`
	Cloud        string            `json:"cloud" db:"cloud"` // aws, azure, gcp
	Service      string            `json:"service,omitempty" db:"service"` // ec2, rds, s3
	Region       string            `json:"region,omitempty" db:"region"`
	Site         string            `json:"site,omitempty" db:"site"`
	Cost         float64           `json:"cost" db:"cost"`
	Currency     string            `json:"currency" db:"currency"`
	UsageHours   float64           `json:"usage_hours,omitempty" db:"usage_hours"`
	Tags         map[string]string `json:"tags,omitempty" db:"tags"`
	RecordedAt   time.Time         `json:"recorded_at" db:"recorded_at"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
}

// CostRecommendation represents a cost optimization recommendation.
type CostRecommendation struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	OrgID            uuid.UUID  `json:"org_id" db:"org_id"`
	Type             string     `json:"type" db:"type"` // rightsizing, reserved_instances, spot_instances, idle_resources, storage_optimization
	ResourceID       string     `json:"resource_id" db:"resource_id"`
	ResourceType     string     `json:"resource_type" db:"resource_type"`
	ResourceName     string     `json:"resource_name,omitempty" db:"resource_name"`
	Platform         string     `json:"platform" db:"platform"`
	CurrentCost      float64    `json:"current_cost" db:"current_cost"`
	PotentialSavings float64    `json:"potential_savings" db:"potential_savings"`
	Currency         string     `json:"currency" db:"currency"`
	Action           string     `json:"action" db:"action"` // Recommended action
	Details          string     `json:"details,omitempty" db:"details"` // Additional details in JSON format
	Priority         string     `json:"priority" db:"priority"` // high, medium, low
	Status           string     `json:"status" db:"status"` // pending, applied, dismissed
	DetectedAt       time.Time  `json:"detected_at" db:"detected_at"`
	AppliedAt        *time.Time `json:"applied_at,omitempty" db:"applied_at"`
	DismissedAt      *time.Time `json:"dismissed_at,omitempty" db:"dismissed_at"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// RecommendationType defines types of cost recommendations.
type RecommendationType string

const (
	RecommendationRightsizing         RecommendationType = "rightsizing"
	RecommendationReservedInstances   RecommendationType = "reserved_instances"
	RecommendationSpotInstances       RecommendationType = "spot_instances"
	RecommendationIdleResources       RecommendationType = "idle_resources"
	RecommendationStorageOptimization RecommendationType = "storage_optimization"
	RecommendationUnusedVolumes       RecommendationType = "unused_volumes"
	RecommendationOldSnapshots        RecommendationType = "old_snapshots"
)

// RecommendationPriority defines priority levels.
type RecommendationPriority string

const (
	PriorityHigh   RecommendationPriority = "high"
	PriorityMedium RecommendationPriority = "medium"
	PriorityLow    RecommendationPriority = "low"
)

// RecommendationStatus defines status values.
type RecommendationStatus string

const (
	StatusPending   RecommendationStatus = "pending"
	StatusApplied   RecommendationStatus = "applied"
	StatusDismissed RecommendationStatus = "dismissed"
)

// CostForecast represents predicted future costs.
type CostForecast struct {
	OrgID         uuid.UUID  `json:"org_id"`
	PredictedCost float64    `json:"predicted_cost"`
	Currency      string     `json:"currency"`
	Period        string     `json:"period"` // next_week, next_month, next_quarter
	StartDate     time.Time  `json:"start_date"`
	EndDate       time.Time  `json:"end_date"`
	Confidence    float64    `json:"confidence"` // 0.0 - 1.0
	Trend         string     `json:"trend"`      // increasing, decreasing, stable
	TrendPercent  float64    `json:"trend_percent"` // % change from current period
	Factors       []string   `json:"factors,omitempty"` // Factors influencing forecast
	ByCloud       map[string]float64 `json:"by_cloud,omitempty"`
	GeneratedAt   time.Time  `json:"generated_at"`
}

// CostBudget represents a cost budget for an organization.
type CostBudget struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	OrgID          uuid.UUID  `json:"org_id" db:"org_id"`
	Name           string     `json:"name" db:"name"`
	Description    string     `json:"description,omitempty" db:"description"`
	Amount         float64    `json:"amount" db:"amount"`
	Currency       string     `json:"currency" db:"currency"`
	Period         string     `json:"period" db:"period"` // daily, weekly, monthly, quarterly, yearly
	Scope          string     `json:"scope" db:"scope"` // organization, cloud, service, site
	ScopeValue     string     `json:"scope_value,omitempty" db:"scope_value"` // e.g., "aws", "ec2", "us-east-1"
	AlertThreshold float64    `json:"alert_threshold" db:"alert_threshold"` // % of budget to trigger alert (e.g., 80.0)
	StartDate      time.Time  `json:"start_date" db:"start_date"`
	EndDate        *time.Time `json:"end_date,omitempty" db:"end_date"`
	CurrentSpend   float64    `json:"current_spend" db:"current_spend"`
	Active         bool       `json:"active" db:"active"`
	CreatedBy      string     `json:"created_by" db:"created_by"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// BudgetPeriod defines budget time periods.
type BudgetPeriod string

const (
	PeriodDaily     BudgetPeriod = "daily"
	PeriodWeekly    BudgetPeriod = "weekly"
	PeriodMonthly   BudgetPeriod = "monthly"
	PeriodQuarterly BudgetPeriod = "quarterly"
	PeriodYearly    BudgetPeriod = "yearly"
)

// BudgetScope defines what a budget applies to.
type BudgetScope string

const (
	ScopeOrganization BudgetScope = "organization"
	ScopeCloud        BudgetScope = "cloud"
	ScopeService      BudgetScope = "service"
	ScopeSite         BudgetScope = "site"
)

// CostAlert represents a budget alert that was triggered.
type CostAlert struct {
	ID          uuid.UUID `json:"id" db:"id"`
	OrgID       uuid.UUID `json:"org_id" db:"org_id"`
	BudgetID    uuid.UUID `json:"budget_id" db:"budget_id"`
	BudgetName  string    `json:"budget_name,omitempty" db:"budget_name"`
	Amount      float64   `json:"amount" db:"amount"` // Current spend
	BudgetLimit float64   `json:"budget_limit" db:"budget_limit"`
	Percentage  float64   `json:"percentage" db:"percentage"` // % of budget spent
	Currency    string    `json:"currency" db:"currency"`
	Message     string    `json:"message" db:"message"`
	Severity    string    `json:"severity" db:"severity"` // warning, critical
	Acknowledged bool     `json:"acknowledged" db:"acknowledged"`
	TriggeredAt time.Time `json:"triggered_at" db:"triggered_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// CostTrend represents cost trend data over time.
type CostTrend struct {
	Date     time.Time `json:"date"`
	Cost     float64   `json:"cost"`
	Currency string    `json:"currency"`
	ByCloud  map[string]float64 `json:"by_cloud,omitempty"`
}

// CostBreakdown represents cost breakdown by a specific dimension.
type CostBreakdown struct {
	Dimension string             `json:"dimension"` // cloud, service, region, site, resource_type
	Items     []CostBreakdownItem `json:"items"`
	TotalCost float64            `json:"total_cost"`
	Currency  string             `json:"currency"`
	Period    string             `json:"period"`
	StartDate time.Time          `json:"start_date"`
	EndDate   time.Time          `json:"end_date"`
}

// CostBreakdownItem represents a single item in a cost breakdown.
type CostBreakdownItem struct {
	Name       string  `json:"name"`
	Cost       float64 `json:"cost"`
	Percentage float64 `json:"percentage"` // % of total cost
}

// TimeRange represents a time range for queries.
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// NewTimeRange creates a new time range.
func NewTimeRange(start, end time.Time) TimeRange {
	return TimeRange{Start: start, End: end}
}

// NewTimeRangeLast creates a time range for the last N days.
func NewTimeRangeLast(days int) TimeRange {
	end := time.Now()
	start := end.AddDate(0, 0, -days)
	return TimeRange{Start: start, End: end}
}

// NewTimeRangeThisMonth creates a time range for the current month.
func NewTimeRangeThisMonth() TimeRange {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 1, 0).Add(-time.Second)
	return TimeRange{Start: start, End: end}
}

// NewTimeRangeLastMonth creates a time range for the previous month.
func NewTimeRangeLastMonth() TimeRange {
	now := time.Now()
	start := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 1, 0).Add(-time.Second)
	return TimeRange{Start: start, End: end}
}
