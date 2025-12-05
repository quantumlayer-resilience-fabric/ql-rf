// Package sbom provides Software Bill of Materials (SBOM) generation and management.
package sbom

import (
	"time"

	"github.com/google/uuid"
)

// Format represents the SBOM format type.
type Format string

const (
	// FormatSPDX represents the SPDX SBOM format (ISO/IEC 5962:2021).
	FormatSPDX Format = "spdx"
	// FormatCycloneDX represents the CycloneDX SBOM format (OWASP).
	FormatCycloneDX Format = "cyclonedx"
)

// String returns the string representation of the format.
func (f Format) String() string {
	return string(f)
}

// IsValid checks if the format is valid.
func (f Format) IsValid() bool {
	switch f {
	case FormatSPDX, FormatCycloneDX:
		return true
	default:
		return false
	}
}

// SBOM represents a Software Bill of Materials document.
type SBOM struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	ImageID     uuid.UUID              `json:"image_id" db:"image_id"`
	OrgID       uuid.UUID              `json:"org_id" db:"org_id"`
	Format      Format                 `json:"format" db:"format"`
	Version     string                 `json:"version" db:"version"` // Format version (e.g., "SPDX-2.3", "CycloneDX-1.5")
	Content     map[string]interface{} `json:"content" db:"content"` // Full SBOM document as JSONB
	PackageCount int                   `json:"package_count" db:"package_count"`
	VulnCount    int                   `json:"vuln_count,omitempty" db:"vuln_count"`
	GeneratedAt time.Time              `json:"generated_at" db:"generated_at"`
	Scanner     string                 `json:"scanner,omitempty" db:"scanner"` // e.g., "syft", "trivy", "grype"
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`

	// Loaded relationships
	Packages        []Package        `json:"packages,omitempty"`
	Vulnerabilities []Vulnerability  `json:"vulnerabilities,omitempty"`
}

// Package represents a software package in an SBOM.
type Package struct {
	ID       uuid.UUID `json:"id" db:"id"`
	SBOMID   uuid.UUID `json:"sbom_id" db:"sbom_id"`
	Name     string    `json:"name" db:"name"`
	Version  string    `json:"version" db:"version"`
	Type     string    `json:"type" db:"type"` // deb, rpm, apk, npm, pip, go, jar, etc.
	PURL     string    `json:"purl,omitempty" db:"purl"` // Package URL (purl)
	CPE      string    `json:"cpe,omitempty" db:"cpe"`   // Common Platform Enumeration
	License  string    `json:"license,omitempty" db:"license"`
	Supplier string    `json:"supplier,omitempty" db:"supplier"`
	Checksum string    `json:"checksum,omitempty" db:"checksum"` // SHA256 hash
	SourceURL string   `json:"source_url,omitempty" db:"source_url"`
	Location  string   `json:"location,omitempty" db:"location"` // File path in image
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Dependency represents a dependency relationship between packages.
type Dependency struct {
	ID         uuid.UUID `json:"id" db:"id"`
	SBOMID     uuid.UUID `json:"sbom_id" db:"sbom_id"`
	PackageRef string    `json:"package_ref" db:"package_ref"` // Package name or ID
	DependsOn  string    `json:"depends_on" db:"depends_on"`   // Dependency package name or ID
	Scope      string    `json:"scope,omitempty" db:"scope"`   // runtime, development, test
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// Vulnerability represents a security vulnerability in a package.
type Vulnerability struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	SBOMID         uuid.UUID  `json:"sbom_id" db:"sbom_id"`
	PackageID      uuid.UUID  `json:"package_id" db:"package_id"`
	CVEID          string     `json:"cve_id" db:"cve_id"` // CVE-2024-1234
	Severity       string     `json:"severity" db:"severity"` // critical, high, medium, low, unknown
	CVSSScore      *float64   `json:"cvss_score,omitempty" db:"cvss_score"`
	CVSSVector     string     `json:"cvss_vector,omitempty" db:"cvss_vector"`
	Description    string     `json:"description,omitempty" db:"description"`
	FixedVersion   string     `json:"fixed_version,omitempty" db:"fixed_version"`
	PublishedDate  *time.Time `json:"published_date,omitempty" db:"published_date"`
	ModifiedDate   *time.Time `json:"modified_date,omitempty" db:"modified_date"`
	References     []string   `json:"references,omitempty" db:"references"` // URLs to advisories
	DataSource     string     `json:"data_source,omitempty" db:"data_source"` // NVD, OSV, GitHub, etc.
	ExploitAvailable bool     `json:"exploit_available" db:"exploit_available"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// SBOMGenerationRequest represents a request to generate an SBOM.
type SBOMGenerationRequest struct {
	ImageID      uuid.UUID `json:"image_id" validate:"required"`
	Format       Format    `json:"format" validate:"required,oneof=spdx cyclonedx"`
	Scanner      string    `json:"scanner,omitempty"` // Optional: syft, trivy, grype
	IncludeVulns bool      `json:"include_vulns"`     // Whether to scan for vulnerabilities
}

// SBOMGenerationResponse represents the response from SBOM generation.
type SBOMGenerationResponse struct {
	SBOM         *SBOM  `json:"sbom"`
	Status       string `json:"status"` // success, partial, failed
	Message      string `json:"message,omitempty"`
	PackageCount int    `json:"package_count"`
	VulnCount    int    `json:"vuln_count"`
	GeneratedAt  time.Time `json:"generated_at"`
}

// SBOMExportRequest represents a request to export an SBOM.
type SBOMExportRequest struct {
	SBOMID uuid.UUID `json:"sbom_id" validate:"required"`
	Format Format    `json:"format" validate:"required,oneof=spdx cyclonedx"`
}

// SBOMExportResponse represents the exported SBOM data.
type SBOMExportResponse struct {
	Format  Format                 `json:"format"`
	Content map[string]interface{} `json:"content"`
}

// SBOMSummary provides a lightweight summary of an SBOM.
type SBOMSummary struct {
	ID           uuid.UUID `json:"id"`
	ImageID      uuid.UUID `json:"image_id"`
	Format       Format    `json:"format"`
	PackageCount int       `json:"package_count"`
	VulnCount    int       `json:"vuln_count"`
	Critical     int       `json:"critical"`
	High         int       `json:"high"`
	Medium       int       `json:"medium"`
	Low          int       `json:"low"`
	GeneratedAt  time.Time `json:"generated_at"`
}

// PackageManifest represents a parsed package manifest file.
type PackageManifest struct {
	Type     string              `json:"type"` // npm, pip, go, maven, nuget
	Packages []ManifestPackage   `json:"packages"`
	Metadata map[string]string   `json:"metadata,omitempty"`
}

// ManifestPackage represents a package from a manifest file.
type ManifestPackage struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	License  string `json:"license,omitempty"`
	Dev      bool   `json:"dev,omitempty"` // Development dependency
}

// VulnerabilityFilter represents filters for querying vulnerabilities.
type VulnerabilityFilter struct {
	SBOMID       uuid.UUID
	Severities   []string // critical, high, medium, low
	MinCVSS      *float64
	HasExploit   *bool
	FixAvailable *bool
}

// SBOMListResponse represents a paginated list of SBOMs.
type SBOMListResponse struct {
	SBOMs      []SBOMSummary `json:"sboms"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}
