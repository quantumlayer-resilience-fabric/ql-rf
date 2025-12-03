package service

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/quantumlayerhq/ql-rf/pkg/database"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

// PredictionService handles risk prediction and anomaly detection.
type PredictionService struct {
	db  *database.DB
	log *logger.Logger
}

// NewPredictionService creates a new PredictionService.
func NewPredictionService(db *database.DB, log *logger.Logger) *PredictionService {
	return &PredictionService{
		db:  db,
		log: log.WithComponent("prediction-service"),
	}
}

// GetRiskForecastInput contains input for getting risk forecast.
type GetRiskForecastInput struct {
	OrgID uuid.UUID
}

// GetRiskForecast generates a comprehensive risk forecast for the organization.
func (s *PredictionService) GetRiskForecast(ctx context.Context, input GetRiskForecastInput) (*models.RiskForecast, error) {
	// Get historical risk data
	history, err := s.getRiskHistory(ctx, input.OrgID, 30)
	if err != nil {
		return nil, err
	}

	// Calculate current score and velocity
	currentScore := 0.0
	if len(history) > 0 {
		currentScore = history[len(history)-1].Score
	}

	velocity, velocityValue := s.calculateVelocity(history)

	// Generate predictions for 7, 14, 30 days
	predictions := s.generatePredictions(currentScore, velocityValue, history)

	// Detect anomalies
	anomalies := s.detectAnomalies(history)

	// Get at-risk and improving assets
	atRiskAssets, improvingAssets, err := s.getAssetTrends(ctx, input.OrgID)
	if err != nil {
		s.log.Warn("failed to get asset trends", "error", err)
	}

	// Generate recommendations
	recommendations := s.generateRecommendations(ctx, input.OrgID, currentScore, atRiskAssets)

	return &models.RiskForecast{
		OrgID:              input.OrgID,
		CurrentScore:       currentScore,
		Predictions:        predictions,
		Velocity:           velocity,
		VelocityValue:      velocityValue,
		Anomalies:          anomalies,
		AtRiskAssets:       atRiskAssets,
		ImprovingAssets:    improvingAssets,
		TopRecommendations: recommendations,
		GeneratedAt:        time.Now(),
	}, nil
}

// riskHistoryPoint represents a historical risk data point.
type riskHistoryPoint struct {
	Date  time.Time
	Score float64
}

// getRiskHistory retrieves historical risk scores from drift reports.
// Risk score is calculated as: 100 - coverage_pct (so 100% coverage = 0 risk, 0% coverage = 100 risk)
func (s *PredictionService) getRiskHistory(ctx context.Context, orgID uuid.UUID, days int) ([]riskHistoryPoint, error) {
	query := `
		SELECT
			DATE(calculated_at) as date,
			AVG(100 - coverage_pct) as score
		FROM drift_reports
		WHERE org_id = $1
		AND calculated_at >= NOW() - make_interval(days => $2)
		GROUP BY DATE(calculated_at)
		ORDER BY date ASC
	`

	if s.db == nil {
		s.log.Warn("database not available, returning empty history")
		return []riskHistoryPoint{}, nil
	}

	rows, err := s.db.Pool.Query(ctx, query, orgID, days)
	if err != nil {
		s.log.Error("failed to query risk history", "error", err)
		return []riskHistoryPoint{}, nil
	}
	defer rows.Close()

	var history []riskHistoryPoint
	for rows.Next() {
		var date time.Time
		var score float64
		if err := rows.Scan(&date, &score); err != nil {
			s.log.Error("failed to scan risk history row", "error", err)
			continue
		}
		history = append(history, riskHistoryPoint{
			Date:  date,
			Score: score,
		})
	}

	if err := rows.Err(); err != nil {
		s.log.Error("error iterating risk history rows", "error", err)
	}

	// If no history, return empty slice - UI will handle empty state
	if len(history) == 0 {
		s.log.Info("no risk history found for organization", "org_id", orgID, "days", days)
	}

	return history, nil
}

