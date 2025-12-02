"use client";

import { cn } from "@/lib/utils";
import {
  Globe,
  Sparkles,
  Shield,
  Zap,
  Cloud,
  Server,
  Activity,
  Lock,
} from "lucide-react";

interface Feature {
  icon: typeof Globe;
  title: string;
  description: string;
  highlight?: string;
}

const mainFeatures: Feature[] = [
  {
    icon: Globe,
    title: "Multi-Cloud Unified",
    description:
      "Single pane of glass for AWS, Azure, GCP, VMware vSphere, and bare metal. No vendor lock-in, complete visibility.",
    highlight: "Platform-agnostic",
  },
  {
    icon: Sparkles,
    title: "AI-Powered Insights",
    description:
      "AI Copilot for anomaly detection, CVE triage, predictive risk scoring, and automated RCA generation.",
    highlight: "AI-driven",
  },
  {
    icon: Shield,
    title: "Compliance-First",
    description:
      "SBOM tracking, SLSA provenance, Cosign verification. Automated evidence packs for SOC2, ISO27001, and more.",
    highlight: "Audit-ready",
  },
  {
    icon: Zap,
    title: "Minutes Not Months",
    description:
      "From drift detection to remediation in minutes. Real-time monitoring with sub-minute sync intervals.",
    highlight: "10x faster",
  },
];

const additionalFeatures: Feature[] = [
  {
    icon: Cloud,
    title: "Golden Image Management",
    description:
      "Version-controlled image registry with multi-cloud format support. AMI, Azure SIG, GCE, vSphere templates.",
  },
  {
    icon: Activity,
    title: "Drift Detection",
    description:
      "Real-time patch parity monitoring. Know exactly which assets are behind baseline and by how much.",
  },
  {
    icon: Server,
    title: "DR Orchestration",
    description:
      "Automated DR drills, RTO/RPO tracking, and failover orchestration. Never miss a recovery target.",
  },
  {
    icon: Lock,
    title: "Supply Chain Security",
    description:
      "Cosign signatures, SLSA attestations, and provenance tracking. Verify every artifact before deployment.",
  },
];

export function FeatureGrid() {
  return (
    <section className="py-20 md:py-32">
      <div className="container mx-auto px-4">
        {/* Section Header */}
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight text-foreground md:text-4xl">
            Everything you need for infrastructure resilience
          </h2>
          <p className="mt-4 text-lg text-muted-foreground">
            A complete platform for managing golden images, detecting drift,
            ensuring compliance, and orchestrating disaster recovery.
          </p>
        </div>

        {/* Main Features - 4 Pillars */}
        <div className="mt-16 grid gap-6 md:grid-cols-2 lg:grid-cols-4">
          {mainFeatures.map((feature) => {
            const Icon = feature.icon;
            return (
              <div
                key={feature.title}
                className="group relative rounded-xl border border-border bg-card p-6 transition-all hover:border-brand-accent/50 hover:shadow-lg"
              >
                {/* Highlight Badge */}
                {feature.highlight && (
                  <div className="absolute -top-3 left-4 rounded-full bg-brand-accent px-3 py-0.5 text-xs font-medium text-white">
                    {feature.highlight}
                  </div>
                )}
                <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-brand-accent/10 text-brand-accent">
                  <Icon className="h-6 w-6" />
                </div>
                <h3 className="mt-4 text-lg font-semibold text-foreground">
                  {feature.title}
                </h3>
                <p className="mt-2 text-sm text-muted-foreground">
                  {feature.description}
                </p>
              </div>
            );
          })}
        </div>

        {/* Additional Features */}
        <div className="mt-16 grid gap-6 md:grid-cols-2 lg:grid-cols-4">
          {additionalFeatures.map((feature) => {
            const Icon = feature.icon;
            return (
              <div
                key={feature.title}
                className="rounded-lg border border-border/50 bg-muted/30 p-5 transition-colors hover:bg-muted/50"
              >
                <div className="flex items-center gap-3">
                  <Icon className="h-5 w-5 text-muted-foreground" />
                  <h3 className="font-medium text-foreground">{feature.title}</h3>
                </div>
                <p className="mt-2 text-sm text-muted-foreground">
                  {feature.description}
                </p>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
