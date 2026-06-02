// PR #21 / CONN-003 — LIVE SSM client.
//
// SAFETY (READ THIS BEFORE EDITING):
// This is THE single file in the tools package allowed to import
// `github.com/aws/aws-sdk-go-v2/service/ssm`. The structural safety test
// in no_ssm_sdk_import_test.go grants this file an allowlist exception by
// name, and the complementary TestLiveSSMClient_IsTheOnlyFileImportingSDK
// asserts (positive direction) that this file DOES import the SDK. Both
// tests run on every push.
//
// Why this discipline matters: PR #20 (dry-run only) made a live
// SendCommand call structurally unreachable from the tools package. PR #21
// re-introduces reachability deliberately and ONLY through this file, so
// auditors can grep for the SDK import path and see exactly one match.
//
// Live mode is gated at four layers, every layer independently enforced:
//
//  1. Boot env opt-in: RF_CONNECTORS_AWS_ALLOW_LIVE_PATCH=true. Without
//     it, registerSSMLiveTools in main.go is a no-op and this client is
//     never constructed.
//  2. Mock-conflict refusal: if FallbackToMock is also true, main.go
//     exits 1 at boot. The combination is incoherent.
//  3. Per-instance whitelist: parsed from
//     RF_CONNECTORS_AWS_LIVE_PATCH_WHITELIST_INSTANCE_IDS at boot. Empty
//     + ALLOW_LIVE_PATCH=true also fails boot.
//  4. Two-approver workflow: enforced by OPA policy
//     (tool_authorization.rego) AND by the coApproveTask handler. The
//     live tool's Execute method also re-checks the whitelist before
//     calling SendCommand, as defense in depth.
package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	pkgconfig "github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// LiveSSMClient SENDS commands to AWS SSM. Distinct from SSMClient (PR #20)
// which only BUILDS command plans. A type-level separation so a caller
// expecting dry-run shape can't accidentally call SendCommand.
type LiveSSMClient interface {
	// SendCommand fires ssm:SendCommand against the plan's instances and
	// returns the AWS-assigned command_id on success. Validates against
	// the whitelist (set at construction) before calling the SDK.
	SendCommand(ctx context.Context, plan *SSMCommandPlan) (commandID string, err error)
}

// liveWhitelistEnvVar is the env name parsed at boot for the per-instance
// allowlist. Comma-separated. Exported as a const so the boot path and the
// docs/README reference the same string.
const liveWhitelistEnvVar = "RF_CONNECTORS_AWS_LIVE_PATCH_WHITELIST_INSTANCE_IDS"

// realLiveSSMClient is the production client. The SDK import in this file
// is the only one in the package; this struct holds the live ssm.Client.
type realLiveSSMClient struct {
	sdk       *ssm.Client
	whitelist []string
	log       *logger.Logger
}

