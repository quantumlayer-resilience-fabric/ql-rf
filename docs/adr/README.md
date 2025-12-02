# Architecture Decision Records (ADRs)

This directory contains Architecture Decision Records for QL-RF.

## Index

| ADR | Title | Status |
|-----|-------|--------|
| [ADR-001](ADR-001-contracts-first.md) | Contracts-First Design | Accepted |
| [ADR-002](ADR-002-agentless-by-default.md) | Agentless by Default | Accepted |
| [ADR-003](ADR-003-cosign-signing.md) | Cosign for Artifact Signing | Accepted |
| [ADR-004](ADR-004-temporal-workflows.md) | Temporal for Workflows | Accepted |
| [ADR-005](ADR-005-opa-policy-engine.md) | OPA as Policy Engine | Accepted |
| [ADR-006](ADR-006-sbom-spdx.md) | SPDX for SBOM Format | Accepted |
| [ADR-007](ADR-007-llm-first-orchestration.md) | LLM-First Orchestration Architecture | Accepted |
| [ADR-008](ADR-008-task-plan-run-lifecycle.md) | Task/Plan/Run Lifecycle & State Machine | Accepted |
| [ADR-009](ADR-009-tool-risk-taxonomy-hitl.md) | Tool Risk Taxonomy & HITL Policy | Accepted |
| [ADR-010](ADR-010-rbac-permission-authorization.md) | RBAC with Permission-Based Authorization | Accepted |
| [ADR-011](ADR-011-row-level-security.md) | Row-Level Security for Multi-Tenancy | Accepted |

## ADR Template

When creating new ADRs, use this template:

```markdown
# ADR-XXX: [Title]

## Status
[Proposed | Accepted | Deprecated | Superseded by ADR-YYY]

## Context
[Why we need to make this decision]

## Decision
[What we decided]

## Consequences

### Positive
[Benefits of this decision]

### Negative
[Drawbacks or risks]

### Mitigations
[How we address the negatives]
```

## References

- [ADR GitHub Organization](https://adr.github.io/)
- [Michael Nygard's ADR Article](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
