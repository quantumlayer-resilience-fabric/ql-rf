// PR #46 / OPS-COMPLIANCE-TESTS — unit tests for pkg/compliance.
//
// Coverage focuses on:
//
//  1. Type constants — verify the string values used in JSON / DB.
//  2. JSON round-trips — catch breaking changes to the wire format.
//     Includes AIToolInvocationID (PR #42).
//  3. SQL-backed Service methods via sqlmock — happy path, empty result,
//     not-found edge cases, error paths.
//  4. GetComplianceScore pass-rate calculation — verify the
//     divide-by-zero guard.
package compliance

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

// -------------------------------------------------------------------------
// Type constants — string values are part of the wire and DB contracts.
// -------------------------------------------------------------------------

func TestSeverityConstants(t *testing.T) {
	cases := map[Severity]string{
		SeverityCritical: "critical",
		SeverityHigh:     "high",
		SeverityMedium:   "medium",
		SeverityLow:      "low",
	}
	for sev, want := range cases {
		if string(sev) != want {
			t.Errorf("severity %v = %q, want %q", sev, string(sev), want)
		}
	}
}

func TestAutomationSupportConstants(t *testing.T) {
	cases := map[AutomationSupport]string{
		AutomationFull:    "automated",
		AutomationPartial: "hybrid",
		AutomationManual:  "manual",
	}
	for a, want := range cases {
		if string(a) != want {
			t.Errorf("automation %v = %q, want %q", a, string(a), want)
		}
	}
}

func TestEvidenceTypeConstants(t *testing.T) {
	cases := map[EvidenceType]string{
		EvidenceScreenshot:  "screenshot",
		EvidenceLog:         "log",
		EvidenceConfig:      "config",
		EvidenceReport:      "report",
		EvidenceAttestation: "attestation",
	}
	for et, want := range cases {
		if string(et) != want {
			t.Errorf("evidence type %v = %q, want %q", et, string(et), want)
		}
	}
}

func TestAssessmentStatusConstants(t *testing.T) {
	cases := map[AssessmentStatus]string{
		AssessmentPending:    "pending",
		AssessmentInProgress: "in_progress",
		AssessmentCompleted:  "completed",
		AssessmentFailed:     "failed",
	}
	for s, want := range cases {
		if string(s) != want {
			t.Errorf("assessment status %v = %q, want %q", s, string(s), want)
		}
	}
}

func TestControlResultStatusConstants(t *testing.T) {
	cases := map[ControlResultStatus]string{
		ControlPassed:        "passed",
		ControlFailed:        "failed",
		ControlNotApplicable: "not_applicable",
		ControlManualReview:  "manual_review",
	}
	for s, want := range cases {
		if string(s) != want {
			t.Errorf("control result %v = %q, want %q", s, string(s), want)
		}
	}
}

// -------------------------------------------------------------------------
// JSON round-trips — catch wire-format regressions.
// -------------------------------------------------------------------------

