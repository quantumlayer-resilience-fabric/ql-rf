package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/sbom"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/handlers"
)

// MockSBOMService implements the handlers.SBOMServiceInterface for testing.
type MockSBOMService struct {
	sboms           map[uuid.UUID]*sbom.SBOM
	sbomsByImage    map[uuid.UUID]*sbom.SBOM
	packages        map[uuid.UUID][]sbom.Package
	vulnerabilities map[uuid.UUID][]sbom.Vulnerability
	vulnStats       map[uuid.UUID]map[string]interface{}
}

// NewMockSBOMService creates a new mock SBOM service.
func NewMockSBOMService() *MockSBOMService {
	return &MockSBOMService{
		sboms:           make(map[uuid.UUID]*sbom.SBOM),
		sbomsByImage:    make(map[uuid.UUID]*sbom.SBOM),
		packages:        make(map[uuid.UUID][]sbom.Package),
		vulnerabilities: make(map[uuid.UUID][]sbom.Vulnerability),
		vulnStats:       make(map[uuid.UUID]map[string]interface{}),
	}
}

// AddSBOM adds an SBOM to the mock service.
func (m *MockSBOMService) AddSBOM(s *sbom.SBOM) {
	m.sboms[s.ID] = s
	m.sbomsByImage[s.ImageID] = s
}

// AddPackages adds packages to an SBOM.
func (m *MockSBOMService) AddPackages(sbomID uuid.UUID, packages []sbom.Package) {
	m.packages[sbomID] = packages
}

// AddVulnerabilities adds vulnerabilities to an SBOM.
func (m *MockSBOMService) AddVulnerabilities(sbomID uuid.UUID, vulns []sbom.Vulnerability) {
	m.vulnerabilities[sbomID] = vulns
}

// SetVulnStats sets vulnerability statistics for an SBOM.
func (m *MockSBOMService) SetVulnStats(sbomID uuid.UUID, stats map[string]interface{}) {
	m.vulnStats[sbomID] = stats
}

// Get implements SBOMServiceInterface.
func (m *MockSBOMService) Get(ctx context.Context, id uuid.UUID) (*sbom.SBOM, error) {
	if s, ok := m.sboms[id]; ok {
		return s, nil
	}
	return nil, errors.New("sbom not found")
}

// GetByImageID implements SBOMServiceInterface.
func (m *MockSBOMService) GetByImageID(ctx context.Context, imageID uuid.UUID) (*sbom.SBOM, error) {
	if s, ok := m.sbomsByImage[imageID]; ok {
		return s, nil
	}
	return nil, errors.New("no sbom found for image")
}

// List implements SBOMServiceInterface.
func (m *MockSBOMService) List(ctx context.Context, orgID uuid.UUID, page, pageSize int) (*sbom.SBOMListResponse, error) {
	var summaries []sbom.SBOMSummary
	for _, s := range m.sboms {
		if s.OrgID == orgID {
			summaries = append(summaries, sbom.SBOMSummary{
				ID:           s.ID,
				ImageID:      s.ImageID,
				Format:       s.Format,
				PackageCount: s.PackageCount,
				VulnCount:    s.VulnCount,
				GeneratedAt:  s.GeneratedAt,
			})
		}
	}
	return &sbom.SBOMListResponse{
		SBOMs:      summaries,
		Total:      len(summaries),
		Page:       page,
		PageSize:   pageSize,
		TotalPages: 1,
	}, nil
}

// Delete implements SBOMServiceInterface.
func (m *MockSBOMService) Delete(ctx context.Context, id uuid.UUID) error {
	if _, ok := m.sboms[id]; !ok {
		return errors.New("sbom not found")
	}
	delete(m.sboms, id)
	return nil
}

// GetPackages implements SBOMServiceInterface.
func (m *MockSBOMService) GetPackages(ctx context.Context, sbomID uuid.UUID) ([]sbom.Package, error) {
	if pkgs, ok := m.packages[sbomID]; ok {
		return pkgs, nil
	}
	return []sbom.Package{}, nil
}

