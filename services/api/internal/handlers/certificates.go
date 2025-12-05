package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
)

// =============================================================================
// Types
// =============================================================================

// Certificate represents a certificate in the system.
type Certificate struct {
	ID                    uuid.UUID         `json:"id"`
	OrgID                 uuid.UUID         `json:"org_id"`
	Fingerprint           string            `json:"fingerprint"`
	SerialNumber          *string           `json:"serial_number,omitempty"`
	CommonName            string            `json:"common_name"`
	SubjectAltNames       []string          `json:"subject_alt_names"`
	Organization          *string           `json:"organization,omitempty"`
	OrganizationalUnit    *string           `json:"organizational_unit,omitempty"`
	Country               *string           `json:"country,omitempty"`
	IssuerCommonName      string            `json:"issuer_common_name"`
	IssuerOrganization    string            `json:"issuer_organization"`
	IsSelfSigned          bool              `json:"is_self_signed"`
	IsCA                  bool              `json:"is_ca"`
	NotBefore             time.Time         `json:"not_before"`
	NotAfter              time.Time         `json:"not_after"`
	DaysUntilExpiry       int               `json:"days_until_expiry"`
	KeyAlgorithm          string            `json:"key_algorithm"`
	KeySize               int               `json:"key_size"`
	SignatureAlgorithm    string            `json:"signature_algorithm"`
	Source                string            `json:"source"`
	SourceRef             string            `json:"source_ref"`
	SourceRegion          *string           `json:"source_region,omitempty"`
	Platform              string            `json:"platform"`
	Status                string            `json:"status"`
	AutoRenew             bool              `json:"auto_renew"`
	RenewalThresholdDays  int               `json:"renewal_threshold_days"`
	LastRotatedAt         *time.Time        `json:"last_rotated_at,omitempty"`
	RotationCount         int               `json:"rotation_count"`
	Tags                  map[string]string `json:"tags"`
	Metadata              map[string]any    `json:"metadata"`
	DiscoveredAt          time.Time         `json:"discovered_at"`
	LastScannedAt         time.Time         `json:"last_scanned_at"`
	CreatedAt             time.Time         `json:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at"`
}

