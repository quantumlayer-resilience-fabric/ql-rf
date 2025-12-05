-- Migration: Add Compliance Frameworks
-- Purpose: CIS, SOC2, NIST control mappings and evidence tracking

-- =============================================================================
-- COMPLIANCE FRAMEWORKS (Extended)
-- =============================================================================

-- Note: compliance_frameworks and compliance_controls tables may already exist
-- This migration extends them with additional fields and data

-- Make org_id nullable to allow system-level frameworks (NULL = system framework)
DO $$
BEGIN
    -- Drop the NOT NULL constraint on org_id if it exists
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'compliance_frameworks'
        AND column_name = 'org_id'
        AND is_nullable = 'NO'
    ) THEN
        ALTER TABLE compliance_frameworks ALTER COLUMN org_id DROP NOT NULL;
    END IF;
END$$;

-- Add is_system flag to distinguish system frameworks from org-specific ones
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'compliance_frameworks' AND column_name = 'is_system') THEN
        ALTER TABLE compliance_frameworks ADD COLUMN is_system BOOLEAN DEFAULT false;
    END IF;
END$$;

-- Add additional fields to compliance_frameworks if not exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'compliance_frameworks' AND column_name = 'category') THEN
        ALTER TABLE compliance_frameworks ADD COLUMN category VARCHAR(100);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'compliance_frameworks' AND column_name = 'version') THEN
        ALTER TABLE compliance_frameworks ADD COLUMN version VARCHAR(50);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'compliance_frameworks' AND column_name = 'effective_date') THEN
        ALTER TABLE compliance_frameworks ADD COLUMN effective_date DATE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'compliance_frameworks' AND column_name = 'regulatory_body') THEN
        ALTER TABLE compliance_frameworks ADD COLUMN regulatory_body VARCHAR(255);
    END IF;
END$$;

-- Add additional fields to compliance_controls if not exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'compliance_controls' AND column_name = 'control_family') THEN
        ALTER TABLE compliance_controls ADD COLUMN control_family VARCHAR(100);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'compliance_controls' AND column_name = 'implementation_guidance') THEN
        ALTER TABLE compliance_controls ADD COLUMN implementation_guidance TEXT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'compliance_controls' AND column_name = 'assessment_procedure') THEN
        ALTER TABLE compliance_controls ADD COLUMN assessment_procedure TEXT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'compliance_controls' AND column_name = 'automation_support') THEN
        ALTER TABLE compliance_controls ADD COLUMN automation_support VARCHAR(50);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'compliance_controls' AND column_name = 'priority') THEN
        ALTER TABLE compliance_controls ADD COLUMN priority VARCHAR(20);
    END IF;
END$$;

-- =============================================================================
-- CONTROL MAPPINGS (Cross-Framework)
-- =============================================================================

-- Map controls across different frameworks
CREATE TABLE IF NOT EXISTS control_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_control_id UUID NOT NULL REFERENCES compliance_controls(id) ON DELETE CASCADE,
    target_control_id UUID NOT NULL REFERENCES compliance_controls(id) ON DELETE CASCADE,
    mapping_type VARCHAR(50) NOT NULL, -- equivalent, partial, related
    confidence_score DECIMAL(3, 2) DEFAULT 1.0, -- 0.0 to 1.0
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(source_control_id, target_control_id)
);

CREATE INDEX idx_control_mappings_source ON control_mappings(source_control_id);
CREATE INDEX idx_control_mappings_target ON control_mappings(target_control_id);

-- =============================================================================
-- COMPLIANCE EVIDENCE
-- =============================================================================

