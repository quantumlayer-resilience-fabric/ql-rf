package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// RateLimitConfig holds configuration for the rate limiter middleware.
type RateLimitConfig struct {
	// Enabled controls whether rate limiting is active
	Enabled bool
	// RequestsPerSecond is the allowed requests per second per IP
	RequestsPerSecond float64
	// BurstSize is the maximum burst size
	BurstSize int
	// CleanupInterval is how often to clean up old entries
	CleanupInterval time.Duration
}

// DefaultRateLimitConfig returns sensible defaults for rate limiting.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 100, // 100 requests per second
		BurstSize:         200, // Allow bursts up to 200
		CleanupInterval:   time.Minute,
	}
}

// tokenBucket implements a simple token bucket rate limiter.
type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

// rateLimiter stores per-IP rate limiters.
type rateLimiter struct {
	buckets  map[string]*tokenBucket
	mu       sync.RWMutex
	rate     float64
	burst    int
	log      *logger.Logger
	stopChan chan struct{}
}

// newRateLimiter creates a new rate limiter.
func newRateLimiter(cfg RateLimitConfig, log *logger.Logger) *rateLimiter {
	rl := &rateLimiter{
		buckets:  make(map[string]*tokenBucket),
		rate:     cfg.RequestsPerSecond,
		burst:    cfg.BurstSize,
		log:      log.WithComponent("rate-limiter"),
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanup(cfg.CleanupInterval)

	return rl
}

// allow checks if a request from the given IP is allowed.
func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	bucket, exists := rl.buckets[ip]
	if !exists {
		bucket = &tokenBucket{
			tokens:     float64(rl.burst),
			lastRefill: time.Now(),
		}
		rl.buckets[ip] = bucket
	}
	rl.mu.Unlock()

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.lastRefill = now

	// Refill tokens based on elapsed time
	bucket.tokens += elapsed * rl.rate
	if bucket.tokens > float64(rl.burst) {
		bucket.tokens = float64(rl.burst)
	}

	// Check if we have tokens available
	if bucket.tokens < 1 {
		return false
	}

	bucket.tokens--
	return true
}

// cleanup removes old entries periodically.
func (rl *rateLimiter) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, bucket := range rl.buckets {
				bucket.mu.Lock()
				// Remove entries that haven't been accessed in 5 minutes
				if now.Sub(bucket.lastRefill) > 5*time.Minute {
					delete(rl.buckets, ip)
				}
				bucket.mu.Unlock()
			}
			rl.mu.Unlock()
		case <-rl.stopChan:
			return
		}
	}
}

// stop stops the cleanup goroutine.
func (rl *rateLimiter) stop() {
	close(rl.stopChan)
}

// RateLimit returns a middleware that limits requests per IP.
func RateLimit(cfg RateLimitConfig, log *logger.Logger) func(next http.Handler) http.Handler {
	if !cfg.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	rl := newRateLimiter(cfg, log)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP
			ip := getClientIP(r)

			if !rl.allow(ip) {
				rl.log.Warn("rate limit exceeded", "ip", ip, "path", r.URL.Path)
				w.Header().Set("Retry-After", "1")
				http.Error(w, `{"error": "rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (set by proxies/load balancers)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the chain
		ips := splitCSV(xff)
		if len(ips) > 0 {
			return ips[0]
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	// RemoteAddr is in the format "IP:port"
	ip := r.RemoteAddr
	for i := len(ip) - 1; i >= 0; i-- {
		if ip[i] == ':' {
			return ip[:i]
		}
	}
	return ip
}

// splitCSV splits a comma-separated string and trims whitespace.
func splitCSV(s string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := s[start:i]
			// Trim whitespace
			for len(part) > 0 && part[0] == ' ' {
				part = part[1:]
			}
			for len(part) > 0 && part[len(part)-1] == ' ' {
				part = part[:len(part)-1]
			}
			if len(part) > 0 {
				result = append(result, part)
			}
			start = i + 1
		}
	}
	return result
}
