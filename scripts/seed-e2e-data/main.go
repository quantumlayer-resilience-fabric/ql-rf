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
// Scope: this seeds the entities the Overview, Sites, Drift, Images, and Risk
// pages need (org, project, environments, sites, assets, images, image
// vulnerabilities, drift reports, alerts). CVE alerts, compliance results, and
// certificates are intentionally deferred until the corresponding page specs
// are tackled (see docs/E2E-001-deterministic-fullstack-e2e.md).
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

	// Golden image IDs (see seedImages: family order ubuntu, amazonlinux,
	// windows, rhel -> suffixes 01..04). Referenced by seedVulnerabilities.
	imageWindows = "33333333-0000-0000-0000-000000000003"
	imageRHEL    = "33333333-0000-0000-0000-000000000004"

	// Project + environments drive the risk service's environment multiplier
	// (production 1.5x, staging 1.0x) via assets.env_id.
	projectID  = "88888888-0000-0000-0000-000000000001"
	envProd    = "99999999-0000-0000-0000-000000000001"
	envStaging = "99999999-0000-0000-0000-000000000002"
	envDev     = "99999999-0000-0000-0000-000000000003"
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

	fmt.Printf("seeded E2E fixture: org=%s user=dev-user project=1 envs=3 sites=3 assets=10 images=4 vulnerabilities=24 drift_reports=3 alerts=4\n", orgID)
	return nil
}

