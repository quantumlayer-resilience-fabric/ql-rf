"use client";

import { useEffect } from "react";
import { setAuthTokenGetter } from "@/lib/api";

// Check if Clerk is configured
const hasClerkKey =
  typeof process !== "undefined" &&
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY &&
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY.startsWith("pk_") &&
  !process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY.includes("xxxxx");

/**
 * AuthProvider component that connects Clerk's auth to our API client.
 * Must be rendered within ClerkProvider (when Clerk is configured).
 */
export function AuthProvider({ children }: { children: React.ReactNode }) {
  // Only try to use Clerk hooks if Clerk is configured
  if (hasClerkKey) {
    return <ClerkAuthProvider>{children}</ClerkAuthProvider>;
  }

  // Dev mode: no auth, use dev token
  useEffect(() => {
    setAuthTokenGetter(async () => "dev-token");
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
        return await getToken();
      } catch {
        return null;
      }
    });
  }, [getToken]);

  return <>{children}</>;
}
