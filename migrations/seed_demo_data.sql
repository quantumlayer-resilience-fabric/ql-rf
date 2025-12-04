-- QuantumLayer Resilience Fabric - Demo Seed Data
-- This script creates realistic demo data for showcasing the platform
-- Run with: psql -d qlrf -f seed_demo_data.sql

-- =============================================================================
-- Organization & Users
-- =============================================================================

-- Demo organization
INSERT INTO organizations (id, name, slug) VALUES
    ('11111111-1111-1111-1111-111111111111', 'Acme Corporation', 'acme')
ON CONFLICT (slug) DO NOTHING;

-- Demo users
INSERT INTO users (id, external_id, email, name, role, org_id) VALUES
    ('22222222-2222-2222-2222-222222222201', 'user_demo_admin', 'admin@acme.example', 'Alice Admin', 'admin', '11111111-1111-1111-1111-111111111111'),
    ('22222222-2222-2222-2222-222222222202', 'user_demo_ops', 'ops@acme.example', 'Bob Operations', 'operator', '11111111-1111-1111-1111-111111111111'),
    ('22222222-2222-2222-2222-222222222203', 'user_demo_viewer', 'viewer@acme.example', 'Carol Viewer', 'viewer', '11111111-1111-1111-1111-111111111111')
ON CONFLICT (external_id) DO NOTHING;

-- =============================================================================
-- Projects & Environments
-- =============================================================================

INSERT INTO projects (id, org_id, name, slug) VALUES
    ('33333333-3333-3333-3333-333333333301', '11111111-1111-1111-1111-111111111111', 'E-Commerce Platform', 'ecommerce'),
    ('33333333-3333-3333-3333-333333333302', '11111111-1111-1111-1111-111111111111', 'Data Analytics', 'analytics'),
    ('33333333-3333-3333-3333-333333333303', '11111111-1111-1111-1111-111111111111', 'Internal Tools', 'internal')
ON CONFLICT (org_id, slug) DO NOTHING;

INSERT INTO environments (id, project_id, name) VALUES
    ('44444444-4444-4444-4444-444444444401', '33333333-3333-3333-3333-333333333301', 'production'),
    ('44444444-4444-4444-4444-444444444402', '33333333-3333-3333-3333-333333333301', 'staging'),
    ('44444444-4444-4444-4444-444444444403', '33333333-3333-3333-3333-333333333301', 'development'),
    ('44444444-4444-4444-4444-444444444404', '33333333-3333-3333-3333-333333333302', 'production'),
    ('44444444-4444-4444-4444-444444444405', '33333333-3333-3333-3333-333333333302', 'staging')
ON CONFLICT (project_id, name) DO NOTHING;

-- =============================================================================
-- Sites (Data Centers / Regions)
-- =============================================================================

INSERT INTO sites (id, org_id, name, region, platform, environment, metadata) VALUES
    ('55555555-5555-5555-5555-555555555501', '11111111-1111-1111-1111-111111111111', 'US-East Production', 'us-east-1', 'aws', 'production', '{"tier": "primary", "cost_center": "CC-001"}'),
    ('55555555-5555-5555-5555-555555555502', '11111111-1111-1111-1111-111111111111', 'US-West DR', 'us-west-2', 'aws', 'production', '{"tier": "dr", "cost_center": "CC-001"}'),
    ('55555555-5555-5555-5555-555555555503', '11111111-1111-1111-1111-111111111111', 'EU-West Production', 'eu-west-1', 'aws', 'production', '{"tier": "primary", "cost_center": "CC-002"}'),
    ('55555555-5555-5555-5555-555555555504', '11111111-1111-1111-1111-111111111111', 'Azure East US', 'eastus', 'azure', 'production', '{"tier": "primary", "cost_center": "CC-003"}'),
    ('55555555-5555-5555-5555-555555555505', '11111111-1111-1111-1111-111111111111', 'Azure West Europe', 'westeurope', 'azure', 'staging', '{"tier": "staging", "cost_center": "CC-003"}'),
    ('55555555-5555-5555-5555-555555555506', '11111111-1111-1111-1111-111111111111', 'GCP US Central', 'us-central1', 'gcp', 'production', '{"tier": "analytics", "cost_center": "CC-004"}'),
    ('55555555-5555-5555-5555-555555555507', '11111111-1111-1111-1111-111111111111', 'DC London', 'uk-lon-01', 'vsphere', 'production', '{"tier": "legacy", "cost_center": "CC-005"}'),
    ('55555555-5555-5555-5555-555555555508', '11111111-1111-1111-1111-111111111111', 'K8s Production', 'eks-prod', 'k8s', 'production', '{"tier": "containers", "cost_center": "CC-006"}')
