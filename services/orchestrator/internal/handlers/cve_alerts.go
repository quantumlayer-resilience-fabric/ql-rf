// Package handlers provides HTTP handlers for the AI orchestrator service.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/middleware"
)

// =============================================================================
// CVE Alert Types
// =============================================================================

// CVESeverity represents the severity level of a CVE.
type CVESeverity string

const (
	CVESeverityCritical CVESeverity = "critical"
	CVESeverityHigh     CVESeverity = "high"
	CVESeverityMedium   CVESeverity = "medium"
	CVESeverityLow      CVESeverity = "low"
	CVESeverityUnknown  CVESeverity = "unknown"
)

// CVEAlertStatus represents the status of a CVE alert.
type CVEAlertStatus string

const (
	CVEAlertStatusNew           CVEAlertStatus = "new"
	CVEAlertStatusInvestigating CVEAlertStatus = "investigating"
	CVEAlertStatusConfirmed     CVEAlertStatus = "confirmed"
	CVEAlertStatusInProgress    CVEAlertStatus = "in_progress"
	CVEAlertStatusResolved      CVEAlertStatus = "resolved"
	CVEAlertStatusDismissed     CVEAlertStatus = "dismissed"
	CVEAlertStatusAutoResolved  CVEAlertStatus = "auto_resolved"
)

// CVECache represents cached CVE details from vulnerability feeds.
type CVECache struct {
	ID                 string   `json:"id"`
	CVEID              string   `json:"cve_id"`
	CVSSV3Score        *float64 `json:"cvss_v3_score,omitempty"`
	CVSSV3Vector       *string  `json:"cvss_v3_vector,omitempty"`
	Severity           string   `json:"severity"`
	EPSSScore          *float64 `json:"epss_score,omitempty"`
	EPSSPercentile     *float64 `json:"epss_percentile,omitempty"`
	ExploitAvailable   bool     `json:"exploit_available"`
	ExploitMaturity    *string  `json:"exploit_maturity,omitempty"`
	CISAKEVListed      bool     `json:"cisa_kev_listed"`
	CISAKEVDueDate     *string  `json:"cisa_kev_due_date,omitempty"`
	CISAKEVRansomware  *bool    `json:"cisa_kev_ransomware,omitempty"`
	Description        *string  `json:"description,omitempty"`
	PublishedDate      *string  `json:"published_date,omitempty"`
	ModifiedDate       *string  `json:"modified_date,omitempty"`
	PrimarySource      string   `json:"primary_source"`
	ReferenceURLs      []string `json:"reference_urls,omitempty"`
	RemediationSummary *string  `json:"remediation_summary,omitempty"`
}

// CVEAlert represents a CVE alert for an organization.
type CVEAlert struct {
	ID                    string         `json:"id"`
	OrgID                 string         `json:"org_id"`
	CVEID                 string         `json:"cve_id"`
	CVECacheID            *string        `json:"cve_cache_id,omitempty"`
	Severity              CVESeverity    `json:"severity"`
	UrgencyScore          float64        `json:"urgency_score"`
	Status                CVEAlertStatus `json:"status"`
	Priority              *string        `json:"priority,omitempty"`
	SLADueAt              *time.Time     `json:"sla_due_at,omitempty"`
	SLABreached           bool           `json:"sla_breached"`
	AffectedImagesCount   int            `json:"affected_images_count"`
	AffectedAssetsCount   int            `json:"affected_assets_count"`
	AffectedPackagesCount int            `json:"affected_packages_count"`
	ProductionAssetsCount int            `json:"production_assets_count"`
	AssignedTo            *string        `json:"assigned_to,omitempty"`
	AssignedAt            *time.Time     `json:"assigned_at,omitempty"`
	ResolutionType        *string        `json:"resolution_type,omitempty"`
	ResolutionNotes       *string        `json:"resolution_notes,omitempty"`
	ResolvedBy            *string        `json:"resolved_by,omitempty"`
	ResolvedAt            *time.Time     `json:"resolved_at,omitempty"`
	PatchCampaignID       *string        `json:"patch_campaign_id,omitempty"`
	TicketID              *string        `json:"ticket_id,omitempty"`
	DetectedAt            time.Time      `json:"detected_at"`
	FirstSeenAt           time.Time      `json:"first_seen_at"`
	LastSeenAt            time.Time      `json:"last_seen_at"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	CVEDetails            *CVECache      `json:"cve_details,omitempty"`
}

// CVEAlertSummary provides aggregate statistics for CVE alerts.
type CVEAlertSummary struct {
	TotalAlerts              int     `json:"total_alerts"`
	NewAlerts                int     `json:"new_alerts"`
	InProgressAlerts         int     `json:"in_progress_alerts"`
	ResolvedAlerts           int     `json:"resolved_alerts"`
	CriticalAlerts           int     `json:"critical_alerts"`
	HighAlerts               int     `json:"high_alerts"`
	MediumAlerts             int     `json:"medium_alerts"`
	LowAlerts                int     `json:"low_alerts"`
	SLABreachedAlerts        int     `json:"sla_breached_alerts"`
	ExploitableAlerts        int     `json:"exploitable_alerts"`
	CISAKEVAlerts            int     `json:"cisa_kev_alerts"`
	AverageUrgencyScore      float64 `json:"average_urgency_score"`
	TotalAffectedAssets      int     `json:"total_affected_assets"`
	ProductionAffectedAssets int     `json:"production_affected_assets"`
}

// CVEAlertListResponse is the response for listing CVE alerts.
type CVEAlertListResponse struct {
	Alerts     []CVEAlert `json:"alerts"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}

