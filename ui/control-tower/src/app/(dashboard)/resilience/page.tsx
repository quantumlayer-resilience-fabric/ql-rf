"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { MetricCard } from "@/components/data/metric-card";
import { StatusBadge } from "@/components/status/status-badge";
import { PlatformIcon } from "@/components/status/platform-icon";
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
} from "lucide-react";

// Mock DR data
const drMetrics = {
  readiness: 98.1,
  rto: "4 min",
  rpo: "< 1 min",
  lastDrill: "3 days ago",
  drPairs: 4,
  protectedAssets: 8542,
};

const drPairs = [
  {
    id: "eu-pair",
    primary: { name: "eu-west-1", platform: "aws" as const, assets: 1234, status: "healthy" },
    secondary: { name: "eu-west-2", platform: "aws" as const, assets: 1234, status: "healthy" },
    syncStatus: "in-sync",
    lastSync: "2 min ago",
    rto: "4 min",
    rpo: "30 sec",
    lastDrill: "3 days ago",
    drillResult: "passed",
  },
  {
    id: "us-pair",
    primary: { name: "us-east-1", platform: "aws" as const, assets: 2156, status: "warning" },
    secondary: { name: "us-west-2", platform: "aws" as const, assets: 2156, status: "healthy" },
    syncStatus: "syncing",
    lastSync: "syncing...",
    rto: "6 min",
    rpo: "1 min",
    lastDrill: "1 week ago",
    drillResult: "passed",
  },
  {
    id: "azure-pair",
    primary: { name: "azure-eastus", platform: "azure" as const, assets: 1432, status: "healthy" },
    secondary: { name: "azure-westeu", platform: "azure" as const, assets: 1432, status: "healthy" },
    syncStatus: "in-sync",
    lastSync: "1 min ago",
    rto: "5 min",
    rpo: "45 sec",
    lastDrill: "5 days ago",
    drillResult: "passed",
  },
  {
    id: "dc-pair",
    primary: { name: "dc-london", platform: "vsphere" as const, assets: 432, status: "healthy" },
    secondary: { name: "dc-singapore", platform: "vsphere" as const, assets: 432, status: "warning" },
    syncStatus: "lag",
    lastSync: "15 min ago",
    rto: "12 min",
    rpo: "5 min",
    lastDrill: "2 weeks ago",
    drillResult: "warning",
  },
];

const drillHistory = [
  {
    id: "drill-1",
    date: "2024-01-12",
    pair: "eu-west-1 → eu-west-2",
    type: "Full Failover",
    rtoAchieved: "4 min 12 sec",
    rpoAchieved: "28 sec",
    result: "passed",
    notes: "Clean failover, all services recovered within RTO",
  },
  {
    id: "drill-2",
    date: "2024-01-10",
    pair: "azure-eastus → azure-westeu",
    type: "Partial Failover",
    rtoAchieved: "5 min 45 sec",
    rpoAchieved: "42 sec",
    result: "passed",
    notes: "Database replication lag noted but within acceptable range",
  },
  {
    id: "drill-3",
    date: "2024-01-08",
    pair: "us-east-1 → us-west-2",
    type: "Full Failover",
    rtoAchieved: "6 min 03 sec",
    rtoTarget: "5 min",
    rpoAchieved: "58 sec",
    result: "passed",
    notes: "RTO slightly exceeded, optimizations recommended",
  },
  {
    id: "drill-4",
    date: "2024-01-01",
    pair: "dc-london → dc-singapore",
    type: "Connectivity Test",
    rtoAchieved: "12 min 30 sec",
    rtoTarget: "10 min",
    rpoAchieved: "4 min 52 sec",
    result: "warning",
    notes: "Network latency issues detected, investigating WAN link",
  },
];

const unprotectedAssets = [
  { hostname: "dev-server-01", site: "us-east-1", reason: "Non-critical workload", lastUpdated: "2 days ago" },
  { hostname: "test-db-03", site: "azure-eastus", reason: "Test environment", lastUpdated: "1 week ago" },
  { hostname: "batch-worker-12", site: "gcp-us-central", reason: "No DR site available", lastUpdated: "3 days ago" },
];

