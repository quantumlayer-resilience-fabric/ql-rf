// Package compliance provides compliance framework management and assessment capabilities.
// Supports CIS, SOC2, NIST and other compliance frameworks with control mappings and evidence tracking.
package compliance

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Framework represents a compliance framework (CIS, SOC2, NIST, etc.).
type Framework struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	Name           string     `json:"name" db:"name"`
	Description    string     `json:"description,omitempty" db:"description"`
	Category       string     `json:"category,omitempty" db:"category"`
	Version        string     `json:"version,omitempty" db:"version"`
	RegulatoryBody string     `json:"regulatory_body,omitempty" db:"regulatory_body"`
	EffectiveDate  *time.Time `json:"effective_date,omitempty" db:"effective_date"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// Severity represents control severity levels.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// AutomationSupport represents the level of automation support for a control.
type AutomationSupport string

const (
	AutomationFull    AutomationSupport = "automated"
	AutomationPartial AutomationSupport = "hybrid"
	AutomationManual  AutomationSupport = "manual"
)

// Control represents a compliance control within a framework.
type Control struct {
	ID             uuid.UUID `json:"id" db:"id"`
	FrameworkID    uuid.UUID `json:"framework_id" db:"framework_id"`
	ControlID      string    `json:"control_id" db:"control_id"`
	Name           string    `json:"name" db:"title"` // Maps to 'title' column in DB
	Description    string    `json:"description,omitempty" db:"description"`
	Severity       Severity  `json:"severity" db:"severity"`
	Recommendation string    `json:"recommendation,omitempty" db:"recommendation"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// ControlMapping represents a mapping between controls in different frameworks.
type ControlMapping struct {
	ID              uuid.UUID `json:"id" db:"id"`
	SourceControlID uuid.UUID `json:"source_control_id" db:"source_control_id"`
	TargetControlID uuid.UUID `json:"target_control_id" db:"target_control_id"`
	MappingType     string    `json:"mapping_type" db:"mapping_type"` // equivalent, partial, related
	ConfidenceScore float64   `json:"confidence_score" db:"confidence_score"`
	Notes           string    `json:"notes,omitempty" db:"notes"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// EvidenceType represents the type of compliance evidence.
type EvidenceType string

const (
	EvidenceScreenshot   EvidenceType = "screenshot"
	EvidenceLog          EvidenceType = "log"
	EvidenceConfig       EvidenceType = "config"
	EvidenceReport       EvidenceType = "report"
	EvidenceAttestation  EvidenceType = "attestation"
)

// Evidence represents compliance evidence for a control.
type Evidence struct {
	ID               uuid.UUID    `json:"id" db:"id"`
	OrgID            uuid.UUID    `json:"org_id" db:"org_id"`
	ControlID        uuid.UUID    `json:"control_id" db:"control_id"`
	EvidenceType     EvidenceType `json:"evidence_type" db:"evidence_type"`
	Title            string       `json:"title" db:"title"`
	Description      string       `json:"description,omitempty" db:"description"`
	StorageType      string       `json:"storage_type" db:"storage_type"`
	StoragePath      string       `json:"storage_path,omitempty" db:"storage_path"`
	ContentHash      string       `json:"content_hash,omitempty" db:"content_hash"`
	FileSizeBytes    int64        `json:"file_size_bytes,omitempty" db:"file_size_bytes"`
	MimeType         string       `json:"mime_type,omitempty" db:"mime_type"`
	CollectedAt      time.Time    `json:"collected_at" db:"collected_at"`
	CollectedBy      string       `json:"collected_by,omitempty" db:"collected_by"`
	CollectionMethod string       `json:"collection_method,omitempty" db:"collection_method"`
	ValidFrom        time.Time    `json:"valid_from" db:"valid_from"`
	ValidUntil       *time.Time   `json:"valid_until,omitempty" db:"valid_until"`
	IsCurrent        bool         `json:"is_current" db:"is_current"`
	ReviewedBy       string       `json:"reviewed_by,omitempty" db:"reviewed_by"`
	ReviewedAt       *time.Time   `json:"reviewed_at,omitempty" db:"reviewed_at"`
	ReviewStatus     string       `json:"review_status,omitempty" db:"review_status"`
	ReviewNotes      string       `json:"review_notes,omitempty" db:"review_notes"`
	CreatedAt        time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at" db:"updated_at"`
}

