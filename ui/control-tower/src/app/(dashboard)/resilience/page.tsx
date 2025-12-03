"use client";

import { useCallback } from "react";
import { useRouter } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { MetricCard } from "@/components/data/metric-card";
import { StatusBadge } from "@/components/status/status-badge";
import { PlatformIcon } from "@/components/status/platform-icon";
import { PageSkeleton, ErrorState, EmptyState } from "@/components/feedback";
import { PermissionGate } from "@/components/auth/permission-gate";
import { Permissions } from "@/hooks/use-permissions";
import { useResilienceSummary, useTriggerFailoverTest, useTriggerDRSync } from "@/hooks/use-resilience";
import {
  RefreshCw,
  Shield,
  Clock,
  Activity,
  Play,
  CheckCircle,
  AlertTriangle,
  ArrowRight,
  Calendar,
  Target,
  Zap,
  Server,
  Link2,
  History,
  Loader2,
  ShieldAlert,
} from "lucide-react";

export default function ResiliencePage() {
  const router = useRouter();
  const { data: resilienceData, isLoading, error, refetch } = useResilienceSummary();
  const triggerFailoverTest = useTriggerFailoverTest();
  const triggerSync = useTriggerDRSync();

  const handleConfigureDR = useCallback(() => {
    router.push("/sites?view=topology");
  }, [router]);

  const handleFailoverTest = (pairId: string) => {
    triggerFailoverTest.mutate(pairId, {
      onSuccess: () => {
        refetch();
      },
    });
  };

  const handleSync = (pairId: string) => {
    triggerSync.mutate(pairId, {
      onSuccess: () => {
        refetch();
      },
    });
  };

  if (isLoading) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Resilience & DR
            </h1>
            <p className="text-muted-foreground">
              Monitor disaster recovery readiness and run failover drills.
            </p>
          </div>
        </div>
        <PageSkeleton metricCards={5} showChart={false} showTable={true} tableRows={4} />
      </div>
    );
  }

  if (error) {
    return (
      <div className="page-transition space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Resilience & DR
          </h1>
          <p className="text-muted-foreground">
            Monitor disaster recovery readiness and run failover drills.
          </p>
        </div>
        <ErrorState
          error={error}
          retry={refetch}
          title="Failed to load resilience data"
          description="We couldn't fetch the DR status. Please try again."
        />
      </div>
    );
  }

  // Extract data from API response with fallbacks
  const drMetrics = {
    readiness: resilienceData?.drReadiness || 0,
    rpoCompliance: resilienceData?.rpoCompliance || 0,
    rtoCompliance: resilienceData?.rtoCompliance || 0,
    lastDrill: resilienceData?.lastFailoverTest ? formatRelativeTime(resilienceData.lastFailoverTest) : "Never",
    drPairs: resilienceData?.totalPairs || 0,
    healthyPairs: resilienceData?.healthyPairs || 0,
  };

  const drPairs = resilienceData?.drPairs || [];
  const unpairedSites = resilienceData?.unpairedSites || [];

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Resilience & DR
          </h1>
          <p className="text-muted-foreground">
            Monitor disaster recovery readiness and run failover drills.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm">
            <History className="mr-2 h-4 w-4" />
            Drill History
          </Button>
          <PermissionGate permission={Permissions.TRIGGER_DRILL}>
            <Button size="sm">
              <Play className="mr-2 h-4 w-4" />
              Run DR Drill
            </Button>
          </PermissionGate>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
        <MetricCard
          title="DR Readiness"
          value={`${drMetrics.readiness.toFixed(1)}%`}
          subtitle="protected"
          status={drMetrics.readiness >= 95 ? "success" : drMetrics.readiness >= 80 ? "warning" : "critical"}
          icon={<Shield className="h-5 w-5" />}
        />
        <MetricCard
          title="RTO Compliance"
          value={`${drMetrics.rtoCompliance.toFixed(1)}%`}
          subtitle="meeting target"
          status={drMetrics.rtoCompliance >= 95 ? "success" : drMetrics.rtoCompliance >= 80 ? "warning" : "critical"}
          icon={<Clock className="h-5 w-5" />}
        />
        <MetricCard
          title="RPO Compliance"
          value={`${drMetrics.rpoCompliance.toFixed(1)}%`}
          subtitle="meeting target"
          status={drMetrics.rpoCompliance >= 95 ? "success" : drMetrics.rpoCompliance >= 80 ? "warning" : "critical"}
          icon={<Activity className="h-5 w-5" />}
        />
        <MetricCard
          title="DR Pairs"
          value={`${drMetrics.healthyPairs}/${drMetrics.drPairs}`}
          subtitle="healthy"
          status={drMetrics.healthyPairs === drMetrics.drPairs ? "success" : "warning"}
          icon={<Link2 className="h-5 w-5" />}
        />
        <MetricCard
          title="Last DR Drill"
          value={drMetrics.lastDrill}
          subtitle="completed"
          status="neutral"
          icon={<Calendar className="h-5 w-5" />}
        />
      </div>

      {/* Tabs */}
      <Tabs defaultValue="pairs" className="space-y-4">
        <TabsList>
          <TabsTrigger value="pairs">DR Pairs</TabsTrigger>
          <TabsTrigger value="unprotected">Unpaired Sites</TabsTrigger>
        </TabsList>

        {/* DR Pairs Tab */}
        <TabsContent value="pairs" className="space-y-4">
          {drPairs.length > 0 ? (
            drPairs.map((pair) => (
              <Card key={pair.id}>
                <CardContent className="p-6">
                  <div className="flex items-center gap-6">
                    {/* Primary Site */}
                    <div className="flex-1 rounded-lg border p-4">
                      <div className="flex items-center gap-3">
                        <PlatformIcon platform={pair.primarySite.platform} size="md" />
                        <div>
                          <div className="flex items-center gap-2">
                            <h4 className="font-semibold">{pair.primarySite.name}</h4>
                            <Badge variant="secondary">Primary</Badge>
                          </div>
                          <p className="text-sm text-muted-foreground">
                            {pair.primarySite.assetCount.toLocaleString()} assets • {pair.primarySite.region}
                          </p>
                        </div>
                      </div>
                    </div>

                    {/* Sync Status */}
                    <div className="flex flex-col items-center gap-2">
                      <div className="flex items-center gap-2">
                        <ArrowRight className="h-5 w-5 text-muted-foreground" />
                        <RefreshCw
                          className={`h-5 w-5 ${
                            pair.status === "syncing"
                              ? "animate-spin text-brand-accent"
                              : pair.replicationStatus === "in-sync"
                              ? "text-status-green"
                              : "text-status-amber"
                          }`}
                        />
                        <ArrowRight className="h-5 w-5 rotate-180 text-muted-foreground" />
                      </div>
                      <StatusBadge
                        status={
                          pair.replicationStatus === "in-sync"
                            ? "success"
                            : pair.replicationStatus === "lagging"
                            ? "warning"
                            : "critical"
                        }
                        size="sm"
                      >
                        {pair.replicationStatus}
                      </StatusBadge>
                      <span className="text-xs text-muted-foreground">
                        {formatRelativeTime(pair.drSite.lastSyncAt)}
                      </span>
                    </div>

                    {/* DR Site */}
                    <div className="flex-1 rounded-lg border border-purple-500/30 bg-purple-500/5 p-4">
                      <div className="flex items-center gap-3">
                        <PlatformIcon platform={pair.drSite.platform} size="md" />
                        <div>
                          <div className="flex items-center gap-2">
                            <h4 className="font-semibold">{pair.drSite.name}</h4>
                            <Badge variant="outline" className="text-purple-500 border-purple-500/30">
                              DR
                            </Badge>
                          </div>
                          <p className="text-sm text-muted-foreground">
                            {pair.drSite.assetCount.toLocaleString()} assets • {pair.drSite.region}
                          </p>
                        </div>
                      </div>
                    </div>

                    {/* Metrics */}
                    <div className="flex gap-6 border-l pl-6">
                      <div className="text-center">
                        <div className="text-lg font-bold">{pair.primarySite.rto}</div>
                        <div className="text-xs text-muted-foreground">RTO</div>
                      </div>
                      <div className="text-center">
                        <div className="text-lg font-bold">{pair.primarySite.rpo}</div>
                        <div className="text-xs text-muted-foreground">RPO</div>
                      </div>
                      <div className="text-center">
                        <StatusBadge
                          status={pair.status === "healthy" ? "success" : pair.status === "warning" ? "warning" : "critical"}
                          size="sm"
                        >
                          {pair.status}
                        </StatusBadge>
                        {pair.lastFailoverTest && (
                          <div className="text-xs text-muted-foreground mt-1">
                            {formatRelativeTime(pair.lastFailoverTest)}
                          </div>
                        )}
                      </div>
                    </div>

                    {/* Actions */}
                    <PermissionGate
                      permission={Permissions.TRIGGER_DRILL}
                      fallback={
                        <div className="flex flex-col gap-2 text-center">
                          <ShieldAlert className="h-4 w-4 mx-auto text-muted-foreground" />
                          <span className="text-xs text-muted-foreground">No permission</span>
                        </div>
                      }
                    >
                      <div className="flex flex-col gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleFailoverTest(pair.id)}
                          disabled={triggerFailoverTest.isPending}
                        >
                          {triggerFailoverTest.isPending ? (
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          ) : (
                            <Play className="mr-2 h-4 w-4" />
                          )}
                          Test
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleSync(pair.id)}
                          disabled={triggerSync.isPending}
                        >
                          {triggerSync.isPending ? (
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          ) : (
                            <RefreshCw className="mr-2 h-4 w-4" />
                          )}
                          Sync
                        </Button>
                      </div>
                    </PermissionGate>
                  </div>
                </CardContent>
              </Card>
            ))
          ) : (
            <Card>
              <CardContent className="p-8">
                <EmptyState
                  variant="data"
                  title="No DR pairs configured"
                  description="Configure disaster recovery pairs to enable failover capabilities."
                  action={{
                    label: "Configure DR",
                    onClick: handleConfigureDR,
                  }}
                />
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {/* Unpaired Sites Tab */}
        <TabsContent value="unprotected" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-base">
                  Unpaired Sites ({unpairedSites.length})
                </CardTitle>
                <Button variant="outline" size="sm">
                  Configure DR
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {unpairedSites.length > 0 ? (
                <div className="rounded-lg border">
                  <table className="w-full">
                    <thead>
                      <tr className="border-b bg-muted/50">
                        <th className="px-4 py-3 text-left text-sm font-medium">Site</th>
                        <th className="px-4 py-3 text-left text-sm font-medium">Region</th>
                        <th className="px-4 py-3 text-left text-sm font-medium">Platform</th>
                        <th className="px-4 py-3 text-left text-sm font-medium">Assets</th>
                        <th className="px-4 py-3 text-left text-sm font-medium">Status</th>
                        <th className="px-4 py-3 text-right text-sm font-medium">Action</th>
                      </tr>
                    </thead>
                    <tbody>
                      {unpairedSites.map((site, i) => (
                        <tr
                          key={site.id}
                          className={i !== unpairedSites.length - 1 ? "border-b" : ""}
                        >
                          <td className="px-4 py-3">
                            <div className="flex items-center gap-2">
                              <PlatformIcon platform={site.platform} size="sm" />
                              <span className="font-medium">{site.name}</span>
                            </div>
                          </td>
                          <td className="px-4 py-3 text-sm text-muted-foreground">{site.region}</td>
                          <td className="px-4 py-3 text-sm capitalize">{site.platform}</td>
                          <td className="px-4 py-3 text-sm">{site.assetCount.toLocaleString()}</td>
                          <td className="px-4 py-3">
                            <StatusBadge
                              status={site.status === "healthy" ? "success" : site.status === "warning" ? "warning" : "critical"}
                              size="sm"
                            >
                              {site.status}
                            </StatusBadge>
                          </td>
                          <td className="px-4 py-3 text-right">
                            <Button variant="outline" size="sm">
                              Add DR Pair
                            </Button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <EmptyState
                  variant="success"
                  title="All sites protected"
                  description="All sites have DR pairs configured."
                />
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return "just now";
  if (diffMins < 60) return `${diffMins} min ago`;

  const diffHours = Math.floor(diffMins / 60);
  if (diffHours < 24) return `${diffHours}h ago`;

  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 7) return `${diffDays} days ago`;
  if (diffDays < 30) return `${Math.floor(diffDays / 7)} weeks ago`;

  return date.toLocaleDateString();
}
