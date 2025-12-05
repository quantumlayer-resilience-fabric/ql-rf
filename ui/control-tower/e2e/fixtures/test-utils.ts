/**
 * Utility functions for E2E tests
 * Common test helpers, assertions, and setup functions
 */

import { Page, expect } from "@playwright/test";

/**
 * Wait for page to be fully loaded and ready
 */
export async function waitForPageReady(page: Page) {
  await page.waitForLoadState("domcontentloaded");
  await page.waitForLoadState("networkidle", { timeout: 30000 });
}

/**
 * Check if an element is visible with optional timeout
 */
export async function isVisible(page: Page, selector: string, timeout = 5000): Promise<boolean> {
  try {
    await page.waitForSelector(selector, { state: "visible", timeout });
    return true;
  } catch {
    return false;
  }
}

/**
 * Mock API response for a given endpoint
 */
export async function mockAPIResponse(
  page: Page,
  endpoint: string | RegExp,
  response: unknown,
  status = 200
) {
  await page.route(endpoint, async (route) => {
    await route.fulfill({
      status,
      contentType: "application/json",
      body: JSON.stringify(response),
    });
  });
}

/**
 * Mock multiple API responses at once
 */
export async function mockMultipleAPIs(
  page: Page,
  mocks: Array<{ endpoint: string | RegExp; response: unknown; status?: number }>
) {
  for (const mock of mocks) {
    await mockAPIResponse(page, mock.endpoint, mock.response, mock.status);
  }
}

/**
 * Take a screenshot with a descriptive name
 */
export async function takeScreenshot(page: Page, name: string) {
  await page.screenshot({
    path: `test-results/screenshots/${name}-${Date.now()}.png`,
    fullPage: true,
  });
}

/**
 * Check for console errors (excluding known acceptable errors)
 */
export function setupConsoleErrorTracking(page: Page) {
  const errors: string[] = [];

  page.on("console", (msg) => {
    if (msg.type() === "error") {
      const text = msg.text();
      // Filter out expected errors (e.g., React dev mode warnings)
      if (
        !text.includes("Download the React DevTools") &&
        !text.includes("Failed to load resource")
      ) {
        errors.push(text);
      }
    }
  });

  return {
    getErrors: () => errors,
    hasErrors: () => errors.length > 0,
  };
}

/**
 * Check accessibility landmarks are present
 */
export async function checkAccessibilityLandmarks(page: Page) {
  await expect(page.getByRole("main")).toBeVisible();
  await expect(page.getByRole("navigation").first()).toBeVisible();
}

/**
 * Check page has proper heading hierarchy
 */
export async function checkHeadingHierarchy(page: Page) {
  const h1 = page.getByRole("heading", { level: 1 });
  await expect(h1).toBeVisible();
  const h1Count = await h1.count();
  expect(h1Count).toBe(1); // Should have exactly one h1
}

/**
 * Fill a form with multiple fields
 */
export async function fillForm(
  page: Page,
  fields: Array<{ label: string; value: string }>
) {
  for (const field of fields) {
    const input = page.getByLabel(field.label);
    await input.fill(field.value);
  }
}

/**
 * Wait for an element to appear and then click it
 */
export async function waitAndClick(page: Page, selector: string) {
  await page.waitForSelector(selector, { state: "visible" });
  await page.click(selector);
}

/**
 * Scroll element into view
 */
export async function scrollIntoView(page: Page, selector: string) {
  await page.locator(selector).scrollIntoViewIfNeeded();
}

/**
 * Check if table has data
 */
export async function checkTableHasData(page: Page, minRows = 1) {
  const table = page.locator("table");
  await expect(table).toBeVisible();

  const rows = table.locator("tbody tr");
  const count = await rows.count();
  expect(count).toBeGreaterThanOrEqual(minRows);

  return count;
}

/**
 * Check if a specific tab is active
 */
export async function checkTabActive(page: Page, tabName: string) {
  const tab = page.getByRole("tab", { name: tabName });
  await expect(tab).toHaveAttribute("data-state", "active");
}

/**
 * Click tab and wait for content to load
 */
export async function clickTab(page: Page, tabName: string) {
  const tab = page.getByRole("tab", { name: tabName });
  await tab.click();
  await checkTabActive(page, tabName);
  await page.waitForTimeout(500); // Brief wait for content transition
}

/**
 * Check metric card displays correct value
 */
