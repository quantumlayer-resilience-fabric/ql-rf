"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { StatusBadge } from "@/components/status/status-badge";
import { MetricCard } from "@/components/data/metric-card";
import { PageSkeleton, ErrorState } from "@/components/feedback";
import {
  LineageTreeView,
  LineageGraph,
  LineagePath,
  VulnerabilitySummaryCard,
  VulnerabilityList,
  VulnerabilityTrendChart,
  BuildHistory,
} from "@/components/images";
import {
  useImageLineage,
  useImageLineageTree,
  useImageVulnerabilities,
  useImageBuilds,
  useImageDeployments,
  useImageComponents,
} from "@/hooks/use-lineage";
import {
  ArrowLeft,
  GitBranch,
  Shield,
  Box,
  Server,
  Package,
  Terminal,
  Clock,
  AlertTriangle,
  Network,
  List,
} from "lucide-react";

type ImageStatus = "production" | "staging" | "deprecated" | "pending";

const statusConfig: Record<ImageStatus, { label: string; variant: "success" | "warning" | "critical" | "info" }> = {
  production: { label: "Production", variant: "success" },
  staging: { label: "Staging", variant: "warning" },
  deprecated: { label: "Deprecated", variant: "critical" },
  pending: { label: "Pending", variant: "info" },
};

