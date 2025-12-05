// Package billing provides LLM usage tracking and cost attribution.
package billing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// UsageTracker tracks LLM usage and costs.
type UsageTracker struct {
	db       *pgxpool.Pool
	log      *logger.Logger
	pricing  map[string]ModelPricing
	priceMu  sync.RWMutex
}

// NewUsageTracker creates a new LLM usage tracker.
func NewUsageTracker(db *pgxpool.Pool, log *logger.Logger) *UsageTracker {
	tracker := &UsageTracker{
		db:      db,
		log:     log.WithComponent("llm-billing"),
		pricing: make(map[string]ModelPricing),
	}

	// Load pricing on startup
	go tracker.loadPricing(context.Background())

	return tracker
}

// ModelPricing contains pricing for a specific model.
type ModelPricing struct {
	Provider                    string
	Model                       string
	InputPricePerMTokCents      int
	OutputPricePerMTokCents     int
	CacheCreationPricePerMTokCents *int
	CacheReadPricePerMTokCents    *int
	ContextWindow               int
	MaxOutputTokens             int
}

// UsageRecord represents a single LLM API call.
type UsageRecord struct {
	// Context
	OrgID     uuid.UUID
	UserID    string
	TaskID    *uuid.UUID
	AgentName string
	RequestID uuid.UUID

	// Model info
	Provider string
	Model    string

	// Token usage
	InputTokens        int
	OutputTokens       int
	CacheCreationTokens int
	CacheReadTokens    int

	// Request metadata
	OperationType string
	LatencyMS     int
	Status        string
	ErrorMessage  string
	PromptHash    string
}

// RecordUsage records an LLM API call and calculates cost.
func (t *UsageTracker) RecordUsage(ctx context.Context, record UsageRecord) error {
	// Calculate costs
	pricing := t.getPricing(record.Provider, record.Model)

	inputCostCents := (record.InputTokens * pricing.InputPricePerMTokCents) / 1_000_000
	outputCostCents := (record.OutputTokens * pricing.OutputPricePerMTokCents) / 1_000_000

	var cacheCreationCostCents, cacheReadCostCents int
	if pricing.CacheCreationPricePerMTokCents != nil {
		cacheCreationCostCents = (record.CacheCreationTokens * *pricing.CacheCreationPricePerMTokCents) / 1_000_000
	}
	if pricing.CacheReadPricePerMTokCents != nil {
		cacheReadCostCents = (record.CacheReadTokens * *pricing.CacheReadPricePerMTokCents) / 1_000_000
	}

	query := `
		INSERT INTO llm_usage (
			org_id, user_id, task_id, agent_name, request_id,
			provider, model,
			input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens,
			input_cost_cents, output_cost_cents, cache_creation_cost_cents, cache_read_cost_cents,
			operation_type, latency_ms, status, error_message, prompt_hash
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7,
			$8, $9, $10, $11,
			$12, $13, $14, $15,
			$16, $17, $18, $19, $20
		)
	`

	_, err := t.db.Exec(ctx, query,
		record.OrgID, record.UserID, record.TaskID, record.AgentName, record.RequestID,
		record.Provider, record.Model,
		record.InputTokens, record.OutputTokens, record.CacheCreationTokens, record.CacheReadTokens,
		inputCostCents, outputCostCents, cacheCreationCostCents, cacheReadCostCents,
		record.OperationType, record.LatencyMS, record.Status, record.ErrorMessage, record.PromptHash,
	)

	if err != nil {
		t.log.Error("failed to record LLM usage", "error", err)
		return fmt.Errorf("failed to record usage: %w", err)
	}

	t.log.Debug("recorded LLM usage",
		"org_id", record.OrgID,
		"model", record.Model,
		"input_tokens", record.InputTokens,
		"output_tokens", record.OutputTokens,
		"cost_cents", inputCostCents+outputCostCents,
	)

	return nil
}