-- Evidence items that satisfy controls
CREATE TABLE IF NOT EXISTS compliance_evidence (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    control_id UUID NOT NULL REFERENCES compliance_controls(id) ON DELETE CASCADE,

    -- Evidence details
    evidence_type VARCHAR(50) NOT NULL, -- screenshot, log, config, report, attestation
    title VARCHAR(255) NOT NULL,
    description TEXT,

    -- Storage
    storage_type VARCHAR(50) NOT NULL, -- s3, gcs, azure_blob, internal
    storage_path VARCHAR(1024),
    content_hash VARCHAR(128), -- SHA-256 of content
    file_size_bytes BIGINT,
    mime_type VARCHAR(100),

    -- Metadata
    collected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    collected_by VARCHAR(255),
    collection_method VARCHAR(50), -- automated, manual

    -- Validity
    valid_from TIMESTAMPTZ DEFAULT NOW(),
    valid_until TIMESTAMPTZ,
    is_current BOOLEAN DEFAULT TRUE,

    -- Review
    reviewed_by VARCHAR(255),
    reviewed_at TIMESTAMPTZ,
    review_status VARCHAR(50), -- pending, approved, rejected
    review_notes TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_evidence_org_control ON compliance_evidence(org_id, control_id);
CREATE INDEX idx_evidence_current ON compliance_evidence(org_id, is_current) WHERE is_current = TRUE;

-- =============================================================================
-- COMPLIANCE ASSESSMENTS
-- =============================================================================

-- Assessment runs
CREATE TABLE IF NOT EXISTS compliance_assessments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    framework_id UUID NOT NULL REFERENCES compliance_frameworks(id),

    -- Assessment info
    assessment_type VARCHAR(50) NOT NULL, -- automated, manual, hybrid
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Scope
    scope_sites UUID[], -- NULL = all sites
    scope_assets UUID[], -- NULL = all assets

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, in_progress, completed, failed
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Results summary
    total_controls INTEGER DEFAULT 0,
    passed_controls INTEGER DEFAULT 0,
    failed_controls INTEGER DEFAULT 0,
    not_applicable INTEGER DEFAULT 0,
    score DECIMAL(5, 2),

    -- Initiated by
    initiated_by VARCHAR(255) NOT NULL,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_assessments_org_framework ON compliance_assessments(org_id, framework_id);

-- Individual control results from an assessment
CREATE TABLE IF NOT EXISTS compliance_assessment_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assessment_id UUID NOT NULL REFERENCES compliance_assessments(id) ON DELETE CASCADE,
    control_id UUID NOT NULL REFERENCES compliance_controls(id),

    -- Result
    status VARCHAR(50) NOT NULL, -- passed, failed, not_applicable, manual_review
    score DECIMAL(5, 2), -- 0-100

    -- Details
    findings TEXT,
    remediation_guidance TEXT,
    evidence_ids UUID[], -- References to compliance_evidence

    -- For automated checks
    check_output JSONB,
    check_duration_ms INTEGER,

    evaluated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_assessment_results ON compliance_assessment_results(assessment_id);

-- =============================================================================
-- COMPLIANCE POLICIES
-- =============================================================================

-- Organization-specific compliance policies
CREATE TABLE IF NOT EXISTS compliance_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    framework_id UUID NOT NULL REFERENCES compliance_frameworks(id),

    -- Policy info
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Schedule
    assessment_schedule VARCHAR(100), -- cron expression
    next_assessment_at TIMESTAMPTZ,

    -- Notifications
    notify_on_failure BOOLEAN DEFAULT TRUE,
    notification_channels JSONB DEFAULT '[]', -- email, slack, webhook

    -- Thresholds
    minimum_score DECIMAL(5, 2) DEFAULT 80.0,
    critical_controls UUID[], -- Controls that must pass

    enabled BOOLEAN DEFAULT TRUE,
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(org_id, framework_id, name)
);

-- =============================================================================
-- COMPLIANCE EXEMPTIONS
-- =============================================================================

