package profiles

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/quantumlayerhq/ql-rf/pkg/inspec"
)

// CISLinuxProfile represents the CIS Linux Benchmark profile mappings.
type CISLinuxProfile struct {
	ProfileID   uuid.UUID
	FrameworkID uuid.UUID
}

// GetCISLinuxLevel1Mappings returns control mappings for CIS Linux Level 1.
func GetCISLinuxLevel1Mappings() []inspec.ControlMapping {
	return []inspec.ControlMapping{
		// Section 1: Initial Setup
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_1.1.1.1_Ensure_cramfs_kernel_module_is_not_available",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 1.1.1.1 - Ensure cramfs kernel module is not available",
		},
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_1.1.2.1_Ensure_tmp_is_a_separate_partition",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 1.1.2.1 - Ensure /tmp is a separate partition",
		},
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_1.1.2.2_Ensure_nodev_option_set_on_tmp_partition",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 1.1.2.2 - Ensure nodev option set on /tmp partition",
		},

		// Section 2: Services
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_2.1.1_Ensure_autofs_services_are_not_in_use",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 2.1.1 - Ensure autofs services are not in use",
		},
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_2.2.1_Ensure_xorg-x11-server-common_is_not_installed",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 2.2.1 - Ensure X Window System is not installed",
		},

		// Section 3: Network Configuration
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_3.1.1_Ensure_system_is_checked_to_determine_if_IPv6_is_enabled",
			MappingConfidence:   0.9,
			Notes:               "CIS Linux 3.1.1 - Verify network settings",
		},
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_3.2.1_Ensure_IP_forwarding_is_disabled",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 3.2.1 - Ensure IP forwarding is disabled",
		},

		// Section 4: Logging and Auditing
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_4.1.1.1_Ensure_auditd_is_installed",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 4.1.1.1 - Ensure auditd is installed",
		},
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_4.1.1.2_Ensure_auditd_service_is_enabled",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 4.1.1.2 - Ensure auditd service is enabled and active",
		},
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_4.1.2.1_Ensure_audit_log_storage_size_is_configured",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 4.1.2.1 - Ensure audit log storage size is configured",
		},

		// Section 5: Access, Authentication and Authorization
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_5.1.1_Ensure_permissions_on_etccrontab_are_configured",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 5.1.1 - Ensure permissions on /etc/crontab are configured",
		},
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_5.2.1_Ensure_permissions_on_etcsshsshd_config_are_configured",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 5.2.1 - Ensure permissions on /etc/ssh/sshd_config are configured",
		},
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_5.2.4_Ensure_SSH_access_is_limited",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 5.2.4 - Ensure SSH access is limited",
		},
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_5.2.5_Ensure_SSH_LogLevel_is_appropriate",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 5.2.5 - Ensure SSH LogLevel is appropriate",
		},
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_5.2.10_Ensure_SSH_PermitRootLogin_is_disabled",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 5.2.10 - Ensure SSH root login is disabled",
		},

		// Section 6: System Maintenance
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_6.1.1_Ensure_permissions_on_etcpasswd_are_configured",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 6.1.1 - Ensure permissions on /etc/passwd are configured",
		},
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_6.1.2_Ensure_permissions_on_etcpasswd-_are_configured",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 6.1.2 - Ensure permissions on /etc/passwd- are configured",
		},
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_6.1.3_Ensure_permissions_on_etcshadow_are_configured",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 6.1.3 - Ensure permissions on /etc/shadow are configured",
		},
	}
}

// GetCISLinuxLevel2Mappings returns additional control mappings for CIS Linux Level 2.
func GetCISLinuxLevel2Mappings() []inspec.ControlMapping {
	// Level 2 includes all Level 1 controls plus additional hardening
	mappings := GetCISLinuxLevel1Mappings()

	// Additional Level 2 controls
	level2 := []inspec.ControlMapping{
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_1.1.1.2_Ensure_freevxfs_kernel_module_is_not_available",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 1.1.1.2 - Ensure freevxfs kernel module is not available",
		},
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_1.1.1.3_Ensure_hfs_kernel_module_is_not_available",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 1.1.1.3 - Ensure hfs kernel module is not available",
		},
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_1.1.1.4_Ensure_hfsplus_kernel_module_is_not_available",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 1.1.1.4 - Ensure hfsplus kernel module is not available",
		},
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_2.1.2_Ensure_avahi_daemon_services_are_not_in_use",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 2.1.2 - Ensure Avahi Server is not installed",
		},
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_3.2.2_Ensure_packet_redirect_sending_is_disabled",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 3.2.2 - Ensure packet redirect sending is disabled",
		},
		{
			InSpecControlID:     "xccdf_org.cisecurity.benchmarks_rule_5.2.15_Ensure_SSH_warning_banner_is_configured",
			MappingConfidence:   1.0,
			Notes:               "CIS Linux 5.2.15 - Ensure SSH warning banner is configured",
		},
	}

	return append(mappings, level2...)
}

// CreateCISLinuxProfile creates a CIS Linux profile with control mappings.
func CreateCISLinuxProfile(ctx context.Context, svc *inspec.Service, frameworkID uuid.UUID, level int) (*inspec.Profile, error) {
	// Create the profile
	profileName := fmt.Sprintf("cis-linux-level-%d", level)
	profile := inspec.Profile{
		Name:        profileName,
		Version:     "1.1.0",
		Title:       fmt.Sprintf("CIS Linux Benchmark Level %d", level),
		Maintainer:  "Center for Internet Security",
		Summary:     fmt.Sprintf("CIS Linux Benchmark Level %d compliance checks", level),
		FrameworkID: frameworkID,
		ProfileURL:  "https://github.com/dev-sec/cis-linux-benchmark",
		Platforms:   []string{"linux", "ubuntu", "debian", "redhat", "centos", "amazon-linux"},
	}

	created, err := svc.CreateProfile(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	// Get the appropriate mappings based on level
	var mappings []inspec.ControlMapping
	if level == 2 {
		mappings = GetCISLinuxLevel2Mappings()
	} else {
		mappings = GetCISLinuxLevel1Mappings()
	}

	// Create control mappings
	// Note: This would require the compliance_control_id to be set based on
	// actual compliance controls in the database. This is a placeholder.
	for _, mapping := range mappings {
		mapping.ProfileID = created.ID
		// In a real implementation, you would look up the compliance_control_id
		// based on the framework and control identifier
		// For now, we'll skip creating the mappings without the compliance control ID
		_ = mapping
	}

	return created, nil
}
