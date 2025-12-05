-- Migration: Add FinOps Tables
-- Purpose: Cost tracking, budgets, and optimization recommendations

-- =============================================================================
-- COST RECORDS
-- =============================================================================

-- Cost records table for tracking resource costs over time
CREATE TABLE IF NOT EXISTS cost_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    resource_id VARCHAR(512) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_name VARCHAR(255),
    cloud VARCHAR(50) NOT NULL CHECK (cloud IN ('aws', 'azure', 'gcp', 'vsphere', 'kubernetes')),
    service VARCHAR(100),
    region VARCHAR(100),
    site VARCHAR(100),
    cost DECIMAL(15,4) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    usage_hours DECIMAL(10,2),
    tags JSONB,
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Ensure one record per org/resource/time combination
    UNIQUE(org_id, resource_id, recorded_at)
);

-- Indexes for efficient querying
CREATE INDEX idx_cost_records_org_id ON cost_records(org_id);
CREATE INDEX idx_cost_records_recorded_at ON cost_records(recorded_at);
CREATE INDEX idx_cost_records_cloud ON cost_records(cloud);
CREATE INDEX idx_cost_records_service ON cost_records(service);
CREATE INDEX idx_cost_records_resource_type ON cost_records(resource_type);
CREATE INDEX idx_cost_records_org_recorded ON cost_records(org_id, recorded_at);
CREATE INDEX idx_cost_records_org_cloud_recorded ON cost_records(org_id, cloud, recorded_at);
CREATE INDEX idx_cost_records_tags ON cost_records USING GIN(tags);

-- Enable row-level security
ALTER TABLE cost_records ENABLE ROW LEVEL SECURITY;

-- RLS policy: Users can only see their organization's cost records
CREATE POLICY cost_records_org_isolation ON cost_records
    FOR ALL
    USING (org_id = current_setting('app.current_org_id', TRUE)::UUID);

-- =============================================================================
-- COST RECOMMENDATIONS
-- =============================================================================

-- Cost optimization recommendations table
CREATE TABLE IF NOT EXISTS cost_recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    type VARCHAR(100) NOT NULL CHECK (type IN (
        'rightsizing',
        'reserved_instances',
        'spot_instances',
        'idle_resources',
        'storage_optimization',
        'unused_volumes',
        'old_snapshots'
    )),
    resource_id VARCHAR(512) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_name VARCHAR(255),
    platform VARCHAR(50) NOT NULL CHECK (platform IN ('aws', 'azure', 'gcp', 'vsphere', 'kubernetes')),
    current_cost DECIMAL(15,4) NOT NULL DEFAULT 0,
    potential_savings DECIMAL(15,4) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    action TEXT NOT NULL,
    details TEXT,
    priority VARCHAR(20) NOT NULL DEFAULT 'medium' CHECK (priority IN ('high', 'medium', 'low')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'applied', 'dismissed')),
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    applied_at TIMESTAMP WITH TIME ZONE,
    dismissed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for cost recommendations
CREATE INDEX idx_cost_recommendations_org_id ON cost_recommendations(org_id);
CREATE INDEX idx_cost_recommendations_status ON cost_recommendations(status);
CREATE INDEX idx_cost_recommendations_priority ON cost_recommendations(priority);
CREATE INDEX idx_cost_recommendations_type ON cost_recommendations(type);
CREATE INDEX idx_cost_recommendations_platform ON cost_recommendations(platform);
CREATE INDEX idx_cost_recommendations_org_status ON cost_recommendations(org_id, status);
CREATE INDEX idx_cost_recommendations_detected_at ON cost_recommendations(detected_at);

-- Enable row-level security
ALTER TABLE cost_recommendations ENABLE ROW LEVEL SECURITY;

-- RLS policy: Users can only see their organization's recommendations
CREATE POLICY cost_recommendations_org_isolation ON cost_recommendations
    FOR ALL
    USING (org_id = current_setting('app.current_org_id', TRUE)::UUID);

-- =============================================================================
-- COST BUDGETS
-- =============================================================================

-- Cost budgets table for tracking spending limits
CREATE TABLE IF NOT EXISTS cost_budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    amount DECIMAL(15,4) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    period VARCHAR(20) NOT NULL CHECK (period IN ('daily', 'weekly', 'monthly', 'quarterly', 'yearly')),
    scope VARCHAR(50) NOT NULL CHECK (scope IN ('organization', 'cloud', 'service', 'site')),
    scope_value VARCHAR(100),
    alert_threshold DECIMAL(5,2) NOT NULL DEFAULT 80.0 CHECK (alert_threshold >= 0 AND alert_threshold <= 100),
    start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    end_date TIMESTAMP WITH TIME ZONE,
    current_spend DECIMAL(15,4) NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Ensure unique budget names per organization
    UNIQUE(org_id, name)
);

