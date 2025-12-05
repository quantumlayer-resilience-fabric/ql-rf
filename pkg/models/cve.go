package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// CVE Feed Sources
// =============================================================================

// CVEFeedSource represents a configured CVE data source.
type CVEFeedSource struct {
	ID                    uuid.UUID  `json:"id" db:"id"`
	Name                  string     `json:"name" db:"name"`                                     // nvd, osv, github_advisory, cisa_kev
	DisplayName           string     `json:"displayName" db:"display_name"`                      // Human-readable name
	SourceType            string     `json:"sourceType" db:"source_type"`                        // api, rss, webhook
	APIURL                string     `json:"apiUrl" db:"api_url"`                                // Base URL for the API
	APIKeyRef             *string    `json:"apiKeyRef,omitempty" db:"api_key_ref"`               // Reference to secret storage
	PollIntervalMinutes   int        `json:"pollIntervalMinutes" db:"poll_interval_minutes"`     // How often to poll
	LastPollAt            *time.Time `json:"lastPollAt,omitempty" db:"last_poll_at"`             // Last poll attempt
	LastSuccessfulPollAt  *time.Time `json:"lastSuccessfulPollAt,omitempty" db:"last_successful_poll_at"`
	LastError             *string    `json:"lastError,omitempty" db:"last_error"`                // Last error message
	Enabled               bool       `json:"enabled" db:"enabled"`
	Priority              int        `json:"priority" db:"priority"`                             // Lower = higher priority
	CreatedAt             time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt             time.Time  `json:"updatedAt" db:"updated_at"`
}

// CVEFeedSourceType constants.
type CVEFeedSourceType string

const (
	CVEFeedSourceNVD           CVEFeedSourceType = "nvd"
	CVEFeedSourceOSV           CVEFeedSourceType = "osv"
	CVEFeedSourceGitHubAdvisory CVEFeedSourceType = "github_advisory"
	CVEFeedSourceCISAKEV       CVEFeedSourceType = "cisa_kev"
)

// =============================================================================
// CVE Cache
// =============================================================================

// CVECache represents a normalized CVE record aggregated from multiple sources.
type CVECache struct {
	ID                  uuid.UUID        `json:"id" db:"id"`
	CVEID               string           `json:"cveId" db:"cve_id"`                          // CVE-2024-1234, GHSA-xxxx
	CVSSV3Score         *float64         `json:"cvssV3Score,omitempty" db:"cvss_v3_score"`   // 0.0-10.0
	CVSSV3Vector        *string          `json:"cvssV3Vector,omitempty" db:"cvss_v3_vector"`
	CVSSV2Score         *float64         `json:"cvssV2Score,omitempty" db:"cvss_v2_score"`
	Severity            string           `json:"severity" db:"severity"`                     // critical, high, medium, low, unknown
	EPSSScore           *float64         `json:"epssScore,omitempty" db:"epss_score"`        // 0.0-1.0 probability
	EPSSPercentile      *float64         `json:"epssPercentile,omitempty" db:"epss_percentile"`
	EPSSUpdatedAt       *time.Time       `json:"epssUpdatedAt,omitempty" db:"epss_updated_at"`
	ExploitAvailable    bool             `json:"exploitAvailable" db:"exploit_available"`
	ExploitMaturity     *string          `json:"exploitMaturity,omitempty" db:"exploit_maturity"`   // unproven, poc, functional, high
	ExploitURLs         json.RawMessage  `json:"exploitUrls,omitempty" db:"exploit_urls"`
	CISAKEVListed       bool             `json:"cisaKevListed" db:"cisa_kev_listed"`
	CISAKEVAddedDate    *time.Time       `json:"cisaKevAddedDate,omitempty" db:"cisa_kev_added_date"`
	CISAKEVDueDate      *time.Time       `json:"cisaKevDueDate,omitempty" db:"cisa_kev_due_date"`
	CISAKEVRansomware   *bool            `json:"cisaKevRansomware,omitempty" db:"cisa_kev_ransomware"`
	Description         *string          `json:"description,omitempty" db:"description"`
	PublishedDate       *time.Time       `json:"publishedDate,omitempty" db:"published_date"`
	ModifiedDate        *time.Time       `json:"modifiedDate,omitempty" db:"modified_date"`
	AffectedCPEPatterns json.RawMessage  `json:"affectedCpePatterns,omitempty" db:"affected_cpe_patterns"`
	ReferenceURLs       json.RawMessage  `json:"referenceUrls,omitempty" db:"reference_urls"`
	RemediationSummary  *string          `json:"remediationSummary,omitempty" db:"remediation_summary"`
	VendorAdvisoryURLs  json.RawMessage  `json:"vendorAdvisoryUrls,omitempty" db:"vendor_advisory_urls"`
	PrimarySource       string           `json:"primarySource" db:"primary_source"`
	Sources             json.RawMessage  `json:"sources" db:"sources"`
	RawData             json.RawMessage  `json:"rawData,omitempty" db:"raw_data"`
	FetchedAt           time.Time        `json:"fetchedAt" db:"fetched_at"`
	CreatedAt           time.Time        `json:"createdAt" db:"created_at"`
	UpdatedAt           time.Time        `json:"updatedAt" db:"updated_at"`
}

