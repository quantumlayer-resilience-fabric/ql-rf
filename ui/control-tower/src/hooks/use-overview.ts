import { useQuery } from "@tanstack/react-query";
import { api, OverviewMetrics } from "@/lib/api";

export const overviewKeys = {
  all: ["overview"] as const,
  metrics: () => [...overviewKeys.all, "metrics"] as const,
};

export function useOverviewMetrics() {
  return useQuery<OverviewMetrics>({
    queryKey: overviewKeys.metrics(),
    queryFn: () => api.overview.getMetrics(),
    staleTime: 30 * 1000, // 30 seconds
    refetchInterval: 60 * 1000, // Refetch every minute
  });
}
