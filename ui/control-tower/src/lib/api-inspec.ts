/**
 * InSpec API Client for QL-RF Control Tower
 * Provides typed API client for InSpec compliance scanning operations
 */

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

// Token getter function - set by the auth provider (Clerk)
let getAuthToken: (() => Promise<string | null>) | null = null;

// Promise that resolves when auth is ready
let authReadyResolve: (() => void) | null = null;
const authReadyPromise = new Promise<void>((resolve) => {
  authReadyResolve = resolve;
});

/**
 * Set the auth token getter function.
 * Called from a client component that has access to Clerk's useAuth.
 */
export function setAuthTokenGetter(getter: () => Promise<string | null>) {
  getAuthToken = getter;
  if (authReadyResolve) {
    authReadyResolve();
    authReadyResolve = null;
  }
}

/**
 * Wait for auth to be ready before making API calls.
 */
async function waitForAuth(timeoutMs = 5000): Promise<void> {
  if (getAuthToken) return;
  await Promise.race([
    authReadyPromise,
    new Promise<void>((resolve) => setTimeout(resolve, timeoutMs)),
  ]);
}

// =============================================================================
// Types
// =============================================================================

export interface InSpecProfile {
  id: string;
  name: string;
  version: string;
  title: string;
  maintainer: string;
  summary: string;
  frameworkId: string;
  framework?: string;
  profileUrl: string;
  platforms: string[];
  controlCount?: number;
  createdAt: string;
  updatedAt: string;
}

export interface InSpecRun {
  id: string;
  orgId: string;
  assetId: string;
  assetName?: string;
  profileId: string;
  profileName?: string;
  framework?: string;
  status: "pending" | "running" | "completed" | "failed" | "cancelled";
  startedAt?: string;
  completedAt?: string;
  duration?: number;
  totalTests: number;
  passedTests: number;
  failedTests: number;
  skippedTests: number;
  passRate?: number;
  errorMessage?: string;
  rawOutput?: string;
  createdAt: string;
  updatedAt: string;
}

export interface InSpecResult {
  id: string;
  runId: string;
  controlId: string;
  controlTitle: string;
  status: "passed" | "failed" | "skipped" | "error";
  message?: string;
  resource?: string;
  sourceLocation?: string;
  runTime: number;
  codeDescription?: string;
  impact?: number;
  createdAt: string;
}

export interface InSpecEvidence {
  controlId: string;
  controlTitle: string;
  status: "passed" | "failed" | "skipped" | "error";
  message?: string;
  resource?: string;
  sourceLocation?: string;
  codeDescription?: string;
  impact?: number;
  recommendations?: string[];
}

