-- QuantumLayer Resilience Fabric - Additional Demo Assets
-- Adds more diverse assets to demonstrate platform capabilities
-- Run with: psql -d qlrf -f seed_more_assets.sql

-- Use existing site IDs from the database
-- US-East Production: 22222222-2222-2222-2222-222222222221
-- US-West DR: 22222222-2222-2222-2222-222222222222
-- EU-West Production: 55555555-5555-5555-5555-555555555503
-- Azure East US: 55555555-5555-5555-5555-555555555504
-- Azure West Europe: 55555555-5555-5555-5555-555555555505
-- GCP US Central: 55555555-5555-5555-5555-555555555506
-- DC London (vSphere): 55555555-5555-5555-5555-555555555507
-- K8s Production: 55555555-5555-5555-5555-555555555508

-- =============================================================================
-- More AWS Assets (US-East Production)
-- =============================================================================
INSERT INTO assets (id, org_id, env_id, platform, account, region, site, site_id, instance_id, name, image_ref, image_version, state, tags, discovered_at, updated_at) VALUES
    -- Web tier - latest image (compliant)
    ('aa000001-0001-0001-0001-000000000001', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'aws', '123456789012', 'us-east-1', 'US-East Production', '22222222-2222-2222-2222-222222222221', 'i-0a1b2c3d4e5f60001', 'web-prod-01', 'ql-base-ubuntu', '2.5.0', 'running', '{"role": "web", "tier": "frontend", "team": "platform"}', NOW() - INTERVAL '2 days', NOW()),
    ('aa000001-0001-0001-0001-000000000002', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'aws', '123456789012', 'us-east-1', 'US-East Production', '22222222-2222-2222-2222-222222222221', 'i-0a1b2c3d4e5f60002', 'web-prod-02', 'ql-base-ubuntu', '2.5.0', 'running', '{"role": "web", "tier": "frontend", "team": "platform"}', NOW() - INTERVAL '2 days', NOW()),

    -- API tier - older image (DRIFTED - 2 versions behind)
    ('aa000001-0001-0001-0001-000000000003', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'aws', '123456789012', 'us-east-1', 'US-East Production', '22222222-2222-2222-2222-222222222221', 'i-0a1b2c3d4e5f60003', 'api-prod-01', 'ql-base-ubuntu', '2.3.0', 'running', '{"role": "api", "tier": "backend", "team": "platform", "critical": true}', NOW() - INTERVAL '45 days', NOW() - INTERVAL '30 days'),
    ('aa000001-0001-0001-0001-000000000004', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'aws', '123456789012', 'us-east-1', 'US-East Production', '22222222-2222-2222-2222-222222222221', 'i-0a1b2c3d4e5f60004', 'api-prod-02', 'ql-base-ubuntu', '2.3.0', 'running', '{"role": "api", "tier": "backend", "team": "platform", "critical": true}', NOW() - INTERVAL '45 days', NOW() - INTERVAL '30 days'),

    -- Database tier - 1 version behind (DRIFTED)
    ('aa000001-0001-0001-0001-000000000005', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'aws', '123456789012', 'us-east-1', 'US-East Production', '22222222-2222-2222-2222-222222222221', 'i-0a1b2c3d4e5f60005', 'db-prod-01', 'ql-base-amazon', '1.7.0', 'running', '{"role": "database", "tier": "data", "team": "dba", "critical": true}', NOW() - INTERVAL '25 days', NOW() - INTERVAL '10 days'),

    -- Batch workers - compliant
    ('aa000001-0001-0001-0001-000000000006', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'aws', '123456789012', 'us-east-1', 'US-East Production', '22222222-2222-2222-2222-222222222221', 'i-0a1b2c3d4e5f60006', 'worker-prod-01', 'ql-base-amazon', '1.8.0', 'running', '{"role": "worker", "tier": "batch", "team": "platform"}', NOW() - INTERVAL '3 days', NOW()),
    ('aa000001-0001-0001-0001-000000000007', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'aws', '123456789012', 'us-east-1', 'US-East Production', '22222222-2222-2222-2222-222222222221', 'i-0a1b2c3d4e5f60007', 'worker-prod-02', 'ql-base-amazon', '1.8.0', 'running', '{"role": "worker", "tier": "batch", "team": "platform"}', NOW() - INTERVAL '3 days', NOW())
