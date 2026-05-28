package handlers

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// This file contains the real (database-backed) query logic for CVE alerts.
// The HTTP handlers in cve_alerts.go call these methods; the mock generators in
// that file are retained (legacy) but no longer used by the live endpoints.

// cveAlertColumns is the SELECT list (aliased ca + LEFT JOINed cve_cache as cc)
// used by all alert reads. Its order must match scanCVEAlertRow.
const cveAlertColumns = `
	ca.id, ca.org_id, ca.cve_id, ca.cve_cache_id, ca.severity, ca.urgency_score, ca.status,
	ca.priority, ca.sla_due_at, ca.sla_breached,
	ca.affected_images_count, ca.affected_assets_count, ca.affected_packages_count, ca.production_assets_count,
	ca.assigned_to, ca.assigned_at, ca.resolution_type, ca.resolution_notes, ca.resolved_by, ca.resolved_at,
	ca.patch_campaign_id, ca.ticket_id,
	ca.detected_at, ca.first_seen_at, ca.last_seen_at, ca.created_at, ca.updated_at,
	cc.id, cc.cve_id, cc.cvss_v3_score, cc.cvss_v3_vector, cc.severity, cc.epss_score, cc.epss_percentile,
	cc.exploit_available, cc.exploit_maturity, cc.cisa_kev_listed, cc.description, cc.primary_source`

const cveAlertFromJoin = ` FROM cve_alerts ca LEFT JOIN cve_cache cc ON cc.id = ca.cve_cache_id`

// cveAlertFilters holds the parsed list/query filters.
type cveAlertFilters struct {
	OrgID           string
	Severity        string
	Status          string
	Priority        string
	CVEID           string
	MinUrgencyScore string
	SLABreached     string
	HasExploit      string
	CISAKEVOnly     string
	Page            int
	PageSize        int
}

