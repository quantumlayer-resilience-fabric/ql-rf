-- QuantumLayer Resilience Fabric - SBOM (Software Bill of Materials)
-- Migration: 000013_add_sbom_tables
-- Description: Adds SBOM generation and management with SPDX/CycloneDX support

-- =============================================================================
-- SBOM Documents
-- =============================================================================

-- Main SBOM table storing complete SBOM documents
CREATE TABLE sboms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- Format and version
    format VARCHAR(31) NOT NULL CHECK (format IN ('spdx', 'cyclonedx')),
    version VARCHAR(31) NOT NULL, -- e.g., "SPDX-2.3", "CycloneDX-1.5"

    -- SBOM content (full document stored as JSONB)
    content JSONB NOT NULL,

    -- Metadata
    package_count INT NOT NULL DEFAULT 0,
    vuln_count INT NOT NULL DEFAULT 0,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    scanner VARCHAR(63), -- e.g., "syft", "trivy", "grype", "ql-rf"

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE sboms IS 'Software Bill of Materials (SBOM) documents for golden images';
COMMENT ON COLUMN sboms.format IS 'SBOM format: spdx (ISO/IEC 5962:2021) or cyclonedx (OWASP)';
COMMENT ON COLUMN sboms.content IS 'Full SBOM document in native format (SPDX or CycloneDX)';
COMMENT ON COLUMN sboms.scanner IS 'Tool used to generate SBOM (syft, trivy, grype, ql-rf, etc.)';

-- =============================================================================
-- SBOM Packages
-- =============================================================================

-- Normalized package data extracted from SBOMs for querying
CREATE TABLE sbom_packages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sbom_id UUID NOT NULL REFERENCES sboms(id) ON DELETE CASCADE,

    -- Package identification
    name VARCHAR(255) NOT NULL,
    version VARCHAR(127) NOT NULL,
    type VARCHAR(63) NOT NULL, -- deb, rpm, apk, npm, pip, go, maven, nuget, etc.

    -- Package URLs and identifiers
    purl VARCHAR(512), -- Package URL (standardized format)
    cpe VARCHAR(512),  -- Common Platform Enumeration

    -- Metadata
    license VARCHAR(255),
    supplier VARCHAR(255),
    checksum VARCHAR(128), -- SHA256 hash
    source_url TEXT,
    location VARCHAR(512), -- File path within image

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE sbom_packages IS 'Individual packages/components extracted from SBOMs';
COMMENT ON COLUMN sbom_packages.purl IS 'Package URL - standardized package identifier (https://github.com/package-url/purl-spec)';
COMMENT ON COLUMN sbom_packages.cpe IS 'Common Platform Enumeration - security identifier';
COMMENT ON COLUMN sbom_packages.type IS 'Package manager type (deb, rpm, apk, npm, pip, go, maven, etc.)';

-- =============================================================================
-- SBOM Dependencies
-- =============================================================================

-- Package dependency relationships
CREATE TABLE sbom_dependencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sbom_id UUID NOT NULL REFERENCES sboms(id) ON DELETE CASCADE,

    package_ref VARCHAR(255) NOT NULL, -- Package name or PURL
    depends_on VARCHAR(255) NOT NULL,  -- Dependency package name or PURL
    scope VARCHAR(31),                 -- runtime, development, test, optional

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(sbom_id, package_ref, depends_on)
);

COMMENT ON TABLE sbom_dependencies IS 'Dependency relationships between packages in an SBOM';
COMMENT ON COLUMN sbom_dependencies.scope IS 'Dependency scope: runtime, development, test, optional';

-- =============================================================================
-- SBOM Vulnerabilities
-- =============================================================================

-- Vulnerabilities associated with SBOM packages
CREATE TABLE sbom_vulnerabilities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sbom_id UUID NOT NULL REFERENCES sboms(id) ON DELETE CASCADE,
    package_id UUID NOT NULL REFERENCES sbom_packages(id) ON DELETE CASCADE,

    -- CVE details
    cve_id VARCHAR(63) NOT NULL,  -- CVE-2024-1234, GHSA-xxxx-xxxx-xxxx, etc.
    severity VARCHAR(31) NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low', 'unknown')),
    cvss_score DECIMAL(3,1) CHECK (cvss_score >= 0.0 AND cvss_score <= 10.0),
    cvss_vector VARCHAR(255), -- CVSS vector string

    -- Details
    description TEXT,
    fixed_version VARCHAR(127),
    published_date TIMESTAMPTZ,
    modified_date TIMESTAMPTZ,

    -- Additional metadata
    references JSONB, -- Array of reference URLs
    data_source VARCHAR(63), -- NVD, OSV, GitHub, Snyk, etc.
    exploit_available BOOLEAN NOT NULL DEFAULT false,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(sbom_id, package_id, cve_id)
);

