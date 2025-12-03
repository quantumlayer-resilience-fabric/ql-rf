package service

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/quantumlayerhq/ql-rf/pkg/database"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

// RiskService handles risk scoring business logic.
type RiskService struct {
	db      *database.DB
	log     *logger.Logger
	weights models.RiskScoreWeights
}

// NewRiskService creates a new RiskService.
func NewRiskService(db *database.DB, log *logger.Logger) *RiskService {
	return &RiskService{
		db:      db,
		log:     log.WithComponent("risk-service"),
		weights: models.DefaultRiskWeights(),
	}
}

// GetRiskSummaryInput contains input for getting risk summary.
type GetRiskSummaryInput struct {
	OrgID uuid.UUID
}

// GetRiskSummary calculates and returns the organization-wide risk summary.
func (s *RiskService) GetRiskSummary(ctx context.Context, input GetRiskSummaryInput) (*models.RiskSummary, error) {
	// Get asset risk scores
	assetRisks, err := s.calculateAssetRisks(ctx, input.OrgID)
	if err != nil {
		return nil, err
	}

	// Calculate overall metrics
	totalAssets := len(assetRisks)
	var criticalCount, highCount, mediumCount, lowCount int
	var totalScore float64

	for _, risk := range assetRisks {
		totalScore += risk.RiskScore
		switch risk.RiskLevel {
		case models.RiskLevelCritical:
			criticalCount++
		case models.RiskLevelHigh:
			highCount++
		case models.RiskLevelMedium:
			mediumCount++
		case models.RiskLevelLow:
			lowCount++
		}
	}

	overallScore := 0.0
	if totalAssets > 0 {
		overallScore = totalScore / float64(totalAssets)
	}

	// Sort by risk score descending for top risks
	sort.Slice(assetRisks, func(i, j int) bool {
		return assetRisks[i].RiskScore > assetRisks[j].RiskScore
	})

	// Get top 10 risks
	topRisks := assetRisks
	if len(topRisks) > 10 {
		topRisks = topRisks[:10]
	}

	// Calculate by scope
	byEnvironment := s.aggregateByScope(assetRisks, "environment")
	byPlatform := s.aggregateByScope(assetRisks, "platform")
	bySite := s.aggregateByScope(assetRisks, "site")

	// Generate trend (last 30 days simulated)
	trend := s.generateTrend(overallScore, 30)

	return &models.RiskSummary{
		OrgID:            input.OrgID,
		OverallRiskScore: overallScore,
		RiskLevel:        models.CalculateRiskLevel(overallScore),
		TotalAssets:      totalAssets,
		CriticalRisk:     criticalCount,
		HighRisk:         highCount,
		MediumRisk:       mediumCount,
		LowRisk:          lowCount,
		TopRisks:         topRisks,
		ByEnvironment:    byEnvironment,
		ByPlatform:       byPlatform,
		BySite:           bySite,
		Trend:            trend,
		CalculatedAt:     time.Now(),
	}, nil
}

// GetTopRisksInput contains input for getting top risk assets.
type GetTopRisksInput struct {
	OrgID uuid.UUID
	Limit int
}

// GetTopRisks returns the top N highest risk assets.
func (s *RiskService) GetTopRisks(ctx context.Context, input GetTopRisksInput) ([]models.AssetRiskScore, error) {
	if input.Limit <= 0 {
		input.Limit = 10
	}

	assetRisks, err := s.calculateAssetRisks(ctx, input.OrgID)
	if err != nil {
		return nil, err
	}

	// Sort by risk score descending
	sort.Slice(assetRisks, func(i, j int) bool {
		return assetRisks[i].RiskScore > assetRisks[j].RiskScore
	})

	if len(assetRisks) > input.Limit {
		assetRisks = assetRisks[:input.Limit]
	}

	return assetRisks, nil
}

