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
7. [Vulnerability Response Endpoints](#vulnerability-response-endpoints)
8. [SBOM Endpoints](#sbom-endpoints)
9. [FinOps Endpoints](#finops-endpoints)
10. [InSpec Endpoints](#inspec-endpoints)
11. [Certificate Endpoints](#certificate-endpoints)
12. [Error Responses](#error-responses)

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

## Vulnerability Response Endpoints

**Base URL:** `http://localhost:8083/api/v1` (AI Orchestrator)

### CVE Alerts

#### List CVE Alerts

```
GET /api/v1/cve-alerts
```

**Query Parameters:**
- `severity` - Filter by severity (CRITICAL, HIGH, MEDIUM, LOW)
- `status` - Filter by status (new, investigating, acknowledged, remediation_in_progress, resolved)
- `limit` - Number of results (default: 50)
- `offset` - Pagination offset

**Response:**
```json
{
  "alerts": [
    {
      "id": "uuid",
      "cve_id": "CVE-2024-21626",
      "severity": "CRITICAL",
      "cvss_score": 9.8,
      "urgency_score": 92.5,
      "status": "new",
      "affected_packages": 3,
      "affected_images": 12,
      "affected_assets": 45,
      "production_assets": 15,
      "cisa_known_exploit": true,
      "first_detected_at": "2025-01-15T10:30:00Z"
    }
  ],
  "total": 25,
  "limit": 50,
  "offset": 0
}
```

#### Get CVE Alert Summary

```
GET /api/v1/cve-alerts/summary
```

**Response:**
```json
{
  "total_alerts": 25,
  "by_severity": {
    "critical": 5,
    "high": 8,
    "medium": 10,
    "low": 2
  },
  "by_status": {
    "new": 3,
    "investigating": 5,
    "acknowledged": 7,
    "remediation_in_progress": 8,
    "resolved": 2
  },
  "production_at_risk": 45,
  "avg_urgency_score": 65.3
}
```

#### Get CVE Alert Details

```
GET /api/v1/cve-alerts/{alertId}
```

**Response:**
```json
{
  "id": "uuid",
  "cve_id": "CVE-2024-21626",
  "severity": "CRITICAL",
  "cvss_score": 9.8,
  "cvss_vector": "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
  "epss_score": 0.85,
  "urgency_score": 92.5,
  "status": "new",
  "cisa_known_exploit": true,
  "exploit_available": true,
  "description": "Container escape vulnerability in runc",
  "affected_packages": [
    {
      "name": "runc",
      "version": "1.1.0",
      "type": "binary",
      "fixed_version": "1.1.12"
    }
  ],
  "references": [
    {
      "type": "ADVISORY",
      "url": "https://nvd.nist.gov/vuln/detail/CVE-2024-21626"
    }
  ],
  "first_detected_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T14:00:00Z"
}
```

#### Get Blast Radius

```
GET /api/v1/cve-alerts/{alertId}/blast-radius
```

**Response:**
```json
{
  "alert_id": "uuid",
  "cve_id": "CVE-2024-21626",
  "packages": [
    {
      "name": "runc",
      "version": "1.1.0",
      "type": "binary",
      "fixed_version": "1.1.12"
    }
  ],
  "images": [
    {
      "id": "uuid",
      "name": "base-ubuntu-24.04",
      "version": "v1.2.3",
      "lineage_depth": 0,
      "is_direct": true,
      "children_count": 8
    }
  ],
  "assets": [
    {
      "id": "uuid",
      "name": "web-server-prod-1",
      "platform": "aws",
      "region": "us-east-1",
      "environment": "production",
      "is_production": true
    }
  ],
  "summary": {
    "total_packages": 3,
    "total_images": 12,
    "direct_images": 3,
    "inherited_images": 9,
    "total_assets": 45,
    "production_assets": 15,
    "non_production_assets": 30
  }
}
```

#### Investigate Alert

```
POST /api/v1/cve-alerts/{alertId}/investigate
```

**Request:**
```json
{
  "assignee_id": "user-uuid",
  "notes": "Starting investigation of container escape vulnerability"
}
```

#### Acknowledge Alert

```
POST /api/v1/cve-alerts/{alertId}/acknowledge
```

**Request:**
```json
{
  "notes": "Confirmed vulnerability affects production systems",
  "planned_remediation_date": "2025-01-20T00:00:00Z"
}
```

#### Resolve Alert

```
POST /api/v1/cve-alerts/{alertId}/resolve
```

**Request:**
```json
{
  "resolution": "patched",
  "patch_campaign_id": "campaign-uuid",
  "notes": "All affected assets patched via campaign"
}
```

---

### Patch Campaigns

#### List Patch Campaigns

```
GET /api/v1/patch-campaigns
```

**Query Parameters:**
- `status` - Filter by status (pending_approval, approved, in_progress, paused, completed, failed, rolled_back)
- `cve_alert_id` - Filter by associated CVE alert
- `limit` - Number of results (default: 50)
- `offset` - Pagination offset

**Response:**
```json
{
  "campaigns": [
    {
      "id": "uuid",
      "name": "CVE-2024-21626 Critical Patch",
      "cve_alert_id": "uuid",
      "status": "in_progress",
      "strategy": "canary",
      "total_assets": 45,
      "completed_assets": 12,
      "failed_assets": 0,
      "progress_percent": 26.7,
      "created_at": "2025-01-16T08:00:00Z",
      "started_at": "2025-01-16T10:00:00Z"
    }
  ],
  "total": 10,
  "limit": 50,
  "offset": 0
}
```

#### Get Campaign Summary

```
GET /api/v1/patch-campaigns/summary
```

**Response:**
```json
{
  "total_campaigns": 15,
  "active_campaigns": 3,
  "by_status": {
    "pending_approval": 2,
    "in_progress": 3,
    "completed": 8,
    "failed": 1,
    "rolled_back": 1
  },
  "assets_patched_this_month": 450,
  "avg_completion_time_hours": 4.5
}
```

#### Create Patch Campaign

```
POST /api/v1/patch-campaigns
```

**Request:**
```json
{
  "name": "CVE-2024-21626 Critical Patch",
  "cve_alert_id": "uuid",
  "strategy": "canary",
  "target_assets": ["asset-uuid-1", "asset-uuid-2"],
  "phases": [
    {
      "name": "canary",
      "percentage": 5,
      "wait_minutes": 30
    },
    {
      "name": "wave_1",
      "percentage": 25,
      "wait_minutes": 60
    },
    {
      "name": "full_rollout",
      "percentage": 100,
      "wait_minutes": 0
    }
  ],
  "health_checks": {
    "error_rate_threshold": 0.05,
    "latency_p99_threshold_ms": 500
  },
  "auto_rollback": true
}
```

#### Get Campaign Details

```
GET /api/v1/patch-campaigns/{campaignId}
```

#### Get Campaign Phases

```
GET /api/v1/patch-campaigns/{campaignId}/phases
```

**Response:**
```json
{
  "phases": [
    {
      "id": "uuid",
      "name": "canary",
      "status": "completed",
      "target_count": 3,
      "completed_count": 3,
      "failed_count": 0,
      "started_at": "2025-01-16T10:00:00Z",
      "completed_at": "2025-01-16T10:25:00Z"
    },
    {
      "id": "uuid",
      "name": "wave_1",
      "status": "in_progress",
      "target_count": 11,
      "completed_count": 5,
      "failed_count": 0,
      "started_at": "2025-01-16T10:55:00Z"
    }
  ]
}
```

#### Get Campaign Progress

```
GET /api/v1/patch-campaigns/{campaignId}/progress
```

**Response:**
```json
{
  "campaign_id": "uuid",
  "status": "in_progress",
  "current_phase": "wave_1",
  "total_assets": 45,
  "completed_assets": 8,
  "failed_assets": 0,
  "pending_assets": 37,
  "progress_percent": 17.8,
  "health_status": "healthy",
  "estimated_completion": "2025-01-16T14:30:00Z"
}
```

#### Approve Campaign

```
POST /api/v1/patch-campaigns/{campaignId}/approve
```

**Request:**
```json
{
  "approver_notes": "Approved for production rollout"
}
```

#### Start Campaign

```
POST /api/v1/patch-campaigns/{campaignId}/start
```

#### Pause Campaign

```
POST /api/v1/patch-campaigns/{campaignId}/pause
```

**Request:**
```json
{
  "reason": "Investigating elevated error rates"
}
```

#### Resume Campaign

```
POST /api/v1/patch-campaigns/{campaignId}/resume
```

#### Cancel Campaign

```
POST /api/v1/patch-campaigns/{campaignId}/cancel
```

**Request:**
```json
{
  "reason": "New patch version available"
}
```

#### Trigger Rollback

```
POST /api/v1/patch-campaigns/{campaignId}/rollback
```

**Request:**
```json
{
  "reason": "Health check failures detected",
  "rollback_scope": "all"
}
```

**Response:**
```json
{
  "rollback_id": "uuid",
  "campaign_id": "uuid",
  "status": "in_progress",
  "assets_to_rollback": 8,
  "started_at": "2025-01-16T12:00:00Z"
}
```

---

## SBOM Endpoints

**Base URL:** `http://localhost:8080/api/v1`

### Get Image SBOM

```
GET /api/v1/sbom/images/{id}
```

**Response:**
```json
{
  "image_id": "uuid",
  "format": "spdx",
  "spdx_version": "SPDX-2.3",
  "created_at": "2025-01-01T00:00:00Z",
  "components": [
    {
      "name": "openssl",
      "version": "3.0.2",
      "purl": "pkg:deb/debian/openssl@3.0.2",
      "licenses": ["Apache-2.0"],
      "supplier": "Debian"
    }
  ],
  "relationships": [],
  "vulnerabilities_count": 5
}
```

### Generate SBOM

```
POST /api/v1/sbom/images/{id}/generate
```

**Request:**
```json
{
  "format": "spdx",
  "include_vulnerabilities": true
}
```

### List Components

```
GET /api/v1/sbom/components
```

**Query Parameters:**
- `image_id` - Filter by image
- `license` - Filter by license
- `vulnerable` - Filter vulnerable components

### Query Vulnerabilities

```
GET /api/v1/sbom/vulnerabilities
```

**Query Parameters:**
- `severity` - Filter by severity (critical, high, medium, low)
- `cve_id` - Search by CVE ID
- `component` - Filter by component name

### Import SBOM

```
POST /api/v1/sbom/import
```

**Request:**
```json
{
  "image_id": "uuid",
  "format": "cyclonedx",
  "content": "{...sbom json...}"
}
```

### Export SBOM

```
GET /api/v1/sbom/export/{format}
```

**Path Parameters:**
- `format` - Export format: `spdx` or `cyclonedx`

**Query Parameters:**
- `image_id` - Image to export (required)

### License Summary

```
GET /api/v1/sbom/licenses
```

**Response:**
```json
{
  "total_components": 150,
  "licenses": [
    {"license": "MIT", "count": 45},
    {"license": "Apache-2.0", "count": 38},
    {"license": "GPL-3.0", "count": 12}
  ],
  "copyleft_count": 15,
  "permissive_count": 135
}
```

---

## FinOps Endpoints

**Base URL:** `http://localhost:8080/api/v1`

### Get Cost Data

```
GET /api/v1/finops/costs
```

**Query Parameters:**
- `start_date` - Start date (YYYY-MM-DD)
- `end_date` - End date (YYYY-MM-DD)
- `cloud` - Filter by cloud (aws, azure, gcp)
- `granularity` - daily, weekly, monthly

**Response:**
```json
{
  "total_cost": 15420.50,
  "currency": "USD",
  "period": {
    "start": "2025-01-01",
    "end": "2025-01-31"
  },
  "by_cloud": {
    "aws": 8500.00,
    "azure": 4200.50,
    "gcp": 2720.00
  },
  "daily_costs": [...]
}
```

### Cost by Service

```
GET /api/v1/finops/costs/by-service
```

**Response:**
```json
{
  "services": [
    {"service": "EC2", "cost": 4500.00, "cloud": "aws"},
    {"service": "RDS", "cost": 2100.00, "cloud": "aws"},
    {"service": "Virtual Machines", "cost": 2800.00, "cloud": "azure"}
  ]
}
```

### Cost by Tag

```
GET /api/v1/finops/costs/by-tag
```

**Query Parameters:**
- `tag_key` - Tag key to group by (e.g., "environment", "team")

### List Budgets

```
GET /api/v1/finops/budgets
```

### Create Budget

```
POST /api/v1/finops/budgets
```

**Request:**
```json
{
  "name": "Production Infrastructure",
  "amount": 50000.00,
  "period": "monthly",
  "alert_thresholds": [50, 80, 100],
  "filters": {
    "tags": {"environment": "production"}
  }
}
```

### Optimization Recommendations

```
GET /api/v1/finops/recommendations
```

**Response:**
```json
{
  "recommendations": [
    {
      "type": "right_sizing",
      "resource": "i-0123456789",
      "current_cost": 500.00,
      "projected_cost": 250.00,
      "savings": 250.00,
      "recommendation": "Downsize from m5.xlarge to m5.large"
    },
    {
      "type": "reserved_instance",
      "resource": "RDS",
      "current_cost": 2100.00,
      "projected_cost": 1400.00,
      "savings": 700.00,
      "recommendation": "Purchase 1-year reserved instance"
    }
  ],
  "total_potential_savings": 950.00
}
```

### Cost Forecast

```
GET /api/v1/finops/forecast
```

**Query Parameters:**
- `months` - Forecast period (1-12)

---

## InSpec Endpoints

**Base URL:** `http://localhost:8080/api/v1`

### List Profiles

```
GET /api/v1/inspec/profiles
```

**Response:**
```json
[
  {
    "id": "cis-aws-foundations",
    "name": "CIS AWS Foundations Benchmark",
    "version": "1.5.0",
    "controls_count": 58,
    "supported_platforms": ["aws"]
  },
  {
    "id": "cis-linux",
    "name": "CIS Linux Benchmark",
    "version": "2.0.0",
    "controls_count": 263,
    "supported_platforms": ["ubuntu", "rhel", "centos"]
  }
]
```

### Get Profile Details

```
GET /api/v1/inspec/profiles/{id}
```

### Trigger Scan

```
POST /api/v1/inspec/scans
```

**Request:**
```json
{
  "profile_id": "cis-aws-foundations",
  "targets": [
    {"type": "aws_account", "id": "123456789012"}
  ],
  "controls": ["1.1", "1.2", "1.3"],
  "collect_evidence": true
}
```

**Response:**
```json
{
  "scan_id": "uuid",
  "status": "pending",
  "workflow_id": "temporal-workflow-id"
}
```

### List Scans

```
GET /api/v1/inspec/scans
```

**Query Parameters:**
- `profile_id` - Filter by profile
- `status` - Filter by status (pending, running, completed, failed)
- `limit` - Max results

### Get Scan Details

```
GET /api/v1/inspec/scans/{id}
```

**Response:**
```json
{
  "id": "uuid",
  "profile_id": "cis-aws-foundations",
  "status": "completed",
  "started_at": "2025-01-01T10:00:00Z",
  "completed_at": "2025-01-01T10:15:00Z",
  "summary": {
    "total": 58,
    "passed": 52,
    "failed": 4,
    "skipped": 2
  }
}
```

### Get Scan Results

```
GET /api/v1/inspec/scans/{id}/results
```

**Response:**
```json
{
  "results": [
    {
      "control_id": "1.1",
      "title": "Avoid the use of root account",
      "status": "passed",
      "impact": 1.0,
      "message": "Root account has MFA enabled"
    },
    {
      "control_id": "1.4",
      "title": "Ensure access keys are rotated",
      "status": "failed",
      "impact": 0.7,
      "message": "3 access keys older than 90 days"
    }
  ]
}
```

### Get Scan Evidence

```
GET /api/v1/inspec/scans/{id}/evidence
```

**Response:**
```json
{
  "evidence": [
    {
      "control_id": "1.1",
      "type": "api_response",
      "collected_at": "2025-01-01T10:05:00Z",
      "data": {...}
    }
  ]
}
```

### Create Scan Schedule

```
POST /api/v1/inspec/schedules
```

**Request:**
```json
{
  "profile_id": "cis-aws-foundations",
  "cron": "0 2 * * *",
  "targets": [...],
  "enabled": true
}
```

### List Schedules

```
GET /api/v1/inspec/schedules
```

### Delete Schedule

```
DELETE /api/v1/inspec/schedules/{id}
```

---

## Certificate Endpoints

**Base URL:** `http://localhost:8080/api/v1`

Certificate lifecycle management endpoints for tracking SSL/TLS certificates across multi-cloud infrastructure.

### List Certificates

```
GET /api/v1/certificates
```

**Query Parameters:**
- `platform` - Filter by platform (aws, azure, gcp, kubernetes, vsphere)
- `status` - Filter by status (valid, expiring_soon, expired, revoked)
- `days_until_expiry` - Filter certificates expiring within N days
- `search` - Search by common name or issuer
- `limit` - Max results (default: 50)
- `offset` - Pagination offset

**Response:**
```json
{
  "certificates": [
    {
      "id": "uuid",
      "common_name": "api.example.com",
      "issuer": "DigiCert Inc",
      "platform": "aws",
      "source": "acm",
      "serial_number": "0A:1B:2C:3D...",
      "status": "valid",
      "not_before": "2024-01-01T00:00:00Z",
      "not_after": "2025-01-01T00:00:00Z",
      "days_until_expiry": 180,
      "auto_renewal_eligible": true,
      "key_algorithm": "RSA",
      "key_size": 2048,
      "signature_algorithm": "SHA256withRSA",
      "san_entries": ["api.example.com", "*.api.example.com"],
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-06-15T10:30:00Z"
    }
  ],
  "total": 156,
  "limit": 50,
  "offset": 0
}
```

### Get Certificate

```
GET /api/v1/certificates/{id}
```

**Response:**
```json
{
  "id": "uuid",
  "common_name": "api.example.com",
  "issuer": "DigiCert Inc",
  "platform": "aws",
  "source": "acm",
  "serial_number": "0A:1B:2C:3D...",
  "status": "valid",
  "not_before": "2024-01-01T00:00:00Z",
  "not_after": "2025-01-01T00:00:00Z",
  "days_until_expiry": 180,
  "auto_renewal_eligible": true,
  "key_algorithm": "RSA",
  "key_size": 2048,
  "signature_algorithm": "SHA256withRSA",
  "san_entries": ["api.example.com", "*.api.example.com"],
  "fingerprint_sha256": "AB:CD:EF:...",
  "raw_pem": "-----BEGIN CERTIFICATE-----\n...",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-06-15T10:30:00Z"
}
```

### Get Certificate Summary

```
GET /api/v1/certificates/summary
```

Returns aggregate statistics about certificates.

**Response:**
```json
{
  "total_certificates": 156,
  "by_status": {
    "valid": 140,
    "expiring_soon": 12,
    "expired": 4,
    "revoked": 0
  },
  "by_platform": {
    "aws": 60,
    "azure": 45,
    "gcp": 30,
    "kubernetes": 15,
    "vsphere": 6
  },
  "expiring_7_days": 3,
  "expiring_30_days": 12,
  "expiring_90_days": 25,
  "auto_renewal_eligible": 95,
  "manual_renewal_required": 61
}
```

### Get Certificate Usage

```
GET /api/v1/certificates/{id}/usage
```

Returns where a certificate is deployed and used.

**Response:**
```json
{
  "certificate_id": "uuid",
  "usages": [
    {
      "id": "uuid",
      "resource_type": "load_balancer",
      "resource_id": "arn:aws:elasticloadbalancing:...",
      "resource_name": "prod-api-alb",
      "platform": "aws",
      "region": "us-east-1",
      "discovered_at": "2024-06-15T10:00:00Z"
    },
    {
      "id": "uuid",
      "resource_type": "kubernetes_ingress",
      "resource_id": "ingress/api-ingress",
      "resource_name": "api-ingress",
      "platform": "kubernetes",
      "namespace": "production",
      "discovered_at": "2024-06-15T10:00:00Z"
    }
  ],
  "total_usages": 5,
  "blast_radius": "high"
}
```

### List Rotations

```
GET /api/v1/certificates/rotations
```

**Query Parameters:**
- `certificate_id` - Filter by certificate
- `status` - Filter by status (pending, in_progress, completed, failed, rolled_back)
- `initiated_by` - Filter by initiator (ai_agent, manual, scheduled)
- `limit` - Max results
- `offset` - Pagination offset

**Response:**
```json
{
  "rotations": [
    {
      "id": "uuid",
      "certificate_id": "uuid",
      "rotation_type": "renewal",
      "status": "completed",
      "initiated_by": "ai_agent",
      "affected_usages": 5,
      "successful_updates": 5,
      "started_at": "2024-06-15T10:00:00Z",
      "completed_at": "2024-06-15T10:15:00Z",
      "plan": {
        "steps": [
          {"action": "generate_new_cert", "status": "completed"},
          {"action": "update_load_balancer", "status": "completed"},
          {"action": "update_ingress", "status": "completed"},
          {"action": "verify_tls", "status": "completed"},
          {"action": "revoke_old_cert", "status": "completed"}
        ]
      }
    }
  ],
  "total": 45,
  "limit": 50,
  "offset": 0
}
```

### Get Rotation Details

```
GET /api/v1/certificates/rotations/{id}
```

**Response:**
```json
{
  "id": "uuid",
  "certificate_id": "uuid",
  "rotation_type": "renewal",
  "status": "completed",
  "initiated_by": "ai_agent",
  "affected_usages": 5,
  "successful_updates": 5,
  "started_at": "2024-06-15T10:00:00Z",
  "completed_at": "2024-06-15T10:15:00Z",
  "old_certificate": {
    "serial_number": "0A:1B:2C...",
    "not_after": "2024-07-01T00:00:00Z"
  },
  "new_certificate": {
    "serial_number": "1D:2E:3F...",
    "not_after": "2025-07-01T00:00:00Z"
  },
  "plan": {
    "steps": [...]
  },
  "logs": [
    {
      "timestamp": "2024-06-15T10:00:00Z",
      "message": "Started certificate rotation",
      "level": "info"
    }
  ]
}
```

### List Alerts

```
GET /api/v1/certificates/alerts
```

**Query Parameters:**
- `severity` - Filter by severity (critical, high, medium, low)
- `status` - Filter by status (open, acknowledged, resolved)
- `certificate_id` - Filter by certificate
- `limit` - Max results
- `offset` - Pagination offset

**Response:**
```json
{
  "alerts": [
    {
      "id": "uuid",
      "certificate_id": "uuid",
      "severity": "critical",
      "status": "open",
      "title": "Certificate Expired",
      "message": "Certificate for api.example.com expired 2 days ago",
      "created_at": "2024-06-15T00:00:00Z",
      "acknowledged_at": null,
      "resolved_at": null
    }
  ],
  "total": 8,
  "limit": 50,
  "offset": 0
}
```

### Acknowledge Alert

```
POST /api/v1/certificates/alerts/{id}/acknowledge
```

**Response:**
```json
{
  "id": "uuid",
  "status": "acknowledged",
  "acknowledged_at": "2024-06-15T12:00:00Z",
  "acknowledged_by": "user-uuid"
}
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