ON CONFLICT (org_id, name) DO NOTHING;

-- Update DR paired sites
UPDATE sites SET dr_paired_site_id = '55555555-5555-5555-5555-555555555502' WHERE id = '55555555-5555-5555-5555-555555555501';
UPDATE sites SET dr_paired_site_id = '55555555-5555-5555-5555-555555555501' WHERE id = '55555555-5555-5555-5555-555555555502';

-- =============================================================================
-- DR Pairs
-- =============================================================================

INSERT INTO dr_pairs (id, org_id, name, primary_site_id, dr_site_id, status, replication_status, rpo, rto, last_failover_test, last_sync_at) VALUES
    ('66666666-6666-6666-6666-666666666601', '11111111-1111-1111-1111-111111111111', 'US East-West DR', '55555555-5555-5555-5555-555555555501', '55555555-5555-5555-5555-555555555502', 'healthy', 'in-sync', '15 min', '4 hours', NOW() - INTERVAL '30 days', NOW() - INTERVAL '5 minutes'),
    ('66666666-6666-6666-6666-666666666602', '11111111-1111-1111-1111-111111111111', 'EU Primary DR', '55555555-5555-5555-5555-555555555503', '55555555-5555-5555-5555-555555555505', 'warning', 'lagging', '30 min', '8 hours', NOW() - INTERVAL '90 days', NOW() - INTERVAL '2 hours'),
    ('66666666-6666-6666-6666-666666666603', '11111111-1111-1111-1111-111111111111', 'Azure Multi-Region', '55555555-5555-5555-5555-555555555504', '55555555-5555-5555-5555-555555555505', 'healthy', 'in-sync', '5 min', '2 hours', NOW() - INTERVAL '14 days', NOW() - INTERVAL '1 minute')
ON CONFLICT (org_id, name) DO NOTHING;

-- =============================================================================
-- Golden Images
-- =============================================================================

INSERT INTO images (id, org_id, family, version, os_name, os_version, cis_level, sbom_url, signed, status, created_at) VALUES
    -- Ubuntu base images
    ('77777777-7777-7777-7777-777777777701', '11111111-1111-1111-1111-111111111111', 'ql-base-ubuntu', '2.5.0', 'Ubuntu', '22.04 LTS', 2, 'https://sbom.example.com/ubuntu-2.5.0', true, 'production', NOW() - INTERVAL '7 days'),
    ('77777777-7777-7777-7777-777777777702', '11111111-1111-1111-1111-111111111111', 'ql-base-ubuntu', '2.4.0', 'Ubuntu', '22.04 LTS', 2, 'https://sbom.example.com/ubuntu-2.4.0', true, 'deprecated', NOW() - INTERVAL '30 days'),
    ('77777777-7777-7777-7777-777777777703', '11111111-1111-1111-1111-111111111111', 'ql-base-ubuntu', '2.3.0', 'Ubuntu', '22.04 LTS', 1, 'https://sbom.example.com/ubuntu-2.3.0', true, 'deprecated', NOW() - INTERVAL '60 days'),

    -- Amazon Linux images
    ('77777777-7777-7777-7777-777777777704', '11111111-1111-1111-1111-111111111111', 'ql-base-amazon', '1.8.0', 'Amazon Linux', '2023', 2, 'https://sbom.example.com/amazon-1.8.0', true, 'production', NOW() - INTERVAL '5 days'),
    ('77777777-7777-7777-7777-777777777705', '11111111-1111-1111-1111-111111111111', 'ql-base-amazon', '1.7.0', 'Amazon Linux', '2023', 2, 'https://sbom.example.com/amazon-1.7.0', true, 'deprecated', NOW() - INTERVAL '21 days'),

    -- Windows images
    ('77777777-7777-7777-7777-777777777706', '11111111-1111-1111-1111-111111111111', 'ql-base-windows', '3.2.0', 'Windows Server', '2022', 2, 'https://sbom.example.com/windows-3.2.0', true, 'production', NOW() - INTERVAL '10 days'),
    ('77777777-7777-7777-7777-777777777707', '11111111-1111-1111-1111-111111111111', 'ql-base-windows', '3.1.0', 'Windows Server', '2022', 2, 'https://sbom.example.com/windows-3.1.0', true, 'deprecated', NOW() - INTERVAL '45 days'),

    -- Container images
    ('77777777-7777-7777-7777-777777777708', '11111111-1111-1111-1111-111111111111', 'ql-app-api', '4.12.0', 'Alpine', '3.18', 2, 'https://sbom.example.com/api-4.12.0', true, 'production', NOW() - INTERVAL '2 days'),
    ('77777777-7777-7777-7777-777777777709', '11111111-1111-1111-1111-111111111111', 'ql-app-api', '4.11.0', 'Alpine', '3.18', 2, 'https://sbom.example.com/api-4.11.0', true, 'deprecated', NOW() - INTERVAL '14 days'),
    ('77777777-7777-7777-7777-777777777710', '11111111-1111-1111-1111-111111111111', 'ql-app-worker', '2.8.0', 'Alpine', '3.18', 2, 'https://sbom.example.com/worker-2.8.0', true, 'production', NOW() - INTERVAL '3 days')
