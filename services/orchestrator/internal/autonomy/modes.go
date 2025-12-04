// Package autonomy provides configurable automation levels for AI operations.
package autonomy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/risk"
)

// Mode represents the automation level for operations.
type Mode string

const (
	// ModePlanOnly - AI generates plans but never executes. Human approves and executes manually.
	ModePlanOnly Mode = "plan_only"

	// ModeApproveAll - AI generates plans, human must approve every execution.
	ModeApproveAll Mode = "approve_all"

	// ModeCanaryOnly - AI executes canary phases automatically, pauses for human approval before full rollout.
	ModeCanaryOnly Mode = "canary_only"

	// ModeRiskBased - AI decides based on risk score: low risk = auto, high risk = approval.
	ModeRiskBased Mode = "risk_based"

	// ModeFullAuto - AI executes all operations automatically (with safety guardrails).
	ModeFullAuto Mode = "full_auto"
)

// Config defines the autonomy configuration for an organization.
type Config struct {
	// Mode is the primary autonomy mode
	Mode Mode `json:"mode"`

	// RiskThreshold is the maximum risk score for auto-execution (for risk_based mode)
	RiskThreshold float64 `json:"risk_threshold"`

	// CanaryPercentage is the initial canary percentage for canary_only mode
	CanaryPercentage int `json:"canary_percentage"`

	// RequireApprovalFor lists operation types that always require approval
	RequireApprovalFor []string `json:"require_approval_for"`

	// AutoApproveFor lists environments where auto-approval is allowed
	AutoApproveFor []string `json:"auto_approve_for"`

	// MaxAssetsPerExecution limits the number of assets in a single auto-execution
	MaxAssetsPerExecution int `json:"max_assets_per_execution"`

	// MaxCriticalAssets limits critical assets in auto-execution
	MaxCriticalAssets int `json:"max_critical_assets"`

	// CooldownPeriod is the minimum time between auto-executions
	CooldownPeriod time.Duration `json:"cooldown_period"`

	// RequireCanarySuccess requires canary to pass before full rollout
	RequireCanarySuccess bool `json:"require_canary_success"`

	// NotifyOnAutoExecution sends notifications when auto-executing
	NotifyOnAutoExecution bool `json:"notify_on_auto_execution"`

	// AllowRollback enables automatic rollback on failure
	AllowRollback bool `json:"allow_rollback"`

	// AllowedPlatforms limits which platforms can be auto-executed
	AllowedPlatforms []string `json:"allowed_platforms"`

	// BlockedTimeWindows defines times when auto-execution is blocked
	BlockedTimeWindows []TimeWindow `json:"blocked_time_windows"`
}

// TimeWindow represents a time window where operations are blocked.
type TimeWindow struct {
	Start    string `json:"start"`     // HH:MM format
	End      string `json:"end"`       // HH:MM format
	Days     []string `json:"days"`    // monday, tuesday, etc. Empty = all days
	Timezone string `json:"timezone"` // e.g., "America/New_York"
}

// Decision represents the autonomy decision for an operation.
type Decision struct {
	CanAutoExecute    bool     `json:"can_auto_execute"`
	RequiresApproval  bool     `json:"requires_approval"`
	RequiresCanary    bool     `json:"requires_canary"`
	CanaryPercentage  int      `json:"canary_percentage"`
	MaxBatchSize      int      `json:"max_batch_size"`
	WaitTimeMinutes   int      `json:"wait_time_minutes"`
	Reason            string   `json:"reason"`
	BlockedReasons    []string `json:"blocked_reasons,omitempty"`
	Recommendations   []string `json:"recommendations,omitempty"`
}

// OperationContext contains context for autonomy decisions.
type OperationContext struct {
	OperationType    string           `json:"operation_type"`
	Environment      string           `json:"environment"`
	Platform         string           `json:"platform"`
	AssetCount       int              `json:"asset_count"`
	CriticalAssets   int              `json:"critical_assets"`
	RiskScore        *risk.RiskScore  `json:"risk_score,omitempty"`
	ScheduledTime    time.Time        `json:"scheduled_time"`
	LastExecutionTime *time.Time      `json:"last_execution_time,omitempty"`
}

// Controller manages autonomy decisions based on configuration.
type Controller struct {
	log    *logger.Logger
	config Config
}

