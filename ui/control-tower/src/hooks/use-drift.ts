import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, DriftSummary, Asset } from "@/lib/api";

export const driftKeys = {
  all: ["drift"] as const,
  summary: () => [...driftKeys.all, "summary"] as const,
  topOffenders: (limit?: number) =>
    [...driftKeys.all, "top-offenders", limit] as const,
};

export function useDriftSummary() {
  return useQuery<DriftSummary>({
    queryKey: driftKeys.summary(),
    queryFn: () => api.drift.getSummary(),
    staleTime: 30 * 1000,
    refetchInterval: 60 * 1000,
  });
}

export function useTopOffenders(limit?: number) {
  return useQuery<Asset[]>({
    queryKey: driftKeys.topOffenders(limit),
    queryFn: () => api.drift.getTopOffenders(limit),
    staleTime: 30 * 1000,
    refetchInterval: 60 * 1000,
  });
}

export function useTriggerDriftScan() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (siteId?: string) => api.drift.triggerScan(siteId),
    onSuccess: () => {
      // Invalidate drift queries to refresh after scan
      queryClient.invalidateQueries({ queryKey: driftKeys.all });
    },
  });
}
