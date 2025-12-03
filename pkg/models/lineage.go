package models

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Image Lineage (Parent-Child Relationships)
// =============================================================================

// RelationshipType represents the type of relationship between images.
type RelationshipType string

const (
	RelationshipDerivedFrom  RelationshipType = "derived_from"  // New base image derived from parent
	RelationshipPatchedFrom  RelationshipType = "patched_from"  // Security patch applied
	RelationshipRebuiltFrom  RelationshipType = "rebuilt_from"  // Same spec, fresh build
)

// ImageLineage represents a parent-child relationship between images.
type ImageLineage struct {
	ID               uuid.UUID        `json:"id" db:"id"`
	ImageID          uuid.UUID        `json:"image_id" db:"image_id"`
	ParentImageID    uuid.UUID        `json:"parent_image_id" db:"parent_image_id"`
	RelationshipType RelationshipType `json:"relationship_type" db:"relationship_type"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`

	// Loaded relationships
	Image       *Image `json:"image,omitempty"`
	ParentImage *Image `json:"parent_image,omitempty"`
}

// =============================================================================
// Image Build Provenance
// =============================================================================

// BuildStatus represents the status of an image build.
type BuildStatus string

const (
	BuildStatusPending  BuildStatus = "pending"
	BuildStatusBuilding BuildStatus = "building"
	BuildStatusSuccess  BuildStatus = "success"
	BuildStatusFailed   BuildStatus = "failed"
)