ON CONFLICT (org_id, family, version) DO NOTHING;

-- Image coordinates (platform-specific identifiers)
INSERT INTO image_coordinates (id, image_id, platform, region, identifier) VALUES
    -- Ubuntu 2.5.0 across platforms
    ('88888888-8888-8888-8888-888888888801', '77777777-7777-7777-7777-777777777701', 'aws', 'us-east-1', 'ami-ubuntu250-east1'),
    ('88888888-8888-8888-8888-888888888802', '77777777-7777-7777-7777-777777777701', 'aws', 'us-west-2', 'ami-ubuntu250-west2'),
    ('88888888-8888-8888-8888-888888888803', '77777777-7777-7777-7777-777777777701', 'aws', 'eu-west-1', 'ami-ubuntu250-euwest1'),
    ('88888888-8888-8888-8888-888888888804', '77777777-7777-7777-7777-777777777701', 'azure', 'eastus', '/subscriptions/xxx/images/ql-ubuntu-2.5.0'),
    ('88888888-8888-8888-8888-888888888805', '77777777-7777-7777-7777-777777777701', 'gcp', 'us-central1', 'projects/acme/global/images/ql-ubuntu-2-5-0'),

    -- Amazon Linux 1.8.0
    ('88888888-8888-8888-8888-888888888806', '77777777-7777-7777-7777-777777777704', 'aws', 'us-east-1', 'ami-amazon180-east1'),
    ('88888888-8888-8888-8888-888888888807', '77777777-7777-7777-7777-777777777704', 'aws', 'us-west-2', 'ami-amazon180-west2'),

    -- Windows 3.2.0
    ('88888888-8888-8888-8888-888888888808', '77777777-7777-7777-7777-777777777706', 'aws', 'us-east-1', 'ami-windows320-east1'),
    ('88888888-8888-8888-8888-888888888809', '77777777-7777-7777-7777-777777777706', 'azure', 'eastus', '/subscriptions/xxx/images/ql-windows-3.2.0'),

    -- Container images
    ('88888888-8888-8888-8888-888888888810', '77777777-7777-7777-7777-777777777708', 'k8s', 'eks-prod', 'acme.azurecr.io/ql-app-api:4.12.0'),
    ('88888888-8888-8888-8888-888888888811', '77777777-7777-7777-7777-777777777710', 'k8s', 'eks-prod', 'acme.azurecr.io/ql-app-worker:2.8.0')
ON CONFLICT (image_id, platform, region) DO NOTHING;

-- =============================================================================
-- Assets (Fleet Inventory) - Mix of compliant and drifted
-- =============================================================================

