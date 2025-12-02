"use client";

import { useState } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { MetricCard } from "@/components/data/metric-card";
import { StatusBadge } from "@/components/status/status-badge";
import { PlatformIcon } from "@/components/status/platform-icon";
import { Logo } from "@/components/brand/logo";
import { Heatmap } from "@/components/charts/heatmap";
import {
  Server,
  TrendingDown,
  Shield,
  RefreshCw,
  AlertTriangle,
  Clock,
  X,
  ArrowRight,
  Play,
  Sparkles,
  ChevronRight,
} from "lucide-react";

// Demo data - showcasing the product capabilities
const demoMetrics = {
  fleetSize: { value: "12,847", trend: { direction: "up" as const, value: "+234", period: "24h" } },
  driftScore: { value: "94.2%", trend: { direction: "down" as const, value: "-2.1%", period: "7d" } },
  compliance: { value: "97.8%", trend: { direction: "neutral" as const, value: "stable", period: "" } },
  drReady: { value: "98.1%", trend: { direction: "up" as const, value: "+0.3%", period: "7d" } },
};

const demoPlatforms = [
  { platform: "aws" as const, count: 4231, percentage: 33 },
  { platform: "azure" as const, count: 3892, percentage: 30 },
  { platform: "gcp" as const, count: 2156, percentage: 17 },
  { platform: "vsphere" as const, count: 1834, percentage: 14 },
  { platform: "k8s" as const, count: 734, percentage: 6 },
];

const demoAlerts = [
  { severity: "critical" as const, count: 3, label: "Critical" },
  { severity: "warning" as const, count: 12, label: "Warning" },
  { severity: "info" as const, count: 847, label: "Info" },
];

const demoHeatmap = [
  { id: "eu-west-1", label: "eu-west-1", value: 98.2, status: "success" as const },
  { id: "eu-west-2", label: "eu-west-2", value: 96.1, status: "success" as const },
  { id: "us-east-1", label: "us-east-1", value: 87.3, status: "warning" as const },
  { id: "us-west-2", label: "us-west-2", value: 95.8, status: "success" as const },
  { id: "ap-south-1", label: "ap-south-1", value: 62.4, status: "critical" as const },
  { id: "ap-northeast-1", label: "ap-northeast-1", value: 94.5, status: "success" as const },
];

const tourSteps = [
  {
    title: "Real-Time Fleet Visibility",
    description: "See your entire infrastructure at a glance. Track 12,847+ assets across AWS, Azure, GCP, and on-premises.",
  },
  {
    title: "Drift Detection",
    description: "Instantly identify configuration drift. Our AI detects when servers deviate from golden images.",
  },
  {
    title: "Compliance Scoring",
    description: "Maintain 97.8% compliance with automated checks against CIS benchmarks and SLSA levels.",
  },
  {
    title: "DR Readiness",
    description: "Know your disaster recovery status at all times. 98.1% of assets are DR-ready.",
  },
];

