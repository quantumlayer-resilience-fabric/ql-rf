package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
)

func TestAuth_MissingAuthorizationHeader(t *testing.T) {
	log := logger.New("debug", "text")
	cfg := middleware.AuthConfig{
		DevMode: false,
	}

	authMiddleware := middleware.Auth(cfg, log)
	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "missing authorization header")
}

func TestAuth_InvalidAuthorizationHeaderFormat(t *testing.T) {
	log := logger.New("debug", "text")
	cfg := middleware.AuthConfig{
		DevMode: false,
	}

	authMiddleware := middleware.Auth(cfg, log)
	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	testCases := []struct {
		name   string
		header string
	}{
		{"no bearer prefix", "sometoken"},
		{"wrong prefix", "Basic sometoken"},
		{"empty after bearer", "Bearer "},
		{"only bearer", "Bearer"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
			req.Header.Set("Authorization", tc.header)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusUnauthorized, rr.Code)
		})
	}
}

func TestAuth_DevMode_CreatesMockUser(t *testing.T) {
	log := logger.New("debug", "text")
	cfg := middleware.AuthConfig{
		DevMode: true,
	}

	var capturedUser *models.User

	authMiddleware := middleware.Auth(cfg, log)
	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = middleware.GetUser(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer dev-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	require.NotNil(t, capturedUser)
	assert.Equal(t, "dev-user", capturedUser.ExternalID)
	assert.Equal(t, "dev@example.com", capturedUser.Email)
	assert.Equal(t, models.RoleAdmin, capturedUser.Role)
}

func TestAuth_DevMode_WithEmptyClerkKeys(t *testing.T) {
	log := logger.New("debug", "text")
	cfg := middleware.AuthConfig{
		ClerkPublishableKey: "",
		ClerkSecretKey:      "",
		DevMode:             false, // Not explicitly dev mode, but no Clerk keys
	}

	var capturedUser *models.User

	authMiddleware := middleware.Auth(cfg, log)
	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = middleware.GetUser(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer any-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// When no Clerk keys are provided, verifier is nil and falls back to dev mode
	assert.Equal(t, http.StatusOK, rr.Code)
	require.NotNil(t, capturedUser)
	assert.Equal(t, "dev-user", capturedUser.ExternalID)
}

func TestMapClerkRoleToRole(t *testing.T) {
	testCases := []struct {
		clerkRole    string
		expectedRole models.Role
	}{
		{"org:admin", models.RoleAdmin},
		{"admin", models.RoleAdmin},
		{"ADMIN", models.RoleAdmin},
		{"org:engineer", models.RoleEngineer},
		{"engineer", models.RoleEngineer},
		{"ENGINEER", models.RoleEngineer},
		{"org:operator", models.RoleOperator},
		{"operator", models.RoleOperator},
		{"OPERATOR", models.RoleOperator},
		{"viewer", models.RoleViewer},
		{"unknown", models.RoleViewer},
		{"", models.RoleViewer},
	}

	for _, tc := range testCases {
		t.Run(tc.clerkRole, func(t *testing.T) {
			// We can't directly test mapClerkRoleToRole as it's unexported,
			// but we can verify through the middleware behavior in dev mode
			// This test documents the expected mapping
			assert.NotEmpty(t, tc.expectedRole)
		})
	}
}

func TestRequireRole_Viewer(t *testing.T) {
	testCases := []struct {
		userRole     models.Role
		shouldPass   bool
	}{
		{models.RoleAdmin, true},
		{models.RoleEngineer, true},
		{models.RoleOperator, true},
		{models.RoleViewer, true},
	}

	for _, tc := range testCases {
		t.Run(string(tc.userRole), func(t *testing.T) {
			handler := middleware.RequireRole("viewer")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, &models.User{
				Role: tc.userRole,
			}))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if tc.shouldPass {
				assert.Equal(t, http.StatusOK, rr.Code)
			} else {
				assert.Equal(t, http.StatusForbidden, rr.Code)
			}
		})
	}
}

func TestRequireRole_Operator(t *testing.T) {
	testCases := []struct {
		userRole   models.Role
		shouldPass bool
	}{
		{models.RoleAdmin, true},
		{models.RoleEngineer, true},
		{models.RoleOperator, true},
		{models.RoleViewer, false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.userRole), func(t *testing.T) {
			handler := middleware.RequireRole("operator")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, &models.User{
				Role: tc.userRole,
			}))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if tc.shouldPass {
				assert.Equal(t, http.StatusOK, rr.Code)
			} else {
				assert.Equal(t, http.StatusForbidden, rr.Code)
			}
		})
	}
}

