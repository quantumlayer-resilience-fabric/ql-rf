import { Skeleton } from "@/components/ui/skeleton";

interface PageSkeletonProps {
  /** Number of metric cards to show in the header */
  metricCards?: number;
  /** Show a large chart skeleton */
  showChart?: boolean;
  /** Show a table skeleton */
  showTable?: boolean;
  /** Number of table rows to show */
  tableRows?: number;
}

export function PageSkeleton({
  metricCards = 4,
  showChart = true,
  showTable = true,
  tableRows = 5,
}: PageSkeletonProps) {
  return (
    <div className="space-y-6">
      {/* Page header skeleton */}
      <div className="flex items-center justify-between">
        <div className="space-y-2">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-4 w-72" />
        </div>
        <Skeleton className="h-10 w-32" />
      </div>

      {/* Metric cards skeleton */}
      {metricCards > 0 && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: metricCards }).map((_, i) => (
            <div
              key={i}
              className="rounded-lg border border-border bg-card p-6"
            >
              <div className="flex items-center justify-between">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-4 w-4 rounded-full" />
              </div>
              <Skeleton className="mt-3 h-8 w-20" />
              <Skeleton className="mt-2 h-3 w-32" />
            </div>
          ))}
        </div>
      )}

      {/* Chart skeleton */}
      {showChart && (
        <div className="rounded-lg border border-border bg-card p-6">
          <div className="flex items-center justify-between mb-6">
            <Skeleton className="h-5 w-40" />
            <div className="flex gap-2">
              <Skeleton className="h-8 w-24" />
              <Skeleton className="h-8 w-24" />
            </div>
          </div>
          <Skeleton className="h-64 w-full" />
        </div>
      )}

      {/* Table skeleton */}
      {showTable && (
        <div className="rounded-lg border border-border bg-card">
          <div className="border-b border-border p-4">
            <div className="flex items-center justify-between">
              <Skeleton className="h-5 w-32" />
              <div className="flex gap-2">
                <Skeleton className="h-9 w-32" />
                <Skeleton className="h-9 w-24" />
              </div>
            </div>
          </div>
          <div className="p-4">
            {/* Table header */}
            <div className="flex items-center gap-4 border-b border-border pb-3 mb-3">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-4 w-24" />
              ))}
            </div>
            {/* Table rows */}
            {Array.from({ length: tableRows }).map((_, i) => (
              <div key={i} className="flex items-center gap-4 py-3">
                {Array.from({ length: 5 }).map((_, j) => (
                  <Skeleton key={j} className="h-4 w-24" />
                ))}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
