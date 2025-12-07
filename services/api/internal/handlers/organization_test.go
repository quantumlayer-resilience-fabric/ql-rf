package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/pkg/multitenancy"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/handlers"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
)

// MockMultitenancyService implements handlers.MultitenancyServiceInterface for testing.
type MockMultitenancyService struct {
	GetQuotaFn               func(ctx context.Context, orgID uuid.UUID) (*multitenancy.OrganizationQuota, error)
	GetUsageFn               func(ctx context.Context, orgID uuid.UUID) (*multitenancy.OrganizationUsage, error)
	GetQuotaStatusFn         func(ctx context.Context, orgID uuid.UUID) ([]multitenancy.QuotaStatus, error)
	GetSubscriptionFn        func(ctx context.Context, orgID uuid.UUID) (*multitenancy.Subscription, error)
	GetPlanFn                func(ctx context.Context, name string) (*multitenancy.SubscriptionPlan, error)
	ListPlansFn              func(ctx context.Context) ([]multitenancy.SubscriptionPlan, error)
	CreateOrganizationFn     func(ctx context.Context, params multitenancy.CreateOrganizationParams) (*multitenancy.CreateOrganizationResult, error)
	GetOrganizationFn        func(ctx context.Context, orgID uuid.UUID) (*multitenancy.Organization, error)
	GetUserOrganizationFn    func(ctx context.Context, userID string) (*multitenancy.Organization, error)
	LinkUserToOrganizationFn func(ctx context.Context, userID string, orgID uuid.UUID, role string) error
	SeedDemoDataFn           func(ctx context.Context, orgID uuid.UUID, params multitenancy.SeedDemoDataParams) (*multitenancy.SeedDemoDataResult, error)
}

// Compile-time check that MockMultitenancyService implements MultitenancyServiceInterface.
var _ handlers.MultitenancyServiceInterface = (*MockMultitenancyService)(nil)

func (m *MockMultitenancyService) GetQuota(ctx context.Context, orgID uuid.UUID) (*multitenancy.OrganizationQuota, error) {
	if m.GetQuotaFn != nil {
		return m.GetQuotaFn(ctx, orgID)
	}
	return nil, nil
}

func (m *MockMultitenancyService) GetUsage(ctx context.Context, orgID uuid.UUID) (*multitenancy.OrganizationUsage, error) {
	if m.GetUsageFn != nil {
		return m.GetUsageFn(ctx, orgID)
	}
	return nil, nil
}

func (m *MockMultitenancyService) GetQuotaStatus(ctx context.Context, orgID uuid.UUID) ([]multitenancy.QuotaStatus, error) {
	if m.GetQuotaStatusFn != nil {
		return m.GetQuotaStatusFn(ctx, orgID)
	}
	return nil, nil
}

func (m *MockMultitenancyService) GetSubscription(ctx context.Context, orgID uuid.UUID) (*multitenancy.Subscription, error) {
	if m.GetSubscriptionFn != nil {
		return m.GetSubscriptionFn(ctx, orgID)
	}
	return nil, nil
}

func (m *MockMultitenancyService) GetPlan(ctx context.Context, name string) (*multitenancy.SubscriptionPlan, error) {
	if m.GetPlanFn != nil {
		return m.GetPlanFn(ctx, name)
	}
	return nil, nil
}

func (m *MockMultitenancyService) ListPlans(ctx context.Context) ([]multitenancy.SubscriptionPlan, error) {
	if m.ListPlansFn != nil {
		return m.ListPlansFn(ctx)
	}
	return nil, nil
}

func (m *MockMultitenancyService) CreateOrganization(ctx context.Context, params multitenancy.CreateOrganizationParams) (*multitenancy.CreateOrganizationResult, error) {
	if m.CreateOrganizationFn != nil {
		return m.CreateOrganizationFn(ctx, params)
	}
	return nil, nil
}

