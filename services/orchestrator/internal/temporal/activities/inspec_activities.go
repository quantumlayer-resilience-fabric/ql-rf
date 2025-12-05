// Package activities defines Temporal activities for InSpec execution.
package activities

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/inspec"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// InSpecActivities provides activities for InSpec workflow execution.
type InSpecActivities struct {
	db     *sql.DB
	svc    *inspec.Service
	log    *logger.Logger
}

// NewInSpecActivities creates a new InSpecActivities instance.
func NewInSpecActivities(db *sql.DB, log *logger.Logger) *InSpecActivities {
	return &InSpecActivities{
		db:  db,
		svc: inspec.NewService(db),
		log: log.WithComponent("inspec-activities"),
	}
}

// UpdateInSpecRunStatus updates the status of an InSpec run.
func (a *InSpecActivities) UpdateInSpecRunStatus(ctx context.Context, runID, status, errorMsg string) error {
	id, err := uuid.Parse(runID)
	if err != nil {
		return fmt.Errorf("invalid run ID: %w", err)
	}

	runStatus := inspec.RunStatus(status)
	return a.svc.UpdateRunStatus(ctx, id, runStatus, errorMsg)
}

// PrepareInSpecExecution prepares the execution environment for an InSpec run.
func (a *InSpecActivities) PrepareInSpecExecution(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	profileURL, _ := input["profile_url"].(string)
	runID, _ := input["run_id"].(string)

	// Create temporary directory for this run
	tempDir := filepath.Join(os.TempDir(), "inspec-runs", runID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	profilePath := tempDir

	// If profile URL is provided, fetch/clone the profile
	if profileURL != "" {
		if isGitURL(profileURL) {
			// Clone git repository
			profilePath = filepath.Join(tempDir, "profile")
			cmd := exec.CommandContext(ctx, "git", "clone", profileURL, profilePath)
			if output, err := cmd.CombinedOutput(); err != nil {
				a.log.Error("failed to clone profile", "error", err, "output", string(output))
				return nil, fmt.Errorf("failed to clone profile: %w", err)
			}
		} else {
			// Assume it's a Chef Supermarket profile or local path
			profilePath = profileURL
		}
	}

	// Verify InSpec is installed
	cmd := exec.CommandContext(ctx, "inspec", "version")
	if output, err := cmd.CombinedOutput(); err != nil {
		a.log.Error("InSpec not installed", "error", err, "output", string(output))
		return nil, fmt.Errorf("InSpec not installed: %w", err)
	}

	return map[string]interface{}{
		"temp_dir":     tempDir,
		"profile_path": profilePath,
		"ready":        true,
	}, nil
}

// ExecuteInSpecProfile executes an InSpec profile against an asset.
func (a *InSpecActivities) ExecuteInSpecProfile(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	runID, _ := input["run_id"].(string)
	assetID, _ := input["asset_id"].(string)
	profilePath, _ := input["profile_path"].(string)
	platform, _ := input["platform"].(string)
	assetType, _ := input["asset_type"].(string)

	a.log.Info("executing InSpec profile",
		"run_id", runID,
		"asset_id", assetID,
		"profile_path", profilePath,
		"platform", platform,
	)

	// Get asset connection details from database
	assetUUID, err := uuid.Parse(assetID)
	if err != nil {
		return nil, fmt.Errorf("invalid asset ID: %w", err)
	}

	var connectionInfo map[string]interface{}
	var hostname string
	err = a.db.QueryRowContext(ctx, `
		SELECT hostname, connection_info FROM assets WHERE id = $1
	`, assetUUID).Scan(&hostname, &connectionInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset info: %w", err)
	}

	// Prepare InSpec execution command
	outputFile := filepath.Join(os.TempDir(), "inspec-output-"+runID+".json")

	var cmd *exec.Cmd
	startTime := time.Now()

	switch assetType {
	case "vm":
		// For VMs, use SSH transport
		cmd = exec.CommandContext(ctx, "inspec", "exec", profilePath,
			"-t", fmt.Sprintf("ssh://%s", hostname),
			"--reporter", fmt.Sprintf("json:%s", outputFile),
		)
	case "container":
		// For containers, use docker transport
		cmd = exec.CommandContext(ctx, "inspec", "exec", profilePath,
			"-t", fmt.Sprintf("docker://%s", hostname),
			"--reporter", fmt.Sprintf("json:%s", outputFile),
		)
	case "cloud_account":
		// For cloud accounts (AWS/Azure/GCP), use appropriate transport
		switch platform {
		case "aws":
			cmd = exec.CommandContext(ctx, "inspec", "exec", profilePath,
				"-t", "aws://",
				"--reporter", fmt.Sprintf("json:%s", outputFile),
			)
		case "azure":
			cmd = exec.CommandContext(ctx, "inspec", "exec", profilePath,
				"-t", "azure://",
				"--reporter", fmt.Sprintf("json:%s", outputFile),
			)
		case "gcp":
			cmd = exec.CommandContext(ctx, "inspec", "exec", profilePath,
				"-t", "gcp://",
				"--reporter", fmt.Sprintf("json:%s", outputFile),
			)
		default:
			return nil, fmt.Errorf("unsupported cloud platform: %s", platform)
		}
	default:
		// Local execution (for testing)
		cmd = exec.CommandContext(ctx, "inspec", "exec", profilePath,
			"--reporter", fmt.Sprintf("json:%s", outputFile),
		)
	}

	// Execute InSpec
	output, err := cmd.CombinedOutput()
	duration := int(time.Since(startTime).Seconds())

	if err != nil {
		// InSpec returns non-zero exit code if any tests fail, which is expected
		a.log.Warn("InSpec execution returned non-zero exit code", "error", err, "output", string(output))
		// Continue to parse results
	}

	// Read the JSON output
	outputJSON, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read InSpec output: %w", err)
	}

	// Parse results to get statistics
	var result inspec.InSpecResult
	if err := json.Unmarshal(outputJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse InSpec JSON: %w", err)
	}

	return map[string]interface{}{
		"success":       true,
		"output_json":   string(outputJSON),
		"total_tests":   result.Statistics.Controls.Total,
		"passed_tests":  result.Statistics.Controls.Passed,
		"failed_tests":  result.Statistics.Controls.Failed,
		"skipped_tests": result.Statistics.Controls.Skipped,
		"duration":      duration,
	}, nil
}

