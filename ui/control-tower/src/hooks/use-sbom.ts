/**
 * React Query hooks for SBOM data
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@clerk/nextjs";
import {
  getSBOMs,
  getSBOM,
  getImageSBOM,
  generateSBOM,
  exportSBOM,
  deleteSBOM,
  getSBOMVulnerabilities,
  getLicenseSummary,
  getSBOMComponents,
  type SBOMListResponse,
  type SBOM,
  type SBOMGenerateRequest,
  type SBOMGenerationResponse,
  type SBOMExportResponse,
  type VulnerabilityListResponse,
  type LicenseSummary,
  type Package,
} from "@/lib/api-sbom";

// Query keys
export const sbomKeys = {
  all: ["sbom"] as const,
  lists: () => [...sbomKeys.all, "list"] as const,
  list: (params?: { page?: number; pageSize?: number }) =>
    [...sbomKeys.lists(), params] as const,
  details: () => [...sbomKeys.all, "detail"] as const,
  detail: (id: string, params?: { includePackages?: boolean; includeVulns?: boolean }) =>
    [...sbomKeys.details(), id, params] as const,
  image: (imageId: string) => [...sbomKeys.all, "image", imageId] as const,
  vulnerabilities: (sbomId: string, params?: {
    severity?: Array<"critical" | "high" | "medium" | "low" | "unknown">;
    minCvss?: number;
    hasExploit?: boolean;
    fixAvailable?: boolean;
  }) => [...sbomKeys.all, "vulnerabilities", sbomId, params] as const,
  components: (params?: { sbomId?: string; type?: string; license?: string }) =>
    [...sbomKeys.all, "components", params] as const,
  licenses: () => [...sbomKeys.all, "licenses"] as const,
};

/**
 * Hook to fetch list of SBOMs
 */
export function useSBOMs(params?: {
  page?: number;
  pageSize?: number;
}) {
  const { isLoaded, isSignedIn } = useAuth();

  return useQuery<SBOMListResponse>({
    queryKey: sbomKeys.list(params),
    queryFn: () => getSBOMs(params),
    staleTime: 1000 * 60 * 5, // 5 minutes
    enabled: isLoaded && isSignedIn, // Only fetch when auth is ready
  });
}

/**
 * Hook to fetch a specific SBOM by ID
 */
export function useSBOM(
  id: string,
  params?: {
    includePackages?: boolean;
    includeVulns?: boolean;
  }
) {
  return useQuery<SBOM>({
    queryKey: sbomKeys.detail(id, params),
    queryFn: () => getSBOM(id, params),
    enabled: !!id,
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to fetch SBOM for a specific image
 */
export function useImageSBOM(
  imageId: string,
  params?: {
    includePackages?: boolean;
    includeVulns?: boolean;
  }
) {
  return useQuery<SBOM>({
    queryKey: sbomKeys.image(imageId),
    queryFn: () => getImageSBOM(imageId, params),
    enabled: !!imageId,
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to generate a new SBOM for an image
 */
export function useGenerateSBOM() {
  const queryClient = useQueryClient();

  return useMutation<
    SBOMGenerationResponse,
    Error,
    { imageId: string; request: SBOMGenerateRequest }
  >({
    mutationFn: ({ imageId, request }) => generateSBOM(imageId, request),
    onSuccess: (_, { imageId }) => {
      // Invalidate SBOM queries to refetch fresh data
      queryClient.invalidateQueries({ queryKey: sbomKeys.lists() });
      queryClient.invalidateQueries({ queryKey: sbomKeys.image(imageId) });
    },
  });
}

/**
 * Hook to export an SBOM
 */
export function useExportSBOM() {
  return useMutation<
    SBOMExportResponse,
    Error,
    { id: string; format?: "spdx" | "cyclonedx" }
  >({
    mutationFn: ({ id, format }) => exportSBOM(id, format),
  });
}

/**
 * Hook to delete an SBOM
 */
export function useDeleteSBOM() {
  const queryClient = useQueryClient();

  return useMutation<void, Error, string>({
    mutationFn: (id) => deleteSBOM(id),
    onSuccess: () => {
      // Invalidate SBOM queries
      queryClient.invalidateQueries({ queryKey: sbomKeys.all });
    },
  });
}

/**
 * Hook to fetch vulnerabilities for an SBOM
 */
export function useSBOMVulnerabilities(
  sbomId: string,
  params?: {
    severity?: Array<"critical" | "high" | "medium" | "low" | "unknown">;
    minCvss?: number;
    hasExploit?: boolean;
    fixAvailable?: boolean;
  }
) {
  return useQuery<VulnerabilityListResponse>({
    queryKey: sbomKeys.vulnerabilities(sbomId, params),
    queryFn: () => getSBOMVulnerabilities(sbomId, params),
    enabled: !!sbomId,
    staleTime: 1000 * 60 * 2, // 2 minutes
  });
}

/**
 * Hook to fetch SBOM components
 */
export function useSBOMComponents(params?: {
  sbomId?: string;
  type?: string;
  license?: string;
}) {
  return useQuery<Package[]>({
    queryKey: sbomKeys.components(params),
    queryFn: () => getSBOMComponents(params),
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to fetch license summary
 */
export function useLicenseSummary() {
  return useQuery<LicenseSummary>({
    queryKey: sbomKeys.licenses(),
    queryFn: () => getLicenseSummary(),
    staleTime: 1000 * 60 * 10, // 10 minutes
  });
}
