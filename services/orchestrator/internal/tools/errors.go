// Package tools provides the tool registry for AI agent operations.
package tools

import (
	"encoding/json"
	"fmt"
)

// =============================================================================
// Tool Error Types
// =============================================================================

// ErrorCode represents a standardized tool error code.
type ErrorCode string

const (
	// ErrorCodeInvalidInput indicates invalid input parameters.
	ErrorCodeInvalidInput ErrorCode = "INVALID_INPUT"

	// ErrorCodeNotFound indicates a requested resource was not found.
	ErrorCodeNotFound ErrorCode = "NOT_FOUND"

	// ErrorCodeUnauthorized indicates insufficient permissions.
	ErrorCodeUnauthorized ErrorCode = "UNAUTHORIZED"

	// ErrorCodeUpstream indicates an error from an external dependency.
	ErrorCodeUpstream ErrorCode = "UPSTREAM_ERROR"

	// ErrorCodeRateLimited indicates rate limiting was applied.
	ErrorCodeRateLimited ErrorCode = "RATE_LIMITED"

	// ErrorCodeTimeout indicates the operation timed out.
	ErrorCodeTimeout ErrorCode = "TIMEOUT"

	// ErrorCodeConflict indicates a conflicting operation state.
	ErrorCodeConflict ErrorCode = "CONFLICT"

	// ErrorCodeInternal indicates an internal error.
	ErrorCodeInternal ErrorCode = "INTERNAL_ERROR"

	// ErrorCodeUnsupported indicates an unsupported operation.
	ErrorCodeUnsupported ErrorCode = "UNSUPPORTED"

	// ErrorCodePreconditionFailed indicates a precondition was not met.
	ErrorCodePreconditionFailed ErrorCode = "PRECONDITION_FAILED"
)

// ToolError represents a business-logic error from tool execution.
// This is distinct from transport/infrastructure errors (returned via error).
type ToolError struct {
	// Code is the standardized error code.
	Code ErrorCode `json:"code"`

	// Message is a human-readable error message.
	Message string `json:"message"`

	// Details contains additional context about the error.
	Details any `json:"details,omitempty"`

	// Retryable indicates if the operation can be retried.
	Retryable bool `json:"retryable,omitempty"`

	// RetryAfterSeconds suggests how long to wait before retrying.
	RetryAfterSeconds int `json:"retry_after_seconds,omitempty"`
}

