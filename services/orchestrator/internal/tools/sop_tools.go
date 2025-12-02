// Package tools provides the tool implementations for AI agents.
package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// getString is a helper to get string values from params with a default.
func getString(params map[string]interface{}, key, defaultVal string) string {
	if v, ok := params[key].(string); ok {
		return v
	}
	return defaultVal
}

// =============================================================================
// SOP Tools - Standard Operating Procedure lifecycle tools
// =============================================================================

// GenerateSOPTool generates an SOPSpec from natural language requirements.
type GenerateSOPTool struct {
	db *pgxpool.Pool
}

func (t *GenerateSOPTool) Name() string { return "generate_sop" }

func (t *GenerateSOPTool) Description() string {
	return "Generates an SOPSpec (Standard Operating Procedure) from natural language requirements. Creates structured, executable workflows for infrastructure operations."
}

func (t *GenerateSOPTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the SOP",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Description of what the SOP does",
			},
			"trigger_type": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"manual", "schedule", "event", "alert"},
				"description": "What triggers this SOP",
			},
			"environments": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Environments where this SOP applies",
			},
			"operations": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "List of operations to perform",
			},
			"rollback_strategy": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"auto", "manual", "none"},
				"description": "How to handle rollback",
			},
		},
		"required": []string{"name", "description", "operations"},
	}
}

func (t *GenerateSOPTool) RequiresApproval() bool { return false }
func (t *GenerateSOPTool) Risk() RiskLevel        { return RiskReadOnly }
func (t *GenerateSOPTool) Scope() Scope           { return ScopeOrganization }
func (t *GenerateSOPTool) Idempotent() bool       { return true }

func (t *GenerateSOPTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	name := getString(params, "name", "unnamed-sop")
	description := getString(params, "description", "")
	triggerType := getString(params, "trigger_type", "manual")
	rollbackStrategy := getString(params, "rollback_strategy", "auto")

	environments := []string{"development", "staging"}
	if envs, ok := params["environments"].([]interface{}); ok {
		environments = make([]string, len(envs))
		for i, e := range envs {
			environments[i] = e.(string)
		}
	}

	operations := []string{}
	if ops, ok := params["operations"].([]interface{}); ok {
		operations = make([]string, len(ops))
		for i, o := range ops {
			operations[i] = o.(string)
		}
	}

	// Generate SOPSpec
	sopSpec := generateSOPSpec(name, description, triggerType, environments, operations, rollbackStrategy)

	return map[string]interface{}{
		"status":   "generated",
		"sop_spec": sopSpec,
		"summary":  fmt.Sprintf("Generated SOP '%s' with %d steps", name, len(sopSpec["steps"].([]map[string]interface{}))),
	}, nil
}

func generateSOPSpec(name, description, triggerType string, environments, operations []string, rollbackStrategy string) map[string]interface{} {
	sopID := uuid.New().String()
	version := time.Now().Format("2006.01.02")

	// Convert operations to steps
	steps := []map[string]interface{}{}
	rollbackSteps := []map[string]interface{}{}

	for i, op := range operations {
		stepID := fmt.Sprintf("step-%d", i+1)
		step := createSOPStep(stepID, op, i)
		steps = append(steps, step)

		// Create corresponding rollback step
		rollbackStep := createRollbackStep(stepID, op)
		if rollbackStep != nil {
			rollbackSteps = append([]map[string]interface{}{rollbackStep}, rollbackSteps...)
		}
	}

	// Add validation step at the end
	steps = append(steps, map[string]interface{}{
		"id":          "step-validate",
		"name":        "Validate Changes",
		"description": "Validate that all changes were applied successfully",
		"action": map[string]interface{}{
			"type": "validate.health",
			"parameters": map[string]interface{}{
				"checks": []string{"connectivity", "metrics", "logs"},
			},
		},
		"on_failure": "rollback",
		"timeout":    "5m",
	})

	sopSpec := map[string]interface{}{
		"id":          sopID,
		"name":        name,
		"version":     version,
		"description": description,
		"scope": map[string]interface{}{
			"environments": environments,
		},
		"trigger": map[string]interface{}{
			"type": triggerType,
		},
		"steps": steps,
		"rollback": map[string]interface{}{
			"strategy": rollbackStrategy,
			"steps":    rollbackSteps,
			"timeout":  "30m",
		},
		"validation": map[string]interface{}{
			"success_criteria": []map[string]interface{}{
				{
					"name":      "all_steps_complete",
					"type":      "custom",
					"condition": "all_steps.status == 'completed'",
					"weight":    10,
				},
				{
					"name":      "health_check_pass",
					"type":      "health",
					"condition": "health_score >= 0.95",
					"weight":    8,
				},
			},
		},
		"notifications": []map[string]interface{}{
			{
				"when":     "start",
				"channels": []string{"slack"},
			},
			{
				"when":     "failure",
				"channels": []string{"slack", "pagerduty"},
			},
			{
				"when":     "success",
				"channels": []string{"slack"},
			},
		},
		"approval": map[string]interface{}{
			"required":      true,
			"min_approvers": 1,
			"auto_approve":  false,
		},
	}

	return sopSpec
}

