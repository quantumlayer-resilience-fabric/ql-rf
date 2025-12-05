-- Migration: Remove RBAC (Role-Based Access Control)
-- Down migration to rollback the RBAC schema

-- Drop functions
DROP FUNCTION IF EXISTS get_user_permissions(VARCHAR(255), UUID);
DROP FUNCTION IF EXISTS check_permission(VARCHAR(255), UUID, VARCHAR(50), UUID, VARCHAR(50));

-- Drop indexes
DROP INDEX IF EXISTS idx_permission_grants_log_org;
DROP INDEX IF EXISTS idx_resource_permissions_grantee;
DROP INDEX IF EXISTS idx_resource_permissions_resource;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS permission_grants_log;
DROP TABLE IF EXISTS access_policies;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS resource_permissions;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
