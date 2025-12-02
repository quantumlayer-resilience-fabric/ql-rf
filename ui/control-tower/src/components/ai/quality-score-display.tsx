"use client";

import { QualityScore } from "@/hooks/use-ai";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Shield,
  FileCheck,
  TestTube,
  History,
  UserCheck,
  CheckCircle,
  XCircle,
  Info,
} from "lucide-react";
import { cn } from "@/lib/utils";

interface QualityScoreDisplayProps {
  score: QualityScore;
  compact?: boolean;
}

const dimensionConfig = {
  structural: {
    label: "Structural",
    icon: FileCheck,
    description: "Schema and syntax validity",
  },
  policy: {
    label: "Policy",
    icon: Shield,
    description: "OPA and security policy compliance",
  },
  tests: {
    label: "Tests",
    icon: TestTube,
    description: "Test coverage and pass rate",
  },
  history: {
    label: "History",
    icon: History,
    description: "Operational success rate",
  },
  review: {
    label: "Review",
    icon: UserCheck,
    description: "Human review and approvals",
  },
};

function getScoreColor(score: number, max: number = 100): string {
  const percentage = (score / max) * 100;
  if (percentage >= 80) return "text-status-green";
  if (percentage >= 60) return "text-status-amber";
  return "text-status-red";
}

function getProgressColor(score: number, max: number = 100): string {
  const percentage = (score / max) * 100;
  if (percentage >= 80) return "bg-status-green";
  if (percentage >= 60) return "bg-status-amber";
  return "bg-status-red";
}

function getEnvironmentBadge(env: string) {
  const colors: Record<string, string> = {
    development: "bg-blue-500/10 text-blue-500 border-blue-500/20",
    staging: "bg-amber-500/10 text-amber-500 border-amber-500/20",
    production: "bg-green-500/10 text-green-500 border-green-500/20",
    production_bulk: "bg-purple-500/10 text-purple-500 border-purple-500/20",
  };
  return colors[env] || "bg-muted text-muted-foreground";
}

