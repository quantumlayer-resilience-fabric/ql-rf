"use client";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useAgents, useTools, AgentInfo, ToolInfo } from "@/hooks/use-ai";
import {
  Bot,
  Wrench,
  Shield,
  Database,
  GitBranch,
  HardDrive,
  DollarSign,
  AlertTriangle,
  RefreshCw,
  FileText,
  Activity,
  CheckCircle2,
  XCircle,
  Loader2,
} from "lucide-react";

// Agent icon mapping
const agentIcons: Record<string, React.ElementType> = {
  drift_agent: GitBranch,
  patch_agent: RefreshCw,
  image_agent: HardDrive,
  compliance_agent: Shield,
  security_agent: Shield,
  dr_agent: Database,
  incident_agent: AlertTriangle,
  cost_agent: DollarSign,
  sop_agent: FileText,
  adapter_agent: Activity,
};

// Agent color mapping for visual distinction
const agentColors: Record<string, string> = {
  drift_agent: "text-purple-500 bg-purple-500/10",
  patch_agent: "text-blue-500 bg-blue-500/10",
  image_agent: "text-green-500 bg-green-500/10",
  compliance_agent: "text-amber-500 bg-amber-500/10",
  security_agent: "text-red-500 bg-red-500/10",
  dr_agent: "text-cyan-500 bg-cyan-500/10",
  incident_agent: "text-orange-500 bg-orange-500/10",
  cost_agent: "text-emerald-500 bg-emerald-500/10",
  sop_agent: "text-indigo-500 bg-indigo-500/10",
  adapter_agent: "text-pink-500 bg-pink-500/10",
};

// Agent descriptions for fallback
const agentDescriptions: Record<string, string> = {
  drift_agent: "Detects and analyzes configuration drift between assets and golden images",
  patch_agent: "Plans and orchestrates patch rollouts with phased deployments",
  image_agent: "Manages golden images, promotes versions, and validates compliance",
  compliance_agent: "Audits infrastructure against compliance frameworks (SOC2, CIS, etc.)",
  security_agent: "Performs security assessments and vulnerability analysis",
  dr_agent: "Manages disaster recovery readiness and failover operations",
  incident_agent: "Investigates incidents and identifies root causes",
  cost_agent: "Analyzes cloud costs and suggests optimization opportunities",
  sop_agent: "Generates and validates standard operating procedures",
  adapter_agent: "Provides flexible task adaptation for multi-step workflows",
};

interface AgentCardProps {
  agent: AgentInfo;
}

