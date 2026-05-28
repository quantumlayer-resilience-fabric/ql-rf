package sbom

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	severityCritical = "critical"
	severityHigh     = "high"
	severityMedium   = "medium"
	severityLow      = "low"
	severityUnknown  = "unknown"
)

// VulnerabilityScanner scans packages for known vulnerabilities.
type VulnerabilityScanner struct {
	logger *slog.Logger
	client *http.Client
}

// NewVulnerabilityScanner creates a new vulnerability scanner.
func NewVulnerabilityScanner(logger *slog.Logger) *VulnerabilityScanner {
	if logger == nil {
		logger = slog.Default()
	}

	return &VulnerabilityScanner{
		logger: logger.With("component", "vuln-scanner"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ScanPackage scans a package for vulnerabilities using public vulnerability databases.
func (s *VulnerabilityScanner) ScanPackage(ctx context.Context, pkg *Package) ([]Vulnerability, error) {
	var vulns []Vulnerability

	// Try OSV (Open Source Vulnerabilities) database first
	osvVulns, err := s.queryOSV(ctx, pkg)
	if err != nil {
		s.logger.Warn("osv query failed",
			"package", pkg.Name,
			"error", err,
		)
	} else {
		vulns = append(vulns, osvVulns...)
	}

	// Could add more sources here:
	// - NVD (National Vulnerability Database)
	// - GitHub Security Advisories
	// - Snyk database
	// - etc.

	return vulns, nil
}

// osvQueryRequest represents a request to OSV API.
type osvQueryRequest struct {
	Package struct {
		Name      string `json:"name"`
		Ecosystem string `json:"ecosystem"`
	} `json:"package"`
	Version string `json:"version,omitempty"`
}

// osvVulnerability represents a vulnerability from OSV.
type osvVulnerability struct {
	ID        string `json:"id"`
	Summary   string `json:"summary"`
	Details   string `json:"details"`
	Published string `json:"published"`
	Modified  string `json:"modified"`
	Severity  []struct {
		Type  string `json:"type"`
		Score string `json:"score"`
	} `json:"severity"`
	Affected []struct {
		Package struct {
			Name      string `json:"name"`
			Ecosystem string `json:"ecosystem"`
		} `json:"package"`
		Ranges []struct {
			Type   string `json:"type"`
			Events []struct {
				Introduced string `json:"introduced,omitempty"`
				Fixed      string `json:"fixed,omitempty"`
			} `json:"events"`
		} `json:"ranges"`
	} `json:"affected"`
	References []struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"references"`
}

// queryOSV queries the OSV database for vulnerabilities.
func (s *VulnerabilityScanner) queryOSV(ctx context.Context, pkg *Package) ([]Vulnerability, error) {
	ecosystem := mapPackageTypeToOSVEcosystem(pkg.Type)
	if ecosystem == "" {
		// Not supported by OSV
		return nil, nil
	}

	// Build query
	query := osvQueryRequest{}
	query.Package.Name = pkg.Name
	query.Package.Ecosystem = ecosystem
	query.Version = pkg.Version

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("marshal query: %w", err)
	}

	// Query OSV API
	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.osv.dev/v1/query",
		strings.NewReader(string(queryJSON)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query osv: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("osv api returned status %d", resp.StatusCode)
	}

	// Parse response
	var result struct {
		Vulns []osvVulnerability `json:"vulns"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Convert to our Vulnerability type
	vulns := make([]Vulnerability, 0, len(result.Vulns))
	for i := range result.Vulns {
		vulns = append(vulns, convertOSVVuln(&result.Vulns[i]))
	}

	return vulns, nil
}

// convertOSVVuln converts an OSV vulnerability to our internal Vulnerability type.
func convertOSVVuln(osv *osvVulnerability) Vulnerability {
	vuln := Vulnerability{
		CVEID:       osv.ID,
		Description: osv.Summary,
		DataSource:  "OSV",
	}

	// Extract severity and CVSS
	for _, sev := range osv.Severity {
		if sev.Type == "CVSS_V3" {
			vuln.CVSSVector = sev.Score
			score := parseCVSSScore(sev.Score)
			if score > 0 {
				vuln.CVSSScore = &score
				vuln.Severity = cvssScoreToSeverity(score)
			}
		}
	}

	// If no CVSS, set severity based on details
	if vuln.Severity == "" {
		vuln.Severity = severityUnknown
	}

	// Extract fixed version
	vuln.FixedVersion = extractFixedVersion(osv)

	// Extract references
	for _, ref := range osv.References {
		vuln.References = append(vuln.References, ref.URL)
	}

	// Parse dates
	if osv.Published != "" {
		if t, err := time.Parse(time.RFC3339, osv.Published); err == nil {
			vuln.PublishedDate = &t
		}
	}
	if osv.Modified != "" {
		if t, err := time.Parse(time.RFC3339, osv.Modified); err == nil {
			vuln.ModifiedDate = &t
		}
	}

	// Check for known exploits (simplified - would need exploit database)
	vuln.ExploitAvailable = strings.Contains(strings.ToLower(osv.Details), "exploit")

	return vuln
}

// extractFixedVersion extracts the first fixed version from OSV affected ranges.
func extractFixedVersion(osv *osvVulnerability) string {
	for _, affected := range osv.Affected {
		for _, r := range affected.Ranges {
			for _, event := range r.Events {
				if event.Fixed != "" {
					return event.Fixed
				}
			}
		}
	}
	return ""
}

// mapPackageTypeToOSVEcosystem maps package types to OSV ecosystems.
func mapPackageTypeToOSVEcosystem(pkgType string) string {
	switch pkgType {
	case pkgTypeNPM:
		return pkgTypeNPM
	case pkgTypePip, pkgTypePypi:
		return "PyPI"
	case "go", pkgTypeGolang:
		return "Go"
	case pkgTypeMaven:
		return "Maven"
	case pkgTypeNuget:
		return "NuGet"
	case "ruby", "gem":
		return "RubyGems"
	case "cargo", "rust":
		return "crates.io"
	case pkgTypeDeb:
		return "Debian"
	case pkgTypeApk:
		return "Alpine"
	default:
		return ""
	}
}

// parseCVSSScore extracts the numeric score from a CVSS vector string.
func parseCVSSScore(vector string) float64 {
	// Simplified CVSS parsing
	// In production, use a proper CVSS parser library
	// For now, return a placeholder based on severity indicators

	lowerVector := strings.ToLower(vector)

	if strings.Contains(lowerVector, "av:n") && strings.Contains(lowerVector, "ac:l") {
		// Network accessible, low complexity - potentially critical
		if strings.Contains(lowerVector, "c:h") || strings.Contains(lowerVector, "i:h") {
			return 9.0
		}
		return 7.5
	}

	// Default medium
	return 5.0
}

// cvssScoreToSeverity converts a CVSS score to a severity rating.
func cvssScoreToSeverity(score float64) string {
	switch {
	case score >= 9.0:
		return severityCritical
	case score >= 7.0:
		return severityHigh
	case score >= 4.0:
		return severityMedium
	case score > 0.0:
		return severityLow
	default:
		return severityUnknown
	}
}

// EnrichVulnerabilityData enriches vulnerability data with additional context.
func (s *VulnerabilityScanner) EnrichVulnerabilityData(_ context.Context, vuln *Vulnerability) error {
	// Could add enrichment from:
	// - EPSS (Exploit Prediction Scoring System)
	// - KEV (Known Exploited Vulnerabilities) catalog
	// - Vendor advisories
	// - etc.

	// For now, just a placeholder
	s.logger.Debug("enriching vulnerability",
		"cve", vuln.CVEID,
	)

	return nil
}

// GetVulnerabilityStats returns statistics about vulnerabilities in an SBOM.
func (s *Service) GetVulnerabilityStats(ctx context.Context, sbomID uuid.UUID) (map[string]interface{}, error) {
	vulns, err := s.GetVulnerabilities(ctx, sbomID, nil)
	if err != nil {
		return nil, fmt.Errorf("get vulnerabilities: %w", err)
	}

	var (
		cntCritical     int
		cntHigh         int
		cntMedium       int
		cntLow          int
		cntUnknown      int
		cntWithExploits int
		cntWithFixes    int
		totalScore      float64
		highestCVSS     float64
		countWithScore  int
	)

	for i := range vulns {
		// Count by severity
		switch vulns[i].Severity {
		case severityCritical:
			cntCritical++
		case severityHigh:
			cntHigh++
		case severityMedium:
			cntMedium++
		case severityLow:
			cntLow++
		default:
			cntUnknown++
		}

		// Count exploits
		if vulns[i].ExploitAvailable {
			cntWithExploits++
		}

		// Count fixes
		if vulns[i].FixedVersion != "" {
			cntWithFixes++
		}

		// Calculate CVSS stats
		if vulns[i].CVSSScore != nil {
			score := *vulns[i].CVSSScore
			totalScore += score
			countWithScore++

			if score > highestCVSS {
				highestCVSS = score
			}
		}
	}

	avgCVSS := 0.0
	if countWithScore > 0 {
		avgCVSS = totalScore / float64(countWithScore)
	}

	return map[string]interface{}{
		"total":              len(vulns),
		severityCritical:     cntCritical,
		severityHigh:         cntHigh,
		severityMedium:       cntMedium,
		severityLow:          cntLow,
		severityUnknown:      cntUnknown,
		"with_exploits":      cntWithExploits,
		"with_fixes":         cntWithFixes,
		"avg_cvss_score":     avgCVSS,
		"highest_cvss_score": highestCVSS,
	}, nil
}
