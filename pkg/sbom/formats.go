package sbom

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// generateSPDXContent generates SPDX 2.3 format content.
func (g *Generator) generateSPDXContent(sbom *SBOM, packages []Package) (map[string]interface{}, error) {
	content := map[string]interface{}{
		"spdxVersion":       "SPDX-2.3",
		"dataLicense":       "CC0-1.0",
		"SPDXID":            "SPDXRef-DOCUMENT",
		"name":              fmt.Sprintf("SBOM for Image %s", sbom.ImageID),
		"documentNamespace": fmt.Sprintf("https://ql-rf.quantumlayer.io/sbom/%s", sbom.ID),
		"creationInfo": map[string]interface{}{
			"created":  sbom.GeneratedAt.Format(time.RFC3339),
			"creators": []string{
				"Tool: QL-RF SBOM Generator",
			},
		},
	}

	// Add packages
	var spdxPackages []map[string]interface{}
	for i, pkg := range packages {
		spdxPkg := map[string]interface{}{
			"SPDXID":       fmt.Sprintf("SPDXRef-Package-%d", i+1),
			"name":         pkg.Name,
			"versionInfo":  pkg.Version,
			"downloadLocation": "NOASSERTION",
		}

		if pkg.Supplier != "" {
			spdxPkg["supplier"] = pkg.Supplier
		}

		if pkg.License != "" {
			spdxPkg["licenseConcluded"] = pkg.License
			spdxPkg["licenseDeclared"] = pkg.License
		} else {
			spdxPkg["licenseConcluded"] = "NOASSERTION"
			spdxPkg["licenseDeclared"] = "NOASSERTION"
		}

		if pkg.Checksum != "" {
			spdxPkg["checksums"] = []map[string]string{
				{
					"algorithm":     "SHA256",
					"checksumValue": pkg.Checksum,
				},
			}
		}

		if pkg.PURL != "" {
			spdxPkg["externalRefs"] = []map[string]interface{}{
				{
					"referenceCategory": "PACKAGE-MANAGER",
					"referenceType":     "purl",
					"referenceLocator":  pkg.PURL,
				},
			}
		}

		if pkg.CPE != "" {
			if refs, ok := spdxPkg["externalRefs"].([]map[string]interface{}); ok {
				refs = append(refs, map[string]interface{}{
					"referenceCategory": "SECURITY",
					"referenceType":     "cpe23Type",
					"referenceLocator":  pkg.CPE,
				})
				spdxPkg["externalRefs"] = refs
			} else {
				spdxPkg["externalRefs"] = []map[string]interface{}{
					{
						"referenceCategory": "SECURITY",
						"referenceType":     "cpe23Type",
						"referenceLocator":  pkg.CPE,
					},
				}
			}
		}

		spdxPackages = append(spdxPackages, spdxPkg)
	}

	content["packages"] = spdxPackages

	// Add relationships
	var relationships []map[string]string
	for i := range packages {
		relationships = append(relationships, map[string]string{
			"spdxElementId":      "SPDXRef-DOCUMENT",
			"relationshipType":   "DESCRIBES",
			"relatedSpdxElement": fmt.Sprintf("SPDXRef-Package-%d", i+1),
		})
	}
	content["relationships"] = relationships

	return content, nil
}

