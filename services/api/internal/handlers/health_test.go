package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayerhq/ql-rf/services/api/internal/handlers"
)

// MockDB implements a mock database for testing.
type MockDB struct {
	HealthError error
}

func (m *MockDB) Health(ctx context.Context) error {
	return m.HealthError
}

// MockKafka implements a mock Kafka client for testing.
type MockKafka struct {
	HealthError error
}

func (m *MockKafka) Health(ctx context.Context) error {
	return m.HealthError
}

// MockRedis implements a mock Redis client for testing.
type MockRedis struct {
	PingError error
}

func (m *MockRedis) Ping(ctx context.Context) error {
	return m.PingError
}

func TestHealthHandler_Liveness(t *testing.T) {
	handler := handlers.NewHealthHandler(nil, "1.0.0", "abc123")

	t.Run("returns 200 OK", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()

		handler.Liveness(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response handlers.HealthResponse
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, "ok", response.Status)
	})
}

func TestHealthHandler_Version(t *testing.T) {
	handler := handlers.NewHealthHandler(nil, "2.0.0", "def456")

	t.Run("returns version info", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/version", nil)
		rr := httptest.NewRecorder()

		handler.Version(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response handlers.VersionResponse
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, "2.0.0", response.Version)
		assert.Equal(t, "def456", response.GitCommit)
		assert.Equal(t, "ql-rf-api", response.Service)
	})
}

func TestNotImplemented(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/not-implemented", nil)
	rr := httptest.NewRecorder()

	handlers.NotImplemented(rr, req)

	assert.Equal(t, http.StatusNotImplemented, rr.Code)

	var response map[string]string
	require.NoError(t, decodeJSON(rr, &response))
	assert.Equal(t, "not_implemented", response["error"])
}

func TestNewHealthHandler(t *testing.T) {
	handler := handlers.NewHealthHandler(nil, "1.0.0", "abc123")
	assert.NotNil(t, handler)
}

func TestNewHealthHandlerWithDeps(t *testing.T) {
	mockRedis := &MockRedis{}

	handler := handlers.NewHealthHandlerWithDeps(handlers.HealthHandlerConfig{
		Redis:     mockRedis,
		Version:   "2.0.0",
		GitCommit: "def456",
	})

	assert.NotNil(t, handler)
}

func TestHealthResponse_Structure(t *testing.T) {
	response := handlers.HealthResponse{
		Status:  "ok",
		Checks:  map[string]string{"database": "healthy"},
		Message: "all systems operational",
	}

	assert.Equal(t, "ok", response.Status)
	assert.Equal(t, "healthy", response.Checks["database"])
	assert.Equal(t, "all systems operational", response.Message)
}

func TestVersionResponse_Structure(t *testing.T) {
	response := handlers.VersionResponse{
		Version:   "1.0.0",
		GitCommit: "abc123",
		Service:   "ql-rf-api",
	}

	assert.Equal(t, "1.0.0", response.Version)
	assert.Equal(t, "abc123", response.GitCommit)
	assert.Equal(t, "ql-rf-api", response.Service)
}

func TestHealthHandlerConfig_Fields(t *testing.T) {
	mockRedis := &MockRedis{}

	cfg := handlers.HealthHandlerConfig{
		Redis:     mockRedis,
		Version:   "1.0.0",
		GitCommit: "abc123",
	}

	assert.NotNil(t, cfg.Redis)
	assert.Equal(t, "1.0.0", cfg.Version)
	assert.Equal(t, "abc123", cfg.GitCommit)
}

func TestNotImplemented_Methods(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/not-implemented", nil)
			rr := httptest.NewRecorder()

			handlers.NotImplemented(rr, req)

			assert.Equal(t, http.StatusNotImplemented, rr.Code)
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
		})
	}
}

func TestHealthHandler_Liveness_Methods(t *testing.T) {
	handler := handlers.NewHealthHandler(nil, "1.0.0", "abc123")

	methods := []string{
		http.MethodGet,
		http.MethodHead,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/healthz", nil)
			rr := httptest.NewRecorder()

			handler.Liveness(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
		})
	}
}

func TestHealthHandler_Version_Methods(t *testing.T) {
	handler := handlers.NewHealthHandler(nil, "1.0.0", "abc123")

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rr := httptest.NewRecorder()

	handler.Version(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestHealthResponse_JSONEncode(t *testing.T) {
	response := handlers.HealthResponse{
		Status: "ok",
		Checks: map[string]string{
			"database": "healthy",
			"kafka":    "healthy",
			"redis":    "healthy",
		},
	}

	data, err := json.Marshal(response)
	require.NoError(t, err)

	var parsed handlers.HealthResponse
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, response.Status, parsed.Status)
	assert.Equal(t, response.Checks["database"], parsed.Checks["database"])
}

func TestVersionResponse_JSONEncode(t *testing.T) {
	response := handlers.VersionResponse{
		Version:   "1.0.0",
		GitCommit: "abc123",
		Service:   "ql-rf-api",
	}

	data, err := json.Marshal(response)
	require.NoError(t, err)

	var parsed handlers.VersionResponse
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, response.Version, parsed.Version)
	assert.Equal(t, response.GitCommit, parsed.GitCommit)
	assert.Equal(t, response.Service, parsed.Service)
}
