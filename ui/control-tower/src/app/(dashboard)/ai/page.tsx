"use client";

import { useState, useRef, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { StatusBadge } from "@/components/status/status-badge";
import { GradientText } from "@/components/brand/gradient-text";
import { useSendAIMessage, useAIContext, useProactiveInsights } from "@/hooks/use-ai";
import {
  Sparkles,
  Send,
  Bot,
  User,
  Lightbulb,
  TrendingDown,
  Shield,
  RefreshCw,
  ChevronRight,
  Copy,
  ThumbsUp,
  ThumbsDown,
  Loader2,
  History,
  Zap,
  AlertCircle,
} from "lucide-react";

interface Message {
  id: string;
  role: "user" | "assistant";
  content: string;
  timestamp: Date;
}

// Suggested prompts for the user
const suggestedPrompts = [
  "What's the current drift situation?",
  "Show me compliance gaps in production",
  "Which images need to be updated?",
  "Analyze DR readiness across regions",
  "Find assets with critical issues",
  "Summarize this week's security posture",
];

export default function AICopilotPage() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const scrollRef = useRef<HTMLDivElement>(null);

  // AI hooks
  const sendMessage = useSendAIMessage();
  const context = useAIContext();
  const proactiveInsights = useProactiveInsights();

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages]);

  const handleSend = async () => {
    if (!input.trim() || sendMessage.isPending) return;

    const userMessage: Message = {
      id: Date.now().toString(),
      role: "user",
      content: input,
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setInput("");

    // Build conversation history for context
    const conversationHistory = messages.map((msg) => ({
      role: msg.role,
      content: msg.content,
    }));

    try {
      const response = await sendMessage.mutateAsync({
        message: input,
        context,
        conversationHistory,
      });

      const assistantMessage: Message = {
        id: (Date.now() + 1).toString(),
        role: "assistant",
        content: response.content,
        timestamp: new Date(),
      };

      setMessages((prev) => [...prev, assistantMessage]);
    } catch (error) {
      // Add error message to chat
      const errorMessage: Message = {
        id: (Date.now() + 1).toString(),
        role: "assistant",
        content: `I apologize, but I encountered an error: ${error instanceof Error ? error.message : "Unknown error"}. Please try again.`,
        timestamp: new Date(),
      };

      setMessages((prev) => [...prev, errorMessage]);
    }
  };

  const handlePromptClick = (prompt: string) => {
    setInput(prompt);
  };

  const handleCopy = (content: string) => {
    navigator.clipboard.writeText(content);
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
            <Badge variant="secondary" className="ml-2">Powered by Claude</Badge>
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
                      <div className="whitespace-pre-wrap text-sm prose prose-sm dark:prose-invert max-w-none">
                        {message.content}
                      </div>
                      {message.role === "assistant" && (
                        <div className="mt-3 flex items-center gap-2">
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-7"
                            onClick={() => handleCopy(message.content)}
                          >
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
                {sendMessage.isPending && (
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
            {sendMessage.isError && (
              <div className="mb-3 flex items-center gap-2 text-sm text-status-red">
                <AlertCircle className="h-4 w-4" />
                Failed to send message. Please try again.
              </div>
            )}
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
                disabled={sendMessage.isPending}
              />
              <Button type="submit" disabled={!input.trim() || sendMessage.isPending}>
                {sendMessage.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Send className="h-4 w-4" />
                )}
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
            {proactiveInsights.length > 0 ? (
              proactiveInsights.map((insight, i) => (
                <div
                  key={i}
                  className="cursor-pointer rounded-lg border p-3 transition-colors hover:border-brand-accent"
                  onClick={() => setInput(`Tell me more about: ${insight.title}`)}
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
                      </div>
                      <p className="mt-1 text-xs text-muted-foreground">
                        {insight.description}
                      </p>
                    </div>
                  </div>
                </div>
              ))
            ) : (
              <div className="text-center text-sm text-muted-foreground py-4">
                <Shield className="h-8 w-8 mx-auto mb-2 text-status-green" />
                <p>All systems healthy!</p>
                <p className="text-xs mt-1">No critical insights at this time.</p>
              </div>
            )}
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
                <ChevronRight className="mr-2 h-3 w-3 text-muted-foreground" />
                {prompt}
              </Button>
            ))}
          </CardContent>
        </Card>

        {/* Context Status */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-base">
              <Zap className="h-4 w-4 text-brand-accent" />
              Context Status
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Fleet Size</span>
              <span className="font-medium">{context.fleetSize?.toLocaleString() ?? "Loading..."}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Drift Score</span>
              <span className="font-medium">{context.driftScore !== undefined ? `${context.driftScore.toFixed(1)}%` : "Loading..."}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Compliance</span>
              <span className="font-medium">{context.complianceScore !== undefined ? `${context.complianceScore.toFixed(1)}%` : "Loading..."}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">DR Readiness</span>
              <span className="font-medium">{context.drReadiness !== undefined ? `${context.drReadiness.toFixed(1)}%` : "Loading..."}</span>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
