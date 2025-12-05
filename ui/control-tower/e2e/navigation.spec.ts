import { test, expect } from "@playwright/test";

test.describe("Navigation", () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the overview page (dev mode bypasses auth)
    await page.goto("/overview");
  });

  test("should display the sidebar navigation", async ({ page }) => {
    // Sidebar should be visible on desktop
    await expect(page.locator("nav")).toBeVisible();

    // Check main navigation items
    await expect(page.getByRole("link", { name: "Overview" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Risk" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Images" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Drift" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Sites" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Compliance" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Resilience" })).toBeVisible();
    await expect(page.getByRole("link", { name: "AI Copilot" })).toBeVisible();
  });

  test("should navigate to Images page", async ({ page }) => {
    await page.getByRole("link", { name: "Images" }).click();
    await expect(page).toHaveURL("/images");
    await expect(page.getByRole("heading", { name: "Golden Images" })).toBeVisible();
  });

  test("should navigate to Drift page", async ({ page }) => {
    await page.getByRole("link", { name: "Drift" }).click();
    await expect(page).toHaveURL("/drift");
    await expect(page.getByRole("heading", { name: "Drift Detection" })).toBeVisible();
  });

  test("should navigate to AI Copilot page", async ({ page }) => {
    await page.getByRole("link", { name: "AI Copilot" }).click();
    await expect(page).toHaveURL("/ai");
    await expect(page.getByText("AI Copilot")).toBeVisible();
  });

  test("should navigate to Resilience page", async ({ page }) => {
    await page.getByRole("link", { name: "Resilience" }).click();
    await expect(page).toHaveURL("/resilience");
    await expect(page.getByRole("heading", { name: "Resilience" })).toBeVisible();
  });

  test("should navigate to Compliance page", async ({ page }) => {
    await page.getByRole("link", { name: "Compliance" }).click();
    await expect(page).toHaveURL("/compliance");
    await expect(page.getByRole("heading", { name: "Compliance" })).toBeVisible();
  });
});

test.describe("Mobile Navigation", () => {
  test.use({ viewport: { width: 375, height: 667 } });

  test("should show mobile menu button on small screens", async ({ page }) => {
    await page.goto("/overview");

    // Mobile menu button should be visible
    const menuButton = page.getByRole("button", { name: "Open navigation menu" });
    await expect(menuButton).toBeVisible();

    // Click to open mobile menu
    await menuButton.click();

    // Navigation links should now be visible in mobile sheet
    await expect(page.getByRole("link", { name: "Images" })).toBeVisible();
  });
});