// DefaultConfig returns a safe default configuration.
func DefaultConfig() Config {
	return Config{
		Mode:                  ModeApproveAll,
		RiskThreshold:         30,
		CanaryPercentage:      10,
		RequireApprovalFor:    []string{"terminate", "dr-drill"},
		AutoApproveFor:        []string{"development", "staging"},
		MaxAssetsPerExecution: 100,
		MaxCriticalAssets:     5,
		CooldownPeriod:        5 * time.Minute,
		RequireCanarySuccess:  true,
		NotifyOnAutoExecution: true,
		AllowRollback:         true,
		AllowedPlatforms:      []string{"aws", "azure", "gcp", "vsphere", "k8s"},
		BlockedTimeWindows:    []TimeWindow{},
	}
}

// NewController creates a new autonomy controller.
func NewController(log *logger.Logger, config Config) *Controller {
	return &Controller{
		log:    log.WithComponent("autonomy"),
		config: config,
	}
}

// SetConfig updates the autonomy configuration.
func (c *Controller) SetConfig(config Config) {
	c.config = config
}

// GetConfig returns the current configuration.
func (c *Controller) GetConfig() Config {
	return c.config
}

// Decide determines what autonomy level is appropriate for an operation.
func (c *Controller) Decide(ctx context.Context, opCtx OperationContext) (*Decision, error) {
	c.log.Info("evaluating autonomy decision",
		"mode", c.config.Mode,
		"operation", opCtx.OperationType,
		"environment", opCtx.Environment,
		"assets", opCtx.AssetCount,
	)

	decision := &Decision{
		CanAutoExecute:   false,
		RequiresApproval: true,
		RequiresCanary:   false,
		CanaryPercentage: c.config.CanaryPercentage,
		MaxBatchSize:     c.config.MaxAssetsPerExecution,
		WaitTimeMinutes:  5,
		BlockedReasons:   []string{},
		Recommendations:  []string{},
	}

	// Check blockers first
	blockers := c.checkBlockers(opCtx)
	if len(blockers) > 0 {
		decision.BlockedReasons = blockers
		decision.CanAutoExecute = false
		decision.RequiresApproval = true
		decision.Reason = fmt.Sprintf("Blocked: %s", blockers[0])
		return decision, nil
	}

	// Make decision based on mode
	switch c.config.Mode {
	case ModePlanOnly:
		return c.decidePlanOnly(decision, opCtx)
	case ModeApproveAll:
		return c.decideApproveAll(decision, opCtx)
	case ModeCanaryOnly:
		return c.decideCanaryOnly(decision, opCtx)
	case ModeRiskBased:
		return c.decideRiskBased(decision, opCtx)
	case ModeFullAuto:
		return c.decideFullAuto(decision, opCtx)
	default:
		decision.Reason = "Unknown mode - defaulting to approval required"
		return decision, nil
	}
}

// checkBlockers identifies any conditions that block auto-execution.
func (c *Controller) checkBlockers(opCtx OperationContext) []string {
	blockers := []string{}

	// Check if operation type requires approval
	for _, blocked := range c.config.RequireApprovalFor {
		if opCtx.OperationType == blocked {
			blockers = append(blockers, fmt.Sprintf("Operation '%s' always requires approval", blocked))
		}
	}

	// Check asset limits
	if opCtx.AssetCount > c.config.MaxAssetsPerExecution {
		blockers = append(blockers, fmt.Sprintf("Asset count (%d) exceeds limit (%d)",
			opCtx.AssetCount, c.config.MaxAssetsPerExecution))
	}

	// Check critical asset limits
	if opCtx.CriticalAssets > c.config.MaxCriticalAssets {
		blockers = append(blockers, fmt.Sprintf("Critical assets (%d) exceed limit (%d)",
			opCtx.CriticalAssets, c.config.MaxCriticalAssets))
	}

	// Check platform allowlist
	if len(c.config.AllowedPlatforms) > 0 {
		allowed := false
		for _, p := range c.config.AllowedPlatforms {
			if p == opCtx.Platform {
				allowed = true
				break
			}
		}
		if !allowed {
			blockers = append(blockers, fmt.Sprintf("Platform '%s' not in allowed list", opCtx.Platform))
		}
	}

	// Check cooldown period
	if opCtx.LastExecutionTime != nil {
		timeSinceLastExec := time.Since(*opCtx.LastExecutionTime)
		if timeSinceLastExec < c.config.CooldownPeriod {
			blockers = append(blockers, fmt.Sprintf("Cooldown period not met (%.0f minutes remaining)",
				(c.config.CooldownPeriod - timeSinceLastExec).Minutes()))
		}
	}

	// Check blocked time windows
	if c.isInBlockedWindow(opCtx.ScheduledTime) {
		blockers = append(blockers, "Currently in blocked time window")
	}

	return blockers
}

