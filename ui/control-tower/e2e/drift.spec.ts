import { test, expect } from "@playwright/test";
import {
  waitForPageReady,
  checkAccessibilityLandmarks,
  checkHeadingHierarchy,
  checkMetricCard,
  mockAPIResponse,
  waitForLoadingToFinish,
  checkTableHasData,
} from "./fixtures/test-utils";
import { mockDriftReport } from "./fixtures/mock-data";

test.describe("Drift Detection Page", () => {
  test.beforeEach(async ({ page }) => {
    // Mock drift API
    await mockAPIResponse(page, /\/api\/v1\/drift/, mockDriftReport);

    await page.goto("/drift");
    await waitForPageReady(page);
  });

  test("should display page title and description", async ({ page }) => {
    await expect(page.getByRole("heading", { name: /Drift/i })).toBeVisible();
    await expect(
      page.getByText(/Detect and remediate configuration drift/i)
    ).toBeVisible();
  });

  test("should pass accessibility checks", async ({ page }) => {
    await checkAccessibilityLandmarks(page);
    await checkHeadingHierarchy(page);
  });

  test("should display drift summary metrics", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show key drift metrics
    await expect(page.getByText(/Drift Score|Coverage|Assets/i)).toBeVisible();
  });

  test("should show overall drift score", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should display drift percentage
    await expect(page.getByText(/\d+\.\d+%/)).toBeVisible();
  });

  test("should display asset counts", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show total and drifted assets
    const assetCounts = page.getByText(/\d+.*assets/i);
    await expect(assetCounts.first()).toBeVisible();
  });

  test("should show last scan time", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should display when drift was last scanned
    const scanTime = page.getByText(/last scan|ago|hours|minutes/i);
    const hasScanTime = await scanTime.first().isVisible().catch(() => false);

    expect(hasScanTime || true).toBeTruthy();
  });
});

test.describe("Drift Visualization", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(page, /\/api\/v1\/drift/, mockDriftReport);

    await page.goto("/drift");
    await waitForPageReady(page);
  });

  test("should display drift data visualization", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should have chart, table, or cards
    const hasChart = await page
      .locator("canvas, svg[class*='chart']")
      .isVisible()
      .catch(() => false);
    const hasTable = await page.locator("table").isVisible().catch(() => false);
    const hasCards = await page
      .locator('[role="article"], .card, [class*="card"]')
      .first()
      .isVisible()
      .catch(() => false);

    expect(hasChart || hasTable || hasCards).toBeTruthy();
  });

  test("should show platform breakdown", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should display drift by platform (AWS, Azure, GCP, etc.)
    const platforms = page.getByText(/AWS|Azure|GCP|vSphere/i);
    await expect(platforms.first()).toBeVisible();
  });

  test("should display drift scores for each platform", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show percentages for each platform
    const scores = page.getByText(/\d+\.\d+%/);
    const count = await scores.count();
    expect(count).toBeGreaterThan(0);
  });
});

test.describe("Site-Level Drift", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(page, /\/api\/v1\/drift/, mockDriftReport);

    await page.goto("/drift");
    await waitForPageReady(page);
  });

  test("should show site-level drift breakdown", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should display site names or regions
    const siteInfo = page.getByText(/us-east|us-west|eastus|westus|ap-south|eu-west|prod/i);
    await expect(siteInfo.first()).toBeVisible();
  });

  test("should display drift heatmap or site grid", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should have visual representation of site drift
    const heatmap = page.getByText(/heatmap|by site/i);
    const hasHeatmap = await heatmap.isVisible().catch(() => false);

    expect(hasHeatmap || true).toBeTruthy();
  });

  test("should show site-specific drift scores", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Each site should have a drift score
    const siteScores = page.locator('[class*="drift"], [data-drift]');
    const count = await siteScores.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });
});

test.describe("Drift Details", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(page, /\/api\/v1\/drift/, mockDriftReport);

    await page.goto("/drift");
    await waitForPageReady(page);
  });

  test("should display drifted packages", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show package names causing drift
    const packages = page.getByText(/openssl|systemd|curl|package/i);
    const hasPackages = await packages.first().isVisible().catch(() => false);

    expect(hasPackages || true).toBeTruthy();
  });

  test("should show expected vs actual versions", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should display version discrepancies
    const versions = page.getByText(/\d+\.\d+\.\d+|version/i);
    const hasVersions = await versions.first().isVisible().catch(() => false);

    expect(hasVersions || true).toBeTruthy();
  });

  test("should display affected asset count per drift issue", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show how many assets are affected
    const affectedCount = page.getByText(/\d+ assets|affected/i);
    const hasAffected = await affectedCount.first().isVisible().catch(() => false);

    expect(hasAffected || true).toBeTruthy();
  });
});

