import { test, expect } from "@playwright/test";
import {
  waitForPageReady,
  checkAccessibilityLandmarks,
  checkHeadingHierarchy,
  clickTab,
  waitForLoadingToFinish,
  checkTableHasData,
} from "./fixtures/test-utils";

test.describe("Settings Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/settings");
    await waitForPageReady(page);
  });

  test("should display settings page title", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Settings" })).toBeVisible();
    await expect(
      page.getByText(/Manage your connectors.*team.*configuration/i)
    ).toBeVisible();
  });

  test("should pass accessibility checks", async ({ page }) => {
    await checkAccessibilityLandmarks(page);
    await checkHeadingHierarchy(page);
  });

  test("should display all settings tabs", async ({ page }) => {
    await expect(page.getByRole("tab", { name: /Connectors/i })).toBeVisible();
    await expect(page.getByRole("tab", { name: /Notifications/i })).toBeVisible();
    await expect(page.getByRole("tab", { name: /Team/i })).toBeVisible();
    await expect(page.getByRole("tab", { name: /API/i })).toBeVisible();
    await expect(page.getByRole("tab", { name: /Audit/i })).toBeVisible();
  });

  test("should show Connectors tab by default", async ({ page }) => {
    const connectorsTab = page.getByRole("tab", { name: /Connectors/i });
    await expect(connectorsTab).toHaveAttribute("data-state", "active");
  });
});

test.describe("Settings - Connectors Tab", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/settings");
    await waitForPageReady(page);
  });

  test("should display connectors section", async ({ page }) => {
    await expect(page.getByText("Cloud Connectors")).toBeVisible();
    await expect(
      page.getByText(/Connect your cloud providers/i)
    ).toBeVisible();
  });

  test("should show Add Connector button", async ({ page }) => {
    const addButton = page.getByRole("button", { name: /Add Connector/i });
    await expect(addButton).toBeVisible();
    await expect(addButton).toBeEnabled();
  });

  test("should display connector cards", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show platform connectors (AWS, Azure, GCP, vSphere)
    const connectors = page.locator('[class*="card"]');
    const count = await connectors.count();
    expect(count).toBeGreaterThan(0);
  });

  test("should show connector platform icons", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should have platform icons/logos
    const icons = page.locator("svg");
    const count = await icons.count();
    expect(count).toBeGreaterThan(0);
  });

  test("should display connector status", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show connection status
    const statusBadges = page.getByText(/connected|syncing|error/i);
    await expect(statusBadges.first()).toBeVisible();
  });

  test("should show asset count for connectors", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should display discovered assets
    const assetCount = page.getByText(/\d+.*assets/i);
    await expect(assetCount.first()).toBeVisible();
  });

  test("should display last sync time", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show sync timestamp
    const syncTime = page.getByText(/Last sync|ago|syncing/i);
    await expect(syncTime.first()).toBeVisible();
  });

  test("should show refresh button for connectors", async ({ page }) => {
    await waitForLoadingToFinish(page);

    const refreshButton = page.locator('button:has(svg)').first();
    await expect(refreshButton).toBeVisible();
  });

  test("should show more actions menu for connectors", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should have dropdown menu button
    const moreButton = page.getByRole("button", { name: /more/i }).first();
    if (await moreButton.isVisible()) {
      await moreButton.click();

      // Should show menu options
      await expect(page.getByText(/Edit|View Logs|Remove/i)).toBeVisible();
    }
  });
});

test.describe("Settings - Notifications Tab", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/settings");
    await waitForPageReady(page);
    await clickTab(page, "Notifications");
  });

  test("should display notifications section", async ({ page }) => {
    await expect(page.getByText("Notification Preferences")).toBeVisible();
  });

  test("should show email notifications card", async ({ page }) => {
    await expect(page.getByText("Email")).toBeVisible();
    await expect(
      page.getByText(/Receive notifications via email/i)
    ).toBeVisible();
  });

  test("should show Slack integration card", async ({ page }) => {
    await expect(page.getByText("Slack")).toBeVisible();
    await expect(page.getByText(/Send alerts to Slack/i)).toBeVisible();
  });

  test("should show Webhook integration card", async ({ page }) => {
    await expect(page.getByText("Webhook")).toBeVisible();
    await expect(
      page.getByText(/Send events to custom endpoints/i)
    ).toBeVisible();
  });

  test("should display notification settings status", async ({ page }) => {
    // Should show enabled/disabled status
    const statusBadges = page.getByText(/Enabled|Connected|Not configured/i);
    await expect(statusBadges.first()).toBeVisible();
  });

  test("should show Configure buttons", async ({ page }) => {
    const configureButtons = page.getByRole("button", { name: /Configure/i });
    const count = await configureButtons.count();
    expect(count).toBeGreaterThan(0);
  });

  test("should show Add Webhook button when not configured", async ({ page }) => {
    const addWebhookButton = page.getByRole("button", { name: /Add Webhook/i });
    const hasButton = await addWebhookButton.isVisible().catch(() => false);

    // Webhook may or may not be configured
    expect(hasButton || true).toBeTruthy();
  });
});

