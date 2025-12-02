"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ArrowRight } from "lucide-react";

export function CTASection() {
  return (
    <section className="relative overflow-hidden py-20 md:py-32">
      {/* Background Gradient */}
      <div className="absolute inset-0 -z-10 bg-gradient-to-br from-brand via-brand-light to-brand-accent" />
      <div className="absolute inset-0 -z-10 bg-[url('/grid.svg')] opacity-10" />

      <div className="container mx-auto px-4">
        <div className="mx-auto max-w-3xl text-center">
          {/* Headline */}
          <h2 className="text-3xl font-bold tracking-tight text-white md:text-4xl lg:text-5xl">
            Ready to see your drift?
          </h2>
          <p className="mx-auto mt-4 max-w-xl text-lg text-white/80">
            Start with 50 assets free. No credit card required. Get visibility
            into your infrastructure in minutes.
          </p>

          {/* Email Signup Form */}
          <div className="mt-10 flex flex-col items-center gap-4 sm:flex-row sm:justify-center">
            <div className="relative w-full max-w-sm">
              <Input
                type="email"
                placeholder="Enter your work email"
                className="h-12 w-full border-white/20 bg-white/10 pr-4 text-white placeholder:text-white/60 focus:border-white focus:ring-white"
              />
            </div>
            <Button
              size="lg"
              variant="secondary"
              asChild
              className="h-12 w-full px-8 sm:w-auto"
            >
              <Link href="/signup">
                Start Free Trial
                <ArrowRight className="ml-2 h-4 w-4" />
              </Link>
            </Button>
          </div>

          {/* Secondary CTA */}
          <p className="mt-6 text-sm text-white/60">
            Want to see it first?{" "}
            <Link
              href="/demo"
              className="font-medium text-white underline underline-offset-4 hover:text-white/90"
            >
              Watch the demo
            </Link>{" "}
            or{" "}
            <Link
              href="/contact"
              className="font-medium text-white underline underline-offset-4 hover:text-white/90"
            >
              talk to sales
            </Link>
          </p>

          {/* Trust Note */}
          <div className="mt-10 flex flex-wrap items-center justify-center gap-x-8 gap-y-4 text-sm text-white/60">
            <div className="flex items-center gap-2">
              <svg
                className="h-4 w-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M5 13l4 4L19 7"
                />
              </svg>
              No credit card required
            </div>
            <div className="flex items-center gap-2">
              <svg
                className="h-4 w-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M5 13l4 4L19 7"
                />
              </svg>
              14-day free trial
            </div>
            <div className="flex items-center gap-2">
              <svg
                className="h-4 w-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M5 13l4 4L19 7"
                />
              </svg>
              Cancel anytime
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
