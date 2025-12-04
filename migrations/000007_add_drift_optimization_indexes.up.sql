-- QuantumLayer Resilience Fabric - Drift Query Optimization Indexes
-- Migration: 000007_add_drift_optimization_indexes
-- Description: Adds indexes to optimize drift-related queries

-- =============================================================================
-- Composite Indexes for Drift Queries
-- =============================================================================

-- Index for drift queries that filter on running state and org
-- Used by: GetDriftByEnvironment, GetDriftByPlatform, GetDriftBySite, CountCompliantAssets
CREATE INDEX IF NOT EXISTS idx_assets_org_state_running
ON assets (org_id, state)
WHERE state = 'running';

-- Index for joining assets with images on family (image_ref)
-- Used by: CountCompliantAssets, ListOutdatedAssets, GetDriftBy* queries
CREATE INDEX IF NOT EXISTS idx_assets_org_image_ref_version
ON assets (org_id, image_ref, image_version);

-- Index for production images lookup (used in drift comparison)
-- Used by: CountCompliantAssets, GetDriftBy* queries
CREATE INDEX IF NOT EXISTS idx_images_org_family_status_production
ON images (org_id, family, version)
WHERE status = 'production';

-- Index for drift reports with date range queries
-- Used by: GetDriftTrend, ListDriftReportsByDateRange
CREATE INDEX IF NOT EXISTS idx_drift_reports_org_calculated_at
ON drift_reports (org_id, calculated_at DESC);

-- Index for platform-specific drift queries
-- Used by: GetLatestDriftReportByPlatform
CREATE INDEX IF NOT EXISTS idx_drift_reports_org_platform_calculated
ON drift_reports (org_id, platform, calculated_at DESC);

-- Index for environment-specific drift queries
-- Used by: GetLatestDriftReportByEnv
CREATE INDEX IF NOT EXISTS idx_drift_reports_org_env_calculated
ON drift_reports (org_id, env_id, calculated_at DESC);

-- =============================================================================
-- Covering Index for Asset Listing
-- =============================================================================

-- Covering index for common asset list queries
-- Includes columns commonly selected to avoid table lookups
CREATE INDEX IF NOT EXISTS idx_assets_list_covering
ON assets (org_id, discovered_at DESC)
INCLUDE (platform, state, instance_id, name, image_ref, image_version, site, region);

-- =============================================================================
-- Indexes for Site-based Queries
-- =============================================================================

-- Index for site-based filtering (used in drift by site)
CREATE INDEX IF NOT EXISTS idx_assets_org_site_state
ON assets (org_id, site, state);

-- =============================================================================
-- GIN Index for Tags (JSONB)
-- =============================================================================

-- GIN index for searching asset tags
CREATE INDEX IF NOT EXISTS idx_assets_tags_gin
ON assets USING GIN (tags);

-- =============================================================================
-- Statistics
-- =============================================================================

-- Update statistics for better query planning
ANALYZE assets;
ANALYZE images;
ANALYZE drift_reports;
