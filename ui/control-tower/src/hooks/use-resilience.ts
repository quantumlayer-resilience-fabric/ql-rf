/**
 * React Query hooks for resilience/DR data
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, ResilienceSummary, DRPair } from "@/lib/api";

// Query keys
export const resilienceKeys = {
  all: ["resilience"] as const,
  summary: () => [...resilienceKeys.all, "summary"] as const,
  drPairs: () => [...resilienceKeys.all, "dr-pairs"] as const,
  drPair: (id: string) => [...resilienceKeys.all, "dr-pairs", id] as const,
};

/**
 * Hook to fetch resilience summary
 */
export function useResilienceSummary() {
  return useQuery<ResilienceSummary>({
    queryKey: resilienceKeys.summary(),
    queryFn: () => api.resilience.getSummary(),
    staleTime: 1000 * 60 * 2, // 2 minutes
  });
}

/**
 * Hook to fetch DR pairs
 */
export function useDRPairs() {
  return useQuery<DRPair[]>({
    queryKey: resilienceKeys.drPairs(),
    queryFn: () => api.resilience.getDRPairs(),
    staleTime: 1000 * 60 * 2, // 2 minutes
  });
}

/**
 * Hook to fetch a single DR pair
 */
export function useDRPair(id: string) {
  return useQuery<DRPair>({
    queryKey: resilienceKeys.drPair(id),
    queryFn: () => api.resilience.getDRPair(id),
    enabled: !!id,
    staleTime: 1000 * 60 * 2, // 2 minutes
  });
}

/**
 * Hook to trigger a failover test
 */
export function useTriggerFailoverTest() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (pairId: string) => api.resilience.triggerFailoverTest(pairId),
    onSuccess: () => {
      // Invalidate DR queries to refetch fresh data
      queryClient.invalidateQueries({ queryKey: resilienceKeys.all });
    },
  });
}

/**
 * Hook to trigger DR sync
 */
export function useTriggerDRSync() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (pairId: string) => api.resilience.triggerSync(pairId),
    onSuccess: () => {
      // Invalidate DR queries to refetch fresh data
      queryClient.invalidateQueries({ queryKey: resilienceKeys.all });
    },
  });
}
