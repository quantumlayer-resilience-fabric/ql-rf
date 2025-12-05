// Package telemetry provides unified OpenTelemetry instrumentation for all QL-RF services.
// It supports traces, metrics, and logging correlation.
package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds configuration for telemetry.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Enabled        bool

	// Exporter configuration
	ExporterType   ExporterType // stdout, otlp_grpc, otlp_http
	OTLPEndpoint   string       // OTLP collector endpoint
	OTLPInsecure   bool         // Use insecure connection (for dev)

	// Sampling
	SampleRate float64 // 0.0 to 1.0

	// Resource attributes
	Attributes map[string]string
}

// ExporterType defines the type of trace exporter.
type ExporterType string

const (
	ExporterStdout   ExporterType = "stdout"
	ExporterOTLPGRPC ExporterType = "otlp_grpc"
	ExporterOTLPHTTP ExporterType = "otlp_http"
)

// Provider wraps the OpenTelemetry TracerProvider.
type Provider struct {
	cfg      *Config
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

// DefaultConfig returns default telemetry configuration.
func DefaultConfig() *Config {
	return &Config{
		ServiceName:    "ql-rf",
		ServiceVersion: "0.1.0",
		Environment:    os.Getenv("RF_ENV"),
		Enabled:        true,
		ExporterType:   ExporterStdout,
		SampleRate:     1.0,
		Attributes:     make(map[string]string),
	}
}

// NewProvider creates a new telemetry provider.
func NewProvider(cfg *Config) (*Provider, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if !cfg.Enabled {
		return &Provider{
			cfg:    cfg,
			tracer: otel.Tracer(cfg.ServiceName),
		}, nil
	}

	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Add custom attributes
	for k, v := range cfg.Attributes {
		res, _ = resource.Merge(res, resource.NewWithAttributes(
			semconv.SchemaURL,
			attribute.String(k, v),
		))
	}

	// Create exporter
	exporter, err := createExporter(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create sampler
	var sampler sdktrace.Sampler
	if cfg.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.SampleRate <= 0.0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(cfg.SampleRate)
	}

	// Create TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global TracerProvider
	otel.SetTracerProvider(tp)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Provider{
		cfg:      cfg,
		provider: tp,
		tracer:   tp.Tracer(cfg.ServiceName),
	}, nil
}

func createExporter(cfg *Config) (sdktrace.SpanExporter, error) {
	ctx := context.Background()

	switch cfg.ExporterType {
	case ExporterOTLPGRPC:
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		}
		if cfg.OTLPInsecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		return otlptracegrpc.New(ctx, opts...)

	case ExporterOTLPHTTP:
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
		}
		if cfg.OTLPInsecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		return otlptracehttp.New(ctx, opts...)

	case ExporterStdout:
		fallthrough
	default:
		return stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)
	}
}

// Shutdown gracefully shuts down the telemetry provider.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.provider != nil {
		return p.provider.Shutdown(ctx)
	}
	return nil
}

// Tracer returns the configured tracer.
func (p *Provider) Tracer() trace.Tracer {
	return p.tracer
}

// StartSpan starts a new span.
func (p *Provider) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return p.tracer.Start(ctx, name, opts...)
}

// Span represents a traced operation.
type Span struct {
	trace.Span
	ctx context.Context
}

// StartSpan is a convenience function to start a span.
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, *Span) {
	ctx, span := otel.Tracer("").Start(ctx, name, opts...)
	return ctx, &Span{Span: span, ctx: ctx}
}

