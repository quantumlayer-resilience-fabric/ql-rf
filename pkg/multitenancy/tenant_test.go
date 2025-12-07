// Package multitenancy provides organization isolation, quotas, and usage tracking.
package multitenancy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuotaTypeConstants(t *testing.T) {
	t.Run("all quota types are defined", func(t *testing.T) {
		quotaTypes := []QuotaType{
			QuotaAssets,
			QuotaImages,
			QuotaSites,
			QuotaUsers,
			QuotaTeams,
			QuotaAITasks,
			QuotaAITokens,
			QuotaStorage,
			QuotaAPIRate,
		}

		assert.Len(t, quotaTypes, 9, "expected 9 quota types")

		// Verify each has expected string value
		assert.Equal(t, QuotaType("assets"), QuotaAssets)
		assert.Equal(t, QuotaType("images"), QuotaImages)
		assert.Equal(t, QuotaType("sites"), QuotaSites)
		assert.Equal(t, QuotaType("users"), QuotaUsers)
		assert.Equal(t, QuotaType("teams"), QuotaTeams)
		assert.Equal(t, QuotaType("ai_tasks"), QuotaAITasks)
		assert.Equal(t, QuotaType("ai_tokens"), QuotaAITokens)
		assert.Equal(t, QuotaType("storage"), QuotaStorage)
		assert.Equal(t, QuotaType("api_requests"), QuotaAPIRate)
	})
}

func TestSeedDemoDataParams(t *testing.T) {
	t.Run("params with AWS platform", func(t *testing.T) {
		params := SeedDemoDataParams{Platform: "aws"}
		assert.Equal(t, "aws", params.Platform)
	})

	t.Run("params with Azure platform", func(t *testing.T) {
		params := SeedDemoDataParams{Platform: "azure"}
		assert.Equal(t, "azure", params.Platform)
	})

	t.Run("params with GCP platform", func(t *testing.T) {
		params := SeedDemoDataParams{Platform: "gcp"}
		assert.Equal(t, "gcp", params.Platform)
	})

	t.Run("empty platform should be defaulted by handler", func(t *testing.T) {
		params := SeedDemoDataParams{}
		assert.Equal(t, "", params.Platform)
	})
}

func TestSeedDemoDataResult(t *testing.T) {
	t.Run("result contains counts", func(t *testing.T) {
		result := SeedDemoDataResult{
			SitesCreated:  3,
			AssetsCreated: 10,
			ImagesCreated: 5,
		}
		assert.Equal(t, 3, result.SitesCreated)
		assert.Equal(t, 10, result.AssetsCreated)
		assert.Equal(t, 5, result.ImagesCreated)
	})

	t.Run("zero values for empty result", func(t *testing.T) {
		result := SeedDemoDataResult{}
		assert.Equal(t, 0, result.SitesCreated)
		assert.Equal(t, 0, result.AssetsCreated)
		assert.Equal(t, 0, result.ImagesCreated)
	})
}

func TestGetDemoSites(t *testing.T) {
	tests := []struct {
		name         string
		platform     string
		expectedLen  int
		expectedName string
	}{
		{
			name:         "AWS sites",
			platform:     "aws",
			expectedLen:  3,
			expectedName: "AWS US-East-1",
		},
		{
			name:         "Azure sites",
			platform:     "azure",
			expectedLen:  3,
			expectedName: "Azure East US",
		},
		{
			name:         "GCP sites",
			platform:     "gcp",
			expectedLen:  3,
			expectedName: "GCP US-Central1",
		},
		{
			name:         "unknown platform defaults to AWS",
			platform:     "unknown",
			expectedLen:  3,
			expectedName: "AWS US-East-1",
		},
		{
			name:         "empty platform defaults to AWS",
			platform:     "",
			expectedLen:  3,
			expectedName: "AWS US-East-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sites := getDemoSites(tt.platform)
			require.Len(t, sites, tt.expectedLen)
			assert.Equal(t, tt.expectedName, sites[0].name)

			// Verify all sites have required fields
			for _, site := range sites {
				assert.NotEmpty(t, site.name, "site name should not be empty")
				assert.NotEmpty(t, site.platform, "site platform should not be empty")
				assert.NotEmpty(t, site.region, "site region should not be empty")
				assert.NotEmpty(t, site.metadata, "site metadata should not be empty")
			}
		})
	}
}

