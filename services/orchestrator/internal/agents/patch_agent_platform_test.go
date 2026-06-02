// PR #36 / OPS-AGENT-001 — patch-agent platform-aware tests.
//
// Verifies that:
//  1. The default fallback plan carries `recommended_tools` per phase
//     and the rollback names the right tool per platform.
//  2. The prompt-catalog renderer produces the expected block.
//  3. Production environment selects dry-run tools (live execution is
//     the two-approver workflow's responsibility, NOT the planner's).
//  4. Non-production environments select live tools.
//  5. The platform catalog correctly populates the agent's tools list.
package agents

import (
	"strings"
	"testing"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

func TestPlatformRecommendedTools_ProductionPicksDryRun(t *testing.T) {
	counts := map[string]int{"aws": 5, "gcp": 2}
	got := platformRecommendedTools(counts, true /* useDryRun */)
	if got["aws"] != "ssm_send_patch_command" {
		t.Errorf("aws prod = %q, want ssm_send_patch_command", got["aws"])
	}
	if got["gcp"] != "gcp_os_config_patch" {
		t.Errorf("gcp prod = %q, want gcp_os_config_patch", got["gcp"])
	}
	for p, tool := range got {
		if strings.HasSuffix(tool, "_live") {
			t.Errorf("production should NEVER recommend a live tool, but %s → %s", p, tool)
		}
	}
}

func TestPlatformRecommendedTools_NonProductionPicksLive(t *testing.T) {
	counts := map[string]int{"azure": 3, "vsphere": 1}
	got := platformRecommendedTools(counts, false /* useDryRun */)
	if got["azure"] != "azure_run_command_live" {
		t.Errorf("azure non-prod = %q, want azure_run_command_live", got["azure"])
	}
	if got["vsphere"] != "vsphere_run_guest_program_live" {
		t.Errorf("vsphere non-prod = %q, want vsphere_run_guest_program_live", got["vsphere"])
	}
}

func TestPlatformRecommendedTools_SkipsUnknownAndZeroCount(t *testing.T) {
	counts := map[string]int{"aws": 5, "openstack": 2, "azure": 0}
	got := platformRecommendedTools(counts, false)
	if _, ok := got["openstack"]; ok {
		t.Error("unknown platform openstack must be skipped")
	}
	if _, ok := got["azure"]; ok {
		t.Error("zero-count platform azure must be skipped")
	}
	if got["aws"] == "" {
		t.Error("aws should still be present")
	}
}

func TestPlatformAwareRollbackProcedure_SinglePlatform(t *testing.T) {
	got := platformAwareRollbackProcedure(map[string]int{"aws": 12})
	if !strings.Contains(got, "aws (12 assets)") {
		t.Errorf("procedure missing aws breakdown: %q", got)
	}
	if !strings.Contains(got, "ssm_send_patch_command") {
		t.Errorf("procedure missing dry-run tool name: %q", got)
	}
	if !strings.Contains(got, "ssm_send_patch_command_live") {
		t.Errorf("procedure missing live tool name: %q", got)
	}
	if !strings.Contains(got, "2-approver") {
		t.Errorf("procedure should mention 2-approver gate: %q", got)
	}
}

func TestPlatformAwareRollbackProcedure_MixedPlatforms(t *testing.T) {
	got := platformAwareRollbackProcedure(map[string]int{
		"aws":   5,
		"gcp":   3,
		"azure": 2,
	})
	for _, want := range []string{"aws", "gcp", "azure",
		"ssm_send_patch_command", "gcp_os_config_patch", "azure_run_command"} {
		if !strings.Contains(got, want) {
			t.Errorf("mixed-platform procedure missing %q: %s", want, got)
		}
	}
}

func TestPlatformAwareRollbackProcedure_OnlyUnknownPlatforms(t *testing.T) {
	got := platformAwareRollbackProcedure(map[string]int{"openstack": 4})
	if !strings.Contains(got, "Manual or image-based") {
		t.Errorf("only-unknown should emit manual-rollback marker: %q", got)
	}
}

func TestPlatformAwareRollbackProcedure_Empty(t *testing.T) {
	got := platformAwareRollbackProcedure(map[string]int{})
	if got == "" {
		t.Error("empty platform counts should still return a default procedure string")
	}
}

func TestRenderPlatformCatalogForPrompt_StableOrder(t *testing.T) {
	counts := map[string]int{
		"vsphere": 1,
		"aws":     5,
		"gcp":     2,
	}
	got := renderPlatformCatalogForPrompt(counts)
	// aws must appear before gcp must appear before vsphere (alphabetical).
	awsIdx := strings.Index(got, "- aws (")
	gcpIdx := strings.Index(got, "- gcp (")
	vsphereIdx := strings.Index(got, "- vsphere (")
	if awsIdx < 0 || gcpIdx < 0 || vsphereIdx < 0 {
		t.Fatalf("missing platform line(s) in:\n%s", got)
	}
	if !(awsIdx < gcpIdx && gcpIdx < vsphereIdx) {
		t.Errorf("expected alphabetical order aws < gcp < vsphere; got idx %d, %d, %d",
			awsIdx, gcpIdx, vsphereIdx)
	}
}

func TestRenderPlatformCatalogForPrompt_EmptyInput(t *testing.T) {
	got := renderPlatformCatalogForPrompt(map[string]int{})
	if got != "(no assets in scope)" {
		t.Errorf("empty input should return marker; got %q", got)
	}
}

func TestRenderPlatformCatalogForPrompt_UnknownPlatform(t *testing.T) {
	got := renderPlatformCatalogForPrompt(map[string]int{"openstack": 3})
	if !strings.Contains(got, "openstack") {
		t.Errorf("unknown platform should still appear: %q", got)
	}
	if !strings.Contains(got, "NO PLATFORM-SPECIFIC TOOL") {
		t.Errorf("unknown platform should carry the no-tool marker: %q", got)
	}
}

func TestDefaultPatchPlan_ProductionAttachesDryRunTools(t *testing.T) {
	log := logger.New("debug", "json")
	agent := &PatchAgent{BaseAgent: BaseAgent{name: "patch_agent", log: log}}

	plan := agent.defaultPatchPlan(20, "medium", 5, 25,
		map[string]int{"aws": 15, "gcp": 5}, "production")

	phases, ok := plan["phases"].([]map[string]any)
	if !ok {
		t.Fatalf("phases type = %T, want []map[string]any", plan["phases"])
	}

	// Find the canary phase
	var canary map[string]any
	for _, p := range phases {
		if p["type"] == "canary" {
			canary = p
			break
		}
	}
	if canary == nil {
		t.Fatal("canary phase not found")
	}

	tools, ok := canary["recommended_tools"].(map[string]string)
	if !ok {
		t.Fatalf("canary.recommended_tools type = %T, want map[string]string", canary["recommended_tools"])
	}
	if tools["aws"] != "ssm_send_patch_command" {
		t.Errorf("production canary aws = %q, want ssm_send_patch_command (dry-run)", tools["aws"])
	}
	if strings.HasSuffix(tools["aws"], "_live") {
		t.Error("production canary must NOT name the live tool")
	}
}

func TestDefaultPatchPlan_NonProductionAttachesLiveTools(t *testing.T) {
	log := logger.New("debug", "json")
	agent := &PatchAgent{BaseAgent: BaseAgent{name: "patch_agent", log: log}}

	plan := agent.defaultPatchPlan(20, "medium", 5, 25,
		map[string]int{"vsphere": 20}, "staging")

	phases, ok := plan["phases"].([]map[string]any)
	if !ok {
		t.Fatalf("phases type = %T", plan["phases"])
	}
	var canary map[string]any
	for _, p := range phases {
		if p["type"] == "canary" {
			canary = p
			break
		}
	}
	tools, _ := canary["recommended_tools"].(map[string]string)
	if tools["vsphere"] != "vsphere_run_guest_program_live" {
		t.Errorf("staging canary vsphere = %q, want vsphere_run_guest_program_live", tools["vsphere"])
	}
}

func TestDefaultPatchPlan_RollbackProcedureNamesPlatformTools(t *testing.T) {
	log := logger.New("debug", "json")
	agent := &PatchAgent{BaseAgent: BaseAgent{name: "patch_agent", log: log}}

	plan := agent.defaultPatchPlan(10, "low", 10, 50,
		map[string]int{"aws": 10}, "production")

	rollback, ok := plan["rollback_plan"].(map[string]any)
	if !ok {
		t.Fatalf("rollback_plan type = %T", plan["rollback_plan"])
	}
	procedure, _ := rollback["procedure"].(string)
	if !strings.Contains(procedure, "ssm_send_patch_command") {
		t.Errorf("rollback procedure missing aws-specific tool: %q", procedure)
	}
}

func TestNewPatchAgent_ToolsListIncludesCatalog(t *testing.T) {
	log := logger.New("debug", "json")
	agent := NewPatchAgent(nil /* llm */, nil /* registry */, log)

	gotTools := agent.tools

	// Must include every catalog tool
	for _, want := range AllCatalogToolNames() {
		found := false
		for _, t := range gotTools {
			if t == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("patch-agent tools missing catalog entry %q", want)
		}
	}

	// Must still include the original base tools
	for _, want := range []string{
		"query_assets", "get_golden_image", "simulate_rollout", "calculate_risk_score",
	} {
		found := false
		for _, t := range gotTools {
			if t == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("patch-agent tools missing base tool %q (regression)", want)
		}
	}
}
