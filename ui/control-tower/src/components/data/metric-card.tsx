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
}

const statusColors = {
  success: "text-status-green",
  warning: "text-status-amber",
  critical: "text-status-red",
  neutral: "text-muted-foreground",
};

const trendColors = {
  up: "text-status-green",
  down: "text-status-red",
  neutral: "text-muted-foreground",
};

export function MetricCard({
  title,
  value,
  subtitle,
  trend,
  status = "neutral",
  icon,
  className,
}: MetricCardProps) {
  return (
    <Card className={cn("", className)}>
      <CardContent className="p-6">
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <p className="text-sm font-medium text-muted-foreground">{title}</p>
            <div className="flex items-baseline gap-2">
              <p
                className={cn(
                  "text-3xl font-bold tracking-tight",
                  statusColors[status]
                )}
              >
                {value}
              </p>
              {subtitle && (
                <span className="text-sm text-muted-foreground">{subtitle}</span>
              )}
            </div>
          </div>
          {icon && (
            <div className="rounded-lg bg-muted p-2 text-muted-foreground">
              {icon}
            </div>
          )}
        </div>

        {trend && (
          <div className="mt-4 flex items-center gap-1">
            {trend.direction === "up" && (
              <TrendingUp className={cn("h-4 w-4", trendColors.up)} />
            )}
            {trend.direction === "down" && (
              <TrendingDown className={cn("h-4 w-4", trendColors.down)} />
            )}
            {trend.direction === "neutral" && (
              <Minus className={cn("h-4 w-4", trendColors.neutral)} />
            )}
            <span className={cn("text-sm font-medium", trendColors[trend.direction])}>
              {trend.value}
            </span>
            {trend.period && (
              <span className="text-sm text-muted-foreground">
                {trend.period}
              </span>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
