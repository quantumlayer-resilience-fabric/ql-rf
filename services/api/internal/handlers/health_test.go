package handlers_test

import (
	"context"
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