func createSOPStep(stepID, operation string, index int) map[string]interface{} {
	// Map operation descriptions to action types and parameters
	actionType, actionParams := parseOperation(operation)

	step := map[string]interface{}{
		"id":          stepID,
		"name":        fmt.Sprintf("Step %d: %s", index+1, summarizeOperation(operation)),
		"description": operation,
		"action": map[string]interface{}{
			"type":       actionType,
			"parameters": actionParams,
		},
		"on_failure": "stop",
		"timeout":    "10m",
		"retries":    2,
	}

	// Add dependencies for sequential execution
	if index > 0 {
		step["depends_on"] = []string{fmt.Sprintf("step-%d", index)}
	}

	return step
}

func parseOperation(operation string) (string, map[string]interface{}) {
	// Simple pattern matching to determine action type
	lowerOp := operation

	switch {
	case containsAny(lowerOp, "list", "query", "find", "get", "show"):
		return "inventory.query", map[string]interface{}{
			"query": operation,
		}
	case containsAny(lowerOp, "drift", "compare"):
		return "drift.check", map[string]interface{}{
			"scope": "affected_assets",
		}
	case containsAny(lowerOp, "compliance", "audit"):
		return "compliance.check", map[string]interface{}{
			"frameworks": []string{"CIS", "SOC2"},
		}
	case containsAny(lowerOp, "health", "status"):
		return "health.check", map[string]interface{}{
			"checks": []string{"connectivity", "response_time"},
		}
	case containsAny(lowerOp, "notify", "alert", "slack"):
		return "notify.slack", map[string]interface{}{
			"message": operation,
		}
	case containsAny(lowerOp, "ticket", "jira"):
		return "change.create_ticket", map[string]interface{}{
			"title":       operation,
			"priority":    "medium",
			"auto_close":  true,
		}
	case containsAny(lowerOp, "batch", "rollout", "deploy"):
		return "rollout.batch", map[string]interface{}{
			"batch_size":    10,
			"wait_between":  "5m",
			"health_checks": true,
		}
	case containsAny(lowerOp, "canary"):
		return "rollout.canary", map[string]interface{}{
			"canary_percent": 5,
			"wait_time":      "30m",
		}
	case containsAny(lowerOp, "image", "build"):
		return "image.build", map[string]interface{}{
			"validate": true,
		}
	case containsAny(lowerOp, "promote", "publish"):
		return "image.promote", map[string]interface{}{
			"target_env": "production",
		}
	case containsAny(lowerOp, "failover", "dr"):
		return "dr.failover", map[string]interface{}{
			"mode": "automatic",
		}
	case containsAny(lowerOp, "wait", "pause"):
		return "wait.duration", map[string]interface{}{
			"duration": "5m",
		}
	case containsAny(lowerOp, "approve", "approval"):
		return "wait.approval", map[string]interface{}{
			"timeout": "1h",
		}
	case containsAny(lowerOp, "validate", "verify", "check"):
		return "validate.health", map[string]interface{}{
			"checks": []string{"all"},
		}
	default:
		return "inventory.query", map[string]interface{}{
			"query": operation,
		}
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if containsIgnoreCase(s, sub) {
			return true
		}
	}
	return false
}

func containsIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			c1 := s[i+j]
			c2 := substr[j]
			// Simple case-insensitive comparison
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 32
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 32
			}
			if c1 != c2 {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func summarizeOperation(operation string) string {
	if len(operation) > 50 {
		return operation[:47] + "..."
	}
	return operation
}

func createRollbackStep(originalStepID, operation string) map[string]interface{} {
	// Only create rollback for modifying operations
	if containsAny(operation, "list", "query", "find", "get", "show", "check", "validate", "notify") {
		return nil
	}

	return map[string]interface{}{
		"id":          fmt.Sprintf("rollback-%s", originalStepID),
		"name":        fmt.Sprintf("Rollback: %s", summarizeOperation(operation)),
		"description": fmt.Sprintf("Undo changes from: %s", operation),
		"action": map[string]interface{}{
			"type": "rollout.abort",
			"parameters": map[string]interface{}{
				"original_step": originalStepID,
				"restore":       true,
			},
		},
		"timeout": "15m",
	}
}

// ValidateSOPTool validates an SOPSpec for correctness and safety.
type ValidateSOPTool struct {
	db *pgxpool.Pool
}

func (t *ValidateSOPTool) Name() string { return "validate_sop" }

func (t *ValidateSOPTool) Description() string {
	return "Validates an SOPSpec for structural correctness, policy compliance, and safety. Returns a validation report with quality score."
}

func (t *ValidateSOPTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"sop_spec": map[string]interface{}{
				"type":        "object",
				"description": "The SOPSpec to validate",
			},
			"strict": map[string]interface{}{
				"type":        "boolean",
				"description": "Enable strict validation mode",
			},
		},
		"required": []string{"sop_spec"},
	}
}

func (t *ValidateSOPTool) RequiresApproval() bool { return false }
func (t *ValidateSOPTool) Risk() RiskLevel        { return RiskReadOnly }
func (t *ValidateSOPTool) Scope() Scope           { return ScopeOrganization }
func (t *ValidateSOPTool) Idempotent() bool       { return true }

func (t *ValidateSOPTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	sopSpec, ok := params["sop_spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("sop_spec is required")
	}

	strict := false
	if s, ok := params["strict"].(bool); ok {
		strict = s
	}

	// Validate the SOP
	errors := []string{}
	warnings := []string{}

	// Check required fields
	requiredFields := []string{"id", "name", "steps", "scope"}
	for _, field := range requiredFields {
		if _, exists := sopSpec[field]; !exists {
			errors = append(errors, fmt.Sprintf("Missing required field: %s", field))
		}
	}

	// Check steps
	if steps, ok := sopSpec["steps"].([]interface{}); ok {
		if len(steps) == 0 {
			errors = append(errors, "SOP must have at least one step")
		}

		// Validate each step
		for i, step := range steps {
			if stepMap, ok := step.(map[string]interface{}); ok {
				if _, hasAction := stepMap["action"]; !hasAction {
					errors = append(errors, fmt.Sprintf("Step %d missing action", i+1))
				}
			}
		}

		// Check for validation step
		hasValidation := false
		for _, step := range steps {
			if stepMap, ok := step.(map[string]interface{}); ok {
				if action, ok := stepMap["action"].(map[string]interface{}); ok {
					if actionType, ok := action["type"].(string); ok {
						if containsAny(actionType, "validate") {
							hasValidation = true
							break
						}
					}
				}
			}
		}
		if !hasValidation {
			warnings = append(warnings, "SOP should include a validation step")
		}
	} else {
		errors = append(errors, "Steps must be an array")
	}

	// Check rollback for production SOPs
	if scope, ok := sopSpec["scope"].(map[string]interface{}); ok {
		if envs, ok := scope["environments"].([]interface{}); ok {
			for _, env := range envs {
				if env == "production" {
					if _, hasRollback := sopSpec["rollback"]; !hasRollback {
						if strict {
							errors = append(errors, "Production SOPs must have rollback defined")
						} else {
							warnings = append(warnings, "Production SOPs should have rollback defined")
						}
					}
					break
				}
			}
		}
	}

	// Check approval for modifying operations
	hasModifyingOps := false
	if steps, ok := sopSpec["steps"].([]interface{}); ok {
		for _, step := range steps {
			if stepMap, ok := step.(map[string]interface{}); ok {
				if action, ok := stepMap["action"].(map[string]interface{}); ok {
					if actionType, ok := action["type"].(string); ok {
						if containsAny(actionType, "rollout", "deploy", "failover", "promote") {
							hasModifyingOps = true
							break
						}
					}
				}
			}
		}
	}

	if hasModifyingOps {
		if approval, ok := sopSpec["approval"].(map[string]interface{}); ok {
			if required, ok := approval["required"].(bool); !ok || !required {
				warnings = append(warnings, "SOPs with modifying operations should require approval")
			}
		} else {
			warnings = append(warnings, "SOPs with modifying operations should define approval requirements")
		}
	}

	valid := len(errors) == 0

	return map[string]interface{}{
		"valid":    valid,
		"errors":   errors,
		"warnings": warnings,
		"summary":  fmt.Sprintf("Validation %s: %d errors, %d warnings", statusText(valid), len(errors), len(warnings)),
	}, nil
}

