// Package middleware provides HTTP middleware for the orchestrator service.
package middleware

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Span names for orchestrator operations.
const (
	SpanIntentParse    = "ai-orchestrator.intent_parse"
	SpanAgentExecute   = "ai-orchestrator.agent_execute"
	SpanToolCall       = "ai-orchestrator.tool_call"
	SpanLLMComplete    = "llm.complete"
	SpanValidation     = "ai-orchestrator.validation"
	SpanPlanGeneration = "ai-orchestrator.plan_generation"
	SpanExecution      = "ai-orchestrator.execution"
)

// AgentSpanOptions returns common span options for agent execution.
func AgentSpanOptions(agentName, taskID string) []trace.SpanStartOption {
	return []trace.SpanStartOption{
		trace.WithAttributes(
			attribute.String("agent.name", agentName),
			attribute.String("task.id", taskID),
		),
	}
}

// ToolSpanOptions returns common span options for tool invocation.
func ToolSpanOptions(toolName string, params map[string]interface{}) []trace.SpanStartOption {
	attrs := []attribute.KeyValue{
		attribute.String("tool.name", toolName),
	}
	// Add select param attributes (avoid large objects)
	if orgID, ok := params["org_id"].(string); ok {
		attrs = append(attrs, attribute.String("tool.org_id", orgID))
	}
	return []trace.SpanStartOption{
		trace.WithAttributes(attrs...),
	}
}

// LLMSpanOptions returns common span options for LLM calls.
func LLMSpanOptions(provider, model string) []trace.SpanStartOption {
	return []trace.SpanStartOption{
		trace.WithAttributes(
			attribute.String("llm.provider", provider),
			attribute.String("llm.model", model),
		),
	}
}

// StartAgentSpan starts a span for agent execution.
func StartAgentSpan(ctx context.Context, agentName, taskID string) (context.Context, trace.Span) {
	return StartSpan(ctx, SpanAgentExecute, AgentSpanOptions(agentName, taskID)...)
}

// StartToolSpan starts a span for tool invocation.
func StartToolSpan(ctx context.Context, toolName string, params map[string]interface{}) (context.Context, trace.Span) {
	return StartSpan(ctx, SpanToolCall, ToolSpanOptions(toolName, params)...)
}

// StartLLMSpan starts a span for LLM completion.
func StartLLMSpan(ctx context.Context, provider, model string) (context.Context, trace.Span) {
	return StartSpan(ctx, SpanLLMComplete, LLMSpanOptions(provider, model)...)
}

// RecordLLMUsage records token usage on the current span.
func RecordLLMUsage(ctx context.Context, inputTokens, outputTokens, totalTokens int) {
	span := SpanFromContext(ctx)
	span.SetAttributes(
		attribute.Int("llm.input_tokens", inputTokens),
		attribute.Int("llm.output_tokens", outputTokens),
		attribute.Int("llm.total_tokens", totalTokens),
	)
}

// RecordAgentResult records agent execution result on the current span.
func RecordAgentResult(ctx context.Context, status string, affectedAssets int, riskLevel string) {
	span := SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("agent.status", status),
		attribute.Int("agent.affected_assets", affectedAssets),
		attribute.String("agent.risk_level", riskLevel),
	)
}

// RecordToolResult records tool execution result on the current span.
func RecordToolResult(ctx context.Context, success bool, errorCode string) {
	span := SpanFromContext(ctx)
	span.SetAttributes(
		attribute.Bool("tool.success", success),
	)
	if errorCode != "" {
		span.SetAttributes(attribute.String("tool.error_code", errorCode))
	}
}

// RecordValidationResult records validation result on the current span.
func RecordValidationResult(ctx context.Context, valid bool, errors []string) {
	span := SpanFromContext(ctx)
	span.SetAttributes(
		attribute.Bool("validation.valid", valid),
		attribute.Int("validation.error_count", len(errors)),
	)
}
