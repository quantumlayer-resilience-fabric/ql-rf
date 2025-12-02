// Package meta provides the meta-prompt engine for intent parsing and task planning.
package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/agents"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
)

// Engine is the meta-prompt engine that converts user intent to structured TaskSpecs.
type Engine struct {
	llm           llm.Client
	agentRegistry *agents.Registry
	log           *logger.Logger
}

// NewEngine creates a new meta-prompt engine.
func NewEngine(llmClient llm.Client, agentRegistry *agents.Registry, log *logger.Logger) *Engine {
	return &Engine{
		llm:           llmClient,
		agentRegistry: agentRegistry,
		log:           log.WithComponent("meta-engine"),
	}
}

// IntentRequest represents a user's natural language request.
type IntentRequest struct {
	UserIntent  string                 `json:"user_intent"`
	OrgID       string                 `json:"org_id"`
	UserID      string                 `json:"user_id"`
	Environment string                 `json:"environment,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// ParsedIntent represents the parsed intent before task creation.
type ParsedIntent struct {
	TaskType       agents.TaskType        `json:"task_type"`
	Goal           string                 `json:"goal"`
	Confidence     float64                `json:"confidence"`
	Agents         []string               `json:"agents"`
	ToolsRequired  []string               `json:"tools_required"`
	RiskLevel      string                 `json:"risk_level"`
	HITLRequired   bool                   `json:"hitl_required"`
	Environment    string                 `json:"environment"`
	Scope          IntentScope            `json:"scope"`
	Constraints    map[string]interface{} `json:"constraints"`
	Reasoning      string                 `json:"reasoning"`
}

// IntentScope defines the scope of the parsed intent.
type IntentScope struct {
	Platforms   []string `json:"platforms,omitempty"`
	Regions     []string `json:"regions,omitempty"`
	AssetFilter string   `json:"asset_filter,omitempty"`
}

// ParseIntent converts a natural language request into a structured ParsedIntent.
func (e *Engine) ParseIntent(ctx context.Context, req *IntentRequest) (*ParsedIntent, error) {
	e.log.Info("parsing user intent",
		"user_id", req.UserID,
		"org_id", req.OrgID,
		"intent_length", len(req.UserIntent),
	)

	// Build the intent parsing prompt
	prompt := e.buildIntentParsingPrompt(req)

	// Call LLM to parse intent
	resp, err := e.llm.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: intentParserSystemPrompt,
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1, // Low temperature for consistent parsing
	})
	if err != nil {
		return nil, fmt.Errorf("LLM completion failed: %w", err)
	}

	// Parse the LLM response
	parsed, err := e.parseIntentResponse(resp.Content)
	if err != nil {
		e.log.Warn("failed to parse LLM response, using fallback",
			"error", err,
			"response", resp.Content,
		)
		return e.fallbackIntentParsing(req)
	}

	// Validate and enrich the parsed intent
	e.enrichParsedIntent(parsed, req)

	e.log.Info("intent parsed successfully",
		"task_type", parsed.TaskType,
		"risk_level", parsed.RiskLevel,
		"hitl_required", parsed.HITLRequired,
		"agents", parsed.Agents,
		"confidence", parsed.Confidence,
	)

	return parsed, nil
}

// CreateTaskSpec creates a full TaskSpec from parsed intent.
func (e *Engine) CreateTaskSpec(ctx context.Context, parsed *ParsedIntent, req *IntentRequest) (*agents.TaskSpec, error) {
	taskID := uuid.New().String()

	spec := &agents.TaskSpec{
		ID:          taskID,
		TaskType:    parsed.TaskType,
		Goal:        parsed.Goal,
		UserIntent:  req.UserIntent,
		OrgID:       req.OrgID,
		UserID:      req.UserID,
		Environment: parsed.Environment,
		Context: agents.TaskContext{
			Platforms:   parsed.Scope.Platforms,
			Regions:     parsed.Scope.Regions,
			AssetFilter: parsed.Scope.AssetFilter,
			Metadata:    req.Context,
		},
		ToolsRequired: parsed.ToolsRequired,
		Validation: agents.ValidationSpec{
			Schema:   e.getSchemaForTaskType(parsed.TaskType),
			Policies: e.getPoliciesForTaskType(parsed.TaskType, parsed.Environment),
			Constraints: map[string]interface{}{
				"max_batch_size":    10,
				"require_canary":    parsed.Environment == "production",
				"rollback_on_failure": true,
			},
		},
		RiskLevel:      parsed.RiskLevel,
		HITLRequired:   parsed.HITLRequired,
		TimeoutMinutes: e.getTimeoutForTaskType(parsed.TaskType),
		Constraints:    parsed.Constraints,
	}

	// Merge any additional constraints from request context
	if parsed.Constraints == nil {
		spec.Constraints = make(map[string]interface{})
	}

	e.log.Info("created task spec",
		"task_id", taskID,
		"task_type", spec.TaskType,
		"goal", spec.Goal,
	)

	return spec, nil
}

// ProcessRequest is the main entry point - parses intent and creates TaskSpec.
func (e *Engine) ProcessRequest(ctx context.Context, req *IntentRequest) (*agents.TaskSpec, error) {
	// Parse the intent
	parsed, err := e.ParseIntent(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to parse intent: %w", err)
	}

	// Create the task specification
	spec, err := e.CreateTaskSpec(ctx, parsed, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create task spec: %w", err)
	}

	return spec, nil
}

// buildIntentParsingPrompt creates the prompt for intent parsing.
func (e *Engine) buildIntentParsingPrompt(req *IntentRequest) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("User Request: %s\n\n", req.UserIntent))

	if req.Environment != "" {
		sb.WriteString(fmt.Sprintf("Environment: %s\n", req.Environment))
	}

	if len(req.Context) > 0 {
		sb.WriteString("Additional Context:\n")
		for k, v := range req.Context {
			sb.WriteString(fmt.Sprintf("  - %s: %v\n", k, v))
		}
	}

	// Add available agents
	sb.WriteString("\nAvailable Agents:\n")
	for _, info := range e.agentRegistry.AgentInfo() {
		sb.WriteString(fmt.Sprintf("  - %s: %s (tasks: %v)\n", info.Name, info.Description, info.SupportedTasks))
	}

	sb.WriteString("\nPlease analyze this request and output a JSON object with the parsed intent.")

	return sb.String()
}

// parseIntentResponse parses the LLM response into ParsedIntent.
func (e *Engine) parseIntentResponse(response string) (*ParsedIntent, error) {
	// Try to extract JSON from the response
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var parsed ParsedIntent
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &parsed, nil
}

// fallbackIntentParsing provides a basic intent parsing when LLM fails.
func (e *Engine) fallbackIntentParsing(req *IntentRequest) (*ParsedIntent, error) {
	intent := strings.ToLower(req.UserIntent)

	// Simple keyword-based fallback
	var taskType agents.TaskType
	var agentName string

	switch {
	case strings.Contains(intent, "drift") || strings.Contains(intent, "remediat"):
		taskType = agents.TaskTypeDriftRemediation
		agentName = "drift_agent"
	case strings.Contains(intent, "patch") || strings.Contains(intent, "update") || strings.Contains(intent, "upgrade"):
		taskType = agents.TaskTypePatchRollout
		agentName = "patch_agent"
	case strings.Contains(intent, "compliance") || strings.Contains(intent, "audit"):
		taskType = agents.TaskTypeComplianceAudit
		agentName = "compliance_agent"
	case strings.Contains(intent, "incident") || strings.Contains(intent, "investigate"):
		taskType = agents.TaskTypeIncidentResponse
		agentName = "incident_agent"
	case strings.Contains(intent, "dr") || strings.Contains(intent, "disaster") || strings.Contains(intent, "failover"):
		taskType = agents.TaskTypeDRDrill
		agentName = "dr_agent"
	case strings.Contains(intent, "cost") || strings.Contains(intent, "optimi"):
		taskType = agents.TaskTypeCostOptimization
		agentName = "cost_agent"
	case strings.Contains(intent, "security") || strings.Contains(intent, "vulnerab"):
		taskType = agents.TaskTypeSecurityScan
		agentName = "security_agent"
	case strings.Contains(intent, "image") || strings.Contains(intent, "golden"):
		taskType = agents.TaskTypeImageManagement
		agentName = "image_agent"
	default:
		taskType = agents.TaskTypeDriftRemediation
		agentName = "drift_agent"
	}

	// Determine environment
	env := req.Environment
	if env == "" {
		if strings.Contains(intent, "prod") {
			env = "production"
		} else if strings.Contains(intent, "staging") || strings.Contains(intent, "stage") {
			env = "staging"
		} else {
			env = "development"
		}
	}

	// Determine risk level
	riskLevel := "medium"
	if env == "production" {
		riskLevel = "high"
	} else if env == "development" {
		riskLevel = "low"
	}

	return &ParsedIntent{
		TaskType:      taskType,
		Goal:          req.UserIntent,
		Confidence:    0.5, // Low confidence for fallback
		Agents:        []string{agentName},
		ToolsRequired: e.getToolsForAgent(agentName),
		RiskLevel:     riskLevel,
		HITLRequired:  riskLevel == "high" || riskLevel == "critical",
		Environment:   env,
		Reasoning:     "Fallback parsing based on keywords",
	}, nil
}

// enrichParsedIntent adds additional information to the parsed intent.
func (e *Engine) enrichParsedIntent(parsed *ParsedIntent, req *IntentRequest) {
	// Ensure environment is set
	if parsed.Environment == "" {
		parsed.Environment = req.Environment
		if parsed.Environment == "" {
			parsed.Environment = "development"
		}
	}

	// Adjust risk level based on environment
	if parsed.Environment == "production" && parsed.RiskLevel != "critical" {
		if parsed.RiskLevel == "low" {
			parsed.RiskLevel = "medium"
		} else if parsed.RiskLevel == "medium" {
			parsed.RiskLevel = "high"
		}
	}

	// Ensure HITL for high-risk operations
	if parsed.RiskLevel == "high" || parsed.RiskLevel == "critical" {
		parsed.HITLRequired = true
	}

	// Add default constraints if missing
	if parsed.Constraints == nil {
		parsed.Constraints = make(map[string]interface{})
	}

	// Add environment-specific constraints
	if parsed.Environment == "production" {
		if _, ok := parsed.Constraints["require_canary"]; !ok {
			parsed.Constraints["require_canary"] = true
		}
		if _, ok := parsed.Constraints["max_batch_percent"]; !ok {
			parsed.Constraints["max_batch_percent"] = 10
		}
	}
}

// getToolsForAgent returns the tools for a given agent.
func (e *Engine) getToolsForAgent(agentName string) []string {
	agent, ok := e.agentRegistry.Get(agentName)
	if !ok {
		return nil
	}
	return agent.RequiredTools()
}

// getSchemaForTaskType returns the JSON schema name for a task type.
func (e *Engine) getSchemaForTaskType(taskType agents.TaskType) string {
	schemas := map[agents.TaskType]string{
		agents.TaskTypeDriftRemediation: "drift_remediation_v1",
		agents.TaskTypePatchRollout:     "patch_rollout_v1",
		agents.TaskTypeComplianceAudit:  "compliance_report_v1",
		agents.TaskTypeIncidentResponse: "incident_analysis_v1",
		agents.TaskTypeDRDrill:          "dr_runbook_v1",
		agents.TaskTypeCostOptimization: "cost_report_v1",
		agents.TaskTypeSecurityScan:     "security_report_v1",
		agents.TaskTypeImageManagement:  "image_spec_v1",
	}
	if schema, ok := schemas[taskType]; ok {
		return schema
	}
	return "generic_v1"
}

// getPoliciesForTaskType returns OPA policies for a task type and environment.
func (e *Engine) getPoliciesForTaskType(taskType agents.TaskType, env string) []string {
	policies := []string{"base_safety"}

	// Add task-specific policies
	switch taskType {
	case agents.TaskTypeDriftRemediation, agents.TaskTypePatchRollout:
		policies = append(policies, "rollout_safety", "batch_size_limits")
	case agents.TaskTypeDRDrill:
		policies = append(policies, "dr_safety")
	case agents.TaskTypeComplianceAudit:
		policies = append(policies, "compliance_scope")
	}

	// Add environment-specific policies
	if env == "production" {
		policies = append(policies, "production_safety", "canary_required")
	}

	return policies
}

// getTimeoutForTaskType returns the default timeout for a task type.
func (e *Engine) getTimeoutForTaskType(taskType agents.TaskType) int {
	timeouts := map[agents.TaskType]int{
		agents.TaskTypeDriftRemediation: 60,
		agents.TaskTypePatchRollout:     120,
		agents.TaskTypeComplianceAudit:  30,
		agents.TaskTypeIncidentResponse: 30,
		agents.TaskTypeDRDrill:          90,
		agents.TaskTypeCostOptimization: 30,
		agents.TaskTypeSecurityScan:     45,
		agents.TaskTypeImageManagement:  60,
	}
	if timeout, ok := timeouts[taskType]; ok {
		return timeout
	}
	return 30
}

// extractJSON attempts to extract a JSON object from a string.
func extractJSON(s string) string {
	// Find the first { and last }
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")

	if start == -1 || end == -1 || end <= start {
		return ""
	}

	return s[start : end+1]
}

// System prompt for intent parsing
const intentParserSystemPrompt = `You are the QL-RF Task Planner. Given a user request about infrastructure management, generate a structured intent analysis.

Available task types:
- drift_remediation: Fix configuration drift
- patch_rollout: Apply patches/updates
- compliance_audit: Run compliance checks
- incident_investigation: Analyze incidents
- dr_drill: Test disaster recovery
- cost_optimization: Optimize costs
- security_scan: Security analysis
- image_management: Golden image operations

Risk levels:
- low: Read-only queries, no changes
- medium: Generates plans (no execution)
- high: Modifies non-prod resources OR any prod read-write
- critical: Modifies production infrastructure

Rules:
1. Always require HITL approval for high/critical risk
2. Production changes are always high or critical risk
3. Prefer smallest scope possible
4. Include rollback constraints for any changes
5. Identify the most appropriate agent(s) for the task

Output a JSON object with these fields:
{
  "task_type": "drift_remediation",
  "goal": "Clear description of what will be accomplished",
  "confidence": 0.95,
  "agents": ["drift_agent"],
  "tools_required": ["query_assets", "get_drift_status"],
  "risk_level": "high",
  "hitl_required": true,
  "environment": "production",
  "scope": {
    "platforms": ["aws"],
    "regions": ["us-east-1"],
    "asset_filter": "role:web-server"
  },
  "constraints": {
    "require_canary": true,
    "max_batch_percent": 10
  },
  "reasoning": "Brief explanation of the analysis"
}`

// TaskPlanningResult contains the full result of intent processing.
type TaskPlanningResult struct {
	TaskSpec     *agents.TaskSpec `json:"task_spec"`
	ParsedIntent *ParsedIntent    `json:"parsed_intent"`
	ProcessedAt  time.Time        `json:"processed_at"`
	TokensUsed   int              `json:"tokens_used"`
}
