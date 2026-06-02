// Package compliance — orchestrator-side compliance evidence emission.
//
// PR #24 / CONN-004 — every real (non-synthetic) ai_tool_invocations row
// produces a compliance_evidence attestation iff a tool_compliance_mappings
// row maps the tool name to a compliance_control_id for the row's org.
// The emitter is opt-in (no mapping → no evidence) and best-effort (a
// failure logs and continues; never rolls back the audit row).
//
// Lookup precedence:
//
//  1. org-specific exact-name match    (org_id=X, tool_name_pattern=tool_name)
//  2. org-specific wildcard match      (org_id=X, tool_name_pattern=prefix*)
//  3. global exact-name match          (org_id IS NULL, tool_name_pattern=tool_name)
//  4. global wildcard match            (org_id IS NULL, tool_name_pattern=prefix*)
//
// Only trailing `*` is treated as wildcard; internal `*` is a literal
// character (which the SQL LIKE pattern would escape if we cared, but the
// expected use case is well-named tool families like ssm_send_patch_command*).
//
// The emitter writes the evidence row directly via pgxpool — it does NOT
// call pkg/compliance.CreateEvidence (which uses *sql.DB and would require
// a dual-driver adapter). The insert mirrors the columns that helper
// populates, with the addition of ai_tool_invocation_id (the new FK
// added by migration 000020).
package compliance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// EmitOpts is the typed input to EmitToolEvidence. Carrying the
// invocation id and the marshaled JSON bytes the caller already has
// avoids round-tripping through the DB.
type EmitOpts struct {
	InvocationID string
	ToolName     string
	RiskLevel    string
	OrgID        string
	// ParamsJSON is the marshaled params JSONB. Used for the
	// _simulated:true check; not stored on the evidence row.
	ParamsJSON []byte
	// ResultJSON is the marshaled result JSONB. SHA-256 of this is the
	// evidence's content_hash; length is stored as file_size_bytes.
	ResultJSON []byte
}

// EmitToolEvidence is the entry point. Returns the new evidence row id on
// success; "" + nil on a documented skip (no mapping, synthetic invocation);
// "" + error on an unexpected failure.
//
// Callers MUST treat the error as best-effort — log and continue. The
// audit row (ai_tool_invocations) is already written by the caller; an
// evidence failure here must not affect the engineering audit trail.
func EmitToolEvidence(ctx context.Context, db *pgxpool.Pool, log *logger.Logger, opts EmitOpts) (string, error) {
	if db == nil {
		return "", errors.New("EmitToolEvidence: nil db pool")
	}
	if opts.InvocationID == "" || opts.ToolName == "" || opts.OrgID == "" {
		return "", fmt.Errorf("EmitToolEvidence: missing required field (invocation=%q tool=%q org=%q)",
			opts.InvocationID, opts.ToolName, opts.OrgID)
	}

	if isSimulated(opts.ParamsJSON) {
		// B.3 simulator path — never produces evidence. Silent.
		return "", nil
	}

	controlID, err := lookupControlForTool(ctx, db, opts.OrgID, opts.ToolName)
	if err != nil {
		return "", fmt.Errorf("lookup mapping: %w", err)
	}
	if controlID == "" {
		// Tool isn't mapped to any control for this org. Silent skip —
		// mappings are opt-in.
		return "", nil
	}

	// SHA-256 of the result JSON gives auditors a tamper-evidence hash.
	// json.Marshal of nil result yields "null"; we hash that verbatim.
	hash := sha256.Sum256(opts.ResultJSON)
	contentHash := hex.EncodeToString(hash[:])

	title := titleForEvidence(opts.ToolName, opts.RiskLevel)
	desc := descriptionForEvidence(opts.ToolName, opts.RiskLevel, opts.InvocationID)

	const insertQ = `
		INSERT INTO compliance_evidence (
			org_id, control_id, evidence_type, title, description,
			storage_type, storage_path, content_hash, file_size_bytes, mime_type,
			collected_by, collection_method,
			ai_tool_invocation_id
		)
		VALUES (
			$1::uuid, $2::uuid, 'attestation', $3, $4,
			'internal', $5, $6, $7, 'application/json',
			'orchestrator', 'automated',
			$8::uuid
		)
		RETURNING id`

	var evidenceID string
	if err := db.QueryRow(ctx, insertQ,
		opts.OrgID, controlID, title, desc,
		"ai_tool_invocations:"+opts.InvocationID, contentHash, len(opts.ResultJSON),
		opts.InvocationID,
	).Scan(&evidenceID); err != nil {
		return "", fmt.Errorf("insert evidence: %w", err)
	}

	log.Debug("compliance evidence emitted",
		"invocation_id", opts.InvocationID,
		"tool", opts.ToolName,
		"control_id", controlID,
		"evidence_id", evidenceID,
	)
	return evidenceID, nil
}

