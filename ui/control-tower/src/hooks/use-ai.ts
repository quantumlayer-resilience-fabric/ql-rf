/**
 * React Query hooks for AI Copilot functionality
 */

import { useMutation } from "@tanstack/react-query";
import { useOverviewMetrics } from "./use-overview";
import { useDriftSummary } from "./use-drift";
import { useComplianceSummary } from "./use-compliance";
import { useResilienceSummary } from "./use-resilience";

// Types for AI messages
export interface AIMessage {
  role: "user" | "assistant";
  content: string;
}

export interface AIContext {
  fleetSize?: number;
  driftScore?: number;
  driftedAssets?: number;
  complianceScore?: number;
  drReadiness?: number;
  totalSites?: number;
  criticalAlerts?: number;
}

interface SendMessageParams {
  message: string;
  context?: AIContext;
  conversationHistory?: AIMessage[];
}

interface AIResponse {
  content: string;
  model?: string;
  usage?: {
    input_tokens: number;
    output_tokens: number;
  };
}

/**
 * Hook to aggregate infrastructure context from various data sources
 */
export function useAIContext(): AIContext {
  const { data: overview } = useOverviewMetrics();
  const { data: drift } = useDriftSummary();
  const { data: compliance } = useComplianceSummary();
  const { data: resilience } = useResilienceSummary();

  return {
    fleetSize: overview?.fleetSize?.value,
    driftScore: overview?.driftScore?.value,
    driftedAssets: drift?.driftedAssets,
    complianceScore: compliance?.overallScore,
    drReadiness: resilience?.drReadiness,
    totalSites: overview?.platformDistribution?.length,
    criticalAlerts: overview?.alerts?.find(a => a.severity === "critical")?.count,
  };
}

/**
 * Hook to send a message to the AI copilot
 */
export function useSendAIMessage() {
  return useMutation<AIResponse, Error, SendMessageParams>({
    mutationFn: async ({ message, context, conversationHistory }) => {
      const response = await fetch("/api/ai/chat", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          message,
          context,
          conversationHistory,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `AI request failed: ${response.status}`);
      }

      return response.json();
    },
  });
}

/**
 * Generate proactive insights based on current infrastructure state
 */
export function useProactiveInsights() {
  const context = useAIContext();

  // Generate insights based on context thresholds
  const insights = [];

  // Drift insights
  if (context.driftScore !== undefined && context.driftScore < 90) {
    insights.push({
      type: "drift" as const,
      title: "Drift Detected",
      description: `Your drift score is ${context.driftScore?.toFixed(1)}%. ${context.driftedAssets || 0} assets have drifted from their golden images.`,
      severity: context.driftScore < 80 ? "critical" as const : "warning" as const,
      action: "Analyze Drift",
    });
  }

  // Compliance insights
  if (context.complianceScore !== undefined && context.complianceScore < 95) {
    insights.push({
      type: "compliance" as const,
      title: "Compliance Gap",
      description: `Compliance score is ${context.complianceScore?.toFixed(1)}%. Review failing controls to improve your security posture.`,
      severity: context.complianceScore < 90 ? "critical" as const : "warning" as const,
      action: "View Controls",
    });
  }

  // DR insights
  if (context.drReadiness !== undefined && context.drReadiness < 95) {
    insights.push({
      type: "dr" as const,
      title: "DR Readiness",
      description: `DR readiness is at ${context.drReadiness?.toFixed(1)}%. Ensure all critical sites have DR pairs configured.`,
      severity: context.drReadiness < 90 ? "critical" as const : "warning" as const,
      action: "Review DR Status",
    });
  }

  // Critical alerts
  if (context.criticalAlerts !== undefined && context.criticalAlerts > 0) {
    insights.push({
      type: "optimization" as const,
      title: "Critical Alerts",
      description: `You have ${context.criticalAlerts} critical alerts requiring immediate attention.`,
      severity: "critical" as const,
      action: "View Alerts",
    });
  }

  return insights;
}
