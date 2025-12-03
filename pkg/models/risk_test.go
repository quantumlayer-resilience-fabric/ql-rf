package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateRiskLevel(t *testing.T) {
	tests := []struct {
		name     string
		score    float64
		expected RiskLevel
	}{
		{"critical - 100", 100, RiskLevelCritical},
		{"critical - 80", 80, RiskLevelCritical},
		{"critical - 95.5", 95.5, RiskLevelCritical},
		{"high - 79", 79, RiskLevelHigh},
		{"high - 60", 60, RiskLevelHigh},
		{"high - 70.5", 70.5, RiskLevelHigh},
		{"medium - 59", 59, RiskLevelMedium},
		{"medium - 40", 40, RiskLevelMedium},
		{"medium - 50", 50, RiskLevelMedium},
		{"low - 39", 39, RiskLevelLow},
		{"low - 0", 0, RiskLevelLow},
		{"low - 20", 20, RiskLevelLow},
		{"low - negative", -5, RiskLevelLow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateRiskLevel(tt.score)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnvironmentRiskMultiplier(t *testing.T) {
	tests := []struct {
		environment string
		expected    float64
	}{
		{"production", 1.5},
		{"staging", 1.0},
		{"development", 0.5},
		{"dr", 1.2},
		{"unknown", 1.0},
		{"", 1.0},
		{"PRODUCTION", 1.0}, // Case sensitive, returns default
	}

	for _, tt := range tests {
		t.Run(tt.environment, func(t *testing.T) {
			result := EnvironmentRiskMultiplier(tt.environment)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultRiskWeights(t *testing.T) {
	weights := DefaultRiskWeights()

	// Verify default weights
	assert.Equal(t, 0.25, weights.DriftAge)
	assert.Equal(t, 0.20, weights.VulnCount)
	assert.Equal(t, 0.25, weights.CriticalVulns)
	assert.Equal(t, 0.15, weights.ComplianceStatus)
	assert.Equal(t, 0.15, weights.Environment)

	// Verify weights sum to ~1.0 (accounting for float precision)
	sum := weights.DriftAge + weights.VulnCount + weights.CriticalVulns + weights.ComplianceStatus + weights.Environment
	assert.InDelta(t, 1.0, sum, 0.001)
}

func TestRiskLevelConstants(t *testing.T) {
	assert.Equal(t, RiskLevel("critical"), RiskLevelCritical)
	assert.Equal(t, RiskLevel("high"), RiskLevelHigh)
	assert.Equal(t, RiskLevel("medium"), RiskLevelMedium)
	assert.Equal(t, RiskLevel("low"), RiskLevelLow)
}