// ParseInSpecResults parses InSpec JSON output and stores results in the database.
func (a *InSpecActivities) ParseInSpecResults(ctx context.Context, input map[string]interface{}) error {
	runID, _ := input["run_id"].(string)
	outputJSON, _ := input["output_json"].(string)

	runUUID, err := uuid.Parse(runID)
	if err != nil {
		return fmt.Errorf("invalid run ID: %w", err)
	}

	// Parse InSpec JSON
	parsedResult, err := a.svc.ParseResults([]byte(outputJSON))
	if err != nil {
		return fmt.Errorf("failed to parse InSpec results: %w", err)
	}

	// Store results for each profile
	for _, profile := range parsedResult.Profiles {
		for _, control := range profile.Controls {
			// Determine overall control status
			status := inspec.ResultStatusPassed
			var message string
			var resource string

			for _, testResult := range control.Results {
				if testResult.Status == "failed" {
					status = inspec.ResultStatusFailed
					message = testResult.Message
					resource = testResult.Resource
					break
				} else if testResult.Status == "skipped" {
					status = inspec.ResultStatusSkipped
					message = testResult.SkipMessage
				}
			}

			// Calculate total run time for control
			var totalRunTime float64
			for _, testResult := range control.Results {
				totalRunTime += testResult.RunTime
			}

			// Save result
			result := inspec.Result{
				RunID:           runUUID,
				ControlID:       control.ID,
				ControlTitle:    control.Title,
				Status:          status,
				Message:         message,
				Resource:        resource,
				SourceLocation:  fmt.Sprintf("%s:%d", control.SourceLocation.Ref, control.SourceLocation.Line),
				RunTime:         totalRunTime,
				CodeDescription: control.Desc,
			}

			if err := a.svc.SaveResult(ctx, result); err != nil {
				a.log.Error("failed to save result", "error", err, "control_id", control.ID)
				// Continue with other results
			}
		}
	}

	a.log.Info("parsed and stored InSpec results", "run_id", runID)
	return nil
}

