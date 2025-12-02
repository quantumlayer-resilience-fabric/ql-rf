package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Site represents a logical site/datacenter/location.
type Site struct {
	ID                 uuid.UUID       `json:"id" db:"id"`
	OrgID              uuid.UUID       `json:"orgId" db:"org_id"`
	Name               string          `json:"name" db:"name"`
	Region             string          `json:"region" db:"region"`
	Platform           Platform        `json:"platform" db:"platform"`
	Environment        string          `json:"environment" db:"environment"`
	DRPairedSiteID     *uuid.UUID      `json:"drPairedSiteId,omitempty" db:"dr_paired_site_id"`
	LastSyncAt         *time.Time      `json:"lastSyncAt,omitempty" db:"last_sync_at"`
	Metadata           json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	CreatedAt          time.Time       `json:"createdAt" db:"created_at"`
	UpdatedAt          time.Time       `json:"updatedAt" db:"updated_at"`

	// Computed fields (not stored in DB)
	AssetCount         int     `json:"assetCount"`
	CompliantCount     int     `json:"compliantCount"`
	DriftedCount       int     `json:"driftedCount"`
	CoveragePercentage float64 `json:"coveragePercentage"`
	Status             string  `json:"status"` // healthy, warning, critical
	DRPaired           bool    `json:"drPaired"`
}

// SiteStatus represents the health status of a site.
type SiteStatus string

const (
	SiteStatusHealthy  SiteStatus = "healthy"
	SiteStatusWarning  SiteStatus = "warning"
	SiteStatusCritical SiteStatus = "critical"
)

// ComputeStatus calculates the site status based on coverage percentage.
func (s *Site) ComputeStatus() {
	if s.AssetCount == 0 {
		s.Status = string(SiteStatusHealthy)
		s.CoveragePercentage = 100
		return
	}

	s.CoveragePercentage = float64(s.CompliantCount) / float64(s.AssetCount) * 100
	s.DriftedCount = s.AssetCount - s.CompliantCount

	switch {
	case s.CoveragePercentage >= 90:
		s.Status = string(SiteStatusHealthy)
	case s.CoveragePercentage >= 70:
		s.Status = string(SiteStatusWarning)
	default:
		s.Status = string(SiteStatusCritical)
	}
}

// GetMetadata returns the metadata as a map.
func (s *Site) GetMetadata() map[string]string {
	if s.Metadata == nil {
		return nil
	}
	var meta map[string]string
	if err := json.Unmarshal(s.Metadata, &meta); err != nil {
		return nil
	}
	return meta
}

// SiteFilter represents filters for listing sites.
type SiteFilter struct {
	Platform    Platform `json:"platform,omitempty"`
	Region      string   `json:"region,omitempty"`
	Environment string   `json:"environment,omitempty"`
	Status      string   `json:"status,omitempty"`
}

// SiteListResponse represents a paginated list of sites.
type SiteListResponse struct {
	Sites      []Site `json:"sites"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"pageSize"`
	TotalPages int    `json:"totalPages"`
}

// CreateSiteRequest represents a request to create a site.
type CreateSiteRequest struct {
	Name        string   `json:"name" validate:"required,min=1,max=255"`
	Region      string   `json:"region" validate:"required"`
	Platform    Platform `json:"platform" validate:"required"`
	Environment string   `json:"environment" validate:"required"`
}