// RecordUsageAsync records usage asynchronously.
func (t *UsageTracker) RecordUsageAsync(ctx context.Context, record UsageRecord) {
	go func() {
		recordCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := t.RecordUsage(recordCtx, record); err != nil {
			t.log.Error("async usage recording failed", "error", err)
		}
	}()
}

// CheckQuota checks if an organization has remaining quota.
func (t *UsageTracker) CheckQuota(ctx context.Context, orgID uuid.UUID) (*QuotaStatus, error) {
	var allowed bool
	var reason *string
	var usagePercent float64

	err := t.db.QueryRow(ctx, "SELECT * FROM check_llm_quota($1)", orgID).Scan(&allowed, &reason, &usagePercent)
	if err != nil {
		return nil, fmt.Errorf("failed to check quota: %w", err)
	}

	status := &QuotaStatus{
		Allowed:      allowed,
		UsagePercent: usagePercent,
	}
	if reason != nil {
		status.Reason = *reason
	}

	return status, nil
}

// QuotaStatus represents the current quota status for an organization.
type QuotaStatus struct {
	Allowed      bool    `json:"allowed"`
	Reason       string  `json:"reason,omitempty"`
	UsagePercent float64 `json:"usage_percent"`
}

// GetMonthlyUsage retrieves monthly usage for an organization.
func (t *UsageTracker) GetMonthlyUsage(ctx context.Context, orgID uuid.UUID, month time.Time) (*MonthlyUsage, error) {
	monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)

	query := `
		SELECT
			total_requests, total_input_tokens, total_output_tokens,
			total_tokens, total_cost_cents,
			usage_by_model, usage_by_agent
		FROM org_monthly_usage
		WHERE org_id = $1 AND month = $2
	`

	var usage MonthlyUsage
	var usageByModel, usageByAgent []byte

	err := t.db.QueryRow(ctx, query, orgID, monthStart).Scan(
		&usage.TotalRequests, &usage.TotalInputTokens, &usage.TotalOutputTokens,
		&usage.TotalTokens, &usage.TotalCostCents,
		&usageByModel, &usageByAgent,
	)
	if err != nil {
		// No usage for this month
		return &MonthlyUsage{
			OrgID: orgID,
			Month: monthStart,
		}, nil
	}

	usage.OrgID = orgID
	usage.Month = monthStart

	// Parse JSON breakdowns (would use json.Unmarshal in full implementation)

	return &usage, nil
}

// MonthlyUsage represents aggregated monthly usage.
type MonthlyUsage struct {
	OrgID             uuid.UUID         `json:"org_id"`
	Month             time.Time         `json:"month"`
	TotalRequests     int               `json:"total_requests"`
	TotalInputTokens  int64             `json:"total_input_tokens"`
	TotalOutputTokens int64             `json:"total_output_tokens"`
	TotalTokens       int64             `json:"total_tokens"`
	TotalCostCents    int               `json:"total_cost_cents"`
	TotalCostUSD      float64           `json:"total_cost_usd"`
	UsageByModel      map[string]ModelUsage `json:"usage_by_model,omitempty"`
	UsageByAgent      map[string]AgentUsage `json:"usage_by_agent,omitempty"`
}

// ModelUsage represents usage for a specific model.
type ModelUsage struct {
	Requests     int   `json:"requests"`
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	CostCents    int   `json:"cost_cents"`
}

// AgentUsage represents usage for a specific agent.
type AgentUsage struct {
	Requests  int   `json:"requests"`
	Tokens    int64 `json:"tokens"`
	CostCents int   `json:"cost_cents"`
}

