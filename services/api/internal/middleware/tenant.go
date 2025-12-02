package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/database"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

// Tenant returns middleware that loads the organization for the authenticated user.
func Tenant(db *database.DB, log *logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r.Context())
			if user == nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// In development mode, create a default organization
			// In production, this should query the database based on user.OrgID
			org := &models.Organization{
				ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name: "Development Org",
				Slug: "dev",
			}

			// TODO: Query organization from database
			// Example:
			// var org models.Organization
			// err := db.QueryRow(r.Context(),
			//     "SELECT id, name, slug, created_at FROM organizations WHERE id = $1",
			//     user.OrgID,
			// ).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt)
			// if err != nil {
			//     log.Error("failed to load organization", "error", err, "org_id", user.OrgID)
			//     http.Error(w, "organization not found", http.StatusNotFound)
			//     return
			// }

			// Update user's org ID
			user.OrgID = org.ID

			// Add organization to context
			ctx := context.WithValue(r.Context(), OrgContextKey, org)
			ctx = context.WithValue(ctx, logger.OrgIDKey, org.ID.String())

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