function AgentCard({ agent }: AgentCardProps) {
  const IconComponent = agentIcons[agent.name] || Bot;
  const colorClass = agentColors[agent.name] || "text-gray-500 bg-gray-500/10";
  const description = agent.description || agentDescriptions[agent.name] || "Specialized AI agent";

  return (
    <Card className="relative overflow-hidden">
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between">
          <div className={`rounded-lg p-2 ${colorClass}`}>
            <IconComponent className="h-5 w-5" />
          </div>
          <Badge
            variant={agent.status === "active" ? "default" : agent.status === "error" ? "destructive" : "secondary"}
            className="text-xs"
          >
            {agent.status === "active" && <CheckCircle2 className="h-3 w-3 mr-1" />}
            {agent.status === "error" && <XCircle className="h-3 w-3 mr-1" />}
            {agent.status === "inactive" && <Loader2 className="h-3 w-3 mr-1" />}
            {agent.status}
          </Badge>
        </div>
        <CardTitle className="text-base font-semibold mt-2">
          {agent.name.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase())}
        </CardTitle>
        <CardDescription className="text-xs line-clamp-2">{description}</CardDescription>
      </CardHeader>
      <CardContent className="pt-0">
        {agent.task_types && agent.task_types.length > 0 && (
          <div className="flex flex-wrap gap-1 mt-2">
            {agent.task_types.slice(0, 3).map((type) => (
              <Badge key={type} variant="outline" className="text-xs">
                {type.replace(/_/g, " ")}
              </Badge>
            ))}
            {agent.task_types.length > 3 && (
              <Badge variant="outline" className="text-xs">
                +{agent.task_types.length - 3} more
              </Badge>
            )}
          </div>
        )}
        {agent.capabilities && agent.capabilities.length > 0 && (
          <div className="mt-3 text-xs text-muted-foreground">
            <span className="font-medium">Capabilities:</span>
            <div className="flex flex-wrap gap-1 mt-1">
              {agent.capabilities.slice(0, 4).map((cap) => (
                <span key={cap} className="text-[10px] px-1.5 py-0.5 bg-muted rounded">
                  {cap}
                </span>
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function AgentCardSkeleton() {
  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between">
          <Skeleton className="h-9 w-9 rounded-lg" />
          <Skeleton className="h-5 w-16" />
        </div>
        <Skeleton className="h-5 w-32 mt-2" />
        <Skeleton className="h-4 w-full mt-2" />
      </CardHeader>
      <CardContent className="pt-0">
        <div className="flex gap-1 mt-2">
          <Skeleton className="h-5 w-16" />
          <Skeleton className="h-5 w-16" />
        </div>
      </CardContent>
    </Card>
  );
}

interface ToolCategoryProps {
  category: string;
  tools: ToolInfo[];
}

function ToolCategory({ category, tools }: ToolCategoryProps) {
  return (
    <div className="space-y-2">
      <h4 className="text-sm font-medium capitalize">{category.replace(/_/g, " ")}</h4>
      <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-2">
        {tools.map((tool) => (
          <div
            key={tool.name}
            className="flex items-center gap-2 p-2 rounded-lg bg-muted/50 hover:bg-muted transition-colors"
          >
            <Wrench className="h-3 w-3 text-muted-foreground shrink-0" />
            <span className="text-xs truncate" title={tool.description}>
              {tool.name}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

export function AgentStatusDashboard() {
  const { data: agents, isLoading: agentsLoading, error: agentsError } = useAgents();
  const { data: tools, isLoading: toolsLoading } = useTools();

  // Group tools by category
  const toolsByCategory = tools?.reduce((acc, tool) => {
    const category = tool.category || "other";
    if (!acc[category]) {
      acc[category] = [];
    }
    acc[category].push(tool);
    return acc;
  }, {} as Record<string, ToolInfo[]>) || {};

  // Calculate stats
  const activeAgents = agents?.filter((a) => a.status === "active").length || 0;
  const totalAgents = agents?.length || 0;
  const totalTools = tools?.length || 0;

  return (
    <div className="space-y-6">
      {/* Summary Stats */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-2">
              <Bot className="h-4 w-4 text-brand-accent" />
              <span className="text-sm text-muted-foreground">Active Agents</span>
            </div>
            <p className="text-2xl font-bold mt-1">
              {agentsLoading ? <Skeleton className="h-8 w-16" /> : `${activeAgents}/${totalAgents}`}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-2">
              <Wrench className="h-4 w-4 text-brand-accent" />
              <span className="text-sm text-muted-foreground">Available Tools</span>
            </div>
            <p className="text-2xl font-bold mt-1">
              {toolsLoading ? <Skeleton className="h-8 w-16" /> : totalTools}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-2">
              <Activity className="h-4 w-4 text-status-green" />
              <span className="text-sm text-muted-foreground">Status</span>
            </div>
            <p className="text-2xl font-bold mt-1 text-status-green">
              {agentsLoading ? <Skeleton className="h-8 w-16" /> : "Healthy"}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-2">
              <Database className="h-4 w-4 text-brand-accent" />
              <span className="text-sm text-muted-foreground">Tool Categories</span>
            </div>
            <p className="text-2xl font-bold mt-1">
              {toolsLoading ? <Skeleton className="h-8 w-8" /> : Object.keys(toolsByCategory).length}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Agents Grid */}
      <div>
        <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Bot className="h-5 w-5" />
          AI Agents ({totalAgents})
        </h3>
        {agentsError ? (
          <Card className="border-destructive">
            <CardContent className="pt-6">
              <div className="flex items-center gap-2 text-destructive">
                <XCircle className="h-5 w-5" />
                <span>Failed to load agents: {agentsError.message}</span>
              </div>
            </CardContent>
          </Card>
        ) : agentsLoading ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {[...Array(8)].map((_, i) => (
              <AgentCardSkeleton key={i} />
            ))}
          </div>
        ) : agents && agents.length > 0 ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {agents.map((agent) => (
              <AgentCard key={agent.name} agent={agent} />
            ))}
          </div>
        ) : (
          <Card>
            <CardContent className="pt-6 text-center text-muted-foreground">
              <Bot className="h-12 w-12 mx-auto mb-2 opacity-50" />
              <p>No agents available</p>
              <p className="text-sm">Check the orchestrator service is running</p>
            </CardContent>
          </Card>
        )}
      </div>

      {/* Tools by Category */}
      <div>
        <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Wrench className="h-5 w-5" />
          Available Tools ({totalTools})
        </h3>
        {toolsLoading ? (
          <Card>
            <CardContent className="pt-6">
              <div className="space-y-4">
                {[...Array(3)].map((_, i) => (
                  <div key={i} className="space-y-2">
                    <Skeleton className="h-4 w-24" />
                    <div className="grid grid-cols-4 gap-2">
                      {[...Array(4)].map((_, j) => (
                        <Skeleton key={j} className="h-8" />
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        ) : Object.keys(toolsByCategory).length > 0 ? (
          <Card>
            <CardContent className="pt-6 space-y-6">
              {Object.entries(toolsByCategory).map(([category, categoryTools]) => (
                <ToolCategory key={category} category={category} tools={categoryTools} />
              ))}
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardContent className="pt-6 text-center text-muted-foreground">
              <Wrench className="h-12 w-12 mx-auto mb-2 opacity-50" />
              <p>No tools available</p>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}