func TestEvidence_JSONRoundTrip(t *testing.T) {
	invID := uuid.New()
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	original := Evidence{
		ID:                 uuid.New(),
		OrgID:              uuid.New(),
		ControlID:          uuid.New(),
		EvidenceType:       EvidenceAttestation,
		Title:              "AWS SSM patch attestation",
		StorageType:        "ai_tool_invocations",
		IsCurrent:          true,
		CollectedAt:        now,
		ValidFrom:          now,
		CreatedAt:          now,
		UpdatedAt:          now,
		AIToolInvocationID: &invID,
	}
	data, err := json.Marshal(&original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Evidence
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != original.ID || got.Title != original.Title {
		t.Errorf("identity fields lost in round-trip")
	}
	if got.AIToolInvocationID == nil || *got.AIToolInvocationID != invID {
		t.Errorf("AIToolInvocationID lost in round-trip: got %v, want %v", got.AIToolInvocationID, invID)
	}
}

func TestEvidence_JSONOmitsNilAIToolInvocationID(t *testing.T) {
	// PR #42 / PR #46: nil FK means "manually uploaded evidence" and must
	// NOT appear in JSON output (omitempty contract).
	e := Evidence{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		ControlID: uuid.New(),
		Title:     "manual upload",
	}
	data, err := json.Marshal(&e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(data); contains(got, "ai_tool_invocation_id") {
		t.Errorf("nil AIToolInvocationID should be omitted; got %q", got)
	}
}

func TestFramework_JSONRoundTrip(t *testing.T) {
	now := time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)
	original := Framework{
		ID:             uuid.New(),
		Name:           "CIS-1.4",
		Category:       "security",
		Version:        "1.4",
		RegulatoryBody: "CIS",
		EffectiveDate:  &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	data, err := json.Marshal(&original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Framework
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Name != original.Name {
		t.Errorf("Name = %q, want %q", got.Name, original.Name)
	}
	if got.EffectiveDate == nil || !got.EffectiveDate.Equal(now) {
		t.Errorf("EffectiveDate lost in round-trip")
	}
}

func TestControl_JSONUsesNameDBTagTitle(t *testing.T) {
	// The Control struct intentionally maps JSON `name` to DB column `title`
	// (see field tag in compliance.go). The test verifies JSON serialization
	// uses `name` so frontend consumers see the friendly key.
	c := Control{
		ID:        uuid.New(),
		Name:      "Ensure root login is disabled",
		Severity:  SeverityHigh,
		CreatedAt: time.Now().UTC(),
	}
	data, err := json.Marshal(&c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !contains(string(data), `"name":"Ensure root login is disabled"`) {
		t.Errorf("Control JSON missing `name` key; got %s", string(data))
	}
}

func TestExemption_JSONRoundTripWithOptionalPointers(t *testing.T) {
	assetID := uuid.New()
	now := time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)
	original := Exemption{
		ID:                  uuid.New(),
		OrgID:               uuid.New(),
		ControlID:           uuid.New(),
		AssetID:             &assetID,
		Reason:              "Compensating control: WAF in front of admin endpoint",
		ApprovedBy:          "security@example.com",
		ApprovedAt:          now,
		ExpiresAt:           now.Add(90 * 24 * time.Hour),
		ReviewFrequencyDays: 30,
		Status:              "active",
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	data, err := json.Marshal(&original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Exemption
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.AssetID == nil || *got.AssetID != assetID {
		t.Errorf("AssetID lost in round-trip: got %v, want %v", got.AssetID, assetID)
	}
	if got.SiteID != nil {
		t.Errorf("SiteID should be nil; got %v", got.SiteID)
	}
}

// -------------------------------------------------------------------------
// SQL-backed Service tests via sqlmock.
// -------------------------------------------------------------------------

func TestNewService(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	svc := NewService(db)
	if svc == nil {
		t.Fatal("NewService returned nil")
	}
	if svc.db != db {
		t.Errorf("service.db != injected db")
	}
}

func TestListFrameworks_HappyPath(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	fwID := uuid.New()
	now := time.Now().UTC()
	mock.ExpectQuery(regexp.QuoteMeta("FROM compliance_frameworks")).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "description", "category", "version",
			"regulatory_body", "effective_date", "created_at", "updated_at",
		}).AddRow(fwID, "CIS-1.4", "Center for Internet Security", "security",
			"1.4", "CIS", now, now, now))

	got, err := NewService(db).ListFrameworks(context.Background())
	if err != nil {
		t.Fatalf("ListFrameworks: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Name != "CIS-1.4" {
		t.Errorf("Name = %q, want CIS-1.4", got[0].Name)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestListFrameworks_Empty(t *testing.T) {
	db, mock, sErr := sqlmock.New()
	if sErr != nil {
		t.Fatalf("sqlmock: %v", sErr)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta("FROM compliance_frameworks")).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "description", "category", "version",
			"regulatory_body", "effective_date", "created_at", "updated_at",
		}))
	got, err := NewService(db).ListFrameworks(context.Background())
	if err != nil {
		t.Fatalf("ListFrameworks: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestListFrameworks_QueryError(t *testing.T) {
	db, mock, sErr := sqlmock.New()
	if sErr != nil {
		t.Fatalf("sqlmock: %v", sErr)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta("FROM compliance_frameworks")).
		WillReturnError(errors.New("connection refused"))
	_, err := NewService(db).ListFrameworks(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetControl_NotFoundReturnsNilNil(t *testing.T) {
	db, mock, sErr := sqlmock.New()
	if sErr != nil {
		t.Fatalf("sqlmock: %v", sErr)
	}
	defer db.Close()
	controlID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta("FROM compliance_controls WHERE id = $1")).
		WithArgs(controlID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "framework_id", "control_id", "title", "description",
			"severity", "recommendation", "created_at",
		}))
	got, err := NewService(db).GetControl(context.Background(), controlID)
	if err != nil {
		t.Fatalf("GetControl: %v", err)
	}
	if got != nil {
		t.Errorf("got %+v, want nil for not-found", got)
	}
}

func TestGetControl_HappyPath(t *testing.T) {
	db, mock, sErr := sqlmock.New()
	if sErr != nil {
		t.Fatalf("sqlmock: %v", sErr)
	}
	defer db.Close()
	controlID := uuid.New()
	fwID := uuid.New()
	now := time.Now().UTC()
	mock.ExpectQuery(regexp.QuoteMeta("FROM compliance_controls WHERE id = $1")).
		WithArgs(controlID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "framework_id", "control_id", "title", "description",
			"severity", "recommendation", "created_at",
		}).AddRow(controlID, fwID, "1.1.1", "Disable root login",
			"Ensure root login over SSH is disabled", "high", "Set PermitRootLogin no", now))
	got, err := NewService(db).GetControl(context.Background(), controlID)
	if err != nil {
		t.Fatalf("GetControl: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil control")
	}
	if got.ControlID != "1.1.1" {
		t.Errorf("ControlID = %q, want 1.1.1", got.ControlID)
	}
	if got.Severity != SeverityHigh {
		t.Errorf("Severity = %v, want SeverityHigh", got.Severity)
	}
}

func TestListEvidence_IncludesAIToolInvocationID(t *testing.T) {
	// PR #42 / PR #46 — verify the new column is read into the struct.
	db, mock, sErr := sqlmock.New()
	if sErr != nil {
		t.Fatalf("sqlmock: %v", sErr)
	}
	defer db.Close()
	orgID := uuid.New()
	controlID := uuid.New()
	evID := uuid.New()
	invID := uuid.New()
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("FROM compliance_evidence")).
		WithArgs(orgID, controlID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "org_id", "control_id", "evidence_type", "title", "description",
			"storage_type", "storage_path", "content_hash", "file_size_bytes", "mime_type",
			"collected_at", "collected_by", "collection_method", "valid_from", "valid_until",
			"is_current", "reviewed_by", "reviewed_at", "review_status", "review_notes",
			"created_at", "updated_at", "ai_tool_invocation_id",
		}).AddRow(
			evID, orgID, controlID, "attestation", "AWS patch evidence", "",
			"ai_tool_invocations", "", "", int64(0), "",
			now, "system", "automated", now, nil,
			true, "", nil, "approved", "",
			now, now, invID,
		))
	got, err := NewService(db).ListEvidence(context.Background(), orgID, controlID)
	if err != nil {
		t.Fatalf("ListEvidence: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].AIToolInvocationID == nil {
		t.Fatal("AIToolInvocationID nil; PR #42 column not scanned")
	}
	if *got[0].AIToolInvocationID != invID {
		t.Errorf("AIToolInvocationID = %v, want %v", got[0].AIToolInvocationID, invID)
	}
}

