import { test, expect } from "@playwright/test";

// Sites is the nearest existing "infrastructure inventory" surface (there is no
// dedicated Assets page yet — see docs backlog UI-001). These specs assert
// against the deterministic E2E fixture seeded by scripts/seed-e2e-data:
//   - AWS US-East-1   (us-east-1, aws, 5 assets, DR-paired with Azure)
//   - Azure East US   (eastus, azure, 3 assets)
//   - GCP US-Central1 (us-central1, gcp, 2 assets)
// We deliberately do NOT use waitForLoadState("networkidle") (background queries
// keep the network busy); toBeVisible auto-waits per element.
const WIDGET_TIMEOUT = 20_000;

test.describe("Sites Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/sites");
    await expect(page.getByRole("heading", { name: "Sites", exact: true })).toBeVisible({
      timeout: WIDGET_TIMEOUT,
    });
  });

  test("should display the seeded sites with their regions", async ({ page }) => {
    const main = page.getByRole("main");
    await expect(main.getByText("AWS US-East-1", { exact: true })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(main.getByText("Azure East US", { exact: true })).toBeVisible();
    await expect(main.getByText("GCP US-Central1", { exact: true })).toBeVisible();

    await expect(main.getByText("us-east-1", { exact: true })).toBeVisible();
    await expect(main.getByText("eastus", { exact: true })).toBeVisible();
    await expect(main.getByText("us-central1", { exact: true })).toBeVisible();
  });

  test("should display site metric cards", async ({ page }) => {
    const main = page.getByRole("main");
    await expect(main.getByText("Total Sites", { exact: true })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(main.getByText("Total Assets", { exact: true })).toBeVisible();
    await expect(main.getByText("DR Pairs", { exact: true })).toBeVisible();
  });

  test("filtering by platform narrows to the matching site", async ({ page }) => {
    const main = page.getByRole("main");
    // Baseline: all three seeded sites present.
    await expect(main.getByText("AWS US-East-1", { exact: true })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(main.getByText("Azure East US", { exact: true })).toBeVisible();

    // Filter to AWS only.
    await page.getByRole("combobox").filter({ hasText: /platform/i }).click();
    await page.getByRole("option", { name: "AWS", exact: true }).click();

    await expect(main.getByText("AWS US-East-1", { exact: true })).toBeVisible();
    await expect(main.getByText("Azure East US", { exact: true })).toHaveCount(0);
    await expect(main.getByText("GCP US-Central1", { exact: true })).toHaveCount(0);
  });
});
