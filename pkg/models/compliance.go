package models

import (
	"time"

	"github.com/google/uuid"
)

// ComplianceFramework represents a compliance framework (CIS, SLSA, SOC2, etc.)
type ComplianceFramework struct {
	ID              uuid.UUID `json:"id" db:"id"`
	OrgID           uuid.UUID `json:"orgId" db:"org_id"`
	Name            string    `json:"name" db:"name"`
	Description     string    `json:"description,omitempty" db:"description"`
	Level           *int      `json:"level,omitempty" db:"level"` // For SLSA levels
	Enabled         bool      `json:"enabled" db:"enabled"`
	Score           float64   `json:"score"`                      // Computed
	PassingControls int       `json:"passingControls"`            // Computed
	TotalControls   int       `json:"totalControls"`              // Computed
	Status          string    `json:"status"`                     // passing, warning, failing - Computed
	CreatedAt       time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time `json:"updatedAt" db:"updated_at"`
}

// ComplianceControl represents a control within a framework.
type ComplianceControl struct {
	ID             uuid.UUID `json:"id" db:"id"`
	FrameworkID    uuid.UUID `json:"frameworkId" db:"framework_id"`
	ControlID      string    `json:"controlId" db:"control_id"` // e.g., "CIS-4.2.1"
	Title          string    `json:"title" db:"title"`
	Description    string    `json:"description,omitempty" db:"description"`
	Severity       string    `json:"severity" db:"severity"` // high, medium, low
	Recommendation string    `json:"recommendation,omitempty" db:"recommendation"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
}

// FailingControl represents a control that's not passing.
type FailingControl struct {
	ID             string `json:"id"`             // Control ID (e.g., "CIS-4.2.1")
	Framework      string `json:"framework"`      // Framework name
	Title          string `json:"title"`
	Severity       string `json:"severity"`       // high, medium, low
	AffectedAssets int    `json:"affectedAssets"`
	Recommendation string `json:"recommendation"`
}

// ComplianceResult represents the result of a compliance check.
type ComplianceResult struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	OrgID          uuid.UUID  `json:"orgId" db:"org_id"`
	FrameworkID    uuid.UUID  `json:"frameworkId" db:"framework_id"`
	ControlID      *uuid.UUID `json:"controlId,omitempty" db:"control_id"`
	Status         string     `json:"status" db:"status"` // passing, failing, warning
	AffectedAssets int        `json:"affectedAssets" db:"affected_assets"`
	Score          float64    `json:"score" db:"score"`
	LastAuditAt    time.Time  `json:"lastAuditAt" db:"last_audit_at"`
}

// ImageComplianceStatus represents compliance status for an image.
type ImageComplianceStatus struct {
	FamilyID    string    `json:"familyId"`
	FamilyName  string    `json:"familyName"`
	Version     string    `json:"version"`
	CIS         bool      `json:"cis"`
	SLSALevel   int       `json:"slsaLevel"`
	CosignSigned bool     `json:"cosignSigned"`
	LastScanAt  time.Time `json:"lastScanAt"`
	IssueCount  int       `json:"issueCount"`
}

// ComplianceSummary represents the overall compliance summary.
type ComplianceSummary struct {
	OverallScore     float64               `json:"overallScore"`
	CISCompliance    float64               `json:"cisCompliance"`
	SLSALevel        int                   `json:"slsaLevel"`
	SigstoreVerified float64               `json:"sigstoreVerified"` // Percentage of signed images
	LastAuditAt      *time.Time            `json:"lastAuditAt,omitempty"`
	Frameworks       []ComplianceFramework `json:"frameworks"`
	FailingControls  []FailingControl      `json:"failingControls"`
	ImageCompliance  []ImageComplianceStatus `json:"imageCompliance"`
}

// ComplianceStatus represents status values.
type ComplianceStatus string

const (
	ComplianceStatusPassing ComplianceStatus = "passing"
	ComplianceStatusWarning ComplianceStatus = "warning"
	ComplianceStatusFailing ComplianceStatus = "failing"
)

// ControlSeverity represents severity levels for controls.
type ControlSeverity string

const (
	ControlSeverityHigh   ControlSeverity = "high"
	ControlSeverityMedium ControlSeverity = "medium"
	ControlSeverityLow    ControlSeverity = "low"
)

// TriggerAuditResponse represents response from triggering an audit.
type TriggerAuditResponse struct {
	JobID   string    `json:"jobId"`
	Status  string    `json:"status"` // queued, running
	StartedAt time.Time `json:"startedAt"`
}
