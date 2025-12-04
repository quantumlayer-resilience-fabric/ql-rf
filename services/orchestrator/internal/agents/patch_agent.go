// Package agents provides the specialist agent registry and implementations.
package agents

import (
	"context"
	"fmt"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
)

// PatchAgent handles patch rollout operations.
type PatchAgent struct {
	BaseAgent
}

// NewPatchAgent creates a new patch agent.
func NewPatchAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *PatchAgent {
	return &PatchAgent{
		BaseAgent: BaseAgent{
			name:        "patch_agent",
			description: "Orchestrates patch rollouts across infrastructure",
			tasks:       []TaskType{TaskTypePatchRollout},
			tools: []string{
				"query_assets",
				"get_golden_image",
				"compare_versions",
				"generate_patch_plan",
				"generate_rollout_plan",
				"simulate_rollout",
				"calculate_risk_score",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("patch-agent"),
		},
	}
}

// Execute runs the patch agent.
func (a *PatchAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing patch agent", "task_id", task.ID, "goal", task.Goal)

	// Step 1: Query affected assets
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
	a.log.Info("found assets for patching", "count", assetCount)

	// Step 2: Get target golden image (if specified)
	var goldenImage interface{}
	goldenImage, err = a.executeTool(ctx, "get_golden_image", map[string]interface{}{
		"org_id":      task.OrgID,
		"environment": task.Environment,
	})
	if err != nil {
		a.log.Warn("failed to get golden image, will use latest available", "error", err)
		goldenImage = map[string]interface{}{"status": "using_latest"}
	}

	// Step 3: Calculate risk score for the rollout
	riskResult, err := a.executeTool(ctx, "calculate_risk_score", map[string]interface{}{
		"asset_count": assetCount,
		"environment": task.Environment,
		"org_id":      task.OrgID,
	})
	if err != nil {
		a.log.Warn("failed to calculate risk score", "error", err)
		riskResult = map[string]interface{}{"risk_level": "medium", "risk_score": 50}
	}

	riskLevel := "medium"
	riskScore := 50
	if rm, ok := riskResult.(map[string]interface{}); ok {
		if rl, ok := rm["risk_level"].(string); ok {
			riskLevel = rl
		}
		if rs, ok := rm["risk_score"].(float64); ok {
			riskScore = int(rs)
		}
	}

	// Step 4: Generate the patch rollout plan using LLM
	plan, tokensUsed, err := a.generatePatchPlan(ctx, assets, goldenImage, riskLevel, task)
	if err != nil {
		return nil, fmt.Errorf("failed to generate patch plan: %w", err)
	}

	// Step 5: Simulate the rollout
	simulationResult, err := a.executeTool(ctx, "simulate_rollout", map[string]interface{}{
		"plan":        plan,
		"asset_count": assetCount,
		"environment": task.Environment,
	})
	if err != nil {
		a.log.Warn("failed to simulate rollout", "error", err)
		simulationResult = map[string]interface{}{"status": "simulation_skipped"}
	}

	// Build comprehensive result
	return &AgentResult{
		TaskID:         task.ID,
		AgentName:      a.name,
		Status:         AgentStatusPendingApproval,
		Plan:           plan,
		Summary:        fmt.Sprintf("Generated patch rollout plan for %d assets (risk: %s, score: %d)", assetCount, riskLevel, riskScore),
		AffectedAssets: assetCount,
		RiskLevel:      riskLevel,
		TokensUsed:     tokensUsed,
		Actions: []Action{
			{Type: "approve", Label: "Approve & Execute", Description: "Approve the plan and begin phased rollout"},
			{Type: "modify", Label: "Modify Plan", Description: "Edit the rollout plan before execution"},
			{Type: "reject", Label: "Reject", Description: "Reject and cancel the patch rollout"},
		},
		Evidence: map[string]interface{}{
			"assets":       assets,
			"golden_image": goldenImage,
			"risk_assessment": map[string]interface{}{
				"level":  riskLevel,
				"score":  riskScore,
				"result": riskResult,
			},
			"simulation": simulationResult,
		},
	}, nil
}

// generatePatchPlan uses LLM to create a detailed phased rollout plan.
func (a *PatchAgent) generatePatchPlan(ctx context.Context, assets, goldenImage interface{}, riskLevel string, task *TaskSpec) (interface{}, int, error) {
	// Determine batch sizes based on risk level
	canarySize := 5
	waveSize := 25
	switch riskLevel {
	case "critical", "high":
		canarySize = 2
		waveSize = 10
	case "low":
		canarySize = 10
		waveSize = 50
	}

	// Check for constraint overrides
	if maxBatch, ok := task.Constraints["max_batch_size"].(float64); ok {
		waveSize = int(maxBatch)
	}
	if canary, ok := task.Constraints["canary_size"].(float64); ok {
		canarySize = int(canary)
	}

	prompt := fmt.Sprintf(`You are the QL-RF Patch Agent. Generate a safe, phased patch rollout plan.

## User Goal
%s

## Environment
%s

## Risk Level
%s

## Current Assets Summary
%v

## Target Golden Image
%v

## Constraints
- Canary size: %d%% of assets
- Wave size: %d%% of assets per wave
- Environment: %s
- Require health checks: true
- Auto-rollback on failure: true

## Your Task
Generate a detailed phased rollout plan with:
1. Pre-flight checks (connectivity, disk space, backup status)
2. Canary phase (%d%% of assets)
3. Progressive waves (each %d%% of remaining assets)
4. Health checks between each phase
5. Rollback triggers and procedures
6. Post-rollout validation

Output ONLY valid JSON with this structure:
{
  "summary": "Brief plan description",
  "estimated_duration": "e.g., 2h 30m",
  "total_phases": 5,
  "phases": [
    {
      "name": "Pre-flight Checks",
      "type": "validation",
      "description": "Verify all assets are ready for patching",
      "asset_percentage": 0,
      "estimated_duration": "5m",
      "checks": ["connectivity", "disk_space", "backup_status"],
      "rollback_on_failure": false
    },
    {
      "name": "Canary Deployment",
      "type": "canary",
      "description": "Patch canary group first",
      "asset_percentage": %d,
      "estimated_duration": "15m",
      "health_check_wait": "5m",
      "success_criteria": {"error_rate": "<1%%", "response_time": "<500ms"},
      "rollback_on_failure": true
    }
  ],
  "rollback_plan": {
    "triggers": ["error_rate > 5%%", "health_check_failure", "manual"],
    "procedure": "Description of rollback steps",
    "estimated_duration": "15m"
  },
  "notifications": {
    "on_start": ["slack", "email"],
    "on_phase_complete": ["slack"],
    "on_failure": ["slack", "email", "pagerduty"],
    "on_complete": ["slack", "email"]
  }
}`, task.Goal, task.Environment, riskLevel, assets, goldenImage, canarySize, waveSize, task.Environment, canarySize, waveSize, canarySize)

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: "You are an infrastructure patch management specialist. Generate safe, validated rollout plans. Output ONLY valid JSON, no markdown or explanation.",
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, 0, fmt.Errorf("LLM completion failed: %w", err)
	}

	// Parse and validate the plan
	var plan map[string]interface{}
	content := resp.Content

	// Extract JSON from response
	if startIdx := findJSONStart(content); startIdx >= 0 {
		if endIdx := findJSONEnd(content, startIdx); endIdx > startIdx {
			content = content[startIdx : endIdx+1]
		}
	}

	if err := parseJSON(content, &plan); err != nil {
		// Fallback to a sensible default plan
		a.log.Warn("failed to parse LLM response, using default plan", "error", err)
		plan = a.defaultPatchPlan(countAssets(assets), riskLevel, canarySize, waveSize)
	}

	return plan, resp.Usage.TotalTokens, nil
}

