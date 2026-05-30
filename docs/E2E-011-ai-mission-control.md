# AI-001 / E2E-011 — Mission Control

**Status:** Decided. Plan locked. v2 (2026-05-30) supersedes v1.
**Canonical name:** Mission Control.
**Subtitle:** *Governed command for infrastructure agents.*
**Approach:** Redesign-first. Phase A only. Read-only. Deterministic. E2E-protected.
**One-line thesis:** QuantumSRE's AI surface is the human command surface for a governed agent fleet — the visible face of QuantumFabric inside QuantumSRE. It is not a chat page.

---

## 1. The decision (one paragraph)

We will build **Mission Control Phase A** — a read-only operational control surface that exposes the agent fleet, the lifecycle of every plan, and the policy gates around every action. We will not test the existing `/ai` chat page; we will replace it. Phase A makes no live LLM calls in CI, executes no cloud tools, mutates no approvals, and does not depend on Temporal. The E2E-011 PR seeds deterministic AI lifecycle state (`ai_tasks` / `ai_plans` / `ai_runs` / `ai_tool_invocations`) and asserts the Mission Control surface renders that state honestly. Mutation, streaming, and conversational state come in later phases.

---

## 2. The market correction (sharper claim)

The previous draft framed the wedge as *"competitors answer questions; we act."* That claim is becoming false. The serious vendors are all converging on agentic action:

- **Datadog Bits AI SRE** — deeper reasoning, faster incident investigation, automated remediation workflows initiated from the assistant (2026 positioning).
- **ServiceNow Action Fabric / AI Control Tower** — explicit "system of action" framing for governed enterprise agents (2026).
- **Dynatrace + ServiceNow** — partnership framed around turning real-time observability into trustworthy autonomous action.
- **Microsoft / New Relic** — agent observability is being elevated to a category; New Relic frames agentic workflows as "a black box without monitoring."

So the correct, defensible wedge for QuantumSRE / Mission Control is **not** "we have AI." It is:

> Competitors are adding agents to their existing worlds — observability, ITSM, security, search, dev tools. **QuantumSRE exposes a governed remediation fabric across the whole infrastructure lifecycle.**

Our differentiator is **cross-domain coordination + governance**, not the existence of agents. We connect:

drift · CVE · SBOM · golden image · certificates · compliance · risk · approval · policy · execution · evidence

through **one operational surface**. No competitor does all of those together with OPA gates, Temporal-durable execution, autonomy modes, and per-org cost tracking in one box.

---

## 3. The three-layer model (what we actually have)

| Layer | What it answers | Where we are |
|---|---|---|
| **1. Dashboards** — Overview, Sites, Drift, Images, Risk, Vulnerabilities, Certificates, Compliance, SBOM | *What is risky?* | Built, protected by blocking E2E (38 specs). Good. **Not unique.** |
| **2. Remediation engine** — 12 agents, 44 tools, plans, OPA gates, HITL approval, Temporal workflows, autonomy modes, audit trail, LLM cost tracking | *What should we do, who/what will do it, is it safe?* | Built, ~5,000 LOC in `services/orchestrator/`. **The differentiated layer.** |
| **3. Mission Control** — *the surface* | *What is the fleet doing? what needs approval? what is checked? what is blocked? what is safe to automate? what evidence exists?* | **Missing.** Today there is a chat page that exposes maybe 10% of Layer 2. |

This document is about closing Layer 3.

---

## 4. External signals that support the direction

Three independent signals validate the redesign:

**Signal A — Governance is becoming the bottleneck.** Gartner-linked reporting on enterprise agents identifies governance failures as a major reason agents get demoted or decommissioned, especially where organisations treat governance as binary instead of *proportional to autonomy and risk*. Our five autonomy modes (`plan_only`, `approve_all`, `canary_only`, `risk_based`, `full_auto`) are exactly the proportional model. They must be **visible in the UI** for the architecture to be legible.

**Signal B — Agent observability is becoming a new product category.** Microsoft's 2026 security analysis says a large share of major enterprises now run active AI agents, and the open frontier is observability, governance and security *for those agents*. New Relic explicitly frames agentic AI as opaque without monitoring around the workflow itself. Mission Control should show: agent status, tool calls, reasoning traces, policy outcomes, cost, failures, approvals.

**Signal C — Traditional observability was built for humans; agents need different surfaces.** Recent commentary argues human-shaped dashboards (query → result → scroll) don't fit machine actors that need continuous full-fidelity context with different retention and economic models. So Mission Control is not "another dashboard with a chat assistant bolted on." It is a **machine-and-human coordination surface** — different from monitoring tools, different from chat UIs.

