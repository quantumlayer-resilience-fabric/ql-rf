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
import { ChevronRight, Lock, Bot, Activity, Send, Loader2 } from "lucide-react";
import { useSendAIMessage } from "@/hooks/use-ai";

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
  tool_invocations_today: number;
  llm_spend_today_cents: number;
  llm_spend_budget_cents: number;
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
  const events = status?.recent_invocations ?? [];
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
            events.map((e) => {
              const owner = agentByTool[e.tool_name];
              const display = owner ? AGENT_DISPLAY[owner]?.label ?? owner : "—";
              return (
                <div
                  key={e.task_id + e.tool_name + e.created_at}
                  data-testid={`activity-${e.tool_name}`}
                  className="flex items-center justify-between gap-3 rounded-md px-3 py-2 text-sm hover:bg-muted/50"
                >
                  <div className="flex min-w-0 items-center gap-3">
                    <span className="font-mono text-xs text-muted-foreground">
                      {e.created_at.slice(11, 16)}
                    </span>
                    <span className="text-muted-foreground">{display}</span>
                    <span className="font-mono text-foreground">{e.tool_name}</span>
                  </div>
                  <div className="flex items-center gap-3 text-xs">
                    <span className={`uppercase ${riskTone(e.risk_level)}`}>
                      {e.risk_level.replace(/_/g, " ")}
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
              {decisions.map((d) => (
                <div
                  key={d.plan_id}
                  data-testid={`pending-${d.plan_id}`}
                  className="rounded-md border bg-card p-3"
                >
                  <div className="mb-2 text-sm font-medium text-foreground">
                    {d.user_intent}
                  </div>
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
                  </div>
                  <div className="flex flex-wrap gap-1">
                    <Button
                      size="sm"
                      variant="outline"
                      disabled
                      title="Approval is read-only in Phase A"
                    >
                      Approve
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      disabled
                      title="Approval is read-only in Phase A"
                    >
                      Modify
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      disabled
                      title="Approval is read-only in Phase A"
                    >
                      Reject
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

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
// Conversation dock (collapsed by default; expand-on-focus)
// -----------------------------------------------------------------------------

function ConversationDock() {
  const [open, setOpen] = useState(false);
  const [text, setText] = useState("");
  const [error, setError] = useState<string | null>(null);
  const queryClient = useQueryClient();
  const send = useSendAIMessage();

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
