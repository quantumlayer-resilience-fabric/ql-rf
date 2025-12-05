"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useSendAIMessage, useAIContext, usePendingTasks } from "@/hooks/use-ai";
import { Sparkles, Zap, Clock, Loader2 } from "lucide-react";

interface CertificateInsightCardProps {
  criticalCount: number;
  expiredCount: number;
  expiring7DaysCount: number;
}

export function CertificateInsightCard({
  criticalCount,
  expiredCount,
  expiring7DaysCount,
}: CertificateInsightCardProps) {
  const router = useRouter();
  const [isCreatingAITask, setIsCreatingAITask] = useState(false);

  // AI hooks
  const aiContext = useAIContext();
  const sendAIMessage = useSendAIMessage();
  const { data: pendingTasks = [] } = usePendingTasks();

  const hasPendingCertTask = (pendingTasks || []).some(
    (task) =>
      task.user_intent?.toLowerCase().includes("certificate") ||
      task.user_intent?.toLowerCase().includes("ssl") ||
      task.user_intent?.toLowerCase().includes("tls")
  );

  const handleAIRotation = async () => {
    setIsCreatingAITask(true);
    try {
      const intent =
        criticalCount > 0
          ? `Analyze and rotate ${criticalCount} certificates that are expiring soon or already expired. Prioritize based on blast radius and auto-renewal eligibility.`
          : `Review certificate inventory for security improvements and rotation recommendations.`;

      await sendAIMessage.mutateAsync({
        message: intent,
        context: aiContext,
      });
      router.push("/ai");
    } catch (error) {
      console.error("Failed to create AI task:", error);
    } finally {
      setIsCreatingAITask(false);
    }
  };

  if (criticalCount <= 0) {
    return null;
  }

  const isCritical = criticalCount > 5;

  return (
    <Card
      className={`border-l-4 ${
        isCritical
          ? "border-l-status-red bg-gradient-to-r from-status-red/5 to-transparent"
          : "border-l-status-amber bg-gradient-to-r from-status-amber/5 to-transparent"
      }`}
    >
      <CardContent className="flex items-start gap-4 p-6">
        <div
          className={`rounded-lg p-2 ${
            isCritical ? "bg-status-red/10" : "bg-status-amber/10"
          }`}
        >
          <Sparkles
            className={`h-5 w-5 ${
              isCritical ? "text-status-red" : "text-status-amber"
            }`}
          />
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <h3 className="font-semibold">
              {criticalCount} Certificates Need Attention
            </h3>
            <Badge
              variant="outline"
              className={`text-xs ${
                isCritical
                  ? "border-status-red/50 text-status-red"
                  : "border-status-amber/50 text-status-amber"
              }`}
            >
              {isCritical ? "critical" : "warning"}
            </Badge>
          </div>
          <p className="mt-1 text-sm text-muted-foreground">
            {expiredCount} expired and {expiring7DaysCount} expiring within 7 days.
            AI can analyze blast radius and orchestrate safe rotations.
          </p>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {hasPendingCertTask ? (
            <Button
              size="sm"
              variant="outline"
              onClick={() => router.push("/ai")}
            >
              <Clock className="mr-2 h-4 w-4" />
              View Pending Task
            </Button>
          ) : (
            <Button
              size="sm"
              onClick={handleAIRotation}
              disabled={isCreatingAITask}
              className={isCritical ? "bg-status-red hover:bg-status-red/90" : ""}
            >
              {isCreatingAITask ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Creating...
                </>
              ) : (
                <>
                  <Zap className="mr-2 h-4 w-4" />
                  Rotate with AI
                </>
              )}
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
