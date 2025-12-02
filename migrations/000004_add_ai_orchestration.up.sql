-- QuantumLayer Resilience Fabric - AI Orchestration Schema
-- Migration: 000004_add_ai_orchestration
-- Description: Adds AI orchestration tables for LLM-first infrastructure operations
-- Reference: ADR-007, ADR-008, ADR-009

-- =============================================================================
-- AI Tasks Table
-- Represents user intent and parsed TaskSpec
-- =============================================================================

CREATE TABLE ai_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- User input
    user_intent TEXT NOT NULL,

    -- Parsed by meta-prompt engine (null until parsing completes)
    task_spec JSONB,

    -- Execution policy (from org defaults or task-specific)
    execution_policy JSONB NOT NULL DEFAULT '{
        "mode": "plan_only",
        "allowed_approver_roles": ["admin"],
        "require_two_approvers": false,
        "timeout_minutes": 30
    }',

    -- LLM usage tracking for cost/performance analysis
    llm_profile JSONB NOT NULL DEFAULT '{
        "model": null,
        "agent_version": null,
        "prompts_version": null,
        "total_tokens": 0,
        "total_latency_ms": 0
    }',

    -- Lifecycle state
    state VARCHAR(31) NOT NULL DEFAULT 'created',
    error TEXT,
    error_code VARCHAR(63),

    -- Metadata
    source VARCHAR(31) NOT NULL DEFAULT 'chat', -- chat, api, scheduled, webhook
    correlation_id VARCHAR(255), -- external correlation (e.g., incident ticket)
    tags JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT ai_tasks_state_check CHECK (state IN ('created', 'parsing', 'planned', 'failed')),
    CONSTRAINT ai_tasks_source_check CHECK (source IN ('chat', 'api', 'scheduled', 'webhook')),
    CONSTRAINT ai_tasks_error_code_check CHECK (error_code IS NULL OR error_code IN (
        'parse_error', 'validation_error', 'scope_error',
        'permission_error', 'timeout', 'llm_error', 'unknown'
    ))
);

-- =============================================================================
-- AI Plans Table
-- AI-generated execution plans
-- =============================================================================

CREATE TABLE ai_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES ai_tasks(id) ON DELETE CASCADE,

    -- Plan type
    type VARCHAR(63) NOT NULL,

    -- Plan content (phases, health checks, rollback conditions)
    payload JSONB NOT NULL,

    -- Validation results
    validation JSONB NOT NULL DEFAULT '{
        "schema_valid": null,
        "schema_errors": [],
        "opa_valid": null,
        "opa_violations": [],
        "safety_valid": null,
        "safety_violations": [],
        "overall_valid": null
    }',

    -- Quality scoring for AI feedback loop (0-100)
    quality_score INT,

    -- Lifecycle state
    state VARCHAR(31) NOT NULL DEFAULT 'draft',

    -- Approval tracking
    approved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMPTZ,
    second_approver UUID REFERENCES users(id) ON DELETE SET NULL,
    second_approved_at TIMESTAMPTZ,
    rejection_reason TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT ai_plans_type_check CHECK (type IN (
        'drift_plan', 'patch_plan', 'dr_runbook', 'compliance_report',
        'incident_analysis', 'cost_optimization_plan', 'security_report', 'image_spec'
    )),
    CONSTRAINT ai_plans_state_check CHECK (state IN (
        'draft', 'validated', 'awaiting_approval', 'approved', 'rejected', 'superseded'
    )),
    CONSTRAINT ai_plans_quality_score_check CHECK (quality_score IS NULL OR (quality_score >= 0 AND quality_score <= 100))
);

-- =============================================================================
-- AI Runs Table
-- Execution records with audit trail
-- =============================================================================

CREATE TABLE ai_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL REFERENCES ai_plans(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES ai_tasks(id) ON DELETE CASCADE, -- denormalized for queries

    -- Execution context
    environment VARCHAR(31) NOT NULL,
    initiated_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Progress tracking
    current_phase VARCHAR(127),
    phases_completed JSONB NOT NULL DEFAULT '[]',
    phases_remaining JSONB NOT NULL DEFAULT '[]',
    percent_complete INT NOT NULL DEFAULT 0,

    -- Lifecycle state
    state VARCHAR(31) NOT NULL DEFAULT 'queued',
    pause_reason TEXT,
    error TEXT,

    -- Outcome metrics (for AI feedback)
    metrics JSONB NOT NULL DEFAULT '{
        "duration_seconds": 0,
        "assets_total": 0,
        "assets_changed": 0,
        "assets_failed": 0,
        "assets_skipped": 0,
        "rollback_triggered": false,
        "rollback_assets": 0,
        "observed_error_rate": 0,
        "health_check_failures": 0
    }',

    -- Audit log (ordered list of events)
    audit_log JSONB NOT NULL DEFAULT '[]',

    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT ai_runs_environment_check CHECK (environment IN ('production', 'staging', 'development', 'sandbox')),
    CONSTRAINT ai_runs_state_check CHECK (state IN ('queued', 'executing', 'paused', 'completed', 'rolled_back', 'failed')),
    CONSTRAINT ai_runs_percent_check CHECK (percent_complete >= 0 AND percent_complete <= 100)
);

-- =============================================================================
-- AI Tool Invocations Table
-- Audit trail for tool usage (for debugging and cost tracking)
-- =============================================================================

