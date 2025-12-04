package llmjson

import (
	"testing"
)

// =============================================================================
// Test Types
// =============================================================================

type SimplePlan struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
}

type DriftPlan struct {
	Plan struct {
		Name   string `json:"name"`
		Phases []struct {
			Name         string `json:"name"`
			BatchPercent int    `json:"batch_percent"`
			WaitMinutes  int    `json:"wait_minutes"`
			HealthCheck  bool   `json:"health_check"`
		} `json:"phases"`
		RollbackPolicy struct {
			Type          string  `json:"type"`
			Threshold     float64 `json:"threshold"`
			WindowMinutes int     `json:"window_minutes"`
		} `json:"rollback_policy"`
	} `json:"plan"`
	Summary        string `json:"summary"`
	RiskAssessment struct {
		Level       string   `json:"level"`
		Factors     []string `json:"factors"`
		Mitigations []string `json:"mitigations"`
	} `json:"risk_assessment"`
}

// =============================================================================
// ExtractJSON Tests
// =============================================================================

func TestExtractJSON_DirectParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantVer  int
	}{
		{
			name:     "clean JSON object",
			input:    `{"name": "test-plan", "version": 1}`,
			wantName: "test-plan",
			wantVer:  1,
		},
		{
			name:     "JSON with whitespace",
			input:    `  { "name" : "whitespace-plan" , "version" : 2 }  `,
			wantName: "whitespace-plan",
			wantVer:  2,
		},
		{
			name: "multiline JSON",
			input: `{
				"name": "multiline-plan",
				"version": 3
			}`,
			wantName: "multiline-plan",
			wantVer:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractJSON[SimplePlan](tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Method != ParseMethodDirect {
				t.Errorf("expected method %s, got %s", ParseMethodDirect, result.Method)
			}

			if result.Value.Name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, result.Value.Name)
			}

			if result.Value.Version != tt.wantVer {
				t.Errorf("expected version %d, got %d", tt.wantVer, result.Value.Version)
			}

			if result.Warning != "" {
				t.Errorf("expected no warning, got %q", result.Warning)
			}
		})
	}
}

func TestExtractJSON_MarkdownCodeBlocks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantVer  int
	}{
		{
			name: "json code block with language tag",
			input: "Here's the plan:\n\n```json\n{\"name\": \"markdown-plan\", \"version\": 1}\n```\n\nLet me know!",
			wantName: "markdown-plan",
			wantVer:  1,
		},
		{
			name: "code block without language tag",
			input: "Here's the plan:\n\n```\n{\"name\": \"no-tag-plan\", \"version\": 2}\n```",
			wantName: "no-tag-plan",
			wantVer:  2,
		},
		{
			name: "multiple code blocks (first is used)",
			input: "First:\n```json\n{\"name\": \"first-plan\", \"version\": 3}\n```\n\nSecond:\n```json\n{\"name\": \"second-plan\", \"version\": 4}\n```",
			wantName: "first-plan",
			wantVer:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractJSON[SimplePlan](tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Method != ParseMethodExtracted {
				t.Errorf("expected method %s, got %s", ParseMethodExtracted, result.Method)
			}

			if result.Value.Name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, result.Value.Name)
			}

			if result.Value.Version != tt.wantVer {
				t.Errorf("expected version %d, got %d", tt.wantVer, result.Value.Version)
			}

			if result.Warning == "" {
				t.Error("expected warning for extracted JSON")
			}
		})
	}
}

func TestExtractJSON_SurroundingText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantVer  int
	}{
		{
			name:     "JSON with prefix text",
			input:    "Based on my analysis, here is the plan: {\"name\": \"prefixed-plan\", \"version\": 1}",
			wantName: "prefixed-plan",
			wantVer:  1,
		},
		{
			name:     "JSON with suffix text",
			input:    "{\"name\": \"suffixed-plan\", \"version\": 2} Please review this plan.",
			wantName: "suffixed-plan",
			wantVer:  2,
		},
		{
			name:     "JSON with both prefix and suffix",
			input:    "Analysis: {\"name\": \"wrapped-plan\", \"version\": 3} End of analysis.",
			wantName: "wrapped-plan",
			wantVer:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractJSON[SimplePlan](tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Method != ParseMethodExtracted {
				t.Errorf("expected method %s, got %s", ParseMethodExtracted, result.Method)
			}

			if result.Value.Name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, result.Value.Name)
			}

			if result.Value.Version != tt.wantVer {
				t.Errorf("expected version %d, got %d", tt.wantVer, result.Value.Version)
			}
		})
	}
}

