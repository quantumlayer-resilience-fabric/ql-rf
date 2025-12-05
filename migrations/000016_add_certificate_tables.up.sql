-- Certificate Management Tables for QL-RF
-- Supports certificate lifecycle management across multi-cloud and hybrid infrastructure

-- =============================================================================
-- Certificate Inventory
-- =============================================================================

CREATE TABLE IF NOT EXISTS certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- Certificate Identity
    fingerprint VARCHAR(128) NOT NULL,
    serial_number VARCHAR(128),

    -- Subject Information
    common_name VARCHAR(255) NOT NULL,
    subject_alt_names TEXT[], -- Array of SANs (DNS names, IPs, emails)
    organization VARCHAR(255),
    organizational_unit VARCHAR(255),
    country VARCHAR(10),

    -- Issuer Information
    issuer_common_name VARCHAR(255),
    issuer_organization VARCHAR(255),
    is_self_signed BOOLEAN DEFAULT FALSE,
    is_ca BOOLEAN DEFAULT FALSE,

    -- Validity Period
    not_before TIMESTAMPTZ NOT NULL,
    not_after TIMESTAMPTZ NOT NULL,

    -- Technical Details
    key_algorithm VARCHAR(50), -- RSA, ECDSA, Ed25519
    key_size INTEGER, -- 2048, 4096 for RSA; 256, 384 for ECDSA
    signature_algorithm VARCHAR(100),

    -- Source Information
    source VARCHAR(50) NOT NULL, -- acm, azure_keyvault, gcp_certificate_manager, k8s_secret, vault, file, vsphere
    source_ref VARCHAR(500), -- ARN, resource ID, secret name, file path
    source_region VARCHAR(50),
    platform VARCHAR(20) NOT NULL, -- aws, azure, gcp, k8s, vsphere

    -- Status and Lifecycle
    status VARCHAR(30) DEFAULT 'active', -- active, expiring_soon, expired, revoked, pending_renewal
    days_until_expiry INTEGER, -- Computed by trigger on insert/update

    -- Rotation Policy
    auto_renew BOOLEAN DEFAULT FALSE,
    renewal_threshold_days INTEGER DEFAULT 30, -- Alert when this many days until expiry
    last_rotated_at TIMESTAMPTZ,
    rotation_count INTEGER DEFAULT 0,

    -- Metadata
    tags JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}', -- Additional platform-specific metadata

    -- Audit Fields
    discovered_at TIMESTAMPTZ DEFAULT NOW(),
    last_scanned_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Constraints
    UNIQUE(org_id, fingerprint)
);

-- Indexes for common queries
CREATE INDEX idx_certificates_org ON certificates(org_id);
CREATE INDEX idx_certificates_expiry ON certificates(not_after);
CREATE INDEX idx_certificates_status ON certificates(status);
CREATE INDEX idx_certificates_platform ON certificates(platform);
CREATE INDEX idx_certificates_days_until ON certificates(days_until_expiry) WHERE days_until_expiry <= 90;
CREATE INDEX idx_certificates_common_name ON certificates(common_name);
CREATE INDEX idx_certificates_san ON certificates USING GIN(subject_alt_names);

-- =============================================================================
-- Certificate Usage (Where certificates are deployed)
-- =============================================================================

CREATE TABLE IF NOT EXISTS certificate_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cert_id UUID NOT NULL REFERENCES certificates(id) ON DELETE CASCADE,
    asset_id UUID REFERENCES assets(id) ON DELETE SET NULL,

    -- Usage Location
    usage_type VARCHAR(50) NOT NULL, -- lb_listener, api_gateway, ingress, pod_mount, vm_tls, service_mesh, cdn, custom
    usage_ref VARCHAR(500) NOT NULL, -- Specific resource reference
    usage_port INTEGER, -- Port if applicable

    -- Platform Details
    platform VARCHAR(20) NOT NULL,
    region VARCHAR(50),

    -- Service Information
    service_name VARCHAR(255),
    endpoint VARCHAR(500), -- The actual endpoint using this cert

    -- Status
    status VARCHAR(30) DEFAULT 'active', -- active, inactive, pending_removal
    last_verified_at TIMESTAMPTZ,
    tls_version VARCHAR(20), -- TLS 1.2, TLS 1.3

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Audit Fields
    discovered_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Constraints
    UNIQUE(cert_id, usage_type, usage_ref)
);

