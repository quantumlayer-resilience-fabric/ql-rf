-- QuantumLayer Resilience Fabric - Sites, Alerts, Compliance Schema
-- Migration: 000003_add_sites_alerts_compliance
-- Description: Adds sites, alerts, activities, DR pairs, and compliance tables

-- =============================================================================
-- Sites Table
-- =============================================================================

CREATE TABLE sites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    region VARCHAR(63) NOT NULL,
    platform VARCHAR(31) NOT NULL,
    environment VARCHAR(31) NOT NULL DEFAULT 'production',
    dr_paired_site_id UUID REFERENCES sites(id) ON DELETE SET NULL,
    last_sync_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, name)
);

-- =============================================================================
-- Alerts Table
-- =============================================================================

CREATE TABLE alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    severity VARCHAR(31) NOT NULL, -- critical, warning, info
    title VARCHAR(255) NOT NULL,
    description TEXT,
    source VARCHAR(127) NOT NULL, -- drift, compliance, connector, system
    site_id UUID REFERENCES sites(id) ON DELETE SET NULL,
    asset_id UUID REFERENCES assets(id) ON DELETE SET NULL,
    image_id UUID REFERENCES images(id) ON DELETE SET NULL,
    status VARCHAR(31) NOT NULL DEFAULT 'open', -- open, acknowledged, resolved
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by UUID REFERENCES users(id) ON DELETE SET NULL,
    resolved_at TIMESTAMPTZ,
    resolved_by UUID REFERENCES users(id) ON DELETE SET NULL
);

-- =============================================================================
-- Activities Table
-- =============================================================================

CREATE TABLE activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    type VARCHAR(31) NOT NULL, -- info, warning, success, critical
    action VARCHAR(255) NOT NULL,
    detail TEXT,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    site_id UUID REFERENCES sites(id) ON DELETE SET NULL,
    asset_id UUID REFERENCES assets(id) ON DELETE SET NULL,
    image_id UUID REFERENCES images(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- DR Pairs Table
-- =============================================================================

CREATE TABLE dr_pairs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    primary_site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    dr_site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    status VARCHAR(31) NOT NULL DEFAULT 'unknown', -- healthy, warning, critical, syncing
    replication_status VARCHAR(31) NOT NULL DEFAULT 'unknown', -- in-sync, lagging, failed
    rpo VARCHAR(31), -- Recovery Point Objective (e.g., "15 min")
    rto VARCHAR(31), -- Recovery Time Objective (e.g., "4 hours")
    last_failover_test TIMESTAMPTZ,
    last_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, name),
    CHECK (primary_site_id != dr_site_id)
);

-- =============================================================================
-- Compliance Tables
-- =============================================================================

-- Compliance frameworks (CIS, SLSA, SOC2, HIPAA, PCI-DSS)
CREATE TABLE compliance_frameworks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(127) NOT NULL, -- CIS, SLSA, SOC2, HIPAA, PCI
    description TEXT,
    level INT, -- For SLSA levels (1-4)
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, name)
);

-- Compliance controls within frameworks
CREATE TABLE compliance_controls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    framework_id UUID NOT NULL REFERENCES compliance_frameworks(id) ON DELETE CASCADE,
    control_id VARCHAR(31) NOT NULL, -- e.g., "CIS-4.2.1", "SLSA-L2"
    title VARCHAR(255) NOT NULL,
    description TEXT,
    severity VARCHAR(31) NOT NULL, -- high, medium, low
    recommendation TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(framework_id, control_id)
);

-- Compliance scan results
CREATE TABLE compliance_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    framework_id UUID NOT NULL REFERENCES compliance_frameworks(id) ON DELETE CASCADE,
    control_id UUID REFERENCES compliance_controls(id) ON DELETE SET NULL,
    status VARCHAR(31) NOT NULL, -- passing, failing, warning
    affected_assets INT NOT NULL DEFAULT 0,
    score DECIMAL(5,2),
    last_audit_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Image compliance status (CIS, SLSA level, Cosign signature)
CREATE TABLE image_compliance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    cis_compliant BOOLEAN NOT NULL DEFAULT FALSE,
    slsa_level INT NOT NULL DEFAULT 0,
    cosign_signed BOOLEAN NOT NULL DEFAULT FALSE,
    last_scan_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    issue_count INT NOT NULL DEFAULT 0,
    UNIQUE(image_id)
);

-- =============================================================================
-- Indexes
-- =============================================================================

-- Sites
CREATE INDEX idx_sites_org_id ON sites(org_id);
CREATE INDEX idx_sites_platform ON sites(platform);
CREATE INDEX idx_sites_region ON sites(region);
CREATE INDEX idx_sites_environment ON sites(environment);

-- Alerts
CREATE INDEX idx_alerts_org_id ON alerts(org_id);
CREATE INDEX idx_alerts_severity ON alerts(severity);
CREATE INDEX idx_alerts_status ON alerts(status);
CREATE INDEX idx_alerts_created_at ON alerts(org_id, created_at DESC);
CREATE INDEX idx_alerts_site_id ON alerts(site_id);

-- Activities
CREATE INDEX idx_activities_org_id ON activities(org_id);
CREATE INDEX idx_activities_created_at ON activities(org_id, created_at DESC);
CREATE INDEX idx_activities_type ON activities(type);

-- DR Pairs
CREATE INDEX idx_dr_pairs_org_id ON dr_pairs(org_id);
CREATE INDEX idx_dr_pairs_primary_site ON dr_pairs(primary_site_id);
CREATE INDEX idx_dr_pairs_dr_site ON dr_pairs(dr_site_id);
CREATE INDEX idx_dr_pairs_status ON dr_pairs(status);

-- Compliance
CREATE INDEX idx_compliance_frameworks_org_id ON compliance_frameworks(org_id);
CREATE INDEX idx_compliance_controls_framework_id ON compliance_controls(framework_id);
CREATE INDEX idx_compliance_results_org_id ON compliance_results(org_id);
CREATE INDEX idx_compliance_results_framework_id ON compliance_results(framework_id);
CREATE INDEX idx_image_compliance_image_id ON image_compliance(image_id);

-- =============================================================================
-- Triggers for updated_at
-- =============================================================================

CREATE TRIGGER update_sites_updated_at
    BEFORE UPDATE ON sites
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_dr_pairs_updated_at
    BEFORE UPDATE ON dr_pairs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_compliance_frameworks_updated_at
    BEFORE UPDATE ON compliance_frameworks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- Update assets table to link to sites
-- =============================================================================

-- Add site_id column to assets for direct site reference
ALTER TABLE assets ADD COLUMN site_id UUID REFERENCES sites(id) ON DELETE SET NULL;
CREATE INDEX idx_assets_site_id ON assets(site_id);
