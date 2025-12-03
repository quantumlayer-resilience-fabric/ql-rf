// Package handlers provides HTTP request handlers.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/quantumlayerhq/ql-rf/pkg/database"
	"github.com/quantumlayerhq/ql-rf/pkg/kafka"
)

// HealthChecker defines the interface for health checks.
type HealthChecker interface {
	Health(ctx context.Context) error
}

// RedisClient defines the interface for Redis client health checks.
type RedisClient interface {
	Ping(ctx context.Context) error
}

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	db        *database.DB
	kafka     *kafka.Client
	redis     RedisClient
	version   string
	gitCommit string
}

// HealthHandlerConfig contains configuration for the health handler.
type HealthHandlerConfig struct {
	DB        *database.DB
	Kafka     *kafka.Client
	Redis     RedisClient
	Version   string
	GitCommit string
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(db *database.DB, version, gitCommit string) *HealthHandler {
	return &HealthHandler{
		db:        db,
		version:   version,
		gitCommit: gitCommit,
	}
}

// NewHealthHandlerWithDeps creates a new HealthHandler with all dependencies.
func NewHealthHandlerWithDeps(cfg HealthHandlerConfig) *HealthHandler {
	return &HealthHandler{
		db:        cfg.DB,
		kafka:     cfg.Kafka,
		redis:     cfg.Redis,
		version:   cfg.Version,
		gitCommit: cfg.GitCommit,
	}
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status  string            `json:"status"`
	Checks  map[string]string `json:"checks,omitempty"`
	Message string            `json:"message,omitempty"`
}

// VersionResponse represents the version endpoint response.
type VersionResponse struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	Service   string `json:"service"`
}

// Liveness handles the liveness probe.
// This endpoint returns 200 if the service is running.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status: "ok",
	}
	writeJSON(w, http.StatusOK, response)
}

// Readiness handles the readiness probe.
// This endpoint returns 200 if the service is ready to accept traffic.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]string)
	allHealthy := true

	// Use context with timeout for health checks
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check database connection
	if err := h.db.Health(ctx); err != nil {
		checks["database"] = "unhealthy: " + err.Error()
		allHealthy = false
	} else {
		checks["database"] = "healthy"
	}

	// Check Kafka connection
	if h.kafka != nil {
		if err := h.kafka.Health(ctx); err != nil {
			checks["kafka"] = "unhealthy: " + err.Error()
			// Kafka is not critical for basic API operations
		} else {
			checks["kafka"] = "healthy"
		}
	} else {
		checks["kafka"] = "not configured"
	}

	// Check Redis connection
	if h.redis != nil {
		if err := h.redis.Ping(ctx); err != nil {
			checks["redis"] = "unhealthy: " + err.Error()
			// Redis is not critical for basic API operations
		} else {
			checks["redis"] = "healthy"
		}
	} else {
		checks["redis"] = "not configured"
	}

	status := "ok"
	httpStatus := http.StatusOK
	if !allHealthy {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status: status,
		Checks: checks,
	}
	writeJSON(w, httpStatus, response)
}

// Version handles the version endpoint.
func (h *HealthHandler) Version(w http.ResponseWriter, r *http.Request) {
	response := VersionResponse{
		Version:   h.version,
		GitCommit: h.gitCommit,
		Service:   "ql-rf-api",
	}
	writeJSON(w, http.StatusOK, response)
}

// Metrics handles the Prometheus metrics endpoint.
func (h *HealthHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// NotImplemented is a placeholder handler for unimplemented endpoints.
func NotImplemented(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"error":   "not_implemented",
		"message": "This endpoint is not yet implemented",
	}
	writeJSON(w, http.StatusNotImplemented, response)
}
