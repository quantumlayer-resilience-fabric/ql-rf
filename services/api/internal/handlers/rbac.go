package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/rbac"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
)

// RBACHandler handles RBAC-related requests.
type RBACHandler struct {
	svc *rbac.Service
	log *logger.Logger
}

// NewRBACHandler creates a new RBACHandler.
func NewRBACHandler(svc *rbac.Service, log *logger.Logger) *RBACHandler {
	return &RBACHandler{
		svc: svc,
		log: log.WithComponent("rbac-handler"),
	}
}

// ListRoles returns all roles available to the organization.
func (h *RBACHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	roles, err := h.svc.ListRoles(ctx, org.ID)
	if err != nil {
		h.log.Error("failed to list roles", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"roles": roles,
	})
}

// GetRole returns role details by ID.
func (h *RBACHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	roleID := chi.URLParam(r, "roleId")
	id, err := uuid.Parse(roleID)
	if err != nil {
		http.Error(w, "invalid role ID", http.StatusBadRequest)
		return
	}

	// Get all roles and find the one with matching ID
	roles, err := h.svc.ListRoles(ctx, org.ID)
	if err != nil {
		h.log.Error("failed to get roles", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	for _, role := range roles {
		if role.ID == id {
			writeJSON(w, http.StatusOK, role)
			return
		}
	}

	http.Error(w, "role not found", http.StatusNotFound)
}

// ListPermissions returns all available permissions in the system.
func (h *RBACHandler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	// For now, return hardcoded system permissions
	// In production, this would query the database
	permissions := []map[string]interface{}{
		{
			"id":            uuid.New(),
			"name":          "read:assets",
			"resource_type": "assets",
			"action":        "read",
			"description":   "View assets",
			"is_system":     true,
		},
		{
			"id":            uuid.New(),
			"name":          "write:assets",
			"resource_type": "assets",
			"action":        "write",
			"description":   "Create and update assets",
			"is_system":     true,
		},
		{
			"id":            uuid.New(),
			"name":          "read:images",
			"resource_type": "images",
			"action":        "read",
			"description":   "View images",
			"is_system":     true,
		},
		{
			"id":            uuid.New(),
			"name":          "write:images",
			"resource_type": "images",
			"action":        "write",
			"description":   "Create and update images",
			"is_system":     true,
		},
		{
			"id":            uuid.New(),
			"name":          "execute:dr",
			"resource_type": "dr",
			"action":        "execute",
			"description":   "Execute DR drills",
			"is_system":     true,
		},
		{
			"id":            uuid.New(),
			"name":          "admin:organization",
			"resource_type": "organization",
			"action":        "admin",
			"description":   "Administer organization",
			"is_system":     true,
		},
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"permissions": permissions,
	})
}

// GetUserRoles returns roles assigned to a user.
func (h *RBACHandler) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	userID := chi.URLParam(r, "userId")
	if userID == "" {
		http.Error(w, "user ID is required", http.StatusBadRequest)
		return
	}

	roles, err := h.svc.GetUserRoles(ctx, userID, org.ID)
	if err != nil {
		h.log.Error("failed to get user roles", "error", err, "user_id", userID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"roles": roles,
	})
}

