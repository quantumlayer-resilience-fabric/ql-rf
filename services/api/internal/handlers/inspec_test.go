package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayerhq/ql-rf/pkg/inspec"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/handlers"
)

// MockInSpecService implements handlers.InSpecServiceInterface for testing.
type MockInSpecService struct {
	profiles        map[uuid.UUID]*inspec.Profile
	availableProfiles []inspec.AvailableProfile
	runs            map[uuid.UUID]*inspec.Run
	runSummaries    []inspec.RunSummary
	results         map[uuid.UUID][]inspec.Result
	mappings        map[uuid.UUID][]inspec.ControlMapping
	shouldFail      bool
}

// NewMockInSpecService creates a new mock InSpec service.
func NewMockInSpecService() *MockInSpecService {
	return &MockInSpecService{
		profiles:        make(map[uuid.UUID]*inspec.Profile),
		availableProfiles: []inspec.AvailableProfile{},
		runs:            make(map[uuid.UUID]*inspec.Run),
		runSummaries:    []inspec.RunSummary{},
		results:         make(map[uuid.UUID][]inspec.Result),
		mappings:        make(map[uuid.UUID][]inspec.ControlMapping),
	}
}

// AddProfile adds a profile to the mock service.
func (m *MockInSpecService) AddProfile(p *inspec.Profile) {
	m.profiles[p.ID] = p
}

// AddAvailableProfile adds an available profile to the mock service.
func (m *MockInSpecService) AddAvailableProfile(p inspec.AvailableProfile) {
	m.availableProfiles = append(m.availableProfiles, p)
}

// AddRun adds a run to the mock service.
func (m *MockInSpecService) AddRun(r *inspec.Run) {
	m.runs[r.ID] = r
}

// AddRunSummary adds a run summary to the mock service.
func (m *MockInSpecService) AddRunSummary(r inspec.RunSummary) {
	m.runSummaries = append(m.runSummaries, r)
}

// AddResults adds results for a run.
func (m *MockInSpecService) AddResults(runID uuid.UUID, results []inspec.Result) {
	m.results[runID] = results
}

// AddMappings adds mappings for a profile.
func (m *MockInSpecService) AddMappings(profileID uuid.UUID, mappings []inspec.ControlMapping) {
	m.mappings[profileID] = mappings
}

// SetShouldFail configures the mock to fail.
func (m *MockInSpecService) SetShouldFail(fail bool) {
	m.shouldFail = fail
}

// GetAvailableProfiles implements InSpecServiceInterface.
func (m *MockInSpecService) GetAvailableProfiles(ctx context.Context) ([]inspec.AvailableProfile, error) {
	if m.shouldFail {
		return nil, errors.New("mock error")
	}
	return m.availableProfiles, nil
}

// GetProfile implements InSpecServiceInterface.
func (m *MockInSpecService) GetProfile(ctx context.Context, profileID uuid.UUID) (*inspec.Profile, error) {
	if m.shouldFail {
		return nil, errors.New("mock error")
	}
	if p, ok := m.profiles[profileID]; ok {
		return p, nil
	}
	return nil, nil
}

// CreateProfile implements InSpecServiceInterface.
func (m *MockInSpecService) CreateProfile(ctx context.Context, profile inspec.Profile) (*inspec.Profile, error) {
	if m.shouldFail {
		return nil, errors.New("mock error")
	}
	profile.ID = uuid.New()
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()
	m.profiles[profile.ID] = &profile
	return &profile, nil
}

// CreateRun implements InSpecServiceInterface.
func (m *MockInSpecService) CreateRun(ctx context.Context, orgID, assetID, profileID uuid.UUID) (*inspec.Run, error) {
	if m.shouldFail {
		return nil, errors.New("mock error")
	}
	run := &inspec.Run{
		ID:        uuid.New(),
		OrgID:     orgID,
		AssetID:   assetID,
		ProfileID: profileID,
		Status:    inspec.RunStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.runs[run.ID] = run
	return run, nil
}

// ListRuns implements InSpecServiceInterface.
func (m *MockInSpecService) ListRuns(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]inspec.RunSummary, error) {
	if m.shouldFail {
		return nil, errors.New("mock error")
	}
	return m.runSummaries, nil
}

// GetRun implements InSpecServiceInterface.
func (m *MockInSpecService) GetRun(ctx context.Context, runID uuid.UUID) (*inspec.Run, error) {
	if m.shouldFail {
		return nil, errors.New("mock error")
	}
	if r, ok := m.runs[runID]; ok {
		return r, nil
	}
	return nil, nil
}

