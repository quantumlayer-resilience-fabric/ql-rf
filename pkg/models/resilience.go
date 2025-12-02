package models

import (
	"time"

	"github.com/google/uuid"
)

// ResilienceSite represents a site in a DR context.
type ResilienceSite struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Region         string    `json:"region"`
	Platform       Platform  `json:"platform"`
	AssetCount     int       `json:"assetCount"`
	Status         string    `json:"status"` // healthy, warning, critical, syncing
	LastSyncAt     time.Time `json:"lastSyncAt"`
	RPO            string    `json:"rpo,omitempty"` // Recovery Point Objective
	RTO            string    `json:"rto,omitempty"` // Recovery Time Objective
	ReplicationLag *string   `json:"replicationLag,omitempty"`
}

// DRPair represents a disaster recovery site pair.
type DRPair struct {
	ID                uuid.UUID      `json:"id" db:"id"`
	OrgID             uuid.UUID      `json:"orgId" db:"org_id"`
	Name              string         `json:"name" db:"name"`
	PrimarySiteID     uuid.UUID      `json:"primarySiteId" db:"primary_site_id"`
	DRSiteID          uuid.UUID      `json:"drSiteId" db:"dr_site_id"`
	Status            string         `json:"status" db:"status"`                     // healthy, warning, critical, syncing
	ReplicationStatus string         `json:"replicationStatus" db:"replication_status"` // in-sync, lagging, failed
	RPO               string         `json:"rpo,omitempty" db:"rpo"`
	RTO               string         `json:"rto,omitempty" db:"rto"`
	LastFailoverTest  *time.Time     `json:"lastFailoverTest,omitempty" db:"last_failover_test"`
	LastSyncAt        *time.Time     `json:"lastSyncAt,omitempty" db:"last_sync_at"`
	CreatedAt         time.Time      `json:"createdAt" db:"created_at"`
	UpdatedAt         time.Time      `json:"updatedAt" db:"updated_at"`

	// Loaded relationships
	PrimarySite ResilienceSite `json:"primarySite"`
	DRSite      ResilienceSite `json:"drSite"`
}

// DRPairStatus represents DR pair status values.
type DRPairStatus string

const (
	DRPairStatusHealthy  DRPairStatus = "healthy"
	DRPairStatusWarning  DRPairStatus = "warning"
	DRPairStatusCritical DRPairStatus = "critical"
	DRPairStatusSyncing  DRPairStatus = "syncing"
	DRPairStatusUnknown  DRPairStatus = "unknown"
)

// ReplicationStatus represents replication status values.
type ReplicationStatus string

const (
	ReplicationStatusInSync  ReplicationStatus = "in-sync"
	ReplicationStatusLagging ReplicationStatus = "lagging"
	ReplicationStatusFailed  ReplicationStatus = "failed"
	ReplicationStatusUnknown ReplicationStatus = "unknown"
)

// ResilienceSummary represents the overall resilience/DR summary.
type ResilienceSummary struct {
	DRReadiness      float64          `json:"drReadiness"`      // Percentage of sites with DR pairs
	RPOCompliance    float64          `json:"rpoCompliance"`    // Percentage meeting RPO
	RTOCompliance    float64          `json:"rtoCompliance"`    // Percentage meeting RTO
	LastFailoverTest *time.Time       `json:"lastFailoverTest,omitempty"`
	TotalPairs       int              `json:"totalPairs"`
	HealthyPairs     int              `json:"healthyPairs"`
	DRPairs          []DRPair         `json:"drPairs"`
	UnpairedSites    []ResilienceSite `json:"unpairedSites"`
}

// CreateDRPairRequest represents a request to create a DR pair.
type CreateDRPairRequest struct {
	Name          string    `json:"name" validate:"required,min=1,max=255"`
	PrimarySiteID uuid.UUID `json:"primarySiteId" validate:"required"`
	DRSiteID      uuid.UUID `json:"drSiteId" validate:"required"`
	RPO           string    `json:"rpo,omitempty"`
	RTO           string    `json:"rto,omitempty"`
}

// TriggerFailoverTestResponse represents response from triggering a failover test.
type TriggerFailoverTestResponse struct {
	JobID     string    `json:"jobId"`
	Status    string    `json:"status"` // queued, running
	StartedAt time.Time `json:"startedAt"`
}

// TriggerSyncResponse represents response from triggering a sync.
type TriggerSyncResponse struct {
	JobID     string    `json:"jobId"`
	Status    string    `json:"status"` // queued, running
	StartedAt time.Time `json:"startedAt"`
}
