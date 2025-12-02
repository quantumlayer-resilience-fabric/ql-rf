# QuantumLayer Resilience Fabric - Golden Image Safety Policies
# Package: ql.ai.images
# Purpose: Validate AI-generated image specifications before build/promotion
# Reference: ADR-011

package ql.ai.images

import future.keywords.in
import future.keywords.every

# Default deny - all image specs must explicitly pass validation
default allow := false

# =============================================================================
# Main allow rule
# =============================================================================

allow {
    count(deny) == 0
}

# =============================================================================
# Image Specification Validation
# =============================================================================

# Must have image family
deny[msg] {
    not input.image.family
    msg := "IMAGE: Must specify image family"
}

# Must have version
deny[msg] {
    not input.image.version
    msg := "IMAGE: Must specify version"
}

# Must have base image
deny[msg] {
    not input.image.base_image
    msg := "IMAGE: Must specify base_image"
}

# Base image must be from approved sources
deny[msg] {
    input.image.base_image
    not base_image_approved
    msg := sprintf("IMAGE: Base image '%s' not from approved source", [input.image.base_image])
}

# =============================================================================
# Security Hardening Requirements
# =============================================================================

# Must have hardening configuration
deny[msg] {
    not input.image.hardening
    msg := "IMAGE: Must specify hardening configuration"
}

# CIS benchmark required for production images
deny[msg] {
    input.image.for_production
    not input.image.hardening.cis_benchmark
    msg := "IMAGE: Production images must specify CIS benchmark level"
}

# Minimum CIS Level 1 for production
deny[msg] {
    input.image.for_production
    input.image.hardening.cis_benchmark
    not input.image.hardening.cis_benchmark in ["level_1", "level_2", "stig"]
    msg := sprintf("IMAGE: CIS benchmark '%s' not acceptable for production (need level_1, level_2, or stig)", [input.image.hardening.cis_benchmark])
}

# SSH hardening required
deny[msg] {
    input.image.for_production
    not input.image.hardening.ssh
    msg := "IMAGE: Production images must have SSH hardening configured"
}

# Password authentication must be disabled
deny[msg] {
    input.image.hardening.ssh
    input.image.hardening.ssh.password_auth == true
    msg := "IMAGE: Password authentication must be disabled"
}

# Root login must be disabled
deny[msg] {
    input.image.hardening.ssh
    input.image.hardening.ssh.root_login == true
    msg := "IMAGE: Root login must be disabled"
}

# =============================================================================
# Package Management
# =============================================================================

# Must have packages defined
deny[msg] {
    not input.image.packages
    msg := "IMAGE: Must define packages section"
}

# No blacklisted packages
deny[msg] {
    some pkg in input.image.packages.install
    pkg in blacklisted_packages
    msg := sprintf("IMAGE: Package '%s' is blacklisted", [pkg])
}

# Security updates must be enabled
deny[msg] {
    input.image.for_production
    input.image.packages.auto_security_updates == false
    msg := "IMAGE: Production images must have auto security updates enabled"
}

# =============================================================================
# Monitoring Requirements
# =============================================================================

# Must have monitoring agents for production
deny[msg] {
    input.image.for_production
    not has_monitoring_agent
    msg := "IMAGE: Production images must include monitoring agent"
}

# Must have log forwarding for production
deny[msg] {
    input.image.for_production
    not has_log_forwarding
    msg := "IMAGE: Production images must include log forwarding agent"
}

# =============================================================================
# Multi-Platform Validation
# =============================================================================

# Must specify at least one platform
deny[msg] {
    not input.image.platforms
    msg := "IMAGE: Must specify target platforms"
}

deny[msg] {
    count(input.image.platforms) == 0
    msg := "IMAGE: Must specify at least one target platform"
}

# Platform must be supported
deny[msg] {
    some platform in input.image.platforms
    not platform in supported_platforms
    msg := sprintf("IMAGE: Platform '%s' not supported", [platform])
}

# =============================================================================
# Testing Requirements
# =============================================================================

# Must have validation tests
deny[msg] {
    not input.image.validation
    msg := "IMAGE: Must specify validation requirements"
}

deny[msg] {
    not input.image.validation.tests
    msg := "IMAGE: Must specify validation tests"
}

deny[msg] {
    count(input.image.validation.tests) == 0
    msg := "IMAGE: Must have at least one validation test"
}

# Security scan required for production
deny[msg] {
    input.image.for_production
    not input.image.validation.security_scan
    msg := "IMAGE: Production images must have security scan enabled"
}

# =============================================================================
# Version Control
# =============================================================================

# Semantic versioning required
deny[msg] {
    input.image.version
    not regex.match(`^v?\d+\.\d+\.\d+(-[a-zA-Z0-9]+)?$`, input.image.version)
    msg := sprintf("IMAGE: Version '%s' must follow semantic versioning (e.g., v1.2.3)", [input.image.version])
}

# =============================================================================
# Helper Definitions
# =============================================================================

base_image_approved {
    some source in approved_base_sources
    startswith(input.image.base_image, source)
}

approved_base_sources := [
    "ubuntu:",
    "debian:",
    "rhel:",
    "centos:",
    "amazon-linux:",
    "windows-server:"
]

blacklisted_packages := {
    "telnet",
    "rsh",
    "rlogin",
    "ftp",
    "tftp",
    "nis",
    "talk"
}

supported_platforms := {
    "aws",
    "azure",
    "gcp",
    "vmware",
    "proxmox"
}

has_monitoring_agent {
    some pkg in input.image.packages.install
    pkg in monitoring_agents
}

monitoring_agents := {
    "datadog-agent",
    "cloudwatch-agent",
    "prometheus-node-exporter",
    "telegraf",
    "newrelic-infra"
}

has_log_forwarding {
    some pkg in input.image.packages.install
    pkg in log_forwarders
}

log_forwarders := {
    "fluent-bit",
    "fluentd",
    "filebeat",
    "vector",
    "rsyslog"
}

# =============================================================================
# Warnings
# =============================================================================

warn[msg] {
    not input.image.owner
    msg := "IMAGE: Consider specifying an owner for accountability"
}

warn[msg] {
    not input.image.retention_days
    msg := "IMAGE: Consider specifying retention policy"
}

warn[msg] {
    input.image.hardening.cis_benchmark == "level_1"
    input.image.for_production
    msg := "IMAGE: Consider CIS Level 2 for production workloads"
}
