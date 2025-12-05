package blastradius

import (
	"fmt"
	"time"

	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

// ScoringConfig contains configuration for urgency scoring.
type ScoringConfig struct {
	// Weight multipliers
	CVSSWeight          float64 // Weight for CVSS score (default: 10)
	ExploitBonus        float64 // Bonus for exploit availability (default: 25)
	CISAKEVBonus        float64 // Bonus for CISA KEV listing (default: 20)
	ProductionBonus     float64 // Bonus for production assets (default: 15)
	FleetFactorMax      float64 // Maximum fleet factor contribution (default: 10)
	EPSSWeight          float64 // Weight for EPSS score (default: 20)

	// Normalization divisor
	NormalizationFactor float64 // Divisor to normalize score (default: 2)

	// SLA thresholds
	CriticalSLAHours    int // Hours until SLA for critical (default: 24)
	HighSLAHours        int // Hours until SLA for high (default: 72)
	MediumSLAHours      int // Hours until SLA for medium (default: 168)
	LowSLAHours         int // Hours until SLA for low (default: 720)
}

// DefaultScoringConfig returns the default scoring configuration.
func DefaultScoringConfig() *ScoringConfig {
	return &ScoringConfig{
		CVSSWeight:          10,
		ExploitBonus:        25,
		CISAKEVBonus:        20,
		ProductionBonus:     15,
		FleetFactorMax:      10,
		EPSSWeight:          20,
		NormalizationFactor: 2,
		CriticalSLAHours:    24,
		HighSLAHours:        72,
		MediumSLAHours:      168,
		LowSLAHours:         720,
	}
}

// Scorer calculates urgency scores and SLA deadlines for CVE alerts.
type Scorer struct {
	config *ScoringConfig
}

// NewScorer creates a new scorer with the given configuration.
func NewScorer(config *ScoringConfig) *Scorer {
	if config == nil {
		config = DefaultScoringConfig()
	}
	return &Scorer{config: config}
}

// ScoringInput contains all inputs for urgency scoring.
type ScoringInput struct {
	// CVE characteristics
	CVSSScore        float64
	EPSSScore        float64
	ExploitAvailable bool
	CISAKEVListed    bool

	// Blast radius
	TotalPackages    int
	TotalImages      int
	TotalAssets      int
	ProductionAssets int

	// Fleet context
	TotalFleetAssets int
}

// ScoreResult contains the calculated score and related metadata.
type ScoreResult struct {
	UrgencyScore      int            `json:"urgencyScore"`
	Priority          string         `json:"priority"` // p1, p2, p3, p4
	SLADueAt          time.Time      `json:"slaDueAt"`
	ScoreBreakdown    ScoreBreakdown `json:"scoreBreakdown"`
}

// ScoreBreakdown shows the contribution of each factor to the final score.
type ScoreBreakdown struct {
	CVSSContribution       float64 `json:"cvssContribution"`
	ExploitContribution    float64 `json:"exploitContribution"`
	CISAKEVContribution    float64 `json:"cisaKevContribution"`
	ProductionContribution float64 `json:"productionContribution"`
	FleetContribution      float64 `json:"fleetContribution"`
	EPSSContribution       float64 `json:"epssContribution"`
	RawScore               float64 `json:"rawScore"`
	NormalizedScore        int     `json:"normalizedScore"`
}

// Calculate computes the urgency score and related outputs.
func (s *Scorer) Calculate(input ScoringInput) *ScoreResult {
	breakdown := ScoreBreakdown{}

	// CVSS contribution (0-100 points raw)
	breakdown.CVSSContribution = input.CVSSScore * s.config.CVSSWeight

	// Exploit availability bonus
	if input.ExploitAvailable {
		breakdown.ExploitContribution = s.config.ExploitBonus
	}

	// CISA KEV bonus
	if input.CISAKEVListed {
		breakdown.CISAKEVContribution = s.config.CISAKEVBonus
	}

	// Production assets bonus
	if input.ProductionAssets > 0 {
		breakdown.ProductionContribution = s.config.ProductionBonus
	}

	// Fleet factor (percentage of fleet affected)
	if input.TotalFleetAssets > 0 && input.TotalAssets > 0 {
		fleetPercentage := float64(input.TotalAssets) / float64(input.TotalFleetAssets) * 100
		breakdown.FleetContribution = min(fleetPercentage/10, s.config.FleetFactorMax)
	}

	// EPSS contribution
	breakdown.EPSSContribution = input.EPSSScore * s.config.EPSSWeight

	// Calculate raw score
	breakdown.RawScore = breakdown.CVSSContribution +
		breakdown.ExploitContribution +
		breakdown.CISAKEVContribution +
		breakdown.ProductionContribution +
		breakdown.FleetContribution +
		breakdown.EPSSContribution

	// Normalize to 0-100
	normalized := int(breakdown.RawScore / s.config.NormalizationFactor)
	if normalized > 100 {
		normalized = 100
	}
	if normalized < 0 {
		normalized = 0
	}
	breakdown.NormalizedScore = normalized

	// Determine priority and SLA
	priority := s.determinePriority(normalized, input)
	slaDueAt := s.calculateSLA(priority)

	return &ScoreResult{
		UrgencyScore:   normalized,
		Priority:       priority,
		SLADueAt:       slaDueAt,
		ScoreBreakdown: breakdown,
	}
}

// determinePriority assigns a priority based on score and other factors.
func (s *Scorer) determinePriority(score int, input ScoringInput) string {
	// P1: Critical - Immediate response required
	// - Score >= 80, OR
	// - CISA KEV with production assets, OR
	// - Exploit available with CVSS >= 9
	if score >= 80 {
		return "p1"
	}
	if input.CISAKEVListed && input.ProductionAssets > 0 {
		return "p1"
	}
	if input.ExploitAvailable && input.CVSSScore >= 9 {
		return "p1"
	}

	// P2: High - Same day response
	// - Score >= 60, OR
	// - Exploit available with production assets
	if score >= 60 {
		return "p2"
	}
	if input.ExploitAvailable && input.ProductionAssets > 0 {
		return "p2"
	}

	// P3: Medium - Within SLA
	// - Score >= 40
	if score >= 40 {
		return "p3"
	}

	// P4: Low - Scheduled maintenance
	return "p4"
}

// calculateSLA calculates the SLA due date based on priority.
func (s *Scorer) calculateSLA(priority string) time.Time {
	now := time.Now().UTC()

	switch priority {
	case "p1":
		return now.Add(time.Duration(s.config.CriticalSLAHours) * time.Hour)
	case "p2":
		return now.Add(time.Duration(s.config.HighSLAHours) * time.Hour)
	case "p3":
		return now.Add(time.Duration(s.config.MediumSLAHours) * time.Hour)
	case "p4":
		return now.Add(time.Duration(s.config.LowSLAHours) * time.Hour)
	default:
		return now.Add(time.Duration(s.config.MediumSLAHours) * time.Hour)
	}
}

// ScoreFromCVEAlert calculates score from a CVE alert and its details.
func (s *Scorer) ScoreFromCVEAlert(alert *models.CVEAlert, cve *models.CVECache, blastRadius *models.BlastRadiusResult, totalFleetAssets int) *ScoreResult {
	input := ScoringInput{
		TotalFleetAssets: totalFleetAssets,
	}

	// Extract CVE characteristics
	if cve != nil {
		if cve.CVSSV3Score != nil {
			input.CVSSScore = *cve.CVSSV3Score
		}
		if cve.EPSSScore != nil {
			input.EPSSScore = *cve.EPSSScore
		}
		input.ExploitAvailable = cve.ExploitAvailable
		input.CISAKEVListed = cve.CISAKEVListed
	}

	// Extract blast radius
	if blastRadius != nil {
		input.TotalPackages = blastRadius.TotalPackages
		input.TotalImages = blastRadius.TotalImages
		input.TotalAssets = blastRadius.TotalAssets
		input.ProductionAssets = blastRadius.ProductionAssets
	} else if alert != nil {
		input.TotalPackages = alert.AffectedPackagesCount
		input.TotalImages = alert.AffectedImagesCount
		input.TotalAssets = alert.AffectedAssetsCount
		input.ProductionAssets = alert.ProductionAssetsCount
	}

	return s.Calculate(input)
}

// SeverityFromScore determines severity from urgency score.
func SeverityFromScore(score int) string {
	switch {
	case score >= 80:
		return "critical"
	case score >= 60:
		return "high"
	case score >= 40:
		return "medium"
	default:
		return "low"
	}
}

// RiskLevel represents a qualitative risk assessment.
type RiskLevel string

const (
	RiskLevelCritical RiskLevel = "critical"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelLow      RiskLevel = "low"
	RiskLevelInfo     RiskLevel = "info"
)

// RiskAssessment provides a comprehensive risk evaluation.
type RiskAssessment struct {
	Level           RiskLevel     `json:"level"`
	Score           int           `json:"score"`
	Factors         []RiskFactor  `json:"factors"`
	Recommendation  string        `json:"recommendation"`
	TimeToRemediate time.Duration `json:"timeToRemediate"`
}

// RiskFactor represents a factor contributing to risk.
type RiskFactor struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Impact      string `json:"impact"` // high, medium, low
	Description string `json:"description"`
}