// SetAttribute sets an attribute on the span.
func (s *Span) SetAttribute(key string, value interface{}) {
	switch v := value.(type) {
	case string:
		s.SetAttributes(attribute.String(key, v))
	case int:
		s.SetAttributes(attribute.Int(key, v))
	case int64:
		s.SetAttributes(attribute.Int64(key, v))
	case float64:
		s.SetAttributes(attribute.Float64(key, v))
	case bool:
		s.SetAttributes(attribute.Bool(key, v))
	default:
		s.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}

// SetError records an error on the span.
func (s *Span) SetError(err error) {
	s.RecordError(err)
	s.SetStatus(codes.Error, err.Error())
}

// SetOK marks the span as successful.
func (s *Span) SetOK() {
	s.SetStatus(codes.Ok, "")
}

// HTTPMiddleware returns middleware that traces HTTP requests.
func HTTPMiddleware(serviceName string) func(next http.Handler) http.Handler {
	tracer := otel.Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract trace context from incoming headers
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// Start span
			spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(r.Method),
					semconv.URLFull(r.URL.String()),
					semconv.HTTPRouteKey.String(r.URL.Path),
					semconv.UserAgentOriginal(r.UserAgent()),
					semconv.ServerAddress(r.Host),
				),
			)
			defer span.End()

			// Wrap response writer
			rw := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

			// Serve request
			next.ServeHTTP(rw, r.WithContext(ctx))

			// Record response
			span.SetAttributes(semconv.HTTPResponseStatusCode(rw.statusCode))

			if rw.statusCode >= 400 {
				span.SetStatus(codes.Error, http.StatusText(rw.statusCode))
			} else {
				span.SetStatus(codes.Ok, "")
			}
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

// DatabaseSpan starts a span for database operations.
func DatabaseSpan(ctx context.Context, operation, statement string) (context.Context, *Span) {
	ctx, span := StartSpan(ctx, "db."+operation,
		trace.WithSpanKind(trace.SpanKindClient),
	)
	span.SetAttributes(
		semconv.DBSystemKey.String("postgresql"),
		semconv.DBOperationKey.String(operation),
		semconv.DBStatementKey.String(statement),
	)
	return ctx, span
}

// HTTPClientSpan starts a span for outgoing HTTP requests.
func HTTPClientSpan(ctx context.Context, method, url string) (context.Context, *Span) {
	ctx, span := StartSpan(ctx, fmt.Sprintf("HTTP %s", method),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	span.SetAttributes(
		semconv.HTTPRequestMethodKey.String(method),
		semconv.URLFull(url),
	)
	return ctx, span
}

// LLMSpan starts a span for LLM API calls.
func LLMSpan(ctx context.Context, provider, model, operation string) (context.Context, *Span) {
	ctx, span := StartSpan(ctx, fmt.Sprintf("llm.%s", operation),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	span.SetAttributes(
		attribute.String("llm.provider", provider),
		attribute.String("llm.model", model),
		attribute.String("llm.operation", operation),
	)
	return ctx, span
}

// RecordLLMUsage records LLM usage metrics on a span.
func RecordLLMUsage(span *Span, inputTokens, outputTokens int, latencyMs int64) {
	span.SetAttributes(
		attribute.Int("llm.input_tokens", inputTokens),
		attribute.Int("llm.output_tokens", outputTokens),
		attribute.Int("llm.total_tokens", inputTokens+outputTokens),
		attribute.Int64("llm.latency_ms", latencyMs),
	)
}

// WorkflowSpan starts a span for workflow operations.
func WorkflowSpan(ctx context.Context, workflowType, workflowID string) (context.Context, *Span) {
	ctx, span := StartSpan(ctx, fmt.Sprintf("workflow.%s", workflowType))
	span.SetAttributes(
		attribute.String("workflow.type", workflowType),
		attribute.String("workflow.id", workflowID),
	)
	return ctx, span
}

// ConnectorSpan starts a span for cloud connector operations.
func ConnectorSpan(ctx context.Context, platform, operation string) (context.Context, *Span) {
	ctx, span := StartSpan(ctx, fmt.Sprintf("connector.%s.%s", platform, operation),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	span.SetAttributes(
		attribute.String("cloud.platform", platform),
		attribute.String("connector.operation", operation),
	)
	return ctx, span
}

// GetTraceID returns the trace ID from context.
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID returns the span ID from context.
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// InjectHTTPHeaders injects trace context into HTTP headers for outgoing requests.
func InjectHTTPHeaders(ctx context.Context, headers http.Header) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(headers))
}

// ExtractHTTPHeaders extracts trace context from incoming HTTP headers.
func ExtractHTTPHeaders(ctx context.Context, headers http.Header) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(headers))
}

// Timed is a helper to measure function duration and add it to the span.
func Timed(span *Span) func() {
	start := time.Now()
	return func() {
		span.SetAttribute("duration_ms", time.Since(start).Milliseconds())
	}
}
