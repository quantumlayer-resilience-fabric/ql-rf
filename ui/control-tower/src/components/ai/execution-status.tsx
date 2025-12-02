"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Separator } from "@/components/ui/separator";
import {
  Execution,
  PhaseExecution,
  ExecutionStatus as ExecStatus,
  PhaseStatus,
  usePauseExecution,
  useResumeExecution,
  useCancelExecution,
} from "@/hooks/use-ai";
import { formatDistanceToNow, format } from "date-fns";
import {
  Loader2,
  CheckCircle,
  XCircle,
  PauseCircle,
  PlayCircle,
  StopCircle,
  Clock,
  RotateCcw,
  AlertTriangle,
  ChevronDown,
  ChevronRight,
} from "lucide-react";
import { useState } from "react";
import { cn } from "@/lib/utils";

interface ExecutionStatusProps {
  execution: Execution;
  showControls?: boolean;
}

const statusConfig: Record<
  ExecStatus,
  { color: string; icon: React.ComponentType<{ className?: string }>; label: string }
> = {
  pending: { color: "bg-gray-500", icon: Clock, label: "Pending" },
  running: { color: "bg-blue-500", icon: Loader2, label: "Running" },
  paused: { color: "bg-yellow-500", icon: PauseCircle, label: "Paused" },
  completed: { color: "bg-green-500", icon: CheckCircle, label: "Completed" },
  failed: { color: "bg-red-500", icon: XCircle, label: "Failed" },
  rolled_back: { color: "bg-orange-500", icon: RotateCcw, label: "Rolled Back" },
  cancelled: { color: "bg-gray-500", icon: StopCircle, label: "Cancelled" },
};

const phaseStatusConfig: Record<
  PhaseStatus,
  { color: string; bgColor: string; icon: React.ComponentType<{ className?: string }> }
> = {
  pending: { color: "text-gray-500", bgColor: "bg-gray-100", icon: Clock },
  running: { color: "text-blue-500", bgColor: "bg-blue-100", icon: Loader2 },
  waiting: { color: "text-yellow-500", bgColor: "bg-yellow-100", icon: Clock },
  completed: { color: "text-green-500", bgColor: "bg-green-100", icon: CheckCircle },
  failed: { color: "text-red-500", bgColor: "bg-red-100", icon: XCircle },
  skipped: { color: "text-gray-400", bgColor: "bg-gray-50", icon: ChevronRight },
};

