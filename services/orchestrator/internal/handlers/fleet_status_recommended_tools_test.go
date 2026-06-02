// PR #44 / OPS-DEMO-001 — unit tests for extractRecommendedTools.
package handlers

import (
	"testing"
)

func TestExtractRecommendedTools_HappyPath(t *testing.T) {
	payload := []byte(`{
		"phases": [
			{"name":"canary","recommended_tools":{"aws":"ssm_send_patch_command","k8s":"k8s_apply"}},
			{"name":"monitor","recommended_tools":{}},
			{"name":"full_rollout","recommended_tools":{"aws":"ssm_send_patch_command","k8s":"k8s_apply"}}
		]
	}`)
	got := extractRecommendedTools(payload)
	if got["aws"] != "ssm_send_patch_command" {
		t.Errorf("aws = %q, want ssm_send_patch_command", got["aws"])
	}
	if got["k8s"] != "k8s_apply" {
		t.Errorf("k8s = %q, want k8s_apply", got["k8s"])
	}
}

func TestExtractRecommendedTools_SkipsEmptyPhases(t *testing.T) {
	payload := []byte(`{
		"phases": [
			{"name":"preflight","recommended_tools":{}},
			{"name":"canary","recommended_tools":{"azure":"azure_run_command"}}
		]
	}`)
	got := extractRecommendedTools(payload)
	if got["azure"] != "azure_run_command" {
		t.Errorf("should fall through preflight to canary; got azure=%q", got["azure"])
	}
}

func TestExtractRecommendedTools_NilWhenNoPhases(t *testing.T) {
	payload := []byte(`{"summary":"no phases here"}`)
	got := extractRecommendedTools(payload)
	if got != nil {
		t.Errorf("expected nil when no phases, got %v", got)
	}
}

func TestExtractRecommendedTools_NilWhenPhasesAreStrings(t *testing.T) {
	// Pre-PR-#43 plans had phases as string arrays. The extractor must
	// not crash on those — it should return nil for backward compat.
	payload := []byte(`{"phases":["canary","monitor","full_rollout"]}`)
	got := extractRecommendedTools(payload)
	if got != nil {
		t.Errorf("expected nil for string-phase plans, got %v", got)
	}
}

func TestExtractRecommendedTools_NilOnInvalidJSON(t *testing.T) {
	got := extractRecommendedTools([]byte(`not valid json`))
	if got != nil {
		t.Errorf("expected nil on invalid JSON, got %v", got)
	}
}

func TestExtractRecommendedTools_NilOnEmptyPayload(t *testing.T) {
	got := extractRecommendedTools(nil)
	if got != nil {
		t.Errorf("expected nil on nil payload, got %v", got)
	}
	got = extractRecommendedTools([]byte{})
	if got != nil {
		t.Errorf("expected nil on empty payload, got %v", got)
	}
}

func TestExtractRecommendedTools_SkipsNonStringValues(t *testing.T) {
	// If a value is somehow not a string (corrupted plan), the extractor
	// should silently skip it rather than panic.
	payload := []byte(`{
		"phases":[
			{"recommended_tools":{"aws":"ssm_send_patch_command","corrupted":123}}
		]
	}`)
	got := extractRecommendedTools(payload)
	if got["aws"] != "ssm_send_patch_command" {
		t.Errorf("aws should still be extracted; got %v", got)
	}
	if _, present := got["corrupted"]; present {
		t.Errorf("corrupted (non-string) should be skipped; got %v", got)
	}
}