// UpdateCVEAlertStatusRequest is the request body for updating alert status.
type UpdateCVEAlertStatusRequest struct {
	Status          CVEAlertStatus `json:"status"`
	AssignedTo      *string        `json:"assigned_to,omitempty"`
	ResolutionType  *string        `json:"resolution_type,omitempty"`
	ResolutionNotes *string        `json:"resolution_notes,omitempty"`
	TicketID        *string        `json:"ticket_id,omitempty"`
}

// =============================================================================
// CVE Alert Handlers
// =============================================================================

// RegisterCVEAlertRoutes registers the CVE alert routes.
func (h *Handler) RegisterCVEAlertRoutes(r chi.Router) {
	r.Route("/cve-alerts", func(r chi.Router) {
		r.Get("/", h.listCVEAlerts)
		r.Get("/summary", h.getCVEAlertSummary)
		r.Get("/{alertID}", h.getCVEAlert)
		r.Patch("/{alertID}/status", h.updateCVEAlertStatus)
		r.Get("/{alertID}/blast-radius", h.getCVEAlertBlastRadius)
		r.Post("/{alertID}/create-campaign", h.createPatchCampaignFromAlert)
	})
}

func (h *Handler) listCVEAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	page := 1
	pageSize := 50
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil && ps > 0 && ps <= 100 {
		pageSize = ps
	}

	orgID := r.URL.Query().Get("org_id")
	if orgID == "" {
		orgID = middleware.GetOrgID(ctx)
	}

	filters := cveAlertFilters{
		OrgID:           orgID,
		Severity:        r.URL.Query().Get("severity"),
		Status:          r.URL.Query().Get("status"),
		Priority:        r.URL.Query().Get("priority"),
		CVEID:           r.URL.Query().Get("cve_id"),
		MinUrgencyScore: r.URL.Query().Get("min_urgency_score"),
		SLABreached:     r.URL.Query().Get("sla_breached"),
		HasExploit:      r.URL.Query().Get("has_exploit"),
		CISAKEVOnly:     r.URL.Query().Get("cisa_kev_only"),
		Page:            page,
		PageSize:        pageSize,
	}

	alerts, total, err := h.queryCVEAlerts(ctx, filters)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to list CVE alerts", err)
		return
	}

	totalPages := 0
	if pageSize > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}

	h.respond(w, http.StatusOK, CVEAlertListResponse{
		Alerts:     alerts,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

func (h *Handler) getCVEAlertSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.URL.Query().Get("org_id")
	if orgID == "" {
		orgID = middleware.GetOrgID(ctx)
	}

	summary, err := h.queryCVEAlertSummary(ctx, orgID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to load CVE alert summary", err)
		return
	}

	h.respond(w, http.StatusOK, summary)
}

func (h *Handler) getCVEAlert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	alertID, err := uuid.Parse(chi.URLParam(r, "alertID"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid alert id", err)
		return
	}

	orgID := r.URL.Query().Get("org_id")
	if orgID == "" {
		orgID = middleware.GetOrgID(ctx)
	}

	alert, err := h.queryCVEAlertByID(ctx, alertID, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.respondError(w, http.StatusNotFound, "CVE alert not found", nil)
			return
		}
		h.respondError(w, http.StatusInternalServerError, "failed to load CVE alert", err)
		return
	}

	h.respond(w, http.StatusOK, alert)
}

