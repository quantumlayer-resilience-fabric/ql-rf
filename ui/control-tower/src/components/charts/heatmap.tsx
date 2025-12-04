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
  animate?: boolean;
}

const statusColors = {
  success: {
    bg: "bg-status-green/15",
    bgHover: "hover:bg-status-green/25",
    border: "border-status-green/30",
    borderHover: "hover:border-status-green/60",
    text: "text-status-green",
    gradient: "from-status-green/20 to-status-green/5",
    shadow: "hover:shadow-[0_0_12px_rgba(5,150,105,0.15)]",
  },
  warning: {
    bg: "bg-status-amber/15",
    bgHover: "hover:bg-status-amber/25",
    border: "border-status-amber/30",
    borderHover: "hover:border-status-amber/60",
    text: "text-status-amber",
    gradient: "from-status-amber/20 to-status-amber/5",
    shadow: "hover:shadow-[0_0_12px_rgba(217,119,6,0.15)]",
  },
  critical: {
    bg: "bg-status-red/15",
    bgHover: "hover:bg-status-red/25",
    border: "border-status-red/30",
    borderHover: "hover:border-status-red/60",
    text: "text-status-red",
    gradient: "from-status-red/20 to-status-red/5",
    shadow: "hover:shadow-[0_0_12px_rgba(220,38,38,0.2)]",
  },
};

export function Heatmap({
  data,
  columns = 4,
  showLabels = true,
  showValues = true,
  onCellClick,
  className,
  animate = true,
}: HeatmapProps) {
  return (
    <TooltipProvider>
      <div
        className={cn(
          "grid gap-3",
          className
        )}
        style={{
          gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
        }}
      >
        {data.map((cell, index) => {
          // Default to 'success' if status is not recognized
          const validStatus = (cell.status && statusColors[cell.status]) ? cell.status : 'success';
          const colors = statusColors[validStatus];

          return (
            <Tooltip key={cell.id}>
              <TooltipTrigger asChild>
                <button
                  onClick={() => onCellClick?.(cell)}
                  className={cn(
                    "group relative flex flex-col items-center justify-center overflow-hidden rounded-xl border p-4 transition-all duration-300",
                    colors.bg,
                    colors.bgHover,
                    colors.border,
                    colors.borderHover,
                    colors.shadow,
                    "hover:scale-[1.02]",
                    onCellClick && "cursor-pointer",
                    animate && "animate-in fade-in-0 slide-in-from-bottom-2",
                  )}
                  style={{
                    animationDelay: animate ? `${index * 50}ms` : undefined,
                    animationFillMode: animate ? 'backwards' : undefined,
                  }}
                >
                  {/* Gradient overlay */}
                  <div
                    className={cn(
                      "absolute inset-0 bg-gradient-to-br opacity-0 transition-opacity duration-300 group-hover:opacity-100",
                      colors.gradient
                    )}
                  />

                  <div className="relative z-10">
                    {showValues && (
                      <span
                        className={cn(
                          "text-2xl font-bold tabular-nums tracking-tight",
                          colors.text
                        )}
                        style={{ fontFamily: "var(--font-display)" }}
                      >
                        {typeof cell.value === 'number' ? cell.value.toFixed(1) : cell.value}%
                      </span>
                    )}
                    {showLabels && (
                      <span className="mt-1 block text-xs text-muted-foreground truncate max-w-full">
                        {cell.label}
                      </span>
                    )}
                  </div>
                </button>
              </TooltipTrigger>
              <TooltipContent
                className="border-border/50 bg-popover/95 backdrop-blur-sm"
                sideOffset={8}
              >
                <div className="space-y-1.5 py-1">
                  <p className="font-semibold" style={{ fontFamily: "var(--font-display)" }}>
                    {cell.label}
                  </p>
                  <p className="text-sm">
                    Coverage:{" "}
                    <span className={cn("font-medium", colors.text)}>
                      {typeof cell.value === 'number' ? cell.value.toFixed(1) : cell.value}%
                    </span>
                  </p>
                  {cell.metadata &&
                    Object.entries(cell.metadata).map(([key, value]) => (
                      <p key={key} className="text-xs text-muted-foreground">
                        {key}: <span className="font-medium text-foreground/80">{value}</span>
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
  animate?: boolean;
}

export function DriftBar({ label, value, total, status, animate = true }: DriftBarProps) {
  const percentage = Math.round((value / total) * 100);
  const validStatus = statusColors[status] ? status : 'success';
  const colors = statusColors[validStatus];

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between text-sm">
        <span className="font-medium">{label}</span>
        <span className="text-muted-foreground tabular-nums">
          {value.toLocaleString()} / {total.toLocaleString()} ({percentage}%)
        </span>
      </div>
      <div className="h-3 overflow-hidden rounded-full bg-muted">
        <div
          className={cn(
            "h-full rounded-full transition-all duration-700 ease-out",
            colors.bg.replace("/15", ""),
            animate && "origin-left animate-in slide-in-from-left"
          )}
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  );
}

// Mini sparkline-style progress indicator
interface ProgressSparklineProps {
  value: number;
  status?: "success" | "warning" | "critical";
  showValue?: boolean;
  className?: string;
}

export function ProgressSparkline({
  value,
  status = "success",
  showValue = true,
  className,
}: ProgressSparklineProps) {
  const validStatus = statusColors[status] ? status : 'success';
  const colors = statusColors[validStatus];

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
        <div
          className={cn("h-full rounded-full", colors.bg.replace("/15", ""))}
          style={{ width: `${Math.min(100, Math.max(0, value))}%` }}
        />
      </div>
      {showValue && (
        <span className={cn("text-xs font-medium tabular-nums", colors.text)}>
          {value.toFixed(0)}%
        </span>
      )}
    </div>
  );
}
