-- Migration: Remove Compliance Frameworks
-- Down migration to rollback the compliance frameworks schema

-- Drop indexes
DROP INDEX IF EXISTS idx_checks_control;
DROP INDEX IF EXISTS idx_exemptions_control;
DROP INDEX IF EXISTS idx_exemptions_org;
DROP INDEX IF EXISTS idx_assessment_results;
DROP INDEX IF EXISTS idx_assessments_org_framework;
DROP INDEX IF EXISTS idx_evidence_current;
DROP INDEX IF EXISTS idx_evidence_org_control;
DROP INDEX IF EXISTS idx_control_mappings_target;
DROP INDEX IF EXISTS idx_control_mappings_source;

-- Drop tables
DROP TABLE IF EXISTS compliance_checks;
DROP TABLE IF EXISTS compliance_exemptions;
DROP TABLE IF EXISTS compliance_policies;
DROP TABLE IF EXISTS compliance_assessment_results;
DROP TABLE IF EXISTS compliance_assessments;
DROP TABLE IF EXISTS compliance_evidence;
DROP TABLE IF EXISTS control_mappings;

-- Note: We don't drop compliance_frameworks and compliance_controls as they may have existed before
-- and contain user data. The columns we added are safe to leave.
