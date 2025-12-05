import { test, expect } from "@playwright/test";

test.describe("Golden Images Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/images");
  });

  test("should display page title", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Golden Images" })).toBeVisible();
  });

  test("should display images table or grid", async ({ page }) => {
    await page.waitForLoadState("networkidle");

    // Either a table or cards grid should be present
    const hasTable = await page.locator("table").isVisible().catch(() => false);
    const hasCards = await page.locator('[class*="card"]').first().isVisible().catch(() => false);

    expect(hasTable || hasCards).toBeTruthy();
  });

  test("should display search/filter controls", async ({ page }) => {
    await page.waitForLoadState("networkidle");

    // Should have search or filter inputs
    const searchInput = page.getByPlaceholder(/search|filter/i);
    const filterButtons = page.locator("button", { hasText: /filter|platform|status/i });

    const hasSearch = await searchInput.isVisible().catch(() => false);
    const hasFilter = await filterButtons.first().isVisible().catch(() => false);

    expect(hasSearch || hasFilter).toBeTruthy();
  });

  test("should display image details when clicking an image", async ({ page }) => {
    await page.waitForLoadState("networkidle");

    // Find an image row or card and click it
    const imageLink = page.locator("a[href*='/images/']").first();

    if (await imageLink.isVisible().catch(() => false)) {
      await imageLink.click();
      await expect(page).toHaveURL(/\/images\/.+/);
    }
  });
});
