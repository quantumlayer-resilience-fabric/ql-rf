/**
 * API Client for QL-RF Control Tower
 *
 * Provides typed API client for communicating with the backend services.
 * Includes Clerk authentication token handling.
 */

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

// Token getter function - set by the auth provider
let getAuthToken: (() => Promise<string | null>) | null = null;

/**
 * Set the auth token getter function.
 * Called from a client component that has access to Clerk's useAuth.
 */
export function setAuthTokenGetter(getter: () => Promise<string | null>) {
  getAuthToken = getter;
}

// Types based on backend contracts
export interface Asset {
  id: string;
  hostname: string;
  siteId: string;
  siteName: string;
  platform: "aws" | "azure" | "gcp" | "vsphere" | "k8s" | "baremetal";
  environment: "production" | "staging" | "development" | "dr";
  currentImageId: string;
  currentImageVersion: string;
  goldenImageId: string;
  goldenImageVersion: string;
  isDrifted: boolean;
  driftDetectedAt?: string;
  lastScannedAt: string;
  metadata: Record<string, string>;
  createdAt: string;
  updatedAt: string;
}

export interface Image {
  id: string;
  familyId: string;
  familyName: string;
  version: string;
  description: string;
  status: "production" | "staging" | "deprecated" | "pending";
  platforms: Array<"aws" | "azure" | "gcp" | "vsphere" | "k8s">;
  compliance: {
    cis: boolean;
    slsaLevel: number;
    cosignSigned: boolean;
  };
  deployedCount: number;
  createdAt: string;
  createdBy: string;
  promotedAt?: string;
  promotedBy?: string;
  deprecatedAt?: string;
}

export interface ImageFamily {
  id: string;
  name: string;
  description: string;
  owner: string;
  latestVersion: string;
  status: "production" | "staging" | "deprecated" | "pending";
  totalDeployed: number;
  versions: Image[];
  createdAt: string;
  updatedAt: string;
}

export interface Site {
  id: string;
  name: string;
  region: string;
  platform: "aws" | "azure" | "gcp" | "vsphere" | "k8s";
  environment: "production" | "staging" | "development" | "dr";
  assetCount: number;
  compliantCount: number;
  driftedCount: number;
  coveragePercentage: number;
  status: "healthy" | "warning" | "critical";
  lastSyncAt: string;
  drPaired?: string;
  metadata: Record<string, string>;
}

export interface DriftSummary {
  totalAssets: number;
  compliantAssets: number;
  driftedAssets: number;
  driftPercentage: number;
  criticalDrift: number;
  averageDriftAge: string;
  byEnvironment: Array<{
    environment: string;
    compliant: number;
    total: number;
    percentage: number;
  }>;
  bySite: Array<{
    siteId: string;
    siteName: string;
    coverage: number;
    status: "success" | "warning" | "critical";
  }>;
  byAge: Array<{
    range: string;
    count: number;
    percentage: number;
  }>;
}

export interface Alert {
  id: string;
  severity: "critical" | "warning" | "info";
  title: string;
  description: string;
  source: string;
  siteId?: string;
  assetId?: string;
  imageId?: string;
  createdAt: string;
  acknowledgedAt?: string;
  resolvedAt?: string;
}

export interface Activity {
  id: string;
  type: "info" | "warning" | "success" | "critical";
  action: string;
  detail: string;
  userId?: string;
  siteId?: string;
  assetId?: string;
  imageId?: string;
  createdAt: string;
}