// CVESeverity constants.
type CVESeverity string

const (
	CVESeverityCritical CVESeverity = "critical"
	CVESeverityHigh     CVESeverity = "high"
	CVESeverityMedium   CVESeverity = "medium"
	CVESeverityLow      CVESeverity = "low"
	CVESeverityUnknown  CVESeverity = "unknown"
)

// SeverityFromCVSS converts a CVSS score to severity.
func SeverityFromCVSS(score float64) CVESeverity {
	switch {
	case score >= 9.0:
		return CVESeverityCritical
	case score >= 7.0:
		return CVESeverityHigh
	case score >= 4.0:
		return CVESeverityMedium
	case score > 0.0:
		return CVESeverityLow
	default:
		return CVESeverityUnknown
	}
}

// =============================================================================
// CVE Alerts
// =============================================================================

// CVEAlert represents a CVE alert for an organization.
type CVEAlert struct {
	ID                    uuid.UUID  `json:"id" db:"id"`
	OrgID                 uuid.UUID  `json:"orgId" db:"org_id"`
	CVEID                 string     `json:"cveId" db:"cve_id"`
	CVECacheID            *uuid.UUID `json:"cveCacheId,omitempty" db:"cve_cache_id"`
	Severity              string     `json:"severity" db:"severity"`
	UrgencyScore          int        `json:"urgencyScore" db:"urgency_score"`              // 0-100
	Status                string     `json:"status" db:"status"`
	Priority              *string    `json:"priority,omitempty" db:"priority"`             // p1, p2, p3, p4
	SLADueAt              *time.Time `json:"slaDueAt,omitempty" db:"sla_due_at"`
	SLABreached           bool       `json:"slaBreached" db:"sla_breached"`
	AffectedImagesCount   int        `json:"affectedImagesCount" db:"affected_images_count"`
	AffectedAssetsCount   int        `json:"affectedAssetsCount" db:"affected_assets_count"`
	AffectedPackagesCount int        `json:"affectedPackagesCount" db:"affected_packages_count"`
	ProductionAssetsCount int        `json:"productionAssetsCount" db:"production_assets_count"`
	AssignedTo            *string    `json:"assignedTo,omitempty" db:"assigned_to"`
	AssignedAt            *time.Time `json:"assignedAt,omitempty" db:"assigned_at"`
	ResolutionType        *string    `json:"resolutionType,omitempty" db:"resolution_type"`
	ResolutionNotes       *string    `json:"resolutionNotes,omitempty" db:"resolution_notes"`
	ResolvedBy            *string    `json:"resolvedBy,omitempty" db:"resolved_by"`
	ResolvedAt            *time.Time `json:"resolvedAt,omitempty" db:"resolved_at"`
	PatchCampaignID       *uuid.UUID `json:"patchCampaignId,omitempty" db:"patch_campaign_id"`
	TicketID              *string    `json:"ticketId,omitempty" db:"ticket_id"`
	DetectedAt            time.Time  `json:"detectedAt" db:"detected_at"`
	FirstSeenAt           time.Time  `json:"firstSeenAt" db:"first_seen_at"`
	LastSeenAt            time.Time  `json:"lastSeenAt" db:"last_seen_at"`
	CreatedAt             time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt             time.Time  `json:"updatedAt" db:"updated_at"`

	// Enriched fields (not in DB, populated from joins)
	CVEDetails *CVECache `json:"cveDetails,omitempty" db:"-"`
}

