"use client";

import { useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Logo } from "@/components/brand/logo";
import { PlatformIcon } from "@/components/status/platform-icon";
import { Progress } from "@/components/ui/progress";
import {
  Check,
  ChevronRight,
  Cloud,
  Loader2,
  PartyPopper,
  Rocket,
  Server,
  Shield,
  Sparkles,
  Users,
} from "lucide-react";

type OnboardingStep = "welcome" | "connect" | "scanning" | "results" | "next-steps";
type Platform = "aws" | "azure" | "gcp";

const platformInfo: Record<Platform, { name: string; description: string; fields: { label: string; placeholder: string; type: string }[] }> = {
  aws: {
    name: "Amazon Web Services",
    description: "Connect using IAM role or access keys",
    fields: [
      { label: "AWS Account ID", placeholder: "123456789012", type: "text" },
      { label: "Access Key ID", placeholder: "AKIAIOSFODNN7EXAMPLE", type: "text" },
      { label: "Secret Access Key", placeholder: "••••••••••••••••••••", type: "password" },
    ],
  },
  azure: {
    name: "Microsoft Azure",
    description: "Connect using service principal",
    fields: [
      { label: "Tenant ID", placeholder: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", type: "text" },
      { label: "Client ID", placeholder: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", type: "text" },
      { label: "Client Secret", placeholder: "••••••••••••••••••••", type: "password" },
    ],
  },
  gcp: {
    name: "Google Cloud Platform",
    description: "Connect using service account",
    fields: [
      { label: "Project ID", placeholder: "my-project-123", type: "text" },
      { label: "Service Account Email", placeholder: "sa@project.iam.gserviceaccount.com", type: "text" },
      { label: "Private Key (JSON)", placeholder: "Paste JSON key or upload file", type: "textarea" },
    ],
  },
};

export default function OnboardingPage() {
  const [step, setStep] = useState<OnboardingStep>("welcome");
  const [selectedPlatform, setSelectedPlatform] = useState<Platform | null>(null);
  const [scanProgress, setScanProgress] = useState(0);
  const [scanPhase, setScanPhase] = useState("");

  const handleConnect = () => {
    setStep("scanning");
    // Simulate scanning progress
    const phases = [
      "Authenticating...",
      "Discovering regions...",
      "Scanning EC2 instances...",
      "Scanning EKS clusters...",
      "Analyzing configurations...",
      "Detecting drift...",
      "Generating insights...",
    ];
    let progress = 0;
    let phaseIndex = 0;

    const interval = setInterval(() => {
      progress += Math.random() * 15;
      if (progress > 100) progress = 100;
      setScanProgress(progress);

      if (progress > (phaseIndex + 1) * (100 / phases.length)) {
        phaseIndex = Math.min(phaseIndex + 1, phases.length - 1);
      }
      setScanPhase(phases[phaseIndex]);

      if (progress >= 100) {
        clearInterval(interval);
        setTimeout(() => setStep("results"), 500);
      }
    }, 400);
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-background to-muted">
      <div className="mx-auto max-w-3xl px-4 py-12">
        {/* Logo */}
        <div className="mb-8 text-center">
          <Logo variant="full" size="lg" />
        </div>

        {/* Progress Indicator */}
        <div className="mb-8 flex items-center justify-center gap-2">
          {["welcome", "connect", "scanning", "results", "next-steps"].map((s, i) => (
            <div key={s} className="flex items-center">
              <div
                className={`flex h-8 w-8 items-center justify-center rounded-full text-sm font-medium ${
                  step === s
                    ? "bg-brand-accent text-white"
                    : ["welcome", "connect", "scanning", "results", "next-steps"].indexOf(step) > i
                    ? "bg-status-green text-white"
                    : "bg-muted text-muted-foreground"
                }`}
              >
                {["welcome", "connect", "scanning", "results", "next-steps"].indexOf(step) > i ? (
                  <Check className="h-4 w-4" />
                ) : (
                  i + 1
                )}
              </div>
              {i < 4 && (
                <div
                  className={`h-0.5 w-8 ${
                    ["welcome", "connect", "scanning", "results", "next-steps"].indexOf(step) > i
                      ? "bg-status-green"
                      : "bg-muted"
                  }`}
                />
              )}
            </div>
          ))}
        </div>

        {/* Step Content */}
        {step === "welcome" && (
          <Card className="text-center">
            <CardHeader className="space-y-4 pb-2">
              <div className="mx-auto rounded-full bg-brand-accent/10 p-4">
                <Rocket className="h-12 w-12 text-brand-accent" />
              </div>
              <CardTitle className="text-3xl">Welcome to QL-RF!</CardTitle>
              <CardDescription className="text-lg">
                Let&apos;s get your infrastructure connected in under 5 minutes.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6 pt-4">
              <div className="mx-auto grid max-w-md gap-4 text-left">
                <div className="flex items-start gap-3">
                  <div className="rounded-lg bg-status-green/10 p-2">
                    <Cloud className="h-5 w-5 text-status-green" />
                  </div>
                  <div>
                    <h4 className="font-medium">Connect Your Cloud</h4>
                    <p className="text-sm text-muted-foreground">
                      AWS, Azure, GCP, or on-premises
                    </p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <div className="rounded-lg bg-brand-accent/10 p-2">
                    <Server className="h-5 w-5 text-brand-accent" />
                  </div>
                  <div>
                    <h4 className="font-medium">Discover Assets</h4>
                    <p className="text-sm text-muted-foreground">
                      Automatic inventory of all your infrastructure
                    </p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <div className="rounded-lg bg-purple-500/10 p-2">
                    <Sparkles className="h-5 w-5 text-purple-500" />
                  </div>
                  <div>
                    <h4 className="font-medium">Get AI Insights</h4>
                    <p className="text-sm text-muted-foreground">
                      Drift detection, compliance scoring, DR readiness
                    </p>
                  </div>
                </div>
              </div>
              <Button size="lg" onClick={() => setStep("connect")}>
                Let&apos;s Get Started
                <ChevronRight className="ml-2 h-5 w-5" />
              </Button>
            </CardContent>
          </Card>
        )}

        {step === "connect" && (
          <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>Connect Your First Cloud</CardTitle>
                <CardDescription>
                  Choose a cloud provider to connect. You can add more later.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="grid gap-4 md:grid-cols-3">
                  {(["aws", "azure", "gcp"] as Platform[]).map((platform) => (
                    <button
                      key={platform}
                      onClick={() => setSelectedPlatform(platform)}
                      className={`flex flex-col items-center gap-3 rounded-lg border-2 p-6 transition-all ${
                        selectedPlatform === platform
                          ? "border-brand-accent bg-brand-accent/5"
                          : "border-border hover:border-brand-accent/50"
                      }`}
                    >
                      <PlatformIcon platform={platform} size="lg" />
                      <span className="font-medium">{platformInfo[platform].name}</span>
                    </button>
                  ))}
                </div>
              </CardContent>
            </Card>

            {selectedPlatform && (
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <PlatformIcon platform={selectedPlatform} size="sm" />
                    {platformInfo[selectedPlatform].name}
                  </CardTitle>
                  <CardDescription>
                    {platformInfo[selectedPlatform].description}
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  {platformInfo[selectedPlatform].fields.map((field) => (
                    <div key={field.label} className="space-y-2">
                      <Label>{field.label}</Label>
                      {field.type === "textarea" ? (
                        <textarea
                          className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                          placeholder={field.placeholder}
                        />
                      ) : (
                        <Input
                          type={field.type}
                          placeholder={field.placeholder}
                        />
                      )}
                    </div>
                  ))}
                  <div className="flex gap-3 pt-4">
                    <Button variant="outline" onClick={() => setSelectedPlatform(null)}>
                      Back
                    </Button>
                    <Button className="flex-1" onClick={handleConnect}>
                      Connect & Scan
                      <ChevronRight className="ml-2 h-4 w-4" />
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        )}

        {step === "scanning" && (
          <Card className="text-center">
            <CardHeader className="space-y-4 pb-2">
              <div className="mx-auto">
                <Loader2 className="h-16 w-16 animate-spin text-brand-accent" />
              </div>
              <CardTitle className="text-2xl">Scanning Your Infrastructure</CardTitle>
              <CardDescription>
                This usually takes 1-2 minutes. Grab a coffee!
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6 pt-4">
              <div className="mx-auto max-w-md space-y-2">
                <Progress value={scanProgress} className="h-2" />
                <p className="text-sm text-muted-foreground">{scanPhase}</p>
              </div>
              <div className="mx-auto grid max-w-sm gap-3 text-left">
                {[
                  { label: "Regions discovered", value: "3" },
                  { label: "Assets found", value: scanProgress > 30 ? "1,234" : "..." },
                  { label: "Images detected", value: scanProgress > 60 ? "47" : "..." },
                ].map((item) => (
                  <div key={item.label} className="flex items-center justify-between rounded-lg bg-muted/50 px-4 py-2">
                    <span className="text-sm text-muted-foreground">{item.label}</span>
                    <span className="font-mono font-medium">{item.value}</span>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        )}

        {step === "results" && (
          <Card className="text-center">
            <CardHeader className="space-y-4 pb-2">
              <div className="mx-auto rounded-full bg-status-green/10 p-4">
                <PartyPopper className="h-12 w-12 text-status-green" />
              </div>
              <CardTitle className="text-3xl">Amazing! Here&apos;s What We Found</CardTitle>
              <CardDescription>
                Your first scan is complete. Let&apos;s see your infrastructure health.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6 pt-4">
              <div className="grid gap-4 md:grid-cols-4">
                {[
                  { label: "Assets Discovered", value: "1,234", icon: Server, color: "text-brand-accent" },
                  { label: "Golden Images", value: "47", icon: Shield, color: "text-status-green" },
                  { label: "Drift Detected", value: "23", icon: Sparkles, color: "text-status-amber" },
                  { label: "Coverage", value: "94.2%", icon: Check, color: "text-status-green" },
                ].map((stat) => (
                  <div key={stat.label} className="rounded-lg border p-4">
                    <stat.icon className={`mx-auto h-8 w-8 ${stat.color}`} />
                    <div className="mt-2 text-2xl font-bold">{stat.value}</div>
                    <div className="text-xs text-muted-foreground">{stat.label}</div>
                  </div>
                ))}
              </div>
              <Button size="lg" onClick={() => setStep("next-steps")}>
                See What&apos;s Next
                <ChevronRight className="ml-2 h-5 w-5" />
              </Button>
            </CardContent>
          </Card>
        )}

        {step === "next-steps" && (
          <div className="space-y-6">
            <Card className="text-center">
              <CardHeader>
                <CardTitle className="text-2xl">You&apos;re All Set!</CardTitle>
                <CardDescription>
                  Here are some things you can do next to get the most out of QL-RF.
                </CardDescription>
              </CardHeader>
            </Card>

            <div className="grid gap-4 md:grid-cols-2">
              <Card className="cursor-pointer transition-all hover:border-brand-accent">
                <CardContent className="flex items-start gap-4 p-6">
                  <div className="rounded-lg bg-brand-accent/10 p-3">
                    <Cloud className="h-6 w-6 text-brand-accent" />
                  </div>
                  <div>
                    <h3 className="font-semibold">Add More Clouds</h3>
                    <p className="text-sm text-muted-foreground">
                      Connect Azure, GCP, or on-premises infrastructure
                    </p>
                  </div>
                </CardContent>
              </Card>

              <Card className="cursor-pointer transition-all hover:border-brand-accent">
                <CardContent className="flex items-start gap-4 p-6">
                  <div className="rounded-lg bg-purple-500/10 p-3">
                    <Users className="h-6 w-6 text-purple-500" />
                  </div>
                  <div>
                    <h3 className="font-semibold">Invite Your Team</h3>
                    <p className="text-sm text-muted-foreground">
                      Add team members to collaborate on infrastructure management
                    </p>
                  </div>
                </CardContent>
              </Card>

              <Card className="cursor-pointer transition-all hover:border-brand-accent">
                <CardContent className="flex items-start gap-4 p-6">
                  <div className="rounded-lg bg-status-green/10 p-3">
                    <Shield className="h-6 w-6 text-status-green" />
                  </div>
                  <div>
                    <h3 className="font-semibold">Review Drift</h3>
                    <p className="text-sm text-muted-foreground">
                      See which assets have drifted from golden images
                    </p>
                  </div>
                </CardContent>
              </Card>

              <Card className="cursor-pointer transition-all hover:border-brand-accent">
                <CardContent className="flex items-start gap-4 p-6">
                  <div className="rounded-lg bg-status-amber/10 p-3">
                    <Sparkles className="h-6 w-6 text-status-amber" />
                  </div>
                  <div>
                    <h3 className="font-semibold">Explore AI Insights</h3>
                    <p className="text-sm text-muted-foreground">
                      Get AI-powered recommendations for your infrastructure
                    </p>
                  </div>
                </CardContent>
              </Card>
            </div>

            <div className="text-center">
              <Button size="lg" onClick={() => window.location.href = "/overview"}>
                Go to Dashboard
                <ChevronRight className="ml-2 h-5 w-5" />
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
