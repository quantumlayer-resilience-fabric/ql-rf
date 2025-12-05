-- Migration: Add Multi-tenancy Support
-- Purpose: Organization isolation, quotas, and usage tracking

-- =============================================================================
-- ORGANIZATION QUOTAS AND LIMITS
-- =============================================================================

-- Quota definitions for organizations
CREATE TABLE organization_quotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- Resource limits
    max_assets INTEGER DEFAULT 1000,
    max_images INTEGER DEFAULT 100,
    max_sites INTEGER DEFAULT 50,
    max_users INTEGER DEFAULT 100,
    max_teams INTEGER DEFAULT 20,

    -- AI/LLM limits
    max_ai_tasks_per_day INTEGER DEFAULT 100,
    max_ai_tokens_per_month BIGINT DEFAULT 10000000,
    max_concurrent_tasks INTEGER DEFAULT 5,

    -- Storage limits (in bytes)
    max_storage_bytes BIGINT DEFAULT 107374182400, -- 100GB
    max_artifact_size_bytes BIGINT DEFAULT 1073741824, -- 1GB

    -- API limits
    api_rate_limit_per_minute INTEGER DEFAULT 1000,
    api_rate_limit_per_day INTEGER DEFAULT 100000,

    -- Feature flags
    dr_enabled BOOLEAN DEFAULT FALSE,
    compliance_enabled BOOLEAN DEFAULT FALSE,
    advanced_analytics_enabled BOOLEAN DEFAULT FALSE,
    custom_integrations_enabled BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(org_id)
);

-- =============================================================================
-- USAGE TRACKING
-- =============================================================================

-- Real-time usage counters
CREATE TABLE organization_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- Resource counts
    asset_count INTEGER DEFAULT 0,
    image_count INTEGER DEFAULT 0,
    site_count INTEGER DEFAULT 0,
    user_count INTEGER DEFAULT 0,
    team_count INTEGER DEFAULT 0,

    -- Storage usage
    storage_used_bytes BIGINT DEFAULT 0,

    -- Period-based counters
    ai_tasks_today INTEGER DEFAULT 0,
    ai_tokens_this_month BIGINT DEFAULT 0,
    api_requests_today INTEGER DEFAULT 0,
    api_requests_this_minute INTEGER DEFAULT 0,

    -- Tracking dates
    last_ai_task_reset DATE DEFAULT CURRENT_DATE,
    last_token_reset DATE DEFAULT DATE_TRUNC('month', CURRENT_DATE)::DATE,
    last_api_day_reset DATE DEFAULT CURRENT_DATE,
    last_api_minute TIMESTAMPTZ DEFAULT NOW(),

    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(org_id)
);

-- Historical usage for billing and analytics
CREATE TABLE organization_usage_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- Period
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    period_type VARCHAR(20) NOT NULL, -- daily, weekly, monthly

    -- Aggregated metrics
    total_assets INTEGER DEFAULT 0,
    total_images INTEGER DEFAULT 0,
    total_ai_tasks INTEGER DEFAULT 0,
    total_ai_tokens BIGINT DEFAULT 0,
    total_api_requests BIGINT DEFAULT 0,

    -- Peak values
    peak_concurrent_tasks INTEGER DEFAULT 0,
    peak_storage_bytes BIGINT DEFAULT 0,

    -- Cost tracking
    estimated_cost_usd DECIMAL(10, 2) DEFAULT 0,

    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(org_id, period_start, period_type)
);

CREATE INDEX idx_usage_history_org_period ON organization_usage_history(org_id, period_start DESC);

-- =============================================================================
-- TENANT ISOLATION
-- =============================================================================

-- Row Level Security policies will reference this
CREATE TABLE tenant_context (
    key VARCHAR(100) PRIMARY KEY,
    description TEXT
);

INSERT INTO tenant_context (key, description) VALUES
('org_id', 'Organization ID for row-level filtering'),
('user_id', 'User ID for audit context');

-- =============================================================================
-- SUBSCRIPTION AND BILLING
-- =============================================================================

-- Subscription plans
CREATE TABLE subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Plan type
    plan_type VARCHAR(50) NOT NULL, -- free, starter, professional, enterprise

    -- Default quotas for this plan
    default_max_assets INTEGER NOT NULL,
    default_max_images INTEGER NOT NULL,
    default_max_sites INTEGER NOT NULL,
    default_max_users INTEGER NOT NULL,
    default_max_ai_tasks_per_day INTEGER NOT NULL,
    default_max_ai_tokens_per_month BIGINT NOT NULL,
    default_max_storage_bytes BIGINT NOT NULL,
    default_api_rate_limit_per_minute INTEGER NOT NULL,

    -- Features
    dr_included BOOLEAN DEFAULT FALSE,
    compliance_included BOOLEAN DEFAULT FALSE,
    advanced_analytics_included BOOLEAN DEFAULT FALSE,
    custom_integrations_included BOOLEAN DEFAULT FALSE,

    -- Pricing (for reference, actual billing handled externally)
    monthly_price_usd DECIMAL(10, 2),
    annual_price_usd DECIMAL(10, 2),

    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Organization subscriptions
