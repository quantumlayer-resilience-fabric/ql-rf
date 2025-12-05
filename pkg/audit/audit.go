// Package audit provides enterprise audit logging with SIEM integration.
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// Logger provides audit logging functionality.
type Logger struct {
	db  *pgxpool.Pool
	log *logger.Logger
}

// NewLogger creates a new audit logger.
func NewLogger(db *pgxpool.Pool, log *logger.Logger) *Logger {
	return &Logger{
		db:  db,
		log: log.WithComponent("audit"),
	}
}

// Entry represents an audit log entry.
type Entry struct {
	// Actor information
	ActorType     ActorType `json:"actor_type"`
	ActorID       string    `json:"actor_id"`
	ActorEmail    string    `json:"actor_email,omitempty"`
	ActorIP       net.IP    `json:"actor_ip,omitempty"`
	ActorUserAgent string   `json:"actor_user_agent,omitempty"`

	// Organization context
	OrgID uuid.UUID `json:"org_id"`

	// Action details
	Action         string         `json:"action"`
	ActionCategory ActionCategory `json:"action_category"`

	// Resource affected
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id,omitempty"`
	ResourceName string `json:"resource_name,omitempty"`

	// Change tracking
	Changes map[string]Change `json:"changes,omitempty"`

	// Request context
	RequestID  uuid.UUID `json:"request_id,omitempty"`
	SessionID  string    `json:"session_id,omitempty"`
	APIVersion string    `json:"api_version,omitempty"`

	// Additional context
	Context map[string]interface{} `json:"context,omitempty"`

	// Risk and compliance
	RiskLevel          RiskLevel `json:"risk_level,omitempty"`
	ComplianceRelevant bool      `json:"compliance_relevant,omitempty"`
	PIIAccessed        bool      `json:"pii_accessed,omitempty"`

	// Outcome
	Status       Status `json:"status"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`

	// Duration
	DurationMS int `json:"duration_ms,omitempty"`
}

// Change represents a field change.
type Change struct {
	Old interface{} `json:"old"`
	New interface{} `json:"new"`
}

// ActorType defines who performed the action.
type ActorType string

const (
	ActorTypeUser   ActorType = "user"
	ActorTypeSystem ActorType = "system"
	ActorTypeAgent  ActorType = "agent"
	ActorTypeAPIKey ActorType = "api_key"
)

// ActionCategory classifies the action type.
type ActionCategory string

const (
	ActionCategoryRead    ActionCategory = "read"
	ActionCategoryCreate  ActionCategory = "create"
	ActionCategoryUpdate  ActionCategory = "update"
	ActionCategoryDelete  ActionCategory = "delete"
	ActionCategoryExecute ActionCategory = "execute"
)

// RiskLevel classifies the risk of the action.
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// Status indicates the outcome of the action.
type Status string

const (
	StatusSuccess Status = "success"
	StatusFailure Status = "failure"
	StatusDenied  Status = "denied"
)

// Log writes an audit entry to the database.
func (l *Logger) Log(ctx context.Context, entry Entry) error {
	changesJSON, err := json.Marshal(entry.Changes)
	if err != nil {
		changesJSON = []byte("{}")
	}

	contextJSON, err := json.Marshal(entry.Context)
	if err != nil {
		contextJSON = []byte("{}")
	}

	var actorIP *string
	if entry.ActorIP != nil {
		ip := entry.ActorIP.String()
		actorIP = &ip
	}

	var requestID *uuid.UUID
	if entry.RequestID != uuid.Nil {
		requestID = &entry.RequestID
	}

	query := `
		INSERT INTO audit_logs (
			actor_type, actor_id, actor_email, actor_ip_address, actor_user_agent,
			org_id, action, action_category,
			resource_type, resource_id, resource_name,
			changes, request_id, session_id, api_version,
			context, risk_level, compliance_relevant, pii_accessed,
			status, error_code, error_message, duration_ms
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11,
			$12, $13, $14, $15,
			$16, $17, $18, $19,
			$20, $21, $22, $23
		)
	`

	_, err = l.db.Exec(ctx, query,
		entry.ActorType, entry.ActorID, entry.ActorEmail, actorIP, entry.ActorUserAgent,
		entry.OrgID, entry.Action, entry.ActionCategory,
		entry.ResourceType, entry.ResourceID, entry.ResourceName,
		changesJSON, requestID, entry.SessionID, entry.APIVersion,
		contextJSON, entry.RiskLevel, entry.ComplianceRelevant, entry.PIIAccessed,
		entry.Status, entry.ErrorCode, entry.ErrorMessage, entry.DurationMS,
	)

	if err != nil {
		l.log.Error("failed to write audit log", "error", err, "action", entry.Action)
		return fmt.Errorf("failed to write audit log: %w", err)
	}

	return nil
}

