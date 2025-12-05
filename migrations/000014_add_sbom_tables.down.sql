-- QuantumLayer Resilience Fabric - SBOM Rollback
-- Migration: 000013_add_sbom_tables (DOWN)
-- Description: Removes SBOM tables and related objects

-- Drop views
DROP VIEW IF EXISTS v_image_sbom_coverage;
DROP VIEW IF EXISTS v_package_vulnerabilities;
DROP VIEW IF EXISTS v_sbom_summary;

-- Drop tables (in reverse order of dependencies)
DROP TABLE IF EXISTS sbom_vulnerabilities;
DROP TABLE IF EXISTS sbom_dependencies;
DROP TABLE IF EXISTS sbom_packages;
DROP TABLE IF EXISTS sboms;
