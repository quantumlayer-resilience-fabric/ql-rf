"use client";

import { StatusBadge } from "@/components/status/status-badge";
import { Clock, Play, CheckCircle, XCircle, Ban } from "lucide-react";

type ScanStatus = "pending" | "running" | "completed" | "failed" | "cancelled";

interface ScanStatusBadgeProps {
  status: ScanStatus;
  size?: "sm" | "md" | "lg";
}

export function ScanStatusBadge({ status, size = "md" }: ScanStatusBadgeProps) {
  const statusMap: Record<
    ScanStatus,
    { variant: "success" | "warning" | "critical" | "neutral" | "info"; label: string; icon?: React.ReactNode }
  > = {
    pending: {
      variant: "neutral",
      label: "Pending",
      icon: <Clock className="h-3 w-3" />,
    },
    running: {
      variant: "info",
      label: "Running",
      icon: <Play className="h-3 w-3" />,
    },
    completed: {
      variant: "success",
      label: "Completed",
      icon: <CheckCircle className="h-3 w-3" />,
    },
    failed: {
      variant: "critical",
      label: "Failed",
      icon: <XCircle className="h-3 w-3" />,
    },
    cancelled: {
      variant: "neutral",
      label: "Cancelled",
      icon: <Ban className="h-3 w-3" />,
    },
  };

  const config = statusMap[status];
  const shouldPulse = status === "running";

  return (
    <StatusBadge status={config.variant} pulse={shouldPulse} size={size}>
      {config.label}
    </StatusBadge>
  );
}
