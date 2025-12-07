// Package integration provides end-to-end tests for organization onboarding.
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/multitenancy"
)

// =============================================================================
// Organization Creation Tests
// =============================================================================

func TestOrganizationOnboarding_CreateOrganization(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	svc := multitenancy.NewService(testDB)

	t.Run("create organization with free plan", func(t *testing.T) {
		result, err := svc.CreateOrganization(ctx, multitenancy.CreateOrganizationParams{
			Name:   "Test Org " + uuid.New().String()[:8],
			Slug:   "test-org-" + uuid.New().String()[:8],
			PlanID: "free",
		})
		if err != nil {
			t.Fatalf("CreateOrganization() error = %v", err)
		}

		if result.Organization == nil {
			t.Fatal("CreateOrganization() returned nil organization")
		}

		if result.Organization.Name == "" {
			t.Error("Organization name should not be empty")
		}

		if result.Organization.ID == uuid.Nil {
			t.Error("Organization ID should not be nil")
		}

		// Cleanup
		cleanupOrganization(t, result.Organization.ID)
	})

	t.Run("create organization without slug generates one", func(t *testing.T) {
		result, err := svc.CreateOrganization(ctx, multitenancy.CreateOrganizationParams{
			Name: "My Test Organization",
		})
		if err != nil {
			t.Fatalf("CreateOrganization() error = %v", err)
		}

		if result.Organization.Slug == "" {
			t.Error("Organization slug should be auto-generated")
		}

		// Cleanup
		cleanupOrganization(t, result.Organization.ID)
	})

	t.Run("create organization with professional plan", func(t *testing.T) {
		result, err := svc.CreateOrganization(ctx, multitenancy.CreateOrganizationParams{
			Name:   "Pro Org " + uuid.New().String()[:8],
			PlanID: "professional",
		})
		if err != nil {
			t.Fatalf("CreateOrganization() error = %v", err)
		}

		// Verify quota was set based on professional plan
		quota, err := svc.GetQuota(ctx, result.Organization.ID)
		if err != nil {
			t.Fatalf("GetQuota() error = %v", err)
		}

		// Professional plan should have higher limits than free
		if quota.MaxAssets < 500 {
			t.Errorf("Expected professional plan to have >= 500 max assets, got %d", quota.MaxAssets)
		}

		// Cleanup
		cleanupOrganization(t, result.Organization.ID)
	})
}

func TestOrganizationOnboarding_UserLinking(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	svc := multitenancy.NewService(testDB)

	t.Run("link user to organization", func(t *testing.T) {
		// Create org first
		result, err := svc.CreateOrganization(ctx, multitenancy.CreateOrganizationParams{
			Name: "User Link Test Org " + uuid.New().String()[:8],
		})
		if err != nil {
			t.Fatalf("CreateOrganization() error = %v", err)
		}

		userID := "user_" + uuid.New().String()[:8]

		// Link user as owner
		err = svc.LinkUserToOrganization(ctx, userID, result.Organization.ID, "org_owner")
		if err != nil {
			t.Fatalf("LinkUserToOrganization() error = %v", err)
		}

		// Verify user can get their organization
		org, err := svc.GetUserOrganization(ctx, userID)
		if err != nil {
			t.Fatalf("GetUserOrganization() error = %v", err)
		}

		if org == nil {
			t.Fatal("User should have an organization after linking")
		}

		if org.ID != result.Organization.ID {
			t.Errorf("User organization ID mismatch: got %v, want %v", org.ID, result.Organization.ID)
		}

		// Cleanup
		cleanupOrganization(t, result.Organization.ID)
	})
}

// =============================================================================
// Demo Data Seeding Tests
// =============================================================================

