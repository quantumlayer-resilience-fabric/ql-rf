/**
 * API Client for QL-RF Control Tower
 *
 * Provides typed API client for communicating with the backend services.
 * Includes Clerk authentication token handling.
 */

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

// Dev bypass flag - if true, use dev-token immediately
// Check both string "true" and boolean coercion for build-time env vars
const devAuthBypass = process.env.NEXT_PUBLIC_DEV_AUTH_BYPASS === "true" ||
                      process.env.NEXT_PUBLIC_DEV_AUTH_BYPASS === "1";
const isDevelopment = process.env.NODE_ENV === "development";

// Token getter function - set by the auth provider
// Initialize with dev-token getter if in dev bypass mode OR if Clerk is not configured
const clerkConfigured = !!process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY;
let getAuthToken: (() => Promise<string | null>) | null =
  (devAuthBypass || isDevelopment || !clerkConfigured) ? (async () => "dev-token") : null;

/**
 * Set the auth token getter function.
 * Called from a client component that has access to Clerk's useAuth.
 */
export function setAuthTokenGetter(getter: () => Promise<string | null>) {
  getAuthToken = getter;
}

// Backend Asset type (what the API returns in snake_case)
interface BackendAsset {
  id: string;
  org_id: string;
  env_id?: string;
  platform: string;
  account?: string;
  region?: string;
  site?: string;
  instance_id: string;
  name?: string;
  image_ref?: string;
  image_version?: string;
  state: string;
  tags?: Record<string, string>;
  discovered_at: string;
  updated_at: string;
}

// Frontend Asset type (camelCase, enriched with display fields)
export interface Asset {
  id: string;
  orgId?: string;
  envId?: string;
  platform: "aws" | "azure" | "gcp" | "vsphere" | "k8s" | "baremetal";
  account?: string;
  region?: string;
  site?: string;
  instanceId?: string;
  name?: string;
  imageRef?: string;
  imageVersion?: string;
  state?: "running" | "stopped" | "terminated" | "pending" | "unknown";
  tags?: Record<string, string>;
  discoveredAt?: string;
  updatedAt?: string;
  // Computed/display fields for UI convenience
  hostname: string; // name || instanceId
  siteId: string;   // site || ""
  siteName: string; // site || "Unknown"
  // Drift-specific fields (populated by drift endpoints)
  isDrifted?: boolean;
  driftDetectedAt?: string;
  currentImageId?: string;
  currentImageVersion?: string;
  goldenImageId?: string;
  goldenImageVersion?: string;
  environment?: string;
  lastScannedAt?: string;
  createdAt?: string;
  metadata?: Record<string, string>;
}

// Transform backend asset to frontend format
function transformAsset(backend: BackendAsset, isDrifted?: boolean): Asset {
  return {
    id: backend.id,
    orgId: backend.org_id,
    envId: backend.env_id,
    platform: backend.platform as Asset["platform"],
    account: backend.account,
    region: backend.region,
    site: backend.site,
    instanceId: backend.instance_id,
    name: backend.name,
    imageRef: backend.image_ref,
    imageVersion: backend.image_version,
    state: backend.state as Asset["state"],
    tags: backend.tags,
    discoveredAt: backend.discovered_at,
    updatedAt: backend.updated_at,
    // Computed fields
    hostname: backend.name || backend.instance_id,
    siteId: backend.site || "",
    siteName: backend.site || "Unknown",
    isDrifted: isDrifted,
  };
}