func (m *MockMultitenancyService) GetOrganization(ctx context.Context, orgID uuid.UUID) (*multitenancy.Organization, error) {
	if m.GetOrganizationFn != nil {
		return m.GetOrganizationFn(ctx, orgID)
	}
	return nil, nil
}

func (m *MockMultitenancyService) GetUserOrganization(ctx context.Context, userID string) (*multitenancy.Organization, error) {
	if m.GetUserOrganizationFn != nil {
		return m.GetUserOrganizationFn(ctx, userID)
	}
	return nil, nil
}

func (m *MockMultitenancyService) LinkUserToOrganization(ctx context.Context, userID string, orgID uuid.UUID, role string) error {
	if m.LinkUserToOrganizationFn != nil {
		return m.LinkUserToOrganizationFn(ctx, userID, orgID, role)
	}
	return nil
}

func (m *MockMultitenancyService) SeedDemoData(ctx context.Context, orgID uuid.UUID, params multitenancy.SeedDemoDataParams) (*multitenancy.SeedDemoDataResult, error) {
	if m.SeedDemoDataFn != nil {
		return m.SeedDemoDataFn(ctx, orgID, params)
	}
	return nil, nil
}

func testLogger() *logger.Logger {
	return logger.New("error", "json")
}

func withOrgAndUserContext(r *http.Request, org *models.Organization, user *models.User) *http.Request {
	ctx := r.Context()
	if user != nil {
		ctx = context.WithValue(ctx, middleware.UserContextKey, user)
	}
	if org != nil {
		ctx = context.WithValue(ctx, middleware.OrgContextKey, org)
	}
	return r.WithContext(ctx)
}

