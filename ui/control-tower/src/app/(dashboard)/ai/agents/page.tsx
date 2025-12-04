"use client";

import { Button } from "@/components/ui/button";
import { GradientText } from "@/components/brand/gradient-text";
import { AgentStatusDashboard } from "@/components/ai/agent-status-dashboard";
import { Bot, ArrowLeft, RefreshCw } from "lucide-react";
import Link from "next/link";
import { useQueryClient } from "@tanstack/react-query";

export default function AgentsPage() {
  const queryClient = useQueryClient();

  const handleRefresh = () => {
    queryClient.invalidateQueries({ queryKey: ["ai-agents"] });
    queryClient.invalidateQueries({ queryKey: ["ai-tools"] });
  };

  return (
    <div className="page-transition space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2 mb-2">
            <Link href="/ai">
              <Button variant="ghost" size="sm" className="-ml-2">
                <ArrowLeft className="h-4 w-4 mr-1" />
                Back to Copilot
              </Button>
            </Link>
          </div>
          <div className="flex items-center gap-2">
            <Bot className="h-6 w-6 text-brand-accent" />
            <h1 className="text-2xl font-bold tracking-tight">
              <GradientText variant="ai">AI Agents</GradientText>
            </h1>
          </div>
          <p className="text-muted-foreground mt-1">
            Monitor the status and capabilities of all AI agents and their available tools.
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={handleRefresh}>
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh
        </Button>
      </div>

      {/* Dashboard */}
      <AgentStatusDashboard />
    </div>
  );
}
