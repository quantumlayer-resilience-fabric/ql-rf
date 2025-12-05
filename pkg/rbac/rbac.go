// Package rbac provides Role-Based Access Control for the QL-RF platform.
// It supports hierarchical roles, resource-level permissions, and team-based access.
package rbac

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Action represents an action that can be performed on a resource.
type Action string

const (
	ActionRead    Action = "read"
	ActionWrite   Action = "write"
	ActionDelete  Action = "delete"
	ActionExecute Action = "execute"
	ActionApprove Action = "approve"
	ActionAdmin   Action = "admin"
)

// ResourceType represents a type of resource in the system.
type ResourceType string

const (
	ResourceAssets       ResourceType = "assets"
	ResourceImages       ResourceType = "images"
	ResourceSites        ResourceType = "sites"
	ResourceDrift        ResourceType = "drift"
	ResourceCompliance   ResourceType = "compliance"
	ResourceDR           ResourceType = "dr"
	ResourceTasks        ResourceType = "tasks"
	ResourceOrganization ResourceType = "organization"
	ResourceAudit        ResourceType = "audit"
)

// SystemRole represents a system-defined role.
type SystemRole string

const (
	RoleOrgOwner      SystemRole = "org_owner"
	RoleOrgAdmin      SystemRole = "org_admin"
	RoleInfraAdmin    SystemRole = "infra_admin"
	RoleSecurityAdmin SystemRole = "security_admin"
	RoleDRAdmin       SystemRole = "dr_admin"
	RoleOperator      SystemRole = "operator"
	RoleAnalyst       SystemRole = "analyst"
	RoleViewer        SystemRole = "viewer"
)

// Role represents a role in the system.
type Role struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Name         string     `json:"name" db:"name"`
	DisplayName  string     `json:"display_name" db:"display_name"`
	Description  string     `json:"description,omitempty" db:"description"`
	OrgID        *uuid.UUID `json:"org_id,omitempty" db:"org_id"`
	IsSystemRole bool       `json:"is_system_role" db:"is_system_role"`
	ParentRoleID *uuid.UUID `json:"parent_role_id,omitempty" db:"parent_role_id"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// Permission represents a permission definition.
type Permission struct {
	ID           uuid.UUID    `json:"id" db:"id"`
	Name         string       `json:"name" db:"name"`
	ResourceType ResourceType `json:"resource_type" db:"resource_type"`
	Action       Action       `json:"action" db:"action"`
	Description  string       `json:"description,omitempty" db:"description"`
	IsSystem     bool         `json:"is_system" db:"is_system"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
}

// UserRole represents a user's role assignment.
type UserRole struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	UserID     string     `json:"user_id" db:"user_id"`
	OrgID      uuid.UUID  `json:"org_id" db:"org_id"`
	RoleID     uuid.UUID  `json:"role_id" db:"role_id"`
	AssignedBy string     `json:"assigned_by,omitempty" db:"assigned_by"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// ResourcePermission represents a fine-grained permission on a specific resource.
type ResourcePermission struct {
	ID           uuid.UUID    `json:"id" db:"id"`
	OrgID        uuid.UUID    `json:"org_id" db:"org_id"`
	ResourceType ResourceType `json:"resource_type" db:"resource_type"`
	ResourceID   uuid.UUID    `json:"resource_id" db:"resource_id"`
	GranteeType  string       `json:"grantee_type" db:"grantee_type"` // user, role, team
	GranteeID    string       `json:"grantee_id" db:"grantee_id"`
	Permission   Action       `json:"permission" db:"permission"`
	GrantedBy    string       `json:"granted_by" db:"granted_by"`
	ExpiresAt    *time.Time   `json:"expires_at,omitempty" db:"expires_at"`
	Conditions   string       `json:"conditions,omitempty" db:"conditions"` // JSON
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at" db:"updated_at"`
}

