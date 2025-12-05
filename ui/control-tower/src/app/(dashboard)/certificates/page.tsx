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
  useCertificates,
  useCertificateSummary,
  useCertificateAlerts,
  useCertificateRotations,
  useAcknowledgeCertificateAlert,
} from "@/hooks/use-certificates";
import { useSendAIMessage, useAIContext, usePendingTasks } from "@/hooks/use-ai";
import {
  type Certificate,
  type CertificateAlert,
  type CertificateRotation,
  type CertificateStatus,
  type CertificatePlatform,
} from "@/lib/api";
import {
  Shield,
  ShieldAlert,
  Clock,
  RotateCcw,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Key,
  Globe,
  Server,
  Cloud,
  Zap,
  Loader2,
  Sparkles,
  RefreshCw,
  Bell,
} from "lucide-react";

// Platform icon mapping
const platformIcons: Record<CertificatePlatform, React.ReactNode> = {
  aws: <Cloud className="h-4 w-4 text-orange-400" />,
  azure: <Cloud className="h-4 w-4 text-blue-400" />,
  gcp: <Cloud className="h-4 w-4 text-red-400" />,
  k8s: <Server className="h-4 w-4 text-blue-500" />,
  vsphere: <Server className="h-4 w-4 text-green-500" />,
};

// Status badge configuration
const statusConfig: Record<CertificateStatus, { variant: "default" | "secondary" | "destructive" | "outline"; label: string }> = {
  active: { variant: "default", label: "Active" },
  expiring_soon: { variant: "secondary", label: "Expiring Soon" },
  expired: { variant: "destructive", label: "Expired" },
  revoked: { variant: "destructive", label: "Revoked" },
  pending_validation: { variant: "outline", label: "Pending" },
};

