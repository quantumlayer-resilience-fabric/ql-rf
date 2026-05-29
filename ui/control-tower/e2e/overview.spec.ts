import { test, expect } from "@playwright/test";

// Data-dependent widgets can take a moment after the dashboard's queries
// resolve; give the visibility assertions a generous timeout. We deliberately
// do NOT use page.waitForLoadState("networkidle"): the dashboard has background
// queries (e.g. AI Insights) that keep the network busy, so networkidle never
// settles in CI. toBeVisible already auto-waits for each specific element.
const WIDGET_TIMEOUT = 20_000;

test.describe("Overview Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/overview");
    // Wait until the page shell has rendered (and any auth/onboarding redirect
    // has settled) before asserting individual widgets.
    await expect(page.getByRole("heading", { name: "Overview" })).toBeVisible({
      timeout: WIDGET_TIMEOUT,
    });
  });

  test("should display page title and description", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Overview" })).toBeVisible();
    await expect(
      page.getByText("Real-time visibility into your infrastructure health and compliance")
    ).toBeVisible();
  });

  test("should display key metrics cards", async ({ page }) => {
    // Scope to the main content area. "Compliance"/"DR Readiness" also appear in
    // the sidebar nav and the AI Insights widget, so we use exact text and take
    // the first match (the metric cards render before the insights in the DOM)
    // to stay unambiguous under Playwright strict mode.
    const main = page.getByRole("main");
    await expect(main.getByText("Fleet Size", { exact: true })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(main.getByText("Drift Score", { exact: true })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(main.getByText("Compliance", { exact: true }).first()).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(main.getByText("DR Readiness", { exact: true }).first()).toBeVisible({ timeout: WIDGET_TIMEOUT });
  });

  test("should display AI Insights widget", async ({ page }) => {
    await expect(page.getByText("AI Insights")).toBeVisible({ timeout: WIDGET_TIMEOUT });
  });

  test("should display Value Delivered card", async ({ page }) => {
    await expect(page.getByText("Value Delivered")).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(page.getByText("Incidents Prevented")).toBeVisible();
    await expect(page.getByText("Hours Automated")).toBeVisible();
  });

  test("should display Active Alerts section", async ({ page }) => {
    await expect(page.getByText("Active Alerts")).toBeVisible({ timeout: WIDGET_TIMEOUT });
  });

  test("should display Platform Distribution", async ({ page }) => {
    await expect(page.getByText("Platform Distribution")).toBeVisible({ timeout: WIDGET_TIMEOUT });
  });

  test("should display Drift Heatmap", async ({ page }) => {
    await expect(page.getByText("Drift Heatmap by Site")).toBeVisible({ timeout: WIDGET_TIMEOUT });
  });
});