-- Exemptions/exceptions for specific controls
CREATE TABLE IF NOT EXISTS compliance_exemptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    control_id UUID NOT NULL REFERENCES compliance_controls(id),

    -- Scope
    asset_id UUID, -- NULL = org-wide exemption
    site_id UUID,

    -- Exemption details
    reason TEXT NOT NULL,
    risk_acceptance TEXT,
    compensating_controls TEXT,

    -- Validity
    approved_by VARCHAR(255) NOT NULL,
    approved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,

    -- Review
    last_reviewed_at TIMESTAMPTZ,
    last_reviewed_by VARCHAR(255),
    review_frequency_days INTEGER DEFAULT 90,

    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, expired, revoked

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_exemptions_org ON compliance_exemptions(org_id, status);
CREATE INDEX idx_exemptions_control ON compliance_exemptions(control_id);

-- =============================================================================
-- AUTOMATED CHECK DEFINITIONS
-- =============================================================================

-- Automated checks that can be run for controls
CREATE TABLE IF NOT EXISTS compliance_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    control_id UUID NOT NULL REFERENCES compliance_controls(id) ON DELETE CASCADE,

    -- Check definition
    name VARCHAR(255) NOT NULL,
    description TEXT,
    check_type VARCHAR(50) NOT NULL, -- script, api, config, query

    -- Implementation
    check_definition JSONB NOT NULL, -- Script, query, or API call definition
    expected_result JSONB, -- What constitutes a pass

    -- Targeting
    applies_to_platforms VARCHAR(50)[], -- aws, azure, gcp, vsphere, k8s
    applies_to_asset_types VARCHAR(50)[], -- vm, container, database, etc.

    -- Execution
    timeout_seconds INTEGER DEFAULT 60,
    requires_agent BOOLEAN DEFAULT FALSE,

    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_checks_control ON compliance_checks(control_id);

-- =============================================================================
-- INSERT CIS BENCHMARK FRAMEWORK
-- =============================================================================

INSERT INTO compliance_frameworks (name, description, category, version, regulatory_body, is_system)
VALUES
('CIS AWS Foundations', 'CIS Amazon Web Services Foundations Benchmark', 'Cloud Security', 'v1.5.0', 'Center for Internet Security', true)
ON CONFLICT DO NOTHING;

INSERT INTO compliance_frameworks (name, description, category, version, regulatory_body, is_system)
VALUES
('CIS Azure Foundations', 'CIS Microsoft Azure Foundations Benchmark', 'Cloud Security', 'v2.0.0', 'Center for Internet Security', true)
ON CONFLICT DO NOTHING;

INSERT INTO compliance_frameworks (name, description, category, version, regulatory_body, is_system)
VALUES
('CIS GCP Foundations', 'CIS Google Cloud Platform Foundation Benchmark', 'Cloud Security', 'v2.0.0', 'Center for Internet Security', true)
ON CONFLICT DO NOTHING;

INSERT INTO compliance_frameworks (name, description, category, version, regulatory_body, is_system)
VALUES
('CIS Kubernetes', 'CIS Kubernetes Benchmark', 'Container Security', 'v1.8.0', 'Center for Internet Security', true)
ON CONFLICT DO NOTHING;

-- =============================================================================
-- INSERT SOC 2 FRAMEWORK
-- =============================================================================

INSERT INTO compliance_frameworks (name, description, category, version, regulatory_body, is_system)
VALUES
('SOC 2 Type II', 'Service Organization Control 2 Type II', 'Security & Privacy', '2017', 'AICPA', true)
ON CONFLICT DO NOTHING;

-- =============================================================================
-- INSERT NIST FRAMEWORKS
-- =============================================================================

INSERT INTO compliance_frameworks (name, description, category, version, regulatory_body, is_system)
VALUES
('NIST CSF', 'NIST Cybersecurity Framework', 'Cybersecurity', 'v1.1', 'National Institute of Standards and Technology', true)
ON CONFLICT DO NOTHING;

INSERT INTO compliance_frameworks (name, description, category, version, regulatory_body, is_system)
VALUES
('NIST 800-53', 'Security and Privacy Controls for Information Systems', 'Security Controls', 'Rev 5', 'National Institute of Standards and Technology', true)
ON CONFLICT DO NOTHING;

-- =============================================================================
-- INSERT SAMPLE CIS CONTROLS
-- =============================================================================

