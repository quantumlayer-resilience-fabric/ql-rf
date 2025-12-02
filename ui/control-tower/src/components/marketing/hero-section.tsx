"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { GradientText, TrustBadgeRow } from "@/components/brand";
import { ArrowRight, Play, Sparkles } from "lucide-react";

export function HeroSection() {
  return (
    <section className="relative overflow-hidden bg-gradient-to-b from-background to-muted/30 pb-20 pt-16 md:pb-32 md:pt-24">
      {/* Background Pattern */}
      <div className="absolute inset-0 -z-10 overflow-hidden">
        <div className="absolute left-1/2 top-0 -translate-x-1/2 -translate-y-1/2 h-[600px] w-[600px] rounded-full bg-gradient-to-r from-brand-accent/20 to-[var(--ai-end)]/20 blur-3xl" />
        <div className="absolute right-0 top-1/2 h-[400px] w-[400px] rounded-full bg-status-green/10 blur-3xl" />
      </div>

      <div className="container mx-auto px-4">
        <div className="mx-auto max-w-4xl text-center">
          {/* AI Badge */}
          <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-border bg-card px-4 py-1.5 text-sm">
            <Sparkles className="h-4 w-4 text-[var(--ai-start)]" />
            <span className="text-muted-foreground">
              Powered by <GradientText variant="ai">AI-Driven Insights</GradientText>
            </span>
          </div>

          {/* Main Headline */}
          <h1 className="text-4xl font-bold tracking-tight text-foreground md:text-5xl lg:text-6xl">
            One Control Tower for{" "}
            <span className="relative">
              <GradientText variant="brand" as="span" className="font-bold">
                All Your Clouds
              </GradientText>
            </span>
          </h1>

          {/* Subheadline */}
          <p className="mx-auto mt-6 max-w-2xl text-lg text-muted-foreground md:text-xl">
            Real-time visibility into golden images, patch drift, compliance, and
            DR readiness across AWS, Azure, GCP, and on-premises infrastructure.
          </p>

          {/* CTAs */}
          <div className="mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row">
            <Button size="lg" asChild className="h-12 px-8 text-base">
              <Link href="/signup">
                Start Free
                <ArrowRight className="ml-2 h-4 w-4" />
              </Link>
            </Button>
            <Button size="lg" variant="outline" asChild className="h-12 px-8 text-base">
              <Link href="/demo">
                <Play className="mr-2 h-4 w-4" />
                Watch Demo
              </Link>
            </Button>
          </div>

          {/* Trust Signals */}
          <div className="mt-10 flex flex-col items-center gap-4">
            <p className="text-sm text-muted-foreground">
              Trusted by security-conscious enterprises
            </p>
            <TrustBadgeRow />
          </div>
        </div>

        {/* Product Screenshot */}
        <div className="mx-auto mt-16 max-w-5xl">
          <div className="relative rounded-xl border border-border bg-card p-2 shadow-2xl">
            {/* Browser Chrome */}
            <div className="flex items-center gap-2 border-b border-border px-4 py-3">
              <div className="flex gap-1.5">
                <div className="h-3 w-3 rounded-full bg-red-500/80" />
                <div className="h-3 w-3 rounded-full bg-yellow-500/80" />
                <div className="h-3 w-3 rounded-full bg-green-500/80" />
              </div>
              <div className="ml-4 flex-1 rounded-md bg-muted px-3 py-1 text-xs text-muted-foreground">
                app.quantumlayer.io/overview
              </div>
            </div>
            {/* Dashboard Preview */}
            <div className="aspect-[16/9] rounded-b-lg bg-gradient-to-br from-[#0a0a0f] to-[#12121a] p-6">
              {/* Mock Dashboard Content */}
              <div className="grid grid-cols-4 gap-4">
                {/* Metric Cards */}
                {[
                  { label: "Fleet Size", value: "12,847", trend: "+234" },
                  { label: "Drift Score", value: "94.2%", trend: "-2.1%" },
                  { label: "Compliance", value: "97.8%", trend: "stable" },
                  { label: "DR Ready", value: "98.1%", trend: "+0.3%" },
                ].map((metric) => (
                  <div
                    key={metric.label}
                    className="rounded-lg border border-white/10 bg-white/5 p-4"
                  >
                    <div className="text-xs text-gray-400">{metric.label}</div>
                    <div className="mt-1 text-2xl font-bold text-white">
                      {metric.value}
                    </div>
                    <div className="mt-1 text-xs text-emerald-400">
                      {metric.trend}
                    </div>
                  </div>
                ))}
              </div>
              {/* Mock Heatmap */}
              <div className="mt-4 rounded-lg border border-white/10 bg-white/5 p-4">
                <div className="mb-3 text-sm font-medium text-white">
                  Drift Heatmap by Site
                </div>
                <div className="flex gap-2">
                  {[
                    { name: "eu-west-1", status: "green" },
                    { name: "us-east-1", status: "green" },
                    { name: "ap-south-1", status: "amber" },
                    { name: "dc-london", status: "green" },
                    { name: "dc-singapore", status: "red" },
                    { name: "us-west-2", status: "green" },
                  ].map((site) => (
                    <div
                      key={site.name}
                      className={`flex-1 rounded-md p-3 text-center text-xs ${
                        site.status === "green"
                          ? "bg-emerald-500/20 text-emerald-400"
                          : site.status === "amber"
                          ? "bg-amber-500/20 text-amber-400"
                          : "bg-red-500/20 text-red-400"
                      }`}
                    >
                      {site.name}
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