func TestListEvidence_NullAIToolInvocationID(t *testing.T) {
	// Manually-uploaded evidence has no ai_tool_invocation_id link.
	db, mock, sErr := sqlmock.New()
	if sErr != nil {
		t.Fatalf("sqlmock: %v", sErr)
	}
	defer db.Close()
	orgID := uuid.New()
	controlID := uuid.New()
	evID := uuid.New()
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("FROM compliance_evidence")).
		WithArgs(orgID, controlID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "org_id", "control_id", "evidence_type", "title", "description",
			"storage_type", "storage_path", "content_hash", "file_size_bytes", "mime_type",
			"collected_at", "collected_by", "collection_method", "valid_from", "valid_until",
			"is_current", "reviewed_by", "reviewed_at", "review_status", "review_notes",
			"created_at", "updated_at", "ai_tool_invocation_id",
		}).AddRow(
			evID, orgID, controlID, "screenshot", "manual upload", "",
			"s3", "s3://bucket/key", "", int64(0), "image/png",
			now, "user@example.com", "manual", now, nil,
			true, "", nil, "pending", "",
			now, now, nil,
		))
	got, err := NewService(db).ListEvidence(context.Background(), orgID, controlID)
	if err != nil {
		t.Fatalf("ListEvidence: %v", err)
	}
	if got[0].AIToolInvocationID != nil {
		t.Errorf("AIToolInvocationID should be nil for manual upload; got %v", got[0].AIToolInvocationID)
	}
}

