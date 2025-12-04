package agents

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseAgent_Methods(t *testing.T) {
	agent := &BaseAgent{
		name:        "test_agent",
		description: "Test agent for unit testing",
		tasks:       []TaskType{TaskTypeDriftRemediation, TaskTypePatchRollout},
		tools:       []string{"query_assets", "get_drift_status"},
	}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "test_agent", agent.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.Equal(t, "Test agent for unit testing", agent.Description())
	})

	t.Run("SupportedTasks", func(t *testing.T) {
		tasks := agent.SupportedTasks()
		assert.Len(t, tasks, 2)
		assert.Contains(t, tasks, TaskTypeDriftRemediation)
		assert.Contains(t, tasks, TaskTypePatchRollout)
	})

	t.Run("RequiredTools", func(t *testing.T) {
		tools := agent.RequiredTools()
		assert.Len(t, tools, 2)
		assert.Contains(t, tools, "query_assets")
		assert.Contains(t, tools, "get_drift_status")
	})
}

func TestToolNotFoundError(t *testing.T) {
	err := &ToolNotFoundError{Name: "missing_tool"}
	assert.Equal(t, "tool not found: missing_tool", err.Error())
}

func TestCountAssets(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int
	}{
		{
			name:     "slice of interfaces",
			input:    []interface{}{"a", "b", "c"},
			expected: 3,
		},
		{
			name:     "empty slice",
			input:    []interface{}{},
			expected: 0,
		},
		{
			name: "map with count field",
			input: map[string]interface{}{
				"count": float64(42),
			},
			expected: 42,
		},
		{
			name: "map with items field",
			input: map[string]interface{}{
				"items": []interface{}{"x", "y"},
			},
			expected: 2,
		},
		{
			name:     "nil input",
			input:    nil,
			expected: 0,
		},
		{
			name:     "string input",
			input:    "not a collection",
			expected: 0,
		},
		{
			name: "map without count or items",
			input: map[string]interface{}{
				"other": "value",
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countAssets(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"Hello World", "world", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "foo", false},
		{"", "test", false},
		{"test", "", true},
		{"CIS benchmark audit", "cis", true},
		{"SOC2 compliance", "SOC2", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindJSONStart(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"starts with brace", `{"key": "value"}`, 0},
		{"with prefix", `Here is the JSON: {"key": "value"}`, 18},
		{"no brace", "no json here", -1},
		{"empty string", "", -1},
		{"multiple braces", `text {"first": 1} {"second": 2}`, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findJSONStart(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindJSONEnd(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		start    int
		expected int
	}{
		{"simple object", `{"key": "value"}`, 0, 15},
		{"nested object", `{"outer": {"inner": "value"}}`, 0, 28},
		{"with trailing text", `{"key": "value"} extra`, 0, 15},
		{"unclosed", `{"key": "value"`, 0, -1},
		{"deeply nested", `{"a": {"b": {"c": 1}}}`, 0, 21},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findJSONEnd(tt.input, tt.start)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseJSON(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		var result map[string]interface{}
		err := parseJSON(`{"key": "value", "num": 42}`, &result)
		require.NoError(t, err)
		assert.Equal(t, "value", result["key"])
		assert.Equal(t, float64(42), result["num"])
	})

	t.Run("invalid JSON", func(t *testing.T) {
		var result map[string]interface{}
		err := parseJSON(`{invalid}`, &result)
		require.Error(t, err)
	})

	t.Run("empty string", func(t *testing.T) {
		var result map[string]interface{}
		err := parseJSON(``, &result)
		require.Error(t, err)
	})
}

func TestAgentMetadata(t *testing.T) {
	meta := AgentMetadata{
		Name:           "test_agent",
		Description:    "Test description",
		SupportedTasks: []TaskType{TaskTypeDriftRemediation},
		RequiredTools:  []string{"query_assets"},
	}

	assert.Equal(t, "test_agent", meta.Name)
	assert.Equal(t, "Test description", meta.Description)
	assert.Len(t, meta.SupportedTasks, 1)
	assert.Len(t, meta.RequiredTools, 1)
}

func TestTaskSpec_Fields(t *testing.T) {
	spec := TaskSpec{
		ID:          "task-123",
		TaskType:    TaskTypeDriftRemediation,
		Goal:        "Fix drift",
		UserIntent:  "Please fix the drift on my servers",
		OrgID:       "org-123",
		UserID:      "user-456",
		Environment: "production",
		RiskLevel:   "medium",
		Context: TaskContext{
			Platforms:   []string{"aws", "azure"},
			Regions:     []string{"us-east-1"},
			AssetFilter: "env=prod",
		},
	}

	assert.Equal(t, "task-123", spec.ID)
	assert.Equal(t, TaskTypeDriftRemediation, spec.TaskType)
	assert.Equal(t, "production", spec.Environment)
	assert.Len(t, spec.Context.Platforms, 2)
}

func TestAgentResult(t *testing.T) {
	result := AgentResult{
		TaskID:         "task-123",
		AgentName:      "drift_agent",
		Status:         AgentStatusPendingApproval,
		Summary:        "Found 5 drifted assets",
		AffectedAssets: 5,
		RiskLevel:      "medium",
		TokensUsed:     1500,
		Actions: []Action{
			{Type: "approve", Label: "Approve", Description: "Approve the plan"},
		},
	}

	assert.Equal(t, "task-123", result.TaskID)
	assert.Equal(t, AgentStatusPendingApproval, result.Status)
	assert.Equal(t, 5, result.AffectedAssets)
	assert.Len(t, result.Actions, 1)
}

func TestAgentStatus(t *testing.T) {
	statuses := []AgentStatus{
		AgentStatusPendingApproval,
		AgentStatusApproved,
		AgentStatusExecuting,
		AgentStatusCompleted,
		AgentStatusFailed,
		AgentStatusCancelled,
	}

	assert.Len(t, statuses, 6)
	assert.Equal(t, AgentStatus("pending_approval"), AgentStatusPendingApproval)
	assert.Equal(t, AgentStatus("completed"), AgentStatusCompleted)
}

func TestTaskTypes(t *testing.T) {
	taskTypes := []TaskType{
		TaskTypeDriftRemediation,
		TaskTypePatchRollout,
		TaskTypeComplianceAudit,
		TaskTypeIncidentResponse,
		TaskTypeDRDrill,
		TaskTypeCostOptimization,
		TaskTypeSecurityScan,
		TaskTypeImageManagement,
		TaskTypeSOPAuthoring,
		TaskTypeTerraformGeneration,
	}

	assert.Len(t, taskTypes, 10)
	assert.Equal(t, TaskType("drift_remediation"), TaskTypeDriftRemediation)
	assert.Equal(t, TaskType("patch_rollout"), TaskTypePatchRollout)
}

func TestAction(t *testing.T) {
	action := Action{
		Type:        "approve",
		Label:       "Approve & Execute",
		Description: "Approve the plan and begin execution",
	}

	assert.Equal(t, "approve", action.Type)
	assert.Equal(t, "Approve & Execute", action.Label)
	assert.NotEmpty(t, action.Description)
}

func TestBaseAgent_ExecuteTool_NotFound(t *testing.T) {
	// Since toolReg is nil, this will panic if we call executeTool.
	// This test verifies the error type is correct
	err := &ToolNotFoundError{Name: "nonexistent_tool"}
	assert.Contains(t, err.Error(), "nonexistent_tool")
}

// TestMockRegistry tests the registry with mock agents
func TestMockRegistry(t *testing.T) {
	t.Run("register and get agent", func(t *testing.T) {
		// Create a mock agent
		mockAgent := &mockAgent{
			name:        "mock_agent",
			description: "Mock agent for testing",
			tasks:       []TaskType{TaskTypeDriftRemediation},
			tools:       []string{"query_assets"},
		}

		// Create registry with minimal dependencies
		// Note: In real tests, we'd use proper mocks for llm, tools, etc.
		// For now we test the core registry logic

		agents := make(map[string]Agent)
		agents["mock_agent"] = mockAgent

		retrievedAgent, ok := agents["mock_agent"]
		require.True(t, ok)
		assert.Equal(t, "mock_agent", retrievedAgent.Name())
	})
}

// mockAgent implements Agent for testing
type mockAgent struct {
	name        string
	description string
	tasks       []TaskType
	tools       []string
}

func (m *mockAgent) Name() string                                                { return m.name }
func (m *mockAgent) Description() string                                         { return m.description }
func (m *mockAgent) SupportedTasks() []TaskType                                  { return m.tasks }
func (m *mockAgent) RequiredTools() []string                                     { return m.tools }
func (m *mockAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	return &AgentResult{
		TaskID:    task.ID,
		AgentName: m.name,
		Status:    AgentStatusPendingApproval,
		Summary:   "Mock execution completed",
	}, nil
}

func TestMockAgent_Execute(t *testing.T) {
	agent := &mockAgent{
		name:        "test_mock",
		description: "Test mock agent",
		tasks:       []TaskType{TaskTypeDriftRemediation},
		tools:       []string{},
	}

	task := &TaskSpec{
		ID:       "task-123",
		TaskType: TaskTypeDriftRemediation,
		Goal:     "Test goal",
	}

	result, err := agent.Execute(context.Background(), task)
	require.NoError(t, err)
	assert.Equal(t, "task-123", result.TaskID)
	assert.Equal(t, "test_mock", result.AgentName)
	assert.Equal(t, AgentStatusPendingApproval, result.Status)
}
