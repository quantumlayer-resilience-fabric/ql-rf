"use client";

import { Logo } from "@/components/brand/logo";
import Link from "next/link";

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen bg-gradient-to-br from-brand to-brand-light">
      {/* Background Pattern */}
      <div className="absolute inset-0 bg-[url('/grid.svg')] bg-center opacity-10" />

      {/* Content */}
      <div className="relative flex min-h-screen flex-col items-center justify-center p-4">
        {/* Logo */}
        <Link href="/" className="mb-8">
          <Logo variant="full" size="lg" className="text-white" />
        </Link>

        {/* Auth Card */}
        <div className="w-full max-w-md">
          {children}
        </div>

        {/* Footer */}
        <p className="mt-8 text-center text-sm text-white/60">
          By continuing, you agree to our{" "}
          <Link href="/terms" className="underline hover:text-white">
            Terms of Service
          </Link>{" "}
          and{" "}
          <Link href="/privacy" className="underline hover:text-white">
            Privacy Policy
          </Link>
        </p>
      </div>
    </div>
  );
}
