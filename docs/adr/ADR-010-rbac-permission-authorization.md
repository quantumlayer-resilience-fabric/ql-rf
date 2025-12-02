# ADR-010: RBAC with Permission-Based Authorization

## Status
Accepted

## Context

QL-RF is a multi-tenant SaaS platform where organizations manage their infrastructure resilience. We need a robust authorization model that:

1. Restricts access to sensitive operations (image management, AI task approval, DR drills)
2. Supports different organizational roles with varying levels of access
3. Integrates with our Clerk-based authentication system
4. Works consistently across API service and AI orchestrator

The PRD defines four roles: Viewer, Operator, Engineer, and Admin. Each role has progressively more permissions.

## Decision

We implement a **permission-based RBAC** system with the following architecture:

### Role Hierarchy

| Role | Description |
|------|-------------|
| `viewer` | Read-only access to dashboards and data |
| `operator` | Can acknowledge alerts, trigger drills, execute AI tasks |
| `engineer` | Can manage images, approve AI tasks, execute rollouts |
| `admin` | Full access including RBAC management and integrations |

### Permission Model

Permissions follow a `action:resource` naming convention:

```
read:dashboard, read:drift, read:assets, read:images
export:reports
trigger:drill
acknowledge:alerts
execute:rollout, execute:ai-tasks
manage:images, manage:rbac
apply:patches
approve:ai-tasks, approve:exceptions
configure:integrations
```

### Implementation

1. **Permissions defined in `pkg/models/organization.go`** as typed constants
2. **Role-to-permission mapping** as a static map for O(1) lookups
3. **`RequirePermission` middleware** checks user's role against required permission
4. **Role propagated from JWT claims** via Clerk authentication

### Middleware Chain

```
Request → Auth Middleware → Extract Role → RequirePermission → Handler
                                ↓
                        Set role in context
```

### Protected Routes

| Endpoint Pattern | Required Permission |
|-----------------|---------------------|
| `POST /images/*` | `manage:images` |
| `POST /alerts/*/acknowledge` | `acknowledge:alerts` |
| `POST /resilience/*/test` | `trigger:drill` |
| `POST /ai/execute` | `execute:ai-tasks` |
| `POST /ai/tasks/*/approve` | `approve:ai-tasks` |

## Consequences

### Positive

- **Fine-grained access control**: Permissions can be checked at route level
- **Extensible**: New permissions can be added without schema changes
- **Auditable**: Role is propagated in context for audit logging
- **Type-safe**: Go's type system prevents permission typos

### Negative

- **Static role definitions**: Adding new roles requires code changes
- **No custom permissions per tenant**: All tenants share same permission model
- **Role sync dependency**: User role must be synced from Clerk to context

### Mitigations

- Use Clerk organization roles which are already synced to JWT claims
- Document permission model clearly for operators
- Consider future ADR for dynamic RBAC if tenant customization needed

## References

- PRD Section 13: Identity and RBAC
- Clerk Organization Roles documentation
- OWASP Authorization Cheat Sheet
