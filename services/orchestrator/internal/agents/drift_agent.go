// Package agents provides the specialist agent registry and implementations.
package agents

import (
	"context"
	"fmt"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
)

// DriftAgent handles drift detection and remediation.
type DriftAgent struct {
	BaseAgent
}

// NewDriftAgent creates a new drift agent.
func NewDriftAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *DriftAgent {
	return &DriftAgent{
		BaseAgent: BaseAgent{
			name:        "drift_agent",
			description: "Detects configuration drift and generates remediation plans",
			tasks:       []TaskType{TaskTypeDriftRemediation},
			tools: []string{
				"query_assets",
				"get_golden_image",
				"get_drift_status",
				"compare_versions",
				"generate_patch_plan",
				"simulate_rollout",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("drift-agent"),
		},
	}
}

// Execute runs the drift agent.
func (a *DriftAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing drift agent", "task_id", task.ID, "goal", task.Goal)

	// Step 1: Query affected assets
	assets, err := a.executeTool(ctx, "query_assets", map[string]interface{}{
		"filter": task.Context.AssetFilter,
		"org_id": task.OrgID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query assets: %w", err)
	}

	// Step 2: Get drift status
	driftStatus, err := a.executeTool(ctx, "get_drift_status", map[string]interface{}{
		"org_id": task.OrgID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get drift status: %w", err)
	}

	// Step 3: Get target golden image
	goldenImage, err := a.executeTool(ctx, "get_golden_image", map[string]interface{}{
		"org_id":      task.OrgID,
		"environment": task.Environment,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get golden image: %w", err)
	}

	// Step 4: Generate remediation plan using LLM
	plan, tokensUsed, err := a.generateRemediationPlan(ctx, assets, driftStatus, goldenImage, task)
	if err != nil {
		return nil, fmt.Errorf("failed to generate remediation plan: %w", err)
	}

	// Return result for HITL approval
	return &AgentResult{
		TaskID:         task.ID,
		AgentName:      a.name,
		Status:         AgentStatusPendingApproval,
		Plan:           plan,
		Summary:        fmt.Sprintf("Generated drift remediation plan for %v assets", assets),
		AffectedAssets: countAssets(assets),
		RiskLevel:      task.RiskLevel,
		TokensUsed:     tokensUsed,
		Actions: []Action{
			{Type: "approve", Label: "Approve & Execute", Description: "Approve the plan and begin execution"},
			{Type: "modify", Label: "Modify Plan", Description: "Edit the plan before execution"},
			{Type: "reject", Label: "Reject", Description: "Reject and cancel the task"},
		},
		Evidence: map[string]interface{}{
			"assets":       assets,
			"drift_status": driftStatus,
			"golden_image": goldenImage,
		},
	}, nil
}

func (a *DriftAgent) generateRemediationPlan(ctx context.Context, assets, driftStatus, goldenImage interface{}, task *TaskSpec) (interface{}, int, error) {
	// Build prompt for LLM
	prompt := fmt.Sprintf(`You are the QL-RF Drift Remediation Agent. Generate a safe remediation plan.

## Current Drift Status
%v

## Target Golden Image
%v

## Affected Assets
%v

## Constraints
- Environment: %s
- Max batch size: %v
- Canary required: %v

Generate a phased rollout plan with:
1. Canary phase (5%% of assets)
2. Progressive waves
3. Health checks between phases
4. Rollback criteria

Output as JSON with fields: summary, phases[], estimated_duration, risk_assessment`,
		driftStatus, goldenImage, assets, task.Environment,
		task.Constraints["max_batch_size"], task.Constraints["require_canary"])

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: "You are an infrastructure remediation specialist. Generate safe, validated remediation plans.",
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, 0, err
	}

	return resp.Content, resp.Usage.TotalTokens, nil
}
