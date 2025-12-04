"use client";

import { useState } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useToolInvocations, ToolInvocation } from "@/hooks/use-ai";
import {
  Wrench,
  CheckCircle2,
  XCircle,
  Clock,
  Eye,
  Code,
  Loader2,
  AlertTriangle,
} from "lucide-react";

interface ToolInvocationAuditProps {
  taskId: string;
}

function InvocationDetails({ invocation }: { invocation: ToolInvocation }) {
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-4 text-sm">
        <div>
          <span className="text-muted-foreground">Tool Name</span>
          <p className="font-medium">{invocation.tool_name}</p>
        </div>
        <div>
          <span className="text-muted-foreground">Duration</span>
          <p className="font-medium">{invocation.duration_ms}ms</p>
        </div>
        <div>
          <span className="text-muted-foreground">Invoked At</span>
          <p className="font-medium">
            {new Date(invocation.invoked_at).toLocaleString()}
          </p>
        </div>
        <div>
          <span className="text-muted-foreground">Status</span>
          <div className="flex items-center gap-2 mt-1">
            {invocation.success ? (
              <>
                <CheckCircle2 className="h-4 w-4 text-status-green" />
                <span className="text-status-green font-medium">Success</span>
              </>
            ) : (
              <>
                <XCircle className="h-4 w-4 text-status-red" />
                <span className="text-status-red font-medium">Failed</span>
              </>
            )}
          </div>
        </div>
      </div>

      {invocation.error && (
        <div className="rounded-lg border border-destructive/20 bg-destructive/5 p-3">
          <div className="flex items-center gap-2 text-destructive mb-2">
            <AlertTriangle className="h-4 w-4" />
            <span className="font-medium">Error</span>
          </div>
          <pre className="text-xs text-destructive overflow-auto whitespace-pre-wrap">
            {invocation.error}
          </pre>
        </div>
      )}

      <div>
        <div className="flex items-center gap-2 mb-2">
          <Code className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm font-medium">Input Parameters</span>
        </div>
        <ScrollArea className="h-40 rounded-lg border bg-muted/50">
          <pre className="p-3 text-xs overflow-auto">
            {JSON.stringify(invocation.parameters, null, 2)}
          </pre>
        </ScrollArea>
      </div>

      <div>
        <div className="flex items-center gap-2 mb-2">
          <Code className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm font-medium">Output Result</span>
        </div>
        <ScrollArea className="h-40 rounded-lg border bg-muted/50">
          <pre className="p-3 text-xs overflow-auto">
            {JSON.stringify(invocation.result, null, 2)}
          </pre>
        </ScrollArea>
      </div>
    </div>
  );
}

function InvocationRow({ invocation }: { invocation: ToolInvocation }) {
  return (
    <TableRow>
      <TableCell>
        <div className="flex items-center gap-2">
          <Wrench className="h-4 w-4 text-muted-foreground" />
          <span className="font-medium">{invocation.tool_name}</span>
        </div>
      </TableCell>
      <TableCell>
        {invocation.success ? (
          <Badge variant="default" className="bg-status-green/20 text-status-green border-status-green/20">
            <CheckCircle2 className="h-3 w-3 mr-1" />
            Success
          </Badge>
        ) : (
          <Badge variant="destructive">
            <XCircle className="h-3 w-3 mr-1" />
            Failed
          </Badge>
        )}
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-1 text-muted-foreground">
          <Clock className="h-3 w-3" />
          <span>{invocation.duration_ms}ms</span>
        </div>
      </TableCell>
      <TableCell className="text-muted-foreground">
        {new Date(invocation.invoked_at).toLocaleTimeString()}
      </TableCell>
      <TableCell>
        <Dialog>
          <DialogTrigger asChild>
            <Button variant="ghost" size="sm">
              <Eye className="h-4 w-4 mr-1" />
              Details
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-2xl">
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <Wrench className="h-5 w-5" />
                Tool Invocation Details
              </DialogTitle>
              <DialogDescription>
                Details of the {invocation.tool_name} tool invocation
              </DialogDescription>
            </DialogHeader>
            <InvocationDetails invocation={invocation} />
          </DialogContent>
        </Dialog>
      </TableCell>
    </TableRow>
  );
}

function InvocationSkeleton() {
  return (
    <TableRow>
      <TableCell>
        <Skeleton className="h-5 w-32" />
      </TableCell>
      <TableCell>
        <Skeleton className="h-5 w-20" />
      </TableCell>
      <TableCell>
        <Skeleton className="h-5 w-16" />
      </TableCell>
      <TableCell>
        <Skeleton className="h-5 w-24" />
      </TableCell>
      <TableCell>
        <Skeleton className="h-8 w-20" />
      </TableCell>
    </TableRow>
  );
}

export function ToolInvocationAudit({ taskId }: ToolInvocationAuditProps) {
  const { data: invocations, isLoading, error } = useToolInvocations(taskId);

  // Calculate stats
  const totalInvocations = invocations?.length || 0;
  const successfulInvocations = invocations?.filter((i) => i.success).length || 0;
  const failedInvocations = totalInvocations - successfulInvocations;
  const totalDuration = invocations?.reduce((sum, i) => sum + i.duration_ms, 0) || 0;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <Wrench className="h-5 w-5" />
          Tool Invocation Audit Trail
        </CardTitle>
        <CardDescription>
          All tool calls made by the AI agent during task execution
        </CardDescription>
      </CardHeader>
      <CardContent>
        {/* Summary Stats */}
        <div className="grid grid-cols-4 gap-4 mb-4">
          <div className="text-center p-3 rounded-lg bg-muted/50">
            <p className="text-2xl font-bold">{totalInvocations}</p>
            <p className="text-xs text-muted-foreground">Total Calls</p>
          </div>
          <div className="text-center p-3 rounded-lg bg-status-green/10">
            <p className="text-2xl font-bold text-status-green">{successfulInvocations}</p>
            <p className="text-xs text-muted-foreground">Successful</p>
          </div>
          <div className="text-center p-3 rounded-lg bg-status-red/10">
            <p className="text-2xl font-bold text-status-red">{failedInvocations}</p>
            <p className="text-xs text-muted-foreground">Failed</p>
          </div>
          <div className="text-center p-3 rounded-lg bg-muted/50">
            <p className="text-2xl font-bold">{totalDuration}ms</p>
            <p className="text-xs text-muted-foreground">Total Time</p>
          </div>
        </div>

        {error ? (
          <div className="flex items-center justify-center py-8 text-destructive">
            <XCircle className="h-5 w-5 mr-2" />
            <span>Failed to load tool invocations</span>
          </div>
        ) : isLoading ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Tool</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Duration</TableHead>
                <TableHead>Time</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {[...Array(5)].map((_, i) => (
                <InvocationSkeleton key={i} />
              ))}
            </TableBody>
          </Table>
        ) : invocations && invocations.length > 0 ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Tool</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Duration</TableHead>
                <TableHead>Time</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {invocations.map((invocation) => (
                <InvocationRow key={invocation.id} invocation={invocation} />
              ))}
            </TableBody>
          </Table>
        ) : (
          <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
            <Wrench className="h-12 w-12 mb-2 opacity-50" />
            <p>No tool invocations recorded</p>
            <p className="text-sm">Tool calls will appear here during task execution</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
