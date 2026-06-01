// PR #19 / CONN-001 — AWS tools.
//
// First real cloud-touching tool in the orchestrator. Registered only when
// an AWSClient was successfully constructed at boot (see main.go); otherwise
// the tool is absent from the registry and GET /api/v1/ai/tools doesn't list
// it. Read-only by API contract — EC2 DescribeInstances cannot modify cloud
// state. State-change AWS tools (SSM SendCommand etc.) land in a follow-up
// PR with stronger safety gates.
package tools

import (
	"context"
	"fmt"
)

// QueryAWSInstancesTool lists EC2 instances in an AWS region via a configured
// AWSClient (real or mock). Risk = read_only. Idempotent. Never modifies
// cloud state.
type QueryAWSInstancesTool struct {
	client AWSClient
}

// NewQueryAWSInstancesTool constructs the tool with a backing AWSClient. The
// caller is responsible for choosing real vs mock (see main.go).
func NewQueryAWSInstancesTool(client AWSClient) *QueryAWSInstancesTool {
	return &QueryAWSInstancesTool{client: client}
}

func (t *QueryAWSInstancesTool) Name() string {
	return "query_aws_instances"
}

func (t *QueryAWSInstancesTool) Description() string {
	return "List EC2 instances in an AWS region (read-only)."
}

func (t *QueryAWSInstancesTool) Risk() RiskLevel {
	return RiskReadOnly
}

func (t *QueryAWSInstancesTool) Scope() Scope {
	return ScopeOrganization
}

func (t *QueryAWSInstancesTool) Idempotent() bool {
	return true
}

func (t *QueryAWSInstancesTool) RequiresApproval() bool {
	return false
}

func (t *QueryAWSInstancesTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"region": map[string]any{
				"type":        "string",
				"description": "AWS region to query (defaults to us-east-1).",
				"default":     "us-east-1",
			},
		},
	}
}

// Execute calls AWSClient.DescribeInstances and wraps the result in a small
// envelope that's audit-log friendly. The tool deliberately does NOT include
// the raw EC2 instance fields verbatim — only the redacted projection from
// AWSClient (no security groups, no IAM roles, no key pair names).
func (t *QueryAWSInstancesTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	if t.client == nil {
		return nil, fmt.Errorf("query_aws_instances: client not configured")
	}

	region := defaultAWSRegion
	if v, ok := params["region"].(string); ok && v != "" {
		region = v
	}

	instances, err := t.client.DescribeInstances(ctx, region)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"region":         region,
		"instance_count": len(instances),
		"instances":      instances,
	}, nil
}

// RegisterCloudTools registers AWS-backed cloud tools on the registry. Called
// from main.go ONLY after an AWSClient was successfully constructed (real or
// mock-with-fallback). Safe to call zero or one time per registry lifetime.
//
// If the registry already has a tool with the same name, it's overwritten —
// caller is responsible for not double-registering.
func (r *Registry) RegisterCloudTools(client AWSClient) {
	if client == nil {
		r.log.Warn("RegisterCloudTools called with nil client; no AWS tools registered")
		return
	}
	r.register(NewQueryAWSInstancesTool(client))
	r.log.Info("aws tools: registered", "tools", []string{"query_aws_instances"})
}

// RegisterToolForTest is a test-only helper that adds a Tool to a Registry by
// name. The internal `register` method is unexported so callers in other
// packages (e.g., the handlers package's invoke-endpoint tests) need this
// trampoline. Marked with a name that's obviously test-only and documented
// as such so it doesn't drift into production usage.
//
// Production code should always go through NewRegistry() or the
// Register*Tools() helpers in this package.
func RegisterToolForTest(r *Registry, tool Tool) {
	r.register(tool)
}
