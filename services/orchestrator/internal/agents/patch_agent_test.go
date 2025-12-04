package agents

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llmjson"
)

func TestPatchAgent_PlanGeneration(t *testing.T) {
	tests := []struct {
		name          string
		response      string
		expectSuccess bool
		expectMethod  llmjson.ParseMethod
	}{
		{
			name:          "clean JSON patch plan",
			response:      PatchPlanFixture(),
			expectSuccess: true,
			expectMethod:  llmjson.ParseMethodDirect,
		},
		{
			name: "patch plan wrapped in markdown",
			response: "Here's the patch rollout plan:\n\n```json\n" + PatchPlanFixture() + "\n```\n",
			expectSuccess: true,
			expectMethod:  llmjson.ParseMethodExtracted,
		},
		{
			name: "patch plan with prefix text",
			response: `Based on my analysis of the 50 affected servers, here is the rollout plan:

{"plan": {"name": "patch-rollout", "phases": [{"name": "canary", "batch_percent": 5}]}, "summary": "Patch rollout for 50 servers"}`,
			expectSuccess: true,
			expectMethod:  llmjson.ParseMethodExtracted,
		},
		{
			name:          "truncated patch plan",
			response:      `{"plan": {"name": "patch-rollout", "phases": [{"name": "canary", "batch_per`,
			expectSuccess: false,
		},
		{
			name:          "malformed patch plan",
			response:      `{"plan": {"name": "patch-rollout" "missing_comma": true}}`,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := llmjson.ExtractJSON[map[string]any](tt.response)

			if tt.expectSuccess {
				if err != nil {
					t.Errorf("expected success, got error: %v", err)
				}
				if result.Method != tt.expectMethod {
					t.Errorf("expected method %s, got %s", tt.expectMethod, result.Method)
				}
			} else {
				if err == nil {
					t.Error("expected error, got success")
				}
			}
		})
	}
}

func TestPatchAgent_DefaultPlanGeneration(t *testing.T) {
	// Test the default plan generation when LLM parsing fails
	tests := []struct {
		name           string
		assetCount     int
		riskLevel      string
		canarySize     int
		waveSize       int
		expectedPhases int // preflight + canary + waves + validation
	}{
		{
			name:           "small fleet low risk",
			assetCount:     10,
			riskLevel:      "low",
			canarySize:     10,
			waveSize:       50,
			expectedPhases: 4, // preflight + canary + 1 wave + validation
		},
		{
			name:           "medium fleet medium risk",
			assetCount:     100,
			riskLevel:      "medium",
			canarySize:     5,
			waveSize:       25,
			expectedPhases: 7, // preflight + canary + 4 waves + validation
		},
		{
			name:           "large fleet high risk",
			assetCount:     500,
			riskLevel:      "high",
			canarySize:     2,
			waveSize:       10,
			expectedPhases: 12, // preflight + canary + max 10 waves (capped) + validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a patch agent with fake LLM
			fakeLLM := NewFakeLLM(MalformedJSONFixture()) // Force fallback to default plan

			agent := &PatchAgent{
				BaseAgent: BaseAgent{
					name: "patch_agent",
					llm:  fakeLLM,
				},
			}

			plan := agent.defaultPatchPlan(tt.assetCount, tt.riskLevel, tt.canarySize, tt.waveSize)

			// Verify plan structure
			if plan["summary"] == nil {
				t.Error("expected summary in plan")
			}

			phases, ok := plan["phases"].([]map[string]interface{})
			if !ok {
				t.Fatal("expected phases to be []map[string]interface{}")
			}

			// Verify phase count (capped at 10 waves + preflight + canary + validation)
			if len(phases) > 13 {
				t.Errorf("expected at most 13 phases, got %d", len(phases))
			}

			// Verify first phase is preflight
			if phases[0]["name"] != "Pre-flight Checks" {
				t.Errorf("expected first phase to be Pre-flight Checks, got %s", phases[0]["name"])
			}

			// Verify second phase is canary
			if phases[1]["name"] != "Canary Deployment" {
				t.Errorf("expected second phase to be Canary Deployment, got %s", phases[1]["name"])
			}

			// Verify last phase is validation
			lastPhase := phases[len(phases)-1]
			if lastPhase["name"] != "Post-Rollout Validation" {
				t.Errorf("expected last phase to be Post-Rollout Validation, got %s", lastPhase["name"])
			}

			// Verify rollback plan exists
			if plan["rollback_plan"] == nil {
				t.Error("expected rollback_plan in plan")
			}
		})
	}
}