// CVEAlertStatus constants.
type CVEAlertStatus string

const (
	CVEAlertStatusNew          CVEAlertStatus = "new"
	CVEAlertStatusInvestigating CVEAlertStatus = "investigating"
	CVEAlertStatusConfirmed    CVEAlertStatus = "confirmed"
	CVEAlertStatusInProgress   CVEAlertStatus = "in_progress"
	CVEAlertStatusResolved     CVEAlertStatus = "resolved"
	CVEAlertStatusDismissed    CVEAlertStatus = "dismissed"
	CVEAlertStatusAutoResolved CVEAlertStatus = "auto_resolved"
)

// CVEAlertPriority constants.
type CVEAlertPriority string

const (
	CVEAlertPriorityP1 CVEAlertPriority = "p1" // Critical - immediate response
	CVEAlertPriorityP2 CVEAlertPriority = "p2" // High - same day
	CVEAlertPriorityP3 CVEAlertPriority = "p3" // Medium - within SLA
	CVEAlertPriorityP4 CVEAlertPriority = "p4" // Low - scheduled maintenance
)

// CVEAlertResolutionType constants.
type CVEAlertResolutionType string

const (
	CVEAlertResolutionPatched      CVEAlertResolutionType = "patched"
	CVEAlertResolutionUpgraded     CVEAlertResolutionType = "upgraded"
	CVEAlertResolutionMitigated    CVEAlertResolutionType = "mitigated"
	CVEAlertResolutionAcceptedRisk CVEAlertResolutionType = "accepted_risk"
	CVEAlertResolutionFalsePositive CVEAlertResolutionType = "false_positive"
)

// =============================================================================
// CVE Alert Affected Items
// =============================================================================

// CVEAlertAffectedItem represents a package, image, or asset affected by a CVE.
type CVEAlertAffectedItem struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	AlertID              uuid.UUID  `json:"alertId" db:"alert_id"`
	PackageID            *uuid.UUID `json:"packageId,omitempty" db:"package_id"`
	ImageID              *uuid.UUID `json:"imageId,omitempty" db:"image_id"`
	AssetID              *uuid.UUID `json:"assetId,omitempty" db:"asset_id"`
	ItemType             string     `json:"itemType" db:"item_type"`                         // package, image, asset
	PackageName          *string    `json:"packageName,omitempty" db:"package_name"`
	PackageVersion       *string    `json:"packageVersion,omitempty" db:"package_version"`
	PackageType          *string    `json:"packageType,omitempty" db:"package_type"`
	FixedVersion         *string    `json:"fixedVersion,omitempty" db:"fixed_version"`
	ImageFamily          *string    `json:"imageFamily,omitempty" db:"image_family"`
	ImageVersion         *string    `json:"imageVersion,omitempty" db:"image_version"`
	AssetName            *string    `json:"assetName,omitempty" db:"asset_name"`
	AssetPlatform        *string    `json:"assetPlatform,omitempty" db:"asset_platform"`
	AssetEnvironment     *string    `json:"assetEnvironment,omitempty" db:"asset_environment"`
	AssetRegion          *string    `json:"assetRegion,omitempty" db:"asset_region"`
	IsProduction         bool       `json:"isProduction" db:"is_production"`
	InheritedFromImageID *uuid.UUID `json:"inheritedFromImageId,omitempty" db:"inherited_from_image_id"`
	LineageDepth         int        `json:"lineageDepth" db:"lineage_depth"`
	ItemStatus           string     `json:"itemStatus" db:"item_status"`
	RemediationMethod    *string    `json:"remediationMethod,omitempty" db:"remediation_method"`
	RemediationStartedAt *time.Time `json:"remediationStartedAt,omitempty" db:"remediation_started_at"`
	RemediationCompletedAt *time.Time `json:"remediationCompletedAt,omitempty" db:"remediation_completed_at"`
	CreatedAt            time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt            time.Time  `json:"updatedAt" db:"updated_at"`
}