func TestCreateEvidence_SetsGeneratedFields(t *testing.T) {
	// Verifies the service stamps ID + timestamps + IsCurrent before INSERT.
	db, mock, sErr := sqlmock.New()
	if sErr != nil {
		t.Fatalf("sqlmock: %v", sErr)
	}
	defer db.Close()
	before := time.Now().Add(-time.Second)
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO compliance_evidence")).
		WillReturnResult(sqlmock.NewResult(1, 1))

	in := Evidence{
		OrgID:        uuid.New(),
		ControlID:    uuid.New(),
		EvidenceType: EvidenceConfig,
		Title:        "rendered nginx config",
		StorageType:  "s3",
		StoragePath:  "s3://bucket/k",
	}
	created, err := NewService(db).CreateEvidence(context.Background(), in)
	if err != nil {
		t.Fatalf("CreateEvidence: %v", err)
	}
	if created.ID == uuid.Nil {
		t.Error("ID not generated")
	}
	if !created.IsCurrent {
		t.Error("IsCurrent should be true")
	}
	if created.CollectedAt.Before(before) {
		t.Error("CollectedAt not stamped")
	}
	if created.ValidFrom.Before(before) {
		t.Error("ValidFrom not stamped")
	}
}

// -------------------------------------------------------------------------
// GetComplianceScore — pass-rate calculation logic.
// -------------------------------------------------------------------------

func TestGetComplianceScore_PassRateCalculation(t *testing.T) {
	db, mock, sErr := sqlmock.New()
	if sErr != nil {
		t.Fatalf("sqlmock: %v", sErr)
	}
	defer db.Close()
	orgID := uuid.New()
	// passed=8 failed=2 → pass rate 80%
	mock.ExpectQuery(regexp.QuoteMeta("FROM compliance_assessments ca")).
		WithArgs(orgID).
		WillReturnRows(sqlmock.NewRows([]string{
			"assessment_count", "avg_score", "total_passed", "total_failed", "total_na",
		}).AddRow(3, 85.0, 8, 2, 1))
	got, err := NewService(db).GetComplianceScore(context.Background(), orgID, nil)
	if err != nil {
		t.Fatalf("GetComplianceScore: %v", err)
	}
	if got.PassRate != 80.0 {
		t.Errorf("PassRate = %v, want 80.0", got.PassRate)
	}
	if got.AssessmentCount != 3 {
		t.Errorf("AssessmentCount = %d, want 3", got.AssessmentCount)
	}
}

func TestGetComplianceScore_DivideByZeroGuarded(t *testing.T) {
	// No passed + no failed → PassRate stays 0, not NaN.
	db, mock, sErr := sqlmock.New()
	if sErr != nil {
		t.Fatalf("sqlmock: %v", sErr)
	}
	defer db.Close()
	orgID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta("FROM compliance_assessments ca")).
		WithArgs(orgID).
		WillReturnRows(sqlmock.NewRows([]string{
			"assessment_count", "avg_score", "total_passed", "total_failed", "total_na",
		}).AddRow(0, 0.0, 0, 0, 0))
	got, err := NewService(db).GetComplianceScore(context.Background(), orgID, nil)
	if err != nil {
		t.Fatalf("GetComplianceScore: %v", err)
	}
	if got.PassRate != 0 {
		t.Errorf("PassRate = %v, want 0 (divide-by-zero guard)", got.PassRate)
	}
}

func TestGetComplianceScore_WithFrameworkFilter(t *testing.T) {
	db, mock, sErr := sqlmock.New()
	if sErr != nil {
		t.Fatalf("sqlmock: %v", sErr)
	}
	defer db.Close()
	orgID := uuid.New()
	fwID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta("ca.framework_id = $2")).
		WithArgs(orgID, fwID).
		WillReturnRows(sqlmock.NewRows([]string{
			"assessment_count", "avg_score", "total_passed", "total_failed", "total_na",
		}).AddRow(1, 90.0, 9, 1, 0))
	got, err := NewService(db).GetComplianceScore(context.Background(), orgID, &fwID)
	if err != nil {
		t.Fatalf("GetComplianceScore with framework: %v", err)
	}
	if got.FrameworkID == nil || *got.FrameworkID != fwID {
		t.Errorf("FrameworkID = %v, want %v", got.FrameworkID, fwID)
	}
	if got.PassRate != 90.0 {
		t.Errorf("PassRate = %v, want 90.0", got.PassRate)
	}
}

// -------------------------------------------------------------------------
// Test helpers.
// -------------------------------------------------------------------------

// contains is a small stdlib-only substring helper.
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
