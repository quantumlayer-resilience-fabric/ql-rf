// Package aws provides AWS connector functionality.
package aws

import (
	"context"
	"fmt"

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
	cfg       Config
	awsCfg    aws.Config
	ec2Client *ec2.Client
	log       *logger.Logger
	connected bool
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
	c.connected = true

	c.log.Info("connected to AWS",
		"region", c.cfg.Region,
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
				asset := c.normalizeInstance(instance, region)
				assets = append(assets, asset)
			}
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	c.log.Debug("discovered assets in region",
		"region", region,
		"count", len(assets),
	)

	return assets, nil
}

func (c *Connector) normalizeInstance(instance ec2Types.Instance, region string) models.NormalizedAsset {
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

	// Extract version from AMI name/tags if available
	imageVersion := ""
	// This would typically come from querying the AMI details
	// For now, we leave it empty

	return models.NormalizedAsset{
		Platform:     models.PlatformAWS,
		Account:      "", // Would come from STS GetCallerIdentity
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
