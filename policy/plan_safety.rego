# QuantumLayer Resilience Fabric - Plan Safety Policies
# Package: ql.ai.safety
# Purpose: Validate AI-generated plans before execution
# Reference: ADR-007, ADR-008, ADR-009

package ql.ai.safety

import future.keywords.in
import future.keywords.every

# Default deny - all plans must explicitly pass validation
default allow := false

# =============================================================================
# Main allow rule - plan is allowed if no deny rules fire
# =============================================================================

allow {
    count(deny) == 0
}

# =============================================================================
# Production Safety Rules
# =============================================================================

# Deny production changes without canary phase
deny[msg] {
    input.plan.environment == "production"
    not has_canary_phase
    msg := "SAFETY: Production changes require a canary phase"
}

# Deny batch size > 20% for production
deny[msg] {
    input.plan.environment == "production"
    some phase in input.plan.phases
    phase_batch_percent := count(phase.assets) * 100 / input.plan.total_assets
    phase_batch_percent > 20
    msg := sprintf("SAFETY: Phase '%s' batch size (%d%%) exceeds 20%% limit for production", [phase.name, phase_batch_percent])
}

# Deny production changes without explicit rollback criteria
deny[msg] {
    input.plan.environment == "production"
    some phase in input.plan.phases
    not phase.rollback_if
    msg := sprintf("SAFETY: Phase '%s' must have rollback criteria for production", [phase.name])
}

# Deny production changes without health checks
deny[msg] {
    input.plan.environment == "production"
    some phase in input.plan.phases
    not phase.health_checks
    msg := sprintf("SAFETY: Phase '%s' must have health checks for production", [phase.name])
}

deny[msg] {
    input.plan.environment == "production"
    some phase in input.plan.phases
    count(phase.health_checks) == 0
    msg := sprintf("SAFETY: Phase '%s' must have at least one health check for production", [phase.name])
}

# =============================================================================
# General Safety Rules
# =============================================================================

# Deny plans with no phases
deny[msg] {
    not input.plan.phases
    msg := "SAFETY: Plan must have at least one phase"
}

deny[msg] {
    count(input.plan.phases) == 0
    msg := "SAFETY: Plan must have at least one phase"
}

# Deny phases with no assets
deny[msg] {
    some phase in input.plan.phases
    count(phase.assets) == 0
    msg := sprintf("SAFETY: Phase '%s' must have at least one asset", [phase.name])
}

# Deny plans without summary
deny[msg] {
    not input.plan.summary
    msg := "SAFETY: Plan must have a summary"
}

deny[msg] {
    input.plan.summary == ""
    msg := "SAFETY: Plan summary cannot be empty"
}

# =============================================================================
# Canary Phase Validation
# =============================================================================

# Canary phase should be small (max 10% or 5 assets)
deny[msg] {
    has_canary_phase
    canary := canary_phase
    canary_size := count(canary.assets)
    canary_percent := canary_size * 100 / input.plan.total_assets
    canary_percent > 10
    canary_size > 5
    msg := sprintf("SAFETY: Canary phase too large (%d assets, %d%%). Max 10%% or 5 assets", [canary_size, canary_percent])
}

# Canary phase should have wait time
deny[msg] {
    input.plan.environment == "production"
    has_canary_phase
    canary := canary_phase
    not canary.wait_time
    msg := "SAFETY: Canary phase must have wait_time for production"
}

# =============================================================================
# Tool Authorization Rules
# =============================================================================

# Deny state-changing tools without approved plan
deny[msg] {
    input.tool.risk == "state_change_prod"
    not input.plan.approved
    msg := sprintf("SAFETY: Tool '%s' requires approved plan for production state changes", [input.tool.name])
}

# Deny execution tools without simulation first
deny[msg] {
    input.tool.name == "execute_rollout_prod"
    not input.simulation_completed
    msg := "SAFETY: Must run simulate_rollout before execute_rollout_prod"
}

# =============================================================================
# HITL (Human-in-the-Loop) Rules
# =============================================================================

# Require HITL approval for production
deny[msg] {
    input.plan.environment == "production"
    not input.plan.hitl_approved
    input.autonomy.mode != "full_auto"
    msg := "SAFETY: Production plans require human approval"
}

# Require two approvers for critical operations
deny[msg] {
    input.plan.risk_level == "critical"
    input.autonomy.require_two_approvers
    count(input.approvals) < 2
    msg := "SAFETY: Critical operations require two approvers"
}

# Validate approver role
deny[msg] {
    input.plan.environment == "production"
    input.approver.role
    not approver_role_allowed
    msg := sprintf("SAFETY: User role '%s' not authorized to approve production plans", [input.approver.role])
}

# =============================================================================
# Time Window Rules
# =============================================================================

# Deny production changes outside maintenance window (if configured)
deny[msg] {
    input.plan.environment == "production"
    input.constraints.maintenance_window
    not within_maintenance_window
    msg := "SAFETY: Production changes must occur within maintenance window"
}

# =============================================================================
# Helper Rules
# =============================================================================

has_canary_phase {
    some phase in input.plan.phases
    lower(phase.name) == "canary"
}

canary_phase := phase {
    some phase in input.plan.phases
    lower(phase.name) == "canary"
}

approver_role_allowed {
    "any" in input.autonomy.allowed_approver_roles
}

approver_role_allowed {
    input.approver.role in input.autonomy.allowed_approver_roles
}

within_maintenance_window {
    # Simplified check - in practice would compare current time
    input.within_window == true
}

# =============================================================================
# Drift-Specific Rules
# =============================================================================

# Deny drift remediation without version comparison
deny[msg] {
    input.plan.type == "drift_plan"
    not input.plan.version_comparison
    msg := "SAFETY: Drift remediation plan must include version comparison data"
}

# =============================================================================
# DR (Disaster Recovery) Rules
# =============================================================================

# Deny DR failover without recent test
deny[msg] {
    input.plan.type == "dr_failover"
    input.plan.environment == "production"
    not input.dr_test_recent
    msg := "SAFETY: Production DR failover requires a recent DR test (within 30 days)"
}

# =============================================================================
# Compliance Rules
# =============================================================================

# Deny changes to compliance-critical assets without evidence
deny[msg] {
    input.plan.compliance_scope
    not input.plan.evidence_generated
    msg := "SAFETY: Changes to compliance-scoped assets require evidence generation"
}
