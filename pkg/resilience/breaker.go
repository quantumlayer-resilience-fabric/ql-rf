// Package resilience provides reliability patterns for cloud SDK calls.
package resilience

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// =============================================================================
// Circuit Breaker States
// =============================================================================

// State represents the circuit breaker state.
type State int

const (
	// StateClosed allows requests to pass through.
	StateClosed State = iota

	// StateOpen blocks all requests.
	StateOpen

	// StateHalfOpen allows limited requests for testing recovery.
	StateHalfOpen
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// =============================================================================
// Circuit Breaker Configuration
// =============================================================================

// BreakerConfig configures the circuit breaker behavior.
type BreakerConfig struct {
	// Name identifies this breaker (used in metrics).
	Name string

	// MaxFailures is the threshold to trip the circuit.
	MaxFailures int

	// Timeout is how long the circuit stays open.
	Timeout time.Duration

	// HalfOpenMaxCalls is how many test calls to allow in half-open state.
	HalfOpenMaxCalls int

	// OnStateChange is called when the breaker changes state.
	OnStateChange func(name string, from, to State)

	// IsSuccessful determines if a response is successful.
	// If nil, all non-error responses are considered successful.
	IsSuccessful func(err error) bool
}

// DefaultBreakerConfig returns a sensible default configuration.
func DefaultBreakerConfig(name string) *BreakerConfig {
	return &BreakerConfig{
		Name:             name,
		MaxFailures:      5,
		Timeout:          30 * time.Second,
		HalfOpenMaxCalls: 3,
	}
}

// =============================================================================
// Circuit Breaker
// =============================================================================

// Breaker implements the circuit breaker pattern.
type Breaker struct {
	config *BreakerConfig

	mu            sync.RWMutex
	state         State
	failures      int
	successes     int
	lastFailure   time.Time
	halfOpenCalls int

	// Metrics
	totalCalls      int64
	totalFailures   int64
	totalSuccesses  int64
	totalRejected   int64
}

// NewBreaker creates a new circuit breaker.
func NewBreaker(config *BreakerConfig) *Breaker {
	if config == nil {
		config = DefaultBreakerConfig("default")
	}

	return &Breaker{
		config: config,
		state:  StateClosed,
	}
}

// Execute wraps a function with circuit breaker protection.
func (b *Breaker) Execute(ctx context.Context, fn func() (any, error)) (any, error) {
	// Check if we can proceed
	if err := b.beforeRequest(); err != nil {
		return nil, err
	}

	// Execute the function
	result, err := fn()

	// Record the result
	b.afterRequest(err)

	return result, err
}

// beforeRequest checks if the request should be allowed.
func (b *Breaker) beforeRequest() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.totalCalls++

	switch b.state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if timeout has elapsed
		if time.Since(b.lastFailure) >= b.config.Timeout {
			b.transition(StateHalfOpen)
			b.halfOpenCalls = 1
			return nil
		}
		b.totalRejected++
		return &BreakerOpenError{
			Name:      b.config.Name,
			RetryAt:   b.lastFailure.Add(b.config.Timeout),
			Failures:  b.failures,
		}

	case StateHalfOpen:
		// Allow limited test calls
		if b.halfOpenCalls < b.config.HalfOpenMaxCalls {
			b.halfOpenCalls++
			return nil
		}
		b.totalRejected++
		return &BreakerOpenError{
			Name:     b.config.Name,
			RetryAt:  time.Now().Add(time.Second),
			Failures: b.failures,
		}
	}

	return nil
}

// afterRequest records the result of a request.
func (b *Breaker) afterRequest(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	success := err == nil
	if b.config.IsSuccessful != nil {
		success = b.config.IsSuccessful(err)
	}

	if success {
		b.recordSuccess()
	} else {
		b.recordFailure()
	}
}

// recordSuccess handles a successful request.
func (b *Breaker) recordSuccess() {
	b.totalSuccesses++

	switch b.state {
	case StateClosed:
		// Reset failure counter on success
		b.failures = 0

	case StateHalfOpen:
		b.successes++
		// Check if we've had enough successes to close
		if b.successes >= b.config.HalfOpenMaxCalls {
			b.transition(StateClosed)
			b.failures = 0
			b.successes = 0
		}
	}
}

