/**
 * API Client for SBOM endpoints
 * Follows the SBOM OpenAPI specification
 */

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

// Token getter function (set by AuthProvider)
let getAuthToken: (() => Promise<string | null>) | null = null;

export function setAuthTokenGetter(getter: () => Promise<string | null>) {
  getAuthToken = getter;
}

// =============================================================================
// Types matching SBOM OpenAPI contract
// =============================================================================

export interface Package {
  id: string;
  sbomId: string;
  name: string;
  version: string;
  type: "deb" | "rpm" | "apk" | "npm" | "pip" | "go" | "jar" | "gem" | "nuget";
  purl?: string;
  cpe?: string;
  license?: string;
  supplier?: string;
  checksum?: string;
  sourceUrl?: string;
  location?: string;
  createdAt: string;
}

export interface Vulnerability {
  id: string;
  sbomId: string;
  packageId: string;
  cveId: string;
  severity: "critical" | "high" | "medium" | "low" | "unknown";
  cvssScore?: number;
  cvssVector?: string;
  description?: string;
  fixedVersion?: string;
  publishedDate?: string;
  modifiedDate?: string;
  references?: string[];
  dataSource?: string;
  exploitAvailable?: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface SBOM {
  id: string;
  imageId: string;
  orgId: string;
  format: "spdx" | "cyclonedx";
  version: string;
  content?: Record<string, unknown>;
  packageCount: number;
  vulnCount: number;
  scanner?: string;
  generatedAt: string;
  createdAt: string;
  updatedAt: string;
  packages?: Package[];
  vulnerabilities?: Vulnerability[];
}

export interface SBOMSummary {
  id: string;
  imageId: string;
  format: "spdx" | "cyclonedx";
  packageCount: number;
  vulnCount: number;
  critical: number;
  high: number;
  medium: number;
  low: number;
  generatedAt: string;
}

export interface SBOMListResponse {
  sboms: SBOMSummary[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface SBOMGenerateRequest {
  format: "spdx" | "cyclonedx";
  scanner?: string;
  includeVulns?: boolean;
  dockerfile?: string;
  manifests?: Record<string, string>;
}

export interface SBOMGenerationResponse {
  sbom: SBOM;
  status: "success" | "partial" | "failed";
  message?: string;
  packageCount: number;
  vulnCount: number;
  generatedAt: string;
}

export interface SBOMExportResponse {
  format: "spdx" | "cyclonedx";
  content: Record<string, unknown>;
}

export interface VulnerabilityListResponse {
  sbomId: string;
  vulnerabilities: Vulnerability[];
  count: number;
  stats: {
    critical: number;
    high: number;
    medium: number;
    low: number;
    unknown: number;
    fixAvailable: number;
    exploitAvailable: number;
  };
}

export interface LicenseSummary {
  licenses: Array<{
    name: string;
    count: number;
    packages: string[];
    category: "permissive" | "copyleft" | "proprietary" | "unknown";
  }>;
  totalPackages: number;
  unlicensedPackages: number;
  riskScore: number;
}

// =============================================================================
// API Error type
// =============================================================================

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
// Fetch wrapper with error handling and auth
// =============================================================================

async function apiFetch<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${API_BASE_URL}${endpoint}`;

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

  // Handle empty responses (e.g., 204 No Content)
  if (response.status === 204) {
    return {} as T;
  }

  const text = await response.text();
  if (!text) {
    return {} as T;
  }

  return JSON.parse(text) as T;
}

// =============================================================================
// SBOM API functions
// =============================================================================

export async function getSBOMs(params?: {
  page?: number;
  pageSize?: number;
}): Promise<SBOMListResponse> {
  const searchParams = new URLSearchParams();
  if (params?.page) searchParams.set("page", String(params.page));
  if (params?.pageSize) searchParams.set("page_size", String(params.pageSize));
  const query = searchParams.toString();

  return apiFetch<SBOMListResponse>(`/sbom${query ? `?${query}` : ""}`);
}

export async function getSBOM(
  id: string,
  params?: {
    includePackages?: boolean;
    includeVulns?: boolean;
  }
): Promise<SBOM> {
  const searchParams = new URLSearchParams();
  if (params?.includePackages) searchParams.set("include_packages", "true");
  if (params?.includeVulns) searchParams.set("include_vulns", "true");
  const query = searchParams.toString();

  return apiFetch<SBOM>(`/sbom/${id}${query ? `?${query}` : ""}`);
}

export async function getImageSBOM(
  imageId: string,
  params?: {
    includePackages?: boolean;
    includeVulns?: boolean;
  }
): Promise<SBOM> {
  const searchParams = new URLSearchParams();
  if (params?.includePackages) searchParams.set("include_packages", "true");
  if (params?.includeVulns) searchParams.set("include_vulns", "true");
  const query = searchParams.toString();

  return apiFetch<SBOM>(`/images/${imageId}/sbom${query ? `?${query}` : ""}`);
}

export async function generateSBOM(
  imageId: string,
  request: SBOMGenerateRequest
): Promise<SBOMGenerationResponse> {
  return apiFetch<SBOMGenerationResponse>(`/images/${imageId}/sbom/generate`, {
    method: "POST",
    body: JSON.stringify(request),
  });
}

export async function exportSBOM(
  id: string,
  format?: "spdx" | "cyclonedx"
): Promise<SBOMExportResponse> {
  const query = format ? `?format=${format}` : "";
  return apiFetch<SBOMExportResponse>(`/sbom/${id}/export${query}`);
}

export async function deleteSBOM(id: string): Promise<void> {
  return apiFetch<void>(`/sbom/${id}`, {
    method: "DELETE",
  });
}

export async function getSBOMVulnerabilities(
  sbomId: string,
  params?: {
    severity?: Array<"critical" | "high" | "medium" | "low" | "unknown">;
    minCvss?: number;
    hasExploit?: boolean;
    fixAvailable?: boolean;
  }
): Promise<VulnerabilityListResponse> {
  const searchParams = new URLSearchParams();
  if (params?.severity) {
    params.severity.forEach((s) => searchParams.append("severity", s));
  }
  if (params?.minCvss !== undefined) searchParams.set("min_cvss", String(params.minCvss));
  if (params?.hasExploit !== undefined) searchParams.set("has_exploit", String(params.hasExploit));
  if (params?.fixAvailable !== undefined) searchParams.set("fix_available", String(params.fixAvailable));
  const query = searchParams.toString();

  return apiFetch<VulnerabilityListResponse>(
    `/sbom/${sbomId}/vulnerabilities${query ? `?${query}` : ""}`
  );
}

export async function getLicenseSummary(): Promise<LicenseSummary> {
  // This endpoint doesn't exist in the OpenAPI spec yet, but we'll add it later
  // For now, return mock data
  return {
    licenses: [],
    totalPackages: 0,
    unlicensedPackages: 0,
    riskScore: 0,
  };
}

export async function getSBOMComponents(params?: {
  sbomId?: string;
  type?: string;
  license?: string;
}): Promise<Package[]> {
  // Helper function to get all components from a specific SBOM or all SBOMs
  if (params?.sbomId) {
    const sbom = await getSBOM(params.sbomId, { includePackages: true });
    return sbom.packages || [];
  }
  return [];
}
