package sbom

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Parser parses package manifest files.
type Parser struct{}

// NewParser creates a new manifest parser.
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses a package manifest based on type.
func (p *Parser) Parse(manifestType, content string) (*PackageManifest, error) {
	switch manifestType {
	case "npm", "package.json":
		return p.parseNPM(content)
	case "pip", "requirements.txt":
		return p.parsePip(content)
	case "go", "go.mod":
		return p.parseGoMod(content)
	case "maven", "pom.xml":
		return p.parseMaven(content)
	case "nuget", "packages.config":
		return p.parseNuGet(content)
	case "gemfile", "ruby":
		return p.parseGemfile(content)
	case "cargo", "rust":
		return p.parseCargo(content)
	default:
		return nil, fmt.Errorf("unsupported manifest type: %s", manifestType)
	}
}

// parseNPM parses package.json files.
func (p *Parser) parseNPM(content string) (*PackageManifest, error) {
	var data struct {
		Name            string            `json:"name"`
		Version         string            `json:"version"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil, fmt.Errorf("unmarshal package.json: %w", err)
	}

	manifest := &PackageManifest{
		Type: "npm",
		Metadata: map[string]string{
			"name":    data.Name,
			"version": data.Version,
		},
	}

	// Add dependencies
	for name, version := range data.Dependencies {
		manifest.Packages = append(manifest.Packages, ManifestPackage{
			Name:    name,
			Version: strings.TrimPrefix(version, "^"),
			Dev:     false,
		})
	}

	// Add dev dependencies
	for name, version := range data.DevDependencies {
		manifest.Packages = append(manifest.Packages, ManifestPackage{
			Name:    name,
			Version: strings.TrimPrefix(version, "^"),
			Dev:     true,
		})
	}

	return manifest, nil
}

// parsePip parses requirements.txt files.
func (p *Parser) parsePip(content string) (*PackageManifest, error) {
	manifest := &PackageManifest{
		Type: "pip",
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse package==version or package>=version
		var name, version string
		if strings.Contains(line, "==") {
			parts := strings.Split(line, "==")
			name = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				version = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, ">=") {
			parts := strings.Split(line, ">=")
			name = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				version = strings.TrimSpace(parts[1])
			}
		} else {
			name = line
			version = "unknown"
		}

		manifest.Packages = append(manifest.Packages, ManifestPackage{
			Name:    name,
			Version: version,
		})
	}

	return manifest, nil
}

// parseGoMod parses go.mod files.
func (p *Parser) parseGoMod(content string) (*PackageManifest, error) {
	manifest := &PackageManifest{
		Type: "go",
	}

	lines := strings.Split(content, "\n")
	inRequire := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "module ") {
			moduleName := strings.TrimPrefix(line, "module ")
			manifest.Metadata = map[string]string{
				"module": strings.TrimSpace(moduleName),
			}
			continue
		}

		if strings.HasPrefix(line, "require (") {
			inRequire = true
			continue
		}

		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		if strings.HasPrefix(line, "require ") || inRequire {
			// Parse: require github.com/pkg/name v1.2.3
			reqLine := strings.TrimPrefix(line, "require ")
			parts := strings.Fields(reqLine)
			if len(parts) >= 2 {
				name := parts[0]
				version := parts[1]

				// Skip indirect dependencies if needed
				isIndirect := len(parts) > 2 && parts[2] == "// indirect"

				manifest.Packages = append(manifest.Packages, ManifestPackage{
					Name:    name,
					Version: version,
					Dev:     isIndirect,
				})
			}
		}
	}

	return manifest, nil
}

// parseMaven parses pom.xml files (simplified).
func (p *Parser) parseMaven(content string) (*PackageManifest, error) {
	manifest := &PackageManifest{
		Type: "maven",
	}

	// Simple XML parsing for <dependency> blocks
	// In production, use encoding/xml for proper parsing
	lines := strings.Split(content, "\n")
	var currentDep ManifestPackage
	inDependency := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "<dependency>") {
			inDependency = true
			currentDep = ManifestPackage{}
			continue
		}

		if strings.Contains(line, "</dependency>") {
			if currentDep.Name != "" {
				manifest.Packages = append(manifest.Packages, currentDep)
			}
			inDependency = false
			currentDep = ManifestPackage{}
			continue
		}

		if inDependency {
			if strings.Contains(line, "<artifactId>") {
				artifactID := extractXMLValue(line, "artifactId")
				currentDep.Name = artifactID
			}
			if strings.Contains(line, "<version>") {
				version := extractXMLValue(line, "version")
				currentDep.Version = version
			}
			if strings.Contains(line, "<scope>test</scope>") {
				currentDep.Dev = true
			}
		}
	}

	return manifest, nil
}

// parseNuGet parses packages.config files (simplified).
func (p *Parser) parseNuGet(content string) (*PackageManifest, error) {
	manifest := &PackageManifest{
		Type: "nuget",
	}

	// Simple XML parsing for <package> elements
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "<package ") {
			// Extract: <package id="PackageName" version="1.0.0" />
			name := extractAttribute(line, "id")
			version := extractAttribute(line, "version")

			if name != "" {
				manifest.Packages = append(manifest.Packages, ManifestPackage{
					Name:    name,
					Version: version,
				})
			}
		}
	}

	return manifest, nil
}

// parseGemfile parses Gemfile files.
func (p *Parser) parseGemfile(content string) (*PackageManifest, error) {
	manifest := &PackageManifest{
		Type: "ruby",
	}

	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse: gem 'package-name', '~> 1.0.0'
		if strings.HasPrefix(line, "gem ") {
			parts := strings.Split(line, ",")
			if len(parts) >= 1 {
				name := strings.Trim(parts[0][4:], "' \"")
				version := "unknown"

				if len(parts) >= 2 {
					version = strings.Trim(parts[1], "' \"~>")
					version = strings.TrimSpace(version)
				}

				manifest.Packages = append(manifest.Packages, ManifestPackage{
					Name:    name,
					Version: version,
				})
			}
		}
	}

	return manifest, nil
}

// parseCargo parses Cargo.toml files.
func (p *Parser) parseCargo(content string) (*PackageManifest, error) {
	manifest := &PackageManifest{
		Type: "cargo",
	}

	lines := strings.Split(content, "\n")
	inDependencies := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "[dependencies]") {
			inDependencies = true
			continue
		}

		if strings.HasPrefix(line, "[") && inDependencies {
			inDependencies = false
			continue
		}

		if inDependencies && strings.Contains(line, "=") {
			// Parse: package = "1.0.0" or package = { version = "1.0.0" }
			parts := strings.Split(line, "=")
			if len(parts) >= 2 {
				name := strings.TrimSpace(parts[0])
				versionStr := strings.TrimSpace(parts[1])

				// Simple version extraction
				version := strings.Trim(versionStr, "\" ")
				if strings.Contains(version, "version") {
					// Extract from { version = "1.0.0" }
					versionParts := strings.Split(version, "\"")
					if len(versionParts) >= 2 {
						version = versionParts[1]
					}
				}

				manifest.Packages = append(manifest.Packages, ManifestPackage{
					Name:    name,
					Version: version,
				})
			}
		}
	}

	return manifest, nil
}

// extractXMLValue extracts value from XML tag.
func extractXMLValue(line, tag string) string {
	openTag := "<" + tag + ">"
	closeTag := "</" + tag + ">"

	startIdx := strings.Index(line, openTag)
	endIdx := strings.Index(line, closeTag)

	if startIdx != -1 && endIdx != -1 {
		startIdx += len(openTag)
		return strings.TrimSpace(line[startIdx:endIdx])
	}

	return ""
}

// extractAttribute extracts an attribute value from an XML tag.
func extractAttribute(line, attr string) string {
	attrPattern := attr + "=\""
	startIdx := strings.Index(line, attrPattern)

	if startIdx == -1 {
		return ""
	}

	startIdx += len(attrPattern)
	endIdx := strings.Index(line[startIdx:], "\"")

	if endIdx == -1 {
		return ""
	}

	return line[startIdx : startIdx+endIdx]
}
