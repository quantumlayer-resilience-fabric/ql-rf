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
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// AWSPlatformClient implements PlatformClient for AWS.
type AWSPlatformClient struct {
	cfg       AWSClientConfig
	awsCfg    aws.Config
	ec2Client *ec2.Client
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
