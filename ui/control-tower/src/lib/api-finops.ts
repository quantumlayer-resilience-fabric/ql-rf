/**
 * FinOps API Client for QL-RF Control Tower
 *
 * Provides typed API client for cost management and financial operations.
 */

import { apiFetch } from "./api";

// =============================================================================
// Types (matching OpenAPI contract)
// =============================================================================

export interface CostSummary {
  orgId: string;
  totalCost: number;
  currency: string;
  period: string;
  startDate: string;
  endDate: string;
  byCloud: Record<string, number>;
  byService: Record<string, number>;
  bySite: Record<string, number>;
  byResource: Record<string, ResourceCost>;
  trendChange: number;
}

export interface ResourceCost {
  resourceId: string;
  resourceType: string;
  resourceName: string;
  platform: "aws" | "azure" | "gcp" | "vsphere";
  cost: number;
  currency: string;
  usageHours?: number;
  tags?: Record<string, string>;
}

export interface CostBreakdown {
  dimension: "cloud" | "service" | "region" | "site" | "resource_type";
  items: CostBreakdownItem[];
  totalCost: number;
  currency: string;
  period: string;
  startDate: string;
  endDate: string;
}

export interface CostBreakdownItem {
  name: string;
  cost: number;
  percentage: number;
}

export interface CostTrend {
  date: string;
  cost: number;
  currency: string;
  byCloud?: Record<string, number>;
}

export interface CostTrendResponse {
  trend: CostTrend[];
  days: number;
}

