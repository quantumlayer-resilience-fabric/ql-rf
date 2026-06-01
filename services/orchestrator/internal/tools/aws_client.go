// PR #19 / CONN-001 — AWS client interface and implementations for real cloud
// tool invocations.
//
// The orchestrator already wraps AWS in services/orchestrator/internal/executor/
// platform_aws.go for the plan-execution path. This file is the *tool-layer*
// surface: a narrow interface that read-only tools (and, in a follow-up PR,
// state-change tools with dry-run safety) call through.
//
// Two implementations:
//
//   - realAWSClient — wraps SDK v2's EC2 client. Validates credentials at
//     construction via STS GetCallerIdentity so misconfigurations fail fast
//     at orchestrator boot, not mid-demo.
//   - mockAWSClient — deterministic fixture. Used by unit tests and by CI
//     (RF_CONNECTORS_AWS_FALLBACK_TO_MOCK=true). Registration emits a loud
//     WARN log so production never silently uses fake data.
//
// Credentials are read from standard AWS env vars (AWS_REGION,
// AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY, AWS_PROFILE) by the SDK's default
// config chain. Optionally an STS AssumeRole ARN can be set via
// RF_CONNECTORS_AWS_ASSUME_ROLE_ARN. Credentials NEVER appear in tool
// parameters, results, audit log entries, or HTTP responses.
package tools

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// defaultAWSRegion is the fallback when no region is provided by config or
// tool parameter. us-east-1 is the AWS SDK's own historical default.
const defaultAWSRegion = "us-east-1"

// AWSClient is the narrow interface AWS tools call. Keeping it small means
// the mock client is trivial and the surface to audit stays tight. Each
// method is read-only by API contract for PR #19.
type AWSClient interface {
	// DescribeInstances lists EC2 instances in the given region.
	DescribeInstances(ctx context.Context, region string) ([]AWSInstance, error)
}

