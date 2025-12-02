"use client";

import Link from "next/link";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Check } from "lucide-react";

interface PricingTier {
  name: string;
  description: string;
  price: string;
  priceNote?: string;
  cta: string;
  ctaHref: string;
  highlighted?: boolean;
  features: string[];
}

const pricingTiers: PricingTier[] = [
  {
    name: "Starter",
    description: "Perfect for teams getting started with drift detection.",
    price: "Free",
    priceNote: "Up to 50 assets",
    cta: "Start Free",
    ctaHref: "/signup",
    features: [
      "1 cloud platform",
      "Basic drift detection",
      "Daily sync interval",
      "RAG dashboard",
      "7-day data retention",
      "Community support",
    ],
  },
  {
    name: "Pro",
    description: "For growing teams needing full visibility and AI insights.",
    price: "$8",
    priceNote: "per asset / month",
    cta: "Start 14-day Trial",
    ctaHref: "/signup?plan=pro",
    highlighted: true,
    features: [
      "Unlimited assets",
      "All cloud platforms",
      "AI-powered insights",
      "5-minute sync interval",
      "SBOM & compliance tracking",
      "90-day data retention",
      "API access",
      "Email support",
    ],
  },
  {
    name: "Enterprise",
    description: "For large organizations with advanced security needs.",
    price: "Custom",
    priceNote: "Contact sales",
    cta: "Contact Sales",
    ctaHref: "/contact",
    features: [
      "Everything in Pro",
      "SSO / SAML",
      "Custom integrations",
      "DR orchestration",
      "Dedicated support",
      "SLA guarantees",
      "Custom data retention",
      "On-premises option",
      "SOC 2 attestation",
    ],
  },
];

export function PricingTable() {
  return (
    <section className="py-20 md:py-32">
      <div className="container mx-auto px-4">
        {/* Section Header */}
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight text-foreground md:text-4xl">
            Simple, transparent pricing
          </h2>
          <p className="mt-4 text-lg text-muted-foreground">
            Start free, scale as you grow. No surprises, no hidden fees.
          </p>
        </div>

        {/* Pricing Cards */}
        <div className="mx-auto mt-16 grid max-w-5xl gap-8 md:grid-cols-3">
          {pricingTiers.map((tier) => (
            <div
              key={tier.name}
              className={cn(
                "relative rounded-xl border p-8",
                tier.highlighted
                  ? "border-brand-accent bg-card shadow-lg shadow-brand-accent/10"
                  : "border-border bg-card"
              )}
            >
              {/* Popular Badge */}
              {tier.highlighted && (
                <div className="absolute -top-4 left-1/2 -translate-x-1/2 rounded-full bg-brand-accent px-4 py-1 text-xs font-medium text-white">
                  Most Popular
                </div>
              )}

              {/* Tier Info */}
              <div>
                <h3 className="text-xl font-semibold text-foreground">
                  {tier.name}
                </h3>
                <p className="mt-2 text-sm text-muted-foreground">
                  {tier.description}
                </p>
              </div>

              {/* Price */}
              <div className="mt-6">
                <div className="flex items-baseline gap-1">
                  <span className="text-4xl font-bold text-foreground">
                    {tier.price}
                  </span>
                  {tier.price !== "Free" && tier.price !== "Custom" && (
                    <span className="text-muted-foreground">/asset/mo</span>
                  )}
                </div>
                {tier.priceNote && (
                  <p className="mt-1 text-sm text-muted-foreground">
                    {tier.priceNote}
                  </p>
                )}
              </div>

              {/* CTA */}
              <Button
                asChild
                className={cn(
                  "mt-6 w-full",
                  tier.highlighted ? "" : "variant-outline"
                )}
                variant={tier.highlighted ? "default" : "outline"}
              >
                <Link href={tier.ctaHref}>{tier.cta}</Link>
              </Button>

              {/* Features */}
              <ul className="mt-8 space-y-3">
                {tier.features.map((feature) => (
                  <li key={feature} className="flex items-start gap-3">
                    <Check className="mt-0.5 h-4 w-4 shrink-0 text-status-green" />
                    <span className="text-sm text-muted-foreground">
                      {feature}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        {/* FAQ or Additional Info */}
        <div className="mx-auto mt-16 max-w-2xl text-center">
          <p className="text-sm text-muted-foreground">
            All plans include access to our documentation, API, and basic analytics.
            Need more? <Link href="/contact" className="text-brand-accent hover:underline">Contact us</Link> for custom enterprise pricing.
          </p>
        </div>
      </div>
    </section>
  );
}