(Specific cites and links are tracked in the PR description, not inlined here.)

---

## 5. Why a chat page is the wrong product

A chat page sets the wrong mental model. It tells the user:

> *I ask. AI answers. Maybe I copy something.*

What Mission Control needs to say:

> *The system is continuously watching. Agents are working. Plans are forming. Policies are checking. Some actions need me. Some actions are already safe. Every action has receipts.*

That is a completely different product. **A chat UI hides the moat. Mission Control reveals it.**

We will keep a conversation input, because it's genuinely useful as an input modality. We will not let it be the centre of the page.

---

## 6. The surface (Phase A layout)

```
┌────────────────────────────────────────────────────────────────────────────────┐
│ Fleet  ●●●○○○○○○○○○  3 working · 9 idle    🟡 4 pending   $1.82 / $50 today    │  status bar
├──────────────┬──────────────────────────────────────┬───────────────────────────┤
│ AGENTS       │  ACTIVITY STREAM                     │  PENDING DECISIONS        │
│              │                                      │  ↑ ranked urgency × blast │
│ Drift     🟢 │  14:32  Vulnerability   list_cve…    │  ┌─────────────────────┐  │
│ Patch     🟢 │  14:31  Drift           analyze_drift│  │ Patch CVE-2024-…    │  │
│ Compliance🟡 │  14:30  Patch           gen_plan ⏸   │  │  4 prod assets      │  │
│ Incident  🔒 │  14:28  Cert            rotate (ok)  │  │  $0.04 LLM          │  │
│ DR        🟡 │  14:25  Compliance      check_ctrl   │  │  OPA: pass          │  │
│ Cost      🟢 │  14:20  Vulnerability   blast_radius │  │  Quality: 87/100    │  │
│ Security  🟡 │  14:15  Image           list_versions│  │  [Approve] [Modify] │  │
│ Image     🟢 │  14:10  SOP             validate_sop │  │  [Reject]  [Plan]   │  │
│ SOP       🟢 │  14:05  Drift           analyze_drift│  └─────────────────────┘  │
│ Adapter   🔒 │  14:00  Incident        (idle)       │  ┌─────────────────────┐  │
│ Cert      🟢 │  …                                   │  │ 3 more pending…     │  │
│ Vuln      🟢 │                                      │  └─────────────────────┘  │
│              │                                      │  AUTONOMY (read-only)     │
│              │                                      │   Drift  · prod  · semi   │
│              │                                      │   Drift  · stg   · auto   │
│              │                                      │   Vuln   · prod  · manual │
│              │                                      │   …                       │
├──────────────┴──────────────────────────────────────┴───────────────────────────┤
│ >_ Ask Mission Control…  (collapsed)                                       ⏎    │  conversation dock
└────────────────────────────────────────────────────────────────────────────────┘
```

### Five surfaces, in priority order

1. **Fleet status bar.** One dense line. Sets the frame: *you are running a fleet*.
2. **Agent roster (left rail).** 12 cards, one per agent. Status, autonomy mode, last action, pending decisions. The single biggest visual differentiator — no chat-shaped competitor has this.
3. **Activity stream (centre).** Live ticker. Each event: agent, tool, intent, assets, OPA result, risk score, current phase. Hover → quick summary. Click → full plan + run + receipts.
4. **Pending decisions (right rail).** Ranked by urgency × blast-radius. Each card: action in one sentence, blast radius, cost, OPA result, quality score, rollback availability, action buttons.
5. **Conversation dock (bottom).** Collapsed by default. Chat is **one input among several**, not the destination.

### Sub-pages (deferred to later phases, but reserve the routes)

- `/ai/agents/{agent}` — per-agent detail: history, error rate, latency, tool-call distribution.
- `/ai/tasks` — task ledger (already exists; keep, polish).
- `/ai/tasks/{id}` — full plan + run + receipts + replay.
- `/ai/ledger` — *new*: audit + cost + alternatives. Phase C.
- `/ai/settings` — autonomy edits, blocked windows, max-assets-per-execution, notifications. Phase C.

---

## 7. What "wow" actually means

The wow comes from:

**clarity · density · authority · evidence · control**

Not from:

~~sparkles · animations · gradients · cute chat polish~~

A first-time visitor should feel slightly out of their depth — like sitting at an operations console for the first time. That is the correct first impression for a product that runs real production infrastructure. Approachability is not a goal in Phase A; legibility for an operator is.

Visual borrowings:

- **From Bloomberg / Reuters terminals** — dense, deliberate. Every pixel earns its place.
- **From PagerDuty's incident view** — *right now* framing, urgency triage, receipts.
- **From Linear** — keyboard-first navigation. `g a` agents, `g d` decisions, `?` help.
- **From Devin's session view** — agent reasoning is browsable.
- **From a flight deck** — colour is *meaningful*, not decoration.

