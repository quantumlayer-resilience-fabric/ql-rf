package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/finops"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
)

// FinOpsHandler handles FinOps cost optimization requests.
type FinOpsHandler struct {
	costSvc *finops.CostService
	log     *logger.Logger
}

// NewFinOpsHandler creates a new FinOpsHandler.
func NewFinOpsHandler(costSvc *finops.CostService, log *logger.Logger) *FinOpsHandler {
	return &FinOpsHandler{
		costSvc: costSvc,
		log:     log.WithComponent("finops-handler"),
	}
}

// GetSummary returns cost summary for the organization.
// GET /api/v1/finops/summary?period=7d|30d|90d|this_month|last_month
func (h *FinOpsHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	// Parse period parameter
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "30d" // Default to last 30 days
	}

	timeRange, err := parseTimeRange(period)
	if err != nil {
		http.Error(w, "invalid period parameter", http.StatusBadRequest)
		return
	}

	summary, err := h.costSvc.GetCostSummary(ctx, org.ID, timeRange)
	if err != nil {
		h.log.Error("failed to get cost summary", "error", err, "org_id", org.ID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

// GetBreakdown returns cost breakdown by dimension.
// GET /api/v1/finops/breakdown?dimension=cloud|service|region|site|resource_type&period=30d
func (h *FinOpsHandler) GetBreakdown(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	dimension := r.URL.Query().Get("dimension")
	if dimension == "" {
		dimension = "cloud" // Default dimension
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "30d"
	}

	timeRange, err := parseTimeRange(period)
	if err != nil {
		http.Error(w, "invalid period parameter", http.StatusBadRequest)
		return
	}

	breakdown, err := h.costSvc.GetCostBreakdown(ctx, org.ID, dimension, timeRange)
	if err != nil {
		h.log.Error("failed to get cost breakdown", "error", err, "org_id", org.ID, "dimension", dimension)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, breakdown)
}

// GetTrend returns cost trend over time.
// GET /api/v1/finops/trend?days=30
func (h *FinOpsHandler) GetTrend(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	days, err := strconv.Atoi(r.URL.Query().Get("days"))
	if err != nil || days < 1 {
		days = 30 // Default to 30 days
	}
	if days > 365 {
		days = 365 // Cap at 1 year
	}

	trend, err := h.costSvc.GetCostTrend(ctx, org.ID, days)
	if err != nil {
		h.log.Error("failed to get cost trend", "error", err, "org_id", org.ID, "days", days)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"trend": trend,
		"days":  days,
	})
}

// GetRecommendations returns cost optimization recommendations.
// GET /api/v1/finops/recommendations?type=all|rightsizing|reserved_instances|idle_resources
func (h *FinOpsHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	recommendations, err := h.costSvc.GetCostOptimizationRecommendations(ctx, org.ID)
	if err != nil {
		h.log.Error("failed to get recommendations", "error", err, "org_id", org.ID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Filter by type if specified
	recType := r.URL.Query().Get("type")
	if recType != "" && recType != "all" {
		filtered := make([]finops.CostRecommendation, 0)
		for _, rec := range recommendations {
			if rec.Type == recType {
				filtered = append(filtered, rec)
			}
		}
		recommendations = filtered
	}

	// Calculate total potential savings
	var totalSavings float64
	for _, rec := range recommendations {
		totalSavings += rec.PotentialSavings
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"recommendations":        recommendations,
		"total_recommendations":  len(recommendations),
		"total_potential_savings": totalSavings,
		"currency":               "USD",
	})
}

// CreateBudget creates a new cost budget.
// POST /api/v1/finops/budgets
func (h *FinOpsHandler) CreateBudget(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	user := middleware.GetUser(ctx)
	if user == nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	var req struct {
		Name           string  `json:"name"`
		Description    string  `json:"description,omitempty"`
		Amount         float64 `json:"amount"`
		Currency       string  `json:"currency,omitempty"`
		Period         string  `json:"period"`
		Scope          string  `json:"scope"`
		ScopeValue     string  `json:"scope_value,omitempty"`
		AlertThreshold float64 `json:"alert_threshold,omitempty"`
		StartDate      string  `json:"start_date"`
		EndDate        string  `json:"end_date,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" || req.Amount <= 0 || req.Period == "" || req.Scope == "" {
		http.Error(w, "name, amount, period, and scope are required", http.StatusBadRequest)
		return
	}

	// Parse dates
	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		// Try alternative format
		startDate, err = time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			http.Error(w, "invalid start_date format", http.StatusBadRequest)
			return
		}
	}

	var endDate *time.Time
	if req.EndDate != "" {
		ed, err := time.Parse(time.RFC3339, req.EndDate)
		if err != nil {
			ed, err = time.Parse("2006-01-02", req.EndDate)
			if err != nil {
				http.Error(w, "invalid end_date format", http.StatusBadRequest)
				return
			}
		}
		endDate = &ed
	}

	// Set defaults
	if req.Currency == "" {
		req.Currency = "USD"
	}
	if req.AlertThreshold == 0 {
		req.AlertThreshold = 80.0
	}

	budget := finops.CostBudget{
		OrgID:          org.ID,
		Name:           req.Name,
		Description:    req.Description,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Period:         req.Period,
		Scope:          req.Scope,
		ScopeValue:     req.ScopeValue,
		AlertThreshold: req.AlertThreshold,
		StartDate:      startDate,
		EndDate:        endDate,
		CreatedBy:      user.ExternalID,
	}

	created, err := h.costSvc.CreateBudget(ctx, budget)
	if err != nil {
		h.log.Error("failed to create budget", "error", err, "org_id", org.ID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.log.Info("budget created",
		"budget_id", created.ID,
		"org_id", org.ID,
		"created_by", user.ExternalID,
	)

	writeJSON(w, http.StatusCreated, created)
}

// ListBudgets returns all budgets for the organization.
// GET /api/v1/finops/budgets?active_only=true
func (h *FinOpsHandler) ListBudgets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	budgets, err := h.costSvc.ListBudgets(ctx, org.ID)
	if err != nil {
		h.log.Error("failed to list budgets", "error", err, "org_id", org.ID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Filter active only if requested
	activeOnly := r.URL.Query().Get("active_only") == "true"
	if activeOnly {
		filtered := make([]finops.CostBudget, 0)
		for _, budget := range budgets {
			if budget.Active {
				filtered = append(filtered, budget)
			}
		}
		budgets = filtered
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"budgets": budgets,
		"total":   len(budgets),
	})
}

// GetResourceCosts returns cost breakdown by resource.
// GET /api/v1/finops/resources?resource_type=ec2_instance&period=30d
func (h *FinOpsHandler) GetResourceCosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	resourceType := r.URL.Query().Get("resource_type")
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "30d"
	}

	timeRange, err := parseTimeRange(period)
	if err != nil {
		http.Error(w, "invalid period parameter", http.StatusBadRequest)
		return
	}

	resources, err := h.costSvc.GetCostByResource(ctx, org.ID, resourceType, timeRange)
	if err != nil {
		h.log.Error("failed to get resource costs", "error", err, "org_id", org.ID, "resource_type", resourceType)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Calculate total
	var totalCost float64
	for _, r := range resources {
		totalCost += r.Cost
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"resources":  resources,
		"total_cost": totalCost,
		"currency":   "USD",
	})
}

// Helper functions

func parseTimeRange(period string) (finops.TimeRange, error) {
	now := time.Now()

	switch period {
	case "7d", "week":
		return finops.NewTimeRangeLast(7), nil
	case "30d", "month":
		return finops.NewTimeRangeLast(30), nil
	case "90d", "quarter":
		return finops.NewTimeRangeLast(90), nil
	case "365d", "year":
		return finops.NewTimeRangeLast(365), nil
	case "this_month":
		return finops.NewTimeRangeThisMonth(), nil
	case "last_month":
		return finops.NewTimeRangeLastMonth(), nil
	default:
		// Try to parse as number of days
		if len(period) > 1 && period[len(period)-1] == 'd' {
			daysStr := period[:len(period)-1]
			days, err := strconv.Atoi(daysStr)
			if err == nil && days > 0 && days <= 365 {
				return finops.NewTimeRangeLast(days), nil
			}
		}
		// Default to 30 days
		return finops.NewTimeRangeLast(30), nil
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log error but can't change response at this point
		return
	}
}
