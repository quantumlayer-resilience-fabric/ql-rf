-- Organizations
-- name: GetOrganization :one
SELECT * FROM organizations WHERE id = $1;

-- name: GetOrganizationBySlug :one
SELECT * FROM organizations WHERE slug = $1;

-- name: ListOrganizations :many
SELECT * FROM organizations ORDER BY name LIMIT $1 OFFSET $2;

-- name: CreateOrganization :one
INSERT INTO organizations (name, slug)
VALUES ($1, $2)
RETURNING *;

-- Projects
-- name: GetProject :one
SELECT * FROM projects WHERE id = $1;

-- name: ListProjectsByOrg :many
SELECT * FROM projects WHERE org_id = $1 ORDER BY name;

-- name: CreateProject :one
INSERT INTO projects (org_id, name, slug)
VALUES ($1, $2, $3)
RETURNING *;

-- Environments
-- name: GetEnvironment :one
SELECT * FROM environments WHERE id = $1;

-- name: ListEnvironmentsByProject :many
SELECT * FROM environments WHERE project_id = $1 ORDER BY name;

-- name: CreateEnvironment :one
INSERT INTO environments (project_id, name)
VALUES ($1, $2)
RETURNING *;

-- Images
-- name: GetImage :one
SELECT * FROM images WHERE id = $1;

-- name: GetImageByFamilyVersion :one
SELECT * FROM images
WHERE org_id = $1 AND family = $2 AND version = $3;

-- name: GetLatestImageByFamily :one
SELECT * FROM images
WHERE org_id = $1 AND family = $2 AND status = 'production'
ORDER BY created_at DESC
LIMIT 1;

-- name: ListImages :many
SELECT * FROM images
WHERE org_id = $1
ORDER BY family, created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListImagesByFamily :many
SELECT * FROM images
WHERE org_id = $1 AND family = $2
ORDER BY created_at DESC;

-- name: ListImagesByStatus :many
SELECT * FROM images
WHERE org_id = $1 AND status = $2
ORDER BY family, created_at DESC
LIMIT $3 OFFSET $4;

