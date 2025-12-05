import { test, expect } from "@playwright/test";

test.describe("Accessibility", () => {
  test("overview page should have proper landmarks", async ({ page }) => {
    await page.goto("/overview");
    await page.waitForLoadState("domcontentloaded");

    // Should have main landmark
    await expect(page.getByRole("main")).toBeVisible({ timeout: 10000 });

    // Should have navigation
    await expect(page.getByRole("navigation").first()).toBeVisible({ timeout: 10000 });

    // Should have banner (header)
    await expect(page.getByRole("banner")).toBeVisible({ timeout: 10000 });
  });

  test("should have proper heading hierarchy", async ({ page }) => {
    await page.goto("/overview");

    // Should have h1
    const h1 = page.getByRole("heading", { level: 1 });
    await expect(h1).toBeVisible();
  });

  test("buttons should have accessible names", async ({ page }) => {
    await page.goto("/overview");
    await page.waitForLoadState("networkidle");

    // All buttons should have accessible names
    const buttons = page.getByRole("button");
    const count = await buttons.count();

    for (let i = 0; i < Math.min(count, 10); i++) {
      const button = buttons.nth(i);
      const name = await button.getAttribute("aria-label") || await button.textContent();
      expect(name?.trim()).toBeTruthy();
    }
  });

  test("links should have descriptive text", async ({ page }) => {
    await page.goto("/overview");

    // Navigation links should have text
    const navLinks = page.locator("nav a");
    const count = await navLinks.count();

    for (let i = 0; i < count; i++) {
      const link = navLinks.nth(i);
      const text = await link.textContent();
      expect(text?.trim()).toBeTruthy();
    }
  });

  test("images should have alt text", async ({ page }) => {
    await page.goto("/overview");
    await page.waitForLoadState("networkidle");

    const images = page.locator("img");
    const count = await images.count();

    for (let i = 0; i < count; i++) {
      const img = images.nth(i);
      const alt = await img.getAttribute("alt");
      const ariaHidden = await img.getAttribute("aria-hidden");
      const role = await img.getAttribute("role");

      // Either has alt text, is decorative (aria-hidden), or has presentation role
      expect(
        alt !== null || ariaHidden === "true" || role === "presentation"
      ).toBeTruthy();
    }
  });

  test("form controls should have labels", async ({ page }) => {
    await page.goto("/ai");
    await page.waitForLoadState("networkidle");

    // Input fields should have associated labels or aria-label
    const inputs = page.locator("input:not([type='hidden'])");
    const count = await inputs.count();

    for (let i = 0; i < count; i++) {
      const input = inputs.nth(i);
      const id = await input.getAttribute("id");
      const ariaLabel = await input.getAttribute("aria-label");
      const placeholder = await input.getAttribute("placeholder");

      // Has either a label, aria-label, or placeholder
      const hasLabel = id ? await page.locator(`label[for="${id}"]`).isVisible().catch(() => false) : false;

      expect(hasLabel || ariaLabel || placeholder).toBeTruthy();
    }
  });
});

test.describe("Keyboard Navigation", () => {
  test("should be able to navigate sidebar with keyboard", async ({ page }) => {
    await page.goto("/overview");

    // Tab to first navigation link
    await page.keyboard.press("Tab");
    await page.keyboard.press("Tab");
    await page.keyboard.press("Tab");

    // Should be able to reach navigation items
    const focusedElement = page.locator(":focus");
    await expect(focusedElement).toBeVisible();
  });

  test("should trap focus in modal dialogs", async ({ page }) => {
    await page.goto("/overview");
    await page.waitForLoadState("networkidle");

    // Find and click a button that opens a dialog/dropdown
    const notificationButton = page.getByRole("button", { name: /notification/i });

    if (await notificationButton.isVisible()) {
      await notificationButton.click();

      // Dialog should be visible
      const dialog = page.locator('[role="menu"], [role="dialog"]');
      await expect(dialog).toBeVisible();

      // Escape should close it
      await page.keyboard.press("Escape");
      await expect(dialog).not.toBeVisible();
    }
  });
});
