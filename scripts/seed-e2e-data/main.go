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
// Scope: this seeds the entities the Overview, Sites, Drift, Images, Risk,
// Vulnerabilities, Certificates, Compliance, SBOM, and Mission Control pages
// need (org, project, environments, sites, assets, images, image
// vulnerabilities, drift reports, alerts, CVE cache + alerts, certificates,
// compliance frameworks + controls + results, sboms + packages, ai tasks +
// plans + runs + tool invocations + llm usage). InSpec and FinOps are
// intentionally deferred until the corresponding page specs are tackled (see
// docs/E2E-001-deterministic-fullstack-e2e.md and
// docs/E2E-011-ai-mission-control.md).
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

	// The orchestrator (which serves CVE alerts) hardcodes this org id in dev
	// mode — see services/orchestrator/internal/middleware/auth.go — rather than
	// resolving the oldest org like the API. So CVE data must be seeded under
	// THIS org to be visible on the Vulnerabilities page. Its created_at is kept
	// recent so the API's dev-mode lookupDefaultOrg still selects the 2020 E2E
	// org for every other page.
	orchestratorDevOrgID = "00000000-0000-0000-0000-000000000001"

	// Day offsets (relative to now) for seeded certificate not_after dates. The
	// certificates table's BEFORE INSERT/UPDATE trigger derives days_until_expiry
	// and status from not_after, so these offsets deterministically yield: active
	// (long expiry), expiring_soon, expiring within 7 days, and expired.
	certActiveDays     = 825
	certActiveAltDays  = 600
	certExpiringDays   = 15
	certExpiring7Days  = 5
	certExpiredDays    = -10
	certRenewThreshold = 30
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

	fmt.Printf("seeded E2E fixture: org=%s user=dev-user project=1 envs=3 sites=3 assets=10 images=4 vulnerabilities=24 drift_reports=3 alerts=4 cve_alerts=5 certificates=5 compliance_frameworks=2 controls=7 mappings=8 sboms=1 sbom_packages=6 ai_tasks=5 ai_plans=5 ai_runs=2 ai_tool_invocations=6 llm_usage=4 ai_conversations=2 ai_conversation_messages=4\n", orgID)
	return nil
}

// seedFixture inserts the full deterministic fixture inside the given tx.
// Every insert is ON CONFLICT (id) DO NOTHING with fixed IDs, so the seed is
// idempotent without deleting — which also avoids tripping the usage-tracking
// triggers that fire on cascade deletes.
func seedFixture(ctx context.Context, tx pgx.Tx) error {
	for _, step := range []func(context.Context, pgx.Tx) error{
		seedOrg, seedUser, seedProjectAndEnvironments, seedSites, seedImages,
		seedAssets, seedVulnerabilities, seedDrift, seedAlerts, seedCVEAlerts,
		seedCertificates, seedCompliance, seedSBOM, seedMissionControl,
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

// seedCVEAlerts seeds the Vulnerability Response Center (orchestrator) data:
// a cve_cache entry plus a cve_alert per CVE. The list/summary endpoints read
// the denormalized *_count columns on cve_alerts and LEFT JOIN cve_cache for
// severity/KEV/exploit details (see cve_alerts_query.go), so the dashboard
// renders entirely from these two tables. cve_alert_affected_items (the
// per-item blast-radius evidence on the detail page) is intentionally deferred,
// like image lineage. Data lives under orchestratorDevOrgID, the org the
// orchestrator resolves in dev mode.
//
// Resulting summary: 5 alerts, 2 critical + 2 high, 2 CISA KEV, 3 exploitable,
// 1 SLA-breached, 15 affected assets (6 production).
func seedCVEAlerts(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO organizations (id, name, slug, created_at, updated_at)
		VALUES ($1, 'QuantumLayer E2E Orchestrator Org', 'quantumlayer-e2e-orchestrator', '2025-01-01T00:00:00Z', NOW())
		ON CONFLICT (id) DO NOTHING`,
		orchestratorDevOrgID,
	); err != nil {
		return fmt.Errorf("insert orchestrator dev org: %w", err)
	}

	rows := []struct {
		cacheID, alertID, cveID           string
		severity, priority, status        string
		description                       string
		cvss                              float64
		kev, exploit, slaBreached         bool
		urgency, pkgs, imgs, assets, prod int
	}{
		{"bbbbbbbb-0000-0000-0000-000000000001", "cccccccc-0000-0000-0000-000000000001", "CVE-2024-3094",
			"critical", "p1", "new", "Malicious backdoor planted in xz/liblzma compression library.",
			10.0, true, true, true, 98, 3, 2, 5, 3},
		{"bbbbbbbb-0000-0000-0000-000000000002", "cccccccc-0000-0000-0000-000000000002", "CVE-2021-44228",
			"critical", "p1", "investigating", "Log4Shell: remote code execution in Apache Log4j 2 via JNDI lookup.",
			10.0, true, true, false, 95, 2, 1, 4, 2},
		{"bbbbbbbb-0000-0000-0000-000000000003", "cccccccc-0000-0000-0000-000000000003", "CVE-2024-21626",
			"high", "p2", "new", "runc container escape via leaked file descriptor.",
			8.6, false, true, false, 72, 1, 1, 2, 1},
		{"bbbbbbbb-0000-0000-0000-000000000004", "cccccccc-0000-0000-0000-000000000004", "CVE-2023-44487",
			"high", "p2", "new", "HTTP/2 Rapid Reset denial-of-service amplification.",
			7.5, false, false, false, 65, 1, 1, 3, 0},
		{"bbbbbbbb-0000-0000-0000-000000000005", "cccccccc-0000-0000-0000-000000000005", "CVE-2024-6387",
			"medium", "p3", "resolved", "regreSSHion: OpenSSH signal handler race condition.",
			5.9, false, false, false, 45, 1, 1, 1, 0},
	}
	for i := range rows {
		c := &rows[i] // pointer avoids copying the large struct each iteration
		if _, err := tx.Exec(ctx, `
			INSERT INTO cve_cache (id, cve_id, cvss_v3_score, severity, exploit_available, cisa_kev_listed, description, primary_source)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'nvd')
			ON CONFLICT (cve_id) DO NOTHING`,
			c.cacheID, c.cveID, c.cvss, c.severity, c.exploit, c.kev, c.description,
		); err != nil {
			return fmt.Errorf("insert cve_cache %s: %w", c.cveID, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO cve_alerts (
				id, org_id, cve_id, cve_cache_id, severity, urgency_score, status, priority, sla_breached,
				affected_images_count, affected_assets_count, affected_packages_count, production_assets_count)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			ON CONFLICT (org_id, cve_id) DO UPDATE SET
				severity = EXCLUDED.severity, urgency_score = EXCLUDED.urgency_score, status = EXCLUDED.status,
				priority = EXCLUDED.priority, sla_breached = EXCLUDED.sla_breached,
				affected_images_count = EXCLUDED.affected_images_count, affected_assets_count = EXCLUDED.affected_assets_count,
				affected_packages_count = EXCLUDED.affected_packages_count, production_assets_count = EXCLUDED.production_assets_count`,
			c.alertID, orchestratorDevOrgID, c.cveID, c.cacheID, c.severity, c.urgency, c.status, c.priority, c.slaBreached,
			c.imgs, c.assets, c.pkgs, c.prod,
		); err != nil {
			return fmt.Errorf("insert cve_alert %s: %w", c.cveID, err)
		}
	}
	return nil
}

