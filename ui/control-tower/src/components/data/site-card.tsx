"use client";

import { cn } from "@/lib/utils";
import { Card, CardContent } from "@/components/ui/card";
import { StatusBadge } from "@/components/status/status-badge";
import { PlatformIcon } from "@/components/status/platform-icon";
import { Badge } from "@/components/ui/badge";
import {
  Server,
  RefreshCw,
  Shield,
  Clock,
  ArrowRight,
  Link2,
} from "lucide-react";

type Platform = "aws" | "azure" | "gcp" | "vsphere" | "k8s" | "baremetal";
type Status = "healthy" | "warning" | "critical";

interface SiteCardProps {
  name: string;
  region: string;
  platform: Platform;
  environment: "production" | "staging" | "development" | "dr";
  assetCount: number;
  compliantCount: number;
  coveragePercentage: number;
  status: Status;
  lastSyncAt: string;
  drPaired?: string;
  onClick?: () => void;
  className?: string;
}

const statusMap: Record<Status, "success" | "warning" | "critical"> = {
  healthy: "success",
  warning: "warning",
  critical: "critical",
};

const envColors: Record<string, string> = {
  production: "bg-status-green/10 text-status-green border-status-green/20",
  staging: "bg-status-amber/10 text-status-amber border-status-amber/20",
  development: "bg-brand-accent/10 text-brand-accent border-brand-accent/20",
  dr: "bg-purple-500/10 text-purple-500 border-purple-500/20",
};

export function SiteCard({
  name,
  region,
  platform,
  environment,
  assetCount,
  compliantCount,
  coveragePercentage,
  status,
  lastSyncAt,
  drPaired,
  onClick,
  className,
}: SiteCardProps) {
  const driftedCount = assetCount - compliantCount;

  return (
    <Card
      className={cn(
        "cursor-pointer transition-all hover:border-brand-accent hover:shadow-md",
        className
      )}
      onClick={onClick}
    >
      <CardContent className="p-5">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <PlatformIcon platform={platform} size="md" />
            <div>
              <h3 className="font-semibold">{name}</h3>
              <p className="text-sm text-muted-foreground">{region}</p>
            </div>
          </div>
          <StatusBadge status={statusMap[status]} size="sm">
            {status}
          </StatusBadge>
        </div>

        {/* Environment Badge */}
        <div className="mt-3">
          <Badge variant="outline" className={cn("text-xs", envColors[environment])}>
            {environment}
          </Badge>
        </div>

        {/* Metrics */}
        <div className="mt-4 grid grid-cols-3 gap-4">
          <div className="text-center">
            <div className="flex items-center justify-center gap-1">
              <Server className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="text-lg font-bold">{assetCount}</span>
            </div>
            <p className="text-xs text-muted-foreground">Assets</p>
          </div>
          <div className="text-center">
            <div className="flex items-center justify-center gap-1">
              <Shield className="h-3.5 w-3.5 text-status-green" />
              <span className="text-lg font-bold text-status-green">
                {compliantCount}
              </span>
            </div>
            <p className="text-xs text-muted-foreground">Compliant</p>
          </div>
          <div className="text-center">
            <div
              className={cn(
                "text-lg font-bold",
                coveragePercentage >= 90
                  ? "text-status-green"
                  : coveragePercentage >= 70
                  ? "text-status-amber"
                  : "text-status-red"
              )}
            >
              {coveragePercentage}%
            </div>
            <p className="text-xs text-muted-foreground">Coverage</p>
          </div>
        </div>

        {/* Coverage Bar */}
        <div className="mt-4">
          <div className="h-2 rounded-full bg-muted overflow-hidden">
            <div
              className={cn(
                "h-full rounded-full transition-all",
                coveragePercentage >= 90
                  ? "bg-status-green"
                  : coveragePercentage >= 70
                  ? "bg-status-amber"
                  : "bg-status-red"
              )}
              style={{ width: `${coveragePercentage}%` }}
            />
          </div>
        </div>

        {/* Footer */}
        <div className="mt-4 flex items-center justify-between text-xs text-muted-foreground">
          <div className="flex items-center gap-1">
            <Clock className="h-3 w-3" />
            <span>Synced {lastSyncAt}</span>
          </div>
          {drPaired && (
            <div className="flex items-center gap-1 text-purple-500">
              <Link2 className="h-3 w-3" />
              <span>DR: {drPaired}</span>
            </div>
          )}
        </div>

        {/* Drifted indicator */}
        {driftedCount > 0 && (
          <div className="mt-3 flex items-center justify-between rounded-lg bg-status-red/10 px-3 py-2">
            <span className="text-sm text-status-red">
              {driftedCount} drifted assets
            </span>
            <ArrowRight className="h-4 w-4 text-status-red" />
          </div>
        )}
      </CardContent>
    </Card>
  );
}
