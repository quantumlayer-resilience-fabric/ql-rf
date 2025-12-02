"use client";

import { cn } from "@/lib/utils";

type StatusVariant = "success" | "warning" | "critical" | "neutral" | "info";

interface StatusBadgeProps {
  status: StatusVariant;
  children: React.ReactNode;
  pulse?: boolean;
  size?: "sm" | "md" | "lg";
  className?: string;
}

const statusStyles: Record<StatusVariant, string> = {
  success: "bg-status-green-bg text-status-green border-status-green/20",
  warning: "bg-status-amber-bg text-status-amber border-status-amber/20",
  critical: "bg-status-red-bg text-status-red border-status-red/20",
  neutral: "bg-muted text-muted-foreground border-border",
  info: "bg-brand-accent/10 text-brand-accent border-brand-accent/20",
};

const sizeStyles = {
  sm: "px-2 py-0.5 text-xs",
  md: "px-2.5 py-1 text-sm",
  lg: "px-3 py-1.5 text-base",
};

export function StatusBadge({
  status,
  children,
  pulse = false,
  size = "md",
  className,
}: StatusBadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border font-medium",
        statusStyles[status],
        sizeStyles[size],
        pulse && "animate-pulse-status",
        className
      )}
    >
      <span
        className={cn(
          "mr-1.5 h-1.5 w-1.5 rounded-full",
          status === "success" && "bg-status-green",
          status === "warning" && "bg-status-amber",
          status === "critical" && "bg-status-red",
          status === "neutral" && "bg-muted-foreground",
          status === "info" && "bg-brand-accent"
        )}
      />
      {children}
    </span>
  );
}
