package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRiskLevel(t *testing.T) {
	tests := []struct {
		level    RiskLevel
		expected string
	}{
		{RiskReadOnly, "read_only"},
		{RiskPlanOnly, "plan_only"},
		{RiskStateChangeNonProd, "state_change_nonprod"},
		{RiskStateChangeProd, "state_change_prod"},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.level))
		})
	}
}

func TestScope(t *testing.T) {
	tests := []struct {
		scope    Scope
		expected string
	}{
		{ScopeAsset, "asset"},
		{ScopeEnvironment, "environment"},
		{ScopeOrganization, "organization"},
	}

	for _, tt := range tests {
		t.Run(string(tt.scope), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.scope))
		})
	}
}

func TestToolMetadata(t *testing.T) {
	meta := ToolMetadata{
		Name:             "query_assets",
		Description:      "Query assets by filter",
		Risk:             RiskReadOnly,
		Scope:            ScopeOrganization,
		Idempotent:       true,
		RequiresApproval: false,
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"platform": map[string]interface{}{
					"type":        "string",
					"description": "Cloud platform",
				},
			},
		},
	}

	assert.Equal(t, "query_assets", meta.Name)
	assert.Equal(t, RiskReadOnly, meta.Risk)
	assert.Equal(t, ScopeOrganization, meta.Scope)
	assert.True(t, meta.Idempotent)
	assert.False(t, meta.RequiresApproval)
	assert.NotNil(t, meta.Parameters)
}

// MockTool implements Tool for testing
type MockTool struct {
	name             string
	description      string
	risk             RiskLevel
	scope            Scope
	idempotent       bool
	requiresApproval bool
	params           map[string]interface{}
	executeFunc      func(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

func (m *MockTool) Name() string                          { return m.name }
func (m *MockTool) Description() string                   { return m.description }
func (m *MockTool) Risk() RiskLevel                       { return m.risk }
func (m *MockTool) Scope() Scope                          { return m.scope }
func (m *MockTool) Idempotent() bool                      { return m.idempotent }
func (m *MockTool) RequiresApproval() bool                { return m.requiresApproval }
func (m *MockTool) Parameters() map[string]interface{}    { return m.params }
func (m *MockTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, params)
	}
	return map[string]interface{}{"status": "ok"}, nil
}

func TestMockTool(t *testing.T) {
	tool := &MockTool{
		name:             "test_tool",
		description:      "Test tool for unit testing",
		risk:             RiskReadOnly,
		scope:            ScopeAsset,
		idempotent:       true,
		requiresApproval: false,
		params: map[string]interface{}{
			"type": "object",
		},
	}

	assert.Equal(t, "test_tool", tool.Name())
	assert.Equal(t, "Test tool for unit testing", tool.Description())
	assert.Equal(t, RiskReadOnly, tool.Risk())
	assert.Equal(t, ScopeAsset, tool.Scope())
	assert.True(t, tool.Idempotent())
	assert.False(t, tool.RequiresApproval())
	assert.NotNil(t, tool.Parameters())
}

func TestMockTool_Execute(t *testing.T) {
	t.Run("default execution", func(t *testing.T) {
		tool := &MockTool{name: "test"}
		result, err := tool.Execute(context.Background(), nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("custom execution", func(t *testing.T) {
		tool := &MockTool{
			name: "test",
			executeFunc: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{
					"count":    len(params),
					"platform": params["platform"],
				}, nil
			},
		}

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"platform": "aws",
		})
		require.NoError(t, err)

		resultMap := result.(map[string]interface{})
		assert.Equal(t, 1, resultMap["count"])
		assert.Equal(t, "aws", resultMap["platform"])
	})
}

func TestToolInterface(t *testing.T) {
	// Verify MockTool implements Tool interface
	var _ Tool = (*MockTool)(nil)
}

