-- PR #24 / CONN-004 — down migration. Reverses the up migration.
--
-- Drops the new column and table. Existing compliance_evidence rows
-- survive (manual uploads were never affected by this column); the
-- ai_tool_invocation_id FK link is lost on rollback.

DROP INDEX IF EXISTS idx_evidence_tool_invocation;
ALTER TABLE compliance_evidence DROP COLUMN IF EXISTS ai_tool_invocation_id;

DROP INDEX IF EXISTS idx_tool_mapping_uniq_global;
DROP INDEX IF EXISTS idx_tool_mapping_uniq_org;
DROP INDEX IF EXISTS idx_tool_mapping_lookup;
DROP TABLE IF EXISTS tool_compliance_mappings;
