"use client";

import { cn } from "@/lib/utils";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface HeatmapCell {
  id: string;
  label: string;
  value: number;
  status: "success" | "warning" | "critical";
  metadata?: Record<string, string | number>;
}

interface HeatmapProps {
  data: HeatmapCell[];
  columns?: number;
  showLabels?: boolean;
  showValues?: boolean;
  onCellClick?: (cell: HeatmapCell) => void;
  className?: string;
}

const statusColors = {
  success: {
    bg: "bg-status-green/20",
    border: "border-status-green/30",
    text: "text-status-green",
    hover: "hover:border-status-green/60",
  },
  warning: {
    bg: "bg-status-amber/20",
    border: "border-status-amber/30",
    text: "text-status-amber",
    hover: "hover:border-status-amber/60",
  },
  critical: {
    bg: "bg-status-red/20",
    border: "border-status-red/30",
    text: "text-status-red",
    hover: "hover:border-status-red/60",
  },
};

export function Heatmap({
  data,
  columns = 4,
  showLabels = true,
  showValues = true,
  onCellClick,
  className,
}: HeatmapProps) {
  return (
    <TooltipProvider>
      <div
        className={cn(
          "grid gap-2",
          className
        )}
        style={{
          gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
        }}
      >
        {data.map((cell) => {
          const colors = statusColors[cell.status];
          return (
            <Tooltip key={cell.id}>
              <TooltipTrigger asChild>
                <button
                  onClick={() => onCellClick?.(cell)}
                  className={cn(
                    "flex flex-col items-center justify-center rounded-lg border p-4 transition-all",
                    colors.bg,
                    colors.border,
                    colors.hover,
                    onCellClick && "cursor-pointer"
                  )}
                >
                  {showValues && (
                    <span className={cn("text-2xl font-bold", colors.text)}>
                      {cell.value}%
                    </span>
                  )}
                  {showLabels && (
                    <span className="mt-1 text-xs text-muted-foreground truncate max-w-full">
                      {cell.label}
                    </span>
                  )}
                </button>
              </TooltipTrigger>
              <TooltipContent>
                <div className="space-y-1">
                  <p className="font-medium">{cell.label}</p>
                  <p className="text-sm">Coverage: {cell.value}%</p>
                  {cell.metadata &&
                    Object.entries(cell.metadata).map(([key, value]) => (
                      <p key={key} className="text-xs text-muted-foreground">
                        {key}: {value}
                      </p>
                    ))}
                </div>
              </TooltipContent>
            </Tooltip>
          );
        })}
      </div>
    </TooltipProvider>
  );
}

// Bar chart for drift by environment
interface DriftBarProps {
  label: string;
  value: number;
  total: number;
  status: "success" | "warning" | "critical";
}

export function DriftBar({ label, value, total, status }: DriftBarProps) {
  const percentage = Math.round((value / total) * 100);
  const colors = statusColors[status];

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between text-sm">
        <span className="font-medium">{label}</span>
        <span className="text-muted-foreground">
          {value.toLocaleString()} / {total.toLocaleString()} ({percentage}%)
        </span>
      </div>
      <div className="h-3 rounded-full bg-muted overflow-hidden">
        <div
          className={cn("h-full rounded-full transition-all", colors.bg.replace("/20", ""))}
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  );
}