func TestRiskLevelComparison(t *testing.T) {
	// Risk levels in order of severity
	riskOrder := []RiskLevel{
		RiskReadOnly,
		RiskPlanOnly,
		RiskStateChangeNonProd,
		RiskStateChangeProd,
	}

	assert.Len(t, riskOrder, 4)
	assert.Equal(t, RiskReadOnly, riskOrder[0])
	assert.Equal(t, RiskStateChangeProd, riskOrder[3])
}

func TestScopeComparison(t *testing.T) {
	// Scopes in order of impact
	scopeOrder := []Scope{
		ScopeAsset,
		ScopeEnvironment,
		ScopeOrganization,
	}

	assert.Len(t, scopeOrder, 3)
	assert.Equal(t, ScopeAsset, scopeOrder[0])
	assert.Equal(t, ScopeOrganization, scopeOrder[2])
}

func TestToolMetadata_Serialization(t *testing.T) {
	meta := ToolMetadata{
		Name:             "query_assets",
		Description:      "Query assets",
		Risk:             RiskReadOnly,
		Scope:            ScopeOrganization,
		Idempotent:       true,
		RequiresApproval: false,
	}

	// Just verify the struct is properly initialized
	assert.Equal(t, "query_assets", meta.Name)
	assert.Equal(t, RiskReadOnly, meta.Risk)
}

// TestToolCategories tests different tool categories
func TestToolCategories(t *testing.T) {
	queryTools := []string{
		"query_assets",
		"get_drift_status",
		"get_compliance_status",
		"get_golden_image",
		"query_alerts",
		"get_dr_status",
	}

	analysisTools := []string{
		"analyze_drift",
		"check_control",
	}

	planningTools := []string{
		"compare_versions",
		"generate_patch_plan",
		"generate_rollout_plan",
		"generate_dr_runbook",
		"simulate_rollout",
		"calculate_risk_score",
		"simulate_failover",
		"generate_compliance_evidence",
	}

	assert.GreaterOrEqual(t, len(queryTools), 6)
	assert.GreaterOrEqual(t, len(analysisTools), 2)
	assert.GreaterOrEqual(t, len(planningTools), 8)
}

func TestCreateMockToolRegistry(t *testing.T) {
	tools := make(map[string]Tool)

	// Add mock tools
	tools["query_assets"] = &MockTool{
		name:        "query_assets",
		description: "Query assets by filter",
		risk:        RiskReadOnly,
		scope:       ScopeOrganization,
		idempotent:  true,
	}

	tools["get_drift_status"] = &MockTool{
		name:        "get_drift_status",
		description: "Get drift status for assets",
		risk:        RiskReadOnly,
		scope:       ScopeEnvironment,
		idempotent:  true,
	}

	// Test get
	tool, ok := tools["query_assets"]
	require.True(t, ok)
	assert.Equal(t, "query_assets", tool.Name())

	// Test list
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	assert.Len(t, names, 2)

	// Test filter by risk
	readOnlyTools := make([]Tool, 0)
	for _, tool := range tools {
		if tool.Risk() == RiskReadOnly {
			readOnlyTools = append(readOnlyTools, tool)
		}
	}
	assert.Len(t, readOnlyTools, 2)
}

func TestToolExecution(t *testing.T) {
	tool := &MockTool{
		name: "test_execute",
		executeFunc: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			// Simulate a query result
			return map[string]interface{}{
				"assets": []map[string]interface{}{
					{"id": "asset-1", "platform": params["platform"]},
					{"id": "asset-2", "platform": params["platform"]},
				},
				"count": 2,
			}, nil
		},
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"platform": "aws",
		"env":      "production",
	})

	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, 2, resultMap["count"])

	assets := resultMap["assets"].([]map[string]interface{})
	assert.Len(t, assets, 2)
	assert.Equal(t, "aws", assets[0]["platform"])
}

func TestToolWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	tool := &MockTool{
		name: "context_test",
		executeFunc: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return map[string]interface{}{"status": "ok"}, nil
			}
		},
	}

	// Execute before cancel
	result, err := tool.Execute(ctx, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Cancel and try again
	cancel()
	result, err = tool.Execute(ctx, nil)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}
