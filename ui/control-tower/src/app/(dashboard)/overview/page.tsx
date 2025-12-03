"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { MetricCard } from "@/components/data/metric-card";
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

export default function OverviewPage() {
  const { data: metrics, isLoading, error, refetch } = useOverviewMetrics();
  const { data: driftSummary, isLoading: isDriftLoading } = useDriftSummary();

  if (isLoading) {
    return (
      <div className="page-transition space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">Overview</h1>
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
          <h1 className="text-2xl font-bold tracking-tight text-foreground">Overview</h1>
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

  // Transform alert data for display
  const alerts = alertsSummary.map((a) => ({
    severity: a.severity,
    count: a.count,
    label: a.severity.charAt(0).toUpperCase() + a.severity.slice(1),
  }));

  // Transform drift API site data for heatmap
  const siteHeatmap = (driftSummary?.bySite || []).map((site) => ({
    name: site.siteName || site.siteId,
    coverage: site.coverage,
    status: site.status,
  }));

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-foreground">Overview</h1>
        <p className="text-muted-foreground">
          Real-time visibility into your infrastructure health and compliance.
        </p>
      </div>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
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
          value={`${driftScore?.value || 0}%`}
          subtitle="coverage"
          trend={driftScore?.trend || { direction: "neutral", value: "", period: "" }}
          status={(driftScore?.value || 0) > 90 ? "success" : "warning"}
          icon={<TrendingDown className="h-5 w-5" />}
        />
        <MetricCard
          title="Compliance"
          value={`${compliance?.value || 0}%`}
          subtitle="passing"
          trend={compliance?.trend || { direction: "neutral", value: "", period: "" }}
          status="success"
          icon={<Shield className="h-5 w-5" />}
        />
        <MetricCard
          title="DR Readiness"
          value={`${drReadiness?.value || 0}%`}
          subtitle="ready"
          trend={drReadiness?.trend || { direction: "neutral", value: "", period: "" }}
          status="success"
          icon={<RefreshCw className="h-5 w-5" />}
        />
      </div>

      {/* Main Content Grid */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Platform Distribution */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Platform Distribution</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {platformDistribution.length > 0 ? (
              platformDistribution.map((item) => (
                <div key={item.platform} className="flex items-center gap-3">
                  <PlatformIcon platform={item.platform} size="sm" />
                  <div className="flex-1">
                    <div className="flex items-center justify-between text-sm">
                      <span className="font-medium capitalize">{item.platform}</span>
                      <span className="text-muted-foreground">{item.count.toLocaleString()}</span>
                    </div>
                    <div className="mt-1 h-2 rounded-full bg-muted">
                      <div
                        className="h-2 rounded-full bg-brand-accent"
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

        {/* Active Alerts */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-base">Active Alerts</CardTitle>
            <AlertTriangle className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent className="space-y-4">
            {alerts.length > 0 ? (
              <>
                {alerts.map((alert) => (
                  <div
                    key={alert.label}
                    className="flex items-center justify-between rounded-lg border border-border p-3"
                  >
                    <div className="flex items-center gap-3">
                      <StatusBadge status={alert.severity} size="sm">
                        {alert.label}
                      </StatusBadge>
                    </div>
                    <span className="text-2xl font-bold">{alert.count}</span>
                  </div>
                ))}
                <button className="w-full text-center text-sm text-brand-accent hover:underline">
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

        {/* Recent Activity */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-base">Recent Activity</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {recentActivity.length > 0 ? (
                recentActivity.map((activity, i) => (
                  <div key={activity.id || i} className="flex items-start gap-3 text-sm">
                    <div
                      className={`mt-1.5 h-2 w-2 rounded-full ${
                        activity.type === "critical"
                          ? "bg-status-red"
                          : activity.type === "warning"
                          ? "bg-status-amber"
                          : activity.type === "success"
                          ? "bg-status-green"
                          : "bg-brand-accent"
                      }`}
                    />
                    <div className="flex-1">
                      <div className="font-medium">{activity.action}</div>
                      <div className="text-muted-foreground">{activity.detail}</div>
                    </div>
                    <span className="text-xs text-muted-foreground">
                      {formatRelativeTime(activity.createdAt)}
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
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Drift Heatmap by Site</CardTitle>
        </CardHeader>
        <CardContent>
          {isDriftLoading ? (
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-8">
              {[...Array(8)].map((_, i) => (
                <div
                  key={i}
                  className="animate-pulse rounded-lg border border-border bg-muted p-4 text-center"
                >
                  <div className="h-8 w-12 mx-auto bg-muted-foreground/20 rounded" />
                  <div className="mt-2 h-3 w-16 mx-auto bg-muted-foreground/20 rounded" />
                </div>
              ))}
            </div>
          ) : siteHeatmap.length > 0 ? (
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-8">
              {siteHeatmap.map((site) => (
                <div
                  key={site.name}
                  className={`cursor-pointer rounded-lg border p-4 text-center transition-colors hover:border-brand-accent ${
                    site.status === "success"
                      ? "border-status-green/30 bg-status-green-bg"
                      : site.status === "warning"
                      ? "border-status-amber/30 bg-status-amber-bg"
                      : "border-status-red/30 bg-status-red-bg"
                  }`}
                >
                  <div
                    className={`text-2xl font-bold ${
                      site.status === "success"
                        ? "text-status-green"
                        : site.status === "warning"
                        ? "text-status-amber"
                        : "text-status-red"
                    }`}
                  >
                    {site.coverage.toFixed(1)}%
                  </div>
                  <div className="mt-1 text-xs text-muted-foreground">{site.name}</div>
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