func TestExtractJSON_LenientParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantVer  int
	}{
		{
			name:     "trailing comma in object",
			input:    `{"name": "trailing-comma", "version": 1,}`,
			wantName: "trailing-comma",
			wantVer:  1,
		},
		{
			name:     "single quotes instead of double",
			input:    `{'name': 'single-quotes', 'version': 2}`,
			wantName: "single-quotes",
			wantVer:  2,
		},
		{
			name:     "trailing comma in array",
			input:    `{"name": "array-comma", "version": 3, "items": [1, 2,]}`,
			wantName: "array-comma",
			wantVer:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractJSON[SimplePlan](tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Method != ParseMethodLenient {
				t.Errorf("expected method %s, got %s", ParseMethodLenient, result.Method)
			}

			if result.Value.Name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, result.Value.Name)
			}

			if result.Value.Version != tt.wantVer {
				t.Errorf("expected version %d, got %d", tt.wantVer, result.Value.Version)
			}

			if result.Warning == "" {
				t.Error("expected warning for lenient parsing")
			}
		})
	}
}

func TestExtractJSON_ComplexPlan(t *testing.T) {
	input := `{
		"plan": {
			"name": "drift-remediation-plan",
			"phases": [
				{"name": "canary", "batch_percent": 5, "wait_minutes": 10, "health_check": true},
				{"name": "rollout", "batch_percent": 100, "wait_minutes": 0, "health_check": true}
			],
			"rollback_policy": {
				"type": "automatic",
				"threshold": 0.05,
				"window_minutes": 15
			}
		},
		"summary": "Remediate drift for production assets",
		"risk_assessment": {
			"level": "medium",
			"factors": ["production environment"],
			"mitigations": ["canary deployment"]
		}
	}`

	result, err := ExtractJSON[DriftPlan](input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Method != ParseMethodDirect {
		t.Errorf("expected direct parsing, got %s", result.Method)
	}

	if result.Value.Plan.Name != "drift-remediation-plan" {
		t.Errorf("expected plan name 'drift-remediation-plan', got %q", result.Value.Plan.Name)
	}

	if len(result.Value.Plan.Phases) != 2 {
		t.Errorf("expected 2 phases, got %d", len(result.Value.Plan.Phases))
	}

	if result.Value.Plan.Phases[0].Name != "canary" {
		t.Errorf("expected first phase 'canary', got %q", result.Value.Plan.Phases[0].Name)
	}

	if result.Value.Plan.RollbackPolicy.Threshold != 0.05 {
		t.Errorf("expected threshold 0.05, got %f", result.Value.Plan.RollbackPolicy.Threshold)
	}

	if result.Value.RiskAssessment.Level != "medium" {
		t.Errorf("expected risk level 'medium', got %q", result.Value.RiskAssessment.Level)
	}
}

func TestExtractJSON_Failures(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "plain text no JSON",
			input: "This is just plain text without any JSON.",
		},
		{
			name:  "truncated JSON",
			input: `{"name": "truncated", "version": `,
		},
		{
			name:  "invalid JSON structure",
			input: `{"name": "invalid" "version": 1}`, // missing comma
		},
		{
			name:  "wrong type for required field",
			input: `{"name": 123, "version": "not-a-number"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExtractJSON[SimplePlan](tt.input)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

// =============================================================================
// MustExtractJSON Tests
// =============================================================================

func TestMustExtractJSON_Success(t *testing.T) {
	input := `{"name": "must-plan", "version": 1}`
	result := MustExtractJSON[SimplePlan](input)

	if result.Name != "must-plan" {
		t.Errorf("expected name 'must-plan', got %q", result.Name)
	}
}

func TestMustExtractJSON_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic, but didn't get one")
		}
	}()

	MustExtractJSON[SimplePlan]("invalid json")
}

// =============================================================================
// extractFromCodeBlock Tests
// =============================================================================

func TestExtractFromCodeBlock(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "json tag",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "no tag",
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "no code block",
			input:    `{"key": "value"}`,
			expected: "",
		},
		{
			name:     "empty code block",
			input:    "```json\n```",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFromCodeBlock(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// =============================================================================
// findJSONSegment Tests
// =============================================================================

func TestFindJSONSegment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "object",
			input:    `prefix {"key": "value"} suffix`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "array",
			input:    `prefix [1, 2, 3] suffix`,
			expected: `[1, 2, 3]`,
		},
		{
			name:     "nested object",
			input:    `prefix {"outer": {"inner": "value"}} suffix`,
			expected: `{"outer": {"inner": "value"}}`,
		},
		{
			name:     "object with string containing braces",
			input:    `prefix {"key": "value with { and }"} suffix`,
			expected: `{"key": "value with { and }"}`,
		},
		{
			name:     "no JSON",
			input:    "just plain text",
			expected: "",
		},
		{
			name:     "unclosed brace",
			input:    `{"key": "value"`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findJSONSegment(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// =============================================================================
// attemptJSONRecovery Tests
// =============================================================================

func TestAttemptJSONRecovery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantJSON bool
	}{
		{
			name:     "trailing comma",
			input:    `{"key": "value",}`,
			wantJSON: true,
		},
		{
			name:     "single quotes",
			input:    `{'key': 'value'}`,
			wantJSON: true,
		},
		{
			name:     "line comment",
			input:    `{"key": "value"} // comment`,
			wantJSON: true,
		},
		{
			name:     "block comment",
			input:    `{"key": /* comment */ "value"}`,
			wantJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := attemptJSONRecovery(tt.input)
			if tt.wantJSON && !IsValidJSON(result) {
				t.Errorf("expected valid JSON after recovery, got %q", result)
			}
		})
	}
}

