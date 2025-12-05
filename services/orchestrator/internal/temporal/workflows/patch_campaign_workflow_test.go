package workflows

import (
	"testing"
)

func TestGeneratePatchCampaignPhases(t *testing.T) {
	tests := []struct {
		name           string
		assetIDs       []string
		strategy       string
		canaryPct      int
		wavePct        int
		expectedPhases int
		checkFunc      func(t *testing.T, phases []PatchPhaseInput)
	}{
		{
			name:           "immediate strategy - single phase",
			assetIDs:       []string{"a1", "a2", "a3", "a4", "a5"},
			strategy:       "immediate",
			canaryPct:      5,
			wavePct:        25,
			expectedPhases: 1,
			checkFunc: func(t *testing.T, phases []PatchPhaseInput) {
				if phases[0].PhaseType != "final" {
					t.Errorf("expected phase type 'final', got %s", phases[0].PhaseType)
				}
				if len(phases[0].AssetIDs) != 5 {
					t.Errorf("expected 5 assets, got %d", len(phases[0].AssetIDs))
				}
			},
		},
		{
			name:           "canary strategy - two phases",
			assetIDs:       []string{"a1", "a2", "a3", "a4", "a5", "a6", "a7", "a8", "a9", "a10"},
			strategy:       "canary",
			canaryPct:      10,
			wavePct:        25,
			expectedPhases: 2,
			checkFunc: func(t *testing.T, phases []PatchPhaseInput) {
				if phases[0].PhaseType != "canary" {
					t.Errorf("expected first phase type 'canary', got %s", phases[0].PhaseType)
				}
				if phases[1].PhaseType != "final" {
					t.Errorf("expected second phase type 'final', got %s", phases[1].PhaseType)
				}
				// 10% of 10 = 1 asset in canary
				if len(phases[0].AssetIDs) != 1 {
					t.Errorf("expected 1 canary asset, got %d", len(phases[0].AssetIDs))
				}
				if len(phases[1].AssetIDs) != 9 {
					t.Errorf("expected 9 final assets, got %d", len(phases[1].AssetIDs))
				}
			},
		},
		{
			name:           "rolling strategy - multiple waves",
			assetIDs:       []string{"a1", "a2", "a3", "a4", "a5", "a6", "a7", "a8", "a9", "a10"},
			strategy:       "rolling",
			canaryPct:      5,
			wavePct:        25,
			expectedPhases: -1, // Variable based on implementation
			checkFunc: func(t *testing.T, phases []PatchPhaseInput) {
				// Should have at least 2 phases
				if len(phases) < 2 {
					t.Errorf("expected at least 2 phases for rolling, got %d", len(phases))
					return
				}
				// First waves should be "wave" type
				for i := 0; i < len(phases)-1; i++ {
					if phases[i].PhaseType != "wave" {
						t.Errorf("expected phase %d type 'wave', got %s", i, phases[i].PhaseType)
					}
				}
				// Last phase should be "final"
				if phases[len(phases)-1].PhaseType != "final" {
					t.Errorf("expected last phase type 'final', got %s", phases[len(phases)-1].PhaseType)
				}
			},
		},
		{
			name:           "blue_green strategy - two phases",
			assetIDs:       []string{"a1", "a2", "a3", "a4", "a5", "a6", "a7", "a8", "a9", "a10"},
			strategy:       "blue_green",
			canaryPct:      5,
			wavePct:        25,
			expectedPhases: 2,
			checkFunc: func(t *testing.T, phases []PatchPhaseInput) {
				if phases[0].Name != "Blue" {
					t.Errorf("expected first phase name 'Blue', got %s", phases[0].Name)
				}
				if phases[1].Name != "Green" {
					t.Errorf("expected second phase name 'Green', got %s", phases[1].Name)
				}
				// Should be roughly 50/50
				if len(phases[0].AssetIDs) != 5 {
					t.Errorf("expected 5 blue assets, got %d", len(phases[0].AssetIDs))
				}
				if len(phases[1].AssetIDs) != 5 {
					t.Errorf("expected 5 green assets, got %d", len(phases[1].AssetIDs))
				}
			},
		},
		{
			name:           "empty assets",
			assetIDs:       []string{},
			strategy:       "immediate",
			canaryPct:      5,
			wavePct:        25,
			expectedPhases: 0,
			checkFunc:      func(t *testing.T, phases []PatchPhaseInput) {},
		},
		{
			name:           "single asset - canary strategy",
			assetIDs:       []string{"a1"},
			strategy:       "canary",
			canaryPct:      5,
			wavePct:        25,
			expectedPhases: 1, // Only canary phase, no final needed
			checkFunc: func(t *testing.T, phases []PatchPhaseInput) {
				if phases[0].PhaseType != "canary" {
					t.Errorf("expected phase type 'canary', got %s", phases[0].PhaseType)
				}
				if len(phases[0].AssetIDs) != 1 {
					t.Errorf("expected 1 asset, got %d", len(phases[0].AssetIDs))
				}
			},
		},
		{
			name:           "default/unknown strategy",
			assetIDs:       []string{"a1", "a2", "a3"},
			strategy:       "unknown",
			canaryPct:      5,
			wavePct:        25,
			expectedPhases: 1,
			checkFunc: func(t *testing.T, phases []PatchPhaseInput) {
				if phases[0].PhaseType != "final" {
					t.Errorf("expected phase type 'final', got %s", phases[0].PhaseType)
				}
				if len(phases[0].AssetIDs) != 3 {
					t.Errorf("expected 3 assets, got %d", len(phases[0].AssetIDs))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phases := GeneratePatchCampaignPhases("campaign-1", tt.assetIDs, tt.strategy, tt.canaryPct, tt.wavePct)

			if tt.expectedPhases >= 0 && len(phases) != tt.expectedPhases {
				t.Errorf("expected %d phases, got %d", tt.expectedPhases, len(phases))
				return
			}

			if tt.checkFunc != nil && len(phases) > 0 {
				tt.checkFunc(t, phases)
			}

			// Verify all phases have unique IDs
			seenIDs := make(map[string]bool)
			for _, p := range phases {
				if seenIDs[p.PhaseID] {
					t.Errorf("duplicate phase ID: %s", p.PhaseID)
				}
				seenIDs[p.PhaseID] = true
			}

			// Verify order is sequential
			for i, p := range phases {
				if p.Order != i+1 {
					t.Errorf("phase %d has order %d, expected %d", i, p.Order, i+1)
				}
			}

			// Verify all assets are accounted for
			totalAssets := 0
			for _, p := range phases {
				totalAssets += len(p.AssetIDs)
			}
			if totalAssets != len(tt.assetIDs) {
				t.Errorf("expected %d total assets, got %d", len(tt.assetIDs), totalAssets)
			}
		})
	}
}

func TestShouldRollback(t *testing.T) {
	tests := []struct {
		name           string
		result         *PatchCampaignWorkflowResult
		threshold      int
		expectedResult bool
	}{
		{
			name: "below threshold",
			result: &PatchCampaignWorkflowResult{
				TotalAssets:   100,
				FailedPatches: 3,
			},
			threshold:      5,
			expectedResult: false,
		},
		{
			name: "at threshold",
			result: &PatchCampaignWorkflowResult{
				TotalAssets:   100,
				FailedPatches: 5,
			},
			threshold:      5,
			expectedResult: true,
		},
		{
			name: "above threshold",
			result: &PatchCampaignWorkflowResult{
				TotalAssets:   100,
				FailedPatches: 10,
			},
			threshold:      5,
			expectedResult: true,
		},
		{
			name: "no assets",
			result: &PatchCampaignWorkflowResult{
				TotalAssets:   0,
				FailedPatches: 0,
			},
			threshold:      5,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldRollback(tt.result, tt.threshold)
			if result != tt.expectedResult {
				t.Errorf("expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}
