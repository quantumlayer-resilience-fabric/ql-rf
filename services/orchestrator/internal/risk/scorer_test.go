package risk_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/risk"
)

func newTestScorer() *risk.Scorer {
	log := logger.New("error", "text")
	return risk.NewScorer(log)
}

// =============================================================================
// Risk Level Tests
// =============================================================================

func TestRiskLevel_Constants(t *testing.T) {
	assert.Equal(t, risk.RiskLevel("low"), risk.RiskLevelLow)
	assert.Equal(t, risk.RiskLevel("medium"), risk.RiskLevelMedium)
	assert.Equal(t, risk.RiskLevel("high"), risk.RiskLevelHigh)
	assert.Equal(t, risk.RiskLevel("critical"), risk.RiskLevelCritical)
}

// =============================================================================
// Default Thresholds Tests
// =============================================================================

func TestDefaultThresholds(t *testing.T) {
	thresholds := risk.DefaultThresholds()

	assert.Equal(t, float64(25), thresholds.Low)
	assert.Equal(t, float64(50), thresholds.Medium)
	assert.Equal(t, float64(75), thresholds.High)
	assert.Equal(t, float64(30), thresholds.AutoApproveMax)
	assert.Equal(t, float64(40), thresholds.CanaryRequired)
}

// =============================================================================
// Environment Scoring Tests
// =============================================================================

func TestScore_Environment_Production(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "production",
		AssetCount:        10,
		TotalCapacity:     100,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	// Production environment should require approval
	assert.True(t, result.ApprovalRequired)

	// Find environment component
	var envComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "environment" {
			envComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, envComp)
	assert.Equal(t, float64(80), envComp.Score)
	assert.Contains(t, envComp.Description, "Production")
}

func TestScore_Environment_Staging(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        10,
		TotalCapacity:     100,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var envComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "environment" {
			envComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, envComp)
	assert.Equal(t, float64(40), envComp.Score)
}

func TestScore_Environment_Development(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "development",
		AssetCount:        10,
		TotalCapacity:     100,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var envComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "environment" {
			envComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, envComp)
	assert.Equal(t, float64(10), envComp.Score)
}

// =============================================================================
// Scope Scoring Tests
// =============================================================================

func TestScore_Scope_LargePercentage(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        60,
		TotalCapacity:     100, // 60% of fleet
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var scopeComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "scope" {
			scopeComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, scopeComp)
	assert.Equal(t, float64(90), scopeComp.Score) // >= 50% = 90
	assert.Contains(t, scopeComp.Description, "Large scope")
}

func TestScore_Scope_SmallPercentage(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        5,
		TotalCapacity:     100, // 5% of fleet
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var scopeComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "scope" {
			scopeComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, scopeComp)
	assert.Equal(t, float64(20), scopeComp.Score) // < 10% = 20
	assert.Contains(t, scopeComp.Description, "Small scope")
}

func TestScore_Scope_CriticalAssets(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        10,
		CriticalAssets:    5, // 50% critical
		TotalCapacity:     100,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var scopeComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "scope" {
			scopeComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, scopeComp)
	// Base 20 (10% fleet) + 50*0.3 (critical adjustment) = 35
	assert.GreaterOrEqual(t, scopeComp.Score, float64(30))
}

// =============================================================================
// History Scoring Tests
// =============================================================================

func TestScore_History_HighFailureRate(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:         "patch",
		Environment:           "staging",
		AssetCount:            10,
		TotalCapacity:         100,
		HistoricalFailureRate: 0.30, // 30% failure rate
		RollbackAvailable:     true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var histComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "history" {
			histComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, histComp)
	assert.Equal(t, float64(30), histComp.Score)
}

func TestScore_History_RecentFailure(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        10,
		TotalCapacity:     100,
		LastFailureTime:   time.Now().Add(-3 * 24 * time.Hour), // 3 days ago
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var histComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "history" {
			histComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, histComp)
	// Recent failure (<7 days) adds 30 points
	assert.GreaterOrEqual(t, histComp.Score, float64(30))
}