// GetVulnerabilities implements SBOMServiceInterface.
func (m *MockSBOMService) GetVulnerabilities(ctx context.Context, sbomID uuid.UUID, filter *sbom.VulnerabilityFilter) ([]sbom.Vulnerability, error) {
	if vulns, ok := m.vulnerabilities[sbomID]; ok {
		return vulns, nil
	}
	return []sbom.Vulnerability{}, nil
}

// GetVulnerabilityStats implements SBOMServiceInterface.
func (m *MockSBOMService) GetVulnerabilityStats(ctx context.Context, sbomID uuid.UUID) (map[string]interface{}, error) {
	if stats, ok := m.vulnStats[sbomID]; ok {
		return stats, nil
	}
	return map[string]interface{}{}, nil
}

// ExportSPDX implements SBOMServiceInterface.
func (m *MockSBOMService) ExportSPDX(ctx context.Context, sbomID string) (map[string]interface{}, error) {
	id, err := uuid.Parse(sbomID)
	if err != nil {
		return nil, err
	}
	if s, ok := m.sboms[id]; ok {
		return s.Content, nil
	}
	return nil, errors.New("sbom not found")
}

// ExportCycloneDX implements SBOMServiceInterface.
func (m *MockSBOMService) ExportCycloneDX(ctx context.Context, sbomID string) (map[string]interface{}, error) {
	id, err := uuid.Parse(sbomID)
	if err != nil {
		return nil, err
	}
	if s, ok := m.sboms[id]; ok {
		return s.Content, nil
	}
	return nil, errors.New("sbom not found")
}

// MockSBOMGenerator implements the handlers.SBOMGeneratorInterface for testing.
type MockSBOMGenerator struct {
	generatedSBOM *sbom.SBOM
	shouldFail    bool
}

// NewMockSBOMGenerator creates a new mock generator.
func NewMockSBOMGenerator() *MockSBOMGenerator {
	return &MockSBOMGenerator{}
}

// SetGeneratedSBOM sets the SBOM that will be returned by Generate.
func (m *MockSBOMGenerator) SetGeneratedSBOM(s *sbom.SBOM) {
	m.generatedSBOM = s
}

// SetShouldFail sets whether Generate should fail.
func (m *MockSBOMGenerator) SetShouldFail(fail bool) {
	m.shouldFail = fail
}

// Generate implements SBOMGeneratorInterface.
func (m *MockSBOMGenerator) Generate(ctx context.Context, req sbom.GenerateRequest) (*sbom.GenerateResult, error) {
	if m.shouldFail {
		return nil, errors.New("generation failed")
	}
	if m.generatedSBOM != nil {
		return &sbom.GenerateResult{
			SBOM:         m.generatedSBOM,
			Status:       "success",
			PackageCount: m.generatedSBOM.PackageCount,
			VulnCount:    m.generatedSBOM.VulnCount,
		}, nil
	}
	// Generate a default SBOM
	newSBOM := &sbom.SBOM{
		ID:           uuid.New(),
		ImageID:      req.ImageID,
		OrgID:        req.OrgID,
		Format:       req.Format,
		Version:      "SPDX-2.3",
		PackageCount: 5,
		GeneratedAt:  time.Now(),
		Scanner:      req.Scanner,
	}
	return &sbom.GenerateResult{
		SBOM:         newSBOM,
		Status:       "success",
		PackageCount: newSBOM.PackageCount,
	}, nil
}

// EnrichWithVulnerabilities implements SBOMGeneratorInterface.
func (m *MockSBOMGenerator) EnrichWithVulnerabilities(ctx context.Context, sbomID uuid.UUID) error {
	if m.shouldFail {
		return errors.New("enrichment failed")
	}
	return nil
}

