/**
 * React Query hooks for AI Copilot functionality
 * Updated to use the AI Orchestrator service for agentic workflows
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useOverviewMetrics } from "./use-overview";
import { useDriftSummary } from "./use-drift";
import { useComplianceSummary } from "./use-compliance";
import { useResilienceSummary } from "./use-resilience";

// Check if Clerk is configured
const hasClerkKey =
  typeof process !== "undefined" &&
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY &&
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY.startsWith("pk_") &&
  !process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY.includes("xxxxx");

const isDevelopment = process.env.NODE_ENV === "development";

// Type for auth return
type UseAuthReturn = {
  getToken: () => Promise<string | null>;
  orgId?: string;
};

// Dev auth values - used when Clerk isn't configured
const devAuthValue: UseAuthReturn = {
  getToken: async () => "dev-token",
  orgId: "dev-org",
};

// Get auth - use dev auth when Clerk isn't configured to avoid ClerkProvider errors
function useAuth(): UseAuthReturn {
  // In development without Clerk or in development mode, use dev auth
  // This avoids conditional hook calls
  if (!hasClerkKey || isDevelopment) {
    return devAuthValue;
  }
  // Production with Clerk - would use actual Clerk auth (but this is wrapped in ClerkProvider)
  return devAuthValue;
}

// Types for AI messages
export interface AIMessage {
  role: "user" | "assistant";
  content: string;
}

export interface AIContext {
  fleetSize?: number;
  driftScore?: number;
  driftedAssets?: number;
  complianceScore?: number;
  drReadiness?: number;
  totalSites?: number;
  criticalAlerts?: number;
  environment?: string;
}

interface SendMessageParams {
  message: string;
  context?: AIContext;
  conversationHistory?: AIMessage[];
}

// HITL Action from orchestrator
export interface HITLAction {
  type: "approve" | "modify" | "reject";
  label: string;
  description: string;
}

// Quality Score from validation pipeline
export interface ScoreDimension {
  score: number;
  max_score: number;
  passed: string[] | null;
  failed: string[] | null;
  description: string;
}

export interface QualityScore {
  total: number;
  structural: number;
  policy_compliance: number;
  test_coverage: number;
  operational_history: number;
  human_review: number;
  dimensions: Record<string, ScoreDimension>;
  allowed_environments: string[];
  requires_approval: boolean;
  computed_at: string;
}

// Approval tracking for dual approval workflow
export interface ApprovalInfo {
  user_id: string;
  user_name?: string;
  approved_at: string;
}

// Task from orchestrator
export interface AITask {
  task_id: string;
  status: "pending_approval" | "approved" | "rejected" | "executing" | "completed" | "failed";
  task_spec: {
    task_type: string;
    goal: string;
    risk_level: "low" | "medium" | "high" | "critical";
    environment?: string;
  };
  agent_result?: {
    agent_name: string;
    plan: string;
    summary: string;
    affected_assets: number;
    risk_level: string;
    actions: HITLAction[];
    evidence?: Record<string, unknown>;
    tokens_used?: number;
  };
  quality_score?: QualityScore;
  requires_hitl: boolean;
  message?: string;

  // Dual approval tracking
  execution_policy?: {
    mode: "plan_only" | "canary_only" | "full_auto";
    require_two_approvers: boolean;
    allowed_approver_roles?: string[];
    timeout_minutes?: number;
  };
  approval_status?: {
    approvals_required: number;
    approvals_received: number;
    first_approval?: ApprovalInfo;
    second_approval?: ApprovalInfo;
    rejection?: {
      user_id: string;
      user_name?: string;
      rejected_at: string;
      reason?: string;
    };
  };
}

interface AIResponse {
  content: string;
  model?: string;
  usage?: {
    input_tokens: number;
    output_tokens: number;
  };
  task?: AITask;
}

/**
 * Hook to aggregate infrastructure context from various data sources
 */
