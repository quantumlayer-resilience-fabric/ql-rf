"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { useState } from "react";
import { ApiError } from "@/lib/api";

/**
 * Determines if a failed query should be retried.
 * - Don't retry on client errors (4xx)
 * - Do retry on server errors (5xx) and network errors
 */
function shouldRetry(failureCount: number, error: unknown): boolean {
  // Max 2 retries
  if (failureCount >= 2) return false;

  // Don't retry on 4xx errors (client errors like 401, 403, 404)
  if (error instanceof ApiError) {
    if (error.status >= 400 && error.status < 500) {
      return false;
    }
  }

  // Retry on network errors and server errors
  return true;
}

export function QueryProvider({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 60 * 1000, // 1 minute
            gcTime: 5 * 60 * 1000, // 5 minutes (formerly cacheTime)
            retry: shouldRetry,
            refetchOnWindowFocus: false,
          },
          mutations: {
            retry: false, // Don't retry mutations by default
          },
        },
      })
  );

  return (
    <QueryClientProvider client={queryClient}>
      {children}
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  );
}