export default function ImageLineagePage() {
  const params = useParams();
  const router = useRouter();
  const imageId = params.id as string;

  const [activeTab, setActiveTab] = useState("overview");
  const [treeViewMode, setTreeViewMode] = useState<"tree" | "graph">("tree");

  // Fetch all lineage-related data
  const { data: lineage, isLoading: lineageLoading, error: lineageError, refetch: refetchLineage } = useImageLineage(imageId);
  const { data: vulnerabilities, isLoading: vulnsLoading } = useImageVulnerabilities(imageId);
  const { data: builds, isLoading: buildsLoading } = useImageBuilds(imageId);
  const { data: deployments, isLoading: deploymentsLoading } = useImageDeployments(imageId);
  const { data: components, isLoading: componentsLoading } = useImageComponents(imageId);

  // Fetch the lineage tree for the image's family
  const familyName = lineage?.image?.familyName;
  const { data: lineageTree } = useImageLineageTree(familyName || "");

  const isLoading = lineageLoading;

  if (isLoading) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="sm" onClick={() => router.back()}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Image Lineage
            </h1>
            <p className="text-muted-foreground">
              Loading image details...
            </p>
          </div>
        </div>
        <PageSkeleton metricCards={4} showChart={false} showTable={true} tableRows={5} />
      </div>
    );
  }

  if (lineageError || !lineage) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="sm" onClick={() => router.back()}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Image Lineage
            </h1>
          </div>
        </div>
        <ErrorState
          error={lineageError || new Error("Image not found")}
          retry={refetchLineage}
          title="Failed to load image lineage"
          description="We couldn't fetch the lineage data for this image."
        />
      </div>
    );
  }

  const image = lineage.image;
  const vulnSummary = lineage.vulnerabilitySummary;
  const config = statusConfig[image.status as ImageStatus] || statusConfig.pending;

  // Build parent/children for LineagePath component
  const parentImages = lineage.parents.map((p) => ({
    id: p.parentImageId,
    family: p.parentImage?.family || "",
    version: p.parentImage?.version || "",
    status: p.parentImage?.status || "pending",
  }));

  const childImages = lineage.children.map((c) => ({
    id: c.imageId,
    family: c.image?.family || "",
    version: c.image?.version || "",
    status: c.image?.status || "pending",
  }));

  const currentImage = {
    id: image.id,
    family: image.familyName,
    version: image.version,
    status: image.status,
  };

  const totalVulns = vulnSummary
    ? vulnSummary.criticalOpen + vulnSummary.highOpen + vulnSummary.mediumOpen + vulnSummary.lowOpen
    : 0;

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="sm" onClick={() => router.back()}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back
          </Button>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold tracking-tight text-foreground">
                {image.familyName}
              </h1>
              <code className="rounded bg-muted px-2 py-1 text-sm">
                v{image.version}
              </code>
              <StatusBadge status={config.variant}>
                {config.label}
              </StatusBadge>
            </div>
            <p className="text-muted-foreground mt-1">
              {image.description || `Golden image lineage and provenance details`}
            </p>
          </div>
        </div>
      </div>

      {/* Lineage Path */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="flex items-center gap-2 text-base">
            <GitBranch className="h-4 w-4" />
            Lineage Path
          </CardTitle>
        </CardHeader>
        <CardContent>
          <LineagePath
            parents={parentImages}
            current={currentImage}
            childImages={childImages}
            onSelectImage={(id) => router.push(`/images/${id}/lineage`)}
          />
        </CardContent>
      </Card>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Parent Images"
          value={lineage.parents.length}
          subtitle="upstream"
          status="neutral"
          icon={<GitBranch className="h-5 w-5" />}
        />
        <MetricCard
          title="Child Images"
          value={lineage.children.length}
          subtitle="downstream"
          status="neutral"
          icon={<GitBranch className="h-5 w-5 rotate-180" />}
        />
        <MetricCard
          title="Open Vulnerabilities"
          value={totalVulns}
          subtitle={vulnSummary?.criticalOpen ? `${vulnSummary.criticalOpen} critical` : "none critical"}
          status={vulnSummary?.criticalOpen ? "critical" : totalVulns > 0 ? "warning" : "success"}
          icon={<AlertTriangle className="h-5 w-5" />}
        />
        <MetricCard
          title="Active Deployments"
          value={lineage.activeDeployments}
          subtitle="running"
          status={lineage.activeDeployments > 0 ? "success" : "neutral"}
          icon={<Server className="h-5 w-5" />}
        />
      </div>

      {/* Tabbed Content */}
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="overview" className="flex items-center gap-2">
            <Box className="h-4 w-4" />
            Overview
          </TabsTrigger>
          <TabsTrigger value="vulnerabilities" className="flex items-center gap-2">
            <Shield className="h-4 w-4" />
            Vulnerabilities
            {totalVulns > 0 && (
              <Badge variant="destructive" className="ml-1 text-xs">
                {totalVulns}
              </Badge>
            )}
          </TabsTrigger>
          <TabsTrigger value="builds" className="flex items-center gap-2">
            <Terminal className="h-4 w-4" />
            Builds
          </TabsTrigger>
          <TabsTrigger value="deployments" className="flex items-center gap-2">
            <Server className="h-4 w-4" />
            Deployments
          </TabsTrigger>
          <TabsTrigger value="components" className="flex items-center gap-2">
            <Package className="h-4 w-4" />
            SBOM
          </TabsTrigger>
          {lineageTree && (
            <TabsTrigger value="tree" className="flex items-center gap-2">
              <GitBranch className="h-4 w-4" />
              Family Tree
            </TabsTrigger>
          )}
        </TabsList>

        <TabsContent value="overview" className="mt-6 space-y-6">
          {/* Vulnerability Trend Chart */}
          {vulnerabilities && vulnSummary && (
            <VulnerabilityTrendChart
              vulnerabilities={vulnerabilities}
              summary={vulnSummary}
            />
          )}

          <div className="grid gap-6 lg:grid-cols-2">
            {/* Vulnerability Summary */}
            {vulnSummary && (
              <VulnerabilitySummaryCard summary={vulnSummary} />
            )}

            {/* Promotion History */}
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="flex items-center gap-2 text-base">
                  <Clock className="h-4 w-4" />
                  Promotion History
                </CardTitle>
              </CardHeader>
              <CardContent>
                {lineage.promotions.length === 0 ? (
                  <div className="text-center py-6 text-muted-foreground">
                    No promotions recorded
                  </div>
                ) : (
                  <div className="space-y-2">
                    {lineage.promotions.slice(0, 5).map((promo) => (
                      <div
                        key={promo.id}
                        className="flex items-center justify-between p-2 rounded-lg bg-muted/50"
                      >
                        <div className="flex items-center gap-2">
                          <Badge variant="outline" className="text-xs capitalize">
                            {promo.fromStatus}
                          </Badge>
                          <span className="text-muted-foreground">→</span>
                          <Badge variant="secondary" className="text-xs capitalize">
                            {promo.toStatus}
                          </Badge>
                        </div>
                        <div className="text-xs text-muted-foreground">
                          {promo.promotedBy} • {new Date(promo.promotedAt).toLocaleDateString()}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
          </div>

          {/* Latest Build */}
          {builds && builds.length > 0 && (
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="flex items-center gap-2 text-base">
                  <Terminal className="h-4 w-4" />
                  Latest Build
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-4">
                    <Badge variant={builds[0].status === "success" ? "default" : "destructive"}>
                      Build #{builds[0].buildNumber}
                    </Badge>
                    <span className="text-sm text-muted-foreground">
                      {builds[0].builderType}
                    </span>
                    {builds[0].sourceCommit && (
                      <code className="text-xs bg-muted px-1.5 py-0.5 rounded">
                        {builds[0].sourceCommit.substring(0, 7)}
                      </code>
                    )}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {new Date(builds[0].createdAt).toLocaleString()}
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="vulnerabilities" className="mt-6">
          {vulnsLoading ? (
            <PageSkeleton metricCards={0} showChart={false} showTable={true} tableRows={5} />
          ) : vulnerabilities ? (
            <VulnerabilityList vulnerabilities={vulnerabilities} />
          ) : null}
        </TabsContent>

        <TabsContent value="builds" className="mt-6">
          {buildsLoading ? (
            <PageSkeleton metricCards={0} showChart={false} showTable={true} tableRows={3} />
          ) : builds ? (
            <BuildHistory builds={builds} />
          ) : null}
        </TabsContent>

        <TabsContent value="deployments" className="mt-6">
          {deploymentsLoading ? (
            <PageSkeleton metricCards={0} showChart={false} showTable={true} tableRows={5} />
          ) : (
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="flex items-center gap-2 text-base">
                  <Server className="h-4 w-4" />
                  Active Deployments ({deployments?.filter(d => d.status === "active").length || 0})
                </CardTitle>
              </CardHeader>
              <CardContent>
                {!deployments || deployments.length === 0 ? (
                  <div className="text-center py-12 text-muted-foreground">
                    <Server className="h-12 w-12 mx-auto mb-4 opacity-50" />
                    <p>No deployments found for this image</p>
                  </div>
                ) : (
                  <div className="space-y-2">
                    {deployments.map((deployment) => (
                      <div
                        key={deployment.id}
                        className="flex items-center justify-between p-3 rounded-lg border bg-muted/50"
                      >
                        <div className="flex items-center gap-3">
                          <Server className="h-4 w-4 text-muted-foreground" />
                          <div>
                            <div className="font-medium">{deployment.assetName}</div>
                            <div className="text-xs text-muted-foreground">
                              {deployment.platform} • {deployment.region}
                            </div>
                          </div>
                        </div>
                        <div className="flex items-center gap-3">
                          <Badge
                            variant={deployment.status === "active" ? "default" : "secondary"}
                            className="capitalize"
                          >
                            {deployment.status}
                          </Badge>
                          <div className="text-xs text-muted-foreground">
                            {new Date(deployment.deployedAt).toLocaleDateString()}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="components" className="mt-6">
          {componentsLoading ? (
            <PageSkeleton metricCards={0} showChart={false} showTable={true} tableRows={10} />
          ) : (
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="flex items-center gap-2 text-base">
                  <Package className="h-4 w-4" />
                  Software Bill of Materials ({components?.length || 0} components)
                </CardTitle>
              </CardHeader>
              <CardContent>
                {!components || components.length === 0 ? (
                  <div className="text-center py-12 text-muted-foreground">
                    <Package className="h-12 w-12 mx-auto mb-4 opacity-50" />
                    <p>No SBOM data available for this image</p>
                  </div>
                ) : (
                  <div className="rounded-lg border">
                    <table className="w-full">
                      <thead>
                        <tr className="border-b bg-muted/50">
                          <th className="px-4 py-2 text-left text-sm font-medium">Package</th>
                          <th className="px-4 py-2 text-left text-sm font-medium">Version</th>
                          <th className="px-4 py-2 text-left text-sm font-medium">Type</th>
                          <th className="px-4 py-2 text-left text-sm font-medium">License</th>
                        </tr>
                      </thead>
                      <tbody>
                        {components.slice(0, 20).map((component) => (
                          <tr key={component.id} className="border-b last:border-b-0">
                            <td className="px-4 py-2 text-sm font-medium">{component.name}</td>
                            <td className="px-4 py-2 text-sm">
                              <code className="bg-muted px-1.5 py-0.5 rounded text-xs">
                                {component.version}
                              </code>
                            </td>
                            <td className="px-4 py-2">
                              <Badge variant="outline" className="text-xs capitalize">
                                {component.componentType.replace("_", " ")}
                              </Badge>
                            </td>
                            <td className="px-4 py-2 text-sm text-muted-foreground">
                              {component.license || "Unknown"}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                    {components.length > 20 && (
                      <div className="p-3 text-center text-sm text-muted-foreground border-t">
                        Showing 20 of {components.length} components
                      </div>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {lineageTree && (
          <TabsContent value="tree" className="mt-6 space-y-4">
            {/* View Toggle */}
            <div className="flex items-center justify-end gap-2">
              <span className="text-sm text-muted-foreground">View:</span>
              <div className="flex items-center border rounded-md">
                <Button
                  variant={treeViewMode === "tree" ? "secondary" : "ghost"}
                  size="sm"
                  className="h-8 px-3 rounded-r-none"
                  onClick={() => setTreeViewMode("tree")}
                >
                  <List className="h-4 w-4 mr-1" />
                  Tree
                </Button>
                <Button
                  variant={treeViewMode === "graph" ? "secondary" : "ghost"}
                  size="sm"
                  className="h-8 px-3 rounded-l-none border-l"
                  onClick={() => setTreeViewMode("graph")}
                >
                  <Network className="h-4 w-4 mr-1" />
                  Graph
                </Button>
              </div>
            </div>

            {treeViewMode === "tree" ? (
              <LineageTreeView
                tree={lineageTree}
                onSelectImage={(id) => router.push(`/images/${id}/lineage`)}
                selectedImageId={imageId}
              />
            ) : (
              <LineageGraph
                tree={lineageTree}
                onSelectImage={(id) => router.push(`/images/${id}/lineage`)}
                selectedImageId={imageId}
              />
            )}
          </TabsContent>
        )}
      </Tabs>
    </div>
  );
}