// AssessmentStatus represents the status of a compliance assessment.
type AssessmentStatus string

const (
	AssessmentPending    AssessmentStatus = "pending"
	AssessmentInProgress AssessmentStatus = "in_progress"
	AssessmentCompleted  AssessmentStatus = "completed"
	AssessmentFailed     AssessmentStatus = "failed"
)

// Assessment represents a compliance assessment run.
type Assessment struct {
	ID              uuid.UUID        `json:"id" db:"id"`
	OrgID           uuid.UUID        `json:"org_id" db:"org_id"`
	FrameworkID     uuid.UUID        `json:"framework_id" db:"framework_id"`
	AssessmentType  string           `json:"assessment_type" db:"assessment_type"`
	Name            string           `json:"name" db:"name"`
	Description     string           `json:"description,omitempty" db:"description"`
	ScopeSites      []uuid.UUID      `json:"scope_sites,omitempty" db:"scope_sites"`
	ScopeAssets     []uuid.UUID      `json:"scope_assets,omitempty" db:"scope_assets"`
	Status          AssessmentStatus `json:"status" db:"status"`
	StartedAt       *time.Time       `json:"started_at,omitempty" db:"started_at"`
	CompletedAt     *time.Time       `json:"completed_at,omitempty" db:"completed_at"`
	TotalControls   int              `json:"total_controls" db:"total_controls"`
	PassedControls  int              `json:"passed_controls" db:"passed_controls"`
	FailedControls  int              `json:"failed_controls" db:"failed_controls"`
	NotApplicable   int              `json:"not_applicable" db:"not_applicable"`
	Score           float64          `json:"score,omitempty" db:"score"`
	InitiatedBy     string           `json:"initiated_by" db:"initiated_by"`
	CreatedAt       time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at" db:"updated_at"`
}

// ControlResultStatus represents the result status of a control assessment.
type ControlResultStatus string

const (
	ControlPassed       ControlResultStatus = "passed"
	ControlFailed       ControlResultStatus = "failed"
	ControlNotApplicable ControlResultStatus = "not_applicable"
	ControlManualReview ControlResultStatus = "manual_review"
)

// AssessmentResult represents the result of assessing a single control.
type AssessmentResult struct {
	ID                  uuid.UUID           `json:"id" db:"id"`
	AssessmentID        uuid.UUID           `json:"assessment_id" db:"assessment_id"`
	ControlID           uuid.UUID           `json:"control_id" db:"control_id"`
	Status              ControlResultStatus `json:"status" db:"status"`
	Score               float64             `json:"score,omitempty" db:"score"`
	Findings            string              `json:"findings,omitempty" db:"findings"`
	RemediationGuidance string              `json:"remediation_guidance,omitempty" db:"remediation_guidance"`
	EvidenceIDs         []uuid.UUID         `json:"evidence_ids,omitempty" db:"evidence_ids"`
	CheckOutput         map[string]any      `json:"check_output,omitempty" db:"check_output"`
	CheckDurationMs     int                 `json:"check_duration_ms,omitempty" db:"check_duration_ms"`
	EvaluatedAt         time.Time           `json:"evaluated_at" db:"evaluated_at"`
}

