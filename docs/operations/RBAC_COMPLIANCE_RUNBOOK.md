# RBAC and Compliance Operations Runbook

This runbook provides step-by-step procedures for common RBAC and compliance operations.

---

## Table of Contents

1. [RBAC Operations](#rbac-operations)
   - [Onboarding a New User](#onboarding-a-new-user)
   - [Role Assignment](#role-assignment)
   - [Team Management](#team-management)
   - [Permission Auditing](#permission-auditing)
2. [Compliance Operations](#compliance-operations)
   - [Running Assessments](#running-assessments)
   - [Managing Exemptions](#managing-exemptions)
   - [Evidence Collection](#evidence-collection)
   - [Audit Preparation](#audit-preparation)
3. [Troubleshooting](#troubleshooting)

---

## RBAC Operations

### Onboarding a New User

#### Prerequisites
- User has Clerk account
- Organization exists in database
- Admin/owner credentials available

#### Procedure

**Step 1: Verify user exists in Clerk**
```bash
# User should have logged in at least once via Clerk
# Check external_user_id in users table
psql $RF_DATABASE_URL -c "SELECT id, external_user_id, email FROM users WHERE email = 'user@example.com';"
```

**Step 2: Determine appropriate role**

| User Type | Recommended Role |
|-----------|------------------|
| Executive/Manager | viewer, analyst |
| Security Engineer | security_admin |
| Platform Engineer | infra_admin, operator |
| DR Specialist | dr_admin |
| DevOps | operator |
| Auditor | analyst |

**Step 3: Assign role via API**
```bash
# Get the role ID
ROLE_ID=$(curl -s http://localhost:8080/api/v1/rbac/roles \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[] | select(.name=="operator") | .id')

# Assign role to user
curl -X POST "http://localhost:8080/api/v1/rbac/users/$USER_ID/roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"role_id\": \"$ROLE_ID\"}"
```

**Step 4: Verify assignment**
```bash
curl "http://localhost:8080/api/v1/rbac/users/$USER_ID/roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

---

### Role Assignment

#### Assigning Scoped Roles

For site-specific or resource-specific access:

```bash
# Assign role with site scope
curl -X POST "http://localhost:8080/api/v1/rbac/users/$USER_ID/roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role_id": "'"$ROLE_ID"'",
    "scope": {
      "site_ids": ["site-uuid-1", "site-uuid-2"]
    }
  }'
```

#### Revoking a Role

```bash
# Get the user_role ID first
ASSIGNMENT_ID=$(curl -s "http://localhost:8080/api/v1/rbac/users/$USER_ID/roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[0].id')

# Revoke the role
curl -X DELETE "http://localhost:8080/api/v1/rbac/users/$USER_ID/roles/$ASSIGNMENT_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

#### Checking Effective Permissions

```bash
# Check if user can perform action
curl "http://localhost:8080/api/v1/rbac/check?resource=assets&action=write" \
  -H "Authorization: Bearer $USER_TOKEN"

# Response: {"allowed": true/false, "reason": "..."}
```

---

### Team Management

#### Creating a Team

```bash
curl -X POST http://localhost:8080/api/v1/rbac/teams \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "platform-team",
    "display_name": "Platform Engineering Team",
    "description": "Manages platform infrastructure"
  }'
```

#### Adding Members to Team

```bash
curl -X POST "http://localhost:8080/api/v1/rbac/teams/$TEAM_ID/members" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "'"$USER_ID"'",
    "role": "member"
  }'
```

#### Assigning Role to Team

```bash
# All team members inherit this role
curl -X POST "http://localhost:8080/api/v1/rbac/teams/$TEAM_ID/roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"role_id": "'"$ROLE_ID"'"}'
```

---

### Permission Auditing

#### View All User Roles in Organization

```sql
SELECT
    u.email,
    r.name as role_name,
    r.level as role_level,
    ur.scope,
    ur.assigned_at,
    ur.assigned_by
FROM user_roles ur
JOIN users u ON ur.user_id = u.id
JOIN roles r ON ur.role_id = r.id
WHERE ur.org_id = 'your-org-uuid'
ORDER BY r.level DESC, u.email;
```

#### View Permission Changes (Audit Log)

```sql
SELECT
    al.action,
    al.resource_type,
    al.resource_id,
    al.actor_id,
    al.changes,
    al.created_at
FROM audit_logs al
WHERE al.resource_type IN ('user_role', 'role', 'team')
AND al.created_at > NOW() - INTERVAL '7 days'
ORDER BY al.created_at DESC;
```

#### Export Permissions Report

```bash
# Generate CSV of all permissions
psql $RF_DATABASE_URL -c "COPY (
  SELECT u.email, r.name, r.level, ur.scope, ur.assigned_at
  FROM user_roles ur
  JOIN users u ON ur.user_id = u.id
  JOIN roles r ON ur.role_id = r.id
  WHERE ur.org_id = 'your-org-uuid'
) TO STDOUT CSV HEADER" > permissions_report.csv
```

---

## Compliance Operations

### Running Assessments

#### Creating an Assessment

**Step 1: Select framework**
```bash
# List available frameworks
curl http://localhost:8080/api/v1/compliance/frameworks \
  -H "Authorization: Bearer $TOKEN" | jq '.[] | {id, name, version}'
```

**Step 2: Define scope**
```bash
# List sites for scoping
curl http://localhost:8080/api/v1/sites \
  -H "Authorization: Bearer $TOKEN" | jq '.[] | {id, name}'
```

**Step 3: Create assessment**
```bash
curl -X POST http://localhost:8080/api/v1/compliance/assessments \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "framework_id": "framework-uuid",
    "name": "Q4 2025 SOC2 Assessment",
    "description": "Quarterly SOC2 compliance assessment",
    "assessment_type": "automated",
    "scope_sites": ["site-uuid-1", "site-uuid-2"]
  }'
```

**Step 4: Monitor progress**
```bash
# Check assessment status
curl "http://localhost:8080/api/v1/compliance/assessments/$ASSESSMENT_ID" \
  -H "Authorization: Bearer $TOKEN"
```

#### Reviewing Results

```bash
# Get assessment results
curl "http://localhost:8080/api/v1/compliance/assessments/$ASSESSMENT_ID/results" \
  -H "Authorization: Bearer $TOKEN"

# Get failed controls
curl "http://localhost:8080/api/v1/compliance/assessments/$ASSESSMENT_ID/results?status=failed" \
  -H "Authorization: Bearer $TOKEN"
```

---

### Managing Exemptions

#### Creating an Exemption

When a control cannot be met, create an exemption with compensating controls:

```bash
curl -X POST http://localhost:8080/api/v1/compliance/exemptions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "control_id": "control-uuid",
    "reason": "Legacy system incompatible with requirement",
    "risk_acceptance": "Risk accepted by CISO - ticket SEC-1234",
    "compensating_controls": "Additional monitoring implemented, manual review quarterly",
    "expires_at": "2025-06-30T00:00:00Z",
    "review_frequency_days": 30
  }'
```

#### Reviewing Exemptions

```bash
# List active exemptions
curl http://localhost:8080/api/v1/compliance/exemptions \
  -H "Authorization: Bearer $TOKEN"

# Exemptions expiring soon (30 days)
curl "http://localhost:8080/api/v1/compliance/exemptions?expires_within_days=30" \
  -H "Authorization: Bearer $TOKEN"
```

#### Renewing an Exemption

```bash
curl -X PUT "http://localhost:8080/api/v1/compliance/exemptions/$EXEMPTION_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "expires_at": "2025-12-31T00:00:00Z",
    "review_notes": "Renewed after quarterly review. Migration planned for Q3."
  }'
```

---

### Evidence Collection

#### Manual Evidence Upload

```bash
curl -X POST http://localhost:8080/api/v1/compliance/evidence \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: multipart/form-data" \
  -F "control_id=control-uuid" \
  -F "title=Firewall Configuration Screenshot" \
  -F "evidence_type=screenshot" \
  -F "file=@/path/to/evidence.png"
```

#### Automated Evidence (InSpec Reports)

```bash
# Upload InSpec JSON report
curl -X POST http://localhost:8080/api/v1/compliance/evidence \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "control_id": "control-uuid",
    "title": "CIS Level 1 InSpec Report",
    "evidence_type": "report",
    "storage_type": "inline",
    "content": '"$(cat inspec-report.json | jq -c)"'
  }'
```

---

### Audit Preparation

#### Generate Compliance Report

```bash
# Get summary for all frameworks
curl http://localhost:8080/api/v1/compliance/summary \
  -H "Authorization: Bearer $TOKEN"

# Get detailed report for specific framework
curl "http://localhost:8080/api/v1/compliance/report?framework_id=$FRAMEWORK_ID&format=pdf" \
  -H "Authorization: Bearer $TOKEN" > compliance_report.pdf
```

#### Export Evidence Pack

```bash
# Export all evidence for an assessment
curl "http://localhost:8080/api/v1/compliance/assessments/$ASSESSMENT_ID/export" \
  -H "Authorization: Bearer $TOKEN" > evidence_pack.zip
```

#### Audit Trail Export

```sql
-- Export audit log for compliance operations
COPY (
  SELECT
    created_at,
    actor_id,
    action,
    resource_type,
    resource_id,
    changes,
    ip_address,
    user_agent
  FROM audit_logs
  WHERE resource_type IN ('compliance_assessment', 'compliance_exemption', 'compliance_evidence')
  AND created_at BETWEEN '2025-01-01' AND '2025-12-31'
  ORDER BY created_at
) TO '/tmp/compliance_audit_2025.csv' CSV HEADER;
```

---

## Troubleshooting

### User Cannot Access Resource

**Symptoms:** 403 Forbidden response

**Diagnosis:**
```bash
# 1. Check user's roles
curl "http://localhost:8080/api/v1/rbac/users/$USER_ID/roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# 2. Check specific permission
curl "http://localhost:8080/api/v1/rbac/check?resource=assets&action=write&user_id=$USER_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# 3. Check role permissions in database
psql $RF_DATABASE_URL -c "
  SELECT r.name, p.resource_type, p.action
  FROM user_roles ur
  JOIN roles r ON ur.role_id = r.id
  JOIN role_permissions rp ON r.id = rp.role_id
  JOIN permissions p ON rp.permission_id = p.id
  WHERE ur.user_id = '$USER_ID';
"
```

**Resolution:**
- Assign appropriate role
- Check scope restrictions
- Verify organization membership

---

### Assessment Stuck in "pending"

**Symptoms:** Assessment never progresses to "in_progress"

**Diagnosis:**
```sql
SELECT id, status, started_at, error_message
FROM compliance_assessments
WHERE id = 'assessment-uuid';
```

**Resolution:**
1. Check Temporal workflow status
2. Restart assessment if stuck:
```bash
curl -X POST "http://localhost:8080/api/v1/compliance/assessments/$ASSESSMENT_ID/restart" \
  -H "Authorization: Bearer $TOKEN"
```

---

### Exemption Not Applied

**Symptoms:** Control still shows as failed despite exemption

**Diagnosis:**
```sql
-- Check exemption status
SELECT id, status, expires_at, control_id
FROM compliance_exemptions
WHERE control_id = 'control-uuid'
AND org_id = 'org-uuid';

-- Verify exemption is active and not expired
SELECT * FROM compliance_exemptions
WHERE id = 'exemption-uuid'
AND status = 'active'
AND expires_at > NOW();
```

**Resolution:**
- Check exemption status is "active"
- Verify exemption hasn't expired
- Re-run assessment to apply exemption

---

### Quota Exceeded

**Symptoms:** 429 or 403 with quota error

**Diagnosis:**
```bash
curl http://localhost:8080/api/v1/organization/usage \
  -H "Authorization: Bearer $TOKEN"
```

**Resolution:**
- Upgrade subscription plan
- Request temporary quota increase
- Clean up unused resources

---

## Emergency Procedures

### Lockout Recovery (Lost Admin Access)

```sql
-- EMERGENCY ONLY: Direct database role assignment
-- Requires database admin access

-- 1. Find org_owner role ID
SELECT id FROM roles WHERE name = 'org_owner';

-- 2. Find user ID
SELECT id FROM users WHERE email = 'admin@example.com';

-- 3. Assign org_owner role directly
INSERT INTO user_roles (user_id, org_id, role_id, assigned_by, assigned_at)
VALUES ('user-uuid', 'org-uuid', 'role-uuid', 'system', NOW());
```

### Disable RBAC Temporarily (Development Only)

```bash
# Set dev mode to bypass authentication
export RF_DEV_MODE=true
export RF_ORCHESTRATOR_DEV_MODE=true

# Restart services
docker-compose restart api orchestrator
```

---

## Best Practices

1. **Principle of Least Privilege**: Start with `viewer` role, escalate as needed
2. **Regular Audits**: Review permissions monthly
3. **Document Exemptions**: Always include compensating controls
4. **Evidence Freshness**: Collect evidence at least quarterly
5. **Team-Based Access**: Use teams for department-wide permissions
6. **Scope Restrictions**: Limit roles to specific sites when possible
7. **Expiration Policies**: Set expiration on temporary elevated access
