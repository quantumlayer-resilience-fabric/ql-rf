// Package agents provides the specialist agent registry and implementations.
package agents

import (
	"context"
	"fmt"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
)

// CertificateAgent handles certificate lifecycle management including expiry detection,
// impact analysis, and rotation automation.
type CertificateAgent struct {
	BaseAgent
}

// NewCertificateAgent creates a new certificate agent.
func NewCertificateAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *CertificateAgent {
	return &CertificateAgent{
		BaseAgent: BaseAgent{
			name:        "certificate_agent",
			description: "Manages certificate lifecycle including expiry detection, impact analysis, and rotation automation",
			tasks:       []TaskType{TaskTypeCertificateRotation},
			tools: []string{
				"list_certificates",
				"get_certificate_details",
				"map_certificate_usage",
				"generate_cert_renewal_plan",
				"propose_cert_rotation",
				"validate_tls_handshake",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("certificate-agent"),
		},
	}
}

// Execute runs the certificate agent for a given task.
func (a *CertificateAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing certificate agent", "task_id", task.ID, "goal", task.Goal)

	// Determine the action based on user intent
	intent := classifyCertificateIntent(task.UserIntent, task.Goal)

	switch intent {
	case "scan_expiring":
		return a.scanExpiringCertificates(ctx, task)
	case "analyze_certificate":
		return a.analyzeCertificate(ctx, task)
	case "rotate_certificate":
		return a.rotateCertificate(ctx, task)
	case "validate_endpoints":
		return a.validateEndpoints(ctx, task)
	default:
		// Default: scan for expiring certificates
		return a.scanExpiringCertificates(ctx, task)
	}
}

// scanExpiringCertificates finds certificates that need attention.
func (a *CertificateAgent) scanExpiringCertificates(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("scanning for expiring certificates", "task_id", task.ID)

	// Step 1: Get expiring certificates (within 30 days by default)
	daysThreshold := 30.0
	if days, ok := task.Constraints["expiry_threshold_days"].(float64); ok {
		daysThreshold = days
	}

	certificates, err := a.executeTool(ctx, "list_certificates", map[string]interface{}{
		"expiring_within_days": daysThreshold,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list certificates: %w", err)
	}

	certData, ok := certificates.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected certificate data format")
	}

	certList, _ := certData["certificates"].([]map[string]interface{})
	total := len(certList)

	// Categorize certificates by urgency
	critical := []map[string]interface{}{}  // expired or < 7 days
	high := []map[string]interface{}{}      // 7-14 days
	medium := []map[string]interface{}{}    // 14-30 days

	for _, cert := range certList {
		daysUntil, _ := cert["days_until_expiry"].(int)
		if daysUntil <= 0 {
			critical = append(critical, cert)
		} else if daysUntil <= 7 {
			critical = append(critical, cert)
		} else if daysUntil <= 14 {
			high = append(high, cert)
		} else {
			medium = append(medium, cert)
		}
	}

	// Generate summary using LLM
	summary, tokensUsed, err := a.generateScanSummary(ctx, critical, high, medium, task)
	if err != nil {
		a.log.Warn("failed to generate LLM summary, using fallback", "error", err)
		summary = fmt.Sprintf("Found %d certificates requiring attention: %d critical, %d high priority, %d medium priority",
			total, len(critical), len(high), len(medium))
		tokensUsed = 0
	}

	// Determine risk level
	riskLevel := "low"
	if len(critical) > 0 {
		riskLevel = "critical"
	} else if len(high) > 0 {
		riskLevel = "high"
	} else if len(medium) > 0 {
		riskLevel = "medium"
	}

	// Build recommended actions
	actions := []Action{}
	if len(critical) > 0 {
		actions = append(actions, Action{
			Type:        "rotate_critical",
			Label:       "Rotate Critical Certs",
			Description: fmt.Sprintf("Immediately rotate %d expired/expiring certificates", len(critical)),
		})
	}
	if total > 0 {
		actions = append(actions, Action{
			Type:        "generate_plan",
			Label:       "Generate Rotation Plan",
			Description: "Create a phased rotation plan for all expiring certificates",
		})
	}
	actions = append(actions, Action{
		Type:        "dismiss",
		Label:       "Acknowledge",
		Description: "Acknowledge the scan results without taking action",
	})

	return &AgentResult{
		TaskID:         task.ID,
		AgentName:      a.name,
		Status:         AgentStatusPendingApproval,
		Plan:           nil,
		Summary:        summary.(string),
		AffectedAssets: total,
		RiskLevel:      riskLevel,
		TokensUsed:     tokensUsed,
		Actions:        actions,
		Evidence: map[string]interface{}{
			"scan_results": map[string]interface{}{
				"total":        total,
				"critical":     critical,
				"high":         high,
				"medium":       medium,
				"threshold_days": daysThreshold,
			},
			"all_certificates": certList,
		},
	}, nil
}

// analyzeCertificate performs deep analysis on a specific certificate.
func (a *CertificateAgent) analyzeCertificate(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("analyzing certificate", "task_id", task.ID)

	// Get certificate ID from context
	certID, ok := task.Context.Metadata["certificate_id"].(string)
	if !ok {
		// Try to find by common name
		certName, _ := task.Context.Metadata["common_name"].(string)
		if certName == "" {
			return nil, fmt.Errorf("certificate_id or common_name required in task context")
		}
		task.Context.Metadata["common_name"] = certName
	}

	// Step 1: Get certificate details
	certDetails, err := a.executeTool(ctx, "get_certificate_details", map[string]interface{}{
		"certificate_id": certID,
		"common_name":    task.Context.Metadata["common_name"],
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate details: %w", err)
	}

	// Step 2: Map certificate usage (blast radius)
	usageMap, err := a.executeTool(ctx, "map_certificate_usage", map[string]interface{}{
		"certificate_id": certID,
		"common_name":    task.Context.Metadata["common_name"],
	})
	if err != nil {
		return nil, fmt.Errorf("failed to map certificate usage: %w", err)
	}

	usageData, _ := usageMap.(map[string]interface{})
	blastRadius, _ := usageData["blast_radius"].(map[string]interface{})
	riskLevel, _ := blastRadius["risk_level"].(string)

	// Generate analysis using LLM
	analysis, tokensUsed, err := a.generateCertificateAnalysis(ctx, certDetails, usageMap, task)
	if err != nil {
		a.log.Warn("failed to generate LLM analysis", "error", err)
		analysis = "Certificate analysis completed. Review the evidence for details."
		tokensUsed = 0
	}

	totalUsages, _ := blastRadius["total_usages"].(int)

	return &AgentResult{
		TaskID:         task.ID,
		AgentName:      a.name,
		Status:         AgentStatusPendingApproval,
		Plan:           nil,
		Summary:        analysis.(string),
		AffectedAssets: totalUsages,
		RiskLevel:      riskLevel,
		TokensUsed:     tokensUsed,
		Actions: []Action{
			{Type: "rotate", Label: "Rotate Certificate", Description: "Generate and execute rotation plan"},
			{Type: "schedule", Label: "Schedule Rotation", Description: "Schedule rotation for a future time"},
			{Type: "dismiss", Label: "No Action", Description: "Acknowledge without taking action"},
		},
		Evidence: map[string]interface{}{
			"certificate": certDetails,
			"usage_map":   usageMap,
		},
	}, nil
}

// rotateCertificate generates and proposes a certificate rotation plan.
func (a *CertificateAgent) rotateCertificate(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("generating certificate rotation plan", "task_id", task.ID)

	// Get certificate ID from context
	certID, ok := task.Context.Metadata["certificate_id"].(string)
	if !ok {
		return nil, fmt.Errorf("certificate_id required in task context for rotation")
	}

	// Step 1: Generate renewal plan
	renewalType := "auto"
	if rt, ok := task.Constraints["renewal_type"].(string); ok {
		renewalType = rt
	}

	strategy := "rolling"
	if s, ok := task.Constraints["strategy"].(string); ok {
		strategy = s
	}

	plan, err := a.executeTool(ctx, "generate_cert_renewal_plan", map[string]interface{}{
		"certificate_id": certID,
		"renewal_type":   renewalType,
		"strategy":       strategy,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate renewal plan: %w", err)
	}

	planData, _ := plan.(map[string]interface{})
	usageCount, _ := planData["usage_count"].(int)
	urgency, _ := planData["urgency"].(string)
	commonName, _ := planData["common_name"].(string)

	// Map urgency to risk level
	riskLevel := "medium"
	switch urgency {
	case "critical":
		riskLevel = "critical"
	case "high":
		riskLevel = "high"
	}

	// Generate execution summary using LLM
	summary, tokensUsed, err := a.generateRotationSummary(ctx, plan, task)
	if err != nil {
		summary = fmt.Sprintf("Certificate rotation plan generated for %s. %d usage locations will be updated using %s strategy.",
			commonName, usageCount, strategy)
		tokensUsed = 0
	}

	return &AgentResult{
		TaskID:         task.ID,
		AgentName:      a.name,
		Status:         AgentStatusPendingApproval,
		Plan:           plan,
		Summary:        summary.(string),
		AffectedAssets: usageCount,
		RiskLevel:      riskLevel,
		TokensUsed:     tokensUsed,
		Actions: []Action{
			{Type: "approve", Label: "Approve & Execute", Description: "Approve the plan and begin rotation"},
			{Type: "modify", Label: "Modify Plan", Description: "Edit the plan before execution"},
			{Type: "reject", Label: "Reject", Description: "Cancel the rotation"},
		},
		Evidence: map[string]interface{}{
			"rotation_plan": plan,
		},
	}, nil
}

// validateEndpoints validates TLS on specified endpoints.
func (a *CertificateAgent) validateEndpoints(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("validating TLS endpoints", "task_id", task.ID)

	endpoints, ok := task.Context.Metadata["endpoints"].([]interface{})
	if !ok || len(endpoints) == 0 {
		return nil, fmt.Errorf("endpoints list required in task context")
	}

	results := []map[string]interface{}{}
	successCount := 0
	failCount := 0

	for _, ep := range endpoints {
		endpoint, _ := ep.(string)
		if endpoint == "" {
			continue
		}

		result, err := a.executeTool(ctx, "validate_tls_handshake", map[string]interface{}{
			"endpoint": endpoint,
		})
		if err != nil {
			results = append(results, map[string]interface{}{
				"endpoint": endpoint,
				"status":   "error",
				"error":    err.Error(),
			})
			failCount++
		} else {
			resultData, _ := result.(map[string]interface{})
			results = append(results, resultData)
			if status, _ := resultData["status"].(string); status == "success" {
				successCount++
			} else {
				failCount++
			}
		}
	}

	riskLevel := "low"
	if failCount > 0 {
		riskLevel = "high"
		if failCount > len(endpoints)/2 {
			riskLevel = "critical"
		}
	}

	summary := fmt.Sprintf("TLS validation completed: %d/%d endpoints healthy, %d failed",
		successCount, len(endpoints), failCount)

	return &AgentResult{
		TaskID:         task.ID,
		AgentName:      a.name,
		Status:         AgentStatusCompleted,
		Plan:           nil,
		Summary:        summary,
		AffectedAssets: len(endpoints),
		RiskLevel:      riskLevel,
		TokensUsed:     0,
		Actions: []Action{
			{Type: "acknowledge", Label: "Acknowledge", Description: "Acknowledge validation results"},
		},
		Evidence: map[string]interface{}{
			"validation_results": results,
			"summary": map[string]interface{}{
				"total":   len(endpoints),
				"success": successCount,
				"failed":  failCount,
			},
		},
	}, nil
}

// Helper functions for LLM interactions

func (a *CertificateAgent) generateScanSummary(ctx context.Context, critical, high, medium []map[string]interface{}, task *TaskSpec) (interface{}, int, error) {
	prompt := fmt.Sprintf(`You are the QL-RF Certificate Management Agent. Summarize the certificate scan results.

## Scan Results
- Critical (expired or < 7 days): %d certificates
- High Priority (7-14 days): %d certificates
- Medium Priority (14-30 days): %d certificates

## Critical Certificates
%v

## High Priority Certificates
%v

Generate a concise executive summary (2-3 sentences) highlighting:
1. The most urgent items requiring immediate attention
2. Recommended immediate actions
3. Overall certificate health status

Be direct and actionable.`,
		len(critical), len(high), len(medium), critical, high)

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: "You are a certificate lifecycle management specialist. Provide clear, actionable summaries.",
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, 0, err
	}

	return resp.Content, resp.Usage.TotalTokens, nil
}

func (a *CertificateAgent) generateCertificateAnalysis(ctx context.Context, certDetails, usageMap interface{}, task *TaskSpec) (interface{}, int, error) {
	prompt := fmt.Sprintf(`You are the QL-RF Certificate Management Agent. Analyze this certificate and its blast radius.

## Certificate Details
%v

## Usage Map (Blast Radius)
%v

Generate a concise analysis (3-4 sentences) including:
1. Certificate health status and urgency
2. Blast radius assessment (how many services affected)
3. Risk factors (weak key, self-signed, expiring soon, etc.)
4. Recommended action

Be direct and actionable.`,
		certDetails, usageMap)

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: "You are a certificate lifecycle management specialist. Provide clear, actionable analysis.",
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, 0, err
	}

	return resp.Content, resp.Usage.TotalTokens, nil
}

func (a *CertificateAgent) generateRotationSummary(ctx context.Context, plan interface{}, task *TaskSpec) (interface{}, int, error) {
	prompt := fmt.Sprintf(`You are the QL-RF Certificate Management Agent. Summarize this rotation plan for approval.

## Rotation Plan
%v

Generate a brief summary (2-3 sentences) for the approver including:
1. What will happen (certificate being rotated)
2. Impact (number of services affected)
3. Estimated duration and risk level

Be clear and concise.`,
		plan)

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: "You are a certificate lifecycle management specialist. Provide clear rotation summaries.",
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, 0, err
	}

	return resp.Content, resp.Usage.TotalTokens, nil
}

// classifyCertificateIntent determines what action to take based on user input.
func classifyCertificateIntent(userIntent, goal string) string {
	// Simple keyword-based classification
	// In production, this could use the LLM for intent classification

	if contains(userIntent, "scan") || contains(userIntent, "expir") || contains(userIntent, "find") || contains(userIntent, "list") {
		return "scan_expiring"
	}
	if contains(userIntent, "analyz") || contains(userIntent, "detail") || contains(userIntent, "inspect") || contains(userIntent, "impact") {
		return "analyze_certificate"
	}
	if contains(userIntent, "rotat") || contains(userIntent, "renew") || contains(userIntent, "replac") || contains(userIntent, "fix") {
		return "rotate_certificate"
	}
	if contains(userIntent, "validat") || contains(userIntent, "test") || contains(userIntent, "check") || contains(userIntent, "verify") {
		return "validate_endpoints"
	}

	// Check goal as well
	if contains(goal, "scan") || contains(goal, "expir") {
		return "scan_expiring"
	}
	if contains(goal, "rotat") || contains(goal, "renew") {
		return "rotate_certificate"
	}

	return "scan_expiring" // default action
}
