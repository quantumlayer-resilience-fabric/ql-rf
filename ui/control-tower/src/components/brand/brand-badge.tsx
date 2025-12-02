"use client";

import { cn } from "@/lib/utils";
import { Shield, Award, CheckCircle, Lock } from "lucide-react";

type BadgeType = "soc2" | "iso27001" | "gdpr" | "hipaa" | "slsa" | "cosign";

interface BrandBadgeProps {
  type: BadgeType;
  size?: "sm" | "md" | "lg";
  showLabel?: boolean;
  className?: string;
}

const badgeConfig: Record<BadgeType, { icon: typeof Shield; label: string; color: string }> = {
  soc2: {
    icon: Shield,
    label: "SOC 2 Type II",
    color: "text-blue-600 dark:text-blue-400",
  },
  iso27001: {
    icon: Award,
    label: "ISO 27001",
    color: "text-emerald-600 dark:text-emerald-400",
  },
  gdpr: {
    icon: Lock,
    label: "GDPR",
    color: "text-violet-600 dark:text-violet-400",
  },
  hipaa: {
    icon: Shield,
    label: "HIPAA",
    color: "text-rose-600 dark:text-rose-400",
  },
  slsa: {
    icon: CheckCircle,
    label: "SLSA Level 3",
    color: "text-amber-600 dark:text-amber-400",
  },
  cosign: {
    icon: Lock,
    label: "Cosign Verified",
    color: "text-cyan-600 dark:text-cyan-400",
  },
};

const sizeClasses = {
  sm: { icon: "h-4 w-4", text: "text-xs", gap: "gap-1", padding: "px-2 py-1" },
  md: { icon: "h-5 w-5", text: "text-sm", gap: "gap-1.5", padding: "px-3 py-1.5" },
  lg: { icon: "h-6 w-6", text: "text-base", gap: "gap-2", padding: "px-4 py-2" },
};

export function BrandBadge({
  type,
  size = "md",
  showLabel = true,
  className,
}: BrandBadgeProps) {
  const config = badgeConfig[type];
  const sizes = sizeClasses[size];
  const Icon = config.icon;

  return (
    <div
      className={cn(
        "inline-flex items-center rounded-full border border-border bg-card",
        sizes.gap,
        sizes.padding,
        className
      )}
    >
      <Icon className={cn(sizes.icon, config.color)} />
      {showLabel && (
        <span className={cn("font-medium text-muted-foreground", sizes.text)}>
          {config.label}
        </span>
      )}
    </div>
  );
}

export function TrustBadgeRow({ className }: { className?: string }) {
  return (
    <div className={cn("flex flex-wrap items-center gap-3", className)}>
      <BrandBadge type="soc2" size="sm" />
      <BrandBadge type="iso27001" size="sm" />
      <BrandBadge type="gdpr" size="sm" />
    </div>
  );
}