export function ExecutionStatus({ execution, showControls = true }: ExecutionStatusProps) {
  const pauseExecution = usePauseExecution();
  const resumeExecution = useResumeExecution();
  const cancelExecution = useCancelExecution();

  const isActionLoading =
    pauseExecution.isPending || resumeExecution.isPending || cancelExecution.isPending;

  const config = statusConfig[execution.status];
  const StatusIcon = config.icon;
  const progress = ((execution.current_phase + 1) / execution.total_phases) * 100;

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className={cn("rounded-full p-2", config.color)}>
              <StatusIcon
                className={cn(
                  "h-4 w-4 text-white",
                  execution.status === "running" && "animate-spin"
                )}
              />
            </div>
            <div>
              <CardTitle className="text-base">Execution Status</CardTitle>
              <p className="text-sm text-muted-foreground">
                Started {formatDistanceToNow(new Date(execution.started_at), { addSuffix: true })}
              </p>
            </div>
          </div>
          <Badge
            variant={
              execution.status === "completed"
                ? "default"
                : execution.status === "failed"
                ? "destructive"
                : "secondary"
            }
          >
            {config.label}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Progress Bar */}
        <div className="space-y-2">
          <div className="flex justify-between text-sm">
            <span>
              Phase {execution.current_phase + 1} of {execution.total_phases}
            </span>
            <span>{Math.round(progress)}%</span>
          </div>
          <Progress value={progress} className="h-2" />
        </div>

        <Separator />

        {/* Phases */}
        <div className="space-y-2">
          <h4 className="text-sm font-medium">Phases</h4>
          <div className="space-y-2">
            {execution.phases.map((phase, index) => (
              <PhaseRow key={index} phase={phase} index={index} isCurrentPhase={index === execution.current_phase} />
            ))}
          </div>
        </div>

        {/* Error Display */}
        {execution.error && (
          <>
            <Separator />
            <div className="rounded-md border border-red-200 bg-red-50 p-3">
              <div className="flex items-start gap-2">
                <AlertTriangle className="h-4 w-4 text-red-500 mt-0.5" />
                <div>
                  <p className="text-sm font-medium text-red-800">Execution Failed</p>
                  <p className="text-sm text-red-600">{execution.error}</p>
                </div>
              </div>
            </div>
          </>
        )}

        {/* Rollback Error */}
        {execution.rollback_error && (
          <div className="rounded-md border border-orange-200 bg-orange-50 p-3">
            <div className="flex items-start gap-2">
              <RotateCcw className="h-4 w-4 text-orange-500 mt-0.5" />
              <div>
                <p className="text-sm font-medium text-orange-800">Rollback Error</p>
                <p className="text-sm text-orange-600">{execution.rollback_error}</p>
              </div>
            </div>
          </div>
        )}

        {/* Controls */}
        {showControls && (execution.status === "running" || execution.status === "paused") && (
          <>
            <Separator />
            <div className="flex gap-2">
              {execution.status === "running" && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => pauseExecution.mutate(execution.id)}
                  disabled={isActionLoading}
                >
                  {pauseExecution.isPending ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <PauseCircle className="mr-2 h-4 w-4" />
                  )}
                  Pause
                </Button>
              )}
              {execution.status === "paused" && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => resumeExecution.mutate(execution.id)}
                  disabled={isActionLoading}
                >
                  {resumeExecution.isPending ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <PlayCircle className="mr-2 h-4 w-4" />
                  )}
                  Resume
                </Button>
              )}
              <Button
                variant="outline"
                size="sm"
                className="text-destructive hover:text-destructive"
                onClick={() => cancelExecution.mutate(execution.id)}
                disabled={isActionLoading}
              >
                {cancelExecution.isPending ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <StopCircle className="mr-2 h-4 w-4" />
                )}
                Cancel
              </Button>
            </div>
          </>
        )}

        {/* Completion Time */}
        {execution.completed_at && (
          <>
            <Separator />
            <div className="text-sm text-muted-foreground">
              Completed {format(new Date(execution.completed_at), "PPpp")}
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

interface PhaseRowProps {
  phase: PhaseExecution;
  index: number;
  isCurrentPhase: boolean;
}

function PhaseRow({ phase, index, isCurrentPhase }: PhaseRowProps) {
  const [expanded, setExpanded] = useState(isCurrentPhase);
  const config = phaseStatusConfig[phase.status];
  const StatusIcon = config.icon;

  return (
    <div className={cn("rounded-md border", isCurrentPhase && "border-brand-accent/30 bg-brand-accent/5")}>
      <button
        className="flex w-full items-center justify-between p-3 text-left"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-center gap-3">
          <div className={cn("rounded-full p-1", config.bgColor)}>
            <StatusIcon
              className={cn(
                "h-3 w-3",
                config.color,
                phase.status === "running" && "animate-spin"
              )}
            />
          </div>
          <div>
            <span className="font-medium text-sm">{phase.name || `Phase ${index + 1}`}</span>
            {phase.started_at && (
              <p className="text-xs text-muted-foreground">
                Started {formatDistanceToNow(new Date(phase.started_at), { addSuffix: true })}
              </p>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          {phase.assets.length > 0 && (
            <Badge variant="outline" className="text-xs">
              {phase.assets.filter((a) => a.status === "completed").length}/{phase.assets.length} assets
            </Badge>
          )}
          {expanded ? (
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          ) : (
            <ChevronRight className="h-4 w-4 text-muted-foreground" />
          )}
        </div>
      </button>

      {expanded && phase.assets.length > 0 && (
        <div className="border-t px-3 py-2">
          <div className="space-y-1">
            {phase.assets.map((asset, assetIndex) => (
              <div
                key={assetIndex}
                className="flex items-center justify-between py-1 text-sm"
              >
                <span className="text-muted-foreground">{asset.asset_name || asset.asset_id}</span>
                <Badge
                  variant={
                    asset.status === "completed"
                      ? "default"
                      : asset.status === "failed"
                      ? "destructive"
                      : asset.status === "running"
                      ? "secondary"
                      : "outline"
                  }
                  className="text-xs"
                >
                  {asset.status}
                </Badge>
              </div>
            ))}
          </div>
          {phase.error && (
            <div className="mt-2 rounded border border-red-200 bg-red-50 p-2 text-xs text-red-600">
              {phase.error}
            </div>
          )}
        </div>
      )}

      {phase.wait_until && phase.status === "waiting" && (
        <div className="border-t px-3 py-2 text-sm text-muted-foreground">
          <Clock className="mr-1 inline h-3 w-3" />
          Waiting until {format(new Date(phase.wait_until), "HH:mm:ss")}
        </div>
      )}
    </div>
  );
}

/**
 * Compact execution status badge for inline display
 */
export function ExecutionStatusBadge({ execution }: { execution: Execution }) {
  const config = statusConfig[execution.status];
  const StatusIcon = config.icon;

  return (
    <Badge variant="outline" className="gap-1">
      <StatusIcon
        className={cn("h-3 w-3", execution.status === "running" && "animate-spin")}
      />
      {config.label}
      <span className="text-muted-foreground">
        {execution.current_phase + 1}/{execution.total_phases}
      </span>
    </Badge>
  );
}
