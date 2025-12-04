import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { api, DriftSummary, Asset } from "@/lib/api";

export interface DriftFilters {
  environment?: string;
  platform?: string;
}

export const driftKeys = {
  all: ["drift"] as const,
  summary: () => [...driftKeys.all, "summary"] as const,
  topOffenders: (limit?: number) =>
    [...driftKeys.all, "top-offenders", limit] as const,
};

export function useDriftSummary(filters?: DriftFilters) {
  const query = useQuery<DriftSummary>({
    queryKey: driftKeys.summary(),
    queryFn: () => api.drift.getSummary(),
    staleTime: 30 * 1000,
    refetchInterval: 60 * 1000,
  });

  // Apply client-side filtering to the summary data
  const filteredData = useMemo(() => {
    if (!query.data) return undefined;

    const data = query.data;
    const env = filters?.environment;
    const platform = filters?.platform;

    // If no filters, return as-is
    if ((!env || env === "all") && (!platform || platform === "all")) {
      return data;
    }

    // Filter byEnvironment
    let filteredByEnv = data.byEnvironment;
    if (env && env !== "all") {
      filteredByEnv = data.byEnvironment.filter(
        (e) => e.environment.toLowerCase() === env.toLowerCase()
      );
    }

    // Filter bySite (we don't have platform info per site, so we keep all)
    // In a real implementation, sites would have platform info
    const filteredBySite = data.bySite;

    // Recalculate totals based on filtered environment data
    let totalAssets = 0;
    let compliantAssets = 0;
    if (filteredByEnv.length > 0) {
      filteredByEnv.forEach((e) => {
        totalAssets += e.total;
        compliantAssets += e.compliant;
      });
    } else {
      // If no environment filter matches, use original totals
      totalAssets = data.totalAssets;
      compliantAssets = data.compliantAssets;
    }

    const driftedAssets = totalAssets - compliantAssets;
    const driftPercentage = totalAssets > 0
      ? (driftedAssets / totalAssets) * 100
      : 0;

    // Calculate critical drift from filtered environments
    let criticalDrift = 0;
    filteredByEnv.forEach((e) => {
      // If percentage < 70, consider it critical
      if (e.percentage < 70) {
        criticalDrift += e.total - e.compliant;
      }
    });

    return {
      ...data,
      totalAssets,
      compliantAssets,
      driftedAssets,
      driftPercentage,
      criticalDrift,
      byEnvironment: filteredByEnv,
      bySite: filteredBySite,
    };
  }, [query.data, filters?.environment, filters?.platform]);

  return {
    ...query,
    data: filteredData,
  };
}

export function useTopOffenders(limit?: number, filters?: DriftFilters) {
  const query = useQuery<Asset[]>({
    queryKey: driftKeys.topOffenders(limit),
    queryFn: () => api.drift.getTopOffenders(limit),
    staleTime: 30 * 1000,
    refetchInterval: 60 * 1000,
  });

  // Apply client-side filtering to top offenders
  const filteredData = useMemo(() => {
    if (!query.data) return undefined;

    const env = filters?.environment;
    const platform = filters?.platform;

    // If no filters, return as-is
    if ((!env || env === "all") && (!platform || platform === "all")) {
      return query.data;
    }

    return query.data.filter((asset) => {
      // Filter by environment (check environment field or derive from tags)
      if (env && env !== "all") {
        const assetEnv = asset.environment?.toLowerCase() || "";
        if (assetEnv && assetEnv !== env.toLowerCase()) {
          return false;
        }
      }

      // Filter by platform
      if (platform && platform !== "all") {
        if (asset.platform.toLowerCase() !== platform.toLowerCase()) {
          return false;
        }
      }

      return true;
    });
  }, [query.data, filters?.environment, filters?.platform]);

  return {
    ...query,
    data: filteredData,
  };
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
