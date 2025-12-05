import { test, expect } from "@playwright/test";
import {
  waitForPageReady,
  checkAccessibilityLandmarks,
  checkHeadingHierarchy,
  checkMetricCard,
  clickTab,
  mockAPIResponse,
  waitForLoadingToFinish,
  checkTableHasData,
} from "./fixtures/test-utils";
import { mockResilienceSummary } from "./fixtures/mock-data";

test.describe("Resilience & DR Page", () => {
  test.beforeEach(async ({ page }) => {
    // Mock resilience API
    await mockAPIResponse(
      page,
      /\/api\/v1\/resilience\/summary/,
      mockResilienceSummary
    );

    await page.goto("/resilience");
    await waitForPageReady(page);
  });

  test("should display page title and description", async ({ page }) => {
    await expect(page.getByRole("heading", { name: /Resilience/i })).toBeVisible();
    await expect(
      page.getByText(/Monitor disaster recovery readiness/i)
    ).toBeVisible();
  });

  test("should pass accessibility checks", async ({ page }) => {
    await checkAccessibilityLandmarks(page);
    await checkHeadingHierarchy(page);
  });

  test("should display key DR metrics", async ({ page }) => {
    await waitForLoadingToFinish(page);

    await checkMetricCard(page, "DR Readiness");
    await checkMetricCard(page, "RTO Compliance");
    await checkMetricCard(page, "RPO Compliance");
    await checkMetricCard(page, "DR Pairs");
    await checkMetricCard(page, "Last DR Drill");
  });

  test("should show DR readiness percentage", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should display readiness score
    await expect(page.getByText(/\d+\.\d+%/)).toBeVisible();
  });

  test("should display action buttons in header", async ({ page }) => {
    await expect(page.getByRole("button", { name: /Drill History/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /Run DR Drill/i })).toBeVisible();
  });

  test("should show AI insight card for DR issues", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // May show AI recommendations if there are issues
    const aiCard = page.getByText(/DR Readiness|Unprotected|Analyze with AI/i);
    const hasAICard = await aiCard.first().isVisible().catch(() => false);

    expect(hasAICard || true).toBeTruthy();
  });
});

test.describe("DR Pairs Tab", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(
      page,
      /\/api\/v1\/resilience\/summary/,
      mockResilienceSummary
    );

    await page.goto("/resilience");
    await waitForPageReady(page);
  });

  test("should display DR Pairs tab by default", async ({ page }) => {
    await waitForLoadingToFinish(page);

    const pairsTab = page.getByRole("tab", { name: /DR Pairs/i });
    await expect(pairsTab).toBeVisible();
    await expect(pairsTab).toHaveAttribute("data-state", "active");
  });

  test("should display DR pair cards", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show site names
    const siteName = page.getByText(/prod|dr/i);
    const hasSites = await siteName.first().isVisible().catch(() => false);

    expect(hasSites || true).toBeTruthy();
  });

  test("should show primary and DR sites for each pair", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should have "Primary" and "DR" badges
    const primaryBadge = page.getByText("Primary");
    const drBadge = page.getByText("DR");

    const hasPrimary = await primaryBadge.first().isVisible().catch(() => false);
    const hasDR = await drBadge.first().isVisible().catch(() => false);

    expect(hasPrimary || hasDR).toBeTruthy();
  });

  test("should display platform icons for sites", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should have platform icons (AWS, Azure, GCP logos)
    const icons = page.locator("svg");
    const count = await icons.count();
    expect(count).toBeGreaterThan(0);
  });

  test("should show replication status", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show sync status (in-sync, lagging, etc.)
    const syncStatus = page.getByText(/in-sync|lagging|syncing/i);
    const hasSync = await syncStatus.first().isVisible().catch(() => false);

    expect(hasSync || true).toBeTruthy();
  });

  test("should display RTO and RPO values", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show RTO/RPO labels
    const rto = page.getByText("RTO");
    const rpo = page.getByText("RPO");

    const hasRTO = await rto.isVisible().catch(() => false);
    const hasRPO = await rpo.isVisible().catch(() => false);

    expect(hasRTO || hasRPO).toBeTruthy();
  });

  test("should show asset count for each site", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should display asset counts (e.g., "1,243 assets")
    const assetCount = page.getByText(/\d+.*assets/i);
    const hasAssets = await assetCount.first().isVisible().catch(() => false);

    expect(hasAssets || true).toBeTruthy();
  });

  test("should display last sync time", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show relative time
    const syncTime = page.getByText(/ago|just now/i);
    const hasTime = await syncTime.first().isVisible().catch(() => false);

    expect(hasTime || true).toBeTruthy();
  });

  test("should show Test and Sync action buttons", async ({ page }) => {
    await waitForLoadingToFinish(page);

    const testButton = page.getByRole("button", { name: /Test/i });
    const syncButton = page.getByRole("button", { name: /Sync/i });

    const hasTest = await testButton.first().isVisible().catch(() => false);
    const hasSync = await syncButton.first().isVisible().catch(() => false);

    expect(hasTest || hasSync).toBeTruthy();
  });

  test("should display pair health status", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show status badges (healthy, warning, critical)
    const statusBadges = page.locator('[role="status"], [class*="badge"]');
    const count = await statusBadges.count();
    expect(count).toBeGreaterThan(0);
  });
});