// seedCertificates seeds the Certificate Lifecycle Management page (API). The
// certificates table has a BEFORE INSERT/UPDATE trigger that derives
// days_until_expiry and status from not_after, so we set not_after relative to
// now (computed at seed time, deterministic for the immediately-following test
// run) and let the trigger classify each cert. This yields a summary of:
// 5 total, 2 active, 2 expiring_soon (1 within 7 days), 1 expired, across 5
// platforms, 2 auto-renew. Non-pointer scan fields (issuer_*, key_*, source_ref)
// are set explicitly so the row scans cleanly.
func seedCertificates(ctx context.Context, tx pgx.Tx) error {
	now := time.Now()
	rows := []struct {
		id, cn, platform, source string
		offsetDays               int
		autoRenew                bool
	}{
		{"dddddddd-0000-0000-0000-000000000001", "api.quantumlayer.io", "aws", "acm", certActiveDays, true},
		{"dddddddd-0000-0000-0000-000000000002", "app.quantumlayer.io", "azure", "azure_keyvault", certActiveAltDays, true},
		{"dddddddd-0000-0000-0000-000000000003", "dashboard.quantumlayer.io", "gcp", "gcp_certificate_manager", certExpiringDays, false},
		{"dddddddd-0000-0000-0000-000000000004", "legacy.quantumlayer.io", "k8s", "k8s_secret", certExpiring7Days, false},
		{"dddddddd-0000-0000-0000-000000000005", "old.quantumlayer.io", "vsphere", "file", certExpiredDays, false},
	}
	for i := range rows {
		c := &rows[i]
		notAfter := now.AddDate(0, 0, c.offsetDays)
		notBefore := now.AddDate(0, 0, -90)
		if _, err := tx.Exec(ctx, `
			INSERT INTO certificates (
				id, org_id, fingerprint, common_name, subject_alt_names,
				issuer_common_name, issuer_organization, not_before, not_after,
				key_algorithm, key_size, signature_algorithm,
				source, source_ref, platform, auto_renew, renewal_threshold_days)
			VALUES ($1, $2, $3, $4, $5, 'QuantumLayer E2E CA', 'QuantumLayer', $6, $7,
				'RSA', 2048, 'SHA256-RSA', $8, $9, $10, $11, $12)
			ON CONFLICT (id) DO UPDATE SET
				not_after = EXCLUDED.not_after, auto_renew = EXCLUDED.auto_renew,
				platform = EXCLUDED.platform, source = EXCLUDED.source`,
			c.id, orgID,
			fmt.Sprintf("e2e-cert-fp-%s", c.id[len(c.id)-2:]),
			c.cn, []string{},
			notBefore, notAfter,
			c.source, fmt.Sprintf("e2e-cert-ref-%s", c.cn), c.platform, c.autoRenew, certRenewThreshold,
		); err != nil {
			return fmt.Errorf("insert certificate %s: %w", c.cn, err)
		}
	}
	return nil
}

