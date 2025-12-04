"use client";

import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { StatusBadge } from "@/components/status/status-badge";
import { Progress } from "@/components/ui/progress";
import { AITask, useApproveTask, useRejectTask, TaskWithPlan } from "@/hooks/use-ai";
import { QualityScoreDisplay, QualityScoreBadge } from "./quality-score-display";
import { PlanModificationDialog } from "./plan-modification-dialog";
import { PermissionGate } from "@/components/auth/permission-gate";
import { Permissions } from "@/hooks/use-permissions";
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
  Shield,
  ShieldAlert,
  Users,
  UserCheck,
  Clock,
} from "lucide-react";
import { useState } from "react";
import ReactMarkdown from "react-markdown";

interface TaskApprovalCardProps {
  task: AITask;
  onApproved?: () => void;
  onRejected?: () => void;
}

export function TaskApprovalCard({ task, onApproved, onRejected }: TaskApprovalCardProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const approveTask = useApproveTask();
  const rejectTask = useRejectTask();

  const isLoading = approveTask.isPending || rejectTask.isPending;

  const handleApprove = async () => {
    try {
      await approveTask.mutateAsync({ taskId: task.task_id });
      onApproved?.();
    } catch (error) {
      console.error("Failed to approve task:", error);
    }
  };

  const handleReject = async () => {
    try {
      await rejectTask.mutateAsync({ taskId: task.task_id });
      onRejected?.();
    } catch (error) {
      console.error("Failed to reject task:", error);
    }
  };

  const getRiskColor = (risk: string) => {
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
                {task.task_spec.task_type.replace(/_/g, " ").replace(/\b\w/g, (l) => l.toUpperCase())}
              </CardTitle>
              <p className="text-xs text-muted-foreground">
                by {task.agent_result?.agent_name || "AI Agent"}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Badge variant={getRiskColor(task.task_spec.risk_level)}>
              {task.task_spec.risk_level}
            </Badge>
            {task.requires_hitl && (
              <StatusBadge status="warning" size="sm">
                Approval Required
              </StatusBadge>
            )}
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Goal */}
        <div>
          <h4 className="text-sm font-medium mb-1">Goal</h4>
          <p className="text-sm text-muted-foreground">{task.task_spec.goal}</p>
        </div>

        {/* Summary */}
        {task.agent_result?.summary && (
          <div>
            <h4 className="text-sm font-medium mb-1">Summary</h4>
            <p className="text-sm text-muted-foreground">{task.agent_result.summary}</p>
          </div>
        )}

        {/* Impact */}
        {task.agent_result && (
          <div className="flex items-center gap-4 text-sm">
            <div className="flex items-center gap-1">
              <Server className="h-4 w-4 text-muted-foreground" />
              <span>{task.agent_result.affected_assets} assets affected</span>
            </div>
            {task.quality_score && (
              <QualityScoreBadge score={task.quality_score} />
            )}
            {task.agent_result.tokens_used && (
              <div className="text-muted-foreground">
                {task.agent_result.tokens_used} tokens used
              </div>
            )}
          </div>
        )}

        {/* Quality Score Details */}
        {task.quality_score && (
          <QualityScoreDisplay score={task.quality_score} />
        )}

        {/* Expandable Plan */}
        {task.agent_result?.plan && (
          <div>
            <Button
              variant="ghost"
              size="sm"
              className="w-full justify-between"
              onClick={() => setIsExpanded(!isExpanded)}
            >
              <span>View Plan Details</span>
              {isExpanded ? (
                <ChevronUp className="h-4 w-4" />
              ) : (
                <ChevronDown className="h-4 w-4" />
              )}
            </Button>
            {isExpanded && (
              <div className="mt-2 rounded-lg border bg-muted/50 p-4 text-sm prose prose-sm dark:prose-invert max-w-none overflow-auto max-h-96">
                <ReactMarkdown>{task.agent_result.plan}</ReactMarkdown>
              </div>
            )}
          </div>
        )}

        {/* Risk Warning */}
        {(task.task_spec.risk_level === "high" || task.task_spec.risk_level === "critical") && (
          <div className="flex items-start gap-2 rounded-lg border border-status-amber/50 bg-status-amber/10 p-3">
            <AlertTriangle className="h-4 w-4 text-status-amber shrink-0 mt-0.5" />
            <p className="text-sm text-status-amber">
              This is a {task.task_spec.risk_level}-risk operation. Please review carefully before approving.
            </p>
          </div>
        )}

        {/* Dual Approval Status */}
        {task.execution_policy?.require_two_approvers && (
          <DualApprovalStatus task={task} />
        )}
      </CardContent>

      <PermissionGate
        permission={Permissions.APPROVE_AI_TASKS}
        fallback={
          <CardFooter className="pt-4">
            <div className="flex items-center gap-2 text-muted-foreground">
              <ShieldAlert className="h-4 w-4" />
              <span className="text-sm">
                You need the &quot;Approve AI Tasks&quot; permission to take action on this task.
              </span>
            </div>
          </CardFooter>
        }
      >
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
          <PlanModificationDialog
            task={convertToTaskWithPlan(task)}
            disabled={isLoading}
            trigger={
              <Button variant="outline" size="sm" disabled={isLoading}>
                <Edit className="mr-2 h-4 w-4" />
                Modify
              </Button>
            }
          />
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
            {getApproveButtonText(task)}
          </Button>
        </CardFooter>
      </PermissionGate>
    </Card>
  );
}

