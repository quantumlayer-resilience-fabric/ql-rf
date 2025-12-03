package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
)

// ImageHandler handles image-related requests.
type ImageHandler struct {
	svc *service.ImageService
	log *logger.Logger
}

// NewImageHandler creates a new ImageHandler.
func NewImageHandler(svc *service.ImageService, log *logger.Logger) *ImageHandler {
	return &ImageHandler{
		svc: svc,
		log: log.WithComponent("image-handler"),
	}
}

// List returns a paginated list of images.
func (h *ImageHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Call service
	result, err := h.svc.ListImages(ctx, service.ListImagesInput{
		OrgID:    org.ID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.log.Error("failed to list images", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	var responseImages []models.Image
	for _, img := range result.Images {
		responseImages = append(responseImages, serviceImageToModel(img))
	}

	response := models.ImageListResponse{
		Images:     responseImages,
		Total:      int(result.Total),
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	}

	writeJSON(w, http.StatusOK, response)
}

// Get returns a single image by ID.
func (h *ImageHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	// Call service
	img, err := h.svc.GetImage(ctx, service.GetImageInput{
		ID:    id,
		OrgID: org.ID,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.Error(w, "image not found", http.StatusNotFound)
			return
		}
		h.log.Error("failed to get image", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, serviceImageToModel(*img))
}

// GetLatest returns the latest version of an image family.
func (h *ImageHandler) GetLatest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	family := chi.URLParam(r, "family")
	if family == "" {
		http.Error(w, "family is required", http.StatusBadRequest)
		return
	}

	// Call service
	img, err := h.svc.GetLatestImage(ctx, service.GetLatestImageInput{
		OrgID:  org.ID,
		Family: family,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.Error(w, "no production image found for family", http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrInvalidInput) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.log.Error("failed to get latest image", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, serviceImageToModel(*img))
}

// Create creates a new image.
func (h *ImageHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	var req models.CreateImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Build input
	input := service.CreateImageInput{
		OrgID:   org.ID,
		Family:  req.Family,
		Version: req.Version,
		Signed:  req.Signed,
	}
	if req.OSName != "" {
		input.OSName = req.OSName
	}
	if req.OSVersion != "" {
		input.OSVersion = req.OSVersion
	}
	if req.CISLevel > 0 {
		input.CISLevel = req.CISLevel
	}

	// Call service
	img, err := h.svc.CreateImage(ctx, input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.log.Error("failed to create image", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.log.Info("image created",
		"image_id", img.ID,
		"family", img.Family,
		"version", img.Version,
	)

	writeJSON(w, http.StatusCreated, serviceImageToModel(*img))
}

// Update updates an existing image.
func (h *ImageHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	var req models.UpdateImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Build update params
	var updateParams service.UpdateImageParams
	if req.SBOMUrl != nil {
		updateParams.SBOMUrl = req.SBOMUrl
	}
	if req.Signed != nil {
		updateParams.Signed = req.Signed
	}
	if req.Status != nil {
		status := string(*req.Status)
		updateParams.Status = &status
	}

	// Update the image
	updated, err := h.svc.UpdateImage(ctx, service.UpdateImageInput{
		ID:     id,
		OrgID:  org.ID,
		Params: updateParams,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.Error(w, "image not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrInvalidInput) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.log.Error("failed to update image", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, serviceImageToModel(*updated))
}

// Delete deletes an image (soft-delete by setting status to deprecated).
func (h *ImageHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	// Soft delete the image (sets status to deprecated)
	err = h.svc.DeleteImage(ctx, service.DeleteImageInput{
		ID:    id,
		OrgID: org.ID,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.Error(w, "image not found", http.StatusNotFound)
			return
		}
		h.log.Error("failed to delete image", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Promote promotes an image to a new status.
func (h *ImageHandler) Promote(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Call service
	img, err := h.svc.PromoteImage(ctx, service.PromoteImageInput{
		ID:       id,
		OrgID:    org.ID,
		ToStatus: req.Status,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.Error(w, "image not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrInvalidInput) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.log.Error("failed to promote image", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.log.Info("image promoted",
		"image_id", img.ID,
		"status", img.Status,
	)

	writeJSON(w, http.StatusOK, serviceImageToModel(*img))
}

// AddCoordinate adds a platform coordinate to an image.
func (h *ImageHandler) AddCoordinate(w http.ResponseWriter, r *http.Request) {
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

	var req models.AddCoordinateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Call service
	coord, err := h.svc.AddCoordinate(ctx, service.AddCoordinateInput{
		ImageID:    id,
		OrgID:      org.ID,
		Platform:   string(req.Platform),
		Region:     req.Region,
		Identifier: req.Identifier,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.Error(w, "image not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrInvalidInput) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.log.Error("failed to add coordinate", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.log.Info("coordinate added",
		"image_id", id,
		"platform", coord.Platform,
		"identifier", coord.Identifier,
	)

	writeJSON(w, http.StatusCreated, serviceCoordinateToModel(*coord))
}

// Helper functions to convert between service and model types
func serviceImageToModel(img service.Image) models.Image {
	result := models.Image{
		ID:        img.ID,
		OrgID:     img.OrgID,
		Family:    img.Family,
		Version:   img.Version,
		Signed:    img.Signed,
		Status:    models.ImageStatus(img.Status),
		CreatedAt: img.CreatedAt,
		UpdatedAt: img.UpdatedAt,
	}
	if img.OSName != nil {
		result.OSName = *img.OSName
	}
	if img.OSVersion != nil {
		result.OSVersion = *img.OSVersion
	}
	if img.CISLevel != nil {
		result.CISLevel = *img.CISLevel
	}
	if img.SBOMUrl != nil {
		result.SBOMUrl = *img.SBOMUrl
	}
	if len(img.Coordinates) > 0 {
		result.Coordinates = make([]models.ImageCoordinate, 0, len(img.Coordinates))
		for _, c := range img.Coordinates {
			result.Coordinates = append(result.Coordinates, serviceCoordinateToModel(c))
		}
	}
	return result
}

func serviceCoordinateToModel(coord service.ImageCoordinate) models.ImageCoordinate {
	result := models.ImageCoordinate{
		ID:         coord.ID,
		ImageID:    coord.ImageID,
		Platform:   models.Platform(coord.Platform),
		Identifier: coord.Identifier,
		CreatedAt:  coord.CreatedAt,
	}
	if coord.Region != nil {
		result.Region = *coord.Region
	}
	return result
}
