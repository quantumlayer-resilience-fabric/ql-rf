# Hero Demo Runbook — CVE-to-Remediation

**Audience:** anyone delivering a 90-second QuantumSRE live demo or recording.
**Purpose:** make the demo repeatable, beat by beat, against the deterministic E2E seed.
**Last updated:** PR #45 (`ai/034-hero-demo-agent-layer-beat`) — the
connector arc is now complete across all 5 clouds (AWS + Azure + GCP +
vSphere + K8s, PRs #19–#40). 5 platforms × 3 risk tiers = 15 cloud
tools. PR #45 adds a callout to BEAT 3 for the now-visible
patch-agent recommendation layer (PRs #36/#43/#44) — every pending
patch plan now carries an "Agent recommends" section listing the
right per-cloud tool tier for each platform in scope.

This document is the *spec* for the demo. It does NOT take screenshots — those are a recording-day task. If a beat below doesn't match what's on screen, the seed is dirty or the product drifted; fix the seed/product, not the doc.

---

## Pre-flight

Run these once before each demo session. Total time: ~30 seconds if the stack is already up, ~2 minutes cold.

> **The seeder alone is not a reset.** It's idempotent on its own deterministic IDs but does NOT delete rows from earlier dry-run sessions (extra pending tasks, extra conversations, extra runs). If the stack has been used at all today, you MUST run the [reset SQL block](#one-shot-reset-sql) BEFORE the seeder. The pre-flight below does this in order.

```bash
# 1. Bring up the compose stack (postgres + redis + temporal + opa + api + orchestrator).
make dev

# 2. Make sure orchestrator is on the stub LLM provider (deterministic responses,
#    no external calls, no LLM tokens burned).
RF_LLM_PROVIDER=stub docker compose up -d --build --no-deps orchestrator

# 3a. Clean out any debris from prior dry-runs (see "One-shot reset SQL" below).
#     Required even on a fresh boot if the DB volume persisted.
#     Paste the reset SQL block into psql, then continue.

# 3b. Re-seed to the known fixture (idempotent on the seeded IDs; refreshes timestamps).
go run ./scripts/seed-e2e-data/

# 4. Start the frontend.
cd ui/control-tower && npm run dev
# (waits ~5s, then localhost:3000 is ready)
```

### Is it clean? (one-shot health check)

```bash
curl -s -H "Authorization: Bearer dev-token" \
  http://localhost:8083/api/v1/ai/fleet/status \
  | jq '{
      pending: (.pending_approvals | length),
      today:   .tool_invocations_today,
      spend:   .llm_spend_today_cents,
      agents:  .agents
  }'
```

**Expected output (anything else → run the reset SQL block, then re-seed):**

```json
{
  "pending": 2,
  "today":   6,
  "spend":   182,
  "agents":  {"total": 12, "working": 1, "idle": 11, "blocked": 0}
}
```

> `pending: 2` is the post-PR-#22 expected. The CVE plan is the primary hero
> beat; the second pending is the "awaiting second approval" SSM-live plan
> that drives the bonus connector arc (see the [Connector arc bonus](#connector-arc-bonus--30s-add-on-to-the-hero-beats) below).

If `pending > 1`, your DB has demo debris from prior runs — apply the [reset SQL](#one-shot-reset-sql) and re-seed.

### Is the stub provider actually serving? (live check, restart-safe)

Long-running containers' boot WARN scrolls off `docker logs --tail`, so don't rely on the startup line. Instead, hit the orchestrator's runtime endpoint and infer from the response shape — only the stub returns `model=stub-canned`:

```bash
docker exec qlrf-orchestrator env | grep RF_LLM_PROVIDER
# expect: RF_LLM_PROVIDER=stub
```

**OR** if you have time to restart the orchestrator (gives you a fresh boot log):

```bash
RF_LLM_PROVIDER=stub docker compose up -d --build --no-deps orchestrator
sleep 4
docker logs qlrf-orchestrator --since 30s | grep "STUB PROVIDER ENABLED"
```

Expected on restart: one WARN line `STUB PROVIDER ENABLED — provider=stub model=stub-canned deterministic=true external_calls=false`. If missing, the orchestrator booted with a real provider — the compose env or `.env` has a non-stub `RF_LLM_PROVIDER`.

---

## The 9 beats

Total runtime: **~90 seconds**. The recording / live perf should land within ±5s.

Each beat: timing · screen position · action · narration.

---

### BEAT 1 — Landing · 0:00–0:05 · Center

**Action:** open `http://localhost:3000/ai` in a fresh browser tab. Pause on the dashboard.

**Narration:**
> "QuantumSRE — Mission Control. Twelve specialist agents, fully governed. One working right now, eleven idle. $1.82 in LLM spend today against a $50 budget. Six tool invocations on the books."

*(All those numbers are visible in the fleet status bar across the top.)*

---

### BEAT 2 — Memory · 0:05–0:15 · Bottom of page (Conversation dock)

**Action:** point at the conversation dock thread. The CVE patch conversation is already there.

**Narration:**
> "Mission Control already remembers our last request — Patch CVE-2024-3094, the xz backdoor. Five minutes ago we asked it to handle this. It said: 'Drafted plan-only patch_rollout. Risk: high, HITL required. Quality 87 out of 100. Awaiting your approval.' That answer was synthesised from the validated plan, not free-form LLM text — every word here is auditable."

---

### BEAT 3 — Plan · 0:15–0:25 · Right rail (Pending decisions)

**Action:** point at the seeded pending decision card in the right column. Same intent text as the dock thread. Then trace down to the "Agent recommends" section.

**Narration:**
> "And here it is on the pending decisions rail. Same intent. Quality 87. OPA policy: pass. Blast radius: 4 assets in production. This isn't a recommendation — it's a *plan* that already passed our schema validator, our policy engine, and our quality scorer. Waiting on me.
>
> See the 'Agent recommends' panel? The patch agent's already picked the right tool per platform: SSM for AWS, Run Command for Azure, server-side apply for Kubernetes. It picks dry-run tools in production — live execution is a separate two-approver workflow. That mapping isn't hardcoded in the prompt; it's read from a single platform → tool catalog. New clouds plug in without re-tuning the agent."

---

### BEAT 4 — Evidence so far · 0:25–0:30 · Center (Activity stream)

**Action:** point at the activity stream entries — the seeded tool invocations.

**Narration:**
> "Look at the activity stream. The plan didn't come from nothing — three read-only tool calls fed it: list_cve_alerts, calculate_blast_radius, query_assets. All read-only. Nothing in our infrastructure has been touched."

---

### BEAT 5 — Approve · 0:30–0:35 · Right rail

**Action:** click the **Approve** button on the seeded pending card.

**Narration:**
> "I'm approving simulated execution. In production this would trigger the connector path. Today it triggers the simulator — same data model, same governance, no real cloud calls."

*(Pending count drops 1 → 0. A new card appears in the "Recent runs" rail below.)*

---

### BEAT 6 — Watch it run · 0:35–0:50 · Right rail (Recent runs)

**Action:** point at the new run card. Don't click yet — let it animate.

**Narration:**
> "Watch the new run card. Queued. Now executing — phase canary. Now monitor. Now full_rollout. Now completed. Three phases. Three minutes in real life — three seconds here, because nothing real is being touched. The simulator runs to completion at one phase per second with a thirty-second safety cap."

*(By ~0:50, the card reads `completed · 3 phases · just now`.)*

---

### BEAT 7 — The ledger · 0:50–1:15 · Right rail (expand the new run card)

**Action:** click the new run card to expand it. Read the audit timeline aloud.

**Narration:**
> "Here is the evidence ledger. Click the card and it opens in place. Six events, each timestamped to the millisecond. *Approved by Mission Control Dev User.* *Started.* *Phase canary completed — generate_patch_plan tool ran, plan-only risk, 295 milliseconds.* *Phase monitor completed — simulate_rollout.* *Phase full_rollout completed — propose_rollout.* *Simulated complete — three invocations, no real changes.*
>
> Every entry is tagged `_simulated: true` in the database. A regulator asking 'what did your AI agent do at 8:32 PM on the third?' gets this answer, not a paragraph of LLM prose. This is the artefact compliance needs."

---

### BEAT 8 — Refusal · 1:15–1:25 · Bottom dock + right rail

**Action:** type a fresh prompt into the conversation dock, submit, then click **Reject** on the resulting pending card.

**Suggested prompt:** `Failover the production database to the DR site right now.`

**Narration:**
> "Same governance works for refusal. I submit a risky request — production DB failover — and a new pending decision appears with quality and OPA results attached. I reject it. The plan moves to rejected, the dock thread records the reason, and no ai_run row is ever created. No phantom 'we considered doing this' artefact. Refusal is first-class evidence."

---

### BEAT 9 — Close · 1:25–1:30 · Pan back to fleet status

**Action:** scroll back up to the dashboard. Let the fleet status numbers settle.

**Narration:**
> "That's the loop. Intent. Memory. Plan. Approval. Simulated execution. Evidence. No cloud was touched. No LLM tokens were spent — the orchestrator was running our deterministic stub provider the whole time. Real connectors plug in next, with exactly this governance discipline. Until then, this is how we develop and demo the safe path end-to-end."

---

## Connector arc bonus — 30s add-on to the hero beats

Use this only if the room asks "but does it actually do real cloud stuff?" or you've got time to spare. The 90-second hero demo above is complete on its own.

This bonus arc runs entirely from the **Real tools** card in the right rail and the **second pending decision**, with no extra setup beyond the standard pre-flight.

### Bonus BEAT A — Read-only real cloud · ~10s · Right rail (Real tools)

Click **Invoke** on `query_aws_instances`.

Say: *"This is a real AWS call — `EC2.DescribeInstances`. The orchestrator runs with `RF_CONNECTORS_AWS_FALLBACK_TO_MOCK=true` in dev, so we're hitting the deterministic mock client. In production, the same path takes real credentials. Audit row goes into the same `ai_tool_invocations` table as the simulator output, distinguishable by JSONB markers. Same surface for Azure (`query_azure_vms`), GCP (`query_gcp_instances`), vSphere (`query_vsphere_vms`), and Kubernetes (`query_pods`) — point at those rows in the Real tools card to show the five clouds side-by-side."*

Within a second, the activity stream gains a `query_aws_instances` row with `risk: read_only` and **no** `_simulated` marker.

### Bonus BEAT B — Dry-run state-change · ~10s · Right rail (Real tools)

Click **Dry-run** on `ssm_send_patch_command`.

Say: *"State-change tools can't be invoked from the read-only endpoint — see, they have a Dry-run button instead. This builds an `AWS-RunPatchBaseline` command plan but never calls `ssm:SendCommand`. The same exists for Azure (`azure_run_command`), GCP (`gcp_os_config_patch`), vSphere (`vsphere_run_guest_program`), and Kubernetes (`k8s_apply`). A Go structural-safety test enforces, per cloud, that the live state-change SDK call is reachable from exactly one file — `live_ssm_client.go`, `live_azure_runcommand_client.go`, `live_gcp_patch_client.go`, `live_vsphere_guest_ops_client.go`, `live_k8s_apply_client.go`. Auditors can grep for the SDK call and see exactly one match per cloud."*

Activity stream picks up `ssm_send_patch_command` with `risk: state_change_prod`. The audit row's `parameters @> '{"dry_run": true}'` distinguishes it from any live invocation.

### Bonus BEAT C — Two-approver awaiting state · ~10s · Right rail (Pending decisions, plan #2)

Point at the **second pending card** — the one with the amber "Awaiting second approval" badge.

Say: *"This is a state-change-prod plan. The first approver has already clicked Approve; the card no longer offers an Approve button. Instead it shows who first-approved, that two approvers are required, and a Co-approve button. The OPA policy enforces this in code: `state_change_prod` requires both `approved_by` and a distinct `second_approver`. In production with the live tool registered (`RF_CONNECTORS_AWS_ALLOW_LIVE_PATCH=true`, or the Azure / GCP / vSphere / K8s equivalent — e.g. `RF_CONNECTORS_K8S_ALLOW_LIVE_APPLY=true`), clicking Co-approve would fire a real cloud mutation."*

Do NOT click Co-approve during a demo — it will succeed in dev (because the live tools aren't registered, no real call fires), but the visual feedback isn't worth the cognitive overhead in 30 seconds. Leave the card in its awaiting state.

### Bonus close — the matrix, one sentence

*"Five clouds × three risk tiers. Read-only real call → state-change dry-run → state-change live with two-approver gate. Fifteen cloud tools, one audit-by-grep discipline, same `ai_tool_invocations` table for everything. Production deployments opt into each cloud's live mode independently via env vars; CI keeps every live opt-in off."*

### The full connector matrix (for the slide deck)

| Cloud | Read-only | Dry-run | Live + 2-approver |
|-------|-----------|---------|-------------------|
| AWS | `query_aws_instances` (PR #19) | `ssm_send_patch_command` (PR #20) | `ssm_send_patch_command_live` (PR #21) |
| Azure | `query_azure_vms` (PR #26) | `azure_run_command` (PR #27) | `azure_run_command_live` (PR #28) |
| GCP | `query_gcp_instances` (PR #29) | `gcp_os_config_patch` (PR #30) | `gcp_os_config_patch_live` (PR #31) |
| vSphere | `query_vsphere_vms` (PR #33) | `vsphere_run_guest_program` (PR #34) | `vsphere_run_guest_program_live` (PR #35) |
| Kubernetes | `query_pods` (PR #38) | `k8s_apply` (PR #39) | `k8s_apply_live` (PR #40) |

All fifteen tools share:
- `ai_tool_invocations` audit table
- The `state_change_prod` two-approver workflow (PR #21's OPA + co-approve handler) — works for every cloud
- The four-gate live boot pattern (env opt-in + mock-conflict refusal + per-target whitelist + 2-approver)
- The `compliance_evidence` attestation path (PR #24) — every dry-run and live invocation produces an evidence row when a `tool_compliance_mappings` row exists (seed maps the state-change tools to CIS-1.4)
- The "Awaiting second approval" notifier ping (PR #25) — fires regardless of which cloud's live tool is in the plan

---

## What this demonstrates

- **Governed cross-domain remediation.** Drift, CVE, certificates, DR — the same approval loop, the same evidence trail, regardless of domain.
- **Multi-cloud connector arc.** PRs #19–#40 ship a complete matrix: 5 clouds (AWS / Azure / GCP / vSphere / Kubernetes) × 3 risk tiers (read-only / dry-run / live + 2-approver) = 15 cloud tools. Same UI, same audit table, same OPA policy, same compliance evidence emission.
- **Platform-aware patch agent, visibly on screen.** PR #36 wires the patch agent into a single platform → tool-tier catalog (`patch_platform_catalog.go`). PR #43 seeds the demo plans with the agent's per-phase `recommended_tools`. PR #44 surfaces those on the pending decision card as an "Agent recommends" section — operators see the per-cloud tool choice (SSM for AWS, Run Command for Azure, server-side apply for K8s) before they click Approve. The agent itself never invokes a state-change tool; it only names them. Live execution remains gated by the two-approver workflow.
- **Four-way audit distinction.** Synthetic (B.3 `_simulated:true`), real read-only (PR #19/#26/#29/#33/#38), state-change dry-run (PR #20/#27/#30/#34/#39 `dry_run:true`), live state-change (PR #21/#28/#31/#35/#40 `dry_run:false` + `risk='state_change_prod'`). One SQL view classifies the entire ledger across all clouds.
- **Compliance attestations are automatic.** PR #24's `tool_compliance_mappings` ties every real invocation to a compliance control; the demo dashboard's CIS-1.4 evidence panel populates from real and dry-run tool fires alike.
- **Deterministic evidence.** Every transition has a microsecond timestamp and a JSONB marker. The marker is the only difference between the simulator and the live path.
- **Read-only by default.** Live state-change tools are unregistered unless explicitly opted in (per cloud). CI is locked off; production deployments flip env vars per platform.
- **SDK isolation by file.** For each cloud, exactly one file in the tools package can construct the state-change SDK client. Structural Go tests enforce this — auditors grep one filename per cloud.

---

## What NOT to demo (yet)

These are **deliberately** out of the hero flow. If a viewer asks, say: "next phase — we want the simulated loop to be rock-solid first."

| Surface | Why it's out |
|---------|--------------|
| Live LLM | Demo runs on stub. Azure Anthropic works but is opt-in via `RF_LLM_PROVIDER=azure_anthropic` + an API key in `.env`. |
| Clicking Co-approve in the bonus arc | The plan goes to `approved` state and the executor fires the simulated tool path (because the live SSM tool isn't registered without `RF_CONNECTORS_AWS_ALLOW_LIVE_PATCH=true`). The visual outcome is anticlimactic; better to describe the gate than to trigger it. |
| Real live cloud calls (any cloud) | The PR #21 / #28 / #31 / #35 / #40 code paths work, but require production credentials + per-target whitelist + the env opt-in. Out of scope for a 90s + 30s demo. Show the dry-run instead. |
| Approve buttons across orgs | Multi-tenant isolation works (verified by unit tests) but not part of the 90s flow. |
| Modify button on pending cards | Disabled — needs a plan-payload editor. Deferred to a later PR. |
| Autonomy mode editing | Read-only display only. Writing autonomy modes is Phase C. |
| Multi-turn conversation refinement | The 60-minute window appends to the same thread, but the dock doesn't visualise multi-turn refinement specially yet. |

---

## Failure modes

If you hit any of these during a demo, the seed is dirty or the stack is off. Reset and retry.

| Symptom | Cause | Fix |
|---------|-------|-----|
| Pending count > 2 on landing | DB has debris from prior dry-runs (the seeder alone won't clean this) | Run the [reset SQL](#one-shot-reset-sql), then re-run the seeder |
| Pending count < 2 on landing | One of the seeded pending plans was approved/rejected/co-approved | Run the [reset SQL](#one-shot-reset-sql) (it restores both `awaiting_approval` plans), then re-seed |
| Bonus arc: second pending card has no "Awaiting second approval" badge | The `approved_by` field on plan #5 got cleared | Re-run the seeder; if the badge is still missing, check `psql … SELECT id, approved_by FROM ai_plans WHERE id='e2000000-0000-0000-0000-000000000005'` — it should be `e0000000-0000-0000-0000-0000000000aa` |
| Dock thread shows "Analyze drift…" instead of "Patch CVE-2024-3094…" | You're on a pre-PR-#17 seed | Pull latest, run the reset SQL + seeder |
| `tool_invocations_today` shows 0 | The seed was last run >24h ago AND nothing new ran today | Re-run `go run ./scripts/seed-e2e-data/` — it refreshes timestamps via UPDATE on conflict |
| `llm_spend_today_cents` shows 0 | Same as above | Same fix |
| Run card doesn't appear after Approve | Orchestrator not on stub | Check `docker exec qlrf-orchestrator env \| grep RF_LLM_PROVIDER` (should be `stub`); if not, `RF_LLM_PROVIDER=stub docker compose up -d --build --no-deps orchestrator` |
| Run card flips to `completed` instantly, no animation | The fleet status polling interval missed the window (rare) | Refresh the page right after approve to re-trigger the 2s in-flight poll |
| Activity stream is empty | Org ID mismatch | Check `curl /fleet/status` matches expected output above |

---

## One-shot reset SQL

Use this if you've been clicking around and the seed isn't worth restarting docker for. Connect via `psql` and run:

```sql
-- Reset Mission Control state without touching schema or seeded reference rows.
-- After running, re-run: go run ./scripts/seed-e2e-data/
DELETE FROM ai_tasks
WHERE id != ALL(ARRAY[
  'e1000000-0000-0000-0000-000000000001',
  'e1000000-0000-0000-0000-000000000002',
  'e1000000-0000-0000-0000-000000000003',
  'e1000000-0000-0000-0000-000000000004',
  'e1000000-0000-0000-0000-000000000005'  -- PR #22: SSM live two-approver fixture
]::uuid[]);

DELETE FROM ai_conversation_messages
WHERE id != ALL(ARRAY[
  'e7000000-0000-0000-0000-000000000001',
  'e7000000-0000-0000-0000-000000000002',
  'e7000000-0000-0000-0000-000000000003',
  'e7000000-0000-0000-0000-000000000004'
]::uuid[]);

DELETE FROM ai_conversations
WHERE id != ALL(ARRAY[
  'e6000000-0000-0000-0000-000000000001',
  'e6000000-0000-0000-0000-000000000002'
]::uuid[]);

DELETE FROM ai_runs
WHERE id != ALL(ARRAY[
  'e3000000-0000-0000-0000-000000000001',
  'e3000000-0000-0000-0000-000000000002'
]::uuid[]);

DELETE FROM ai_tool_invocations
WHERE id != ALL(ARRAY[
  'e4000000-0000-0000-0000-000000000001',
  'e4000000-0000-0000-0000-000000000002',
  'e4000000-0000-0000-0000-000000000003',
  'e4000000-0000-0000-0000-000000000004',
  'e4000000-0000-0000-0000-000000000005',
  'e4000000-0000-0000-0000-000000000006'
]::uuid[]);

-- Restore the original seeded pending decision (CVE plan, hero beat) to
-- awaiting_approval with both approver columns cleared.
UPDATE ai_plans
SET state = 'awaiting_approval', approved_by = NULL, approved_at = NULL,
    second_approver = NULL, second_approved_at = NULL
WHERE id = 'e2000000-0000-0000-0000-000000000001';

-- PR #22: restore the SSM-live two-approver fixture to
-- awaiting_approval with the first approver pre-set (the value the seed
-- writes). second_approver MUST be NULL so the UI renders the
-- "Awaiting second approval" badge for the bonus arc.
UPDATE ai_plans
SET state = 'awaiting_approval',
    approved_by = 'e0000000-0000-0000-0000-0000000000aa',
    approved_at = NOW(),
    second_approver = NULL,
    second_approved_at = NULL
WHERE id = 'e2000000-0000-0000-0000-000000000005';
```

**Then re-run the seeder** to refresh timestamps and the conversation breadcrumbs:

```bash
go run ./scripts/seed-e2e-data/
```

> **Note:** If a future migration adds tables that hold demo state (`ai_runs_execution_log`, etc.), this reset block needs a `DELETE` for each. The convention is: anything Mission Control reads, this reset cleans.

---

## Beat-by-beat checklist (printable)

Tear this off, tape it to the second monitor:

- [ ] **0:00** Open `/ai`. Read fleet bar metrics.
- [ ] **0:05** Point at dock thread — CVE conversation.
- [ ] **0:15** Point at pending decision — same intent, quality 87, OPA pass. Trace down to the "Agent recommends" section (AWS / Azure / K8s rows).
- [ ] **0:25** Point at activity stream — three read-only invocations.
- [ ] **0:30** Click **Approve** on the **CVE-2024-3094 card** (the one without the amber "Awaiting second approval" badge).
- [ ] **0:35** Watch the new run card animate queued → executing → completed.
- [ ] **0:50** Click the completed run card. Read the 6-entry timeline.
- [ ] **1:15** Submit risky prompt; click **Reject**; point at the system message.
- [ ] **1:25** Pan back to fleet status. Close.

**Bonus arc (optional, 30s, multi-cloud):**

- [ ] **+0:00** Right rail → Real tools → click **Invoke** on `query_aws_instances`. Activity stream gains a `read_only` row. Point at the `query_azure_vms` + `query_gcp_instances` + `query_vsphere_vms` + `query_pods` rows to call out the five-cloud parity.
- [ ] **+0:10** Right rail → click **Dry-run** on `ssm_send_patch_command`. Activity stream gains a `state_change_prod` row with no SendCommand fired. Point at the Azure (`azure_run_command`), GCP (`gcp_os_config_patch`), vSphere (`vsphere_run_guest_program`), and Kubernetes (`k8s_apply`) rows — same Dry-run button.
- [ ] **+0:20** Right rail → point at the second pending card (amber badge). Read: "Awaiting second approval, 1st: e0000000…, Co-approve required." Do NOT click.

If any beat takes more than 10 seconds, you're over-explaining — let the UI carry it.
