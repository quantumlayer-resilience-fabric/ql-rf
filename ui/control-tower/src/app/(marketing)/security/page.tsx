"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import Link from "next/link";
import {
  Shield,
  Lock,
  Eye,
  CheckCircle,
  ArrowRight,
  FileText,
  Globe,
  KeyRound,
  UserCheck,
  Database,
  Fingerprint,
  Building,
  Award,
  AlertTriangle,
} from "lucide-react";

const certifications = [
  {
    name: "SOC 2 Type II",
    description: "Annual audit by independent third-party",
    status: "Certified",
    icon: Award,
  },
  {
    name: "ISO 27001",
    description: "Information security management",
    status: "Certified",
    icon: Shield,
  },
  {
    name: "GDPR",
    description: "EU data protection compliance",
    status: "Compliant",
    icon: Globe,
  },
  {
    name: "HIPAA",
    description: "Healthcare data protection",
    status: "Compliant",
    icon: Building,
  },
];

const securityFeatures = [
  {
    icon: Lock,
    title: "Encryption at Rest & Transit",
    description:
      "All data encrypted with AES-256 at rest and TLS 1.3 in transit. Customer-managed keys available for Enterprise.",
  },
  {
    icon: KeyRound,
    title: "SSO & SAML 2.0",
    description:
      "Enterprise-grade identity management with support for Okta, Azure AD, Google Workspace, and custom SAML providers.",
  },
  {
    icon: UserCheck,
    title: "Role-Based Access Control",
    description:
      "Granular permissions with predefined roles (Admin, Editor, Viewer) and custom role creation for Enterprise.",
  },
  {
    icon: Eye,
    title: "Audit Logging",
    description:
      "Complete audit trail of all user actions, API calls, and system events. Export to your SIEM of choice.",
  },
  {
    icon: Database,
    title: "Data Isolation",
    description:
      "Dedicated tenant databases with logical separation. Physical isolation available for Enterprise customers.",
  },
  {
    icon: Fingerprint,
    title: "MFA & Passwordless",
    description:
      "Multi-factor authentication with TOTP, hardware keys (FIDO2), and passwordless options via WebAuthn.",
  },
];

const securityPractices = [
  {
    title: "Secure Development",
    items: [
      "Automated SAST/DAST scanning in CI/CD",
      "Mandatory code review for all changes",
      "Dependency vulnerability scanning",
      "Regular penetration testing",
    ],
  },
  {
    title: "Infrastructure Security",
    items: [
      "Zero-trust network architecture",
      "WAF and DDoS protection",
      "Intrusion detection systems",
      "24/7 security monitoring",
    ],
  },
  {
    title: "Data Protection",
    items: [
      "Automated backups with geo-redundancy",
      "Point-in-time recovery",
      "Data retention policies",
      "Right to erasure (GDPR)",
    ],
  },
  {
    title: "Incident Response",
    items: [
      "Documented IR procedures",
      "< 1 hour initial response SLA",
      "Post-incident analysis reports",
      "Customer notification within 24h",
    ],
  },
];

const trustStats = [
  { value: "99.99%", label: "Uptime SLA" },
  { value: "< 1hr", label: "Incident Response" },
  { value: "0", label: "Data Breaches" },
  { value: "24/7", label: "Security Monitoring" },
];

