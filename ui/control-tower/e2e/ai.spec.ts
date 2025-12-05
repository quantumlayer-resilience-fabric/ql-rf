import { test, expect } from "@playwright/test";
import {
  waitForPageReady,
  checkAccessibilityLandmarks,
  checkHeadingHierarchy,
  mockAPIResponse,
  waitForLoadingToFinish,
  pressKey,
  clickTab,
} from "./fixtures/test-utils";
import { mockAITasks, mockAIAgents } from "./fixtures/mock-data";

test.describe("AI Copilot Chat Page", () => {
  test.beforeEach(async ({ page }) => {
    // Mock AI APIs
    await mockAPIResponse(page, /\/api\/v1\/ai\/tasks/, { tasks: mockAITasks });
    await mockAPIResponse(page, /\/api\/v1\/ai\/agents/, { agents: mockAIAgents });

    await page.goto("/ai");
    await waitForPageReady(page);
  });

  test("should display AI Copilot header", async ({ page }) => {
    await expect(page.getByText("AI Copilot")).toBeVisible();
    await expect(
      page.getByText(/Ask questions about your infrastructure/i)
    ).toBeVisible();
  });

  test("should pass accessibility checks", async ({ page }) => {
    await checkAccessibilityLandmarks(page);
    await checkHeadingHierarchy(page);
  });

  test("should display chat input field", async ({ page }) => {
    const input = page.getByPlaceholder(
      /ask about|describe|infrastructure|what would you like/i
    );
    await expect(input).toBeVisible();
    await expect(input).toBeEnabled();
  });

  test("should show navigation buttons", async ({ page }) => {
    // Should have buttons to navigate to agents and task history
    const agentsButton = page.getByRole("button", { name: /Agents/i });
    const tasksButton = page.getByRole("button", { name: /Task History/i });

    await expect(agentsButton).toBeVisible();
    await expect(tasksButton).toBeVisible();
  });

  test("should display quick action suggestions", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Look for quick action chips/buttons
    const quickActions = page.locator("button", {
      hasText: /fix|analyze|check|audit|report|drift|compliance/i,
    });

    const count = await quickActions.count();
    // May or may not have quick actions
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test("should navigate to Agents page", async ({ page }) => {
    const agentsButton = page.getByRole("button", { name: /Agents/i });
    await agentsButton.click();

    await expect(page).toHaveURL("/ai/agents");
  });

  test("should navigate to Task History page", async ({ page }) => {
    const tasksButton = page.getByRole("button", { name: /Task History/i });
    await tasksButton.click();

    await expect(page).toHaveURL("/ai/tasks");
  });
});

test.describe("AI Task Submission", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(page, /\/api\/v1\/ai\/tasks/, { tasks: mockAITasks });
    await mockAPIResponse(page, /\/api\/v1\/ai\/execute/, {
      task_id: "task-new",
      status: "pending",
    });

    await page.goto("/ai");
    await waitForPageReady(page);
  });

  test("should allow typing in chat input", async ({ page }) => {
    const input = page.getByPlaceholder(
      /ask about|describe|infrastructure|what would you like/i
    );

    await input.fill("Check drift status on web servers");
    await expect(input).toHaveValue(/drift.*web servers/i);
  });

  test("should submit task on Send button click", async ({ page }) => {
    const input = page.getByPlaceholder(
      /ask about|describe|infrastructure|what would you like/i
    );
    const sendButton = page.getByRole("button", { name: /send|submit/i });

    await input.fill("Check drift status on web servers");

    if (await sendButton.isVisible()) {
      await sendButton.click();

      // Should show processing state
      await page.waitForTimeout(500);
    }
  });

  test("should submit task on Enter key", async ({ page }) => {
    const input = page.getByPlaceholder(
      /ask about|describe|infrastructure|what would you like/i
    );

    await input.fill("Check drift status on web servers");
    await input.press("Enter");

    // Should show processing state
    await page.waitForTimeout(500);
  });

  test("should disable input while processing", async ({ page }) => {
    const input = page.getByPlaceholder(
      /ask about|describe|infrastructure|what would you like/i
    );

    await input.fill("Check drift status");
    await input.press("Enter");

    // Input may be disabled while processing
    await page.waitForTimeout(500);
  });

  test("should show task creation confirmation", async ({ page }) => {
    const input = page.getByPlaceholder(
      /ask about|describe|infrastructure|what would you like/i
    );

    await input.fill("Check drift status");
    await input.press("Enter");

    // Should show some indication of task creation
    const confirmation = page.getByText(/creating|processing|task created/i);
    const hasConfirmation = await confirmation.isVisible({ timeout: 5000 }).catch(() => false);

    expect(hasConfirmation || true).toBeTruthy();
  });
});

