"use client";

import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { MetricCard } from "@/components/data/metric-card";
import { StatusBadge } from "@/components/status/status-badge";
import { PlatformIcon } from "@/components/status/platform-icon";
import { Heatmap, DriftBar } from "@/components/charts/heatmap";
import { Badge } from "@/components/ui/badge";
import { PageSkeleton, ErrorState } from "@/components/feedback";
import { useDriftSummary, useTopOffenders, useTriggerDriftScan, DriftFilters } from "@/hooks/use-drift";
import { useSendAIMessage, useAIContext, usePendingTasks } from "@/hooks/use-ai";
import {
  TrendingDown,
  AlertTriangle,
  Filter,
  Download,
  RefreshCw,
  Sparkles,
  ChevronRight,
  Clock,
  Server,
  Loader2,
  Zap,
} from "lucide-react";

export default function DriftPage() {
  const router = useRouter();
  const [selectedEnv, setSelectedEnv] = useState<string>("all");
  const [selectedPlatform, setSelectedPlatform] = useState<string>("all");
  const [isCreatingAITask, setIsCreatingAITask] = useState(false);

  // AI hooks
  const aiContext = useAIContext();
  const sendAIMessage = useSendAIMessage();
  const { data: pendingTasks = [] } = usePendingTasks();

  // Check if there's already a pending drift task
  const hasPendingDriftTask = pendingTasks.some(
    (task) => task.user_intent?.toLowerCase().includes("drift")
  );

  // Build filters object for hooks
  const filters: DriftFilters = {
    environment: selectedEnv,
    platform: selectedPlatform,
  };

  const { data: driftSummary, isLoading: summaryLoading, error: summaryError, refetch: refetchSummary } = useDriftSummary(filters);

  const handleSiteCellClick = useCallback((cell: { label: string; value: number }) => {
    // Navigate to sites page filtered by the selected site
    router.push(`/sites?search=${encodeURIComponent(cell.label)}`);
  }, [router]);
  const { data: topOffenders, isLoading: offendersLoading, error: offendersError, refetch: refetchOffenders } = useTopOffenders(10, filters);
  const triggerScan = useTriggerDriftScan();

  const isLoading = summaryLoading || offendersLoading;
  const error = summaryError || offendersError;

  const handleRefresh = () => {
    triggerScan.mutate(undefined, {
      onSuccess: () => {
        refetchSummary();
        refetchOffenders();
      },
    });
  };

  // Export drift data to CSV
  const handleExport = useCallback(() => {
    if (!topOffenders || topOffenders.length === 0) {
      return;
    }

    // Build CSV content
    const headers = ["Hostname", "Site", "Platform", "Environment", "Current Image", "Golden Image", "Status", "Drift Detected"];
    const rows = topOffenders.map(asset => [
      asset.hostname || asset.instanceId || "",
      asset.siteName || asset.site || "",
      asset.platform || "",
      asset.environment || "",
      asset.currentImageVersion || "",
      asset.goldenImageVersion || "",
      asset.isDrifted ? "Drifted" : "Compliant",
      asset.driftDetectedAt || "",
    ]);

    const csvContent = [
      headers.join(","),
      ...rows.map(row => row.map(cell => `"${String(cell).replace(/"/g, '""')}"`).join(","))
    ].join("\n");

    // Download file
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
    const link = document.createElement("a");
    const url = URL.createObjectURL(blob);
    link.setAttribute("href", url);
    link.setAttribute("download", `drift-report-${new Date().toISOString().split("T")[0]}.csv`);
    link.style.visibility = "hidden";
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  }, [topOffenders]);

  if (isLoading) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Drift Analysis
            </h1>
            <p className="text-muted-foreground">
              Monitor and remediate configuration drift across your infrastructure.
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
            Drift Analysis
          </h1>
          <p className="text-muted-foreground">
            Monitor and remediate configuration drift across your infrastructure.
          </p>
        </div>
        <ErrorState
          error={error}
          retry={() => {
            refetchSummary();
            refetchOffenders();
          }}
          title="Failed to load drift data"
          description="We couldn't fetch the drift analysis data. Please try again."
        />
      </div>
    );
  }

  // Extract data from API response
  const driftMetrics = {
    totalAssets: driftSummary?.totalAssets || 0,
    compliant: driftSummary?.compliantAssets || 0,
    drifted: driftSummary?.driftedAssets || 0,
    driftPercentage: driftSummary?.driftPercentage || 0,
    criticalDrift: driftSummary?.criticalDrift || 0,
    avgDriftAge: driftSummary?.averageDriftAge || "N/A",
  };

  // Transform environment data - deduplicate by environment name
  const envMap = new Map<string, { compliant: number; total: number }>();
  (driftSummary?.byEnvironment || []).forEach((env) => {
    const envName = env.environment.charAt(0).toUpperCase() + env.environment.slice(1);
    const existing = envMap.get(envName);
    if (existing) {
      existing.compliant += env.compliant;
      existing.total += env.total;
    } else {
      envMap.set(envName, { compliant: env.compliant, total: env.total });
    }
  });
  const driftByEnvironment = Array.from(envMap.entries()).map(([envName, data]) => {
    const percentage = data.total > 0 ? (data.compliant / data.total) * 100 : 100;
    return {
      env: envName,
      compliant: data.compliant,
      total: data.total,
      status: percentage >= 95 ? "success" as const : percentage >= 80 ? "warning" as const : "critical" as const,
    };
  });

  // Transform site data for heatmap - normalize status to expected values
  const normalizeStatus = (status: string): "success" | "warning" | "critical" => {
    const s = status?.toLowerCase();
    if (s === "critical" || s === "error" || s === "danger") return "critical";
    if (s === "warning" || s === "warn") return "warning";
    return "success";
  };

  const siteHeatmap = (driftSummary?.bySite || []).map((site) => ({
    id: site.siteId,
    label: site.siteName,
    value: site.coverage,
    status: normalizeStatus(site.status),
    metadata: { siteId: site.siteId },
  }));

  // Transform age distribution
  const ageDistribution = (driftSummary?.byAge || []).map((item) => ({
    range: item.range,
    count: item.count,
    percentage: item.percentage,
  }));

  // Handle AI remediation request
  const handleAIRemediation = async () => {
    setIsCreatingAITask(true);
    try {
      const intent = driftMetrics.criticalDrift > 0
        ? `Analyze and remediate critical drift on ${driftMetrics.criticalDrift} assets. Current drift rate is ${driftMetrics.driftPercentage.toFixed(1)}%.`
        : `Review drift status across ${driftMetrics.totalAssets} assets and suggest optimizations.`;

      await sendAIMessage.mutateAsync({
        message: intent,
        context: aiContext,
      });
      router.push("/ai");
    } catch (error) {
      console.error("Failed to create AI task:", error);
    } finally {
      setIsCreatingAITask(false);
    }
  };

  // Dynamic AI insight based on actual data
  const aiInsight = {
    title: driftMetrics.criticalDrift > 0 ? "Critical Drift Detected" : "Drift Pattern Analysis",
    description: driftMetrics.criticalDrift > 0
      ? `${driftMetrics.criticalDrift} critical assets and ${driftMetrics.drifted} total drifted assets require attention. AI can generate a phased remediation plan.`
      : driftMetrics.drifted > 0
      ? `${driftMetrics.drifted} assets have drifted from golden images. AI can analyze patterns and suggest preventive measures.`
      : "All systems are within acceptable drift thresholds. Continue monitoring for changes.",
    severity: driftMetrics.criticalDrift > 0 ? "critical" : driftMetrics.drifted > 0 ? "warning" : "success",
    confidence: 94,
  };

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Drift Analysis
          </h1>
          <p className="text-muted-foreground">
            Monitor and remediate configuration drift across your infrastructure.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={handleExport}
            disabled={!topOffenders || topOffenders.length === 0}
          >
            <Download className="mr-2 h-4 w-4" />
            Export
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={handleRefresh}
            disabled={triggerScan.isPending}
          >
            {triggerScan.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <RefreshCw className="mr-2 h-4 w-4" />
            )}
            Refresh
          </Button>
        </div>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="flex items-center gap-4 p-4">
          <Filter className="h-4 w-4 text-muted-foreground" />
          <Select value={selectedEnv} onValueChange={setSelectedEnv}>
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="Environment" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Environments</SelectItem>
              <SelectItem value="production">Production</SelectItem>
              <SelectItem value="staging">Staging</SelectItem>
              <SelectItem value="development">Development</SelectItem>
              <SelectItem value="dr">DR</SelectItem>
            </SelectContent>
          </Select>
          <Select value={selectedPlatform} onValueChange={setSelectedPlatform}>
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="Platform" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Platforms</SelectItem>
              <SelectItem value="aws">AWS</SelectItem>
              <SelectItem value="azure">Azure</SelectItem>
              <SelectItem value="gcp">GCP</SelectItem>
              <SelectItem value="vsphere">vSphere</SelectItem>
              <SelectItem value="k8s">Kubernetes</SelectItem>
            </SelectContent>
          </Select>
          <div className="flex-1" />
          <span className="text-sm text-muted-foreground">
            Last scan: just now
          </span>
        </CardContent>
      </Card>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Total Assets"
          value={driftMetrics.totalAssets.toLocaleString()}
          subtitle="monitored"
          status="neutral"
          icon={<Server className="h-5 w-5" />}
        />
        <MetricCard
          title="Drift Rate"
          value={`${driftMetrics.driftPercentage.toFixed(1)}%`}
          subtitle={`${driftMetrics.drifted} assets`}
          trend={{ direction: "up", value: "+0.3%", period: "24h" }}
          status={driftMetrics.driftPercentage > 10 ? "critical" : driftMetrics.driftPercentage > 5 ? "warning" : "success"}
          icon={<TrendingDown className="h-5 w-5" />}
        />
        <MetricCard
          title="Critical Drift"
          value={driftMetrics.criticalDrift}
          subtitle="require action"
          status={driftMetrics.criticalDrift > 0 ? "critical" : "success"}
          icon={<AlertTriangle className="h-5 w-5" />}
        />
        <MetricCard
          title="Avg Drift Age"
          value={driftMetrics.avgDriftAge}
          subtitle="since detection"
          trend={{ direction: "down", value: "-0.5d", period: "7d" }}
          status="success"
          icon={<Clock className="h-5 w-5" />}
        />
      </div>

      {/* AI Insight Card */}
      <Card className={`border-l-4 ${
        aiInsight.severity === "critical"
          ? "border-l-status-red bg-gradient-to-r from-status-red/5 to-transparent"
          : aiInsight.severity === "warning"
          ? "border-l-status-amber bg-gradient-to-r from-status-amber/5 to-transparent"
          : "border-l-brand-accent bg-gradient-to-r from-brand-accent/5 to-transparent"
      }`}>
        <CardContent className="flex items-start gap-4 p-6">
          <div className={`rounded-lg p-2 ${
            aiInsight.severity === "critical"
              ? "bg-status-red/10"
              : aiInsight.severity === "warning"
              ? "bg-status-amber/10"
              : "bg-brand-accent/10"
          }`}>
            <Sparkles className={`h-5 w-5 ${
              aiInsight.severity === "critical"
                ? "text-status-red"
                : aiInsight.severity === "warning"
                ? "text-status-amber"
                : "text-brand-accent"
            }`} />
          </div>
          <div className="flex-1">
            <div className="flex items-center gap-2">
              <h3 className="font-semibold">{aiInsight.title}</h3>
              <Badge
                variant="outline"
                className={`text-xs ${
                  aiInsight.severity === "critical"
                    ? "border-status-red/50 text-status-red"
                    : aiInsight.severity === "warning"
                    ? "border-status-amber/50 text-status-amber"
                    : ""
                }`}
              >
                {aiInsight.severity}
              </Badge>
            </div>
            <p className="mt-1 text-sm text-muted-foreground">
              {aiInsight.description}
            </p>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            {hasPendingDriftTask ? (
              <Button
                size="sm"
                variant="outline"
                onClick={() => router.push("/ai")}
              >
                <Clock className="mr-2 h-4 w-4" />
                View Pending Task
              </Button>
            ) : (
              <Button
                size="sm"
                onClick={handleAIRemediation}
                disabled={isCreatingAITask || aiInsight.severity === "success"}
                className={aiInsight.severity === "critical" ? "bg-status-red hover:bg-status-red/90" : ""}
              >
                {isCreatingAITask ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Creating...
                  </>
                ) : (
                  <>
                    <Zap className="mr-2 h-4 w-4" />
                    Remediate with AI
                  </>
                )}
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Main Content Grid */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Drift by Environment */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Drift by Environment</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {driftByEnvironment.length > 0 ? (
              driftByEnvironment.map((env) => (
                <DriftBar
                  key={env.env}
                  label={env.env}
                  value={env.compliant}
                  total={env.total}
                  status={env.status}
                />
              ))
            ) : (
              <div className="text-center text-sm text-muted-foreground py-4">
                No environment data available
              </div>
            )}
          </CardContent>
        </Card>

        {/* Age Distribution */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Drift Age Distribution</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {ageDistribution.length > 0 ? (
                ageDistribution.map((item) => (
                  <div key={item.range} className="flex items-center gap-4">
                    <span className="w-24 text-sm text-muted-foreground">
                      {item.range}
                    </span>
                    <div className="flex-1">
                      <div className="h-6 rounded bg-muted overflow-hidden">
                        <div
                          className="h-full bg-brand-accent/60 rounded flex items-center justify-end pr-2"
                          style={{ width: `${item.percentage}%` }}
                        >
                          {item.percentage > 15 && (
                            <span className="text-xs font-medium text-white">
                              {item.count}
                            </span>
                          )}
                        </div>
                      </div>
                    </div>
                    {item.percentage <= 15 && (
                      <span className="text-sm font-medium">{item.count}</span>
                    )}
                  </div>
                ))
              ) : (
                <div className="text-center text-sm text-muted-foreground py-4">
                  No age distribution data available
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Site Heatmap */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Coverage by Site</CardTitle>
        </CardHeader>
        <CardContent>
          {siteHeatmap.length > 0 ? (
            <Heatmap
              data={siteHeatmap}
              columns={6}
              onCellClick={handleSiteCellClick}
            />
          ) : (
            <div className="text-center text-sm text-muted-foreground py-8">
              No site coverage data available
            </div>
          )}
        </CardContent>
      </Card>

      {/* Top Offenders */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-base">Top Offenders</CardTitle>
          <Button variant="ghost" size="sm">
            View All <ChevronRight className="ml-1 h-4 w-4" />
          </Button>
        </CardHeader>
        <CardContent>
          {topOffenders && topOffenders.length > 0 ? (
            <div className="rounded-lg border">
              <table className="w-full">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="px-4 py-3 text-left text-sm font-medium">
                      Hostname
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium">
                      Site
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium">
                      Current Image
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium">
                      Golden Image
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium">
                      Age
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium">
                      Status
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {topOffenders.map((asset, i) => (
                    <tr
                      key={asset.id}
                      className={i !== topOffenders.length - 1 ? "border-b" : ""}
                    >
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <PlatformIcon platform={asset.platform} size="sm" />
                          <span className="font-medium">{asset.hostname}</span>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-sm text-muted-foreground">
                        {asset.siteName}
                      </td>
                      <td className="px-4 py-3">
                        <code className="rounded bg-muted px-2 py-1 text-xs">
                          {asset.currentImageVersion}
                        </code>
                      </td>
                      <td className="px-4 py-3">
                        <code className="rounded bg-muted px-2 py-1 text-xs">
                          {asset.goldenImageVersion}
                        </code>
                      </td>
                      <td className="px-4 py-3 text-sm text-muted-foreground">
                        {asset.driftDetectedAt ? formatDriftAge(asset.driftDetectedAt) : "N/A"}
                      </td>
                      <td className="px-4 py-3">
                        <StatusBadge
                          status={asset.isDrifted ? "critical" : "success"}
                          size="sm"
                        >
                          {asset.isDrifted ? "Drifted" : "Compliant"}
                        </StatusBadge>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="text-center text-sm text-muted-foreground py-8">
              No drifted assets found
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function formatDriftAge(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

  if (diffDays === 0) return "Today";
  if (diffDays === 1) return "1 day";
  if (diffDays < 7) return `${diffDays} days`;
  if (diffDays < 30) return `${Math.floor(diffDays / 7)} weeks`;
  return `${Math.floor(diffDays / 30)} months`;
}
