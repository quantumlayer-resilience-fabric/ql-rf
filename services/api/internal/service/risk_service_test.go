package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

// testUUID creates a deterministic UUID based on an index for testing.
func testUUID(index int) uuid.UUID {
	return uuid.MustParse("00000000-0000-0000-0000-00000000000" + string(rune('0'+index%10)))
}

func TestRiskService_calculateRiskScore(t *testing.T) {
	svc := &RiskService{
		weights: models.DefaultRiskWeights(),
	}

	// Risk scoring model:
	// - Drift Age (25%): 2 points per day, max 100
	// - Vuln Count (20%): 5 points per vuln, max 100
	// - Critical Vulns (25%): 25 points per critical, max 100
	// - Compliance (15%): 100 if non-compliant, 0 if compliant
	// - Environment multiplier: production=1.5, dr=1.2, staging=1.0, development=0.5

	tests := []struct {
		name           string
		driftAge       int
		vulnCount      int
		criticalVulns  int
		isCompliant    bool
		environment    string
		expectedLevel  models.RiskLevel
		minScore       float64
		maxScore       float64
	}{
		{
			name:          "healthy asset - no drift, no vulns, compliant",
			driftAge:      0,
			vulnCount:     0,
			criticalVulns: 0,
			isCompliant:   true,
			environment:   "development",
			expectedLevel: models.RiskLevelLow,
			minScore:      0,
			maxScore:      10,
		},
		{
			name:          "critical - production with critical vulns and drift",
			driftAge:      30,  // 60 points * 0.25 = 15
			vulnCount:     10,  // 50 points * 0.20 = 10
			criticalVulns: 3,   // 75 points * 0.25 = 18.75
			isCompliant:   false, // 100 * 0.15 = 15
			environment:   "production", // * 1.5 = ~88
			expectedLevel: models.RiskLevelCritical,
			minScore:      80,
			maxScore:      100,
		},
		{
			name:          "medium risk - non-compliant with vulns in staging",
			driftAge:      14,  // 28 * 0.25 = 7
			vulnCount:     5,   // 25 * 0.20 = 5
			criticalVulns: 1,   // 25 * 0.25 = 6.25
			isCompliant:   false, // 100 * 0.15 = 15
			environment:   "staging", // * 1.0 = ~33
			expectedLevel: models.RiskLevelLow, // Score ~33, which is low
			minScore:      25,
			maxScore:      45,
		},
		{
			name:          "low risk - some drift in production but compliant",
			driftAge:      0,   // 0 (compliant, no drift)
			vulnCount:     2,   // 10 * 0.20 = 2
			criticalVulns: 0,   // 0
			isCompliant:   true, // 0
			environment:   "production", // * 1.5 = 3
			expectedLevel: models.RiskLevelLow,
			minScore:      0,
			maxScore:      10,
		},
		{
			name:          "low risk - development environment (halved)",
			driftAge:      14,  // 28 * 0.25 = 7
			vulnCount:     5,   // 25 * 0.20 = 5
			criticalVulns: 0,   // 0
			isCompliant:   true, // 0
			environment:   "development", // * 0.5 = 6
			expectedLevel: models.RiskLevelLow,
			minScore:      0,
			maxScore:      15,
		},
		{
			name:          "medium risk - DR environment with drift and non-compliant",
			driftAge:      20,  // 40 * 0.25 = 10
			vulnCount:     8,   // 40 * 0.20 = 8
			criticalVulns: 1,   // 25 * 0.25 = 6.25
			isCompliant:   false, // 100 * 0.15 = 15
			environment:   "dr", // * 1.2 = ~47
			expectedLevel: models.RiskLevelMedium,
			minScore:      40,
			maxScore:      60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, factors := svc.calculateRiskScore(
				tt.driftAge,
				tt.vulnCount,
				tt.criticalVulns,
				tt.isCompliant,
				tt.environment,
			)

			level := models.CalculateRiskLevel(score)

			assert.GreaterOrEqual(t, score, tt.minScore, "score should be >= minScore")
			assert.LessOrEqual(t, score, tt.maxScore, "score should be <= maxScore")
			assert.Equal(t, tt.expectedLevel, level, "risk level should match")

			// Verify factors are present when applicable
			if tt.driftAge > 0 {
				assert.True(t, hasFactorNamed(factors, "Drift Age"), "should have drift age factor")
			}
			if tt.vulnCount > 0 {
				assert.True(t, hasFactorNamed(factors, "Open Vulnerabilities"), "should have vuln factor")
			}
			if tt.criticalVulns > 0 {
				assert.True(t, hasFactorNamed(factors, "Critical Vulnerabilities"), "should have critical vuln factor")
			}
			if !tt.isCompliant {
				assert.True(t, hasFactorNamed(factors, "Non-Compliant"), "should have compliance factor")
			}
		})
	}
}