---

## 8. Phase A — exact scope (what AI-001 / E2E-011 ships)

**Frontend:**

- New `/ai` route — replaces the current chat page.
- Layout per §6: status bar + agent roster + activity stream + pending decisions + autonomy state (read-only) + collapsed conversation dock.
- Hooks read from existing orchestrator endpoints: `GET /api/v1/ai/agents`, `/ai/tools`, `/ai/tasks` (with status filters), task detail.
- Old chat page state is preserved (deep link `/ai/chat` or similar) only if cheap; otherwise removed.

**Backend:**

- **Prefer no new mutation endpoints for Phase A.** A small read-only aggregation endpoint such as `GET /api/v1/ai/fleet/status` is acceptable if it keeps the frontend simple and avoids duplicating fleet-count logic client-side. The orchestrator already exposes the underlying primitives we need (agents list, tools list, tasks list with status, task detail, executions); aggregation is a convenience, not a dependency.

**Seed (`scripts/seed-e2e-data`):**

- Add `seedMissionControl` step under the orchestrator dev org (`00000000-…-01`, same org used by CVE alerts).
- Insert deterministic rows in:
  - `ai_tasks` — one in each of: `pending_approval`, `executing`, `completed`, `rejected`/`failed`. ~4-5 tasks total.
  - `ai_plans` — one per task, with OPA pass result and a quality score (e.g., 87/100).
  - `ai_runs` — for completed/executing tasks, with at least one phase.
  - `ai_tool_invocations` — ~6-8 across at least 3 distinct agents (Drift, Vulnerability, Certificate) and 3 distinct tools.
  - `llm_usage` — small token spend rows to back the status-bar `$1.82 / $50` value.

**Spec (`ui/control-tower/e2e/ai-mission-control.spec.ts`):**

Per §11 below — 8–10 deterministic assertions.

---

## 9. What Phase A explicitly will NOT do

- **No live LLM calls in CI.** Period. The orchestrator's LLM client is not invoked from a test path.
- **No cloud execution.** No real AWS / Azure / GCP / vSphere calls.
- **No cloud credentials required in CI.** Mission Control Phase A renders seeded state only; it must not require AWS / Azure / GCP / vSphere / Kubernetes credentials to load or pass E2E.
- **No Temporal mutation.** Phase A does not start, signal, or fail-over Temporal workflows. (Reading workflow state from DB is fine.)
- **No approval mutation.** The approve / modify / reject buttons can be visible but must be wired to a no-op or an in-flight banner in Phase A. Mutation lands in Phase B alongside the LLM stub provider.
- **No streaming UI.** Server-sent events / websockets are Phase D. Phase A uses the existing polling shape.
- **No conversational state.** Multi-turn memory is Phase B.
- **No autonomy writes.** Autonomy panel is read-only. Edits land in Phase C.
- **No cute chat polish.** No avatars, no typing dots, no "Hi! I'm your AI assistant 👋". The conversation dock is a slim input — that's it.
- **No over-animation.** No celebrating green checkmarks, no fly-in cards on every update.

---

## 10. Test mode strategy

For Phase A the strategy is **seed terminal state, read it back** — no execute path tested.

- All assertions render against the deterministic seed inserted before the Playwright run.
- The orchestrator's LLM provider is not exercised. If any code path in the page accidentally triggers an LLM call (a "regenerate insight" button on render, etc.), that's a bug to fix in this PR — the page must be inert in steady state.

For Phase B (later) we will introduce a `RF_LLM_PROVIDER=stub` mode whose `Complete` and `CompleteWithTools` return canned responses keyed by intent. That unlocks E2E tests of the execute path. **Not in scope for AI-001.**

---

## 11. E2E-011 spec list (target assertions)

`ui/control-tower/e2e/ai-mission-control.spec.ts` — target ~5–8 specs:

1. **Header & subtitle** — *"Mission Control"* heading and *"Governed command for infrastructure agents."* subtitle visible.
2. **Status bar** — fleet counts visible (e.g., `3 working`, `9 idle`, `4 pending`), today's LLM spend visible (e.g., `$1.82`).
3. **Agent roster — all 12 agents** — each agent name is visible (Drift, Patch, Compliance, Incident, DR, Cost, Security, Image, SOP, Adapter, Certificate, Vulnerability).
4. **Agent autonomy state visible** — at least one agent's mode (e.g., `semi`, `auto`, `manual`) is rendered on its card.
5. **Activity stream — seeded events visible** — at least 3 of the seeded tool invocations are visible by name (e.g., `analyze_drift`, `calculate_blast_radius`, `rotate_cert` — pick stable ones from the seed).
6. **Pending decisions — seeded pending-approval task visible** — the seeded `pending_approval` task title is visible in the right rail.
7. **OPA / policy result visible on a pending card** — e.g., *"OPA: pass"* renders.
8. **Quality score visible on a pending card** — e.g., *"Quality: 87/100"* renders.
9. **Completed CVE task is in the activity stream / ledger** — the seeded `completed` task with a CVE intent renders (either in the stream or via navigation to `/ai/tasks`).
10. **Conversation dock visible but not dominant** — input field exists, is collapsed by default, occupies the bottom dock, does not steal layout focus.

Locally and in CI, the run is single-worker to match the existing pattern.

---

## 12. Phases B / C / D — what's deferred

Documented here so we don't forget what we owe, but **not in AI-001 PR**.

- **Phase B — conversation as a citizen.**
  Multi-turn conversation memory. Conversations stream into the activity feed alongside agent-initiated events. `RF_LLM_PROVIDER=stub` for deterministic test mode. E2E extends to: submit prompt → assert task appears in stream → assert pending decision appears (because the stub flagged it).

- **Phase C — make it ambient.**
  Autonomy writes (per agent × per env, OPA-checked). `/ai/ledger` page (audit + cost + alternatives). Push notifications (Slack & email) honouring per-org preferences. Replay view at `/ai/tasks/{id}` showing every tool call, every LLM call, every policy check, every approval.

- **Phase D — make it feel alive.**
  Streaming activity feed (SSE or websocket — replace polling). *"Why now"* proactive banners (the agent speaks first on threshold crossings — CVE drop, cert near expiry, risk band change). Multi-agent collaboration view (when 2+ agents work on the same intent). Memory affordances (*"Last time you said X, I remembered"*).

---

## 13. Product positioning

The full hierarchy this slots into:

```
QuantumLayer           — ambient software factory (umbrella)
  └─ QuantumFabric     — agentic execution fabric (shared engine: agents, OPA, Temporal, HITL, audit)
        └─ QuantumSRE  — operate / remediate station (this codebase)
              └─ Mission Control   — human command surface  ← THIS PAGE
                    └─ QuantumCapsule — evidence-backed output of executed plans
```

Mission Control is **the visible face of QuantumFabric inside QuantumSRE**. It is what an operator opens every morning, what an auditor opens after an incident, what a compliance officer points to during a review. It is not "the AI tab."

Marketing sentence governing every design decision:

> **Mission Control — see every agent, decision, policy check and action in one place. Governed command for infrastructure agents.**

---

## 14. Decisions log (the v1 open questions, resolved)

| # | Question | Decision |
|---|---|---|
| 1 | Redesign-first, or test the existing page first? | **Redesign-first.** Testing the chat page locks in a UX we're throwing away. |
| 2 | Canonical name? | **Mission Control.** Subtitle: *"Governed command for infrastructure agents."* |
| 3 | Agent roster aspirational (all 12) or earned (only active)? | **Aspirational** with `idle since X` subtext on quiet agents. Sells the fleet. |
| 4 | Where does the conversation strip live long-term? | **Bottom dock, collapsed.** Signals "input among many", not "the destination". |
| 5 | Default autonomy for a new org? | **All agents `plan_only` by default** (most conservative). User opts in to higher autonomy explicitly per agent × per env. |
| 6 | Stub LLM provider now or defer? | **Defer to Phase B.** Phase A doesn't need it. |
| 7 | OPA policy result as first-class UI? | **Yes.** It's an actual differentiator. Phase A shows pass/fail; human-readable explanation comes in Phase C with the ledger. |
| 8 | Positioning sentence? | *"Mission Control — governed command for infrastructure agents."* |

---

## 15. TL;DR for whoever opens AI-001

- Build a new page, not a test for the old one.
- Name it Mission Control.
- Phase A is read-only: 5 surfaces (status, agents, stream, decisions, dock), backed by a deterministic seed, no live LLM, no mutation.
- The wow is clarity, density, authority, evidence, control. Not sparkles.
- E2E-011 = ~5–8 specs asserting the seeded fleet state renders honestly.
- Phases B/C/D are real but not in scope.
- The wedge is **cross-domain governed remediation across the whole infrastructure lifecycle** — not "we have AI". Datadog, ServiceNow, Dynatrace, Microsoft, New Relic are all moving on agentic action; the differentiator is the coordination + governance surface, not the agents themselves.

Ready to be picked up as the next branch (`ai/001-mission-control-phase-a`).