// LogAsync writes an audit entry asynchronously (fire and forget).
func (l *Logger) LogAsync(ctx context.Context, entry Entry) {
	go func() {
		// Create a new context with timeout since the original may be cancelled
		logCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := l.Log(logCtx, entry); err != nil {
			l.log.Error("async audit log failed", "error", err, "action", entry.Action)
		}
	}()
}

// Query retrieves audit logs with filters.
func (l *Logger) Query(ctx context.Context, filters QueryFilters) ([]AuditLogRow, error) {
	query := `
		SELECT
			id, timestamp, actor_type, actor_id, actor_email, actor_ip_address,
			org_id, action, action_category,
			resource_type, resource_id, resource_name,
			changes, request_id, context,
			risk_level, compliance_relevant, pii_accessed,
			status, error_code, error_message, duration_ms,
			integrity_hash
		FROM audit_logs
		WHERE org_id = $1
	`
	args := []interface{}{filters.OrgID}
	argIdx := 2

	if filters.ActorID != "" {
		query += fmt.Sprintf(" AND actor_id = $%d", argIdx)
		args = append(args, filters.ActorID)
		argIdx++
	}

	if filters.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, filters.Action)
		argIdx++
	}

	if filters.ResourceType != "" {
		query += fmt.Sprintf(" AND resource_type = $%d", argIdx)
		args = append(args, filters.ResourceType)
		argIdx++
	}

	if filters.ResourceID != "" {
		query += fmt.Sprintf(" AND resource_id = $%d", argIdx)
		args = append(args, filters.ResourceID)
		argIdx++
	}

	if filters.RiskLevel != "" {
		query += fmt.Sprintf(" AND risk_level = $%d", argIdx)
		args = append(args, filters.RiskLevel)
		argIdx++
	}

	if !filters.StartTime.IsZero() {
		query += fmt.Sprintf(" AND timestamp >= $%d", argIdx)
		args = append(args, filters.StartTime)
		argIdx++
	}

	if !filters.EndTime.IsZero() {
		query += fmt.Sprintf(" AND timestamp <= $%d", argIdx)
		args = append(args, filters.EndTime)
		argIdx++
	}

	if filters.ComplianceOnly {
		query += " AND compliance_relevant = TRUE"
	}

	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filters.Status)
		argIdx++
	}

	query += " ORDER BY timestamp DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filters.Limit)
	} else {
		query += " LIMIT 100"
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filters.Offset)
	}

	rows, err := l.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var results []AuditLogRow
	for rows.Next() {
		var row AuditLogRow
		var changes, context []byte
		var actorIP *string
		var requestID *uuid.UUID

		err := rows.Scan(
			&row.ID, &row.Timestamp, &row.ActorType, &row.ActorID, &row.ActorEmail, &actorIP,
			&row.OrgID, &row.Action, &row.ActionCategory,
			&row.ResourceType, &row.ResourceID, &row.ResourceName,
			&changes, &requestID, &context,
			&row.RiskLevel, &row.ComplianceRelevant, &row.PIIAccessed,
			&row.Status, &row.ErrorCode, &row.ErrorMessage, &row.DurationMS,
			&row.IntegrityHash,
		)
		if err != nil {
			l.log.Warn("failed to scan audit row", "error", err)
			continue
		}

		if actorIP != nil {
			row.ActorIP = *actorIP
		}
		if requestID != nil {
			row.RequestID = *requestID
		}

		json.Unmarshal(changes, &row.Changes)
		json.Unmarshal(context, &row.Context)

		results = append(results, row)
	}

	return results, nil
}

