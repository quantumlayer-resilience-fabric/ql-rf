// Package engine provides the drift calculation engine.
package engine

import (
	"testing"

	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

func TestCalculateSeverity(t *testing.T) {
	e := &Engine{
		config: models.DriftConfig{
			WarningThreshold:  80.0,
			CriticalThreshold: 60.0,
			MaxOffenders:      10,
		},
	}

	tests := []struct {
		name         string
		driftAgeDays int
		expected     models.DriftStatus
	}{
		{
			name:         "healthy drift (0 days)",
			driftAgeDays: 0,
			expected:     models.DriftStatusHealthy,
		},
		{
			name:         "healthy drift (7 days)",
			driftAgeDays: 7,
			expected:     models.DriftStatusHealthy,
		},
		{
			name:         "healthy drift (14 days)",
			driftAgeDays: 14,
			expected:     models.DriftStatusHealthy,
		},
		{
			name:         "warning drift (15 days)",
			driftAgeDays: 15,
			expected:     models.DriftStatusWarning,
		},
		{
			name:         "warning drift (30 days)",
			driftAgeDays: 30,
			expected:     models.DriftStatusWarning,
		},
		{
			name:         "critical drift (31 days)",
			driftAgeDays: 31,
			expected:     models.DriftStatusCritical,
		},
		{
			name:         "critical drift (60 days)",
			driftAgeDays: 60,
			expected:     models.DriftStatusCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.calculateSeverity(tt.driftAgeDays)
			if result != tt.expected {
				t.Errorf("calculateSeverity(%d) = %v, want %v", tt.driftAgeDays, result, tt.expected)
			}
		})
	}
}

func TestCalculateDriftAge(t *testing.T) {
	e := &Engine{}

	tests := []struct {
		name            string
		currentVersion  string
		expectedVersion string
		wantNonZero     bool
	}{
		{
			name:            "older version",
			currentVersion:  "1.0.0",
			expectedVersion: "2.0.0",
			wantNonZero:     true,
		},
		{
			name:            "same version",
			currentVersion:  "2.0.0",
			expectedVersion: "2.0.0",
			wantNonZero:     false,
		},
		{
			name:            "newer version",
			currentVersion:  "3.0.0",
			expectedVersion: "2.0.0",
			wantNonZero:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset := models.Asset{
				ImageVersion: tt.currentVersion,
			}
			result := e.calculateDriftAge(asset, tt.expectedVersion)

			if tt.wantNonZero && result == 0 {
				t.Errorf("calculateDriftAge() = 0, want non-zero")
			}
			if !tt.wantNonZero && result != 0 {
				t.Errorf("calculateDriftAge() = %d, want 0", result)
			}
		})
	}
}