// CertificateUsage represents where a certificate is used.
type CertificateUsage struct {
	ID             uuid.UUID      `json:"id"`
	CertID         uuid.UUID      `json:"cert_id"`
	AssetID        *uuid.UUID     `json:"asset_id,omitempty"`
	UsageType      string         `json:"usage_type"`
	UsageRef       string         `json:"usage_ref"`
	UsagePort      *int           `json:"usage_port,omitempty"`
	Platform       string         `json:"platform"`
	Region         *string        `json:"region,omitempty"`
	ServiceName    *string        `json:"service_name,omitempty"`
	Endpoint       *string        `json:"endpoint,omitempty"`
	Status         string         `json:"status"`
	LastVerifiedAt *time.Time     `json:"last_verified_at,omitempty"`
	TLSVersion     *string        `json:"tls_version,omitempty"`
	Metadata       map[string]any `json:"metadata"`
	DiscoveredAt   time.Time      `json:"discovered_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// CertificateRotation represents a certificate rotation event.
type CertificateRotation struct {
	ID                    uuid.UUID      `json:"id"`
	OrgID                 uuid.UUID      `json:"org_id"`
	OldCertID             *uuid.UUID     `json:"old_cert_id,omitempty"`
	NewCertID             *uuid.UUID     `json:"new_cert_id,omitempty"`
	RotationType          string         `json:"rotation_type"`
	InitiatedBy           string         `json:"initiated_by"`
	InitiatedByUserID     *string        `json:"initiated_by_user_id,omitempty"`
	AITaskID              *uuid.UUID     `json:"ai_task_id,omitempty"`
	AIPlan                map[string]any `json:"ai_plan,omitempty"`
	Status                string         `json:"status"`
	StartedAt             *time.Time     `json:"started_at,omitempty"`
	CompletedAt           *time.Time     `json:"completed_at,omitempty"`
	AffectedUsages        int            `json:"affected_usages"`
	SuccessfulUpdates     int            `json:"successful_updates"`
	FailedUpdates         int            `json:"failed_updates"`
	RollbackAvailable     bool           `json:"rollback_available"`
	RolledBackAt          *time.Time     `json:"rolled_back_at,omitempty"`
	RollbackReason        *string        `json:"rollback_reason,omitempty"`
	PreRotationValidation map[string]any `json:"pre_rotation_validation,omitempty"`
	PostRotationValidation map[string]any `json:"post_rotation_validation,omitempty"`
	ErrorMessage          *string        `json:"error_message,omitempty"`
	ErrorDetails          map[string]any `json:"error_details,omitempty"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

// CertificateAlert represents a certificate alert.
type CertificateAlert struct {
	ID                   uuid.UUID   `json:"id"`
	OrgID                uuid.UUID   `json:"org_id"`
	CertID               uuid.UUID   `json:"cert_id"`
	AlertType            string      `json:"alert_type"`
	Severity             string      `json:"severity"`
	Title                string      `json:"title"`
	Message              string      `json:"message"`
	DaysUntilExpiry      *int        `json:"days_until_expiry,omitempty"`
	ThresholdDays        *int        `json:"threshold_days,omitempty"`
	Status               string      `json:"status"`
	AcknowledgedAt       *time.Time  `json:"acknowledged_at,omitempty"`
	AcknowledgedBy       *string     `json:"acknowledged_by,omitempty"`
	ResolvedAt           *time.Time  `json:"resolved_at,omitempty"`
	AutoRotationTriggered bool       `json:"auto_rotation_triggered"`
	RotationID           *uuid.UUID  `json:"rotation_id,omitempty"`
	NotificationsSent    []any       `json:"notifications_sent"`
	CreatedAt            time.Time   `json:"created_at"`
	UpdatedAt            time.Time   `json:"updated_at"`
}

// CertificateSummary represents a summary of certificate status.
type CertificateSummary struct {
	TotalCertificates   int `json:"total_certificates"`
	ActiveCertificates  int `json:"active_certificates"`
	ExpiringSoon        int `json:"expiring_soon"`
	Expired             int `json:"expired"`
	Expiring7Days       int `json:"expiring_7_days"`
	Expiring30Days      int `json:"expiring_30_days"`
	Expiring90Days      int `json:"expiring_90_days"`
	AutoRenewEnabled    int `json:"auto_renew_enabled"`
	SelfSigned          int `json:"self_signed"`
	PlatformsCount      int `json:"platforms_count"`
}

// CertificateListResponse represents the response for listing certificates.
type CertificateListResponse struct {
	Certificates []Certificate `json:"certificates"`
	Total        int           `json:"total"`
	Page         int           `json:"page"`
	PageSize     int           `json:"page_size"`
}

// =============================================================================
// Handler
// =============================================================================

// CertificateHandler handles certificate-related HTTP requests.
type CertificateHandler struct {
	db  *pgxpool.Pool
	log *logger.Logger
}

// NewCertificateHandler creates a new certificate handler.
func NewCertificateHandler(db *pgxpool.Pool, log *logger.Logger) *CertificateHandler {
	return &CertificateHandler{
		db:  db,
		log: log.WithComponent("certificate-handler"),
	}
}

// =============================================================================
// Certificate Endpoints
// =============================================================================

// ListCertificates lists certificates with optional filters.
// GET /api/v1/certificates
func (h *CertificateHandler) ListCertificates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	status := r.URL.Query().Get("status")
	platform := r.URL.Query().Get("platform")
	expiringDays := r.URL.Query().Get("expiring_within_days")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// Build query
	query := `
		SELECT id, org_id, fingerprint, serial_number, common_name, subject_alt_names,
		       organization, organizational_unit, country,
		       issuer_common_name, issuer_organization, is_self_signed, is_ca,
		       not_before, not_after, days_until_expiry,
		       key_algorithm, key_size, signature_algorithm,
		       source, source_ref, source_region, platform,
		       status, auto_renew, renewal_threshold_days, last_rotated_at, rotation_count,
		       tags, metadata, discovered_at, last_scanned_at, created_at, updated_at
		FROM certificates
		WHERE org_id = $1
	`
	args := []interface{}{org.ID}
	argIdx := 2

	if status != "" {
		query += " AND status = $" + strconv.Itoa(argIdx)
		args = append(args, status)
		argIdx++
	}
	if platform != "" {
		query += " AND platform = $" + strconv.Itoa(argIdx)
		args = append(args, platform)
		argIdx++
	}
	if expiringDays != "" {
		days, err := strconv.Atoi(expiringDays)
		if err == nil && days > 0 {
			query += " AND days_until_expiry <= $" + strconv.Itoa(argIdx) + " AND days_until_expiry >= 0"
			args = append(args, days)
			argIdx++
		}
	}

	query += " ORDER BY days_until_expiry ASC LIMIT $" + strconv.Itoa(argIdx) + " OFFSET $" + strconv.Itoa(argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		h.log.Error("failed to query certificates", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	certificates := []Certificate{}
	for rows.Next() {
		var cert Certificate
		err := rows.Scan(
			&cert.ID, &cert.OrgID, &cert.Fingerprint, &cert.SerialNumber,
			&cert.CommonName, &cert.SubjectAltNames,
			&cert.Organization, &cert.OrganizationalUnit, &cert.Country,
			&cert.IssuerCommonName, &cert.IssuerOrganization, &cert.IsSelfSigned, &cert.IsCA,
			&cert.NotBefore, &cert.NotAfter, &cert.DaysUntilExpiry,
			&cert.KeyAlgorithm, &cert.KeySize, &cert.SignatureAlgorithm,
			&cert.Source, &cert.SourceRef, &cert.SourceRegion, &cert.Platform,
			&cert.Status, &cert.AutoRenew, &cert.RenewalThresholdDays, &cert.LastRotatedAt, &cert.RotationCount,
			&cert.Tags, &cert.Metadata, &cert.DiscoveredAt, &cert.LastScannedAt, &cert.CreatedAt, &cert.UpdatedAt,
		)
		if err != nil {
			h.log.Error("failed to scan certificate", "error", err)
			continue
		}
		certificates = append(certificates, cert)
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM certificates WHERE org_id = $1"
	h.db.QueryRow(ctx, countQuery, org.ID).Scan(&total)

	respondJSON(w, http.StatusOK, CertificateListResponse{
		Certificates: certificates,
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
	})
}

// GetCertificate retrieves a specific certificate by ID.
// GET /api/v1/certificates/{id}
func (h *CertificateHandler) GetCertificate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	certID := chi.URLParam(r, "id")
	id, err := uuid.Parse(certID)
	if err != nil {
		http.Error(w, "invalid certificate ID", http.StatusBadRequest)
		return
	}

	cert, err := h.getCertificateByID(ctx, id, org.ID)
	if err != nil {
		h.log.Error("failed to get certificate", "error", err, "id", id)
		http.Error(w, "certificate not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, cert)
}

// GetCertificateSummary retrieves certificate statistics for the organization.
// GET /api/v1/certificates/summary
func (h *CertificateHandler) GetCertificateSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	var summary CertificateSummary
	err := h.db.QueryRow(ctx, `
		SELECT
			COUNT(*) as total_certificates,
			COUNT(*) FILTER (WHERE status = 'active') as active_certificates,
			COUNT(*) FILTER (WHERE status = 'expiring_soon') as expiring_soon,
			COUNT(*) FILTER (WHERE status = 'expired') as expired,
			COUNT(*) FILTER (WHERE days_until_expiry <= 7 AND days_until_expiry >= 0) as expiring_7_days,
			COUNT(*) FILTER (WHERE days_until_expiry <= 30 AND days_until_expiry >= 0) as expiring_30_days,
			COUNT(*) FILTER (WHERE days_until_expiry <= 90 AND days_until_expiry >= 0) as expiring_90_days,
			COUNT(*) FILTER (WHERE auto_renew = true) as auto_renew_enabled,
			COUNT(*) FILTER (WHERE is_self_signed = true) as self_signed,
			COUNT(DISTINCT platform) as platforms_count
		FROM certificates
		WHERE org_id = $1
	`, org.ID).Scan(
		&summary.TotalCertificates,
		&summary.ActiveCertificates,
		&summary.ExpiringSoon,
		&summary.Expired,
		&summary.Expiring7Days,
		&summary.Expiring30Days,
		&summary.Expiring90Days,
		&summary.AutoRenewEnabled,
		&summary.SelfSigned,
		&summary.PlatformsCount,
	)
	if err != nil {
		h.log.Error("failed to get certificate summary", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, summary)
}

// GetCertificateUsage retrieves usage locations for a certificate.
// GET /api/v1/certificates/{id}/usage
func (h *CertificateHandler) GetCertificateUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	certID := chi.URLParam(r, "id")
	id, err := uuid.Parse(certID)
	if err != nil {
		http.Error(w, "invalid certificate ID", http.StatusBadRequest)
		return
	}

	// Verify certificate belongs to org
	var certOrgID uuid.UUID
	err = h.db.QueryRow(ctx, "SELECT org_id FROM certificates WHERE id = $1", id).Scan(&certOrgID)
	if err != nil || certOrgID != org.ID {
		http.Error(w, "certificate not found", http.StatusNotFound)
		return
	}

	// Get usage
	rows, err := h.db.Query(ctx, `
		SELECT id, cert_id, asset_id, usage_type, usage_ref, usage_port,
		       platform, region, service_name, endpoint, status,
		       last_verified_at, tls_version, metadata, discovered_at, created_at, updated_at
		FROM certificate_usage
		WHERE cert_id = $1
		ORDER BY usage_type, service_name
	`, id)
	if err != nil {
		h.log.Error("failed to query certificate usage", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	usages := []CertificateUsage{}
	for rows.Next() {
		var usage CertificateUsage
		err := rows.Scan(
			&usage.ID, &usage.CertID, &usage.AssetID, &usage.UsageType, &usage.UsageRef, &usage.UsagePort,
			&usage.Platform, &usage.Region, &usage.ServiceName, &usage.Endpoint, &usage.Status,
			&usage.LastVerifiedAt, &usage.TLSVersion, &usage.Metadata, &usage.DiscoveredAt, &usage.CreatedAt, &usage.UpdatedAt,
		)
		if err != nil {
			h.log.Error("failed to scan usage", "error", err)
			continue
		}
		usages = append(usages, usage)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"usages":       usages,
		"total_usages": len(usages),
	})
}

// =============================================================================
// Rotation Endpoints
// =============================================================================

// ListRotations lists certificate rotations.
// GET /api/v1/certificates/rotations
func (h *CertificateHandler) ListRotations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	status := r.URL.Query().Get("status")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query := `
		SELECT id, org_id, old_cert_id, new_cert_id, rotation_type, initiated_by,
		       initiated_by_user_id, ai_task_id, ai_plan, status, started_at, completed_at,
		       affected_usages, successful_updates, failed_updates, rollback_available,
		       rolled_back_at, rollback_reason, pre_rotation_validation, post_rotation_validation,
		       error_message, error_details, created_at, updated_at
		FROM certificate_rotations
		WHERE org_id = $1
	`
	args := []interface{}{org.ID}
	argIdx := 2

	if status != "" {
		query += " AND status = $" + strconv.Itoa(argIdx)
		args = append(args, status)
		argIdx++
	}

	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(argIdx) + " OFFSET $" + strconv.Itoa(argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		h.log.Error("failed to query rotations", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	rotations := []CertificateRotation{}
	for rows.Next() {
		var rot CertificateRotation
		err := rows.Scan(
			&rot.ID, &rot.OrgID, &rot.OldCertID, &rot.NewCertID, &rot.RotationType, &rot.InitiatedBy,
			&rot.InitiatedByUserID, &rot.AITaskID, &rot.AIPlan, &rot.Status, &rot.StartedAt, &rot.CompletedAt,
			&rot.AffectedUsages, &rot.SuccessfulUpdates, &rot.FailedUpdates, &rot.RollbackAvailable,
			&rot.RolledBackAt, &rot.RollbackReason, &rot.PreRotationValidation, &rot.PostRotationValidation,
			&rot.ErrorMessage, &rot.ErrorDetails, &rot.CreatedAt, &rot.UpdatedAt,
		)
		if err != nil {
			h.log.Error("failed to scan rotation", "error", err)
			continue
		}
		rotations = append(rotations, rot)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"rotations": rotations,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetRotation retrieves a specific rotation by ID.
// GET /api/v1/certificates/rotations/{id}
func (h *CertificateHandler) GetRotation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	rotID := chi.URLParam(r, "id")
	id, err := uuid.Parse(rotID)
	if err != nil {
		http.Error(w, "invalid rotation ID", http.StatusBadRequest)
		return
	}

	var rot CertificateRotation
	err = h.db.QueryRow(ctx, `
		SELECT id, org_id, old_cert_id, new_cert_id, rotation_type, initiated_by,
		       initiated_by_user_id, ai_task_id, ai_plan, status, started_at, completed_at,
		       affected_usages, successful_updates, failed_updates, rollback_available,
		       rolled_back_at, rollback_reason, pre_rotation_validation, post_rotation_validation,
		       error_message, error_details, created_at, updated_at
		FROM certificate_rotations
		WHERE id = $1 AND org_id = $2
	`, id, org.ID).Scan(
		&rot.ID, &rot.OrgID, &rot.OldCertID, &rot.NewCertID, &rot.RotationType, &rot.InitiatedBy,
		&rot.InitiatedByUserID, &rot.AITaskID, &rot.AIPlan, &rot.Status, &rot.StartedAt, &rot.CompletedAt,
		&rot.AffectedUsages, &rot.SuccessfulUpdates, &rot.FailedUpdates, &rot.RollbackAvailable,
		&rot.RolledBackAt, &rot.RollbackReason, &rot.PreRotationValidation, &rot.PostRotationValidation,
		&rot.ErrorMessage, &rot.ErrorDetails, &rot.CreatedAt, &rot.UpdatedAt,
	)
	if err != nil {
		http.Error(w, "rotation not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, rot)
}

// =============================================================================
// Alert Endpoints
// =============================================================================

// ListAlerts lists certificate alerts.
// GET /api/v1/certificates/alerts
func (h *CertificateHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	status := r.URL.Query().Get("status")
	severity := r.URL.Query().Get("severity")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query := `
		SELECT id, org_id, cert_id, alert_type, severity, title, message,
		       days_until_expiry, threshold_days, status, acknowledged_at, acknowledged_by,
		       resolved_at, auto_rotation_triggered, rotation_id, notifications_sent,
		       created_at, updated_at
		FROM certificate_alerts
		WHERE org_id = $1
	`
	args := []interface{}{org.ID}
	argIdx := 2

	if status != "" {
		query += " AND status = $" + strconv.Itoa(argIdx)
		args = append(args, status)
		argIdx++
	}
	if severity != "" {
		query += " AND severity = $" + strconv.Itoa(argIdx)
		args = append(args, severity)
		argIdx++
	}

	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(argIdx) + " OFFSET $" + strconv.Itoa(argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		h.log.Error("failed to query alerts", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	alerts := []CertificateAlert{}
	for rows.Next() {
		var alert CertificateAlert
		err := rows.Scan(
			&alert.ID, &alert.OrgID, &alert.CertID, &alert.AlertType, &alert.Severity,
			&alert.Title, &alert.Message, &alert.DaysUntilExpiry, &alert.ThresholdDays,
			&alert.Status, &alert.AcknowledgedAt, &alert.AcknowledgedBy, &alert.ResolvedAt,
			&alert.AutoRotationTriggered, &alert.RotationID, &alert.NotificationsSent,
			&alert.CreatedAt, &alert.UpdatedAt,
		)
		if err != nil {
			h.log.Error("failed to scan alert", "error", err)
			continue
		}
		alerts = append(alerts, alert)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"alerts":    alerts,
		"page":      page,
		"page_size": pageSize,
	})
}

// AcknowledgeAlert acknowledges a certificate alert.
// POST /api/v1/certificates/alerts/{id}/acknowledge
func (h *CertificateHandler) AcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	alertID := chi.URLParam(r, "id")
	id, err := uuid.Parse(alertID)
	if err != nil {
		http.Error(w, "invalid alert ID", http.StatusBadRequest)
		return
	}

	// Get user ID from context (if available)
	userID := "system"
	if user := middleware.GetUser(ctx); user != nil {
		userID = user.ID.String()
	}

	result, err := h.db.Exec(ctx, `
		UPDATE certificate_alerts
		SET status = 'acknowledged', acknowledged_at = NOW(), acknowledged_by = $1, updated_at = NOW()
		WHERE id = $2 AND org_id = $3 AND status = 'open'
	`, userID, id, org.ID)
	if err != nil {
		h.log.Error("failed to acknowledge alert", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if result.RowsAffected() == 0 {
		http.Error(w, "alert not found or already acknowledged", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "acknowledged",
	})
}

// =============================================================================
// Helper Methods
// =============================================================================

func (h *CertificateHandler) getCertificateByID(ctx context.Context, id, orgID uuid.UUID) (*Certificate, error) {
	var cert Certificate
	err := h.db.QueryRow(ctx, `
		SELECT id, org_id, fingerprint, serial_number, common_name, subject_alt_names,
		       organization, organizational_unit, country,
		       issuer_common_name, issuer_organization, is_self_signed, is_ca,
		       not_before, not_after, days_until_expiry,
		       key_algorithm, key_size, signature_algorithm,
		       source, source_ref, source_region, platform,
		       status, auto_renew, renewal_threshold_days, last_rotated_at, rotation_count,
		       tags, metadata, discovered_at, last_scanned_at, created_at, updated_at
		FROM certificates
		WHERE id = $1 AND org_id = $2
	`, id, orgID).Scan(
		&cert.ID, &cert.OrgID, &cert.Fingerprint, &cert.SerialNumber,
		&cert.CommonName, &cert.SubjectAltNames,
		&cert.Organization, &cert.OrganizationalUnit, &cert.Country,
		&cert.IssuerCommonName, &cert.IssuerOrganization, &cert.IsSelfSigned, &cert.IsCA,
		&cert.NotBefore, &cert.NotAfter, &cert.DaysUntilExpiry,
		&cert.KeyAlgorithm, &cert.KeySize, &cert.SignatureAlgorithm,
		&cert.Source, &cert.SourceRef, &cert.SourceRegion, &cert.Platform,
		&cert.Status, &cert.AutoRenew, &cert.RenewalThresholdDays, &cert.LastRotatedAt, &cert.RotationCount,
		&cert.Tags, &cert.Metadata, &cert.DiscoveredAt, &cert.LastScannedAt, &cert.CreatedAt, &cert.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
