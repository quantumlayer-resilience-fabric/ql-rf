// Package validation provides the validation pipeline for AI outputs.
package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// Pipeline is the validation pipeline that checks AI outputs.
type Pipeline struct {
	opaClient   *OPAClient
	schemas     map[string]interface{}
	enabled     bool
	log         *logger.Logger
}

// NewPipeline creates a new validation pipeline.
func NewPipeline(cfg config.OPAConfig, log *logger.Logger) (*Pipeline, error) {
	var opaClient *OPAClient
	var err error

	if cfg.Enabled {
		opaClient, err = NewOPAClient(cfg, log)
		if err != nil {
			return nil, fmt.Errorf("failed to create OPA client: %w", err)
		}
	}

	return &Pipeline{
		opaClient: opaClient,
		schemas:   make(map[string]interface{}),
		enabled:   cfg.Enabled,
		log:       log.WithComponent("validation-pipeline"),
	}, nil
}

// ValidationResult contains the result of validation.
type ValidationResult struct {
	Valid        bool              `json:"valid"`
	Errors       []ValidationError `json:"errors,omitempty"`
	Warnings     []string          `json:"warnings,omitempty"`
	PolicyPath   string            `json:"policy_path,omitempty"`
	ValidatedAt  time.Time         `json:"validated_at"`
	QualityScore *QualityScore     `json:"quality_score,omitempty"`
}

// QualityScore provides a 0-100 confidence score for AI-generated artifacts.
// This score determines what automation level is allowed:
//   - Dev: score >= 40
//   - Staging: score >= 60
//   - Prod (canary): score >= 80
//   - Prod (bulk): score >= 90 + human approval
type QualityScore struct {
	Total              int                    `json:"total"`               // 0-100 aggregate score
	Structural         int                    `json:"structural"`          // 0-20: schema/syntax validity
	PolicyCompliance   int                    `json:"policy_compliance"`   // 0-20: OPA/security policy checks
	TestCoverage       int                    `json:"test_coverage"`       // 0-20: test pass rate
	OperationalHistory int                    `json:"operational_history"` // 0-20: past success rate
	HumanReview        int                    `json:"human_review"`        // 0-20: explicit approvals
	Dimensions         map[string]ScoreDimension `json:"dimensions,omitempty"`
	AllowedEnvironments []string              `json:"allowed_environments"`
	RequiresApproval   bool                   `json:"requires_approval"`
	ComputedAt         time.Time              `json:"computed_at"`
}

// ScoreDimension provides detail on a single scoring dimension.
type ScoreDimension struct {
	Score       int      `json:"score"`       // 0-20
	MaxScore    int      `json:"max_score"`   // Usually 20
	Passed      []string `json:"passed"`      // Checks that passed
	Failed      []string `json:"failed"`      // Checks that failed
	Description string   `json:"description"`
}

// EnvironmentThresholds defines minimum quality scores per environment.
var EnvironmentThresholds = map[string]int{
	"development": 40,
	"staging":     60,
	"production":  80,
	"production_bulk": 90,
}

// ValidationError represents a single validation error.
type ValidationError struct {
	Code     string                 `json:"code"`
	Message  string                 `json:"message"`
	Path     string                 `json:"path,omitempty"`
	Severity string                 `json:"severity"` // error, warning
	Context  map[string]interface{} `json:"context,omitempty"`
}

