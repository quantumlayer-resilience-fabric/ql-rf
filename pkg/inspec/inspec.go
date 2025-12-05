package inspec

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Service provides InSpec integration functionality.
type Service struct {
	db *sql.DB
}

// NewService creates a new InSpec service.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// CreateProfile creates a new InSpec profile.
func (s *Service) CreateProfile(ctx context.Context, profile Profile) (*Profile, error) {
	profile.ID = uuid.New()
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO inspec_profiles (
			id, name, version, title, maintainer, summary, framework_id,
			profile_url, platforms, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, profile.ID, profile.Name, profile.Version, profile.Title, profile.Maintainer,
		profile.Summary, profile.FrameworkID, profile.ProfileURL, pq.Array(profile.Platforms),
		profile.CreatedAt, profile.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	return &profile, nil
}

// GetProfile retrieves a profile by ID.
func (s *Service) GetProfile(ctx context.Context, profileID uuid.UUID) (*Profile, error) {
	var p Profile
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, version, title, maintainer, summary, framework_id,
		       profile_url, platforms, created_at, updated_at
		FROM inspec_profiles
		WHERE id = $1
	`, profileID).Scan(
		&p.ID, &p.Name, &p.Version, &p.Title, &p.Maintainer, &p.Summary,
		&p.FrameworkID, &p.ProfileURL, pq.Array(&p.Platforms), &p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	return &p, nil
}

// GetAvailableProfiles returns all available InSpec profiles with their framework information.
func (s *Service) GetAvailableProfiles(ctx context.Context) ([]AvailableProfile, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			p.id, p.name, p.title, p.version, p.platforms,
			f.id as framework_id, f.name as framework_name,
			COUNT(m.id) as control_count
		FROM inspec_profiles p
		JOIN compliance_frameworks f ON p.framework_id = f.id
		LEFT JOIN inspec_control_mappings m ON p.id = m.profile_id
		GROUP BY p.id, p.name, p.title, p.version, p.platforms, f.id, f.name
		ORDER BY f.name, p.name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}
	defer rows.Close()

	var profiles []AvailableProfile
	for rows.Next() {
		var p AvailableProfile
		if err := rows.Scan(
			&p.ProfileID, &p.Name, &p.Title, &p.Version, pq.Array(&p.Platforms),
			&p.FrameworkID, &p.Framework, &p.ControlCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan profile: %w", err)
		}
		profiles = append(profiles, p)
	}

	return profiles, rows.Err()
}

// CreateRun creates a new InSpec run.
func (s *Service) CreateRun(ctx context.Context, orgID, assetID, profileID uuid.UUID) (*Run, error) {
	run := &Run{
		ID:        uuid.New(),
		OrgID:     orgID,
		AssetID:   assetID,
		ProfileID: profileID,
		Status:    RunStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO inspec_runs (
			id, org_id, asset_id, profile_id, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, run.ID, run.OrgID, run.AssetID, run.ProfileID, run.Status, run.CreatedAt, run.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create run: %w", err)
	}

	return run, nil
}

// UpdateRunStatus updates the status of an InSpec run.
func (s *Service) UpdateRunStatus(ctx context.Context, runID uuid.UUID, status RunStatus, errorMsg string) error {
	now := time.Now()
	var err error

	if status == RunStatusRunning {
		_, err = s.db.ExecContext(ctx, `
			UPDATE inspec_runs
			SET status = $1, started_at = $2, updated_at = $3
			WHERE id = $4
		`, status, now, now, runID)
	} else if status == RunStatusCompleted || status == RunStatusFailed || status == RunStatusCancelled {
		_, err = s.db.ExecContext(ctx, `
			UPDATE inspec_runs
			SET status = $1, completed_at = $2, error_message = $3, updated_at = $4
			WHERE id = $5
		`, status, now, errorMsg, now, runID)
	} else {
		_, err = s.db.ExecContext(ctx, `
			UPDATE inspec_runs
			SET status = $1, updated_at = $2
			WHERE id = $3
		`, status, now, runID)
	}

	if err != nil {
		return fmt.Errorf("failed to update run status: %w", err)
	}
	return nil
}

// CompleteRun marks a run as completed with statistics.
func (s *Service) CompleteRun(ctx context.Context, runID uuid.UUID, duration int, stats Statistics) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE inspec_runs
		SET status = $1, completed_at = $2, duration = $3,
		    total_tests = $4, passed_tests = $5, failed_tests = $6, skipped_tests = $7,
		    updated_at = $8
		WHERE id = $9
	`, RunStatusCompleted, now, duration, stats.Controls.Total, stats.Controls.Passed,
		stats.Controls.Failed, stats.Controls.Skipped, now, runID)
	if err != nil {
		return fmt.Errorf("failed to complete run: %w", err)
	}
	return nil
}

