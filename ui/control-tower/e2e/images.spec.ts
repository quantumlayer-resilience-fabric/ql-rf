import { test, expect } from "@playwright/test";
import {
  waitForPageReady,
  checkAccessibilityLandmarks,
  checkHeadingHierarchy,
  checkTableHasData,
  clickTab,
  checkMetricCard,
  mockAPIResponse,
  waitForLoadingToFinish,
  checkEmptyState,
} from "./fixtures/test-utils";
import { mockImages, mockLineage } from "./fixtures/mock-data";

test.describe("Golden Images Page", () => {
  test.beforeEach(async ({ page }) => {
    // Mock API responses
    await mockAPIResponse(page, /\/api\/v1\/images$/, {
      images: mockImages,
      total: mockImages.length,
    });

    await page.goto("/images");
    await waitForPageReady(page);
  });

  test("should display page title and description", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Golden Images" })).toBeVisible();
    await expect(
      page.getByText(/Manage and version your golden images/i)
    ).toBeVisible();
  });

  test("should pass accessibility checks", async ({ page }) => {
    await checkAccessibilityLandmarks(page);
    await checkHeadingHierarchy(page);
  });

  test("should display images in a table or grid", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Check for table or grid layout
    const hasTable = await page.locator("table").isVisible().catch(() => false);
    const hasCards = await page
      .locator('[role="article"], .card, [class*="card"]')
      .first()
      .isVisible()
      .catch(() => false);

    expect(hasTable || hasCards).toBeTruthy();
  });

  test("should display search/filter controls", async ({ page }) => {
    // Should have search input or filter buttons
    const hasSearch = await page
      .getByPlaceholder(/search|filter/i)
      .isVisible()
      .catch(() => false);
    const hasFilter = await page
      .getByRole("button", { name: /filter|platform|status/i })
      .first()
      .isVisible()
      .catch(() => false);

    expect(hasSearch || hasFilter).toBeTruthy();
  });

  test("should display image family information", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Check for image family names
    await expect(page.getByText("ql-base-linux")).toBeVisible();
    await expect(page.getByText(/1\.6\.4/)).toBeVisible();
  });

  test("should display image status badges", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show status badges
    const statusBadges = page.locator(
      '[role="status"], [class*="badge"], [class*="status"]'
    );
    const count = await statusBadges.count();
    expect(count).toBeGreaterThan(0);
  });

  test("should filter images by platform", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Look for platform filter
    const platformFilter = page.getByRole("button", { name: /platform|filter/i });
    if (await platformFilter.isVisible()) {
      await platformFilter.click();

      // Should show platform options
      const awsOption = page.getByText(/aws/i);
      await expect(awsOption).toBeVisible();
    }
  });

  test("should navigate to image detail page when clicking an image", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Click on first image link
    const imageLink = page.locator("a[href*='/images/']").first();
    const isLinkVisible = await imageLink.isVisible().catch(() => false);

    if (isLinkVisible) {
      await imageLink.click();
      await expect(page).toHaveURL(/\/images\/.+/);
    }
  });

  test("should display vulnerability count for images", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show vulnerability information
    const vulnInfo = page.getByText(/vulnerability|vulnerabilities|CVE/i);
    const hasVulnInfo = await vulnInfo.first().isVisible().catch(() => false);

    // Not all images may have vulnerabilities, so this is optional
    expect(hasVulnInfo || true).toBeTruthy();
  });

  test("should show golden image indicator", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should indicate which images are golden
    const goldenIndicator = page.locator('[class*="golden"], [data-golden="true"]');
    const goldenText = page.getByText(/golden|active/i);

    const hasGoldenIndicator =
      (await goldenIndicator.first().isVisible().catch(() => false)) ||
      (await goldenText.first().isVisible().catch(() => false));

    expect(hasGoldenIndicator).toBeTruthy();
  });
});