// seedCompliance seeds the Compliance page (API). The service's
// GetComplianceSummary computes overall/CIS/SLSA scores by joining
// compliance_frameworks -> compliance_controls -> compliance_results, so the
// seed sets up two frameworks with controls + per-control results:
//
//	CIS Benchmark : 5 controls, 3 passing + 2 failing -> 60.0% (failing status)
//	SLSA          : 1 control, 1 passing, level 3    -> 100.0% (passing)
//
// Resulting summary: overall 80.0%, cis 60.0%, slsa level 3, sigstore 0% (no
// image_compliance rows seeded), 2 failing controls (1 high + 1 medium).
func seedCompliance(ctx context.Context, tx pgx.Tx) error {
	frameworks := []struct {
		id, name, description string
		level                 *int
	}{
		{"eeeeeeee-0000-0000-0000-000000000001", "CIS Benchmark", "Center for Internet Security Benchmarks", nil},
		{"eeeeeeee-0000-0000-0000-000000000002", "SLSA", "Supply-chain Levels for Software Artifacts", ptrInt(3)},
	}
	for _, f := range frameworks {
		if _, err := tx.Exec(ctx, `
			INSERT INTO compliance_frameworks (id, org_id, name, description, level, enabled)
			VALUES ($1, $2, $3, $4, $5, true)
			ON CONFLICT (org_id, name) DO NOTHING`,
			f.id, orgID, f.name, f.description, f.level,
		); err != nil {
			return fmt.Errorf("insert framework %s: %w", f.name, err)
		}
	}

	const cisFrameworkID = "eeeeeeee-0000-0000-0000-000000000001"
	const slsaFrameworkID = "eeeeeeee-0000-0000-0000-000000000002"
	controls := []struct {
		id, frameworkID, controlID, title, severity, recommendation, resultStatus string
		affectedAssets                                                            int
		score                                                                     float64
		// skipResult excludes the control from the compliance_results
		// scoring fixture. The control still exists in the registry (so
		// mappings can reference it) but doesn't move the overall score.
		// Used for controls introduced as evidence-emission targets that
		// don't have an active scan result on the demo dashboard.
		skipResult bool
	}{
		{"eeeeeeee-1000-0000-0000-000000000001", cisFrameworkID, "CIS-1.1", "Ensure SSH root login is disabled", "high", "Set PermitRootLogin no in /etc/ssh/sshd_config.", "failing", 3, 0, false},
		{"eeeeeeee-1000-0000-0000-000000000002", cisFrameworkID, "CIS-1.2", "Ensure password expiration is configured", "medium", "Set PASS_MAX_DAYS to 90 in /etc/login.defs.", "failing", 2, 0, false},
		{"eeeeeeee-1000-0000-0000-000000000003", cisFrameworkID, "CIS-2.1", "Ensure auditd is enabled and running", "medium", "Enable the auditd service.", "passing", 0, 100, false},
		{"eeeeeeee-1000-0000-0000-000000000004", cisFrameworkID, "CIS-2.2", "Ensure firewalld is active", "medium", "systemctl enable --now firewalld.", "passing", 0, 100, false},
		{"eeeeeeee-1000-0000-0000-000000000005", cisFrameworkID, "CIS-3.1", "Ensure system is up to date", "low", "Run package updates.", "passing", 0, 100, false},
		{"eeeeeeee-1000-0000-0000-000000000006", slsaFrameworkID, "SLSA-L3", "Source and build platform meet SLSA Level 3", "high", "Use a hardened build platform.", "passing", 0, 100, false},
		// PR #24 / CONN-004: new patch-management control. This is the
		// control the SSM tools map to via tool_compliance_mappings, so
		// dry-run and live invocations of ssm_send_patch_command produce
		// compliance_evidence rows under this control. Status=passing
		// with score=100; this shifts CIS from 60.0% to 66.7% and overall
		// from 80.0% to 83.3% (Playwright expectations updated to match).
		// Skipping the result row entirely would make the control show
		// up as "failing" (the failing-controls SQL counts cr.status IS
		// NULL as failing), which would break the "2 Compliance Gaps
		// Detected" test.
		{"eeeeeeee-1000-0000-0000-000000000007", cisFrameworkID, "CIS-1.4", "Ensure systems are patched against known vulnerabilities", "high", "Apply patches via approved orchestration tooling (e.g., AWS SSM, Azure Update Manager).", "passing", 0, 100, false},
	}
	for i := range controls {
		c := &controls[i]
		if _, err := tx.Exec(ctx, `
			INSERT INTO compliance_controls (id, framework_id, control_id, title, severity, recommendation)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (framework_id, control_id) DO NOTHING`,
			c.id, c.frameworkID, c.controlID, c.title, c.severity, c.recommendation,
		); err != nil {
			return fmt.Errorf("insert control %s: %w", c.controlID, err)
		}
		if c.skipResult {
			continue
		}
		resultID := "eeeeeeee-2000-0000-0000-0000000000" + c.id[len(c.id)-2:]
		if _, err := tx.Exec(ctx, `
			INSERT INTO compliance_results (id, org_id, framework_id, control_id, status, affected_assets, score)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO UPDATE SET
				status = EXCLUDED.status, affected_assets = EXCLUDED.affected_assets, score = EXCLUDED.score`,
			resultID, orgID, c.frameworkID, c.id, c.resultStatus, c.affectedAssets, c.score,
		); err != nil {
			return fmt.Errorf("insert result for %s: %w", c.controlID, err)
		}
	}

	// PR #24 / CONN-004: tool → compliance control mappings. The SSM
	// tool family (dry-run from PR #20, live from PR #21) maps to the new
	// CIS-1.4 patch-management control. NULL org_id = global default; any
	// org can override by inserting an org-specific row, which wins by the
	// emitter's precedence order. ON CONFLICT (...) DO NOTHING keeps the
	// seed idempotent.
	const cisPatchControlID = "eeeeeeee-1000-0000-0000-000000000007"
	mappings := []struct {
		toolPattern, controlID, notes string
	}{
		{"ssm_send_patch_command", cisPatchControlID,
			"PR #20 dry-run tool. Records an attestation that a patch plan was constructed (proof of intent + approval flow)."},
		{"ssm_send_patch_command_live", cisPatchControlID,
			"PR #21 live tool. Records an attestation that ssm:SendCommand fired against whitelisted instances after two-approver workflow."},
		{"azure_run_command", cisPatchControlID,
			"PR #27 dry-run tool. Records an attestation that an Azure VM Run Command plan was constructed (proof of intent + approval flow)."},
		{"azure_run_command_live", cisPatchControlID,
			"PR #28 live tool (planned). Records an attestation that armcompute.VirtualMachineRunCommandsClient.BeginCreateOrUpdate fired against whitelisted VMs after two-approver workflow."},
		{"gcp_os_config_patch", cisPatchControlID,
			"PR #30 dry-run tool. Records an attestation that a GCP OS Config patch job plan was constructed (proof of intent + approval flow)."},
		{"gcp_os_config_patch_live", cisPatchControlID,
			"PR #31 live tool. Records an attestation that osconfig.ExecutePatchJob fired against whitelisted instances after two-approver workflow."},
		{"vsphere_run_guest_program", cisPatchControlID,
			"PR #34 dry-run tool. Records an attestation that a vSphere guest-program-run plan was constructed (proof of intent + approval flow)."},
		{"vsphere_run_guest_program_live", cisPatchControlID,
			"PR #35 live tool (planned). Records an attestation that ProcessManager.StartProgramInGuest fired against whitelisted VMs after two-approver workflow."},
	}
	for _, m := range mappings {
		if _, err := tx.Exec(ctx, `
			INSERT INTO tool_compliance_mappings (org_id, tool_name_pattern, control_id, notes)
			VALUES (NULL, $1, $2, $3)
			ON CONFLICT DO NOTHING`,
			m.toolPattern, m.controlID, m.notes,
		); err != nil {
			return fmt.Errorf("insert tool mapping %s: %w", m.toolPattern, err)
		}
	}
	return nil
}

