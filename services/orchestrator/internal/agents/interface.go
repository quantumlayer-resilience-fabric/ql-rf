// Package agents provides the specialist agent registry and implementations.
package agents

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// =============================================================================
// Enhanced Agent Contracts (v2)
// =============================================================================

// AgentRequest provides a strongly-typed request for agent execution.
// This enhances the existing TaskSpec with additional contract fields.
type AgentRequest struct {
	// Core identifiers
	TaskID      string    `json:"task_id"`
	OrgID       uuid.UUID `json:"org_id"`
	UserID      uuid.UUID `json:"user_id,omitempty"`
	Environment string    `json:"environment"`

	// The user's natural language intent
	Intent string `json:"intent"`

	// Structured task specification (for programmatic use)
	TaskSpec json.RawMessage `json:"task_spec,omitempty"`

	// Execution context
	Context AgentContext `json:"context"`

	// Guardrails and constraints
	Guardrails AgentGuardrails `json:"guardrails,omitempty"`

	// Previous steps in multi-step execution
	PreviousSteps []StepResult `json:"previous_steps,omitempty"`
}

// AgentContext provides execution context for the agent.
type AgentContext struct {
	Platforms   []string          `json:"platforms,omitempty"`
	Regions     []string          `json:"regions,omitempty"`
	AssetFilter string            `json:"asset_filter,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
}

// AgentGuardrails defines safety constraints for agent execution.
type AgentGuardrails struct {
	MaxBatchSize     int      `json:"max_batch_size,omitempty"`
	RequireCanary    bool     `json:"require_canary,omitempty"`
	MaxRiskLevel     string   `json:"max_risk_level,omitempty"`
	ExcludedEnvs     []string `json:"excluded_envs,omitempty"`
	TimeoutMinutes   int      `json:"timeout_minutes,omitempty"`
	RequireHITL      bool     `json:"require_hitl,omitempty"`
	DryRunOnly       bool     `json:"dry_run_only,omitempty"`
}

// StepResult represents a result from a previous step.
type StepResult struct {
	StepName   string          `json:"step_name"`
	Status     string          `json:"status"`
	Output     json.RawMessage `json:"output,omitempty"`
	Error      string          `json:"error,omitempty"`
	DurationMs int64           `json:"duration_ms,omitempty"`
}

// AgentResponse provides a strongly-typed response from agent execution.
// This enhances the existing AgentResult with additional fields for debugging and audit.
type AgentResponse struct {
	// Task identification
	TaskID    string `json:"task_id"`
	AgentName string `json:"agent_name"`
	Version   string `json:"version"`

	// Status
	Status      AgentStatus `json:"status"`
	StatusCode  int         `json:"status_code,omitempty"` // HTTP-like status codes
	StatusText  string      `json:"status_text,omitempty"`

	// The generated plan (strongly-typed JSON)
	PlanJSON json.RawMessage `json:"plan_json,omitempty"`

	// Plan quality metrics
	QualityScore float64 `json:"quality_score"`      // 0-100
	RiskLevel    string  `json:"risk_level"`         // low, medium, high, critical
	Confidence   float64 `json:"confidence,omitempty"` // 0-1, agent's confidence in plan

	// Summary for human review
	Summary        string `json:"summary"`
	AffectedAssets int    `json:"affected_assets"`

	// Tool invocations made during planning
	ToolCalls []ToolInvocation `json:"tool_calls,omitempty"`

	// Available actions for HITL
	Actions []Action `json:"actions,omitempty"`

	// Evidence/context for the plan
	Evidence map[string]any `json:"evidence,omitempty"`

	// Errors encountered
	Errors []AgentError `json:"errors,omitempty"`

	// Token usage
	TokensUsed   int `json:"tokens_used"`
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`

	// Raw LLM response (always persisted for audit/debug)
	RawLLMResponse string `json:"raw_llm_response,omitempty"`

	// Parsing metadata
	ParseMethod  string `json:"parse_method,omitempty"`  // "direct", "extracted", "lenient"
	ParseWarning string `json:"parse_warning,omitempty"` // Warning if parsing required fallback
}

