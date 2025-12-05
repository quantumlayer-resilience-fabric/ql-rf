# SBOM Package - Quick Start Guide

## Installation

The SBOM package is already part of QL-RF. Just import it:

```go
import "github.com/quantumlayerhq/ql-rf/pkg/sbom"
```

## Database Setup

Run the migration to create the SBOM tables:

```bash
make migrate-up
```

Verify the tables were created:

```bash
psql $RF_DATABASE_URL -c "\dt sbom*"
```

Expected output:
```
               List of relations
 Schema |          Name          | Type  |  Owner
--------+------------------------+-------+----------
 public | sbom_dependencies      | table | postgres
 public | sbom_packages          | table | postgres
 public | sbom_vulnerabilities   | table | postgres
 public | sboms                  | table | postgres
```

## Basic Usage

### 1. Generate an SBOM

```go
package main

import (
    "context"
    "database/sql"
    "log"
    "log/slog"

    "github.com/google/uuid"
    "github.com/quantumlayerhq/ql-rf/pkg/sbom"
)

func main() {
    // Connect to database
    db, err := sql.Open("postgres", "your-connection-string")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create generator
    logger := slog.Default()
    generator := sbom.NewGenerator(db, logger)

    // Generate SBOM for an image
    result, err := generator.Generate(context.Background(), sbom.GenerateRequest{
        ImageID: uuid.MustParse("your-image-id"),
        OrgID:   uuid.MustParse("your-org-id"),
        Format:  sbom.FormatSPDX,
        Scanner: "ql-rf",
        Dockerfile: `
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y nginx curl wget
        `,
        Manifests: map[string]string{
            "npm": `{
                "name": "my-app",
                "version": "1.0.0",
                "dependencies": {
                    "express": "^4.18.2",
                    "lodash": "^4.17.21"
                }
            }`,
            "pip": `
django==4.2.0
requests>=2.28.0
psycopg2-binary==2.9.6
            `,
        },
        IncludeVulns: true,
    })

    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Generated SBOM: %s", result.SBOM.ID)
    log.Printf("Packages: %d", result.PackageCount)
    log.Printf("Vulnerabilities: %d", result.VulnCount)
}
```

### 2. Retrieve an SBOM

```go
// Create service
svc := sbom.NewService(db, logger)

// Get SBOM by ID
sbomDoc, err := svc.Get(context.Background(), sbomID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("SBOM Format: %s\n", sbomDoc.Format)
fmt.Printf("Packages: %d\n", sbomDoc.PackageCount)
fmt.Printf("Generated: %s\n", sbomDoc.GeneratedAt)
```

### 3. Get Packages

```go
packages, err := svc.GetPackages(context.Background(), sbomID)
if err != nil {
    log.Fatal(err)
}

for _, pkg := range packages {
    fmt.Printf("Package: %s@%s (%s)\n", pkg.Name, pkg.Version, pkg.Type)
    fmt.Printf("  PURL: %s\n", pkg.PURL)
    fmt.Printf("  License: %s\n", pkg.License)
}
```

### 4. Get Vulnerabilities

```go
// Get all vulnerabilities
vulns, err := svc.GetVulnerabilities(context.Background(), sbomID, nil)
if err != nil {
    log.Fatal(err)
}

// Get critical/high vulnerabilities only
filter := &sbom.VulnerabilityFilter{
    SBOMID:     sbomID,
    Severities: []string{"critical", "high"},
}

criticalVulns, err := svc.GetVulnerabilities(context.Background(), sbomID, filter)
if err != nil {
    log.Fatal(err)
}

for _, vuln := range criticalVulns {
    fmt.Printf("CVE: %s (%s)\n", vuln.CVEID, vuln.Severity)
    if vuln.CVSSScore != nil {
        fmt.Printf("  CVSS: %.1f\n", *vuln.CVSSScore)
    }
    fmt.Printf("  Fixed in: %s\n", vuln.FixedVersion)
}
```

### 5. Export SBOM

```go
// Export as SPDX
spdxContent, err := svc.ExportSPDX(context.Background(), sbomID.String())
if err != nil {
    log.Fatal(err)
}

// Save to file
jsonBytes, _ := json.MarshalIndent(spdxContent, "", "  ")
os.WriteFile("sbom.spdx.json", jsonBytes, 0644)

// Export as CycloneDX
cycloneDXContent, err := svc.ExportCycloneDX(context.Background(), sbomID.String())
if err != nil {
    log.Fatal(err)
}
```

## API Usage

### Generate SBOM via API

```bash
curl -X POST http://localhost:8080/api/v1/images/{image-id}/sbom/generate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "format": "spdx",
    "scanner": "ql-rf",
    "include_vulns": true,
    "dockerfile": "FROM ubuntu:22.04\nRUN apt-get install -y nginx",
    "manifests": {
      "npm": "{\"name\":\"app\",\"dependencies\":{\"express\":\"4.18.2\"}}"
    }
  }'
```

### Get SBOM for Image

```bash
curl "http://localhost:8080/api/v1/images/{image-id}/sbom?include_packages=true&include_vulns=true" \
  -H "Authorization: Bearer $TOKEN"
```

### Filter Vulnerabilities

```bash
# Critical vulnerabilities with exploits
curl "http://localhost:8080/api/v1/sbom/{sbom-id}/vulnerabilities?severity=critical&has_exploit=true" \
  -H "Authorization: Bearer $TOKEN"

# High CVSS score vulnerabilities with fixes
curl "http://localhost:8080/api/v1/sbom/{sbom-id}/vulnerabilities?min_cvss=7.0&fix_available=true" \
  -H "Authorization: Bearer $TOKEN"
```

### Export SBOM

