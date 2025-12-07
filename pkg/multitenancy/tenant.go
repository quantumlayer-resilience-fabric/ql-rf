// Package multitenancy provides organization isolation, quotas, and usage tracking.
package multitenancy

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// QuotaType represents a type of resource quota.
type QuotaType string

const (
	QuotaAssets    QuotaType = "assets"
	QuotaImages    QuotaType = "images"
	QuotaSites     QuotaType = "sites"
	QuotaUsers     QuotaType = "users"
	QuotaTeams     QuotaType = "teams"
	QuotaAITasks   QuotaType = "ai_tasks"
	QuotaAITokens  QuotaType = "ai_tokens"
	QuotaStorage   QuotaType = "storage"
	QuotaAPIRate   QuotaType = "api_requests"
)

// OrganizationQuota represents the quota limits for an organization.
type OrganizationQuota struct {
	ID                     uuid.UUID `json:"id" db:"id"`
	OrgID                  uuid.UUID `json:"org_id" db:"org_id"`
	MaxAssets              int       `json:"max_assets" db:"max_assets"`
	MaxImages              int       `json:"max_images" db:"max_images"`
	MaxSites               int       `json:"max_sites" db:"max_sites"`
	MaxUsers               int       `json:"max_users" db:"max_users"`
	MaxTeams               int       `json:"max_teams" db:"max_teams"`
	MaxAITasksPerDay       int       `json:"max_ai_tasks_per_day" db:"max_ai_tasks_per_day"`
	MaxAITokensPerMonth    int64     `json:"max_ai_tokens_per_month" db:"max_ai_tokens_per_month"`
	MaxConcurrentTasks     int       `json:"max_concurrent_tasks" db:"max_concurrent_tasks"`
	MaxStorageBytes        int64     `json:"max_storage_bytes" db:"max_storage_bytes"`
	MaxArtifactSizeBytes   int64     `json:"max_artifact_size_bytes" db:"max_artifact_size_bytes"`
	APIRateLimitPerMinute  int       `json:"api_rate_limit_per_minute" db:"api_rate_limit_per_minute"`
	APIRateLimitPerDay     int       `json:"api_rate_limit_per_day" db:"api_rate_limit_per_day"`
	DREnabled              bool      `json:"dr_enabled" db:"dr_enabled"`
	ComplianceEnabled      bool      `json:"compliance_enabled" db:"compliance_enabled"`
	AdvancedAnalytics      bool      `json:"advanced_analytics_enabled" db:"advanced_analytics_enabled"`
	CustomIntegrations     bool      `json:"custom_integrations_enabled" db:"custom_integrations_enabled"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}

// OrganizationUsage represents the current usage for an organization.
type OrganizationUsage struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	OrgID              uuid.UUID `json:"org_id" db:"org_id"`
	AssetCount         int       `json:"asset_count" db:"asset_count"`
	ImageCount         int       `json:"image_count" db:"image_count"`
	SiteCount          int       `json:"site_count" db:"site_count"`
	UserCount          int       `json:"user_count" db:"user_count"`
	TeamCount          int       `json:"team_count" db:"team_count"`
	StorageUsedBytes   int64     `json:"storage_used_bytes" db:"storage_used_bytes"`
	AITasksToday       int       `json:"ai_tasks_today" db:"ai_tasks_today"`
	AITokensThisMonth  int64     `json:"ai_tokens_this_month" db:"ai_tokens_this_month"`
	APIRequestsToday   int       `json:"api_requests_today" db:"api_requests_today"`
	APIRequestsMinute  int       `json:"api_requests_this_minute" db:"api_requests_this_minute"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// QuotaStatus represents the status of quota usage.
type QuotaStatus struct {
	ResourceType   QuotaType `json:"resource_type"`
	Limit          int64     `json:"limit"`
	Used           int64     `json:"used"`
	Remaining      int64     `json:"remaining"`
	PercentageUsed float64   `json:"percentage_used"`
	IsExceeded     bool      `json:"is_exceeded"`
}

// SubscriptionPlan represents a subscription plan.
type SubscriptionPlan struct {
	ID                     uuid.UUID       `json:"id" db:"id"`
	Name                   string          `json:"name" db:"name"`
	DisplayName            string          `json:"display_name" db:"display_name"`
	Description            string          `json:"description,omitempty" db:"description"`
	PlanType               string          `json:"plan_type" db:"plan_type"`
	DefaultMaxAssets       int             `json:"default_max_assets" db:"default_max_assets"`
	DefaultMaxImages       int             `json:"default_max_images" db:"default_max_images"`
	DefaultMaxSites        int             `json:"default_max_sites" db:"default_max_sites"`
	DefaultMaxUsers        int             `json:"default_max_users" db:"default_max_users"`
	DefaultMaxAITasks      int             `json:"default_max_ai_tasks_per_day" db:"default_max_ai_tasks_per_day"`
	DefaultMaxAITokens     int64           `json:"default_max_ai_tokens_per_month" db:"default_max_ai_tokens_per_month"`
	DefaultMaxStorage      int64           `json:"default_max_storage_bytes" db:"default_max_storage_bytes"`
	DefaultAPIRateLimit    int             `json:"default_api_rate_limit_per_minute" db:"default_api_rate_limit_per_minute"`
	DRIncluded             bool            `json:"dr_included" db:"dr_included"`
	ComplianceIncluded     bool            `json:"compliance_included" db:"compliance_included"`
	AdvancedAnalytics      bool            `json:"advanced_analytics_included" db:"advanced_analytics_included"`
	CustomIntegrations     bool            `json:"custom_integrations_included" db:"custom_integrations_included"`
	MonthlyPriceUSD        *float64        `json:"monthly_price_usd,omitempty" db:"monthly_price_usd"`
	AnnualPriceUSD         *float64        `json:"annual_price_usd,omitempty" db:"annual_price_usd"`
	IsActive               bool            `json:"is_active" db:"is_active"`
	CreatedAt              time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at" db:"updated_at"`
}

// Subscription represents an organization's subscription.
type Subscription struct {
	ID                     uuid.UUID  `json:"id" db:"id"`
	OrgID                  uuid.UUID  `json:"org_id" db:"org_id"`
	PlanID                 uuid.UUID  `json:"plan_id" db:"plan_id"`
	Status                 string     `json:"status" db:"status"`
	TrialEndsAt            *time.Time `json:"trial_ends_at,omitempty" db:"trial_ends_at"`
	CurrentPeriodStart     time.Time  `json:"current_period_start" db:"current_period_start"`
	CurrentPeriodEnd       time.Time  `json:"current_period_end" db:"current_period_end"`
	ExternalSubscriptionID *string    `json:"external_subscription_id,omitempty" db:"external_subscription_id"`
	ExternalCustomerID     *string    `json:"external_customer_id,omitempty" db:"external_customer_id"`
	CancelledAt            *time.Time `json:"cancelled_at,omitempty" db:"cancelled_at"`
	CancelReason           *string    `json:"cancel_reason,omitempty" db:"cancel_reason"`
	CreatedAt              time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at" db:"updated_at"`
}

// Service provides multi-tenancy functionality.
type Service struct {
	db *sql.DB
}

// NewService creates a new multi-tenancy service.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// SetTenantContext sets the tenant context for the current database session.
func (s *Service) SetTenantContext(ctx context.Context, orgID uuid.UUID, userID string) error {
	_, err := s.db.ExecContext(ctx, "SELECT set_tenant_context($1, $2)", orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}
	return nil
}

// ClearTenantContext clears the tenant context.
func (s *Service) ClearTenantContext(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "SELECT clear_tenant_context()")
	if err != nil {
		return fmt.Errorf("failed to clear tenant context: %w", err)
	}
	return nil
}

