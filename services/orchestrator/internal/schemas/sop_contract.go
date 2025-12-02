// Package schemas defines structured output schemas for AI-generated artifacts.
package schemas

// SOPSpec represents a Standard Operating Procedure specification.
// This is the canonical format that LLM generates from natural language,
// which defines automated operational workflows.
type SOPSpec struct {
	// Metadata
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	Version     string            `json:"version" yaml:"version"`
	Description string            `json:"description" yaml:"description"`
	Author      string            `json:"author,omitempty" yaml:"author,omitempty"`
	Tags        map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// Scope defines when this SOP applies
	Scope SOPScope `json:"scope" yaml:"scope"`

	// Trigger defines what initiates this SOP
	Trigger SOPTrigger `json:"trigger" yaml:"trigger"`

	// Steps defines the ordered actions to execute
	Steps []SOPStep `json:"steps" yaml:"steps"`

	// Rollback defines how to undo this SOP
	Rollback *SOPRollback `json:"rollback,omitempty" yaml:"rollback,omitempty"`

	// Validation defines success criteria
	Validation SOPValidation `json:"validation" yaml:"validation"`

	// Notifications defines who to notify
	Notifications []SOPNotification `json:"notifications,omitempty" yaml:"notifications,omitempty"`

	// Approval requirements
	Approval SOPApproval `json:"approval" yaml:"approval"`
}

// SOPScope defines the applicability of an SOP.
type SOPScope struct {
	Environments []string `json:"environments" yaml:"environments"`           // production, staging, development
	Platforms    []string `json:"platforms,omitempty" yaml:"platforms,omitempty"` // aws, azure, gcp
	AssetTypes   []string `json:"asset_types,omitempty" yaml:"asset_types,omitempty"` // vm, container, database
	AssetFilter  string   `json:"asset_filter,omitempty" yaml:"asset_filter,omitempty"` // query expression
}

// SOPTrigger defines what initiates an SOP.
type SOPTrigger struct {
	Type       string                 `json:"type" yaml:"type"`             // manual, schedule, event, alert
	Schedule   string                 `json:"schedule,omitempty" yaml:"schedule,omitempty"` // cron expression
	Event      string                 `json:"event,omitempty" yaml:"event,omitempty"`       // event type
	Conditions map[string]interface{} `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// SOPStep defines a single step in the SOP.
type SOPStep struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Action      SOPAction              `json:"action" yaml:"action"`
	Condition   string                 `json:"condition,omitempty" yaml:"condition,omitempty"` // when to execute
	OnFailure   string                 `json:"on_failure,omitempty" yaml:"on_failure,omitempty"` // continue, stop, rollback
	Timeout     string                 `json:"timeout,omitempty" yaml:"timeout,omitempty"` // e.g., "5m", "1h"
	Retries     int                    `json:"retries,omitempty" yaml:"retries,omitempty"`
	DependsOn   []string               `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
}

// SOPAction defines the action to perform in a step.
type SOPAction struct {
	Type       string                 `json:"type" yaml:"type"`   // primitive action type
	Parameters map[string]interface{} `json:"parameters" yaml:"parameters"`
}

// SOPActionType defines allowed action primitives.
type SOPActionType string

const (
	// Query actions (read-only)
	ActionInventoryList     SOPActionType = "inventory.list"
	ActionInventoryQuery    SOPActionType = "inventory.query"
	ActionDriftCheck        SOPActionType = "drift.check"
	ActionComplianceCheck   SOPActionType = "compliance.check"
	ActionHealthCheck       SOPActionType = "health.check"

	// Notification actions
	ActionNotifySlack       SOPActionType = "notify.slack"
	ActionNotifyEmail       SOPActionType = "notify.email"
	ActionNotifyPagerDuty   SOPActionType = "notify.pagerduty"
	ActionCreateTicket      SOPActionType = "change.create_ticket"

	// Rollout actions (require approval)
	ActionRolloutBatch      SOPActionType = "rollout.batch"
	ActionRolloutCanary     SOPActionType = "rollout.canary"
	ActionRolloutBlueGreen  SOPActionType = "rollout.blue_green"
	ActionRolloutPause      SOPActionType = "rollout.pause"
	ActionRolloutResume     SOPActionType = "rollout.resume"
	ActionRolloutAbort      SOPActionType = "rollout.abort"

	// Validation actions
	ActionValidateHealth    SOPActionType = "validate.health"
	ActionValidateMetrics   SOPActionType = "validate.metrics"
	ActionValidateLogs      SOPActionType = "validate.logs"
	ActionValidateCompliance SOPActionType = "validate.compliance"

	// Image actions
	ActionImageBuild        SOPActionType = "image.build"
	ActionImagePromote      SOPActionType = "image.promote"
	ActionImageTest         SOPActionType = "image.test"

	// DR actions
	ActionDRFailover        SOPActionType = "dr.failover"
	ActionDRFailback        SOPActionType = "dr.failback"
	ActionDRDrill           SOPActionType = "dr.drill"

	// Wait actions
	ActionWait              SOPActionType = "wait.duration"
	ActionWaitApproval      SOPActionType = "wait.approval"
	ActionWaitCondition     SOPActionType = "wait.condition"
)