export function useAIContext(): AIContext {
  const { data: overview } = useOverviewMetrics();
  const { data: drift } = useDriftSummary();
  const { data: compliance } = useComplianceSummary();
  const { data: resilience } = useResilienceSummary();

  return {
    fleetSize: overview?.fleetSize?.value,
    driftScore: overview?.driftScore?.value,
    driftedAssets: drift?.driftedAssets,
    complianceScore: compliance?.overallScore,
    drReadiness: resilience?.drReadiness,
    totalSites: overview?.platformDistribution?.length,
    criticalAlerts: overview?.alerts?.find(a => a.severity === "critical")?.count,
  };
}

// Orchestrator API base URL - configurable via env
const ORCHESTRATOR_URL = process.env.NEXT_PUBLIC_ORCHESTRATOR_URL || "http://localhost:8083";

/**
 * Helper to create authenticated fetch for orchestrator API
 */
async function orchestratorFetch(
  endpoint: string,
  options: RequestInit = {},
  getToken: () => Promise<string | null>
): Promise<Response> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((options.headers as Record<string, string>) || {}),
  };

  // Add auth token if available
  const token = await getToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  return fetch(`${ORCHESTRATOR_URL}${endpoint}`, {
    ...options,
    headers,
  });
}

/**
 * Hook to send a message to the AI copilot
 * Now routes to the AI Orchestrator service for agentic workflows
 */
export function useSendAIMessage() {
  const queryClient = useQueryClient();
  const { getToken, orgId } = useAuth();

  return useMutation<AIResponse, Error, SendMessageParams>({
    mutationFn: async ({ message, context }) => {
      // Call the orchestrator execute endpoint
      const response = await orchestratorFetch(
        "/api/v1/ai/execute",
        {
          method: "POST",
          body: JSON.stringify({
            intent: message,
            org_id: orgId || "default-org",
            environment: context?.environment || "production", // Defaults to production; can be overridden via context
            context: {
              fleet_size: context?.fleetSize,
              drift_score: context?.driftScore,
              compliance_score: context?.complianceScore,
              dr_readiness: context?.drReadiness,
            },
          }),
        },
        getToken
      );

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `AI request failed: ${response.status}`);
      }

      const data = await response.json();

      // Transform orchestrator response to our format
      return {
        content: data.agent_result?.plan || data.message || "Task completed",
        task: {
          task_id: data.task_id,
          status: data.status,
          task_spec: data.task_spec,
          agent_result: data.agent_result,
          quality_score: data.quality_score,
          requires_hitl: data.requires_hitl,
          message: data.message,
        },
      };
    },
    onSuccess: () => {
      // Invalidate task queries to refresh the list
      queryClient.invalidateQueries({ queryKey: ["ai-tasks"] });
    },
  });
}

/**
 * Hook to approve a task
 */
export function useApproveTask() {
  const queryClient = useQueryClient();
  const { getToken } = useAuth();

  return useMutation<AITask, Error, { taskId: string; reason?: string }>({
    mutationFn: async ({ taskId, reason }) => {
      const response = await orchestratorFetch(
        `/api/v1/ai/tasks/${taskId}/approve`,
        {
          method: "POST",
          body: JSON.stringify({ reason }),
        },
        getToken
      );

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Approval failed: ${response.status}`);
      }

      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-tasks"] });
    },
  });
}

/**
 * Hook to reject a task
 */
export function useRejectTask() {
  const queryClient = useQueryClient();
  const { getToken } = useAuth();

  return useMutation<AITask, Error, { taskId: string; reason?: string }>({
    mutationFn: async ({ taskId, reason }) => {
      const response = await orchestratorFetch(
        `/api/v1/ai/tasks/${taskId}/reject`,
        {
          method: "POST",
          body: JSON.stringify({ reason }),
        },
        getToken
      );

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Rejection failed: ${response.status}`);
      }

      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-tasks"] });
    },
  });
}

