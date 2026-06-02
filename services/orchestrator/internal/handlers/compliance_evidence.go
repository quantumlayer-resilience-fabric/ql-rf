// PR #24 / CONN-004 — handler-side wrapper around the compliance
// evidence emitter.
//
// Centralizes the best-effort emit call so the two endpoints (/invoke
// and /dry-run) stay simple. Both call this method post-audit-row-insert.
// A failure here MUST NOT fail the request — the audit row is already
// written and the engineering trail is preserved.
package handlers

import (
	"context"

	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/compliance"
)

// tryEmitComplianceEvidence is the single handler-side entry point into
// compliance.EmitToolEvidence. Returns nothing; on error, logs a Warn and
// continues. The emitter silently skips when no mapping is configured or
// when the params carry the B.3 _simulated:true marker.
func (h *Handler) tryEmitComplianceEvidence(
	ctx context.Context,
	invocationID, toolName, riskLevel, orgID string,
	paramsJSON, resultJSON []byte,
) {
	if _, eErr := compliance.EmitToolEvidence(ctx, h.db.Pool, h.log, compliance.EmitOpts{
		InvocationID: invocationID,
		ToolName:     toolName,
		RiskLevel:    riskLevel,
		OrgID:        orgID,
		ParamsJSON:   paramsJSON,
		ResultJSON:   resultJSON,
	}); eErr != nil {
		h.log.Warn("evidence emission failed; audit row preserved",
			"invocation_id", invocationID, "tool", toolName, "error", eErr)
	}
}
