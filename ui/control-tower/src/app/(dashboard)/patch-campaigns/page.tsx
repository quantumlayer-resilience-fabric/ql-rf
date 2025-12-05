"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Progress } from "@/components/ui/progress";
import { MetricCard } from "@/components/data/metric-card";
import { PageSkeleton, ErrorState, EmptyState } from "@/components/feedback";
import {
  usePatchCampaigns,
  usePatchCampaignSummary,
  type PatchCampaign,
  type PatchCampaignStatus,
  type RolloutStrategy,
} from "@/hooks/use-patch-campaigns";
import {
  Rocket,
  Play,
  Pause,
  CheckCircle,
  XCircle,
  Clock,
  RefreshCw,
  Plus,
  ChevronRight,
  Zap,
  Target,
  Shield,
  RotateCcw,
  Loader2,
} from "lucide-react";

// Status badge configuration
const statusConfig: Record<PatchCampaignStatus, { label: string; className: string; icon: React.ReactNode }> = {
  draft: { label: "Draft", className: "bg-gray-500/10 text-gray-500 border-gray-500/50", icon: null },
  pending_approval: { label: "Pending Approval", className: "bg-purple-500/10 text-purple-500 border-purple-500/50", icon: <Clock className="h-3 w-3" /> },
  approved: { label: "Approved", className: "bg-blue-500/10 text-blue-500 border-blue-500/50", icon: <CheckCircle className="h-3 w-3" /> },
  scheduled: { label: "Scheduled", className: "bg-cyan-500/10 text-cyan-500 border-cyan-500/50", icon: <Clock className="h-3 w-3" /> },
  in_progress: { label: "In Progress", className: "bg-amber-500/10 text-amber-500 border-amber-500/50", icon: <Loader2 className="h-3 w-3 animate-spin" /> },
  paused: { label: "Paused", className: "bg-yellow-500/10 text-yellow-500 border-yellow-500/50", icon: <Pause className="h-3 w-3" /> },
  completed: { label: "Completed", className: "bg-green-500/10 text-green-500 border-green-500/50", icon: <CheckCircle className="h-3 w-3" /> },
  failed: { label: "Failed", className: "bg-red-500/10 text-red-500 border-red-500/50", icon: <XCircle className="h-3 w-3" /> },
  rolled_back: { label: "Rolled Back", className: "bg-orange-500/10 text-orange-500 border-orange-500/50", icon: <RotateCcw className="h-3 w-3" /> },
  cancelled: { label: "Cancelled", className: "bg-gray-500/10 text-gray-500 border-gray-500/50", icon: <XCircle className="h-3 w-3" /> },
};

// Rollout strategy configuration
const strategyConfig: Record<RolloutStrategy, { label: string; description: string }> = {
  immediate: { label: "Immediate", description: "All at once" },
  canary: { label: "Canary", description: "Small group first" },
  rolling: { label: "Rolling", description: "Progressive waves" },
  blue_green: { label: "Blue/Green", description: "50/50 deployment" },
};

