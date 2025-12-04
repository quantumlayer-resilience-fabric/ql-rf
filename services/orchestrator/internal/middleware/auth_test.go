package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

func TestContextKey(t *testing.T) {
	assert.Equal(t, ContextKey("user_id"), UserIDKey)
	assert.Equal(t, ContextKey("org_id"), OrgIDKey)
	assert.Equal(t, ContextKey("email"), EmailKey)
	assert.Equal(t, ContextKey("role"), RoleKey)
}

func TestAuthConfig(t *testing.T) {
	cfg := AuthConfig{
		ClerkPublishableKey: "pk_test_xxx",
		DevMode:             true,
	}

	assert.Equal(t, "pk_test_xxx", cfg.ClerkPublishableKey)
	assert.True(t, cfg.DevMode)
}

func TestAuth_DevMode(t *testing.T) {
	log := logger.New("error", "text")
	cfg := AuthConfig{DevMode: true}

	middleware := Auth(cfg, log)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
		orgID := GetOrgID(r.Context())
		email := GetEmail(r.Context())
		role := GetRole(r.Context())

		assert.Equal(t, "dev-user", userID)
		assert.Equal(t, "00000000-0000-0000-0000-000000000001", orgID)
		assert.Equal(t, "dev@example.com", email)
		assert.Equal(t, "admin", role)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer dev-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuth_MissingAuthHeader(t *testing.T) {
	log := logger.New("error", "text")
	cfg := AuthConfig{DevMode: false}

	middleware := Auth(cfg, log)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "missing authorization header")
}

func TestAuth_InvalidAuthHeaderFormat(t *testing.T) {
	log := logger.New("error", "text")
	cfg := AuthConfig{DevMode: false}

	middleware := Auth(cfg, log)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name   string
		header string
	}{
		{"no bearer", "Basic abc123"},
		{"no space", "Bearerabc123"},
		{"single word", "token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tt.header)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusUnauthorized, rr.Code)
		})
	}
}

func TestAuth_MissingToken(t *testing.T) {
	log := logger.New("error", "text")
	cfg := AuthConfig{DevMode: false}

	middleware := Auth(cfg, log)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer ")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "missing token")
}

func TestOptionalAuth_NoAuth_DevMode(t *testing.T) {
	log := logger.New("error", "text")
	cfg := AuthConfig{DevMode: true}

	middleware := OptionalAuth(cfg, log)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
		orgID := GetOrgID(r.Context())

		assert.Equal(t, "dev-user", userID)
		assert.Equal(t, "00000000-0000-0000-0000-000000000001", orgID)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestOptionalAuth_WithAuth_DevMode(t *testing.T) {
	log := logger.New("error", "text")
	cfg := AuthConfig{DevMode: true}

	middleware := OptionalAuth(cfg, log)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
		assert.Equal(t, "dev-user", userID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestOptionalAuth_InvalidAuth_DevMode(t *testing.T) {
	log := logger.New("error", "text")
	cfg := AuthConfig{DevMode: true}

	middleware := OptionalAuth(cfg, log)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
		assert.Equal(t, "dev-user", userID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Invalid")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestGetUserID(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), UserIDKey, "user-123")
		assert.Equal(t, "user-123", GetUserID(ctx))
	})

	t.Run("without value", func(t *testing.T) {
		assert.Equal(t, "", GetUserID(context.Background()))
	})

	t.Run("wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), UserIDKey, 123)
		assert.Equal(t, "", GetUserID(ctx))
	})
}

func TestGetOrgID(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), OrgIDKey, "org-123")
		assert.Equal(t, "org-123", GetOrgID(ctx))
	})

	t.Run("without value", func(t *testing.T) {
		assert.Equal(t, "", GetOrgID(context.Background()))
	})
}

func TestGetOrgUUID(t *testing.T) {
	t.Run("valid UUID", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), OrgIDKey, "11111111-1111-1111-1111-111111111111")
		id := GetOrgUUID(ctx)
		assert.Equal(t, uuid.MustParse("11111111-1111-1111-1111-111111111111"), id)
	})

	t.Run("empty org ID", func(t *testing.T) {
		id := GetOrgUUID(context.Background())
		assert.Equal(t, uuid.Nil, id)
	})

	t.Run("invalid UUID", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), OrgIDKey, "not-a-uuid")
		id := GetOrgUUID(ctx)
		assert.Equal(t, uuid.Nil, id)
	})
}

func TestGetEmail(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), EmailKey, "test@example.com")
		assert.Equal(t, "test@example.com", GetEmail(ctx))
	})

	t.Run("without value", func(t *testing.T) {
		assert.Equal(t, "", GetEmail(context.Background()))
	})
}

func TestGetRole(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), RoleKey, "admin")
		assert.Equal(t, "admin", GetRole(ctx))
	})

	t.Run("without value", func(t *testing.T) {
		assert.Equal(t, "", GetRole(context.Background()))
	})
}

func TestRequireRole(t *testing.T) {
	t.Run("admin has access to everything", func(t *testing.T) {
		middleware := RequireRole("viewer")

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), RoleKey, "admin")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("matching role has access", func(t *testing.T) {
		middleware := RequireRole("editor")

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), RoleKey, "editor")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("non-matching role denied", func(t *testing.T) {
		middleware := RequireRole("admin")

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), RoleKey, "viewer")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)

		var response map[string]string
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "insufficient permissions", response["error"])
		assert.Equal(t, "admin", response["required"])
		assert.Equal(t, "viewer", response["current"])
	})

	t.Run("no role returns unauthorized", func(t *testing.T) {
		middleware := RequireRole("admin")

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestRequirePermission(t *testing.T) {
	// Note: This requires the models.Permission and models.Role types
	// to be properly implemented. For now, we just test basic functionality.

	t.Run("no role returns unauthorized", func(t *testing.T) {
		middleware := RequirePermission("read")

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestMiddlewareChaining(t *testing.T) {
	log := logger.New("error", "text")
	authCfg := AuthConfig{DevMode: true}

	// Chain auth -> require role
	authMiddleware := Auth(authCfg, log)
	roleMiddleware := RequireRole("admin")

	handler := authMiddleware(roleMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer dev-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "success", rr.Body.String())
}

func TestContextValuePreservation(t *testing.T) {
	log := logger.New("error", "text")
	cfg := AuthConfig{DevMode: true}

	middleware := Auth(cfg, log)

	var capturedCtx context.Context

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer dev-token")

	// Add a custom value to context
	ctx := context.WithValue(req.Context(), "custom_key", "custom_value")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Verify both custom value and auth values are preserved
	assert.Equal(t, "custom_value", capturedCtx.Value("custom_key"))
	assert.Equal(t, "dev-user", capturedCtx.Value(UserIDKey))
}
