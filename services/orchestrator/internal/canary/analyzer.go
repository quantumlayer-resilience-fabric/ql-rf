// Package canary provides canary analysis capabilities for rollout validation.
package canary

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// AnalysisResult represents the result of a canary analysis.
type AnalysisResult struct {
	Passed      bool                   `json:"passed"`
	Score       float64                `json:"score"`
	Metrics     []MetricResult         `json:"metrics"`
	Duration    time.Duration          `json:"duration"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	Summary     string                 `json:"summary"`
	Errors      []string               `json:"errors,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// MetricResult represents the result of a single metric analysis.
type MetricResult struct {
	Name       string  `json:"name"`
	Query      string  `json:"query,omitempty"`
	Value      float64 `json:"value"`
	Threshold  float64 `json:"threshold"`
	Comparison string  `json:"comparison"`
	Passed     bool    `json:"passed"`
	Error      string  `json:"error,omitempty"`
}

// AnalysisConfig defines the configuration for canary analysis.
type AnalysisConfig struct {
	Metrics  []MetricConfig `json:"metrics"`
	Duration time.Duration  `json:"duration"`
	Provider string         `json:"provider"` // prometheus, cloudwatch, datadog
}

// MetricConfig defines a single metric to analyze.
type MetricConfig struct {
	Name       string  `json:"name"`
	Query      string  `json:"query,omitempty"`
	Threshold  float64 `json:"threshold"`
	Comparison string  `json:"comparison"` // less-than, greater-than, equals
}

// Analyzer performs canary analysis against metrics providers.
type Analyzer struct {
	log         *logger.Logger
	prometheus  *PrometheusClient
	cloudwatch  *CloudWatchClient
	datadog     *DatadogClient
}

// NewAnalyzer creates a new canary analyzer.
func NewAnalyzer(log *logger.Logger) *Analyzer {
	return &Analyzer{
		log: log.WithComponent("canary-analyzer"),
	}
}

// SetPrometheusClient sets the Prometheus client for metrics queries.
func (a *Analyzer) SetPrometheusClient(url string) {
	a.prometheus = &PrometheusClient{
		url:    url,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// SetCloudWatchClient sets the CloudWatch client for metrics queries.
func (a *Analyzer) SetCloudWatchClient(region string) {
	a.cloudwatch = &CloudWatchClient{
		region: region,
	}
}

// SetDatadogClient sets the Datadog client for metrics queries.
func (a *Analyzer) SetDatadogClient(apiKey, appKey string) {
	a.datadog = &DatadogClient{
		apiKey: apiKey,
		appKey: appKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Analyze performs canary analysis based on the provided configuration.
func (a *Analyzer) Analyze(ctx context.Context, config AnalysisConfig) (*AnalysisResult, error) {
	a.log.Info("starting canary analysis",
		"provider", config.Provider,
		"duration", config.Duration,
		"metrics", len(config.Metrics),
	)

	result := &AnalysisResult{
		StartTime: time.Now(),
		Metrics:   make([]MetricResult, 0, len(config.Metrics)),
		Metadata:  make(map[string]interface{}),
	}

	// Wait for analysis duration
	select {
	case <-time.After(config.Duration):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Query each metric
	passedCount := 0
	for _, metric := range config.Metrics {
		metricResult, err := a.queryMetric(ctx, config.Provider, metric)
		if err != nil {
			metricResult = MetricResult{
				Name:       metric.Name,
				Threshold:  metric.Threshold,
				Comparison: metric.Comparison,
				Passed:     false,
				Error:      err.Error(),
			}
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", metric.Name, err.Error()))
		}

		result.Metrics = append(result.Metrics, metricResult)
		if metricResult.Passed {
			passedCount++
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Calculate overall score and pass/fail
	if len(config.Metrics) > 0 {
		result.Score = float64(passedCount) / float64(len(config.Metrics)) * 100
	}
	result.Passed = passedCount == len(config.Metrics)

	// Generate summary
	result.Summary = a.generateSummary(result)

	a.log.Info("canary analysis complete",
		"passed", result.Passed,
		"score", result.Score,
		"duration", result.Duration,
	)

	return result, nil
}

// queryMetric queries a single metric from the configured provider.
func (a *Analyzer) queryMetric(ctx context.Context, provider string, metric MetricConfig) (MetricResult, error) {
	result := MetricResult{
		Name:       metric.Name,
		Query:      metric.Query,
		Threshold:  metric.Threshold,
		Comparison: metric.Comparison,
	}

	var value float64
	var err error

	switch provider {
	case "prometheus":
		if a.prometheus == nil {
			return result, fmt.Errorf("prometheus client not configured")
		}
		value, err = a.prometheus.Query(ctx, metric.Query)
	case "cloudwatch":
		if a.cloudwatch == nil {
			return result, fmt.Errorf("cloudwatch client not configured")
		}
		value, err = a.cloudwatch.Query(ctx, metric.Query)
	case "datadog":
		if a.datadog == nil {
			return result, fmt.Errorf("datadog client not configured")
		}
		value, err = a.datadog.Query(ctx, metric.Query)
	default:
		// Use built-in metric if no provider configured
		value, err = a.queryBuiltInMetric(ctx, metric.Name)
	}

	if err != nil {
		return result, err
	}

	result.Value = value
	result.Passed = a.evaluateThreshold(value, metric.Threshold, metric.Comparison)

	return result, nil
}

// queryBuiltInMetric queries built-in metrics without external provider.
func (a *Analyzer) queryBuiltInMetric(ctx context.Context, name string) (float64, error) {
	// Built-in metrics for simple health checks
	switch strings.ToLower(name) {
	case "error-rate":
		// Simulated - in real implementation, query from service mesh
		return 0.001, nil
	case "latency-p99":
		// Simulated
		return 150.0, nil
	case "cpu-usage":
		return 45.0, nil
	case "memory-usage":
		return 60.0, nil
	default:
		return 0, fmt.Errorf("unknown built-in metric: %s", name)
	}
}

// evaluateThreshold evaluates if a value passes the threshold.
func (a *Analyzer) evaluateThreshold(value, threshold float64, comparison string) bool {
	switch comparison {
	case "less-than":
		return value < threshold
	case "greater-than":
		return value > threshold
	case "equals":
		return value == threshold
	case "less-than-or-equal":
		return value <= threshold
	case "greater-than-or-equal":
		return value >= threshold
	default:
		return value < threshold // Default to less-than
	}
}

// generateSummary generates a human-readable summary of the analysis.
func (a *Analyzer) generateSummary(result *AnalysisResult) string {
	if result.Passed {
		return fmt.Sprintf("Canary analysis PASSED with score %.1f%% (%d/%d metrics passed)",
			result.Score, len(result.Metrics)-len(result.Errors), len(result.Metrics))
	}

	failedMetrics := make([]string, 0)
	for _, m := range result.Metrics {
		if !m.Passed {
			failedMetrics = append(failedMetrics, m.Name)
		}
	}

	return fmt.Sprintf("Canary analysis FAILED with score %.1f%%. Failed metrics: %s",
		result.Score, strings.Join(failedMetrics, ", "))
}

// =============================================================================
// Prometheus Client
// =============================================================================

// PrometheusClient queries Prometheus for metrics.
type PrometheusClient struct {
	url    string
	client *http.Client
}

// Query executes a PromQL query and returns the result.
func (p *PrometheusClient) Query(ctx context.Context, query string) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/query?query=%s", p.url, query)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("prometheus query failed: %d", resp.StatusCode)
	}

	var result struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Value []interface{} `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if len(result.Data.Result) == 0 || len(result.Data.Result[0].Value) < 2 {
		return 0, fmt.Errorf("no data returned from prometheus")
	}

	// Value is in format [timestamp, "value"]
	valueStr, ok := result.Data.Result[0].Value[1].(string)
	if !ok {
		return 0, fmt.Errorf("unexpected value format from prometheus")
	}

	var value float64
	_, err = fmt.Sscanf(valueStr, "%f", &value)
	return value, err
}

// =============================================================================
// CloudWatch Client
// =============================================================================

// CloudWatchClient queries AWS CloudWatch for metrics.
type CloudWatchClient struct {
	region string
}

// Query executes a CloudWatch metrics query.
func (c *CloudWatchClient) Query(ctx context.Context, query string) (float64, error) {
	// In real implementation, use AWS SDK
	// This is a placeholder
	return 0, fmt.Errorf("cloudwatch query not implemented")
}

// =============================================================================
// Datadog Client
// =============================================================================

// DatadogClient queries Datadog for metrics.
type DatadogClient struct {
	apiKey string
	appKey string
	client *http.Client
}

// Query executes a Datadog metrics query.
func (d *DatadogClient) Query(ctx context.Context, query string) (float64, error) {
	// In real implementation, use Datadog API
	// This is a placeholder
	return 0, fmt.Errorf("datadog query not implemented")
}

// =============================================================================
// Analysis Templates
// =============================================================================

// PredefinedAnalysis contains predefined analysis configurations.
var PredefinedAnalysis = map[string]AnalysisConfig{
	"basic": {
		Duration: 5 * time.Minute,
		Provider: "prometheus",
		Metrics: []MetricConfig{
			{Name: "error-rate", Query: `sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))`, Threshold: 0.01, Comparison: "less-than"},
		},
	},
	"standard": {
		Duration: 10 * time.Minute,
		Provider: "prometheus",
		Metrics: []MetricConfig{
			{Name: "error-rate", Query: `sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))`, Threshold: 0.01, Comparison: "less-than"},
			{Name: "latency-p99", Query: `histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])) * 1000`, Threshold: 500, Comparison: "less-than"},
		},
	},
	"comprehensive": {
		Duration: 30 * time.Minute,
		Provider: "prometheus",
		Metrics: []MetricConfig{
			{Name: "error-rate", Query: `sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))`, Threshold: 0.005, Comparison: "less-than"},
			{Name: "latency-p99", Query: `histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])) * 1000`, Threshold: 300, Comparison: "less-than"},
			{Name: "latency-p50", Query: `histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m])) * 1000`, Threshold: 100, Comparison: "less-than"},
			{Name: "cpu-usage", Query: `avg(rate(container_cpu_usage_seconds_total[5m])) * 100`, Threshold: 80, Comparison: "less-than"},
			{Name: "memory-usage", Query: `avg(container_memory_usage_bytes / container_memory_limit_bytes) * 100`, Threshold: 85, Comparison: "less-than"},
		},
	},
}