// orgUUID parses an org id string; ok is false when empty/invalid (no org scoping).
func orgUUID(s string) (uuid.UUID, bool) {
	if s == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// scanCVEAlertRow scans a row (from QueryRow or Rows) selected with cveAlertColumns
// into the handler's CVEAlert response type.
func scanCVEAlertRow(scan func(dest ...any) error) (CVEAlert, error) {
	var (
		a               CVEAlert
		id              uuid.UUID
		orgID           uuid.UUID
		cveCacheID      *uuid.UUID
		severity        string
		status          string
		patchCampaignID *uuid.UUID

		ccID       *uuid.UUID
		ccCVEID    *string
		ccCVSS     *float64
		ccVector   *string
		ccSeverity *string
		ccEPSS     *float64
		ccEPSSPct  *float64
		ccExploit  *bool
		ccMaturity *string
		ccKEV      *bool
		ccDesc     *string
		ccSource   *string
	)

	err := scan(
		&id, &orgID, &a.CVEID, &cveCacheID, &severity, &a.UrgencyScore, &status,
		&a.Priority, &a.SLADueAt, &a.SLABreached,
		&a.AffectedImagesCount, &a.AffectedAssetsCount, &a.AffectedPackagesCount, &a.ProductionAssetsCount,
		&a.AssignedTo, &a.AssignedAt, &a.ResolutionType, &a.ResolutionNotes, &a.ResolvedBy, &a.ResolvedAt,
		&patchCampaignID, &a.TicketID,
		&a.DetectedAt, &a.FirstSeenAt, &a.LastSeenAt, &a.CreatedAt, &a.UpdatedAt,
		&ccID, &ccCVEID, &ccCVSS, &ccVector, &ccSeverity, &ccEPSS, &ccEPSSPct,
		&ccExploit, &ccMaturity, &ccKEV, &ccDesc, &ccSource,
	)
	if err != nil {
		return a, err
	}

	a.ID = id.String()
	a.OrgID = orgID.String()
	if cveCacheID != nil {
		s := cveCacheID.String()
		a.CVECacheID = &s
	}
	a.Severity = CVESeverity(severity)
	a.Status = CVEAlertStatus(status)
	if patchCampaignID != nil {
		s := patchCampaignID.String()
		a.PatchCampaignID = &s
	}

	if ccID != nil {
		cd := &CVECache{ID: ccID.String()}
		if ccCVEID != nil {
			cd.CVEID = *ccCVEID
		}
		cd.CVSSV3Score = ccCVSS
		cd.CVSSV3Vector = ccVector
		if ccSeverity != nil {
			cd.Severity = *ccSeverity
		}
		cd.EPSSScore = ccEPSS
		cd.EPSSPercentile = ccEPSSPct
		if ccExploit != nil {
			cd.ExploitAvailable = *ccExploit
		}
		cd.ExploitMaturity = ccMaturity
		if ccKEV != nil {
			cd.CISAKEVListed = *ccKEV
		}
		cd.Description = ccDesc
		if ccSource != nil {
			cd.PrimarySource = *ccSource
		}
		a.CVEDetails = cd
	}

	return a, nil
}

// buildAlertWhere builds the WHERE clause and args shared by the list and count
// queries from the supplied filters.
func buildAlertWhere(f cveAlertFilters) (string, []any) {
	var conds []string
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}

	if oid, ok := orgUUID(f.OrgID); ok {
		conds = append(conds, "ca.org_id = "+arg(oid))
	}
	if f.Severity != "" {
		conds = append(conds, "ca.severity = "+arg(f.Severity))
	}
	if f.Status != "" {
		conds = append(conds, "ca.status = "+arg(f.Status))
	}
	if f.Priority != "" {
		conds = append(conds, "ca.priority = "+arg(f.Priority))
	}
	if f.CVEID != "" {
		conds = append(conds, "ca.cve_id = "+arg(f.CVEID))
	}
	if f.MinUrgencyScore != "" {
		if n, err := strconv.Atoi(f.MinUrgencyScore); err == nil {
			conds = append(conds, "ca.urgency_score >= "+arg(n))
		}
	}
	switch f.SLABreached {
	case "true":
		conds = append(conds, "ca.sla_breached = "+arg(true))
	case "false":
		conds = append(conds, "ca.sla_breached = "+arg(false))
	}
	if f.HasExploit == "true" {
		conds = append(conds, "cc.exploit_available = "+arg(true))
	}
	if f.CISAKEVOnly == "true" {
		conds = append(conds, "cc.cisa_kev_listed = "+arg(true))
	}

	if len(conds) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// queryCVEAlerts returns a filtered, paginated page of alerts plus the total count.
func (h *Handler) queryCVEAlerts(ctx context.Context, f cveAlertFilters) ([]CVEAlert, int, error) {
	where, args := buildAlertWhere(f)

	var total int
	if err := h.db.Pool.QueryRow(ctx, "SELECT count(*)"+cveAlertFromJoin+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitArg := "$" + strconv.Itoa(len(args)+1)
	offsetArg := "$" + strconv.Itoa(len(args)+2)
	query := "SELECT" + cveAlertColumns + cveAlertFromJoin + where +
		" ORDER BY ca.urgency_score DESC, ca.detected_at DESC LIMIT " + limitArg + " OFFSET " + offsetArg
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)

	rows, err := h.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	alerts := []CVEAlert{}
	for rows.Next() {
		a, err := scanCVEAlertRow(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		alerts = append(alerts, a)
	}
	return alerts, total, rows.Err()
}

// queryCVEAlertByID fetches a single alert, optionally scoped to an org.
// Returns pgx.ErrNoRows when not found.
func (h *Handler) queryCVEAlertByID(ctx context.Context, alertID uuid.UUID, orgID string) (CVEAlert, error) {
	query := "SELECT" + cveAlertColumns + cveAlertFromJoin + " WHERE ca.id = $1"
	args := []any{alertID}
	if oid, ok := orgUUID(orgID); ok {
		query += " AND ca.org_id = $2"
		args = append(args, oid)
	}
	return scanCVEAlertRow(h.db.Pool.QueryRow(ctx, query, args...).Scan)
}

// queryCVEAlertSummary aggregates alert statistics, optionally scoped to an org.
func (h *Handler) queryCVEAlertSummary(ctx context.Context, orgID string) (CVEAlertSummary, error) {
	query := `
		SELECT
			count(*),
			count(*) FILTER (WHERE ca.status = 'new'),
			count(*) FILTER (WHERE ca.status = 'in_progress'),
			count(*) FILTER (WHERE ca.status IN ('resolved', 'auto_resolved')),
			count(*) FILTER (WHERE ca.severity = 'critical'),
			count(*) FILTER (WHERE ca.severity = 'high'),
			count(*) FILTER (WHERE ca.severity = 'medium'),
			count(*) FILTER (WHERE ca.severity = 'low'),
			count(*) FILTER (WHERE ca.sla_breached),
			count(*) FILTER (WHERE cc.exploit_available),
			count(*) FILTER (WHERE cc.cisa_kev_listed),
			COALESCE(AVG(ca.urgency_score), 0)::float8,
			COALESCE(SUM(ca.affected_assets_count), 0)::bigint,
			COALESCE(SUM(ca.production_assets_count), 0)::bigint
		` + cveAlertFromJoin
	var args []any
	if oid, ok := orgUUID(orgID); ok {
		query += " WHERE ca.org_id = $1"
		args = append(args, oid)
	}

	var s CVEAlertSummary
	err := h.db.Pool.QueryRow(ctx, query, args...).Scan(
		&s.TotalAlerts, &s.NewAlerts, &s.InProgressAlerts, &s.ResolvedAlerts,
		&s.CriticalAlerts, &s.HighAlerts, &s.MediumAlerts, &s.LowAlerts,
		&s.SLABreachedAlerts, &s.ExploitableAlerts, &s.CISAKEVAlerts,
		&s.AverageUrgencyScore, &s.TotalAffectedAssets, &s.ProductionAffectedAssets,
	)
	return s, err
}

// execUpdateCVEAlertStatus updates an alert's status/assignment fields. Returns
// the number of rows affected (0 = not found / not in org).
func (h *Handler) execUpdateCVEAlertStatus(ctx context.Context, alertID uuid.UUID, orgID string, req UpdateCVEAlertStatusRequest) (int64, error) {
	query := `
		UPDATE cve_alerts SET
			status = $2::text,
			assigned_to = COALESCE($3, assigned_to),
			resolution_type = COALESCE($4, resolution_type),
			resolution_notes = COALESCE($5, resolution_notes),
			ticket_id = COALESCE($6, ticket_id),
			resolved_at = CASE WHEN $2::text IN ('resolved', 'auto_resolved') THEN NOW() ELSE resolved_at END,
			updated_at = NOW()
		WHERE id = $1`
	args := []any{alertID, string(req.Status), req.AssignedTo, req.ResolutionType, req.ResolutionNotes, req.TicketID}
	if oid, ok := orgUUID(orgID); ok {
		query += " AND org_id = $7"
		args = append(args, oid)
	}

	tag, err := h.db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// queryBlastRadius reads the stored blast-radius details (affected packages,
// images, assets) for an alert and returns them in the dashboard's shape.
// Returns pgx.ErrNoRows when the alert is not found.
func (h *Handler) queryBlastRadius(ctx context.Context, alertID uuid.UUID, orgID string) (map[string]any, error) {
	headerQuery := `
		SELECT cve_id, urgency_score, affected_packages_count, affected_images_count,
		       affected_assets_count, production_assets_count
		FROM cve_alerts WHERE id = $1`
	headerArgs := []any{alertID}
	if oid, ok := orgUUID(orgID); ok {
		headerQuery += " AND org_id = $2"
		headerArgs = append(headerArgs, oid)
	}

	var (
		cveID                                           string
		urgency                                         float64
		totalPkgs, totalImages, totalAssets, prodAssets int
	)
	if err := h.db.Pool.QueryRow(ctx, headerQuery, headerArgs...).Scan(
		&cveID, &urgency, &totalPkgs, &totalImages, &totalAssets, &prodAssets,
	); err != nil {
		return nil, err
	}

	rows, err := h.db.Pool.Query(ctx, `
		SELECT item_type, package_id, package_name, package_version, package_type, fixed_version,
		       image_id, image_family, image_version, lineage_depth,
		       asset_id, asset_name, asset_platform, asset_region, asset_environment, is_production
		FROM cve_alert_affected_items
		WHERE alert_id = $1`, alertID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	packages := []map[string]any{}
	images := []map[string]any{}
	assets := []map[string]any{}
	platformSet := map[string]struct{}{}
	regionSet := map[string]struct{}{}

	for rows.Next() {
		var (
			itemType                                               string
			packageID                                              *uuid.UUID
			packageName, packageVersion, packageType, fixedVersion *string
			imageID                                                *uuid.UUID
			imageFamily, imageVersion                              *string
			lineageDepth                                           int
			assetID                                                *uuid.UUID
			assetName, assetPlatform, assetRegion, assetEnv        *string
			isProduction                                           bool
		)
		if err := rows.Scan(
			&itemType, &packageID, &packageName, &packageVersion, &packageType, &fixedVersion,
			&imageID, &imageFamily, &imageVersion, &lineageDepth,
			&assetID, &assetName, &assetPlatform, &assetRegion, &assetEnv, &isProduction,
		); err != nil {
			return nil, err
		}

		switch itemType {
		case "package":
			packages = append(packages, map[string]any{
				"package_id":      uuidStr(packageID),
				"package_name":    strVal(packageName),
				"package_version": strVal(packageVersion),
				"package_type":    strVal(packageType),
				"fixed_version":   strVal(fixedVersion),
			})
		case "image":
			images = append(images, map[string]any{
				"image_id":      uuidStr(imageID),
				"image_family":  strVal(imageFamily),
				"image_version": strVal(imageVersion),
				"is_direct":     lineageDepth == 0,
				"lineage_depth": lineageDepth,
			})
		case "asset":
			assets = append(assets, map[string]any{
				"asset_id":      uuidStr(assetID),
				"asset_name":    strVal(assetName),
				"platform":      strVal(assetPlatform),
				"region":        strVal(assetRegion),
				"environment":   strVal(assetEnv),
				"is_production": isProduction,
			})
			if assetPlatform != nil && *assetPlatform != "" {
				platformSet[*assetPlatform] = struct{}{}
			}
			if assetRegion != nil && *assetRegion != "" {
				regionSet[*assetRegion] = struct{}{}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return map[string]any{
		"alert_id":           alertID.String(),
		"cve_id":             cveID,
		"total_packages":     totalPkgs,
		"total_images":       totalImages,
		"total_assets":       totalAssets,
		"production_assets":  prodAssets,
		"affected_platforms": keys(platformSet),
		"affected_regions":   keys(regionSet),
		"affected_packages":  packages,
		"affected_images":    images,
		"affected_assets":    assets,
		"urgency_score":      urgency,
		"calculated_at":      time.Now().UTC(),
	}, nil
}

func uuidStr(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func keys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}