// CheckQuota checks if an organization is within quota for a resource type.
func (s *Service) CheckQuota(ctx context.Context, orgID uuid.UUID, resourceType QuotaType, increment int) (bool, error) {
	var allowed bool
	err := s.db.QueryRowContext(ctx,
		"SELECT check_quota($1, $2, $3)",
		orgID, string(resourceType), increment,
	).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("failed to check quota: %w", err)
	}
	return allowed, nil
}

// IncrementUsage increments the usage counter for a resource type.
func (s *Service) IncrementUsage(ctx context.Context, orgID uuid.UUID, resourceType QuotaType, increment int) error {
	_, err := s.db.ExecContext(ctx,
		"SELECT increment_usage($1, $2, $3)",
		orgID, string(resourceType), increment,
	)
	if err != nil {
		return fmt.Errorf("failed to increment usage: %w", err)
	}
	return nil
}

// DecrementUsage decrements the usage counter for a resource type.
func (s *Service) DecrementUsage(ctx context.Context, orgID uuid.UUID, resourceType QuotaType, decrement int) error {
	_, err := s.db.ExecContext(ctx,
		"SELECT decrement_usage($1, $2, $3)",
		orgID, string(resourceType), decrement,
	)
	if err != nil {
		return fmt.Errorf("failed to decrement usage: %w", err)
	}
	return nil
}

// CheckAPIRateLimit checks and enforces API rate limits.
func (s *Service) CheckAPIRateLimit(ctx context.Context, orgID uuid.UUID) (bool, error) {
	var allowed bool
	err := s.db.QueryRowContext(ctx,
		"SELECT check_api_rate_limit($1)",
		orgID,
	).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("failed to check API rate limit: %w", err)
	}
	return allowed, nil
}

// GetQuota returns the quota configuration for an organization.
func (s *Service) GetQuota(ctx context.Context, orgID uuid.UUID) (*OrganizationQuota, error) {
	var q OrganizationQuota
	err := s.db.QueryRowContext(ctx, `
		SELECT id, org_id, max_assets, max_images, max_sites, max_users, max_teams,
		       max_ai_tasks_per_day, max_ai_tokens_per_month, max_concurrent_tasks,
		       max_storage_bytes, max_artifact_size_bytes, api_rate_limit_per_minute,
		       api_rate_limit_per_day, dr_enabled, compliance_enabled,
		       advanced_analytics_enabled, custom_integrations_enabled,
		       created_at, updated_at
		FROM organization_quotas WHERE org_id = $1
	`, orgID).Scan(
		&q.ID, &q.OrgID, &q.MaxAssets, &q.MaxImages, &q.MaxSites, &q.MaxUsers, &q.MaxTeams,
		&q.MaxAITasksPerDay, &q.MaxAITokensPerMonth, &q.MaxConcurrentTasks,
		&q.MaxStorageBytes, &q.MaxArtifactSizeBytes, &q.APIRateLimitPerMinute,
		&q.APIRateLimitPerDay, &q.DREnabled, &q.ComplianceEnabled,
		&q.AdvancedAnalytics, &q.CustomIntegrations,
		&q.CreatedAt, &q.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get quota: %w", err)
	}
	return &q, nil
}

// GetUsage returns the current usage for an organization.
func (s *Service) GetUsage(ctx context.Context, orgID uuid.UUID) (*OrganizationUsage, error) {
	var u OrganizationUsage
	err := s.db.QueryRowContext(ctx, `
		SELECT id, org_id, asset_count, image_count, site_count, user_count, team_count,
		       storage_used_bytes, ai_tasks_today, ai_tokens_this_month,
		       api_requests_today, api_requests_this_minute, updated_at
		FROM organization_usage WHERE org_id = $1
	`, orgID).Scan(
		&u.ID, &u.OrgID, &u.AssetCount, &u.ImageCount, &u.SiteCount, &u.UserCount, &u.TeamCount,
		&u.StorageUsedBytes, &u.AITasksToday, &u.AITokensThisMonth,
		&u.APIRequestsToday, &u.APIRequestsMinute, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		// Return zeroed usage if no record exists
		return &OrganizationUsage{OrgID: orgID}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}
	return &u, nil
}

