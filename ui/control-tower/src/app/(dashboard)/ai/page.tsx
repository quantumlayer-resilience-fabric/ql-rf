"use client";

import { useState, useRef, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { StatusBadge } from "@/components/status/status-badge";
import { GradientText } from "@/components/brand/gradient-text";
import {
  Sparkles,
  Send,
  Bot,
  User,
  Lightbulb,
  TrendingDown,
  Shield,
  RefreshCw,
  AlertTriangle,
  ChevronRight,
  Copy,
  ThumbsUp,
  ThumbsDown,
  Loader2,
  History,
  Trash2,
  Zap,
} from "lucide-react";

interface Message {
  id: string;
  role: "user" | "assistant";
  content: string;
  timestamp: Date;
  insights?: Insight[];
}

interface Insight {
  type: "drift" | "compliance" | "dr" | "optimization";
  title: string;
  description: string;
  action?: string;
  severity?: "critical" | "warning" | "info";
}

// Sample insights for AI to surface
const suggestedPrompts = [
  "What's causing the drift in ap-south-1?",
  "Show me compliance gaps in production",
  "Which images need to be updated?",
  "Analyze DR readiness for US regions",
  "Find assets with failed deployments",
  "Summarize this week's security posture",
];

const proactiveInsights: Insight[] = [
  {
    type: "drift",
    title: "Drift Pattern Detected",
    description: "47 assets in ap-south-1 are running outdated images. Root cause: failed deployment pipeline 12 days ago.",
    action: "View Remediation Plan",
    severity: "critical",
  },
  {
    type: "compliance",
    title: "CIS Benchmark Gap",
    description: "12 servers missing SSH protocol 2 configuration. This affects your CIS compliance score.",
    action: "Auto-Remediate",
    severity: "warning",
  },
  {
    type: "optimization",
    title: "Cost Optimization",
    description: "23 staging instances haven't been used in 30 days. Potential monthly savings: $2,340.",
    action: "Review Instances",
    severity: "info",
  },
];

// Simulated AI responses
const aiResponses: Record<string, { content: string; insights?: Insight[] }> = {
  "drift": {
    content: `Based on my analysis of the ap-south-1 region, I've identified the root cause of the drift:

**Root Cause Analysis:**
- A deployment pipeline failure occurred on January 3rd, 2024
- The pipeline was attempting to roll out ql-base-linux v1.6.4
- The failure was caused by a network timeout to the image registry

**Affected Assets:**
- 47 EC2 instances running ql-base-linux v1.4.2 (should be v1.6.4)
- Primarily web tier servers in the production environment

**Recommended Actions:**
1. Re-trigger the deployment pipeline for ap-south-1
2. Increase registry timeout from 30s to 120s
3. Add retry logic to the deployment job

Would you like me to create a remediation ticket or show you the affected assets?`,
    insights: [
      {
        type: "drift",
        title: "Re-trigger Deployment",
        description: "Run the deployment pipeline to update 47 assets to v1.6.4",
        action: "Execute",
        severity: "critical",
      },
    ],
  },
  "compliance": {
    content: `I've analyzed your production environment for compliance gaps. Here's what I found:

**Current Compliance Status:**
- Overall Score: 97.8%
- CIS Benchmarks: 96.2% (6 failing controls)
- HIPAA: 89.2% (5 failing controls)

**Critical Gaps:**
1. **SSH Protocol Version** (CIS-4.2.1)
   - 12 servers not enforcing SSH protocol 2
   - Risk: Vulnerable to man-in-the-middle attacks

2. **Access Control** (HIPAA-164.312)
   - 5 systems lacking unique user identification
   - Risk: Compliance violation

**Quick Wins:**
- The SSH fix can be auto-remediated across all 12 servers
- Estimated time: 5 minutes
- No downtime required

Shall I proceed with the auto-remediation?`,
    insights: [
      {
        type: "compliance",
        title: "Auto-fix SSH Config",
        description: "Apply SSH protocol 2 configuration to 12 servers",
        action: "Auto-Remediate",
        severity: "warning",
      },
    ],
  },
  "default": {
    content: `I'm analyzing your infrastructure now. Based on the current state:

**Infrastructure Overview:**
- 12,847 total assets across 5 platforms
- 94.2% drift coverage (749 drifted assets)
- 97.8% compliance score
- 98.1% DR readiness

**Key Observations:**
1. Drift increased by 2.1% in the last 7 days
2. ap-south-1 region has the lowest coverage at 62.4%
3. 3 critical alerts require immediate attention

Is there a specific area you'd like me to dive deeper into?`,
  },
};

export default function AICopilotPage() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages]);

  const handleSend = async () => {
    if (!input.trim() || isLoading) return;

    const userMessage: Message = {
      id: Date.now().toString(),
      role: "user",
      content: input,
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setInput("");
    setIsLoading(true);

    // Simulate AI thinking
    await new Promise((resolve) => setTimeout(resolve, 1500));

    // Determine response based on keywords
    let response = aiResponses.default;
    if (input.toLowerCase().includes("drift") || input.toLowerCase().includes("ap-south")) {
      response = aiResponses.drift;
    } else if (input.toLowerCase().includes("compliance") || input.toLowerCase().includes("gap")) {
      response = aiResponses.compliance;
    }

    const assistantMessage: Message = {
      id: (Date.now() + 1).toString(),
      role: "assistant",
      content: response.content,
      timestamp: new Date(),
      insights: response.insights,
    };

    setMessages((prev) => [...prev, assistantMessage]);
    setIsLoading(false);
  };

  const handlePromptClick = (prompt: string) => {
    setInput(prompt);
  };

  return (
    <div className="page-transition flex h-[calc(100vh-theme(spacing.32))] gap-6">
      {/* Main Chat Area */}
      <div className="flex flex-1 flex-col">
        {/* Header */}
        <div className="mb-4">
          <div className="flex items-center gap-2">
            <Sparkles className="h-6 w-6 text-brand-accent" />
            <h1 className="text-2xl font-bold tracking-tight">
              <GradientText variant="ai">AI Copilot</GradientText>
            </h1>
            <Badge variant="secondary" className="ml-2">Beta</Badge>
          </div>
          <p className="text-muted-foreground">
            Ask questions about your infrastructure and get AI-powered insights.
          </p>
        </div>

        {/* Chat Area */}
        <Card className="flex flex-1 flex-col overflow-hidden">
          <ScrollArea className="flex-1 p-4" ref={scrollRef}>
            {messages.length === 0 ? (
              <div className="flex h-full flex-col items-center justify-center text-center">
                <div className="rounded-full bg-gradient-to-r from-brand-accent/20 to-purple-500/20 p-6">
                  <Bot className="h-12 w-12 text-brand-accent" />
                </div>
                <h3 className="mt-4 text-lg font-semibold">
                  How can I help you today?
                </h3>
                <p className="mt-2 max-w-sm text-sm text-muted-foreground">
                  I can analyze your infrastructure, identify issues, suggest optimizations, and help you maintain compliance.
                </p>
                <div className="mt-6 flex flex-wrap justify-center gap-2">
                  {suggestedPrompts.slice(0, 3).map((prompt) => (
                    <Button
                      key={prompt}
                      variant="outline"
                      size="sm"
                      onClick={() => handlePromptClick(prompt)}
                    >
                      {prompt}
                    </Button>
                  ))}
                </div>
              </div>
            ) : (
              <div className="space-y-4">
                {messages.map((message) => (
                  <div
                    key={message.id}
                    className={`flex gap-3 ${
                      message.role === "user" ? "justify-end" : ""
                    }`}
                  >
                    {message.role === "assistant" && (
                      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-gradient-to-r from-brand-accent to-purple-500">
                        <Bot className="h-4 w-4 text-white" />
                      </div>
                    )}
                    <div
                      className={`max-w-[80%] rounded-lg p-4 ${
                        message.role === "user"
                          ? "bg-brand-accent text-white"
                          : "bg-muted"
                      }`}
                    >
                      <div className="whitespace-pre-wrap text-sm">
                        {message.content}
                      </div>
                      {message.insights && message.insights.length > 0 && (
                        <div className="mt-4 space-y-2">
                          {message.insights.map((insight, i) => (
                            <div
                              key={i}
                              className="flex items-center justify-between rounded-lg border bg-background p-3"
                            >
                              <div className="flex items-center gap-2">
                                <Zap className="h-4 w-4 text-brand-accent" />
                                <span className="text-sm font-medium">
                                  {insight.title}
                                </span>
                              </div>
                              {insight.action && (
                                <Button size="sm" variant="secondary">
                                  {insight.action}
                                  <ChevronRight className="ml-1 h-3 w-3" />
                                </Button>
                              )}
                            </div>
                          ))}
                        </div>
                      )}
                      {message.role === "assistant" && (
                        <div className="mt-3 flex items-center gap-2">
                          <Button variant="ghost" size="sm" className="h-7">
                            <Copy className="mr-1 h-3 w-3" />
                            Copy
                          </Button>
                          <Button variant="ghost" size="sm" className="h-7">
                            <ThumbsUp className="h-3 w-3" />
                          </Button>
                          <Button variant="ghost" size="sm" className="h-7">
                            <ThumbsDown className="h-3 w-3" />
                          </Button>
                        </div>
                      )}
                    </div>
                    {message.role === "user" && (
                      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-muted">
                        <User className="h-4 w-4" />
                      </div>
                    )}
                  </div>
                ))}
                {isLoading && (
                  <div className="flex gap-3">
                    <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-gradient-to-r from-brand-accent to-purple-500">
                      <Bot className="h-4 w-4 text-white" />
                    </div>
                    <div className="rounded-lg bg-muted p-4">
                      <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                    </div>
                  </div>
                )}
              </div>
            )}
          </ScrollArea>

          {/* Input Area */}
          <div className="border-t p-4">
            <form
              onSubmit={(e) => {
                e.preventDefault();
                handleSend();
              }}
              className="flex gap-2"
            >
              <Input
                placeholder="Ask about your infrastructure..."
                value={input}
                onChange={(e) => setInput(e.target.value)}
                disabled={isLoading}
              />
              <Button type="submit" disabled={!input.trim() || isLoading}>
                <Send className="h-4 w-4" />
              </Button>
            </form>
          </div>
        </Card>
      </div>

      {/* Sidebar - Proactive Insights */}
      <div className="hidden w-80 space-y-4 lg:block">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-base">
              <Lightbulb className="h-4 w-4 text-status-amber" />
              Proactive Insights
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {proactiveInsights.map((insight, i) => (
              <div
                key={i}
                className="cursor-pointer rounded-lg border p-3 transition-colors hover:border-brand-accent"
                onClick={() =>
                  setInput(
                    insight.type === "drift"
                      ? "What's causing the drift in ap-south-1?"
                      : insight.type === "compliance"
                      ? "Show me compliance gaps in production"
                      : "Tell me about the unused staging instances"
                  )
                }
              >
                <div className="flex items-start gap-2">
                  {insight.type === "drift" && (
                    <TrendingDown className="h-4 w-4 text-status-red" />
                  )}
                  {insight.type === "compliance" && (
                    <Shield className="h-4 w-4 text-status-amber" />
                  )}
                  {insight.type === "dr" && (
                    <RefreshCw className="h-4 w-4 text-purple-500" />
                  )}
                  {insight.type === "optimization" && (
                    <Zap className="h-4 w-4 text-brand-accent" />
                  )}
                  <div className="flex-1">
                    <div className="flex items-center justify-between">
                      <h4 className="text-sm font-medium">{insight.title}</h4>
                      {insight.severity && (
                        <StatusBadge
                          status={
                            insight.severity === "critical"
                              ? "critical"
                              : insight.severity === "warning"
                              ? "warning"
                              : "info"
                          }
                          size="sm"
                        >
                          {insight.severity}
                        </StatusBadge>
                      )}
                    </div>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {insight.description}
                    </p>
                  </div>
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-base">
              <History className="h-4 w-4" />
              Suggested Questions
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {suggestedPrompts.map((prompt) => (
              <Button
                key={prompt}
                variant="ghost"
                className="w-full justify-start text-left text-sm font-normal h-auto py-2"
                onClick={() => handlePromptClick(prompt)}
              >
                {prompt}
              </Button>
            ))}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
