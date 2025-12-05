// Package inspec provides InSpec integration for automated compliance assessment.
// Supports running InSpec profiles against assets and mapping results to compliance controls.
package inspec

import (
	"time"

	"github.com/google/uuid"
)

// Profile represents an InSpec profile that can be run against assets.
type Profile struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Version     string    `json:"version" db:"version"`
	Title       string    `json:"title" db:"title"`
	Maintainer  string    `json:"maintainer" db:"maintainer"`
	Summary     string    `json:"summary" db:"summary"`
	FrameworkID uuid.UUID `json:"framework_id" db:"framework_id"`
	ProfileURL  string    `json:"profile_url" db:"profile_url"` // Git URL or supermarket URL
	Platforms   []string  `json:"platforms" db:"platforms"`     // linux, windows, aws, azure, etc.
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// RunStatus represents the status of an InSpec run.
type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "cancelled"
)

// Run represents an InSpec profile execution run.
type Run struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	OrgID       uuid.UUID  `json:"org_id" db:"org_id"`
	AssetID     uuid.UUID  `json:"asset_id" db:"asset_id"`
	ProfileID   uuid.UUID  `json:"profile_id" db:"profile_id"`
	Status      RunStatus  `json:"status" db:"status"`
	StartedAt   *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	Duration    int        `json:"duration,omitempty" db:"duration"` // Duration in seconds
	TotalTests  int        `json:"total_tests" db:"total_tests"`
	PassedTests int        `json:"passed_tests" db:"passed_tests"`
	FailedTests int        `json:"failed_tests" db:"failed_tests"`
	SkippedTests int       `json:"skipped_tests" db:"skipped_tests"`
	ErrorMessage string    `json:"error_message,omitempty" db:"error_message"`
	RawOutput   string     `json:"raw_output,omitempty" db:"raw_output"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// ResultStatus represents the status of a single control result.
type ResultStatus string

const (
	ResultStatusPassed  ResultStatus = "passed"
	ResultStatusFailed  ResultStatus = "failed"
	ResultStatusSkipped ResultStatus = "skipped"
	ResultStatusError   ResultStatus = "error"
)

// Result represents the result of a single InSpec control from a run.
type Result struct {
	ID              uuid.UUID    `json:"id" db:"id"`
	RunID           uuid.UUID    `json:"run_id" db:"run_id"`
	ControlID       string       `json:"control_id" db:"control_id"` // InSpec control ID
	ControlTitle    string       `json:"control_title" db:"control_title"`
	Status          ResultStatus `json:"status" db:"status"`
	Message         string       `json:"message,omitempty" db:"message"`
	Resource        string       `json:"resource,omitempty" db:"resource"` // Resource being tested
	SourceLocation  string       `json:"source_location,omitempty" db:"source_location"`
	RunTime         float64      `json:"run_time" db:"run_time"` // Time in seconds
	CodeDescription string       `json:"code_description,omitempty" db:"code_description"`
	CreatedAt       time.Time    `json:"created_at" db:"created_at"`
}

// ControlMapping represents a mapping between an InSpec control and a compliance control.
type ControlMapping struct {
	ID                   uuid.UUID `json:"id" db:"id"`
	InSpecControlID      string    `json:"inspec_control_id" db:"inspec_control_id"`
	ComplianceControlID  uuid.UUID `json:"compliance_control_id" db:"compliance_control_id"`
	ProfileID            uuid.UUID `json:"profile_id" db:"profile_id"`
	MappingConfidence    float64   `json:"mapping_confidence" db:"mapping_confidence"` // 0.0 to 1.0
	Notes                string    `json:"notes,omitempty" db:"notes"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time `json:"updated_at" db:"updated_at"`
}

// InSpecProfile represents the full InSpec profile JSON structure.
type InSpecProfile struct {
	Name        string              `json:"name"`
	Version     string              `json:"version"`
	Title       string              `json:"title"`
	Maintainer  string              `json:"maintainer"`
	Summary     string              `json:"summary"`
	License     string              `json:"license"`
	Copyright   string              `json:"copyright"`
	Supports    []map[string]string `json:"supports"`
	Attributes  []Attribute         `json:"attributes"`
	Groups      []Group             `json:"groups"`
	Controls    []Control           `json:"controls"`
	SHA256      string              `json:"sha256"`
}

// Attribute represents an InSpec profile attribute.
type Attribute struct {
	Name    string      `json:"name"`
	Options interface{} `json:"options"`
}

// Group represents an InSpec control group.
type Group struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Controls []string  `json:"controls"`
}