COMMENT ON TABLE sbom_vulnerabilities IS 'Security vulnerabilities found in SBOM packages';
COMMENT ON COLUMN sbom_vulnerabilities.cve_id IS 'CVE ID, GHSA ID, or other vulnerability identifier';
COMMENT ON COLUMN sbom_vulnerabilities.cvss_score IS 'CVSS score (0.0 to 10.0)';
COMMENT ON COLUMN sbom_vulnerabilities.data_source IS 'Vulnerability database: NVD, OSV, GitHub, Snyk, etc.';
COMMENT ON COLUMN sbom_vulnerabilities.exploit_available IS 'Whether a public exploit exists';

-- =============================================================================
-- Indexes
-- =============================================================================

-- SBOMs
CREATE INDEX idx_sboms_image_id ON sboms(image_id);
CREATE INDEX idx_sboms_org_id ON sboms(org_id);
CREATE INDEX idx_sboms_format ON sboms(format);
CREATE INDEX idx_sboms_generated_at ON sboms(generated_at DESC);
CREATE INDEX idx_sboms_scanner ON sboms(scanner);

-- SBOM Packages
CREATE INDEX idx_sbom_packages_sbom_id ON sbom_packages(sbom_id);
CREATE INDEX idx_sbom_packages_name ON sbom_packages(name);
CREATE INDEX idx_sbom_packages_type ON sbom_packages(type);
CREATE INDEX idx_sbom_packages_purl ON sbom_packages(purl);
CREATE INDEX idx_sbom_packages_cpe ON sbom_packages(cpe);
CREATE INDEX idx_sbom_packages_name_version ON sbom_packages(name, version);

-- SBOM Dependencies
CREATE INDEX idx_sbom_deps_sbom_id ON sbom_dependencies(sbom_id);
CREATE INDEX idx_sbom_deps_package_ref ON sbom_dependencies(package_ref);
CREATE INDEX idx_sbom_deps_depends_on ON sbom_dependencies(depends_on);

-- SBOM Vulnerabilities
CREATE INDEX idx_sbom_vulns_sbom_id ON sbom_vulnerabilities(sbom_id);
CREATE INDEX idx_sbom_vulns_package_id ON sbom_vulnerabilities(package_id);
CREATE INDEX idx_sbom_vulns_cve_id ON sbom_vulnerabilities(cve_id);
CREATE INDEX idx_sbom_vulns_severity ON sbom_vulnerabilities(severity);
CREATE INDEX idx_sbom_vulns_cvss_score ON sbom_vulnerabilities(cvss_score DESC);
CREATE INDEX idx_sbom_vulns_exploit ON sbom_vulnerabilities(exploit_available) WHERE exploit_available = true;

-- =============================================================================
-- Triggers
-- =============================================================================

-- Auto-update updated_at for sboms
CREATE TRIGGER update_sboms_updated_at
    BEFORE UPDATE ON sboms
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Auto-update updated_at for vulnerabilities
CREATE TRIGGER update_sbom_vulnerabilities_updated_at
    BEFORE UPDATE ON sbom_vulnerabilities
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- Row Level Security (RLS)
-- =============================================================================

ALTER TABLE sboms ENABLE ROW LEVEL SECURITY;
ALTER TABLE sbom_packages ENABLE ROW LEVEL SECURITY;
ALTER TABLE sbom_dependencies ENABLE ROW LEVEL SECURITY;
ALTER TABLE sbom_vulnerabilities ENABLE ROW LEVEL SECURITY;

-- SBOMs: Users can only access their organization's SBOMs
CREATE POLICY sboms_org_isolation ON sboms
    FOR ALL
    USING (org_id = current_setting('app.current_org_id')::UUID);

