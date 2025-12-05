import { test, expect } from "@playwright/test";
import {
  waitForPageReady,
  checkAccessibilityLandmarks,
  checkHeadingHierarchy,
  clickTab,
  checkMetricCard,
  mockAPIResponse,
  mockMultipleAPIs,
  waitForLoadingToFinish,
  checkEmptyState,
  checkCardCount,
} from "./fixtures/test-utils";
import {
  mockInSpecProfiles,
  mockInSpecScans,
  mockInSpecSchedules,
  mockInSpecScanResults,
} from "./fixtures/mock-data";

test.describe("InSpec Page", () => {
  test.beforeEach(async ({ page }) => {
    // Mock all required API responses
    await mockMultipleAPIs(page, [
      { endpoint: /\/api\/v1\/inspec\/profiles$/, response: { profiles: mockInSpecProfiles } },
      { endpoint: /\/api\/v1\/inspec\/runs$/, response: mockInSpecScans },
      { endpoint: /\/api\/v1\/inspec\/schedules$/, response: mockInSpecSchedules },
    ]);

    await page.goto("/inspec");
    await waitForPageReady(page);
  });

  test("should display page title and description", async ({ page }) => {
    await expect(
      page.getByRole("heading", { name: /InSpec Compliance Scanning/i })
    ).toBeVisible();
    await expect(
      page.getByText(/Automated compliance assessment/i)
    ).toBeVisible();
  });

  test("should pass accessibility checks", async ({ page }) => {
    await checkAccessibilityLandmarks(page);
    await checkHeadingHierarchy(page);
  });

  test("should display key metrics cards", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Check for metric cards
    await checkMetricCard(page, "Profiles Available");
    await checkMetricCard(page, "Last Scan Score");
    await checkMetricCard(page, "Controls Passed");
    await checkMetricCard(page, "Scans This Month");
  });

  test("should have tabs for profiles, scans, and schedules", async ({ page }) => {
    await waitForLoadingToFinish(page);

    await expect(page.getByRole("tab", { name: /Profiles/i })).toBeVisible();
    await expect(page.getByRole("tab", { name: /Recent Scans/i })).toBeVisible();
    await expect(page.getByRole("tab", { name: /Schedules/i })).toBeVisible();
  });

  test("should display Run Scan button", async ({ page }) => {
    await expect(
      page.getByRole("button", { name: /Run Scan/i })
    ).toBeVisible();
  });
});

test.describe("InSpec Profiles Tab", () => {
  test.beforeEach(async ({ page }) => {
    await mockMultipleAPIs(page, [
      { endpoint: /\/api\/v1\/inspec\/profiles$/, response: { profiles: mockInSpecProfiles } },
      { endpoint: /\/api\/v1\/inspec\/runs$/, response: mockInSpecScans },
      { endpoint: /\/api\/v1\/inspec\/schedules$/, response: mockInSpecSchedules },
    ]);

    await page.goto("/inspec");
    await waitForPageReady(page);
  });

  test("should display profiles tab by default", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Profiles tab should be active by default
    const profilesTab = page.getByRole("tab", { name: /Profiles/i });
    await expect(profilesTab).toHaveAttribute("data-state", "active");
  });

  test("should display profile cards", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show profile names from mock data
    await expect(page.getByText("Linux Security Baseline")).toBeVisible();
    await expect(page.getByText("AWS Foundations Benchmark")).toBeVisible();
    await expect(page.getByText("Docker CIS Benchmark")).toBeVisible();
  });

  test("should display profile frameworks", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show framework names
    await expect(page.getByText("CIS Benchmarks").first()).toBeVisible();
  });

  test("should display profile control counts", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show control counts
    await expect(page.getByText(/142/).first()).toBeVisible();
    await expect(page.getByText(/89/).first()).toBeVisible();
  });

  test("should have run buttons on profile cards", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should have run buttons on each profile card
    const runButtons = page.locator('[class*="card"]').getByRole("button", { name: /Run/i });
    const count = await runButtons.count();
    expect(count).toBeGreaterThan(0);
  });
});

