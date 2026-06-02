// PR #30 / CONN-010 — unit tests for GCPPatchJobDryRunTool + helpers.

package tools

import (
	"context"
	"strings"
	"testing"
)

func newGCPPatchRealForTest(t *testing.T) GCPPatchClient {
	t.Helper()
	return NewRealGCPPatchClient(testLoggerForSSM())
}

func TestGCPPatch_BuildsPlan(t *testing.T) {
	tool := NewGCPPatchJobDryRunTool(newGCPPatchRealForTest(t))

	got, err := tool.Execute(context.Background(), map[string]any{
		"project_id":      "ql-rf-prod-001",
		"zone":            "us-central1-a",
		"instance_filter": "labels.env=prod",
		"reboot_config":   "ALWAYS",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	envelope, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("got = %T, want map[string]any", got)
	}
	if envelope["dry_run"] != true {
		t.Errorf("dry_run = %v, want true", envelope["dry_run"])
	}
	plan, ok := envelope["patch_plan"].(*GCPPatchJobPlan)
	if !ok {
		t.Fatalf("patch_plan = %T, want *GCPPatchJobPlan", envelope["patch_plan"])
	}
	if plan.RebootConfig != "ALWAYS" {
		t.Errorf("RebootConfig = %q, want ALWAYS", plan.RebootConfig)
	}
}

func TestGCPPatch_DefaultsToRebootDefault(t *testing.T) {
	tool := NewGCPPatchJobDryRunTool(newGCPPatchRealForTest(t))

	got, err := tool.Execute(context.Background(), map[string]any{
		"project_id":      "ql-rf-prod-001",
		"zone":            "us-central1-a",
		"instance_filter": "labels.env=prod",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	envelope, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("got = %T, want map[string]any", got)
	}
	plan, ok := envelope["patch_plan"].(*GCPPatchJobPlan)
	if !ok {
		t.Fatalf("patch_plan = %T, want *GCPPatchJobPlan", envelope["patch_plan"])
	}
	if plan.RebootConfig != "DEFAULT" {
		t.Errorf("default RebootConfig = %q, want DEFAULT", plan.RebootConfig)
	}
}

func TestGCPPatch_RejectsBadProjectID(t *testing.T) {
	tool := NewGCPPatchJobDryRunTool(newGCPPatchRealForTest(t))

	_, err := tool.Execute(context.Background(), map[string]any{
		"project_id":      "Bad_ProjectID", // intentionally invalid (uppercase plus underscore)
		"zone":            "us-central1-a",
		"instance_filter": "labels.env=prod",
	})
	if err == nil {
		t.Fatal("expected error for invalid project_id, got nil")
	}
	if !strings.Contains(err.Error(), "invalid project_id") {
		t.Errorf("error = %q, want substring 'invalid project_id'", err.Error())
	}
}

func TestGCPPatch_RejectsBadZone(t *testing.T) {
	tool := NewGCPPatchJobDryRunTool(newGCPPatchRealForTest(t))

	_, err := tool.Execute(context.Background(), map[string]any{
		"project_id":      "ql-rf-prod-001",
		"zone":            "not-a-zone",
		"instance_filter": "labels.env=prod",
	})
	if err == nil {
		t.Fatal("expected error for invalid zone, got nil")
	}
	if !strings.Contains(err.Error(), "invalid zone") {
		t.Errorf("error = %q, want substring 'invalid zone'", err.Error())
	}
}

func TestGCPPatch_RejectsBadRebootConfig(t *testing.T) {
	tool := NewGCPPatchJobDryRunTool(newGCPPatchRealForTest(t))

	_, err := tool.Execute(context.Background(), map[string]any{
		"project_id":      "ql-rf-prod-001",
		"zone":            "us-central1-a",
		"instance_filter": "labels.env=prod",
		"reboot_config":   "MAYBE",
	})
	if err == nil {
		t.Fatal("expected error for invalid reboot_config, got nil")
	}
	if !strings.Contains(err.Error(), "reboot_config must be") {
		t.Errorf("error = %q, want substring 'reboot_config must be'", err.Error())
	}
}

func TestGCPPatch_RejectsEmptyFilter(t *testing.T) {
	tool := NewGCPPatchJobDryRunTool(newGCPPatchRealForTest(t))

	_, err := tool.Execute(context.Background(), map[string]any{
		"project_id":      "ql-rf-prod-001",
		"zone":            "us-central1-a",
		"instance_filter": "",
	})
	if err == nil {
		t.Fatal("expected error for empty instance_filter, got nil")
	}
}

func TestMockGCPPatchClient(t *testing.T) {
	c := NewMockGCPPatchClient()
	plan, err := c.BuildPatchJobPlan(context.Background(), GCPPatchJobRequest{
		ProjectID:      "garbage",
		Zone:           "garbage-zone",
		InstanceFilter: "garbage",
	})
	if err != nil {
		t.Fatalf("BuildPatchJobPlan: %v", err)
	}
	if plan.ProjectID != "ql-rf-mock-project" {
		t.Errorf("mock returned non-mock project_id %q", plan.ProjectID)
	}
	if !plan.DryRun {
		t.Errorf("DryRun = false; mock client must always produce dry-run plans")
	}
}

func TestRegisterGCPPatchDryRunTools_NoOpOnNil(t *testing.T) {
	r := NewRegistry(nil, testLoggerForSSM())
	r.RegisterGCPPatchDryRunTools(nil)
	if _, ok := r.Get("gcp_os_config_patch"); ok {
		t.Error("should not register with nil client")
	}
}
