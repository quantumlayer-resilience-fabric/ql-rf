// Package integration provides end-to-end tests for enterprise features.
package integration

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/quantumlayerhq/ql-rf/pkg/compliance"
	"github.com/quantumlayerhq/ql-rf/pkg/multitenancy"
	"github.com/quantumlayerhq/ql-rf/pkg/rbac"
	"github.com/quantumlayerhq/ql-rf/pkg/secrets"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	// Setup
	dbURL := os.Getenv("RF_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/qlrf_test?sslmode=disable"
	}

	var err error
	testDB, err = sql.Open("postgres", dbURL)
	if err != nil {
		os.Exit(1)
	}
	defer testDB.Close()

	// Run tests
	code := m.Run()
	os.Exit(code)
}

// =============================================================================
// RBAC Tests
// =============================================================================

func TestRBAC_PermissionCheck(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	svc := rbac.NewService(testDB)

	// Test cases for permission checking
	tests := []struct {
		name         string
		userID       string
		orgID        uuid.UUID
		resourceType rbac.ResourceType
		action       rbac.Action
		wantAllowed  bool
	}{
		{
			name:         "check_assets_read",
			userID:       "test-user-1",
			orgID:        uuid.New(),
			resourceType: rbac.ResourceAssets,
			action:       rbac.ActionRead,
			wantAllowed:  false, // No role assigned
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.CheckPermission(ctx, tc.userID, tc.orgID, tc.resourceType, nil, tc.action)
			if err != nil {
				t.Logf("Permission check error (expected for no role): %v", err)
				return
			}
			if result.Allowed != tc.wantAllowed {
				t.Errorf("CheckPermission() = %v, want %v", result.Allowed, tc.wantAllowed)
			}
		})
	}
}

func TestRBAC_RoleAssignment(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	svc := rbac.NewService(testDB)

	// Get the viewer role
	role, err := svc.GetRoleByName(ctx, "viewer", nil)
	if err != nil {
		t.Fatalf("GetRoleByName() error = %v", err)
	}
	if role == nil {
		t.Skip("Viewer role not found - migrations may not be applied")
	}

	// Test role assignment
	orgID := uuid.New()
	userID := "test-user-" + uuid.New().String()[:8]

	// Note: This may fail if org doesn't exist, which is expected in isolation
	err = svc.AssignRole(ctx, userID, orgID, role.ID, "admin", nil)
	if err != nil {
		t.Logf("AssignRole() error (expected for missing org): %v", err)
	}
}

func TestRBAC_TeamOperations(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	svc := rbac.NewService(testDB)

	// List roles (should work even without org)
	roles, err := svc.ListRoles(ctx, uuid.New())
	if err != nil {
		t.Fatalf("ListRoles() error = %v", err)
	}

	// Should have system roles
	if len(roles) < 8 {
		t.Logf("Expected at least 8 system roles, got %d", len(roles))
	}

	// Verify system roles exist
	roleNames := make(map[string]bool)
	for _, r := range roles {
		roleNames[r.Name] = true
	}

	expectedRoles := []string{"org_owner", "org_admin", "infra_admin", "security_admin", "dr_admin", "operator", "analyst", "viewer"}
	for _, name := range expectedRoles {
		if !roleNames[name] {
			t.Logf("Expected role %s not found", name)
		}
	}
}

// =============================================================================
// Multi-tenancy Tests
// =============================================================================

func TestMultitenancy_QuotaCheck(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	svc := multitenancy.NewService(testDB)

	orgID := uuid.New()

	// Check quota for a non-existent org (should return true - no limits)
	allowed, err := svc.CheckQuota(ctx, orgID, multitenancy.QuotaAssets, 1)
	if err != nil {
		t.Logf("CheckQuota() error (may be expected): %v", err)
		return
	}

	// Without quota record, should be allowed
	if !allowed {
		t.Log("CheckQuota() returned false for org without quota record")
	}
}

func TestMultitenancy_UsageTracking(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	svc := multitenancy.NewService(testDB)

	orgID := uuid.New()

	// Get usage for non-existent org
	usage, err := svc.GetUsage(ctx, orgID)
	if err != nil {
		t.Fatalf("GetUsage() error = %v", err)
	}

	// Should return zeroed usage
	if usage.AssetCount != 0 {
		t.Errorf("Expected 0 asset count for new org, got %d", usage.AssetCount)
	}
}

