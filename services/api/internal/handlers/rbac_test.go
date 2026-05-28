package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/rbac"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/handlers"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
)

// MockRBACService is a mock implementation of the RBAC service for testing.
// Since rbac.Service uses concrete *sql.DB, we create handlers tests that
// verify request handling without requiring a real database.

func TestRBACHandler_ListPermissions(t *testing.T) {
	log := logger.New("error", "text")
	// We can create handler with nil service for endpoints that don't use it
	h := handlers.NewRBACHandler(nil, log)

	t.Run("returns system permissions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/rbac/permissions", nil)
		req = withOrgContext(req)
		rr := httptest.NewRecorder()

		h.ListPermissions(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp map[string]interface{}
		err := json.NewDecoder(rr.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "permissions")

		permissions := resp["permissions"].([]interface{})
		assert.NotEmpty(t, permissions)

		// Check that at least one permission has expected structure
		firstPerm := permissions[0].(map[string]interface{})
		assert.Contains(t, firstPerm, "id")
		assert.Contains(t, firstPerm, "name")
		assert.Contains(t, firstPerm, "resource_type")
		assert.Contains(t, firstPerm, "action")
		assert.Contains(t, firstPerm, "is_system")
	})
}

func TestRBACHandler_ListRoles_NoOrg(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rbac/roles", nil)
	// No org context
	rr := httptest.NewRecorder()

	h.ListRoles(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "organization not found")
}

func TestRBACHandler_GetRole_InvalidID(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	// Use chi context to set URL params
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rbac/roles/invalid-uuid", nil)
	req = withOrgContext(req)

	// Add chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("roleId", "invalid-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	h.GetRole(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid role ID")
}