export async function checkMetricCard(
  page: Page,
  title: string,
  options?: { hasValue?: boolean; hasIcon?: boolean }
) {
  const card = page.locator(`[role="article"], .card, [class*="metric"]`, {
    has: page.getByText(title),
  }).first();

  await expect(card).toBeVisible();

  if (options?.hasValue !== false) {
    // Should have some numeric or percentage value
    await expect(card.locator("text=/\\d+|%/")).toBeVisible();
  }

  if (options?.hasIcon !== false) {
    // Should have an icon (svg)
    await expect(card.locator("svg").first()).toBeVisible();
  }
}

/**
 * Check status badge has correct variant
 */
export async function checkStatusBadge(
  page: Page,
  text: string,
  expectedStatus?: "success" | "warning" | "critical" | "neutral"
) {
  const badge = page.getByText(text, { exact: false }).locator("..").filter({
    has: page.locator('[class*="badge"], [role="status"]'),
  }).first();

  await expect(badge).toBeVisible();

  if (expectedStatus) {
    // Check if badge has appropriate color class
    const classList = await badge.getAttribute("class");
    expect(classList).toBeTruthy();
  }
}

/**
 * Wait for loading state to finish
 */
export async function waitForLoadingToFinish(page: Page) {
  // Wait for any loading indicators to disappear
  const loadingIndicators = page.locator(
    '[role="status"], [aria-busy="true"], [class*="loading"], [class*="skeleton"]'
  );

  await page.waitForTimeout(500);

  try {
    await loadingIndicators.first().waitFor({ state: "hidden", timeout: 10000 });
  } catch {
    // No loading indicators or already hidden
  }
}

/**
 * Check for error state
 */
export async function checkForError(page: Page): Promise<boolean> {
  const errorSelectors = [
    'text=/error/i',
    'text=/failed/i',
    '[role="alert"]',
    '[class*="error"]',
  ];

  for (const selector of errorSelectors) {
    if (await isVisible(page, selector, 1000)) {
      return true;
    }
  }

  return false;
}

/**
 * Check empty state is displayed
 */
export async function checkEmptyState(page: Page, expectedMessage?: string) {
  const emptyState = page.locator('[class*="empty"], [data-testid="empty-state"]');
  await expect(emptyState).toBeVisible();

  if (expectedMessage) {
    await expect(page.getByText(expectedMessage)).toBeVisible();
  }
}

/**
 * Hover over element
 */
export async function hoverElement(page: Page, selector: string) {
  await page.locator(selector).hover();
  await page.waitForTimeout(300); // Wait for hover effects
}

/**
 * Check tooltip appears on hover
 */
export async function checkTooltip(page: Page, triggerSelector: string, expectedText?: string) {
  await hoverElement(page, triggerSelector);

  const tooltip = page.locator('[role="tooltip"]');
  await expect(tooltip).toBeVisible();

  if (expectedText) {
    await expect(tooltip).toContainText(expectedText);
  }
}

/**
 * Press keyboard key
 */
export async function pressKey(page: Page, key: string) {
  await page.keyboard.press(key);
  await page.waitForTimeout(200);
}

/**
 * Check button is disabled
 */
export async function checkButtonDisabled(page: Page, buttonText: string) {
  const button = page.getByRole("button", { name: buttonText });
  await expect(button).toBeDisabled();
}

/**
 * Check button is enabled
 */
export async function checkButtonEnabled(page: Page, buttonText: string) {
  const button = page.getByRole("button", { name: buttonText });
  await expect(button).toBeEnabled();
}

/**
 * Wait for URL to match pattern
 */
export async function waitForURL(page: Page, urlPattern: string | RegExp) {
  await page.waitForURL(urlPattern, { timeout: 10000 });
}

/**
 * Check card count on page
 */
export async function checkCardCount(page: Page, minCount = 1) {
  const cards = page.locator('[role="article"], .card, [class*="card"]');
  const count = await cards.count();
  expect(count).toBeGreaterThanOrEqual(minCount);
  return count;
}

/**
 * Simulate API delay for testing loading states
 */
export async function mockAPIWithDelay(
  page: Page,
  endpoint: string | RegExp,
  response: unknown,
  delayMs = 2000
) {
  await page.route(endpoint, async (route) => {
    await new Promise((resolve) => setTimeout(resolve, delayMs));
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(response),
    });
  });
}