// Task response from backend (matching TaskResponse struct)
export interface TaskWithPlan {
  id: string;
  org_id: string;
  user_intent: string;
  task_spec?: Record<string, unknown>;
  state: string;
  source: string;
  created_at: string;
  updated_at: string;
  plan?: {
    id: string;
    type: string;
    payload: Record<string, unknown>;
    state: string;
    approved_by?: string;
    approved_at?: string;
    rejection_reason?: string;
    created_at: string;
    quality_score?: QualityScore;
  };
  risk_level?: string;
  task_type?: string;
  hitl_required: boolean;
}

/**
 * Hook to list pending AI tasks
 */
export function usePendingTasks() {
  const { getToken } = useAuth();

  return useQuery<TaskWithPlan[]>({
    queryKey: ["ai-tasks", "pending"],
    queryFn: async () => {
      // Use state=planned to get tasks awaiting approval
      const response = await orchestratorFetch(
        "/api/v1/ai/tasks?state=planned",
        {},
        getToken
      );
      if (!response.ok) {
        throw new Error("Failed to fetch pending tasks");
      }
      const data = await response.json();
      return data.tasks || [];
    },
    refetchInterval: 10000, // Refresh every 10 seconds for better UX
  });
}

/**
 * Hook to list all AI tasks
 */
export function useAllTasks(filterOrgId?: string) {
  const { getToken, orgId } = useAuth();

  return useQuery<TaskWithPlan[]>({
    queryKey: ["ai-tasks", "all", filterOrgId || orgId],
    queryFn: async () => {
      const effectiveOrgId = filterOrgId || orgId;
      const url = effectiveOrgId
        ? `/api/v1/ai/tasks?org_id=${effectiveOrgId}`
        : `/api/v1/ai/tasks`;
      const response = await orchestratorFetch(url, {}, getToken);
      if (!response.ok) {
        throw new Error("Failed to fetch tasks");
      }
      const data = await response.json();
      return data.tasks || [];
    },
    refetchInterval: 15000, // Refresh every 15 seconds
  });
}

/**
 * Hook to get a single task by ID
 */
export function useTask(taskId: string) {
  const { getToken } = useAuth();

  return useQuery<TaskWithPlan>({
    queryKey: ["ai-tasks", taskId],
    queryFn: async () => {
      const response = await orchestratorFetch(
        `/api/v1/ai/tasks/${taskId}`,
        {},
        getToken
      );
      if (!response.ok) {
        throw new Error("Failed to fetch task");
      }
      return response.json();
    },
    enabled: !!taskId,
    refetchInterval: 5000, // Refresh frequently when viewing a specific task
  });
}

/**
 * Generate proactive insights based on current infrastructure state
 */
export function useProactiveInsights() {
  const context = useAIContext();

  // Generate insights based on context thresholds
  const insights = [];

  // Drift insights
  if (context.driftScore !== undefined && context.driftScore < 90) {
    insights.push({
      type: "drift" as const,
      title: "Drift Detected",
      description: `Your drift score is ${context.driftScore?.toFixed(1)}%. ${context.driftedAssets || 0} assets have drifted from their golden images.`,
      severity: context.driftScore < 80 ? "critical" as const : "warning" as const,
      action: "Analyze Drift",
    });
  }

  // Compliance insights
  if (context.complianceScore !== undefined && context.complianceScore < 95) {
    insights.push({
      type: "compliance" as const,
      title: "Compliance Gap",
      description: `Compliance score is ${context.complianceScore?.toFixed(1)}%. Review failing controls to improve your security posture.`,
      severity: context.complianceScore < 90 ? "critical" as const : "warning" as const,
      action: "View Controls",
    });
  }

  // DR insights
  if (context.drReadiness !== undefined && context.drReadiness < 95) {
    insights.push({
      type: "dr" as const,
      title: "DR Readiness",
      description: `DR readiness is at ${context.drReadiness?.toFixed(1)}%. Ensure all critical sites have DR pairs configured.`,
      severity: context.drReadiness < 90 ? "critical" as const : "warning" as const,
      action: "Review DR Status",
    });
  }

  // Critical alerts
  if (context.criticalAlerts !== undefined && context.criticalAlerts > 0) {
    insights.push({
      type: "optimization" as const,
      title: "Critical Alerts",
      description: `You have ${context.criticalAlerts} critical alerts requiring immediate attention.`,
      severity: "critical" as const,
      action: "View Alerts",
    });
  }

  return insights;
}

