import { test, expect } from "@playwright/test";

// Golden Images runs against the deterministic E2E fixture seeded by
// scripts/seed-e2e-data. The seed inserts 4 production golden images, one per
// family; the API client groups them into 4 single-version families:
//   golden-amazonlinux-2023  v2024.11.0  production
//   golden-rhel-9            v2024.09.0  production
//   golden-ubuntu-22         v2024.11.0  production
//   golden-windows-2022      v2024.10.0  production
// So: Image Families = 4, Active Versions = 4, 0 pending, 0 deprecated. These
// are the stable facts the specs assert. We deliberately do NOT use
// waitForLoadState("networkidle") (background queries keep the network busy);
// toBeVisible auto-waits per element.
const WIDGET_TIMEOUT = 20_000;

test.describe("Images Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/images");
    await expect(page.getByRole("heading", { name: "Golden Images", exact: true })).toBeVisible({
      timeout: WIDGET_TIMEOUT,
    });
  });

  test("should display the header and description", async ({ page }) => {
    const main = page.getByRole("main");
    await expect(
      main.getByText("Manage and track your golden image families and versions.", { exact: true }),
    ).toBeVisible({ timeout: WIDGET_TIMEOUT });
  });

  test("should display image metric cards", async ({ page }) => {
    const main = page.getByRole("main");
    await expect(main.getByText("Image Families", { exact: true })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(main.getByText("Active Versions", { exact: true })).toBeVisible();
    await expect(main.getByText("Pending Promotions", { exact: true })).toBeVisible();
    await expect(main.getByText("Deprecated", { exact: true })).toBeVisible();
  });

  test("should list the seeded golden-image families and versions", async ({ page }) => {
    const main = page.getByRole("main");

    // All four seeded families render in the table.
    await expect(main.getByText("golden-amazonlinux-2023", { exact: true })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(main.getByText("golden-ubuntu-22", { exact: true })).toBeVisible();
    await expect(main.getByText("golden-windows-2022", { exact: true })).toBeVisible();
    await expect(main.getByText("golden-rhel-9", { exact: true })).toBeVisible();

    // Latest-version codes (rendered as `v{version}`). 2024.11.0 is shared by
    // two families, so assert presence (first); the others are unique.
    await expect(main.getByText("v2024.11.0", { exact: true }).first()).toBeVisible();
    await expect(main.getByText("v2024.10.0", { exact: true })).toBeVisible();
    await expect(main.getByText("v2024.09.0", { exact: true })).toBeVisible();

    // Every seeded image is production.
    await expect(main.getByText("Production", { exact: true }).first()).toBeVisible();
  });

  test("filtering by a status with no seeded images shows the empty state", async ({ page }) => {
    const main = page.getByRole("main");
    // Baseline: a seeded family is present.
    await expect(main.getByText("golden-ubuntu-22", { exact: true })).toBeVisible({ timeout: WIDGET_TIMEOUT });

    // No images are deprecated in the fixture, so filtering to Deprecated empties
    // the table.
    await main.getByRole("combobox").click();
    await page.getByRole("option", { name: "Deprecated", exact: true }).click();

    await expect(main.getByText("No images found", { exact: true })).toBeVisible();
    await expect(main.getByText("golden-ubuntu-22", { exact: true })).toHaveCount(0);
  });
});
