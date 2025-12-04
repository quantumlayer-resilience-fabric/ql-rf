package llm

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionRequest_Fields(t *testing.T) {
	req := CompletionRequest{
		SystemPrompt: "You are a helpful assistant.",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
		},
		MaxTokens:     1024,
		Temperature:   0.7,
		StopSequences: []string{"END", "STOP"},
	}

	assert.Equal(t, "You are a helpful assistant.", req.SystemPrompt)
	assert.Len(t, req.Messages, 2)
	assert.Equal(t, 1024, req.MaxTokens)
	assert.Equal(t, 0.7, req.Temperature)
	assert.Len(t, req.StopSequences, 2)
}

func TestCompletionRequest_JSONSerialization(t *testing.T) {
	req := CompletionRequest{
		SystemPrompt: "Test system prompt",
		Messages: []Message{
			{Role: "user", Content: "Test message"},
		},
		MaxTokens:   512,
		Temperature: 0.5,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed CompletionRequest
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, req.SystemPrompt, parsed.SystemPrompt)
	assert.Len(t, parsed.Messages, 1)
	assert.Equal(t, req.MaxTokens, parsed.MaxTokens)
}

func TestMessage_Fields(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		content string
	}{
		{"user message", "user", "Hello, how are you?"},
		{"assistant message", "assistant", "I'm doing well, thank you!"},
		{"system message", "system", "You are a helpful assistant."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := Message{Role: tt.role, Content: tt.content}
			assert.Equal(t, tt.role, msg.Role)
			assert.Equal(t, tt.content, msg.Content)
		})
	}
}

func TestMessage_JSONSerialization(t *testing.T) {
	msg := Message{Role: "user", Content: "Hello"}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var parsed Message
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, msg.Role, parsed.Role)
	assert.Equal(t, msg.Content, parsed.Content)
}

func TestCompletionResponse_Fields(t *testing.T) {
	resp := CompletionResponse{
		Content: "This is the response content.",
		ToolCalls: []ToolCall{
			{
				ID:         "call-123",
				Name:       "query_assets",
				Parameters: map[string]interface{}{"platform": "aws"},
			},
		},
		Usage: Usage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
		StopReason:   "end_turn",
		FinishReason: "end_turn",
		Latency:      500 * time.Millisecond,
	}

	assert.Equal(t, "This is the response content.", resp.Content)
	assert.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "call-123", resp.ToolCalls[0].ID)
	assert.Equal(t, 100, resp.Usage.InputTokens)
	assert.Equal(t, 50, resp.Usage.OutputTokens)
	assert.Equal(t, 150, resp.Usage.TotalTokens)
	assert.Equal(t, "end_turn", resp.StopReason)
	assert.Equal(t, 500*time.Millisecond, resp.Latency)
}

func TestCompletionResponse_JSONSerialization(t *testing.T) {
	resp := CompletionResponse{
		Content: "Test response",
		Usage: Usage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
		StopReason: "end_turn",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed CompletionResponse
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, resp.Content, parsed.Content)
	assert.Equal(t, resp.Usage.InputTokens, parsed.Usage.InputTokens)
}

func TestToolCall_Fields(t *testing.T) {
	tc := ToolCall{
		ID:   "toolu_123abc",
		Name: "query_assets",
		Parameters: map[string]interface{}{
			"platform":    "aws",
			"environment": "production",
			"limit":       float64(100),
		},
	}

	assert.Equal(t, "toolu_123abc", tc.ID)
	assert.Equal(t, "query_assets", tc.Name)
	assert.Equal(t, "aws", tc.Parameters["platform"])
	assert.Equal(t, "production", tc.Parameters["environment"])
	assert.Equal(t, float64(100), tc.Parameters["limit"])
}

func TestToolCall_JSONSerialization(t *testing.T) {
	tc := ToolCall{
		ID:   "call-456",
		Name: "get_drift_status",
		Parameters: map[string]interface{}{
			"org_id": "org-123",
		},
	}

	data, err := json.Marshal(tc)
	require.NoError(t, err)

	var parsed ToolCall
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, tc.ID, parsed.ID)
	assert.Equal(t, tc.Name, parsed.Name)
	assert.Equal(t, "org-123", parsed.Parameters["org_id"])
}

func TestToolDefinition_Fields(t *testing.T) {
	td := ToolDefinition{
		Name:        "query_assets",
		Description: "Query infrastructure assets with filters",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"platform": map[string]interface{}{
					"type":        "string",
					"description": "Cloud platform",
				},
			},
			"required": []interface{}{"platform"},
		},
	}

	assert.Equal(t, "query_assets", td.Name)
	assert.Equal(t, "Query infrastructure assets with filters", td.Description)
	assert.NotNil(t, td.Parameters)
	assert.Equal(t, "object", td.Parameters["type"])
}

func TestToolDefinition_JSONSerialization(t *testing.T) {
	td := ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: map[string]interface{}{
			"type": "object",
		},
	}

	data, err := json.Marshal(td)
	require.NoError(t, err)

	var parsed ToolDefinition
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, td.Name, parsed.Name)
	assert.Equal(t, td.Description, parsed.Description)
}

