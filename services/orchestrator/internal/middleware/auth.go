// Package middleware provides HTTP middleware for the orchestrator service.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/auth"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

// ContextKey is a custom type for context keys.
type ContextKey string

const (
	// UserIDKey is the context key for the user ID.
	UserIDKey ContextKey = "user_id"
	// OrgIDKey is the context key for the organization ID.
	OrgIDKey ContextKey = "org_id"
	// EmailKey is the context key for the user email.
	EmailKey ContextKey = "email"
	// RoleKey is the context key for the user role.
	RoleKey ContextKey = "role"
)

// AuthConfig holds configuration for the auth middleware.
type AuthConfig struct {
	ClerkPublishableKey string
	DevMode             bool
}

// Auth returns a middleware that validates JWT tokens from Clerk.
func Auth(cfg AuthConfig, log *logger.Logger) func(next http.Handler) http.Handler {
	var verifier *auth.ClerkVerifier
	if !cfg.DevMode && cfg.ClerkPublishableKey != "" {
		verifier = auth.NewClerkVerifier(cfg.ClerkPublishableKey)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error": "missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, `{"error": "invalid authorization header format"}`, http.StatusUnauthorized)
				return
			}

			token := parts[1]
			if token == "" {
				http.Error(w, `{"error": "missing token"}`, http.StatusUnauthorized)
				return
			}

			ctx := r.Context()

			if cfg.DevMode || verifier == nil {
				// Development mode - use defaults
				log.Debug("orchestrator auth running in dev mode")
				ctx = context.WithValue(ctx, UserIDKey, "dev-user")
				ctx = context.WithValue(ctx, OrgIDKey, "00000000-0000-0000-0000-000000000001")
				ctx = context.WithValue(ctx, EmailKey, "dev@example.com")
				ctx = context.WithValue(ctx, RoleKey, "admin")
			} else {
				// Production mode - verify JWT
				claims, err := verifier.Verify(r.Context(), token)
				if err != nil {
					log.Warn("JWT verification failed", "error", err)
					http.Error(w, `{"error": "invalid or expired token"}`, http.StatusUnauthorized)
					return
				}

				ctx = context.WithValue(ctx, UserIDKey, claims.Subject)
				ctx = context.WithValue(ctx, EmailKey, claims.Email)
				ctx = context.WithValue(ctx, RoleKey, claims.OrgRole)

				if claims.OrgID != "" {
					ctx = context.WithValue(ctx, OrgIDKey, claims.OrgID)
				}

				log.Debug("authenticated request",
					"user_id", claims.Subject,
					"org_id", claims.OrgID,
				)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth is like Auth but doesn't require authentication.
func OptionalAuth(cfg AuthConfig, log *logger.Logger) func(next http.Handler) http.Handler {
	var verifier *auth.ClerkVerifier
	if !cfg.DevMode && cfg.ClerkPublishableKey != "" {
		verifier = auth.NewClerkVerifier(cfg.ClerkPublishableKey)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// No auth, use defaults in dev mode
				if cfg.DevMode {
					ctx = context.WithValue(ctx, UserIDKey, "dev-user")
					ctx = context.WithValue(ctx, OrgIDKey, "00000000-0000-0000-0000-000000000001")
				}
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
				if cfg.DevMode {
					ctx = context.WithValue(ctx, UserIDKey, "dev-user")
					ctx = context.WithValue(ctx, OrgIDKey, "00000000-0000-0000-0000-000000000001")
				}
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if cfg.DevMode || verifier == nil {
				ctx = context.WithValue(ctx, UserIDKey, "dev-user")
				ctx = context.WithValue(ctx, OrgIDKey, "00000000-0000-0000-0000-000000000001")
				ctx = context.WithValue(ctx, EmailKey, "dev@example.com")
				ctx = context.WithValue(ctx, RoleKey, "admin")
			} else {
				claims, err := verifier.Verify(r.Context(), parts[1])
				if err == nil {
					ctx = context.WithValue(ctx, UserIDKey, claims.Subject)
					ctx = context.WithValue(ctx, EmailKey, claims.Email)
					ctx = context.WithValue(ctx, RoleKey, claims.OrgRole)
					if claims.OrgID != "" {
						ctx = context.WithValue(ctx, OrgIDKey, claims.OrgID)
					}
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID returns the user ID from context.
func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}

// GetOrgID returns the organization ID from context.
func GetOrgID(ctx context.Context) string {
	if v, ok := ctx.Value(OrgIDKey).(string); ok {
		return v
	}
	return ""
}

// GetOrgUUID returns the organization ID as UUID from context.
func GetOrgUUID(ctx context.Context) uuid.UUID {
	orgID := GetOrgID(ctx)
	if orgID == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(orgID)
	if err != nil {
		return uuid.Nil
	}
	return id
}

// GetEmail returns the user email from context.
func GetEmail(ctx context.Context) string {
	if v, ok := ctx.Value(EmailKey).(string); ok {
		return v
	}
	return ""
}

// GetRole returns the user role from context.
func GetRole(ctx context.Context) string {
	if v, ok := ctx.Value(RoleKey).(string); ok {
		return v
	}
	return ""
}

// RequireRole returns a middleware that checks if the user has the required role.
func RequireRole(requiredRole string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetRole(r.Context())
			if role == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "authentication required"})
				return
			}

			// Admin has access to everything
			if role == "admin" {
				next.ServeHTTP(w, r)
				return
			}

			// Check specific role requirement
			if role != requiredRole {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{
					"error":    "insufficient permissions",
					"required": requiredRole,
					"current":  role,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission returns a middleware that checks if the user has the required permission.
func RequirePermission(perm models.Permission) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roleStr := GetRole(r.Context())
			if roleStr == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "authentication required"})
				return
			}

			role := models.Role(roleStr)
			if !role.IsValid() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "invalid role",
					"role":  roleStr,
				})
				return
			}

			if !role.HasPermission(perm) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{
					"error":      "insufficient permissions",
					"required":   string(perm),
					"role":       roleStr,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
