"use client";

import { use, useState } from "react";
import { useRouter } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
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
import { Separator } from "@/components/ui/separator";
import { PageSkeleton, ErrorState, EmptyState } from "@/components/feedback";
import {
  usePatchCampaign,
  usePatchCampaignPhases,
  usePatchCampaignAssets,
  usePatchCampaignProgress,
  useApprovePatchCampaign,
  useRejectPatchCampaign,
  useStartPatchCampaign,
  usePausePatchCampaign,
  useResumePatchCampaign,
  useCancelPatchCampaign,
  useRollbackPatchCampaign,
  type PatchCampaign,
  type PatchCampaignPhase,
  type PatchCampaignAsset,
  type PatchCampaignProgress as ProgressType,
  type PatchCampaignStatus,
  type RolloutStrategy,
} from "@/hooks/use-patch-campaigns";
import {
  ArrowLeft,
  Rocket,
  Play,
  Pause,
  Square,
  CheckCircle,
  XCircle,
  Clock,
  RotateCcw,
  Loader2,
  Shield,
  Target,
  Server,
  AlertTriangle,
  ChevronRight,
  Zap,
  Timer,
  Activity,
} from "lucide-react";

// Status configuration
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

// Strategy configuration
const strategyConfig: Record<RolloutStrategy, { label: string; description: string }> = {
  immediate: { label: "Immediate", description: "All assets patched simultaneously" },
  canary: { label: "Canary", description: "Small percentage first, then full rollout" },
  rolling: { label: "Rolling", description: "Progressive waves with health checks" },
  blue_green: { label: "Blue/Green", description: "50/50 deployment with instant switch" },
};

