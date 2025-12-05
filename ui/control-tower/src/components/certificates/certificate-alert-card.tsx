"use client";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { type CertificateAlert, type AlertSeverity } from "@/lib/api";
import { AlertTriangle, Bell } from "lucide-react";

interface AlertCardProps {
  alert: CertificateAlert;
  onAcknowledge?: (id: string) => void;
  isAcknowledging?: boolean;
}

const severityColors: Record<AlertSeverity, string> = {
  critical: "border-l-status-red bg-status-red/5",
  high: "border-l-status-amber bg-status-amber/5",
  medium: "border-l-yellow-500 bg-yellow-500/5",
  low: "border-l-blue-500 bg-blue-500/5",
};

const severityIcons: Record<AlertSeverity, React.ReactNode> = {
  critical: <AlertTriangle className="h-5 w-5 text-status-red" />,
  high: <AlertTriangle className="h-5 w-5 text-status-amber" />,
  medium: <Bell className="h-5 w-5 text-yellow-500" />,
  low: <Bell className="h-5 w-5 text-blue-500" />,
};

export function CertificateAlertCard({
  alert,
  onAcknowledge,
  isAcknowledging = false,
}: AlertCardProps) {
  return (
    <div
      className={`flex items-start gap-4 rounded-lg border-l-4 p-4 ${severityColors[alert.severity]}`}
    >
      <div className="shrink-0">{severityIcons[alert.severity]}</div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <h4 className="font-medium text-sm">{alert.title}</h4>
          <Badge variant="outline" className="text-xs capitalize">
            {alert.severity}
          </Badge>
        </div>
        <p className="text-sm text-muted-foreground mt-1">{alert.message}</p>
        <div className="text-xs text-muted-foreground mt-2">
          {formatRelativeTime(alert.createdAt)}
        </div>
      </div>
      {alert.status === "open" && onAcknowledge && (
        <Button
          variant="outline"
          size="sm"
          onClick={() => onAcknowledge(alert.id)}
          disabled={isAcknowledging}
        >
          Acknowledge
        </Button>
      )}
    </div>
  );
}

interface AlertListProps {
  alerts: CertificateAlert[];
  onAcknowledge?: (id: string) => void;
  isAcknowledging?: boolean;
}

export function CertificateAlertList({
  alerts,
  onAcknowledge,
  isAcknowledging = false,
}: AlertListProps) {
  return (
    <div className="space-y-3">
      {alerts.map((alert) => (
        <CertificateAlertCard
          key={alert.id}
          alert={alert}
          onAcknowledge={onAcknowledge}
          isAcknowledging={isAcknowledging}
        />
      ))}
    </div>
  );
}

function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return "just now";
  if (diffMins < 60) return `${diffMins} min ago`;

  const diffHours = Math.floor(diffMins / 60);
  if (diffHours < 24) return `${diffHours} hours ago`;

  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 7) return `${diffDays} days ago`;

  return date.toLocaleDateString();
}
