package models

import (
	"time"

	"github.com/google/uuid"
)

// Image represents a golden image in the registry.
type Image struct {
	ID        uuid.UUID `json:"id" db:"id"`
	OrgID     uuid.UUID `json:"org_id" db:"org_id"`
	Family    string    `json:"family" db:"family"`       // e.g., "ql-base-linux"
	Version   string    `json:"version" db:"version"`     // e.g., "1.6.4"
	OSName    string    `json:"os_name" db:"os_name"`     // e.g., "ubuntu"
	OSVersion string    `json:"os_version" db:"os_version"` // e.g., "22.04"
	CISLevel  int       `json:"cis_level,omitempty" db:"cis_level"` // 1 or 2
	SBOMUrl   string    `json:"sbom_url,omitempty" db:"sbom_url"`
	Signed    bool      `json:"signed" db:"signed"`
	Status    ImageStatus `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Loaded relationships
	Coordinates []ImageCoordinate `json:"coordinates,omitempty"`
}

// ImageStatus represents the promotion status of an image.
type ImageStatus string

const (
	ImageStatusDraft     ImageStatus = "draft"
	ImageStatusDev       ImageStatus = "dev"
	ImageStatusStaging   ImageStatus = "staging"
	ImageStatusProduction ImageStatus = "production"
	ImageStatusDeprecated ImageStatus = "deprecated"
)

// IsValid checks if the status is valid.
func (s ImageStatus) IsValid() bool {
	switch s {
	case ImageStatusDraft, ImageStatusDev, ImageStatusStaging, ImageStatusProduction, ImageStatusDeprecated:
		return true
	default:
		return false
	}
}

// ImageCoordinate represents platform-specific image identifiers.
type ImageCoordinate struct {
	ID         uuid.UUID `json:"id" db:"id"`
	ImageID    uuid.UUID `json:"image_id" db:"image_id"`
	Platform   Platform  `json:"platform" db:"platform"`
	Region     string    `json:"region,omitempty" db:"region"`
	Identifier string    `json:"identifier" db:"identifier"` // ami-xxx, /subscriptions/.../versions/x
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// Platform represents a cloud or infrastructure platform.
type Platform string

const (
	PlatformAWS      Platform = "aws"
	PlatformAzure    Platform = "azure"
	PlatformGCP      Platform = "gcp"
	PlatformVSphere  Platform = "vsphere"
	PlatformK8s      Platform = "k8s"
	PlatformBareMetal Platform = "baremetal"
)

// IsValid checks if the platform is valid.
func (p Platform) IsValid() bool {
	switch p {
	case PlatformAWS, PlatformAzure, PlatformGCP, PlatformVSphere, PlatformK8s, PlatformBareMetal:
		return true
	default:
		return false
	}
}

// String returns the string representation of the platform.
func (p Platform) String() string {
	return string(p)
}

// CreateImageRequest represents a request to create a new image.
type CreateImageRequest struct {
	Family    string `json:"family" validate:"required,min=1,max=255"`
	Version   string `json:"version" validate:"required,min=1,max=63"`
	OSName    string `json:"os_name" validate:"required"`
	OSVersion string `json:"os_version" validate:"required"`
	CISLevel  int    `json:"cis_level,omitempty" validate:"omitempty,oneof=1 2"`
	SBOMUrl   string `json:"sbom_url,omitempty" validate:"omitempty,url"`
	Signed    bool   `json:"signed"`
}

// UpdateImageRequest represents a request to update an image.
type UpdateImageRequest struct {
	SBOMUrl *string      `json:"sbom_url,omitempty"`
	Signed  *bool        `json:"signed,omitempty"`
	Status  *ImageStatus `json:"status,omitempty"`
}

// AddCoordinateRequest represents a request to add a platform coordinate.
type AddCoordinateRequest struct {
	Platform   Platform `json:"platform" validate:"required"`
	Region     string   `json:"region,omitempty"`
	Identifier string   `json:"identifier" validate:"required"`
}

// ImageFilter represents filters for listing images.
type ImageFilter struct {
	Family   string      `json:"family,omitempty"`
	Status   ImageStatus `json:"status,omitempty"`
	Platform Platform    `json:"platform,omitempty"`
	Signed   *bool       `json:"signed,omitempty"`
}

// ImageListResponse represents a paginated list of images.
type ImageListResponse struct {
	Images     []Image `json:"images"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	TotalPages int     `json:"total_pages"`
}
