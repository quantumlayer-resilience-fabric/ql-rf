-- Migration: Add RBAC (Role-Based Access Control)
-- Purpose: Resource-level permissions for enterprise multi-tenancy

-- =============================================================================
-- ROLES AND PERMISSIONS
-- =============================================================================

-- System-defined roles
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    org_id UUID REFERENCES organizations(id), -- NULL = system role
    is_system_role BOOLEAN DEFAULT FALSE, -- System roles cannot be deleted
    parent_role_id UUID REFERENCES roles(id), -- Role hierarchy
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, name)
);

-- Permissions define actions on resources
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE, -- e.g., assets:read, images:write
    resource_type VARCHAR(50) NOT NULL, -- e.g., assets, images, sites, tasks
    action VARCHAR(50) NOT NULL, -- read, write, delete, execute, approve
    description TEXT,
    is_system BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Role-Permission assignments
CREATE TABLE role_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    conditions JSONB DEFAULT '{}', -- Optional conditions like {"site_ids": [...]}
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(role_id, permission_id)
);

-- User-Role assignments
CREATE TABLE user_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL, -- Clerk user ID
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_by VARCHAR(255), -- Who assigned this role
    expires_at TIMESTAMPTZ, -- Optional expiration
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, org_id, role_id)
);

-- =============================================================================
-- RESOURCE-LEVEL PERMISSIONS
-- =============================================================================

-- Fine-grained permissions on specific resources
CREATE TABLE resource_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    resource_type VARCHAR(50) NOT NULL, -- asset, site, image, etc.
    resource_id UUID NOT NULL, -- ID of the specific resource
    
    -- Who has access
    grantee_type VARCHAR(20) NOT NULL, -- user, role, team
    grantee_id VARCHAR(255) NOT NULL, -- User ID, role ID, or team ID
    
    -- What access
    permission VARCHAR(50) NOT NULL, -- read, write, delete, execute, admin
    
    -- Constraints
    granted_by VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ,
    conditions JSONB DEFAULT '{}', -- Additional conditions
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(org_id, resource_type, resource_id, grantee_type, grantee_id, permission)
);

CREATE INDEX idx_resource_permissions_resource ON resource_permissions(org_id, resource_type, resource_id);
CREATE INDEX idx_resource_permissions_grantee ON resource_permissions(grantee_type, grantee_id);

-- =============================================================================
-- TEAMS (for group-based permissions)
-- =============================================================================

CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, name)
);

CREATE TABLE team_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'member', -- member, admin
    added_by VARCHAR(255),
    added_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(team_id, user_id)
);

-- =============================================================================
-- ACCESS POLICIES (advanced policy-based access)
-- =============================================================================

CREATE TABLE access_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    
    -- Policy definition (OPA-compatible Rego or simple JSON)
    policy_type VARCHAR(20) NOT NULL DEFAULT 'json', -- json, rego
    policy JSONB NOT NULL,
    
    -- Targeting
    applies_to_roles UUID[], -- Roles this policy applies to
    applies_to_resources VARCHAR(50)[], -- Resource types
    
    -- Priority (higher = evaluated first)
    priority INTEGER DEFAULT 0,
    
    enabled BOOLEAN DEFAULT TRUE,
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- PERMISSION GRANTS LOG (audit trail for permissions)
-- =============================================================================

CREATE TABLE permission_grants_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    action VARCHAR(20) NOT NULL, -- grant, revoke, modify
    
    -- What was changed
    target_type VARCHAR(20) NOT NULL, -- user, role, team
    target_id VARCHAR(255) NOT NULL,
    
    -- The permission change
    resource_type VARCHAR(50),
    resource_id UUID,
    permission VARCHAR(50),
    old_value JSONB,
    new_value JSONB,
    
    -- Who made the change
    changed_by VARCHAR(255) NOT NULL,
    reason TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_permission_grants_log_org ON permission_grants_log(org_id, created_at DESC);

-- =============================================================================
-- INSERT SYSTEM ROLES
-- =============================================================================