func (h *Handler) updateCVEAlertStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	alertID, err := uuid.Parse(chi.URLParam(r, "alertID"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid alert id", err)
		return
	}

	var req UpdateCVEAlertStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if req.Status == "" {
		h.respondError(w, http.StatusBadRequest, "status is required", nil)
		return
	}

	orgID := r.URL.Query().Get("org_id")
	if orgID == "" {
		orgID = middleware.GetOrgID(ctx)
	}

	affected, err := h.execUpdateCVEAlertStatus(ctx, alertID, orgID, req)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to update CVE alert", err)
		return
	}
	if affected == 0 {
		h.respondError(w, http.StatusNotFound, "CVE alert not found", nil)
		return
	}

	alert, err := h.queryCVEAlertByID(ctx, alertID, orgID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to load updated CVE alert", err)
		return
	}

	h.respond(w, http.StatusOK, alert)
}

func (h *Handler) getCVEAlertBlastRadius(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	alertID, err := uuid.Parse(chi.URLParam(r, "alertID"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid alert id", err)
		return
	}

	orgID := r.URL.Query().Get("org_id")
	if orgID == "" {
		orgID = middleware.GetOrgID(ctx)
	}

	blastRadius, err := h.queryBlastRadius(ctx, alertID, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.respondError(w, http.StatusNotFound, "CVE alert not found", nil)
			return
		}
		h.respondError(w, http.StatusInternalServerError, "failed to load blast radius", err)
		return
	}

	h.respond(w, http.StatusOK, blastRadius)
}

