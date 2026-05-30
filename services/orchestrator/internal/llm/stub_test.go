package llm

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// newStubForTest constructs the stub provider with a discarded logger. We
// don't need to assert on the WARN log here — the integration test (orchestrator
// startup in compose) already covers that.
func newStubForTest(t *testing.T) Client {
	t.Helper()
	log := logger.New("error", "text")
	c, err := newStubClient(config.LLMConfig{Provider: "stub"}, log)
	if err != nil {
		t.Fatalf("newStubClient: %v", err)
	}
	return c
}

// intentReq builds a CompletionRequest whose system prompt mimics the meta
// engine's intent parser (so the stub routes into intentEnvelope).
func intentReq(userIntent string) *CompletionRequest {
	return &CompletionRequest{
		SystemPrompt: "You are the QL-RF Task Planner. Given a user request about infrastructure management...",
		Messages: []Message{
			{Role: "user", Content: userIntent},
		},
	}
}

// parseIntentJSON unmarshals the stub's canned content into a map so tests can
// assert specific fields without coupling to formatting.
func parseIntentJSON(t *testing.T, content string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(content), &m); err != nil {
		t.Fatalf("stub content is not valid JSON: %v\n---\n%s", err, content)
	}
	return m
}

// TestStub_CompleteCVEIntent — CVE / patch intent routes to patch_rollout with
// high risk and HITL required; the _stub marker is present.
func TestStub_CompleteCVEIntent(t *testing.T) {
	c := newStubForTest(t)
	resp, err := c.Complete(context.Background(), intentReq("Patch CVE-2024-3094 on production assets"))
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	m := parseIntentJSON(t, resp.Content)
	if m["task_type"] != "patch_rollout" {
		t.Errorf("task_type = %v, want patch_rollout", m["task_type"])
	}
	if m["risk_level"] != "high" {
		t.Errorf("risk_level = %v, want high", m["risk_level"])
	}
	if m["hitl_required"] != true {
		t.Errorf("hitl_required = %v, want true", m["hitl_required"])
	}
	if m["_stub"] != true {
		t.Errorf("_stub marker missing — got %v", m["_stub"])
	}
}

// TestStub_CompleteDriftIntent — drift intent routes to drift_remediation.
func TestStub_CompleteDriftIntent(t *testing.T) {
	c := newStubForTest(t)
	resp, err := c.Complete(context.Background(), intentReq("Analyze drift across azure production sites"))
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	m := parseIntentJSON(t, resp.Content)
	if m["task_type"] != "drift_remediation" {
		t.Errorf("task_type = %v, want drift_remediation", m["task_type"])
	}
	if m["_stub"] != true {
		t.Errorf("_stub marker missing")
	}
}

// TestStub_CompleteUnknownIntent — an unmatched / nonsense intent falls back
// to the safest task type (drift_remediation, which is more table-driven and
// less likely to touch cloud SDKs).
func TestStub_CompleteUnknownIntent(t *testing.T) {
	c := newStubForTest(t)
	resp, err := c.Complete(context.Background(), intentReq("xyzzy lorem ipsum"))
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp == nil {
		t.Fatal("Complete returned nil response")
	}
	m := parseIntentJSON(t, resp.Content)
	if m["task_type"] != "drift_remediation" {
		t.Errorf("fallback task_type = %v, want drift_remediation", m["task_type"])
	}
}

// TestStub_CompleteWithToolsReturnsNoToolCalls — CompleteWithTools never
// synthesises tool calls in B.1. The plan's actions are embedded in Content.
func TestStub_CompleteWithToolsReturnsNoToolCalls(t *testing.T) {
	c := newStubForTest(t)
	tools := []ToolDefinition{
		{Name: "query_assets", Description: "Query asset inventory"},
		{Name: "analyze_drift", Description: "Analyze drift"},
	}
	resp, err := c.CompleteWithTools(context.Background(), intentReq("Patch CVE-2024-3094"), tools)
	if err != nil {
		t.Fatalf("CompleteWithTools: %v", err)
	}
	if got := len(resp.ToolCalls); got != 0 {
		t.Errorf("ToolCalls length = %d, want 0 (stub does not synthesise tool calls in B.1)", got)
	}
	if !strings.Contains(resp.Content, "_stub") {
		t.Errorf("response Content missing _stub marker: %s", resp.Content)
	}
}

// TestStub_CompleteRespectsCancelledContext — a pre-cancelled context yields
// ctx.Err() with no response body, matching the live providers' behaviour.
func TestStub_CompleteRespectsCancelledContext(t *testing.T) {
	c := newStubForTest(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	resp, err := c.Complete(ctx, intentReq("Patch CVE-2024-3094"))
	if err == nil {
		t.Fatal("Complete with cancelled ctx: expected error, got nil")
	}
	if resp != nil {
		t.Errorf("expected nil response on cancelled ctx, got %+v", resp)
	}

	// CompleteWithTools should behave the same.
	resp, err = c.CompleteWithTools(ctx, intentReq("Patch CVE-2024-3094"), nil)
	if err == nil {
		t.Fatal("CompleteWithTools with cancelled ctx: expected error, got nil")
	}
	if resp != nil {
		t.Errorf("expected nil response on cancelled ctx, got %+v", resp)
	}
}

// TestStub_ProviderAndModel — interface identity.
func TestStub_ProviderAndModel(t *testing.T) {
	c := newStubForTest(t)
	if c.Provider() != "stub" {
		t.Errorf("Provider() = %q, want stub", c.Provider())
	}
	if c.Model() != "stub-canned" {
		t.Errorf("Model() = %q, want stub-canned", c.Model())
	}
}
