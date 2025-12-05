"use client";

import { useState, Fragment, useCallback, useMemo } from "react";
import { useRouter } from "next/navigation";
import { Card, CardContent } from "@/components/ui/card";
import { PaginationFooter } from "@/components/ui/pagination";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { StatusBadge } from "@/components/status/status-badge";
import { PlatformIcon } from "@/components/status/platform-icon";
import { MetricCard } from "@/components/data/metric-card";
import { PageSkeleton, ErrorState, EmptyState } from "@/components/feedback";
import { useImageFamilies, usePromoteImage, useDeprecateImage } from "@/hooks/use-images";
import { useSendAIMessage, useAIContext, usePendingTasks } from "@/hooks/use-ai";
import { ImageFamily } from "@/lib/api";
import {
  Search,
  Plus,
  MoreHorizontal,
  Download,
  Upload,
  Shield,
  CheckCircle,
  Clock,
  Archive,
  ExternalLink,
  Copy,
  Trash2,
  Box,
  Layers,
  GitBranch,
  Calendar,
  Loader2,
  Network,
  AlertTriangle,
  Sparkles,
  Zap,
} from "lucide-react";

type ImageStatus = "production" | "staging" | "testing" | "deprecated" | "pending";

const statusConfig: Record<ImageStatus, { label: string; variant: "success" | "warning" | "critical" | "info" }> = {
  production: { label: "Production", variant: "success" },
  staging: { label: "Staging", variant: "warning" },
  testing: { label: "Testing", variant: "info" },
  deprecated: { label: "Deprecated", variant: "critical" },
  pending: { label: "Pending", variant: "info" },
};

const PAGE_SIZE = 10;

