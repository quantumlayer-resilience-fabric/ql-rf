// Package models contains domain models used across services.
package models

import (
	"time"

	"github.com/google/uuid"
)

// Organization represents a tenant in the system.
type Organization struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Project represents a project within an organization.
type Project struct {
	ID        uuid.UUID `json:"id" db:"id"`
	OrgID     uuid.UUID `json:"org_id" db:"org_id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Environment represents an environment within a project.
type Environment struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ProjectID uuid.UUID `json:"project_id" db:"project_id"`
	Name      string    `json:"name" db:"name"` // prod, staging, dev, dr
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// EnvironmentName constants for common environment names.
const (
	EnvProduction  = "prod"
	EnvStaging     = "staging"
	EnvDevelopment = "dev"
	EnvDR          = "dr"
)

// User represents a user in the system.
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	ExternalID   string    `json:"external_id" db:"external_id"` // Clerk user ID
	Email        string    `json:"email" db:"email"`
	Name         string    `json:"name" db:"name"`
	Role         Role      `json:"role" db:"role"`
	OrgID        uuid.UUID `json:"org_id" db:"org_id"`
	LastLoginAt  time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Role represents a user's role in the system.
type Role string

const (
	RoleViewer   Role = "viewer"
	RoleOperator Role = "operator"
	RoleEngineer Role = "engineer"
	RoleAdmin    Role = "admin"
)

// RolePermissions maps roles to their permissions.
var RolePermissions = map[Role][]Permission{
	RoleViewer: {
		PermReadDashboard,
		PermReadDrift,
		PermReadAssets,
		PermReadImages,
		PermExportReports,
	},
	RoleOperator: {
		PermReadDashboard,
		PermReadDrift,
		PermReadAssets,
		PermReadImages,
		PermExportReports,
		PermTriggerDrill,
		PermAcknowledgeAlerts,
	},
	RoleEngineer: {
		PermReadDashboard,
		PermReadDrift,
		PermReadAssets,
		PermReadImages,
		PermExportReports,
		PermTriggerDrill,
		PermAcknowledgeAlerts,
		PermExecuteRollout,
		PermManageImages,
		PermApplyPatches,
	},
	RoleAdmin: {
		PermReadDashboard,
		PermReadDrift,
		PermReadAssets,
		PermReadImages,
		PermExportReports,
		PermTriggerDrill,
		PermAcknowledgeAlerts,
		PermExecuteRollout,
		PermManageImages,
		PermApplyPatches,
		PermManageRBAC,
		PermConfigureIntegrations,
		PermApproveExceptions,
	},
}

// Permission represents an action that can be performed.
type Permission string

const (
	PermReadDashboard         Permission = "read:dashboard"
	PermReadDrift             Permission = "read:drift"
	PermReadAssets            Permission = "read:assets"
	PermReadImages            Permission = "read:images"
	PermExportReports         Permission = "export:reports"
	PermTriggerDrill          Permission = "trigger:drill"
	PermAcknowledgeAlerts     Permission = "acknowledge:alerts"
	PermExecuteRollout        Permission = "execute:rollout"
	PermManageImages          Permission = "manage:images"
	PermApplyPatches          Permission = "apply:patches"
	PermManageRBAC            Permission = "manage:rbac"
	PermConfigureIntegrations Permission = "configure:integrations"
	PermApproveExceptions     Permission = "approve:exceptions"
)

// HasPermission checks if a role has a specific permission.
func (r Role) HasPermission(perm Permission) bool {
	permissions, ok := RolePermissions[r]
	if !ok {
		return false
	}
	for _, p := range permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// IsValid checks if the role is valid.
func (r Role) IsValid() bool {
	switch r {
	case RoleViewer, RoleOperator, RoleEngineer, RoleAdmin:
		return true
	default:
		return false
	}
}
