/**
 * React Query hooks for FinOps data
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  getCostSummary,
  getCostBreakdown,
  getCostTrend,
  getResourceCosts,
  getRecommendations,
  getBudgets,
  createBudget,
  CostSummary,
  CostBreakdown,
  CostTrendResponse,
  ResourceCostListResponse,
  RecommendationListResponse,
  BudgetListResponse,
  CostBudget,
  CreateBudgetRequest,
} from "@/lib/api-finops";

// Query keys
export const finopsKeys = {
  all: ["finops"] as const,
  summary: (period: string) => [...finopsKeys.all, "summary", period] as const,
  breakdown: (dimension: string, period: string) =>
    [...finopsKeys.all, "breakdown", dimension, period] as const,
  trend: (days: number) => [...finopsKeys.all, "trend", days] as const,
  resources: (resourceType?: string, period?: string) =>
    [...finopsKeys.all, "resources", resourceType, period] as const,
  recommendations: (type?: string) =>
    [...finopsKeys.all, "recommendations", type] as const,
  budgets: (activeOnly?: boolean) =>
    [...finopsKeys.all, "budgets", activeOnly] as const,
};

/**
 * Hook to fetch cost summary
 */
export function useCostSummary(period: string = "30d") {
  return useQuery<CostSummary>({
    queryKey: finopsKeys.summary(period),
    queryFn: () => getCostSummary(period),
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to fetch cost breakdown
 */
export function useCostBreakdown(
  dimension: "cloud" | "service" | "region" | "site" | "resource_type" = "cloud",
  period: string = "30d"
) {
  return useQuery<CostBreakdown>({
    queryKey: finopsKeys.breakdown(dimension, period),
    queryFn: () => getCostBreakdown(dimension, period),
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to fetch cost trend
 */
export function useCostTrend(days: number = 30) {
  return useQuery<CostTrendResponse>({
    queryKey: finopsKeys.trend(days),
    queryFn: () => getCostTrend(days),
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to fetch resource costs
 */
export function useResourceCosts(resourceType?: string, period: string = "30d") {
  return useQuery<ResourceCostListResponse>({
    queryKey: finopsKeys.resources(resourceType, period),
    queryFn: () => getResourceCosts(resourceType, period),
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to fetch cost recommendations
 */
export function useRecommendations(type?: string) {
  return useQuery<RecommendationListResponse>({
    queryKey: finopsKeys.recommendations(type),
    queryFn: () => getRecommendations(type),
    staleTime: 1000 * 60 * 10, // 10 minutes
  });
}

/**
 * Hook to fetch budgets
 */
export function useBudgets(activeOnly: boolean = false) {
  return useQuery<BudgetListResponse>({
    queryKey: finopsKeys.budgets(activeOnly),
    queryFn: () => getBudgets(activeOnly),
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to create a budget
 */
export function useCreateBudget() {
  const queryClient = useQueryClient();

  return useMutation<CostBudget, Error, CreateBudgetRequest>({
    mutationFn: (budget: CreateBudgetRequest) => createBudget(budget),
    onSuccess: () => {
      // Invalidate all budget queries to refetch fresh data
      queryClient.invalidateQueries({ queryKey: finopsKeys.budgets() });
    },
  });
}
