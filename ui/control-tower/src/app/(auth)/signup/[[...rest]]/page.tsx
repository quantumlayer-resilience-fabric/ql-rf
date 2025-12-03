"use client";

import dynamic from "next/dynamic";

// Force dynamic rendering to skip prerendering
export const dynamic_rendering = "force-dynamic";

// Dev bypass - set NEXT_PUBLIC_DEV_AUTH_BYPASS=true to skip Clerk entirely
const devAuthBypass = process.env.NEXT_PUBLIC_DEV_AUTH_BYPASS === "true";

// Check if Clerk is configured (at module level for tree-shaking)
const hasClerkKey =
  !devAuthBypass &&
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY &&
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY.startsWith("pk_") &&
  !process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY.includes("xxxxx");

// Only load Clerk SignUp component if we have a valid key
const ClerkSignUp = hasClerkKey
  ? dynamic(() => import("@clerk/nextjs").then((mod) => mod.SignUp), {
      loading: () => (
        <div className="flex items-center justify-center min-h-screen">
          <div className="animate-pulse">Loading...</div>
        </div>
      ),
    })
  : null;

function DevModeSignup() {
  return (
    <div className="flex flex-col items-center justify-center min-h-screen bg-background">
      <div className="p-8 border rounded-lg shadow-lg max-w-md text-center">
        <h1 className="text-2xl font-bold mb-4">Development Mode</h1>
        <p className="text-muted-foreground mb-6">
          Authentication is disabled. To enable Clerk authentication, add a
          valid NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY to your environment.
        </p>
        <a
          href="/overview"
          className="inline-block px-6 py-3 bg-primary text-primary-foreground rounded-lg hover:opacity-90 transition-opacity"
        >
          Continue to Dashboard
        </a>
      </div>
    </div>
  );
}

export default function SignupPage() {
  if (!ClerkSignUp) {
    return <DevModeSignup />;
  }

  return (
    <ClerkSignUp
      appearance={{
        elements: {
          rootBox: "mx-auto",
          card: "border-0 shadow-2xl",
          headerTitle: "text-2xl font-bold",
          headerSubtitle: "text-muted-foreground",
          socialButtonsBlockButton: "border-border hover:bg-accent",
          formButtonPrimary: "bg-brand hover:bg-brand-light",
          footerActionLink: "text-brand-accent hover:text-brand-accent-light",
        },
      }}
      routing="path"
      path="/signup"
      signInUrl="/login"
      forceRedirectUrl="/onboarding"
    />
  );
}