INSERT INTO assets (id, org_id, env_id, site_id, platform, account, region, instance_id, name, image_ref, image_version, state, tags, discovered_at, updated_at) VALUES
    -- US-East Production (mostly compliant)
    ('99999999-9999-9999-9999-999999999901', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', '55555555-5555-5555-5555-555555555501', 'aws', '123456789012', 'us-east-1', 'i-prod-web-001', 'prod-web-001', 'ql-base-ubuntu', '2.5.0', 'running', '{"role": "web", "tier": "frontend"}', NOW() - INTERVAL '90 days', NOW() - INTERVAL '1 day'),
    ('99999999-9999-9999-9999-999999999902', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', '55555555-5555-5555-5555-555555555501', 'aws', '123456789012', 'us-east-1', 'i-prod-web-002', 'prod-web-002', 'ql-base-ubuntu', '2.5.0', 'running', '{"role": "web", "tier": "frontend"}', NOW() - INTERVAL '90 days', NOW() - INTERVAL '1 day'),
    ('99999999-9999-9999-9999-999999999903', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', '55555555-5555-5555-5555-555555555501', 'aws', '123456789012', 'us-east-1', 'i-prod-api-001', 'prod-api-001', 'ql-base-amazon', '1.8.0', 'running', '{"role": "api", "tier": "backend"}', NOW() - INTERVAL '60 days', NOW() - INTERVAL '2 days'),
    ('99999999-9999-9999-9999-999999999904', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', '55555555-5555-5555-5555-555555555501', 'aws', '123456789012', 'us-east-1', 'i-prod-api-002', 'prod-api-002', 'ql-base-amazon', '1.8.0', 'running', '{"role": "api", "tier": "backend"}', NOW() - INTERVAL '60 days', NOW() - INTERVAL '2 days'),

    -- US-East DRIFTED assets
    ('99999999-9999-9999-9999-999999999905', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', '55555555-5555-5555-5555-555555555501', 'aws', '123456789012', 'us-east-1', 'i-prod-db-001', 'prod-db-001', 'ql-base-ubuntu', '2.4.0', 'running', '{"role": "database", "tier": "data", "critical": true}', NOW() - INTERVAL '120 days', NOW() - INTERVAL '25 days'),
    ('99999999-9999-9999-9999-999999999906', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', '55555555-5555-5555-5555-555555555501', 'aws', '123456789012', 'us-east-1', 'i-prod-cache-001', 'prod-cache-001', 'ql-base-amazon', '1.7.0', 'running', '{"role": "cache", "tier": "data"}', NOW() - INTERVAL '45 days', NOW() - INTERVAL '18 days'),

    -- US-West DR Site
    ('99999999-9999-9999-9999-999999999907', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', '55555555-5555-5555-5555-555555555502', 'aws', '123456789012', 'us-west-2', 'i-dr-web-001', 'dr-web-001', 'ql-base-ubuntu', '2.5.0', 'running', '{"role": "web", "dr": true}', NOW() - INTERVAL '90 days', NOW() - INTERVAL '1 day'),
    ('99999999-9999-9999-9999-999999999908', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', '55555555-5555-5555-5555-555555555502', 'aws', '123456789012', 'us-west-2', 'i-dr-api-001', 'dr-api-001', 'ql-base-amazon', '1.8.0', 'running', '{"role": "api", "dr": true}', NOW() - INTERVAL '60 days', NOW() - INTERVAL '2 days'),

    -- EU-West Production (with drift)
    ('99999999-9999-9999-9999-999999999909', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', '55555555-5555-5555-5555-555555555503', 'aws', '987654321098', 'eu-west-1', 'i-eu-web-001', 'eu-web-001', 'ql-base-ubuntu', '2.5.0', 'running', '{"role": "web", "region": "eu"}', NOW() - INTERVAL '60 days', NOW() - INTERVAL '3 days'),
    ('99999999-9999-9999-9999-999999999910', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', '55555555-5555-5555-5555-555555555503', 'aws', '987654321098', 'eu-west-1', 'i-eu-web-002', 'eu-web-002', 'ql-base-ubuntu', '2.3.0', 'running', '{"role": "web", "region": "eu"}', NOW() - INTERVAL '90 days', NOW() - INTERVAL '55 days'),

    -- Azure assets
    ('99999999-9999-9999-9999-999999999911', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', '55555555-5555-5555-5555-555555555504', 'azure', 'sub-12345', 'eastus', 'vm-azure-web-001', 'azure-web-001', 'ql-base-ubuntu', '2.5.0', 'running', '{"role": "web", "cloud": "azure"}', NOW() - INTERVAL '45 days', NOW() - INTERVAL '2 days'),
    ('99999999-9999-9999-9999-999999999912', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', '55555555-5555-5555-5555-555555555504', 'azure', 'sub-12345', 'eastus', 'vm-azure-api-001', 'azure-api-001', 'ql-base-windows', '3.2.0', 'running', '{"role": "api", "cloud": "azure"}', NOW() - INTERVAL '30 days', NOW() - INTERVAL '5 days'),
    ('99999999-9999-9999-9999-999999999913', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', '55555555-5555-5555-5555-555555555504', 'azure', 'sub-12345', 'eastus', 'vm-azure-db-001', 'azure-db-001', 'ql-base-windows', '3.1.0', 'running', '{"role": "database", "cloud": "azure", "critical": true}', NOW() - INTERVAL '60 days', NOW() - INTERVAL '40 days'),

    -- GCP assets
    ('99999999-9999-9999-9999-999999999914', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444404', '55555555-5555-5555-5555-555555555506', 'gcp', 'acme-analytics', 'us-central1', 'gce-analytics-001', 'gcp-analytics-001', 'ql-base-ubuntu', '2.5.0', 'running', '{"role": "analytics", "cloud": "gcp"}', NOW() - INTERVAL '30 days', NOW() - INTERVAL '1 day'),
    ('99999999-9999-9999-9999-999999999915', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444404', '55555555-5555-5555-5555-555555555506', 'gcp', 'acme-analytics', 'us-central1', 'gce-analytics-002', 'gcp-analytics-002', 'ql-base-ubuntu', '2.4.0', 'running', '{"role": "analytics", "cloud": "gcp"}', NOW() - INTERVAL '45 days', NOW() - INTERVAL '12 days'),

    -- vSphere assets (legacy DC)
    ('99999999-9999-9999-9999-999999999916', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', '55555555-5555-5555-5555-555555555507', 'vsphere', 'vcenter-lon', 'uk-lon-01', 'vm-legacy-erp-001', 'legacy-erp-001', 'ql-base-windows', '3.2.0', 'running', '{"role": "erp", "legacy": true}', NOW() - INTERVAL '180 days', NOW() - INTERVAL '5 days'),
    ('99999999-9999-9999-9999-999999999917', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', '55555555-5555-5555-5555-555555555507', 'vsphere', 'vcenter-lon', 'uk-lon-01', 'vm-legacy-erp-002', 'legacy-erp-002', 'ql-base-windows', '3.1.0', 'running', '{"role": "erp", "legacy": true, "critical": true}', NOW() - INTERVAL '180 days', NOW() - INTERVAL '42 days'),

    -- Kubernetes workloads
    ('99999999-9999-9999-9999-999999999918', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', '55555555-5555-5555-5555-555555555508', 'k8s', 'eks-prod-cluster', 'eks-prod', 'deploy/api-deployment', 'k8s-api', 'ql-app-api', '4.12.0', 'running', '{"role": "api", "replicas": 3}', NOW() - INTERVAL '14 days', NOW() - INTERVAL '1 day'),
    ('99999999-9999-9999-9999-999999999919', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', '55555555-5555-5555-5555-555555555508', 'k8s', 'eks-prod-cluster', 'eks-prod', 'deploy/worker-deployment', 'k8s-worker', 'ql-app-worker', '2.8.0', 'running', '{"role": "worker", "replicas": 5}', NOW() - INTERVAL '14 days', NOW() - INTERVAL '2 days'),
    ('99999999-9999-9999-9999-999999999920', '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444401', '55555555-5555-5555-5555-555555555508', 'k8s', 'eks-prod-cluster', 'eks-prod', 'deploy/frontend-deployment', 'k8s-frontend', 'ql-app-api', '4.11.0', 'running', '{"role": "frontend", "replicas": 2}', NOW() - INTERVAL '21 days', NOW() - INTERVAL '10 days')
ON CONFLICT (org_id, platform, instance_id) DO NOTHING;

-- =============================================================================
-- Compliance Frameworks & Controls
-- =============================================================================

INSERT INTO compliance_frameworks (id, org_id, name, description, level, enabled) VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa001', '11111111-1111-1111-1111-111111111111', 'CIS', 'CIS Benchmarks for hardened images', 2, true),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa002', '11111111-1111-1111-1111-111111111111', 'SLSA', 'Supply-chain Levels for Software Artifacts', 3, true),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa003', '11111111-1111-1111-1111-111111111111', 'SOC2', 'SOC 2 Type II Controls', NULL, true),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa004', '11111111-1111-1111-1111-111111111111', 'HIPAA', 'HIPAA Security Controls', NULL, true),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa005', '11111111-1111-1111-1111-111111111111', 'PCI-DSS', 'Payment Card Industry Data Security Standard', NULL, true)
ON CONFLICT (org_id, name) DO NOTHING;

-- CIS Controls
INSERT INTO compliance_controls (id, framework_id, control_id, title, description, severity, recommendation) VALUES
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb001', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa001', 'CIS-1.1', 'Ensure root login is disabled', 'SSH root login should be disabled for security', 'high', 'Set PermitRootLogin no in /etc/ssh/sshd_config'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb002', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa001', 'CIS-1.2', 'Ensure password authentication is disabled', 'SSH should use key-based auth only', 'high', 'Set PasswordAuthentication no in /etc/ssh/sshd_config'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb003', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa001', 'CIS-2.1', 'Ensure firewall is active', 'Host firewall must be enabled', 'medium', 'Enable ufw or iptables'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb004', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa001', 'CIS-3.1', 'Ensure audit logging is enabled', 'System audit logs must be configured', 'medium', 'Enable auditd service'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb005', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa001', 'CIS-4.1', 'Ensure automatic updates are configured', 'Security updates should be automatic', 'low', 'Configure unattended-upgrades')
ON CONFLICT (framework_id, control_id) DO NOTHING;

-- SLSA Controls
INSERT INTO compliance_controls (id, framework_id, control_id, title, description, severity, recommendation) VALUES
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb010', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa002', 'SLSA-L1', 'Build process documented', 'Build process must be documented', 'low', 'Document build steps in CI/CD'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb011', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa002', 'SLSA-L2', 'Signed provenance', 'Builds must generate signed provenance', 'medium', 'Use cosign to sign artifacts'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb012', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa002', 'SLSA-L3', 'Hardened builds', 'Builds must be on hardened infrastructure', 'high', 'Use ephemeral build agents'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb013', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa002', 'SLSA-L4', 'Two-party review', 'All changes require two-party review', 'high', 'Enforce PR approvals')
ON CONFLICT (framework_id, control_id) DO NOTHING;

-- Compliance Results (mix of passing/failing)
INSERT INTO compliance_results (id, org_id, framework_id, control_id, status, affected_assets, score, last_audit_at) VALUES
    -- CIS Results
    ('cccccccc-cccc-cccc-cccc-ccccccccc001', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa001', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb001', 'passing', 0, 100.00, NOW() - INTERVAL '1 day'),
    ('cccccccc-cccc-cccc-cccc-ccccccccc002', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa001', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb002', 'passing', 0, 100.00, NOW() - INTERVAL '1 day'),
    ('cccccccc-cccc-cccc-cccc-ccccccccc003', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa001', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb003', 'failing', 3, 85.00, NOW() - INTERVAL '1 day'),
    ('cccccccc-cccc-cccc-cccc-ccccccccc004', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa001', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb004', 'passing', 0, 100.00, NOW() - INTERVAL '1 day'),
    ('cccccccc-cccc-cccc-cccc-ccccccccc005', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa001', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb005', 'warning', 5, 75.00, NOW() - INTERVAL '1 day'),

    -- SLSA Results
    ('cccccccc-cccc-cccc-cccc-ccccccccc010', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa002', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb010', 'passing', 0, 100.00, NOW() - INTERVAL '2 days'),
    ('cccccccc-cccc-cccc-cccc-ccccccccc011', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa002', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb011', 'passing', 0, 100.00, NOW() - INTERVAL '2 days'),
    ('cccccccc-cccc-cccc-cccc-ccccccccc012', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa002', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb012', 'failing', 2, 60.00, NOW() - INTERVAL '2 days'),
    ('cccccccc-cccc-cccc-cccc-ccccccccc013', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa002', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb013', 'warning', 1, 80.00, NOW() - INTERVAL '2 days')
ON CONFLICT DO NOTHING;

-- Image Compliance
INSERT INTO image_compliance (image_id, cis_compliant, slsa_level, cosign_signed, last_scan_at, issue_count) VALUES
    ('77777777-7777-7777-7777-777777777701', true, 3, true, NOW() - INTERVAL '1 day', 0),
    ('77777777-7777-7777-7777-777777777702', true, 3, true, NOW() - INTERVAL '7 days', 2),
    ('77777777-7777-7777-7777-777777777703', true, 2, true, NOW() - INTERVAL '30 days', 5),
    ('77777777-7777-7777-7777-777777777704', true, 3, true, NOW() - INTERVAL '1 day', 0),
    ('77777777-7777-7777-7777-777777777705', true, 2, true, NOW() - INTERVAL '14 days', 3),
    ('77777777-7777-7777-7777-777777777706', true, 3, true, NOW() - INTERVAL '2 days', 0),
    ('77777777-7777-7777-7777-777777777707', true, 2, true, NOW() - INTERVAL '21 days', 4),
    ('77777777-7777-7777-7777-777777777708', true, 4, true, NOW() - INTERVAL '1 day', 0),
    ('77777777-7777-7777-7777-777777777709', true, 3, true, NOW() - INTERVAL '7 days', 1),
    ('77777777-7777-7777-7777-777777777710', true, 4, true, NOW() - INTERVAL '1 day', 0)
ON CONFLICT (image_id) DO NOTHING;

-- =============================================================================
-- Alerts
-- =============================================================================

INSERT INTO alerts (id, org_id, severity, title, description, source, site_id, asset_id, status, created_at) VALUES
    ('dddddddd-dddd-dddd-dddd-ddddddddd001', '11111111-1111-1111-1111-111111111111', 'critical', 'Critical asset drifted 25+ days', 'Database server prod-db-001 has been on outdated image for 25 days', 'drift', '55555555-5555-5555-5555-555555555501', '99999999-9999-9999-9999-999999999905', 'open', NOW() - INTERVAL '3 days'),
    ('dddddddd-dddd-dddd-dddd-ddddddddd002', '11111111-1111-1111-1111-111111111111', 'warning', 'DR failover test overdue', 'EU Primary DR pair has not been tested in 90 days', 'compliance', '55555555-5555-5555-5555-555555555503', NULL, 'open', NOW() - INTERVAL '5 days'),
    ('dddddddd-dddd-dddd-dddd-ddddddddd003', '11111111-1111-1111-1111-111111111111', 'warning', 'Legacy ERP system drifted', 'vSphere legacy-erp-002 running outdated Windows image', 'drift', '55555555-5555-5555-5555-555555555507', '99999999-9999-9999-9999-999999999917', 'open', NOW() - INTERVAL '7 days'),
    ('dddddddd-dddd-dddd-dddd-ddddddddd004', '11111111-1111-1111-1111-111111111111', 'info', 'New golden image available', 'Ubuntu 2.5.0 is now production-ready', 'system', NULL, NULL, 'acknowledged', NOW() - INTERVAL '7 days'),
    ('dddddddd-dddd-dddd-dddd-ddddddddd005', '11111111-1111-1111-1111-111111111111', 'critical', 'CIS control failing', '3 assets failing firewall compliance check', 'compliance', NULL, NULL, 'open', NOW() - INTERVAL '1 day'),
    ('dddddddd-dddd-dddd-dddd-ddddddddd006', '11111111-1111-1111-1111-111111111111', 'warning', 'K8s deployment outdated', 'Frontend deployment using old API image version', 'drift', '55555555-5555-5555-5555-555555555508', '99999999-9999-9999-9999-999999999920', 'open', NOW() - INTERVAL '2 days')
ON CONFLICT DO NOTHING;

-- =============================================================================
-- Activities (Recent Actions)
-- =============================================================================

INSERT INTO activities (id, org_id, type, action, detail, user_id, site_id, image_id, created_at) VALUES
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeee001', '11111111-1111-1111-1111-111111111111', 'success', 'Image promoted to production', 'Ubuntu 2.5.0 promoted by Alice Admin', '22222222-2222-2222-2222-222222222201', NULL, '77777777-7777-7777-7777-777777777701', NOW() - INTERVAL '7 days'),
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeee002', '11111111-1111-1111-1111-111111111111', 'success', 'DR failover test completed', 'US East-West DR pair tested successfully', '22222222-2222-2222-2222-222222222202', '55555555-5555-5555-5555-555555555501', NULL, NOW() - INTERVAL '30 days'),
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeee003', '11111111-1111-1111-1111-111111111111', 'warning', 'Connector sync failed', 'Azure connector failed to sync - retrying', NULL, '55555555-5555-5555-5555-555555555504', NULL, NOW() - INTERVAL '2 hours'),
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeee004', '11111111-1111-1111-1111-111111111111', 'info', 'Compliance scan completed', 'Weekly CIS benchmark scan completed', NULL, NULL, NULL, NOW() - INTERVAL '1 day'),
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeee005', '11111111-1111-1111-1111-111111111111', 'success', 'Patch rollout completed', '5 assets updated to Amazon Linux 1.8.0', '22222222-2222-2222-2222-222222222202', '55555555-5555-5555-5555-555555555501', '77777777-7777-7777-7777-777777777704', NOW() - INTERVAL '5 days'),
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeee006', '11111111-1111-1111-1111-111111111111', 'critical', 'Security vulnerability detected', 'CVE-2024-1234 affects 3 production images', NULL, NULL, NULL, NOW() - INTERVAL '12 hours')
ON CONFLICT DO NOTHING;

-- =============================================================================
-- Drift Reports
-- =============================================================================

INSERT INTO drift_reports (id, org_id, platform, site, total_assets, compliant_assets, coverage_pct, status, calculated_at) VALUES
    ('ffffffff-ffff-ffff-ffff-fffffffffff1', '11111111-1111-1111-1111-111111111111', 'aws', 'us-east-1', 6, 4, 66.67, 'warning', NOW() - INTERVAL '1 hour'),
    ('ffffffff-ffff-ffff-ffff-fffffffffff2', '11111111-1111-1111-1111-111111111111', 'aws', 'us-west-2', 2, 2, 100.00, 'healthy', NOW() - INTERVAL '1 hour'),
    ('ffffffff-ffff-ffff-ffff-fffffffffff3', '11111111-1111-1111-1111-111111111111', 'aws', 'eu-west-1', 2, 1, 50.00, 'critical', NOW() - INTERVAL '1 hour'),
    ('ffffffff-ffff-ffff-ffff-fffffffffff4', '11111111-1111-1111-1111-111111111111', 'azure', 'eastus', 3, 2, 66.67, 'warning', NOW() - INTERVAL '1 hour'),
    ('ffffffff-ffff-ffff-ffff-fffffffffff5', '11111111-1111-1111-1111-111111111111', 'gcp', 'us-central1', 2, 1, 50.00, 'warning', NOW() - INTERVAL '1 hour'),
    ('ffffffff-ffff-ffff-ffff-fffffffffff6', '11111111-1111-1111-1111-111111111111', 'vsphere', 'uk-lon-01', 2, 1, 50.00, 'critical', NOW() - INTERVAL '1 hour'),
    ('ffffffff-ffff-ffff-ffff-fffffffffff7', '11111111-1111-1111-1111-111111111111', 'k8s', 'eks-prod', 3, 2, 66.67, 'warning', NOW() - INTERVAL '1 hour')
ON CONFLICT DO NOTHING;

-- =============================================================================
-- Connectors
-- =============================================================================

INSERT INTO connectors (id, org_id, name, platform, enabled, config, last_sync_at, last_sync_status) VALUES
    ('11111111-cccc-cccc-cccc-111111111101', '11111111-1111-1111-1111-111111111111', 'AWS Production', 'aws', true, '{"regions": ["us-east-1", "us-west-2", "eu-west-1"], "role_arn": "arn:aws:iam::123456789012:role/QLRFConnector"}', NOW() - INTERVAL '5 minutes', 'success'),
    ('11111111-cccc-cccc-cccc-111111111102', '11111111-1111-1111-1111-111111111111', 'Azure Enterprise', 'azure', true, '{"subscription_id": "sub-12345", "tenant_id": "tenant-xxx"}', NOW() - INTERVAL '2 hours', 'warning'),
    ('11111111-cccc-cccc-cccc-111111111103', '11111111-1111-1111-1111-111111111111', 'GCP Analytics', 'gcp', true, '{"project_id": "acme-analytics"}', NOW() - INTERVAL '10 minutes', 'success'),
    ('11111111-cccc-cccc-cccc-111111111104', '11111111-1111-1111-1111-111111111111', 'vSphere London DC', 'vsphere', true, '{"vcenter": "vcenter-lon.internal"}', NOW() - INTERVAL '30 minutes', 'success'),
    ('11111111-cccc-cccc-cccc-111111111105', '11111111-1111-1111-1111-111111111111', 'EKS Production', 'k8s', true, '{"cluster": "eks-prod-cluster", "namespace": "production"}', NOW() - INTERVAL '3 minutes', 'success')
ON CONFLICT (org_id, name) DO NOTHING;

-- =============================================================================
-- Summary
-- =============================================================================
-- Demo data creates:
-- - 1 organization with 3 users
-- - 3 projects with 5 environments
-- - 8 sites across AWS, Azure, GCP, vSphere, K8s
-- - 3 DR pairs with various health states
-- - 10 golden images (3 families, multiple versions)
-- - 20 assets with mix of compliant/drifted
-- - 5 compliance frameworks with controls and results
-- - 6 alerts of varying severity
-- - 6 recent activities
-- - 7 drift reports
-- - 5 cloud connectors

SELECT 'Demo data seeded successfully!' as status;
SELECT 'Organization: Acme Corporation (11111111-1111-1111-1111-111111111111)' as info;
SELECT 'Total assets: ' || COUNT(*) as assets FROM assets WHERE org_id = '11111111-1111-1111-1111-111111111111';
SELECT 'Drifted assets: ' || COUNT(*) as drifted FROM assets
    WHERE org_id = '11111111-1111-1111-1111-111111111111'
    AND image_version NOT IN (
        SELECT version FROM images
        WHERE family = assets.image_ref
        AND status = 'production'
        AND org_id = '11111111-1111-1111-1111-111111111111'
    );
