"use client";

import { cn } from "@/lib/utils";

// Placeholder logos - in production these would be real customer logos
const customerLogos = [
  { name: "Fortune 500 Bank", placeholder: "FIN" },
  { name: "Global Healthcare", placeholder: "MED" },
  { name: "Tech Enterprise", placeholder: "TECH" },
  { name: "Manufacturing Corp", placeholder: "MFG" },
  { name: "Retail Giant", placeholder: "RTL" },
  { name: "Insurance Leader", placeholder: "INS" },
];

const stats = [
  { value: "12M+", label: "Assets Protected" },
  { value: "99.9%", label: "Uptime SLA" },
  { value: "<1%", label: "False Positive Rate" },
  { value: "24/7", label: "Global Support" },
];

interface TestimonialProps {
  quote: string;
  author: string;
  role: string;
  company: string;
}

const testimonials: TestimonialProps[] = [
  {
    quote:
      "QL-RF transformed how we manage golden images across 15,000 VMs. We went from days to minutes for drift detection.",
    author: "Sarah Chen",
    role: "VP of Platform Engineering",
    company: "Fortune 500 Financial Services",
  },
  {
    quote:
      "The AI-powered CVE triage alone saved our security team 20 hours per week. The compliance evidence packs are a game-changer for audits.",
    author: "Michael Torres",
    role: "CISO",
    company: "Global Healthcare Provider",
  },
  {
    quote:
      "Finally, a single control tower that works across AWS, Azure, and our on-prem vSphere environment. No more spreadsheet tracking.",
    author: "James Wright",
    role: "Director of Cloud Operations",
    company: "Enterprise SaaS Company",
  },
];

export function SocialProof() {
  return (
    <section className="border-y border-border bg-muted/30 py-20 md:py-32">
      <div className="container mx-auto px-4">
        {/* Customer Logos */}
        <div className="text-center">
          <p className="text-sm font-medium uppercase tracking-wider text-muted-foreground">
            Trusted by security-conscious enterprises worldwide
          </p>
          <div className="mt-8 flex flex-wrap items-center justify-center gap-8 md:gap-12">
            {customerLogos.map((logo) => (
              <div
                key={logo.name}
                className="flex h-12 w-24 items-center justify-center rounded border border-border/50 bg-card text-sm font-semibold text-muted-foreground"
              >
                {logo.placeholder}
              </div>
            ))}
          </div>
        </div>

        {/* Stats */}
        <div className="mt-20 grid grid-cols-2 gap-8 md:grid-cols-4">
          {stats.map((stat) => (
            <div key={stat.label} className="text-center">
              <div className="text-4xl font-bold tracking-tight text-foreground md:text-5xl">
                {stat.value}
              </div>
              <div className="mt-2 text-sm text-muted-foreground">
                {stat.label}
              </div>
            </div>
          ))}
        </div>

        {/* Testimonials */}
        <div className="mt-20">
          <h3 className="text-center text-lg font-semibold text-foreground">
            What our customers say
          </h3>
          <div className="mt-10 grid gap-8 md:grid-cols-3">
            {testimonials.map((testimonial) => (
              <div
                key={testimonial.author}
                className="rounded-xl border border-border bg-card p-6"
              >
                <blockquote className="text-muted-foreground">
                  &ldquo;{testimonial.quote}&rdquo;
                </blockquote>
                <div className="mt-4 border-t border-border pt-4">
                  <div className="font-medium text-foreground">
                    {testimonial.author}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {testimonial.role}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {testimonial.company}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
