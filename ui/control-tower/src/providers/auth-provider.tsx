"use client";

import { useEffect, useRef } from "react";
import { useAuth } from "@clerk/nextjs";
import { setAuthTokenGetter } from "@/lib/api";

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

  // Keep refs in sync with latest values
  tokenGetterRef.current = getToken;
  isLoadedRef.current = isLoaded;

  // Set up the token getter immediately on first render
  // The getter will wait for auth to be loaded before returning a token
  useEffect(() => {
    setAuthTokenGetter(async () => {
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
    });
  }, []); // Only run once on mount

  // Log when auth state changes (for debugging)
  useEffect(() => {
    if (isLoaded) {
      console.log("[Auth] Clerk loaded, signed in:", isSignedIn);
    }
  }, [isLoaded, isSignedIn]);

  return <>{children}</>;
}