export default function PatchCampaignsPage() {
  const router = useRouter();
  const [statusFilter, setStatusFilter] = useState<PatchCampaignStatus | "all">("all");
  const [typeFilter, setTypeFilter] = useState<string>("all");

  // Fetch data
  const {
    data: campaignsData,
    isLoading: campaignsLoading,
    isPending: campaignsPending,
    error: campaignsError,
    refetch: refetchCampaigns,
  } = usePatchCampaigns({
    status: statusFilter === "all" ? undefined : statusFilter,
    campaign_type: typeFilter === "all" ? undefined : typeFilter,
    page_size: 100,
  });

  const {
    data: summary,
    isLoading: summaryLoading,
    isPending: summaryPending,
    error: summaryError,
  } = usePatchCampaignSummary();

  // Loading state
  if (campaignsLoading || campaignsPending || summaryLoading || summaryPending) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Patch Campaigns
            </h1>
            <p className="text-muted-foreground">
              Orchestrate and monitor patch deployments across your infrastructure.
            </p>
          </div>
        </div>
        <PageSkeleton metricCards={4} showChart={false} showTable={true} tableRows={5} />
      </div>
    );
  }

  // Error state
  if (campaignsError || summaryError) {
    return (
      <div className="page-transition space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Patch Campaigns
          </h1>
          <p className="text-muted-foreground">
            Orchestrate and monitor patch deployments across your infrastructure.
          </p>
        </div>
        <ErrorState
          error={campaignsError || summaryError || new Error("Unknown error")}
          retry={refetchCampaigns}
          title="Failed to load patch campaigns"
          description="We couldn't fetch the patch campaign data. Please try again."
        />
      </div>
    );
  }

  const campaigns = campaignsData?.campaigns || [];
  const activeCampaigns = campaigns.filter((c) => ["in_progress", "paused"].includes(c.status));
  const pendingCampaigns = campaigns.filter((c) => ["pending_approval", "approved", "scheduled"].includes(c.status));

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Patch Campaigns
          </h1>
          <p className="text-muted-foreground">
            Orchestrate and monitor patch deployments across your infrastructure.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => refetchCampaigns()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
          <Button size="sm" onClick={() => router.push("/vulnerabilities")}>
            <Plus className="mr-2 h-4 w-4" />
            New Campaign
          </Button>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Total Campaigns"
          value={summary?.total_campaigns?.toString() || "0"}
          subtitle={`${summary?.active_campaigns || 0} active`}
          status="neutral"
          icon={<Rocket className="h-5 w-5" />}
        />
        <MetricCard
          title="Completed"
          value={summary?.completed_campaigns?.toString() || "0"}
          subtitle={`${summary?.success_rate?.toFixed(1) || 0}% success rate`}
          status="success"
          icon={<CheckCircle className="h-5 w-5" />}
        />
        <MetricCard
          title="Assets Patched"
          value={summary?.total_assets_patched?.toString() || "0"}
          subtitle="across all campaigns"
          status="neutral"
          icon={<Target className="h-5 w-5" />}
        />
        <MetricCard
          title="Rollbacks"
          value={summary?.total_rollbacks?.toString() || "0"}
          subtitle={summary?.failed_campaigns ? `${summary.failed_campaigns} failed` : "0 failed"}
          status={summary?.total_rollbacks && summary.total_rollbacks > 0 ? "warning" : "success"}
          icon={<RotateCcw className="h-5 w-5" />}
        />
      </div>

      {/* Active Campaigns Alert */}
      {activeCampaigns.length > 0 && (
        <Card className="border-l-4 border-l-status-amber bg-gradient-to-r from-status-amber/5 to-transparent">
          <CardContent className="flex items-center gap-4 p-4">
            <div className="rounded-lg bg-status-amber/10 p-2">
              <Zap className="h-5 w-5 text-status-amber" />
            </div>
            <div className="flex-1">
              <span className="font-medium">
                {activeCampaigns.length} Active Campaign{activeCampaigns.length > 1 ? "s" : ""}
              </span>
              <span className="text-sm text-muted-foreground ml-2">
                - Click to monitor progress
              </span>
            </div>
            <Button
              size="sm"
              variant="outline"
              onClick={() => router.push(`/patch-campaigns/${activeCampaigns[0].id}`)}
            >
              View Active
              <ChevronRight className="ml-2 h-4 w-4" />
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Pending Approvals */}
      {pendingCampaigns.filter((c) => c.status === "pending_approval").length > 0 && (
        <Card className="border-l-4 border-l-purple-500 bg-gradient-to-r from-purple-500/5 to-transparent">
          <CardContent className="flex items-center gap-4 p-4">
            <div className="rounded-lg bg-purple-500/10 p-2">
              <Shield className="h-5 w-5 text-purple-500" />
            </div>
            <div className="flex-1">
              <span className="font-medium">
                {pendingCampaigns.filter((c) => c.status === "pending_approval").length} Campaign{pendingCampaigns.filter((c) => c.status === "pending_approval").length > 1 ? "s" : ""} Awaiting Approval
              </span>
              <span className="text-sm text-muted-foreground ml-2">
                - Human approval required before deployment
              </span>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Filters */}
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium">Status:</span>
              <Select
                value={statusFilter}
                onValueChange={(value) => setStatusFilter(value as PatchCampaignStatus | "all")}
              >
                <SelectTrigger className="w-[180px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Statuses</SelectItem>
                  <SelectItem value="draft">Draft</SelectItem>
                  <SelectItem value="pending_approval">Pending Approval</SelectItem>
                  <SelectItem value="approved">Approved</SelectItem>
                  <SelectItem value="scheduled">Scheduled</SelectItem>
                  <SelectItem value="in_progress">In Progress</SelectItem>
                  <SelectItem value="paused">Paused</SelectItem>
                  <SelectItem value="completed">Completed</SelectItem>
                  <SelectItem value="failed">Failed</SelectItem>
                  <SelectItem value="rolled_back">Rolled Back</SelectItem>
                  <SelectItem value="cancelled">Cancelled</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium">Type:</span>
              <Select
                value={typeFilter}
                onValueChange={setTypeFilter}
              >
                <SelectTrigger className="w-[180px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Types</SelectItem>
                  <SelectItem value="cve_response">CVE Response</SelectItem>
                  <SelectItem value="scheduled">Scheduled</SelectItem>
                  <SelectItem value="emergency">Emergency</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Tabs */}
      <Tabs defaultValue="all" className="space-y-4">
        <TabsList>
          <TabsTrigger value="all">All Campaigns ({campaigns.length})</TabsTrigger>
          <TabsTrigger value="active">
            Active ({activeCampaigns.length})
          </TabsTrigger>
          <TabsTrigger value="pending">
            Pending ({pendingCampaigns.length})
          </TabsTrigger>
        </TabsList>

        {/* All Campaigns Tab */}
        <TabsContent value="all" className="space-y-4">
          <CampaignsTable campaigns={campaigns} />
        </TabsContent>

        {/* Active Tab */}
        <TabsContent value="active" className="space-y-4">
          <CampaignsTable campaigns={activeCampaigns} />
        </TabsContent>

        {/* Pending Tab */}
        <TabsContent value="pending" className="space-y-4">
          <CampaignsTable campaigns={pendingCampaigns} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

// Campaigns table component
function CampaignsTable({ campaigns }: { campaigns: PatchCampaign[] }) {
  const router = useRouter();

  if (campaigns.length === 0) {
    return (
      <Card>
        <CardContent className="pt-6">
          <EmptyState
            variant="data"
            title="No patch campaigns"
            description="No patch campaigns match your current filters. Create a new campaign from a CVE alert."
          />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Patch Campaigns</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Campaign</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Strategy</TableHead>
              <TableHead>Progress</TableHead>
              <TableHead>Assets</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {campaigns.map((campaign) => (
              <CampaignRow
                key={campaign.id}
                campaign={campaign}
                onNavigate={() => router.push(`/patch-campaigns/${campaign.id}`)}
              />
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

// Campaign row component
function CampaignRow({
  campaign,
  onNavigate,
}: {
  campaign: PatchCampaign;
  onNavigate: () => void;
}) {
  const statusCfg = statusConfig[campaign.status];
  const strategyCfg = strategyConfig[campaign.rollout_strategy];

  const completionPct = campaign.total_assets > 0
    ? Math.round((campaign.completed_assets / campaign.total_assets) * 100)
    : 0;

  const failurePct = campaign.total_assets > 0
    ? Math.round((campaign.failed_assets / campaign.total_assets) * 100)
    : 0;

  return (
    <TableRow className="cursor-pointer hover:bg-muted/50" onClick={onNavigate}>
      <TableCell>
        <div className="flex items-center gap-2">
          <Rocket className="h-4 w-4 text-muted-foreground" />
          <div>
            <div className="font-medium">{campaign.name}</div>
            <div className="text-xs text-muted-foreground">
              {campaign.campaign_type === "cve_response" ? "CVE Response" : campaign.campaign_type}
              {campaign.cve_alert_ids && campaign.cve_alert_ids.length > 0 && (
                <span className="ml-2">• {campaign.cve_alert_ids.length} CVE{campaign.cve_alert_ids.length > 1 ? "s" : ""}</span>
              )}
            </div>
          </div>
        </div>
      </TableCell>
      <TableCell>
        <Badge variant="outline" className={`flex items-center gap-1 w-fit ${statusCfg.className}`}>
          {statusCfg.icon}
          {statusCfg.label}
        </Badge>
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-2">
          <Badge variant="outline">{strategyCfg.label}</Badge>
          {campaign.canary_percentage && (
            <span className="text-xs text-muted-foreground">
              {campaign.canary_percentage}% canary
            </span>
          )}
        </div>
      </TableCell>
      <TableCell>
        <div className="w-32">
          <div className="flex items-center justify-between text-xs mb-1">
            <span>{completionPct}%</span>
            {failurePct > 0 && (
              <span className="text-status-red">{failurePct}% failed</span>
            )}
          </div>
          <Progress value={completionPct} className="h-2" />
        </div>
      </TableCell>
      <TableCell>
        <div className="flex flex-col text-sm">
          <span>{campaign.completed_assets}/{campaign.total_assets} complete</span>
          {campaign.failed_assets > 0 && (
            <span className="text-status-red text-xs">{campaign.failed_assets} failed</span>
          )}
        </div>
      </TableCell>
      <TableCell className="text-right">
        <div className="flex items-center justify-end gap-2">
          {campaign.status === "pending_approval" && (
            <Badge variant="outline" className="border-purple-500 text-purple-500">
              Needs Approval
            </Badge>
          )}
          <Button variant="ghost" size="sm" onClick={onNavigate}>
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}
