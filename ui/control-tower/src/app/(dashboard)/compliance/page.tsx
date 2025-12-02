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
import { PlatformIcon } from "@/components/status/platform-icon";
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
} from "lucide-react";

// Mock compliance data
const complianceMetrics = {
  overallScore: 97.8,
  cisCompliance: 96.2,
  slsaLevel: 3,
  sigstoreVerified: 98.5,
  lastAudit: "2 hours ago",
};

const frameworks = [
  {
    id: "cis",
    name: "CIS Benchmarks",
    description: "Center for Internet Security hardening standards",
    score: 96.2,
    passingControls: 142,
    totalControls: 148,
    status: "passing",
  },
  {
    id: "slsa",
    name: "SLSA Framework",
    description: "Supply-chain Levels for Software Artifacts",
    score: 100,
    passingControls: 24,
    totalControls: 24,
    status: "passing",
    level: 3,
  },
  {
    id: "soc2",
    name: "SOC 2 Type II",
    description: "Service Organization Control standards",
    score: 94.5,
    passingControls: 86,
    totalControls: 91,
    status: "passing",
  },
  {
    id: "hipaa",
    name: "HIPAA",
    description: "Health Insurance Portability and Accountability Act",
    score: 89.2,
    passingControls: 41,
    totalControls: 46,
    status: "warning",
  },
  {
    id: "pci",
    name: "PCI DSS",
    description: "Payment Card Industry Data Security Standard",
    score: 92.1,
    passingControls: 234,
    totalControls: 254,
    status: "passing",
  },
];

const failingControls = [
  {
    id: "CIS-4.2.1",
    framework: "CIS",
    title: "Ensure SSH Protocol is set to 2",
    severity: "high",
    affectedAssets: 12,
    recommendation: "Update SSH configuration to enforce protocol version 2",
  },
  {
    id: "CIS-5.1.3",
    framework: "CIS",
    title: "Ensure permissions on /etc/cron.d are configured",
    severity: "medium",
    affectedAssets: 8,
    recommendation: "Run chmod 700 /etc/cron.d on affected instances",
  },
  {
    id: "HIPAA-164.312(a)",
    framework: "HIPAA",
    title: "Access Control - Unique User Identification",
    severity: "high",
    affectedAssets: 5,
    recommendation: "Implement unique user IDs for all system access",
  },
  {
    id: "CIS-1.4.2",
    framework: "CIS",
    title: "Ensure bootloader password is set",
    severity: "medium",
    affectedAssets: 23,
    recommendation: "Configure GRUB bootloader password",
  },
  {
    id: "SOC2-CC6.1",
    framework: "SOC 2",
    title: "Logical Access Controls",
    severity: "low",
    affectedAssets: 3,
    recommendation: "Review and update access control policies",
  },
];

const imageCompliance = [
  {
    family: "ql-base-linux",
    version: "1.6.4",
    cis: true,
    slsa: 3,
    cosign: true,
    lastScan: "1 hour ago",
    issues: 0,
  },
  {
    family: "ql-database",
    version: "2.3.1",
    cis: true,
    slsa: 3,
    cosign: true,
    lastScan: "2 hours ago",
    issues: 0,
  },
  {
    family: "ql-worker",
    version: "1.3.2",
    cis: true,
    slsa: 2,
    cosign: true,
    lastScan: "3 hours ago",
    issues: 2,
  },
  {
    family: "ql-cache",
    version: "3.0.1",
    cis: false,
    slsa: 3,
    cosign: true,
    lastScan: "1 hour ago",
    issues: 3,
  },
  {
    family: "ql-web",
    version: "2.0.0",
    cis: true,
    slsa: 3,
    cosign: true,
    lastScan: "30 min ago",
    issues: 0,
  },
];

export default function CompliancePage() {
  const [selectedFramework, setSelectedFramework] = useState<string>("all");

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
          <Button size="sm">
            <RefreshCw className="mr-2 h-4 w-4" />
            Run Audit
          </Button>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Overall Score"
          value={`${complianceMetrics.overallScore}%`}
          subtitle="compliant"
          status="success"
          icon={<Shield className="h-5 w-5" />}
        />
        <MetricCard
          title="CIS Compliance"
          value={`${complianceMetrics.cisCompliance}%`}
          subtitle="benchmarks"
          status="success"
          icon={<CheckCircle className="h-5 w-5" />}
        />
        <MetricCard
          title="SLSA Level"
          value={`Level ${complianceMetrics.slsaLevel}`}
          subtitle="supply chain"
          status="success"
          icon={<Lock className="h-5 w-5" />}
        />
        <MetricCard
          title="Sigstore Verified"
          value={`${complianceMetrics.sigstoreVerified}%`}
          subtitle="images signed"
          status="success"
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
                      status={framework.status === "passing" ? "success" : "warning"}
                      size="sm"
                    >
                      {framework.status}
                    </StatusBadge>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    <div className="flex items-center justify-between">
                      <span className="text-3xl font-bold">{framework.score}%</span>
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
        </TabsContent>

        {/* Failing Controls Tab */}
        <TabsContent value="controls" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-base">
                  Failing Controls ({failingControls.length})
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
              <div className="space-y-4">
                {failingControls.map((control) => (
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
                        key={image.family}
                        className={i !== imageCompliance.length - 1 ? "border-b" : ""}
                      >
                        <td className="px-4 py-3">
                          <div>
                            <div className="font-medium">{image.family}</div>
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
                              image.slsa >= 3
                                ? "text-status-green border-status-green/30"
                                : "text-status-amber border-status-amber/30"
                            }
                          >
                            Level {image.slsa}
                          </Badge>
                        </td>
                        <td className="px-4 py-3">
                          {image.cosign ? (
                            <CheckCircle className="h-5 w-5 text-status-green" />
                          ) : (
                            <XCircle className="h-5 w-5 text-status-red" />
                          )}
                        </td>
                        <td className="px-4 py-3 text-sm text-muted-foreground">
                          <div className="flex items-center gap-1">
                            <Clock className="h-3 w-3" />
                            {image.lastScan}
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          {image.issues === 0 ? (
                            <Badge variant="secondary" className="bg-status-green/10 text-status-green">
                              Clean
                            </Badge>
                          ) : (
                            <Badge variant="secondary" className="bg-status-amber/10 text-status-amber">
                              {image.issues} issues
                            </Badge>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
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
