-- QuantumLayer Resilience Fabric - Image Lineage (Rollback)
-- Migration: 000006_add_image_lineage
-- Description: Removes golden image lineage tracking tables

-- Drop views
DROP VIEW IF EXISTS v_image_deployment_summary;
DROP VIEW IF EXISTS v_image_vuln_summary;
DROP VIEW IF EXISTS v_image_lineage_tree;

-- Drop triggers
DROP TRIGGER IF EXISTS update_image_vulnerabilities_updated_at ON image_vulnerabilities;

-- Drop tables (in reverse order of creation due to foreign keys)
DROP TABLE IF EXISTS image_tags;
DROP TABLE IF EXISTS image_components;
DROP TABLE IF EXISTS image_promotions;
DROP TABLE IF EXISTS image_deployments;
DROP TABLE IF EXISTS image_vulnerabilities;
DROP TABLE IF EXISTS image_builds;
DROP TABLE IF EXISTS image_lineage;