// seedFixture inserts the full deterministic fixture inside the given tx.
// Every insert is ON CONFLICT (id) DO NOTHING with fixed IDs, so the seed is
// idempotent without deleting — which also avoids tripping the usage-tracking
// triggers that fire on cascade deletes.
func seedFixture(ctx context.Context, tx pgx.Tx) error {
	for _, step := range []func(context.Context, pgx.Tx) error{
		seedOrg, seedUser, seedProjectAndEnvironments, seedSites, seedImages,
		seedAssets, seedVulnerabilities, seedDrift, seedAlerts,
	} {
		if err := step(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}

// seedProjectAndEnvironments creates one project with production/staging/
// development environments. Assets reference these via env_id so the risk
// service applies the production environment multiplier (see
// models.EnvironmentRiskMultiplier). Must run before seedAssets (FK).
func seedProjectAndEnvironments(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO projects (id, org_id, name, slug)
		VALUES ($1, $2, 'QuantumLayer E2E Project', 'quantumlayer-e2e')
		ON CONFLICT (org_id, slug) DO NOTHING`,
		projectID, orgID,
	); err != nil {
		return fmt.Errorf("insert project: %w", err)
	}
	envs := []struct{ id, name string }{
		{envProd, "production"},
		{envStaging, "staging"},
		{envDev, "development"},
	}
	for _, e := range envs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO environments (id, project_id, name)
			VALUES ($1, $2, $3)
			ON CONFLICT (project_id, name) DO NOTHING`,
			e.id, projectID, e.name,
		); err != nil {
			return fmt.Errorf("insert environment %s: %w", e.name, err)
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
		// status='production' so assets matching family+version count as compliant
		// (see CountCompliantAssets).
		if _, err := tx.Exec(ctx, `
			INSERT INTO images (id, org_id, family, version, status)
			VALUES ($1, $2, $3, $4, 'production')
			ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status`,
			fmt.Sprintf("33333333-0000-0000-0000-0000000000%02d", i+1), orgID, im.family, im.version,
		); err != nil {
			return fmt.Errorf("insert image %s: %w", im.family, err)
		}
	}
	return nil
}

func seedAssets(ctx context.Context, tx pgx.Tx) error {
	// 5 aws, 3 azure, 2 gcp, linked to their platform's site via site_id (Sites
	// page counts) and to golden images via image_ref/image_version. A compliant
	// asset matches a production golden image family+version; a drifted one uses
	// an unmanaged ref. 8 compliant / 2 drifted = 20% drift (Drift page).
	//
	// Image families are also chosen to isolate risk: the high/medium-risk
	// assets are the SOLE users of golden-rhel-9 and golden-windows-2022
	// respectively, so the vulnerabilities seeded on those images (see
	// seedVulnerabilities) raise exactly one asset each. Combined with the
	// production environment multiplier this yields: 1 high, 1 medium, 8 low
	// (Risk page). env drives both env_id (risk multiplier) and the tag.
	//
	// All assets are in the production environment on purpose: the Drift page's
	// criticalDrift counts drifted assets in environments whose coverage falls
	// into the "critical" band. Keeping all 10 in one 80%-coverage group (8
	// compliant / 2 drifted) leaves it at "warning", so criticalDrift stays 0
	// and the Drift spec's "Drift Pattern Analysis" insight is preserved.
	siteFor := map[string]string{"aws": siteAWS, "azure": siteAzure, "gcp": siteGCP}
	regionFor := map[string]string{"aws": "us-east-1", "azure": "eastus", "gcp": "us-central1"}
	envID := map[string]string{"production": envProd, "staging": envStaging, "development": envDev}
	const drifted, driftedVer = "legacy-unmanaged", "0.0.0"
	rows := []struct{ platform, env, imageRef, imageVer, name string }{
		{"aws", "production", "golden-rhel-9", "2024.09.0", "e2e-aws-risk-high"},
		{"aws", "production", "golden-amazonlinux-2023", "2024.11.0", "e2e-aws-host-02"},
		{"aws", "production", "golden-amazonlinux-2023", "2024.11.0", "e2e-aws-host-03"},
		{"aws", "production", "golden-amazonlinux-2023", "2024.11.0", "e2e-aws-host-04"},
		{"aws", "production", "golden-amazonlinux-2023", "2024.11.0", "e2e-aws-host-05"},
		{"azure", "production", "golden-windows-2022", "2024.10.0", "e2e-azure-risk-medium"},
		{"azure", "production", "golden-ubuntu-22", "2024.11.0", "e2e-azure-host-07"},
		{"azure", "production", drifted, driftedVer, "e2e-azure-host-08"},
		{"gcp", "production", "golden-ubuntu-22", "2024.11.0", "e2e-gcp-host-09"},
		{"gcp", "production", drifted, driftedVer, "e2e-gcp-host-10"},
	}
	for i, a := range rows {
		if _, err := tx.Exec(ctx, `
			INSERT INTO assets (id, org_id, env_id, site_id, platform, instance_id, name, region, state, image_ref, image_version, tags)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'running', $9, $10, $11)
			ON CONFLICT (id) DO UPDATE SET
				env_id = EXCLUDED.env_id, site_id = EXCLUDED.site_id, name = EXCLUDED.name,
				state = EXCLUDED.state, image_ref = EXCLUDED.image_ref, image_version = EXCLUDED.image_version`,
			fmt.Sprintf("44444444-0000-0000-0000-0000000000%02d", i+1),
			orgID, envID[a.env], siteFor[a.platform], a.platform,
			fmt.Sprintf("i-e2e-%s-%02d", a.platform, i+1),
			a.name, regionFor[a.platform], a.imageRef, a.imageVer,
			fmt.Sprintf(`{"environment": %q}`, a.env),
		); err != nil {
			return fmt.Errorf("insert asset %d: %w", i+1, err)
		}
	}
	return nil
}

// seedVulnerabilities attaches open image_vulnerabilities to two golden images
// so the risk service produces deterministic scores. Vulnerabilities are keyed
// to an image family, so every asset on that family inherits them — the asset
// fixture deliberately puts a single asset on each of these families.
//
//	golden-rhel-9    : 4 critical + 16 high = 20 open  -> sole asset scores HIGH
//	golden-windows-2022 : 4 critical          = 4 open  -> sole asset scores MEDIUM
//
// (Reaching the CRITICAL band would require the drift/compliance factors, which
// are coupled to the Drift page's drifted count, so we deliberately stop at
// HIGH to keep the Drift fixture invariant intact.)
func seedVulnerabilities(ctx context.Context, tx pgx.Tx) error {
	specs := []struct {
		imageID         string
		critical, total int
		base            int // id/cve offset, keeps ids unique across images
	}{
		{imageRHEL, 4, 20, 0},
		{imageWindows, 4, 4, 100},
	}
	for _, s := range specs {
		for n := 0; n < s.total; n++ {
			severity := "high"
			if n < s.critical {
				severity = "critical"
			}
			seq := s.base + n
			if _, err := tx.Exec(ctx, `
				INSERT INTO image_vulnerabilities (id, image_id, cve_id, severity, package_name, status)
				VALUES ($1, $2, $3, $4, $5, 'open')
				ON CONFLICT (id) DO NOTHING`,
				fmt.Sprintf("aaaaaaaa-0000-0000-0000-%012d", seq),
				s.imageID,
				fmt.Sprintf("CVE-2024-%05d", seq),
				severity,
				fmt.Sprintf("pkg-%d", seq),
			); err != nil {
				return fmt.Errorf("insert vulnerability %d for %s: %w", seq, s.imageID, err)
			}
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
