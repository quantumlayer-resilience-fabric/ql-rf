# Backlog

Tracked items that are intentionally deferred — not implemented yet.

## UI-001: Decide whether to add a dedicated Assets page

**Status:** Open (decision needed)
**Raised:** 2026-05-29 (during E2E-002)

### Context

The API (`GET /api/v1/assets`) and the frontend `useAssets` hook
(`ui/control-tower/src/hooks/use-assets.ts`) both exist, but there is **no
`/assets` route, no sidebar nav item, and nothing imports `useAssets`** — it's
unused. Assets are currently represented indirectly through the **Sites** page
(per-site counts) and detail views (vulnerability blast radius, patch-campaign
targets).

A dedicated Assets page would be **product/UI work**, not E2E infrastructure, so
it was kept out of the E2E-002 branch. E2E-002 was retargeted to the Sites page
(see `docs/E2E-001-deterministic-fullstack-e2e.md`).

### Acceptance criteria (if built later)

- `/assets` route
- sidebar nav item
- uses the existing `useAssets` hook
- table/list of assets
- filters by platform, environment, site, status
- deterministic E2E coverage (extend `scripts/seed-e2e-data` as needed)
