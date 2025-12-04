package agents

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentRequest_Fields(t *testing.T) {
	req := AgentRequest{
		TaskID:      "task-123",
		OrgID:       uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		UserID:      uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Environment: "production",
		Intent:      "Fix all drifted servers",
		Context: AgentContext{
			Platforms:   []string{"aws", "azure"},
			Regions:     []string{"us-east-1", "us-west-2"},
			AssetFilter: "env=prod",
			Tags:        map[string]string{"team": "infra"},
		},
		Guardrails: AgentGuardrails{
			MaxBatchSize:   10,
			RequireCanary:  true,
			MaxRiskLevel:   "high",
			TimeoutMinutes: 30,
			RequireHITL:    true,
		},
	}

	assert.Equal(t, "task-123", req.TaskID)
	assert.Equal(t, "production", req.Environment)
	assert.Len(t, req.Context.Platforms, 2)
	assert.Equal(t, 10, req.Guardrails.MaxBatchSize)
	assert.True(t, req.Guardrails.RequireCanary)
}

func TestAgentContext(t *testing.T) {
	ctx := AgentContext{
		Platforms:   []string{"aws"},
		Regions:     []string{"us-east-1"},
		AssetFilter: "type=ec2",
		Tags:        map[string]string{"env": "prod"},
		Metadata:    map[string]any{"source": "api"},
	}

	assert.Contains(t, ctx.Platforms, "aws")
	assert.Contains(t, ctx.Regions, "us-east-1")
	assert.Equal(t, "type=ec2", ctx.AssetFilter)
	assert.Equal(t, "prod", ctx.Tags["env"])
}

func TestAgentGuardrails(t *testing.T) {
	tests := []struct {
		name       string
		guardrails AgentGuardrails
	}{
		{
			name: "strict guardrails",
			guardrails: AgentGuardrails{
				MaxBatchSize:   5,
				RequireCanary:  true,
				MaxRiskLevel:   "medium",
				ExcludedEnvs:   []string{"production"},
				TimeoutMinutes: 15,
				RequireHITL:    true,
				DryRunOnly:     true,
			},
		},
		{
			name: "relaxed guardrails",
			guardrails: AgentGuardrails{
				MaxBatchSize:   100,
				RequireCanary:  false,
				MaxRiskLevel:   "critical",
				TimeoutMinutes: 120,
				RequireHITL:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify serialization works
			data, err := json.Marshal(tt.guardrails)
			require.NoError(t, err)

			var parsed AgentGuardrails
			err = json.Unmarshal(data, &parsed)
			require.NoError(t, err)

			assert.Equal(t, tt.guardrails.MaxBatchSize, parsed.MaxBatchSize)
			assert.Equal(t, tt.guardrails.RequireHITL, parsed.RequireHITL)
		})
	}
}

func TestStepResult(t *testing.T) {
	step := StepResult{
		StepName:   "query_assets",
		Status:     "completed",
		Output:     json.RawMessage(`{"count": 50}`),
		DurationMs: 1500,
	}

	assert.Equal(t, "query_assets", step.StepName)
	assert.Equal(t, "completed", step.Status)
	assert.Equal(t, int64(1500), step.DurationMs)
}

func TestAgentResponse(t *testing.T) {
	resp := &AgentResponse{
		TaskID:         "task-123",
		AgentName:      "drift_agent",
		Version:        "1.0.0",
		Status:         AgentStatusPendingApproval,
		StatusCode:     200,
		StatusText:     "Plan generated successfully",
		QualityScore:   85.5,
		RiskLevel:      RiskLevelMedium,
		Confidence:     0.92,
		Summary:        "Found 10 drifted assets",
		AffectedAssets: 10,
		TokensUsed:     2500,
		InputTokens:    1500,
		OutputTokens:   1000,
	}

	assert.Equal(t, "task-123", resp.TaskID)
	assert.Equal(t, 85.5, resp.QualityScore)
	assert.Equal(t, 10, resp.AffectedAssets)
	assert.Equal(t, 2500, resp.TokensUsed)
}

func TestToolInvocation(t *testing.T) {
	inv := ToolInvocation{
		ToolName:   "query_assets",
		Parameters: map[string]any{"platform": "aws", "env": "prod"},
		Result:     map[string]any{"count": 50},
		DurationMs: 250,
		Timestamp:  "2024-01-15T10:30:00Z",
	}

	assert.Equal(t, "query_assets", inv.ToolName)
	assert.Equal(t, "aws", inv.Parameters["platform"])
	assert.Equal(t, int64(250), inv.DurationMs)
}

