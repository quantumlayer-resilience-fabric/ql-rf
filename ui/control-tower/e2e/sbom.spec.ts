import { test, expect } from "@playwright/test";

// SBOM runs against the deterministic fixture seeded by scripts/seed-e2e-data:
// one SPDX SBOM linked to golden-ubuntu-22 with six realistic deb packages
// (bash, ca-certificates, libc6, openssl, systemd, zlib1g). No vulnerabilities
// are seeded. fix(sbom) shipped two corrections so the page renders the real
// API data:
//   - api-sbom.ts now transforms snake_case -> camelCase, so generatedAt
//     resolves and Last Generated shows a real relative time;
//   - /sbom/licenses/summary is a real endpoint that aggregates licenses out of
//     sbom_packages (5 distinct licenses: GPL-3.0, LGPL-2.1, MPL-2.0, OpenSSL,
//     Zlib).
//
// Resulting page state:
//   Total Components    6
//   Vulnerabilities     0 (subtitle "critical/high of 0 total")
//   License Compliance  100.0% (6 licensed / 6 total)
//   Last Generated      a "just now" / "X min ago" relative-time string
//   Tab labels          Components (6) / Vulnerabilities (0) / Licenses (5)
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

    // License Compliance is 100.0% (all 6 packages licensed). Last Generated is
    // now a real relative-time string ("just now" or "<N> min ago") because the
    // api-sbom client transforms generated_at to generatedAt — assert the
    // pattern rather than an exact value to stay clock-robust.
    await expect(main.getByText("100.0%", { exact: true })).toBeVisible();
    await expect(main.getByText(/just now|ago/)).toBeVisible();
  });

  test("should show seeded counts in the tab labels", async ({ page }) => {
    await expect(page.getByRole("tab", { name: /Components \(6\)/ })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(page.getByRole("tab", { name: /Vulnerabilities \(0\)/ })).toBeVisible();
    await expect(page.getByRole("tab", { name: /Licenses \(5\)/ })).toBeVisible();
  });

  test("should list the seeded packages in the components tab", async ({ page }) => {
    const main = page.getByRole("main");
    // Default tab is Components, so the inventory table renders on first paint.
    await expect(main.getByText("bash", { exact: true })).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(main.getByText("openssl", { exact: true })).toBeVisible();
    await expect(main.getByText("ca-certificates", { exact: true })).toBeVisible();
  });
});
