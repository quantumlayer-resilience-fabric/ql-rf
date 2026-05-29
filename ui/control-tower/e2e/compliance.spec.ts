import { test, expect } from "@playwright/test";

// Compliance runs against the deterministic fixture seeded by
// scripts/seed-e2e-data. The service joins compliance_frameworks ->
// compliance_controls -> compliance_results to compute scores, so the seed
// inserts two frameworks with per-control results:
//   CIS Benchmark : 5 controls, 3 passing + 2 failing -> 60.0% (failing)
//   SLSA          : 1 control, 1 passing, level 3    -> 100.0% (passing)
// Resulting summary (verified via /compliance/summary): overall 80.0%, cis
// 60.0%, slsa level 3, sigstore 0.0% (no image_compliance rows seeded),
// 2 failing controls (1 high + 1 medium). We deliberately do NOT use
// networkidle; toBeVisible auto-waits per element.
const WIDGET_TIMEOUT = 20_000;

test.describe("Compliance Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/compliance");
    await expect(page.getByRole("heading", { name: "Compliance", exact: true })).toBeVisible({
      timeout: WIDGET_TIMEOUT,
    });
  });

  test("should display the header and description", async ({ page }) => {
    const main = page.getByRole("main");
    await expect(
      main.getByText("Monitor compliance posture across security frameworks and standards.", {
        exact: true,
      }),
    ).toBeVisible({ timeout: WIDGET_TIMEOUT });
  });

  test("should display the compliance metric cards with seeded values", async ({ page }) => {
    const main = page.getByRole("main");
    // Card titles.
    await expect(main.getByText("Overall Score", { exact: true })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(main.getByText("CIS Compliance", { exact: true })).toBeVisible();
    await expect(main.getByText("SLSA Level", { exact: true })).toBeVisible();
    await expect(main.getByText("Sigstore Verified", { exact: true })).toBeVisible();

    // Data-derived values unique on the page (60.0% appears in both the CIS
    // card and the CIS framework card, so prefer the unique markers).
    await expect(main.getByText("80.0%", { exact: true })).toBeVisible();
    await expect(main.getByText("Level 3", { exact: true })).toBeVisible();
  });

  test("should list the seeded frameworks with their statuses", async ({ page }) => {
    const main = page.getByRole("main");
    // Default tab is Frameworks, so the framework cards render on first paint.
    await expect(main.getByText("CIS Benchmark", { exact: true })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    // The SLSA framework's CardTitle inlines an L3 badge (text becomes "SLSAL3"),
    // so the L3 badge itself is the unique marker.
    await expect(main.getByText("L3", { exact: true })).toBeVisible();
    // Status text rendered inside each framework's status badge.
    await expect(main.getByText("failing", { exact: true })).toBeVisible();
    await expect(main.getByText("passing", { exact: true })).toBeVisible();
  });

  test("should surface the seeded compliance-gap AI insight", async ({ page }) => {
    const main = page.getByRole("main");
    // failingControls.length = 2 -> "2 Compliance Gaps Detected"; overallScore
    // 80 falls in [80, 95) -> warning variant.
    await expect(main.getByText("2 Compliance Gaps Detected", { exact: true })).toBeVisible({
      timeout: WIDGET_TIMEOUT,
    });
  });
});