INSERT INTO roles (name, display_name, description, is_system_role) VALUES
('org_owner', 'Organization Owner', 'Full control over the organization', TRUE),
('org_admin', 'Organization Admin', 'Administrative access to organization resources', TRUE),
('infra_admin', 'Infrastructure Admin', 'Manage infrastructure assets and images', TRUE),
('security_admin', 'Security Admin', 'Manage security policies and compliance', TRUE),
('dr_admin', 'DR Administrator', 'Manage disaster recovery and BCP', TRUE),
('operator', 'Operator', 'Execute operations and view results', TRUE),
('analyst', 'Analyst', 'View and analyze data, create reports', TRUE),
('viewer', 'Viewer', 'Read-only access to resources', TRUE);

-- =============================================================================
-- INSERT SYSTEM PERMISSIONS
-- =============================================================================

INSERT INTO permissions (name, resource_type, action, description) VALUES
-- Asset permissions
('assets:read', 'assets', 'read', 'View assets'),
('assets:write', 'assets', 'write', 'Create and modify assets'),
('assets:delete', 'assets', 'delete', 'Delete assets'),
('assets:execute', 'assets', 'execute', 'Execute operations on assets'),

-- Image permissions
('images:read', 'images', 'read', 'View golden images'),
('images:write', 'images', 'write', 'Create and modify images'),
('images:delete', 'images', 'delete', 'Delete images'),
('images:publish', 'images', 'execute', 'Publish images'),

-- Site permissions
('sites:read', 'sites', 'read', 'View sites'),
('sites:write', 'sites', 'write', 'Create and modify sites'),
('sites:delete', 'sites', 'delete', 'Delete sites'),

-- Drift permissions
('drift:read', 'drift', 'read', 'View drift reports'),
('drift:remediate', 'drift', 'execute', 'Remediate drift'),

-- Compliance permissions
('compliance:read', 'compliance', 'read', 'View compliance status'),
('compliance:scan', 'compliance', 'execute', 'Run compliance scans'),
('compliance:configure', 'compliance', 'write', 'Configure compliance frameworks'),

-- DR permissions
('dr:read', 'dr', 'read', 'View DR configurations'),
('dr:configure', 'dr', 'write', 'Configure DR pairs'),
('dr:drill', 'dr', 'execute', 'Execute DR drills'),
('dr:failover', 'dr', 'execute', 'Execute failover'),

-- Task permissions
('tasks:read', 'tasks', 'read', 'View AI tasks'),
('tasks:create', 'tasks', 'write', 'Create AI tasks'),
('tasks:approve', 'tasks', 'approve', 'Approve AI task plans'),
('tasks:execute', 'tasks', 'execute', 'Execute AI tasks'),

-- Organization permissions
('org:read', 'organization', 'read', 'View organization settings'),
('org:write', 'organization', 'write', 'Modify organization settings'),
('org:members', 'organization', 'write', 'Manage organization members'),
('org:roles', 'organization', 'write', 'Manage roles and permissions'),
('org:billing', 'organization', 'write', 'Manage billing and quotas'),

-- Audit permissions
('audit:read', 'audit', 'read', 'View audit logs'),
('audit:export', 'audit', 'execute', 'Export audit logs');

-- =============================================================================
-- ASSIGN PERMISSIONS TO ROLES
-- =============================================================================

-- org_owner gets all permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'org_owner';

-- org_admin gets most permissions except billing
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'org_admin' AND p.name NOT IN ('org:billing');

-- infra_admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'infra_admin' AND p.resource_type IN ('assets', 'images', 'sites', 'drift');

-- security_admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'security_admin' AND p.resource_type IN ('compliance', 'audit');

-- dr_admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'dr_admin' AND p.resource_type = 'dr';

-- operator
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'operator' AND p.action IN ('read', 'execute')
AND p.resource_type IN ('assets', 'images', 'drift', 'tasks');

-- analyst
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'analyst' AND p.action = 'read';

-- viewer
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'viewer' AND p.action = 'read'
AND p.resource_type NOT IN ('audit', 'organization');

-- =============================================================================
-- HELPER FUNCTIONS
-- =============================================================================

-- Check if user has permission on a resource
CREATE OR REPLACE FUNCTION check_permission(
    p_user_id VARCHAR(255),
    p_org_id UUID,
    p_resource_type VARCHAR(50),
    p_resource_id UUID,
    p_action VARCHAR(50)
) RETURNS BOOLEAN AS $$
DECLARE
    has_access BOOLEAN := FALSE;