// =============================================================================
// IsValidJSON Tests
// =============================================================================

func TestIsValidJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid object",
			input:    `{"key": "value"}`,
			expected: true,
		},
		{
			name:     "valid array",
			input:    `[1, 2, 3]`,
			expected: true,
		},
		{
			name:     "valid string",
			input:    `"hello"`,
			expected: true,
		},
		{
			name:     "valid number",
			input:    `42`,
			expected: true,
		},
		{
			name:     "valid null",
			input:    `null`,
			expected: true,
		},
		{
			name:     "invalid",
			input:    `{invalid}`,
			expected: false,
		},
		{
			name:     "empty",
			input:    ``,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidJSON(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// =============================================================================
// PrettyPrint Tests
// =============================================================================

func TestPrettyPrint(t *testing.T) {
	input := map[string]any{
		"name":    "test",
		"version": 1,
	}

	result, err := PrettyPrint(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}

	// Should be indented
	if !IsValidJSON(result) {
		t.Error("expected valid JSON output")
	}
}

// =============================================================================
// ExtractField Tests
// =============================================================================

func TestExtractField(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		field    string
		expected string
		found    bool
	}{
		{
			name:     "simple field",
			input:    `{"name": "test-plan", "version": 1}`,
			field:    "name",
			expected: "test-plan",
			found:    true,
		},
		{
			name:     "nested field (not supported)",
			input:    `{"plan": {"name": "nested"}}`,
			field:    "name",
			expected: "nested",
			found:    true,
		},
		{
			name:     "field not found",
			input:    `{"name": "test"}`,
			field:    "version",
			expected: "",
			found:    false,
		},
		{
			name:     "numeric value (not extracted as string)",
			input:    `{"name": "test", "count": 42}`,
			field:    "count",
			expected: "",
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := ExtractField(tt.input, tt.field)
			if found != tt.found {
				t.Errorf("expected found=%v, got %v", tt.found, found)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// =============================================================================
// ExtractIntField Tests
// =============================================================================

func TestExtractIntField(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		field    string
		expected int
		found    bool
	}{
		{
			name:     "positive integer",
			input:    `{"count": 42}`,
			field:    "count",
			expected: 42,
			found:    true,
		},
		{
			name:     "negative integer",
			input:    `{"offset": -10}`,
			field:    "offset",
			expected: -10,
			found:    true,
		},
		{
			name:     "zero",
			input:    `{"count": 0}`,
			field:    "count",
			expected: 0,
			found:    true,
		},
		{
			name:     "string value",
			input:    `{"count": "42"}`,
			field:    "count",
			expected: 0,
			found:    false,
		},
		{
			name:     "field not found",
			input:    `{"name": "test"}`,
			field:    "count",
			expected: 0,
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := ExtractIntField(tt.input, tt.field)
			if found != tt.found {
				t.Errorf("expected found=%v, got %v", tt.found, found)
			}
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// =============================================================================
// ExtractFloatField Tests
// =============================================================================

func TestExtractFloatField(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		field    string
		expected float64
		found    bool
	}{
		{
			name:     "float with decimal",
			input:    `{"score": 85.5}`,
			field:    "score",
			expected: 85.5,
			found:    true,
		},
		{
			name:     "integer as float",
			input:    `{"score": 100}`,
			field:    "score",
			expected: 100.0,
			found:    true,
		},
		{
			name:     "negative float",
			input:    `{"delta": -0.05}`,
			field:    "delta",
			expected: -0.05,
			found:    true,
		},
		{
			name:     "field not found",
			input:    `{"name": "test"}`,
			field:    "score",
			expected: 0,
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := ExtractFloatField(tt.input, tt.field)
			if found != tt.found {
				t.Errorf("expected found=%v, got %v", tt.found, found)
			}
			// Use approximate comparison for floats
			if found && (result-tt.expected > 0.0001 || tt.expected-result > 0.0001) {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

// =============================================================================
// Real-World LLM Response Simulation Tests
// =============================================================================

func TestExtractJSON_RealWorldLLMResponses(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantMethod  ParseMethod
		wantSuccess bool
	}{
		{
			name: "Claude-style response with explanation",
			input: `I'll create a drift remediation plan based on your requirements.

Here's the plan:

` + "```json" + `
{
  "name": "drift-plan",
  "version": 1
}
` + "```" + `

This plan includes a canary phase to minimize risk.`,
			wantMethod:  ParseMethodExtracted,
			wantSuccess: true,
		},
		{
			name: "GPT-style response with thinking",
			input: `Let me analyze the drift situation...

Based on my analysis:
{"name": "gpt-plan", "version": 2}

I recommend proceeding with caution.`,
			wantMethod:  ParseMethodExtracted,
			wantSuccess: true,
		},
		{
			name: "direct JSON response (well-behaved model)",
			input: `{"name": "direct-plan", "version": 3}`,
			wantMethod:  ParseMethodDirect,
			wantSuccess: true,
		},
		{
			name: "response with trailing text after JSON",
			input: `{"name": "trailing-plan", "version": 4}

Note: This plan requires approval before execution.`,
			wantMethod:  ParseMethodExtracted,
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractJSON[SimplePlan](tt.input)

			if tt.wantSuccess {
				if err != nil {
					t.Fatalf("expected success, got error: %v", err)
				}
				if result.Method != tt.wantMethod {
					t.Errorf("expected method %s, got %s", tt.wantMethod, result.Method)
				}
			} else {
				if err == nil {
					t.Error("expected error, got nil")
				}
			}
		})
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkExtractJSON_Direct(b *testing.B) {
	input := `{"name": "benchmark-plan", "version": 1}`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ExtractJSON[SimplePlan](input)
	}
}

func BenchmarkExtractJSON_Extracted(b *testing.B) {
	input := "Here's the plan:\n```json\n{\"name\": \"benchmark-plan\", \"version\": 1}\n```"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ExtractJSON[SimplePlan](input)
	}
}

func BenchmarkExtractJSON_Lenient(b *testing.B) {
	input := `{"name": "benchmark-plan", "version": 1,}`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ExtractJSON[SimplePlan](input)
	}
}
