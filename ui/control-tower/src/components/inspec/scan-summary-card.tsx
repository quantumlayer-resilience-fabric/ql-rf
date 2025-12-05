"use client";

import { Card, CardContent } from "@/components/ui/card";
import { CheckCircle, XCircle, MinusCircle } from "lucide-react";
import { cn } from "@/lib/utils";

interface ScanSummaryCardProps {
  type: "passed" | "failed" | "skipped";
  count: number;
  total: number;
}

export function ScanSummaryCard({ type, count, total }: ScanSummaryCardProps) {
  const config = {
    passed: {
      icon: <CheckCircle className="h-8 w-8" />,
      color: "text-status-green",
      bgColor: "bg-status-green/10",
      borderColor: "border-status-green/20",
      label: "Passed",
    },
    failed: {
      icon: <XCircle className="h-8 w-8" />,
      color: "text-status-red",
      bgColor: "bg-status-red/10",
      borderColor: "border-status-red/20",
      label: "Failed",
    },
    skipped: {
      icon: <MinusCircle className="h-8 w-8" />,
      color: "text-muted-foreground",
      bgColor: "bg-muted",
      borderColor: "border-border",
      label: "Skipped",
    },
  };

  const { icon, color, bgColor, borderColor, label } = config[type];
  const percentage = total > 0 ? ((count / total) * 100).toFixed(1) : "0";

  return (
    <Card className={cn("border-l-4", borderColor)}>
      <CardContent className="p-6">
        <div className="flex items-center gap-4">
          <div className={cn("rounded-xl p-3", bgColor)}>
            <div className={color}>{icon}</div>
          </div>
          <div className="flex-1">
            <p className="text-sm font-medium text-muted-foreground">{label}</p>
            <div className="flex items-baseline gap-2 mt-1">
              <p className={cn("text-3xl font-bold", color)}>{count}</p>
              <span className="text-sm text-muted-foreground">
                ({percentage}%)
              </span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
