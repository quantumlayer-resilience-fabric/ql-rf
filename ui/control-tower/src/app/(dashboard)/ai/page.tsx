"use client";

// Mission Control — AI-001 Phase A.
// See docs/E2E-011-ai-mission-control.md for the strategic rationale and the
// "will NOT do" boundary. Phase A is read-only: status bar, agent roster,
// activity stream, pending decisions, autonomy panel (display only), and a
// collapsed conversation dock. No live LLM, no cloud execution, no approval
// mutation, no Temporal mutation, no streaming.

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@clerk/nextjs";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ChevronRight, Lock, Bot, Activity, Send, Loader2, ListChecks, CheckCircle2, PlayCircle, Hourglass, Wrench } from "lucide-react";
import {
  useSendAIMessage,
  useLatestConversation,
  useConversationMessages,
  useApproveTask,
  useCoApproveTask,
  useRejectTask,
  useRecentRuns,
  useRun,
  useTools,
  useInvokeTool,
  useDryRunTool,
  type ConversationMessage,
  type RunSummary,
  type RunAuditEntry,
  type ToolInfo,
} from "@/hooks/use-ai";
import { Zap } from "lucide-react";
import { MessageSquare } from "lucide-react";

// -----------------------------------------------------------------------------
// Backend types (snake_case) — match
//   GET /api/v1/ai/agents
//   GET /api/v1/ai/fleet/status
// -----------------------------------------------------------------------------

interface Agent {
  name: string;
  description: string;
  supported_tasks?: string[];
  required_tools?: string[];
}

interface AgentCounts {
  total: number;
  working: number;
  idle: number;
  blocked: number;
}

interface PendingDecision {
  task_id: string;
  plan_id: string;
  user_intent: string;
  plan_type: string;
  quality_score?: number;
  opa_pass: boolean;
  blast_radius_assets: number;
  environment: string;
  created_at: string;
  // PR #22 / CONN-003 (UI): two-approver workflow fields. Empty string
  // means "not yet set"; requires_two_approvers is the registry-derived
  // flag so the UI doesn't have to know which tool risks need two
  // approvers.
  approved_by?: string;
  second_approver?: string;
  requires_two_approvers?: boolean;
}

interface ToolInvocation {
  task_id: string;
  tool_name: string;
  risk_level: string;
  duration_ms?: number;
  created_at: string;
}

interface FleetStatus {
  agents: AgentCounts;
  pending_approvals: PendingDecision[];
  recent_invocations: ToolInvocation[];
  // Phase B.2: unified activity feed. Each event is discriminated by `kind`.
  recent_activity?: ActivityEvent[];
  tool_invocations_today: number;
  llm_spend_today_cents: number;
  llm_spend_budget_cents: number;
}

interface ActivityEvent {
  kind: "tool_invocation" | "conversation_message";
  task_id?: string;
  created_at: string;
  // tool_invocation fields
  tool_name?: string;
  risk_level?: string;
  duration_ms?: number;
  // conversation_message fields
  conversation_id?: string;
  message_id?: string;
  role?: string;
  content_preview?: string;
}

// -----------------------------------------------------------------------------
// Static maps for Phase A
//   * autonomyByAgent — read-only display per doc §6; storage lands in Phase C.
//   * agentByTool     — derived attribution for the activity stream until we
//                        store agent_name on ai_tool_invocations.
// -----------------------------------------------------------------------------

const autonomyByAgent: Record<string, "manual" | "semi" | "auto"> = {
  drift_agent: "semi",
  patch_agent: "manual",
  compliance_agent: "auto",
  incident_agent: "manual",
  dr_agent: "manual",
  cost_agent: "auto",
  security_agent: "manual",
  image_agent: "semi",
  sop_agent: "auto",
  adapter_agent: "manual",
  certificate_agent: "semi",
  vulnerability_agent: "semi",
};

const agentByTool: Record<string, string> = {
  list_cve_alerts: "vulnerability_agent",
  get_cve_details: "vulnerability_agent",
  calculate_blast_radius: "vulnerability_agent",
  get_alert_blast_radius: "vulnerability_agent",
  list_patch_campaigns: "vulnerability_agent",
  create_patch_campaign: "vulnerability_agent",
  get_campaign_status: "vulnerability_agent",
  calculate_urgency_score: "vulnerability_agent",
  analyze_drift: "drift_agent",
  query_assets: "drift_agent",
  get_drift_status: "drift_agent",
  get_golden_image: "drift_agent",
  compare_versions: "drift_agent",
  generate_patch_plan: "patch_agent",
  generate_rollout_plan: "patch_agent",
  simulate_rollout: "patch_agent",
  propose_rollout: "patch_agent",
  list_certificates: "certificate_agent",
  get_certificate_details: "certificate_agent",
  map_certificate_usage: "certificate_agent",
  generate_cert_renewal_plan: "certificate_agent",
  propose_cert_rotation: "certificate_agent",
  validate_tls_handshake: "certificate_agent",
  check_control: "compliance_agent",
  generate_compliance_evidence: "compliance_agent",
  get_compliance_status: "compliance_agent",
  generate_dr_runbook: "dr_agent",
  simulate_failover: "dr_agent",
  get_dr_status: "dr_agent",
  acknowledge_alert: "incident_agent",
  query_alerts: "incident_agent",
  calculate_risk_score: "security_agent",
  generate_sop: "sop_agent",
  validate_sop: "sop_agent",
  simulate_sop: "sop_agent",
  execute_sop: "sop_agent",
  list_sops: "sop_agent",
  generate_image_contract: "image_agent",
  generate_packer_template: "image_agent",
  generate_ansible_playbook: "image_agent",
  build_image: "image_agent",
  list_image_versions: "image_agent",
  promote_image: "image_agent",
};