// GetRunResults implements InSpecServiceInterface.
func (m *MockInSpecService) GetRunResults(ctx context.Context, runID uuid.UUID) ([]inspec.Result, error) {
	if m.shouldFail {
		return nil, errors.New("mock error")
	}
	if r, ok := m.results[runID]; ok {
		return r, nil
	}
	return []inspec.Result{}, nil
}

// UpdateRunStatus implements InSpecServiceInterface.
func (m *MockInSpecService) UpdateRunStatus(ctx context.Context, runID uuid.UUID, status inspec.RunStatus, errorMsg string) error {
	if m.shouldFail {
		return errors.New("mock error")
	}
	if r, ok := m.runs[runID]; ok {
		r.Status = status
		r.ErrorMessage = errorMsg
		return nil
	}
	return errors.New("run not found")
}

// GetControlMappings implements InSpecServiceInterface.
func (m *MockInSpecService) GetControlMappings(ctx context.Context, profileID uuid.UUID) ([]inspec.ControlMapping, error) {
	if m.shouldFail {
		return nil, errors.New("mock error")
	}
	if mappings, ok := m.mappings[profileID]; ok {
		return mappings, nil
	}
	return []inspec.ControlMapping{}, nil
}

// CreateControlMapping implements InSpecServiceInterface.
func (m *MockInSpecService) CreateControlMapping(ctx context.Context, mapping inspec.ControlMapping) (*inspec.ControlMapping, error) {
	if m.shouldFail {
		return nil, errors.New("mock error")
	}
	mapping.ID = uuid.New()
	mapping.CreatedAt = time.Now()
	mapping.UpdatedAt = time.Now()
	if _, ok := m.mappings[mapping.ProfileID]; !ok {
		m.mappings[mapping.ProfileID] = []inspec.ControlMapping{}
	}
	m.mappings[mapping.ProfileID] = append(m.mappings[mapping.ProfileID], mapping)
	return &mapping, nil
}

