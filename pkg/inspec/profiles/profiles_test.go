package profiles

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/quantumlayerhq/ql-rf/pkg/inspec"
)

// TestGetCISAWSMappings tests CIS AWS control mappings
func TestGetCISAWSMappings(t *testing.T) {
	mappings := GetCISAWSMappings()

	if len(mappings) == 0 {
		t.Error("GetCISAWSMappings() returned empty slice")
	}

	// Test that all mappings have required fields
	for i, mapping := range mappings {
		if mapping.InSpecControlID == "" {
			t.Errorf("mapping[%d] has empty InSpecControlID", i)
		}
		if mapping.MappingConfidence <= 0 || mapping.MappingConfidence > 1 {
			t.Errorf("mapping[%d] has invalid confidence: %f", i, mapping.MappingConfidence)
		}
		if mapping.Notes == "" {
			t.Errorf("mapping[%d] has empty Notes", i)
		}
	}
}

// TestGetCISAWSMappings_SpecificControls tests specific AWS control mappings
func TestGetCISAWSMappings_SpecificControls(t *testing.T) {
	mappings := GetCISAWSMappings()

	// Convert to map for easier testing
	controlMap := make(map[string]inspec.ControlMapping)
	for _, m := range mappings {
		controlMap[m.InSpecControlID] = m
	}

	tests := []struct {
		controlID         string
		wantConfidence    float64
		shouldExist       bool
		descriptionSubstr string
	}{
		{
			controlID:         "cis-aws-foundations-benchmark-1.4",
			wantConfidence:    1.0,
			shouldExist:       true,
			descriptionSubstr: "root account",
		},
		{
			controlID:         "cis-aws-foundations-benchmark-1.5",
			wantConfidence:    1.0,
			shouldExist:       true,
			descriptionSubstr: "MFA",
		},
		{
			controlID:         "cis-aws-foundations-benchmark-3.1",
			wantConfidence:    1.0,
			shouldExist:       true,
			descriptionSubstr: "CloudTrail",
		},
		{
			controlID:      "non-existent-control",
			shouldExist:    false,
			wantConfidence: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.controlID, func(t *testing.T) {
			control, exists := controlMap[tt.controlID]
			if exists != tt.shouldExist {
				t.Errorf("control %s exists = %v, want %v", tt.controlID, exists, tt.shouldExist)
				return
			}

			if tt.shouldExist {
				if control.MappingConfidence != tt.wantConfidence {
					t.Errorf("control %s confidence = %f, want %f",
						tt.controlID, control.MappingConfidence, tt.wantConfidence)
				}
			}
		})
	}
}

// TestGetCISAWSMappings_Coverage tests coverage of CIS AWS sections
func TestGetCISAWSMappings_Coverage(t *testing.T) {
	mappings := GetCISAWSMappings()

	// Count controls by section
	sectionCounts := make(map[string]int)
	for _, m := range mappings {
		// Extract section from control ID (e.g., "1" from "cis-aws-foundations-benchmark-1.4")
		var section string
		if len(m.InSpecControlID) > 30 {
			section = string(m.InSpecControlID[30])
		}
		sectionCounts[section]++
	}

	// Verify we have controls from multiple sections
	if len(sectionCounts) < 3 {
		t.Errorf("Expected controls from at least 3 sections, got %d", len(sectionCounts))
	}

	t.Logf("CIS AWS controls by section: %+v", sectionCounts)
}

