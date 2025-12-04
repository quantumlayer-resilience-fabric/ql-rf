"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { MetricCard } from "@/components/data/metric-card";
import { ValueDeliveredCard } from "@/components/data/value-delivered-card";
import { StatusBadge } from "@/components/status/status-badge";
import { PlatformIcon } from "@/components/status/platform-icon";
import { PageSkeleton, ErrorState } from "@/components/feedback";
import { useOverviewMetrics } from "@/hooks/use-overview";
import { useDriftSummary } from "@/hooks/use-drift";
import {
  Server,
  TrendingDown,
  Shield,
  RefreshCw,
  AlertTriangle,
  Clock,
} from "lucide-react";
import {
  mapAlertSeverityToUIStatus,
  mapDriftStatusToUIStatus,
  type AlertSeverity,
  type DriftStatus,
} from "@/lib/api-types";

export default function OverviewPage() {
  const { data: metrics, isLoading, error, refetch } = useOverviewMetrics();
  const { data: driftSummary, isLoading: isDriftLoading } = useDriftSummary();

  if (isLoading) {
    return (
      <div className="page-transition space-y-6">
        <div>
          <h1
            className="text-2xl font-bold tracking-tight text-foreground"
            style={{ fontFamily: "var(--font-display)" }}
          >
            Overview
          </h1>
          <p className="text-muted-foreground">
            Real-time visibility into your infrastructure health and compliance.
          </p>
        </div>
        <PageSkeleton metricCards={4} showChart={false} showTable={false} />
        <div className="grid gap-6 lg:grid-cols-3">
          <PageSkeleton metricCards={0} showChart={false} showTable={true} tableRows={5} />
          <PageSkeleton metricCards={0} showChart={false} showTable={true} tableRows={3} />
          <PageSkeleton metricCards={0} showChart={false} showTable={true} tableRows={5} />
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="page-transition space-y-6">
        <div>
          <h1
            className="text-2xl font-bold tracking-tight text-foreground"
            style={{ fontFamily: "var(--font-display)" }}
          >
            Overview
          </h1>
          <p className="text-muted-foreground">
            Real-time visibility into your infrastructure health and compliance.
          </p>
        </div>
        <ErrorState
          error={error}
          retry={refetch}
          title="Failed to load overview metrics"
          description="We couldn't fetch the latest metrics. Please try again."
        />
      </div>
    );
  }

  // Format values for display
  const fleetSize = metrics?.fleetSize;
  const driftScore = metrics?.driftScore;
  const compliance = metrics?.compliance;
  const drReadiness = metrics?.drReadiness;
  const platformDistribution = metrics?.platformDistribution || [];
  const alertsSummary = metrics?.alerts || [];
  const recentActivity = metrics?.recentActivity || [];

  // Transform alert data for display using generated type mappings
  const alerts = alertsSummary.map((a) => ({
    severity: mapAlertSeverityToUIStatus(a.severity as AlertSeverity),
    count: a.count,
    label: a.severity.charAt(0).toUpperCase() + a.severity.slice(1),
  }));

  // Transform drift API site data for heatmap using generated type mappings
  const siteHeatmap = (driftSummary?.bySite || []).map((site) => ({
    name: site.siteName || site.siteId,
    coverage: site.coverage,
    status: mapDriftStatusToUIStatus(site.status as DriftStatus),
  }));

  // Get status for metrics
  const getDriftStatus = (value: number | undefined) => {
    if (!value) return "neutral";
    if (value >= 95) return "success";
    if (value >= 80) return "warning";
    return "critical";
  };

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div className="animate-in fade-in-0 slide-in-from-bottom-2 duration-500">
        <h1
          className="text-2xl font-bold tracking-tight text-foreground"
          style={{ fontFamily: "var(--font-display)" }}
        >
          Overview
        </h1>
        <p className="text-muted-foreground">
          Real-time visibility into your infrastructure health and compliance.
        </p>
      </div>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4 stagger-children">
        <MetricCard
          title="Fleet Size"
          value={fleetSize?.value.toLocaleString() || "0"}
          subtitle="assets"
          trend={fleetSize?.trend || { direction: "neutral", value: "", period: "" }}
          status="success"
          icon={<Server className="h-5 w-5" />}
        />
        <MetricCard
          title="Drift Score"
          value={`${formatPercentage(driftScore?.value)}%`}
          subtitle="coverage"
          trend={driftScore?.trend || { direction: "neutral", value: "", period: "" }}
          status={getDriftStatus(driftScore?.value)}
          icon={<TrendingDown className="h-5 w-5" />}
        />
        <MetricCard
          title="Compliance"
          value={`${formatPercentage(compliance?.value)}%`}
          subtitle="passing"
          trend={compliance?.trend || { direction: "neutral", value: "", period: "" }}
          status={(compliance?.value || 0) >= 95 ? "success" : "warning"}
          icon={<Shield className="h-5 w-5" />}
        />
        <MetricCard
          title="DR Readiness"
          value={`${formatPercentage(drReadiness?.value)}%`}
          subtitle="ready"
          trend={drReadiness?.trend || { direction: "neutral", value: "", period: "" }}
          status={(drReadiness?.value || 0) >= 90 ? "success" : "warning"}
          icon={<RefreshCw className="h-5 w-5" />}
        />
      </div>

      {/* Value Delivered / ROI Section */}
      <div className="grid gap-6 lg:grid-cols-3">
        <ValueDeliveredCard className="lg:col-span-2 animate-in fade-in-0 slide-in-from-bottom-4 duration-700" />

        {/* Active Alerts - moved here for better layout */}
        <Card variant="elevated" hover="lift" className="animate-in fade-in-0 slide-in-from-bottom-4 duration-700">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle
              className="text-base"
              style={{ fontFamily: "var(--font-display)" }}
            >
              Active Alerts
            </CardTitle>
            <AlertTriangle className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent className="space-y-4">
            {alerts.length > 0 ? (
              <>
                {alerts.map((alert, index) => (
                  <div
                    key={alert.label}
                    className="flex items-center justify-between rounded-lg border border-border p-3 transition-all hover:border-border/80 hover:shadow-sm animate-in fade-in-0"
                    style={{ animationDelay: `${index * 100}ms`, animationFillMode: 'backwards' }}
                  >
                    <div className="flex items-center gap-3">
                      <StatusBadge
                        status={alert.severity}
                        size="sm"
                        glow={alert.severity === "critical"}
                      >
                        {alert.label}
                      </StatusBadge>
                    </div>
                    <span
                      className="text-2xl font-bold tabular-nums"
                      style={{ fontFamily: "var(--font-display)" }}
                    >
                      {alert.count}
                    </span>
                  </div>
                ))}
                <button className="w-full text-center text-sm font-medium text-brand-accent hover:underline transition-colors">
                  View All Alerts →
                </button>
              </>
            ) : (
              <div className="text-center text-sm text-muted-foreground py-4">
                No active alerts
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Main Content Grid */}
      <div className="grid gap-6 lg:grid-cols-2 stagger-children">
        {/* Platform Distribution */}
        <Card variant="elevated" hover="lift">
          <CardHeader>
            <CardTitle
              className="text-base"
              style={{ fontFamily: "var(--font-display)" }}
            >
              Platform Distribution
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {platformDistribution.length > 0 ? (
              platformDistribution.map((item, index) => (
                <div
                  key={item.platform}
                  className="flex items-center gap-3 animate-in fade-in-0 slide-in-from-left-2"
                  style={{ animationDelay: `${index * 100}ms`, animationFillMode: 'backwards' }}
                >
                  <PlatformIcon platform={item.platform} size="sm" />
                  <div className="flex-1">
                    <div className="flex items-center justify-between text-sm">
                      <span className="font-medium capitalize">{item.platform}</span>
                      <span className="text-muted-foreground tabular-nums">
                        {item.count.toLocaleString()}
                      </span>
                    </div>
                    <div className="mt-1.5 h-2 overflow-hidden rounded-full bg-muted">
                      <div
                        className="h-2 rounded-full bg-brand-accent transition-all duration-700 ease-out"
                        style={{ width: `${item.percentage}%` }}
                      />
                    </div>
                  </div>
                </div>
              ))
            ) : (
              <div className="text-center text-sm text-muted-foreground py-4">
                No platform data available
              </div>
            )}
          </CardContent>
        </Card>

        {/* Recent Activity */}
        <Card variant="elevated" hover="lift">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle
              className="text-base"
              style={{ fontFamily: "var(--font-display)" }}
            >
              Recent Activity
            </CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {recentActivity.length > 0 ? (
                recentActivity.map((activity, i) => (
                  <div
                    key={activity.id || i}
                    className="flex items-start gap-3 text-sm animate-in fade-in-0 slide-in-from-right-2"
                    style={{ animationDelay: `${i * 100}ms`, animationFillMode: 'backwards' }}
                  >
                    <div
                      className={`mt-1.5 h-2 w-2 rounded-full shrink-0 ${
                        activity.type === "critical"
                          ? "bg-status-red"
                          : activity.type === "warning"
                          ? "bg-status-amber"
                          : activity.type === "success"
                          ? "bg-status-green"
                          : "bg-brand-accent"
                      }`}
                    />
                    <div className="flex-1 min-w-0">
                      <div className="font-medium truncate">{activity.action}</div>
                      <div className="text-muted-foreground truncate">
                        {activity.detail}
                      </div>
                    </div>
                    <span className="text-xs text-muted-foreground whitespace-nowrap">
                      {formatRelativeTime(activity.timestamp)}
                    </span>
                  </div>
                ))
              ) : (
                <div className="text-center text-sm text-muted-foreground py-4">
                  No recent activity
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Drift Heatmap */}
      <Card variant="elevated" className="animate-in fade-in-0 slide-in-from-bottom-4 duration-700">
        <CardHeader>
          <CardTitle
            className="text-base"
            style={{ fontFamily: "var(--font-display)" }}
          >
            Drift Heatmap by Site
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isDriftLoading ? (
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-8">
              {[...Array(8)].map((_, i) => (
                <div
                  key={i}
                  className="animate-pulse rounded-xl border border-border bg-muted p-4 text-center"
                >
                  <div className="h-8 w-12 mx-auto bg-muted-foreground/20 rounded" />
                  <div className="mt-2 h-3 w-16 mx-auto bg-muted-foreground/20 rounded" />
                </div>
              ))}
            </div>
          ) : siteHeatmap.length > 0 ? (
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-8">
              {siteHeatmap.map((site, index) => (
                <div
                  key={site.name}
                  className={`group cursor-pointer rounded-xl border p-4 text-center transition-all duration-300 hover:scale-[1.02] animate-in fade-in-0 ${
                    site.status === "success"
                      ? "border-status-green/30 bg-status-green/10 hover:border-status-green/50 hover:shadow-[0_0_12px_rgba(5,150,105,0.15)]"
                      : site.status === "warning"
                      ? "border-status-amber/30 bg-status-amber/10 hover:border-status-amber/50 hover:shadow-[0_0_12px_rgba(217,119,6,0.15)]"
                      : "border-status-red/30 bg-status-red/10 hover:border-status-red/50 hover:shadow-[0_0_12px_rgba(220,38,38,0.2)]"
                  }`}
                  style={{ animationDelay: `${index * 50}ms`, animationFillMode: 'backwards' }}
                >
                  <div
                    className={`text-2xl font-bold tabular-nums ${
                      site.status === "success"
                        ? "text-status-green"
                        : site.status === "warning"
                        ? "text-status-amber"
                        : "text-status-red"
                    }`}
                    style={{ fontFamily: "var(--font-display)" }}
                  >
                    {formatPercentage(site.coverage)}%
                  </div>
                  <div className="mt-1 text-xs text-muted-foreground truncate">
                    {site.name}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center text-sm text-muted-foreground py-8">
              No site drift data available. Sites with assets will appear here.
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

// Helper function to format percentages with max 1 decimal place
function formatPercentage(value: number | undefined | null): string {
  if (value === undefined || value === null) return "0";
  // Round to 1 decimal place
  return Number(value.toFixed(1)).toString();
}

// Helper function to format timestamps
function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return "just now";
  if (diffMins < 60) return `${diffMins}m ago`;

  const diffHours = Math.floor(diffMins / 60);
  if (diffHours < 24) return `${diffHours}h ago`;

  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 7) return `${diffDays}d ago`;

  return date.toLocaleDateString();
}
