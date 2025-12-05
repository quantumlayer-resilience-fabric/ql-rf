"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { MetricCard } from "@/components/data/metric-card";
import { PageSkeleton, ErrorState, EmptyState } from "@/components/feedback";
import {
  useCostSummary,
  useCostBreakdown,
  useCostTrend,
  useRecommendations,
  useBudgets,
} from "@/hooks/use-finops";
import { CostTrendChart } from "@/components/finops/cost-trend-chart";
import { CostBreakdownChart } from "@/components/finops/cost-breakdown-chart";
import { CloudSpendCard } from "@/components/finops/cloud-spend-card";
import { RecommendationCard } from "@/components/finops/recommendation-card";
import { BudgetProgressCard } from "@/components/finops/budget-progress-card";
import { formatCurrency } from "@/lib/utils";
import {
  DollarSign,
  TrendingUp,
  TrendingDown,
  Target,
  Lightbulb,
  Plus,
  Download,
  Calendar,
} from "lucide-react";

export default function CostsPage() {
  const [period, setPeriod] = useState<string>("30d");
  const [breakdownDimension, setBreakdownDimension] = useState<
    "cloud" | "service" | "region" | "site" | "resource_type"
  >("cloud");

  // Fetch data
  const { data: costSummary, isLoading, error, refetch } = useCostSummary(period);
  const { data: breakdown } = useCostBreakdown(breakdownDimension, period);
  const { data: trendData } = useCostTrend(30);
  const { data: recommendations } = useRecommendations();
  const { data: budgetsData } = useBudgets(true);

  if (isLoading) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              FinOps
            </h1>
            <p className="text-muted-foreground">
              Cost management and financial operations across cloud platforms.
            </p>
          </div>
        </div>
        <PageSkeleton metricCards={4} showChart={true} showTable={true} tableRows={5} />
      </div>
    );
  }

  if (error) {
    return (
      <div className="page-transition space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            FinOps
          </h1>
          <p className="text-muted-foreground">
            Cost management and financial operations across cloud platforms.
          </p>
        </div>
        <ErrorState
          error={error}
          retry={refetch}
          title="Failed to load cost data"
          description="We couldn't fetch the cost data. Please try again."
        />
      </div>
    );
  }

  if (!costSummary) {
    return null;
  }

  // Calculate metrics
  const totalSpend = costSummary.totalCost;
  const trendChange = costSummary.trendChange;
  const potentialSavings = recommendations?.totalPotentialSavings || 0;

  // Calculate projected month end (assuming we're in the current month)
  const daysInMonth = 30;
  const currentDay = new Date().getDate();
  const projectedMonthEnd = (totalSpend / currentDay) * daysInMonth;

  // Calculate budget utilization
  const activeBudgets = budgetsData?.budgets || [];
  const totalBudget = activeBudgets.reduce((sum, b) => sum + b.amount, 0);
  const budgetUtilization = totalBudget > 0 ? (totalSpend / totalBudget) * 100 : 0;

  // Cloud breakdown for cards
  const cloudSpend = Object.entries(costSummary.byCloud).map(([cloud, amount]) => ({
    cloud,
    amount,
    percentage: totalSpend > 0 ? (amount / totalSpend) * 100 : 0,
  }));

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            FinOps
          </h1>
          <p className="text-muted-foreground">
            Cost management and financial operations across cloud platforms.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Select value={period} onValueChange={setPeriod}>
            <SelectTrigger className="w-[160px]">
              <Calendar className="mr-2 h-4 w-4" />
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="7d">Last 7 days</SelectItem>
              <SelectItem value="30d">Last 30 days</SelectItem>
              <SelectItem value="90d">Last 90 days</SelectItem>
              <SelectItem value="this_month">This month</SelectItem>
              <SelectItem value="last_month">Last month</SelectItem>
            </SelectContent>
          </Select>
          <Button variant="outline" size="sm">
            <Download className="mr-2 h-4 w-4" />
            Export
          </Button>
          <Button size="sm">
            <Plus className="mr-2 h-4 w-4" />
            Create Budget
          </Button>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Total Spend (MTD)"
          value={formatCurrency(totalSpend, costSummary.currency)}
          subtitle={period}
          status={trendChange > 10 ? "warning" : "neutral"}
          icon={<DollarSign className="h-5 w-5" />}
        />
        <MetricCard
          title="Projected Month End"
          value={formatCurrency(projectedMonthEnd, costSummary.currency)}
          subtitle={`${((projectedMonthEnd - totalSpend) / totalSpend * 100).toFixed(0)}% higher`}
          status={projectedMonthEnd > totalSpend * 1.2 ? "warning" : "neutral"}
          icon={<TrendingUp className="h-5 w-5" />}
        />
        <MetricCard
          title="Budget Utilization"
          value={`${budgetUtilization.toFixed(0)}%`}
          subtitle={totalBudget > 0 ? `of ${formatCurrency(totalBudget, costSummary.currency)}` : "No budgets set"}
          status={
            budgetUtilization > 100
              ? "critical"
              : budgetUtilization > 80
              ? "warning"
              : "success"
          }
          icon={<Target className="h-5 w-5" />}
        />
        <MetricCard
          title="Potential Savings"
          value={formatCurrency(potentialSavings, costSummary.currency)}
          subtitle={`${recommendations?.totalRecommendations || 0} recommendations`}
          status="success"
          icon={<Lightbulb className="h-5 w-5" />}
        />
      </div>

      {/* Cloud Breakdown Cards */}
      {cloudSpend.length > 0 && (
        <div className="grid gap-4 md:grid-cols-3">
          {cloudSpend.map(({ cloud, amount, percentage }) => (
            <CloudSpendCard
              key={cloud}
              cloud={cloud}
              amount={amount}
              currency={costSummary.currency}
              percentage={percentage}
            />
          ))}
        </div>
      )}

      {/* Tabs */}
      <Tabs defaultValue="overview" className="space-y-4">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="breakdown">
            By {breakdownDimension}
          </TabsTrigger>
          <TabsTrigger value="budgets">
            Budgets
            {activeBudgets.length > 0 && (
              <Badge variant="secondary" className="ml-2">
                {activeBudgets.length}
              </Badge>
            )}
          </TabsTrigger>
          <TabsTrigger value="recommendations">
            Recommendations
            {recommendations && recommendations.totalRecommendations > 0 && (
              <Badge variant="secondary" className="ml-2">
                {recommendations.totalRecommendations}
              </Badge>
            )}
          </TabsTrigger>
        </TabsList>

        {/* Overview Tab */}
        <TabsContent value="overview" className="space-y-4">
          {trendData && (
            <CostTrendChart data={trendData.trend} currency={costSummary.currency} />
          )}

          {/* Trend Summary */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Cost Trend Analysis</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid gap-4 md:grid-cols-3">
                <div className="space-y-2">
                  <p className="text-sm text-muted-foreground">Period Change</p>
                  <div className="flex items-center gap-2">
                    {trendChange > 0 ? (
                      <TrendingUp className="h-5 w-5 text-status-red" />
                    ) : (
                      <TrendingDown className="h-5 w-5 text-status-green" />
                    )}
                    <span
                      className={`text-2xl font-bold ${
                        trendChange > 0 ? "text-status-red" : "text-status-green"
                      }`}
                    >
                      {trendChange > 0 ? "+" : ""}
                      {trendChange.toFixed(1)}%
                    </span>
                  </div>
                </div>
                <div className="space-y-2">
                  <p className="text-sm text-muted-foreground">Total Services</p>
                  <p className="text-2xl font-bold">
                    {Object.keys(costSummary.byService).length}
                  </p>
                </div>
                <div className="space-y-2">
                  <p className="text-sm text-muted-foreground">Total Resources</p>
                  <p className="text-2xl font-bold">
                    {Object.keys(costSummary.byResource).length}
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Breakdown Tab */}
        <TabsContent value="breakdown" className="space-y-4">
          <div className="flex items-center gap-2">
            <Select
              value={breakdownDimension}
              onValueChange={(v) =>
                setBreakdownDimension(
                  v as "cloud" | "service" | "region" | "site" | "resource_type"
                )
              }
            >
              <SelectTrigger className="w-[180px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="cloud">Cloud</SelectItem>
                <SelectItem value="service">Service</SelectItem>
                <SelectItem value="region">Region</SelectItem>
                <SelectItem value="site">Site</SelectItem>
                <SelectItem value="resource_type">Resource Type</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {breakdown && (
            <CostBreakdownChart
              items={breakdown.items}
              totalCost={breakdown.totalCost}
              currency={breakdown.currency}
              title={`Cost by ${breakdownDimension}`}
            />
          )}
        </TabsContent>

        {/* Budgets Tab */}
        <TabsContent value="budgets" className="space-y-4">
          {activeBudgets.length > 0 ? (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {activeBudgets.map((budget) => (
                <BudgetProgressCard key={budget.id} budget={budget} />
              ))}
            </div>
          ) : (
            <Card>
              <CardContent className="p-8">
                <EmptyState
                  variant="data"
                  title="No budgets configured"
                  description="Create budgets to track spending and receive alerts when thresholds are exceeded."
                />
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {/* Recommendations Tab */}
        <TabsContent value="recommendations" className="space-y-4">
          {recommendations && recommendations.recommendations.length > 0 ? (
            <>
              <div className="rounded-lg border bg-card p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="font-semibold">
                      {recommendations.totalRecommendations} Optimization Opportunities
                    </h3>
                    <p className="text-sm text-muted-foreground">
                      Potential savings:{" "}
                      <span className="font-semibold text-status-green">
                        {formatCurrency(
                          recommendations.totalPotentialSavings,
                          recommendations.currency
                        )}
                        /mo
                      </span>
                    </p>
                  </div>
                  <Button size="sm">
                    <Lightbulb className="mr-2 h-4 w-4" />
                    Apply All High Priority
                  </Button>
                </div>
              </div>

              <div className="space-y-4">
                {recommendations.recommendations.map((rec) => (
                  <RecommendationCard key={rec.id} recommendation={rec} />
                ))}
              </div>
            </>
          ) : (
            <Card>
              <CardContent className="p-8">
                <EmptyState
                  variant="success"
                  title="No recommendations"
                  description="Your infrastructure is optimized. Check back later for new recommendations."
                />
              </CardContent>
            </Card>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}