BEGIN
    -- Check role-based permissions
    SELECT EXISTS (
        SELECT 1
        FROM user_roles ur
        JOIN role_permissions rp ON ur.role_id = rp.role_id
        JOIN permissions p ON rp.permission_id = p.id
        WHERE ur.user_id = p_user_id
        AND ur.org_id = p_org_id
        AND p.resource_type = p_resource_type
        AND p.action = p_action
        AND (ur.expires_at IS NULL OR ur.expires_at > NOW())
    ) INTO has_access;

    IF has_access THEN
        RETURN TRUE;
    END IF;

    -- Check direct resource permissions
    SELECT EXISTS (
        SELECT 1
        FROM resource_permissions rp
        WHERE rp.org_id = p_org_id
        AND rp.resource_type = p_resource_type
        AND (rp.resource_id = p_resource_id OR rp.resource_id IS NULL)
        AND rp.permission = p_action
        AND rp.grantee_type = 'user'
        AND rp.grantee_id = p_user_id
        AND (rp.expires_at IS NULL OR rp.expires_at > NOW())
    ) INTO has_access;

    IF has_access THEN
        RETURN TRUE;
    END IF;

    -- Check team-based permissions
    SELECT EXISTS (
        SELECT 1
        FROM resource_permissions rp
        JOIN team_members tm ON rp.grantee_id = tm.team_id::text
        WHERE rp.org_id = p_org_id
        AND rp.resource_type = p_resource_type
        AND (rp.resource_id = p_resource_id OR rp.resource_id IS NULL)
        AND rp.permission = p_action
        AND rp.grantee_type = 'team'
        AND tm.user_id = p_user_id
        AND (rp.expires_at IS NULL OR rp.expires_at > NOW())
    ) INTO has_access;

    RETURN has_access;
END;
$$ LANGUAGE plpgsql;

-- Get all permissions for a user in an org
CREATE OR REPLACE FUNCTION get_user_permissions(
    p_user_id VARCHAR(255),
    p_org_id UUID
) RETURNS TABLE(
    permission_name VARCHAR(100),
    resource_type VARCHAR(50),
    action VARCHAR(50),
    source VARCHAR(20)
) AS $$
BEGIN
    -- Role-based permissions
    RETURN QUERY
    SELECT DISTINCT
        p.name,
        p.resource_type,
        p.action,
        'role'::VARCHAR(20) as source
    FROM user_roles ur
    JOIN role_permissions rp ON ur.role_id = rp.role_id
    JOIN permissions p ON rp.permission_id = p.id
    WHERE ur.user_id = p_user_id
    AND ur.org_id = p_org_id
    AND (ur.expires_at IS NULL OR ur.expires_at > NOW());

    -- Direct permissions
    RETURN QUERY
    SELECT DISTINCT
        (rp.resource_type || ':' || rp.permission)::VARCHAR(100),
        rp.resource_type,
        rp.permission,
        'direct'::VARCHAR(20) as source
    FROM resource_permissions rp
    WHERE rp.org_id = p_org_id
    AND rp.grantee_type = 'user'
    AND rp.grantee_id = p_user_id
    AND (rp.expires_at IS NULL OR rp.expires_at > NOW());

    -- Team permissions
    RETURN QUERY
    SELECT DISTINCT
        (rp.resource_type || ':' || rp.permission)::VARCHAR(100),
        rp.resource_type,
        rp.permission,
        'team'::VARCHAR(20) as source
    FROM resource_permissions rp
    JOIN team_members tm ON rp.grantee_id = tm.team_id::text
    WHERE rp.org_id = p_org_id
    AND rp.grantee_type = 'team'
    AND tm.user_id = p_user_id
    AND (rp.expires_at IS NULL OR rp.expires_at > NOW());
END;
$$ LANGUAGE plpgsql;

-- Comments
COMMENT ON TABLE roles IS 'System and organization-defined roles';
COMMENT ON TABLE permissions IS 'Available permissions in the system';
COMMENT ON TABLE user_roles IS 'User to role assignments';
COMMENT ON TABLE resource_permissions IS 'Fine-grained resource-level permissions';
COMMENT ON TABLE teams IS 'Teams for group-based access control';
COMMENT ON TABLE access_policies IS 'Advanced policy-based access rules';
COMMENT ON FUNCTION check_permission IS 'Check if user has specific permission';