export interface Image {
  id: string;
  familyId: string;
  familyName: string;
  version: string;
  description: string;
  status: "production" | "staging" | "testing" | "deprecated" | "pending";
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
  status: "production" | "staging" | "testing" | "deprecated" | "pending";
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

// Scanner Import Types
export type ScannerType = "trivy" | "grype" | "snyk" | "clair" | "anchore" | "aqua" | "twistlock" | "qualys";

export interface ScanVulnerability {
  cveId: string;
  severity: "critical" | "high" | "medium" | "low" | "unknown";
  cvssScore?: number;
  cvssVector?: string;
  packageName?: string;
  packageVersion?: string;
  packageType?: string;
  fixedVersion?: string;
  description?: string;
  references?: string[];
}

export interface ImportScanRequest {
  scanner: ScannerType;
  scanVersion?: string;
  scanStartedAt?: string;
  vulnerabilities: ScanVulnerability[];
}

export interface ImportScanResponse {
  imageId: string;
  scanner: string;
  imported: number;
  updated: number;
  fixed: number;
}

// SBOM Import Types
export type SBOMFormat = "spdx" | "cyclonedx" | "syft";

export interface SBOMComponent {
  name: string;
  version: string;
  componentType?: string;
  packageManager?: string;
  license?: string;
  licenseUrl?: string;
  sourceUrl?: string;
  checksum?: string;
  purl?: string;
}

export interface ImportSBOMRequest {
  format: SBOMFormat;
  sbomUrl?: string;
  components: SBOMComponent[];
}

export interface ImportSBOMResponse {
  imageId: string;
  format: string;
  components: number;
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

// Risk Scoring Types
export type RiskLevel = "critical" | "high" | "medium" | "low";

export interface RiskFactor {
  name: string;
  description: string;
  weight: number;
  score: number;
  impact: string;
}

export interface AssetRiskScore {
  assetId: string;
  assetName: string;
  platform: string;
  environment: string;
  site: string;
  riskScore: number;
  riskLevel: RiskLevel;
  factors: RiskFactor[];
  driftAge: number;
  vulnCount: number;
  criticalVulns: number;
  isCompliant: boolean;
  lastUpdated: string;
}

export interface RiskByScope {
  scope: string;
  riskScore: number;
  riskLevel: RiskLevel;
  assetCount: number;
  criticalRisk: number;
  highRisk: number;
}

export interface RiskTrendPoint {
  date: string;
  riskScore: number;
  riskLevel: RiskLevel;
}

export interface RiskSummary {
  orgId: string;
  overallRiskScore: number;
  riskLevel: RiskLevel;
  totalAssets: number;
  criticalRisk: number;
  highRisk: number;
  mediumRisk: number;
  lowRisk: number;
  topRisks: AssetRiskScore[];
  byEnvironment: RiskByScope[];
  byPlatform: RiskByScope[];
  bySite: RiskByScope[];
  trend: RiskTrendPoint[];
  calculatedAt: string;
}

// Predictive Risk Types
export type RiskVelocity = "rapid_increase" | "increasing" | "stable" | "decreasing" | "rapid_decrease";

export interface RiskPrediction {
  assetId?: string;
  scope?: string;
  currentScore: number;
  predictedScore: number;
  predictedLevel: RiskLevel;
  confidence: number;
  predictionHorizon: number;
  velocity: RiskVelocity;
  velocityValue: number;
  factors: string[];
  recommendedAction?: string;
  predictedAt: string;
}

export interface RiskAnomaly {
  id: string;
  assetId?: string;
  scope?: string;
  anomalyType: "spike" | "drop" | "pattern_break";
  severity: RiskLevel;
  description: string;
  expectedScore: number;
  actualScore: number;
  deviation: number;
  detectedAt: string;
  isActive: boolean;
}

export interface RiskRecommendation {
  id: string;
  priority: number;
  category: "patch" | "compliance" | "vulnerability" | "drift";
  title: string;
  description: string;
  impact: string;
  effort: "low" | "medium" | "high";
  affectedAssets: number;
  autoRemediable: boolean;
  actionType: "ai_task" | "manual" | "scheduled";
}

export interface RiskForecast {
  orgId: string;
  currentScore: number;
  predictions: RiskPrediction[];
  velocity: RiskVelocity;
  velocityValue: number;
  anomalies: RiskAnomaly[];
  atRiskAssets: AssetRiskScore[];
  improvingAssets: AssetRiskScore[];
  topRecommendations: RiskRecommendation[];
  generatedAt: string;
}

// Auto-Remediation Types
export interface MaintenanceWindow {
  dayOfWeek: number;
  startHour: number;
  endHour: number;
  timezone: string;
}

export interface AutoRemediationPolicy {
  id: string;
  orgId: string;
  name: string;
  description: string;
  enabled: boolean;
  maxRiskLevel: RiskLevel;
  environments: string[];
  platforms: string[];
  categories: string[];
  requireApproval: boolean;
  notifyOnAction: boolean;
  maxActionsPerDay: number;
  allowedWindows: MaintenanceWindow[];
  createdAt: string;
  updatedAt: string;
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
    list: async (params?: {
      site?: string;
      platform?: string;
      state?: string;
      envId?: string;
      page?: number;
      pageSize?: number;
    }): Promise<{ assets: Asset[]; total: number; page: number; pageSize: number; totalPages: number }> => {
      const searchParams = new URLSearchParams();
      if (params) {
        // Map frontend params to backend params
        if (params.site) searchParams.set("site", params.site);
        if (params.platform) searchParams.set("platform", params.platform);
        if (params.state) searchParams.set("state", params.state);
        if (params.envId) searchParams.set("env_id", params.envId);
        if (params.page) searchParams.set("page", String(params.page));
        if (params.pageSize) searchParams.set("page_size", String(params.pageSize));
      }
      const query = searchParams.toString();

      // Backend response type
      interface BackendAssetListResponse {
        assets: BackendAsset[];
        total: number;
        page: number;
        page_size: number;
        total_pages: number;
      }

      const response = await apiFetch<BackendAssetListResponse>(
        `/assets${query ? `?${query}` : ""}`
      );

      return {
        assets: (response.assets || []).map(a => transformAsset(a)),
        total: response.total,
        page: response.page,
        pageSize: response.page_size,
        totalPages: response.total_pages,
      };
    },

    get: async (id: string): Promise<Asset> => {
      const response = await apiFetch<BackendAsset>(`/assets/${id}`);
      return transformAsset(response);
    },

