# QL-RF Contracts

This directory contains YAML contracts and JSON schemas that define the structure of golden images and events in QL-RF.

## Contract Files

| File | Description |
|------|-------------|
| `image.contract.yaml` | Golden image contract schema (JSON Schema format) |
| `image.contract.windows.yaml` | Extended schema for Windows images |
| `events.schema.json` | Event schema for Kafka messages |
| `examples/` | Example contracts for reference |

## Image Contract

Golden images are defined using YAML contracts that specify:

- **Metadata**: Family, version, labels
- **OS**: Operating system name, version, architecture
- **Compliance**: CIS level, signing, SBOM requirements
- **Platforms**: AWS AMI IDs, Azure SIG, GCP images, vSphere templates
- **Lifecycle**: Status, retention policy, deprecation dates

### Example Usage

```yaml
apiVersion: ql-rf.io/v1
kind: GoldenImage

metadata:
  family: rhel9-baseline
  version: 1.6.4

spec:
  os:
    name: rhel
    version: "9.3"

  compliance:
    cisLevel: 2
    signed: true
    sbomRequired: true

  platforms:
    aws:
      regions: [us-east-1, eu-west-1]
      amiIds:
        us-east-1: ami-0123456789abcdef0
```

## Validating Contracts

Use `yq` and `ajv` to validate contracts:

```bash
# Install tools
npm install -g ajv-cli

# Validate a contract
ajv validate -s contracts/image.contract.yaml -d contracts/examples/rhel9-baseline.yaml
```

## Event Schema

Events follow the CloudEvents specification with QL-RF extensions:

| Event Type | Description |
|------------|-------------|
| `image.published` | New image version published |
| `image.deprecated` | Image marked as deprecated |
| `asset.discovered` | New asset discovered by connector |
| `asset.updated` | Asset state changed |
| `drift.detected` | Drift threshold exceeded |
| `compliance.violation` | Policy violation detected |
| `workflow.*` | Workflow lifecycle events |

### Event Structure

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "drift.detected",
  "source": "drift-service",
  "timestamp": "2024-01-15T10:30:00Z",
  "specversion": "1.0",
  "orgId": "org-123",
  "data": {
    "reportId": "...",
    "status": "warning",
    "coveragePct": 87.5
  }
}
```

## Schema Evolution

Contract schemas are versioned via `apiVersion`. When making breaking changes:

1. Create new version (e.g., `ql-rf.io/v2`)
2. Support both versions during migration
3. Deprecate old version with timeline
4. Remove old version after migration complete
