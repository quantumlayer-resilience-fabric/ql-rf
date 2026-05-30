// Stub LLM provider — Phase B.1 of Mission Control (AI-002).
//
// Deterministic, hardcoded canned responses. Never reaches external services.
// NOT FOR PRODUCTION. Used by Phase B test fixtures, CI E2E, and local dev so
// the orchestrator's execute path can be exercised without burning tokens or
// being non-deterministic. See docs/E2E-011-ai-mission-control.md §3.1.
//
// Properties:
//   * deterministic = true
//   * external_calls = false
//   * tool_calls = none (empty []) — Phase B.2 may introduce stubbed tool calls
//   * audit marker = "_stub": true is embedded in every canned JSON payload
//
// The orchestrator's executeTask handler also short-circuits when this
// provider is active (see services/orchestrator/internal/handlers/handlers.go),
// so a stub-driven prompt creates ai_tasks + ai_plans rows in `awaiting_approval`
// state and never reaches agent.Execute or Temporal workflow progress.

package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// ProviderStub is the provider-name constant for the stub LLM. The constant
// exists so the orchestrator's executeTask handler can guard against it
// (`if h.cfg.LLM.Provider == llm.ProviderStub { ... }`) without scattering the
// raw string literal across packages.
const ProviderStub = "stub"

// stubClient is a deterministic LLM client that returns canned responses keyed
// on the system-prompt prefix (which identifies the orchestrator code path that
// invoked us — intent parser, patch agent, drift agent, etc.) and a secondary
// user-intent keyword switch within the intent-parser branch.
type stubClient struct {
	log *logger.Logger
}

// newStubClient constructs the stub provider and emits a loud WARN log so its
// presence is grep-able in any post-mortem. The warning fires once per process
// lifetime, at construction — not per request.
//
// deterministic = true, external_calls = false.
func newStubClient(_ config.LLMConfig, log *logger.Logger) (Client, error) {
	c := &stubClient{log: log.WithComponent("llm-stub")}
	c.log.Warn("STUB PROVIDER ENABLED — provider=stub model=stub-canned deterministic=true external_calls=false. Responses are canned. DO NOT USE IN PRODUCTION.")
	return c, nil
}

// Provider returns the provider name. Always "stub".
func (c *stubClient) Provider() string { return ProviderStub }

// Model returns the model name. Always "stub-canned".
func (c *stubClient) Model() string { return "stub-canned" }

// Complete returns a canned response keyed on the system prompt and the last
// user message. Honours context cancellation.
//
// deterministic = true, external_calls = false.
func (c *stubClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	content := c.cannedResponse(req)
	return &CompletionResponse{
		Content:      content,
		ToolCalls:    nil,
		Usage:        Usage{InputTokens: 250, OutputTokens: 400, TotalTokens: 650},
		StopReason:   "end_turn",
		FinishReason: "stop",
		Latency:      50 * time.Millisecond,
	}, nil
}

// CompleteWithTools mirrors Complete but never returns any tool calls. The
// stub's canned plans embed action descriptions directly in `Content` — Phase
// B.1 does not exercise the agent tool-loop path.
//
// deterministic = true, external_calls = false.
func (c *stubClient) CompleteWithTools(ctx context.Context, req *CompletionRequest, _ []ToolDefinition) (*CompletionResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	content := c.cannedResponse(req)
	return &CompletionResponse{
		Content:      content,
		ToolCalls:    []ToolCall{},
		Usage:        Usage{InputTokens: 250, OutputTokens: 400, TotalTokens: 650},
		StopReason:   "end_turn",
		FinishReason: "stop",
		Latency:      50 * time.Millisecond,
	}, nil
}

