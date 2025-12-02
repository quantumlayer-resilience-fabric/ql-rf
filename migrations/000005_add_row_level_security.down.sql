-- QuantumLayer Resilience Fabric - Row-Level Security Rollback
-- Migration: 000005_add_row_level_security (down)
-- Description: Removes Row-Level Security (RLS) policies

-- =============================================================================
-- Drop RLS Policies
-- =============================================================================

-- Org AI Settings
DROP POLICY IF EXISTS org_ai_settings_tenant_isolation ON org_ai_settings;

-- AI Tool Invocations
DROP POLICY IF EXISTS ai_tool_invocation_tenant_isolation ON ai_tool_invocations;

-- AI Runs
DROP POLICY IF EXISTS ai_run_tenant_isolation ON ai_runs;

-- AI Plans
DROP POLICY IF EXISTS ai_plan_tenant_isolation ON ai_plans;

-- AI Tasks
DROP POLICY IF EXISTS ai_task_tenant_isolation ON ai_tasks;

-- Connectors
DROP POLICY IF EXISTS connector_tenant_isolation ON connectors;

-- Drift Reports
DROP POLICY IF EXISTS drift_report_tenant_isolation ON drift_reports;

-- Assets
DROP POLICY IF EXISTS asset_tenant_isolation ON assets;

-- Image Coordinates
DROP POLICY IF EXISTS image_coord_tenant_isolation ON image_coordinates;

-- Images
DROP POLICY IF EXISTS image_tenant_isolation ON images;

-- Users
DROP POLICY IF EXISTS user_tenant_isolation ON users;

-- Environments
DROP POLICY IF EXISTS environment_tenant_isolation ON environments;

-- Projects
DROP POLICY IF EXISTS project_tenant_isolation ON projects;

-- Organizations
DROP POLICY IF EXISTS org_tenant_isolation ON organizations;

-- =============================================================================
-- Disable RLS on all tables
-- =============================================================================

ALTER TABLE org_ai_settings DISABLE ROW LEVEL SECURITY;
ALTER TABLE ai_tool_invocations DISABLE ROW LEVEL SECURITY;
ALTER TABLE ai_runs DISABLE ROW LEVEL SECURITY;
ALTER TABLE ai_plans DISABLE ROW LEVEL SECURITY;
ALTER TABLE ai_tasks DISABLE ROW LEVEL SECURITY;
ALTER TABLE connectors DISABLE ROW LEVEL SECURITY;
ALTER TABLE drift_reports DISABLE ROW LEVEL SECURITY;
ALTER TABLE assets DISABLE ROW LEVEL SECURITY;
ALTER TABLE image_coordinates DISABLE ROW LEVEL SECURITY;
ALTER TABLE images DISABLE ROW LEVEL SECURITY;
ALTER TABLE users DISABLE ROW LEVEL SECURITY;
ALTER TABLE environments DISABLE ROW LEVEL SECURITY;
ALTER TABLE projects DISABLE ROW LEVEL SECURITY;
ALTER TABLE organizations DISABLE ROW LEVEL SECURITY;

-- =============================================================================
-- Drop helper functions
-- =============================================================================

DROP FUNCTION IF EXISTS set_admin_mode(BOOLEAN);
DROP FUNCTION IF EXISTS user_has_org_access(UUID);
DROP FUNCTION IF EXISTS current_org_id();

-- =============================================================================
-- Drop context table
-- =============================================================================

DROP TABLE IF EXISTS app_context;
