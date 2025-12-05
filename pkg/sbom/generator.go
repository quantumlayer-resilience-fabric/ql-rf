package sbom

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/package-url/packageurl-go"
)

// Generator generates SBOMs for images.
type Generator struct {
	db     *sql.DB
	svc    *Service
	parser *Parser
	logger *slog.Logger
}

// NewGenerator creates a new SBOM generator.
func NewGenerator(db *sql.DB, logger *slog.Logger) *Generator {
	if logger == nil {
		logger = slog.Default()
	}
	svc := NewService(db, logger)
	parser := NewParser()

	return &Generator{
		db:     db,
		svc:    svc,
		parser: parser,
		logger: logger.With("component", "sbom-generator"),
	}
}

// GenerateRequest contains parameters for SBOM generation.
type GenerateRequest struct {
	ImageID      uuid.UUID
	OrgID        uuid.UUID
	Format       Format
	Scanner      string
	Dockerfile   string // Optional Dockerfile content
	Manifests    map[string]string // Optional package manifests (type -> content)
	IncludeVulns bool
}

// GenerateResult contains the result of SBOM generation.
type GenerateResult struct {
	SBOM         *SBOM
	Packages     []Package
	Status       string
	Message      string
	PackageCount int
	VulnCount    int
}

// Generate generates an SBOM for an image.
func (g *Generator) Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error) {
	g.logger.Info("generating sbom",
		"image_id", req.ImageID,
		"format", req.Format,
		"scanner", req.Scanner,
	)

	// Validate format
	if !req.Format.IsValid() {
		return nil, fmt.Errorf("invalid format: %s", req.Format)
	}

	// Parse packages from various sources
	var allPackages []Package

	// Parse Dockerfile if provided
	if req.Dockerfile != "" {
		dockerfilePackages, err := g.parseDockerfile(req.Dockerfile)
		if err != nil {
			g.logger.Warn("failed to parse dockerfile", "error", err)
		} else {
			allPackages = append(allPackages, dockerfilePackages...)
		}
	}

	// Parse package manifests
	for manifestType, content := range req.Manifests {
		packages, err := g.parseManifest(manifestType, content)
		if err != nil {
			g.logger.Warn("failed to parse manifest",
				"type", manifestType,
				"error", err,
			)
			continue
		}
		allPackages = append(allPackages, packages...)
	}

	// If no packages found, add placeholder
	if len(allPackages) == 0 {
		g.logger.Warn("no packages found, creating minimal sbom")
		allPackages = []Package{
			{
				Name:    "base-os",
				Version: "unknown",
				Type:    "os",
			},
		}
	}

	// Create SBOM document
	sbom := &SBOM{
		ID:           uuid.New(),
		ImageID:      req.ImageID,
		OrgID:        req.OrgID,
		Format:       req.Format,
		Version:      getFormatVersion(req.Format),
		PackageCount: len(allPackages),
		GeneratedAt:  time.Now(),
		Scanner:      req.Scanner,
	}

	// Generate format-specific content
	var err error
	switch req.Format {
	case FormatSPDX:
		sbom.Content, err = g.generateSPDXContent(sbom, allPackages)
	case FormatCycloneDX:
		sbom.Content, err = g.generateCycloneDXContent(sbom, allPackages)
	default:
		return nil, fmt.Errorf("unsupported format: %s", req.Format)
	}

	if err != nil {
		return nil, fmt.Errorf("generate %s content: %w", req.Format, err)
	}

	// Store SBOM
	if err := g.svc.Create(ctx, sbom); err != nil {
		return nil, fmt.Errorf("store sbom: %w", err)
	}

	// Store packages
	for i := range allPackages {
		allPackages[i].SBOMID = sbom.ID
	}

	if err := g.svc.CreatePackageBatch(ctx, allPackages); err != nil {
		return nil, fmt.Errorf("store packages: %w", err)
	}

	result := &GenerateResult{
		SBOM:         sbom,
		Packages:     allPackages,
		Status:       "success",
		PackageCount: len(allPackages),
	}

	g.logger.Info("sbom generated successfully",
		"sbom_id", sbom.ID,
		"packages", len(allPackages),
	)

	return result, nil
}