// GetRun retrieves a run by ID.
func (s *Service) GetRun(ctx context.Context, runID uuid.UUID) (*Run, error) {
	var r Run
	var startedAt, completedAt sql.NullTime
	var duration, totalTests, passedTests, failedTests, skippedTests sql.NullInt64
	var errorMsg, rawOutput sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, org_id, asset_id, profile_id, status, started_at, completed_at,
		       duration, total_tests, passed_tests, failed_tests, skipped_tests,
		       error_message, raw_output, created_at, updated_at
		FROM inspec_runs
		WHERE id = $1
	`, runID).Scan(
		&r.ID, &r.OrgID, &r.AssetID, &r.ProfileID, &r.Status, &startedAt, &completedAt,
		&duration, &totalTests, &passedTests, &failedTests, &skippedTests,
		&errorMsg, &rawOutput, &r.CreatedAt, &r.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get run: %w", err)
	}

	if startedAt.Valid {
		r.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		r.CompletedAt = &completedAt.Time
	}
	if duration.Valid {
		r.Duration = int(duration.Int64)
	}
	if totalTests.Valid {
		r.TotalTests = int(totalTests.Int64)
	}
	if passedTests.Valid {
		r.PassedTests = int(passedTests.Int64)
	}
	if failedTests.Valid {
		r.FailedTests = int(failedTests.Int64)
	}
	if skippedTests.Valid {
		r.SkippedTests = int(skippedTests.Int64)
	}
	if errorMsg.Valid {
		r.ErrorMessage = errorMsg.String
	}
	if rawOutput.Valid {
		r.RawOutput = rawOutput.String
	}

	return &r, nil
}

// ListRuns returns runs for an organization.
func (s *Service) ListRuns(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]RunSummary, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			r.id, r.asset_id, r.status, r.started_at, r.completed_at, r.duration,
			r.total_tests, r.passed_tests, r.failed_tests,
			a.name as asset_name,
			p.name as profile_name,
			f.name as framework_name
		FROM inspec_runs r
		JOIN assets a ON r.asset_id = a.id
		JOIN inspec_profiles p ON r.profile_id = p.id
		JOIN compliance_frameworks f ON p.framework_id = f.id
		WHERE r.org_id = $1
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3
	`, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list runs: %w", err)
	}
	defer rows.Close()

	var runs []RunSummary
	for rows.Next() {
		var r RunSummary
		var startedAt, completedAt sql.NullTime
		var duration, totalTests, passedTests, failedTests sql.NullInt64

		if err := rows.Scan(
			&r.RunID, &r.AssetID, &r.Status, &startedAt, &completedAt, &duration,
			&totalTests, &passedTests, &failedTests,
			&r.AssetName, &r.ProfileName, &r.Framework,
		); err != nil {
			return nil, fmt.Errorf("failed to scan run: %w", err)
		}

		if startedAt.Valid {
			r.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			r.CompletedAt = &completedAt.Time
		}
		if duration.Valid {
			r.Duration = int(duration.Int64)
		}
		if totalTests.Valid {
			r.TotalTests = int(totalTests.Int64)
		}
		if passedTests.Valid {
			r.PassedTests = int(passedTests.Int64)
		}
		if failedTests.Valid {
			r.FailedTests = int(failedTests.Int64)
		}

		// Calculate pass rate
		if r.TotalTests > 0 {
			r.PassRate = float64(r.PassedTests) / float64(r.TotalTests) * 100
		}

		runs = append(runs, r)
	}

	return runs, rows.Err()
}

// SaveResult saves an InSpec control result.
func (s *Service) SaveResult(ctx context.Context, result Result) error {
	result.ID = uuid.New()
	result.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO inspec_results (
			id, run_id, control_id, control_title, status, message,
			resource, source_location, run_time, code_description, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, result.ID, result.RunID, result.ControlID, result.ControlTitle, result.Status,
		result.Message, result.Resource, result.SourceLocation, result.RunTime,
		result.CodeDescription, result.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to save result: %w", err)
	}

	return nil
}

