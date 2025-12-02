package models

import (
	"time"

	"github.com/google/uuid"
)

// Activity represents a recent activity/event in the system.
type Activity struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	OrgID     uuid.UUID  `json:"orgId" db:"org_id"`
	Type      string     `json:"type" db:"type"`       // info, warning, success, critical
	Action    string     `json:"action" db:"action"`
	Detail    string     `json:"detail,omitempty" db:"detail"`
	UserID    *uuid.UUID `json:"userId,omitempty" db:"user_id"`
	SiteID    *uuid.UUID `json:"siteId,omitempty" db:"site_id"`
	AssetID   *uuid.UUID `json:"assetId,omitempty" db:"asset_id"`
	ImageID   *uuid.UUID `json:"imageId,omitempty" db:"image_id"`
	Timestamp time.Time  `json:"timestamp" db:"created_at"`
}

// ActivityType represents the type of activity.
type ActivityType string

const (
	ActivityTypeInfo     ActivityType = "info"
	ActivityTypeWarning  ActivityType = "warning"
	ActivityTypeSuccess  ActivityType = "success"
	ActivityTypeCritical ActivityType = "critical"
)

// CreateActivityRequest represents a request to log an activity.
type CreateActivityRequest struct {
	Type    string     `json:"type" validate:"required,oneof=info warning success critical"`
	Action  string     `json:"action" validate:"required,min=1,max=255"`
	Detail  string     `json:"detail,omitempty"`
	SiteID  *uuid.UUID `json:"siteId,omitempty"`
	AssetID *uuid.UUID `json:"assetId,omitempty"`
	ImageID *uuid.UUID `json:"imageId,omitempty"`
}
