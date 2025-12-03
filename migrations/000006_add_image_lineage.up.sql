-- QuantumLayer Resilience Fabric - Image Lineage
-- Migration: 000006_add_image_lineage
-- Description: Adds golden image lineage tracking, provenance, and vulnerability mapping

-- =============================================================================
-- Image Lineage (Parent-Child Relationships)
-- =============================================================================

-- Image lineage tracks parent-child relationships between images
CREATE TABLE image_lineage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    parent_image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    relationship_type VARCHAR(31) NOT NULL DEFAULT 'derived_from', -- derived_from, patched_from, rebuilt_from
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(image_id, parent_image_id),
    CHECK (image_id != parent_image_id) -- Prevent self-reference
);

COMMENT ON TABLE image_lineage IS 'Tracks parent-child relationships between golden images';
COMMENT ON COLUMN image_lineage.relationship_type IS 'Type: derived_from (new base), patched_from (security patch), rebuilt_from (same spec rebuild)';

-- =============================================================================
-- Image Build Provenance (SLSA-style)
-- =============================================================================

-- Build provenance tracks how each image was created
CREATE TABLE image_builds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    build_number INT NOT NULL,

    -- Build source
    source_repo VARCHAR(512),           -- git repo URL
    source_commit VARCHAR(64),          -- git commit SHA
    source_branch VARCHAR(255),         -- git branch
    source_tag VARCHAR(255),            -- git tag if applicable

    -- Build system
    builder_type VARCHAR(63) NOT NULL,  -- packer, docker, azure_image_builder, etc.
    builder_version VARCHAR(63),
    build_template TEXT,                -- Packer HCL, Dockerfile, etc. (stored as text)
    build_config JSONB,                 -- Build variables/parameters

    -- Build environment
    build_runner VARCHAR(255),          -- CI system: github_actions, azure_devops, jenkins
    build_runner_id VARCHAR(255),       -- CI run ID
    build_runner_url TEXT,              -- Link to CI run

    -- Build artifacts
    build_log_url TEXT,
    build_duration_seconds INT,

    -- Security
    built_by VARCHAR(255),              -- User or service principal
    signed_by VARCHAR(255),             -- Signing identity
    signature TEXT,                     -- Cosign/Notary signature
    attestation_url TEXT,               -- SLSA attestation URL

    -- Status
    status VARCHAR(31) NOT NULL DEFAULT 'pending', -- pending, building, success, failed
    error_message TEXT,

    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(image_id, build_number)
);

COMMENT ON TABLE image_builds IS 'Build provenance for each image version (SLSA-compatible)';

-- =============================================================================
-- Image Vulnerability Tracking
-- =============================================================================

-- Vulnerabilities associated with images
CREATE TABLE image_vulnerabilities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,

    -- CVE details
    cve_id VARCHAR(31) NOT NULL,        -- CVE-2024-1234
    severity VARCHAR(31) NOT NULL,       -- critical, high, medium, low, unknown
    cvss_score DECIMAL(3,1),            -- 0.0 - 10.0
    cvss_vector VARCHAR(255),           -- CVSS vector string

    -- Affected component
    package_name VARCHAR(255),          -- e.g., openssl
    package_version VARCHAR(63),        -- e.g., 1.1.1k
    package_type VARCHAR(31),           -- deb, rpm, apk, npm, pip, etc.
    fixed_version VARCHAR(63),          -- Version that fixes this CVE

    -- Status in this image
    status VARCHAR(31) NOT NULL DEFAULT 'open', -- open, fixed, wont_fix, false_positive
    status_reason TEXT,

    -- Scan info
    scanner VARCHAR(63),                -- trivy, grype, qualys, etc.
    scanned_at TIMESTAMPTZ,

    -- Resolution tracking
    fixed_in_image_id UUID REFERENCES images(id) ON DELETE SET NULL,
    resolved_at TIMESTAMPTZ,
    resolved_by VARCHAR(255),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(image_id, cve_id, package_name)
);

COMMENT ON TABLE image_vulnerabilities IS 'Vulnerability tracking per image for compliance reporting';

-- =============================================================================
-- Image Deployment Tracking
-- =============================================================================

-- Track where images are deployed
CREATE TABLE image_deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,

    -- Deployment info
    deployed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deployed_by VARCHAR(255),           -- User or automation
    deployment_method VARCHAR(63),      -- terraform, ansible, manual, auto_scale

    -- Status
    status VARCHAR(31) NOT NULL DEFAULT 'active', -- active, replaced, terminated
    replaced_at TIMESTAMPTZ,
    replaced_by_image_id UUID REFERENCES images(id) ON DELETE SET NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE image_deployments IS 'Tracks which assets are running which image versions';

-- =============================================================================
-- Image Promotion History
-- =============================================================================

-- Track status transitions (promotion/demotion)
CREATE TABLE image_promotions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,

    from_status VARCHAR(31) NOT NULL,
    to_status VARCHAR(31) NOT NULL,

    -- Approval workflow
    promoted_by VARCHAR(255) NOT NULL,
    approved_by VARCHAR(255),           -- If different from promoter
    approval_ticket VARCHAR(255),       -- Jira/ServiceNow ticket

    -- Reason
    reason TEXT,

    -- Validation results
    validation_passed BOOLEAN,
    validation_results JSONB,           -- Test results, compliance checks, etc.

    promoted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE image_promotions IS 'Audit trail of image status changes (promotions/demotions)';

-- =============================================================================
-- Image Components (SBOM-style)
-- =============================================================================