// calculateVelocity calculates the rate of risk change.
func (s *PredictionService) calculateVelocity(history []riskHistoryPoint) (models.RiskVelocity, float64) {
	if len(history) < 7 {
		return models.RiskVelocityStable, 0
	}

	// Calculate 7-day moving average change
	recentDays := 7
	if len(history) < recentDays {
		recentDays = len(history)
	}

	recentHistory := history[len(history)-recentDays:]
	startScore := recentHistory[0].Score
	endScore := recentHistory[len(recentHistory)-1].Score

	pointsPerDay := (endScore - startScore) / float64(recentDays)
	velocity := models.CalculateVelocity(pointsPerDay)

	return velocity, math.Round(pointsPerDay*100) / 100
}

// generatePredictions creates future risk predictions.
func (s *PredictionService) generatePredictions(currentScore, velocityValue float64, history []riskHistoryPoint) []models.RiskPrediction {
	horizons := []int{7, 14, 30}
	predictions := make([]models.RiskPrediction, len(horizons))
	now := time.Now()

	for i, horizon := range horizons {
		// Linear projection with dampening for longer horizons
		dampening := 1.0 - (float64(horizon) / 100.0) // Reduce confidence over time
		if dampening < 0.5 {
			dampening = 0.5
		}

		predictedScore := currentScore + (velocityValue * float64(horizon) * dampening)

		// Bound the prediction
		if predictedScore < 0 {
			predictedScore = 0
		}
		if predictedScore > 100 {
			predictedScore = 100
		}

		// Calculate confidence (decreases with horizon)
		confidence := 0.95 - (float64(horizon) * 0.015)
		if confidence < 0.5 {
			confidence = 0.5
		}

		// Add variance based on historical volatility
		volatility := s.calculateVolatility(history)
		confidence = confidence * (1 - volatility/100)

		// Determine factors
		factors := s.determinePredictionFactors(velocityValue, predictedScore)

		// Recommended action
		action := s.recommendAction(currentScore, predictedScore)

		predictions[i] = models.RiskPrediction{
			CurrentScore:      currentScore,
			PredictedScore:    math.Round(predictedScore*10) / 10,
			PredictedLevel:    models.CalculateRiskLevel(predictedScore),
			Confidence:        math.Round(confidence*100) / 100,
			PredictionHorizon: horizon,
			Velocity:          models.CalculateVelocity(velocityValue),
			VelocityValue:     velocityValue,
			Factors:           factors,
			RecommendedAction: action,
			PredictedAt:       now,
		}
	}

	return predictions
}

// calculateVolatility calculates the standard deviation of risk scores.
func (s *PredictionService) calculateVolatility(history []riskHistoryPoint) float64 {
	if len(history) < 2 {
		return 0
	}

	// Calculate mean
	sum := 0.0
	for _, h := range history {
		sum += h.Score
	}
	mean := sum / float64(len(history))

	// Calculate variance
	variance := 0.0
	for _, h := range history {
		diff := h.Score - mean
		variance += diff * diff
	}
	variance /= float64(len(history))

	return math.Sqrt(variance)
}

// determinePredictionFactors identifies what's driving the prediction.
func (s *PredictionService) determinePredictionFactors(velocity, predictedScore float64) []string {
	var factors []string

	if velocity > 5 {
		factors = append(factors, "Rapid risk increase detected")
	} else if velocity > 2 {
		factors = append(factors, "Risk trending upward")
	} else if velocity < -5 {
		factors = append(factors, "Significant risk improvement")
	} else if velocity < -2 {
		factors = append(factors, "Risk trending downward")
	}

	if predictedScore >= 80 {
		factors = append(factors, "Predicted to reach critical level")
	} else if predictedScore >= 60 {
		factors = append(factors, "Predicted to reach high risk level")
	}

	if len(factors) == 0 {
		factors = append(factors, "Risk levels stable")
	}

	return factors
}

// recommendAction suggests what to do based on prediction.
func (s *PredictionService) recommendAction(current, predicted float64) string {
	diff := predicted - current

	if predicted >= 80 {
		return "Immediate remediation required - schedule emergency maintenance"
	}
	if predicted >= 60 && current < 60 {
		return "Risk will escalate to high - prioritize remediation this week"
	}
	if diff > 10 {
		return "Risk increasing rapidly - investigate contributing factors"
	}
	if diff < -10 {
		return "Risk improving - continue current remediation efforts"
	}
	if predicted < 40 {
		return "Risk well controlled - maintain current posture"
	}
	return "Monitor risk trends and address high-priority items"
}

