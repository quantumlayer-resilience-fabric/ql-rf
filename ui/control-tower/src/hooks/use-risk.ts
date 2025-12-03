/**
 * React Query hooks for risk scoring data
 */

import { useQuery } from "@tanstack/react-query";
import {
  api,
  RiskSummary,
  AssetRiskScore,
  RiskForecast,
  RiskRecommendation,
  RiskAnomaly,
  RiskPrediction,
} from "@/lib/api";

// Query keys
export const riskKeys = {
  all: ["risk"] as const,
  summary: () => [...riskKeys.all, "summary"] as const,
  topRisks: (limit?: number) => [...riskKeys.all, "top", limit] as const,
  forecast: () => [...riskKeys.all, "forecast"] as const,
  recommendations: () => [...riskKeys.all, "recommendations"] as const,
  anomalies: () => [...riskKeys.all, "anomalies"] as const,
  assetPrediction: (assetId: string) => [...riskKeys.all, "asset", assetId, "prediction"] as const,
};

/**
 * Hook to fetch risk summary
 */
export function useRiskSummary() {
  return useQuery<RiskSummary>({
    queryKey: riskKeys.summary(),
    queryFn: () => api.risk.getSummary(),
    staleTime: 1000 * 60, // 1 minute
    refetchInterval: 1000 * 60 * 2, // 2 minutes
  });
}

/**
 * Hook to fetch top risk assets
 */
export function useTopRisks(limit?: number) {
  return useQuery<AssetRiskScore[]>({
    queryKey: riskKeys.topRisks(limit),
    queryFn: () => api.risk.getTopRisks(limit),
    staleTime: 1000 * 60, // 1 minute
    refetchInterval: 1000 * 60 * 2, // 2 minutes
  });
}

/**
 * Hook to fetch risk forecast with predictions
 */
export function useRiskForecast() {
  return useQuery<RiskForecast>({
    queryKey: riskKeys.forecast(),
    queryFn: () => api.risk.getForecast(),
    staleTime: 1000 * 60 * 5, // 5 minutes
    refetchInterval: 1000 * 60 * 10, // 10 minutes
  });
}

/**
 * Hook to fetch risk recommendations
 */
export function useRiskRecommendations() {
  return useQuery<RiskRecommendation[]>({
    queryKey: riskKeys.recommendations(),
    queryFn: () => api.risk.getRecommendations(),
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to fetch risk anomalies
 */
export function useRiskAnomalies() {
  return useQuery<RiskAnomaly[]>({
    queryKey: riskKeys.anomalies(),
    queryFn: () => api.risk.getAnomalies(),
    staleTime: 1000 * 60, // 1 minute
    refetchInterval: 1000 * 60 * 2, // 2 minutes
  });
}

/**
 * Hook to fetch asset-specific risk prediction
 */
export function useAssetPrediction(assetId: string) {
  return useQuery<RiskPrediction>({
    queryKey: riskKeys.assetPrediction(assetId),
    queryFn: () => api.risk.getAssetPrediction(assetId),
    staleTime: 1000 * 60 * 5, // 5 minutes
    enabled: !!assetId,
  });
}