// ImageBuild represents build provenance for an image (SLSA-compatible).
type ImageBuild struct {
	ID          uuid.UUID `json:"id" db:"id"`
	ImageID     uuid.UUID `json:"image_id" db:"image_id"`
	BuildNumber int       `json:"build_number" db:"build_number"`

	// Source
	SourceRepo   string `json:"source_repo,omitempty" db:"source_repo"`
	SourceCommit string `json:"source_commit,omitempty" db:"source_commit"`
	SourceBranch string `json:"source_branch,omitempty" db:"source_branch"`
	SourceTag    string `json:"source_tag,omitempty" db:"source_tag"`

	// Build system
	BuilderType    string            `json:"builder_type" db:"builder_type"` // packer, docker, etc.
	BuilderVersion string            `json:"builder_version,omitempty" db:"builder_version"`
	BuildTemplate  string            `json:"build_template,omitempty" db:"build_template"`
	BuildConfig    map[string]any    `json:"build_config,omitempty" db:"build_config"`

	// Build runner (CI)
	BuildRunner    string `json:"build_runner,omitempty" db:"build_runner"`
	BuildRunnerID  string `json:"build_runner_id,omitempty" db:"build_runner_id"`
	BuildRunnerURL string `json:"build_runner_url,omitempty" db:"build_runner_url"`

	// Artifacts
	BuildLogURL          string `json:"build_log_url,omitempty" db:"build_log_url"`
	BuildDurationSeconds int    `json:"build_duration_seconds,omitempty" db:"build_duration_seconds"`

	// Security
	BuiltBy        string `json:"built_by,omitempty" db:"built_by"`
	SignedBy       string `json:"signed_by,omitempty" db:"signed_by"`
	Signature      string `json:"signature,omitempty" db:"signature"`
	AttestationURL string `json:"attestation_url,omitempty" db:"attestation_url"`

	// Status
	Status       BuildStatus `json:"status" db:"status"`
	ErrorMessage string      `json:"error_message,omitempty" db:"error_message"`

	StartedAt   *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// =============================================================================
// Image Vulnerabilities
// =============================================================================

// VulnerabilitySeverity represents CVE severity levels.
type VulnerabilitySeverity string

const (
	SeverityCritical VulnerabilitySeverity = "critical"
	SeverityHigh     VulnerabilitySeverity = "high"
	SeverityMedium   VulnerabilitySeverity = "medium"
	SeverityLow      VulnerabilitySeverity = "low"
	SeverityUnknown  VulnerabilitySeverity = "unknown"
)

// VulnerabilityStatus represents the status of a vulnerability in an image.
type VulnerabilityStatus string

const (
	VulnStatusOpen        VulnerabilityStatus = "open"
	VulnStatusFixed       VulnerabilityStatus = "fixed"
	VulnStatusWontFix     VulnerabilityStatus = "wont_fix"
	VulnStatusFalsePositive VulnerabilityStatus = "false_positive"
)

// ImageVulnerability represents a CVE associated with an image.
type ImageVulnerability struct {
	ID      uuid.UUID `json:"id" db:"id"`
	ImageID uuid.UUID `json:"image_id" db:"image_id"`

	// CVE details
	CVEID      string                `json:"cve_id" db:"cve_id"`
	Severity   VulnerabilitySeverity `json:"severity" db:"severity"`
	CVSSScore  float64               `json:"cvss_score,omitempty" db:"cvss_score"`
	CVSSVector string                `json:"cvss_vector,omitempty" db:"cvss_vector"`

	// Affected package
	PackageName    string `json:"package_name,omitempty" db:"package_name"`
	PackageVersion string `json:"package_version,omitempty" db:"package_version"`
	PackageType    string `json:"package_type,omitempty" db:"package_type"`
	FixedVersion   string `json:"fixed_version,omitempty" db:"fixed_version"`

	// Status
	Status       VulnerabilityStatus `json:"status" db:"status"`
	StatusReason string              `json:"status_reason,omitempty" db:"status_reason"`

	// Scan info
	Scanner   string     `json:"scanner,omitempty" db:"scanner"`
	ScannedAt *time.Time `json:"scanned_at,omitempty" db:"scanned_at"`

	// Resolution
	FixedInImageID *uuid.UUID `json:"fixed_in_image_id,omitempty" db:"fixed_in_image_id"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	ResolvedBy     string     `json:"resolved_by,omitempty" db:"resolved_by"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// =============================================================================
// Image Deployments
// =============================================================================

// DeploymentStatus represents the status of an image deployment.
type DeploymentStatus string

const (
	DeploymentStatusActive     DeploymentStatus = "active"
	DeploymentStatusReplaced   DeploymentStatus = "replaced"
	DeploymentStatusTerminated DeploymentStatus = "terminated"
)

// ImageDeployment tracks where an image is deployed.
type ImageDeployment struct {
	ID      uuid.UUID `json:"id" db:"id"`
	ImageID uuid.UUID `json:"image_id" db:"image_id"`
	AssetID uuid.UUID `json:"asset_id" db:"asset_id"`

	DeployedAt       time.Time `json:"deployed_at" db:"deployed_at"`
	DeployedBy       string    `json:"deployed_by,omitempty" db:"deployed_by"`
	DeploymentMethod string    `json:"deployment_method,omitempty" db:"deployment_method"`

	Status            DeploymentStatus `json:"status" db:"status"`
	ReplacedAt        *time.Time       `json:"replaced_at,omitempty" db:"replaced_at"`
	ReplacedByImageID *uuid.UUID       `json:"replaced_by_image_id,omitempty" db:"replaced_by_image_id"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// Loaded relationships
	Image *Image `json:"image,omitempty"`
	Asset *Asset `json:"asset,omitempty"`
}

// =============================================================================
// Image Promotions
// =============================================================================

// ImagePromotion tracks status transitions (promotions/demotions).
type ImagePromotion struct {
	ID      uuid.UUID `json:"id" db:"id"`
	ImageID uuid.UUID `json:"image_id" db:"image_id"`

	FromStatus ImageStatus `json:"from_status" db:"from_status"`
	ToStatus   ImageStatus `json:"to_status" db:"to_status"`

	PromotedBy     string `json:"promoted_by" db:"promoted_by"`
	ApprovedBy     string `json:"approved_by,omitempty" db:"approved_by"`
	ApprovalTicket string `json:"approval_ticket,omitempty" db:"approval_ticket"`

	Reason string `json:"reason,omitempty" db:"reason"`

	ValidationPassed  *bool          `json:"validation_passed,omitempty" db:"validation_passed"`
	ValidationResults map[string]any `json:"validation_results,omitempty" db:"validation_results"`

	PromotedAt time.Time `json:"promoted_at" db:"promoted_at"`
}

// =============================================================================
// Image Components (SBOM)
// =============================================================================

// ComponentType represents the type of software component.
type ComponentType string

const (
	ComponentTypeOSPackage ComponentType = "os_package"
	ComponentTypeLibrary   ComponentType = "library"
	ComponentTypeBinary    ComponentType = "binary"
	ComponentTypeContainer ComponentType = "container"
)

// ImageComponent represents a software component in an image (SBOM).
type ImageComponent struct {
	ID      uuid.UUID `json:"id" db:"id"`
	ImageID uuid.UUID `json:"image_id" db:"image_id"`

	Name           string        `json:"name" db:"name"`
	Version        string        `json:"version" db:"version"`
	ComponentType  ComponentType `json:"component_type" db:"component_type"`
	PackageManager string        `json:"package_manager,omitempty" db:"package_manager"`

	License    string `json:"license,omitempty" db:"license"`
	LicenseURL string `json:"license_url,omitempty" db:"license_url"`

	SourceURL string `json:"source_url,omitempty" db:"source_url"`
	Checksum  string `json:"checksum,omitempty" db:"checksum"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// =============================================================================
// Image Tags
// =============================================================================

// ImageTag represents a key-value tag on an image.
type ImageTag struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ImageID   uuid.UUID `json:"image_id" db:"image_id"`
	Key       string    `json:"key" db:"key"`
	Value     string    `json:"value" db:"value"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// =============================================================================
// Lineage Tree Types (for API responses)
// =============================================================================

// LineageNode represents a node in the image lineage tree.
type LineageNode struct {
	Image    *Image         `json:"image"`
	Depth    int            `json:"depth"`
	Children []*LineageNode `json:"children,omitempty"`
	Parents  []*LineageNode `json:"parents,omitempty"`
}

// ImageLineageTree represents the full lineage tree for an image family.
type ImageLineageTree struct {
	Family string         `json:"family"`
	Roots  []*LineageNode `json:"roots"`
	Nodes  int            `json:"total_nodes"`
}

// =============================================================================
// Vulnerability Summary Types
// =============================================================================

// VulnerabilitySummary provides a count of vulnerabilities by severity.
type VulnerabilitySummary struct {
	ImageID       uuid.UUID  `json:"image_id"`
	Family        string     `json:"family"`
	Version       string     `json:"version"`
	CriticalOpen  int        `json:"critical_open"`
	HighOpen      int        `json:"high_open"`
	MediumOpen    int        `json:"medium_open"`
	LowOpen       int        `json:"low_open"`
	FixedCount    int        `json:"fixed_count"`
	LastScannedAt *time.Time `json:"last_scanned_at,omitempty"`
}

// =============================================================================
// API Request/Response Types
// =============================================================================

// CreateLineageRequest represents a request to create a lineage relationship.
type CreateLineageRequest struct {
	ParentImageID    uuid.UUID        `json:"parent_image_id" validate:"required"`
	RelationshipType RelationshipType `json:"relationship_type" validate:"required"`
}

// CreateBuildRequest represents a request to create a build record.
type CreateBuildRequest struct {
	SourceRepo     string         `json:"source_repo,omitempty"`
	SourceCommit   string         `json:"source_commit,omitempty"`
	SourceBranch   string         `json:"source_branch,omitempty"`
	BuilderType    string         `json:"builder_type" validate:"required"`
	BuilderVersion string         `json:"builder_version,omitempty"`
	BuildTemplate  string         `json:"build_template,omitempty"`
	BuildConfig    map[string]any `json:"build_config,omitempty"`
	BuildRunner    string         `json:"build_runner,omitempty"`
	BuildRunnerID  string         `json:"build_runner_id,omitempty"`
	BuiltBy        string         `json:"built_by,omitempty"`
}

// CreateVulnerabilityRequest represents a request to record a vulnerability.
type CreateVulnerabilityRequest struct {
	CVEID          string                `json:"cve_id" validate:"required"`
	Severity       VulnerabilitySeverity `json:"severity" validate:"required"`
	CVSSScore      float64               `json:"cvss_score,omitempty"`
	CVSSVector     string                `json:"cvss_vector,omitempty"`
	PackageName    string                `json:"package_name,omitempty"`
	PackageVersion string                `json:"package_version,omitempty"`
	PackageType    string                `json:"package_type,omitempty"`
	FixedVersion   string                `json:"fixed_version,omitempty"`
	Scanner        string                `json:"scanner,omitempty"`
}

// CreatePromotionRequest represents a request to promote an image.
type CreatePromotionRequest struct {
	ToStatus       ImageStatus    `json:"to_status" validate:"required"`
	ApprovedBy     string         `json:"approved_by,omitempty"`
	ApprovalTicket string         `json:"approval_ticket,omitempty"`
	Reason         string         `json:"reason,omitempty"`
	ValidationResults map[string]any `json:"validation_results,omitempty"`
}

// ImageLineageResponse is the API response for image lineage queries.
type ImageLineageResponse struct {
	Image     *Image            `json:"image"`
	Parents   []ImageLineage    `json:"parents"`
	Children  []ImageLineage    `json:"children"`
	Builds    []ImageBuild      `json:"builds"`
	Vulns     VulnerabilitySummary `json:"vulnerability_summary"`
	Deployments int             `json:"active_deployments"`
	Promotions []ImagePromotion `json:"promotions"`
}