// Team represents a team for group-based permissions.
type Team struct {
	ID          uuid.UUID `json:"id" db:"id"`
	OrgID       uuid.UUID `json:"org_id" db:"org_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	CreatedBy   string    `json:"created_by" db:"created_by"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// TeamMember represents a team membership.
type TeamMember struct {
	ID      uuid.UUID `json:"id" db:"id"`
	TeamID  uuid.UUID `json:"team_id" db:"team_id"`
	UserID  string    `json:"user_id" db:"user_id"`
	Role    string    `json:"role" db:"role"` // member, admin
	AddedBy string    `json:"added_by,omitempty" db:"added_by"`
	AddedAt time.Time `json:"added_at" db:"added_at"`
}

// PermissionCheck represents the result of a permission check.
type PermissionCheck struct {
	Allowed bool   `json:"allowed"`
	Source  string `json:"source"` // role, direct, team
	Reason  string `json:"reason,omitempty"`
}

// UserPermission represents a permission held by a user.
type UserPermission struct {
	PermissionName string       `json:"permission_name"`
	ResourceType   ResourceType `json:"resource_type"`
	Action         Action       `json:"action"`
	Source         string       `json:"source"` // role, direct, team
}

// Service provides RBAC functionality.
type Service struct {
	db *sql.DB
}

// NewService creates a new RBAC service.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// CheckPermission checks if a user has permission to perform an action on a resource.
func (s *Service) CheckPermission(ctx context.Context, userID string, orgID uuid.UUID, resourceType ResourceType, resourceID *uuid.UUID, action Action) (*PermissionCheck, error) {
	// Use the database function for permission checking
	var hasAccess bool
	var resourceIDArg interface{} = nil
	if resourceID != nil {
		resourceIDArg = *resourceID
	}

	err := s.db.QueryRowContext(ctx,
		"SELECT check_permission($1, $2, $3, $4, $5)",
		userID, orgID, string(resourceType), resourceIDArg, string(action),
	).Scan(&hasAccess)
	if err != nil {
		return nil, fmt.Errorf("failed to check permission: %w", err)
	}

	result := &PermissionCheck{
		Allowed: hasAccess,
	}

	if hasAccess {
		// Determine the source of the permission
		source, err := s.getPermissionSource(ctx, userID, orgID, resourceType, resourceID, action)
		if err == nil {
			result.Source = source
		}
	} else {
		result.Reason = "no matching permission found"
	}

	return result, nil
}

// getPermissionSource determines where a permission came from.
func (s *Service) getPermissionSource(ctx context.Context, userID string, orgID uuid.UUID, resourceType ResourceType, resourceID *uuid.UUID, action Action) (string, error) {
	// Check role-based first
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM user_roles ur
			JOIN role_permissions rp ON ur.role_id = rp.role_id
			JOIN permissions p ON rp.permission_id = p.id
			WHERE ur.user_id = $1
			AND ur.org_id = $2
			AND p.resource_type = $3
			AND p.action = $4
			AND (ur.expires_at IS NULL OR ur.expires_at > NOW())
		)
	`, userID, orgID, string(resourceType), string(action)).Scan(&exists)
	if err == nil && exists {
		return "role", nil
	}

	// Check direct permissions
	err = s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM resource_permissions rp
			WHERE rp.org_id = $1
			AND rp.resource_type = $2
			AND rp.permission = $3
			AND rp.grantee_type = 'user'
			AND rp.grantee_id = $4
			AND (rp.expires_at IS NULL OR rp.expires_at > NOW())
		)
	`, orgID, string(resourceType), string(action), userID).Scan(&exists)
	if err == nil && exists {
		return "direct", nil
	}

	// Check team permissions
	err = s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM resource_permissions rp
			JOIN team_members tm ON rp.grantee_id = tm.team_id::text
			WHERE rp.org_id = $1
			AND rp.resource_type = $2
			AND rp.permission = $3
			AND rp.grantee_type = 'team'
			AND tm.user_id = $4
			AND (rp.expires_at IS NULL OR rp.expires_at > NOW())
		)
	`, orgID, string(resourceType), string(action), userID).Scan(&exists)
	if err == nil && exists {
		return "team", nil
	}

	return "unknown", nil
}

// GetUserPermissions returns all permissions for a user in an organization.
func (s *Service) GetUserPermissions(ctx context.Context, userID string, orgID uuid.UUID) ([]UserPermission, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT * FROM get_user_permissions($1, $2)",
		userID, orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}
	defer rows.Close()

	var permissions []UserPermission
	for rows.Next() {
		var p UserPermission
		if err := rows.Scan(&p.PermissionName, &p.ResourceType, &p.Action, &p.Source); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, p)
	}

	return permissions, rows.Err()
}

// AssignRole assigns a role to a user in an organization.
func (s *Service) AssignRole(ctx context.Context, userID string, orgID, roleID uuid.UUID, assignedBy string, expiresAt *time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_roles (user_id, org_id, role_id, assigned_by, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, org_id, role_id) DO UPDATE
		SET assigned_by = $4, expires_at = $5
	`, userID, orgID, roleID, assignedBy, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	// Log the grant
	s.logPermissionGrant(ctx, orgID, "grant", "user", userID, "", nil, "", nil, assignedBy, "role assignment")

	return nil
}