// CVEAlertItemType constants.
type CVEAlertItemType string

const (
	CVEAlertItemTypePackage CVEAlertItemType = "package"
	CVEAlertItemTypeImage   CVEAlertItemType = "image"
	CVEAlertItemTypeAsset   CVEAlertItemType = "asset"
)

// CVEAlertItemStatus constants.
type CVEAlertItemStatus string

const (
	CVEAlertItemStatusVulnerable     CVEAlertItemStatus = "vulnerable"
	CVEAlertItemStatusPatchPending   CVEAlertItemStatus = "patch_pending"
	CVEAlertItemStatusPatching       CVEAlertItemStatus = "patching"
	CVEAlertItemStatusPatched        CVEAlertItemStatus = "patched"
	CVEAlertItemStatusMitigated      CVEAlertItemStatus = "mitigated"
	CVEAlertItemStatusNotAffected    CVEAlertItemStatus = "not_affected"
	CVEAlertItemStatusDecommissioned CVEAlertItemStatus = "decommissioned"
)

// =============================================================================
// CVE Package Matches
// =============================================================================

// CVEPackageMatch maps CVEs to affected package patterns.
type CVEPackageMatch struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	CVECacheID        uuid.UUID  `json:"cveCacheId" db:"cve_cache_id"`
	PackageName       string     `json:"packageName" db:"package_name"`
	PackageType       *string    `json:"packageType,omitempty" db:"package_type"`
	VersionStart      *string    `json:"versionStart,omitempty" db:"version_start"`
	VersionEnd        *string    `json:"versionEnd,omitempty" db:"version_end"`
	VersionConstraint string     `json:"versionConstraint" db:"version_constraint"` // exact, range, less_than, less_than_eq, all
	CPEPattern        *string    `json:"cpePattern,omitempty" db:"cpe_pattern"`
	PURLPattern       *string    `json:"purlPattern,omitempty" db:"purl_pattern"`
	FixedVersion      *string    `json:"fixedVersion,omitempty" db:"fixed_version"`
	FixAvailable      bool       `json:"fixAvailable" db:"fix_available"`
	CreatedAt         time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt         time.Time  `json:"updatedAt" db:"updated_at"`
}

// VersionConstraintType constants.
type VersionConstraintType string

const (
	VersionConstraintExact     VersionConstraintType = "exact"
	VersionConstraintRange     VersionConstraintType = "range"
	VersionConstraintLessThan  VersionConstraintType = "less_than"
	VersionConstraintLessThanEq VersionConstraintType = "less_than_eq"
	VersionConstraintAll       VersionConstraintType = "all"
)

// =============================================================================
// Urgency Score Calculation
// =============================================================================

// UrgencyScoreInput contains the inputs for calculating urgency score.
type UrgencyScoreInput struct {
	CVSSScore            float64
	EPSSScore            float64
	ExploitAvailable     bool
	CISAKEVListed        bool
	ProductionAssetCount int
	TotalAssetCount      int
	FleetPercentage      float64 // Percentage of fleet affected
}

