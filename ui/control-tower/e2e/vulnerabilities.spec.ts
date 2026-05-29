import { test, expect } from "@playwright/test";

// The Vulnerability Response Center is served by the orchestrator (:8083), not
// the API. It runs against the deterministic CVE fixture seeded by
// scripts/seed-e2e-data, which inserts 5 cve_alerts (+ cve_cache) under the org
// the orchestrator resolves in dev mode:
//   CVE-2024-3094   critical  urgency 98  KEV + exploit + SLA-breached
//   CVE-2021-44228  critical  urgency 95  KEV + exploit
//   CVE-2024-21626  high      urgency 72  exploit
//   CVE-2023-44487  high      urgency 65
//   CVE-2024-6387   medium    urgency 45  (resolved)
// So the summary is 5 total, 2 critical + 2 high, 2 CISA KEV, 3 exploitable,
// 1 SLA-breached, 15 affected assets (6 production). We deliberately do NOT use
// waitForLoadState("networkidle") (background queries keep the network busy);
// toBeVisible auto-waits per element.
const WIDGET_TIMEOUT = 20_000;

test.describe("Vulnerabilities Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/vulnerabilities");
    await expect(
      page.getByRole("heading", { name: "Vulnerability Response Center", exact: true }),
    ).toBeVisible({ timeout: WIDGET_TIMEOUT });
  });

  test("should display the header and description", async ({ page }) => {
    const main = page.getByRole("main");
    await expect(
      main.getByText(
        "Real-time CVE detection, blast radius analysis, and automated patch orchestration.",
        { exact: true },
      ),
    ).toBeVisible({ timeout: WIDGET_TIMEOUT });
  });

  test("should display the CVE metric cards", async ({ page }) => {
    const main = page.getByRole("main");
    await expect(main.getByText("Total Alerts", { exact: true })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(main.getByText("Critical/High", { exact: true })).toBeVisible();
    // "CISA KEV" is also the AI-insight badge text, so scope to the first match (the card).
    await expect(main.getByText("CISA KEV", { exact: true }).first()).toBeVisible();
    await expect(main.getByText("Affected Assets", { exact: true })).toBeVisible();
  });

  test("should list the seeded critical and high CVEs with indicators", async ({ page }) => {
    const table = page.getByRole("table");
    await expect(table).toBeVisible({ timeout: WIDGET_TIMEOUT });

    // Top critical CVE: severity + CISA KEV indicator in its row.
    const critRow = table.getByRole("row").filter({ hasText: "CVE-2024-3094" });
    await expect(critRow.getByText("Critical", { exact: true })).toBeVisible();
    await expect(critRow.getByText("KEV", { exact: true })).toBeVisible();

    // Second critical and a high CVE are present.
    await expect(table.getByText("CVE-2021-44228", { exact: true })).toBeVisible();
    const highRow = table.getByRole("row").filter({ hasText: "CVE-2024-21626" });
    await expect(highRow.getByText("High", { exact: true })).toBeVisible();
  });

  test("should surface the CISA KEV insight and SLA breach warning", async ({ page }) => {
    const main = page.getByRole("main");
    await expect(main.getByText("2 CISA KEV CVEs Detected", { exact: true })).toBeVisible({
      timeout: WIDGET_TIMEOUT,
    });
    await expect(main.getByText("1 alerts have breached SLA")).toBeVisible();
  });

  test("should show seeded counts in the tab labels", async ({ page }) => {
    await expect(page.getByRole("tab", { name: /All Alerts \(5\)/ })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(page.getByRole("tab", { name: /CISA KEV \(2\)/ })).toBeVisible();
    await expect(page.getByRole("tab", { name: /Exploitable \(3\)/ })).toBeVisible();
  });
});
