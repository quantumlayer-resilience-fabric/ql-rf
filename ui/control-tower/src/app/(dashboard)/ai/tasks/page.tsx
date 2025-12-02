"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { StatusBadge } from "@/components/status/status-badge";
import { GradientText } from "@/components/brand/gradient-text";
import { useAllTasks, TaskWithPlan } from "@/hooks/use-ai";
import { formatDistanceToNow, format } from "date-fns";
import {
  History,
  Search,
  Filter,
  RefreshCw,
  CheckCircle,
  XCircle,
  Clock,
  Bot,
  ChevronRight,
  Loader2,
  Shield,
} from "lucide-react";
import Link from "next/link";

type StateFilter = "all" | "planned" | "approved" | "rejected" | "executing" | "completed" | "failed";

export default function TaskHistoryPage() {
  const [stateFilter, setStateFilter] = useState<StateFilter>("all");
  const [searchQuery, setSearchQuery] = useState("");
  const { data: tasks, isLoading, refetch, isRefetching } = useAllTasks();

  const filteredTasks = tasks?.filter((task) => {
    // Filter by state
    if (stateFilter !== "all" && task.state !== stateFilter) {
      return false;
    }
    // Filter by search query
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      return (
        task.user_intent.toLowerCase().includes(query) ||
        task.task_type?.toLowerCase().includes(query) ||
        task.id.toLowerCase().includes(query)
      );
    }
    return true;
  });

  const getStateIcon = (state: string) => {
    switch (state) {
      case "approved":
      case "completed":
        return <CheckCircle className="h-4 w-4 text-status-green" />;
      case "rejected":
      case "failed":
        return <XCircle className="h-4 w-4 text-destructive" />;
      case "planned":
        return <Clock className="h-4 w-4 text-status-amber" />;
      case "executing":
        return <Loader2 className="h-4 w-4 animate-spin text-brand-accent" />;
      default:
        return <Clock className="h-4 w-4 text-muted-foreground" />;
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

  return (
    <div className="page-transition space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-2">
            <History className="h-6 w-6 text-brand-accent" />
            <h1 className="text-2xl font-bold tracking-tight">
              <GradientText variant="ai">Task History</GradientText>
            </h1>
          </div>
          <p className="text-muted-foreground">
            View and manage AI-generated tasks and their execution history.
          </p>
        </div>
        <Button
          variant="outline"
          onClick={() => refetch()}
          disabled={isRefetching}
        >
          {isRefetching ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="mr-2 h-4 w-4" />
          )}
          Refresh
        </Button>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex flex-col sm:flex-row gap-4">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="Search tasks..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
              />
            </div>
            <Select
              value={stateFilter}
              onValueChange={(value) => setStateFilter(value as StateFilter)}
            >
              <SelectTrigger className="w-[180px]">
                <Filter className="mr-2 h-4 w-4" />
                <SelectValue placeholder="Filter by state" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All States</SelectItem>
                <SelectItem value="planned">Pending Approval</SelectItem>
                <SelectItem value="approved">Approved</SelectItem>
                <SelectItem value="rejected">Rejected</SelectItem>
                <SelectItem value="executing">Executing</SelectItem>
                <SelectItem value="completed">Completed</SelectItem>
                <SelectItem value="failed">Failed</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* Tasks Table */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Bot className="h-5 w-5 text-brand-accent" />
            Tasks
            {filteredTasks && (
              <Badge variant="secondary" className="ml-2">
                {filteredTasks.length}
              </Badge>
            )}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : filteredTasks && filteredTasks.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Task</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Risk</TableHead>
                  <TableHead>State</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredTasks.map((task) => (
                  <TableRow key={task.id}>
                    <TableCell>
                      <div className="max-w-xs">
                        <p className="font-medium truncate">{task.user_intent}</p>
                        <p className="text-xs text-muted-foreground truncate">
                          ID: {task.id.slice(0, 8)}...
                        </p>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline" className="capitalize">
                        {task.task_type?.replace(/_/g, " ") || "Unknown"}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={getRiskBadgeVariant(task.risk_level)}>
                        {task.risk_level || "low"}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        {getStateIcon(task.state)}
                        <StatusBadge status={getStateStatus(task.state)} size="sm">
                          {task.state.replace(/_/g, " ")}
                        </StatusBadge>
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="text-sm">
                        <p>{format(new Date(task.created_at), "MMM d, yyyy")}</p>
                        <p className="text-xs text-muted-foreground">
                          {formatDistanceToNow(new Date(task.created_at), { addSuffix: true })}
                        </p>
                      </div>
                    </TableCell>
                    <TableCell className="text-right">
                      <Link href={`/ai/tasks/${task.id}`}>
                        <Button variant="ghost" size="sm">
                          View
                          <ChevronRight className="ml-1 h-4 w-4" />
                        </Button>
                      </Link>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <History className="h-12 w-12 text-muted-foreground mb-4" />
              <h3 className="text-lg font-medium">No tasks found</h3>
              <p className="text-sm text-muted-foreground mt-1">
                {searchQuery || stateFilter !== "all"
                  ? "Try adjusting your filters"
                  : "Tasks will appear here when you use the AI Copilot"}
              </p>
              <Link href="/ai">
                <Button variant="outline" className="mt-4">
                  Go to AI Copilot
                </Button>
              </Link>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Summary Cards */}
      {tasks && tasks.length > 0 && (
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Total Tasks</p>
                  <p className="text-2xl font-bold">{tasks.length}</p>
                </div>
                <Bot className="h-8 w-8 text-brand-accent/20" />
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Pending</p>
                  <p className="text-2xl font-bold text-status-amber">
                    {tasks.filter((t) => t.state === "planned").length}
                  </p>
                </div>
                <Clock className="h-8 w-8 text-status-amber/20" />
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Approved</p>
                  <p className="text-2xl font-bold text-status-green">
                    {tasks.filter((t) => t.state === "approved" || t.state === "completed").length}
                  </p>
                </div>
                <CheckCircle className="h-8 w-8 text-status-green/20" />
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Rejected</p>
                  <p className="text-2xl font-bold text-destructive">
                    {tasks.filter((t) => t.state === "rejected" || t.state === "failed").length}
                  </p>
                </div>
                <XCircle className="h-8 w-8 text-destructive/20" />
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