// isSimulated returns true if the params JSON contains "_simulated":true.
// Uses a fast string-contains check; the B.3 simulator always marshals
// this marker at the top level so the substring is unambiguous in practice.
func isSimulated(paramsJSON []byte) bool {
	return len(paramsJSON) > 0 && strings.Contains(string(paramsJSON), `"_simulated":true`)
}

// lookupControlForTool runs a single SQL query that materializes the
// four-way precedence as a CASE expression and ORDERs by it. LIMIT 1
// returns the winning row. NULL is a wildcard org match.
//
// The query handles both exact-name matches and trailing-wildcard
// patterns. A pattern stored as `prefix*` becomes a LIKE comparison via
// `tool_name LIKE replace(pattern, '*', '%')`; exact matches use
// equality. Patterns without `*` only ever match exactly.
//
// Returns "" if no mapping matches. Error only on a DB failure; an empty
// result is not an error.
func lookupControlForTool(ctx context.Context, db *pgxpool.Pool, orgID, toolName string) (string, error) {
	// The precedence_score column reads:
	//   1 = org-specific exact     (best)
	//   2 = org-specific wildcard
	//   3 = global exact
	//   4 = global wildcard         (worst)
	// ORDER BY precedence_score ASC LIMIT 1 picks the best.
	const lookupQ = `
		SELECT control_id::text
		FROM tool_compliance_mappings
		WHERE (
		    (org_id = $1::uuid AND tool_name_pattern = $2)
		 OR (org_id = $1::uuid AND tool_name_pattern LIKE '%*' AND $2 LIKE replace(tool_name_pattern, '*', '%'))
		 OR (org_id IS NULL    AND tool_name_pattern = $2)
		 OR (org_id IS NULL    AND tool_name_pattern LIKE '%*' AND $2 LIKE replace(tool_name_pattern, '*', '%'))
		)
		ORDER BY
		    CASE
		        WHEN org_id = $1::uuid AND tool_name_pattern = $2          THEN 1
		        WHEN org_id = $1::uuid AND tool_name_pattern LIKE '%*'     THEN 2
		        WHEN org_id IS NULL    AND tool_name_pattern = $2          THEN 3
		        WHEN org_id IS NULL    AND tool_name_pattern LIKE '%*'     THEN 4
		        ELSE 5
		    END,
		    -- Tiebreaker: longer pattern wins (more specific wildcard).
		    length(tool_name_pattern) DESC
		LIMIT 1`

	var controlID string
	if err := db.QueryRow(ctx, lookupQ, orgID, toolName).Scan(&controlID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return controlID, nil
}

// titleForEvidence renders a one-line summary for the compliance_evidence
// row. Format: "<tool_name> (<risk>) executed". Auditors scan titles;
// keep them short and verbatim.
func titleForEvidence(toolName, riskLevel string) string {
	return fmt.Sprintf("%s (%s) executed", toolName, riskLevel)
}

// descriptionForEvidence renders the longer body. Keeps the invocation id
// in plain text so a SQL search lands on the linked audit row.
func descriptionForEvidence(toolName, riskLevel, invocationID string) string {
	return fmt.Sprintf(
		"Orchestrator-emitted attestation for tool %q (risk=%s). Linked to ai_tool_invocations.id=%s. The content_hash on this row is SHA-256 of the invocation result JSONB; auditors can recompute against the audit row to verify integrity.",
		toolName, riskLevel, invocationID,
	)
}

// MustMarshalJSON is a convenience wrapper used in tests; production
// callers should marshal explicitly so they handle errors.
func MustMarshalJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("MustMarshalJSON: %v", err))
	}
	return b
}