export interface CostRecommendation {
  id: string;
  orgId: string;
  type: "rightsizing" | "reserved_instances" | "spot_instances" | "idle_resources" | "storage_optimization" | "unused_volumes" | "old_snapshots";
  resourceId: string;
  resourceType: string;
  resourceName?: string;
  platform: "aws" | "azure" | "gcp" | "vsphere";
  currentCost: number;
  potentialSavings: number;
  currency: string;
  action: string;
  details?: string;
  priority: "high" | "medium" | "low";
  status: "pending" | "applied" | "dismissed";
  detectedAt: string;
  appliedAt?: string;
  dismissedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface RecommendationListResponse {
  recommendations: CostRecommendation[];
  totalRecommendations: number;
  totalPotentialSavings: number;
  currency: string;
}

export interface CostBudget {
  id: string;
  orgId: string;
  name: string;
  description?: string;
  amount: number;
  currency: string;
  period: "daily" | "weekly" | "monthly" | "quarterly" | "yearly";
  scope: "organization" | "cloud" | "service" | "site";
  scopeValue?: string;
  alertThreshold: number;
  startDate: string;
  endDate?: string;
  currentSpend: number;
  active: boolean;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateBudgetRequest {
  name: string;
  description?: string;
  amount: number;
  currency?: string;
  period: "daily" | "weekly" | "monthly" | "quarterly" | "yearly";
  scope: "organization" | "cloud" | "service" | "site";
  scopeValue?: string;
  alertThreshold?: number;
  startDate: string;
  endDate?: string;
}

export interface BudgetListResponse {
  budgets: CostBudget[];
  total: number;
}

export interface ResourceCostListResponse {
  resources: ResourceCost[];
  totalCost: number;
  currency: string;
}

// =============================================================================
// API Functions
// =============================================================================

// Import the fetch wrapper from the main API client
// We need to access it through a workaround since it's not exported
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

// Token getter function - set by the auth provider (Clerk)
let getAuthToken: (() => Promise<string | null>) | null = null;

// Promise that resolves when auth is ready
let authReadyResolve: (() => void) | null = null;
const authReadyPromise = new Promise<void>((resolve) => {
  authReadyResolve = resolve;
});

/**
 * Set the auth token getter function.
 * Called from a client component that has access to Clerk's useAuth.
 */
export function setAuthTokenGetter(getter: () => Promise<string | null>) {
  getAuthToken = getter;
  // Signal that auth is ready
  if (authReadyResolve) {
    authReadyResolve();
    authReadyResolve = null;
  }
}

/**
 * Wait for auth to be ready before making API calls.
 */
async function waitForAuth(timeoutMs = 5000): Promise<void> {
  if (getAuthToken) return;

  await Promise.race([
    authReadyPromise,
    new Promise<void>((resolve) => setTimeout(resolve, timeoutMs)),
  ]);
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
async function finopsApiFetch<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${API_BASE_URL}${endpoint}`;

  // Wait for auth to be ready before making API calls
  await waitForAuth();

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

/**
 * Get cost summary for the organization
 */
export async function getCostSummary(
  period: string = "30d"
): Promise<CostSummary> {
  const params = new URLSearchParams({ period });
  const response = await finopsApiFetch<{
    org_id: string;
    total_cost: number;
    currency: string;
    period: string;
    start_date: string;
    end_date: string;
    by_cloud: Record<string, number>;
    by_service: Record<string, number>;
    by_site: Record<string, number>;
    by_resource: Record<string, any>;
    trend_change: number;
  }>(`/finops/summary?${params}`);

  // Transform snake_case to camelCase
  return {
    orgId: response.org_id,
    totalCost: response.total_cost,
    currency: response.currency,
    period: response.period,
    startDate: response.start_date,
    endDate: response.end_date,
    byCloud: response.by_cloud,
    byService: response.by_service,
    bySite: response.by_site,
    byResource: response.by_resource,
    trendChange: response.trend_change,
  };
}

/**
 * Get cost breakdown by dimension
 */
export async function getCostBreakdown(
  dimension: "cloud" | "service" | "region" | "site" | "resource_type" = "cloud",
  period: string = "30d"
): Promise<CostBreakdown> {
  const params = new URLSearchParams({ dimension, period });
  const response = await finopsApiFetch<{
    dimension: string;
    items: CostBreakdownItem[];
    total_cost: number;
    currency: string;
    period: string;
    start_date: string;
    end_date: string;
  }>(`/finops/breakdown?${params}`);

  return {
    dimension: response.dimension as CostBreakdown["dimension"],
    items: response.items,
    totalCost: response.total_cost,
    currency: response.currency,
    period: response.period,
    startDate: response.start_date,
    endDate: response.end_date,
  };
}

/**
 * Get cost trend over time
 */
export async function getCostTrend(days: number = 30): Promise<CostTrendResponse> {
  const params = new URLSearchParams({ days: days.toString() });
  const response = await finopsApiFetch<{
    trend: Array<{
      date: string;
      cost: number;
      currency: string;
      by_cloud?: Record<string, number>;
    }>;
    days: number;
  }>(`/finops/trend?${params}`);

  return {
    trend: response.trend.map(t => ({
      date: t.date,
      cost: t.cost,
      currency: t.currency,
      byCloud: t.by_cloud,
    })),
    days: response.days,
  };
}

/**
 * Get resource costs
 */
export async function getResourceCosts(
  resourceType?: string,
  period: string = "30d"
): Promise<ResourceCostListResponse> {
  const params = new URLSearchParams({ period });
  if (resourceType) {
    params.set("resource_type", resourceType);
  }

  const response = await finopsApiFetch<{
    resources: Array<{
      resource_id: string;
      resource_type: string;
      resource_name: string;
      platform: string;
      cost: number;
      currency: string;
      usage_hours?: number;
      tags?: Record<string, string>;
    }>;
    total_cost: number;
    currency: string;
  }>(`/finops/resources?${params}`);

  return {
    resources: response.resources.map(r => ({
      resourceId: r.resource_id,
      resourceType: r.resource_type,
      resourceName: r.resource_name,
      platform: r.platform as ResourceCost["platform"],
      cost: r.cost,
      currency: r.currency,
      usageHours: r.usage_hours,
      tags: r.tags,
    })),
    totalCost: response.total_cost,
    currency: response.currency,
  };
}

/**
 * Get cost optimization recommendations
 */
export async function getRecommendations(
  type?: string
): Promise<RecommendationListResponse> {
  const params = new URLSearchParams();
  if (type && type !== "all") {
    params.set("type", type);
  }

  const response = await finopsApiFetch<{
    recommendations: Array<{
      id: string;
      org_id: string;
      type: string;
      resource_id: string;
      resource_type: string;
      resource_name?: string;
      platform: string;
      current_cost: number;
      potential_savings: number;
      currency: string;
      action: string;
      details?: string;
      priority: string;
      status: string;
      detected_at: string;
      applied_at?: string;
      dismissed_at?: string;
      created_at: string;
      updated_at: string;
    }>;
    total_recommendations: number;
    total_potential_savings: number;
    currency: string;
  }>(`/finops/recommendations${params.toString() ? `?${params}` : ""}`);

  return {
    recommendations: response.recommendations.map(r => ({
      id: r.id,
      orgId: r.org_id,
      type: r.type as CostRecommendation["type"],
      resourceId: r.resource_id,
      resourceType: r.resource_type,
      resourceName: r.resource_name,
      platform: r.platform as CostRecommendation["platform"],
      currentCost: r.current_cost,
      potentialSavings: r.potential_savings,
      currency: r.currency,
      action: r.action,
      details: r.details,
      priority: r.priority as CostRecommendation["priority"],
      status: r.status as CostRecommendation["status"],
      detectedAt: r.detected_at,
      appliedAt: r.applied_at,
      dismissedAt: r.dismissed_at,
      createdAt: r.created_at,
      updatedAt: r.updated_at,
    })),
    totalRecommendations: response.total_recommendations,
    totalPotentialSavings: response.total_potential_savings,
    currency: response.currency,
  };
}

/**
 * List budgets
 */
export async function getBudgets(activeOnly: boolean = false): Promise<BudgetListResponse> {
  const params = new URLSearchParams();
  if (activeOnly) {
    params.set("active_only", "true");
  }

  const response = await finopsApiFetch<{
    budgets: Array<{
      id: string;
      org_id: string;
      name: string;
      description?: string;
      amount: number;
      currency: string;
      period: string;
      scope: string;
      scope_value?: string;
      alert_threshold: number;
      start_date: string;
      end_date?: string;
      current_spend: number;
      active: boolean;
      created_by: string;
      created_at: string;
      updated_at: string;
    }>;
    total: number;
  }>(`/finops/budgets${params.toString() ? `?${params}` : ""}`);

  return {
    budgets: response.budgets.map(b => ({
      id: b.id,
      orgId: b.org_id,
      name: b.name,
      description: b.description,
      amount: b.amount,
      currency: b.currency,
      period: b.period as CostBudget["period"],
      scope: b.scope as CostBudget["scope"],
      scopeValue: b.scope_value,
      alertThreshold: b.alert_threshold,
      startDate: b.start_date,
      endDate: b.end_date,
      currentSpend: b.current_spend,
      active: b.active,
      createdBy: b.created_by,
      createdAt: b.created_at,
      updatedAt: b.updated_at,
    })),
    total: response.total,
  };
}

/**
 * Create a new budget
 */
export async function createBudget(budget: CreateBudgetRequest): Promise<CostBudget> {
  const response = await finopsApiFetch<{
    id: string;
    org_id: string;
    name: string;
    description?: string;
    amount: number;
    currency: string;
    period: string;
    scope: string;
    scope_value?: string;
    alert_threshold: number;
    start_date: string;
    end_date?: string;
    current_spend: number;
    active: boolean;
    created_by: string;
    created_at: string;
    updated_at: string;
  }>("/finops/budgets", {
    method: "POST",
    body: JSON.stringify({
      name: budget.name,
      description: budget.description,
      amount: budget.amount,
      currency: budget.currency || "USD",
      period: budget.period,
      scope: budget.scope,
      scope_value: budget.scopeValue,
      alert_threshold: budget.alertThreshold || 80,
      start_date: budget.startDate,
      end_date: budget.endDate,
    }),
  });

  return {
    id: response.id,
    orgId: response.org_id,
    name: response.name,
    description: response.description,
    amount: response.amount,
    currency: response.currency,
    period: response.period as CostBudget["period"],
    scope: response.scope as CostBudget["scope"],
    scopeValue: response.scope_value,
    alertThreshold: response.alert_threshold,
    startDate: response.start_date,
    endDate: response.end_date,
    currentSpend: response.current_spend,
    active: response.active,
    createdBy: response.created_by,
    createdAt: response.created_at,
    updatedAt: response.updated_at,
  };
}
