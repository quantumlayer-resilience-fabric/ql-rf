// Package aws provides AWS connector functionality.
package aws

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
	"github.com/google/uuid"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/connector"
)

// Connector implements the AWS platform connector.
type Connector struct {
	cfg        Config
	awsCfg     aws.Config
	ec2Client  *ec2.Client
	stsClient  *sts.Client
	ssmPatcher *SSMPatcher
	log        *logger.Logger
	connected  bool
	accountID  string // AWS Account ID
}

// Config holds AWS-specific configuration.
type Config struct {
	Region        string
	AssumeRoleARN string
	ExternalID    string
	Regions       []string // List of regions to scan
}

// New creates a new AWS connector.
func New(cfg Config, log *logger.Logger) *Connector {
	return &Connector{
		cfg: cfg,
		log: log.WithComponent("aws-connector"),
	}
}

// Name returns the connector name.
func (c *Connector) Name() string {
	return "aws"
}

// Platform returns the platform type.
func (c *Connector) Platform() models.Platform {
	return models.PlatformAWS
}

// Connect establishes a connection to AWS.
func (c *Connector) Connect(ctx context.Context) error {
	// Load default AWS config
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
	c.stsClient = sts.NewFromConfig(awsCfg)
	c.ssmPatcher = NewSSMPatcher(awsCfg, c.log)
	c.connected = true

	// Get caller identity to retrieve account ID
	identity, err := c.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		c.log.Warn("failed to get caller identity", "error", err)
	} else {
		c.accountID = aws.ToString(identity.Account)
	}

	c.log.Info("connected to AWS",
		"region", c.cfg.Region,
		"account_id", c.accountID,
		"assume_role", c.cfg.AssumeRoleARN != "",
	)

	return nil
}

// Close closes the AWS connection.
func (c *Connector) Close() error {
	c.connected = false
	return nil
}

// Health checks the health of the AWS connection.
func (c *Connector) Health(ctx context.Context) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	// Try to describe regions as a health check
	_, err := c.ec2Client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// DiscoverAssets discovers all EC2 instances from AWS.
func (c *Connector) DiscoverAssets(ctx context.Context, orgID uuid.UUID) ([]models.NormalizedAsset, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	regions := c.cfg.Regions
	if len(regions) == 0 {
		// Discover all enabled regions
		regionsOutput, err := c.ec2Client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
		if err != nil {
			return nil, fmt.Errorf("failed to describe regions: %w", err)
		}
		for _, r := range regionsOutput.Regions {
			regions = append(regions, aws.ToString(r.RegionName))
		}
	}

	var allAssets []models.NormalizedAsset

	for _, region := range regions {
		assets, err := c.discoverAssetsInRegion(ctx, region)
		if err != nil {
			c.log.Error("failed to discover assets in region",
				"region", region,
				"error", err,
			)
			continue
		}
		allAssets = append(allAssets, assets...)
	}

	c.log.Info("asset discovery completed",
		"total_assets", len(allAssets),
		"regions_scanned", len(regions),
	)

	return allAssets, nil
}

