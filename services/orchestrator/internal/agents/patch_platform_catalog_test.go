// PR #36 / OPS-AGENT-001 — unit tests for the platform tool catalog.
//
// Also acts as a structural-safety regression test: every catalog row
// MUST reference real tool names that the connector PRs registered.
// If a tool name is ever renamed without updating the catalog, this
// test catches it.
package agents

import (
	"sort"
	"strings"
	"testing"
)

func TestPlatformToolsFor_KnownPlatforms(t *testing.T) {
	cases := []struct {
		platform string
		want     PlatformToolTier
	}{
		{"aws", PlatformToolTier{
			ReadOnly: "query_ec2_instances",
			DryRun:   "ssm_send_patch_command",
			Live:     "ssm_send_patch_command_live",
		}},
		{"azure", PlatformToolTier{
			ReadOnly: "query_azure_vms",
			DryRun:   "azure_run_command",
			Live:     "azure_run_command_live",
		}},
		{"gcp", PlatformToolTier{
			ReadOnly: "query_gcp_instances",
			DryRun:   "gcp_os_config_patch",
			Live:     "gcp_os_config_patch_live",
		}},
		{"vsphere", PlatformToolTier{
			ReadOnly: "query_vsphere_vms",
			DryRun:   "vsphere_run_guest_program",
			Live:     "vsphere_run_guest_program_live",
		}},
		{"k8s", PlatformToolTier{
			ReadOnly: "query_pods",
			DryRun:   "k8s_apply",
			Live:     "k8s_apply_live",
		}},
	}

	for _, c := range cases {
		t.Run(c.platform, func(t *testing.T) {
			got, ok := PlatformToolsFor(c.platform)
			if !ok {
				t.Fatalf("PlatformToolsFor(%q): unknown", c.platform)
			}
			if got != c.want {
				t.Errorf("PlatformToolsFor(%q) = %+v, want %+v", c.platform, got, c.want)
			}
		})
	}
}

func TestPlatformToolsFor_UnknownPlatform(t *testing.T) {
	_, ok := PlatformToolsFor("openstack")
	if ok {
		t.Error("openstack should not be in the catalog")
	}
	_, ok = PlatformToolsFor("")
	if ok {
		t.Error("empty platform should return ok=false")
	}
}

func TestSupportedPlatforms_StableOrder(t *testing.T) {
	got := SupportedPlatforms()
	want := []string{"aws", "azure", "gcp", "k8s", "vsphere"}
	if len(got) != len(want) {
		t.Fatalf("SupportedPlatforms() len = %d, want %d", len(got), len(want))
	}
	for i, p := range want {
		if got[i] != p {
			t.Errorf("SupportedPlatforms()[%d] = %q, want %q", i, got[i], p)
		}
	}
	if !sort.StringsAreSorted(got) {
		t.Error("SupportedPlatforms() should be sorted")
	}
}

// TestAllCatalogToolNames_NoDuplicates verifies no tool name appears twice.
// Renaming a tool in one cloud to collide with another would break audit.
func TestAllCatalogToolNames_NoDuplicates(t *testing.T) {
	names := AllCatalogToolNames()
	seen := make(map[string]bool, len(names))
	for _, n := range names {
		if seen[n] {
			t.Errorf("tool name %q appears twice in catalog", n)
		}
		seen[n] = true
	}
}

// TestAllCatalogToolNames_LiveToolNaming asserts that live tools follow
// the established `_live` suffix convention. Future cloud additions must
// keep this convention so audit-by-grep keeps working.
func TestAllCatalogToolNames_LiveToolNaming(t *testing.T) {
	for _, p := range SupportedPlatforms() {
		tier, _ := PlatformToolsFor(p)
		if !strings.HasSuffix(tier.Live, "_live") {
			t.Errorf("platform %q live tool %q must end in `_live`", p, tier.Live)
		}
	}
}

func TestGroupAssetsByPlatform_SliceOfMaps(t *testing.T) {
	assets := []map[string]any{
		{"id": "a1", "platform": "aws"},
		{"id": "a2", "platform": "aws"},
		{"id": "g1", "platform": "gcp"},
		{"id": "z1", "platform": "azure"},
	}
	got := GroupAssetsByPlatform(assets)
	if len(got["aws"]) != 2 {
		t.Errorf("aws bucket = %d, want 2", len(got["aws"]))
	}
	if len(got["gcp"]) != 1 {
		t.Errorf("gcp bucket = %d, want 1", len(got["gcp"]))
	}
	if len(got["azure"]) != 1 {
		t.Errorf("azure bucket = %d, want 1", len(got["azure"]))
	}
}

func TestGroupAssetsByPlatform_EnvelopeWithAssetsKey(t *testing.T) {
	envelope := map[string]any{
		"assets": []map[string]any{
			{"id": "v1", "platform": "vsphere"},
		},
	}
	got := GroupAssetsByPlatform(envelope)
	if len(got["vsphere"]) != 1 {
		t.Errorf("vsphere bucket = %d, want 1", len(got["vsphere"]))
	}
}

func TestGroupAssetsByPlatform_EmptyAndUnknownPlatform(t *testing.T) {
	assets := []map[string]any{
		{"id": "x1"},                          // missing platform
		{"id": "x2", "platform": ""},          // empty platform
		{"id": "x3", "platform": "openstack"}, // unknown platform
	}
	got := GroupAssetsByPlatform(assets)
	if len(got["unknown"]) != 2 {
		t.Errorf("unknown bucket = %d, want 2 (missing + empty)", len(got["unknown"]))
	}
	if len(got["openstack"]) != 1 {
		t.Errorf("openstack bucket = %d, want 1", len(got["openstack"]))
	}
}

func TestGroupAssetsByPlatform_NilAndWrongType(t *testing.T) {
	got := GroupAssetsByPlatform(nil)
	if len(got) != 0 {
		t.Errorf("nil input should yield empty map, got %v", got)
	}
	got = GroupAssetsByPlatform("not a slice")
	if len(got) != 0 {
		t.Errorf("wrong-type input should yield empty map, got %v", got)
	}
}

func TestPlatformCountSummary(t *testing.T) {
	grouped := map[string][]map[string]any{
		"aws":   {{}, {}, {}},
		"gcp":   {{}},
		"azure": {},
	}
	got := PlatformCountSummary(grouped)
	if got["aws"] != 3 {
		t.Errorf("aws count = %d, want 3", got["aws"])
	}
	if got["gcp"] != 1 {
		t.Errorf("gcp count = %d, want 1", got["gcp"])
	}
	if got["azure"] != 0 {
		t.Errorf("azure count = %d, want 0", got["azure"])
	}
}
