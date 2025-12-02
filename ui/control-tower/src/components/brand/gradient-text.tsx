"use client";

import { cn } from "@/lib/utils";
import { ReactNode } from "react";

interface GradientTextProps {
  children: ReactNode;
  variant?: "ai" | "brand" | "status";
  className?: string;
  as?: "span" | "h1" | "h2" | "h3" | "h4" | "p";
}

export function GradientText({
  children,
  variant = "ai",
  className,
  as: Component = "span",
}: GradientTextProps) {
  const gradientClasses = {
    ai: "bg-gradient-to-r from-[var(--ai-start)] to-[var(--ai-end)]",
    brand: "bg-gradient-to-r from-[var(--brand)] to-[var(--brand-accent)]",
    status: "bg-gradient-to-r from-[var(--status-green)] via-[var(--status-amber)] to-[var(--status-red)]",
  };

  return (
    <Component
      className={cn(
        "bg-clip-text text-transparent",
        gradientClasses[variant],
        className
      )}
    >
      {children}
    </Component>
  );
}