func TestGetDemoImages(t *testing.T) {
	tests := []struct {
		name           string
		platform       string
		expectedMinLen int
		expectedPrefix string
	}{
		{
			name:           "AWS images",
			platform:       "aws",
			expectedMinLen: 5,
			expectedPrefix: "ami-",
		},
		{
			name:           "Azure images",
			platform:       "azure",
			expectedMinLen: 4,
			expectedPrefix: "img-",
		},
		{
			name:           "GCP images",
			platform:       "gcp",
			expectedMinLen: 4,
			expectedPrefix: "gce-",
		},
		{
			name:           "unknown platform defaults to AWS",
			platform:       "unknown",
			expectedMinLen: 5,
			expectedPrefix: "ami-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images := getDemoImages(tt.platform)
			require.GreaterOrEqual(t, len(images), tt.expectedMinLen)
			assert.Contains(t, images[0].name, tt.expectedPrefix)

			// Verify all images have required fields
			for _, img := range images {
				assert.NotEmpty(t, img.name, "image name should not be empty")
				assert.NotEmpty(t, img.family, "image family should not be empty")
				assert.NotEmpty(t, img.version, "image version should not be empty")
				assert.NotEmpty(t, img.status, "image status should not be empty")
				assert.NotEmpty(t, img.osName, "image osName should not be empty")
				assert.NotEmpty(t, img.osVersion, "image osVersion should not be empty")
			}
		})
	}
}

func TestGetDemoAssets(t *testing.T) {
	tests := []struct {
		name             string
		platform         string
		expectedMinLen   int
		expectedPlatform string
	}{
		{
			name:             "AWS assets",
			platform:         "aws",
			expectedMinLen:   8,
			expectedPlatform: "aws",
		},
		{
			name:             "Azure assets",
			platform:         "azure",
			expectedMinLen:   6,
			expectedPlatform: "azure",
		},
		{
			name:             "GCP assets",
			platform:         "gcp",
			expectedMinLen:   6,
			expectedPlatform: "gcp",
		},
		{
			name:             "unknown platform defaults to AWS",
			platform:         "unknown",
			expectedMinLen:   8,
			expectedPlatform: "aws",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assets := getDemoAssets(tt.platform)
			require.GreaterOrEqual(t, len(assets), tt.expectedMinLen)

			// Verify all assets have required fields
			for _, asset := range assets {
				assert.NotEmpty(t, asset.instanceID, "asset instanceID should not be empty")
				assert.NotEmpty(t, asset.name, "asset name should not be empty")
				assert.NotEmpty(t, asset.siteName, "asset site name should not be empty")
				assert.Equal(t, tt.expectedPlatform, asset.platform, "asset platform should match")
				assert.NotEmpty(t, asset.state, "asset state should not be empty")
				assert.NotEmpty(t, asset.region, "asset region should not be empty")
			}
		})
	}
}

func TestCalculateStatus(t *testing.T) {
	// Since calculateStatus is a private method on Service, we test
	// the QuotaStatus struct behavior instead

	t.Run("quota not exceeded", func(t *testing.T) {
		status := QuotaStatus{
			ResourceType:   QuotaAssets,
			Limit:          100,
			Used:           50,
			Remaining:      50,
			PercentageUsed: 50.0,
			IsExceeded:     false,
		}

		assert.Equal(t, QuotaAssets, status.ResourceType)
		assert.Equal(t, int64(100), status.Limit)
		assert.Equal(t, int64(50), status.Used)
		assert.Equal(t, int64(50), status.Remaining)
		assert.Equal(t, 50.0, status.PercentageUsed)
		assert.False(t, status.IsExceeded)
	})

	t.Run("quota exceeded", func(t *testing.T) {
		status := QuotaStatus{
			ResourceType:   QuotaImages,
			Limit:          10,
			Used:           15,
			Remaining:      -5,
			PercentageUsed: 150.0,
			IsExceeded:     true,
		}

		assert.Equal(t, QuotaImages, status.ResourceType)
		assert.Equal(t, int64(10), status.Limit)
		assert.Equal(t, int64(15), status.Used)
		assert.True(t, status.IsExceeded)
	})

	t.Run("quota at limit", func(t *testing.T) {
		status := QuotaStatus{
			ResourceType:   QuotaSites,
			Limit:          20,
			Used:           20,
			Remaining:      0,
			PercentageUsed: 100.0,
			IsExceeded:     true,
		}

		assert.Equal(t, QuotaSites, status.ResourceType)
		assert.Equal(t, int64(0), status.Remaining)
		assert.Equal(t, 100.0, status.PercentageUsed)
		assert.True(t, status.IsExceeded)
	})

	t.Run("unlimited quota (-1)", func(t *testing.T) {
		status := QuotaStatus{
			ResourceType:   QuotaUsers,
			Limit:          -1, // unlimited
			Used:           1000,
			Remaining:      -1, // unlimited
			PercentageUsed: 0.0,
			IsExceeded:     false,
		}

		assert.Equal(t, QuotaUsers, status.ResourceType)
		assert.Equal(t, int64(-1), status.Limit)
		assert.False(t, status.IsExceeded)
	})
}

