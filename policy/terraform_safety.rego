# QuantumLayer Resilience Fabric - Terraform Module Safety Policies
# Package: ql.ai.terraform
# Purpose: Validate AI-generated Terraform modules before apply
# Reference: ADR-012

package ql.ai.terraform

import future.keywords.in
import future.keywords.every

# Default deny - all Terraform modules must pass validation
default allow := false

# =============================================================================
# Main allow rule
# =============================================================================

allow {
    count(deny) == 0
}

# =============================================================================
# Module Structure Validation
# =============================================================================

# Must have terraform block
deny[msg] {
    not input.module.terraform
    msg := "TERRAFORM: Must have terraform configuration block"
}

# Must specify required version
deny[msg] {
    not input.module.terraform.required_version
    msg := "TERRAFORM: Must specify required_version"
}

# Must have provider configuration
deny[msg] {
    not input.module.provider
    msg := "TERRAFORM: Must have provider configuration"
}

# =============================================================================
# Provider Validation
# =============================================================================

# Only approved providers
deny[msg] {
    some provider_name, _ in input.module.provider
    not provider_name in approved_providers
    msg := sprintf("TERRAFORM: Provider '%s' not in approved list", [provider_name])
}

# Provider version must be pinned
deny[msg] {
    some provider_name, config in input.module.terraform.required_providers
    not config.version
    msg := sprintf("TERRAFORM: Provider '%s' must have version constraint", [provider_name])
}

# No latest tags
deny[msg] {
    some provider_name, config in input.module.terraform.required_providers
    config.version == "latest"
    msg := sprintf("TERRAFORM: Provider '%s' cannot use 'latest' version", [provider_name])
}

# =============================================================================
# Security Rules
# =============================================================================

# No hardcoded secrets
deny[msg] {
    walk(input.module, [path, value])
    is_string(value)
    looks_like_secret(path[count(path)-1])
    not is_variable_reference(value)
    msg := sprintf("TERRAFORM: Potential hardcoded secret at path %v", [path])
}

# No public IPs without explicit flag
deny[msg] {
    input.module.resource.aws_instance
    some instance_name, instance in input.module.resource.aws_instance
    instance.associate_public_ip_address == true
    not input.context.allow_public_ip
    msg := sprintf("TERRAFORM: Instance '%s' has public IP - requires explicit approval", [instance_name])
}

# Security groups must not allow 0.0.0.0/0 ingress on sensitive ports
deny[msg] {
    input.module.resource.aws_security_group
    some sg_name, sg in input.module.resource.aws_security_group
    some rule in sg.ingress
    rule.cidr_blocks
    "0.0.0.0/0" in rule.cidr_blocks
    rule.from_port in sensitive_ports
    msg := sprintf("TERRAFORM: Security group '%s' allows 0.0.0.0/0 on sensitive port %d", [sg_name, rule.from_port])
}

# Azure NSG rules - same check
deny[msg] {
    input.module.resource.azurerm_network_security_rule
    some rule_name, rule in input.module.resource.azurerm_network_security_rule
    rule.source_address_prefix == "*"
    rule.destination_port_range in sensitive_port_strings
    rule.access == "Allow"
    msg := sprintf("TERRAFORM: NSG rule '%s' allows * source on sensitive port", [rule_name])
}

# =============================================================================
# Encryption Requirements
# =============================================================================

# EBS volumes must be encrypted
deny[msg] {
    input.module.resource.aws_ebs_volume
    some vol_name, vol in input.module.resource.aws_ebs_volume
    not vol.encrypted
    msg := sprintf("TERRAFORM: EBS volume '%s' must be encrypted", [vol_name])
}

deny[msg] {
    input.module.resource.aws_ebs_volume
    some vol_name, vol in input.module.resource.aws_ebs_volume
    vol.encrypted == false
    msg := sprintf("TERRAFORM: EBS volume '%s' must be encrypted", [vol_name])
}

# S3 buckets must have encryption
deny[msg] {
    input.module.resource.aws_s3_bucket
    some bucket_name, _ in input.module.resource.aws_s3_bucket
    not has_s3_encryption(bucket_name)
    msg := sprintf("TERRAFORM: S3 bucket '%s' must have encryption configured", [bucket_name])
}

# RDS must be encrypted
deny[msg] {
    input.module.resource.aws_db_instance
    some db_name, db in input.module.resource.aws_db_instance
    db.storage_encrypted == false
    msg := sprintf("TERRAFORM: RDS instance '%s' must have storage encryption enabled", [db_name])
}

# Azure storage must have encryption
deny[msg] {
    input.module.resource.azurerm_storage_account
    some sa_name, sa in input.module.resource.azurerm_storage_account
    sa.enable_https_traffic_only == false
    msg := sprintf("TERRAFORM: Storage account '%s' must enforce HTTPS", [sa_name])
}

# =============================================================================
# Tagging Requirements
# =============================================================================

