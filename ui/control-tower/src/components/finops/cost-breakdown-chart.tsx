"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { CostBreakdownItem } from "@/lib/api-finops";
import { formatCurrency } from "@/lib/utils";

interface CostBreakdownChartProps {
  items: CostBreakdownItem[];
  totalCost: number;
  currency: string;
  title?: string;
}

const COLORS = [
  "bg-brand-accent",
  "bg-status-blue",
  "bg-status-green",
  "bg-status-amber",
  "bg-status-red",
  "bg-status-purple",
];

export function CostBreakdownChart({
  items,
  totalCost,
  currency,
  title = "Cost Breakdown",
}: CostBreakdownChartProps) {
  if (!items || items.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{title}</CardTitle>
        </CardHeader>
        <CardContent className="flex items-center justify-center p-8">
          <p className="text-muted-foreground">No breakdown data available</p>
        </CardContent>
      </Card>
    );
  }

  // Sort items by cost descending
  const sortedItems = [...items].sort((a, b) => b.cost - a.cost);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Horizontal stacked bar chart */}
        <div className="flex h-8 w-full overflow-hidden rounded-lg">
          {sortedItems.map((item, index) => (
            <div
              key={item.name}
              className={`${COLORS[index % COLORS.length]} transition-all duration-300 hover:opacity-80 cursor-pointer`}
              style={{ width: `${item.percentage}%` }}
              title={`${item.name}: ${formatCurrency(item.cost, currency)} (${item.percentage.toFixed(1)}%)`}
            />
          ))}
        </div>

        {/* Legend and details */}
        <div className="space-y-2">
          {sortedItems.map((item, index) => (
            <div
              key={item.name}
              className="flex items-center justify-between text-sm hover:bg-muted/50 rounded p-2 transition-colors"
            >
              <div className="flex items-center gap-2">
                <div className={`h-3 w-3 rounded ${COLORS[index % COLORS.length]}`} />
                <span className="font-medium">{item.name}</span>
              </div>
              <div className="flex items-center gap-4">
                <span className="text-muted-foreground">
                  {item.percentage.toFixed(1)}%
                </span>
                <span className="font-semibold tabular-nums">
                  {formatCurrency(item.cost, currency)}
                </span>
              </div>
            </div>
          ))}
        </div>

        {/* Total */}
        <div className="flex items-center justify-between border-t pt-2">
          <span className="font-semibold">Total</span>
          <span className="text-lg font-bold tabular-nums">
            {formatCurrency(totalCost, currency)}
          </span>
        </div>
      </CardContent>
    </Card>
  );
}
