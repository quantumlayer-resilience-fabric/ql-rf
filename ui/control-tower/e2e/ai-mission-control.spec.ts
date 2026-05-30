import { test, expect } from "@playwright/test";

// Mission Control — AI-001 / E2E-011 Phase A.
//
// Backed by scripts/seed-e2e-data (seedMissionControl) which inserts a
// deterministic AI lifecycle under the orchestrator dev org:
//
//   ai_tasks  = 4
//   ai_plans  = 4  (1 awaiting_approval, 2 approved, 1 rejected)
//   ai_runs   = 2  (1 executing, 1 completed)
//   ai_tool_invocations = 6  across vulnerability/drift/certificate agents
//   llm_usage = 4 rows totalling 182 cents = $1.82
//
// Resulting page state, verified against /api/v1/ai/fleet/status before
// writing assertions:
//   agents:        12 total, 1 working, 11 idle, 0 blocked
//   pending:       1  (Patch CVE-2024-3094, OPA pass, quality 87)
//   tool runs:     6 today
//   LLM spend:     $1.82 / $50.00
//
// Phase A is read-only — no mutation paths exercised.
const WIDGET_TIMEOUT = 20_000;

test.describe("Mission Control (AI-001 Phase A)", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/ai");
    await expect(
      page.getByRole("heading", { name: "Mission Control", exact: true }),
    ).toBeVisible({ timeout: WIDGET_TIMEOUT });
  });

  test("should display the header and subtitle", async ({ page }) => {
    const main = page.getByRole("main");
    await expect(
      main.getByText("Governed command for infrastructure agents.", { exact: true }),
    ).toBeVisible({ timeout: WIDGET_TIMEOUT });
  });

  test("should display the fleet status bar with seeded counts and spend", async ({ page }) => {
    await expect(page.getByTestId("fleet-working")).toContainText("1");
    await expect(page.getByTestId("fleet-working")).toContainText("working");
    await expect(page.getByTestId("fleet-idle")).toContainText("11");
    await expect(page.getByTestId("fleet-pending")).toContainText("1");
    await expect(page.getByTestId("fleet-actions-today")).toContainText("6");
    await expect(page.getByTestId("fleet-spend")).toContainText("$1.82");
    await expect(page.getByTestId("fleet-spend")).toContainText("$50.00");
  });

  test("should render the 12-agent roster", async ({ page }) => {
    for (const short of [
      "drift",
      "patch",
      "compliance",
      "incident",
      "dr",
      "cost",
      "security",
      "image",
      "sop",
      "adapter",
      "certificate",
      "vulnerability",
    ]) {
      await expect(page.getByTestId(`agent-${short}`)).toBeVisible({
        timeout: WIDGET_TIMEOUT,
      });
    }
  });

  test("should display read-only autonomy state on the roster", async ({ page }) => {
    // Compliance is seeded "auto" in the Phase A autonomy map.
    const complianceRow = page.getByTestId("agent-compliance");
    await expect(complianceRow.getByText("auto", { exact: true })).toBeVisible();
    // Drift is "semi".
    const driftRow = page.getByTestId("agent-drift");
    await expect(driftRow.getByText("semi", { exact: true })).toBeVisible();
  });

  test("should render the activity stream with at least 3 seeded tool invocations", async ({ page }) => {
    // From seedMissionControl across 3 agents (vulnerability, drift, certificate).
    for (const tool of [
      "calculate_blast_radius",
      "analyze_drift",
      "propose_cert_rotation",
    ]) {
      await expect(page.getByTestId(`activity-${tool}`)).toBeVisible({
        timeout: WIDGET_TIMEOUT,
      });
    }
  });

  test("should show the seeded pending decision with OPA result and quality score", async ({ page }) => {
    const main = page.getByRole("main");
    // The seeded pending task's intent.
    await expect(
      main.getByText("Patch CVE-2024-3094 (xz backdoor) on production assets"),
    ).toBeVisible({ timeout: WIDGET_TIMEOUT });
    // OPA pass + quality 87.
    await expect(page.getByTestId("pending-opa")).toHaveText("pass");
    await expect(page.getByTestId("pending-quality")).toHaveText("87/100");
  });

  test("should render a collapsed, read-only conversation dock", async ({ page }) => {
    const input = page.getByTestId("conversation-input");
    await expect(input).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(input).toBeDisabled();
    await expect(input).toHaveAttribute(
      "placeholder",
      /Ask Mission Control/i,
    );
  });
});