test.describe("InSpec Recent Scans Tab", () => {
  test.beforeEach(async ({ page }) => {
    await mockMultipleAPIs(page, [
      { endpoint: /\/api\/v1\/inspec\/profiles$/, response: { profiles: mockInSpecProfiles } },
      { endpoint: /\/api\/v1\/inspec\/runs$/, response: mockInSpecScans },
      { endpoint: /\/api\/v1\/inspec\/schedules$/, response: mockInSpecSchedules },
    ]);

    await page.goto("/inspec");
    await waitForPageReady(page);
  });

  test("should switch to scans tab", async ({ page }) => {
    await waitForLoadingToFinish(page);

    await clickTab(page, "Recent Scans");

    const scansTab = page.getByRole("tab", { name: /Recent Scans/i });
    await expect(scansTab).toHaveAttribute("data-state", "active");
  });

  test("should display scans table", async ({ page }) => {
    await waitForLoadingToFinish(page);

    await clickTab(page, "Recent Scans");

    // Should show table headers
    await expect(page.getByText("Date")).toBeVisible();
    await expect(page.getByText("Profile")).toBeVisible();
    await expect(page.getByText("Status")).toBeVisible();
    await expect(page.getByText("Score")).toBeVisible();
  });

  test("should display scan profile names", async ({ page }) => {
    await waitForLoadingToFinish(page);

    await clickTab(page, "Recent Scans");

    // Should show profile names from mock scan data
    await expect(page.getByText("linux-baseline").first()).toBeVisible();
    await expect(page.getByText("aws-foundations").first()).toBeVisible();
  });

  test("should display scan status badges", async ({ page }) => {
    await waitForLoadingToFinish(page);

    await clickTab(page, "Recent Scans");

    // Should show scan statuses
    const statusBadges = page.locator('[class*="badge"], [role="status"]');
    const count = await statusBadges.count();
    expect(count).toBeGreaterThan(0);
  });

  test("should display scan scores", async ({ page }) => {
    await waitForLoadingToFinish(page);

    await clickTab(page, "Recent Scans");

    // Should show pass rates
    await expect(page.getByText(/94\.5%/).first()).toBeVisible();
    await expect(page.getByText(/87\.6%/).first()).toBeVisible();
  });

  test("should have View Results buttons", async ({ page }) => {
    await waitForLoadingToFinish(page);

    await clickTab(page, "Recent Scans");

    // Should have view results buttons
    const viewButtons = page.getByRole("button", { name: /View Results/i });
    const count = await viewButtons.count();
    expect(count).toBeGreaterThan(0);
  });
});

test.describe("InSpec Schedules Tab", () => {
  test.beforeEach(async ({ page }) => {
    await mockMultipleAPIs(page, [
      { endpoint: /\/api\/v1\/inspec\/profiles$/, response: { profiles: mockInSpecProfiles } },
      { endpoint: /\/api\/v1\/inspec\/runs$/, response: mockInSpecScans },
      { endpoint: /\/api\/v1\/inspec\/schedules$/, response: mockInSpecSchedules },
    ]);

    await page.goto("/inspec");
    await waitForPageReady(page);
  });

  test("should switch to schedules tab", async ({ page }) => {
    await waitForLoadingToFinish(page);

    await clickTab(page, "Schedules");

    const schedulesTab = page.getByRole("tab", { name: /Schedules/i });
    await expect(schedulesTab).toHaveAttribute("data-state", "active");
  });

  test("should display scheduled scans", async ({ page }) => {
    await waitForLoadingToFinish(page);

    await clickTab(page, "Schedules");

    // Should show schedule names from mock data
    await expect(page.getByText("linux-baseline").first()).toBeVisible();
    await expect(page.getByText("aws-foundations").first()).toBeVisible();
  });

  test("should display cron expressions", async ({ page }) => {
    await waitForLoadingToFinish(page);

    await clickTab(page, "Schedules");

    // Should show cron expressions
    await expect(page.getByText(/0 2 \* \* \*/).first()).toBeVisible();
  });

  test("should display enabled/disabled badges", async ({ page }) => {
    await waitForLoadingToFinish(page);

    await clickTab(page, "Schedules");

    // Should show enabled/disabled status
    await expect(page.getByText("Enabled").first()).toBeVisible();
    await expect(page.getByText("Disabled")).toBeVisible();
  });

  test("should have New Schedule button", async ({ page }) => {
    await waitForLoadingToFinish(page);

    await clickTab(page, "Schedules");

    await expect(
      page.getByRole("button", { name: /New Schedule/i })
    ).toBeVisible();
  });

  test("should have Configure buttons on schedules", async ({ page }) => {
    await waitForLoadingToFinish(page);

    await clickTab(page, "Schedules");

    // Should have configure buttons
    const configureButtons = page.getByRole("button", { name: /Configure/i });
    const count = await configureButtons.count();
    expect(count).toBeGreaterThan(0);
  });
});

