// Package middleware provides HTTP middleware for the orchestrator service.
package middleware

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	// TracerName is the name of the tracer for the orchestrator service.
	TracerName = "github.com/quantumlayerhq/ql-rf/services/orchestrator"

	// TraceIDKey is the context key for the trace ID.
	TraceIDKey ContextKey = "trace_id"
)

// TracingConfig holds configuration for the tracing middleware.
type TracingConfig struct {
	ServiceName    string
	ServiceVersion string
	Enabled        bool
}

// Tracer returns the global tracer for the orchestrator.
func Tracer() trace.Tracer {
	return otel.Tracer(TracerName)
}

// Tracing returns a middleware that adds OpenTelemetry tracing to requests.
func Tracing(cfg TracingConfig) func(next http.Handler) http.Handler {
	if !cfg.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	tracer := otel.Tracer(TracerName,
		trace.WithInstrumentationVersion(cfg.ServiceVersion),
	)
	propagator := otel.GetTextMapPropagator()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract trace context from incoming request headers
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// Start a new span for this request
			spanName := r.Method + " " + r.URL.Path
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.url", r.URL.String()),
					attribute.String("http.route", r.URL.Path),
					attribute.String("http.user_agent", r.UserAgent()),
					attribute.String("service.name", cfg.ServiceName),
				),
			)
			defer span.End()

			// Add trace ID to context for logging correlation
			traceID := span.SpanContext().TraceID().String()
			ctx = context.WithValue(ctx, TraceIDKey, traceID)

			// Create response wrapper to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Serve the request
			next.ServeHTTP(rw, r.WithContext(ctx))

			// Record response attributes
			span.SetAttributes(
				attribute.Int("http.status_code", rw.statusCode),
			)

			// Mark span as error if status code >= 400
			if rw.statusCode >= 400 {
				span.SetAttributes(attribute.Bool("error", true))
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// GetTraceID returns the trace ID from context.
func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(TraceIDKey).(string); ok {
		return v
	}
	return ""
}

// StartSpan starts a new span as a child of any existing span in the context.
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Tracer().Start(ctx, name, opts...)
}

// SpanFromContext returns the current span from context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddSpanAttributes adds attributes to the current span.
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error on the current span.
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}

// SetSpanStatus sets the status of the current span.
func SetSpanStatus(ctx context.Context, code trace.SpanKind, description string) {
	// Note: This is a simplified helper. For proper status setting,
	// use span.SetStatus(codes.Error, description) from go.opentelemetry.io/otel/codes
}
