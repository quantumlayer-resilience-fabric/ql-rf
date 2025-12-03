// Package models provides domain models for QL-RF.
package models

import (
	"time"

	"github.com/google/uuid"
)

// RiskLevel represents the risk severity level.
type RiskLevel string

const (
	RiskLevelCritical RiskLevel = "critical"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelLow      RiskLevel = "low"
)

// RiskFactor represents a contributing factor to the risk score.
type RiskFactor struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Weight      float64 `json:"weight"`
	Score       float64 `json:"score"`
	Impact      string  `json:"impact"` // "positive" or "negative"
}

// AssetRiskScore represents the calculated risk score for an asset.
type AssetRiskScore struct {
	AssetID       uuid.UUID    `json:"assetId"`
	AssetName     string       `json:"assetName"`
	Platform      string       `json:"platform"`
	Environment   string       `json:"environment"`
	Site          string       `json:"site"`
	RiskScore     float64      `json:"riskScore"`     // 0-100, higher = more risky
	RiskLevel     RiskLevel    `json:"riskLevel"`     // critical, high, medium, low
	Factors       []RiskFactor `json:"factors"`       // Contributing factors
	DriftAge      int          `json:"driftAge"`      // Days since drift detected
	VulnCount     int          `json:"vulnCount"`     // Open vulnerability count
	CriticalVulns int          `json:"criticalVulns"` // Critical severity vulns
	IsCompliant   bool         `json:"isCompliant"`   // Compliance status
	LastUpdated   time.Time    `json:"lastUpdated"`
}

// RiskSummary provides organization-wide risk metrics.
type RiskSummary struct {
	OrgID            uuid.UUID        `json:"orgId"`
	OverallRiskScore float64          `json:"overallRiskScore"`
	RiskLevel        RiskLevel        `json:"riskLevel"`
	TotalAssets      int              `json:"totalAssets"`
	CriticalRisk     int              `json:"criticalRisk"`
	HighRisk         int              `json:"highRisk"`
	MediumRisk       int              `json:"mediumRisk"`
	LowRisk          int              `json:"lowRisk"`
	TopRisks         []AssetRiskScore `json:"topRisks"`
	ByEnvironment    []RiskByScope    `json:"byEnvironment"`
	ByPlatform       []RiskByScope    `json:"byPlatform"`
	BySite           []RiskByScope    `json:"bySite"`
	Trend            []RiskTrendPoint `json:"trend"`
	CalculatedAt     time.Time        `json:"calculatedAt"`
}

// RiskByScope provides risk metrics grouped by scope (env/platform/site).
type RiskByScope struct {
	Scope        string    `json:"scope"`
	RiskScore    float64   `json:"riskScore"`
	RiskLevel    RiskLevel `json:"riskLevel"`
	AssetCount   int       `json:"assetCount"`
	CriticalRisk int       `json:"criticalRisk"`
	HighRisk     int       `json:"highRisk"`
}

// RiskTrendPoint represents risk score at a point in time.
type RiskTrendPoint struct {
	Date      time.Time `json:"date"`
	RiskScore float64   `json:"riskScore"`
	RiskLevel RiskLevel `json:"riskLevel"`
}

// RiskScoreWeights defines the weights for risk calculation.
type RiskScoreWeights struct {
	DriftAge         float64 `json:"driftAge"`         // Weight for drift age factor
	VulnCount        float64 `json:"vulnCount"`        // Weight for vulnerability count
	CriticalVulns    float64 `json:"criticalVulns"`    // Weight for critical vulns
	ComplianceStatus float64 `json:"complianceStatus"` // Weight for compliance
	Environment      float64 `json:"environment"`      // Weight for environment impact
}

// DefaultRiskWeights returns the default risk calculation weights.
func DefaultRiskWeights() RiskScoreWeights {
	return RiskScoreWeights{
		DriftAge:         0.25, // 25% weight
		VulnCount:        0.20, // 20% weight
		CriticalVulns:    0.25, // 25% weight
		ComplianceStatus: 0.15, // 15% weight
		Environment:      0.15, // 15% weight
	}
}