func TestAgentError(t *testing.T) {
	err := AgentError{
		Code:    "TOOL_FAILED",
		Message: "Query assets tool timed out",
		Details: map[string]any{"timeout_ms": 30000},
	}

	assert.Equal(t, "TOOL_FAILED", err.Code)
	assert.Contains(t, err.Message, "timed out")
}

func TestGetQualityLevel(t *testing.T) {
	tests := []struct {
		score    float64
		expected QualityLevel
	}{
		{100, QualityExcellent},
		{95, QualityExcellent},
		{90, QualityExcellent},
		{89.9, QualityGood},
		{85, QualityGood},
		{70, QualityGood},
		{69.9, QualityFair},
		{55, QualityFair},
		{50, QualityFair},
		{49.9, QualityPoor},
		{25, QualityPoor},
		{0, QualityPoor},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			level := GetQualityLevel(tt.score)
			assert.Equal(t, tt.expected, level, "score: %v", tt.score)
		})
	}
}

func TestRiskLevelConstants(t *testing.T) {
	assert.Equal(t, "low", RiskLevelLow)
	assert.Equal(t, "medium", RiskLevelMedium)
	assert.Equal(t, "high", RiskLevelHigh)
	assert.Equal(t, "critical", RiskLevelCritical)
}

func TestShouldRequireHITL(t *testing.T) {
	tests := []struct {
		name         string
		response     *AgentResponse
		expectedHITL bool
	}{
		{
			name: "high quality low risk - no HITL",
			response: &AgentResponse{
				QualityScore: 85,
				RiskLevel:    RiskLevelLow,
				ParseMethod:  "direct",
			},
			expectedHITL: false,
		},
		{
			name: "poor quality - requires HITL",
			response: &AgentResponse{
				QualityScore: 45,
				RiskLevel:    RiskLevelLow,
				ParseMethod:  "direct",
			},
			expectedHITL: true,
		},
		{
			name: "high risk - requires HITL",
			response: &AgentResponse{
				QualityScore: 90,
				RiskLevel:    RiskLevelHigh,
				ParseMethod:  "direct",
			},
			expectedHITL: true,
		},
		{
			name: "critical risk - requires HITL",
			response: &AgentResponse{
				QualityScore: 95,
				RiskLevel:    RiskLevelCritical,
				ParseMethod:  "direct",
			},
			expectedHITL: true,
		},
		{
			name: "extracted parse - requires HITL",
			response: &AgentResponse{
				QualityScore: 85,
				RiskLevel:    RiskLevelLow,
				ParseMethod:  "extracted",
			},
			expectedHITL: true,
		},
		{
			name: "lenient parse - requires HITL",
			response: &AgentResponse{
				QualityScore: 85,
				RiskLevel:    RiskLevelLow,
				ParseMethod:  "lenient",
			},
			expectedHITL: true,
		},
		{
			name: "has errors - requires HITL",
			response: &AgentResponse{
				QualityScore: 85,
				RiskLevel:    RiskLevelLow,
				ParseMethod:  "direct",
				Errors:       []AgentError{{Code: "ERR", Message: "test error"}},
			},
			expectedHITL: true,
		},
		{
			name: "medium risk good quality - no HITL",
			response: &AgentResponse{
				QualityScore: 75,
				RiskLevel:    RiskLevelMedium,
				ParseMethod:  "direct",
			},
			expectedHITL: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldRequireHITL(tt.response)
			assert.Equal(t, tt.expectedHITL, result)
		})
	}
}

func TestNewAgentResponse(t *testing.T) {
	resp := NewAgentResponse("task-123", "drift_agent", "1.0.0")

	assert.Equal(t, "task-123", resp.TaskID)
	assert.Equal(t, "drift_agent", resp.AgentName)
	assert.Equal(t, "1.0.0", resp.Version)
	assert.Equal(t, AgentStatusPendingApproval, resp.Status)
	assert.Len(t, resp.Actions, 3) // approve, modify, reject
}

func TestAgentResponse_WithPlan(t *testing.T) {
	resp := NewAgentResponse("task-123", "drift_agent", "1.0.0")

	plan := map[string]any{
		"name":   "drift-remediation",
		"phases": []string{"identify", "remediate", "verify"},
	}

	resp.WithPlan(plan)

	assert.NotNil(t, resp.PlanJSON)

	var parsed map[string]any
	err := json.Unmarshal(resp.PlanJSON, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "drift-remediation", parsed["name"])
}

