"use client";

import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { CostRecommendation } from "@/lib/api-finops";
import { formatCurrency } from "@/lib/utils";
import {
  TrendingDown,
  Server,
  DollarSign,
  Zap,
  CheckCircle,
  XCircle,
} from "lucide-react";

interface RecommendationCardProps {
  recommendation: CostRecommendation;
  onApply?: (id: string) => void;
  onDismiss?: (id: string) => void;
}

const typeIcons: Record<string, typeof Server> = {
  rightsizing: Server,
  reserved_instances: DollarSign,
  spot_instances: Zap,
  idle_resources: Server,
  storage_optimization: Server,
  unused_volumes: Server,
  old_snapshots: Server,
};

const typeLabels: Record<string, string> = {
  rightsizing: "Rightsizing",
  reserved_instances: "Reserved Instances",
  spot_instances: "Spot Instances",
  idle_resources: "Idle Resources",
  storage_optimization: "Storage",
  unused_volumes: "Unused Volumes",
  old_snapshots: "Old Snapshots",
};

const priorityColors = {
  high: {
    badge: "bg-status-red/10 text-status-red border-status-red/30",
    border: "border-l-status-red",
  },
  medium: {
    badge: "bg-status-amber/10 text-status-amber border-status-amber/30",
    border: "border-l-status-amber",
  },
  low: {
    badge: "bg-status-blue/10 text-status-blue border-status-blue/30",
    border: "border-l-status-blue",
  },
};

export function RecommendationCard({
  recommendation,
  onApply,
  onDismiss,
}: RecommendationCardProps) {
  const Icon = typeIcons[recommendation.type] || Server;
  const priorityStyle = priorityColors[recommendation.priority];
  const isApplied = recommendation.status === "applied";
  const isDismissed = recommendation.status === "dismissed";

  return (
    <Card
      className={`border-l-4 ${priorityStyle.border} ${
        isDismissed ? "opacity-60" : ""
      }`}
    >
      <CardContent className="p-6">
        <div className="space-y-4">
          {/* Header */}
          <div className="flex items-start gap-4">
            <div className="rounded-lg bg-status-green/10 p-2">
              <Icon className="h-5 w-5 text-status-green" />
            </div>
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-2">
                <Badge variant="outline" className="text-xs">
                  {typeLabels[recommendation.type] || recommendation.type}
                </Badge>
                <Badge variant="outline" className={`text-xs ${priorityStyle.badge}`}>
                  {recommendation.priority}
                </Badge>
                <Badge variant="outline" className="text-xs">
                  {recommendation.platform.toUpperCase()}
                </Badge>
                {isApplied && (
                  <Badge className="bg-status-green/10 text-status-green border-status-green/30">
                    <CheckCircle className="mr-1 h-3 w-3" />
                    Applied
                  </Badge>
                )}
                {isDismissed && (
                  <Badge className="bg-muted text-muted-foreground">
                    <XCircle className="mr-1 h-3 w-3" />
                    Dismissed
                  </Badge>
                )}
              </div>
              <h4 className="font-semibold">{recommendation.action}</h4>
              {recommendation.resourceName && (
                <p className="text-sm text-muted-foreground mt-1">
                  Resource: {recommendation.resourceName} ({recommendation.resourceType})
                </p>
              )}
            </div>
          </div>

          {/* Savings */}
          <div className="flex items-center gap-6 p-4 rounded-lg bg-status-green/5 border border-status-green/20">
            <div className="flex items-center gap-2">
              <TrendingDown className="h-5 w-5 text-status-green" />
              <div>
                <p className="text-xs text-muted-foreground">Potential Savings</p>
                <p className="text-2xl font-bold text-status-green tabular-nums">
                  {formatCurrency(recommendation.potentialSavings, recommendation.currency)}
                  <span className="text-sm font-normal text-muted-foreground ml-1">
                    /mo
                  </span>
                </p>
              </div>
            </div>
            <div className="h-10 w-px bg-border" />
            <div>
              <p className="text-xs text-muted-foreground">Current Cost</p>
              <p className="text-lg font-semibold tabular-nums">
                {formatCurrency(recommendation.currentCost, recommendation.currency)}
                <span className="text-xs font-normal text-muted-foreground ml-1">
                  /mo
                </span>
              </p>
            </div>
          </div>

          {/* Details */}
          {recommendation.details && (
            <div className="text-sm text-muted-foreground bg-muted/50 rounded p-3">
              {recommendation.details}
            </div>
          )}

          {/* Actions */}
          {recommendation.status === "pending" && (
            <div className="flex items-center gap-2 pt-2">
              <Button
                size="sm"
                onClick={() => onApply?.(recommendation.id)}
                className="bg-status-green hover:bg-status-green/90"
              >
                <Zap className="mr-2 h-4 w-4" />
                Apply Recommendation
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => onDismiss?.(recommendation.id)}
              >
                Dismiss
              </Button>
            </div>
          )}

          {/* Metadata */}
          <div className="flex items-center gap-4 text-xs text-muted-foreground pt-2 border-t">
            <span>ID: {recommendation.resourceId}</span>
            <span>Detected: {new Date(recommendation.detectedAt).toLocaleDateString()}</span>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