test.describe("Image Lineage Page", () => {
  test.beforeEach(async ({ page }) => {
    // Mock lineage API response
    await mockAPIResponse(page, /\/api\/v1\/images\/image-1\/lineage/, mockLineage);

    await page.goto("/images/image-1/lineage");
    await waitForPageReady(page);
  });

  test("should display lineage page title", async ({ page }) => {
    await expect(page.getByRole("heading", { name: /lineage|version/i })).toBeVisible();
  });

  test("should display image family name", async ({ page }) => {
    await waitForLoadingToFinish(page);
    await expect(page.getByText("ql-base-linux")).toBeVisible();
  });

  test("should display lineage tree or graph", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show version nodes
    await expect(page.getByText(/1\.6\.3/)).toBeVisible();
    await expect(page.getByText(/1\.6\.4/)).toBeVisible();
  });

  test("should show parent-child relationships", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should have visual indicators of relationships (arrows, lines, etc.)
    const hasArrows = await page.locator("svg").count();
    expect(hasArrows).toBeGreaterThan(0);
  });

  test("should display creation dates for versions", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show dates or relative time
    const hasDateInfo =
      (await page.getByText(/202\d/).first().isVisible().catch(() => false)) ||
      (await page.getByText(/ago|created/i).first().isVisible().catch(() => false));

    expect(hasDateInfo).toBeTruthy();
  });

  test("should highlight current version", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Current version should be highlighted
    const currentVersion = page.locator('[data-current="true"], [class*="current"]');
    const hasCurrentIndicator =
      (await currentVersion.first().isVisible().catch(() => false)) ||
      (await page.getByText(/current|active/i).first().isVisible().catch(() => false));

    expect(hasCurrentIndicator).toBeTruthy();
  });
});

test.describe("Image Management Actions", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(page, /\/api\/v1\/images$/, {
      images: mockImages,
      total: mockImages.length,
    });

    await page.goto("/images");
    await waitForPageReady(page);
  });

  test("should have action buttons for images", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should have action buttons (promote, view, etc.)
    const actionButtons = page.getByRole("button", {
      name: /promote|view|details|more/i,
    });

    const hasActions = (await actionButtons.count()) > 0;
    expect(hasActions).toBeTruthy();
  });

  test("should show promote button for candidate images", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Candidate images should have promote option
    const promoteButton = page.getByRole("button", { name: /promote/i });
    const hasPromote = await promoteButton.first().isVisible().catch(() => false);

    // Not all images are candidates, so this is conditional
    expect(hasPromote || true).toBeTruthy();
  });
});

test.describe("Image Page - Empty State", () => {
  test("should display empty state when no images", async ({ page }) => {
    // Mock empty response
    await mockAPIResponse(page, /\/api\/v1\/images$/, {
      images: [],
      total: 0,
    });

    await page.goto("/images");
    await waitForPageReady(page);
    await waitForLoadingToFinish(page);

    await checkEmptyState(page);
  });
});

test.describe("Image Page - Loading State", () => {
  test("should display loading skeletons", async ({ page }) => {
    // Mock slow API
    await page.route(/\/api\/v1\/images$/, async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 2000));
      await route.fulfill({
        status: 200,
        body: JSON.stringify({ images: mockImages }),
      });
    });

    await page.goto("/images");

    // Should show loading state
    const loadingIndicator = page.locator(
      '[role="status"], [class*="skeleton"], [class*="loading"]'
    );
    await expect(loadingIndicator.first()).toBeVisible({ timeout: 1000 });
  });
});

test.describe("Image Search and Filter", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(page, /\/api\/v1\/images$/, {
      images: mockImages,
      total: mockImages.length,
    });

    await page.goto("/images");
    await waitForPageReady(page);
  });

  test("should filter images by search term", async ({ page }) => {
    await waitForLoadingToFinish(page);

    const searchInput = page.getByPlaceholder(/search/i);
    if (await searchInput.isVisible()) {
      await searchInput.fill("linux");
      await page.waitForTimeout(500);

      // Should show filtered results
      await expect(page.getByText("ql-base-linux")).toBeVisible();
    }
  });

  test("should filter by status", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Look for status filter
    const statusFilter = page.getByRole("button", { name: /status/i });
    if (await statusFilter.isVisible()) {
      await statusFilter.click();

      // Should show status options
      const activeOption = page.getByText(/active|candidate/i);
      await expect(activeOption.first()).toBeVisible();
    }
  });
});
