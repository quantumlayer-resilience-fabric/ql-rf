// PR #31 / CONN-011 — unit tests for GCPPatchJobLiveTool + live client.

package tools

import (
	"context"
	"strings"
	"testing"

	pkgconfig "github.com/quantumlayerhq/ql-rf/pkg/config"
)

func TestParseGCPLiveWhitelistCSV(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{",,,", nil},
		{"us-central1-a:labels.env=prod", []string{"us-central1-a:labels.env=prod"}},
		{" us-central1-a:labels.env=prod , us-east1-b:labels.env=stage ", []string{"us-central1-a:labels.env=prod", "us-east1-b:labels.env=stage"}},
	}
	for _, c := range cases {
		got := parseGCPLiveWhitelistCSV(c.in)
		if !stringSliceEq(got, c.want) {
			t.Errorf("parseGCPLiveWhitelistCSV(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestMockLiveGCPPatchClient_ValidatesAndReturnsJobName(t *testing.T) {
	wl := []string{"us-central1-a:labels.env=prod"}
	c := NewMockLiveGCPPatchClient(wl)
	plan := &GCPPatchJobPlan{
		ProjectID:      "ql-rf-prod",
		Zone:           "us-central1-a",
		InstanceFilter: "labels.env=prod",
	}
	name, err := c.SendPatchJob(context.Background(), plan)
	if err != nil {
		t.Fatalf("SendPatchJob: %v", err)
	}
	if !strings.HasPrefix(name, "projects/ql-rf-mock-project/patchJobs/mock-") {
		t.Errorf("job name = %q, want projects/ql-rf-mock-project/patchJobs/mock-* prefix", name)
	}
}

func TestMockLiveGCPPatchClient_RejectsNonWhitelistedTarget(t *testing.T) {
	c := NewMockLiveGCPPatchClient([]string{"us-central1-a:labels.env=prod"})
	plan := &GCPPatchJobPlan{
		Zone:           "us-east1-b",
		InstanceFilter: "labels.env=stage",
	}
	if _, err := c.SendPatchJob(context.Background(), plan); err == nil {
		t.Fatal("expected whitelist rejection, got nil")
	}
}

func TestMockLiveGCPPatchClient_RejectsNilPlan(t *testing.T) {
	c := NewMockLiveGCPPatchClient([]string{"us-central1-a:labels.env=prod"})
	if _, err := c.SendPatchJob(context.Background(), nil); err == nil {
		t.Fatal("expected nil-plan rejection, got nil")
	}
}

func TestNewLiveGCPPatchClient_RefusesEmptyWhitelist(t *testing.T) {
	_, err := NewLiveGCPPatchClient(context.Background(), pkgconfig.GCPConfig{}, nil, testLoggerForSSM())
	if err == nil {
		t.Fatal("expected refusal for empty whitelist, got nil")
	}
	if !strings.Contains(err.Error(), "whitelist") {
		t.Errorf("error should mention whitelist; got %q", err.Error())
	}
}

func TestNewLiveGCPPatchClient_RefusesMissingProjectID(t *testing.T) {
	_, err := NewLiveGCPPatchClient(context.Background(), pkgconfig.GCPConfig{}, []string{"us-central1-a:labels.env=prod"}, testLoggerForSSM())
	if err == nil {
		t.Fatal("expected refusal for missing ProjectID, got nil")
	}
	if !strings.Contains(err.Error(), "ProjectID is required") {
		t.Errorf("error should mention ProjectID; got %q", err.Error())
	}
}

func TestGCPPatchJobLiveTool_FiresMockAndReturnsLiveEnvelope(t *testing.T) {
	whitelist := []string{"us-central1-a:labels.env=prod"}
	tool := NewGCPPatchJobLiveTool(
		NewRealGCPPatchClient(testLoggerForSSM()),
		NewMockLiveGCPPatchClient(whitelist),
	)

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
	if envelope["dry_run"] != false {
		t.Errorf("dry_run = %v, want false", envelope["dry_run"])
	}
	if envelope["real_changes"] != true {
		t.Errorf("real_changes = %v, want true", envelope["real_changes"])
	}
	name, ok := envelope["patch_job_name"].(string)
	if !ok || !strings.Contains(name, "patchJobs/mock-") {
		t.Errorf("patch_job_name = %v, want resource path containing patchJobs/mock-", envelope["patch_job_name"])
	}
}

func TestGCPPatchJobLiveTool_RejectsNonWhitelistedTarget(t *testing.T) {
	tool := NewGCPPatchJobLiveTool(
		NewRealGCPPatchClient(testLoggerForSSM()),
		NewMockLiveGCPPatchClient([]string{"us-central1-a:labels.env=prod"}),
	)

	_, err := tool.Execute(context.Background(), map[string]any{
		"project_id":      "ql-rf-prod-001",
		"zone":            "us-east1-b", // valid format, not on whitelist
		"instance_filter": "labels.env=stage",
	})
	if err == nil {
		t.Fatal("expected whitelist rejection, got nil")
	}
}

func TestGCPPatchJobLiveTool_RisksAndApprovalShape(t *testing.T) {
	tool := NewGCPPatchJobLiveTool(
		NewRealGCPPatchClient(testLoggerForSSM()),
		NewMockLiveGCPPatchClient([]string{"us-central1-a:labels.env=prod"}),
	)
	if tool.Risk() != RiskStateChangeProd {
		t.Errorf("Risk = %v, want %v", tool.Risk(), RiskStateChangeProd)
	}
	if !tool.RequiresApproval() {
		t.Error("RequiresApproval should be true")
	}
	if tool.Idempotent() {
		t.Error("Idempotent should be false")
	}
	if tool.Name() != "gcp_os_config_patch_live" {
		t.Errorf("Name = %q, want gcp_os_config_patch_live", tool.Name())
	}
}

func TestRegisterGCPLivePatchTools_NoOpOnNilClient(t *testing.T) {
	r := NewRegistry(nil, testLoggerForSSM())
	r.RegisterGCPLivePatchTools(nil, NewMockLiveGCPPatchClient([]string{"us-central1-a:labels.env=prod"}))
	if _, ok := r.Get("gcp_os_config_patch_live"); ok {
		t.Error("should not register with nil dry client")
	}
	r.RegisterGCPLivePatchTools(NewRealGCPPatchClient(testLoggerForSSM()), nil)
	if _, ok := r.Get("gcp_os_config_patch_live"); ok {
		t.Error("should not register with nil live client")
	}
}
