package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
)

// ConnectorHandler handles connector-related HTTP requests.
type ConnectorHandler struct {
	svc *service.ConnectorService
	log *logger.Logger
}

// NewConnectorHandler creates a new ConnectorHandler.
func NewConnectorHandler(svc *service.ConnectorService, log *logger.Logger) *ConnectorHandler {
	return &ConnectorHandler{
		svc: svc,
		log: log.WithComponent("connector-handler"),
	}
}

// CreateConnectorRequest represents the request body for creating a connector.
type CreateConnectorRequest struct {
	Name     string                 `json:"name"`
	Platform string                 `json:"platform"`
	Config   map[string]interface{} `json:"config"`
}

// Create creates a new connector.
// POST /api/v1/connectors
func (h *ConnectorHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	var req CreateConnectorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.Platform == "" {
		http.Error(w, "platform is required", http.StatusBadRequest)
		return
	}
	if req.Config == nil {
		http.Error(w, "config is required", http.StatusBadRequest)
		return
	}

	connector, err := h.svc.Create(ctx, service.CreateConnectorParams{
		OrgID:    org.ID,
		Name:     req.Name,
		Platform: req.Platform,
		Config:   req.Config,
	})
	if err != nil {
		h.log.Error("failed to create connector", "error", err, "org_id", org.ID)
		// Check for specific error types
		if err.Error() == "connector with name '"+req.Name+"' already exists" {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, connector)
}

// List lists all connectors for the organization.
// GET /api/v1/connectors
func (h *ConnectorHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	connectors, err := h.svc.List(ctx, org.ID)
	if err != nil {
		h.log.Error("failed to list connectors", "error", err, "org_id", org.ID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"connectors": connectors,
		"total":      len(connectors),
	})
}

// Get retrieves a connector by ID.
// GET /api/v1/connectors/{id}
func (h *ConnectorHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid connector id", http.StatusBadRequest)
		return
	}

	connector, err := h.svc.Get(ctx, id, org.ID)
	if err != nil {
		h.log.Error("failed to get connector", "error", err, "connector_id", id, "org_id", org.ID)
		http.Error(w, "connector not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, connector)
}

// Delete deletes a connector.
// DELETE /api/v1/connectors/{id}
func (h *ConnectorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid connector id", http.StatusBadRequest)
		return
	}

	if err := h.svc.Delete(ctx, id, org.ID); err != nil {
		h.log.Error("failed to delete connector", "error", err, "connector_id", id, "org_id", org.ID)
		http.Error(w, "connector not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TestConnection tests the connector's connection to the cloud platform.
// POST /api/v1/connectors/{id}/test
func (h *ConnectorHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid connector id", http.StatusBadRequest)
		return
	}

	result, err := h.svc.TestConnection(ctx, id, org.ID)
	if err != nil {
		h.log.Error("failed to test connector", "error", err, "connector_id", id, "org_id", org.ID)
		http.Error(w, "connector not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// TriggerSync triggers asset discovery for the connector.
// POST /api/v1/connectors/{id}/sync
func (h *ConnectorHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid connector id", http.StatusBadRequest)
		return
	}

	result, err := h.svc.TriggerSync(ctx, id, org.ID)
	if err != nil {
		h.log.Error("failed to trigger sync", "error", err, "connector_id", id, "org_id", org.ID)
		if err.Error() == "connector is disabled" {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "connector not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Enable enables a connector.
// POST /api/v1/connectors/{id}/enable
func (h *ConnectorHandler) Enable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid connector id", http.StatusBadRequest)
		return
	}

	if err := h.svc.Enable(ctx, id, org.ID); err != nil {
		h.log.Error("failed to enable connector", "error", err, "connector_id", id, "org_id", org.ID)
		http.Error(w, "connector not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "enabled"})
}

// Disable disables a connector.
// POST /api/v1/connectors/{id}/disable
func (h *ConnectorHandler) Disable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid connector id", http.StatusBadRequest)
		return
	}

	if err := h.svc.Disable(ctx, id, org.ID); err != nil {
		h.log.Error("failed to disable connector", "error", err, "connector_id", id, "org_id", org.ID)
		http.Error(w, "connector not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "disabled"})
}
