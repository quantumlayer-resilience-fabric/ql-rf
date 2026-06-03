"use client";

import { ClerkProvider } from "@clerk/nextjs";
import { type ReactNode } from "react";

/**
 * ConditionalClerkProvider - Wraps children with ClerkProvider only if
 * a valid publishable key is available. This allows the app to run in
 * development mode without Clerk authentication.
 */
export function ConditionalClerkProvider({ children }: { children: ReactNode }) {
  const publishableKey = process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY;

  // Check if we have a valid Clerk key (not a placeholder).
  // We wrap with ClerkProvider whenever a valid key is present so client-side
  // useAuth() hooks don't throw — DEV_AUTH_BYPASS only affects the middleware
  // (server-side route protection), not the provider tree.
  const hasValidKey =
    publishableKey &&
    publishableKey.startsWith("pk_") &&
    !publishableKey.includes("xxxxx");

  if (hasValidKey) {
    return <ClerkProvider>{children}</ClerkProvider>;
  }

  // In dev mode without Clerk or with DEV_AUTH_BYPASS, just render children directly
  return <>{children}</>;
}
