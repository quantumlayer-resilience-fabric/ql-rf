-- Migration Rollback: Remove InSpec Integration Tables

-- Drop views
DROP VIEW IF EXISTS inspec_compliance_scores;
DROP VIEW IF EXISTS inspec_latest_runs;

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_inspec_control_mappings_updated_at ON inspec_control_mappings;
DROP TRIGGER IF EXISTS trigger_inspec_runs_updated_at ON inspec_runs;
DROP TRIGGER IF EXISTS trigger_inspec_profiles_updated_at ON inspec_profiles;

-- Drop function
DROP FUNCTION IF EXISTS update_inspec_updated_at();

-- Drop tables (in reverse order of dependencies)
DROP TABLE IF EXISTS inspec_control_mappings;
DROP TABLE IF EXISTS inspec_results;
DROP TABLE IF EXISTS inspec_runs;
DROP TABLE IF EXISTS inspec_profiles;
