-- Migration: Add InSpec Integration Tables
-- Purpose: Support InSpec profile execution and compliance assessment

-- =============================================================================
-- INSPEC PROFILES
-- =============================================================================

-- InSpec profiles that can be run against assets
CREATE TABLE IF NOT EXISTS inspec_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    title VARCHAR(500) NOT NULL,
    maintainer VARCHAR(255),
    summary TEXT,
    framework_id UUID NOT NULL REFERENCES compliance_frameworks(id) ON DELETE CASCADE,
    profile_url VARCHAR(1024), -- Git URL or Chef Supermarket URL
    platforms VARCHAR(100)[], -- linux, windows, aws, azure, gcp, k8s
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(name, version)
);

CREATE INDEX idx_inspec_profiles_framework ON inspec_profiles(framework_id);
CREATE INDEX idx_inspec_profiles_name ON inspec_profiles(name);

-- =============================================================================
-- INSPEC RUNS
-- =============================================================================

-- InSpec profile execution runs
CREATE TABLE IF NOT EXISTS inspec_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES inspec_profiles(id) ON DELETE CASCADE,

    -- Execution status
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, running, completed, failed, cancelled
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration INTEGER, -- Duration in seconds

    -- Results summary
    total_tests INTEGER DEFAULT 0,
    passed_tests INTEGER DEFAULT 0,
    failed_tests INTEGER DEFAULT 0,
    skipped_tests INTEGER DEFAULT 0,

    -- Error handling
    error_message TEXT,
    raw_output TEXT, -- Store raw JSON output for debugging

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_inspec_runs_org ON inspec_runs(org_id);
CREATE INDEX idx_inspec_runs_asset ON inspec_runs(asset_id);
CREATE INDEX idx_inspec_runs_profile ON inspec_runs(profile_id);
CREATE INDEX idx_inspec_runs_status ON inspec_runs(status);
CREATE INDEX idx_inspec_runs_created ON inspec_runs(created_at DESC);

-- =============================================================================
-- INSPEC RESULTS
-- =============================================================================

-- Individual control results from InSpec runs
CREATE TABLE IF NOT EXISTS inspec_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES inspec_runs(id) ON DELETE CASCADE,

    -- Control information
    control_id VARCHAR(255) NOT NULL, -- InSpec control ID
    control_title VARCHAR(500),

    -- Result
    status VARCHAR(50) NOT NULL, -- passed, failed, skipped, error
    message TEXT,
    resource VARCHAR(500), -- Resource being tested (e.g., file path, AWS resource ARN)

    -- Metadata
    source_location VARCHAR(1024), -- Source file and line number
    run_time DECIMAL(10, 6), -- Execution time in seconds
    code_description TEXT, -- Human-readable description of what was tested

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_inspec_results_run ON inspec_results(run_id);
CREATE INDEX idx_inspec_results_status ON inspec_results(status);
CREATE INDEX idx_inspec_results_control ON inspec_results(control_id);

-- =============================================================================
-- INSPEC CONTROL MAPPINGS
-- =============================================================================

-- Map InSpec controls to compliance framework controls
CREATE TABLE IF NOT EXISTS inspec_control_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inspec_control_id VARCHAR(255) NOT NULL, -- InSpec control ID
    compliance_control_id UUID NOT NULL REFERENCES compliance_controls(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES inspec_profiles(id) ON DELETE CASCADE,

    -- Mapping metadata
    mapping_confidence DECIMAL(3, 2) DEFAULT 1.0, -- 0.0 to 1.0, indicates confidence in mapping
    notes TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(profile_id, inspec_control_id, compliance_control_id)
);

CREATE INDEX idx_inspec_mappings_profile ON inspec_control_mappings(profile_id);
CREATE INDEX idx_inspec_mappings_compliance ON inspec_control_mappings(compliance_control_id);
CREATE INDEX idx_inspec_mappings_inspec_control ON inspec_control_mappings(inspec_control_id);

-- =============================================================================
-- TRIGGERS FOR UPDATED_AT
-- =============================================================================

CREATE OR REPLACE FUNCTION update_inspec_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_inspec_profiles_updated_at
    BEFORE UPDATE ON inspec_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_inspec_updated_at();

CREATE TRIGGER trigger_inspec_runs_updated_at
    BEFORE UPDATE ON inspec_runs
    FOR EACH ROW
    EXECUTE FUNCTION update_inspec_updated_at();

CREATE TRIGGER trigger_inspec_control_mappings_updated_at
    BEFORE UPDATE ON inspec_control_mappings
    FOR EACH ROW
    EXECUTE FUNCTION update_inspec_updated_at();

-- =============================================================================
-- VIEWS FOR REPORTING
-- =============================================================================

-- View for latest run per asset-profile combination
CREATE OR REPLACE VIEW inspec_latest_runs AS
SELECT DISTINCT ON (r.asset_id, r.profile_id)
    r.id,
    r.org_id,
    r.asset_id,
    r.profile_id,
    r.status,
    r.started_at,
    r.completed_at,
    r.duration,
    r.total_tests,
    r.passed_tests,
    r.failed_tests,
    r.skipped_tests,
    CASE
        WHEN r.total_tests > 0 THEN ROUND((r.passed_tests::DECIMAL / r.total_tests::DECIMAL) * 100, 2)
        ELSE 0
    END as pass_rate,
    a.name as asset_name,
    a.platform as asset_platform,
    p.name as profile_name,
    p.title as profile_title,
    f.name as framework_name
FROM inspec_runs r
JOIN assets a ON r.asset_id = a.id
JOIN inspec_profiles p ON r.profile_id = p.id
JOIN compliance_frameworks f ON p.framework_id = f.id
ORDER BY r.asset_id, r.profile_id, r.created_at DESC;

-- View for compliance score by framework from InSpec runs
CREATE OR REPLACE VIEW inspec_compliance_scores AS
SELECT
    r.org_id,
    p.framework_id,
    f.name as framework_name,
    COUNT(DISTINCT r.asset_id) as assets_assessed,
    COUNT(r.id) as total_runs,
    SUM(r.total_tests) as total_tests,
    SUM(r.passed_tests) as total_passed,
    SUM(r.failed_tests) as total_failed,
    SUM(r.skipped_tests) as total_skipped,
    CASE
        WHEN SUM(r.total_tests) > 0 THEN ROUND((SUM(r.passed_tests)::DECIMAL / SUM(r.total_tests)::DECIMAL) * 100, 2)
        ELSE 0
    END as overall_pass_rate,
    MAX(r.completed_at) as last_assessment_at
FROM inspec_runs r
JOIN inspec_profiles p ON r.profile_id = p.id
JOIN compliance_frameworks f ON p.framework_id = f.id
WHERE r.status = 'completed'
AND r.completed_at > NOW() - INTERVAL '90 days'
GROUP BY r.org_id, p.framework_id, f.name;

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE inspec_profiles IS 'InSpec profiles that can be run against assets for compliance assessment';
COMMENT ON TABLE inspec_runs IS 'InSpec profile execution runs with results summary';
COMMENT ON TABLE inspec_results IS 'Individual control results from InSpec runs';
COMMENT ON TABLE inspec_control_mappings IS 'Mappings between InSpec controls and compliance framework controls';

COMMENT ON VIEW inspec_latest_runs IS 'Latest InSpec run for each asset-profile combination';
COMMENT ON VIEW inspec_compliance_scores IS 'Compliance scores aggregated from InSpec runs by framework';
