"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Logo } from "@/components/brand/logo";
import { PlatformIcon } from "@/components/status/platform-icon";
import { Progress } from "@/components/ui/progress";
import { api, SubscriptionPlan } from "@/lib/api";
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
  Zap,
  Building2,
  AlertCircle,
} from "lucide-react";

type OnboardingStep = "welcome" | "plan" | "org-setup" | "connect" | "scanning" | "results" | "next-steps";
type Platform = "aws" | "azure" | "gcp";

interface PlatformField {
  label: string;
  placeholder: string;
  type: string;
  key: string;
  required?: boolean;
}

interface PlatformConfig {
  name: string;
  description: string;
  fields: PlatformField[];
  toConnectorConfig: (credentials: Record<string, string>) => Record<string, unknown>;
}

const platformInfo: Record<Platform, PlatformConfig> = {
  aws: {
    name: "Amazon Web Services",
    description: "Connect using IAM role or access keys",
    fields: [
      { label: "Region", placeholder: "us-east-1", type: "text", key: "region", required: true },
      { label: "Access Key ID", placeholder: "AKIAIOSFODNN7EXAMPLE", type: "text", key: "accessKeyId" },
      { label: "Secret Access Key", placeholder: "Enter your secret key", type: "password", key: "secretAccessKey" },
      { label: "Assume Role ARN (optional)", placeholder: "arn:aws:iam::123456789012:role/RoleName", type: "text", key: "assumeRoleArn" },
    ],
    toConnectorConfig: (creds) => ({
      region: creds.region,
      access_key_id: creds.accessKeyId || undefined,
      secret_access_key: creds.secretAccessKey || undefined,
      assume_role_arn: creds.assumeRoleArn || undefined,
    }),
  },
  azure: {
    name: "Microsoft Azure",
    description: "Connect using service principal",
    fields: [
      { label: "Subscription ID", placeholder: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", type: "text", key: "subscriptionId", required: true },
      { label: "Tenant ID", placeholder: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", type: "text", key: "tenantId", required: true },
      { label: "Client ID", placeholder: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", type: "text", key: "clientId", required: true },
      { label: "Client Secret", placeholder: "Enter your client secret", type: "password", key: "clientSecret", required: true },
    ],
    toConnectorConfig: (creds) => ({
      subscription_id: creds.subscriptionId,
      tenant_id: creds.tenantId,
      client_id: creds.clientId,
      client_secret: creds.clientSecret,
    }),
  },
  gcp: {
    name: "Google Cloud Platform",
    description: "Connect using service account",
    fields: [
      { label: "Project ID", placeholder: "my-project-123", type: "text", key: "projectId", required: true },
      { label: "Credentials JSON", placeholder: "Paste service account JSON key", type: "textarea", key: "credentialsJson" },
    ],
    toConnectorConfig: (creds) => ({
      project_id: creds.projectId,
      credentials_file: creds.credentialsJson || undefined,
    }),
  },
};

export default function OnboardingPage() {
  const router = useRouter();
  const [step, setStep] = useState<OnboardingStep>("welcome");
  const [selectedPlan, setSelectedPlan] = useState<string>("free");
  const [selectedPlatform, setSelectedPlatform] = useState<Platform | null>(null);
  const [useDemoData, setUseDemoData] = useState(true);
  const [scanProgress, setScanProgress] = useState(0);
  const [scanPhase, setScanPhase] = useState("");
  const [scanResults, setScanResults] = useState({ sites: 0, assets: 0, images: 0 });
  const [orgName, setOrgName] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [plans, setPlans] = useState<SubscriptionPlan[]>([]);
  const [createdOrg, setCreatedOrg] = useState<{ id: string; name: string; slug: string } | null>(null);
  const [credentials, setCredentials] = useState<Record<string, string>>({});
  const [connectorId, setConnectorId] = useState<string | null>(null);
  const [connectionTested, setConnectionTested] = useState(false);

  // Load plans on mount
  useEffect(() => {
    async function loadPlans() {
      try {
        const result = await api.organization.listPlans();
        setPlans(result.plans);
      } catch {
        // Use default plans if API fails
        setPlans([
          {
            id: "free",
            name: "free",
            displayName: "Free",
            description: "Free tier with basic features",
            planType: "free",
            defaultMaxAssets: 100,
            defaultMaxImages: 10,
            defaultMaxSites: 5,
            defaultMaxUsers: 5,
            defaultMaxAiTasksPerDay: 10,
            defaultMaxAiTokensPerMonth: 100000,
            defaultMaxStorageBytes: 10737418240,
            drIncluded: false,
            complianceIncluded: false,
            advancedAnalyticsIncluded: false,
            customIntegrationsIncluded: false,
            isActive: true,
          },
          {
            id: "starter",
            name: "starter",
            displayName: "Starter",
            description: "Starter plan for small teams",
            planType: "starter",
            defaultMaxAssets: 500,
            defaultMaxImages: 50,
            defaultMaxSites: 20,
            defaultMaxUsers: 25,
            defaultMaxAiTasksPerDay: 50,
            defaultMaxAiTokensPerMonth: 1000000,
            defaultMaxStorageBytes: 53687091200,
            monthlyPriceUsd: 99,
            annualPriceUsd: 990,
            drIncluded: true,
            complianceIncluded: true,
            advancedAnalyticsIncluded: false,
            customIntegrationsIncluded: false,
            isActive: true,
          },
          {
            id: "professional",
            name: "professional",
            displayName: "Professional",
            description: "Professional plan for growing organizations",
            planType: "professional",
            defaultMaxAssets: 2000,
            defaultMaxImages: 200,
            defaultMaxSites: 100,
            defaultMaxUsers: 100,
            defaultMaxAiTasksPerDay: 200,
            defaultMaxAiTokensPerMonth: 10000000,
            defaultMaxStorageBytes: 214748364800,
            monthlyPriceUsd: 499,
            annualPriceUsd: 4990,
            drIncluded: true,
            complianceIncluded: true,
            advancedAnalyticsIncluded: true,
            customIntegrationsIncluded: false,
            isActive: true,
          },
        ]);
      }
    }
    loadPlans();
  }, []);

  const handleCreateOrg = async () => {
    if (!orgName.trim()) {
      setError("Organization name is required");
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const result = await api.organization.create({
        name: orgName.trim(),
        plan_id: selectedPlan,
      });
      setCreatedOrg(result.organization);
      setStep("connect");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create organization");
    } finally {
      setIsLoading(false);
    }
  };

  const validateCredentials = (): boolean => {
    if (!selectedPlatform) return false;
    const requiredFields = platformInfo[selectedPlatform].fields.filter(f => f.required);
    for (const field of requiredFields) {
      if (!credentials[field.key]?.trim()) {
        setError(`${field.label} is required`);
        return false;
      }
    }
    return true;
  };

  const handleConnect = async () => {
    if (!selectedPlatform) return;

    setStep("scanning");
    setError(null);

    if (useDemoData) {
      // Simulate scanning with demo data
      const phases = [
        "Initializing connection...",
        "Discovering regions...",
        "Scanning compute instances...",
        "Scanning container workloads...",
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
          // Seed demo data after "scan" completes
          seedDemoData();
        }
      }, 400);
    } else {
      // Real cloud connection using connectors API
      await connectRealCloud();
    }
  };

  const connectRealCloud = async () => {
    if (!selectedPlatform) return;

    try {
      setScanPhase("Creating connector...");
      setScanProgress(10);

      // Step 1: Create connector with credentials
      const config = platformInfo[selectedPlatform].toConnectorConfig(credentials);
      const connectorName = `${selectedPlatform}-${orgName || "primary"}`.toLowerCase().replace(/\s+/g, "-");

      const connector = await api.connectors.create({
        name: connectorName,
        platform: selectedPlatform,
        config,
      });
      setConnectorId(connector.id);
      setScanProgress(30);

      // Step 2: Test connection
      setScanPhase("Testing connection...");
      const testResult = await api.connectors.test(connector.id);
      setScanProgress(50);

      if (!testResult.success) {
        setError(`Connection test failed: ${testResult.message}`);
        setStep("connect");
        return;
      }
      setConnectionTested(true);

      // Step 3: Trigger sync to discover assets
      setScanPhase("Discovering assets...");
      setScanProgress(60);

      const syncResult = await api.connectors.sync(connector.id);
      setScanProgress(80);

      // Step 4: Show results
      setScanPhase("Finalizing...");
      setScanProgress(100);

      setScanResults({
        sites: syncResult.sites_created || 0,
        assets: syncResult.assets_found || 0,
        images: syncResult.images_found || 0,
      });

      setTimeout(() => setStep("results"), 500);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "Failed to connect to cloud platform";
      setError(errorMessage);
      setStep("connect");
    }
  };

  const seedDemoData = async () => {
    if (!selectedPlatform) return;

    try {
      const result = await api.organization.seedDemo(selectedPlatform);
      setScanResults({
        sites: result.sites_created,
        assets: result.assets_created,
        images: result.images_created,
      });
      setTimeout(() => setStep("results"), 500);
    } catch (err) {
      // If seeding fails, show mock results
      setScanResults({
        sites: 3,
        assets: 10,
        images: 5,
      });
      setTimeout(() => setStep("results"), 500);
    }
  };

  const steps: OnboardingStep[] = ["welcome", "plan", "org-setup", "connect", "scanning", "results", "next-steps"];
  const currentStepIndex = steps.indexOf(step);

  const getPlanIcon = (planType: string) => {
    switch (planType) {
      case "free": return Zap;
      case "starter": return Rocket;
      case "professional": return Building2;
      default: return Sparkles;
    }
  };

  const formatStorage = (bytes: number) => {
    if (bytes >= 1099511627776) return `${Math.round(bytes / 1099511627776)}TB`;
    if (bytes >= 1073741824) return `${Math.round(bytes / 1073741824)}GB`;
    return `${Math.round(bytes / 1048576)}MB`;
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-background to-muted">
      <div className="mx-auto max-w-4xl px-4 py-12">
        {/* Logo */}
        <div className="mb-8 text-center">
          <Logo variant="full" size="lg" />
        </div>

        {/* Progress Indicator */}
        <div className="mb-8 flex items-center justify-center gap-2">
          {steps.slice(0, 5).map((s, i) => (
            <div key={s} className="flex items-center">
              <div
                className={`flex h-8 w-8 items-center justify-center rounded-full text-sm font-medium ${
                  step === s || (step === "next-steps" && s === "results")
                    ? "bg-brand-accent text-white"
                    : currentStepIndex > i || step === "next-steps"
                    ? "bg-status-green text-white"
                    : "bg-muted text-muted-foreground"
                }`}
              >
                {currentStepIndex > i || step === "next-steps" ? (
                  <Check className="h-4 w-4" />
                ) : (
                  i + 1
                )}
              </div>
              {i < 4 && (
                <div
                  className={`h-0.5 w-8 ${
                    currentStepIndex > i || step === "next-steps"
                      ? "bg-status-green"
                      : "bg-muted"
                  }`}
                />
              )}
            </div>
          ))}
        </div>

        {/* Error Display */}
        {error && (
          <div className="mb-4 rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-destructive">
            <div className="flex items-center gap-2">
              <AlertCircle className="h-5 w-5" />
              <span>{error}</span>
            </div>
          </div>
        )}

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
              <Button size="lg" onClick={() => setStep("plan")}>
                Let&apos;s Get Started
                <ChevronRight className="ml-2 h-5 w-5" />
              </Button>
            </CardContent>
          </Card>
        )}

        {step === "plan" && (
          <div className="space-y-6">
            <Card className="text-center">
              <CardHeader>
                <CardTitle className="text-2xl">Choose Your Plan</CardTitle>
                <CardDescription>
                  Start with a 14-day free trial on any plan. No credit card required.
                </CardDescription>
              </CardHeader>
            </Card>

            <div className="grid gap-4 md:grid-cols-3">
              {plans.filter(p => p.planType !== "enterprise").map((plan) => {
                const Icon = getPlanIcon(plan.planType || "free");
                const isSelected = selectedPlan === plan.name;

                return (
                  <button
                    key={plan.id}
                    onClick={() => setSelectedPlan(plan.name || "free")}
                    className={`flex flex-col rounded-lg border-2 p-6 text-left transition-all ${
                      isSelected
                        ? "border-brand-accent bg-brand-accent/5"
                        : "border-border hover:border-brand-accent/50"
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div className={`rounded-lg p-2 ${isSelected ? "bg-brand-accent/10" : "bg-muted"}`}>
                        <Icon className={`h-6 w-6 ${isSelected ? "text-brand-accent" : "text-muted-foreground"}`} />
                      </div>
                      {isSelected && (
                        <div className="rounded-full bg-brand-accent p-1">
                          <Check className="h-4 w-4 text-white" />
                        </div>
                      )}
                    </div>

                    <h3 className="mt-4 text-lg font-semibold">{plan.displayName}</h3>
                    <p className="text-sm text-muted-foreground">{plan.description}</p>

                    <div className="mt-4">
                      {plan.monthlyPriceUsd ? (
                        <div className="flex items-baseline gap-1">
                          <span className="text-2xl font-bold">${plan.monthlyPriceUsd}</span>
                          <span className="text-muted-foreground">/month</span>
                        </div>
                      ) : (
                        <span className="text-2xl font-bold">Free</span>
                      )}
                    </div>

                    <div className="mt-4 space-y-2 text-sm">
                      <div className="flex items-center gap-2">
                        <Check className="h-4 w-4 text-status-green" />
                        <span>Up to {plan.defaultMaxAssets?.toLocaleString()} assets</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <Check className="h-4 w-4 text-status-green" />
                        <span>{plan.defaultMaxSites} sites</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <Check className="h-4 w-4 text-status-green" />
                        <span>{plan.defaultMaxUsers} users</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <Check className="h-4 w-4 text-status-green" />
                        <span>{formatStorage(plan.defaultMaxStorageBytes || 0)} storage</span>
                      </div>
                      {plan.drIncluded && (
                        <div className="flex items-center gap-2">
                          <Check className="h-4 w-4 text-status-green" />
                          <span>DR Automation</span>
                        </div>
                      )}
                      {plan.complianceIncluded && (
                        <div className="flex items-center gap-2">
                          <Check className="h-4 w-4 text-status-green" />
                          <span>Compliance Frameworks</span>
                        </div>
                      )}
                    </div>
                  </button>
                );
              })}
            </div>

            <div className="flex justify-center">
              <Button size="lg" onClick={() => setStep("org-setup")}>
                Continue with {plans.find(p => p.name === selectedPlan)?.displayName || "Free"}
                <ChevronRight className="ml-2 h-5 w-5" />
              </Button>
            </div>
          </div>
        )}

        {step === "org-setup" && (
          <Card>
            <CardHeader className="text-center">
              <div className="mx-auto rounded-full bg-brand-accent/10 p-4">
                <Building2 className="h-12 w-12 text-brand-accent" />
              </div>
              <CardTitle className="text-2xl">Create Your Organization</CardTitle>
              <CardDescription>
                Your organization is the workspace where your team manages infrastructure.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="mx-auto max-w-md space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="orgName">Organization Name</Label>
                  <Input
                    id="orgName"
                    placeholder="Acme Corporation"
                    value={orgName}
                    onChange={(e) => setOrgName(e.target.value)}
                    onKeyDown={(e) => e.key === "Enter" && handleCreateOrg()}
                  />
                  <p className="text-xs text-muted-foreground">
                    This can be your company name or team name.
                  </p>
                </div>

                <Button
                  className="w-full"
                  size="lg"
                  onClick={handleCreateOrg}
                  disabled={isLoading || !orgName.trim()}
                >
                  {isLoading ? (
                    <>
                      <Loader2 className="mr-2 h-5 w-5 animate-spin" />
                      Creating...
                    </>
                  ) : (
                    <>
                      Create Organization
                      <ChevronRight className="ml-2 h-5 w-5" />
                    </>
                  )}
                </Button>
              </div>
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
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <PlatformIcon platform={selectedPlatform} size="sm" />
                      <CardTitle>{platformInfo[selectedPlatform].name}</CardTitle>
                    </div>
                    <div className="flex items-center gap-2">
                      <Label htmlFor="demoToggle" className="text-sm text-muted-foreground">
                        Use Demo Data
                      </Label>
                      <button
                        id="demoToggle"
                        onClick={() => setUseDemoData(!useDemoData)}
                        className={`relative h-6 w-11 rounded-full transition-colors ${
                          useDemoData ? "bg-brand-accent" : "bg-muted"
                        }`}
                      >
                        <span
                          className={`absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white transition-transform ${
                            useDemoData ? "translate-x-5" : ""
                          }`}
                        />
                      </button>
                    </div>
                  </div>
                  <CardDescription>
                    {useDemoData
                      ? "Demo mode: We'll create sample data so you can explore the platform."
                      : platformInfo[selectedPlatform].description}
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  {!useDemoData && (
                    <>
                      {platformInfo[selectedPlatform].fields.map((field) => (
                        <div key={field.key} className="space-y-2">
                          <Label>
                            {field.label}
                            {field.required && <span className="text-destructive ml-1">*</span>}
                          </Label>
                          {field.type === "textarea" ? (
                            <textarea
                              className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono"
                              placeholder={field.placeholder}
                              value={credentials[field.key] || ""}
                              onChange={(e) => setCredentials(prev => ({ ...prev, [field.key]: e.target.value }))}
                            />
                          ) : (
                            <Input
                              type={field.type}
                              placeholder={field.placeholder}
                              value={credentials[field.key] || ""}
                              onChange={(e) => setCredentials(prev => ({ ...prev, [field.key]: e.target.value }))}
                            />
                          )}
                        </div>
                      ))}
                    </>
                  )}

                  {useDemoData && (
                    <div className="rounded-lg border border-brand-accent/20 bg-brand-accent/5 p-4">
                      <div className="flex items-start gap-3">
                        <Sparkles className="mt-0.5 h-5 w-5 text-brand-accent" />
                        <div>
                          <h4 className="font-medium">Demo Mode Enabled</h4>
                          <p className="text-sm text-muted-foreground">
                            We&apos;ll populate your dashboard with realistic sample data including
                            sites, assets, and golden images for {platformInfo[selectedPlatform].name}.
                          </p>
                        </div>
                      </div>
                    </div>
                  )}

                  <div className="flex gap-3 pt-4">
                    <Button variant="outline" onClick={() => {
                      setSelectedPlatform(null);
                      setCredentials({});
                      setError(null);
                    }}>
                      Back
                    </Button>
                    <Button
                      className="flex-1"
                      onClick={() => {
                        if (!useDemoData && !validateCredentials()) {
                          return;
                        }
                        handleConnect();
                      }}
                      disabled={isLoading}
                    >
                      {isLoading ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Connecting...
                        </>
                      ) : (
                        <>
                          {useDemoData ? "Start with Demo Data" : "Connect & Scan"}
                          <ChevronRight className="ml-2 h-4 w-4" />
                        </>
                      )}
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
              <CardTitle className="text-2xl">
                {useDemoData ? "Setting Up Your Demo Environment" : "Scanning Your Infrastructure"}
              </CardTitle>
              <CardDescription>
                {useDemoData
                  ? "Creating sample data for you to explore..."
                  : "This usually takes 1-2 minutes. Grab a coffee!"}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6 pt-4">
              <div className="mx-auto max-w-md space-y-2">
                <Progress value={scanProgress} className="h-2" />
                <p className="text-sm text-muted-foreground">{scanPhase}</p>
              </div>
              <div className="mx-auto grid max-w-sm gap-3 text-left">
                {[
                  { label: "Regions discovered", value: scanProgress > 20 ? "3" : "..." },
                  { label: "Assets found", value: scanProgress > 50 ? `${scanResults.assets || "..."}` : "..." },
                  { label: "Images detected", value: scanProgress > 80 ? `${scanResults.images || "..."}` : "..." },
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
              <CardTitle className="text-3xl">
                {useDemoData ? "Your Demo Environment is Ready!" : "Amazing! Here's What We Found"}
              </CardTitle>
              <CardDescription>
                {createdOrg && (
                  <span className="font-medium text-foreground">{createdOrg.name}</span>
                )} is all set up. Let&apos;s see your infrastructure.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6 pt-4">
              <div className="grid gap-4 md:grid-cols-4">
                {[
                  { label: "Sites Created", value: scanResults.sites.toString(), icon: Cloud, color: "text-brand-accent" },
                  { label: "Assets Discovered", value: scanResults.assets.toString(), icon: Server, color: "text-status-green" },
                  { label: "Golden Images", value: scanResults.images.toString(), icon: Shield, color: "text-purple-500" },
                  { label: "Ready to Go", value: "100%", icon: Check, color: "text-status-green" },
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
              <Card className="cursor-pointer transition-all hover:border-brand-accent" onClick={() => router.push("/sites")}>
                <CardContent className="flex items-start gap-4 p-6">
                  <div className="rounded-lg bg-brand-accent/10 p-3">
                    <Cloud className="h-6 w-6 text-brand-accent" />
                  </div>
                  <div>
                    <h3 className="font-semibold">Explore Your Sites</h3>
                    <p className="text-sm text-muted-foreground">
                      View your cloud regions and infrastructure sites
                    </p>
                  </div>
                </CardContent>
              </Card>

              <Card className="cursor-pointer transition-all hover:border-brand-accent" onClick={() => router.push("/images")}>
                <CardContent className="flex items-start gap-4 p-6">
                  <div className="rounded-lg bg-purple-500/10 p-3">
                    <Shield className="h-6 w-6 text-purple-500" />
                  </div>
                  <div>
                    <h3 className="font-semibold">View Golden Images</h3>
                    <p className="text-sm text-muted-foreground">
                      Explore your image families and lineage
                    </p>
                  </div>
                </CardContent>
              </Card>

              <Card className="cursor-pointer transition-all hover:border-brand-accent" onClick={() => router.push("/drift")}>
                <CardContent className="flex items-start gap-4 p-6">
                  <div className="rounded-lg bg-status-amber/10 p-3">
                    <Sparkles className="h-6 w-6 text-status-amber" />
                  </div>
                  <div>
                    <h3 className="font-semibold">Check for Drift</h3>
                    <p className="text-sm text-muted-foreground">
                      See which assets have drifted from golden images
                    </p>
                  </div>
                </CardContent>
              </Card>

              <Card className="cursor-pointer transition-all hover:border-brand-accent" onClick={() => router.push("/ai")}>
                <CardContent className="flex items-start gap-4 p-6">
                  <div className="rounded-lg bg-status-green/10 p-3">
                    <Users className="h-6 w-6 text-status-green" />
                  </div>
                  <div>
                    <h3 className="font-semibold">Try AI Copilot</h3>
                    <p className="text-sm text-muted-foreground">
                      Ask questions about your infrastructure in natural language
                    </p>
                  </div>
                </CardContent>
              </Card>
            </div>

            <div className="text-center">
              <Button size="lg" onClick={() => router.push("/overview")}>
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
