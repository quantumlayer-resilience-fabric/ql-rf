// Package logger provides structured logging using slog.
package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// RequestIDKey is the context key for request ID.
	RequestIDKey contextKey = "request_id"
	// OrgIDKey is the context key for organization ID.
	OrgIDKey contextKey = "org_id"
	// UserIDKey is the context key for user ID.
	UserIDKey contextKey = "user_id"
)

// Logger wraps slog.Logger with additional functionality.
type Logger struct {
	*slog.Logger
}

// New creates a new logger with the given configuration.
func New(level, format string) *Logger {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level:     parseLevel(level),
		AddSource: true,
	}

	if strings.ToLower(format) == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return &Logger{Logger: logger}
}

// parseLevel parses a log level string into slog.Level.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithContext returns a logger with context values.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	attrs := []any{}

	if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
		attrs = append(attrs, slog.String("request_id", reqID))
	}

	if orgID, ok := ctx.Value(OrgIDKey).(string); ok && orgID != "" {
		attrs = append(attrs, slog.String("org_id", orgID))
	}

	if userID, ok := ctx.Value(UserIDKey).(string); ok && userID != "" {
		attrs = append(attrs, slog.String("user_id", userID))
	}

	if len(attrs) == 0 {
		return l
	}

	return &Logger{Logger: l.With(attrs...)}
}

// WithRequestID returns a logger with the request ID.
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{Logger: l.With(slog.String("request_id", requestID))}
}

// WithOrg returns a logger with the organization ID.
func (l *Logger) WithOrg(orgID string) *Logger {
	return &Logger{Logger: l.With(slog.String("org_id", orgID))}
}

// WithService returns a logger with the service name.
func (l *Logger) WithService(service string) *Logger {
	return &Logger{Logger: l.With(slog.String("service", service))}
}

// WithComponent returns a logger with the component name.
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{Logger: l.With(slog.String("component", component))}
}

// WithError returns a logger with the error.
func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}
	return &Logger{Logger: l.With(slog.String("error", err.Error()))}
}

// InfoContext logs an info message with context.
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).Info(msg, args...)
}

// ErrorContext logs an error message with context.
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).Error(msg, args...)
}

// DebugContext logs a debug message with context.
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).Debug(msg, args...)
}

// WarnContext logs a warning message with context.
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).Warn(msg, args...)
}

// SetContextValue sets a value in the context.
func SetContextValue(ctx context.Context, key contextKey, value string) context.Context {
	return context.WithValue(ctx, key, value)
}

// GetRequestID gets the request ID from context.
func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(RequestIDKey).(string); ok {
		return v
	}
	return ""
}

// GetOrgID gets the organization ID from context.
func GetOrgID(ctx context.Context) string {
	if v, ok := ctx.Value(OrgIDKey).(string); ok {
		return v
	}
	return ""
}

// GetUserID gets the user ID from context.
func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}