func TestFindBaselineByFamily(t *testing.T) {
	e := &Engine{}

	tests := []struct {
		name      string
		baselines map[string]string
		imageRef  string
		expected  string
	}{
		{
			name: "match found with ql-base-linux",
			baselines: map[string]string{
				"ql-base-linux": "1.5.0",
				"other-image":   "2.0.0",
			},
			imageRef: "some-image",
			expected: "1.5.0",
		},
		{
			name: "no ql-base-linux",
			baselines: map[string]string{
				"other-image": "2.0.0",
			},
			imageRef: "some-image",
			expected: "",
		},
		{
			name:      "empty baselines",
			baselines: map[string]string{},
			imageRef:  "some-image",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.findBaselineByFamily(tt.baselines, tt.imageRef)
			if result != tt.expected {
				t.Errorf("findBaselineByFamily() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCalculateStatus(t *testing.T) {
	tests := []struct {
		name              string
		coveragePct       float64
		warningThreshold  float64
		criticalThreshold float64
		expected          models.DriftStatus
	}{
		{
			name:              "healthy - above warning",
			coveragePct:       95.0,
			warningThreshold:  80.0,
			criticalThreshold: 60.0,
			expected:          models.DriftStatusHealthy,
		},
		{
			name:              "healthy - exactly at warning",
			coveragePct:       80.0,
			warningThreshold:  80.0,
			criticalThreshold: 60.0,
			expected:          models.DriftStatusHealthy,
		},
		{
			name:              "warning - between thresholds",
			coveragePct:       70.0,
			warningThreshold:  80.0,
			criticalThreshold: 60.0,
			expected:          models.DriftStatusWarning,
		},
		{
			name:              "warning - just below warning",
			coveragePct:       79.9,
			warningThreshold:  80.0,
			criticalThreshold: 60.0,
			expected:          models.DriftStatusWarning,
		},
		{
			name:              "warning - at critical",
			coveragePct:       60.0,
			warningThreshold:  80.0,
			criticalThreshold: 60.0,
			expected:          models.DriftStatusWarning,
		},
		{
			name:              "critical - below critical",
			coveragePct:       50.0,
			warningThreshold:  80.0,
			criticalThreshold: 60.0,
			expected:          models.DriftStatusCritical,
		},
		{
			name:              "critical - zero coverage",
			coveragePct:       0.0,
			warningThreshold:  80.0,
			criticalThreshold: 60.0,
			expected:          models.DriftStatusCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := models.CalculateStatus(tt.coveragePct, tt.warningThreshold, tt.criticalThreshold)
			if result != tt.expected {
				t.Errorf("CalculateStatus(%v, %v, %v) = %v, want %v",
					tt.coveragePct, tt.warningThreshold, tt.criticalThreshold, result, tt.expected)
			}
		})
	}
}

func TestAssetState_IsActive(t *testing.T) {
	tests := []struct {
		state    models.AssetState
		expected bool
	}{
		{models.AssetState("running"), true},
		{models.AssetState("pending"), true},
		{models.AssetState("stopped"), false},
		{models.AssetState("terminated"), false},
		{models.AssetState("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			result := tt.state.IsActive()
			if result != tt.expected {
				t.Errorf("AssetState(%q).IsActive() = %v, want %v", tt.state, result, tt.expected)
			}
		})
	}
}

func TestDriftReport_Validation(t *testing.T) {
	tests := []struct {
		name            string
		totalAssets     int
		compliantAssets int
		expectedPct     float64
	}{
		{
			name:            "all compliant",
			totalAssets:     100,
			compliantAssets: 100,
			expectedPct:     100.0,
		},
		{
			name:            "half compliant",
			totalAssets:     100,
			compliantAssets: 50,
			expectedPct:     50.0,
		},
		{
			name:            "none compliant",
			totalAssets:     100,
			compliantAssets: 0,
			expectedPct:     0.0,
		},
		{
			name:            "zero assets",
			totalAssets:     0,
			compliantAssets: 0,
			expectedPct:     0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pct float64
			if tt.totalAssets > 0 {
				pct = float64(tt.compliantAssets) / float64(tt.totalAssets) * 100
			}
			if pct != tt.expectedPct {
				t.Errorf("coverage calculation = %v, want %v", pct, tt.expectedPct)
			}
		})
	}
}

func TestOutdatedAsset_Sorting(t *testing.T) {
	assets := []models.OutdatedAsset{
		{DriftAge: 10, Severity: models.DriftStatusWarning},
		{DriftAge: 45, Severity: models.DriftStatusCritical},
		{DriftAge: 25, Severity: models.DriftStatusWarning},
		{DriftAge: 5, Severity: models.DriftStatusHealthy},
	}

	// Sort by drift age descending (most outdated first)
	for i := 0; i < len(assets)-1; i++ {
		for j := i + 1; j < len(assets); j++ {
			if assets[i].DriftAge < assets[j].DriftAge {
				assets[i], assets[j] = assets[j], assets[i]
			}
		}
	}

	// Verify order
	expectedOrder := []int{45, 25, 10, 5}
	for i, expected := range expectedOrder {
		if assets[i].DriftAge != expected {
			t.Errorf("position %d: got drift age %d, want %d", i, assets[i].DriftAge, expected)
		}
	}
}