// Error implements the error interface.
func (e *ToolError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// IsRetryable returns true if the error is retryable.
func (e *ToolError) IsRetryable() bool {
	return e.Retryable
}

// =============================================================================
// ToolResult - Standardized Tool Response
// =============================================================================

// ToolResult is the standardized response format for tool execution.
// It provides a clear separation between successful data and business errors.
type ToolResult struct {
	// Success indicates whether the operation succeeded.
	Success bool `json:"success"`

	// Data contains the result data on success.
	Data any `json:"data,omitempty"`

	// Error contains error details on failure.
	Error *ToolError `json:"error,omitempty"`

	// Metadata contains additional context about the execution.
	Metadata *ResultMetadata `json:"metadata,omitempty"`
}

// ResultMetadata contains execution metadata.
type ResultMetadata struct {
	// DurationMs is the execution duration in milliseconds.
	DurationMs int64 `json:"duration_ms,omitempty"`

	// ItemCount is the number of items processed/returned.
	ItemCount int `json:"item_count,omitempty"`

	// Truncated indicates if results were truncated.
	Truncated bool `json:"truncated,omitempty"`

	// Source indicates where the data came from.
	Source string `json:"source,omitempty"`

	// CacheHit indicates if the result was from cache.
	CacheHit bool `json:"cache_hit,omitempty"`
}

// =============================================================================
// Result Constructors
// =============================================================================

// NewSuccessResult creates a successful ToolResult.
func NewSuccessResult(data any) *ToolResult {
	return &ToolResult{
		Success: true,
		Data:    data,
	}
}

// NewSuccessResultWithMetadata creates a successful ToolResult with metadata.
func NewSuccessResultWithMetadata(data any, metadata *ResultMetadata) *ToolResult {
	return &ToolResult{
		Success:  true,
		Data:     data,
		Metadata: metadata,
	}
}

// NewErrorResult creates an error ToolResult.
func NewErrorResult(err *ToolError) *ToolResult {
	return &ToolResult{
		Success: false,
		Error:   err,
	}
}

// =============================================================================
// Error Constructors
// =============================================================================

// NewInvalidInputError creates an INVALID_INPUT error.
func NewInvalidInputError(message string, details any) *ToolError {
	return &ToolError{
		Code:      ErrorCodeInvalidInput,
		Message:   message,
		Details:   details,
		Retryable: false,
	}
}

// NewNotFoundError creates a NOT_FOUND error.
func NewNotFoundError(resourceType, resourceID string) *ToolError {
	return &ToolError{
		Code:    ErrorCodeNotFound,
		Message: fmt.Sprintf("%s not found: %s", resourceType, resourceID),
		Details: map[string]string{
			"resource_type": resourceType,
			"resource_id":   resourceID,
		},
		Retryable: false,
	}
}

// NewUnauthorizedError creates an UNAUTHORIZED error.
func NewUnauthorizedError(message string) *ToolError {
	return &ToolError{
		Code:      ErrorCodeUnauthorized,
		Message:   message,
		Retryable: false,
	}
}

// NewUpstreamError creates an UPSTREAM_ERROR for external dependency failures.
func NewUpstreamError(service, message string, retryable bool) *ToolError {
	return &ToolError{
		Code:    ErrorCodeUpstream,
		Message: fmt.Sprintf("%s: %s", service, message),
		Details: map[string]string{
			"service": service,
		},
		Retryable: retryable,
	}
}

// NewRateLimitedError creates a RATE_LIMITED error.
func NewRateLimitedError(service string, retryAfterSeconds int) *ToolError {
	return &ToolError{
		Code:              ErrorCodeRateLimited,
		Message:           fmt.Sprintf("rate limited by %s", service),
		Retryable:         true,
		RetryAfterSeconds: retryAfterSeconds,
	}
}

// NewTimeoutError creates a TIMEOUT error.
func NewTimeoutError(operation string, timeoutMs int64) *ToolError {
	return &ToolError{
		Code:    ErrorCodeTimeout,
		Message: fmt.Sprintf("%s timed out after %dms", operation, timeoutMs),
		Details: map[string]int64{
			"timeout_ms": timeoutMs,
		},
		Retryable: true,
	}
}

// NewConflictError creates a CONFLICT error.
func NewConflictError(message string, currentState any) *ToolError {
	return &ToolError{
		Code:    ErrorCodeConflict,
		Message: message,
		Details: map[string]any{
			"current_state": currentState,
		},
		Retryable: false,
	}
}

// NewInternalError creates an INTERNAL_ERROR.
func NewInternalError(message string) *ToolError {
	return &ToolError{
		Code:      ErrorCodeInternal,
		Message:   message,
		Retryable: true, // Internal errors are often transient
	}
}

// NewUnsupportedError creates an UNSUPPORTED error.
func NewUnsupportedError(operation, reason string) *ToolError {
	return &ToolError{
		Code:    ErrorCodeUnsupported,
		Message: fmt.Sprintf("%s is not supported: %s", operation, reason),
		Details: map[string]string{
			"operation": operation,
			"reason":    reason,
		},
		Retryable: false,
	}
}

// NewPreconditionFailedError creates a PRECONDITION_FAILED error.
func NewPreconditionFailedError(precondition, reason string) *ToolError {
	return &ToolError{
		Code:    ErrorCodePreconditionFailed,
		Message: fmt.Sprintf("precondition failed: %s - %s", precondition, reason),
		Details: map[string]string{
			"precondition": precondition,
			"reason":       reason,
		},
		Retryable: false,
	}
}

// =============================================================================
// Error Classification
// =============================================================================

// IsClientError returns true if the error is a client-side error (4xx equivalent).
func IsClientError(code ErrorCode) bool {
	switch code {
	case ErrorCodeInvalidInput, ErrorCodeNotFound, ErrorCodeUnauthorized,
		ErrorCodeConflict, ErrorCodeUnsupported, ErrorCodePreconditionFailed:
		return true
	default:
		return false
	}
}

// IsServerError returns true if the error is a server-side error (5xx equivalent).
func IsServerError(code ErrorCode) bool {
	switch code {
	case ErrorCodeUpstream, ErrorCodeInternal, ErrorCodeTimeout:
		return true
	default:
		return false
	}
}

// IsTransient returns true if the error is likely transient and can be retried.
func IsTransient(code ErrorCode) bool {
	switch code {
	case ErrorCodeUpstream, ErrorCodeRateLimited, ErrorCodeTimeout, ErrorCodeInternal:
		return true
	default:
		return false
	}
}

// =============================================================================
// ToolResult Helpers
// =============================================================================

// IsSuccess returns true if the result represents success.
func (r *ToolResult) IsSuccess() bool {
	return r.Success
}

// IsError returns true if the result represents an error.
func (r *ToolResult) IsError() bool {
	return !r.Success && r.Error != nil
}

// GetData returns the data, or nil if there was an error.
func (r *ToolResult) GetData() any {
	if r.Success {
		return r.Data
	}
	return nil
}

// GetError returns the error, or nil if successful.
func (r *ToolResult) GetError() *ToolError {
	return r.Error
}

// ToJSON returns the result as JSON bytes.
func (r *ToolResult) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// MustJSON returns the result as JSON bytes, panicking on error.
// For use in tests only.
func (r *ToolResult) MustJSON() []byte {
	b, err := json.Marshal(r)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal ToolResult: %v", err))
	}
	return b
}

// =============================================================================
// Error Wrapping
// =============================================================================

// WrapError wraps a Go error into a ToolError.
// If the error is already a ToolError, it is returned as-is.
func WrapError(err error) *ToolError {
	if err == nil {
		return nil
	}

	// Check if already a ToolError
	if te, ok := err.(*ToolError); ok {
		return te
	}

	// Wrap as internal error
	return &ToolError{
		Code:      ErrorCodeInternal,
		Message:   err.Error(),
		Retryable: true,
	}
}

// FromError creates a ToolResult from a Go error.
// Returns a success result if err is nil.
func FromError(err error) *ToolResult {
	if err == nil {
		return NewSuccessResult(nil)
	}

	return NewErrorResult(WrapError(err))
}
