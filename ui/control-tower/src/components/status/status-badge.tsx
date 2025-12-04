"use client";

import { cn } from "@/lib/utils";

type StatusVariant = "success" | "warning" | "critical" | "neutral" | "info";

interface StatusBadgeProps {
  status: StatusVariant;
  children: React.ReactNode;
  pulse?: boolean;
  glow?: boolean;
  size?: "sm" | "md" | "lg";
  variant?: "default" | "outline" | "solid";
  className?: string;
}

const statusStyles: Record<StatusVariant, { bg: string; solid: string; outline: string; dot: string; glow: string }> = {
  success: {
    bg: "bg-status-green/10 text-status-green border-status-green/20",
    solid: "bg-status-green text-white border-status-green",
    outline: "bg-transparent text-status-green border-status-green/40",
    dot: "bg-status-green",
    glow: "shadow-[0_0_8px_rgba(5,150,105,0.4)]",
  },
  warning: {
    bg: "bg-status-amber/10 text-status-amber border-status-amber/20",
    solid: "bg-status-amber text-white border-status-amber",
    outline: "bg-transparent text-status-amber border-status-amber/40",
    dot: "bg-status-amber",
    glow: "shadow-[0_0_8px_rgba(217,119,6,0.4)]",
  },
  critical: {
    bg: "bg-status-red/10 text-status-red border-status-red/20",
    solid: "bg-status-red text-white border-status-red",
    outline: "bg-transparent text-status-red border-status-red/40",
    dot: "bg-status-red",
    glow: "shadow-[0_0_12px_rgba(220,38,38,0.5)] animate-glow-critical",
  },
  neutral: {
    bg: "bg-muted text-muted-foreground border-border",
    solid: "bg-muted-foreground text-background border-muted-foreground",
    outline: "bg-transparent text-muted-foreground border-muted-foreground/40",
    dot: "bg-muted-foreground",
    glow: "",
  },
  info: {
    bg: "bg-brand-accent/10 text-brand-accent border-brand-accent/20",
    solid: "bg-brand-accent text-white border-brand-accent",
    outline: "bg-transparent text-brand-accent border-brand-accent/40",
    dot: "bg-brand-accent",
    glow: "shadow-[0_0_8px_rgba(37,99,235,0.4)]",
  },
};

const sizeStyles = {
  sm: "px-2 py-0.5 text-xs gap-1",
  md: "px-2.5 py-1 text-sm gap-1.5",
  lg: "px-3 py-1.5 text-base gap-2",
};

const dotSizeStyles = {
  sm: "h-1.5 w-1.5",
  md: "h-2 w-2",
  lg: "h-2.5 w-2.5",
};

export function StatusBadge({
  status,
  children,
  pulse = false,
  glow = false,
  size = "md",
  variant = "default",
  className,
}: StatusBadgeProps) {
  const config = statusStyles[status];

  // Get the appropriate background style based on variant
  const variantStyle = variant === "solid"
    ? config.solid
    : variant === "outline"
      ? config.outline
      : config.bg;

  // Auto-enable glow for critical status unless explicitly disabled
  const shouldGlow = glow || (status === "critical" && glow !== false);

  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border font-medium transition-all duration-200",
        variantStyle,
        sizeStyles[size],
        pulse && "animate-pulse-status",
        shouldGlow && config.glow,
        className
      )}
    >
      <span
        className={cn(
          "rounded-full transition-all",
          dotSizeStyles[size],
          config.dot,
          pulse && "animate-ping-slow"
        )}
      />
      {children}
    </span>
  );
}

// Compact status indicator (just the dot)
interface StatusDotProps {
  status: StatusVariant;
  size?: "sm" | "md" | "lg";
  pulse?: boolean;
  className?: string;
}

export function StatusDot({ status, size = "md", pulse = false, className }: StatusDotProps) {
  const config = statusStyles[status];

  return (
    <span
      className={cn(
        "inline-block rounded-full",
        dotSizeStyles[size],
        config.dot,
        pulse && "animate-ping-slow",
        className
      )}
    />
  );
}