// ValidationRequest contains the input for validation.
type ValidationRequest struct {
	Data        interface{}            `json:"data"`
	Schema      string                 `json:"schema,omitempty"`
	Policies    []string               `json:"policies,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Environment string                 `json:"environment"`
}

// Validate runs the full validation pipeline on input data.
func (p *Pipeline) Validate(ctx context.Context, req *ValidationRequest) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:       true,
		ValidatedAt: time.Now(),
	}

	// Skip validation if disabled
	if !p.enabled {
		p.log.Debug("validation disabled, skipping")
		return result, nil
	}

	// Step 1: Schema validation (if schema specified)
	if req.Schema != "" {
		if err := p.validateSchema(req.Data, req.Schema); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:     "SCHEMA_INVALID",
				Message:  err.Error(),
				Severity: "error",
			})
		}
	}

	// Step 2: OPA policy validation
	if len(req.Policies) > 0 && p.opaClient != nil {
		policyResult, err := p.validatePolicies(ctx, req)
		if err != nil {
			p.log.Error("policy validation failed", "error", err)
			result.Errors = append(result.Errors, ValidationError{
				Code:     "POLICY_ERROR",
				Message:  err.Error(),
				Severity: "error",
			})
			result.Valid = false
		} else {
			result.Errors = append(result.Errors, policyResult.Errors...)
			result.Warnings = append(result.Warnings, policyResult.Warnings...)
			if !policyResult.Valid {
				result.Valid = false
			}
		}
	}

	// Step 3: Safety checks
	safetyResult := p.runSafetyChecks(req)
	result.Errors = append(result.Errors, safetyResult.Errors...)
	result.Warnings = append(result.Warnings, safetyResult.Warnings...)
	if !safetyResult.Valid {
		result.Valid = false
	}

	p.log.Info("validation complete",
		"valid", result.Valid,
		"error_count", len(result.Errors),
		"warning_count", len(result.Warnings),
	)

	return result, nil
}

// ValidatePlan validates an AI-generated plan.
func (p *Pipeline) ValidatePlan(ctx context.Context, plan interface{}, env string) (*ValidationResult, error) {
	policies := []string{"plan_safety"}
	if env == "production" {
		policies = append(policies, "production_safety")
	}

	return p.Validate(ctx, &ValidationRequest{
		Data:        plan,
		Policies:    policies,
		Environment: env,
	})
}

// ValidateToolInvocation validates a tool invocation request.
func (p *Pipeline) ValidateToolInvocation(ctx context.Context, toolName string, params map[string]interface{}, context ToolContext) (*ValidationResult, error) {
	input := map[string]interface{}{
		"tool": map[string]interface{}{
			"name":       toolName,
			"parameters": params,
			"risk":       context.ToolRisk,
			"scope":      context.ToolScope,
		},
		"autonomy": map[string]interface{}{
			"mode": context.AutonomyMode,
		},
		"approval": map[string]interface{}{
			"status": context.ApprovalStatus,
		},
		"simulation_completed": context.SimulationCompleted,
		"user": map[string]interface{}{
			"can_modify_org": context.CanModifyOrg,
		},
		"org": map[string]interface{}{
			"tokens_used_this_month": context.TokensUsed,
			"monthly_token_budget":   context.TokenBudget,
		},
	}

	return p.Validate(ctx, &ValidationRequest{
		Data:        input,
		Policies:    []string{"tool_authorization"},
		Environment: context.Environment,
	})
}

// ToolContext provides context for tool validation.
type ToolContext struct {
	ToolRisk            string `json:"tool_risk"`
	ToolScope           string `json:"tool_scope"`
	AutonomyMode        string `json:"autonomy_mode"`
	ApprovalStatus      string `json:"approval_status"`
	SimulationCompleted bool   `json:"simulation_completed"`
	CanModifyOrg        bool   `json:"can_modify_org"`
	TokensUsed          int    `json:"tokens_used"`
	TokenBudget         int    `json:"token_budget"`
	Environment         string `json:"environment"`
}

// validateSchema validates data against a JSON schema.
func (p *Pipeline) validateSchema(data interface{}, schemaName string) error {
	// TODO: Implement actual JSON Schema validation
	// For now, just check that schema exists
	if _, ok := p.schemas[schemaName]; !ok {
		p.log.Debug("schema not found, skipping validation", "schema", schemaName)
	}
	return nil
}

// validatePolicies validates data against OPA policies.
func (p *Pipeline) validatePolicies(ctx context.Context, req *ValidationRequest) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}

	for _, policy := range req.Policies {
		policyResult, err := p.opaClient.Query(ctx, policy, req.Data, req.Context)
		if err != nil {
			return nil, fmt.Errorf("policy %s query failed: %w", policy, err)
		}

		// Check for denials
		if denials, ok := policyResult["deny"].([]interface{}); ok && len(denials) > 0 {
			for _, d := range denials {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Code:     "POLICY_DENIED",
					Message:  fmt.Sprintf("%v", d),
					Path:     policy,
					Severity: "error",
				})
			}
		}

		// Check for warnings
		if warnings, ok := policyResult["warn"].([]interface{}); ok {
			for _, w := range warnings {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%v", w))
			}
		}
	}

	return result, nil
}

// runSafetyChecks performs built-in safety checks.
func (p *Pipeline) runSafetyChecks(req *ValidationRequest) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Convert data to JSON for inspection
	jsonData, err := json.Marshal(req.Data)
	if err != nil {
		return result
	}
	dataStr := string(jsonData)

	// Check for dangerous patterns
	dangerousPatterns := []struct {
		pattern string
		message string
	}{
		{"rm -rf /", "Detected dangerous recursive delete command"},
		{"DROP DATABASE", "Detected DROP DATABASE command"},
		{"TRUNCATE TABLE", "Detected TRUNCATE TABLE command"},
		{"format C:", "Detected disk format command"},
		{":(){ :|:& };:", "Detected fork bomb pattern"},
		{"dd if=/dev/zero", "Detected disk wipe command"},
	}

	for _, dp := range dangerousPatterns {
		if strings.Contains(dataStr, dp.pattern) {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:     "DANGEROUS_PATTERN",
				Message:  dp.message,
				Severity: "error",
			})
		}
	}

	// Environment-specific checks
	if req.Environment == "production" {
		// Check for required production safeguards
		if !strings.Contains(dataStr, "canary") && !strings.Contains(dataStr, "Canary") {
			result.Warnings = append(result.Warnings, "Production changes should include canary deployment")
		}
	}

	return result
}

// =============================================================================
// OPA Client
// =============================================================================

// OPAClient communicates with the OPA server.
type OPAClient struct {
	url        string
	policyPath string
	client     *http.Client
	log        *logger.Logger
}

// NewOPAClient creates a new OPA client.
func NewOPAClient(cfg config.OPAConfig, log *logger.Logger) (*OPAClient, error) {
	return &OPAClient{
		url:        cfg.URL,
		policyPath: cfg.PoliciesDir,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		log: log.WithComponent("opa-client"),
	}, nil
}

// Query executes an OPA policy query.
func (c *OPAClient) Query(ctx context.Context, policy string, data interface{}, context map[string]interface{}) (map[string]interface{}, error) {
	// Build the query input
	input := map[string]interface{}{
		"input": data,
	}

	// Add context if provided
	if context != nil {
		for k, v := range context {
			input[k] = v
		}
	}

	// Marshal the input
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	// Build the OPA query URL
	// Policy path: ql.ai.{policy} -> /v1/data/ql/ai/{policy}
	policyParts := strings.Split(policy, "_")
	queryPath := fmt.Sprintf("/v1/data/ql/ai/%s", strings.Join(policyParts, "/"))
	url := fmt.Sprintf("%s%s", c.url, queryPath)

	c.log.Debug("querying OPA",
		"url", url,
		"policy", policy,
	)

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OPA request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OPA returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse the response
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract the result
	if resultData, ok := result["result"].(map[string]interface{}); ok {
		return resultData, nil
	}

	return result, nil
}

// =============================================================================
// Quality Score Computation
// =============================================================================

// ComputeQualityScore calculates a comprehensive quality score for an artifact.
func (p *Pipeline) ComputeQualityScore(ctx context.Context, req *QualityScoreRequest) *QualityScore {
	score := &QualityScore{
		Dimensions:  make(map[string]ScoreDimension),
		ComputedAt:  time.Now(),
	}

	// Dimension 1: Structural Validity (0-20)
	structural := p.computeStructuralScore(req)
	score.Structural = structural.Score
	score.Dimensions["structural"] = structural

	// Dimension 2: Policy Compliance (0-20)
	policy := p.computePolicyScore(ctx, req)
	score.PolicyCompliance = policy.Score
	score.Dimensions["policy"] = policy

	// Dimension 3: Test Coverage (0-20)
	tests := p.computeTestScore(req)
	score.TestCoverage = tests.Score
	score.Dimensions["tests"] = tests

	// Dimension 4: Operational History (0-20)
	history := p.computeHistoryScore(req)
	score.OperationalHistory = history.Score
	score.Dimensions["history"] = history

	// Dimension 5: Human Review (0-20)
	review := p.computeReviewScore(req)
	score.HumanReview = review.Score
	score.Dimensions["review"] = review

	// Calculate total
	score.Total = score.Structural + score.PolicyCompliance + score.TestCoverage +
		score.OperationalHistory + score.HumanReview

	// Determine allowed environments based on score
	score.AllowedEnvironments = []string{}
	for env, threshold := range EnvironmentThresholds {
		if score.Total >= threshold {
			if env != "production_bulk" {
				score.AllowedEnvironments = append(score.AllowedEnvironments, env)
			}
		}
	}

	// Determine if approval required
	score.RequiresApproval = score.Total < 90 || req.Environment == "production"

	p.log.Info("computed quality score",
		"total", score.Total,
		"structural", score.Structural,
		"policy", score.PolicyCompliance,
		"tests", score.TestCoverage,
		"history", score.OperationalHistory,
		"review", score.HumanReview,
		"allowed_envs", score.AllowedEnvironments,
	)

	return score
}

// QualityScoreRequest contains input for quality score computation.
type QualityScoreRequest struct {
	ArtifactType     string                 `json:"artifact_type"`     // sop, image, terraform, dr_plan
	ArtifactID       string                 `json:"artifact_id"`
	ArtifactVersion  string                 `json:"artifact_version"`
	Data             interface{}            `json:"data"`
	Schema           string                 `json:"schema"`
	Environment      string                 `json:"environment"`
	ValidationResult *ValidationResult      `json:"validation_result"`
	TestResults      []TestResult           `json:"test_results"`
	HistoryStats     *HistoryStats          `json:"history_stats"`
	HumanApprovals   []HumanApproval        `json:"human_approvals"`
}

// TestResult represents a test execution result.
type TestResult struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // unit, integration, simulation, drill
	Passed   bool   `json:"passed"`
	Duration string `json:"duration"`
	Coverage float64 `json:"coverage,omitempty"`
}

// HistoryStats contains operational history for an artifact family.
type HistoryStats struct {
	TotalExecutions   int     `json:"total_executions"`
	SuccessfulRuns    int     `json:"successful_runs"`
	FailedRuns        int     `json:"failed_runs"`
	RollbackCount     int     `json:"rollback_count"`
	AvgExecutionTime  string  `json:"avg_execution_time"`
	LastFailure       *time.Time `json:"last_failure,omitempty"`
	SuccessRate       float64 `json:"success_rate"`
}

// HumanApproval represents a human review/approval.
type HumanApproval struct {
	ApproverID   string    `json:"approver_id"`
	ApproverRole string    `json:"approver_role"` // engineer, lead, manager
	ApprovedAt   time.Time `json:"approved_at"`
	Scope        string    `json:"scope"` // artifact, version, family
	Notes        string    `json:"notes,omitempty"`
}

func (p *Pipeline) computeStructuralScore(req *QualityScoreRequest) ScoreDimension {
	dim := ScoreDimension{
		MaxScore:    20,
		Description: "Schema and syntax validity",
	}

	passed := []string{}
	failed := []string{}

	// Check 1: Valid JSON/YAML structure (+5)
	if req.Data != nil {
		passed = append(passed, "valid_structure")
		dim.Score += 5
	} else {
		failed = append(failed, "valid_structure")
	}

	// Check 2: Schema validation passed (+5)
	if req.ValidationResult != nil && req.ValidationResult.Valid {
		passed = append(passed, "schema_valid")
		dim.Score += 5
	} else if req.ValidationResult != nil {
		schemaErrors := 0
		for _, e := range req.ValidationResult.Errors {
			if e.Code == "SCHEMA_INVALID" {
				schemaErrors++
			}
		}
		if schemaErrors == 0 {
			passed = append(passed, "schema_valid")
			dim.Score += 5
		} else {
			failed = append(failed, "schema_valid")
		}
	}

	// Check 3: Required fields present (+5)
	if hasRequiredFields(req.Data, req.ArtifactType) {
		passed = append(passed, "required_fields")
		dim.Score += 5
	} else {
		failed = append(failed, "required_fields")
	}

	// Check 4: No syntax warnings (+5)
	if req.ValidationResult == nil || len(req.ValidationResult.Warnings) == 0 {
		passed = append(passed, "no_warnings")
		dim.Score += 5
	} else {
		failed = append(failed, "no_warnings")
	}

	dim.Passed = passed
	dim.Failed = failed
	return dim
}

func (p *Pipeline) computePolicyScore(ctx context.Context, req *QualityScoreRequest) ScoreDimension {
	dim := ScoreDimension{
		MaxScore:    20,
		Description: "OPA and security policy compliance",
	}

	passed := []string{}
	failed := []string{}

	// Check 1: No policy denials (+8)
	policyErrors := 0
	if req.ValidationResult != nil {
		for _, e := range req.ValidationResult.Errors {
			if e.Code == "POLICY_DENIED" || e.Code == "POLICY_ERROR" {
				policyErrors++
			}
		}
	}
	if policyErrors == 0 {
		passed = append(passed, "no_policy_denials")
		dim.Score += 8
	} else {
		failed = append(failed, "no_policy_denials")
	}

	// Check 2: No dangerous patterns (+6)
	dangerousErrors := 0
	if req.ValidationResult != nil {
		for _, e := range req.ValidationResult.Errors {
			if e.Code == "DANGEROUS_PATTERN" {
				dangerousErrors++
			}
		}
	}
	if dangerousErrors == 0 {
		passed = append(passed, "no_dangerous_patterns")
		dim.Score += 6
	} else {
		failed = append(failed, "no_dangerous_patterns")
	}

	// Check 3: Environment-appropriate (+6)
	if req.Environment != "production" {
		passed = append(passed, "env_appropriate")
		dim.Score += 6
	} else {
		// For production, need extra validation
		if req.ValidationResult != nil && req.ValidationResult.Valid {
			passed = append(passed, "env_appropriate")
			dim.Score += 6
		} else {
			failed = append(failed, "env_appropriate")
		}
	}

	dim.Passed = passed
	dim.Failed = failed
	return dim
}

func (p *Pipeline) computeTestScore(req *QualityScoreRequest) ScoreDimension {
	dim := ScoreDimension{
		MaxScore:    20,
		Description: "Test coverage and pass rate",
	}

	passed := []string{}
	failed := []string{}

	if len(req.TestResults) == 0 {
		// No tests - give minimal score
		dim.Score = 5
		failed = append(failed, "has_tests")
		dim.Failed = failed
		dim.Passed = passed
		return dim
	}

	// Check 1: Has tests (+5)
	passed = append(passed, "has_tests")
	dim.Score += 5

	// Check 2: Test pass rate (+10)
	passedTests := 0
	for _, t := range req.TestResults {
		if t.Passed {
			passedTests++
		}
	}
	passRate := float64(passedTests) / float64(len(req.TestResults))
	if passRate >= 1.0 {
		passed = append(passed, "all_tests_pass")
		dim.Score += 10
	} else if passRate >= 0.9 {
		passed = append(passed, "tests_90_percent")
		dim.Score += 7
	} else if passRate >= 0.7 {
		failed = append(failed, "tests_below_90")
		dim.Score += 4
	} else {
		failed = append(failed, "tests_below_70")
	}

	// Check 3: Coverage (+5)
	avgCoverage := 0.0
	coverageCount := 0
	for _, t := range req.TestResults {
		if t.Coverage > 0 {
			avgCoverage += t.Coverage
			coverageCount++
		}
	}
	if coverageCount > 0 {
		avgCoverage /= float64(coverageCount)
		if avgCoverage >= 80 {
			passed = append(passed, "coverage_80_plus")
			dim.Score += 5
		} else if avgCoverage >= 60 {
			passed = append(passed, "coverage_60_plus")
			dim.Score += 3
		} else {
			failed = append(failed, "coverage_below_60")
		}
	}

	dim.Passed = passed
	dim.Failed = failed
	return dim
}

func (p *Pipeline) computeHistoryScore(req *QualityScoreRequest) ScoreDimension {
	dim := ScoreDimension{
		MaxScore:    20,
		Description: "Operational history and success rate",
	}

	passed := []string{}
	failed := []string{}

	if req.HistoryStats == nil || req.HistoryStats.TotalExecutions == 0 {
		// No history - give minimal score for new artifacts
		dim.Score = 5
		passed = append(passed, "new_artifact")
		dim.Passed = passed
		return dim
	}

	stats := req.HistoryStats

	// Check 1: Has execution history (+5)
	if stats.TotalExecutions >= 5 {
		passed = append(passed, "sufficient_history")
		dim.Score += 5
	} else {
		passed = append(passed, "limited_history")
		dim.Score += 2
	}

	// Check 2: Success rate (+10)
	if stats.SuccessRate >= 99 {
		passed = append(passed, "success_99_plus")
		dim.Score += 10
	} else if stats.SuccessRate >= 95 {
		passed = append(passed, "success_95_plus")
		dim.Score += 8
	} else if stats.SuccessRate >= 90 {
		passed = append(passed, "success_90_plus")
		dim.Score += 5
	} else {
		failed = append(failed, "success_below_90")
	}

	// Check 3: Low rollback rate (+5)
	if stats.TotalExecutions > 0 {
		rollbackRate := float64(stats.RollbackCount) / float64(stats.TotalExecutions)
		if rollbackRate <= 0.01 {
			passed = append(passed, "rollback_1_percent")
			dim.Score += 5
		} else if rollbackRate <= 0.05 {
			passed = append(passed, "rollback_5_percent")
			dim.Score += 3
		} else {
			failed = append(failed, "rollback_above_5_percent")
		}
	}

	dim.Passed = passed
	dim.Failed = failed
	return dim
}

func (p *Pipeline) computeReviewScore(req *QualityScoreRequest) ScoreDimension {
	dim := ScoreDimension{
		MaxScore:    20,
		Description: "Human review and approvals",
	}

	passed := []string{}
	failed := []string{}

	if len(req.HumanApprovals) == 0 {
		// No approvals yet
		dim.Score = 0
		failed = append(failed, "no_approvals")
		dim.Failed = failed
		return dim
	}

	// Check 1: Has at least one approval (+8)
	passed = append(passed, "has_approval")
	dim.Score += 8

	// Check 2: Recent approval (within 30 days) (+6)
	hasRecentApproval := false
	for _, a := range req.HumanApprovals {
		if time.Since(a.ApprovedAt) < 30*24*time.Hour {
			hasRecentApproval = true
			break
		}
	}
	if hasRecentApproval {
		passed = append(passed, "recent_approval")
		dim.Score += 6
	} else {
		failed = append(failed, "stale_approval")
	}

	// Check 3: Senior approval (+6)
	hasSeniorApproval := false
	for _, a := range req.HumanApprovals {
		if a.ApproverRole == "lead" || a.ApproverRole == "manager" || a.ApproverRole == "architect" {
			hasSeniorApproval = true
			break
		}
	}
	if hasSeniorApproval {
		passed = append(passed, "senior_approval")
		dim.Score += 6
	} else {
		failed = append(failed, "no_senior_approval")
	}

	dim.Passed = passed
	dim.Failed = failed
	return dim
}

// IsAllowedForEnvironment checks if a score allows deployment to an environment.
func (s *QualityScore) IsAllowedForEnvironment(env string) bool {
	threshold, ok := EnvironmentThresholds[env]
	if !ok {
		return false
	}
	return s.Total >= threshold
}

// GetGrade returns a letter grade for the quality score.
func (s *QualityScore) GetGrade() string {
	switch {
	case s.Total >= 90:
		return "A"
	case s.Total >= 80:
		return "B"
	case s.Total >= 70:
		return "C"
	case s.Total >= 60:
		return "D"
	default:
		return "F"
	}
}

func hasRequiredFields(data interface{}, artifactType string) bool {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return false
	}

	requiredFields := map[string][]string{
		"sop":       {"id", "steps", "scope"},
		"image":     {"name", "base", "security"},
		"terraform": {"resources", "providers"},
		"dr_plan":   {"sources", "targets", "rto", "rpo"},
	}

	fields, ok := requiredFields[artifactType]
	if !ok {
		return true // Unknown type, assume OK
	}

	for _, field := range fields {
		if _, exists := dataMap[field]; !exists {
			return false
		}
	}
	return true
}

// =============================================================================
// Schema Registry
// =============================================================================

// RegisterSchema adds a schema to the registry.
func (p *Pipeline) RegisterSchema(name string, schema interface{}) {
	p.schemas[name] = schema
	p.log.Debug("registered schema", "name", name)
}

// LoadSchemas loads schemas from the filesystem.
func (p *Pipeline) LoadSchemas(dir string) error {
	// TODO: Implement schema loading from directory
	// For now, register some default schemas
	p.RegisterSchema("drift_remediation_v1", map[string]interface{}{
		"type": "object",
		"required": []string{"summary", "phases"},
	})
	p.RegisterSchema("patch_rollout_v1", map[string]interface{}{
		"type": "object",
		"required": []string{"summary", "patches", "schedule"},
	})
	p.RegisterSchema("compliance_report_v1", map[string]interface{}{
		"type": "object",
		"required": []string{"summary", "controls", "findings"},
	})
	p.RegisterSchema("dr_runbook_v1", map[string]interface{}{
		"type": "object",
		"required": []string{"summary", "steps", "recovery_objectives"},
	})

	return nil
}