// parseDockerfile extracts packages from a Dockerfile.
func (g *Generator) parseDockerfile(content string) ([]Package, error) {
	var packages []Package
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for RUN apt-get install, RUN apk add, RUN yum install, etc.
		if strings.HasPrefix(line, "RUN ") {
			pkgs := g.extractPackagesFromRunCommand(line)
			packages = append(packages, pkgs...)
		}

		// Look for COPY of package.json, requirements.txt, etc.
		if strings.HasPrefix(line, "COPY ") {
			if strings.Contains(line, "package.json") {
				packages = append(packages, Package{
					Name:    "nodejs-dependencies",
					Version: "unknown",
					Type:    "npm",
					Location: "/app/package.json",
				})
			}
			if strings.Contains(line, "requirements.txt") {
				packages = append(packages, Package{
					Name:    "python-dependencies",
					Version: "unknown",
					Type:    "pip",
					Location: "/app/requirements.txt",
				})
			}
		}
	}

	return packages, nil
}

// extractPackagesFromRunCommand extracts package names from RUN commands.
func (g *Generator) extractPackagesFromRunCommand(line string) []Package {
	var packages []Package

	// Remove "RUN " prefix
	cmd := strings.TrimPrefix(line, "RUN ")

	// Detect package manager and extract packages
	if strings.Contains(cmd, "apt-get install") || strings.Contains(cmd, "apt install") {
		pkgs := g.extractDebPackages(cmd)
		for _, pkg := range pkgs {
			packages = append(packages, Package{
				Name:    pkg,
				Version: "unknown",
				Type:    "deb",
			})
		}
	} else if strings.Contains(cmd, "apk add") {
		pkgs := g.extractApkPackages(cmd)
		for _, pkg := range pkgs {
			packages = append(packages, Package{
				Name:    pkg,
				Version: "unknown",
				Type:    "apk",
			})
		}
	} else if strings.Contains(cmd, "yum install") || strings.Contains(cmd, "dnf install") {
		pkgs := g.extractRpmPackages(cmd)
		for _, pkg := range pkgs {
			packages = append(packages, Package{
				Name:    pkg,
				Version: "unknown",
				Type:    "rpm",
			})
		}
	}

	return packages
}

// extractDebPackages extracts Debian package names from apt-get commands.
func (g *Generator) extractDebPackages(cmd string) []string {
	var packages []string

	// Find the install command and extract package names
	parts := strings.Fields(cmd)
	inPackages := false

	for _, part := range parts {
		if part == "install" {
			inPackages = true
			continue
		}
		if inPackages && !strings.HasPrefix(part, "-") {
			// Clean up package name (remove version specifications)
			pkg := strings.Split(part, "=")[0]
			pkg = strings.TrimSpace(pkg)
			if pkg != "" && pkg != "&&" && pkg != "\\" {
				packages = append(packages, pkg)
			}
		}
	}

	return packages
}

// extractApkPackages extracts Alpine package names from apk commands.
func (g *Generator) extractApkPackages(cmd string) []string {
	var packages []string

	parts := strings.Fields(cmd)
	inPackages := false

	for _, part := range parts {
		if part == "add" {
			inPackages = true
			continue
		}
		if inPackages && !strings.HasPrefix(part, "-") {
			pkg := strings.Split(part, "=")[0]
			pkg = strings.TrimSpace(pkg)
			if pkg != "" && pkg != "&&" && pkg != "\\" {
				packages = append(packages, pkg)
			}
		}
	}

	return packages
}