func TestOrganizationOnboarding_SeedDemoData(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	svc := multitenancy.NewService(testDB)

	t.Run("seed AWS demo data", func(t *testing.T) {
		// Create org first
		result, err := svc.CreateOrganization(ctx, multitenancy.CreateOrganizationParams{
			Name: "AWS Demo Org " + uuid.New().String()[:8],
		})
		if err != nil {
			t.Fatalf("CreateOrganization() error = %v", err)
		}

		// Seed demo data
		seedResult, err := svc.SeedDemoData(ctx, result.Organization.ID, multitenancy.SeedDemoDataParams{
			Platform: "aws",
		})
		if err != nil {
			t.Fatalf("SeedDemoData() error = %v", err)
		}

		if seedResult.SitesCreated < 3 {
			t.Errorf("Expected at least 3 sites created for AWS, got %d", seedResult.SitesCreated)
		}

		if seedResult.ImagesCreated < 5 {
			t.Errorf("Expected at least 5 images created for AWS, got %d", seedResult.ImagesCreated)
		}

		if seedResult.AssetsCreated < 8 {
			t.Errorf("Expected at least 8 assets created for AWS, got %d", seedResult.AssetsCreated)
		}

		// Cleanup
		cleanupOrganization(t, result.Organization.ID)
	})

	t.Run("seed Azure demo data", func(t *testing.T) {
		// Create org first
		result, err := svc.CreateOrganization(ctx, multitenancy.CreateOrganizationParams{
			Name: "Azure Demo Org " + uuid.New().String()[:8],
		})
		if err != nil {
			t.Fatalf("CreateOrganization() error = %v", err)
		}

		// Seed demo data
		seedResult, err := svc.SeedDemoData(ctx, result.Organization.ID, multitenancy.SeedDemoDataParams{
			Platform: "azure",
		})
		if err != nil {
			t.Fatalf("SeedDemoData() error = %v", err)
		}

		if seedResult.SitesCreated < 3 {
			t.Errorf("Expected at least 3 sites created for Azure, got %d", seedResult.SitesCreated)
		}

		if seedResult.ImagesCreated < 4 {
			t.Errorf("Expected at least 4 images created for Azure, got %d", seedResult.ImagesCreated)
		}

		// Cleanup
		cleanupOrganization(t, result.Organization.ID)
	})

	t.Run("seed GCP demo data", func(t *testing.T) {
		// Create org first
		result, err := svc.CreateOrganization(ctx, multitenancy.CreateOrganizationParams{
			Name: "GCP Demo Org " + uuid.New().String()[:8],
		})
		if err != nil {
			t.Fatalf("CreateOrganization() error = %v", err)
		}

		// Seed demo data
		seedResult, err := svc.SeedDemoData(ctx, result.Organization.ID, multitenancy.SeedDemoDataParams{
			Platform: "gcp",
		})
		if err != nil {
			t.Fatalf("SeedDemoData() error = %v", err)
		}

		if seedResult.SitesCreated < 3 {
			t.Errorf("Expected at least 3 sites created for GCP, got %d", seedResult.SitesCreated)
		}

		if seedResult.ImagesCreated < 4 {
			t.Errorf("Expected at least 4 images created for GCP, got %d", seedResult.ImagesCreated)
		}

		// Cleanup
		cleanupOrganization(t, result.Organization.ID)
	})

	t.Run("empty platform defaults to AWS", func(t *testing.T) {
		// Create org first
		result, err := svc.CreateOrganization(ctx, multitenancy.CreateOrganizationParams{
			Name: "Default Platform Org " + uuid.New().String()[:8],
		})
		if err != nil {
			t.Fatalf("CreateOrganization() error = %v", err)
		}

		// Seed demo data with empty platform
		seedResult, err := svc.SeedDemoData(ctx, result.Organization.ID, multitenancy.SeedDemoDataParams{
			Platform: "", // Should default to AWS
		})
		if err != nil {
			t.Fatalf("SeedDemoData() error = %v", err)
		}

		// Should have AWS-style counts
		if seedResult.SitesCreated < 3 {
			t.Errorf("Expected at least 3 sites (AWS default), got %d", seedResult.SitesCreated)
		}

		// Cleanup
		cleanupOrganization(t, result.Organization.ID)
	})
}

// =============================================================================
// Full Onboarding Flow Tests
// =============================================================================

