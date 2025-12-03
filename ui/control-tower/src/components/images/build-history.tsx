"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ImageBuild } from "@/lib/api";
import {
  CheckCircle,
  XCircle,
  Clock,
  Loader2,
  GitCommit,
  GitBranch,
  ExternalLink,
  Terminal,
  User,
  Calendar,
  Timer,
} from "lucide-react";
import { cn } from "@/lib/utils";

interface BuildHistoryProps {
  builds: ImageBuild[];
  className?: string;
}

const buildStatusConfig: Record<string, {
  icon: typeof Clock;
  color: string;
  bgColor: string;
  label: string;
  animate?: boolean;
}> = {
  pending: {
    icon: Clock,
    color: "text-gray-500",
    bgColor: "bg-gray-500/10",
    label: "Pending",
  },
  building: {
    icon: Loader2,
    color: "text-blue-500",
    bgColor: "bg-blue-500/10",
    label: "Building",
    animate: true,
  },
  success: {
    icon: CheckCircle,
    color: "text-status-green",
    bgColor: "bg-green-500/10",
    label: "Success",
  },
  failed: {
    icon: XCircle,
    color: "text-status-red",
    bgColor: "bg-red-500/10",
    label: "Failed",
  },
};

export function BuildHistory({ builds, className }: BuildHistoryProps) {
  if (builds.length === 0) {
    return (
      <Card className={className}>
        <CardContent className="flex flex-col items-center justify-center py-12 text-center">
          <Terminal className="h-12 w-12 text-muted-foreground mb-4" />
          <h3 className="text-lg font-medium">No Build History</h3>
          <p className="text-sm text-muted-foreground mt-1">
            No builds have been recorded for this image.
          </p>
        </CardContent>
      </Card>
    );
  }

  const formatDuration = (seconds: number) => {
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
    const hours = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    return `${hours}h ${mins}m`;
  };

  return (
    <Card className={className}>
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center gap-2 text-base">
          <Terminal className="h-4 w-4" />
          Build History ({builds.length})
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {builds.map((build, index) => {
            const config = buildStatusConfig[build.status] || buildStatusConfig.pending;
            const Icon = config.icon;

            return (
              <div
                key={build.id}
                className={cn(
                  "relative p-4 rounded-lg border",
                  config.bgColor,
                  "border-border"
                )}
              >
                {/* Build number indicator */}
                <div className="absolute -left-3 top-4 flex h-6 w-6 items-center justify-center rounded-full bg-background border text-xs font-medium">
                  #{build.buildNumber}
                </div>

                <div className="ml-4">
                  {/* Status and timestamp row */}
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Icon
                        className={cn(
                          "h-4 w-4",
                          config.color,
                          config.animate && "animate-spin"
                        )}
                      />
                      <Badge variant="secondary" className="text-xs">
                        {config.label}
                      </Badge>
                      <Badge variant="outline" className="text-xs">
                        {build.builderType}
                      </Badge>
                    </div>
                    <div className="flex items-center gap-1 text-xs text-muted-foreground">
                      <Calendar className="h-3 w-3" />
                      {new Date(build.createdAt).toLocaleString()}
                    </div>
                  </div>

                  {/* Source info */}
                  {(build.sourceRepo || build.sourceCommit) && (
                    <div className="flex items-center gap-4 text-xs text-muted-foreground mb-2">
                      {build.sourceRepo && (
                        <div className="flex items-center gap-1">
                          <GitBranch className="h-3 w-3" />
                          {build.sourceBranch || "main"}
                        </div>
                      )}
                      {build.sourceCommit && (
                        <div className="flex items-center gap-1">
                          <GitCommit className="h-3 w-3" />
                          <code className="bg-muted px-1 rounded">
                            {build.sourceCommit.substring(0, 7)}
                          </code>
                        </div>
                      )}
                    </div>
                  )}

                  {/* Build details row */}
                  <div className="flex items-center gap-4 text-xs text-muted-foreground">
                    {build.builtBy && (
                      <div className="flex items-center gap-1">
                        <User className="h-3 w-3" />
                        {build.builtBy}
                      </div>
                    )}
                    {build.buildDurationSeconds && (
                      <div className="flex items-center gap-1">
                        <Timer className="h-3 w-3" />
                        {formatDuration(build.buildDurationSeconds)}
                      </div>
                    )}
                    {build.buildRunner && (
                      <div className="flex items-center gap-1">
                        <Terminal className="h-3 w-3" />
                        {build.buildRunner}
                      </div>
                    )}
                  </div>

                  {/* Error message for failed builds */}
                  {build.status === "failed" && build.errorMessage && (
                    <div className="mt-2 p-2 rounded bg-red-500/10 text-xs text-status-red">
                      {build.errorMessage}
                    </div>
                  )}

                  {/* Action buttons */}
                  <div className="flex items-center gap-2 mt-3">
                    {build.buildLogUrl && (
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 text-xs"
                        onClick={() => window.open(build.buildLogUrl, "_blank")}
                      >
                        <ExternalLink className="h-3 w-3 mr-1" />
                        View Logs
                      </Button>
                    )}
                    {build.buildRunnerUrl && (
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 text-xs"
                        onClick={() => window.open(build.buildRunnerUrl, "_blank")}
                      >
                        <ExternalLink className="h-3 w-3 mr-1" />
                        CI Run
                      </Button>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}
