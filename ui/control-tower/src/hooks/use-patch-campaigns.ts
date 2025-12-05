"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";

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
  if (!hasClerkKey || isDevelopment) {
    return devAuthValue;
  }
  return devAuthValue;
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

// Types
export type PatchCampaignStatus =
  | "draft"
  | "pending_approval"
  | "approved"
  | "scheduled"
  | "in_progress"
  | "paused"
  | "completed"
  | "failed"
  | "rolled_back"
  | "cancelled";

export type RolloutStrategy = "immediate" | "canary" | "rolling" | "blue_green";

export interface PatchCampaign {
  id: string;
  org_id: string;
  name: string;
  description?: string;
  campaign_type: string;
  cve_alert_ids?: string[];
  status: PatchCampaignStatus;
  requires_approval: boolean;
  approved_by?: string;
  approved_at?: string;
  rollout_strategy: RolloutStrategy;
  canary_percentage?: number;
  wave_percentage?: number;
  failure_threshold_percentage?: number;
  health_check_enabled: boolean;
  auto_rollback_enabled: boolean;
  total_assets: number;
  pending_assets: number;
  in_progress_assets: number;
  completed_assets: number;
  failed_assets: number;
  skipped_assets: number;
  scheduled_start_at?: string;
  started_at?: string;
  completed_at?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  phases?: PatchCampaignPhase[];
}

export interface PatchCampaignPhase {
  id: string;
  campaign_id: string;
  phase_number: number;
  name: string;
  phase_type: string;
  target_percentage: number;
  status: string;
  total_assets: number;
  completed_assets: number;
  failed_assets: number;
  health_check_passed?: boolean;
  started_at?: string;
  completed_at?: string;
}

export interface PatchCampaignAsset {
  id: string;
  campaign_id: string;
  asset_id: string;
  asset_name: string;
  platform: string;
  status: string;
  before_version?: string;
  after_version?: string;
  error_message?: string;
}

export interface PatchCampaignProgress {
  campaign_id: string;
  status: string;
  total_assets: number;
  completed_assets: number;
  failed_assets: number;
  skipped_assets: number;
  completion_percentage: number;
  failure_percentage: number;
  total_phases: number;
  completed_phases: number;
  current_phase: string;
  current_phase_progress: number;
  estimated_completion?: string;
  started_at?: string;
  elapsed_time_minutes?: number;
}

export interface PatchCampaignSummary {
  total_campaigns: number;
  active_campaigns: number;
  completed_campaigns: number;
  failed_campaigns: number;
  total_assets_patched: number;
  total_rollbacks: number;
  success_rate: number;
}

// Query parameters
export interface PatchCampaignListParams {
  status?: PatchCampaignStatus;
  campaign_type?: string;
  page?: number;
  page_size?: number;
}

// Response types
interface PatchCampaignListResponse {
  campaigns: PatchCampaign[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// Request types
export interface CreatePatchCampaignRequest {
  name: string;
  description?: string;
  campaign_type: string;
  cve_alert_ids?: string[];
  rollout_strategy: RolloutStrategy;
  canary_percentage?: number;
  wave_percentage?: number;
  failure_threshold_percentage?: number;
  health_check_enabled?: boolean;
  auto_rollback_enabled?: boolean;
  requires_approval?: boolean;
  scheduled_start_at?: string;
  target_asset_ids?: string[];
}

export interface ApprovePatchCampaignRequest {
  approved_by: string;
  comment?: string;
}

export interface RejectPatchCampaignRequest {
  rejected_by: string;
  reason: string;
}

export interface TriggerRollbackRequest {
  scope: "all" | "partial" | "phase";
  reason: string;
  asset_ids?: string[];
  phase_id?: string;
}

// List patch campaigns
export function usePatchCampaigns(params: PatchCampaignListParams = {}) {
  const { getToken } = useAuth();

  return useQuery({
    queryKey: ["patch-campaigns", params],
    queryFn: async (): Promise<PatchCampaignListResponse> => {
      // Build query string
      const searchParams = new URLSearchParams();
      if (params.status) searchParams.set("status", params.status);
      if (params.campaign_type) searchParams.set("campaign_type", params.campaign_type);
      if (params.page) searchParams.set("page", params.page.toString());
      if (params.page_size) searchParams.set("page_size", params.page_size.toString());

      const query = searchParams.toString();
      const url = query ? `/api/v1/patch-campaigns?${query}` : "/api/v1/patch-campaigns";

      const response = await orchestratorFetch(url, { method: "GET" }, getToken);

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to fetch patch campaigns: ${response.status}`);
      }

      return response.json();
    },
  });
}

// Get patch campaign summary
export function usePatchCampaignSummary() {
  const { getToken } = useAuth();

  return useQuery({
    queryKey: ["patch-campaign-summary"],
    queryFn: async (): Promise<PatchCampaignSummary> => {
      const response = await orchestratorFetch("/api/v1/patch-campaigns/summary", { method: "GET" }, getToken);

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to fetch patch campaign summary: ${response.status}`);
      }

      return response.json();
    },
  });
}

