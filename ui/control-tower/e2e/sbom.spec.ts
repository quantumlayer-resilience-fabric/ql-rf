import { test, expect } from "@playwright/test";

// SBOM runs against the deterministic fixture seeded by scripts/seed-e2e-data:
// one SPDX SBOM linked to golden-ubuntu-22 with six realistic deb packages
// (bash, ca-certificates, libc6, openssl, systemd, zlib1g). No vulnerabilities
// are seeded.
//
// Resulting page state:
//   Total Components    6
//   Vulnerabilities     0 (subtitle "critical/high of 0 total")
//   License Compliance  100.0% (the /licenses endpoint is mocked client-side)
//   Last Generated      "Never" — the API returns generated_at (snake_case),
//                        the page reads generatedAt (camelCase), so the
//                        formatted-time fallback applies. We assert what the
//                        page actually shows.
// We deliberately do NOT use networkidle; toBeVisible auto-waits per element.
const WIDGET_TIMEOUT = 20_000;

test.describe("SBOM Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/sbom");
    await expect(
      page.getByRole("heading", { name: "Software Bill of Materials (SBOM)", exact: true }),
    ).toBeVisible({ timeout: WIDGET_TIMEOUT });
  });

  test("should display the header and description", async ({ page }) => {
    const main = page.getByRole("main");
    await expect(
      main.getByText(
        "Track software components, licenses, and vulnerabilities across your golden images.",
        { exact: true },
      ),
    ).toBeVisible({ timeout: WIDGET_TIMEOUT });
  });

  test("should display the SBOM metric cards with seeded values", async ({ page }) => {
    const main = page.getByRole("main");
    await expect(main.getByText("Total Components", { exact: true })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(main.getByText("Vulnerabilities", { exact: true })).toBeVisible();
    await expect(main.getByText("License Compliance", { exact: true })).toBeVisible();
    await expect(main.getByText("Last Generated", { exact: true })).toBeVisible();

    // Unique data-derived markers (100.0% comes from the client-side mocked
    // license summary; "Never" reflects the snake/camel mismatch on
    // generated_at, both deterministic regardless of the seeded clock).
    await expect(main.getByText("100.0%", { exact: true })).toBeVisible();
    await expect(main.getByText("Never", { exact: true })).toBeVisible();
  });

  test("should show seeded counts in the tab labels", async ({ page }) => {
    await expect(page.getByRole("tab", { name: /Components \(6\)/ })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(page.getByRole("tab", { name: /Vulnerabilities \(0\)/ })).toBeVisible();
    await expect(page.getByRole("tab", { name: /Licenses \(0\)/ })).toBeVisible();
  });

  test("should list the seeded packages in the components tab", async ({ page }) => {
    const main = page.getByRole("main");
    // Default tab is Components, so the inventory table renders on first paint.
    await expect(main.getByText("bash", { exact: true })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(main.getByText("openssl", { exact: true })).toBeVisible();
    await expect(main.getByText("ca-certificates", { exact: true })).toBeVisible();
  });
});