// CalculateRiskLevel converts a risk score to a risk level.
func CalculateRiskLevel(score float64) RiskLevel {
	switch {
	case score >= 80:
		return RiskLevelCritical
	case score >= 60:
		return RiskLevelHigh
	case score >= 40:
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}

// EnvironmentRiskMultiplier returns the risk multiplier for an environment.
func EnvironmentRiskMultiplier(env string) float64 {
	switch env {
	case "production":
		return 1.5 // Production has 50% higher impact
	case "staging":
		return 1.0 // Baseline
	case "development":
		return 0.5 // Development has 50% lower impact
	case "dr":
		return 1.2 // DR sites are important
	default:
		return 1.0
	}
}

// RiskVelocity represents the rate of change in risk score.
type RiskVelocity string

const (
	RiskVelocityRapidIncrease   RiskVelocity = "rapid_increase"   // >10 points/day
	RiskVelocityIncreasing      RiskVelocity = "increasing"       // 2-10 points/day
	RiskVelocityStable          RiskVelocity = "stable"           // -2 to 2 points/day
	RiskVelocityDecreasing      RiskVelocity = "decreasing"       // -10 to -2 points/day
	RiskVelocityRapidDecrease   RiskVelocity = "rapid_decrease"   // <-10 points/day
)

// RiskPrediction represents a predicted future risk state.
type RiskPrediction struct {
	AssetID           uuid.UUID    `json:"assetId,omitempty"`
	Scope             string       `json:"scope,omitempty"`             // For org/env/platform predictions
	CurrentScore      float64      `json:"currentScore"`
	PredictedScore    float64      `json:"predictedScore"`
	PredictedLevel    RiskLevel    `json:"predictedLevel"`
	Confidence        float64      `json:"confidence"`                  // 0-1 confidence level
	PredictionHorizon int          `json:"predictionHorizon"`           // Days ahead
	Velocity          RiskVelocity `json:"velocity"`
	VelocityValue     float64      `json:"velocityValue"`               // Points per day
	Factors           []string     `json:"factors"`                     // Contributing factors
	RecommendedAction string       `json:"recommendedAction,omitempty"`
	PredictedAt       time.Time    `json:"predictedAt"`
}

// RiskAnomaly represents an unusual risk pattern.
type RiskAnomaly struct {
	ID           uuid.UUID  `json:"id"`
	AssetID      uuid.UUID  `json:"assetId,omitempty"`
	Scope        string     `json:"scope,omitempty"`
	AnomalyType  string     `json:"anomalyType"`  // spike, drop, pattern_break
	Severity     RiskLevel  `json:"severity"`
	Description  string     `json:"description"`
	ExpectedScore float64   `json:"expectedScore"`
	ActualScore  float64    `json:"actualScore"`
	Deviation    float64    `json:"deviation"`    // Standard deviations from mean
	DetectedAt   time.Time  `json:"detectedAt"`
	IsActive     bool       `json:"isActive"`
}

// RiskForecast provides organization-wide risk predictions.
type RiskForecast struct {
	OrgID             uuid.UUID        `json:"orgId"`
	CurrentScore      float64          `json:"currentScore"`
	Predictions       []RiskPrediction `json:"predictions"`       // 7, 14, 30 day predictions
	Velocity          RiskVelocity     `json:"velocity"`
	VelocityValue     float64          `json:"velocityValue"`
	Anomalies         []RiskAnomaly    `json:"anomalies"`
	AtRiskAssets      []AssetRiskScore `json:"atRiskAssets"`      // Assets predicted to breach threshold
	ImprovingAssets   []AssetRiskScore `json:"improvingAssets"`   // Assets with decreasing risk
	TopRecommendations []RiskRecommendation `json:"topRecommendations"`
	GeneratedAt       time.Time        `json:"generatedAt"`
}

// RiskRecommendation represents an actionable risk mitigation suggestion.
type RiskRecommendation struct {
	ID             string    `json:"id"`
	Priority       int       `json:"priority"`       // 1 = highest
	Category       string    `json:"category"`       // patch, compliance, vulnerability, drift
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Impact         string    `json:"impact"`         // Expected risk reduction
	Effort         string    `json:"effort"`         // low, medium, high
	AffectedAssets int       `json:"affectedAssets"`
	AutoRemediable bool      `json:"autoRemediable"` // Can be auto-fixed
	ActionType     string    `json:"actionType"`     // ai_task, manual, scheduled
}

// AutoRemediationPolicy defines rules for automatic remediation.
type AutoRemediationPolicy struct {
	ID              uuid.UUID `json:"id"`
	OrgID           uuid.UUID `json:"orgId"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Enabled         bool      `json:"enabled"`

	// Conditions
	MaxRiskLevel    RiskLevel `json:"maxRiskLevel"`    // Only auto-remediate up to this level
	Environments    []string  `json:"environments"`    // Allowed environments
	Platforms       []string  `json:"platforms"`       // Allowed platforms
	Categories      []string  `json:"categories"`      // drift, patch, compliance

	// Actions
	RequireApproval bool      `json:"requireApproval"` // Still require human approval
	NotifyOnAction  bool      `json:"notifyOnAction"`
	MaxActionsPerDay int      `json:"maxActionsPerDay"`

	// Schedule
	AllowedWindows  []MaintenanceWindow `json:"allowedWindows"`

	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// MaintenanceWindow defines when auto-remediation can occur.
type MaintenanceWindow struct {
	DayOfWeek int    `json:"dayOfWeek"` // 0=Sunday, 6=Saturday
	StartHour int    `json:"startHour"` // 0-23
	EndHour   int    `json:"endHour"`   // 0-23
	Timezone  string `json:"timezone"`
}

// CalculateVelocity determines the risk velocity from a rate of change.
func CalculateVelocity(pointsPerDay float64) RiskVelocity {
	switch {
	case pointsPerDay > 10:
		return RiskVelocityRapidIncrease
	case pointsPerDay > 2:
		return RiskVelocityIncreasing
	case pointsPerDay >= -2:
		return RiskVelocityStable
	case pointsPerDay >= -10:
		return RiskVelocityDecreasing
	default:
		return RiskVelocityRapidDecrease
	}
}