// CalculateUrgencyScore calculates the urgency score (0-100).
func CalculateUrgencyScore(input UrgencyScoreInput) int {
	score := 0.0

	// CVSS contribution (0-100 points, weighted at 10x)
	score += input.CVSSScore * 10

	// Exploit availability (+25 points)
	if input.ExploitAvailable {
		score += 25
	}

	// CISA KEV (+20 points)
	if input.CISAKEVListed {
		score += 20
	}

	// Production assets (+15 points if any production assets affected)
	if input.ProductionAssetCount > 0 {
		score += 15
	}

	// Fleet percentage factor (0-10 points)
	score += min(input.FleetPercentage/10, 10)

	// EPSS contribution (0-20 points)
	score += input.EPSSScore * 20

	// Normalize to 0-100
	result := int(score / 2)
	if result > 100 {
		result = 100
	}
	if result < 0 {
		result = 0
	}

	return result
}

// =============================================================================
// API Request/Response Types
// =============================================================================

// CVEAlertFilter represents filters for listing CVE alerts.
type CVEAlertFilter struct {
	Severity        string     `json:"severity,omitempty"`
	Status          string     `json:"status,omitempty"`
	Priority        string     `json:"priority,omitempty"`
	CVEID           string     `json:"cveId,omitempty"`
	MinUrgencyScore *int       `json:"minUrgencyScore,omitempty"`
	SLABreached     *bool      `json:"slaBreached,omitempty"`
	HasExploit      *bool      `json:"hasExploit,omitempty"`
	CISAKEVOnly     *bool      `json:"cisaKevOnly,omitempty"`
	AssignedTo      *string    `json:"assignedTo,omitempty"`
}

// CVEAlertListResponse represents a paginated list of CVE alerts.
type CVEAlertListResponse struct {
	Alerts     []CVEAlert `json:"alerts"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"pageSize"`
	TotalPages int        `json:"totalPages"`
}

// CVEAlertSummary represents aggregated alert statistics.
type CVEAlertSummary struct {
	TotalAlerts           int `json:"totalAlerts"`
	NewAlerts             int `json:"newAlerts"`
	InProgressAlerts      int `json:"inProgressAlerts"`
	ResolvedAlerts        int `json:"resolvedAlerts"`
	CriticalAlerts        int `json:"criticalAlerts"`
	HighAlerts            int `json:"highAlerts"`
	MediumAlerts          int `json:"mediumAlerts"`
	LowAlerts             int `json:"lowAlerts"`
	SLABreachedAlerts     int `json:"slaBreachedAlerts"`
	ExploitableAlerts     int `json:"exploitableAlerts"`
	CISAKEVAlerts         int `json:"cisaKevAlerts"`
	AverageUrgencyScore   int `json:"averageUrgencyScore"`
	TotalAffectedAssets   int `json:"totalAffectedAssets"`
	ProductionAffectedAssets int `json:"productionAffectedAssets"`
}

// CVEAlertWithBlastRadius represents an alert with its full blast radius details.
type CVEAlertWithBlastRadius struct {
	CVEAlert
	AffectedItems   []CVEAlertAffectedItem `json:"affectedItems"`
	AffectedPlatforms []string             `json:"affectedPlatforms"`
	AffectedRegions   []string             `json:"affectedRegions"`
}

// UpdateCVEAlertStatusRequest represents a request to update alert status.
type UpdateCVEAlertStatusRequest struct {
	Status          string  `json:"status" validate:"required,oneof=new investigating confirmed in_progress resolved dismissed"`
	AssignedTo      *string `json:"assignedTo,omitempty"`
	ResolutionType  *string `json:"resolutionType,omitempty"`
	ResolutionNotes *string `json:"resolutionNotes,omitempty"`
	TicketID        *string `json:"ticketId,omitempty"`
}

