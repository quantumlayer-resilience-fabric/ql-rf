# ADR-005: OPA as Policy Engine

## Status
Accepted

## Context
QL-RF must enforce compliance policies across multiple dimensions:
- Image requirements (CIS level, signing, SBOM presence)
- Deployment rules (approved images per environment)
- Access control (who can promote images to production)
- Drift thresholds (acceptable coverage percentages)

These policies must be:
- Declarative and auditable
- Decoupled from application code
- Testable in isolation
- Customizable per organization

Options considered:
1. **Open Policy Agent (OPA)**: General-purpose policy engine
2. **Kyverno**: Kubernetes-native, YAML policies
3. **Cedar (AWS)**: New, limited ecosystem
4. **Hardcoded logic**: Fastest but inflexible

## Decision
We adopt **Open Policy Agent (OPA)** with **Rego** policies:

1. **Decoupled policies**: Policies stored separately from code
2. **Rego language**: Declarative, purpose-built for policy
3. **Bundle distribution**: Policies synced from Git/OCI
4. **Partial evaluation**: Pre-compile for performance
5. **Audit logging**: Track all policy decisions

Example policy:
```rego
# policy/image_compliance.rego
package ql_rf.image

default allow = false

# Allow image if it meets all compliance requirements
allow {
    input.image.signed == true
    input.image.sbom_present == true
    input.image.cis_level >= 1
    not image_deprecated
}

image_deprecated {
    input.image.status == "deprecated"
}

# Deny reasons for audit
deny[msg] {
    not input.image.signed
    msg := "Image must be signed with Cosign"
}

deny[msg] {
    not input.image.sbom_present
    msg := "Image must have SBOM attached"
}
```

Integration points:
- API request validation (before image registration)
- Deployment gate (before Terraform apply)
- Drift alerts (coverage thresholds)
- RBAC enhancement (fine-grained permissions)

## Consequences

### Positive
- Industry standard (CNCF graduated)
- Policies as code (version controlled, reviewed)
- Rich ecosystem (Conftest, Gatekeeper, etc.)
- Testable policies with OPA test framework
- Decoupled from application releases

### Negative
- Rego learning curve
- Additional service dependency (OPA server)
- Policy debugging can be challenging

### Mitigations
- Provide policy templates for common use cases
- Use OPA's built-in REPL for policy development
- Integrate Conftest in CI for policy testing
- Start with embedded OPA (no separate server)
