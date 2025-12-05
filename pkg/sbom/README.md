# SBOM Package

Software Bill of Materials (SBOM) generation and management for QL-RF.

## Overview

This package provides comprehensive SBOM capabilities including:

- **SPDX 2.3** format support (ISO/IEC 5962:2021 standard)
- **CycloneDX 1.5** format support (OWASP standard)
- Package dependency parsing from multiple manifest formats
- Vulnerability correlation with CVE databases
- SBOM storage and retrieval
- Export and format conversion

## Features

### Supported Package Formats

- **npm** - package.json
- **pip** - requirements.txt
- **Go** - go.mod
- **Maven** - pom.xml
- **NuGet** - packages.config
- **Ruby** - Gemfile
- **Rust** - Cargo.toml
- **Debian** - apt packages
- **Alpine** - apk packages
- **RPM** - yum/dnf packages

### Vulnerability Scanning

- Integration with OSV (Open Source Vulnerabilities) database
- CVSS scoring and severity classification
- Exploit availability tracking
- Fix version recommendations

## Usage

### Generating an SBOM

```go
import (
    "context"
    "database/sql"
    "log/slog"

    "github.com/google/uuid"
    "github.com/quantumlayerhq/ql-rf/pkg/sbom"
)

// Initialize generator
db := // ... your database connection
logger := slog.Default()
generator := sbom.NewGenerator(db, logger)

// Generate SBOM
result, err := generator.Generate(context.Background(), sbom.GenerateRequest{
    ImageID:      imageID,
    OrgID:        orgID,
    Format:       sbom.FormatSPDX,
    Scanner:      "ql-rf",
    Dockerfile:   dockerfileContent,
    Manifests: map[string]string{
        "npm": packageJSONContent,
        "pip": requirementsTxtContent,
    },
    IncludeVulns: true,
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Generated SBOM with %d packages\n", result.PackageCount)
```

### Retrieving an SBOM

```go
svc := sbom.NewService(db, logger)

// Get by SBOM ID
sbomDoc, err := svc.Get(ctx, sbomID)

// Get by image ID
sbomDoc, err := svc.GetByImageID(ctx, imageID)

// Get packages
packages, err := svc.GetPackages(ctx, sbomID)

// Get vulnerabilities
vulns, err := svc.GetVulnerabilities(ctx, sbomID, nil)
```

### Filtering Vulnerabilities

```go
filter := &sbom.VulnerabilityFilter{
    SBOMID:       sbomID,
    Severities:   []string{"critical", "high"},
    MinCVSS:      &minScore,
    HasExploit:   &trueVal,
    FixAvailable: &trueVal,
}

vulns, err := svc.GetVulnerabilities(ctx, sbomID, filter)
```

### Exporting SBOMs

```go
// Export as SPDX
spdxContent, err := svc.ExportSPDX(ctx, sbomID.String())

// Export as CycloneDX
cycloneDXContent, err := svc.ExportCycloneDX(ctx, sbomID.String())
```

## API Endpoints

### Image SBOM Operations

- `GET /api/v1/images/{id}/sbom` - Get SBOM for image
- `POST /api/v1/images/{id}/sbom/generate` - Generate new SBOM

### SBOM Operations

- `GET /api/v1/sbom` - List all SBOMs
- `GET /api/v1/sbom/{id}` - Get specific SBOM
- `DELETE /api/v1/sbom/{id}` - Delete SBOM
- `GET /api/v1/sbom/{id}/vulnerabilities` - Get vulnerabilities
- `GET /api/v1/sbom/{id}/export?format=spdx|cyclonedx` - Export SBOM

### Query Parameters

**GetImageSBOM / GetSBOM:**
- `include_packages=true` - Include package details
- `include_vulns=true` - Include vulnerability details

**GetVulnerabilities:**
- `severity=critical,high` - Filter by severity
- `min_cvss=7.0` - Minimum CVSS score
- `has_exploit=true` - Only exploitable vulnerabilities
- `fix_available=true` - Only vulnerabilities with fixes

## Database Schema

### Tables

**sboms** - Main SBOM documents
- Stores complete SBOM in JSONB format
- Supports both SPDX and CycloneDX formats
- Tracks generation metadata

**sbom_packages** - Normalized package data
- Extracted from SBOM documents
- Includes PURL and CPE identifiers
- Enables efficient querying

**sbom_dependencies** - Package relationships
- Tracks dependency graphs
- Supports scoped dependencies

**sbom_vulnerabilities** - Security findings
- CVE/GHSA identifiers
- CVSS scores and severity
- Fix versions and references