CREATE TABLE ai_tool_invocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES ai_tasks(id) ON DELETE CASCADE,
    plan_id UUID REFERENCES ai_plans(id) ON DELETE SET NULL,
    run_id UUID REFERENCES ai_runs(id) ON DELETE SET NULL,

    -- Tool details
    tool_name VARCHAR(127) NOT NULL,
    tool_version VARCHAR(31),
    risk_level VARCHAR(31) NOT NULL,

    -- Invocation details
    parameters JSONB NOT NULL DEFAULT '{}',
    result JSONB,
    error TEXT,

    -- Timing
    duration_ms INT,

    -- Approval (for state-changing tools)
    required_approval BOOLEAN NOT NULL DEFAULT FALSE,
    approved_by UUID REFERENCES users(id) ON DELETE SET NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT ai_tool_invocations_risk_check CHECK (risk_level IN (
        'read_only', 'plan_only', 'state_change_nonprod', 'state_change_prod'
    ))
);

-- =============================================================================
-- AI Prompts Table
-- Versioned prompt templates for regression tracking
-- =============================================================================

CREATE TABLE ai_prompts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Identification
    name VARCHAR(127) NOT NULL,
    version VARCHAR(31) NOT NULL,
    agent_type VARCHAR(63) NOT NULL,

    -- Content
    system_prompt TEXT NOT NULL,
    user_prompt_template TEXT NOT NULL,

    -- Configuration
    model VARCHAR(63) NOT NULL,
    temperature DECIMAL(3,2) NOT NULL DEFAULT 0.3,
    max_tokens INT NOT NULL DEFAULT 4096,

    -- Status
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT ai_prompts_agent_type_check CHECK (agent_type IN (
        'meta_prompt', 'drift_agent', 'patch_agent', 'compliance_agent',
        'incident_agent', 'dr_agent', 'cost_agent', 'security_agent', 'image_agent'
    )),
    CONSTRAINT ai_prompts_temperature_check CHECK (temperature >= 0 AND temperature <= 2),
    UNIQUE(name, version)
);

-- =============================================================================
-- Organization AI Settings Table
-- Tenant-level AI configuration and autonomy settings
-- =============================================================================

CREATE TABLE org_ai_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- Global autonomy mode
    autonomy_mode VARCHAR(31) NOT NULL DEFAULT 'plan_only',

    -- Per-environment overrides
    environment_overrides JSONB NOT NULL DEFAULT '{
        "production": {"mode": "plan_only", "require_two_approvers": true},
        "staging": {"mode": "canary_only"},
        "development": {"mode": "full_auto"}
    }',

    -- Tool-specific overrides
    tool_overrides JSONB NOT NULL DEFAULT '{}',

    -- LLM provider settings
    llm_provider VARCHAR(31) NOT NULL DEFAULT 'anthropic',
    llm_model VARCHAR(63) NOT NULL DEFAULT 'claude-3-5-sonnet-20241022',

    -- Feature flags
    ai_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    auto_remediation_enabled BOOLEAN NOT NULL DEFAULT FALSE,

    -- Cost controls
    monthly_token_budget INT,
    tokens_used_this_month INT NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT org_ai_settings_autonomy_check CHECK (autonomy_mode IN ('plan_only', 'canary_only', 'full_auto')),
    CONSTRAINT org_ai_settings_provider_check CHECK (llm_provider IN ('anthropic', 'azure_openai', 'openai')),
    UNIQUE(org_id)
);

-- =============================================================================
-- Indexes
-- =============================================================================

-- AI Tasks
CREATE INDEX idx_ai_tasks_org_id ON ai_tasks(org_id);
CREATE INDEX idx_ai_tasks_created_by ON ai_tasks(created_by);
CREATE INDEX idx_ai_tasks_state ON ai_tasks(state);
CREATE INDEX idx_ai_tasks_created_at ON ai_tasks(org_id, created_at DESC);
CREATE INDEX idx_ai_tasks_source ON ai_tasks(source);

-- AI Plans
CREATE INDEX idx_ai_plans_task_id ON ai_plans(task_id);
CREATE INDEX idx_ai_plans_state ON ai_plans(state);
CREATE INDEX idx_ai_plans_type ON ai_plans(type);
CREATE INDEX idx_ai_plans_approved_by ON ai_plans(approved_by);

-- AI Runs
CREATE INDEX idx_ai_runs_plan_id ON ai_runs(plan_id);
CREATE INDEX idx_ai_runs_task_id ON ai_runs(task_id);
CREATE INDEX idx_ai_runs_state ON ai_runs(state);
CREATE INDEX idx_ai_runs_environment ON ai_runs(environment);
CREATE INDEX idx_ai_runs_initiated_by ON ai_runs(initiated_by);
CREATE INDEX idx_ai_runs_created_at ON ai_runs(created_at DESC);

-- AI Tool Invocations
CREATE INDEX idx_ai_tool_invocations_task_id ON ai_tool_invocations(task_id);
CREATE INDEX idx_ai_tool_invocations_plan_id ON ai_tool_invocations(plan_id);
CREATE INDEX idx_ai_tool_invocations_run_id ON ai_tool_invocations(run_id);
CREATE INDEX idx_ai_tool_invocations_tool_name ON ai_tool_invocations(tool_name);
CREATE INDEX idx_ai_tool_invocations_risk_level ON ai_tool_invocations(risk_level);

-- AI Prompts
CREATE INDEX idx_ai_prompts_agent_type ON ai_prompts(agent_type);
CREATE INDEX idx_ai_prompts_is_active ON ai_prompts(is_active);

-- =============================================================================
-- Triggers for updated_at
-- =============================================================================

CREATE TRIGGER update_ai_tasks_updated_at
    BEFORE UPDATE ON ai_tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_ai_plans_updated_at
    BEFORE UPDATE ON ai_plans
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_ai_runs_updated_at
    BEFORE UPDATE ON ai_runs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_ai_prompts_updated_at
    BEFORE UPDATE ON ai_prompts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_org_ai_settings_updated_at
    BEFORE UPDATE ON org_ai_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
