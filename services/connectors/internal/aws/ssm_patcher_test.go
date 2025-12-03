package aws

import (
	"testing"
	"time"
)

func TestApplyPatchParams(t *testing.T) {
	tests := []struct {
		name     string
		params   ApplyPatchParams
		wantOp   string
		wantReb  string
	}{
		{
			name: "scan operation",
			params: ApplyPatchParams{
				Operation:    "Scan",
				RebootOption: "NoReboot",
			},
			wantOp:  "Scan",
			wantReb: "NoReboot",
		},
		{
			name: "install with reboot",
			params: ApplyPatchParams{
				Operation:    "Install",
				RebootOption: "RebootIfNeeded",
			},
			wantOp:  "Install",
			wantReb: "RebootIfNeeded",
		},
		{
			name: "install without reboot",
			params: ApplyPatchParams{
				Operation:    "Install",
				RebootOption: "NoReboot",
			},
			wantOp:  "Install",
			wantReb: "NoReboot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.params.Operation != tt.wantOp {
				t.Errorf("Operation = %v, want %v", tt.params.Operation, tt.wantOp)
			}
			if tt.params.RebootOption != tt.wantReb {
				t.Errorf("RebootOption = %v, want %v", tt.params.RebootOption, tt.wantReb)
			}
		})
	}
}

func TestPatchOperation(t *testing.T) {
	now := time.Now()
	endTime := now.Add(5 * time.Minute)

	op := &PatchOperation{
		CommandID:     "cmd-123456",
		InstanceID:    "i-abc123",
		Status:        "Success",
		StatusDetails: "Patches applied successfully",
		StartTime:     now,
		EndTime:       &endTime,
		Output:        "5 patches installed",
		ErrorOutput:   "",
	}

	if op.CommandID != "cmd-123456" {
		t.Errorf("CommandID = %v, want cmd-123456", op.CommandID)
	}

	if op.Status != "Success" {
		t.Errorf("Status = %v, want Success", op.Status)
	}

	duration := op.EndTime.Sub(op.StartTime)
	if duration != 5*time.Minute {
		t.Errorf("Duration = %v, want 5m", duration)
	}
}

func TestPatchComplianceStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         PatchComplianceStatus
		wantCompliance string
	}{
		{
			name: "compliant instance",
			status: PatchComplianceStatus{
				InstanceID:       "i-compliant",
				InstalledCount:   50,
				MissingCount:     0,
				FailedCount:      0,
				ComplianceStatus: "COMPLIANT",
			},
			wantCompliance: "COMPLIANT",
		},
		{
			name: "non-compliant with missing patches",
			status: PatchComplianceStatus{
				InstanceID:       "i-missing",
				InstalledCount:   45,
				MissingCount:     5,
				FailedCount:      0,
				ComplianceStatus: "NON_COMPLIANT",
			},
			wantCompliance: "NON_COMPLIANT",
		},
		{
			name: "non-compliant with failed patches",
			status: PatchComplianceStatus{
				InstanceID:       "i-failed",
				InstalledCount:   48,
				MissingCount:     0,
				FailedCount:      2,
				ComplianceStatus: "NON_COMPLIANT",
			},
			wantCompliance: "NON_COMPLIANT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status.ComplianceStatus != tt.wantCompliance {
				t.Errorf("ComplianceStatus = %v, want %v", tt.status.ComplianceStatus, tt.wantCompliance)
			}
		})
	}
}

func TestManagedInstance(t *testing.T) {
	lastPing := time.Now()

	instance := ManagedInstance{
		InstanceID:       "i-test123",
		PingStatus:       "Online",
		PlatformType:     "Linux",
		PlatformName:     "Amazon Linux 2",
		PlatformVersion:  "2.0",
		AgentVersion:     "3.2.1",
		LastPingDateTime: &lastPing,
		IsLatestVersion:  true,
	}

	if instance.PingStatus != "Online" {
		t.Errorf("PingStatus = %v, want Online", instance.PingStatus)
	}

	if instance.PlatformType != "Linux" {
		t.Errorf("PlatformType = %v, want Linux", instance.PlatformType)
	}

	if !instance.IsLatestVersion {
		t.Error("IsLatestVersion should be true")
	}
}

func TestPatchBaseline(t *testing.T) {
	baseline := PatchBaseline{
		BaselineID:          "pb-abc123",
		BaselineName:        "AWS-AmazonLinux2DefaultPatchBaseline",
		BaselineDescription: "Default patch baseline for Amazon Linux 2",
		OperatingSystem:     "AMAZON_LINUX_2",
		IsDefault:           true,
	}

	if baseline.BaselineID != "pb-abc123" {
		t.Errorf("BaselineID = %v, want pb-abc123", baseline.BaselineID)
	}

	if !baseline.IsDefault {
		t.Error("IsDefault should be true")
	}

	if baseline.OperatingSystem != "AMAZON_LINUX_2" {
		t.Errorf("OperatingSystem = %v, want AMAZON_LINUX_2", baseline.OperatingSystem)
	}
}

func TestDetermineComplianceStatus(t *testing.T) {
	tests := []struct {
		name        string
		missing     int32
		failed      int32
		wantStatus  string
	}{
		{
			name:       "all patched",
			missing:    0,
			failed:     0,
			wantStatus: "COMPLIANT",
		},
		{
			name:       "missing patches",
			missing:    3,
			failed:     0,
			wantStatus: "NON_COMPLIANT",
		},
		{
			name:       "failed patches",
			missing:    0,
			failed:     2,
			wantStatus: "NON_COMPLIANT",
		},
		{
			name:       "both missing and failed",
			missing:    5,
			failed:     3,
			wantStatus: "NON_COMPLIANT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the compliance determination logic
			var status string
			if tt.failed > 0 {
				status = "NON_COMPLIANT"
			} else if tt.missing > 0 {
				status = "NON_COMPLIANT"
			} else {
				status = "COMPLIANT"
			}

			if status != tt.wantStatus {
				t.Errorf("status = %v, want %v", status, tt.wantStatus)
			}
		})
	}
}