func ptrInt(v int) *int { return &v }

// seedSBOM seeds one SPDX SBOM linked to the existing golden-ubuntu-22 image
// plus six realistic deb packages so the SBOM page renders meaningful inventory
// data. The page reads the SBOM list and the latest SBOM's packages
// (/sbom/{id}?include_packages=true) — no vulnerabilities are seeded, so the
// Vulnerabilities tab shows 0 and the AI-insight card stays hidden. License
// compliance is computed client-side from a mocked /licenses summary (always
// 100.0%), independent of seeded license values.
func seedSBOM(ctx context.Context, tx pgx.Tx) error {
	const sbomID = "ffffffff-0000-0000-0000-000000000001"
	const imageUbuntu = "33333333-0000-0000-0000-000000000001" // golden-ubuntu-22
	if _, err := tx.Exec(ctx, `
		INSERT INTO sboms (id, image_id, org_id, format, version, content, package_count, scanner)
		VALUES ($1, $2, $3, 'spdx', 'SPDX-2.3', '{}'::jsonb, 6, 'ql-rf-e2e')
		ON CONFLICT (id) DO UPDATE SET package_count = EXCLUDED.package_count`,
		sbomID, imageUbuntu, orgID,
	); err != nil {
		return fmt.Errorf("insert sbom: %w", err)
	}

	packages := []struct{ id, name, version, license string }{
		{"ffffffff-1000-0000-0000-000000000001", "ca-certificates", "20210119ubuntu0.20.04.1", "MPL-2.0"},
		{"ffffffff-1000-0000-0000-000000000002", "openssl", "1.1.1f-1ubuntu2.19", "OpenSSL"},
		{"ffffffff-1000-0000-0000-000000000003", "libc6", "2.31-0ubuntu9.9", "LGPL-2.1"},
		{"ffffffff-1000-0000-0000-000000000004", "bash", "5.0-6ubuntu1.2", "GPL-3.0"},
		{"ffffffff-1000-0000-0000-000000000005", "zlib1g", "1:1.2.11.dfsg-2ubuntu1.5", "Zlib"},
		{"ffffffff-1000-0000-0000-000000000006", "systemd", "245.4-4ubuntu3.21", "LGPL-2.1"},
	}
	for i := range packages {
		p := &packages[i]
		if _, err := tx.Exec(ctx, `
			INSERT INTO sbom_packages (id, sbom_id, name, version, type, license)
			VALUES ($1, $2, $3, $4, 'deb', $5)
			ON CONFLICT (id) DO NOTHING`,
			p.id, sbomID, p.name, p.version, p.license,
		); err != nil {
			return fmt.Errorf("insert sbom_package %s: %w", p.name, err)
		}
	}
	return nil
}

