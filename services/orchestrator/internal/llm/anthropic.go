// Package llm provides LLM client implementations.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

const (
	anthropicAPIURL     = "https://api.anthropic.com/v1/messages"
	anthropicAPIVersion = "2023-06-01"
)

// anthropicClient implements the Client interface for Anthropic Claude.
type anthropicClient struct {
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	httpClient  *http.Client
	log         *logger.Logger
}

func newAnthropicClient(cfg config.LLMConfig, log *logger.Logger) (*anthropicClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("anthropic API key is required")
	}

	model := cfg.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	return &anthropicClient{
		apiKey:      cfg.APIKey,
		model:       model,
		maxTokens:   maxTokens,
		temperature: cfg.Temperature,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // LLM calls can take a while
		},
		log: log.WithComponent("anthropic-client"),
	}, nil
}

func (c *anthropicClient) Provider() string {
	return "anthropic"
}

func (c *anthropicClient) Model() string {
	return c.model
}

func (c *anthropicClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	return c.CompleteWithTools(ctx, req, nil)
}

func (c *anthropicClient) CompleteWithTools(ctx context.Context, req *CompletionRequest, tools []ToolDefinition) (*CompletionResponse, error) {
	start := time.Now()

	// Apply defaults
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.maxTokens
	}
	temperature := req.Temperature
	if temperature == 0 {
		temperature = c.temperature
	}

	// Build the API request
	apiReq := anthropicRequest{
		Model:       c.model,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Messages:    make([]anthropicMessage, 0, len(req.Messages)),
	}

	// Add system prompt if provided
	if req.SystemPrompt != "" {
		apiReq.System = req.SystemPrompt
	}

	// Convert messages
	for _, msg := range req.Messages {
		apiReq.Messages = append(apiReq.Messages, anthropicMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add tools if provided
	if len(tools) > 0 {
		apiReq.Tools = make([]anthropicTool, 0, len(tools))
		for _, tool := range tools {
			apiReq.Tools = append(apiReq.Tools, anthropicTool{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tool.Parameters,
			})
		}
	}

	// Add stop sequences if provided
	if len(req.StopSequences) > 0 {
		apiReq.StopSequences = req.StopSequences
	}

	c.log.Debug("sending completion request to Anthropic",
		"model", c.model,
		"max_tokens", maxTokens,
		"temperature", temperature,
		"message_count", len(req.Messages),
		"tool_count", len(tools),
	)

	// Marshal request
	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var apiErr anthropicError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("anthropic API error (%s): %s", apiErr.Error.Type, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("anthropic API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Build completion response
	response := &CompletionResponse{
		Usage: Usage{
			InputTokens:  apiResp.Usage.InputTokens,
			OutputTokens: apiResp.Usage.OutputTokens,
			TotalTokens:  apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
		},
		StopReason:   apiResp.StopReason,
		FinishReason: apiResp.StopReason,
		Latency:      time.Since(start),
	}

	// Extract content and tool calls
	for _, block := range apiResp.Content {
		switch block.Type {
		case "text":
			response.Content = block.Text
		case "tool_use":
			response.ToolCalls = append(response.ToolCalls, ToolCall{
				ID:         block.ID,
				Name:       block.Name,
				Parameters: block.Input,
			})
		}
	}

	c.log.Debug("received completion response from Anthropic",
		"latency", response.Latency,
		"input_tokens", response.Usage.InputTokens,
		"output_tokens", response.Usage.OutputTokens,
		"stop_reason", response.StopReason,
		"tool_calls", len(response.ToolCalls),
	)

	return response, nil
}

// Anthropic API request/response types

type anthropicRequest struct {
	Model         string             `json:"model"`
	MaxTokens     int                `json:"max_tokens"`
	Temperature   float64            `json:"temperature,omitempty"`
	System        string             `json:"system,omitempty"`
	Messages      []anthropicMessage `json:"messages"`
	Tools         []anthropicTool    `json:"tools,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type anthropicResponse struct {
	ID           string                   `json:"id"`
	Type         string                   `json:"type"`
	Role         string                   `json:"role"`
	Content      []anthropicContentBlock  `json:"content"`
	Model        string                   `json:"model"`
	StopReason   string                   `json:"stop_reason"`
	StopSequence string                   `json:"stop_sequence,omitempty"`
	Usage        anthropicUsage           `json:"usage"`
}

type anthropicContentBlock struct {
	Type  string                 `json:"type"`
	Text  string                 `json:"text,omitempty"`
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicError struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}
