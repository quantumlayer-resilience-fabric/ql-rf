package finops

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTimeRangeHelpers(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() TimeRange
		validate func(TimeRange) bool
	}{
		{
			name: "NewTimeRangeLast 7 days",
			fn:   func() TimeRange { return NewTimeRangeLast(7) },
			validate: func(tr TimeRange) bool {
				duration := tr.End.Sub(tr.Start)
				days := int(duration.Hours() / 24)
				return days >= 6 && days <= 7 // Allow for rounding
			},
		},
		{
			name: "NewTimeRangeLast 30 days",
			fn:   func() TimeRange { return NewTimeRangeLast(30) },
			validate: func(tr TimeRange) bool {
				duration := tr.End.Sub(tr.Start)
				days := int(duration.Hours() / 24)
				return days >= 29 && days <= 30
			},
		},
		{
			name: "NewTimeRangeThisMonth",
			fn:   NewTimeRangeThisMonth,
			validate: func(tr TimeRange) bool {
				now := time.Now()
				return tr.Start.Year() == now.Year() &&
					tr.Start.Month() == now.Month() &&
					tr.Start.Day() == 1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := tt.fn()
			if !tt.validate(tr) {
				t.Errorf("validation failed for %s", tt.name)
			}
			if tr.Start.After(tr.End) {
				t.Errorf("start time should be before end time")
			}
		})
	}
}

func TestCostRecord(t *testing.T) {
	orgID := uuid.New()
	record := CostRecord{
		OrgID:        orgID,
		ResourceID:   "i-1234567890",
		ResourceType: "ec2_instance",
		ResourceName: "web-server-01",
		Cloud:        "aws",
		Service:      "ec2",
		Region:       "us-east-1",
		Cost:         125.50,
		Currency:     "USD",
		UsageHours:   720,
		Tags: map[string]string{
			"Environment": "production",
		},
		RecordedAt: time.Now(),
	}

	if record.OrgID != orgID {
		t.Errorf("expected org_id %s, got %s", orgID, record.OrgID)
	}

	if record.Cost != 125.50 {
		t.Errorf("expected cost 125.50, got %f", record.Cost)
	}

	if record.Tags["Environment"] != "production" {
		t.Errorf("expected environment tag to be 'production'")
	}
}

func TestCostRecommendation(t *testing.T) {
	orgID := uuid.New()
	rec := CostRecommendation{
		OrgID:            orgID,
		Type:             string(RecommendationRightsizing),
		ResourceID:       "i-1234567890",
		ResourceType:     "ec2_instance",
		Platform:         "aws",
		CurrentCost:      125.50,
		PotentialSavings: 37.65,
		Currency:         "USD",
		Action:           "Downsize instance",
		Priority:         string(PriorityHigh),
		Status:           string(StatusPending),
	}

	if rec.Type != string(RecommendationRightsizing) {
		t.Errorf("expected type 'rightsizing', got %s", rec.Type)
	}

	if rec.PotentialSavings <= 0 {
		t.Errorf("potential savings should be positive")
	}

	savingsPercent := (rec.PotentialSavings / rec.CurrentCost) * 100
	if savingsPercent < 0 || savingsPercent > 100 {
		t.Errorf("savings percent should be between 0 and 100, got %f", savingsPercent)
	}
}

func TestCostBudget(t *testing.T) {
	orgID := uuid.New()
	budget := CostBudget{
		OrgID:          orgID,
		Name:           "Monthly AWS Budget",
		Amount:         5000.00,
		Currency:       "USD",
		Period:         string(PeriodMonthly),
		Scope:          string(ScopeCloud),
		ScopeValue:     "aws",
		AlertThreshold: 80.0,
		StartDate:      time.Now(),
		Active:         true,
		CreatedBy:      "user_123",
	}

	if budget.Amount <= 0 {
		t.Errorf("budget amount should be positive")
	}

	if budget.AlertThreshold < 0 || budget.AlertThreshold > 100 {
		t.Errorf("alert threshold should be between 0 and 100, got %f", budget.AlertThreshold)
	}

	if budget.Period != string(PeriodMonthly) {
		t.Errorf("expected period 'monthly', got %s", budget.Period)
	}
}

func TestCostSummary(t *testing.T) {
	summary := CostSummary{
		OrgID:       uuid.New(),
		TotalCost:   1250.50,
		Currency:    "USD",
		Period:      "monthly",
		StartDate:   time.Now().AddDate(0, -1, 0),
		EndDate:     time.Now(),
		ByCloud: map[string]float64{
			"aws":   750.00,
			"azure": 500.50,
		},
		ByService: map[string]float64{
			"ec2": 450.00,
			"rds": 300.00,
		},
		BySite: map[string]float64{
			"us-east-1": 800.00,
			"eu-west-1": 450.50,
		},
		TrendChange: 5.5,
	}

	// Verify total matches sum of clouds
	cloudTotal := 0.0
	for _, cost := range summary.ByCloud {
		cloudTotal += cost
	}
	if cloudTotal != summary.TotalCost {
		t.Errorf("cloud total %f does not match total cost %f", cloudTotal, summary.TotalCost)
	}

	if summary.TrendChange < -100 || summary.TrendChange > 100 {
		t.Logf("warning: trend change of %f%% seems unusual", summary.TrendChange)
	}
}

func TestResourceCost(t *testing.T) {
	rc := ResourceCost{
		ResourceID:   "i-1234567890",
		ResourceType: "ec2_instance",
		ResourceName: "web-server-01",
		Platform:     "aws",
		Cost:         125.50,
		Currency:     "USD",
		UsageHours:   720,
	}

	if rc.Cost <= 0 {
		t.Errorf("cost should be positive")
	}

	if rc.UsageHours > 744 { // Maximum hours in a month
		t.Errorf("usage hours %f exceeds maximum monthly hours", rc.UsageHours)
	}

	hourlyRate := rc.Cost / rc.UsageHours
	if hourlyRate <= 0 {
		t.Errorf("hourly rate should be positive")
	}
}

func TestDeterminePeriod(t *testing.T) {
	tests := []struct {
		name     string
		days     int
		expected string
	}{
		{"1 day", 1, "daily"},
		{"7 days", 7, "weekly"},
		{"30 days", 30, "monthly"},
		{"90 days", 90, "quarterly"},
		{"365 days", 365, "yearly"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			end := time.Now()
			start := end.AddDate(0, 0, -tt.days)
			tr := TimeRange{Start: start, End: end}

			period := determinePeriod(tr)
			if period != tt.expected {
				t.Errorf("expected period %s, got %s", tt.expected, period)
			}
		})
	}
}
