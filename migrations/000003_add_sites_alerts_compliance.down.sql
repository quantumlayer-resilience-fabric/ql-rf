-- QuantumLayer Resilience Fabric - Rollback Sites, Alerts, Compliance Schema
-- Migration: 000003_add_sites_alerts_compliance (down)

-- Remove site_id from assets
ALTER TABLE assets DROP COLUMN IF EXISTS site_id;

-- Drop triggers
DROP TRIGGER IF EXISTS update_compliance_frameworks_updated_at ON compliance_frameworks;
DROP TRIGGER IF EXISTS update_dr_pairs_updated_at ON dr_pairs;
DROP TRIGGER IF EXISTS update_sites_updated_at ON sites;

-- Drop compliance tables
DROP TABLE IF EXISTS image_compliance;
DROP TABLE IF EXISTS compliance_results;
DROP TABLE IF EXISTS compliance_controls;
DROP TABLE IF EXISTS compliance_frameworks;

-- Drop DR pairs
DROP TABLE IF EXISTS dr_pairs;

-- Drop activities
DROP TABLE IF EXISTS activities;

-- Drop alerts
DROP TABLE IF EXISTS alerts;

-- Drop sites
DROP TABLE IF EXISTS sites;
