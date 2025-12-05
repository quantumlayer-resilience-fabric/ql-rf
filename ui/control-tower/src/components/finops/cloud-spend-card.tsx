"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatCurrency } from "@/lib/utils";
import { Cloud, TrendingUp, TrendingDown } from "lucide-react";

interface CloudSpendCardProps {
  cloud: string;
  amount: number;
  currency: string;
  percentage: number;
  trend?: number;
}

const cloudColors: Record<string, { bg: string; text: string; icon: string }> = {
  aws: {
    bg: "bg-[#FF9900]/10",
    text: "text-[#FF9900]",
    icon: "bg-[#FF9900]",
  },
  azure: {
    bg: "bg-[#0078D4]/10",
    text: "text-[#0078D4]",
    icon: "bg-[#0078D4]",
  },
  gcp: {
    bg: "bg-[#4285F4]/10",
    text: "text-[#4285F4]",
    icon: "bg-[#4285F4]",
  },
  vsphere: {
    bg: "bg-status-purple/10",
    text: "text-status-purple",
    icon: "bg-status-purple",
  },
};

const cloudLabels: Record<string, string> = {
  aws: "AWS",
  azure: "Azure",
  gcp: "Google Cloud",
  vsphere: "vSphere",
};

export function CloudSpendCard({
  cloud,
  amount,
  currency,
  percentage,
  trend,
}: CloudSpendCardProps) {
  const colors = cloudColors[cloud.toLowerCase()] || {
    bg: "bg-muted",
    text: "text-foreground",
    icon: "bg-muted-foreground",
  };

  const hasTrend = trend !== undefined;
  const isIncreasing = trend ? trend > 0 : false;

  return (
    <Card className="hover:border-brand-accent transition-colors">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            {cloudLabels[cloud.toLowerCase()] || cloud}
          </CardTitle>
          <div className={`rounded-lg p-2 ${colors.bg}`}>
            <Cloud className={`h-4 w-4 ${colors.text}`} />
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-2">
        <div className="flex items-end justify-between">
          <div>
            <div className="text-2xl font-bold tabular-nums">
              {formatCurrency(amount, currency)}
            </div>
            <div className="text-xs text-muted-foreground mt-1">
              {percentage.toFixed(1)}% of total
            </div>
          </div>
          {hasTrend && (
            <div className="flex items-center gap-1">
              {isIncreasing ? (
                <TrendingUp className="h-4 w-4 text-status-red" />
              ) : (
                <TrendingDown className="h-4 w-4 text-status-green" />
              )}
              <span
                className={`text-sm font-medium ${
                  isIncreasing ? "text-status-red" : "text-status-green"
                }`}
              >
                {isIncreasing ? "+" : ""}
                {Math.abs(trend || 0).toFixed(1)}%
              </span>
            </div>
          )}
        </div>

        {/* Visual bar */}
        <div className="h-2 w-full bg-muted rounded-full overflow-hidden">
          <div
            className={`h-full ${colors.icon} transition-all duration-500`}
            style={{ width: `${Math.min(100, percentage)}%` }}
          />
        </div>
      </CardContent>
    </Card>
  );
}