// cannedResponse picks the right canned JSON envelope for the call. It routes
// on the system prompt's leading marker — each orchestrator-side prompt starts
// with a distinctive phrase the stub can match on.
func (c *stubClient) cannedResponse(req *CompletionRequest) string {
	prompt := req.SystemPrompt
	switch {
	case strings.Contains(prompt, "QL-RF Task Planner"):
		return c.intentEnvelope(lastUserMessage(req))
	case strings.Contains(prompt, "patch management specialist"):
		return c.patchPlan()
	case strings.Contains(prompt, "drift") && strings.Contains(prompt, "specialist"):
		return c.driftPlan()
	default:
		return c.genericPlan()
	}
}

// lastUserMessage returns the content of the most recent "user"-role message,
// falling back to the first message of any role, then to the empty string.
func lastUserMessage(req *CompletionRequest) string {
	if req == nil {
		return ""
	}
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			return req.Messages[i].Content
		}
	}
	if len(req.Messages) > 0 {
		return req.Messages[0].Content
	}
	return ""
}

// intentEnvelope returns a canned JSON envelope matching what the meta engine's
// intent parser expects. Keyword-matches the user intent to pick the task type;
// risk_level is always "high" so the resulting plan is stored with state
// `awaiting_approval` and surfaces in the pending decisions rail.
//
// Default: drift_remediation (the safest fallback — drift agent is more
// table-driven and less likely to invoke cloud SDKs than the patch agent).
func (c *stubClient) intentEnvelope(intent string) string {
	lower := strings.ToLower(intent)
	taskType := "drift_remediation"
	agents := `["drift_agent"]`
	tools := `["query_assets", "get_drift_status", "analyze_drift"]`
	switch {
	case strings.Contains(lower, "cve") || strings.Contains(lower, "patch"):
		taskType = "patch_rollout"
		agents = `["patch_agent"]`
		tools = `["query_assets", "compare_versions", "generate_patch_plan"]`
	case strings.Contains(lower, "drift"):
		taskType = "drift_remediation"
	case strings.Contains(lower, "cert") || strings.Contains(lower, "rotate"):
		taskType = "patch_rollout"
		agents = `["certificate_agent"]`
		tools = `["list_certificates", "propose_cert_rotation"]`
	}
	return fmt.Sprintf(`{
  "_stub": true,
  "task_type": %q,
  "goal": "Stub-generated plan for: %s",
  "confidence": 0.92,
  "agents": %s,
  "tools_required": %s,
  "risk_level": "high",
  "hitl_required": true,
  "environment": "staging",
  "scope": {
    "platforms": ["aws"],
    "regions": ["us-east-1"]
  },
  "constraints": {
    "require_canary": true,
    "max_batch_percent": 10
  },
  "reasoning": "Stub provider response. Deterministic. No external calls. DO NOT USE IN PRODUCTION."
}`, taskType, sanitize(intent), agents, tools)
}

// patchPlan returns a canned patch_rollout_v1-shaped plan. The patch agent has
// a defaultPatchPlan fallback so even minimal output succeeds.
func (c *stubClient) patchPlan() string {
	return `{
  "_stub": true,
  "summary": "Stub patch rollout plan",
  "patches": [{"asset_id": "stub-asset-1", "from_version": "1.0.0", "to_version": "1.0.1"}],
  "schedule": {"start_at": "2026-06-01T00:00:00Z", "wave_minutes": 30}
}`
}

// driftPlan returns a canned drift_remediation_v1-shaped plan.
func (c *stubClient) driftPlan() string {
	return `{
  "_stub": true,
  "summary": "Stub drift remediation plan",
  "blast_radius": {"assets": 3, "environment": "staging"},
  "phases": ["canary", "monitor", "full_rollout"]
}`
}

// genericPlan returns a minimal fallback plan for unmatched system prompts.
func (c *stubClient) genericPlan() string {
	return `{
  "_stub": true,
  "summary": "Stub generic plan",
  "blast_radius": {"assets": 1, "environment": "staging"},
  "phases": ["canary"]
}`
}

// sanitize trims and escapes characters that would break the embedded JSON
// string literal.
func sanitize(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}
