// Package aws provides AWS connector functionality.
package aws

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	acmTypes "github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"

	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/connector"
)

// DiscoverCertificates discovers all ACM certificates from AWS.
// This implements the connector.CertificateDiscoverer interface.
func (c *Connector) DiscoverCertificates(ctx context.Context) ([]connector.CertificateInfo, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	regions := c.cfg.Regions
	if len(regions) == 0 {
		// Discover all enabled regions
		regionsOutput, err := c.ec2Client.DescribeRegions(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to describe regions: %w", err)
		}
		for _, r := range regionsOutput.Regions {
			regions = append(regions, aws.ToString(r.RegionName))
		}
	}

	var allCerts []connector.CertificateInfo

	for _, region := range regions {
		certs, err := c.discoverCertificatesInRegion(ctx, region)
		if err != nil {
			c.log.Error("failed to discover certificates in region",
				"region", region,
				"error", err,
			)
			continue
		}
		allCerts = append(allCerts, certs...)
	}

	c.log.Info("certificate discovery completed",
		"total_certificates", len(allCerts),
		"regions_scanned", len(regions),
	)

	return allCerts, nil
}

func (c *Connector) discoverCertificatesInRegion(ctx context.Context, region string) ([]connector.CertificateInfo, error) {
	// Create regional ACM client
	regionalCfg := c.awsCfg.Copy()
	regionalCfg.Region = region
	acmClient := acm.NewFromConfig(regionalCfg)

	var certs []connector.CertificateInfo
	var nextToken *string

	for {
		input := &acm.ListCertificatesInput{
			NextToken: nextToken,
		}

		output, err := acmClient.ListCertificates(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list certificates: %w", err)
		}

		for _, cert := range output.CertificateSummaryList {
			certInfo, err := c.getCertificateDetails(ctx, acmClient, cert, region)
			if err != nil {
				c.log.Warn("failed to get certificate details",
					"arn", aws.ToString(cert.CertificateArn),
					"error", err,
				)
				continue
			}

			// Get certificate usage (load balancers, CloudFront, etc.)
			usages, err := c.getCertificateUsages(ctx, regionalCfg, aws.ToString(cert.CertificateArn))
			if err != nil {
				c.log.Warn("failed to get certificate usages",
					"arn", aws.ToString(cert.CertificateArn),
					"error", err,
				)
			}
			certInfo.Usages = usages

			certs = append(certs, *certInfo)
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	c.log.Debug("discovered certificates in region",
		"region", region,
		"count", len(certs),
	)

	return certs, nil
}

func (c *Connector) getCertificateDetails(ctx context.Context, client *acm.Client, summary acmTypes.CertificateSummary, region string) (*connector.CertificateInfo, error) {
	arn := aws.ToString(summary.CertificateArn)

	// Get detailed certificate info
	descOutput, err := client.DescribeCertificate(ctx, &acm.DescribeCertificateInput{
		CertificateArn: summary.CertificateArn,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe certificate: %w", err)
	}

	cert := descOutput.Certificate

	// Generate a fingerprint from the certificate ARN (ACM doesn't expose actual cert fingerprint via API)
	// In production, you might want to use GetCertificate and parse the PEM to get real fingerprint
	fingerprint := generateFingerprint(arn)

	// Determine if self-signed (ACM certs are usually not self-signed)
	isSelfSigned := cert.Issuer != nil && cert.Subject != nil &&
		aws.ToString(cert.Issuer) == aws.ToString(cert.Subject)

	// Map ACM status to our status
	status := mapACMStatus(cert.Status)

	// Check for auto-renewal
	autoRenew := cert.RenewalEligibility == acmTypes.RenewalEligibilityEligible

	// Extract tags
	tags := make(map[string]string)
	tagsOutput, err := client.ListTagsForCertificate(ctx, &acm.ListTagsForCertificateInput{
		CertificateArn: summary.CertificateArn,
	})
	if err == nil {
		for _, tag := range tagsOutput.Tags {
			tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}

	// Extract SAN entries
	var sans []string
	for _, san := range cert.SubjectAlternativeNames {
		sans = append(sans, san)
	}

	return &connector.CertificateInfo{
		Platform:           models.PlatformAWS,
		Fingerprint:        fingerprint,
		SerialNumber:       aws.ToString(cert.Serial),
		CommonName:         aws.ToString(summary.DomainName),
		SubjectAltNames:    sans,
		Organization:       "", // Not available in ACM API
		IssuerCommonName:   aws.ToString(cert.Issuer),
		IssuerOrganization: "", // Not available in ACM API
		IsSelfSigned:       isSelfSigned,
		IsCA:               false, // ACM certs are typically end-entity certs
		NotBefore:          formatTime(cert.NotBefore),
		NotAfter:           formatTime(cert.NotAfter),
		KeyAlgorithm:       string(cert.KeyAlgorithm),
		KeySize:            getKeySize(cert.KeyAlgorithm),
		SignatureAlgorithm: aws.ToString(cert.SignatureAlgorithm),
		Source:             "acm",
		SourceRef:          arn,
		Region:             region,
		AutoRenew:          autoRenew,
		Status:             status,
		Tags:               tags,
	}, nil
}

func (c *Connector) getCertificateUsages(ctx context.Context, cfg aws.Config, certArn string) ([]connector.CertificateUsageInfo, error) {
	var usages []connector.CertificateUsageInfo

	// Check ELBv2 (ALB/NLB) usage
	elbClient := elasticloadbalancingv2.NewFromConfig(cfg)

	// List all load balancers
	lbOutput, err := elbClient.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{})
	if err != nil {
		c.log.Debug("failed to list load balancers", "error", err)
	} else {
		for _, lb := range lbOutput.LoadBalancers {
			// Get listeners for this load balancer
			listenersOutput, err := elbClient.DescribeListeners(ctx, &elasticloadbalancingv2.DescribeListenersInput{
				LoadBalancerArn: lb.LoadBalancerArn,
			})
			if err != nil {
				continue
			}

			for _, listener := range listenersOutput.Listeners {
				// Check if this listener uses our certificate
				for _, cert := range listener.Certificates {
					if aws.ToString(cert.CertificateArn) == certArn {
						usages = append(usages, connector.CertificateUsageInfo{
							UsageType:   "load_balancer",
							UsageRef:    aws.ToString(lb.LoadBalancerArn),
							ServiceName: aws.ToString(lb.LoadBalancerName),
							Endpoint:    aws.ToString(lb.DNSName),
							Port:        int(aws.ToInt32(listener.Port)),
						})
					}
				}
			}
		}
	}

	// Note: CloudFront and API Gateway certificate usage would require
	// additional API calls to those services. For now, we focus on ELB.
	// This can be extended later.

	return usages, nil
}

// Helper functions

func generateFingerprint(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

func mapACMStatus(status acmTypes.CertificateStatus) string {
	switch status {
	case acmTypes.CertificateStatusIssued:
		return "active"
	case acmTypes.CertificateStatusPendingValidation:
		return "pending_validation"
	case acmTypes.CertificateStatusExpired:
		return "expired"
	case acmTypes.CertificateStatusRevoked:
		return "revoked"
	case acmTypes.CertificateStatusFailed:
		return "failed"
	case acmTypes.CertificateStatusValidationTimedOut:
		return "validation_timed_out"
	case acmTypes.CertificateStatusInactive:
		return "inactive"
	default:
		return "unknown"
	}
}

func getKeySize(alg acmTypes.KeyAlgorithm) int {
	algStr := string(alg)
	if strings.Contains(algStr, "2048") {
		return 2048
	}
	if strings.Contains(algStr, "4096") {
		return 4096
	}
	if strings.Contains(algStr, "256") {
		return 256
	}
	if strings.Contains(algStr, "384") {
		return 384
	}
	if strings.Contains(algStr, "521") {
		return 521
	}
	return 0
}