export interface OverviewMetrics {
  fleetSize: {
    value: number;
    trend: { direction: "up" | "down" | "neutral"; value: string; period: string };
  };
  driftScore: {
    value: number;
    trend: { direction: "up" | "down" | "neutral"; value: string; period: string };
  };
  compliance: {
    value: number;
    trend: { direction: "up" | "down" | "neutral"; value: string; period: string };
  };
  drReadiness: {
    value: number;
    trend: { direction: "up" | "down" | "neutral"; value: string; period: string };
  };
  platformDistribution: Array<{
    platform: "aws" | "azure" | "gcp" | "vsphere" | "k8s";
    count: number;
    percentage: number;
  }>;
  alerts: Array<{
    severity: "critical" | "warning" | "info";
    count: number;
  }>;
  recentActivity: Activity[];
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

// Fetch wrapper with error handling and auth
async function apiFetch<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${API_BASE_URL}${endpoint}`;

  // Get auth token if available
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

  // Handle empty responses
  const text = await response.text();
  if (!text) {
    return {} as T;
  }

  return JSON.parse(text) as T;
}

// API Client
export const api = {
  // Overview
  overview: {
    getMetrics: () => apiFetch<OverviewMetrics>("/overview/metrics"),
  },

  // Assets
  assets: {
    list: (params?: {
      siteId?: string;
      platform?: string;
      environment?: string;
      isDrifted?: boolean;
      limit?: number;
      offset?: number;
    }) => {
      const searchParams = new URLSearchParams();
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          if (value !== undefined) {
            searchParams.set(key, String(value));
          }
        });
      }
      const query = searchParams.toString();
      return apiFetch<{ assets: Asset[]; total: number }>(
        `/assets${query ? `?${query}` : ""}`
      );
    },
    get: (id: string) => apiFetch<Asset>(`/assets/${id}`),
    getDrifted: (limit?: number) =>
      apiFetch<Asset[]>(`/assets/drifted${limit ? `?limit=${limit}` : ""}`),
  },

  // Images
  images: {
    listFamilies: () => apiFetch<ImageFamily[]>("/images/families"),
    getFamily: (id: string) => apiFetch<ImageFamily>(`/images/families/${id}`),
    listVersions: (familyId: string) =>
      apiFetch<Image[]>(`/images/families/${familyId}/versions`),
    getVersion: (familyId: string, version: string) =>
      apiFetch<Image>(`/images/families/${familyId}/versions/${version}`),
    promote: (familyId: string, version: string, targetStatus: string) =>
      apiFetch<Image>(`/images/families/${familyId}/versions/${version}/promote`, {
        method: "POST",
        body: JSON.stringify({ targetStatus }),
      }),
    deprecate: (familyId: string, version: string) =>
      apiFetch<Image>(`/images/families/${familyId}/versions/${version}/deprecate`, {
        method: "POST",
      }),
  },

  // Sites
  sites: {
    list: () => apiFetch<Site[]>("/sites"),
    get: (id: string) => apiFetch<Site>(`/sites/${id}`),
    getAssets: (id: string, limit?: number) =>
      apiFetch<Asset[]>(`/sites/${id}/assets${limit ? `?limit=${limit}` : ""}`),
  },

  // Drift
  drift: {
    getSummary: () => apiFetch<DriftSummary>("/drift/summary"),
    getTopOffenders: (limit?: number) =>
      apiFetch<Asset[]>(`/drift/top-offenders${limit ? `?limit=${limit}` : ""}`),
    triggerScan: (siteId?: string) =>
      apiFetch<{ jobId: string }>("/drift/scan", {
        method: "POST",
        body: JSON.stringify({ siteId }),
      }),
  },

  // Alerts
  alerts: {
    list: (params?: { severity?: string; limit?: number }) => {
      const searchParams = new URLSearchParams();
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          if (value !== undefined) {
            searchParams.set(key, String(value));
          }
        });
      }
      const query = searchParams.toString();
      return apiFetch<Alert[]>(`/alerts${query ? `?${query}` : ""}`);
    },
    acknowledge: (id: string) =>
      apiFetch<Alert>(`/alerts/${id}/acknowledge`, { method: "POST" }),
    resolve: (id: string) =>
      apiFetch<Alert>(`/alerts/${id}/resolve`, { method: "POST" }),
  },

  // Activity
  activity: {
    list: (limit?: number) =>
      apiFetch<Activity[]>(`/activity${limit ? `?limit=${limit}` : ""}`),
  },
};

export default api;