// generateCycloneDXContent generates CycloneDX 1.5 format content.
func (g *Generator) generateCycloneDXContent(sbom *SBOM, packages []Package) (map[string]interface{}, error) {
	content := map[string]interface{}{
		"bomFormat":    "CycloneDX",
		"specVersion":  "1.5",
		"serialNumber": fmt.Sprintf("urn:uuid:%s", sbom.ID),
		"version":      1,
		"metadata": map[string]interface{}{
			"timestamp": sbom.GeneratedAt.Format(time.RFC3339),
			"tools": []map[string]interface{}{
				{
					"vendor":  "QuantumLayer",
					"name":    "QL-RF SBOM Generator",
					"version": "1.0.0",
				},
			},
			"component": map[string]interface{}{
				"type":    "container",
				"name":    fmt.Sprintf("image-%s", sbom.ImageID),
				"version": "latest",
				"bom-ref": fmt.Sprintf("image-%s", sbom.ImageID),
			},
		},
	}

	// Add components (packages)
	var components []map[string]interface{}
	for _, pkg := range packages {
		component := map[string]interface{}{
			"type":    mapPackageTypeToCycloneDX(pkg.Type),
			"name":    pkg.Name,
			"version": pkg.Version,
			"bom-ref": fmt.Sprintf("pkg:%s/%s@%s", pkg.Type, pkg.Name, pkg.Version),
		}

		if pkg.PURL != "" {
			component["purl"] = pkg.PURL
		}

		if pkg.CPE != "" {
			component["cpe"] = pkg.CPE
		}

		if pkg.Supplier != "" {
			component["supplier"] = map[string]interface{}{
				"name": pkg.Supplier,
			}
		}

		if pkg.License != "" {
			component["licenses"] = []map[string]interface{}{
				{
					"license": map[string]string{
						"id": pkg.License,
					},
				},
			}
		}

		if pkg.Checksum != "" {
			component["hashes"] = []map[string]string{
				{
					"alg":     "SHA-256",
					"content": pkg.Checksum,
				},
			}
		}

		if pkg.SourceURL != "" {
			component["externalReferences"] = []map[string]string{
				{
					"type": "distribution",
					"url":  pkg.SourceURL,
				},
			}
		}

		components = append(components, component)
	}

	content["components"] = components

	// Add dependencies (simplified - all components depend on the root)
	var dependencies []map[string]interface{}
	rootRef := fmt.Sprintf("image-%s", sbom.ImageID)
	var dependsOn []string
	for _, pkg := range packages {
		dependsOn = append(dependsOn, fmt.Sprintf("pkg:%s/%s@%s", pkg.Type, pkg.Name, pkg.Version))
	}
	dependencies = append(dependencies, map[string]interface{}{
		"ref":       rootRef,
		"dependsOn": dependsOn,
	})

	content["dependencies"] = dependencies

	return content, nil
}

// mapPackageTypeToCycloneDX maps package types to CycloneDX component types.
func mapPackageTypeToCycloneDX(pkgType string) string {
	switch pkgType {
	case "deb", "rpm", "apk":
		return "library"
	case "npm", "pip", "pypi", "go", "golang", "maven", "nuget":
		return "library"
	case "container", "docker":
		return "container"
	case "os":
		return "operating-system"
	case "application", "app":
		return "application"
	default:
		return "library"
	}
}

// ExportSPDX exports an SBOM in SPDX format.
func (s *Service) ExportSPDX(ctx context.Context, sbomID string) (map[string]interface{}, error) {
	id, err := uuid.Parse(sbomID)
	if err != nil {
		return nil, fmt.Errorf("invalid sbom id: %w", err)
	}

	sbom, err := s.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get sbom: %w", err)
	}

	// If already in SPDX format, return as-is
	if sbom.Format == FormatSPDX {
		return sbom.Content, nil
	}

	// Otherwise, convert from CycloneDX to SPDX
	packages, err := s.GetPackages(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get packages: %w", err)
	}

	// Create a generator to convert
	g := &Generator{logger: s.logger}
	return g.generateSPDXContent(sbom, packages)
}

// ExportCycloneDX exports an SBOM in CycloneDX format.
func (s *Service) ExportCycloneDX(ctx context.Context, sbomID string) (map[string]interface{}, error) {
	id, err := uuid.Parse(sbomID)
	if err != nil {
		return nil, fmt.Errorf("invalid sbom id: %w", err)
	}

	sbom, err := s.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get sbom: %w", err)
	}

	// If already in CycloneDX format, return as-is
	if sbom.Format == FormatCycloneDX {
		return sbom.Content, nil
	}

	// Otherwise, convert from SPDX to CycloneDX
	packages, err := s.GetPackages(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get packages: %w", err)
	}

	// Create a generator to convert
	g := &Generator{logger: s.logger}
	return g.generateCycloneDXContent(sbom, packages)
}
