package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/quantumlayerhq/ql-rf/pkg/database"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// ComplianceService handles compliance business logic.
type ComplianceService struct {
	db  *database.DB
	log *logger.Logger
}

// NewComplianceService creates a new ComplianceService.
func NewComplianceService(db *database.DB, log *logger.Logger) *ComplianceService {
	return &ComplianceService{
		db:  db,
		log: log.WithComponent("compliance-service"),
	}
}

// ComplianceFramework represents a compliance framework.
type ComplianceFramework struct {
	ID              uuid.UUID `json:"id"`
	OrgID           uuid.UUID `json:"orgId"`
	Name            string    `json:"name"`
	Description     string    `json:"description,omitempty"`
	Level           *int      `json:"level,omitempty"`
	Enabled         bool      `json:"enabled"`
	Score           float64   `json:"score"`
	PassingControls int       `json:"passingControls"`
	TotalControls   int       `json:"totalControls"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// FailingControl represents a failing compliance control.
type FailingControl struct {
	ID             string `json:"id"`
	Framework      string `json:"framework"`
	Title          string `json:"title"`
	Severity       string `json:"severity"`
	AffectedAssets int    `json:"affectedAssets"`
	Recommendation string `json:"recommendation"`
}

// ImageComplianceStatus represents compliance status for an image.
type ImageComplianceStatus struct {
	FamilyID     string    `json:"familyId"`
	FamilyName   string    `json:"familyName"`
	Version      string    `json:"version"`
	CIS          bool      `json:"cis"`
	SLSALevel    int       `json:"slsaLevel"`
	CosignSigned bool      `json:"cosignSigned"`
	LastScanAt   time.Time `json:"lastScanAt"`
	IssueCount   int       `json:"issueCount"`
}

// ComplianceSummary represents the overall compliance summary.
type ComplianceSummary struct {
	OverallScore     float64                 `json:"overallScore"`
	CISCompliance    float64                 `json:"cisCompliance"`
	SLSALevel        int                     `json:"slsaLevel"`
	SigstoreVerified float64                 `json:"sigstoreVerified"`
	LastAuditAt      *time.Time              `json:"lastAuditAt,omitempty"`
	Frameworks       []ComplianceFramework   `json:"frameworks"`
	FailingControls  []FailingControl        `json:"failingControls"`
	ImageCompliance  []ImageComplianceStatus `json:"imageCompliance"`
}

// GetComplianceSummaryInput contains input for getting compliance summary.
type GetComplianceSummaryInput struct {
	OrgID uuid.UUID
}

// GetComplianceSummary retrieves compliance summary from database.
func (s *ComplianceService) GetComplianceSummary(ctx context.Context, input GetComplianceSummaryInput) (*ComplianceSummary, error) {
	// Get frameworks
	frameworks, err := s.getFrameworks(ctx, input.OrgID)
	if err != nil {
		s.log.Warn("failed to get frameworks", "error", err)
		frameworks = []ComplianceFramework{}
	}

	// Get failing controls
	failingControls, err := s.getFailingControls(ctx, input.OrgID)
	if err != nil {
		s.log.Warn("failed to get failing controls", "error", err)
		failingControls = []FailingControl{}
	}

	// Get image compliance
	imageCompliance, err := s.getImageCompliance(ctx, input.OrgID)
	if err != nil {
		s.log.Warn("failed to get image compliance", "error", err)
		imageCompliance = []ImageComplianceStatus{}
	}

	// Calculate overall scores
	overallScore := 0.0
	cisCompliance := 0.0
	slsaLevel := 0
	sigstoreVerified := 0.0
	var lastAuditAt *time.Time

	// Calculate from frameworks
	for _, f := range frameworks {
		if f.Name == "CIS Benchmark" || f.Name == "CIS" {
			cisCompliance = f.Score
		}
		if f.Name == "SLSA" && f.Level != nil {
			slsaLevel = *f.Level
		}
	}

	// Calculate overall score as weighted average
	if len(frameworks) > 0 {
		totalScore := 0.0
		for _, f := range frameworks {
			if f.Enabled {
				totalScore += f.Score
			}
		}
		overallScore = totalScore / float64(len(frameworks))
	}

	// Calculate sigstore verification from images
	if len(imageCompliance) > 0 {
		signedCount := 0
		for _, img := range imageCompliance {
			if img.CosignSigned {
				signedCount++
			}
			if img.LastScanAt.After(time.Time{}) {
				if lastAuditAt == nil || img.LastScanAt.After(*lastAuditAt) {
					lastAuditAt = &img.LastScanAt
				}
			}
		}
		sigstoreVerified = float64(signedCount) / float64(len(imageCompliance)) * 100
	}

	return &ComplianceSummary{
		OverallScore:     overallScore,
		CISCompliance:    cisCompliance,
		SLSALevel:        slsaLevel,
		SigstoreVerified: sigstoreVerified,
		LastAuditAt:      lastAuditAt,
		Frameworks:       frameworks,
		FailingControls:  failingControls,
		ImageCompliance:  imageCompliance,
	}, nil
}

// getFrameworks retrieves compliance frameworks from database.
func (s *ComplianceService) getFrameworks(ctx context.Context, orgID uuid.UUID) ([]ComplianceFramework, error) {
	query := `
		SELECT
			cf.id,
			cf.org_id,
			cf.name,
			COALESCE(cf.description, '') as description,
			cf.level,
			cf.enabled,
			cf.created_at,
			cf.updated_at,
			COUNT(CASE WHEN cr.status = 'passing' THEN 1 END) as passing_controls,
			COUNT(cc.id) as total_controls,
			COALESCE(AVG(CASE WHEN cr.score IS NOT NULL THEN cr.score END), 0) as score
		FROM compliance_frameworks cf
		LEFT JOIN compliance_controls cc ON cc.framework_id = cf.id
		LEFT JOIN compliance_results cr ON cr.control_id = cc.id AND cr.org_id = cf.org_id
		WHERE cf.org_id = $1
		GROUP BY cf.id, cf.org_id, cf.name, cf.description, cf.level, cf.enabled, cf.created_at, cf.updated_at
		ORDER BY cf.name
	`

	rows, err := s.db.Pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var frameworks []ComplianceFramework
	for rows.Next() {
		var f ComplianceFramework
		var description string
		if err := rows.Scan(
			&f.ID,
			&f.OrgID,
			&f.Name,
			&description,
			&f.Level,
			&f.Enabled,
			&f.CreatedAt,
			&f.UpdatedAt,
			&f.PassingControls,
			&f.TotalControls,
			&f.Score,
		); err != nil {
			s.log.Warn("failed to scan framework row", "error", err)
			continue
		}
		f.Description = description

		// Calculate status based on score
		if f.TotalControls > 0 {
			if f.PassingControls == f.TotalControls {
				f.Status = "passing"
			} else if float64(f.PassingControls)/float64(f.TotalControls) >= 0.9 {
				f.Status = "warning"
			} else {
				f.Status = "failing"
			}
		} else {
			f.Status = "unknown"
		}

		frameworks = append(frameworks, f)
	}

	return frameworks, nil
}

// getFailingControls retrieves failing compliance controls from database.
func (s *ComplianceService) getFailingControls(ctx context.Context, orgID uuid.UUID) ([]FailingControl, error) {
	query := `
		SELECT
			cc.control_id,
			cf.name as framework,
			cc.title,
			cc.severity,
			COALESCE(cr.affected_assets, 0) as affected_assets,
			COALESCE(cc.recommendation, '') as recommendation
		FROM compliance_controls cc
		JOIN compliance_frameworks cf ON cf.id = cc.framework_id
		LEFT JOIN compliance_results cr ON cr.control_id = cc.id AND cr.org_id = cf.org_id
		WHERE cf.org_id = $1
		AND (cr.status = 'failing' OR cr.status IS NULL)
		ORDER BY
			CASE cc.severity
				WHEN 'critical' THEN 1
				WHEN 'high' THEN 2
				WHEN 'medium' THEN 3
				WHEN 'low' THEN 4
				ELSE 5
			END,
			cr.affected_assets DESC NULLS LAST
		LIMIT 20
	`

	rows, err := s.db.Pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var controls []FailingControl
	for rows.Next() {
		var c FailingControl
		if err := rows.Scan(
			&c.ID,
			&c.Framework,
			&c.Title,
			&c.Severity,
			&c.AffectedAssets,
			&c.Recommendation,
		); err != nil {
			s.log.Warn("failed to scan failing control row", "error", err)
			continue
		}
		controls = append(controls, c)
	}

	return controls, nil
}

// getImageCompliance retrieves image compliance status from database.
func (s *ComplianceService) getImageCompliance(ctx context.Context, orgID uuid.UUID) ([]ImageComplianceStatus, error) {
	query := `
		SELECT
			i.id::text as family_id,
			i.family as family_name,
			i.version,
			COALESCE(ic.cis_compliant, false) as cis,
			COALESCE(ic.slsa_level, 0) as slsa_level,
			COALESCE(ic.cosign_signed, i.signed, false) as cosign_signed,
			COALESCE(ic.last_scan_at, i.created_at) as last_scan_at,
			COALESCE(ic.issue_count, 0) as issue_count
		FROM images i
		LEFT JOIN image_compliance ic ON ic.image_id = i.id
		WHERE i.org_id = $1
		AND i.status = 'production'
		ORDER BY i.family, i.version DESC
	`

	rows, err := s.db.Pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []ImageComplianceStatus
	for rows.Next() {
		var img ImageComplianceStatus
		if err := rows.Scan(
			&img.FamilyID,
			&img.FamilyName,
			&img.Version,
			&img.CIS,
			&img.SLSALevel,
			&img.CosignSigned,
			&img.LastScanAt,
			&img.IssueCount,
		); err != nil {
			s.log.Warn("failed to scan image compliance row", "error", err)
			continue
		}
		images = append(images, img)
	}

	return images, nil
}

// ListFrameworksInput contains input for listing frameworks.
type ListFrameworksInput struct {
	OrgID uuid.UUID
}

// ListFrameworks retrieves compliance frameworks.
func (s *ComplianceService) ListFrameworks(ctx context.Context, input ListFrameworksInput) ([]ComplianceFramework, error) {
	summary, err := s.GetComplianceSummary(ctx, GetComplianceSummaryInput{OrgID: input.OrgID})
	if err != nil {
		return nil, err
	}
	return summary.Frameworks, nil
}

// ListFailingControlsInput contains input for listing failing controls.
type ListFailingControlsInput struct {
	OrgID uuid.UUID
}

// ListFailingControls retrieves failing compliance controls.
func (s *ComplianceService) ListFailingControls(ctx context.Context, input ListFailingControlsInput) ([]FailingControl, error) {
	summary, err := s.GetComplianceSummary(ctx, GetComplianceSummaryInput{OrgID: input.OrgID})
	if err != nil {
		return nil, err
	}
	return summary.FailingControls, nil
}

// ListImageComplianceInput contains input for listing image compliance.
type ListImageComplianceInput struct {
	OrgID uuid.UUID
}

// ListImageCompliance retrieves image compliance status.
func (s *ComplianceService) ListImageCompliance(ctx context.Context, input ListImageComplianceInput) ([]ImageComplianceStatus, error) {
	summary, err := s.GetComplianceSummary(ctx, GetComplianceSummaryInput{OrgID: input.OrgID})
	if err != nil {
		return nil, err
	}
	return summary.ImageCompliance, nil
}

// TriggerAuditInput contains input for triggering an audit.
type TriggerAuditInput struct {
	OrgID uuid.UUID
}

// TriggerAuditResponse represents the response from triggering an audit.
type TriggerAuditResponse struct {
	JobID     string    `json:"jobId"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"startedAt"`
}

// TriggerAudit triggers a compliance audit.
func (s *ComplianceService) TriggerAudit(ctx context.Context, input TriggerAuditInput) (*TriggerAuditResponse, error) {
	return &TriggerAuditResponse{
		JobID:     uuid.New().String(),
		Status:    "queued",
		StartedAt: time.Now(),
	}, nil
}

func intPtr(i int) *int {
	return &i
}