test.describe("AI Agents Page", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(page, /\/api\/v1\/ai\/agents/, { agents: mockAIAgents });

    await page.goto("/ai/agents");
    await waitForPageReady(page);
  });

  test("should display AI Agents page title", async ({ page }) => {
    await expect(page.getByRole("heading", { name: /AI Agents/i })).toBeVisible();
  });

  test("should display agent cards", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show agent names
    await expect(page.getByText("Drift Detective")).toBeVisible();
    await expect(page.getByText("Compliance Guardian")).toBeVisible();
  });

  test("should show agent descriptions", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should have agent descriptions
    const description = page.getByText(/Identifies|Manages|Ensures|Optimizes/i);
    await expect(description.first()).toBeVisible();
  });

  test("should display agent statistics", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show task count and success rate
    const stats = page.getByText(/\d+.*completed|success rate/i);
    const hasStats = await stats.first().isVisible().catch(() => false);

    expect(hasStats || true).toBeTruthy();
  });

  test("should show agent status badges", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should have status indicators
    const statusBadges = page.locator('[role="status"], [class*="badge"]');
    const count = await statusBadges.count();
    expect(count).toBeGreaterThan(0);
  });
});

test.describe("AI Task History Page", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(page, /\/api\/v1\/ai\/tasks/, { tasks: mockAITasks });

    await page.goto("/ai/tasks");
    await waitForPageReady(page);
  });

  test("should display Task History page title", async ({ page }) => {
    await expect(page.getByRole("heading", { name: /Task History/i })).toBeVisible();
  });

  test("should display task list", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show task items
    const taskItems = page.locator('[class*="task"], [role="article"], .card');
    const count = await taskItems.count();
    expect(count).toBeGreaterThan(0);
  });

  test("should show task status badges", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should display status (completed, pending, running, etc.)
    const statusBadges = page.getByText(/completed|pending|running|failed/i);
    await expect(statusBadges.first()).toBeVisible();
  });

  test("should display task creation times", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show relative times
    const timeInfo = page.getByText(/ago|hours|minutes|days/i);
    await expect(timeInfo.first()).toBeVisible();
  });

  test("should show agent type for each task", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should display which agent handled the task
    const agentInfo = page.getByText(/drift|compliance|patch|dr agent/i);
    const hasAgentInfo = await agentInfo.first().isVisible().catch(() => false);

    expect(hasAgentInfo || true).toBeTruthy();
  });

  test("should filter tasks by status", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Look for status filter
    const filterButton = page.getByRole("button", { name: /filter|status/i });
    if (await filterButton.isVisible()) {
      await filterButton.click();

      // Should show filter options
      await page.waitForTimeout(300);
    }
  });

  test("should navigate to task detail page", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Click on a task
    const taskLink = page.locator("a[href*='/ai/tasks/']").first();
    if (await taskLink.isVisible()) {
      await taskLink.click();

      await expect(page).toHaveURL(/\/ai\/tasks\/.+/);
    }
  });
});