// AWSInstance is the redacted projection of an EC2 instance we surface to
// the audit log and the UI. Fields are deliberately the boring metadata —
// no security group ids, no IAM roles, no key pair names — to keep the
// blast radius of an audit-log leak minimal.
type AWSInstance struct {
	InstanceID   string            `json:"instance_id"`
	InstanceType string            `json:"instance_type"`
	State        string            `json:"state"`
	Region       string            `json:"region"`
	PrivateIP    string            `json:"private_ip,omitempty"`
	PublicIP     string            `json:"public_ip,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
	LaunchedAt   string            `json:"launched_at,omitempty"`
}

// realAWSClient wraps SDK v2's EC2 client. Constructed once at boot; the
// aws.Config is shared across regional calls via Copy() per request.
type realAWSClient struct {
	cfg aws.Config
	log *logger.Logger
}

// NewRealAWSClient builds a real EC2-capable client. Validates credentials
// at construction via STS GetCallerIdentity and returns an error if the
// credentials are missing or unusable. The caller decides whether to fall
// back to a mock client or to skip tool registration entirely.
//
// The validation call is one STS request per orchestrator boot — negligible
// cost, and the difference between "boots cleanly but tool silently broken"
// and "boots cleanly and tool actually works" at demo time.
func NewRealAWSClient(ctx context.Context, cfg config.AWSConfig, log *logger.Logger) (AWSClient, error) {
	region := cfg.Region
	if region == "" {
		region = envOr("AWS_REGION", defaultAWSRegion)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load aws default config: %w", err)
	}

	if cfg.AssumeRoleARN != "" {
		stsClient := sts.NewFromConfig(awsCfg)
		creds := stscreds.NewAssumeRoleProvider(stsClient, cfg.AssumeRoleARN, func(o *stscreds.AssumeRoleOptions) {
			if cfg.AssumeRoleExternalID != "" {
				o.ExternalID = aws.String(cfg.AssumeRoleExternalID)
			}
		})
		awsCfg.Credentials = aws.NewCredentialsCache(creds)
	}

	// Boot-time credential validation. If this fails the caller will fall
	// back to mock or skip registration. Returning a typed error here lets
	// the caller log the env-state without leaking the key material itself.
	stsCheckCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := sts.NewFromConfig(awsCfg).GetCallerIdentity(stsCheckCtx, &sts.GetCallerIdentityInput{}); err != nil {
		return nil, fmt.Errorf("aws credential validation (sts:GetCallerIdentity) failed: %w", err)
	}

	return &realAWSClient{
		cfg: awsCfg,
		log: log.WithComponent("aws-tools"),
	}, nil
}

func (c *realAWSClient) DescribeInstances(ctx context.Context, region string) ([]AWSInstance, error) {
	if region == "" {
		region = c.cfg.Region
	}
	regionalCfg := c.cfg.Copy()
	regionalCfg.Region = region
	client := ec2.NewFromConfig(regionalCfg)

	var (
		results   []AWSInstance
		nextToken *string
	)
	for {
		out, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{NextToken: nextToken})
		if err != nil {
			return nil, fmt.Errorf("ec2 describe instances: %w", err)
		}
		for i := range out.Reservations {
			res := &out.Reservations[i]
			for j := range res.Instances {
				results = append(results, normalizeEC2Instance(&res.Instances[j], region))
			}
		}
		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}
	return results, nil
}

// normalizeEC2Instance projects the (huge) ec2.Instance type into the small
// AWSInstance shape we surface. Intentionally drops fields that have any
// security relevance (security groups, IAM roles, key names) — the audit
// log shouldn't carry those. Takes a pointer to avoid copying the 670+
// byte ec2Types.Instance struct per iteration.
func normalizeEC2Instance(inst *ec2Types.Instance, region string) AWSInstance {
	result := AWSInstance{
		InstanceID:   aws.ToString(inst.InstanceId),
		InstanceType: string(inst.InstanceType),
		Region:       region,
		PrivateIP:    aws.ToString(inst.PrivateIpAddress),
		PublicIP:     aws.ToString(inst.PublicIpAddress),
	}
	if inst.State != nil {
		result.State = string(inst.State.Name)
	}
	if inst.LaunchTime != nil {
		result.LaunchedAt = inst.LaunchTime.UTC().Format(time.RFC3339)
	}
	if len(inst.Tags) > 0 {
		result.Tags = make(map[string]string, len(inst.Tags))
		for _, t := range inst.Tags {
			result.Tags[aws.ToString(t.Key)] = aws.ToString(t.Value)
		}
	}
	return result
}

// mockAWSClient returns a deterministic two-instance fixture. Used by unit
// tests and by CI/dev when RF_CONNECTORS_AWS_FALLBACK_TO_MOCK=true. The
// fixture is intentionally obvious ("i-mock-0001") so a viewer who sees it
// in real audit logs knows immediately something is wrong.
type mockAWSClient struct{}

// NewMockAWSClient constructs the deterministic mock client. The orchestrator
// emits a loud WARN log when this is used in place of the real client.
func NewMockAWSClient() AWSClient {
	return &mockAWSClient{}
}

func (m *mockAWSClient) DescribeInstances(_ context.Context, region string) ([]AWSInstance, error) {
	if region == "" {
		region = "us-east-1"
	}
	return []AWSInstance{
		{
			InstanceID:   "i-mock-0001",
			InstanceType: "t3.small",
			State:        "running",
			Region:       region,
			Tags:         map[string]string{"Name": "mock-web", "Environment": "demo"},
			LaunchedAt:   "2026-01-01T00:00:00Z",
		},
		{
			InstanceID:   "i-mock-0002",
			InstanceType: "t3.medium",
			State:        "stopped",
			Region:       region,
			Tags:         map[string]string{"Name": "mock-batch", "Environment": "demo"},
			LaunchedAt:   "2026-01-02T00:00:00Z",
		},
	}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