-- Get CIS AWS framework ID
DO $$
DECLARE
    v_framework_id UUID;
BEGIN
    SELECT id INTO v_framework_id FROM compliance_frameworks WHERE name = 'CIS AWS Foundations' LIMIT 1;

    IF v_framework_id IS NOT NULL THEN
        -- IAM Controls
        INSERT INTO compliance_controls (framework_id, control_id, title, description, severity, control_family, automation_support, priority)
        VALUES
        (v_framework_id, '1.1', 'Maintain current contact details', 'Ensure contact email and telephone details are current', 'low', 'IAM', 'manual', 'P3'),
        (v_framework_id, '1.4', 'Ensure no root account access key exists', 'The root account is the most privileged user in AWS', 'critical', 'IAM', 'automated', 'P1'),
        (v_framework_id, '1.5', 'Ensure MFA is enabled for root account', 'Multi-factor authentication adds extra protection', 'critical', 'IAM', 'automated', 'P1'),
        (v_framework_id, '1.10', 'Ensure multi-factor authentication is enabled for all IAM users', 'MFA provides extra security for IAM users', 'high', 'IAM', 'automated', 'P1'),
        (v_framework_id, '1.14', 'Ensure access keys are rotated every 90 days', 'Rotating access keys reduces the risk of compromised keys', 'medium', 'IAM', 'automated', 'P2'),
        (v_framework_id, '1.16', 'Ensure IAM policies are attached only to groups or roles', 'Attaching policies to groups/roles is more manageable', 'medium', 'IAM', 'automated', 'P2')
        ON CONFLICT DO NOTHING;

        -- Logging Controls
        INSERT INTO compliance_controls (framework_id, control_id, title, description, severity, control_family, automation_support, priority)
        VALUES
        (v_framework_id, '3.1', 'Ensure CloudTrail is enabled in all regions', 'CloudTrail provides event history of AWS account activity', 'high', 'Logging', 'automated', 'P1'),
        (v_framework_id, '3.2', 'Ensure CloudTrail log file validation is enabled', 'Validates log file integrity', 'medium', 'Logging', 'automated', 'P2'),
        (v_framework_id, '3.4', 'Ensure CloudTrail trails are integrated with CloudWatch Logs', 'Enables real-time monitoring', 'medium', 'Logging', 'automated', 'P2'),
        (v_framework_id, '3.7', 'Ensure CloudTrail logs are encrypted at rest using KMS CMKs', 'Adds additional protection for log data', 'high', 'Logging', 'automated', 'P1')
        ON CONFLICT DO NOTHING;

        -- Networking Controls
        INSERT INTO compliance_controls (framework_id, control_id, title, description, severity, control_family, automation_support, priority)
        VALUES
        (v_framework_id, '5.1', 'Ensure no Network ACLs allow ingress from 0.0.0.0/0 to SSH', 'Restrict SSH access', 'high', 'Networking', 'automated', 'P1'),
        (v_framework_id, '5.2', 'Ensure no security groups allow ingress from 0.0.0.0/0 to port 3389', 'Restrict RDP access', 'high', 'Networking', 'automated', 'P1'),
        (v_framework_id, '5.3', 'Ensure the default security group restricts all traffic', 'Default SG should not allow traffic', 'medium', 'Networking', 'automated', 'P2')
        ON CONFLICT DO NOTHING;
    END IF;
END$$;

-- =============================================================================
-- INSERT SAMPLE SOC 2 CONTROLS
-- =============================================================================

DO $$
DECLARE
    v_framework_id UUID;