CREATE TABLE organization_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES subscription_plans(id),

    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, cancelled, suspended, trial

    -- Trial info
    trial_ends_at TIMESTAMPTZ,

    -- Billing period
    current_period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    current_period_end TIMESTAMPTZ NOT NULL,

    -- External billing reference
    external_subscription_id VARCHAR(255), -- Stripe, etc.
    external_customer_id VARCHAR(255),

    -- Cancellation
    cancelled_at TIMESTAMPTZ,
    cancel_reason TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(org_id)
);

-- =============================================================================
-- DATA ISOLATION HELPERS
-- =============================================================================

-- Function to get current tenant org_id (for RLS policies)
CREATE OR REPLACE FUNCTION current_tenant_org_id() RETURNS UUID AS $$
BEGIN
    RETURN NULLIF(current_setting('app.current_org_id', TRUE), '')::UUID;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to set tenant context
CREATE OR REPLACE FUNCTION set_tenant_context(p_org_id UUID, p_user_id VARCHAR(255) DEFAULT NULL) RETURNS VOID AS $$
BEGIN
    PERFORM set_config('app.current_org_id', p_org_id::TEXT, TRUE);
    IF p_user_id IS NOT NULL THEN
        PERFORM set_config('app.current_user_id', p_user_id, TRUE);
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to clear tenant context
CREATE OR REPLACE FUNCTION clear_tenant_context() RETURNS VOID AS $$
BEGIN
    PERFORM set_config('app.current_org_id', '', TRUE);
    PERFORM set_config('app.current_user_id', '', TRUE);
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- QUOTA ENFORCEMENT FUNCTIONS
-- =============================================================================

-- Check if org is within quota for a resource type
CREATE OR REPLACE FUNCTION check_quota(
    p_org_id UUID,
    p_resource_type VARCHAR(50),
    p_increment INTEGER DEFAULT 1
) RETURNS BOOLEAN AS $$
DECLARE
    v_quota INTEGER;
    v_usage INTEGER;
BEGIN
    -- Get quota
    CASE p_resource_type
        WHEN 'assets' THEN
            SELECT max_assets INTO v_quota FROM organization_quotas WHERE org_id = p_org_id;
            SELECT asset_count INTO v_usage FROM organization_usage WHERE org_id = p_org_id;
        WHEN 'images' THEN
            SELECT max_images INTO v_quota FROM organization_quotas WHERE org_id = p_org_id;
            SELECT image_count INTO v_usage FROM organization_usage WHERE org_id = p_org_id;
        WHEN 'sites' THEN
            SELECT max_sites INTO v_quota FROM organization_quotas WHERE org_id = p_org_id;
            SELECT site_count INTO v_usage FROM organization_usage WHERE org_id = p_org_id;
        WHEN 'users' THEN
            SELECT max_users INTO v_quota FROM organization_quotas WHERE org_id = p_org_id;
            SELECT user_count INTO v_usage FROM organization_usage WHERE org_id = p_org_id;
        WHEN 'teams' THEN
            SELECT max_teams INTO v_quota FROM organization_quotas WHERE org_id = p_org_id;
            SELECT team_count INTO v_usage FROM organization_usage WHERE org_id = p_org_id;
        WHEN 'ai_tasks' THEN
            SELECT max_ai_tasks_per_day INTO v_quota FROM organization_quotas WHERE org_id = p_org_id;
            SELECT ai_tasks_today INTO v_usage FROM organization_usage WHERE org_id = p_org_id;
        ELSE
            RETURN TRUE; -- Unknown resource type, allow
    END CASE;

    -- No quota set = no limit
    IF v_quota IS NULL THEN
        RETURN TRUE;
    END IF;

    -- No usage record = within quota
    IF v_usage IS NULL THEN
        v_usage := 0;
    END IF;

    RETURN (v_usage + p_increment) <= v_quota;
END;
$$ LANGUAGE plpgsql;

