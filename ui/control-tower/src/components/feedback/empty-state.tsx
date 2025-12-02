import { ReactNode } from "react";
import { Inbox, Search, Plus, Filter, Settings, Database, CheckCircle } from "lucide-react";
import { Button } from "@/components/ui/button";

export type EmptyStateVariant = "default" | "search" | "filter" | "no-data" | "data" | "success";

interface EmptyStateProps {
  /** Variant determines the icon and default messaging */
  variant?: EmptyStateVariant;
  /** Custom icon to display */
  icon?: ReactNode;
  /** Title text */
  title: string;
  /** Description text */
  description?: string;
  /** Primary action button */
  action?: {
    label: string;
    onClick: () => void;
    icon?: ReactNode;
  };
  /** Secondary action link */
  secondaryAction?: {
    label: string;
    onClick: () => void;
  };
}

const variantIcons: Record<EmptyStateVariant, ReactNode> = {
  default: <Inbox className="h-10 w-10" />,
  search: <Search className="h-10 w-10" />,
  filter: <Filter className="h-10 w-10" />,
  "no-data": <Settings className="h-10 w-10" />,
  data: <Database className="h-10 w-10" />,
  success: <CheckCircle className="h-10 w-10 text-status-green" />,
};

export function EmptyState({
  variant = "default",
  icon,
  title,
  description,
  action,
  secondaryAction,
}: EmptyStateProps) {
  const displayIcon = icon || variantIcons[variant];

  return (
    <div className="flex min-h-[300px] flex-col items-center justify-center text-center">
      <div className="rounded-full bg-muted p-4 text-muted-foreground">
        {displayIcon}
      </div>
      <h3 className="mt-4 text-lg font-semibold">{title}</h3>
      {description && (
        <p className="mt-2 max-w-sm text-sm text-muted-foreground">
          {description}
        </p>
      )}
      {(action || secondaryAction) && (
        <div className="mt-6 flex flex-col items-center gap-2">
          {action && (
            <Button onClick={action.onClick}>
              {action.icon || <Plus className="mr-2 h-4 w-4" />}
              {action.label}
            </Button>
          )}
          {secondaryAction && (
            <Button
              variant="link"
              className="text-sm"
              onClick={secondaryAction.onClick}
            >
              {secondaryAction.label}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