func TestUsage_Fields(t *testing.T) {
	usage := Usage{
		InputTokens:  1500,
		OutputTokens: 500,
		TotalTokens:  2000,
	}

	assert.Equal(t, 1500, usage.InputTokens)
	assert.Equal(t, 500, usage.OutputTokens)
	assert.Equal(t, 2000, usage.TotalTokens)
}

func TestUsage_JSONSerialization(t *testing.T) {
	usage := Usage{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
	}

	data, err := json.Marshal(usage)
	require.NoError(t, err)

	var parsed Usage
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, usage.InputTokens, parsed.InputTokens)
	assert.Equal(t, usage.OutputTokens, parsed.OutputTokens)
	assert.Equal(t, usage.TotalTokens, parsed.TotalTokens)
}

func TestUsage_Calculation(t *testing.T) {
	tests := []struct {
		name         string
		inputTokens  int
		outputTokens int
		expectedTotal int
	}{
		{"small usage", 10, 5, 15},
		{"medium usage", 1000, 500, 1500},
		{"large usage", 100000, 50000, 150000},
		{"zero output", 100, 0, 100},
		{"zero input", 0, 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage := Usage{
				InputTokens:  tt.inputTokens,
				OutputTokens: tt.outputTokens,
				TotalTokens:  tt.inputTokens + tt.outputTokens,
			}
			assert.Equal(t, tt.expectedTotal, usage.TotalTokens)
		})
	}
}

func TestCompletionResponse_EmptyToolCalls(t *testing.T) {
	resp := CompletionResponse{
		Content:   "Just text response",
		ToolCalls: nil,
	}

	assert.Empty(t, resp.ToolCalls)
	assert.Len(t, resp.ToolCalls, 0)
}

func TestCompletionResponse_MultipleToolCalls(t *testing.T) {
	resp := CompletionResponse{
		Content: "",
		ToolCalls: []ToolCall{
			{ID: "call-1", Name: "query_assets", Parameters: map[string]interface{}{"platform": "aws"}},
			{ID: "call-2", Name: "get_drift_status", Parameters: map[string]interface{}{"org_id": "org-1"}},
			{ID: "call-3", Name: "analyze_compliance", Parameters: map[string]interface{}{"framework": "CIS"}},
		},
	}

	assert.Len(t, resp.ToolCalls, 3)
	assert.Equal(t, "query_assets", resp.ToolCalls[0].Name)
	assert.Equal(t, "get_drift_status", resp.ToolCalls[1].Name)
	assert.Equal(t, "analyze_compliance", resp.ToolCalls[2].Name)
}

func TestCompletionRequest_EmptyMessages(t *testing.T) {
	req := CompletionRequest{
		SystemPrompt: "System prompt only",
		Messages:     []Message{},
	}

	assert.Empty(t, req.Messages)
	assert.NotEmpty(t, req.SystemPrompt)
}

func TestCompletionRequest_Defaults(t *testing.T) {
	// Test that zero values work correctly
	req := CompletionRequest{}

	assert.Equal(t, "", req.SystemPrompt)
	assert.Empty(t, req.Messages)
	assert.Equal(t, 0, req.MaxTokens)
	assert.Equal(t, 0.0, req.Temperature)
	assert.Empty(t, req.StopSequences)
}

func TestToolDefinition_ComplexSchema(t *testing.T) {
	td := ToolDefinition{
		Name:        "complex_tool",
		Description: "A tool with complex schema",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"filters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"platform": map[string]interface{}{
							"type": "string",
							"enum": []interface{}{"aws", "azure", "gcp"},
						},
						"environment": map[string]interface{}{
							"type": "string",
							"enum": []interface{}{"production", "staging", "development"},
						},
					},
				},
				"options": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"limit": map[string]interface{}{
							"type":    "integer",
							"minimum": 1,
							"maximum": 1000,
						},
						"offset": map[string]interface{}{
							"type":    "integer",
							"minimum": 0,
						},
					},
				},
			},
			"required": []interface{}{"filters"},
		},
	}

	// Verify nested properties
	props := td.Parameters["properties"].(map[string]interface{})
	assert.Contains(t, props, "filters")
	assert.Contains(t, props, "options")

	// Verify required array
	required := td.Parameters["required"].([]interface{})
	assert.Contains(t, required, "filters")
}

func TestLatency_Duration(t *testing.T) {
	tests := []struct {
		name     string
		latency  time.Duration
		expected string
	}{
		{"milliseconds", 500 * time.Millisecond, "500ms"},
		{"seconds", 2 * time.Second, "2s"},
		{"mixed", 2500 * time.Millisecond, "2.5s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := CompletionResponse{Latency: tt.latency}
			assert.Equal(t, tt.latency, resp.Latency)
		})
	}
}

func TestStopReasons(t *testing.T) {
	stopReasons := []string{
		"end_turn",
		"max_tokens",
		"stop_sequence",
		"tool_use",
	}

	for _, reason := range stopReasons {
		t.Run(reason, func(t *testing.T) {
			resp := CompletionResponse{StopReason: reason}
			assert.Equal(t, reason, resp.StopReason)
		})
	}
}
