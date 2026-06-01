// PR #19 / CONN-001 — unit tests for QueryAWSInstancesTool.
//
// Three tests using a programmable fake AWSClient (defined locally so the
// tests don't depend on the mock client used in production fallback). Pure
// in-memory; no DB, no real cloud, no env dependencies.

package tools

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeAWSClient is a programmable AWSClient for unit tests. Lets each test
// control exactly what DescribeInstances returns or fails with.
type fakeAWSClient struct {
	describeFn func(ctx context.Context, region string) ([]AWSInstance, error)
}

func (f *fakeAWSClient) DescribeInstances(ctx context.Context, region string) ([]AWSInstance, error) {
	return f.describeFn(ctx, region)
}

// TestQueryAWSInstancesTool_Success — fake client returns 2 instances; tool
// wraps them with the expected envelope (region + instance_count + instances).
func TestQueryAWSInstancesTool_Success(t *testing.T) {
	client := &fakeAWSClient{
		describeFn: func(_ context.Context, region string) ([]AWSInstance, error) {
			return []AWSInstance{
				{InstanceID: "i-aaa", InstanceType: "t3.small", State: "running", Region: region},
				{InstanceID: "i-bbb", InstanceType: "t3.medium", State: "stopped", Region: region},
			}, nil
		},
	}

	tool := NewQueryAWSInstancesTool(client)
	if tool.Risk() != RiskReadOnly {
		t.Fatalf("Risk = %q, want %q", tool.Risk(), RiskReadOnly)
	}
	if tool.RequiresApproval() {
		t.Errorf("RequiresApproval = true, want false (read-only tool)")
	}

	got, err := tool.Execute(context.Background(), map[string]any{"region": "eu-west-1"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("Execute returned %T, want map[string]any", got)
	}
	if m["region"] != "eu-west-1" {
		t.Errorf("region = %v, want eu-west-1", m["region"])
	}
	if m["instance_count"] != 2 {
		t.Errorf("instance_count = %v, want 2", m["instance_count"])
	}
}

// TestQueryAWSInstancesTool_DefaultRegion — empty region in params; tool
// uses us-east-1.
func TestQueryAWSInstancesTool_DefaultRegion(t *testing.T) {
	var sawRegion string
	client := &fakeAWSClient{
		describeFn: func(_ context.Context, region string) ([]AWSInstance, error) {
			sawRegion = region
			return []AWSInstance{}, nil
		},
	}

	tool := NewQueryAWSInstancesTool(client)
	if _, err := tool.Execute(context.Background(), map[string]any{}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if sawRegion != "us-east-1" {
		t.Errorf("default region = %q, want us-east-1", sawRegion)
	}
}

// TestQueryAWSInstancesTool_ClientError — fake client returns error; the tool
// propagates it instead of swallowing or wrapping in a success-shaped map.
func TestQueryAWSInstancesTool_ClientError(t *testing.T) {
	wantErr := errors.New("ec2 describe instances: SignatureDoesNotMatch")
	client := &fakeAWSClient{
		describeFn: func(_ context.Context, _ string) ([]AWSInstance, error) {
			return nil, wantErr
		},
	}

	tool := NewQueryAWSInstancesTool(client)
	got, err := tool.Execute(context.Background(), map[string]any{"region": "us-east-1"})
	if err == nil {
		t.Fatal("Execute: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "SignatureDoesNotMatch") {
		t.Errorf("error message = %q, want substring SignatureDoesNotMatch", err.Error())
	}
	if got != nil {
		t.Errorf("Execute returned non-nil result on error: %+v", got)
	}
}

// TestQueryAWSInstancesTool_NilClient — defensive: a misconfigured tool with
// nil client should fail loudly, not panic.
func TestQueryAWSInstancesTool_NilClient(t *testing.T) {
	tool := NewQueryAWSInstancesTool(nil)
	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Fatal("Execute with nil client: expected error, got nil")
	}
}

// TestMockAWSClient — confirm the fallback mock returns the documented
// two-instance fixture and that the instance IDs make the mock origin
// obvious to a viewer.
func TestMockAWSClient(t *testing.T) {
	c := NewMockAWSClient()
	instances, err := c.DescribeInstances(context.Background(), "us-east-1")
	if err != nil {
		t.Fatalf("DescribeInstances: %v", err)
	}
	if len(instances) != 2 {
		t.Fatalf("len(instances) = %d, want 2", len(instances))
	}
	for _, inst := range instances {
		if !strings.HasPrefix(inst.InstanceID, "i-mock-") {
			t.Errorf("instance id %q lacks i-mock- prefix; viewers won't recognise the mock", inst.InstanceID)
		}
	}
}
