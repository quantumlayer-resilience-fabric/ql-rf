// Package aws provides AWS connector functionality.
package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

func TestNew(t *testing.T) {
	log := logger.New("debug", "text")
	cfg := Config{
		Region:  "us-east-1",
		Regions: []string{"us-east-1", "us-west-2"},
	}

	connector := New(cfg, log)

	if connector.Name() != "aws" {
		t.Errorf("expected name 'aws', got '%s'", connector.Name())
	}

	if connector.Platform() != models.PlatformAWS {
		t.Errorf("expected platform 'aws', got '%s'", connector.Platform())
	}
}

func TestNormalizeInstance(t *testing.T) {
	log := logger.New("debug", "text")
	connector := &Connector{
		log:       log,
		accountID: "123456789012",
	}

	tests := []struct {
		name           string
		instance       ec2Types.Instance
		ami            *ec2Types.Image
		expectedState  models.AssetState
		expectedName   string
		expectedRegion string
	}{
		{
			name: "running instance with name",
			instance: ec2Types.Instance{
				InstanceId: aws.String("i-1234567890abcdef0"),
				ImageId:    aws.String("ami-12345678"),
				State: &ec2Types.InstanceState{
					Name: ec2Types.InstanceStateNameRunning,
				},
				Tags: []ec2Types.Tag{
					{Key: aws.String("Name"), Value: aws.String("web-server-1")},
					{Key: aws.String("Environment"), Value: aws.String("production")},
				},
			},
			ami:            nil,
			expectedState:  models.AssetStateRunning,
			expectedName:   "web-server-1",
			expectedRegion: "us-east-1",
		},
		{
			name: "stopped instance",
			instance: ec2Types.Instance{
				InstanceId: aws.String("i-abcdef1234567890"),
				ImageId:    aws.String("ami-87654321"),
				State: &ec2Types.InstanceState{
					Name: ec2Types.InstanceStateNameStopped,
				},
				Tags: []ec2Types.Tag{},
			},
			ami:            nil,
			expectedState:  models.AssetStateStopped,
			expectedName:   "",
			expectedRegion: "us-west-2",
		},
		{
			name: "instance with AMI version tag",
			instance: ec2Types.Instance{
				InstanceId: aws.String("i-version-test"),
				ImageId:    aws.String("ami-versioned"),
				State: &ec2Types.InstanceState{
					Name: ec2Types.InstanceStateNameRunning,
				},
			},
			ami: &ec2Types.Image{
				ImageId: aws.String("ami-versioned"),
				Name:    aws.String("my-golden-image"),
				Tags: []ec2Types.Tag{
					{Key: aws.String("Version"), Value: aws.String("v2.1.0")},
				},
			},
			expectedState:  models.AssetStateRunning,
			expectedName:   "",
			expectedRegion: "eu-west-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset := connector.normalizeInstance(tt.instance, tt.expectedRegion, tt.ami)

			if asset.State != tt.expectedState {
				t.Errorf("expected state %v, got %v", tt.expectedState, asset.State)
			}

			if asset.Name != tt.expectedName {
				t.Errorf("expected name '%s', got '%s'", tt.expectedName, asset.Name)
			}

			if asset.Region != tt.expectedRegion {
				t.Errorf("expected region '%s', got '%s'", tt.expectedRegion, asset.Region)
			}

			if asset.Account != "123456789012" {
				t.Errorf("expected account '123456789012', got '%s'", asset.Account)
			}

			if asset.Platform != models.PlatformAWS {
				t.Errorf("expected platform AWS, got %v", asset.Platform)
			}

			// Check version extraction when AMI is provided
			if tt.ami != nil && tt.ami.Tags != nil {
				for _, tag := range tt.ami.Tags {
					if *tag.Key == "Version" {
						if asset.ImageVersion != *tag.Value {
							t.Errorf("expected image version '%s', got '%s'", *tag.Value, asset.ImageVersion)
						}
					}
				}
			}
		})
	}
}

func TestConnectorNotConnected(t *testing.T) {
	log := logger.New("debug", "text")
	connector := New(Config{Region: "us-east-1"}, log)

	// Should fail when not connected
	err := connector.Health(nil)
	if err == nil {
		t.Error("expected error when not connected")
	}

	_, err = connector.DiscoverImages(nil)
	if err == nil {
		t.Error("expected error when not connected")
	}
}
