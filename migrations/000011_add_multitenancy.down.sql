-- Migration: Remove Multi-tenancy Support
-- Down migration to rollback the multi-tenancy schema

-- Drop triggers
DROP TRIGGER IF EXISTS track_asset_count_trigger ON assets;
DROP TRIGGER IF EXISTS track_image_count_trigger ON images;
DROP TRIGGER IF EXISTS track_site_count_trigger ON sites;

-- Drop functions
DROP FUNCTION IF EXISTS track_site_count();
DROP FUNCTION IF EXISTS track_image_count();
DROP FUNCTION IF EXISTS track_asset_count();
DROP FUNCTION IF EXISTS check_api_rate_limit(UUID);
DROP FUNCTION IF EXISTS decrement_usage(UUID, VARCHAR, INTEGER);
DROP FUNCTION IF EXISTS increment_usage(UUID, VARCHAR, INTEGER);
DROP FUNCTION IF EXISTS check_quota(UUID, VARCHAR, INTEGER);
DROP FUNCTION IF EXISTS clear_tenant_context();
DROP FUNCTION IF EXISTS set_tenant_context(UUID, VARCHAR);
DROP FUNCTION IF EXISTS current_tenant_org_id();

-- Drop indexes
DROP INDEX IF EXISTS idx_usage_history_org_period;

-- Drop tables
DROP TABLE IF EXISTS organization_subscriptions;
DROP TABLE IF EXISTS subscription_plans;
DROP TABLE IF EXISTS tenant_context;
DROP TABLE IF EXISTS organization_usage_history;
DROP TABLE IF EXISTS organization_usage;
DROP TABLE IF EXISTS organization_quotas;