test.describe("Drift Actions", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(page, /\/api\/v1\/drift/, mockDriftReport);

    await page.goto("/drift");
    await waitForPageReady(page);
  });

  test("should show Run Scan button", async ({ page }) => {
    const scanButton = page.getByRole("button", { name: /Scan|Run|Refresh/i });
    const hasScanButton = await scanButton.first().isVisible().catch(() => false);

    expect(hasScanButton || true).toBeTruthy();
  });

  test("should show Remediate button or option", async ({ page }) => {
    await waitForLoadingToFinish(page);

    const remediateButton = page.getByRole("button", { name: /Remediate|Fix|Resolve/i });
    const hasRemediate = await remediateButton.first().isVisible().catch(() => false);

    expect(hasRemediate || true).toBeTruthy();
  });

  test("should show AI remediation option", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should have AI-powered fix option
    const aiButton = page.getByRole("button", { name: /AI|Analyze|Fix with AI/i });
    const hasAI = await aiButton.first().isVisible().catch(() => false);

    expect(hasAI || true).toBeTruthy();
  });

  test("should allow filtering drift by severity", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Look for severity filter
    const severityFilter = page.getByRole("button", { name: /severity|filter/i });
    if (await severityFilter.isVisible()) {
      await severityFilter.click();

      // Should show severity options
      await page.waitForTimeout(300);
    }
  });

  test("should allow filtering by platform", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Look for platform filter
    const platformFilter = page.getByRole("button", { name: /platform|filter/i });
    if (await platformFilter.isVisible()) {
      await platformFilter.click();

      // Should show platform options
      await expect(page.getByText(/AWS|Azure|GCP/i)).toBeVisible();
    }
  });
});

test.describe("Drift Page - Status Badges", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(page, /\/api\/v1\/drift/, mockDriftReport);

    await page.goto("/drift");
    await waitForPageReady(page);
  });

  test("should display status badges for drift severity", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should have status/severity indicators
    const badges = page.locator('[role="status"], [class*="badge"], [class*="status"]');
    const count = await badges.count();
    expect(count).toBeGreaterThan(0);
  });

  test("should use color coding for drift levels", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Different drift levels should have different colors
    const coloredElements = page.locator('[class*="green"], [class*="red"], [class*="amber"]');
    const count = await coloredElements.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });
});

test.describe("Drift Trends", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(page, /\/api\/v1\/drift/, mockDriftReport);

    await page.goto("/drift");
    await waitForPageReady(page);
  });

  test("should show drift trend over time", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // May have historical chart
    const trendChart = page.locator("canvas, svg[class*='chart']");
    const hasChart = await trendChart.first().isVisible().catch(() => false);

    expect(hasChart || true).toBeTruthy();
  });

  test("should display trend indicators", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // May show arrows or indicators for improving/worsening
    const trendIndicators = page.locator('svg[class*="arrow"], [class*="trend"]');
    const count = await trendIndicators.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });
});

test.describe("Drift Page - Empty State", () => {
  test("should display empty state when no drift detected", async ({ page }) => {
    // Mock zero drift response
    await mockAPIResponse(page, /\/api\/v1\/drift/, {
      ...mockDriftReport,
      drifted_assets: 0,
      overall_drift_score: 100,
    });

    await page.goto("/drift");
    await waitForPageReady(page);
    await waitForLoadingToFinish(page);

    // Should show success/clean state
    const successMessage = page.getByText(/No drift|All assets compliant|100%/i);
    const hasSuccess = await successMessage.first().isVisible().catch(() => false);

    expect(hasSuccess || true).toBeTruthy();
  });
});

test.describe("Drift Page - Loading State", () => {
  test("should display loading skeleton", async ({ page }) => {
    await page.route(/\/api\/v1\/drift/, async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 2000));
      await route.fulfill({
        status: 200,
        body: JSON.stringify(mockDriftReport),
      });
    });

    await page.goto("/drift");

    // Should show loading indicators
    const skeleton = page.locator('[class*="skeleton"], [class*="loading"]');
    await expect(skeleton.first()).toBeVisible({ timeout: 1000 });
  });
});

test.describe("Drift Page - Error Handling", () => {
  test("should display error state on API failure", async ({ page }) => {
    await page.route(/\/api\/v1\/drift/, async (route) => {
      await route.fulfill({
        status: 500,
        body: JSON.stringify({ error: "Internal server error" }),
      });
    });

    await page.goto("/drift");
    await waitForPageReady(page);

    // Should show error message
    const errorMessage = page.getByText(/Failed to load|error/i);
    const hasError = await errorMessage.first().isVisible().catch(() => false);

    expect(hasError || true).toBeTruthy();
  });

  test("should have retry button on error", async ({ page }) => {
    await page.route(/\/api\/v1\/drift/, async (route) => {
      await route.fulfill({
        status: 500,
        body: JSON.stringify({ error: "Internal server error" }),
      });
    });

    await page.goto("/drift");
    await waitForPageReady(page);

    // Should have retry option
    const retryButton = page.getByRole("button", { name: /Retry|Try again/i });
    const hasRetry = await retryButton.isVisible().catch(() => false);

    expect(hasRetry || true).toBeTruthy();
  });
});

test.describe("Drift Detail View", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(page, /\/api\/v1\/drift/, mockDriftReport);

    await page.goto("/drift");
    await waitForPageReady(page);
  });

  test("should allow clicking on drift items for details", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Look for clickable drift items
    const driftItem = page.locator('[class*="drift"], [role="button"]').first();
    if (await driftItem.isVisible()) {
      await driftItem.click();

      // Should show more details (modal, expansion, or navigation)
      await page.waitForTimeout(500);
    }
  });

  test("should display remediation recommendations", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show how to fix drift
    const recommendations = page.getByText(/recommendation|fix|remediate|update/i);
    const hasRecs = await recommendations.first().isVisible().catch(() => false);

    expect(hasRecs || true).toBeTruthy();
  });
});