// GetQuotaStatus returns the quota status for all resource types.
func (s *Service) GetQuotaStatus(ctx context.Context, orgID uuid.UUID) ([]QuotaStatus, error) {
	quota, err := s.GetQuota(ctx, orgID)
	if err != nil {
		return nil, err
	}

	usage, err := s.GetUsage(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Default quotas if not set
	if quota == nil {
		quota = &OrganizationQuota{
			MaxAssets:           1000,
			MaxImages:           100,
			MaxSites:            50,
			MaxUsers:            100,
			MaxTeams:            20,
			MaxAITasksPerDay:    100,
			MaxAITokensPerMonth: 10000000,
			MaxStorageBytes:     107374182400,
		}
	}

	statuses := []QuotaStatus{
		s.calculateStatus(QuotaAssets, int64(quota.MaxAssets), int64(usage.AssetCount)),
		s.calculateStatus(QuotaImages, int64(quota.MaxImages), int64(usage.ImageCount)),
		s.calculateStatus(QuotaSites, int64(quota.MaxSites), int64(usage.SiteCount)),
		s.calculateStatus(QuotaUsers, int64(quota.MaxUsers), int64(usage.UserCount)),
		s.calculateStatus(QuotaTeams, int64(quota.MaxTeams), int64(usage.TeamCount)),
		s.calculateStatus(QuotaAITasks, int64(quota.MaxAITasksPerDay), int64(usage.AITasksToday)),
		s.calculateStatus(QuotaAITokens, quota.MaxAITokensPerMonth, usage.AITokensThisMonth),
		s.calculateStatus(QuotaStorage, quota.MaxStorageBytes, usage.StorageUsedBytes),
	}

	return statuses, nil
}

func (s *Service) calculateStatus(resourceType QuotaType, limit, used int64) QuotaStatus {
	remaining := limit - used
	if remaining < 0 {
		remaining = 0
	}

	percentageUsed := float64(0)
	if limit > 0 {
		percentageUsed = float64(used) / float64(limit) * 100
	}

	return QuotaStatus{
		ResourceType:   resourceType,
		Limit:          limit,
		Used:           used,
		Remaining:      remaining,
		PercentageUsed: percentageUsed,
		IsExceeded:     used >= limit,
	}
}

// CreateQuota creates quota configuration for an organization.
func (s *Service) CreateQuota(ctx context.Context, quota OrganizationQuota) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO organization_quotas (
			org_id, max_assets, max_images, max_sites, max_users, max_teams,
			max_ai_tasks_per_day, max_ai_tokens_per_month, max_concurrent_tasks,
			max_storage_bytes, max_artifact_size_bytes, api_rate_limit_per_minute,
			api_rate_limit_per_day, dr_enabled, compliance_enabled,
			advanced_analytics_enabled, custom_integrations_enabled
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (org_id) DO UPDATE SET
			max_assets = EXCLUDED.max_assets,
			max_images = EXCLUDED.max_images,
			max_sites = EXCLUDED.max_sites,
			max_users = EXCLUDED.max_users,
			max_teams = EXCLUDED.max_teams,
			max_ai_tasks_per_day = EXCLUDED.max_ai_tasks_per_day,
			max_ai_tokens_per_month = EXCLUDED.max_ai_tokens_per_month,
			max_concurrent_tasks = EXCLUDED.max_concurrent_tasks,
			max_storage_bytes = EXCLUDED.max_storage_bytes,
			max_artifact_size_bytes = EXCLUDED.max_artifact_size_bytes,
			api_rate_limit_per_minute = EXCLUDED.api_rate_limit_per_minute,
			api_rate_limit_per_day = EXCLUDED.api_rate_limit_per_day,
			dr_enabled = EXCLUDED.dr_enabled,
			compliance_enabled = EXCLUDED.compliance_enabled,
			advanced_analytics_enabled = EXCLUDED.advanced_analytics_enabled,
			custom_integrations_enabled = EXCLUDED.custom_integrations_enabled,
			updated_at = NOW()
	`, quota.OrgID, quota.MaxAssets, quota.MaxImages, quota.MaxSites, quota.MaxUsers, quota.MaxTeams,
		quota.MaxAITasksPerDay, quota.MaxAITokensPerMonth, quota.MaxConcurrentTasks,
		quota.MaxStorageBytes, quota.MaxArtifactSizeBytes, quota.APIRateLimitPerMinute,
		quota.APIRateLimitPerDay, quota.DREnabled, quota.ComplianceEnabled,
		quota.AdvancedAnalytics, quota.CustomIntegrations)
	if err != nil {
		return fmt.Errorf("failed to create quota: %w", err)
	}
	return nil
}

// GetSubscription returns the subscription for an organization.
func (s *Service) GetSubscription(ctx context.Context, orgID uuid.UUID) (*Subscription, error) {
	var sub Subscription
	err := s.db.QueryRowContext(ctx, `
		SELECT id, org_id, plan_id, status, trial_ends_at, current_period_start,
		       current_period_end, external_subscription_id, external_customer_id,
		       cancelled_at, cancel_reason, created_at, updated_at
		FROM organization_subscriptions WHERE org_id = $1
	`, orgID).Scan(
		&sub.ID, &sub.OrgID, &sub.PlanID, &sub.Status, &sub.TrialEndsAt,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.ExternalSubscriptionID,
		&sub.ExternalCustomerID, &sub.CancelledAt, &sub.CancelReason,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	return &sub, nil
}

// GetPlan returns a subscription plan by name.
func (s *Service) GetPlan(ctx context.Context, name string) (*SubscriptionPlan, error) {
	var p SubscriptionPlan
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, display_name, description, plan_type,
		       default_max_assets, default_max_images, default_max_sites, default_max_users,
		       default_max_ai_tasks_per_day, default_max_ai_tokens_per_month,
		       default_max_storage_bytes, default_api_rate_limit_per_minute,
		       dr_included, compliance_included, advanced_analytics_included,
		       custom_integrations_included, monthly_price_usd, annual_price_usd,
		       is_active, created_at, updated_at
		FROM subscription_plans WHERE name = $1
	`, name).Scan(
		&p.ID, &p.Name, &p.DisplayName, &p.Description, &p.PlanType,
		&p.DefaultMaxAssets, &p.DefaultMaxImages, &p.DefaultMaxSites, &p.DefaultMaxUsers,
		&p.DefaultMaxAITasks, &p.DefaultMaxAITokens,
		&p.DefaultMaxStorage, &p.DefaultAPIRateLimit,
		&p.DRIncluded, &p.ComplianceIncluded, &p.AdvancedAnalytics,
		&p.CustomIntegrations, &p.MonthlyPriceUSD, &p.AnnualPriceUSD,
		&p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	return &p, nil
}

// ListPlans returns all active subscription plans.
func (s *Service) ListPlans(ctx context.Context) ([]SubscriptionPlan, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, display_name, description, plan_type,
		       default_max_assets, default_max_images, default_max_sites, default_max_users,
		       default_max_ai_tasks_per_day, default_max_ai_tokens_per_month,
		       default_max_storage_bytes, default_api_rate_limit_per_minute,
		       dr_included, compliance_included, advanced_analytics_included,
		       custom_integrations_included, monthly_price_usd, annual_price_usd,
		       is_active, created_at, updated_at
		FROM subscription_plans WHERE is_active = TRUE
		ORDER BY monthly_price_usd NULLS LAST
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}
	defer rows.Close()

	var plans []SubscriptionPlan
	for rows.Next() {
		var p SubscriptionPlan
		err := rows.Scan(
			&p.ID, &p.Name, &p.DisplayName, &p.Description, &p.PlanType,
			&p.DefaultMaxAssets, &p.DefaultMaxImages, &p.DefaultMaxSites, &p.DefaultMaxUsers,
			&p.DefaultMaxAITasks, &p.DefaultMaxAITokens,
			&p.DefaultMaxStorage, &p.DefaultAPIRateLimit,
			&p.DRIncluded, &p.ComplianceIncluded, &p.AdvancedAnalytics,
			&p.CustomIntegrations, &p.MonthlyPriceUSD, &p.AnnualPriceUSD,
			&p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plan: %w", err)
		}
		plans = append(plans, p)
	}

	return plans, rows.Err()
}

