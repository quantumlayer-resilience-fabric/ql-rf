// Package agents provides the specialist agent registry and implementations.
package agents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
)

// =============================================================================
// Fake LLM Client for Testing
// =============================================================================

// FakeLLM is a mock LLM client for testing agents.
type FakeLLM struct {
	mu        sync.Mutex
	responses []FakeLLMResponse
	idx       int
	calls     []FakeLLMCall
}

// FakeLLMResponse defines a canned response.
type FakeLLMResponse struct {
	Content    string
	ToolCalls  []llm.ToolCall
	Error      error
	Latency    time.Duration
	InputToks  int
	OutputToks int
}

// FakeLLMCall records a call made to the fake LLM.
type FakeLLMCall struct {
	SystemPrompt string
	Messages     []llm.Message
	Tools        []llm.ToolDefinition
	Timestamp    time.Time
}

// NewFakeLLM creates a new fake LLM with canned responses.
func NewFakeLLM(responses ...string) *FakeLLM {
	f := &FakeLLM{
		responses: make([]FakeLLMResponse, len(responses)),
		calls:     make([]FakeLLMCall, 0),
	}
	for i, r := range responses {
		f.responses[i] = FakeLLMResponse{
			Content:    r,
			InputToks:  100,
			OutputToks: 50,
		}
	}
	return f
}

// NewFakeLLMWithResponses creates a fake LLM with detailed responses.
func NewFakeLLMWithResponses(responses []FakeLLMResponse) *FakeLLM {
	return &FakeLLM{
		responses: responses,
		calls:     make([]FakeLLMCall, 0),
	}
}

// Complete implements llm.Client.
func (f *FakeLLM) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Record the call (handle nil request for test simplicity)
	call := FakeLLMCall{
		Timestamp: time.Now(),
	}
	if req != nil {
		call.SystemPrompt = req.SystemPrompt
		call.Messages = req.Messages
	}
	f.calls = append(f.calls, call)

	// Check if we have more responses
	if f.idx >= len(f.responses) {
		return nil, fmt.Errorf("fake LLM: no more responses (received %d calls, have %d responses)", f.idx+1, len(f.responses))
	}

	resp := f.responses[f.idx]
	f.idx++

	// Simulate latency if specified
	if resp.Latency > 0 {
		time.Sleep(resp.Latency)
	}

	// Return error if specified
	if resp.Error != nil {
		return nil, resp.Error
	}

	return &llm.CompletionResponse{
		Content:      resp.Content,
		ToolCalls:    resp.ToolCalls,
		Usage: llm.Usage{
			InputTokens:  resp.InputToks,
			OutputTokens: resp.OutputToks,
			TotalTokens:  resp.InputToks + resp.OutputToks,
		},
		StopReason:   "end_turn",
		FinishReason: "stop",
		Latency:      resp.Latency,
	}, nil
}

// CompleteWithTools implements llm.Client.
func (f *FakeLLM) CompleteWithTools(ctx context.Context, req *llm.CompletionRequest, tools []llm.ToolDefinition) (*llm.CompletionResponse, error) {
	f.mu.Lock()
	// Record tools in the call (handle nil request for test simplicity)
	call := FakeLLMCall{
		Tools:     tools,
		Timestamp: time.Now(),
	}
	if req != nil {
		call.SystemPrompt = req.SystemPrompt
		call.Messages = req.Messages
	}
	f.calls = append(f.calls, call)
	f.mu.Unlock()

	// Use same logic as Complete
	return f.Complete(ctx, req)
}

// Provider implements llm.Client.
func (f *FakeLLM) Provider() string {
	return "fake"
}

// Model implements llm.Client.
func (f *FakeLLM) Model() string {
	return "fake-model"
}

// Calls returns all recorded calls.
func (f *FakeLLM) Calls() []FakeLLMCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]FakeLLMCall{}, f.calls...)
}

// LastCall returns the most recent call.
func (f *FakeLLM) LastCall() *FakeLLMCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) == 0 {
		return nil
	}
	return &f.calls[len(f.calls)-1]
}

// CallCount returns the number of calls made.
func (f *FakeLLM) CallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// Reset clears recorded calls and resets response index.
func (f *FakeLLM) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.idx = 0
	f.calls = make([]FakeLLMCall, 0)
}

// =============================================================================
// Test Fixtures
// =============================================================================

// DriftPlanFixture returns a valid drift remediation plan JSON.
func DriftPlanFixture() string {
	return `{
		"plan": {
			"name": "drift-remediation-plan",
			"phases": [
				{
					"name": "canary",
					"batch_percent": 5,
					"wait_minutes": 10,
					"health_check": true
				},
				{
					"name": "early_rollout",
					"batch_percent": 25,
					"wait_minutes": 5,
					"health_check": true
				},
				{
					"name": "full_rollout",
					"batch_percent": 100,
					"wait_minutes": 0,
					"health_check": true
				}
			],
			"rollback_policy": {
				"type": "automatic",
				"threshold": 0.05,
				"window_minutes": 15
			},
			"notifications": {
				"slack_channel": "#ops-alerts",
				"on_start": true,
				"on_complete": true,
				"on_failure": true
			}
		},
		"summary": "Remediate drift for 10 assets in production",
		"risk_assessment": {
			"level": "medium",
			"factors": ["production environment", "10 assets affected"],
			"mitigations": ["canary deployment", "automatic rollback"]
		}
	}`
}