const AGENT_DISPLAY: Record<string, { label: string; short: string }> = {
  drift_agent: { label: "Drift", short: "drift" },
  patch_agent: { label: "Patch", short: "patch" },
  compliance_agent: { label: "Compliance", short: "compliance" },
  incident_agent: { label: "Incident", short: "incident" },
  dr_agent: { label: "DR", short: "dr" },
  cost_agent: { label: "Cost", short: "cost" },
  security_agent: { label: "Security", short: "security" },
  image_agent: { label: "Image", short: "image" },
  sop_agent: { label: "SOP", short: "sop" },
  adapter_agent: { label: "Adapter", short: "adapter" },
  certificate_agent: { label: "Certificate", short: "certificate" },
  vulnerability_agent: { label: "Vulnerability", short: "vulnerability" },
};

// -----------------------------------------------------------------------------
// Fetch helpers — mirror the orchestratorFetch pattern in use-ai.ts (Clerk
// Bearer token in real auth, "dev-token" fallback in dev mode).
// -----------------------------------------------------------------------------

const ORCHESTRATOR_URL =
  process.env.NEXT_PUBLIC_ORCHESTRATOR_URL || "http://localhost:8083";

async function orchestratorGet<T>(
  endpoint: string,
  getToken: () => Promise<string | null>,
): Promise<T> {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  const token = await getToken();
  if (token) headers["Authorization"] = `Bearer ${token}`;
  const r = await fetch(`${ORCHESTRATOR_URL}${endpoint}`, { headers });
  if (!r.ok) throw new Error(`${endpoint} -> ${r.status}`);
  return (await r.json()) as T;
}

function useFleetStatus() {
  const { getToken } = useAuth();
  const tokenFn = async () => (await getToken()) || "dev-token";
  return useQuery({
    queryKey: ["ai", "fleet", "status"],
    queryFn: () => orchestratorGet<FleetStatus>("/api/v1/ai/fleet/status", tokenFn),
    refetchInterval: 15_000,
  });
}

function useAgentsList() {
  const { getToken } = useAuth();
  const tokenFn = async () => (await getToken()) || "dev-token";
  return useQuery({
    queryKey: ["ai", "agents"],
    queryFn: () => orchestratorGet<{ agents: Agent[] }>("/api/v1/ai/agents", tokenFn),
  });
}

// -----------------------------------------------------------------------------
// Formatting helpers
// -----------------------------------------------------------------------------

