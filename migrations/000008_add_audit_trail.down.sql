-- Rollback: Remove Enterprise Audit Trail

DROP TABLE IF EXISTS resource_access_logs;
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS api_key_usage;
DROP TABLE IF EXISTS audit_export_destinations;
DROP TABLE IF EXISTS audit_export_queue;

DROP TRIGGER IF EXISTS audit_logs_queue_export ON audit_logs;
DROP TRIGGER IF EXISTS audit_logs_calculate_hash ON audit_logs;
DROP TRIGGER IF EXISTS audit_logs_immutable_delete ON audit_logs;
DROP TRIGGER IF EXISTS audit_logs_immutable_update ON audit_logs;

DROP FUNCTION IF EXISTS queue_audit_export();
DROP FUNCTION IF EXISTS calculate_audit_hash();
DROP FUNCTION IF EXISTS prevent_audit_modification();

DROP TABLE IF EXISTS audit_logs;