// VerifyIntegrity verifies the hash chain integrity for a time range.
func (l *Logger) VerifyIntegrity(ctx context.Context, orgID uuid.UUID, startTime, endTime time.Time) (*IntegrityReport, error) {
	query := `
		SELECT id, timestamp, integrity_hash, previous_hash
		FROM audit_logs
		WHERE org_id = $1 AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp ASC
	`

	rows, err := l.db.Query(ctx, query, orgID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	report := &IntegrityReport{
		OrgID:     orgID,
		StartTime: startTime,
		EndTime:   endTime,
		Valid:     true,
	}

	var prevHash string
	for rows.Next() {
		var id uuid.UUID
		var timestamp time.Time
		var hash, previousHash string

		err := rows.Scan(&id, &timestamp, &hash, &previousHash)
		if err != nil {
			continue
		}

		report.TotalEntries++

		// Verify hash chain
		if prevHash != "" && previousHash != prevHash {
			report.Valid = false
			report.Violations = append(report.Violations, IntegrityViolation{
				EntryID:      id,
				Timestamp:    timestamp,
				ExpectedHash: prevHash,
				ActualHash:   previousHash,
			})
		}

		prevHash = hash
	}

	report.VerifiedAt = time.Now()
	return report, nil
}

// QueryFilters contains filters for querying audit logs.
type QueryFilters struct {
	OrgID          uuid.UUID
	ActorID        string
	Action         string
	ResourceType   string
	ResourceID     string
	RiskLevel      string
	StartTime      time.Time
	EndTime        time.Time
	ComplianceOnly bool
	Status         string
	Limit          int
	Offset         int
}

// AuditLogRow represents a row from the audit_logs table.
type AuditLogRow struct {
	ID                 uuid.UUID              `json:"id"`
	Timestamp          time.Time              `json:"timestamp"`
	ActorType          string                 `json:"actor_type"`
	ActorID            string                 `json:"actor_id"`
	ActorEmail         string                 `json:"actor_email,omitempty"`
	ActorIP            string                 `json:"actor_ip,omitempty"`
	OrgID              uuid.UUID              `json:"org_id"`
	Action             string                 `json:"action"`
	ActionCategory     string                 `json:"action_category"`
	ResourceType       string                 `json:"resource_type"`
	ResourceID         string                 `json:"resource_id,omitempty"`
	ResourceName       string                 `json:"resource_name,omitempty"`
	Changes            map[string]Change      `json:"changes,omitempty"`
	RequestID          uuid.UUID              `json:"request_id,omitempty"`
	Context            map[string]interface{} `json:"context,omitempty"`
	RiskLevel          string                 `json:"risk_level,omitempty"`
	ComplianceRelevant bool                   `json:"compliance_relevant"`
	PIIAccessed        bool                   `json:"pii_accessed"`
	Status             string                 `json:"status"`
	ErrorCode          string                 `json:"error_code,omitempty"`
	ErrorMessage       string                 `json:"error_message,omitempty"`
	DurationMS         int                    `json:"duration_ms,omitempty"`
	IntegrityHash      string                 `json:"integrity_hash"`
}

// IntegrityReport contains the result of an integrity verification.
type IntegrityReport struct {
	OrgID        uuid.UUID            `json:"org_id"`
	StartTime    time.Time            `json:"start_time"`
	EndTime      time.Time            `json:"end_time"`
	TotalEntries int                  `json:"total_entries"`
	Valid        bool                 `json:"valid"`
	Violations   []IntegrityViolation `json:"violations,omitempty"`
	VerifiedAt   time.Time            `json:"verified_at"`
}

// IntegrityViolation represents a hash chain violation.
type IntegrityViolation struct {
	EntryID      uuid.UUID `json:"entry_id"`
	Timestamp    time.Time `json:"timestamp"`
	ExpectedHash string    `json:"expected_hash"`
	ActualHash   string    `json:"actual_hash"`
}

// Predefined action strings for consistency
const (
	// Asset actions
	ActionAssetCreate = "asset.create"
	ActionAssetUpdate = "asset.update"
	ActionAssetDelete = "asset.delete"
	ActionAssetView   = "asset.view"

	// Image actions
	ActionImageCreate  = "image.create"
	ActionImagePromote = "image.promote"
	ActionImageDeprecate = "image.deprecate"
	ActionImageDelete  = "image.delete"

	// AI Task actions
	ActionAITaskCreate  = "ai.task.create"
	ActionAITaskApprove = "ai.task.approve"
	ActionAITaskReject  = "ai.task.reject"
	ActionAITaskExecute = "ai.task.execute"
	ActionAITaskCancel  = "ai.task.cancel"

	// User actions
	ActionUserLogin   = "user.login"
	ActionUserLogout  = "user.logout"
	ActionUserCreate  = "user.create"
	ActionUserUpdate  = "user.update"
	ActionUserDelete  = "user.delete"

	// Connector actions
	ActionConnectorConnect    = "connector.connect"
	ActionConnectorDisconnect = "connector.disconnect"
	ActionConnectorSync       = "connector.sync"

	// Patching actions
	ActionPatchScan     = "patch.scan"
	ActionPatchInstall  = "patch.install"
	ActionPatchApprove  = "patch.approve"
	ActionPatchSchedule = "patch.schedule"

	// Compliance actions
	ActionComplianceScan   = "compliance.scan"
	ActionComplianceExport = "compliance.export"
)

// Helper functions for common audit patterns

// LogUserAction logs a user-initiated action.
func (l *Logger) LogUserAction(ctx context.Context, userID, email string, orgID uuid.UUID, action string, resource ResourceInfo, status Status) {
	l.LogAsync(ctx, Entry{
		ActorType:      ActorTypeUser,
		ActorID:        userID,
		ActorEmail:     email,
		OrgID:          orgID,
		Action:         action,
		ActionCategory: categorizeAction(action),
		ResourceType:   resource.Type,
		ResourceID:     resource.ID,
		ResourceName:   resource.Name,
		Status:         status,
		RiskLevel:      assessRisk(action),
	})
}

// LogSystemAction logs a system-initiated action.
func (l *Logger) LogSystemAction(ctx context.Context, orgID uuid.UUID, action string, resource ResourceInfo, status Status) {
	l.LogAsync(ctx, Entry{
		ActorType:      ActorTypeSystem,
		ActorID:        "system",
		OrgID:          orgID,
		Action:         action,
		ActionCategory: categorizeAction(action),
		ResourceType:   resource.Type,
		ResourceID:     resource.ID,
		ResourceName:   resource.Name,
		Status:         status,
	})
}

// LogAgentAction logs an AI agent action.
func (l *Logger) LogAgentAction(ctx context.Context, agentName string, orgID uuid.UUID, action string, resource ResourceInfo, status Status) {
	l.LogAsync(ctx, Entry{
		ActorType:      ActorTypeAgent,
		ActorID:        agentName,
		OrgID:          orgID,
		Action:         action,
		ActionCategory: categorizeAction(action),
		ResourceType:   resource.Type,
		ResourceID:     resource.ID,
		ResourceName:   resource.Name,
		Status:         status,
		RiskLevel:      assessRisk(action),
	})
}

// ResourceInfo contains information about a resource.
type ResourceInfo struct {
	Type string
	ID   string
	Name string
}

func categorizeAction(action string) ActionCategory {
	switch {
	case action[len(action)-6:] == "create":
		return ActionCategoryCreate
	case action[len(action)-6:] == "update":
		return ActionCategoryUpdate
	case action[len(action)-6:] == "delete":
		return ActionCategoryDelete
	case action[len(action)-4:] == "view":
		return ActionCategoryRead
	default:
		return ActionCategoryExecute
	}
}

func assessRisk(action string) RiskLevel {
	highRiskActions := map[string]bool{
		ActionUserDelete:    true,
		ActionAssetDelete:   true,
		ActionImageDelete:   true,
		ActionAITaskExecute: true,
		ActionPatchInstall:  true,
	}

	if highRiskActions[action] {
		return RiskLevelHigh
	}
	return RiskLevelLow
}