// ToolInvocation records a tool call made by the agent.
type ToolInvocation struct {
	ToolName   string         `json:"tool_name"`
	Parameters map[string]any `json:"parameters"`
	Result     any            `json:"result,omitempty"`
	Error      string         `json:"error,omitempty"`
	DurationMs int64          `json:"duration_ms"`
	Timestamp  string         `json:"timestamp"`
}

// AgentError represents an error during agent execution.
type AgentError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// =============================================================================
// Enhanced Agent Interface (v2)
// =============================================================================

// AgentV2 is the enhanced interface for specialist agents.
// It extends the existing Agent interface with additional methods.
type AgentV2 interface {
	Agent // Embed existing interface for backward compatibility

	// Version returns the agent version.
	Version() string

	// Capabilities returns the capabilities this agent provides.
	Capabilities() []string

	// ExecuteV2 runs the agent with the enhanced request/response types.
	ExecuteV2(ctx context.Context, req *AgentRequest) (*AgentResponse, error)
}

// =============================================================================
// Quality Score Helpers
// =============================================================================

// QualityLevel represents the quality level of a plan.
type QualityLevel string

const (
	QualityExcellent QualityLevel = "excellent" // 90-100
	QualityGood      QualityLevel = "good"      // 70-89
	QualityFair      QualityLevel = "fair"      // 50-69
	QualityPoor      QualityLevel = "poor"      // 0-49
)

// GetQualityLevel returns the quality level for a score.
func GetQualityLevel(score float64) QualityLevel {
	switch {
	case score >= 90:
		return QualityExcellent
	case score >= 70:
		return QualityGood
	case score >= 50:
		return QualityFair
	default:
		return QualityPoor
	}
}

// RiskLevel constants
const (
	RiskLevelLow      = "low"
	RiskLevelMedium   = "medium"
	RiskLevelHigh     = "high"
	RiskLevelCritical = "critical"
)

// ShouldRequireHITL determines if HITL approval should be required based on response.
func ShouldRequireHITL(resp *AgentResponse) bool {
	// Always require HITL if quality is poor
	if resp.QualityScore < 50 {
		return true
	}

	// Always require HITL for high/critical risk
	if resp.RiskLevel == RiskLevelHigh || resp.RiskLevel == RiskLevelCritical {
		return true
	}

	// Require HITL if parsing required fallback
	if resp.ParseMethod == "extracted" || resp.ParseMethod == "lenient" {
		return true
	}

	// Require HITL if there were errors
	if len(resp.Errors) > 0 {
		return true
	}

	return false
}

// =============================================================================
// Response Builders
// =============================================================================

// NewAgentResponse creates a new AgentResponse with defaults.
func NewAgentResponse(taskID, agentName, version string) *AgentResponse {
	return &AgentResponse{
		TaskID:    taskID,
		AgentName: agentName,
		Version:   version,
		Status:    AgentStatusPendingApproval,
		Actions: []Action{
			{Type: "approve", Label: "Approve & Execute", Description: "Approve the plan and begin execution"},
			{Type: "modify", Label: "Modify Plan", Description: "Edit the plan before execution"},
			{Type: "reject", Label: "Reject", Description: "Reject and cancel the task"},
		},
	}
}

// WithPlan sets the plan on the response.
func (r *AgentResponse) WithPlan(plan any) *AgentResponse {
	planJSON, _ := json.Marshal(plan)
	r.PlanJSON = planJSON
	return r
}

// WithQuality sets quality metrics on the response.
func (r *AgentResponse) WithQuality(score float64, risk string) *AgentResponse {
	r.QualityScore = score
	r.RiskLevel = risk
	return r
}

// WithTokens sets token usage on the response.
func (r *AgentResponse) WithTokens(input, output int) *AgentResponse {
	r.InputTokens = input
	r.OutputTokens = output
	r.TokensUsed = input + output
	return r
}

// WithError adds an error to the response.
func (r *AgentResponse) WithError(code, message string) *AgentResponse {
	r.Errors = append(r.Errors, AgentError{Code: code, Message: message})
	return r
}

// WithParseWarning sets a parse warning and adjusts quality.
func (r *AgentResponse) WithParseWarning(method, warning string) *AgentResponse {
	r.ParseMethod = method
	r.ParseWarning = warning
	// Cap quality score at 60 for non-direct parsing
	if method != "direct" && r.QualityScore > 60 {
		r.QualityScore = 60
	}
	return r
}
