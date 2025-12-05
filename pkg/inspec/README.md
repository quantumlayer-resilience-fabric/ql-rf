# InSpec Integration

Comprehensive InSpec integration for automated compliance assessment in QuantumLayer Resilience Fabric (QL-RF).

## Overview

The InSpec integration enables automated compliance assessment by executing InSpec profiles against infrastructure assets and mapping results to compliance framework controls (CIS, SOC 2, NIST, etc.).

## Features

- **Profile Management**: Create and manage InSpec profiles mapped to compliance frameworks
- **Automated Execution**: Run InSpec profiles against assets (VMs, containers, cloud accounts)
- **Result Tracking**: Store and analyze control execution results
- **Control Mapping**: Map InSpec controls to compliance framework controls
- **Temporal Workflows**: Durable, reliable execution via Temporal
- **Batch Execution**: Run profiles across multiple assets in parallel

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      API Layer                              │
│  /api/v1/inspec/profiles  - List/Create profiles           │
│  /api/v1/inspec/run       - Execute profile                │
│  /api/v1/inspec/runs      - List/View runs                 │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────┼───────────────────────────────────────┐
│                  InSpec Service                             │
│  • Profile Management                                       │
│  • Run Orchestration                                        │
│  • Result Storage                                           │
│  • Control Mapping                                          │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────┼───────────────────────────────────────┐
│              Temporal Workflows                             │
│  • InSpecExecutionWorkflow                                  │
│  • BatchInSpecExecutionWorkflow                             │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────┼───────────────────────────────────────┐
│              Temporal Activities                            │
│  • PrepareInSpecExecution                                   │
│  • ExecuteInSpecProfile                                     │
│  • ParseInSpecResults                                       │
│  • MapInSpecToComplianceControls                            │
└─────────────────────────────────────────────────────────────┘
```

## Database Schema

### inspec_profiles
Stores InSpec profile metadata and framework associations.

```sql
CREATE TABLE inspec_profiles (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    title VARCHAR(500) NOT NULL,
    framework_id UUID REFERENCES compliance_frameworks(id),
    profile_url VARCHAR(1024),
    platforms VARCHAR(100)[],
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
```

### inspec_runs
Tracks InSpec profile executions.

```sql
CREATE TABLE inspec_runs (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES organizations(id),
    asset_id UUID REFERENCES assets(id),
    profile_id UUID REFERENCES inspec_profiles(id),
    status VARCHAR(50), -- pending, running, completed, failed
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration INTEGER,
    total_tests INTEGER,
    passed_tests INTEGER,
    failed_tests INTEGER,
    skipped_tests INTEGER,
    error_message TEXT,
    created_at TIMESTAMPTZ
);
```

### inspec_results
Stores individual control results from runs.

```sql
CREATE TABLE inspec_results (
    id UUID PRIMARY KEY,
    run_id UUID REFERENCES inspec_runs(id),
    control_id VARCHAR(255),
    control_title VARCHAR(500),
    status VARCHAR(50), -- passed, failed, skipped, error
    message TEXT,
    resource VARCHAR(500),
    run_time DECIMAL(10, 6),
    created_at TIMESTAMPTZ
);
```

### inspec_control_mappings
Maps InSpec controls to compliance framework controls.

```sql
CREATE TABLE inspec_control_mappings (
    id UUID PRIMARY KEY,
    inspec_control_id VARCHAR(255),
    compliance_control_id UUID REFERENCES compliance_controls(id),
    profile_id UUID REFERENCES inspec_profiles(id),
    mapping_confidence DECIMAL(3, 2),
    notes TEXT,
    created_at TIMESTAMPTZ
);
```

## Usage

### 1. List Available Profiles

```bash
curl -X GET http://localhost:8080/api/v1/inspec/profiles \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "profiles": [
    {
      "profile_id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "cis-linux-level-1",
      "title": "CIS Linux Benchmark Level 1",
      "version": "1.1.0",
      "framework": "CIS Linux",
      "framework_id": "...",
      "platforms": ["linux", "ubuntu", "debian"],
      "control_count": 42
    }
  ]
}
```

### 2. Run InSpec Profile

```bash
curl -X POST http://localhost:8080/api/v1/inspec/run \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "profile_id": "550e8400-e29b-41d4-a716-446655440000",
    "asset_id": "660e8400-e29b-41d4-a716-446655440001"
  }'
```

Response:
```json
{
  "id": "770e8400-e29b-41d4-a716-446655440002",
  "org_id": "...",
  "asset_id": "660e8400-e29b-41d4-a716-446655440001",
  "profile_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "created_at": "2025-12-05T10:00:00Z"
}
```

### 3. Get Run Results

```bash
curl -X GET http://localhost:8080/api/v1/inspec/runs/{runId}/results \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "run": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "status": "completed",
    "started_at": "2025-12-05T10:00:01Z",
    "completed_at": "2025-12-05T10:05:23Z",
    "duration": 322,
    "total_tests": 42,
    "passed_tests": 38,
    "failed_tests": 4,
    "skipped_tests": 0
  },
  "results": [
    {
      "id": "...",
      "control_id": "cis-1.1.1.1",
      "control_title": "Ensure cramfs kernel module is not available",
      "status": "passed",
      "run_time": 0.523,
      "created_at": "2025-12-05T10:05:22Z"
    },
    {
      "id": "...",
      "control_id": "cis-5.2.10",
      "control_title": "Ensure SSH root login is disabled",
      "status": "failed",
      "message": "PermitRootLogin is set to 'yes' but should be 'no'",
      "resource": "/etc/ssh/sshd_config",
      "run_time": 0.312,
      "created_at": "2025-12-05T10:05:23Z"
    }
  ]
}
```

### 4. List Runs

```bash
curl -X GET http://localhost:8080/api/v1/inspec/runs?limit=20&offset=0 \
  -H "Authorization: Bearer $TOKEN"