ON CONFLICT (org_id, platform, instance_id) DO UPDATE SET
    image_version = EXCLUDED.image_version,
    updated_at = EXCLUDED.updated_at;

-- =============================================================================
-- AWS DR Assets (US-West DR)
-- =============================================================================
INSERT INTO assets (id, org_id, env_id, platform, account, region, site, site_id, instance_id, name, image_ref, image_version, state, tags, discovered_at, updated_at) VALUES
    -- DR Web - DRIFTED from primary (different version)
    ('aa000002-0002-0002-0002-000000000001', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'aws', '123456789012', 'us-west-2', 'US-West DR', '22222222-2222-2222-2222-222222222222', 'i-0b2c3d4e5f600001', 'web-dr-01', 'ql-base-ubuntu', '2.4.0', 'stopped', '{"role": "web", "tier": "frontend", "dr": true}', NOW() - INTERVAL '20 days', NOW() - INTERVAL '15 days'),
    ('aa000002-0002-0002-0002-000000000002', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'aws', '123456789012', 'us-west-2', 'US-West DR', '22222222-2222-2222-2222-222222222222', 'i-0b2c3d4e5f600002', 'web-dr-02', 'ql-base-ubuntu', '2.4.0', 'stopped', '{"role": "web", "tier": "frontend", "dr": true}', NOW() - INTERVAL '20 days', NOW() - INTERVAL '15 days'),

    -- DR API - compliant (matches primary target)
    ('aa000002-0002-0002-0002-000000000003', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'aws', '123456789012', 'us-west-2', 'US-West DR', '22222222-2222-2222-2222-222222222222', 'i-0b2c3d4e5f600003', 'api-dr-01', 'ql-base-ubuntu', '2.5.0', 'stopped', '{"role": "api", "tier": "backend", "dr": true}', NOW() - INTERVAL '5 days', NOW()),

    -- DR Database
    ('aa000002-0002-0002-0002-000000000004', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'aws', '123456789012', 'us-west-2', 'US-West DR', '22222222-2222-2222-2222-222222222222', 'i-0b2c3d4e5f600004', 'db-dr-01', 'ql-base-amazon', '1.8.0', 'stopped', '{"role": "database", "tier": "data", "dr": true}', NOW() - INTERVAL '4 days', NOW())
ON CONFLICT (org_id, platform, instance_id) DO UPDATE SET
    image_version = EXCLUDED.image_version,
    updated_at = EXCLUDED.updated_at;