export default function ResiliencePage() {
  const [selectedTab, setSelectedTab] = useState("overview");

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
          <Button size="sm">
            <Play className="mr-2 h-4 w-4" />
            Run DR Drill
          </Button>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
        <MetricCard
          title="DR Readiness"
          value={`${drMetrics.readiness}%`}
          subtitle="protected"
          status="success"
          icon={<Shield className="h-5 w-5" />}
        />
        <MetricCard
          title="Avg RTO"
          value={drMetrics.rto}
          subtitle="recovery time"
          status="success"
          icon={<Clock className="h-5 w-5" />}
        />
        <MetricCard
          title="Avg RPO"
          value={drMetrics.rpo}
          subtitle="data loss"
          status="success"
          icon={<Activity className="h-5 w-5" />}
        />
        <MetricCard
          title="DR Pairs"
          value={drMetrics.drPairs}
          subtitle="configured"
          status="neutral"
          icon={<Link2 className="h-5 w-5" />}
        />
        <MetricCard
          title="Protected"
          value={drMetrics.protectedAssets.toLocaleString()}
          subtitle="assets"
          status="success"
          icon={<Server className="h-5 w-5" />}
        />
      </div>

      {/* Tabs */}
      <Tabs defaultValue="pairs" className="space-y-4">
        <TabsList>
          <TabsTrigger value="pairs">DR Pairs</TabsTrigger>
          <TabsTrigger value="drills">Drill History</TabsTrigger>
          <TabsTrigger value="unprotected">Unprotected Assets</TabsTrigger>
        </TabsList>

        {/* DR Pairs Tab */}
        <TabsContent value="pairs" className="space-y-4">
          {drPairs.map((pair) => (
            <Card key={pair.id}>
              <CardContent className="p-6">
                <div className="flex items-center gap-6">
                  {/* Primary Site */}
                  <div className="flex-1 rounded-lg border p-4">
                    <div className="flex items-center gap-3">
                      <PlatformIcon platform={pair.primary.platform} size="md" />
                      <div>
                        <div className="flex items-center gap-2">
                          <h4 className="font-semibold">{pair.primary.name}</h4>
                          <Badge variant="secondary">Primary</Badge>
                        </div>
                        <p className="text-sm text-muted-foreground">
                          {pair.primary.assets.toLocaleString()} assets
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
                          pair.syncStatus === "syncing"
                            ? "animate-spin text-brand-accent"
                            : pair.syncStatus === "in-sync"
                            ? "text-status-green"
                            : "text-status-amber"
                        }`}
                      />
                      <ArrowRight className="h-5 w-5 rotate-180 text-muted-foreground" />
                    </div>
                    <StatusBadge
                      status={
                        pair.syncStatus === "in-sync"
                          ? "success"
                          : pair.syncStatus === "syncing"
                          ? "info"
                          : "warning"
                      }
                      size="sm"
                    >
                      {pair.syncStatus}
                    </StatusBadge>
                    <span className="text-xs text-muted-foreground">
                      {pair.lastSync}
                    </span>
                  </div>

                  {/* Secondary Site */}
                  <div className="flex-1 rounded-lg border border-purple-500/30 bg-purple-500/5 p-4">
                    <div className="flex items-center gap-3">
                      <PlatformIcon platform={pair.secondary.platform} size="md" />
                      <div>
                        <div className="flex items-center gap-2">
                          <h4 className="font-semibold">{pair.secondary.name}</h4>
                          <Badge variant="outline" className="text-purple-500 border-purple-500/30">
                            DR
                          </Badge>
                        </div>
                        <p className="text-sm text-muted-foreground">
                          {pair.secondary.assets.toLocaleString()} assets
                        </p>
                      </div>
                    </div>
                  </div>

                  {/* Metrics */}
                  <div className="flex gap-6 border-l pl-6">
                    <div className="text-center">
                      <div className="text-lg font-bold">{pair.rto}</div>
                      <div className="text-xs text-muted-foreground">RTO</div>
                    </div>
                    <div className="text-center">
                      <div className="text-lg font-bold">{pair.rpo}</div>
                      <div className="text-xs text-muted-foreground">RPO</div>
                    </div>
                    <div className="text-center">
                      <StatusBadge
                        status={pair.drillResult === "passed" ? "success" : "warning"}
                        size="sm"
                      >
                        {pair.drillResult}
                      </StatusBadge>
                      <div className="text-xs text-muted-foreground mt-1">
                        {pair.lastDrill}
                      </div>
                    </div>
                  </div>

                  {/* Actions */}
                  <Button variant="outline" size="sm">
                    <Play className="mr-2 h-4 w-4" />
                    Test Failover
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </TabsContent>

        {/* Drill History Tab */}
        <TabsContent value="drills" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Recent DR Drills</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {drillHistory.map((drill) => (
                  <div
                    key={drill.id}
                    className="flex items-start gap-4 rounded-lg border p-4"
                  >
                    <div
                      className={`rounded-full p-2 ${
                        drill.result === "passed"
                          ? "bg-status-green/10"
                          : "bg-status-amber/10"
                      }`}
                    >
                      {drill.result === "passed" ? (
                        <CheckCircle className="h-5 w-5 text-status-green" />
                      ) : (
                        <AlertTriangle className="h-5 w-5 text-status-amber" />
                      )}
                    </div>
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <h4 className="font-medium">{drill.pair}</h4>
                        <Badge variant="outline">{drill.type}</Badge>
                        <StatusBadge
                          status={drill.result === "passed" ? "success" : "warning"}
                          size="sm"
                        >
                          {drill.result}
                        </StatusBadge>
                      </div>
                      <p className="mt-1 text-sm text-muted-foreground">
                        {drill.notes}
                      </p>
                      <div className="mt-2 flex items-center gap-6 text-sm">
                        <span className="flex items-center gap-1">
                          <Target className="h-3 w-3 text-muted-foreground" />
                          RTO: {drill.rtoAchieved}
                        </span>
                        <span className="flex items-center gap-1">
                          <Zap className="h-3 w-3 text-muted-foreground" />
                          RPO: {drill.rpoAchieved}
                        </span>
                        <span className="flex items-center gap-1 text-muted-foreground">
                          <Calendar className="h-3 w-3" />
                          {drill.date}
                        </span>
                      </div>
                    </div>
                    <Button variant="ghost" size="sm">
                      View Details
                    </Button>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Unprotected Assets Tab */}
        <TabsContent value="unprotected" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-base">
                  Unprotected Assets ({unprotectedAssets.length})
                </CardTitle>
                <Button variant="outline" size="sm">
                  Configure DR
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <div className="rounded-lg border">
                <table className="w-full">
                  <thead>
                    <tr className="border-b bg-muted/50">
                      <th className="px-4 py-3 text-left text-sm font-medium">Asset</th>
                      <th className="px-4 py-3 text-left text-sm font-medium">Site</th>
                      <th className="px-4 py-3 text-left text-sm font-medium">Reason</th>
                      <th className="px-4 py-3 text-left text-sm font-medium">Last Updated</th>
                      <th className="px-4 py-3 text-right text-sm font-medium">Action</th>
                    </tr>
                  </thead>
                  <tbody>
                    {unprotectedAssets.map((asset, i) => (
                      <tr
                        key={asset.hostname}
                        className={i !== unprotectedAssets.length - 1 ? "border-b" : ""}
                      >
                        <td className="px-4 py-3 font-medium">{asset.hostname}</td>
                        <td className="px-4 py-3 text-sm text-muted-foreground">{asset.site}</td>
                        <td className="px-4 py-3 text-sm">{asset.reason}</td>
                        <td className="px-4 py-3 text-sm text-muted-foreground">{asset.lastUpdated}</td>
                        <td className="px-4 py-3 text-right">
                          <Button variant="outline" size="sm">
                            Add to DR
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
