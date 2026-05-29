import { test, expect } from "@playwright/test";

// Drift Analysis runs against the deterministic E2E fixture seeded by
// scripts/seed-e2e-data. The page computes drift LIVE from assets (not the
// drift_reports table): an asset is compliant when its image_ref/image_version
// match a production golden image. The seed links 10 assets so that 8 are
// compliant and 2 (one Azure, one GCP) reference an unmanaged image:
//   total = 10, drifted = 2, drift rate = 20.0%, critical = 0.
// These are the stable facts the specs assert. We deliberately do NOT use
// waitForLoadState("networkidle") (background queries keep the network busy);
// toBeVisible auto-waits per element.
const WIDGET_TIMEOUT = 20_000;

test.describe("Drift Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/drift");
    await expect(page.getByRole("heading", { name: "Drift Analysis", exact: true })).toBeVisible({
      timeout: WIDGET_TIMEOUT,
    });
  });

  test("should display the header and description", async ({ page }) => {
    const main = page.getByRole("main");
    await expect(
      main.getByText("Monitor and remediate configuration drift across your infrastructure.", {
        exact: true,
      }),
    ).toBeVisible({ timeout: WIDGET_TIMEOUT });
  });

  test("should display drift metric cards with seeded totals", async ({ page }) => {
    const main = page.getByRole("main");

    // Card titles.
    await expect(main.getByText("Total Assets", { exact: true })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(main.getByText("Drift Rate", { exact: true })).toBeVisible();
    await expect(main.getByText("Critical Drift", { exact: true })).toBeVisible();
    await expect(main.getByText("Avg Drift Age", { exact: true })).toBeVisible();

    // Seeded values: 10 assets total, 2 drifted -> 20.0% drift rate, 0 critical.
    await expect(main.getByText("10", { exact: true }).first()).toBeVisible();
    await expect(main.getByText("20.0%", { exact: true })).toBeVisible();
    await expect(main.getByText("2 assets", { exact: true })).toBeVisible();
  });

  test("should display the data-derived AI drift insight", async ({ page }) => {
    const main = page.getByRole("main");
    // With 2 drifted and 0 critical assets the page renders the non-critical
    // "Drift Pattern Analysis" insight, quoting the drifted count.
    await expect(main.getByText("Drift Pattern Analysis", { exact: true })).toBeVisible({
      timeout: WIDGET_TIMEOUT,
    });
    await expect(
      main.getByText("2 assets have drifted from golden images.", { exact: false }),
    ).toBeVisible();
  });
});
