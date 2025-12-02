import { AlertTriangle, RefreshCw, Home } from "lucide-react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api";

interface ErrorStateProps {
  /** Error object from API or other source */
  error: Error | ApiError | unknown;
  /** Retry function to refetch data */
  retry?: () => void;
  /** Custom title override */
  title?: string;
  /** Custom description override */
  description?: string;
  /** Show home link */
  showHomeLink?: boolean;
}

function getErrorMessage(error: unknown): { title: string; description: string } {
  if (error instanceof ApiError) {
    switch (error.status) {
      case 401:
        return {
          title: "Authentication Required",
          description: "Please sign in to access this resource.",
        };
      case 403:
        return {
          title: "Access Denied",
          description: "You don't have permission to view this resource.",
        };
      case 404:
        return {
          title: "Not Found",
          description: "The requested resource could not be found.",
        };
      case 500:
        return {
          title: "Server Error",
          description: "Something went wrong on our end. Please try again later.",
        };
      case 503:
        return {
          title: "Service Unavailable",
          description: "The service is temporarily unavailable. Please try again in a few minutes.",
        };
      default:
        return {
          title: "Request Failed",
          description: error.message || `Error ${error.status}: ${error.statusText}`,
        };
    }
  }

  if (error instanceof Error) {
    if (error.message.includes("fetch") || error.message.includes("network")) {
      return {
        title: "Connection Error",
        description: "Unable to connect to the server. Please check your internet connection.",
      };
    }
    return {
      title: "Something Went Wrong",
      description: error.message,
    };
  }

  return {
    title: "Unexpected Error",
    description: "An unexpected error occurred. Please try again.",
  };
}

export function ErrorState({
  error,
  retry,
  title,
  description,
  showHomeLink = false,
}: ErrorStateProps) {
  const errorInfo = getErrorMessage(error);

  return (
    <div className="flex min-h-[400px] flex-col items-center justify-center text-center">
      <div className="rounded-full bg-status-red/10 p-4">
        <AlertTriangle className="h-8 w-8 text-status-red" />
      </div>
      <h2 className="mt-4 text-xl font-semibold">
        {title || errorInfo.title}
      </h2>
      <p className="mt-2 max-w-md text-muted-foreground">
        {description || errorInfo.description}
      </p>
      <div className="mt-6 flex gap-3">
        {retry && (
          <Button onClick={retry} variant="default">
            <RefreshCw className="mr-2 h-4 w-4" />
            Try Again
          </Button>
        )}
        {showHomeLink && (
          <Button variant="outline" asChild>
            <Link href="/overview">
              <Home className="mr-2 h-4 w-4" />
              Go to Dashboard
            </Link>
          </Button>
        )}
      </div>
    </div>
  );
}
