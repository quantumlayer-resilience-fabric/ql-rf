package profiles

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/quantumlayerhq/ql-rf/pkg/inspec"
)

// GetSOC2Mappings returns control mappings for SOC 2 Type II compliance.
func GetSOC2Mappings() []inspec.ControlMapping {
	return []inspec.ControlMapping{
		// CC6 - Logical and Physical Access Controls
		{
			InSpecControlID:     "soc2-cc6.1-access-control",
			MappingConfidence:   0.9,
			Notes:               "CC6.1 - The entity implements logical access security software, infrastructure, and architectures",
		},
		{
			InSpecControlID:     "soc2-cc6.2-authentication",
			MappingConfidence:   0.9,
			Notes:               "CC6.2 - Prior to issuing system credentials and granting system access",
		},
		{
			InSpecControlID:     "soc2-cc6.3-authorization",
			MappingConfidence:   0.9,
			Notes:               "CC6.3 - The entity authorizes, modifies, or removes access to data, software, functions, and services",
		},
		{
			InSpecControlID:     "soc2-cc6.6-encryption",
			MappingConfidence:   1.0,
			Notes:               "CC6.6 - The entity implements logical access security measures to protect against threats from sources outside its system boundaries",
		},
		{
			InSpecControlID:     "soc2-cc6.7-transmission-encryption",
			MappingConfidence:   1.0,
			Notes:               "CC6.7 - The entity restricts the transmission, movement, and removal of information",
		},
		{
			InSpecControlID:     "soc2-cc6.8-access-removal",
			MappingConfidence:   0.9,
			Notes:               "CC6.8 - The entity implements controls to prevent or detect and act upon the introduction of unauthorized software",
		},

		// CC7 - System Operations
		{
			InSpecControlID:     "soc2-cc7.1-detection",
			MappingConfidence:   0.9,
			Notes:               "CC7.1 - To meet its objectives, the entity uses detection and monitoring procedures to identify anomalies",
		},
		{
			InSpecControlID:     "soc2-cc7.2-monitoring",
			MappingConfidence:   1.0,
			Notes:               "CC7.2 - The entity monitors system components and the operation of those components for anomalies",
		},
		{
			InSpecControlID:     "soc2-cc7.3-incident-response",
			MappingConfidence:   0.8,
			Notes:               "CC7.3 - The entity evaluates security events to determine whether they could or have resulted in a failure",
		},
		{
			InSpecControlID:     "soc2-cc7.4-response-plan",
			MappingConfidence:   0.7,
			Notes:               "CC7.4 - The entity responds to identified security incidents by executing a defined incident response program",
		},

		// CC8 - Change Management
		{
			InSpecControlID:     "soc2-cc8.1-change-management",
			MappingConfidence:   0.8,
			Notes:               "CC8.1 - The entity authorizes, designs, develops or acquires, configures, documents, tests, approves, and implements changes",
		},

		// A1 - Availability
		{
			InSpecControlID:     "soc2-a1.1-availability",
			MappingConfidence:   0.9,
			Notes:               "A1.1 - The entity maintains, monitors, and evaluates current processing capacity and use of system components",
		},
		{
			InSpecControlID:     "soc2-a1.2-backup",
			MappingConfidence:   1.0,
			Notes:               "A1.2 - The entity authorizes, designs, develops or acquires, implements, operates, approves, maintains, and monitors backup and recovery procedures",
		},
		{
			InSpecControlID:     "soc2-a1.3-recovery",
			MappingConfidence:   1.0,
			Notes:               "A1.3 - The entity tests recovery plan procedures supporting system recovery",
		},

		// C1 - Confidentiality
		{
			InSpecControlID:     "soc2-c1.1-confidential-info",
			MappingConfidence:   0.9,
			Notes:               "C1.1 - The entity identifies and maintains confidential information to meet the entity's objectives",
		},
		{
			InSpecControlID:     "soc2-c1.2-disposal",
			MappingConfidence:   1.0,
			Notes:               "C1.2 - The entity disposes of confidential information to meet the entity's objectives",
		},

		// P1 - Privacy (if applicable)
		{
			InSpecControlID:     "soc2-p1.1-privacy-notice",
			MappingConfidence:   0.7,
			Notes:               "P1.1 - The entity provides notice to data subjects about its privacy practices",
		},
		{
			InSpecControlID:     "soc2-p2.1-data-collection",
			MappingConfidence:   0.8,
			Notes:               "P2.1 - The entity collects personal information only for the purposes identified in the notice",
		},

		// Infrastructure and system-level controls
		{
			InSpecControlID:     "soc2-system-hardening",
			MappingConfidence:   1.0,
			Notes:               "System hardening and secure configuration baseline",
		},
		{
			InSpecControlID:     "soc2-patch-management",
			MappingConfidence:   1.0,
			Notes:               "Timely application of security patches and updates",
		},
		{
			InSpecControlID:     "soc2-logging-monitoring",
			MappingConfidence:   1.0,
			Notes:               "Comprehensive logging and monitoring of system events",
		},
		{
			InSpecControlID:     "soc2-network-segmentation",
			MappingConfidence:   0.9,
			Notes:               "Network segmentation and boundary protection",
		},
		{
			InSpecControlID:     "soc2-vulnerability-management",
			MappingConfidence:   1.0,
			Notes:               "Regular vulnerability scanning and remediation",
		},
		{
			InSpecControlID:     "soc2-data-encryption-at-rest",
			MappingConfidence:   1.0,
			Notes:               "Encryption of sensitive data at rest",
		},
		{
			InSpecControlID:     "soc2-data-encryption-in-transit",
			MappingConfidence:   1.0,
			Notes:               "Encryption of sensitive data in transit",
		},
		{
			InSpecControlID:     "soc2-password-policy",
			MappingConfidence:   1.0,
			Notes:               "Strong password policies and complexity requirements",
		},
		{
			InSpecControlID:     "soc2-mfa-enforcement",
			MappingConfidence:   1.0,
			Notes:               "Multi-factor authentication for privileged access",
		},
		{
			InSpecControlID:     "soc2-session-management",
			MappingConfidence:   0.9,
			Notes:               "Secure session management and timeout policies",
		},
		{
			InSpecControlID:     "soc2-audit-logging",
			MappingConfidence:   1.0,
			Notes:               "Comprehensive audit logging of security events",
		},
		{
			InSpecControlID:     "soc2-log-retention",
			MappingConfidence:   1.0,
			Notes:               "Appropriate log retention and protection",
		},
		{
			InSpecControlID:     "soc2-time-synchronization",
			MappingConfidence:   1.0,
			Notes:               "Synchronized time across all systems for accurate logging",
		},
		{
			InSpecControlID:     "soc2-antimalware",
			MappingConfidence:   1.0,
			Notes:               "Anti-malware software installed and updated",
		},
		{
			InSpecControlID:     "soc2-firewall-configuration",
			MappingConfidence:   1.0,
			Notes:               "Proper firewall configuration and rule management",
		},
	}
}

// CreateSOC2Profile creates a SOC 2 Type II profile with control mappings.
func CreateSOC2Profile(ctx context.Context, svc *inspec.Service, frameworkID uuid.UUID) (*inspec.Profile, error) {
	// Create the profile
	profile := inspec.Profile{
		Name:        "soc2-type-ii-baseline",
		Version:     "1.0.0",
		Title:       "SOC 2 Type II Baseline",
		Maintainer:  "QuantumLayer",
		Summary:     "SOC 2 Type II compliance baseline checks for infrastructure and systems",
		FrameworkID: frameworkID,
		ProfileURL:  "https://github.com/quantumlayer/soc2-baseline",
		Platforms:   []string{"linux", "windows", "aws", "azure", "gcp"},
	}

	created, err := svc.CreateProfile(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	// Get mappings
	mappings := GetSOC2Mappings()

	// Create control mappings
	// Note: This would require the compliance_control_id to be set based on
	// actual compliance controls in the database. This is a placeholder.
	for _, mapping := range mappings {
		mapping.ProfileID = created.ID
		// In a real implementation, you would look up the compliance_control_id
		// based on the framework and control identifier
		_ = mapping
	}

	return created, nil
}
