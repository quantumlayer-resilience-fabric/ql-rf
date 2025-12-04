// Package risk provides predictive risk scoring for change operations.
package risk

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// RiskLevel represents the overall risk category.
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// RiskScore represents the complete risk assessment for an operation.
type RiskScore struct {
	OverallScore     float64               `json:"overall_score"`      // 0-100, higher = more risky
	Level            RiskLevel             `json:"level"`              // Categorical level
	Confidence       float64               `json:"confidence"`         // 0-1, how confident we are
	Components       []RiskComponent       `json:"components"`         // Individual risk factors
	Recommendations  []Recommendation      `json:"recommendations"`    // Mitigation suggestions
	ApprovalRequired bool                  `json:"approval_required"`  // Whether human approval is needed
	AutomationSafe   bool                  `json:"automation_safe"`    // Whether safe for full automation
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// RiskComponent represents a single factor in the risk calculation.
type RiskComponent struct {
	Name        string  `json:"name"`
	Score       float64 `json:"score"`       // 0-100
	Weight      float64 `json:"weight"`      // Contribution to overall score
	Description string  `json:"description"`
	Mitigations []string `json:"mitigations,omitempty"`
}

// Recommendation suggests how to reduce risk.
type Recommendation struct {
	Priority    string `json:"priority"` // high, medium, low
	Action      string `json:"action"`
	Rationale   string `json:"rationale"`
	Automated   bool   `json:"automated"` // Can be automated
}

// RiskInput contains all inputs for risk assessment.
type RiskInput struct {
	// Operation details
	OperationType   string `json:"operation_type"`   // patch, reimage, dr-drill, config-change
	Environment     string `json:"environment"`      // production, staging, development
	Platform        string `json:"platform"`         // aws, azure, gcp, vsphere, k8s

	// Scope
	AssetCount      int    `json:"asset_count"`
	CriticalAssets  int    `json:"critical_assets"`
	TotalCapacity   int    `json:"total_capacity"`   // Total assets in environment

	// Asset characteristics
	AssetAge        time.Duration `json:"asset_age"`        // Average age since last update
	DriftDays       int           `json:"drift_days"`       // Days since last golden image
	CurrentVersion  string        `json:"current_version"`
	TargetVersion   string        `json:"target_version"`

	// Historical data
	HistoricalFailureRate float64 `json:"historical_failure_rate"` // Past failure percentage
	LastFailureTime       time.Time `json:"last_failure_time"`
	SuccessStreak         int       `json:"success_streak"`         // Consecutive successes

	// Change characteristics
	ChangeSize       string `json:"change_size"`       // minor, moderate, major
	RollbackAvailable bool   `json:"rollback_available"`
	TestedInStaging   bool   `json:"tested_in_staging"`

	// Time factors
	ScheduledTime     time.Time `json:"scheduled_time"`
	MaintenanceWindow bool      `json:"maintenance_window"`
	PeakHours         bool      `json:"peak_hours"`

	// Dependencies
	DependentServices int  `json:"dependent_services"`
	HasExternalDeps   bool `json:"has_external_deps"`

	// Compliance
	ComplianceRequired bool `json:"compliance_required"`
	AuditMode          bool `json:"audit_mode"`
}

// Scorer calculates risk scores for operations.
type Scorer struct {
	log     *logger.Logger
	weights map[string]float64
	thresholds RiskThresholds
}

// RiskThresholds defines the boundaries for risk levels.
type RiskThresholds struct {
	Low      float64 // Score below this = low risk
	Medium   float64 // Score below this = medium risk
	High     float64 // Score below this = high risk
	// Above High = critical

	AutoApproveMax  float64 // Max score for auto-approval
	CanaryRequired  float64 // Score above this requires canary
}

// DefaultThresholds returns sensible default thresholds.
func DefaultThresholds() RiskThresholds {
	return RiskThresholds{
		Low:            25,
		Medium:         50,
		High:           75,
		AutoApproveMax: 30,
		CanaryRequired: 40,
	}
}

// NewScorer creates a new risk scorer.
func NewScorer(log *logger.Logger) *Scorer {
	return &Scorer{
		log: log.WithComponent("risk-scorer"),
		weights: map[string]float64{
			"environment":     0.20,
			"scope":           0.15,
			"history":         0.15,
			"change_size":     0.15,
			"timing":          0.10,
			"dependencies":    0.10,
			"drift":           0.10,
			"rollback":        0.05,
		},
		thresholds: DefaultThresholds(),
	}
}

// SetThresholds updates the risk thresholds.
func (s *Scorer) SetThresholds(t RiskThresholds) {
	s.thresholds = t
}

// Score calculates the risk score for an operation.
func (s *Scorer) Score(ctx context.Context, input RiskInput) (*RiskScore, error) {
	s.log.Info("calculating risk score",
		"operation", input.OperationType,
		"environment", input.Environment,
		"assets", input.AssetCount,
	)

	components := make([]RiskComponent, 0, 8)

	// 1. Environment risk
	envScore := s.scoreEnvironment(input)
	components = append(components, envScore)

	// 2. Scope risk
	scopeScore := s.scoreScope(input)
	components = append(components, scopeScore)

	// 3. Historical risk
	historyScore := s.scoreHistory(input)
	components = append(components, historyScore)

	// 4. Change size risk
	changeScore := s.scoreChangeSize(input)
	components = append(components, changeScore)

	// 5. Timing risk
	timingScore := s.scoreTiming(input)
	components = append(components, timingScore)

	// 6. Dependencies risk
	depsScore := s.scoreDependencies(input)
	components = append(components, depsScore)

	// 7. Drift risk
	driftScore := s.scoreDrift(input)
	components = append(components, driftScore)

	// 8. Rollback availability risk
	rollbackScore := s.scoreRollback(input)
	components = append(components, rollbackScore)

	// Calculate weighted overall score
	var overallScore float64
	var totalWeight float64
	for _, comp := range components {
		overallScore += comp.Score * comp.Weight
		totalWeight += comp.Weight
	}
	if totalWeight > 0 {
		overallScore = overallScore / totalWeight
	}

	// Determine risk level
	level := s.determineLevel(overallScore)

	// Generate recommendations
	recommendations := s.generateRecommendations(input, components, overallScore)

	// Calculate confidence based on available data
	confidence := s.calculateConfidence(input)

	result := &RiskScore{
		OverallScore:     math.Round(overallScore*100) / 100,
		Level:            level,
		Confidence:       confidence,
		Components:       components,
		Recommendations:  recommendations,
		ApprovalRequired: overallScore > s.thresholds.AutoApproveMax || input.Environment == "production",
		AutomationSafe:   overallScore <= s.thresholds.AutoApproveMax && input.RollbackAvailable,
		Metadata: map[string]interface{}{
			"canary_required":    overallScore > s.thresholds.CanaryRequired,
			"suggested_batch":    s.suggestBatchSize(overallScore, input.AssetCount),
			"suggested_wait":     s.suggestWaitTime(overallScore),
			"calculated_at":      time.Now().UTC(),
		},
	}

	s.log.Info("risk score calculated",
		"overall_score", result.OverallScore,
		"level", result.Level,
		"approval_required", result.ApprovalRequired,
		"automation_safe", result.AutomationSafe,
	)

	return result, nil
}

// scoreEnvironment assesses environment-based risk.
func (s *Scorer) scoreEnvironment(input RiskInput) RiskComponent {
	var score float64
	var desc string
	mitigations := []string{}

	switch input.Environment {
	case "production":
		score = 80
		desc = "Production environment - highest impact"
		mitigations = append(mitigations, "Use canary deployment", "Require manual approval")
	case "staging":
		score = 40
		desc = "Staging environment - moderate impact"
		mitigations = append(mitigations, "Validate before production")
	case "development":
		score = 10
		desc = "Development environment - low impact"
	default:
		score = 50
		desc = "Unknown environment"
	}

	return RiskComponent{
		Name:        "environment",
		Score:       score,
		Weight:      s.weights["environment"],
		Description: desc,
		Mitigations: mitigations,
	}
}

// scoreScope assesses risk based on operation scope.
func (s *Scorer) scoreScope(input RiskInput) RiskComponent {
	var score float64
	var desc string
	mitigations := []string{}

	// Calculate percentage of fleet affected
	var pctAffected float64
	if input.TotalCapacity > 0 {
		pctAffected = float64(input.AssetCount) / float64(input.TotalCapacity) * 100
	}

	// Base score on percentage
	if pctAffected >= 50 {
		score = 90
		desc = fmt.Sprintf("Large scope: %.1f%% of fleet (%d assets)", pctAffected, input.AssetCount)
		mitigations = append(mitigations, "Split into smaller batches", "Use rolling deployment")
	} else if pctAffected >= 25 {
		score = 60
		desc = fmt.Sprintf("Medium scope: %.1f%% of fleet (%d assets)", pctAffected, input.AssetCount)
		mitigations = append(mitigations, "Consider phased rollout")
	} else if pctAffected >= 10 {
		score = 40
		desc = fmt.Sprintf("Moderate scope: %.1f%% of fleet (%d assets)", pctAffected, input.AssetCount)
	} else {
		score = 20
		desc = fmt.Sprintf("Small scope: %.1f%% of fleet (%d assets)", pctAffected, input.AssetCount)
	}

	// Adjust for critical assets
	if input.CriticalAssets > 0 {
		criticalPct := float64(input.CriticalAssets) / float64(input.AssetCount) * 100
		score = math.Min(100, score + criticalPct*0.3)
		mitigations = append(mitigations, fmt.Sprintf("Update %d critical assets last", input.CriticalAssets))
	}

	return RiskComponent{
		Name:        "scope",
		Score:       score,
		Weight:      s.weights["scope"],
		Description: desc,
		Mitigations: mitigations,
	}
}

// scoreHistory assesses risk based on historical performance.
func (s *Scorer) scoreHistory(input RiskInput) RiskComponent {
	var score float64
	var desc string
	mitigations := []string{}

	// Base score on historical failure rate
	score = input.HistoricalFailureRate * 100

	// Adjust for recent failures
	if !input.LastFailureTime.IsZero() {
		daysSinceFailure := time.Since(input.LastFailureTime).Hours() / 24
		if daysSinceFailure < 7 {
			score = math.Min(100, score + 30)
			mitigations = append(mitigations, "Recent failure detected - investigate root cause")
		} else if daysSinceFailure < 30 {
			score = math.Min(100, score + 15)
		}
	}

	// Reward success streaks
	if input.SuccessStreak > 10 {
		score = math.Max(0, score - 20)
		desc = fmt.Sprintf("Good track record: %d consecutive successes, %.1f%% historical failure rate",
			input.SuccessStreak, input.HistoricalFailureRate*100)
	} else {
		desc = fmt.Sprintf("Historical failure rate: %.1f%%, success streak: %d",
			input.HistoricalFailureRate*100, input.SuccessStreak)
	}

	return RiskComponent{
		Name:        "history",
		Score:       score,
		Weight:      s.weights["history"],
		Description: desc,
		Mitigations: mitigations,
	}
}

// scoreChangeSize assesses risk based on change magnitude.
func (s *Scorer) scoreChangeSize(input RiskInput) RiskComponent {
	var score float64
	var desc string
	mitigations := []string{}

	switch input.ChangeSize {
	case "major":
		score = 80
		desc = "Major change - significant modifications"
		mitigations = append(mitigations,
			"Test in staging first",
			"Use canary deployment",
			"Prepare rollback plan",
		)
	case "moderate":
		score = 50
		desc = "Moderate change"
		mitigations = append(mitigations, "Verify in staging")
	case "minor":
		score = 20
		desc = "Minor change - low modification"
	default:
		score = 50
		desc = "Unknown change size"
	}

	// Adjust if tested in staging
	if input.TestedInStaging {
		score = math.Max(0, score - 15)
		desc += " (validated in staging)"
	} else if input.Environment == "production" {
		mitigations = append(mitigations, "CRITICAL: Test in staging before production")
	}

	return RiskComponent{
		Name:        "change_size",
		Score:       score,
		Weight:      s.weights["change_size"],
		Description: desc,
		Mitigations: mitigations,
	}
}

// scoreTiming assesses risk based on when the operation runs.
func (s *Scorer) scoreTiming(input RiskInput) RiskComponent {
	var score float64
	var desc string
	mitigations := []string{}

	if input.PeakHours {
		score = 80
		desc = "Peak hours - high user impact"
		mitigations = append(mitigations, "Schedule during maintenance window")
	} else if input.MaintenanceWindow {
		score = 10
		desc = "Within maintenance window"
	} else {
		// Check day of week
		scheduledDay := input.ScheduledTime.Weekday()
		if scheduledDay == time.Friday || scheduledDay == time.Saturday || scheduledDay == time.Sunday {
			score = 60
			desc = "Weekend/Friday deployment - limited support"
			mitigations = append(mitigations, "Consider scheduling for mid-week")
		} else {
			score = 30
			desc = "Standard business hours"
		}
	}

	return RiskComponent{
		Name:        "timing",
		Score:       score,
		Weight:      s.weights["timing"],
		Description: desc,
		Mitigations: mitigations,
	}
}

// scoreDependencies assesses risk based on service dependencies.
func (s *Scorer) scoreDependencies(input RiskInput) RiskComponent {
	var score float64
	var desc string
	mitigations := []string{}

	// More dependencies = more risk
	score = math.Min(100, float64(input.DependentServices) * 15)

	if input.HasExternalDeps {
		score = math.Min(100, score + 25)
		mitigations = append(mitigations, "Notify external dependency owners")
	}

	if input.DependentServices > 5 {
		desc = fmt.Sprintf("High coupling: %d dependent services", input.DependentServices)
		mitigations = append(mitigations,
			"Notify dependent service owners",
			"Stagger updates to avoid cascade failures",
		)
	} else if input.DependentServices > 0 {
		desc = fmt.Sprintf("%d dependent services", input.DependentServices)
	} else {
		desc = "No known dependencies"
	}

	return RiskComponent{
		Name:        "dependencies",
		Score:       score,
		Weight:      s.weights["dependencies"],
		Description: desc,
		Mitigations: mitigations,
	}
}

// scoreDrift assesses risk based on configuration drift.
func (s *Scorer) scoreDrift(input RiskInput) RiskComponent {
	var score float64
	var desc string
	mitigations := []string{}

	if input.DriftDays > 90 {
		score = 90
		desc = fmt.Sprintf("Severe drift: %d days behind golden image", input.DriftDays)
		mitigations = append(mitigations,
			"HIGH PRIORITY: Update immediately",
			"Use smaller batch sizes",
			"Extended monitoring after update",
		)
	} else if input.DriftDays > 30 {
		score = 60
		desc = fmt.Sprintf("Moderate drift: %d days behind golden image", input.DriftDays)
		mitigations = append(mitigations, "Schedule update within 2 weeks")
	} else if input.DriftDays > 14 {
		score = 40
		desc = fmt.Sprintf("Minor drift: %d days behind golden image", input.DriftDays)
	} else {
		score = 10
		desc = fmt.Sprintf("Low drift: %d days behind golden image", input.DriftDays)
	}

	return RiskComponent{
		Name:        "drift",
		Score:       score,
		Weight:      s.weights["drift"],
		Description: desc,
		Mitigations: mitigations,
	}
}

// scoreRollback assesses risk based on rollback capability.
func (s *Scorer) scoreRollback(input RiskInput) RiskComponent {
	var score float64
	var desc string
	mitigations := []string{}

	if input.RollbackAvailable {
		score = 10
		desc = "Rollback available"
	} else {
		score = 70
		desc = "No rollback available - high risk"
		mitigations = append(mitigations,
			"Create snapshot before operation",
			"Ensure backup exists",
			"Test restore procedure",
		)
	}

	return RiskComponent{
		Name:        "rollback",
		Score:       score,
		Weight:      s.weights["rollback"],
		Description: desc,
		Mitigations: mitigations,
	}
}

// determineLevel converts numeric score to categorical level.
func (s *Scorer) determineLevel(score float64) RiskLevel {
	if score < s.thresholds.Low {
		return RiskLevelLow
	} else if score < s.thresholds.Medium {
		return RiskLevelMedium
	} else if score < s.thresholds.High {
		return RiskLevelHigh
	}
	return RiskLevelCritical
}

// generateRecommendations creates actionable suggestions.
func (s *Scorer) generateRecommendations(input RiskInput, components []RiskComponent, overallScore float64) []Recommendation {
	recs := []Recommendation{}

	// Always recommend canary for high-risk production deployments
	if overallScore > s.thresholds.CanaryRequired && input.Environment == "production" {
		recs = append(recs, Recommendation{
			Priority:  "high",
			Action:    "Use canary deployment with 5% initial rollout",
			Rationale: "High-risk production change requires gradual rollout",
			Automated: true,
		})
	}

	// Collect high-score component mitigations
	for _, comp := range components {
		if comp.Score > 60 {
			for _, mit := range comp.Mitigations {
				recs = append(recs, Recommendation{
					Priority:  "high",
					Action:    mit,
					Rationale: fmt.Sprintf("High %s risk (%.0f)", comp.Name, comp.Score),
					Automated: false,
				})
			}
		}
	}

	// Add rollback recommendation if not available
	if !input.RollbackAvailable {
		recs = append(recs, Recommendation{
			Priority:  "high",
			Action:    "Create snapshot or backup before proceeding",
			Rationale: "No rollback mechanism available",
			Automated: true,
		})
	}

	// Add staging recommendation for untested production changes
	if !input.TestedInStaging && input.Environment == "production" {
		recs = append(recs, Recommendation{
			Priority:  "high",
			Action:    "Test change in staging environment first",
			Rationale: "Production changes should be validated in staging",
			Automated: false,
		})
	}

	return recs
}

// calculateConfidence determines how confident we are in the score.
func (s *Scorer) calculateConfidence(input RiskInput) float64 {
	confidence := 1.0

	// Reduce confidence if we lack historical data
	if input.SuccessStreak == 0 && input.HistoricalFailureRate == 0 {
		confidence -= 0.2 // No history
	}

	// Reduce confidence for unknown change size
	if input.ChangeSize == "" {
		confidence -= 0.1
	}

	// Reduce confidence for unknown environment
	if input.Environment == "" {
		confidence -= 0.15
	}

	return math.Max(0.5, confidence) // Minimum 50% confidence
}

// suggestBatchSize recommends a safe batch size based on risk.
func (s *Scorer) suggestBatchSize(score float64, totalAssets int) int {
	if totalAssets <= 1 {
		return 1
	}

	var pct float64
	if score >= 75 {
		pct = 0.05 // 5% for critical risk
	} else if score >= 50 {
		pct = 0.10 // 10% for high risk
	} else if score >= 25 {
		pct = 0.25 // 25% for medium risk
	} else {
		pct = 0.50 // 50% for low risk
	}

	batch := int(float64(totalAssets) * pct)
	if batch < 1 {
		batch = 1
	}
	return batch
}

// suggestWaitTime recommends wait time between phases.
func (s *Scorer) suggestWaitTime(score float64) string {
	if score >= 75 {
		return "30m"
	} else if score >= 50 {
		return "15m"
	} else if score >= 25 {
		return "5m"
	}
	return "2m"
}

// ToJSON serializes the risk score to JSON.
func (r *RiskScore) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}
