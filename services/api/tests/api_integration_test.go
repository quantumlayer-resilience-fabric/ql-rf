//go:build integration

// Package tests contains integration tests for the API service.
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/database"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/routes"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
)

// APITestEnvironment holds the API test environment.
type APITestEnvironment struct {
	DB       *database.DB
	Server   *httptest.Server
	Config   *config.Config
	Logger   *logger.Logger
	OrgID    string
	UserID   string
	AuthToken string
}

// setupAPITestEnvironment creates an API test environment.
func setupAPITestEnvironment(t *testing.T) *APITestEnvironment {
	t.Helper()

	cfg := &config.Config{
		Env: "test",
		Database: config.DatabaseConfig{
			Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
			Port:     5432,
			User:     getEnvOrDefault("TEST_DB_USER", "postgres"),
			Password: getEnvOrDefault("TEST_DB_PASSWORD", "postgres"),
			Database: getEnvOrDefault("TEST_DB_NAME", "qlrf_test"),
		},
		API: config.APIConfig{
			DevMode: true,
		},
	}

	log := logger.New("error", "text")

	db, err := database.New(context.Background(), cfg.Database)
	if err != nil {
		t.Skipf("Skipping API integration test: database not available: %v", err)
	}

	// Initialize services
	assetService := service.NewAssetService(db, log)
	imageService := service.NewImageService(db, log)
	driftService := service.NewDriftService(db, log)

	// Create router
	router := routes.New(routes.Config{
		DB:           db,
		Config:       cfg,
		Logger:       log,
		AssetService: assetService,
		ImageService: imageService,
		DriftService: driftService,
	})

	server := httptest.NewServer(router)

	return &APITestEnvironment{
		DB:     db,
		Server: server,
		Config: cfg,
		Logger: log,
		OrgID:  "00000000-0000-0000-0000-000000000001",
		UserID: "test-user",
	}
}

func (env *APITestEnvironment) teardown() {
	if env.Server != nil {
		env.Server.Close()
	}
	if env.DB != nil {
		env.DB.Close()
	}
}

// =============================================================================
// Health Check Tests
// =============================================================================

func TestAPIHealthEndpoint(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// =============================================================================
// Overview Tests
// =============================================================================

func TestOverviewMetrics(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/api/v1/overview")
	require.NoError(t, err)
	defer resp.Body.Close()

	// May return 200 or 401 depending on auth setup
	assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized}, resp.StatusCode)
}

// =============================================================================
// Asset Tests
// =============================================================================

func TestListAssets(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/api/v1/assets")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized}, resp.StatusCode)

	if resp.StatusCode == http.StatusOK {
		var body map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&body)
		require.NoError(t, err)
		assert.Contains(t, body, "assets")
	}
}

func TestAssetFilters(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer env.teardown()

	testCases := []struct {
		name   string
		query  string
	}{
		{"by_platform", "platform=aws"},
		{"by_status", "status=running"},
		{"by_site", "site_id=00000000-0000-0000-0000-000000000001"},
		{"with_pagination", "limit=10&offset=0"},
		{"with_search", "search=web"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/api/v1/assets?%s", env.Server.URL, tc.query)
			resp, err := http.Get(url)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should return OK or unauthorized
			assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized}, resp.StatusCode)
		})
	}
}

// =============================================================================
// Image Tests
// =============================================================================

func TestListImages(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/api/v1/images")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized}, resp.StatusCode)
}

func TestImageFilters(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer env.teardown()

	testCases := []struct {
		name  string
		query string
	}{
		{"by_platform", "platform=aws"},
		{"golden_only", "golden=true"},
		{"with_search", "search=ubuntu"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/api/v1/images?%s", env.Server.URL, tc.query)
			resp, err := http.Get(url)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized}, resp.StatusCode)
		})
	}
}

// =============================================================================
// Drift Tests
// =============================================================================

func TestDriftSummary(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/api/v1/drift/summary")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized}, resp.StatusCode)

	if resp.StatusCode == http.StatusOK {
		var body map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&body)
		require.NoError(t, err)

		// Check expected fields
		assert.Contains(t, body, "drift_score")
		assert.Contains(t, body, "total_assets")
		assert.Contains(t, body, "drifted_assets")
	}
}

func TestDriftByAsset(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/api/v1/drift/assets")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized}, resp.StatusCode)
}

// =============================================================================
// Site Tests
// =============================================================================

func TestListSites(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/api/v1/sites")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized}, resp.StatusCode)
}

// =============================================================================
// Compliance Tests
// =============================================================================

func TestComplianceSummary(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/api/v1/compliance/summary")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized}, resp.StatusCode)
}

func TestComplianceControls(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/api/v1/compliance/controls")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized}, resp.StatusCode)
}

// =============================================================================
// Resilience Tests
// =============================================================================

func TestResilienceSummary(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/api/v1/resilience/summary")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized}, resp.StatusCode)
}

func TestDRPairs(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/api/v1/resilience/dr-pairs")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized}, resp.StatusCode)
}

// =============================================================================
// Alert Tests
// =============================================================================

func TestListAlerts(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/api/v1/alerts")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized}, resp.StatusCode)
}

func TestAlertFilters(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer env.teardown()

	testCases := []struct {
		name  string
		query string
	}{
		{"by_severity", "severity=critical"},
		{"active_only", "status=active"},
		{"with_limit", "limit=20"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/api/v1/alerts?%s", env.Server.URL, tc.query)
			resp, err := http.Get(url)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized}, resp.StatusCode)
		})
	}
}

// =============================================================================
// User Info Tests
// =============================================================================

func TestGetCurrentUser(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/api/v1/users/me")
	require.NoError(t, err)
	defer resp.Body.Close()

	// This requires authentication
	assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized}, resp.StatusCode)
}

// =============================================================================
// Error Response Tests
// =============================================================================

func TestNotFoundRoute(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/api/v1/nonexistent")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestMethodNotAllowed(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer env.teardown()

	resp, err := http.Post(env.Server.URL+"/health", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should be either 405 or 404 depending on router config
	assert.Contains(t, []int{http.StatusNotFound, http.StatusMethodNotAllowed}, resp.StatusCode)
}
