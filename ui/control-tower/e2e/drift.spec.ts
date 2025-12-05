import { test, expect } from "@playwright/test";

test.describe("Drift Detection Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/drift");
  });

  test("should display page title", async ({ page }) => {
    await expect(page.getByRole("heading", { name: /Drift/i })).toBeVisible();
  });

  test("should display drift summary metrics", async ({ page }) => {
    await page.waitForLoadState("networkidle");

    // Check for drift-related metrics
    await expect(page.getByText(/Coverage|Drift Score|Assets/i)).toBeVisible();
  });

  test("should display drift data visualization", async ({ page }) => {
    await page.waitForLoadState("networkidle");

    // Should have either a chart, table, or cards showing drift data
    const hasChart = await page.locator("canvas, svg[class*='chart']").isVisible().catch(() => false);
    const hasTable = await page.locator("table").isVisible().catch(() => false);
    const hasCards = await page.locator('[class*="card"]').first().isVisible().catch(() => false);

    expect(hasChart || hasTable || hasCards).toBeTruthy();
  });

  test("should show site-level drift breakdown", async ({ page }) => {
    await page.waitForLoadState("networkidle");

    // Should display site names or regions
    const siteInfo = page.getByText(/us-east|us-west|eastus|westus|ap-south|eu-west/i);
    await expect(siteInfo.first()).toBeVisible();
  });
});