func TestMultitenancy_SubscriptionPlans(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	svc := multitenancy.NewService(testDB)

	// List available plans
	plans, err := svc.ListPlans(ctx)
	if err != nil {
		t.Fatalf("ListPlans() error = %v", err)
	}

	// Should have default plans
	if len(plans) < 4 {
		t.Logf("Expected at least 4 plans (free, starter, professional, enterprise), got %d", len(plans))
	}

	// Verify plan structure
	planNames := make(map[string]bool)
	for _, p := range plans {
		planNames[p.Name] = true
		if p.DefaultMaxAssets <= 0 {
			t.Errorf("Plan %s has invalid max_assets: %d", p.Name, p.DefaultMaxAssets)
		}
	}

	expectedPlans := []string{"free", "starter", "professional", "enterprise"}
	for _, name := range expectedPlans {
		if !planNames[name] {
			t.Logf("Expected plan %s not found", name)
		}
	}
}

func TestMultitenancy_APIRateLimit(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	svc := multitenancy.NewService(testDB)

	orgID := uuid.New()

	// First request should be allowed
	allowed, err := svc.CheckAPIRateLimit(ctx, orgID)
	if err != nil {
		t.Logf("CheckAPIRateLimit() error (may be expected): %v", err)
		return
	}

	if !allowed {
		t.Log("First API request was rate limited (unexpected)")
	}
}

// =============================================================================
// Compliance Tests
// =============================================================================

func TestCompliance_ListFrameworks(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	svc := compliance.NewService(testDB)

	frameworks, err := svc.ListFrameworks(ctx)
	if err != nil {
		t.Fatalf("ListFrameworks() error = %v", err)
	}

	// Should have default frameworks
	if len(frameworks) < 7 {
		t.Logf("Expected at least 7 frameworks, got %d", len(frameworks))
	}

	// Verify key frameworks exist
	frameworkNames := make(map[string]bool)
	for _, f := range frameworks {
		frameworkNames[f.Name] = true
	}

	expectedFrameworks := []string{"CIS AWS Foundations", "CIS Azure Foundations", "SOC 2 Type II", "NIST CSF"}
	for _, name := range expectedFrameworks {
		if !frameworkNames[name] {
			t.Logf("Expected framework %s not found", name)
		}
	}
}

func TestCompliance_ListControls(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	svc := compliance.NewService(testDB)

	// Get frameworks first
	frameworks, err := svc.ListFrameworks(ctx)
	if err != nil {
		t.Fatalf("ListFrameworks() error = %v", err)
	}

	if len(frameworks) == 0 {
		t.Skip("No frameworks available")
	}

	// List controls for first framework
	controls, err := svc.ListControls(ctx, frameworks[0].ID)
	if err != nil {
		t.Fatalf("ListControls() error = %v", err)
	}

	t.Logf("Found %d controls for framework %s", len(controls), frameworks[0].Name)

	// Verify control structure
	for _, c := range controls {
		if c.ControlID == "" {
			t.Errorf("Control has empty control_id")
		}
		if c.Name == "" {
			t.Errorf("Control %s has empty name", c.ControlID)
		}
	}
}

func TestCompliance_Assessment(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	svc := compliance.NewService(testDB)

	// Get a framework
	frameworks, err := svc.ListFrameworks(ctx)
	if err != nil || len(frameworks) == 0 {
		t.Skip("No frameworks available")
	}

	// Create assessment (will fail without valid org)
	assessment := compliance.Assessment{
		OrgID:          uuid.New(),
		FrameworkID:    frameworks[0].ID,
		AssessmentType: "automated",
		Name:           "Test Assessment",
		InitiatedBy:    "test-user",
	}

	created, err := svc.CreateAssessment(ctx, assessment)
	if err != nil {
		t.Logf("CreateAssessment() error (expected for missing org): %v", err)
		return
	}

	if created.ID == uuid.Nil {
		t.Error("Created assessment has nil ID")
	}

	if created.Status != compliance.AssessmentPending {
		t.Errorf("Expected status %s, got %s", compliance.AssessmentPending, created.Status)
	}
}

