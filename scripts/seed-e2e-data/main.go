// Command seed-e2e-data populates a deterministic fixture for the end-to-end
// (Playwright) test suite. It is intentionally separate from the human-facing
// demo seed (multitenancy.SeedDemoData): this data is fixed and predictable so
// E2E specs can assert stable facts rather than vague "something rendered".
//
// It is idempotent — it deletes the fixed E2E org (cascading to its children)
// and re-inserts a clean fixture on every run. All IDs are fixed, and the org's
// created_at is pinned to the past so the API's dev-mode "first organization"
// resolution (lookupDefaultOrg) always selects it.
//
// Usage: RF_DATABASE_URL=... go run ./scripts/seed-e2e-data
//
// Scope: this currently seeds the entities the Overview dashboard needs (org,
// sites, assets, images, drift reports, alerts). Risk items, CVE alerts,
// compliance results, and certificates are intentionally deferred until the
// corresponding page specs are tackled (see docs/E2E-001-deterministic-fullstack-e2e.md).
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/database"
)

// Fixed identifiers for the E2E fixture. Keep these stable: specs and the
// dev-mode org resolution depend on them.
const (
	orgID = "11111111-1111-1111-1111-111111111111"

	siteAWS   = "22222222-0000-0000-0000-000000000001"
	siteAzure = "22222222-0000-0000-0000-000000000002"
	siteGCP   = "22222222-0000-0000-0000-000000000003"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed-e2e-data:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.New(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			fmt.Fprintln(os.Stderr, "seed-e2e-data: rollback:", rbErr)
		}
	}()

	if err := seedFixture(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	fmt.Printf("seeded E2E fixture: org=%s user=dev-user sites=3 assets=10 images=4 drift_reports=3 alerts=4\n", orgID)
	return nil
}