// Exemption represents a control exemption.
type Exemption struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	OrgID                uuid.UUID  `json:"org_id" db:"org_id"`
	ControlID            uuid.UUID  `json:"control_id" db:"control_id"`
	AssetID              *uuid.UUID `json:"asset_id,omitempty" db:"asset_id"`
	SiteID               *uuid.UUID `json:"site_id,omitempty" db:"site_id"`
	Reason               string     `json:"reason" db:"reason"`
	RiskAcceptance       string     `json:"risk_acceptance,omitempty" db:"risk_acceptance"`
	CompensatingControls string     `json:"compensating_controls,omitempty" db:"compensating_controls"`
	ApprovedBy           string     `json:"approved_by" db:"approved_by"`
	ApprovedAt           time.Time  `json:"approved_at" db:"approved_at"`
	ExpiresAt            time.Time  `json:"expires_at" db:"expires_at"`
	LastReviewedAt       *time.Time `json:"last_reviewed_at,omitempty" db:"last_reviewed_at"`
	LastReviewedBy       string     `json:"last_reviewed_by,omitempty" db:"last_reviewed_by"`
	ReviewFrequencyDays  int        `json:"review_frequency_days" db:"review_frequency_days"`
	Status               string     `json:"status" db:"status"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

// Service provides compliance management functionality.
type Service struct {
	db *sql.DB
}

// NewService creates a new compliance service.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// ListFrameworks returns all compliance frameworks.
func (s *Service) ListFrameworks(ctx context.Context) ([]Framework, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, category, version, regulatory_body, effective_date, created_at, updated_at
		FROM compliance_frameworks
		ORDER BY category, name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list frameworks: %w", err)
	}
	defer rows.Close()

	var frameworks []Framework
	for rows.Next() {
		var f Framework
		var description, category, version, regulatoryBody sql.NullString
		if err := rows.Scan(&f.ID, &f.Name, &description, &category, &version,
			&regulatoryBody, &f.EffectiveDate, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan framework: %w", err)
		}
		f.Description = description.String
		f.Category = category.String
		f.Version = version.String
		f.RegulatoryBody = regulatoryBody.String
		frameworks = append(frameworks, f)
	}

	return frameworks, rows.Err()
}

// GetFramework returns a framework by ID.
func (s *Service) GetFramework(ctx context.Context, frameworkID uuid.UUID) (*Framework, error) {
	var f Framework
	var description, category, version, regulatoryBody sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, category, version, regulatory_body, effective_date, created_at, updated_at
		FROM compliance_frameworks WHERE id = $1
	`, frameworkID).Scan(&f.ID, &f.Name, &description, &category, &version,
		&regulatoryBody, &f.EffectiveDate, &f.CreatedAt, &f.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get framework: %w", err)
	}
	f.Description = description.String
	f.Category = category.String
	f.Version = version.String
	f.RegulatoryBody = regulatoryBody.String
	return &f, nil
}

// ListControls returns all controls for a framework.
func (s *Service) ListControls(ctx context.Context, frameworkID uuid.UUID) ([]Control, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, framework_id, control_id, title, description, severity, recommendation, created_at
		FROM compliance_controls
		WHERE framework_id = $1
		ORDER BY control_id
	`, frameworkID)
	if err != nil {
		return nil, fmt.Errorf("failed to list controls: %w", err)
	}
	defer rows.Close()

	var controls []Control
	for rows.Next() {
		var c Control
		var description, recommendation sql.NullString
		if err := rows.Scan(&c.ID, &c.FrameworkID, &c.ControlID, &c.Name, &description,
			&c.Severity, &recommendation, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan control: %w", err)
		}
		c.Description = description.String
		c.Recommendation = recommendation.String
		controls = append(controls, c)
	}

	return controls, rows.Err()
}

// GetControl returns a control by ID.
func (s *Service) GetControl(ctx context.Context, controlID uuid.UUID) (*Control, error) {
	var c Control
	var description, recommendation sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, framework_id, control_id, title, description, severity, recommendation, created_at
		FROM compliance_controls WHERE id = $1
	`, controlID).Scan(&c.ID, &c.FrameworkID, &c.ControlID, &c.Name, &description,
		&c.Severity, &recommendation, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get control: %w", err)
	}
	c.Description = description.String
	c.Recommendation = recommendation.String
	return &c, nil
}

