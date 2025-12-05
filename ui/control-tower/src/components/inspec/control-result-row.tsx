"use client";

import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CheckCircle, XCircle, MinusCircle, AlertCircle, ChevronDown, ChevronRight, Zap } from "lucide-react";
import { InSpecResult } from "@/lib/api-inspec";
import { cn } from "@/lib/utils";

interface ControlResultRowProps {
  result: InSpecResult;
  onRemediate?: (result: InSpecResult) => void;
}

export function ControlResultRow({ result, onRemediate }: ControlResultRowProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  const statusConfig = {
    passed: {
      icon: <CheckCircle className="h-4 w-4" />,
      color: "text-status-green",
      bgColor: "bg-status-green/10",
      label: "Passed",
    },
    failed: {
      icon: <XCircle className="h-4 w-4" />,
      color: "text-status-red",
      bgColor: "bg-status-red/10",
      label: "Failed",
    },
    skipped: {
      icon: <MinusCircle className="h-4 w-4" />,
      color: "text-muted-foreground",
      bgColor: "bg-muted",
      label: "Skipped",
    },
    error: {
      icon: <AlertCircle className="h-4 w-4" />,
      color: "text-status-amber",
      bgColor: "bg-status-amber/10",
      label: "Error",
    },
  };

  const config = statusConfig[result.status];
  const impactLevel = result.impact || 0;

  const getImpactColor = (impact: number) => {
    if (impact >= 0.7) return "text-status-red";
    if (impact >= 0.5) return "text-status-amber";
    if (impact >= 0.3) return "text-blue-500";
    return "text-muted-foreground";
  };

  const getImpactLabel = (impact: number) => {
    if (impact >= 0.7) return "Critical";
    if (impact >= 0.5) return "High";
    if (impact >= 0.3) return "Medium";
    return "Low";
  };

  return (
    <div className="border-b last:border-b-0">
      <div
        className="flex items-center gap-4 p-4 hover:bg-muted/50 cursor-pointer transition-colors"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <button className="shrink-0">
          {isExpanded ? (
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          ) : (
            <ChevronRight className="h-4 w-4 text-muted-foreground" />
          )}
        </button>

        <div className={cn("rounded-full p-1.5 shrink-0", config.bgColor)}>
          <div className={config.color}>{config.icon}</div>
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <code className="text-xs font-mono text-muted-foreground">
              {result.controlId}
            </code>
            <Badge variant="outline" className="text-xs">
              {config.label}
            </Badge>
            {impactLevel > 0 && (
              <Badge variant="outline" className={cn("text-xs", getImpactColor(impactLevel))}>
                {getImpactLabel(impactLevel)}
              </Badge>
            )}
          </div>
          <p className="text-sm font-medium mt-1 truncate">{result.controlTitle}</p>
        </div>

        <div className="shrink-0 text-xs text-muted-foreground">
          {result.runTime.toFixed(3)}s
        </div>

        {result.status === "failed" && onRemediate && (
          <Button
            size="sm"
            variant="outline"
            onClick={(e) => {
              e.stopPropagation();
              onRemediate(result);
            }}
          >
            <Zap className="mr-1 h-3 w-3" />
            Fix
          </Button>
        )}
      </div>

      {isExpanded && (
        <div className="px-4 pb-4 pl-16 space-y-3 bg-muted/30">
          {result.message && (
            <div>
              <p className="text-xs font-semibold text-muted-foreground mb-1">Message</p>
              <p className="text-sm">{result.message}</p>
            </div>
          )}

          {result.codeDescription && (
            <div>
              <p className="text-xs font-semibold text-muted-foreground mb-1">Description</p>
              <p className="text-sm">{result.codeDescription}</p>
            </div>
          )}

          {result.resource && (
            <div>
              <p className="text-xs font-semibold text-muted-foreground mb-1">Resource</p>
              <code className="text-xs bg-muted px-2 py-1 rounded">{result.resource}</code>
            </div>
          )}

          {result.sourceLocation && (
            <div>
              <p className="text-xs font-semibold text-muted-foreground mb-1">Source Location</p>
              <code className="text-xs bg-muted px-2 py-1 rounded">{result.sourceLocation}</code>
            </div>
          )}

          {result.status === "failed" && (
            <div className="mt-3 p-3 bg-status-amber/10 border border-status-amber/20 rounded-lg">
              <p className="text-xs font-semibold text-status-amber mb-1">Recommendations</p>
              <p className="text-sm text-muted-foreground">
                Review the control requirements and adjust the configuration to meet compliance standards.
                Use AI remediation to generate automated fix scripts.
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
