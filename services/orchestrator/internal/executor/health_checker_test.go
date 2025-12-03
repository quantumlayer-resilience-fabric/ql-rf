package executor

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

func TestHealthChecker_HTTP(t *testing.T) {
	log := logger.New("debug", "text")
	hc := NewHealthChecker(log)

	t.Run("successful HTTP check", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))
		defer server.Close()

		check := &HealthCheck{
			Name:   "test-http",
			Type:   "http",
			Target: server.URL,
		}

		result, err := hc.Check(context.Background(), check)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.Contains(t, result.Response, "OK")
	})

	t.Run("HTTP check with expected status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		check := &HealthCheck{
			Name:     "test-http-201",
			Type:     "http",
			Target:   server.URL,
			Expected: "201",
		}

		result, err := hc.Check(context.Background(), check)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, http.StatusCreated, result.StatusCode)
	})

	t.Run("HTTP check with expected body content", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy","service":"api"}`))
		}))
		defer server.Close()

		check := &HealthCheck{
			Name:     "test-http-body",
			Type:     "http",
			Target:   server.URL,
			Expected: "healthy",
		}

		result, err := hc.Check(context.Background(), check)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Response, "healthy")
	})

	t.Run("HTTP check failure - wrong status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		check := &HealthCheck{
			Name:   "test-http-fail",
			Type:   "http",
			Target: server.URL,
		}

		result, err := hc.Check(context.Background(), check)
		require.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, err.Error(), "unexpected status code")
	})

	t.Run("HTTP check failure - connection refused", func(t *testing.T) {
		check := &HealthCheck{
			Name:    "test-http-refused",
			Type:    "http",
			Target:  "http://127.0.0.1:1", // Port 1 should be unused
			Timeout: "2s",
		}

		result, err := hc.Check(context.Background(), check)
		require.Error(t, err)
		assert.False(t, result.Success)
	})
}

func TestHealthChecker_TCP(t *testing.T) {
	log := logger.New("debug", "text")
	hc := NewHealthChecker(log)

	t.Run("successful TCP check", func(t *testing.T) {
		// Start a TCP server
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		go func() {
			conn, _ := listener.Accept()
			if conn != nil {
				conn.Write([]byte("HELLO"))
				conn.Close()
			}
		}()

		check := &HealthCheck{
			Name:    "test-tcp",
			Type:    "tcp",
			Target:  listener.Addr().String(),
			Timeout: "5s",
		}

		result, err := hc.Check(context.Background(), check)
		require.NoError(t, err)
		assert.True(t, result.Success)
	})

	t.Run("TCP check with expected banner", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		go func() {
			conn, _ := listener.Accept()
			if conn != nil {
				conn.Write([]byte("SSH-2.0-OpenSSH"))
				conn.Close()
			}
		}()

		check := &HealthCheck{
			Name:     "test-tcp-banner",
			Type:     "tcp",
			Target:   listener.Addr().String(),
			Expected: "SSH",
			Timeout:  "5s",
		}

		result, err := hc.Check(context.Background(), check)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Response, "SSH")
	})

	t.Run("TCP check failure - connection refused", func(t *testing.T) {
		check := &HealthCheck{
			Name:    "test-tcp-refused",
			Type:    "tcp",
			Target:  "127.0.0.1:1",
			Timeout: "2s",
		}

		result, err := hc.Check(context.Background(), check)
		require.Error(t, err)
		assert.False(t, result.Success)
	})

	t.Run("TCP check failure - missing port", func(t *testing.T) {
		check := &HealthCheck{
			Name:   "test-tcp-no-port",
			Type:   "tcp",
			Target: "127.0.0.1",
		}

		result, err := hc.Check(context.Background(), check)
		require.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, err.Error(), "host:port")
	})
}

func TestHealthChecker_Command(t *testing.T) {
	log := logger.New("debug", "text")
	hc := NewHealthChecker(log)

	t.Run("successful command check", func(t *testing.T) {
		check := &HealthCheck{
			Name:   "test-cmd",
			Type:   "command",
			Target: "echo hello",
		}

		result, err := hc.Check(context.Background(), check)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Response, "hello")
	})

	t.Run("command check with expected output", func(t *testing.T) {
		check := &HealthCheck{
			Name:     "test-cmd-expected",
			Type:     "command",
			Target:   "echo success",
			Expected: "success",
		}

		result, err := hc.Check(context.Background(), check)
		require.NoError(t, err)
		assert.True(t, result.Success)
	})

	t.Run("command check failure - non-zero exit", func(t *testing.T) {
		check := &HealthCheck{
			Name:   "test-cmd-fail",
			Type:   "command",
			Target: "exit 1",
		}

		// This will execute "exit" which is a shell builtin
		// We need to use sh -c for this
		check.Target = "sh -c 'exit 1'"

		result, err := hc.Check(context.Background(), check)
		require.Error(t, err)
		assert.False(t, result.Success)
	})

	t.Run("command check failure - missing expected", func(t *testing.T) {
		check := &HealthCheck{
			Name:     "test-cmd-missing",
			Type:     "command",
			Target:   "echo hello",
			Expected: "world",
		}

		result, err := hc.Check(context.Background(), check)
		require.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, err.Error(), "does not contain expected")
	})
}

func TestHealthChecker_DNS(t *testing.T) {
	log := logger.New("debug", "text")
	hc := NewHealthChecker(log)

	t.Run("successful DNS check", func(t *testing.T) {
		check := &HealthCheck{
			Name:   "test-dns",
			Type:   "dns",
			Target: "localhost",
		}

		result, err := hc.Check(context.Background(), check)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.NotEmpty(t, result.Response)
	})

	t.Run("DNS check failure - non-existent domain", func(t *testing.T) {
		check := &HealthCheck{
			Name:    "test-dns-fail",
			Type:    "dns",
			Target:  "this-domain-definitely-does-not-exist-12345.invalid",
			Timeout: "5s",
		}

		result, err := hc.Check(context.Background(), check)
		require.Error(t, err)
		assert.False(t, result.Success)
	})
}

func TestHealthChecker_CheckWithRetry(t *testing.T) {
	log := logger.New("debug", "text")
	hc := NewHealthChecker(log)

	t.Run("succeeds after retries", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		check := &HealthCheck{
			Name:    "test-retry",
			Type:    "http",
			Target:  server.URL,
			Retries: 5,
		}

		result, err := hc.CheckWithRetry(context.Background(), check)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 3, attempts)
	})

	t.Run("fails after max retries", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		check := &HealthCheck{
			Name:    "test-retry-fail",
			Type:    "http",
			Target:  server.URL,
			Retries: 2,
		}

		result, err := hc.CheckWithRetry(context.Background(), check)
		require.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, err.Error(), "failed after 2 retries")
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		check := &HealthCheck{
			Name:    "test-retry-cancel",
			Type:    "http",
			Target:  server.URL,
			Retries: 10,
		}

		_, err := hc.CheckWithRetry(ctx, check)
		require.Error(t, err)
	})
}

func TestHealthChecker_UnsupportedType(t *testing.T) {
	log := logger.New("debug", "text")
	hc := NewHealthChecker(log)

	check := &HealthCheck{
		Name:   "test-unsupported",
		Type:   "unknown",
		Target: "something",
	}

	result, err := hc.Check(context.Background(), check)
	require.Error(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, err.Error(), "unsupported health check type")
}