### Views

**v_sbom_summary** - SBOM overview with vulnerability counts
**v_package_vulnerabilities** - Per-package vulnerability summary
**v_image_sbom_coverage** - SBOM coverage metrics for images

## Data Formats

### SPDX 2.3 Example

```json
{
  "spdxVersion": "SPDX-2.3",
  "dataLicense": "CC0-1.0",
  "SPDXID": "SPDXRef-DOCUMENT",
  "name": "SBOM for Image abc123",
  "documentNamespace": "https://ql-rf.quantumlayer.io/sbom/uuid",
  "packages": [
    {
      "SPDXID": "SPDXRef-Package-1",
      "name": "express",
      "versionInfo": "4.18.2",
      "licenseConcluded": "MIT",
      "externalRefs": [
        {
          "referenceCategory": "PACKAGE-MANAGER",
          "referenceType": "purl",
          "referenceLocator": "pkg:npm/express@4.18.2"
        }
      ]
    }
  ]
}
```

### CycloneDX 1.5 Example

```json
{
  "bomFormat": "CycloneDX",
  "specVersion": "1.5",
  "serialNumber": "urn:uuid:...",
  "metadata": {
    "tools": [
      {
        "vendor": "QuantumLayer",
        "name": "QL-RF SBOM Generator"
      }
    ]
  },
  "components": [
    {
      "type": "library",
      "name": "express",
      "version": "4.18.2",
      "purl": "pkg:npm/express@4.18.2",
      "licenses": [
        {
          "license": {
            "id": "MIT"
          }
        }
      ]
    }
  ]
}
```

## Testing

Run the test suite:

```bash
go test ./pkg/sbom/...
```

Run with coverage:

```bash
go test -cover ./pkg/sbom/...
```

## Architecture

```
┌─────────────────────────────────────────────┐
│           SBOM Generator                    │
│  ┌────────────┐  ┌──────────────────────┐  │
│  │  Parser    │  │  Vulnerability       │  │
│  │            │  │  Scanner (OSV)       │  │
│  └────────────┘  └──────────────────────┘  │
│         │                   │               │
│         v                   v               │
│  ┌────────────────────────────────────┐    │
│  │     Format Generator               │    │
│  │  • SPDX 2.3                       │    │
│  │  • CycloneDX 1.5                  │    │
│  └────────────────────────────────────┘    │
└─────────────────────────────────────────────┘
                    │
                    v
┌─────────────────────────────────────────────┐
│          SBOM Service                       │
│  • Create/Read/Update/Delete                │
│  • Package Management                       │
│  • Vulnerability Tracking                   │
│  • Statistics & Reporting                   │
└─────────────────────────────────────────────┘
                    │
                    v
┌─────────────────────────────────────────────┐
│         PostgreSQL Database                 │
│  • sboms                                    │
│  • sbom_packages                            │
│  • sbom_dependencies                        │
│  • sbom_vulnerabilities                     │
└─────────────────────────────────────────────┘
```

## Compliance

### Standards Supported

- **SPDX 2.3** (ISO/IEC 5962:2021)
- **CycloneDX 1.5** (OWASP)
- **PURL** (Package URL specification)
- **CPE 2.3** (Common Platform Enumeration)
- **CVSS 3.1** (Common Vulnerability Scoring System)

### Regulatory Compliance

This SBOM implementation supports compliance with:

- Executive Order 14028 (Improving the Nation's Cybersecurity)
- NIST SP 800-218 (Secure Software Development Framework)
- NTIA Minimum Elements for SBOM
- EU Cyber Resilience Act requirements

## Performance

- Package parsing: < 100ms for typical manifests
- SBOM generation: < 1s for images with 100+ packages
- Vulnerability scanning: ~200ms per package (OSV API)
- Database queries: Indexed for sub-second response

## Security

- Row-level security (RLS) enforces org isolation
- All queries filtered by organization ID
- SBOM content validated before storage
- SQL injection protection via parameterized queries
- HTTPS-only communication with external APIs

## Future Enhancements

- [ ] Support for container image scanning (Trivy/Grype integration)
- [ ] SLSA provenance integration
- [ ] VEX (Vulnerability Exploitability eXchange) support
- [ ] License compliance checking
- [ ] SBOM diffing and comparison
- [ ] Automated SBOM updates on new CVEs
- [ ] Integration with GitHub Dependency Graph
- [ ] Support for SWID tags
- [ ] SBOM signing and verification

## License

Copyright (c) 2025 QuantumLayer. All rights reserved.