// seedMissionControl seeds the Mission Control surface (AI-001 / E2E-011): a
// deterministic AI lifecycle under the orchestrator dev org covering one task
// in each lifecycle position (pending approval, executing, completed, rejected)
// plus tool invocations across multiple agents and LLM usage rows so the fleet
// status bar reports a real spend. All data is read-only; the seed does not
// trigger any agent execution or LLM call.
func seedMissionControl(ctx context.Context, tx pgx.Tx) error {
	const (
		userID = "e0000000-0000-0000-0000-000000000001"
		// firstApproverID is a second seeded user used as the recorded first
		// approver on task5 (the PR #22 "awaiting second approval" fixture).
		// The Mission Control UI shows their UUID prefix in the "1st: ..."
		// chip; co-approve will be triggered by userID, which is distinct.
		firstApproverID = "e0000000-0000-0000-0000-0000000000aa"
		task1           = "e1000000-0000-0000-0000-000000000001" // CVE patch, awaiting approval
		task2           = "e1000000-0000-0000-0000-000000000002" // drift analysis, executing
		task3           = "e1000000-0000-0000-0000-000000000003" // cert rotation, completed
		task4           = "e1000000-0000-0000-0000-000000000004" // DR failover, rejected
		task5           = "e1000000-0000-0000-0000-000000000005" // PR #22: SSM live patch, awaiting SECOND approval
	)

	// Mission Control user under the orchestrator dev org (the org the
	// orchestrator resolves in dev mode — see services/orchestrator/internal/
	// middleware/auth.go). FK ai_tasks.created_by references this user.
	if _, err := tx.Exec(ctx, `
		INSERT INTO users (id, external_id, email, name, role, org_id)
		VALUES ($1, 'mission-control-dev-user', 'mission-control-dev@example.com',
		        'Mission Control Dev User', 'admin', $2)
		ON CONFLICT (id) DO NOTHING`,
		userID, orchestratorDevOrgID,
	); err != nil {
		return fmt.Errorf("insert mission control user: %w", err)
	}

	// PR #22 / CONN-003 (UI): seed a second user who plays the role of
	// "first approver" on the awaiting-second-approval fixture. The
	// co-approve flow requires the second approver to differ from the
	// first, so we need a real second user_id in the DB to FK against.
	if _, err := tx.Exec(ctx, `
		INSERT INTO users (id, external_id, email, name, role, org_id)
		VALUES ($1, 'mission-control-first-approver', 'first-approver@example.com',
		        'First Approver Dev User', 'admin', $2)
		ON CONFLICT (id) DO NOTHING`,
		firstApproverID, orchestratorDevOrgID,
	); err != nil {
		return fmt.Errorf("insert first approver user: %w", err)
	}

	tasks := []struct {
		id, intent string
	}{
		{task1, "Patch CVE-2024-3094 (xz backdoor) on production assets"},
		{task2, "Analyze drift across azure production sites"},
		{task3, "Rotate api.quantumlayer.io certificate before expiry"},
		{task4, "Failover production database to DR site immediately"},
		{task5, "Live-patch production fleet via SSM (CONN-003 demo fixture)"},
	}
	for i := range tasks {
		t := &tasks[i]
		if _, err := tx.Exec(ctx, `
			INSERT INTO ai_tasks (id, org_id, created_by, user_intent, state, source)
			VALUES ($1, $2, $3, $4, 'planned', 'chat')
			ON CONFLICT (id) DO NOTHING`,
			t.id, orchestratorDevOrgID, userID, t.intent,
		); err != nil {
			return fmt.Errorf("insert ai_task %s: %w", t.intent[:32], err)
		}
	}

	// Plans: one per task. State + quality + OPA result vary by lifecycle stage.
	//
	// PR #22 / CONN-003 (UI) adds plan #5: a CONN-003 "live SSM" plan that
	// references ssm_send_patch_command (state_change_prod risk in the
	// registry, always registered since PR #20). State is still
	// awaiting_approval, but approved_by is pre-populated with a DIFFERENT
	// user. The fleet status response computes requires_two_approvers=true
	// from the payload + tool registry, and the UI renders the "Awaiting
	// second approval" badge + Co-approve button instead of Approve.
	plans := []struct {
		id, taskID, planType, state, payload string
		quality                              int
		opaPass                              bool
		opaViolations                        string
		approved                             bool
		rejectionReason                      string
		approvedByOverride                   string // PR #22: pre-set first approver for the two-approver fixture
	}{
		{"e2000000-0000-0000-0000-000000000001", task1, "patch_plan", "awaiting_approval",
			`{"summary":"Patch CVE-2024-3094","blast_radius":{"assets":4,"environment":"production"},"phases":["canary","monitor","full_rollout"],"rollback":"available"}`,
			87, true, `[]`, false, "", ""},
		{"e2000000-0000-0000-0000-000000000002", task2, "drift_plan", "approved",
			`{"summary":"Bring azure production assets back to golden image","blast_radius":{"assets":3,"environment":"production"},"phases":["canary","monitor","full_rollout"]}`,
			92, true, `[]`, true, "", ""},
		{"e2000000-0000-0000-0000-000000000003", task3, "patch_plan", "approved",
			`{"summary":"Rotate api.quantumlayer.io certificate","blast_radius":{"assets":1,"environment":"production"},"phases":["issue","stage","cutover"]}`,
			90, true, `[]`, true, "", ""},
		{"e2000000-0000-0000-0000-000000000004", task4, "dr_runbook", "rejected",
			`{"summary":"Failover prod DB to DR site","blast_radius":{"assets":12,"environment":"production"},"phases":["snapshot","cutover","reroute"]}`,
			65, false, `["production failover blocked by policy: requires two-approver override"]`, false,
			"OPA policy blocked: production_failover_requires_dual_approval", ""},
		{"e2000000-0000-0000-0000-000000000005", task5, "patch_plan", "awaiting_approval",
			// Payload references ssm_send_patch_command — state_change_prod
			// in the tool registry. fleet_status's planPayloadNeedsTwoApprovers
			// walks "tool"/"tool_name" keys and finds this one.
			`{"summary":"Live-patch fleet via SSM","blast_radius":{"assets":2,"environment":"production"},"phases":[{"tool":"ssm_send_patch_command","operation":"Install"}],"rollback":"available"}`,
			88, true, `[]`, false, "", firstApproverID},
	}
	for i := range plans {
		p := &plans[i]
		validation := fmt.Sprintf(`{"schema_valid":true,"schema_errors":[],"opa_valid":%t,"opa_violations":%s,"safety_valid":true,"safety_violations":[],"overall_valid":%t}`,
			p.opaPass, p.opaViolations, p.opaPass)
		var approvedBy *string
		var approvedAt *time.Time
		now := time.Now()
		if p.approved {
			approvedBy = ptrString(userID)
			approvedAt = &now
		}
		// PR #22 / CONN-003 (UI): for the two-approver fixture plan, record
		// first approval up-front so the awaiting-second-approval shape is
		// already on screen when the seeded dashboard loads.
		if p.approvedByOverride != "" {
			approvedBy = ptrString(p.approvedByOverride)
			approvedAt = &now
		}
		var rejReason *string
		if p.rejectionReason != "" {
			rejReason = &p.rejectionReason
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO ai_plans (id, task_id, type, payload, validation, quality_score, state,
				approved_by, approved_at, rejection_reason)
			VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6, $7, $8, $9, $10)
			ON CONFLICT (id) DO UPDATE SET
				state = EXCLUDED.state, validation = EXCLUDED.validation,
				quality_score = EXCLUDED.quality_score, approved_by = EXCLUDED.approved_by,
				approved_at = EXCLUDED.approved_at, rejection_reason = EXCLUDED.rejection_reason`,
			p.id, p.taskID, p.planType, p.payload, validation, p.quality, p.state,
			approvedBy, approvedAt, rejReason,
		); err != nil {
			return fmt.Errorf("insert ai_plan for %s: %w", p.taskID[:8], err)
		}
	}

	// Runs: one executing (task2), one completed (task3). The pending and
	// rejected tasks have no run.
	//
	// Phase B.3 polish: audit_log carries the same shape the simulator
	// produces — {kind, ts, _simulated} entries — so the seeded dashboard
	// already shows a realistic evidence ledger trail without needing a
	// user click. Every entry is tagged _simulated:true.
	const executingAudit = `[
		{"kind":"approved","_simulated":true,"by":"mission-control","ts":"2026-05-30T10:00:00Z"},
		{"kind":"started","_simulated":true,"ts":"2026-05-30T10:00:01Z"}
	]`
	const completedAudit = `[
		{"kind":"approved","_simulated":true,"by":"mission-control","ts":"2026-05-30T09:00:00Z"},
		{"kind":"started","_simulated":true,"ts":"2026-05-30T09:00:01Z"},
		{"kind":"phase_complete","phase":"issue","_simulated":true,"ts":"2026-05-30T09:00:02Z"},
		{"kind":"phase_complete","phase":"stage","_simulated":true,"ts":"2026-05-30T09:00:03Z"},
		{"kind":"phase_complete","phase":"cutover","_simulated":true,"ts":"2026-05-30T09:00:04Z"},
		{"kind":"simulated_complete","_simulated":true,"real_changes":false,"tool_invocations":3,"ts":"2026-05-30T09:00:05Z"}
	]`

	// phases_completed and phases_remaining are populated so phases_total
	// (derived in the GET /runs query) reports the true plan length, not
	// just current_phase=1. The B.3 simulator keeps these in sync per run;
	// the seed mirrors that contract for the two seeded runs.
	runs := []struct {
		id, planID, taskID, state, phase, auditLog string
		phasesCompleted, phasesRemaining           string
		percent                                    int
		completed                                  bool
	}{
		{"e3000000-0000-0000-0000-000000000001",
			"e2000000-0000-0000-0000-000000000002", task2, "executing", "canary", executingAudit,
			`[]`, `["monitor","full_rollout"]`, 50, false},
		{"e3000000-0000-0000-0000-000000000002",
			"e2000000-0000-0000-0000-000000000003", task3, "completed", "", completedAudit,
			`["issue","stage","cutover"]`, `[]`, 100, true},
	}
	for i := range runs {
		r := &runs[i]
		var completedAt *time.Time
		if r.completed {
			now := time.Now()
			completedAt = &now
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO ai_runs (id, plan_id, task_id, environment, initiated_by,
				current_phase, phases_completed, phases_remaining, percent_complete,
				state, audit_log, started_at, completed_at)
			VALUES ($1, $2, $3, 'production', $4, $5, $6::jsonb, $7::jsonb, $8, $9, $10::jsonb, NOW(), $11)
			ON CONFLICT (id) DO UPDATE SET
				state = EXCLUDED.state, current_phase = EXCLUDED.current_phase,
				phases_completed = EXCLUDED.phases_completed,
				phases_remaining = EXCLUDED.phases_remaining,
				percent_complete = EXCLUDED.percent_complete,
				audit_log = EXCLUDED.audit_log,
				completed_at = EXCLUDED.completed_at`,
			r.id, r.planID, r.taskID, userID, r.phase,
			r.phasesCompleted, r.phasesRemaining, r.percent, r.state,
			r.auditLog, completedAt,
		); err != nil {
			return fmt.Errorf("insert ai_run for %s: %w", r.taskID[:8], err)
		}
	}

	if err := seedMissionControlToolsAndUsage(ctx, tx, userID, task1, task2, task3, task4); err != nil {
		return err
	}

	return seedMissionControlConversations(ctx, tx, userID, task1, task2)
}

