package resilience

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBreaker_ClosedState(t *testing.T) {
	b := NewBreaker(&BreakerConfig{
		Name:        "test",
		MaxFailures: 3,
		Timeout:     100 * time.Millisecond,
	})

	// Should start closed
	if b.State() != StateClosed {
		t.Errorf("expected state closed, got %s", b.State())
	}

	// Should allow requests
	result, err := b.Execute(context.Background(), func() (any, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result != "success" {
		t.Errorf("expected 'success', got %v", result)
	}
}

func TestBreaker_OpensAfterFailures(t *testing.T) {
	b := NewBreaker(&BreakerConfig{
		Name:        "test",
		MaxFailures: 3,
		Timeout:     100 * time.Millisecond,
	})

	testErr := errors.New("test error")

	// Cause failures
	for i := 0; i < 3; i++ {
		_, _ = b.Execute(context.Background(), func() (any, error) {
			return nil, testErr
		})
	}

	// Should be open now
	if b.State() != StateOpen {
		t.Errorf("expected state open, got %s", b.State())
	}

	// Should reject requests
	_, err := b.Execute(context.Background(), func() (any, error) {
		return "should not run", nil
	})

	if err == nil {
		t.Error("expected error when circuit is open")
	}

	var breakerErr *BreakerOpenError
	if !errors.As(err, &breakerErr) {
		t.Errorf("expected BreakerOpenError, got %T", err)
	}
}

func TestBreaker_TransitionsToHalfOpen(t *testing.T) {
	b := NewBreaker(&BreakerConfig{
		Name:             "test",
		MaxFailures:      2,
		Timeout:          50 * time.Millisecond,
		HalfOpenMaxCalls: 2,
	})

	testErr := errors.New("test error")

	// Trip the breaker
	for i := 0; i < 2; i++ {
		_, _ = b.Execute(context.Background(), func() (any, error) {
			return nil, testErr
		})
	}

	if b.State() != StateOpen {
		t.Fatalf("expected open state, got %s", b.State())
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Next request should transition to half-open
	_, err := b.Execute(context.Background(), func() (any, error) {
		return "test", nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if b.State() != StateHalfOpen {
		t.Errorf("expected half-open state, got %s", b.State())
	}
}

func TestBreaker_ClosesAfterHalfOpenSuccess(t *testing.T) {
	b := NewBreaker(&BreakerConfig{
		Name:             "test",
		MaxFailures:      1,
		Timeout:          10 * time.Millisecond,
		HalfOpenMaxCalls: 2,
	})

	// Trip the breaker
	_, _ = b.Execute(context.Background(), func() (any, error) {
		return nil, errors.New("fail")
	})

	if b.State() != StateOpen {
		t.Fatalf("expected open state, got %s", b.State())
	}

	// Wait for timeout
	time.Sleep(20 * time.Millisecond)

	// Successful requests in half-open should close
	for i := 0; i < 2; i++ {
		_, err := b.Execute(context.Background(), func() (any, error) {
			return "success", nil
		})
		if err != nil {
			t.Errorf("unexpected error on call %d: %v", i, err)
		}
	}

	if b.State() != StateClosed {
		t.Errorf("expected closed state, got %s", b.State())
	}
}

func TestBreaker_ReOpensOnHalfOpenFailure(t *testing.T) {
	b := NewBreaker(&BreakerConfig{
		Name:             "test",
		MaxFailures:      1,
		Timeout:          10 * time.Millisecond,
		HalfOpenMaxCalls: 2,
	})

	// Trip the breaker
	_, _ = b.Execute(context.Background(), func() (any, error) {
		return nil, errors.New("fail")
	})

	time.Sleep(20 * time.Millisecond)

	// Fail in half-open
	_, _ = b.Execute(context.Background(), func() (any, error) {
		return nil, errors.New("fail again")
	})

	if b.State() != StateOpen {
		t.Errorf("expected open state after half-open failure, got %s", b.State())
	}
}

func TestBreaker_SuccessResetsFailureCount(t *testing.T) {
	b := NewBreaker(&BreakerConfig{
		Name:        "test",
		MaxFailures: 3,
		Timeout:     100 * time.Millisecond,
	})

	testErr := errors.New("test error")

	// Two failures
	for i := 0; i < 2; i++ {
		_, _ = b.Execute(context.Background(), func() (any, error) {
			return nil, testErr
		})
	}

	if b.Failures() != 2 {
		t.Errorf("expected 2 failures, got %d", b.Failures())
	}

	// Success should reset
	_, _ = b.Execute(context.Background(), func() (any, error) {
		return "success", nil
	})

	if b.Failures() != 0 {
		t.Errorf("expected 0 failures after success, got %d", b.Failures())
	}
}

func TestBreaker_Metrics(t *testing.T) {
	b := NewBreaker(&BreakerConfig{
		Name:        "test-metrics",
		MaxFailures: 5,
		Timeout:     100 * time.Millisecond,
	})

	// Some successful calls
	for i := 0; i < 3; i++ {
		_, _ = b.Execute(context.Background(), func() (any, error) {
			return "ok", nil
		})
	}

	// Some failures
	for i := 0; i < 2; i++ {
		_, _ = b.Execute(context.Background(), func() (any, error) {
			return nil, errors.New("fail")
		})
	}

	metrics := b.Metrics()

	if metrics.Name != "test-metrics" {
		t.Errorf("expected name 'test-metrics', got %q", metrics.Name)
	}

	if metrics.TotalCalls != 5 {
		t.Errorf("expected total_calls=5, got %d", metrics.TotalCalls)
	}

	if metrics.TotalSuccesses != 3 {
		t.Errorf("expected total_successes=3, got %d", metrics.TotalSuccesses)
	}

	if metrics.TotalFailures != 2 {
		t.Errorf("expected total_failures=2, got %d", metrics.TotalFailures)
	}
}

func TestBreaker_Reset(t *testing.T) {
	b := NewBreaker(&BreakerConfig{
		Name:        "test",
		MaxFailures: 2,
		Timeout:     100 * time.Millisecond,
	})

	// Trip the breaker
	for i := 0; i < 2; i++ {
		_, _ = b.Execute(context.Background(), func() (any, error) {
			return nil, errors.New("fail")
		})
	}

	if b.State() != StateOpen {
		t.Fatalf("expected open state, got %s", b.State())
	}

	// Reset
	b.Reset()

	if b.State() != StateClosed {
		t.Errorf("expected closed state after reset, got %s", b.State())
	}

	if b.Failures() != 0 {
		t.Errorf("expected 0 failures after reset, got %d", b.Failures())
	}
}

func TestBreaker_CustomIsSuccessful(t *testing.T) {
	// Treat "soft" errors as successes
	softErr := errors.New("soft error")

	b := NewBreaker(&BreakerConfig{
		Name:        "test",
		MaxFailures: 2,
		Timeout:     100 * time.Millisecond,
		IsSuccessful: func(err error) bool {
			return err == nil || errors.Is(err, softErr)
		},
	})

	// Soft errors shouldn't count as failures
	for i := 0; i < 5; i++ {
		_, _ = b.Execute(context.Background(), func() (any, error) {
			return nil, softErr
		})
	}

	if b.State() != StateClosed {
		t.Errorf("expected closed state (soft errors), got %s", b.State())
	}

	if b.Failures() != 0 {
		t.Errorf("expected 0 failures (soft errors), got %d", b.Failures())
	}
}

func TestBreaker_OnStateChange(t *testing.T) {
	var transitions []struct {
		from, to State
	}
	var mu sync.Mutex

	b := NewBreaker(&BreakerConfig{
		Name:        "test",
		MaxFailures: 1,
		Timeout:     10 * time.Millisecond,
		OnStateChange: func(name string, from, to State) {
			mu.Lock()
			transitions = append(transitions, struct{ from, to State }{from, to})
			mu.Unlock()
		},
	})

	// Trip the breaker
	_, _ = b.Execute(context.Background(), func() (any, error) {
		return nil, errors.New("fail")
	})

	time.Sleep(20 * time.Millisecond)

	// Transition to half-open
	_, _ = b.Execute(context.Background(), func() (any, error) {
		return "ok", nil
	})

	// Wait for async callbacks
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(transitions) < 1 {
		t.Fatal("expected at least 1 transition")
	}

	// First transition should be closed -> open
	if transitions[0].from != StateClosed || transitions[0].to != StateOpen {
		t.Errorf("expected closed->open, got %s->%s",
			transitions[0].from, transitions[0].to)
	}
}

func TestBreaker_ConcurrentAccess(t *testing.T) {
	b := NewBreaker(&BreakerConfig{
		Name:        "concurrent",
		MaxFailures: 100,
		Timeout:     100 * time.Millisecond,
	})

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	// Concurrent calls
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, err := b.Execute(context.Background(), func() (any, error) {
				if n%3 == 0 {
					return nil, errors.New("fail")
				}
				return "ok", nil
			})
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	metrics := b.Metrics()
	if metrics.TotalCalls != 100 {
		t.Errorf("expected 100 total calls, got %d", metrics.TotalCalls)
	}
}

func TestBreakerOpenError(t *testing.T) {
	err := &BreakerOpenError{
		Name:     "test",
		RetryAt:  time.Now().Add(30 * time.Second),
		Failures: 5,
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("expected non-empty error string")
	}

	retryAfter := err.RetryAfter()
	if retryAfter <= 0 || retryAfter > 30*time.Second {
		t.Errorf("unexpected retry_after: %v", retryAfter)
	}
}

func TestBreakerOpenError_RetryAfterPast(t *testing.T) {
	err := &BreakerOpenError{
		Name:     "test",
		RetryAt:  time.Now().Add(-1 * time.Second),
		Failures: 5,
	}

	retryAfter := err.RetryAfter()
	if retryAfter != 0 {
		t.Errorf("expected retry_after=0 for past time, got %v", retryAfter)
	}
}

func TestState_String(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.state.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.state.String())
			}
		})
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry(&BreakerConfig{
		MaxFailures: 3,
		Timeout:     100 * time.Millisecond,
	})

	// First call creates breaker
	b1 := r.Get("aws.ec2.describe")
	if b1 == nil {
		t.Fatal("expected non-nil breaker")
	}

	// Second call returns same breaker
	b2 := r.Get("aws.ec2.describe")
	if b1 != b2 {
		t.Error("expected same breaker instance")
	}

	// Different key returns different breaker
	b3 := r.Get("aws.s3.list")
	if b3 == b1 {
		t.Error("expected different breaker for different key")
	}
}

func TestRegistry_AllMetrics(t *testing.T) {
	r := NewRegistry(nil)

	// Create some breakers
	r.Get("service-a")
	r.Get("service-b")
	r.Get("service-c")

	metrics := r.AllMetrics()
	if len(metrics) != 3 {
		t.Errorf("expected 3 breakers, got %d", len(metrics))
	}
}

func TestRegistry_ResetAll(t *testing.T) {
	r := NewRegistry(&BreakerConfig{
		MaxFailures: 1,
		Timeout:     100 * time.Millisecond,
	})

	// Create and trip breakers
	for _, name := range []string{"a", "b", "c"} {
		b := r.Get(name)
		_, _ = b.Execute(context.Background(), func() (any, error) {
			return nil, errors.New("fail")
		})
	}

	// Verify they're all open
	for _, name := range []string{"a", "b", "c"} {
		if r.Get(name).State() != StateOpen {
			t.Errorf("breaker %s should be open", name)
		}
	}

	// Reset all
	r.ResetAll()

	// Verify they're all closed
	for _, name := range []string{"a", "b", "c"} {
		if r.Get(name).State() != StateClosed {
			t.Errorf("breaker %s should be closed after reset", name)
		}
	}
}

func TestRegistry_ConcurrentGet(t *testing.T) {
	r := NewRegistry(nil)

	var wg sync.WaitGroup
	breakers := make(chan *Breaker, 100)

	// Concurrent gets for the same key
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			breakers <- r.Get("shared-key")
		}()
	}

	wg.Wait()
	close(breakers)

	// All should be the same instance
	var first *Breaker
	for b := range breakers {
		if first == nil {
			first = b
		} else if b != first {
			t.Error("expected same breaker instance for all concurrent gets")
			break
		}
	}
}

func TestDefaultBreakerConfig(t *testing.T) {
	config := DefaultBreakerConfig("test-service")

	if config.Name != "test-service" {
		t.Errorf("expected name 'test-service', got %q", config.Name)
	}

	if config.MaxFailures != 5 {
		t.Errorf("expected max_failures=5, got %d", config.MaxFailures)
	}

	if config.Timeout != 30*time.Second {
		t.Errorf("expected timeout=30s, got %v", config.Timeout)
	}

	if config.HalfOpenMaxCalls != 3 {
		t.Errorf("expected half_open_max_calls=3, got %d", config.HalfOpenMaxCalls)
	}
}
