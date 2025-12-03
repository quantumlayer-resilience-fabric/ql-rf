package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quantumlayerhq/ql-rf/pkg/auth"
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
	// ClaimsContextKey is the context key for the raw Clerk claims.
	ClaimsContextKey ContextKey = "claims"
)

// AuthConfig holds configuration for the auth middleware.
type AuthConfig struct {
	ClerkPublishableKey string
	ClerkSecretKey      string
	DevMode             bool // Skip JWT validation in dev mode
}

// Auth returns a middleware that validates JWT tokens from Clerk.
func Auth(cfg AuthConfig, log *logger.Logger) func(next http.Handler) http.Handler {
	var verifier *auth.ClerkVerifier
	if !cfg.DevMode && cfg.ClerkPublishableKey != "" {
		verifier = auth.NewClerkVerifier(cfg.ClerkPublishableKey)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error": "missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			// Check Bearer prefix
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

			var user *models.User

			if cfg.DevMode || verifier == nil {
				// Development mode - create mock user
				log.Debug("auth middleware running in dev mode")
				user = &models.User{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ExternalID: "dev-user",
					Email:      "dev@example.com",
					Name:       "Development User",
					Role:       models.RoleAdmin,
				}
			} else {
				// Production mode - verify JWT with Clerk
				claims, err := verifier.Verify(r.Context(), token)
				if err != nil {
					log.Warn("JWT verification failed", "error", err)
					http.Error(w, `{"error": "invalid or expired token"}`, http.StatusUnauthorized)
					return
				}

				// Extract user info from claims
				user = &models.User{
					ExternalID: claims.Subject, // Clerk user ID
					Email:      claims.Email,
					Name:       claims.Name,
					Role:       mapClerkRoleToRole(claims.OrgRole),
				}

				// If org_id is set, try to parse it
				if claims.OrgID != "" {
					if orgID, err := uuid.Parse(claims.OrgID); err == nil {
						user.OrgID = orgID
					}
				}

				// Store raw claims in context for later use
				r = r.WithContext(context.WithValue(r.Context(), ClaimsContextKey, claims))

				log.Debug("authenticated user",
					"user_id", claims.Subject,
					"email", claims.Email,
					"org_id", claims.OrgID,
					"org_role", claims.OrgRole,
				)
			}

			// Add user to context
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			ctx = context.WithValue(ctx, logger.UserIDKey, user.ExternalID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// mapClerkRoleToRole maps Clerk organization roles to internal roles.
func mapClerkRoleToRole(clerkRole string) models.Role {
	switch strings.ToLower(clerkRole) {
	case "org:admin", "admin":
		return models.RoleAdmin
	case "org:engineer", "engineer":
		return models.RoleEngineer
	case "org:operator", "operator":
		return models.RoleOperator
	default:
		return models.RoleViewer
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

// GetClaims retrieves the raw Clerk claims from the context.
func GetClaims(ctx context.Context) *auth.ClerkClaims {
	if claims, ok := ctx.Value(ClaimsContextKey).(*auth.ClerkClaims); ok {
		return claims
	}
	return nil
}

// RequireRole returns middleware that checks if the user has the required role.
func RequireRole(requiredRole string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r.Context())
			if user == nil {
				http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
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
				http.Error(w, `{"error": "forbidden"}`, http.StatusForbidden)
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
				http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
				return
			}

			if !user.Role.HasPermission(perm) {
				http.Error(w, `{"error": "forbidden"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// OptionalAuth is like Auth but doesn't require authentication.
// If a valid token is provided, user info is added to context.
// If no token or invalid token, request continues without user context.
func OptionalAuth(cfg AuthConfig, log *logger.Logger) func(next http.Handler) http.Handler {
	var verifier *auth.ClerkVerifier
	if !cfg.DevMode && cfg.ClerkPublishableKey != "" {
		verifier = auth.NewClerkVerifier(cfg.ClerkPublishableKey)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// No auth header, continue without user
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				next.ServeHTTP(w, r)
				return
			}

			token := parts[1]
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			var user *models.User

			if cfg.DevMode || verifier == nil {
				user = &models.User{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ExternalID: "dev-user",
					Email:      "dev@example.com",
					Name:       "Development User",
					Role:       models.RoleAdmin,
				}
			} else {
				claims, err := verifier.Verify(r.Context(), token)
				if err != nil {
					// Invalid token, continue without user
					next.ServeHTTP(w, r)
					return
				}

				user = &models.User{
					ExternalID: claims.Subject,
					Email:      claims.Email,
					Name:       claims.Name,
					Role:       mapClerkRoleToRole(claims.OrgRole),
				}

				if claims.OrgID != "" {
					if orgID, err := uuid.Parse(claims.OrgID); err == nil {
						user.OrgID = orgID
					}
				}

				r = r.WithContext(context.WithValue(r.Context(), ClaimsContextKey, claims))
			}

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			ctx = context.WithValue(ctx, logger.UserIDKey, user.ExternalID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TenantConfig holds configuration for the tenant middleware.
type TenantConfig struct {
	DB      *pgxpool.Pool
	DevMode bool
	Log     *logger.Logger
}

// Tenant returns middleware that resolves the organization from the authenticated user.
// It should be used after the Auth middleware.
func Tenant(cfg TenantConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			user := GetUser(ctx)

			if user == nil {
				// No authenticated user, skip org lookup
				next.ServeHTTP(w, r)
				return
			}

			var org *models.Organization

			// Try to get org from user's org_id
			if user.OrgID != uuid.Nil {
				org = lookupOrgByID(ctx, cfg.DB, user.OrgID)
			}

			// If no org found and we have claims, try org from claims
			if org == nil {
				if claims := GetClaims(ctx); claims != nil && claims.OrgID != "" {
					if orgID, err := uuid.Parse(claims.OrgID); err == nil {
						org = lookupOrgByID(ctx, cfg.DB, orgID)
					}
				}
			}

			// Development mode fallback - use first org for the user or create default
			if org == nil && cfg.DevMode {
				org = lookupOrgByUserExternalID(ctx, cfg.DB, user.ExternalID)

				// If still no org in dev mode, try to find default org
				if org == nil {
					org = lookupDefaultOrg(ctx, cfg.DB)
				}

				if org != nil && cfg.Log != nil {
					cfg.Log.Debug("dev mode: resolved org for user",
						"user_external_id", user.ExternalID,
						"org_id", org.ID,
						"org_name", org.Name,
					)
				}
			}

			// Add organization to context if found
			if org != nil {
				ctx = context.WithValue(ctx, OrgContextKey, org)
				ctx = context.WithValue(ctx, logger.OrgIDKey, org.ID.String())
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// lookupOrgByID fetches an organization by ID.
func lookupOrgByID(ctx context.Context, db *pgxpool.Pool, orgID uuid.UUID) *models.Organization {
	if db == nil {
		return nil
	}

	var org models.Organization
	err := db.QueryRow(ctx, `
		SELECT id, name, slug, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`, orgID).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)

	if err != nil {
		return nil
	}
	return &org
}

// lookupOrgByUserExternalID finds the org for a user by their external ID.
func lookupOrgByUserExternalID(ctx context.Context, db *pgxpool.Pool, externalID string) *models.Organization {
	if db == nil {
		return nil
	}

	var org models.Organization
	err := db.QueryRow(ctx, `
		SELECT o.id, o.name, o.slug, o.created_at, o.updated_at
		FROM organizations o
		JOIN users u ON u.org_id = o.id
		WHERE u.external_id = $1
		LIMIT 1
	`, externalID).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)

	if err != nil {
		return nil
	}
	return &org
}

// lookupDefaultOrg finds the first organization (for dev mode).
func lookupDefaultOrg(ctx context.Context, db *pgxpool.Pool) *models.Organization {
	if db == nil {
		return nil
	}

	var org models.Organization
	err := db.QueryRow(ctx, `
		SELECT id, name, slug, created_at, updated_at
		FROM organizations
		ORDER BY created_at ASC
		LIMIT 1
	`).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)

	if err != nil {
		return nil
	}
	return &org
}