-- name: CreateImage :one
INSERT INTO images (
    org_id, family, version, os_name, os_version,
    cis_level, sbom_url, signed, status
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateImageStatus :one
UPDATE images
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CountImagesByOrg :one
SELECT COUNT(*) FROM images WHERE org_id = $1;

-- Image Coordinates
-- name: GetImageCoordinates :many
SELECT * FROM image_coordinates WHERE image_id = $1;

-- name: GetImageCoordinateByPlatformRegion :one
SELECT * FROM image_coordinates
WHERE image_id = $1 AND platform = $2 AND region = $3;

-- name: CreateImageCoordinate :one
INSERT INTO image_coordinates (image_id, platform, region, identifier)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: DeleteImageCoordinates :exec
DELETE FROM image_coordinates WHERE image_id = $1;

-- Assets
-- name: GetAsset :one
SELECT * FROM assets WHERE id = $1;

-- name: GetAssetByInstanceID :one
SELECT * FROM assets
WHERE org_id = $1 AND platform = $2 AND instance_id = $3;

-- name: ListAssets :many
SELECT * FROM assets
WHERE org_id = $1
ORDER BY discovered_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAssetsByEnv :many
SELECT * FROM assets
WHERE org_id = $1 AND env_id = $2
ORDER BY discovered_at DESC
LIMIT $3 OFFSET $4;

-- name: ListAssetsByPlatform :many
SELECT * FROM assets
WHERE org_id = $1 AND platform = $2
ORDER BY discovered_at DESC
LIMIT $3 OFFSET $4;

-- name: ListAssetsByState :many
SELECT * FROM assets
WHERE org_id = $1 AND state = $2
ORDER BY discovered_at DESC
LIMIT $3 OFFSET $4;

-- name: ListOutdatedAssets :many
SELECT a.* FROM assets a
JOIN images i ON a.org_id = i.org_id AND a.image_ref = i.family
WHERE a.org_id = $1
  AND a.state = 'running'
  AND a.image_version != i.version
  AND i.status = 'production'
ORDER BY a.discovered_at DESC
LIMIT $2 OFFSET $3;

-- name: UpsertAsset :one
INSERT INTO assets (
    org_id, env_id, platform, account, region,
    instance_id, image_ref, image_version, state, tags
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (org_id, platform, instance_id)
DO UPDATE SET
    env_id = EXCLUDED.env_id,
    account = EXCLUDED.account,
    region = EXCLUDED.region,
    image_ref = EXCLUDED.image_ref,
    image_version = EXCLUDED.image_version,
    state = EXCLUDED.state,
    tags = EXCLUDED.tags,
    updated_at = NOW()
RETURNING *;

-- name: DeleteAsset :exec
DELETE FROM assets WHERE id = $1;

-- name: CountAssetsByOrg :one
SELECT COUNT(*) FROM assets WHERE org_id = $1;

-- name: CountAssetsByOrgAndState :one
SELECT COUNT(*) FROM assets WHERE org_id = $1 AND state = $2;

-- name: CountCompliantAssets :one
SELECT COUNT(*) FROM assets a
JOIN images i ON a.org_id = i.org_id AND a.image_ref = i.family AND a.image_version = i.version
WHERE a.org_id = $1 AND a.state = 'running' AND i.status = 'production';

-- Drift Reports
-- name: GetDriftReport :one
SELECT * FROM drift_reports WHERE id = $1;

-- name: GetLatestDriftReport :one
SELECT * FROM drift_reports
WHERE org_id = $1
ORDER BY calculated_at DESC
LIMIT 1;

-- name: GetLatestDriftReportByEnv :one
SELECT * FROM drift_reports
WHERE org_id = $1 AND env_id = $2
ORDER BY calculated_at DESC
LIMIT 1;

-- name: GetLatestDriftReportByPlatform :one
SELECT * FROM drift_reports
WHERE org_id = $1 AND platform = $2
ORDER BY calculated_at DESC
LIMIT 1;

-- name: ListDriftReports :many
SELECT * FROM drift_reports
WHERE org_id = $1
ORDER BY calculated_at DESC
LIMIT $2 OFFSET $3;

-- name: ListDriftReportsByDateRange :many
SELECT * FROM drift_reports
WHERE org_id = $1
  AND calculated_at >= $2
  AND calculated_at <= $3
ORDER BY calculated_at DESC;

-- name: CreateDriftReport :one
INSERT INTO drift_reports (
    org_id, env_id, platform, site,
    total_assets, compliant_assets, coverage_pct
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetDriftTrend :many
SELECT
    DATE_TRUNC('day', calculated_at) as date,
    AVG(coverage_pct) as avg_coverage,
    SUM(total_assets) as total_assets,
    SUM(compliant_assets) as compliant_assets
FROM drift_reports
WHERE org_id = $1
  AND calculated_at >= $2
GROUP BY DATE_TRUNC('day', calculated_at)
ORDER BY date DESC;

-- Aggregation queries for drift summary
-- name: GetDriftByEnvironment :many
SELECT
    e.name as scope,
    COUNT(DISTINCT a.id) as total_assets,
    COUNT(DISTINCT CASE
        WHEN i.id IS NOT NULL AND a.image_version = i.version THEN a.id
    END) as compliant_assets
FROM assets a
LEFT JOIN environments e ON a.env_id = e.id
LEFT JOIN images i ON a.org_id = i.org_id
    AND a.image_ref = i.family
    AND i.status = 'production'
WHERE a.org_id = $1 AND a.state = 'running'
GROUP BY e.id, e.name;

-- name: GetDriftByPlatform :many
SELECT
    a.platform as scope,
    COUNT(*) as total_assets,
    COUNT(CASE
        WHEN i.id IS NOT NULL AND a.image_version = i.version THEN 1
    END) as compliant_assets
FROM assets a
LEFT JOIN images i ON a.org_id = i.org_id
    AND a.image_ref = i.family
    AND i.status = 'production'
WHERE a.org_id = $1 AND a.state = 'running'
GROUP BY a.platform;

-- name: GetDriftBySite :many
SELECT
    a.region as scope,
    COUNT(*) as total_assets,
    COUNT(CASE
        WHEN i.id IS NOT NULL AND a.image_version = i.version THEN 1
    END) as compliant_assets
FROM assets a
LEFT JOIN images i ON a.org_id = i.org_id
    AND a.image_ref = i.family
    AND i.status = 'production'
WHERE a.org_id = $1 AND a.state = 'running'
GROUP BY a.region;

-- Connectors
-- name: GetConnector :one
SELECT * FROM connectors WHERE id = $1;

-- name: ListConnectorsByOrg :many
SELECT * FROM connectors WHERE org_id = $1 ORDER BY name;

-- name: CreateConnector :one
INSERT INTO connectors (org_id, name, platform, config, enabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateConnectorLastSync :one
UPDATE connectors
SET last_sync_at = NOW(), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateConnectorStatus :one
UPDATE connectors
SET status = $2, status_message = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateConnectorEnabled :one
UPDATE connectors
SET enabled = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;