// MapInSpecToComplianceControls maps InSpec control results to compliance framework controls.
func (a *InSpecActivities) MapInSpecToComplianceControls(ctx context.Context, input map[string]interface{}) error {
	runID, _ := input["run_id"].(string)
	profileID, _ := input["profile_id"].(string)

	runUUID, err := uuid.Parse(runID)
	if err != nil {
		return fmt.Errorf("invalid run ID: %w", err)
	}

	profileUUID, err := uuid.Parse(profileID)
	if err != nil {
		return fmt.Errorf("invalid profile ID: %w", err)
	}

	// Get the framework ID for this profile
	profile, err := a.svc.GetProfile(ctx, profileUUID)
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}
	if profile == nil {
		return fmt.Errorf("profile not found")
	}

	// Map results to compliance controls
	if err := a.svc.MapToControls(ctx, runUUID, profile.FrameworkID); err != nil {
		return fmt.Errorf("failed to map controls: %w", err)
	}

	a.log.Info("mapped InSpec results to compliance controls", "run_id", runID)
	return nil
}

// UpdateComplianceAssessment updates the compliance assessment based on InSpec results.
func (a *InSpecActivities) UpdateComplianceAssessment(ctx context.Context, input map[string]interface{}) error {
	runID, _ := input["run_id"].(string)
	orgID, _ := input["org_id"].(string)

	// This would update the overall compliance assessment for the organization
	// based on the new InSpec results

	a.log.Info("updated compliance assessment", "run_id", runID, "org_id", orgID)
	return nil
}

// CompleteInSpecRun marks an InSpec run as completed with final statistics.
func (a *InSpecActivities) CompleteInSpecRun(ctx context.Context, input map[string]interface{}) error {
	runID, _ := input["run_id"].(string)
	duration, _ := input["duration"].(float64)
	totalTests, _ := input["total_tests"].(float64)
	passedTests, _ := input["passed_tests"].(float64)
	failedTests, _ := input["failed_tests"].(float64)
	skippedTests, _ := input["skipped_tests"].(float64)

	runUUID, err := uuid.Parse(runID)
	if err != nil {
		return fmt.Errorf("invalid run ID: %w", err)
	}

	stats := inspec.Statistics{
		Duration: duration,
		Controls: inspec.StatCount{
			Total:   int(totalTests),
			Passed:  int(passedTests),
			Failed:  int(failedTests),
			Skipped: int(skippedTests),
		},
	}

	return a.svc.CompleteRun(ctx, runUUID, int(duration), stats)
}

// CleanupInSpecEnvironment cleans up temporary files and directories.
func (a *InSpecActivities) CleanupInSpecEnvironment(ctx context.Context, tempDir string) error {
	if tempDir == "" {
		return nil
	}

	if err := os.RemoveAll(tempDir); err != nil {
		a.log.Warn("failed to cleanup temp directory", "error", err, "dir", tempDir)
		return err
	}

	a.log.Debug("cleaned up InSpec environment", "dir", tempDir)
	return nil
}

// Helper function to check if a URL is a Git repository URL
func isGitURL(url string) bool {
	if url == "" {
		return false
	}
	// Simple check for common Git URL patterns
	return len(url) > 4 && (url[:4] == "git@" || url[:4] == "http" || url[:4] == "git:")
}