// seedMissionControlToolsAndUsage seeds the activity stream (tool invocations
// across 3 agents) and the fleet status-bar spend (4 LLM usage rows totalling
// $1.82). Extracted from seedMissionControl so the parent stays under
// golangci-lint's cyclomatic-complexity ceiling.
func seedMissionControlToolsAndUsage(ctx context.Context, tx pgx.Tx, userID, task1, task2, task3, task4 string) error {
	// Run IDs from seedMissionControl runs block — task2 → executing run,
	// task3 → completed run. task1 has no run (it's still awaiting_approval).
	// PR #16: link the per-task invocations to their runs so the audit
	// timeline renders the tool invocations under each phase_complete entry.
	const (
		runTask2 = "e3000000-0000-0000-0000-000000000001"
		runTask3 = "e3000000-0000-0000-0000-000000000002"
	)
	invocations := []struct {
		id, taskID, runID, toolName, risk string
		durationMs                        int
	}{
		{"e4000000-0000-0000-0000-000000000001", task1, "", "list_cve_alerts", "read_only", 110},
		{"e4000000-0000-0000-0000-000000000002", task1, "", "calculate_blast_radius", "read_only", 230},
		{"e4000000-0000-0000-0000-000000000003", task2, runTask2, "query_assets", "read_only", 80},
		{"e4000000-0000-0000-0000-000000000004", task2, runTask2, "analyze_drift", "read_only", 450},
		{"e4000000-0000-0000-0000-000000000005", task3, runTask3, "list_certificates", "read_only", 95},
		{"e4000000-0000-0000-0000-000000000006", task3, runTask3, "propose_cert_rotation", "plan_only", 320},
	}
	for i := range invocations {
		v := &invocations[i]
		var runID any
		if v.runID != "" {
			runID = v.runID
		}
		// Refresh `created_at` on every seed run so fleet_status's "today"
		// tool-invocations counter (`WHERE created_at::date = CURRENT_DATE`)
		// keeps reporting the seeded baseline of 6.
		if _, err := tx.Exec(ctx, `
			INSERT INTO ai_tool_invocations (id, task_id, run_id, tool_name, risk_level, duration_ms, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW())
			ON CONFLICT (id) DO UPDATE SET
				run_id = EXCLUDED.run_id,
				duration_ms = EXCLUDED.duration_ms,
				created_at = NOW()`,
			v.id, v.taskID, runID, v.toolName, v.risk, v.durationMs,
		); err != nil {
			return fmt.Errorf("insert ai_tool_invocation %s: %w", v.toolName, err)
		}
	}

	// LLM usage rows so the fleet status-bar spend is a real number — total
	// 182 cents = $1.82 today across 4 tasks.
	usages := []struct {
		id, taskID                      string
		agentName                       string
		inputTokens, outputTokens       int
		inputCostCents, outputCostCents int
	}{
		{"e5000000-0000-0000-0000-000000000001", task1, "vulnerability_agent", 1200, 800, 40, 20},
		{"e5000000-0000-0000-0000-000000000002", task2, "drift_agent", 900, 600, 30, 20},
		{"e5000000-0000-0000-0000-000000000003", task3, "certificate_agent", 700, 500, 25, 20},
		{"e5000000-0000-0000-0000-000000000004", task4, "dr_agent", 500, 400, 18, 9},
	}
	for i := range usages {
		u := &usages[i]
		// Refresh `timestamp` on every seed run so the "today" counter on
		// fleet status (`WHERE timestamp::date = CURRENT_DATE`) keeps reporting
		// the seeded $1.82 spend regardless of how many days the row has been
		// in the DB. Previously the row used ON CONFLICT DO NOTHING and the
		// counter drained to 0 after the first day.
		if _, err := tx.Exec(ctx, `
			INSERT INTO llm_usage (id, org_id, user_id, task_id, agent_name, request_id,
				provider, model, input_tokens, output_tokens,
				input_cost_cents, output_cost_cents, operation_type, status, timestamp)
			VALUES ($1, $2, $3, $4, $5, gen_random_uuid(),
				'azure_anthropic', 'claude-sonnet-4-5', $6, $7, $8, $9, 'plan_generation', 'success', NOW())
			ON CONFLICT (id) DO UPDATE SET timestamp = NOW()`,
			u.id, orchestratorDevOrgID, userID, u.taskID, u.agentName,
			u.inputTokens, u.outputTokens, u.inputCostCents, u.outputCostCents,
		); err != nil {
			return fmt.Errorf("insert llm_usage for %s: %w", u.agentName, err)
		}
	}

	return nil
}

