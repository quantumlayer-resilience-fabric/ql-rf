-- Migration: Rollback FinOps Tables
-- Purpose: Remove cost tracking, budgets, and optimization tables

-- Drop materialized views
DROP MATERIALIZED VIEW IF EXISTS mv_daily_cost_summary;

-- Drop triggers
DROP TRIGGER IF EXISTS cost_budgets_updated_at ON cost_budgets;
DROP TRIGGER IF EXISTS cost_recommendations_updated_at ON cost_recommendations;

-- Drop trigger functions
DROP FUNCTION IF EXISTS update_cost_budgets_updated_at();
DROP FUNCTION IF EXISTS update_cost_recommendations_updated_at();

-- Drop tables (in reverse order of dependencies)
DROP TABLE IF EXISTS cost_alerts;
DROP TABLE IF EXISTS cost_budgets;
DROP TABLE IF EXISTS cost_recommendations;
DROP TABLE IF EXISTS cost_records;