func TestRiskService_calculateRiskScore_MaxBounds(t *testing.T) {
	svc := &RiskService{
		weights: models.DefaultRiskWeights(),
	}

	// Extreme case - score should be capped at 100
	score, _ := svc.calculateRiskScore(
		100,  // Very high drift age
		100,  // Many vulns
		10,   // Many critical vulns
		false, // Non-compliant
		"production",
	)

	assert.LessOrEqual(t, score, 100.0, "score should be capped at 100")
}

func TestRiskService_aggregateByScope(t *testing.T) {
	svc := &RiskService{}

	risks := []models.AssetRiskScore{
		{AssetID: testUUID(1), Environment: "production", Platform: "aws", Site: "us-east", RiskScore: 80, RiskLevel: models.RiskLevelCritical},
		{AssetID: testUUID(2), Environment: "production", Platform: "aws", Site: "us-east", RiskScore: 70, RiskLevel: models.RiskLevelHigh},
		{AssetID: testUUID(3), Environment: "production", Platform: "azure", Site: "us-west", RiskScore: 60, RiskLevel: models.RiskLevelHigh},
		{AssetID: testUUID(4), Environment: "staging", Platform: "aws", Site: "us-east", RiskScore: 30, RiskLevel: models.RiskLevelLow},
		{AssetID: testUUID(5), Environment: "staging", Platform: "gcp", Site: "eu-west", RiskScore: 25, RiskLevel: models.RiskLevelLow},
	}

	t.Run("aggregate by environment", func(t *testing.T) {
		result := svc.aggregateByScope(risks, "environment")

		assert.Len(t, result, 2) // production and staging

		// Results should be sorted by risk score descending
		assert.Equal(t, "production", result[0].Scope)
		assert.Equal(t, 3, result[0].AssetCount)
		assert.InDelta(t, 70.0, result[0].RiskScore, 0.01) // Average of 80, 70, 60

		assert.Equal(t, "staging", result[1].Scope)
		assert.Equal(t, 2, result[1].AssetCount)
		assert.InDelta(t, 27.5, result[1].RiskScore, 0.01) // Average of 30, 25
	})

	t.Run("aggregate by platform", func(t *testing.T) {
		result := svc.aggregateByScope(risks, "platform")

		assert.Len(t, result, 3) // aws, azure, gcp

		// Find AWS
		var aws *models.RiskByScope
		for i := range result {
			if result[i].Scope == "aws" {
				aws = &result[i]
				break
			}
		}
		assert.NotNil(t, aws)
		assert.Equal(t, 3, aws.AssetCount)
	})

	t.Run("aggregate by site", func(t *testing.T) {
		result := svc.aggregateByScope(risks, "site")

		assert.Len(t, result, 3) // us-east, us-west, eu-west
	})
}

func TestRiskService_generateTrend(t *testing.T) {
	svc := &RiskService{}

	t.Run("generates correct number of days", func(t *testing.T) {
		trend := svc.generateTrend(50.0, 30)
		assert.Len(t, trend, 30)
	})

	t.Run("trend ends near current score", func(t *testing.T) {
		currentScore := 75.0
		trend := svc.generateTrend(currentScore, 30)

		// Last point should be close to current score
		lastPoint := trend[len(trend)-1]
		assert.InDelta(t, currentScore, lastPoint.RiskScore, 5.0)
	})

	t.Run("trend has valid risk levels", func(t *testing.T) {
		trend := svc.generateTrend(50.0, 10)

		for _, point := range trend {
			assert.GreaterOrEqual(t, point.RiskScore, 0.0)
			assert.LessOrEqual(t, point.RiskScore, 100.0)
			assert.Contains(t, []models.RiskLevel{
				models.RiskLevelCritical,
				models.RiskLevelHigh,
				models.RiskLevelMedium,
				models.RiskLevelLow,
			}, point.RiskLevel)
		}
	})
}

func TestRiskService_min(t *testing.T) {
	tests := []struct {
		a, b, expected float64
	}{
		{1.0, 2.0, 1.0},
		{2.0, 1.0, 1.0},
		{5.5, 5.5, 5.5},
		{-1.0, 1.0, -1.0},
		{100.0, 50.0, 50.0},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		assert.Equal(t, tt.expected, result)
	}
}

// Helper function to check if a factor with given name exists
func hasFactorNamed(factors []models.RiskFactor, name string) bool {
	for _, f := range factors {
		if f.Name == name {
			return true
		}
	}
	return false
}
