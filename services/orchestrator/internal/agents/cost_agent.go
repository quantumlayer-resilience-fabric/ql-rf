// Package agents provides the specialist agent registry and implementations.
package agents

import (
	"context"
	"fmt"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
)

// CostAgent handles cost optimization recommendations.
type CostAgent struct {
	BaseAgent
}

// NewCostAgent creates a new cost agent.
func NewCostAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *CostAgent {
	return &CostAgent{
		BaseAgent: BaseAgent{
			name:        "cost_agent",
			description: "Analyzes infrastructure costs and recommends optimizations",
			tasks:       []TaskType{TaskTypeCostOptimization},
			tools: []string{
				"query_assets",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("cost-agent"),
		},
	}
}

// Execute runs the cost agent.
func (a *CostAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing cost agent", "task_id", task.ID, "goal", task.Goal)

	// Step 1: Query all assets for cost analysis
	assets, err := a.executeTool(ctx, "query_assets", map[string]interface{}{
		"filter": "state:running",
		"org_id": task.OrgID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query assets: %w", err)
	}

	assetCount := countAssets(assets)

	// Step 2: Analyze assets and generate cost optimization recommendations using LLM
	analysis, tokensUsed, err := a.generateCostAnalysis(ctx, assets, task)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cost analysis: %w", err)
	}

	// Extract potential savings from analysis
	potentialSavings := "$0"
	savingsPercentage := 0
	if am, ok := analysis.(map[string]interface{}); ok {
		if ps, ok := am["total_potential_savings"].(string); ok {
			potentialSavings = ps
		}
		if sp, ok := am["savings_percentage"].(float64); ok {
			savingsPercentage = int(sp)
		}
	}

	return &AgentResult{
		TaskID:         task.ID,
		AgentName:      a.name,
		Status:         AgentStatusPendingApproval,
		Plan:           analysis,
		Summary:        fmt.Sprintf("Cost optimization analysis: %d assets analyzed. Potential savings: %s (%d%%)", assetCount, potentialSavings, savingsPercentage),
		AffectedAssets: assetCount,
		RiskLevel:      "low",
		TokensUsed:     tokensUsed,
		Actions: []Action{
			{Type: "approve", Label: "Apply Recommendations", Description: "Apply the cost optimization recommendations"},
			{Type: "modify", Label: "Modify Recommendations", Description: "Edit recommendations before applying"},
			{Type: "reject", Label: "Dismiss", Description: "Dismiss recommendations without action"},
		},
		Evidence: map[string]interface{}{
			"assets":             assets,
			"potential_savings":  potentialSavings,
			"savings_percentage": savingsPercentage,
		},
	}, nil
}

// generateCostAnalysis uses LLM to analyze costs and generate recommendations.
func (a *CostAgent) generateCostAnalysis(ctx context.Context, assets interface{}, task *TaskSpec) (interface{}, int, error) {
	prompt := fmt.Sprintf(`You are the QL-RF Cost Optimization Agent. Analyze infrastructure costs and provide recommendations.

## User Request
%s

## Environment
%s

## Current Assets
%v

## Your Task
Analyze the infrastructure and provide:
1. Current cost breakdown by category
2. Idle/underutilized resources
3. Right-sizing recommendations
4. Reserved instance opportunities
5. Scheduling optimizations (dev/test environments)
6. Storage optimization opportunities

Output ONLY valid JSON:
{
  "summary": "Brief cost analysis summary",
  "current_monthly_estimate": "$10,000",
  "total_potential_savings": "$2,500",
  "savings_percentage": 25,
  "analysis_date": "2024-01-15",
  "cost_breakdown": [
    {"category": "Compute", "monthly_cost": "$5,000", "percentage": 50},
    {"category": "Storage", "monthly_cost": "$2,000", "percentage": 20}
  ],
  "recommendations": [
    {
      "id": "REC-001",
      "title": "Right-size underutilized instances",
      "category": "compute|storage|network|database",
      "priority": "high|medium|low",
      "potential_savings": "$500/month",
      "effort": "low|medium|high",
      "risk": "low|medium|high",
      "affected_assets": 5,
      "description": "Detailed description",
      "implementation_steps": ["Step 1", "Step 2"],
      "automated": true
    }
  ],
  "idle_resources": [
    {
      "resource_type": "EC2 Instance",
      "resource_id": "i-xxx",
      "idle_duration": "30 days",
      "monthly_cost": "$100",
      "recommendation": "Terminate or schedule"
    }
  ],
  "rightsizing_opportunities": [
    {
      "resource_id": "i-xxx",
      "current_type": "m5.xlarge",
      "recommended_type": "m5.large",
      "avg_cpu_utilization": 15,
      "savings": "$50/month"
    }
  ],
  "scheduling_opportunities": [
    {
      "resource_group": "dev-environment",
      "current_schedule": "24/7",
      "recommended_schedule": "weekdays 8am-6pm",
      "savings": "$200/month"
    }
  ]
}`, task.UserIntent, task.Environment, assets)

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: "You are a cloud cost optimization specialist. Provide actionable, data-driven recommendations. Output ONLY valid JSON.",
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, 0, fmt.Errorf("LLM completion failed: %w", err)
	}

	var analysis map[string]interface{}
	content := resp.Content

	if startIdx := findJSONStart(content); startIdx >= 0 {
		if endIdx := findJSONEnd(content, startIdx); endIdx > startIdx {
			content = content[startIdx : endIdx+1]
		}
	}

	if err := parseJSON(content, &analysis); err != nil {
		a.log.Warn("failed to parse LLM response, using default analysis", "error", err)
		analysis = map[string]interface{}{
			"summary":                 "Cost analysis completed. Review recommendations.",
			"total_potential_savings": "$0",
			"savings_percentage":      0,
			"recommendations": []map[string]interface{}{
				{
					"title":       "Review resource utilization",
					"priority":    "medium",
					"description": "Manually review resource utilization metrics",
				},
			},
		}
	}

	return analysis, resp.Usage.TotalTokens, nil
}
