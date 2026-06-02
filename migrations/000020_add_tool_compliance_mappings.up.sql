-- PR #24 / CONN-004 — wire tool invocations into the compliance evidence trail.
--
-- Two structural additions:
--
--   1. `tool_compliance_mappings` — org-aware lookup from tool name (or
--      prefix wildcard) to a compliance control. NULL org_id is a global
--      default; org-specific rows override.
--
--   2. `compliance_evidence.ai_tool_invocation_id` — nullable FK so an
--      evidence row can link back to the audit row that produced it.
--      Manual evidence uploads leave this NULL (unchanged path).
--
-- Lookup precedence in code (services/orchestrator/internal/compliance/
-- evidence_emitter.go): org-exact > org-wildcard > global-exact >
-- global-wildcard. A missing mapping is a silent skip — tools opt INTO
-- evidence emission, they do not opt out.

CREATE TABLE IF NOT EXISTS tool_compliance_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- NULL org_id = global default. The lookup query checks org-specific
    -- rows first and falls back to global rows; per-org rows let a
    -- customer override scoring without affecting other tenants.
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,

    -- Either an exact tool name (e.g. "ssm_send_patch_command") OR a
    -- trailing-wildcard pattern (e.g. "ssm_send_patch_command*"). Only
    -- trailing `*` is treated as wildcard; internal `*` is a literal.
    tool_name_pattern VARCHAR(127) NOT NULL,

    -- The compliance control this mapping attests to.
    control_id UUID NOT NULL REFERENCES compliance_controls(id) ON DELETE CASCADE,

    -- Operator-readable note: why was this mapping added? What does this
    -- tool prove? Surfaces in admin UI when one ships.
    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Lookup index — covers the common path (org_id, pattern) for the
-- evidence emitter's WHERE clause.
CREATE INDEX idx_tool_mapping_lookup
    ON tool_compliance_mappings (org_id, tool_name_pattern);

-- Uniqueness — same (org, pattern, control) tuple must not appear twice.
-- Postgres NULL semantics mean a single UNIQUE constraint over
-- (org_id, pattern, control_id) treats two NULL org_ids as distinct, so
-- we split into two partial unique indexes.
CREATE UNIQUE INDEX idx_tool_mapping_uniq_org
    ON tool_compliance_mappings (org_id, tool_name_pattern, control_id)
    WHERE org_id IS NOT NULL;

CREATE UNIQUE INDEX idx_tool_mapping_uniq_global
    ON tool_compliance_mappings (tool_name_pattern, control_id)
    WHERE org_id IS NULL;

-- compliance_evidence gets a nullable FK back to the audit row that
-- produced the attestation. ON DELETE SET NULL: deleting an audit row
-- (rare; mostly cascades via task delete) preserves the evidence trail
-- but breaks the engineering linkage. SET NULL communicates this more
-- accurately than CASCADE.
ALTER TABLE compliance_evidence
    ADD COLUMN IF NOT EXISTS ai_tool_invocation_id UUID REFERENCES ai_tool_invocations(id) ON DELETE SET NULL;

-- Partial index — most evidence rows are manual uploads (NULL); index
-- only the orchestrator-emitted rows so /compliance UI can resolve
-- "show me the audit row behind this evidence" in O(log N).
CREATE INDEX IF NOT EXISTS idx_evidence_tool_invocation
    ON compliance_evidence (ai_tool_invocation_id)
    WHERE ai_tool_invocation_id IS NOT NULL;