// AssessRisk provides a comprehensive risk assessment.
func (s *Scorer) AssessRisk(input ScoringInput) *RiskAssessment {
	result := s.Calculate(input)

	assessment := &RiskAssessment{
		Score: result.UrgencyScore,
	}

	// Determine risk level
	switch {
	case result.UrgencyScore >= 80:
		assessment.Level = RiskLevelCritical
		assessment.Recommendation = "Immediate remediation required. Consider emergency change process."
		assessment.TimeToRemediate = 24 * time.Hour
	case result.UrgencyScore >= 60:
		assessment.Level = RiskLevelHigh
		assessment.Recommendation = "High priority remediation. Schedule within same business day."
		assessment.TimeToRemediate = 72 * time.Hour
	case result.UrgencyScore >= 40:
		assessment.Level = RiskLevelMedium
		assessment.Recommendation = "Schedule remediation within next maintenance window."
		assessment.TimeToRemediate = 168 * time.Hour
	case result.UrgencyScore >= 20:
		assessment.Level = RiskLevelLow
		assessment.Recommendation = "Include in regular patching cycle."
		assessment.TimeToRemediate = 720 * time.Hour
	default:
		assessment.Level = RiskLevelInfo
		assessment.Recommendation = "Monitor and address during routine maintenance."
		assessment.TimeToRemediate = 720 * time.Hour
	}

	// Add contributing factors
	if input.CVSSScore >= 9.0 {
		assessment.Factors = append(assessment.Factors, RiskFactor{
			Name:        "CVSS Score",
			Value:       fmt.Sprintf("%.1f", input.CVSSScore),
			Impact:      "high",
			Description: "Critical CVSS score indicates severe vulnerability",
		})
	} else if input.CVSSScore >= 7.0 {
		assessment.Factors = append(assessment.Factors, RiskFactor{
			Name:        "CVSS Score",
			Value:       fmt.Sprintf("%.1f", input.CVSSScore),
			Impact:      "high",
			Description: "High CVSS score indicates significant vulnerability",
		})
	}

	if input.ExploitAvailable {
		assessment.Factors = append(assessment.Factors, RiskFactor{
			Name:        "Exploit Available",
			Value:       "Yes",
			Impact:      "high",
			Description: "Active exploitation in the wild increases risk significantly",
		})
	}

	if input.CISAKEVListed {
		assessment.Factors = append(assessment.Factors, RiskFactor{
			Name:        "CISA KEV Listed",
			Value:       "Yes",
			Impact:      "high",
			Description: "Listed in CISA Known Exploited Vulnerabilities catalog",
		})
	}

	if input.ProductionAssets > 0 {
		assessment.Factors = append(assessment.Factors, RiskFactor{
			Name:        "Production Impact",
			Value:       fmt.Sprintf("%d assets", input.ProductionAssets),
			Impact:      "high",
			Description: "Production systems are affected",
		})
	}

	if input.TotalFleetAssets > 0 {
		pct := float64(input.TotalAssets) / float64(input.TotalFleetAssets) * 100
		if pct > 50 {
			assessment.Factors = append(assessment.Factors, RiskFactor{
				Name:        "Fleet Coverage",
				Value:       fmt.Sprintf("%.1f%%", pct),
				Impact:      "high",
				Description: "Majority of fleet is affected",
			})
		} else if pct > 20 {
			assessment.Factors = append(assessment.Factors, RiskFactor{
				Name:        "Fleet Coverage",
				Value:       fmt.Sprintf("%.1f%%", pct),
				Impact:      "medium",
				Description: "Significant portion of fleet is affected",
			})
		}
	}

	if input.EPSSScore >= 0.5 {
		assessment.Factors = append(assessment.Factors, RiskFactor{
			Name:        "EPSS Score",
			Value:       fmt.Sprintf("%.1f%%", input.EPSSScore*100),
			Impact:      "high",
			Description: "High probability of exploitation in next 30 days",
		})
	} else if input.EPSSScore >= 0.1 {
		assessment.Factors = append(assessment.Factors, RiskFactor{
			Name:        "EPSS Score",
			Value:       fmt.Sprintf("%.1f%%", input.EPSSScore*100),
			Impact:      "medium",
			Description: "Moderate probability of exploitation",
		})
	}

	return assessment
}