// Control represents an InSpec control definition.
type Control struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Desc        string                 `json:"desc"`
	Descriptions map[string]string     `json:"descriptions"`
	Impact      float64                `json:"impact"`
	Refs        []map[string]string    `json:"refs"`
	Tags        map[string]interface{} `json:"tags"`
	Code        string                 `json:"code"`
	SourceLocation SourceLocation      `json:"source_location"`
}

// SourceLocation represents the source file location of a control.
type SourceLocation struct {
	Ref  string `json:"ref"`
	Line int    `json:"line"`
}

// InSpecResult represents the full InSpec JSON output structure.
type InSpecResult struct {
	Platform   Platform   `json:"platform"`
	Profiles   []ProfileResult `json:"profiles"`
	Statistics Statistics `json:"statistics"`
	Version    string     `json:"version"`
}

// Platform represents the platform information from InSpec output.
type Platform struct {
	Name    string `json:"name"`
	Release string `json:"release"`
	Target  string `json:"target"`
}

// ProfileResult represents a profile's results in InSpec output.
type ProfileResult struct {
	Name           string               `json:"name"`
	Version        string               `json:"version"`
	SHA256         string               `json:"sha256"`
	Title          string               `json:"title"`
	Maintainer     string               `json:"maintainer"`
	Summary        string               `json:"summary"`
	License        string               `json:"license"`
	Copyright      string               `json:"copyright"`
	Controls       []ControlResult      `json:"controls"`
	Groups         []Group              `json:"groups"`
	Attributes     []Attribute          `json:"attributes"`
	Status         string               `json:"status"`
	StatusMessage  string               `json:"status_message"`
}

// ControlResult represents a control's execution result.
type ControlResult struct {
	ID             string                 `json:"id"`
	Title          string                 `json:"title"`
	Desc           string                 `json:"desc"`
	Descriptions   map[string]string      `json:"descriptions"`
	Impact         float64                `json:"impact"`
	Refs           []map[string]string    `json:"refs"`
	Tags           map[string]interface{} `json:"tags"`
	Code           string                 `json:"code"`
	SourceLocation SourceLocation         `json:"source_location"`
	Results        []TestResult           `json:"results"`
}

// TestResult represents an individual test result within a control.
type TestResult struct {
	Status         string  `json:"status"`
	CodeDesc       string  `json:"code_desc"`
	RunTime        float64 `json:"run_time"`
	StartTime      string  `json:"start_time"`
	Message        string  `json:"message,omitempty"`
	Resource       string  `json:"resource,omitempty"`
	SkipMessage    string  `json:"skip_message,omitempty"`
	ExceptionMessage string `json:"exception,omitempty"`
}

// Statistics represents the summary statistics from InSpec output.
type Statistics struct {
	Duration float64 `json:"duration"`
	Controls StatCount `json:"controls"`
}

// StatCount represents count statistics.
type StatCount struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

// AvailableProfile represents a profile available to run.
type AvailableProfile struct {
	ProfileID   uuid.UUID `json:"profile_id"`
	Name        string    `json:"name"`
	Title       string    `json:"title"`
	Version     string    `json:"version"`
	Framework   string    `json:"framework"`
	FrameworkID uuid.UUID `json:"framework_id"`
	Platforms   []string  `json:"platforms"`
	ControlCount int      `json:"control_count"`
}

// RunSummary represents a summary of a run for listing.
type RunSummary struct {
	RunID       uuid.UUID  `json:"run_id"`
	AssetID     uuid.UUID  `json:"asset_id"`
	AssetName   string     `json:"asset_name"`
	ProfileName string     `json:"profile_name"`
	Framework   string     `json:"framework"`
	Status      RunStatus  `json:"status"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Duration    int        `json:"duration,omitempty"`
	PassRate    float64    `json:"pass_rate"`
	TotalTests  int        `json:"total_tests"`
	PassedTests int        `json:"passed_tests"`
	FailedTests int        `json:"failed_tests"`
}

// CreateProfileRequest represents a request to create a new profile.
type CreateProfileRequest struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Title       string    `json:"title"`
	Maintainer  string    `json:"maintainer"`
	Summary     string    `json:"summary"`
	FrameworkID uuid.UUID `json:"framework_id"`
	ProfileURL  string    `json:"profile_url"`
	Platforms   []string  `json:"platforms"`
}

// RunProfileRequest represents a request to run an InSpec profile.
type RunProfileRequest struct {
	ProfileID uuid.UUID `json:"profile_id"`
	AssetID   uuid.UUID `json:"asset_id"`
}

// RunResultsResponse represents the response with run results.
type RunResultsResponse struct {
	Run     Run      `json:"run"`
	Results []Result `json:"results"`
}