// isInBlockedWindow checks if a time falls within a blocked window.
func (c *Controller) isInBlockedWindow(t time.Time) bool {
	for _, window := range c.config.BlockedTimeWindows {
		if c.timeInWindow(t, window) {
			return true
		}
	}
	return false
}

// timeInWindow checks if a time falls within a specific window.
func (c *Controller) timeInWindow(t time.Time, window TimeWindow) bool {
	// Parse timezone
	loc := time.UTC
	if window.Timezone != "" {
		if parsed, err := time.LoadLocation(window.Timezone); err == nil {
			loc = parsed
		}
	}

	localTime := t.In(loc)

	// Check day of week
	if len(window.Days) > 0 {
		dayName := fmt.Sprintf("%s", localTime.Weekday())
		found := false
		for _, d := range window.Days {
			if d == dayName {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Parse start and end times
	startHour, startMin := 0, 0
	fmt.Sscanf(window.Start, "%d:%d", &startHour, &startMin)
	endHour, endMin := 23, 59
	fmt.Sscanf(window.End, "%d:%d", &endHour, &endMin)

	currentMinutes := localTime.Hour()*60 + localTime.Minute()
	startMinutes := startHour*60 + startMin
	endMinutes := endHour*60 + endMin

	return currentMinutes >= startMinutes && currentMinutes <= endMinutes
}

// decidePlanOnly always requires manual execution.
func (c *Controller) decidePlanOnly(d *Decision, opCtx OperationContext) (*Decision, error) {
	d.CanAutoExecute = false
	d.RequiresApproval = true
	d.Reason = "Plan-only mode - manual execution required"
	d.Recommendations = append(d.Recommendations, "Review plan and execute manually")
	return d, nil
}

// decideApproveAll requires approval for everything.
func (c *Controller) decideApproveAll(d *Decision, opCtx OperationContext) (*Decision, error) {
	d.CanAutoExecute = false
	d.RequiresApproval = true
	d.Reason = "Approve-all mode - human approval required"

	// Check if environment allows auto-approval
	for _, env := range c.config.AutoApproveFor {
		if env == opCtx.Environment {
			d.Recommendations = append(d.Recommendations,
				fmt.Sprintf("Environment '%s' is configured for auto-approval in higher modes", env))
		}
	}

	return d, nil
}

// decideCanaryOnly auto-executes canary, requires approval for full rollout.
func (c *Controller) decideCanaryOnly(d *Decision, opCtx OperationContext) (*Decision, error) {
	d.RequiresCanary = true
	d.CanaryPercentage = c.config.CanaryPercentage

	// Canary phase can auto-execute
	d.CanAutoExecute = true
	d.RequiresApproval = false // Canary auto-executes
	d.Reason = fmt.Sprintf("Canary-only mode - auto-executing %d%% canary, approval needed for full rollout",
		c.config.CanaryPercentage)
	d.Recommendations = append(d.Recommendations,
		"Canary will execute automatically",
		"Monitor canary metrics before approving full rollout",
	)

	// Calculate canary batch size
	canarySize := (opCtx.AssetCount * c.config.CanaryPercentage) / 100
	if canarySize < 1 {
		canarySize = 1
	}
	d.MaxBatchSize = canarySize

	return d, nil
}

// decideRiskBased makes decisions based on risk score.
func (c *Controller) decideRiskBased(d *Decision, opCtx OperationContext) (*Decision, error) {
	// Need risk score for this mode
	if opCtx.RiskScore == nil {
		d.Reason = "Risk-based mode requires risk assessment - defaulting to approval required"
		d.Recommendations = append(d.Recommendations, "Run risk assessment before execution")
		return d, nil
	}

	score := opCtx.RiskScore.OverallScore

	if score <= c.config.RiskThreshold {
		// Low risk - can auto-execute
		d.CanAutoExecute = true
		d.RequiresApproval = false
		d.Reason = fmt.Sprintf("Risk-based mode - score %.1f below threshold %.1f, auto-execution allowed",
			score, c.config.RiskThreshold)

		// Still recommend canary for production
		if opCtx.Environment == "production" {
			d.RequiresCanary = true
			d.Recommendations = append(d.Recommendations, "Using canary deployment for production")
		}
	} else {
		// High risk - require approval
		d.CanAutoExecute = false
		d.RequiresApproval = true
		d.RequiresCanary = true
		d.Reason = fmt.Sprintf("Risk-based mode - score %.1f exceeds threshold %.1f, approval required",
			score, c.config.RiskThreshold)
		d.Recommendations = append(d.Recommendations,
			"High risk score detected",
			"Review risk factors before approval",
		)
	}

	// Use risk score recommendations for batch size and wait time
	if metadata, ok := opCtx.RiskScore.Metadata["suggested_batch"].(int); ok {
		d.MaxBatchSize = metadata
	}
	if metadata, ok := opCtx.RiskScore.Metadata["suggested_wait"].(string); ok {
		if parsed, err := time.ParseDuration(metadata); err == nil {
			d.WaitTimeMinutes = int(parsed.Minutes())
		}
	}

	return d, nil
}

// decideFullAuto allows automatic execution with safety guardrails.
func (c *Controller) decideFullAuto(d *Decision, opCtx OperationContext) (*Decision, error) {
	d.CanAutoExecute = true
	d.RequiresApproval = false
	d.Reason = "Full-auto mode - automatic execution with safety guardrails"

	// Production still gets canary
	if opCtx.Environment == "production" {
		d.RequiresCanary = true
		d.CanaryPercentage = c.config.CanaryPercentage
		d.Recommendations = append(d.Recommendations, "Using canary deployment for production safety")
	}

	// Apply risk-based adjustments if available
	if opCtx.RiskScore != nil {
		if opCtx.RiskScore.Level == risk.RiskLevelCritical {
			// Critical risk still requires approval even in full auto
			d.CanAutoExecute = false
			d.RequiresApproval = true
			d.Reason = "Full-auto mode - CRITICAL risk detected, approval required"
			d.Recommendations = append(d.Recommendations, "Critical risk level overrides full-auto mode")
		}
	}

	return d, nil
}

// Validate checks if a configuration is valid.
func (c Config) Validate() error {
	// Check mode is valid
	validModes := map[Mode]bool{
		ModePlanOnly:   true,
		ModeApproveAll: true,
		ModeCanaryOnly: true,
		ModeRiskBased:  true,
		ModeFullAuto:   true,
	}
	if !validModes[c.Mode] {
		return fmt.Errorf("invalid autonomy mode: %s", c.Mode)
	}

	// Check thresholds
	if c.RiskThreshold < 0 || c.RiskThreshold > 100 {
		return fmt.Errorf("risk_threshold must be between 0 and 100")
	}

	if c.CanaryPercentage < 1 || c.CanaryPercentage > 50 {
		return fmt.Errorf("canary_percentage must be between 1 and 50")
	}

	if c.MaxAssetsPerExecution < 1 {
		return fmt.Errorf("max_assets_per_execution must be at least 1")
	}

	return nil
}

// ToJSON serializes the decision to JSON.
func (d *Decision) ToJSON() ([]byte, error) {
	return json.Marshal(d)
}

// ModeDescriptions returns human-readable descriptions of each mode.
func ModeDescriptions() map[Mode]string {
	return map[Mode]string{
		ModePlanOnly:   "AI generates plans only. No automatic execution. Human must execute manually.",
		ModeApproveAll: "AI generates plans and prepares execution. Human must approve every operation.",
		ModeCanaryOnly: "AI auto-executes canary deployments. Human approves full rollout after canary passes.",
		ModeRiskBased:  "AI decides based on risk score. Low-risk operations auto-execute, high-risk require approval.",
		ModeFullAuto:   "AI executes all operations automatically. Safety guardrails prevent critical failures.",
	}
}
