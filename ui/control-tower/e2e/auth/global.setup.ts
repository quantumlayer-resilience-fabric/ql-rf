/**
 * Playwright Global Setup for Clerk Authentication
 *
 * This file runs BEFORE all tests to:
 * 1. Initialize Clerk testing mode
 * 2. Authenticate a test user via UI
 * 3. Save the session for reuse
 *
 * Required environment variables:
 * - E2E_CLERK_USER_USERNAME: Test user email
 * - E2E_CLERK_USER_PASSWORD: Test user password
 *
 * Setup:
 * 1. Create a test user in Clerk Dashboard with password enabled
 * 2. Enable "Password" under Configure > Email, Phone, Username
 * 3. Add credentials to .env.local
 *
 * @see https://clerk.com/docs/guides/development/testing/playwright/overview
 */
import { clerkSetup, setupClerkTestingToken } from "@clerk/testing/playwright";
import { chromium, FullConfig } from "@playwright/test";
import path from "path";

// Path to store authenticated session
const authFile = path.join(__dirname, "../../playwright/.clerk/user.json");

async function globalSetup(config: FullConfig) {
  // Step 1: Initialize Clerk testing
  await clerkSetup();

  const username = process.env.E2E_CLERK_USER_USERNAME;
  const password = process.env.E2E_CLERK_USER_PASSWORD;

  if (!username || !password) {
    throw new Error(
      "E2E tests require authentication credentials.\n\n" +
        "Add to .env.local:\n" +
        "  E2E_CLERK_USER_USERNAME=your-test-user@example.com\n" +
        "  E2E_CLERK_USER_PASSWORD=your-test-password\n\n" +
        "Also create this user in Clerk Dashboard with password auth enabled."
    );
  }

  // Step 2: Launch browser and authenticate
  const baseURL = config.projects[0].use.baseURL || "http://localhost:3000";
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

  try {
    // Set up testing token to bypass Clerk bot detection
    await setupClerkTestingToken({ page });

    // Navigate to login page directly
    console.log("[E2E Auth] Navigating to login page...");
    await page.goto(`${baseURL}/login`);
    await page.waitForLoadState("domcontentloaded");

    // Take screenshot to debug
    await page.screenshot({ path: "playwright/.clerk/step1-login-page.png" });

    // Fill in email/username
    console.log("[E2E Auth] Filling email:", username);
    const emailInput = page.locator('input[name="identifier"], input[type="email"], input[placeholder*="email"]').first();
    await emailInput.waitFor({ state: "visible", timeout: 10000 });
    await emailInput.fill(username);

    // Click continue button
    const continueButton = page.locator('button:has-text("Continue")').first();
    await continueButton.click();

    // Wait for password field to appear
    console.log("[E2E Auth] Waiting for password field...");
    await page.waitForTimeout(1000);
    await page.screenshot({ path: "playwright/.clerk/step2-after-continue.png" });

    const passwordInput = page.locator('input[type="password"], input[name="password"]').first();
    await passwordInput.waitFor({ state: "visible", timeout: 10000 });
    await passwordInput.fill(password);

    // Click sign in button
    const signInButton = page.locator('button:has-text("Continue"), button:has-text("Sign in")').first();
    await signInButton.click();

    // Handle potential email verification (new device check)
    await page.waitForTimeout(2000);
    const currentUrl = page.url();

    if (currentUrl.includes("factor-two") || currentUrl.includes("verification")) {
      console.log("[E2E Auth] Email verification required (new device). Entering code 424242...");
      await page.screenshot({ path: "playwright/.clerk/step3-verification.png" });

      // Use test code 424242 for Clerk test emails
      // This works for emails containing +clerk_test
      const codeInputs = page.locator('input[type="text"], input[inputmode="numeric"]');
      const inputCount = await codeInputs.count();

      if (inputCount >= 6) {
        // Individual digit inputs
        const code = "424242";
        for (let i = 0; i < 6; i++) {
          await codeInputs.nth(i).fill(code[i]);
        }
      } else if (inputCount >= 1) {
        // Single input for full code
        await codeInputs.first().fill("424242");
      }

      // Click continue after entering code
      const verifyButton = page.locator('button:has-text("Continue"), button:has-text("Verify")').first();
      await verifyButton.click();
    }

    // Wait for redirect to dashboard
    console.log("[E2E Auth] Waiting for redirect...");
    await page.waitForURL(/\/(overview|dashboard)/, { timeout: 15000 });

    await page.screenshot({ path: "playwright/.clerk/step3-authenticated.png" });
    console.log("[E2E Auth] Current URL:", page.url());

    // Save the authenticated state for reuse in tests
    await context.storageState({ path: authFile });
    console.log("[E2E Auth] Authentication successful, session saved to", authFile);
  } catch (e) {
    // Take screenshot on failure
    await page.screenshot({ path: "playwright/.clerk/auth-failed.png" });
    console.error("[E2E Auth] Authentication failed. Current URL:", page.url());
    console.error("[E2E Auth] See playwright/.clerk/*.png for debug screenshots");
    throw e;
  } finally {
    await browser.close();
  }
}

export default globalSetup;
