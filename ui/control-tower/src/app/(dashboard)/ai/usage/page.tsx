"use client";

import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { GradientText } from "@/components/brand/gradient-text";
import { useAIUsageStats, AIUsageStats } from "@/hooks/use-ai";
import {
  Bot,
  Coins,
  BarChart3,
  TrendingUp,
  Calendar,
  Zap,
  AlertTriangle,
  CheckCircle,
  Settings,
  ArrowLeft,
  Loader2,
  Brain,
  Activity,
} from "lucide-react";
import Link from "next/link";
import { cn } from "@/lib/utils";

export default function AIUsagePage() {
  const { data: stats, isLoading, error } = useAIUsageStats();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error || !stats) {
    // Return mock data for demo purposes
    const mockStats: AIUsageStats = {
      tokens_used_this_month: 125000,
      monthly_token_budget: 500000,
      token_usage_percent: 25,
      tasks_this_month: 47,
      tasks_this_week: 12,
      tasks_today: 3,
      estimated_cost_this_month: 6.25,
      cost_per_1k_tokens: 0.05,
      llm_provider: "anthropic",
      llm_model: "claude-3-5-sonnet-20241022",
      usage_by_task_type: [
        { task_type: "drift_remediation", count: 15, tokens: 45000 },
        { task_type: "compliance_audit", count: 12, tokens: 32000 },
        { task_type: "incident_investigation", count: 8, tokens: 24000 },
        { task_type: "patch_rollout", count: 6, tokens: 12000 },
        { task_type: "cost_optimization", count: 4, tokens: 8000 },
        { task_type: "security_scan", count: 2, tokens: 4000 },
      ],
      usage_by_agent: [
        { agent: "drift_agent", count: 15, tokens: 45000, avg_tokens: 3000 },
        { agent: "compliance_agent", count: 12, tokens: 32000, avg_tokens: 2667 },
        { agent: "incident_agent", count: 8, tokens: 24000, avg_tokens: 3000 },
        { agent: "patch_agent", count: 6, tokens: 12000, avg_tokens: 2000 },
        { agent: "cost_agent", count: 4, tokens: 8000, avg_tokens: 2000 },
        { agent: "security_agent", count: 2, tokens: 4000, avg_tokens: 2000 },
      ],
      daily_usage: generateMockDailyUsage(),
      ai_enabled: true,
      auto_remediation_enabled: false,
      autonomy_mode: "plan_only",
    };

    return <AIUsageDashboard stats={mockStats} />;
  }

  return <AIUsageDashboard stats={stats} />;
}

function generateMockDailyUsage() {
  const days = [];
  const now = new Date();
  for (let i = 29; i >= 0; i--) {
    const date = new Date(now);
    date.setDate(date.getDate() - i);
    const tokens = Math.floor(Math.random() * 8000) + 2000;
    days.push({
      date: date.toISOString().split("T")[0],
      tokens,
      tasks: Math.floor(tokens / 2500),
      cost: tokens * 0.00005,
    });
  }
  return days;
}

