package rbac

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

// ContextKey is the type for context keys.
type ContextKey string

const (
	// ContextKeyUserID is the context key for the user ID.
	ContextKeyUserID ContextKey = "user_id"
	// ContextKeyOrgID is the context key for the organization ID.
	ContextKeyOrgID ContextKey = "org_id"
	// ContextKeyPermissions is the context key for user permissions.
	ContextKeyPermissions ContextKey = "permissions"
)

// Middleware provides HTTP middleware for RBAC.
type Middleware struct {
	service *Service
}

// NewMiddleware creates a new RBAC middleware.
func NewMiddleware(service *Service) *Middleware {
	return &Middleware{service: service}
}

// RequirePermission returns middleware that requires a specific permission.
func (m *Middleware) RequirePermission(resourceType ResourceType, action Action) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserIDFromContext(r.Context())
			orgID := GetOrgIDFromContext(r.Context())

			if userID == "" || orgID == uuid.Nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			check, err := m.service.CheckPermission(r.Context(), userID, orgID, resourceType, nil, action)
			if err != nil {
				http.Error(w, "permission check failed", http.StatusInternalServerError)
				return
			}

			if !check.Allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "forbidden",
					"message": "insufficient permissions",
					"details": map[string]string{
						"resource_type": string(resourceType),
						"action":        string(action),
						"reason":        check.Reason,
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireResourcePermission returns middleware that requires permission on a specific resource.
// The resourceIDExtractor function extracts the resource ID from the request.
func (m *Middleware) RequireResourcePermission(resourceType ResourceType, action Action, resourceIDExtractor func(*http.Request) (uuid.UUID, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserIDFromContext(r.Context())
			orgID := GetOrgIDFromContext(r.Context())

			if userID == "" || orgID == uuid.Nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			resourceID, err := resourceIDExtractor(r)
			if err != nil {
				http.Error(w, "invalid resource ID", http.StatusBadRequest)
				return
			}

			check, err := m.service.CheckPermission(r.Context(), userID, orgID, resourceType, &resourceID, action)
			if err != nil {
				http.Error(w, "permission check failed", http.StatusInternalServerError)
				return
			}

			if !check.Allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "forbidden",
					"message": "insufficient permissions for this resource",
					"details": map[string]string{
						"resource_type": string(resourceType),
						"resource_id":   resourceID.String(),
						"action":        string(action),
						"reason":        check.Reason,
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission returns middleware that requires any of the specified permissions.
func (m *Middleware) RequireAnyPermission(permissions ...struct {
	ResourceType ResourceType
	Action       Action
}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserIDFromContext(r.Context())
			orgID := GetOrgIDFromContext(r.Context())

			if userID == "" || orgID == uuid.Nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			for _, perm := range permissions {
				check, err := m.service.CheckPermission(r.Context(), userID, orgID, perm.ResourceType, nil, perm.Action)
				if err != nil {
					continue
				}
				if check.Allowed {
					next.ServeHTTP(w, r)
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "forbidden",
				"message": "insufficient permissions",
			})
		})
	}
}

// RequireRole returns middleware that requires the user to have a specific role.
func (m *Middleware) RequireRole(roleName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserIDFromContext(r.Context())
			orgID := GetOrgIDFromContext(r.Context())

			if userID == "" || orgID == uuid.Nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			roles, err := m.service.GetUserRoles(r.Context(), userID, orgID)
			if err != nil {
				http.Error(w, "failed to get user roles", http.StatusInternalServerError)
				return
			}

			for _, role := range roles {
				if role.Name == roleName {
					next.ServeHTTP(w, r)
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "forbidden",
				"message": "required role not assigned",
				"details": map[string]string{
					"required_role": roleName,
				},
			})
		})
	}
}

// LoadUserPermissions loads all user permissions into context.
func (m *Middleware) LoadUserPermissions() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserIDFromContext(r.Context())
			orgID := GetOrgIDFromContext(r.Context())

			if userID == "" || orgID == uuid.Nil {
				next.ServeHTTP(w, r)
				return
			}

			permissions, err := m.service.GetUserPermissions(r.Context(), userID, orgID)
			if err != nil {
				// Don't fail - just continue without permissions in context
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyPermissions, permissions)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext returns the user ID from context.
func GetUserIDFromContext(ctx context.Context) string {
	if v := ctx.Value(ContextKeyUserID); v != nil {
		if userID, ok := v.(string); ok {
			return userID
		}
	}
	return ""
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

// GetPermissionsFromContext returns the user permissions from context.
func GetPermissionsFromContext(ctx context.Context) []UserPermission {
	if v := ctx.Value(ContextKeyPermissions); v != nil {
		if permissions, ok := v.([]UserPermission); ok {
			return permissions
		}
	}
	return nil
}

// SetUserContext sets user and org ID in context.
func SetUserContext(ctx context.Context, userID string, orgID uuid.UUID) context.Context {
	ctx = context.WithValue(ctx, ContextKeyUserID, userID)
	ctx = context.WithValue(ctx, ContextKeyOrgID, orgID)
	return ctx
}

// HasPermission checks if the current user has a specific permission.
// This is a helper for use in handlers after LoadUserPermissions middleware.
func HasPermission(ctx context.Context, resourceType ResourceType, action Action) bool {
	permissions := GetPermissionsFromContext(ctx)
	for _, p := range permissions {
		if p.ResourceType == resourceType && p.Action == action {
			return true
		}
	}
	return false
}
