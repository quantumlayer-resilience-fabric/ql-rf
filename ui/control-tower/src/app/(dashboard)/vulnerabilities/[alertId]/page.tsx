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
import { Separator } from "@/components/ui/separator";
import { PageSkeleton, ErrorState, EmptyState } from "@/components/feedback";
import {
  useCVEAlert,
  useBlastRadius,
  useUpdateCVEAlertStatus,
  useCreatePatchCampaign,
  type CVEAlertWithBlastRadius,
  type BlastRadiusResult,
  type CVESeverity,
} from "@/hooks/use-cve-alerts";
import { useSendAIMessage, useAIContext } from "@/hooks/use-ai";
import {
  ArrowLeft,
  Bug,
  Shield,
  ShieldAlert,
  AlertTriangle,
  Clock,
  Target,
  Server,
  Cloud,
  Package,
  ExternalLink,
  Zap,
  Loader2,
  Sparkles,
  CheckCircle,
  XCircle,
  Calendar,
  User,
  FileText,
  GitBranch,
} from "lucide-react";

// Severity badge configuration
const severityConfig: Record<CVESeverity, { variant: "default" | "secondary" | "destructive" | "outline"; label: string; className: string }> = {
  critical: { variant: "destructive", label: "Critical", className: "bg-status-red text-white" },
  high: { variant: "default", label: "High", className: "bg-status-amber text-white" },
  medium: { variant: "secondary", label: "Medium", className: "bg-yellow-500 text-white" },
  low: { variant: "outline", label: "Low", className: "border-blue-500 text-blue-500" },
  unknown: { variant: "outline", label: "Unknown", className: "" },
};

