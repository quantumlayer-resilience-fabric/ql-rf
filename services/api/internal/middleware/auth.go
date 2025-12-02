package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

// ContextKey is a custom type for context keys.
type ContextKey string

const (
	// UserContextKey is the context key for the authenticated user.
	UserContextKey ContextKey = "user"
	// OrgContextKey is the context key for the organization.
	OrgContextKey ContextKey = "org"
)

// Auth returns a middleware that validates JWT tokens from Clerk.
func Auth(clerkSecretKey string, log *logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			// Check Bearer prefix
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}

			token := parts[1]
			if token == "" {
				http.Error(w, "missing token", http.StatusUnauthorized)
				return
			}

			// TODO: Validate JWT with Clerk SDK
			// For now, we'll create a mock user for development
			// In production, this should verify the JWT with Clerk's JWKS
			user := &models.User{
				ExternalID: "dev-user",
				Email:      "dev@example.com",
				Name:       "Development User",
				Role:       models.RoleAdmin,
			}

			// Add user to context
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUser retrieves the authenticated user from the context.
func GetUser(ctx context.Context) *models.User {
	if user, ok := ctx.Value(UserContextKey).(*models.User); ok {
		return user
	}
	return nil
}

// GetOrg retrieves the organization from the context.
func GetOrg(ctx context.Context) *models.Organization {
	if org, ok := ctx.Value(OrgContextKey).(*models.Organization); ok {
		return org
	}
	return nil
}

// RequireRole returns middleware that checks if the user has the required role.
func RequireRole(requiredRole string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r.Context())
			if user == nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Check role hierarchy
			hasAccess := false
			switch models.Role(requiredRole) {
			case models.RoleViewer:
				hasAccess = true // All authenticated users have viewer access
			case models.RoleOperator:
				hasAccess = user.Role == models.RoleOperator ||
					user.Role == models.RoleEngineer ||
					user.Role == models.RoleAdmin
			case models.RoleEngineer:
				hasAccess = user.Role == models.RoleEngineer || user.Role == models.RoleAdmin
			case models.RoleAdmin:
				hasAccess = user.Role == models.RoleAdmin
			}

			if !hasAccess {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission returns middleware that checks if the user has the required permission.
func RequirePermission(perm models.Permission) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r.Context())
			if user == nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if !user.Role.HasPermission(perm) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
