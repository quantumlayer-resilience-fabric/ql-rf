"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { GradientText, TrustBadgeRow } from "@/components/brand";
import { ArrowRight, Play, Sparkles } from "lucide-react";

export function HeroSection() {
  return (
    <section className="relative overflow-hidden bg-gradient-to-b from-background to-muted/30 pb-20 pt-16 md:pb-32 md:pt-24">
      {/* Animated Background Pattern */}
      <div className="absolute inset-0 -z-10 overflow-hidden">
        <div className="absolute left-1/2 top-0 -translate-x-1/2 -translate-y-1/2 h-[600px] w-[600px] rounded-full bg-gradient-to-r from-brand-accent/20 to-[var(--ai-end)]/20 blur-3xl animate-float" />
        <div className="absolute right-0 top-1/2 h-[400px] w-[400px] rounded-full bg-status-green/10 blur-3xl animate-float" style={{ animationDelay: '2s' }} />
        <div className="absolute left-0 bottom-0 h-[300px] w-[300px] rounded-full bg-status-amber/10 blur-3xl animate-float" style={{ animationDelay: '4s' }} />
      </div>

      <div className="container mx-auto px-4">
        <div className="mx-auto max-w-4xl text-center">
          {/* AI Badge */}
          <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-border bg-card/80 backdrop-blur-sm px-4 py-1.5 text-sm shadow-sm animate-in fade-in-0 slide-in-from-bottom-4 duration-700">
            <Sparkles className="h-4 w-4 text-[var(--ai-start)] animate-pulse" />
            <span className="text-muted-foreground">
              Powered by <GradientText variant="ai">AI-Driven Insights</GradientText>
            </span>
          </div>

          {/* Main Headline */}
          <h1
            className="text-4xl font-bold tracking-tight text-foreground md:text-5xl lg:text-6xl animate-in fade-in-0 slide-in-from-bottom-6 duration-700"
            style={{ fontFamily: "var(--font-display)", animationDelay: '100ms', animationFillMode: 'backwards' }}
          >
            One Control Tower for{" "}
            <span className="relative inline-block">
              <GradientText variant="brand" as="span" className="font-bold">
                All Your Clouds
              </GradientText>
              <span className="absolute -inset-1 -z-10 rounded-lg bg-brand-accent/10 blur-xl" />
            </span>
          </h1>

          {/* Subheadline */}
          <p
            className="mx-auto mt-6 max-w-2xl text-lg text-muted-foreground md:text-xl animate-in fade-in-0 slide-in-from-bottom-6 duration-700"
            style={{ animationDelay: '200ms', animationFillMode: 'backwards' }}
          >
            Real-time visibility into golden images, patch drift, compliance, and
            DR readiness across AWS, Azure, GCP, and on-premises infrastructure.
          </p>

          {/* CTAs */}
          <div
            className="mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row animate-in fade-in-0 slide-in-from-bottom-6 duration-700"
            style={{ animationDelay: '300ms', animationFillMode: 'backwards' }}
          >
            <Button variant="brand" size="xl" asChild className="shadow-lg hover:shadow-xl">
              <Link href="/signup">
                Start Free
                <ArrowRight className="ml-2 h-4 w-4" />
              </Link>
            </Button>
            <Button size="xl" variant="outline" asChild className="shadow-sm hover:shadow-md">
              <Link href="/demo">
                <Play className="mr-2 h-4 w-4" />
                Watch Demo
              </Link>
            </Button>
          </div>

          {/* Trust Signals */}
          <div
            className="mt-10 flex flex-col items-center gap-4 animate-in fade-in-0 duration-1000"
            style={{ animationDelay: '500ms', animationFillMode: 'backwards' }}
          >
            <p className="text-sm text-muted-foreground">
              Trusted by security-conscious enterprises
            </p>
            <TrustBadgeRow />
          </div>
        </div>

        {/* Product Screenshot */}
        <div
          className="mx-auto mt-16 max-w-5xl animate-in fade-in-0 slide-in-from-bottom-8 duration-1000"
          style={{ animationDelay: '400ms', animationFillMode: 'backwards' }}
        >
          <div className="relative rounded-xl border border-border bg-card p-2 shadow-2xl transition-shadow hover:shadow-[0_25px_50px_-12px_rgba(0,0,0,0.25)]">
            {/* Glow effect */}
            <div className="absolute -inset-px rounded-xl bg-gradient-to-r from-brand-accent/20 via-[var(--ai-start)]/20 to-status-green/20 blur-sm opacity-50" />

            {/* Browser Chrome */}
            <div className="relative flex items-center gap-2 border-b border-border px-4 py-3 bg-card rounded-t-lg">
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
            <div className="relative aspect-[16/9] rounded-b-lg bg-gradient-to-br from-[#0a0a0f] to-[#12121a] p-6 overflow-hidden">
              {/* Mock Dashboard Content */}
              <div className="grid grid-cols-4 gap-4 stagger-children">
                {/* Metric Cards */}
                {[
                  { label: "Fleet Size", value: "12,847", trend: "+234", status: "success" },
                  { label: "Drift Score", value: "94.2%", trend: "-2.1%", status: "warning" },
                  { label: "Compliance", value: "97.8%", trend: "stable", status: "success" },
                  { label: "DR Ready", value: "98.1%", trend: "+0.3%", status: "success" },
                ].map((metric) => (
                  <div
                    key={metric.label}
                    className="rounded-lg border border-white/10 bg-white/5 p-4 transition-all hover:border-white/20 hover:bg-white/10"
                  >
                    <div className="text-xs text-gray-400">{metric.label}</div>
                    <div
                      className="mt-1 text-2xl font-bold text-white tabular-nums"
                      style={{ fontFamily: "var(--font-display)" }}
                    >
                      {metric.value}
                    </div>
                    <div className={`mt-1 text-xs ${
                      metric.status === "warning" ? "text-amber-400" : "text-emerald-400"
                    }`}>
                      {metric.trend}
                    </div>
                  </div>
                ))}
              </div>
              {/* Mock Heatmap */}
              <div className="mt-4 rounded-lg border border-white/10 bg-white/5 p-4">
                <div
                  className="mb-3 text-sm font-medium text-white"
                  style={{ fontFamily: "var(--font-display)" }}
                >
                  Drift Heatmap by Site
                </div>
                <div className="flex gap-2">
                  {[
                    { name: "eu-west-1", status: "green", value: "98%" },
                    { name: "us-east-1", status: "green", value: "96%" },
                    { name: "ap-south-1", status: "amber", value: "82%" },
                    { name: "dc-london", status: "green", value: "95%" },
                    { name: "dc-singapore", status: "red", value: "67%" },
                    { name: "us-west-2", status: "green", value: "94%" },
                  ].map((site) => (
                    <div
                      key={site.name}
                      className={`flex-1 rounded-md p-3 text-center transition-all hover:scale-105 ${
                        site.status === "green"
                          ? "bg-emerald-500/20 text-emerald-400 hover:bg-emerald-500/30"
                          : site.status === "amber"
                          ? "bg-amber-500/20 text-amber-400 hover:bg-amber-500/30"
                          : "bg-red-500/20 text-red-400 hover:bg-red-500/30"
                      }`}
                    >
                      <div className="text-lg font-bold tabular-nums">{site.value}</div>
                      <div className="text-[10px] opacity-70">{site.name}</div>
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