export default function CVEAlertDetailPage({
  params,
}: {
  params: Promise<{ alertId: string }>;
}) {
  const { alertId } = use(params);
  const router = useRouter();
  const [isCreatingCampaign, setIsCreatingCampaign] = useState(false);
  const [isCreatingAITask, setIsCreatingAITask] = useState(false);

  // Fetch data
  const {
    data: alert,
    isLoading: alertLoading,
    error: alertError,
    refetch: refetchAlert,
  } = useCVEAlert(alertId);

  const {
    data: blastRadius,
    isLoading: blastRadiusLoading,
    error: blastRadiusError,
  } = useBlastRadius(alertId);

  const updateStatus = useUpdateCVEAlertStatus();
  const createCampaign = useCreatePatchCampaign();

  // AI hooks
  const aiContext = useAIContext();
  const sendAIMessage = useSendAIMessage();

  const handleStatusChange = async (newStatus: string) => {
    try {
      await updateStatus.mutateAsync({
        alertId,
        data: { status: newStatus as any },
      });
    } catch (error) {
      console.error("Failed to update status:", error);
    }
  };

  const handleCreateCampaign = async () => {
    if (!alert) return;
    setIsCreatingCampaign(true);
    try {
      await createCampaign.mutateAsync({
        alertId,
        data: {
          name: `CVE-Response-${alert.cve_id}`,
          campaign_type: "cve_response",
          rollout_strategy: "canary",
          canary_percentage: 5,
          wave_percentage: 25,
          requires_approval: true,
        },
      });
      router.push("/ai");
    } catch (error) {
      console.error("Failed to create campaign:", error);
    } finally {
      setIsCreatingCampaign(false);
    }
  };

  const handleAIPatch = async () => {
    if (!alert) return;
    setIsCreatingAITask(true);
    try {
      const intent = `Analyze and patch ${alert.cve_id} (${alert.severity} severity, urgency score ${alert.urgency_score}).
      Affected: ${alert.affected_packages_count} packages, ${alert.affected_images_count} images, ${alert.affected_assets_count} assets (${alert.production_assets_count} production).
      ${alert.cve_details?.cisa_kev_listed ? "This CVE is on the CISA KEV list - immediate action required." : ""}
      Calculate full blast radius through image lineage and create a safe patch campaign with canary deployment.`;

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

  // Loading state
  if (alertLoading || blastRadiusLoading) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => router.back()}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">CVE Alert Details</h1>
            <p className="text-muted-foreground">Loading alert information...</p>
          </div>
        </div>
        <PageSkeleton metricCards={3} showChart={false} showTable={true} tableRows={5} />
      </div>
    );
  }

  // Error state
  if (alertError || blastRadiusError) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => router.back()}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">CVE Alert Details</h1>
          </div>
        </div>
        <ErrorState
          error={alertError || blastRadiusError || new Error("Unknown error")}
          retry={refetchAlert}
          title="Failed to load alert details"
          description="We couldn't fetch the CVE alert data. Please try again."
        />
      </div>
    );
  }

  if (!alert) {
    return (
      <div className="page-transition space-y-6">
        <EmptyState
          variant="data"
          title="Alert not found"
          description="The requested CVE alert could not be found."
        />
      </div>
    );
  }

  const severityCfg = severityConfig[alert.severity];
  const cveDetails = alert.cve_details;

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
              <h1 className="text-2xl font-bold tracking-tight text-foreground font-mono">
                {alert.cve_id}
              </h1>
              <Badge className={severityCfg.className}>{severityCfg.label}</Badge>
              {cveDetails?.cisa_kev_listed && (
                <Badge variant="destructive">CISA KEV</Badge>
              )}
              {cveDetails?.exploit_available && (
                <Badge variant="outline" className="border-status-red text-status-red">
                  Exploit Available
                </Badge>
              )}
            </div>
            <p className="text-muted-foreground mt-1 max-w-2xl">
              {cveDetails?.description || "No description available"}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={handleAIPatch}
            disabled={isCreatingAITask}
          >
            {isCreatingAITask ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Creating...
              </>
            ) : (
              <>
                <Sparkles className="mr-2 h-4 w-4" />
                Analyze with AI
              </>
            )}
          </Button>
          <Button onClick={handleCreateCampaign} disabled={isCreatingCampaign}>
            {isCreatingCampaign ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Creating...
              </>
            ) : (
              <>
                <Zap className="mr-2 h-4 w-4" />
                Create Patch Campaign
              </>
            )}
          </Button>
        </div>
      </div>

      {/* Overview Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Urgency Score
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-4">
              <div className="text-3xl font-bold">{alert.urgency_score}</div>
              <div className="flex-1">
                <div className="h-3 bg-muted rounded-full overflow-hidden">
                  <div
                    className={`h-full ${
                      alert.urgency_score >= 80
                        ? "bg-status-red"
                        : alert.urgency_score >= 60
                        ? "bg-status-amber"
                        : alert.urgency_score >= 40
                        ? "bg-yellow-500"
                        : "bg-blue-500"
                    }`}
                    style={{ width: `${alert.urgency_score}%` }}
                  />
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              CVSS Score
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <div className="text-3xl font-bold">
                {cveDetails?.cvss_v3_score?.toFixed(1) || "N/A"}
              </div>
              {cveDetails?.cvss_v3_vector && (
                <Badge variant="outline" className="text-xs font-mono">
                  {cveDetails.cvss_v3_vector.slice(0, 20)}...
                </Badge>
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              EPSS Score
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <div className="text-3xl font-bold">
                {cveDetails?.epss_score ? `${(cveDetails.epss_score * 100).toFixed(1)}%` : "N/A"}
              </div>
              {cveDetails?.epss_percentile && (
                <span className="text-sm text-muted-foreground">
                  (top {(100 - cveDetails.epss_percentile * 100).toFixed(1)}%)
                </span>
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              SLA Status
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              {alert.sla_breached ? (
                <>
                  <XCircle className="h-5 w-5 text-status-red" />
                  <span className="text-status-red font-medium">Breached</span>
                </>
              ) : alert.sla_due_at ? (
                <>
                  <Clock className="h-5 w-5 text-status-amber" />
                  <span>Due {formatDate(alert.sla_due_at)}</span>
                </>
              ) : (
                <>
                  <CheckCircle className="h-5 w-5 text-status-green" />
                  <span className="text-status-green">On Track</span>
                </>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Blast Radius Summary */}
      <Card className="border-l-4 border-l-status-amber">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Target className="h-5 w-5" />
            Blast Radius
          </CardTitle>
          <CardDescription>
            Impact analysis showing affected packages, images, and assets
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-6 md:grid-cols-4">
            <div className="flex items-center gap-4">
              <div className="rounded-lg bg-purple-500/10 p-3">
                <Package className="h-5 w-5 text-purple-500" />
              </div>
              <div>
                <div className="text-2xl font-bold">{alert.affected_packages_count}</div>
                <div className="text-sm text-muted-foreground">Packages</div>
              </div>
            </div>
            <div className="flex items-center gap-4">
              <div className="rounded-lg bg-blue-500/10 p-3">
                <Cloud className="h-5 w-5 text-blue-500" />
              </div>
              <div>
                <div className="text-2xl font-bold">{alert.affected_images_count}</div>
                <div className="text-sm text-muted-foreground">Images</div>
              </div>
            </div>
            <div className="flex items-center gap-4">
              <div className="rounded-lg bg-green-500/10 p-3">
                <Server className="h-5 w-5 text-green-500" />
              </div>
              <div>
                <div className="text-2xl font-bold">{alert.affected_assets_count}</div>
                <div className="text-sm text-muted-foreground">Assets</div>
              </div>
            </div>
            <div className="flex items-center gap-4">
              <div className="rounded-lg bg-red-500/10 p-3">
                <AlertTriangle className="h-5 w-5 text-red-500" />
              </div>
              <div>
                <div className="text-2xl font-bold text-status-red">
                  {alert.production_assets_count}
                </div>
                <div className="text-sm text-muted-foreground">Production</div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Tabs for Details */}
      <Tabs defaultValue="affected" className="space-y-4">
        <TabsList>
          <TabsTrigger value="affected">Affected Items</TabsTrigger>
          <TabsTrigger value="details">CVE Details</TabsTrigger>
          <TabsTrigger value="timeline">Timeline</TabsTrigger>
        </TabsList>

        {/* Affected Items Tab */}
        <TabsContent value="affected" className="space-y-4">
          {blastRadius && <BlastRadiusDetails blastRadius={blastRadius} />}
        </TabsContent>

        {/* CVE Details Tab */}
        <TabsContent value="details" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>CVE Information</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <div>
                  <h4 className="text-sm font-medium text-muted-foreground">Published Date</h4>
                  <p>{cveDetails?.published_date ? formatDate(cveDetails.published_date) : "Unknown"}</p>
                </div>
                <div>
                  <h4 className="text-sm font-medium text-muted-foreground">Last Modified</h4>
                  <p>{cveDetails?.modified_date ? formatDate(cveDetails.modified_date) : "Unknown"}</p>
                </div>
                <div>
                  <h4 className="text-sm font-medium text-muted-foreground">Primary Source</h4>
                  <p className="uppercase">{cveDetails?.primary_source || "Unknown"}</p>
                </div>
                <div>
                  <h4 className="text-sm font-medium text-muted-foreground">Exploit Maturity</h4>
                  <p className="capitalize">{cveDetails?.exploit_maturity || "Unknown"}</p>
                </div>
              </div>

              {cveDetails?.cisa_kev_listed && (
                <>
                  <Separator />
                  <div>
                    <h4 className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                      <ShieldAlert className="h-4 w-4 text-status-red" />
                      CISA KEV Information
                    </h4>
                    <div className="mt-2 grid gap-2 md:grid-cols-2">
                      {cveDetails.cisa_kev_due_date && (
                        <div>
                          <span className="text-sm text-muted-foreground">Due Date: </span>
                          <span className="font-medium">{formatDate(cveDetails.cisa_kev_due_date)}</span>
                        </div>
                      )}
                      {cveDetails.cisa_kev_ransomware && (
                        <Badge variant="destructive">Known Ransomware Use</Badge>
                      )}
                    </div>
                  </div>
                </>
              )}

              {cveDetails?.remediation_summary && (
                <>
                  <Separator />
                  <div>
                    <h4 className="text-sm font-medium text-muted-foreground">Remediation</h4>
                    <p className="mt-1">{cveDetails.remediation_summary}</p>
                  </div>
                </>
              )}

              {cveDetails?.reference_urls && cveDetails.reference_urls.length > 0 && (
                <>
                  <Separator />
                  <div>
                    <h4 className="text-sm font-medium text-muted-foreground">References</h4>
                    <div className="mt-2 space-y-1">
                      {cveDetails.reference_urls.slice(0, 5).map((url, i) => (
                        <a
                          key={i}
                          href={url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="flex items-center gap-2 text-sm text-primary hover:underline"
                        >
                          <ExternalLink className="h-3 w-3" />
                          {url}
                        </a>
                      ))}
                    </div>
                  </div>
                </>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Timeline Tab */}
        <TabsContent value="timeline" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Alert Timeline</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <TimelineItem
                  icon={<Bug className="h-4 w-4" />}
                  title="First Detected"
                  date={alert.detected_at}
                  description="CVE first matched to packages in your SBOM inventory"
                />
                <TimelineItem
                  icon={<Calendar className="h-4 w-4" />}
                  title="Alert Created"
                  date={alert.created_at}
                  description="Alert generated and added to queue"
                />
                {alert.assigned_to && alert.assigned_at && (
                  <TimelineItem
                    icon={<User className="h-4 w-4" />}
                    title="Assigned"
                    date={alert.assigned_at}
                    description={`Assigned to ${alert.assigned_to}`}
                  />
                )}
                {alert.resolved_at && (
                  <TimelineItem
                    icon={<CheckCircle className="h-4 w-4 text-status-green" />}
                    title="Resolved"
                    date={alert.resolved_at}
                    description={`Resolved by ${alert.resolved_by || "System"}: ${alert.resolution_type || "N/A"}`}
                  />
                )}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Status Actions */}
      <Card>
        <CardHeader>
          <CardTitle>Actions</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-2">
            {alert.status === "new" && (
              <Button
                variant="outline"
                onClick={() => handleStatusChange("investigating")}
                disabled={updateStatus.isPending}
              >
                Start Investigation
              </Button>
            )}
            {alert.status === "investigating" && (
              <Button
                variant="outline"
                onClick={() => handleStatusChange("confirmed")}
                disabled={updateStatus.isPending}
              >
                Confirm Vulnerability
              </Button>
            )}
            {["new", "investigating", "confirmed"].includes(alert.status) && (
              <Button
                variant="outline"
                onClick={() => handleStatusChange("in_progress")}
                disabled={updateStatus.isPending}
              >
                Mark In Progress
              </Button>
            )}
            {["new", "investigating", "confirmed", "in_progress"].includes(alert.status) && (
              <>
                <Button
                  variant="default"
                  onClick={() => handleStatusChange("resolved")}
                  disabled={updateStatus.isPending}
                >
                  Mark Resolved
                </Button>
                <Button
                  variant="ghost"
                  onClick={() => handleStatusChange("dismissed")}
                  disabled={updateStatus.isPending}
                >
                  Dismiss
                </Button>
              </>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// Blast Radius Details Component
function BlastRadiusDetails({ blastRadius }: { blastRadius: BlastRadiusResult }) {
  return (
    <div className="space-y-4">
      {/* Affected Packages */}
      {blastRadius.affected_packages.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Package className="h-4 w-4" />
              Affected Packages ({blastRadius.affected_packages.length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Package</TableHead>
                  <TableHead>Version</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Fixed Version</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {blastRadius.affected_packages.slice(0, 10).map((pkg) => (
                  <TableRow key={pkg.package_id}>
                    <TableCell className="font-medium">{pkg.package_name}</TableCell>
                    <TableCell className="font-mono text-sm">{pkg.package_version}</TableCell>
                    <TableCell>{pkg.package_type}</TableCell>
                    <TableCell>
                      {pkg.fixed_version ? (
                        <Badge variant="outline" className="text-status-green border-status-green">
                          {pkg.fixed_version}
                        </Badge>
                      ) : (
                        <span className="text-muted-foreground">No fix available</span>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Affected Images */}
      {blastRadius.affected_images.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Cloud className="h-4 w-4" />
              Affected Images ({blastRadius.affected_images.length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Image</TableHead>
                  <TableHead>Version</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Lineage</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {blastRadius.affected_images.slice(0, 10).map((img) => (
                  <TableRow key={img.image_id}>
                    <TableCell className="font-medium">{img.image_family}</TableCell>
                    <TableCell className="font-mono text-sm">{img.image_version}</TableCell>
                    <TableCell>
                      {img.is_direct ? (
                        <Badge variant="destructive">Direct</Badge>
                      ) : (
                        <Badge variant="outline">Inherited</Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">
                        <GitBranch className="h-3 w-3 text-muted-foreground" />
                        <span>Depth {img.lineage_depth}</span>
                        {img.child_image_ids && img.child_image_ids.length > 0 && (
                          <span className="text-muted-foreground">
                            ({img.child_image_ids.length} children)
                          </span>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Affected Assets */}
      {blastRadius.affected_assets.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Server className="h-4 w-4" />
              Affected Assets ({blastRadius.affected_assets.length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Asset</TableHead>
                  <TableHead>Platform</TableHead>
                  <TableHead>Region</TableHead>
                  <TableHead>Environment</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {blastRadius.affected_assets.slice(0, 10).map((asset) => (
                  <TableRow key={asset.asset_id}>
                    <TableCell className="font-medium">{asset.asset_name}</TableCell>
                    <TableCell className="uppercase">{asset.platform}</TableCell>
                    <TableCell>{asset.region}</TableCell>
                    <TableCell>
                      {asset.is_production ? (
                        <Badge variant="destructive">Production</Badge>
                      ) : (
                        <Badge variant="outline" className="capitalize">
                          {asset.environment}
                        </Badge>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

// Timeline Item Component
function TimelineItem({
  icon,
  title,
  date,
  description,
}: {
  icon: React.ReactNode;
  title: string;
  date: string;
  description: string;
}) {
  return (
    <div className="flex items-start gap-4">
      <div className="rounded-full bg-muted p-2">{icon}</div>
      <div className="flex-1">
        <div className="flex items-center justify-between">
          <h4 className="font-medium">{title}</h4>
          <span className="text-sm text-muted-foreground">{formatDate(date)}</span>
        </div>
        <p className="text-sm text-muted-foreground">{description}</p>
      </div>
    </div>
  );
}

function formatDate(dateString: string): string {
  const date = new Date(dateString);
  return date.toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}
