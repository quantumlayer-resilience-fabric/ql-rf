-- Migration: Add LLM Cost Tracking
-- Purpose: Track token usage and costs per organization/task for billing

-- LLM usage tracking table
CREATE TABLE llm_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Organization and user context
    org_id UUID NOT NULL REFERENCES organizations(id),
    user_id VARCHAR(255),

    -- Task context (if applicable)
    task_id UUID REFERENCES ai_tasks(id),
    agent_name VARCHAR(100),

    -- Request details
    request_id UUID NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Provider and model
    provider VARCHAR(50) NOT NULL, -- anthropic, openai, azure_openai
    model VARCHAR(100) NOT NULL, -- claude-3-5-sonnet, gpt-4, etc.

    -- Token usage
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER GENERATED ALWAYS AS (input_tokens + output_tokens) STORED,

    -- Caching (for Anthropic prompt caching)
    cache_creation_tokens INTEGER DEFAULT 0,
    cache_read_tokens INTEGER DEFAULT 0,

    -- Cost calculation (in USD cents to avoid floating point)
    input_cost_cents INTEGER NOT NULL DEFAULT 0,
    output_cost_cents INTEGER NOT NULL DEFAULT 0,
    cache_creation_cost_cents INTEGER DEFAULT 0,
    cache_read_cost_cents INTEGER DEFAULT 0,
    total_cost_cents INTEGER GENERATED ALWAYS AS (
        input_cost_cents + output_cost_cents +
        COALESCE(cache_creation_cost_cents, 0) + COALESCE(cache_read_cost_cents, 0)
    ) STORED,

    -- Request metadata
    operation_type VARCHAR(50), -- intent_parsing, plan_generation, tool_execution
    latency_ms INTEGER,
    status VARCHAR(20) NOT NULL DEFAULT 'success', -- success, error, rate_limited
    error_message TEXT,

    -- For debugging and optimization
    prompt_hash VARCHAR(64), -- SHA-256 of system prompt for caching analysis

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX idx_llm_usage_org_time ON llm_usage(org_id, timestamp DESC);
CREATE INDEX idx_llm_usage_task ON llm_usage(task_id) WHERE task_id IS NOT NULL;
CREATE INDEX idx_llm_usage_agent ON llm_usage(agent_name, timestamp DESC) WHERE agent_name IS NOT NULL;
CREATE INDEX idx_llm_usage_model ON llm_usage(provider, model, timestamp DESC);
CREATE INDEX idx_llm_usage_billing ON llm_usage(org_id, timestamp) WHERE total_cost_cents > 0;

-- Pricing table for different models
CREATE TABLE llm_pricing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) NOT NULL,
    model VARCHAR(100) NOT NULL,

    -- Pricing per 1M tokens (in cents for precision)
    input_price_per_mtok_cents INTEGER NOT NULL,
    output_price_per_mtok_cents INTEGER NOT NULL,

    -- Caching pricing (if supported)
    cache_creation_price_per_mtok_cents INTEGER,
    cache_read_price_per_mtok_cents INTEGER,

    -- Context window and limits
    context_window INTEGER, -- Max tokens
    max_output_tokens INTEGER,

    -- Effective dates for pricing changes
    effective_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    effective_until TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(provider, model, effective_from)
);

-- Insert current pricing (as of late 2024)
INSERT INTO llm_pricing (provider, model, input_price_per_mtok_cents, output_price_per_mtok_cents,
                         cache_creation_price_per_mtok_cents, cache_read_price_per_mtok_cents,
                         context_window, max_output_tokens) VALUES
-- Anthropic Claude models
('anthropic', 'claude-3-5-sonnet-20241022', 300, 1500, 375, 30, 200000, 8192),
('anthropic', 'claude-3-5-haiku-20241022', 100, 500, 125, 10, 200000, 8192),
('anthropic', 'claude-3-opus-20240229', 1500, 7500, NULL, NULL, 200000, 4096),
('anthropic', 'claude-sonnet-4-20250514', 300, 1500, 375, 30, 200000, 64000),

-- OpenAI models
('openai', 'gpt-4o', 250, 1000, NULL, NULL, 128000, 16384),
('openai', 'gpt-4o-mini', 15, 60, NULL, NULL, 128000, 16384),
('openai', 'gpt-4-turbo', 1000, 3000, NULL, NULL, 128000, 4096),

-- Azure OpenAI (same pricing structure as OpenAI)
('azure_openai', 'gpt-4o', 250, 1000, NULL, NULL, 128000, 16384),
('azure_openai', 'gpt-4o-mini', 15, 60, NULL, NULL, 128000, 16384);

