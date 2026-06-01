# Hero Demo Runbook — CVE-to-Remediation

**Audience:** anyone delivering a 90-second QuantumSRE live demo or recording.
**Purpose:** make the demo repeatable, beat by beat, against the deterministic E2E seed.
**Last updated:** PR #17 (`ai/006-hero-demo-polish`).

This document is the *spec* for the demo. It does NOT take screenshots — those are a recording-day task. If a beat below doesn't match what's on screen, the seed is dirty or the product drifted; fix the seed/product, not the doc.

---

## Pre-flight

Run these once before each demo session. Total time: ~30 seconds if the stack is already up, ~2 minutes cold.

```bash
# 1. Bring up the compose stack (postgres + redis + temporal + opa + api + orchestrator).
make dev

# 2. Make sure orchestrator is on the stub LLM provider (deterministic responses,
#    no external calls, no LLM tokens burned).
RF_LLM_PROVIDER=stub docker compose up -d --build --no-deps orchestrator

# 3. Reset to the known seeded fixture.
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

**Expected output (anything else → reset):**

```json
{
  "pending": 1,
  "today":   6,
  "spend":   182,
  "agents":  {"total": 12, "working": 1, "idle": 11, "blocked": 0}
}
```

```bash
docker logs qlrf-orchestrator --tail 100 | grep "STUB PROVIDER ENABLED"
```

**Expected:** one WARN line that includes `STUB PROVIDER ENABLED — provider=stub model=stub-canned deterministic=true external_calls=false`. If this is missing, the orchestrator booted with a real provider — restart with `RF_LLM_PROVIDER=stub docker compose up -d --build --no-deps orchestrator` and try again.

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

**Action:** point at the seeded pending decision card in the right column. Same intent text as the dock thread.

**Narration:**
> "And here it is on the pending decisions rail. Same intent. Quality 87. OPA policy: pass. Blast radius: 4 assets in production. This isn't a recommendation — it's a *plan* that already passed our schema validator, our policy engine, and our quality scorer. Waiting on me."

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

## What this demonstrates

- **Governed cross-domain remediation.** Drift, CVE, certificates, DR — the same approval loop, the same evidence trail, regardless of domain.
- **No rip-and-replace.** The connectors path lands next; the simulator is its architectural twin. Same data model, same UI, same audit ledger.
- **Deterministic evidence.** Every transition has a microsecond timestamp and a `_simulated: true` marker. Real runs will lose that marker; everything else stays identical.
- **Read-only by default.** Even the simulator only ever uses `plan_only` risk-level tool invocations. State-change tools need real connectors AND explicit two-approver workflows — out of scope for the simulator path.

---

## What NOT to demo (yet)

These are **deliberately** out of the hero flow. If a viewer asks, say: "next phase — we want the simulated loop to be rock-solid first."

| Surface | Why it's out |
|---------|--------------|
| Real cloud SDK calls | No connectors live yet. The simulator IS the connector path's twin. |
| Live LLM | Demo runs on stub. Azure Anthropic works but is opt-in via `RF_LLM_PROVIDER=azure_anthropic` + an API key in `.env`. |
| Approve buttons across orgs | Multi-tenant isolation works (verified by unit tests) but not part of the 90s flow. |
| Modify button on pending cards | Disabled — needs a plan-payload editor. Deferred to a later PR. |
| `compliance_evidence` integration | The audit_log is the ledger today. Tying simulated runs into the compliance page is a follow-up PR. |
| Autonomy mode editing | Read-only display only. Writing autonomy modes is Phase C. |
| Multi-turn conversation refinement | The 60-minute window appends to the same thread, but the dock doesn't visualise multi-turn refinement specially yet. |

---

## Failure modes

If you hit any of these during a demo, the seed is dirty or the stack is off. Reset and retry.

| Symptom | Cause | Fix |
|---------|-------|-----|
| Pending count ≠ 1 on landing | A previous demo approved/rejected the seeded plan | `go run ./scripts/seed-e2e-data/` and refresh |
| Dock thread shows "Analyze drift…" instead of "Patch CVE-2024-3094…" | You're on a pre-PR-#17 seed | Pull latest, re-run `go run ./scripts/seed-e2e-data/` |
| `tool_invocations_today` shows 0 | The seed was last run >24h ago AND nothing new ran today | Re-run `go run ./scripts/seed-e2e-data/` — it refreshes timestamps |
| `llm_spend_today_cents` shows 0 | Same as above | Same fix |
| Run card doesn't appear after Approve | Orchestrator not on stub | `docker logs qlrf-orchestrator \| grep "STUB PROVIDER ENABLED"` should print; if not, `RF_LLM_PROVIDER=stub docker compose up -d --build --no-deps orchestrator` |
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
  'e1000000-0000-0000-0000-000000000004'
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

-- Restore the seeded pending decision if it was approved or rejected.
UPDATE ai_plans
SET state = 'awaiting_approval', approved_by = NULL, approved_at = NULL
WHERE id = 'e2000000-0000-0000-0000-000000000001';
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
- [ ] **0:15** Point at pending decision — same intent, quality 87, OPA pass.
- [ ] **0:25** Point at activity stream — three read-only invocations.
- [ ] **0:30** Click **Approve**.
- [ ] **0:35** Watch the new run card animate queued → executing → completed.
- [ ] **0:50** Click the completed run card. Read the 6-entry timeline.
- [ ] **1:15** Submit risky prompt; click **Reject**; point at the system message.
- [ ] **1:25** Pan back to fleet status. Close.

If any beat takes more than 10 seconds, you're over-explaining — let the UI carry it.
