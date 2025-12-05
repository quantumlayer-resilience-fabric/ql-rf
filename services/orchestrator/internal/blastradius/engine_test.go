package blastradius

import (
	"testing"

	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

func TestCalculateUrgencyScore(t *testing.T) {
	scorer := NewScorer(nil)

	tests := []struct {
		name     string
		input    ScoringInput
		wantMin  int
		wantMax  int
		priority string
	}{
		{
			name: "Critical CVE with exploit and KEV",
			input: ScoringInput{
				CVSSScore:        9.8,
				EPSSScore:        0.9,
				ExploitAvailable: true,
				CISAKEVListed:    true,
				ProductionAssets: 10,
				TotalAssets:      50,
				TotalFleetAssets: 100,
			},
			wantMin:  80,
			wantMax:  100,
			priority: "p1",
		},
		{
			name: "High CVE without exploit",
			input: ScoringInput{
				CVSSScore:        8.0,
				EPSSScore:        0.1,
				ExploitAvailable: false,
				CISAKEVListed:    false,
				ProductionAssets: 5,
				TotalAssets:      20,
				TotalFleetAssets: 100,
			},
			wantMin:  40,
			wantMax:  70,
			priority: "p3",
		},
		{
			name: "Medium CVE",
			input: ScoringInput{
				CVSSScore:        5.0,
				EPSSScore:        0.05,
				ExploitAvailable: false,
				CISAKEVListed:    false,
				ProductionAssets: 0,
				TotalAssets:      5,
				TotalFleetAssets: 100,
			},
			wantMin:  20,
			wantMax:  40,
			priority: "p4",
		},
		{
			name: "Low CVE",
			input: ScoringInput{
				CVSSScore:        2.0,
				EPSSScore:        0.01,
				ExploitAvailable: false,
				CISAKEVListed:    false,
				ProductionAssets: 0,
				TotalAssets:      1,
				TotalFleetAssets: 100,
			},
			wantMin:  0,
			wantMax:  20,
			priority: "p4",
		},
		{
			name: "KEV with production = P1",
			input: ScoringInput{
				CVSSScore:        7.0,
				EPSSScore:        0.3,
				ExploitAvailable: false,
				CISAKEVListed:    true,
				ProductionAssets: 1,
				TotalAssets:      5,
				TotalFleetAssets: 100,
			},
			wantMin:  50,
			wantMax:  80,
			priority: "p1", // KEV + prod = P1 regardless of score
		},
		{
			name: "Exploit with CVSS 9+ = P1",
			input: ScoringInput{
				CVSSScore:        9.0,
				EPSSScore:        0.5,
				ExploitAvailable: true,
				CISAKEVListed:    false,
				ProductionAssets: 0,
				TotalAssets:      5,
				TotalFleetAssets: 100,
			},
			wantMin:  60,
			wantMax:  90,
			priority: "p1", // Exploit + CVSS 9+ = P1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scorer.Calculate(tt.input)

			if result.UrgencyScore < tt.wantMin || result.UrgencyScore > tt.wantMax {
				t.Errorf("UrgencyScore = %d, want between %d and %d", result.UrgencyScore, tt.wantMin, tt.wantMax)
			}

			if result.Priority != tt.priority {
				t.Errorf("Priority = %s, want %s", result.Priority, tt.priority)
			}
		})
	}
}

func TestSeverityFromScore(t *testing.T) {
	tests := []struct {
		score    int
		expected string
	}{
		{100, "critical"},
		{80, "critical"},
		{79, "high"},
		{60, "high"},
		{59, "medium"},
		{40, "medium"},
		{39, "low"},
		{0, "low"},
	}

	for _, tt := range tests {
		got := SeverityFromScore(tt.score)
		if got != tt.expected {
			t.Errorf("SeverityFromScore(%d) = %s, want %s", tt.score, got, tt.expected)
		}
	}
}

