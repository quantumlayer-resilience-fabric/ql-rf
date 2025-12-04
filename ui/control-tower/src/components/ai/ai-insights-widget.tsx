"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Sparkles,
  AlertTriangle,
  TrendingDown,
  Shield,
  RefreshCw,
  ArrowRight,
  CheckCircle2,
  Clock,
  Zap
} from "lucide-react";
import { useProactiveInsights, usePendingTasks, useSendAIMessage, useAIContext } from "@/hooks/use-ai";
import { useRouter } from "next/navigation";
import { useState } from "react";

interface AIInsightsWidgetProps {
  className?: string;
  compact?: boolean;
}

const insightIcons = {
  drift: TrendingDown,
  compliance: Shield,
  dr: RefreshCw,
  optimization: AlertTriangle,
};

const severityColors = {
  critical: "bg-status-red/10 border-status-red/30 text-status-red",
  warning: "bg-status-amber/10 border-status-amber/30 text-status-amber",
  info: "bg-brand-accent/10 border-brand-accent/30 text-brand-accent",
};

export function AIInsightsWidget({ className, compact = false }: AIInsightsWidgetProps) {
  const router = useRouter();
  const insights = useProactiveInsights();
  const { data: pendingTasks = [] } = usePendingTasks();
  const context = useAIContext();
  const sendMessage = useSendAIMessage();
  const [loadingAction, setLoadingAction] = useState<string | null>(null);

  // Quick actions that trigger AI tasks
  const handleQuickAction = async (intent: string, actionId: string) => {
    setLoadingAction(actionId);
    try {
      await sendMessage.mutateAsync({
        message: intent,
        context,
      });
      // Navigate to AI page to see the task
      router.push("/ai");
    } catch (error) {
      console.error("Failed to create AI task:", error);
    } finally {
      setLoadingAction(null);
    }
  };

  // Get top 3 insights sorted by severity
  const topInsights = insights
    .sort((a, b) => {
      const severityOrder = { critical: 0, warning: 1, info: 2 };
      return (severityOrder[a.severity] || 2) - (severityOrder[b.severity] || 2);
    })
    .slice(0, compact ? 2 : 3);

  const hasPendingApprovals = pendingTasks.length > 0;
  const hasInsights = topInsights.length > 0;

  if (!hasInsights && !hasPendingApprovals) {
    return (
      <Card className={`border-status-green/30 bg-gradient-to-br from-status-green/5 to-transparent ${className}`}>
        <CardContent className="flex items-center gap-4 py-6">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-status-green/10">
            <CheckCircle2 className="h-6 w-6 text-status-green" />
          </div>
          <div>
            <h3 className="font-semibold text-foreground">All Systems Healthy</h3>
            <p className="text-sm text-muted-foreground">
              No critical issues detected. Your infrastructure is in good shape.
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card
      variant="elevated"
      className={`border-brand-accent/20 bg-gradient-to-br from-brand-accent/5 via-transparent to-transparent ${className}`}
    >
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="flex items-center gap-2 text-base" style={{ fontFamily: "var(--font-display)" }}>
          <Sparkles className="h-5 w-5 text-brand-accent" />
          AI Insights
        </CardTitle>
        {hasPendingApprovals && (
          <Badge variant="secondary" className="bg-status-amber/10 text-status-amber border-status-amber/30">
            <Clock className="h-3 w-3 mr-1" />
            {pendingTasks.length} Pending
          </Badge>
        )}
      </CardHeader>
      <CardContent className="space-y-3">
        {/* Pending Approval Alert */}
        {hasPendingApprovals && (
          <button
            onClick={() => router.push("/ai")}
            className="w-full flex items-center gap-3 p-3 rounded-lg border border-status-amber/30 bg-status-amber/5 hover:bg-status-amber/10 transition-colors text-left group"
          >
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-status-amber/10 shrink-0">
              <Clock className="h-4 w-4 text-status-amber" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-foreground">
                {pendingTasks.length} task{pendingTasks.length > 1 ? "s" : ""} awaiting approval
              </p>
              <p className="text-xs text-muted-foreground truncate">
                {pendingTasks[0]?.user_intent || "Review and approve AI-generated plans"}
              </p>
            </div>
            <ArrowRight className="h-4 w-4 text-muted-foreground group-hover:translate-x-1 transition-transform" />
          </button>
        )}

        {/* Proactive Insights */}
        {topInsights.map((insight, index) => {
          const Icon = insightIcons[insight.type] || AlertTriangle;
          const colorClass = severityColors[insight.severity] || severityColors.info;
          const actionId = `${insight.type}-${index}`;
          const isLoading = loadingAction === actionId;

          return (
            <div
              key={actionId}
              className={`flex items-start gap-3 p-3 rounded-lg border ${colorClass} transition-all`}
            >
              <div className={`flex h-9 w-9 items-center justify-center rounded-lg shrink-0 ${
                insight.severity === "critical" ? "bg-status-red/10" :
                insight.severity === "warning" ? "bg-status-amber/10" : "bg-brand-accent/10"
              }`}>
                <Icon className={`h-4 w-4 ${
                  insight.severity === "critical" ? "text-status-red" :
                  insight.severity === "warning" ? "text-status-amber" : "text-brand-accent"
                }`} />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <p className="text-sm font-medium text-foreground">{insight.title}</p>
                  <Badge
                    variant="outline"
                    className={`text-[10px] px-1.5 py-0 ${
                      insight.severity === "critical" ? "border-status-red/50 text-status-red" :
                      insight.severity === "warning" ? "border-status-amber/50 text-status-amber" : ""
                    }`}
                  >
                    {insight.severity}
                  </Badge>
                </div>
                <p className="text-xs text-muted-foreground mt-0.5 line-clamp-2">
                  {insight.description}
                </p>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 px-2 mt-2 text-xs hover:bg-background/50"
                  onClick={() => handleQuickAction(
                    insight.type === "drift" ? "Analyze drift and suggest remediation" :
                    insight.type === "compliance" ? "Review compliance gaps and recommend fixes" :
                    insight.type === "dr" ? "Check DR readiness and suggest improvements" :
                    "Analyze and fix the issue",
                    actionId
                  )}
                  disabled={isLoading}
                >
                  {isLoading ? (
                    <>
                      <RefreshCw className="h-3 w-3 mr-1 animate-spin" />
                      Creating task...
                    </>
                  ) : (
                    <>
                      <Zap className="h-3 w-3 mr-1" />
                      Fix with AI
                    </>
                  )}
                </Button>
              </div>
            </div>
          );
        })}

        {/* View All Link */}
        {!compact && (hasInsights || hasPendingApprovals) && (
          <Button
            variant="ghost"
            className="w-full justify-center text-sm text-brand-accent hover:text-brand-accent/80"
            onClick={() => router.push("/ai")}
          >
            Open AI Copilot
            <ArrowRight className="h-4 w-4 ml-1" />
          </Button>
        )}
      </CardContent>
    </Card>
  );
}
