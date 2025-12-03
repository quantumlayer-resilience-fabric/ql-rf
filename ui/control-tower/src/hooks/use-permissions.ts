/**
 * React hooks for RBAC permission checking
 * Mirrors the backend permission model from pkg/models/organization.go
 */

import { useQuery } from "@tanstack/react-query";

// Check environment and Clerk configuration
const isDevelopment = process.env.NODE_ENV === "development";
const hasClerkKey =
  typeof process !== "undefined" &&
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY &&
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY.startsWith("pk_") &&
  !process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY.includes("xxxxx");

// Dev/no-auth mode - always returns signed in with dev token
function useDevAuth() {
  return { getToken: async () => "dev-token" as string | null, isSignedIn: true };
}

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
  // Use dev auth when Clerk isn't configured
  const devAuth = useDevAuth();

  // Only try to use Clerk if it's configured - avoid calling the hook otherwise
  // This prevents the "useAuth can only be used within ClerkProvider" error
  const clerkConfigured = hasClerkKey;

  return useQuery<UserInfo>({
    queryKey: ["current-user"],
    queryFn: async () => {
      // In dev mode without Clerk, return a mock admin user
      if (!clerkConfigured && isDevelopment) {
        return {
          id: "dev-user",
          name: "Dev User",
          email: "dev@example.com",
          role: Roles.ADMIN as Role,
          org_id: "dev-org",
          permissions: RolePermissions[Roles.ADMIN],
        };
      }

      // In production without Clerk, throw error
      if (!clerkConfigured) {
        throw new Error("Authentication not configured");
      }

      // Get token - use dev token if Clerk isn't configured
      const token = clerkConfigured ? await devAuth.getToken() : "dev-token";
      if (!token) {
        throw new Error("No authentication token available");
      }

      const response = await fetch(`${API_BASE_URL}/users/me`, {
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
      });

      if (!response.ok) {
        // In development, fallback for convenience
        if (isDevelopment) {
          console.warn("Failed to fetch user info, using dev fallback");
          return {
            id: "demo-user",
            name: "Demo User",
            email: "demo@example.com",
            role: Roles.ADMIN as Role,
            org_id: "dev-org",
            permissions: RolePermissions[Roles.ADMIN],
          };
        }
        // In production, throw the error
        throw new Error(`Failed to fetch user info: ${response.status}`);
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
    // Always enabled in dev mode, otherwise based on Clerk config
    enabled: isDevelopment || clerkConfigured,
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
