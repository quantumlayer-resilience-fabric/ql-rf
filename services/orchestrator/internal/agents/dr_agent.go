// Package agents provides the specialist agent registry and implementations.
package agents

import (
	"context"
	"fmt"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
)

// DRAgent handles disaster recovery planning and drills.
type DRAgent struct {
	BaseAgent
}

// NewDRAgent creates a new DR agent.
func NewDRAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *DRAgent {
	return &DRAgent{
		BaseAgent: BaseAgent{
			name:        "dr_agent",
			description: "Plans and executes disaster recovery drills",
			tasks:       []TaskType{TaskTypeDRDrill},
			tools: []string{
				"query_assets",
				"get_dr_status",
				"generate_dr_runbook",
				"simulate_failover",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("dr-agent"),
		},
	}
}

// Execute runs the DR agent.
func (a *DRAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing DR agent", "task_id", task.ID, "goal", task.Goal)

	// Step 1: Query assets for DR assessment
	assetFilter := task.Context.AssetFilter
	if assetFilter == "" {
		assetFilter = "state:running"
	}

	assets, err := a.executeTool(ctx, "query_assets", map[string]interface{}{
		"filter": assetFilter,
		"org_id": task.OrgID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query assets: %w", err)
	}

	assetCount := countAssets(assets)

	// Step 2: Get current DR status
	drStatus, err := a.executeTool(ctx, "get_dr_status", map[string]interface{}{
		"org_id": task.OrgID,
	})
	if err != nil {
		a.log.Warn("failed to get DR status", "error", err)
		drStatus = map[string]interface{}{"status": "unknown"}
	}

	// Step 3: Determine operation type (drill, runbook, assessment)
	operationType := a.determineDROperation(task.Goal, task.UserIntent)

	// Step 4: Generate DR runbook if needed
	var runbook interface{}
	if operationType == "drill" || operationType == "runbook" {
		runbook, err = a.executeTool(ctx, "generate_dr_runbook", map[string]interface{}{
			"org_id":      task.OrgID,
			"environment": task.Environment,
			"dr_type":     operationType,
		})
		if err != nil {
			a.log.Warn("failed to generate DR runbook", "error", err)
			runbook = map[string]interface{}{"status": "generation_failed"}
		}
	}

	// Step 5: Simulate failover if requested
	var failoverSimulation interface{}
	if operationType == "drill" {
		failoverSimulation, err = a.executeTool(ctx, "simulate_failover", map[string]interface{}{
			"org_id":      task.OrgID,
			"environment": task.Environment,
			"dry_run":     true,
		})
		if err != nil {
			a.log.Warn("failed to simulate failover", "error", err)
			failoverSimulation = map[string]interface{}{"status": "simulation_failed"}
		}
	}

	// Step 6: Generate comprehensive DR plan using LLM
	plan, tokensUsed, err := a.generateDRPlan(ctx, assets, drStatus, runbook, failoverSimulation, operationType, task)
	if err != nil {
		return nil, fmt.Errorf("failed to generate DR plan: %w", err)
	}

	// Determine readiness score
	readinessScore := 0
	if ds, ok := drStatus.(map[string]interface{}); ok {
		if score, ok := ds["readiness_score"].(float64); ok {
			readinessScore = int(score)
		}
	}

	riskLevel := "high"
	if readinessScore >= 90 {
		riskLevel = "low"
	} else if readinessScore >= 70 {
		riskLevel = "medium"
	}

	actionLabel := "Approve & Execute Drill"
	if operationType == "runbook" {
		actionLabel = "Approve Runbook"
	} else if operationType == "assessment" {
		actionLabel = "Approve Assessment"
	}

	return &AgentResult{
		TaskID:         task.ID,
		AgentName:      a.name,
		Status:         AgentStatusPendingApproval,
		Plan:           plan,
		Summary:        fmt.Sprintf("DR %s plan generated. Readiness: %d%%. Assets: %d", operationType, readinessScore, assetCount),
		AffectedAssets: assetCount,
		RiskLevel:      riskLevel,
		TokensUsed:     tokensUsed,
		Actions: []Action{
			{Type: "approve", Label: actionLabel, Description: "Approve and proceed with DR operation"},
			{Type: "modify", Label: "Modify Plan", Description: "Edit the DR plan before execution"},
			{Type: "reject", Label: "Cancel", Description: "Cancel the DR operation"},
		},
		Evidence: map[string]interface{}{
			"assets":              assets,
			"dr_status":           drStatus,
			"runbook":             runbook,
			"failover_simulation": failoverSimulation,
			"operation_type":      operationType,
			"readiness_score":     readinessScore,
		},
	}, nil
}

// determineDROperation determines what DR operation the user wants.
func (a *DRAgent) determineDROperation(goal, intent string) string {
	combined := goal + " " + intent

	if contains(combined, "drill") || contains(combined, "test") || contains(combined, "execute") {
		return "drill"
	}
	if contains(combined, "runbook") || contains(combined, "procedure") || contains(combined, "document") {
		return "runbook"
	}
	return "assessment"
}

// generateDRPlan uses LLM to create a comprehensive DR plan.
func (a *DRAgent) generateDRPlan(ctx context.Context, assets, drStatus, runbook, failoverSimulation interface{}, operationType string, task *TaskSpec) (interface{}, int, error) {
	prompt := fmt.Sprintf(`You are the QL-RF Disaster Recovery Agent. Generate a comprehensive DR plan.

## Operation Type
%s

## Environment
%s

## Assets
%v

## Current DR Status
%v

## Runbook (if generated)
%v

## Failover Simulation Results (if performed)
%v

## Your Task
Generate a detailed DR plan for a %s operation:
1. Pre-requisites and readiness checks
2. Step-by-step execution plan
3. Communication plan
4. Rollback procedures
5. Success criteria
6. Post-operation validation

Output ONLY valid JSON:
{
  "operation": "%s",
  "summary": "Brief description",
  "readiness_score": 85,
  "estimated_duration": "4h",
  "prerequisites": [
    {"check": "Check description", "status": "ready|not_ready", "remediation": "Fix if not ready"}
  ],
  "phases": [
    {
      "name": "Phase name",
      "description": "Phase description",
      "estimated_duration": "30m",
      "steps": ["Step 1", "Step 2"],
      "success_criteria": "Criteria for phase success",
      "rollback_steps": ["Rollback step 1"]
    }
  ],
  "communication": {
    "stakeholders": ["Team A", "Team B"],
    "notification_channels": ["slack", "email"],
    "escalation_path": "Description of escalation"
  },
  "rollback_plan": {
    "triggers": ["Trigger 1", "Trigger 2"],
    "procedure": "Rollback description",
    "estimated_duration": "1h"
  },
  "success_criteria": {
    "rto_target": "4h",
    "rpo_target": "1h",
    "validation_checks": ["Check 1", "Check 2"]
  }
}`, operationType, task.Environment, assets, drStatus, runbook, failoverSimulation, operationType, operationType)

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: "You are a disaster recovery specialist. Generate thorough, executable DR plans. Output ONLY valid JSON.",
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, 0, fmt.Errorf("LLM completion failed: %w", err)
	}

	var plan map[string]interface{}
	content := resp.Content

	if startIdx := findJSONStart(content); startIdx >= 0 {
		if endIdx := findJSONEnd(content, startIdx); endIdx > startIdx {
			content = content[startIdx : endIdx+1]
		}
	}

	if err := parseJSON(content, &plan); err != nil {
		a.log.Warn("failed to parse LLM response, using default plan", "error", err)
		plan = map[string]interface{}{
			"operation":       operationType,
			"summary":         "DR plan requires manual review",
			"readiness_score": 50,
			"phases": []map[string]interface{}{
				{
					"name":        "Preparation",
					"description": "Verify all prerequisites",
					"steps":       []string{"Verify backups", "Check replication status", "Notify stakeholders"},
				},
			},
		}
	}

	return plan, resp.Usage.TotalTokens, nil
}
