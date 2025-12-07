-- Migration: Add scheduling support to connectors table
-- This allows per-connector sync schedules instead of only global config

-- Add sync_schedule column (cron expression or interval like "1h", "30m", "6h")
ALTER TABLE connectors
ADD COLUMN IF NOT EXISTS sync_schedule VARCHAR(100) DEFAULT '1h',
ADD COLUMN IF NOT EXISTS sync_enabled BOOLEAN DEFAULT TRUE,
ADD COLUMN IF NOT EXISTS next_sync_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS sync_timeout_seconds INT DEFAULT 300;

-- Add index for finding connectors that need to be synced
CREATE INDEX IF NOT EXISTS idx_connectors_next_sync
ON connectors(next_sync_at)
WHERE enabled = TRUE AND sync_enabled = TRUE;

-- Add sync history table for tracking sync runs
CREATE TABLE IF NOT EXISTS connector_sync_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connector_id UUID NOT NULL REFERENCES connectors(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- Sync timing
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_ms INT,

    -- Results
    status VARCHAR(20) NOT NULL DEFAULT 'running', -- running, completed, failed, timeout
    assets_discovered INT DEFAULT 0,
    assets_created INT DEFAULT 0,
    assets_updated INT DEFAULT 0,
    assets_removed INT DEFAULT 0,
    images_discovered INT DEFAULT 0,

    -- Error tracking
    error_message TEXT,
    error_code VARCHAR(50),

    -- Metadata
    trigger_type VARCHAR(20) NOT NULL DEFAULT 'scheduled', -- scheduled, manual, webhook
    metadata JSONB DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for sync history
CREATE INDEX IF NOT EXISTS idx_sync_history_connector ON connector_sync_history(connector_id);
CREATE INDEX IF NOT EXISTS idx_sync_history_org ON connector_sync_history(org_id);
CREATE INDEX IF NOT EXISTS idx_sync_history_started ON connector_sync_history(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_sync_history_status ON connector_sync_history(status) WHERE status = 'running';

-- Add trigger to update next_sync_at when schedule changes
CREATE OR REPLACE FUNCTION update_connector_next_sync()
RETURNS TRIGGER AS $$
BEGIN
    -- Calculate next sync time based on schedule interval
    -- For simplicity, treat schedule as Go duration format (e.g., "1h", "30m")
    -- In production, could parse cron expressions
    IF NEW.sync_enabled AND NEW.enabled THEN
        NEW.next_sync_at := NOW() + (
            CASE
                WHEN NEW.sync_schedule LIKE '%h' THEN
                    (CAST(REPLACE(NEW.sync_schedule, 'h', '') AS INT) || ' hours')::INTERVAL
                WHEN NEW.sync_schedule LIKE '%m' THEN
                    (CAST(REPLACE(NEW.sync_schedule, 'm', '') AS INT) || ' minutes')::INTERVAL
                WHEN NEW.sync_schedule LIKE '%d' THEN
                    (CAST(REPLACE(NEW.sync_schedule, 'd', '') AS INT) || ' days')::INTERVAL
                ELSE
                    '1 hour'::INTERVAL
            END
        );
    ELSE
        NEW.next_sync_at := NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS connector_schedule_update ON connectors;
CREATE TRIGGER connector_schedule_update
    BEFORE INSERT OR UPDATE OF sync_schedule, sync_enabled, enabled
    ON connectors
    FOR EACH ROW
    EXECUTE FUNCTION update_connector_next_sync();

-- Update existing connectors to have default next_sync_at
UPDATE connectors
SET next_sync_at = NOW() + INTERVAL '1 hour'
WHERE enabled = TRUE AND next_sync_at IS NULL;

COMMENT ON COLUMN connectors.sync_schedule IS 'Sync schedule as duration (1h, 30m, 6h, 1d) or cron expression';
COMMENT ON COLUMN connectors.sync_enabled IS 'Whether automatic sync is enabled for this connector';
COMMENT ON COLUMN connectors.next_sync_at IS 'Next scheduled sync time';
COMMENT ON COLUMN connectors.sync_timeout_seconds IS 'Maximum time allowed for sync operation';
COMMENT ON TABLE connector_sync_history IS 'History of connector sync operations for audit and debugging';
