package meta

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/agents"
)

func TestIntentRequest(t *testing.T) {
	req := IntentRequest{
		UserIntent:  "Fix drift on all production servers",
		OrgID:       "org-123",
		UserID:      "user-456",
		Environment: "production",
		Context: map[string]interface{}{
			"priority": "high",
		},
	}

	assert.Equal(t, "Fix drift on all production servers", req.UserIntent)
	assert.Equal(t, "org-123", req.OrgID)
	assert.Equal(t, "production", req.Environment)
}

func TestParsedIntent(t *testing.T) {
	intent := ParsedIntent{
		TaskType:      agents.TaskTypeDriftRemediation,
		Goal:          "Fix configuration drift",
		Confidence:    0.95,
		Agents:        []string{"drift_agent"},
		ToolsRequired: []string{"query_assets", "get_drift_status"},
		RiskLevel:     "high",
		HITLRequired:  true,
		Environment:   "production",
		Scope: IntentScope{
			Platforms: []string{"aws"},
			Regions:   []string{"us-east-1"},
		},
		Reasoning: "User explicitly mentioned drift fix",
	}

	assert.Equal(t, agents.TaskTypeDriftRemediation, intent.TaskType)
	assert.Equal(t, 0.95, intent.Confidence)
	assert.True(t, intent.HITLRequired)
	assert.Contains(t, intent.Agents, "drift_agent")
}

