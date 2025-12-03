/**
 * React hooks for RBAC permission checking
 * Mirrors the backend permission model from pkg/models/organization.go
 */

import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@clerk/nextjs";

// Permission constants - mirrors backend PermXxx constants
export const Permissions = {
  READ_DASHBOARD: "read:dashboard",
  READ_DRIFT: "read:drift",
  READ_ASSETS: "read:assets",
  READ_IMAGES: "read:images",
  EXPORT_REPORTS: "export:reports",
  TRIGGER_DRILL: "trigger:drill",
  ACKNOWLEDGE_ALERTS: "acknowledge:alerts",
  EXECUTE_ROLLOUT: "execute:rollout",
  MANAGE_IMAGES: "manage:images",
  APPLY_PATCHES: "apply:patches",
  MANAGE_RBAC: "manage:rbac",
  CONFIGURE_INTEGRATIONS: "configure:integrations",
  APPROVE_EXCEPTIONS: "approve:exceptions",
  APPROVE_AI_TASKS: "approve:ai-tasks",
  EXECUTE_AI_TASKS: "execute:ai-tasks",
} as const;

export type Permission = (typeof Permissions)[keyof typeof Permissions];

// Role constants - mirrors backend Role type
export const Roles = {
  VIEWER: "viewer",
  OPERATOR: "operator",
  ENGINEER: "engineer",
  ADMIN: "admin",
} as const;

export type Role = (typeof Roles)[keyof typeof Roles];

// Role to permissions mapping - mirrors backend RolePermissions
export const RolePermissions: Record<Role, Permission[]> = {
  [Roles.VIEWER]: [
    Permissions.READ_DASHBOARD,
    Permissions.READ_DRIFT,
    Permissions.READ_ASSETS,
    Permissions.READ_IMAGES,
    Permissions.EXPORT_REPORTS,
  ],
  [Roles.OPERATOR]: [
    Permissions.READ_DASHBOARD,
    Permissions.READ_DRIFT,
    Permissions.READ_ASSETS,
    Permissions.READ_IMAGES,
    Permissions.EXPORT_REPORTS,
    Permissions.TRIGGER_DRILL,
    Permissions.ACKNOWLEDGE_ALERTS,
    Permissions.EXECUTE_AI_TASKS,
  ],
  [Roles.ENGINEER]: [
    Permissions.READ_DASHBOARD,
    Permissions.READ_DRIFT,
    Permissions.READ_ASSETS,
    Permissions.READ_IMAGES,
    Permissions.EXPORT_REPORTS,
    Permissions.TRIGGER_DRILL,
    Permissions.ACKNOWLEDGE_ALERTS,
    Permissions.EXECUTE_ROLLOUT,
    Permissions.MANAGE_IMAGES,
    Permissions.APPLY_PATCHES,
    Permissions.EXECUTE_AI_TASKS,
    Permissions.APPROVE_AI_TASKS,
  ],
  [Roles.ADMIN]: [
    Permissions.READ_DASHBOARD,
    Permissions.READ_DRIFT,
    Permissions.READ_ASSETS,
    Permissions.READ_IMAGES,
    Permissions.EXPORT_REPORTS,
    Permissions.TRIGGER_DRILL,
    Permissions.ACKNOWLEDGE_ALERTS,
    Permissions.EXECUTE_ROLLOUT,
    Permissions.MANAGE_IMAGES,
    Permissions.APPLY_PATCHES,
    Permissions.MANAGE_RBAC,
    Permissions.CONFIGURE_INTEGRATIONS,
    Permissions.APPROVE_EXCEPTIONS,
    Permissions.EXECUTE_AI_TASKS,
    Permissions.APPROVE_AI_TASKS,
  ],
};

// User info returned from the backend
export interface UserInfo {
  id: string;
  email: string;
  name: string;
  role: Role;
  org_id: string;
  permissions: Permission[];
}

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

/**
 * Hook to get the current user's info including role and permissions
 */
export function useCurrentUser() {
  const { getToken, isSignedIn } = useAuth();

  return useQuery<UserInfo>({
    queryKey: ["current-user"],
    queryFn: async () => {
      const token = await getToken();
      const response = await fetch(`${API_BASE_URL}/users/me`, {
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
      });

      if (!response.ok) {
        throw new Error("Failed to fetch user info");
      }

      const user = await response.json();

      // Enrich with permissions based on role
      const role = user.role as Role;
      const permissions = RolePermissions[role] || [];

      return {
        ...user,
        permissions,
      };
    },
    enabled: isSignedIn,
    staleTime: 5 * 60 * 1000, // Cache for 5 minutes
  });
}

/**
 * Hook to check if the current user has a specific permission
 */
export function useHasPermission(permission: Permission): boolean {
  const { data: user, isLoading } = useCurrentUser();

  if (isLoading || !user) {
    return false;
  }

  return user.permissions.includes(permission);
}

/**
 * Hook to check if the current user has any of the specified permissions
 */
export function useHasAnyPermission(permissions: Permission[]): boolean {
  const { data: user, isLoading } = useCurrentUser();

  if (isLoading || !user) {
    return false;
  }

  return permissions.some((p) => user.permissions.includes(p));
}

/**
 * Hook to check if the current user has all of the specified permissions
 */
export function useHasAllPermissions(permissions: Permission[]): boolean {
  const { data: user, isLoading } = useCurrentUser();

  if (isLoading || !user) {
    return false;
  }

  return permissions.every((p) => user.permissions.includes(p));
}

/**
 * Hook to get the current user's role
 */
export function useUserRole(): Role | undefined {
  const { data: user, isLoading } = useCurrentUser();

  if (isLoading || !user) {
    return undefined;
  }

  return user.role;
}

/**
 * Hook to check if the current user has at least a certain role level
 */
export function useHasMinimumRole(minimumRole: Role): boolean {
  const role = useUserRole();

  if (!role) {
    return false;
  }

  const roleHierarchy: Role[] = [Roles.VIEWER, Roles.OPERATOR, Roles.ENGINEER, Roles.ADMIN];
  const currentIndex = roleHierarchy.indexOf(role);
  const requiredIndex = roleHierarchy.indexOf(minimumRole);

  return currentIndex >= requiredIndex;
}

/**
 * Hook to get all permissions for a given role
 */
export function getPermissionsForRole(role: Role): Permission[] {
  return RolePermissions[role] || [];
}

/**
 * Check if a role has a specific permission (non-hook version for utilities)
 */
export function roleHasPermission(role: Role, permission: Permission): boolean {
  const permissions = RolePermissions[role];
  return permissions?.includes(permission) ?? false;
}