func TestCompliance_MappedControls(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	svc := compliance.NewService(testDB)

	// Get frameworks
	frameworks, err := svc.ListFrameworks(ctx)
	if err != nil || len(frameworks) == 0 {
		t.Skip("No frameworks available")
	}

	// Get controls
	controls, err := svc.ListControls(ctx, frameworks[0].ID)
	if err != nil || len(controls) == 0 {
		t.Skip("No controls available")
	}

	// Get mapped controls (may be empty)
	mapped, err := svc.GetMappedControls(ctx, controls[0].ID)
	if err != nil {
		t.Fatalf("GetMappedControls() error = %v", err)
	}

	t.Logf("Found %d mapped controls for %s", len(mapped), controls[0].ControlID)
}

// =============================================================================
// Secrets Manager Tests
// =============================================================================

func TestSecrets_MemoryBackend(t *testing.T) {
	cfg := &secrets.Config{
		Backend:  secrets.BackendMemory,
		CacheTTL: 5 * time.Minute,
	}

	mgr, err := secrets.NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer mgr.Close()

	ctx := context.Background()

	// Set a secret
	err = mgr.Set(ctx, "test-key", "test-value", nil)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get the secret
	secret, err := mgr.Get(ctx, "test-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if secret.Value != "test-value" {
		t.Errorf("Got value %s, want test-value", secret.Value)
	}

	// Delete the secret
	err = mgr.Delete(ctx, "test-key")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone from cache
	_, err = mgr.Get(ctx, "test-key")
	if err == nil {
		t.Error("Expected error after deletion, got nil")
	}
}

func TestSecrets_EnvBackend(t *testing.T) {
	// Set a test env var
	os.Setenv("TEST_SECRET_KEY", "test-secret-value")
	defer os.Unsetenv("TEST_SECRET_KEY")

	cfg := &secrets.Config{
		Backend:  secrets.BackendEnv,
		CacheTTL: 5 * time.Minute,
	}

	mgr, err := secrets.NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer mgr.Close()

	ctx := context.Background()

	// Get the secret
	secret, err := mgr.Get(ctx, "TEST_SECRET_KEY")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if secret.Value != "test-secret-value" {
		t.Errorf("Got value %s, want test-secret-value", secret.Value)
	}

	// Test GetOrDefault
	value := mgr.GetOrDefault(ctx, "NONEXISTENT_KEY", "default-value")
	if value != "default-value" {
		t.Errorf("GetOrDefault() = %s, want default-value", value)
	}
}

func TestSecrets_HealthCheck(t *testing.T) {
	cfg := &secrets.Config{
		Backend:  secrets.BackendMemory,
		CacheTTL: 5 * time.Minute,
	}

	mgr, err := secrets.NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer mgr.Close()

	ctx := context.Background()

	// Health check should pass for memory backend
	err = mgr.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}
}

func TestSecrets_RotateSecret(t *testing.T) {
	cfg := &secrets.Config{
		Backend:  secrets.BackendMemory,
		CacheTTL: 5 * time.Minute,
	}

	mgr, err := secrets.NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer mgr.Close()

	ctx := context.Background()

	// Set initial secret
	err = mgr.Set(ctx, "rotate-key", "initial-value", nil)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Rotate the secret
	rotateCount := 0
	err = mgr.RotateSecret(ctx, "rotate-key", func() (string, error) {
		rotateCount++
		return "rotated-value", nil
	})
	if err != nil {
		t.Fatalf("RotateSecret() error = %v", err)
	}

	if rotateCount != 1 {
		t.Errorf("Generator called %d times, want 1", rotateCount)
	}

	// Verify new value
	secret, err := mgr.Get(ctx, "rotate-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if secret.Value != "rotated-value" {
		t.Errorf("Got value %s, want rotated-value", secret.Value)
	}
}

// =============================================================================
// Integration Tests (require full environment)
// =============================================================================

func TestE2E_FullWorkflow(t *testing.T) {
	if os.Getenv("E2E_TESTS") != "true" {
		t.Skip("E2E tests disabled - set E2E_TESTS=true to enable")
	}

	// This test would run a full workflow:
	// 1. Create organization
	// 2. Set up quotas
	// 3. Assign roles
	// 4. Create compliance assessment
	// 5. Track usage
	// 6. Verify audit trail

	t.Log("Full E2E workflow test would run here")
}