export default function CertificatesPage() {
  const router = useRouter();
  const [statusFilter, setStatusFilter] = useState<CertificateStatus | "all">("all");
  const [platformFilter, setPlatformFilter] = useState<CertificatePlatform | "all">("all");
  const [isCreatingAITask, setIsCreatingAITask] = useState(false);

  // Fetch data
  const {
    data: certificatesData,
    isLoading: certsLoading,
    isPending: certsPending,
    error: certsError,
    refetch: refetchCerts,
  } = useCertificates({
    status: statusFilter === "all" ? undefined : statusFilter,
    platform: platformFilter === "all" ? undefined : platformFilter,
    pageSize: 100,
  });

  const {
    data: summary,
    isLoading: summaryLoading,
    isPending: summaryPending,
    error: summaryError,
  } = useCertificateSummary();

  const { data: alertsData } = useCertificateAlerts({ status: "open", pageSize: 10 });
  const { data: rotationsData } = useCertificateRotations({ pageSize: 10 });
  const acknowledgeAlert = useAcknowledgeCertificateAlert();

  // AI hooks
  const aiContext = useAIContext();
  const sendAIMessage = useSendAIMessage();
  const { data: pendingTasks = [] } = usePendingTasks();

  const hasPendingCertTask = (pendingTasks || []).some(
    (task) => task.user_intent?.toLowerCase().includes("certificate") ||
              task.user_intent?.toLowerCase().includes("ssl") ||
              task.user_intent?.toLowerCase().includes("tls")
  );

  const handleAIRotation = async () => {
    setIsCreatingAITask(true);
    try {
      const expiringSoon = summary?.expiringSoon || 0;
      const expired = summary?.expired || 0;
      const intent = expiringSoon + expired > 0
        ? `Analyze and rotate ${expiringSoon + expired} certificates that are expiring soon or already expired. Prioritize based on blast radius and auto-renewal eligibility.`
        : `Review certificate inventory for security improvements and rotation recommendations.`;

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

  const handleAcknowledgeAlert = async (alertId: string) => {
    try {
      await acknowledgeAlert.mutateAsync(alertId);
    } catch (error) {
      console.error("Failed to acknowledge alert:", error);
    }
  };

  // Loading state
  if (certsLoading || certsPending || summaryLoading || summaryPending) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Certificate Lifecycle Management
            </h1>
            <p className="text-muted-foreground">
              Monitor, rotate, and manage TLS/SSL certificates across your infrastructure.
            </p>
          </div>
        </div>
        <PageSkeleton metricCards={4} showChart={false} showTable={true} tableRows={5} />
      </div>
    );
  }

  // Error state
  if (certsError || summaryError) {
    return (
      <div className="page-transition space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Certificate Lifecycle Management
          </h1>
          <p className="text-muted-foreground">
            Monitor, rotate, and manage TLS/SSL certificates across your infrastructure.
          </p>
        </div>
        <ErrorState
          error={certsError || summaryError || new Error("Unknown error")}
          retry={refetchCerts}
          title="Failed to load certificate data"
          description="We couldn't fetch the certificate data. Please try again."
        />
      </div>
    );
  }

  const certificates = certificatesData?.certificates || [];
  const alerts = alertsData?.alerts || [];
  const rotations = rotationsData?.rotations || [];

  // Calculate metrics
  const criticalCount = (summary?.expired || 0) + (summary?.expiring7Days || 0);
  const warningCount = summary?.expiring30Days || 0;

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Certificate Lifecycle Management
          </h1>
          <p className="text-muted-foreground">
            Monitor, rotate, and manage TLS/SSL certificates across your infrastructure.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => refetchCerts()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Total Certificates"
          value={summary?.totalCertificates?.toString() || "0"}
          subtitle={`${summary?.platformsCount || 0} platforms`}
          status="neutral"
          icon={<Shield className="h-5 w-5" />}
        />
        <MetricCard
          title="Active"
          value={summary?.activeCertificates?.toString() || "0"}
          subtitle={`${summary?.autoRenewEnabled || 0} auto-renew`}
          status="success"
          icon={<CheckCircle className="h-5 w-5" />}
        />
        <MetricCard
          title="Expiring Soon"
          value={summary?.expiringSoon?.toString() || "0"}
          subtitle={`${summary?.expiring7Days || 0} within 7 days`}
          status={criticalCount > 0 ? "critical" : warningCount > 0 ? "warning" : "success"}
          icon={<Clock className="h-5 w-5" />}
        />
        <MetricCard
          title="Expired"
          value={summary?.expired?.toString() || "0"}
          subtitle="requires attention"
          status={(summary?.expired || 0) > 0 ? "critical" : "success"}
          icon={<ShieldAlert className="h-5 w-5" />}
        />
      </div>

      {/* AI Insight Card */}
      {criticalCount > 0 && (
        <Card className={`border-l-4 ${
          criticalCount > 5
            ? "border-l-status-red bg-gradient-to-r from-status-red/5 to-transparent"
            : "border-l-status-amber bg-gradient-to-r from-status-amber/5 to-transparent"
        }`}>
          <CardContent className="flex items-start gap-4 p-6">
            <div className={`rounded-lg p-2 ${
              criticalCount > 5 ? "bg-status-red/10" : "bg-status-amber/10"
            }`}>
              <Sparkles className={`h-5 w-5 ${
                criticalCount > 5 ? "text-status-red" : "text-status-amber"
              }`} />
            </div>
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <h3 className="font-semibold">
                  {criticalCount} Certificates Need Attention
                </h3>
                <Badge
                  variant="outline"
                  className={`text-xs ${
                    criticalCount > 5
                      ? "border-status-red/50 text-status-red"
                      : "border-status-amber/50 text-status-amber"
                  }`}
                >
                  {criticalCount > 5 ? "critical" : "warning"}
                </Badge>
              </div>
              <p className="mt-1 text-sm text-muted-foreground">
                {summary?.expired || 0} expired and {summary?.expiring7Days || 0} expiring within 7 days.
                AI can analyze blast radius and orchestrate safe rotations.
              </p>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              {hasPendingCertTask ? (
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
                  onClick={handleAIRotation}
                  disabled={isCreatingAITask}
                  className={criticalCount > 5 ? "bg-status-red hover:bg-status-red/90" : ""}
                >
                  {isCreatingAITask ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Creating...
                    </>
                  ) : (
                    <>
                      <Zap className="mr-2 h-4 w-4" />
                      Rotate with AI
                    </>
                  )}
                </Button>
              )}
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
                onValueChange={(value) => setStatusFilter(value as CertificateStatus | "all")}
              >
                <SelectTrigger className="w-[180px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Statuses</SelectItem>
                  <SelectItem value="active">Active</SelectItem>
                  <SelectItem value="expiring_soon">Expiring Soon</SelectItem>
                  <SelectItem value="expired">Expired</SelectItem>
                  <SelectItem value="revoked">Revoked</SelectItem>
                  <SelectItem value="pending_validation">Pending</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium">Platform:</span>
              <Select
                value={platformFilter}
                onValueChange={(value) => setPlatformFilter(value as CertificatePlatform | "all")}
              >
                <SelectTrigger className="w-[150px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Platforms</SelectItem>
                  <SelectItem value="aws">AWS</SelectItem>
                  <SelectItem value="azure">Azure</SelectItem>
                  <SelectItem value="gcp">GCP</SelectItem>
                  <SelectItem value="k8s">Kubernetes</SelectItem>
                  <SelectItem value="vsphere">vSphere</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Tabs */}
      <Tabs defaultValue="certificates" className="space-y-4">
        <TabsList>
          <TabsTrigger value="certificates">
            Certificates ({certificates.length})
          </TabsTrigger>
          <TabsTrigger value="alerts">
            Alerts ({alerts.length})
          </TabsTrigger>
          <TabsTrigger value="rotations">
            Rotations ({rotations.length})
          </TabsTrigger>
        </TabsList>

        {/* Certificates Tab */}
        <TabsContent value="certificates" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Certificate Inventory</CardTitle>
            </CardHeader>
            <CardContent>
              {certificates.length > 0 ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Common Name</TableHead>
                      <TableHead>Platform</TableHead>
                      <TableHead>Source</TableHead>
                      <TableHead>Expires</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Auto-Renew</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {certificates.map((cert) => (
                      <CertificateRow key={cert.id} certificate={cert} />
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <EmptyState
                  variant="data"
                  title="No certificates found"
                  description="No certificates match your current filters."
                />
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Alerts Tab */}
        <TabsContent value="alerts" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Certificate Alerts</CardTitle>
            </CardHeader>
            <CardContent>
              {alerts.length > 0 ? (
                <div className="space-y-3">
                  {alerts.map((alert) => (
                    <AlertCard
                      key={alert.id}
                      alert={alert}
                      onAcknowledge={handleAcknowledgeAlert}
                      isAcknowledging={acknowledgeAlert.isPending}
                    />
                  ))}
                </div>
              ) : (
                <EmptyState
                  variant="success"
                  title="No open alerts"
                  description="All certificate alerts have been addressed."
                />
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Rotations Tab */}
        <TabsContent value="rotations" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Recent Rotations</CardTitle>
            </CardHeader>
            <CardContent>
              {rotations.length > 0 ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Type</TableHead>
                      <TableHead>Initiated By</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Affected</TableHead>
                      <TableHead>Success Rate</TableHead>
                      <TableHead>Started</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {rotations.map((rotation) => (
                      <RotationRow key={rotation.id} rotation={rotation} />
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <EmptyState
                  variant="data"
                  title="No recent rotations"
                  description="No certificate rotations have been performed recently."
                />
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

// Certificate row component
function CertificateRow({ certificate }: { certificate: Certificate }) {
  const statusCfg = statusConfig[certificate.status];
  const daysUntilExpiry = certificate.daysUntilExpiry;

  return (
    <TableRow>
      <TableCell>
        <div className="flex items-center gap-2">
          <Key className="h-4 w-4 text-muted-foreground" />
          <div>
            <div className="font-medium">{certificate.commonName}</div>
            {certificate.subjectAltNames && certificate.subjectAltNames.length > 0 && (
              <div className="text-xs text-muted-foreground">
                +{certificate.subjectAltNames.length} SANs
              </div>
            )}
          </div>
        </div>
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-2">
          {platformIcons[certificate.platform]}
          <span className="capitalize">{certificate.platform}</span>
        </div>
      </TableCell>
      <TableCell>
        <span className="text-sm text-muted-foreground capitalize">
          {certificate.source.replace(/_/g, " ")}
        </span>
      </TableCell>
      <TableCell>
        <div className={daysUntilExpiry < 0 ? "text-status-red" : daysUntilExpiry < 7 ? "text-status-amber" : ""}>
          {daysUntilExpiry < 0 ? (
            <span className="font-medium">Expired {Math.abs(daysUntilExpiry)} days ago</span>
          ) : (
            <span>{daysUntilExpiry} days</span>
          )}
        </div>
      </TableCell>
      <TableCell>
        <Badge variant={statusCfg.variant}>{statusCfg.label}</Badge>
      </TableCell>
      <TableCell>
        {certificate.autoRenew ? (
          <CheckCircle className="h-4 w-4 text-status-green" />
        ) : (
          <XCircle className="h-4 w-4 text-muted-foreground" />
        )}
      </TableCell>
    </TableRow>
  );
}

// Alert card component
function AlertCard({
  alert,
  onAcknowledge,
  isAcknowledging,
}: {
  alert: CertificateAlert;
  onAcknowledge: (id: string) => void;
  isAcknowledging: boolean;
}) {
  const severityColors = {
    critical: "border-l-status-red bg-status-red/5",
    high: "border-l-status-amber bg-status-amber/5",
    medium: "border-l-yellow-500 bg-yellow-500/5",
    low: "border-l-blue-500 bg-blue-500/5",
  };

  const severityIcons = {
    critical: <AlertTriangle className="h-5 w-5 text-status-red" />,
    high: <AlertTriangle className="h-5 w-5 text-status-amber" />,
    medium: <Bell className="h-5 w-5 text-yellow-500" />,
    low: <Bell className="h-5 w-5 text-blue-500" />,
  };

  return (
    <div className={`flex items-start gap-4 rounded-lg border-l-4 p-4 ${severityColors[alert.severity]}`}>
      <div className="shrink-0">{severityIcons[alert.severity]}</div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <h4 className="font-medium text-sm">{alert.title}</h4>
          <Badge variant="outline" className="text-xs capitalize">
            {alert.severity}
          </Badge>
        </div>
        <p className="text-sm text-muted-foreground mt-1">{alert.message}</p>
        <div className="text-xs text-muted-foreground mt-2">
          {formatRelativeTime(alert.createdAt)}
        </div>
      </div>
      {alert.status === "open" && (
        <Button
          variant="outline"
          size="sm"
          onClick={() => onAcknowledge(alert.id)}
          disabled={isAcknowledging}
        >
          Acknowledge
        </Button>
      )}
    </div>
  );
}

// Rotation row component
function RotationRow({ rotation }: { rotation: CertificateRotation }) {
  const statusColors = {
    pending: "bg-gray-100 text-gray-700",
    in_progress: "bg-blue-100 text-blue-700",
    completed: "bg-green-100 text-green-700",
    failed: "bg-red-100 text-red-700",
    rolled_back: "bg-amber-100 text-amber-700",
  };

  const successRate = rotation.affectedUsages > 0
    ? Math.round((rotation.successfulUpdates / rotation.affectedUsages) * 100)
    : 0;

  return (
    <TableRow>
      <TableCell className="capitalize">{rotation.rotationType}</TableCell>
      <TableCell>
        <div className="flex items-center gap-2">
          {rotation.initiatedBy === "ai_agent" && (
            <Sparkles className="h-4 w-4 text-primary" />
          )}
          <span className="capitalize">{rotation.initiatedBy.replace(/_/g, " ")}</span>
        </div>
      </TableCell>
      <TableCell>
        <Badge className={statusColors[rotation.status]}>
          {rotation.status.replace(/_/g, " ")}
        </Badge>
      </TableCell>
      <TableCell>{rotation.affectedUsages}</TableCell>
      <TableCell>
        <div className="flex items-center gap-2">
          <span className={successRate === 100 ? "text-status-green" : successRate > 0 ? "text-status-amber" : "text-muted-foreground"}>
            {successRate}%
          </span>
          <span className="text-xs text-muted-foreground">
            ({rotation.successfulUpdates}/{rotation.affectedUsages})
          </span>
        </div>
      </TableCell>
      <TableCell>
        {rotation.startedAt ? formatRelativeTime(rotation.startedAt) : "-"}
      </TableCell>
    </TableRow>
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
  if (diffHours < 24) return `${diffHours} hours ago`;

  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 7) return `${diffDays} days ago`;

  return date.toLocaleDateString();
}