export default function SecurityPage() {
  return (
    <div className="py-20">
      {/* Hero */}
      <section className="mx-auto max-w-7xl px-4 text-center">
        <Badge variant="outline" className="mb-4">
          <Shield className="mr-1 h-3 w-3" />
          Security & Trust
        </Badge>
        <h1 className="text-4xl font-bold tracking-tight md:text-5xl">
          Enterprise-grade security
          <br />
          <span className="text-brand-accent">you can trust</span>
        </h1>
        <p className="mx-auto mt-4 max-w-2xl text-lg text-muted-foreground">
          Your infrastructure data is protected by the same security standards used by Fortune 500 companies.
          We take security seriously so you can focus on your mission.
        </p>

        {/* Trust Stats */}
        <div className="mt-12 grid grid-cols-2 gap-6 md:grid-cols-4">
          {trustStats.map((stat) => (
            <div key={stat.label} className="rounded-lg border p-6">
              <div className="text-3xl font-bold text-brand-accent">{stat.value}</div>
              <div className="mt-1 text-sm text-muted-foreground">{stat.label}</div>
            </div>
          ))}
        </div>
      </section>

      {/* Certifications */}
      <section className="mx-auto mt-20 max-w-7xl px-4">
        <h2 className="text-center text-2xl font-bold">
          Certifications & Compliance
        </h2>
        <p className="mx-auto mt-2 max-w-2xl text-center text-muted-foreground">
          We maintain industry-leading security certifications and undergo regular third-party audits.
        </p>
        <div className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {certifications.map((cert) => (
            <Card key={cert.name} className="text-center">
              <CardContent className="pt-6">
                <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-brand-accent/10">
                  <cert.icon className="h-6 w-6 text-brand-accent" />
                </div>
                <h3 className="mt-4 font-semibold">{cert.name}</h3>
                <p className="mt-1 text-sm text-muted-foreground">{cert.description}</p>
                <Badge variant="secondary" className="mt-3 bg-status-green/10 text-status-green">
                  {cert.status}
                </Badge>
              </CardContent>
            </Card>
          ))}
        </div>
      </section>

      {/* Security Features */}
      <section className="mx-auto mt-20 max-w-7xl px-4">
        <h2 className="text-center text-2xl font-bold">
          Security Features
        </h2>
        <p className="mx-auto mt-2 max-w-2xl text-center text-muted-foreground">
          Built-in security controls protect your data at every layer.
        </p>
        <div className="mt-8 grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {securityFeatures.map((feature) => (
            <Card key={feature.title}>
              <CardContent className="p-6">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-brand-accent/10">
                  <feature.icon className="h-5 w-5 text-brand-accent" />
                </div>
                <h3 className="mt-4 font-semibold">{feature.title}</h3>
                <p className="mt-2 text-sm text-muted-foreground">{feature.description}</p>
              </CardContent>
            </Card>
          ))}
        </div>
      </section>

      {/* Security Practices */}
      <section className="mx-auto mt-20 max-w-7xl px-4">
        <h2 className="text-center text-2xl font-bold">
          Our Security Practices
        </h2>
        <p className="mx-auto mt-2 max-w-2xl text-center text-muted-foreground">
          Security is embedded in everything we do, from development to operations.
        </p>
        <div className="mt-8 grid gap-6 md:grid-cols-2">
          {securityPractices.map((practice) => (
            <Card key={practice.title}>
              <CardHeader>
                <CardTitle className="text-base">{practice.title}</CardTitle>
              </CardHeader>
              <CardContent>
                <ul className="space-y-2">
                  {practice.items.map((item) => (
                    <li key={item} className="flex items-center gap-2 text-sm">
                      <CheckCircle className="h-4 w-4 text-status-green" />
                      {item}
                    </li>
                  ))}
                </ul>
              </CardContent>
            </Card>
          ))}
        </div>
      </section>

      {/* Responsible Disclosure */}
      <section className="mx-auto mt-20 max-w-7xl px-4">
        <Card className="border-status-amber/30 bg-status-amber/5">
          <CardContent className="flex flex-col items-center gap-6 p-8 md:flex-row">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-status-amber/20">
              <AlertTriangle className="h-6 w-6 text-status-amber" />
            </div>
            <div className="flex-1 text-center md:text-left">
              <h3 className="text-lg font-semibold">Responsible Disclosure</h3>
              <p className="mt-1 text-sm text-muted-foreground">
                Found a security vulnerability? We appreciate your help in keeping QL-RF secure.
                Please report security issues to our security team and we&apos;ll respond within 24 hours.
              </p>
            </div>
            <Button variant="outline" asChild>
              <Link href="mailto:security@ql-rf.io">
                Report Vulnerability
                <ArrowRight className="ml-2 h-4 w-4" />
              </Link>
            </Button>
          </CardContent>
        </Card>
      </section>

      {/* Documentation CTA */}
      <section className="mx-auto mt-20 max-w-7xl px-4">
        <Card className="bg-brand text-white">
          <CardContent className="flex flex-col items-center justify-between gap-6 p-8 md:flex-row">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-white/10">
                <FileText className="h-6 w-6" />
              </div>
              <div>
                <h3 className="text-xl font-bold">Security Documentation</h3>
                <p className="mt-1 text-white/80">
                  Download our SOC 2 report, security whitepaper, and data processing agreement.
                </p>
              </div>
            </div>
            <div className="flex gap-3">
              <Button variant="secondary" asChild>
                <Link href="/docs/security">
                  View Docs
                </Link>
              </Button>
              <Button variant="outline" className="border-white text-white hover:bg-white hover:text-brand" asChild>
                <Link href="/contact">
                  Request NDA
                  <ArrowRight className="ml-2 h-4 w-4" />
                </Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      </section>

      {/* FAQ-like Section */}
      <section className="mx-auto mt-20 max-w-3xl px-4">
        <h2 className="text-center text-2xl font-bold">
          Common Security Questions
        </h2>
        <div className="mt-8 space-y-4">
          <Card>
            <CardContent className="p-6">
              <h4 className="font-medium">Where is my data stored?</h4>
              <p className="mt-2 text-sm text-muted-foreground">
                Data is stored in AWS data centers in your chosen region (US, EU, or APAC).
                Enterprise customers can specify exact regions for compliance requirements.
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-6">
              <h4 className="font-medium">How long do you retain my data?</h4>
              <p className="mt-2 text-sm text-muted-foreground">
                Active account data is retained for the duration of your subscription.
                After cancellation, data is retained for 30 days before permanent deletion.
                You can request immediate deletion at any time.
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-6">
              <h4 className="font-medium">Do you share data with third parties?</h4>
              <p className="mt-2 text-sm text-muted-foreground">
                We never sell your data. We only share data with essential service providers
                (cloud hosting, payment processing) under strict contractual obligations.
                All sub-processors are listed in our DPA.
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-6">
              <h4 className="font-medium">What happens during an outage?</h4>
              <p className="mt-2 text-sm text-muted-foreground">
                We maintain real-time status at status.ql-rf.io. During incidents, we provide
                updates every 15 minutes. Post-incident reports are shared within 5 business days.
                Enterprise customers get dedicated Slack channel updates.
              </p>
            </CardContent>
          </Card>
        </div>
      </section>
    </div>
  );
}
