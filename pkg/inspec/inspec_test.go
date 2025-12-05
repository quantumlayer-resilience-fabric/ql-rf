package inspec

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

// TestNewService tests service initialization
func TestNewService(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	svc := NewService(db)
	if svc == nil {
		t.Error("NewService() returned nil")
	}
	if svc.db == nil {
		t.Error("NewService() db is nil")
	}
}

// TestCreateProfile tests profile creation
func TestCreateProfile(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		profile Profile
		mockFn  func()
		wantErr bool
	}{
		{
			name: "successful create",
			profile: Profile{
				Name:        "test-profile",
				Version:     "1.0.0",
				Title:       "Test Profile",
				Maintainer:  "Test Maintainer",
				Summary:     "Test summary",
				FrameworkID: uuid.New(),
				ProfileURL:  "https://example.com/profile",
				Platforms:   []string{"linux", "aws"},
			},
			mockFn: func() {
				mock.ExpectExec("INSERT INTO inspec_profiles").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "database error",
			profile: Profile{
				Name:        "test-profile",
				Version:     "1.0.0",
				FrameworkID: uuid.New(),
			},
			mockFn: func() {
				mock.ExpectExec("INSERT INTO inspec_profiles").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			result, err := svc.CreateProfile(ctx, tt.profile)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateProfile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Error("CreateProfile() returned nil result")
					return
				}
				if result.ID == uuid.Nil {
					t.Error("CreateProfile() did not set ID")
				}
				if result.CreatedAt.IsZero() {
					t.Error("CreateProfile() did not set CreatedAt")
				}
				if result.UpdatedAt.IsZero() {
					t.Error("CreateProfile() did not set UpdatedAt")
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestGetProfile tests profile retrieval
func TestGetProfile(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	profileID := uuid.New()
	frameworkID := uuid.New()
	now := time.Now()

	tests := []struct {
		name    string
		mockFn  func()
		wantNil bool
		wantErr bool
	}{
		{
			name: "profile found",
			mockFn: func() {
				rows := sqlmock.NewRows([]string{
					"id", "name", "version", "title", "maintainer", "summary",
					"framework_id", "profile_url", "platforms", "created_at", "updated_at",
				}).AddRow(
					profileID, "test-profile", "1.0.0", "Test Profile", "Test Maintainer",
					"Test summary", frameworkID, "https://example.com/profile",
					"{linux,aws}", now, now,
				)
				mock.ExpectQuery("SELECT (.+) FROM inspec_profiles").
					WithArgs(profileID).
					WillReturnRows(rows)
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "profile not found",
			mockFn: func() {
				mock.ExpectQuery("SELECT (.+) FROM inspec_profiles").
					WithArgs(profileID).
					WillReturnError(sql.ErrNoRows)
			},
			wantNil: true,
			wantErr: false,
		},
		{
			name: "database error",
			mockFn: func() {
				mock.ExpectQuery("SELECT (.+) FROM inspec_profiles").
					WithArgs(profileID).
					WillReturnError(sql.ErrConnDone)
			},
			wantNil: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			result, err := svc.GetProfile(ctx, profileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetProfile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantNil && result != nil {
				t.Error("GetProfile() should return nil for not found")
			}
			if !tt.wantNil && !tt.wantErr && result == nil {
				t.Error("GetProfile() returned nil result")
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestGetAvailableProfiles tests listing available profiles
func TestGetAvailableProfiles(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	tests := []struct {
		name      string
		mockFn    func()
		wantCount int
		wantErr   bool
	}{
		{
			name: "multiple profiles",
			mockFn: func() {
				rows := sqlmock.NewRows([]string{
					"id", "name", "title", "version", "platforms",
					"framework_id", "framework_name", "control_count",
				}).
					AddRow(uuid.New(), "cis-aws", "CIS AWS", "1.5.0", "{aws}", uuid.New(), "CIS", 50).
					AddRow(uuid.New(), "cis-linux", "CIS Linux", "1.1.0", "{linux}", uuid.New(), "CIS", 120)

				mock.ExpectQuery("SELECT (.+) FROM inspec_profiles").
					WillReturnRows(rows)
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "no profiles",
			mockFn: func() {
				rows := sqlmock.NewRows([]string{
					"id", "name", "title", "version", "platforms",
					"framework_id", "framework_name", "control_count",
				})
				mock.ExpectQuery("SELECT (.+) FROM inspec_profiles").
					WillReturnRows(rows)
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "database error",
			mockFn: func() {
				mock.ExpectQuery("SELECT (.+) FROM inspec_profiles").
					WillReturnError(sql.ErrConnDone)
			},
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			profiles, err := svc.GetAvailableProfiles(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAvailableProfiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(profiles) != tt.wantCount {
				t.Errorf("GetAvailableProfiles() count = %d, want %d", len(profiles), tt.wantCount)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestCreateRun tests run creation
func TestCreateRun(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	orgID := uuid.New()
	assetID := uuid.New()
	profileID := uuid.New()

	tests := []struct {
		name    string
		mockFn  func()
		wantErr bool
	}{
		{
			name: "successful create",
			mockFn: func() {
				mock.ExpectExec("INSERT INTO inspec_runs").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "database error",
			mockFn: func() {
				mock.ExpectExec("INSERT INTO inspec_runs").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			run, err := svc.CreateRun(ctx, orgID, assetID, profileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateRun() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if run == nil {
					t.Error("CreateRun() returned nil")
					return
				}
				if run.ID == uuid.Nil {
					t.Error("CreateRun() did not set ID")
				}
				if run.Status != RunStatusPending {
					t.Errorf("CreateRun() status = %s, want %s", run.Status, RunStatusPending)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestUpdateRunStatus tests run status updates
func TestUpdateRunStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()
	runID := uuid.New()

	tests := []struct {
		name     string
		status   RunStatus
		errorMsg string
		mockFn   func()
		wantErr  bool
	}{
		{
			name:   "update to running",
			status: RunStatusRunning,
			mockFn: func() {
				mock.ExpectExec("UPDATE inspec_runs").
					WithArgs(RunStatusRunning, sqlmock.AnyArg(), sqlmock.AnyArg(), runID).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:   "update to completed",
			status: RunStatusCompleted,
			mockFn: func() {
				mock.ExpectExec("UPDATE inspec_runs").
					WithArgs(RunStatusCompleted, sqlmock.AnyArg(), "", sqlmock.AnyArg(), runID).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:     "update to failed with error",
			status:   RunStatusFailed,
			errorMsg: "test error",
			mockFn: func() {
				mock.ExpectExec("UPDATE inspec_runs").
					WithArgs(RunStatusFailed, sqlmock.AnyArg(), "test error", sqlmock.AnyArg(), runID).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:   "database error",
			status: RunStatusRunning,
			mockFn: func() {
				mock.ExpectExec("UPDATE inspec_runs").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			err := svc.UpdateRunStatus(ctx, runID, tt.status, tt.errorMsg)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateRunStatus() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestCompleteRun tests completing a run with statistics
func TestCompleteRun(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()
	runID := uuid.New()

	tests := []struct {
		name     string
		duration int
		stats    Statistics
		mockFn   func()
		wantErr  bool
	}{
		{
			name:     "successful completion",
			duration: 120,
			stats: Statistics{
				Duration: 120.5,
				Controls: StatCount{
					Total:   100,
					Passed:  90,
					Failed:  8,
					Skipped: 2,
				},
			},
			mockFn: func() {
				mock.ExpectExec("UPDATE inspec_runs").
					WithArgs(
						RunStatusCompleted,
						sqlmock.AnyArg(),
						120,
						100, 90, 8, 2,
						sqlmock.AnyArg(),
						runID,
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:     "database error",
			duration: 120,
			stats:    Statistics{},
			mockFn: func() {
				mock.ExpectExec("UPDATE inspec_runs").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			err := svc.CompleteRun(ctx, runID, tt.duration, tt.stats)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompleteRun() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestParseResults tests parsing InSpec JSON output
func TestParseResults(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	svc := NewService(db)

	tests := []struct {
		name       string
		jsonOutput string
		wantErr    bool
		validate   func(*InSpecResult) bool
	}{
		{
			name: "valid InSpec output",
			jsonOutput: `{
				"platform": {
					"name": "ubuntu",
					"release": "20.04",
					"target": "ssh://user@host"
				},
				"profiles": [{
					"name": "cis-aws",
					"version": "1.5.0",
					"title": "CIS AWS Benchmark",
					"controls": [{
						"id": "cis-1.1",
						"title": "Test Control",
						"results": [{
							"status": "passed",
							"code_desc": "Test passed",
							"run_time": 0.5
						}]
					}]
				}],
				"statistics": {
					"duration": 10.5,
					"controls": {
						"total": 100,
						"passed": 95,
						"failed": 3,
						"skipped": 2
					}
				},
				"version": "5.21.29"
			}`,
			wantErr: false,
			validate: func(r *InSpecResult) bool {
				return r.Platform.Name == "ubuntu" &&
					r.Statistics.Controls.Total == 100 &&
					len(r.Profiles) == 1
			},
		},
		{
			name:       "invalid JSON",
			jsonOutput: `{invalid json}`,
			wantErr:    true,
			validate:   nil,
		},
		{
			name:       "empty JSON",
			jsonOutput: `{}`,
			wantErr:    false,
			validate: func(r *InSpecResult) bool {
				return r != nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.ParseResults([]byte(tt.jsonOutput))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseResults() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				if !tt.validate(result) {
					t.Error("ParseResults() validation failed")
				}
			}
		})
	}
}

// TestSaveResult tests saving individual control results
func TestSaveResult(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		result  Result
		mockFn  func()
		wantErr bool
	}{
		{
			name: "successful save",
			result: Result{
				RunID:           uuid.New(),
				ControlID:       "cis-1.1",
				ControlTitle:    "Test Control",
				Status:          ResultStatusPassed,
				Message:         "Test passed",
				Resource:        "aws_iam_user",
				SourceLocation:  "controls/iam.rb:10",
				RunTime:         0.5,
				CodeDescription: "Ensure MFA is enabled",
			},
			mockFn: func() {
				mock.ExpectExec("INSERT INTO inspec_results").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "database error",
			result: Result{
				RunID:     uuid.New(),
				ControlID: "cis-1.1",
			},
			mockFn: func() {
				mock.ExpectExec("INSERT INTO inspec_results").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			err := svc.SaveResult(ctx, tt.result)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveResult() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestMapToControls tests mapping InSpec results to compliance controls
func TestMapToControls(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	runID := uuid.New()
	profileID := uuid.New()
	frameworkID := uuid.New()

	tests := []struct {
		name    string
		mockFn  func()
		wantErr bool
	}{
		{
			name: "successful mapping",
			mockFn: func() {
				// Mock getting profile ID
				profileRow := sqlmock.NewRows([]string{"profile_id"}).
					AddRow(profileID)
				mock.ExpectQuery("SELECT profile_id FROM inspec_runs").
					WithArgs(runID).
					WillReturnRows(profileRow)

				// Mock getting results
				resultsRows := sqlmock.NewRows([]string{
					"control_id", "status", "message", "control_title",
				})
				mock.ExpectQuery("SELECT (.+) FROM inspec_results").
					WithArgs(runID).
					WillReturnRows(resultsRows)
			},
			wantErr: false,
		},
		{
			name: "run not found",
			mockFn: func() {
				mock.ExpectQuery("SELECT profile_id FROM inspec_runs").
					WithArgs(runID).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			err := svc.MapToControls(ctx, runID, frameworkID)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapToControls() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestCreateControlMapping tests control mapping creation
func TestCreateControlMapping(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		mapping ControlMapping
		mockFn  func()
		wantErr bool
	}{
		{
			name: "successful create",
			mapping: ControlMapping{
				InSpecControlID:     "cis-1.1",
				ComplianceControlID: uuid.New(),
				ProfileID:           uuid.New(),
				MappingConfidence:   1.0,
				Notes:               "Direct mapping",
			},
			mockFn: func() {
				mock.ExpectExec("INSERT INTO inspec_control_mappings").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "database error",
			mapping: ControlMapping{
				InSpecControlID:     "cis-1.1",
				ComplianceControlID: uuid.New(),
			},
			mockFn: func() {
				mock.ExpectExec("INSERT INTO inspec_control_mappings").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			result, err := svc.CreateControlMapping(ctx, tt.mapping)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateControlMapping() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != nil {
				if result.ID == uuid.Nil {
					t.Error("CreateControlMapping() did not set ID")
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestListRuns tests listing runs for an organization
func TestListRuns(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()
	orgID := uuid.New()

	tests := []struct {
		name      string
		limit     int
		offset    int
		mockFn    func()
		wantCount int
		wantErr   bool
	}{
		{
			name:   "successful list with default limit",
			limit:  0,
			offset: 0,
			mockFn: func() {
				rows := sqlmock.NewRows([]string{
					"id", "asset_id", "status", "started_at", "completed_at", "duration",
					"total_tests", "passed_tests", "failed_tests", "asset_name",
					"profile_name", "framework_name",
				}).
					AddRow(
						uuid.New(), uuid.New(), RunStatusCompleted, time.Now(), time.Now(), 120,
						100, 95, 5, "test-asset", "cis-aws", "CIS",
					)

				mock.ExpectQuery("SELECT (.+) FROM inspec_runs").
					WithArgs(orgID, 50, 0).
					WillReturnRows(rows)
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:   "with pagination",
			limit:  10,
			offset: 20,
			mockFn: func() {
				rows := sqlmock.NewRows([]string{
					"id", "asset_id", "status", "started_at", "completed_at", "duration",
					"total_tests", "passed_tests", "failed_tests", "asset_name",
					"profile_name", "framework_name",
				})

				mock.ExpectQuery("SELECT (.+) FROM inspec_runs").
					WithArgs(orgID, 10, 20).
					WillReturnRows(rows)
			},
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			runs, err := svc.ListRuns(ctx, orgID, tt.limit, tt.offset)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListRuns() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(runs) != tt.wantCount {
				t.Errorf("ListRuns() count = %d, want %d", len(runs), tt.wantCount)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestRunStatus tests RunStatus type
func TestRunStatus(t *testing.T) {
	tests := []struct {
		name   string
		status RunStatus
		want   string
	}{
		{"pending", RunStatusPending, "pending"},
		{"running", RunStatusRunning, "running"},
		{"completed", RunStatusCompleted, "completed"},
		{"failed", RunStatusFailed, "failed"},
		{"cancelled", RunStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("RunStatus = %v, want %v", tt.status, tt.want)
			}
		})
	}
}

// TestResultStatus tests ResultStatus type
func TestResultStatus(t *testing.T) {
	tests := []struct {
		name   string
		status ResultStatus
		want   string
	}{
		{"passed", ResultStatusPassed, "passed"},
		{"failed", ResultStatusFailed, "failed"},
		{"skipped", ResultStatusSkipped, "skipped"},
		{"error", ResultStatusError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("ResultStatus = %v, want %v", tt.status, tt.want)
			}
		})
	}
}

// TestInSpecResultStructure tests InSpec result structure parsing
func TestInSpecResultStructure(t *testing.T) {
	jsonData := `{
		"platform": {
			"name": "ubuntu",
			"release": "20.04",
			"target": "local://"
		},
		"profiles": [{
			"name": "test-profile",
			"version": "1.0.0",
			"controls": [{
				"id": "test-1",
				"title": "Test Control",
				"results": [{
					"status": "passed",
					"run_time": 0.5
				}]
			}]
		}],
		"statistics": {
			"duration": 10.5,
			"controls": {
				"total": 1,
				"passed": 1,
				"failed": 0,
				"skipped": 0
			}
		},
		"version": "5.21.29"
	}`

	var result InSpecResult
	err := json.Unmarshal([]byte(jsonData), &result)
	if err != nil {
		t.Fatalf("failed to unmarshal InSpec result: %v", err)
	}

	if result.Platform.Name != "ubuntu" {
		t.Errorf("Platform.Name = %v, want ubuntu", result.Platform.Name)
	}

	if len(result.Profiles) != 1 {
		t.Errorf("Profiles count = %d, want 1", len(result.Profiles))
	}

	if result.Statistics.Controls.Total != 1 {
		t.Errorf("Statistics.Controls.Total = %d, want 1", result.Statistics.Controls.Total)
	}

	if result.Version != "5.21.29" {
		t.Errorf("Version = %v, want 5.21.29", result.Version)
	}
}
