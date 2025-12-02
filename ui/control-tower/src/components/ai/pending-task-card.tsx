"use client";

import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { StatusBadge } from "@/components/status/status-badge";
import { TaskWithPlan, useApproveTask, useRejectTask } from "@/hooks/use-ai";
import {
  CheckCircle,
  XCircle,
  Edit,
  Bot,
  AlertTriangle,
  Server,
  Loader2,
  ChevronDown,
  ChevronUp,
  Clock,
} from "lucide-react";
import { useState } from "react";
import { formatDistanceToNow } from "date-fns";

interface PendingTaskCardProps {
  task: TaskWithPlan;
  onApproved?: () => void;
  onRejected?: () => void;
}

export function PendingTaskCard({ task, onApproved, onRejected }: PendingTaskCardProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const approveTask = useApproveTask();
  const rejectTask = useRejectTask();

  const isLoading = approveTask.isPending || rejectTask.isPending;

  const handleApprove = async () => {
    try {
      await approveTask.mutateAsync({ taskId: task.id });
      onApproved?.();
    } catch (error) {
      console.error("Failed to approve task:", error);
    }
  };

  const handleReject = async () => {
    try {
      await rejectTask.mutateAsync({ taskId: task.id });
      onRejected?.();
    } catch (error) {
      console.error("Failed to reject task:", error);
    }
  };

  const getRiskColor = (risk?: string) => {
    switch (risk) {
      case "critical":
        return "destructive";
      case "high":
        return "destructive";
      case "medium":
        return "secondary";
      default:
        return "outline";
    }
  };

  const getPlanStateColor = (state?: string): "success" | "warning" | "critical" | "neutral" | "info" => {
    switch (state) {
      case "awaiting_approval":
        return "warning";
      case "approved":
        return "success";
      case "rejected":
        return "critical";
      default:
        return "neutral";
    }
  };

  // Extract data from task_spec if available
  const taskSpec = task.task_spec as Record<string, unknown> | undefined;
  const goal = (taskSpec?.goal as string) || task.user_intent;
  const riskLevel = task.risk_level || (taskSpec?.risk_level as string) || "low";
  const taskType = task.task_type || (taskSpec?.task_type as string) || "unknown";
  const environment = (taskSpec?.environment as string) || "production";

  // Plan data
  const plan = task.plan;
  const planPayload = plan?.payload;
  const summary = planPayload?.summary as string | undefined;
  const affectedAssets = planPayload?.affected_assets as number | undefined;
  const phases = planPayload?.phases as Array<Record<string, unknown>> | undefined;

  return (
    <Card className="border-brand-accent/20 bg-gradient-to-br from-brand-accent/5 to-transparent">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-brand-accent/10">
              <Bot className="h-4 w-4 text-brand-accent" />
            </div>
            <div>
              <CardTitle className="text-base">
                {taskType.replace(/_/g, " ").replace(/\b\w/g, (l) => l.toUpperCase())}
              </CardTitle>
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <Clock className="h-3 w-3" />
                <span>{formatDistanceToNow(new Date(task.created_at), { addSuffix: true })}</span>
                {environment && (
                  <Badge variant="outline" className="text-xs">
                    {environment}
                  </Badge>
                )}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Badge variant={getRiskColor(riskLevel)}>
              {riskLevel}
            </Badge>
            {task.hitl_required && plan?.state === "awaiting_approval" && (
              <StatusBadge status="warning" size="sm">
                Approval Required
              </StatusBadge>
            )}
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Goal / User Intent */}
        <div>
          <h4 className="text-sm font-medium mb-1">Request</h4>
          <p className="text-sm text-muted-foreground">{task.user_intent}</p>
        </div>

        {/* Goal (if different from intent) */}
        {goal && goal !== task.user_intent && (
          <div>
            <h4 className="text-sm font-medium mb-1">Interpreted Goal</h4>
            <p className="text-sm text-muted-foreground">{goal}</p>
          </div>
        )}

        {/* Summary from plan */}
        {summary && (
          <div>
            <h4 className="text-sm font-medium mb-1">Summary</h4>
            <p className="text-sm text-muted-foreground">{summary}</p>
          </div>
        )}

        {/* Impact */}
        <div className="flex items-center gap-4 text-sm">
          {affectedAssets !== undefined && (
            <div className="flex items-center gap-1">
              <Server className="h-4 w-4 text-muted-foreground" />
              <span>{affectedAssets} assets affected</span>
            </div>
          )}
          {plan && (
            <StatusBadge status={getPlanStateColor(plan.state)} size="sm">
              {plan.state.replace(/_/g, " ")}
            </StatusBadge>
          )}
        </div>

        {/* Expandable Plan Phases */}
        {phases && phases.length > 0 && (
          <div>
            <Button
              variant="ghost"
              size="sm"
              className="w-full justify-between"
              onClick={() => setIsExpanded(!isExpanded)}
            >
              <span>View Plan Details ({phases.length} phases)</span>
              {isExpanded ? (
                <ChevronUp className="h-4 w-4" />
              ) : (
                <ChevronDown className="h-4 w-4" />
              )}
            </Button>
            {isExpanded && (
              <div className="mt-2 rounded-lg border bg-muted/50 p-4 text-sm overflow-auto max-h-96 space-y-3">
                {phases.map((phase, index) => (
                  <div key={index} className="border-l-2 border-brand-accent/30 pl-3">
                    <div className="font-medium">{String(phase.name || `Phase ${index + 1}`)}</div>
                    {phase.assets !== undefined && (
                      <div className="text-muted-foreground text-xs">
                        {Array.isArray(phase.assets) ? phase.assets.length : String(phase.assets)} assets
                      </div>
                    )}
                    {phase.wait_time !== undefined && (
                      <div className="text-muted-foreground text-xs">
                        Wait: {String(phase.wait_time)}
                      </div>
                    )}
                    {phase.rollback_if !== undefined && (
                      <div className="text-xs text-status-amber">
                        Rollback if: {String(phase.rollback_if)}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Risk Warning */}
        {(riskLevel === "high" || riskLevel === "critical") && (
          <div className="flex items-start gap-2 rounded-lg border border-status-amber/50 bg-status-amber/10 p-3">
            <AlertTriangle className="h-4 w-4 text-status-amber shrink-0 mt-0.5" />
            <p className="text-sm text-status-amber">
              This is a {riskLevel}-risk operation. Please review carefully before approving.
            </p>
          </div>
        )}
      </CardContent>

      {plan?.state === "awaiting_approval" && (
        <CardFooter className="flex justify-end gap-2 pt-4">
          <Button
            variant="outline"
            size="sm"
            onClick={handleReject}
            disabled={isLoading}
          >
            {rejectTask.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <XCircle className="mr-2 h-4 w-4" />
            )}
            Reject
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={isLoading}
          >
            <Edit className="mr-2 h-4 w-4" />
            Modify
          </Button>
          <Button
            size="sm"
            onClick={handleApprove}
            disabled={isLoading}
            className="bg-status-green hover:bg-status-green/90"
          >
            {approveTask.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <CheckCircle className="mr-2 h-4 w-4" />
            )}
            Approve & Execute
          </Button>
        </CardFooter>
      )}

      {/* Show status for already processed tasks */}
      {plan?.state === "approved" && (
        <CardFooter className="pt-4">
          <div className="flex items-center gap-2 text-status-green">
            <CheckCircle className="h-4 w-4" />
            <span className="text-sm">
              Approved {plan.approved_at && formatDistanceToNow(new Date(plan.approved_at), { addSuffix: true })}
            </span>
          </div>
        </CardFooter>
      )}

      {plan?.state === "rejected" && (
        <CardFooter className="pt-4">
          <div className="flex flex-col gap-1">
            <div className="flex items-center gap-2 text-destructive">
              <XCircle className="h-4 w-4" />
              <span className="text-sm">Rejected</span>
            </div>
            {plan.rejection_reason && (
              <p className="text-xs text-muted-foreground">{plan.rejection_reason}</p>
            )}
          </div>
        </CardFooter>
      )}
    </Card>
  );
}