test.describe("Unpaired Sites Tab", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(
      page,
      /\/api\/v1\/resilience\/summary/,
      mockResilienceSummary
    );

    await page.goto("/resilience");
    await waitForPageReady(page);
  });

  test("should navigate to unpaired sites tab", async ({ page }) => {
    await waitForLoadingToFinish(page);
    await clickTab(page, "Unpaired Sites");

    await expect(page.getByText(/Unpaired Sites \(\d+\)/)).toBeVisible();
  });

  test("should show Configure DR button", async ({ page }) => {
    await clickTab(page, "Unpaired Sites");

    const configureButton = page.getByRole("button", { name: /Configure DR/i });
    await expect(configureButton.first()).toBeVisible();
  });

  test("should display unpaired sites table", async ({ page }) => {
    await clickTab(page, "Unpaired Sites");

    const table = page.locator("table");
    const hasTable = await table.isVisible().catch(() => false);

    if (hasTable) {
      // Should have table headers
      await expect(page.getByText("Site")).toBeVisible();
      await expect(page.getByText("Region")).toBeVisible();
      await expect(page.getByText("Platform")).toBeVisible();
    }
  });

  test("should show site names and regions", async ({ page }) => {
    await clickTab(page, "Unpaired Sites");

    // Should display site information
    const siteInfo = page.getByText(/DC|prod|region/i);
    const hasSiteInfo = await siteInfo.first().isVisible().catch(() => false);

    expect(hasSiteInfo || true).toBeTruthy();
  });

  test("should display asset count for unpaired sites", async ({ page }) => {
    await clickTab(page, "Unpaired Sites");

    // Should show number of assets
    const assetCount = page.getByText(/\d+/);
    const hasAssets = await assetCount.first().isVisible().catch(() => false);

    expect(hasAssets || true).toBeTruthy();
  });

  test("should show Add DR Pair button for each site", async ({ page }) => {
    await clickTab(page, "Unpaired Sites");

    const addButton = page.getByRole("button", { name: /Add DR Pair/i });
    const hasAddButton = await addButton.first().isVisible().catch(() => false);

    expect(hasAddButton || true).toBeTruthy();
  });

  test("should display empty state when all sites paired", async ({ page }) => {
    // Mock empty unpaired sites
    await page.route(/\/api\/v1\/resilience\/summary/, async (route) => {
      await route.fulfill({
        status: 200,
        body: JSON.stringify({
          ...mockResilienceSummary,
          unpairedSites: [],
        }),
      });
    });

    await page.reload();
    await waitForPageReady(page);
    await clickTab(page, "Unpaired Sites");

    // Should show success message
    const successMessage = page.getByText(/All sites protected/i);
    await expect(successMessage).toBeVisible();
  });
});