// RevokeRole revokes a role from a user.
func (s *Service) RevokeRole(ctx context.Context, userID string, orgID, roleID uuid.UUID, revokedBy string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM user_roles
		WHERE user_id = $1 AND org_id = $2 AND role_id = $3
	`, userID, orgID, roleID)
	if err != nil {
		return fmt.Errorf("failed to revoke role: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("role assignment not found")
	}

	// Log the revocation
	s.logPermissionGrant(ctx, orgID, "revoke", "user", userID, "", nil, "", nil, revokedBy, "role revocation")

	return nil
}

// GetUserRoles returns all roles for a user in an organization.
func (s *Service) GetUserRoles(ctx context.Context, userID string, orgID uuid.UUID) ([]Role, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.name, r.display_name, r.description, r.org_id,
		       r.is_system_role, r.parent_role_id, r.created_at, r.updated_at
		FROM roles r
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND ur.org_id = $2
		AND (ur.expires_at IS NULL OR ur.expires_at > NOW())
	`, userID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.ID, &r.Name, &r.DisplayName, &r.Description, &r.OrgID,
			&r.IsSystemRole, &r.ParentRoleID, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, r)
	}

	return roles, rows.Err()
}

// GrantResourcePermission grants a specific permission on a resource.
func (s *Service) GrantResourcePermission(ctx context.Context, perm ResourcePermission) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO resource_permissions
		(org_id, resource_type, resource_id, grantee_type, grantee_id, permission, granted_by, expires_at, conditions)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (org_id, resource_type, resource_id, grantee_type, grantee_id, permission)
		DO UPDATE SET granted_by = $7, expires_at = $8, conditions = $9, updated_at = NOW()
	`, perm.OrgID, string(perm.ResourceType), perm.ResourceID, perm.GranteeType, perm.GranteeID,
		string(perm.Permission), perm.GrantedBy, perm.ExpiresAt, perm.Conditions)
	if err != nil {
		return fmt.Errorf("failed to grant resource permission: %w", err)
	}

	// Log the grant
	s.logPermissionGrant(ctx, perm.OrgID, "grant", perm.GranteeType, perm.GranteeID,
		string(perm.ResourceType), &perm.ResourceID, string(perm.Permission), nil, perm.GrantedBy, "resource permission grant")

	return nil
}

// RevokeResourcePermission revokes a specific permission on a resource.
func (s *Service) RevokeResourcePermission(ctx context.Context, orgID uuid.UUID, resourceType ResourceType, resourceID uuid.UUID, granteeType, granteeID string, permission Action, revokedBy string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM resource_permissions
		WHERE org_id = $1 AND resource_type = $2 AND resource_id = $3
		AND grantee_type = $4 AND grantee_id = $5 AND permission = $6
	`, orgID, string(resourceType), resourceID, granteeType, granteeID, string(permission))
	if err != nil {
		return fmt.Errorf("failed to revoke resource permission: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("resource permission not found")
	}

	// Log the revocation
	s.logPermissionGrant(ctx, orgID, "revoke", granteeType, granteeID,
		string(resourceType), &resourceID, string(permission), nil, revokedBy, "resource permission revocation")

	return nil
}

// CreateTeam creates a new team.
func (s *Service) CreateTeam(ctx context.Context, team Team) (*Team, error) {
	team.ID = uuid.New()
	team.CreatedAt = time.Now()
	team.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO teams (id, org_id, name, description, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, team.ID, team.OrgID, team.Name, team.Description, team.CreatedBy, team.CreatedAt, team.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create team: %w", err)
	}

	return &team, nil
}

// AddTeamMember adds a user to a team.
func (s *Service) AddTeamMember(ctx context.Context, teamID uuid.UUID, userID, role, addedBy string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO team_members (team_id, user_id, role, added_by, added_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (team_id, user_id) DO UPDATE SET role = $3
	`, teamID, userID, role, addedBy)
	if err != nil {
		return fmt.Errorf("failed to add team member: %w", err)
	}

	return nil
}

// RemoveTeamMember removes a user from a team.
func (s *Service) RemoveTeamMember(ctx context.Context, teamID uuid.UUID, userID string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM team_members WHERE team_id = $1 AND user_id = $2
	`, teamID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove team member: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("team member not found")
	}

	return nil
}