func TestScoreBreakdown(t *testing.T) {
	scorer := NewScorer(nil)

	input := ScoringInput{
		CVSSScore:        9.0,
		EPSSScore:        0.5,
		ExploitAvailable: true,
		CISAKEVListed:    true,
		ProductionAssets: 10,
		TotalAssets:      50,
		TotalFleetAssets: 100,
	}

	result := scorer.Calculate(input)
	breakdown := result.ScoreBreakdown

	// Verify individual contributions
	if breakdown.CVSSContribution != 90.0 { // 9.0 * 10
		t.Errorf("CVSSContribution = %.1f, want 90.0", breakdown.CVSSContribution)
	}

	if breakdown.ExploitContribution != 25.0 {
		t.Errorf("ExploitContribution = %.1f, want 25.0", breakdown.ExploitContribution)
	}

	if breakdown.CISAKEVContribution != 20.0 {
		t.Errorf("CISAKEVContribution = %.1f, want 20.0", breakdown.CISAKEVContribution)
	}

	if breakdown.ProductionContribution != 15.0 {
		t.Errorf("ProductionContribution = %.1f, want 15.0", breakdown.ProductionContribution)
	}

	// Fleet contribution should be capped at 10
	if breakdown.FleetContribution > 10.0 {
		t.Errorf("FleetContribution = %.1f, should not exceed 10.0", breakdown.FleetContribution)
	}

	if breakdown.EPSSContribution != 10.0 { // 0.5 * 20
		t.Errorf("EPSSContribution = %.1f, want 10.0", breakdown.EPSSContribution)
	}
}

func TestAssessRisk(t *testing.T) {
	scorer := NewScorer(nil)

	tests := []struct {
		name           string
		input          ScoringInput
		expectedLevel  RiskLevel
		minFactors     int
	}{
		{
			name: "Critical risk",
			input: ScoringInput{
				CVSSScore:        9.8,
				EPSSScore:        0.5,
				ExploitAvailable: true,
				CISAKEVListed:    true,
				ProductionAssets: 10,
			},
			expectedLevel: RiskLevelCritical,
			minFactors:    4, // CVSS, Exploit, KEV, Production
		},
		{
			name: "High risk",
			input: ScoringInput{
				CVSSScore:        8.0,
				EPSSScore:        0.3,
				ExploitAvailable: true,
				ProductionAssets: 5,
			},
			expectedLevel: RiskLevelHigh,
			minFactors:    2, // CVSS, Exploit
		},
		{
			name: "Low risk",
			input: ScoringInput{
				CVSSScore: 4.0,
				EPSSScore: 0.05,
			},
			expectedLevel: RiskLevelLow,
			minFactors:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment := scorer.AssessRisk(tt.input)

			if assessment.Level != tt.expectedLevel {
				t.Errorf("Level = %s, want %s", assessment.Level, tt.expectedLevel)
			}

			if len(assessment.Factors) < tt.minFactors {
				t.Errorf("Factors count = %d, want at least %d", len(assessment.Factors), tt.minFactors)
			}

			if assessment.Recommendation == "" {
				t.Error("Recommendation should not be empty")
			}
		})
	}
}

func TestScoreFromCVEAlert(t *testing.T) {
	scorer := NewScorer(nil)

	cvss := 8.5
	epss := 0.3
	cve := &models.CVECache{
		CVSSV3Score:      &cvss,
		EPSSScore:        &epss,
		ExploitAvailable: true,
		CISAKEVListed:    false,
	}

	alert := &models.CVEAlert{
		AffectedPackagesCount: 5,
		AffectedImagesCount:   3,
		AffectedAssetsCount:   10,
		ProductionAssetsCount: 2,
	}

	result := scorer.ScoreFromCVEAlert(alert, cve, nil, 100)

	if result.UrgencyScore < 50 || result.UrgencyScore > 100 {
		t.Errorf("UrgencyScore = %d, expected between 50 and 100", result.UrgencyScore)
	}

	// Should be P2 (exploit available with production assets)
	if result.Priority != "p2" {
		t.Errorf("Priority = %s, expected p2", result.Priority)
	}
}

func TestCustomScoringConfig(t *testing.T) {
	config := &ScoringConfig{
		CVSSWeight:          5,  // Half the default
		ExploitBonus:        50, // Double the default
		CISAKEVBonus:        10,
		ProductionBonus:     5,
		FleetFactorMax:      5,
		EPSSWeight:          10,
		NormalizationFactor: 1.5,
		CriticalSLAHours:    12,
		HighSLAHours:        48,
		MediumSLAHours:      96,
		LowSLAHours:         480,
	}

	scorer := NewScorer(config)

	input := ScoringInput{
		CVSSScore:        10.0,
		ExploitAvailable: true,
	}

	result := scorer.Calculate(input)

	// With custom config:
	// CVSS: 10 * 5 = 50
	// Exploit: 50
	// Raw: 100
	// Normalized: 100 / 1.5 = 66

	if result.UrgencyScore < 60 || result.UrgencyScore > 70 {
		t.Errorf("UrgencyScore with custom config = %d, expected around 66", result.UrgencyScore)
	}
}