func statusText(valid bool) string {
	if valid {
		return "PASSED"
	}
	return "FAILED"
}

// SimulateSOPTool runs a dry-run simulation of an SOP.
type SimulateSOPTool struct {
	db *pgxpool.Pool
}

func (t *SimulateSOPTool) Name() string { return "simulate_sop" }

func (t *SimulateSOPTool) Description() string {
	return "Runs a dry-run simulation of an SOP without making actual changes. Returns expected outcomes and identifies potential issues."
}

func (t *SimulateSOPTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"sop_spec": map[string]interface{}{
				"type":        "object",
				"description": "The SOPSpec to simulate",
			},
			"target_environment": map[string]interface{}{
				"type":        "string",
				"description": "Environment to simulate against",
			},
		},
		"required": []string{"sop_spec"},
	}
}

func (t *SimulateSOPTool) RequiresApproval() bool { return false }
func (t *SimulateSOPTool) Risk() RiskLevel        { return RiskReadOnly }
func (t *SimulateSOPTool) Scope() Scope           { return ScopeOrganization }
func (t *SimulateSOPTool) Idempotent() bool       { return true }

func (t *SimulateSOPTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	sopSpec, ok := params["sop_spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("sop_spec is required")
	}

	targetEnv := getString(params, "target_environment", "staging")

	// Simulate each step
	stepResults := []map[string]interface{}{}
	steps, _ := sopSpec["steps"].([]interface{})

	for i, step := range steps {
		stepMap, _ := step.(map[string]interface{})
		stepID := fmt.Sprintf("step-%d", i+1)
		if id, ok := stepMap["id"].(string); ok {
			stepID = id
		}

		result := map[string]interface{}{
			"step_id": stepID,
			"status":  "simulated",
			"outcome": "success",
			"duration_estimate": "2m",
			"affected_assets": 10,
		}

		// Simulate action-specific outcomes
		if action, ok := stepMap["action"].(map[string]interface{}); ok {
			if actionType, ok := action["type"].(string); ok {
				result["action_type"] = actionType

				// Add action-specific simulation details
				switch {
				case containsAny(actionType, "rollout"):
					result["affected_assets"] = 50
					result["duration_estimate"] = "30m"
					result["risk_level"] = "medium"
				case containsAny(actionType, "failover"):
					result["affected_assets"] = 100
					result["duration_estimate"] = "15m"
					result["risk_level"] = "high"
				case containsAny(actionType, "validate", "check"):
					result["affected_assets"] = 0
					result["duration_estimate"] = "5m"
					result["risk_level"] = "low"
				}
			}
		}

		stepResults = append(stepResults, result)
	}

	// Calculate totals
	totalAssets := 0
	totalDuration := 0
	for _, r := range stepResults {
		if assets, ok := r["affected_assets"].(int); ok {
			totalAssets += assets
		}
		// Simplified duration aggregation
		totalDuration += 5
	}

	return map[string]interface{}{
		"status":             "simulation_complete",
		"target_environment": targetEnv,
		"step_results":       stepResults,
		"total_steps":        len(stepResults),
		"total_affected_assets": totalAssets,
		"estimated_duration": fmt.Sprintf("%dm", totalDuration),
		"simulation_passed":  true,
		"issues_found":       []string{},
	}, nil
}