// CreatePatchCampaignFromAlertRequest represents a request to create a patch campaign from an alert.
type CreatePatchCampaignFromAlertRequest struct {
	Name              string   `json:"name" validate:"required,min=1,max=255"`
	Description       *string  `json:"description,omitempty"`
	CampaignType      string   `json:"campaignType" validate:"required,oneof=cve_response emergency"`
	RolloutStrategy   string   `json:"rolloutStrategy" validate:"required,oneof=immediate canary blue_green rolling"`
	CanaryPercentage  *int     `json:"canaryPercentage,omitempty"`
	WavePercentage    *int     `json:"wavePercentage,omitempty"`
	RequiresApproval  bool     `json:"requiresApproval"`
	ScheduledStartAt  *time.Time `json:"scheduledStartAt,omitempty"`
	TargetAssetIDs    []uuid.UUID `json:"targetAssetIds,omitempty"`
}

// =============================================================================
// CVE Enrichment Types
// =============================================================================

// CVEEnrichmentData represents additional data to enrich a CVE.
type CVEEnrichmentData struct {
	EPSSScore         *float64   `json:"epssScore,omitempty"`
	EPSSPercentile    *float64   `json:"epssPercentile,omitempty"`
	ExploitAvailable  *bool      `json:"exploitAvailable,omitempty"`
	ExploitMaturity   *string    `json:"exploitMaturity,omitempty"`
	ExploitURLs       []string   `json:"exploitUrls,omitempty"`
	CISAKEVListed     *bool      `json:"cisaKevListed,omitempty"`
	CISAKEVAddedDate  *time.Time `json:"cisaKevAddedDate,omitempty"`
	CISAKEVDueDate    *time.Time `json:"cisaKevDueDate,omitempty"`
	CISAKEVRansomware *bool      `json:"cisaKevRansomware,omitempty"`
}

// =============================================================================
// Blast Radius Types
// =============================================================================

// BlastRadiusResult represents the result of blast radius calculation.
type BlastRadiusResult struct {
	CVEID              string              `json:"cveId"`
	TotalPackages      int                 `json:"totalPackages"`
	TotalImages        int                 `json:"totalImages"`
	TotalAssets        int                 `json:"totalAssets"`
	ProductionAssets   int                 `json:"productionAssets"`
	AffectedPlatforms  []string            `json:"affectedPlatforms"`
	AffectedRegions    []string            `json:"affectedRegions"`
	AffectedPackages   []AffectedPackage   `json:"affectedPackages"`
	AffectedImages     []AffectedImage     `json:"affectedImages"`
	AffectedAssets     []AffectedAsset     `json:"affectedAssets"`
	UrgencyScore       int                 `json:"urgencyScore"`
	CalculatedAt       time.Time           `json:"calculatedAt"`
}

// AffectedPackage represents a package affected by a CVE.
type AffectedPackage struct {
	PackageID      uuid.UUID `json:"packageId"`
	SBOMID         uuid.UUID `json:"sbomId"`
	ImageID        uuid.UUID `json:"imageId"`
	PackageName    string    `json:"packageName"`
	PackageVersion string    `json:"packageVersion"`
	PackageType    string    `json:"packageType"`
	FixedVersion   *string   `json:"fixedVersion,omitempty"`
}

// AffectedImage represents an image affected by a CVE.
type AffectedImage struct {
	ImageID        uuid.UUID   `json:"imageId"`
	ImageFamily    string      `json:"imageFamily"`
	ImageVersion   string      `json:"imageVersion"`
	IsDirect       bool        `json:"isDirect"`       // True if CVE is in this image's packages
	InheritedFrom  *uuid.UUID  `json:"inheritedFrom,omitempty"` // Parent image if inherited
	LineageDepth   int         `json:"lineageDepth"`
	ChildImageIDs  []uuid.UUID `json:"childImageIds,omitempty"`
}

// AffectedAsset represents an asset affected by a CVE.
type AffectedAsset struct {
	AssetID       uuid.UUID `json:"assetId"`
	AssetName     string    `json:"assetName"`
	Platform      string    `json:"platform"`
	Region        string    `json:"region"`
	Environment   string    `json:"environment"`
	IsProduction  bool      `json:"isProduction"`
	ImageRef      string    `json:"imageRef"`
	ImageID       *uuid.UUID `json:"imageId,omitempty"`
}