// GetMappedControls returns controls that map to a given control.
// Note: Returns empty slice if control_mappings table doesn't exist.
func (s *Service) GetMappedControls(ctx context.Context, controlID uuid.UUID) ([]Control, error) {
	// Check if control_mappings table exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = 'control_mappings'
		)
	`).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check control_mappings table: %w", err)
	}
	if !exists {
		return []Control{}, nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.framework_id, c.control_id, c.title, c.description, c.severity,
		       c.recommendation, c.created_at
		FROM compliance_controls c
		JOIN control_mappings m ON c.id = m.target_control_id
		WHERE m.source_control_id = $1
		UNION
		SELECT c.id, c.framework_id, c.control_id, c.title, c.description, c.severity,
		       c.recommendation, c.created_at
		FROM compliance_controls c
		JOIN control_mappings m ON c.id = m.source_control_id
		WHERE m.target_control_id = $1
	`, controlID)
	if err != nil {
		return nil, fmt.Errorf("failed to get mapped controls: %w", err)
	}
	defer rows.Close()

	var controls []Control
	for rows.Next() {
		var c Control
		var description, recommendation sql.NullString
		if err := rows.Scan(&c.ID, &c.FrameworkID, &c.ControlID, &c.Name, &description,
			&c.Severity, &recommendation, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan control: %w", err)
		}
		c.Description = description.String
		c.Recommendation = recommendation.String
		controls = append(controls, c)
	}

	return controls, rows.Err()
}

// CreateAssessment creates a new compliance assessment.
func (s *Service) CreateAssessment(ctx context.Context, assessment Assessment) (*Assessment, error) {
	assessment.ID = uuid.New()
	assessment.Status = AssessmentPending
	assessment.CreatedAt = time.Now()
	assessment.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO compliance_assessments (
			id, org_id, framework_id, assessment_type, name, description,
			scope_sites, scope_assets, status, initiated_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, assessment.ID, assessment.OrgID, assessment.FrameworkID, assessment.AssessmentType,
		assessment.Name, assessment.Description, assessment.ScopeSites, assessment.ScopeAssets,
		assessment.Status, assessment.InitiatedBy, assessment.CreatedAt, assessment.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create assessment: %w", err)
	}

	return &assessment, nil
}

