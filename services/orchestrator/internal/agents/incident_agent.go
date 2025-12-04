// Package agents provides the specialist agent registry and implementations.
package agents

import (
	"context"
	"fmt"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
)

// IncidentAgent handles incident investigation and response.
type IncidentAgent struct {
	BaseAgent
}

// NewIncidentAgent creates a new incident agent.
func NewIncidentAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *IncidentAgent {
	return &IncidentAgent{
		BaseAgent: BaseAgent{
			name:        "incident_agent",
			description: "Investigates incidents and generates root cause analysis",
			tasks:       []TaskType{TaskTypeIncidentResponse},
			tools: []string{
				"query_assets",
				"query_alerts",
				"get_drift_status",
				"get_compliance_status",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("incident-agent"),
		},
	}
}

// Execute runs the incident agent.
func (a *IncidentAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing incident agent", "task_id", task.ID, "goal", task.Goal)

	// Step 1: Query related assets
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

	// Step 2: Query active alerts
	alerts, err := a.executeTool(ctx, "query_alerts", map[string]interface{}{
		"org_id": task.OrgID,
		"status": "active",
	})
	if err != nil {
		a.log.Warn("failed to query alerts", "error", err)
		alerts = []interface{}{}
	}

	// Step 3: Get drift status (potential cause)
	driftStatus, err := a.executeTool(ctx, "get_drift_status", map[string]interface{}{
		"org_id": task.OrgID,
	})
	if err != nil {
		a.log.Warn("failed to get drift status", "error", err)
		driftStatus = map[string]interface{}{}
	}

	// Step 4: Get compliance status (potential cause)
	complianceStatus, err := a.executeTool(ctx, "get_compliance_status", map[string]interface{}{
		"org_id": task.OrgID,
	})
	if err != nil {
		a.log.Warn("failed to get compliance status", "error", err)
		complianceStatus = map[string]interface{}{}
	}

	// Step 5: Generate incident analysis using LLM
	analysis, tokensUsed, err := a.generateIncidentAnalysis(ctx, assets, alerts, driftStatus, complianceStatus, task)
	if err != nil {
		return nil, fmt.Errorf("failed to generate incident analysis: %w", err)
	}

	// Determine severity from analysis
	severity := "medium"
	if am, ok := analysis.(map[string]interface{}); ok {
		if s, ok := am["severity"].(string); ok {
			severity = s
		}
	}

	return &AgentResult{
		TaskID:         task.ID,
		AgentName:      a.name,
		Status:         AgentStatusPendingApproval,
		Plan:           analysis,
		Summary:        fmt.Sprintf("Incident analysis completed for %d assets. Severity: %s", assetCount, severity),
		AffectedAssets: assetCount,
		RiskLevel:      severity,
		TokensUsed:     tokensUsed,
		Actions: []Action{
			{Type: "approve", Label: "Approve & Remediate", Description: "Approve analysis and execute remediation steps"},
			{Type: "modify", Label: "Modify Analysis", Description: "Edit the incident analysis"},
			{Type: "reject", Label: "Close Incident", Description: "Close without remediation"},
		},
		Evidence: map[string]interface{}{
			"assets":            assets,
			"alerts":            alerts,
			"drift_status":      driftStatus,
			"compliance_status": complianceStatus,
		},
	}, nil
}

// generateIncidentAnalysis uses LLM to perform root cause analysis.
func (a *IncidentAgent) generateIncidentAnalysis(ctx context.Context, assets, alerts, driftStatus, complianceStatus interface{}, task *TaskSpec) (interface{}, int, error) {
	prompt := fmt.Sprintf(`You are the QL-RF Incident Agent. Perform root cause analysis for the reported incident.

## Incident Description
%s

## Environment
%s

## Affected Assets
%v

## Active Alerts
%v

## Drift Status
%v

## Compliance Status
%v

## Your Task
Perform a thorough incident investigation and generate:
1. Root cause analysis
2. Impact assessment
3. Timeline of events (if determinable)
4. Remediation steps
5. Prevention recommendations

Output ONLY valid JSON:
{
  "incident_id": "INC-001",
  "title": "Incident title",
  "severity": "critical|high|medium|low",
  "status": "investigating|identified|mitigating|resolved",
  "root_cause": {
    "category": "drift|compliance|security|performance|availability",
    "description": "Root cause description",
    "confidence": "high|medium|low"
  },
  "impact": {
    "affected_systems": 5,
    "business_impact": "Description of business impact",
    "users_affected": "estimated number or 'unknown'"
  },
  "timeline": [
    {"time": "2024-01-15T10:00:00Z", "event": "Event description"}
  ],
  "remediation": {
    "immediate_actions": ["Action 1", "Action 2"],
    "long_term_fixes": ["Fix 1", "Fix 2"],
    "estimated_time": "2h"
  },
  "prevention": [
    {"recommendation": "Recommendation text", "priority": "high"}
  ],
  "related_incidents": []
}`, task.UserIntent, task.Environment, assets, alerts, driftStatus, complianceStatus)

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: "You are an incident response specialist. Perform thorough root cause analysis. Output ONLY valid JSON.",
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
			"title":    "Incident Investigation",
			"severity": "medium",
			"status":   "investigating",
			"root_cause": map[string]interface{}{
				"category":    "unknown",
				"description": "Further investigation required",
				"confidence":  "low",
			},
			"remediation": map[string]interface{}{
				"immediate_actions": []string{"Review affected assets", "Check recent changes"},
			},
		}
	}

	return analysis, resp.Usage.TotalTokens, nil
}
