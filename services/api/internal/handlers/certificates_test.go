package handlers_test

import (
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
	"github.com/quantumlayerhq/ql-rf/services/api/internal/handlers"
)

// MockCertificateDB implements a mock database for certificate testing.
type MockCertificateDB struct {
	certificates []handlers.Certificate
	usages       map[uuid.UUID][]handlers.CertificateUsage
	rotations    []handlers.CertificateRotation
	alerts       []handlers.CertificateAlert
}

// NewMockCertificateDB creates a new mock certificate database.
func NewMockCertificateDB() *MockCertificateDB {
	return &MockCertificateDB{
		certificates: []handlers.Certificate{},
		usages:       make(map[uuid.UUID][]handlers.CertificateUsage),
		rotations:    []handlers.CertificateRotation{},
		alerts:       []handlers.CertificateAlert{},
	}
}

// AddCertificate adds a certificate to the mock database.
func (m *MockCertificateDB) AddCertificate(cert handlers.Certificate) {
	m.certificates = append(m.certificates, cert)
}

// AddUsage adds usage for a certificate.
func (m *MockCertificateDB) AddUsage(certID uuid.UUID, usage handlers.CertificateUsage) {
	m.usages[certID] = append(m.usages[certID], usage)
}

// AddRotation adds a rotation record.
func (m *MockCertificateDB) AddRotation(rotation handlers.CertificateRotation) {
	m.rotations = append(m.rotations, rotation)
}

// AddAlert adds an alert.
func (m *MockCertificateDB) AddAlert(alert handlers.CertificateAlert) {
	m.alerts = append(m.alerts, alert)
}

// setupCertificateTestHandler creates a test handler with mock data.
func setupCertificateTestHandler(t *testing.T) (*handlers.CertificateHandler, *MockCertificateDB) {
	log := logger.New("error", "text")

	// Note: In a real test, we'd use a test database or mock pgxpool
	// For unit tests, we test the handler logic with mocked context
	handler := handlers.NewCertificateHandler(nil, log)
	mockDB := NewMockCertificateDB()

	return handler, mockDB
}

func TestCertificateHandler_ListCertificates(t *testing.T) {
	t.Run("unauthorized without org context", func(t *testing.T) {
		log := logger.New("error", "text")
		handler := handlers.NewCertificateHandler(nil, log)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/certificates", nil)
		w := httptest.NewRecorder()

		handler.ListCertificates(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "organization not found")
	})
}

