package agents

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llmjson"
)

func TestDriftAgent_Execute(t *testing.T) {
	// Note: Full DriftAgent execution requires tool registry with working tools.
	// These tests focus on the LLM integration via FakeLLM.
	t.Run("happy path with valid JSON response", func(t *testing.T) {
		// Create fake LLM with drift plan fixture
		fakeLLM := NewFakeLLM(DriftPlanFixture())

		// Verify the fixture can be parsed
		result, err := llmjson.ExtractJSON[map[string]any](DriftPlanFixture())
		if err != nil {
			t.Fatalf("fixture should be valid JSON: %v", err)
		}

		if result.Method != llmjson.ParseMethodDirect {
			t.Errorf("expected direct parse, got %s", result.Method)
		}

		// Verify call was recorded after Complete
		_, _ = fakeLLM.Complete(context.Background(), nil)
		if fakeLLM.CallCount() != 1 {
			t.Errorf("expected 1 call, got %d", fakeLLM.CallCount())
		}
	})

	t.Run("response wrapped in markdown", func(t *testing.T) {
		// Create fake LLM with wrapped response
		fakeLLM := NewFakeLLM(WrappedJSONFixture())

		resp, err := fakeLLM.Complete(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should be able to extract JSON from markdown
		result, err := llmjson.ExtractJSON[map[string]any](resp.Content)
		if err != nil {
			t.Fatalf("should extract JSON from markdown: %v", err)
		}

		if result.Method != llmjson.ParseMethodExtracted {
			t.Errorf("expected extracted parse method, got %s", result.Method)
		}
	})

	t.Run("malformed JSON response", func(t *testing.T) {
		fakeLLM := NewFakeLLM(MalformedJSONFixture())

		resp, err := fakeLLM.Complete(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should fail to extract
		_, err = llmjson.ExtractJSON[map[string]any](resp.Content)
		if err == nil {
			t.Error("expected error for malformed JSON")
		}
	})

	t.Run("truncated JSON response", func(t *testing.T) {
		fakeLLM := NewFakeLLM(TruncatedJSONFixture())

		resp, err := fakeLLM.Complete(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should fail to extract
		_, err = llmjson.ExtractJSON[map[string]any](resp.Content)
		if err == nil {
			t.Error("expected error for truncated JSON")
		}
	})

	t.Run("extra fields in response", func(t *testing.T) {
		fakeLLM := NewFakeLLM(ExtraFieldsJSONFixture())

		resp, err := fakeLLM.Complete(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should still parse (Go ignores extra fields)
		result, err := llmjson.ExtractJSON[map[string]any](resp.Content)
		if err != nil {
			t.Fatalf("should handle extra fields: %v", err)
		}

		// Verify hallucinated fields are present in raw but can be handled
		plan := result.Value
		if plan["plan"] == nil {
			t.Error("expected plan field")
		}
	})
}

func TestDriftAgent_PlanParsing(t *testing.T) {
	tests := []struct {
		name          string
		response      string
		expectSuccess bool
		expectMethod  llmjson.ParseMethod
	}{
		{
			name:          "clean JSON",
			response:      DriftPlanFixture(),
			expectSuccess: true,
			expectMethod:  llmjson.ParseMethodDirect,
		},
		{
			name:          "wrapped in markdown",
			response:      WrappedJSONFixture(),
			expectSuccess: true,
			expectMethod:  llmjson.ParseMethodExtracted,
		},
		{
			name:          "with trailing comma",
			response:      `{"plan": {"name": "test",}, "summary": "test"}`,
			expectSuccess: true,
			expectMethod:  llmjson.ParseMethodLenient,
		},
		{
			name:          "truncated",
			response:      TruncatedJSONFixture(),
			expectSuccess: false,
		},
		{
			name:          "malformed",
			response:      MalformedJSONFixture(),
			expectSuccess: false,
		},
		{
			name: "with prefix text",
			response: `Based on my analysis, here is the plan:

{"plan": {"name": "drift-plan", "phases": []}, "summary": "test plan"}`,
			expectSuccess: true,
			expectMethod:  llmjson.ParseMethodExtracted,
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

func TestDriftAgent_QualityScoring(t *testing.T) {
	tests := []struct {
		name           string
		parseMethod    llmjson.ParseMethod
		riskLevel      string
		expectedHITL   bool
		maxQuality     float64
	}{
		{
			name:         "direct parse, low risk",
			parseMethod:  llmjson.ParseMethodDirect,
			riskLevel:    RiskLevelLow,
			expectedHITL: false,
			maxQuality:   100,
		},
		{
			name:         "extracted parse, medium risk",
			parseMethod:  llmjson.ParseMethodExtracted,
			riskLevel:    RiskLevelMedium,
			expectedHITL: true, // Non-direct parse forces HITL
			maxQuality:   60,
		},
		{
			name:         "lenient parse, any risk",
			parseMethod:  llmjson.ParseMethodLenient,
			riskLevel:    RiskLevelLow,
			expectedHITL: true,
			maxQuality:   60,
		},
		{
			name:         "direct parse, high risk",
			parseMethod:  llmjson.ParseMethodDirect,
			riskLevel:    RiskLevelHigh,
			expectedHITL: true, // High risk forces HITL
			maxQuality:   100,
		},
		{
			name:         "direct parse, critical risk",
			parseMethod:  llmjson.ParseMethodDirect,
			riskLevel:    RiskLevelCritical,
			expectedHITL: true,
			maxQuality:   100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewAgentResponse("task-123", "drift_agent", "1.0.0")
			resp.RiskLevel = tt.riskLevel
			resp.QualityScore = 85

			// Apply parse warning if not direct
			if tt.parseMethod != llmjson.ParseMethodDirect {
				resp.WithParseWarning(string(tt.parseMethod), "test warning")
			}

			// Check HITL requirement
			requiresHITL := ShouldRequireHITL(resp)
			if requiresHITL != tt.expectedHITL {
				t.Errorf("expected requires_hitl=%v, got %v", tt.expectedHITL, requiresHITL)
			}

			// Check quality capping
			if resp.QualityScore > tt.maxQuality {
				t.Errorf("expected quality <= %v, got %v", tt.maxQuality, resp.QualityScore)
			}
		})
	}
}

func TestDriftAgent_ResponseBuilder(t *testing.T) {
	resp := NewAgentResponse("task-123", "drift_agent", "1.0.0")

	// Verify defaults
	if resp.Status != AgentStatusPendingApproval {
		t.Errorf("expected status pending_approval, got %s", resp.Status)
	}

	if len(resp.Actions) != 3 {
		t.Errorf("expected 3 actions, got %d", len(resp.Actions))
	}

	// Test chaining
	plan := map[string]any{
		"phases": []any{
			map[string]any{"name": "canary", "batch_percent": 5},
		},
	}

	resp.WithPlan(plan).
		WithQuality(85, RiskLevelMedium).
		WithTokens(1000, 500)

	if resp.PlanJSON == nil {
		t.Error("expected plan to be set")
	}

	if resp.QualityScore != 85 {
		t.Errorf("expected quality_score=85, got %v", resp.QualityScore)
	}

	if resp.RiskLevel != RiskLevelMedium {
		t.Errorf("expected risk_level=medium, got %s", resp.RiskLevel)
	}

	if resp.TokensUsed != 1500 {
		t.Errorf("expected tokens_used=1500, got %d", resp.TokensUsed)
	}
}

func TestDriftAgent_FakeLLMRecording(t *testing.T) {
	// Create fake with multiple responses
	fakeLLM := NewFakeLLMWithResponses([]FakeLLMResponse{
		{Content: `{"step": 1}`, InputToks: 100, OutputToks: 50},
		{Content: `{"step": 2}`, InputToks: 200, OutputToks: 75},
		{Content: `{"step": 3}`, InputToks: 150, OutputToks: 60},
	})

	// Make multiple calls
	for i := 0; i < 3; i++ {
		_, err := fakeLLM.Complete(context.Background(), nil)
		if err != nil {
			t.Fatalf("call %d failed: %v", i, err)
		}
	}

	// Verify call count
	if fakeLLM.CallCount() != 3 {
		t.Errorf("expected 3 calls, got %d", fakeLLM.CallCount())
	}

	// Verify last call
	lastCall := fakeLLM.LastCall()
	if lastCall == nil {
		t.Fatal("expected last call")
	}

	// Fourth call should error
	_, err := fakeLLM.Complete(context.Background(), nil)
	if err == nil {
		t.Error("expected error when responses exhausted")
	}
}

func TestDriftAgent_TestFixtures(t *testing.T) {
	// Verify all fixtures are valid for their intended purpose
	t.Run("DriftPlanFixture is valid JSON", func(t *testing.T) {
		var v map[string]any
		if err := json.Unmarshal([]byte(DriftPlanFixture()), &v); err != nil {
			t.Errorf("fixture should be valid JSON: %v", err)
		}

		if v["plan"] == nil {
			t.Error("expected plan field")
		}
		if v["summary"] == nil {
			t.Error("expected summary field")
		}
	})

	t.Run("PatchPlanFixture is valid JSON", func(t *testing.T) {
		var v map[string]any
		if err := json.Unmarshal([]byte(PatchPlanFixture()), &v); err != nil {
			t.Errorf("fixture should be valid JSON: %v", err)
		}
	})

	t.Run("CompliancePlanFixture is valid JSON", func(t *testing.T) {
		var v map[string]any
		if err := json.Unmarshal([]byte(CompliancePlanFixture()), &v); err != nil {
			t.Errorf("fixture should be valid JSON: %v", err)
		}
	})

	t.Run("MalformedJSONFixture is invalid JSON", func(t *testing.T) {
		var v map[string]any
		if err := json.Unmarshal([]byte(MalformedJSONFixture()), &v); err == nil {
			t.Error("malformed fixture should be invalid JSON")
		}
	})

	t.Run("TruncatedJSONFixture is invalid JSON", func(t *testing.T) {
		var v map[string]any
		if err := json.Unmarshal([]byte(TruncatedJSONFixture()), &v); err == nil {
			t.Error("truncated fixture should be invalid JSON")
		}
	})
}

func TestDriftAgent_TestHelpers(t *testing.T) {
	t.Run("TestTaskSpec", func(t *testing.T) {
		spec := TestTaskSpec(TaskTypeDriftRemediation, "Remediate drift in production")

		if spec.ID == "" {
			t.Error("expected non-empty task ID")
		}
		if spec.TaskType != TaskTypeDriftRemediation {
			t.Errorf("expected task type %s, got %s", TaskTypeDriftRemediation, spec.TaskType)
		}
		if spec.Goal != "Remediate drift in production" {
			t.Errorf("unexpected goal: %s", spec.Goal)
		}
	})

	t.Run("TestAgentRequest", func(t *testing.T) {
		req := TestAgentRequest("Check drift status")

		if req.TaskID == "" {
			t.Error("expected non-empty task ID")
		}
		if req.Intent != "Check drift status" {
			t.Errorf("unexpected intent: %s", req.Intent)
		}
		if req.Environment != "staging" {
			t.Errorf("expected environment 'staging', got %s", req.Environment)
		}
	})
}

func TestDriftAgent_AssertHelpers(t *testing.T) {
	resp := NewAgentResponse("task-123", "drift_agent", "1.0.0")
	resp.QualityScore = 85
	resp.RiskLevel = RiskLevelMedium

	assert := Assert(resp)

	if !assert.HasStatus(AgentStatusPendingApproval) {
		t.Error("expected status pending_approval")
	}

	if !assert.HasQualityAbove(80) {
		t.Error("expected quality above 80")
	}

	if assert.HasQualityAbove(90) {
		t.Error("quality should not be above 90")
	}

	if !assert.HasNoErrors() {
		t.Error("expected no errors")
	}

	// Add an error
	resp.WithError("TEST_ERROR", "test error message")

	if assert.HasNoErrors() {
		t.Error("expected errors after adding one")
	}
}

func TestDriftAgent_QualityLevels(t *testing.T) {
	tests := []struct {
		score    float64
		expected QualityLevel
	}{
		{95, QualityExcellent},
		{90, QualityExcellent},
		{89, QualityGood},
		{70, QualityGood},
		{69, QualityFair},
		{50, QualityFair},
		{49, QualityPoor},
		{0, QualityPoor},
	}

	for _, tt := range tests {
		t.Run(string(tt.expected), func(t *testing.T) {
			level := GetQualityLevel(tt.score)
			if level != tt.expected {
				t.Errorf("score %v: expected %s, got %s", tt.score, tt.expected, level)
			}
		})
	}
}