// extractRpmPackages extracts RPM package names from yum/dnf commands.
func (g *Generator) extractRpmPackages(cmd string) []string {
	var packages []string

	parts := strings.Fields(cmd)
	inPackages := false

	for _, part := range parts {
		if part == "install" {
			inPackages = true
			continue
		}
		if inPackages && !strings.HasPrefix(part, "-") {
			pkg := strings.Split(part, "-")[0]
			pkg = strings.TrimSpace(pkg)
			if pkg != "" && pkg != "&&" && pkg != "\\" && pkg != "-y" {
				packages = append(packages, pkg)
			}
		}
	}

	return packages
}

// parseManifest parses a package manifest file.
func (g *Generator) parseManifest(manifestType, content string) ([]Package, error) {
	manifest, err := g.parser.Parse(manifestType, content)
	if err != nil {
		return nil, fmt.Errorf("parse %s manifest: %w", manifestType, err)
	}

	var packages []Package
	for _, mp := range manifest.Packages {
		pkg := Package{
			Name:    mp.Name,
			Version: mp.Version,
			Type:    manifestType,
			License: mp.License,
		}

		// Generate PURL (Package URL)
		pkg.PURL = generatePURL(manifestType, mp.Name, mp.Version)

		packages = append(packages, pkg)
	}

	return packages, nil
}

// generatePURL generates a Package URL for a package.
func generatePURL(pkgType, name, version string) string {
	var purlType string

	switch pkgType {
	case "npm":
		purlType = packageurl.TypeNPM
	case "pip", "pypi":
		purlType = packageurl.TypePyPi
	case "go", "golang":
		purlType = packageurl.TypeGolang
	case "maven":
		purlType = packageurl.TypeMaven
	case "nuget":
		purlType = packageurl.TypeNuget
	case "deb":
		purlType = packageurl.TypeDebian
	case "rpm":
		purlType = packageurl.TypeRPM
	case "apk":
		purlType = "apk" // Alpine packages
	default:
		purlType = packageurl.TypeGeneric
	}

	purl := packageurl.NewPackageURL(purlType, "", name, version, nil, "")
	return purl.ToString()
}

// getFormatVersion returns the version string for an SBOM format.
func getFormatVersion(format Format) string {
	switch format {
	case FormatSPDX:
		return "SPDX-2.3"
	case FormatCycloneDX:
		return "CycloneDX-1.5"
	default:
		return "unknown"
	}
}

// generateChecksum generates a SHA256 checksum for content.
func generateChecksum(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// EnrichWithVulnerabilities adds vulnerability data to an existing SBOM.
func (g *Generator) EnrichWithVulnerabilities(ctx context.Context, sbomID uuid.UUID) error {
	g.logger.Info("enriching sbom with vulnerabilities", "sbom_id", sbomID)

	// Get SBOM packages
	packages, err := g.svc.GetPackages(ctx, sbomID)
	if err != nil {
		return fmt.Errorf("get packages: %w", err)
	}

	// Get vulnerability scanner
	scanner := NewVulnerabilityScanner(g.logger)

	var allVulns []Vulnerability
	for _, pkg := range packages {
		// Scan for vulnerabilities
		vulns, err := scanner.ScanPackage(ctx, pkg)
		if err != nil {
			g.logger.Warn("failed to scan package",
				"package", pkg.Name,
				"error", err,
			)
			continue
		}

		// Associate vulnerabilities with the package
		for i := range vulns {
			vulns[i].SBOMID = sbomID
			vulns[i].PackageID = pkg.ID
		}

		allVulns = append(allVulns, vulns...)
	}

	// Store vulnerabilities
	if len(allVulns) > 0 {
		if err := g.svc.CreateVulnerabilityBatch(ctx, allVulns); err != nil {
			return fmt.Errorf("store vulnerabilities: %w", err)
		}
	}

	g.logger.Info("sbom enriched with vulnerabilities",
		"sbom_id", sbomID,
		"vulnerabilities", len(allVulns),
	)

	return nil
}