// GetUsageByTask retrieves usage for a specific task.
func (t *UsageTracker) GetUsageByTask(ctx context.Context, taskID uuid.UUID) (*TaskUsage, error) {
	query := `
		SELECT
			COUNT(*) as requests,
			SUM(input_tokens) as input_tokens,
			SUM(output_tokens) as output_tokens,
			SUM(total_tokens) as total_tokens,
			SUM(total_cost_cents) as total_cost_cents,
			AVG(latency_ms) as avg_latency_ms
		FROM llm_usage
		WHERE task_id = $1
	`

	var usage TaskUsage
	err := t.db.QueryRow(ctx, query, taskID).Scan(
		&usage.TotalRequests,
		&usage.TotalInputTokens,
		&usage.TotalOutputTokens,
		&usage.TotalTokens,
		&usage.TotalCostCents,
		&usage.AvgLatencyMS,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get task usage: %w", err)
	}

	usage.TaskID = taskID
	usage.TotalCostUSD = float64(usage.TotalCostCents) / 100

	return &usage, nil
}

// TaskUsage represents LLM usage for a specific AI task.
type TaskUsage struct {
	TaskID            uuid.UUID `json:"task_id"`
	TotalRequests     int       `json:"total_requests"`
	TotalInputTokens  int64     `json:"total_input_tokens"`
	TotalOutputTokens int64     `json:"total_output_tokens"`
	TotalTokens       int64     `json:"total_tokens"`
	TotalCostCents    int       `json:"total_cost_cents"`
	TotalCostUSD      float64   `json:"total_cost_usd"`
	AvgLatencyMS      float64   `json:"avg_latency_ms"`
}

// SetQuota sets the quota for an organization.
func (t *UsageTracker) SetQuota(ctx context.Context, orgID uuid.UUID, quota QuotaSettings) error {
	query := `
		INSERT INTO org_llm_quotas (
			org_id, monthly_token_limit, monthly_cost_limit_cents,
			requests_per_minute, tokens_per_minute, alert_at_percent
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (org_id) DO UPDATE SET
			monthly_token_limit = EXCLUDED.monthly_token_limit,
			monthly_cost_limit_cents = EXCLUDED.monthly_cost_limit_cents,
			requests_per_minute = EXCLUDED.requests_per_minute,
			tokens_per_minute = EXCLUDED.tokens_per_minute,
			alert_at_percent = EXCLUDED.alert_at_percent,
			updated_at = NOW()
	`

	_, err := t.db.Exec(ctx, query,
		orgID, quota.MonthlyTokenLimit, quota.MonthlyCostLimitCents,
		quota.RequestsPerMinute, quota.TokensPerMinute, quota.AlertAtPercent,
	)
	if err != nil {
		return fmt.Errorf("failed to set quota: %w", err)
	}

	t.log.Info("set LLM quota",
		"org_id", orgID,
		"monthly_token_limit", quota.MonthlyTokenLimit,
		"monthly_cost_limit_cents", quota.MonthlyCostLimitCents,
	)

	return nil
}

// QuotaSettings contains quota configuration for an organization.
type QuotaSettings struct {
	MonthlyTokenLimit     *int64 `json:"monthly_token_limit,omitempty"`
	MonthlyCostLimitCents *int   `json:"monthly_cost_limit_cents,omitempty"`
	RequestsPerMinute     int    `json:"requests_per_minute"`
	TokensPerMinute       int    `json:"tokens_per_minute"`
	AlertAtPercent        int    `json:"alert_at_percent"`
}

// loadPricing loads model pricing from the database.
func (t *UsageTracker) loadPricing(ctx context.Context) {
	query := `
		SELECT provider, model, input_price_per_mtok_cents, output_price_per_mtok_cents,
		       cache_creation_price_per_mtok_cents, cache_read_price_per_mtok_cents,
		       context_window, max_output_tokens
		FROM llm_pricing
		WHERE effective_until IS NULL OR effective_until > NOW()
		ORDER BY effective_from DESC
	`

	rows, err := t.db.Query(ctx, query)
	if err != nil {
		t.log.Error("failed to load pricing", "error", err)
		return
	}
	defer rows.Close()

	t.priceMu.Lock()
	defer t.priceMu.Unlock()

	for rows.Next() {
		var p ModelPricing
		err := rows.Scan(
			&p.Provider, &p.Model, &p.InputPricePerMTokCents, &p.OutputPricePerMTokCents,
			&p.CacheCreationPricePerMTokCents, &p.CacheReadPricePerMTokCents,
			&p.ContextWindow, &p.MaxOutputTokens,
		)
		if err != nil {
			continue
		}

		key := p.Provider + ":" + p.Model
		if _, exists := t.pricing[key]; !exists {
			t.pricing[key] = p
		}
	}

	t.log.Info("loaded LLM pricing", "models", len(t.pricing))
}

