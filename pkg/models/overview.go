package models

import "fmt"

// MetricTrend represents the trend direction for a metric.
type MetricTrend struct {
	Direction string `json:"direction"` // up, down, neutral
	Value     string `json:"value"`     // e.g., "+5%", "-2%"
	Period    string `json:"period"`    // e.g., "vs last 7 days"
}

// MetricWithTrend represents a metric value with its trend.
type MetricWithTrend struct {
	Value int64       `json:"value"`
	Trend MetricTrend `json:"trend"`
}

// FloatMetricWithTrend represents a float metric value with its trend.
type FloatMetricWithTrend struct {
	Value float64     `json:"value"`
	Trend MetricTrend `json:"trend"`
}

// PlatformCount represents asset count by platform.
type PlatformCount struct {
	Platform   Platform `json:"platform"`
	Count      int      `json:"count"`
	Percentage float64  `json:"percentage"`
}

// OverviewMetrics represents the dashboard overview metrics.
type OverviewMetrics struct {
	FleetSize            MetricWithTrend      `json:"fleetSize"`
	DriftScore           FloatMetricWithTrend `json:"driftScore"`
	Compliance           FloatMetricWithTrend `json:"compliance"`
	DRReadiness          FloatMetricWithTrend `json:"drReadiness"`
	PlatformDistribution []PlatformCount      `json:"platformDistribution"`
	Alerts               []AlertCount         `json:"alerts"`
	RecentActivity       []Activity           `json:"recentActivity"`
}

// TrendDirection represents trend direction values.
type TrendDirection string

const (
	TrendDirectionUp      TrendDirection = "up"
	TrendDirectionDown    TrendDirection = "down"
	TrendDirectionNeutral TrendDirection = "neutral"
)

// NewMetricTrend creates a new MetricTrend.
func NewMetricTrend(current, previous float64, period string) MetricTrend {
	if previous == 0 {
		return MetricTrend{
			Direction: string(TrendDirectionNeutral),
			Value:     "0%",
			Period:    period,
		}
	}

	change := ((current - previous) / previous) * 100
	var direction TrendDirection
	var sign string

	if change > 0 {
		direction = TrendDirectionUp
		sign = "+"
	} else if change < 0 {
		direction = TrendDirectionDown
		sign = ""
	} else {
		direction = TrendDirectionNeutral
		sign = ""
	}

	return MetricTrend{
		Direction: string(direction),
		Value:     sign + formatPercent(change),
		Period:    period,
	}
}

// formatPercent formats a float as a percentage string.
func formatPercent(v float64) string {
	if v < 0 {
		v = -v
	}
	if v < 1 {
		return "< 1%"
	}
	return fmt.Sprintf("%.0f%%", v)
}