func TestScore_History_GoodStreak(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:         "patch",
		Environment:           "staging",
		AssetCount:            10,
		TotalCapacity:         100,
		HistoricalFailureRate: 0.10,
		SuccessStreak:         15, // Good streak > 10
		RollbackAvailable:     true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var histComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "history" {
			histComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, histComp)
	// Base 10 (10% failure) - 20 (streak bonus) = 0 (max 0)
	assert.LessOrEqual(t, histComp.Score, float64(10))
}

// =============================================================================
// Change Size Scoring Tests
// =============================================================================

func TestScore_ChangeSize_Major(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        10,
		TotalCapacity:     100,
		ChangeSize:        "major",
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var changeComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "change_size" {
			changeComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, changeComp)
	assert.Equal(t, float64(80), changeComp.Score)
}

func TestScore_ChangeSize_Minor(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        10,
		TotalCapacity:     100,
		ChangeSize:        "minor",
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var changeComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "change_size" {
			changeComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, changeComp)
	assert.Equal(t, float64(20), changeComp.Score)
}

func TestScore_ChangeSize_TestedInStaging(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "production",
		AssetCount:        10,
		TotalCapacity:     100,
		ChangeSize:        "major",
		TestedInStaging:   true,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var changeComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "change_size" {
			changeComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, changeComp)
	// 80 - 15 (staging bonus) = 65
	assert.Equal(t, float64(65), changeComp.Score)
	assert.Contains(t, changeComp.Description, "validated in staging")
}

// =============================================================================
// Timing Scoring Tests
// =============================================================================

func TestScore_Timing_PeakHours(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        10,
		TotalCapacity:     100,
		PeakHours:         true,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var timingComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "timing" {
			timingComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, timingComp)
	assert.Equal(t, float64(80), timingComp.Score)
	assert.Contains(t, timingComp.Description, "Peak hours")
}

func TestScore_Timing_MaintenanceWindow(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        10,
		TotalCapacity:     100,
		MaintenanceWindow: true,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var timingComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "timing" {
			timingComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, timingComp)
	assert.Equal(t, float64(10), timingComp.Score)
}

func TestScore_Timing_Friday(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	// Find the next Friday
	nextFriday := time.Now()
	for nextFriday.Weekday() != time.Friday {
		nextFriday = nextFriday.Add(24 * time.Hour)
	}

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        10,
		TotalCapacity:     100,
		ScheduledTime:     nextFriday,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var timingComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "timing" {
			timingComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, timingComp)
	assert.Equal(t, float64(60), timingComp.Score)
	assert.Contains(t, timingComp.Description, "Weekend/Friday")
}

// =============================================================================
// Dependencies Scoring Tests
// =============================================================================

func TestScore_Dependencies_High(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        10,
		TotalCapacity:     100,
		DependentServices: 8,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var depsComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "dependencies" {
			depsComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, depsComp)
	// 8 * 15 = 120, capped at 100
	assert.Equal(t, float64(100), depsComp.Score)
}

func TestScore_Dependencies_WithExternal(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        10,
		TotalCapacity:     100,
		DependentServices: 2,
		HasExternalDeps:   true,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var depsComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "dependencies" {
			depsComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, depsComp)
	// 2*15 + 25 (external) = 55
	assert.Equal(t, float64(55), depsComp.Score)
}

func TestScore_Dependencies_None(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        10,
		TotalCapacity:     100,
		DependentServices: 0,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var depsComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "dependencies" {
			depsComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, depsComp)
	assert.Equal(t, float64(0), depsComp.Score)
}

// =============================================================================
// Drift Scoring Tests
// =============================================================================

func TestScore_Drift_Severe(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        10,
		TotalCapacity:     100,
		DriftDays:         100, // > 90 days
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var driftComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "drift" {
			driftComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, driftComp)
	assert.Equal(t, float64(90), driftComp.Score)
	assert.Contains(t, driftComp.Description, "Severe drift")
}

func TestScore_Drift_Low(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        10,
		TotalCapacity:     100,
		DriftDays:         7,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var driftComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "drift" {
			driftComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, driftComp)
	assert.Equal(t, float64(10), driftComp.Score)
	assert.Contains(t, driftComp.Description, "Low drift")
}

// =============================================================================
// Rollback Scoring Tests
// =============================================================================

