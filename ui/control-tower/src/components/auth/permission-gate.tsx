"use client";

import { ReactNode } from "react";
import {
  useHasPermission,
  useHasAnyPermission,
  useHasAllPermissions,
  useHasMinimumRole,
  Permission,
  Role,
} from "@/hooks/use-permissions";
import { Loader2, ShieldAlert } from "lucide-react";
import { useCurrentUser } from "@/hooks/use-permissions";

interface PermissionGateProps {
  children: ReactNode;
  /**
   * Single permission required to access this content
   */
  permission?: Permission;
  /**
   * Multiple permissions - user must have ALL of them
   */
  permissions?: Permission[];
  /**
   * Multiple permissions - user must have ANY of them
   */
  anyPermission?: Permission[];
  /**
   * Minimum role required to access this content
   */
  minimumRole?: Role;
  /**
   * Content to show when permission is denied.
   * If not provided, nothing is rendered.
   */
  fallback?: ReactNode;
  /**
   * Show loading state while checking permissions
   */
  showLoading?: boolean;
  /**
   * Show a permission denied message instead of nothing
   */
  showDenied?: boolean;
}

/**
 * PermissionGate component for RBAC-based UI visibility.
 * Conditionally renders children based on the current user's permissions or role.
 *
 * @example
 * // Require single permission
 * <PermissionGate permission={Permissions.APPROVE_AI_TASKS}>
 *   <ApproveButton />
 * </PermissionGate>
 *
 * @example
 * // Require all permissions
 * <PermissionGate permissions={[Permissions.MANAGE_IMAGES, Permissions.EXECUTE_ROLLOUT]}>
 *   <DeploySection />
 * </PermissionGate>
 *
 * @example
 * // Require any of the permissions
 * <PermissionGate anyPermission={[Permissions.MANAGE_RBAC, Permissions.APPROVE_EXCEPTIONS]}>
 *   <AdminPanel />
 * </PermissionGate>
 *
 * @example
 * // Require minimum role
 * <PermissionGate minimumRole={Roles.ENGINEER}>
 *   <EngineerTools />
 * </PermissionGate>
 */
export function PermissionGate({
  children,
  permission,
  permissions,
  anyPermission,
  minimumRole,
  fallback,
  showLoading = false,
  showDenied = false,
}: PermissionGateProps) {
  const { isLoading } = useCurrentUser();
  const hasSinglePermission = useHasPermission(permission || ("" as Permission));
  const hasAllPermissions = useHasAllPermissions(permissions || []);
  const hasAnyPermissions = useHasAnyPermission(anyPermission || []);
  const hasMinRole = useHasMinimumRole(minimumRole || ("viewer" as Role));

  // Show loading state
  if (isLoading && showLoading) {
    return (
      <div className="flex items-center justify-center p-4">
        <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
      </div>
    );
  }

  // Determine if access is granted
  let hasAccess = true;

  if (permission) {
    hasAccess = hasAccess && hasSinglePermission;
  }

  if (permissions && permissions.length > 0) {
    hasAccess = hasAccess && hasAllPermissions;
  }

  if (anyPermission && anyPermission.length > 0) {
    hasAccess = hasAccess && hasAnyPermissions;
  }

  if (minimumRole) {
    hasAccess = hasAccess && hasMinRole;
  }

  // If no access criteria specified, default to showing content
  if (!permission && !permissions?.length && !anyPermission?.length && !minimumRole) {
    hasAccess = true;
  }

  // Render based on access
  if (hasAccess) {
    return <>{children}</>;
  }

  // Show fallback if provided
  if (fallback) {
    return <>{fallback}</>;
  }

  // Show denied message if requested
  if (showDenied) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground p-4 border rounded-lg bg-muted/50">
        <ShieldAlert className="h-4 w-4" />
        <span>You don&apos;t have permission to access this content.</span>
      </div>
    );
  }

  // Default: render nothing
  return null;
}

/**
 * Higher-order component version of PermissionGate
 */
export function withPermission<P extends object>(
  WrappedComponent: React.ComponentType<P>,
  permission: Permission
) {
  return function WithPermissionWrapper(props: P) {
    return (
      <PermissionGate permission={permission}>
        <WrappedComponent {...props} />
      </PermissionGate>
    );
  };
}

/**
 * Higher-order component for minimum role requirement
 */
export function withMinimumRole<P extends object>(
  WrappedComponent: React.ComponentType<P>,
  minimumRole: Role
) {
  return function WithMinimumRoleWrapper(props: P) {
    return (
      <PermissionGate minimumRole={minimumRole}>
        <WrappedComponent {...props} />
      </PermissionGate>
    );
  };
}
