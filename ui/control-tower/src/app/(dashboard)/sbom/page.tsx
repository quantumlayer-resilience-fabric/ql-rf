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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { MetricCard } from "@/components/data/metric-card";
import { StatusBadge } from "@/components/status/status-badge";
import { PageSkeleton, ErrorState, EmptyState } from "@/components/feedback";
import { SBOMComponentsTable } from "@/components/sbom/sbom-components-table";
import { SBOMVulnerabilityCard } from "@/components/sbom/sbom-vulnerability-card";
import { LicenseDistributionChart } from "@/components/sbom/license-distribution-chart";
import {
  useSBOMs,
  useSBOMComponents,
  useSBOMVulnerabilities,
  useLicenseSummary,
  useGenerateSBOM,
} from "@/hooks/use-sbom";
import { useSendAIMessage, useAIContext, usePendingTasks } from "@/hooks/use-ai";
import { api } from "@/lib/api";
import { useQuery } from "@tanstack/react-query";
import {
  Package,
  Shield,
  Scale,
  Clock,
  Download,
  FileText,
  Loader2,
  Sparkles,
  Zap,
  AlertTriangle,
} from "lucide-react";

export default function SBOMPage() {
  const router = useRouter();
  const [selectedSBOMId, setSelectedSBOMId] = useState<string | undefined>();
  const [generateDialogOpen, setGenerateDialogOpen] = useState(false);
  const [selectedImageId, setSelectedImageId] = useState<string>("");
  const [selectedFormat, setSelectedFormat] = useState<"spdx" | "cyclonedx">("spdx");
  const [isCreatingAITask, setIsCreatingAITask] = useState(false);

  // Fetch data
  const { data: sbomList, isLoading, error, refetch } = useSBOMs({ page: 1, pageSize: 100 });
  const { data: images } = useQuery({
    queryKey: ["images"],
    queryFn: () => api.images.listFamilies(),
  });

  // Get the most recent SBOM if we have data
  const latestSBOM = sbomList?.sboms?.[0];
  const activeSBOMId = selectedSBOMId || latestSBOM?.id;

  const { data: components = [] } = useSBOMComponents({
    sbomId: activeSBOMId,
  });

  const { data: vulnerabilitiesData } = useSBOMVulnerabilities(
    activeSBOMId || "",
    {}
  );

  const { data: licenseSummary } = useLicenseSummary();
  const generateSBOM = useGenerateSBOM();

  // AI hooks
  const aiContext = useAIContext();
  const sendAIMessage = useSendAIMessage();
  const { data: pendingTasks = [] } = usePendingTasks();

  const hasPendingSBOMTask = pendingTasks.some(
    (task) => task.user_intent?.toLowerCase().includes("sbom") ||
              task.user_intent?.toLowerCase().includes("vulnerability")
  );

  const handleGenerateSBOM = async () => {
    if (!selectedImageId) return;

    try {
      await generateSBOM.mutateAsync({
        imageId: selectedImageId,
        request: {
          format: selectedFormat,
          includeVulns: true,
        },
      });
      setGenerateDialogOpen(false);
      refetch();
    } catch (error) {
      console.error("Failed to generate SBOM:", error);
    }
  };

  const handleAIRemediation = async () => {
    setIsCreatingAITask(true);
    try {
      const criticalCount = vulnerabilitiesData?.stats?.critical || 0;
      const highCount = vulnerabilitiesData?.stats?.high || 0;
      const intent = criticalCount + highCount > 0
        ? `Analyze and remediate ${criticalCount + highCount} critical and high severity vulnerabilities found in SBOM. Prioritize components with available fixes.`
        : `Review SBOM for security improvements and license compliance.`;

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

  if (isLoading) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Software Bill of Materials (SBOM)
            </h1>
            <p className="text-muted-foreground">
              Track software components, licenses, and vulnerabilities across your golden images.
            </p>
          </div>
        </div>
        <PageSkeleton metricCards={4} showChart={false} showTable={true} tableRows={5} />
      </div>
    );
  }

  if (error) {
    return (
      <div className="page-transition space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Software Bill of Materials (SBOM)
          </h1>
          <p className="text-muted-foreground">
            Track software components, licenses, and vulnerabilities across your golden images.
          </p>
        </div>
        <ErrorState
          error={error}
          retry={refetch}
          title="Failed to load SBOM data"
          description="We couldn't fetch the SBOM data. Please try again."
        />
      </div>
    );
  }

  // Calculate metrics
  const totalComponents = components.length;
  const totalVulnerabilities = (vulnerabilitiesData?.stats?.critical ?? 0) +
                               (vulnerabilitiesData?.stats?.high ?? 0) +
                               (vulnerabilitiesData?.stats?.medium ?? 0) +
                               (vulnerabilitiesData?.stats?.low ?? 0);
  const criticalHighCount = (vulnerabilitiesData?.stats?.critical ?? 0) +
                           (vulnerabilitiesData?.stats?.high ?? 0);

  // Calculate license compliance percentage
  const licensedPackages = (licenseSummary?.totalPackages ?? 0) - (licenseSummary?.unlicensedPackages ?? 0);
  const licenseCompliance = licenseSummary?.totalPackages
    ? (licensedPackages / licenseSummary.totalPackages) * 100
    : 100;

  const lastGenerated = latestSBOM?.generatedAt
    ? formatRelativeTime(latestSBOM.generatedAt)
    : "Never";

  // Get vulnerabilities array
  const vulnerabilities = vulnerabilitiesData?.vulnerabilities || [];

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Software Bill of Materials (SBOM)
          </h1>
          <p className="text-muted-foreground">
            Track software components, licenses, and vulnerabilities across your golden images.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              if (activeSBOMId) {
                // Export SBOM functionality would go here
                console.log("Export SBOM:", activeSBOMId);
              }
            }}
            disabled={!activeSBOMId}
          >
            <Download className="mr-2 h-4 w-4" />
            Export
          </Button>
          <Dialog open={generateDialogOpen} onOpenChange={setGenerateDialogOpen}>
            <DialogTrigger asChild>
              <Button size="sm">
                <FileText className="mr-2 h-4 w-4" />
                Generate SBOM
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Generate SBOM</DialogTitle>
                <DialogDescription>
                  Create a Software Bill of Materials for a golden image.
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4 py-4">
                <div className="space-y-2">
                  <Label htmlFor="image">Golden Image</Label>
                  <Select value={selectedImageId} onValueChange={setSelectedImageId}>
                    <SelectTrigger id="image">
                      <SelectValue placeholder="Select an image" />
                    </SelectTrigger>
                    <SelectContent>
                      {images?.map((family) =>
                        family.versions.map((version) => (
                          <SelectItem key={version.id} value={version.id}>
                            {family.name} - {version.version}
                          </SelectItem>
                        ))
                      )}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="format">SBOM Format</Label>
                  <Select
                    value={selectedFormat}
                    onValueChange={(value: "spdx" | "cyclonedx") =>
                      setSelectedFormat(value)
                    }
                  >
                    <SelectTrigger id="format">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="spdx">SPDX</SelectItem>
                      <SelectItem value="cyclonedx">CycloneDX</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setGenerateDialogOpen(false)}
                >
                  Cancel
                </Button>
                <Button
                  onClick={handleGenerateSBOM}
                  disabled={!selectedImageId || generateSBOM.isPending}
                >
                  {generateSBOM.isPending ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Generating...
                    </>
                  ) : (
                    "Generate"
                  )}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Total Components"
          value={totalComponents.toString()}
          subtitle="packages tracked"
          status="neutral"
          icon={<Package className="h-5 w-5" />}
        />
        <MetricCard
          title="Vulnerabilities"
          value={`${criticalHighCount}`}
          subtitle={`critical/high of ${totalVulnerabilities} total`}
          status={criticalHighCount > 10 ? "critical" : criticalHighCount > 5 ? "warning" : "success"}
          icon={<Shield className="h-5 w-5" />}
        />
        <MetricCard
          title="License Compliance"
          value={`${licenseCompliance.toFixed(1)}%`}
          subtitle="packages licensed"
          status={licenseCompliance >= 95 ? "success" : licenseCompliance >= 80 ? "warning" : "critical"}
          icon={<Scale className="h-5 w-5" />}
        />
        <MetricCard
          title="Last Generated"
          value={lastGenerated}
          subtitle="SBOM scan"
          status="neutral"
          icon={<Clock className="h-5 w-5" />}
        />
      </div>

      {/* AI Insight Card */}
      {criticalHighCount > 0 && (
        <Card className={`border-l-4 ${
          criticalHighCount > 10
            ? "border-l-status-red bg-gradient-to-r from-status-red/5 to-transparent"
            : "border-l-status-amber bg-gradient-to-r from-status-amber/5 to-transparent"
        }`}>
          <CardContent className="flex items-start gap-4 p-6">
            <div className={`rounded-lg p-2 ${
              criticalHighCount > 10 ? "bg-status-red/10" : "bg-status-amber/10"
            }`}>
              <Sparkles className={`h-5 w-5 ${
                criticalHighCount > 10 ? "text-status-red" : "text-status-amber"
              }`} />
            </div>
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <h3 className="font-semibold">
                  {criticalHighCount} Critical/High Vulnerabilities Detected
                </h3>
                <Badge
                  variant="outline"
                  className={`text-xs ${
                    criticalHighCount > 10
                      ? "border-status-red/50 text-status-red"
                      : "border-status-amber/50 text-status-amber"
                  }`}
                >
                  {criticalHighCount > 10 ? "critical" : "warning"}
                </Badge>
              </div>
              <p className="mt-1 text-sm text-muted-foreground">
                {vulnerabilitiesData?.stats?.fixAvailable || 0} vulnerabilities have fixes available.
                AI can analyze impact and generate remediation plans.
              </p>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              {hasPendingSBOMTask ? (
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
                  disabled={isCreatingAITask}
                  className={criticalHighCount > 10 ? "bg-status-red hover:bg-status-red/90" : ""}
                >
                  {isCreatingAITask ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Creating...
                    </>
                  ) : (
                    <>
                      <Zap className="mr-2 h-4 w-4" />
                      Fix with AI
                    </>
                  )}
                </Button>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* SBOM Selector */}
      {sbomList && sbomList.sboms.length > 1 && (
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center gap-4">
              <Label htmlFor="sbom-select" className="text-sm font-medium">
                Viewing SBOM:
              </Label>
              <Select
                value={activeSBOMId}
                onValueChange={setSelectedSBOMId}
              >
                <SelectTrigger id="sbom-select" className="w-[300px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {sbomList.sboms.map((sbom) => (
                    <SelectItem key={sbom.id} value={sbom.id}>
                      {sbom.format.toUpperCase()} - {formatRelativeTime(sbom.generatedAt)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Tabs */}
      <Tabs defaultValue="components" className="space-y-4">
        <TabsList>
          <TabsTrigger value="components">
            Components ({totalComponents})
          </TabsTrigger>
          <TabsTrigger value="vulnerabilities">
            Vulnerabilities ({totalVulnerabilities})
          </TabsTrigger>
          <TabsTrigger value="licenses">
            Licenses ({licenseSummary?.licenses.length || 0})
          </TabsTrigger>
        </TabsList>

        {/* Components Tab */}
        <TabsContent value="components" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Software Components</CardTitle>
            </CardHeader>
            <CardContent>
              {components.length > 0 ? (
                <SBOMComponentsTable components={components} />
              ) : (
                <EmptyState
                  variant="data"
                  title="No components found"
                  description="Generate an SBOM to see software components."
                />
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Vulnerabilities Tab */}
        <TabsContent value="vulnerabilities" className="space-y-4">
          {/* Vulnerability Stats */}
          {vulnerabilitiesData && (
            <div className="grid gap-4 md:grid-cols-5">
              <Card>
                <CardContent className="p-4 text-center">
                  <div className="text-2xl font-bold text-status-red">
                    {vulnerabilitiesData.stats.critical}
                  </div>
                  <div className="text-xs text-muted-foreground">Critical</div>
                </CardContent>
              </Card>
              <Card>
                <CardContent className="p-4 text-center">
                  <div className="text-2xl font-bold text-status-amber">
                    {vulnerabilitiesData.stats.high}
                  </div>
                  <div className="text-xs text-muted-foreground">High</div>
                </CardContent>
              </Card>
              <Card>
                <CardContent className="p-4 text-center">
                  <div className="text-2xl font-bold text-status-amber">
                    {vulnerabilitiesData.stats.medium}
                  </div>
                  <div className="text-xs text-muted-foreground">Medium</div>
                </CardContent>
              </Card>
              <Card>
                <CardContent className="p-4 text-center">
                  <div className="text-2xl font-bold">
                    {vulnerabilitiesData.stats.low}
                  </div>
                  <div className="text-xs text-muted-foreground">Low</div>
                </CardContent>
              </Card>
              <Card>
                <CardContent className="p-4 text-center">
                  <div className="text-2xl font-bold text-status-green">
                    {vulnerabilitiesData.stats.fixAvailable}
                  </div>
                  <div className="text-xs text-muted-foreground">Fixes Available</div>
                </CardContent>
              </Card>
            </div>
          )}

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Vulnerability Details</CardTitle>
            </CardHeader>
            <CardContent>
              {vulnerabilities.length > 0 ? (
                <div className="space-y-4">
                  {vulnerabilities.map((vuln) => {
                    // Find the package for this vulnerability
                    const pkg = components.find(c => c.id === vuln.packageId);
                    return (
                      <SBOMVulnerabilityCard
                        key={vuln.id}
                        vulnerability={vuln}
                        packageName={pkg?.name}
                      />
                    );
                  })}
                </div>
              ) : (
                <EmptyState
                  variant="success"
                  title="No vulnerabilities found"
                  description="This SBOM has no known vulnerabilities. Keep it up to date!"
                />
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Licenses Tab */}
        <TabsContent value="licenses" className="space-y-4">
          {licenseSummary ? (
            <LicenseDistributionChart licenseSummary={licenseSummary} />
          ) : (
            <Card>
              <CardContent className="p-8">
                <EmptyState
                  variant="data"
                  title="No license data available"
                  description="Generate an SBOM to analyze license compliance."
                />
              </CardContent>
            </Card>
          )}
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
  if (diffHours < 24) return `${diffHours} hours ago`;

  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 7) return `${diffDays} days ago`;

  return date.toLocaleDateString();
}