// GetTeamMembers returns all members of a team.
func (s *Service) GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]TeamMember, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, team_id, user_id, role, added_by, added_at
		FROM team_members WHERE team_id = $1
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}
	defer rows.Close()

	var members []TeamMember
	for rows.Next() {
		var m TeamMember
		if err := rows.Scan(&m.ID, &m.TeamID, &m.UserID, &m.Role, &m.AddedBy, &m.AddedAt); err != nil {
			return nil, fmt.Errorf("failed to scan team member: %w", err)
		}
		members = append(members, m)
	}

	return members, rows.Err()
}

// GetRoleByName returns a role by name.
func (s *Service) GetRoleByName(ctx context.Context, name string, orgID *uuid.UUID) (*Role, error) {
	var r Role
	var err error

	if orgID != nil {
		err = s.db.QueryRowContext(ctx, `
			SELECT id, name, display_name, description, org_id, is_system_role, parent_role_id, created_at, updated_at
			FROM roles WHERE name = $1 AND (org_id = $2 OR org_id IS NULL)
			ORDER BY org_id DESC NULLS LAST LIMIT 1
		`, name, *orgID).Scan(&r.ID, &r.Name, &r.DisplayName, &r.Description, &r.OrgID,
			&r.IsSystemRole, &r.ParentRoleID, &r.CreatedAt, &r.UpdatedAt)
	} else {
		err = s.db.QueryRowContext(ctx, `
			SELECT id, name, display_name, description, org_id, is_system_role, parent_role_id, created_at, updated_at
			FROM roles WHERE name = $1 AND org_id IS NULL
		`, name).Scan(&r.ID, &r.Name, &r.DisplayName, &r.Description, &r.OrgID,
			&r.IsSystemRole, &r.ParentRoleID, &r.CreatedAt, &r.UpdatedAt)
	}

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return &r, nil
}

// ListRoles returns all roles for an organization (including system roles).
func (s *Service) ListRoles(ctx context.Context, orgID uuid.UUID) ([]Role, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, display_name, description, org_id, is_system_role, parent_role_id, created_at, updated_at
		FROM roles WHERE org_id = $1 OR org_id IS NULL
		ORDER BY is_system_role DESC, name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.ID, &r.Name, &r.DisplayName, &r.Description, &r.OrgID,
			&r.IsSystemRole, &r.ParentRoleID, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, r)
	}

	return roles, rows.Err()
}

// logPermissionGrant logs a permission change to the audit log.
func (s *Service) logPermissionGrant(ctx context.Context, orgID uuid.UUID, action, targetType, targetID, resourceType string, resourceID *uuid.UUID, permission string, oldValue interface{}, changedBy, reason string) {
	// Best effort logging - don't fail the operation if logging fails
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO permission_grants_log
		(org_id, action, target_type, target_id, resource_type, resource_id, permission, changed_by, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, orgID, action, targetType, targetID, resourceType, resourceID, permission, changedBy, reason)
}

// RequirePermission is a helper that returns an error if the user lacks permission.
func (s *Service) RequirePermission(ctx context.Context, userID string, orgID uuid.UUID, resourceType ResourceType, resourceID *uuid.UUID, action Action) error {
	check, err := s.CheckPermission(ctx, userID, orgID, resourceType, resourceID, action)
	if err != nil {
		return fmt.Errorf("permission check failed: %w", err)
	}

	if !check.Allowed {
		return &PermissionDeniedError{
			UserID:       userID,
			ResourceType: resourceType,
			Action:       action,
			Reason:       check.Reason,
		}
	}

	return nil
}

// PermissionDeniedError represents a permission denied error.
type PermissionDeniedError struct {
	UserID       string
	ResourceType ResourceType
	Action       Action
	Reason       string
}

func (e *PermissionDeniedError) Error() string {
	return fmt.Sprintf("permission denied: user %s cannot %s on %s: %s",
		e.UserID, e.Action, e.ResourceType, e.Reason)
}

// IsPermissionDenied checks if an error is a permission denied error.
func IsPermissionDenied(err error) bool {
	_, ok := err.(*PermissionDeniedError)
	return ok
}