-- SBOM Packages: Via SBOM's org_id
CREATE POLICY sbom_packages_org_isolation ON sbom_packages
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM sboms
            WHERE sboms.id = sbom_packages.sbom_id
            AND sboms.org_id = current_setting('app.current_org_id')::UUID
        )
    );

-- SBOM Dependencies: Via SBOM's org_id
CREATE POLICY sbom_dependencies_org_isolation ON sbom_dependencies
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM sboms
            WHERE sboms.id = sbom_dependencies.sbom_id
            AND sboms.org_id = current_setting('app.current_org_id')::UUID
        )
    );

-- SBOM Vulnerabilities: Via SBOM's org_id
CREATE POLICY sbom_vulnerabilities_org_isolation ON sbom_vulnerabilities
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM sboms
            WHERE sboms.id = sbom_vulnerabilities.sbom_id
            AND sboms.org_id = current_setting('app.current_org_id')::UUID
        )
    );

-- =============================================================================
-- Views
-- =============================================================================

-- SBOM Summary View with vulnerability counts
CREATE OR REPLACE VIEW v_sbom_summary AS
SELECT
    s.id,
    s.image_id,
    s.org_id,
    s.format,
    s.version,
    s.package_count,
    s.generated_at,
    s.scanner,
    COUNT(DISTINCT sp.id) as actual_package_count,
    COUNT(sv.id) FILTER (WHERE sv.severity = 'critical') as critical_vulns,
    COUNT(sv.id) FILTER (WHERE sv.severity = 'high') as high_vulns,
    COUNT(sv.id) FILTER (WHERE sv.severity = 'medium') as medium_vulns,
    COUNT(sv.id) FILTER (WHERE sv.severity = 'low') as low_vulns,
    COUNT(sv.id) as total_vulns,
    MAX(sv.cvss_score) as highest_cvss_score,
    COUNT(sv.id) FILTER (WHERE sv.exploit_available = true) as exploitable_vulns
FROM sboms s
LEFT JOIN sbom_packages sp ON sp.sbom_id = s.id
LEFT JOIN sbom_vulnerabilities sv ON sv.sbom_id = s.id
GROUP BY s.id, s.image_id, s.org_id, s.format, s.version, s.package_count, s.generated_at, s.scanner;

COMMENT ON VIEW v_sbom_summary IS 'Summary view of SBOMs with vulnerability counts';

-- Package Vulnerability Summary
CREATE OR REPLACE VIEW v_package_vulnerabilities AS
SELECT
    sp.id as package_id,
    sp.sbom_id,
    sp.name as package_name,
    sp.version as package_version,
    sp.type as package_type,
    COUNT(sv.id) as vuln_count,
    COUNT(sv.id) FILTER (WHERE sv.severity = 'critical') as critical_count,
    COUNT(sv.id) FILTER (WHERE sv.severity = 'high') as high_count,
    COUNT(sv.id) FILTER (WHERE sv.severity = 'medium') as medium_count,
    COUNT(sv.id) FILTER (WHERE sv.severity = 'low') as low_count,
    MAX(sv.cvss_score) as highest_cvss_score,
    BOOL_OR(sv.exploit_available) as has_known_exploit
FROM sbom_packages sp
LEFT JOIN sbom_vulnerabilities sv ON sv.package_id = sp.id
GROUP BY sp.id, sp.sbom_id, sp.name, sp.version, sp.type;

COMMENT ON VIEW v_package_vulnerabilities IS 'Vulnerability summary for each package';

-- Image SBOM Coverage
CREATE OR REPLACE VIEW v_image_sbom_coverage AS
SELECT
    i.id as image_id,
    i.org_id,
    i.family,
    i.version,
    i.status,
    COUNT(DISTINCT s.id) as sbom_count,
    MAX(s.generated_at) as latest_sbom_date,
    BOOL_AND(s.id IS NOT NULL) as has_sbom,
    SUM(s.package_count) as total_packages_tracked,
    SUM((SELECT COUNT(*) FROM sbom_vulnerabilities WHERE sbom_id = s.id)) as total_vulns_found
FROM images i
LEFT JOIN sboms s ON s.image_id = i.id
GROUP BY i.id, i.org_id, i.family, i.version, i.status;

COMMENT ON VIEW v_image_sbom_coverage IS 'SBOM coverage metrics for all images';