-- Indexes
CREATE INDEX idx_cert_usage_cert ON certificate_usage(cert_id);
CREATE INDEX idx_cert_usage_asset ON certificate_usage(asset_id);
CREATE INDEX idx_cert_usage_type ON certificate_usage(usage_type);
CREATE INDEX idx_cert_usage_platform ON certificate_usage(platform);

-- =============================================================================
-- Certificate Rotation History
-- =============================================================================

CREATE TABLE IF NOT EXISTS certificate_rotations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- Certificates Involved
    old_cert_id UUID REFERENCES certificates(id) ON DELETE SET NULL,
    new_cert_id UUID REFERENCES certificates(id) ON DELETE SET NULL,

    -- Rotation Details
    rotation_type VARCHAR(30) NOT NULL, -- renewal, replacement, emergency, scheduled
    initiated_by VARCHAR(50), -- auto, user, alert, api
    initiated_by_user_id VARCHAR(255),

    -- AI Task Reference
    ai_task_id UUID, -- Reference to AI orchestrator task
    ai_plan JSONB, -- The generated rotation plan

    -- Execution Status
    status VARCHAR(30) DEFAULT 'pending', -- pending, in_progress, completed, failed, rolled_back
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Affected Resources
    affected_usages INTEGER DEFAULT 0,
    successful_updates INTEGER DEFAULT 0,
    failed_updates INTEGER DEFAULT 0,

    -- Rollback Information
    rollback_available BOOLEAN DEFAULT TRUE,
    rolled_back_at TIMESTAMPTZ,
    rollback_reason TEXT,

    -- Validation
    pre_rotation_validation JSONB, -- Health checks before rotation
    post_rotation_validation JSONB, -- Health checks after rotation

    -- Error Details
    error_message TEXT,
    error_details JSONB,

    -- Audit Fields
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_cert_rotations_org ON certificate_rotations(org_id);
CREATE INDEX idx_cert_rotations_old ON certificate_rotations(old_cert_id);
CREATE INDEX idx_cert_rotations_new ON certificate_rotations(new_cert_id);
CREATE INDEX idx_cert_rotations_status ON certificate_rotations(status);
CREATE INDEX idx_cert_rotations_task ON certificate_rotations(ai_task_id);

-- =============================================================================
-- Certificate Alerts
-- =============================================================================

CREATE TABLE IF NOT EXISTS certificate_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    cert_id UUID NOT NULL REFERENCES certificates(id) ON DELETE CASCADE,

    -- Alert Details
    alert_type VARCHAR(50) NOT NULL, -- expiring_soon, expired, weak_key, self_signed_prod, san_mismatch, revoked
    severity VARCHAR(20) NOT NULL, -- critical, high, medium, low

    -- Alert Content
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,

    -- Thresholds
    days_until_expiry INTEGER,
    threshold_days INTEGER,

    -- Status
    status VARCHAR(30) DEFAULT 'open', -- open, acknowledged, resolved, suppressed
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by VARCHAR(255),
    resolved_at TIMESTAMPTZ,

    -- Actions Taken
    auto_rotation_triggered BOOLEAN DEFAULT FALSE,
    rotation_id UUID REFERENCES certificate_rotations(id),

    -- Notification Status
    notifications_sent JSONB DEFAULT '[]', -- Array of {channel, sent_at, status}

    -- Audit Fields
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_cert_alerts_org ON certificate_alerts(org_id);
CREATE INDEX idx_cert_alerts_cert ON certificate_alerts(cert_id);
CREATE INDEX idx_cert_alerts_status ON certificate_alerts(status);
CREATE INDEX idx_cert_alerts_severity ON certificate_alerts(severity);
CREATE INDEX idx_cert_alerts_type ON certificate_alerts(alert_type);

