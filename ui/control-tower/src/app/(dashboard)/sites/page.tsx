"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { MetricCard } from "@/components/data/metric-card";
import { SiteCard } from "@/components/data/site-card";
import { StatusBadge } from "@/components/status/status-badge";
import { PlatformIcon } from "@/components/status/platform-icon";
import { PageSkeleton, ErrorState, EmptyState } from "@/components/feedback";
import { useSites } from "@/hooks/use-sites";
import { Site } from "@/lib/api";
import {
  Search,
  Plus,
  RefreshCw,
  Map,
  LayoutGrid,
  Globe,
  Server,
  Shield,
  Link2,
  ArrowRight,
  Loader2,
} from "lucide-react";

export default function SitesPage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [platformFilter, setPlatformFilter] = useState<string>("all");
  const [envFilter, setEnvFilter] = useState<string>("all");
  const [viewMode, setViewMode] = useState<"grid" | "topology">("grid");

  const { data: sites, isLoading, error, refetch } = useSites();

  if (isLoading) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Sites
            </h1>
            <p className="text-muted-foreground">
              Manage your infrastructure sites and disaster recovery topology.
            </p>
          </div>
        </div>
        <PageSkeleton metricCards={4} showChart={false} showTable={false} />
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <div key={i} className="h-48 rounded-lg border bg-muted/20 animate-pulse" />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="page-transition space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Sites
          </h1>
          <p className="text-muted-foreground">
            Manage your infrastructure sites and disaster recovery topology.
          </p>
        </div>
        <ErrorState
          error={error}
          retry={refetch}
          title="Failed to load sites"
          description="We couldn't fetch the sites data. Please try again."
        />
      </div>
    );
  }

  const siteList = sites || [];

  // Calculate metrics from real data
  const siteMetrics = {
    totalSites: siteList.length,
    totalAssets: siteList.reduce((acc, s) => acc + s.assetCount, 0),
    avgCoverage: siteList.length > 0
      ? siteList.reduce((acc, s) => acc + s.coveragePercentage, 0) / siteList.length
      : 0,
    drPairs: siteList.filter((s) => s.drPaired).length / 2,
  };

  const filteredSites = siteList.filter((site) => {
    const matchesSearch =
      site.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      site.region.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesPlatform =
      platformFilter === "all" || site.platform === platformFilter;
    const matchesEnv = envFilter === "all" || site.environment === envFilter;
    return matchesSearch && matchesPlatform && matchesEnv;
  });

  // Build DR pairs from sites with drPaired field
  const drPairs: Array<{ primary: Site; secondary: Site }> = [];
  const processedIds = new Set<string>();

  siteList.forEach((site) => {
    if (site.drPaired && !processedIds.has(site.id)) {
      const paired = siteList.find((s) => s.id === site.drPaired);
      if (paired) {
        // Determine which is primary (not DR environment) and which is secondary
        const [primary, secondary] = site.environment !== "dr"
          ? [site, paired]
          : [paired, site];
        drPairs.push({ primary, secondary });
        processedIds.add(site.id);
        processedIds.add(paired.id);
      }
    }
  });

  const standaloneSites = siteList.filter((s) => !s.drPaired);

  const formatLastSync = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / (1000 * 60));

    if (diffMins < 1) return "just now";
    if (diffMins < 60) return `${diffMins} min ago`;
    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h ago`;
    return date.toLocaleDateString();
  };

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Sites
          </h1>
          <p className="text-muted-foreground">
            Manage your infrastructure sites and disaster recovery topology.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Sync All
          </Button>
          <Button size="sm">
            <Plus className="mr-2 h-4 w-4" />
            Add Site
          </Button>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Total Sites"
          value={siteMetrics.totalSites}
          subtitle="connected"
          status="neutral"
          icon={<Globe className="h-5 w-5" />}
        />
        <MetricCard
          title="Total Assets"
          value={siteMetrics.totalAssets.toLocaleString()}
          subtitle="across all sites"
          status="success"
          icon={<Server className="h-5 w-5" />}
        />
        <MetricCard
          title="Avg Coverage"
          value={`${siteMetrics.avgCoverage.toFixed(1)}%`}
          subtitle="compliant"
          status={siteMetrics.avgCoverage >= 90 ? "success" : "warning"}
          icon={<Shield className="h-5 w-5" />}
        />
        <MetricCard
          title="DR Pairs"
          value={Math.floor(siteMetrics.drPairs)}
          subtitle="configured"
          status="success"
          icon={<Link2 className="h-5 w-5" />}
        />
      </div>

      {/* Filters and View Toggle */}
      <Card>
        <CardContent className="flex items-center gap-4 p-4">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search sites..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9"
            />
          </div>
          <Select value={platformFilter} onValueChange={setPlatformFilter}>
            <SelectTrigger className="w-[150px]">
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
          <Select value={envFilter} onValueChange={setEnvFilter}>
            <SelectTrigger className="w-[150px]">
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
          <div className="flex items-center rounded-lg border p-1">
            <Button
              variant={viewMode === "grid" ? "secondary" : "ghost"}
              size="sm"
              onClick={() => setViewMode("grid")}
            >
              <LayoutGrid className="h-4 w-4" />
            </Button>
            <Button
              variant={viewMode === "topology" ? "secondary" : "ghost"}
              size="sm"
              onClick={() => setViewMode("topology")}
            >
              <Map className="h-4 w-4" />
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Content based on view mode */}
      {viewMode === "grid" ? (
        filteredSites.length > 0 ? (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {filteredSites.map((site) => (
              <SiteCard
                key={site.id}
                name={site.name}
                region={site.region}
                platform={site.platform}
                environment={site.environment}
                assetCount={site.assetCount}
                compliantCount={site.compliantCount}
                coveragePercentage={site.coveragePercentage}
                status={site.status}
                lastSyncAt={formatLastSync(site.lastSyncAt)}
                drPaired={site.drPaired}
                onClick={() => console.log("Navigate to site:", site.id)}
              />
            ))}
          </div>
        ) : (
          <Card>
            <CardContent className="p-8">
              <EmptyState
                variant="search"
                title="No sites found"
                description={searchQuery || platformFilter !== "all" || envFilter !== "all"
                  ? "Try adjusting your search or filter criteria"
                  : "Get started by adding your first site"}
                action={searchQuery || platformFilter !== "all" || envFilter !== "all" ? undefined : {
                  label: "Add Site",
                  onClick: () => console.log("Add site"),
                }}
              />
            </CardContent>
          </Card>
        )
      ) : (
        /* Topology View */
        <Card>
          <CardHeader>
            <CardTitle className="text-base">DR Topology</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-6">
              {drPairs.length > 0 ? (
                drPairs.map((pair) => (
                  <div
                    key={pair.primary.id}
                    className="flex items-center gap-4 rounded-lg border p-4"
                  >
                    {/* Primary Site */}
                    <div className="flex-1 rounded-lg border bg-background p-4">
                      <div className="flex items-center gap-3">
                        <PlatformIcon platform={pair.primary.platform} size="md" />
                        <div>
                          <div className="flex items-center gap-2">
                            <h4 className="font-semibold">{pair.primary.name}</h4>
                            <StatusBadge
                              status={
                                pair.primary.status === "healthy"
                                  ? "success"
                                  : pair.primary.status === "warning"
                                  ? "warning"
                                  : "critical"
                              }
                              size="sm"
                            >
                              Primary
                            </StatusBadge>
                          </div>
                          <p className="text-sm text-muted-foreground">
                            {pair.primary.region}
                          </p>
                        </div>
                      </div>
                      <div className="mt-3 flex items-center gap-4 text-sm">
                        <span>
                          <strong>{pair.primary.assetCount}</strong> assets
                        </span>
                        <span
                          className={
                            pair.primary.coveragePercentage >= 90
                              ? "text-status-green"
                              : "text-status-amber"
                          }
                        >
                          <strong>{pair.primary.coveragePercentage.toFixed(1)}%</strong> coverage
                        </span>
                      </div>
                    </div>

                    {/* Connection Arrow */}
                    <div className="flex flex-col items-center gap-1">
                      <div className="flex items-center gap-2 text-muted-foreground">
                        <ArrowRight className="h-5 w-5" />
                        <Link2 className="h-4 w-4 text-purple-500" />
                        <ArrowRight className="h-5 w-5 rotate-180" />
                      </div>
                      <span className="text-xs text-muted-foreground">
                        DR Pair
                      </span>
                    </div>

                    {/* Secondary Site */}
                    <div className="flex-1 rounded-lg border border-purple-500/30 bg-purple-500/5 p-4">
                      <div className="flex items-center gap-3">
                        <PlatformIcon platform={pair.secondary.platform} size="md" />
                        <div>
                          <div className="flex items-center gap-2">
                            <h4 className="font-semibold">{pair.secondary.name}</h4>
                            <StatusBadge status="info" size="sm">
                              DR
                            </StatusBadge>
                          </div>
                          <p className="text-sm text-muted-foreground">
                            {pair.secondary.region}
                          </p>
                        </div>
                      </div>
                      <div className="mt-3 flex items-center gap-4 text-sm">
                        <span>
                          <strong>{pair.secondary.assetCount}</strong> assets
                        </span>
                        <span
                          className={
                            pair.secondary.coveragePercentage >= 90
                              ? "text-status-green"
                              : "text-status-amber"
                          }
                        >
                          <strong>{pair.secondary.coveragePercentage.toFixed(1)}%</strong>{" "}
                          coverage
                        </span>
                      </div>
                    </div>
                  </div>
                ))
              ) : (
                <div className="text-center text-sm text-muted-foreground py-8">
                  No DR pairs configured
                </div>
              )}

              {/* Standalone Sites */}
              {standaloneSites.length > 0 && (
                <div className="mt-6">
                  <h4 className="mb-3 text-sm font-medium text-muted-foreground">
                    Standalone Sites (No DR Pair)
                  </h4>
                  <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
                    {standaloneSites.map((site) => (
                      <div
                        key={site.id}
                        className="flex items-center gap-3 rounded-lg border p-3"
                      >
                        <PlatformIcon platform={site.platform} size="sm" />
                        <div className="flex-1">
                          <div className="font-medium">{site.name}</div>
                          <div className="text-xs text-muted-foreground">
                            {site.region}
                          </div>
                        </div>
                        <span
                          className={`text-sm font-medium ${
                            site.coveragePercentage >= 90
                              ? "text-status-green"
                              : site.coveragePercentage >= 70
                              ? "text-status-amber"
                              : "text-status-red"
                          }`}
                        >
                          {site.coveragePercentage.toFixed(1)}%
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