-- Organization usage quotas
CREATE TABLE org_llm_quotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) UNIQUE,

    -- Monthly limits (in tokens or cents)
    monthly_token_limit BIGINT, -- NULL = unlimited
    monthly_cost_limit_cents INTEGER, -- NULL = unlimited

    -- Rate limits
    requests_per_minute INTEGER DEFAULT 60,
    tokens_per_minute INTEGER DEFAULT 100000,

    -- Alerts
    alert_at_percent INTEGER DEFAULT 80, -- Alert when usage reaches this %

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Monthly usage summary (materialized for billing)
CREATE TABLE org_monthly_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    month DATE NOT NULL, -- First day of month

    -- Aggregate metrics
    total_requests INTEGER DEFAULT 0,
    total_input_tokens BIGINT DEFAULT 0,
    total_output_tokens BIGINT DEFAULT 0,
    total_tokens BIGINT DEFAULT 0,

    -- Cost aggregates
    total_cost_cents INTEGER DEFAULT 0,

    -- By model breakdown (stored as JSONB for flexibility)
    usage_by_model JSONB DEFAULT '{}',

    -- By agent breakdown
    usage_by_agent JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(org_id, month)
);

-- Function to update monthly usage after each LLM call
CREATE OR REPLACE FUNCTION update_monthly_usage()
RETURNS TRIGGER AS $$
DECLARE
    usage_month DATE;
    model_key TEXT;
    agent_key TEXT;
BEGIN
    usage_month := DATE_TRUNC('month', NEW.timestamp)::DATE;
    model_key := NEW.provider || ':' || NEW.model;
    agent_key := COALESCE(NEW.agent_name, 'direct');

    INSERT INTO org_monthly_usage (org_id, month, total_requests, total_input_tokens,
                                   total_output_tokens, total_tokens, total_cost_cents,
                                   usage_by_model, usage_by_agent)
    VALUES (
        NEW.org_id,
        usage_month,
        1,
        NEW.input_tokens,
        NEW.output_tokens,
        NEW.input_tokens + NEW.output_tokens,
        NEW.total_cost_cents,
        jsonb_build_object(model_key, jsonb_build_object(
            'requests', 1,
            'input_tokens', NEW.input_tokens,
            'output_tokens', NEW.output_tokens,
            'cost_cents', NEW.total_cost_cents
        )),
        jsonb_build_object(agent_key, jsonb_build_object(
            'requests', 1,
            'tokens', NEW.input_tokens + NEW.output_tokens,
            'cost_cents', NEW.total_cost_cents
        ))
    )
    ON CONFLICT (org_id, month) DO UPDATE SET
        total_requests = org_monthly_usage.total_requests + 1,
        total_input_tokens = org_monthly_usage.total_input_tokens + NEW.input_tokens,
        total_output_tokens = org_monthly_usage.total_output_tokens + NEW.output_tokens,
        total_tokens = org_monthly_usage.total_tokens + NEW.input_tokens + NEW.output_tokens,
        total_cost_cents = org_monthly_usage.total_cost_cents + NEW.total_cost_cents,
        usage_by_model = org_monthly_usage.usage_by_model || jsonb_build_object(
            model_key,
            COALESCE(org_monthly_usage.usage_by_model->model_key, '{}'::jsonb) || jsonb_build_object(
                'requests', COALESCE((org_monthly_usage.usage_by_model->model_key->>'requests')::int, 0) + 1,
                'input_tokens', COALESCE((org_monthly_usage.usage_by_model->model_key->>'input_tokens')::bigint, 0) + NEW.input_tokens,
                'output_tokens', COALESCE((org_monthly_usage.usage_by_model->model_key->>'output_tokens')::bigint, 0) + NEW.output_tokens,
                'cost_cents', COALESCE((org_monthly_usage.usage_by_model->model_key->>'cost_cents')::int, 0) + NEW.total_cost_cents
            )
        ),
        usage_by_agent = org_monthly_usage.usage_by_agent || jsonb_build_object(
            agent_key,
            COALESCE(org_monthly_usage.usage_by_agent->agent_key, '{}'::jsonb) || jsonb_build_object(
                'requests', COALESCE((org_monthly_usage.usage_by_agent->agent_key->>'requests')::int, 0) + 1,
                'tokens', COALESCE((org_monthly_usage.usage_by_agent->agent_key->>'tokens')::bigint, 0) + NEW.input_tokens + NEW.output_tokens,
                'cost_cents', COALESCE((org_monthly_usage.usage_by_agent->agent_key->>'cost_cents')::int, 0) + NEW.total_cost_cents
            )
        ),
        updated_at = NOW();

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER llm_usage_update_monthly
    AFTER INSERT ON llm_usage
    FOR EACH ROW
    EXECUTE FUNCTION update_monthly_usage();

