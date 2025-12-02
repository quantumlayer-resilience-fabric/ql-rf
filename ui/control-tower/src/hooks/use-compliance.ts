/**
 * React Query hooks for compliance data
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, ComplianceSummary, ComplianceFramework, FailingControl, ImageComplianceStatus } from "@/lib/api";

// Query keys
export const complianceKeys = {
  all: ["compliance"] as const,
  summary: () => [...complianceKeys.all, "summary"] as const,
  frameworks: () => [...complianceKeys.all, "frameworks"] as const,
  failingControls: (framework?: string) => [...complianceKeys.all, "failing-controls", framework] as const,
  imageCompliance: () => [...complianceKeys.all, "images"] as const,
};

/**
 * Hook to fetch compliance summary
 */
export function useComplianceSummary() {
  return useQuery<ComplianceSummary>({
    queryKey: complianceKeys.summary(),
    queryFn: () => api.compliance.getSummary(),
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to fetch compliance frameworks
 */
export function useComplianceFrameworks() {
  return useQuery<ComplianceFramework[]>({
    queryKey: complianceKeys.frameworks(),
    queryFn: () => api.compliance.getFrameworks(),
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to fetch failing controls
 */
export function useFailingControls(framework?: string) {
  return useQuery<FailingControl[]>({
    queryKey: complianceKeys.failingControls(framework),
    queryFn: () => api.compliance.getFailingControls(framework),
    staleTime: 1000 * 60 * 2, // 2 minutes
  });
}

/**
 * Hook to fetch image compliance status
 */
export function useImageCompliance() {
  return useQuery<ImageComplianceStatus[]>({
    queryKey: complianceKeys.imageCompliance(),
    queryFn: () => api.compliance.getImageCompliance(),
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to trigger a compliance audit
 */
export function useRunComplianceAudit() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => api.compliance.runAudit(),
    onSuccess: () => {
      // Invalidate all compliance queries to refetch fresh data
      queryClient.invalidateQueries({ queryKey: complianceKeys.all });
    },
  });
}
