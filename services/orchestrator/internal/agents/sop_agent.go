// Package agents provides the specialist agent registry and implementations.
package agents

import (
	"context"
	"fmt"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
)

// SOPAgent handles SOP authoring from natural language.
type SOPAgent struct {
	BaseAgent
}

// NewSOPAgent creates a new SOP agent.
func NewSOPAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *SOPAgent {
	return &SOPAgent{
		BaseAgent: BaseAgent{
			name:        "sop_agent",
			description: "Authors Standard Operating Procedures from natural language requirements",
			tasks:       []TaskType{TaskTypeSOPAuthoring},
			tools: []string{
				"generate_sop",
				"validate_sop",
				"simulate_sop",
				"execute_sop",
				"list_sops",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("sop-agent"),
		},
	}
}

// Execute runs the SOP agent.
func (a *SOPAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing SOP agent", "task_id", task.ID, "goal", task.Goal)

	// Step 1: Parse the user intent to extract SOP requirements
	requirements, tokensUsed, err := a.parseSOPRequirements(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SOP requirements: %w", err)
	}

	// Step 2: Generate the SOP using the tool
	sopResult, err := a.executeTool(ctx, "generate_sop", requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SOP: %w", err)
	}

	sopResultMap := sopResult.(map[string]interface{})
	sopSpec := sopResultMap["sop_spec"].(map[string]interface{})

	// Step 3: Validate the SOP
	validationResult, err := a.executeTool(ctx, "validate_sop", map[string]interface{}{
		"sop_spec": sopSpec,
		"strict":   task.Environment == "production",
	})
	if err != nil {
		a.log.Warn("SOP validation failed", "error", err)
	}

	// Step 4: Simulate the SOP
	simulationResult, err := a.executeTool(ctx, "simulate_sop", map[string]interface{}{
		"sop_spec":           sopSpec,
		"target_environment": task.Environment,
	})
	if err != nil {
		a.log.Warn("SOP simulation failed", "error", err)
	}

	// Build the plan
	steps := []map[string]interface{}{}
	if s, ok := sopSpec["steps"].([]map[string]interface{}); ok {
		steps = s
	} else if s, ok := sopSpec["steps"].([]interface{}); ok {
		for _, step := range s {
			if sm, ok := step.(map[string]interface{}); ok {
				steps = append(steps, sm)
			}
		}
	}

	plan := map[string]interface{}{
		"summary":         fmt.Sprintf("Generated SOP: %s", sopSpec["name"]),
		"sop_spec":        sopSpec,
		"validation":      validationResult,
		"simulation":      simulationResult,
		"total_steps":     len(steps),
		"affected_assets": 0,
		"phases": []map[string]interface{}{
			{
				"name":        "SOP Validation",
				"description": "Validate SOP structure and policies",
			},
			{
				"name":        "Dry-Run Simulation",
				"description": "Simulate SOP execution without changes",
			},
			{
				"name":        "SOP Registration",
				"description": "Register SOP in the system",
			},
		},
	}

	return &AgentResult{
		TaskID:         task.ID,
		AgentName:      a.name,
		Status:         AgentStatusPendingApproval,
		Plan:           plan,
		Summary:        fmt.Sprintf("Generated SOP '%s' with %d steps", sopSpec["name"], len(steps)),
		AffectedAssets: 0,
		RiskLevel:      task.RiskLevel,
		TokensUsed:     tokensUsed,
		Actions: []Action{
			{Type: "approve", Label: "Approve & Register", Description: "Approve the SOP and register it"},
			{Type: "modify", Label: "Modify SOP", Description: "Edit the SOP before registering"},
			{Type: "reject", Label: "Reject", Description: "Reject and discard the SOP"},
		},
		Evidence: map[string]interface{}{
			"sop_spec":   sopSpec,
			"validation": validationResult,
			"simulation": simulationResult,
		},
	}, nil
}

func (a *SOPAgent) parseSOPRequirements(ctx context.Context, task *TaskSpec) (map[string]interface{}, int, error) {
	prompt := fmt.Sprintf(`You are the QL-RF SOP Agent. Parse the user's request into SOP requirements.

## User Request
%s

## Available Action Types
- inventory.list, inventory.query - Query assets
- drift.check, compliance.check, health.check - Validation checks
- notify.slack, notify.email, change.create_ticket - Notifications
- rollout.batch, rollout.canary - Deployments
- validate.health, validate.metrics - Post-change validation
- wait.duration, wait.approval - Wait operations

## Your Task
Extract:
1. SOP name and description
2. Trigger type (manual, schedule, event, alert)
3. Target environments
4. List of operations to perform (in natural language)
5. Rollback strategy (auto, manual, none)

Output ONLY valid JSON:
{
  "name": "string",
  "description": "string",
  "trigger_type": "manual|schedule|event|alert",
  "environments": ["staging", "production"],
  "operations": ["step 1 description", "step 2 description", ...],
  "rollback_strategy": "auto|manual|none"
}`, task.UserIntent)

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: "You are an SOP authoring expert. Parse requirements into structured JSON. Output ONLY valid JSON.",
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, 0, err
	}

	// Parse the JSON response
	var requirements map[string]interface{}
	content := resp.Content

	if startIdx := findJSONStart(content); startIdx >= 0 {
		if endIdx := findJSONEnd(content, startIdx); endIdx > startIdx {
			content = content[startIdx : endIdx+1]
		}
	}

	if err := parseJSON(content, &requirements); err != nil {
		// Fallback to defaults
		requirements = map[string]interface{}{
			"name":              "generated-sop",
			"description":       task.UserIntent,
			"trigger_type":      "manual",
			"environments":      []interface{}{"staging"},
			"operations":        []interface{}{task.UserIntent},
			"rollback_strategy": "auto",
		}
	}

	return requirements, resp.Usage.TotalTokens, nil
}