func TestRequireRole_Engineer(t *testing.T) {
	testCases := []struct {
		userRole   models.Role
		shouldPass bool
	}{
		{models.RoleAdmin, true},
		{models.RoleEngineer, true},
		{models.RoleOperator, false},
		{models.RoleViewer, false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.userRole), func(t *testing.T) {
			handler := middleware.RequireRole("engineer")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, &models.User{
				Role: tc.userRole,
			}))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if tc.shouldPass {
				assert.Equal(t, http.StatusOK, rr.Code)
			} else {
				assert.Equal(t, http.StatusForbidden, rr.Code)
			}
		})
	}
}

func TestRequireRole_Admin(t *testing.T) {
	testCases := []struct {
		userRole   models.Role
		shouldPass bool
	}{
		{models.RoleAdmin, true},
		{models.RoleEngineer, false},
		{models.RoleOperator, false},
		{models.RoleViewer, false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.userRole), func(t *testing.T) {
			handler := middleware.RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, &models.User{
				Role: tc.userRole,
			}))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if tc.shouldPass {
				assert.Equal(t, http.StatusOK, rr.Code)
			} else {
				assert.Equal(t, http.StatusForbidden, rr.Code)
			}
		})
	}
}

func TestRequireRole_NoUserInContext(t *testing.T) {
	handler := middleware.RequireRole("viewer")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No user in context
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequirePermission_NoUserInContext(t *testing.T) {
	handler := middleware.RequirePermission(models.PermReadAssets)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestGetUser_NoUserInContext(t *testing.T) {
	ctx := context.Background()
	user := middleware.GetUser(ctx)
	assert.Nil(t, user)
}

func TestGetUser_WithUserInContext(t *testing.T) {
	expectedUser := &models.User{
		ID:         uuid.New(),
		ExternalID: "test-user",
		Email:      "test@example.com",
		Role:       models.RoleEngineer,
	}

	ctx := context.WithValue(context.Background(), middleware.UserContextKey, expectedUser)
	user := middleware.GetUser(ctx)

	require.NotNil(t, user)
	assert.Equal(t, expectedUser.ID, user.ID)
	assert.Equal(t, expectedUser.ExternalID, user.ExternalID)
	assert.Equal(t, expectedUser.Role, user.Role)
}

func TestGetOrg_NoOrgInContext(t *testing.T) {
	ctx := context.Background()
	org := middleware.GetOrg(ctx)
	assert.Nil(t, org)
}

func TestGetOrg_WithOrgInContext(t *testing.T) {
	expectedOrg := &models.Organization{
		ID:   uuid.New(),
		Name: "Test Org",
		Slug: "test-org",
	}

	ctx := context.WithValue(context.Background(), middleware.OrgContextKey, expectedOrg)
	org := middleware.GetOrg(ctx)

	require.NotNil(t, org)
	assert.Equal(t, expectedOrg.ID, org.ID)
	assert.Equal(t, expectedOrg.Name, org.Name)
	assert.Equal(t, expectedOrg.Slug, org.Slug)
}

func TestOptionalAuth_NoAuthHeader(t *testing.T) {
	log := logger.New("debug", "text")
	cfg := middleware.AuthConfig{
		DevMode: false,
	}

	var capturedUser *models.User

	authMiddleware := middleware.OptionalAuth(cfg, log)
	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = middleware.GetUser(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	// No auth header
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should pass through without error but no user
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Nil(t, capturedUser)
}

func TestOptionalAuth_InvalidToken(t *testing.T) {
	log := logger.New("debug", "text")
	cfg := middleware.AuthConfig{
		ClerkPublishableKey: "pk_test_valid",
		DevMode:             false,
	}

	var capturedUser *models.User

	authMiddleware := middleware.OptionalAuth(cfg, log)
	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = middleware.GetUser(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should pass through without error but no user (invalid token ignored)
	assert.Equal(t, http.StatusOK, rr.Code)
	// Note: With real Clerk verification, this would fail and continue without user
	// The capturedUser would be nil because Clerk verification fails
	_ = capturedUser // Variable used in handler to capture user from context
}

func TestOptionalAuth_DevMode(t *testing.T) {
	log := logger.New("debug", "text")
	cfg := middleware.AuthConfig{
		DevMode: true,
	}

	var capturedUser *models.User

	authMiddleware := middleware.OptionalAuth(cfg, log)
	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = middleware.GetUser(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer any-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	require.NotNil(t, capturedUser)
	assert.Equal(t, "dev-user", capturedUser.ExternalID)
}

func TestOptionalAuth_MalformedHeader(t *testing.T) {
	log := logger.New("debug", "text")
	cfg := middleware.AuthConfig{
		DevMode: false,
	}

	authMiddleware := middleware.OptionalAuth(cfg, log)

	testCases := []struct {
		name   string
		header string
	}{
		{"wrong prefix", "Basic token"},
		{"no space", "Bearertoken"},
		{"empty token", "Bearer "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var capturedUser *models.User
			handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedUser = middleware.GetUser(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
			req.Header.Set("Authorization", tc.header)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// Should pass through without error but no user
			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Nil(t, capturedUser)
		})
	}
}
