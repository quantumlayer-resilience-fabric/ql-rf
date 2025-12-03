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

export interface ComplianceFramework {
  id: string;
  name: string;
  description: string;
  score: number;
  passingControls: number;
  totalControls: number;
  status: "passing" | "warning" | "failing";
  level?: number;
}

export interface FailingControl {
  id: string;
  framework: string;
  title: string;
  severity: "high" | "medium" | "low";
  affectedAssets: number;
  recommendation: string;
}

export interface ImageComplianceStatus {
  familyId: string;
  familyName: string;
  version: string;
  cis: boolean;
  slsaLevel: number;
  cosignSigned: boolean;
  lastScanAt: string;
  issueCount: number;
}

// Image Lineage Types
export interface ImageLineageRelationship {
  id: string;
  imageId: string;
  parentImageId: string;
  relationshipType: "derived_from" | "patched_from" | "rebuilt_from";
  createdAt: string;
  parentImage?: {
    id: string;
    family: string;
    version: string;
    status: string;
  };
  image?: {
    id: string;
    family: string;
    version: string;
    status: string;
  };
}

export interface ImageBuild {
  id: string;
  imageId: string;
  buildNumber: number;
  sourceRepo?: string;
  sourceCommit?: string;
  sourceBranch?: string;
  builderType: string;
  builderVersion?: string;
  buildRunner?: string;
  buildRunnerId?: string;
  buildRunnerUrl?: string;
  buildLogUrl?: string;
  buildDurationSeconds?: number;
  builtBy?: string;
  signedBy?: string;
  status: "pending" | "building" | "success" | "failed";
  errorMessage?: string;
  startedAt?: string;
  completedAt?: string;
  createdAt: string;
}

export interface ImageVulnerability {
  id: string;
  imageId: string;
  cveId: string;
  severity: "critical" | "high" | "medium" | "low" | "unknown";
  cvssScore?: number;
  cvssVector?: string;
  packageName?: string;
  packageVersion?: string;
  packageType?: string;
  fixedVersion?: string;
  status: "open" | "fixed" | "wont_fix" | "false_positive";
  statusReason?: string;
  scanner?: string;
  scannedAt?: string;
  fixedInImageId?: string;
  resolvedAt?: string;
  resolvedBy?: string;
  createdAt: string;
  updatedAt: string;
}

export interface VulnerabilitySummary {
  imageId: string;
  family: string;
  version: string;
  criticalOpen: number;
  highOpen: number;
  mediumOpen: number;
  lowOpen: number;
  fixedCount: number;
  lastScannedAt?: string;
}

export interface ImageDeployment {
  id: string;
  imageId: string;
  assetId: string;
  deployedAt: string;
  deployedBy?: string;
  deploymentMethod?: string;
  status: "active" | "replaced" | "terminated";
  replacedAt?: string;
  replacedByImageId?: string;
  assetName?: string;
  platform?: string;
  region?: string;
}

export interface ImagePromotion {
  id: string;
  imageId: string;
  fromStatus: string;
  toStatus: string;
  promotedBy: string;
  approvedBy?: string;
  approvalTicket?: string;
  reason?: string;
  validationPassed?: boolean;
  promotedAt: string;
}

export interface ImageComponent {
  id: string;
  imageId: string;
  name: string;
  version: string;
  componentType: "os_package" | "library" | "binary" | "container";
  packageManager?: string;
  license?: string;
  licenseUrl?: string;
  sourceUrl?: string;
  checksum?: string;
  createdAt: string;
}

export interface LineageNode {
  image: {
    id: string;
    family: string;
    version: string;
    status: string;
    osName?: string;
    osVersion?: string;
  };
  depth: number;
  children?: LineageNode[];
  parents?: LineageNode[];
}

export interface ImageLineageTree {
  family: string;
  roots: LineageNode[];
  totalNodes: number;
}

export interface ImageLineageResponse {
  image: Image;
  parents: ImageLineageRelationship[];
  children: ImageLineageRelationship[];
  builds: ImageBuild[];
  vulnerabilitySummary: VulnerabilitySummary;
  activeDeployments: number;
  promotions: ImagePromotion[];
}