// defaultPatchPlan creates a safe default plan when LLM parsing fails.
func (a *PatchAgent) defaultPatchPlan(assetCount int, riskLevel string, canarySize, waveSize int) map[string]interface{} {
	canaryCount := (assetCount * canarySize) / 100
	if canaryCount < 1 {
		canaryCount = 1
	}
	remainingAfterCanary := assetCount - canaryCount
	wavesNeeded := (remainingAfterCanary + waveSize - 1) / waveSize // Ceiling division

	phases := []map[string]interface{}{
		{
			"name":                "Pre-flight Checks",
			"type":                "validation",
			"description":         "Verify all assets are ready for patching",
			"asset_percentage":    0,
			"estimated_duration":  "5m",
			"checks":              []string{"connectivity", "disk_space", "backup_status"},
			"rollback_on_failure": false,
		},
		{
			"name":                "Canary Deployment",
			"type":                "canary",
			"description":         fmt.Sprintf("Patch %d canary assets first", canaryCount),
			"asset_percentage":    canarySize,
			"asset_count":         canaryCount,
			"estimated_duration":  "15m",
			"health_check_wait":   "5m",
			"success_criteria":    map[string]interface{}{"error_rate": "<1%", "health_status": "healthy"},
			"rollback_on_failure": true,
		},
	}

	// Add waves
	for i := 0; i < wavesNeeded && i < 10; i++ { // Cap at 10 waves
		waveAssets := waveSize
		if i == wavesNeeded-1 {
			waveAssets = remainingAfterCanary - (i * waveSize)
		}
		phases = append(phases, map[string]interface{}{
			"name":                fmt.Sprintf("Wave %d", i+1),
			"type":                "wave",
			"description":         fmt.Sprintf("Patch wave %d (%d assets)", i+1, waveAssets),
			"asset_percentage":    waveSize,
			"asset_count":         waveAssets,
			"estimated_duration":  "20m",
			"health_check_wait":   "5m",
			"success_criteria":    map[string]interface{}{"error_rate": "<2%", "health_status": "healthy"},
			"rollback_on_failure": true,
		})
	}

	// Add final validation phase
	phases = append(phases, map[string]interface{}{
		"name":                "Post-Rollout Validation",
		"type":                "validation",
		"description":         "Verify all assets are healthy after patching",
		"asset_percentage":    100,
		"estimated_duration":  "10m",
		"checks":              []string{"health_check", "service_status", "log_errors"},
		"rollback_on_failure": false,
	})

	return map[string]interface{}{
		"summary":            fmt.Sprintf("Phased patch rollout for %d assets with canary (%d%%) and %d waves", assetCount, canarySize, wavesNeeded),
		"estimated_duration": fmt.Sprintf("%dm", 5+15+(wavesNeeded*25)+10),
		"total_phases":       len(phases),
		"phases":             phases,
		"rollback_plan": map[string]interface{}{
			"triggers":           []string{"error_rate > 5%", "health_check_failure", "manual"},
			"procedure":          "Revert patched assets to previous image version using SSM or platform-specific rollback",
			"estimated_duration": "15m",
		},
		"notifications": map[string]interface{}{
			"on_start":          []string{"slack"},
			"on_phase_complete": []string{"slack"},
			"on_failure":        []string{"slack", "email"},
			"on_complete":       []string{"slack", "email"},
		},
	}
}
