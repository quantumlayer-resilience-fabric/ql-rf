package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/multitenancy"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
)

// MultitenancyServiceInterface defines the methods required from the multitenancy service.
type MultitenancyServiceInterface interface {
	GetQuota(ctx context.Context, orgID uuid.UUID) (*multitenancy.OrganizationQuota, error)
	GetUsage(ctx context.Context, orgID uuid.UUID) (*multitenancy.OrganizationUsage, error)
	GetQuotaStatus(ctx context.Context, orgID uuid.UUID) ([]multitenancy.QuotaStatus, error)
	GetSubscription(ctx context.Context, orgID uuid.UUID) (*multitenancy.Subscription, error)
	GetPlan(ctx context.Context, name string) (*multitenancy.SubscriptionPlan, error)
	ListPlans(ctx context.Context) ([]multitenancy.SubscriptionPlan, error)
	CreateOrganization(ctx context.Context, params multitenancy.CreateOrganizationParams) (*multitenancy.CreateOrganizationResult, error)
	GetOrganization(ctx context.Context, orgID uuid.UUID) (*multitenancy.Organization, error)
	GetUserOrganization(ctx context.Context, userID string) (*multitenancy.Organization, error)
	LinkUserToOrganization(ctx context.Context, userID string, orgID uuid.UUID, role string) error
	SeedDemoData(ctx context.Context, orgID uuid.UUID, params multitenancy.SeedDemoDataParams) (*multitenancy.SeedDemoDataResult, error)
}

// OrganizationHandler handles organization and multi-tenancy requests.
type OrganizationHandler struct {
	svc MultitenancyServiceInterface
	log *logger.Logger
}

// NewOrganizationHandler creates a new OrganizationHandler.
func NewOrganizationHandler(svc MultitenancyServiceInterface, log *logger.Logger) *OrganizationHandler {
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

// CreateOrganizationRequest represents the request body for creating an organization.
type CreateOrganizationRequest struct {
	Name   string `json:"name"`
	Slug   string `json:"slug,omitempty"`
	PlanID string `json:"plan_id,omitempty"` // defaults to "free"
}

// CreateOrganization creates a new organization with quota and subscription.
func (h *OrganizationHandler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	// Create the organization
	result, err := h.svc.CreateOrganization(ctx, multitenancy.CreateOrganizationParams{
		Name:   req.Name,
		Slug:   req.Slug,
		PlanID: req.PlanID,
	})
	if err != nil {
		h.log.Error("failed to create organization", "error", err)
		http.Error(w, "failed to create organization", http.StatusInternalServerError)
		return
	}

	// Link the current user to the organization as owner
	user := middleware.GetUser(ctx)
	if user != nil && user.ExternalID != "" {
		if err := h.svc.LinkUserToOrganization(ctx, user.ExternalID, result.Organization.ID, "org_owner"); err != nil {
			h.log.Error("failed to link user to organization", "error", err, "user_id", user.ExternalID, "org_id", result.Organization.ID)
			// Don't fail the request, org was created successfully
		}
	}

	writeJSON(w, http.StatusCreated, result)
}

// GetCurrentOrganization returns the current organization context.
func (h *OrganizationHandler) GetCurrentOrganization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	// Get the full organization details
	fullOrg, err := h.svc.GetOrganization(ctx, org.ID)
	if err != nil {
		h.log.Error("failed to get organization", "error", err, "org_id", org.ID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, fullOrg)
}

// CheckUserOrganization checks if the current user has an organization.
func (h *OrganizationHandler) CheckUserOrganization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)
	if user == nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	org, err := h.svc.GetUserOrganization(ctx, user.ExternalID)
	if err != nil {
		h.log.Error("failed to get user organization", "error", err, "user_id", user.ExternalID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"has_organization": org != nil,
		"organization":     org,
	})
}

// SeedDemoDataRequest represents the request body for seeding demo data.
type SeedDemoDataRequest struct {
	Platform string `json:"platform"` // aws, azure, gcp
}

// SeedDemoData seeds demo data for the organization.
func (h *OrganizationHandler) SeedDemoData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	var req SeedDemoDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to AWS if no body provided
		req.Platform = "aws"
	}

	result, err := h.svc.SeedDemoData(ctx, org.ID, multitenancy.SeedDemoDataParams{
		Platform: req.Platform,
	})
	if err != nil {
		h.log.Error("failed to seed demo data", "error", err, "org_id", org.ID)
		http.Error(w, "failed to seed demo data", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
