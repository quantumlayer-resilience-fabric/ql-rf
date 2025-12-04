package agents

import (
	"encoding/json"
	"testing"

	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llmjson"
)

func TestComplianceAgent_PlanGeneration(t *testing.T) {
	tests := []struct {
		name          string
		response      string
		expectSuccess bool
		expectMethod  llmjson.ParseMethod
	}{
		{
			name:          "clean JSON compliance report",
			response:      CompliancePlanFixture(),
			expectSuccess: true,
			expectMethod:  llmjson.ParseMethodDirect,
		},
		{
			name: "compliance report wrapped in markdown",
			response: "Here's the compliance audit report:\n\n```json\n" + CompliancePlanFixture() + "\n```\n",
			expectSuccess: true,
			expectMethod:  llmjson.ParseMethodExtracted,
		},
		{
			name: "compliance report with prefix text",
			response: `After analyzing your infrastructure against CIS benchmarks, here are the findings:

{"plan": {"name": "compliance-audit", "framework": "CIS", "score": 92.5}, "summary": "CIS audit completed"}`,
			expectSuccess: true,
			expectMethod:  llmjson.ParseMethodExtracted,
		},
		{
			name:          "truncated compliance report",
			response:      `{"plan": {"name": "compliance-audit", "controls": [{"id": "CIS-1.1", "sta`,
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

func TestComplianceAgent_FrameworkParsing(t *testing.T) {
	tests := []struct {
		name               string
		goal               string
		intent             string
		expectedFrameworks []string
	}{
		{
			name:               "CIS mentioned",
			goal:               "Run a CIS compliance audit",
			intent:             "",
			expectedFrameworks: []string{"CIS"},
		},
		{
			name:               "SOC2 mentioned",
			goal:               "Generate SOC2 compliance report",
			intent:             "",
			expectedFrameworks: []string{"SOC2"},
		},
		{
			name:               "SOC 2 with space",
			goal:               "Run SOC 2 audit",
			intent:             "",
			expectedFrameworks: []string{"SOC2"},
		},
		{
			name:               "multiple frameworks",
			goal:               "Run CIS and SOC2 compliance audit",
			intent:             "also check HIPAA",
			expectedFrameworks: []string{"CIS", "SOC2", "HIPAA"},
		},
		{
			name:               "PCI-DSS",
			goal:               "PCI-DSS compliance check",
			intent:             "",
			expectedFrameworks: []string{"PCI-DSS"},
		},
		{
			name:               "NIST framework",
			goal:               "Run NIST compliance audit",
			intent:             "",
			expectedFrameworks: []string{"NIST"},
		},
		{
			name:               "ISO27001",
			goal:               "ISO27001 certification audit",
			intent:             "",
			expectedFrameworks: []string{"ISO27001"},
		},
		{
			name:               "no framework defaults to CIS",
			goal:               "Run compliance audit",
			intent:             "",
			expectedFrameworks: []string{"CIS"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &ComplianceAgent{}
			frameworks := agent.parseFrameworks(tt.goal, tt.intent)

			// Check that expected frameworks are present
			for _, expected := range tt.expectedFrameworks {
				found := false
				for _, f := range frameworks {
					if f == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected framework %s not found in %v", expected, frameworks)
				}
			}
		})
	}
}

func TestComplianceAgent_DefaultReport(t *testing.T) {
	agent := &ComplianceAgent{}

	controlResults := map[string]interface{}{
		"CIS": map[string]interface{}{
			"passed":   10,
			"failed":   2,
			"warnings": 1,
		},
	}

	report := agent.defaultComplianceReport([]string{"CIS"}, controlResults)

	// Verify report structure
	if report["executive_summary"] == nil {
		t.Error("expected executive_summary in report")
	}

	if report["frameworks_audited"] == nil {
		t.Error("expected frameworks_audited in report")
	}

	frameworks, ok := report["frameworks_audited"].([]string)
	if !ok {
		t.Fatal("expected frameworks_audited to be []string")
	}

	if len(frameworks) != 1 || frameworks[0] != "CIS" {
		t.Errorf("expected frameworks_audited to be [CIS], got %v", frameworks)
	}

	if report["recommendations"] == nil {
		t.Error("expected recommendations in report")
	}
}

func TestComplianceAgent_ComplianceScoreCalculation(t *testing.T) {
	tests := []struct {
		name            string
		passed          int
		failed          int
		expectedScore   int
		expectedLevel   string
	}{
		{
			name:          "perfect compliance",
			passed:        100,
			failed:        0,
			expectedScore: 100,
			expectedLevel: "compliant",
		},
		{
			name:          "high compliance",
			passed:        95,
			failed:        5,
			expectedScore: 95,
			expectedLevel: "compliant",
		},
		{
			name:          "minor issues",
			passed:        85,
			failed:        15,
			expectedScore: 85,
			expectedLevel: "minor_issues",
		},
		{
			name:          "needs attention",
			passed:        70,
			failed:        30,
			expectedScore: 70,
			expectedLevel: "needs_attention",
		},
		{
			name:          "critical",
			passed:        50,
			failed:        50,
			expectedScore: 50,
			expectedLevel: "critical",
		},
		{
			name:          "no controls",
			passed:        0,
			failed:        0,
			expectedScore: 0,
			expectedLevel: "critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalControls := tt.passed + tt.failed
			complianceScore := 0
			if totalControls > 0 {
				complianceScore = (tt.passed * 100) / totalControls
			}

			if complianceScore != tt.expectedScore {
				t.Errorf("expected score %d, got %d", tt.expectedScore, complianceScore)
			}

			complianceLevel := "critical"
			if complianceScore >= 95 {
				complianceLevel = "compliant"
			} else if complianceScore >= 80 {
				complianceLevel = "minor_issues"
			} else if complianceScore >= 60 {
				complianceLevel = "needs_attention"
			}

			if complianceLevel != tt.expectedLevel {
				t.Errorf("expected level %s, got %s", tt.expectedLevel, complianceLevel)
			}
		})
	}
}

func TestComplianceAgent_FixtureValidity(t *testing.T) {
	t.Run("CompliancePlanFixture is valid JSON", func(t *testing.T) {
		var v map[string]any
		if err := json.Unmarshal([]byte(CompliancePlanFixture()), &v); err != nil {
			t.Errorf("fixture should be valid JSON: %v", err)
		}

		if v["plan"] == nil {
			t.Error("expected plan field")
		}
		if v["summary"] == nil {
			t.Error("expected summary field")
		}
	})

	t.Run("CompliancePlanFixture has required structure", func(t *testing.T) {
		result, err := llmjson.ExtractJSON[map[string]any](CompliancePlanFixture())
		if err != nil {
			t.Fatalf("failed to parse fixture: %v", err)
		}

		plan := result.Value["plan"].(map[string]any)
		if plan["framework"] == nil {
			t.Error("expected plan.framework")
		}
		if plan["controls"] == nil {
			t.Error("expected plan.controls")
		}
		if plan["score"] == nil {
			t.Error("expected plan.score")
		}
	})
}

func TestComplianceAgent_QualityScoring(t *testing.T) {
	tests := []struct {
		name            string
		complianceScore int
		parseMethod     llmjson.ParseMethod
		expectedHITL    bool
	}{
		{
			name:            "high compliance direct parse - no HITL",
			complianceScore: 95,
			parseMethod:     llmjson.ParseMethodDirect,
			expectedHITL:    false,
		},
		{
			name:            "low compliance direct parse - requires HITL",
			complianceScore: 50,
			parseMethod:     llmjson.ParseMethodDirect,
			expectedHITL:    true,
		},
		{
			name:            "high compliance extracted parse - requires HITL",
			complianceScore: 95,
			parseMethod:     llmjson.ParseMethodExtracted,
			expectedHITL:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewAgentResponse("task-123", "compliance_agent", "1.0.0")

			// Map compliance score to risk level
			riskLevel := RiskLevelLow
			if tt.complianceScore < 60 {
				riskLevel = RiskLevelCritical
			} else if tt.complianceScore < 80 {
				riskLevel = RiskLevelHigh
			} else if tt.complianceScore < 95 {
				riskLevel = RiskLevelMedium
			}

			resp.RiskLevel = riskLevel
			resp.QualityScore = float64(tt.complianceScore)

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

func TestComplianceAgent_ControlResultAggregation(t *testing.T) {
	t.Run("aggregates results from multiple frameworks", func(t *testing.T) {
		controlResults := map[string]interface{}{
			"CIS": map[string]interface{}{
				"passed":   float64(10),
				"failed":   float64(2),
				"warnings": float64(1),
			},
			"SOC2": map[string]interface{}{
				"passed":   float64(15),
				"failed":   float64(3),
				"warnings": float64(2),
			},
		}

		totalPassed := 0
		totalFailed := 0
		totalWarnings := 0

		for _, result := range controlResults {
			if rm, ok := result.(map[string]interface{}); ok {
				if p, ok := rm["passed"].(float64); ok {
					totalPassed += int(p)
				}
				if f, ok := rm["failed"].(float64); ok {
					totalFailed += int(f)
				}
				if w, ok := rm["warnings"].(float64); ok {
					totalWarnings += int(w)
				}
			}
		}

		if totalPassed != 25 {
			t.Errorf("expected totalPassed=25, got %d", totalPassed)
		}
		if totalFailed != 5 {
			t.Errorf("expected totalFailed=5, got %d", totalFailed)
		}
		if totalWarnings != 3 {
			t.Errorf("expected totalWarnings=3, got %d", totalWarnings)
		}
	})
}