// StartAssessment starts an assessment.
func (s *Service) StartAssessment(ctx context.Context, assessmentID uuid.UUID) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE compliance_assessments
		SET status = $1, started_at = $2, updated_at = $3
		WHERE id = $4
	`, AssessmentInProgress, now, now, assessmentID)
	if err != nil {
		return fmt.Errorf("failed to start assessment: %w", err)
	}
	return nil
}

// CompleteAssessment completes an assessment with results.
func (s *Service) CompleteAssessment(ctx context.Context, assessmentID uuid.UUID, passed, failed, notApplicable int, score float64) error {
	now := time.Now()
	total := passed + failed + notApplicable
	_, err := s.db.ExecContext(ctx, `
		UPDATE compliance_assessments
		SET status = $1, completed_at = $2, total_controls = $3,
		    passed_controls = $4, failed_controls = $5, not_applicable = $6,
		    score = $7, updated_at = $8
		WHERE id = $9
	`, AssessmentCompleted, now, total, passed, failed, notApplicable, score, now, assessmentID)
	if err != nil {
		return fmt.Errorf("failed to complete assessment: %w", err)
	}
	return nil
}

// GetAssessment returns an assessment by ID.
func (s *Service) GetAssessment(ctx context.Context, assessmentID uuid.UUID) (*Assessment, error) {
	var a Assessment
	err := s.db.QueryRowContext(ctx, `
		SELECT id, org_id, framework_id, assessment_type, name, description,
		       scope_sites, scope_assets, status, started_at, completed_at,
		       total_controls, passed_controls, failed_controls, not_applicable,
		       score, initiated_by, created_at, updated_at
		FROM compliance_assessments WHERE id = $1
	`, assessmentID).Scan(
		&a.ID, &a.OrgID, &a.FrameworkID, &a.AssessmentType, &a.Name, &a.Description,
		&a.ScopeSites, &a.ScopeAssets, &a.Status, &a.StartedAt, &a.CompletedAt,
		&a.TotalControls, &a.PassedControls, &a.FailedControls, &a.NotApplicable,
		&a.Score, &a.InitiatedBy, &a.CreatedAt, &a.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get assessment: %w", err)
	}
	return &a, nil
}

// ListAssessments returns assessments for an organization.
func (s *Service) ListAssessments(ctx context.Context, orgID uuid.UUID, limit int) ([]Assessment, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, org_id, framework_id, assessment_type, name, description,
		       scope_sites, scope_assets, status, started_at, completed_at,
		       total_controls, passed_controls, failed_controls, not_applicable,
		       score, initiated_by, created_at, updated_at
		FROM compliance_assessments
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list assessments: %w", err)
	}
	defer rows.Close()

	var assessments []Assessment
	for rows.Next() {
		var a Assessment
		if err := rows.Scan(
			&a.ID, &a.OrgID, &a.FrameworkID, &a.AssessmentType, &a.Name, &a.Description,
			&a.ScopeSites, &a.ScopeAssets, &a.Status, &a.StartedAt, &a.CompletedAt,
			&a.TotalControls, &a.PassedControls, &a.FailedControls, &a.NotApplicable,
			&a.Score, &a.InitiatedBy, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan assessment: %w", err)
		}
		assessments = append(assessments, a)
	}

	return assessments, rows.Err()
}

// RecordControlResult records the result of assessing a control.
func (s *Service) RecordControlResult(ctx context.Context, result AssessmentResult) error {
	result.ID = uuid.New()
	result.EvaluatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO compliance_assessment_results (
			id, assessment_id, control_id, status, score, findings,
			remediation_guidance, evidence_ids, check_output, check_duration_ms, evaluated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, result.ID, result.AssessmentID, result.ControlID, result.Status,
		result.Score, result.Findings, result.RemediationGuidance, result.EvidenceIDs,
		result.CheckOutput, result.CheckDurationMs, result.EvaluatedAt)
	if err != nil {
		return fmt.Errorf("failed to record control result: %w", err)
	}
	return nil
}

// CreateEvidence creates compliance evidence.
func (s *Service) CreateEvidence(ctx context.Context, evidence Evidence) (*Evidence, error) {
	evidence.ID = uuid.New()
	evidence.CollectedAt = time.Now()
	evidence.ValidFrom = time.Now()
	evidence.IsCurrent = true
	evidence.CreatedAt = time.Now()
	evidence.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO compliance_evidence (
			id, org_id, control_id, evidence_type, title, description,
			storage_type, storage_path, content_hash, file_size_bytes, mime_type,
			collected_at, collected_by, collection_method, valid_from, valid_until,
			is_current, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`, evidence.ID, evidence.OrgID, evidence.ControlID, evidence.EvidenceType,
		evidence.Title, evidence.Description, evidence.StorageType, evidence.StoragePath,
		evidence.ContentHash, evidence.FileSizeBytes, evidence.MimeType, evidence.CollectedAt,
		evidence.CollectedBy, evidence.CollectionMethod, evidence.ValidFrom, evidence.ValidUntil,
		evidence.IsCurrent, evidence.CreatedAt, evidence.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create evidence: %w", err)
	}

	return &evidence, nil
}

