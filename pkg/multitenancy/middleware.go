package multitenancy

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

// ContextKey is the type for context keys.
type ContextKey string

const (
	// ContextKeyOrgID is the context key for the organization ID.
	ContextKeyOrgID ContextKey = "tenant_org_id"
	// ContextKeyQuota is the context key for the organization quota.
	ContextKeyQuota ContextKey = "tenant_quota"
	// ContextKeyUsage is the context key for the organization usage.
	ContextKeyUsage ContextKey = "tenant_usage"
)

// Middleware provides HTTP middleware for multi-tenancy.
type Middleware struct {
	service *Service
}

// NewMiddleware creates a new multi-tenancy middleware.
func NewMiddleware(service *Service) *Middleware {
	return &Middleware{service: service}
}

// RateLimiter returns middleware that enforces API rate limits.
func (m *Middleware) RateLimiter() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgID := GetOrgIDFromContext(r.Context())
			if orgID == uuid.Nil {
				next.ServeHTTP(w, r)
				return
			}

			allowed, err := m.service.CheckAPIRateLimit(r.Context(), orgID)
			if err != nil {
				// Log error but allow request to proceed
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "rate_limit_exceeded",
					"message": "API rate limit exceeded. Please retry after some time.",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireQuota returns middleware that checks if org has quota for a resource.
func (m *Middleware) RequireQuota(resourceType QuotaType) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgID := GetOrgIDFromContext(r.Context())
			if orgID == uuid.Nil {
				next.ServeHTTP(w, r)
				return
			}

			allowed, err := m.service.CheckQuota(r.Context(), orgID, resourceType, 1)
			if err != nil {
				http.Error(w, "quota check failed", http.StatusInternalServerError)
				return
			}

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusPaymentRequired)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":         "quota_exceeded",
					"message":       "Resource quota exceeded. Please upgrade your plan.",
					"resource_type": string(resourceType),
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireFeature returns middleware that checks if a feature is enabled.
func (m *Middleware) RequireFeature(feature string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgID := GetOrgIDFromContext(r.Context())
			if orgID == uuid.Nil {
				next.ServeHTTP(w, r)
				return
			}

			enabled, err := m.service.IsFeatureEnabled(r.Context(), orgID, feature)
			if err != nil {
				http.Error(w, "feature check failed", http.StatusInternalServerError)
				return
			}

			if !enabled {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusPaymentRequired)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "feature_not_available",
					"message": "This feature is not available in your current plan. Please upgrade.",
					"feature": feature,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// LoadQuota loads quota information into context.
func (m *Middleware) LoadQuota() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgID := GetOrgIDFromContext(r.Context())
			if orgID == uuid.Nil {
				next.ServeHTTP(w, r)
				return
			}

			quota, err := m.service.GetQuota(r.Context(), orgID)
			if err == nil && quota != nil {
				ctx := context.WithValue(r.Context(), ContextKeyQuota, quota)
				r = r.WithContext(ctx)
			}

			usage, err := m.service.GetUsage(r.Context(), orgID)
			if err == nil && usage != nil {
				ctx := context.WithValue(r.Context(), ContextKeyUsage, usage)
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SetTenantContext sets the tenant context in the database session.
func (m *Middleware) SetTenantContext(getUserID func(context.Context) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgID := GetOrgIDFromContext(r.Context())
			if orgID == uuid.Nil {
				next.ServeHTTP(w, r)
				return
			}

			userID := ""
			if getUserID != nil {
				userID = getUserID(r.Context())
			}

			// Set tenant context in database session (for RLS)
			_ = m.service.SetTenantContext(r.Context(), orgID, userID)
			defer m.service.ClearTenantContext(r.Context())

			next.ServeHTTP(w, r)
		})
	}
}

// GetOrgIDFromContext returns the organization ID from context.
func GetOrgIDFromContext(ctx context.Context) uuid.UUID {
	if v := ctx.Value(ContextKeyOrgID); v != nil {
		if orgID, ok := v.(uuid.UUID); ok {
			return orgID
		}
	}
	return uuid.Nil
}

// GetQuotaFromContext returns the quota from context.
func GetQuotaFromContext(ctx context.Context) *OrganizationQuota {
	if v := ctx.Value(ContextKeyQuota); v != nil {
		if quota, ok := v.(*OrganizationQuota); ok {
			return quota
		}
	}
	return nil
}

// GetUsageFromContext returns the usage from context.
func GetUsageFromContext(ctx context.Context) *OrganizationUsage {
	if v := ctx.Value(ContextKeyUsage); v != nil {
		if usage, ok := v.(*OrganizationUsage); ok {
			return usage
		}
	}
	return nil
}

// SetOrgIDContext sets the organization ID in context.
func SetOrgIDContext(ctx context.Context, orgID uuid.UUID) context.Context {
	return context.WithValue(ctx, ContextKeyOrgID, orgID)
}

// QuotaExceededError represents a quota exceeded error.
type QuotaExceededError struct {
	ResourceType QuotaType
	Limit        int64
	Used         int64
}

func (e *QuotaExceededError) Error() string {
	return "quota exceeded for " + string(e.ResourceType)
}

// IsQuotaExceeded checks if an error is a quota exceeded error.
func IsQuotaExceeded(err error) bool {
	_, ok := err.(*QuotaExceededError)
	return ok
}

// FeatureNotEnabledError represents a feature not enabled error.
type FeatureNotEnabledError struct {
	Feature string
}

func (e *FeatureNotEnabledError) Error() string {
	return "feature not enabled: " + e.Feature
}

// IsFeatureNotEnabled checks if an error is a feature not enabled error.
func IsFeatureNotEnabled(err error) bool {
	_, ok := err.(*FeatureNotEnabledError)
	return ok
}