-- Increment usage counter
CREATE OR REPLACE FUNCTION increment_usage(
    p_org_id UUID,
    p_resource_type VARCHAR(50),
    p_increment INTEGER DEFAULT 1
) RETURNS VOID AS $$
BEGIN
    -- Ensure usage record exists
    INSERT INTO organization_usage (org_id)
    VALUES (p_org_id)
    ON CONFLICT (org_id) DO NOTHING;

    -- Update the appropriate counter
    CASE p_resource_type
        WHEN 'assets' THEN
            UPDATE organization_usage SET asset_count = asset_count + p_increment, updated_at = NOW() WHERE org_id = p_org_id;
        WHEN 'images' THEN
            UPDATE organization_usage SET image_count = image_count + p_increment, updated_at = NOW() WHERE org_id = p_org_id;
        WHEN 'sites' THEN
            UPDATE organization_usage SET site_count = site_count + p_increment, updated_at = NOW() WHERE org_id = p_org_id;
        WHEN 'users' THEN
            UPDATE organization_usage SET user_count = user_count + p_increment, updated_at = NOW() WHERE org_id = p_org_id;
        WHEN 'teams' THEN
            UPDATE organization_usage SET team_count = team_count + p_increment, updated_at = NOW() WHERE org_id = p_org_id;
        WHEN 'ai_tasks' THEN
            -- Reset daily counter if needed
            UPDATE organization_usage
            SET ai_tasks_today = CASE
                    WHEN last_ai_task_reset < CURRENT_DATE THEN p_increment
                    ELSE ai_tasks_today + p_increment
                END,
                last_ai_task_reset = CURRENT_DATE,
                updated_at = NOW()
            WHERE org_id = p_org_id;
        WHEN 'ai_tokens' THEN
            -- Reset monthly counter if needed
            UPDATE organization_usage
            SET ai_tokens_this_month = CASE
                    WHEN last_token_reset < DATE_TRUNC('month', CURRENT_DATE)::DATE THEN p_increment
                    ELSE ai_tokens_this_month + p_increment
                END,
                last_token_reset = DATE_TRUNC('month', CURRENT_DATE)::DATE,
                updated_at = NOW()
            WHERE org_id = p_org_id;
        WHEN 'api_requests' THEN
            UPDATE organization_usage
            SET api_requests_today = CASE
                    WHEN last_api_day_reset < CURRENT_DATE THEN p_increment
                    ELSE api_requests_today + p_increment
                END,
                last_api_day_reset = CURRENT_DATE,
                updated_at = NOW()
            WHERE org_id = p_org_id;
        WHEN 'storage' THEN
            UPDATE organization_usage SET storage_used_bytes = storage_used_bytes + p_increment, updated_at = NOW() WHERE org_id = p_org_id;
    END CASE;
END;
$$ LANGUAGE plpgsql;

-- Decrement usage counter
CREATE OR REPLACE FUNCTION decrement_usage(
    p_org_id UUID,
    p_resource_type VARCHAR(50),
    p_decrement INTEGER DEFAULT 1
) RETURNS VOID AS $$
BEGIN
    PERFORM increment_usage(p_org_id, p_resource_type, -p_decrement);
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- QUOTA CHECK API RATE LIMITING
-- =============================================================================

-- Check and increment API rate limit (returns true if allowed)
CREATE OR REPLACE FUNCTION check_api_rate_limit(p_org_id UUID) RETURNS BOOLEAN AS $$
DECLARE
    v_limit_per_minute INTEGER;
    v_limit_per_day INTEGER;
    v_requests_minute INTEGER;
    v_requests_day INTEGER;
    v_last_minute TIMESTAMPTZ;
    v_last_day DATE;
BEGIN
    -- Get limits
    SELECT api_rate_limit_per_minute, api_rate_limit_per_day
    INTO v_limit_per_minute, v_limit_per_day
    FROM organization_quotas
    WHERE org_id = p_org_id;

    -- Default limits if not set
    IF v_limit_per_minute IS NULL THEN v_limit_per_minute := 1000; END IF;
    IF v_limit_per_day IS NULL THEN v_limit_per_day := 100000; END IF;

    -- Get current usage
    SELECT api_requests_this_minute, api_requests_today, last_api_minute, last_api_day_reset
    INTO v_requests_minute, v_requests_day, v_last_minute, v_last_day
    FROM organization_usage
    WHERE org_id = p_org_id;

    -- Create usage record if not exists
    IF v_requests_minute IS NULL THEN
        INSERT INTO organization_usage (org_id, api_requests_this_minute, api_requests_today, last_api_minute, last_api_day_reset)
        VALUES (p_org_id, 0, 0, NOW(), CURRENT_DATE)
        ON CONFLICT (org_id) DO NOTHING;
        v_requests_minute := 0;
        v_requests_day := 0;
        v_last_minute := NOW();
        v_last_day := CURRENT_DATE;
    END IF;

    -- Reset minute counter if more than 1 minute passed
    IF v_last_minute < NOW() - INTERVAL '1 minute' THEN
        v_requests_minute := 0;
    END IF;

    -- Reset daily counter if new day
    IF v_last_day < CURRENT_DATE THEN
        v_requests_day := 0;
    END IF;

    -- Check limits
    IF v_requests_minute >= v_limit_per_minute OR v_requests_day >= v_limit_per_day THEN
        RETURN FALSE;
    END IF;

    -- Increment counters
    UPDATE organization_usage
    SET api_requests_this_minute = CASE
            WHEN last_api_minute < NOW() - INTERVAL '1 minute' THEN 1
            ELSE api_requests_this_minute + 1
        END,
        api_requests_today = CASE
            WHEN last_api_day_reset < CURRENT_DATE THEN 1
            ELSE api_requests_today + 1
        END,
        last_api_minute = NOW(),
        last_api_day_reset = CURRENT_DATE,
        updated_at = NOW()
    WHERE org_id = p_org_id;

    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- INSERT DEFAULT SUBSCRIPTION PLANS