test.describe("InSpec Page - Empty State", () => {
  test("should display empty state when no profiles", async ({ page }) => {
    await mockMultipleAPIs(page, [
      { endpoint: /\/api\/v1\/inspec\/profiles$/, response: { profiles: [] } },
      { endpoint: /\/api\/v1\/inspec\/runs$/, response: { runs: [] } },
      { endpoint: /\/api\/v1\/inspec\/schedules$/, response: [] },
    ]);

    await page.goto("/inspec");
    await waitForPageReady(page);
    await waitForLoadingToFinish(page);

    // Should show empty state message
    const emptyState = page.getByText(/No profiles available/i);
    await expect(emptyState).toBeVisible();
  });

  test("should display empty state in scans tab when no scans", async ({ page }) => {
    await mockMultipleAPIs(page, [
      { endpoint: /\/api\/v1\/inspec\/profiles$/, response: { profiles: mockInSpecProfiles } },
      { endpoint: /\/api\/v1\/inspec\/runs$/, response: { runs: [] } },
      { endpoint: /\/api\/v1\/inspec\/schedules$/, response: mockInSpecSchedules },
    ]);

    await page.goto("/inspec");
    await waitForPageReady(page);
    await waitForLoadingToFinish(page);

    await clickTab(page, "Recent Scans");

    // Should show empty state message
    const emptyState = page.getByText(/No scans yet/i);
    await expect(emptyState).toBeVisible();
  });

  test("should display empty state in schedules tab when no schedules", async ({ page }) => {
    await mockMultipleAPIs(page, [
      { endpoint: /\/api\/v1\/inspec\/profiles$/, response: { profiles: mockInSpecProfiles } },
      { endpoint: /\/api\/v1\/inspec\/runs$/, response: mockInSpecScans },
      { endpoint: /\/api\/v1\/inspec\/schedules$/, response: [] },
    ]);

    await page.goto("/inspec");
    await waitForPageReady(page);
    await waitForLoadingToFinish(page);

    await clickTab(page, "Schedules");

    // Should show empty state message
    const emptyState = page.getByText(/No schedules configured/i);
    await expect(emptyState).toBeVisible();
  });
});

test.describe("InSpec Page - Loading State", () => {
  test("should display loading skeletons", async ({ page }) => {
    // Mock slow API
    await page.route(/\/api\/v1\/inspec\/profiles$/, async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 2000));
      await route.fulfill({
        status: 200,
        body: JSON.stringify({ profiles: mockInSpecProfiles }),
      });
    });

    await page.goto("/inspec");

    // Should show loading state
    const loadingIndicator = page.locator(
      '[role="status"], [class*="skeleton"], [class*="loading"]'
    );
    await expect(loadingIndicator.first()).toBeVisible({ timeout: 1000 });
  });
});

test.describe("InSpec Page - Error State", () => {
  test("should display error state on API failure", async ({ page }) => {
    await mockAPIResponse(page, /\/api\/v1\/inspec\/profiles$/, { error: "Server error" }, 500);

    await page.goto("/inspec");
    await waitForPageReady(page);

    // Should show error state
    const errorState = page.getByText(/Failed to load|error|try again/i);
    const hasError = await errorState.first().isVisible().catch(() => false);

    expect(hasError).toBeTruthy();
  });
});

test.describe("InSpec Navigation", () => {
  test.beforeEach(async ({ page }) => {
    await mockMultipleAPIs(page, [
      { endpoint: /\/api\/v1\/inspec\/profiles$/, response: { profiles: mockInSpecProfiles } },
      { endpoint: /\/api\/v1\/inspec\/runs$/, response: mockInSpecScans },
      { endpoint: /\/api\/v1\/inspec\/schedules$/, response: mockInSpecSchedules },
    ]);

    await page.goto("/inspec");
    await waitForPageReady(page);
  });

  test("should navigate to profile detail page when clicking profile", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Click on first profile card
    const profileCard = page.locator('[class*="card"]').first();
    const isClickable = await profileCard.isVisible().catch(() => false);

    if (isClickable) {
      await profileCard.click();
      // Should navigate to profile detail page
      await expect(page).toHaveURL(/\/inspec\/profiles\/.+/);
    }
  });

  test("should navigate to scan detail page when clicking View Results", async ({ page }) => {
    await waitForLoadingToFinish(page);

    await clickTab(page, "Recent Scans");

    // Click on first View Results button
    const viewButton = page.getByRole("button", { name: /View Results/i }).first();
    const isClickable = await viewButton.isVisible().catch(() => false);

    if (isClickable) {
      await viewButton.click();
      // Should navigate to scan detail page
      await expect(page).toHaveURL(/\/inspec\/scans\/.+/);
    }
  });
});

test.describe("InSpec Last Scan Info Card", () => {
  test.beforeEach(async ({ page }) => {
    await mockMultipleAPIs(page, [
      { endpoint: /\/api\/v1\/inspec\/profiles$/, response: { profiles: mockInSpecProfiles } },
      { endpoint: /\/api\/v1\/inspec\/runs$/, response: mockInSpecScans },
      { endpoint: /\/api\/v1\/inspec\/schedules$/, response: mockInSpecSchedules },
    ]);

    await page.goto("/inspec");
    await waitForPageReady(page);
  });

  test("should display last scan info card", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show last scan info
    await expect(page.getByText(/Last Scan/i).first()).toBeVisible();
  });

  test("should have View Details button on last scan card", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should have view details button
    const viewDetailsButton = page.getByRole("button", { name: /View Details/i });
    await expect(viewDetailsButton).toBeVisible();
  });
});