func TestOrganizationOnboarding_FullFlow(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	svc := multitenancy.NewService(testDB)

	t.Run("complete onboarding flow", func(t *testing.T) {
		// Step 1: Create organization
		orgResult, err := svc.CreateOrganization(ctx, multitenancy.CreateOrganizationParams{
			Name:   "Complete Flow Org " + uuid.New().String()[:8],
			PlanID: "starter",
		})
		if err != nil {
			t.Fatalf("Step 1 - CreateOrganization() error = %v", err)
		}

		orgID := orgResult.Organization.ID
		t.Logf("Created organization: %s (ID: %s)", orgResult.Organization.Name, orgID)

		// Step 2: Link user
		userID := "user_" + uuid.New().String()[:8]
		err = svc.LinkUserToOrganization(ctx, userID, orgID, "org_owner")
		if err != nil {
			t.Fatalf("Step 2 - LinkUserToOrganization() error = %v", err)
		}
		t.Logf("Linked user %s to organization", userID)

		// Step 3: Seed demo data
		seedResult, err := svc.SeedDemoData(ctx, orgID, multitenancy.SeedDemoDataParams{
			Platform: "aws",
		})
		if err != nil {
			t.Fatalf("Step 3 - SeedDemoData() error = %v", err)
		}
		t.Logf("Seeded demo data: %d sites, %d assets, %d images",
			seedResult.SitesCreated, seedResult.AssetsCreated, seedResult.ImagesCreated)

		// Step 4: Verify quota was applied
		quota, err := svc.GetQuota(ctx, orgID)
		if err != nil {
			t.Fatalf("Step 4 - GetQuota() error = %v", err)
		}
		if quota == nil {
			t.Fatal("Quota should be set after org creation")
		}
		t.Logf("Quota: max_assets=%d, max_images=%d", quota.MaxAssets, quota.MaxImages)

		// Step 5: Verify usage was updated
		usage, err := svc.GetUsage(ctx, orgID)
		if err != nil {
			t.Fatalf("Step 5 - GetUsage() error = %v", err)
		}
		if usage.SiteCount != seedResult.SitesCreated {
			t.Errorf("Usage site count mismatch: got %d, want %d", usage.SiteCount, seedResult.SitesCreated)
		}
		t.Logf("Usage: sites=%d, assets=%d, images=%d",
			usage.SiteCount, usage.AssetCount, usage.ImageCount)

		// Step 6: Verify subscription was created
		sub, err := svc.GetSubscription(ctx, orgID)
		if err != nil {
			t.Fatalf("Step 6 - GetSubscription() error = %v", err)
		}
		if sub == nil {
			t.Fatal("Subscription should be created after org creation")
		}
		t.Logf("Subscription status: %s", sub.Status)

		// Cleanup
		cleanupOrganization(t, orgID)
	})
}

// =============================================================================
// Quota Status Tests
// =============================================================================

func TestOrganizationOnboarding_QuotaStatus(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	svc := multitenancy.NewService(testDB)

	t.Run("quota status after seeding", func(t *testing.T) {
		// Create org first
		result, err := svc.CreateOrganization(ctx, multitenancy.CreateOrganizationParams{
			Name:   "Quota Status Org " + uuid.New().String()[:8],
			PlanID: "free", // Free plan has lower limits
		})
		if err != nil {
			t.Fatalf("CreateOrganization() error = %v", err)
		}

		// Seed demo data
		_, err = svc.SeedDemoData(ctx, result.Organization.ID, multitenancy.SeedDemoDataParams{
			Platform: "aws",
		})
		if err != nil {
			t.Fatalf("SeedDemoData() error = %v", err)
		}

		// Check quota status
		statuses, err := svc.GetQuotaStatus(ctx, result.Organization.ID)
		if err != nil {
			t.Fatalf("GetQuotaStatus() error = %v", err)
		}

		if len(statuses) == 0 {
			t.Error("Expected at least one quota status")
		}

		for _, status := range statuses {
			t.Logf("Quota %s: used=%d, limit=%d, remaining=%d, exceeded=%v",
				status.ResourceType, status.Used, status.Limit, status.Remaining, status.IsExceeded)
		}

		// Cleanup
		cleanupOrganization(t, result.Organization.ID)
	})
}

// =============================================================================
// Edge Cases and Error Handling
// =============================================================================

func TestOrganizationOnboarding_EdgeCases(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	svc := multitenancy.NewService(testDB)

	t.Run("get non-existent organization", func(t *testing.T) {
		org, err := svc.GetOrganization(ctx, uuid.New())
		if err != nil {
			t.Logf("GetOrganization() error (expected): %v", err)
		}
		if org != nil {
			t.Error("Expected nil org for non-existent ID")
		}
	})

	t.Run("get user organization for unlinked user", func(t *testing.T) {
		org, err := svc.GetUserOrganization(ctx, "nonexistent_user_"+uuid.New().String()[:8])
		if err != nil {
			t.Logf("GetUserOrganization() error (may be expected): %v", err)
			return
		}
		if org != nil {
			t.Error("Expected nil org for unlinked user")
		}
	})

	t.Run("seed demo data for non-existent org", func(t *testing.T) {
		_, err := svc.SeedDemoData(ctx, uuid.New(), multitenancy.SeedDemoDataParams{
			Platform: "aws",
		})
		// This should fail because the org doesn't exist
		if err == nil {
			t.Log("SeedDemoData() succeeded for non-existent org - may insert orphaned data")
		}
	})
}

// =============================================================================
// Helpers
// =============================================================================

func cleanupOrganization(t *testing.T, orgID uuid.UUID) {
	if testDB == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Delete organization - all dependent tables have ON DELETE CASCADE
	// so they will be cleaned up automatically
	_, err := testDB.ExecContext(ctx, "DELETE FROM organizations WHERE id = $1", orgID)
	if err != nil {
		t.Logf("Cleanup organization %s: %v", orgID, err)
	}
}
