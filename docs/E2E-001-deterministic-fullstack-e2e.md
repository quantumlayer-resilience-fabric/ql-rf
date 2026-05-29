# E2E-001: Deterministic Full-Stack E2E Fixture

**Status:** In progress — branch `ci/e2e-001-deterministic-fixture` (overview milestone)
**Created:** 2026-05-28

## Progress (2026-05-29)

- **Auth mode fixed.** The API ignored `RF_DEV_MODE` (dev mode only kicked in
  when Clerk was unconfigured). Added a top-level `RF_DEV_MODE` config flag,
  honored in `services/api/internal/routes/routes.go`, so the API can skip JWT
  validation while Clerk-configured and resolve a deterministic default org.
  Production default is unchanged (`dev_mode=false`). Verified locally: with the
  rebuilt API in dev mode, `GET /api/v1/overview/metrics` returns the seeded
  org's data (`fleetSize: 10`) instead of 401.
- **Deterministic seed added.** `scripts/seed-e2e-data` (idempotent, fixed IDs,
  org `created_at` pinned to the past so dev-mode resolves to it). Seeds the
  Overview-relevant entities: 1 org, 3 sites (1 DR-paired), 10 assets (5 aws /
  3 azure / 2 gcp), 4 images, 3 drift reports, 4 alerts.
- **CI job wired (still advisory).** `frontend-e2e` now `docker compose up`s the
  full stack, applies migrations (compose `migrate`), runs the seed, waits for
  health, and runs the overview spec. `continue-on-error` and the ci-complete
  carve-out remain until the suite is reliably green.

Remaining for the milestone: confirm the overview spec passes in CI against the
seeded stack. Then expand page-by-page and finally make E2E blocking again.

## Summary

The Playwright E2E suite (`ui/control-tower/e2e/*.spec.ts`) is currently
**advisory / non-blocking** in CI (`continue-on-error: true` on the
`frontend-e2e` job; excluded from the `ci-complete` gate). This issue tracks the
work to make it a reliable, blocking gate again.

## Why it's advisory right now

The specs assert against a populated, authenticated dashboard, but CI has no way
to produce that state yet. Validated locally on 2026-05-28:

- ✅ The full stack starts via `docker compose` (api, orchestrator, postgres,
  redis, opa, temporal all become healthy).
- ✅ Clerk auth works end-to-end — the `E2E_CLERK_USER_*` credentials are valid
  and the auth state is saved (`playwright/.clerk/user.json`).
- ❌ Dashboard specs fail because there is **no seeded data**: the DB has no
  sites/assets/drift/alerts, so the widgets render nothing to assert on.
- ⚠️ The API rejected requests with `invalid or expired token` even with
  `RF_DEV_MODE=true` in the container — the CI/dev auth path needs to be made
  explicit so the frontend's calls are accepted deterministically.
- ⚠️ The existing `SeedDemoData` only creates sites/assets/images — not drift
  reports, alerts, compliance results, SBOMs, or CVE alerts that other specs
  assert on.

Everything else in CI (Go lint/test/build, frontend lint/build, all Docker
images, security scan) is green and remains **blocking**.

## Scope / tasks

1. Fix the container/CI auth mode so API + orchestrator accept the frontend's
   requests deterministically (document the dev-auth or test-token path).
2. Start the full app stack in CI (API, orchestrator, frontend, Postgres) and
   apply migrations.
3. Seed a **deterministic test org** plus a complete demo dataset (sites,
   assets, images, drift reports, alerts, compliance results, SBOMs, CVE
   alerts) so every covered page has stable expected data.
4. Cover the dashboard pages the specs target: overview, assets, drift,
   compliance, risk, images, sbom, inspec, resilience, vulnerabilities, ai.
5. Have Playwright wait on health/readiness of the stack before running.
6. Harden specs so they fail on **real regressions**, not on empty fixtures.

## Acceptance criteria

1. CI starts API, orchestrator, frontend, and Postgres.
2. Migrations are applied.
3. A deterministic test org is seeded.
4. The Clerk / dev-auth path is explicit and documented.
5. The overview dashboard has stable, expected data.
6. Playwright waits on health/readiness before running specs.
7. The `frontend-e2e` job becomes blocking again (remove `continue-on-error`
   and restore it in the `ci-complete` gate).

## E2E-002: Sites page (2026-05-29)

Second page on the deterministic foundation. **Note:** there is no dedicated
**Assets page** today — no `/assets` route or sidebar item exists, and the
`useAssets` hook is unused. So E2E-002 was retargeted to **Sites**, the nearest
existing infrastructure-inventory surface (a future Assets page is tracked as
`UI-001` in `docs/BACKLOG.md`).

