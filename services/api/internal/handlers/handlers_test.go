package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
)

// testOrg returns a test organization for use in tests.
func testOrg() *models.Organization {
	return &models.Organization{
		ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Name: "Test Org",
		Slug: "test-org",
	}
}

// testUser returns a test user for use in tests.
func testUser() *models.User {
	return &models.User{
		ExternalID: "test-user",
		Email:      "test@example.com",
		Name:       "Test User",
		Role:       models.RoleAdmin,
	}
}

// withOrgContext adds organization and user to the request context.
func withOrgContext(r *http.Request) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, middleware.UserContextKey, testUser())
	ctx = context.WithValue(ctx, middleware.OrgContextKey, testOrg())
	return r.WithContext(ctx)
}

// executeRequest executes an HTTP request and returns the response recorder.
func executeRequest(handler http.HandlerFunc, r *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, r)
	return rr
}

// decodeJSON decodes JSON response body into the target struct.
func decodeJSON(rr *httptest.ResponseRecorder, target any) error {
	return json.NewDecoder(rr.Body).Decode(target)
}