// calculateAssetRisks calculates risk scores for all assets.
func (s *RiskService) calculateAssetRisks(ctx context.Context, orgID uuid.UUID) ([]models.AssetRiskScore, error) {
	// Query assets with their risk factors
	query := `
		SELECT
			a.id,
			COALESCE(a.name, a.instance_id) as name,
			a.platform,
			COALESCE(e.name, 'unknown') as environment,
			COALESCE(a.site, 'unknown') as site,
			a.image_version,
			COALESCE(gi.version, '') as golden_version,
			COALESCE(a.discovered_at, NOW()) as discovered_at,
			COALESCE(vuln.critical_count, 0) as critical_vulns,
			COALESCE(vuln.total_count, 0) as vuln_count
		FROM assets a
		LEFT JOIN environments e ON e.id = a.env_id
		LEFT JOIN images gi ON gi.family = a.image_ref AND gi.org_id = a.org_id AND gi.status = 'production'
		LEFT JOIN LATERAL (
			SELECT
				COUNT(*) FILTER (WHERE severity = 'critical') as critical_count,
				COUNT(*) as total_count
			FROM image_vulnerabilities iv
			JOIN images img ON img.id = iv.image_id
			WHERE img.family = a.image_ref
			AND img.org_id = a.org_id
			AND iv.status = 'open'
		) vuln ON true
		WHERE a.org_id = $1
		AND a.state IN ('running', 'stopped')
		ORDER BY a.discovered_at DESC
	`

	rows, err := s.db.Pool.Query(ctx, query, orgID)
	if err != nil {
		s.log.Error("failed to query asset risks", "error", err)
		return nil, err
	}
	defer rows.Close()

	var results []models.AssetRiskScore
	now := time.Now()

	for rows.Next() {
		var (
			assetID        uuid.UUID
			name           string
			platform       string
			environment    string
			site           string
			imageVersion   *string
			goldenVersion  string
			discoveredAt   time.Time
			criticalVulns  int
			vulnCount      int
		)

		if err := rows.Scan(
			&assetID, &name, &platform, &environment, &site,
			&imageVersion, &goldenVersion, &discoveredAt,
			&criticalVulns, &vulnCount,
		); err != nil {
			s.log.Warn("failed to scan asset risk row", "error", err)
			continue
		}

		// Calculate drift age
		isDrifted := false
		driftAge := 0
		if imageVersion != nil && goldenVersion != "" && *imageVersion != goldenVersion {
			isDrifted = true
			driftAge = int(now.Sub(discoveredAt).Hours() / 24) // Days since discovered
		}

		// Calculate risk score
		riskScore, factors := s.calculateRiskScore(
			driftAge,
			vulnCount,
			criticalVulns,
			!isDrifted, // isCompliant
			environment,
		)

		results = append(results, models.AssetRiskScore{
			AssetID:       assetID,
			AssetName:     name,
			Platform:      platform,
			Environment:   environment,
			Site:          site,
			RiskScore:     riskScore,
			RiskLevel:     models.CalculateRiskLevel(riskScore),
			Factors:       factors,
			DriftAge:      driftAge,
			VulnCount:     vulnCount,
			CriticalVulns: criticalVulns,
			IsCompliant:   !isDrifted,
			LastUpdated:   now,
		})
	}

	return results, nil
}

