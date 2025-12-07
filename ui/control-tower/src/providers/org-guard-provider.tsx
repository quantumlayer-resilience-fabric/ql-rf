"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter, usePathname } from "next/navigation";
import { useAuth } from "@clerk/nextjs";
import { api } from "@/lib/api";
import { Loader2 } from "lucide-react";

/**
 * OrgGuardProvider checks if the current user has an organization.
 * If not, redirects them to the onboarding page.
 *
 * This provider should be wrapped inside AuthProvider to ensure
 * the API client has the auth token available.
 */
export function OrgGuardProvider({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const { isLoaded, isSignedIn } = useAuth();
  const [isChecking, setIsChecking] = useState(true);
  const [hasOrg, setHasOrg] = useState<boolean | null>(null);

  const checkOrganization = useCallback(async () => {
    // Skip check if on onboarding page
    if (pathname === "/onboarding") {
      setIsChecking(false);
      setHasOrg(true); // Allow rendering on onboarding
      return;
    }

    // Wait for auth to be loaded and user signed in
    if (!isLoaded || !isSignedIn) {
      return;
    }

    try {
      const result = await api.organization.check();

      if (!result.has_organization) {
        console.log("[OrgGuard] User has no organization, redirecting to onboarding");
        router.push("/onboarding");
        return;
      }

      setHasOrg(true);
    } catch (error) {
      console.warn("[OrgGuard] Failed to check organization:", error);
      // Fail open - allow access if check fails
      setHasOrg(true);
    } finally {
      setIsChecking(false);
    }
  }, [pathname, isLoaded, isSignedIn, router]);

  useEffect(() => {
    checkOrganization();
  }, [checkOrganization]);

  // If on onboarding page, always render children
  if (pathname === "/onboarding") {
    return <>{children}</>;
  }

  // Show loading state while checking
  if (isChecking || hasOrg === null) {
    return (
      <div className="flex h-screen w-full items-center justify-center bg-background">
        <div className="flex flex-col items-center gap-4">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
          <p className="text-sm text-muted-foreground">Loading...</p>
        </div>
      </div>
    );
  }

  // Render children if user has an organization
  return <>{children}</>;
}