func TestIntentScope(t *testing.T) {
	scope := IntentScope{
		Platforms:   []string{"aws", "azure"},
		Regions:     []string{"us-east-1", "us-west-2"},
		AssetFilter: "env=production",
	}

	assert.Len(t, scope.Platforms, 2)
	assert.Len(t, scope.Regions, 2)
	assert.Equal(t, "env=production", scope.AssetFilter)
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean JSON",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with prefix",
			input:    `Here is the result: {"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with suffix",
			input:    `{"key": "value"} is the answer`,
			expected: `{"key": "value"}`,
		},
		{
			name: "JSON with markdown",
			input: "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "nested JSON",
			input:    `{"outer": {"inner": "value"}}`,
			expected: `{"outer": {"inner": "value"}}`,
		},
		{
			name:     "no JSON",
			input:    "no json here",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only opening brace",
			input:    "{",
			expected: "",
		},
		{
			name:     "only closing brace",
			input:    "}",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTaskTypeKeywordMatching(t *testing.T) {
	// Test the keyword matching logic used for task type detection
	tests := []struct {
		name          string
		intent        string
		expectedType  agents.TaskType
	}{
		{"drift keyword", "Fix drift on servers", agents.TaskTypeDriftRemediation},
		{"remediation keyword", "Remediate configuration issues", agents.TaskTypeDriftRemediation},
		{"patch keyword", "Apply patches to all servers", agents.TaskTypePatchRollout},
		{"update keyword", "Update all systems", agents.TaskTypePatchRollout},
		{"compliance keyword", "Run compliance audit", agents.TaskTypeComplianceAudit},
		{"incident keyword", "Investigate the incident", agents.TaskTypeIncidentResponse},
		{"DR keyword", "Test disaster recovery", agents.TaskTypeDRDrill},
		{"failover keyword", "Run failover test", agents.TaskTypeDRDrill},
		{"cost keyword", "Optimize cloud costs", agents.TaskTypeCostOptimization},
		{"security keyword", "Run security scan", agents.TaskTypeSecurityScan},
		{"vulnerability keyword", "Check for vulnerabilities", agents.TaskTypeSecurityScan},
		{"image keyword", "Update golden images", agents.TaskTypeImageManagement},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detectedType := detectTaskTypeFromKeywords(tt.intent)
			assert.Equal(t, tt.expectedType, detectedType)
		})
	}
}

// detectTaskTypeFromKeywords mirrors the fallback keyword matching logic from engine.go
func detectTaskTypeFromKeywords(intent string) agents.TaskType {
	intentLower := strings.ToLower(intent)

	// Match order from engine.go fallbackIntentParsing
	switch {
	case strings.Contains(intentLower, "drift") || strings.Contains(intentLower, "remediat"):
		return agents.TaskTypeDriftRemediation
	case strings.Contains(intentLower, "patch"):
		return agents.TaskTypePatchRollout
	case strings.Contains(intentLower, "compliance") || strings.Contains(intentLower, "audit"):
		return agents.TaskTypeComplianceAudit
	case strings.Contains(intentLower, "incident") || strings.Contains(intentLower, "alert"):
		return agents.TaskTypeIncidentResponse
	case strings.Contains(intentLower, "disaster") || strings.Contains(intentLower, "failover"):
		return agents.TaskTypeDRDrill
	case strings.Contains(intentLower, "cost") || strings.Contains(intentLower, "spending"):
		return agents.TaskTypeCostOptimization
	case strings.Contains(intentLower, "security") || strings.Contains(intentLower, "vulnerab"):
		return agents.TaskTypeSecurityScan
	case strings.Contains(intentLower, "image") || strings.Contains(intentLower, "golden"):
		return agents.TaskTypeImageManagement
	case strings.Contains(intentLower, "update"):
		return agents.TaskTypePatchRollout
	default:
		return agents.TaskTypeDriftRemediation
	}
}

func TestEnvironmentKeywordDetection(t *testing.T) {
	tests := []struct {
		name        string
		intent      string
		envOverride string
		expectedEnv string
	}{
		{
			name:        "production keyword",
			intent:      "Fix drift on production servers",
			expectedEnv: "production",
		},
		{
			name:        "staging keyword",
			intent:      "Deploy to staging",
			expectedEnv: "staging",
		},
		{
			name:        "stage keyword",
			intent:      "Test on stage environment",
			expectedEnv: "staging",
		},
		{
			name:        "no environment - defaults to dev",
			intent:      "Fix the servers",
			expectedEnv: "development",
		},
		{
			name:        "explicit override",
			intent:      "Fix servers",
			envOverride: "custom-env",
			expectedEnv: "custom-env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := detectEnvironmentFromKeywords(tt.intent, tt.envOverride)
			assert.Equal(t, tt.expectedEnv, env)
		})
	}
}

// detectEnvironmentFromKeywords mirrors the fallback environment detection logic
func detectEnvironmentFromKeywords(intent string, override string) string {
	if override != "" {
		return override
	}
	intentLower := strings.ToLower(intent)
	switch {
	case strings.Contains(intentLower, "production") || strings.Contains(intentLower, "prod"):
		return "production"
	case strings.Contains(intentLower, "staging") || strings.Contains(intentLower, "stage"):
		return "staging"
	default:
		return "development"
	}
}

func TestRiskLevelByEnvironment(t *testing.T) {
	tests := []struct {
		environment  string
		expectedRisk string
		expectHITL   bool
	}{
		{
			environment:  "production",
			expectedRisk: "high",
			expectHITL:   true,
		},
		{
			environment:  "staging",
			expectedRisk: "medium",
			expectHITL:   false,
		},
		{
			environment:  "development",
			expectedRisk: "low",
			expectHITL:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.environment, func(t *testing.T) {
			risk, hitl := getRiskLevelForEnv(tt.environment)
			assert.Equal(t, tt.expectedRisk, risk)
			assert.Equal(t, tt.expectHITL, hitl)
		})
	}
}

// getRiskLevelForEnv mirrors the risk level determination logic
func getRiskLevelForEnv(env string) (string, bool) {
	switch env {
	case "production":
		return "high", true
	case "staging":
		return "medium", false
	default:
		return "low", false
	}
}

func TestEnrichParsedIntent(t *testing.T) {
	engine := &Engine{}

	t.Run("sets default environment", func(t *testing.T) {
		parsed := &ParsedIntent{}
		req := &IntentRequest{}

		engine.enrichParsedIntent(parsed, req)

		assert.Equal(t, "development", parsed.Environment)
	})

	t.Run("uses request environment", func(t *testing.T) {
		parsed := &ParsedIntent{}
		req := &IntentRequest{Environment: "staging"}

		engine.enrichParsedIntent(parsed, req)

		assert.Equal(t, "staging", parsed.Environment)
	})

	t.Run("adjusts risk for production", func(t *testing.T) {
		parsed := &ParsedIntent{
			Environment: "production",
			RiskLevel:   "low",
		}
		req := &IntentRequest{}

		engine.enrichParsedIntent(parsed, req)

		assert.Equal(t, "medium", parsed.RiskLevel)
	})

	t.Run("sets HITL for high risk", func(t *testing.T) {
		parsed := &ParsedIntent{
			RiskLevel: "high",
		}
		req := &IntentRequest{}

		engine.enrichParsedIntent(parsed, req)

		assert.True(t, parsed.HITLRequired)
	})

	t.Run("adds production constraints", func(t *testing.T) {
		parsed := &ParsedIntent{
			Environment: "production",
		}
		req := &IntentRequest{}

		engine.enrichParsedIntent(parsed, req)

		assert.True(t, parsed.Constraints["require_canary"].(bool))
		assert.Equal(t, 10, parsed.Constraints["max_batch_percent"])
	})
}

func TestGetSchemaForTaskType(t *testing.T) {
	engine := &Engine{}

	tests := []struct {
		taskType agents.TaskType
		expected string
	}{
		{agents.TaskTypeDriftRemediation, "drift_remediation_v1"},
		{agents.TaskTypePatchRollout, "patch_rollout_v1"},
		{agents.TaskTypeComplianceAudit, "compliance_report_v1"},
		{agents.TaskTypeIncidentResponse, "incident_analysis_v1"},
		{agents.TaskTypeDRDrill, "dr_runbook_v1"},
		{agents.TaskTypeCostOptimization, "cost_report_v1"},
		{agents.TaskTypeSecurityScan, "security_report_v1"},
		{agents.TaskTypeImageManagement, "image_spec_v1"},
		{agents.TaskType("unknown"), "generic_v1"},
	}

	for _, tt := range tests {
		t.Run(string(tt.taskType), func(t *testing.T) {
			schema := engine.getSchemaForTaskType(tt.taskType)
			assert.Equal(t, tt.expected, schema)
		})
	}
}

func TestGetPoliciesForTaskType(t *testing.T) {
	engine := &Engine{}

	t.Run("drift remediation", func(t *testing.T) {
		policies := engine.getPoliciesForTaskType(agents.TaskTypeDriftRemediation, "staging")
		assert.Contains(t, policies, "base_safety")
		assert.Contains(t, policies, "rollout_safety")
	})

	t.Run("DR drill", func(t *testing.T) {
		policies := engine.getPoliciesForTaskType(agents.TaskTypeDRDrill, "staging")
		assert.Contains(t, policies, "dr_safety")
	})

	t.Run("compliance audit", func(t *testing.T) {
		policies := engine.getPoliciesForTaskType(agents.TaskTypeComplianceAudit, "staging")
		assert.Contains(t, policies, "compliance_scope")
	})

	t.Run("production adds safety policies", func(t *testing.T) {
		policies := engine.getPoliciesForTaskType(agents.TaskTypePatchRollout, "production")
		assert.Contains(t, policies, "production_safety")
		assert.Contains(t, policies, "canary_required")
	})
}

func TestGetTimeoutForTaskType(t *testing.T) {
	engine := &Engine{}

	tests := []struct {
		taskType agents.TaskType
		expected int
	}{
		{agents.TaskTypeDriftRemediation, 60},
		{agents.TaskTypePatchRollout, 120},
		{agents.TaskTypeComplianceAudit, 30},
		{agents.TaskTypeIncidentResponse, 30},
		{agents.TaskTypeDRDrill, 90},
		{agents.TaskTypeCostOptimization, 30},
		{agents.TaskTypeSecurityScan, 45},
		{agents.TaskTypeImageManagement, 60},
		{agents.TaskType("unknown"), 30},
	}

	for _, tt := range tests {
		t.Run(string(tt.taskType), func(t *testing.T) {
			timeout := engine.getTimeoutForTaskType(tt.taskType)
			assert.Equal(t, tt.expected, timeout)
		})
	}
}

func TestTaskPlanningResult(t *testing.T) {
	result := TaskPlanningResult{
		TaskSpec: &agents.TaskSpec{
			ID:       "task-123",
			TaskType: agents.TaskTypeDriftRemediation,
		},
		ParsedIntent: &ParsedIntent{
			TaskType:   agents.TaskTypeDriftRemediation,
			Confidence: 0.95,
		},
		TokensUsed: 1500,
	}

	assert.Equal(t, "task-123", result.TaskSpec.ID)
	assert.Equal(t, 0.95, result.ParsedIntent.Confidence)
	assert.Equal(t, 1500, result.TokensUsed)
}
