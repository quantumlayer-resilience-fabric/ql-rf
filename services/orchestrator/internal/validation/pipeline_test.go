package validation

import (
	"context"
	"testing"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPipeline(t *testing.T) {
	log := logger.New("debug", "text")
	cfg := config.OPAConfig{Enabled: false}

	pipeline, err := NewPipeline(cfg, log)
	require.NoError(t, err)
	assert.NotNil(t, pipeline)
	assert.NotNil(t, pipeline.schemaCompiler)
	assert.NotNil(t, pipeline.compiledSchemas)
}

func TestPipeline_RegisterSchema(t *testing.T) {
	log := logger.New("debug", "text")
	cfg := config.OPAConfig{Enabled: false}
	pipeline, _ := NewPipeline(cfg, log)

	t.Run("register valid schema", func(t *testing.T) {
		schema := map[string]interface{}{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type":    "object",
			"required": []interface{}{"name", "value"},
			"properties": map[string]interface{}{
				"name":  map[string]interface{}{"type": "string"},
				"value": map[string]interface{}{"type": "number"},
			},
		}

		err := pipeline.RegisterSchema("test_schema", schema)
		require.NoError(t, err)

		// Verify schema was registered
		compiled, ok := pipeline.GetSchema("test_schema")
		assert.True(t, ok)
		assert.NotNil(t, compiled)
	})

	t.Run("list schemas", func(t *testing.T) {
		schemas := pipeline.ListSchemas()
		assert.Contains(t, schemas, "test_schema")
	})
}

func TestPipeline_LoadDefaultSchemas(t *testing.T) {
	log := logger.New("debug", "text")
	cfg := config.OPAConfig{Enabled: false}
	pipeline, _ := NewPipeline(cfg, log)

	err := pipeline.LoadSchemas("/nonexistent/path")
	require.NoError(t, err)

	// Check that default schemas were loaded
	schemas := pipeline.ListSchemas()
	assert.Contains(t, schemas, "drift_remediation_v1")
	assert.Contains(t, schemas, "patch_rollout_v1")
	assert.Contains(t, schemas, "compliance_report_v1")
	assert.Contains(t, schemas, "dr_runbook_v1")
	assert.Contains(t, schemas, "execution_plan_v1")
}

func TestPipeline_ValidateSchema(t *testing.T) {
	log := logger.New("debug", "text")
	cfg := config.OPAConfig{Enabled: true}
	pipeline, _ := NewPipeline(cfg, log)

	// Load default schemas
	pipeline.LoadSchemas("/nonexistent")

	t.Run("valid drift remediation data", func(t *testing.T) {
		data := map[string]interface{}{
			"summary": "Fix drift on web servers",
			"phases": []interface{}{
				map[string]interface{}{
					"name": "Canary",
					"assets": []interface{}{
						map[string]interface{}{"id": "asset-1"},
					},
				},
			},
		}

		result, err := pipeline.Validate(context.Background(), &ValidationRequest{
			Data:        data,
			Schema:      "drift_remediation_v1",
			Environment: "staging",
		})
		require.NoError(t, err)
		if !result.Valid {
			t.Logf("Validation errors: %+v", result.Errors)
		}
		assert.True(t, result.Valid)
	})

	t.Run("invalid drift remediation - missing summary", func(t *testing.T) {
		data := map[string]interface{}{
			"phases": []interface{}{
				map[string]interface{}{
					"name":   "Canary",
					"assets": []interface{}{},
				},
			},
		}

		result, err := pipeline.Validate(context.Background(), &ValidationRequest{
			Data:        data,
			Schema:      "drift_remediation_v1",
			Environment: "staging",
		})
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)

		// Check that error mentions missing field
		hasSchemaError := false
		for _, e := range result.Errors {
			if e.Code == "SCHEMA_INVALID" {
				hasSchemaError = true
				break
			}
		}
		assert.True(t, hasSchemaError, "expected schema validation error")
	})

	t.Run("invalid drift remediation - wrong type", func(t *testing.T) {
		data := map[string]interface{}{
			"summary": 123, // Should be string
			"phases":  []interface{}{},
		}

		result, err := pipeline.Validate(context.Background(), &ValidationRequest{
			Data:        data,
			Schema:      "drift_remediation_v1",
			Environment: "staging",
		})
		require.NoError(t, err)
		assert.False(t, result.Valid)
	})

	t.Run("valid compliance report", func(t *testing.T) {
		data := map[string]interface{}{
			"summary": "Monthly compliance check",
			"controls": []interface{}{
				map[string]interface{}{
					"id":     "CIS-1.1",
					"name":   "Password Policy",
					"status": "passed",
				},
			},
			"findings":       []interface{}{},
			"overall_status": "compliant",
		}

		result, err := pipeline.Validate(context.Background(), &ValidationRequest{
			Data:        data,
			Schema:      "compliance_report_v1",
			Environment: "staging",
		})
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("invalid compliance - wrong status enum", func(t *testing.T) {
		data := map[string]interface{}{
			"summary": "Monthly compliance check",
			"controls": []interface{}{
				map[string]interface{}{
					"id":     "CIS-1.1",
					"status": "unknown", // Invalid enum value
				},
			},
			"findings": []interface{}{},
		}

		result, err := pipeline.Validate(context.Background(), &ValidationRequest{
			Data:        data,
			Schema:      "compliance_report_v1",
			Environment: "staging",
		})
		require.NoError(t, err)
		assert.False(t, result.Valid)
	})

	t.Run("valid DR runbook", func(t *testing.T) {
		data := map[string]interface{}{
			"summary": "Database failover runbook",
			"steps": []interface{}{
				map[string]interface{}{
					"order":  1,
					"action": "Stop writes to primary",
				},
				map[string]interface{}{
					"order":  2,
					"action": "Promote replica",
				},
			},
			"recovery_objectives": map[string]interface{}{
				"rto": "15m",
				"rpo": "5m",
			},
		}

		result, err := pipeline.Validate(context.Background(), &ValidationRequest{
			Data:        data,
			Schema:      "dr_runbook_v1",
			Environment: "staging",
		})
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("schema not found - skips validation", func(t *testing.T) {
		data := map[string]interface{}{"foo": "bar"}

		result, err := pipeline.Validate(context.Background(), &ValidationRequest{
			Data:        data,
			Schema:      "nonexistent_schema",
			Environment: "staging",
		})
		require.NoError(t, err)
		assert.True(t, result.Valid) // No error when schema not found
	})
}

func TestPipeline_SafetyChecks(t *testing.T) {
	log := logger.New("debug", "text")
	cfg := config.OPAConfig{Enabled: true}
	pipeline, _ := NewPipeline(cfg, log)

	t.Run("detects dangerous rm -rf", func(t *testing.T) {
		data := map[string]interface{}{
			"command": "rm -rf /",
		}

		result, err := pipeline.Validate(context.Background(), &ValidationRequest{
			Data:        data,
			Environment: "staging",
		})
		require.NoError(t, err)
		assert.False(t, result.Valid)

		hasDangerousPattern := false
		for _, e := range result.Errors {
			if e.Code == "DANGEROUS_PATTERN" {
				hasDangerousPattern = true
				break
			}
		}
		assert.True(t, hasDangerousPattern)
	})

	t.Run("detects DROP DATABASE", func(t *testing.T) {
		data := map[string]interface{}{
			"sql": "DROP DATABASE production",
		}

		result, err := pipeline.Validate(context.Background(), &ValidationRequest{
			Data:        data,
			Environment: "staging",
		})
		require.NoError(t, err)
		assert.False(t, result.Valid)
	})

	t.Run("production without canary warning", func(t *testing.T) {
		data := map[string]interface{}{
			"phases": []interface{}{"wave1", "wave2"},
		}

		result, err := pipeline.Validate(context.Background(), &ValidationRequest{
			Data:        data,
			Environment: "production",
		})
		require.NoError(t, err)
		assert.Contains(t, result.Warnings, "Production changes should include canary deployment")
	})

	t.Run("production with canary - no warning", func(t *testing.T) {
		data := map[string]interface{}{
			"phases": []interface{}{"canary", "wave1"},
		}

		result, err := pipeline.Validate(context.Background(), &ValidationRequest{
			Data:        data,
			Environment: "production",
		})
		require.NoError(t, err)
		// Should not have the canary warning
		for _, w := range result.Warnings {
			assert.NotContains(t, w, "canary deployment")
		}
	})
}

func TestPipeline_ComputeQualityScore(t *testing.T) {
	log := logger.New("debug", "text")
	cfg := config.OPAConfig{Enabled: false}
	pipeline, _ := NewPipeline(cfg, log)

	t.Run("high quality artifact", func(t *testing.T) {
		score := pipeline.ComputeQualityScore(context.Background(), &QualityScoreRequest{
			ArtifactType: "sop",
			Data: map[string]interface{}{
				"id":    "sop-1",
				"steps": []interface{}{},
				"scope": "database",
			},
			ValidationResult: &ValidationResult{Valid: true},
			TestResults: []TestResult{
				{Name: "unit-1", Passed: true, Coverage: 85},
				{Name: "unit-2", Passed: true, Coverage: 90},
			},
			HistoryStats: &HistoryStats{
				TotalExecutions: 100,
				SuccessfulRuns:  99,
				SuccessRate:     99,
				RollbackCount:   0,
			},
			HumanApprovals: []HumanApproval{
				{ApproverRole: "lead"},
			},
		})

		assert.True(t, score.Total >= 80, "expected high score, got %d", score.Total)
		assert.Contains(t, score.AllowedEnvironments, "production")
	})

	t.Run("low quality artifact", func(t *testing.T) {
		score := pipeline.ComputeQualityScore(context.Background(), &QualityScoreRequest{
			ArtifactType: "sop",
			Data:         map[string]interface{}{},
			ValidationResult: &ValidationResult{
				Valid: false,
				Errors: []ValidationError{
					{Code: "SCHEMA_INVALID"},
					{Code: "DANGEROUS_PATTERN"},
				},
			},
			TestResults: []TestResult{
				{Name: "test-1", Passed: false},
			},
			HistoryStats:   nil, // No history
			HumanApprovals: nil, // No approvals
		})

		assert.True(t, score.Total < 60, "expected low score, got %d", score.Total)
		assert.NotContains(t, score.AllowedEnvironments, "production")
	})

	t.Run("grade calculation", func(t *testing.T) {
		score := &QualityScore{Total: 95}
		assert.Equal(t, "A", score.GetGrade())

		score.Total = 85
		assert.Equal(t, "B", score.GetGrade())

		score.Total = 75
		assert.Equal(t, "C", score.GetGrade())

		score.Total = 65
		assert.Equal(t, "D", score.GetGrade())

		score.Total = 50
		assert.Equal(t, "F", score.GetGrade())
	})

	t.Run("environment threshold check", func(t *testing.T) {
		score := &QualityScore{Total: 65}

		assert.True(t, score.IsAllowedForEnvironment("development"))
		assert.True(t, score.IsAllowedForEnvironment("staging"))
		assert.False(t, score.IsAllowedForEnvironment("production"))
	})
}

func TestPipeline_Disabled(t *testing.T) {
	log := logger.New("debug", "text")
	cfg := config.OPAConfig{Enabled: false}
	pipeline, _ := NewPipeline(cfg, log)

	// When disabled, validation should pass everything
	result, err := pipeline.Validate(context.Background(), &ValidationRequest{
		Data:        map[string]interface{}{"rm -rf /": true},
		Schema:      "some_schema",
		Environment: "production",
	})
	require.NoError(t, err)
	assert.True(t, result.Valid)
}

func TestPipeline_RegisterSchemaFromJSON(t *testing.T) {
	log := logger.New("debug", "text")
	cfg := config.OPAConfig{Enabled: false}
	pipeline, _ := NewPipeline(cfg, log)

	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["id"],
		"properties": {
			"id": {"type": "string", "minLength": 1}
		}
	}`

	err := pipeline.RegisterSchemaFromJSON("custom_schema", schemaJSON)
	require.NoError(t, err)

	// Verify it works
	_, ok := pipeline.GetSchema("custom_schema")
	assert.True(t, ok)
}
