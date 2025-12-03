package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
)

// MockDriftRepository is a mock implementation of DriftRepository.
type MockDriftRepository struct {
	mu               sync.RWMutex
	reports          []service.DriftReport
	environmentScope []service.DriftByScope
	platformScope    []service.DriftByScope
	siteScope        []service.DriftByScope
	trends           map[uuid.UUID][]service.DriftTrendPoint

	// Control behavior for testing
	GetDriftReportFunc        func(ctx context.Context, id uuid.UUID) (*service.DriftReport, error)
	GetLatestDriftReportFunc  func(ctx context.Context, orgID uuid.UUID) (*service.DriftReport, error)
	GetDriftByEnvironmentFunc func(ctx context.Context, orgID uuid.UUID) ([]service.DriftByScope, error)
	GetDriftByPlatformFunc    func(ctx context.Context, orgID uuid.UUID) ([]service.DriftByScope, error)
	GetDriftBySiteFunc        func(ctx context.Context, orgID uuid.UUID) ([]service.DriftByScope, error)
	GetDriftTrendFunc         func(ctx context.Context, orgID uuid.UUID, days int) ([]service.DriftTrendPoint, error)
	ListDriftReportsFunc      func(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]service.DriftReport, error)
	CreateDriftReportFunc     func(ctx context.Context, params service.CreateDriftReportParams) (*service.DriftReport, error)
}

// NewMockDriftRepository creates a new MockDriftRepository.
func NewMockDriftRepository() *MockDriftRepository {
	return &MockDriftRepository{
		reports:          make([]service.DriftReport, 0),
		environmentScope: make([]service.DriftByScope, 0),
		platformScope:    make([]service.DriftByScope, 0),
		siteScope:        make([]service.DriftByScope, 0),
		trends:           make(map[uuid.UUID][]service.DriftTrendPoint),
	}
}

// GetDriftReport returns a drift report by ID.
func (m *MockDriftRepository) GetDriftReport(ctx context.Context, id uuid.UUID) (*service.DriftReport, error) {
	if m.GetDriftReportFunc != nil {
		return m.GetDriftReportFunc(ctx, id)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for i := range m.reports {
		if m.reports[i].ID == id {
			return &m.reports[i], nil
		}
	}

	return nil, service.ErrNotFound
}

// GetLatestDriftReport returns the latest drift report for an org.
func (m *MockDriftRepository) GetLatestDriftReport(ctx context.Context, orgID uuid.UUID) (*service.DriftReport, error) {
	if m.GetLatestDriftReportFunc != nil {
		return m.GetLatestDriftReportFunc(ctx, orgID)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var latest *service.DriftReport
	for i := range m.reports {
		if m.reports[i].OrgID == orgID {
			if latest == nil || m.reports[i].CalculatedAt.After(latest.CalculatedAt) {
				latest = &m.reports[i]
			}
		}
	}

	if latest == nil {
		return nil, service.ErrNotFound
	}
	return latest, nil
}

// GetDriftByEnvironment returns drift grouped by environment.
func (m *MockDriftRepository) GetDriftByEnvironment(ctx context.Context, orgID uuid.UUID) ([]service.DriftByScope, error) {
	if m.GetDriftByEnvironmentFunc != nil {
		return m.GetDriftByEnvironmentFunc(ctx, orgID)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.environmentScope, nil
}

// GetDriftByPlatform returns drift grouped by platform.
func (m *MockDriftRepository) GetDriftByPlatform(ctx context.Context, orgID uuid.UUID) ([]service.DriftByScope, error) {
	if m.GetDriftByPlatformFunc != nil {
		return m.GetDriftByPlatformFunc(ctx, orgID)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.platformScope, nil
}

// GetDriftBySite returns drift grouped by site.
func (m *MockDriftRepository) GetDriftBySite(ctx context.Context, orgID uuid.UUID) ([]service.DriftByScope, error) {
	if m.GetDriftBySiteFunc != nil {
		return m.GetDriftBySiteFunc(ctx, orgID)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.siteScope, nil
}

// GetDriftTrend returns drift trend over time.
func (m *MockDriftRepository) GetDriftTrend(ctx context.Context, orgID uuid.UUID, days int) ([]service.DriftTrendPoint, error) {
	if m.GetDriftTrendFunc != nil {
		return m.GetDriftTrendFunc(ctx, orgID, days)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if trends, ok := m.trends[orgID]; ok {
		// Return up to 'days' points
		if len(trends) <= days {
			return trends, nil
		}
		return trends[:days], nil
	}

	// Default: return mock trend data
	points := make([]service.DriftTrendPoint, days)
	for i := 0; i < days; i++ {
		points[i] = service.DriftTrendPoint{
			Date:            time.Now().AddDate(0, 0, -days+i+1),
			AvgCoverage:     85.0 + float64(i%10),
			TotalAssets:     100,
			CompliantAssets: int64(85 + i%10),
		}
	}
	return points, nil
}

// ListDriftReports returns a paginated list of drift reports.
func (m *MockDriftRepository) ListDriftReports(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]service.DriftReport, error) {
	if m.ListDriftReportsFunc != nil {
		return m.ListDriftReportsFunc(ctx, orgID, limit, offset)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []service.DriftReport
	for _, r := range m.reports {
		if r.OrgID == orgID {
			result = append(result, r)
		}
	}

	// Apply pagination
	start := int(offset)
	if start >= len(result) {
		return []service.DriftReport{}, nil
	}
	end := start + int(limit)
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
}

// CreateDriftReport creates a new drift report.
func (m *MockDriftRepository) CreateDriftReport(ctx context.Context, params service.CreateDriftReportParams) (*service.DriftReport, error) {
	if m.CreateDriftReportFunc != nil {
		return m.CreateDriftReportFunc(ctx, params)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	report := &service.DriftReport{
		ID:              uuid.New(),
		OrgID:           params.OrgID,
		EnvID:           params.EnvID,
		Platform:        params.Platform,
		Site:            params.Site,
		TotalAssets:     params.TotalAssets,
		CompliantAssets: params.CompliantAssets,
		CoveragePct:     params.CoveragePct,
		CalculatedAt:    time.Now(),
	}

	m.reports = append(m.reports, *report)
	return report, nil
}

// AddReport adds a report directly to the mock (for test setup).
func (m *MockDriftRepository) AddReport(report *service.DriftReport) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reports = append(m.reports, *report)
}

// Reset clears all data from the mock.
func (m *MockDriftRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reports = make([]service.DriftReport, 0)
	m.environmentScope = make([]service.DriftByScope, 0)
	m.platformScope = make([]service.DriftByScope, 0)
	m.siteScope = make([]service.DriftByScope, 0)
	m.trends = make(map[uuid.UUID][]service.DriftTrendPoint)
}

// AddEnvironmentScope adds an environment scope to the mock.
func (m *MockDriftRepository) AddEnvironmentScope(scope *service.DriftByScope) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.environmentScope = append(m.environmentScope, *scope)
}

// AddPlatformScope adds a platform scope to the mock.
func (m *MockDriftRepository) AddPlatformScope(scope *service.DriftByScope) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.platformScope = append(m.platformScope, *scope)
}

// AddSiteScope adds a site scope to the mock.
func (m *MockDriftRepository) AddSiteScope(scope *service.DriftByScope) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.siteScope = append(m.siteScope, *scope)
}

// AddTrendPoint adds a trend point to the mock for a specific org.
func (m *MockDriftRepository) AddTrendPoint(point *service.DriftTrendPoint, orgID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trends[orgID] = append(m.trends[orgID], *point)
}
