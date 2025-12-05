-- Migration: Add Enterprise Audit Trail
-- Purpose: Immutable audit logging for compliance (SOC2, HIPAA, etc.)

-- Audit log table - append-only, no updates or deletes allowed
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Timestamp with timezone for global deployments
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Actor information
    actor_type VARCHAR(50) NOT NULL, -- user, system, agent, api_key
    actor_id VARCHAR(255) NOT NULL,
    actor_email VARCHAR(255),
    actor_ip_address INET,
    actor_user_agent TEXT,

    -- Organization context
    org_id UUID NOT NULL REFERENCES organizations(id),

    -- Action details
    action VARCHAR(100) NOT NULL, -- e.g., asset.create, image.promote, ai.task.approve
    action_category VARCHAR(50) NOT NULL, -- read, create, update, delete, execute

    -- Resource affected
    resource_type VARCHAR(100) NOT NULL, -- asset, image, ai_task, user, etc.
    resource_id VARCHAR(255),
    resource_name VARCHAR(500),

    -- Change tracking
    changes JSONB, -- { "field": {"old": x, "new": y} }

    -- Request context
    request_id UUID, -- Correlation ID for distributed tracing
    session_id VARCHAR(255),
    api_version VARCHAR(20),

    -- Additional context
    context JSONB, -- Additional metadata (environment, tags, etc.)

    -- Risk and compliance flags
    risk_level VARCHAR(20), -- low, medium, high, critical
    compliance_relevant BOOLEAN DEFAULT FALSE,
    pii_accessed BOOLEAN DEFAULT FALSE,

    -- Outcome
    status VARCHAR(20) NOT NULL DEFAULT 'success', -- success, failure, denied
    error_code VARCHAR(50),
    error_message TEXT,

    -- Duration for performance tracking
    duration_ms INTEGER,

    -- Hash for integrity verification (SHA-256 of previous row + this row's data)
    integrity_hash VARCHAR(64),
    previous_hash VARCHAR(64),

    -- Retention
    retention_days INTEGER DEFAULT 2555, -- 7 years default for compliance
    expires_at TIMESTAMPTZ
);

-- Prevent updates and deletes on audit_logs
CREATE OR REPLACE FUNCTION prevent_audit_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit logs are immutable and cannot be modified or deleted';
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_logs_immutable_update
    BEFORE UPDATE ON audit_logs
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_modification();

CREATE TRIGGER audit_logs_immutable_delete
    BEFORE DELETE ON audit_logs
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_modification();

-- Indexes for common query patterns
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_org_timestamp ON audit_logs(org_id, timestamp DESC);
CREATE INDEX idx_audit_logs_actor ON audit_logs(actor_id, timestamp DESC);
CREATE INDEX idx_audit_logs_action ON audit_logs(action, timestamp DESC);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id, timestamp DESC);
CREATE INDEX idx_audit_logs_request_id ON audit_logs(request_id);
CREATE INDEX idx_audit_logs_compliance ON audit_logs(compliance_relevant, timestamp DESC) WHERE compliance_relevant = TRUE;
CREATE INDEX idx_audit_logs_risk ON audit_logs(risk_level, timestamp DESC) WHERE risk_level IN ('high', 'critical');

-- Partition by month for performance (optional, enable for high-volume deployments)
-- Note: Uncomment and modify for production partitioning
-- CREATE TABLE audit_logs_2024_01 PARTITION OF audit_logs
--     FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- Audit log export queue for SIEM integration
CREATE TABLE audit_export_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    audit_log_id UUID NOT NULL,
    destination VARCHAR(100) NOT NULL, -- splunk, elastic, datadog, s3, etc.
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, processing, completed, failed
    attempts INTEGER DEFAULT 0,
    last_attempt_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_export_queue_status ON audit_export_queue(status, created_at);

-- Function to calculate integrity hash
CREATE OR REPLACE FUNCTION calculate_audit_hash()
RETURNS TRIGGER AS $$
DECLARE
    prev_hash VARCHAR(64);
    hash_input TEXT;