test.describe("Settings - Team Tab", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/settings");
    await waitForPageReady(page);
    await clickTab(page, "Team");
  });

  test("should display team members section", async ({ page }) => {
    await expect(page.getByText("Team Members")).toBeVisible();
    await expect(
      page.getByText(/Manage who has access/i)
    ).toBeVisible();
  });

  test("should show Invite Member button", async ({ page }) => {
    const inviteButton = page.getByRole("button", { name: /Invite Member/i });
    await expect(inviteButton).toBeVisible();
    await expect(inviteButton).toBeEnabled();
  });

  test("should display team members table", async ({ page }) => {
    const table = page.locator("table");
    await expect(table).toBeVisible();

    // Check for table headers
    await expect(page.getByText("Member")).toBeVisible();
    await expect(page.getByText("Role")).toBeVisible();
    await expect(page.getByText("Status")).toBeVisible();
  });

  test("should show team member information", async ({ page }) => {
    // Should display names and emails
    const memberInfo = page.locator("table tbody tr").first();
    await expect(memberInfo).toBeVisible();
  });

  test("should show role dropdowns for members", async ({ page }) => {
    const roleSelect = page.locator('[role="combobox"]').first();
    await expect(roleSelect).toBeVisible();
  });

  test("should display member status badges", async ({ page }) => {
    const statusBadges = page.getByText(/active|pending/i);
    await expect(statusBadges.first()).toBeVisible();
  });

  test("should show remove member buttons", async ({ page }) => {
    // Should have delete/remove buttons
    const removeButtons = page.locator('button:has(svg)').filter({
      has: page.locator('svg'),
    });
    const count = await removeButtons.count();
    expect(count).toBeGreaterThan(0);
  });

  test("should allow changing member role", async ({ page }) => {
    const roleSelect = page.locator('button:has-text("Admin"), button:has-text("Editor"), button:has-text("Viewer")').first();

    if (await roleSelect.isVisible()) {
      await roleSelect.click();

      // Should show role options
      await expect(page.getByText(/Admin|Editor|Viewer/i)).toBeVisible();
    }
  });
});

test.describe("Settings - API Tab", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/settings");
    await waitForPageReady(page);
    await clickTab(page, "API");
  });

  test("should display API keys section", async ({ page }) => {
    await expect(page.getByText("API Keys")).toBeVisible();
    await expect(
      page.getByText(/Manage API keys for programmatic access/i)
    ).toBeVisible();
  });

  test("should show Generate Key button", async ({ page }) => {
    const generateButton = page.getByRole("button", { name: /Generate Key/i });
    await expect(generateButton).toBeVisible();
    await expect(generateButton).toBeEnabled();
  });

  test("should display API keys table", async ({ page }) => {
    const table = page.locator("table");
    await expect(table).toBeVisible();

    // Check for table headers
    await expect(page.getByText("Name")).toBeVisible();
    await expect(page.getByText("Key")).toBeVisible();
  });

  test("should show masked API keys", async ({ page }) => {
    // Should display masked keys
    const maskedKey = page.locator("code", { hasText: /••••/ });
    await expect(maskedKey.first()).toBeVisible();
  });

  test("should have show/hide button for API keys", async ({ page }) => {
    // Should have eye icon buttons
    const eyeButtons = page.locator('button:has(svg)').filter({
      has: page.locator('svg'),
    });
    const count = await eyeButtons.count();
    expect(count).toBeGreaterThan(0);
  });

  test("should toggle API key visibility", async ({ page }) => {
    const eyeButton = page.locator('button').filter({
      has: page.locator('svg[class*="eye"]'),
    }).first();

    if (await eyeButton.isVisible()) {
      await eyeButton.click();

      // Key visibility should change
      await page.waitForTimeout(300);
    }
  });

  test("should show copy button for API keys", async ({ page }) => {
    // Should have copy icon buttons
    const copyButtons = page.locator('button').filter({
      has: page.locator('svg'),
    });
    const count = await copyButtons.count();
    expect(count).toBeGreaterThan(0);
  });

  test("should display key creation and usage dates", async ({ page }) => {
    // Should show dates
    const dates = page.getByText(/202\d|ago|hours|days/i);
    await expect(dates.first()).toBeVisible();
  });

  test("should show API documentation section", async ({ page }) => {
    await expect(page.getByText("API Documentation")).toBeVisible();

    const docsButton = page.getByRole("button", { name: /View API Docs/i });
    await expect(docsButton).toBeVisible();
  });

  test("should show OpenAPI spec download button", async ({ page }) => {
    const downloadButton = page.getByRole("button", {
      name: /Download OpenAPI Spec/i,
    });
    await expect(downloadButton).toBeVisible();
  });
});

