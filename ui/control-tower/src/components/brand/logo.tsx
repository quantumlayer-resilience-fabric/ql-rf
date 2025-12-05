"use client";

import { cn } from "@/lib/utils";

interface LogoProps {
  variant?: "full" | "icon" | "text";
  size?: "sm" | "md" | "lg" | "xl";
  className?: string;
}

const sizeClasses = {
  sm: { icon: "h-6 w-6", text: "text-lg" },
  md: { icon: "h-8 w-8", text: "text-xl" },
  lg: { icon: "h-10 w-10", text: "text-2xl" },
  xl: { icon: "h-12 w-12", text: "text-3xl" },
};

// Extracted icon component to avoid creating components during render
function LogoSvg({ iconSize }: { iconSize: string }) {
  return (
    <svg
      viewBox="0 0 40 40"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={cn(iconSize, "flex-shrink-0")}
    >
      {/* Outer ring - representing multi-cloud coverage */}
      <circle
        cx="20"
        cy="20"
        r="18"
        stroke="currentColor"
        strokeWidth="2"
        className="text-brand"
      />
      {/* Inner hexagon - representing control/stability */}
      <path
        d="M20 6L32 13V27L20 34L8 27V13L20 6Z"
        stroke="currentColor"
        strokeWidth="2"
        fill="none"
        className="text-brand"
      />
      {/* Center point - command center */}
      <circle cx="20" cy="20" r="4" fill="currentColor" className="text-brand-accent" />
      {/* Connection lines - representing drift detection */}
      <line x1="20" y1="16" x2="20" y2="10" stroke="currentColor" strokeWidth="1.5" className="text-brand-accent" />
      <line x1="23.5" y1="22" x2="28" y2="25" stroke="currentColor" strokeWidth="1.5" className="text-brand-accent" />
      <line x1="16.5" y1="22" x2="12" y2="25" stroke="currentColor" strokeWidth="1.5" className="text-brand-accent" />
    </svg>
  );
}

export function Logo({ variant = "full", size = "md", className }: LogoProps) {
  const { icon: iconSize, text: textSize } = sizeClasses[size];

  if (variant === "icon") {
    return (
      <div className={cn("flex items-center", className)}>
        <LogoSvg iconSize={iconSize} />
      </div>
    );
  }

  if (variant === "text") {
    return (
      <div className={cn("flex items-center", className)}>
        <span className={cn("font-semibold tracking-tight text-foreground", textSize)}>
          QuantumLayer
        </span>
        <span className={cn("ml-1 font-light text-muted-foreground", textSize)}>
          RF
        </span>
      </div>
    );
  }

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <LogoSvg iconSize={iconSize} />
      <div className="flex items-baseline">
        <span className={cn("font-semibold tracking-tight text-foreground", textSize)}>
          QuantumLayer
        </span>
        <span className={cn("ml-1 font-light text-muted-foreground", textSize)}>
          RF
        </span>
      </div>
    </div>
  );
}

export function LogoIcon({ size = "md", className }: Omit<LogoProps, "variant">) {
  return <Logo variant="icon" size={size} className={className} />;
}
