// PR #36 / OPS-AGENT-001 — platform-aware tool catalog for the patch agent.
//
// The full connector arc shipped 4 platforms × 3 risk tiers = 12 cloud tools.
// Before PR #36, the patch agent's prompt only knew about "SSM" by name and
// had no way to recommend the right tool per asset platform. This file is
// the canonical data table mapping platform → (read-only, dry-run, live)
// tool names.
//
// SAFETY: this is data only. The catalog is consumed by the LLM prompt and
// by per-platform grouping helpers. The agent's Execute method does NOT
// invoke any of the state-change tools listed here — they are only NAMED
// in the generated plan, and execution gates remain at the existing
// state_change_prod approval + 2-approver workflow boundaries.
package agents

import (
	"fmt"
	"sort"
	"strings"
)

// PlatformToolTier names the three risk tiers for a given platform.
type PlatformToolTier struct {
	// ReadOnly is the inventory / discovery tool. Risk=read_only.
	ReadOnly string
	// DryRun is the state-change-shaped tool that REFUSES to actually
	// fire the SDK; it returns the would-be plan only. Risk=state_change_test.
	DryRun string
	// Live is the live state-change tool gated by the 4-gate boot logic
	// (env opt-in + mock-conflict refusal + whitelist + 2-approver).
	// Risk=state_change_prod, RequiresApproval=true.
	Live string
}

// platformCatalog maps platform identifier to its tool tier.
// Keys MUST match the strings returned by QueryAssetsTool's `platform` column
// (aws | azure | gcp | vsphere).
//
// This is the single source of truth — when a new cloud's tools land
// (e.g. K8s arc in PRs #38-#40), add the row here.
var platformCatalog = map[string]PlatformToolTier{
	"aws": {
		ReadOnly: "query_ec2_instances",
		DryRun:   "ssm_send_patch_command",
		Live:     "ssm_send_patch_command_live",
	},
	"azure": {
		ReadOnly: "query_azure_vms",
		DryRun:   "azure_run_command",
		Live:     "azure_run_command_live",
	},
	"gcp": {
		ReadOnly: "query_gcp_instances",
		DryRun:   "gcp_os_config_patch",
		Live:     "gcp_os_config_patch_live",
	},
	"vsphere": {
		ReadOnly: "query_vsphere_vms",
		DryRun:   "vsphere_run_guest_program",
		Live:     "vsphere_run_guest_program_live",
	},
}

// PlatformToolsFor returns the tool-tier names for a given platform.
// The second return value is false when the platform is unknown — callers
// should treat that as "no platform-specific tooling available; fall back
// to generic image-based patch flow".
func PlatformToolsFor(platform string) (PlatformToolTier, bool) {
	t, ok := platformCatalog[platform]
	return t, ok
}