export interface ScanSchedule {
  id: string;
  profileId: string;
  profileName?: string;
  assetId?: string;
  assetName?: string;
  cronExpression: string;
  nextRunAt?: string;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface ControlMapping {
  id: string;
  inspecControlId: string;
  complianceControlId: string;
  profileId: string;
  mappingConfidence: number;
  notes?: string;
  createdAt: string;
  updatedAt: string;
}

// Response types
export interface ProfileListResponse {
  profiles: InSpecProfile[];
}

export interface RunListResponse {
  runs: InSpecRun[];
  limit: number;
  offset: number;
}

export interface RunResultsResponse {
  run: InSpecRun;
  results: InSpecResult[];
}

export interface ControlMappingListResponse {
  mappings: ControlMapping[];
}

// Request types
export interface TriggerScanRequest {
  profileId: string;
  assetId: string;
}

export interface CreateScheduleRequest {
  profileId: string;
  assetId?: string;
  cronExpression: string;
  enabled?: boolean;
}

// API Error type
export class ApiError extends Error {
  constructor(
    public status: number,
    public statusText: string,
    message: string
  ) {
    super(message);
    this.name = "ApiError";
  }
}

// =============================================================================
// API Client
// =============================================================================

async function apiFetch<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${API_BASE_URL}${endpoint}`;

  await waitForAuth();

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((options.headers as Record<string, string>) || {}),
  };

  if (getAuthToken) {
    const token = await getAuthToken();
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
  }

  const response = await fetch(url, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const errorBody = await response.text();
    throw new ApiError(
      response.status,
      response.statusText,
      errorBody || `API request failed: ${response.statusText}`
    );
  }

  const text = await response.text();
  if (!text) {
    return {} as T;
  }

  return JSON.parse(text) as T;
}

// Transform backend response to frontend format
function transformProfile(backend: {
  profile_id?: string;
  id?: string;
  name: string;
  version: string;
  title: string;
  maintainer?: string;
  summary?: string;
  framework_id?: string;
  framework?: string;
  profile_url?: string;
  platforms?: string[];
  control_count?: number;
  created_at?: string;
  updated_at?: string;
}): InSpecProfile {
  return {
    id: backend.profile_id || backend.id || "",
    name: backend.name,
    version: backend.version,
    title: backend.title,
    maintainer: backend.maintainer || "",
    summary: backend.summary || "",
    frameworkId: backend.framework_id || "",
    framework: backend.framework,
    profileUrl: backend.profile_url || "",
    platforms: backend.platforms || [],
    controlCount: backend.control_count,
    createdAt: backend.created_at || new Date().toISOString(),
    updatedAt: backend.updated_at || new Date().toISOString(),
  };
}

function transformRun(backend: {
  id?: string;
  run_id?: string;
  org_id?: string;
  asset_id?: string;
  asset_name?: string;
  profile_id?: string;
  profile_name?: string;
  framework?: string;
  status: string;
  started_at?: string;
  completed_at?: string;
  duration?: number;
  total_tests: number;
  passed_tests: number;
  failed_tests: number;
  skipped_tests: number;
  pass_rate?: number;
  error_message?: string;
  raw_output?: string;
  created_at: string;
  updated_at: string;
}): InSpecRun {
  return {
    id: backend.id || backend.run_id || "",
    orgId: backend.org_id || "",
    assetId: backend.asset_id || "",
    assetName: backend.asset_name,
    profileId: backend.profile_id || "",
    profileName: backend.profile_name,
    framework: backend.framework,
    status: backend.status as InSpecRun["status"],
    startedAt: backend.started_at,
    completedAt: backend.completed_at,
    duration: backend.duration,
    totalTests: backend.total_tests,
    passedTests: backend.passed_tests,
    failedTests: backend.failed_tests,
    skippedTests: backend.skipped_tests,
    passRate: backend.pass_rate,
    errorMessage: backend.error_message,
    rawOutput: backend.raw_output,
    createdAt: backend.created_at,
    updatedAt: backend.updated_at,
  };
}

function transformResult(backend: {
  id: string;
  run_id: string;
  control_id: string;
  control_title: string;
  status: string;
  message?: string;
  resource?: string;
  source_location?: string;
  run_time: number;
  code_description?: string;
  created_at: string;
}): InSpecResult {
  return {
    id: backend.id,
    runId: backend.run_id,
    controlId: backend.control_id,
    controlTitle: backend.control_title,
    status: backend.status as InSpecResult["status"],
    message: backend.message,
    resource: backend.resource,
    sourceLocation: backend.source_location,
    runTime: backend.run_time,
    codeDescription: backend.code_description,
    createdAt: backend.created_at,
  };
}

// =============================================================================
// API Functions
// =============================================================================

export const inspecApi = {
  // Profiles
  getProfiles: async (): Promise<InSpecProfile[]> => {
    const response = await apiFetch<{ profiles: any[] }>("/inspec/profiles");
    return (response.profiles || []).map(transformProfile);
  },

  getProfile: async (id: string): Promise<InSpecProfile> => {
    const response = await apiFetch<any>(`/inspec/profiles/${id}`);
    return transformProfile(response);
  },

  // Runs
  triggerScan: async (data: TriggerScanRequest): Promise<InSpecRun> => {
    const response = await apiFetch<any>("/inspec/run", {
      method: "POST",
      body: JSON.stringify({
        profile_id: data.profileId,
        asset_id: data.assetId,
      }),
    });
    return transformRun(response);
  },

  getScans: async (params?: {
    limit?: number;
    offset?: number;
  }): Promise<{ runs: InSpecRun[]; limit: number; offset: number }> => {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set("limit", String(params.limit));
    if (params?.offset) searchParams.set("offset", String(params.offset));
    const query = searchParams.toString();

    const response = await apiFetch<{
      runs: any[];
      limit: number;
      offset: number;
    }>(`/inspec/runs${query ? `?${query}` : ""}`);

    return {
      runs: (response.runs || []).map(transformRun),
      limit: response.limit,
      offset: response.offset,
    };
  },

  getScan: async (id: string): Promise<InSpecRun> => {
    const response = await apiFetch<any>(`/inspec/runs/${id}`);
    return transformRun(response);
  },

  getScanResults: async (scanId: string): Promise<RunResultsResponse> => {
    const response = await apiFetch<{
      run: any;
      results: any[];
    }>(`/inspec/runs/${scanId}/results`);

    return {
      run: transformRun(response.run),
      results: (response.results || []).map(transformResult),
    };
  },

  cancelScan: async (scanId: string): Promise<void> => {
    await apiFetch(`/inspec/runs/${scanId}/cancel`, {
      method: "POST",
    });
  },

  // Control Mappings
  getControlMappings: async (profileId: string): Promise<ControlMapping[]> => {
    const response = await apiFetch<{ mappings: any[] }>(
      `/inspec/profiles/${profileId}/mappings`
    );
    return (response.mappings || []).map((m) => ({
      id: m.id,
      inspecControlId: m.inspec_control_id,
      complianceControlId: m.compliance_control_id,
      profileId: m.profile_id,
      mappingConfidence: m.mapping_confidence,
      notes: m.notes,
      createdAt: m.created_at,
      updatedAt: m.updated_at,
    }));
  },

  // Schedules (if implemented in backend)
  getSchedules: async (): Promise<ScanSchedule[]> => {
    // Placeholder - implement when backend supports schedules
    return [];
  },

  createSchedule: async (data: CreateScheduleRequest): Promise<ScanSchedule> => {
    // Placeholder - implement when backend supports schedules
    throw new Error("Not implemented");
  },

  deleteSchedule: async (id: string): Promise<void> => {
    // Placeholder - implement when backend supports schedules
    throw new Error("Not implemented");
  },
};

export default inspecApi;