func TestCertificateHandler_GetCertificate(t *testing.T) {
	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		log := logger.New("error", "text")
		handler := handlers.NewCertificateHandler(nil, log)

		r := chi.NewRouter()
		r.Get("/certificates/{id}", handler.GetCertificate)

		req := httptest.NewRequest(http.MethodGet, "/certificates/invalid-uuid", nil)
		req = withOrgContext(req)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid certificate ID")
	})

	t.Run("unauthorized without org context", func(t *testing.T) {
		log := logger.New("error", "text")
		handler := handlers.NewCertificateHandler(nil, log)

		r := chi.NewRouter()
		r.Get("/certificates/{id}", handler.GetCertificate)

		certID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/certificates/"+certID.String(), nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestCertificateHandler_GetCertificateSummary(t *testing.T) {
	t.Run("unauthorized without org context", func(t *testing.T) {
		log := logger.New("error", "text")
		handler := handlers.NewCertificateHandler(nil, log)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/certificates/summary", nil)
		w := httptest.NewRecorder()

		handler.GetCertificateSummary(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestCertificateHandler_GetCertificateUsage(t *testing.T) {
	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		log := logger.New("error", "text")
		handler := handlers.NewCertificateHandler(nil, log)

		r := chi.NewRouter()
		r.Get("/certificates/{id}/usage", handler.GetCertificateUsage)

		req := httptest.NewRequest(http.MethodGet, "/certificates/invalid-uuid/usage", nil)
		req = withOrgContext(req)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unauthorized without org context", func(t *testing.T) {
		log := logger.New("error", "text")
		handler := handlers.NewCertificateHandler(nil, log)

		r := chi.NewRouter()
		r.Get("/certificates/{id}/usage", handler.GetCertificateUsage)

		certID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/certificates/"+certID.String()+"/usage", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestCertificateHandler_ListRotations(t *testing.T) {
	t.Run("unauthorized without org context", func(t *testing.T) {
		log := logger.New("error", "text")
		handler := handlers.NewCertificateHandler(nil, log)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/certificates/rotations", nil)
		w := httptest.NewRecorder()

		handler.ListRotations(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestCertificateHandler_GetRotation(t *testing.T) {
	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		log := logger.New("error", "text")
		handler := handlers.NewCertificateHandler(nil, log)

		r := chi.NewRouter()
		r.Get("/rotations/{id}", handler.GetRotation)

		req := httptest.NewRequest(http.MethodGet, "/rotations/invalid-uuid", nil)
		req = withOrgContext(req)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCertificateHandler_ListAlerts(t *testing.T) {
	t.Run("unauthorized without org context", func(t *testing.T) {
		log := logger.New("error", "text")
		handler := handlers.NewCertificateHandler(nil, log)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/certificates/alerts", nil)
		w := httptest.NewRecorder()

		handler.ListAlerts(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestCertificateHandler_AcknowledgeAlert(t *testing.T) {
	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		log := logger.New("error", "text")
		handler := handlers.NewCertificateHandler(nil, log)

		r := chi.NewRouter()
		r.Post("/alerts/{id}/acknowledge", handler.AcknowledgeAlert)

		req := httptest.NewRequest(http.MethodPost, "/alerts/invalid-uuid/acknowledge", nil)
		req = withOrgContext(req)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unauthorized without org context", func(t *testing.T) {
		log := logger.New("error", "text")
		handler := handlers.NewCertificateHandler(nil, log)

		r := chi.NewRouter()
		r.Post("/alerts/{id}/acknowledge", handler.AcknowledgeAlert)

		alertID := uuid.New()
		req := httptest.NewRequest(http.MethodPost, "/alerts/"+alertID.String()+"/acknowledge", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// Test certificate type structures
func TestCertificateTypes(t *testing.T) {
	t.Run("Certificate JSON serialization", func(t *testing.T) {
		cert := handlers.Certificate{
			ID:                   uuid.New(),
			OrgID:                uuid.New(),
			Fingerprint:          "abc123",
			CommonName:           "example.com",
			SubjectAltNames:      []string{"www.example.com", "api.example.com"},
			IssuerCommonName:     "Let's Encrypt",
			IssuerOrganization:   "Let's Encrypt",
			IsSelfSigned:         false,
			IsCA:                 false,
			NotBefore:            time.Now().Add(-365 * 24 * time.Hour),
			NotAfter:             time.Now().Add(30 * 24 * time.Hour),
			DaysUntilExpiry:      30,
			KeyAlgorithm:         "RSA",
			KeySize:              2048,
			SignatureAlgorithm:   "SHA256-RSA",
			Source:               "acm",
			SourceRef:            "arn:aws:acm:us-east-1:123456789:certificate/abc",
			Platform:             "aws",
			Status:               "active",
			AutoRenew:            true,
			RenewalThresholdDays: 30,
			RotationCount:        0,
			Tags:                 map[string]string{"env": "production"},
			Metadata:             map[string]any{"imported": true},
			DiscoveredAt:         time.Now(),
			LastScannedAt:        time.Now(),
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		}

		data, err := json.Marshal(cert)
		require.NoError(t, err)
		assert.Contains(t, string(data), "example.com")
		assert.Contains(t, string(data), "www.example.com")

		var decoded handlers.Certificate
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, cert.CommonName, decoded.CommonName)
		assert.Equal(t, cert.SubjectAltNames, decoded.SubjectAltNames)
	})

	t.Run("CertificateSummary JSON serialization", func(t *testing.T) {
		summary := handlers.CertificateSummary{
			TotalCertificates:  100,
			ActiveCertificates: 90,
			ExpiringSoon:       5,
			Expired:            5,
			Expiring7Days:      2,
			Expiring30Days:     5,
			Expiring90Days:     10,
			AutoRenewEnabled:   70,
			SelfSigned:         10,
			PlatformsCount:     3,
		}

		data, err := json.Marshal(summary)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"total_certificates":100`)
		assert.Contains(t, string(data), `"expiring_soon":5`)
	})

	t.Run("CertificateUsage JSON serialization", func(t *testing.T) {
		port := 443
		usage := handlers.CertificateUsage{
			ID:           uuid.New(),
			CertID:       uuid.New(),
			UsageType:    "load_balancer",
			UsageRef:     "arn:aws:elasticloadbalancing:us-east-1:123456789:loadbalancer/app/my-lb/abc123",
			UsagePort:    &port,
			Platform:     "aws",
			ServiceName:  strPtr("my-app"),
			Endpoint:     strPtr("my-lb.us-east-1.elb.amazonaws.com"),
			Status:       "active",
			TLSVersion:   strPtr("TLSv1.3"),
			Metadata:     map[string]any{"listeners": 2},
			DiscoveredAt: time.Now(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		data, err := json.Marshal(usage)
		require.NoError(t, err)
		assert.Contains(t, string(data), "load_balancer")
		assert.Contains(t, string(data), "TLSv1.3")
	})

	t.Run("CertificateRotation JSON serialization", func(t *testing.T) {
		rotation := handlers.CertificateRotation{
			ID:                uuid.New(),
			OrgID:             uuid.New(),
			RotationType:      "renewal",
			InitiatedBy:       "ai_agent",
			Status:            "completed",
			AffectedUsages:    5,
			SuccessfulUpdates: 5,
			FailedUpdates:     0,
			RollbackAvailable: true,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		data, err := json.Marshal(rotation)
		require.NoError(t, err)
		assert.Contains(t, string(data), "renewal")
		assert.Contains(t, string(data), "ai_agent")
	})

	t.Run("CertificateAlert JSON serialization", func(t *testing.T) {
		daysUntilExpiry := 7
		alert := handlers.CertificateAlert{
			ID:                    uuid.New(),
			OrgID:                 uuid.New(),
			CertID:                uuid.New(),
			AlertType:             "expiry_warning",
			Severity:              "high",
			Title:                 "Certificate expiring soon",
			Message:               "Certificate example.com will expire in 7 days",
			DaysUntilExpiry:       &daysUntilExpiry,
			Status:                "open",
			AutoRotationTriggered: false,
			NotificationsSent:     []any{"email", "slack"},
			CreatedAt:             time.Now(),
			UpdatedAt:             time.Now(),
		}

		data, err := json.Marshal(alert)
		require.NoError(t, err)
		assert.Contains(t, string(data), "expiry_warning")
		assert.Contains(t, string(data), "Certificate expiring soon")
	})
}

func strPtr(s string) *string {
	return &s
}

// Ensure types are exported and usable
var _ *handlers.CertificateHandler
var _ handlers.Certificate
var _ handlers.CertificateSummary
var _ handlers.CertificateUsage
var _ handlers.CertificateRotation
var _ handlers.CertificateAlert
var _ handlers.CertificateListResponse