```bash
# Export as CycloneDX
curl "http://localhost:8080/api/v1/sbom/{sbom-id}/export?format=cyclonedx" \
  -H "Authorization: Bearer $TOKEN" > sbom.cyclonedx.json

# Export as SPDX
curl "http://localhost:8080/api/v1/sbom/{sbom-id}/export?format=spdx" \
  -H "Authorization: Bearer $TOKEN" > sbom.spdx.json
```

## Integration with Image Build Pipeline

Add SBOM generation to your image build process:

```go
// After building/updating an image
imageID := image.ID

// Generate SBOM
generator := sbom.NewGenerator(db, logger)
result, err := generator.Generate(ctx, sbom.GenerateRequest{
    ImageID:      imageID,
    OrgID:        org.ID,
    Format:       sbom.FormatSPDX,
    Scanner:      "ql-rf",
    Dockerfile:   dockerfileContent,
    IncludeVulns: true,
})

if err != nil {
    logger.Error("sbom generation failed", "error", err)
    return err
}

// Update image with SBOM reference
sbomURL := fmt.Sprintf("/api/v1/sbom/%s/export?format=spdx", result.SBOM.ID)
_, err = imageService.UpdateImage(ctx, UpdateImageInput{
    ID:    imageID,
    OrgID: org.ID,
    Params: UpdateImageParams{
        SBOMUrl: &sbomURL,
    },
})
```

## Parsing Different Manifest Types

### NPM (package.json)

```go
manifest, err := parser.Parse("npm", packageJSONContent)
// or
manifest, err := parser.Parse("package.json", packageJSONContent)
```

### Python (requirements.txt)

```go
manifest, err := parser.Parse("pip", requirementsContent)
// or
manifest, err := parser.Parse("requirements.txt", requirementsContent)
```

### Go (go.mod)

```go
manifest, err := parser.Parse("go", goModContent)
// or
manifest, err := parser.Parse("go.mod", goModContent)
```

### Maven (pom.xml)

```go
manifest, err := parser.Parse("maven", pomContent)
// or
manifest, err := parser.Parse("pom.xml", pomContent)
```

## Vulnerability Scanning

The SBOM generator automatically scans for vulnerabilities using the OSV database:

```go
// Generate with vulnerability scanning
result, err := generator.Generate(ctx, sbom.GenerateRequest{
    ImageID:      imageID,
    OrgID:        orgID,
    Format:       sbom.FormatSPDX,
    IncludeVulns: true, // Enable vulnerability scanning
})

// Or add vulnerabilities to existing SBOM
err = generator.EnrichWithVulnerabilities(ctx, sbomID)
```

## Filtering and Statistics

### Get Vulnerability Statistics

```go
stats, err := svc.GetVulnerabilityStats(context.Background(), sbomID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Total vulnerabilities: %d\n", stats["total"])
fmt.Printf("Critical: %d\n", stats["critical"])
fmt.Printf("High: %d\n", stats["high"])
fmt.Printf("With exploits: %d\n", stats["with_exploits"])
fmt.Printf("With fixes: %d\n", stats["with_fixes"])
fmt.Printf("Highest CVSS: %.1f\n", stats["highest_cvss_score"])
```

### Advanced Filtering

```go
minCVSS := 7.0
hasExploit := true
fixAvailable := true

filter := &sbom.VulnerabilityFilter{
    SBOMID:       sbomID,
    Severities:   []string{"critical", "high"},
    MinCVSS:      &minCVSS,
    HasExploit:   &hasExploit,
    FixAvailable: &fixAvailable,
}

vulns, err := svc.GetVulnerabilities(ctx, sbomID, filter)
```

## Testing

Run the SBOM package tests:

```bash
# All tests
go test ./pkg/sbom/...

# With coverage
go test -cover ./pkg/sbom/...

# Verbose output
go test -v ./pkg/sbom/...

# Specific test
go test -run TestFormatIsValid ./pkg/sbom/...
```

## Configuration

Environment variables (optional):

```bash
# No special configuration needed - SBOM package uses existing QL-RF config
RF_DATABASE_URL=postgres://...
RF_LLM_API_KEY=...  # For enhanced vulnerability analysis (future)
```

## Troubleshooting

### SBOM Generation Fails

```go
// Check if image exists
image, err := imageService.GetImage(ctx, imageID)
if err != nil {
    log.Printf("Image not found: %v", err)
}

// Validate manifest content
parser := sbom.NewParser()
manifest, err := parser.Parse("npm", content)
if err != nil {
    log.Printf("Invalid manifest: %v", err)
}
```

### No Vulnerabilities Found

- OSV database may not have data for all packages
- Some package types aren't yet supported by OSV
- Check package name/version formatting

```go
// Check OSV ecosystem mapping
ecosystem := mapPackageTypeToOSVEcosystem("npm")  // Returns "npm"
ecosystem := mapPackageTypeToOSVEcosystem("pip")  // Returns "PyPI"
```

### Database Connection Issues

```bash
# Verify database connection
psql $RF_DATABASE_URL -c "SELECT version()"

# Check if SBOM tables exist
psql $RF_DATABASE_URL -c "\dt sbom*"

# Check migration status
make migrate-status
```

## Next Steps

1. **Wire up API handlers** - Add SBOM handlers to your router
2. **Integrate with CI/CD** - Auto-generate SBOMs on image builds
3. **Set up monitoring** - Track SBOM generation and vulnerability trends
4. **Configure alerts** - Alert on critical vulnerabilities
5. **Compliance reporting** - Export SBOMs for audits

## Support

For issues or questions:
- See full documentation: `pkg/sbom/README.md`
- Implementation summary: `SBOM_IMPLEMENTATION_SUMMARY.md`
- Run tests: `go test ./pkg/sbom/...`
