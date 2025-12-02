package models

import (
	"time"

	"github.com/google/uuid"
)

// DriftReport represents a drift analysis report for a scope.
type DriftReport struct {
	ID              uuid.UUID `json:"id" db:"id"`
	OrgID           uuid.UUID `json:"org_id" db:"org_id"`
	EnvID           uuid.UUID `json:"env_id,omitempty" db:"env_id"`
	Platform        Platform  `json:"platform,omitempty" db:"platform"`
	Site            string    `json:"site,omitempty" db:"site"`
	TotalAssets     int       `json:"total_assets" db:"total_assets"`
	CompliantAssets int       `json:"compliant_assets" db:"compliant_assets"`
	CoveragePct     float64   `json:"coverage_pct" db:"coverage_pct"`
	Status          DriftStatus `json:"status" db:"status"`
	CalculatedAt    time.Time `json:"calculated_at" db:"calculated_at"`
}

// DriftStatus represents the health status based on drift.
type DriftStatus string

const (
	DriftStatusHealthy  DriftStatus = "healthy"  // >= 90%
	DriftStatusWarning  DriftStatus = "warning"  // 70-90%
	DriftStatusCritical DriftStatus = "critical" // < 70%
)

// CalculateStatus determines the status based on coverage percentage.
func CalculateStatus(coveragePct float64, warningThreshold, criticalThreshold float64) DriftStatus {
	if coveragePct >= warningThreshold {
		return DriftStatusHealthy
	}
	if coveragePct >= criticalThreshold {
		return DriftStatusWarning
	}
	return DriftStatusCritical
}

// OutdatedAsset represents an asset that is behind the golden image baseline.
type OutdatedAsset struct {
	Asset           Asset   `json:"asset"`
	CurrentVersion  string  `json:"current_version"`
	ExpectedVersion string  `json:"expected_version"`
	DriftAge        int     `json:"drift_age_days"` // Days behind baseline
	Severity        DriftStatus `json:"severity"`
}

// DriftSummary provides an aggregated view of drift across the organization.
type DriftSummary struct {
	OrgID           uuid.UUID        `json:"org_id"`
	TotalAssets     int              `json:"total_assets"`
	CompliantAssets int              `json:"compliant_assets"`
	CoveragePct     float64          `json:"coverage_pct"`
	Status          DriftStatus      `json:"status"`
	ByEnvironment   []DriftByScope   `json:"by_environment,omitempty"`
	ByPlatform      []DriftByScope   `json:"by_platform,omitempty"`
	BySite          []DriftByScope   `json:"by_site,omitempty"`
	TopOffenders    []OutdatedAsset  `json:"top_offenders,omitempty"`
	CalculatedAt    time.Time        `json:"calculated_at"`
}

// DriftByScope represents drift metrics for a specific scope (env/platform/site).
type DriftByScope struct {
	Scope           string      `json:"scope"` // Environment name, platform, or site
	TotalAssets     int         `json:"total_assets"`
	CompliantAssets int         `json:"compliant_assets"`
	CoveragePct     float64     `json:"coverage_pct"`
	Status          DriftStatus `json:"status"`
}

// DriftTrend represents historical drift data for trending.
type DriftTrend struct {
	Date        time.Time   `json:"date"`
	CoveragePct float64     `json:"coverage_pct"`
	Status      DriftStatus `json:"status"`
}

// DriftFilter represents filters for querying drift reports.
type DriftFilter struct {
	EnvID     uuid.UUID   `json:"env_id,omitempty"`
	Platform  Platform    `json:"platform,omitempty"`
	Site      string      `json:"site,omitempty"`
	Status    DriftStatus `json:"status,omitempty"`
	StartDate time.Time   `json:"start_date,omitempty"`
	EndDate   time.Time   `json:"end_date,omitempty"`
}

// DriftDetectedEvent is published when drift exceeds thresholds.
type DriftDetectedEvent struct {
	Report       DriftReport     `json:"report"`
	TopOffenders []OutdatedAsset `json:"top_offenders"`
	Timestamp    time.Time       `json:"timestamp"`
}

// DriftCalculationRequest represents a request to calculate drift.
type DriftCalculationRequest struct {
	OrgID    uuid.UUID `json:"org_id"`
	EnvID    uuid.UUID `json:"env_id,omitempty"`
	Platform Platform  `json:"platform,omitempty"`
	Site     string    `json:"site,omitempty"`
}

// DriftConfig holds thresholds and settings for drift calculation.
type DriftConfig struct {
	WarningThreshold  float64 `json:"warning_threshold"`  // Default: 90%
	CriticalThreshold float64 `json:"critical_threshold"` // Default: 70%
	MaxOffenders      int     `json:"max_offenders"`      // Number of top offenders to include
}

// DefaultDriftConfig returns default drift configuration.
func DefaultDriftConfig() DriftConfig {
	return DriftConfig{
		WarningThreshold:  90.0,
		CriticalThreshold: 70.0,
		MaxOffenders:      10,
	}
}