func TestInSpecHandler_ListProfiles(t *testing.T) {
	log := logger.New("debug", "text")
	mockSvc := NewMockInSpecService()

	// Add test profiles
	mockSvc.AddAvailableProfile(inspec.AvailableProfile{
		ProfileID:    uuid.New(),
		Name:         "linux-baseline",
		Title:        "Linux Security Baseline",
		Version:      "2.0.0",
		Framework:    "CIS",
		FrameworkID:  uuid.New(),
		Platforms:    []string{"linux"},
		ControlCount: 150,
	})
	mockSvc.AddAvailableProfile(inspec.AvailableProfile{
		ProfileID:    uuid.New(),
		Name:         "aws-baseline",
		Title:        "AWS Security Baseline",
		Version:      "1.0.0",
		Framework:    "CIS",
		FrameworkID:  uuid.New(),
		Platforms:    []string{"aws"},
		ControlCount: 50,
	})

	handler := handlers.NewInSpecHandlerWithInterface(mockSvc, log)

	t.Run("returns available profiles", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inspec/profiles", nil)
		req = withOrgContext(req)

		rr := executeRequest(handler.ListProfiles, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string][]inspec.AvailableProfile
		require.NoError(t, decodeJSON(rr, &response))
		assert.Len(t, response["profiles"], 2)
	})

	t.Run("unauthorized without org context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inspec/profiles", nil)

		rr := executeRequest(handler.ListProfiles, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("returns 500 on service error", func(t *testing.T) {
		mockSvc.SetShouldFail(true)
		defer mockSvc.SetShouldFail(false)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/inspec/profiles", nil)
		req = withOrgContext(req)

		rr := executeRequest(handler.ListProfiles, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestInSpecHandler_GetProfile(t *testing.T) {
	log := logger.New("debug", "text")
	mockSvc := NewMockInSpecService()

	profileID := uuid.New()
	frameworkID := uuid.New()

	// Add test profile
	mockSvc.AddProfile(&inspec.Profile{
		ID:          profileID,
		Name:        "linux-baseline",
		Version:     "2.0.0",
		Title:       "Linux Security Baseline",
		Maintainer:  "Test",
		Summary:     "Linux security baseline profile",
		FrameworkID: frameworkID,
		ProfileURL:  "https://github.com/example/linux-baseline",
		Platforms:   []string{"linux"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})

	handler := handlers.NewInSpecHandlerWithInterface(mockSvc, log)

	t.Run("returns profile by ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inspec/profiles/"+profileID.String(), nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"profileId": profileID.String()})

		rr := executeRequest(handler.GetProfile, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response inspec.Profile
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, profileID, response.ID)
		assert.Equal(t, "linux-baseline", response.Name)
	})

	t.Run("returns 404 for non-existent profile", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inspec/profiles/"+nonExistentID.String(), nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"profileId": nonExistentID.String()})

		rr := executeRequest(handler.GetProfile, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inspec/profiles/invalid", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"profileId": "invalid"})

		rr := executeRequest(handler.GetProfile, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestInSpecHandler_CreateProfile(t *testing.T) {
	log := logger.New("debug", "text")
	mockSvc := NewMockInSpecService()
	frameworkID := uuid.New()

	handler := handlers.NewInSpecHandlerWithInterface(mockSvc, log)

	t.Run("creates new profile", func(t *testing.T) {
		body := inspec.CreateProfileRequest{
			Name:        "custom-profile",
			Version:     "1.0.0",
			Title:       "Custom Security Profile",
			Maintainer:  "Test User",
			Summary:     "Custom security checks",
			FrameworkID: frameworkID,
			ProfileURL:  "https://github.com/example/custom-profile",
			Platforms:   []string{"linux", "windows"},
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/inspec/profiles", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)

		rr := executeRequest(handler.CreateProfile, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var response inspec.Profile
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, "custom-profile", response.Name)
		assert.Equal(t, "1.0.0", response.Version)
	})

	t.Run("returns 400 for missing required fields", func(t *testing.T) {
		body := inspec.CreateProfileRequest{
			Name: "test",
			// Missing version, title, framework_id
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/inspec/profiles", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)

		rr := executeRequest(handler.CreateProfile, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/inspec/profiles", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)

		rr := executeRequest(handler.CreateProfile, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("unauthorized without org context", func(t *testing.T) {
		body := inspec.CreateProfileRequest{
			Name:        "test",
			Version:     "1.0.0",
			Title:       "Test",
			FrameworkID: frameworkID,
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/inspec/profiles", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		rr := executeRequest(handler.CreateProfile, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestInSpecHandler_RunProfile(t *testing.T) {
	log := logger.New("debug", "text")
	mockSvc := NewMockInSpecService()

	profileID := uuid.New()
	assetID := uuid.New()

	handler := handlers.NewInSpecHandlerWithInterface(mockSvc, log)

	t.Run("creates run successfully", func(t *testing.T) {
		body := inspec.RunProfileRequest{
			ProfileID: profileID,
			AssetID:   assetID,
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/inspec/run", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)

		rr := executeRequest(handler.RunProfile, req)

		assert.Equal(t, http.StatusAccepted, rr.Code)

		var response inspec.Run
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, inspec.RunStatusPending, response.Status)
	})

	t.Run("returns 400 for missing profile_id", func(t *testing.T) {
		body := inspec.RunProfileRequest{
			AssetID: assetID,
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/inspec/run", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)

		rr := executeRequest(handler.RunProfile, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns 400 for missing asset_id", func(t *testing.T) {
		body := inspec.RunProfileRequest{
			ProfileID: profileID,
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/inspec/run", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)

		rr := executeRequest(handler.RunProfile, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns 500 on service error", func(t *testing.T) {
		mockSvc.SetShouldFail(true)
		defer mockSvc.SetShouldFail(false)

		body := inspec.RunProfileRequest{
			ProfileID: profileID,
			AssetID:   assetID,
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/inspec/run", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)

		rr := executeRequest(handler.RunProfile, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestInSpecHandler_ListRuns(t *testing.T) {
	log := logger.New("debug", "text")
	mockSvc := NewMockInSpecService()

	// Add test run summaries
	now := time.Now()
	mockSvc.AddRunSummary(inspec.RunSummary{
		RunID:       uuid.New(),
		AssetID:     uuid.New(),
		AssetName:   "server-01",
		ProfileName: "linux-baseline",
		Framework:   "CIS",
		Status:      inspec.RunStatusCompleted,
		StartedAt:   &now,
		CompletedAt: &now,
		Duration:    120,
		PassRate:    95.0,
		TotalTests:  100,
		PassedTests: 95,
		FailedTests: 5,
	})

	handler := handlers.NewInSpecHandlerWithInterface(mockSvc, log)

	t.Run("returns runs list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inspec/runs", nil)
		req = withOrgContext(req)

		rr := executeRequest(handler.ListRuns, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		require.NoError(t, decodeJSON(rr, &response))
		runs := response["runs"].([]interface{})
		assert.Len(t, runs, 1)
	})

	t.Run("respects limit and offset params", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inspec/runs?limit=10&offset=0", nil)
		req = withOrgContext(req)

		rr := executeRequest(handler.ListRuns, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, float64(10), response["limit"])
		assert.Equal(t, float64(0), response["offset"])
	})

	t.Run("unauthorized without org context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inspec/runs", nil)

		rr := executeRequest(handler.ListRuns, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestInSpecHandler_GetRun(t *testing.T) {
	log := logger.New("debug", "text")
	mockSvc := NewMockInSpecService()

	orgID := testOrg().ID
	runID := uuid.New()

	// Add test run
	mockSvc.AddRun(&inspec.Run{
		ID:          runID,
		OrgID:       orgID,
		AssetID:     uuid.New(),
		ProfileID:   uuid.New(),
		Status:      inspec.RunStatusCompleted,
		TotalTests:  50,
		PassedTests: 48,
		FailedTests: 2,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})

	handler := handlers.NewInSpecHandlerWithInterface(mockSvc, log)

	t.Run("returns run by ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inspec/runs/"+runID.String(), nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"runId": runID.String()})

		rr := executeRequest(handler.GetRun, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response inspec.Run
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, runID, response.ID)
		assert.Equal(t, inspec.RunStatusCompleted, response.Status)
	})

	t.Run("returns 404 for non-existent run", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inspec/runs/"+nonExistentID.String(), nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"runId": nonExistentID.String()})

		rr := executeRequest(handler.GetRun, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns 404 for run from different org", func(t *testing.T) {
		otherOrgRunID := uuid.New()
		mockSvc.AddRun(&inspec.Run{
			ID:        otherOrgRunID,
			OrgID:     uuid.New(), // Different org
			AssetID:   uuid.New(),
			ProfileID: uuid.New(),
			Status:    inspec.RunStatusPending,
		})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/inspec/runs/"+otherOrgRunID.String(), nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"runId": otherOrgRunID.String()})

		rr := executeRequest(handler.GetRun, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns 400 for invalid run ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inspec/runs/invalid", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"runId": "invalid"})

		rr := executeRequest(handler.GetRun, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestInSpecHandler_GetRunResults(t *testing.T) {
	log := logger.New("debug", "text")
	mockSvc := NewMockInSpecService()

	orgID := testOrg().ID
	runID := uuid.New()

	// Add test run
	mockSvc.AddRun(&inspec.Run{
		ID:        runID,
		OrgID:     orgID,
		AssetID:   uuid.New(),
		ProfileID: uuid.New(),
		Status:    inspec.RunStatusCompleted,
	})

	// Add test results
	mockSvc.AddResults(runID, []inspec.Result{
		{
			ID:           uuid.New(),
			RunID:        runID,
			ControlID:    "control-01",
			ControlTitle: "Ensure SSH is configured",
			Status:       inspec.ResultStatusPassed,
			RunTime:      0.5,
		},
		{
			ID:           uuid.New(),
			RunID:        runID,
			ControlID:    "control-02",
			ControlTitle: "Ensure firewall is enabled",
			Status:       inspec.ResultStatusFailed,
			Message:      "Firewall is disabled",
			RunTime:      0.3,
		},
	})

	handler := handlers.NewInSpecHandlerWithInterface(mockSvc, log)

	t.Run("returns run results", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inspec/runs/"+runID.String()+"/results", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"runId": runID.String()})

		rr := executeRequest(handler.GetRunResults, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response inspec.RunResultsResponse
		require.NoError(t, decodeJSON(rr, &response))
		assert.Len(t, response.Results, 2)
	})

	t.Run("returns 404 for non-existent run", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inspec/runs/"+nonExistentID.String()+"/results", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"runId": nonExistentID.String()})

		rr := executeRequest(handler.GetRunResults, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns 400 for invalid run ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inspec/runs/invalid/results", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"runId": "invalid"})

		rr := executeRequest(handler.GetRunResults, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestInSpecHandler_CancelRun(t *testing.T) {
	log := logger.New("debug", "text")
	mockSvc := NewMockInSpecService()

	orgID := testOrg().ID

	handler := handlers.NewInSpecHandlerWithInterface(mockSvc, log)

	t.Run("cancels pending run", func(t *testing.T) {
		runID := uuid.New()
		mockSvc.AddRun(&inspec.Run{
			ID:        runID,
			OrgID:     orgID,
			AssetID:   uuid.New(),
			ProfileID: uuid.New(),
			Status:    inspec.RunStatusPending,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/inspec/runs/"+runID.String()+"/cancel", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"runId": runID.String()})

		rr := executeRequest(handler.CancelRun, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, "run cancelled successfully", response["message"])
	})

	t.Run("cancels running run", func(t *testing.T) {
		runID := uuid.New()
		mockSvc.AddRun(&inspec.Run{
			ID:        runID,
			OrgID:     orgID,
			AssetID:   uuid.New(),
			ProfileID: uuid.New(),
			Status:    inspec.RunStatusRunning,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/inspec/runs/"+runID.String()+"/cancel", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"runId": runID.String()})

		rr := executeRequest(handler.CancelRun, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("returns 400 for completed run", func(t *testing.T) {
		runID := uuid.New()
		mockSvc.AddRun(&inspec.Run{
			ID:        runID,
			OrgID:     orgID,
			AssetID:   uuid.New(),
			ProfileID: uuid.New(),
			Status:    inspec.RunStatusCompleted,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/inspec/runs/"+runID.String()+"/cancel", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"runId": runID.String()})

		rr := executeRequest(handler.CancelRun, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns 404 for non-existent run", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/inspec/runs/"+nonExistentID.String()+"/cancel", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"runId": nonExistentID.String()})

		rr := executeRequest(handler.CancelRun, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns 400 for invalid run ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/inspec/runs/invalid/cancel", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"runId": "invalid"})

		rr := executeRequest(handler.CancelRun, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestInSpecHandler_GetControlMappings(t *testing.T) {
	log := logger.New("debug", "text")
	mockSvc := NewMockInSpecService()

	profileID := uuid.New()

	// Add test mappings
	mockSvc.AddMappings(profileID, []inspec.ControlMapping{
		{
			ID:                  uuid.New(),
			InSpecControlID:     "control-01",
			ComplianceControlID: uuid.New(),
			ProfileID:           profileID,
			MappingConfidence:   1.0,
		},
		{
			ID:                  uuid.New(),
			InSpecControlID:     "control-02",
			ComplianceControlID: uuid.New(),
			ProfileID:           profileID,
			MappingConfidence:   0.9,
		},
	})

	handler := handlers.NewInSpecHandlerWithInterface(mockSvc, log)

	t.Run("returns control mappings", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inspec/profiles/"+profileID.String()+"/mappings", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"profileId": profileID.String()})

		rr := executeRequest(handler.GetControlMappings, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string][]inspec.ControlMapping
		require.NoError(t, decodeJSON(rr, &response))
		assert.Len(t, response["mappings"], 2)
	})

	t.Run("returns 400 for invalid profile ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inspec/profiles/invalid/mappings", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"profileId": "invalid"})

		rr := executeRequest(handler.GetControlMappings, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestInSpecHandler_CreateControlMapping(t *testing.T) {
	log := logger.New("debug", "text")
	mockSvc := NewMockInSpecService()

	profileID := uuid.New()
	complianceControlID := uuid.New()

	handler := handlers.NewInSpecHandlerWithInterface(mockSvc, log)

	t.Run("creates control mapping", func(t *testing.T) {
		body := map[string]interface{}{
			"inspec_control_id":     "control-01",
			"compliance_control_id": complianceControlID.String(),
			"mapping_confidence":    0.95,
			"notes":                 "Mapped automatically",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/inspec/profiles/"+profileID.String()+"/mappings", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"profileId": profileID.String()})

		rr := executeRequest(handler.CreateControlMapping, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var response inspec.ControlMapping
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, "control-01", response.InSpecControlID)
		assert.Equal(t, 0.95, response.MappingConfidence)
	})

	t.Run("returns 400 for missing required fields", func(t *testing.T) {
		body := map[string]interface{}{
			"inspec_control_id": "control-01",
			// Missing compliance_control_id
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/inspec/profiles/"+profileID.String()+"/mappings", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"profileId": profileID.String()})

		rr := executeRequest(handler.CreateControlMapping, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/inspec/profiles/"+profileID.String()+"/mappings", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"profileId": profileID.String()})

		rr := executeRequest(handler.CreateControlMapping, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns 400 for invalid profile ID", func(t *testing.T) {
		body := map[string]interface{}{
			"inspec_control_id":     "control-01",
			"compliance_control_id": complianceControlID.String(),
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/inspec/profiles/invalid/mappings", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"profileId": "invalid"})

		rr := executeRequest(handler.CreateControlMapping, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