// TestCreateCISAWSProfile tests AWS profile creation
func TestCreateCISAWSProfile(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	svc := inspec.NewService(db)
	ctx := context.Background()
	frameworkID := uuid.New()

	tests := []struct {
		name    string
		mockFn  func()
		wantErr bool
	}{
		{
			name: "successful creation",
			mockFn: func() {
				mock.ExpectExec("INSERT INTO inspec_profiles").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "database error",
			mockFn: func() {
				mock.ExpectExec("INSERT INTO inspec_profiles").
					WillReturnError(sqlmock.ErrCancelled)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			profile, err := CreateCISAWSProfile(ctx, svc, frameworkID)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateCISAWSProfile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && profile != nil {
				if profile.Name != "cis-aws-foundations-benchmark" {
					t.Errorf("profile.Name = %v, want cis-aws-foundations-benchmark", profile.Name)
				}
				if len(profile.Platforms) == 0 {
					t.Error("profile.Platforms is empty")
				}
				if profile.Platforms[0] != "aws" {
					t.Errorf("profile.Platforms[0] = %v, want aws", profile.Platforms[0])
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestGetCISLinuxLevel1Mappings tests CIS Linux Level 1 control mappings
func TestGetCISLinuxLevel1Mappings(t *testing.T) {
	mappings := GetCISLinuxLevel1Mappings()

	if len(mappings) == 0 {
		t.Error("GetCISLinuxLevel1Mappings() returned empty slice")
	}

	// Test that all mappings have required fields
	for i, mapping := range mappings {
		if mapping.InSpecControlID == "" {
			t.Errorf("mapping[%d] has empty InSpecControlID", i)
		}
		if mapping.MappingConfidence <= 0 || mapping.MappingConfidence > 1 {
			t.Errorf("mapping[%d] has invalid confidence: %f", i, mapping.MappingConfidence)
		}
	}

	t.Logf("CIS Linux Level 1 has %d control mappings", len(mappings))
}

// TestGetCISLinuxLevel2Mappings tests CIS Linux Level 2 control mappings
func TestGetCISLinuxLevel2Mappings(t *testing.T) {
	level1 := GetCISLinuxLevel1Mappings()
	level2 := GetCISLinuxLevel2Mappings()

	if len(level2) <= len(level1) {
		t.Errorf("Level 2 should have more controls than Level 1, got L1=%d, L2=%d",
			len(level1), len(level2))
	}

	// Level 2 should include all Level 1 controls
	level1IDs := make(map[string]bool)
	for _, m := range level1 {
		level1IDs[m.InSpecControlID] = true
	}

	foundLevel1Count := 0
	for _, m := range level2 {
		if level1IDs[m.InSpecControlID] {
			foundLevel1Count++
		}
	}

	// Should find most if not all Level 1 controls in Level 2
	if foundLevel1Count < len(level1)-5 {
		t.Errorf("Level 2 should include most Level 1 controls, found %d of %d",
			foundLevel1Count, len(level1))
	}

	t.Logf("CIS Linux Level 2 has %d controls (includes %d from Level 1)",
		len(level2), foundLevel1Count)
}

// TestGetCISLinuxLevel1Mappings_SpecificControls tests specific Linux control mappings
func TestGetCISLinuxLevel1Mappings_SpecificControls(t *testing.T) {
	mappings := GetCISLinuxLevel1Mappings()

	controlMap := make(map[string]inspec.ControlMapping)
	for _, m := range mappings {
		controlMap[m.InSpecControlID] = m
	}

	tests := []struct {
		name        string
		controlID   string
		shouldExist bool
	}{
		{
			name:        "cramfs kernel module",
			controlID:   "xccdf_org.cisecurity.benchmarks_rule_1.1.1.1_Ensure_cramfs_kernel_module_is_not_available",
			shouldExist: true,
		},
		{
			name:        "auditd installed",
			controlID:   "xccdf_org.cisecurity.benchmarks_rule_4.1.1.1_Ensure_auditd_is_installed",
			shouldExist: true,
		},
		{
			name:        "SSH root login disabled",
			controlID:   "xccdf_org.cisecurity.benchmarks_rule_5.2.10_Ensure_SSH_PermitRootLogin_is_disabled",
			shouldExist: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, exists := controlMap[tt.controlID]
			if exists != tt.shouldExist {
				t.Errorf("control %s exists = %v, want %v", tt.controlID, exists, tt.shouldExist)
			}
		})
	}
}

// TestCreateCISLinuxProfile tests Linux profile creation
func TestCreateCISLinuxProfile(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	svc := inspec.NewService(db)
	ctx := context.Background()
	frameworkID := uuid.New()

	tests := []struct {
		name    string
		level   int
		mockFn  func()
		wantErr bool
	}{
		{
			name:  "Level 1 creation",
			level: 1,
			mockFn: func() {
				mock.ExpectExec("INSERT INTO inspec_profiles").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name:  "Level 2 creation",
			level: 2,
			mockFn: func() {
				mock.ExpectExec("INSERT INTO inspec_profiles").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name:  "database error",
			level: 1,
			mockFn: func() {
				mock.ExpectExec("INSERT INTO inspec_profiles").
					WillReturnError(sqlmock.ErrCancelled)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			profile, err := CreateCISLinuxProfile(ctx, svc, frameworkID, tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateCISLinuxProfile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && profile != nil {
				expectedName := "cis-linux-level-1"
				if tt.level == 2 {
					expectedName = "cis-linux-level-2"
				}
				if profile.Name != expectedName {
					t.Errorf("profile.Name = %v, want %v", profile.Name, expectedName)
				}

				// Should support multiple Linux platforms
				if len(profile.Platforms) == 0 {
					t.Error("profile.Platforms is empty")
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestGetSOC2Mappings tests SOC2 control mappings
func TestGetSOC2Mappings(t *testing.T) {
	mappings := GetSOC2Mappings()

	if len(mappings) == 0 {
		t.Error("GetSOC2Mappings() returned empty slice")
	}

	// Test that all mappings have required fields
	for i, mapping := range mappings {
		if mapping.InSpecControlID == "" {
			t.Errorf("mapping[%d] has empty InSpecControlID", i)
		}
		if mapping.MappingConfidence <= 0 || mapping.MappingConfidence > 1 {
			t.Errorf("mapping[%d] has invalid confidence: %f", i, mapping.MappingConfidence)
		}
		if mapping.Notes == "" {
			t.Errorf("mapping[%d] has empty Notes", i)
		}
	}

	t.Logf("SOC2 has %d control mappings", len(mappings))
}

// TestGetSOC2Mappings_TrustServiceCriteria tests SOC2 trust service criteria coverage
func TestGetSOC2Mappings_TrustServiceCriteria(t *testing.T) {
	mappings := GetSOC2Mappings()

	// Count controls by trust service criteria
	criteriaFound := make(map[string]int)
	for _, m := range mappings {
		// Extract criteria prefix (CC, A, C, P)
		if len(m.InSpecControlID) >= 5 {
			prefix := m.InSpecControlID[5:7]
			criteriaFound[prefix]++
		}
	}

	// Should have controls from multiple trust service criteria
	expectedCriteria := []string{"cc", "a1", "c1", "p1", "p2"}
	for _, criteria := range expectedCriteria {
		if count := criteriaFound[criteria]; count == 0 {
			t.Logf("Warning: No controls found for criteria %s", criteria)
		}
	}

	t.Logf("SOC2 controls by criteria: %+v", criteriaFound)
}

// TestGetSOC2Mappings_SpecificControls tests specific SOC2 control mappings
func TestGetSOC2Mappings_SpecificControls(t *testing.T) {
	mappings := GetSOC2Mappings()

	controlMap := make(map[string]inspec.ControlMapping)
	for _, m := range mappings {
		controlMap[m.InSpecControlID] = m
	}

	tests := []struct {
		name              string
		controlID         string
		shouldExist       bool
		minConfidence     float64
		descriptionSubstr string
	}{
		{
			name:              "access control",
			controlID:         "soc2-cc6.1-access-control",
			shouldExist:       true,
			minConfidence:     0.8,
			descriptionSubstr: "CC6.1",
		},
		{
			name:          "encryption",
			controlID:     "soc2-cc6.6-encryption",
			shouldExist:   true,
			minConfidence: 1.0,
		},
		{
			name:          "monitoring",
			controlID:     "soc2-cc7.2-monitoring",
			shouldExist:   true,
			minConfidence: 1.0,
		},
		{
			name:          "backup",
			controlID:     "soc2-a1.2-backup",
			shouldExist:   true,
			minConfidence: 1.0,
		},
		{
			name:          "data encryption at rest",
			controlID:     "soc2-data-encryption-at-rest",
			shouldExist:   true,
			minConfidence: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			control, exists := controlMap[tt.controlID]
			if exists != tt.shouldExist {
				t.Errorf("control %s exists = %v, want %v", tt.controlID, exists, tt.shouldExist)
				return
			}

			if tt.shouldExist {
				if control.MappingConfidence < tt.minConfidence {
					t.Errorf("control %s confidence = %f, want >= %f",
						tt.controlID, control.MappingConfidence, tt.minConfidence)
				}
			}
		})
	}
}

// TestCreateSOC2Profile tests SOC2 profile creation
func TestCreateSOC2Profile(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	svc := inspec.NewService(db)
	ctx := context.Background()
	frameworkID := uuid.New()

	tests := []struct {
		name    string
		mockFn  func()
		wantErr bool
	}{
		{
			name: "successful creation",
			mockFn: func() {
				mock.ExpectExec("INSERT INTO inspec_profiles").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "database error",
			mockFn: func() {
				mock.ExpectExec("INSERT INTO inspec_profiles").
					WillReturnError(sqlmock.ErrCancelled)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			profile, err := CreateSOC2Profile(ctx, svc, frameworkID)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSOC2Profile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && profile != nil {
				if profile.Name != "soc2-type-ii-baseline" {
					t.Errorf("profile.Name = %v, want soc2-type-ii-baseline", profile.Name)
				}

				// Should support multiple platforms
				if len(profile.Platforms) == 0 {
					t.Error("profile.Platforms is empty")
				}

				// Should include common platforms
				platformMap := make(map[string]bool)
				for _, p := range profile.Platforms {
					platformMap[p] = true
				}

				expectedPlatforms := []string{"linux", "aws", "azure"}
				for _, expected := range expectedPlatforms {
					if !platformMap[expected] {
						t.Errorf("profile should support platform %s", expected)
					}
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestMappingConfidenceLevels tests mapping confidence levels
func TestMappingConfidenceLevels(t *testing.T) {
	allMappings := []struct {
		name     string
		mappings []inspec.ControlMapping
	}{
		{"CIS AWS", GetCISAWSMappings()},
		{"CIS Linux Level 1", GetCISLinuxLevel1Mappings()},
		{"CIS Linux Level 2", GetCISLinuxLevel2Mappings()},
		{"SOC2", GetSOC2Mappings()},
	}

	for _, test := range allMappings {
		t.Run(test.name, func(t *testing.T) {
			confidenceLevels := make(map[float64]int)
			for _, m := range test.mappings {
				confidenceLevels[m.MappingConfidence]++
			}

			t.Logf("%s confidence distribution: %+v", test.name, confidenceLevels)

			// All confidences should be between 0 and 1
			for confidence := range confidenceLevels {
				if confidence <= 0 || confidence > 1 {
					t.Errorf("invalid confidence level: %f", confidence)
				}
			}
		})
	}
}

// TestProfileNamingConventions tests profile naming conventions
func TestProfileNamingConventions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	svc := inspec.NewService(db)
	ctx := context.Background()
	frameworkID := uuid.New()

	tests := []struct {
		name         string
		createFunc   func() (*inspec.Profile, error)
		expectedName string
		mockFn       func()
	}{
		{
			name: "CIS AWS",
			createFunc: func() (*inspec.Profile, error) {
				return CreateCISAWSProfile(ctx, svc, frameworkID)
			},
			expectedName: "cis-aws-foundations-benchmark",
			mockFn: func() {
				mock.ExpectExec("INSERT INTO inspec_profiles").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name: "CIS Linux Level 1",
			createFunc: func() (*inspec.Profile, error) {
				return CreateCISLinuxProfile(ctx, svc, frameworkID, 1)
			},
			expectedName: "cis-linux-level-1",
			mockFn: func() {
				mock.ExpectExec("INSERT INTO inspec_profiles").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name: "SOC2",
			createFunc: func() (*inspec.Profile, error) {
				return CreateSOC2Profile(ctx, svc, frameworkID)
			},
			expectedName: "soc2-type-ii-baseline",
			mockFn: func() {
				mock.ExpectExec("INSERT INTO inspec_profiles").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			profile, err := tt.createFunc()
			if err != nil {
				t.Fatalf("failed to create profile: %v", err)
			}

			if profile.Name != tt.expectedName {
				t.Errorf("profile.Name = %v, want %v", profile.Name, tt.expectedName)
			}

			// All profile names should be lowercase with hyphens
			for _, c := range profile.Name {
				if c >= 'A' && c <= 'Z' {
					t.Errorf("profile name should be lowercase: %s", profile.Name)
					break
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestControlMappingQuality tests the quality of control mappings
func TestControlMappingQuality(t *testing.T) {
	tests := []struct {
		name     string
		mappings []inspec.ControlMapping
	}{
		{"CIS AWS", GetCISAWSMappings()},
		{"CIS Linux L1", GetCISLinuxLevel1Mappings()},
		{"SOC2", GetSOC2Mappings()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			highConfidenceCount := 0
			uniqueControls := make(map[string]bool)

			for _, m := range tt.mappings {
				// Track unique control IDs
				uniqueControls[m.InSpecControlID] = true

				// Count high confidence mappings
				if m.MappingConfidence >= 0.9 {
					highConfidenceCount++
				}

				// All controls should have notes
				if m.Notes == "" {
					t.Errorf("control %s missing notes", m.InSpecControlID)
				}
			}

			// Check for duplicate control IDs
			if len(uniqueControls) != len(tt.mappings) {
				t.Errorf("found duplicate control IDs: unique=%d, total=%d",
					len(uniqueControls), len(tt.mappings))
			}

			// Log quality metrics
			highConfidencePercent := float64(highConfidenceCount) / float64(len(tt.mappings)) * 100
			t.Logf("%s: %d controls, %.1f%% high confidence (>=0.9)",
				tt.name, len(tt.mappings), highConfidencePercent)
		})
	}
}