-- =============================================================================
-- Certificate Scan Jobs
-- =============================================================================

CREATE TABLE IF NOT EXISTS certificate_scan_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- Scan Configuration
    scan_type VARCHAR(30) NOT NULL, -- full, incremental, targeted
    platforms TEXT[] DEFAULT ARRAY['aws', 'azure', 'gcp', 'k8s', 'vsphere'],
    regions TEXT[],

    -- Execution Status
    status VARCHAR(30) DEFAULT 'pending', -- pending, running, completed, failed
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Results Summary
    certificates_found INTEGER DEFAULT 0,
    new_certificates INTEGER DEFAULT 0,
    updated_certificates INTEGER DEFAULT 0,
    usages_found INTEGER DEFAULT 0,
    alerts_generated INTEGER DEFAULT 0,

    -- Error Handling
    errors JSONB DEFAULT '[]',

    -- Scheduling
    scheduled BOOLEAN DEFAULT FALSE,
    schedule_cron VARCHAR(100),
    next_run_at TIMESTAMPTZ,

    -- Audit Fields
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_cert_scans_org ON certificate_scan_jobs(org_id);
CREATE INDEX idx_cert_scans_status ON certificate_scan_jobs(status);

-- =============================================================================
-- Views for Common Queries
-- =============================================================================

-- Expiring certificates with usage count
CREATE OR REPLACE VIEW v_expiring_certificates AS
SELECT
    c.id,
    c.org_id,
    c.common_name,
    c.subject_alt_names,
    c.not_after,
    c.days_until_expiry,
    c.status,
    c.platform,
    c.source,
    c.auto_renew,
    COUNT(cu.id) as usage_count,
    ARRAY_AGG(DISTINCT cu.usage_type) FILTER (WHERE cu.id IS NOT NULL) as usage_types,
    ARRAY_AGG(DISTINCT cu.service_name) FILTER (WHERE cu.service_name IS NOT NULL) as services
FROM certificates c
LEFT JOIN certificate_usage cu ON c.id = cu.cert_id
WHERE c.days_until_expiry <= 90
  AND c.status NOT IN ('revoked', 'expired')
GROUP BY c.id
ORDER BY c.days_until_expiry ASC;

-- Certificate blast radius (impact analysis)
CREATE OR REPLACE VIEW v_certificate_blast_radius AS
SELECT
    c.id as cert_id,
    c.org_id,
    c.common_name,
    c.days_until_expiry,
    c.status as cert_status,
    cu.id as usage_id,
    cu.usage_type,
    cu.usage_ref,
    cu.service_name,
    cu.endpoint,
    a.id as asset_id,
    a.name as asset_name,
    a.platform as asset_platform,
    a.region as asset_region,
    a.state as asset_state
FROM certificates c
JOIN certificate_usage cu ON c.id = cu.cert_id
LEFT JOIN assets a ON cu.asset_id = a.id
ORDER BY c.days_until_expiry ASC, c.common_name;

-- Certificate dashboard summary
CREATE OR REPLACE VIEW v_certificate_summary AS
SELECT
    org_id,
    COUNT(*) as total_certificates,
    COUNT(*) FILTER (WHERE status = 'active') as active_certificates,
    COUNT(*) FILTER (WHERE status = 'expiring_soon') as expiring_soon,
    COUNT(*) FILTER (WHERE status = 'expired') as expired,
    COUNT(*) FILTER (WHERE days_until_expiry <= 7) as expiring_7_days,
    COUNT(*) FILTER (WHERE days_until_expiry <= 30) as expiring_30_days,
    COUNT(*) FILTER (WHERE days_until_expiry <= 90) as expiring_90_days,
    COUNT(*) FILTER (WHERE auto_renew = true) as auto_renew_enabled,
    COUNT(*) FILTER (WHERE is_self_signed = true) as self_signed,
    COUNT(DISTINCT platform) as platforms_count
FROM certificates
GROUP BY org_id;

-- =============================================================================
-- Triggers
-- =============================================================================

