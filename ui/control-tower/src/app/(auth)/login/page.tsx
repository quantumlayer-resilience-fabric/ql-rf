"use client";

import { SignIn } from "@clerk/nextjs";

export default function LoginPage() {
  return (
    <SignIn
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
      path="/login"
      signUpUrl="/signup"
      forceRedirectUrl="/overview"
    />
  );
}
