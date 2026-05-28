package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/database"
)

func handlerTestDB(t *testing.T) *database.DB {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Skipf("config load failed: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db, err := database.New(ctx, cfg.Database)
	if err != nil {
		t.Skip("Database not available")
	}
	if err := db.Health(ctx); err != nil {
		db.Close()
		t.Skip("Database not available")
	}
	return db
}

// TestCVEAlertQueries_DBBacked seeds an org with an alert + blast-radius items and
// exercises the real query methods behind the de-mocked CVE alert endpoints.
func TestCVEAlertQueries_DBBacked(t *testing.T) {
	db := handlerTestDB(t)
	defer db.Close()
	pool := db.Pool
	ctx := context.Background()

	suffix := uuid.NewString()[:8]
	orgID := uuid.New()
	imageID := uuid.New()
	sbomID := uuid.New()
	cveID := "CVE-TEST-" + suffix

	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, "DELETE FROM organizations WHERE id = $1", orgID)
		_, _ = pool.Exec(bg, "DELETE FROM cve_cache WHERE cve_id = $1", cveID)
	})

	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("seed exec failed (%s): %v", sql, err)
		}
	}

	exec("INSERT INTO organizations (id, name, slug) VALUES ($1, $2, $3)", orgID, "h-org-"+suffix, "h-org-"+suffix)
	exec("INSERT INTO images (id, org_id, family, version) VALUES ($1, $2, $3, '1.0.0')", imageID, orgID, "h-fam-"+suffix)
	exec("INSERT INTO sboms (id, image_id, org_id, format, version, content) VALUES ($1, $2, $3, 'spdx', 'SPDX-2.3', '{}')", sbomID, imageID, orgID)

	var pkgID uuid.UUID
	if err := pool.QueryRow(ctx,
		"INSERT INTO sbom_packages (sbom_id, name, version, type) VALUES ($1, 'openssl', '3.0.0', 'deb') RETURNING id",
		sbomID,
	).Scan(&pkgID); err != nil {
		t.Fatalf("seed sbom_packages: %v", err)
	}

	var cveCacheID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO cve_cache (cve_id, severity, cvss_v3_score, exploit_available, cisa_kev_listed, primary_source, sources)
		 VALUES ($1, 'critical', 9.8, true, true, 'nvd', '[]') RETURNING id`,
		cveID,
	).Scan(&cveCacheID); err != nil {
		t.Fatalf("seed cve_cache: %v", err)
	}

	var alertID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO cve_alerts (org_id, cve_cache_id, cve_id, severity, urgency_score, status,
		   affected_packages_count, affected_images_count, affected_assets_count, production_assets_count)
		 VALUES ($1, $2, $3, 'critical', 90, 'new', 1, 1, 0, 0) RETURNING id`,
		orgID, cveCacheID, cveID,
	).Scan(&alertID); err != nil {
		t.Fatalf("seed cve_alerts: %v", err)
	}

	exec(`INSERT INTO cve_alert_affected_items (alert_id, package_id, item_type, package_name, package_version, package_type, lineage_depth, item_status)
	      VALUES ($1, $2, 'package', 'openssl', '3.0.0', 'deb', 0, 'vulnerable')`, alertID, pkgID)
	exec(`INSERT INTO cve_alert_affected_items (alert_id, image_id, item_type, image_family, image_version, lineage_depth, item_status)
	      VALUES ($1, $2, 'image', $3, '1.0.0', 0, 'vulnerable')`, alertID, imageID, "h-fam-"+suffix)

	h := &Handler{db: db}
	org := orgID.String()

	// queryCVEAlerts
	alerts, total, err := h.queryCVEAlerts(ctx, cveAlertFilters{OrgID: org, Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("queryCVEAlerts: %v", err)
	}
	if total != 1 || len(alerts) != 1 {
		t.Fatalf("expected exactly 1 alert for org, got total=%d len=%d", total, len(alerts))
	}
	a := alerts[0]
	if a.CVEID != cveID {
		t.Errorf("expected cve_id %s, got %s", cveID, a.CVEID)
	}
	if a.UrgencyScore != 90 {
		t.Errorf("expected urgency 90, got %v", a.UrgencyScore)
	}
	if a.CVEDetails == nil || !a.CVEDetails.ExploitAvailable || !a.CVEDetails.CISAKEVListed {
		t.Errorf("expected joined CVE details with exploit+KEV true, got %+v", a.CVEDetails)
	}

	// filter: has_exploit
	if _, total, err := h.queryCVEAlerts(ctx, cveAlertFilters{OrgID: org, HasExploit: "true", Page: 1, PageSize: 50}); err != nil || total != 1 {
		t.Fatalf("has_exploit filter: total=%d err=%v", total, err)
	}

	// queryCVEAlertByID
	got, err := h.queryCVEAlertByID(ctx, alertID, org)
	if err != nil {
		t.Fatalf("queryCVEAlertByID: %v", err)
	}
	if got.CVEID != cveID {
		t.Errorf("byID: expected %s, got %s", cveID, got.CVEID)
	}

	// not found
	if _, err := h.queryCVEAlertByID(ctx, uuid.New(), org); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("expected pgx.ErrNoRows for missing alert, got %v", err)
	}

	// queryCVEAlertSummary
	summary, err := h.queryCVEAlertSummary(ctx, org)
	if err != nil {
		t.Fatalf("queryCVEAlertSummary: %v", err)
	}
	if summary.TotalAlerts != 1 || summary.CriticalAlerts != 1 || summary.ExploitableAlerts != 1 || summary.CISAKEVAlerts != 1 {
		t.Errorf("unexpected summary: %+v", summary)
	}

	// queryBlastRadius
	br, err := h.queryBlastRadius(ctx, alertID, org)
	if err != nil {
		t.Fatalf("queryBlastRadius: %v", err)
	}
	if br["cve_id"] != cveID {
		t.Errorf("blast radius cve_id mismatch: %v", br["cve_id"])
	}
	if pkgs, ok := br["affected_packages"].([]map[string]any); !ok || len(pkgs) != 1 {
		t.Errorf("expected 1 affected package, got %v", br["affected_packages"])
	}
	if imgs, ok := br["affected_images"].([]map[string]any); !ok || len(imgs) != 1 {
		t.Errorf("expected 1 affected image, got %v", br["affected_images"])
	}

	// execUpdateCVEAlertStatus
	affected, err := h.execUpdateCVEAlertStatus(ctx, alertID, org, UpdateCVEAlertStatusRequest{Status: CVEAlertStatusInProgress})
	if err != nil {
		t.Fatalf("execUpdateCVEAlertStatus: %v", err)
	}
	if affected != 1 {
		t.Fatalf("expected 1 row updated, got %d", affected)
	}
	updated, err := h.queryCVEAlertByID(ctx, alertID, org)
	if err != nil {
		t.Fatalf("re-read after update: %v", err)
	}
	if updated.Status != CVEAlertStatusInProgress {
		t.Errorf("expected status in_progress, got %s", updated.Status)
	}
}
