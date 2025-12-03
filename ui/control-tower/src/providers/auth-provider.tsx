"use client";

import { useEffect } from "react";
import { setAuthTokenGetter } from "@/lib/api";

// Check environment
const isDevelopment = process.env.NODE_ENV === "development";

// Dev bypass - set NEXT_PUBLIC_DEV_AUTH_BYPASS=true to skip Clerk entirely
const devAuthBypass = process.env.NEXT_PUBLIC_DEV_AUTH_BYPASS === "true";

// Check if Clerk is configured with a real key (and dev bypass is not enabled)
const hasClerkKey =
  !devAuthBypass &&
  typeof process !== "undefined" &&
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY &&
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY.startsWith("pk_") &&
  !process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY.includes("xxxxx");

/**
 * AuthProvider component that connects Clerk's auth to our API client.
 * Must be rendered within ClerkProvider (when Clerk is configured).
 *
 * PRODUCTION: Requires Clerk to be properly configured
 * DEVELOPMENT: Falls back to dev-token when Clerk is not configured
 */
export function AuthProvider({ children }: { children: React.ReactNode }) {
  // In development without proper Clerk config, always use dev auth
  // This avoids "useAuth can only be used within ClerkProvider" errors
  if (isDevelopment && !hasClerkKey) {
    return <DevAuthProvider>{children}</DevAuthProvider>;
  }

  // In development with Clerk config but possibly not wrapped in ClerkProvider,
  // still use dev auth to avoid errors
  if (isDevelopment) {
    return <DevAuthProvider>{children}</DevAuthProvider>;
  }

  // Production with Clerk - use Clerk auth
  if (hasClerkKey) {
    return <ClerkAuthProvider>{children}</ClerkAuthProvider>;
  }

  // Production without Clerk - this is a configuration error
  console.error("[Auth] CRITICAL: Clerk is not configured in production. Authentication will fail.");
  return <ProductionAuthErrorProvider>{children}</ProductionAuthErrorProvider>;
}

// Dev mode auth provider - sets dev token (ONLY for development)
function DevAuthProvider({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    setAuthTokenGetter(async () => "dev-token");
  }, []);

  return <>{children}</>;
}

// Production error state - no auth available
function ProductionAuthErrorProvider({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    // In production without Clerk, return null to force auth errors
    // This will cause API calls to fail, alerting operators to misconfiguration
    setAuthTokenGetter(async () => null);
  }, []);

  return <>{children}</>;
}

// Separate component that uses Clerk hooks
function ClerkAuthProvider({ children }: { children: React.ReactNode }) {
  // Dynamic import to avoid loading Clerk when not configured
  const { useAuth } = require("@clerk/nextjs");
  const { getToken } = useAuth();

  useEffect(() => {
    // Set up the token getter for the API client
    setAuthTokenGetter(async () => {
      try {
        const token = await getToken();
        if (!token) {
          // In development, fallback to dev-token for convenience
          if (isDevelopment) {
            console.warn("[Auth] Clerk returned no token - using dev-token fallback");
            return "dev-token";
          }
          // In production, return null - user needs to sign in
          console.warn("[Auth] No auth token available - user may need to sign in");
          return null;
        }
        return token;
      } catch (error) {
        // In development, fallback to dev-token for convenience
        if (isDevelopment) {
          console.warn("[Auth] Clerk error - using dev-token fallback:", error);
          return "dev-token";
        }
        // In production, return null and log the error
        console.error("[Auth] Failed to get auth token:", error);
        return null;
      }
    });
  }, [getToken]);

  return <>{children}</>;
}
