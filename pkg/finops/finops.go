// Package finops provides FinOps cost optimization and tracking functionality.
package finops

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CostService provides cost management and optimization services.
type CostService struct {
	db *pgxpool.Pool
}

// NewCostService creates a new CostService instance.
func NewCostService(db *pgxpool.Pool) *CostService {
	return &CostService{
		db: db,
	}
}

// GetCostSummary retrieves aggregated cost data for an organization.
func (s *CostService) GetCostSummary(ctx context.Context, orgID uuid.UUID, timeRange TimeRange) (*CostSummary, error) {
	query := `
		SELECT
			COALESCE(SUM(cost), 0) as total_cost,
			currency,
			cloud,
			service,
			site
		FROM cost_records
		WHERE org_id = $1
			AND recorded_at >= $2
			AND recorded_at <= $3
		GROUP BY currency, cloud, service, site
		ORDER BY total_cost DESC
	`

	rows, err := s.db.Query(ctx, query, orgID, timeRange.Start, timeRange.End)
	if err != nil {
		return nil, fmt.Errorf("query cost records: %w", err)
	}
	defer rows.Close()

	var totalCost float64
	currency := "USD" // Default
	byCloud := make(map[string]float64)
	byService := make(map[string]float64)
	bySite := make(map[string]float64)

	for rows.Next() {
		var cost float64
		var curr, cloud, service, site string
		if err := rows.Scan(&cost, &curr, &cloud, &service, &site); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		totalCost += cost
		currency = curr
		byCloud[cloud] += cost
		if service != "" {
			byService[service] += cost
		}
		if site != "" {
			bySite[site] += cost
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	// Get top resources by cost
	byResource, err := s.getTopResourceCosts(ctx, orgID, timeRange, 10)
	if err != nil {
		return nil, fmt.Errorf("get top resources: %w", err)
	}

	// Calculate trend change
	trendChange, err := s.calculateTrendChange(ctx, orgID, timeRange)
	if err != nil {
		// Log error but don't fail
		trendChange = 0
	}

	period := determinePeriod(timeRange)

	return &CostSummary{
		OrgID:       orgID,
		TotalCost:   totalCost,
		Currency:    currency,
		Period:      period,
		StartDate:   timeRange.Start,
		EndDate:     timeRange.End,
		ByCloud:     byCloud,
		ByService:   byService,
		BySite:      bySite,
		ByResource:  byResource,
		TrendChange: trendChange,
	}, nil
}

// GetCostByResource retrieves cost breakdown by resource type.
func (s *CostService) GetCostByResource(ctx context.Context, orgID uuid.UUID, resourceType string, timeRange TimeRange) ([]ResourceCost, error) {
	query := `
		SELECT
			resource_id,
			resource_type,
			resource_name,
			cloud,
			COALESCE(SUM(cost), 0) as total_cost,
			currency,
			COALESCE(SUM(usage_hours), 0) as total_usage_hours,
			tags
		FROM cost_records
		WHERE org_id = $1
			AND recorded_at >= $2
			AND recorded_at <= $3
	`

	args := []interface{}{orgID, timeRange.Start, timeRange.End}

	if resourceType != "" {
		query += " AND resource_type = $4"
		args = append(args, resourceType)
	}

	query += `
		GROUP BY resource_id, resource_type, resource_name, cloud, currency, tags
		ORDER BY total_cost DESC
		LIMIT 100
	`

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query cost by resource: %w", err)
	}
	defer rows.Close()

	var resources []ResourceCost
	for rows.Next() {
		var rc ResourceCost
		var tagsJSON []byte

		if err := rows.Scan(
			&rc.ResourceID,
			&rc.ResourceType,
			&rc.ResourceName,
			&rc.Platform,
			&rc.Cost,
			&rc.Currency,
			&rc.UsageHours,
			&tagsJSON,
		); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &rc.Tags); err != nil {
				// Log error but continue
				rc.Tags = nil
			}
		}

		resources = append(resources, rc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	return resources, nil
}

// GetCostTrend retrieves cost trend data over the specified number of days.
func (s *CostService) GetCostTrend(ctx context.Context, orgID uuid.UUID, days int) ([]CostTrend, error) {
	query := `
		SELECT
			DATE(recorded_at) as date,
			COALESCE(SUM(cost), 0) as total_cost,
			currency,
			cloud
		FROM cost_records
		WHERE org_id = $1
			AND recorded_at >= $2
		GROUP BY DATE(recorded_at), currency, cloud
		ORDER BY date ASC
	`

	startDate := time.Now().AddDate(0, 0, -days)
	rows, err := s.db.Query(ctx, query, orgID, startDate)
	if err != nil {
		return nil, fmt.Errorf("query cost trend: %w", err)
	}
	defer rows.Close()

	trendMap := make(map[time.Time]*CostTrend)

	for rows.Next() {
		var date time.Time
		var cost float64
		var currency, cloud string

		if err := rows.Scan(&date, &cost, &currency, &cloud); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		if trend, exists := trendMap[date]; exists {
			trend.Cost += cost
			trend.ByCloud[cloud] += cost
		} else {
			trendMap[date] = &CostTrend{
				Date:     date,
				Cost:     cost,
				Currency: currency,
				ByCloud: map[string]float64{
					cloud: cost,
				},
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	// Convert map to slice and sort by date
	trends := make([]CostTrend, 0, len(trendMap))
	for _, trend := range trendMap {
		trends = append(trends, *trend)
	}

	sort.Slice(trends, func(i, j int) bool {
		return trends[i].Date.Before(trends[j].Date)
	})

	return trends, nil
}

// GetCostOptimizationRecommendations retrieves cost optimization recommendations.
func (s *CostService) GetCostOptimizationRecommendations(ctx context.Context, orgID uuid.UUID) ([]CostRecommendation, error) {
	query := `
		SELECT
			id,
			org_id,
			type,
			resource_id,
			resource_type,
			resource_name,
			platform,
			current_cost,
			potential_savings,
			currency,
			action,
			details,
			priority,
			status,
			detected_at,
			applied_at,
			dismissed_at,
			created_at,
			updated_at
		FROM cost_recommendations
		WHERE org_id = $1
			AND status = $2
		ORDER BY potential_savings DESC, priority DESC
		LIMIT 100
	`

	rows, err := s.db.Query(ctx, query, orgID, StatusPending)
	if err != nil {
		return nil, fmt.Errorf("query recommendations: %w", err)
	}
	defer rows.Close()

	var recommendations []CostRecommendation
	for rows.Next() {
		var rec CostRecommendation
		if err := rows.Scan(
			&rec.ID,
			&rec.OrgID,
			&rec.Type,
			&rec.ResourceID,
			&rec.ResourceType,
			&rec.ResourceName,
			&rec.Platform,
			&rec.CurrentCost,
			&rec.PotentialSavings,
			&rec.Currency,
			&rec.Action,
			&rec.Details,
			&rec.Priority,
			&rec.Status,
			&rec.DetectedAt,
			&rec.AppliedAt,
			&rec.DismissedAt,
			&rec.CreatedAt,
			&rec.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		recommendations = append(recommendations, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	return recommendations, nil
}

// CreateBudget creates a new cost budget.
func (s *CostService) CreateBudget(ctx context.Context, budget CostBudget) (*CostBudget, error) {
	query := `
		INSERT INTO cost_budgets (
			id, org_id, name, description, amount, currency,
			period, scope, scope_value, alert_threshold,
			start_date, end_date, current_spend, active,
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17
		)
		RETURNING id, created_at, updated_at
	`

	budget.ID = uuid.New()
	budget.CurrentSpend = 0
	budget.Active = true
	now := time.Now()
	budget.CreatedAt = now
	budget.UpdatedAt = now

	err := s.db.QueryRow(ctx, query,
		budget.ID,
		budget.OrgID,
		budget.Name,
		budget.Description,
		budget.Amount,
		budget.Currency,
		budget.Period,
		budget.Scope,
		budget.ScopeValue,
		budget.AlertThreshold,
		budget.StartDate,
		budget.EndDate,
		budget.CurrentSpend,
		budget.Active,
		budget.CreatedBy,
		budget.CreatedAt,
		budget.UpdatedAt,
	).Scan(&budget.ID, &budget.CreatedAt, &budget.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("create budget: %w", err)
	}

	return &budget, nil
}

// ListBudgets retrieves all budgets for an organization.
func (s *CostService) ListBudgets(ctx context.Context, orgID uuid.UUID) ([]CostBudget, error) {
	query := `
		SELECT
			id, org_id, name, description, amount, currency,
			period, scope, scope_value, alert_threshold,
			start_date, end_date, current_spend, active,
			created_by, created_at, updated_at
		FROM cost_budgets
		WHERE org_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("query budgets: %w", err)
	}
	defer rows.Close()

	var budgets []CostBudget
	for rows.Next() {
		var budget CostBudget
		if err := rows.Scan(
			&budget.ID,
			&budget.OrgID,
			&budget.Name,
			&budget.Description,
			&budget.Amount,
			&budget.Currency,
			&budget.Period,
			&budget.Scope,
			&budget.ScopeValue,
			&budget.AlertThreshold,
			&budget.StartDate,
			&budget.EndDate,
			&budget.CurrentSpend,
			&budget.Active,
			&budget.CreatedBy,
			&budget.CreatedAt,
			&budget.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		budgets = append(budgets, budget)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	return budgets, nil
}

// UpdateBudgetSpend updates the current spend for a budget.
func (s *CostService) UpdateBudgetSpend(ctx context.Context, budgetID uuid.UUID, currentSpend float64) error {
	query := `
		UPDATE cost_budgets
		SET current_spend = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := s.db.Exec(ctx, query, currentSpend, time.Now(), budgetID)
	if err != nil {
		return fmt.Errorf("update budget spend: %w", err)
	}

	return nil
}

// CreateCostAlert creates a cost alert when a budget threshold is exceeded.
func (s *CostService) CreateCostAlert(ctx context.Context, alert CostAlert) error {
	query := `
		INSERT INTO cost_alerts (
			id, org_id, budget_id, budget_name, amount,
			budget_limit, percentage, currency, message,
			severity, acknowledged, triggered_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
	`

	alert.ID = uuid.New()
	alert.Acknowledged = false
	alert.TriggeredAt = time.Now()
	alert.CreatedAt = time.Now()

	_, err := s.db.Exec(ctx, query,
		alert.ID,
		alert.OrgID,
		alert.BudgetID,
		alert.BudgetName,
		alert.Amount,
		alert.BudgetLimit,
		alert.Percentage,
		alert.Currency,
		alert.Message,
		alert.Severity,
		alert.Acknowledged,
		alert.TriggeredAt,
		alert.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("create cost alert: %w", err)
	}

	return nil
}

// RecordCost records a cost entry.
func (s *CostService) RecordCost(ctx context.Context, record CostRecord) error {
	query := `
		INSERT INTO cost_records (
			id, org_id, resource_id, resource_type, resource_name,
			cloud, service, region, site, cost, currency,
			usage_hours, tags, recorded_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
		ON CONFLICT (org_id, resource_id, recorded_at)
		DO UPDATE SET
			cost = EXCLUDED.cost,
			usage_hours = EXCLUDED.usage_hours,
			tags = EXCLUDED.tags
	`

	record.ID = uuid.New()
	record.CreatedAt = time.Now()

	var tagsJSON []byte
	var err error
	if record.Tags != nil {
		tagsJSON, err = json.Marshal(record.Tags)
		if err != nil {
			return fmt.Errorf("marshal tags: %w", err)
		}
	}

	_, err = s.db.Exec(ctx, query,
		record.ID,
		record.OrgID,
		record.ResourceID,
		record.ResourceType,
		record.ResourceName,
		record.Cloud,
		record.Service,
		record.Region,
		record.Site,
		record.Cost,
		record.Currency,
		record.UsageHours,
		tagsJSON,
		record.RecordedAt,
		record.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("record cost: %w", err)
	}

	return nil
}

// GetCostBreakdown retrieves cost breakdown by a specific dimension.
func (s *CostService) GetCostBreakdown(ctx context.Context, orgID uuid.UUID, dimension string, timeRange TimeRange) (*CostBreakdown, error) {
	var column string
	switch dimension {
	case "cloud":
		column = "cloud"
	case "service":
		column = "service"
	case "region":
		column = "region"
	case "site":
		column = "site"
	case "resource_type":
		column = "resource_type"
	default:
		return nil, fmt.Errorf("invalid dimension: %s", dimension)
	}

	query := fmt.Sprintf(`
		SELECT
			%s as name,
			COALESCE(SUM(cost), 0) as total_cost,
			currency
		FROM cost_records
		WHERE org_id = $1
			AND recorded_at >= $2
			AND recorded_at <= $3
			AND %s IS NOT NULL
			AND %s != ''
		GROUP BY %s, currency
		ORDER BY total_cost DESC
	`, column, column, column, column)

	rows, err := s.db.Query(ctx, query, orgID, timeRange.Start, timeRange.End)
	if err != nil {
		return nil, fmt.Errorf("query cost breakdown: %w", err)
	}
	defer rows.Close()

	var items []CostBreakdownItem
	var totalCost float64
	currency := "USD"

	for rows.Next() {
		var name string
		var cost float64
		var curr string

		if err := rows.Scan(&name, &cost, &curr); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		items = append(items, CostBreakdownItem{
			Name: name,
			Cost: cost,
		})
		totalCost += cost
		currency = curr
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	// Calculate percentages
	for i := range items {
		if totalCost > 0 {
			items[i].Percentage = (items[i].Cost / totalCost) * 100
		}
	}

	period := determinePeriod(timeRange)

	return &CostBreakdown{
		Dimension: dimension,
		Items:     items,
		TotalCost: totalCost,
		Currency:  currency,
		Period:    period,
		StartDate: timeRange.Start,
		EndDate:   timeRange.End,
	}, nil
}

// Helper functions

func (s *CostService) getTopResourceCosts(ctx context.Context, orgID uuid.UUID, timeRange TimeRange, limit int) (map[string]ResourceCost, error) {
	query := `
		SELECT
			resource_id,
			resource_type,
			resource_name,
			cloud,
			COALESCE(SUM(cost), 0) as total_cost,
			currency,
			COALESCE(SUM(usage_hours), 0) as total_usage_hours
		FROM cost_records
		WHERE org_id = $1
			AND recorded_at >= $2
			AND recorded_at <= $3
		GROUP BY resource_id, resource_type, resource_name, cloud, currency
		ORDER BY total_cost DESC
		LIMIT $4
	`

	rows, err := s.db.Query(ctx, query, orgID, timeRange.Start, timeRange.End, limit)
	if err != nil {
		return nil, fmt.Errorf("query top resources: %w", err)
	}
	defer rows.Close()

	byResource := make(map[string]ResourceCost)
	for rows.Next() {
		var rc ResourceCost
		if err := rows.Scan(
			&rc.ResourceID,
			&rc.ResourceType,
			&rc.ResourceName,
			&rc.Platform,
			&rc.Cost,
			&rc.Currency,
			&rc.UsageHours,
		); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		byResource[rc.ResourceID] = rc
	}

	return byResource, nil
}

func (s *CostService) calculateTrendChange(ctx context.Context, orgID uuid.UUID, timeRange TimeRange) (float64, error) {
	duration := timeRange.End.Sub(timeRange.Start)
	previousStart := timeRange.Start.Add(-duration)
	previousEnd := timeRange.Start

	currentQuery := `
		SELECT COALESCE(SUM(cost), 0)
		FROM cost_records
		WHERE org_id = $1 AND recorded_at >= $2 AND recorded_at <= $3
	`

	var currentCost, previousCost float64

	if err := s.db.QueryRow(ctx, currentQuery, orgID, timeRange.Start, timeRange.End).Scan(&currentCost); err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("query current cost: %w", err)
	}

	if err := s.db.QueryRow(ctx, currentQuery, orgID, previousStart, previousEnd).Scan(&previousCost); err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("query previous cost: %w", err)
	}

	if previousCost == 0 {
		return 0, nil
	}

	return ((currentCost - previousCost) / previousCost) * 100, nil
}

func determinePeriod(tr TimeRange) string {
	duration := tr.End.Sub(tr.Start)
	days := int(duration.Hours() / 24)

	switch {
	case days <= 1:
		return "daily"
	case days <= 7:
		return "weekly"
	case days <= 31:
		return "monthly"
	case days <= 92:
		return "quarterly"
	default:
		return "yearly"
	}
}