// SupportedPlatforms returns the platform identifiers known to the catalog,
// sorted alphabetically for stable prompt rendering.
func SupportedPlatforms() []string {
	out := make([]string, 0, len(platformCatalog))
	for k := range platformCatalog {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// AllCatalogToolNames returns every tool name the catalog references,
// in stable order. Used by NewPatchAgent's `tools` metadata field and by
// the LLM prompt's "available tools" section.
func AllCatalogToolNames() []string {
	platforms := SupportedPlatforms()
	out := make([]string, 0, len(platforms)*3)
	for _, p := range platforms {
		t := platformCatalog[p]
		out = append(out, t.ReadOnly, t.DryRun, t.Live)
	}
	return out
}

// GroupAssetsByPlatform partitions a query_assets response (a slice of maps)
// into platform → assets buckets. Unknown / empty platforms go to the
// "unknown" bucket. The returned map's keys are stable iteration order
// when iterated via SupportedPlatforms() (plus "unknown" last).
func GroupAssetsByPlatform(assets any) map[string][]map[string]any {
	out := map[string][]map[string]any{}
	rows, ok := assets.([]map[string]any)
	if !ok {
		// Try the other common envelope: {"assets": [...]}.
		if env, ok := assets.(map[string]any); ok {
			switch inner := env["assets"].(type) {
			case []map[string]any:
				rows = inner
			case []any:
				for _, r := range inner {
					if m, ok := r.(map[string]any); ok {
						rows = append(rows, m)
					}
				}
			}
		}
	}
	if rows == nil {
		return out
	}
	for _, row := range rows {
		platform, _ := row["platform"].(string)
		if platform == "" {
			platform = "unknown"
		}
		out[platform] = append(out[platform], row)
	}
	return out
}

// PlatformCountSummary turns the platform→assets grouping into a
// platform→count map for the LLM prompt + the AgentResult evidence.
func PlatformCountSummary(grouped map[string][]map[string]any) map[string]int {
	out := make(map[string]int, len(grouped))
	for p, rows := range grouped {
		out[p] = len(rows)
	}
	return out
}

// platformRecommendedTools returns a platform → tool-name map naming
// the dry-run tool for each platform when useDryRun is true, the live
// tool otherwise. Used by both phase metadata and rollback metadata
// so the audit trail records the agent's per-platform recommendation.
//
// Unknown platforms are skipped — the catalog has nothing safe to
// recommend, and the LLM prompt's "no platform-specific tool" hint
// covers them.
func platformRecommendedTools(platformCounts map[string]int, useDryRun bool) map[string]string {
	out := map[string]string{}
	for p, count := range platformCounts {
		if count == 0 {
			continue
		}
		tier, ok := PlatformToolsFor(p)
		if !ok {
			continue
		}
		if useDryRun {
			out[p] = tier.DryRun
		} else {
			out[p] = tier.Live
		}
	}
	return out
}

// platformAwareRollbackProcedure builds a human-readable rollback
// procedure string that names the right tool per platform.
//
// Single-platform fleets get a tight one-liner; mixed fleets get a
// per-platform breakdown. Unknown platforms are folded into a "manual
// or image-based rollback" footnote.
func platformAwareRollbackProcedure(platformCounts map[string]int) string {
	if len(platformCounts) == 0 {
		return "Revert patched assets to previous image version using platform-specific rollback"
	}
	knownLines := make([]string, 0, len(platformCounts))
	unknownPlatforms := make([]string, 0)
	for _, p := range SupportedPlatforms() {
		count, ok := platformCounts[p]
		if !ok || count == 0 {
			continue
		}
		tier := platformCatalog[p]
		knownLines = append(knownLines, fmt.Sprintf(
			"%s (%d assets) → %s (dry-run) or %s (live, requires 2-approver)",
			p, count, tier.DryRun, tier.Live,
		))
	}
	for p, count := range platformCounts {
		if _, ok := platformCatalog[p]; ok || count == 0 {
			continue
		}
		unknownPlatforms = append(unknownPlatforms, fmt.Sprintf("%s (%d assets)", p, count))
	}
	if len(knownLines) == 0 && len(unknownPlatforms) > 0 {
		return fmt.Sprintf("Manual or image-based rollback required for: %s",
			strings.Join(unknownPlatforms, ", "))
	}
	procedure := "Revert patched assets via: " + strings.Join(knownLines, "; ")
	if len(unknownPlatforms) > 0 {
		procedure += ". Manual rollback for: " + strings.Join(unknownPlatforms, ", ")
	}
	return procedure
}

// renderPlatformCatalogForPrompt produces a Markdown block listing the
// available tool tiers for ONLY the platforms present in this rollout.
// Keeps the LLM prompt tight (no irrelevant platforms) and stable.
//
// Format:
//
//   - aws (12 assets): read-only=query_ec2_instances, dry-run=ssm_send_patch_command, live=ssm_send_patch_command_live
//   - gcp (3 assets):  read-only=query_gcp_instances, dry-run=gcp_os_config_patch, live=gcp_os_config_patch_live
//
// When platformCounts is empty (no assets, or no known platforms), returns
// an explicit "no platform-specific tooling available" marker so the LLM
// falls back to generic image-based phrasing.
func renderPlatformCatalogForPrompt(platformCounts map[string]int) string {
	if len(platformCounts) == 0 {
		return "(no assets in scope)"
	}
	// Stable order: iterate SupportedPlatforms first, then any unknown.
	lines := make([]string, 0, len(platformCounts))
	for _, p := range SupportedPlatforms() {
		count, ok := platformCounts[p]
		if !ok || count == 0 {
			continue
		}
		tier, _ := PlatformToolsFor(p)
		lines = append(lines, fmt.Sprintf(
			"- %s (%d assets): read-only=%s, dry-run=%s, live=%s",
			p, count, tier.ReadOnly, tier.DryRun, tier.Live,
		))
	}
	// Catch any unknown platforms so the user can see them in the plan.
	for p, count := range platformCounts {
		if _, known := PlatformToolsFor(p); known || count == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf(
			"- %s (%d assets): NO PLATFORM-SPECIFIC TOOL — recommend manual or image-based patch",
			p, count,
		))
	}
	if len(lines) == 0 {
		return "(no platforms with known tooling)"
	}
	return strings.Join(lines, "\n")
}