BEGIN
    -- Get the previous hash
    SELECT integrity_hash INTO prev_hash
    FROM audit_logs
    WHERE org_id = NEW.org_id
    ORDER BY timestamp DESC
    LIMIT 1;

    IF prev_hash IS NULL THEN
        prev_hash := 'genesis';
    END IF;

    NEW.previous_hash := prev_hash;

    -- Build hash input from immutable fields
    hash_input := COALESCE(prev_hash, '') ||
                  NEW.timestamp::TEXT ||
                  NEW.actor_type ||
                  NEW.actor_id ||
                  NEW.org_id::TEXT ||
                  NEW.action ||
                  COALESCE(NEW.resource_type, '') ||
                  COALESCE(NEW.resource_id, '') ||
                  COALESCE(NEW.changes::TEXT, '');

    -- Calculate SHA-256 hash
    NEW.integrity_hash := encode(sha256(hash_input::bytea), 'hex');

    -- Set expiration if not set
    IF NEW.expires_at IS NULL THEN
        NEW.expires_at := NEW.timestamp + (NEW.retention_days || ' days')::INTERVAL;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_logs_calculate_hash
    BEFORE INSERT ON audit_logs
    FOR EACH ROW
    EXECUTE FUNCTION calculate_audit_hash();

-- Function to queue audit log for SIEM export
CREATE OR REPLACE FUNCTION queue_audit_export()
RETURNS TRIGGER AS $$
BEGIN
    -- Queue for all configured destinations
    -- In production, this would read from a config table
    INSERT INTO audit_export_queue (audit_log_id, destination)
    VALUES (NEW.id, 'default');

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_logs_queue_export
    AFTER INSERT ON audit_logs
    FOR EACH ROW
    EXECUTE FUNCTION queue_audit_export();

-- SIEM export destinations configuration
CREATE TABLE audit_export_destinations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID REFERENCES organizations(id),
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL, -- splunk, elastic, datadog, s3, webhook
    config JSONB NOT NULL, -- destination-specific config
    enabled BOOLEAN DEFAULT TRUE,
    filter_query TEXT, -- Optional: SQL WHERE clause for filtering
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, name)
);

-- API key audit (track API key usage separately for security)
CREATE TABLE api_key_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id UUID NOT NULL,
    org_id UUID NOT NULL REFERENCES organizations(id),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    status_code INTEGER,
    response_time_ms INTEGER,
    ip_address INET,
    user_agent TEXT,
    request_size_bytes INTEGER,
    response_size_bytes INTEGER
);

CREATE INDEX idx_api_key_usage_key_time ON api_key_usage(api_key_id, timestamp DESC);
CREATE INDEX idx_api_key_usage_org_time ON api_key_usage(org_id, timestamp DESC);

-- Session tracking for user audit
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    org_id UUID NOT NULL REFERENCES organizations(id),
    session_token_hash VARCHAR(64) NOT NULL, -- Never store actual token
    ip_address INET,
    user_agent TEXT,
    device_fingerprint VARCHAR(64),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_activity_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    revoke_reason VARCHAR(100)
);

CREATE INDEX idx_user_sessions_user ON user_sessions(user_id, created_at DESC);
CREATE INDEX idx_user_sessions_org ON user_sessions(org_id, created_at DESC);

-- Resource access audit (for RBAC tracking)
CREATE TABLE resource_access_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id VARCHAR(255) NOT NULL,
    org_id UUID NOT NULL REFERENCES organizations(id),
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255) NOT NULL,
    permission VARCHAR(50) NOT NULL, -- read, write, delete, admin
    granted BOOLEAN NOT NULL,
    denied_reason VARCHAR(255),
    policy_id UUID, -- Which policy granted/denied access
    request_context JSONB
);

CREATE INDEX idx_resource_access_user ON resource_access_logs(user_id, timestamp DESC);
CREATE INDEX idx_resource_access_resource ON resource_access_logs(resource_type, resource_id, timestamp DESC);
CREATE INDEX idx_resource_access_denied ON resource_access_logs(granted, timestamp DESC) WHERE granted = FALSE;

-- Comments for documentation
COMMENT ON TABLE audit_logs IS 'Immutable audit trail for all system actions - SOC2/HIPAA compliant';
COMMENT ON COLUMN audit_logs.integrity_hash IS 'SHA-256 hash chain for tamper detection';
COMMENT ON COLUMN audit_logs.compliance_relevant IS 'Flag for compliance-specific actions (data access, config changes)';
COMMENT ON TABLE audit_export_queue IS 'Queue for exporting audit logs to external SIEM systems';
