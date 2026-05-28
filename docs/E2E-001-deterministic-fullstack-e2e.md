# E2E-001: Deterministic Full-Stack E2E Fixture

**Status:** Open (tracked follow-up)
**Created:** 2026-05-28

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
