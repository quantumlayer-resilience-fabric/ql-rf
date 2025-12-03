// Package aws provides AWS connector functionality including SSM patching.
package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmTypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// SSMPatcher handles AWS Systems Manager patching operations.
type SSMPatcher struct {
	ssmClient *ssm.Client
	awsCfg    aws.Config
	log       *logger.Logger
}

// PatchOperation represents a patch operation status.
type PatchOperation struct {
	CommandID     string
	InstanceID    string
	Status        string
	StatusDetails string
	StartTime     time.Time
	EndTime       *time.Time
	Output        string
	ErrorOutput   string
}

// PatchComplianceStatus represents the patch compliance status of an instance.
type PatchComplianceStatus struct {
	InstanceID           string
	PatchGroup           string
	BaselineID           string
	InstalledCount       int32
	InstalledOtherCount  int32
	MissingCount         int32
	FailedCount          int32
	NotApplicableCount   int32
	ComplianceStatus     string
	LastOperationTime    *time.Time
	LastOperationEndTime *time.Time
}

// PatchBaseline represents an SSM patch baseline.
type PatchBaseline struct {
	BaselineID          string
	BaselineName        string
	BaselineDescription string
	OperatingSystem     string
	IsDefault           bool
}

// NewSSMPatcher creates a new SSM patcher from an existing AWS config.
func NewSSMPatcher(awsCfg aws.Config, log *logger.Logger) *SSMPatcher {
	return &SSMPatcher{
		ssmClient: ssm.NewFromConfig(awsCfg),
		awsCfg:    awsCfg,
		log:       log.WithComponent("ssm-patcher"),
	}
}

// NewSSMPatcherForRegion creates a new SSM patcher for a specific region.
func (p *SSMPatcher) ForRegion(region string) *SSMPatcher {
	regionalCfg := p.awsCfg.Copy()
	regionalCfg.Region = region
	return &SSMPatcher{
		ssmClient: ssm.NewFromConfig(regionalCfg),
		awsCfg:    regionalCfg,
		log:       p.log,
	}
}

// ApplyPatchBaseline applies a patch baseline to an EC2 instance using SSM Run Command.
func (p *SSMPatcher) ApplyPatchBaseline(ctx context.Context, instanceID string, params ApplyPatchParams) (*PatchOperation, error) {
	p.log.Info("applying patch baseline",
		"instance_id", instanceID,
		"operation", params.Operation,
		"reboot_option", params.RebootOption,
	)

	// Build parameters for AWS-RunPatchBaseline document
	documentParams := map[string][]string{
		"Operation":    {params.Operation},
		"RebootOption": {params.RebootOption},
	}

	if params.SnapshotID != "" {
		documentParams["SnapshotId"] = []string{params.SnapshotID}
	}

	if params.BaselineOverride != "" {
		documentParams["BaselineOverride"] = []string{params.BaselineOverride}
	}

	input := &ssm.SendCommandInput{
		DocumentName: aws.String("AWS-RunPatchBaseline"),
		InstanceIds:  []string{instanceID},
		Parameters:   documentParams,
		Comment:      aws.String(fmt.Sprintf("QL-RF patch operation: %s", params.Operation)),
		TimeoutSeconds: aws.Int32(3600), // 1 hour timeout
	}

	output, err := p.ssmClient.SendCommand(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to send patch command: %w", err)
	}

	commandID := aws.ToString(output.Command.CommandId)
	p.log.Info("patch command sent",
		"command_id", commandID,
		"instance_id", instanceID,
	)

	return &PatchOperation{
		CommandID:  commandID,
		InstanceID: instanceID,
		Status:     string(output.Command.Status),
		StartTime:  time.Now(),
	}, nil
}

// ApplyPatchParams holds parameters for patch operations.
type ApplyPatchParams struct {
	// Operation: Scan or Install
	Operation string
	// RebootOption: RebootIfNeeded, NoReboot
	RebootOption string
	// Optional: specific snapshot ID
	SnapshotID string
	// Optional: baseline override JSON
	BaselineOverride string
}

