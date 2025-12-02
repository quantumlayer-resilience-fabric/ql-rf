# QuantumLayer Resilience Fabric - Tool Authorization Policies
# Package: ql.ai.tools
# Purpose: Control tool access based on risk level and autonomy mode
# Reference: ADR-009

package ql.ai.tools

import future.keywords.in

# Default deny tool execution
default allow := false

# =============================================================================
# Tool Execution Allow Rules
# =============================================================================

# Allow read-only tools always
allow {
    input.tool.risk == "read_only"
}

# Allow plan-only tools always
allow {
    input.tool.risk == "plan_only"
}

# Allow state-changing non-prod tools based on autonomy mode
allow {
    input.tool.risk == "state_change_nonprod"
    input.autonomy.mode in ["canary_only", "full_auto"]
}

# Allow state-changing prod tools only with approval
allow {
    input.tool.risk == "state_change_prod"
    input.approval.status == "approved"
}

# =============================================================================
# Tool Denial Rules
# =============================================================================

# Deny state-changing non-prod in plan_only mode
deny[msg] {
    input.tool.risk == "state_change_nonprod"
    input.autonomy.mode == "plan_only"
    msg := sprintf("Tool '%s' blocked: organization is in plan_only mode", [input.tool.name])
}

# Deny state-changing prod without approval
deny[msg] {
    input.tool.risk == "state_change_prod"
    not input.approval.status == "approved"
    msg := sprintf("Tool '%s' requires approval for production state changes", [input.tool.name])
}

# Deny tools that require simulation without prior simulation
deny[msg] {
    input.tool.requires_simulation
    not input.simulation_completed
    msg := sprintf("Tool '%s' requires simulation to be run first", [input.tool.name])
}

# Deny tools exceeding scope
deny[msg] {
    input.tool.scope == "organization"
    not input.user.can_modify_org
    msg := sprintf("Tool '%s' requires organization-level permissions", [input.tool.name])
}

# Deny tools when token budget exceeded
deny[msg] {
    input.org.tokens_used_this_month >= input.org.monthly_token_budget
    input.tool.risk != "read_only"
    msg := "Monthly token budget exceeded - only read-only tools available"
}

# =============================================================================
# Tool Risk Classification
# =============================================================================

tool_risk_level[risk] {
    risk := data.tool_registry[input.tool.name].risk
}

# Read-only tools
read_only_tools := {
    "query_assets",
    "get_drift_status",
    "get_compliance_status",
    "get_golden_image",
    "query_alerts",
    "get_dr_status"
}

# Plan-only tools
plan_only_tools := {
    "compare_versions",
    "generate_patch_plan",
    "generate_rollout_plan",
    "generate_dr_runbook",
    "generate_compliance_evidence",
    "simulate_rollout",
    "simulate_failover",
    "calculate_risk_score"
}

# State-changing production tools
state_change_prod_tools := {
    "execute_rollout_prod",
    "acknowledge_alert",
    "trigger_dr_failover",
    "modify_firewall_rule"
}

# =============================================================================
# Audit Requirements
# =============================================================================

# Require full audit for state-changing tools
requires_full_audit {
    input.tool.risk in ["state_change_nonprod", "state_change_prod"]
}

# Require evidence for compliance-related tools
requires_evidence {
    input.tool.name in ["generate_compliance_evidence", "execute_rollout_prod"]
    input.context.compliance_scope
}
