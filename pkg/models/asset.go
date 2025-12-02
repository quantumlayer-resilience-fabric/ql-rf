package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Asset represents a discovered infrastructure asset.
type Asset struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	OrgID        uuid.UUID       `json:"org_id" db:"org_id"`
	EnvID        uuid.UUID       `json:"env_id,omitempty" db:"env_id"`
	Platform     Platform        `json:"platform" db:"platform"`
	Account      string          `json:"account,omitempty" db:"account"`   // AWS account ID, Azure subscription, etc.
	Region       string          `json:"region,omitempty" db:"region"`
	Site         string          `json:"site,omitempty" db:"site"`         // Logical site name (e.g., "dc-london")
	InstanceID   string          `json:"instance_id" db:"instance_id"`     // Platform-specific ID
	Name         string          `json:"name,omitempty" db:"name"`
	ImageRef     string          `json:"image_ref,omitempty" db:"image_ref"` // AMI ID, template name, etc.
	ImageVersion string          `json:"image_version,omitempty" db:"image_version"`
	State        AssetState      `json:"state" db:"state"`
	Tags         json.RawMessage `json:"tags,omitempty" db:"tags"`
	DiscoveredAt time.Time       `json:"discovered_at" db:"discovered_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
}

// AssetState represents the running state of an asset.
type AssetState string

const (
	AssetStateRunning    AssetState = "running"
	AssetStateStopped    AssetState = "stopped"
	AssetStateTerminated AssetState = "terminated"
	AssetStatePending    AssetState = "pending"
	AssetStateUnknown    AssetState = "unknown"
)

// IsActive checks if the asset is in an active state.
func (s AssetState) IsActive() bool {
	return s == AssetStateRunning || s == AssetStatePending
}

// GetTags returns the tags as a map.
func (a *Asset) GetTags() map[string]string {
	if a.Tags == nil {
		return nil
	}
	var tags map[string]string
	if err := json.Unmarshal(a.Tags, &tags); err != nil {
		return nil
	}
	return tags
}

// SetTags sets the tags from a map.
func (a *Asset) SetTags(tags map[string]string) error {
	data, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	a.Tags = data
	return nil
}

// AssetFilter represents filters for listing assets.
type AssetFilter struct {
	Platform  Platform   `json:"platform,omitempty"`
	EnvID     uuid.UUID  `json:"env_id,omitempty"`
	Region    string     `json:"region,omitempty"`
	Site      string     `json:"site,omitempty"`
	State     AssetState `json:"state,omitempty"`
	ImageRef  string     `json:"image_ref,omitempty"`
	Compliant *bool      `json:"compliant,omitempty"` // Filter by compliance status
}

// AssetListResponse represents a paginated list of assets.
type AssetListResponse struct {
	Assets     []Asset `json:"assets"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	TotalPages int     `json:"total_pages"`
}

// AssetSummary represents a summary of assets by platform/region.
type AssetSummary struct {
	Platform  Platform `json:"platform"`
	Region    string   `json:"region,omitempty"`
	Site      string   `json:"site,omitempty"`
	Total     int      `json:"total"`
	Running   int      `json:"running"`
	Stopped   int      `json:"stopped"`
	Compliant int      `json:"compliant"`
}

// AssetDiscoveredEvent is published when a new asset is discovered or updated.
type AssetDiscoveredEvent struct {
	Asset     Asset     `json:"asset"`
	Action    string    `json:"action"` // "created", "updated", "deleted"
	Timestamp time.Time `json:"timestamp"`
}

// NormalizedAsset is the intermediate representation during normalization.
type NormalizedAsset struct {
	Platform     Platform          `json:"platform"`
	Account      string            `json:"account"`
	Region       string            `json:"region"`
	InstanceID   string            `json:"instance_id"`
	Name         string            `json:"name"`
	ImageRef     string            `json:"image_ref"`
	ImageVersion string            `json:"image_version"`
	State        AssetState        `json:"state"`
	Tags         map[string]string `json:"tags"`
}

// ToAsset converts a NormalizedAsset to an Asset.
func (n *NormalizedAsset) ToAsset(orgID uuid.UUID) (*Asset, error) {
	asset := &Asset{
		ID:           uuid.New(),
		OrgID:        orgID,
		Platform:     n.Platform,
		Account:      n.Account,
		Region:       n.Region,
		InstanceID:   n.InstanceID,
		Name:         n.Name,
		ImageRef:     n.ImageRef,
		ImageVersion: n.ImageVersion,
		State:        n.State,
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}

	if n.Tags != nil {
		if err := asset.SetTags(n.Tags); err != nil {
			return nil, err
		}
	}

	// Extract site from tags if available
	if site, ok := n.Tags["Site"]; ok {
		asset.Site = site
	} else if site, ok := n.Tags["site"]; ok {
		asset.Site = site
	}

	return asset, nil
}