-- =============================================================================

INSERT INTO subscription_plans (
    name, display_name, description, plan_type,
    default_max_assets, default_max_images, default_max_sites, default_max_users,
    default_max_ai_tasks_per_day, default_max_ai_tokens_per_month, default_max_storage_bytes,
    default_api_rate_limit_per_minute,
    dr_included, compliance_included, advanced_analytics_included, custom_integrations_included,
    monthly_price_usd, annual_price_usd
) VALUES
(
    'free', 'Free', 'Perfect for evaluation and small environments',
    'free',
    50, 10, 5, 5,
    10, 100000, 5368709120, -- 5GB
    100,
    FALSE, FALSE, FALSE, FALSE,
    0, 0
),
(
    'starter', 'Starter', 'For small teams getting started with infrastructure resilience',
    'starter',
    250, 50, 20, 25,
    50, 1000000, 53687091200, -- 50GB
    500,
    FALSE, TRUE, FALSE, FALSE,
    99, 990
),
(
    'professional', 'Professional', 'Full-featured plan for growing organizations',
    'professional',
    1000, 200, 100, 100,
    200, 10000000, 536870912000, -- 500GB
    2000,
    TRUE, TRUE, TRUE, FALSE,
    499, 4990
),
(
    'enterprise', 'Enterprise', 'Unlimited scale with premium features and support',
    'enterprise',
    10000, 1000, 500, 500,
    1000, 100000000, 5368709120000, -- 5TB
    10000,
    TRUE, TRUE, TRUE, TRUE,
    NULL, NULL -- Custom pricing
);

-- =============================================================================
-- TRIGGERS FOR AUTOMATIC USAGE TRACKING
-- =============================================================================

-- Function to track asset count changes
CREATE OR REPLACE FUNCTION track_asset_count() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM increment_usage(NEW.org_id, 'assets', 1);
    ELSIF TG_OP = 'DELETE' THEN
        PERFORM decrement_usage(OLD.org_id, 'assets', 1);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to track image count changes
CREATE OR REPLACE FUNCTION track_image_count() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM increment_usage(NEW.org_id, 'images', 1);
    ELSIF TG_OP = 'DELETE' THEN
        PERFORM decrement_usage(OLD.org_id, 'images', 1);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to track site count changes
CREATE OR REPLACE FUNCTION track_site_count() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM increment_usage(NEW.org_id, 'sites', 1);
    ELSIF TG_OP = 'DELETE' THEN
        PERFORM decrement_usage(OLD.org_id, 'sites', 1);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers (only if tables exist - they should from earlier migrations)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'assets') THEN
        DROP TRIGGER IF EXISTS track_asset_count_trigger ON assets;
        CREATE TRIGGER track_asset_count_trigger
            AFTER INSERT OR DELETE ON assets
            FOR EACH ROW EXECUTE FUNCTION track_asset_count();
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'images') THEN
        DROP TRIGGER IF EXISTS track_image_count_trigger ON images;
        CREATE TRIGGER track_image_count_trigger
            AFTER INSERT OR DELETE ON images
            FOR EACH ROW EXECUTE FUNCTION track_image_count();
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'sites') THEN
        DROP TRIGGER IF EXISTS track_site_count_trigger ON sites;
        CREATE TRIGGER track_site_count_trigger
            AFTER INSERT OR DELETE ON sites
            FOR EACH ROW EXECUTE FUNCTION track_site_count();
    END IF;
END$$;

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE organization_quotas IS 'Resource quotas and feature flags per organization';
COMMENT ON TABLE organization_usage IS 'Real-time usage counters for quota enforcement';
COMMENT ON TABLE organization_usage_history IS 'Historical usage data for billing and analytics';
COMMENT ON TABLE subscription_plans IS 'Available subscription plans with default quotas';
COMMENT ON TABLE organization_subscriptions IS 'Organization subscription status and billing';
COMMENT ON FUNCTION check_quota IS 'Check if organization is within quota for a resource';
COMMENT ON FUNCTION check_api_rate_limit IS 'Check and enforce API rate limits';
COMMENT ON FUNCTION set_tenant_context IS 'Set the current tenant context for RLS';
