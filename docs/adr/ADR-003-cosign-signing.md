# ADR-003: Cosign for Artifact Signing

## Status
Accepted

## Context
Golden images must be verifiable as authentic and unmodified. This is critical for:
- Compliance (prove image provenance)
- Security (prevent supply chain attacks)
- Audit trails (non-repudiation)

Options considered:
1. **Cloud-native signing**: AWS AMI signing, Azure Trusted Launch
2. **Cosign (Sigstore)**: Open-source, keyless signing with OIDC
3. **GPG**: Traditional but complex key management
4. **Notary v2**: CNCF project, container-focused

## Decision
We adopt **Cosign (Sigstore)** for artifact signing:

1. **Keyless signing**: Uses OIDC identity (GitHub Actions, Workday Identity) instead of long-lived keys
2. **Transparency log**: Signatures recorded in Rekor for auditability
3. **Multi-artifact**: Works with container images, SBOMs, and attestations
4. **Ecosystem support**: Integrates with Kubernetes admission controllers

Signing workflow:
```bash
# Sign during image build
cosign sign --key cosign.key ami://ami-0123456789abcdef0

# Verify before deployment
cosign verify --key cosign.pub ami://ami-0123456789abcdef0
```

Enforcement levels:
- **Development**: Warn if unsigned (allow deployment)
- **Staging**: Require signature (block unsigned)
- **Production**: Require signature + attestations (SLSA)

## Consequences

### Positive
- Industry-standard tooling (CNCF project)
- Keyless option reduces key management burden
- Integrates with existing CI/CD (GitHub Actions, GitLab)
- Supports SBOM attestations
- Transparent audit log via Rekor

### Negative
- Requires Sigstore infrastructure (or self-hosted)
- Learning curve for teams unfamiliar with Sigstore
- AMI signing requires custom integration (not native to AWS)

### Mitigations
- Use Sigstore public good instance initially
- Provide wrapper CLI for simplified signing
- Document signing workflow in runbooks
- Store signature metadata in QL-RF database for quick verification