func TestOrganizationHandler_GetQuota(t *testing.T) {
	orgID := uuid.New()
	org := &models.Organization{ID: orgID, Name: "Test Org", Slug: "test-org"}
	user := &models.User{ExternalID: "user-1", Email: "test@example.com"}

	t.Run("returns quota successfully", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{
			GetQuotaFn: func(ctx context.Context, id uuid.UUID) (*multitenancy.OrganizationQuota, error) {
				return &multitenancy.OrganizationQuota{
					OrgID:     id,
					MaxAssets: 500,
					MaxImages: 50,
				}, nil
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		req := httptest.NewRequest(http.MethodGet, "/organization/quota", nil)
		req = withOrgAndUserContext(req, org, user)
		rr := httptest.NewRecorder()

		handler.GetQuota(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response multitenancy.OrganizationQuota
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
		assert.Equal(t, 500, response.MaxAssets)
		assert.Equal(t, 50, response.MaxImages)
	})

	t.Run("returns default quota when none configured", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{
			GetQuotaFn: func(ctx context.Context, id uuid.UUID) (*multitenancy.OrganizationQuota, error) {
				return nil, nil
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		req := httptest.NewRequest(http.MethodGet, "/organization/quota", nil)
		req = withOrgAndUserContext(req, org, user)
		rr := httptest.NewRecorder()

		handler.GetQuota(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response multitenancy.OrganizationQuota
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
		assert.Equal(t, 1000, response.MaxAssets)
		assert.Equal(t, 100, response.MaxImages)
	})

	t.Run("returns unauthorized when no org context", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{}
		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		req := httptest.NewRequest(http.MethodGet, "/organization/quota", nil)
		rr := httptest.NewRecorder()

		handler.GetQuota(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("returns error on service failure", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{
			GetQuotaFn: func(ctx context.Context, id uuid.UUID) (*multitenancy.OrganizationQuota, error) {
				return nil, errors.New("database error")
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		req := httptest.NewRequest(http.MethodGet, "/organization/quota", nil)
		req = withOrgAndUserContext(req, org, user)
		rr := httptest.NewRecorder()

		handler.GetQuota(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestOrganizationHandler_GetUsage(t *testing.T) {
	orgID := uuid.New()
	org := &models.Organization{ID: orgID, Name: "Test Org", Slug: "test-org"}
	user := &models.User{ExternalID: "user-1", Email: "test@example.com"}

	t.Run("returns usage successfully", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{
			GetUsageFn: func(ctx context.Context, id uuid.UUID) (*multitenancy.OrganizationUsage, error) {
				return &multitenancy.OrganizationUsage{
					OrgID:      id,
					AssetCount: 100,
					ImageCount: 25,
				}, nil
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		req := httptest.NewRequest(http.MethodGet, "/organization/usage", nil)
		req = withOrgAndUserContext(req, org, user)
		rr := httptest.NewRecorder()

		handler.GetUsage(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response multitenancy.OrganizationUsage
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
		assert.Equal(t, 100, response.AssetCount)
		assert.Equal(t, 25, response.ImageCount)
	})

	t.Run("returns unauthorized when no org context", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{}
		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		req := httptest.NewRequest(http.MethodGet, "/organization/usage", nil)
		rr := httptest.NewRecorder()

		handler.GetUsage(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestOrganizationHandler_ListPlans(t *testing.T) {
	t.Run("returns plans from service", func(t *testing.T) {
		plans := []multitenancy.SubscriptionPlan{
			{ID: uuid.New(), Name: "free", DisplayName: "Free"},
			{ID: uuid.New(), Name: "starter", DisplayName: "Starter"},
		}

		mockSvc := &MockMultitenancyService{
			ListPlansFn: func(ctx context.Context) ([]multitenancy.SubscriptionPlan, error) {
				return plans, nil
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		req := httptest.NewRequest(http.MethodGet, "/organization/plans", nil)
		rr := httptest.NewRecorder()

		handler.ListPlans(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response struct {
			Plans []multitenancy.SubscriptionPlan `json:"plans"`
		}
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
		assert.Len(t, response.Plans, 2)
		assert.Equal(t, "free", response.Plans[0].Name)
	})

	t.Run("returns default plans when none in database", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{
			ListPlansFn: func(ctx context.Context) ([]multitenancy.SubscriptionPlan, error) {
				return []multitenancy.SubscriptionPlan{}, nil
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		req := httptest.NewRequest(http.MethodGet, "/organization/plans", nil)
		rr := httptest.NewRecorder()

		handler.ListPlans(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response struct {
			Plans []multitenancy.SubscriptionPlan `json:"plans"`
		}
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
		assert.Len(t, response.Plans, 4) // free, starter, professional, enterprise
	})

	t.Run("returns error on service failure", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{
			ListPlansFn: func(ctx context.Context) ([]multitenancy.SubscriptionPlan, error) {
				return nil, errors.New("database error")
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		req := httptest.NewRequest(http.MethodGet, "/organization/plans", nil)
		rr := httptest.NewRecorder()

		handler.ListPlans(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestOrganizationHandler_CreateOrganization(t *testing.T) {
	user := &models.User{ExternalID: "user-1", Email: "test@example.com"}

	t.Run("creates organization successfully", func(t *testing.T) {
		orgID := uuid.New()
		mockSvc := &MockMultitenancyService{
			CreateOrganizationFn: func(ctx context.Context, params multitenancy.CreateOrganizationParams) (*multitenancy.CreateOrganizationResult, error) {
				return &multitenancy.CreateOrganizationResult{
					Organization: &multitenancy.Organization{
						ID:   orgID,
						Name: params.Name,
						Slug: "test-org",
					},
					Quota: &multitenancy.OrganizationQuota{
						OrgID:     orgID,
						MaxAssets: 100,
					},
					Subscription: &multitenancy.Subscription{
						ID:     uuid.New(),
						OrgID:  orgID,
						Status: "active",
					},
				}, nil
			},
			LinkUserToOrganizationFn: func(ctx context.Context, userID string, orgID uuid.UUID, role string) error {
				return nil
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		body := bytes.NewBufferString(`{"name": "Test Organization"}`)
		req := httptest.NewRequest(http.MethodPost, "/organizations", body)
		req.Header.Set("Content-Type", "application/json")
		req = withOrgAndUserContext(req, nil, user)
		rr := httptest.NewRecorder()

		handler.CreateOrganization(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var response multitenancy.CreateOrganizationResult
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
		assert.Equal(t, "Test Organization", response.Organization.Name)
		assert.NotNil(t, response.Quota)
		assert.NotNil(t, response.Subscription)
	})

	t.Run("returns bad request for missing name", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{}
		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		body := bytes.NewBufferString(`{}`)
		req := httptest.NewRequest(http.MethodPost, "/organizations", body)
		req.Header.Set("Content-Type", "application/json")
		req = withOrgAndUserContext(req, nil, user)
		rr := httptest.NewRecorder()

		handler.CreateOrganization(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns bad request for invalid JSON", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{}
		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		body := bytes.NewBufferString(`{invalid}`)
		req := httptest.NewRequest(http.MethodPost, "/organizations", body)
		req.Header.Set("Content-Type", "application/json")
		req = withOrgAndUserContext(req, nil, user)
		rr := httptest.NewRecorder()

		handler.CreateOrganization(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns error on service failure", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{
			CreateOrganizationFn: func(ctx context.Context, params multitenancy.CreateOrganizationParams) (*multitenancy.CreateOrganizationResult, error) {
				return nil, errors.New("database error")
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		body := bytes.NewBufferString(`{"name": "Test Organization"}`)
		req := httptest.NewRequest(http.MethodPost, "/organizations", body)
		req.Header.Set("Content-Type", "application/json")
		req = withOrgAndUserContext(req, nil, user)
		rr := httptest.NewRecorder()

		handler.CreateOrganization(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestOrganizationHandler_GetCurrentOrganization(t *testing.T) {
	orgID := uuid.New()
	org := &models.Organization{ID: orgID, Name: "Test Org", Slug: "test-org"}
	user := &models.User{ExternalID: "user-1", Email: "test@example.com"}

	t.Run("returns current organization", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{
			GetOrganizationFn: func(ctx context.Context, id uuid.UUID) (*multitenancy.Organization, error) {
				return &multitenancy.Organization{
					ID:   id,
					Name: "Full Org Details",
					Slug: "full-org",
				}, nil
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		req := httptest.NewRequest(http.MethodGet, "/organization", nil)
		req = withOrgAndUserContext(req, org, user)
		rr := httptest.NewRecorder()

		handler.GetCurrentOrganization(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response multitenancy.Organization
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
		assert.Equal(t, "Full Org Details", response.Name)
	})

	t.Run("returns unauthorized when no org context", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{}
		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		req := httptest.NewRequest(http.MethodGet, "/organization", nil)
		rr := httptest.NewRecorder()

		handler.GetCurrentOrganization(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestOrganizationHandler_CheckUserOrganization(t *testing.T) {
	user := &models.User{ExternalID: "user-1", Email: "test@example.com"}

	t.Run("returns true when user has organization", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{
			GetUserOrganizationFn: func(ctx context.Context, userID string) (*multitenancy.Organization, error) {
				return &multitenancy.Organization{
					ID:   uuid.New(),
					Name: "User Org",
					Slug: "user-org",
				}, nil
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		req := httptest.NewRequest(http.MethodGet, "/organization/check", nil)
		req = withOrgAndUserContext(req, nil, user)
		rr := httptest.NewRecorder()

		handler.CheckUserOrganization(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response struct {
			HasOrganization bool                     `json:"has_organization"`
			Organization    *multitenancy.Organization `json:"organization"`
		}
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
		assert.True(t, response.HasOrganization)
		assert.NotNil(t, response.Organization)
	})

	t.Run("returns false when user has no organization", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{
			GetUserOrganizationFn: func(ctx context.Context, userID string) (*multitenancy.Organization, error) {
				return nil, nil
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		req := httptest.NewRequest(http.MethodGet, "/organization/check", nil)
		req = withOrgAndUserContext(req, nil, user)
		rr := httptest.NewRecorder()

		handler.CheckUserOrganization(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response struct {
			HasOrganization bool                     `json:"has_organization"`
			Organization    *multitenancy.Organization `json:"organization"`
		}
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
		assert.False(t, response.HasOrganization)
		assert.Nil(t, response.Organization)
	})

	t.Run("returns unauthorized when no user context", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{}
		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		req := httptest.NewRequest(http.MethodGet, "/organization/check", nil)
		rr := httptest.NewRecorder()

		handler.CheckUserOrganization(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestOrganizationHandler_SeedDemoData(t *testing.T) {
	orgID := uuid.New()
	org := &models.Organization{ID: orgID, Name: "Test Org", Slug: "test-org"}
	user := &models.User{ExternalID: "user-1", Email: "test@example.com"}

	t.Run("seeds demo data for AWS", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{
			SeedDemoDataFn: func(ctx context.Context, id uuid.UUID, params multitenancy.SeedDemoDataParams) (*multitenancy.SeedDemoDataResult, error) {
				assert.Equal(t, "aws", params.Platform)
				return &multitenancy.SeedDemoDataResult{
					SitesCreated:  3,
					AssetsCreated: 15,
					ImagesCreated: 5,
				}, nil
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		body := bytes.NewBufferString(`{"platform": "aws"}`)
		req := httptest.NewRequest(http.MethodPost, "/organization/seed-demo", body)
		req.Header.Set("Content-Type", "application/json")
		req = withOrgAndUserContext(req, org, user)
		rr := httptest.NewRecorder()

		handler.SeedDemoData(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response multitenancy.SeedDemoDataResult
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
		assert.Equal(t, 3, response.SitesCreated)
		assert.Equal(t, 15, response.AssetsCreated)
		assert.Equal(t, 5, response.ImagesCreated)
	})

	t.Run("defaults to AWS when no platform specified", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{
			SeedDemoDataFn: func(ctx context.Context, id uuid.UUID, params multitenancy.SeedDemoDataParams) (*multitenancy.SeedDemoDataResult, error) {
				// Empty body results in decode error which defaults to aws
				return &multitenancy.SeedDemoDataResult{
					SitesCreated:  3,
					AssetsCreated: 15,
					ImagesCreated: 5,
				}, nil
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		// Empty body (nil) triggers the decode error path which defaults to aws
		req := httptest.NewRequest(http.MethodPost, "/organization/seed-demo", nil)
		req.Header.Set("Content-Type", "application/json")
		req = withOrgAndUserContext(req, org, user)
		rr := httptest.NewRecorder()

		handler.SeedDemoData(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("seeds demo data for Azure", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{
			SeedDemoDataFn: func(ctx context.Context, id uuid.UUID, params multitenancy.SeedDemoDataParams) (*multitenancy.SeedDemoDataResult, error) {
				assert.Equal(t, "azure", params.Platform)
				return &multitenancy.SeedDemoDataResult{
					SitesCreated:  2,
					AssetsCreated: 10,
					ImagesCreated: 4,
				}, nil
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		body := bytes.NewBufferString(`{"platform": "azure"}`)
		req := httptest.NewRequest(http.MethodPost, "/organization/seed-demo", body)
		req.Header.Set("Content-Type", "application/json")
		req = withOrgAndUserContext(req, org, user)
		rr := httptest.NewRecorder()

		handler.SeedDemoData(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("returns unauthorized when no org context", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{}
		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		req := httptest.NewRequest(http.MethodPost, "/organization/seed-demo", nil)
		rr := httptest.NewRecorder()

		handler.SeedDemoData(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("returns error on service failure", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{
			SeedDemoDataFn: func(ctx context.Context, id uuid.UUID, params multitenancy.SeedDemoDataParams) (*multitenancy.SeedDemoDataResult, error) {
				return nil, errors.New("database error")
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		body := bytes.NewBufferString(`{"platform": "aws"}`)
		req := httptest.NewRequest(http.MethodPost, "/organization/seed-demo", body)
		req.Header.Set("Content-Type", "application/json")
		req = withOrgAndUserContext(req, org, user)
		rr := httptest.NewRecorder()

		handler.SeedDemoData(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestOrganizationHandler_GetQuotaStatus(t *testing.T) {
	orgID := uuid.New()
	org := &models.Organization{ID: orgID, Name: "Test Org", Slug: "test-org"}
	user := &models.User{ExternalID: "user-1", Email: "test@example.com"}

	t.Run("returns quota status successfully", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{
			GetQuotaStatusFn: func(ctx context.Context, id uuid.UUID) ([]multitenancy.QuotaStatus, error) {
				return []multitenancy.QuotaStatus{
					{
						ResourceType:   multitenancy.QuotaAssets,
						Limit:          100,
						Used:           50,
						Remaining:      50,
						PercentageUsed: 50.0,
						IsExceeded:     false,
					},
				}, nil
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		req := httptest.NewRequest(http.MethodGet, "/organization/quota-status", nil)
		req = withOrgAndUserContext(req, org, user)
		rr := httptest.NewRecorder()

		handler.GetQuotaStatus(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response struct {
			Statuses []multitenancy.QuotaStatus `json:"statuses"`
		}
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
		assert.Len(t, response.Statuses, 1)
		assert.Equal(t, multitenancy.QuotaAssets, response.Statuses[0].ResourceType)
	})
}

func TestOrganizationHandler_GetSubscription(t *testing.T) {
	orgID := uuid.New()
	org := &models.Organization{ID: orgID, Name: "Test Org", Slug: "test-org"}
	user := &models.User{ExternalID: "user-1", Email: "test@example.com"}

	t.Run("returns subscription successfully", func(t *testing.T) {
		mockSvc := &MockMultitenancyService{
			GetSubscriptionFn: func(ctx context.Context, id uuid.UUID) (*multitenancy.Subscription, error) {
				return &multitenancy.Subscription{
					ID:     uuid.New(),
					OrgID:  id,
					Status: "active",
				}, nil
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		req := httptest.NewRequest(http.MethodGet, "/organization/subscription", nil)
		req = withOrgAndUserContext(req, org, user)
		rr := httptest.NewRecorder()

		handler.GetSubscription(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response multitenancy.Subscription
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
		assert.Equal(t, "active", response.Status)
	})

	t.Run("returns default trial subscription when none exists", func(t *testing.T) {
		planID := uuid.New()
		mockSvc := &MockMultitenancyService{
			GetSubscriptionFn: func(ctx context.Context, id uuid.UUID) (*multitenancy.Subscription, error) {
				return nil, nil
			},
			GetPlanFn: func(ctx context.Context, name string) (*multitenancy.SubscriptionPlan, error) {
				return &multitenancy.SubscriptionPlan{
					ID:   planID,
					Name: "free",
				}, nil
			},
		}

		handler := handlers.NewOrganizationHandler(mockSvc, testLogger())

		req := httptest.NewRequest(http.MethodGet, "/organization/subscription", nil)
		req = withOrgAndUserContext(req, org, user)
		rr := httptest.NewRecorder()

		handler.GetSubscription(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response multitenancy.Subscription
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
		assert.Equal(t, "trial", response.Status)
		assert.Equal(t, planID, response.PlanID)
	})
}

func TestNewOrganizationHandler(t *testing.T) {
	mockSvc := &MockMultitenancyService{}
	handler := handlers.NewOrganizationHandler(mockSvc, testLogger())
	assert.NotNil(t, handler)
}
