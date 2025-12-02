-- QuantumLayer Resilience Fabric - AI Orchestration Schema Rollback
-- Migration: 000004_add_ai_orchestration (DOWN)
-- Description: Removes AI orchestration tables

-- Drop triggers first
DROP TRIGGER IF EXISTS update_org_ai_settings_updated_at ON org_ai_settings;
DROP TRIGGER IF EXISTS update_ai_prompts_updated_at ON ai_prompts;
DROP TRIGGER IF EXISTS update_ai_runs_updated_at ON ai_runs;
DROP TRIGGER IF EXISTS update_ai_plans_updated_at ON ai_plans;
DROP TRIGGER IF EXISTS update_ai_tasks_updated_at ON ai_tasks;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS org_ai_settings;
DROP TABLE IF EXISTS ai_prompts;
DROP TABLE IF EXISTS ai_tool_invocations;
DROP TABLE IF EXISTS ai_runs;
DROP TABLE IF EXISTS ai_plans;
DROP TABLE IF EXISTS ai_tasks;