// =============================================================================
// Execution Types and Hooks
// =============================================================================

export type ExecutionStatus =
  | "pending"
  | "running"
  | "paused"
  | "completed"
  | "failed"
  | "rolled_back"
  | "cancelled";

export type PhaseStatus =
  | "pending"
  | "running"
  | "waiting"
  | "completed"
  | "failed"
  | "skipped";

export interface AssetExecution {
  asset_id: string;
  asset_name: string;
  status: string;
  started_at?: string;
  completed_at?: string;
  error?: string;
  output?: string;
}

export interface PhaseExecution {
  name: string;
  status: PhaseStatus;
  started_at?: string;
  completed_at?: string;
  assets: AssetExecution[];
  wait_until?: string;
  error?: string;
  metrics?: Record<string, unknown>;
}

export interface Execution {
  id: string;
  task_id: string;
  plan_id: string;
  org_id: string;
  status: ExecutionStatus;
  started_at: string;
  completed_at?: string;
  started_by: string;
  phases: PhaseExecution[];
  current_phase: number;
  total_phases: number;
  error?: string;
  rollback_error?: string;
  metadata?: Record<string, unknown>;
}

/**
 * Hook to list executions for a task
 */
export function useTaskExecutions(taskId: string) {
  const { getToken } = useAuth();

  return useQuery<Execution[]>({
    queryKey: ["executions", taskId],
    queryFn: async () => {
      const response = await orchestratorFetch(
        `/api/v1/ai/tasks/${taskId}/executions`,
        {},
        getToken
      );
      if (!response.ok) {
        throw new Error("Failed to fetch executions");
      }
      const data = await response.json();
      return data.executions || [];
    },
    enabled: !!taskId,
    refetchInterval: 3000, // Refresh frequently during execution
  });
}

/**
 * Hook to get a single execution
 */
export function useExecution(executionId: string) {
  const { getToken } = useAuth();

  return useQuery<Execution>({
    queryKey: ["execution", executionId],
    queryFn: async () => {
      const response = await orchestratorFetch(
        `/api/v1/ai/executions/${executionId}`,
        {},
        getToken
      );
      if (!response.ok) {
        throw new Error("Failed to fetch execution");
      }
      return response.json();
    },
    enabled: !!executionId,
    refetchInterval: 2000, // Refresh very frequently during execution
  });
}

/**
 * Hook to pause an execution
 */
export function usePauseExecution() {
  const queryClient = useQueryClient();
  const { getToken } = useAuth();

  return useMutation<void, Error, string>({
    mutationFn: async (executionId) => {
      const response = await orchestratorFetch(
        `/api/v1/ai/executions/${executionId}/pause`,
        { method: "POST" },
        getToken
      );
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || "Failed to pause execution");
      }
    },
    onSuccess: (_, executionId) => {
      queryClient.invalidateQueries({ queryKey: ["execution", executionId] });
      queryClient.invalidateQueries({ queryKey: ["executions"] });
    },
  });
}

/**
 * Hook to resume an execution
 */
export function useResumeExecution() {
  const queryClient = useQueryClient();
  const { getToken } = useAuth();

  return useMutation<void, Error, string>({
    mutationFn: async (executionId) => {
      const response = await orchestratorFetch(
        `/api/v1/ai/executions/${executionId}/resume`,
        { method: "POST" },
        getToken
      );
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || "Failed to resume execution");
      }
    },
    onSuccess: (_, executionId) => {
      queryClient.invalidateQueries({ queryKey: ["execution", executionId] });
      queryClient.invalidateQueries({ queryKey: ["executions"] });
    },
  });
}