    getSummary: () => apiFetch<{
      total_assets: number;
      running_assets: number;
      stopped_assets: number;
      by_platform: Record<string, number>;
      by_state: Record<string, number>;
    }>("/assets/summary"),

    // Get drifted assets via top-offenders endpoint (returns already transformed)
    getDrifted: (limit?: number) =>
      apiFetch<Asset[]>(`/drift/top-offenders${limit ? `?limit=${limit}` : ""}`),
  },

  // Images
  images: {
    // List images and group by family for backward compatibility with frontend
    listFamilies: async (): Promise<ImageFamily[]> => {
      // Backend returns snake_case fields, we need to handle the transformation
      interface BackendImage {
        id: string;
        org_id: string;
        family: string;  // Backend uses 'family' not 'familyId'
        version: string;
        os_name?: string;
        os_version?: string;
        signed?: boolean;
        status: string;
        created_at: string;
        updated_at: string;
      }

      const response = await apiFetch<{ images: BackendImage[]; total: number }>("/images");
      const backendImages = response.images || [];

      // Group images by family name
      const familyMap = new Map<string, BackendImage[]>();
      for (const img of backendImages) {
        const familyName = img.family || "unknown";
        if (!familyMap.has(familyName)) {
          familyMap.set(familyName, []);
        }
        familyMap.get(familyName)!.push(img);
      }

      // Convert to ImageFamily format
      const families: ImageFamily[] = [];
      for (const [familyName, familyImages] of familyMap) {
        // Find latest image for this family (by version or created_at)
        const sortedImages = [...familyImages].sort(
          (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        );
        const latestImage = sortedImages[0];

        // Transform backend images to frontend Image format
        const transformedVersions: Image[] = familyImages.map(img => ({
          id: img.id,
          familyId: familyName,
          familyName: familyName,
          version: img.version,
          description: `${img.os_name || ''} ${img.os_version || ''}`.trim(),
          status: (img.status as Image["status"]) || "pending",
          platforms: ["aws"] as Image["platforms"], // Default, could be derived from coordinates
          compliance: {
            cis: false,
            slsaLevel: 0,
            cosignSigned: img.signed || false,
          },
          deployedCount: 0,
          createdAt: img.created_at,
          createdBy: "system",
        }));

        families.push({
          id: familyName, // Use family name as ID since backend doesn't have separate family entity
          name: familyName,
          description: `${latestImage?.os_name || ''} ${latestImage?.os_version || ''}`.trim() || "Golden image family",
          owner: "system",
          latestVersion: latestImage?.version || "0.0.0",
          status: (latestImage?.status as ImageFamily["status"]) || "pending",
          totalDeployed: 0,
          versions: transformedVersions,
          createdAt: sortedImages[sortedImages.length - 1]?.created_at || new Date().toISOString(),
          updatedAt: latestImage?.created_at || new Date().toISOString(),
        });
      }

      return families;
    },
    getFamily: (id: string) => apiFetch<ImageFamily>(`/images/${id}`),
    listVersions: (familyId: string) =>
      apiFetch<Image[]>(`/images?family=${familyId}`),
    getVersion: (familyId: string, version: string) =>
      apiFetch<Image>(`/images/${familyId}/latest`),
    promote: (imageId: string, _version: string, targetStatus: string) =>
      apiFetch<Image>(`/images/${imageId}/promote`, {
        method: "POST",
        body: JSON.stringify({ status: targetStatus }),
      }),
    deprecate: (imageId: string, _version: string) =>
      apiFetch<Image>(`/images/${imageId}/promote`, {
        method: "POST",
        body: JSON.stringify({ status: "deprecated" }),
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
    // Import endpoints
    importScanResults: (imageId: string, data: ImportScanRequest) =>
      apiFetch<ImportScanResponse>(`/images/${imageId}/vulnerabilities/import`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    importSBOM: (imageId: string, data: ImportSBOMRequest) =>
      apiFetch<ImportSBOMResponse>(`/images/${imageId}/sbom`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
  },

  // Sites
  sites: {
    list: async (): Promise<Site[]> => {
      const response = await apiFetch<{ sites: Site[] }>("/sites");
      return response.sites || [];
    },
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

  // Risk Scoring
  risk: {
    getSummary: () => apiFetch<RiskSummary>("/risk/summary"),
    getTopRisks: (limit?: number) =>
      apiFetch<AssetRiskScore[]>(`/risk/top${limit ? `?limit=${limit}` : ""}`),
    // Predictive risk endpoints
    getForecast: () => apiFetch<RiskForecast>("/risk/forecast"),
    getRecommendations: () => apiFetch<RiskRecommendation[]>("/risk/recommendations"),
    getAnomalies: () => apiFetch<RiskAnomaly[]>("/risk/anomalies"),
    getAssetPrediction: (assetId: string) =>
      apiFetch<RiskPrediction>(`/risk/assets/${assetId}/prediction`),
  },
};

export default api;
