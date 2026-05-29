import { test, expect } from "@playwright/test";

// Certificate Lifecycle Management runs against the deterministic certificate
// fixture seeded by scripts/seed-e2e-data. The certificates table's status
// trigger derives status/days_until_expiry from not_after, so the seed's five
// certs (set relative to now) classify deterministically:
//   api.quantumlayer.io        aws      active        (~825d, auto-renew)
//   app.quantumlayer.io        azure    active        (~600d, auto-renew)
//   dashboard.quantumlayer.io  gcp      expiring_soon (~15d)
//   legacy.quantumlayer.io     k8s      expiring_soon (~5d, within 7)
//   old.quantumlayer.io        vsphere  expired       (~-10d)
// Summary: 5 total, 2 active, 2 expiring soon (1 within 7 days), 1 expired,
// 5 platforms, 2 auto-renew. We deliberately do NOT assert exact day counts
// (the trigger truncates) nor use networkidle; toBeVisible auto-waits.
const WIDGET_TIMEOUT = 20_000;

test.describe("Certificates Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/certificates");
    await expect(
      page.getByRole("heading", { name: "Certificate Lifecycle Management", exact: true }),
    ).toBeVisible({ timeout: WIDGET_TIMEOUT });
  });

  test("should display the header and description", async ({ page }) => {
    const main = page.getByRole("main");
    await expect(
      main.getByText("Monitor, rotate, and manage TLS/SSL certificates across your infrastructure.", {
        exact: true,
      }),
    ).toBeVisible({ timeout: WIDGET_TIMEOUT });
  });

  test("should display the certificate metric cards with seeded values", async ({ page }) => {
    const main = page.getByRole("main");
    await expect(main.getByText("Total Certificates", { exact: true })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    // Data-derived card subtitles (unique on the page, unlike the status words
    // which also appear as table badges).
    await expect(main.getByText("5 platforms", { exact: true })).toBeVisible();
    await expect(main.getByText("2 auto-renew", { exact: true })).toBeVisible();
    await expect(main.getByText("1 within 7 days", { exact: true })).toBeVisible();
    await expect(main.getByText("requires attention", { exact: true })).toBeVisible();
  });

  test("should list the seeded certificates with their statuses", async ({ page }) => {
    const table = page.getByRole("table");
    await expect(table).toBeVisible({ timeout: WIDGET_TIMEOUT });

    // Common names render in the inventory table.
    await expect(table.getByText("api.quantumlayer.io", { exact: true })).toBeVisible();
    await expect(table.getByText("dashboard.quantumlayer.io", { exact: true })).toBeVisible();
    await expect(table.getByText("old.quantumlayer.io", { exact: true })).toBeVisible();

    // Status badges, scoped to their certificate's row.
    const activeRow = table.getByRole("row").filter({ hasText: "api.quantumlayer.io" });
    await expect(activeRow.getByText("Active", { exact: true })).toBeVisible();

    const expiringRow = table.getByRole("row").filter({ hasText: "dashboard.quantumlayer.io" });
    await expect(expiringRow.getByText("Expiring Soon", { exact: true })).toBeVisible();

    const expiredRow = table.getByRole("row").filter({ hasText: "old.quantumlayer.io" });
    await expect(expiredRow.getByText("Expired", { exact: true })).toBeVisible();
  });

  test("should show the seeded tab count and the attention insight", async ({ page }) => {
    await expect(page.getByRole("tab", { name: /Certificates \(5\)/ })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    // criticalCount = expired (1) + expiring-within-7-days (1) = 2.
    await expect(
      page.getByRole("main").getByText("2 Certificates Need Attention", { exact: true }),
    ).toBeVisible();
  });
});