The E2E seed now links assets to their site via `site_id`, giving deterministic
per-site counts: **AWS US-East-1 = 5, Azure East US = 3, GCP US-Central1 = 2**
(DR-paired AWS↔Azure). `e2e/sites.spec.ts` asserts the seeded site names +
regions, the Total Sites / Total Assets / DR Pairs metric cards, and that the
platform filter narrows to the matching site. The CI job now runs
`overview.spec.ts` + `sites.spec.ts`; E2E remains **advisory** until more pages
are covered.

## E2E-003: Drift page (2026-05-29)

Third page on the deterministic foundation. The Drift Analysis page computes
drift **live from assets**, not from the `drift_reports` table: an asset is
compliant when its `image_ref`/`image_version` match a **production** golden
image (see `CountCompliantAssets`). The E2E seed was extended so the 4 golden
images are `status='production'` and each asset references one — 8 compliant, 2
(one Azure, one GCP) pointing at an unmanaged `legacy-unmanaged@0.0.0`. This
yields deterministic coverage **AWS 100% / Azure 66.7% / GCP 50%** = 8/10 (80%),
so `/api/v1/drift/summary` returns a stable `{totalAssets: 10, driftedAssets: 2,
driftPercentage: 20, criticalDrift: 0}`.

`e2e/drift.spec.ts` (rewritten from the old mock-based version) asserts the page
header/description, the metric cards (Total Assets = 10, Drift Rate = 20.0%, "2
assets", Critical Drift, Avg Drift Age), and the data-derived AI insight
("Drift Pattern Analysis" → "2 assets have drifted from golden images."). The CI
job now runs `overview.spec.ts` + `sites.spec.ts` + `drift.spec.ts`; E2E remains
**advisory** until more pages are covered.

## E2E-004: Images page (2026-05-29)

Fourth page on the deterministic foundation, reusing the golden-image fixture
that E2E-003 already depends on (no fixture expansion needed). The seed's 4
production golden images (one per family) are grouped by the API client into 4
single-version families:

| Family | Latest version | Status |
|--------|----------------|--------|
| `golden-amazonlinux-2023` | v2024.11.0 | production |
| `golden-rhel-9` | v2024.09.0 | production |
| `golden-ubuntu-22` | v2024.11.0 | production |
| `golden-windows-2022` | v2024.10.0 | production |

So the page shows **Image Families = 4, Active Versions = 4, 0 pending, 0
deprecated**. `e2e/images.spec.ts` (rewritten from the old mock-based version)
asserts the header/description, the metric cards, all four seeded family names +
version codes + production status, and that filtering by **Deprecated** (none
seeded) shows the empty state. The CI job now runs `overview.spec.ts` +
`sites.spec.ts` + `drift.spec.ts` + `images.spec.ts`; E2E remains **advisory**
until more pages are covered.

**Next:** E2E-005 Risk, then E2E-006 CVE / Vulnerability (both likely need
fixture expansion). Image *lineage* deep coverage is deferred (E2E-004B/E2E-007)
— this PR covers the inventory surface only.

## E2E-005: Risk page (2026-05-29)

Fifth page on the deterministic foundation, and the first to require real
fixture expansion. The risk service scores each asset from drift age, open
vulnerabilities, critical vulnerabilities, compliance, and an **environment
multiplier** (production 1.5x). The seed now adds:

- a **project + environments** (production/staging/development) and sets every
  asset's `env_id` to **production** (so the 1.5x multiplier applies);
- **image_vulnerabilities** on two golden images, each used by exactly one asset
  so the score is isolated:
  - `golden-rhel-9`: 4 critical + 16 high (20 open) → `e2e-aws-risk-high` scores **68 (HIGH)**
  - `golden-windows-2022`: 4 critical (4 open) → `e2e-azure-risk-medium` scores **44 (MEDIUM)**

This produces a stable distribution: **0 critical, 1 high, 1 medium, 8 low**,
verified directly against `/api/v1/risk/summary` before writing assertions.

**Why no critical-risk asset:** a non-drifted asset caps at 67.5 (= high) even
with maxed vulnerabilities and the production multiplier. Reaching the critical
band (≥80) needs the drift/compliance factors, which are coupled to the Drift
page's drifted count — so seeding a critical-risk asset would break the E2E-003
drift invariant. We keep all assets in one production environment for the same
reason: it leaves the drift coverage as a single 80% ("warning") group so
`criticalDrift` stays 0 and the Drift spec's "Drift Pattern Analysis" insight is
unchanged. Drift (8 compliant / 2 drifted) and Images (4 families) invariants
are preserved.

`e2e/risk.spec.ts` asserts the header/description, the five summary cards, the
seeded high/medium assets (by name + level badge + critical-CVE factor) in the
top-risks table, and the level-badge distribution (1 high, 1 medium, 0
critical). The CI job now runs `overview` + `sites` + `drift` + `images` +
`risk`; E2E remains **advisory**.

**Next:** E2E-006 CVE / Vulnerability.

## E2E-006: Vulnerabilities page (2026-05-29)

Sixth page on the deterministic foundation. The Vulnerability Response Center is
served by the **orchestrator** (:8083), not the API — and in dev mode the
orchestrator hardcodes `org_id = 00000000-0000-0000-0000-000000000001` (see
`services/orchestrator/internal/middleware/auth.go`) rather than resolving the
oldest org like the API. So the seed adds that org (with a recent `created_at`
so the API's `lookupDefaultOrg` still picks the 2020 E2E org for every other
page) and seeds CVE data under it.

The list/summary endpoints (`cve_alerts_query.go`) read the **denormalized
`*_count` columns on `cve_alerts`** and LEFT JOIN `cve_cache` for
severity/KEV/exploit details, so the dashboard renders entirely from those two
tables. The seed inserts 5 `cve_cache` + `cve_alerts`:

| CVE | Severity | Urgency | Indicators |
|-----|----------|---------|------------|
| CVE-2024-3094 | critical | 98 | CISA KEV, exploit, SLA-breached |
| CVE-2021-44228 | critical | 95 | CISA KEV, exploit |
| CVE-2024-21626 | high | 72 | exploit |
| CVE-2023-44487 | high | 65 | — |
| CVE-2024-6387 | medium | 45 | resolved |

Summary: **5 total, 2 critical + 2 high, 2 CISA KEV, 3 exploitable, 1
SLA-breached, 15 affected assets (6 production)** — verified against
`/api/v1/cve-alerts` + `/summary` on :8083 before writing assertions.

`e2e/vulnerabilities.spec.ts` (rewritten from the mock-based version) asserts
the header/description, the four metric cards, the seeded critical/high CVEs
(severity + KEV indicator) in the alerts table, the CISA-KEV AI insight + SLA
warning, and the seeded tab counts. The CI job now runs `overview` + `sites` +
`drift` + `images` + `risk` + `vulnerabilities`; E2E remains **advisory**.

**Deferred:** the CVE detail / blast-radius page (`/vulnerabilities/[alertId]`,
reads `cve_alert_affected_items`) and patch-campaign execution, like image
lineage — this PR covers the list/response-center surface only.

**Next:** with six pages covered, the remaining candidates are Compliance,
SBOM, InSpec, Certificates, Resilience, Costs, and the AI Copilot — plus
flipping E2E from advisory to **blocking** once the suite is proven stable.

## E2E-007: promote Frontend E2E to blocking (2026-05-29)

The seeded full-stack suite (overview, sites, drift, images, risk,
vulnerabilities — 26 specs) has run green across six page-additions and the
intervening `main` merges, so it has earned the right to protect `main`. This is
a CI-policy-only change:

- removed `continue-on-error: true` from the `frontend-e2e` job, so a real spec
  failure fails the job;
- added `needs.frontend-e2e.result` to the `ci-complete` required gate, alongside
  the Go/frontend builds and the security scan.

No fixture, spec, or page-coverage changes — the spec list is unchanged. The
acceptance criterion #7 above ("the `frontend-e2e` job becomes blocking again")
is now satisfied.

## E2E-008: Certificates page (2026-05-29)

First page added **after** E2E became blocking, so it was validated locally
against the running stack before pushing. The Certificate Lifecycle Management
page is served by the API (:8080). The `certificates` table has a BEFORE
INSERT/UPDATE trigger that derives `days_until_expiry` and `status` from
`not_after`, so the seed sets `not_after` relative to now (computed at seed
time — deterministic for the immediately-following test run) and lets the
trigger classify each cert:

| Common name | Platform | Source | not_after | Status |
|-------------|----------|--------|-----------|--------|
| api.quantumlayer.io | aws | acm | ~+825d | active (auto-renew) |
| app.quantumlayer.io | azure | azure_keyvault | ~+600d | active (auto-renew) |
| dashboard.quantumlayer.io | gcp | gcp_certificate_manager | ~+15d | expiring_soon |
| legacy.quantumlayer.io | k8s | k8s_secret | ~+5d | expiring_soon (within 7) |
| old.quantumlayer.io | vsphere | file | ~-10d | expired |

Summary (verified against `/api/v1/certificates` + `/summary`): **5 total, 2
active, 2 expiring soon (1 within 7 days), 1 expired, 5 platforms, 2
auto-renew**. `e2e/certificates.spec.ts` asserts the header/description, the
metric cards (via their unique data-derived subtitles), the seeded common names
+ status badges in the inventory table, the `Certificates (5)` tab count, and
the `2 Certificates Need Attention` insight. Exact day counts are not asserted
(the trigger truncates). The blocking CI run now covers overview + sites +
drift + images + risk + vulnerabilities + certificates (30 specs).

The Alerts and Rotations tabs, certificate usage/blast-radius (detail), and
rotation execution are deferred — this PR covers the certificate inventory
surface only.