export function QualityScoreDisplay({ score, compact = false }: QualityScoreDisplayProps) {
  if (compact) {
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <div className="flex items-center gap-2">
              <div className={cn("text-2xl font-bold", getScoreColor(score.total))}>
                {score.total}
              </div>
              <div className="text-xs text-muted-foreground">/100</div>
              {score.requires_approval && (
                <Badge variant="outline" className="text-xs">
                  Approval Required
                </Badge>
              )}
            </div>
          </TooltipTrigger>
          <TooltipContent side="bottom" className="w-64">
            <div className="space-y-2">
              <p className="font-medium">Quality Score Breakdown</p>
              <div className="space-y-1 text-xs">
                <div className="flex justify-between">
                  <span>Structural</span>
                  <span>{score.structural}/20</span>
                </div>
                <div className="flex justify-between">
                  <span>Policy</span>
                  <span>{score.policy_compliance}/20</span>
                </div>
                <div className="flex justify-between">
                  <span>Tests</span>
                  <span>{score.test_coverage}/20</span>
                </div>
                <div className="flex justify-between">
                  <span>History</span>
                  <span>{score.operational_history}/20</span>
                </div>
                <div className="flex justify-between">
                  <span>Review</span>
                  <span>{score.human_review}/20</span>
                </div>
              </div>
            </div>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  }

  return (
    <Card className="border-muted">
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <Shield className="h-4 w-4 text-brand-accent" />
            Quality Score
          </CardTitle>
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger>
                <Info className="h-4 w-4 text-muted-foreground" />
              </TooltipTrigger>
              <TooltipContent side="left" className="max-w-xs">
                <p>
                  Quality scores determine which environments an artifact can be deployed to.
                  Higher scores unlock more sensitive environments.
                </p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Overall Score */}
        <div className="flex items-center justify-between">
          <div className="flex items-baseline gap-1">
            <span className={cn("text-4xl font-bold", getScoreColor(score.total))}>
              {score.total}
            </span>
            <span className="text-lg text-muted-foreground">/100</span>
          </div>
          <div className="text-right">
            {score.requires_approval ? (
              <Badge variant="outline" className="text-status-amber border-status-amber/50">
                Approval Required
              </Badge>
            ) : (
              <Badge variant="outline" className="text-status-green border-status-green/50">
                Auto-Approve Eligible
              </Badge>
            )}
          </div>
        </div>

        {/* Progress Bar */}
        <div className="relative">
          <Progress value={score.total} className="h-2" />
          {/* Environment thresholds */}
          <div className="absolute top-0 left-[40%] h-2 w-px bg-muted-foreground/30" />
          <div className="absolute top-0 left-[60%] h-2 w-px bg-muted-foreground/30" />
          <div className="absolute top-0 left-[80%] h-2 w-px bg-muted-foreground/30" />
          <div className="absolute top-0 left-[90%] h-2 w-px bg-muted-foreground/30" />
        </div>

        {/* Allowed Environments */}
        <div>
          <p className="text-xs text-muted-foreground mb-2">Allowed Environments</p>
          <div className="flex flex-wrap gap-1">
            {score.allowed_environments.length > 0 ? (
              score.allowed_environments.map((env) => (
                <Badge
                  key={env}
                  variant="outline"
                  className={cn("text-xs", getEnvironmentBadge(env))}
                >
                  {env.replace("_", " ")}
                </Badge>
              ))
            ) : (
              <Badge variant="outline" className="text-xs text-status-red border-status-red/50">
                No environments allowed
              </Badge>
            )}
          </div>
        </div>

        {/* Dimension Breakdown */}
        <div className="space-y-3">
          <p className="text-xs text-muted-foreground">Score Breakdown</p>
          {Object.entries(score.dimensions).map(([key, dim]) => {
            const config = dimensionConfig[key as keyof typeof dimensionConfig];
            if (!config) return null;
            const Icon = config.icon;
            const percentage = (dim.score / dim.max_score) * 100;

            return (
              <div key={key} className="space-y-1">
                <div className="flex items-center justify-between text-xs">
                  <div className="flex items-center gap-2">
                    <Icon className="h-3 w-3 text-muted-foreground" />
                    <span>{config.label}</span>
                  </div>
                  <span className={cn("font-medium", getScoreColor(dim.score, dim.max_score))}>
                    {dim.score}/{dim.max_score}
                  </span>
                </div>
                <div className="relative h-1.5 bg-muted rounded-full overflow-hidden">
                  <div
                    className={cn("h-full rounded-full transition-all", getProgressColor(dim.score, dim.max_score))}
                    style={{ width: `${percentage}%` }}
                  />
                </div>
                {/* Passed/Failed checks */}
                {(dim.passed?.length || dim.failed?.length) && (
                  <div className="flex flex-wrap gap-1 mt-1">
                    {dim.passed?.map((check) => (
                      <span
                        key={check}
                        className="inline-flex items-center gap-0.5 text-[10px] text-status-green"
                      >
                        <CheckCircle className="h-2.5 w-2.5" />
                        {check.replace(/_/g, " ")}
                      </span>
                    ))}
                    {dim.failed?.map((check) => (
                      <span
                        key={check}
                        className="inline-flex items-center gap-0.5 text-[10px] text-status-red"
                      >
                        <XCircle className="h-2.5 w-2.5" />
                        {check.replace(/_/g, " ")}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            );
          })}
        </div>

        {/* Environment Thresholds Legend */}
        <div className="pt-2 border-t">
          <p className="text-xs text-muted-foreground mb-2">Environment Thresholds</p>
          <div className="grid grid-cols-2 gap-2 text-xs">
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-blue-500" />
              <span>Development: 40+</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-amber-500" />
              <span>Staging: 60+</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-green-500" />
              <span>Production: 80+</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-purple-500" />
              <span>Prod Bulk: 90+</span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// Minimal inline version for chat messages
export function QualityScoreBadge({ score }: { score: QualityScore }) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Badge
            variant="outline"
            className={cn(
              "cursor-help",
              score.total >= 80
                ? "border-status-green/50 text-status-green"
                : score.total >= 60
                ? "border-status-amber/50 text-status-amber"
                : "border-status-red/50 text-status-red"
            )}
          >
            <Shield className="h-3 w-3 mr-1" />
            Quality: {score.total}/100
          </Badge>
        </TooltipTrigger>
        <TooltipContent side="top" className="w-48">
          <div className="text-xs space-y-1">
            <p className="font-medium">Can deploy to:</p>
            <div className="flex flex-wrap gap-1">
              {score.allowed_environments.map((env) => (
                <Badge key={env} variant="secondary" className="text-xs">
                  {env}
                </Badge>
              ))}
            </div>
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
