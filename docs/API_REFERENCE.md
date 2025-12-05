# QL-RF API Reference

Complete API reference for QuantumLayer Resilience Fabric.

**Base URL:** `http://localhost:8080/api/v1`
**Authentication:** Bearer token (Clerk JWT)

---

## Table of Contents

1. [Authentication](#authentication)
2. [Core Endpoints](#core-endpoints)
3. [RBAC Endpoints](#rbac-endpoints)
4. [Organization Endpoints](#organization-endpoints)
5. [Compliance Endpoints](#compliance-endpoints)
6. [AI Orchestrator Endpoints](#ai-orchestrator-endpoints)
7. [Error Responses](#error-responses)

---

## Authentication

All API requests require a valid JWT token in the Authorization header:

```
Authorization: Bearer <token>
```

### Development Mode

Set `RF_DEV_MODE=true` to bypass authentication for local development.

---

## Core Endpoints

### Health Check

```
GET /healthz
```

Returns service health status. No authentication required.

**Response:**
```json
{"status": "ok"}
```

### Assets

```
GET /api/v1/assets
GET /api/v1/assets/{id}
POST /api/v1/assets
PUT /api/v1/assets/{id}
DELETE /api/v1/assets/{id}
```

### Images

```
GET /api/v1/images
GET /api/v1/images/{id}
GET /api/v1/images/{id}/lineage
GET /api/v1/images/{id}/vulnerabilities
POST /api/v1/images
PUT /api/v1/images/{id}
```

### Drift

```
GET /api/v1/drift/summary
GET /api/v1/drift/reports
GET /api/v1/drift/top-offenders
```

### Sites

```
GET /api/v1/sites
GET /api/v1/sites/{id}
POST /api/v1/sites
PUT /api/v1/sites/{id}
```

### Overview

```
GET /api/v1/overview/metrics
```

---

## RBAC Endpoints

### Roles

#### List All Roles

```
GET /api/v1/rbac/roles
```

**Response:**
```json
[
  {
    "id": "uuid",
    "name": "org_owner",
    "display_name": "Organization Owner",
    "description": "Full access to all organization resources",
    "level": 100,
    "is_system": true,
    "created_at": "2025-01-01T00:00:00Z"
  }
]
```

#### Get Role by ID

```
GET /api/v1/rbac/roles/{id}
```

#### Create Custom Role

```
POST /api/v1/rbac/roles
```

**Request:**
```json
{
  "name": "site_operator",
  "display_name": "Site Operator",
  "description": "Operator scoped to specific sites",
  "level": 40,
  "permissions": [
    {"resource": "assets", "action": "read"},
    {"resource": "drift", "action": "read"},
    {"resource": "drift", "action": "remediate"}
  ]
}
```

#### Update Role

```
PUT /api/v1/rbac/roles/{id}
```

#### Delete Role

```
DELETE /api/v1/rbac/roles/{id}
```

Note: System roles cannot be deleted.

---

### Permissions

#### List All Permissions

```
GET /api/v1/rbac/permissions
```

**Response:**
```json
[
  {
    "id": "uuid",
    "resource_type": "assets",
    "action": "read",
    "description": "View assets"
  }
]
```

#### Check Permission

```
GET /api/v1/rbac/check?resource={resource}&action={action}
```

**Query Parameters:**
- `resource` - Resource type (assets, images, drift, etc.)
- `action` - Action (read, write, delete, etc.)
- `resource_id` - (Optional) Specific resource ID

**Response:**
```json
{
  "allowed": true,
  "reason": "Role 'operator' grants 'assets:read' permission"
}
```

---

### User Roles

#### Get User Roles

```
GET /api/v1/rbac/users/{userId}/roles
```

**Response:**
```json
[
  {
    "id": "uuid",
    "role": {
      "id": "uuid",
      "name": "operator",
      "display_name": "Operator"
    },
    "scope": {
      "site_ids": ["site-uuid"]
    },
    "assigned_at": "2025-01-01T00:00:00Z",
    "assigned_by": "admin-user-id"
  }
]
```

#### Assign Role to User

```
POST /api/v1/rbac/users/{userId}/roles
```

**Request:**
```json
{
  "role_id": "uuid",
  "scope": {
    "site_ids": ["site-uuid-1", "site-uuid-2"]
  }
}
```

#### Remove Role from User

```
DELETE /api/v1/rbac/users/{userId}/roles/{assignmentId}
```

---

### Teams

#### List Teams

```
GET /api/v1/rbac/teams
```

#### Create Team

```
POST /api/v1/rbac/teams
```

**Request:**
```json
{
  "name": "platform-team",
  "display_name": "Platform Engineering Team",
  "description": "Manages platform infrastructure"
}
```

#### Get Team

```
GET /api/v1/rbac/teams/{id}
```

#### Update Team

```
PUT /api/v1/rbac/teams/{id}
```

#### Delete Team

```
DELETE /api/v1/rbac/teams/{id}
```

#### Team Members

```
GET /api/v1/rbac/teams/{id}/members
POST /api/v1/rbac/teams/{id}/members
DELETE /api/v1/rbac/teams/{id}/members/{userId}
```

#### Team Roles

```
GET /api/v1/rbac/teams/{id}/roles
POST /api/v1/rbac/teams/{id}/roles
DELETE /api/v1/rbac/teams/{id}/roles/{roleId}
```

---

## Organization Endpoints

### Quota

#### Get Organization Quota

```
GET /api/v1/organization/quota
```

**Response:**
```json
{
  "org_id": "uuid",
  "plan_id": "uuid",
  "plan_name": "professional",
  "max_assets": 5000,
  "max_images": 100,
  "max_sites": 25,
  "max_users": 50,
  "max_ai_requests_per_month": 10000,
  "api_rate_limit_per_hour": 10000,
  "features": {
    "advanced_compliance": true,
    "custom_integrations": true,
    "sla_support": true
  }
}
```

#### Update Quota (Admin)

```
PUT /api/v1/organization/quota
```

**Request:**
```json
{
  "max_assets": 10000,
  "max_images": 200,
  "api_rate_limit_per_hour": 20000
}
```

---

### Usage

#### Get Organization Usage

```
GET /api/v1/organization/usage
```

**Response:**
```json
{
  "org_id": "uuid",
  "asset_count": 1250,
  "image_count": 45,
  "site_count": 8,
  "user_count": 23,
  "api_requests_this_hour": 156,
  "ai_requests_this_month": 890,
  "llm_tokens_this_month": 125000,
  "storage_used_bytes": 1073741824,
  "usage_percentages": {
    "assets": 25.0,
    "images": 45.0,
    "sites": 32.0,
    "users": 46.0
  },
  "last_updated": "2025-01-01T12:00:00Z"
}
```

---

### Subscription

#### Get Subscription

```
GET /api/v1/organization/subscription
```

**Response:**
```json
{
  "org_id": "uuid",
  "plan": {
    "id": "uuid",
    "name": "professional",
    "display_name": "Professional",
    "price_monthly": 499.00,
    "price_yearly": 4990.00
  },
  "status": "active",
  "billing_cycle": "monthly",
  "current_period_start": "2025-01-01T00:00:00Z",
  "current_period_end": "2025-02-01T00:00:00Z",
  "cancel_at_period_end": false
}
```

#### Update Subscription

```
PUT /api/v1/organization/subscription
```

**Request:**
```json
{
  "plan_id": "enterprise-plan-uuid",
  "billing_cycle": "yearly"
}
```

---

## Compliance Endpoints

### Frameworks

#### List Frameworks

```
GET /api/v1/compliance/frameworks
```

**Response:**
```json
[
  {
    "id": "uuid",
    "name": "CIS AWS Foundations",
    "description": "Center for Internet Security AWS Foundations Benchmark",
    "category": "Cloud Security",
    "version": "v1.5.0",
    "regulatory_body": "Center for Internet Security",
    "is_system": true,
    "control_count": 13
  }
]
```

#### Get Framework

```
GET /api/v1/compliance/frameworks/{id}
```

#### Get Framework Controls

```
GET /api/v1/compliance/frameworks/{id}/controls
```

**Response:**
```json
[
  {
    "id": "uuid",
    "control_id": "1.1",
    "name": "Avoid the use of root account",
    "description": "The root account should not be used for day-to-day operations",
    "severity": "critical",
    "recommendation": "Create IAM users with appropriate permissions"
  }
]
```

---

### Assessments

#### List Assessments

```
GET /api/v1/compliance/assessments
```

**Query Parameters:**
- `framework_id` - Filter by framework
- `status` - Filter by status (pending, in_progress, completed, failed)
- `limit` - Max results (default: 50)

#### Create Assessment

```
POST /api/v1/compliance/assessments
```

**Request:**
```json
{
  "framework_id": "uuid",
  "name": "Q4 2025 SOC2 Assessment",
  "description": "Quarterly SOC2 compliance assessment",
  "assessment_type": "automated",
  "scope_sites": ["uuid-1", "uuid-2"],
  "scope_assets": ["uuid-3", "uuid-4"]
}
```

#### Get Assessment

```
GET /api/v1/compliance/assessments/{id}
```

**Response:**
```json
{
  "id": "uuid",
  "framework_id": "uuid",
  "framework_name": "SOC 2 Type II",
  "name": "Q4 2025 SOC2 Assessment",
  "status": "completed",
  "started_at": "2025-01-01T10:00:00Z",
  "completed_at": "2025-01-01T10:30:00Z",
  "total_controls": 50,
  "passed_controls": 45,
  "failed_controls": 3,
  "not_applicable": 2,
  "score": 90.0
}
```

#### Get Assessment Results

```
GET /api/v1/compliance/assessments/{id}/results
```

#### Approve Assessment

```
POST /api/v1/compliance/assessments/{id}/approve
```

#### Reject Assessment

```
POST /api/v1/compliance/assessments/{id}/reject
```

---

### Evidence

#### List Evidence

```
GET /api/v1/compliance/evidence?control_id={controlId}
```

#### Create Evidence

```
POST /api/v1/compliance/evidence
```

**Request (JSON):**
```json
{
  "control_id": "uuid",
  "title": "Firewall Configuration Export",
  "evidence_type": "config",
  "storage_type": "inline",
  "content": "{...}"
}
```

**Request (File Upload):**
```
Content-Type: multipart/form-data

control_id: uuid
title: Screenshot
evidence_type: screenshot
file: <binary>
```

#### Get Evidence

```
GET /api/v1/compliance/evidence/{id}
```

#### Delete Evidence

```
DELETE /api/v1/compliance/evidence/{id}
```

---

### Exemptions

#### List Exemptions

```
GET /api/v1/compliance/exemptions
```

**Query Parameters:**
- `control_id` - Filter by control
- `status` - Filter by status (active, expired, revoked)
- `expires_within_days` - Filter by expiration

#### Create Exemption

```
POST /api/v1/compliance/exemptions
```

**Request:**
```json
{
  "control_id": "uuid",
  "asset_id": "uuid",
  "reason": "Legacy system incompatible",
  "risk_acceptance": "Accepted by CISO",
  "compensating_controls": "Additional monitoring implemented",
  "expires_at": "2025-06-30T00:00:00Z",
  "review_frequency_days": 30
}
```

#### Update Exemption

```
PUT /api/v1/compliance/exemptions/{id}
```

#### Revoke Exemption

```
DELETE /api/v1/compliance/exemptions/{id}
```

---

### Compliance Summary

```
GET /api/v1/compliance/summary
```

**Response:**
```json
{
  "overall_score": 87.5,
  "frameworks": [
    {
      "framework_id": "uuid",
      "framework_name": "CIS AWS Foundations",
      "score": 92.0,
      "last_assessment": "2025-01-01T00:00:00Z"
    }
  ],
  "controls_by_status": {
    "passed": 450,
    "failed": 35,
    "not_applicable": 15
  },
  "active_exemptions": 5,
  "upcoming_assessments": 2
}
```

---

## AI Orchestrator Endpoints

**Base URL:** `http://localhost:8083/api/v1`

### Execute Task

```
POST /api/v1/ai/execute
```

**Request:**
```json
{
  "prompt": "Analyze drift for production assets and suggest remediation",
  "context": {
    "site_id": "uuid",
    "priority": "high"
  }
}
```

### List Tasks

```
GET /api/v1/ai/tasks
```

### Get Task

```
GET /api/v1/ai/tasks/{id}
```

### Approve Task

```
POST /api/v1/ai/tasks/{id}/approve
```

### Reject Task

```
POST /api/v1/ai/tasks/{id}/reject
```

### List Agents

```
GET /api/v1/ai/agents
```

### List Tools

```
GET /api/v1/ai/tools
```

---

## Error Responses

### Standard Error Format

```json
{
  "error": "Error message",
  "code": "ERROR_CODE",
  "details": {}
}
```

### Common Error Codes

| HTTP Status | Code | Description |
|-------------|------|-------------|
| 400 | `INVALID_REQUEST` | Malformed request |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | Insufficient permissions |
| 404 | `NOT_FOUND` | Resource not found |
| 409 | `CONFLICT` | Resource conflict |
| 422 | `VALIDATION_ERROR` | Validation failed |
| 429 | `RATE_LIMITED` | Rate limit exceeded |
| 500 | `INTERNAL_ERROR` | Server error |

### RBAC-Specific Errors

| Code | Description |
|------|-------------|
| `PERMISSION_DENIED` | User lacks required permission |
| `ROLE_NOT_FOUND` | Role does not exist |
| `QUOTA_EXCEEDED` | Organization quota exceeded |
| `PLAN_LIMIT` | Feature not available in current plan |

---

## Rate Limits

Rate limits are per-organization based on subscription plan:

| Plan | Requests/Hour |
|------|---------------|
| Free | 100 |
| Starter | 1,000 |
| Professional | 10,000 |
| Enterprise | 100,000 |

Rate limit headers:
- `X-RateLimit-Limit` - Max requests per hour
- `X-RateLimit-Remaining` - Remaining requests
- `X-RateLimit-Reset` - Reset timestamp

---

## Pagination

List endpoints support pagination:

**Query Parameters:**
- `limit` - Max items per page (default: 50, max: 100)
- `offset` - Number of items to skip
- `cursor` - Cursor for cursor-based pagination

**Response Headers:**
- `X-Total-Count` - Total number of items
- `Link` - Pagination links (next, prev)
