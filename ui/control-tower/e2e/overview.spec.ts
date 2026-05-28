import { test, expect } from "@playwright/test";

test.describe("Overview Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/overview");
  });

  test("should display page title and description", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Overview" })).toBeVisible();
    await expect(
      page.getByText("Real-time visibility into your infrastructure health and compliance")
    ).toBeVisible();
  });

  test("should display key metrics cards", async ({ page }) => {
    // Wait for data to load
    await page.waitForLoadState("networkidle");

    // Scope to the main content area. "Compliance"/"DR Readiness" also appear in
    // the sidebar nav and the AI Insights widget, so we use exact text and take
    // the first match (the metric cards render before the insights in the DOM)
    // to stay unambiguous under Playwright strict mode.
    const main = page.getByRole("main");
    await expect(main.getByText("Fleet Size", { exact: true })).toBeVisible();
    await expect(main.getByText("Drift Score", { exact: true })).toBeVisible();
    await expect(main.getByText("Compliance", { exact: true }).first()).toBeVisible();
    await expect(main.getByText("DR Readiness", { exact: true }).first()).toBeVisible();
  });

  test("should display AI Insights widget", async ({ page }) => {
    await page.waitForLoadState("networkidle");

    // AI Insights section should be visible
    await expect(page.getByText("AI Insights")).toBeVisible();
  });

  test("should display Value Delivered card", async ({ page }) => {
    await page.waitForLoadState("networkidle");

    await expect(page.getByText("Value Delivered")).toBeVisible();
    await expect(page.getByText("Incidents Prevented")).toBeVisible();
    await expect(page.getByText("Hours Automated")).toBeVisible();
  });

  test("should display Active Alerts section", async ({ page }) => {
    await page.waitForLoadState("networkidle");

    await expect(page.getByText("Active Alerts")).toBeVisible();
  });

  test("should display Platform Distribution", async ({ page }) => {
    await page.waitForLoadState("networkidle");

    await expect(page.getByText("Platform Distribution")).toBeVisible();
  });

  test("should display Drift Heatmap", async ({ page }) => {
    await page.waitForLoadState("networkidle");

    await expect(page.getByText("Drift Heatmap by Site")).toBeVisible();
  });
});
