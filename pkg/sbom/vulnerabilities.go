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
func (s *VulnerabilityScanner) ScanPackage(ctx context.Context, pkg Package) ([]Vulnerability, error) {
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
func (s *VulnerabilityScanner) queryOSV(ctx context.Context, pkg Package) ([]Vulnerability, error) {
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
	var vulns []Vulnerability
	for _, osv := range result.Vulns {
		vuln := Vulnerability{
			CVEID:       osv.ID,
			Description: osv.Summary,
			DataSource:  "OSV",
		}

		// Extract severity and CVSS
		for _, sev := range osv.Severity {
			if sev.Type == "CVSS_V3" {
				vuln.CVSSVector = sev.Score
				// Parse CVSS score from vector
				score := parseCVSSScore(sev.Score)
				if score > 0 {
					vuln.CVSSScore = &score
					vuln.Severity = cvssScoreToSeverity(score)
				}
			}
		}

		// If no CVSS, set severity based on details
		if vuln.Severity == "" {
			vuln.Severity = "unknown"
		}

		// Extract fixed version
		for _, affected := range osv.Affected {
			for _, r := range affected.Ranges {
				for _, event := range r.Events {
					if event.Fixed != "" {
						vuln.FixedVersion = event.Fixed
						break
					}
				}
			}
		}

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

		vulns = append(vulns, vuln)
	}

	return vulns, nil
}

// mapPackageTypeToOSVEcosystem maps package types to OSV ecosystems.
func mapPackageTypeToOSVEcosystem(pkgType string) string {
	switch pkgType {
	case "npm":
		return "npm"
	case "pip", "pypi":
		return "PyPI"
	case "go", "golang":
		return "Go"
	case "maven":
		return "Maven"
	case "nuget":
		return "NuGet"
	case "ruby", "gem":
		return "RubyGems"
	case "cargo", "rust":
		return "crates.io"
	case "deb":
		return "Debian"
	case "apk":
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
		return "critical"
	case score >= 7.0:
		return "high"
	case score >= 4.0:
		return "medium"
	case score > 0.0:
		return "low"
	default:
		return "unknown"
	}
}

// EnrichVulnerabilityData enriches vulnerability data with additional context.
func (s *VulnerabilityScanner) EnrichVulnerabilityData(ctx context.Context, vuln *Vulnerability) error {
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

	stats := map[string]interface{}{
		"total":              len(vulns),
		"critical":           0,
		"high":               0,
		"medium":             0,
		"low":                0,
		"unknown":            0,
		"with_exploits":      0,
		"with_fixes":         0,
		"avg_cvss_score":     0.0,
		"highest_cvss_score": 0.0,
	}

	totalScore := 0.0
	countWithScore := 0

	for _, vuln := range vulns {
		// Count by severity
		switch vuln.Severity {
		case "critical":
			stats["critical"] = stats["critical"].(int) + 1
		case "high":
			stats["high"] = stats["high"].(int) + 1
		case "medium":
			stats["medium"] = stats["medium"].(int) + 1
		case "low":
			stats["low"] = stats["low"].(int) + 1
		default:
			stats["unknown"] = stats["unknown"].(int) + 1
		}

		// Count exploits
		if vuln.ExploitAvailable {
			stats["with_exploits"] = stats["with_exploits"].(int) + 1
		}

		// Count fixes
		if vuln.FixedVersion != "" {
			stats["with_fixes"] = stats["with_fixes"].(int) + 1
		}

		// Calculate CVSS stats
		if vuln.CVSSScore != nil {
			score := *vuln.CVSSScore
			totalScore += score
			countWithScore++

			if score > stats["highest_cvss_score"].(float64) {
				stats["highest_cvss_score"] = score
			}
		}
	}

	if countWithScore > 0 {
		stats["avg_cvss_score"] = totalScore / float64(countWithScore)
	}

	return stats, nil
}