// NewLiveSSMClient builds a real, credential-validated live SSM client.
// Reuses the same STS GetCallerIdentity check as PR #19's NewRealAWSClient
// so credential misconfigurations fail at boot, not on first SendCommand.
func NewLiveSSMClient(ctx context.Context, cfg pkgconfig.AWSConfig, whitelist []string, log *logger.Logger) (LiveSSMClient, error) {
	if len(whitelist) == 0 {
		return nil, fmt.Errorf("NewLiveSSMClient: refuse to construct with empty whitelist")
	}

	region := cfg.Region
	if region == "" {
		region = defaultAWSRegion
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

	stsCheckCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := sts.NewFromConfig(awsCfg).GetCallerIdentity(stsCheckCtx, &sts.GetCallerIdentityInput{}); err != nil {
		return nil, fmt.Errorf("live ssm credential validation (sts:GetCallerIdentity) failed: %w", err)
	}

	return &realLiveSSMClient{
		sdk:       ssm.NewFromConfig(awsCfg),
		whitelist: append([]string(nil), whitelist...),
		log:       log.WithComponent("ssm-live"),
	}, nil
}

// SendCommand validates every target instance against the whitelist, then
// fires ssm:SendCommand. Returns the AWS command_id verbatim on success.
// Failures from the SDK are wrapped with the plan's region and the first
// instance for log triage but the SDK error is preserved via %w.
func (c *realLiveSSMClient) SendCommand(ctx context.Context, plan *SSMCommandPlan) (string, error) {
	if plan == nil {
		return "", fmt.Errorf("SendCommand: nil plan")
	}
	if err := requireAllWhitelisted(plan.InstanceIDs, c.whitelist); err != nil {
		return "", err
	}

	out, err := c.sdk.SendCommand(ctx, &ssm.SendCommandInput{
		DocumentName:    aws.String(plan.DocumentName),
		DocumentVersion: aws.String(plan.DocumentVersion),
		InstanceIds:     append([]string(nil), plan.InstanceIDs...),
		Parameters:      cloneStringSliceMap(plan.Parameters),
		Comment:         aws.String(plan.Comment),
		TimeoutSeconds:  aws.Int32(plan.TimeoutSeconds),
	})
	if err != nil {
		return "", fmt.Errorf("ssm:SendCommand region=%s first_instance=%s: %w",
			plan.Region, firstOrEmpty(plan.InstanceIDs), err)
	}
	if out == nil || out.Command == nil || out.Command.CommandId == nil {
		return "", fmt.Errorf("ssm:SendCommand returned no command id")
	}
	c.log.Info("live ssm:SendCommand fired",
		"command_id", aws.ToString(out.Command.CommandId),
		"region", plan.Region,
		"instance_count", len(plan.InstanceIDs),
	)
	return aws.ToString(out.Command.CommandId), nil
}

// mockLiveSSMClient is the test/CI variant. Lives in the same file as the
// real client so a single place defines "what does live mode look like."
// Validates the whitelist (mirroring the real client's defense in depth)
// but never touches the AWS SDK — the response is a deterministic
// cmd-mock-<hex> id so audit-log consumers can recognize mock origin.
type mockLiveSSMClient struct {
	whitelist []string
}

// NewMockLiveSSMClient constructs a mock that respects the same whitelist
// contract as the real client. Used by tests and by local smoke when
// RF_CONNECTORS_AWS_LIVE_PATCH_CLIENT_MODE=mock.
func NewMockLiveSSMClient(whitelist []string) LiveSSMClient {
	return &mockLiveSSMClient{whitelist: append([]string(nil), whitelist...)}
}

// SendCommand returns cmd-mock-<random hex> without calling the SDK. The
// whitelist check runs first so tests against the mock catch whitelist
// regressions just as the real client would.
func (m *mockLiveSSMClient) SendCommand(_ context.Context, plan *SSMCommandPlan) (string, error) {
	if plan == nil {
		return "", fmt.Errorf("SendCommand: nil plan")
	}
	if err := requireAllWhitelisted(plan.InstanceIDs, m.whitelist); err != nil {
		return "", err
	}
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("mock SendCommand: read entropy: %w", err)
	}
	return "cmd-mock-" + hex.EncodeToString(buf), nil
}

// LoadInstanceWhitelistFromEnv parses the comma-separated whitelist from
// the env var. Returns nil for an unset or empty var. Exposed so the boot
// path can produce the same parsed result the tests use.
func LoadInstanceWhitelistFromEnv() []string {
	v := os.Getenv(liveWhitelistEnvVar)
	return parseWhitelistCSV(v)
}

// parseWhitelistCSV is the pure helper underneath LoadInstanceWhitelistFromEnv,
// extracted so unit tests don't need to set env vars.
func parseWhitelistCSV(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// isInstanceWhitelisted is a pure helper for whitelist membership.
func isInstanceWhitelisted(id string, whitelist []string) bool {
	return slices.Contains(whitelist, id)
}

// requireAllWhitelisted enforces that EVERY target instance is on the
// whitelist. A single non-whitelisted ID rejects the entire batch — partial
// sends would create audit-log entries that are misleading about scope.
func requireAllWhitelisted(targets, whitelist []string) error {
	if len(targets) == 0 {
		return fmt.Errorf("at least one target instance is required")
	}
	for _, id := range targets {
		if !isInstanceWhitelisted(id, whitelist) {
			return fmt.Errorf("instance %q is not on the live-patch whitelist (have %d allowed)", id, len(whitelist))
		}
	}
	return nil
}

// cloneStringSliceMap copies map[string][]string so the SDK call's input
// can never share backing arrays with the audit-logged plan.
func cloneStringSliceMap(in map[string][]string) map[string][]string {
	if in == nil {
		return nil
	}
	out := make(map[string][]string, len(in))
	for k, v := range in {
		out[k] = append([]string(nil), v...)
	}
	return out
}

// firstOrEmpty returns the first element of a slice or "" if empty. Used
// for log triage strings that should never panic on edge cases.
func firstOrEmpty(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}