export default function ImagesPage() {
  const router = useRouter();
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [expandedFamily, setExpandedFamily] = useState<string | null>(null);
  const [currentPage, setCurrentPage] = useState(1);
  const [isCreatingAITask, setIsCreatingAITask] = useState(false);
  const [optimizingImageId, setOptimizingImageId] = useState<string | null>(null);

  const { data: imageFamilies, isLoading, error, refetch } = useImageFamilies();
  const promoteImage = usePromoteImage();
  const deprecateImage = useDeprecateImage();

  // AI hooks
  const aiContext = useAIContext();
  const sendAIMessage = useSendAIMessage();
  const { data: pendingTasks = [] } = usePendingTasks();

  // Check if there's already a pending image-related task
  const hasPendingImageTask = pendingTasks.some(
    (task) => task.user_intent?.toLowerCase().includes("image")
  );

  // Must be called before any conditional returns to satisfy Rules of Hooks
  const handleCreateImage = useCallback(() => {
    router.push("/images/new");
  }, [router]);

  const families = useMemo(() => imageFamilies || [], [imageFamilies]);

  // Calculate metrics from real data
  const imageMetrics = useMemo(() => ({
    totalFamilies: families.length,
    activeVersions: families.reduce((acc, f) => acc + (f.versions?.length || 0), 0),
    pendingPromotions: families.filter((f) => f.status === "pending").length,
    deprecatedImages: families.filter((f) => f.status === "deprecated").length,
  }), [families]);

  const filteredFamilies = useMemo(() => {
    return families.filter((family) => {
      const matchesSearch =
        family.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        family.description.toLowerCase().includes(searchQuery.toLowerCase());
      const matchesStatus =
        statusFilter === "all" || family.status === statusFilter;
      return matchesSearch && matchesStatus;
    });
  }, [families, searchQuery, statusFilter]);

  // Paginate filtered results
  const totalPages = Math.ceil(filteredFamilies.length / PAGE_SIZE);
  const paginatedFamilies = useMemo(() => {
    const start = (currentPage - 1) * PAGE_SIZE;
    return filteredFamilies.slice(start, start + PAGE_SIZE);
  }, [filteredFamilies, currentPage]);

  if (isLoading) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Golden Images
            </h1>
            <p className="text-muted-foreground">
              Manage and track your golden image families and versions.
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
            Golden Images
          </h1>
          <p className="text-muted-foreground">
            Manage and track your golden image families and versions.
          </p>
        </div>
        <ErrorState
          error={error}
          retry={refetch}
          title="Failed to load images"
          description="We couldn't fetch the image families. Please try again."
        />
      </div>
    );
  }

  // Reset to page 1 when filters change
  const handleSearchChange = (value: string) => {
    setSearchQuery(value);
    setCurrentPage(1);
  };

  const handleStatusFilterChange = (value: string) => {
    setStatusFilter(value);
    setCurrentPage(1);
  };

  const handlePromote = (familyId: string, version: string) => {
    promoteImage.mutate({ familyId, version, targetStatus: "production" });
  };

  const handleDeprecate = (familyId: string, version: string) => {
    deprecateImage.mutate({ familyId, version });
  };

  // Handle AI optimization for all images
  const handleAIOptimization = async () => {
    setIsCreatingAITask(true);
    try {
      const deprecatedCount = imageMetrics.deprecatedImages;
      const pendingCount = imageMetrics.pendingPromotions;
      const intent = deprecatedCount > 0 || pendingCount > 0
        ? `Analyze ${imageMetrics.totalFamilies} image families. ${deprecatedCount} deprecated images need cleanup, ${pendingCount} pending promotions need review. Suggest lifecycle optimizations.`
        : `Review ${imageMetrics.totalFamilies} golden image families for security updates, optimization opportunities, and lifecycle improvements.`;

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

  // Handle AI optimization for a specific image family
  const handleImageAIOptimization = async (family: ImageFamily) => {
    setOptimizingImageId(family.id);
    try {
      const isDeprecated = family.status === "deprecated";
      const isPending = family.status === "pending";
      const intent = isDeprecated
        ? `Analyze deprecated image family "${family.name}" (v${family.latestVersion}) and generate a cleanup plan. Check for any assets still using this image and suggest migration steps.`
        : isPending
        ? `Review image family "${family.name}" (v${family.latestVersion}) pending promotion. Validate compliance, security, and readiness for production deployment.`
        : `Review image family "${family.name}" (v${family.latestVersion}) for optimization opportunities, compliance improvements, and update recommendations.`;

      await sendAIMessage.mutateAsync({
        message: intent,
        context: aiContext,
      });
      router.push("/ai");
    } catch (error) {
      console.error("Failed to create AI task:", error);
    } finally {
      setOptimizingImageId(null);
    }
  };

  const formatRelativeTime = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));

    if (diffHours < 1) return "just now";
    if (diffHours < 24) return `${diffHours} hours ago`;
    const diffDays = Math.floor(diffHours / 24);
    if (diffDays === 1) return "1 day ago";
    if (diffDays < 7) return `${diffDays} days ago`;
    return date.toLocaleDateString();
  };

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between animate-in fade-in-0 slide-in-from-bottom-2 duration-500">
        <div>
          <h1
            className="text-2xl font-bold tracking-tight text-foreground"
            style={{ fontFamily: "var(--font-display)" }}
          >
            Golden Images
          </h1>
          <p className="text-muted-foreground">
            Manage and track your golden image families and versions.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm">
            <Upload className="mr-2 h-4 w-4" />
            Import
          </Button>
          <Button variant="brand" size="sm">
            <Plus className="mr-2 h-4 w-4" />
            New Image
          </Button>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4 stagger-children">
        <MetricCard
          title="Image Families"
          value={imageMetrics.totalFamilies}
          subtitle="registered"
          status="neutral"
          icon={<Layers className="h-5 w-5" />}
        />
        <MetricCard
          title="Active Versions"
          value={imageMetrics.activeVersions}
          subtitle="in use"
          status="success"
          icon={<GitBranch className="h-5 w-5" />}
        />
        <MetricCard
          title="Pending Promotions"
          value={imageMetrics.pendingPromotions}
          subtitle="awaiting approval"
          status={imageMetrics.pendingPromotions > 0 ? "warning" : "success"}
          icon={<Clock className="h-5 w-5" />}
        />
        <MetricCard
          title="Deprecated"
          value={imageMetrics.deprecatedImages}
          subtitle="to be removed"
          status={imageMetrics.deprecatedImages > 0 ? "critical" : "success"}
          icon={<Archive className="h-5 w-5" />}
        />
      </div>

      {/* AI Insight Card */}
      {(imageMetrics.deprecatedImages > 0 || imageMetrics.pendingPromotions > 0 || imageMetrics.totalFamilies > 0) && (
        <Card className={`border-l-4 ${
          imageMetrics.deprecatedImages > 0
            ? "border-l-status-red bg-gradient-to-r from-status-red/5 to-transparent"
            : imageMetrics.pendingPromotions > 0
            ? "border-l-status-amber bg-gradient-to-r from-status-amber/5 to-transparent"
            : "border-l-brand-accent bg-gradient-to-r from-brand-accent/5 to-transparent"
        }`}>
          <CardContent className="flex items-start gap-4 p-6">
            <div className={`rounded-lg p-2 ${
              imageMetrics.deprecatedImages > 0 ? "bg-status-red/10" :
              imageMetrics.pendingPromotions > 0 ? "bg-status-amber/10" : "bg-brand-accent/10"
            }`}>
              <Sparkles className={`h-5 w-5 ${
                imageMetrics.deprecatedImages > 0 ? "text-status-red" :
                imageMetrics.pendingPromotions > 0 ? "text-status-amber" : "text-brand-accent"
              }`} />
            </div>
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <h3 className="font-semibold">
                  {imageMetrics.deprecatedImages > 0
                    ? `${imageMetrics.deprecatedImages} Deprecated Image${imageMetrics.deprecatedImages > 1 ? "s" : ""} Need Cleanup`
                    : imageMetrics.pendingPromotions > 0
                    ? `${imageMetrics.pendingPromotions} Image${imageMetrics.pendingPromotions > 1 ? "s" : ""} Pending Promotion`
                    : "Image Lifecycle Optimization Available"}
                </h3>
                <Badge
                  variant="outline"
                  className={`text-xs ${
                    imageMetrics.deprecatedImages > 0
                      ? "border-status-red/50 text-status-red"
                      : imageMetrics.pendingPromotions > 0
                      ? "border-status-amber/50 text-status-amber"
                      : ""
                  }`}
                >
                  {imageMetrics.deprecatedImages > 0 ? "action needed" : imageMetrics.pendingPromotions > 0 ? "review pending" : "optimization"}
                </Badge>
              </div>
              <p className="mt-1 text-sm text-muted-foreground">
                {imageMetrics.deprecatedImages > 0
                  ? "AI can analyze deprecated images and generate cleanup playbooks to remove unused resources."
                  : imageMetrics.pendingPromotions > 0
                  ? "AI can review pending promotions and validate readiness for production deployment."
                  : "AI can analyze your image portfolio for security updates and optimization opportunities."}
              </p>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              {hasPendingImageTask ? (
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
                  onClick={handleAIOptimization}
                  disabled={isCreatingAITask}
                  className={imageMetrics.deprecatedImages > 0 ? "bg-status-red hover:bg-status-red/90" : ""}
                >
                  {isCreatingAITask ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Creating...
                    </>
                  ) : (
                    <>
                      <Zap className="mr-2 h-4 w-4" />
                      Optimize with AI
                    </>
                  )}
                </Button>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Filters */}
      <Card variant="elevated" className="animate-in fade-in-0 slide-in-from-bottom-2 duration-500" style={{ animationDelay: '200ms', animationFillMode: 'backwards' }}>
        <CardContent className="flex items-center gap-4 p-4">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search images..."
              value={searchQuery}
              onChange={(e) => handleSearchChange(e.target.value)}
              className="pl-9"
            />
          </div>
          <Select value={statusFilter} onValueChange={handleStatusFilterChange}>
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="Status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Statuses</SelectItem>
              <SelectItem value="production">Production</SelectItem>
              <SelectItem value="staging">Staging</SelectItem>
              <SelectItem value="pending">Pending</SelectItem>
              <SelectItem value="deprecated">Deprecated</SelectItem>
            </SelectContent>
          </Select>
        </CardContent>
      </Card>

      {/* Image Families Table */}
      <Card variant="elevated" className="animate-in fade-in-0 slide-in-from-bottom-3 duration-700" style={{ animationDelay: '300ms', animationFillMode: 'backwards' }}>
        <CardContent className="p-0">
          {paginatedFamilies.length > 0 ? (
            <div className="rounded-lg border">
              <table className="w-full">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="px-4 py-3 text-left text-sm font-medium">
                      Image Family
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium">
                      Latest Version
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium">
                      Status
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium">
                      Platforms
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium">
                      Deployed
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium">
                      Compliance
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium">
                      Updated
                    </th>
                    <th className="px-4 py-3 text-right text-sm font-medium">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {paginatedFamilies.map((family, i) => {
                    const latestVersion = family.versions?.[0];
                    const config = statusConfig[family.status as ImageStatus] || statusConfig.pending;
                    const isExpanded = expandedFamily === family.id;

                    return (
                      <Fragment key={family.id}>
                        <tr
                          className={`cursor-pointer hover:bg-muted/50 ${
                            i !== paginatedFamilies.length - 1 && !isExpanded
                              ? "border-b"
                              : ""
                          }`}
                          onClick={() =>
                            setExpandedFamily(isExpanded ? null : family.id)
                          }
                        >
                          <td className="px-4 py-4">
                            <div className="flex items-center gap-3">
                              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
                                <Box className="h-5 w-5 text-muted-foreground" />
                              </div>
                              <div>
                                <div className="font-medium">{family.name}</div>
                                <div className="text-sm text-muted-foreground truncate max-w-[200px]">
                                  {family.description}
                                </div>
                              </div>
                            </div>
                          </td>
                          <td className="px-4 py-4">
                            <code className="rounded bg-muted px-2 py-1 text-sm">
                              v{family.latestVersion}
                            </code>
                          </td>
                          <td className="px-4 py-4">
                            <StatusBadge status={config.variant} size="sm">
                              {config.label}
                            </StatusBadge>
                          </td>
                          <td className="px-4 py-4">
                            <div className="flex items-center gap-1">
                              {latestVersion?.platforms
                                ?.slice(0, 3)
                                .map((platform) => (
                                  <PlatformIcon
                                    key={platform}
                                    platform={platform}
                                    size="sm"
                                  />
                                ))}
                              {latestVersion?.platforms && latestVersion.platforms.length > 3 && (
                                <Badge variant="secondary" className="text-xs">
                                  +{latestVersion.platforms.length - 3}
                                </Badge>
                              )}
                            </div>
                          </td>
                          <td className="px-4 py-4 text-sm">
                            {family.totalDeployed?.toLocaleString() || 0} assets
                          </td>
                          <td className="px-4 py-4">
                            <div className="flex items-center gap-2">
                              {latestVersion?.compliance?.cis && (
                                <Badge
                                  variant="outline"
                                  className="text-xs text-status-green border-status-green/30"
                                >
                                  CIS
                                </Badge>
                              )}
                              {latestVersion?.compliance?.slsaLevel !== undefined && (
                                <Badge
                                  variant="outline"
                                  className="text-xs text-brand-accent border-brand-accent/30"
                                >
                                  SLSA L{latestVersion.compliance.slsaLevel}
                                </Badge>
                              )}
                              {latestVersion?.compliance?.cosignSigned && (
                                <Shield className="h-4 w-4 text-status-green" />
                              )}
                            </div>
                          </td>
                          <td className="px-4 py-4 text-sm text-muted-foreground">
                            {formatRelativeTime(family.updatedAt)}
                          </td>
                          <td className="px-4 py-4 text-right">
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={(e) => e.stopPropagation()}
                                >
                                  {(promoteImage.isPending || deprecateImage.isPending) ? (
                                    <Loader2 className="h-4 w-4 animate-spin" />
                                  ) : (
                                    <MoreHorizontal className="h-4 w-4" />
                                  )}
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                <DropdownMenuItem
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    router.push(`/images/${family.id}`);
                                  }}
                                >
                                  <ExternalLink className="mr-2 h-4 w-4" />
                                  View Details
                                </DropdownMenuItem>
                                <DropdownMenuItem
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    router.push(`/images/${family.id}/lineage`);
                                  }}
                                >
                                  <Network className="mr-2 h-4 w-4" />
                                  View Lineage
                                </DropdownMenuItem>
                                <DropdownMenuItem
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    router.push(`/images/${family.id}/lineage?tab=vulnerabilities`);
                                  }}
                                >
                                  <AlertTriangle className="mr-2 h-4 w-4" />
                                  Vulnerabilities
                                </DropdownMenuItem>
                                <DropdownMenuSeparator />
                                <DropdownMenuItem
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    handleImageAIOptimization(family);
                                  }}
                                  disabled={optimizingImageId === family.id}
                                  className="text-brand-accent"
                                >
                                  {optimizingImageId === family.id ? (
                                    <>
                                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                      Creating...
                                    </>
                                  ) : (
                                    <>
                                      <Sparkles className="mr-2 h-4 w-4" />
                                      Optimize with AI
                                    </>
                                  )}
                                </DropdownMenuItem>
                                <DropdownMenuSeparator />
                                <DropdownMenuItem>
                                  <Copy className="mr-2 h-4 w-4" />
                                  Copy ID
                                </DropdownMenuItem>
                                <DropdownMenuItem>
                                  <Download className="mr-2 h-4 w-4" />
                                  Download
                                </DropdownMenuItem>
                                <DropdownMenuSeparator />
                                <DropdownMenuItem
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    handlePromote(family.id, family.latestVersion);
                                  }}
                                >
                                  <CheckCircle className="mr-2 h-4 w-4" />
                                  Promote to Production
                                </DropdownMenuItem>
                                <DropdownMenuItem
                                  className="text-status-red"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    handleDeprecate(family.id, family.latestVersion);
                                  }}
                                >
                                  <Trash2 className="mr-2 h-4 w-4" />
                                  Deprecate
                                </DropdownMenuItem>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </td>
                        </tr>
                        {isExpanded && family.versions && (
                          <tr className="border-b bg-muted/30">
                            <td colSpan={8} className="px-4 py-4">
                              <div className="ml-[52px] space-y-3">
                                <div className="text-sm font-medium">
                                  Version History
                                </div>
                                <div className="space-y-2">
                                  {family.versions.map((version) => (
                                    <div
                                      key={version.version}
                                      className="flex items-center gap-4 rounded-lg border bg-background p-3"
                                    >
                                      <code className="rounded bg-muted px-2 py-1 text-sm">
                                        v{version.version}
                                      </code>
                                      <StatusBadge
                                        status={statusConfig[version.status as ImageStatus]?.variant || "info"}
                                        size="sm"
                                      >
                                        {statusConfig[version.status as ImageStatus]?.label || version.status}
                                      </StatusBadge>
                                      <div className="flex items-center gap-1">
                                        {version.platforms?.map((platform) => (
                                          <PlatformIcon
                                            key={platform}
                                            platform={platform}
                                            size="sm"
                                          />
                                        ))}
                                      </div>
                                      <span className="text-sm text-muted-foreground">
                                        {version.deployedCount?.toLocaleString() || 0} deployed
                                      </span>
                                      <div className="flex items-center gap-1 text-sm text-muted-foreground">
                                        <Calendar className="h-3 w-3" />
                                        {new Date(version.createdAt).toLocaleDateString()}
                                      </div>
                                      <div className="ml-auto flex items-center gap-2">
                                        {version.compliance?.cis && (
                                          <Badge
                                            variant="outline"
                                            className="text-xs"
                                          >
                                            CIS
                                          </Badge>
                                        )}
                                        {version.compliance?.slsaLevel !== undefined && (
                                          <Badge
                                            variant="outline"
                                            className="text-xs"
                                          >
                                            SLSA L{version.compliance.slsaLevel}
                                          </Badge>
                                        )}
                                        {version.compliance?.cosignSigned && (
                                          <Shield className="h-4 w-4 text-status-green" />
                                        )}
                                      </div>
                                    </div>
                                  ))}
                                </div>
                              </div>
                            </td>
                          </tr>
                        )}
                      </Fragment>
                    );
                  })}
                </tbody>
              </table>
              {totalPages > 1 && (
                <PaginationFooter
                  currentPage={currentPage}
                  pageSize={PAGE_SIZE}
                  totalItems={filteredFamilies.length}
                  totalPages={totalPages}
                  onPageChange={setCurrentPage}
                />
              )}
            </div>
          ) : (
            <div className="p-8">
              <EmptyState
                variant="search"
                title="No images found"
                description={searchQuery || statusFilter !== "all"
                  ? "Try adjusting your search or filter criteria"
                  : "Get started by creating your first golden image"}
                action={searchQuery || statusFilter !== "all" ? undefined : {
                  label: "Create Image",
                  onClick: handleCreateImage,
                }}
              />
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