// GetRunResults retrieves all results for a run.
func (s *Service) GetRunResults(ctx context.Context, runID uuid.UUID) ([]Result, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, run_id, control_id, control_title, status, message,
		       resource, source_location, run_time, code_description, created_at
		FROM inspec_results
		WHERE run_id = $1
		ORDER BY
			CASE status
				WHEN 'failed' THEN 1
				WHEN 'error' THEN 2
				WHEN 'skipped' THEN 3
				WHEN 'passed' THEN 4
			END,
			control_id
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get run results: %w", err)
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var r Result
		var message, resource, sourceLocation, codeDesc sql.NullString

		if err := rows.Scan(
			&r.ID, &r.RunID, &r.ControlID, &r.ControlTitle, &r.Status,
			&message, &resource, &sourceLocation, &r.RunTime, &codeDesc, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		if message.Valid {
			r.Message = message.String
		}
		if resource.Valid {
			r.Resource = resource.String
		}
		if sourceLocation.Valid {
			r.SourceLocation = sourceLocation.String
		}
		if codeDesc.Valid {
			r.CodeDescription = codeDesc.String
		}

		results = append(results, r)
	}

	return results, rows.Err()
}

// ParseResults parses InSpec JSON output and returns structured results.
func (s *Service) ParseResults(jsonOutput []byte) (*InSpecResult, error) {
	var result InSpecResult
	if err := json.Unmarshal(jsonOutput, &result); err != nil {
		return nil, fmt.Errorf("failed to parse InSpec JSON: %w", err)
	}
	return &result, nil
}

// MapToControls maps InSpec control results to compliance controls.
func (s *Service) MapToControls(ctx context.Context, runID uuid.UUID, frameworkID uuid.UUID) error {
	// Get the profile ID from the run
	var profileID uuid.UUID
	err := s.db.QueryRowContext(ctx, `
		SELECT profile_id FROM inspec_runs WHERE id = $1
	`, runID).Scan(&profileID)
	if err != nil {
		return fmt.Errorf("failed to get profile ID: %w", err)
	}

	// Get all results for this run that failed
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.control_id, r.status, r.message, r.control_title
		FROM inspec_results r
		WHERE r.run_id = $1
	`, runID)
	if err != nil {
		return fmt.Errorf("failed to get results: %w", err)
	}
	defer rows.Close()

	// For each result, find the compliance control mapping and create assessment result
	for rows.Next() {
		var controlID, status, message, title string
		if err := rows.Scan(&controlID, &status, &message, &title); err != nil {
			return fmt.Errorf("failed to scan result: %w", err)
		}

		// Find the compliance control mapping
		var complianceControlID uuid.UUID
		err := s.db.QueryRowContext(ctx, `
			SELECT compliance_control_id
			FROM inspec_control_mappings
			WHERE profile_id = $1 AND inspec_control_id = $2
		`, profileID, controlID).Scan(&complianceControlID)
		if err == sql.ErrNoRows {
			// No mapping found, skip
			continue
		}
		if err != nil {
			return fmt.Errorf("failed to get control mapping: %w", err)
		}

		// Map InSpec status to compliance status
		var complianceStatus string
		switch status {
		case "passed":
			complianceStatus = "passed"
		case "failed", "error":
			complianceStatus = "failed"
		case "skipped":
			complianceStatus = "not_applicable"
		default:
			complianceStatus = "manual_review"
		}

		// Note: This would typically create a compliance assessment result
		// but we'll leave this as a placeholder for now
		_ = complianceControlID
		_ = complianceStatus
	}

	return rows.Err()
}

// CreateControlMapping creates a mapping between an InSpec control and a compliance control.
func (s *Service) CreateControlMapping(ctx context.Context, mapping ControlMapping) (*ControlMapping, error) {
	mapping.ID = uuid.New()
	mapping.CreatedAt = time.Now()
	mapping.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO inspec_control_mappings (
			id, inspec_control_id, compliance_control_id, profile_id,
			mapping_confidence, notes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, mapping.ID, mapping.InSpecControlID, mapping.ComplianceControlID, mapping.ProfileID,
		mapping.MappingConfidence, mapping.Notes, mapping.CreatedAt, mapping.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create control mapping: %w", err)
	}

	return &mapping, nil
}

// GetControlMappings retrieves all control mappings for a profile.
func (s *Service) GetControlMappings(ctx context.Context, profileID uuid.UUID) ([]ControlMapping, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, inspec_control_id, compliance_control_id, profile_id,
		       mapping_confidence, notes, created_at, updated_at
		FROM inspec_control_mappings
		WHERE profile_id = $1
		ORDER BY inspec_control_id
	`, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get control mappings: %w", err)
	}
	defer rows.Close()

	var mappings []ControlMapping
	for rows.Next() {
		var m ControlMapping
		var notes sql.NullString
		if err := rows.Scan(
			&m.ID, &m.InSpecControlID, &m.ComplianceControlID, &m.ProfileID,
			&m.MappingConfidence, &notes, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan mapping: %w", err)
		}
		if notes.Valid {
			m.Notes = notes.String
		}
		mappings = append(mappings, m)
	}

	return mappings, rows.Err()
}