// GetCommandStatus retrieves the status of an SSM command.
func (p *SSMPatcher) GetCommandStatus(ctx context.Context, commandID, instanceID string) (*PatchOperation, error) {
	input := &ssm.GetCommandInvocationInput{
		CommandId:  aws.String(commandID),
		InstanceId: aws.String(instanceID),
	}

	output, err := p.ssmClient.GetCommandInvocation(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get command invocation: %w", err)
	}

	op := &PatchOperation{
		CommandID:     commandID,
		InstanceID:    instanceID,
		Status:        string(output.Status),
		StatusDetails: aws.ToString(output.StatusDetails),
		Output:        aws.ToString(output.StandardOutputContent),
		ErrorOutput:   aws.ToString(output.StandardErrorContent),
	}

	if output.ExecutionStartDateTime != nil {
		t, _ := time.Parse(time.RFC3339, *output.ExecutionStartDateTime)
		op.StartTime = t
	}

	if output.ExecutionEndDateTime != nil {
		t, _ := time.Parse(time.RFC3339, *output.ExecutionEndDateTime)
		op.EndTime = &t
	}

	return op, nil
}

// WaitForCommand waits for an SSM command to complete.
func (p *SSMPatcher) WaitForCommand(ctx context.Context, commandID, instanceID string, timeout time.Duration) (*PatchOperation, error) {
	deadline := time.Now().Add(timeout)
	pollInterval := 10 * time.Second

	for time.Now().Before(deadline) {
		op, err := p.GetCommandStatus(ctx, commandID, instanceID)
		if err != nil {
			// Command might not be ready yet
			p.log.Debug("waiting for command status", "command_id", commandID, "error", err)
			time.Sleep(pollInterval)
			continue
		}

		switch op.Status {
		case "Success":
			p.log.Info("command completed successfully", "command_id", commandID)
			return op, nil
		case "Failed", "Cancelled", "TimedOut":
			p.log.Error("command failed",
				"command_id", commandID,
				"status", op.Status,
				"details", op.StatusDetails,
				"error_output", op.ErrorOutput,
			)
			return op, fmt.Errorf("command %s: %s - %s", op.Status, op.StatusDetails, op.ErrorOutput)
		case "Pending", "InProgress", "Delayed":
			p.log.Debug("command in progress", "command_id", commandID, "status", op.Status)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return nil, fmt.Errorf("timeout waiting for command %s to complete", commandID)
}

// GetPatchComplianceStatus retrieves patch compliance information for an instance.
func (p *SSMPatcher) GetPatchComplianceStatus(ctx context.Context, instanceID string) (*PatchComplianceStatus, error) {
	// Get instance patch states
	input := &ssm.DescribeInstancePatchStatesInput{
		InstanceIds: []string{instanceID},
	}

	output, err := p.ssmClient.DescribeInstancePatchStates(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance patch states: %w", err)
	}

	if len(output.InstancePatchStates) == 0 {
		return nil, fmt.Errorf("no patch state found for instance %s", instanceID)
	}

	state := output.InstancePatchStates[0]

	status := &PatchComplianceStatus{
		InstanceID:           instanceID,
		PatchGroup:           aws.ToString(state.PatchGroup),
		BaselineID:           aws.ToString(state.BaselineId),
		InstalledCount:       state.InstalledCount,
		InstalledOtherCount:  state.InstalledOtherCount,
		MissingCount:         state.MissingCount,
		FailedCount:          state.FailedCount,
		NotApplicableCount:   state.NotApplicableCount,
		LastOperationTime:    state.OperationStartTime,
		LastOperationEndTime: state.OperationEndTime,
	}

	// Determine compliance status
	if state.FailedCount > 0 {
		status.ComplianceStatus = "NON_COMPLIANT"
	} else if state.MissingCount > 0 {
		status.ComplianceStatus = "NON_COMPLIANT"
	} else {
		status.ComplianceStatus = "COMPLIANT"
	}

	return status, nil
}

// ListPatchBaselines lists available patch baselines.
func (p *SSMPatcher) ListPatchBaselines(ctx context.Context, operatingSystem string) ([]PatchBaseline, error) {
	var baselines []PatchBaseline
	var nextToken *string

	for {
		input := &ssm.DescribePatchBaselinesInput{
			NextToken: nextToken,
		}

		if operatingSystem != "" {
			input.Filters = []ssmTypes.PatchOrchestratorFilter{
				{
					Key:    aws.String("OPERATING_SYSTEM"),
					Values: []string{operatingSystem},
				},
			}
		}

		output, err := p.ssmClient.DescribePatchBaselines(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe patch baselines: %w", err)
		}

		for _, baseline := range output.BaselineIdentities {
			baselines = append(baselines, PatchBaseline{
				BaselineID:          aws.ToString(baseline.BaselineId),
				BaselineName:        aws.ToString(baseline.BaselineName),
				BaselineDescription: aws.ToString(baseline.BaselineDescription),
				OperatingSystem:     string(baseline.OperatingSystem),
				IsDefault:           baseline.DefaultBaseline,
			})
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return baselines, nil
}

// ScanForPatches performs a patch scan on an instance without installing patches.
func (p *SSMPatcher) ScanForPatches(ctx context.Context, instanceID string) (*PatchOperation, error) {
	return p.ApplyPatchBaseline(ctx, instanceID, ApplyPatchParams{
		Operation:    "Scan",
		RebootOption: "NoReboot",
	})
}

// InstallPatches installs patches on an instance.
func (p *SSMPatcher) InstallPatches(ctx context.Context, instanceID string, rebootIfNeeded bool) (*PatchOperation, error) {
	rebootOption := "NoReboot"
	if rebootIfNeeded {
		rebootOption = "RebootIfNeeded"
	}

	return p.ApplyPatchBaseline(ctx, instanceID, ApplyPatchParams{
		Operation:    "Install",
		RebootOption: rebootOption,
	})
}

// GetManagedInstances lists instances managed by SSM.
func (p *SSMPatcher) GetManagedInstances(ctx context.Context) ([]ManagedInstance, error) {
	var instances []ManagedInstance
	var nextToken *string

	for {
		input := &ssm.DescribeInstanceInformationInput{
			NextToken: nextToken,
		}

		output, err := p.ssmClient.DescribeInstanceInformation(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe instance information: %w", err)
		}

		for _, info := range output.InstanceInformationList {
			instances = append(instances, ManagedInstance{
				InstanceID:       aws.ToString(info.InstanceId),
				PingStatus:       string(info.PingStatus),
				PlatformType:     string(info.PlatformType),
				PlatformName:     aws.ToString(info.PlatformName),
				PlatformVersion:  aws.ToString(info.PlatformVersion),
				AgentVersion:     aws.ToString(info.AgentVersion),
				LastPingDateTime: info.LastPingDateTime,
				IsLatestVersion:  info.IsLatestVersion != nil && *info.IsLatestVersion,
			})
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return instances, nil
}

// ManagedInstance represents an SSM-managed instance.
type ManagedInstance struct {
	InstanceID       string
	PingStatus       string // Online, ConnectionLost, Inactive
	PlatformType     string // Windows, Linux
	PlatformName     string
	PlatformVersion  string
	AgentVersion     string
	LastPingDateTime *time.Time
	IsLatestVersion  bool
}

// RunCommand executes a custom command on an instance.
func (p *SSMPatcher) RunCommand(ctx context.Context, instanceID string, commands []string) (*PatchOperation, error) {
	input := &ssm.SendCommandInput{
		DocumentName: aws.String("AWS-RunShellCommand"),
		InstanceIds:  []string{instanceID},
		Parameters: map[string][]string{
			"commands": commands,
		},
		Comment:        aws.String("QL-RF custom command"),
		TimeoutSeconds: aws.Int32(600), // 10 minute timeout
	}

	output, err := p.ssmClient.SendCommand(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	return &PatchOperation{
		CommandID:  aws.ToString(output.Command.CommandId),
		InstanceID: instanceID,
		Status:     string(output.Command.Status),
		StartTime:  time.Now(),
	}, nil
}

// RunPowerShellCommand executes a PowerShell command on a Windows instance.
func (p *SSMPatcher) RunPowerShellCommand(ctx context.Context, instanceID string, commands []string) (*PatchOperation, error) {
	input := &ssm.SendCommandInput{
		DocumentName: aws.String("AWS-RunPowerShellScript"),
		InstanceIds:  []string{instanceID},
		Parameters: map[string][]string{
			"commands": commands,
		},
		Comment:        aws.String("QL-RF PowerShell command"),
		TimeoutSeconds: aws.Int32(600),
	}

	output, err := p.ssmClient.SendCommand(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to send PowerShell command: %w", err)
	}

	return &PatchOperation{
		CommandID:  aws.ToString(output.Command.CommandId),
		InstanceID: instanceID,
		Status:     string(output.Command.Status),
		StartTime:  time.Now(),
	}, nil
}

// GetPatchSummary retrieves a summary of patches for instances.
func (p *SSMPatcher) GetPatchSummary(ctx context.Context, instanceIDs []string) (map[string]*PatchComplianceStatus, error) {
	results := make(map[string]*PatchComplianceStatus)

	for _, instanceID := range instanceIDs {
		status, err := p.GetPatchComplianceStatus(ctx, instanceID)
		if err != nil {
			p.log.Warn("failed to get patch compliance", "instance_id", instanceID, "error", err)
			continue
		}
		results[instanceID] = status
	}

	return results, nil
}