// detectAnomalies identifies unusual patterns in risk data.
func (s *PredictionService) detectAnomalies(history []riskHistoryPoint) []models.RiskAnomaly {
	if len(history) < 7 {
		return nil
	}

	var anomalies []models.RiskAnomaly

	// Calculate mean and standard deviation
	sum := 0.0
	for _, h := range history {
		sum += h.Score
	}
	mean := sum / float64(len(history))

	variance := 0.0
	for _, h := range history {
		diff := h.Score - mean
		variance += diff * diff
	}
	stdDev := math.Sqrt(variance / float64(len(history)))

	// Detect anomalies (more than 2 standard deviations from mean)
	for i, h := range history {
		deviation := math.Abs(h.Score-mean) / stdDev
		if deviation > 2 {
			anomalyType := "spike"
			if h.Score < mean {
				anomalyType = "drop"
			}

			severity := models.RiskLevelMedium
			if deviation > 3 {
				severity = models.RiskLevelHigh
			}
			if deviation > 4 {
				severity = models.RiskLevelCritical
			}

			anomalies = append(anomalies, models.RiskAnomaly{
				ID:            uuid.New(),
				AnomalyType:   anomalyType,
				Severity:      severity,
				Description:   s.describeAnomaly(anomalyType, h.Score, mean, deviation),
				ExpectedScore: mean,
				ActualScore:   h.Score,
				Deviation:     math.Round(deviation*100) / 100,
				DetectedAt:    h.Date,
				IsActive:      i == len(history)-1, // Only the latest is active
			})
		}
	}

	return anomalies
}

// describeAnomaly generates a human-readable anomaly description.
func (s *PredictionService) describeAnomaly(anomalyType string, actual, expected, deviation float64) string {
	direction := "above"
	if actual < expected {
		direction = "below"
	}

	return "Risk score " + direction + " expected range by " +
		string(rune('0'+int(deviation))) + " standard deviations"
}

// getAssetTrends identifies assets with increasing/decreasing risk based on drift status.
func (s *PredictionService) getAssetTrends(ctx context.Context, orgID uuid.UUID) ([]models.AssetRiskScore, []models.AssetRiskScore, error) {
	if s.db == nil {
		return []models.AssetRiskScore{}, []models.AssetRiskScore{}, nil
	}

	// Query assets with their current drift status
	// "At risk" = assets not matching their golden image (stale or drifted)
	// "Improving" = assets that were updated recently (within last 7 days)
	query := `
		SELECT
			a.id,
			a.name,
			a.platform,
			COALESCE(e.name, 'unknown') as environment,
			COALESCE(a.site, 'unknown') as site,
			a.state,
			a.image_ref,
			a.updated_at
		FROM assets a
		LEFT JOIN environments e ON a.env_id = e.id
		WHERE a.org_id = $1
		ORDER BY a.updated_at DESC
		LIMIT 20
	`

	rows, err := s.db.Pool.Query(ctx, query, orgID)
	if err != nil {
		s.log.Error("failed to query assets for trends", "error", err)
		return []models.AssetRiskScore{}, []models.AssetRiskScore{}, nil
	}
	defer rows.Close()

	var atRisk, improving []models.AssetRiskScore
	now := time.Now()

	for rows.Next() {
		var id uuid.UUID
		var name, platform, environment, site, state string
		var imageRef *string
		var updatedAt time.Time

		if err := rows.Scan(&id, &name, &platform, &environment, &site, &state, &imageRef, &updatedAt); err != nil {
			s.log.Error("failed to scan asset row", "error", err)
			continue
		}

		// Calculate risk score based on state and drift age
		driftAge := int(now.Sub(updatedAt).Hours() / 24)
		riskScore := s.calculateAssetRiskScore(state, driftAge, imageRef != nil)

		assetScore := models.AssetRiskScore{
			AssetID:     id,
			AssetName:   name,
			Platform:    platform,
			Environment: environment,
			Site:        site,
			RiskScore:   riskScore,
			RiskLevel:   models.CalculateRiskLevel(riskScore),
			DriftAge:    driftAge,
			LastUpdated: updatedAt,
		}

		// Classify based on update recency and state
		if updatedAt.After(now.AddDate(0, 0, -7)) && state == "running" {
			improving = append(improving, assetScore)
		} else if riskScore >= 60 || state != "running" {
			atRisk = append(atRisk, assetScore)
		}
	}

	if err := rows.Err(); err != nil {
		s.log.Error("error iterating asset rows", "error", err)
	}

	// Limit to top 5 each
	if len(atRisk) > 5 {
		atRisk = atRisk[:5]
	}
	if len(improving) > 5 {
		improving = improving[:5]
	}

	return atRisk, improving, nil
}