BEGIN
    SELECT id INTO v_framework_id FROM compliance_frameworks WHERE name = 'SOC 2 Type II' LIMIT 1;

    IF v_framework_id IS NOT NULL THEN
        -- Security Controls (CC)
        INSERT INTO compliance_controls (framework_id, control_id, title, description, severity, control_family, automation_support, priority)
        VALUES
        (v_framework_id, 'CC1.1', 'Control Environment', 'The entity demonstrates a commitment to integrity and ethical values', 'high', 'Security', 'manual', 'P1'),
        (v_framework_id, 'CC2.1', 'Information and Communication', 'Information about entity objectives is communicated', 'medium', 'Security', 'manual', 'P2'),
        (v_framework_id, 'CC3.1', 'Risk Assessment', 'Entity specifies objectives with sufficient clarity', 'high', 'Security', 'manual', 'P1'),
        (v_framework_id, 'CC4.1', 'Monitoring Activities', 'Entity selects and develops ongoing monitoring', 'high', 'Security', 'automated', 'P1'),
        (v_framework_id, 'CC5.1', 'Control Activities', 'Entity selects and develops control activities', 'high', 'Security', 'hybrid', 'P1'),
        (v_framework_id, 'CC6.1', 'Logical and Physical Access', 'Entity implements access controls to protect information', 'critical', 'Security', 'automated', 'P1'),
        (v_framework_id, 'CC6.6', 'System Boundaries', 'Entity implements boundary protection for systems', 'high', 'Security', 'automated', 'P1'),
        (v_framework_id, 'CC6.7', 'Data in Transit', 'Entity protects data in transit', 'high', 'Security', 'automated', 'P1'),
        (v_framework_id, 'CC7.1', 'System Operations', 'Entity uses detection and monitoring to identify anomalies', 'high', 'Security', 'automated', 'P1'),
        (v_framework_id, 'CC7.2', 'Incident Response', 'Entity responds to identified security incidents', 'critical', 'Security', 'hybrid', 'P1'),
        (v_framework_id, 'CC8.1', 'Change Management', 'Entity authorizes and implements changes', 'high', 'Security', 'hybrid', 'P1'),
        (v_framework_id, 'CC9.1', 'Risk Mitigation', 'Entity identifies and mitigates risks', 'high', 'Security', 'hybrid', 'P1')
        ON CONFLICT DO NOTHING;

        -- Availability Controls (A)
        INSERT INTO compliance_controls (framework_id, control_id, title, description, severity, control_family, automation_support, priority)
        VALUES
        (v_framework_id, 'A1.1', 'System Availability', 'Entity maintains system availability to meet objectives', 'high', 'Availability', 'automated', 'P1'),
        (v_framework_id, 'A1.2', 'Capacity Management', 'Entity plans for capacity to meet availability goals', 'medium', 'Availability', 'automated', 'P2'),
        (v_framework_id, 'A1.3', 'Recovery Operations', 'Entity tests backup and recovery procedures', 'high', 'Availability', 'hybrid', 'P1')
        ON CONFLICT DO NOTHING;

        -- Confidentiality Controls (C)
        INSERT INTO compliance_controls (framework_id, control_id, title, description, severity, control_family, automation_support, priority)
        VALUES
        (v_framework_id, 'C1.1', 'Confidential Information', 'Entity identifies confidential information', 'high', 'Confidentiality', 'hybrid', 'P1'),
        (v_framework_id, 'C1.2', 'Confidential Information Disposal', 'Entity disposes of confidential information properly', 'high', 'Confidentiality', 'hybrid', 'P1')
        ON CONFLICT DO NOTHING;
    END IF;
END$$;

-- =============================================================================
-- INSERT SAMPLE NIST CSF CONTROLS
-- =============================================================================

DO $$
DECLARE
    v_framework_id UUID;
