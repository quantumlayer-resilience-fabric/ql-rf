# Enterprise Features Deployment Guide

This guide covers deploying and configuring the enterprise features introduced in Phase 4.5.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Database Migrations](#database-migrations)
3. [RBAC Configuration](#rbac-configuration)
4. [Multi-Tenancy Setup](#multi-tenancy-setup)
5. [Compliance Frameworks](#compliance-frameworks)
6. [Secrets Management](#secrets-management)
7. [Environment Variables](#environment-variables)
8. [Verification](#verification)

---

## Prerequisites

### Required Components
- PostgreSQL 16+
- Redis 7+
- Go 1.24+
- Docker + Docker Compose (for local development)

### Required Migrations
Ensure migrations 000008-000012 are applied:
```bash
make migrate-up
# OR
~/go/bin/migrate -path migrations -database "$RF_DATABASE_URL" up
```

---

## Database Migrations

### Migration Overview

| Migration | Purpose |
|-----------|---------|
| 000008 | RBAC tables (roles, permissions, user_roles, teams) |
| 000009 | Multi-tenancy (subscription_plans, organization_quotas, organization_usage) |
| 000010 | Compliance frameworks (compliance_frameworks, compliance_controls) |
| 000011 | Audit trail (audit_logs, llm_usage) |
| 000012 | System compliance frameworks (7 pre-populated frameworks) |

### Applying Migrations

```bash
# Set database URL
export RF_DATABASE_URL="postgres://postgres:postgres@localhost:5432/qlrf?sslmode=disable"

# Run all migrations
make migrate-up

# Check migration status
psql $RF_DATABASE_URL -c "SELECT * FROM schema_migrations ORDER BY version;"

# Verify tables exist
psql $RF_DATABASE_URL -c "\dt" | grep -E "(roles|permissions|subscription_plans|compliance)"
```

### Rollback (if needed)

```bash
# Rollback last migration
make migrate-down

# Rollback specific version
~/go/bin/migrate -path migrations -database "$RF_DATABASE_URL" down 1
```

---

## RBAC Configuration

### System Roles

The following roles are created by migration 000008:

| Role | Level | Description |
|------|-------|-------------|
| `org_owner` | 100 | Full organization access including billing |
| `org_admin` | 90 | Admin without billing access |
| `infra_admin` | 80 | Infrastructure management |
| `security_admin` | 80 | Security and compliance management |
| `dr_admin` | 70 | Disaster recovery operations |
| `operator` | 50 | Day-to-day operations |
| `analyst` | 30 | Read access with analysis capabilities |
| `viewer` | 10 | Read-only access |

### Permission Types

Resources and their available actions:

| Resource | Actions |
|----------|---------|
| `assets` | read, write, delete |
| `images` | read, write, delete, promote |
| `drift` | read, remediate |
| `compliance` | read, assess, remediate |
| `dr` | read, drill, failover |
| `settings` | read, write |
| `users` | read, write, delete |
| `billing` | read, write |

### Assigning Roles via API

```bash
# Assign a role to a user
curl -X POST http://localhost:8080/api/v1/rbac/users/{userId}/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role_id": "uuid-of-role",
    "scope": {
      "site_ids": ["site-uuid"]
    }
  }'
```

### Creating Custom Roles

```bash
# Create a custom role
curl -X POST http://localhost:8080/api/v1/rbac/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "site_operator",
    "display_name": "Site Operator",
    "description": "Operator scoped to specific sites",
    "level": 40,
    "permissions": [
      {"resource": "assets", "action": "read"},
      {"resource": "drift", "action": "read"},
      {"resource": "drift", "action": "remediate"}
    ]
  }'
```

---

## Multi-Tenancy Setup

### Subscription Plans

Pre-configured plans (from migration 000009):

| Plan | Price | Max Assets | Max Images | Max Sites | API Rate/hr |
|------|-------|-----------|------------|-----------|-------------|
| Free | $0 | 50 | 5 | 1 | 100 |
| Starter | $99 | 500 | 25 | 5 | 1,000 |
| Professional | $499 | 5,000 | 100 | 25 | 10,000 |
| Enterprise | Custom | Unlimited | Unlimited | Unlimited | 100,000 |

### Setting Organization Quotas

```bash
# View current quota
curl http://localhost:8080/api/v1/organization/quota \
  -H "Authorization: Bearer $TOKEN"

# Update quota (admin only)
curl -X PUT http://localhost:8080/api/v1/organization/quota \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "max_assets": 1000,
    "max_images": 50,
    "max_sites": 10,
    "max_users": 25,
    "api_rate_limit_per_hour": 5000
  }'
```

### Monitoring Usage

```bash
# Get usage statistics
curl http://localhost:8080/api/v1/organization/usage \
  -H "Authorization: Bearer $TOKEN"

# Response includes:
# - asset_count, image_count, site_count, user_count
# - api_requests_this_hour, llm_tokens_this_month
# - usage percentages vs quota
```

---

## Compliance Frameworks

### Pre-loaded Frameworks

Migration 000012 loads these frameworks:

| Framework | Version | Controls |
|-----------|---------|----------|
| CIS AWS Foundations | v1.5.0 | 13+ |
| CIS Azure Foundations | v2.0.0 | 15+ |
| CIS GCP Foundations | v1.3.0 | 12+ |
| CIS Kubernetes Benchmark | v1.7.0 | 20+ |
| SOC 2 Type II | 2024 | 50+ |
| NIST Cybersecurity Framework | v1.1 | 100+ |
| NIST 800-53 | Rev 5 | 200+ |

### Listing Frameworks

```bash
curl http://localhost:8080/api/v1/compliance/frameworks \
  -H "Authorization: Bearer $TOKEN"
```

### Creating Compliance Assessments

```bash
# Create a new assessment
curl -X POST http://localhost:8080/api/v1/compliance/assessments \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "framework_id": "framework-uuid",
    "name": "Q4 2025 SOC2 Assessment",
    "assessment_type": "automated",
    "scope_sites": ["site-uuid-1", "site-uuid-2"]
  }'
```

---

## Secrets Management

### Supported Backends

| Backend | Use Case | Configuration |
|---------|----------|---------------|
| `memory` | Development/testing | Default |
| `env` | Simple deployments | Environment variables |
| `vault` | Production | HashiCorp Vault |

### Configuration

```bash
# Memory backend (default)
export RF_SECRETS_BACKEND=memory

# Environment backend
export RF_SECRETS_BACKEND=env

# Vault backend
export RF_SECRETS_BACKEND=vault
export RF_VAULT_ADDR=https://vault.example.com:8200
export RF_VAULT_TOKEN=hvs.xxxxx
export RF_VAULT_MOUNT_PATH=secret
export RF_VAULT_PREFIX=ql-rf/
```

### Using Secrets in Code

```go
import "github.com/quantumlayerhq/ql-rf/pkg/secrets"

cfg := &secrets.Config{
    Backend:  secrets.BackendVault,
    CacheTTL: 5 * time.Minute,
}
mgr, _ := secrets.NewManager(cfg)

// Get a secret
secret, _ := mgr.Get(ctx, "aws/credentials")

// Get with default
value := mgr.GetOrDefault(ctx, "feature-flag", "false")

// Rotate a secret
mgr.RotateSecret(ctx, "db/password", func() (string, error) {
    return generateNewPassword(), nil
})
```

---

## Environment Variables

### Required Variables

```bash
# Database
RF_DATABASE_URL=postgres://user:pass@host:5432/qlrf?sslmode=disable

# Authentication
RF_CLERK_SECRET_KEY=sk_live_xxxxx

# LLM (choose one or more)
RF_LLM_PROVIDER=anthropic
RF_LLM_API_KEY=sk-ant-xxxxx
RF_LLM_MODEL=claude-3-5-sonnet-20241022
```

### Optional Enterprise Variables

```bash
# Secrets Management
RF_SECRETS_BACKEND=vault
RF_VAULT_ADDR=https://vault.example.com:8200
RF_VAULT_TOKEN=hvs.xxxxx

# Observability
RF_OTEL_ENABLED=true
RF_OTEL_ENDPOINT=http://jaeger:4317

# Audit Logging
RF_AUDIT_LOG_RETENTION_DAYS=90
RF_AUDIT_LOG_LEVEL=info

# Rate Limiting
RF_RATE_LIMIT_ENABLED=true
RF_RATE_LIMIT_REQUESTS_PER_SECOND=100
```

---

## Verification

### Health Checks

```bash
# API health
curl http://localhost:8080/healthz

# Orchestrator health
curl http://localhost:8083/healthz

# Database connectivity
curl http://localhost:8080/readyz
```

### Verify Enterprise Features

```bash
# 1. Check RBAC roles exist
psql $RF_DATABASE_URL -c "SELECT name, level FROM roles ORDER BY level DESC;"

# 2. Check subscription plans
psql $RF_DATABASE_URL -c "SELECT name, default_max_assets FROM subscription_plans;"

# 3. Check compliance frameworks
psql $RF_DATABASE_URL -c "SELECT name, version FROM compliance_frameworks;"

# 4. Test RBAC API
curl http://localhost:8080/api/v1/rbac/roles \
  -H "Authorization: Bearer $TOKEN"

# 5. Test compliance API
curl http://localhost:8080/api/v1/compliance/frameworks \
  -H "Authorization: Bearer $TOKEN"
```

### Run Integration Tests

```bash
# Run all enterprise feature tests
RF_DATABASE_URL="postgres://postgres:postgres@localhost:5432/qlrf?sslmode=disable" \
  go test -v ./tests/integration/...

# Expected output: 15 PASS, 1 SKIP
```

---

## Troubleshooting

### Common Issues

**Migration fails with "relation already exists"**
```bash
# Check dirty flag
psql $RF_DATABASE_URL -c "SELECT * FROM schema_migrations;"
# If dirty=true, reset and retry
psql $RF_DATABASE_URL -c "UPDATE schema_migrations SET dirty=false;"
```

**RBAC permission denied**
```bash
# Check user's roles
curl http://localhost:8080/api/v1/rbac/users/{userId}/roles \
  -H "Authorization: Bearer $TOKEN"

# Check specific permission
curl "http://localhost:8080/api/v1/rbac/check?resource=assets&action=write" \
  -H "Authorization: Bearer $TOKEN"
```

**Quota exceeded**
```bash
# Check current usage
curl http://localhost:8080/api/v1/organization/usage \
  -H "Authorization: Bearer $TOKEN"

# Upgrade plan or request quota increase
```

---

## Support

For issues with enterprise features:
1. Check logs: `docker logs qlrf-api`
2. Review audit trail: `SELECT * FROM audit_logs ORDER BY created_at DESC LIMIT 20;`
3. File issue: https://github.com/quantumlayer-resilience-fabric/ql-rf/issues
