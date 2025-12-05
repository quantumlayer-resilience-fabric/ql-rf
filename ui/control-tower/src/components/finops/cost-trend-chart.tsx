"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { CostTrend } from "@/lib/api-finops";
import { formatCurrency } from "@/lib/utils";
import { TrendingUp, TrendingDown } from "lucide-react";

interface CostTrendChartProps {
  data: CostTrend[];
  currency: string;
}

export function CostTrendChart({ data, currency }: CostTrendChartProps) {
  if (!data || data.length === 0) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center p-8">
          <p className="text-muted-foreground">No cost trend data available</p>
        </CardContent>
      </Card>
    );
  }

  // Calculate max value for scaling
  const maxCost = Math.max(...data.map((d) => d.cost));
  const minCost = Math.min(...data.map((d) => d.cost));

  // Calculate trend
  const firstCost = data[0]?.cost || 0;
  const lastCost = data[data.length - 1]?.cost || 0;
  const trendPercentage = firstCost !== 0 ? ((lastCost - firstCost) / firstCost) * 100 : 0;
  const isIncreasing = trendPercentage > 0;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">Cost Trend ({data.length} days)</CardTitle>
          <div className="flex items-center gap-2">
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
              {trendPercentage.toFixed(1)}%
            </span>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {/* Simple bar chart visualization */}
          <div className="flex items-end gap-1 h-48">
            {data.map((point, index) => {
              const height = maxCost > 0 ? (point.cost / maxCost) * 100 : 0;
              const date = new Date(point.date);
              const isWeekend = date.getDay() === 0 || date.getDay() === 6;

              return (
                <div
                  key={index}
                  className="flex-1 flex flex-col items-center group cursor-pointer"
                  title={`${date.toLocaleDateString()}: ${formatCurrency(point.cost, currency)}`}
                >
                  <div className="w-full flex items-end" style={{ height: "100%" }}>
                    <div
                      className={`w-full rounded-t transition-all duration-300 ${
                        isWeekend
                          ? "bg-brand-accent/30 hover:bg-brand-accent/50"
                          : "bg-brand-accent hover:bg-brand-accent/80"
                      }`}
                      style={{ height: `${height}%` }}
                    />
                  </div>
                </div>
              );
            })}
          </div>

          {/* Date range labels */}
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span>{new Date(data[0].date).toLocaleDateString()}</span>
            <span>{new Date(data[data.length - 1].date).toLocaleDateString()}</span>
          </div>

          {/* Cost range */}
          <div className="flex items-center justify-between text-sm">
            <div>
              <span className="text-muted-foreground">Min: </span>
              <span className="font-medium">{formatCurrency(minCost, currency)}</span>
            </div>
            <div>
              <span className="text-muted-foreground">Max: </span>
              <span className="font-medium">{formatCurrency(maxCost, currency)}</span>
            </div>
            <div>
              <span className="text-muted-foreground">Avg: </span>
              <span className="font-medium">
                {formatCurrency(
                  data.reduce((sum, d) => sum + d.cost, 0) / data.length,
                  currency
                )}
              </span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