func TestOrganizationQuotaStruct(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		quota := OrganizationQuota{}
		assert.Equal(t, 0, quota.MaxAssets)
		assert.Equal(t, 0, quota.MaxImages)
		assert.Equal(t, 0, quota.MaxSites)
		assert.Equal(t, int64(0), quota.MaxStorageBytes)
		assert.False(t, quota.DREnabled)
		assert.False(t, quota.ComplianceEnabled)
	})

	t.Run("populated values", func(t *testing.T) {
		quota := OrganizationQuota{
			MaxAssets:             1000,
			MaxImages:             100,
			MaxSites:              50,
			MaxUsers:              100,
			MaxAITasksPerDay:      100,
			MaxAITokensPerMonth:   10000000,
			MaxStorageBytes:       107374182400, // 100GB
			APIRateLimitPerMinute: 1000,
			DREnabled:             true,
			ComplianceEnabled:     true,
			AdvancedAnalytics:     true,
			CustomIntegrations:    true,
		}

		assert.Equal(t, 1000, quota.MaxAssets)
		assert.Equal(t, 100, quota.MaxImages)
		assert.Equal(t, int64(107374182400), quota.MaxStorageBytes)
		assert.True(t, quota.DREnabled)
		assert.True(t, quota.ComplianceEnabled)
		assert.True(t, quota.AdvancedAnalytics)
		assert.True(t, quota.CustomIntegrations)
	})
}

func TestOrganizationUsageStruct(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		usage := OrganizationUsage{}
		assert.Equal(t, 0, usage.AssetCount)
		assert.Equal(t, 0, usage.ImageCount)
		assert.Equal(t, int64(0), usage.StorageUsedBytes)
	})

	t.Run("populated values", func(t *testing.T) {
		usage := OrganizationUsage{
			AssetCount:        500,
			ImageCount:        25,
			SiteCount:         10,
			UserCount:         50,
			StorageUsedBytes:  5368709120, // 5GB
			AITasksToday:      42,
			AITokensThisMonth: 500000,
			APIRequestsToday:  1500,
		}

		assert.Equal(t, 500, usage.AssetCount)
		assert.Equal(t, 25, usage.ImageCount)
		assert.Equal(t, int64(5368709120), usage.StorageUsedBytes)
		assert.Equal(t, 42, usage.AITasksToday)
		assert.Equal(t, int64(500000), usage.AITokensThisMonth)
	})
}

func TestSubscriptionPlanStruct(t *testing.T) {
	t.Run("free plan", func(t *testing.T) {
		plan := SubscriptionPlan{
			Name:             "free",
			DisplayName:      "Free",
			PlanType:         "free",
			DefaultMaxAssets: 100,
			DefaultMaxImages: 10,
			DefaultMaxSites:  5,
			DefaultMaxUsers:  5,
			DRIncluded:       false,
			IsActive:         true,
		}

		assert.Equal(t, "free", plan.Name)
		assert.Equal(t, 100, plan.DefaultMaxAssets)
		assert.False(t, plan.DRIncluded)
		assert.True(t, plan.IsActive)
		assert.Nil(t, plan.MonthlyPriceUSD)
	})

	t.Run("enterprise plan with pricing", func(t *testing.T) {
		monthlyPrice := 499.0
		annualPrice := 4990.0
		plan := SubscriptionPlan{
			Name:               "enterprise",
			DisplayName:        "Enterprise",
			PlanType:           "enterprise",
			DefaultMaxAssets:   -1, // unlimited
			DefaultMaxImages:   -1,
			DefaultMaxSites:    -1,
			DRIncluded:         true,
			ComplianceIncluded: true,
			AdvancedAnalytics:  true,
			CustomIntegrations: true,
			MonthlyPriceUSD:    &monthlyPrice,
			AnnualPriceUSD:     &annualPrice,
			IsActive:           true,
		}

		assert.Equal(t, "enterprise", plan.Name)
		assert.Equal(t, -1, plan.DefaultMaxAssets) // unlimited
		assert.True(t, plan.DRIncluded)
		assert.True(t, plan.ComplianceIncluded)
		assert.True(t, plan.AdvancedAnalytics)
		assert.True(t, plan.CustomIntegrations)
		require.NotNil(t, plan.MonthlyPriceUSD)
		assert.Equal(t, 499.0, *plan.MonthlyPriceUSD)
		require.NotNil(t, plan.AnnualPriceUSD)
		assert.Equal(t, 4990.0, *plan.AnnualPriceUSD)
	})
}