-- =============================================================================
-- Azure Assets
-- =============================================================================
INSERT INTO assets (id, org_id, env_id, platform, account, region, site, site_id, instance_id, name, image_ref, image_version, state, tags, discovered_at, updated_at) VALUES
    -- Azure East - Production Windows servers
    ('aa000003-0003-0003-0003-000000000001', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'azure', 'sub-azure-prod', 'eastus', 'Azure East US', '55555555-5555-5555-5555-555555555504', 'vm-az-iis-prod-01', 'iis-prod-01', 'ql-base-windows', '3.2.0', 'running', '{"role": "web", "tier": "frontend", "os": "windows"}', NOW() - INTERVAL '8 days', NOW()),
    ('aa000003-0003-0003-0003-000000000002', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'azure', 'sub-azure-prod', 'eastus', 'Azure East US', '55555555-5555-5555-5555-555555555504', 'vm-az-iis-prod-02', 'iis-prod-02', 'ql-base-windows', '3.2.0', 'running', '{"role": "web", "tier": "frontend", "os": "windows"}', NOW() - INTERVAL '8 days', NOW()),

    -- Azure East - DRIFTED Windows server (old version)
    ('aa000003-0003-0003-0003-000000000003', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'azure', 'sub-azure-prod', 'eastus', 'Azure East US', '55555555-5555-5555-5555-555555555504', 'vm-az-sql-prod-01', 'sql-prod-01', 'ql-base-windows', '3.1.0', 'running', '{"role": "database", "tier": "data", "os": "windows", "critical": true}', NOW() - INTERVAL '50 days', NOW() - INTERVAL '40 days'),

    -- Azure West Europe - Staging
    ('aa000003-0003-0003-0003-000000000004', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444402', 'azure', 'sub-azure-staging', 'westeurope', 'Azure West Europe', '55555555-5555-5555-5555-555555555505', 'vm-az-staging-01', 'staging-app-01', 'ql-base-windows', '3.2.0', 'running', '{"role": "app", "tier": "staging"}', NOW() - INTERVAL '3 days', NOW())
ON CONFLICT (org_id, platform, instance_id) DO UPDATE SET
    image_version = EXCLUDED.image_version,
    updated_at = EXCLUDED.updated_at;

-- =============================================================================
-- GCP Assets
-- =============================================================================
INSERT INTO assets (id, org_id, env_id, platform, account, region, site, site_id, instance_id, name, image_ref, image_version, state, tags, discovered_at, updated_at) VALUES
    -- GCP Analytics cluster - compliant
    ('aa000004-0004-0004-0004-000000000001', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444404', 'gcp', 'prj-analytics-prod', 'us-central1', 'GCP US Central', '55555555-5555-5555-5555-555555555506', 'gce-analytics-01', 'analytics-01', 'ql-base-ubuntu', '2.5.0', 'running', '{"role": "analytics", "tier": "data", "team": "data-science"}', NOW() - INTERVAL '5 days', NOW()),
    ('aa000004-0004-0004-0004-000000000002', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444404', 'gcp', 'prj-analytics-prod', 'us-central1', 'GCP US Central', '55555555-5555-5555-5555-555555555506', 'gce-analytics-02', 'analytics-02', 'ql-base-ubuntu', '2.5.0', 'running', '{"role": "analytics", "tier": "data", "team": "data-science"}', NOW() - INTERVAL '5 days', NOW()),

    -- GCP ML workers - DRIFTED
    ('aa000004-0004-0004-0004-000000000003', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444404', 'gcp', 'prj-analytics-prod', 'us-central1', 'GCP US Central', '55555555-5555-5555-5555-555555555506', 'gce-ml-worker-01', 'ml-worker-01', 'ql-base-ubuntu', '2.4.0', 'running', '{"role": "ml", "tier": "compute", "gpu": true, "critical": true}', NOW() - INTERVAL '18 days', NOW() - INTERVAL '12 days'),
    ('aa000004-0004-0004-0004-000000000004', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444404', 'gcp', 'prj-analytics-prod', 'us-central1', 'GCP US Central', '55555555-5555-5555-5555-555555555506', 'gce-ml-worker-02', 'ml-worker-02', 'ql-base-ubuntu', '2.4.0', 'running', '{"role": "ml", "tier": "compute", "gpu": true, "critical": true}', NOW() - INTERVAL '18 days', NOW() - INTERVAL '12 days')
ON CONFLICT (org_id, platform, instance_id) DO UPDATE SET
    image_version = EXCLUDED.image_version,
    updated_at = EXCLUDED.updated_at;

-- =============================================================================
-- vSphere (On-Prem) Assets
-- =============================================================================
INSERT INTO assets (id, org_id, env_id, platform, account, region, site, site_id, instance_id, name, image_ref, image_version, state, tags, discovered_at, updated_at) VALUES
    -- Legacy app servers - SEVERELY DRIFTED (very old)
    ('aa000005-0005-0005-0005-000000000001', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'vsphere', 'vcenter-london', 'uk-lon-01', 'DC London', '55555555-5555-5555-5555-555555555507', 'vm-legacy-erp-01', 'legacy-erp-01', 'ql-base-ubuntu', '2.3.0', 'running', '{"role": "erp", "tier": "legacy", "migration_planned": true, "critical": true}', NOW() - INTERVAL '90 days', NOW() - INTERVAL '60 days'),
    ('aa000005-0005-0005-0005-000000000002', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'vsphere', 'vcenter-london', 'uk-lon-01', 'DC London', '55555555-5555-5555-5555-555555555507', 'vm-legacy-erp-02', 'legacy-erp-02', 'ql-base-ubuntu', '2.3.0', 'running', '{"role": "erp", "tier": "legacy", "migration_planned": true, "critical": true}', NOW() - INTERVAL '90 days', NOW() - INTERVAL '60 days'),

    -- File servers - compliant
    ('aa000005-0005-0005-0005-000000000003', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'vsphere', 'vcenter-london', 'uk-lon-01', 'DC London', '55555555-5555-5555-5555-555555555507', 'vm-fileserver-01', 'fileserver-01', 'ql-base-windows', '3.2.0', 'running', '{"role": "storage", "tier": "shared"}', NOW() - INTERVAL '6 days', NOW())
ON CONFLICT (org_id, platform, instance_id) DO UPDATE SET
    image_version = EXCLUDED.image_version,
    updated_at = EXCLUDED.updated_at;

-- =============================================================================
-- Kubernetes Assets (Containers)
-- =============================================================================
INSERT INTO assets (id, org_id, env_id, platform, account, region, site, site_id, instance_id, name, image_ref, image_version, state, tags, discovered_at, updated_at) VALUES
    -- K8s API deployments - compliant
    ('aa000006-0006-0006-0006-000000000001', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'k8s', 'eks-prod-cluster', 'eks-prod', 'K8s Production', '55555555-5555-5555-5555-555555555508', 'deploy/api-gateway', 'api-gateway', 'ql-app-api', '4.12.0', 'running', '{"role": "gateway", "replicas": 3, "namespace": "production"}', NOW() - INTERVAL '1 day', NOW()),
    ('aa000006-0006-0006-0006-000000000002', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'k8s', 'eks-prod-cluster', 'eks-prod', 'K8s Production', '55555555-5555-5555-5555-555555555508', 'deploy/user-service', 'user-service', 'ql-app-api', '4.12.0', 'running', '{"role": "service", "replicas": 2, "namespace": "production"}', NOW() - INTERVAL '1 day', NOW()),
    ('aa000006-0006-0006-0006-000000000003', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'k8s', 'eks-prod-cluster', 'eks-prod', 'K8s Production', '55555555-5555-5555-5555-555555555508', 'deploy/order-service', 'order-service', 'ql-app-api', '4.12.0', 'running', '{"role": "service", "replicas": 2, "namespace": "production", "critical": true}', NOW() - INTERVAL '1 day', NOW()),

    -- K8s Workers - DRIFTED (1 version behind)
    ('aa000006-0006-0006-0006-000000000004', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'k8s', 'eks-prod-cluster', 'eks-prod', 'K8s Production', '55555555-5555-5555-5555-555555555508', 'deploy/background-worker', 'background-worker', 'ql-app-worker', '2.7.0', 'running', '{"role": "worker", "replicas": 5, "namespace": "production"}', NOW() - INTERVAL '16 days', NOW() - INTERVAL '10 days'),
    ('aa000006-0006-0006-0006-000000000005', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', 'k8s', 'eks-prod-cluster', 'eks-prod', 'K8s Production', '55555555-5555-5555-5555-555555555508', 'deploy/notification-worker', 'notification-worker', 'ql-app-worker', '2.7.0', 'running', '{"role": "worker", "replicas": 2, "namespace": "production"}', NOW() - INTERVAL '16 days', NOW() - INTERVAL '10 days')
ON CONFLICT (org_id, platform, instance_id) DO UPDATE SET
    image_version = EXCLUDED.image_version,
    updated_at = EXCLUDED.updated_at;

-- =============================================================================
-- DR Pairs
-- =============================================================================
INSERT INTO dr_pairs (id, org_id, name, primary_site_id, dr_site_id, status, replication_status, rpo, rto, last_failover_test, last_sync_at) VALUES
    ('66666666-6666-6666-6666-666666666601', '11111111-1111-1111-1111-111111111111', 'US East-West DR', '22222222-2222-2222-2222-222222222221', '22222222-2222-2222-2222-222222222222', 'healthy', 'in-sync', '15 min', '4 hours', NOW() - INTERVAL '30 days', NOW() - INTERVAL '5 minutes'),
    ('66666666-6666-6666-6666-666666666602', '11111111-1111-1111-1111-111111111111', 'EU-Azure Cross-Region', '55555555-5555-5555-5555-555555555503', '55555555-5555-5555-5555-555555555505', 'warning', 'lagging', '30 min', '8 hours', NOW() - INTERVAL '90 days', NOW() - INTERVAL '2 hours'),
    ('66666666-6666-6666-6666-666666666603', '11111111-1111-1111-1111-111111111111', 'Azure Primary-Secondary', '55555555-5555-5555-5555-555555555504', '55555555-5555-5555-5555-555555555505', 'healthy', 'in-sync', '5 min', '2 hours', NOW() - INTERVAL '14 days', NOW() - INTERVAL '1 minute')
ON CONFLICT (org_id, name) DO UPDATE SET
    status = EXCLUDED.status,
    replication_status = EXCLUDED.replication_status,
    last_sync_at = EXCLUDED.last_sync_at;

-- =============================================================================
-- Alerts (Active issues)
-- =============================================================================
INSERT INTO alerts (id, org_id, site_id, asset_id, severity, type, title, description, status, created_at) VALUES
    -- Critical drift alert
    ('bb000001-0001-0001-0001-000000000001', '11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222221', 'aa000001-0001-0001-0001-000000000003', 'critical', 'drift', 'Critical Drift Detected: API Server', 'api-prod-01 is 2 versions behind golden image (2.3.0 vs 2.5.0). Contains critical security patches.', 'open', NOW() - INTERVAL '30 days'),
    ('bb000001-0001-0001-0001-000000000002', '11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222221', 'aa000001-0001-0001-0001-000000000004', 'critical', 'drift', 'Critical Drift Detected: API Server', 'api-prod-02 is 2 versions behind golden image (2.3.0 vs 2.5.0). Contains critical security patches.', 'open', NOW() - INTERVAL '30 days'),

    -- High severity DR drift
    ('bb000001-0001-0001-0001-000000000003', '11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', 'aa000002-0002-0002-0002-000000000001', 'high', 'drift', 'DR Site Image Mismatch', 'DR web servers running different image version than production. Failover may cause inconsistencies.', 'open', NOW() - INTERVAL '15 days'),

    -- Windows server drift
    ('bb000001-0001-0001-0001-000000000004', '11111111-1111-1111-1111-111111111111', '55555555-5555-5555-5555-555555555504', 'aa000003-0003-0003-0003-000000000003', 'high', 'drift', 'SQL Server Image Outdated', 'sql-prod-01 running Windows Server image 3.1.0, missing February security patches.', 'open', NOW() - INTERVAL '40 days'),

    -- Legacy system warning
    ('bb000001-0001-0001-0001-000000000005', '11111111-1111-1111-1111-111111111111', '55555555-5555-5555-5555-555555555507', 'aa000005-0005-0005-0005-000000000001', 'critical', 'compliance', 'Legacy ERP Severely Outdated', 'legacy-erp-01 has not been patched in 90 days. CIS Level 2 compliance at risk.', 'open', NOW() - INTERVAL '60 days'),

    -- K8s worker drift
    ('bb000001-0001-0001-0001-000000000006', '11111111-1111-1111-1111-111111111111', '55555555-5555-5555-5555-555555555508', 'aa000006-0006-0006-0006-000000000004', 'medium', 'drift', 'Worker Deployment Behind', 'background-worker running ql-app-worker:2.7.0, latest is 2.8.0.', 'open', NOW() - INTERVAL '10 days'),

    -- DR test overdue warning
    ('bb000001-0001-0001-0001-000000000007', '11111111-1111-1111-1111-111111111111', '55555555-5555-5555-5555-555555555503', NULL, 'warning', 'dr', 'DR Failover Test Overdue', 'EU-Azure Cross-Region DR has not been tested in 90 days. Compliance requires quarterly testing.', 'open', NOW() - INTERVAL '7 days')
ON CONFLICT DO NOTHING;

-- =============================================================================
-- Activities (Recent actions)
-- =============================================================================
INSERT INTO activities (id, org_id, site_id, asset_id, type, description, user_id, created_at) VALUES
    -- Recent patching activities
    ('cc000001-0001-0001-0001-000000000001', '11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222221', 'aa000001-0001-0001-0001-000000000001', 'patch', 'Upgraded web-prod-01 from ql-base-ubuntu:2.4.0 to 2.5.0', '22222222-2222-2222-2222-222222222202', NOW() - INTERVAL '2 days'),
    ('cc000001-0001-0001-0001-000000000002', '11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222221', 'aa000001-0001-0001-0001-000000000002', 'patch', 'Upgraded web-prod-02 from ql-base-ubuntu:2.4.0 to 2.5.0', '22222222-2222-2222-2222-222222222202', NOW() - INTERVAL '2 days'),

    -- Image publication
    ('cc000001-0001-0001-0001-000000000003', '11111111-1111-1111-1111-111111111111', NULL, NULL, 'image', 'Published new golden image ql-base-ubuntu:2.5.0 to all regions', '22222222-2222-2222-2222-222222222201', NOW() - INTERVAL '7 days'),
    ('cc000001-0001-0001-0001-000000000004', '11111111-1111-1111-1111-111111111111', NULL, NULL, 'image', 'Published new golden image ql-app-api:4.12.0', '22222222-2222-2222-2222-222222222201', NOW() - INTERVAL '2 days'),

    -- DR test activity
    ('cc000001-0001-0001-0001-000000000005', '11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', NULL, 'dr_test', 'Successfully completed DR failover test for US East-West pair. RTO achieved: 3h 42m', '22222222-2222-2222-2222-222222222201', NOW() - INTERVAL '30 days'),

    -- Compliance scan
    ('cc000001-0001-0001-0001-000000000006', '11111111-1111-1111-1111-111111111111', NULL, NULL, 'compliance', 'Completed CIS Level 2 compliance scan. 94% of assets compliant.', '22222222-2222-2222-2222-222222222202', NOW() - INTERVAL '1 day'),

    -- AI task activities
    ('cc000001-0001-0001-0001-000000000007', '11111111-1111-1111-1111-111111111111', NULL, NULL, 'ai_task', 'AI Copilot generated patch plan for 8 drifted assets', NULL, NOW() - INTERVAL '12 hours'),
    ('cc000001-0001-0001-0001-000000000008', '11111111-1111-1111-1111-111111111111', NULL, NULL, 'ai_task', 'Operator approved AI-generated patching plan', '22222222-2222-2222-2222-222222222202', NOW() - INTERVAL '10 hours')
ON CONFLICT DO NOTHING;

-- =============================================================================
-- Summary Stats
-- =============================================================================
SELECT
    'Assets' as entity,
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE image_version != (
        SELECT MAX(i.version) FROM images i
        WHERE i.family = assets.image_ref AND i.org_id = assets.org_id
    )) as drifted
FROM assets
WHERE org_id = '11111111-1111-1111-1111-111111111111'
UNION ALL
SELECT 'Alerts', COUNT(*), COUNT(*) FILTER (WHERE status = 'open') FROM alerts WHERE org_id = '11111111-1111-1111-1111-111111111111'
UNION ALL
SELECT 'DR Pairs', COUNT(*), COUNT(*) FILTER (WHERE status != 'healthy') FROM dr_pairs WHERE org_id = '11111111-1111-1111-1111-111111111111'
UNION ALL
SELECT 'Sites', COUNT(*), 0 FROM sites WHERE org_id = '11111111-1111-1111-1111-111111111111'
UNION ALL
SELECT 'Images', COUNT(*), COUNT(*) FILTER (WHERE status = 'deprecated') FROM images WHERE org_id = '11111111-1111-1111-1111-111111111111';
