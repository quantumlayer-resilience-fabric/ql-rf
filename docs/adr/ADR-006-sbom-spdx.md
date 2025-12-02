# ADR-006: SPDX for SBOM Format

## Status
Accepted

## Context
Software Bill of Materials (SBOM) is required for:
- Vulnerability management (know what's in the image)
- Compliance (executive orders, regulatory requirements)
- License compliance (open source obligations)
- Incident response (identify affected systems)

Two dominant SBOM formats exist:
1. **SPDX**: Linux Foundation standard, ISO/IEC 5962:2021
2. **CycloneDX**: OWASP project, security-focused

Both are supported by major tools, but differ in:
- Schema design (SPDX: document-centric, CycloneDX: component-centric)
- Tooling ecosystem
- Government mandates (US NTIA prefers SPDX)

## Decision
We adopt **SPDX 2.3** as the primary SBOM format:

1. **ISO standard**: International recognition
2. **Government alignment**: US federal requirements reference SPDX
3. **Mature ecosystem**: Supported by Syft, Trivy, Grype
4. **License focus**: Strong license expression support
5. **Interoperability**: Converters available for CycloneDX

SBOM generation workflow:
```bash
# Generate during image build
syft packages ami://ami-0123456789 -o spdx-json > sbom.spdx.json

# Attach to image metadata
cosign attach sbom --sbom sbom.spdx.json ami://ami-0123456789

# Scan for vulnerabilities
grype sbom:sbom.spdx.json
```

Storage and retrieval:
- SBOMs stored in S3/Blob with versioned paths
- SBOM URL referenced in image contract
- API endpoint for SBOM retrieval and diff

## Consequences

### Positive
- Standards compliance (ISO, NTIA)
- Wide tooling support
- Can convert to CycloneDX if needed
- Enables vulnerability correlation
- Supports license compliance tracking

### Negative
- SPDX JSON is verbose
- Some tools prefer CycloneDX
- Two formats to potentially support

### Mitigations
- Store only SPDX, convert on-demand to CycloneDX
- Use compression for storage
- Provide SBOM summary view in UI (not raw JSON)
- Implement SBOM diff for version comparison
