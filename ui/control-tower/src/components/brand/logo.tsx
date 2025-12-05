"use client";

import { cn } from "@/lib/utils";

interface LogoProps {
  variant?: "full" | "icon" | "text";
  size?: "sm" | "md" | "lg" | "xl";
  className?: string;
}

const sizeClasses = {
  sm: { icon: "h-6 w-6", text: "text-lg", subtext: "text-xs" },
  md: { icon: "h-8 w-8", text: "text-xl", subtext: "text-sm" },
  lg: { icon: "h-10 w-10", text: "text-2xl", subtext: "text-base" },
  xl: { icon: "h-12 w-12", text: "text-3xl", subtext: "text-lg" },
};

// QuantumLayer Logo - Hexagon with quantum gradient
function LogoSvg({ iconSize }: { iconSize: string }) {
  return (
    <svg
      viewBox="0 0 40 40"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={cn(iconSize, "flex-shrink-0")}
    >
      <defs>
        <linearGradient id="quantumGradient" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#00ff88" />
          <stop offset="100%" stopColor="#8b5cf6" />
        </linearGradient>
        <linearGradient id="quantumGradientStroke" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#00ff88" />
          <stop offset="50%" stopColor="#8b5cf6" />
          <stop offset="100%" stopColor="#3b82f6" />
        </linearGradient>
      </defs>
      {/* Outer hexagon - QuantumLayer signature shape */}
      <path
        d="M20 2L36 11V29L20 38L4 29V11L20 2Z"
        stroke="url(#quantumGradientStroke)"
        strokeWidth="2"
        fill="none"
      />
      {/* Inner hexagon - representing resilience/stability */}
      <path
        d="M20 8L30 14V26L20 32L10 26V14L20 8Z"
        stroke="url(#quantumGradient)"
        strokeWidth="1.5"
        fill="none"
        opacity="0.6"
      />
      {/* Center node - command center */}
      <circle cx="20" cy="20" r="4" fill="url(#quantumGradient)" />
      {/* Connection lines - representing resilience network */}
      <line x1="20" y1="16" x2="20" y2="10" stroke="#00ff88" strokeWidth="1.5" />
      <line x1="23.5" y1="22" x2="28" y2="25" stroke="#8b5cf6" strokeWidth="1.5" />
      <line x1="16.5" y1="22" x2="12" y2="25" stroke="#3b82f6" strokeWidth="1.5" />
      {/* Pulse effect dots at line ends */}
      <circle cx="20" cy="10" r="1.5" fill="#00ff88" />
      <circle cx="28" cy="25" r="1.5" fill="#8b5cf6" />
      <circle cx="12" cy="25" r="1.5" fill="#3b82f6" />
    </svg>
  );
}

export function Logo({ variant = "full", size = "md", className }: LogoProps) {
  const { icon: iconSize, text: textSize, subtext: subtextSize } = sizeClasses[size];

  if (variant === "icon") {
    return (
      <div className={cn("flex items-center", className)}>
        <LogoSvg iconSize={iconSize} />
      </div>
    );
  }

  if (variant === "text") {
    return (
      <div className={cn("flex flex-col", className)}>
        <span className={cn("font-bold tracking-tight gradient-quantum-text", textSize)}>
          QuantumLayer
        </span>
        <span className={cn("font-medium text-muted-foreground tracking-wide uppercase", subtextSize)}>
          Resilience Fabric
        </span>
      </div>
    );
  }

  return (
    <div className={cn("flex items-center gap-3", className)}>
      <LogoSvg iconSize={iconSize} />
      <div className="flex flex-col">
        <span className={cn("font-bold tracking-tight gradient-quantum-text", textSize)}>
          QuantumLayer
        </span>
        <span className={cn("font-medium text-muted-foreground tracking-wide uppercase", subtextSize)}>
          Resilience Fabric
        </span>
      </div>
    </div>
  );
}

export function LogoIcon({ size = "md", className }: Omit<LogoProps, "variant">) {
  return <Logo variant="icon" size={size} className={className} />;
}
