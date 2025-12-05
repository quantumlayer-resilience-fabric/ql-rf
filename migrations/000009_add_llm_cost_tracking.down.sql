-- Rollback: Remove LLM Cost Tracking

DROP VIEW IF EXISTS v_llm_cost_report;
DROP FUNCTION IF EXISTS check_llm_quota(UUID);
DROP TRIGGER IF EXISTS llm_usage_update_monthly ON llm_usage;
DROP FUNCTION IF EXISTS update_monthly_usage();
DROP TABLE IF EXISTS org_monthly_usage;
DROP TABLE IF EXISTS org_llm_quotas;
DROP TABLE IF EXISTS llm_pricing;
DROP TABLE IF EXISTS llm_usage;