func TestSBOMHandler_ListSBOMs(t *testing.T) {
	log := logger.New("debug", "text")
	mockSvc := NewMockSBOMService()
	mockGen := NewMockSBOMGenerator()

	orgID := testOrg().ID

	// Add test SBOMs
	for i := 0; i < 3; i++ {
		sbomID := uuid.New()
		mockSvc.AddSBOM(&sbom.SBOM{
			ID:           sbomID,
			ImageID:      uuid.New(),
			OrgID:        orgID,
			Format:       sbom.FormatSPDX,
			Version:      "SPDX-2.3",
			PackageCount: 10 + i,
			GeneratedAt:  time.Now().Add(-time.Duration(i) * time.Hour),
			Scanner:      "ql-rf",
		})
	}

	handler := handlers.NewSBOMHandlerWithInterfaces(mockSvc, mockGen, log)

	t.Run("returns paginated list of SBOMs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom?page=1&page_size=10", nil)
		req = withOrgContext(req)

		rr := executeRequest(handler.ListSBOMs, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response sbom.SBOMListResponse
		require.NoError(t, decodeJSON(rr, &response))
		assert.Len(t, response.SBOMs, 3)
		assert.Equal(t, 1, response.Page)
	})

	t.Run("unauthorized without org context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom", nil)

		rr := executeRequest(handler.ListSBOMs, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestSBOMHandler_GetSBOM(t *testing.T) {
	log := logger.New("debug", "text")
	mockSvc := NewMockSBOMService()
	mockGen := NewMockSBOMGenerator()

	orgID := testOrg().ID
	sbomID := uuid.New()
	imageID := uuid.New()

	// Add test SBOM
	mockSvc.AddSBOM(&sbom.SBOM{
		ID:           sbomID,
		ImageID:      imageID,
		OrgID:        orgID,
		Format:       sbom.FormatSPDX,
		Version:      "SPDX-2.3",
		PackageCount: 15,
		GeneratedAt:  time.Now(),
		Scanner:      "ql-rf",
		Content: map[string]interface{}{
			"spdxVersion": "SPDX-2.3",
		},
	})

	// Add packages
	mockSvc.AddPackages(sbomID, []sbom.Package{
		{
			ID:      uuid.New(),
			SBOMID:  sbomID,
			Name:    "lodash",
			Version: "4.17.21",
			Type:    "npm",
			License: "MIT",
		},
		{
			ID:      uuid.New(),
			SBOMID:  sbomID,
			Name:    "express",
			Version: "4.18.2",
			Type:    "npm",
			License: "MIT",
		},
	})

	handler := handlers.NewSBOMHandlerWithInterfaces(mockSvc, mockGen, log)

	t.Run("returns SBOM by ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/"+sbomID.String(), nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": sbomID.String()})

		rr := executeRequest(handler.GetSBOM, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response sbom.SBOM
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, sbomID, response.ID)
		assert.Equal(t, sbom.FormatSPDX, response.Format)
	})

	t.Run("returns SBOM with packages when requested", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/"+sbomID.String()+"?include_packages=true", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": sbomID.String()})

		rr := executeRequest(handler.GetSBOM, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response sbom.SBOM
		require.NoError(t, decodeJSON(rr, &response))
		assert.Len(t, response.Packages, 2)
	})

	t.Run("returns 404 for non-existent SBOM", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/"+nonExistentID.String(), nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": nonExistentID.String()})

		rr := executeRequest(handler.GetSBOM, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/not-a-uuid", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": "not-a-uuid"})

		rr := executeRequest(handler.GetSBOM, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns 404 for SBOM from different org", func(t *testing.T) {
		// Add SBOM for different org
		otherOrgSBOMID := uuid.New()
		mockSvc.AddSBOM(&sbom.SBOM{
			ID:      otherOrgSBOMID,
			ImageID: uuid.New(),
			OrgID:   uuid.New(), // Different org
			Format:  sbom.FormatSPDX,
		})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/"+otherOrgSBOMID.String(), nil)
		req = withOrgContext(req) // Uses testOrg
		req = withChiURLParams(req, map[string]string{"id": otherOrgSBOMID.String()})

		rr := executeRequest(handler.GetSBOM, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func TestSBOMHandler_GetImageSBOM(t *testing.T) {
	log := logger.New("debug", "text")
	mockSvc := NewMockSBOMService()
	mockGen := NewMockSBOMGenerator()

	orgID := testOrg().ID
	imageID := uuid.New()
	sbomID := uuid.New()

	// Add SBOM for the image
	mockSvc.AddSBOM(&sbom.SBOM{
		ID:           sbomID,
		ImageID:      imageID,
		OrgID:        orgID,
		Format:       sbom.FormatCycloneDX,
		Version:      "CycloneDX-1.5",
		PackageCount: 25,
		GeneratedAt:  time.Now(),
		Scanner:      "trivy",
	})

	handler := handlers.NewSBOMHandlerWithInterfaces(mockSvc, mockGen, log)

	t.Run("returns SBOM for image", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/images/"+imageID.String()+"/sbom", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": imageID.String()})

		rr := executeRequest(handler.GetImageSBOM, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response sbom.SBOM
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, imageID, response.ImageID)
		assert.Equal(t, sbom.FormatCycloneDX, response.Format)
	})

	t.Run("returns 500 for image without SBOM", func(t *testing.T) {
		noSBOMImageID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/images/"+noSBOMImageID.String()+"/sbom", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": noSBOMImageID.String()})

		rr := executeRequest(handler.GetImageSBOM, req)

		// The handler returns 500 on error (see line 57-59 in handler)
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("returns 400 for invalid image ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/images/invalid/sbom", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": "invalid"})

		rr := executeRequest(handler.GetImageSBOM, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestSBOMHandler_GenerateSBOM(t *testing.T) {
	log := logger.New("debug", "text")
	mockSvc := NewMockSBOMService()
	mockGen := NewMockSBOMGenerator()

	imageID := uuid.New()

	handler := handlers.NewSBOMHandlerWithInterfaces(mockSvc, mockGen, log)

	t.Run("generates SBOM in SPDX format", func(t *testing.T) {
		body := map[string]interface{}{
			"format":        "spdx",
			"scanner":       "trivy",
			"include_vulns": false,
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+imageID.String()+"/sbom/generate", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": imageID.String()})

		rr := executeRequest(handler.GenerateSBOM, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var response sbom.SBOMGenerationResponse
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, "success", response.Status)
	})

	t.Run("generates SBOM in CycloneDX format", func(t *testing.T) {
		body := map[string]interface{}{
			"format": "cyclonedx",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+imageID.String()+"/sbom/generate", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": imageID.String()})

		rr := executeRequest(handler.GenerateSBOM, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
	})

	t.Run("returns 400 for invalid format", func(t *testing.T) {
		body := map[string]interface{}{
			"format": "invalid-format",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+imageID.String()+"/sbom/generate", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": imageID.String()})

		rr := executeRequest(handler.GenerateSBOM, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+imageID.String()+"/sbom/generate", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": imageID.String()})

		rr := executeRequest(handler.GenerateSBOM, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns 400 for invalid image ID", func(t *testing.T) {
		body := map[string]interface{}{
			"format": "spdx",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/images/invalid/sbom/generate", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": "invalid"})

		rr := executeRequest(handler.GenerateSBOM, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("unauthorized without org context", func(t *testing.T) {
		body := map[string]interface{}{
			"format": "spdx",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+imageID.String()+"/sbom/generate", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withChiURLParams(req, map[string]string{"id": imageID.String()})

		rr := executeRequest(handler.GenerateSBOM, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("returns 500 when generation fails", func(t *testing.T) {
		mockGen.SetShouldFail(true)
		defer mockGen.SetShouldFail(false)

		body := map[string]interface{}{
			"format": "spdx",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+imageID.String()+"/sbom/generate", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": imageID.String()})

		rr := executeRequest(handler.GenerateSBOM, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestSBOMHandler_GetSBOMVulnerabilities(t *testing.T) {
	log := logger.New("debug", "text")
	mockSvc := NewMockSBOMService()
	mockGen := NewMockSBOMGenerator()

	orgID := testOrg().ID
	sbomID := uuid.New()
	imageID := uuid.New()

	// Add test SBOM
	mockSvc.AddSBOM(&sbom.SBOM{
		ID:           sbomID,
		ImageID:      imageID,
		OrgID:        orgID,
		Format:       sbom.FormatSPDX,
		PackageCount: 10,
		VulnCount:    5,
	})

	// Add vulnerabilities
	cvss8 := 8.5
	cvss7 := 7.2
	cvss4 := 4.0
	mockSvc.AddVulnerabilities(sbomID, []sbom.Vulnerability{
		{
			ID:        uuid.New(),
			SBOMID:    sbomID,
			PackageID: uuid.New(),
			CVEID:     "CVE-2024-1234",
			Severity:  "critical",
			CVSSScore: &cvss8,
		},
		{
			ID:        uuid.New(),
			SBOMID:    sbomID,
			PackageID: uuid.New(),
			CVEID:     "CVE-2024-5678",
			Severity:  "high",
			CVSSScore: &cvss7,
		},
		{
			ID:        uuid.New(),
			SBOMID:    sbomID,
			PackageID: uuid.New(),
			CVEID:     "CVE-2024-9999",
			Severity:  "low",
			CVSSScore: &cvss4,
		},
	})

	// Add stats
	mockSvc.SetVulnStats(sbomID, map[string]interface{}{
		"critical": 1,
		"high":     1,
		"medium":   0,
		"low":      1,
		"total":    3,
	})

	handler := handlers.NewSBOMHandlerWithInterfaces(mockSvc, mockGen, log)

	t.Run("returns vulnerabilities for SBOM", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/"+sbomID.String()+"/vulnerabilities", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": sbomID.String()})

		rr := executeRequest(handler.GetSBOMVulnerabilities, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		require.NoError(t, decodeJSON(rr, &response))
		vulns := response["vulnerabilities"].([]interface{})
		assert.Len(t, vulns, 3)
		assert.Equal(t, float64(3), response["count"])
	})

	t.Run("filters by severity", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/"+sbomID.String()+"/vulnerabilities?severity=critical&severity=high", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": sbomID.String()})

		rr := executeRequest(handler.GetSBOMVulnerabilities, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("filters by min CVSS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/"+sbomID.String()+"/vulnerabilities?min_cvss=7.0", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": sbomID.String()})

		rr := executeRequest(handler.GetSBOMVulnerabilities, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("filters by has_exploit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/"+sbomID.String()+"/vulnerabilities?has_exploit=true", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": sbomID.String()})

		rr := executeRequest(handler.GetSBOMVulnerabilities, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("filters by fix_available", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/"+sbomID.String()+"/vulnerabilities?fix_available=true", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": sbomID.String()})

		rr := executeRequest(handler.GetSBOMVulnerabilities, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("returns 404 for non-existent SBOM", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/"+nonExistentID.String()+"/vulnerabilities", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": nonExistentID.String()})

		rr := executeRequest(handler.GetSBOMVulnerabilities, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns 400 for invalid SBOM ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/invalid/vulnerabilities", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": "invalid"})

		rr := executeRequest(handler.GetSBOMVulnerabilities, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestSBOMHandler_ExportSBOM(t *testing.T) {
	log := logger.New("debug", "text")
	mockSvc := NewMockSBOMService()
	mockGen := NewMockSBOMGenerator()

	orgID := testOrg().ID
	sbomID := uuid.New()

	// Add test SBOM
	mockSvc.AddSBOM(&sbom.SBOM{
		ID:      sbomID,
		ImageID: uuid.New(),
		OrgID:   orgID,
		Format:  sbom.FormatSPDX,
		Version: "SPDX-2.3",
		Content: map[string]interface{}{
			"spdxVersion": "SPDX-2.3",
			"packages":    []interface{}{},
		},
	})

	handler := handlers.NewSBOMHandlerWithInterfaces(mockSvc, mockGen, log)

	t.Run("exports SBOM in SPDX format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/"+sbomID.String()+"/export?format=spdx", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": sbomID.String()})

		rr := executeRequest(handler.ExportSBOM, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response sbom.SBOMExportResponse
		require.NoError(t, decodeJSON(rr, &response))
		assert.Equal(t, sbom.FormatSPDX, response.Format)
	})

	t.Run("exports SBOM in CycloneDX format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/"+sbomID.String()+"/export?format=cyclonedx", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": sbomID.String()})

		rr := executeRequest(handler.ExportSBOM, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("uses original format when not specified", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/"+sbomID.String()+"/export", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": sbomID.String()})

		rr := executeRequest(handler.ExportSBOM, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("returns 400 for invalid format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/"+sbomID.String()+"/export?format=invalid", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": sbomID.String()})

		rr := executeRequest(handler.ExportSBOM, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns 404 for non-existent SBOM", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/"+nonExistentID.String()+"/export?format=spdx", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": nonExistentID.String()})

		rr := executeRequest(handler.ExportSBOM, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns 400 for invalid SBOM ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/invalid/export", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": "invalid"})

		rr := executeRequest(handler.ExportSBOM, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestSBOMHandler_DeleteSBOM(t *testing.T) {
	log := logger.New("debug", "text")
	mockSvc := NewMockSBOMService()
	mockGen := NewMockSBOMGenerator()

	orgID := testOrg().ID
	sbomID := uuid.New()

	// Add test SBOM
	mockSvc.AddSBOM(&sbom.SBOM{
		ID:      sbomID,
		ImageID: uuid.New(),
		OrgID:   orgID,
		Format:  sbom.FormatSPDX,
	})

	handler := handlers.NewSBOMHandlerWithInterfaces(mockSvc, mockGen, log)

	t.Run("deletes SBOM successfully", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/sbom/"+sbomID.String(), nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": sbomID.String()})

		rr := executeRequest(handler.DeleteSBOM, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("returns 404 for non-existent SBOM", func(t *testing.T) {
		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/sbom/"+nonExistentID.String(), nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": nonExistentID.String()})

		rr := executeRequest(handler.DeleteSBOM, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns 400 for invalid SBOM ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/sbom/invalid", nil)
		req = withOrgContext(req)
		req = withChiURLParams(req, map[string]string{"id": "invalid"})

		rr := executeRequest(handler.DeleteSBOM, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns 404 for SBOM from different org", func(t *testing.T) {
		// Add SBOM for different org
		otherOrgSBOMID := uuid.New()
		mockSvc.AddSBOM(&sbom.SBOM{
			ID:      otherOrgSBOMID,
			ImageID: uuid.New(),
			OrgID:   uuid.New(), // Different org
			Format:  sbom.FormatSPDX,
		})

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/sbom/"+otherOrgSBOMID.String(), nil)
		req = withOrgContext(req) // Uses testOrg
		req = withChiURLParams(req, map[string]string{"id": otherOrgSBOMID.String()})

		rr := executeRequest(handler.DeleteSBOM, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("unauthorized without org context", func(t *testing.T) {
		// Re-add the deleted SBOM
		mockSvc.AddSBOM(&sbom.SBOM{
			ID:      sbomID,
			ImageID: uuid.New(),
			OrgID:   orgID,
			Format:  sbom.FormatSPDX,
		})

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/sbom/"+sbomID.String(), nil)
		req = withChiURLParams(req, map[string]string{"id": sbomID.String()})

		rr := executeRequest(handler.DeleteSBOM, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}
