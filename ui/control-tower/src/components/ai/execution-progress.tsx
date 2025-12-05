"use client";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Skeleton } from "@/components/ui/skeleton";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  useTaskExecutions,
  useExecution,
  usePauseExecution,
  useResumeExecution,
  useCancelExecution,
  useRollbackExecution,
  PhaseExecution,
  ExecutionStatus,
  PhaseStatus,
} from "@/hooks/use-ai";
import {
  Play,
  Pause,
  RotateCcw,
  XCircle,
  CheckCircle2,
  Clock,
  Loader2,
  AlertTriangle,
  Activity,
  Server,
} from "lucide-react";

// Status colors
const statusColors: Record<ExecutionStatus | PhaseStatus, string> = {
  pending: "text-muted-foreground bg-muted",
  running: "text-blue-500 bg-blue-500/10",
  waiting: "text-amber-500 bg-amber-500/10",
  paused: "text-amber-500 bg-amber-500/10",
  completed: "text-status-green bg-status-green/10",
  failed: "text-status-red bg-status-red/10",
  rolled_back: "text-purple-500 bg-purple-500/10",
  cancelled: "text-muted-foreground bg-muted",
  skipped: "text-muted-foreground bg-muted",
};

// Status icons
function StatusIcon({ status }: { status: ExecutionStatus | PhaseStatus }) {
  switch (status) {
    case "running":
      return <Loader2 className="h-4 w-4 animate-spin" />;
    case "waiting":
    case "paused":
      return <Pause className="h-4 w-4" />;
    case "completed":
      return <CheckCircle2 className="h-4 w-4" />;
    case "failed":
      return <XCircle className="h-4 w-4" />;
    case "rolled_back":
      return <RotateCcw className="h-4 w-4" />;
    case "cancelled":
    case "skipped":
      return <XCircle className="h-4 w-4" />;
    default:
      return <Clock className="h-4 w-4" />;
  }
}

interface PhaseCardProps {
  phase: PhaseExecution;
  index: number;
  isActive: boolean;
}

function PhaseCard({ phase, index, isActive }: PhaseCardProps) {
  const completedAssets = phase.assets?.filter((a) => a.status === "completed").length || 0;
  const failedAssets = phase.assets?.filter((a) => a.status === "failed").length || 0;
  const totalAssets = phase.assets?.length || 0;
  const progress = totalAssets > 0 ? (completedAssets / totalAssets) * 100 : 0;

  return (
    <div
      className={`rounded-lg border p-4 transition-all ${
        isActive ? "border-brand-accent bg-brand-accent/5" : "border-border"
      }`}
    >
      <div className="flex items-start justify-between mb-2">
        <div className="flex items-center gap-2">
          <div
            className={`flex items-center justify-center w-6 h-6 rounded-full text-xs font-bold ${
              statusColors[phase.status]
            }`}
          >
            {index + 1}
          </div>
          <div>
            <h4 className="font-medium text-sm">{phase.name}</h4>
            {phase.status === "waiting" && phase.wait_until && (
              <p className="text-xs text-muted-foreground">
                Waiting until {new Date(phase.wait_until).toLocaleTimeString()}
              </p>
            )}
          </div>
        </div>
        <Badge
          variant="outline"
          className={`${statusColors[phase.status]} border-current/20`}
        >
          <StatusIcon status={phase.status} />
          <span className="ml-1 capitalize">{phase.status}</span>
        </Badge>
      </div>

      {totalAssets > 0 && (
        <>
          <Progress value={progress} className="h-1.5 mb-2" />
          <div className="flex justify-between text-xs text-muted-foreground">
            <span>
              {completedAssets}/{totalAssets} assets
            </span>
            {failedAssets > 0 && (
              <span className="text-status-red">{failedAssets} failed</span>
            )}
          </div>
        </>
      )}

      {phase.error && (
        <div className="mt-2 p-2 rounded bg-destructive/10 text-xs text-destructive">
          {phase.error}
        </div>
      )}

      {/* Asset details (collapsed by default) */}
      {phase.assets && phase.assets.length > 0 && (
        <details className="mt-3">
          <summary className="text-xs text-muted-foreground cursor-pointer hover:text-foreground">
            View {phase.assets.length} assets
          </summary>
          <div className="mt-2 space-y-1">
            {phase.assets.slice(0, 10).map((asset) => (
              <div
                key={asset.asset_id}
                className="flex items-center justify-between text-xs p-1.5 rounded bg-muted/50"
              >
                <div className="flex items-center gap-2">
                  <Server className="h-3 w-3 text-muted-foreground" />
                  <span className="truncate max-w-[150px]">{asset.asset_name}</span>
                </div>
                <Badge variant="outline" className="text-[10px] h-5">
                  {asset.status}
                </Badge>
              </div>
            ))}
            {phase.assets.length > 10 && (
              <p className="text-xs text-muted-foreground text-center">
                +{phase.assets.length - 10} more assets
              </p>
            )}
          </div>
        </details>
      )}
    </div>
  );
}

interface ExecutionProgressProps {
  taskId: string;
}

