-- Migration: 000002_add_connector_status (rollback)
-- Description: Remove status columns from connectors table

DROP INDEX IF EXISTS idx_connectors_status;

ALTER TABLE connectors
DROP COLUMN IF EXISTS status,
DROP COLUMN IF EXISTS status_message;