// Get single patch campaign
export function usePatchCampaign(campaignId: string | null) {
  const { getToken } = useAuth();

  return useQuery({
    queryKey: ["patch-campaign", campaignId],
    queryFn: async (): Promise<PatchCampaign> => {
      if (!campaignId) throw new Error("Campaign ID is required");

      const response = await orchestratorFetch(`/api/v1/patch-campaigns/${campaignId}`, { method: "GET" }, getToken);

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to fetch patch campaign: ${response.status}`);
      }

      return response.json();
    },
    enabled: !!campaignId,
  });
}

// Get patch campaign phases
export function usePatchCampaignPhases(campaignId: string | null) {
  const { getToken } = useAuth();

  return useQuery({
    queryKey: ["patch-campaign-phases", campaignId],
    queryFn: async (): Promise<{ campaign_id: string; phases: PatchCampaignPhase[] }> => {
      if (!campaignId) throw new Error("Campaign ID is required");

      const response = await orchestratorFetch(`/api/v1/patch-campaigns/${campaignId}/phases`, { method: "GET" }, getToken);

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to fetch campaign phases: ${response.status}`);
      }

      return response.json();
    },
    enabled: !!campaignId,
  });
}

// Get patch campaign assets
export function usePatchCampaignAssets(campaignId: string | null, status?: string, phaseId?: string) {
  const { getToken } = useAuth();

  return useQuery({
    queryKey: ["patch-campaign-assets", campaignId, status, phaseId],
    queryFn: async (): Promise<{ campaign_id: string; assets: PatchCampaignAsset[]; total: number }> => {
      if (!campaignId) throw new Error("Campaign ID is required");

      const searchParams = new URLSearchParams();
      if (status) searchParams.set("status", status);
      if (phaseId) searchParams.set("phase_id", phaseId);

      const query = searchParams.toString();
      const url = query
        ? `/api/v1/patch-campaigns/${campaignId}/assets?${query}`
        : `/api/v1/patch-campaigns/${campaignId}/assets`;

      const response = await orchestratorFetch(url, { method: "GET" }, getToken);

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to fetch campaign assets: ${response.status}`);
      }

      return response.json();
    },
    enabled: !!campaignId,
  });
}

// Get patch campaign progress
export function usePatchCampaignProgress(campaignId: string | null) {
  const { getToken } = useAuth();

  return useQuery({
    queryKey: ["patch-campaign-progress", campaignId],
    queryFn: async (): Promise<PatchCampaignProgress> => {
      if (!campaignId) throw new Error("Campaign ID is required");

      const response = await orchestratorFetch(`/api/v1/patch-campaigns/${campaignId}/progress`, { method: "GET" }, getToken);

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to fetch campaign progress: ${response.status}`);
      }

      return response.json();
    },
    enabled: !!campaignId,
    refetchInterval: 5000, // Poll every 5 seconds for active campaigns
  });
}

// Create patch campaign
export function useCreatePatchCampaign() {
  const queryClient = useQueryClient();
  const { getToken } = useAuth();

  return useMutation({
    mutationFn: async (data: CreatePatchCampaignRequest): Promise<PatchCampaign> => {
      const response = await orchestratorFetch(
        "/api/v1/patch-campaigns",
        {
          method: "POST",
          body: JSON.stringify(data),
        },
        getToken
      );

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to create patch campaign: ${response.status}`);
      }

      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["patch-campaigns"] });
      queryClient.invalidateQueries({ queryKey: ["patch-campaign-summary"] });
    },
  });
}

// Approve patch campaign
export function useApprovePatchCampaign() {
  const queryClient = useQueryClient();
  const { getToken } = useAuth();

  return useMutation({
    mutationFn: async ({ campaignId, data }: { campaignId: string; data: ApprovePatchCampaignRequest }): Promise<PatchCampaign> => {
      const response = await orchestratorFetch(
        `/api/v1/patch-campaigns/${campaignId}/approve`,
        {
          method: "POST",
          body: JSON.stringify(data),
        },
        getToken
      );

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to approve patch campaign: ${response.status}`);
      }

      return response.json();
    },
    onSuccess: (_, { campaignId }) => {
      queryClient.invalidateQueries({ queryKey: ["patch-campaigns"] });
      queryClient.invalidateQueries({ queryKey: ["patch-campaign", campaignId] });
    },
  });
}