export default function PatchCampaignDetailPage({
  params,
}: {
  params: Promise<{ campaignId: string }>;
}) {
  const { campaignId } = use(params);
  const router = useRouter();
  const [isActioning, setIsActioning] = useState(false);

  // Fetch data
  const {
    data: campaign,
    isLoading: campaignLoading,
    error: campaignError,
    refetch: refetchCampaign,
  } = usePatchCampaign(campaignId);

  const {
    data: phasesData,
    isLoading: phasesLoading,
  } = usePatchCampaignPhases(campaignId);

  const {
    data: assetsData,
    isLoading: assetsLoading,
  } = usePatchCampaignAssets(campaignId);

  const {
    data: progress,
    isLoading: progressLoading,
  } = usePatchCampaignProgress(campaignId);

  // Mutations
  const approveCampaign = useApprovePatchCampaign();
  const rejectCampaign = useRejectPatchCampaign();
  const startCampaign = useStartPatchCampaign();
  const pauseCampaign = usePausePatchCampaign();
  const resumeCampaign = useResumePatchCampaign();
  const cancelCampaign = useCancelPatchCampaign();
  const rollbackCampaign = useRollbackPatchCampaign();

  // Action handlers
  const handleApprove = async () => {
    setIsActioning(true);
    try {
      await approveCampaign.mutateAsync({
        campaignId,
        data: { approved_by: "current-user@example.com" },
      });
    } catch (error) {
      console.error("Failed to approve campaign:", error);
    } finally {
      setIsActioning(false);
    }
  };

  const handleReject = async () => {
    setIsActioning(true);
    try {
      await rejectCampaign.mutateAsync({
        campaignId,
        data: { rejected_by: "current-user@example.com", reason: "Requires additional review" },
      });
    } catch (error) {
      console.error("Failed to reject campaign:", error);
    } finally {
      setIsActioning(false);
    }
  };

  const handleStart = async () => {
    setIsActioning(true);
    try {
      await startCampaign.mutateAsync(campaignId);
    } catch (error) {
      console.error("Failed to start campaign:", error);
    } finally {
      setIsActioning(false);
    }
  };

  const handlePause = async () => {
    setIsActioning(true);
    try {
      await pauseCampaign.mutateAsync(campaignId);
    } catch (error) {
      console.error("Failed to pause campaign:", error);
    } finally {
      setIsActioning(false);
    }
  };

  const handleResume = async () => {
    setIsActioning(true);
    try {
      await resumeCampaign.mutateAsync(campaignId);
    } catch (error) {
      console.error("Failed to resume campaign:", error);
    } finally {
      setIsActioning(false);
    }
  };

  const handleCancel = async () => {
    setIsActioning(true);
    try {
      await cancelCampaign.mutateAsync(campaignId);
    } catch (error) {
      console.error("Failed to cancel campaign:", error);
    } finally {
      setIsActioning(false);
    }
  };

  const handleRollback = async () => {
    setIsActioning(true);
    try {
      await rollbackCampaign.mutateAsync({
        campaignId,
        data: { scope: "all", reason: "Manual rollback triggered" },
      });
    } catch (error) {
      console.error("Failed to rollback campaign:", error);
    } finally {
      setIsActioning(false);
    }
  };

  // Loading state
  if (campaignLoading) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => router.back()}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Patch Campaign
            </h1>
            <p className="text-muted-foreground">Loading campaign details...</p>
          </div>
        </div>
        <PageSkeleton metricCards={3} showChart={false} showTable={true} tableRows={5} />
      </div>
    );
  }

  // Error state
  if (campaignError) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => router.back()}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Patch Campaign
            </h1>
          </div>
        </div>
        <ErrorState
          error={campaignError}
          retry={refetchCampaign}
          title="Failed to load campaign"
          description="We couldn't fetch the campaign details. Please try again."
        />
      </div>
    );
  }

  if (!campaign) {
    return (
      <div className="page-transition space-y-6">
        <EmptyState
          variant="data"
          title="Campaign not found"
          description="The requested patch campaign could not be found."
        />
      </div>
    );
  }

  const statusCfg = statusConfig[campaign.status];
  const strategyCfg = strategyConfig[campaign.rollout_strategy];
  const phases = phasesData?.phases || [];
  const assets = assetsData?.assets || [];

  const completionPct = campaign.total_assets > 0
    ? Math.round((campaign.completed_assets / campaign.total_assets) * 100)
    : 0;

  return (
    <div className="page-transition space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => router.back()}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold tracking-tight text-foreground">
                {campaign.name}
              </h1>
              <Badge variant="outline" className={`flex items-center gap-1 ${statusCfg.className}`}>
                {statusCfg.icon}
                {statusCfg.label}
              </Badge>
            </div>
            <p className="text-muted-foreground mt-1">
              {campaign.description || `${campaign.campaign_type} campaign using ${strategyCfg.label.toLowerCase()} strategy`}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {/* Approval Actions */}
          {campaign.status === "pending_approval" && (
            <>
              <Button
                variant="outline"
                onClick={handleReject}
                disabled={isActioning}
              >
                {isActioning ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <XCircle className="mr-2 h-4 w-4" />}
                Reject
              </Button>
              <Button onClick={handleApprove} disabled={isActioning}>
                {isActioning ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <CheckCircle className="mr-2 h-4 w-4" />}
                Approve
              </Button>
            </>
          )}

          {/* Start Action */}
          {campaign.status === "approved" && (
            <Button onClick={handleStart} disabled={isActioning}>
              {isActioning ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Play className="mr-2 h-4 w-4" />}
              Start Campaign
            </Button>
          )}

          {/* In Progress Actions */}
          {campaign.status === "in_progress" && (
            <>
              <Button variant="outline" onClick={handlePause} disabled={isActioning}>
                {isActioning ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Pause className="mr-2 h-4 w-4" />}
                Pause
              </Button>
              <Button variant="outline" onClick={handleRollback} disabled={isActioning}>
                {isActioning ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <RotateCcw className="mr-2 h-4 w-4" />}
                Rollback
              </Button>
            </>
          )}

          {/* Paused Actions */}
          {campaign.status === "paused" && (
            <>
              <Button onClick={handleResume} disabled={isActioning}>
                {isActioning ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Play className="mr-2 h-4 w-4" />}
                Resume
              </Button>
              <Button variant="outline" onClick={handleCancel} disabled={isActioning}>
                {isActioning ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Square className="mr-2 h-4 w-4" />}
                Cancel
              </Button>
            </>
          )}
        </div>
      </div>

      {/* Progress Overview */}
      {progress && ["in_progress", "paused"].includes(campaign.status) && (
        <Card className="border-l-4 border-l-status-amber">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-5 w-5" />
              Live Progress
            </CardTitle>
            <CardDescription>
              Current phase: {progress.current_phase}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-6 md:grid-cols-4">
              <div>
                <div className="text-sm text-muted-foreground">Overall Progress</div>
                <div className="flex items-center gap-4 mt-1">
                  <div className="text-2xl font-bold">{progress.completion_percentage}%</div>
                  <Progress value={progress.completion_percentage} className="flex-1" />
                </div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">Phase Progress</div>
                <div className="flex items-center gap-4 mt-1">
                  <div className="text-2xl font-bold">{progress.current_phase_progress}%</div>
                  <Progress value={progress.current_phase_progress} className="flex-1" />
                </div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">Assets</div>
                <div className="text-2xl font-bold mt-1">
                  {progress.completed_assets}/{progress.total_assets}
                </div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">Time Elapsed</div>
                <div className="text-2xl font-bold mt-1">
                  {progress.elapsed_time_minutes || 0}m
                </div>
              </div>
            </div>

            {progress.failure_percentage > 0 && (
              <div className="flex items-center gap-2 text-status-red">
                <AlertTriangle className="h-4 w-4" />
                <span>{progress.failure_percentage}% failure rate ({progress.failed_assets} assets)</span>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Strategy
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Zap className="h-5 w-5 text-muted-foreground" />
              <div>
                <div className="font-semibold">{strategyCfg.label}</div>
                <div className="text-xs text-muted-foreground">{strategyCfg.description}</div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Total Assets
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Server className="h-5 w-5 text-muted-foreground" />
              <div className="text-2xl font-bold">{campaign.total_assets}</div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Completed
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-4">
              <div className="text-2xl font-bold text-status-green">{campaign.completed_assets}</div>
              <Progress value={completionPct} className="flex-1" />
              <span className="text-sm text-muted-foreground">{completionPct}%</span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Failed
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <div className={`text-2xl font-bold ${campaign.failed_assets > 0 ? "text-status-red" : "text-muted-foreground"}`}>
                {campaign.failed_assets}
              </div>
              {campaign.failure_threshold_percentage && (
                <span className="text-sm text-muted-foreground">
                  (threshold: {campaign.failure_threshold_percentage}%)
                </span>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Configuration Card */}
      <Card>
        <CardHeader>
          <CardTitle>Configuration</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-3">
            <div>
              <div className="text-sm text-muted-foreground">Health Checks</div>
              <div className="flex items-center gap-2 mt-1">
                {campaign.health_check_enabled ? (
                  <>
                    <CheckCircle className="h-4 w-4 text-status-green" />
                    <span>Enabled</span>
                  </>
                ) : (
                  <>
                    <XCircle className="h-4 w-4 text-muted-foreground" />
                    <span className="text-muted-foreground">Disabled</span>
                  </>
                )}
              </div>
            </div>
            <div>
              <div className="text-sm text-muted-foreground">Auto-Rollback</div>
              <div className="flex items-center gap-2 mt-1">
                {campaign.auto_rollback_enabled ? (
                  <>
                    <CheckCircle className="h-4 w-4 text-status-green" />
                    <span>Enabled</span>
                  </>
                ) : (
                  <>
                    <XCircle className="h-4 w-4 text-muted-foreground" />
                    <span className="text-muted-foreground">Disabled</span>
                  </>
                )}
              </div>
            </div>
            <div>
              <div className="text-sm text-muted-foreground">Requires Approval</div>
              <div className="flex items-center gap-2 mt-1">
                {campaign.requires_approval ? (
                  <>
                    <Shield className="h-4 w-4 text-purple-500" />
                    <span>Required</span>
                    {campaign.approved_by && (
                      <span className="text-sm text-muted-foreground">
                        (by {campaign.approved_by})
                      </span>
                    )}
                  </>
                ) : (
                  <>
                    <XCircle className="h-4 w-4 text-muted-foreground" />
                    <span className="text-muted-foreground">Not Required</span>
                  </>
                )}
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Tabs */}
      <Tabs defaultValue="phases" className="space-y-4">
        <TabsList>
          <TabsTrigger value="phases">Phases ({phases.length})</TabsTrigger>
          <TabsTrigger value="assets">Assets ({assets.length})</TabsTrigger>
        </TabsList>

        {/* Phases Tab */}
        <TabsContent value="phases" className="space-y-4">
          {phases.length === 0 ? (
            <Card>
              <CardContent className="pt-6">
                <EmptyState
                  variant="data"
                  title="No phases"
                  description="Phase information will be available once the campaign starts."
                />
              </CardContent>
            </Card>
          ) : (
            <Card>
              <CardHeader>
                <CardTitle>Deployment Phases</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {phases.map((phase, index) => (
                    <PhaseCard key={phase.id} phase={phase} index={index} />
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {/* Assets Tab */}
        <TabsContent value="assets" className="space-y-4">
          {assets.length === 0 ? (
            <Card>
              <CardContent className="pt-6">
                <EmptyState
                  variant="data"
                  title="No assets"
                  description="Asset information will be available once the campaign starts."
                />
              </CardContent>
            </Card>
          ) : (
            <Card>
              <CardHeader>
                <CardTitle>Affected Assets</CardTitle>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Asset</TableHead>
                      <TableHead>Platform</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Version</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {assets.map((asset) => (
                      <AssetRow key={asset.id} asset={asset} />
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}

// Phase Card Component
function PhaseCard({ phase, index }: { phase: PatchCampaignPhase; index: number }) {
  const isActive = phase.status === "in_progress";
  const isCompleted = phase.status === "completed";
  const isFailed = phase.status === "failed";

  const completionPct = phase.total_assets > 0
    ? Math.round((phase.completed_assets / phase.total_assets) * 100)
    : 0;

  return (
    <div
      className={`rounded-lg border p-4 ${
        isActive
          ? "border-status-amber bg-status-amber/5"
          : isCompleted
          ? "border-status-green bg-status-green/5"
          : isFailed
          ? "border-status-red bg-status-red/5"
          : "border-border"
      }`}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div
            className={`flex h-8 w-8 items-center justify-center rounded-full text-sm font-bold ${
              isActive
                ? "bg-status-amber text-white"
                : isCompleted
                ? "bg-status-green text-white"
                : isFailed
                ? "bg-status-red text-white"
                : "bg-muted text-muted-foreground"
            }`}
          >
            {index + 1}
          </div>
          <div>
            <div className="font-medium">{phase.name}</div>
            <div className="text-sm text-muted-foreground">
              {phase.phase_type} • {phase.target_percentage}% of assets
            </div>
          </div>
        </div>
        <div className="flex items-center gap-4">
          <div className="text-right">
            <div className="text-sm">
              {phase.completed_assets}/{phase.total_assets} assets
            </div>
            {phase.failed_assets > 0 && (
              <div className="text-sm text-status-red">
                {phase.failed_assets} failed
              </div>
            )}
          </div>
          <div className="w-24">
            <Progress
              value={completionPct}
              className={
                isCompleted
                  ? "[&>div]:bg-status-green"
                  : isActive
                  ? "[&>div]:bg-status-amber"
                  : ""
              }
            />
          </div>
          {phase.health_check_passed !== undefined && (
            <div>
              {phase.health_check_passed ? (
                <Badge variant="outline" className="border-status-green text-status-green">
                  <CheckCircle className="mr-1 h-3 w-3" />
                  Healthy
                </Badge>
              ) : (
                <Badge variant="outline" className="border-status-red text-status-red">
                  <XCircle className="mr-1 h-3 w-3" />
                  Unhealthy
                </Badge>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// Asset Row Component
function AssetRow({ asset }: { asset: PatchCampaignAsset }) {
  const statusBadge = {
    completed: { className: "bg-status-green/10 text-status-green border-status-green/50", label: "Completed" },
    in_progress: { className: "bg-status-amber/10 text-status-amber border-status-amber/50", label: "In Progress" },
    failed: { className: "bg-status-red/10 text-status-red border-status-red/50", label: "Failed" },
    pending: { className: "bg-gray-500/10 text-gray-500 border-gray-500/50", label: "Pending" },
    skipped: { className: "bg-gray-500/10 text-gray-500 border-gray-500/50", label: "Skipped" },
  }[asset.status] || { className: "", label: asset.status };

  return (
    <TableRow>
      <TableCell>
        <div className="font-medium">{asset.asset_name}</div>
        {asset.error_message && (
          <div className="text-xs text-status-red mt-1">{asset.error_message}</div>
        )}
      </TableCell>
      <TableCell className="uppercase">{asset.platform}</TableCell>
      <TableCell>
        <Badge variant="outline" className={statusBadge.className}>
          {statusBadge.label}
        </Badge>
      </TableCell>
      <TableCell>
        {asset.before_version && asset.after_version ? (
          <div className="flex items-center gap-2 font-mono text-sm">
            <span className="text-muted-foreground">{asset.before_version}</span>
            <ChevronRight className="h-3 w-3" />
            <span className="text-status-green">{asset.after_version}</span>
          </div>
        ) : asset.before_version ? (
          <span className="font-mono text-sm text-muted-foreground">{asset.before_version}</span>
        ) : (
          <span className="text-muted-foreground">-</span>
        )}
      </TableCell>
    </TableRow>
  );
}
