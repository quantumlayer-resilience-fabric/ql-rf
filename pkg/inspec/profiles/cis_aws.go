package profiles

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/quantumlayerhq/ql-rf/pkg/inspec"
)

// GetCISAWSMappings returns control mappings for CIS AWS Foundations Benchmark.
func GetCISAWSMappings() []inspec.ControlMapping {
	return []inspec.ControlMapping{
		// Section 1: Identity and Access Management
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-1.4",
			MappingConfidence:   1.0,
			Notes:               "Ensure no root account access key exists",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-1.5",
			MappingConfidence:   1.0,
			Notes:               "Ensure MFA is enabled for the root account",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-1.6",
			MappingConfidence:   1.0,
			Notes:               "Ensure hardware MFA is enabled for the root account",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-1.10",
			MappingConfidence:   1.0,
			Notes:               "Ensure multi-factor authentication (MFA) is enabled for all IAM users",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-1.12",
			MappingConfidence:   1.0,
			Notes:               "Ensure credentials unused for 90 days or greater are disabled",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-1.14",
			MappingConfidence:   1.0,
			Notes:               "Ensure access keys are rotated every 90 days or less",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-1.16",
			MappingConfidence:   1.0,
			Notes:               "Ensure IAM policies are attached only to groups or roles",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-1.20",
			MappingConfidence:   1.0,
			Notes:               "Ensure a support role has been created to manage incidents with AWS Support",
		},

		// Section 2: Storage
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-2.1.1",
			MappingConfidence:   1.0,
			Notes:               "Ensure S3 Bucket Policy is set to deny HTTP requests",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-2.1.2",
			MappingConfidence:   1.0,
			Notes:               "Ensure MFA Delete is enabled on S3 buckets",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-2.2.1",
			MappingConfidence:   1.0,
			Notes:               "Ensure EBS volume encryption is enabled",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-2.3.1",
			MappingConfidence:   1.0,
			Notes:               "Ensure that encryption is enabled for RDS Instances",
		},

		// Section 3: Logging
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-3.1",
			MappingConfidence:   1.0,
			Notes:               "Ensure CloudTrail is enabled in all regions",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-3.2",
			MappingConfidence:   1.0,
			Notes:               "Ensure CloudTrail log file validation is enabled",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-3.3",
			MappingConfidence:   1.0,
			Notes:               "Ensure the S3 bucket used to store CloudTrail logs is not publicly accessible",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-3.4",
			MappingConfidence:   1.0,
			Notes:               "Ensure CloudTrail trails are integrated with CloudWatch Logs",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-3.5",
			MappingConfidence:   1.0,
			Notes:               "Ensure AWS Config is enabled in all regions",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-3.6",
			MappingConfidence:   1.0,
			Notes:               "Ensure S3 bucket access logging is enabled on the CloudTrail S3 bucket",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-3.7",
			MappingConfidence:   1.0,
			Notes:               "Ensure CloudTrail logs are encrypted at rest using KMS CMKs",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-3.8",
			MappingConfidence:   1.0,
			Notes:               "Ensure rotation for customer created CMKs is enabled",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-3.9",
			MappingConfidence:   1.0,
			Notes:               "Ensure VPC flow logging is enabled in all VPCs",
		},

		// Section 4: Monitoring
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-4.1",
			MappingConfidence:   1.0,
			Notes:               "Ensure a log metric filter and alarm exist for unauthorized API calls",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-4.2",
			MappingConfidence:   1.0,
			Notes:               "Ensure a log metric filter and alarm exist for Management Console sign-in without MFA",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-4.3",
			MappingConfidence:   1.0,
			Notes:               "Ensure a log metric filter and alarm exist for usage of root account",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-4.4",
			MappingConfidence:   1.0,
			Notes:               "Ensure a log metric filter and alarm exist for IAM policy changes",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-4.5",
			MappingConfidence:   1.0,
			Notes:               "Ensure a log metric filter and alarm exist for CloudTrail configuration changes",
		},

		// Section 5: Networking
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-5.1",
			MappingConfidence:   1.0,
			Notes:               "Ensure no Network ACLs allow ingress from 0.0.0.0/0 to remote server administration ports",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-5.2",
			MappingConfidence:   1.0,
			Notes:               "Ensure no security groups allow ingress from 0.0.0.0/0 to remote server administration ports",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-5.3",
			MappingConfidence:   1.0,
			Notes:               "Ensure the default security group of every VPC restricts all traffic",
		},
		{
			InSpecControlID:     "cis-aws-foundations-benchmark-5.4",
			MappingConfidence:   1.0,
			Notes:               "Ensure routing tables for VPC peering are least access",
		},
	}
}

// CreateCISAWSProfile creates a CIS AWS Foundations Benchmark profile with control mappings.
func CreateCISAWSProfile(ctx context.Context, svc *inspec.Service, frameworkID uuid.UUID) (*inspec.Profile, error) {
	// Create the profile
	profile := inspec.Profile{
		Name:        "cis-aws-foundations-benchmark",
		Version:     "1.5.0",
		Title:       "CIS AWS Foundations Benchmark",
		Maintainer:  "Center for Internet Security",
		Summary:     "CIS AWS Foundations Benchmark v1.5.0 compliance checks",
		FrameworkID: frameworkID,
		ProfileURL:  "https://github.com/inspec/cis-aws-foundations-baseline",
		Platforms:   []string{"aws"},
	}

	created, err := svc.CreateProfile(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	// Get mappings
	mappings := GetCISAWSMappings()

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