function formatCents(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`;
}

function autonomyTone(mode: string): string {
  switch (mode) {
    case "auto":
      return "bg-status-green/10 text-status-green border-status-green/30";
    case "semi":
      return "bg-status-amber/10 text-status-amber border-status-amber/30";
    case "manual":
    default:
      return "bg-muted text-muted-foreground border-muted-foreground/30";
  }
}

function riskTone(risk: string): string {
  if (risk.includes("prod")) return "text-status-red";
  if (risk === "plan_only") return "text-status-amber";
  return "text-muted-foreground";
}

// -----------------------------------------------------------------------------
// Page
// -----------------------------------------------------------------------------

export default function MissionControlPage() {
  const fleet = useFleetStatus();
  const agents = useAgentsList();

  const status = fleet.data;
  const agentList = agents.data?.agents ?? [];

  return (
    <div className="page-transition space-y-4">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Mission Control
          </h1>
          <p className="text-muted-foreground">
            Governed command for infrastructure agents.
          </p>
        </div>
      </div>

      <FleetStatusBar status={status} />

      <div className="grid gap-4 lg:grid-cols-[260px_minmax(0,1fr)_320px]">
        <AgentRoster agents={agentList} status={status} />
        <ActivityStream status={status} />
        <PendingDecisionsRail status={status} />
      </div>

      <ConversationDock />
    </div>
  );
}

// -----------------------------------------------------------------------------
// Fleet status bar
// -----------------------------------------------------------------------------

function FleetStatusBar({ status }: { status?: FleetStatus }) {
  return (
    <Card>
      <CardContent className="flex flex-wrap items-center justify-between gap-4 p-4 text-sm">
        <div className="flex items-center gap-6">
          <div className="flex items-center gap-2">
            <Bot className="h-4 w-4 text-brand-accent" />
            <span className="font-medium">Fleet</span>
          </div>
          <span data-testid="fleet-working">
            <span className="font-semibold text-foreground">{status?.agents.working ?? 0}</span>{" "}
            <span className="text-muted-foreground">working</span>
          </span>
          <span data-testid="fleet-idle">
            <span className="font-semibold text-foreground">{status?.agents.idle ?? 0}</span>{" "}
            <span className="text-muted-foreground">idle</span>
          </span>
          <span data-testid="fleet-blocked">
            <span className="font-semibold text-foreground">{status?.agents.blocked ?? 0}</span>{" "}
            <span className="text-muted-foreground">blocked</span>
          </span>
        </div>
        <div className="flex items-center gap-6">
          <span data-testid="fleet-pending">
            <span className="font-semibold text-status-amber">
              {status?.pending_approvals.length ?? 0}
            </span>{" "}
            <span className="text-muted-foreground">pending</span>
          </span>
          <span data-testid="fleet-actions-today">
            <span className="font-semibold text-foreground">
              {status?.tool_invocations_today ?? 0}
            </span>{" "}
            <span className="text-muted-foreground">tool runs today</span>
          </span>
          <span data-testid="fleet-spend">
            <span className="font-semibold text-foreground">
              {formatCents(status?.llm_spend_today_cents ?? 0)}
            </span>{" "}
            <span className="text-muted-foreground">
              / {formatCents(status?.llm_spend_budget_cents ?? 0)} today
            </span>
          </span>
        </div>
      </CardContent>
    </Card>
  );
}

// -----------------------------------------------------------------------------
// Agent roster
// -----------------------------------------------------------------------------

function AgentRoster({
  agents,
  status,
}: {
  agents: Agent[];
  status?: FleetStatus;
}) {
  const activeAgents = new Set<string>();
  for (const inv of status?.recent_invocations ?? []) {
    const owner = agentByTool[inv.tool_name];
    if (owner) activeAgents.add(owner);
  }

  return (
    <Card className="h-fit">
      <CardContent className="p-3">
        <div className="mb-2 flex items-center justify-between px-1 text-xs uppercase tracking-wider text-muted-foreground">
          <span>Agents</span>
          <span>{agents.length}</span>
        </div>
        <div className="space-y-1">
          {agents.map((a) => {
            const display = AGENT_DISPLAY[a.name] ?? { label: a.name, short: a.name };
            const autonomy = autonomyByAgent[a.name] ?? "manual";
            const active = activeAgents.has(a.name);
            return (
              <div
                key={a.name}
                data-testid={`agent-${display.short}`}
                className="flex items-center justify-between gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-muted/50"
              >
                <div className="flex min-w-0 items-center gap-2">
                  <span
                    className={`h-2 w-2 shrink-0 rounded-full ${
                      active ? "bg-status-green" : "bg-muted-foreground/40"
                    }`}
                    aria-hidden
                  />
                  <span className="truncate font-medium text-foreground">
                    {display.label}
                  </span>
                </div>
                <Badge
                  variant="outline"
                  className={`text-[10px] uppercase ${autonomyTone(autonomy)}`}
                >
                  {autonomy}
                </Badge>
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}

// -----------------------------------------------------------------------------
// Activity stream
// -----------------------------------------------------------------------------

function ActivityStream({ status }: { status?: FleetStatus }) {
  // Phase B.2: prefer the unified `recent_activity` feed (discriminated by
  // `kind`). Fall back to the legacy `recent_invocations` shape so the page
  // keeps rendering during the brief window between an orchestrator deploy
  // and the next fleet-status poll.
  const events: ActivityEvent[] =
    status?.recent_activity && status.recent_activity.length > 0
      ? status.recent_activity
      : (status?.recent_invocations ?? []).map((inv) => ({
          kind: "tool_invocation" as const,
          task_id: inv.task_id,
          tool_name: inv.tool_name,
          risk_level: inv.risk_level,
          duration_ms: inv.duration_ms,
          created_at: inv.created_at,
        }));

  return (
    <Card className="h-fit">
      <CardContent className="p-3">
        <div className="mb-2 flex items-center justify-between px-1 text-xs uppercase tracking-wider text-muted-foreground">
          <span className="flex items-center gap-2">
            <Activity className="h-3.5 w-3.5" />
            Activity stream
          </span>
          <span>{events.length}</span>
        </div>
        <div className="space-y-1">
          {events.length === 0 ? (
            <div className="px-3 py-6 text-center text-sm text-muted-foreground">
              No recent activity.
            </div>
          ) : (
            events.map((e, idx) => {
              if (e.kind === "conversation_message") {
                return (
                  <div
                    key={`conv-${e.message_id ?? idx}`}
                    data-testid={`activity-conversation-${e.message_id ?? idx}`}
                    className="flex items-center justify-between gap-3 rounded-md px-3 py-2 text-sm hover:bg-muted/50"
                  >
                    <div className="flex min-w-0 items-center gap-3">
                      <span className="font-mono text-xs text-muted-foreground">
                        {e.created_at.slice(11, 16)}
                      </span>
                      <MessageSquare className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                      <span className="text-muted-foreground">you</span>
                      <span className="truncate text-foreground" title={e.content_preview ?? ""}>
                        {e.content_preview ?? ""}
                      </span>
                    </div>
                  </div>
                );
              }
              const toolName = e.tool_name ?? "";
              const riskLevel = e.risk_level ?? "";
              const owner = agentByTool[toolName];
              const display = owner ? AGENT_DISPLAY[owner]?.label ?? owner : "—";
              return (
                <div
                  key={`tool-${e.task_id ?? ""}-${toolName}-${e.created_at}`}
                  data-testid={`activity-${toolName}`}
                  className="flex items-center justify-between gap-3 rounded-md px-3 py-2 text-sm hover:bg-muted/50"
                >
                  <div className="flex min-w-0 items-center gap-3">
                    <span className="font-mono text-xs text-muted-foreground">
                      {e.created_at.slice(11, 16)}
                    </span>
                    <span className="text-muted-foreground">{display}</span>
                    <span className="font-mono text-foreground">{toolName}</span>
                  </div>
                  <div className="flex items-center gap-3 text-xs">
                    <span className={`uppercase ${riskTone(riskLevel)}`}>
                      {riskLevel.replace(/_/g, " ")}
                    </span>
                    {e.duration_ms != null && (
                      <span className="text-muted-foreground">{e.duration_ms}ms</span>
                    )}
                  </div>
                </div>
              );
            })
          )}
        </div>
      </CardContent>
    </Card>
  );
}

// -----------------------------------------------------------------------------
// Pending decisions rail
// -----------------------------------------------------------------------------

function PendingDecisionsRail({ status }: { status?: FleetStatus }) {
  const decisions = status?.pending_approvals ?? [];
  // Phase B.3: approve + reject mutations. Modify stays disabled until B.4
  // (needs a plan-payload editor). On success the hooks invalidate fleet
  // status + conversation queries so the dock thread and pending count
  // refresh immediately rather than waiting for the 15s poll.
  const approve = useApproveTask();
  const coApprove = useCoApproveTask();
  const reject = useRejectTask();
  const [errorByPlan, setErrorByPlan] = useState<Record<string, string>>({});

  const handleAction = async (
    planID: string,
    taskID: string,
    fn: (args: { taskId: string }) => Promise<unknown>,
  ) => {
    setErrorByPlan((prev) => ({ ...prev, [planID]: "" }));
    try {
      await fn({ taskId: taskID });
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Action failed";
      setErrorByPlan((prev) => ({ ...prev, [planID]: msg }));
    }
  };

  return (
    <div className="space-y-3">
      <Card>
        <CardContent className="p-3">
          <div className="mb-2 flex items-center justify-between px-1 text-xs uppercase tracking-wider text-muted-foreground">
            <span>Pending decisions</span>
            <span>{decisions.length}</span>
          </div>
          {decisions.length === 0 ? (
            <div className="px-2 py-6 text-center text-sm text-muted-foreground">
              Nothing waiting on you.
            </div>
          ) : (
            <div className="space-y-3">
              {decisions.map((d) => {
                const pending = approve.isPending || coApprove.isPending || reject.isPending;
                const errMsg = errorByPlan[d.plan_id];
                // PR #22 / CONN-003 (UI): a card is "awaiting second
                // approval" when the plan needs two approvers AND the first
                // one has been recorded but the second hasn't. The Approve
                // button hides in that state (it would 409 server-side
                // anyway) and the Co-approve button takes its place.
                const needsTwo = d.requires_two_approvers === true;
                const firstApprover = d.approved_by ?? "";
                const awaitingSecond = needsTwo && firstApprover !== "" && (d.second_approver ?? "") === "";
                return (
                  <div
                    key={d.plan_id}
                    data-testid={`pending-${d.plan_id}`}
                    className="rounded-md border bg-card p-3"
                  >
                    <div className="mb-2 text-sm font-medium text-foreground">
                      {d.user_intent}
                    </div>
                    {awaitingSecond && (
                      <div
                        data-testid={`pending-awaiting-second-${d.plan_id}`}
                        className="mb-2 flex items-center justify-between rounded-md border border-status-amber/40 bg-status-amber/10 px-2 py-1 text-[11px]"
                      >
                        <span className="font-medium uppercase tracking-wider text-status-amber">
                          Awaiting second approval
                        </span>
                        <span
                          className="font-mono text-muted-foreground"
                          title={`First approver: ${firstApprover}`}
                        >
                          1st: {firstApprover.slice(0, 8)}…
                        </span>
                      </div>
                    )}
                    <div className="mb-2 grid grid-cols-2 gap-1 text-xs">
                      <span className="text-muted-foreground">Blast radius</span>
                      <span className="text-right">
                        {d.blast_radius_assets} {d.environment || "—"}
                      </span>
                      <span className="text-muted-foreground">OPA</span>
                      <span
                        className={`text-right font-medium ${
                          d.opa_pass ? "text-status-green" : "text-status-red"
                        }`}
                        data-testid="pending-opa"
                      >
                        {d.opa_pass ? "pass" : "fail"}
                      </span>
                      {d.quality_score != null && (
                        <>
                          <span className="text-muted-foreground">Quality</span>
                          <span className="text-right" data-testid="pending-quality">
                            {d.quality_score}/100
                          </span>
                        </>
                      )}
                      {needsTwo && (
                        <>
                          <span className="text-muted-foreground">Approvers</span>
                          <span
                            className="text-right font-mono text-[11px]"
                            data-testid={`pending-approvers-${d.plan_id}`}
                            title="Two distinct approvers required for state_change_prod tools"
                          >
                            2 required
                          </span>
                        </>
                      )}
                    </div>
                    <div className="flex flex-wrap gap-1">
                      {awaitingSecond ? (
                        <Button
                          size="sm"
                          variant="outline"
                          disabled={pending}
                          data-testid={`pending-co-approve-${d.plan_id}`}
                          onClick={() => void handleAction(d.plan_id, d.task_id, coApprove.mutateAsync)}
                          title="Co-approve (must be a different user from the first approver)"
                        >
                          {coApprove.isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : "Co-approve"}
                        </Button>
                      ) : (
                        <Button
                          size="sm"
                          variant="outline"
                          disabled={pending}
                          data-testid={`pending-approve-${d.plan_id}`}
                          onClick={() => void handleAction(d.plan_id, d.task_id, approve.mutateAsync)}
                          title={needsTwo ? "Approve (first of two approvers required)" : "Approve and start simulated execution"}
                        >
                          {approve.isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : "Approve"}
                        </Button>
                      )}
                      <Button
                        size="sm"
                        variant="outline"
                        disabled
                        title="Modify deferred to B.4"
                      >
                        Modify
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        disabled={pending}
                        data-testid={`pending-reject-${d.plan_id}`}
                        onClick={() => void handleAction(d.plan_id, d.task_id, reject.mutateAsync)}
                        title="Reject this plan"
                      >
                        {reject.isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : "Reject"}
                      </Button>
                    </div>
                    {errMsg && (
                      <p
                        data-testid={`pending-error-${d.plan_id}`}
                        className="mt-2 px-1 text-xs text-status-red"
                      >
                        {errMsg}
                      </p>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      <RealToolsCard />

      <RecentRunsRail />

      <Card>
        <CardContent className="p-3">
          <div className="mb-2 flex items-center justify-between px-1 text-xs uppercase tracking-wider text-muted-foreground">
            <span className="flex items-center gap-1">
              <Lock className="h-3 w-3" />
              Autonomy
            </span>
            <span>read-only</span>
          </div>
          <div className="space-y-1 text-xs">
            {Object.entries(autonomyByAgent)
              .slice(0, 4)
              .map(([agentName, mode]) => {
                const display = AGENT_DISPLAY[agentName] ?? {
                  label: agentName,
                  short: agentName,
                };
                return (
                  <div
                    key={agentName}
                    className="flex items-center justify-between gap-2"
                    data-testid={`autonomy-${display.short}`}
                  >
                    <span className="text-muted-foreground">
                      {display.label} · prod
                    </span>
                    <Badge
                      variant="outline"
                      className={`text-[10px] uppercase ${autonomyTone(mode)}`}
                    >
                      {mode}
                    </Badge>
                  </div>
                );
              })}
            <div className="pt-1 text-[11px] text-muted-foreground">
              Editing lands in Phase C.
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// -----------------------------------------------------------------------------
// Recent runs rail + audit timeline (PR #16 / UX-001)
//   * RecentRunsRail polls /api/v1/ai/runs at 2s/15s depending on whether any
//     run is in-flight. Renders up to 5 RunCards.
//   * RunCard is an expandable mini-card showing a state badge + current
//     phase + relative time. data-state attribute exposes the lifecycle
//     state for E2E selectors.
//   * RunAuditTimeline fetches /api/v1/ai/runs/{id} and renders the
//     interleaved audit_log + tool_invocation timeline. Read-only.
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// Real tools panel (PR #19 / CONN-001)
//   * RealToolsCard lists registered tools and shows "Invoke" buttons for
//     read_only ones. Plan-only / state-change tools are visible but not
//     directly invocable here — they flow through the approval pipeline.
//   * If no read-only real tools are registered (e.g., AWS creds missing
//     and fallback-to-mock disabled), the card shows an empty state. This
//     is the most demo-leverage message in the product right now.
//   * Invoking a tool inserts a REAL ai_tool_invocations row — no
//     _simulated marker. Distinguishable from the B.3 simulator's output.
// -----------------------------------------------------------------------------

// Demo defaults for state-change tools' dry-run params. The mock SSM client
// ignores these; the real SSM client validates strictly. The IDs are
// well-formed real-looking EC2 IDs so both paths accept them.
const ssmDryRunDefaultParams: Record<string, Record<string, unknown>> = {
  ssm_send_patch_command: {
    region: "us-east-1",
    instance_ids: ["i-0a1b2c3d4e5f6a7b8", "i-1234567890abcdef0"],
    operation: "Scan",
  },
};

function RealToolsCard() {
  const tools = useTools();
  const invoke = useInvokeTool();
  const dryRun = useDryRunTool();
  const [lastResult, setLastResult] = useState<{ tool: string; ok: boolean; message: string } | null>(null);

  const allTools = tools.data ?? [];
  // Read-only tools (invocable via /invoke), state-change tools (dry-run-able
  // via /dry-run). Plan-only tools are shown as informational badges only —
  // they have no /invoke or /dry-run path in PR #20.
  const readOnly = allTools.filter((t) => t.risk === "read_only");
  const planOnly = allTools.filter((t) => t.risk === "plan_only");
  // State-change tools sorted so the ones we have dry-run handlers for
  // (currently just ssm_send_patch_command) come first — the slice(0, 3)
  // below would otherwise drop them depending on registration order.
  const stateChange = allTools
    .filter((t) => t.risk === "state_change_nonprod" || t.risk === "state_change_prod")
    .sort((a, b) => {
      const aHandled = a.name in ssmDryRunDefaultParams ? 0 : 1;
      const bHandled = b.name in ssmDryRunDefaultParams ? 0 : 1;
      return aHandled - bHandled;
    });

  const onInvoke = async (toolName: string) => {
    setLastResult(null);
    try {
      const resp = await invoke.mutateAsync({ toolName });
      const summary =
        typeof resp.result === "object" && resp.result !== null && "instance_count" in resp.result
          ? `Got ${(resp.result as { instance_count: number }).instance_count} result(s) in ${resp.duration_ms}ms.`
          : `Completed in ${resp.duration_ms}ms.`;
      setLastResult({ tool: toolName, ok: true, message: summary });
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Invoke failed";
      setLastResult({ tool: toolName, ok: false, message: msg });
    }
  };

  const onDryRun = async (toolName: string) => {
    setLastResult(null);
    try {
      const params = ssmDryRunDefaultParams[toolName] ?? {};
      const resp = await dryRun.mutateAsync({ toolName, params });
      const result = resp.result as { command_plan?: { document_name?: string; instance_ids?: string[] } } | undefined;
      const summary = result?.command_plan
        ? `Constructed ${result.command_plan.document_name} for ${result.command_plan.instance_ids?.length ?? 0} instance(s). No SendCommand fired.`
        : `Dry-run completed in ${resp.duration_ms}ms. No SendCommand fired.`;
      setLastResult({ tool: toolName, ok: true, message: summary });
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Dry-run failed";
      setLastResult({ tool: toolName, ok: false, message: msg });
    }
  };

  return (
    <Card>
      <CardContent className="p-3">
        <div className="mb-2 flex items-center justify-between px-1 text-xs uppercase tracking-wider text-muted-foreground">
          <span className="flex items-center gap-2">
            <Zap className="h-3.5 w-3.5" />
            Real tools
          </span>
          <span data-testid="real-tools-count">{readOnly.length}</span>
        </div>

        {readOnly.length === 0 ? (
          <div
            data-testid="real-tools-empty"
            className="px-3 py-4 text-center text-xs text-muted-foreground"
          >
            No real cloud tools configured. Set AWS credentials and restart the
            orchestrator (or enable fallback mock for local dev).
          </div>
        ) : (
          <div className="space-y-1">
            {readOnly.map((t) => (
              <div
                key={t.name}
                data-testid={`real-tools-${t.name}`}
                className="flex items-center justify-between gap-2 rounded-md border bg-card px-3 py-2"
              >
                <div className="min-w-0 flex-1">
                  <div className="font-mono text-xs">{t.name}</div>
                  <div className="text-[10px] uppercase tracking-wider text-status-green">read-only</div>
                </div>
                <Button
                  size="sm"
                  variant="outline"
                  data-testid={`real-tools-invoke-${t.name}`}
                  onClick={() => void onInvoke(t.name)}
                  disabled={invoke.isPending}
                  title="Invoke this real cloud tool"
                >
                  {invoke.isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : "Invoke"}
                </Button>
              </div>
            ))}
          </div>
        )}

        {stateChange.length > 0 && (
          <div className="mt-2 space-y-1 border-t pt-2">
            {stateChange.slice(0, 3).map((t) => (
              <div
                key={t.name}
                data-testid={`real-tools-${t.name}`}
                className="flex items-center justify-between gap-2 rounded-md border bg-card px-3 py-2"
              >
                <div className="min-w-0 flex-1">
                  <div className="font-mono text-xs">{t.name}</div>
                  <div className="text-[10px] uppercase tracking-wider text-status-amber">
                    {t.risk.replace(/_/g, " ")}
                  </div>
                </div>
                <Button
                  size="sm"
                  variant="outline"
                  data-testid={`real-tools-dry-run-${t.name}`}
                  onClick={() => void onDryRun(t.name)}
                  disabled={dryRun.isPending}
                  title="Dry-run (constructs command, never calls SSM SendCommand)"
                >
                  {dryRun.isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : "Dry-run"}
                </Button>
              </div>
            ))}
          </div>
        )}

        {planOnly.length > 0 && (
          <div className="mt-2 space-y-1 border-t pt-2">
            {planOnly.slice(0, 2).map((t) => (
              <div
                key={t.name}
                className="flex items-center justify-between gap-2 rounded-md px-3 py-1 text-xs text-muted-foreground"
              >
                <span className="truncate font-mono">{t.name}</span>
                <span className="uppercase">plan-only</span>
              </div>
            ))}
          </div>
        )}

        {lastResult && (
          <p
            data-testid={`real-tools-result-${lastResult.tool}`}
            className={`mt-2 px-1 text-xs ${
              lastResult.ok ? "text-status-green" : "text-status-red"
            }`}
          >
            {lastResult.message}
          </p>
        )}
      </CardContent>
    </Card>
  );
}

function RecentRunsRail() {
  const runs = useRecentRuns(5);
  const [expandedRunID, setExpandedRunID] = useState<string | null>(null);
  const list = runs.data ?? [];

  return (
    <Card>
      <CardContent className="p-3">
        <div className="mb-2 flex items-center justify-between px-1 text-xs uppercase tracking-wider text-muted-foreground">
          <span className="flex items-center gap-2">
            <ListChecks className="h-3.5 w-3.5" />
            Recent runs
          </span>
          <span data-testid="recent-runs-count">{list.length}</span>
        </div>
        <div className="space-y-1">
          {list.length === 0 ? (
            <div
              data-testid="recent-runs-empty"
              className="px-3 py-6 text-center text-sm text-muted-foreground"
            >
              No runs yet. Approve a pending decision to start one.
            </div>
          ) : (
            list.map((r) => (
              <RunCard
                key={r.id}
                run={r}
                expanded={expandedRunID === r.id}
                onToggle={() =>
                  setExpandedRunID(expandedRunID === r.id ? null : r.id)
                }
              />
            ))
          )}
        </div>
      </CardContent>
    </Card>
  );
}

function RunCard({
  run,
  expanded,
  onToggle,
}: {
  run: RunSummary;
  expanded: boolean;
  onToggle: () => void;
}) {
  return (
    <div
      data-testid={`run-${run.id}`}
      data-state={run.state}
      className="rounded-md border bg-card"
    >
      <button
        type="button"
        onClick={onToggle}
        data-testid={`run-toggle-${run.id}`}
        className="flex w-full items-center justify-between gap-2 p-3 text-left hover:bg-muted/40"
      >
        <div className="min-w-0 flex-1">
          <div
            className="truncate text-sm font-medium text-foreground"
            title={run.user_intent}
          >
            {run.user_intent || "(no intent)"}
          </div>
          <div className="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
            <RunStateBadge state={run.state} />
            {run.state === "executing" && run.current_phase && (
              <span>phase: {run.current_phase}</span>
            )}
            <span>{describeProgress(run)}</span>
            <span>{relativeTime(run.updated_at)}</span>
          </div>
        </div>
        <ChevronRight
          className={`h-4 w-4 shrink-0 text-muted-foreground transition-transform ${
            expanded ? "rotate-90" : ""
          }`}
        />
      </button>
      {expanded && <RunAuditTimeline runID={run.id} />}
    </div>
  );
}

function RunAuditTimeline({ runID }: { runID: string }) {
  const { data, isLoading, error } = useRun(runID);

  if (isLoading) {
    return (
      <div className="border-t bg-muted/10 p-3 text-xs text-muted-foreground">
        Loading…
      </div>
    );
  }
  if (error || !data) {
    return (
      <div
        data-testid={`run-timeline-error-${runID}`}
        className="border-t bg-muted/10 p-3 text-xs text-status-red"
      >
        Failed to load run.
      </div>
    );
  }

  const { run, tool_invocations } = data;
  // Tool invocations are inserted 1:1 per phase by the B.3 simulator, ordered
  // by created_at. We walk the audit_log in order and attach the next pending
  // invocation under each phase_complete entry. If a future writer breaks the
  // 1:1, the timeline gracefully misaligns (best-effort UX, not a correctness
  // invariant).
  let invIdx = 0;

  return (
    <div
      data-testid={`run-timeline-${run.id}`}
      className="border-t bg-muted/10 p-3 text-xs"
    >
      <ol className="space-y-2">
        {run.audit_log.map((entry, i) => {
          const ts = formatTimestamp(String(entry.ts ?? ""));
          const kind = String(entry.kind ?? "");
          const inv =
            kind === "phase_complete" ? tool_invocations[invIdx++] : null;
          return (
            <li
              key={i}
              data-testid={`audit-${run.id}-${i}`}
              data-kind={kind}
              className="flex flex-col gap-1"
            >
              <div className="flex items-center gap-2">
                <span className="font-mono text-muted-foreground">{ts}</span>
                <KindIcon kind={kind} />
                <span className="text-foreground">{describeAuditEntry(entry)}</span>
              </div>
              {inv && (
                <div className="ml-10 flex items-center gap-2 rounded-sm border bg-card px-2 py-1">
                  <Wrench className="h-3 w-3 text-muted-foreground" />
                  <span className="font-mono">{inv.tool_name}</span>
                  <span className={`uppercase ${riskTone(inv.risk_level)}`}>
                    {inv.risk_level.replace(/_/g, " ")}
                  </span>
                  {inv.duration_ms != null && (
                    <span className="text-muted-foreground">{inv.duration_ms}ms</span>
                  )}
                </div>
              )}
            </li>
          );
        })}
      </ol>
      {run.simulated && (
        <p className="mt-3 text-[10px] uppercase tracking-wider text-muted-foreground">
          Simulated run — no real infrastructure changes.
        </p>
      )}
    </div>
  );
}

// -----------------------------------------------------------------------------
// Run rendering helpers — pure functions, no DOM, no new deps.
// -----------------------------------------------------------------------------

function RunStateBadge({ state }: { state: RunSummary["state"] }) {
  const tone =
    state === "completed"
      ? "text-status-green"
      : state === "executing"
      ? "text-status-amber"
      : state === "queued"
      ? "text-muted-foreground"
      : state === "failed" || state === "rolled_back"
      ? "text-status-red"
      : "text-muted-foreground";
  return <span className={`font-medium uppercase ${tone}`}>{state}</span>;
}

function KindIcon({ kind }: { kind: string }) {
  const cls = "h-3.5 w-3.5 shrink-0";
  switch (kind) {
    case "approved":
      return <CheckCircle2 className={`${cls} text-status-green`} />;
    case "started":
      return <PlayCircle className={`${cls} text-status-amber`} />;
    case "phase_complete":
      return <CheckCircle2 className={`${cls} text-muted-foreground`} />;
    case "simulated_complete":
      return <CheckCircle2 className={`${cls} text-status-green`} />;
    default:
      // Unknown kinds (forward-compatible for future real-run event shapes).
      return <Hourglass className={`${cls} text-muted-foreground`} />;
  }
}

function describeAuditEntry(entry: RunAuditEntry): string {
  const kind = String(entry.kind ?? "");
  switch (kind) {
    case "approved":
      return entry.by ? `approved by ${String(entry.by).slice(0, 8)}…` : "approved";
    case "started":
      return "started";
    case "phase_complete": {
      const phase = entry.phase ? String(entry.phase) : "phase";
      return `phase ${phase} completed`;
    }
    case "simulated_complete": {
      const n = typeof entry.tool_invocations === "number" ? entry.tool_invocations : 0;
      return `simulated complete · ${n} invocation${n === 1 ? "" : "s"} · no real changes`;
    }
    default:
      return kind || "(unknown event)";
  }
}

function describeProgress(run: RunSummary): string {
  if (run.state === "completed") return `${run.phases_total} phase${run.phases_total === 1 ? "" : "s"}`;
  if (run.state === "executing") return `${run.percent_complete}%`;
  if (run.state === "queued") return "queued";
  return run.state;
}

function relativeTime(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const diffSec = Math.round((Date.now() - d.getTime()) / 1000);
  if (diffSec < 5) return "just now";
  if (diffSec < 60) return `${diffSec}s ago`;
  if (diffSec < 3600) return `${Math.round(diffSec / 60)}m ago`;
  if (diffSec < 86400) return `${Math.round(diffSec / 3600)}h ago`;
  return d.toLocaleDateString();
}

function formatTimestamp(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  // HH:MM:SS in local time — keeps the timeline compact.
  return d.toLocaleTimeString([], { hour12: false });
}

// -----------------------------------------------------------------------------
// Conversation dock (collapsed by default; expand-on-focus)
// -----------------------------------------------------------------------------

function ConversationDock() {
  const [open, setOpen] = useState(false);
  const [text, setText] = useState("");
  const [error, setError] = useState<string | null>(null);
  const queryClient = useQueryClient();
  const send = useSendAIMessage();

  // Phase B.2: render the active conversation thread above the input. The
  // backend decides which conversation is "active" (60-min append window per
  // user, see services/orchestrator/internal/handlers/conversations.go). The
  // dock just shows whatever GET /conversations?limit=1 returns. Successive
  // submits within the window grow the same thread; outside the window a new
  // conversation starts automatically.
  const latest = useLatestConversation();
  const messages = useConversationMessages(latest.data?.id);
  const thread: ConversationMessage[] = messages.data?.messages ?? [];

  // Submit goes through POST /ai/execute. With the stub LLM provider active
  // (Phase B.1), the orchestrator short-circuits to plan-only: a new
  // ai_task + ai_plan in `awaiting_approval` state is created and the fleet
  // status query is invalidated below so the pending decisions rail updates
  // immediately rather than waiting on the 15s refetch interval.
  // Environment is "staging" to keep `production_safety` OPA out of the path.
  const handleSubmit = async () => {
    const value = text.trim();
    if (!value || send.isPending) return;
    setError(null);
    try {
      await send.mutateAsync({ message: value, context: { environment: "staging" } });
      setText("");
      queryClient.invalidateQueries({ queryKey: ["ai", "fleet", "status"] });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Submission failed");
    }
  };

  return (
    <Card>
      <CardContent className="p-3">
        <div
          data-testid="conversation-thread"
          className="mb-3 max-h-48 space-y-2 overflow-y-auto rounded-md border border-border/50 bg-muted/20 p-2"
        >
          {thread.length === 0 ? (
            <p
              data-testid="conversation-empty"
              className="px-2 py-3 text-center text-xs text-muted-foreground"
            >
              Send your first request to start a thread.
            </p>
          ) : (
            thread.map((m) => (
              <div
                key={m.id}
                data-testid={`conversation-message-${m.id}`}
                data-role={m.role}
                className={
                  m.role === "user"
                    ? "ml-auto max-w-[85%] rounded-md bg-primary/10 px-3 py-2 text-sm text-foreground"
                    : "mr-auto max-w-[85%] rounded-md bg-background px-3 py-2 text-sm text-foreground"
                }
              >
                {m.role === "assistant" && (
                  <p className="mb-1 text-[10px] uppercase tracking-wider text-muted-foreground">
                    Mission Control
                  </p>
                )}
                <p className="whitespace-pre-wrap break-words">{m.content}</p>
              </div>
            ))
          )}
        </div>
        <div className="flex items-center gap-2">
          <ChevronRight
            className={`h-4 w-4 text-muted-foreground transition-transform ${
              open ? "rotate-90" : ""
            }`}
          />
          <Input
            data-testid="conversation-input"
            placeholder="Ask Mission Control…"
            value={text}
            onChange={(e) => setText(e.target.value)}
            onFocus={() => setOpen(true)}
            onBlur={() => setOpen(false)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                void handleSubmit();
              }
            }}
            className="flex-1 bg-transparent"
          />
          <Button
            data-testid="conversation-submit"
            size="sm"
            variant="outline"
            onClick={() => void handleSubmit()}
            disabled={send.isPending || text.trim().length === 0}
            title="Submit to Mission Control"
          >
            {send.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Send className="h-4 w-4" />
            )}
          </Button>
        </div>
        {error && (
          <p
            data-testid="conversation-error"
            className="mt-2 px-2 text-xs text-status-red"
          >
            {error}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