// recordFailure handles a failed request.
func (b *Breaker) recordFailure() {
	b.totalFailures++
	b.failures++
	b.lastFailure = time.Now()

	switch b.state {
	case StateClosed:
		// Check if we've hit the threshold
		if b.failures >= b.config.MaxFailures {
			b.transition(StateOpen)
		}

	case StateHalfOpen:
		// Any failure in half-open trips back to open
		b.transition(StateOpen)
		b.successes = 0
	}
}

// transition changes the breaker state.
func (b *Breaker) transition(to State) {
	from := b.state
	b.state = to

	if b.config.OnStateChange != nil {
		// Call async to avoid holding the lock
		go b.config.OnStateChange(b.config.Name, from, to)
	}
}

// =============================================================================
// Breaker State Access
// =============================================================================

// State returns the current state of the breaker.
func (b *Breaker) State() State {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// Failures returns the current failure count.
func (b *Breaker) Failures() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.failures
}

// Metrics returns the breaker metrics.
func (b *Breaker) Metrics() BreakerMetrics {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return BreakerMetrics{
		Name:           b.config.Name,
		State:          b.state.String(),
		TotalCalls:     b.totalCalls,
		TotalFailures:  b.totalFailures,
		TotalSuccesses: b.totalSuccesses,
		TotalRejected:  b.totalRejected,
		CurrentFailures: b.failures,
	}
}

// Reset resets the breaker to closed state.
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.state = StateClosed
	b.failures = 0
	b.successes = 0
	b.halfOpenCalls = 0
}

// =============================================================================
// Metrics
// =============================================================================

// BreakerMetrics contains circuit breaker metrics.
type BreakerMetrics struct {
	Name            string `json:"name"`
	State           string `json:"state"`
	TotalCalls      int64  `json:"total_calls"`
	TotalFailures   int64  `json:"total_failures"`
	TotalSuccesses  int64  `json:"total_successes"`
	TotalRejected   int64  `json:"total_rejected"`
	CurrentFailures int    `json:"current_failures"`
}

// =============================================================================
// Errors
// =============================================================================

// BreakerOpenError is returned when the circuit is open.
type BreakerOpenError struct {
	Name     string
	RetryAt  time.Time
	Failures int
}

// Error implements the error interface.
func (e *BreakerOpenError) Error() string {
	return fmt.Sprintf("circuit breaker %q is open (failures=%d, retry at %s)",
		e.Name, e.Failures, e.RetryAt.Format(time.RFC3339))
}

// RetryAfter returns the duration until retry.
func (e *BreakerOpenError) RetryAfter() time.Duration {
	d := time.Until(e.RetryAt)
	if d < 0 {
		return 0
	}
	return d
}

// =============================================================================
// Breaker Registry
// =============================================================================

// Registry manages multiple circuit breakers.
type Registry struct {
	mu       sync.RWMutex
	breakers map[string]*Breaker
	config   *BreakerConfig
}

// NewRegistry creates a new breaker registry.
func NewRegistry(defaultConfig *BreakerConfig) *Registry {
	if defaultConfig == nil {
		defaultConfig = DefaultBreakerConfig("default")
	}
	return &Registry{
		breakers: make(map[string]*Breaker),
		config:   defaultConfig,
	}
}

// Get returns or creates a breaker for the given key.
func (r *Registry) Get(key string) *Breaker {
	r.mu.RLock()
	if b, ok := r.breakers[key]; ok {
		r.mu.RUnlock()
		return b
	}
	r.mu.RUnlock()

	// Create new breaker
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if b, ok := r.breakers[key]; ok {
		return b
	}

	config := &BreakerConfig{
		Name:             key,
		MaxFailures:      r.config.MaxFailures,
		Timeout:          r.config.Timeout,
		HalfOpenMaxCalls: r.config.HalfOpenMaxCalls,
		OnStateChange:    r.config.OnStateChange,
		IsSuccessful:     r.config.IsSuccessful,
	}

	b := NewBreaker(config)
	r.breakers[key] = b
	return b
}

// AllMetrics returns metrics for all breakers.
func (r *Registry) AllMetrics() []BreakerMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics := make([]BreakerMetrics, 0, len(r.breakers))
	for _, b := range r.breakers {
		metrics = append(metrics, b.Metrics())
	}
	return metrics
}

// ResetAll resets all breakers.
func (r *Registry) ResetAll() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, b := range r.breakers {
		b.Reset()
	}
}