-- Function to check quota before allowing request
CREATE OR REPLACE FUNCTION check_llm_quota(p_org_id UUID)
RETURNS TABLE(allowed BOOLEAN, reason TEXT, current_usage_percent NUMERIC) AS $$
DECLARE
    quota_rec RECORD;
    usage_rec RECORD;
    current_month DATE;
BEGIN
    current_month := DATE_TRUNC('month', NOW())::DATE;

    -- Get quota
    SELECT * INTO quota_rec FROM org_llm_quotas WHERE org_id = p_org_id;

    -- If no quota configured, allow
    IF NOT FOUND THEN
        RETURN QUERY SELECT TRUE, 'No quota configured'::TEXT, 0::NUMERIC;
        RETURN;
    END IF;

    -- Get current month usage
    SELECT * INTO usage_rec FROM org_monthly_usage
    WHERE org_id = p_org_id AND month = current_month;

    -- Check token limit
    IF quota_rec.monthly_token_limit IS NOT NULL THEN
        IF COALESCE(usage_rec.total_tokens, 0) >= quota_rec.monthly_token_limit THEN
            RETURN QUERY SELECT FALSE, 'Monthly token limit exceeded'::TEXT,
                100::NUMERIC;
            RETURN;
        END IF;
    END IF;

    -- Check cost limit
    IF quota_rec.monthly_cost_limit_cents IS NOT NULL THEN
        IF COALESCE(usage_rec.total_cost_cents, 0) >= quota_rec.monthly_cost_limit_cents THEN
            RETURN QUERY SELECT FALSE, 'Monthly cost limit exceeded'::TEXT,
                100::NUMERIC;
            RETURN;
        END IF;
    END IF;

    -- Calculate usage percentage
    DECLARE
        token_percent NUMERIC := 0;
        cost_percent NUMERIC := 0;
    BEGIN
        IF quota_rec.monthly_token_limit IS NOT NULL AND quota_rec.monthly_token_limit > 0 THEN
            token_percent := (COALESCE(usage_rec.total_tokens, 0)::NUMERIC / quota_rec.monthly_token_limit) * 100;
        END IF;
        IF quota_rec.monthly_cost_limit_cents IS NOT NULL AND quota_rec.monthly_cost_limit_cents > 0 THEN
            cost_percent := (COALESCE(usage_rec.total_cost_cents, 0)::NUMERIC / quota_rec.monthly_cost_limit_cents) * 100;
        END IF;

        RETURN QUERY SELECT TRUE, NULL::TEXT, GREATEST(token_percent, cost_percent);
    END;
END;
$$ LANGUAGE plpgsql;

-- View for easy cost reporting
CREATE VIEW v_llm_cost_report AS
SELECT
    o.id as org_id,
    o.name as org_name,
    m.month,
    m.total_requests,
    m.total_input_tokens,
    m.total_output_tokens,
    m.total_tokens,
    m.total_cost_cents,
    ROUND(m.total_cost_cents / 100.0, 2) as total_cost_usd,
    m.usage_by_model,
    m.usage_by_agent,
    q.monthly_token_limit,
    q.monthly_cost_limit_cents,
    CASE
        WHEN q.monthly_token_limit IS NOT NULL
        THEN ROUND((m.total_tokens::NUMERIC / q.monthly_token_limit) * 100, 1)
        ELSE NULL
    END as token_usage_percent,
    CASE
        WHEN q.monthly_cost_limit_cents IS NOT NULL
        THEN ROUND((m.total_cost_cents::NUMERIC / q.monthly_cost_limit_cents) * 100, 1)
        ELSE NULL
    END as cost_usage_percent
FROM organizations o
LEFT JOIN org_monthly_usage m ON m.org_id = o.id
LEFT JOIN org_llm_quotas q ON q.org_id = o.id
ORDER BY m.month DESC, o.name;

-- Comments
COMMENT ON TABLE llm_usage IS 'Individual LLM API call tracking for cost attribution';
COMMENT ON TABLE llm_pricing IS 'Model pricing data for cost calculation';
COMMENT ON TABLE org_llm_quotas IS 'Per-organization LLM usage quotas';
COMMENT ON TABLE org_monthly_usage IS 'Aggregated monthly usage for billing';