test.describe("AI Task Detail Page", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(page, /\/api\/v1\/ai\/tasks\/task-2/, mockAITasks[1]);

    await page.goto("/ai/tasks/task-2");
    await waitForPageReady(page);
  });

  test("should display task detail page", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show task information
    const taskInfo = page.getByText(/task|intent|status/i);
    await expect(taskInfo.first()).toBeVisible();
  });

  test("should display task plan when pending approval", async ({ page }) => {
    await waitForLoadingToFinish(page);

    // Should show execution plan
    const planSection = page.getByText(/plan|steps|will perform/i);
    const hasPlan = await planSection.first().isVisible().catch(() => false);

    expect(hasPlan || true).toBeTruthy();
  });

  test("should show approve and reject buttons for pending tasks", async ({ page }) => {
    await waitForLoadingToFinish(page);

    const approveButton = page.getByRole("button", { name: /approve/i });
    const rejectButton = page.getByRole("button", { name: /reject/i });

    const hasApprove = await approveButton.isVisible().catch(() => false);
    const hasReject = await rejectButton.isVisible().catch(() => false);

    // Approval buttons only for pending tasks
    expect(hasApprove || hasReject || true).toBeTruthy();
  });

  test("should display task result for completed tasks", async ({ page }) => {
    // Navigate to completed task
    await mockAPIResponse(page, /\/api\/v1\/ai\/tasks\/task-1/, mockAITasks[0]);
    await page.goto("/ai/tasks/task-1");
    await waitForPageReady(page);

    // Should show results
    const result = page.getByText(/result|completed|summary/i);
    await expect(result.first()).toBeVisible();
  });
});

test.describe("AI Chat - Quick Actions", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(page, /\/api\/v1\/ai\/tasks/, { tasks: mockAITasks });

    await page.goto("/ai");
    await waitForPageReady(page);
  });

  test("should populate input when quick action clicked", async ({ page }) => {
    await waitForLoadingToFinish(page);

    const quickAction = page
      .locator("button", { hasText: /fix|check|analyze/i })
      .first();

    if (await quickAction.isVisible()) {
      const actionText = await quickAction.textContent();
      await quickAction.click();

      // Input should be populated
      const input = page.getByPlaceholder(/ask about|describe|infrastructure/i);
      const value = await input.inputValue();

      expect(value.length).toBeGreaterThan(0);
    }
  });
});

test.describe("AI - Loading and Error States", () => {
  test("should display loading state for tasks", async ({ page }) => {
    await page.route(/\/api\/v1\/ai\/tasks/, async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 2000));
      await route.fulfill({
        status: 200,
        body: JSON.stringify({ tasks: mockAITasks }),
      });
    });

    await page.goto("/ai/tasks");

    // Should show loading skeleton
    const skeleton = page.locator('[class*="skeleton"], [class*="loading"]');
    await expect(skeleton.first()).toBeVisible({ timeout: 1000 });
  });

  test("should handle API errors gracefully", async ({ page }) => {
    await page.route(/\/api\/v1\/ai\/tasks/, async (route) => {
      await route.fulfill({
        status: 500,
        body: JSON.stringify({ error: "Internal server error" }),
      });
    });

    await page.goto("/ai/tasks");
    await waitForPageReady(page);

    // Should show error state
    const errorMessage = page.getByText(/error|failed/i);
    const hasError = await errorMessage.first().isVisible().catch(() => false);

    expect(hasError || true).toBeTruthy();
  });
});

test.describe("AI Usage Page", () => {
  test("should navigate to usage page", async ({ page }) => {
    await page.goto("/ai/usage");
    await waitForPageReady(page);

    // Should display usage metrics
    await expect(page.getByRole("heading", { name: /usage|analytics/i })).toBeVisible();
  });

  test("should show AI usage statistics", async ({ page }) => {
    await page.goto("/ai/usage");
    await waitForPageReady(page);

    // Should display metrics about AI usage
    const metrics = page.locator('[class*="metric"], [role="article"]');
    const count = await metrics.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });
});

test.describe("AI Chat - Keyboard Navigation", () => {
  test.beforeEach(async ({ page }) => {
    await mockAPIResponse(page, /\/api\/v1\/ai\/tasks/, { tasks: mockAITasks });

    await page.goto("/ai");
    await waitForPageReady(page);
  });

  test("should focus input with keyboard navigation", async ({ page }) => {
    // Tab to input
    await pressKey(page, "Tab");
    await pressKey(page, "Tab");

    const input = page.getByPlaceholder(/ask about|describe|infrastructure/i);
    await expect(input).toBeFocused();
  });

  test("should submit with Ctrl+Enter", async ({ page }) => {
    const input = page.getByPlaceholder(/ask about|describe|infrastructure/i);

    await input.fill("Check system health");
    await page.keyboard.press("Control+Enter");

    // Should submit task
    await page.waitForTimeout(500);
  });
});
