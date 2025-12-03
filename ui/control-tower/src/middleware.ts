import { clerkMiddleware, createRouteMatcher } from "@clerk/nextjs/server";
import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

// Dev bypass - set NEXT_PUBLIC_DEV_AUTH_BYPASS=true to skip Clerk entirely
const devAuthBypass = process.env.NEXT_PUBLIC_DEV_AUTH_BYPASS === "true";

// Check if Clerk is configured with a valid key
const hasClerkKey =
  !devAuthBypass &&
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY &&
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY.startsWith("pk_") &&
  !process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY.includes("xxxxx");

// Public routes that don't require authentication
const isPublicRoute = createRouteMatcher([
  "/",
  "/features",
  "/pricing",
  "/security",
  "/demo",
  "/login(.*)",
  "/signup(.*)",
  "/api/webhooks(.*)",
]);

// Use Clerk middleware if configured, otherwise allow all routes
export default hasClerkKey
  ? clerkMiddleware(async (auth, request) => {
      if (!isPublicRoute(request)) {
        await auth.protect();
      }
    })
  : function devMiddleware(_request: NextRequest) {
      // In development without Clerk or with DEV_AUTH_BYPASS, allow all requests
      return NextResponse.next();
    };

export const config = {
  matcher: [
    // Skip Next.js internals and static files
    "/((?!_next|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|webp|png|gif|svg|ttf|woff2?|ico|csv|docx?|xlsx?|zip|webmanifest)).*)",
    // Always run for API routes
    "/(api|trpc)(.*)",
  ],
};
