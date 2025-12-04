"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  DollarSign,
  Clock,
  ShieldCheck,
  TrendingUp,
  AlertTriangle,
  Zap
} from "lucide-react";

interface ValueMetric {
  label: string;
  value: string;
  subtext: string;
  icon: React.ReactNode;
  color: string;
}

interface ValueDeliveredCardProps {
  // These would come from API in real implementation
  incidentsPreventedCount?: number;
  incidentsPreventedValue?: number;
  hoursAutoRemediated?: number;
  hourlyRate?: number;
  complianceViolationsCaught?: number;
  violationCost?: number;
  driftFixesApplied?: number;
  mttrReduction?: number;
  className?: string;
}

export function ValueDeliveredCard({
  incidentsPreventedCount = 12,
  incidentsPreventedValue = 45000,
  hoursAutoRemediated = 156,
  hourlyRate = 150,
  complianceViolationsCaught = 34,
  violationCost = 2500,
  driftFixesApplied = 89,
  mttrReduction = 67,
  className,
}: ValueDeliveredCardProps) {
  // Calculate total value
  const incidentSavings = incidentsPreventedCount * (incidentsPreventedValue / incidentsPreventedCount);
  const timeSavings = hoursAutoRemediated * hourlyRate;
  const complianceSavings = complianceViolationsCaught * violationCost;
  const totalValue = incidentSavings + timeSavings + complianceSavings;

  const metrics: ValueMetric[] = [
    {
      label: "Incidents Prevented",
      value: incidentsPreventedCount.toString(),
      subtext: `$${formatCurrency(incidentsPreventedValue)} saved`,
      icon: <AlertTriangle className="h-4 w-4" />,
      color: "text-status-red",
    },
    {
      label: "Hours Automated",
      value: hoursAutoRemediated.toString(),
      subtext: `$${formatCurrency(timeSavings)} in labor`,
      icon: <Clock className="h-4 w-4" />,
      color: "text-status-amber",
    },
    {
      label: "Violations Caught",
      value: complianceViolationsCaught.toString(),
      subtext: `$${formatCurrency(complianceSavings)} in fines avoided`,
      icon: <ShieldCheck className="h-4 w-4" />,
      color: "text-status-green",
    },
    {
      label: "Drift Fixes Applied",
      value: driftFixesApplied.toString(),
      subtext: `${mttrReduction}% faster MTTR`,
      icon: <Zap className="h-4 w-4" />,
      color: "text-brand-accent",
    },
  ];

  return (
    <Card variant="elevated" className={className}>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle
            className="text-base flex items-center gap-2"
            style={{ fontFamily: "var(--font-display)" }}
          >
            <TrendingUp className="h-4 w-4 text-status-green" />
            Value Delivered
          </CardTitle>
          <span className="text-xs text-muted-foreground">This Month</span>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Total Value Hero */}
        <div className="rounded-lg bg-gradient-to-br from-status-green/10 to-status-green/5 border border-status-green/20 p-4 text-center">
          <div className="flex items-center justify-center gap-2 text-status-green">
            <DollarSign className="h-6 w-6" />
            <span
              className="text-3xl font-bold tabular-nums"
              style={{ fontFamily: "var(--font-display)" }}
            >
              {formatCurrency(totalValue)}
            </span>
          </div>
          <p className="mt-1 text-sm text-muted-foreground">
            Total estimated savings
          </p>
        </div>

        {/* Metric Breakdown */}
        <div className="grid grid-cols-2 gap-3">
          {metrics.map((metric, index) => (
            <div
              key={metric.label}
              className="rounded-lg border border-border p-3 transition-all hover:border-border/80 hover:shadow-sm animate-in fade-in-0"
              style={{ animationDelay: `${index * 100}ms`, animationFillMode: 'backwards' }}
            >
              <div className="flex items-center gap-2 mb-1">
                <span className={metric.color}>{metric.icon}</span>
                <span className="text-xs text-muted-foreground">{metric.label}</span>
              </div>
              <div
                className="text-xl font-bold tabular-nums"
                style={{ fontFamily: "var(--font-display)" }}
              >
                {metric.value}
              </div>
              <div className="text-xs text-muted-foreground">{metric.subtext}</div>
            </div>
          ))}
        </div>

        {/* ROI Callout */}
        <div className="flex items-center justify-between rounded-lg bg-muted/50 px-3 py-2 text-sm">
          <span className="text-muted-foreground">Estimated ROI</span>
          <span className="font-semibold text-status-green">
            {calculateROI(totalValue)}x
          </span>
        </div>
      </CardContent>
    </Card>
  );
}

function formatCurrency(value: number): string {
  if (value >= 1000000) {
    return `${(value / 1000000).toFixed(1)}M`;
  }
  if (value >= 1000) {
    return `${(value / 1000).toFixed(0)}K`;
  }
  return value.toFixed(0);
}

function calculateROI(savings: number): string {
  // Assume $2K/month subscription cost for ROI calculation
  const monthlyCost = 2000;
  const roi = savings / monthlyCost;
  return roi.toFixed(1);
}