-- Update certificate status and days_until_expiry
CREATE OR REPLACE FUNCTION update_certificate_status()
RETURNS TRIGGER AS $$
BEGIN
    -- Calculate days until expiry
    NEW.days_until_expiry := EXTRACT(DAY FROM (NEW.not_after - NOW()))::INTEGER;

    -- Update status based on expiry
    IF NEW.not_after < NOW() THEN
        NEW.status := 'expired';
    ELSIF NEW.days_until_expiry <= NEW.renewal_threshold_days THEN
        NEW.status := 'expiring_soon';
    ELSIF NEW.status NOT IN ('revoked', 'pending_renewal') THEN
        NEW.status := 'active';
    END IF;
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_certificate_status
    BEFORE INSERT OR UPDATE ON certificates
    FOR EACH ROW
    EXECUTE FUNCTION update_certificate_status();

-- Update timestamps
CREATE OR REPLACE FUNCTION update_certificate_timestamps()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_cert_usage_updated
    BEFORE UPDATE ON certificate_usage
    FOR EACH ROW
    EXECUTE FUNCTION update_certificate_timestamps();

CREATE TRIGGER trg_cert_rotations_updated
    BEFORE UPDATE ON certificate_rotations
    FOR EACH ROW
    EXECUTE FUNCTION update_certificate_timestamps();

CREATE TRIGGER trg_cert_alerts_updated
    BEFORE UPDATE ON certificate_alerts
    FOR EACH ROW
    EXECUTE FUNCTION update_certificate_timestamps();

-- =============================================================================
-- Sample Data for Development
-- =============================================================================

-- Insert sample certificates (will use demo org from seed data)
INSERT INTO certificates (
    org_id, fingerprint, common_name, subject_alt_names,
    issuer_common_name, issuer_organization,
    not_before, not_after,
    key_algorithm, key_size, signature_algorithm,
    source, source_ref, platform,
    auto_renew, renewal_threshold_days
)
SELECT
    o.id,
    'SHA256:' || md5(domain || o.id::text) || md5(source_ref),
    domain,
    ARRAY[domain, 'www.' || domain],
    'Let''s Encrypt Authority X3',
    'Let''s Encrypt',
    NOW() - interval '60 days',
    NOW() + (days_left || ' days')::interval,
    'RSA',
    2048,
    'SHA256WithRSA',
    source,
    source_ref,
    platform,
    auto_renew,
    30
FROM organizations o
CROSS JOIN (VALUES
    ('api.example.com', '15 days', 'acm', 'arn:aws:acm:us-east-1:123456789:certificate/abc123', 'aws', true),
    ('app.example.com', '45 days', 'acm', 'arn:aws:acm:us-west-2:123456789:certificate/def456', 'aws', true),
    ('internal.example.com', '5 days', 'k8s_secret', 'default/tls-internal', 'k8s', false),
    ('admin.example.com', '90 days', 'azure_keyvault', '/subscriptions/xxx/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/kv/certificates/admin-cert', 'azure', true),
    ('legacy.example.com', '-10 days', 'file', '/etc/ssl/certs/legacy.pem', 'vsphere', false)
) AS certs(domain, days_left, source, source_ref, platform, auto_renew)
WHERE o.name = 'Demo Organization'
ON CONFLICT DO NOTHING;

-- Insert sample certificate usage
INSERT INTO certificate_usage (cert_id, usage_type, usage_ref, platform, service_name, endpoint)
SELECT
    c.id,
    usage.type,
    usage.ref,
    c.platform,
    usage.service,
    'https://' || c.common_name || usage.path
FROM certificates c
CROSS JOIN (VALUES
    ('lb_listener', 'alb-prod-1/443', 'api-gateway', '/api'),
    ('ingress', 'ingress-nginx/tls', 'web-app', '/'),
    ('api_gateway', 'apigw-main/stage-prod', 'public-api', '/v1')
) AS usage(type, ref, service, path)
WHERE c.common_name = 'api.example.com'
ON CONFLICT DO NOTHING;
