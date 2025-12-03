// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
)

// MockImageRepository is a mock implementation of ImageRepository.
type MockImageRepository struct {
	mu          sync.RWMutex
	images      map[uuid.UUID]*service.Image
	coordinates map[uuid.UUID][]service.ImageCoordinate

	// Control behavior for testing
	GetImageFunc                  func(ctx context.Context, id uuid.UUID) (*service.Image, error)
	GetLatestImageByFamilyFunc    func(ctx context.Context, orgID uuid.UUID, family string) (*service.Image, error)
	ListImagesFunc                func(ctx context.Context, params service.ListImagesParams) ([]service.Image, error)
	CreateImageFunc               func(ctx context.Context, params service.CreateImageParams) (*service.Image, error)
	UpdateImageFunc               func(ctx context.Context, id uuid.UUID, params service.UpdateImageParams) (*service.Image, error)
	UpdateImageStatusFunc         func(ctx context.Context, id uuid.UUID, status string) (*service.Image, error)
	CountImagesByOrgFunc          func(ctx context.Context, orgID uuid.UUID) (int64, error)
	GetImageCoordinatesFunc       func(ctx context.Context, imageID uuid.UUID) ([]service.ImageCoordinate, error)
	CreateImageCoordinateFunc     func(ctx context.Context, params service.CreateImageCoordinateParams) (*service.ImageCoordinate, error)
}

// NewMockImageRepository creates a new MockImageRepository.
func NewMockImageRepository() *MockImageRepository {
	return &MockImageRepository{
		images:      make(map[uuid.UUID]*service.Image),
		coordinates: make(map[uuid.UUID][]service.ImageCoordinate),
	}
}

// GetImage returns an image by ID.
func (m *MockImageRepository) GetImage(ctx context.Context, id uuid.UUID) (*service.Image, error) {
	if m.GetImageFunc != nil {
		return m.GetImageFunc(ctx, id)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if img, ok := m.images[id]; ok {
		return img, nil
	}
	return nil, service.ErrNotFound
}

// GetLatestImageByFamily returns the latest production image for a family.
func (m *MockImageRepository) GetLatestImageByFamily(ctx context.Context, orgID uuid.UUID, family string) (*service.Image, error) {
	if m.GetLatestImageByFamilyFunc != nil {
		return m.GetLatestImageByFamilyFunc(ctx, orgID, family)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var latest *service.Image
	for _, img := range m.images {
		if img.OrgID == orgID && img.Family == family && img.Status == "production" {
			if latest == nil || img.CreatedAt.After(latest.CreatedAt) {
				latest = img
			}
		}
	}

	if latest == nil {
		return nil, service.ErrNotFound
	}
	return latest, nil
}

// ListImages returns a list of images.
func (m *MockImageRepository) ListImages(ctx context.Context, params service.ListImagesParams) ([]service.Image, error) {
	if m.ListImagesFunc != nil {
		return m.ListImagesFunc(ctx, params)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []service.Image
	for _, img := range m.images {
		if img.OrgID != params.OrgID {
			continue
		}
		if params.Family != nil && img.Family != *params.Family {
			continue
		}
		if params.Status != nil && img.Status != *params.Status {
			continue
		}
		result = append(result, *img)
	}

	// Apply pagination
	start := int(params.Offset)
	if start >= len(result) {
		return []service.Image{}, nil
	}
	end := start + int(params.Limit)
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
}

// CreateImage creates a new image.
func (m *MockImageRepository) CreateImage(ctx context.Context, params service.CreateImageParams) (*service.Image, error) {
	if m.CreateImageFunc != nil {
		return m.CreateImageFunc(ctx, params)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	img := &service.Image{
		ID:        uuid.New(),
		OrgID:     params.OrgID,
		Family:    params.Family,
		Version:   params.Version,
		OSName:    params.OSName,
		OSVersion: params.OSVersion,
		CISLevel:  params.CISLevel,
		SBOMUrl:   params.SBOMUrl,
		Signed:    params.Signed,
		Status:    params.Status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	m.images[img.ID] = img
	return img, nil
}

// UpdateImageStatus updates an image's status.
func (m *MockImageRepository) UpdateImageStatus(ctx context.Context, id uuid.UUID, status string) (*service.Image, error) {
	if m.UpdateImageStatusFunc != nil {
		return m.UpdateImageStatusFunc(ctx, id, status)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	img, ok := m.images[id]
	if !ok {
		return nil, service.ErrNotFound
	}

	img.Status = status
	img.UpdatedAt = time.Now()
	return img, nil
}

// UpdateImage updates an image's metadata.
func (m *MockImageRepository) UpdateImage(ctx context.Context, id uuid.UUID, params service.UpdateImageParams) (*service.Image, error) {
	if m.UpdateImageFunc != nil {
		return m.UpdateImageFunc(ctx, id, params)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	img, ok := m.images[id]
	if !ok {
		return nil, service.ErrNotFound
	}

	if params.Version != nil {
		img.Version = *params.Version
	}
	if params.OSName != nil {
		img.OSName = params.OSName
	}
	if params.OSVersion != nil {
		img.OSVersion = params.OSVersion
	}
	if params.CISLevel != nil {
		img.CISLevel = params.CISLevel
	}
	if params.SBOMUrl != nil {
		img.SBOMUrl = params.SBOMUrl
	}
	if params.Signed != nil {
		img.Signed = *params.Signed
	}
	if params.Status != nil {
		img.Status = *params.Status
	}
	img.UpdatedAt = time.Now()
	return img, nil
}

// CountImagesByOrg counts images for an organization.
func (m *MockImageRepository) CountImagesByOrg(ctx context.Context, orgID uuid.UUID) (int64, error) {
	if m.CountImagesByOrgFunc != nil {
		return m.CountImagesByOrgFunc(ctx, orgID)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var count int64
	for _, img := range m.images {
		if img.OrgID == orgID {
			count++
		}
	}
	return count, nil
}

// GetImageCoordinates returns coordinates for an image.
func (m *MockImageRepository) GetImageCoordinates(ctx context.Context, imageID uuid.UUID) ([]service.ImageCoordinate, error) {
	if m.GetImageCoordinatesFunc != nil {
		return m.GetImageCoordinatesFunc(ctx, imageID)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if coords, ok := m.coordinates[imageID]; ok {
		return coords, nil
	}
	return []service.ImageCoordinate{}, nil
}

// CreateImageCoordinate creates a new image coordinate.
func (m *MockImageRepository) CreateImageCoordinate(ctx context.Context, params service.CreateImageCoordinateParams) (*service.ImageCoordinate, error) {
	if m.CreateImageCoordinateFunc != nil {
		return m.CreateImageCoordinateFunc(ctx, params)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	coord := &service.ImageCoordinate{
		ID:         uuid.New(),
		ImageID:    params.ImageID,
		Platform:   params.Platform,
		Region:     params.Region,
		Identifier: params.Identifier,
		CreatedAt:  time.Now(),
	}

	m.coordinates[params.ImageID] = append(m.coordinates[params.ImageID], *coord)
	return coord, nil
}

// AddImage adds an image directly to the mock (for test setup).
func (m *MockImageRepository) AddImage(img *service.Image) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.images[img.ID] = img
}

// Reset clears all data from the mock.
func (m *MockImageRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.images = make(map[uuid.UUID]*service.Image)
	m.coordinates = make(map[uuid.UUID][]service.ImageCoordinate)
}
