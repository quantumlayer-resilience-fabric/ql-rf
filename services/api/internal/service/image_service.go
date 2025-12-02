package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Common errors
var (
	ErrNotFound      = errors.New("not found")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrInvalidInput  = errors.New("invalid input")
	ErrAlreadyExists = errors.New("already exists")
)

// ImageService handles image business logic.
type ImageService struct {
	repo ImageRepository
}

// NewImageService creates a new ImageService.
func NewImageService(repo ImageRepository) *ImageService {
	return &ImageService{repo: repo}
}

// GetImageInput contains input for getting an image.
type GetImageInput struct {
	ID    uuid.UUID
	OrgID uuid.UUID
}

// GetImage retrieves an image by ID with authorization check.
func (s *ImageService) GetImage(ctx context.Context, input GetImageInput) (*Image, error) {
	img, err := s.repo.GetImage(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("get image: %w", err)
	}

	// Authorization: verify org ownership
	if img.OrgID != input.OrgID {
		return nil, ErrNotFound
	}

	// Load coordinates
	coords, err := s.repo.GetImageCoordinates(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("get image coordinates: %w", err)
	}
	img.Coordinates = coords

	return img, nil
}

// GetLatestImageInput contains input for getting latest image.
type GetLatestImageInput struct {
	OrgID  uuid.UUID
	Family string
}

// GetLatestImage retrieves the latest production image for a family.
func (s *ImageService) GetLatestImage(ctx context.Context, input GetLatestImageInput) (*Image, error) {
	if input.Family == "" {
		return nil, fmt.Errorf("%w: family is required", ErrInvalidInput)
	}

	img, err := s.repo.GetLatestImageByFamily(ctx, input.OrgID, input.Family)
	if err != nil {
		return nil, fmt.Errorf("get latest image: %w", err)
	}

	// Load coordinates
	coords, err := s.repo.GetImageCoordinates(ctx, img.ID)
	if err != nil {
		return nil, fmt.Errorf("get image coordinates: %w", err)
	}
	img.Coordinates = coords

	return img, nil
}

// ListImagesInput contains input for listing images.
type ListImagesInput struct {
	OrgID    uuid.UUID
	Family   *string
	Status   *string
	Page     int
	PageSize int
}

// ListImagesOutput contains output for listing images.
type ListImagesOutput struct {
	Images     []Image `json:"images"`
	Total      int64   `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	TotalPages int     `json:"total_pages"`
}

// ListImages retrieves a paginated list of images.
func (s *ImageService) ListImages(ctx context.Context, input ListImagesInput) (*ListImagesOutput, error) {
	// Apply defaults
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	offset := int32((input.Page - 1) * input.PageSize)

	images, err := s.repo.ListImages(ctx, ListImagesParams{
		OrgID:  input.OrgID,
		Family: input.Family,
		Status: input.Status,
		Limit:  int32(input.PageSize),
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list images: %w", err)
	}

	total, err := s.repo.CountImagesByOrg(ctx, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("count images: %w", err)
	}

	totalPages := int(total) / input.PageSize
	if int(total)%input.PageSize > 0 {
		totalPages++
	}

	return &ListImagesOutput{
		Images:     images,
		Total:      total,
		Page:       input.Page,
		PageSize:   input.PageSize,
		TotalPages: totalPages,
	}, nil
}

// CreateImageInput contains input for creating an image.
type CreateImageInput struct {
	OrgID     uuid.UUID
	Family    string
	Version   string
	OSName    string
	OSVersion string
	CISLevel  int
	SBOMUrl   string
	Signed    bool
}

// Validate validates the create image input.
func (i CreateImageInput) Validate() error {
	if i.Family == "" {
		return fmt.Errorf("%w: family is required", ErrInvalidInput)
	}
	if i.Version == "" {
		return fmt.Errorf("%w: version is required", ErrInvalidInput)
	}
	return nil
}

// CreateImage creates a new image.
func (s *ImageService) CreateImage(ctx context.Context, input CreateImageInput) (*Image, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	params := CreateImageParams{
		OrgID:   input.OrgID,
		Family:  input.Family,
		Version: input.Version,
		Signed:  input.Signed,
		Status:  "draft",
	}

	// Set optional fields
	if input.OSName != "" {
		params.OSName = &input.OSName
	}
	if input.OSVersion != "" {
		params.OSVersion = &input.OSVersion
	}
	if input.CISLevel > 0 {
		params.CISLevel = &input.CISLevel
	}
	if input.SBOMUrl != "" {
		params.SBOMUrl = &input.SBOMUrl
	}

	img, err := s.repo.CreateImage(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create image: %w", err)
	}

	return img, nil
}

// PromoteImageInput contains input for promoting an image.
type PromoteImageInput struct {
	ID       uuid.UUID
	OrgID    uuid.UUID
	ToStatus string
}

// PromoteImage promotes an image to a new status.
func (s *ImageService) PromoteImage(ctx context.Context, input PromoteImageInput) (*Image, error) {
	// Validate status transition
	validStatuses := map[string]bool{
		"draft":      true,
		"testing":    true,
		"production": true,
		"deprecated": true,
		"retired":    true,
	}
	if !validStatuses[input.ToStatus] {
		return nil, fmt.Errorf("%w: invalid status %s", ErrInvalidInput, input.ToStatus)
	}

	// Verify ownership
	img, err := s.repo.GetImage(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("get image: %w", err)
	}
	if img.OrgID != input.OrgID {
		return nil, ErrNotFound
	}

	// Update status
	updated, err := s.repo.UpdateImageStatus(ctx, input.ID, input.ToStatus)
	if err != nil {
		return nil, fmt.Errorf("update image status: %w", err)
	}

	return updated, nil
}

// AddCoordinateInput contains input for adding a coordinate.
type AddCoordinateInput struct {
	ImageID    uuid.UUID
	OrgID      uuid.UUID
	Platform   string
	Region     string
	Identifier string
}

// Validate validates the add coordinate input.
func (i AddCoordinateInput) Validate() error {
	validPlatforms := map[string]bool{
		"aws":     true,
		"azure":   true,
		"gcp":     true,
		"vsphere": true,
	}
	if !validPlatforms[i.Platform] {
		return fmt.Errorf("%w: invalid platform %s", ErrInvalidInput, i.Platform)
	}
	if i.Identifier == "" {
		return fmt.Errorf("%w: identifier is required", ErrInvalidInput)
	}
	return nil
}

// AddCoordinate adds a platform coordinate to an image.
func (s *ImageService) AddCoordinate(ctx context.Context, input AddCoordinateInput) (*ImageCoordinate, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Verify image ownership
	img, err := s.repo.GetImage(ctx, input.ImageID)
	if err != nil {
		return nil, fmt.Errorf("get image: %w", err)
	}
	if img.OrgID != input.OrgID {
		return nil, ErrNotFound
	}

	params := CreateImageCoordinateParams{
		ImageID:    input.ImageID,
		Platform:   input.Platform,
		Identifier: input.Identifier,
	}
	if input.Region != "" {
		params.Region = &input.Region
	}

	coord, err := s.repo.CreateImageCoordinate(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create coordinate: %w", err)
	}

	return coord, nil
}