// calculateAssetRiskScore computes risk score for an individual asset.
func (s *PredictionService) calculateAssetRiskScore(state string, driftAge int, hasImage bool) float64 {
	score := 30.0 // Base score

	// State factor
	switch state {
	case "running":
		score -= 10
	case "stopped", "terminated":
		score += 20
	case "unknown":
		score += 30
	}

	// Drift age factor (older = higher risk)
	if driftAge > 30 {
		score += 30
	} else if driftAge > 14 {
		score += 20
	} else if driftAge > 7 {
		score += 10
	}

	// Image compliance factor
	if !hasImage {
		score += 15 // No golden image reference
	}

	// Clamp to 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// generateRecommendations creates actionable recommendations based on real data.
func (s *PredictionService) generateRecommendations(ctx context.Context, orgID uuid.UUID, currentScore float64, atRiskAssets []models.AssetRiskScore) []models.RiskRecommendation {
	var recommendations []models.RiskRecommendation

	// Get asset counts from database
	driftedCount, nonCompliantCount, totalAssets := s.getAssetCounts(ctx, orgID)

	// Priority 1: Critical - assets with high drift age
	if len(atRiskAssets) > 0 {
		recommendations = append(recommendations, models.RiskRecommendation{
			ID:             "rec-critical-drift",
			Priority:       1,
			Category:       "drift",
			Title:          "Address Critical Drift",
			Description:    "Remediate assets with significant configuration drift to reduce immediate risk.",
			Impact:         "Reduces risk score by up to 25 points",
			Effort:         "medium",
			AffectedAssets: len(atRiskAssets),
			AutoRemediable: true,
			ActionType:     "ai_task",
		})
	}

	// Priority 2: Drift remediation
	if driftedCount > 0 && currentScore >= 40 {
		recommendations = append(recommendations, models.RiskRecommendation{
			ID:             "rec-drift-fix",
			Priority:       2,
			Category:       "drift",
			Title:          "Remediate Configuration Drift",
			Description:    "Align drifted assets with their golden image baselines.",
			Impact:         "Reduces risk score by up to 15 points",
			Effort:         "low",
			AffectedAssets: driftedCount,
			AutoRemediable: true,
			ActionType:     "ai_task",
		})
	}

	// Priority 3: Compliance gaps
	if nonCompliantCount > 0 {
		recommendations = append(recommendations, models.RiskRecommendation{
			ID:             "rec-compliance",
			Priority:       3,
			Category:       "compliance",
			Title:          "Address Compliance Gaps",
			Description:    "Ensure all assets are running approved, signed golden images.",
			Impact:         "Reduces risk score by up to 10 points",
			Effort:         "medium",
			AffectedAssets: nonCompliantCount,
			AutoRemediable: false,
			ActionType:     "manual",
		})
	}

	// Priority 4: Schedule maintenance for high risk
	if currentScore >= 60 && totalAssets > 0 {
		recommendations = append(recommendations, models.RiskRecommendation{
			ID:             "rec-maintenance",
			Priority:       4,
			Category:       "patch",
			Title:          "Schedule Patch Maintenance Window",
			Description:    "Plan a maintenance window to apply pending patches to production systems.",
			Impact:         "Reduces risk score by up to 20 points",
			Effort:         "high",
			AffectedAssets: totalAssets,
			AutoRemediable: false,
			ActionType:     "scheduled",
		})
	}

	// If no specific recommendations, add a general one
	if len(recommendations) == 0 {
		recommendations = append(recommendations, models.RiskRecommendation{
			ID:             "rec-monitor",
			Priority:       5,
			Category:       "general",
			Title:          "Continue Monitoring",
			Description:    "Risk levels are well controlled. Maintain current posture and monitoring.",
			Impact:         "Maintains stable risk profile",
			Effort:         "low",
			AffectedAssets: totalAssets,
			AutoRemediable: false,
			ActionType:     "manual",
		})
	}

	// Sort by priority
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Priority < recommendations[j].Priority
	})

	// Return top 5
	if len(recommendations) > 5 {
		recommendations = recommendations[:5]
	}

	return recommendations
}