BEGIN
    SELECT id INTO v_framework_id FROM compliance_frameworks WHERE name = 'NIST CSF' LIMIT 1;

    IF v_framework_id IS NOT NULL THEN
        -- Identify Function
        INSERT INTO compliance_controls (framework_id, control_id, title, description, severity, control_family, automation_support, priority)
        VALUES
        (v_framework_id, 'ID.AM-1', 'Asset Management', 'Physical devices and systems are inventoried', 'high', 'Identify', 'automated', 'P1'),
        (v_framework_id, 'ID.AM-2', 'Software Inventory', 'Software platforms and applications are inventoried', 'high', 'Identify', 'automated', 'P1'),
        (v_framework_id, 'ID.RA-1', 'Risk Assessment', 'Asset vulnerabilities are identified and documented', 'high', 'Identify', 'automated', 'P1'),
        (v_framework_id, 'ID.RA-5', 'Risk Assessment', 'Threats and vulnerabilities are used to identify risk', 'high', 'Identify', 'hybrid', 'P1')
        ON CONFLICT DO NOTHING;

        -- Protect Function
        INSERT INTO compliance_controls (framework_id, control_id, title, description, severity, control_family, automation_support, priority)
        VALUES
        (v_framework_id, 'PR.AC-1', 'Access Control', 'Identities and credentials are managed', 'critical', 'Protect', 'automated', 'P1'),
        (v_framework_id, 'PR.AC-3', 'Access Control', 'Remote access is managed', 'high', 'Protect', 'automated', 'P1'),
        (v_framework_id, 'PR.AC-4', 'Access Control', 'Access permissions are managed with least privilege', 'high', 'Protect', 'automated', 'P1'),
        (v_framework_id, 'PR.DS-1', 'Data Security', 'Data-at-rest is protected', 'high', 'Protect', 'automated', 'P1'),
        (v_framework_id, 'PR.DS-2', 'Data Security', 'Data-in-transit is protected', 'high', 'Protect', 'automated', 'P1'),
        (v_framework_id, 'PR.IP-1', 'Information Protection', 'Baseline configuration is maintained', 'high', 'Protect', 'automated', 'P1'),
        (v_framework_id, 'PR.IP-9', 'Information Protection', 'Response and recovery plans are in place', 'high', 'Protect', 'manual', 'P1'),
        (v_framework_id, 'PR.MA-1', 'Maintenance', 'Maintenance is performed and logged', 'medium', 'Protect', 'hybrid', 'P2'),
        (v_framework_id, 'PR.PT-1', 'Protective Technology', 'Audit logs are determined and reviewed', 'high', 'Protect', 'automated', 'P1')
        ON CONFLICT DO NOTHING;

        -- Detect Function
        INSERT INTO compliance_controls (framework_id, control_id, title, description, severity, control_family, automation_support, priority)
        VALUES
        (v_framework_id, 'DE.AE-1', 'Anomalies and Events', 'Baseline of operations is established', 'high', 'Detect', 'automated', 'P1'),
        (v_framework_id, 'DE.CM-1', 'Continuous Monitoring', 'Network is monitored for security events', 'high', 'Detect', 'automated', 'P1'),
        (v_framework_id, 'DE.CM-4', 'Continuous Monitoring', 'Malicious code is detected', 'high', 'Detect', 'automated', 'P1'),
        (v_framework_id, 'DE.CM-7', 'Continuous Monitoring', 'Unauthorized activity is monitored', 'high', 'Detect', 'automated', 'P1')
        ON CONFLICT DO NOTHING;

        -- Respond Function
        INSERT INTO compliance_controls (framework_id, control_id, title, description, severity, control_family, automation_support, priority)
        VALUES
        (v_framework_id, 'RS.RP-1', 'Response Planning', 'Response plan is executed during incidents', 'critical', 'Respond', 'hybrid', 'P1'),
        (v_framework_id, 'RS.CO-1', 'Communications', 'Personnel know their incident response roles', 'high', 'Respond', 'manual', 'P1'),
        (v_framework_id, 'RS.AN-1', 'Analysis', 'Notifications from detection systems are investigated', 'high', 'Respond', 'hybrid', 'P1'),
        (v_framework_id, 'RS.MI-1', 'Mitigation', 'Incidents are contained', 'critical', 'Respond', 'hybrid', 'P1')
        ON CONFLICT DO NOTHING;

        -- Recover Function
        INSERT INTO compliance_controls (framework_id, control_id, title, description, severity, control_family, automation_support, priority)
        VALUES
        (v_framework_id, 'RC.RP-1', 'Recovery Planning', 'Recovery plan is executed during incidents', 'critical', 'Recover', 'hybrid', 'P1'),
        (v_framework_id, 'RC.IM-1', 'Improvements', 'Recovery plans incorporate lessons learned', 'medium', 'Recover', 'manual', 'P2'),
        (v_framework_id, 'RC.CO-1', 'Communications', 'Public relations are managed', 'medium', 'Recover', 'manual', 'P2')
        ON CONFLICT DO NOTHING;
    END IF;
