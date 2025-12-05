// Package k8s provides Kubernetes connector functionality.
package k8s

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/connector"
)

// DiscoverCertificates discovers all TLS certificates from Kubernetes secrets.
// This implements the connector.CertificateDiscoverer interface.
func (c *Connector) DiscoverCertificates(ctx context.Context) ([]connector.CertificateInfo, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	var allCerts []connector.CertificateInfo

	// Get namespaces to scan
	namespaces, err := c.getNamespacesToScan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespaces: %w", err)
	}

	for _, namespace := range namespaces {
		certs, err := c.discoverCertificatesInNamespace(ctx, namespace)
		if err != nil {
			c.log.Warn("failed to discover certificates in namespace",
				"namespace", namespace,
				"error", err,
			)
			continue
		}
		allCerts = append(allCerts, certs...)
	}

	c.log.Info("certificate discovery completed",
		"total_certificates", len(allCerts),
		"namespaces_scanned", len(namespaces),
	)

	return allCerts, nil
}

func (c *Connector) discoverCertificatesInNamespace(ctx context.Context, namespace string) ([]connector.CertificateInfo, error) {
	var certs []connector.CertificateInfo

	// List TLS secrets
	secrets, err := c.client.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "type=kubernetes.io/tls",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list TLS secrets: %w", err)
	}

	for _, secret := range secrets.Items {
		certInfo, err := c.parseSecretCertificate(secret, namespace)
		if err != nil {
			c.log.Debug("failed to parse certificate from secret",
				"secret", secret.Name,
				"namespace", namespace,
				"error", err,
			)
			continue
		}

		// Get certificate usages (ingresses using this secret)
		usages, err := c.getCertificateUsages(ctx, namespace, secret.Name)
		if err != nil {
			c.log.Debug("failed to get certificate usages",
				"secret", secret.Name,
				"error", err,
			)
		}
		certInfo.Usages = usages

		certs = append(certs, *certInfo)
	}

	// Also check for certificates in generic secrets with cert data
	genericSecrets, err := c.client.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "type=Opaque",
	})
	if err == nil {
		for _, secret := range genericSecrets.Items {
			// Look for common certificate-related keys
			if _, hasCert := secret.Data["tls.crt"]; hasCert {
				certInfo, err := c.parseSecretCertificate(secret, namespace)
				if err != nil {
					continue
				}
				certs = append(certs, *certInfo)
			}
		}
	}

	c.log.Debug("discovered certificates in namespace",
		"namespace", namespace,
		"count", len(certs),
	)

	return certs, nil
}

func (c *Connector) parseSecretCertificate(secret corev1.Secret, namespace string) (*connector.CertificateInfo, error) {
	// Get the certificate data
	certData, ok := secret.Data["tls.crt"]
	if !ok {
		// Try alternate key names
		certData, ok = secret.Data["cert.pem"]
		if !ok {
			certData, ok = secret.Data["certificate.pem"]
			if !ok {
				return nil, fmt.Errorf("no certificate data found in secret")
			}
		}
	}

	// Parse the PEM encoded certificate
	block, _ := pem.Decode(certData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Generate fingerprint
	fingerprint := sha256Fingerprint(cert.Raw)

	// Extract SANs
	var sans []string
	sans = append(sans, cert.DNSNames...)
	for _, ip := range cert.IPAddresses {
		sans = append(sans, ip.String())
	}

	// Determine if self-signed
	isSelfSigned := cert.Issuer.String() == cert.Subject.String()

	// Build tags from secret labels
	tags := make(map[string]string)
	for k, v := range secret.Labels {
		tags[k] = v
	}
	tags["namespace"] = namespace
	tags["secret_name"] = secret.Name

	// Determine status based on validity
	status := "active"
	// Note: We can't easily check expiry here without current time comparison
	// The caller should use the NotAfter field to determine actual status

	return &connector.CertificateInfo{
		Platform:           models.PlatformK8s,
		Fingerprint:        fingerprint,
		SerialNumber:       cert.SerialNumber.String(),
		CommonName:         cert.Subject.CommonName,
		SubjectAltNames:    sans,
		Organization:       strings.Join(cert.Subject.Organization, ", "),
		IssuerCommonName:   cert.Issuer.CommonName,
		IssuerOrganization: strings.Join(cert.Issuer.Organization, ", "),
		IsSelfSigned:       isSelfSigned,
		IsCA:               cert.IsCA,
		NotBefore:          cert.NotBefore.Format("2006-01-02T15:04:05Z"),
		NotAfter:           cert.NotAfter.Format("2006-01-02T15:04:05Z"),
		KeyAlgorithm:       keyAlgorithmName(cert.PublicKeyAlgorithm),
		KeySize:            getPublicKeySize(cert),
		SignatureAlgorithm: cert.SignatureAlgorithm.String(),
		Source:             "k8s_secret",
		SourceRef:          fmt.Sprintf("%s/%s", namespace, secret.Name),
		Region:             namespace,
		AutoRenew:          false, // K8s secrets don't auto-renew unless managed by cert-manager
		Status:             status,
		Tags:               tags,
	}, nil
}

func (c *Connector) getCertificateUsages(ctx context.Context, namespace, secretName string) ([]connector.CertificateUsageInfo, error) {
	var usages []connector.CertificateUsageInfo

	// Check Ingresses using this secret
	ingresses, err := c.client.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, ingress := range ingresses.Items {
			for _, tls := range ingress.Spec.TLS {
				if tls.SecretName == secretName {
					// This ingress uses our certificate
					for _, host := range tls.Hosts {
						usages = append(usages, connector.CertificateUsageInfo{
							UsageType:   "ingress",
							UsageRef:    fmt.Sprintf("%s/%s", namespace, ingress.Name),
							ServiceName: ingress.Name,
							Endpoint:    host,
							Port:        443,
						})
					}
				}
			}
		}
	}

	// Check Services that might reference this secret (for TLS termination)
	// This is less common but can happen with certain service mesh configurations

	return usages, nil
}

// Helper functions

func sha256Fingerprint(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func keyAlgorithmName(algo x509.PublicKeyAlgorithm) string {
	switch algo {
	case x509.RSA:
		return "RSA"
	case x509.ECDSA:
		return "ECDSA"
	case x509.Ed25519:
		return "Ed25519"
	case x509.DSA:
		return "DSA"
	default:
		return "Unknown"
	}
}

func getPublicKeySize(cert *x509.Certificate) int {
	switch pub := cert.PublicKey.(type) {
	case interface{ Size() int }:
		return pub.Size() * 8 // Convert bytes to bits
	default:
		return 0
	}
}
