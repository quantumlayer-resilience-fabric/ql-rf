# ADR-001: Contracts-First Design

## Status
Accepted

## Context
QL-RF manages golden images across multiple platforms (AWS, Azure, GCP, vSphere) with varying metadata formats, lifecycle policies, and compliance requirements. Without a standardized interface, platform-specific details leak into business logic, making it difficult to:
- Enforce consistent compliance policies
- Add new platforms without code changes
- Validate image configurations before deployment
- Generate audit evidence

## Decision
We adopt a **contracts-first design** where all golden images are defined via YAML contracts that specify:
- Image metadata (family, version, OS)
- Compliance requirements (CIS level, SBOM, signatures)
- Platform coordinates (AMI IDs, Azure SIG paths, etc.)
- Lifecycle policies (retention, deprecation)

The contract schema is versioned and validated with JSON Schema. Platform-specific adapters translate contracts to native formats.

```yaml
# Example: contracts/image.contract.yaml
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
    azure:
      subscriptions: [prod-sub-id]
```

## Consequences

### Positive
- Platform-agnostic business logic
- Declarative, version-controlled image definitions
- Easy validation and drift detection
- Self-documenting compliance requirements
- Supports GitOps workflows

### Negative
- Learning curve for contract schema
- Additional abstraction layer
- Need to keep contracts in sync with actual platform resources

### Mitigations
- Provide CLI tooling for contract validation
- Auto-generate contracts from discovered assets
- Implement reconciliation to detect drift between contracts and reality
