// Package agents provides the specialist agent registry and implementations.
package agents

import (
	"context"
	"fmt"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
)

// ComplianceAgent handles compliance audits and evidence generation.
type ComplianceAgent struct {
	BaseAgent
}

// NewComplianceAgent creates a new compliance agent.
func NewComplianceAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *ComplianceAgent {
	return &ComplianceAgent{
		BaseAgent: BaseAgent{
			name:        "compliance_agent",
			description: "Performs compliance audits and generates evidence packages",
			tasks:       []TaskType{TaskTypeComplianceAudit},
			tools: []string{
				"query_assets",
				"get_compliance_status",
				"check_control",
				"generate_compliance_evidence",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("compliance-agent"),
		},
	}
}

// Execute runs the compliance agent.
func (a *ComplianceAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing compliance agent", "task_id", task.ID, "goal", task.Goal)

	// Step 1: Query assets to audit
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
	a.log.Info("found assets for compliance audit", "count", assetCount)

	// Step 2: Get current compliance status
	complianceStatus, err := a.executeTool(ctx, "get_compliance_status", map[string]interface{}{
		"org_id": task.OrgID,
	})
	if err != nil {
		a.log.Warn("failed to get compliance status", "error", err)
		complianceStatus = map[string]interface{}{"status": "unknown"}
	}

	// Step 3: Determine which frameworks to audit (from goal or defaults)
	frameworks := a.parseFrameworks(task.Goal, task.UserIntent)

	// Step 4: Run control checks for each framework
	controlResults := make(map[string]interface{})
	totalPassed := 0
	totalFailed := 0
	totalWarnings := 0

	for _, framework := range frameworks {
		result, err := a.executeTool(ctx, "check_control", map[string]interface{}{
			"framework": framework,
			"org_id":    task.OrgID,
			"assets":    assets,
		})
		if err != nil {
			a.log.Warn("failed to check controls", "framework", framework, "error", err)
			result = map[string]interface{}{
				"framework": framework,
				"status":    "error",
				"error":     err.Error(),
			}
		}

		controlResults[framework] = result

		// Count results
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

	// Step 5: Generate compliance evidence using LLM
	report, tokensUsed, err := a.generateComplianceReport(ctx, assets, complianceStatus, controlResults, frameworks, task)
	if err != nil {
		return nil, fmt.Errorf("failed to generate compliance report: %w", err)
	}

	// Step 6: Generate evidence package
	evidencePackage, err := a.executeTool(ctx, "generate_compliance_evidence", map[string]interface{}{
		"org_id":     task.OrgID,
		"frameworks": frameworks,
		"results":    controlResults,
	})
	if err != nil {
		a.log.Warn("failed to generate evidence package", "error", err)
		evidencePackage = map[string]interface{}{"status": "generation_failed"}
	}

	// Calculate overall compliance score
	totalControls := totalPassed + totalFailed
	complianceScore := 0
	if totalControls > 0 {
		complianceScore = (totalPassed * 100) / totalControls
	}

	complianceLevel := "critical"
	if complianceScore >= 95 {
		complianceLevel = "compliant"
	} else if complianceScore >= 80 {
		complianceLevel = "minor_issues"
	} else if complianceScore >= 60 {
		complianceLevel = "needs_attention"
	}

	return &AgentResult{
		TaskID:         task.ID,
		AgentName:      a.name,
		Status:         AgentStatusPendingApproval,
		Plan:           report,
		Summary:        fmt.Sprintf("Compliance audit: %d%% compliant (%d passed, %d failed, %d warnings) across %d frameworks", complianceScore, totalPassed, totalFailed, totalWarnings, len(frameworks)),
		AffectedAssets: assetCount,
		RiskLevel:      complianceLevel,
		TokensUsed:     tokensUsed,
		Actions: []Action{
			{Type: "approve", Label: "Approve & Export", Description: "Approve the compliance report and generate evidence package"},
			{Type: "modify", Label: "Modify Report", Description: "Edit findings before export"},
			{Type: "reject", Label: "Reject", Description: "Reject and discard the audit"},
		},
		Evidence: map[string]interface{}{
			"assets":            assets,
			"compliance_status": complianceStatus,
			"control_results":   controlResults,
			"evidence_package":  evidencePackage,
			"metrics": map[string]interface{}{
				"compliance_score": complianceScore,
				"total_passed":     totalPassed,
				"total_failed":     totalFailed,
				"total_warnings":   totalWarnings,
				"frameworks":       frameworks,
			},
		},
	}, nil
}

// parseFrameworks extracts compliance frameworks from the user's request.
func (a *ComplianceAgent) parseFrameworks(goal, intent string) []string {
	combined := goal + " " + intent
	frameworks := []string{}

	frameworkKeywords := map[string]string{
		"cis":      "CIS",
		"soc2":     "SOC2",
		"soc 2":    "SOC2",
		"hipaa":    "HIPAA",
		"pci":      "PCI-DSS",
		"pci-dss":  "PCI-DSS",
		"gdpr":     "GDPR",
		"nist":     "NIST",
		"iso":      "ISO27001",
		"iso27001": "ISO27001",
		"fedramp":  "FedRAMP",
	}

	for keyword, framework := range frameworkKeywords {
		if contains(combined, keyword) {
			frameworks = append(frameworks, framework)
		}
	}

	// Default to CIS if no specific framework mentioned
	if len(frameworks) == 0 {
		frameworks = []string{"CIS"}
	}

	return frameworks
}

// generateComplianceReport uses LLM to create a detailed compliance report.
func (a *ComplianceAgent) generateComplianceReport(ctx context.Context, assets, complianceStatus, controlResults interface{}, frameworks []string, task *TaskSpec) (interface{}, int, error) {
	prompt := fmt.Sprintf(`You are the QL-RF Compliance Agent. Generate a comprehensive compliance audit report.

## Audit Scope
- Organization: %s
- Environment: %s
- Frameworks: %v

## Assets Audited
%v

## Current Compliance Status
%v

## Control Check Results
%v

## Your Task
Generate a detailed compliance report with:
1. Executive summary
2. Per-framework findings
3. Critical issues requiring immediate attention
4. Remediation recommendations
5. Evidence references

Output ONLY valid JSON:
{
  "executive_summary": "Brief overview of compliance posture",
  "audit_date": "2024-01-15",
  "frameworks_audited": ["CIS", "SOC2"],
  "overall_score": 85,
  "overall_status": "minor_issues",
  "findings": [
    {
      "framework": "CIS",
      "control_id": "CIS-1.1",
      "title": "Control title",
      "status": "passed|failed|warning",
      "severity": "critical|high|medium|low",
      "description": "Finding description",
      "remediation": "Steps to fix",
      "evidence_ref": "evidence-001"
    }
  ],
  "critical_issues": [
    {
      "title": "Issue title",
      "affected_assets": 5,
      "remediation_deadline": "7 days"
    }
  ],
  "recommendations": [
    {
      "priority": 1,
      "title": "Recommendation",
      "impact": "High",
      "effort": "Medium"
    }
  ],
  "next_audit_date": "2024-04-15"
}`, task.OrgID, task.Environment, frameworks, assets, complianceStatus, controlResults)

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: "You are a compliance and security audit specialist. Generate thorough, actionable compliance reports. Output ONLY valid JSON.",
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, 0, fmt.Errorf("LLM completion failed: %w", err)
	}

	var report map[string]interface{}
	content := resp.Content

	if startIdx := findJSONStart(content); startIdx >= 0 {
		if endIdx := findJSONEnd(content, startIdx); endIdx > startIdx {
			content = content[startIdx : endIdx+1]
		}
	}

	if err := parseJSON(content, &report); err != nil {
		a.log.Warn("failed to parse LLM response, using default report", "error", err)
		report = a.defaultComplianceReport(frameworks, controlResults)
	}

	return report, resp.Usage.TotalTokens, nil
}

// defaultComplianceReport creates a fallback report when LLM parsing fails.
func (a *ComplianceAgent) defaultComplianceReport(frameworks []string, controlResults interface{}) map[string]interface{} {
	return map[string]interface{}{
		"executive_summary":  "Compliance audit completed. Review findings for details.",
		"frameworks_audited": frameworks,
		"overall_status":     "review_required",
		"control_results":    controlResults,
		"recommendations": []map[string]interface{}{
			{
				"priority": 1,
				"title":    "Review failed controls",
				"impact":   "High",
			},
		},
	}
}