```

## Profile Mappings

Pre-built profile mappings are available for common frameworks:

### CIS Linux Benchmark
- **Profile**: `cis-linux-level-1` / `cis-linux-level-2`
- **Framework**: CIS Linux Benchmark
- **Platforms**: linux, ubuntu, debian, redhat, centos, amazon-linux
- **Controls**: 40+ automated checks

### CIS AWS Foundations
- **Profile**: `cis-aws-foundations-benchmark`
- **Framework**: CIS AWS Foundations Benchmark
- **Platforms**: aws
- **Controls**: 50+ AWS security checks

### SOC 2 Type II
- **Profile**: `soc2-type-ii-baseline`
- **Framework**: SOC 2 Type II
- **Platforms**: linux, windows, aws, azure, gcp
- **Controls**: 30+ infrastructure security checks

## Temporal Workflows

### InSpecExecutionWorkflow

Orchestrates the execution of an InSpec profile:

1. **Prepare Environment**: Create temp directory, fetch profile
2. **Execute Profile**: Run InSpec against target asset
3. **Parse Results**: Extract control results from JSON output
4. **Map Controls**: Map InSpec controls to compliance framework controls
5. **Update Assessment**: Update compliance assessment results
6. **Cleanup**: Remove temporary files

**Input**:
```json
{
  "run_id": "770e8400-e29b-41d4-a716-446655440002",
  "profile_id": "550e8400-e29b-41d4-a716-446655440000",
  "asset_id": "660e8400-e29b-41d4-a716-446655440001",
  "org_id": "...",
  "profile_url": "https://github.com/dev-sec/cis-linux-benchmark",
  "platform": "linux",
  "asset_type": "vm"
}
```

**Result**:
```json
{
  "run_id": "770e8400-e29b-41d4-a716-446655440002",
  "status": "completed",
  "started_at": "2025-12-05T10:00:01Z",
  "completed_at": "2025-12-05T10:05:23Z",
  "duration": "5m22s",
  "total_tests": 42,
  "passed_tests": 38,
  "failed_tests": 4,
  "skipped_tests": 0
}
```

### BatchInSpecExecutionWorkflow

Executes an InSpec profile across multiple assets in parallel.

**Input**:
```json
{
  "profile_id": "550e8400-e29b-41d4-a716-446655440000",
  "asset_ids": [
    "660e8400-e29b-41d4-a716-446655440001",
    "660e8400-e29b-41d4-a716-446655440002",
    "660e8400-e29b-41d4-a716-446655440003"
  ],
  "org_id": "..."
}
```

## Development

### Prerequisites

- Go 1.21+
- PostgreSQL 14+
- InSpec CLI installed on execution nodes
- Temporal cluster running

### Running Tests

```bash
# Unit tests
go test ./pkg/inspec/...

# Integration tests (requires database)
go test ./pkg/inspec/... -tags=integration
```

### Adding New Profiles

1. Create profile mapping file in `pkg/inspec/profiles/`:

```go
package profiles

func GetMyFrameworkMappings() []inspec.ControlMapping {
    return []inspec.ControlMapping{
        {
            InSpecControlID:   "control-1",
            MappingConfidence: 1.0,
            Notes:            "Control description",
        },
        // ... more mappings
    }
}
```

2. Create profile in database:

```bash
curl -X POST http://localhost:8080/api/v1/inspec/profiles \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-framework-baseline",
    "version": "1.0.0",
    "title": "My Framework Baseline",
    "framework_id": "...",
    "profile_url": "https://github.com/org/profile",
    "platforms": ["linux"]
  }'
```

3. Create control mappings via API or seed data.

## Performance Considerations

- **Parallel Execution**: Use `BatchInSpecExecutionWorkflow` for multiple assets
- **Caching**: Profile fetching is cached per execution
- **Timeouts**: InSpec executions timeout after 15 minutes
- **Retries**: Failed executions retry up to 3 times with exponential backoff
- **Resource Limits**: Consider CPU/memory when running many concurrent profiles

## Security

- InSpec executions run in isolated temporary directories
- SSH keys and cloud credentials managed via secure vault
- Results stored with org-level isolation
- API endpoints protected by RBAC
- Audit logging for all profile executions

## Troubleshooting

### Run Stuck in "pending" Status

Check Temporal worker is running:
```bash
kubectl logs -f deployment/orchestrator -n ql-rf
```

### InSpec Execution Failures

Check activity logs for detailed error messages:
```bash
temporal workflow show --workflow-id inspec-{profile_id}-{asset_id}
```

### Profile Not Found

Verify profile exists and framework mapping is correct:
```bash
curl -X GET http://localhost:8080/api/v1/inspec/profiles/{profileId}
```

## References

- [InSpec Documentation](https://docs.chef.io/inspec/)
- [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks/)
- [SOC 2 Compliance](https://www.aicpa.org/soc)
- [Temporal Documentation](https://docs.temporal.io/)
