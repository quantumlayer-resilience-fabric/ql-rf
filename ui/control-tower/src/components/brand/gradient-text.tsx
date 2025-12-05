"use client";

import { cn } from "@/lib/utils";
import { ReactNode } from "react";

interface GradientTextProps {
  children: ReactNode;
  variant?: "ai" | "brand" | "quantum" | "status";
  className?: string;
  as?: "span" | "h1" | "h2" | "h3" | "h4" | "p";
}

export function GradientText({
  children,
  variant = "quantum",
  className,
  as: Component = "span",
}: GradientTextProps) {
  const gradientClasses = {
    quantum: "bg-gradient-to-r from-[#00ff88] to-[#8b5cf6]",
    ai: "bg-gradient-to-r from-[var(--ai-start)] via-[var(--ai-mid)] to-[var(--ai-end)]",
    brand: "bg-gradient-to-r from-[var(--quantum-green)] to-[var(--quantum-purple)]",
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
