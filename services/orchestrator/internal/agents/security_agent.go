// Package agents provides the specialist agent registry and implementations.
package agents

import (
	"context"
	"fmt"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
)

// SecurityAgent handles security scanning and vulnerability assessment.
type SecurityAgent struct {
	BaseAgent
}

// NewSecurityAgent creates a new security agent.
func NewSecurityAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *SecurityAgent {
	return &SecurityAgent{
		BaseAgent: BaseAgent{
			name:        "security_agent",
			description: "Performs security scans and vulnerability assessments",
			tasks:       []TaskType{TaskTypeSecurityScan},
			tools: []string{
				"query_assets",
				"get_compliance_status",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("security-agent"),
		},
	}
}

// Execute runs the security agent.
func (a *SecurityAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing security agent", "task_id", task.ID, "goal", task.Goal)

	// Step 1: Query assets for security scan
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

	// Step 2: Get compliance status (includes security controls)
	complianceStatus, err := a.executeTool(ctx, "get_compliance_status", map[string]interface{}{
		"org_id": task.OrgID,
	})
	if err != nil {
		a.log.Warn("failed to get compliance status", "error", err)
		complianceStatus = map[string]interface{}{}
	}

	// Step 3: Generate security assessment using LLM
	assessment, tokensUsed, err := a.generateSecurityAssessment(ctx, assets, complianceStatus, task)
	if err != nil {
		return nil, fmt.Errorf("failed to generate security assessment: %w", err)
	}

	// Extract severity from assessment
	criticalCount := 0
	highCount := 0
	overallRisk := "medium"
	if am, ok := assessment.(map[string]interface{}); ok {
		if c, ok := am["critical_count"].(float64); ok {
			criticalCount = int(c)
		}
		if h, ok := am["high_count"].(float64); ok {
			highCount = int(h)
		}
		if criticalCount > 0 {
			overallRisk = "critical"
		} else if highCount > 0 {
			overallRisk = "high"
		}
	}

	return &AgentResult{
		TaskID:         task.ID,
		AgentName:      a.name,
		Status:         AgentStatusPendingApproval,
		Plan:           assessment,
		Summary:        fmt.Sprintf("Security scan: %d assets scanned. Critical: %d, High: %d. Overall risk: %s", assetCount, criticalCount, highCount, overallRisk),
		AffectedAssets: assetCount,
		RiskLevel:      overallRisk,
		TokensUsed:     tokensUsed,
		Actions: []Action{
			{Type: "approve", Label: "Approve & Remediate", Description: "Approve findings and generate remediation plan"},
			{Type: "modify", Label: "Modify Assessment", Description: "Edit security assessment findings"},
			{Type: "reject", Label: "Dismiss", Description: "Dismiss without action"},
		},
		Evidence: map[string]interface{}{
			"assets":            assets,
			"compliance_status": complianceStatus,
			"vulnerability_summary": map[string]interface{}{
				"critical": criticalCount,
				"high":     highCount,
			},
		},
	}, nil
}

// generateSecurityAssessment uses LLM to create a security assessment.
func (a *SecurityAgent) generateSecurityAssessment(ctx context.Context, assets, complianceStatus interface{}, task *TaskSpec) (interface{}, int, error) {
	prompt := fmt.Sprintf(`You are the QL-RF Security Agent. Perform a comprehensive security assessment.

## User Request
%s

## Environment
%s

## Assets to Assess
%v

## Current Compliance Status
%v

## Your Task
Perform a security assessment including:
1. Vulnerability analysis
2. Configuration weaknesses
3. Access control review
4. Network security posture
5. Data protection status
6. Prioritized remediation recommendations

Output ONLY valid JSON:
{
  "summary": "Brief security assessment summary",
  "assessment_date": "2024-01-15",
  "overall_risk_score": 75,
  "overall_risk_level": "high",
  "critical_count": 2,
  "high_count": 5,
  "medium_count": 10,
  "low_count": 20,
  "vulnerability_findings": [
    {
      "id": "VULN-001",
      "title": "Vulnerability title",
      "severity": "critical|high|medium|low",
      "cvss_score": 9.8,
      "cve_id": "CVE-2024-XXXX",
      "affected_assets": 5,
      "description": "Vulnerability description",
      "remediation": "Steps to fix",
      "remediation_effort": "low|medium|high"
    }
  ],
  "configuration_findings": [
    {
      "id": "CONFIG-001",
      "title": "Configuration issue",
      "severity": "high",
      "category": "access_control|network|encryption|logging",
      "affected_assets": 3,
      "description": "Issue description",
      "remediation": "Steps to fix"
    }
  ],
  "access_control_findings": [
    {
      "finding": "Finding description",
      "risk_level": "high",
      "recommendation": "Recommendation"
    }
  ],
  "network_security_findings": [
    {
      "finding": "Finding description",
      "risk_level": "medium",
      "recommendation": "Recommendation"
    }
  ],
  "remediation_plan": {
    "immediate_actions": [
      {
        "title": "Action title",
        "severity": "critical",
        "deadline": "24 hours",
        "steps": ["Step 1", "Step 2"]
      }
    ],
    "short_term_actions": [],
    "long_term_improvements": []
  },
  "compliance_gaps": [
    {
      "framework": "CIS",
      "control_id": "CIS-1.1",
      "gap": "Gap description",
      "remediation": "Fix steps"
    }
  ]
}`, task.UserIntent, task.Environment, assets, complianceStatus)

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: "You are a cybersecurity specialist. Perform thorough security assessments and provide actionable recommendations. Output ONLY valid JSON.",
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, 0, fmt.Errorf("LLM completion failed: %w", err)
	}

	var assessment map[string]interface{}
	content := resp.Content

	if startIdx := findJSONStart(content); startIdx >= 0 {
		if endIdx := findJSONEnd(content, startIdx); endIdx > startIdx {
			content = content[startIdx : endIdx+1]
		}
	}

	if err := parseJSON(content, &assessment); err != nil {
		a.log.Warn("failed to parse LLM response, using default assessment", "error", err)
		assessment = map[string]interface{}{
			"summary":            "Security assessment completed. Manual review required.",
			"overall_risk_level": "medium",
			"critical_count":     0,
			"high_count":         0,
			"remediation_plan": map[string]interface{}{
				"immediate_actions": []string{"Review security configurations"},
			},
		}
	}

	return assessment, resp.Usage.TotalTokens, nil
}