func TestRBACHandler_GetRole_NoOrg(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rbac/roles/"+uuid.New().String(), nil)
	// No org context

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("roleId", uuid.New().String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	h.GetRole(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "organization not found")
}

func TestRBACHandler_GetUserRoles_NoOrg(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rbac/users/test-user/roles", nil)
	// No org context

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", "test-user")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	h.GetUserRoles(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "organization not found")
}

func TestRBACHandler_GetUserRoles_MissingUserID(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rbac/users//roles", nil)
	req = withOrgContext(req)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", "") // Empty user ID
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	h.GetUserRoles(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "user ID is required")
}

func TestRBACHandler_AssignRole_NoOrg(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	body := map[string]string{"role_id": uuid.New().String()}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rbac/users/test-user/roles", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	// No org context

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", "test-user")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	h.AssignRole(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "organization not found")
}

func TestRBACHandler_AssignRole_NoUser(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	body := map[string]string{"role_id": uuid.New().String()}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rbac/users/test-user/roles", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Add org but no user
	ctx := context.WithValue(req.Context(), middleware.OrgContextKey, testOrg())

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", "test-user")
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	h.AssignRole(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "user not found")
}

func TestRBACHandler_AssignRole_MissingUserID(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	body := map[string]string{"role_id": uuid.New().String()}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rbac/users//roles", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withOrgContext(req)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", "") // Empty
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	h.AssignRole(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "user ID is required")
}

func TestRBACHandler_AssignRole_InvalidBody(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rbac/users/test-user/roles", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req = withOrgContext(req)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", "test-user")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	h.AssignRole(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid request body")
}

func TestRBACHandler_AssignRole_InvalidRoleID(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	body := map[string]string{"role_id": "not-a-uuid"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rbac/users/test-user/roles", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withOrgContext(req)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", "test-user")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	h.AssignRole(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid role ID")
}

func TestRBACHandler_RevokeRole_NoOrg(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/rbac/users/test-user/roles/"+uuid.New().String(), nil)
	// No org context

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", "test-user")
	rctx.URLParams.Add("roleId", uuid.New().String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	h.RevokeRole(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "organization not found")
}

func TestRBACHandler_RevokeRole_NoUser(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/rbac/users/test-user/roles/"+uuid.New().String(), nil)

	// Add org but no user
	ctx := context.WithValue(req.Context(), middleware.OrgContextKey, testOrg())

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", "test-user")
	rctx.URLParams.Add("roleId", uuid.New().String())
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	h.RevokeRole(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "user not found")
}

func TestRBACHandler_RevokeRole_MissingParams(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	testCases := []struct {
		name    string
		userId  string
		roleId  string
		wantErr string
	}{
		{"missing user ID", "", uuid.New().String(), "user ID and role ID are required"},
		{"missing role ID", "test-user", "", "user ID and role ID are required"},
		{"both missing", "", "", "user ID and role ID are required"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/rbac/users/"+tc.userId+"/roles/"+tc.roleId, nil)
			req = withOrgContext(req)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userId", tc.userId)
			rctx.URLParams.Add("roleId", tc.roleId)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()

			h.RevokeRole(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.wantErr)
		})
	}
}

func TestRBACHandler_RevokeRole_InvalidRoleID(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/rbac/users/test-user/roles/invalid-uuid", nil)
	req = withOrgContext(req)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", "test-user")
	rctx.URLParams.Add("roleId", "invalid-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	h.RevokeRole(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid role ID")
}

func TestRBACHandler_GetUserPermissions_NoOrg(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rbac/users/test-user/permissions", nil)
	// No org context

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", "test-user")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	h.GetUserPermissions(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "organization not found")
}

func TestRBACHandler_GetUserPermissions_MissingUserID(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rbac/users//permissions", nil)
	req = withOrgContext(req)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	h.GetUserPermissions(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "user ID is required")
}

func TestRBACHandler_CheckPermission_NoOrg(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	body := map[string]string{
		"user_id":       "test-user",
		"resource_type": "assets",
		"action":        "read",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rbac/check-permission", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	// No org context

	rr := httptest.NewRecorder()

	h.CheckPermission(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "organization not found")
}

func TestRBACHandler_CheckPermission_InvalidBody(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rbac/check-permission", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req = withOrgContext(req)

	rr := httptest.NewRecorder()

	h.CheckPermission(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid request body")
}

func TestRBACHandler_CheckPermission_MissingFields(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	testCases := []struct {
		name string
		body map[string]string
	}{
		{"missing user_id", map[string]string{"resource_type": "assets", "action": "read"}},
		{"missing resource_type", map[string]string{"user_id": "test", "action": "read"}},
		{"missing action", map[string]string{"user_id": "test", "resource_type": "assets"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/rbac/check-permission", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = withOrgContext(req)

			rr := httptest.NewRecorder()

			h.CheckPermission(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, rr.Body.String(), "user_id, resource_type, and action are required")
		})
	}
}

func TestRBACHandler_ListTeams_NoOrg(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rbac/teams", nil)
	// No org context

	rr := httptest.NewRecorder()

	h.ListTeams(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "organization not found")
}

func TestRBACHandler_ListTeams_Success(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rbac/teams", nil)
	req = withOrgContext(req)

	rr := httptest.NewRecorder()

	h.ListTeams(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp, "teams")
}

func TestRBACHandler_CreateTeam_NoOrg(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	body := map[string]string{"name": "Test Team"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rbac/teams", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	// No org context

	rr := httptest.NewRecorder()

	h.CreateTeam(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "organization not found")
}

func TestRBACHandler_CreateTeam_NoUser(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	body := map[string]string{"name": "Test Team"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rbac/teams", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Add org but no user
	ctx := context.WithValue(req.Context(), middleware.OrgContextKey, testOrg())
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	h.CreateTeam(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "user not found")
}

func TestRBACHandler_CreateTeam_InvalidBody(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rbac/teams", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req = withOrgContext(req)

	rr := httptest.NewRecorder()

	h.CreateTeam(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid request body")
}

func TestRBACHandler_CreateTeam_MissingName(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	body := map[string]string{"description": "A team without name"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rbac/teams", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withOrgContext(req)

	rr := httptest.NewRecorder()

	h.CreateTeam(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "name is required")
}

func TestRBACHandler_GetTeamMembers_NoOrg(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rbac/teams/"+uuid.New().String()+"/members", nil)
	// No org context

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("teamId", uuid.New().String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	h.GetTeamMembers(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "organization not found")
}

func TestRBACHandler_GetTeamMembers_InvalidTeamID(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rbac/teams/invalid-uuid/members", nil)
	req = withOrgContext(req)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("teamId", "invalid-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	h.GetTeamMembers(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid team ID")
}

func TestRBACHandler_AddTeamMember_NoOrg(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	body := map[string]string{"user_id": "new-member"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rbac/teams/"+uuid.New().String()+"/members", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	// No org context

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("teamId", uuid.New().String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	h.AddTeamMember(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "organization not found")
}

func TestRBACHandler_AddTeamMember_NoUser(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	body := map[string]string{"user_id": "new-member"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rbac/teams/"+uuid.New().String()+"/members", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Add org but no user
	ctx := context.WithValue(req.Context(), middleware.OrgContextKey, testOrg())

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("teamId", uuid.New().String())
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	h.AddTeamMember(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "user not found")
}

func TestRBACHandler_AddTeamMember_InvalidTeamID(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	body := map[string]string{"user_id": "new-member"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rbac/teams/invalid-uuid/members", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withOrgContext(req)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("teamId", "invalid-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	h.AddTeamMember(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid team ID")
}

func TestRBACHandler_AddTeamMember_InvalidBody(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	teamID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rbac/teams/"+teamID.String()+"/members", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	req = withOrgContext(req)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("teamId", teamID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	h.AddTeamMember(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid request body")
}

func TestRBACHandler_AddTeamMember_MissingUserID(t *testing.T) {
	log := logger.New("error", "text")
	h := handlers.NewRBACHandler(nil, log)

	teamID := uuid.New()
	body := map[string]string{"role": "member"} // No user_id
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rbac/teams/"+teamID.String()+"/members", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withOrgContext(req)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("teamId", teamID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	h.AddTeamMember(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "user_id is required")
}

// =============================================================================
// RBAC Types Test
// =============================================================================

func TestRBACTypes(t *testing.T) {
	t.Run("Action constants", func(t *testing.T) {
		assert.Equal(t, rbac.Action("read"), rbac.ActionRead)
		assert.Equal(t, rbac.Action("write"), rbac.ActionWrite)
		assert.Equal(t, rbac.Action("delete"), rbac.ActionDelete)
		assert.Equal(t, rbac.Action("execute"), rbac.ActionExecute)
		assert.Equal(t, rbac.Action("approve"), rbac.ActionApprove)
		assert.Equal(t, rbac.Action("admin"), rbac.ActionAdmin)
	})

	t.Run("ResourceType constants", func(t *testing.T) {
		assert.Equal(t, rbac.ResourceType("assets"), rbac.ResourceAssets)
		assert.Equal(t, rbac.ResourceType("images"), rbac.ResourceImages)
		assert.Equal(t, rbac.ResourceType("sites"), rbac.ResourceSites)
		assert.Equal(t, rbac.ResourceType("drift"), rbac.ResourceDrift)
		assert.Equal(t, rbac.ResourceType("compliance"), rbac.ResourceCompliance)
		assert.Equal(t, rbac.ResourceType("dr"), rbac.ResourceDR)
		assert.Equal(t, rbac.ResourceType("tasks"), rbac.ResourceTasks)
		assert.Equal(t, rbac.ResourceType("organization"), rbac.ResourceOrganization)
		assert.Equal(t, rbac.ResourceType("audit"), rbac.ResourceAudit)
	})

	t.Run("SystemRole constants", func(t *testing.T) {
		assert.Equal(t, rbac.SystemRole("org_owner"), rbac.RoleOrgOwner)
		assert.Equal(t, rbac.SystemRole("org_admin"), rbac.RoleOrgAdmin)
		assert.Equal(t, rbac.SystemRole("infra_admin"), rbac.RoleInfraAdmin)
		assert.Equal(t, rbac.SystemRole("security_admin"), rbac.RoleSecurityAdmin)
		assert.Equal(t, rbac.SystemRole("dr_admin"), rbac.RoleDRAdmin)
		assert.Equal(t, rbac.SystemRole("operator"), rbac.RoleOperator)
		assert.Equal(t, rbac.SystemRole("analyst"), rbac.RoleAnalyst)
		assert.Equal(t, rbac.SystemRole("viewer"), rbac.RoleViewer)
	})
}

func TestPermissionDeniedError(t *testing.T) {
	err := &rbac.PermissionDeniedError{
		UserID:       "test-user",
		ResourceType: rbac.ResourceAssets,
		Action:       rbac.ActionWrite,
		Reason:       "no matching permission found",
	}

	assert.Contains(t, err.Error(), "permission denied")
	assert.Contains(t, err.Error(), "test-user")
	assert.Contains(t, err.Error(), "write")
	assert.Contains(t, err.Error(), "assets")
	assert.Contains(t, err.Error(), "no matching permission found")

	assert.True(t, rbac.IsPermissionDenied(err))
	assert.False(t, rbac.IsPermissionDenied(nil))
	assert.False(t, rbac.IsPermissionDenied(context.Canceled))
}

func TestRole(t *testing.T) {
	role := rbac.Role{
		ID:           uuid.New(),
		Name:         "test_role",
		DisplayName:  "Test Role",
		Description:  "A test role",
		OrgID:        nil, // System role
		IsSystemRole: true,
		ParentRoleID: nil,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	assert.Equal(t, "test_role", role.Name)
	assert.True(t, role.IsSystemRole)
	assert.Nil(t, role.OrgID)
}

func TestTeam(t *testing.T) {
	orgID := uuid.New()
	team := rbac.Team{
		ID:          uuid.New(),
		OrgID:       orgID,
		Name:        "Engineering",
		Description: "Engineering team",
		CreatedBy:   "admin-user",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	assert.Equal(t, "Engineering", team.Name)
	assert.Equal(t, orgID, team.OrgID)
}

func TestPermissionCheck(t *testing.T) {
	t.Run("allowed permission", func(t *testing.T) {
		check := &rbac.PermissionCheck{
			Allowed: true,
			Source:  "role",
			Reason:  "",
		}
		assert.True(t, check.Allowed)
		assert.Equal(t, "role", check.Source)
	})

	t.Run("denied permission", func(t *testing.T) {
		check := &rbac.PermissionCheck{
			Allowed: false,
			Source:  "",
			Reason:  "no matching permission found",
		}
		assert.False(t, check.Allowed)
		assert.Equal(t, "no matching permission found", check.Reason)
	})
}