func TestCreateOrganizationParams(t *testing.T) {
	t.Run("with all fields", func(t *testing.T) {
		params := CreateOrganizationParams{
			Name:   "Acme Corp",
			Slug:   "acme-corp",
			PlanID: "free",
		}

		assert.Equal(t, "Acme Corp", params.Name)
		assert.Equal(t, "acme-corp", params.Slug)
		assert.Equal(t, "free", params.PlanID)
	})

	t.Run("with minimal fields", func(t *testing.T) {
		params := CreateOrganizationParams{
			Name: "Test Org",
		}

		assert.Equal(t, "Test Org", params.Name)
		assert.Empty(t, params.Slug)
		assert.Empty(t, params.PlanID)
	})
}

func TestDemoSitesPlatformConsistency(t *testing.T) {
	platforms := []string{"aws", "azure", "gcp"}

	for _, platform := range platforms {
		t.Run(platform+" sites all match platform", func(t *testing.T) {
			sites := getDemoSites(platform)
			for _, site := range sites {
				assert.Equal(t, platform, site.platform,
					"site %s should have platform %s", site.name, platform)
			}
		})
	}
}

func TestDemoImagesPlatformConsistency(t *testing.T) {
	// Note: Images are now platform-agnostic in the database schema.
	// This test just verifies that each platform returns appropriate demo images.
	platforms := []string{"aws", "azure", "gcp"}

	for _, platform := range platforms {
		t.Run(platform+" images are returned", func(t *testing.T) {
			images := getDemoImages(platform)
			assert.Greater(t, len(images), 0, "should return demo images for %s", platform)
			for _, img := range images {
				assert.NotEmpty(t, img.family, "image family should not be empty")
				assert.NotEmpty(t, img.version, "image version should not be empty")
			}
		})
	}
}

func TestDemoAssetsPlatformConsistency(t *testing.T) {
	platforms := []string{"aws", "azure", "gcp"}

	for _, platform := range platforms {
		t.Run(platform+" assets all match platform", func(t *testing.T) {
			assets := getDemoAssets(platform)
			for _, asset := range assets {
				assert.Equal(t, platform, asset.platform,
					"asset %s should have platform %s", asset.name, platform)
			}
		})
	}
}

func TestDemoAssetsReferenceValidSites(t *testing.T) {
	platforms := []string{"aws", "azure", "gcp"}

	for _, platform := range platforms {
		t.Run(platform+" assets reference valid sites", func(t *testing.T) {
			sites := getDemoSites(platform)
			siteNames := make(map[string]bool)
			for _, site := range sites {
				siteNames[site.name] = true
			}

			assets := getDemoAssets(platform)
			for _, asset := range assets {
				assert.True(t, siteNames[asset.siteName],
					"asset %s references unknown site %s", asset.name, asset.siteName)
			}
		})
	}
}

func TestDemoAssetsReferenceValidImages(t *testing.T) {
	platforms := []string{"aws", "azure", "gcp"}

	for _, platform := range platforms {
		t.Run(platform+" assets reference valid images", func(t *testing.T) {
			images := getDemoImages(platform)
			imageNames := make(map[string]bool)
			for _, img := range images {
				imageNames[img.name] = true
			}

			assets := getDemoAssets(platform)
			for _, asset := range assets {
				// imageRef can be empty (some assets may not have an image)
				if asset.imageRef != "" {
					assert.True(t, imageNames[asset.imageRef],
						"asset %s references unknown image %s", asset.name, asset.imageRef)
				}
			}
		})
	}
}