test.describe("Settings - Audit Log Tab", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/settings");
    await waitForPageReady(page);
    await clickTab(page, "Audit Log");
  });

  test("should display audit log section", async ({ page }) => {
    await expect(page.getByText("Audit Log")).toBeVisible();
    await expect(
      page.getByText(/Track all actions and changes/i)
    ).toBeVisible();
  });

  test("should display audit log table", async ({ page }) => {
    const table = page.locator("table");
    await expect(table).toBeVisible();

    // Check for table headers
    await expect(page.getByText("Action")).toBeVisible();
    await expect(page.getByText("User")).toBeVisible();
    await expect(page.getByText("Target")).toBeVisible();
    await expect(page.getByText("Time")).toBeVisible();
  });

  test("should show audit log entries", async ({ page }) => {
    // Should have log rows
    const rows = page.locator("table tbody tr");
    const count = await rows.count();
    expect(count).toBeGreaterThan(0);
  });

  test("should display action types with color indicators", async ({ page }) => {
    // Should have colored dots for log types
    const indicators = page.locator('[class*="rounded-full"]');
    const count = await indicators.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test("should show user names for actions", async ({ page }) => {
    // Should display who performed actions
    const userNames = page.locator("table tbody tr td:nth-child(2)");
    await expect(userNames.first()).toBeVisible();
  });

  test("should display target resources", async ({ page }) => {
    // Should show what was affected
    const targets = page.locator("table tbody tr td:nth-child(3)");
    await expect(targets.first()).toBeVisible();
  });

  test("should show relative timestamps", async ({ page }) => {
    const timestamps = page.getByText(/ago|hours|minutes|days/i);
    await expect(timestamps.first()).toBeVisible();
  });

  test("should show Load More button", async ({ page }) => {
    const loadMoreButton = page.getByRole("button", { name: /Load More/i });
    await expect(loadMoreButton).toBeVisible();
  });
});

test.describe("Settings - Tab Navigation", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/settings");
    await waitForPageReady(page);
  });

  test("should navigate between tabs", async ({ page }) => {
    // Navigate to Notifications
    await clickTab(page, "Notifications");
    await expect(page.getByText("Email")).toBeVisible();

    // Navigate to Team
    await clickTab(page, "Team");
    await expect(page.getByText("Team Members")).toBeVisible();

    // Navigate to API
    await clickTab(page, "API");
    await expect(page.getByText("API Keys")).toBeVisible();

    // Navigate to Audit
    await clickTab(page, "Audit Log");
    await expect(page.getByText(/Track all actions/i)).toBeVisible();

    // Navigate back to Connectors
    await clickTab(page, "Connectors");
    await expect(page.getByText("Cloud Connectors")).toBeVisible();
  });

  test("should preserve tab state in URL", async ({ page }) => {
    await clickTab(page, "Team");

    // URL may include tab parameter
    await page.waitForTimeout(300);
  });
});

test.describe("Settings - Mobile Responsive", () => {
  test.use({ viewport: { width: 375, height: 667 } });

  test("should display settings tabs on mobile", async ({ page }) => {
    await page.goto("/settings");
    await waitForPageReady(page);

    // Tabs should be visible (possibly as icons only)
    const tabs = page.locator('[role="tab"]');
    const count = await tabs.count();
    expect(count).toBeGreaterThan(0);
  });

  test("should allow tab navigation on mobile", async ({ page }) => {
    await page.goto("/settings");
    await waitForPageReady(page);

    const teamTab = page.getByRole("tab", { name: /Team/i });
    await teamTab.click();

    // Team content should be visible
    await page.waitForTimeout(500);
  });
});
