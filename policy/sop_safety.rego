# QuantumLayer Resilience Fabric - SOP Safety Policies
# Package: ql.ai.sop
# Purpose: Validate AI-generated SOPs before activation
# Reference: ADR-010

package ql.ai.sop

import future.keywords.in
import future.keywords.every

# Default deny - all SOPs must explicitly pass validation
default allow := false

# =============================================================================
# Main allow rule - SOP is allowed if no deny rules fire
# =============================================================================

allow {
    count(deny) == 0
}

# =============================================================================
# SOP Structure Validation
# =============================================================================

# Deny SOPs without required metadata
deny[msg] {
    not input.sop.name
    msg := "SOP: Must have a name"
}

deny[msg] {
    not input.sop.version
    msg := "SOP: Must have a version"
}

deny[msg] {
    not input.sop.description
    msg := "SOP: Must have a description"
}

# Deny SOPs without steps
deny[msg] {
    not input.sop.steps
    msg := "SOP: Must have at least one step"
}

deny[msg] {
    count(input.sop.steps) == 0
    msg := "SOP: Must have at least one step"
}

# =============================================================================
# Step Validation
# =============================================================================

# Every step must have an ID
deny[msg] {
    some i, step in input.sop.steps
    not step.id
    msg := sprintf("SOP: Step %d must have an ID", [i + 1])
}

# Every step must have a name
deny[msg] {
    some i, step in input.sop.steps
    not step.name
    msg := sprintf("SOP: Step %d must have a name", [i + 1])
}

# Every step must have an action
deny[msg] {
    some i, step in input.sop.steps
    not step.action
    msg := sprintf("SOP: Step '%s' must have an action", [step.id])
}

# Action must have a type
deny[msg] {
    some step in input.sop.steps
    not step.action.type
    msg := sprintf("SOP: Step '%s' action must have a type", [step.id])
}

# =============================================================================
# Production SOP Rules
# =============================================================================

# Production SOPs must have approval required
deny[msg] {
    "production" in input.sop.scope.environments
    not input.sop.approval.required
    msg := "SOP: Production SOPs must require approval"
}

# Production SOPs must have rollback defined
deny[msg] {
    "production" in input.sop.scope.environments
    not input.sop.rollback
    msg := "SOP: Production SOPs must have rollback strategy defined"
}

# Production SOPs must have success criteria
deny[msg] {
    "production" in input.sop.scope.environments
    not input.sop.validation.success_criteria
    msg := "SOP: Production SOPs must have success criteria"
}

deny[msg] {
    "production" in input.sop.scope.environments
    count(input.sop.validation.success_criteria) == 0
    msg := "SOP: Production SOPs must have at least one success criterion"
}

# =============================================================================
# Timeout Validation
# =============================================================================

# Steps with state changes should have timeouts
deny[msg] {
    some step in input.sop.steps
    step.action.type in state_changing_actions
    not step.timeout
    msg := sprintf("SOP: Step '%s' with state-changing action must have timeout", [step.id])
}

# Global timeout required
deny[msg] {
    "production" in input.sop.scope.environments
    not input.sop.timeout
    msg := "SOP: Production SOPs must have a global timeout"
}

# =============================================================================
# Notification Rules
# =============================================================================

# Production SOPs should notify on failure
deny[msg] {
    "production" in input.sop.scope.environments
    not has_failure_notification
    msg := "SOP: Production SOPs must have failure notifications configured"
}

# =============================================================================
# Dangerous Action Rules
# =============================================================================

# Deny dangerous actions without explicit confirmation
deny[msg] {
    some step in input.sop.steps
    step.action.type in dangerous_actions
    not step.requires_confirmation
    msg := sprintf("SOP: Step '%s' uses dangerous action '%s' and must require confirmation", [step.id, step.action.type])
}

# Deny DR failover SOPs without recent drill
deny[msg] {
    some step in input.sop.steps
    step.action.type == "dr.failover"
    "production" in input.sop.scope.environments
    not input.context.dr_drill_recent
    msg := "SOP: DR failover in production requires recent DR drill (within 30 days)"
}

# =============================================================================
# Batch Size Limits
# =============================================================================

# Rollout steps should respect batch limits
deny[msg] {
    some step in input.sop.steps
    step.action.type in ["rollout.batch", "rollout.canary", "rollout.blue_green"]
    step.action.parameters.batch_percent
    step.action.parameters.batch_percent > 25
    "production" in input.sop.scope.environments
    msg := sprintf("SOP: Step '%s' batch size %d%% exceeds 25%% limit for production", [step.id, step.action.parameters.batch_percent])
}

# =============================================================================
# Helper Definitions
# =============================================================================

state_changing_actions := {
    "rollout.batch",
    "rollout.canary",
    "rollout.blue_green",
    "dr.failover",
    "dr.failback",
    "image.promote"
}

dangerous_actions := {
    "dr.failover",
    "dr.failback",
    "rollout.abort"
}

has_failure_notification {
    some notification in input.sop.notifications
    notification.when == "failure"
}

# =============================================================================
# Warnings (not denials)
# =============================================================================

warn[msg] {
    not input.sop.author
    msg := "SOP: Consider adding an author for accountability"
}

warn[msg] {
    not input.sop.tags
    msg := "SOP: Consider adding tags for organization"
}

warn[msg] {
    "production" in input.sop.scope.environments
    not input.sop.approval.min_approvers
    msg := "SOP: Consider requiring multiple approvers for production SOPs"
}

warn[msg] {
    some step in input.sop.steps
    not step.retries
    step.action.type in state_changing_actions
    msg := sprintf("SOP: Step '%s' could benefit from retry configuration", [step.id])
}
