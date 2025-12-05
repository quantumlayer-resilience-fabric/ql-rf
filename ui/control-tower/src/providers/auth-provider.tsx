"use client";

import { useEffect, useRef, useCallback } from "react";
import { useAuth } from "@clerk/nextjs";
import { setAuthTokenGetter } from "@/lib/api";
import { setAuthTokenGetter as setFinOpsAuthTokenGetter } from "@/lib/api-finops";
import { setAuthTokenGetter as setSBOMAuthTokenGetter } from "@/lib/api-sbom";
import { setAuthTokenGetter as setInSpecAuthTokenGetter } from "@/lib/api-inspec";

/**
 * AuthProvider component that connects Clerk's auth to our API client.
 * Must be rendered within ClerkProvider.
 *
 * This component sets up the token getter for the API client to use
 * Clerk's JWT tokens for authentication with the backend.
 */
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const { getToken, isLoaded, isSignedIn } = useAuth();
  const tokenGetterRef = useRef(getToken);
  const isLoadedRef = useRef(isLoaded);

  // Keep refs in sync with latest values using useEffect
  useEffect(() => {
    tokenGetterRef.current = getToken;
  }, [getToken]);

  useEffect(() => {
    isLoadedRef.current = isLoaded;
  }, [isLoaded]);

  // Create stable token getter using useCallback
  const stableTokenGetter = useCallback(async () => {
    // Wait for Clerk to be loaded (with timeout)
    let attempts = 0;
    while (!isLoadedRef.current && attempts < 50) {
      await new Promise(resolve => setTimeout(resolve, 100));
      attempts++;
    }

    if (!isLoadedRef.current) {
      console.warn("[Auth] Clerk not loaded after 5s timeout");
      return null;
    }

    try {
      const token = await tokenGetterRef.current();
      if (!token) {
        console.warn("[Auth] No auth token available - user may need to sign in");
        return null;
      }
      return token;
    } catch (error) {
      console.error("[Auth] Failed to get auth token:", error);
      return null;
    }
  }, []);

  // Set up the token getter on mount
  useEffect(() => {
    // Set the token getter for all API modules
    setAuthTokenGetter(stableTokenGetter);
    setFinOpsAuthTokenGetter(stableTokenGetter);
    setSBOMAuthTokenGetter(stableTokenGetter);
    setInSpecAuthTokenGetter(stableTokenGetter);
  }, [stableTokenGetter]);

  // Log when auth state changes (for debugging)
  useEffect(() => {
    if (isLoaded) {
      console.log("[Auth] Clerk loaded, signed in:", isSignedIn);
    }
  }, [isLoaded, isSignedIn]);

  return <>{children}</>;
}
