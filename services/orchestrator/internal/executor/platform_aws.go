// Package executor implements the plan execution engine.
package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// AWSPlatformClient implements PlatformClient for AWS.
type AWSPlatformClient struct {
	cfg       AWSClientConfig
	awsCfg    aws.Config
	ec2Client *ec2.Client
	ssmClient *ssm.Client
	log       *logger.Logger
	connected bool
}

// AWSClientConfig holds AWS client configuration.
type AWSClientConfig struct {
	Region        string
	AssumeRoleARN string
	ExternalID    string
}

// NewAWSPlatformClient creates a new AWS platform client.
func NewAWSPlatformClient(cfg AWSClientConfig, log *logger.Logger) *AWSPlatformClient {
	return &AWSPlatformClient{
		cfg: cfg,
		log: log.WithComponent("aws-platform-client"),
	}
}

// Connect establishes a connection to AWS.
func (c *AWSPlatformClient) Connect(ctx context.Context) error {
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(c.cfg.Region),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// If assume role is configured, create STS credentials provider
	if c.cfg.AssumeRoleARN != "" {
		stsClient := sts.NewFromConfig(awsCfg)
		creds := stscreds.NewAssumeRoleProvider(stsClient, c.cfg.AssumeRoleARN,
			func(o *stscreds.AssumeRoleOptions) {
				if c.cfg.ExternalID != "" {
					o.ExternalID = aws.String(c.cfg.ExternalID)
				}
			},
		)
		awsCfg.Credentials = aws.NewCredentialsCache(creds)
	}

	c.awsCfg = awsCfg
	c.ec2Client = ec2.NewFromConfig(awsCfg)
	c.ssmClient = ssm.NewFromConfig(awsCfg)
	c.connected = true

	c.log.Info("connected to AWS",
		"region", c.cfg.Region,
		"assume_role", c.cfg.AssumeRoleARN != "",
	)

	return nil
}

// Close closes the AWS connection.
func (c *AWSPlatformClient) Close() error {
	c.connected = false
	return nil
}

// ReimageInstance reimages an EC2 instance with a new AMI.
// This stops the instance, creates a new instance from the AMI, and terminates the old one.
// For immutable infrastructure, this is the standard approach.
func (c *AWSPlatformClient) ReimageInstance(ctx context.Context, instanceID, imageID string) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	c.log.Info("reimaging instance",
		"instance_id", instanceID,
		"target_ami", imageID,
	)

	// Get the current instance details
	describeOutput, err := c.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to describe instance: %w", err)
	}

	if len(describeOutput.Reservations) == 0 || len(describeOutput.Reservations[0].Instances) == 0 {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	instance := describeOutput.Reservations[0].Instances[0]

	// For production immutable infrastructure:
	// 1. Stop accepting new connections (via load balancer)
	// 2. Create new instance from target AMI
	// 3. Wait for new instance to be healthy
	// 4. Switch traffic to new instance
	// 5. Terminate old instance

	// For simpler cases, we can use the instance replacement approach:
	// Stop and terminate the old instance, launch new from AMI with same config

	// Stop the instance first
	_, err = c.ec2Client.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	// Wait for instance to stop
	if err := c.WaitForInstanceState(ctx, instanceID, "stopped", 5*time.Minute); err != nil {
		return fmt.Errorf("instance did not stop: %w", err)
	}

	// Get the instance configuration for launching replacement
	launchInput := &ec2.RunInstancesInput{
		ImageId:      aws.String(imageID),
		InstanceType: instance.InstanceType,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	}

	// Copy key pair
	if instance.KeyName != nil {
		launchInput.KeyName = instance.KeyName
	}

	// Copy security groups
	var securityGroupIds []string
	for _, sg := range instance.SecurityGroups {
		if sg.GroupId != nil {
			securityGroupIds = append(securityGroupIds, *sg.GroupId)
		}
	}
	if len(securityGroupIds) > 0 {
		launchInput.SecurityGroupIds = securityGroupIds
	}

	// Copy subnet
	if instance.SubnetId != nil {
		launchInput.SubnetId = instance.SubnetId
	}

	// Copy IAM instance profile
	if instance.IamInstanceProfile != nil && instance.IamInstanceProfile.Arn != nil {
		launchInput.IamInstanceProfile = &ec2Types.IamInstanceProfileSpecification{
			Arn: instance.IamInstanceProfile.Arn,
		}
	}

	// Copy tags (including Name)
	if len(instance.Tags) > 0 {
		var tags []ec2Types.Tag
		for _, tag := range instance.Tags {
			tags = append(tags, ec2Types.Tag{
				Key:   tag.Key,
				Value: tag.Value,
			})
		}
		launchInput.TagSpecifications = []ec2Types.TagSpecification{
			{
				ResourceType: ec2Types.ResourceTypeInstance,
				Tags:         tags,
			},
		}
	}

	// Launch new instance
	runOutput, err := c.ec2Client.RunInstances(ctx, launchInput)
	if err != nil {
		// Restart old instance if launch fails
		c.ec2Client.StartInstances(ctx, &ec2.StartInstancesInput{
			InstanceIds: []string{instanceID},
		})
		return fmt.Errorf("failed to launch replacement instance: %w", err)
	}

	newInstanceID := aws.ToString(runOutput.Instances[0].InstanceId)
	c.log.Info("launched replacement instance",
		"old_instance_id", instanceID,
		"new_instance_id", newInstanceID,
		"ami", imageID,
	)

	// Wait for new instance to be running
	if err := c.WaitForInstanceState(ctx, newInstanceID, "running", 5*time.Minute); err != nil {
		c.log.Error("new instance failed to start, keeping old instance",
			"new_instance_id", newInstanceID,
			"error", err,
		)
		// Terminate failed new instance
		c.ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
			InstanceIds: []string{newInstanceID},
		})
		// Restart old instance
		c.ec2Client.StartInstances(ctx, &ec2.StartInstancesInput{
			InstanceIds: []string{instanceID},
		})
		return fmt.Errorf("replacement instance failed to start: %w", err)
	}

	// Terminate old instance
	_, err = c.ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		c.log.Warn("failed to terminate old instance",
			"instance_id", instanceID,
			"error", err,
		)
	}

	c.log.Info("reimage completed",
		"old_instance_id", instanceID,
		"new_instance_id", newInstanceID,
	)

	return nil
}