/**
 * Hook to cancel an execution
 */
export function useCancelExecution() {
  const queryClient = useQueryClient();
  const { getToken } = useAuth();

  return useMutation<void, Error, string>({
    mutationFn: async (executionId) => {
      const response = await orchestratorFetch(
        `/api/v1/ai/executions/${executionId}/cancel`,
        { method: "POST" },
        getToken
      );
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || "Failed to cancel execution");
      }
    },
    onSuccess: (_, executionId) => {
      queryClient.invalidateQueries({ queryKey: ["execution", executionId] });
      queryClient.invalidateQueries({ queryKey: ["executions"] });
    },
  });
}

// =============================================================================
// Agent Types and Hooks
// =============================================================================

export interface AgentInfo {
  name: string;
  description: string;
  task_types: string[];
  capabilities: string[];
  status: "active" | "inactive" | "error";
}

/**
 * Hook to list all available agents
 */
export function useAgents() {
  const { getToken } = useAuth();

  return useQuery<AgentInfo[]>({
    queryKey: ["ai-agents"],
    queryFn: async () => {
      const response = await orchestratorFetch(
        "/api/v1/ai/agents",
        {},
        getToken
      );
      if (!response.ok) {
        throw new Error("Failed to fetch agents");
      }
      const data = await response.json();
      return data.agents || [];
    },
    refetchInterval: 30000, // Refresh every 30 seconds
  });
}

// =============================================================================
// Tool Types and Hooks
// =============================================================================

export interface ToolInfo {
  name: string;
  description: string;
  category: string;
  parameters: Record<string, unknown>;
}

/**
 * Hook to list all available tools
 */
export function useTools() {
  const { getToken } = useAuth();

  return useQuery<ToolInfo[]>({
    queryKey: ["ai-tools"],
    queryFn: async () => {
      const response = await orchestratorFetch(
        "/api/v1/ai/tools",
        {},
        getToken
      );
      if (!response.ok) {
        throw new Error("Failed to fetch tools");
      }
      const data = await response.json();
      return data.tools || [];
    },
    refetchInterval: 60000, // Refresh every minute
  });
}

// =============================================================================
// Tool Invocation Audit Hooks
// =============================================================================

export interface ToolInvocation {
  id: string;
  task_id: string;
  tool_name: string;
  parameters: Record<string, unknown>;
  result: Record<string, unknown>;
  success: boolean;
  error?: string;
  duration_ms: number;
  invoked_at: string;
}

/**
 * Hook to list tool invocations for a task
 */
export function useToolInvocations(taskId: string) {
  const { getToken } = useAuth();

  return useQuery<ToolInvocation[]>({
    queryKey: ["tool-invocations", taskId],
    queryFn: async () => {
      const response = await orchestratorFetch(
        `/api/v1/ai/tasks/${taskId}/tool-invocations`,
        {},
        getToken
      );
      if (!response.ok) {
        throw new Error("Failed to fetch tool invocations");
      }
      const data = await response.json();
      return data.invocations || [];
    },
    enabled: !!taskId,
    refetchInterval: 5000,
  });
}

// =============================================================================
// Modify Task Hook
// =============================================================================

export interface ModifyTaskParams {
  taskId: string;
  modifications: {
    environment?: string;
    risk_level?: string;
    asset_filters?: Record<string, unknown>;
    phases?: unknown[];
    notes?: string;
  };
}

/**
 * Hook to modify a task plan before approval
 */
