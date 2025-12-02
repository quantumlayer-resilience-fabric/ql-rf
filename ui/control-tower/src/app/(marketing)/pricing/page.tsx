"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import Link from "next/link";
import { Check, X, HelpCircle, ArrowRight, Sparkles } from "lucide-react";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

const plans = [
  {
    name: "Starter",
    description: "Perfect for small teams getting started",
    price: "Free",
    priceDetail: "forever",
    cta: "Start Free",
    ctaVariant: "outline" as const,
    highlighted: false,
    features: [
      { name: "Up to 50 assets", included: true },
      { name: "1 cloud provider", included: true },
      { name: "Basic drift detection", included: true },
      { name: "7-day data retention", included: true },
      { name: "Community support", included: true },
      { name: "CIS compliance", included: false },
      { name: "AI insights", included: false },
      { name: "DR management", included: false },
      { name: "SSO / SAML", included: false },
      { name: "Custom integrations", included: false },
    ],
  },
  {
    name: "Pro",
    description: "For growing teams with serious infrastructure",
    price: "$8",
    priceDetail: "per asset / month",
    cta: "Start Free Trial",
    ctaVariant: "default" as const,
    highlighted: true,
    badge: "Most Popular",
    features: [
      { name: "Unlimited assets", included: true },
      { name: "All cloud providers", included: true },
      { name: "Advanced drift detection", included: true },
      { name: "90-day data retention", included: true },
      { name: "Priority support", included: true },
      { name: "CIS compliance", included: true },
      { name: "AI insights", included: true },
      { name: "DR management", included: true },
      { name: "SSO / SAML", included: false },
      { name: "Custom integrations", included: false },
    ],
  },
  {
    name: "Enterprise",
    description: "For large organizations with complex needs",
    price: "Custom",
    priceDetail: "contact sales",
    cta: "Contact Sales",
    ctaVariant: "outline" as const,
    highlighted: false,
    features: [
      { name: "Unlimited assets", included: true },
      { name: "All cloud providers", included: true },
      { name: "Advanced drift detection", included: true },
      { name: "Unlimited data retention", included: true },
      { name: "Dedicated support", included: true },
      { name: "CIS compliance", included: true },
      { name: "AI insights", included: true },
      { name: "DR management", included: true },
      { name: "SSO / SAML", included: true },
      { name: "Custom integrations", included: true },
    ],
  },
];

const faqs = [
  {
    question: "How is pricing calculated?",
    answer: "Pricing is based on the number of assets (servers, VMs, containers) you monitor. We count unique assets, not duplicates across environments.",
  },
  {
    question: "What counts as an asset?",
    answer: "Any compute resource we monitor: EC2 instances, Azure VMs, GCP instances, vSphere VMs, Kubernetes nodes, and bare metal servers.",
  },
  {
    question: "Can I change plans anytime?",
    answer: "Yes, you can upgrade or downgrade your plan at any time. Changes take effect immediately, and we'll prorate your billing.",
  },
  {
    question: "Is there a free trial for Pro?",
    answer: "Yes! Pro comes with a 14-day free trial. No credit card required to start. You'll only pay after your trial ends.",
  },
  {
    question: "What payment methods do you accept?",
    answer: "We accept all major credit cards, ACH bank transfers, and can invoice enterprise customers for annual contracts.",
  },
  {
    question: "Do you offer discounts for annual billing?",
    answer: "Yes, annual billing comes with a 20% discount. Enterprise customers can also negotiate volume discounts.",
  },
];

