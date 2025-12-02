-- Migration: 000002_add_connector_status
-- Description: Add status and status_message columns to connectors table

ALTER TABLE connectors
ADD COLUMN IF NOT EXISTS status VARCHAR(31) NOT NULL DEFAULT 'unknown',
ADD COLUMN IF NOT EXISTS status_message TEXT;

-- Update index to include status
CREATE INDEX IF NOT EXISTS idx_connectors_status ON connectors(status);
