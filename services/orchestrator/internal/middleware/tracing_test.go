package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TestTracingConfig(t *testing.T) {
	cfg := TracingConfig{
		ServiceName:    "orchestrator",
		ServiceVersion: "1.0.0",
		Enabled:        true,
	}

	assert.Equal(t, "orchestrator", cfg.ServiceName)
	assert.Equal(t, "1.0.0", cfg.ServiceVersion)
	assert.True(t, cfg.Enabled)
}

func TestTracer(t *testing.T) {
	tracer := Tracer()
	assert.NotNil(t, tracer)
}

func TestTracing_Disabled(t *testing.T) {
	cfg := TracingConfig{
		Enabled: false,
	}

	middleware := Tracing(cfg)

	var called bool
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestTracing_Enabled(t *testing.T) {
	cfg := TracingConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Enabled:        true,
	}

	middleware := Tracing(cfg)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = GetTraceID(r.Context()) // Trace ID may be empty if no trace provider configured
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// Trace ID should be set (may be empty if no trace provider configured)
}

func TestTracing_StatusCodes(t *testing.T) {
	cfg := TracingConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Enabled:        true,
	}

	middleware := Tracing(cfg)

	tests := []struct {
		name       string
		statusCode int
	}{
		{"success", http.StatusOK},
		{"created", http.StatusCreated},
		{"bad request", http.StatusBadRequest},
		{"not found", http.StatusNotFound},
		{"server error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))

			req := httptest.NewRequest("GET", "/test", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.statusCode, rr.Code)
		})
	}
}

func TestResponseWriter(t *testing.T) {
	t.Run("captures status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		rw.WriteHeader(http.StatusCreated)

		assert.Equal(t, http.StatusCreated, rw.statusCode)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("default status code is 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		assert.Equal(t, http.StatusOK, rw.statusCode)
	})

	t.Run("writes body", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		_, err := rw.Write([]byte("hello"))
		assert.NoError(t, err)
		assert.Equal(t, "hello", w.Body.String())
	})
}

func TestGetTraceID(t *testing.T) {
	t.Run("with trace ID", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), TraceIDKey, "abc123")
		assert.Equal(t, "abc123", GetTraceID(ctx))
	})

	t.Run("without trace ID", func(t *testing.T) {
		assert.Equal(t, "", GetTraceID(context.Background()))
	})

	t.Run("wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), TraceIDKey, 123)
		assert.Equal(t, "", GetTraceID(ctx))
	})
}

func TestStartSpan(t *testing.T) {
	ctx, span := StartSpan(context.Background(), "test-span")
	defer span.End()

	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
}

func TestStartSpan_WithOptions(t *testing.T) {
	opts := []trace.SpanStartOption{
		trace.WithAttributes(
			attribute.String("key", "value"),
		),
	}

	ctx, span := StartSpan(context.Background(), "test-span", opts...)
	defer span.End()

	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
}

func TestSpanFromContext(t *testing.T) {
	ctx, span := StartSpan(context.Background(), "test-span")
	defer span.End()

	spanFromCtx := SpanFromContext(ctx)
	assert.NotNil(t, spanFromCtx)
}

func TestAddSpanAttributes(t *testing.T) {
	ctx, span := StartSpan(context.Background(), "test-span")
	defer span.End()

	// Should not panic
	AddSpanAttributes(ctx,
		attribute.String("key1", "value1"),
		attribute.Int("key2", 42),
	)
}

func TestRecordError(t *testing.T) {
	ctx, span := StartSpan(context.Background(), "test-span")
	defer span.End()

	err := assert.AnError

	// Should not panic
	RecordError(ctx, err)
}

func TestSetSpanStatus(t *testing.T) {
	ctx, span := StartSpan(context.Background(), "test-span")
	defer span.End()

	// Should not panic
	SetSpanStatus(ctx, trace.SpanKindServer, "test status")
}

func TestTracerName(t *testing.T) {
	assert.Equal(t, "github.com/quantumlayerhq/ql-rf/services/orchestrator", TracerName)
}

func TestTraceIDKey(t *testing.T) {
	assert.Equal(t, ContextKey("trace_id"), TraceIDKey)
}

func TestTracing_HTTPMethods(t *testing.T) {
	cfg := TracingConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Enabled:        true,
	}

	middleware := Tracing(cfg)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(method, "/api/test", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
		})
	}
}