export default function PricingPage() {
  const [billingCycle, setBillingCycle] = useState<"monthly" | "annual">("monthly");

  return (
    <TooltipProvider>
      <div className="py-20">
        {/* Hero */}
        <section className="mx-auto max-w-7xl px-4 text-center">
          <Badge variant="outline" className="mb-4">
            Pricing
          </Badge>
          <h1 className="text-4xl font-bold tracking-tight md:text-5xl">
            Simple, transparent pricing
          </h1>
          <p className="mx-auto mt-4 max-w-2xl text-lg text-muted-foreground">
            Start free, scale as you grow. No hidden fees, no surprises.
          </p>

          {/* Billing Toggle */}
          <div className="mt-8 flex items-center justify-center gap-4">
            <span className={billingCycle === "monthly" ? "font-medium" : "text-muted-foreground"}>
              Monthly
            </span>
            <button
              onClick={() => setBillingCycle(billingCycle === "monthly" ? "annual" : "monthly")}
              className="relative h-6 w-12 rounded-full bg-muted"
            >
              <div
                className={`absolute top-1 h-4 w-4 rounded-full bg-brand-accent transition-all ${
                  billingCycle === "annual" ? "left-7" : "left-1"
                }`}
              />
            </button>
            <span className={billingCycle === "annual" ? "font-medium" : "text-muted-foreground"}>
              Annual
              <Badge variant="secondary" className="ml-2 text-xs">
                Save 20%
              </Badge>
            </span>
          </div>
        </section>

        {/* Pricing Cards */}
        <section className="mx-auto mt-12 max-w-7xl px-4">
          <div className="grid gap-8 md:grid-cols-3">
            {plans.map((plan) => (
              <Card
                key={plan.name}
                className={`relative ${
                  plan.highlighted
                    ? "border-brand-accent shadow-lg scale-105"
                    : ""
                }`}
              >
                {plan.badge && (
                  <Badge className="absolute -top-3 left-1/2 -translate-x-1/2 bg-brand-accent">
                    {plan.badge}
                  </Badge>
                )}
                <CardHeader className="text-center">
                  <CardTitle className="text-xl">{plan.name}</CardTitle>
                  <p className="text-sm text-muted-foreground">{plan.description}</p>
                  <div className="mt-4">
                    <span className="text-4xl font-bold">
                      {plan.price === "Free" || plan.price === "Custom"
                        ? plan.price
                        : billingCycle === "annual"
                        ? `$${Math.round(parseInt(plan.price.replace("$", "")) * 0.8)}`
                        : plan.price}
                    </span>
                    {plan.priceDetail && (
                      <span className="text-muted-foreground">
                        {" "}
                        {plan.priceDetail}
                      </span>
                    )}
                  </div>
                </CardHeader>
                <CardContent className="space-y-6">
                  <Button
                    className="w-full"
                    variant={plan.ctaVariant}
                    asChild
                  >
                    <Link href={plan.name === "Enterprise" ? "/contact" : "/signup"}>
                      {plan.cta}
                      <ArrowRight className="ml-2 h-4 w-4" />
                    </Link>
                  </Button>
                  <ul className="space-y-3">
                    {plan.features.map((feature) => (
                      <li
                        key={feature.name}
                        className={`flex items-center gap-2 text-sm ${
                          feature.included ? "" : "text-muted-foreground"
                        }`}
                      >
                        {feature.included ? (
                          <Check className="h-4 w-4 text-status-green" />
                        ) : (
                          <X className="h-4 w-4" />
                        )}
                        {feature.name}
                      </li>
                    ))}
                  </ul>
                </CardContent>
              </Card>
            ))}
          </div>
        </section>

        {/* Enterprise CTA */}
        <section className="mx-auto mt-20 max-w-7xl px-4">
          <Card className="bg-gradient-to-r from-brand to-brand-light text-white">
            <CardContent className="flex flex-col items-center justify-between gap-6 p-8 md:flex-row">
              <div>
                <div className="flex items-center gap-2">
                  <Sparkles className="h-5 w-5" />
                  <span className="font-medium">Enterprise</span>
                </div>
                <h3 className="mt-2 text-2xl font-bold">
                  Need a custom solution?
                </h3>
                <p className="mt-2 text-white/80">
                  Talk to our team about volume discounts, custom SLAs, dedicated support, and on-premise deployment options.
                </p>
              </div>
              <Button variant="secondary" size="lg" asChild>
                <Link href="/contact">
                  Talk to Sales
                  <ArrowRight className="ml-2 h-4 w-4" />
                </Link>
              </Button>
            </CardContent>
          </Card>
        </section>

        {/* FAQs */}
        <section className="mx-auto mt-20 max-w-3xl px-4">
          <h2 className="text-center text-2xl font-bold">
            Frequently Asked Questions
          </h2>
          <div className="mt-8 space-y-4">
            {faqs.map((faq) => (
              <Card key={faq.question}>
                <CardContent className="p-6">
                  <h4 className="font-medium">{faq.question}</h4>
                  <p className="mt-2 text-sm text-muted-foreground">{faq.answer}</p>
                </CardContent>
              </Card>
            ))}
          </div>
        </section>
      </div>
    </TooltipProvider>
  );
}
