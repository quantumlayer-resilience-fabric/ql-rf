package tools

import (
	"errors"
	"testing"
)

func TestToolError_Error(t *testing.T) {
	err := &ToolError{
		Code:    ErrorCodeNotFound,
		Message: "asset not found",
	}

	expected := "NOT_FOUND: asset not found"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestToolError_IsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       *ToolError
		retryable bool
	}{
		{
			name:      "retryable error",
			err:       &ToolError{Code: ErrorCodeTimeout, Retryable: true},
			retryable: true,
		},
		{
			name:      "non-retryable error",
			err:       &ToolError{Code: ErrorCodeInvalidInput, Retryable: false},
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.IsRetryable() != tt.retryable {
				t.Errorf("expected retryable=%v, got %v", tt.retryable, tt.err.IsRetryable())
			}
		})
	}
}

func TestNewSuccessResult(t *testing.T) {
	data := map[string]string{"key": "value"}
	result := NewSuccessResult(data)

	if !result.Success {
		t.Error("expected success=true")
	}

	if result.Error != nil {
		t.Error("expected error=nil")
	}

	if result.Data == nil {
		t.Error("expected data to be set")
	}
}

func TestNewErrorResult(t *testing.T) {
	err := NewNotFoundError("asset", "asset-123")
	result := NewErrorResult(err)

	if result.Success {
		t.Error("expected success=false")
	}

	if result.Error == nil {
		t.Error("expected error to be set")
	}

	if result.Error.Code != ErrorCodeNotFound {
		t.Errorf("expected code %s, got %s", ErrorCodeNotFound, result.Error.Code)
	}
}

func TestNewInvalidInputError(t *testing.T) {
	err := NewInvalidInputError("missing required field", map[string]string{"field": "name"})

	if err.Code != ErrorCodeInvalidInput {
		t.Errorf("expected code %s, got %s", ErrorCodeInvalidInput, err.Code)
	}

	if err.Retryable {
		t.Error("expected non-retryable")
	}
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("image", "img-456")

	if err.Code != ErrorCodeNotFound {
		t.Errorf("expected code %s, got %s", ErrorCodeNotFound, err.Code)
	}

	if err.Message != "image not found: img-456" {
		t.Errorf("unexpected message: %s", err.Message)
	}
}

func TestNewUpstreamError(t *testing.T) {
	err := NewUpstreamError("AWS", "service unavailable", true)

	if err.Code != ErrorCodeUpstream {
		t.Errorf("expected code %s, got %s", ErrorCodeUpstream, err.Code)
	}

	if !err.Retryable {
		t.Error("expected retryable")
	}
}

func TestNewRateLimitedError(t *testing.T) {
	err := NewRateLimitedError("OpenAI", 60)

	if err.Code != ErrorCodeRateLimited {
		t.Errorf("expected code %s, got %s", ErrorCodeRateLimited, err.Code)
	}

	if err.RetryAfterSeconds != 60 {
		t.Errorf("expected retry_after=60, got %d", err.RetryAfterSeconds)
	}

	if !err.Retryable {
		t.Error("expected retryable")
	}
}

func TestNewTimeoutError(t *testing.T) {
	err := NewTimeoutError("database query", 5000)

	if err.Code != ErrorCodeTimeout {
		t.Errorf("expected code %s, got %s", ErrorCodeTimeout, err.Code)
	}

	if !err.Retryable {
		t.Error("expected retryable")
	}
}

func TestIsClientError(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		isClient bool
	}{
		{ErrorCodeInvalidInput, true},
		{ErrorCodeNotFound, true},
		{ErrorCodeUnauthorized, true},
		{ErrorCodeConflict, true},
		{ErrorCodeUnsupported, true},
		{ErrorCodePreconditionFailed, true},
		{ErrorCodeUpstream, false},
		{ErrorCodeInternal, false},
		{ErrorCodeTimeout, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if IsClientError(tt.code) != tt.isClient {
				t.Errorf("IsClientError(%s) = %v, want %v", tt.code, !tt.isClient, tt.isClient)
			}
		})
	}
}