// getPricing returns pricing for a model, with fallback defaults.
func (t *UsageTracker) getPricing(provider, model string) ModelPricing {
	t.priceMu.RLock()
	defer t.priceMu.RUnlock()

	key := provider + ":" + model
	if pricing, ok := t.pricing[key]; ok {
		return pricing
	}

	// Default pricing if not found
	t.log.Warn("using default pricing for unknown model", "provider", provider, "model", model)
	return ModelPricing{
		Provider:                provider,
		Model:                   model,
		InputPricePerMTokCents:  300,  // $3/MTok default
		OutputPricePerMTokCents: 1500, // $15/MTok default
	}
}

// HashPrompt creates a hash of a system prompt for caching analysis.
func HashPrompt(prompt string) string {
	hash := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(hash[:])
}

// GetCostReport generates a cost report for a time range.
func (t *UsageTracker) GetCostReport(ctx context.Context, orgID uuid.UUID, startDate, endDate time.Time) (*CostReport, error) {
	query := `
		SELECT
			DATE_TRUNC('day', timestamp) as day,
			provider,
			model,
			COUNT(*) as requests,
			SUM(input_tokens) as input_tokens,
			SUM(output_tokens) as output_tokens,
			SUM(total_cost_cents) as cost_cents
		FROM llm_usage
		WHERE org_id = $1 AND timestamp BETWEEN $2 AND $3
		GROUP BY DATE_TRUNC('day', timestamp), provider, model
		ORDER BY day DESC, cost_cents DESC
	`

	rows, err := t.db.Query(ctx, query, orgID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage: %w", err)
	}
	defer rows.Close()

	report := &CostReport{
		OrgID:     orgID,
		StartDate: startDate,
		EndDate:   endDate,
		DailyUsage: make([]DailyUsage, 0),
	}

	for rows.Next() {
		var daily DailyUsage
		err := rows.Scan(
			&daily.Date, &daily.Provider, &daily.Model,
			&daily.Requests, &daily.InputTokens, &daily.OutputTokens, &daily.CostCents,
		)
		if err != nil {
			continue
		}
		daily.CostUSD = float64(daily.CostCents) / 100
		report.DailyUsage = append(report.DailyUsage, daily)
		report.TotalCostCents += daily.CostCents
		report.TotalRequests += daily.Requests
		report.TotalTokens += daily.InputTokens + daily.OutputTokens
	}

	report.TotalCostUSD = float64(report.TotalCostCents) / 100

	return report, nil
}

// CostReport contains a cost report for a time period.
type CostReport struct {
	OrgID           uuid.UUID    `json:"org_id"`
	StartDate       time.Time    `json:"start_date"`
	EndDate         time.Time    `json:"end_date"`
	TotalRequests   int          `json:"total_requests"`
	TotalTokens     int64        `json:"total_tokens"`
	TotalCostCents  int          `json:"total_cost_cents"`
	TotalCostUSD    float64      `json:"total_cost_usd"`
	DailyUsage      []DailyUsage `json:"daily_usage"`
}

// DailyUsage represents usage for a single day and model.
type DailyUsage struct {
	Date         time.Time `json:"date"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	Requests     int       `json:"requests"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	CostCents    int       `json:"cost_cents"`
	CostUSD      float64   `json:"cost_usd"`
}
