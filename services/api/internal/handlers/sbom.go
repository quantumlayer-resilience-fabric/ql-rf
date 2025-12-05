package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/sbom"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
)

// SBOMHandler handles SBOM-related HTTP requests.
type SBOMHandler struct {
	svc       *sbom.Service
	generator *sbom.Generator
	log       *logger.Logger
}

// NewSBOMHandler creates a new SBOM handler.
func NewSBOMHandler(svc *sbom.Service, generator *sbom.Generator, log *logger.Logger) *SBOMHandler {
	return &SBOMHandler{
		svc:       svc,
		generator: generator,
		log:       log.WithComponent("sbom-handler"),
	}
}

// GetImageSBOM retrieves the most recent SBOM for an image.
// GET /api/v1/images/{id}/sbom
func (h *SBOMHandler) GetImageSBOM(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	imageID := chi.URLParam(r, "id")
	id, err := uuid.Parse(imageID)
	if err != nil {
		http.Error(w, "invalid image ID", http.StatusBadRequest)
		return
	}

	// Get SBOM
	sbomDoc, err := h.svc.GetByImageID(ctx, id)
	if err != nil {
		if errors.Is(err, errors.New("no sbom found for image")) {
			http.Error(w, "SBOM not found for this image", http.StatusNotFound)
			return
		}
		h.log.Error("failed to get sbom", "error", err, "image_id", id)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Verify org ownership
	if sbomDoc.OrgID != org.ID {
		http.Error(w, "SBOM not found", http.StatusNotFound)
		return
	}

	// Optionally load packages and vulnerabilities
	includePackages := r.URL.Query().Get("include_packages") == "true"
	includeVulns := r.URL.Query().Get("include_vulns") == "true"

	if includePackages {
		packages, err := h.svc.GetPackages(ctx, sbomDoc.ID)
		if err != nil {
			h.log.Warn("failed to load packages", "error", err)
		} else {
			sbomDoc.Packages = packages
		}
	}

	if includeVulns {
		vulns, err := h.svc.GetVulnerabilities(ctx, sbomDoc.ID, nil)
		if err != nil {
			h.log.Warn("failed to load vulnerabilities", "error", err)
		} else {
			sbomDoc.Vulnerabilities = vulns
		}
	}

	writeJSON(w, http.StatusOK, sbomDoc)
}

// GenerateSBOM generates a new SBOM for an image.
// POST /api/v1/images/{id}/sbom/generate
func (h *SBOMHandler) GenerateSBOM(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	imageID := chi.URLParam(r, "id")
	id, err := uuid.Parse(imageID)
	if err != nil {
		http.Error(w, "invalid image ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req struct {
		Format       string            `json:"format"` // spdx or cyclonedx
		Scanner      string            `json:"scanner,omitempty"`
		IncludeVulns bool              `json:"include_vulns"`
		Dockerfile   string            `json:"dockerfile,omitempty"`
		Manifests    map[string]string `json:"manifests,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate format
	format := sbom.Format(req.Format)
	if !format.IsValid() {
		http.Error(w, "invalid format: must be 'spdx' or 'cyclonedx'", http.StatusBadRequest)
		return
	}

	// Set default scanner
	if req.Scanner == "" {
		req.Scanner = "ql-rf"
	}

	// Generate SBOM
	result, err := h.generator.Generate(ctx, sbom.GenerateRequest{
		ImageID:      id,
		OrgID:        org.ID,
		Format:       format,
		Scanner:      req.Scanner,
		Dockerfile:   req.Dockerfile,
		Manifests:    req.Manifests,
		IncludeVulns: req.IncludeVulns,
	})
	if err != nil {
		h.log.Error("failed to generate sbom",
			"error", err,
			"image_id", id,
		)
		http.Error(w, "failed to generate SBOM", http.StatusInternalServerError)
		return
	}

	// Enrich with vulnerabilities if requested
	if req.IncludeVulns {
		if err := h.generator.EnrichWithVulnerabilities(ctx, result.SBOM.ID); err != nil {
			h.log.Warn("failed to enrich with vulnerabilities",
				"error", err,
				"sbom_id", result.SBOM.ID,
			)
		} else {
			// Reload vulnerability count
			vulns, _ := h.svc.GetVulnerabilities(ctx, result.SBOM.ID, nil)
			result.VulnCount = len(vulns)
		}
	}

	h.log.Info("sbom generated",
		"sbom_id", result.SBOM.ID,
		"image_id", id,
		"format", format,
		"packages", result.PackageCount,
		"vulnerabilities", result.VulnCount,
	)

	// Build response
	response := sbom.SBOMGenerationResponse{
		SBOM:         result.SBOM,
		Status:       result.Status,
		Message:      result.Message,
		PackageCount: result.PackageCount,
		VulnCount:    result.VulnCount,
		GeneratedAt:  result.SBOM.GeneratedAt,
	}

	writeJSON(w, http.StatusCreated, response)
}

// GetSBOMVulnerabilities retrieves vulnerabilities for an SBOM.
// GET /api/v1/sbom/{id}/vulnerabilities
func (h *SBOMHandler) GetSBOMVulnerabilities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	sbomID := chi.URLParam(r, "id")
	id, err := uuid.Parse(sbomID)
	if err != nil {
		http.Error(w, "invalid SBOM ID", http.StatusBadRequest)
		return
	}

	// Verify SBOM exists and belongs to org
	sbomDoc, err := h.svc.Get(ctx, id)
	if err != nil {
		http.Error(w, "SBOM not found", http.StatusNotFound)
		return
	}
	if sbomDoc.OrgID != org.ID {
		http.Error(w, "SBOM not found", http.StatusNotFound)
		return
	}

	// Parse filters
	filter := &sbom.VulnerabilityFilter{
		SBOMID: id,
	}

	// Severity filter
	if severities := r.URL.Query()["severity"]; len(severities) > 0 {
		filter.Severities = severities
	}

	// Min CVSS filter
	if minCVSS := r.URL.Query().Get("min_cvss"); minCVSS != "" {
		if score, err := strconv.ParseFloat(minCVSS, 64); err == nil {
			filter.MinCVSS = &score
		}
	}

	// Has exploit filter
	if hasExploit := r.URL.Query().Get("has_exploit"); hasExploit != "" {
		val := hasExploit == "true"
		filter.HasExploit = &val
	}

	// Fix available filter
	if fixAvailable := r.URL.Query().Get("fix_available"); fixAvailable != "" {
		val := fixAvailable == "true"
		filter.FixAvailable = &val
	}

	// Get vulnerabilities
	vulns, err := h.svc.GetVulnerabilities(ctx, id, filter)
	if err != nil {
		h.log.Error("failed to get vulnerabilities",
			"error", err,
			"sbom_id", id,
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Get vulnerability statistics
	stats, err := h.svc.GetVulnerabilityStats(ctx, id)
	if err != nil {
		h.log.Warn("failed to get vulnerability stats", "error", err)
		stats = map[string]interface{}{}
	}

	response := map[string]interface{}{
		"sbom_id":         id,
		"vulnerabilities": vulns,
		"count":           len(vulns),
		"stats":           stats,
	}

	writeJSON(w, http.StatusOK, response)
}

// ExportSBOM exports an SBOM in a specific format.
// GET /api/v1/sbom/{id}/export?format=spdx|cyclonedx
func (h *SBOMHandler) ExportSBOM(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	sbomID := chi.URLParam(r, "id")
	id, err := uuid.Parse(sbomID)
	if err != nil {
		http.Error(w, "invalid SBOM ID", http.StatusBadRequest)
		return
	}

	// Verify SBOM exists and belongs to org
	sbomDoc, err := h.svc.Get(ctx, id)
	if err != nil {
		http.Error(w, "SBOM not found", http.StatusNotFound)
		return
	}
	if sbomDoc.OrgID != org.ID {
		http.Error(w, "SBOM not found", http.StatusNotFound)
		return
	}

	// Get requested format
	formatParam := r.URL.Query().Get("format")
	if formatParam == "" {
		formatParam = string(sbomDoc.Format) // Use original format
	}

	format := sbom.Format(formatParam)
	if !format.IsValid() {
		http.Error(w, "invalid format: must be 'spdx' or 'cyclonedx'", http.StatusBadRequest)
		return
	}

	// Export in requested format
	var content map[string]interface{}
	switch format {
	case sbom.FormatSPDX:
		content, err = h.svc.ExportSPDX(ctx, sbomID)
	case sbom.FormatCycloneDX:
		content, err = h.svc.ExportCycloneDX(ctx, sbomID)
	default:
		http.Error(w, "unsupported format", http.StatusBadRequest)
		return
	}

	if err != nil {
		h.log.Error("failed to export sbom",
			"error", err,
			"sbom_id", id,
			"format", format,
		)
		http.Error(w, "failed to export SBOM", http.StatusInternalServerError)
		return
	}

	response := sbom.SBOMExportResponse{
		Format:  format,
		Content: content,
	}

	writeJSON(w, http.StatusOK, response)
}

// ListSBOMs lists all SBOMs for an organization.
// GET /api/v1/sbom
func (h *SBOMHandler) ListSBOMs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	// Parse pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// List SBOMs
	result, err := h.svc.List(ctx, org.ID, page, pageSize)
	if err != nil {
		h.log.Error("failed to list sboms",
			"error", err,
			"org_id", org.ID,
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetSBOM retrieves a specific SBOM by ID.
// GET /api/v1/sbom/{id}
func (h *SBOMHandler) GetSBOM(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	sbomID := chi.URLParam(r, "id")
	id, err := uuid.Parse(sbomID)
	if err != nil {
		http.Error(w, "invalid SBOM ID", http.StatusBadRequest)
		return
	}

	// Get SBOM
	sbomDoc, err := h.svc.Get(ctx, id)
	if err != nil {
		http.Error(w, "SBOM not found", http.StatusNotFound)
		return
	}

	// Verify org ownership
	if sbomDoc.OrgID != org.ID {
		http.Error(w, "SBOM not found", http.StatusNotFound)
		return
	}

	// Optionally load packages and vulnerabilities
	includePackages := r.URL.Query().Get("include_packages") == "true"
	includeVulns := r.URL.Query().Get("include_vulns") == "true"

	if includePackages {
		packages, err := h.svc.GetPackages(ctx, sbomDoc.ID)
		if err != nil {
			h.log.Warn("failed to load packages", "error", err)
		} else {
			sbomDoc.Packages = packages
		}
	}

	if includeVulns {
		vulns, err := h.svc.GetVulnerabilities(ctx, sbomDoc.ID, nil)
		if err != nil {
			h.log.Warn("failed to load vulnerabilities", "error", err)
		} else {
			sbomDoc.Vulnerabilities = vulns
		}
	}

	writeJSON(w, http.StatusOK, sbomDoc)
}

// DeleteSBOM deletes an SBOM.
// DELETE /api/v1/sbom/{id}
func (h *SBOMHandler) DeleteSBOM(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	sbomID := chi.URLParam(r, "id")
	id, err := uuid.Parse(sbomID)
	if err != nil {
		http.Error(w, "invalid SBOM ID", http.StatusBadRequest)
		return
	}

	// Verify SBOM exists and belongs to org
	sbomDoc, err := h.svc.Get(ctx, id)
	if err != nil {
		http.Error(w, "SBOM not found", http.StatusNotFound)
		return
	}
	if sbomDoc.OrgID != org.ID {
		http.Error(w, "SBOM not found", http.StatusNotFound)
		return
	}

	// Delete SBOM
	if err := h.svc.Delete(ctx, id); err != nil {
		h.log.Error("failed to delete sbom",
			"error", err,
			"sbom_id", id,
		)
		http.Error(w, "failed to delete SBOM", http.StatusInternalServerError)
		return
	}

	h.log.Info("sbom deleted",
		"sbom_id", id,
		"image_id", sbomDoc.ImageID,
	)

	w.WriteHeader(http.StatusNoContent)
}
