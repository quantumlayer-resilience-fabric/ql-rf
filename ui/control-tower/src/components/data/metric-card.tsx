"use client";

import { cn } from "@/lib/utils";
import { Card, CardContent } from "@/components/ui/card";
import { TrendingUp, TrendingDown, Minus } from "lucide-react";

interface MetricCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  trend?: {
    direction: "up" | "down" | "neutral";
    value: string;
    period?: string;
  };
  status?: "success" | "warning" | "critical" | "neutral";
  icon?: React.ReactNode;
  className?: string;
  animate?: boolean;
}

const statusConfig = {
  success: {
    text: "text-status-green",
    bg: "bg-status-green/10",
    border: "border-status-green/20",
    iconBg: "bg-status-green/15",
    iconText: "text-status-green",
  },
  warning: {
    text: "text-status-amber",
    bg: "bg-status-amber/10",
    border: "border-status-amber/20",
    iconBg: "bg-status-amber/15",
    iconText: "text-status-amber",
  },
  critical: {
    text: "text-status-red",
    bg: "bg-status-red/10",
    border: "border-status-red/20",
    iconBg: "bg-status-red/15",
    iconText: "text-status-red",
  },
  neutral: {
    text: "text-foreground",
    bg: "bg-muted/50",
    border: "border-border",
    iconBg: "bg-muted",
    iconText: "text-muted-foreground",
  },
};

const trendConfig = {
  up: {
    text: "text-status-green",
    bg: "bg-status-green/10",
  },
  down: {
    text: "text-status-red",
    bg: "bg-status-red/10",
  },
  neutral: {
    text: "text-muted-foreground",
    bg: "bg-muted",
  },
};

export function MetricCard({
  title,
  value,
  subtitle,
  trend,
  status = "neutral",
  icon,
  className,
  animate = true,
}: MetricCardProps) {
  const config = statusConfig[status];
  const trendStyle = trend ? trendConfig[trend.direction] : null;

  return (
    <Card
      variant="elevated"
      hover="lift"
      className={cn(
        "relative overflow-hidden transition-all duration-300",
        animate && "animate-in fade-in-0 slide-in-from-bottom-2 duration-500",
        className
      )}
    >
      <CardContent className="p-6">
        {/* Background gradient accent for non-neutral status */}
        {status !== "neutral" && (
          <div
            className={cn(
              "absolute inset-0 opacity-30",
              status === "success" && "bg-gradient-to-br from-status-green/5 to-transparent",
              status === "warning" && "bg-gradient-to-br from-status-amber/5 to-transparent",
              status === "critical" && "bg-gradient-to-br from-status-red/5 to-transparent"
            )}
          />
        )}

        <div className="relative flex items-start justify-between">
          <div className="space-y-1">
            <p
              className="text-sm font-medium text-muted-foreground"
              style={{ fontFamily: "var(--font-body)" }}
            >
              {title}
            </p>
            <div className="flex items-baseline gap-2">
              <p
                className={cn(
                  "text-3xl font-bold tracking-tight transition-colors",
                  config.text
                )}
                style={{ fontFamily: "var(--font-display)" }}
              >
                {value}
              </p>
              {subtitle && (
                <span className="text-sm text-muted-foreground">{subtitle}</span>
              )}
            </div>
          </div>

          {icon && (
            <div
              className={cn(
                "rounded-xl p-2.5 transition-all duration-300",
                config.iconBg,
                config.iconText
              )}
            >
              {icon}
            </div>
          )}
        </div>

        {trend && trend.value && (
          <div className="relative mt-4 flex items-center gap-2">
            <div
              className={cn(
                "flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
                trendStyle?.bg,
                trendStyle?.text
              )}
            >
              {trend.direction === "up" && (
                <TrendingUp className="h-3 w-3" />
              )}
              {trend.direction === "down" && (
                <TrendingDown className="h-3 w-3" />
              )}
              {trend.direction === "neutral" && (
                <Minus className="h-3 w-3" />
              )}
              <span>{trend.value}</span>
            </div>
            {trend.period && (
              <span className="text-xs text-muted-foreground">
                {trend.period}
              </span>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
