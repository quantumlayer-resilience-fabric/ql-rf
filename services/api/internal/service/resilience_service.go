package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ResilienceService handles resilience/DR business logic.
type ResilienceService struct {
	siteRepo   SiteRepository
	drPairRepo DRPairRepository
}

// NewResilienceService creates a new ResilienceService.
func NewResilienceService(siteRepo SiteRepository, drPairRepo DRPairRepository) *ResilienceService {
	return &ResilienceService{
		siteRepo:   siteRepo,
		drPairRepo: drPairRepo,
	}
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

	// Fetch DR pairs from database
	dbPairs, err := s.drPairRepo.ListDRPairs(ctx, input.OrgID)
	if err != nil {
		return nil, err
	}

	// Build site lookup map for enriching DR pairs
	siteMap := make(map[uuid.UUID]*SiteWithStats)
	for i := range sites {
		siteMap[sites[i].ID] = &sites[i]
	}

	// Convert DB pairs to response format with enriched site data
	var lastFailoverTest *time.Time
	healthyPairs := 0
	for _, dbPair := range dbPairs {
		pair := DRPair{
			ID:                dbPair.ID,
			OrgID:             dbPair.OrgID,
			Name:              dbPair.Name,
			PrimarySiteID:     dbPair.PrimarySiteID,
			DRSiteID:          dbPair.DRSiteID,
			Status:            dbPair.Status,
			ReplicationStatus: dbPair.ReplicationStatus,
			LastFailoverTest:  dbPair.LastFailoverTest,
			LastSyncAt:        dbPair.LastSyncAt,
			CreatedAt:         dbPair.CreatedAt,
			UpdatedAt:         dbPair.UpdatedAt,
		}

		// Handle nullable RPO/RTO
		if dbPair.RPO != nil {
			pair.RPO = *dbPair.RPO
		}
		if dbPair.RTO != nil {
			pair.RTO = *dbPair.RTO
		}

		// Enrich with primary site data
		if primarySite, ok := siteMap[dbPair.PrimarySiteID]; ok {
			pair.PrimarySite = ResilienceSite{
				ID:         primarySite.ID,
				Name:       primarySite.Name,
				Region:     primarySite.Region,
				Platform:   primarySite.Platform,
				AssetCount: primarySite.AssetCount,
				Status:     primarySite.Status,
				RPO:        pair.RPO,
				RTO:        pair.RTO,
			}
			if primarySite.LastSyncAt != nil {
				pair.PrimarySite.LastSyncAt = *primarySite.LastSyncAt
			}
		}

		// Enrich with DR site data
		if drSite, ok := siteMap[dbPair.DRSiteID]; ok {
			pair.DRSite = ResilienceSite{
				ID:         drSite.ID,
				Name:       drSite.Name,
				Region:     drSite.Region,
				Platform:   drSite.Platform,
				AssetCount: drSite.AssetCount,
				Status:     drSite.Status,
				RPO:        pair.RPO,
				RTO:        pair.RTO,
			}
			if drSite.LastSyncAt != nil {
				pair.DRSite.LastSyncAt = *drSite.LastSyncAt
			}
		}

		drPairs = append(drPairs, pair)

		// Track healthy pairs and most recent failover test
		if dbPair.Status == "healthy" {
			healthyPairs++
		}
		if dbPair.LastFailoverTest != nil {
			if lastFailoverTest == nil || dbPair.LastFailoverTest.After(*lastFailoverTest) {
				lastFailoverTest = dbPair.LastFailoverTest
			}
		}
	}

	// Calculate RPO/RTO compliance based on actual pair data
	rpoCompliance := 100.0
	rtoCompliance := 100.0
	if len(drPairs) > 0 {
		rpoCompliant := 0
		rtoCompliant := 0
		for _, pair := range drPairs {
			// Consider in-sync or syncing as RPO compliant
			if pair.ReplicationStatus == "in-sync" || pair.ReplicationStatus == "syncing" {
				rpoCompliant++
			}
			// Consider healthy status as RTO compliant
			if pair.Status == "healthy" {
				rtoCompliant++
			}
		}
		rpoCompliance = float64(rpoCompliant) / float64(len(drPairs)) * 100
		rtoCompliance = float64(rtoCompliant) / float64(len(drPairs)) * 100
	}

	return &ResilienceSummary{
		DRReadiness:      drReadiness,
		RPOCompliance:    rpoCompliance,
		RTOCompliance:    rtoCompliance,
		LastFailoverTest: lastFailoverTest,
		TotalPairs:       len(drPairs),
		HealthyPairs:     healthyPairs,
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
	dbPair, err := s.drPairRepo.GetDRPair(ctx, input.ID, input.OrgID)
	if err != nil {
		return nil, err
	}

	pair := &DRPair{
		ID:                dbPair.ID,
		OrgID:             dbPair.OrgID,
		Name:              dbPair.Name,
		PrimarySiteID:     dbPair.PrimarySiteID,
		DRSiteID:          dbPair.DRSiteID,
		Status:            dbPair.Status,
		ReplicationStatus: dbPair.ReplicationStatus,
		LastFailoverTest:  dbPair.LastFailoverTest,
		LastSyncAt:        dbPair.LastSyncAt,
		CreatedAt:         dbPair.CreatedAt,
		UpdatedAt:         dbPair.UpdatedAt,
	}

	if dbPair.RPO != nil {
		pair.RPO = *dbPair.RPO
	}
	if dbPair.RTO != nil {
		pair.RTO = *dbPair.RTO
	}

	// Enrich with site data
	primarySite, err := s.siteRepo.GetSiteWithAssetStats(ctx, dbPair.PrimarySiteID, input.OrgID)
	if err == nil && primarySite != nil {
		pair.PrimarySite = ResilienceSite{
			ID:         primarySite.ID,
			Name:       primarySite.Name,
			Region:     primarySite.Region,
			Platform:   primarySite.Platform,
			AssetCount: primarySite.AssetCount,
			Status:     primarySite.Status,
			RPO:        pair.RPO,
			RTO:        pair.RTO,
		}
		if primarySite.LastSyncAt != nil {
			pair.PrimarySite.LastSyncAt = *primarySite.LastSyncAt
		}
	}

	drSite, err := s.siteRepo.GetSiteWithAssetStats(ctx, dbPair.DRSiteID, input.OrgID)
	if err == nil && drSite != nil {
		pair.DRSite = ResilienceSite{
			ID:         drSite.ID,
			Name:       drSite.Name,
			Region:     drSite.Region,
			Platform:   drSite.Platform,
			AssetCount: drSite.AssetCount,
			Status:     drSite.Status,
			RPO:        pair.RPO,
			RTO:        pair.RTO,
		}
		if drSite.LastSyncAt != nil {
			pair.DRSite.LastSyncAt = *drSite.LastSyncAt
		}
	}

	return pair, nil
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