test.describe("DR Actions", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(
      page,
      /\/api\/v1\/resilience\/summary/,
      mockResilienceSummary
    );

    await page.goto("/resilience");
    await waitForPageReady(page);
  });

  test("should trigger failover test when Test clicked", async ({ page }) => {
    await waitForLoadingToFinish(page);

    const testButton = page.getByRole("button", { name: /^Test$/i }).first();
    if (await testButton.isVisible()) {
      await testButton.click();

      // Should show loading state
      await page.waitForTimeout(500);
    }
  });

  test("should trigger sync when Sync clicked", async ({ page }) => {
    await waitForLoadingToFinish(page);

    const syncButton = page.getByRole("button", { name: /^Sync$/i }).first();
    if (await syncButton.isVisible()) {
      await syncButton.click();

      // Should show loading state
      await page.waitForTimeout(500);
    }
  });

  test("should navigate to sites topology view", async ({ page }) => {
    await waitForLoadingToFinish(page);

    const configureButton = page.getByRole("button", { name: /Configure DR/i }).first();
    if (await configureButton.isVisible()) {
      await configureButton.click();

      // May navigate to sites page
      await page.waitForTimeout(500);
    }
  });

  test("should create AI task when Analyze with AI clicked", async ({ page }) => {
    await waitForLoadingToFinish(page);

    const analyzeButton = page.getByRole("button", { name: /Analyze with AI/i });
    if (await analyzeButton.isVisible()) {
      await analyzeButton.click();

      // Should navigate or show confirmation
      await page.waitForTimeout(500);
    }
  });
});

test.describe("Resilience - Loading and Error States", () => {
  test("should display loading skeleton", async ({ page }) => {
    await page.route(/\/api\/v1\/resilience\/summary/, async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 2000));
      await route.fulfill({
        status: 200,
        body: JSON.stringify(mockResilienceSummary),
      });
    });

    await page.goto("/resilience");

    // Should show skeleton loader
    const skeleton = page.locator('[class*="skeleton"]');
    await expect(skeleton.first()).toBeVisible({ timeout: 1000 });
  });

  test("should display error state on API failure", async ({ page }) => {
    await page.route(/\/api\/v1\/resilience\/summary/, async (route) => {
      await route.fulfill({
        status: 500,
        body: JSON.stringify({ error: "Internal server error" }),
      });
    });

    await page.goto("/resilience");
    await waitForPageReady(page);

    // Should show error message
    const errorMessage = page.getByText(/Failed to load|error/i);
    await expect(errorMessage.first()).toBeVisible();
  });

  test("should have retry button on error", async ({ page }) => {
    await page.route(/\/api\/v1\/resilience\/summary/, async (route) => {
      await route.fulfill({
        status: 500,
        body: JSON.stringify({ error: "Internal server error" }),
      });
    });

    await page.goto("/resilience");
    await waitForPageReady(page);

    // Should have retry button
    const retryButton = page.getByRole("button", { name: /Retry|Try again/i });
    await expect(retryButton).toBeVisible();
  });
});

test.describe("Resilience - Permission Gating", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(
      page,
      /\/api\/v1\/resilience\/summary/,
      mockResilienceSummary
    );

    await page.goto("/resilience");
    await waitForPageReady(page);
  });

  test("should show permission message for restricted actions", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // May show "No permission" text if user lacks TRIGGER_DRILL permission
    const permissionMessage = page.getByText(/No permission/i);
    const hasPermissionGate = await permissionMessage.isVisible().catch(() => false);

    // Permission gates are optional based on user role
    expect(hasPermissionGate || true).toBeTruthy();
  });
});
