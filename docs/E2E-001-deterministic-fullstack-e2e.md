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