-- Indexes for cost budgets
CREATE INDEX idx_cost_budgets_org_id ON cost_budgets(org_id);
CREATE INDEX idx_cost_budgets_active ON cost_budgets(active);
CREATE INDEX idx_cost_budgets_start_date ON cost_budgets(start_date);
CREATE INDEX idx_cost_budgets_end_date ON cost_budgets(end_date);
CREATE INDEX idx_cost_budgets_org_active ON cost_budgets(org_id, active);

-- Enable row-level security
ALTER TABLE cost_budgets ENABLE ROW LEVEL SECURITY;

-- RLS policy: Users can only see their organization's budgets
CREATE POLICY cost_budgets_org_isolation ON cost_budgets
    FOR ALL
    USING (org_id = current_setting('app.current_org_id', TRUE)::UUID);

-- =============================================================================
-- COST ALERTS
-- =============================================================================

-- Cost alerts table for budget threshold notifications
CREATE TABLE IF NOT EXISTS cost_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    budget_id UUID NOT NULL REFERENCES cost_budgets(id) ON DELETE CASCADE,
    budget_name VARCHAR(255),
    amount DECIMAL(15,4) NOT NULL,
    budget_limit DECIMAL(15,4) NOT NULL,
    percentage DECIMAL(5,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    message TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('warning', 'critical')),
    acknowledged BOOLEAN NOT NULL DEFAULT FALSE,
    triggered_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for cost alerts
CREATE INDEX idx_cost_alerts_org_id ON cost_alerts(org_id);
CREATE INDEX idx_cost_alerts_budget_id ON cost_alerts(budget_id);
CREATE INDEX idx_cost_alerts_acknowledged ON cost_alerts(acknowledged);
CREATE INDEX idx_cost_alerts_triggered_at ON cost_alerts(triggered_at);
CREATE INDEX idx_cost_alerts_org_ack ON cost_alerts(org_id, acknowledged);

-- Enable row-level security
ALTER TABLE cost_alerts ENABLE ROW LEVEL SECURITY;

-- RLS policy: Users can only see their organization's alerts
CREATE POLICY cost_alerts_org_isolation ON cost_alerts
    FOR ALL
    USING (org_id = current_setting('app.current_org_id', TRUE)::UUID);

-- =============================================================================
-- TRIGGERS
-- =============================================================================

-- Trigger to update updated_at timestamp on cost_recommendations
CREATE OR REPLACE FUNCTION update_cost_recommendations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER cost_recommendations_updated_at
    BEFORE UPDATE ON cost_recommendations
    FOR EACH ROW
    EXECUTE FUNCTION update_cost_recommendations_updated_at();

-- Trigger to update updated_at timestamp on cost_budgets
CREATE OR REPLACE FUNCTION update_cost_budgets_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER cost_budgets_updated_at
    BEFORE UPDATE ON cost_budgets
    FOR EACH ROW
    EXECUTE FUNCTION update_cost_budgets_updated_at();

-- =============================================================================
-- MATERIALIZED VIEWS (Optional, for performance)
-- =============================================================================

-- Materialized view for daily cost aggregates (can be refreshed periodically)
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_daily_cost_summary AS
SELECT
    org_id,
    cloud,
    service,
    DATE(recorded_at) as cost_date,
    SUM(cost) as total_cost,
    currency,
    COUNT(DISTINCT resource_id) as resource_count
FROM cost_records
GROUP BY org_id, cloud, service, DATE(recorded_at), currency;

-- Indexes on materialized view
CREATE INDEX idx_mv_daily_cost_org_date ON mv_daily_cost_summary(org_id, cost_date);
CREATE INDEX idx_mv_daily_cost_cloud ON mv_daily_cost_summary(cloud);

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE cost_records IS 'Stores historical cost data for resources across all cloud platforms';
COMMENT ON TABLE cost_recommendations IS 'Cost optimization recommendations generated by platform analysis';
COMMENT ON TABLE cost_budgets IS 'Cost budgets and spending limits configured by organizations';
COMMENT ON TABLE cost_alerts IS 'Alerts triggered when budgets exceed configured thresholds';
COMMENT ON MATERIALIZED VIEW mv_daily_cost_summary IS 'Pre-aggregated daily cost summaries for faster reporting';

COMMENT ON COLUMN cost_records.resource_id IS 'Platform-specific resource identifier (e.g., instance ID, ARN, resource path)';
COMMENT ON COLUMN cost_records.recorded_at IS 'Timestamp when this cost was recorded (typically daily or hourly)';
COMMENT ON COLUMN cost_budgets.alert_threshold IS 'Percentage of budget at which to trigger alerts (0-100)';
COMMENT ON COLUMN cost_budgets.scope IS 'What this budget applies to (organization, cloud, service, or site)';
COMMENT ON COLUMN cost_budgets.scope_value IS 'Value for the scope (e.g., "aws", "ec2", "us-east-1")';