# Resources must have required tags
deny[msg] {
    input.module.resource
    some resource_type, resources in input.module.resource
    resource_type in taggable_resources
    some resource_name, resource in resources
    not has_required_tags(resource)
    msg := sprintf("TERRAFORM: Resource '%s.%s' missing required tags (ManagedBy, Environment)", [resource_type, resource_name])
}

# =============================================================================
# Cost Control
# =============================================================================

# Instance types must be in approved list (if configured)
deny[msg] {
    input.context.approved_instance_types
    input.module.resource.aws_instance
    some instance_name, instance in input.module.resource.aws_instance
    not instance.instance_type in input.context.approved_instance_types
    msg := sprintf("TERRAFORM: Instance type '%s' for '%s' not in approved list", [instance.instance_type, instance_name])
}

# No reserved instances without approval
deny[msg] {
    input.module.resource.aws_ec2_capacity_reservation
    not input.context.allow_reserved_capacity
    msg := "TERRAFORM: Reserved capacity requires explicit approval"
}

# =============================================================================
# High Availability
# =============================================================================

# RDS in production must be multi-az
deny[msg] {
    input.context.environment == "production"
    input.module.resource.aws_db_instance
    some db_name, db in input.module.resource.aws_db_instance
    not db.multi_az
    msg := sprintf("TERRAFORM: Production RDS '%s' must be multi-az", [db_name])
}

deny[msg] {
    input.context.environment == "production"
    input.module.resource.aws_db_instance
    some db_name, db in input.module.resource.aws_db_instance
    db.multi_az == false
    msg := sprintf("TERRAFORM: Production RDS '%s' must be multi-az", [db_name])
}

# =============================================================================
# Backup Requirements
# =============================================================================

# RDS must have backups enabled
deny[msg] {
    input.module.resource.aws_db_instance
    some db_name, db in input.module.resource.aws_db_instance
    db.backup_retention_period == 0
    msg := sprintf("TERRAFORM: RDS '%s' must have backup retention > 0", [db_name])
}

# Production backups must be >= 7 days
deny[msg] {
    input.context.environment == "production"
    input.module.resource.aws_db_instance
    some db_name, db in input.module.resource.aws_db_instance
    db.backup_retention_period
    db.backup_retention_period < 7
    msg := sprintf("TERRAFORM: Production RDS '%s' backup retention must be >= 7 days", [db_name])
}

# =============================================================================
# Helper Definitions
# =============================================================================

approved_providers := {
    "aws",
    "azurerm",
    "google",
    "kubernetes",
    "helm",
    "random",
    "null",
    "local",
    "tls"
}

sensitive_ports := {22, 3389, 5432, 3306, 1433, 27017, 6379, 9200}
sensitive_port_strings := {"22", "3389", "5432", "3306", "1433", "27017", "6379", "9200"}

taggable_resources := {
    "aws_instance",
    "aws_ebs_volume",
    "aws_s3_bucket",
    "aws_db_instance",
    "aws_vpc",
    "aws_subnet",
    "azurerm_virtual_machine",
    "azurerm_storage_account",
    "azurerm_resource_group"
}

looks_like_secret(key) {
    lower(key) in {"password", "secret", "api_key", "apikey", "token", "credential", "private_key"}
}

is_variable_reference(value) {
    startswith(value, "${var.")
}

is_variable_reference(value) {
    startswith(value, "var.")
}

has_required_tags(resource) {
    resource.tags
    resource.tags.ManagedBy
    resource.tags.Environment
}

has_required_tags(resource) {
    resource.tags
    resource.tags.managed_by
    resource.tags.environment
}

has_s3_encryption(bucket_name) {
    input.module.resource.aws_s3_bucket_server_side_encryption_configuration
    some _, enc in input.module.resource.aws_s3_bucket_server_side_encryption_configuration
    enc.bucket == sprintf("${aws_s3_bucket.%s.id}", [bucket_name])
}

has_s3_encryption(bucket_name) {
    input.module.resource.aws_s3_bucket_server_side_encryption_configuration
    some _, enc in input.module.resource.aws_s3_bucket_server_side_encryption_configuration
    enc.bucket == sprintf("aws_s3_bucket.%s.id", [bucket_name])
}

# =============================================================================
# Warnings
# =============================================================================

warn[msg] {
    not input.module.output
    msg := "TERRAFORM: Consider adding outputs for resource IDs"
}

warn[msg] {
    input.module.resource.aws_instance
    some instance_name, instance in input.module.resource.aws_instance
    not instance.monitoring
    msg := sprintf("TERRAFORM: Consider enabling detailed monitoring for instance '%s'", [instance_name])
}

warn[msg] {
    input.module.resource.aws_instance
    some instance_name, instance in input.module.resource.aws_instance
    instance.monitoring == false
    msg := sprintf("TERRAFORM: Consider enabling detailed monitoring for instance '%s'", [instance_name])
}