// Reject patch campaign
export function useRejectPatchCampaign() {
  const queryClient = useQueryClient();
  const { getToken } = useAuth();

  return useMutation({
    mutationFn: async ({ campaignId, data }: { campaignId: string; data: RejectPatchCampaignRequest }): Promise<PatchCampaign> => {
      const response = await orchestratorFetch(
        `/api/v1/patch-campaigns/${campaignId}/reject`,
        {
          method: "POST",
          body: JSON.stringify(data),
        },
        getToken
      );

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to reject patch campaign: ${response.status}`);
      }

      return response.json();
    },
    onSuccess: (_, { campaignId }) => {
      queryClient.invalidateQueries({ queryKey: ["patch-campaigns"] });
      queryClient.invalidateQueries({ queryKey: ["patch-campaign", campaignId] });
    },
  });
}

// Start patch campaign
export function useStartPatchCampaign() {
  const queryClient = useQueryClient();
  const { getToken } = useAuth();

  return useMutation({
    mutationFn: async (campaignId: string): Promise<PatchCampaign> => {
      const response = await orchestratorFetch(
        `/api/v1/patch-campaigns/${campaignId}/start`,
        { method: "POST" },
        getToken
      );

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to start patch campaign: ${response.status}`);
      }

      return response.json();
    },
    onSuccess: (_, campaignId) => {
      queryClient.invalidateQueries({ queryKey: ["patch-campaigns"] });
      queryClient.invalidateQueries({ queryKey: ["patch-campaign", campaignId] });
    },
  });
}

// Pause patch campaign
export function usePausePatchCampaign() {
  const queryClient = useQueryClient();
  const { getToken } = useAuth();

  return useMutation({
    mutationFn: async (campaignId: string): Promise<PatchCampaign> => {
      const response = await orchestratorFetch(
        `/api/v1/patch-campaigns/${campaignId}/pause`,
        { method: "POST" },
        getToken
      );

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to pause patch campaign: ${response.status}`);
      }

      return response.json();
    },
    onSuccess: (_, campaignId) => {
      queryClient.invalidateQueries({ queryKey: ["patch-campaigns"] });
      queryClient.invalidateQueries({ queryKey: ["patch-campaign", campaignId] });
    },
  });
}

// Resume patch campaign
export function useResumePatchCampaign() {
  const queryClient = useQueryClient();
  const { getToken } = useAuth();

  return useMutation({
    mutationFn: async (campaignId: string): Promise<PatchCampaign> => {
      const response = await orchestratorFetch(
        `/api/v1/patch-campaigns/${campaignId}/resume`,
        { method: "POST" },
        getToken
      );

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to resume patch campaign: ${response.status}`);
      }

      return response.json();
    },
    onSuccess: (_, campaignId) => {
      queryClient.invalidateQueries({ queryKey: ["patch-campaigns"] });
      queryClient.invalidateQueries({ queryKey: ["patch-campaign", campaignId] });
    },
  });
}

// Cancel patch campaign
export function useCancelPatchCampaign() {
  const queryClient = useQueryClient();
  const { getToken } = useAuth();

  return useMutation({
    mutationFn: async (campaignId: string): Promise<PatchCampaign> => {
      const response = await orchestratorFetch(
        `/api/v1/patch-campaigns/${campaignId}/cancel`,
        { method: "POST" },
        getToken
      );

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to cancel patch campaign: ${response.status}`);
      }

      return response.json();
    },
    onSuccess: (_, campaignId) => {
      queryClient.invalidateQueries({ queryKey: ["patch-campaigns"] });
      queryClient.invalidateQueries({ queryKey: ["patch-campaign", campaignId] });
    },
  });
}

// Rollback patch campaign
export function useRollbackPatchCampaign() {
  const queryClient = useQueryClient();
  const { getToken } = useAuth();

  return useMutation({
    mutationFn: async ({ campaignId, data }: { campaignId: string; data: TriggerRollbackRequest }): Promise<{ campaign: PatchCampaign; message: string }> => {
      const response = await orchestratorFetch(
        `/api/v1/patch-campaigns/${campaignId}/rollback`,
        {
          method: "POST",
          body: JSON.stringify(data),
        },
        getToken
      );

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to rollback patch campaign: ${response.status}`);
      }

      return response.json();
    },
    onSuccess: (_, { campaignId }) => {
      queryClient.invalidateQueries({ queryKey: ["patch-campaigns"] });
      queryClient.invalidateQueries({ queryKey: ["patch-campaign", campaignId] });
    },
  });
}
