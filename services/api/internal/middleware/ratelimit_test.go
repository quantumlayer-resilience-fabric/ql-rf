package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

func TestRateLimit_AllowsRequests(t *testing.T) {
	log := logger.New("error", "json")
	cfg := RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 10,
		BurstSize:         20,
		CleanupInterval:   time.Minute,
	}

	middleware := RateLimit(cfg, log)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First 20 requests should succeed (burst size)
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i+1, rr.Code)
		}
	}
}

func TestRateLimit_BlocksExcessRequests(t *testing.T) {
	log := logger.New("error", "json")
	cfg := RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 10,
		BurstSize:         5,
		CleanupInterval:   time.Minute,
	}

	middleware := RateLimit(cfg, log)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust the burst
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.2:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	// Next request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", rr.Code)
	}

	// Check Retry-After header
	if rr.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header")
	}
}

func TestRateLimit_DifferentIPsAreSeparate(t *testing.T) {
	log := logger.New("error", "json")
	cfg := RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 10,
		BurstSize:         2,
		CleanupInterval:   time.Minute,
	}

	middleware := RateLimit(cfg, log)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust IP 1
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.3:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	// IP 2 should still be allowed
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.4:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for different IP, got %d", rr.Code)
	}
}

func TestRateLimit_Disabled(t *testing.T) {
	log := logger.New("error", "json")
	cfg := RateLimitConfig{
		Enabled: false,
	}

	middleware := RateLimit(cfg, log)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Should allow unlimited requests when disabled
	for i := 0; i < 1000; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.5:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200 when disabled, got %d", i+1, rr.Code)
		}
	}
}

func TestRateLimit_XForwardedFor(t *testing.T) {
	log := logger.New("error", "json")
	cfg := RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 10,
		BurstSize:         1,
		CleanupInterval:   time.Minute,
	}

	middleware := RateLimit(cfg, log)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request with X-Forwarded-For
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345" // Proxy IP
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	// Second request from same X-Forwarded-For should be rate limited
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", rr.Code)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xri        string
		expected   string
	}{
		{
			name:       "remote addr only",
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "x-forwarded-for single",
			remoteAddr: "10.0.0.1:12345",
			xff:        "203.0.113.1",
			expected:   "203.0.113.1",
		},
		{
			name:       "x-forwarded-for chain",
			remoteAddr: "10.0.0.1:12345",
			xff:        "203.0.113.1, 198.51.100.1, 10.0.0.1",
			expected:   "203.0.113.1",
		},
		{
			name:       "x-real-ip",
			remoteAddr: "10.0.0.1:12345",
			xri:        "203.0.113.2",
			expected:   "203.0.113.2",
		},
		{
			name:       "x-forwarded-for takes precedence",
			remoteAddr: "10.0.0.1:12345",
			xff:        "203.0.113.1",
			xri:        "203.0.113.2",
			expected:   "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}

			ip := getClientIP(req)
			if ip != tt.expected {
				t.Errorf("expected IP %s, got %s", tt.expected, ip)
			}
		})
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"a, b, c", []string{"a", "b", "c"}},
		{"  a  ,  b  ,  c  ", []string{"a", "b", "c"}},
		{"single", []string{"single"}},
		{"", nil},
		{"  ", nil},
	}

	for _, tt := range tests {
		result := splitCSV(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("splitCSV(%q): expected %v, got %v", tt.input, tt.expected, result)
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("splitCSV(%q)[%d]: expected %q, got %q", tt.input, i, tt.expected[i], result[i])
			}
		}
	}
}