// RebootInstance reboots an EC2 instance.
func (c *AWSPlatformClient) RebootInstance(ctx context.Context, instanceID string) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	c.log.Info("rebooting instance", "instance_id", instanceID)

	_, err := c.ec2Client.RebootInstances(ctx, &ec2.RebootInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to reboot instance: %w", err)
	}

	// Wait a bit for the reboot to initiate
	time.Sleep(10 * time.Second)

	return nil
}

// TerminateInstance terminates an EC2 instance.
func (c *AWSPlatformClient) TerminateInstance(ctx context.Context, instanceID string) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	c.log.Info("terminating instance", "instance_id", instanceID)

	_, err := c.ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}

	return nil
}

// GetInstanceStatus gets the current status of an EC2 instance.
func (c *AWSPlatformClient) GetInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	if !c.connected {
		return "", fmt.Errorf("not connected")
	}

	output, err := c.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe instance: %w", err)
	}

	if len(output.Reservations) == 0 || len(output.Reservations[0].Instances) == 0 {
		return "", fmt.Errorf("instance not found: %s", instanceID)
	}

	state := output.Reservations[0].Instances[0].State
	if state == nil || state.Name == "" {
		return "unknown", nil
	}

	return string(state.Name), nil
}

// WaitForInstanceState waits for an instance to reach a specific state.
func (c *AWSPlatformClient) WaitForInstanceState(ctx context.Context, instanceID, targetState string, timeout time.Duration) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	c.log.Debug("waiting for instance state",
		"instance_id", instanceID,
		"target_state", targetState,
		"timeout", timeout,
	)

	deadline := time.Now().Add(timeout)
	pollInterval := 10 * time.Second

	for time.Now().Before(deadline) {
		status, err := c.GetInstanceStatus(ctx, instanceID)
		if err != nil {
			// For terminated state, the instance might not be found
			if targetState == "terminated" {
				return nil
			}
			return err
		}

		if status == targetState {
			c.log.Debug("instance reached target state",
				"instance_id", instanceID,
				"state", targetState,
			)
			return nil
		}

		c.log.Debug("waiting for instance state",
			"instance_id", instanceID,
			"current_state", status,
			"target_state", targetState,
		)

		select {
		case <-time.After(pollInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("timeout waiting for instance %s to reach state %s", instanceID, targetState)
}

// GetRegionalClient creates a client for a specific region.
func (c *AWSPlatformClient) GetRegionalClient(region string) *ec2.Client {
	regionalCfg := c.awsCfg.Copy()
	regionalCfg.Region = region
	return ec2.NewFromConfig(regionalCfg)
}

// GetRegionalSSMClient creates an SSM client for a specific region.
func (c *AWSPlatformClient) GetRegionalSSMClient(region string) *ssm.Client {
	regionalCfg := c.awsCfg.Copy()
	regionalCfg.Region = region
	return ssm.NewFromConfig(regionalCfg)
}

// ApplyPatches applies patches to an EC2 instance using AWS SSM.
func (c *AWSPlatformClient) ApplyPatches(ctx context.Context, instanceID string, params map[string]interface{}) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	// Get regional SSM client if region is specified
	ssmClient := c.ssmClient
	if region, ok := params["region"].(string); ok && region != "" && region != c.cfg.Region {
		ssmClient = c.GetRegionalSSMClient(region)
	}

	// Build patch operation parameters
	operation := "Install"
	if op, ok := params["operation"].(string); ok && op != "" {
		operation = op
	}

	rebootOption := "NoReboot"
	if reboot, ok := params["reboot_if_needed"].(bool); ok && reboot {
		rebootOption = "RebootIfNeeded"
	}

	documentParams := map[string][]string{
		"Operation":    {operation},
		"RebootOption": {rebootOption},
	}

	if baseline, ok := params["baseline_override"].(string); ok && baseline != "" {
		documentParams["BaselineOverride"] = []string{baseline}
	}

	c.log.Info("sending SSM patch command",
		"instance_id", instanceID,
		"operation", operation,
		"reboot_option", rebootOption,
	)

	// Send the patch command
	sendInput := &ssm.SendCommandInput{
		DocumentName:   aws.String("AWS-RunPatchBaseline"),
		InstanceIds:    []string{instanceID},
		Parameters:     documentParams,
		Comment:        aws.String(fmt.Sprintf("QL-RF patch operation: %s", operation)),
		TimeoutSeconds: aws.Int32(3600), // 1 hour timeout
	}

	output, err := ssmClient.SendCommand(ctx, sendInput)
	if err != nil {
		return fmt.Errorf("failed to send SSM patch command: %w", err)
	}

	commandID := aws.ToString(output.Command.CommandId)
	c.log.Info("SSM patch command sent",
		"command_id", commandID,
		"instance_id", instanceID,
	)

	// If synchronous mode, wait for completion
	if sync, ok := params["synchronous"].(bool); ok && sync {
		timeout := 30 * time.Minute
		if t, ok := params["timeout"].(time.Duration); ok {
			timeout = t
		}

		if err := c.waitForSSMCommand(ctx, ssmClient, commandID, instanceID, timeout); err != nil {
			return err
		}
	}

	return nil
}

// waitForSSMCommand waits for an SSM command to complete.
func (c *AWSPlatformClient) waitForSSMCommand(ctx context.Context, ssmClient *ssm.Client, commandID, instanceID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	pollInterval := 10 * time.Second

	for time.Now().Before(deadline) {
		input := &ssm.GetCommandInvocationInput{
			CommandId:  aws.String(commandID),
			InstanceId: aws.String(instanceID),
		}

		output, err := ssmClient.GetCommandInvocation(ctx, input)
		if err != nil {
			// Command might not be ready yet
			c.log.Debug("waiting for SSM command status", "command_id", commandID, "error", err)
			time.Sleep(pollInterval)
			continue
		}

		status := string(output.Status)
		switch status {
		case "Success":
			c.log.Info("SSM command completed successfully",
				"command_id", commandID,
				"instance_id", instanceID,
			)
			return nil
		case "Failed", "Cancelled", "TimedOut":
			errorOutput := aws.ToString(output.StandardErrorContent)
			c.log.Error("SSM command failed",
				"command_id", commandID,
				"status", status,
				"error_output", errorOutput,
			)
			return fmt.Errorf("SSM command %s: %s", status, errorOutput)
		case "Pending", "InProgress", "Delayed":
			c.log.Debug("SSM command in progress", "command_id", commandID, "status", status)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return fmt.Errorf("timeout waiting for SSM command %s to complete", commandID)
}

// GetPatchStatus retrieves patch compliance status for an EC2 instance.
func (c *AWSPlatformClient) GetPatchStatus(ctx context.Context, instanceID string) (string, error) {
	if !c.connected {
		return "", fmt.Errorf("not connected")
	}

	input := &ssm.DescribeInstancePatchStatesInput{
		InstanceIds: []string{instanceID},
	}

	output, err := c.ssmClient.DescribeInstancePatchStates(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to describe instance patch states: %w", err)
	}

	if len(output.InstancePatchStates) == 0 {
		return "unknown", nil
	}

	state := output.InstancePatchStates[0]

	// Determine compliance status
	if state.FailedCount > 0 || state.MissingCount > 0 {
		return "NON_COMPLIANT", nil
	}

	return "COMPLIANT", nil
}

// GetPatchComplianceData retrieves detailed patch compliance data for an instance.
func (c *AWSPlatformClient) GetPatchComplianceData(ctx context.Context, instanceID string) (interface{}, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	input := &ssm.DescribeInstancePatchStatesInput{
		InstanceIds: []string{instanceID},
	}

	output, err := c.ssmClient.DescribeInstancePatchStates(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance patch states: %w", err)
	}

	if len(output.InstancePatchStates) == 0 {
		return nil, fmt.Errorf("no patch state found for instance %s", instanceID)
	}

	state := output.InstancePatchStates[0]

	// Build compliance data map
	complianceData := map[string]interface{}{
		"instance_id":           instanceID,
		"patch_group":           aws.ToString(state.PatchGroup),
		"baseline_id":           aws.ToString(state.BaselineId),
		"installed_count":       state.InstalledCount,
		"installed_other_count": state.InstalledOtherCount,
		"missing_count":         state.MissingCount,
		"failed_count":          state.FailedCount,
		"not_applicable_count":  state.NotApplicableCount,
		"operation":             string(state.Operation),
	}

	if state.OperationStartTime != nil {
		complianceData["operation_start_time"] = state.OperationStartTime.Format(time.RFC3339)
	}
	if state.OperationEndTime != nil {
		complianceData["operation_end_time"] = state.OperationEndTime.Format(time.RFC3339)
	}

	// Determine compliance status
	if state.FailedCount > 0 || state.MissingCount > 0 {
		complianceData["compliance_status"] = "NON_COMPLIANT"
	} else {
		complianceData["compliance_status"] = "COMPLIANT"
	}

	return complianceData, nil
}
