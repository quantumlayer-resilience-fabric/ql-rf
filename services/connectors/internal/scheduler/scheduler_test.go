package scheduler

import (
	"testing"
	"time"
)

func TestParseScheduleInterval(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		expected time.Duration
	}{
		{
			name:     "hours",
			schedule: "1h",
			expected: time.Hour,
		},
		{
			name:     "minutes",
			schedule: "30m",
			expected: 30 * time.Minute,
		},
		{
			name:     "complex duration",
			schedule: "2h30m",
			expected: 2*time.Hour + 30*time.Minute,
		},
		{
			name:     "invalid falls back to 1h",
			schedule: "invalid",
			expected: time.Hour,
		},
		{
			name:     "empty falls back to 1h",
			schedule: "",
			expected: time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseScheduleInterval(tt.schedule)
			if got != tt.expected {
				t.Errorf("parseScheduleInterval(%q) = %v, want %v", tt.schedule, got, tt.expected)
			}
		})
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected string
	}{
		{
			name:     "existing key",
			m:        map[string]interface{}{"foo": "bar"},
			key:      "foo",
			expected: "bar",
		},
		{
			name:     "missing key",
			m:        map[string]interface{}{},
			key:      "foo",
			expected: "",
		},
		{
			name:     "non-string value",
			m:        map[string]interface{}{"foo": 123},
			key:      "foo",
			expected: "",
		},
		{
			name:     "nil map",
			m:        nil,
			key:      "foo",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getString(tt.m, tt.key)
			if got != tt.expected {
				t.Errorf("getString(%v, %q) = %q, want %q", tt.m, tt.key, got, tt.expected)
			}
		})
	}
}

func TestHasScheme(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com", true},
		{"http://example.com", true},
		{"example.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := hasScheme(tt.url)
			if got != tt.expected {
				t.Errorf("hasScheme(%q) = %v, want %v", tt.url, got, tt.expected)
			}
		})
	}
}

func TestHasPath(t *testing.T) {
	tests := []struct {
		url      string
		path     string
		expected bool
	}{
		{"https://vcenter.example.com/sdk", "/sdk", true},
		{"https://vcenter.example.com", "/sdk", false},
		{"https://vcenter.example.com/api", "/sdk", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := hasPath(tt.url, tt.path)
			if got != tt.expected {
				t.Errorf("hasPath(%q, %q) = %v, want %v", tt.url, tt.path, got, tt.expected)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.PollInterval != 30*time.Second {
		t.Errorf("DefaultConfig().PollInterval = %v, want %v", cfg.PollInterval, 30*time.Second)
	}

	if cfg.MaxConcurrent != 5 {
		t.Errorf("DefaultConfig().MaxConcurrent = %d, want %d", cfg.MaxConcurrent, 5)
	}
}
