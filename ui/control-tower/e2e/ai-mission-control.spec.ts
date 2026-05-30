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
    // Scope to the pending rail — the seeded CVE text now also appears in the
    // activity stream (Phase B.2 conversation event), so a `main.getByText(...)`
    // would resolve to two elements.
    const pendingCard = page.locator('[data-testid^="pending-e2"]').first();
    await expect(pendingCard).toContainText(
      "Patch CVE-2024-3094 (xz backdoor) on production assets",
      { timeout: WIDGET_TIMEOUT },
    );
    // OPA pass + quality 87.
    await expect(page.getByTestId("pending-opa")).toHaveText("pass");
    await expect(page.getByTestId("pending-quality")).toHaveText("87/100");
  });

  test("should render an enabled conversation dock", async ({ page }) => {
    // Phase B.1: the dock is enabled and accepts submissions via the stub
    // LLM provider. Phase A's "(read-only preview in Phase A)" suffix is gone.
    const input = page.getByTestId("conversation-input");
    await expect(input).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(input).toBeEnabled();
    await expect(input).toHaveAttribute("placeholder", /Ask Mission Control/i);
    // Submit button is present but disabled until the input has content.
    const submit = page.getByTestId("conversation-submit");
    await expect(submit).toBeVisible();
    await expect(submit).toBeDisabled();
  });
});

test.describe("Mission Control (AI-003 Phase B.2 — conversation thread)", () => {
  test("submitting a prompt appends to the dock thread", async ({ page }) => {
    await page.goto("/ai");
    await expect(
      page.getByRole("heading", { name: "Mission Control", exact: true }),
    ).toBeVisible({ timeout: WIDGET_TIMEOUT });

    // Unique per-run prompt so successive local runs don't collide on text.
    const prompt = `Patch CVE-2026-1234 from B.2 thread (e2e ${Date.now()})`;

    const input = page.getByTestId("conversation-input");
    await input.fill(prompt);
    await page.getByTestId("conversation-submit").click();

    // The dock thread renders the active conversation. After a submit:
    //   * the user message bubble shows the submitted text;
    //   * the assistant bubble shows the server-synthesised summary, which
    //     for a CVE-keyword prompt always references "patch_rollout" and ends
    //     with "Awaiting your approval."
    const thread = page.getByTestId("conversation-thread");
    await expect(thread).toBeVisible({ timeout: WIDGET_TIMEOUT });
    await expect(thread.getByText(prompt).first()).toBeVisible({ timeout: 20_000 });

    const assistantBubble = thread.locator('[data-role="assistant"]').last();
    await expect(assistantBubble).toContainText("patch_rollout", { timeout: 20_000 });
    await expect(assistantBubble).toContainText("Awaiting your approval", { timeout: 20_000 });
  });

  test("submitting a prompt surfaces a conversation event in the activity stream", async ({ page }) => {
    await page.goto("/ai");
    await expect(
      page.getByRole("heading", { name: "Mission Control", exact: true }),
    ).toBeVisible({ timeout: WIDGET_TIMEOUT });

    const prompt = `Drift check from B.2 stream (e2e ${Date.now()})`;
    await page.getByTestId("conversation-input").fill(prompt);
    await page.getByTestId("conversation-submit").click();

    // Activity stream entries for conversation events have
    // data-testid="activity-conversation-<message_id>". Message id isn't
    // known up front, so match by partial selector + visible preview text.
    const main = page.getByRole("main");
    await expect(
      main.locator('[data-testid^="activity-conversation-"]').first(),
    ).toContainText(prompt.slice(0, 60), { timeout: 20_000 });
  });
});

test.describe("Mission Control (AI-002 Phase B.1 — conversation submission)", () => {
  test("submitting a prompt creates a new pending decision", async ({ page }) => {
    await page.goto("/ai");
    await expect(
      page.getByRole("heading", { name: "Mission Control", exact: true }),
    ).toBeVisible({ timeout: WIDGET_TIMEOUT });

    // Read the current pending count rather than hard-coding 1. The seeded
    // fixture starts at 1, but a persistent local stack may carry debris from
    // a previous run; either way the assertion below is "+1", which catches
    // the failure mode "count moved for some other reason" while staying
    // robust to baseline drift.
    const pendingEl = page.getByTestId("fleet-pending");
    const initialText = (await pendingEl.textContent()) ?? "";
    const initialCount = parseInt(initialText.match(/\d+/)?.[0] ?? "0", 10);
    const expectedCount = String(initialCount + 1);

    // Unique per-run prompt — safe whether CI re-seeds (yes) or a developer
    // runs the spec multiple times locally against a persistent stack.
    const prompt = `Patch CVE-2026-1234 safely from Mission Control (e2e ${Date.now()})`;

    const input = page.getByTestId("conversation-input");
    await input.fill(prompt);
    await page.getByTestId("conversation-submit").click();

    // The stub provider short-circuits to plan-only on the orchestrator side,
    // so the new ai_plan lands in awaiting_approval and surfaces on the
    // pending decisions rail. Fleet status invalidation makes this near-instant;
    // 20s is the safety net for the 15s refetch interval.
    await expect(pendingEl).toContainText(expectedCount, { timeout: 20_000 });

    // Beyond the count, assert the new pending card actually shows the
    // submitted text — catches the failure mode "count moved for some other
    // reason."
    await expect(
      page.getByRole("main").getByText(prompt).first(),
    ).toBeVisible({ timeout: WIDGET_TIMEOUT });
  });
});
