// Package llm provides LLM client implementations.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

const (
	// Azure Anthropic (Microsoft Foundry) uses the same API version as direct Anthropic
	azureAnthropicAPIVersion = "2023-06-01"
)

// azureAnthropicClient implements the Client interface for Anthropic Claude on Azure (Microsoft Foundry).
// This is different from Azure OpenAI - it's Claude models hosted on Azure infrastructure.
type azureAnthropicClient struct {
	endpoint    string // Base URL: https://<resource>.services.ai.azure.com/anthropic
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	httpClient  *http.Client
	log         *logger.Logger
}

func newAzureAnthropicClient(cfg config.LLMConfig, log *logger.Logger) (*azureAnthropicClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("azure Anthropic API key is required")
	}
	if cfg.AzureAnthropicEndpoint == "" {
		return nil, fmt.Errorf("azure Anthropic endpoint is required")
	}

	model := cfg.Model
	if model == "" {
		model = "claude-sonnet-4-5"
	}

	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	// Ensure endpoint doesn't have trailing slash or /v1/messages
	endpoint := strings.TrimSuffix(cfg.AzureAnthropicEndpoint, "/")
	endpoint = strings.TrimSuffix(endpoint, "/v1/messages")
	endpoint = strings.TrimSuffix(endpoint, "/anthropic")

	return &azureAnthropicClient{
		endpoint:    endpoint,
		apiKey:      cfg.APIKey,
		model:       model,
		maxTokens:   maxTokens,
		temperature: cfg.Temperature,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		log: log.WithComponent("azure-anthropic-client"),
	}, nil
}

func (c *azureAnthropicClient) Provider() string {
	return "azure_anthropic"
}

func (c *azureAnthropicClient) Model() string {
	return c.model
}

func (c *azureAnthropicClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	return c.CompleteWithTools(ctx, req, nil)
}

func (c *azureAnthropicClient) CompleteWithTools(ctx context.Context, req *CompletionRequest, tools []ToolDefinition) (*CompletionResponse, error) {
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

	// Build the API request (same format as direct Anthropic)
	apiReq := azureAnthropicRequest{
		Model:       c.model,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Messages:    make([]azureAnthropicMessage, 0, len(req.Messages)),
	}

	// Add system prompt if provided
	if req.SystemPrompt != "" {
		apiReq.System = req.SystemPrompt
	}

	// Convert messages
	for _, msg := range req.Messages {
		apiReq.Messages = append(apiReq.Messages, azureAnthropicMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add tools if provided
	if len(tools) > 0 {
		apiReq.Tools = make([]azureAnthropicTool, 0, len(tools))
		for _, tool := range tools {
			apiReq.Tools = append(apiReq.Tools, azureAnthropicTool{
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

	// Azure Anthropic endpoint: https://<resource>.services.ai.azure.com/anthropic/v1/messages
	url := fmt.Sprintf("%s/anthropic/v1/messages", c.endpoint)

	c.log.Debug("sending completion request to Azure Anthropic (Microsoft Foundry)",
		"endpoint", c.endpoint,
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
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers - Azure Anthropic uses same headers as direct Anthropic
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", azureAnthropicAPIVersion)

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
		var apiErr azureAnthropicError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("azure Anthropic API error (%s): %s", apiErr.Error.Type, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("azure Anthropic API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response (same format as direct Anthropic)
	var apiResp azureAnthropicResponse
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

	c.log.Debug("received completion response from Azure Anthropic",
		"latency", response.Latency,
		"input_tokens", response.Usage.InputTokens,
		"output_tokens", response.Usage.OutputTokens,
		"stop_reason", response.StopReason,
		"tool_calls", len(response.ToolCalls),
	)

	return response, nil
}

// Azure Anthropic API request/response types (same as direct Anthropic)

type azureAnthropicRequest struct {
	Model         string                  `json:"model"`
	MaxTokens     int                     `json:"max_tokens"`
	Temperature   float64                 `json:"temperature,omitempty"`
	System        string                  `json:"system,omitempty"`
	Messages      []azureAnthropicMessage `json:"messages"`
	Tools         []azureAnthropicTool    `json:"tools,omitempty"`
	StopSequences []string                `json:"stop_sequences,omitempty"`
}

type azureAnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type azureAnthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type azureAnthropicResponse struct {
	ID           string                        `json:"id"`
	Type         string                        `json:"type"`
	Role         string                        `json:"role"`
	Content      []azureAnthropicContentBlock  `json:"content"`
	Model        string                        `json:"model"`
	StopReason   string                        `json:"stop_reason"`
	StopSequence string                        `json:"stop_sequence,omitempty"`
	Usage        azureAnthropicUsage           `json:"usage"`
}

type azureAnthropicContentBlock struct {
	Type  string                 `json:"type"`
	Text  string                 `json:"text,omitempty"`
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
}

type azureAnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type azureAnthropicError struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}
