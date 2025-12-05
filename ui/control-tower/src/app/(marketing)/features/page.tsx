"use client";

import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import Link from "next/link";
import {
  Globe,
  Sparkles,
  Shield,
  TrendingDown,
  RefreshCw,
  Eye,
  Bell,
  Lock,
  CheckCircle,
  ArrowRight,
  Cloud,
  Database,
  GitBranch,
  Layers,
} from "lucide-react";

const coreFeatures = [
  {
    icon: Globe,
    title: "Multi-Cloud Visibility",
    description: "Unified view across AWS, Azure, GCP, vSphere, and Kubernetes. One dashboard to rule them all.",
    highlights: [
      "Auto-discovery of assets across all platforms",
      "Real-time inventory sync every 5 minutes",
      "Cross-cloud dependency mapping",
      "Unified tagging and metadata",
    ],
  },
  {
    icon: TrendingDown,
    title: "Drift Detection",
    description: "Instantly identify when servers deviate from golden images. Never miss a configuration drift again.",
    highlights: [
      "Real-time drift monitoring",
      "Image version comparison",
      "Automated alerting on drift",
      "Historical drift analysis",
    ],
  },
  {
    icon: Shield,
    title: "Compliance Automation",
    description: "Continuous compliance monitoring against CIS, SOC 2, HIPAA, PCI DSS, and custom frameworks.",
    highlights: [
      "Pre-built compliance templates",
      "Automated evidence collection",
      "One-click remediation",
      "Audit-ready reports",
    ],
  },
  {
    icon: RefreshCw,
    title: "DR Readiness",
    description: "Know your disaster recovery status at all times. Automated DR drills and RTO/RPO tracking.",
    highlights: [
      "DR pair management",
      "Automated failover testing",
      "RTO/RPO monitoring",
      "Replication lag alerts",
    ],
  },
  {
    icon: Sparkles,
    title: "AI-Powered Insights",
    description: "Let AI analyze your infrastructure and surface actionable recommendations automatically.",
    highlights: [
      "Root cause analysis",
      "Predictive drift detection",
      "Cost optimization suggestions",
      "Natural language queries",
    ],
  },
  {
    icon: Layers,
    title: "Golden Image Management",
    description: "Full lifecycle management for your golden images. Build, version, promote, and deprecate with confidence.",
    highlights: [
      "Image family organization",
      "Version history tracking",
      "SLSA compliance verification",
      "Cosign signature validation",
    ],
  },
];

const additionalFeatures = [
  { icon: Bell, title: "Smart Alerting", description: "Intelligent alerts with noise reduction" },
  { icon: Lock, title: "RBAC", description: "Fine-grained role-based access control" },
  { icon: GitBranch, title: "CI/CD Integration", description: "Native integration with pipelines" },
  { icon: Database, title: "API Access", description: "Full REST API for automation" },
  { icon: Cloud, title: "SSO Support", description: "SAML, OIDC, and OAuth support" },
  { icon: Eye, title: "Audit Logging", description: "Complete audit trail of all actions" },
];

export default function FeaturesPage() {
  return (
    <div className="py-20">
      {/* Hero */}
      <section className="mx-auto max-w-7xl px-4 text-center">
        <Badge variant="outline" className="mb-4">
          <Sparkles className="mr-1 h-3 w-3" />
          Features
        </Badge>
        <h1 className="text-4xl font-bold tracking-tight md:text-5xl">
          Everything you need to manage
          <br />
          <span className="text-brand-accent">infrastructure at scale</span>
        </h1>
        <p className="mx-auto mt-4 max-w-2xl text-lg text-muted-foreground">
          From drift detection to compliance automation, QL-RF gives you complete
          visibility and control over your multi-cloud infrastructure.
        </p>
      </section>

      {/* Core Features */}
      <section className="mx-auto mt-20 max-w-7xl px-4">
        <div className="grid gap-8 md:grid-cols-2 lg:grid-cols-3">
          {coreFeatures.map((feature) => (
            <Card key={feature.title} className="overflow-hidden">
              <CardContent className="p-6">
                <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-brand-accent/10">
                  <feature.icon className="h-6 w-6 text-brand-accent" />
                </div>
                <h3 className="mt-4 text-xl font-semibold">{feature.title}</h3>
                <p className="mt-2 text-muted-foreground">{feature.description}</p>
                <ul className="mt-4 space-y-2">
                  {feature.highlights.map((highlight) => (
                    <li key={highlight} className="flex items-center gap-2 text-sm">
                      <CheckCircle className="h-4 w-4 text-status-green" />
                      {highlight}
                    </li>
                  ))}
                </ul>
              </CardContent>
            </Card>
          ))}
        </div>
      </section>

      {/* Additional Features */}
      <section className="mx-auto mt-20 max-w-7xl px-4">
        <h2 className="text-center text-2xl font-bold">And much more...</h2>
        <div className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {additionalFeatures.map((feature) => (
            <div
              key={feature.title}
              className="flex items-center gap-4 rounded-lg border p-4"
            >
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
                <feature.icon className="h-5 w-5 text-muted-foreground" />
              </div>
              <div>
                <h4 className="font-medium">{feature.title}</h4>
                <p className="text-sm text-muted-foreground">{feature.description}</p>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* CTA */}
      <section className="mx-auto mt-20 max-w-7xl px-4">
        <Card className="bg-brand text-white">
          <CardContent className="flex flex-col items-center justify-between gap-6 p-8 md:flex-row">
            <div>
              <h3 className="text-2xl font-bold">Ready to see it in action?</h3>
              <p className="mt-2 text-white/80">
                Start your free trial today. No credit card required.
              </p>
            </div>
            <div className="flex gap-3">
              <Button variant="secondary" size="lg" asChild>
                <Link href="/demo">
                  Try Demo
                </Link>
              </Button>
              <Button variant="outline" size="lg" className="border-white text-white hover:bg-white hover:text-brand" asChild>
                <Link href="/signup">
                  Start Free
                  <ArrowRight className="ml-2 h-4 w-4" />
                </Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      </section>
    </div>
  );
}
