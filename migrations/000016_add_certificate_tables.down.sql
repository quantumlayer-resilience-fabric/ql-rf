-- Rollback certificate management tables

-- Drop triggers first
DROP TRIGGER IF EXISTS trg_certificate_status ON certificates;
DROP TRIGGER IF EXISTS trg_cert_usage_updated ON certificate_usage;
DROP TRIGGER IF EXISTS trg_cert_rotations_updated ON certificate_rotations;
DROP TRIGGER IF EXISTS trg_cert_alerts_updated ON certificate_alerts;

-- Drop functions
DROP FUNCTION IF EXISTS update_certificate_status();
DROP FUNCTION IF EXISTS update_certificate_timestamps();

-- Drop views
DROP VIEW IF EXISTS v_certificate_summary;
DROP VIEW IF EXISTS v_certificate_blast_radius;
DROP VIEW IF EXISTS v_expiring_certificates;

-- Drop tables in dependency order
DROP TABLE IF EXISTS certificate_scan_jobs;
DROP TABLE IF EXISTS certificate_alerts;
DROP TABLE IF EXISTS certificate_rotations;
DROP TABLE IF EXISTS certificate_usage;
DROP TABLE IF EXISTS certificates;