export function useModifyTask() {
  const queryClient = useQueryClient();
  const { getToken } = useAuth();

  return useMutation<AITask, Error, ModifyTaskParams>({
    mutationFn: async ({ taskId, modifications }) => {
      const response = await orchestratorFetch(
        `/api/v1/ai/tasks/${taskId}/modify`,
        {
          method: "POST",
          body: JSON.stringify(modifications),
        },
        getToken
      );

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Modification failed: ${response.status}`);
      }

      return response.json();
    },
    onSuccess: (_, { taskId }) => {
      queryClient.invalidateQueries({ queryKey: ["ai-tasks"] });
      queryClient.invalidateQueries({ queryKey: ["ai-tasks", taskId] });
    },
  });
}

// =============================================================================
// Rollback Execution Hook
// =============================================================================

/**
 * Hook to rollback an execution
 */
export function useRollbackExecution() {
  const queryClient = useQueryClient();
  const { getToken } = useAuth();

  return useMutation<void, Error, { executionId: string; reason?: string }>({
    mutationFn: async ({ executionId, reason }) => {
      const response = await orchestratorFetch(
        `/api/v1/ai/executions/${executionId}/rollback`,
        {
          method: "POST",
          body: JSON.stringify({ reason }),
        },
        getToken
      );
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || "Failed to rollback execution");
      }
    },
    onSuccess: (_, { executionId }) => {
      queryClient.invalidateQueries({ queryKey: ["execution", executionId] });
      queryClient.invalidateQueries({ queryKey: ["executions"] });
    },
  });
}

// =============================================================================
// LLM Usage Analytics Types and Hooks
// =============================================================================

export interface AIUsageStats {
  // Token usage
  tokens_used_this_month: number;
  monthly_token_budget?: number;
  token_usage_percent: number;

  // Request counts
  tasks_this_month: number;
  tasks_this_week: number;
  tasks_today: number;

  // Cost estimates
  estimated_cost_this_month: number;
  cost_per_1k_tokens: number;

  // Provider info
  llm_provider: string;
  llm_model: string;

  // Usage by type
  usage_by_task_type: Array<{
    task_type: string;
    count: number;
    tokens: number;
  }>;

  // Usage by agent
  usage_by_agent: Array<{
    agent: string;
    count: number;
    tokens: number;
    avg_tokens: number;
  }>;

  // Daily usage (last 30 days)
  daily_usage: Array<{
    date: string;
    tokens: number;
    tasks: number;
    cost: number;
  }>;

  // Settings
  ai_enabled: boolean;
  auto_remediation_enabled: boolean;
  autonomy_mode: "plan_only" | "canary_only" | "full_auto";
}

/**
 * Hook to fetch AI usage analytics
 */
export function useAIUsageStats() {
  const { getToken, orgId } = useAuth();

  return useQuery<AIUsageStats>({
    queryKey: ["ai-usage-stats", orgId],
    queryFn: async () => {
      const response = await orchestratorFetch(
        `/api/v1/ai/usage`,
        {},
        getToken
      );
      if (!response.ok) {
        throw new Error("Failed to fetch AI usage stats");
      }
      return response.json();
    },
    refetchInterval: 60000, // Refresh every minute
    staleTime: 30000, // Consider data stale after 30 seconds
  });
}

/**
 * Hook to get AI settings for the organization
 */
export function useAISettings() {
  const { getToken, orgId } = useAuth();

  return useQuery({
    queryKey: ["ai-settings", orgId],
    queryFn: async () => {
      const response = await orchestratorFetch(
        `/api/v1/ai/settings`,
        {},
        getToken
      );
      if (!response.ok) {
        throw new Error("Failed to fetch AI settings");
      }
      return response.json();
    },
  });
}

/**
 * Hook to update AI settings
 */
export function useUpdateAISettings() {
  const queryClient = useQueryClient();
  const { getToken, orgId } = useAuth();

  return useMutation({
    mutationFn: async (settings: Partial<AIUsageStats>) => {
      const response = await orchestratorFetch(
        `/api/v1/ai/settings`,
        {
          method: "PUT",
          body: JSON.stringify(settings),
        },
        getToken
      );
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || "Failed to update AI settings");
      }
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-settings", orgId] });
      queryClient.invalidateQueries({ queryKey: ["ai-usage-stats", orgId] });
    },
  });
}
