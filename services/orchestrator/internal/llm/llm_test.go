package llm_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
)

// TestAzureAnthropicIntegration tests the Azure Anthropic client against the real API.
// Run with: go test -v -run TestAzureAnthropicIntegration ./services/orchestrator/internal/llm/...
// Requires environment variables:
//   - RF_LLM_AZURE_ANTHROPIC_ENDPOINT
//   - RF_LLM_API_KEY
func TestAzureAnthropicIntegration(t *testing.T) {
	endpoint := os.Getenv("RF_LLM_AZURE_ANTHROPIC_ENDPOINT")
	apiKey := os.Getenv("RF_LLM_API_KEY")

	if endpoint == "" || apiKey == "" {
		t.Skip("Skipping integration test: RF_LLM_AZURE_ANTHROPIC_ENDPOINT and RF_LLM_API_KEY not set")
	}

	log := logger.New("debug", "text")

	cfg := config.LLMConfig{
		Provider:               "azure_anthropic",
		AzureAnthropicEndpoint: endpoint,
		APIKey:                 apiKey,
		Model:                  "claude-sonnet-4-5",
		MaxTokens:              1024,
		Temperature:            0.3,
	}

	client, err := llm.NewClient(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Logf("Created Azure Anthropic client: provider=%s, model=%s", client.Provider(), client.Model())

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Test 1: Simple completion
	t.Run("SimpleCompletion", func(t *testing.T) {
		resp, err := client.Complete(ctx, &llm.CompletionRequest{
			SystemPrompt: "You are a helpful infrastructure assistant. Be concise.",
			Messages: []llm.Message{
				{Role: "user", Content: "What is configuration drift in infrastructure management? Answer in one sentence."},
			},
			MaxTokens:   256,
			Temperature: 0.3,
		})
		if err != nil {
			t.Fatalf("Completion failed: %v", err)
		}

		t.Logf("Response received:")
		t.Logf("  Content: %s", resp.Content)
		t.Logf("  Tokens: input=%d, output=%d, total=%d", resp.Usage.InputTokens, resp.Usage.OutputTokens, resp.Usage.TotalTokens)
		t.Logf("  Latency: %v", resp.Latency)
		t.Logf("  Stop reason: %s", resp.StopReason)

		if resp.Content == "" {
			t.Error("Expected non-empty response content")
		}
	})

	// Test 2: Tool use
	t.Run("ToolUse", func(t *testing.T) {
		tools := []llm.ToolDefinition{
			{
				Name:        "query_assets",
				Description: "Query infrastructure assets with filters",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"platform": map[string]interface{}{
							"type":        "string",
							"description": "Cloud platform (aws, azure, gcp)",
						},
						"environment": map[string]interface{}{
							"type":        "string",
							"description": "Environment (production, staging, development)",
						},
					},
					"required": []string{"platform"},
				},
			},
			{
				Name:        "get_drift_status",
				Description: "Get configuration drift status for assets",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"org_id": map[string]interface{}{
							"type":        "string",
							"description": "Organization ID",
						},
					},
					"required": []string{"org_id"},
				},
			},
		}

		resp, err := client.CompleteWithTools(ctx, &llm.CompletionRequest{
			SystemPrompt: "You are an infrastructure management AI. Use the available tools to help users.",
			Messages: []llm.Message{
				{Role: "user", Content: "Show me all drifted assets in AWS production environment"},
			},
			MaxTokens:   1024,
			Temperature: 0.3,
		}, tools)
		if err != nil {
			t.Fatalf("Completion with tools failed: %v", err)
		}

		t.Logf("Response received:")
		t.Logf("  Content: %s", resp.Content)
		t.Logf("  Tool calls: %d", len(resp.ToolCalls))
		for i, tc := range resp.ToolCalls {
			t.Logf("    [%d] %s: %v", i, tc.Name, tc.Parameters)
		}
		t.Logf("  Tokens: input=%d, output=%d", resp.Usage.InputTokens, resp.Usage.OutputTokens)
		t.Logf("  Stop reason: %s", resp.StopReason)

		// The model should either respond with text or request a tool call
		if resp.Content == "" && len(resp.ToolCalls) == 0 {
			t.Error("Expected either content or tool calls in response")
		}
	})
}

// TestClientCreation tests that clients are created correctly for each provider.
func TestClientCreation(t *testing.T) {
	log := logger.New("error", "text")

	tests := []struct {
		name        string
		cfg         config.LLMConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "azure_anthropic_missing_endpoint",
			cfg: config.LLMConfig{
				Provider: "azure_anthropic",
				APIKey:   "test-key",
			},
			wantErr:     true,
			errContains: "endpoint is required",
		},
		{
			name: "azure_anthropic_missing_key",
			cfg: config.LLMConfig{
				Provider:               "azure_anthropic",
				AzureAnthropicEndpoint: "https://test.services.ai.azure.com",
			},
			wantErr:     true,
			errContains: "API key is required",
		},
		{
			name: "anthropic_missing_key",
			cfg: config.LLMConfig{
				Provider: "anthropic",
			},
			wantErr:     true,
			errContains: "API key is required",
		},
		{
			name: "openai_missing_key",
			cfg: config.LLMConfig{
				Provider: "openai",
			},
			wantErr:     true,
			errContains: "API key is required",
		},
		{
			name: "azure_openai_missing_endpoint",
			cfg: config.LLMConfig{
				Provider: "azure_openai",
				APIKey:   "test-key",
			},
			wantErr:     true,
			errContains: "endpoint is required",
		},
		{
			name: "unsupported_provider",
			cfg: config.LLMConfig{
				Provider: "unsupported",
			},
			wantErr:     true,
			errContains: "unsupported LLM provider",
		},
		{
			name: "azure_anthropic_valid",
			cfg: config.LLMConfig{
				Provider:               "azure_anthropic",
				AzureAnthropicEndpoint: "https://test.services.ai.azure.com",
				APIKey:                 "test-key",
				Model:                  "claude-sonnet-4-5",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := llm.NewClient(tt.cfg, log)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errContains)
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Error("Expected non-nil client")
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchString(s, substr)))
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
