-- QuantumLayer Resilience Fabric - Row-Level Security
-- Migration: 000005_add_row_level_security
-- Description: Adds Row-Level Security (RLS) policies for multi-tenancy

-- =============================================================================
-- Setup: Create application user and configuration
-- =============================================================================

-- Configuration table for RLS context
-- This allows the application to set the current org_id before queries
CREATE TABLE IF NOT EXISTS app_context (
    key VARCHAR(63) PRIMARY KEY,
    value TEXT NOT NULL
);

-- Function to get current organization ID from context
CREATE OR REPLACE FUNCTION current_org_id()
RETURNS UUID AS $$
DECLARE
    org_id UUID;
BEGIN
    -- First try session variable (set by application)
    BEGIN
        org_id := current_setting('app.current_org_id')::UUID;
        RETURN org_id;
    EXCEPTION WHEN OTHERS THEN
        -- No org_id set, return nil (will match nothing)
        RETURN NULL;
    END;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to check if user has access to an organization
CREATE OR REPLACE FUNCTION user_has_org_access(org_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    -- Check if the requested org_id matches the current session org_id
    RETURN org_id = current_org_id();
END;
$$ LANGUAGE plpgsql STABLE;

-- =============================================================================
-- Enable RLS on all tenant-scoped tables
-- =============================================================================

-- Organizations: Users can only see their own org
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;

-- Projects: Users can only see projects in their org
ALTER TABLE projects ENABLE ROW LEVEL SECURITY;

-- Environments: Users can only see environments in their org's projects
ALTER TABLE environments ENABLE ROW LEVEL SECURITY;

-- Users: Users can see other users in their org
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Images: Users can only see images in their org
ALTER TABLE images ENABLE ROW LEVEL SECURITY;

-- Image Coordinates: Users can only see coordinates for their org's images
ALTER TABLE image_coordinates ENABLE ROW LEVEL SECURITY;

-- Assets: Users can only see assets in their org
ALTER TABLE assets ENABLE ROW LEVEL SECURITY;

-- Drift Reports: Users can only see drift reports for their org
ALTER TABLE drift_reports ENABLE ROW LEVEL SECURITY;

-- Connectors: Users can only see connectors for their org
ALTER TABLE connectors ENABLE ROW LEVEL SECURITY;

-- AI Tasks: Users can only see AI tasks for their org
ALTER TABLE ai_tasks ENABLE ROW LEVEL SECURITY;

-- AI Plans: Users can only see plans for their org's tasks
ALTER TABLE ai_plans ENABLE ROW LEVEL SECURITY;

-- AI Runs: Users can only see runs for their org's tasks
ALTER TABLE ai_runs ENABLE ROW LEVEL SECURITY;

-- AI Tool Invocations: Users can only see invocations for their org's tasks
ALTER TABLE ai_tool_invocations ENABLE ROW LEVEL SECURITY;

-- Org AI Settings: Users can only see settings for their org
ALTER TABLE org_ai_settings ENABLE ROW LEVEL SECURITY;

-- =============================================================================
-- RLS Policies: Organizations
-- =============================================================================

CREATE POLICY org_tenant_isolation ON organizations
    FOR ALL
    USING (id = current_org_id());

-- =============================================================================
-- RLS Policies: Projects
-- =============================================================================

CREATE POLICY project_tenant_isolation ON projects
    FOR ALL
    USING (org_id = current_org_id());

-- =============================================================================
-- RLS Policies: Environments
-- =============================================================================

CREATE POLICY environment_tenant_isolation ON environments
    FOR ALL
    USING (EXISTS (
        SELECT 1 FROM projects p
        WHERE p.id = environments.project_id
        AND p.org_id = current_org_id()
    ));

-- =============================================================================
-- RLS Policies: Users
-- =============================================================================

CREATE POLICY user_tenant_isolation ON users
    FOR ALL
    USING (org_id = current_org_id());

-- =============================================================================
-- RLS Policies: Images
-- =============================================================================

CREATE POLICY image_tenant_isolation ON images
    FOR ALL
    USING (org_id = current_org_id());

-- =============================================================================
-- RLS Policies: Image Coordinates
-- =============================================================================

CREATE POLICY image_coord_tenant_isolation ON image_coordinates
    FOR ALL
    USING (EXISTS (
        SELECT 1 FROM images i
        WHERE i.id = image_coordinates.image_id
        AND i.org_id = current_org_id()
    ));

-- =============================================================================
-- RLS Policies: Assets
-- =============================================================================

CREATE POLICY asset_tenant_isolation ON assets
    FOR ALL
    USING (org_id = current_org_id());

-- =============================================================================
-- RLS Policies: Drift Reports
-- =============================================================================

CREATE POLICY drift_report_tenant_isolation ON drift_reports
    FOR ALL
    USING (org_id = current_org_id());

-- =============================================================================
-- RLS Policies: Connectors
-- =============================================================================

CREATE POLICY connector_tenant_isolation ON connectors
    FOR ALL
    USING (org_id = current_org_id());

-- =============================================================================
-- RLS Policies: AI Tasks
-- =============================================================================

CREATE POLICY ai_task_tenant_isolation ON ai_tasks
    FOR ALL
    USING (org_id = current_org_id());

-- =============================================================================
-- RLS Policies: AI Plans
-- =============================================================================

CREATE POLICY ai_plan_tenant_isolation ON ai_plans
    FOR ALL
    USING (EXISTS (
        SELECT 1 FROM ai_tasks t
        WHERE t.id = ai_plans.task_id
        AND t.org_id = current_org_id()
    ));

-- =============================================================================
-- RLS Policies: AI Runs
-- =============================================================================

CREATE POLICY ai_run_tenant_isolation ON ai_runs
    FOR ALL
    USING (EXISTS (
        SELECT 1 FROM ai_tasks t
        WHERE t.id = ai_runs.task_id
        AND t.org_id = current_org_id()
    ));

-- =============================================================================
-- RLS Policies: AI Tool Invocations
-- =============================================================================

CREATE POLICY ai_tool_invocation_tenant_isolation ON ai_tool_invocations
    FOR ALL
    USING (EXISTS (
        SELECT 1 FROM ai_tasks t
        WHERE t.id = ai_tool_invocations.task_id
        AND t.org_id = current_org_id()
    ));

-- =============================================================================
-- RLS Policies: Org AI Settings
-- =============================================================================

CREATE POLICY org_ai_settings_tenant_isolation ON org_ai_settings
    FOR ALL
    USING (org_id = current_org_id());

-- =============================================================================
-- Bypass RLS for service accounts (superuser operations)
-- The application user should NOT be a superuser, but we need a way
-- for migrations and admin operations to bypass RLS
-- =============================================================================

-- Create a function to bypass RLS for admin operations
CREATE OR REPLACE FUNCTION set_admin_mode(enable BOOLEAN)
RETURNS VOID AS $$
BEGIN
    IF enable THEN
        SET app.admin_mode = 'true';
    ELSE
        SET app.admin_mode = 'false';
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Note: The application should use SET LOCAL app.current_org_id = '<uuid>'
-- at the beginning of each request to enable RLS filtering.
-- Example in Go:
--   _, err := db.Exec("SET LOCAL app.current_org_id = $1", orgID)