// getAssetCounts retrieves asset counts for recommendations.
func (s *PredictionService) getAssetCounts(ctx context.Context, orgID uuid.UUID) (drifted, nonCompliant, total int) {
	if s.db == nil {
		return 0, 0, 0
	}

	// Get total asset count
	err := s.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM assets WHERE org_id = $1
	`, orgID).Scan(&total)
	if err != nil {
		s.log.Error("failed to count assets", "error", err)
		return 0, 0, 0
	}

	// Get drifted assets (updated more than 7 days ago)
	err = s.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM assets
		WHERE org_id = $1 AND updated_at < NOW() - INTERVAL '7 days'
	`, orgID).Scan(&drifted)
	if err != nil {
		s.log.Error("failed to count drifted assets", "error", err)
	}

	// Get non-compliant assets (no image reference)
	err = s.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM assets
		WHERE org_id = $1 AND image_ref IS NULL
	`, orgID).Scan(&nonCompliant)
	if err != nil {
		s.log.Error("failed to count non-compliant assets", "error", err)
	}

	return drifted, nonCompliant, total
}

// GetAssetPrediction generates a risk prediction for a specific asset.
func (s *PredictionService) GetAssetPrediction(ctx context.Context, assetID uuid.UUID) (*models.RiskPrediction, error) {
	now := time.Now()

	if s.db == nil {
		return &models.RiskPrediction{
			AssetID:           assetID,
			CurrentScore:      0,
			PredictedScore:    0,
			PredictedLevel:    models.RiskLevelLow,
			Confidence:        0,
			PredictionHorizon: 7,
			Velocity:          models.RiskVelocityStable,
			VelocityValue:     0,
			Factors:           []string{"No data available"},
			RecommendedAction: "Connect asset data sources",
			PredictedAt:       now,
		}, nil
	}

	// Get asset data
	var state string
	var imageRef *string
	var updatedAt time.Time
	err := s.db.Pool.QueryRow(ctx, `
		SELECT state, image_ref, updated_at
		FROM assets WHERE id = $1
	`, assetID).Scan(&state, &imageRef, &updatedAt)
	if err != nil {
		return nil, err
	}

	// Calculate current risk score
	driftAge := int(now.Sub(updatedAt).Hours() / 24)
	currentScore := s.calculateAssetRiskScore(state, driftAge, imageRef != nil)

	// Predict future score based on drift age trajectory
	predictedDriftAge := driftAge + 7
	predictedScore := s.calculateAssetRiskScore(state, predictedDriftAge, imageRef != nil)

	// Calculate velocity
	velocityValue := (predictedScore - currentScore) / 7.0

	// Determine factors
	var factors []string
	if driftAge > 14 {
		factors = append(factors, "Significant drift age")
	}
	if imageRef == nil {
		factors = append(factors, "No golden image reference")
	}
	if state != "running" {
		factors = append(factors, "Asset not running")
	}
	if len(factors) == 0 {
		factors = []string{"Asset within compliance parameters"}
	}

	// Determine action
	action := s.recommendAction(currentScore, predictedScore)

	return &models.RiskPrediction{
		AssetID:           assetID,
		CurrentScore:      currentScore,
		PredictedScore:    predictedScore,
		PredictedLevel:    models.CalculateRiskLevel(predictedScore),
		Confidence:        0.85,
		PredictionHorizon: 7,
		Velocity:          models.CalculateVelocity(velocityValue),
		VelocityValue:     math.Round(velocityValue*100) / 100,
		Factors:           factors,
		RecommendedAction: action,
		PredictedAt:       now,
	}, nil
}