export default function DemoPage() {
  const [currentTourStep, setCurrentTourStep] = useState(0);
  const [showTour, setShowTour] = useState(true);

  return (
    <div className="min-h-screen bg-background">
      {/* Demo Banner */}
      <div className="sticky top-0 z-50 border-b bg-brand-accent text-white">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3">
          <div className="flex items-center gap-4">
            <Badge variant="secondary" className="bg-white/20 text-white">
              <Play className="mr-1 h-3 w-3" />
              Demo Mode
            </Badge>
            <span className="text-sm">
              Explore with sample data. Connect your own cloud to see real insights.
            </span>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="secondary" size="sm" asChild>
              <Link href="/signup">
                Start Free Trial
                <ArrowRight className="ml-2 h-4 w-4" />
              </Link>
            </Button>
            <Button variant="ghost" size="sm" className="text-white hover:bg-white/10" asChild>
              <Link href="/">Exit Demo</Link>
            </Button>
          </div>
        </div>
      </div>

      {/* Tour Overlay */}
      {showTour && (
        <div className="fixed bottom-4 right-4 z-50 w-96">
          <Card className="border-brand-accent shadow-xl">
            <CardHeader className="flex flex-row items-start justify-between pb-2">
              <div>
                <Badge variant="outline" className="mb-2">
                  Step {currentTourStep + 1} of {tourSteps.length}
                </Badge>
                <CardTitle className="text-lg">
                  {tourSteps[currentTourStep].title}
                </CardTitle>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowTour(false)}
              >
                <X className="h-4 w-4" />
              </Button>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground mb-4">
                {tourSteps[currentTourStep].description}
              </p>
              <div className="flex items-center justify-between">
                <div className="flex gap-1">
                  {tourSteps.map((_, i) => (
                    <div
                      key={i}
                      className={`h-1.5 w-6 rounded-full ${
                        i === currentTourStep ? "bg-brand-accent" : "bg-muted"
                      }`}
                    />
                  ))}
                </div>
                <div className="flex gap-2">
                  {currentTourStep > 0 && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentTourStep(currentTourStep - 1)}
                    >
                      Back
                    </Button>
                  )}
                  {currentTourStep < tourSteps.length - 1 ? (
                    <Button
                      size="sm"
                      onClick={() => setCurrentTourStep(currentTourStep + 1)}
                    >
                      Next
                    </Button>
                  ) : (
                    <Button size="sm" onClick={() => setShowTour(false)}>
                      Got it!
                    </Button>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Demo Dashboard */}
      <div className="flex">
        {/* Sidebar */}
        <aside className="sticky top-[53px] h-[calc(100vh-53px)] w-64 border-r bg-card p-4">
          <div className="mb-6">
            <Logo variant="full" size="md" />
          </div>
          <nav className="space-y-1">
            {[
              { name: "Overview", active: true },
              { name: "Drift Analysis", active: false },
              { name: "Golden Images", active: false },
              { name: "Sites", active: false },
              { name: "Compliance", active: false },
              { name: "Settings", active: false },
            ].map((item) => (
              <div
                key={item.name}
                className={`rounded-lg px-3 py-2 text-sm ${
                  item.active
                    ? "bg-brand-accent/10 text-brand-accent font-medium"
                    : "text-muted-foreground hover:bg-muted"
                }`}
              >
                {item.name}
              </div>
            ))}
          </nav>
        </aside>

        {/* Main Content */}
        <main className="flex-1 p-6">
          <div className="space-y-6">
            {/* Header */}
            <div>
              <h1 className="text-2xl font-bold">Overview</h1>
              <p className="text-muted-foreground">
                Real-time visibility into your infrastructure health and compliance.
              </p>
            </div>

            {/* Key Metrics */}
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
              <MetricCard
                title="Fleet Size"
                value={demoMetrics.fleetSize.value}
                subtitle="assets"
                trend={demoMetrics.fleetSize.trend}
                status="success"
                icon={<Server className="h-5 w-5" />}
              />
              <MetricCard
                title="Drift Score"
                value={demoMetrics.driftScore.value}
                subtitle="coverage"
                trend={demoMetrics.driftScore.trend}
                status="success"
                icon={<TrendingDown className="h-5 w-5" />}
              />
              <MetricCard
                title="Compliance"
                value={demoMetrics.compliance.value}
                subtitle="passing"
                trend={demoMetrics.compliance.trend}
                status="success"
                icon={<Shield className="h-5 w-5" />}
              />
              <MetricCard
                title="DR Readiness"
                value={demoMetrics.drReady.value}
                subtitle="ready"
                trend={demoMetrics.drReady.trend}
                status="success"
                icon={<RefreshCw className="h-5 w-5" />}
              />
            </div>

            {/* AI Insight */}
            <Card className="border-brand-accent/30 bg-gradient-to-r from-brand-accent/5 to-transparent">
              <CardContent className="flex items-start gap-4 p-6">
                <div className="rounded-lg bg-brand-accent/10 p-2">
                  <Sparkles className="h-5 w-5 text-brand-accent" />
                </div>
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <h3 className="font-semibold">AI Insight: Drift Pattern Detected</h3>
                    <Badge variant="secondary" className="text-xs">94% confidence</Badge>
                  </div>
                  <p className="mt-1 text-sm text-muted-foreground">
                    47 assets in ap-south-1 are running outdated images due to a failed deployment 12 days ago.
                    Remediation: Re-trigger the deployment pipeline.
                  </p>
                </div>
                <Button size="sm">
                  View Plan
                  <ChevronRight className="ml-1 h-4 w-4" />
                </Button>
              </CardContent>
            </Card>

            {/* Main Grid */}
            <div className="grid gap-6 lg:grid-cols-3">
              {/* Platform Distribution */}
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Platform Distribution</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  {demoPlatforms.map((item) => (
                    <div key={item.platform} className="flex items-center gap-3">
                      <PlatformIcon platform={item.platform} size="sm" />
                      <div className="flex-1">
                        <div className="flex items-center justify-between text-sm">
                          <span className="font-medium capitalize">{item.platform}</span>
                          <span className="text-muted-foreground">{item.count.toLocaleString()}</span>
                        </div>
                        <div className="mt-1 h-2 rounded-full bg-muted">
                          <div
                            className="h-2 rounded-full bg-brand-accent"
                            style={{ width: `${item.percentage}%` }}
                          />
                        </div>
                      </div>
                    </div>
                  ))}
                </CardContent>
              </Card>

              {/* Active Alerts */}
              <Card>
                <CardHeader className="flex flex-row items-center justify-between">
                  <CardTitle className="text-base">Active Alerts</CardTitle>
                  <AlertTriangle className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent className="space-y-4">
                  {demoAlerts.map((alert) => (
                    <div
                      key={alert.label}
                      className="flex items-center justify-between rounded-lg border p-3"
                    >
                      <StatusBadge status={alert.severity} size="sm">
                        {alert.label}
                      </StatusBadge>
                      <span className="text-2xl font-bold">{alert.count}</span>
                    </div>
                  ))}
                </CardContent>
              </Card>

              {/* Recent Activity */}
              <Card>
                <CardHeader className="flex flex-row items-center justify-between">
                  <CardTitle className="text-base">Recent Activity</CardTitle>
                  <Clock className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="space-y-4">
                    {[
                      { action: "Image promoted", detail: "ql-base-linux v1.6.4", time: "5m ago", type: "info" },
                      { action: "Drift detected", detail: "ap-south-1 coverage dropped", time: "12m ago", type: "warning" },
                      { action: "DR drill completed", detail: "dc-london: RTO 4m", time: "1h ago", type: "success" },
                    ].map((activity, i) => (
                      <div key={i} className="flex items-start gap-3 text-sm">
                        <div
                          className={`mt-1.5 h-2 w-2 rounded-full ${
                            activity.type === "warning"
                              ? "bg-status-amber"
                              : activity.type === "success"
                              ? "bg-status-green"
                              : "bg-brand-accent"
                          }`}
                        />
                        <div className="flex-1">
                          <div className="font-medium">{activity.action}</div>
                          <div className="text-muted-foreground">{activity.detail}</div>
                        </div>
                        <span className="text-xs text-muted-foreground">{activity.time}</span>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Drift Heatmap */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Drift Heatmap by Site</CardTitle>
              </CardHeader>
              <CardContent>
                <Heatmap data={demoHeatmap} columns={6} />
              </CardContent>
            </Card>

            {/* CTA */}
            <Card className="bg-brand text-white">
              <CardContent className="flex items-center justify-between p-6">
                <div>
                  <h3 className="text-xl font-bold">Ready to see your own data?</h3>
                  <p className="text-white/80">
                    Connect your cloud providers and get real-time insights in minutes.
                  </p>
                </div>
                <Button variant="secondary" size="lg" asChild>
                  <Link href="/signup">
                    Start Free Trial
                    <ArrowRight className="ml-2 h-5 w-5" />
                  </Link>
                </Button>
              </CardContent>
            </Card>
          </div>
        </main>
      </div>
    </div>
  );
}