// PatchPlanFixture returns a valid patch rollout plan JSON.
func PatchPlanFixture() string {
	return `{
		"plan": {
			"name": "patch-rollout-plan",
			"patches": [
				{"kb": "KB5001234", "severity": "critical", "cve": "CVE-2024-1234"}
			],
			"phases": [
				{"name": "canary", "batch_percent": 5, "wait_minutes": 15},
				{"name": "rollout", "batch_percent": 95, "wait_minutes": 0}
			],
			"rollback_policy": {"type": "automatic", "threshold": 0.03}
		},
		"summary": "Apply critical security patch KB5001234",
		"risk_assessment": {"level": "high", "factors": ["critical CVE"]}
	}`
}

// CompliancePlanFixture returns a valid compliance audit plan JSON.
func CompliancePlanFixture() string {
	return `{
		"plan": {
			"name": "compliance-audit-plan",
			"framework": "CIS",
			"controls": [
				{"id": "CIS-1.1", "name": "Password Policy", "status": "pass"},
				{"id": "CIS-1.2", "name": "SSH Configuration", "status": "fail", "remediation": "Disable password auth"}
			],
			"score": 85.5
		},
		"summary": "CIS compliance audit with 85.5% score",
		"risk_assessment": {"level": "low"}
	}`
}

// MalformedJSONFixture returns intentionally malformed JSON.
func MalformedJSONFixture() string {
	return `{
		"plan": {
			"name": "broken-plan"
			"missing_comma": true
		}
	}`
}

// TruncatedJSONFixture returns truncated JSON (simulating token limit).
func TruncatedJSONFixture() string {
	return `{
		"plan": {
			"name": "truncated-plan",
			"phases": [
				{"name": "canary", "batch_percent": 5},
				{"name": "rollout", "batch_per`
}

// WrongTypesJSONFixture returns JSON with wrong field types.
func WrongTypesJSONFixture() string {
	return `{
		"plan": {
			"name": "wrong-types-plan",
			"phases": [
				{"name": "canary", "batch_percent": "a lot"}
			]
		}
	}`
}

// ExtraFieldsJSONFixture returns JSON with hallucinated fields.
func ExtraFieldsJSONFixture() string {
	return `{
		"plan": {
			"name": "extra-fields-plan",
			"phases": [
				{"name": "canary", "batch_percent": 5}
			],
			"hallucinated_field": "this should not be here",
			"another_fake_field": 12345
		},
		"summary": "Plan with extra fields"
	}`
}

// WrappedJSONFixture returns JSON wrapped in markdown code blocks.
func WrappedJSONFixture() string {
	return "Here's the plan:\n\n```json\n" + DriftPlanFixture() + "\n```\n\nLet me know if you need changes."
}

// =============================================================================
// Test Helpers
// =============================================================================

// TestTaskSpec creates a TaskSpec for testing.
func TestTaskSpec(taskType TaskType, goal string) *TaskSpec {
	return &TaskSpec{
		ID:          "test-task-123",
		TaskType:    taskType,
		Goal:        goal,
		UserIntent:  goal,
		OrgID:       "test-org-id",
		UserID:      "test-user-id",
		Environment: "staging",
		Context: TaskContext{
			Platforms: []string{"aws"},
			Regions:   []string{"us-east-1"},
		},
		RiskLevel:      "medium",
		HITLRequired:   true,
		TimeoutMinutes: 30,
	}
}

// TestAgentRequest creates an AgentRequest for testing.
func TestAgentRequest(intent string) *AgentRequest {
	return &AgentRequest{
		TaskID:      "test-task-123",
		OrgID:       mustParseUUID("11111111-1111-1111-1111-111111111111"),
		Environment: "staging",
		Intent:      intent,
		Context: AgentContext{
			Platforms: []string{"aws"},
			Regions:   []string{"us-east-1"},
		},
		Guardrails: AgentGuardrails{
			MaxBatchSize:   25,
			RequireCanary:  true,
			MaxRiskLevel:   "high",
			TimeoutMinutes: 30,
		},
	}
}

func mustParseUUID(s string) [16]byte {
	var id [16]byte
	// Simple UUID parsing for test fixtures
	copy(id[:], []byte(s)[:16])
	return id
}

// =============================================================================
// Assertions
// =============================================================================

// AssertAgentResponse provides assertion helpers for AgentResponse.
type AssertAgentResponse struct {
	Response *AgentResponse
}

// Assert creates a new assertion helper.
func Assert(resp *AgentResponse) *AssertAgentResponse {
	return &AssertAgentResponse{Response: resp}
}

// HasStatus checks if the response has the expected status.
func (a *AssertAgentResponse) HasStatus(expected AgentStatus) bool {
	return a.Response.Status == expected
}

// HasQualityAbove checks if quality score is above threshold.
func (a *AssertAgentResponse) HasQualityAbove(threshold float64) bool {
	return a.Response.QualityScore > threshold
}

// HasNoErrors checks if there are no errors.
func (a *AssertAgentResponse) HasNoErrors() bool {
	return len(a.Response.Errors) == 0
}

// HasToolCall checks if a specific tool was called.
func (a *AssertAgentResponse) HasToolCall(toolName string) bool {
	for _, tc := range a.Response.ToolCalls {
		if tc.ToolName == toolName {
			return true
		}
	}
	return false
}

// RequiresHITL checks if HITL approval is required.
func (a *AssertAgentResponse) RequiresHITL() bool {
	return ShouldRequireHITL(a.Response)
}