function AIUsageDashboard({ stats }: { stats: AIUsageStats }) {
  const budgetUsagePercent = stats.monthly_token_budget
    ? (stats.tokens_used_this_month / stats.monthly_token_budget) * 100
    : 0;

  const isOverBudget = budgetUsagePercent > 100;
  const isNearBudget = budgetUsagePercent > 80 && !isOverBudget;

  return (
    <div className="page-transition space-y-6">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Link href="/ai" className="hover:text-foreground">
          AI Copilot
        </Link>
        <span>/</span>
        <span className="text-foreground">Usage Analytics</span>
      </div>

      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            <GradientText variant="ai">LLM Usage Analytics</GradientText>
          </h1>
          <p className="text-muted-foreground mt-1">
            Monitor AI token usage, costs, and performance metrics
          </p>
        </div>
        <Link href="/settings">
          <Button variant="outline" size="sm">
            <Settings className="mr-2 h-4 w-4" />
            AI Settings
          </Button>
        </Link>
      </div>

      {/* Top Stats */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {/* Token Usage */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Tokens Used</CardTitle>
            <Coins className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {(stats.tokens_used_this_month / 1000).toFixed(1)}k
            </div>
            <p className="text-xs text-muted-foreground">
              {stats.monthly_token_budget
                ? `of ${(stats.monthly_token_budget / 1000).toFixed(0)}k budget`
                : "this month"}
            </p>
            {stats.monthly_token_budget && (
              <Progress
                value={Math.min(budgetUsagePercent, 100)}
                className={cn(
                  "mt-2 h-2",
                  isOverBudget ? "[&>div]:bg-status-red" : isNearBudget ? "[&>div]:bg-status-amber" : ""
                )}
              />
            )}
          </CardContent>
        </Card>

        {/* Tasks Completed */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Tasks This Month</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.tasks_this_month}</div>
            <p className="text-xs text-muted-foreground">
              {stats.tasks_today} today, {stats.tasks_this_week} this week
            </p>
          </CardContent>
        </Card>

        {/* Estimated Cost */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Estimated Cost</CardTitle>
            <BarChart3 className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              ${stats.estimated_cost_this_month.toFixed(2)}
            </div>
            <p className="text-xs text-muted-foreground">
              ${stats.cost_per_1k_tokens.toFixed(4)} per 1k tokens
            </p>
          </CardContent>
        </Card>

        {/* AI Status */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">AI Status</CardTitle>
            <Brain className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              {stats.ai_enabled ? (
                <>
                  <CheckCircle className="h-5 w-5 text-status-green" />
                  <span className="text-lg font-medium">Active</span>
                </>
              ) : (
                <>
                  <AlertTriangle className="h-5 w-5 text-status-amber" />
                  <span className="text-lg font-medium">Disabled</span>
                </>
              )}
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              Mode: {stats.autonomy_mode.replace("_", " ")}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Budget Warning */}
      {(isOverBudget || isNearBudget) && (
        <Card className={cn(
          "border-2",
          isOverBudget ? "border-status-red bg-status-red/5" : "border-status-amber bg-status-amber/5"
        )}>
          <CardContent className="pt-6">
            <div className="flex items-center gap-3">
              <AlertTriangle className={cn(
                "h-5 w-5",
                isOverBudget ? "text-status-red" : "text-status-amber"
              )} />
              <div>
                <h4 className={cn(
                  "font-medium",
                  isOverBudget ? "text-status-red" : "text-status-amber"
                )}>
                  {isOverBudget ? "Budget Exceeded" : "Approaching Budget Limit"}
                </h4>
                <p className="text-sm text-muted-foreground mt-1">
                  {isOverBudget
                    ? `You've used ${budgetUsagePercent.toFixed(0)}% of your monthly token budget. Consider reviewing AI usage patterns or increasing your budget.`
                    : `You've used ${budgetUsagePercent.toFixed(0)}% of your monthly token budget. Monitor usage to avoid overages.`}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Tabs for detailed views */}
      <Tabs defaultValue="usage" className="space-y-4">
        <TabsList>
          <TabsTrigger value="usage">Usage by Type</TabsTrigger>
          <TabsTrigger value="agents">Agent Performance</TabsTrigger>
          <TabsTrigger value="trends">Trends</TabsTrigger>
        </TabsList>

        <TabsContent value="usage" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            {/* Usage by Task Type */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Usage by Task Type</CardTitle>
                <CardDescription>Token consumption by operation type</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {stats.usage_by_task_type.map((item) => {
                  const percentage = (item.tokens / stats.tokens_used_this_month) * 100;
                  return (
                    <div key={item.task_type} className="space-y-2">
                      <div className="flex items-center justify-between text-sm">
                        <span className="capitalize">
                          {item.task_type.replace(/_/g, " ")}
                        </span>
                        <span className="text-muted-foreground">
                          {(item.tokens / 1000).toFixed(1)}k tokens ({item.count} tasks)
                        </span>
                      </div>
                      <Progress value={percentage} className="h-2" />
                    </div>
                  );
                })}
              </CardContent>
            </Card>

            {/* Provider Info */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Provider Configuration</CardTitle>
                <CardDescription>Current LLM provider settings</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">Provider</span>
                  <Badge variant="secondary" className="capitalize">
                    {stats.llm_provider}
                  </Badge>
                </div>
                <Separator />
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">Model</span>
                  <span className="text-sm font-mono">{stats.llm_model}</span>
                </div>
                <Separator />
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">Auto Remediation</span>
                  <Badge variant={stats.auto_remediation_enabled ? "default" : "outline"}>
                    {stats.auto_remediation_enabled ? "Enabled" : "Disabled"}
                  </Badge>
                </div>
                <Separator />
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">Autonomy Mode</span>
                  <Badge variant="outline" className="capitalize">
                    {stats.autonomy_mode.replace(/_/g, " ")}
                  </Badge>
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="agents" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Agent Performance</CardTitle>
              <CardDescription>Token usage and efficiency by AI agent</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {stats.usage_by_agent.map((agent) => (
                  <div key={agent.agent} className="flex items-center gap-4 p-3 rounded-lg border">
                    <div className="flex h-10 w-10 items-center justify-center rounded-full bg-brand-accent/10">
                      <Bot className="h-5 w-5 text-brand-accent" />
                    </div>
                    <div className="flex-1">
                      <div className="flex items-center justify-between">
                        <span className="font-medium capitalize">
                          {agent.agent.replace(/_/g, " ")}
                        </span>
                        <Badge variant="outline">{agent.count} tasks</Badge>
                      </div>
                      <div className="flex items-center gap-4 text-sm text-muted-foreground mt-1">
                        <span>{(agent.tokens / 1000).toFixed(1)}k tokens total</span>
                        <span>|</span>
                        <span>{(agent.avg_tokens / 1000).toFixed(1)}k avg/task</span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="trends" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Daily Usage (Last 30 Days)</CardTitle>
              <CardDescription>Token consumption trends over time</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="h-64 flex items-end gap-1">
                {stats.daily_usage.map((day, index) => {
                  const maxTokens = Math.max(...stats.daily_usage.map((d) => d.tokens));
                  const height = (day.tokens / maxTokens) * 100;
                  return (
                    <div
                      key={day.date}
                      className="flex-1 flex flex-col items-center gap-1"
                      title={`${day.date}: ${(day.tokens / 1000).toFixed(1)}k tokens, ${day.tasks} tasks, $${day.cost.toFixed(2)}`}
                    >
                      <div
                        className={cn(
                          "w-full rounded-t transition-all hover:opacity-80",
                          day.tokens > maxTokens * 0.8
                            ? "bg-status-amber"
                            : "bg-brand-accent"
                        )}
                        style={{ height: `${Math.max(height, 2)}%` }}
                      />
                      {index % 5 === 0 && (
                        <span className="text-[10px] text-muted-foreground rotate-45">
                          {new Date(day.date).toLocaleDateString("en-US", {
                            month: "short",
                            day: "numeric",
                          })}
                        </span>
                      )}
                    </div>
                  );
                })}
              </div>
              <div className="flex justify-between text-xs text-muted-foreground mt-4">
                <span>30 days ago</span>
                <span>Today</span>
              </div>
            </CardContent>
          </Card>

          {/* Summary Stats */}
          <div className="grid gap-4 md:grid-cols-3">
            <Card>
              <CardContent className="pt-6">
                <div className="flex items-center gap-2">
                  <TrendingUp className="h-4 w-4 text-status-green" />
                  <span className="text-sm text-muted-foreground">Avg Daily Tokens</span>
                </div>
                <div className="text-2xl font-bold mt-2">
                  {(
                    stats.daily_usage.reduce((sum, d) => sum + d.tokens, 0) /
                    stats.daily_usage.length /
                    1000
                  ).toFixed(1)}
                  k
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="pt-6">
                <div className="flex items-center gap-2">
                  <Calendar className="h-4 w-4 text-brand-accent" />
                  <span className="text-sm text-muted-foreground">Peak Day Usage</span>
                </div>
                <div className="text-2xl font-bold mt-2">
                  {(Math.max(...stats.daily_usage.map((d) => d.tokens)) / 1000).toFixed(1)}k
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="pt-6">
                <div className="flex items-center gap-2">
                  <Zap className="h-4 w-4 text-status-amber" />
                  <span className="text-sm text-muted-foreground">Avg Tasks/Day</span>
                </div>
                <div className="text-2xl font-bold mt-2">
                  {(
                    stats.daily_usage.reduce((sum, d) => sum + d.tasks, 0) /
                    stats.daily_usage.length
                  ).toFixed(1)}
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>
      </Tabs>

      {/* Back Button */}
      <div className="pt-6">
        <Link href="/ai">
          <Button variant="outline">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to AI Copilot
          </Button>
        </Link>
      </div>
    </div>
  );
}
