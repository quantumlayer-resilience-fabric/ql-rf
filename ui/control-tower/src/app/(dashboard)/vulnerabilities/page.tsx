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
import { MetricCard } from "@/components/data/metric-card";
import { PageSkeleton, ErrorState, EmptyState } from "@/components/feedback";
import {
  useCVEAlerts,
  useCVEAlertSummary,
  useUpdateCVEAlertStatus,
  type CVEAlert,
  type CVESeverity,
  type CVEAlertStatus,
  type CVEAlertPriority,
} from "@/hooks/use-cve-alerts";
import { useSendAIMessage, useAIContext, usePendingTasks } from "@/hooks/use-ai";
import {
  ShieldAlert,
  Shield,
  AlertTriangle,
  Clock,
  Bug,
  Target,
  Server,
  Zap,
  Loader2,
  Sparkles,
  RefreshCw,
  ExternalLink,
  Search,
  ChevronRight,
  Cloud,
} from "lucide-react";

// Severity badge configuration
const severityConfig: Record<CVESeverity, { variant: "default" | "secondary" | "destructive" | "outline"; label: string; className: string }> = {
  critical: { variant: "destructive", label: "Critical", className: "bg-status-red text-white" },
  high: { variant: "default", label: "High", className: "bg-status-amber text-white" },
  medium: { variant: "secondary", label: "Medium", className: "bg-yellow-500 text-white" },
  low: { variant: "outline", label: "Low", className: "border-blue-500 text-blue-500" },
  unknown: { variant: "outline", label: "Unknown", className: "" },
};

// Status configuration
const statusConfig: Record<CVEAlertStatus, { label: string; className: string }> = {
  new: { label: "New", className: "bg-blue-500/10 text-blue-500 border-blue-500/50" },
  investigating: { label: "Investigating", className: "bg-purple-500/10 text-purple-500 border-purple-500/50" },
  confirmed: { label: "Confirmed", className: "bg-orange-500/10 text-orange-500 border-orange-500/50" },
  in_progress: { label: "In Progress", className: "bg-amber-500/10 text-amber-500 border-amber-500/50" },
  resolved: { label: "Resolved", className: "bg-green-500/10 text-green-500 border-green-500/50" },
  dismissed: { label: "Dismissed", className: "bg-gray-500/10 text-gray-500 border-gray-500/50" },
  auto_resolved: { label: "Auto-Resolved", className: "bg-green-500/10 text-green-500 border-green-500/50" },
};

// Priority configuration
const priorityConfig: Record<CVEAlertPriority, { label: string; className: string }> = {
  p1: { label: "P1", className: "bg-status-red text-white" },
  p2: { label: "P2", className: "bg-status-amber text-white" },
  p3: { label: "P3", className: "bg-yellow-500 text-white" },
  p4: { label: "P4", className: "bg-blue-500 text-white" },
};

