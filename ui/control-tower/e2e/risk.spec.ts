import { test, expect } from "@playwright/test";

// Risk Analysis runs against the deterministic E2E fixture seeded by
// scripts/seed-e2e-data. The risk service scores each asset from drift,
// vulnerabilities, compliance and an environment multiplier. The fixture seeds
// vulnerabilities on two isolated golden images (each used by a single asset),
// with all assets in the production environment (1.5x multiplier), producing a
// stable distribution:
//   e2e-aws-risk-high     golden-rhel-9    20 vulns / 4 critical -> 68  HIGH
//   e2e-azure-risk-medium golden-windows-2022  4 vulns / 4 critical -> 44  MEDIUM
//   (8 other assets, no vulnerabilities)                          ->  0  LOW
// So the summary is 0 critical, 1 high, 1 medium, 8 low. We deliberately do NOT
// use waitForLoadState("networkidle") (background queries keep the network
// busy); toBeVisible auto-waits per element.
const WIDGET_TIMEOUT = 20_000;

test.describe("Risk Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/risk");
    await expect(page.getByRole("heading", { name: "Risk Analysis", exact: true })).toBeVisible({
      timeout: WIDGET_TIMEOUT,
    });
  });

  test("should display the header and description", async ({ page }) => {
    const main = page.getByRole("main");
    await expect(
      main.getByText("AI-powered risk scoring across your infrastructure", { exact: true }),
    ).toBeVisible({ timeout: WIDGET_TIMEOUT });
  });

  test("should display the risk summary cards", async ({ page }) => {
    const main = page.getByRole("main");
    await expect(main.getByText("Overall Risk", { exact: true })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(main.getByText("Critical Risk", { exact: true })).toBeVisible();
    await expect(main.getByText("High Risk", { exact: true })).toBeVisible();
    await expect(main.getByText("Medium Risk", { exact: true })).toBeVisible();
    await expect(main.getByText("Low Risk", { exact: true })).toBeVisible();
  });

  test("should list the seeded high- and medium-risk assets in the top risks table", async ({ page }) => {
    const table = page.getByRole("table");
    await expect(table).toBeVisible({ timeout: WIDGET_TIMEOUT });

    const highRow = table.getByRole("row").filter({ hasText: "e2e-aws-risk-high" });
    await expect(highRow.getByText("High", { exact: true })).toBeVisible();
    await expect(highRow.getByText("4 Critical CVE")).toBeVisible();

    const mediumRow = table.getByRole("row").filter({ hasText: "e2e-azure-risk-medium" });
    await expect(mediumRow.getByText("Medium", { exact: true })).toBeVisible();
  });

  test("should reflect the seeded risk distribution (1 high, 1 medium, no critical)", async ({ page }) => {
    const table = page.getByRole("table");
    await expect(table).toBeVisible({ timeout: WIDGET_TIMEOUT });

    // Level badges live only in the top-risks table (other tabs are unmounted).
    await expect(table.getByText("High", { exact: true })).toHaveCount(1);
    await expect(table.getByText("Medium", { exact: true })).toHaveCount(1);
    await expect(table.getByText("Critical", { exact: true })).toHaveCount(0);
  });
});