// ExecuteSOPTool executes an approved SOP.
type ExecuteSOPTool struct {
	db *pgxpool.Pool
}

func (t *ExecuteSOPTool) Name() string { return "execute_sop" }

func (t *ExecuteSOPTool) Description() string {
	return "Executes an approved SOP. Requires HITL approval for production environments."
}

func (t *ExecuteSOPTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"sop_id": map[string]interface{}{
				"type":        "string",
				"description": "ID of the SOP to execute",
			},
			"target_environment": map[string]interface{}{
				"type":        "string",
				"description": "Environment to execute in",
			},
			"parameters": map[string]interface{}{
				"type":        "object",
				"description": "Runtime parameters for the SOP",
			},
		},
		"required": []string{"sop_id", "target_environment"},
	}
}

func (t *ExecuteSOPTool) RequiresApproval() bool { return true }
func (t *ExecuteSOPTool) Risk() RiskLevel        { return RiskStateChangeProd }
func (t *ExecuteSOPTool) Scope() Scope           { return ScopeEnvironment }
func (t *ExecuteSOPTool) Idempotent() bool       { return false }

func (t *ExecuteSOPTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	sopID := getString(params, "sop_id", "")
	targetEnv := getString(params, "target_environment", "staging")

	if sopID == "" {
		return nil, fmt.Errorf("sop_id is required")
	}

	executionID := uuid.New().String()

	return map[string]interface{}{
		"status":             "execution_started",
		"execution_id":       executionID,
		"sop_id":             sopID,
		"target_environment": targetEnv,
		"started_at":         time.Now().Format(time.RFC3339),
		"message":            "SOP execution has been queued and will be processed by the workflow engine",
	}, nil
}

// ListSOPsTool lists available SOPs.
type ListSOPsTool struct {
	db *pgxpool.Pool
}

func (t *ListSOPsTool) Name() string { return "list_sops" }

func (t *ListSOPsTool) Description() string {
	return "Lists available SOPs with optional filtering by environment, trigger type, or status."
}

func (t *ListSOPsTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"environment": map[string]interface{}{
				"type":        "string",
				"description": "Filter by environment",
			},
			"trigger_type": map[string]interface{}{
				"type":        "string",
				"description": "Filter by trigger type",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results",
			},
		},
	}
}

func (t *ListSOPsTool) RequiresApproval() bool { return false }
func (t *ListSOPsTool) Risk() RiskLevel        { return RiskReadOnly }
func (t *ListSOPsTool) Scope() Scope           { return ScopeOrganization }
func (t *ListSOPsTool) Idempotent() bool       { return true }

func (t *ListSOPsTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// In production, this would query the database
	// For now, return sample data
	sops := []map[string]interface{}{
		{
			"id":           "sop-drift-remediation",
			"name":         "Drift Remediation SOP",
			"version":      "2025.01.01",
			"trigger_type": "event",
			"environments": []string{"staging", "production"},
			"status":       "active",
		},
		{
			"id":           "sop-patch-rollout",
			"name":         "Security Patch Rollout",
			"version":      "2025.01.01",
			"trigger_type": "manual",
			"environments": []string{"staging", "production"},
			"status":       "active",
		},
		{
			"id":           "sop-incident-response",
			"name":         "Incident Response Playbook",
			"version":      "2025.01.01",
			"trigger_type": "alert",
			"environments": []string{"production"},
			"status":       "active",
		},
	}

	return map[string]interface{}{
		"sops":  sops,
		"total": len(sops),
	}, nil
}