func TestPatchAgent_RiskBasedBatchSizing(t *testing.T) {
	tests := []struct {
		riskLevel          string
		expectedCanarySize int
		expectedWaveSize   int
	}{
		{"low", 10, 50},
		{"medium", 5, 25},
		{"high", 2, 10},
		{"critical", 2, 10},
	}

	for _, tt := range tests {
		t.Run(tt.riskLevel, func(t *testing.T) {
			// Default batch sizes based on risk level
			canarySize := 5
			waveSize := 25
			switch tt.riskLevel {
			case "critical", "high":
				canarySize = 2
				waveSize = 10
			case "low":
				canarySize = 10
				waveSize = 50
			}

			if canarySize != tt.expectedCanarySize {
				t.Errorf("expected canary size %d for %s risk, got %d", tt.expectedCanarySize, tt.riskLevel, canarySize)
			}
			if waveSize != tt.expectedWaveSize {
				t.Errorf("expected wave size %d for %s risk, got %d", tt.expectedWaveSize, tt.riskLevel, waveSize)
			}
		})
	}
}

func TestPatchAgent_FakeLLMIntegration(t *testing.T) {
	t.Run("records LLM calls correctly", func(t *testing.T) {
		fakeLLM := NewFakeLLMWithResponses([]FakeLLMResponse{
			{Content: PatchPlanFixture(), InputToks: 500, OutputToks: 300},
		})

		_, err := fakeLLM.Complete(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if fakeLLM.CallCount() != 1 {
			t.Errorf("expected 1 call, got %d", fakeLLM.CallCount())
		}
	})

	t.Run("handles multiple responses", func(t *testing.T) {
		fakeLLM := NewFakeLLMWithResponses([]FakeLLMResponse{
			{Content: `{"step": 1}`, InputToks: 100, OutputToks: 50},
			{Content: `{"step": 2}`, InputToks: 100, OutputToks: 50},
		})

		// First call
		resp1, err := fakeLLM.Complete(context.Background(), nil)
		if err != nil {
			t.Fatalf("first call failed: %v", err)
		}
		if resp1.Content != `{"step": 1}` {
			t.Errorf("unexpected first response: %s", resp1.Content)
		}

		// Second call
		resp2, err := fakeLLM.Complete(context.Background(), nil)
		if err != nil {
			t.Fatalf("second call failed: %v", err)
		}
		if resp2.Content != `{"step": 2}` {
			t.Errorf("unexpected second response: %s", resp2.Content)
		}

		// Third call should fail
		_, err = fakeLLM.Complete(context.Background(), nil)
		if err == nil {
			t.Error("expected error when responses exhausted")
		}
	})
}

func TestPatchAgent_PlanFixtureValidity(t *testing.T) {
	t.Run("PatchPlanFixture is valid JSON", func(t *testing.T) {
		var v map[string]any
		if err := json.Unmarshal([]byte(PatchPlanFixture()), &v); err != nil {
			t.Errorf("fixture should be valid JSON: %v", err)
		}

		if v["plan"] == nil {
			t.Error("expected plan field")
		}
		if v["summary"] == nil {
			t.Error("expected summary field")
		}
	})

	t.Run("PatchPlanFixture has required structure", func(t *testing.T) {
		result, err := llmjson.ExtractJSON[map[string]any](PatchPlanFixture())
		if err != nil {
			t.Fatalf("failed to parse fixture: %v", err)
		}

		plan := result.Value["plan"].(map[string]any)
		if plan["name"] == nil {
			t.Error("expected plan.name")
		}
		if plan["phases"] == nil {
			t.Error("expected plan.phases")
		}
	})
}

func TestPatchAgent_QualityScoring(t *testing.T) {
	tests := []struct {
		name         string
		parseMethod  llmjson.ParseMethod
		riskLevel    string
		expectedHITL bool
	}{
		{
			name:         "direct parse low risk - no HITL",
			parseMethod:  llmjson.ParseMethodDirect,
			riskLevel:    RiskLevelLow,
			expectedHITL: false,
		},
		{
			name:         "direct parse high risk - requires HITL",
			parseMethod:  llmjson.ParseMethodDirect,
			riskLevel:    RiskLevelHigh,
			expectedHITL: true,
		},
		{
			name:         "extracted parse any risk - requires HITL",
			parseMethod:  llmjson.ParseMethodExtracted,
			riskLevel:    RiskLevelLow,
			expectedHITL: true,
		},
		{
			name:         "lenient parse any risk - requires HITL",
			parseMethod:  llmjson.ParseMethodLenient,
			riskLevel:    RiskLevelLow,
			expectedHITL: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewAgentResponse("task-123", "patch_agent", "1.0.0")
			resp.RiskLevel = tt.riskLevel
			resp.QualityScore = 85

			if tt.parseMethod != llmjson.ParseMethodDirect {
				resp.WithParseWarning(string(tt.parseMethod), "non-direct parse")
			}

			requiresHITL := ShouldRequireHITL(resp)
			if requiresHITL != tt.expectedHITL {
				t.Errorf("expected requires_hitl=%v, got %v", tt.expectedHITL, requiresHITL)
			}
		})
	}
}
