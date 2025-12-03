package handlers

import (
	"net/http"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
)

// UserHandler handles user-related requests.
type UserHandler struct {
	log *logger.Logger
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(log *logger.Logger) *UserHandler {
	return &UserHandler{
		log: log.WithComponent("user-handler"),
	}
}

// GetCurrentUser returns the currently authenticated user.
func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)
	if user == nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	org := middleware.GetOrg(ctx)

	// Build response matching frontend UserInfo type
	response := map[string]interface{}{
		"id":         user.ID.String(),
		"externalId": user.ExternalID,
		"email":      user.Email,
		"name":       user.Name,
		"role":       string(user.Role),
	}

	// Add org info if available
	if org != nil {
		response["orgId"] = org.ID.String()
		response["orgName"] = org.Name
	}

	writeJSON(w, http.StatusOK, response)
}