// CreateSubscription creates a subscription for an organization.
func (s *Service) CreateSubscription(ctx context.Context, orgID, planID uuid.UUID, trialDays int) error {
	now := time.Now()
	periodEnd := now.AddDate(0, 1, 0) // 1 month from now

	var trialEndsAt *time.Time
	if trialDays > 0 {
		trialEnd := now.AddDate(0, 0, trialDays)
		trialEndsAt = &trialEnd
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO organization_subscriptions (
			org_id, plan_id, status, trial_ends_at, current_period_start, current_period_end
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (org_id) DO UPDATE SET
			plan_id = EXCLUDED.plan_id,
			status = EXCLUDED.status,
			trial_ends_at = EXCLUDED.trial_ends_at,
			current_period_start = EXCLUDED.current_period_start,
			current_period_end = EXCLUDED.current_period_end,
			updated_at = NOW()
	`, orgID, planID, "active", trialEndsAt, now, periodEnd)
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	// Apply plan quotas to org
	plan, err := s.GetPlanByID(ctx, planID)
	if err != nil {
		return err
	}
	if plan != nil {
		quota := OrganizationQuota{
			OrgID:                 orgID,
			MaxAssets:             plan.DefaultMaxAssets,
			MaxImages:             plan.DefaultMaxImages,
			MaxSites:              plan.DefaultMaxSites,
			MaxUsers:              plan.DefaultMaxUsers,
			MaxAITasksPerDay:      plan.DefaultMaxAITasks,
			MaxAITokensPerMonth:   plan.DefaultMaxAITokens,
			MaxStorageBytes:       plan.DefaultMaxStorage,
			APIRateLimitPerMinute: plan.DefaultAPIRateLimit,
			DREnabled:             plan.DRIncluded,
			ComplianceEnabled:     plan.ComplianceIncluded,
			AdvancedAnalytics:     plan.AdvancedAnalytics,
			CustomIntegrations:    plan.CustomIntegrations,
		}
		return s.CreateQuota(ctx, quota)
	}

	return nil
}

// GetPlanByID returns a subscription plan by ID.
func (s *Service) GetPlanByID(ctx context.Context, planID uuid.UUID) (*SubscriptionPlan, error) {
	var p SubscriptionPlan
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, display_name, description, plan_type,
		       default_max_assets, default_max_images, default_max_sites, default_max_users,
		       default_max_ai_tasks_per_day, default_max_ai_tokens_per_month,
		       default_max_storage_bytes, default_api_rate_limit_per_minute,
		       dr_included, compliance_included, advanced_analytics_included,
		       custom_integrations_included, monthly_price_usd, annual_price_usd,
		       is_active, created_at, updated_at
		FROM subscription_plans WHERE id = $1
	`, planID).Scan(
		&p.ID, &p.Name, &p.DisplayName, &p.Description, &p.PlanType,
		&p.DefaultMaxAssets, &p.DefaultMaxImages, &p.DefaultMaxSites, &p.DefaultMaxUsers,
		&p.DefaultMaxAITasks, &p.DefaultMaxAITokens,
		&p.DefaultMaxStorage, &p.DefaultAPIRateLimit,
		&p.DRIncluded, &p.ComplianceIncluded, &p.AdvancedAnalytics,
		&p.CustomIntegrations, &p.MonthlyPriceUSD, &p.AnnualPriceUSD,
		&p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	return &p, nil
}

// IsFeatureEnabled checks if a feature is enabled for an organization.
func (s *Service) IsFeatureEnabled(ctx context.Context, orgID uuid.UUID, feature string) (bool, error) {
	quota, err := s.GetQuota(ctx, orgID)
	if err != nil {
		return false, err
	}
	if quota == nil {
		return false, nil
	}

	switch feature {
	case "dr":
		return quota.DREnabled, nil
	case "compliance":
		return quota.ComplianceEnabled, nil
	case "advanced_analytics":
		return quota.AdvancedAnalytics, nil
	case "custom_integrations":
		return quota.CustomIntegrations, nil
	default:
		return false, fmt.Errorf("unknown feature: %s", feature)
	}
}

// Organization represents a tenant organization.
type Organization struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CreateOrganizationParams contains parameters for creating an organization.
type CreateOrganizationParams struct {
	Name   string `json:"name"`
	Slug   string `json:"slug,omitempty"`
	PlanID string `json:"plan_id,omitempty"` // defaults to "free"
}

// CreateOrganizationResult contains the result of creating an organization.
type CreateOrganizationResult struct {
	Organization *Organization        `json:"organization"`
	Quota        *OrganizationQuota   `json:"quota"`
	Subscription *Subscription        `json:"subscription"`
}

// CreateOrganization creates a new organization with quota and subscription.
func (s *Service) CreateOrganization(ctx context.Context, params CreateOrganizationParams) (*CreateOrganizationResult, error) {
	// Generate slug from name if not provided
	slug := params.Slug
	if slug == "" {
		slug = generateSlug(params.Name)
	}

	// Start a transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create the organization
	var org Organization
	err = tx.QueryRowContext(ctx, `
		INSERT INTO organizations (name, slug)
		VALUES ($1, $2)
		RETURNING id, name, slug, created_at, updated_at
	`, params.Name, slug).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	// Get the plan (default to free)
	planName := params.PlanID
	if planName == "" {
		planName = "free"
	}
	plan, err := s.GetPlan(ctx, planName)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	// Use default values if plan not found in database
	var planID uuid.UUID
	var quota OrganizationQuota
	if plan != nil {
		planID = plan.ID
		quota = OrganizationQuota{
			OrgID:                 org.ID,
			MaxAssets:             plan.DefaultMaxAssets,
			MaxImages:             plan.DefaultMaxImages,
			MaxSites:              plan.DefaultMaxSites,
			MaxUsers:              plan.DefaultMaxUsers,
			MaxTeams:              10,
			MaxAITasksPerDay:      plan.DefaultMaxAITasks,
			MaxAITokensPerMonth:   plan.DefaultMaxAITokens,
			MaxConcurrentTasks:    5,
			MaxStorageBytes:       plan.DefaultMaxStorage,
			MaxArtifactSizeBytes:  1073741824, // 1GB
			APIRateLimitPerMinute: plan.DefaultAPIRateLimit,
			APIRateLimitPerDay:    plan.DefaultAPIRateLimit * 60 * 24,
			DREnabled:             plan.DRIncluded,
			ComplianceEnabled:     plan.ComplianceIncluded,
			AdvancedAnalytics:     plan.AdvancedAnalytics,
			CustomIntegrations:    plan.CustomIntegrations,
		}
	} else {
		// Default free plan values
		planID = uuid.New()
		quota = OrganizationQuota{
			OrgID:                 org.ID,
			MaxAssets:             100,
			MaxImages:             10,
			MaxSites:              5,
			MaxUsers:              5,
			MaxTeams:              2,
			MaxAITasksPerDay:      10,
			MaxAITokensPerMonth:   100000,
			MaxConcurrentTasks:    2,
			MaxStorageBytes:       10737418240, // 10GB
			MaxArtifactSizeBytes:  104857600,   // 100MB
			APIRateLimitPerMinute: 60,
			APIRateLimitPerDay:    1000,
			DREnabled:             false,
			ComplianceEnabled:     false,
			AdvancedAnalytics:     false,
			CustomIntegrations:    false,
		}
	}

	// Create organization quota
	_, err = tx.ExecContext(ctx, `
		INSERT INTO organization_quotas (
			org_id, max_assets, max_images, max_sites, max_users, max_teams,
			max_ai_tasks_per_day, max_ai_tokens_per_month, max_concurrent_tasks,
			max_storage_bytes, max_artifact_size_bytes, api_rate_limit_per_minute,
			api_rate_limit_per_day, dr_enabled, compliance_enabled,
			advanced_analytics_enabled, custom_integrations_enabled
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, quota.OrgID, quota.MaxAssets, quota.MaxImages, quota.MaxSites, quota.MaxUsers, quota.MaxTeams,
		quota.MaxAITasksPerDay, quota.MaxAITokensPerMonth, quota.MaxConcurrentTasks,
		quota.MaxStorageBytes, quota.MaxArtifactSizeBytes, quota.APIRateLimitPerMinute,
		quota.APIRateLimitPerDay, quota.DREnabled, quota.ComplianceEnabled,
		quota.AdvancedAnalytics, quota.CustomIntegrations)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization quota: %w", err)
	}

	// Create organization usage (initialized to zero)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO organization_usage (org_id)
		VALUES ($1)
	`, org.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization usage: %w", err)
	}

	// Create subscription
	now := time.Now()
	trialEnd := now.AddDate(0, 0, 14) // 14-day trial
	periodEnd := now.AddDate(0, 1, 0) // 1 month from now

	var subscription Subscription
	err = tx.QueryRowContext(ctx, `
		INSERT INTO organization_subscriptions (
			org_id, plan_id, status, trial_ends_at, current_period_start, current_period_end
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, org_id, plan_id, status, trial_ends_at, current_period_start,
		          current_period_end, external_subscription_id, external_customer_id,
		          cancelled_at, cancel_reason, created_at, updated_at
	`, org.ID, planID, "trial", trialEnd, now, periodEnd).Scan(
		&subscription.ID, &subscription.OrgID, &subscription.PlanID, &subscription.Status,
		&subscription.TrialEndsAt, &subscription.CurrentPeriodStart, &subscription.CurrentPeriodEnd,
		&subscription.ExternalSubscriptionID, &subscription.ExternalCustomerID,
		&subscription.CancelledAt, &subscription.CancelReason, &subscription.CreatedAt, &subscription.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &CreateOrganizationResult{
		Organization: &org,
		Quota:        &quota,
		Subscription: &subscription,
	}, nil
}

// GetOrganization retrieves an organization by ID.
func (s *Service) GetOrganization(ctx context.Context, orgID uuid.UUID) (*Organization, error) {
	var org Organization
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, created_at, updated_at
		FROM organizations WHERE id = $1
	`, orgID).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	return &org, nil
}

// GetOrganizationBySlug retrieves an organization by slug.
func (s *Service) GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	var org Organization
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, created_at, updated_at
		FROM organizations WHERE slug = $1
	`, slug).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	return &org, nil
}

// LinkUserToOrganization links a user to an organization with a role.
func (s *Service) LinkUserToOrganization(ctx context.Context, userID string, orgID uuid.UUID, role string) error {
	// Create placeholder email and name from user ID
	placeholderEmail := userID + "@placeholder.local"
	placeholderName := userID

	// First, ensure the user exists in the users table
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (external_id, org_id, email, name, role)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (external_id) DO UPDATE SET
			org_id = EXCLUDED.org_id,
			role = EXCLUDED.role,
			updated_at = NOW()
	`, userID, orgID, placeholderEmail, placeholderName, role)
	if err != nil {
		return fmt.Errorf("failed to link user to organization: %w", err)
	}
	return nil
}

// GetUserOrganization retrieves the organization for a user.
func (s *Service) GetUserOrganization(ctx context.Context, userID string) (*Organization, error) {
	var org Organization
	err := s.db.QueryRowContext(ctx, `
		SELECT o.id, o.name, o.slug, o.created_at, o.updated_at
		FROM organizations o
		JOIN users u ON u.org_id = o.id
		WHERE u.external_id = $1
	`, userID).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user organization: %w", err)
	}
	return &org, nil
}

// SeedDemoDataParams contains parameters for seeding demo data.
type SeedDemoDataParams struct {
	Platform string `json:"platform"` // aws, azure, gcp
}

// SeedDemoDataResult contains the result of seeding demo data.
type SeedDemoDataResult struct {
	SitesCreated  int `json:"sites_created"`
	AssetsCreated int `json:"assets_created"`
	ImagesCreated int `json:"images_created"`
}

// SeedDemoData seeds demo data for an organization.
func (s *Service) SeedDemoData(ctx context.Context, orgID uuid.UUID, params SeedDemoDataParams) (*SeedDemoDataResult, error) {
	platform := params.Platform
	if platform == "" {
		platform = "aws"
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	result := &SeedDemoDataResult{}

	// Create demo sites
	sites := getDemoSites(platform)
	for _, site := range sites {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO sites (org_id, name, platform, region, environment, metadata)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT DO NOTHING
		`, orgID, site.name, site.platform, site.region, "production", site.metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to create site %s: %w", site.name, err)
		}
		result.SitesCreated++
	}

	// Get site IDs for linking assets
	rows, err := tx.QueryContext(ctx, `SELECT id, name FROM sites WHERE org_id = $1`, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sites: %w", err)
	}
	siteMap := make(map[string]uuid.UUID)
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan site: %w", err)
		}
		siteMap[name] = id
	}
	rows.Close()

	// Create demo images first (assets reference images)
	// Images table uses: family, version, os_name, os_version, status
	// Unique constraint is on (org_id, family, version)
	images := getDemoImages(platform)
	imageMap := make(map[string]uuid.UUID)
	for _, img := range images {
		var imageID uuid.UUID
		err := tx.QueryRowContext(ctx, `
			INSERT INTO images (org_id, family, version, os_name, os_version, status)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (org_id, family, version) DO NOTHING
			RETURNING id
		`, orgID, img.family, img.version, img.osName, img.osVersion, img.status).Scan(&imageID)
		if err != nil {
			// Image might already exist, try to get its ID
			_ = tx.QueryRowContext(ctx, `SELECT id FROM images WHERE org_id = $1 AND family = $2 AND version = $3`, orgID, img.family, img.version).Scan(&imageID)
		}
		if imageID != uuid.Nil {
			imageMap[img.name] = imageID
			result.ImagesCreated++
		}
	}

	// Create demo assets
	assets := getDemoAssets(platform)
	for _, asset := range assets {
		siteID, ok := siteMap[asset.siteName]
		if !ok {
			continue
		}
		// Assets table uses: instance_id (required), name (optional), state (not status),
		// image_ref (not image_id), and tags for metadata
		_, err := tx.ExecContext(ctx, `
			INSERT INTO assets (org_id, site_id, instance_id, name, platform, state, image_ref, region, tags)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (org_id, platform, instance_id) DO NOTHING
		`, orgID, siteID, asset.instanceID, asset.name, asset.platform, asset.state, asset.imageRef, asset.region, asset.tags)
		if err != nil {
			return nil, fmt.Errorf("failed to create asset %s: %w", asset.name, err)
		}
		result.AssetsCreated++
	}

	// Update organization usage
	_, err = tx.ExecContext(ctx, `
		UPDATE organization_usage
		SET site_count = $1, asset_count = $2, image_count = $3, updated_at = NOW()
		WHERE org_id = $4
	`, result.SitesCreated, result.AssetsCreated, result.ImagesCreated, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to update usage: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

type demoSite struct {
	name     string
	platform string
	region   string
	metadata string
}

type demoImage struct {
	name      string // Internal reference name (for mapping assets to images)
	family    string // Image family (required for DB)
	version   string // Version (required for DB)
	osName    string // OS name (e.g., "Ubuntu", "Amazon Linux")
	osVersion string // OS version (e.g., "22.04", "2023")
	status    string // production, staging, draft
}

type demoAsset struct {
	instanceID string // Required: unique instance identifier
	name       string // Optional: display name
	siteName   string // For linking to site
	imageRef   string // Image reference (AMI ID, etc.)
	platform   string
	state      string // running, stopped, etc.
	region     string
	tags       string // JSON tags
}

func getDemoSites(platform string) []demoSite {
	switch platform {
	case "aws":
		return []demoSite{
			{name: "AWS US-East-1", platform: "aws", region: "us-east-1", metadata: `{"account_id": "123456789012"}`},
			{name: "AWS US-West-2", platform: "aws", region: "us-west-2", metadata: `{"account_id": "123456789012"}`},
			{name: "AWS EU-West-1", platform: "aws", region: "eu-west-1", metadata: `{"account_id": "123456789012"}`},
		}
	case "azure":
		return []demoSite{
			{name: "Azure East US", platform: "azure", region: "eastus", metadata: `{"subscription_id": "sub-12345"}`},
			{name: "Azure West Europe", platform: "azure", region: "westeurope", metadata: `{"subscription_id": "sub-12345"}`},
			{name: "Azure Southeast Asia", platform: "azure", region: "southeastasia", metadata: `{"subscription_id": "sub-12345"}`},
		}
	case "gcp":
		return []demoSite{
			{name: "GCP US-Central1", platform: "gcp", region: "us-central1", metadata: `{"project_id": "my-project-123"}`},
			{name: "GCP Europe-West1", platform: "gcp", region: "europe-west1", metadata: `{"project_id": "my-project-123"}`},
			{name: "GCP Asia-East1", platform: "gcp", region: "asia-east1", metadata: `{"project_id": "my-project-123"}`},
		}
	default:
		return getDemoSites("aws")
	}
}

func getDemoImages(platform string) []demoImage {
	switch platform {
	case "aws":
		return []demoImage{
			{name: "ami-prod-web-2024.12", family: "prod-web", version: "2024.12.01", osName: "Amazon Linux", osVersion: "2023", status: "production"},
			{name: "ami-prod-api-2024.12", family: "prod-api", version: "2024.12.01", osName: "Amazon Linux", osVersion: "2023", status: "production"},
			{name: "ami-prod-db-2024.11", family: "prod-db", version: "2024.11.15", osName: "Amazon Linux", osVersion: "2023", status: "deprecated"},
			{name: "ami-dev-base-2024.12", family: "dev-base", version: "2024.12.05", osName: "Ubuntu", osVersion: "22.04", status: "production"},
			{name: "ami-staging-web-2024.12", family: "staging-web", version: "2024.12.03", osName: "Amazon Linux", osVersion: "2023", status: "staging"},
		}
	case "azure":
		return []demoImage{
			{name: "img-prod-web-2024.12", family: "prod-web", version: "2024.12.01", osName: "Ubuntu", osVersion: "22.04", status: "production"},
			{name: "img-prod-api-2024.12", family: "prod-api", version: "2024.12.01", osName: "Ubuntu", osVersion: "22.04", status: "production"},
			{name: "img-prod-db-2024.11", family: "prod-db", version: "2024.11.15", osName: "Windows Server", osVersion: "2022", status: "deprecated"},
			{name: "img-dev-base-2024.12", family: "dev-base", version: "2024.12.05", osName: "Ubuntu", osVersion: "22.04", status: "production"},
		}
	case "gcp":
		return []demoImage{
			{name: "gce-prod-web-2024.12", family: "prod-web", version: "2024.12.01", osName: "Debian", osVersion: "12", status: "production"},
			{name: "gce-prod-api-2024.12", family: "prod-api", version: "2024.12.01", osName: "Debian", osVersion: "12", status: "production"},
			{name: "gce-prod-db-2024.11", family: "prod-db", version: "2024.11.15", osName: "Debian", osVersion: "11", status: "deprecated"},
			{name: "gce-dev-base-2024.12", family: "dev-base", version: "2024.12.05", osName: "Ubuntu", osVersion: "22.04", status: "production"},
		}
	default:
		return getDemoImages("aws")
	}
}

func getDemoAssets(platform string) []demoAsset {
	switch platform {
	case "aws":
		return []demoAsset{
			// US-East-1
			{instanceID: "i-demo0001", name: "web-prod-01", siteName: "AWS US-East-1", imageRef: "ami-prod-web-2024.12", platform: "aws", state: "running", region: "us-east-1", tags: `{"instance_type": "t3.medium", "environment": "production"}`},
			{instanceID: "i-demo0002", name: "web-prod-02", siteName: "AWS US-East-1", imageRef: "ami-prod-web-2024.12", platform: "aws", state: "running", region: "us-east-1", tags: `{"instance_type": "t3.medium", "environment": "production"}`},
			{instanceID: "i-demo0003", name: "api-prod-01", siteName: "AWS US-East-1", imageRef: "ami-prod-api-2024.12", platform: "aws", state: "running", region: "us-east-1", tags: `{"instance_type": "t3.large", "environment": "production"}`},
			{instanceID: "i-demo0004", name: "api-prod-02", siteName: "AWS US-East-1", imageRef: "ami-prod-api-2024.12", platform: "aws", state: "running", region: "us-east-1", tags: `{"instance_type": "t3.large", "environment": "production"}`},
			{instanceID: "i-demo0005", name: "db-prod-01", siteName: "AWS US-East-1", imageRef: "ami-prod-db-2024.11", platform: "aws", state: "running", region: "us-east-1", tags: `{"instance_type": "r5.xlarge", "environment": "production"}`},
			// US-West-2
			{instanceID: "i-demo0006", name: "web-dr-01", siteName: "AWS US-West-2", imageRef: "ami-prod-web-2024.12", platform: "aws", state: "running", region: "us-west-2", tags: `{"instance_type": "t3.medium", "environment": "dr"}`},
			{instanceID: "i-demo0007", name: "api-dr-01", siteName: "AWS US-West-2", imageRef: "ami-prod-api-2024.12", platform: "aws", state: "running", region: "us-west-2", tags: `{"instance_type": "t3.large", "environment": "dr"}`},
			{instanceID: "i-demo0008", name: "db-dr-01", siteName: "AWS US-West-2", imageRef: "ami-prod-db-2024.11", platform: "aws", state: "stopped", region: "us-west-2", tags: `{"instance_type": "r5.xlarge", "environment": "dr"}`},
			// EU-West-1
			{instanceID: "i-demo0009", name: "dev-base-01", siteName: "AWS EU-West-1", imageRef: "ami-dev-base-2024.12", platform: "aws", state: "running", region: "eu-west-1", tags: `{"instance_type": "t3.small", "environment": "development"}`},
			{instanceID: "i-demo0010", name: "staging-web-01", siteName: "AWS EU-West-1", imageRef: "ami-staging-web-2024.12", platform: "aws", state: "running", region: "eu-west-1", tags: `{"instance_type": "t3.medium", "environment": "staging"}`},
		}
	case "azure":
		return []demoAsset{
			// East US
			{instanceID: "vm-demo0001", name: "vm-web-prod-01", siteName: "Azure East US", imageRef: "img-prod-web-2024.12", platform: "azure", state: "running", region: "eastus", tags: `{"vm_size": "Standard_D2s_v3", "environment": "production"}`},
			{instanceID: "vm-demo0002", name: "vm-web-prod-02", siteName: "Azure East US", imageRef: "img-prod-web-2024.12", platform: "azure", state: "running", region: "eastus", tags: `{"vm_size": "Standard_D2s_v3", "environment": "production"}`},
			{instanceID: "vm-demo0003", name: "vm-api-prod-01", siteName: "Azure East US", imageRef: "img-prod-api-2024.12", platform: "azure", state: "running", region: "eastus", tags: `{"vm_size": "Standard_D4s_v3", "environment": "production"}`},
			{instanceID: "vm-demo0004", name: "vm-db-prod-01", siteName: "Azure East US", imageRef: "img-prod-db-2024.11", platform: "azure", state: "running", region: "eastus", tags: `{"vm_size": "Standard_E4s_v3", "environment": "production"}`},
			// West Europe
			{instanceID: "vm-demo0005", name: "vm-web-dr-01", siteName: "Azure West Europe", imageRef: "img-prod-web-2024.12", platform: "azure", state: "running", region: "westeurope", tags: `{"vm_size": "Standard_D2s_v3", "environment": "dr"}`},
			{instanceID: "vm-demo0006", name: "vm-api-dr-01", siteName: "Azure West Europe", imageRef: "img-prod-api-2024.12", platform: "azure", state: "stopped", region: "westeurope", tags: `{"vm_size": "Standard_D4s_v3", "environment": "dr"}`},
			// Southeast Asia
			{instanceID: "vm-demo0007", name: "vm-dev-01", siteName: "Azure Southeast Asia", imageRef: "img-dev-base-2024.12", platform: "azure", state: "running", region: "southeastasia", tags: `{"vm_size": "Standard_D2s_v3", "environment": "development"}`},
		}
	case "gcp":
		return []demoAsset{
			// US-Central1
			{instanceID: "gce-demo0001", name: "gce-web-prod-01", siteName: "GCP US-Central1", imageRef: "gce-prod-web-2024.12", platform: "gcp", state: "running", region: "us-central1", tags: `{"machine_type": "n2-standard-2", "environment": "production"}`},
			{instanceID: "gce-demo0002", name: "gce-web-prod-02", siteName: "GCP US-Central1", imageRef: "gce-prod-web-2024.12", platform: "gcp", state: "running", region: "us-central1", tags: `{"machine_type": "n2-standard-2", "environment": "production"}`},
			{instanceID: "gce-demo0003", name: "gce-api-prod-01", siteName: "GCP US-Central1", imageRef: "gce-prod-api-2024.12", platform: "gcp", state: "running", region: "us-central1", tags: `{"machine_type": "n2-standard-4", "environment": "production"}`},
			{instanceID: "gce-demo0004", name: "gce-db-prod-01", siteName: "GCP US-Central1", imageRef: "gce-prod-db-2024.11", platform: "gcp", state: "running", region: "us-central1", tags: `{"machine_type": "n2-highmem-4", "environment": "production"}`},
			// Europe-West1
			{instanceID: "gce-demo0005", name: "gce-web-dr-01", siteName: "GCP Europe-West1", imageRef: "gce-prod-web-2024.12", platform: "gcp", state: "stopped", region: "europe-west1", tags: `{"machine_type": "n2-standard-2", "environment": "dr"}`},
			// Asia-East1
			{instanceID: "gce-demo0006", name: "gce-dev-01", siteName: "GCP Asia-East1", imageRef: "gce-dev-base-2024.12", platform: "gcp", state: "running", region: "asia-east1", tags: `{"machine_type": "n2-standard-2", "environment": "development"}`},
		}
	default:
		return getDemoAssets("aws")
	}
}

// generateSlug generates a URL-friendly slug from a name.
func generateSlug(name string) string {
	slug := ""
	for _, r := range name {
		if r >= 'a' && r <= 'z' {
			slug += string(r)
		} else if r >= 'A' && r <= 'Z' {
			slug += string(r + 32) // Convert to lowercase
		} else if r >= '0' && r <= '9' {
			slug += string(r)
		} else if r == ' ' || r == '-' || r == '_' {
			if len(slug) > 0 && slug[len(slug)-1] != '-' {
				slug += "-"
			}
		}
	}
	// Trim trailing dashes
	for len(slug) > 0 && slug[len(slug)-1] == '-' {
		slug = slug[:len(slug)-1]
	}
	// Add a short unique suffix to avoid collisions
	suffix := uuid.New().String()[:8]
	if len(slug) > 0 {
		return slug + "-" + suffix
	}
	return suffix
}