-- Key components/packages in each image
CREATE TABLE image_components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,

    -- Component info
    name VARCHAR(255) NOT NULL,
    version VARCHAR(63) NOT NULL,
    component_type VARCHAR(31) NOT NULL, -- os_package, library, binary, container
    package_manager VARCHAR(31),        -- apt, yum, apk, pip, npm, etc.

    -- License
    license VARCHAR(127),
    license_url TEXT,

    -- Source
    source_url TEXT,
    checksum VARCHAR(128),              -- SHA256 of the component

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(image_id, name, version, component_type)
);

COMMENT ON TABLE image_components IS 'Software bill of materials (SBOM) components per image';

-- =============================================================================
-- Image Tags (for custom metadata)
-- =============================================================================

CREATE TABLE image_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    key VARCHAR(63) NOT NULL,
    value VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(image_id, key)
);

COMMENT ON TABLE image_tags IS 'Custom key-value tags for images';

-- =============================================================================
-- Indexes
-- =============================================================================

-- Image Lineage
CREATE INDEX idx_image_lineage_image_id ON image_lineage(image_id);
CREATE INDEX idx_image_lineage_parent_id ON image_lineage(parent_image_id);

-- Image Builds
CREATE INDEX idx_image_builds_image_id ON image_builds(image_id);
CREATE INDEX idx_image_builds_status ON image_builds(status);
CREATE INDEX idx_image_builds_source_commit ON image_builds(source_commit);

-- Image Vulnerabilities
CREATE INDEX idx_image_vulns_image_id ON image_vulnerabilities(image_id);
CREATE INDEX idx_image_vulns_cve_id ON image_vulnerabilities(cve_id);
CREATE INDEX idx_image_vulns_severity ON image_vulnerabilities(severity);
CREATE INDEX idx_image_vulns_status ON image_vulnerabilities(status);
CREATE INDEX idx_image_vulns_fixed_in ON image_vulnerabilities(fixed_in_image_id);

-- Image Deployments
CREATE INDEX idx_image_deployments_image_id ON image_deployments(image_id);
CREATE INDEX idx_image_deployments_asset_id ON image_deployments(asset_id);
CREATE INDEX idx_image_deployments_status ON image_deployments(status);

-- Image Promotions
CREATE INDEX idx_image_promotions_image_id ON image_promotions(image_id);
CREATE INDEX idx_image_promotions_promoted_at ON image_promotions(promoted_at DESC);

-- Image Components
CREATE INDEX idx_image_components_image_id ON image_components(image_id);
CREATE INDEX idx_image_components_name ON image_components(name);

-- Image Tags
CREATE INDEX idx_image_tags_image_id ON image_tags(image_id);
CREATE INDEX idx_image_tags_key ON image_tags(key);

-- =============================================================================
-- Triggers
-- =============================================================================

-- Auto-update updated_at for vulnerabilities
CREATE TRIGGER update_image_vulnerabilities_updated_at
    BEFORE UPDATE ON image_vulnerabilities
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- Views for Common Queries
-- =============================================================================

-- View: Image lineage tree (with depth)
CREATE OR REPLACE VIEW v_image_lineage_tree AS
WITH RECURSIVE lineage_tree AS (
    -- Base case: images with no parents
    SELECT
        i.id,
        i.family,
        i.version,
        i.status,
        NULL::UUID as parent_id,
        NULL::VARCHAR as parent_version,
        0 as depth,
        ARRAY[i.id] as path
    FROM images i
    WHERE NOT EXISTS (
        SELECT 1 FROM image_lineage il WHERE il.image_id = i.id
    )

    UNION ALL

    -- Recursive case: images with parents
    SELECT
        i.id,
        i.family,
        i.version,
        i.status,
        lt.id as parent_id,
        lt.version as parent_version,
        lt.depth + 1,
        lt.path || i.id
    FROM images i
    JOIN image_lineage il ON il.image_id = i.id
    JOIN lineage_tree lt ON lt.id = il.parent_image_id
    WHERE NOT (i.id = ANY(lt.path)) -- Prevent cycles
)
SELECT * FROM lineage_tree;

-- View: Image vulnerability summary
CREATE OR REPLACE VIEW v_image_vuln_summary AS
SELECT
    i.id as image_id,
    i.family,
    i.version,
    i.status,
    COUNT(*) FILTER (WHERE iv.severity = 'critical' AND iv.status = 'open') as critical_open,
    COUNT(*) FILTER (WHERE iv.severity = 'high' AND iv.status = 'open') as high_open,
    COUNT(*) FILTER (WHERE iv.severity = 'medium' AND iv.status = 'open') as medium_open,
    COUNT(*) FILTER (WHERE iv.severity = 'low' AND iv.status = 'open') as low_open,
    COUNT(*) FILTER (WHERE iv.status = 'fixed') as fixed_count,
    MAX(iv.scanned_at) as last_scanned_at
FROM images i
LEFT JOIN image_vulnerabilities iv ON iv.image_id = i.id
GROUP BY i.id, i.family, i.version, i.status;

-- View: Image deployment count
CREATE OR REPLACE VIEW v_image_deployment_summary AS
SELECT
    i.id as image_id,
    i.family,
    i.version,
    i.status,
    COUNT(*) FILTER (WHERE id.status = 'active') as active_deployments,
    COUNT(*) FILTER (WHERE id.status = 'replaced') as replaced_deployments,
    MIN(id.deployed_at) as first_deployed_at,
    MAX(id.deployed_at) as last_deployed_at
FROM images i
LEFT JOIN image_deployments id ON id.image_id = i.id
GROUP BY i.id, i.family, i.version, i.status;
