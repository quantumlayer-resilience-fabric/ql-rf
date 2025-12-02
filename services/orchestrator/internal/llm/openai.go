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
	openAIAPIURL = "https://api.openai.com/v1/chat/completions"
)

// openAIClient implements the Client interface for OpenAI.
type openAIClient struct {
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	httpClient  *http.Client
	log         *logger.Logger
}

func newOpenAIClient(cfg config.LLMConfig, log *logger.Logger) (*openAIClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openAI API key is required")
	}

	model := cfg.Model
	if model == "" {
		model = "gpt-4o"
	}

	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	return &openAIClient{
		apiKey:      cfg.APIKey,
		model:       model,
		maxTokens:   maxTokens,
		temperature: cfg.Temperature,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		log: log.WithComponent("openai-client"),
	}, nil
}

func (c *openAIClient) Provider() string {
	return "openai"
}

func (c *openAIClient) Model() string {
	return c.model
}

func (c *openAIClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	return c.CompleteWithTools(ctx, req, nil)
}

func (c *openAIClient) CompleteWithTools(ctx context.Context, req *CompletionRequest, tools []ToolDefinition) (*CompletionResponse, error) {
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
	apiReq := openAIRequest{
		Model:       c.model,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Messages:    make([]openAIMessage, 0, len(req.Messages)+1),
	}

	// Add system prompt if provided
	if req.SystemPrompt != "" {
		apiReq.Messages = append(apiReq.Messages, openAIMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	// Convert messages
	for _, msg := range req.Messages {
		apiReq.Messages = append(apiReq.Messages, openAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add tools if provided
	if len(tools) > 0 {
		apiReq.Tools = make([]openAITool, 0, len(tools))
		for _, tool := range tools {
			apiReq.Tools = append(apiReq.Tools, openAITool{
				Type: "function",
				Function: openAIFunction{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.Parameters,
				},
			})
		}
	}

	// Add stop sequences if provided
	if len(req.StopSequences) > 0 {
		apiReq.Stop = req.StopSequences
	}

	c.log.Debug("sending completion request to OpenAI",
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
	httpReq, err := http.NewRequestWithContext(ctx, "POST", openAIAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

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
		var apiErr openAIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("openAI API error (%s): %s", apiErr.Error.Type, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("openAI API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var apiResp openAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Build completion response
	response := &CompletionResponse{
		Usage: Usage{
			InputTokens:  apiResp.Usage.PromptTokens,
			OutputTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:  apiResp.Usage.TotalTokens,
		},
		Latency: time.Since(start),
	}

	// Extract content and tool calls from the first choice
	if len(apiResp.Choices) > 0 {
		choice := apiResp.Choices[0]
		response.Content = choice.Message.Content
		response.StopReason = choice.FinishReason
		response.FinishReason = choice.FinishReason

		// Extract tool calls
		for _, tc := range choice.Message.ToolCalls {
			var params map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
				c.log.Warn("failed to parse tool call arguments",
					"tool", tc.Function.Name,
					"error", err,
				)
				params = make(map[string]interface{})
			}
			response.ToolCalls = append(response.ToolCalls, ToolCall{
				ID:         tc.ID,
				Name:       tc.Function.Name,
				Parameters: params,
			})
		}
	}

	c.log.Debug("received completion response from OpenAI",
		"latency", response.Latency,
		"input_tokens", response.Usage.InputTokens,
		"output_tokens", response.Usage.OutputTokens,
		"finish_reason", response.FinishReason,
		"tool_calls", len(response.ToolCalls),
	)

	return response, nil
}

// OpenAI API request/response types

type openAIRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Messages    []openAIMessage `json:"messages"`
	Tools       []openAITool    `json:"tools,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int           `json:"index"`
		Message      openAIMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type openAIError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}
