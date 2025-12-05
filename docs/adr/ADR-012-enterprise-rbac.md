# ADR-012: Enterprise RBAC with Hierarchical Roles

## Status
Accepted

## Context

QL-RF is evolving into an enterprise-grade platform serving multiple organizations with diverse team structures and security requirements. The existing simple RBAC model (viewer, operator, engineer, admin) is insufficient for:

1. **Complex Organizational Structures**: Large enterprises have multiple teams with specialized responsibilities (infrastructure, security, DR, compliance)
2. **Fine-Grained Access Control**: Need to control access at the resource level (specific assets, sites, images) not just globally
3. **Team Collaboration**: Teams need to share access to resources without granting individual permissions to each member
4. **Audit and Compliance**: Regulatory requirements demand detailed tracking of who can do what, when permissions were granted/revoked
5. **Multi-Tenant Isolation**: Organizations need strict isolation with no permission leakage across tenants

The ADR-010 basic RBAC implementation provided a foundation but lacked:
- Hierarchical role inheritance
- Resource-level permissions
- Team-based access
- Time-bounded permissions
- Comprehensive audit trail

## Decision

We implement a **hierarchical RBAC system** with the following architecture:

### 1. Eight System Roles

Hierarchical roles with increasing privilege levels:

| Role | Level | Description | Use Case |
|------|-------|-------------|----------|
| `org_owner` | 8 | Full organizational control | CEO, CTO |
| `org_admin` | 7 | Organization administration | IT Director |
| `infra_admin` | 6 | Infrastructure management | Platform Team Lead |
| `security_admin` | 5 | Security and compliance | Security Team Lead |
| `dr_admin` | 4 | Disaster recovery operations | DR Team Lead |
| `operator` | 3 | Day-to-day operations | Operations Engineer |
| `analyst` | 2 | Read-only analysis | Business Analyst |
| `viewer` | 1 | Read-only dashboard access | Stakeholder |

**Role Inheritance**: Higher-level roles inherit all permissions from lower levels (e.g., `infra_admin` inherits `operator` permissions).

### 2. Permission Model

**Actions** (6 types):
- `read` - View resources
- `write` - Create/update resources
- `delete` - Remove resources
- `execute` - Execute operations (AI tasks, DR drills)
- `approve` - Approve pending operations
- `admin` - Administrative actions (RBAC management, integrations)

**Resources** (9 types):
- `assets` - Infrastructure assets
- `images` - Golden images
- `sites` - Geographic sites
- `drift` - Drift reports
- `compliance` - Compliance frameworks and controls
- `dr` - Disaster recovery pairs and drills
- `tasks` - AI tasks
- `organization` - Organization settings
- `audit` - Audit logs

### 3. Three Permission Sources

**Role-Based Permissions**:
```sql
roles → role_permissions → permissions
user_roles → roles (with expiration support)
```

**Direct Resource Permissions**:
```sql
resource_permissions (org_id, resource_type, resource_id, grantee_type, grantee_id, permission)
```
Allows granting specific permissions on individual resources to users.

**Team-Based Permissions**:
```sql
teams → team_members → users
resource_permissions (grantee_type='team')
```
Permissions granted to teams apply to all team members.

### 4. Database Implementation

**Core Tables**:
- `roles` - Role definitions (system and custom)
- `permissions` - Permission definitions
- `role_permissions` - Role-to-permission mappings
- `user_roles` - User role assignments with expiration
- `resource_permissions` - Fine-grained resource-level permissions
- `teams` - Team definitions
- `team_members` - Team membership
- `permission_grants_log` - Audit trail for permission changes

**Database Functions**:
```sql
-- Check if user has permission (combines all three sources)
check_permission(user_id, org_id, resource_type, resource_id, action) → boolean

-- Get all permissions for a user
get_user_permissions(user_id, org_id) → table(permission_name, resource_type, action, source)
```

### 5. Key Features

**Hierarchical Inheritance**:
- Roles can have parent roles (`parent_role_id`)
- Permission checks traverse the hierarchy
- Custom organization-specific roles can inherit from system roles

**Time-Bounded Permissions**:
- `user_roles.expires_at` - Role assignments can expire
- `resource_permissions.expires_at` - Resource permissions can expire
- Automatic cleanup of expired permissions

**Permission Conditions**:
- `resource_permissions.conditions` - JSON field for conditional logic
- Future support for time-of-day, IP-based, or attribute-based access control

**Full Audit Trail**:
- `permission_grants_log` table tracks all permission changes
- Records: grant/revoke action, target user/role/team, resource, timestamp, changed_by, reason

## Consequences

### Positive

1. **Flexibility**: Supports simple flat roles and complex hierarchical structures
2. **Fine-Grained Control**: Resource-level permissions for regulatory compliance
3. **Team Collaboration**: Teams share access without individual grant overhead
4. **Auditability**: Complete permission change history for compliance audits
5. **Performance**: Database-level permission checks with optimized queries and indexes
6. **Multi-Tenancy**: Strong isolation via `org_id` in all permission queries

### Negative

1. **Complexity**: More complex than simple role-based systems
2. **Performance Considerations**: Permission checks involve multiple table joins (mitigated with indexes and caching)
3. **Migration Overhead**: Existing simple roles need mapping to new hierarchy
4. **Learning Curve**: Administrators must understand role hierarchy and permission sources

### Mitigations

1. **Default Roles**: Pre-populated system roles cover 90% of use cases
2. **Database Indexes**: Strategic indexes on permission lookup paths
3. **Caching**: Permission checks cached in application layer (5-minute TTL)
4. **UI Simplification**: Permission management UI hides complexity with wizards and templates
5. **Migration Script**: Automatic migration from ADR-010 simple roles to hierarchical model

## Implementation Notes

### Permission Check Algorithm

```go
1. Check if user has direct permission on resource
2. If not, check team permissions (user's teams)
3. If not, check role-based permissions (user's roles)
4. If role has parent, check parent role permissions (recursive)
5. Return first match or deny
```

### Performance Optimizations

- Materialized view for user effective permissions
- Redis cache for permission check results (TTL: 5 minutes)
- Bulk permission checks for list operations
- Lazy loading of permission details

### Security Considerations

- All permission checks include `org_id` to prevent cross-tenant access
- Permission grants require `admin` action on `organization` resource
- Time-bounded permissions automatically expire (background job)
- Audit log is append-only and tamper-evident

## Migration Path

**Phase 1** (Completed):
- Create new RBAC tables
- Implement database functions
- Implement Go service layer (`pkg/rbac`)

**Phase 2** (In Progress):
- Migrate existing users from ADR-010 roles to new system roles
- Add RBAC middleware to API services
- Update UI with permission gates

**Phase 3** (Planned):
- Implement permission caching layer
- Add RBAC management UI
- Create team management features

## References

- ADR-010: RBAC with Permission-Based Authorization (predecessor)
- Migration 000010: RBAC database schema
- `pkg/rbac/rbac.go`: Core RBAC service implementation
- NIST RBAC Model: https://csrc.nist.gov/projects/role-based-access-control
- OWASP Authorization Cheat Sheet
