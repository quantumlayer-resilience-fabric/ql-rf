"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { CostBudget } from "@/lib/api-finops";
import { formatCurrency } from "@/lib/utils";
import { AlertTriangle, CheckCircle, TrendingUp } from "lucide-react";

interface BudgetProgressCardProps {
  budget: CostBudget;
  onClick?: () => void;
}

export function BudgetProgressCard({ budget, onClick }: BudgetProgressCardProps) {
  const utilizationPercentage = (budget.currentSpend / budget.amount) * 100;
  const isOverBudget = utilizationPercentage > 100;
  const isNearLimit = utilizationPercentage >= budget.alertThreshold;

  const status = isOverBudget
    ? "critical"
    : isNearLimit
    ? "warning"
    : "success";

  const statusColors = {
    success: {
      bg: "bg-status-green/10",
      text: "text-status-green",
      border: "border-status-green/30",
    },
    warning: {
      bg: "bg-status-amber/10",
      text: "text-status-amber",
      border: "border-status-amber/30",
    },
    critical: {
      bg: "bg-status-red/10",
      text: "text-status-red",
      border: "border-status-red/30",
    },
  };

  const colors = statusColors[status];

  return (
    <Card
      className={`cursor-pointer hover:border-brand-accent transition-colors ${
        isOverBudget ? "border-l-4 border-l-status-red" : ""
      }`}
      onClick={onClick}
    >
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <CardTitle className="text-base flex items-center gap-2">
              {budget.name}
              {!budget.active && (
                <Badge variant="outline" className="text-xs">
                  Inactive
                </Badge>
              )}
            </CardTitle>
            {budget.description && (
              <p className="text-xs text-muted-foreground mt-1">
                {budget.description}
              </p>
            )}
          </div>
          {isOverBudget ? (
            <AlertTriangle className="h-5 w-5 text-status-red" />
          ) : isNearLimit ? (
            <TrendingUp className="h-5 w-5 text-status-amber" />
          ) : (
            <CheckCircle className="h-5 w-5 text-status-green" />
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">
            {budget.scope === "organization"
              ? "Organization"
              : budget.scope === "cloud"
              ? budget.scopeValue?.toUpperCase()
              : budget.scopeValue}
          </span>
          <Badge variant="outline" className="text-xs">
            {budget.period}
          </Badge>
        </div>

        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-2xl font-bold tabular-nums">
              {utilizationPercentage.toFixed(0)}%
            </span>
            <span className="text-sm text-muted-foreground">
              utilized
            </span>
          </div>
          <Progress
            value={Math.min(100, utilizationPercentage)}
            className="h-3"
            indicatorClassName={
              isOverBudget
                ? "bg-status-red"
                : isNearLimit
                ? "bg-status-amber"
                : "bg-status-green"
            }
          />
          <div className="flex items-center justify-between text-sm">
            <span className="font-medium">
              {formatCurrency(budget.currentSpend, budget.currency)}
            </span>
            <span className="text-muted-foreground">
              of {formatCurrency(budget.amount, budget.currency)}
            </span>
          </div>
        </div>

        {isOverBudget && (
          <div className={`rounded-lg p-2 ${colors.bg} ${colors.border} border`}>
            <p className={`text-xs font-medium ${colors.text}`}>
              Over budget by{" "}
              {formatCurrency(budget.currentSpend - budget.amount, budget.currency)}
            </p>
          </div>
        )}

        {isNearLimit && !isOverBudget && (
          <div className={`rounded-lg p-2 ${colors.bg} ${colors.border} border`}>
            <p className={`text-xs font-medium ${colors.text}`}>
              Alert threshold ({budget.alertThreshold}%) reached
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
