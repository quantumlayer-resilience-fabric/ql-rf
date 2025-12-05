package handlers

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/multitenancy"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
)

// OrganizationHandler handles organization and multi-tenancy requests.
type OrganizationHandler struct {
	svc *multitenancy.Service
	log *logger.Logger
}

// NewOrganizationHandler creates a new OrganizationHandler.
func NewOrganizationHandler(svc *multitenancy.Service, log *logger.Logger) *OrganizationHandler {
	return &OrganizationHandler{
		svc: svc,
		log: log.WithComponent("organization-handler"),
	}
}

// GetQuota returns the quota configuration for the organization.
func (h *OrganizationHandler) GetQuota(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	quota, err := h.svc.GetQuota(ctx, org.ID)
	if err != nil {
		h.log.Error("failed to get quota", "error", err, "org_id", org.ID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Return default quota if not configured
	if quota == nil {
		quota = &multitenancy.OrganizationQuota{
			OrgID:                 org.ID,
			MaxAssets:             1000,
			MaxImages:             100,
			MaxSites:              50,
			MaxUsers:              100,
			MaxTeams:              20,
			MaxAITasksPerDay:      100,
			MaxAITokensPerMonth:   10000000,
			MaxStorageBytes:       107374182400, // 100GB
			APIRateLimitPerMinute: 1000,
			DREnabled:             true,
			ComplianceEnabled:     true,
			AdvancedAnalytics:     false,
			CustomIntegrations:    false,
		}
	}

	writeJSON(w, http.StatusOK, quota)
}

// GetUsage returns the current resource usage for the organization.
func (h *OrganizationHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	usage, err := h.svc.GetUsage(ctx, org.ID)
	if err != nil {
		h.log.Error("failed to get usage", "error", err, "org_id", org.ID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, usage)
}

// GetQuotaStatus returns detailed quota status with remaining allocations.
func (h *OrganizationHandler) GetQuotaStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	statuses, err := h.svc.GetQuotaStatus(ctx, org.ID)
	if err != nil {
		h.log.Error("failed to get quota status", "error", err, "org_id", org.ID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"statuses": statuses,
	})
}

// GetSubscription returns the current subscription for the organization.
func (h *OrganizationHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	subscription, err := h.svc.GetSubscription(ctx, org.ID)
	if err != nil {
		h.log.Error("failed to get subscription", "error", err, "org_id", org.ID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// If no subscription, return a default trial subscription
	if subscription == nil {
		// Get free plan
		plan, err := h.svc.GetPlan(ctx, "free")
		if err == nil && plan != nil {
			subscription = &multitenancy.Subscription{
				ID:     uuid.New(),
				OrgID:  org.ID,
				PlanID: plan.ID,
				Status: "trial",
			}
		}
	}

	writeJSON(w, http.StatusOK, subscription)
}

// ListPlans returns all available subscription plans.
func (h *OrganizationHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	plans, err := h.svc.ListPlans(ctx)
	if err != nil {
		h.log.Error("failed to list plans", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// If no plans in database, return default plans
	if len(plans) == 0 {
		plans = []multitenancy.SubscriptionPlan{
			{
				ID:                  uuid.New(),
				Name:                "free",
				DisplayName:         "Free",
				Description:         "Free tier with basic features",
				PlanType:            "free",
				DefaultMaxAssets:    100,
				DefaultMaxImages:    10,
				DefaultMaxSites:     5,
				DefaultMaxUsers:     5,
				DefaultMaxAITasks:   10,
				DefaultMaxAITokens:  100000,
				DefaultMaxStorage:   10737418240, // 10GB
				DRIncluded:          false,
				ComplianceIncluded:  false,
				AdvancedAnalytics:   false,
				CustomIntegrations:  false,
				IsActive:            true,
			},
			{
				ID:                  uuid.New(),
				Name:                "starter",
				DisplayName:         "Starter",
				Description:         "Starter plan for small teams",
				PlanType:            "starter",
				DefaultMaxAssets:    500,
				DefaultMaxImages:    50,
				DefaultMaxSites:     20,
				DefaultMaxUsers:     25,
				DefaultMaxAITasks:   50,
				DefaultMaxAITokens:  1000000,
				DefaultMaxStorage:   53687091200, // 50GB
				DRIncluded:          true,
				ComplianceIncluded:  true,
				AdvancedAnalytics:   false,
				CustomIntegrations:  false,
				MonthlyPriceUSD:     floatPtr(99.0),
				AnnualPriceUSD:      floatPtr(990.0),
				IsActive:            true,
			},
			{
				ID:                  uuid.New(),
				Name:                "professional",
				DisplayName:         "Professional",
				Description:         "Professional plan for growing organizations",
				PlanType:            "professional",
				DefaultMaxAssets:    2000,
				DefaultMaxImages:    200,
				DefaultMaxSites:     100,
				DefaultMaxUsers:     100,
				DefaultMaxAITasks:   200,
				DefaultMaxAITokens:  10000000,
				DefaultMaxStorage:   214748364800, // 200GB
				DRIncluded:          true,
				ComplianceIncluded:  true,
				AdvancedAnalytics:   true,
				CustomIntegrations:  false,
				MonthlyPriceUSD:     floatPtr(499.0),
				AnnualPriceUSD:      floatPtr(4990.0),
				IsActive:            true,
			},
			{
				ID:                  uuid.New(),
				Name:                "enterprise",
				DisplayName:         "Enterprise",
				Description:         "Enterprise plan with unlimited resources",
				PlanType:            "enterprise",
				DefaultMaxAssets:    -1, // unlimited
				DefaultMaxImages:    -1,
				DefaultMaxSites:     -1,
				DefaultMaxUsers:     -1,
				DefaultMaxAITasks:   -1,
				DefaultMaxAITokens:  -1,
				DefaultMaxStorage:   -1,
				DRIncluded:          true,
				ComplianceIncluded:  true,
				AdvancedAnalytics:   true,
				CustomIntegrations:  true,
				IsActive:            true,
			},
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"plans": plans,
	})
}

// Helper function to create float64 pointer
func floatPtr(f float64) *float64 {
	return &f
}
