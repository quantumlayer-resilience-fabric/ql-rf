-- Rollback: 000007_add_drift_optimization_indexes

DROP INDEX IF EXISTS idx_assets_org_state_running;
DROP INDEX IF EXISTS idx_assets_org_image_ref_version;
DROP INDEX IF EXISTS idx_images_org_family_status_production;
DROP INDEX IF EXISTS idx_drift_reports_org_calculated_at;
DROP INDEX IF EXISTS idx_drift_reports_org_platform_calculated;
DROP INDEX IF EXISTS idx_drift_reports_org_env_calculated;
DROP INDEX IF EXISTS idx_assets_list_covering;
DROP INDEX IF EXISTS idx_assets_org_site_state;
DROP INDEX IF EXISTS idx_assets_tags_gin;