// calculateRiskScore calculates the risk score for an asset.
func (s *RiskService) calculateRiskScore(
	driftAge int,
	vulnCount int,
	criticalVulns int,
	isCompliant bool,
	environment string,
) (float64, []models.RiskFactor) {
	var factors []models.RiskFactor
	var totalScore float64

	// Drift age factor (0-100 based on days)
	driftScore := 0.0
	if driftAge > 0 {
		driftScore = min(float64(driftAge)*2, 100) // 2 points per day, max 100
		factors = append(factors, models.RiskFactor{
			Name:        "Drift Age",
			Description: "Asset has been drifted for extended period",
			Weight:      s.weights.DriftAge,
			Score:       driftScore,
			Impact:      "negative",
		})
	}
	totalScore += driftScore * s.weights.DriftAge

	// Vulnerability count factor
	vulnScore := min(float64(vulnCount)*5, 100) // 5 points per vuln, max 100
	if vulnCount > 0 {
		factors = append(factors, models.RiskFactor{
			Name:        "Open Vulnerabilities",
			Description: "Asset has unpatched vulnerabilities",
			Weight:      s.weights.VulnCount,
			Score:       vulnScore,
			Impact:      "negative",
		})
	}
	totalScore += vulnScore * s.weights.VulnCount

	// Critical vulnerabilities factor (heavily weighted)
	criticalScore := min(float64(criticalVulns)*25, 100) // 25 points per critical, max 100
	if criticalVulns > 0 {
		factors = append(factors, models.RiskFactor{
			Name:        "Critical Vulnerabilities",
			Description: "Asset has critical severity vulnerabilities",
			Weight:      s.weights.CriticalVulns,
			Score:       criticalScore,
			Impact:      "negative",
		})
	}
	totalScore += criticalScore * s.weights.CriticalVulns

	// Compliance factor
	complianceScore := 0.0
	if !isCompliant {
		complianceScore = 100
		factors = append(factors, models.RiskFactor{
			Name:        "Non-Compliant",
			Description: "Asset is not using approved golden image",
			Weight:      s.weights.ComplianceStatus,
			Score:       complianceScore,
			Impact:      "negative",
		})
	}
	totalScore += complianceScore * s.weights.ComplianceStatus

	// Environment multiplier
	envMultiplier := models.EnvironmentRiskMultiplier(environment)
	if envMultiplier != 1.0 {
		factors = append(factors, models.RiskFactor{
			Name:        "Environment Impact",
			Description: "Risk adjusted for environment criticality",
			Weight:      s.weights.Environment,
			Score:       (envMultiplier - 1) * 100, // Show as percentage adjustment
			Impact:      "multiplier",
		})
	}

	// Apply environment multiplier
	finalScore := totalScore * envMultiplier

	// Ensure score is within bounds
	if finalScore > 100 {
		finalScore = 100
	}
	if finalScore < 0 {
		finalScore = 0
	}

	return finalScore, factors
}

// aggregateByScope aggregates risk scores by a given scope.
func (s *RiskService) aggregateByScope(risks []models.AssetRiskScore, scopeType string) []models.RiskByScope {
	scopeMap := make(map[string]*models.RiskByScope)

	for _, risk := range risks {
		var scopeValue string
		switch scopeType {
		case "environment":
			scopeValue = risk.Environment
		case "platform":
			scopeValue = risk.Platform
		case "site":
			scopeValue = risk.Site
		}

		if _, exists := scopeMap[scopeValue]; !exists {
			scopeMap[scopeValue] = &models.RiskByScope{
				Scope: scopeValue,
			}
		}

		scope := scopeMap[scopeValue]
		scope.AssetCount++
		scope.RiskScore += risk.RiskScore

		switch risk.RiskLevel {
		case models.RiskLevelCritical:
			scope.CriticalRisk++
		case models.RiskLevelHigh:
			scope.HighRisk++
		}
	}

	// Calculate averages and convert to slice
	var results []models.RiskByScope
	for _, scope := range scopeMap {
		if scope.AssetCount > 0 {
			scope.RiskScore = scope.RiskScore / float64(scope.AssetCount)
			scope.RiskLevel = models.CalculateRiskLevel(scope.RiskScore)
		}
		results = append(results, *scope)
	}

	// Sort by risk score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].RiskScore > results[j].RiskScore
	})

	return results
}

// generateTrend generates a simulated trend for the last N days.
func (s *RiskService) generateTrend(currentScore float64, days int) []models.RiskTrendPoint {
	trend := make([]models.RiskTrendPoint, days)
	now := time.Now()

	// Generate a slight upward trend to current
	baseScore := currentScore * 0.85 // Start at 85% of current
	increment := (currentScore - baseScore) / float64(days)

	for i := 0; i < days; i++ {
		dayScore := baseScore + (increment * float64(i))
		// Add some random variation
		variation := (float64(i%7) - 3) * 0.5 // +/- 1.5 variance
		dayScore += variation

		if dayScore < 0 {
			dayScore = 0
		}
		if dayScore > 100 {
			dayScore = 100
		}

		trend[i] = models.RiskTrendPoint{
			Date:      now.AddDate(0, 0, -(days - 1 - i)),
			RiskScore: dayScore,
			RiskLevel: models.CalculateRiskLevel(dayScore),
		}
	}

	return trend
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