func TestAgentResponse_WithQuality(t *testing.T) {
	resp := NewAgentResponse("task-123", "drift_agent", "1.0.0")
	resp.WithQuality(87.5, RiskLevelMedium)

	assert.Equal(t, 87.5, resp.QualityScore)
	assert.Equal(t, RiskLevelMedium, resp.RiskLevel)
}

func TestAgentResponse_WithTokens(t *testing.T) {
	resp := NewAgentResponse("task-123", "drift_agent", "1.0.0")
	resp.WithTokens(1500, 1000)

	assert.Equal(t, 1500, resp.InputTokens)
	assert.Equal(t, 1000, resp.OutputTokens)
	assert.Equal(t, 2500, resp.TokensUsed)
}

func TestAgentResponse_WithError(t *testing.T) {
	resp := NewAgentResponse("task-123", "drift_agent", "1.0.0")
	resp.WithError("TIMEOUT", "Request timed out after 30s")
	resp.WithError("RETRY", "Will retry with backoff")

	assert.Len(t, resp.Errors, 2)
	assert.Equal(t, "TIMEOUT", resp.Errors[0].Code)
	assert.Equal(t, "RETRY", resp.Errors[1].Code)
}

func TestAgentResponse_WithParseWarning(t *testing.T) {
	t.Run("caps quality for non-direct parse", func(t *testing.T) {
		resp := NewAgentResponse("task-123", "drift_agent", "1.0.0")
		resp.QualityScore = 85 // Set high quality initially
		resp.WithParseWarning("extracted", "JSON was extracted from markdown block")

		assert.Equal(t, "extracted", resp.ParseMethod)
		assert.Equal(t, 60.0, resp.QualityScore) // Capped at 60
	})

	t.Run("no cap for direct parse", func(t *testing.T) {
		resp := NewAgentResponse("task-123", "drift_agent", "1.0.0")
		resp.QualityScore = 85
		resp.WithParseWarning("direct", "")

		assert.Equal(t, "direct", resp.ParseMethod)
		assert.Equal(t, 85.0, resp.QualityScore) // Not capped
	})

	t.Run("no cap if already below 60", func(t *testing.T) {
		resp := NewAgentResponse("task-123", "drift_agent", "1.0.0")
		resp.QualityScore = 45
		resp.WithParseWarning("lenient", "JSON required lenient parsing")

		assert.Equal(t, "lenient", resp.ParseMethod)
		assert.Equal(t, 45.0, resp.QualityScore) // Stays at 45
	})
}

func TestAgentResponse_Chaining(t *testing.T) {
	resp := NewAgentResponse("task-123", "drift_agent", "1.0.0").
		WithPlan(map[string]any{"name": "test"}).
		WithQuality(85, RiskLevelLow).
		WithTokens(1000, 500).
		WithError("WARN", "Minor warning")

	assert.Equal(t, 85.0, resp.QualityScore)
	assert.Equal(t, 1500, resp.TokensUsed)
	assert.Len(t, resp.Errors, 1)
	assert.NotNil(t, resp.PlanJSON)
}

func TestQualityLevel(t *testing.T) {
	levels := []QualityLevel{
		QualityExcellent,
		QualityGood,
		QualityFair,
		QualityPoor,
	}

	assert.Equal(t, QualityLevel("excellent"), QualityExcellent)
	assert.Equal(t, QualityLevel("good"), QualityGood)
	assert.Equal(t, QualityLevel("fair"), QualityFair)
	assert.Equal(t, QualityLevel("poor"), QualityPoor)
	assert.Len(t, levels, 4)
}

func TestAgentRequest_Serialization(t *testing.T) {
	req := AgentRequest{
		TaskID:      "task-123",
		OrgID:       uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Environment: "production",
		Intent:      "Fix drift",
		Context: AgentContext{
			Platforms: []string{"aws"},
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed AgentRequest
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, req.TaskID, parsed.TaskID)
	assert.Equal(t, req.OrgID, parsed.OrgID)
	assert.Equal(t, req.Intent, parsed.Intent)
}

func TestAgentResponse_Serialization(t *testing.T) {
	resp := NewAgentResponse("task-123", "drift_agent", "1.0.0")
	resp.WithQuality(85, RiskLevelMedium)
	resp.WithPlan(map[string]any{"name": "test-plan"})

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed AgentResponse
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, resp.TaskID, parsed.TaskID)
	assert.Equal(t, resp.QualityScore, parsed.QualityScore)
}
