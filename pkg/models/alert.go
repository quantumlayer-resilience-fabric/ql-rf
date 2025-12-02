package models

import (
	"time"

	"github.com/google/uuid"
)

// Alert represents a system alert.
type Alert struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	OrgID          uuid.UUID  `json:"orgId" db:"org_id"`
	Severity       string     `json:"severity" db:"severity"`     // critical, warning, info
	Title          string     `json:"title" db:"title"`
	Description    string     `json:"description" db:"description"`
	Source         string     `json:"source" db:"source"`         // drift, compliance, connector, system
	SiteID         *uuid.UUID `json:"siteId,omitempty" db:"site_id"`
	AssetID        *uuid.UUID `json:"assetId,omitempty" db:"asset_id"`
	ImageID        *uuid.UUID `json:"imageId,omitempty" db:"image_id"`
	Status         string     `json:"status" db:"status"`         // open, acknowledged, resolved
	CreatedAt      time.Time  `json:"createdAt" db:"created_at"`
	AcknowledgedAt *time.Time `json:"acknowledgedAt,omitempty" db:"acknowledged_at"`
	AcknowledgedBy *uuid.UUID `json:"acknowledgedBy,omitempty" db:"acknowledged_by"`
	ResolvedAt     *time.Time `json:"resolvedAt,omitempty" db:"resolved_at"`
	ResolvedBy     *uuid.UUID `json:"resolvedBy,omitempty" db:"resolved_by"`
}

// AlertSeverity represents alert severity levels.
type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityInfo     AlertSeverity = "info"
)

// AlertStatus represents alert statuses.
type AlertStatus string

const (
	AlertStatusOpen         AlertStatus = "open"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
)

// AlertSource represents alert sources.
type AlertSource string

const (
	AlertSourceDrift      AlertSource = "drift"
	AlertSourceCompliance AlertSource = "compliance"
	AlertSourceConnector  AlertSource = "connector"
	AlertSourceSystem     AlertSource = "system"
)

// AlertFilter represents filters for listing alerts.
type AlertFilter struct {
	Severity string     `json:"severity,omitempty"`
	Status   string     `json:"status,omitempty"`
	Source   string     `json:"source,omitempty"`
	SiteID   *uuid.UUID `json:"siteId,omitempty"`
}

// AlertListResponse represents a paginated list of alerts.
type AlertListResponse struct {
	Alerts     []Alert `json:"alerts"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"pageSize"`
	TotalPages int     `json:"totalPages"`
}

// AlertCount represents alert counts by severity.
type AlertCount struct {
	Severity string `json:"severity"`
	Count    int    `json:"count"`
}

// CreateAlertRequest represents a request to create an alert.
type CreateAlertRequest struct {
	Severity    string     `json:"severity" validate:"required,oneof=critical warning info"`
	Title       string     `json:"title" validate:"required,min=1,max=255"`
	Description string     `json:"description"`
	Source      string     `json:"source" validate:"required"`
	SiteID      *uuid.UUID `json:"siteId,omitempty"`
	AssetID     *uuid.UUID `json:"assetId,omitempty"`
	ImageID     *uuid.UUID `json:"imageId,omitempty"`
}
