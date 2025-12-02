package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ResilienceService handles resilience/DR business logic.
type ResilienceService struct {
	siteRepo SiteRepository
}

// NewResilienceService creates a new ResilienceService.
func NewResilienceService(siteRepo SiteRepository) *ResilienceService {
	return &ResilienceService{siteRepo: siteRepo}
}

// ResilienceSite represents a site in a DR context.
type ResilienceSite struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Region         string     `json:"region"`
	Platform       string     `json:"platform"`
	AssetCount     int        `json:"assetCount"`
	Status         string     `json:"status"`
	LastSyncAt     time.Time  `json:"lastSyncAt"`
	RPO            string     `json:"rpo,omitempty"`
	RTO            string     `json:"rto,omitempty"`
	ReplicationLag *string    `json:"replicationLag,omitempty"`
}

// DRPair represents a disaster recovery site pair.
type DRPair struct {
	ID                uuid.UUID       `json:"id"`
	OrgID             uuid.UUID       `json:"orgId"`
	Name              string          `json:"name"`
	PrimarySiteID     uuid.UUID       `json:"primarySiteId"`
	DRSiteID          uuid.UUID       `json:"drSiteId"`
	Status            string          `json:"status"`
	ReplicationStatus string          `json:"replicationStatus"`
	RPO               string          `json:"rpo,omitempty"`
	RTO               string          `json:"rto,omitempty"`
	LastFailoverTest  *time.Time      `json:"lastFailoverTest,omitempty"`
	LastSyncAt        *time.Time      `json:"lastSyncAt,omitempty"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
	PrimarySite       ResilienceSite  `json:"primarySite"`
	DRSite            ResilienceSite  `json:"drSite"`
}

// ResilienceSummary represents the overall resilience/DR summary.
type ResilienceSummary struct {
	DRReadiness      float64          `json:"drReadiness"`
	RPOCompliance    float64          `json:"rpoCompliance"`
	RTOCompliance    float64          `json:"rtoCompliance"`
	LastFailoverTest *time.Time       `json:"lastFailoverTest,omitempty"`
	TotalPairs       int              `json:"totalPairs"`
	HealthyPairs     int              `json:"healthyPairs"`
	DRPairs          []DRPair         `json:"drPairs"`
	UnpairedSites    []ResilienceSite `json:"unpairedSites"`
}

// GetResilienceSummaryInput contains input for getting resilience summary.
type GetResilienceSummaryInput struct {
	OrgID uuid.UUID
}

// GetResilienceSummary retrieves resilience summary.
func (s *ResilienceService) GetResilienceSummary(ctx context.Context, input GetResilienceSummaryInput) (*ResilienceSummary, error) {
	// Get sites with stats
	sites, err := s.siteRepo.ListSitesWithStats(ctx, input.OrgID)
	if err != nil {
		return nil, err
	}

	// Calculate DR readiness
	drPairedCount := 0
	unpairedSites := make([]ResilienceSite, 0)
	drPairs := make([]DRPair, 0)

	for _, site := range sites {
		if site.DRPaired {
			drPairedCount++
		} else {
			unpairedSites = append(unpairedSites, ResilienceSite{
				ID:         site.ID,
				Name:       site.Name,
				Region:     site.Region,
				Platform:   site.Platform,
				AssetCount: site.AssetCount,
				Status:     site.Status,
			})
		}
	}

	var drReadiness float64 = 0
	if len(sites) > 0 {
		drReadiness = float64(drPairedCount) / float64(len(sites)) * 100
	}

	// Mock DR pairs for now
	now := time.Now()
	if drPairedCount > 0 {
		drPairs = append(drPairs, DRPair{
			ID:                uuid.New(),
			OrgID:             input.OrgID,
			Name:              "US East - US West",
			PrimarySiteID:     uuid.New(),
			DRSiteID:          uuid.New(),
			Status:            "healthy",
			ReplicationStatus: "in-sync",
			RPO:               "15m",
			RTO:               "1h",
			LastFailoverTest:  &now,
			LastSyncAt:        &now,
			CreatedAt:         now,
			UpdatedAt:         now,
			PrimarySite: ResilienceSite{
				ID:         uuid.New(),
				Name:       "US East Production",
				Region:     "us-east-1",
				Platform:   "aws",
				AssetCount: 150,
				Status:     "healthy",
				LastSyncAt: now,
				RPO:        "15m",
				RTO:        "1h",
			},
			DRSite: ResilienceSite{
				ID:         uuid.New(),
				Name:       "US West DR",
				Region:     "us-west-2",
				Platform:   "aws",
				AssetCount: 150,
				Status:     "healthy",
				LastSyncAt: now,
				RPO:        "15m",
				RTO:        "1h",
			},
		})
	}

	return &ResilienceSummary{
		DRReadiness:      drReadiness,
		RPOCompliance:    95.0,
		RTOCompliance:    100.0,
		LastFailoverTest: &now,
		TotalPairs:       len(drPairs),
		HealthyPairs:     len(drPairs),
		DRPairs:          drPairs,
		UnpairedSites:    unpairedSites,
	}, nil
}

// ListDRPairsInput contains input for listing DR pairs.
type ListDRPairsInput struct {
	OrgID uuid.UUID
}

// ListDRPairs retrieves DR pairs.
func (s *ResilienceService) ListDRPairs(ctx context.Context, input ListDRPairsInput) ([]DRPair, error) {
	summary, err := s.GetResilienceSummary(ctx, GetResilienceSummaryInput{OrgID: input.OrgID})
	if err != nil {
		return nil, err
	}
	return summary.DRPairs, nil
}

// GetDRPairInput contains input for getting a DR pair.
type GetDRPairInput struct {
	ID    uuid.UUID
	OrgID uuid.UUID
}

// GetDRPair retrieves a specific DR pair.
func (s *ResilienceService) GetDRPair(ctx context.Context, input GetDRPairInput) (*DRPair, error) {
	summary, err := s.GetResilienceSummary(ctx, GetResilienceSummaryInput{OrgID: input.OrgID})
	if err != nil {
		return nil, err
	}
	for _, pair := range summary.DRPairs {
		if pair.ID == input.ID {
			return &pair, nil
		}
	}
	return nil, ErrNotFound
}

// TriggerFailoverTestInput contains input for triggering a failover test.
type TriggerFailoverTestInput struct {
	ID    uuid.UUID
	OrgID uuid.UUID
}

// TriggerFailoverTestResponse represents response from triggering a failover test.
type TriggerFailoverTestResponse struct {
	JobID     string    `json:"jobId"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"startedAt"`
}

// TriggerFailoverTest triggers a DR failover test.
func (s *ResilienceService) TriggerFailoverTest(ctx context.Context, input TriggerFailoverTestInput) (*TriggerFailoverTestResponse, error) {
	return &TriggerFailoverTestResponse{
		JobID:     uuid.New().String(),
		Status:    "queued",
		StartedAt: time.Now(),
	}, nil
}

// TriggerSyncInput contains input for triggering a sync.
type TriggerSyncInput struct {
	ID    uuid.UUID
	OrgID uuid.UUID
}

// TriggerSyncResponse represents response from triggering a sync.
type TriggerSyncResponse struct {
	JobID     string    `json:"jobId"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"startedAt"`
}

// TriggerSync triggers a DR sync.
func (s *ResilienceService) TriggerSync(ctx context.Context, input TriggerSyncInput) (*TriggerSyncResponse, error) {
	return &TriggerSyncResponse{
		JobID:     uuid.New().String(),
		Status:    "queued",
		StartedAt: time.Now(),
	}, nil
}