// Helper function to get appropriate approve button text
function getApproveButtonText(task: AITask): string {
  if (!task.execution_policy?.require_two_approvers) {
    return "Approve & Execute";
  }

  const approvalsReceived = task.approval_status?.approvals_received ?? 0;
  const approvalsRequired = task.approval_status?.approvals_required ?? 2;

  if (approvalsReceived === 0) {
    return "First Approval";
  } else if (approvalsReceived < approvalsRequired) {
    return "Second Approval & Execute";
  }
  return "Approve & Execute";
}

// Component to display dual approval status
function DualApprovalStatus({ task }: { task: AITask }) {
  const approvalsRequired = task.approval_status?.approvals_required ?? 2;
  const approvalsReceived = task.approval_status?.approvals_received ?? 0;
  const progress = (approvalsReceived / approvalsRequired) * 100;

  return (
    <div className="rounded-lg border border-brand-accent/30 bg-brand-accent/5 p-4 space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Users className="h-4 w-4 text-brand-accent" />
          <span className="text-sm font-medium">Dual Approval Required</span>
        </div>
        <Badge variant="outline" className="text-xs">
          {approvalsReceived}/{approvalsRequired} approvals
        </Badge>
      </div>

      {/* Progress bar */}
      <div className="space-y-1">
        <Progress value={progress} className="h-2" />
        <p className="text-xs text-muted-foreground">
          {approvalsReceived === 0 && "Waiting for first approval..."}
          {approvalsReceived === 1 && "Waiting for second approval to execute..."}
          {approvalsReceived >= approvalsRequired && "All approvals received - ready to execute"}
        </p>
      </div>

      {/* Approval timeline */}
      <div className="space-y-2">
        {/* First approver slot */}
        <div className="flex items-center gap-2">
          {task.approval_status?.first_approval ? (
            <>
              <UserCheck className="h-4 w-4 text-status-green" />
              <span className="text-sm">
                {task.approval_status.first_approval.user_name || "User"}
              </span>
              <span className="text-xs text-muted-foreground">
                • {formatApprovalTime(task.approval_status.first_approval.approved_at)}
              </span>
            </>
          ) : (
            <>
              <Clock className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm text-muted-foreground">
                Awaiting first approval
              </span>
            </>
          )}
        </div>

        {/* Second approver slot */}
        <div className="flex items-center gap-2">
          {task.approval_status?.second_approval ? (
            <>
              <UserCheck className="h-4 w-4 text-status-green" />
              <span className="text-sm">
                {task.approval_status.second_approval.user_name || "User"}
              </span>
              <span className="text-xs text-muted-foreground">
                • {formatApprovalTime(task.approval_status.second_approval.approved_at)}
              </span>
            </>
          ) : (
            <>
              <Clock className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm text-muted-foreground">
                Awaiting second approval
              </span>
            </>
          )}
        </div>
      </div>

      {/* Rejection info if present */}
      {task.approval_status?.rejection && (
        <div className="flex items-start gap-2 rounded border border-status-red/50 bg-status-red/10 p-2 mt-2">
          <XCircle className="h-4 w-4 text-status-red shrink-0 mt-0.5" />
          <div>
            <p className="text-sm text-status-red">
              Rejected by {task.approval_status.rejection.user_name || "User"}
            </p>
            {task.approval_status.rejection.reason && (
              <p className="text-xs text-muted-foreground mt-1">
                Reason: {task.approval_status.rejection.reason}
              </p>
            )}
          </div>
        </div>
      )}

      {/* Warning if same user tries to approve twice */}
      {approvalsReceived === 1 && task.approval_status?.first_approval && (
        <div className="flex items-start gap-2 rounded border border-status-amber/50 bg-status-amber/10 p-2 mt-2">
          <Shield className="h-4 w-4 text-status-amber shrink-0 mt-0.5" />
          <p className="text-xs text-status-amber">
            A different user must provide the second approval for separation of duties.
          </p>
        </div>
      )}
    </div>
  );
}

// Helper function to format approval timestamps
function formatApprovalTime(timestamp: string): string {
  try {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) return "just now";
    if (diffMins < 60) return `${diffMins}m ago`;

    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h ago`;

    const diffDays = Math.floor(diffHours / 24);
    if (diffDays < 7) return `${diffDays}d ago`;

    return date.toLocaleDateString();
  } catch {
    return timestamp;
  }
}

// Helper function to convert AITask to TaskWithPlan format
function convertToTaskWithPlan(task: AITask): TaskWithPlan {
  const now = new Date().toISOString();
  // Map AITask status to TaskWithPlan state
  const state = task.status === "pending_approval" ? "planned" :
                task.status === "executing" ? "executing" :
                task.status === "completed" ? "completed" :
                task.status === "failed" ? "failed" :
                task.status;

  return {
    id: task.task_id,
    org_id: "", // Not available in AITask, will be resolved by backend
    user_intent: task.task_spec.goal || "",
    task_spec: task.task_spec,
    state: state,
    source: "agent",
    created_at: now,
    updated_at: now,
    risk_level: task.task_spec.risk_level,
    task_type: task.task_spec.task_type,
    hitl_required: task.requires_hitl,
    plan: task.agent_result?.plan ? {
      id: task.task_id + "-plan",
      type: task.task_spec.task_type,
      payload: {
        phases: [],  // Will be extracted from plan if available
        summary: task.agent_result.summary || "",
      },
      state: state,
      created_at: now,
    } : undefined,
  };
}
