package vulnmatch

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/database"
)

// TestStringVersionComparison_LimitationDocumented documents the known
// limitation flagged by the TODO(semver) in findCandidates: lexical string
// comparison (used by the version_constraint SQL) misorders semantic versions.
// This test will need to be updated when a semver-aware comparator replaces it.
func TestStringVersionComparison_LimitationDocumented(t *testing.T) {
	// Lexically "1.10" < "1.9" is true, but semantically 1.10 > 1.9.
	if !("1.10" < "1.9") {
		t.Fatal("expected lexical string comparison to (incorrectly) order 1.10 before 1.9; " +
			"if this changed, the version comparison strategy may have been fixed — update findCandidates and remove the TODO(semver)")
	}
}

func testDB(t *testing.T) *database.DB {
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

// TestScanAndAlert_CreatesOneAlertIdempotent seeds an org with one image/SBOM/
// package and a matching CVE, then verifies ScanAndAlert creates exactly one
// alert with blast-radius items, and that running it again does not duplicate
// alerts or affected items.
func TestScanAndAlert_CreatesOneAlertIdempotent(t *testing.T) {
	db := testDB(t)
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
		// Deleting the org cascades images, sboms, sbom_packages and cve_alerts.
		_, _ = pool.Exec(bg, "DELETE FROM organizations WHERE id = $1", orgID)
		// cve_package_matches cascades from cve_cache.
		_, _ = pool.Exec(bg, "DELETE FROM cve_cache WHERE cve_id = $1", cveID)
	})

	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("seed exec failed (%s): %v", sql, err)
		}
	}

	mustExec("INSERT INTO organizations (id, name, slug) VALUES ($1, $2, $3)", orgID, "test-org-"+suffix, "test-org-"+suffix)
	mustExec("INSERT INTO images (id, org_id, family, version) VALUES ($1, $2, $3, $4)", imageID, orgID, "test-family-"+suffix, "1.0.0")
	mustExec("INSERT INTO sboms (id, image_id, org_id, format, version, content) VALUES ($1, $2, $3, 'spdx', 'SPDX-2.3', '{}')", sbomID, imageID, orgID)
	mustExec("INSERT INTO sbom_packages (sbom_id, name, version, type) VALUES ($1, 'openssl', '3.0.0', 'deb')", sbomID)

	var cveCacheID uuid.UUID
	if err := pool.QueryRow(ctx,
		"INSERT INTO cve_cache (cve_id, severity, primary_source, sources) VALUES ($1, 'critical', 'test', '[]') RETURNING id",
		cveID,
	).Scan(&cveCacheID); err != nil {
		t.Fatalf("seed cve_cache: %v", err)
	}
	mustExec("INSERT INTO cve_package_matches (cve_cache_id, package_name, version_constraint) VALUES ($1, 'openssl', 'all')", cveCacheID)

	svc := NewService(pool, slog.Default())

	countAlerts := func() int {
		var n int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM cve_alerts WHERE org_id = $1 AND cve_id = $2", orgID, cveID).Scan(&n); err != nil {
			t.Fatalf("count alerts: %v", err)
		}
		return n
	}
	countItems := func() int {
		var n int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM cve_alert_affected_items cai
			 JOIN cve_alerts ca ON ca.id = cai.alert_id
			 WHERE ca.org_id = $1 AND ca.cve_id = $2`, orgID, cveID).Scan(&n); err != nil {
			t.Fatalf("count items: %v", err)
		}
		return n
	}

	// First pass.
	if _, err := svc.ScanAndAlert(ctx); err != nil {
		t.Fatalf("first ScanAndAlert: %v", err)
	}
	if got := countAlerts(); got != 1 {
		t.Fatalf("expected 1 alert after first scan, got %d", got)
	}
	itemsFirst := countItems()
	if itemsFirst == 0 {
		t.Fatalf("expected blast-radius affected items after first scan, got 0")
	}

	// Second pass — must be idempotent.
	if _, err := svc.ScanAndAlert(ctx); err != nil {
		t.Fatalf("second ScanAndAlert: %v", err)
	}
	if got := countAlerts(); got != 1 {
		t.Fatalf("expected still 1 alert after second scan, got %d", got)
	}
	if got := countItems(); got != itemsFirst {
		t.Fatalf("affected items not idempotent: %d after first, %d after second", itemsFirst, got)
	}
}
