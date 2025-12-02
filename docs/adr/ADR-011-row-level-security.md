# ADR-011: Row-Level Security for Multi-Tenancy

## Status
Accepted

## Context

QL-RF is a multi-tenant platform where multiple organizations share the same database. We need to ensure **complete tenant isolation** at the data layer to:

1. Prevent data leakage between organizations
2. Simplify application code by moving isolation to the database
3. Provide defense-in-depth beyond application-level checks
4. Meet compliance requirements for data segregation

### Options Considered

1. **Application-level filtering**: Add `WHERE org_id = ?` to every query
2. **Separate databases per tenant**: Physical isolation
3. **PostgreSQL Row-Level Security (RLS)**: Policy-based row filtering
4. **Schema-per-tenant**: Logical isolation via schemas

## Decision

We implement **PostgreSQL Row-Level Security (RLS)** for tenant isolation.

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     Application                          │
│  ┌─────────────────────────────────────────────────┐    │
│  │  SET LOCAL app.current_org_id = '<org-uuid>'    │    │
│  └─────────────────────────────────────────────────┘    │
│                           ↓                              │
│  ┌─────────────────────────────────────────────────┐    │
│  │         Query: SELECT * FROM assets             │    │
│  └─────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────┐
│                      PostgreSQL                          │
│  ┌─────────────────────────────────────────────────┐    │
│  │  RLS Policy: org_id = current_org_id()          │    │
│  │  → Only returns rows for current tenant         │    │
│  └─────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
```

### Implementation

1. **Session variable** `app.current_org_id` set at connection acquisition
2. **`current_org_id()` function** reads the session variable
3. **RLS policies on all tenant tables** filter by `org_id = current_org_id()`
4. **`TenantConn` wrapper** in application code handles context setup

### Tables with RLS Enabled

| Table | RLS Policy |
|-------|------------|
| `organizations` | `id = current_org_id()` |
| `projects` | `org_id = current_org_id()` |
| `environments` | Via projects join |
| `users` | `org_id = current_org_id()` |
| `images` | `org_id = current_org_id()` |
| `image_coordinates` | Via images join |
| `assets` | `org_id = current_org_id()` |
| `drift_reports` | `org_id = current_org_id()` |
| `connectors` | `org_id = current_org_id()` |
| `ai_tasks` | `org_id = current_org_id()` |
| `ai_plans` | Via ai_tasks join |
| `ai_runs` | Via ai_tasks join |
| `ai_tool_invocations` | Via ai_tasks join |
| `org_ai_settings` | `org_id = current_org_id()` |

### Application Usage

```go
// Using TenantConn for RLS-protected queries
err := db.WithTenantContext(ctx, orgID, func(tc *database.TenantConn) error {
    // All queries automatically filtered to org
    rows, err := tc.Query(ctx, "SELECT * FROM assets")
    return err
})
```

## Consequences

### Positive

- **Defense-in-depth**: Even if application logic fails, database enforces isolation
- **Simpler queries**: No need for `WHERE org_id = ?` in every query
- **Audit-friendly**: Isolation is declarative and verifiable
- **Performance**: RLS policies use indexes efficiently

### Negative

- **Connection overhead**: Must set session variable on each request
- **Testing complexity**: Tests need RLS context setup
- **Migrations awareness**: Must enable RLS on new tables
- **Cross-tenant queries disabled**: Admin operations need workaround

### Mitigations

- Use `TenantConn` wrapper that handles context setup
- Add RLS to migration template checklist
- Superuser connections bypass RLS for admin operations
- Document RLS patterns in developer guide

## References

- PostgreSQL Row-Level Security documentation
- OWASP Multi-Tenancy Security
- Migration: `000005_add_row_level_security.up.sql`