// SOPRollback defines how to undo an SOP.
type SOPRollback struct {
	Strategy string    `json:"strategy" yaml:"strategy"` // auto, manual, none
	Steps    []SOPStep `json:"steps,omitempty" yaml:"steps,omitempty"`
	Timeout  string    `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// SOPValidation defines success criteria for the SOP.
type SOPValidation struct {
	SuccessCriteria []SOPCriterion `json:"success_criteria" yaml:"success_criteria"`
	FailureCriteria []SOPCriterion `json:"failure_criteria,omitempty" yaml:"failure_criteria,omitempty"`
}

// SOPCriterion defines a single validation criterion.
type SOPCriterion struct {
	Name      string `json:"name" yaml:"name"`
	Type      string `json:"type" yaml:"type"`           // metric, log, health, custom
	Condition string `json:"condition" yaml:"condition"` // expression to evaluate
	Weight    int    `json:"weight,omitempty" yaml:"weight,omitempty"` // importance 1-10
}

// SOPNotification defines notification settings.
type SOPNotification struct {
	When     string   `json:"when" yaml:"when"`         // start, step_complete, success, failure
	Channels []string `json:"channels" yaml:"channels"` // slack, email, pagerduty
	Template string   `json:"template,omitempty" yaml:"template,omitempty"`
}

// SOPApproval defines approval requirements.
type SOPApproval struct {
	Required     bool     `json:"required" yaml:"required"`
	Approvers    []string `json:"approvers,omitempty" yaml:"approvers,omitempty"`     // user IDs or roles
	MinApprovers int      `json:"min_approvers,omitempty" yaml:"min_approvers,omitempty"` // minimum approvals needed
	AutoApprove  bool     `json:"auto_approve,omitempty" yaml:"auto_approve,omitempty"` // for low-risk ops
}

// SOPExecutionRecord tracks a single SOP execution.
type SOPExecutionRecord struct {
	ID            string                 `json:"id" yaml:"id"`
	SOPID         string                 `json:"sop_id" yaml:"sop_id"`
	SOPVersion    string                 `json:"sop_version" yaml:"sop_version"`
	Status        string                 `json:"status" yaml:"status"` // pending, running, completed, failed, rolled_back
	StartedAt     string                 `json:"started_at" yaml:"started_at"`
	CompletedAt   string                 `json:"completed_at,omitempty" yaml:"completed_at,omitempty"`
	TriggeredBy   string                 `json:"triggered_by" yaml:"triggered_by"`
	ApprovedBy    []string               `json:"approved_by,omitempty" yaml:"approved_by,omitempty"`
	StepResults   []SOPStepResult        `json:"step_results" yaml:"step_results"`
	AffectedAssets []string              `json:"affected_assets,omitempty" yaml:"affected_assets,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// SOPStepResult tracks the result of a single step execution.
type SOPStepResult struct {
	StepID      string                 `json:"step_id" yaml:"step_id"`
	Status      string                 `json:"status" yaml:"status"` // pending, running, completed, failed, skipped
	StartedAt   string                 `json:"started_at" yaml:"started_at"`
	CompletedAt string                 `json:"completed_at,omitempty" yaml:"completed_at,omitempty"`
	Output      interface{}            `json:"output,omitempty" yaml:"output,omitempty"`
	Error       string                 `json:"error,omitempty" yaml:"error,omitempty"`
	Retries     int                    `json:"retries,omitempty" yaml:"retries,omitempty"`
}