export function ExecutionProgress({ taskId }: ExecutionProgressProps) {
  const { data: executions, isLoading: executionsLoading } = useTaskExecutions(taskId);
  const latestExecution = executions?.[0];

  const pauseExecution = usePauseExecution();
  const resumeExecution = useResumeExecution();
  const cancelExecution = useCancelExecution();
  const rollbackExecution = useRollbackExecution();

  // Fetch detailed execution data if we have one
  const { data: execution, isLoading: executionLoading } = useExecution(
    latestExecution?.id || ""
  );

  const isLoading = executionsLoading || executionLoading;
  const currentExecution = execution || latestExecution;

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-6 w-40" />
          <Skeleton className="h-4 w-60" />
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {[...Array(3)].map((_, i) => (
              <Skeleton key={i} className="h-24" />
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!currentExecution) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Activity className="h-5 w-5" />
            Execution Progress
          </CardTitle>
          <CardDescription>No active execution for this task</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
            <Activity className="h-12 w-12 mb-2 opacity-50" />
            <p>Task has not been executed yet</p>
            <p className="text-sm">Approve the task to start execution</p>
          </div>
        </CardContent>
      </Card>
    );
  }

  const overallProgress =
    currentExecution.total_phases > 0
      ? (currentExecution.current_phase / currentExecution.total_phases) * 100
      : 0;

  const isRunning = currentExecution.status === "running";
  const isPaused = currentExecution.status === "paused";
  const canPause = isRunning;
  const canResume = isPaused;
  const canCancel = isRunning || isPaused;
  const canRollback = currentExecution.status === "failed";

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between">
          <div>
            <CardTitle className="flex items-center gap-2 text-base">
              <Activity className="h-5 w-5" />
              Execution Progress
            </CardTitle>
            <CardDescription>
              Phase {currentExecution.current_phase} of {currentExecution.total_phases}
            </CardDescription>
          </div>
          <Badge
            variant="outline"
            className={`${statusColors[currentExecution.status]} border-current/20`}
          >
            <StatusIcon status={currentExecution.status} />
            <span className="ml-1 capitalize">{currentExecution.status.replace("_", " ")}</span>
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Overall Progress */}
        <div>
          <div className="flex justify-between text-sm mb-2">
            <span className="text-muted-foreground">Overall Progress</span>
            <span className="font-medium">{Math.round(overallProgress)}%</span>
          </div>
          <Progress value={overallProgress} className="h-2" />
        </div>

        {/* Control Buttons */}
        <div className="flex gap-2">
          <TooltipProvider>
            {canPause && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => pauseExecution.mutate(currentExecution.id)}
                    disabled={pauseExecution.isPending}
                  >
                    {pauseExecution.isPending ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Pause className="h-4 w-4" />
                    )}
                    <span className="ml-2">Pause</span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Pause execution after current phase</TooltipContent>
              </Tooltip>
            )}

            {canResume && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => resumeExecution.mutate(currentExecution.id)}
                    disabled={resumeExecution.isPending}
                  >
                    {resumeExecution.isPending ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Play className="h-4 w-4" />
                    )}
                    <span className="ml-2">Resume</span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Resume paused execution</TooltipContent>
              </Tooltip>
            )}

            {canCancel && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => cancelExecution.mutate(currentExecution.id)}
                    disabled={cancelExecution.isPending}
                    className="text-destructive hover:text-destructive"
                  >
                    {cancelExecution.isPending ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <XCircle className="h-4 w-4" />
                    )}
                    <span className="ml-2">Cancel</span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Cancel execution</TooltipContent>
              </Tooltip>
            )}

            {canRollback && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() =>
                      rollbackExecution.mutate({
                        executionId: currentExecution.id,
                        reason: "Manual rollback requested",
                      })
                    }
                    disabled={rollbackExecution.isPending}
                    className="text-purple-500 hover:text-purple-500"
                  >
                    {rollbackExecution.isPending ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <RotateCcw className="h-4 w-4" />
                    )}
                    <span className="ml-2">Rollback</span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Rollback failed changes</TooltipContent>
              </Tooltip>
            )}
          </TooltipProvider>
        </div>

        {/* Error Display */}
        {currentExecution.error && (
          <div className="rounded-lg border border-destructive/20 bg-destructive/5 p-3">
            <div className="flex items-center gap-2 text-destructive mb-1">
              <AlertTriangle className="h-4 w-4" />
              <span className="font-medium">Execution Error</span>
            </div>
            <p className="text-sm text-destructive">{currentExecution.error}</p>
          </div>
        )}

        {/* Phases */}
        <div>
          <h4 className="text-sm font-medium mb-3">Execution Phases</h4>
          <ScrollArea className="h-[300px] pr-4">
            <div className="space-y-3">
              {currentExecution.phases?.map((phase, index) => (
                <PhaseCard
                  key={`${phase.name}-${index}`}
                  phase={phase}
                  index={index}
                  isActive={index === currentExecution.current_phase - 1}
                />
              ))}
            </div>
          </ScrollArea>
        </div>

        {/* Metadata */}
        <div className="pt-4 border-t text-xs text-muted-foreground">
          <div className="flex justify-between">
            <span>Started by: {currentExecution.started_by}</span>
            <span>Started: {new Date(currentExecution.started_at).toLocaleString()}</span>
          </div>
          {currentExecution.completed_at && (
            <div className="mt-1">
              Completed: {new Date(currentExecution.completed_at).toLocaleString()}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