// AssignRole assigns a role to a user.
func (h *RBACHandler) AssignRole(w http.ResponseWriter, r *http.Request) {
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

	userID := chi.URLParam(r, "userId")
	if userID == "" {
		http.Error(w, "user ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		RoleID    string     `json:"role_id"`
		ExpiresAt *time.Time `json:"expires_at,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		http.Error(w, "invalid role ID", http.StatusBadRequest)
		return
	}

	err = h.svc.AssignRole(ctx, userID, org.ID, roleID, user.ExternalID, req.ExpiresAt)
	if err != nil {
		h.log.Error("failed to assign role", "error", err, "user_id", userID, "role_id", roleID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.log.Info("role assigned",
		"user_id", userID,
		"role_id", roleID,
		"assigned_by", user.ExternalID,
	)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "role assigned successfully",
	})
}

// RevokeRole revokes a role from a user.
func (h *RBACHandler) RevokeRole(w http.ResponseWriter, r *http.Request) {
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

	userID := chi.URLParam(r, "userId")
	roleIDStr := chi.URLParam(r, "roleId")

	if userID == "" || roleIDStr == "" {
		http.Error(w, "user ID and role ID are required", http.StatusBadRequest)
		return
	}

	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		http.Error(w, "invalid role ID", http.StatusBadRequest)
		return
	}

	err = h.svc.RevokeRole(ctx, userID, org.ID, roleID, user.ExternalID)
	if err != nil {
		h.log.Error("failed to revoke role", "error", err, "user_id", userID, "role_id", roleID)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.log.Info("role revoked",
		"user_id", userID,
		"role_id", roleID,
		"revoked_by", user.ExternalID,
	)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "role revoked successfully",
	})
}

// GetUserPermissions returns all effective permissions for a user.
func (h *RBACHandler) GetUserPermissions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	userID := chi.URLParam(r, "userId")
	if userID == "" {
		http.Error(w, "user ID is required", http.StatusBadRequest)
		return
	}

	permissions, err := h.svc.GetUserPermissions(ctx, userID, org.ID)
	if err != nil {
		h.log.Error("failed to get user permissions", "error", err, "user_id", userID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"permissions": permissions,
	})
}

// CheckPermission checks if a user has a specific permission.
func (h *RBACHandler) CheckPermission(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	var req struct {
		UserID       string     `json:"user_id"`
		ResourceType string     `json:"resource_type"`
		ResourceID   *uuid.UUID `json:"resource_id,omitempty"`
		Action       string     `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" || req.ResourceType == "" || req.Action == "" {
		http.Error(w, "user_id, resource_type, and action are required", http.StatusBadRequest)
		return
	}

	result, err := h.svc.CheckPermission(
		ctx,
		req.UserID,
		org.ID,
		rbac.ResourceType(req.ResourceType),
		req.ResourceID,
		rbac.Action(req.Action),
	)
	if err != nil {
		h.log.Error("failed to check permission", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ListTeams returns all teams in the organization.
func (h *RBACHandler) ListTeams(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	// For now, return empty list
	// In production, this would query the database for org teams
	h.log.Info("listing teams", "org_id", org.ID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"teams": []interface{}{},
	})
}

// CreateTeam creates a new team.
func (h *RBACHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
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
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	team := rbac.Team{
		OrgID:       org.ID,
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   user.ExternalID,
	}

	created, err := h.svc.CreateTeam(ctx, team)
	if err != nil {
		h.log.Error("failed to create team", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.log.Info("team created",
		"team_id", created.ID,
		"name", created.Name,
		"created_by", user.ExternalID,
	)

	writeJSON(w, http.StatusCreated, created)
}

// GetTeamMembers returns all members of a team.
func (h *RBACHandler) GetTeamMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	teamIDStr := chi.URLParam(r, "teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		http.Error(w, "invalid team ID", http.StatusBadRequest)
		return
	}

	members, err := h.svc.GetTeamMembers(ctx, teamID)
	if err != nil {
		h.log.Error("failed to get team members", "error", err, "team_id", teamID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"members": members,
	})
}

// AddTeamMember adds a user to a team.
func (h *RBACHandler) AddTeamMember(w http.ResponseWriter, r *http.Request) {
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

	teamIDStr := chi.URLParam(r, "teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		http.Error(w, "invalid team ID", http.StatusBadRequest)
		return
	}

	var req struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	// Default role is member
	if req.Role == "" {
		req.Role = "member"
	}

	err = h.svc.AddTeamMember(ctx, teamID, req.UserID, req.Role, user.ExternalID)
	if err != nil {
		h.log.Error("failed to add team member", "error", err, "team_id", teamID, "user_id", req.UserID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.log.Info("team member added",
		"team_id", teamID,
		"user_id", req.UserID,
		"role", req.Role,
		"added_by", user.ExternalID,
	)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "team member added successfully",
	})
}