func (c *Connector) discoverAssetsInRegion(ctx context.Context, region string) ([]models.NormalizedAsset, error) {
	// Create regional client
	regionalCfg := c.awsCfg.Copy()
	regionalCfg.Region = region
	regionalClient := ec2.NewFromConfig(regionalCfg)

	var assets []models.NormalizedAsset
	var nextToken *string

	// Collect all unique AMI IDs for batch lookup
	amiIDSet := make(map[string]bool)
	var allInstances []ec2Types.Instance

	for {
		input := &ec2.DescribeInstancesInput{
			NextToken: nextToken,
		}

		output, err := regionalClient.DescribeInstances(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe instances: %w", err)
		}

		for _, reservation := range output.Reservations {
			for _, instance := range reservation.Instances {
				allInstances = append(allInstances, instance)
				if instance.ImageId != nil {
					amiIDSet[*instance.ImageId] = true
				}
			}
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	// Batch lookup AMI information
	amiInfo := make(map[string]*ec2Types.Image)
	if len(amiIDSet) > 0 {
		amiIDs := make([]string, 0, len(amiIDSet))
		for id := range amiIDSet {
			amiIDs = append(amiIDs, id)
		}

		// DescribeImages can handle up to 1000 image IDs per call
		for i := 0; i < len(amiIDs); i += 500 {
			end := i + 500
			if end > len(amiIDs) {
				end = len(amiIDs)
			}

			imagesOutput, err := regionalClient.DescribeImages(ctx, &ec2.DescribeImagesInput{
				ImageIds: amiIDs[i:end],
			})
			if err != nil {
				c.log.Warn("failed to describe images", "error", err, "region", region)
			} else {
				for j := range imagesOutput.Images {
					img := &imagesOutput.Images[j]
					if img.ImageId != nil {
						amiInfo[*img.ImageId] = img
					}
				}
			}
		}
	}

	// Normalize instances with AMI info
	for _, instance := range allInstances {
		var ami *ec2Types.Image
		if instance.ImageId != nil {
			ami = amiInfo[*instance.ImageId]
		}
		asset := c.normalizeInstance(instance, region, ami)
		assets = append(assets, asset)
	}

	c.log.Debug("discovered assets in region",
		"region", region,
		"count", len(assets),
		"unique_amis", len(amiIDSet),
	)

	return assets, nil
}

func (c *Connector) normalizeInstance(instance ec2Types.Instance, region string, ami *ec2Types.Image) models.NormalizedAsset {
	// Extract tags
	tags := make(map[string]string)
	var name string
	for _, tag := range instance.Tags {
		key := aws.ToString(tag.Key)
		value := aws.ToString(tag.Value)
		tags[key] = value
		if key == "Name" {
			name = value
		}
	}

	// Map EC2 state to our state
	state := models.AssetStateUnknown
	if instance.State != nil {
		switch instance.State.Name {
		case ec2Types.InstanceStateNameRunning:
			state = models.AssetStateRunning
		case ec2Types.InstanceStateNameStopped:
			state = models.AssetStateStopped
		case ec2Types.InstanceStateNameTerminated:
			state = models.AssetStateTerminated
		case ec2Types.InstanceStateNamePending:
			state = models.AssetStatePending
		}
	}

	// Extract image version from AMI metadata
	imageVersion := ""
	imageName := ""
	if ami != nil {
		imageName = aws.ToString(ami.Name)
		// Try to extract version from AMI tags
		for _, tag := range ami.Tags {
			key := aws.ToString(tag.Key)
			value := aws.ToString(tag.Value)
			if key == "Version" || key == "version" {
				imageVersion = value
				break
			}
		}
		// Fall back to creation date as version if no version tag
		if imageVersion == "" && ami.CreationDate != nil {
			imageVersion = aws.ToString(ami.CreationDate)
		}
	}

	// Add image name to tags for reference
	if imageName != "" {
		tags["aws:ami:name"] = imageName
	}

	return models.NormalizedAsset{
		Platform:     models.PlatformAWS,
		Account:      c.accountID,
		Region:       region,
		InstanceID:   aws.ToString(instance.InstanceId),
		Name:         name,
		ImageRef:     aws.ToString(instance.ImageId),
		ImageVersion: imageVersion,
		State:        state,
		Tags:         tags,
	}
}

// DiscoverImages discovers all AMIs owned by the account.
func (c *Connector) DiscoverImages(ctx context.Context) ([]connector.ImageInfo, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	input := &ec2.DescribeImagesInput{
		Owners: []string{"self"},
	}

	output, err := c.ec2Client.DescribeImages(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe images: %w", err)
	}

	var images []connector.ImageInfo
	for _, image := range output.Images {
		tags := make(map[string]string)
		for _, tag := range image.Tags {
			tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}

		images = append(images, connector.ImageInfo{
			Platform:    models.PlatformAWS,
			Identifier:  aws.ToString(image.ImageId),
			Name:        aws.ToString(image.Name),
			Region:      c.cfg.Region,
			CreatedAt:   aws.ToString(image.CreationDate),
			Description: aws.ToString(image.Description),
			Tags:        tags,
		})
	}

	c.log.Info("image discovery completed", "count", len(images))

	return images, nil
}

// SSMPatcher returns the SSM patcher for this connector.
func (c *Connector) SSMPatcher() *SSMPatcher {
	return c.ssmPatcher
}

// SSMPatcherForRegion returns an SSM patcher for a specific region.
func (c *Connector) SSMPatcherForRegion(region string) *SSMPatcher {
	return c.ssmPatcher.ForRegion(region)
}

// ApplyPatches applies patches to an EC2 instance using SSM.
func (c *Connector) ApplyPatches(ctx context.Context, instanceID string, params map[string]interface{}) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	// Extract region from instance ID or use default
	region := c.cfg.Region
	if r, ok := params["region"].(string); ok && r != "" {
		region = r
	}

	patcher := c.ssmPatcher.ForRegion(region)

	// Build patch parameters
	patchParams := ApplyPatchParams{
		Operation:    "Install",
		RebootOption: "NoReboot",
	}

	if op, ok := params["operation"].(string); ok {
		patchParams.Operation = op
	}
	if reboot, ok := params["reboot_if_needed"].(bool); ok && reboot {
		patchParams.RebootOption = "RebootIfNeeded"
	}
	if baseline, ok := params["baseline_override"].(string); ok {
		patchParams.BaselineOverride = baseline
	}

	// Start the patch operation
	op, err := patcher.ApplyPatchBaseline(ctx, instanceID, patchParams)
	if err != nil {
		return fmt.Errorf("failed to start patch operation: %w", err)
	}

	c.log.Info("patch operation started",
		"command_id", op.CommandID,
		"instance_id", instanceID,
	)

	// If synchronous mode requested, wait for completion
	if sync, ok := params["synchronous"].(bool); ok && sync {
		timeout := 30 * time.Minute
		if t, ok := params["timeout"].(time.Duration); ok {
			timeout = t
		}

		op, err = patcher.WaitForCommand(ctx, op.CommandID, instanceID, timeout)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetPatchStatus retrieves the patch compliance status for an instance.
func (c *Connector) GetPatchStatus(ctx context.Context, instanceID string) (string, error) {
	if !c.connected {
		return "", fmt.Errorf("not connected")
	}

	status, err := c.ssmPatcher.GetPatchComplianceStatus(ctx, instanceID)
	if err != nil {
		return "", err
	}

	return status.ComplianceStatus, nil
}

// GetPatchComplianceData retrieves detailed patch compliance data for an instance.
func (c *Connector) GetPatchComplianceData(ctx context.Context, instanceID string) (interface{}, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	return c.ssmPatcher.GetPatchComplianceStatus(ctx, instanceID)
}

// ScanForPatches initiates a patch scan on an instance.
func (c *Connector) ScanForPatches(ctx context.Context, instanceID, region string) (*PatchOperation, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	patcher := c.ssmPatcher
	if region != "" && region != c.cfg.Region {
		patcher = c.ssmPatcher.ForRegion(region)
	}

	return patcher.ScanForPatches(ctx, instanceID)
}

// GetManagedInstances returns all SSM-managed instances.
func (c *Connector) GetManagedInstances(ctx context.Context) ([]ManagedInstance, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	return c.ssmPatcher.GetManagedInstances(ctx)
}