END$$;

-- =============================================================================
-- CREATE SAMPLE CROSS-FRAMEWORK MAPPINGS
-- =============================================================================

-- Map CIS AWS controls to NIST CSF
DO $$
DECLARE
    v_cis_framework_id UUID;
    v_nist_framework_id UUID;
    v_source_id UUID;
    v_target_id UUID;
BEGIN
    SELECT id INTO v_cis_framework_id FROM compliance_frameworks WHERE name = 'CIS AWS Foundations' LIMIT 1;
    SELECT id INTO v_nist_framework_id FROM compliance_frameworks WHERE name = 'NIST CSF' LIMIT 1;

    IF v_cis_framework_id IS NOT NULL AND v_nist_framework_id IS NOT NULL THEN
        -- Map CIS 1.4 (no root access key) to NIST PR.AC-1 (credential management)
        SELECT id INTO v_source_id FROM compliance_controls WHERE framework_id = v_cis_framework_id AND control_id = '1.4';
        SELECT id INTO v_target_id FROM compliance_controls WHERE framework_id = v_nist_framework_id AND control_id = 'PR.AC-1';
        IF v_source_id IS NOT NULL AND v_target_id IS NOT NULL THEN
            INSERT INTO control_mappings (source_control_id, target_control_id, mapping_type, confidence_score)
            VALUES (v_source_id, v_target_id, 'related', 0.9)
            ON CONFLICT DO NOTHING;
        END IF;

        -- Map CIS 3.1 (CloudTrail) to NIST PR.PT-1 (audit logs)
        SELECT id INTO v_source_id FROM compliance_controls WHERE framework_id = v_cis_framework_id AND control_id = '3.1';
        SELECT id INTO v_target_id FROM compliance_controls WHERE framework_id = v_nist_framework_id AND control_id = 'PR.PT-1';
        IF v_source_id IS NOT NULL AND v_target_id IS NOT NULL THEN
            INSERT INTO control_mappings (source_control_id, target_control_id, mapping_type, confidence_score)
            VALUES (v_source_id, v_target_id, 'equivalent', 0.95)
            ON CONFLICT DO NOTHING;
        END IF;

        -- Map CIS 5.1 (SSH access) to NIST PR.AC-3 (remote access)
        SELECT id INTO v_source_id FROM compliance_controls WHERE framework_id = v_cis_framework_id AND control_id = '5.1';
        SELECT id INTO v_target_id FROM compliance_controls WHERE framework_id = v_nist_framework_id AND control_id = 'PR.AC-3';
        IF v_source_id IS NOT NULL AND v_target_id IS NOT NULL THEN
            INSERT INTO control_mappings (source_control_id, target_control_id, mapping_type, confidence_score)
            VALUES (v_source_id, v_target_id, 'related', 0.85)
            ON CONFLICT DO NOTHING;
        END IF;
    END IF;
END$$;

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE control_mappings IS 'Cross-framework control mappings for unified compliance view';
COMMENT ON TABLE compliance_evidence IS 'Evidence items that satisfy compliance controls';
COMMENT ON TABLE compliance_assessments IS 'Compliance assessment runs and results';
COMMENT ON TABLE compliance_policies IS 'Organization-specific compliance policies';
COMMENT ON TABLE compliance_exemptions IS 'Control exemptions with risk acceptance';
COMMENT ON TABLE compliance_checks IS 'Automated check definitions for controls';
