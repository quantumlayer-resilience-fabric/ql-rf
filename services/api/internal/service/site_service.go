package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// SiteService handles site business logic.
type SiteService struct {
	repo SiteRepository
}

// NewSiteService creates a new SiteService.
func NewSiteService(repo SiteRepository) *SiteService {
	return &SiteService{repo: repo}
}

// GetSiteInput contains input for getting a site.
type GetSiteInput struct {
	ID    uuid.UUID
	OrgID uuid.UUID
}

// GetSite retrieves a site by ID with authorization check.
func (s *SiteService) GetSite(ctx context.Context, input GetSiteInput) (*SiteWithStats, error) {
	site, err := s.repo.GetSiteWithAssetStats(ctx, input.ID, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("get site: %w", err)
	}

	// Authorization: verify org ownership
	if site.OrgID != input.OrgID {
		return nil, ErrNotFound
	}

	return site, nil
}

// ListSitesInput contains input for listing sites.
type ListSitesInput struct {
	OrgID    uuid.UUID
	Platform *string
	Region   *string
	Page     int
	PageSize int
}

// ListSitesOutput contains output for listing sites.
type ListSitesOutput struct {
	Sites      []SiteWithStats `json:"sites"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	TotalPages int             `json:"totalPages"`
}

// ListSites retrieves a list of sites with asset statistics.
func (s *SiteService) ListSites(ctx context.Context, input ListSitesInput) (*ListSitesOutput, error) {
	// Apply defaults
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	// Get sites with stats
	sites, err := s.repo.ListSitesWithStats(ctx, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}

	// Apply platform filter if specified
	if input.Platform != nil {
		filtered := make([]SiteWithStats, 0)
		for _, site := range sites {
			if site.Platform == *input.Platform {
				filtered = append(filtered, site)
			}
		}
		sites = filtered
	}

	// Apply region filter if specified
	if input.Region != nil {
		filtered := make([]SiteWithStats, 0)
		for _, site := range sites {
			if site.Region == *input.Region {
				filtered = append(filtered, site)
			}
		}
		sites = filtered
	}

	total := int64(len(sites))

	// Apply pagination
	start := (input.Page - 1) * input.PageSize
	end := start + input.PageSize
	if start > len(sites) {
		start = len(sites)
	}
	if end > len(sites) {
		end = len(sites)
	}

	pagedSites := sites[start:end]

	totalPages := int(total) / input.PageSize
	if int(total)%input.PageSize > 0 {
		totalPages++
	}

	return &ListSitesOutput{
		Sites:      pagedSites,
		Total:      total,
		Page:       input.Page,
		PageSize:   input.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetSiteSummaryInput contains input for getting site summary.
type GetSiteSummaryInput struct {
	OrgID uuid.UUID
}

// SiteSummary contains summary statistics for sites.
type SiteSummary struct {
	TotalSites     int64            `json:"totalSites"`
	HealthySites   int64            `json:"healthySites"`
	WarningSites   int64            `json:"warningSites"`
	CriticalSites  int64            `json:"criticalSites"`
	DRPairedSites  int64            `json:"drPairedSites"`
	ByPlatform     map[string]int64 `json:"byPlatform"`
	ByRegion       map[string]int64 `json:"byRegion"`
	ByEnvironment  map[string]int64 `json:"byEnvironment"`
}

// GetSiteSummary retrieves site summary statistics.
func (s *SiteService) GetSiteSummary(ctx context.Context, input GetSiteSummaryInput) (*SiteSummary, error) {
	sites, err := s.repo.ListSitesWithStats(ctx, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}

	summary := &SiteSummary{
		TotalSites:    int64(len(sites)),
		ByPlatform:    make(map[string]int64),
		ByRegion:      make(map[string]int64),
		ByEnvironment: make(map[string]int64),
	}

	for _, site := range sites {
		// Count by status
		switch site.Status {
		case "healthy":
			summary.HealthySites++
		case "warning":
			summary.WarningSites++
		case "critical":
			summary.CriticalSites++
		}

		// Count DR paired
		if site.DRPaired {
			summary.DRPairedSites++
		}

		// Count by platform
		summary.ByPlatform[site.Platform]++

		// Count by region
		summary.ByRegion[site.Region]++

		// Count by environment
		summary.ByEnvironment[site.Environment]++
	}

	return summary, nil
}
