"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { MetricCard } from "@/components/data/metric-card";
import { StatusBadge } from "@/components/status/status-badge";
import { PageSkeleton, ErrorState, EmptyState } from "@/components/feedback";
import { useComplianceSummary, useRunComplianceAudit } from "@/hooks/use-compliance";
import {
  Shield,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Download,
  RefreshCw,
  FileText,
  Clock,
  ChevronRight,
  Lock,
  Eye,
  Server,
  Loader2,
} from "lucide-react";

export default function CompliancePage() {
  const [selectedFramework, setSelectedFramework] = useState<string>("all");

  const { data: complianceData, isLoading, error, refetch } = useComplianceSummary();
  const runAudit = useRunComplianceAudit();

  const handleRunAudit = () => {
    runAudit.mutate(undefined, {
      onSuccess: () => {
        refetch();
      },
    });
  };

  if (isLoading) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Compliance
            </h1>
            <p className="text-muted-foreground">
              Monitor compliance posture across security frameworks and standards.
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
            Compliance
          </h1>
          <p className="text-muted-foreground">
            Monitor compliance posture across security frameworks and standards.
          </p>
        </div>
        <ErrorState
          error={error}
          retry={refetch}
          title="Failed to load compliance data"
          description="We couldn't fetch the compliance data. Please try again."
        />
      </div>
    );
  }

  // Extract data from API response with fallbacks
  const complianceMetrics = {
    overallScore: complianceData?.overallScore || 0,
    cisCompliance: complianceData?.cisCompliance || 0,
    slsaLevel: complianceData?.slsaLevel || 0,
    sigstoreVerified: complianceData?.sigstoreVerified || 0,
    lastAudit: complianceData?.lastAuditAt ? formatRelativeTime(complianceData.lastAuditAt) : "Never",
  };

  const frameworks = complianceData?.frameworks || [];
  const failingControls = complianceData?.failingControls || [];
  const imageCompliance = complianceData?.imageCompliance || [];

  // Filter failing controls by framework
  const filteredControls = selectedFramework === "all"
    ? failingControls
    : failingControls.filter((c) => c.framework.toLowerCase() === selectedFramework);

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Compliance
          </h1>
          <p className="text-muted-foreground">
            Monitor compliance posture across security frameworks and standards.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm">
            <Download className="mr-2 h-4 w-4" />
            Export Report
          </Button>
          <Button
            size="sm"
            onClick={handleRunAudit}
            disabled={runAudit.isPending}
          >
            {runAudit.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <RefreshCw className="mr-2 h-4 w-4" />
            )}
            Run Audit
          </Button>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Overall Score"
          value={`${complianceMetrics.overallScore.toFixed(1)}%`}
          subtitle="compliant"
          status={complianceMetrics.overallScore >= 95 ? "success" : complianceMetrics.overallScore >= 80 ? "warning" : "critical"}
          icon={<Shield className="h-5 w-5" />}
        />
        <MetricCard
          title="CIS Compliance"
          value={`${complianceMetrics.cisCompliance.toFixed(1)}%`}
          subtitle="benchmarks"
          status={complianceMetrics.cisCompliance >= 95 ? "success" : complianceMetrics.cisCompliance >= 80 ? "warning" : "critical"}
          icon={<CheckCircle className="h-5 w-5" />}
        />
        <MetricCard
          title="SLSA Level"
          value={`Level ${complianceMetrics.slsaLevel}`}
          subtitle="supply chain"
          status={complianceMetrics.slsaLevel >= 3 ? "success" : complianceMetrics.slsaLevel >= 2 ? "warning" : "critical"}
          icon={<Lock className="h-5 w-5" />}
        />
        <MetricCard
          title="Sigstore Verified"
          value={`${complianceMetrics.sigstoreVerified.toFixed(1)}%`}
          subtitle="images signed"
          status={complianceMetrics.sigstoreVerified >= 95 ? "success" : complianceMetrics.sigstoreVerified >= 80 ? "warning" : "critical"}
          icon={<FileText className="h-5 w-5" />}
        />
      </div>

      {/* Tabs */}
      <Tabs defaultValue="frameworks" className="space-y-4">
        <TabsList>
          <TabsTrigger value="frameworks">Frameworks</TabsTrigger>
          <TabsTrigger value="controls">Failing Controls</TabsTrigger>
          <TabsTrigger value="images">Image Compliance</TabsTrigger>
        </TabsList>

        {/* Frameworks Tab */}
        <TabsContent value="frameworks" className="space-y-4">
          {frameworks.length > 0 ? (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {frameworks.map((framework) => (
                <Card key={framework.id} className="cursor-pointer hover:border-brand-accent">
                  <CardHeader className="pb-2">
                    <div className="flex items-start justify-between">
                      <div>
                        <CardTitle className="text-base flex items-center gap-2">
                          {framework.name}
                          {framework.level && (
                            <Badge variant="secondary" className="text-xs">
                              L{framework.level}
                            </Badge>
                          )}
                        </CardTitle>
                        <p className="text-xs text-muted-foreground mt-1">
                          {framework.description}
                        </p>
                      </div>
                      <StatusBadge
                        status={framework.status === "passing" ? "success" : framework.status === "warning" ? "warning" : "critical"}
                        size="sm"
                      >
                        {framework.status}
                      </StatusBadge>
                    </div>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-3">
                      <div className="flex items-center justify-between">
                        <span className="text-3xl font-bold">{framework.score.toFixed(1)}%</span>
                        <span className="text-sm text-muted-foreground">
                          {framework.passingControls}/{framework.totalControls} controls
                        </span>
                      </div>
                      <Progress
                        value={framework.score}
                        className="h-2"
                      />
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : (
            <Card>
              <CardContent className="p-8">
                <EmptyState
                  variant="data"
                  title="No frameworks configured"
                  description="Configure compliance frameworks to start monitoring your security posture."
                />
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {/* Failing Controls Tab */}
        <TabsContent value="controls" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-base">
                  Failing Controls ({filteredControls.length})
                </CardTitle>
                <Select value={selectedFramework} onValueChange={setSelectedFramework}>
                  <SelectTrigger className="w-[180px]">
                    <SelectValue placeholder="Framework" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Frameworks</SelectItem>
                    <SelectItem value="cis">CIS Benchmarks</SelectItem>
                    <SelectItem value="slsa">SLSA</SelectItem>
                    <SelectItem value="soc2">SOC 2</SelectItem>
                    <SelectItem value="hipaa">HIPAA</SelectItem>
                    <SelectItem value="pci">PCI DSS</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </CardHeader>
            <CardContent>
              {filteredControls.length > 0 ? (
                <div className="space-y-4">
                  {filteredControls.map((control) => (
                    <div
                      key={control.id}
                      className="flex items-start gap-4 rounded-lg border p-4"
                    >
                      <div
                        className={`rounded-full p-2 ${
                          control.severity === "high"
                            ? "bg-status-red/10"
                            : control.severity === "medium"
                            ? "bg-status-amber/10"
                            : "bg-muted"
                        }`}
                      >
                        {control.severity === "high" ? (
                          <XCircle className="h-5 w-5 text-status-red" />
                        ) : control.severity === "medium" ? (
                          <AlertTriangle className="h-5 w-5 text-status-amber" />
                        ) : (
                          <AlertTriangle className="h-5 w-5 text-muted-foreground" />
                        )}
                      </div>
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <code className="rounded bg-muted px-2 py-0.5 text-xs">
                            {control.id}
                          </code>
                          <Badge variant="outline" className="text-xs">
                            {control.framework}
                          </Badge>
                          <StatusBadge
                            status={
                              control.severity === "high"
                                ? "critical"
                                : control.severity === "medium"
                                ? "warning"
                                : "neutral"
                            }
                            size="sm"
                          >
                            {control.severity}
                          </StatusBadge>
                        </div>
                        <h4 className="mt-1 font-medium">{control.title}</h4>
                        <p className="mt-1 text-sm text-muted-foreground">
                          {control.recommendation}
                        </p>
                        <div className="mt-2 flex items-center gap-4 text-sm">
                          <span className="flex items-center gap-1 text-muted-foreground">
                            <Server className="h-3 w-3" />
                            {control.affectedAssets} affected assets
                          </span>
                        </div>
                      </div>
                      <Button variant="outline" size="sm">
                        Remediate
                        <ChevronRight className="ml-1 h-4 w-4" />
                      </Button>
                    </div>
                  ))}
                </div>
              ) : (
                <EmptyState
                  variant="success"
                  title="All controls passing"
                  description="No failing controls found. Your infrastructure is fully compliant."
                />
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Image Compliance Tab */}
        <TabsContent value="images" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Golden Image Compliance</CardTitle>
            </CardHeader>
            <CardContent>
              {imageCompliance.length > 0 ? (
                <div className="rounded-lg border">
                  <table className="w-full">
                    <thead>
                      <tr className="border-b bg-muted/50">
                        <th className="px-4 py-3 text-left text-sm font-medium">Image</th>
                        <th className="px-4 py-3 text-left text-sm font-medium">CIS</th>
                        <th className="px-4 py-3 text-left text-sm font-medium">SLSA</th>
                        <th className="px-4 py-3 text-left text-sm font-medium">Cosign</th>
                        <th className="px-4 py-3 text-left text-sm font-medium">Last Scan</th>
                        <th className="px-4 py-3 text-left text-sm font-medium">Issues</th>
                      </tr>
                    </thead>
                    <tbody>
                      {imageCompliance.map((image, i) => (
                        <tr
                          key={image.familyId}
                          className={i !== imageCompliance.length - 1 ? "border-b" : ""}
                        >
                          <td className="px-4 py-3">
                            <div>
                              <div className="font-medium">{image.familyName}</div>
                              <div className="text-xs text-muted-foreground">
                                v{image.version}
                              </div>
                            </div>
                          </td>
                          <td className="px-4 py-3">
                            {image.cis ? (
                              <CheckCircle className="h-5 w-5 text-status-green" />
                            ) : (
                              <XCircle className="h-5 w-5 text-status-red" />
                            )}
                          </td>
                          <td className="px-4 py-3">
                            <Badge
                              variant="outline"
                              className={
                                image.slsaLevel >= 3
                                  ? "text-status-green border-status-green/30"
                                  : "text-status-amber border-status-amber/30"
                              }
                            >
                              Level {image.slsaLevel}
                            </Badge>
                          </td>
                          <td className="px-4 py-3">
                            {image.cosignSigned ? (
                              <CheckCircle className="h-5 w-5 text-status-green" />
                            ) : (
                              <XCircle className="h-5 w-5 text-status-red" />
                            )}
                          </td>
                          <td className="px-4 py-3 text-sm text-muted-foreground">
                            <div className="flex items-center gap-1">
                              <Clock className="h-3 w-3" />
                              {formatRelativeTime(image.lastScanAt)}
                            </div>
                          </td>
                          <td className="px-4 py-3">
                            {image.issueCount === 0 ? (
                              <Badge variant="secondary" className="bg-status-green/10 text-status-green">
                                Clean
                              </Badge>
                            ) : (
                              <Badge variant="secondary" className="bg-status-amber/10 text-status-amber">
                                {image.issueCount} issues
                              </Badge>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <EmptyState
                  variant="data"
                  title="No image compliance data"
                  description="Run a compliance scan to see image compliance status."
                />
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Last Audit Info */}
      <Card>
        <CardContent className="flex items-center justify-between p-4">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-muted p-2">
              <Eye className="h-5 w-5 text-muted-foreground" />
            </div>
            <div>
              <p className="font-medium">Last Compliance Audit</p>
              <p className="text-sm text-muted-foreground">
                Completed {complianceMetrics.lastAudit} • All frameworks scanned
              </p>
            </div>
          </div>
          <Button variant="outline" size="sm">
            View Audit Log
          </Button>
        </CardContent>
      </Card>
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