func TestIsServerError(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		isServer bool
	}{
		{ErrorCodeUpstream, true},
		{ErrorCodeInternal, true},
		{ErrorCodeTimeout, true},
		{ErrorCodeInvalidInput, false},
		{ErrorCodeNotFound, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if IsServerError(tt.code) != tt.isServer {
				t.Errorf("IsServerError(%s) = %v, want %v", tt.code, !tt.isServer, tt.isServer)
			}
		})
	}
}

func TestIsTransient(t *testing.T) {
	tests := []struct {
		code        ErrorCode
		isTransient bool
	}{
		{ErrorCodeUpstream, true},
		{ErrorCodeRateLimited, true},
		{ErrorCodeTimeout, true},
		{ErrorCodeInternal, true},
		{ErrorCodeInvalidInput, false},
		{ErrorCodeNotFound, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if IsTransient(tt.code) != tt.isTransient {
				t.Errorf("IsTransient(%s) = %v, want %v", tt.code, !tt.isTransient, tt.isTransient)
			}
		})
	}
}

func TestToolResult_Helpers(t *testing.T) {
	t.Run("success result", func(t *testing.T) {
		data := "test data"
		result := NewSuccessResult(data)

		if !result.IsSuccess() {
			t.Error("expected IsSuccess=true")
		}

		if result.IsError() {
			t.Error("expected IsError=false")
		}

		if result.GetData() != data {
			t.Errorf("expected data %q, got %v", data, result.GetData())
		}

		if result.GetError() != nil {
			t.Error("expected GetError=nil")
		}
	})

	t.Run("error result", func(t *testing.T) {
		err := NewInternalError("something went wrong")
		result := NewErrorResult(err)

		if result.IsSuccess() {
			t.Error("expected IsSuccess=false")
		}

		if !result.IsError() {
			t.Error("expected IsError=true")
		}

		if result.GetData() != nil {
			t.Error("expected GetData=nil")
		}

		if result.GetError() == nil {
			t.Error("expected GetError not nil")
		}
	})
}

func TestToolResult_ToJSON(t *testing.T) {
	result := NewSuccessResult(map[string]int{"count": 42})

	jsonBytes, err := result.ToJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(jsonBytes) == 0 {
		t.Error("expected non-empty JSON")
	}
}

func TestToolResult_MustJSON(t *testing.T) {
	result := NewSuccessResult("test")

	// Should not panic
	jsonBytes := result.MustJSON()
	if len(jsonBytes) == 0 {
		t.Error("expected non-empty JSON")
	}
}

func TestWrapError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		wrapped := WrapError(nil)
		if wrapped != nil {
			t.Error("expected nil")
		}
	})

	t.Run("regular error", func(t *testing.T) {
		err := errors.New("something failed")
		wrapped := WrapError(err)

		if wrapped.Code != ErrorCodeInternal {
			t.Errorf("expected code %s, got %s", ErrorCodeInternal, wrapped.Code)
		}

		if wrapped.Message != "something failed" {
			t.Errorf("expected message 'something failed', got %q", wrapped.Message)
		}
	})

	t.Run("already ToolError", func(t *testing.T) {
		original := NewNotFoundError("asset", "123")
		wrapped := WrapError(original)

		if wrapped != original {
			t.Error("expected same error instance")
		}
	})
}

func TestFromError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		result := FromError(nil)
		if !result.Success {
			t.Error("expected success=true for nil error")
		}
	})

	t.Run("non-nil error", func(t *testing.T) {
		err := errors.New("failed")
		result := FromError(err)

		if result.Success {
			t.Error("expected success=false")
		}

		if result.Error == nil {
			t.Error("expected error to be set")
		}
	})
}

func TestNewSuccessResultWithMetadata(t *testing.T) {
	data := []string{"item1", "item2"}
	metadata := &ResultMetadata{
		DurationMs: 150,
		ItemCount:  2,
		Source:     "database",
	}

	result := NewSuccessResultWithMetadata(data, metadata)

	if !result.Success {
		t.Error("expected success=true")
	}

	if result.Metadata == nil {
		t.Fatal("expected metadata to be set")
	}

	if result.Metadata.DurationMs != 150 {
		t.Errorf("expected duration_ms=150, got %d", result.Metadata.DurationMs)
	}

	if result.Metadata.ItemCount != 2 {
		t.Errorf("expected item_count=2, got %d", result.Metadata.ItemCount)
	}
}
