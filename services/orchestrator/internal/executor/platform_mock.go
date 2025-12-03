// Package executor implements the plan execution engine.
package executor

import (
	"context"
	"time"
)

// MockPlatformClient is a mock implementation of PlatformClient for testing.
type MockPlatformClient struct {
	ReimageInstanceFunc         func(ctx context.Context, instanceID, imageID string) error
	RebootInstanceFunc          func(ctx context.Context, instanceID string) error
	TerminateInstanceFunc       func(ctx context.Context, instanceID string) error
	GetInstanceStatusFunc       func(ctx context.Context, instanceID string) (string, error)
	WaitForInstanceStateFunc    func(ctx context.Context, instanceID, targetState string, timeout time.Duration) error
	ApplyPatchesFunc            func(ctx context.Context, instanceID string, params map[string]interface{}) error
	GetPatchStatusFunc          func(ctx context.Context, instanceID string) (string, error)
	GetPatchComplianceDataFunc  func(ctx context.Context, instanceID string) (interface{}, error)
}

// NewMockPlatformClient creates a new mock platform client with default successful implementations.
func NewMockPlatformClient() *MockPlatformClient {
	return &MockPlatformClient{
		ReimageInstanceFunc: func(ctx context.Context, instanceID, imageID string) error {
			return nil
		},
		RebootInstanceFunc: func(ctx context.Context, instanceID string) error {
			return nil
		},
		TerminateInstanceFunc: func(ctx context.Context, instanceID string) error {
			return nil
		},
		GetInstanceStatusFunc: func(ctx context.Context, instanceID string) (string, error) {
			return "running", nil
		},
		WaitForInstanceStateFunc: func(ctx context.Context, instanceID, targetState string, timeout time.Duration) error {
			return nil
		},
		ApplyPatchesFunc: func(ctx context.Context, instanceID string, params map[string]interface{}) error {
			return nil
		},
		GetPatchStatusFunc: func(ctx context.Context, instanceID string) (string, error) {
			return "COMPLIANT", nil
		},
		GetPatchComplianceDataFunc: func(ctx context.Context, instanceID string) (interface{}, error) {
			return map[string]interface{}{
				"compliance_status": "COMPLIANT",
				"installed_count":   50,
				"missing_count":     0,
				"failed_count":      0,
			}, nil
		},
	}
}

// ReimageInstance reimages an instance with a new image.
func (m *MockPlatformClient) ReimageInstance(ctx context.Context, instanceID, imageID string) error {
	if m.ReimageInstanceFunc != nil {
		return m.ReimageInstanceFunc(ctx, instanceID, imageID)
	}
	return nil
}

// RebootInstance reboots an instance.
func (m *MockPlatformClient) RebootInstance(ctx context.Context, instanceID string) error {
	if m.RebootInstanceFunc != nil {
		return m.RebootInstanceFunc(ctx, instanceID)
	}
	return nil
}

// TerminateInstance terminates an instance.
func (m *MockPlatformClient) TerminateInstance(ctx context.Context, instanceID string) error {
	if m.TerminateInstanceFunc != nil {
		return m.TerminateInstanceFunc(ctx, instanceID)
	}
	return nil
}

// GetInstanceStatus gets the current status of an instance.
func (m *MockPlatformClient) GetInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	if m.GetInstanceStatusFunc != nil {
		return m.GetInstanceStatusFunc(ctx, instanceID)
	}
	return "running", nil
}

// WaitForInstanceState waits for an instance to reach a specific state.
func (m *MockPlatformClient) WaitForInstanceState(ctx context.Context, instanceID, targetState string, timeout time.Duration) error {
	if m.WaitForInstanceStateFunc != nil {
		return m.WaitForInstanceStateFunc(ctx, instanceID, targetState, timeout)
	}
	return nil
}

// ApplyPatches applies patches to an instance using platform-native tooling.
func (m *MockPlatformClient) ApplyPatches(ctx context.Context, instanceID string, params map[string]interface{}) error {
	if m.ApplyPatchesFunc != nil {
		return m.ApplyPatchesFunc(ctx, instanceID, params)
	}
	return nil
}

// GetPatchStatus retrieves patch compliance status for an instance.
func (m *MockPlatformClient) GetPatchStatus(ctx context.Context, instanceID string) (string, error) {
	if m.GetPatchStatusFunc != nil {
		return m.GetPatchStatusFunc(ctx, instanceID)
	}
	return "COMPLIANT", nil
}

// GetPatchComplianceData retrieves detailed patch compliance data.
func (m *MockPlatformClient) GetPatchComplianceData(ctx context.Context, instanceID string) (interface{}, error) {
	if m.GetPatchComplianceDataFunc != nil {
		return m.GetPatchComplianceDataFunc(ctx, instanceID)
	}
	return map[string]interface{}{
		"compliance_status": "COMPLIANT",
		"installed_count":   50,
		"missing_count":     0,
		"failed_count":      0,
	}, nil
}
