"use client";

import { use } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardFooter } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { StatusBadge } from "@/components/status/status-badge";
import { GradientText } from "@/components/brand/gradient-text";
import { useTask, useApproveTask, useRejectTask, useTaskExecutions, TaskWithPlan } from "@/hooks/use-ai";
import { ExecutionStatus } from "@/components/ai/execution-status";
import { formatDistanceToNow, format } from "date-fns";
import ReactMarkdown from "react-markdown";
import {
  ArrowLeft,
  Bot,
  CheckCircle,
  XCircle,
  Clock,
  Loader2,
  AlertTriangle,
  Server,
  Calendar,
  User,
  Shield,
  FileText,
  Edit,
} from "lucide-react";
import Link from "next/link";

interface PageProps {
  params: Promise<{ taskId: string }>;
}

export default function TaskDetailPage({ params }: PageProps) {
  const resolvedParams = use(params);
  const { data: task, isLoading, error } = useTask(resolvedParams.taskId);
  const { data: executions } = useTaskExecutions(resolvedParams.taskId);
  const approveTask = useApproveTask();
  const rejectTask = useRejectTask();

  const latestExecution = executions?.[0];

  const isActionLoading = approveTask.isPending || rejectTask.isPending;

  const handleApprove = async () => {
    if (!task) return;
    try {
      await approveTask.mutateAsync({ taskId: task.id });
    } catch (error) {
      console.error("Failed to approve task:", error);
    }
  };

  const handleReject = async () => {
    if (!task) return;
    try {
      await rejectTask.mutateAsync({ taskId: task.id });
    } catch (error) {
      console.error("Failed to reject task:", error);
    }
  };

  const getStateStatus = (state: string): "success" | "warning" | "critical" | "neutral" | "info" => {
    switch (state) {
      case "approved":
      case "completed":
        return "success";
      case "rejected":
      case "failed":
        return "critical";
      case "planned":
        return "warning";
      case "executing":
        return "info";
      default:
        return "neutral";
    }
  };

  const getRiskBadgeVariant = (risk?: string): "default" | "destructive" | "secondary" | "outline" => {
    switch (risk) {
      case "critical":
      case "high":
        return "destructive";
      case "medium":
        return "secondary";
      default:
        return "outline";
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error || !task) {
    return (
      <div className="flex flex-col items-center justify-center py-24 text-center">
        <AlertTriangle className="h-12 w-12 text-destructive mb-4" />
        <h2 className="text-xl font-semibold">Task Not Found</h2>
        <p className="text-muted-foreground mt-2">
          The task you&apos;re looking for doesn&apos;t exist or has been deleted.
        </p>
        <Link href="/ai/tasks">
          <Button variant="outline" className="mt-4">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Tasks
          </Button>
        </Link>
      </div>
    );
  }

  // Extract task details
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
  const planContent = planPayload?.plan_content as string | undefined;

  return (
    <div className="page-transition space-y-6">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Link href="/ai" className="hover:text-foreground">
          AI Copilot
        </Link>
        <span>/</span>
        <Link href="/ai/tasks" className="hover:text-foreground">
          Tasks
        </Link>
        <span>/</span>
        <span className="text-foreground">{task.id.slice(0, 8)}...</span>
      </div>

      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-brand-accent/10">
              <Bot className="h-5 w-5 text-brand-accent" />
            </div>
            <div>
              <h1 className="text-2xl font-bold tracking-tight">
                <GradientText variant="ai">
                  {taskType.replace(/_/g, " ").replace(/\b\w/g, (l) => l.toUpperCase())}
                </GradientText>
              </h1>
              <p className="text-muted-foreground text-sm">Task ID: {task.id}</p>
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Badge variant={getRiskBadgeVariant(riskLevel)}>{riskLevel}</Badge>
          <StatusBadge status={getStateStatus(task.state)}>
            {task.state.replace(/_/g, " ")}
          </StatusBadge>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-6">
          {/* Request */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                <FileText className="h-4 w-4" />
                Request Details
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <h4 className="text-sm font-medium mb-1">User Intent</h4>
                <p className="text-muted-foreground">{task.user_intent}</p>
              </div>
              {goal && goal !== task.user_intent && (
                <div>
                  <h4 className="text-sm font-medium mb-1">Interpreted Goal</h4>
                  <p className="text-muted-foreground">{goal}</p>
                </div>
              )}
              {summary && (
                <div>
                  <h4 className="text-sm font-medium mb-1">Summary</h4>
                  <p className="text-muted-foreground">{summary}</p>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Plan */}
          {plan && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <Shield className="h-4 w-4" />
                  Execution Plan
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* Phases */}
                {phases && phases.length > 0 && (
                  <div className="space-y-3">
                    <h4 className="text-sm font-medium">Phases ({phases.length})</h4>
                    {phases.map((phase, index) => (
                      <div
                        key={index}
                        className="border-l-2 border-brand-accent/30 pl-4 py-2"
                      >
                        <div className="flex items-center justify-between">
                          <span className="font-medium">
                            {String(phase.name || `Phase ${index + 1}`)}
                          </span>
                          {phase.assets !== undefined && (
                            <Badge variant="outline" className="text-xs">
                              {Array.isArray(phase.assets)
                                ? phase.assets.length
                                : String(phase.assets)}{" "}
                              assets
                            </Badge>
                          )}
                        </div>
                        <div className="mt-1 text-sm text-muted-foreground space-y-1">
                          {phase.wait_time !== undefined && phase.wait_time !== null && (
                            <p>Wait time: {String(phase.wait_time)}</p>
                          )}
                          {phase.rollback_if !== undefined && phase.rollback_if !== null && (
                            <p className="text-status-amber">
                              Rollback if: {String(phase.rollback_if)}
                            </p>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                )}

                {/* Full Plan Content */}
                {planContent && (
                  <div>
                    <h4 className="text-sm font-medium mb-2">Full Plan</h4>
                    <div className="rounded-lg border bg-muted/50 p-4 prose prose-sm dark:prose-invert max-w-none overflow-auto max-h-96">
                      <ReactMarkdown>{planContent}</ReactMarkdown>
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {/* Risk Warning */}
          {(riskLevel === "high" || riskLevel === "critical") && (
            <Card className="border-status-amber/50 bg-status-amber/5">
              <CardContent className="pt-6">
                <div className="flex items-start gap-3">
                  <AlertTriangle className="h-5 w-5 text-status-amber shrink-0" />
                  <div>
                    <h4 className="font-medium text-status-amber">High Risk Operation</h4>
                    <p className="text-sm text-muted-foreground mt-1">
                      This is a {riskLevel}-risk operation that may significantly impact
                      your infrastructure. Please review all details carefully before
                      approving.
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Metadata */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">Environment</span>
                <Badge variant="outline">{environment}</Badge>
              </div>
              <Separator />
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">Affected Assets</span>
                <span className="font-medium">{affectedAssets ?? "N/A"}</span>
              </div>
              <Separator />
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">HITL Required</span>
                <span className="font-medium">{task.hitl_required ? "Yes" : "No"}</span>
              </div>
              <Separator />
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">Source</span>
                <Badge variant="secondary">{task.source}</Badge>
              </div>
            </CardContent>
          </Card>

          {/* Timeline */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Timeline</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-start gap-3">
                <Calendar className="h-4 w-4 text-muted-foreground mt-0.5" />
                <div>
                  <p className="text-sm font-medium">Created</p>
                  <p className="text-xs text-muted-foreground">
                    {format(new Date(task.created_at), "PPpp")}
                  </p>
                </div>
              </div>
              {task.updated_at !== task.created_at && (
                <div className="flex items-start gap-3">
                  <Clock className="h-4 w-4 text-muted-foreground mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Last Updated</p>
                    <p className="text-xs text-muted-foreground">
                      {formatDistanceToNow(new Date(task.updated_at), { addSuffix: true })}
                    </p>
                  </div>
                </div>
              )}
              {plan?.approved_at && (
                <div className="flex items-start gap-3">
                  <CheckCircle className="h-4 w-4 text-status-green mt-0.5" />
                  <div>
                    <p className="text-sm font-medium text-status-green">Approved</p>
                    <p className="text-xs text-muted-foreground">
                      {format(new Date(plan.approved_at), "PPpp")}
                    </p>
                    {plan.approved_by && (
                      <p className="text-xs text-muted-foreground">
                        by {plan.approved_by}
                      </p>
                    )}
                  </div>
                </div>
              )}
              {plan?.rejection_reason && (
                <div className="flex items-start gap-3">
                  <XCircle className="h-4 w-4 text-destructive mt-0.5" />
                  <div>
                    <p className="text-sm font-medium text-destructive">Rejected</p>
                    <p className="text-xs text-muted-foreground">{plan.rejection_reason}</p>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Actions */}
          {/* Execution Status */}
          {latestExecution && (
            <ExecutionStatus execution={latestExecution} />
          )}

          {task.state === "planned" && plan?.state === "awaiting_approval" && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Actions</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <Button
                  className="w-full bg-status-green hover:bg-status-green/90"
                  onClick={handleApprove}
                  disabled={isActionLoading}
                >
                  {approveTask.isPending ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <CheckCircle className="mr-2 h-4 w-4" />
                  )}
                  Approve & Execute
                </Button>
                <Button
                  variant="outline"
                  className="w-full"
                  disabled={isActionLoading}
                >
                  <Edit className="mr-2 h-4 w-4" />
                  Modify Plan
                </Button>
                <Button
                  variant="outline"
                  className="w-full text-destructive hover:text-destructive"
                  onClick={handleReject}
                  disabled={isActionLoading}
                >
                  {rejectTask.isPending ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <XCircle className="mr-2 h-4 w-4" />
                  )}
                  Reject
                </Button>
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      {/* Back Button */}
      <div className="pt-6">
        <Link href="/ai/tasks">
          <Button variant="outline">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Tasks
          </Button>
        </Link>
      </div>
    </div>
  );
}