export interface ComplianceSummary {
  overallScore: number;
  cisCompliance: number;
  slsaLevel: number;
  sigstoreVerified: number;
  lastAuditAt: string;
  frameworks: ComplianceFramework[];
  failingControls: FailingControl[];
  imageCompliance: ImageComplianceStatus[];
}

export interface ResilienceSite {
  id: string;
  name: string;
  region: string;
  platform: "aws" | "azure" | "gcp" | "vsphere" | "k8s";
  type: "primary" | "dr";
  status: "healthy" | "warning" | "critical" | "syncing";
  assetCount: number;
  lastSyncAt: string;
  rpo: string;
  rto: string;
  replicationLag?: string;
}

export interface DRPair {
  id: string;
  name: string;
  primarySite: ResilienceSite;
  drSite: ResilienceSite;
  status: "healthy" | "warning" | "critical" | "syncing";
  lastFailoverTest?: string;
  replicationStatus: "in-sync" | "lagging" | "failed";
}

export interface ResilienceSummary {
  drReadiness: number;
  rpoCompliance: number;
  rtoCompliance: number;
  lastFailoverTest: string;
  totalPairs: number;
  healthyPairs: number;
  drPairs: DRPair[];
  unpairedSites: ResilienceSite[];
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
    // Lineage endpoints
    getLineage: (imageId: string) =>
      apiFetch<ImageLineageResponse>(`/images/${imageId}/lineage`),
    getLineageTree: (family: string) =>
      apiFetch<ImageLineageTree>(`/images/families/${family}/lineage-tree`),
    addParent: (imageId: string, parentImageId: string, relationshipType: string) =>
      apiFetch<ImageLineageRelationship>(`/images/${imageId}/lineage/parents`, {
        method: "POST",
        body: JSON.stringify({ parent_image_id: parentImageId, relationship_type: relationshipType }),
      }),
    getVulnerabilities: (imageId: string) =>
      apiFetch<ImageVulnerability[]>(`/images/${imageId}/vulnerabilities`),
    addVulnerability: (imageId: string, vulnerability: Partial<ImageVulnerability>) =>
      apiFetch<ImageVulnerability>(`/images/${imageId}/vulnerabilities`, {
        method: "POST",
        body: JSON.stringify(vulnerability),
      }),
    getBuilds: (imageId: string) =>
      apiFetch<ImageBuild[]>(`/images/${imageId}/builds`),
    getDeployments: (imageId: string) =>
      apiFetch<ImageDeployment[]>(`/images/${imageId}/deployments`),
    getComponents: (imageId: string) =>
      apiFetch<ImageComponent[]>(`/images/${imageId}/components`),
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

  // Compliance
  compliance: {
    getSummary: () => apiFetch<ComplianceSummary>("/compliance/summary"),
    getFrameworks: () => apiFetch<ComplianceFramework[]>("/compliance/frameworks"),
    getFailingControls: (framework?: string) => {
      const query = framework ? `?framework=${framework}` : "";
      return apiFetch<FailingControl[]>(`/compliance/controls/failing${query}`);
    },
    getImageCompliance: () => apiFetch<ImageComplianceStatus[]>("/compliance/images"),
    runAudit: () =>
      apiFetch<{ jobId: string }>("/compliance/audit", { method: "POST" }),
  },

  // Resilience
  resilience: {
    getSummary: () => apiFetch<ResilienceSummary>("/resilience/summary"),
    getDRPairs: () => apiFetch<DRPair[]>("/resilience/dr-pairs"),
    getDRPair: (id: string) => apiFetch<DRPair>(`/resilience/dr-pairs/${id}`),
    triggerFailoverTest: (pairId: string) =>
      apiFetch<{ jobId: string }>(`/resilience/dr-pairs/${pairId}/test`, {
        method: "POST",
      }),
    triggerSync: (pairId: string) =>
      apiFetch<{ jobId: string }>(`/resilience/dr-pairs/${pairId}/sync`, {
        method: "POST",
      }),
  },
};

export default api;
