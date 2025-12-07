-- Rollback: Remove connector scheduling support

DROP TRIGGER IF EXISTS connector_schedule_update ON connectors;
DROP FUNCTION IF EXISTS update_connector_next_sync();
DROP TABLE IF EXISTS connector_sync_history;
ALTER TABLE connectors
DROP COLUMN IF EXISTS sync_schedule,
DROP COLUMN IF EXISTS sync_enabled,
DROP COLUMN IF EXISTS next_sync_at,
DROP COLUMN IF EXISTS sync_timeout_seconds;