// seedFixture inserts the full deterministic fixture inside the given tx.
// Every insert is ON CONFLICT (id) DO NOTHING with fixed IDs, so the seed is
// idempotent without deleting — which also avoids tripping the usage-tracking
// triggers that fire on cascade deletes.
func seedFixture(ctx context.Context, tx pgx.Tx) error {
	for _, step := range []func(context.Context, pgx.Tx) error{
		seedOrg, seedUser, seedSites, seedImages, seedAssets, seedDrift, seedAlerts,
	} {
		if err := step(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}

func seedOrg(ctx context.Context, tx pgx.Tx) error {
	// created_at pinned to the past so dev-mode (oldest org) resolves to it.
	_, err := tx.Exec(ctx, `
		INSERT INTO organizations (id, name, slug, created_at, updated_at)
		VALUES ($1, 'QuantumLayer E2E Org', 'quantumlayer-e2e', '2020-01-01T00:00:00Z', NOW())
		ON CONFLICT (id) DO NOTHING`,
		orgID,
	)
	if err != nil {
		return fmt.Errorf("insert org: %w", err)
	}
	return nil
}

// seedUser links the API's dev-mode mock user (external_id "dev-user", see
// services/api/internal/middleware/auth.go) to the E2E org. Without this,
// /organization/check returns has_organization=false and the frontend's
// OrgGuard redirects /overview to /onboarding.
func seedUser(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO users (id, external_id, email, name, role, org_id)
		VALUES ('77777777-0000-0000-0000-000000000001', 'dev-user', 'dev@example.com', 'Development User', 'admin', $1)
		ON CONFLICT (id) DO NOTHING`,
		orgID,
	)
	if err != nil {
		return fmt.Errorf("insert dev user: %w", err)
	}
	return nil
}

func seedSites(ctx context.Context, tx pgx.Tx) error {
	// AWS is DR-paired with Azure; GCP is unpaired -> DR readiness 1/3.
	rows := []struct{ id, name, region, platform, env string }{
		{siteAWS, "AWS US-East-1", "us-east-1", "aws", "production"},
		{siteAzure, "Azure East US", "eastus", "azure", "production"},
		{siteGCP, "GCP US-Central1", "us-central1", "gcp", "staging"},
	}
	for _, s := range rows {
		if _, err := tx.Exec(ctx, `
			INSERT INTO sites (id, org_id, name, region, platform, environment)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO NOTHING`,
			s.id, orgID, s.name, s.region, s.platform, s.env,
		); err != nil {
			return fmt.Errorf("insert site %s: %w", s.name, err)
		}
	}
	if _, err := tx.Exec(ctx,
		`UPDATE sites SET dr_paired_site_id = $1 WHERE id = $2`, siteAzure, siteAWS,
	); err != nil {
		return fmt.Errorf("pair DR sites: %w", err)
	}
	return nil
}

func seedImages(ctx context.Context, tx pgx.Tx) error {
	rows := []struct{ family, version string }{
		{"golden-ubuntu-22", "2024.11.0"},
		{"golden-amazonlinux-2023", "2024.11.0"},
		{"golden-windows-2022", "2024.10.0"},
		{"golden-rhel-9", "2024.09.0"},
	}
	for i, im := range rows {
		if _, err := tx.Exec(ctx, `
			INSERT INTO images (id, org_id, family, version)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO NOTHING`,
			fmt.Sprintf("33333333-0000-0000-0000-0000000000%02d", i+1), orgID, im.family, im.version,
		); err != nil {
			return fmt.Errorf("insert image %s: %w", im.family, err)
		}
	}
	return nil
}

func seedAssets(ctx context.Context, tx pgx.Tx) error {
	// 5 aws, 3 azure, 2 gcp. Each asset is linked to its platform's site via
	// site_id and is 'running', so the Sites page shows deterministic per-site
	// counts (AWS=5, Azure=3, GCP=2; 10 total).
	siteFor := map[string]string{"aws": siteAWS, "azure": siteAzure, "gcp": siteGCP}
	regionFor := map[string]string{"aws": "us-east-1", "azure": "eastus", "gcp": "us-central1"}
	rows := []struct{ platform, env string }{
		{"aws", "production"}, {"aws", "production"}, {"aws", "production"}, {"aws", "production"}, {"aws", "staging"},
		{"azure", "production"}, {"azure", "production"}, {"azure", "staging"},
		{"gcp", "production"}, {"gcp", "staging"},
	}
	for i, a := range rows {
		if _, err := tx.Exec(ctx, `
			INSERT INTO assets (id, org_id, site_id, platform, instance_id, name, region, state, tags)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'running', $8)
			ON CONFLICT (id) DO UPDATE SET site_id = EXCLUDED.site_id, state = EXCLUDED.state`,
			fmt.Sprintf("44444444-0000-0000-0000-0000000000%02d", i+1),
			orgID, siteFor[a.platform], a.platform,
			fmt.Sprintf("i-e2e-%s-%02d", a.platform, i+1),
			fmt.Sprintf("e2e-%s-host-%02d", a.platform, i+1),
			regionFor[a.platform],
			fmt.Sprintf(`{"environment": %q}`, a.env),
		); err != nil {
			return fmt.Errorf("insert asset %d: %w", i+1, err)
		}
	}
	return nil
}

func seedDrift(ctx context.Context, tx pgx.Tx) error {
	rows := []struct {
		id, platform, site, status string
		total, compliant           int
		coverage                   float64
		ago                        time.Duration
	}{
		{"55555555-0000-0000-0000-000000000001", "aws", "AWS US-East-1", "compliant", 5, 5, 100.0, 30 * time.Minute},
		{"55555555-0000-0000-0000-000000000002", "azure", "Azure East US", "warning", 3, 2, 66.7, 60 * time.Minute},
		{"55555555-0000-0000-0000-000000000003", "gcp", "GCP US-Central1", "drifted", 2, 1, 50.0, 90 * time.Minute},
	}
	for _, d := range rows {
		if _, err := tx.Exec(ctx, `
			INSERT INTO drift_reports (id, org_id, platform, site, total_assets, compliant_assets, coverage_pct, status, calculated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO NOTHING`,
			d.id, orgID, d.platform, d.site, d.total, d.compliant, d.coverage, d.status, time.Now().Add(-d.ago),
		); err != nil {
			return fmt.Errorf("insert drift report %s: %w", d.platform, err)
		}
	}
	return nil
}

func seedAlerts(ctx context.Context, tx pgx.Tx) error {
	// 1 critical, 2 warning, 1 info.
	rows := []struct{ id, severity, title, source string }{
		{"66666666-0000-0000-0000-000000000001", "critical", "Production drift exceeds threshold", "drift"},
		{"66666666-0000-0000-0000-000000000002", "warning", "Image golden-rhel-9 is 60 days old", "image"},
		{"66666666-0000-0000-0000-000000000003", "warning", "Azure site compliance below target", "compliance"},
		{"66666666-0000-0000-0000-000000000004", "info", "Connector sync completed", "connector"},
	}
	for _, al := range rows {
		if _, err := tx.Exec(ctx, `
			INSERT INTO alerts (id, org_id, severity, title, source, status)
			VALUES ($1, $2, $3, $4, $5, 'open')
			ON CONFLICT (id) DO NOTHING`,
			al.id, orgID, al.severity, al.title, al.source,
		); err != nil {
			return fmt.Errorf("insert alert %s: %w", al.title, err)
		}
	}
	return nil
}
