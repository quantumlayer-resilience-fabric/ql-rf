import { defineConfig, devices } from "@playwright/test";

/**
 * Playwright E2E Test Configuration
 * See https://playwright.dev/docs/test-configuration
 *
 * Authentication with Clerk:
 * - Uses @clerk/testing for auth integration
 * - Requires E2E_CLERK_USER_USERNAME and E2E_CLERK_USER_PASSWORD env vars
 * - Auth state is saved in playwright/.clerk/user.json and reused
 *
 * To set up:
 * 1. Create a test user in Clerk Dashboard with password auth
 * 2. Add to .env.local:
 *    E2E_CLERK_USER_USERNAME=your-test-email@example.com
 *    E2E_CLERK_USER_PASSWORD=your-test-password
 * 3. Run: npm run test:e2e
 */
export default defineConfig({
  testDir: "./e2e",
  // Ignore auth setup files in regular test runs (they run via globalSetup)
  testIgnore: ["**/auth/**"],
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 1,
  workers: process.env.CI ? 1 : undefined,
  timeout: 60000, // 60 seconds per test
  expect: {
    timeout: 10000, // 10 seconds for assertions
  },
  reporter: [
    ["html", { open: "never", outputFolder: "playwright-report" }],
    ["list"],
    ["json", { outputFile: "test-results/results.json" }],
  ],
  // Global setup - handles Clerk auth before all tests
  globalSetup: require.resolve("./e2e/auth/global.setup.ts"),
  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL || "http://localhost:3000",
    // Load authenticated session from global setup
    storageState: "playwright/.clerk/user.json",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
    actionTimeout: 15000, // 15 seconds for actions
    navigationTimeout: 30000, // 30 seconds for navigation
    extraHTTPHeaders: {
      "Accept-Language": "en-US",
    },
  },
  // Output folder for test artifacts
  outputDir: "test-results",
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
    {
      name: "firefox",
      use: { ...devices["Desktop Firefox"] },
    },
    {
      name: "webkit",
      use: { ...devices["Desktop Safari"] },
    },
    /* Mobile viewports */
    {
      name: "mobile-chrome",
      use: { ...devices["Pixel 5"] },
    },
    {
      name: "mobile-safari",
      use: { ...devices["iPhone 12"] },
    },
  ],
  /* Run local dev server before tests if not in CI */
  webServer: process.env.CI
    ? undefined
    : {
        command: "npm run dev",
        url: "http://localhost:3000",
        reuseExistingServer: true,
        timeout: 120 * 1000,
      },
});
