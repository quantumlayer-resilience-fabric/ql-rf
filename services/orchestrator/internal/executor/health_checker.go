// Package executor implements the plan execution engine.
package executor

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// HealthChecker performs health checks for assets and services.
type HealthChecker struct {
	httpClient *http.Client
	log        *logger.Logger
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(log *logger.Logger) *HealthChecker {
	return &HealthChecker{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false, // For production, require valid certs
				},
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				ResponseHeaderTimeout: 10 * time.Second,
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   10,
				IdleConnTimeout:       90 * time.Second,
			},
		},
		log: log.WithComponent("health-checker"),
	}
}

// HealthCheckResult represents the result of a health check.
type HealthCheckResult struct {
	Name       string        `json:"name"`
	Type       string        `json:"type"`
	Target     string        `json:"target"`
	Success    bool          `json:"success"`
	StatusCode int           `json:"status_code,omitempty"`
	Response   string        `json:"response,omitempty"`
	Error      string        `json:"error,omitempty"`
	Duration   time.Duration `json:"duration"`
	Timestamp  time.Time     `json:"timestamp"`
}

// Check performs a health check based on the check type.
func (h *HealthChecker) Check(ctx context.Context, hc *HealthCheck) (*HealthCheckResult, error) {
	result := &HealthCheckResult{
		Name:      hc.Name,
		Type:      hc.Type,
		Target:    hc.Target,
		Timestamp: time.Now(),
	}

	start := time.Now()
	defer func() {
		result.Duration = time.Since(start)
	}()

	var err error
	switch strings.ToLower(hc.Type) {
	case "http", "https":
		err = h.checkHTTP(ctx, hc, result)
	case "tcp":
		err = h.checkTCP(ctx, hc, result)
	case "command", "exec":
		err = h.checkCommand(ctx, hc, result)
	case "dns":
		err = h.checkDNS(ctx, hc, result)
	default:
		err = fmt.Errorf("unsupported health check type: %s", hc.Type)
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, err
	}

	result.Success = true
	return result, nil
}

// checkHTTP performs an HTTP health check.
func (h *HealthChecker) checkHTTP(ctx context.Context, hc *HealthCheck, result *HealthCheckResult) error {
	target := hc.Target
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		if hc.Type == "https" {
			target = "https://" + target
		} else {
			target = "http://" + target
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add custom headers if needed
	req.Header.Set("User-Agent", "QL-RF-HealthChecker/1.0")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	// Read response body (limited to 1KB for safety)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err == nil {
		result.Response = string(body)
	}

	// Check expected status code
	expectedStatus := 200
	if hc.Expected != "" {
		if _, err := fmt.Sscanf(hc.Expected, "%d", &expectedStatus); err != nil {
			// Expected might be a response body check
			if !strings.Contains(result.Response, hc.Expected) {
				return fmt.Errorf("response does not contain expected string: %s", hc.Expected)
			}
			expectedStatus = resp.StatusCode // Accept any status if body check passes
		}
	}

	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("unexpected status code: got %d, expected %d", resp.StatusCode, expectedStatus)
	}

	h.log.Debug("HTTP health check passed",
		"target", target,
		"status", resp.StatusCode,
		"duration", result.Duration,
	)

	return nil
}

// checkTCP performs a TCP connection health check.
func (h *HealthChecker) checkTCP(ctx context.Context, hc *HealthCheck, result *HealthCheckResult) error {
	timeout := 10 * time.Second
	if hc.Timeout != "" {
		if d, err := time.ParseDuration(hc.Timeout); err == nil {
			timeout = d
		}
	}

	// Parse target as host:port
	target := hc.Target
	if !strings.Contains(target, ":") {
		return fmt.Errorf("TCP target must be in format host:port, got: %s", target)
	}

	dialer := &net.Dialer{
		Timeout: timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", target)
	if err != nil {
		return fmt.Errorf("TCP connection failed: %w", err)
	}
	defer conn.Close()

	// If expected is set, try to read from the connection
	if hc.Expected != "" {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			// Some services don't send data until we send first
			// Consider connection successful if we connected
			h.log.Debug("could not read banner, but connection successful", "target", target)
		} else if n > 0 {
			result.Response = string(buf[:n])
			if !strings.Contains(result.Response, hc.Expected) {
				return fmt.Errorf("banner does not contain expected string: %s", hc.Expected)
			}
		}
	}

	h.log.Debug("TCP health check passed",
		"target", target,
		"duration", result.Duration,
	)

	return nil
}

// checkCommand executes a command and checks the exit code.
func (h *HealthChecker) checkCommand(ctx context.Context, hc *HealthCheck, result *HealthCheckResult) error {
	timeout := 30 * time.Second
	if hc.Timeout != "" {
		if d, err := time.ParseDuration(hc.Timeout); err == nil {
			timeout = d
		}
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Parse command - target is the command to run
	parts := strings.Fields(hc.Target)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(cmdCtx, parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()
	result.Response = string(output)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("command exited with code %d: %s", exitErr.ExitCode(), string(output))
		}
		return fmt.Errorf("command failed: %w", err)
	}

	// Check expected output if specified
	if hc.Expected != "" && !strings.Contains(result.Response, hc.Expected) {
		return fmt.Errorf("output does not contain expected string: %s", hc.Expected)
	}

	h.log.Debug("command health check passed",
		"command", parts[0],
		"duration", result.Duration,
	)

	return nil
}

// checkDNS performs a DNS resolution health check.
func (h *HealthChecker) checkDNS(ctx context.Context, hc *HealthCheck, result *HealthCheckResult) error {
	timeout := 10 * time.Second
	if hc.Timeout != "" {
		if d, err := time.ParseDuration(hc.Timeout); err == nil {
			timeout = d
		}
	}

	resolver := &net.Resolver{
		PreferGo: true,
	}

	dnsCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addrs, err := resolver.LookupHost(dnsCtx, hc.Target)
	if err != nil {
		return fmt.Errorf("DNS lookup failed: %w", err)
	}

	if len(addrs) == 0 {
		return fmt.Errorf("no addresses returned for %s", hc.Target)
	}

	result.Response = strings.Join(addrs, ", ")

	// Check expected if specified (should match one of the resolved addresses)
	if hc.Expected != "" {
		found := false
		for _, addr := range addrs {
			if addr == hc.Expected {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("expected address %s not found in resolved addresses", hc.Expected)
		}
	}

	h.log.Debug("DNS health check passed",
		"target", hc.Target,
		"addresses", result.Response,
		"duration", result.Duration,
	)

	return nil
}

// CheckWithRetry performs a health check with retries.
func (h *HealthChecker) CheckWithRetry(ctx context.Context, hc *HealthCheck) (*HealthCheckResult, error) {
	retries := hc.Retries
	if retries <= 0 {
		retries = 3
	}

	var lastResult *HealthCheckResult
	var lastErr error

	for i := 0; i < retries; i++ {
		if i > 0 {
			h.log.Debug("retrying health check",
				"name", hc.Name,
				"attempt", i+1,
				"max_retries", retries,
			)
			// Wait before retry with exponential backoff
			backoff := time.Duration(1<<uint(i)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return lastResult, ctx.Err()
			}
		}

		result, err := h.Check(ctx, hc)
		lastResult = result
		lastErr = err

		if err == nil {
			return result, nil
		}

		h.log.Warn("health check failed",
			"name", hc.Name,
			"attempt", i+1,
			"error", err,
		)
	}

	return lastResult, fmt.Errorf("health check failed after %d retries: %w", retries, lastErr)
}
