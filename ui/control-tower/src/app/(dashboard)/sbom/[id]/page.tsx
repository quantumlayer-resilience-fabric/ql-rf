"use client";

import { use } from "react";
import { useRouter } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { MetricCard } from "@/components/data/metric-card";
import { PageSkeleton, ErrorState } from "@/components/feedback";
import { SBOMComponentsTable } from "@/components/sbom/sbom-components-table";
import { SBOMVulnerabilityCard } from "@/components/sbom/sbom-vulnerability-card";
import { useSBOM, useSBOMVulnerabilities, useExportSBOM } from "@/hooks/use-sbom";
import {
  ArrowLeft,
  Download,
  FileText,
  Package,
  Shield,
  Clock,
  Loader2,
} from "lucide-react";

interface SBOMDetailPageProps {
  params: Promise<{ id: string }>;
}

export default function SBOMDetailPage({ params }: SBOMDetailPageProps) {
  const { id } = use(params);
  const router = useRouter();

  // Fetch SBOM with packages and vulnerabilities
  const {
    data: sbom,
    isLoading,
    error,
    refetch,
  } = useSBOM(id, {
    includePackages: true,
    includeVulns: true,
  });

  const { data: vulnerabilitiesData } = useSBOMVulnerabilities(id, {});
  const exportSBOM = useExportSBOM();

  const handleExport = async (format?: "spdx" | "cyclonedx") => {
    try {
      const result = await exportSBOM.mutateAsync({ id, format });

      // Download the exported SBOM
      const blob = new Blob([JSON.stringify(result.content, null, 2)], {
        type: "application/json",
      });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `sbom-${id}-${format || sbom?.format}.json`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      console.error("Failed to export SBOM:", error);
    }
  };

  if (isLoading) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              SBOM Details
            </h1>
            <p className="text-muted-foreground">
              View detailed information about this software bill of materials.
            </p>
          </div>
        </div>
        <PageSkeleton metricCards={3} showChart={false} showTable={true} tableRows={5} />
      </div>
    );
  }

  if (error || !sbom) {
    return (
      <div className="page-transition space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            SBOM Details
          </h1>
          <p className="text-muted-foreground">
            View detailed information about this software bill of materials.
          </p>
        </div>
        <ErrorState
          error={error || new Error("SBOM not found")}
          retry={refetch}
          title="Failed to load SBOM"
          description="We couldn't fetch the SBOM details. Please try again."
        />
      </div>
    );
  }

  const components = sbom.packages || [];
  const vulnerabilities = sbom.vulnerabilities || [];
  const criticalHighCount = (vulnerabilitiesData?.stats?.critical || 0) +
                           (vulnerabilitiesData?.stats?.high || 0);

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between">
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => router.push("/sbom")}
            >
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              SBOM Details
            </h1>
          </div>
          <p className="text-muted-foreground">
            View detailed information about this software bill of materials.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleExport()}
            disabled={exportSBOM.isPending}
          >
            {exportSBOM.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Download className="mr-2 h-4 w-4" />
            )}
            Export {sbom.format.toUpperCase()}
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleExport(sbom.format === "spdx" ? "cyclonedx" : "spdx")}
            disabled={exportSBOM.isPending}
          >
            {exportSBOM.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Download className="mr-2 h-4 w-4" />
            )}
            Export {sbom.format === "spdx" ? "CycloneDX" : "SPDX"}
          </Button>
        </div>
      </div>

      {/* SBOM Metadata */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">SBOM Information</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <div>
              <div className="text-sm font-medium text-muted-foreground">Format</div>
              <Badge variant="outline" className="mt-1">
                {sbom.format.toUpperCase()}
              </Badge>
            </div>
            <div>
              <div className="text-sm font-medium text-muted-foreground">Version</div>
              <div className="mt-1 text-sm">{sbom.version}</div>
            </div>
            <div>
              <div className="text-sm font-medium text-muted-foreground">Scanner</div>
              <div className="mt-1 text-sm">{sbom.scanner || "Unknown"}</div>
            </div>
            <div>
              <div className="text-sm font-medium text-muted-foreground">Generated</div>
              <div className="mt-1 text-sm">
                {new Date(sbom.generatedAt).toLocaleDateString()} at{" "}
                {new Date(sbom.generatedAt).toLocaleTimeString()}
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-3">
        <MetricCard
          title="Total Components"
          value={sbom.packageCount.toString()}
          subtitle="software packages"
          status="neutral"
          icon={<Package className="h-5 w-5" />}
        />
        <MetricCard
          title="Vulnerabilities"
          value={`${criticalHighCount}`}
          subtitle={`critical/high of ${sbom.vulnCount} total`}
          status={criticalHighCount > 10 ? "critical" : criticalHighCount > 5 ? "warning" : "success"}
          icon={<Shield className="h-5 w-5" />}
        />
        <MetricCard
          title="Last Updated"
          value={formatRelativeTime(sbom.updatedAt)}
          subtitle="SBOM metadata"
          status="neutral"
          icon={<Clock className="h-5 w-5" />}
        />
      </div>

      {/* Tabs */}
      <Tabs defaultValue="components" className="space-y-4">
        <TabsList>
          <TabsTrigger value="components">
            Components ({components.length})
          </TabsTrigger>
          <TabsTrigger value="vulnerabilities">
            Vulnerabilities ({vulnerabilities.length})
          </TabsTrigger>
          <TabsTrigger value="raw">Raw SBOM</TabsTrigger>
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
                <div className="py-8 text-center text-muted-foreground">
                  No components found in this SBOM
                </div>
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
                <div className="py-8 text-center">
                  <Shield className="mx-auto h-12 w-12 text-status-green" />
                  <p className="mt-2 font-medium">No vulnerabilities found</p>
                  <p className="text-sm text-muted-foreground">
                    This SBOM has no known vulnerabilities
                  </p>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Raw SBOM Tab */}
        <TabsContent value="raw" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-base">Raw SBOM Data</CardTitle>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    const json = JSON.stringify(sbom.content, null, 2);
                    navigator.clipboard.writeText(json);
                  }}
                >
                  <FileText className="mr-2 h-4 w-4" />
                  Copy JSON
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {sbom.content ? (
                <pre className="max-h-[600px] overflow-auto rounded-lg bg-muted p-4 text-xs">
                  {JSON.stringify(sbom.content, null, 2)}
                </pre>
              ) : (
                <div className="py-8 text-center text-muted-foreground">
                  Raw SBOM content not available
                </div>
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
  if (diffHours < 24) return `${diffHours} hours ago`;

  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 7) return `${diffDays} days ago`;

  return date.toLocaleDateString();
}
