// Package llm provides LLM client abstraction for multiple providers.
package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// Client is the interface for LLM interactions.
type Client interface {
	// Complete sends a completion request to the LLM.
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// CompleteWithTools sends a completion request with tool definitions.
	CompleteWithTools(ctx context.Context, req *CompletionRequest, tools []ToolDefinition) (*CompletionResponse, error)

	// Provider returns the provider name.
	Provider() string

	// Model returns the model name.
	Model() string
}

// CompletionRequest represents a request to the LLM.
type CompletionRequest struct {
	SystemPrompt  string    `json:"system_prompt"`
	Messages      []Message `json:"messages"`
	MaxTokens     int       `json:"max_tokens,omitempty"`
	Temperature   float64   `json:"temperature,omitempty"`
	StopSequences []string  `json:"stop_sequences,omitempty"`
}

// Message represents a conversation message.
type Message struct {
	Role    string `json:"role"` // "user", "assistant", "system"
	Content string `json:"content"`
}

// CompletionResponse represents a response from the LLM.
type CompletionResponse struct {
	Content      string        `json:"content"`
	ToolCalls    []ToolCall    `json:"tool_calls,omitempty"`
	Usage        Usage         `json:"usage"`
	StopReason   string        `json:"stop_reason"`
	FinishReason string        `json:"finish_reason"`
	Latency      time.Duration `json:"latency"`
}

// ToolCall represents a tool invocation requested by the LLM.
type ToolCall struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ToolDefinition defines a tool that the LLM can invoke.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
}

// Usage tracks token consumption.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// NewClient creates a new LLM client based on configuration.
func NewClient(cfg config.LLMConfig, log *logger.Logger) (Client, error) {
	switch cfg.Provider {
	case "anthropic":
		return newAnthropicClient(cfg, log)
	case "azure_anthropic":
		// Claude models on Azure (Microsoft Foundry)
		return newAzureAnthropicClient(cfg, log)
	case "azure_openai":
		return newAzureOpenAIClient(cfg, log)
	case "openai":
		return newOpenAIClient(cfg, log)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s (supported: anthropic, azure_anthropic, azure_openai, openai)", cfg.Provider)
	}
}