// seedMissionControlConversations seeds the Phase B.2 conversation surface:
//
//	A — active (5m old). Linked to task1 (Patch CVE-2024-3094), the seeded
//	    pending decision. Active on landing so the dock thread and the
//	    pending decision rail tell the same story on a cold demo open
//	    (PR #17 / UX-002). Previously B was active and A was stale — the
//	    swap removes the "why is the chat about drift but pending shows
//	    CVE?" confusion in the first 30 seconds of the demo.
//	B — stale (2h old). Linked to task2 (drift). Kept stale so the seed
//	    still exercises the B.2 60-min append window's "stale conversation
//	    exists but isn't surfaced" behavior.
//
// Tasks 1 and 2 are linked back to A and B respectively via
// ai_tasks.conversation_id so the seeded thread maps to real task rows.
//
// Split out from seedMissionControl so the parent stays under golangci-lint's
// cyclomatic-complexity ceiling.
func seedMissionControlConversations(ctx context.Context, tx pgx.Tx, userID, task1, task2 string) error {
	const (
		convA = "e6000000-0000-0000-0000-000000000001"
		convB = "e6000000-0000-0000-0000-000000000002"
	)

	convSeeds := []struct {
		id, title string
		ageMins   int
	}{
		{convA, "Patch CVE-2024-3094 (xz backdoor) on production assets", 5}, // active (PR #17)
		{convB, "Analyze drift across azure production sites", 120},          // stale (PR #17)
	}
	for i := range convSeeds {
		c := &convSeeds[i]
		if _, err := tx.Exec(ctx, `
			INSERT INTO ai_conversations (id, org_id, created_by, title, state, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 'active', NOW() - make_interval(mins => $5), NOW() - make_interval(mins => $5))
			ON CONFLICT (id) DO UPDATE SET
				updated_at = EXCLUDED.updated_at,
				title = EXCLUDED.title`,
			c.id, orchestratorDevOrgID, userID, c.title, c.ageMins,
		); err != nil {
			return fmt.Errorf("insert ai_conversation %s: %w", c.id[:8], err)
		}
	}

	taskLinks := []struct{ convID, taskID string }{
		{convA, task1},
		{convB, task2},
	}
	for _, l := range taskLinks {
		if _, err := tx.Exec(ctx,
			`UPDATE ai_tasks SET conversation_id = $1 WHERE id = $2`,
			l.convID, l.taskID,
		); err != nil {
			return fmt.Errorf("link task %s to conv: %w", l.taskID[:8], err)
		}
	}

	// Messages. Each thread has one user prompt and one assistant summary.
	// The assistant summaries match the shape of synthesizeAssistantMessage
	// output so the seeded view looks identical to a live submission.
	msgSeeds := []struct {
		id, convID, role, content, taskID string
		ageMinsOffset                     int
	}{
		// Conv A (active, 5m old) — CVE patch thread.
		{"e7000000-0000-0000-0000-000000000001", convA, "user",
			"Patch CVE-2024-3094 (xz backdoor) on production assets",
			task1, 5},
		{"e7000000-0000-0000-0000-000000000002", convA, "assistant",
			`Drafted plan-only patch_rollout for: "Patch CVE-2024-3094 (xz backdoor) on production assets". Risk: high (HITL required). Quality: 87/100. Awaiting your approval.`,
			task1, 4},
		// Conv B (stale, 2h old) — drift thread.
		{"e7000000-0000-0000-0000-000000000003", convB, "user",
			"Analyze drift across azure production sites",
			task2, 120},
		{"e7000000-0000-0000-0000-000000000004", convB, "assistant",
			`Drafted plan-only drift_remediation for: "Analyze drift across azure production sites". Risk: medium. Quality: 92/100. Awaiting your approval.`,
			task2, 119},
	}
	for i := range msgSeeds {
		m := &msgSeeds[i]
		if _, err := tx.Exec(ctx, `
			INSERT INTO ai_conversation_messages (id, conversation_id, role, content, task_id, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW() - make_interval(mins => $6))
			ON CONFLICT (id) DO UPDATE SET
				content = EXCLUDED.content,
				created_at = EXCLUDED.created_at`,
			m.id, m.convID, m.role, m.content, m.taskID, m.ageMinsOffset,
		); err != nil {
			return fmt.Errorf("insert ai_conversation_message %s: %w", m.id[:8], err)
		}
	}

	return nil
}

func ptrString(s string) *string { return &s }