func (h *Handler) createPatchCampaignFromAlert(w http.ResponseWriter, r *http.Request) {
	alertID := chi.URLParam(r, "alertID")

	var req struct {
		Name             string   `json:"name"`
		Description      string   `json:"description,omitempty"`
		CampaignType     string   `json:"campaign_type"`
		RolloutStrategy  string   `json:"rollout_strategy"`
		CanaryPercentage *int     `json:"canary_percentage,omitempty"`
		WavePercentage   *int     `json:"wave_percentage,omitempty"`
		RequiresApproval *bool    `json:"requires_approval,omitempty"`
		ScheduledStartAt *string  `json:"scheduled_start_at,omitempty"`
		TargetAssetIDs   []string `json:"target_asset_ids,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// For development: return mock campaign creation response
	campaignID := uuid.New().String()

	h.respond(w, http.StatusCreated, map[string]interface{}{
		"campaign_id": campaignID,
		"alert_id":    alertID,
		"message":     "Patch campaign created successfully",
	})
}

// =============================================================================
// Mock Data Generators
//
// LEGACY / UNUSED: the CVE alert endpoints now query the database (see
// cve_alerts_query.go). These generators are retained for reference and local
// fixtures only; they are not wired into any handler.
// =============================================================================

//nolint:unused // retained legacy fixtures; superseded by DB-backed handlers
func (h *Handler) generateMockCVEAlerts() []CVEAlert {
	now := time.Now().UTC()
	alerts := []CVEAlert{}

	// Critical CVE - CISA KEV
	cvssScore := 9.8
	desc1 := "A critical remote code execution vulnerability in OpenSSL that allows attackers to execute arbitrary code."
	alerts = append(alerts, CVEAlert{
		ID:                    uuid.New().String(),
		OrgID:                 "default-org",
		CVEID:                 "CVE-2024-3094",
		Severity:              CVESeverityCritical,
		UrgencyScore:          95.5,
		Status:                CVEAlertStatusNew,
		SLABreached:           true,
		AffectedImagesCount:   8,
		AffectedAssetsCount:   45,
		AffectedPackagesCount: 3,
		ProductionAssetsCount: 18,
		DetectedAt:            now.Add(-2 * time.Hour),
		FirstSeenAt:           now.Add(-2 * time.Hour),
		LastSeenAt:            now,
		CreatedAt:             now.Add(-2 * time.Hour),
		UpdatedAt:             now,
		CVEDetails: &CVECache{
			CVEID:            "CVE-2024-3094",
			CVSSV3Score:      &cvssScore,
			Severity:         "critical",
			ExploitAvailable: true,
			CISAKEVListed:    true,
			Description:      &desc1,
			PrimarySource:    "nvd",
		},
	})

	// Critical CVE - with exploit
	cvssScore2 := 9.1
	desc2 := "Authentication bypass vulnerability in enterprise management software allowing unauthorized access."
	alerts = append(alerts, CVEAlert{
		ID:                    uuid.New().String(),
		OrgID:                 "default-org",
		CVEID:                 "CVE-2024-2891",
		Severity:              CVESeverityCritical,
		UrgencyScore:          88.0,
		Status:                CVEAlertStatusInvestigating,
		SLABreached:           false,
		AffectedImagesCount:   4,
		AffectedAssetsCount:   23,
		AffectedPackagesCount: 1,
		ProductionAssetsCount: 8,
		DetectedAt:            now.Add(-6 * time.Hour),
		FirstSeenAt:           now.Add(-6 * time.Hour),
		LastSeenAt:            now,
		CreatedAt:             now.Add(-6 * time.Hour),
		UpdatedAt:             now.Add(-1 * time.Hour),
		CVEDetails: &CVECache{
			CVEID:            "CVE-2024-2891",
			CVSSV3Score:      &cvssScore2,
			Severity:         "critical",
			ExploitAvailable: true,
			CISAKEVListed:    false,
			Description:      &desc2,
			PrimarySource:    "nvd",
		},
	})

	// High severity CVE
	cvssScore3 := 7.5
	desc3 := "Denial of service vulnerability in web server that can be exploited remotely."
	alerts = append(alerts, CVEAlert{
		ID:                    uuid.New().String(),
		OrgID:                 "default-org",
		CVEID:                 "CVE-2024-1567",
		Severity:              CVESeverityHigh,
		UrgencyScore:          72.0,
		Status:                CVEAlertStatusInProgress,
		SLABreached:           false,
		AffectedImagesCount:   12,
		AffectedAssetsCount:   67,
		AffectedPackagesCount: 2,
		ProductionAssetsCount: 15,
		DetectedAt:            now.Add(-24 * time.Hour),
		FirstSeenAt:           now.Add(-24 * time.Hour),
		LastSeenAt:            now,
		CreatedAt:             now.Add(-24 * time.Hour),
		UpdatedAt:             now.Add(-2 * time.Hour),
		CVEDetails: &CVECache{
			CVEID:            "CVE-2024-1567",
			CVSSV3Score:      &cvssScore3,
			Severity:         "high",
			ExploitAvailable: false,
			CISAKEVListed:    false,
			Description:      &desc3,
			PrimarySource:    "nvd",
		},
	})

	// High severity - CISA KEV
	cvssScore4 := 8.1
	desc4 := "SQL injection vulnerability actively exploited in the wild."
	ransomware := true
	alerts = append(alerts, CVEAlert{
		ID:                    uuid.New().String(),
		OrgID:                 "default-org",
		CVEID:                 "CVE-2024-0987",
		Severity:              CVESeverityHigh,
		UrgencyScore:          82.5,
		Status:                CVEAlertStatusNew,
		SLABreached:           false,
		AffectedImagesCount:   3,
		AffectedAssetsCount:   15,
		AffectedPackagesCount: 1,
		ProductionAssetsCount: 5,
		DetectedAt:            now.Add(-4 * time.Hour),
		FirstSeenAt:           now.Add(-4 * time.Hour),
		LastSeenAt:            now,
		CreatedAt:             now.Add(-4 * time.Hour),
		UpdatedAt:             now,
		CVEDetails: &CVECache{
			CVEID:             "CVE-2024-0987",
			CVSSV3Score:       &cvssScore4,
			Severity:          "high",
			ExploitAvailable:  true,
			CISAKEVListed:     true,
			CISAKEVRansomware: &ransomware,
			Description:       &desc4,
			PrimarySource:     "cisa_kev",
		},
	})

	// Medium severity CVEs
	cvssScore5 := 5.5
	desc5 := "Information disclosure vulnerability in logging library."
	alerts = append(alerts, CVEAlert{
		ID:                    uuid.New().String(),
		OrgID:                 "default-org",
		CVEID:                 "CVE-2024-4532",
		Severity:              CVESeverityMedium,
		UrgencyScore:          45.0,
		Status:                CVEAlertStatusNew,
		SLABreached:           false,
		AffectedImagesCount:   6,
		AffectedAssetsCount:   28,
		AffectedPackagesCount: 1,
		ProductionAssetsCount: 10,
		DetectedAt:            now.Add(-48 * time.Hour),
		FirstSeenAt:           now.Add(-48 * time.Hour),
		LastSeenAt:            now,
		CreatedAt:             now.Add(-48 * time.Hour),
		UpdatedAt:             now.Add(-12 * time.Hour),
		CVEDetails: &CVECache{
			CVEID:            "CVE-2024-4532",
			CVSSV3Score:      &cvssScore5,
			Severity:         "medium",
			ExploitAvailable: false,
			CISAKEVListed:    false,
			Description:      &desc5,
			PrimarySource:    "osv",
		},
	})

	// Resolved alerts
	cvssScore6 := 6.5
	desc6 := "Cross-site scripting vulnerability in admin panel."
	alerts = append(alerts, CVEAlert{
		ID:                    uuid.New().String(),
		OrgID:                 "default-org",
		CVEID:                 "CVE-2024-2234",
		Severity:              CVESeverityMedium,
		UrgencyScore:          55.0,
		Status:                CVEAlertStatusResolved,
		SLABreached:           false,
		AffectedImagesCount:   2,
		AffectedAssetsCount:   12,
		AffectedPackagesCount: 1,
		ProductionAssetsCount: 4,
		DetectedAt:            now.Add(-72 * time.Hour),
		FirstSeenAt:           now.Add(-72 * time.Hour),
		LastSeenAt:            now.Add(-24 * time.Hour),
		CreatedAt:             now.Add(-72 * time.Hour),
		UpdatedAt:             now.Add(-24 * time.Hour),
		CVEDetails: &CVECache{
			CVEID:            "CVE-2024-2234",
			CVSSV3Score:      &cvssScore6,
			Severity:         "medium",
			ExploitAvailable: false,
			CISAKEVListed:    false,
			Description:      &desc6,
			PrimarySource:    "github",
		},
	})

	return alerts
}

//nolint:unused // retained legacy fixture; superseded by DB-backed handlers
func (h *Handler) generateMockCVEAlert(alertID string) CVEAlert {
	now := time.Now().UTC()
	cvssScore := 9.8
	desc := "A critical remote code execution vulnerability that allows attackers to execute arbitrary code on affected systems."

	return CVEAlert{
		ID:                    alertID,
		OrgID:                 "default-org",
		CVEID:                 "CVE-2024-1234",
		Severity:              CVESeverityCritical,
		UrgencyScore:          92.5,
		Status:                CVEAlertStatusNew,
		SLABreached:           false,
		AffectedImagesCount:   5,
		AffectedAssetsCount:   42,
		AffectedPackagesCount: 3,
		ProductionAssetsCount: 12,
		DetectedAt:            now.Add(-4 * time.Hour),
		FirstSeenAt:           now.Add(-4 * time.Hour),
		LastSeenAt:            now,
		CreatedAt:             now.Add(-4 * time.Hour),
		UpdatedAt:             now,
		CVEDetails: &CVECache{
			CVEID:            "CVE-2024-1234",
			CVSSV3Score:      &cvssScore,
			Severity:         "critical",
			ExploitAvailable: true,
			CISAKEVListed:    true,
			Description:      &desc,
			PrimarySource:    "nvd",
		},
	}
}