// ListEvidence returns evidence for a control.
func (s *Service) ListEvidence(ctx context.Context, orgID, controlID uuid.UUID) ([]Evidence, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, org_id, control_id, evidence_type, title, description,
		       storage_type, storage_path, content_hash, file_size_bytes, mime_type,
		       collected_at, collected_by, collection_method, valid_from, valid_until,
		       is_current, reviewed_by, reviewed_at, review_status, review_notes,
		       created_at, updated_at
		FROM compliance_evidence
		WHERE org_id = $1 AND control_id = $2
		ORDER BY collected_at DESC
	`, orgID, controlID)
	if err != nil {
		return nil, fmt.Errorf("failed to list evidence: %w", err)
	}
	defer rows.Close()

	var evidences []Evidence
	for rows.Next() {
		var e Evidence
		if err := rows.Scan(
			&e.ID, &e.OrgID, &e.ControlID, &e.EvidenceType, &e.Title, &e.Description,
			&e.StorageType, &e.StoragePath, &e.ContentHash, &e.FileSizeBytes, &e.MimeType,
			&e.CollectedAt, &e.CollectedBy, &e.CollectionMethod, &e.ValidFrom, &e.ValidUntil,
			&e.IsCurrent, &e.ReviewedBy, &e.ReviewedAt, &e.ReviewStatus, &e.ReviewNotes,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan evidence: %w", err)
		}
		evidences = append(evidences, e)
	}

	return evidences, rows.Err()
}

// CreateExemption creates a control exemption.
func (s *Service) CreateExemption(ctx context.Context, exemption Exemption) (*Exemption, error) {
	exemption.ID = uuid.New()
	exemption.ApprovedAt = time.Now()
	exemption.Status = "active"
	exemption.CreatedAt = time.Now()
	exemption.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO compliance_exemptions (
			id, org_id, control_id, asset_id, site_id, reason, risk_acceptance,
			compensating_controls, approved_by, approved_at, expires_at,
			review_frequency_days, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, exemption.ID, exemption.OrgID, exemption.ControlID, exemption.AssetID, exemption.SiteID,
		exemption.Reason, exemption.RiskAcceptance, exemption.CompensatingControls,
		exemption.ApprovedBy, exemption.ApprovedAt, exemption.ExpiresAt,
		exemption.ReviewFrequencyDays, exemption.Status, exemption.CreatedAt, exemption.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create exemption: %w", err)
	}

	return &exemption, nil
}

// GetActiveExemptions returns active exemptions for an organization.
func (s *Service) GetActiveExemptions(ctx context.Context, orgID uuid.UUID) ([]Exemption, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, org_id, control_id, asset_id, site_id, reason, risk_acceptance,
		       compensating_controls, approved_by, approved_at, expires_at,
		       last_reviewed_at, last_reviewed_by, review_frequency_days, status,
		       created_at, updated_at
		FROM compliance_exemptions
		WHERE org_id = $1 AND status = 'active' AND expires_at > NOW()
		ORDER BY expires_at ASC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get exemptions: %w", err)
	}
	defer rows.Close()

	var exemptions []Exemption
	for rows.Next() {
		var e Exemption
		if err := rows.Scan(
			&e.ID, &e.OrgID, &e.ControlID, &e.AssetID, &e.SiteID, &e.Reason, &e.RiskAcceptance,
			&e.CompensatingControls, &e.ApprovedBy, &e.ApprovedAt, &e.ExpiresAt,
			&e.LastReviewedAt, &e.LastReviewedBy, &e.ReviewFrequencyDays, &e.Status,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan exemption: %w", err)
		}
		exemptions = append(exemptions, e)
	}

	return exemptions, rows.Err()
}

// GetComplianceScore calculates the overall compliance score for an organization.
func (s *Service) GetComplianceScore(ctx context.Context, orgID uuid.UUID, frameworkID *uuid.UUID) (*ComplianceScore, error) {
	query := `
		SELECT
			COUNT(DISTINCT ca.id) as assessment_count,
			COALESCE(AVG(ca.score), 0) as avg_score,
			COALESCE(SUM(ca.passed_controls), 0) as total_passed,
			COALESCE(SUM(ca.failed_controls), 0) as total_failed,
			COALESCE(SUM(ca.not_applicable), 0) as total_na
		FROM compliance_assessments ca
		WHERE ca.org_id = $1
		AND ca.status = 'completed'
		AND ca.completed_at > NOW() - INTERVAL '90 days'
	`
	args := []interface{}{orgID}

	if frameworkID != nil {
		query += " AND ca.framework_id = $2"
		args = append(args, *frameworkID)
	}

	var score ComplianceScore
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&score.AssessmentCount, &score.AverageScore,
		&score.TotalPassed, &score.TotalFailed, &score.TotalNA,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get compliance score: %w", err)
	}

	score.OrgID = orgID
	score.FrameworkID = frameworkID
	if score.TotalPassed+score.TotalFailed > 0 {
		score.PassRate = float64(score.TotalPassed) / float64(score.TotalPassed+score.TotalFailed) * 100
	}

	return &score, nil
}

// ComplianceScore represents the overall compliance score.
type ComplianceScore struct {
	OrgID           uuid.UUID  `json:"org_id"`
	FrameworkID     *uuid.UUID `json:"framework_id,omitempty"`
	AssessmentCount int        `json:"assessment_count"`
	AverageScore    float64    `json:"average_score"`
	TotalPassed     int        `json:"total_passed"`
	TotalFailed     int        `json:"total_failed"`
	TotalNA         int        `json:"total_not_applicable"`
	PassRate        float64    `json:"pass_rate"`
}
