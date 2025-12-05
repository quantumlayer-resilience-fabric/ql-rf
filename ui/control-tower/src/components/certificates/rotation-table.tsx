"use client";

import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { type CertificateRotation, type RotationStatus } from "@/lib/api";
import { Sparkles } from "lucide-react";

const statusColors: Record<RotationStatus, string> = {
  pending: "bg-gray-100 text-gray-700",
  in_progress: "bg-blue-100 text-blue-700",
  completed: "bg-green-100 text-green-700",
  failed: "bg-red-100 text-red-700",
  rolled_back: "bg-amber-100 text-amber-700",
};

interface RotationRowProps {
  rotation: CertificateRotation;
  onRowClick?: (rotation: CertificateRotation) => void;
}

export function RotationRow({ rotation, onRowClick }: RotationRowProps) {
  const successRate =
    rotation.affectedUsages > 0
      ? Math.round((rotation.successfulUpdates / rotation.affectedUsages) * 100)
      : 0;

  return (
    <TableRow
      className={onRowClick ? "cursor-pointer hover:bg-muted/50" : undefined}
      onClick={() => onRowClick?.(rotation)}
    >
      <TableCell className="capitalize">{rotation.rotationType}</TableCell>
      <TableCell>
        <div className="flex items-center gap-2">
          {rotation.initiatedBy === "ai_agent" && (
            <Sparkles className="h-4 w-4 text-primary" />
          )}
          <span className="capitalize">{rotation.initiatedBy.replace(/_/g, " ")}</span>
        </div>
      </TableCell>
      <TableCell>
        <Badge className={statusColors[rotation.status]}>
          {rotation.status.replace(/_/g, " ")}
        </Badge>
      </TableCell>
      <TableCell>{rotation.affectedUsages}</TableCell>
      <TableCell>
        <SuccessRateDisplay
          successfulUpdates={rotation.successfulUpdates}
          totalUsages={rotation.affectedUsages}
        />
      </TableCell>
      <TableCell>
        {rotation.startedAt ? formatRelativeTime(rotation.startedAt) : "-"}
      </TableCell>
    </TableRow>
  );
}

interface SuccessRateDisplayProps {
  successfulUpdates: number;
  totalUsages: number;
}

export function SuccessRateDisplay({
  successfulUpdates,
  totalUsages,
}: SuccessRateDisplayProps) {
  const successRate = totalUsages > 0
    ? Math.round((successfulUpdates / totalUsages) * 100)
    : 0;

  const colorClass =
    successRate === 100
      ? "text-status-green"
      : successRate > 0
        ? "text-status-amber"
        : "text-muted-foreground";

  return (
    <div className="flex items-center gap-2">
      <span className={colorClass}>{successRate}%</span>
      <span className="text-xs text-muted-foreground">
        ({successfulUpdates}/{totalUsages})
      </span>
    </div>
  );
}

interface RotationTableProps {
  rotations: CertificateRotation[];
  onRowClick?: (rotation: CertificateRotation) => void;
}

export function RotationTable({ rotations, onRowClick }: RotationTableProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Type</TableHead>
          <TableHead>Initiated By</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Affected</TableHead>
          <TableHead>Success Rate</TableHead>
          <TableHead>Started</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rotations.map((rotation) => (
          <RotationRow
            key={rotation.id}
            rotation={rotation}
            onRowClick={onRowClick}
          />
        ))}
      </TableBody>
    </Table>
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