export default function VulnerabilitiesPage() {
  const router = useRouter();
  const [severityFilter, setSeverityFilter] = useState<CVESeverity | "all">("all");
  const [statusFilter, setStatusFilter] = useState<CVEAlertStatus | "all">("all");
  const [isCreatingAITask, setIsCreatingAITask] = useState(false);

  // Fetch data
  const {
    data: alertsData,
    isLoading: alertsLoading,
    isPending: alertsPending,
    error: alertsError,
    refetch: refetchAlerts,
  } = useCVEAlerts({
    severity: severityFilter === "all" ? undefined : severityFilter,
    status: statusFilter === "all" ? undefined : statusFilter,
    page_size: 100,
  });

  const {
    data: summary,
    isLoading: summaryLoading,
    isPending: summaryPending,
    error: summaryError,
  } = useCVEAlertSummary();

  const updateStatus = useUpdateCVEAlertStatus();

  // AI hooks
  const aiContext = useAIContext();
  const sendAIMessage = useSendAIMessage();
  const { data: pendingTasks = [] } = usePendingTasks();

  const hasPendingCVETask = (pendingTasks || []).some(
    (task) =>
      task.user_intent?.toLowerCase().includes("cve") ||
      task.user_intent?.toLowerCase().includes("vulnerability") ||
      task.user_intent?.toLowerCase().includes("patch")
  );

  const handleAIPatch = async () => {
    setIsCreatingAITask(true);
    try {
      const criticalCount = (summary?.critical_alerts || 0) + (summary?.high_alerts || 0);
      const cisaKevCount = summary?.cisa_kev_alerts || 0;

      const intent =
        cisaKevCount > 0
          ? `Analyze and patch ${cisaKevCount} CVEs on the CISA KEV list. These are actively exploited and require immediate attention. Calculate blast radius and create a safe patch campaign.`
          : criticalCount > 0
          ? `Analyze and remediate ${criticalCount} critical/high severity CVE alerts. Prioritize by urgency score and production asset impact. Create a phased patch campaign with canary deployment.`
          : `Review the current CVE alert inventory for security improvements and patch recommendations.`;

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

  const handleInvestigate = async (alertId: string) => {
    try {
      await updateStatus.mutateAsync({
        alertId,
        data: { status: "investigating" },
      });
    } catch (error) {
      console.error("Failed to update status:", error);
    }
  };

  // Loading state
  if (alertsLoading || alertsPending || summaryLoading || summaryPending) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Vulnerability Response Center
            </h1>
            <p className="text-muted-foreground">
              Real-time CVE detection, blast radius analysis, and automated patch orchestration.
            </p>
          </div>
        </div>
        <PageSkeleton metricCards={4} showChart={false} showTable={true} tableRows={5} />
      </div>
    );
  }

  // Error state
  if (alertsError || summaryError) {
    return (
      <div className="page-transition space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Vulnerability Response Center
          </h1>
          <p className="text-muted-foreground">
            Real-time CVE detection, blast radius analysis, and automated patch orchestration.
          </p>
        </div>
        <ErrorState
          error={alertsError || summaryError || new Error("Unknown error")}
          retry={refetchAlerts}
          title="Failed to load vulnerability data"
          description="We couldn't fetch the CVE alert data. Please try again."
        />
      </div>
    );
  }

  const alerts = alertsData?.alerts || [];
  const criticalCount = (summary?.critical_alerts || 0) + (summary?.high_alerts || 0);
  const cisaKevCount = summary?.cisa_kev_alerts || 0;
  const activeAlerts = (summary?.new_alerts || 0) + (summary?.in_progress_alerts || 0);

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Vulnerability Response Center
          </h1>
          <p className="text-muted-foreground">
            Real-time CVE detection, blast radius analysis, and automated patch orchestration.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => refetchAlerts()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Total Alerts"
          value={summary?.total_alerts?.toString() || "0"}
          subtitle={`${activeAlerts} active`}
          status="neutral"
          icon={<Shield className="h-5 w-5" />}
        />
        <MetricCard
          title="Critical/High"
          value={criticalCount.toString()}
          subtitle={`${summary?.critical_alerts || 0} critical`}
          status={criticalCount > 0 ? "critical" : "success"}
          icon={<ShieldAlert className="h-5 w-5" />}
        />
        <MetricCard
          title="CISA KEV"
          value={cisaKevCount.toString()}
          subtitle="actively exploited"
          status={cisaKevCount > 0 ? "critical" : "success"}
          icon={<Bug className="h-5 w-5" />}
        />
        <MetricCard
          title="Affected Assets"
          value={summary?.total_affected_assets?.toString() || "0"}
          subtitle={`${summary?.production_affected_assets || 0} production`}
          status={(summary?.production_affected_assets || 0) > 0 ? "warning" : "success"}
          icon={<Server className="h-5 w-5" />}
        />
      </div>

      {/* AI Insight Card */}
      {(criticalCount > 0 || cisaKevCount > 0) && (
        <Card
          className={`border-l-4 ${
            cisaKevCount > 0
              ? "border-l-status-red bg-gradient-to-r from-status-red/5 to-transparent"
              : "border-l-status-amber bg-gradient-to-r from-status-amber/5 to-transparent"
          }`}
        >
          <CardContent className="flex items-start gap-4 p-6">
            <div
              className={`rounded-lg p-2 ${
                cisaKevCount > 0 ? "bg-status-red/10" : "bg-status-amber/10"
              }`}
            >
              <Sparkles
                className={`h-5 w-5 ${cisaKevCount > 0 ? "text-status-red" : "text-status-amber"}`}
              />
            </div>
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <h3 className="font-semibold">
                  {cisaKevCount > 0
                    ? `${cisaKevCount} CISA KEV CVEs Detected`
                    : `${criticalCount} High Priority Vulnerabilities`}
                </h3>
                <Badge
                  variant="outline"
                  className={`text-xs ${
                    cisaKevCount > 0
                      ? "border-status-red/50 text-status-red"
                      : "border-status-amber/50 text-status-amber"
                  }`}
                >
                  {cisaKevCount > 0 ? "CISA KEV" : "high priority"}
                </Badge>
              </div>
              <p className="mt-1 text-sm text-muted-foreground">
                {cisaKevCount > 0
                  ? "These CVEs are on the CISA Known Exploited Vulnerabilities catalog and require immediate remediation."
                  : `${summary?.production_affected_assets || 0} production assets affected. AI can calculate blast radius and orchestrate safe patching.`}
              </p>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              {hasPendingCVETask ? (
                <Button size="sm" variant="outline" onClick={() => router.push("/ai")}>
                  <Clock className="mr-2 h-4 w-4" />
                  View Pending Task
                </Button>
              ) : (
                <Button
                  size="sm"
                  onClick={handleAIPatch}
                  disabled={isCreatingAITask}
                  className={cisaKevCount > 0 ? "bg-status-red hover:bg-status-red/90" : ""}
                >
                  {isCreatingAITask ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Creating...
                    </>
                  ) : (
                    <>
                      <Zap className="mr-2 h-4 w-4" />
                      Patch with AI
                    </>
                  )}
                </Button>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* SLA Breached Warning */}
      {(summary?.sla_breached_alerts || 0) > 0 && (
        <Card className="border-l-4 border-l-status-red bg-gradient-to-r from-status-red/5 to-transparent">
          <CardContent className="flex items-center gap-4 p-4">
            <AlertTriangle className="h-5 w-5 text-status-red" />
            <div className="flex-1">
              <span className="font-medium text-status-red">
                {summary?.sla_breached_alerts} alerts have breached SLA
              </span>
              <span className="text-sm text-muted-foreground ml-2">
                - Immediate action required
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
              <span className="text-sm font-medium">Severity:</span>
              <Select
                value={severityFilter}
                onValueChange={(value) => setSeverityFilter(value as CVESeverity | "all")}
              >
                <SelectTrigger className="w-[150px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Severities</SelectItem>
                  <SelectItem value="critical">Critical</SelectItem>
                  <SelectItem value="high">High</SelectItem>
                  <SelectItem value="medium">Medium</SelectItem>
                  <SelectItem value="low">Low</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium">Status:</span>
              <Select
                value={statusFilter}
                onValueChange={(value) => setStatusFilter(value as CVEAlertStatus | "all")}
              >
                <SelectTrigger className="w-[180px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Statuses</SelectItem>
                  <SelectItem value="new">New</SelectItem>
                  <SelectItem value="investigating">Investigating</SelectItem>
                  <SelectItem value="confirmed">Confirmed</SelectItem>
                  <SelectItem value="in_progress">In Progress</SelectItem>
                  <SelectItem value="resolved">Resolved</SelectItem>
                  <SelectItem value="dismissed">Dismissed</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Tabs */}
      <Tabs defaultValue="all" className="space-y-4">
        <TabsList>
          <TabsTrigger value="all">All Alerts ({alerts.length})</TabsTrigger>
          <TabsTrigger value="cisa-kev">
            CISA KEV ({alerts.filter((a) => a.cve_details?.cisa_kev_listed).length})
          </TabsTrigger>
          <TabsTrigger value="exploitable">
            Exploitable ({alerts.filter((a) => a.cve_details?.exploit_available).length})
          </TabsTrigger>
        </TabsList>

        {/* All Alerts Tab */}
        <TabsContent value="all" className="space-y-4">
          <AlertsTable
            alerts={alerts}
            onInvestigate={handleInvestigate}
            isUpdating={updateStatus.isPending}
          />
        </TabsContent>

        {/* CISA KEV Tab */}
        <TabsContent value="cisa-kev" className="space-y-4">
          <AlertsTable
            alerts={alerts.filter((a) => a.cve_details?.cisa_kev_listed)}
            onInvestigate={handleInvestigate}
            isUpdating={updateStatus.isPending}
          />
        </TabsContent>

        {/* Exploitable Tab */}
        <TabsContent value="exploitable" className="space-y-4">
          <AlertsTable
            alerts={alerts.filter((a) => a.cve_details?.exploit_available)}
            onInvestigate={handleInvestigate}
            isUpdating={updateStatus.isPending}
          />
        </TabsContent>
      </Tabs>
    </div>
  );
}

// Alerts table component
function AlertsTable({
  alerts,
  onInvestigate,
  isUpdating,
}: {
  alerts: CVEAlert[];
  onInvestigate: (id: string) => void;
  isUpdating: boolean;
}) {
  const router = useRouter();

  if (alerts.length === 0) {
    return (
      <Card>
        <CardContent className="pt-6">
          <EmptyState
            variant="success"
            title="No CVE alerts"
            description="No CVE alerts match your current filters."
          />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">CVE Alerts</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>CVE ID</TableHead>
              <TableHead>Severity</TableHead>
              <TableHead>Urgency</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Blast Radius</TableHead>
              <TableHead>Indicators</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {alerts.map((alert) => (
              <AlertRow
                key={alert.id}
                alert={alert}
                onInvestigate={onInvestigate}
                isUpdating={isUpdating}
                onNavigate={() => router.push(`/vulnerabilities/${alert.id}`)}
              />
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

// Alert row component
function AlertRow({
  alert,
  onInvestigate,
  isUpdating,
  onNavigate,
}: {
  alert: CVEAlert;
  onInvestigate: (id: string) => void;
  isUpdating: boolean;
  onNavigate: () => void;
}) {
  const severityCfg = severityConfig[alert.severity];
  const statusCfg = statusConfig[alert.status];

  return (
    <TableRow className="cursor-pointer hover:bg-muted/50" onClick={onNavigate}>
      <TableCell>
        <div className="flex items-center gap-2">
          <Bug className="h-4 w-4 text-muted-foreground" />
          <div>
            <div className="font-medium font-mono">{alert.cve_id}</div>
            {alert.cve_details?.description && (
              <div className="text-xs text-muted-foreground max-w-[300px] truncate">
                {alert.cve_details.description}
              </div>
            )}
          </div>
        </div>
      </TableCell>
      <TableCell>
        <Badge className={severityCfg.className}>{severityCfg.label}</Badge>
      </TableCell>
      <TableCell>
        <UrgencyScoreBadge score={alert.urgency_score} />
      </TableCell>
      <TableCell>
        <Badge variant="outline" className={statusCfg.className}>
          {statusCfg.label}
        </Badge>
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-4 text-sm">
          <div className="flex items-center gap-1">
            <Target className="h-3 w-3 text-muted-foreground" />
            <span>{alert.affected_packages_count} pkgs</span>
          </div>
          <div className="flex items-center gap-1">
            <Cloud className="h-3 w-3 text-muted-foreground" />
            <span>{alert.affected_images_count} imgs</span>
          </div>
          <div className="flex items-center gap-1">
            <Server className="h-3 w-3 text-muted-foreground" />
            <span
              className={
                alert.production_assets_count > 0 ? "text-status-red font-medium" : ""
              }
            >
              {alert.affected_assets_count} assets
              {alert.production_assets_count > 0 && ` (${alert.production_assets_count} prod)`}
            </span>
          </div>
        </div>
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-2">
          {alert.cve_details?.cisa_kev_listed && (
            <Badge variant="destructive" className="text-[10px] px-1.5 py-0">
              KEV
            </Badge>
          )}
          {alert.cve_details?.exploit_available && (
            <Badge variant="outline" className="text-[10px] px-1.5 py-0 border-status-red text-status-red">
              Exploit
            </Badge>
          )}
          {alert.sla_breached && (
            <Badge variant="outline" className="text-[10px] px-1.5 py-0 border-status-amber text-status-amber">
              SLA
            </Badge>
          )}
        </div>
      </TableCell>
      <TableCell className="text-right">
        <div className="flex items-center justify-end gap-2">
          {alert.status === "new" && (
            <Button
              variant="outline"
              size="sm"
              onClick={(e) => {
                e.stopPropagation();
                onInvestigate(alert.id);
              }}
              disabled={isUpdating}
            >
              <Search className="h-3 w-3 mr-1" />
              Investigate
            </Button>
          )}
          <Button variant="ghost" size="sm" onClick={onNavigate}>
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}

// Urgency score badge component
function UrgencyScoreBadge({ score }: { score: number }) {
  const getScoreColor = (score: number) => {
    if (score >= 80) return "bg-status-red text-white";
    if (score >= 60) return "bg-status-amber text-white";
    if (score >= 40) return "bg-yellow-500 text-white";
    return "bg-blue-500 text-white";
  };

  return (
    <div className="flex items-center gap-2">
      <div className={`px-2 py-0.5 rounded text-xs font-medium ${getScoreColor(score)}`}>
        {score}
      </div>
      <div className="w-16 h-2 bg-muted rounded-full overflow-hidden">
        <div
          className={`h-full ${getScoreColor(score)}`}
          style={{ width: `${score}%` }}
        />
      </div>
    </div>
  );
}
