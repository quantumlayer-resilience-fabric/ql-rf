import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

// Check if Clerk is configured
const hasClerkKey =
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY &&
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY.startsWith("pk_") &&
  !process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY.includes("xxxxx");

// Dynamic import of Clerk middleware only when key is available
let clerkMiddleware: typeof import("@clerk/nextjs/server").clerkMiddleware | null = null;
let createRouteMatcher: typeof import("@clerk/nextjs/server").createRouteMatcher | null = null;

if (hasClerkKey) {
  try {
    const clerk = require("@clerk/nextjs/server");
    clerkMiddleware = clerk.clerkMiddleware;
    createRouteMatcher = clerk.createRouteMatcher;
  } catch {
    // Clerk not available
  }
}

// Public routes that don't require authentication
const publicPaths = [
  "/",
  "/features",
  "/pricing",
  "/security",
  "/demo",
  "/login",
  "/signup",
  "/api/webhooks",
];

function isPublicPath(pathname: string): boolean {
  return publicPaths.some(
    (path) => pathname === path || pathname.startsWith(`${path}/`)
  );
}

export default async function middleware(req: NextRequest) {
  // If Clerk is not configured, allow all requests (dev mode)
  if (!hasClerkKey || !clerkMiddleware || !createRouteMatcher) {
    return NextResponse.next();
  }

  // Use Clerk middleware when configured
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

  return clerkMiddleware(async (auth, req) => {
    if (!isPublicRoute(req)) {
      await auth.protect();
    }
  })(req, {} as any);
}

export const config = {
  matcher: [
    // Skip Next.js internals and static files
    "/((?!_next|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|webp|png|gif|svg|ttf|woff2?|ico|csv|docx?|xlsx?|zip|webmanifest)).*)",
    // Always run for API routes
    "/(api|trpc)(.*)",
  ],
};
