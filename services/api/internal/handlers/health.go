// Package handlers provides HTTP request handlers.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/quantumlayerhq/ql-rf/pkg/database"
)

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	db        *database.DB
	version   string
	gitCommit string
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(db *database.DB, version, gitCommit string) *HealthHandler {
	return &HealthHandler{
		db:        db,
		version:   version,
		gitCommit: gitCommit,
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

	// Check database connection
	if err := h.db.Health(r.Context()); err != nil {
		checks["database"] = "unhealthy: " + err.Error()
		allHealthy = false
	} else {
		checks["database"] = "healthy"
	}

	// TODO: Add Kafka health check
	// TODO: Add Redis health check

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