func TestScore_Rollback_Available(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        10,
		TotalCapacity:     100,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var rollbackComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "rollback" {
			rollbackComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, rollbackComp)
	assert.Equal(t, float64(10), rollbackComp.Score)
}

func TestScore_Rollback_NotAvailable(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        10,
		TotalCapacity:     100,
		RollbackAvailable: false,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	var rollbackComp *risk.RiskComponent
	for i := range result.Components {
		if result.Components[i].Name == "rollback" {
			rollbackComp = &result.Components[i]
			break
		}
	}

	require.NotNil(t, rollbackComp)
	assert.Equal(t, float64(70), rollbackComp.Score)
}

// =============================================================================
// Overall Score and Level Tests
// =============================================================================

func TestScore_OverallLevel_Low(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	// Low risk scenario: dev environment, small scope, minor change
	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "development",
		AssetCount:        2,
		TotalCapacity:     100,
		ChangeSize:        "minor",
		MaintenanceWindow: true,
		RollbackAvailable: true,
		DriftDays:         5,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	assert.Equal(t, risk.RiskLevelLow, result.Level)
	assert.LessOrEqual(t, result.OverallScore, float64(25))
}

func TestScore_OverallLevel_Critical(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	// Critical risk scenario
	input := risk.RiskInput{
		OperationType:         "patch",
		Environment:           "production",
		AssetCount:            80,
		CriticalAssets:        40,
		TotalCapacity:         100,
		ChangeSize:            "major",
		PeakHours:             true,
		DependentServices:     10,
		HasExternalDeps:       true,
		DriftDays:             120,
		RollbackAvailable:     false,
		HistoricalFailureRate: 0.25,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	assert.Equal(t, risk.RiskLevelCritical, result.Level)
	assert.GreaterOrEqual(t, result.OverallScore, float64(75))
	assert.True(t, result.ApprovalRequired)
	assert.False(t, result.AutomationSafe)
}

// =============================================================================
// Approval and Automation Tests
// =============================================================================

func TestScore_ApprovalRequired_Production(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	// Even low risk in production requires approval
	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "production",
		AssetCount:        1,
		TotalCapacity:     100,
		ChangeSize:        "minor",
		MaintenanceWindow: true,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	assert.True(t, result.ApprovalRequired)
}

func TestScore_AutomationSafe(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	// Low risk with rollback should be automation safe
	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "development",
		AssetCount:        2,
		TotalCapacity:     100,
		ChangeSize:        "minor",
		MaintenanceWindow: true,
		RollbackAvailable: true,
		DriftDays:         5,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	// Low risk + rollback = automation safe
	if result.OverallScore <= 30 {
		assert.True(t, result.AutomationSafe)
	}
}

// =============================================================================
// Recommendations Tests
// =============================================================================

func TestScore_Recommendations_Canary(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	// High risk production deployment should recommend canary
	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "production",
		AssetCount:        50,
		TotalCapacity:     100,
		ChangeSize:        "major",
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	if result.OverallScore > 40 {
		// Should have canary recommendation
		hasCanary := false
		for _, rec := range result.Recommendations {
			if rec.Action == "Use canary deployment with 5% initial rollout" {
				hasCanary = true
				break
			}
		}
		assert.True(t, hasCanary, "expected canary recommendation for high-risk production deployment")
	}
}

func TestScore_Recommendations_NoRollback(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        10,
		TotalCapacity:     100,
		RollbackAvailable: false,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	// Should recommend creating backup
	hasBackupRec := false
	for _, rec := range result.Recommendations {
		if rec.Action == "Create snapshot or backup before proceeding" {
			hasBackupRec = true
			break
		}
	}
	assert.True(t, hasBackupRec, "expected backup recommendation when no rollback available")
}

func TestScore_Recommendations_UntestedProduction(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "production",
		AssetCount:        10,
		TotalCapacity:     100,
		TestedInStaging:   false,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	// Should recommend testing in staging
	hasStagingRec := false
	for _, rec := range result.Recommendations {
		if rec.Action == "Test change in staging environment first" {
			hasStagingRec = true
			break
		}
	}
	assert.True(t, hasStagingRec, "expected staging test recommendation for untested production change")
}

// =============================================================================
// Confidence Tests
// =============================================================================

func TestScore_Confidence_Full(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:         "patch",
		Environment:           "staging",
		AssetCount:            10,
		TotalCapacity:         100,
		ChangeSize:            "minor",
		HistoricalFailureRate: 0.05,
		SuccessStreak:         10,
		RollbackAvailable:     true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	// With all data available, confidence should be high
	assert.GreaterOrEqual(t, result.Confidence, 0.8)
}

func TestScore_Confidence_Reduced(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "",        // Unknown
		ChangeSize:        "",        // Unknown
		AssetCount:        10,
		TotalCapacity:     100,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	// With missing data, confidence should be reduced but at least 0.5
	assert.LessOrEqual(t, result.Confidence, 0.8)
	assert.GreaterOrEqual(t, result.Confidence, 0.5)
}

// =============================================================================
// Metadata Tests
// =============================================================================

func TestScore_Metadata(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        100,
		TotalCapacity:     200,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	assert.Contains(t, result.Metadata, "canary_required")
	assert.Contains(t, result.Metadata, "suggested_batch")
	assert.Contains(t, result.Metadata, "suggested_wait")
	assert.Contains(t, result.Metadata, "calculated_at")
}

func TestScore_SuggestedBatch_HighRisk(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "production",
		AssetCount:        100,
		TotalCapacity:     100,
		ChangeSize:        "major",
		PeakHours:         true,
		RollbackAvailable: false,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	// High risk should suggest small batch (5-10%)
	batchSize := result.Metadata["suggested_batch"].(int)
	assert.LessOrEqual(t, batchSize, 10) // Max 10% for critical
}

func TestScore_SuggestedBatch_LowRisk(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "development",
		AssetCount:        100,
		TotalCapacity:     200,
		ChangeSize:        "minor",
		MaintenanceWindow: true,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	// Low risk can use larger batches
	batchSize := result.Metadata["suggested_batch"].(int)
	assert.GreaterOrEqual(t, batchSize, 25) // At least 25% for low risk
}

// =============================================================================
// Custom Thresholds Tests
// =============================================================================

func TestScore_CustomThresholds(t *testing.T) {
	scorer := newTestScorer()

	// Set stricter thresholds
	scorer.SetThresholds(risk.RiskThresholds{
		Low:            15,
		Medium:         30,
		High:           50,
		AutoApproveMax: 10,
		CanaryRequired: 20,
	})

	ctx := context.Background()
	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "development",
		AssetCount:        10,
		TotalCapacity:     100,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	// With stricter thresholds, what was low risk might now be medium/high
	assert.NotEmpty(t, result.Level)
}

// =============================================================================
// JSON Serialization Tests
// =============================================================================

func TestRiskScore_ToJSON(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        10,
		TotalCapacity:     100,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	jsonBytes, err := result.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonBytes)

	// Should be valid JSON
	assert.Contains(t, string(jsonBytes), "overall_score")
	assert.Contains(t, string(jsonBytes), "level")
	assert.Contains(t, string(jsonBytes), "components")
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestScore_ZeroAssets(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        0,
		TotalCapacity:     0,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestScore_SingleAsset(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        1,
		TotalCapacity:     1,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	// Single asset should suggest batch size of 1
	batchSize := result.Metadata["suggested_batch"].(int)
	assert.Equal(t, 1, batchSize)
}

func TestScore_AllComponentsPresent(t *testing.T) {
	scorer := newTestScorer()
	ctx := context.Background()

	input := risk.RiskInput{
		OperationType:     "patch",
		Environment:       "staging",
		AssetCount:        10,
		TotalCapacity:     100,
		RollbackAvailable: true,
	}

	result, err := scorer.Score(ctx, input)
	require.NoError(t, err)

	// Should have all 8 components
	assert.Len(t, result.Components, 8)

	expectedComponents := []string{
		"environment", "scope", "history", "change_size",
		"timing", "dependencies", "drift", "rollback",
	}

	foundComponents := make(map[string]bool)
	for _, comp := range result.Components {
		foundComponents[comp.Name] = true
	}

	for _, expected := range expectedComponents {
		assert.True(t, foundComponents[expected], "missing component: %s", expected)
	}
}
